package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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
	// checkRSSHeadroom is what the check leaves under the instance's memory
	// limit for the server and the tarball held in memory. The cap itself is
	// derived from the limit at startup (instanceRSSCap) so it follows the
	// terraform setting: a hardcoded 384 MiB outlived the move to 1 GiB
	// (2026-11-02, sunholo/celestial@0.6.0 refused at 384 MiB on a 1 GiB box).
	checkRSSHeadroom uint64 = 128 << 20
	// fallbackRSSCap applies only when no memory limit is readable (a dev
	// machine); it is the cap for the original 512 MiB instance.
	fallbackRSSCap uint64 = 384 << 20
	// checkWallCap stays under the 300 s request timeout so the refusal is
	// delivered before Cloud Run abandons the request.
	checkWallCap = 200 * time.Second
	rssPollEvery = 100 * time.Millisecond
)

// checkRSSCap is set once in main from instanceRSSCap, before serving.
var checkRSSCap = fallbackRSSCap

// budgetError reports a child stopped for exceeding the validator's budget.
type budgetError struct {
	resource string // "memory" or "time"
	limit    string
	observed string
	context  string
}

func (e *budgetError) Error() string {
	return fmt.Sprintf("`ailang check --package .` exceeded the registry validator's %s budget (%s; observed %s) and was stopped.%s", e.resource, e.limit, e.observed, e.context)
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

// checkStats records only the check child, excluding dependency preparation.
type checkStats struct {
	PeakRSS uint64
	Wall    time.Duration
}

// checkEnv replaces all inherited pacing values, preserving other settings.
func checkEnv(inherited []string, cap uint64) ([]string, error) {
	const margin = 64 << 20
	if cap < margin+(1<<20) {
		return nil, fmt.Errorf("check RSS cap %d leaves no usable GOMEMLIMIT after 64 MiB margin", cap)
	}
	env := make([]string, 0, len(inherited)+1)
	for _, value := range inherited {
		if !strings.HasPrefix(value, "GOMEMLIMIT=") {
			env = append(env, value)
		}
	}
	if envValue(env, "GOMAXPROCS") == "unavailable" {
		env = append(env, fmt.Sprintf("GOMAXPROCS=%d", runtime.GOMAXPROCS(0)))
	}
	return append(env, fmt.Sprintf("GOMEMLIMIT=%dMiB", (cap-margin)>>20)), nil
}

func runBounded(dir string, cap uint64, wall time.Duration, readRSS rssReader, name string, args ...string) ([]byte, error) {
	out, _, err := runBoundedEnv(dir, cap, wall, readRSS, nil, name, args...)
	return out, err
}

// runBounded runs name+args in dir and returns its combined output. A child that
// passes rssCap or wall is killed and reported as a *budgetError; any other
// failure is returned as the plain run error.
func runBoundedEnv(dir string, rssCap uint64, wall time.Duration, readRSS rssReader, env []string, name string, args ...string) (output []byte, stats checkStats, runErr error) {
	start := time.Now()
	defer func() {
		stats.Wall = time.Since(start)
		if be, ok := asBudgetError(runErr); ok {
			be.context = refusalContext(rssCap, env, stats)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		return nil, stats, err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	ticker := time.NewTicker(rssPollEvery)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			if ctx.Err() == context.DeadlineExceeded {
				return out.Bytes(), stats, &budgetError{resource: "time", limit: wall.String(), observed: "still running at the cap"}
			}
			return out.Bytes(), stats, err
		case <-ticker.C:
			rss, ok := readRSS(cmd.Process.Pid)
			if !ok {
				continue
			}
			if rss > stats.PeakRSS {
				stats.PeakRSS = rss
			}
			if rss > rssCap {
				_ = cmd.Process.Kill()
				<-done
				return out.Bytes(), stats, &budgetError{resource: "memory", limit: mib(rssCap), observed: mib(stats.PeakRSS)}
			}
		}
	}
}

// memLimitSources are tried in order; the first that yields a finite limit wins.
// cgroup v2, cgroup v1, then MemTotal (Cloud Run gen1 runs under gVisor, which
// reports the instance limit as MemTotal and may expose no cgroup files).
var memLimitSources = []struct {
	path  string
	parse func([]byte) (uint64, bool)
}{
	{"/sys/fs/cgroup/memory.max", parseCgroupBytes},
	{"/sys/fs/cgroup/memory/memory.limit_in_bytes", parseCgroupBytes},
	{"/proc/meminfo", parseMemTotal},
}

// cgroupUnlimited: cgroup v1 reports "no limit" as a page-rounded max int64.
const cgroupUnlimited uint64 = 1 << 62

func parseCgroupBytes(b []byte) (uint64, bool) {
	v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64) // "max" fails here
	if err != nil || v == 0 || v >= cgroupUnlimited {
		return 0, false
	}
	return v, true
}

func parseMemTotal(b []byte) (uint64, bool) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) >= 2 && f[0] == "MemTotal:" {
			kb, err := strconv.ParseUint(f[1], 10, 64)
			if err != nil || kb == 0 {
				return 0, false
			}
			return kb << 10, true
		}
	}
	return 0, false
}

// instanceRSSCap returns the check's memory cap and where the limit came from.
func instanceRSSCap(readFile func(string) ([]byte, error)) (uint64, string) {
	for _, src := range memLimitSources {
		b, err := readFile(src.path)
		if err != nil {
			continue
		}
		if limit, ok := src.parse(b); ok {
			return rssCapForLimit(limit), fmt.Sprintf("%s limit %s", src.path, mib(limit))
		}
	}
	return fallbackRSSCap, "no readable memory limit; fallback"
}

// rssCapForLimit leaves checkRSSHeadroom under limit, or a quarter of it on an
// instance too small for the fixed headroom.
func rssCapForLimit(limit uint64) uint64 {
	if limit <= 2*checkRSSHeadroom {
		return limit / 4 * 3
	}
	return limit - checkRSSHeadroom
}

func mib(b uint64) string { return fmt.Sprintf("%d MiB", b>>20) }

// asBudgetError reports whether err is a validator budget refusal.
func asBudgetError(err error) (*budgetError, bool) {
	var be *budgetError
	ok := errors.As(err, &be)
	return be, ok
}
