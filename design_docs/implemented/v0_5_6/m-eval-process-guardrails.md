# M-EVAL-GUARD: Eval Process Guardrails & Orphan Detection

**Status**: Planned
**Target**: v0.5.6
**Priority**: P1 - Medium (operational stability)
**Estimated**: 2 hours
**Dependencies**: None

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Infrastructure, not syntax |
| Preserve Semantic Clarity | 0 | 0 | No semantic changes |
| Increase Determinism | + | +1 | Predictable eval termination |
| Lower Token Cost | + | +1 | Faster feedback when evals fail |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

### The Incident (December 2-3, 2025)

Two eval benchmark runs consumed CPU for **37+ hours**:

```
PID 62697 | 162.6% CPU | 2259 min | log_file_analyzer (claude-haiku-4-5)
PID 64150 | 107.6% CPU | 1471 min | merge_sort (claude-haiku-4-5)
```

### Root Cause Analysis

**The eval harness DOES have timeout handling** - research shows:

| Component | Timeout Mechanism | Location |
|-----------|-------------------|----------|
| Standard mode | `time.After()` + `Process.Kill()` | [runner.go:287-308](../../../internal/eval_harness/runner.go) |
| Agent mode | `time.NewTimer()` + `Process.Kill()` | [agent_runner_streaming.go:229-249](../../../internal/eval_harness/agent_runner_streaming.go) |
| Output limiting | `LimitedWriter` (1 MB max) | [runner.go:17-75](../../../internal/eval_harness/runner.go) |
| Workspace cleanup | `defer os.RemoveAll()` | [runner.go:230](../../../internal/eval_harness/runner.go) |

**So why did processes run for 37 hours?**

The issue is **orphaned processes**. When the parent eval harness process dies (crash, Ctrl+C, SSH disconnect), child `ailang run` processes become orphaned:

```
[Eval Harness] ──spawns──> [ailang run]
      │                          │
   dies/crashes            becomes orphan
      │                          │
      ✗                    reparented to launchd
                                 │
                           runs forever
```

**Evidence**: The processes were in `/var/folders/.../ailang_eval/...` paths, confirming they were eval harness spawns. The parent harness must have died without cleanup.

### Why Orphans Escape Timeouts

1. **Context not propagated**: `context.Background()` used at [eval_suite.go:612](../../../cmd/ailang/eval_suite.go), not connected to timeout
2. **No process groups**: Child processes not in same process group as parent
3. **Signal handlers incomplete**: Ctrl+C kills harness but not children
4. **No orphan detection**: No mechanism to find/kill abandoned processes

## Goals

**Primary Goal**: Ensure child processes die when parent harness dies, and detect/kill orphans.

**Success Metrics**:
- Ctrl+C on eval harness kills all child processes
- SSH disconnect doesn't leave orphans
- Watchdog detects and kills orphans older than max age
- Clear logging when orphans are killed

## Solution Design

### Overview

Two-layer defense against orphans:

1. **Layer 1: Process groups** - Children die with parent
2. **Layer 2: Watchdog** - Detect and kill escaped orphans

### Layer 1: Process Groups (Primary Fix)

Use Unix process groups to ensure children die with parent:

```go
// internal/eval_harness/runner.go - Modified cmd setup

func (r *AILANGRunner) Run(code string, timeout time.Duration) (*RunResult, error) {
    cmd := exec.Command(r.ailangPath, args...)

    // NEW: Put child in same process group
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,  // Create new process group with child as leader
    }

    // ... existing code ...

    // On timeout, kill entire process group
    select {
    case <-time.After(timeout):
        // Kill process group (negative PID)
        syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
        // ... rest of timeout handling
    }
}
```

**Signal handler for graceful shutdown**:

```go
// cmd/ailang/eval_suite.go - Add to main

func setupSignalHandlers(childPIDs *sync.Map) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-c
        log.Println("Received interrupt, killing child processes...")
        childPIDs.Range(func(key, value interface{}) bool {
            pid := key.(int)
            syscall.Kill(-pid, syscall.SIGKILL)  // Kill process group
            return true
        })
        os.Exit(1)
    }()
}
```

### Layer 2: Orphan Watchdog (Safety Net)

Background process that periodically checks for orphaned eval processes:

