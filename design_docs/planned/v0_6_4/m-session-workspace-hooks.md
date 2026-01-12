# M-SESSION-WORKSPACE: Claude Code Rich Telemetry via Hooks

**Status**: Implemented
**Target**: v0.6.4
**Priority**: P1 (Medium)
**Estimated**: 6 hours (expanded scope)
**Actual**: 4 hours
**Dependencies**: None (uses existing hooks infrastructure)

> **Scope Expansion (2026-01-12)**: Original design focused only on workspace mapping.
> Updated to capture rich turn-level and tool-level metadata for reliable hierarchy building.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact - mapping is metadata, not execution |
| A2: Replayability | +1 | Improves trace attribution and correlation |
| A3: Effect Legibility | +1 | Makes session→workspace relationship explicit |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Enables machine analysis of session provenance |
| A8: Minimal Syntax | +1 | No new syntax - uses existing hook infrastructure |
| A9: Cost Visibility | +1 | Enables per-workspace cost aggregation |
| A10: Composability | 0 | No impact on composability |
| A11: Structured Failure | 0 | No failure mode changes |
| A12: System Boundary | +1 | Makes session→workspace boundary explicit |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects - hook explicitly reports metadata
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Designed for machine analysis of session data

## Problem Statement

**Claude Code telemetry is incomplete**, causing several dashboard hierarchy and filtering issues:

### Gap 1: Missing Workspace (process.cwd)

Claude Code OTEL spans don't include `process.cwd`, making workspace filtering impossible.

**Current State (as of 2026-01-12):**
```sql
-- AILANG spans: have workspace
SELECT COUNT(*) FROM spans WHERE json_extract(resource_attributes, '$."process.cwd"') IS NOT NULL;
-- Result: 1,247 spans with workspace

-- Claude Code spans: no workspace
SELECT COUNT(*) FROM spans
WHERE json_extract(resource_attributes, '$."service.name"') = 'claude-code'
  AND json_extract(resource_attributes, '$."process.cwd"') IS NOT NULL;
-- Result: 0 spans with workspace
```

### Gap 2: Empty Tool Metadata

Claude Code tool spans (`claude_code.tool.*`) lack rich metadata that would enable proper hierarchy building.

**Current State:**
```sql
SELECT name,
       json_extract(attributes, '$."tool.name"') as tool_name,
       json_extract(attributes, '$."tool.input"') as tool_input,
       json_extract(attributes, '$."tool_use_id"') as tool_use_id
FROM spans
WHERE name LIKE 'claude_code.tool.%'
LIMIT 5;
-- Result: ALL NULL - no tool metadata captured
```

**What we COULD have (from PreToolUse hooks):**
- `tool_name`: "Bash", "Edit", "Read", etc.
- `tool_input.command`: Actual command for Bash
- `tool_input.file_path`: File path for Read/Edit/Write
- `tool_input.old_string/new_string`: Edit details
- `tool_use_id`: Correlation ID for matching Pre/Post

### Gap 3: No Turn-Level Tracking

Currently we build hierarchy using timestamp correlation, which is fragile. Hooks could provide:
- Explicit turn boundaries (each `api_request` → set of tool calls)
- `tool_use_id` for precise correlation
- Session→Turn→ToolCall hierarchy

### Gap 4: Session Linkage

Current SessionStart hook checks inbox but doesn't POST session metadata to observatory.

**Impact Summary:**
| Gap | Dashboard Impact |
|-----|-----------------|
| No workspace | "Unknown Workspace" for all Claude Code |
| No tool metadata | Can't show what commands were run, files edited |
| No turn tracking | Fragile timestamp-based hierarchy |
| No session linkage | Can't reliably group turns into sessions |

## Goals

**Primary Goal:** Enable rich telemetry for Claude Code sessions including workspace, tool metadata, and reliable hierarchy.

**Success Metrics:**

