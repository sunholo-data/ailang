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
	if b.Info == "" {
		return lang
	}
	return lang + " " + b.Info
}

type pageStat struct {
	page                                     string
	blocks, pass, baselined, failing, sketch int
	repl, checked, misfenced                 int
}

// report prints failures (always), every block (-v), counts, the worst pages and
// over-sketched pages. It returns whether the gate failed.
func report(w io.Writer, js []Judged, o options, work string) bool {
	counts := map[string]int{}
	pages := map[string]*pageStat{}
	failed := false
	for _, j := range js {
		if j.Verdict.Failing() {
			failed = true
		}
		if j.Block == nil {
			counts["orphan"]++
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
		counts["blocks"]++
		switch b.Class {
		case ClassSketch:
			ps.sketch++
			counts["sketch"]++
		case ClassRepl:
			ps.repl++
			counts["repl"]++
		default:
			ps.checked++
		}
		if b.Class == ClassMisfenced {
			ps.misfenced++
			counts["misfenced"]++
		}
		switch j.Verdict {
		case VOK:
			ps.pass++
			counts["pass_"+string(b.Class)]++
		case VBaselined:
			ps.baselined++
			counts["baselined"]++
		case VFail, VStale:
			ps.failing++
			counts[strings.ToLower(string(j.Verdict))]++
		}
		if j.Verdict.Failing() {
			printFailure(w, j, o, work)
		} else if o.verbose {
			fmt.Fprintf(w, "%-9s %s:%d [%s] %s\n", j.Verdict, b.Source, b.Line, tag(b), oc.Code)
		}
	}
	fmt.Fprintf(w, "\nblocks %d on %d pages | pass: program %d, parse %d, error= %d | skipped: sketch %d, repl %d | "+
		"baselined failing %d (misfenced %d) | NEW failures %d | stale baseline %d, orphaned %d\n",
		counts["blocks"], len(pages), counts["pass_program"], counts["pass_parse"], counts["pass_error"],
		counts["sketch"], counts["repl"], counts["baselined"], counts["misfenced"], counts["fail"], counts["stale"], counts["orphan"])
	worstPages(w, pages)
	if failed {
		fmt.Fprintln(w, "\nFAIL: fix the block, tag it (```ailang parse | sketch | repl | error=CODE | file=path.ail), "+
			"or, for a block that passes now, delete its baseline line. Tag grammar: .claude/rules/docs-review.md")
	} else {
		fmt.Fprintln(w, "\nOK: no new failing block, no stale baseline entry")
	}
	return failed
}

func printFailure(w io.Writer, j Judged, o options, work string) {
	if j.Block == nil {
		e := j.Entry
		fmt.Fprintf(w, "ORPHAN    %s %s (%s): baselined block no longer exists — delete this line from %s\n", e.Page, e.Key, e.Code, o.baseline)
		if o.github {
			fmt.Fprintf(w, "::error file=%s::doc-samples: orphaned baseline entry %s %s — delete it\n", o.baseline, e.Page, e.Key)
		}
		return
	}
	b, oc := j.Block, j.Outcome
	msg := oc.FirstErr
	if j.Verdict == VStale {
		why := "PASSES now"
		if !oc.Ran {
			why = "is tagged " + string(b.Class) + " and no longer checked"
		}
		msg = "listed in the baseline but " + why + " — delete its line: " + b.Page + " " + b.Key
	}
	fmt.Fprintf(w, "%-9s %s:%d [%s] %s: %s\n", j.Verdict, b.Source, b.Line, tag(b), firstNonEmpty(oc.Code, "-"), msg)
	if oc.Cmd != "" && j.Verdict == VFail {
		only := fmt.Sprintf("%s:%d", b.Source, b.Line)
		if o.keep != "" {
			fmt.Fprintf(w, "          repro: cd %s && %s\n", filepath.Join(work, filepath.Base(filepath.Dir(oc.Dir)), filepath.Base(oc.Dir)), oc.Cmd)
		} else {
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

// worstPages prints the 15 pages with the most baselined failures, and any page
// with more sketch blocks than checked ones (the "tagging exercise" risk).
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
	fmt.Fprintf(w, "\nworst pages (blocks / pass / baselined+failing / sketch / repl / misfenced):\n")
	for i, p := range ps {
		if i == 15 || p.baselined+p.failing == 0 {
			break
		}
		fmt.Fprintf(w, "  %-58s %3d %3d %3d %3d %3d %3d\n", p.page, p.blocks, p.pass, p.baselined+p.failing, p.sketch, p.repl, p.misfenced)
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
