// Command docsamples is the doc-sample gate (make check-doc-samples).
//
// It extracts every fenced AILANG block from the published docs (docs/docs/**),
// README.md, docs/LIMITATIONS.md and the active teaching prompt's source file, and
// checks each one with the ailang binary it is given: `ailang check` for complete
// programs, `ailang fmt` for parse-only fragments, and an expected failure for
// anti-examples tagged `error=CODE`. Failures are compared with a shrink-only
// baseline keyed by page and content hash. The info-string grammar is documented
// in .claude/rules/docs-review.md; the design is M-V1-DOCS-FOR-1-0 §A.
//
// Exit codes: 0 gate green, 1 gate failure, 2 instrument failure (no binary, no
// blocks found, unreadable baseline, a check that could not run).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type options struct {
	root, bin, baseline, baseRef, keep, only, jsonOut, stdlib string
	jobs                                                      int
	write, github, verbose                                    bool
}

func main() {
	var o options
	flag.StringVar(&o.root, "root", ".", "checkout to scan")
	flag.StringVar(&o.bin, "ailang", "", "ailang binary built from THIS commit (required)")
	flag.StringVar(&o.baseline, "baseline", "scripts/doc_samples_baseline.txt", "baseline file, relative to -root")
	flag.StringVar(&o.baseRef, "base-ref", "", "git ref whose baseline entry count this one may not exceed (CI: origin/dev)")
	flag.StringVar(&o.stdlib, "stdlib", "", "stdlib directory (default <root>/std when present)")
	flag.StringVar(&o.keep, "keep", "", "stage blocks under this directory and keep it (to reproduce a failure)")
	flag.StringVar(&o.only, "only", "", "check only blocks whose <page>:<line> or <page> matches")
	flag.StringVar(&o.jsonOut, "json", "", "write every block's verdict as JSON to this file")
	flag.IntVar(&o.jobs, "jobs", min(runtime.NumCPU(), 8), "parallel checks")
	flag.BoolVar(&o.write, "write-baseline", false, "rewrite the baseline from the current failures (seed / burn-down)")
	flag.BoolVar(&o.github, "github", false, "emit ::error annotations")
	flag.BoolVar(&o.verbose, "v", false, "list every block, not only failures")
	flag.Parse()
	os.Exit(run(o, os.Stdout))
}

func run(o options, w io.Writer) int {
	start := time.Now()
	if o.bin == "" {
		fmt.Fprintln(w, "doc-samples: -ailang <binary> is required (make check-doc-samples builds one from this commit)")
		return 2
	}
	bin, err := filepath.Abs(o.bin)
	if err == nil {
		_, err = os.Stat(bin)
	}
	if err != nil {
		fmt.Fprintf(w, "doc-samples: ailang binary: %v\n", err)
		return 2
	}
	root, _ := filepath.Abs(o.root)
	blocks, err := collect(root, o.only)
	if err != nil {
		fmt.Fprintf(w, "doc-samples: %v\n", err)
		return 2
	}
	if len(blocks) == 0 {
		fmt.Fprintf(w, "doc-samples: no AILANG blocks found under %s (filter %q) — refusing a vacuous pass\n", root, o.only)
		return 2
	}
	blPath := filepath.Join(root, o.baseline)
	bl, err := LoadBaseline(blPath, o.write)
	if err != nil {
		fmt.Fprintf(w, "doc-samples: baseline: %v\n", err)
		return 2
	}
	work := o.keep
	if work == "" {
		if work, err = os.MkdirTemp("", "docsamples-"); err != nil {
			fmt.Fprintf(w, "doc-samples: %v\n", err)
			return 2
		}
		defer os.RemoveAll(work)
	}
	stdlib := o.stdlib
	if stdlib == "" {
		if st, e := os.Stat(filepath.Join(root, "std", "io.ail")); e == nil && !st.IsDir() {
			stdlib = filepath.Join(root, "std")
		}
	} else {
		// Absolute: every check runs in its own staged directory.
		if stdlib, err = filepath.Abs(stdlib); err == nil {
			_, err = os.Stat(filepath.Join(stdlib, "io.ail"))
		}
		if err != nil {
			fmt.Fprintf(w, "doc-samples: -stdlib: %v\n", err)
			return 2
		}
	}
	fmt.Fprintf(w, "doc-samples: %s\n", binVersion(bin))
	outcomes, err := CheckAll(context.Background(), NewBinChecker(bin, work, stdlib), blocks, work, max(o.jobs, 1))
	if err != nil {
		fmt.Fprintf(w, "doc-samples: instrument failure: %v\n", err)
		return 2
	}
	judged := Judge(blocks, outcomes, bl)
	if o.only != "" {
		judged = dropOrphans(judged) // a filtered run cannot see the rest of the baseline
	}
	if o.jsonOut != "" {
		if err := writeJSON(o.jsonOut, judged); err != nil {
			fmt.Fprintf(w, "doc-samples: %v\n", err)
			return 2
		}
	}
	if o.write {
		if o.only != "" {
			fmt.Fprintln(w, "doc-samples: -write-baseline with -only would drop every other page's entries; refusing")
			return 2
		}
		n, err := WriteBaseline(blPath, judged)
		if err != nil {
			fmt.Fprintf(w, "doc-samples: %v\n", err)
			return 2
		}
		fmt.Fprintf(w, "doc-samples: wrote %d entries to %s\n", n, o.baseline)
		bl, _ = LoadBaseline(blPath, false)
		judged = Judge(blocks, outcomes, bl)
	}
	failed := report(w, judged, o, work)
	if o.baseRef != "" && o.only == "" {
		n, found, err := CountAtRef(root, o.baseRef, o.baseline)
		switch {
		case err != nil:
			fmt.Fprintf(w, "doc-samples: shrink-only check: %v\n", err)
			return 2
		case !found:
			fmt.Fprintf(w, "shrink-only: %s has no baseline at %s yet; %d entries seeded here\n", o.baseRef, o.baseline, len(bl.Entries))
		case len(bl.Entries) > n:
			fmt.Fprintf(w, "FAIL shrink-only: %s has %d entries, %s has %d — the baseline may only shrink.\n"+
				"     Fix or tag the new failing blocks instead of listing them.\n", o.baseline, len(bl.Entries), o.baseRef, n)
			failed = true
		default:
			fmt.Fprintf(w, "shrink-only: %d entries (%s: %d)\n", len(bl.Entries), o.baseRef, n)
		}
	}
	fmt.Fprintf(w, "doc-samples: %.1fs\n", time.Since(start).Seconds())
	if failed {
		return 1
	}
	return 0
}

