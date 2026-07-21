package eval_harness

// Memory watchdog for generated-code execution (M-EVAL-MEM-GUARD).
//
// Every eval lane executes model-generated programs. A generated program that
// allocates memory as fast as it can has, until now, only been bounded by the
// wall-clock timeout — on the local-model rotation that is 1500s, which is
// enough time to exhaust host RAM and panic the machine (2026-07-20 rig kernel
// panic: three generated Python processes at ~80-120GB each, watchdogd
// timeout, VM compressor exhausted).
//
// macOS does not reliably enforce RLIMIT_AS/RLIMIT_DATA, so the dependable
// approach is a polling watchdog: sample the child's PROCESS GROUP resident
// set size every memPollInterval and kill the whole group when it exceeds the
// cap. Group-level accounting matters because the interesting lanes are
// wrappers (`uv run` → python, `go run` → compiled binary): the allocation
// happens in a child of the process we started.

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	// EnvEvalMaxRSS is the environment variable that tunes the per-run
	// resident-memory cap for generated-code execution. Accepts a byte count
	// ("8589934592") or an integer with a K/M/G/T suffix ("8G", "512M";
	// "GB"/"GiB" spellings allowed, all binary multiples). "0" or "off"
	// disables the watchdog.
	EnvEvalMaxRSS = "AILANG_EVAL_MAX_RSS"

	// DefaultEvalMaxRSS is the default cap: 8 GiB. Far above what any
	// legitimate benchmark solution needs, far below host RAM even with
	// several eval lanes running concurrently.
	DefaultEvalMaxRSS = int64(8) << 30

	// memPollInterval is how often the watchdog samples process-group RSS.
	memPollInterval = 2 * time.Second

	// MemKillMarker prefixes the harness-written stderr of a run killed by the
	// memory watchdog. Error categorization keys on it (CategorizeRunError) so
	// these runs bank as resource_limit — a model failure — instead of
	// crashing the host or masquerading as a generic runtime_error.
	MemKillMarker = "[resource_limit]"
)

// evalMaxRSS returns the resident-memory cap in bytes: DefaultEvalMaxRSS when
// EnvEvalMaxRSS is unset, 0 when explicitly disabled, or an error when the
// value is unparseable. A bad value is an error, not a silent fallback — a
// misconfigured safety cap must fail loudly (CLAUDE.md §2).
func evalMaxRSS() (int64, error) {
	raw := strings.TrimSpace(os.Getenv(EnvEvalMaxRSS))
	if raw == "" {
		return DefaultEvalMaxRSS, nil
	}
	switch strings.ToLower(raw) {
	case "0", "off", "disabled", "none":
		return 0, nil
	}

	num := raw
	mult := int64(1)
	lower := strings.ToLower(raw)
	for _, s := range []struct {
		suffix string
		mult   int64
	}{
		{"kib", 1 << 10}, {"kb", 1 << 10}, {"k", 1 << 10},
		{"mib", 1 << 20}, {"mb", 1 << 20}, {"m", 1 << 20},
		{"gib", 1 << 30}, {"gb", 1 << 30}, {"g", 1 << 30},
		{"tib", 1 << 40}, {"tb", 1 << 40}, {"t", 1 << 40},
	} {
		if strings.HasSuffix(lower, s.suffix) {
			num = strings.TrimSpace(raw[:len(raw)-len(s.suffix)])
			mult = s.mult
			break
		}
	}

	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s=%q: want a byte count or integer with K/M/G/T suffix (e.g. \"8G\"), or \"off\" to disable", EnvEvalMaxRSS, raw)
	}
	return n * mult, nil
}

// guardedWait is the outcome of waitWithGuards.
type guardedWait struct {
	waitErr   error // result of cmd.Wait (after any kill)
	timedOut  bool  // wall-clock timeout fired
	memKilled bool  // memory watchdog fired
	peakRSS   int64 // highest sampled process-group RSS in bytes (best-effort)
}

