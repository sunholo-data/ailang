package eval_analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// TestD2DiscordantCount is the MEASUREMENT behind ruling D2 (M-V1-SIMPLIFY-S3
// M1, Mark 2026-09-15): over every banked baseline dir, how often does the
// canonical pass predicate (CompileOk && RuntimeOk && StdoutOk — now
// RunMetrics.Passed) disagree with the StdoutOk-alone reads that ~86 sites
// used, and what does that do to each version's pass rate?
//
// It stands in for the design-quorum step the program doc asked for: the
// ruling was made attended with the evidence in front of Mark, and this is the
// number the quorum would have demanded. Nothing is re-banked — the table is
// recorded in the PR and in the OS-history boundary note.
//
// Skipped unless AILANG_EVAL_RESULTS_DIR points at a directory of per-version
// baseline dirs (eval_results/baselines). Run once with:
//
//	AILANG_EVAL_RESULTS_DIR=eval_results/baselines \
//	  go test ./internal/eval_analysis/ -run TestD2DiscordantCount -v
func TestD2DiscordantCount(t *testing.T) {
	root := os.Getenv("AILANG_EVAL_RESULTS_DIR")
	if root == "" {
		t.Skip("set AILANG_EVAL_RESULTS_DIR=eval_results/baselines to run the D2 measurement")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}

	type tally struct {
		rows, stdoutPass, conjPass, discordant int
		byMode                                 map[string]int
		invalid                                int
	}
	perVersion := map[string]*tally{}
	perBench := map[string]int{}
	perBenchRows := map[string]int{}
	total := &tally{byMode: map[string]int{}}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		version := e.Name()
		dir := filepath.Join(root, version)
		// Valid rows only, deduped per slot: exactly what a published rate is
		// computed from. Invalid rows are counted separately so the
		// denominator is visible.
		rows, stats, err := eval_harness.LoadRows([]string{dir}, eval_harness.LoadOptions{})
		if err != nil {
			t.Fatalf("%s: %v", version, err)
		}
		v := &tally{byMode: map[string]int{}, invalid: stats.Invalid}
		perVersion[version] = v
		for i := range rows {
			r := &rows[i]
			v.rows++
			total.rows++
			if r.StdoutOk {
				v.stdoutPass++
				total.stdoutPass++
			}
			if r.Passed() {
				v.conjPass++
				total.conjPass++
			}
			if r.StdoutOk != r.Passed() {
				v.discordant++
				total.discordant++
				mode := r.EvalMode
				if mode == "" {
					mode = "standard(legacy)"
				}
				v.byMode[mode]++
				total.byMode[mode]++
				perBench[r.ID]++
			}
			perBenchRows[r.ID]++
		}
		total.invalid += stats.Invalid
	}

	versions := make([]string, 0, len(perVersion))
	for v := range perVersion {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	var b strings.Builder
	pct := func(n, d int) float64 {
		if d == 0 {
			return 0
		}
		return 100 * float64(n) / float64(d)
	}
	fmt.Fprintf(&b, "\nD2 discordant rows: StdoutOk vs CompileOk&&RuntimeOk&&StdoutOk (valid, deduped rows)\n\n")
	fmt.Fprintf(&b, "| version | rows | invalid excl. | pass(stdout_ok) | pass(conj) | discordant | rate(stdout_ok) | rate(conj) | delta pp | discordant by mode |\n")
	fmt.Fprintf(&b, "|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, ver := range versions {
		v := perVersion[ver]
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d | %.2f%% | %.2f%% | %+.2f | %s |\n",
			ver, v.rows, v.invalid, v.stdoutPass, v.conjPass, v.discordant,
			pct(v.stdoutPass, v.rows), pct(v.conjPass, v.rows),
			pct(v.conjPass, v.rows)-pct(v.stdoutPass, v.rows), modeList(v.byMode))
	}
	fmt.Fprintf(&b, "| **all** | %d | %d | %d | %d | %d | %.2f%% | %.2f%% | %+.2f | %s |\n",
		total.rows, total.invalid, total.stdoutPass, total.conjPass, total.discordant,
		pct(total.stdoutPass, total.rows), pct(total.conjPass, total.rows),
		pct(total.conjPass, total.rows)-pct(total.stdoutPass, total.rows), modeList(total.byMode))

	benches := make([]string, 0, len(perBench))
	for id := range perBench {
		benches = append(benches, id)
	}
	sort.Slice(benches, func(i, j int) bool {
		if perBench[benches[i]] != perBench[benches[j]] {
			return perBench[benches[i]] > perBench[benches[j]]
		}
		return benches[i] < benches[j]
	})
	fmt.Fprintf(&b, "\nDiscordant rows by benchmark (all versions):\n\n| benchmark | discordant | rows | share |\n|---|---:|---:|---:|\n")
	for _, id := range benches {
		fmt.Fprintf(&b, "| %s | %d | %d | %.2f%% |\n", id, perBench[id], perBenchRows[id], pct(perBench[id], perBenchRows[id]))
	}
	t.Log(b.String())

	// Every discordant row must be a StdoutOk=true row the conjunction rejects:
	// the conjunction can never pass a row StdoutOk fails, so the delta is
	// one-directional (pass rates can only go DOWN under D2).
	if total.conjPass > total.stdoutPass {
		t.Fatalf("conjunction passed %d rows but stdout_ok only %d — predicate is not a strict subset", total.conjPass, total.stdoutPass)
	}
}

func modeList(m map[string]int) string {
	if len(m) == 0 {
		return "—"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}
