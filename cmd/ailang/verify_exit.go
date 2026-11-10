package main

// verify_exit.go — the stderr half of the shared verification exit contract
// (M-V1-VERIFICATION-HONESTY M1). The exit status itself is decided by
// verification.ExitCode; this file makes every lane that is not a clean proof
// say so on stderr, so a consumer that reads only the exit code and a human
// who reads only the terminal both see that something was not proved.
//
// Used by `check --verify`, `ai-check`, `verify` and `verify --package`.

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

// parseInterspersed parses flags wherever they appear among the positional
// arguments and leaves the positionals as fs.Args().
//
// Go's flag package stops at the first positional argument, so
// `check --verify f.ail --strict` used to drop --strict (it became a second
// "file" nobody read). The flag sets use ExitOnError, so a bad flag still exits
// 2 with usage. Arguments after a literal "--" stay positional.
func parseInterspersed(fs *flag.FlagSet, args []string) {
	var positionals []string
	_ = fs.Parse(args)
	for fs.NArg() > 0 {
		rest := fs.Args()
		positionals = append(positionals, rest[0])
		_ = fs.Parse(rest[1:])
	}
	_ = fs.Parse(append([]string{"--"}, positionals...))
}

// verificationFileArg returns the one file a verification command checks, or
// "" when none was given. A second file is a usage error (exit 2): the
// commands check one file, and `check --verify ok.ail bad.ail` used to verify
// ok.ail, exit 0 and never look at bad.ail.
func verificationFileArg(fs *flag.FlagSet, command string) string {
	if fs.NArg() < 1 {
		return ""
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(os.Stderr, "%s: %s checks one file at a time; %q would not be checked (only %q would be). Run it once per file.\n",
			red("Error"), command, fs.Arg(1), fs.Arg(0))
		os.Exit(2)
	}
	return fs.Arg(0)
}

// writeZ3NotFound prints where the solver was looked for and how to provide
// one. It names the actual search (smt.Z3Lookup) rather than assuming one
// platform's installer.
func writeZ3NotFound(w io.Writer, what string) {
	fmt.Fprintf(w, "Error: Z3 solver not found (%s): %s\n", smt.Z3Lookup(), what)
	fmt.Fprintln(w, "Install Z3 (https://github.com/Z3Prover/z3/releases, or your package manager's z3 package) and put it on PATH, or set AILANG_Z3_PATH to the z3 binary.")
}

// writeVerificationNotice prints the stderr lines for one run's outcome.
// results may be nil (the solver did not run). It prints nothing for a clean
// proof or a file with no obligations.
func writeVerificationNotice(w io.Writer, t verification.Tally, p verification.ExitPolicy, results []smt.VerifyResult) {
	switch verification.Classify(t) {
	case verification.OutcomeUnavailable:
		writeZ3NotFound(w, fmt.Sprintf("%d contract obligation(s) were not verified (verify.outcome: unavailable)", t.Obligations))
		return
	case verification.OutcomeNotRun, verification.OutcomeNoObligations, verification.OutcomeVerified,
		verification.OutcomeNotRequested: // --no-verify: the caller asked for no verifier
		return
	case verification.OutcomeUnproved:
		fmt.Fprintf(w, "Error: no contract was proved: %d contract(s) skipped and none verified (verify.outcome: unproved): %s\n",
			t.Skipped, notProvedList(results, "skipped"))
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
		if r.Status != status || !smt.IsObligation(r) {
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
