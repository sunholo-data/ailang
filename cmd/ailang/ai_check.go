package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

// aiCheckOutput is the unified JSON output for ai-check
type aiCheckOutput struct {
	File   string          `json:"file"`
	Check  aiCheckSection  `json:"check"`
	Verify aiVerifySection `json:"verify"`
}

// aiCheckSection is the type-check portion of ai-check output
type aiCheckSection struct {
	Passed     bool             `json:"passed"`
	ErrorCount int              `json:"error_count"`
	Errors     []checkJSONError `json:"errors"`
}

// aiVerifySection is the contract verification portion of ai-check output
type aiVerifySection struct {
	verification.Evidence
	// Outcome is the run's verdict under the shared exit contract
	// (verification.Classify); Obligations counts the contract-bearing
	// functions, known even when the solver is not. Both are additive
	// (M-V1-VERIFICATION-HONESTY M1): Available keeps meaning "solver found".
	Outcome        verification.Outcome `json:"outcome"`
	Obligations    int                  `json:"obligations"`
	Available      bool                 `json:"available"`
	Verified       int                  `json:"verified"`
	Counterexample int                  `json:"counterexample"`
	// Inconclusive: obligations attempted but undecided — a sat under bounded
	// recursion unrolling (M-VERIFY-BOUNDED-UNROLLING-FALSE-COUNTEREXAMPLE).
	Inconclusive int                `json:"inconclusive"`
	Skipped      int                `json:"skipped"`
	Errors       int                `json:"errors"`
	Results      []smt.VerifyResult `json:"results"`
}

// The verification defaults of the unified check+verify report. They are named
// because TWO flag sets now offer them — `ai-check`'s below and `check
// --verify`'s in commands_language.go — and a report that answered differently
// depending on which spelling reached it would be the same drift the S4 M3A
// fold closed.
const (
	aiCheckDefaultTimeout        = 5 * time.Second
	aiCheckDefaultRecursiveDepth = 2
)

// aiCheckCommand implements the `ailang ai-check` CLI command.
// It runs type checking + contract verification in a single invocation
// with unified JSON output designed for AI/machine consumption.
//
// S5 M2 absorbed this into `ailang check --verify` and kept `ai-check` as an
// alias of `check` (D1: the eval harness, the agent convergence loops and the
// DP7 done-gate all call it). This function is the LEGACY spelling's entry
// point — its own flag set, unchanged, so the bytes it prints and the status
// it exits with are the ones ai_check_exit_test.go pins.
func aiCheckCommand() {
	fs := flag.NewFlagSet("ai-check", flag.ExitOnError)
	timeoutFlag := fs.Duration("timeout", aiCheckDefaultTimeout, "Per-function Z3 timeout (hard backstop adds 2s grace)")
	recursiveDepthFlag := fs.Int("verify-recursive-depth", aiCheckDefaultRecursiveDepth, "Bounded recursion unrolling depth (1-10, 0 to disable)")
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches with warning)")
	strictFlag := fs.Bool("strict", false, strictFlagHelp)
	allowInconclusiveFlag := fs.Bool("allow-inconclusive", false, allowInconclusiveFlagHelp)

	parseInterspersed(fs, os.Args[2:])
	file := verificationFileArg(fs, "ai-check")

	if file == "" {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		fmt.Println("Usage: ailang ai-check [options] <file.ail>")
		fmt.Println()
		fmt.Println("Runs type checking + contract verification in one pass.")
		fmt.Println("Always outputs JSON (designed for AI/machine consumption).")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --timeout                 Per-function Z3 timeout; hard backstop adds 2s grace (default: 5s)")
		fmt.Println("  --verify-recursive-depth  Bounded recursion depth (default: 2)")
		fmt.Println("  --relax-modules           Relax MOD010 validation")
		fmt.Println("  --strict                  " + strictFlagHelp)
		fmt.Println("  --allow-inconclusive      " + allowInconclusiveFlagHelp)
		os.Exit(1)
	}

	runAICheckReport(file, *timeoutFlag, *recursiveDepthFlag, *relaxModulesFlag,
		verification.ExitPolicy{Strict: *strictFlag, AllowInconclusive: *allowInconclusiveFlag})
}