```go
// internal/eval_harness/watchdog.go (NEW FILE)

package eval_harness

import (
    "log"
    "os/exec"
    "strconv"
    "strings"
    "syscall"
    "time"
)

type Watchdog struct {
    MaxAge       time.Duration  // Kill processes older than this
    CheckPeriod  time.Duration  // How often to check
    Pattern      string         // Process pattern to match
    KilledCount  int
}

func NewWatchdog(maxAge, checkPeriod time.Duration) *Watchdog {
    return &Watchdog{
        MaxAge:      maxAge,
        CheckPeriod: checkPeriod,
        Pattern:     "ailang run.*benchmark/solution.ail",
    }
}

func (w *Watchdog) Start(done <-chan struct{}) {
    ticker := time.NewTicker(w.CheckPeriod)
    defer ticker.Stop()

    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            w.checkAndKill()
        }
    }
}

func (w *Watchdog) checkAndKill() {
    // Find ailang processes matching pattern
    // ps -eo pid,etimes,command | grep 'ailang run.*benchmark'
    // etimes = elapsed time in seconds

    cmd := exec.Command("bash", "-c",
        `ps -eo pid,etimes,command | grep -E 'ailang run.*benchmark' | grep -v grep`)
    output, _ := cmd.Output()

    for _, line := range strings.Split(string(output), "\n") {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }

        fields := strings.Fields(line)
        if len(fields) < 3 {
            continue
        }

        pid, _ := strconv.Atoi(fields[0])
        elapsedSec, _ := strconv.Atoi(fields[1])
        elapsed := time.Duration(elapsedSec) * time.Second

        if elapsed > w.MaxAge {
            log.Printf("WATCHDOG: Killing orphan PID %d (running %v, max %v)",
                pid, elapsed, w.MaxAge)
            syscall.Kill(pid, syscall.SIGKILL)
            w.KilledCount++
        }
    }
}

func (w *Watchdog) Report() string {
    if w.KilledCount == 0 {
        return "No orphaned processes detected"
    }
    return fmt.Sprintf("Killed %d orphaned processes", w.KilledCount)
}
```

### Integration with Eval Suite

```go
// cmd/ailang/eval_suite.go - Modified main

func runEvalSuite(...) {
    // Start watchdog
    watchdog := eval_harness.NewWatchdog(
        15*time.Minute,   // Max age before kill
        60*time.Second,   // Check every minute
    )
    watchdogDone := make(chan struct{})
    go watchdog.Start(watchdogDone)
    defer func() {
        close(watchdogDone)
        log.Println(watchdog.Report())
    }()

    // ... existing eval logic ...
}
```

## Implementation Plan

### Phase 1: Process Groups (~1 hour)

- [ ] Add `SysProcAttr{Setpgid: true}` to AILANGRunner ([runner.go:265](../../../internal/eval_harness/runner.go))
- [ ] Add `SysProcAttr{Setpgid: true}` to PythonRunner ([runner.go:142](../../../internal/eval_harness/runner.go))
- [ ] Add `SysProcAttr{Setpgid: true}` to agent runner ([agent_runner_streaming.go:78](../../../internal/eval_harness/agent_runner_streaming.go))
- [ ] Change `Process.Kill()` to `syscall.Kill(-pid, SIGKILL)` for process group kill
- [ ] Add signal handler in eval_suite.go to kill children on Ctrl+C

### Phase 2: Watchdog (~1 hour)

- [ ] Create `internal/eval_harness/watchdog.go` (~80 LOC)
- [ ] Add `--watchdog-max-age` flag (default: 15m)
- [ ] Add `--no-watchdog` flag for disabling
- [ ] Integrate with eval_suite main loop
- [ ] Add watchdog summary to eval report

## Files to Modify/Create

**New files:**
- `internal/eval_harness/watchdog.go` (~80 LOC)

**Modified files:**
- `internal/eval_harness/runner.go` - Process groups (~10 LOC)
- `internal/eval_harness/agent_runner_streaming.go` - Process groups (~5 LOC)
- `cmd/ailang/eval_suite.go` - Signal handlers, watchdog integration (~30 LOC)

**Total**: ~125 LOC

## Configuration

### CLI Flags (New)

```bash
# Watchdog settings
ailang eval-suite --watchdog-max-age 15m   # Kill processes older than 15m
ailang eval-suite --no-watchdog            # Disable watchdog (not recommended)

# Existing flags (unchanged)
ailang eval-suite --timeout 30s            # Per-execution timeout
```

### Environment Variables

```bash
# Emergency orphan cleanup (manual)
pkill -f 'ailang run.*benchmark/solution.ail'
```

## Success Criteria

- [ ] Ctrl+C on eval harness kills all child `ailang run` processes
- [ ] SSH disconnect doesn't leave orphaned processes
- [ ] Watchdog detects processes older than max age
- [ ] Killed orphans are logged
- [ ] Watchdog summary in eval report
- [ ] All existing eval tests still pass
- [ ] Works on macOS and Linux

## Testing Strategy

**Unit tests:**
- Test process group setup
- Test watchdog age detection
- Test signal handler registration

**Integration tests:**
- Start eval, send SIGTERM, verify no orphans
- Create long-running process, verify watchdog kills it

