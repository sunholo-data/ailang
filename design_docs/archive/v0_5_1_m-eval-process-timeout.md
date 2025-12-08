# M-EVAL-TIMEOUT: Process-Level Timeout for Eval Runs

## Problem Statement

Eval benchmark runs can spawn `ailang run` processes that get stuck in infinite loops, consuming massive CPU resources indefinitely. On November 29, 2025, two stuck processes were discovered:

| PID | CPU | Duration | Benchmark |
|-----|-----|----------|-----------|
| 15957 | 202.7% | 442 minutes | csv_to_json_converter |
| 92634 | 99.2% | 212 minutes | tree_transformation_pipeline |

These processes escaped the eval harness timeout mechanism and ran for 7+ hours, consuming 300% CPU combined.

## Root Cause Analysis

The current timeout implementation has a gap:

1. **Eval harness timeout**: `RunHeadlessSessionStreaming` has a timeout for the Claude session itself
2. **Missing**: No timeout on the `ailang run` subprocess that executes the generated code
3. **Result**: If AI generates an infinite loop, the subprocess runs forever

The timeout applies to Claude's response time, not to the actual code execution.

## Proposed Solution

### 1. Add Process-Level Timeout to `ailang run`

Add a `--timeout` flag to the `ailang run` command that kills the process after N seconds:

```bash
# Current (no timeout)
ailang run --entry main --caps IO benchmark/solution.ail

# Proposed
ailang run --entry main --caps IO --timeout 60 benchmark/solution.ail
```

Implementation in `cmd/ailang/run.go`:
```go
func runWithTimeout(timeout time.Duration, fn func() error) error {
    done := make(chan error, 1)
    go func() { done <- fn() }()

    select {
    case err := <-done:
        return err
    case <-time.After(timeout):
        return fmt.Errorf("execution timeout after %v", timeout)
    }
}
```

### 2. Add Process Group Kill in Eval Harness

The eval harness should track spawned processes and kill them on timeout:

```go
// In internal/eval_harness/runner.go
cmd := exec.Command("ailang", "run", ...)
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

// On timeout, kill the entire process group
if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
    log.Printf("Failed to kill process group: %v", err)
}
```

### 3. Add Watchdog for Orphaned Processes

A background goroutine that monitors for runaway `ailang run` processes:

```go
// In eval harness startup
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        killOrphanedEvalProcesses(maxRuntime: 30*time.Minute)
    }
}()
```

## Implementation Plan

### Phase 1: Immediate Fix (v0.5.0)
- [ ] Add `--timeout` flag to `ailang run` command
- [ ] Default timeout: 120 seconds for eval runs
- [ ] Eval harness passes `--timeout` to spawned processes

### Phase 2: Process Group Management (v0.5.1)
- [ ] Use process groups (`Setpgid`) for spawned processes
- [ ] Kill entire process group on timeout (catches child processes)
- [ ] Add cleanup on eval harness exit (kill all spawned processes)

### Phase 3: Monitoring (v0.5.2)
- [ ] Add metrics for process duration and resource usage
- [ ] Watchdog for orphaned processes
- [ ] Alert on processes exceeding thresholds

## Configuration

```yaml
# In internal/eval_harness/config.go
type EvalConfig struct {
    ProcessTimeoutSeconds int  `yaml:"process_timeout_seconds"` // Default: 120
    MaxCPUPercent         int  `yaml:"max_cpu_percent"`         // Default: 100
    MaxMemoryMB           int  `yaml:"max_memory_mb"`           // Default: 512
    WatchdogEnabled       bool `yaml:"watchdog_enabled"`        // Default: true
}
```

## Testing

1. Create a benchmark with intentional infinite loop
2. Verify it's killed within timeout period
3. Verify process group cleanup works
4. Verify watchdog catches orphaned processes

## Migration

No breaking changes. New `--timeout` flag is optional with a sensible default.

## References

- Incident: November 29, 2025 - 2 stuck processes consuming 300% CPU
- Related: M-EVAL-LOOP eval infrastructure
- Files: `internal/eval_harness/runner.go`, `cmd/ailang/run.go`
