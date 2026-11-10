package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
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
	Retried bool     `json:"retried,omitempty"` // timed out once and was retried at 2x
}

const (
	codeTimeout = "TIMEOUT"
	codeCrash   = "CRASH" // no JSON on stdout: the binary died or printed something else
)

// Runner runs `ailang check` per entry, isolated: perl alarm for the timeout (macOS
// has no timeout(1)), nice 15, a fixed number of workers.
type Runner struct {
	Jobs        int
	Timeout     int // seconds; a timeout is retried once at 2x
	PackageRoot string
	Nice        bool
}

type checkJSON struct {
	Passed *bool `json:"passed"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func (r *Runner) command(a *Arm, e Entry, timeout int) *exec.Cmd {
	args := []string{"-e", "alarm shift @ARGV; exec @ARGV or die qq(exec: $!)", strconv.Itoa(timeout), a.Bin, "check"}
	if e.Kind == "package" {
		args = append(args, "--package", "--json", ".")
	} else {
		args = append(args, "--json", e.Arg)
	}
	name := "perl"
	if r.Nice {
		args = append([]string{"-n", "15", "perl"}, args...)
		name = "nice"
	}
	cmd := exec.Command(name, args...)
	// The alarm kills the compiler, but a child it spawned (z3, a shell) can keep
	// stdout open; bound the wait for the pipes after the process is gone.
	cmd.WaitDelay = 2 * time.Second
	cmd.Dir = e.Cwd
	cmd.Env = append(a.Env(), "AILANG_PACKAGE_ROOT="+r.PackageRoot)
	return cmd
}

func (r *Runner) runOnce(a *Arm, e Entry, timeout int) Result {
	cmd := r.command(a, e, timeout)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	t0 := time.Now()
	err := cmd.Run()
	res := Result{Secs: float64(time.Since(t0).Milliseconds()) / 1000}
	var ee *exec.ExitError
	switch {
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		// exit 0; a lingering child's pipe was closed after WaitDelay
	case errors.As(err, &ee):
		res.RC = ee.ExitCode()
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			res.RC = 128 + int(ws.Signal())
			if ws.Signal() == syscall.SIGALRM {
				res.Codes = []string{codeTimeout}
				return res
			}
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
		set[normaliseCode(d.Code, d.Message)] = true
	}
	if len(j.Errors) > 0 {
		res.Message = j.Errors[0].Message
		res.Msg = firstLine(res.Message)
	}
	if len(set) == 0 {
		set["FAILED/no-diagnostic"] = true
	}
	for c := range set {
		res.Codes = append(res.Codes, c)
	}
	sort.Strings(res.Codes)
	return res
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
