package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// printPhaseTimings outputs a formatted breakdown of compilation phase timings.
// When jsonFlag is true, timings are routed to stderr so stdout remains valid JSON.
func printPhaseTimings(timings map[string]int64, jsonFlag bool) {
	w := io.Writer(os.Stdout)
	if jsonFlag {
		w = os.Stderr
	}
	printPhaseTimingsTo(w, timings)
}

// printPhaseTimingsTo writes the formatted compilation phase timing table to w.
func printPhaseTimingsTo(w io.Writer, timings map[string]int64) {
	fmt.Fprintf(w, "\n%s Compilation Phase Timings:\n", cyan("⏱"))

	phaseOrder := []string{
		"load", "topo", "parse", "elaborate", "typecheck",
		"dict_elaboration", "monomorphization", "lower",
		"dict_elab", "anf_verify", "link", "compile", "evaluate",
	}
	phaseNames := map[string]string{
		"load":             "Loading",
		"topo":             "Topo Sort",
		"parse":            "Parsing",
		"elaborate":        "Elaboration",
		"typecheck":        "Type Checking",
		"dict_elaboration": "Dict Elaboration",
		"monomorphization": "Monomorphization",
		"lower":            "Lowering",
		"dict_elab":        "Dict Linking",
		"anf_verify":       "ANF Verify",
		"link":             "Linking",
		"compile":          "Compile",
		"evaluate":         "Evaluate",
	}

	var total int64
	var slowestPhase string
	var slowestTime int64
	for _, ms := range timings {
		total += ms
		if ms > slowestTime {
			slowestTime = ms
		}
	}

	for _, phase := range phaseOrder {
		if ms, ok := timings[phase]; ok {
			name := phaseNames[phase]
			if name == "" {
				name = phase
			}
			marker := ""
			if ms == slowestTime && ms > 100 {
				marker = yellow(" ← Slowest")
				slowestPhase = name
			} else if ms > 100 {
				marker = yellow(" ⚠")
			}
			fmt.Fprintf(w, "  %-18s %6dms%s\n", name+":", ms, marker)
		}
	}

	var extra []string
	for phase := range timings {
		found := false
		for _, p := range phaseOrder {
			if p == phase {
				found = true
				break
			}
		}
		if !found {
			extra = append(extra, phase)
		}
	}
	sort.Strings(extra)
	for _, phase := range extra {
		ms := timings[phase]
		fmt.Fprintf(w, "  %-18s %6dms\n", phase+":", ms)
	}

	fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 28))
	fmt.Fprintf(w, "  %-18s %6dms\n", "Total:", total)

	if slowestPhase != "" && slowestTime > 100 {
		fmt.Fprintf(w, "\n%s %s took %dms (threshold: 100ms)\n",
			yellow("Warning:"), slowestPhase, slowestTime)
		fmt.Fprintln(w, "  Consider checking for complex recursive types.")
	}
}
