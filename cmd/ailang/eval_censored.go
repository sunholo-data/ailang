package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
)

// loadCensoredArm loads one arm for the censored-pair analyzer.
//
// It deliberately does NOT use LoadArmForPairing: that helper wraps
// FilterValidResults, which drops every row whose Validity is invalid at LOAD
// time. The censored-pair analyzer's own contract (design doc section 5) is to
// SEE those rows, drop them itself, and COUNT them — the ">20% of ON rows
// quarantined" gate is defined over the banked set, not over the survivors.
// Loading through the filtering helper makes that gate unreachable (it can
// never observe a quarantined row) and additionally corrupts the section 5.3
// order-integrity verdict, because the executed order is a fact about the run
// and a silently-dropped row changes the block sequence.
//
// Measured on the banked AC5 arms: the filtering loader returns 5 ON rows with
// 0 quarantined visible, the raw loader returns 6 with 1 visible; and the order
// gate's refusal reason changes from order_integrity_unpaired_block (an
// artifact of the odd count) to order_integrity_nonadjacent_arms (the defect
// the data actually has).
func loadCensoredArm(dir string) ([]*eval_analysis.BenchmarkResult, error) {
	return eval_analysis.LoadResultsFromDirsIncludingInvalid(dir)
}

// evalCensoredPairs is the testable body of the command: it loads both arms the
// way the command does and returns the analyzer's verdict.
func evalCensoredPairs(onDir, offDir string) (eval_analysis.CensoredPairResult, error) {
	on, err := loadCensoredArm(onDir)
	if err != nil {
		return eval_analysis.CensoredPairResult{}, fmt.Errorf("loading ON arm %s: %w", onDir, err)
	}
	off, err := loadCensoredArm(offDir)
	if err != nil {
		return eval_analysis.CensoredPairResult{}, fmt.Errorf("loading OFF arm %s: %w", offDir, err)
	}
	return eval_analysis.AnalyzeCensoredPairs(on, off), nil
}

func runEvalCensoredPairs() {
	fs := flag.NewFlagSet("eval-censored-pairs", flag.ExitOnError)
	pretty := fs.Bool("pretty", false, "Indent the JSON output")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	args := fs.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: ailang eval-censored-pairs [flags] <on-dir> <off-dir>")
		fmt.Fprintln(os.Stderr, "\nApplies fmt treatment/order integrity gates and the pre-registered censored-pair decision rule.")
		os.Exit(1)
	}

	result, err := evalCensoredPairs(args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding result: %v\n", err)
		os.Exit(1)
	}
	if result.Verdict == eval_analysis.CensoredVerdictVoid {
		os.Exit(1)
	}
}
