# Sprint Plan: M-HTTP-HOOKS-CLOUD-TELEMETRY

## Summary

Migrate AILANG's Claude Code telemetry from shell-script command hooks to native HTTP hooks, eliminating bash/jq/curl dependencies and enabling cloud deployment. This is a prerequisite for M-CLOUD-INFRA.

**Duration:** 5 days (estimated 35 hours)
**Dependencies:** `ailang serve` running locally for integration testing
**Risk Level:** Low — existing endpoints do the hard work; this is mostly a protocol upgrade

## Current Status Analysis

### Completed Recently
- M-PERF-OBSERVATORY: Two-phase aggregation, trace summaries, CLI/API sync (~600 LOC)
- M-PROTOCOL-SUPPORT: Multi-protocol support for serve-api (~1400 LOC)
- M-SMT-BOUNDED-RECURSION: SMT verification features (~1740 LOC)
- Release v0.8.1.1

### Velocity
- Recent average: ~200-300 LOC/day (impl + tests) based on last 14 days
- This sprint: ~580 LOC total (well within capacity for 5 days)
- Low-risk infrastructure work (no parser/type system changes)

### Remaining from Design Doc
- ⏳ Phase 1: Unified hook receiver endpoint (~200 LOC)
- ⏳ Phase 2: HTTP hook configuration (~80 LOC config changes)
- ⏳ Phase 3: Coordinator environment injection (~25 LOC)
- ⏳ Phase 4: Remove command telemetry hooks (config-only)
- ⏳ Phase 5: New event support (~100 LOC)
- 📋 Phase 6: Auth middleware (~50 LOC) — cloud-only, can defer

## Proposed Milestones

### M1: Unified Hook Receiver Endpoint
**Goal:** Create `POST /api/hooks/claude` that accepts raw Claude Code hook JSON and routes to existing observatory/exec storage.
**Estimated:** 200 LOC implementation + 300 LOC tests = 500 LOC
**Duration:** 2 days

**Tasks:**
- Day 1 AM: Create `internal/server/handlers_claude_hooks.go` with `ClaudeHookPayload` struct matching Claude Code's full JSON schema. Implement `HandleClaudeHook` with event routing switch. Extract AILANG correlation IDs from HTTP headers (`X-AILANG-Task-ID`, etc.).
- Day 1 PM: Wire handler methods for SessionStart, PreToolUse, PostToolUse, Stop — delegate to existing `obsBackend` methods (UpsertSession, InsertToolStart, UpdateToolEnd, UpdateSessionEnded). Register route in `server.go`.
- Day 2 AM: Add handlers for new events: SubagentStart, SubagentStop (store in new `subagent_events` or reuse sessions table), TaskCompleted, PostToolUseFailure.
- Day 2 PM: Write tests in `handlers_claude_hooks_test.go` — unit tests for JSON parsing of all 7 event types, header extraction, routing. Integration test: POST sample payloads, verify observatory rows.

**Acceptance Criteria:**
- [x] `POST /api/hooks/claude` accepts SessionStart, PreToolUse, PostToolUse, PostToolUseFailure, SubagentStart, SubagentStop, TaskCompleted, Stop
- [x] Full JSON payload stored (no field subsetting — `json.RawMessage` for tool_input/response)
- [x] Correlation IDs extracted from `X-AILANG-*` headers
- [x] Existing `/api/observatory/hooks` endpoint unchanged (backward compatible)
- [x] All tests passing, `make lint` clean
- [x] Test coverage >85% on new handler file

**Risks:**
- Observatory schema may need extension for SubagentStart/Stop events — Mitigation: store in existing `session_tools` table with event type column, or add lightweight `subagent_events` table

### M2: HTTP Hook Configuration + Coordinator Env
**Goal:** Update `.claude/settings.json` to use HTTP hooks alongside command hooks (dual-mode), and inject `AILANG_HUB_URL` from coordinator.
**Estimated:** 25 LOC Go + 80 LOC config = 105 LOC
**Duration:** 1 day

**Tasks:**
- Day 3 AM: Update `internal/executor/environment.go` to inject `AILANG_HUB_URL` (from config or default `http://127.0.0.1:1957`) and `AILANG_HUB_TOKEN` (if configured) into Claude Code process environment. Add `hub` section parsing to coordinator config.
- Day 3 PM: Update `.claude/settings.json` to add HTTP hooks for all telemetry events (SessionStart, PreToolUse, PostToolUse, PostToolUseFailure, SubagentStart, SubagentStop, TaskCompleted, Stop) while keeping existing command hooks (dual-mode for validation). Set `AILANG_HUB_URL` default in local environment.