**Manual testing:**
- Run eval-suite, Ctrl+C mid-run, check `ps aux | grep ailang`
- Run eval-suite, close terminal, check for orphans after reconnect

## Current Implementation Reference

### Existing Timeout Code (Working)

**AILANGRunner** ([runner.go:287-308](../../../internal/eval_harness/runner.go)):
```go
select {
case <-time.After(timeout):
    _ = cmd.Process.Kill()   // ← Only kills single process
    <-done
    return &RunResult{TimedOut: true, ...}
case err := <-done:
    // Normal completion
}
```

**Agent Mode** ([agent_runner_streaming.go:229-249](../../../internal/eval_harness/agent_runner_streaming.go)):
```go
select {
case <-timer.C:
    _ = cmd.Process.Kill()   // ← Only kills single process
    return &ClaudeHeadlessResult{Subtype: "timeout", ...}
case err := <-done:
    // Normal completion
}
```

### Existing Cleanup Code (Working)

**Workspace cleanup** ([runner.go:230](../../../internal/eval_harness/runner.go)):
```go
defer os.RemoveAll(workspace)  // ← Works even on timeout
```

**Output limiting** ([runner.go:269-270](../../../internal/eval_harness/runner.go)):
```go
stdout := NewLimitedWriter(MaxOutputSize)  // 1 MB limit
stderr := NewLimitedWriter(MaxOutputSize)
```

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Process groups on Windows | Low | Windows doesn't use process groups; fall back to single kill |
| False positives | Low | Match specific pattern `ailang run.*benchmark` |
| Race between timeout and watchdog | Low | Watchdog is safety net, not primary mechanism |

## References

- Incident: December 2-3, 2025 (PIDs 62697, 64150)
- [internal/eval_harness/runner.go](../../../internal/eval_harness/runner.go) - Current timeout implementation
- [internal/eval_harness/agent_runner_streaming.go](../../../internal/eval_harness/agent_runner_streaming.go) - Agent timeout
- [cmd/ailang/eval_suite.go](../../../cmd/ailang/eval_suite.go) - CLI entry point
- Unix process groups: `man setpgid`

---

**Document created**: 2025-12-03
**Last updated**: 2025-12-04

---

## Implementation Notes (v0.5.6)

**Status**: IMPLEMENTED
**Implemented**: 2025-12-04
**Actual Time**: ~30 minutes

### Changes Made

**1. Process Group Support (`internal/eval_harness/runner.go`)**:
- Added `syscall` import
- Added `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` to both PythonRunner and AILANGRunner
- Changed timeout kill from `cmd.Process.Kill()` to `syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)` (negative PID kills process group)

**2. Watchdog (`internal/eval_harness/watchdog.go`)** - NEW FILE:
- `Watchdog` struct with configurable `MaxAge`, `CheckPeriod`, `Pattern`
- `Start()` method runs in background goroutine
- `checkAndKill()` uses `ps -eo pid,etimes,command` to find orphans
- `KillOrphans()` for immediate cleanup on shutdown
- `Report()` returns summary of killed processes

**3. Signal Handler (`cmd/ailang/eval_suite.go`)**:
- Added `os/signal` and `syscall` imports
- Watchdog starts with 15-minute max age, 60-second check period
- Signal handler catches SIGINT/SIGTERM, calls `watchdog.KillOrphans()` before exit
- Deferred cleanup reports any orphans killed

### Files Changed

| File | LOC | Description |
|------|-----|-------------|
| `internal/eval_harness/runner.go` | ~10 | Process groups + group kill |
| `internal/eval_harness/watchdog.go` | ~110 | NEW: Watchdog implementation |
| `cmd/ailang/eval_suite.go` | ~25 | Signal handler + watchdog integration |

**Total**: ~145 LOC

### Testing

- All existing eval harness tests pass
- Process groups verified on macOS (Darwin)
- Watchdog pattern matches `ailang run.*benchmark/solution.ail`

### Limitations

- Process groups are Unix-only (macOS, Linux)
- Windows uses fallback single-process kill (no process tree management)
- Watchdog uses `ps` command (Unix-specific, no-op on Windows)

### Platform-Specific Implementation (Added Post-Release)

To fix Windows CI build failure, syscalls were moved to platform-specific files:

| File | Build Tag | Purpose |
|------|-----------|---------|
| `process_unix.go` | `!windows` | Unix process groups with `Setpgid`, `Kill(-pid)` |
| `process_windows.go` | `windows` | Stubs using `os.Process.Kill()` only |

**API (platform-agnostic):**
```go
SetProcessGroup(cmd *exec.Cmd)    // Configure process group (no-op on Windows)
KillProcessGroup(pid int) error   // Kill group (single process on Windows)
KillProcess(pid int) error        // Kill single process
```
