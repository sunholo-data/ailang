package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/verification"
)

// verifyCommand implements the `ailang verify` CLI command: parse flags,
// compile the file, hand the artifacts to smt.Verify, print the report.
// The verification logic itself lives in internal/smt (M-V1-SIMPLIFY-S2 M3).
func verifyCommand() {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose", false, "Show generated SMT-LIB for each function")
	jsonFlag := fs.Bool("json", false, "Output results in JSON format")
	strictFlag := fs.Bool("strict", false, strictFlagHelp)
	allowInconclusiveFlag := fs.Bool("allow-inconclusive", false, allowInconclusiveFlagHelp)
	timeoutFlag := fs.Duration("timeout", 5*time.Second, "Per-function Z3 timeout (hard backstop adds 2s grace)")
	recursiveDepthFlag := fs.Int("verify-recursive-depth", 2, "Bounded recursion unrolling depth (1-10, 0 to disable)")
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")
	packageFlag := fs.String("package", "", "Verify every exported module of the package at this directory (M-PKG-QUALITY-LADDER)")
	wallCapFlag := fs.Duration("wall-cap", 0, "With --package: whole-package time cap (0 = unbounded)")

	parseInterspersed(fs, os.Args[2:])
	policy := verification.ExitPolicy{Strict: *strictFlag, AllowInconclusive: *allowInconclusiveFlag}
	refuseContradictoryPolicy(policy)

	if *packageFlag != "" {
		verifyPackageCommand(*packageFlag, verifyPackageOptions{
			Timeout:        *timeoutFlag,
			RecursiveDepth: *recursiveDepthFlag,
			Verbose:        *verboseFlag,
			WallCap:        *wallCapFlag,
		}, *jsonFlag, policy)
		return
	}

	filename := verificationFileArg(fs, "verify")
	if filename == "" {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang verify [options] <file.ail>")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --verbose    Show generated SMT-LIB for each function")
		fmt.Println("  --json            Output results in JSON format")
		fmt.Println("  --strict          " + strictFlagHelp)
		fmt.Println("  --allow-inconclusive  " + allowInconclusiveFlagHelp)
		fmt.Println("  --timeout         Per-function Z3 timeout; hard backstop adds 2s grace (default: 5s)")
		fmt.Println("  --relax-modules   Relax MOD010 validation (allow module path mismatches)")
		fmt.Println("  --package <dir>   Verify every exported module of a package (flat or canonical layout)")
		fmt.Println("  --wall-cap        With --package: whole-package time cap (default: unbounded)")
		fmt.Println()
		fmt.Println("Verifies requires/ensures contracts using Z3 SMT solver.")
		fmt.Println("Exit 0: every contract proved, or some proved + some skipped (skips listed on stderr), or no ensures to prove.")
		fmt.Println("Exit 1: a counterexample, a solver error/timeout, an inconclusive result, every contract skipped (none proved), or Z3 missing while")
		fmt.Println("        the file has contracts. The JSON 'outcome' field names which.")
		fmt.Println()
		fmt.Println("IFC labels (v0.16.0+):")
		fmt.Println("  Functions can use T<label> for tainted sources and T{not LABEL}")
		fmt.Println("  for sinks that refuse a given label. Declassify in the effect row")
		fmt.Println("  marks intentional label changes. See:")
		fmt.Println("    https://ailang.sunholo.com/docs/guides/ifc-labels")
		fmt.Println("    examples/runnable/contracts/inbox_injection_v2.ail")
		os.Exit(1)
	}

	// Read and compile the file. The solver is looked for AFTER compiling:
	// whether a missing Z3 fails the run depends on whether the file has any
	// contract to prove (verification.ExitCode, `unavailable` lane).
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Suppress warnings in JSON mode so they don't pollute output
	if *jsonFlag {
		os.Setenv("AILANG_QUIET_WARNINGS", "1")
	}

	// Check AILANG_RELAX_MODULES environment variable
	relaxModulesEffective := *relaxModulesFlag || config.RelaxModules()

	cfg := pipeline.Config{
		DryLink:      true, // Don't evaluate, just compile
		RelaxModules: relaxModulesEffective,
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
	var compileErrors []string
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: compilation failed: %v\n", red("Error"), err)
		compileErrors = append(compileErrors, err.Error())
	}
	for _, e := range result.Errors {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
		compileErrors = append(compileErrors, fmt.Sprint(e))
	}
	if len(compileErrors) > 0 {
		// --json always prints a report, so a consumer reads `not_run` rather
		// than an empty stdout.
		if *jsonFlag {
			printVerifyJSONNotRun(filename, compileErrors)
		}
		os.Exit(1)
	}

	coreProg := result.Artifacts.Core
	surfaceAST := result.Artifacts.AST
	imported := importedContractModules(result)

	// Solver missing: count what would have been proved. No obligations is a
	// pass (nothing needed Z3); any obligation is the `unavailable` lane.
	if !smt.Z3Available() {
		t := verification.Tally{CheckPassed: true, Obligations: smt.CountObligations(coreProg)}
		if *jsonFlag {
			printVerifyJSON(&smt.VerifyReport{Evidence: verification.EncodedEvidence()}, filename, t, imported)
		} else if t.Obligations == 0 {
			fmt.Fprintf(os.Stderr, "%s: no functions with contracts found (Z3 not found; nothing needed it)\n", yellow("Warning"))
		}
		writeVerificationNotice(os.Stderr, t, policy, nil)
		writeImportedNote(os.Stderr, imported)
		os.Exit(verification.ExitCode(t, policy))
	}

	if coreProg == nil || coreProg.Meta == nil {
		fmt.Fprintf(os.Stderr, "%s: no functions with contracts found\n", yellow("Warning"))
		writeImportedNote(os.Stderr, imported)
		os.Exit(0)
	}

	report := smt.Verify(coreProg, surfaceAST, verifyModulesFromPipeline(result), smt.VerifyOptions{
		CoreTI:         result.Artifacts.CoreTI,
		Timeout:        *timeoutFlag,
		RecursiveDepth: *recursiveDepthFlag,
		Verbose:        *verboseFlag,
	})
	t := report.Tally()

	// Output results
	if *jsonFlag {
		printVerifyJSON(report, filename, t, imported)
	} else {
		printVerifyHuman(report, filename, *verboseFlag)
	}

	// The shared exit contract (verification.ExitCode): the same function
	// `check --verify` and `ai-check` exit through. An uncontracted function is
	// never counted — it is a function nobody wrote a contract for, not one
	// that failed to verify.
	writeVerificationNotice(os.Stderr, t, policy, report.Results)
	writeImportedNote(os.Stderr, imported)
	if code := verification.ExitCode(t, policy); code != 0 {
		os.Exit(code)
	}
}