| Metric | Target |
|--------|--------|
| Workspace attribution | 100% of new sessions have workspace |
| Tool metadata capture | tool_name, file_path populated for 100% of tool calls |
| Hierarchy reliability | Session→Turn→ToolCall hierarchy via tool_use_id |
| Dashboard filtering | Workspace filter works for Claude Code events |
| Cost aggregation | Per-workspace, per-session cost breakdown |

## Solution Design

### Overview

Use a **hybrid approach** that integrates hook data into the existing OTEL data flow:

1. **Hooks POST to lookup tables** - Session/tool metadata stored in `sessions` and `session_tools`
2. **OTLP receiver enriches spans on insert** - Checks lookup tables, adds `process.cwd` to span
3. **Existing queries work unchanged** - Already check `process.cwd` attribute

| Hook | Purpose | Data Captured |
|------|---------|---------------|
| **SessionStart** | Session metadata | session_id, workspace (cwd), claude_version |
| **PreToolUse** | Tool call start | tool_name, tool_input, tool_use_id, timestamp |
| **PostToolUse** | Tool call result | tool_use_id, success, tool_response |
| **Stop** | Session end | session_id, reason, final metrics |

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Claude Code Session                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   OTEL Telemetry (built-in)                 Hooks (our scripts)             │
│   ─────────────────────────                 ──────────────────              │
│   • api_request spans                       • SessionStart: cwd, session_id │
│   • claude_code.tool.* spans                • PreToolUse: tool_input        │
│   • session.id ✓                            • PostToolUse: tool_response    │
│   • process.cwd: NULL ✗                     • Stop: session end             │
│                                                                             │
│         ↓                                            ↓                      │
│   OTLP Exporter                              Shell script (curl)            │
│   (Claude Code)                              (we control)                   │
│                                                                             │
└─────────┬────────────────────────────────────────────┬──────────────────────┘
          ↓                                            ↓
┌─────────────────────┐                     ┌─────────────────────┐
│ OTLP Receiver       │                     │ Hooks API           │
│ /v1/traces          │                     │ /api/observatory/   │
│ (existing)          │                     │ hooks (new)         │
└─────────┬───────────┘                     └──────────┬──────────┘
          │                                            │
          │  ┌─────────────────────────────────────────┘
          │  │
          ▼  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Observatory SQLite                             │
│                                                                             │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│  │ sessions        │     │ session_tools   │     │ spans           │       │
│  │ ───────────     │     │ ─────────────   │     │ ─────           │       │
│  │ session_id  PK  │◀────│ session_id  FK  │     │ id          PK  │       │
│  │ workspace       │     │ tool_use_id PK  │     │ session.id attr │       │
│  │ claude_version  │     │ tool_name       │     │ process.cwd  ←──┼─ ENRICHED
│  │ started_at      │     │ tool_input      │     │ tokens, cost    │       │
│  └────────┬────────┘     │ tool_response   │     └─────────────────┘       │
│           │              │ success         │              ↑                │
│           │              └─────────────────┘              │                │
│           │                                               │                │
│           └───────── OTLP receiver checks ────────────────┘                │
│                      sessions table on insert,                             │
│                      enriches span with workspace                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

Existing queries (GetBreakdownByWorkspace) already check process.cwd → just works!
```

### Data Flow Detail

**Scenario 1: Hook arrives BEFORE OTEL span (normal case):**
```
SessionStart hook → INSERT INTO sessions (session_id, workspace)
                    ↓
OTEL span arrives → OTLP receiver checks: SELECT workspace FROM sessions WHERE session_id = ?
                    ↓
                    Span enriched: resource_attributes["process.cwd"] = workspace
                    ↓
                    INSERT INTO spans (with process.cwd populated)
```

**Scenario 2: OTEL span arrives BEFORE hook (race condition):**
```
OTEL span arrives → No session record yet
                    ↓
                    INSERT INTO spans (process.cwd = NULL)
                    ↓