// collect extracts the blocks in scope, optionally filtered by -only.
func collect(root, only string) ([]*Block, error) {
	pages, prompt, err := Scope(root)
	if err != nil {
		return nil, err
	}
	type src struct{ page, file string }
	var srcs []src
	for _, p := range pages {
		srcs = append(srcs, src{p, p})
	}
	if prompt != "" {
		srcs = append(srcs, src{PromptPage, prompt})
	}
	var blocks []*Block
	for _, s := range srcs {
		raw, err := os.ReadFile(filepath.Join(root, s.file))
		if err != nil {
			return nil, err
		}
		for _, b := range ExtractBlocks(s.page, s.file, string(raw)) {
			if only == "" || only == b.Page || only == b.Source || only == b.Source+":"+strconv.Itoa(b.Line) {
				blocks = append(blocks, b)
			}
		}
	}
	sortBlocks(blocks)
	return blocks, nil
}

func dropOrphans(js []Judged) []Judged {
	var out []Judged
	for _, j := range js {
		if j.Verdict != VOrphan {
			out = append(out, j)
		}
	}
	return out
}

func binVersion(bin string) string {
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return bin + " (version unavailable: " + err.Error() + ")"
	}
	return bin + " " + strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
}

type jsonRow struct {
	Page     string `json:"page"`
	Source   string `json:"source,omitempty"`
	Line     int    `json:"line,omitempty"`
	Lang     string `json:"lang,omitempty"`
	Class    string `json:"class,omitempty"`
	Key      string `json:"key"`
	Verdict  string `json:"verdict"`
	Code     string `json:"code,omitempty"`
	FirstErr string `json:"first_error,omitempty"`
}

func writeJSON(path string, js []Judged) error {
	rows := make([]jsonRow, 0, len(js))
	for _, j := range js {
		r := jsonRow{Verdict: string(j.Verdict)}
		if j.Block != nil {
			b := j.Block
			r.Page, r.Source, r.Line, r.Lang, r.Class, r.Key = b.Page, b.Source, b.Line, b.Lang, string(b.Class), b.Key
			r.Code, r.FirstErr = j.Outcome.Code, j.Outcome.FirstErr
		} else {
			r.Page, r.Key, r.Code = j.Entry.Page, j.Entry.Key, j.Entry.Code
		}
		rows = append(rows, r)
	}
	raw, err := json.MarshalIndent(rows, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
