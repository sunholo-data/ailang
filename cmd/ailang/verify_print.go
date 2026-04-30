package main

// verify_print.go — output formatting for `ailang verify`.
//
// Extracted from verify.go in the M-SMT-CROSS-MODULE-TYPES follow-up to keep
// verify.go under the 800-line organisation budget. Contains only formatting
// helpers — no verification logic.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/smt"
)

// printVerifyHuman prints human-readable verification results.
func printVerifyHuman(results []verifyResult, filename string, verified, counterexample, skipped, errCount int, verbose bool) {
	total := verified + counterexample + skipped + errCount

	fmt.Printf("\n%s Verifying contracts in %s\n", cyan("→"), filename)
	if z3ver := smt.Z3Version(); z3ver != "" {
		fmt.Printf("  Solver: %s\n", z3ver)
	}
	fmt.Println()

	hasBounded := false
	for _, r := range results {
		switch r.Status {
		case "verified":
			if r.BoundedDepth > 0 {
				fmt.Printf("  %s %s  %s\n", green(fmt.Sprintf("✓ VERIFIED (bounded: depth %d)", r.BoundedDepth)), bold(r.Function), dim(r.Duration.String()))
				hasBounded = true
			} else {
				fmt.Printf("  %s %s  %s\n", green("✓ VERIFIED"), bold(r.Function), dim(r.Duration.String()))
			}
		case "counterexample":
			fmt.Printf("  %s %s\n", red("✗ VIOLATION"), bold(r.Function))
			if len(r.Model) > 0 {
				fmt.Printf("    Counterexample:\n")
				for _, b := range r.Model {
					fmt.Printf("      %s: %s = %s\n", b.Name, b.Sort, b.Value)
				}
			}
		case "skipped":
			fmt.Printf("  %s %s\n", yellow("⚠ SKIPPED"), bold(r.Function))
			if r.Reason != "" {
				fmt.Printf("    Reason: %s\n", r.Reason)
			}
			if len(r.Rejections) > 0 {
				for _, rej := range r.Rejections {
					if rej.Hint != "" {
						fmt.Printf("    Hint: %s\n", rej.Hint)
					}
				}
			}
		case "error", "unknown":
			fmt.Printf("  %s %s\n", red("! ERROR"), bold(r.Function))
			if r.Reason != "" {
				fmt.Printf("    %s\n", r.Reason)
			}
		}

		if verbose && r.SMTLib != "" {
			fmt.Printf("    SMT-LIB:\n")
			for _, line := range strings.Split(r.SMTLib, "\n") {
				if line != "" {
					fmt.Printf("      %s\n", line)
				}
			}
			fmt.Println()
		}
	}

	// Summary line
	fmt.Println()
	summary := fmt.Sprintf("  %d functions: ", total)
	parts := []string{}
	if verified > 0 {
		parts = append(parts, green(fmt.Sprintf("%d verified", verified)))
	}
	if counterexample > 0 {
		parts = append(parts, red(fmt.Sprintf("%d violations", counterexample)))
	}
	if skipped > 0 {
		parts = append(parts, yellow(fmt.Sprintf("%d skipped", skipped)))
	}
	if errCount > 0 {
		parts = append(parts, red(fmt.Sprintf("%d errors", errCount)))
	}
	if len(parts) == 0 {
		parts = append(parts, "no functions with contracts")
	}
	fmt.Printf("%s%s\n", summary, strings.Join(parts, ", "))

	if hasBounded {
		fmt.Printf("\n  %s \"bounded: depth N\" means the property was verified assuming at most N\n", dim("Note:"))
		fmt.Printf("  %s levels of recursion. This is sound but not a full inductive proof.\n", dim("      "))
	}
	fmt.Println()
}

// printVerifyJSON outputs verification results as JSON.
func printVerifyJSON(results []verifyResult, filename string, verified, counterexample, skipped, errCount int) {
	output := struct {
		File           string         `json:"file"`
		Verified       int            `json:"verified"`
		Counterexample int            `json:"counterexample"`
		Skipped        int            `json:"skipped"`
		Errors         int            `json:"errors"`
		Results        []verifyResult `json:"results"`
	}{
		File:           filename,
		Verified:       verified,
		Counterexample: counterexample,
		Skipped:        skipped,
		Errors:         errCount,
		Results:        results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: JSON encoding error: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
