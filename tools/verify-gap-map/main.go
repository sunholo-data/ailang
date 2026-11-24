// Command verify-gap-map measures what `ailang verify` proves, skips and
// cannot decide across the downstream corpora, banks the result and diffs it
// against the previous snapshot (M-VERIFY-SOUNDNESS-COVERAGE M0). Every later
// milestone of the program is accepted against it.
//
// Corpora come from tools/downstream/corpora.yaml (the same inventory as
// `make downstream-check`): `self` plus each corpus's local checkout, read as
// it is. Snapshots are banked OUTSIDE the repository by default
// (~/.ailang/state/verify-gaps): several corpora are private, and a snapshot
// names their functions.
//
// Exit codes: 0 measured; 2 a regression against the previous snapshot in
// unchanged source (a proof lost, or a new error/unknown); 3 nothing could be
// measured; 64 usage.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() { os.Exit(run()) }

func run() int {
	var (
		bin      = flag.String("ailang", "ailang", "ailang binary to measure")
		manifest = flag.String("corpora", "tools/downstream/corpora.yaml", "corpus inventory")
		only     = flag.String("only", "", "comma-separated corpus names (default: all present)")
		add      = flag.String("add", "", "extra corpora, comma-separated name=dir")
		state    = flag.String("state", "", "snapshot directory (default ~/.ailang/state/verify-gaps)")
		compare  = flag.String("compare", "", "snapshot to diff against (default: the latest banked)")
		noBank   = flag.Bool("no-bank", false, "measure and diff, but do not bank")
		jobs     = flag.Int("jobs", 4, "files verified in parallel")
		timeout  = flag.Duration("timeout", 10*time.Minute, "per-file verify timeout")
		topN     = flag.Int("top", 12, "rows per ranked table")
	)
	flag.Parse()
	if *jobs < 1 || flag.NArg() > 0 {
		flag.Usage()
		return 64
	}
	repoRoot, _ := os.Getwd()
	corpora, err := loadCorpora(*manifest, repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify-gap-map:", err)
		return 64
	}
	for _, spec := range splitList(*add) {
		name, dir, ok := strings.Cut(spec, "=")
		if !ok {
			fmt.Fprintf(os.Stderr, "verify-gap-map: --add %q: want name=dir\n", spec)
			return 64
		}
		corpora = append(corpora, gapCorpus{Name: name, Root: expandHome(dir)})
	}
	if names := splitList(*only); len(names) > 0 {
		if corpora, err = selectCorpora(corpora, names); err != nil {
			fmt.Fprintln(os.Stderr, "verify-gap-map:", err)
			return 64
		}
	}
	version, err := exec.Command(*bin, "--version").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify-gap-map: %s --version: %v\n", *bin, err)
		return 3
	}

	snap := &snapshot{Taken: time.Now().UTC().Format(time.RFC3339), Ailang: firstLine(string(version)), Files: map[string]string{}}
	var work []job
	for _, c := range corpora {
		info := corpusInfo{Name: c.Name, Root: c.Root}
		if c.Root == "" {
			info.Note = "no local checkout"
			snap.Corpora = append(snap.Corpora, info)
			continue
		}
		info.Head, info.Dirty = gitHead(c.Root)
		files, err := contractFiles(c)
		if err != nil {
			info.Note = err.Error()
		}
		info.Files = len(files)
		for _, f := range files {
			work = append(work, job{corpus: c, file: f})
			snap.Files[c.Name+"|"+f] = fileHash(filepath.Join(c.Root, f))
		}
		snap.Corpora = append(snap.Corpora, info)
	}
	if len(work) == 0 {
		fmt.Fprintln(os.Stderr, "verify-gap-map: no contracted files in any present corpus")
		return 3
	}
	fmt.Fprintf(os.Stderr, "verifying %d files with %s (%d jobs)\n", len(work), snap.Ailang, *jobs)
	snap.Rows, snap.Failures = runAll(*bin, work, *jobs, *timeout, func(done, total int) {
		if done%25 == 0 || done == total {
			fmt.Fprintf(os.Stderr, "  %d/%d\n", done, total)
		}
	})
	sort.Slice(snap.Rows, func(i, j int) bool { return snap.Rows[i].key() < snap.Rows[j].key() })
	sort.Slice(snap.Failures, func(i, j int) bool {
		return snap.Failures[i].Corpus+snap.Failures[i].File < snap.Failures[j].Corpus+snap.Failures[j].File
	})
	for i := range snap.Corpora {
		for _, r := range snap.Rows {
			if r.Corpus == snap.Corpora[i].Name {
				snap.Corpora[i].Rows++
			}
		}
	}
	snap.Totals = totals(snap.Rows)
	snap.D1 = d1(snap.Rows)

	dir := *state
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".ailang", "state", "verify-gaps")
	}
	var prev *snapshot
	var prevPath string
	if *compare != "" {
		prev, prevPath, err = readSnapshot(*compare)
	} else {
		prev, prevPath, err = latestSnapshot(dir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify-gap-map: previous snapshot:", err)
		return 3
	}

	regressions := report(os.Stdout, snap, prev, prevPath, *topN)

	if !*noBank {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "verify-gap-map:", err)
			return 3
		}
		path := filepath.Join(dir, time.Now().UTC().Format("2006-01-02T150405Z")+".json")
		raw, _ := json.MarshalIndent(snap, "", " ")
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "verify-gap-map:", err)
			return 3
		}
		fmt.Printf("\nbanked %s\n", path)
	}
	if regressions > 0 {
		return 2
	}
	return 0
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func selectCorpora(cs []gapCorpus, names []string) ([]gapCorpus, error) {
	byName := map[string]gapCorpus{}
	for _, c := range cs {
		byName[c.Name] = c
	}
	out := make([]gapCorpus, 0, len(names))
	for _, n := range names {
		c, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("--only: no corpus named %q", n)
		}
		out = append(out, c)
	}
	return out, nil
}
