// chain-export writes eval-suite results recorded in observatory chains back
// out as standard result files, so the directory-based pipeline (eval-report,
// eval-elo, publish-unified-dashboard.sh) can read them.
//
// Cloud placements run one model at a time (model-manager §5.5) and are
// recorded in chains; the raw files often lived in /tmp and are gone. This is
// the backfill for the cloud-rolling bank the dashboard publishes from between
// releases.
//
// Usage:
//
//	go run ./tools/chain-export --since 2026-08-27 --out eval_results/rotation/cloud-rolling \
//	    [--exclude-model-substr qwen,motoko-local] [--min-rows 15] [--dry-run]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func main() {
	since := flag.String("since", "", "export chains created after this date (YYYY-MM-DD), required")
	out := flag.String("out", "", "output directory, required")
	exclude := flag.String("exclude-model-substr", "", "comma-separated substrings; models containing any are skipped")
	minRows := flag.Int("min-rows", 15, "skip (mode, model) pairs with fewer rows — stray probes, not measurements")
	dryRun := flag.Bool("dry-run", false, "print per-model counts without writing files")
	flag.Parse()

	if *since == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: chain-export --since YYYY-MM-DD --out DIR [--exclude-model-substr a,b] [--dry-run]")
		os.Exit(2)
	}
	after, err := time.Parse("2006-01-02", *since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bad --since: %v\n", err)
		os.Exit(2)
	}
	var skip []string
	for _, s := range strings.Split(*exclude, ",") {
		if s = strings.TrimSpace(s); s != "" {
			skip = append(skip, s)
		}
	}

	results, err := eval_analysis.LoadResultsFromEvalChainsSince(after)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}

	if !*dryRun {
		if err := os.MkdirAll(*out, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
			os.Exit(1)
		}
	}

	// api_error rows are not measurements (eval_harness.applyValidityBackstop):
	// the harness failed, not the model. Live eval-suite rows are marked invalid
	// when logged, but chain stages carry no validity field, so a dead run (a
	// 429 storm, a crashed executor) would otherwise publish as a 0% model.
	measured := results[:0]
	invalid := 0
	for _, r := range results {
		if r.ErrorCategory == eval_harness.ErrorCategoryAPI {
			invalid++
			continue
		}
		measured = append(measured, r)
	}
	results = measured

	type key struct{ mode, model string }
	counts := map[key]int{}
	for _, r := range results {
		m := r.RunMetrics
		if m.Model != "" && m.Lang != "" && !skipped(m.Model, skip) {
			counts[key{m.EvalMode, m.Model}]++
		}
	}
	for k, n := range counts {
		if n < *minRows {
			delete(counts, k)
		}
	}

	// Chain stages do not record the ailang version. Stamp each row with the
	// release that was current when it ran (the newest v* tag created at or
	// before its timestamp) — the binary that ran it was that release or a dev
	// build on top of it, which ReleaseTag buckets the same way.
	releases, err := releaseDates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: no release tags (%v) — rows exported without ailang_version\n", err)
	}

	written := 0
	for i, r := range results {
		m := r.RunMetrics
		if _, ok := counts[key{m.EvalMode, m.Model}]; !ok || skipped(m.Model, skip) {
			continue
		}
		if *dryRun {
			continue
		}
		if m.AilangVersion == "" {
			m.AilangVersion = releaseAt(releases, m.Timestamp)
		}
		data, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal %s/%s: %v\n", m.Model, m.ID, err)
			os.Exit(1)
		}
		// Same shape as eval-suite's own files; the index keeps two stages that
		// completed in the same second from overwriting each other.
		name := fmt.Sprintf("%s_%s_%s_%d%03d.json", m.ID, m.Lang, m.Model, m.Timestamp.Unix(), i%1000)
		if err := os.WriteFile(filepath.Join(*out, name), data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
		written++
	}

	keys := make([]key, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].mode != keys[j].mode {
			return keys[i].mode < keys[j].mode
		}
		return keys[i].model < keys[j].model
	})
	for _, k := range keys {
		fmt.Printf("%-9s %-40s %4d\n", k.mode, k.model, counts[k])
	}
	fmt.Printf("%d measured rows from chains after %s (%d api_error rows skipped); %d files written to %s\n",
		len(results), *since, invalid, written, *out)
}

func skipped(model string, skip []string) bool {
	for _, s := range skip {
		if strings.Contains(model, s) {
			return true
		}
	}
	return false
}

type release struct {
	tag string
	at  time.Time
}

var releaseTagRE = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// releaseDates lists the repo's release tags (vX.Y.Z only) oldest first.
func releaseDates() ([]release, error) {
	out, err := exec.Command("git", "for-each-ref", "--sort=creatordate",
		"--format=%(creatordate:unix) %(refname:short)", "refs/tags/v*").Output()
	if err != nil {
		return nil, err
	}
	var rs []release
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 || !releaseTagRE.MatchString(f[1]) {
			continue
		}
		sec, err := strconv.ParseInt(f[0], 10, 64)
		if err != nil {
			continue
		}
		rs = append(rs, release{f[1], time.Unix(sec, 0)})
	}
	return rs, nil
}

// releaseAt returns the newest release created at or before t ("" if none).
func releaseAt(rs []release, t time.Time) string {
	tag := ""
	for _, r := range rs {
		if r.at.After(t) {
			break
		}
		tag = r.tag
	}
	return tag
}