SessionStart hook → INSERT INTO sessions (session_id, workspace)
                    ↓
                    UPDATE spans SET resource_attributes = json_set(..., 'process.cwd', workspace)
                    WHERE json_extract(attributes, '$."session.id"') = ?
                      AND json_extract(resource_attributes, '$."process.cwd"') IS NULL
```

**Tool metadata (separate table, not span enrichment):**
```
PreToolUse hook  → INSERT INTO session_tools (tool_use_id, tool_name, tool_input, start_time)
PostToolUse hook → UPDATE session_tools SET end_time, success, tool_response WHERE tool_use_id = ?
                   ↓
GetClaudeCodeHierarchy() → JOIN session_tools for rich tool metadata
```

**Components:**

1. **Global Hook Configuration** (`~/.claude/settings.json`):
   - Configures all 4 hooks at user level (applies to all projects)
   - Each hook runs a shell script that POSTs to observatory

2. **Sessions Table** (`observatory.db`):
   - Stores session_id → workspace mapping
   - Also captures: Claude version, start_time, end_time, turn_count
   - Idempotent: duplicate session_id updates rather than fails

3. **Session Tools Table** (`observatory.db`):
   - Stores tool call metadata keyed by `tool_use_id`
   - Links to session via `session_id`
   - Captures: tool_name, tool_input (JSON), start_time, end_time, success

4. **Observatory Hooks API** (`/api/observatory/hooks`):
   - Single POST endpoint for all hook events
   - Routes by `event` field to appropriate handler
   - On SessionStart: also UPDATE existing spans missing workspace
   - Returns 200 immediately (non-blocking for Claude Code)

5. **OTLP Receiver Enhancement** (`internal/observatory/otlp_receiver.go`):
   - On span insert, check if Claude Code span (`service.name = claude-code`)
   - Look up session by `session.id` attribute
   - If found, enrich span with `process.cwd` before storing
   - Existing queries work without any changes

### Implementation Plan

**Phase 1: Database Schema** (~1 hour) - COMPLETE
- [x] Add `sessions` table to observatory schema
- [x] Add `session_tools` table for tool call tracking
- [x] Create `internal/observatory/store_sessions.go` with CRUD operations
- [x] Write migration for existing observatory.db (v3 migration)

**Phase 2: OTLP Receiver Enrichment** (~1 hour) - COMPLETE
- [x] Modify `internal/observatory/otlp_receiver.go` to check sessions table
- [x] On Claude Code span insert, lookup workspace by session.id
- [x] Enrich span's resource_attributes with process.cwd before storing
- [x] Add backfill UPDATE on SessionStart for existing spans

**Phase 3: Hooks API Endpoint** (~1.5 hours) - COMPLETE
- [x] Add POST `/api/observatory/hooks` unified handler (`internal/server/handlers_hooks.go`)
- [x] Handle SessionStart: upsert sessions table + backfill spans
- [x] Handle PreToolUse: insert session_tools with start_time
- [x] Handle PostToolUse: update session_tools with end_time, success
- [x] Handle Stop: update sessions.ended_at

**Phase 4: Hook Scripts** (~1.5 hours) - COMPLETE
- [x] Create `scripts/hooks/claude_telemetry.sh` - unified hook script
- [x] Handle all 4 event types via $HOOK_EVENT_NAME routing
- [x] Extract data from stdin JSON, POST to observatory
- [x] Add project-level hooks in `.claude/settings.json`
- [x] Add global installation in `~/.claude/settings.json`
- [x] Symlink script to `~/.claude/hooks/`

**Phase 5: Testing & Verification** (~1 hour) - COMPLETE
- [x] All observatory tests passing
- [x] All backend interface implementations complete
- [x] Build successful
- [ ] End-to-end test with real Claude Code sessions (pending live test)

### Files to Modify/Create

**New files:**
- `internal/observatory/store_sessions.go` - Session/tool CRUD operations (~200 LOC)
- `internal/observatory/handlers_hooks.go` - Hooks API endpoint (~150 LOC)
- `scripts/hooks/claude_telemetry.sh` - Unified hook script (~100 LOC)

**Modified files:**
- `internal/observatory/schema.sql` - Add sessions + session_tools tables (~40 LOC)
- `internal/observatory/otlp_receiver.go` - Add session lookup + span enrichment (~50 LOC)
- `internal/server/server.go` - Register hooks API route (~5 LOC)
- `internal/observatory/backend_controlplane.go` - No changes needed (already uses process.cwd!)

## Examples

### Example 1: Global Hook Configuration

**User-level hook configuration** (`~/.claude/settings.json`):
```json
{
  "hooks": {
    "SessionStart": [{
      "hooks": [{
        "type": "command",
        "command": "~/.claude/hooks/claude_telemetry.sh",
        "timeout": 5
      }]
    }],
    "PreToolUse": [{
      "matcher": "*",
      "hooks": [{
        "type": "command",
        "command": "~/.claude/hooks/claude_telemetry.sh",
        "timeout": 3
      }]
    }],
    "PostToolUse": [{
      "matcher": "*",
      "hooks": [{
        "type": "command",
        "command": "~/.claude/hooks/claude_telemetry.sh",
        "timeout": 3
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "~/.claude/hooks/claude_telemetry.sh",
        "timeout": 5
      }]
    }]
  }
}
```

### Example 2: Unified Hook Script

**Hook script** (`~/.claude/hooks/claude_telemetry.sh`):
```bash
#!/bin/bash
# Report Claude Code telemetry to AILANG observatory
# Handles: SessionStart, PreToolUse, PostToolUse, Stop

