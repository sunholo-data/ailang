package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// stagedFile is where a block lives in its page's temp tree, and its content there.
// A module line is synthesised only for a library fragment (no `module`, no `main`):
// a block that defines main is a program a reader copies into a file and runs, so it is
// checked exactly as written and fails MOD014 without its own module line. synth
// reports whether the gate added the line.
func stagedFile(b *Block) (rel, content string, synth bool) {
	switch {
	case b.File != "":
		rel = b.File
		if moduleOf(b.Body) == "" {
			return rel, "module " + strings.TrimSuffix(rel, ".ail") + "\n\n" + b.Body, true
		}
		return rel, b.Body, false
	case moduleOf(b.Body) != "":
		return moduleOf(b.Body) + ".ail", b.Body, false
	}
	mod := fmt.Sprintf("docsample/b%d", b.Ordinal)
	if definesMain(b.Body) || b.Class == ClassParse || b.Class == ClassCandidate {
		return mod + ".ail", b.Body, false
	}
	return mod + ".ail", "module " + mod + "\n\n" + b.Body, true
}

func writeFile(dir, rel, content string) error {
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

// elision matches `...` that is not a spread (`...rest`): an elided body in a snippet.
var elision = regexp.MustCompile(`\.\.\.([^A-Za-z_]|$)`)

// CheckAll checks every block, `jobs` at a time. Each block gets its own directory
// holding the page's `file=` siblings plus itself, so no block sees another's output.
// Imported .ail files are checked in place under root (their imports are repo paths).
// A ClassCandidate block is parsed: if it parses as AILANG it becomes ClassMisfenced,
// otherwise its outcome is marked Dropped.
func CheckAll(ctx context.Context, ck Checker, blocks []*Block, work, root string, jobs int) (map[*Block]*Outcome, error) {
	siblings := map[string][]*Block{}
	pageIdx := map[string]int{}
	for _, b := range blocks {
		if _, ok := pageIdx[b.Page]; !ok {
			pageIdx[b.Page] = len(pageIdx) + 1
		}
		if b.File != "" && b.Class != ClassBadTag {
			siblings[b.Page] = append(siblings[b.Page], b)
		}
	}
	out := make(map[*Block]*Outcome, len(blocks))
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	for _, b := range blocks {
		b := b
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			dir := filepath.Join(work, fmt.Sprintf("p%04d", pageIdx[b.Page]), fmt.Sprintf("b%03d", b.Ordinal))
			if b.Class == ClassFile {
				dir = root
			}
			o, err := checkOne(ctx, ck, b, siblings[b.Page], dir)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = fmt.Errorf("%s:%d: %w", b.Source, b.Line, err)
			}
			out[b] = o
		}()
	}
	wg.Wait()
	return out, firstErr
}

func structuralOutcome(o *Outcome, code, msg string) (*Outcome, error) {
	o.Ran, o.Code, o.Codes, o.FirstErr = true, code, []string{code}, msg
	return o, nil
}

func checkOne(ctx context.Context, ck Checker, b *Block, sibs []*Block, dir string) (*Outcome, error) {
	o := &Outcome{Dir: dir}
	switch b.Class {
	case ClassSketch, ClassRepl:
		return o, nil
	case ClassStructural:
		return structuralOutcome(o, b.Defect, b.DefectMsg)
	case ClassBadTag:
		return structuralOutcome(o, "DOC_BADTAG", fmt.Sprintf("unknown or conflicting info-string word %q (known: parse, sketch, repl, error=CODE, file=path.ail)", b.BadTag))
	case ClassMisfenced:
		return structuralOutcome(o, "DOC_MISFENCED", misfenceMsg(b))
	case ClassFile:
		o.Ran, o.Cmd = true, "ailang check --json "+b.Source
		codes, first, ok, err := ck.Check(ctx, dir, b.Source)
		if err != nil {
			return o, err
		}
		o.Pass, o.Codes = ok, codes
		if !ok {
			o.Code, o.FirstErr = codes[0], first
		}
		return o, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return o, err
	}
	for _, s := range sibs {
		if s == b {
			continue
		}
		rel, content, _ := stagedFile(s)
		if err := writeFile(dir, rel, content); err != nil {
			return o, err
		}
	}
	rel, content, synth := stagedFile(b)
	if b.Class == ClassCandidate {
		content = elision.ReplaceAllString(content, "elided$1")
	}
	if err := writeFile(dir, rel, content); err != nil {
		return o, err
	}
	var codes []string
	var first string
	var ok bool
	var err error
	if b.Class == ClassParse || b.Class == ClassCandidate {
		o.Cmd = "ailang fmt " + rel
		codes, first, ok, err = ck.Parse(ctx, dir, rel)
	} else {
		o.Cmd = "ailang check --json " + rel
		codes, first, ok, err = ck.Check(ctx, dir, rel)
	}
	if err != nil {
		return o, err
	}
	if b.Class == ClassCandidate {
		if !ok {
			o.Dropped = true
			return o, nil
		}
		b.Class = ClassMisfenced
		return structuralOutcome(o, "DOC_MISFENCED", misfenceMsg(b)+" — it parses as AILANG")
	}
	o.Ran, o.Codes, o.Synth = true, codes, synth
	if b.Class == ClassError {
		// The PRIMARY diagnostic must be the tagged code, exactly: a block that trips an
		// unrelated error first does not demonstrate the one the page is about.
		switch {
		case ok:
			o.Code, o.FirstErr = "DOC_EXPECTED_FAIL", fmt.Sprintf("tagged error=%s but the block now PASSES: fix the claim around it, then retag", b.WantCode)
		case codes[0] == b.WantCode:
			o.Pass = true
		default:
			o.Code = "DOC_WRONG_ERROR"
			o.FirstErr = fmt.Sprintf("tagged error=%s but the first error is %s: %s", b.WantCode, codes[0], first)
		}
		return o, nil
	}
	o.Pass = ok
	if !ok {
		o.Code, o.FirstErr = codes[0], first
	}
	return o, nil
}

func misfenceMsg(b *Block) string {
	lang := b.Lang
	if lang == "" {
		lang = "(none)"
	}
	return fmt.Sprintf("AILANG fenced as %s: fence it ```ailang (add not-ailang to the info string if this is a false positive)", lang)
}

// sortBlocks orders blocks by page, then line — the report and baseline order.
func sortBlocks(bs []*Block) {
	sort.SliceStable(bs, func(i, j int) bool {
		if bs[i].Page != bs[j].Page {
			return bs[i].Page < bs[j].Page
		}
		return bs[i].Line < bs[j].Line
	})
}