// The two flags that move a lane of the shared exit contract
// (verification.ExitCode). Every verification command offers both.
const (
	strictFlagHelp            = "Exit 1 when any contract is skipped or inconclusive (only a full proof passes)"
	allowInconclusiveFlagHelp = "Exit 0 on inconclusive obligations (bounded-unrolling sat); they are still listed on stderr"
)

// runAICheckReport is the unified check+verify report: type-check and contract
// verification in ONE pipeline run, emitted as JSON, with the exit status
// aiCheckExitCode decides.
//
// It is reached by two spellings and exists exactly once. `ailang ai-check` is
// the original one and keeps its own flag set above; `ailang check --verify`
// is the absorbed one (S5 M2, Phase 3 item 2). Extracting it was the point of
// the absorption: a second copy of this function is how ai-check drifted from
// `verify` in three measured ways before M-V1-SIMPLIFY-S4 M3A folded the
// verification loops together, and adding a second entry point without
// extracting it would have re-created exactly that.
//
// It prints the JSON, then the stderr notice, then exits with
// verification.ExitCode of the numbers it just printed.
func runAICheckReport(filename string, timeout time.Duration, recursiveDepth int, relaxModules bool, policy verification.ExitPolicy) {
	// Suppress warnings so they don't pollute JSON output
	os.Setenv("AILANG_QUIET_WARNINGS", "1")

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		outputAICheck(aiCheckOutput{
			File: filename,
			Check: aiCheckSection{
				Passed:     false,
				ErrorCount: 1,
				Errors:     []checkJSONError{{Code: "IO_ERROR", Message: fmt.Sprintf("cannot read file: %v", err), File: filename}},
			},
			Verify: aiVerifySection{Outcome: verification.OutcomeNotRun, Available: smt.Z3Available(), Results: []smt.VerifyResult{}},
		})
		os.Exit(1)
	}

	// The flag and AILANG_RELAX_MODULES are OR-ed.
	relaxModulesEffective := relaxModules || config.RelaxModules()

	// Run pipeline ONCE (both check and verify use the same compilation result)
	cfg := pipeline.Config{
		DryLink:      true,
		RelaxModules: relaxModulesEffective,
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, pipelineErr := pipeline.Run(cfg, src)

	// Build check section
	checkSection := aiCheckSection{
		Passed:     true,
		ErrorCount: 0,
		Errors:     []checkJSONError{},
	}

	if pipelineErr != nil {
		checkSection.Passed = false
		checkSection.ErrorCount = 1
		checkSection.Errors = []checkJSONError{errorToCheckJSONError(pipelineErr, filename)}
	} else if len(result.Errors) > 0 {
		checkSection.Passed = false
		checkSection.ErrorCount = len(result.Errors)
		jsonErrors := make([]checkJSONError, len(result.Errors))
		for i, e := range result.Errors {
			jsonErrors[i] = errorToCheckJSONError(e, filename)
		}
		checkSection.Errors = jsonErrors
	}

	verifySection := buildAIVerifySection(checkSection.Passed, result, timeout, recursiveDepth, smt.Z3Available())
	tally := verifySection.tally(checkSection.Passed)
	verifySection.Outcome = verification.Classify(tally)

	// The JSON is ALWAYS emitted before the exit decision: `ai-check` is an AI
	// convergence signal, so a consumer must be able to read the full report even
	// on the failing exit path.
	outputAICheck(aiCheckOutput{File: filename, Check: checkSection, Verify: verifySection})
	writeVerificationNotice(os.Stderr, tally, policy, verifySection.Results)

	if code := verification.ExitCode(tally, policy); code != 0 {
		os.Exit(code)
	}
}