set -euo pipefail

# Read hook JSON from stdin
HOOK_JSON=$(cat || echo "{}")

# Extract common fields
EVENT_NAME=$(echo "$HOOK_JSON" | jq -r '.hook_event_name // "unknown"')
SESSION_ID=$(echo "$HOOK_JSON" | jq -r '.session_id // "unknown"')
WORKSPACE=$(echo "$HOOK_JSON" | jq -r '.cwd // ""')

# If cwd not in payload, use PWD
[ -z "$WORKSPACE" ] && WORKSPACE=$(pwd)

# Build event payload based on event type
case "$EVENT_NAME" in
  "SessionStart")
    PAYLOAD=$(jq -n \
      --arg event "$EVENT_NAME" \
      --arg session_id "$SESSION_ID" \
      --arg workspace "$WORKSPACE" \
      --arg source "$(echo "$HOOK_JSON" | jq -r '.source // "startup"')" \
      '{
        event: $event,
        session_id: $session_id,
        workspace: $workspace,
        source: $source,
        timestamp: (now | todate)
      }')
    ;;
  "PreToolUse")
    PAYLOAD=$(jq -n \
      --arg event "$EVENT_NAME" \
      --arg session_id "$SESSION_ID" \
      --arg tool_name "$(echo "$HOOK_JSON" | jq -r '.tool_name // "unknown"')" \
      --arg tool_use_id "$(echo "$HOOK_JSON" | jq -r '.tool_use_id // ""')" \
      --argjson tool_input "$(echo "$HOOK_JSON" | jq '.tool_input // {}')" \
      '{
        event: $event,
        session_id: $session_id,
        tool_name: $tool_name,
        tool_use_id: $tool_use_id,
        tool_input: $tool_input,
        timestamp: (now | todate)
      }')
    ;;
  "PostToolUse")
    PAYLOAD=$(jq -n \
      --arg event "$EVENT_NAME" \
      --arg session_id "$SESSION_ID" \
      --arg tool_name "$(echo "$HOOK_JSON" | jq -r '.tool_name // "unknown"')" \
      --arg tool_use_id "$(echo "$HOOK_JSON" | jq -r '.tool_use_id // ""')" \
      --argjson tool_response "$(echo "$HOOK_JSON" | jq '.tool_response // {}')" \
      '{
        event: $event,
        session_id: $session_id,
        tool_name: $tool_name,
        tool_use_id: $tool_use_id,
        tool_response: $tool_response,
        timestamp: (now | todate)
      }')
    ;;
  "Stop")
    PAYLOAD=$(jq -n \
      --arg event "$EVENT_NAME" \
      --arg session_id "$SESSION_ID" \
      '{
        event: $event,
        session_id: $session_id,
        timestamp: (now | todate)
      }')
    ;;
  *)
    # Unknown event, skip
    exit 0
    ;;
