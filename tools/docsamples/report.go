package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// tag renders a block's info string the way an author writes it.
func tag(b *Block) string {
	lang := b.Lang
	if lang == "" {
		lang = "(no language)"
	}
	if b.Info == "" || b.Class == ClassFile {
		return lang
	}
	return lang + " " + b.Info
}

type pageStat struct {
	page                                     string
	blocks, pass, baselined, failing, sketch int
	repl, checked                            int
}

// report prints failures (always), every block (-v), counts, the worst pages and
// over-sketched pages. It returns whether the gate failed.
func report(w io.Writer, js []Judged, o options, work string) bool {
	c := map[string]int{}
	pages := map[string]*pageStat{}
	failed := false
	for _, j := range js {
		if j.Verdict.Failing() {
			failed = true
		}
		if j.Block == nil {
			c["orphan"]++
			printFailure(w, j, o, work)
			continue
		}
		b, oc := j.Block, j.Outcome
		ps := pages[b.Page]
		if ps == nil {
			ps = &pageStat{page: b.Page}
			pages[b.Page] = ps
		}
		ps.blocks++
		c["blocks"]++
		if b.Class == ClassFile {
			c["files"]++
		}
		switch b.Class {
		case ClassSketch:
			ps.sketch++
			c["sketch"]++
		case ClassRepl:
			ps.repl++
			c["repl"]++
		default:
			ps.checked++
		}
		switch {
		case j.Verdict == VOK && oc.Synth && b.Class == ClassProgram:
			c["pass_synth"]++
			ps.pass++
		case j.Verdict == VOK:
			c["pass_"+string(b.Class)]++
			ps.pass++
		case j.Verdict == VBaselined:
			ps.baselined++
			c["baselined"]++
		case j.Verdict == VExempt:
			c["exempt"]++
		case j.Verdict == VFail && isStructural(oc.Code):
			ps.failing++
			c["structural"]++
		case j.Verdict == VFail || j.Verdict == VStale:
			ps.failing++
			c[strings.ToLower(string(j.Verdict))]++
		}
		switch {
		case j.Verdict.Failing():
			printFailure(w, j, o, work)
		case j.Verdict == VExempt:
			fmt.Fprintf(w, "EXEMPT    %s:%d [%s] %s: %s\n          exemption: %s\n", b.Source, b.Line, tag(b), oc.Code, oc.FirstErr, j.Why)
		case o.verbose:
			fmt.Fprintf(w, "%-9s %s:%d [%s] %s\n", j.Verdict, b.Source, b.Line, tag(b), oc.Code)
		}
	}
	fmt.Fprintf(w, "\nblocks %d on %d pages/files (imported .ail files %d) | pass: program %d, only with a synthesised module line %d, "+
		"imported file %d, parse %d, error= %d | unchecked: sketch %d, repl %d | baselined failing %d | "+
		"FAIL: new %d, structural %d | stale %d, orphaned %d | exempt %d\n",
		c["blocks"], len(pages), c["files"], c["pass_program"], c["pass_synth"], c["pass_file"], c["pass_parse"], c["pass_error"],
		c["sketch"], c["repl"], c["baselined"], c["fail"], c["structural"], c["stale"], c["orphan"], c["exempt"])
	worstPages(w, pages)
	if failed {
		fmt.Fprintln(w, "\nFAIL: fix the block, or tag it (```ailang parse | sketch | repl | error=CODE | file=path.ail); "+
			"for a block that passes now, delete its list line. Grammar: .claude/rules/docs-review.md")
	} else {
		fmt.Fprintln(w, "\nOK: no new or changed failure, no structural defect, no stale list entry")
	}
	return failed
}

func printFailure(w io.Writer, j Judged, o options, work string) {
	if j.Block == nil {
		e := j.Entry
		file := o.baseline
		if j.List == "unchecked" {
			file = o.unchecked
		}
		fmt.Fprintf(w, "ORPHAN    %s %s (%s): listed block no longer exists — delete this line from %s\n", e.Page, e.Key, e.Code, file)
		if o.github {
			fmt.Fprintf(w, "::error file=%s::doc-samples: orphaned entry %s %s — delete it\n", file, e.Page, e.Key)
		}
		return
	}
	b, oc := j.Block, j.Outcome
	msg := oc.FirstErr
	if j.Why != "" {
		msg = j.Why
		if oc.FirstErr != "" && j.Verdict == VFail {
			msg += " [" + oc.FirstErr + "]"
		}
	}
	fmt.Fprintf(w, "%-9s %s:%d [%s] %s: %s\n", j.Verdict, b.Source, b.Line, tag(b), firstNonEmpty(oc.Code, "-"), msg)
	if oc.Cmd != "" && j.Verdict == VFail {
		only := fmt.Sprintf("%s:%d", b.Source, b.Line)
		switch {
		case b.Class == ClassFile:
			fmt.Fprintf(w, "          repro: %s   (from the repository root)\n", oc.Cmd)
		case o.keep != "":
			fmt.Fprintf(w, "          repro: cd %s && %s\n", filepath.Join(work, filepath.Base(filepath.Dir(oc.Dir)), filepath.Base(oc.Dir)), oc.Cmd)
		default:
			fmt.Fprintf(w, "          repro: make check-doc-samples DOC_SAMPLES_ARGS='-only %s -keep /tmp/docsamples'  (then: %s)\n", only, oc.Cmd)
		}
	}
	if o.github {
		fmt.Fprintf(w, "::error file=%s,line=%d::doc-samples %s [%s] %s: %s\n", b.Source, b.Line, j.Verdict, tag(b), oc.Code, msg)
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// worstPages prints the 15 pages with the most failures, and any page with more sketch
// blocks than checked ones (the "tagging exercise" risk).
func worstPages(w io.Writer, pages map[string]*pageStat) {
	var ps []*pageStat
	for _, p := range pages {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].baselined+ps[i].failing != ps[j].baselined+ps[j].failing {
			return ps[i].baselined+ps[i].failing > ps[j].baselined+ps[j].failing
		}
		return ps[i].page < ps[j].page
	})
	fmt.Fprintf(w, "\nworst pages (blocks / pass / baselined+failing / sketch / repl):\n")
	for i, p := range ps {
		if i == 15 || p.baselined+p.failing == 0 {
			break
		}
		fmt.Fprintf(w, "  %-58s %3d %3d %3d %3d %3d\n", p.page, p.blocks, p.pass, p.baselined+p.failing, p.sketch, p.repl)
	}
	var over []string
	for _, p := range ps {
		if p.sketch > p.checked {
			over = append(over, fmt.Sprintf("%s (%d sketch, %d checked)", p.page, p.sketch, p.checked))
		}
	}
	sort.Strings(over)
	if len(over) > 0 {
		fmt.Fprintf(w, "pages with more sketch than checked blocks (review, not a failure): %s\n", strings.Join(over, "; "))
	}
}
