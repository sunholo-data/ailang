# M-PROCESS-PGID-CLEANUP: kill process group on Process effect timeout

**Status**: Planned
**Target**: v0.18.9 (small follow-up patch)
**Priority**: P1 (silent hang of AILANG runtime when child shell pipelines orphan their grandchildren)
**Estimated**: ~120 LOC + tests + Windows portability shim
**Dependencies**: None (touches `internal/effects/process.go` only)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-09

---

## Source incident

2026-05-09, motoko_agent acceptance testing of v0.18.7 streaming + v0.18.8 thinking. After 4 successive BashExec calls in a single agent turn (Search → BashExec find .md → BashExec ls → **BashExec `find . -type d -name "*omnigraph*" -o -type f -name "*omnigraph*" 2>/dev/null | head -20`**), the AILANG `Process` effect's `cmd.Run()` never returned. `_complete` event never fired. Default 30s timeout did not trigger. Hang lasted >3 minutes before user killed bun.

Forensics from `.motoko/logfile/session_2026-05-09T19-24-19-571Z.jsonl`:
- 248 `thinking_delta` events delivered cleanly across 5 SSE streams (streaming wiring fine)
- Steps 0-2's `_complete` events all fired with proper `native_tool_results`
- Step 3's `v2_tool_dispatch_start` fired; nothing after that for 3+ min
- Bun process at 0% CPU when checked — waiting for AILANG to write
- Total JSONL volume only 47KB (well under 64KB pipe buffer — not back-pressure)
- `find . ... | head -20` from CLI returns 17 results in <1s — command itself isn't slow

## Hypothesis

`exec.CommandContext`'s context-cancellation only sends SIGKILL to the direct child PID. When the child is `bash -lc 'find ... | head -20'`, the bash process spawns `find` and `head` as grandchildren in its own process group. SIGKILL on bash leaves `find` and `head` orphaned (reparented to launchd/init). Go's stdout-drain goroutine still has the pipe open via the orphaned children — `cmd.Wait()` blocks waiting for the goroutine to finish, which never happens because the orphans are still writing.

Documented Go gotcha: https://github.com/golang/go/issues/23019 and others.

## Solution

Replace context-based kill with manual process-group kill:

```go
// internal/effects/process.go (POSIX path, build-tagged !windows)
cmd := exec.Command(resolvedPath, cmdArgs...)
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // child becomes pgid leader

done := make(chan error, 1)
if err := cmd.Start(); err != nil { ... }
go func() { done <- cmd.Wait() }()

select {
case err = <-done:
    // normal completion
case <-time.After(pc.Timeout):
    // Kill the ENTIRE process group (negative pid means pgid)
    _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    <-done // drain
    return makeProcessResultErrInt("Timeout", durationMs), nil
}
```

Windows path (`process_windows.go`): keep `exec.CommandContext` since `JOB_OBJECT` semantics differ — Windows has its own process-tree-kill mechanism via job objects.

## Acceptance

- [ ] New test: spawn `bash -lc 'sleep 60 | head -1'` with 1s timeout. Verify it returns within ~1.5s (not 60s) AND no orphan `sleep` processes remain after kill.
- [ ] Existing 30 process tests still pass.
- [ ] Add `t_process_pgid_cleanup` integration test in `process_managed_test.go`.
- [ ] DEBUG_PROCESS=1 trace lines (already shipped in v0.18.8) show `[process] done` within timeout window even for pipeline orphan scenarios.

## Out of scope

- General process-tree-aware management for `Process.spawn` / managed processes (separate concern; managed processes already handle group lifecycle via `process_context_managed.go`).
- Windows behavior — keep using `exec.CommandContext` (acceptable since Windows users are minority and the bug only manifests in shell-pipeline scenarios).