**Acceptance Criteria:**
- [x] HTTP hooks fire alongside command hooks (both populate observatory)
- [x] `AILANG_HUB_URL` and `AILANG_HUB_TOKEN` injected by coordinator into Claude Code env
- [x] Hub URL defaults to `http://127.0.0.1:1957` when not configured
- [x] Config parsing tested (unit test for hub section)
- [x] All tests passing, `make lint` clean

**Risks:**
- HTTP hooks with unset `AILANG_HUB_URL` will fail silently — Mitigation: Claude Code's HTTP hooks are non-blocking on failure; set default URL in settings or env

### M3: Remove Command Hooks + Validate
**Goal:** Remove `claude_telemetry.sh` command hooks, verify HTTP-only telemetry works, add auth middleware for cloud readiness.
**Estimated:** 50 LOC (auth middleware) + 0 LOC (config removal) = 50 LOC
**Duration:** 1 day

**Tasks:**
- Day 4 AM: Run a full Claude Code session with dual-mode hooks. Compare observatory data from HTTP hooks vs command hooks — verify parity. Check dashboard WebSocket stream still receives live events.
- Day 4 PM: Remove `~/.ailang/hooks/claude_telemetry.sh` command hooks from `.claude/settings.json` (keep command hooks for session_start.sh, agent_handoff.sh, format_go.sh). Create `internal/server/middleware_auth.go` — bearer token validation that skips when no token configured (local mode). Update CLAUDE.md hooks documentation.

**Acceptance Criteria:**
- [x] `claude_telemetry.sh` removed from all hook configs
- [x] Observatory still receives all events via HTTP hooks only
- [x] Dashboard WebSocket still receives live events
- [x] Auth middleware blocks unauthenticated requests when token configured
- [x] Auth middleware passes through when no token configured (local mode)
- [x] CLAUDE.md updated with new hook architecture
- [x] All tests passing, `make lint` clean

**Risks:**
- Removing command hooks before HTTP hooks are validated could lose telemetry — Mitigation: dual-mode in M2 provides overlap period; only remove after verified parity

### M4: Documentation + Cloud Config
**Goal:** Update all documentation, add hub config to `config.yaml` schema, prepare for M-CLOUD-INFRA handoff.
**Duration:** 1 day

**Tasks:**
- Day 5 AM: Add `hub` section documentation to config.yaml reference. Update design doc status to "In Progress". Verify all new endpoints are documented in API reference / Swagger (if applicable).
- Day 5 PM: End-to-end test: start `ailang serve`, run a Claude Code session, verify full telemetry pipeline (HTTP hook → server → observatory → dashboard). Write up any findings or adjustments needed for M-CLOUD-INFRA. Move design doc to implemented if complete.

**Acceptance Criteria:**
- [x] `hub` config section documented in config.yaml
- [x] End-to-end telemetry verified: Claude Code → HTTP hook → observatory → dashboard
- [x] Design doc updated with implementation report
- [x] No regressions in `make test`

**Risks:**
- None significant — documentation/validation milestone

## Success Metrics
- Test coverage: >85% on new handler files
- All existing tests passing: `make test` ✅
- All linting clean: `make lint` ✅
- Dashboard WebSocket live events working ✅
- Zero shell dependencies for telemetry ✅
- Documentation: CLAUDE.md, config.yaml reference updated

## Dependencies
- `ailang serve` must be running for integration testing
- Claude Code must support `type: "http"` hooks (confirmed in current version)
- No dependency on M-CLOUD-INFRA (this is a prerequisite FOR it)

## Open Questions
1. Should SubagentStart/Stop events get their own observatory table or reuse `session_tools`?
2. Should we set `AILANG_HUB_URL=http://127.0.0.1:1957` as a default in `~/.claude/settings.json` or require explicit configuration?
3. For cloud: should auth use bearer tokens or GCP Identity Tokens? (Bearer for now, GCP Identity in M-CLOUD-INFRA)

## Notes
- Velocity estimate is conservative — this is infrastructure work with well-understood patterns
- The existing `handlers_hooks.go` (185 LOC) provides the exact patterns to follow
- HTTP hooks are non-blocking in Claude Code — a server outage won't break sessions
- Dual-mode transition period (M2) is key to safe migration