// verifyModulesFromPipeline projects the pipeline's loaded modules onto the
// syntax, Core and checked types the verifier consumes, so internal/smt does not depend on
// internal/loader.
func verifyModulesFromPipeline(result pipeline.Result) map[string]smt.VerifyModule {
	if result.Modules == nil {
		return nil
	}
	mods := make(map[string]smt.VerifyModule, len(result.Modules))
	for path, mod := range result.Modules {
		if mod == nil {
			continue
		}
		coreTI, _ := mod.CoreTI.(types.CoreTypeInfo)
		mods[path] = smt.VerifyModule{File: mod.File, Core: mod.Core, CoreTI: coreTI}
	}
	return mods
}

// verifyPackageCommand is the --package arm: Z3 check, verify, print, exit.
// A package run is an explicit request to verify a whole package, so a missing
// solver is refused up front (the `unavailable` lane) without compiling every
// module to count obligations.
func verifyPackageCommand(dir string, opts verifyPackageOptions, jsonOut bool, policy verification.ExitPolicy) {
	if !smt.Z3Available() {
		writeZ3NotFound(os.Stderr, "the package was not verified (verify.outcome: unavailable)")
		os.Exit(1)
	}
	if jsonOut {
		os.Setenv("AILANG_QUIET_WARNINGS", "1")
	}
	report, err := verifyPackage(dir, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if jsonOut {
		printPackageVerifyJSON(report)
	} else {
		printPackageVerifyHuman(report)
	}
	// Module-level outcomes (a compile error, a wall-cap skip) have no rows, so
	// the package counters are the base; requires-only rows are taken back out
	// of `skipped` because they are not obligations (smt.IsObligation).
	nonObligations := 0
	for _, m := range report.Modules {
		for _, r := range m.Results {
			if !smt.IsObligation(smt.VerifyResult{Status: r.Status, ReasonCode: r.ReasonCode}) && r.Status == "skipped" {
				nonObligations++
			}
		}
	}
	t := verification.Tally{
		CheckPassed:     true,
		SolverAvailable: true,
		Verified:        report.Verified,
		Counterexample:  report.Counterexample,
		Inconclusive:    report.Inconclusive,
		Skipped:         report.Skipped - nonObligations,
		Errors:          report.Errors,
	}
	t.Obligations = t.Verified + t.Counterexample + t.Inconclusive + t.Skipped + t.Errors
	writeVerificationNotice(os.Stderr, t, policy, nil)
	if code := verification.ExitCode(t, policy); code != 0 {
		os.Exit(code)
	}
}