// buildAIVerifySection runs the verifier when the check passed and the solver
// is available. Without a solver it still counts the obligations it would
// have proved, from the compiled Core: that count is what turns "Z3 missing"
// from a silent pass into the `unavailable` lane. Available always means
// "solver found", on every lane.
func buildAIVerifySection(checkPassed bool, result pipeline.Result, timeout time.Duration, recursiveDepth int, solverAvailable bool) aiVerifySection {
	section := aiVerifySection{Available: solverAvailable, Results: []smt.VerifyResult{}}
	if !checkPassed {
		return section
	}
	coreProg := result.Artifacts.Core
	surfaceAST := result.Artifacts.AST
	if !solverAvailable {
		section.Obligations = smt.CountObligations(coreProg)
		return section
	}
	if coreProg == nil || coreProg.Meta == nil || surfaceAST == nil {
		return section
	}
	opts := aiCheckVerifyOptions(timeout, recursiveDepth)
	opts.CoreTI = result.Artifacts.CoreTI
	return aiVerifySectionFromReport(smt.Verify(coreProg, surfaceAST, verifyModulesFromPipeline(result), opts))
}

// tally projects the JSON section onto the shared exit contract, so the exit
// status is computed from exactly the numbers the consumer just read.
//
// With a solver it is computed from the result rows the report carries
// (smt.TallyResults), which leaves requires-only rows out: they count in the
// legacy `skipped` field but are not obligations. Without a solver, or when
// the check failed, only Obligations (counted from Core) is known.
func (v aiVerifySection) tally(checkPassed bool) verification.Tally {
	if checkPassed && v.Available {
		return smt.TallyResults(v.Results)
	}
	return verification.Tally{
		CheckPassed:     checkPassed,
		SolverAvailable: v.Available,
		Obligations:     v.Obligations,
	}
}

// aiCheckVerifyOptions is how ai-check drives the ONE verification loop
// (smt.Verify — the same loop `ailang verify` runs). Until M-V1-SIMPLIFY-S4
// M3A ai-check carried a private copy that had drifted in three measured
// ways: it passed no ImportedPrograms (so every cross-module callee was an
// "unknown constant" Z3 error — cross_module_functions.ail: verify 3 verified
// / ai-check 3 errors), it ignored a per-function @verify(depth: N) override
// (fibonacci bounded_depth 2 instead of 5), and it counted an unsupported
// construct as an error where verify skips it. All three are now shared; the
// third went with the shared exit contract (M-V1-VERIFICATION-HONESTY M1),
// under which an unsupported construct is a skip on every command.
func aiCheckVerifyOptions(timeout time.Duration, recursiveDepth int) smt.VerifyOptions {
	return smt.VerifyOptions{
		Timeout:        timeout,
		RecursiveDepth: recursiveDepth,
	}
}

// aiVerifySectionFromReport projects a VerifyReport onto ai-check's JSON.
// "uncontracted" rows (exported functions with no contract, which verify
// lists for denominator honesty) are left out: ai-check's results have only
// ever carried contract-bearing functions, its consumers (the eval harness,
// agent convergence loops) read the counters, and adding a status value to a
// machine contract is a change to announce, not to slip in. ("inconclusive"
// was announced: M-VERIFY-BOUNDED-UNROLLING-FALSE-COUNTEREXAMPLE, with the
// harness readers landed first.)
func aiVerifySectionFromReport(report *smt.VerifyReport) aiVerifySection {
	section := aiVerifySection{
		Evidence:       report.Evidence,
		Obligations:    report.Obligations(),
		Available:      true,
		Verified:       report.Verified,
		Counterexample: report.Counterexample,
		Inconclusive:   report.Inconclusive,
		Skipped:        report.Skipped,
		Errors:         report.Errors,
		Results:        make([]smt.VerifyResult, 0, len(report.Results)),
	}
	for _, r := range report.Results {
		if r.Status == "uncontracted" {
			continue
		}
		section.Results = append(section.Results, r)
	}
	return section
}

// outputAICheck writes the unified ai-check JSON output
func outputAICheck(out aiCheckOutput) {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
