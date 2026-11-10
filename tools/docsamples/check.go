package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Outcome is what checking one block produced, before the baseline is consulted.
type Outcome struct {
	Ran      bool     // false for sketch/repl (counted, not run)
	Pass     bool     // the block met its class's contract
	Code     string   // first diagnostic code (or a class: TC_UNDEFINED_VAR, ...); "" on pass
	Codes    []string // every code seen in the diagnostics
	FirstErr string   // one line of the first diagnostic
	Dir      string   // staged directory (kept with --keep)
	Cmd      string   // the command, relative to Dir
}

// Checker runs one staged block. The real one shells out to `ailang`; tests fake it.
type Checker interface {
	Check(ctx context.Context, dir, rel string) (codes []string, first string, ok bool, err error)
	Parse(ctx context.Context, dir, rel string) (codes []string, first string, ok bool, err error)
}

var codeRe = regexp.MustCompile(`\b([A-Z]{2,5}(?:\d{3}|_[A-Z][A-Z0-9_]+))\b`)

// diagCodes pulls diagnostic codes out of message text, in order, de-duplicated.
func diagCodes(msgs ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range msgs {
		for _, c := range codeRe.FindAllStringSubmatch(m, -1) {
			if !seen[c[1]] {
				seen[c[1]] = true
				out = append(out, c[1])
			}
		}
	}
	return out
}

// fallbackClass names a diagnostic that carries no code, so the baseline and
// `error=` have something stable to say about it.
func fallbackClass(msg string) string {
	l := strings.ToLower(msg)
	switch {
	case strings.Contains(l, "undefined variable"):
		return "TC_UNDEFINED_VAR"
	case strings.Contains(l, "effect"):
		return "TC_EFFECT"
	case strings.Contains(l, "type error") || strings.Contains(l, "unify") || strings.Contains(l, "mismatch"):
		return "TC_TYPE"
	case strings.Contains(l, "module") || strings.Contains(l, "import"):
		return "MODULE"
	}
	return "OTHER"
}

// firstLine picks the most useful line of a diagnostic: the first naming a code,
// else the first non-empty one.
func firstLine(msg string) string {
	var first string
	for _, l := range strings.Split(msg, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if first == "" {
			first = l
		}
		if codeRe.MatchString(l) {
			first = l
			break
		}
	}
	if len(first) > 200 {
		first = first[:200] + "…"
	}
	return first
}

// BinChecker is the real Checker: the ailang binary under test, isolated from the
// developer's environment so the verdict depends only on the commit.
type BinChecker struct {
	Bin    string
	Env    []string
	Budget time.Duration
}

// NewBinChecker builds the isolated environment: every AILANG_* variable from the
// caller is dropped (no relaxed modules, no stray stdlib, no shared cache), the
// compile cache is disabled and pointed into the work dir, and the stdlib is the
// checkout's own std/ when there is one.
func NewBinChecker(bin, work, stdlib string) *BinChecker {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "AILANG_") {
			env = append(env, kv)
		}
	}
	env = append(env, "AILANG_NO_CACHE=1", "AILANG_CACHE_DIR="+filepath.Join(work, "cache"))
	if stdlib != "" {
		env = append(env, "AILANG_STDLIB_PATH="+stdlib)
	}
	return &BinChecker{Bin: bin, Env: env, Budget: 60 * time.Second}
}

func (c *BinChecker) run(ctx context.Context, dir string, args ...string) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.Budget)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Bin, args...)
	cmd.Dir, cmd.Env = dir, c.Env
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if ctx.Err() != nil {
		return out.String(), errb.String(), -1, fmt.Errorf("timed out after %s", c.Budget)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return out.String(), errb.String(), ee.ExitCode(), nil
	}
	return out.String(), errb.String(), 0, err
}

// Check runs `ailang check --json`.
func (c *BinChecker) Check(ctx context.Context, dir, rel string) ([]string, string, bool, error) {
	stdout, stderr, rc, err := c.run(ctx, dir, "check", "--json", rel)
	if err != nil {
		return nil, "", false, err
	}
	var rep struct {
		Passed bool `json:"passed"`
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &rep); jerr != nil {
		if rc == 0 {
			return nil, "", true, nil
		}
		msg := strings.TrimSpace(stdout + "\n" + stderr)
		return []string{"CHECK_NO_JSON"}, firstLine(msg), false, nil
	}
	if rc == 0 && rep.Passed {
		return nil, "", true, nil
	}
	var msgs []string
	for _, e := range rep.Errors {
		if e.Code != "" && e.Code != "ERROR" {
			msgs = append(msgs, e.Code)
		}
		msgs = append(msgs, e.Message)
	}
	codes := diagCodes(msgs...)
	first := ""
	if len(rep.Errors) > 0 {
		first = firstLine(rep.Errors[0].Message)
		if len(diagCodes(rep.Errors[0].Message)) == 0 {
			codes = append([]string{fallbackClass(rep.Errors[0].Message)}, codes...)
		}
	}
	if len(codes) == 0 {
		codes = []string{"OTHER"}
	}
	return codes, first, false, nil
}

