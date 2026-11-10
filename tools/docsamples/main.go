// Command docsamples is the doc-sample gate (make check-doc-samples).
//
// It extracts every fenced AILANG block from the published docs (docs/docs/**),
// README.md, docs/LIMITATIONS.md, the active teaching prompt's source file and AILANG
// embedded in docs/src, plus every .ail file the docs import with raw-loader, and checks
// each with the ailang binary it is given: `ailang check` for complete programs and
// imported files, `ailang fmt` for parse-only fragments, and an expected first error for
// anti-examples tagged `error=CODE`. Failures are compared with a shrink-only baseline
// keyed by page and content hash; sketch/repl blocks with a second shrink-only list;
// structural defects (DOC_*) cannot be listed. The grammar is documented in
// .claude/rules/docs-review.md; the design is M-V1-DOCS-FOR-1-0 §A.
//
// Exit codes: 0 gate green, 1 gate failure, 2 instrument failure (no binary, no
// blocks found, unreadable list, unresolvable base ref, a check that could not run).
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
	"sort"
	"strconv"
	"strings"
	"time"
)

type options struct {
	root, bin, baseline, unchecked, baseRef, keep, only, jsonOut, stdlib string
	jobs                                                                 int
	write, github, verbose                                               bool
}

func main() {
	var o options
	flag.StringVar(&o.root, "root", ".", "checkout to scan")
	flag.StringVar(&o.bin, "ailang", "", "ailang binary built from THIS commit (required)")
	flag.StringVar(&o.baseline, "baseline", "scripts/doc_samples_baseline.txt", "baseline of known-failing blocks, relative to -root")
	flag.StringVar(&o.unchecked, "unchecked", "scripts/doc_samples_unchecked.txt", "list of sketch/repl blocks, relative to -root")
	flag.StringVar(&o.baseRef, "base-ref", "", "git commit whose lists this one's may not grow beyond, as sets (CI: the PR base, or the push's previous tip)")
	flag.StringVar(&o.stdlib, "stdlib", "", "stdlib directory (default <root>/std when present)")
	flag.StringVar(&o.keep, "keep", "", "stage blocks under this directory and keep it (to reproduce a failure)")
	flag.StringVar(&o.only, "only", "", "check only blocks whose <page>:<line> or <page> matches")
	flag.StringVar(&o.jsonOut, "json", "", "write every block's verdict as JSON to this file")
	flag.IntVar(&o.jobs, "jobs", min(runtime.NumCPU(), 8), "parallel checks")
	flag.BoolVar(&o.write, "write-baseline", false, "rewrite both lists from the current state (seed / burn-down)")
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
	if o.unchecked == "" {
		o.unchecked = "scripts/doc_samples_unchecked.txt"
	}
	root, _ := filepath.Abs(o.root)
	blocks, sources, err := collect(root, o.only)
	if err != nil {
		fmt.Fprintf(w, "doc-samples: %v\n", err)
		return 2
	}
	if len(blocks) == 0 {
		fmt.Fprintf(w, "doc-samples: no AILANG blocks found under %s (filter %q) — refusing a vacuous pass\n", root, o.only)
		return 2
	}
	blPath, unPath := filepath.Join(root, o.baseline), filepath.Join(root, o.unchecked)
	bl, err := LoadList(blPath, o.write)
	if err == nil {
		var un *List
		if un, err = LoadList(unPath, o.write); err == nil {
			return runChecks(o, w, start, root, bin, blocks, sources, bl, un, blPath, unPath)
		}
	}
	fmt.Fprintf(w, "doc-samples: %v\n", err)
	return 2
}

func runChecks(o options, w io.Writer, start time.Time, root, bin string, blocks []*Block, sources map[string]bool, bl, un *List, blPath, unPath string) int {
	work := o.keep
	var err error
	if work == "" {
		if work, err = os.MkdirTemp("", "docsamples-"); err != nil {
			fmt.Fprintf(w, "doc-samples: %v\n", err)
			return 2
		}
		defer os.RemoveAll(work)
	}
	stdlib, err := resolveStdlib(root, o.stdlib)
	if err != nil {
		fmt.Fprintf(w, "doc-samples: -stdlib: %v\n", err)
		return 2
	}
	fmt.Fprintf(w, "doc-samples: %s\n", binVersion(bin))
	outcomes, err := CheckAll(context.Background(), NewBinChecker(bin, work, stdlib), blocks, work, root, max(o.jobs, 1))
	if err != nil {
		fmt.Fprintf(w, "doc-samples: instrument failure: %v\n", err)
		return 2
	}
	judged := Judge(blocks, outcomes, bl, un)
	if o.only != "" {
		judged = dropOrphans(judged) // a filtered run cannot see the rest of the lists
	}
	if o.write {
		if o.only != "" {
			fmt.Fprintln(w, "doc-samples: -write-baseline with -only would drop every other page's entries; refusing")
			return 2
		}
		nb, nu, err := WriteLists(blPath, unPath, judged, un)
		if err != nil {
			fmt.Fprintf(w, "doc-samples: %v\n", err)
			return 2
		}
		fmt.Fprintf(w, "doc-samples: wrote %d entries to %s, %d to %s\n", nb, o.baseline, nu, o.unchecked)
		bl, _ = LoadList(blPath, false)
		un, _ = LoadList(unPath, false)
		judged = Judge(blocks, outcomes, bl, un)
	}
	if o.jsonOut != "" {
		if err := writeJSON(o.jsonOut, judged); err != nil {
			fmt.Fprintf(w, "doc-samples: %v\n", err)
			return 2
		}
	}
	failed := report(w, judged, o, work)
	if o.only == "" {
		staleFail, staleNote := staleExemptions(blocks, outcomes, sources)
		for _, k := range staleFail {
			fmt.Fprintf(w, "FAIL      structural exemption %q matches no defect any more — delete it from tools/docsamples/structural.go\n", k)
			failed = true
		}
		for _, k := range staleNote {
			fmt.Fprintf(w, "note      structural exemption %q: its file is no longer in scope — delete it from tools/docsamples/structural.go\n", k)
		}
	}
	if o.baseRef != "" && o.only == "" {
		rc := ratchet(w, root, o, bl, un)
		if rc == 2 {
			return 2
		}
		failed = failed || rc == 1
	}
	fmt.Fprintf(w, "doc-samples: %.1fs\n", time.Since(start).Seconds())
	if failed {
		return 1
	}
	return 0
}

