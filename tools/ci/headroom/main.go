// Command headroom is a WARN-ONLY per-package headroom reporter for the
// `go test ./...` step in CI. It consumes the log that step already produces
// (it NEVER re-runs the test suite), parses each `ok <pkg> <N>s` /
// `FAIL <pkg> <N>s` line, and reports the slowest packages as a percentage of
// the derived `-timeout` budget.
//
// Exit codes:
//
//	0  — report printed; a slow package is a data point, not a failure (WARN-ONLY)
//	1  — a broken INSTRUMENT, three conditions: cannot-read-log, anti-vacuity
//	     (non-empty log, zero records), or a package count below the floor.
//	     All three fail loudly. Distinct from a slow package.
//	2  — usage error.
//
// The three non-zero conditions sharing exit 1, the usage path (2), and the
// WARN-ONLY path (0) are kept distinct in code and in tests: a slow package
// must never red the job, and a silent instrument must never pass.
package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	// budgetWarnPct is the first WARN tier: a package at or above this share of
	// the budget is approaching the ceiling. WARN-ONLY — never reds the job.
	budgetWarnPct = 75
	// budgetHighWarnPct is the second, higher-severity WARN tier.
	budgetHighWarnPct = 90
	// topN is how many slowest packages the report lists.
	topN = 5
	// minPackages is the package-count floor. A full-suite `go test ./...` run
	// reports ~127-131 packages; a parser finding fewer than 50 is broken.
	minPackages = 50
)

// record is one parsed package-timing line.
type record struct {
	pkg        string
	seconds    float64
	secondsStr string // raw captured value, e.g. "100", "12.3", "228.7"
	failed     bool   // true when the line was `FAIL ... <N>s`
}

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout))
}

// run is the testable entry point. args includes args[0] (program name);
// args[1] is the log path (or "-" for stdin) and args[2] is the budget in
// seconds. All output — report, ::warning::, ::error:: — goes to stdout, which
// is what GitHub Actions scrapes.
func run(args []string, stdin io.Reader, stdout io.Writer) int {
	if len(args) < 3 {
		fmt.Fprintln(stdout, "usage: headroom <logfile|-> <budget-seconds>")
		return 2
	}
	logPath := args[1]
	budget, err := strconv.ParseFloat(args[2], 64)
	if err != nil || budget <= 0 {
		fmt.Fprintln(stdout, "usage: headroom <logfile|-> <budget-seconds>")
		return 2
	}

	var data []byte
	if logPath == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(logPath)
	}
	if err != nil {
		fmt.Fprintf(stdout, "::error:: headroom: cannot read log: %v\n", err)
		return 1
	}

	records := parse(data)
	n := len(records)

	// Anti-vacuity guard: a non-empty log that parses to zero records means the
	// parser has gone stale — a silent instrument is worse than a noisy one.
	if len(data) > 0 && n == 0 {
		fmt.Fprintln(stdout, "::error:: headroom: parsed 0 packages from non-empty log — parser may be stale")
		return 1
	}
	// Package-count floor: a parser finding 3 of 130 packages is as broken as
	// one finding 0. Checked after the zero-on-non-empty case (an empty log
	// fails here via 0 < 50, which is correct).
	if n < minPackages {
		fmt.Fprintf(stdout, "::error:: headroom: parsed %d packages (< %d) — parser may be stale\n", n, minPackages)
		return 1
	}

	// Sort by seconds descending, ties broken by package name ascending
	// (deterministic CI logs).
	sort.Slice(records, func(i, j int) bool {
		if records[i].seconds != records[j].seconds {
			return records[i].seconds > records[j].seconds
		}
		return records[i].pkg < records[j].pkg
	})

	fmt.Fprintf(stdout, "HEADROOM: top %d slowest packages (budget %ss)\n", topN, args[2])
	top := records
	if len(top) > topN {
		top = top[:topN]
	}
	for _, r := range top {
		pct := percentOfBudget(r.seconds, budget)
		line := fmt.Sprintf("%s %ss (%d%% of budget)", r.pkg, r.secondsStr, pct)
		if r.failed {
			line += " [go test FAIL]"
		}
		fmt.Fprintln(stdout, line)
	}

	// WARN tiers — WARN-ONLY. Both report; neither reds the job.
	for _, r := range records {
		pct := percentOfBudget(r.seconds, budget)
		switch {
		case pct >= budgetHighWarnPct:
			fmt.Fprintf(stdout, "::warning::headroom: %s at %d%% of budget (warn threshold %d%%)\n", r.pkg, pct, budgetHighWarnPct)
		case pct >= budgetWarnPct:
			fmt.Fprintf(stdout, "::warning::headroom: %s at %d%% of budget (warn threshold %d%%)\n", r.pkg, pct, budgetWarnPct)
		}
	}

	return 0
}

// parse extracts package-timing records from go test output by splitting each
// line on TAB and inspecting the fields — NOT by a hand-anchored literal
// prefix. Go pads the status column so `ok` aligns with `FAIL`, so a real line
// is `ok` + two spaces + TAB + pkg + TAB + duration; a prefix regex that
// expects `ok\t` misses every real line. Field 0 trimmed of spaces must be
// exactly `ok` or `FAIL` (a `?` line is not a timing record); field 1 is the
// package; field 2 is the duration or `(cached)`.
//
// Lines that do not match a record shape (a `FAIL <pkg> [build failed]` line, a
// bare `FAIL`, panics, build output, etc.) are skipped — that is what the
// runtime anti-vacuity guard is for.
func parse(data []byte) []record {
	var records []record
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue // bare FAIL, ? line, build output, etc.
		}
		status := strings.TrimSpace(fields[0])
		if status != "ok" && status != "FAIL" {
			continue // a `?` line is not a timing record
		}
		pkg := strings.TrimSpace(fields[1])
		// Field 2 is the duration (e.g. "12.3s", possibly with a
		// "[no tests to run]" suffix) or "(cached)".
		if fields[2] == "(cached)" {
			// A cached package consumed none of this run's budget (0 seconds),
			// but it must still count toward the package-count floor so a
			// cache-warm run does not trip the floor for the wrong reason.
			records = append(records, record{pkg: pkg, seconds: 0, secondsStr: "0"})
			continue
		}
		// The duration is the first whitespace-delimited token of field 2.
		toks := strings.Fields(fields[2])
		if len(toks) == 0 {
			continue
		}
		numStr := strings.TrimSuffix(toks[0], "s")
		sec, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			continue // e.g. `FAIL <pkg> [build failed]` — not a timing record
		}
		records = append(records, record{
			pkg:        pkg,
			seconds:    sec,
			secondsStr: numStr,
			failed:     status == "FAIL",
		})
	}
	return records
}

// percentOfBudget rounds 100 * seconds / budget to the nearest integer.
func percentOfBudget(seconds, budget float64) int {
	return int(math.Round(100 * seconds / budget))
}
