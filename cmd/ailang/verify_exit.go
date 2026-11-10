package main

// verify_exit.go — the stderr half of the shared verification exit contract
// (M-V1-VERIFICATION-HONESTY M1). The exit status itself is decided by
// verification.ExitCode; this file makes every lane that is not a clean proof
// say so on stderr, so a consumer that reads only the exit code and a human
// who reads only the terminal both see that something was not proved.
//
// Used by `check --verify`, `ai-check`, `verify` and `verify --package`.

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

// z3InstallLines is the one install hint every verification command prints
// when the solver is missing.
var z3InstallLines = []string{
	"Install with: brew install z3 (macOS), apt install z3 (Linux), or choco install z3 / scoop install z3 (Windows)",
	"Or download from https://github.com/Z3Prover/z3/releases and set AILANG_Z3_PATH.",
}

// writeVerificationNotice prints the stderr lines for one run's outcome.
// results may be nil (the solver did not run). It prints nothing for a clean
// proof or a file with no obligations.
func writeVerificationNotice(w io.Writer, t verification.Tally, p verification.ExitPolicy, results []smt.VerifyResult) {
	switch verification.Classify(t) {
	case verification.OutcomeUnavailable:
		fmt.Fprintf(w, "Error: Z3 solver not found: %d contract obligation(s) were not verified (verify.outcome: unavailable)\n", t.Obligations)
		for _, l := range z3InstallLines {
			fmt.Fprintln(w, l)
		}
		return
	case verification.OutcomeNotRun, verification.OutcomeNoObligations, verification.OutcomeVerified:
		return
	}
	if t.Inconclusive > 0 {
		verdict := "fails this run"
		if p.AllowInconclusive && !p.Strict {
			verdict = "accepted by --allow-inconclusive"
		}
		fmt.Fprintf(w, "%d contract(s) inconclusive (not proved; %s): %s\n",
			t.Inconclusive, verdict, notProvedList(results, "inconclusive"))
	}
	if t.Skipped > 0 {
		verdict := "pass --strict to fail on these"
		if p.Strict {
			verdict = "--strict fails this run"
		}
		fmt.Fprintf(w, "%d contract(s) not verified (%s): %s\n",
			t.Skipped, verdict, notProvedList(results, "skipped"))
	}
}

// notProvedList renders "f (reason), g (reason)" for the results with the
// given status, sorted by function name so the line is deterministic.
func notProvedList(results []smt.VerifyResult, status string) string {
	var parts []string
	for _, r := range results {
		if r.Status != status {
			continue
		}
		reason := shortReason(r)
		if reason == "" {
			parts = append(parts, r.Function)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", r.Function, reason))
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return "(see the report)"
	}
	return strings.Join(parts, ", ")
}

// shortReason is the first clause of a result's reason, capped so one stderr
// line stays readable; the full text is in the report.
func shortReason(r smt.VerifyResult) string {
	reason := r.Reason
	if i := strings.Index(reason, "; "); i >= 0 {
		reason = reason[:i]
	}
	const maxRunes = 100
	if runes := []rune(reason); len(runes) > maxRunes {
		reason = string(runes[:maxRunes]) + "…"
	}
	if reason == "" && r.ReasonCode != "" {
		return string(r.ReasonCode)
	}
	return reason
}
