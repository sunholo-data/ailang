package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/smt"
)

// verifyCommand implements the `ailang verify` CLI command: parse flags,
// compile the file, hand the artifacts to smt.Verify, print the report.
// The verification logic itself lives in internal/smt (M-V1-SIMPLIFY-S2 M3).
func verifyCommand() {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose", false, "Show generated SMT-LIB for each function")
	jsonFlag := fs.Bool("json", false, "Output results in JSON format")
	strictFlag := fs.Bool("strict", false, "Exit with error if any function cannot be verified")
	timeoutFlag := fs.Duration("timeout", 5*time.Second, "Per-function Z3 timeout (hard backstop adds 2s grace)")
	recursiveDepthFlag := fs.Int("verify-recursive-depth", 2, "Bounded recursion unrolling depth (1-10, 0 to disable)")
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")
	packageFlag := fs.String("package", "", "Verify every exported module of the package at this directory (M-PKG-QUALITY-LADDER)")
	wallCapFlag := fs.Duration("wall-cap", 0, "With --package: whole-package time cap (0 = unbounded)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *packageFlag != "" {
		verifyPackageCommand(*packageFlag, verifyPackageOptions{
			Timeout:        *timeoutFlag,
			RecursiveDepth: *recursiveDepthFlag,
			Verbose:        *verboseFlag,
			WallCap:        *wallCapFlag,
		}, *jsonFlag, *strictFlag)
		return
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang verify [options] <file.ail>")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --verbose    Show generated SMT-LIB for each function")
		fmt.Println("  --json            Output results in JSON format")
		fmt.Println("  --strict          Exit with error if any function cannot be verified")
		fmt.Println("  --timeout         Per-function Z3 timeout; hard backstop adds 2s grace (default: 5s)")
		fmt.Println("  --relax-modules   Relax MOD010 validation (allow module path mismatches)")
		fmt.Println("  --package <dir>   Verify every exported module of a package (flat or canonical layout)")
		fmt.Println("  --wall-cap        With --package: whole-package time cap (default: unbounded)")
		fmt.Println()
		fmt.Println("Verifies requires/ensures contracts using Z3 SMT solver.")
		fmt.Println("Returns exit code 0 if all verifiable contracts are proven.")
		fmt.Println()
		fmt.Println("IFC labels (v0.16.0+):")
		fmt.Println("  Functions can use T<label> for tainted sources and T{not LABEL}")
		fmt.Println("  for sinks that refuse a given label. Declassify in the effect row")
		fmt.Println("  marks intentional label changes. See:")
		fmt.Println("    https://ailang.sunholo.com/docs/guides/ifc-labels")
		fmt.Println("    examples/runnable/contracts/inbox_injection_v2.ail")
		os.Exit(1)
	}

	filename := fs.Arg(0)

	// Check Z3 availability upfront
	if !smt.Z3Available() {
		fmt.Fprintf(os.Stderr, "%s Z3 solver not found\n", red("Error:"))
		fmt.Fprintf(os.Stderr, "Install with: brew install z3 (macOS), apt install z3 (Linux), or choco install z3 / scoop install z3 (Windows)\n")
		fmt.Fprintf(os.Stderr, "Or download from https://github.com/Z3Prover/z3/releases and set AILANG_Z3_PATH.\n")
		os.Exit(1)
	}

	// Read and compile the file
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
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: compilation failed: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
		}
		os.Exit(1)
	}

	coreProg := result.Artifacts.Core
	surfaceAST := result.Artifacts.AST

	if coreProg == nil || coreProg.Meta == nil {
		fmt.Fprintf(os.Stderr, "%s: no functions with contracts found\n", yellow("Warning"))
		os.Exit(0)
	}

	report := smt.Verify(coreProg, surfaceAST, verifyModulesFromPipeline(result), smt.VerifyOptions{
		Timeout:        *timeoutFlag,
		RecursiveDepth: *recursiveDepthFlag,
		Verbose:        *verboseFlag,
	})

	// Output results
	if *jsonFlag {
		printVerifyJSON(report.Results, filename, report.Verified, report.Counterexample, report.Skipped, report.Errors, report.Uncontracted)
	} else {
		printVerifyHuman(report.Results, filename, report.Verified, report.Counterexample, report.Skipped, report.Errors, report.Uncontracted, *verboseFlag)
	}

	// Exit code logic
	if report.Counterexample > 0 {
		os.Exit(1) // Contract violations found
	}
	// Strict mode is deliberately unchanged: an uncontracted function is not a
	// verification failure, it is a function nobody has written a contract for.
	// Counting it here would turn --strict red on every module with a helper.
	if *strictFlag && (report.Skipped > 0 || report.Errors > 0) {
		os.Exit(1) // Strict mode: non-verifiable functions are failures
	}
}

// verifyModulesFromPipeline projects the pipeline's loaded modules onto the
// (File, Core) pair the verifier consumes, so internal/smt does not depend on
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
		mods[path] = smt.VerifyModule{File: mod.File, Core: mod.Core}
	}
	return mods
}

// verifyPackageCommand is the --package arm: Z3 check, verify, print, exit.
func verifyPackageCommand(dir string, opts verifyPackageOptions, jsonOut, strict bool) {
	if !smt.Z3Available() {
		fmt.Fprintf(os.Stderr, "%s Z3 solver not found\n", red("Error:"))
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
	if report.Counterexample > 0 || report.Errors > 0 {
		os.Exit(1)
	}
	if strict && report.Skipped > 0 {
		os.Exit(1)
	}
}