esac

# POST to observatory (ignore failures silently - don't block Claude Code)
curl -s -X POST http://localhost:1957/api/observatory/hooks \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD" >/dev/null 2>&1 || true
```

### Example 3: Database Schema

**Sessions table:**
```sql
CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    workspace TEXT NOT NULL,
    claude_version TEXT,
    source TEXT DEFAULT 'hook',  -- 'hook', 'resume', 'compact'
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    turn_count INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace);
```

**Session tools table:**
```sql
CREATE TABLE IF NOT EXISTS session_tools (
    tool_use_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(session_id),
    tool_name TEXT NOT NULL,
    tool_input TEXT,           -- JSON blob
    tool_response TEXT,        -- JSON blob
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    success BOOLEAN,

    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);

CREATE INDEX IF NOT EXISTS idx_session_tools_session ON session_tools(session_id);
CREATE INDEX IF NOT EXISTS idx_session_tools_time ON session_tools(start_time DESC);
```

### Example 4: Workspace Breakdown Query (No Changes Needed!)

**Key benefit of the hybrid approach**: Existing `GetBreakdownByWorkspace()` already works!

Since spans are enriched with `process.cwd` on insert, the current query continues to work:

```sql
-- CURRENT QUERY (no changes needed)
WITH workspace_data AS (
  SELECT
    COALESCE(json_extract(resource_attributes, '$."process.cwd"'), 'unknown') as cwd,
    tokens_in, tokens_out, cost_usd, id
  FROM spans
)
-- ... rest of query unchanged
```

The OTLP receiver enrichment ensures Claude Code spans now have `process.cwd` populated,
so they automatically appear in the correct workspace bucket. No JOINs required!

### Example 5: Tool Hierarchy Query

**Get tool calls for a session with rich metadata:**
```sql
SELECT
    st.tool_name,
    st.tool_use_id,
    json_extract(st.tool_input, '$.command') as bash_command,
    json_extract(st.tool_input, '$.file_path') as file_path,
    st.success,
    (julianday(st.end_time) - julianday(st.start_time)) * 86400000 as duration_ms
FROM session_tools st
WHERE st.session_id = 'abc-123'
ORDER BY st.start_time ASC;