// waitWithGuards waits for an already-started cmd (which MUST be in its own
// process group via SetProcessGroup) while enforcing both the wall-clock
// timeout and, when maxRSS > 0, the process-group resident-memory cap. On
// either breach the entire process group is killed, so wrapper children
// (uv → python, go run → binary) die with the leader.
func waitWithGuards(cmd *exec.Cmd, timeout time.Duration, maxRSS int64) guardedWait {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var tick <-chan time.Time
	if maxRSS > 0 {
		ticker := time.NewTicker(memPollInterval)
		defer ticker.Stop()
		tick = ticker.C
	}

	var g guardedWait
	for {
		select {
		case err := <-done:
			g.waitErr = err
			return g
		case <-timer.C:
			g.timedOut = true
			_ = KillProcessGroup(cmd.Process.Pid)
			// Drain Wait after the kill to avoid racing the goroutine.
			g.waitErr = <-done
			return g
		case <-tick:
			rss, ok := ProcessGroupRSS(cmd.Process.Pid)
			if !ok {
				continue // group already gone or ps unavailable; timeout still guards
			}
			if rss > g.peakRSS {
				g.peakRSS = rss
			}
			if rss > maxRSS {
				g.memKilled = true
				_ = KillProcessGroup(cmd.Process.Pid)
				g.waitErr = <-done
				return g
			}
		}
	}
}

// runGuarded executes an already-configured cmd (caller sets Dir/Stdin/Env)
// under the guards shared by every generated-code lane: its own process group,
// size-limited stdout/stderr capture, the wall-clock timeout, and the
// EnvEvalMaxRSS memory watchdog. It returns a base RunResult; callers layer on
// language-specific fields (compile-error detection, CodeHash, WorkspaceDir).
//
// timeoutMsg is the stderr text banked when the wall-clock timeout fires,
// preserving each lane's historical message (e.g. "execution timed out").
func runGuarded(cmd *exec.Cmd, timeout time.Duration, timeoutMsg string) *RunResult {
	start := time.Now()

	maxRSS, err := evalMaxRSS()
	if err != nil {
		// Fail loudly per-run: a typo'd cap must never silently run unguarded.
		return &RunResult{
			Stderr:    err.Error(),
			ExitCode:  -1,
			Duration:  time.Since(start),
			CompileOk: false,
			RuntimeOk: false,
		}
	}

	SetProcessGroup(cmd)

	stdout := NewLimitedWriter(MaxOutputSize)
	stderr := NewLimitedWriter(MaxOutputSize)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return &RunResult{
			Stderr:    err.Error(),
			ExitCode:  -1,
			Duration:  time.Since(start),
			CompileOk: false,
			RuntimeOk: false,
		}
	}

	g := waitWithGuards(cmd, timeout, maxRSS)
	switch {
	case g.timedOut:
		return &RunResult{
			Stdout:    stdout.String(),
			Stderr:    timeoutMsg,
			ExitCode:  -1,
			Duration:  timeout,
			CompileOk: true,
			RuntimeOk: false,
			TimedOut:  true,
		}
	case g.memKilled:
		return &RunResult{
			Stdout: stdout.String(),
			Stderr: fmt.Sprintf("%s process group resident memory %.2f GiB exceeded the eval cap %.2f GiB (%s); process tree killed",
				MemKillMarker, float64(g.peakRSS)/(1<<30), float64(maxRSS)/(1<<30), EnvEvalMaxRSS),
			ExitCode:    -1,
			Duration:    time.Since(start),
			CompileOk:   true,
			RuntimeOk:   false,
			MemExceeded: true,
		}
	}

	exitCode := 0
	if g.waitErr != nil {
		if exitErr, ok := g.waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return &RunResult{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		ExitCode:  exitCode,
		Duration:  time.Since(start),
		CompileOk: true,
		RuntimeOk: exitCode == 0,
	}
}

// CategorizeRunError extends CategorizeError with resource-limit promotion:
// a run whose stderr carries the memory-watchdog marker banks as
// resource_limit regardless of the compile/runtime/logic classification.
// Use this wherever a RunResult/ValidationResult stderr is available.
func CategorizeRunError(compileOk, runtimeOk, stdoutOk bool, stderr string) string {
	cat := CategorizeError(compileOk, runtimeOk, stdoutOk)
	if cat != ErrorCategoryNone && strings.Contains(stderr, MemKillMarker) {
		return ErrorCategoryResourceLimit
	}
	return cat
}
