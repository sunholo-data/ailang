# M-TELEMETRY-HOOKS: Worktree Session Enrichment - Handoff Document

**Status:** FAILED - Needs fresh implementation
**Date:** 2026-01-13
**Previous Attempt By:** Claude Opus 4.5 session

## Problem Statement

Coordinator-spawned Claude Code sessions (running in worktrees) don't have enriched spans in the observatory. Specifically:
- `session_tools` table has 0 entries for worktree sessions
- Tool use events (PreToolUse/PostToolUse) aren't being recorded
- Dashboard shows degraded display names for Bash tools

## Root Cause Analysis

### Finding 1: Claude Code Headless Mode Doesn't Load User Settings
When Claude Code runs with `-p` flag (headless/programmatic mode), it does NOT automatically load `~/.claude/settings.json`. The hooks configured there are ignored.

**Evidence:** Worktree sessions had no hook entries initially until `--settings` flag was added.

### Finding 2: Project Settings Override Global Settings
Even with `--settings <path>` flag, Claude Code ALSO loads `.claude/settings.json` from the working directory (project settings). These can override or conflict with the provided settings.

**Evidence:** Worktree's `.claude/settings.json` was using `$CLAUDE_PROJECT_DIR/scripts/hooks/claude_telemetry.sh` which pointed to OLD copies of the hook script in the git checkout.

### Finding 3: Worktrees Contain Stale Code
Git worktrees are created from commits. If the hook script is updated in the main repo but not committed, worktrees have the old version.

**Evidence:** Worktree scripts had old error message format `(observatory may not be running)` while repo had new format with `curl_exit=$X http_code=$X`.

## Changes Made (Partially Working)

### 1. Added `--settings` flag to Claude executor
**File:** `internal/executor/claude/claude.go`
```go
// Get AILANG-specific Claude settings path (creates hooks if needed)
settingsPath, err := executor.GetClaudeSettingsPath()
// ...
args := []string{
    "-p", task.Directive,
    "--settings", settingsPath, // Load AILANG hooks
    // ...
}
```

### 2. Created embedded hook deployment
**File:** `internal/executor/environment.go`
- Uses `//go:embed` to embed `claude_settings.json` and `claude_telemetry.sh`
- `GetClaudeSettingsPath()` deploys embedded files to `~/.ailang/claude/` and `~/.ailang/hooks/`
- Files are overwritten on each deploy to ensure latest version

### 3. Updated repo settings to use centralized hook path
**File:** `.claude/settings.json` (COMMITTED as 7b0a65b4)
- Changed from: `"$CLAUDE_PROJECT_DIR"/scripts/hooks/claude_telemetry.sh`
- Changed to: `~/.ailang/hooks/claude_telemetry.sh`

## Current State (BROKEN)

1. **SessionStart and Stop hooks:** Working for worktree sessions
2. **PreToolUse/PostToolUse hooks:** Processing but NOT recording success/failure
3. **Dashboard display names:** REGRESSED - Bash entries lost rich display
4. **session_tools table:** Does NOT exist in database

## What's Still Wrong

### Issue 1: No session_tools table
The observatory hooks endpoint accepts data (`{"status":"ok"}`) but there's no `session_tools` table to store it. Either:
- The table was never created
- The endpoint isn't actually storing data
- Data goes somewhere else

**Check:** `internal/server/handlers_observatory.go` - what does the hooks endpoint do?

### Issue 2: Hook result logging missing
The hook script logs "Processing EVENT_NAME" but doesn't log "Successfully reported" or "Failed to report". The curl command runs but result isn't captured.

**Possible causes:**
- Script execution interrupted by Claude Code timeout (3 seconds)
- Bash `set -euo pipefail` causing silent exit on curl failure
- Log file permissions or path issues

### Issue 3: Duplicate log entries
Every hook event produces 2x log entries (RAW_INPUT appears twice, Processing appears twice). Unknown cause.

