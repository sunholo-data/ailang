package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
)

// runEvalPaired emits a PAIRED comparison of two A/B arms as JSON.
//
// Exists so the weekly A/B in tools/launchd/nightly-eval.sh can bank
// per-benchmark pairs and a McNemar result without reimplementing the join in
// bash — the previous shell-side arithmetic is exactly what produced the
// 2026-07-20 artefact (an empty pass count coerced to a literal 0).
//
// Arms are read WITHOUT the re-run dedup: that key omits Trial, so a --trials N
// run would otherwise lose half its observations and the arms would stop
// lining up. See eval_analysis.LoadArmForPairing.
func runEvalPaired() {
	fs := flag.NewFlagSet("eval-paired", flag.ExitOnError)
	pretty := fs.Bool("pretty", false, "Indent the JSON output")
	withPairs := fs.Bool("with-pairs", true, "Include the per-benchmark pairs array (set false for a compact summary)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	args := fs.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: ailang eval-paired [flags] <on-arm-dir> <off-arm-dir>")
		fmt.Fprintln(os.Stderr, "\nEmits paired per-benchmark outcomes plus a McNemar result.")
		os.Exit(1)
	}

	on, err := eval_analysis.LoadArmForPairing(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ON arm %s: %v\n", args[0], err)
		os.Exit(1)
	}
	off, err := eval_analysis.LoadArmForPairing(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading OFF arm %s: %v\n", args[1], err)
		os.Exit(1)
	}

	result := eval_analysis.PairArms(on, off)
	if !*withPairs {
		result.Pairs = nil
	}

	enc := json.NewEncoder(os.Stdout)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding result: %v\n", err)
		os.Exit(1)
	}
}
