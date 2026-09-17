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
	Available      bool               `json:"available"`
	Verified       int                `json:"verified"`
	Counterexample int                `json:"counterexample"`
	Skipped        int                `json:"skipped"`
	Errors         int                `json:"errors"`
	Results        []smt.VerifyResult `json:"results"`
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

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
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
		os.Exit(1)
	}

	runAICheckReport(fs.Arg(0), *timeoutFlag, *recursiveDepthFlag, *relaxModulesFlag)
}

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
// It never returns: every lane ends in os.Exit or falls off the end after the
// JSON is printed.
func runAICheckReport(filename string, timeout time.Duration, recursiveDepth int, relaxModules bool) {
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
			Verify: aiVerifySection{Available: false, Results: []smt.VerifyResult{}},
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

	// Build verify section
	verifySection := aiVerifySection{
		Available: false,
		Results:   []smt.VerifyResult{},
	}

	// Only attempt verification if check passed and Z3 is available
	if checkSection.Passed && smt.Z3Available() {
		verifySection.Available = true

		coreProg := result.Artifacts.Core
		surfaceAST := result.Artifacts.AST

		if coreProg != nil && coreProg.Meta != nil && surfaceAST != nil {
			report := smt.Verify(coreProg, surfaceAST, verifyModulesFromPipeline(result), aiCheckVerifyOptions(timeout, recursiveDepth))
			verifySection = aiVerifySectionFromReport(report)
		}
	}

	output := aiCheckOutput{
		File:   filename,
		Check:  checkSection,
		Verify: verifySection,
	}

	// The JSON is ALWAYS emitted before the exit decision: `ai-check` is an AI
	// convergence signal, so a consumer must be able to read the full report even
	// on the failing exit path.
	outputAICheck(output)

	if aiCheckExitCode(checkSection, verifySection) != 0 {
		os.Exit(1)
	}
}

// aiCheckExitCode decides `ai-check`'s process exit status from its own JSON report.
//
// Extracted from aiCheckCommand so the exit lanes are unit-testable without a
// subprocess (os.Exit is untestable in-process). The contract, which the eval
// harness and every agent convergence loop depend on:
//
//	check failed        -> 1
//	counterexample > 0  -> 1
//	verifier errors > 0 -> 1   (added by M-Z3-ADT-RECORD-SORT; process status must
//	                            never disagree with the JSON it just printed)
//	skipped-only        -> 0   (a skip is "not proved", not "disproved")
//	verified-only       -> 0
func aiCheckExitCode(check aiCheckSection, verify aiVerifySection) int {
	if !check.Passed || verify.Counterexample > 0 || verify.Errors > 0 {
		return 1
	}
	return 0
}

// aiCheckVerifyOptions is how ai-check drives the ONE verification loop
// (smt.Verify — the same loop `ailang verify` runs). Until M-V1-SIMPLIFY-S4
// M3A ai-check carried a private copy that had drifted in three measured
// ways: it passed no ImportedPrograms (so every cross-module callee was an
// "unknown constant" Z3 error — cross_module_functions.ail: verify 3 verified
// / ai-check 3 errors), it ignored a per-function @verify(depth: N) override
// (fibonacci bounded_depth 2 instead of 5), and it counted an unsupported
// construct as an error where verify skips it. The first two are now shared.
// The third is kept as a deliberate ai-check difference behind
// UnsupportedConstructIsError: the exit-code contract pinned in
// ai_check_exit_test.go has always surfaced it as a verifier error (exit 1),
// and the eval harness banks verify_errors from it — changing that is a
// bank-forward ruling, not a refactor.
func aiCheckVerifyOptions(timeout time.Duration, recursiveDepth int) smt.VerifyOptions {
	return smt.VerifyOptions{
		Timeout:                     timeout,
		RecursiveDepth:              recursiveDepth,
		UnsupportedConstructIsError: true,
	}
}

// aiVerifySectionFromReport projects a VerifyReport onto ai-check's JSON.
// "uncontracted" rows (exported functions with no contract, which verify
// lists for denominator honesty) are left out: ai-check's results have only
// ever carried contract-bearing functions, its consumers (the eval harness,
// agent convergence loops) read the four counters, and adding a fifth status
// value to a machine contract is a change to announce, not to slip in.
func aiVerifySectionFromReport(report *smt.VerifyReport) aiVerifySection {
	section := aiVerifySection{
		Available:      true,
		Verified:       report.Verified,
		Counterexample: report.Counterexample,
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
