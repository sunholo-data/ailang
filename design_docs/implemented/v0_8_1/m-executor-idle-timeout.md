# M-EXECUTOR-IDLE-TIMEOUT: Activity-Based Idle Timeout for Executors

**Status:** Planned
**Risk:** HIGH
**Target:** v0.8.0
**Priority:** Medium (nice-to-have, not blocking)

## Context

The coordinator daemon needs to detect when an executor process (Claude CLI, Gemini CLI) has stalled — producing zero output for an extended period. The hard timeout (default 60m) is too long to wait for obviously stuck processes.

### Post-Mortem: Feb 7-8 2026 Incident

Commit `4f4fa419` attempted to implement this feature and **broke ALL coordinator tasks for 24+ hours**. The implementation was reverted in `3ef6eb0b`.

**What went wrong:**
1. Two unrelated changes shipped in one commit (NVM discovery + idle timeout)
2. The dual-timer `for { select }` + `sync/atomic` approach introduced subtle concurrency issues
3. NVM binary resolution changed PATH semantics, masking the real failure mode
4. No smoke test was run after the change
5. Debugging was complicated because zero stdout could be caused by either change

**Root cause:** The NVM-resolved claude binary (`#!/usr/bin/env node`) needed the matching Node version's bin directory in PATH. The dual timeout masked this by attributing the zero-output hang to an "idle timeout" rather than a startup failure.

## Requirements

1. Kill executor processes that produce no stdout for a configurable period (default: 3m)
2. Must NOT interfere with normal operation (processes that ARE producing output)
3. Must distinguish "never started" (zero output from start) from "stalled mid-execution"
4. Must be testable in isolation (unit tests, not just integration)
5. Config: `idle_timeout` field on agent config (already exists, currently ignored)

## Proposed Design

### Option A: Goroutine-based activity monitor (Recommended)

Instead of dual timers in a `for { select }` loop, use a separate goroutine that periodically checks a `lastActivity` timestamp. The main goroutine stays as a simple `select`.

```go
// Main goroutine: unchanged simple select
select {
case <-timer.C:
    // hard timeout
case err := <-done:
    // normal completion
case <-idleKill:
    // idle monitor detected stall
}
```

The idle monitor runs as a separate goroutine:
```go
idleKill := make(chan struct{})
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            if time.Since(lastActivityTime()) > idleTimeout {
                close(idleKill)
                return
            }
        case <-done:
            return
        }
    }
}()
```

**Advantages:**
- Main select stays simple (3 cases, no loop)
- Idle monitoring is isolated and independently testable
- No `sync/atomic` needed if `lastActivity` uses `sync.Mutex` (simpler)

### Option B: Context-based timeout reset

Use `context.WithDeadline` and reset it on each stdout line. More Go-idiomatic but requires careful context management.

## Risk Mitigation

Given the HIGH risk rating from the Feb 2026 incident:

1. **Separate commit** — idle timeout MUST be a standalone commit, never bundled with other changes
2. **Feature flag** — gate behind `AILANG_ENABLE_IDLE_TIMEOUT=1` initially
3. **Smoke test** — automated test that sends a coordinator message and verifies completion within 60s
4. **Unit test** — test the idle monitor goroutine in isolation with a mock process
5. **Canary period** — run with feature flag for 1 week before making it default
6. **Separate from NVM changes** — any PATH/binary resolution changes must never ship in same commit

## Testing Plan

```go
func TestIdleMonitor_KillsAfterTimeout(t *testing.T) {
    // Mock process that produces output, then stops
    // Verify idle monitor fires after configured timeout
}

func TestIdleMonitor_DoesNotKillActiveProcess(t *testing.T) {
    // Mock process that produces output continuously
    // Verify idle monitor never fires
}

func TestIdleMonitor_NeverStarted(t *testing.T) {
    // Mock process that produces zero output from start
    // Verify idle monitor fires after timeout (not hard timeout)
}
```

## Files

- `internal/executor/claude/claude.go` — add idle monitor goroutine
- `internal/executor/claude/idle_monitor.go` — extracted idle monitoring logic (testable)
- `internal/executor/claude/idle_monitor_test.go` — unit tests
- `internal/executor/gemini/gemini.go` — same idle monitor (shared or duplicated)

## Non-Goals

- Idle timeout for script-based agents (they have their own timeout via `invoke.timeout`)
- Idle timeout based on stderr activity (only stdout matters for progress)
- Automatic retry after idle timeout (existing retry logic handles this)

## Effort Estimate

- Implementation: 2-3 hours
- Testing: 2 hours
- Canary monitoring: 1 week

## References

- Breaking commit: `4f4fa419` (Feb 7, 2026)
- Fix commit: `3ef6eb0b` (Feb 8, 2026)
- MEMORY.md: "NVM Claude Binary PATH Fix" section
