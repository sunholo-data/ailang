package main

import (
	"fmt"
	"io"
	"strings"
)

var statusOrder = []string{"verified", "counterexample", "inconclusive", "unknown", "error", "skipped", "skip_mp", "uncontracted"}

// report prints the gap map and the diff, and returns the number of regressions.
func report(w io.Writer, cur, prev *snapshot, prevPath string, n int) int {
	model := ""
	if cur.IntModel != "" {
		model = " — --int-model=" + cur.IntModel
	}
	fmt.Fprintf(w, "verify-gap-map — %s — %s%s\n\n", cur.Ailang, cur.Taken, model)

	fmt.Fprintln(w, "Corpora")
	for _, c := range cur.Corpora {
		switch {
		case c.Note != "" && c.Files == 0:
			fmt.Fprintf(w, "  %-16s not measured: %s\n", c.Name, c.Note)
		default:
			dirty := ""
			if c.Dirty {
				dirty = " (uncommitted changes)"
			}
			fmt.Fprintf(w, "  %-16s %-12s %4d files %5d results%s\n", c.Name, c.Head, c.Files, c.Rows, dirty)
		}
	}

	total := len(cur.Rows)
	obligations := total - cur.Totals["uncontracted"]
	fmt.Fprintf(w, "\nStatus (%d contracted results; uncontracted exports listed separately)\n", obligations)
	for _, s := range statusOrder {
		line := fmt.Sprintf("  %-15s %5d", s, cur.Totals[s])
		if prev != nil {
			if d := cur.Totals[s] - prev.Totals[s]; d != 0 {
				line += fmt.Sprintf("  (%+d)", d)
			}
		}
		fmt.Fprintln(w, line)
	}
	replayed := 0
	for _, r := range cur.Rows {
		if r.Replayed {
			replayed++
		}
	}
	fmt.Fprintf(w, "  counterexamples confirmed by witness replay: %d of %d\n", replayed, cur.Totals["counterexample"])
	if len(cur.Failures) > 0 {
		fmt.Fprintf(w, "  files the instrument could not measure: %d\n", len(cur.Failures))
	}

	section(w, "Skip reasons (normalised)", top(reasonBuckets(cur.Rows, "skipped"), n))
	section(w, "Inconclusive reasons (normalised)", top(reasonBuckets(cur.Rows, "inconclusive", "unknown"), n))
	section(w, "Error reasons (normalised)", top(reasonBuckets(cur.Rows, "error"), n))
	blockerSection(w, cur.Rows, n)
	a := attribute(cur.Rows)
	section(w, "Skips blamed on a callee signature: callee", top(a.Callees, n))
	section(w, "Skips blamed on a callee signature: type", top(a.Types, n))
	section(w, "Unsupported applications", top(a.Builtins, n))

	if cc := caveatCounts(cur.Rows); len(cc) > 0 {
		section(w, "Verified results with caveats (kind)", top(cc, n))
	}
	if uc := uncheckedCounts(cur.Rows); len(uc) > 0 {
		section(w, "Verified results with obligations checked only to an unrolling depth (kind)", top(uc, n))
	}

	fmt.Fprintf(w, "\nCall-site requires (D1): %d results reach a callee with a requires\n", cur.D1.Results)
	fmt.Fprintf(w, "  %-15s %5s %10s\n", "", "all", "unchecked")
	for _, s := range statusOrder {
		if c := cur.D1.ByStatus[s]; c > 0 {
			fmt.Fprintf(w, "  %-15s %5d %10d\n", s, c, cur.D1.Unchecked[s])
		}
	}
	fmt.Fprintln(w, "  (unchecked: reached only through a helper body, HOF lambda, recursion or contract)")

	if len(cur.Failures) > 0 {
		fmt.Fprintln(w, "\nNot measured")
		for i, f := range cur.Failures {
			if i == n {
				fmt.Fprintf(w, "  … %d more\n", len(cur.Failures)-n)
				break
			}
			fmt.Fprintf(w, "  %s %s: %s\n", f.Corpus, f.File, f.Detail)
		}
	}

	if prev == nil {
		fmt.Fprintln(w, "\nNo previous snapshot: this run is the baseline.")
		return 0
	}
	trans, added, removed := diff(prev, cur)
	fmt.Fprintf(w, "\nAgainst %s (%s)\n", prevPath, prev.Ailang)
	fmt.Fprintf(w, "  results added %d, removed %d, changed %d\n", added, removed, len(trans))
	counts := map[string]int{}
	regressions := 0
	for _, t := range trans {
		label := t.From + " → " + t.To
		if t.SourceChanged {
			label += " (source changed)"
		}
		counts[label]++
		if t.regression() {
			regressions++
		}
	}
	for _, e := range top(counts, len(counts)) {
		fmt.Fprintf(w, "  %-45s %4d\n", e.Key, e.Count)
	}
	if regressions > 0 {
		fmt.Fprintf(w, "\nREGRESSIONS in unchanged source: %d\n", regressions)
		for _, t := range trans {
			if t.regression() {
				fmt.Fprintf(w, "  %s  %s → %s\n", strings.ReplaceAll(t.Key, "|", " "), t.From, t.To)
			}
		}
	}
	return regressions
}

func section(w io.Writer, title string, rows []kv) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%s\n", title)
	for _, r := range rows {
		fmt.Fprintf(w, "  %5d  %s\n", r.Count, r.Key)
	}
}

// blockerSection ranks what skips are blamed on once each reason's chain of
// encoder layers is stripped (blame.go): any = skips it appears in, sole =
// skips it alone blocks.
func blockerSection(w io.Writer, rows []row, n int) {
	anyOf, sole := blockerCounts(rows)
	ranked := top(anyOf, n)
	if len(ranked) == 0 {
		return
	}
	fmt.Fprintln(w, "\nSkips by root blocker (any = skips it appears in, sole = skips it alone blocks)")
	fmt.Fprintf(w, "  %5s %5s\n", "any", "sole")
	for _, r := range ranked {
		fmt.Fprintf(w, "  %5d %5d  %s\n", r.Count, sole[r.Key], r.Key)
	}
}
