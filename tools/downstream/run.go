package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Result is one entry checked by one arm.
type Result struct {
	Passed  bool     `json:"passed"`
	Codes   []string `json:"codes"` // normalised, sorted, unique; empty when passed
	RC      int      `json:"rc"`
	Secs    float64  `json:"secs"`
	Msg     string   `json:"msg,omitempty"`     // first diagnostic, first line, truncated
	Message string   `json:"-"`                 // full first-diagnostic text (expected_fail patterns)
	Diags   []string `json:"diags,omitempty"`   // every error as "code message", volatile text scrubbed, sorted
	Retried bool     `json:"retried,omitempty"` // timed out once and was retried at 2x
}

const (
	codeTimeout = "TIMEOUT"
	codeCrash   = "CRASH" // no JSON on stdout: the binary died or printed something else
	// codeNoDiagnostic: the check said "failed" and named no error.
	codeNoDiagnostic = "FAILED/no-diagnostic"
)

// Runner runs `ailang check` per entry, isolated: a Go-side deadline that kills the
// check's whole process group, nice 15, a fixed number of workers.
//
// The deadline is enforced here, not by a signal the compiler is asked to honour:
// a Go binary ignores SIGALRM unless it registers for it, so the earlier
// `perl -e 'alarm N; exec …'` wrapper never fired on the real compiler.
type Runner struct {
	Jobs        int
	Timeout     int // seconds; a timeout is retried once at 2x
	PackageRoot string
	Nice        bool
	RunDir      string // scrubbed out of diagnostics, with each arm's dir
}

// pipeGrace bounds the wait for stdout/stderr after the check is gone: a
// grandchild that escaped the group kill must not hold the gate open.
const pipeGrace = 2 * time.Second

type checkJSON struct {
	Passed *bool `json:"passed"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func (r *Runner) command(ctx context.Context, a *Arm, e Entry) *exec.Cmd {
	args := []string{"check"}
	if e.Kind == "package" {
		args = append(args, "--package", "--json", ".")
	} else {
		args = append(args, "--json", e.Arg)
	}
	name := a.Bin
	if r.Nice {
		args = append([]string{"-n", "15", a.Bin}, args...)
		name = "nice"
	}
	cmd := exec.CommandContext(ctx, name, args...)
	ownGroup(cmd) // on the deadline: SIGKILL the whole group, grandchildren included
	cmd.WaitDelay = pipeGrace
	cmd.Dir = e.Cwd
	cmd.Env = append(a.Env(), "AILANG_PACKAGE_ROOT="+r.PackageRoot)
	return cmd
}

func (r *Runner) runOnce(a *Arm, e Entry, timeout int) Result {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	cmd := r.command(ctx, a, e)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	t0 := time.Now()
	err := cmd.Run()
	res := Result{Secs: float64(time.Since(t0).Milliseconds()) / 1000}
	if errors.Is(err, exec.ErrWaitDelay) {
		// The check exited but something it started still held a pipe: the
		// group is still populated, so clean it up rather than leak it.
		_ = killGroup(cmd)
	}
	if ctx.Err() == context.DeadlineExceeded {
		res.RC = -1
		res.Codes = []string{codeTimeout}
		return res
	}
	var ee *exec.ExitError
	switch {
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		// exit 0; a lingering child's pipe was closed after WaitDelay
	case errors.As(err, &ee):
		res.RC = ee.ExitCode()
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			res.RC = 128 + int(ws.Signal())
		}
	default:
		res.RC = -1
		res.Codes, res.Msg = []string{codeCrash}, "could not start: "+err.Error()
		return res
	}
	out := stdout.String()
	var j checkJSON
	if i := strings.IndexByte(out, '{'); i < 0 || json.Unmarshal([]byte(out[i:]), &j) != nil || j.Passed == nil {
		res.Codes = []string{fmt.Sprintf("%s/rc=%d", codeCrash, res.RC)}
		res.Msg = firstLine(strings.TrimSpace(stderr.String() + " " + out))
		return res
	}
	res.Passed = *j.Passed
	if res.Passed {
		return res
	}
	set := map[string]bool{}
	for _, d := range j.Errors {
		// `check --package --json` lists warnings beside errors (code WARNING);
		// they never fail a check, so a new warning must not read as a new break.
		if d.Code == "WARNING" {
			continue
		}
		code := normaliseCode(d.Code, d.Message)
		set[code] = true
		res.Diags = append(res.Diags, code+" "+r.diagText(a, d.Message))
		if res.Message == "" {
			res.Message = d.Message
			res.Msg = firstLine(d.Message)
		}
	}
	if len(set) == 0 {
		set[codeNoDiagnostic] = true
	}
	for c := range set {
		res.Codes = append(res.Codes, c)
	}
	sort.Strings(res.Codes)
	sort.Strings(res.Diags)
	return res
}

var (
	tmpPathRE = regexp.MustCompile(`(?:/private)?/(?:var/folders|tmp)/[^\s'"` + "`" + `:,)]*`)
	spaceRE   = regexp.MustCompile(`\s+`)
)

// diagText is a diagnostic's message with the run's volatile text removed (each
// arm's scratch dir, the run dir, the package root, temp dirs), so the same error
// compares equal across arms while a different line, column or count does not.
func (r *Runner) diagText(a *Arm, msg string) string {
	for _, rep := range [][2]string{{a.Dir, "<arm>"}, {r.RunDir, "<run>"}, {r.PackageRoot, "<pkgroot>"}} {
		if rep[0] != "" {
			msg = strings.ReplaceAll(msg, rep[0], rep[1])
		}
	}
	msg = tmpPathRE.ReplaceAllString(msg, "<tmp>")
	msg = strings.TrimSpace(spaceRE.ReplaceAllString(msg, " "))
	if len(msg) > 400 {
		msg = msg[:400]
	}
	return msg
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func (r *Runner) runEntry(a *Arm, e Entry) Result {
	res := r.runOnce(a, e, r.Timeout)
	if len(res.Codes) == 1 && res.Codes[0] == codeTimeout {
		res = r.runOnce(a, e, 2*r.Timeout)
		res.Retried = true
	}
	return res
}

// RunArm checks every entry with one arm and returns results keyed by Entry.key().
func (r *Runner) RunArm(a *Arm, entries []Entry) map[string]Result {
	out := make(map[string]Result, len(entries))
	var mu sync.Mutex
	work := make(chan Entry)
	var wg sync.WaitGroup
	jobs := r.Jobs
	if jobs < 1 {
		jobs = 1
	}
	for i := 0; i < jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range work {
				res := r.runEntry(a, e)
				mu.Lock()
				out[e.key()] = res
				mu.Unlock()
			}
		}()
	}
	for _, e := range entries {
		work <- e
	}
	close(work)
	wg.Wait()
	return out
}