-- Result:
-- tool_name | tool_use_id    | bash_command | file_path                  | success | duration_ms
-- Bash      | toolu_01ABC... | git status   | NULL                       | 1       | 234
-- Read      | toolu_01DEF... | NULL         | /path/to/file.go           | 1       | 45
-- Edit      | toolu_01GHI... | NULL         | /path/to/file.go           | 1       | 89
```

## Success Criteria

- [x] SessionStart hook captures workspace for 100% of new sessions
- [x] PreToolUse/PostToolUse hooks capture tool metadata (name, input, response)
- [x] session_tools table populated with tool_use_id for hierarchy
- [x] All tests passing
- [x] Global hook config in `~/.claude/settings.json` (symlinked)
- [x] Project hook config in `.claude/settings.json` (git-tracked)
- [ ] Workspace breakdown shows separate entries for different projects (pending live test)
- [ ] Clicking workspace filter in UI correctly filters Event Queue (pending live test)
- [ ] GetClaudeCodeHierarchy uses session_tools for reliable hierarchy (future enhancement)

## Testing Strategy

**Unit tests:**
- `store_sessions_test.go` - CRUD for sessions and session_tools tables
- `handlers_hooks_test.go` - POST /api/observatory/hooks for all event types

**Integration tests:**
- End-to-end: Hook → API → DB → Query → Dashboard
- Test PreToolUse → PostToolUse correlation via tool_use_id
- Test session workspace attribution in workspace breakdown

**Manual testing:**
- Install global hooks in `~/.claude/settings.json`
- Start Claude Code in different workspaces
- Run various tools (Bash, Edit, Read)
- Verify:
  - Workspace filter works in dashboard
  - Tool hierarchy shows rich metadata (commands, file paths)
  - Session timeline accurate

## Non-Goals

**Not in this feature:**
- **Backfilling historical sessions** - Would require parsing conversation history
- **Auto-installing hook** - User must manually configure global hooks
- **Cross-machine session sync** - Local SQLite only, per-machine
- **Token/cost tracking in hooks** - Use existing OTEL spans for this
- **UserPromptSubmit hook** - Privacy concerns with prompt content

## Timeline

**Session 1** (~3 hours):
- Phase 1: Database Schema (1.5 hours)
- Phase 2: Hooks API Endpoint (1.5 hours)

**Session 2** (~3 hours):
- Phase 3: Hook Scripts (1.5 hours)
- Phase 4: Query Integration (1.5 hours)
- Testing & documentation

**Total: ~6 hours across two sessions**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hook not installed globally | Med | Provide clear installation instructions + ailang setup command |
| Observatory server not running | Low | Hooks fail silently (no crash), data lost until server starts |
| Session ID changes mid-session | Low | SessionStart only fires once per session |
| High latency from hook POST | Low | Async POST with 3-5s timeout, non-blocking |
| PreToolUse volume too high | Med | Batch or sample if >100 tools/session |
| Tool input too large (Edit content) | Med | Truncate tool_input to 10KB max |
| Hooks not running on all platforms | Low | Test on macOS, Linux; provide Windows docs |

## Current API/CLI Gaps Identified

During investigation, these gaps were found in the existing dashboard CLI and APIs:

| Gap | Current State | Recommendation |
|-----|---------------|----------------|
| No `ailang trace hierarchy` CLI | Must use API directly | Add CLI command for debugging |
| No session listing in CLI | Only via DB query | Add `ailang sessions list` |
| Observatory API 404 | `/api/observatory/breakdown/source-type` not registered | Check route registration |
| No workspace filter in Event Queue API | Only in breakdown | Add to `/api/controlplane/events` |
| Missing hook validation CLI | No way to test hooks | Add `ailang hooks test --event SessionStart` |

## Related Documents

**Implemented (informs design):**
- [agent-inbox-sessionstart-hook.md](design_docs/implemented/v0_3_14/agent-inbox-sessionstart-hook.md) - Existing hook infrastructure
- [M-CLAUDE-CODE-INTEGRATION-HOOKS.md](design_docs/implemented/v0_3_20/M-CLAUDE-CODE-INTEGRATION-HOOKS.md) - Hook patterns

**Planned (related):**
- [m-collab-provider-stats.md](design_docs/planned/v0_6_3/m-collab-provider-stats.md) - Provider statistics

## References

- [Claude Code Hooks Documentation](https://code.claude.com/docs/en/hooks.md) - Official hook format and events
- [Claude Code Hooks Guide](https://code.claude.com/docs/en/hooks-guide.md) - Detailed examples and patterns
- [OTEL Resource Attributes](https://opentelemetry.io/docs/specs/otel/resource/semantic_conventions/) - Standard attributes

## Future Work

**Phase 2 (v0.6.5):**
- **Auto-install hook during `ailang` setup** - First-run experience
- **ailang hooks test** - CLI to validate hook configuration
- **ailang sessions list** - CLI to view sessions with workspace

**Phase 3 (v0.7+):**
- **Backfill from transcript.jsonl** - Parse Claude Code transcript files for historical mapping
- **Additional metadata** - Git branch, project type, permission mode
- **Cross-session correlation** - Link resumed sessions to original workspace
- **Prompt-based hooks** - LLM-powered tool filtering or approval
- **MCP tool support** - Capture mcp__server__tool patterns

---

**Document created**: 2026-01-12
**Last updated**: 2026-01-12
