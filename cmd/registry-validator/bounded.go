package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// The compile check runs in a child process on a shared Cloud Run instance
// (512 MiB, concurrency 80). Unbounded, one heavy package exceeds the instance
// limit, Cloud Run kills the whole container, and the publisher sees a bare
// HTTP 503 with no reason (#1811: crew_core and crew_lab, seven kills in three
// days). These caps stop the child first so the request can say why.
const (
	// checkRSSCap leaves room under the 512 MiB service limit for the server
	// and the tarball held in memory.
	checkRSSCap uint64 = 384 << 20
	// checkWallCap stays under the 300 s request timeout so the refusal is
	// delivered before Cloud Run abandons the request.
	checkWallCap = 200 * time.Second
	rssPollEvery = 100 * time.Millisecond
)

// budgetError reports a child stopped for exceeding the validator's budget.
type budgetError struct {
	resource string // "memory" or "time"
	limit    string
	observed string
}

func (e *budgetError) Error() string {
	return fmt.Sprintf("`ailang check --package .` exceeded the registry validator's %s budget (%s; observed %s) and was stopped. "+
		"The package itself may be valid: measure locally with a cold cache — "+
		"`AILANG_NO_CACHE=1 /usr/bin/time -l ailang check --package .` — and report the numbers to AILANG core.",
		e.resource, e.limit, e.observed)
}

// rssReader returns a process's resident set size in bytes, or false when it
// cannot be read (the process exited, or the platform has no /proc).
type rssReader func(pid int) (uint64, bool)

// procRSS reads VmRSS from /proc/<pid>/status (Linux, which is what Cloud Run runs).
func procRSS(pid int) (uint64, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, false
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "VmRSS:"))
		if len(fields) == 0 {
			return 0, false
		}
		kb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return 0, false
		}
		return kb << 10, true
	}
	return 0, false
}

// runBounded runs name+args in dir and returns its combined output. A child that
// passes rssCap or wall is killed and reported as a *budgetError; any other
// failure is returned as the plain run error.
func runBounded(dir string, rssCap uint64, wall time.Duration, readRSS rssReader, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	ticker := time.NewTicker(rssPollEvery)
	defer ticker.Stop()
	var peak uint64
	for {
		select {
		case err := <-done:
			if ctx.Err() == context.DeadlineExceeded {
				return out.Bytes(), &budgetError{resource: "time", limit: wall.String(), observed: "still running at the cap"}
			}
			return out.Bytes(), err
		case <-ticker.C:
			rss, ok := readRSS(cmd.Process.Pid)
			if !ok {
				continue
			}
			if rss > peak {
				peak = rss
			}
			if rss > rssCap {
				_ = cmd.Process.Kill()
				<-done
				return out.Bytes(), &budgetError{resource: "memory", limit: mib(rssCap), observed: mib(peak)}
			}
		}
	}
}

func mib(b uint64) string { return fmt.Sprintf("%d MiB", b>>20) }

// asBudgetError reports whether err is a validator budget refusal.
func asBudgetError(err error) (*budgetError, bool) {
	var be *budgetError
	ok := errors.As(err, &be)
	return be, ok
}