// ratchet enforces "the lists only shrink" against o.baseRef, as SETS: a line whose
// page+hash the base does not have is growth, even when another line left (a swap keeps
// the count). New unchecked lines pass only with a `# reviewed:` marker.
func ratchet(w io.Writer, root string, o options, bl, un *List) int {
	rc := 0
	for _, f := range []struct {
		path  string
		cur   *List
		allow func(Entry) bool
		hint  string
	}{
		{o.baseline, bl, nil, "fix or tag the block instead of listing it"},
		{o.unchecked, un, reviewed, "a new sketch/repl line needs `# reviewed: <why it cannot be checked>`"},
	} {
		base, found, err := ListAtRef(root, o.baseRef, f.path)
		if err != nil {
			fmt.Fprintf(w, "doc-samples: shrink-only check: %v\n", err)
			return 2
		}
		if !found {
			fmt.Fprintf(w, "shrink-only: %s has no %s yet; %d entries seeded here\n", o.baseRef, f.path, len(f.cur.Entries))
			continue
		}
		added := Added(f.cur, base, f.allow)
		for _, id := range added {
			fmt.Fprintf(w, "FAIL      shrink-only: %s gained %q, which %s does not have — %s\n", f.path, id, o.baseRef, f.hint)
			if o.github {
				fmt.Fprintf(w, "::error file=%s::doc-samples: new entry %s (%s)\n", f.path, id, f.hint)
			}
		}
		if len(added) > 0 {
			rc = 1
		}
		fmt.Fprintf(w, "shrink-only: %s %d entries (%s: %d), %d new\n", f.path, len(f.cur.Entries), o.baseRef, len(base.Entries), len(added))
	}
	return rc
}

func resolveStdlib(root, stdlib string) (string, error) {
	if stdlib == "" {
		if st, e := os.Stat(filepath.Join(root, "std", "io.ail")); e == nil && !st.IsDir() {
			return filepath.Join(root, "std"), nil
		}
		return "", nil
	}
	// Absolute: every check runs in its own staged directory.
	abs, err := filepath.Abs(stdlib)
	if err == nil {
		_, err = os.Stat(filepath.Join(abs, "io.ail"))
	}
	return abs, err
}

// collect extracts the blocks in scope, optionally filtered by -only, and returns the
// set of source files read (for exemption bookkeeping).
func collect(root, only string) ([]*Block, map[string]bool, error) {
	s, err := Scope(root)
	if err != nil {
		return nil, nil, err
	}
	sources := map[string]bool{}
	var blocks []*Block
	importers := map[string][]string{}
	read := func(f string) (string, error) {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		sources[f] = true
		return strings.ReplaceAll(string(raw), "\r\n", "\n"), err
	}
	addImports := func(page, text string) {
		imps, defects := ImportsOf(root, page, text)
		blocks = append(blocks, defects...)
		for _, im := range imps {
			importers[im.Path] = append(importers[im.Path], fmt.Sprintf("%s:%d", im.From, im.Line))
		}
	}
	for _, p := range s.Pages {
		text, err := read(p)
		if err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, ExtractBlocks(p, p, text)...)
		addImports(p, text)
	}
	if s.Prompt != "" {
		text, err := read(s.Prompt)
		if err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, ExtractBlocks(PromptPage, s.Prompt, text)...)
	}
	for _, f := range s.JS {
		text, err := read(f)
		if err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, LiteralsOf(f, text)...)
		addImports(f, text)
	}
	files := make([]string, 0, len(importers))
	for f := range importers {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		text, err := read(f)
		if err != nil {
			return nil, nil, err
		}
		blocks = append(blocks, &Block{Page: f, Source: f, Line: 1, Lang: "file", Info: "imported by " + strings.Join(importers[f], ", "),
			Class: ClassFile, Body: text, Key: KeyOf(text)})
	}
	var kept []*Block
	for _, b := range blocks {
		if only == "" || only == b.Page || only == b.Source || only == b.Source+":"+strconv.Itoa(b.Line) {
			kept = append(kept, b)
		}
	}
	sortBlocks(kept)
	return kept, sources, nil
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
	Synth    bool   `json:"synthesised_module,omitempty"`
	FirstErr string `json:"first_error,omitempty"`
	Why      string `json:"why,omitempty"`
}

func writeJSON(path string, js []Judged) error {
	rows := make([]jsonRow, 0, len(js))
	for _, j := range js {
		r := jsonRow{Verdict: string(j.Verdict), Why: j.Why}
		if j.Block != nil {
			b := j.Block
			r.Page, r.Source, r.Line, r.Lang, r.Class, r.Key = b.Page, b.Source, b.Line, b.Lang, string(b.Class), b.Key
			r.Code, r.FirstErr, r.Synth = j.Outcome.Code, j.Outcome.FirstErr, j.Outcome.Synth
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
