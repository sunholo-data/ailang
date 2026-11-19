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
	"strings"
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
	Synth    bool     // passed/failed with a module line the gate synthesised
	Dropped  bool     // a parse-probe candidate that is not AILANG: not reported
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