### Issue 4: Display name regression
Dashboard Bash tool entries lost rich metadata. Need to check what data the endpoint expects vs what the hook sends.

## Files To Review

| File | Purpose |
|------|---------|
| `internal/executor/claude/claude.go` | Claude executor, adds `--settings` flag |
| `internal/executor/environment.go` | Embedded hooks deployment |
| `internal/executor/claude_settings.json` | Embedded settings template |
| `internal/executor/claude_telemetry.sh` | Embedded hook script |
| `scripts/hooks/claude_telemetry.sh` | Source hook script (symlinked from ~/.claude/hooks/) |
| `.claude/settings.json` | Project settings (uses centralized hook path now) |
| `internal/server/handlers_observatory.go` | Observatory hooks endpoint handler |
| `internal/server/server.go` | Server routes registration |

## Recommendations for Next Attempt

1. **Start fresh** - Don't try to build on this broken state

2. **Verify endpoint actually stores data**
   - Check `handlers_observatory.go` for what the `/api/observatory/hooks` endpoint does
   - Ensure `session_tools` table exists or create it
   - Verify data flow: hook → endpoint → database → dashboard

3. **Simplify hook script**
   - Remove duplicate logging
   - Add explicit error handling
   - Consider longer timeout (5s instead of 3s)
   - Log curl command before execution for debugging

4. **Test incrementally**
   - First: Verify endpoint stores data (curl directly)
   - Second: Verify hook script works standalone (run manually)
   - Third: Verify hook works from Claude Code (single session)
   - Fourth: Verify works from coordinator worktree

5. **Don't touch `.claude/settings.json`** without understanding the full hook loading order:
   - `--settings <path>` settings
   - User settings (`~/.claude/settings.json`)
   - Project settings (`.claude/settings.json` in cwd)
   - How these merge/override each other

6. **Use `make services-restart`** - Individual start/stop commands hang

## Key Lesson Learned

The worktree approach creates isolated git checkouts. Any file that's part of the repo (like `.claude/settings.json`) will be the VERSION FROM THAT COMMIT, not the latest. Either:
- Use absolute paths to centralized scripts (what we attempted)
- Don't store hook paths in repo settings at all
- Deploy hooks as part of worktree setup (copy fresh scripts)

## Commands For Debugging

```bash
# Check if server is running
curl -s http://localhost:1957/health

# Test hooks endpoint directly
curl -s -X POST "http://localhost:1957/api/observatory/hooks" \
  -H "Content-Type: application/json" \
  -d '{"event":"PreToolUse","session_id":"test","tool_name":"Bash","tool_use_id":"123"}'

# Check hook logs
tail -f ~/.ailang/state/telemetry_hooks.log

# Check what script worktree is using
cat ~/.ailang/state/worktrees/design-doc-creator/task-*/. claude/settings.json | jq '.hooks.PreToolUse'

# Check deployed centralized hook
cat ~/.ailang/hooks/claude_telemetry.sh

# Restart services properly
make services-restart

# Check database tables
sqlite3 ~/.ailang/state/collaboration.db ".tables"
```

## Commit History

- `7b0a65b4` - fix(hooks): use centralized telemetry hook path for worktrees (COMMITTED)

**Uncommitted changes:**
```
M internal/executor/claude_telemetry.sh  (+20 lines - debug curl logging)
M scripts/hooks/claude_telemetry.sh      (+20 lines - debug curl logging)
```

These changes added detailed curl error logging but the logging isn't appearing in output, suggesting the script isn't reaching that code path.

## What To Do With This Session's Changes

Option A: **Discard and start fresh**
```bash
git checkout -- internal/executor/claude_telemetry.sh scripts/hooks/claude_telemetry.sh
```

Option B: **Keep debug logging, investigate why it's not working**
The changes add `curl_exit`, `http_code`, and `response` to error messages. If these aren't appearing, the script is failing before reaching curl.
