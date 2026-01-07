# M-SCRIPT-INVOKE Sprint Plan

## Sprint Summary

| Field | Value |
|-------|-------|
| **Sprint ID** | M-SCRIPT-INVOKE |
| **Design Doc** | [m-script-invoke.md](m-script-invoke.md) |
| **Duration** | 1 day (~6 hours implementation) |
| **Total LOC** | ~500 lines (implementation + tests) |
| **Risk Level** | Low (extends existing patterns) |
| **Dependencies** | M-COORD-GENERIC-WORKFLOWS (completed v0.6.3) |

## Goal

Add `script` invoke type to any agent inbox, enabling deterministic shell script execution with JSON payload variables using the existing messaging infrastructure.

## Current Status Analysis

### Existing Infrastructure (Ready to Extend)

| Component | Status | Notes |
|-----------|--------|-------|
| `InvokeConfig` struct | ✅ Ready | Has `skill`, `agent`, `prompt` types |
| `TaskExecutor` routing | ✅ Ready | Provider selection pattern exists |
| `AgentConfig` | ✅ Ready | Supports workspace, output_markers |
| Output streaming | ✅ Ready | `EventHandler` interface exists |
| Dashboard integration | ✅ Ready | WebSocket broadcasting works |

### Recent Velocity

From CHANGELOG and git history (last 14 days):
- ~1,150 LOC across telemetry, UI, and coordinator features
- Average: ~80-100 LOC/day
- Similar complexity features (M-OTEL, M-TASK-HIERARCHY): 1-2 days each

## Milestone Breakdown

### M1: Core Script Provider (~3 hours, ~250 LOC)

**Goal**: Implement `ScriptProvider` that executes shell scripts with environment variables.

**Tasks**:
1. Extend `InvokeConfig` with script fields (~30 LOC)
2. Implement payload-to-env conversion (~60 LOC)
3. Create `ScriptProvider` struct and `Execute` method (~120 LOC)
4. Add `AILANG_*` context variable injection (~20 LOC)
5. Unit tests for provider (~100 LOC)

**Files**:
- `internal/coordinator/agent_registry.go` (+30 lines)
- `internal/coordinator/provider_script.go` (~200 lines, new)
- `internal/coordinator/provider_script_test.go` (~100 lines, new)

**Acceptance Criteria**:
- [ ] `InvokeConfig` accepts `type: script` with `command`, `env_from_payload`, `timeout`
- [ ] JSON payload converts to UPPER_SNAKE_CASE env vars
- [ ] Nested JSON flattens: `{"db": {"host": "x"}}` → `DB_HOST=x`
- [ ] `AILANG_TASK_ID`, `AILANG_WORKSPACE` injected automatically
- [ ] Script exit code determines success/failure
- [ ] Timeout kills long-running scripts

### M2: Task Executor Integration (~1.5 hours, ~100 LOC)

**Goal**: Route script-type agents through ScriptProvider instead of AI providers.

**Tasks**:
1. Register `ScriptProvider` in `TaskExecutor` (~20 LOC)
2. Update `selectProvider` to check invoke type (~30 LOC)
3. Pass `InvokeConfig` to provider via `ExecuteOptions` (~20 LOC)
4. Integration tests (~50 LOC)

**Files**:
- `internal/coordinator/task_executor.go` (+50 lines)
- `internal/coordinator/provider.go` (+10 lines - add InvokeConfig to options)
- `internal/coordinator/task_executor_test.go` (+50 lines)

**Acceptance Criteria**:
- [ ] Script agents route to `ScriptProvider`
- [ ] AI agents still route to Claude/Gemini
- [ ] Mixed pipelines work (AI → Script → AI)

### M3: Output Streaming & Markers (~1 hour, ~80 LOC)

**Goal**: Stream script output to dashboard and parse output markers.

**Tasks**:
1. Implement `streamWriter` for stdout/stderr (~40 LOC)
2. Connect to `EventHandler` interface (~20 LOC)
3. Verify output marker parsing (reuses existing code) (~10 LOC)
4. Test real-time streaming (~30 LOC)

**Files**:
- `internal/coordinator/provider_script.go` (+50 lines)
- `internal/coordinator/stream_writer.go` (~40 lines, new)

**Acceptance Criteria**:
- [ ] Script stdout streams to dashboard in real-time
- [ ] Output markers (e.g., `RESULT:`) parsed same as AI output
- [ ] Cost shows $0.00 in dashboard

### M4: Documentation & Examples (~0.5 hours, ~70 LOC)

**Goal**: Document script invoke type and provide example configurations.

**Tasks**:
1. Update CLAUDE.md coordinator section (~30 lines)
2. Add example script configs to sample config (~20 lines)
3. Add inline code comments (~20 lines)

**Files**:
- `CLAUDE.md` (+30 lines)
- `internal/coordinator/agent_config.go` - update `SampleAgentConfig()` (+20 lines)

**Acceptance Criteria**:
- [ ] CLAUDE.md documents script invoke type
- [ ] Sample config shows script agent example
- [ ] Code has clear comments for new fields

## Implementation Schedule

| Phase | Milestone | Est. Time | Cumulative |
|-------|-----------|-----------|------------|
| 1 | M1: Core Script Provider | 3h | 3h |
| 2 | M2: Task Executor Integration | 1.5h | 4.5h |
| 3 | M3: Output Streaming | 1h | 5.5h |
| 4 | M4: Documentation | 0.5h | 6h |

## Success Metrics

- [ ] All unit tests passing (`go test ./internal/coordinator/...`)
- [ ] Script agent executes with JSON → env var conversion
- [ ] Dashboard shows script execution with $0.00 cost
- [ ] Mixed AI + Script pipeline works end-to-end
- [ ] Documentation updated

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Shell injection | Low | High | Sanitize env var keys (alphanumeric only) |
| Timeout race conditions | Low | Medium | Use context cancellation |
| Platform differences | Low | Low | Use `/bin/sh` default, test on macOS |

## Open Questions

None - design is well-defined and extends existing patterns.

## Files Summary

| File | Action | LOC |
|------|--------|-----|
| `internal/coordinator/agent_registry.go` | Modify | +30 |
| `internal/coordinator/provider_script.go` | Create | ~250 |
| `internal/coordinator/provider_script_test.go` | Create | ~100 |
| `internal/coordinator/task_executor.go` | Modify | +50 |
| `internal/coordinator/provider.go` | Modify | +10 |
| `internal/coordinator/stream_writer.go` | Create | ~40 |
| `CLAUDE.md` | Modify | +30 |
| **Total** | | **~510** |