// Parse runs `ailang fmt`: exit 0 means the fragment parses (names are not resolved).
func (c *BinChecker) Parse(ctx context.Context, dir, rel string) ([]string, string, bool, error) {
	_, stderr, rc, err := c.run(ctx, dir, "fmt", rel)
	if err != nil {
		return nil, "", false, err
	}
	if rc == 0 {
		return nil, "", true, nil
	}
	codes := diagCodes(stderr)
	if rc != 3 || len(codes) == 0 {
		codes = append([]string{fmt.Sprintf("FMT_EXIT%d", rc)}, codes...)
	}
	return codes, firstLine(stderr), false, nil
}

// stagedFile is where a block lives in its page's temp tree, and its content there.
func stagedFile(b *Block) (rel, content string) {
	switch {
	case b.File != "":
		rel = b.File
		if moduleOf(b.Body) == "" {
			return rel, "module " + strings.TrimSuffix(rel, ".ail") + "\n\n" + b.Body
		}
		return rel, b.Body
	case moduleOf(b.Body) != "":
		return moduleOf(b.Body) + ".ail", b.Body
	}
	mod := fmt.Sprintf("docsample/b%d", b.Ordinal)
	return mod + ".ail", "module " + mod + "\n\n" + b.Body
}

func writeFile(dir, rel, content string) error {
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

// CheckAll checks every block, `jobs` at a time. Each block gets its own directory
// holding the page's `file=` siblings plus itself, so no block sees another's output.
func CheckAll(ctx context.Context, ck Checker, blocks []*Block, work string, jobs int) (map[*Block]*Outcome, error) {
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
			o, err := checkOne(ctx, ck, b, siblings[b.Page], filepath.Join(work, fmt.Sprintf("p%04d", pageIdx[b.Page]), fmt.Sprintf("b%03d", b.Ordinal)))
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

func checkOne(ctx context.Context, ck Checker, b *Block, sibs []*Block, dir string) (*Outcome, error) {
	o := &Outcome{Dir: dir}
	switch b.Class {
	case ClassSketch, ClassRepl:
		return o, nil
	case ClassBadTag:
		o.Ran, o.Code, o.Codes = true, "DOC_BADTAG", []string{"DOC_BADTAG"}
		o.FirstErr = fmt.Sprintf("unknown or conflicting info-string word %q (known: parse, sketch, repl, error=CODE, file=path.ail)", b.BadTag)
		return o, nil
	case ClassMisfenced:
		o.Ran, o.Code, o.Codes = true, "DOC_MISFENCED", []string{"DOC_MISFENCED"}
		lang := b.Lang
		if lang == "" {
			lang = "(none)"
		}
		o.FirstErr = fmt.Sprintf("AILANG fenced as %s: fence it ```ailang (add not-ailang to the info string if this is a false positive)", lang)
		return o, nil
	}
	o.Ran = true
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return o, err
	}
	for _, s := range sibs {
		if s == b {
			continue
		}
		rel, content := stagedFile(s)
		if err := writeFile(dir, rel, content); err != nil {
			return o, err
		}
	}
	rel, content := stagedFile(b)
	if err := writeFile(dir, rel, content); err != nil {
		return o, err
	}
	var codes []string
	var first string
	var ok bool
	var err error
	if b.Class == ClassParse {
		o.Cmd = "ailang fmt " + rel
		codes, first, ok, err = ck.Parse(ctx, dir, rel)
	} else {
		o.Cmd = "ailang check --json " + rel
		codes, first, ok, err = ck.Check(ctx, dir, rel)
	}
	if err != nil {
		return o, err
	}
	o.Codes = codes
	if b.Class == ClassError {
		if ok {
			o.Code, o.FirstErr = "DOC_EXPECTED_FAIL", fmt.Sprintf("tagged error=%s but the block now PASSES: fix the claim around it, then retag", b.WantCode)
			return o, nil
		}
		for _, c := range codes {
			if c == b.WantCode {
				o.Pass = true
				return o, nil
			}
		}
		o.Code = "DOC_WRONG_ERROR"
		o.FirstErr = fmt.Sprintf("tagged error=%s but got %s: %s", b.WantCode, strings.Join(codes, ","), first)
		return o, nil
	}
	o.Pass = ok
	if !ok {
		o.Code, o.FirstErr = codes[0], first
	}
	return o, nil
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
