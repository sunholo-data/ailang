# Sprint Plan: M-OTEL-CROSS-PROCESS

## Summary

Enable end-to-end distributed tracing from AILANG coordinator through CLI executors to `ailang run` subprocesses, using W3C TRACEPARENT propagation via environment variables.

**Duration:** 1.5 days (12 hours)
**Dependencies:** Telemetry infrastructure (v0.6.1 - complete), M-OTEL-EXTENDED M1 (complete)
**Risk Level:** Low (additive changes, proven patterns)

## Current Status Analysis

### Completed Recently
- ✅ M-OTEL: Core telemetry infrastructure (~500 LOC) in v0.6.2
- ✅ M-OTEL-ENHANCED-TRACING-DX: Helper functions (~260 LOC) in v0.6.3
- ✅ M-OTEL-EXTENDED M1: Compiler pipeline instrumentation (~120 LOC)

### Velocity
- Recent telemetry work: ~200-250 LOC/day
- Estimated capacity: ~400 LOC for this sprint
- Infrastructure already exists - this is extension work

### Key Insight from Design

Environment variables are inherited by child processes. Even if Claude/Gemini CLI don't explicitly use TRACEPARENT, they **pass it through** to `ailang run` subprocesses. This means:
- Layer 1 (Executor → CLI): We control ✅
- Layer 2 (CLI internal spans): External dependency ❓
- Layer 3 (CLI → ailang run): Works via env inheritance ✅

## Proposed Milestones

### M1: Context Propagation Helpers (3 hours)

**Goal:** Create telemetry helpers to inject/extract W3C trace context from environment variables.

**Estimated:** 80 LOC implementation + 60 LOC tests = 140 LOC

**Tasks:**
- Create `internal/telemetry/context_propagation.go`
  - `InjectTraceContext(ctx, env []string) []string` - Adds TRACEPARENT/TRACESTATE to env
  - `InjectCorrelationIDs(env, taskID, sessionID) []string` - Adds AILANG_TASK_ID, AILANG_SESSION_ID
  - `ExtractTraceContext(ctx) context.Context` - Reads TRACEPARENT from os.Environ()
  - `ExtractCorrelationIDs() (taskID, sessionID string)` - Reads from env
- Create `internal/telemetry/context_propagation_test.go`
  - Test inject/extract round-trip
  - Test with missing env vars (graceful fallback)
  - Test W3C format parsing

**Acceptance Criteria:**
- [ ] `InjectTraceContext` produces valid W3C TRACEPARENT format
- [ ] `ExtractTraceContext` correctly parses TRACEPARENT from env
- [ ] Graceful no-op when trace context is missing
- [ ] Unit tests with 100% coverage
- [ ] `make lint` passes

**Files:**
| File | Change | LOC |
|------|--------|-----|
| `internal/telemetry/context_propagation.go` | New file | +80 |
| `internal/telemetry/context_propagation_test.go` | New file | +60 |

---

### M2: Executor Integration (2 hours)

**Goal:** Inject trace context and correlation IDs into Claude/Gemini executor subprocess environments.

**Estimated:** 20 LOC (10 per executor)

**Tasks:**
- Update `internal/executor/claude/claude.go`:
  - Import telemetry package
  - Call `telemetry.InjectTraceContext(ctx, env)` before `cmd.Env = env`
  - Call `telemetry.InjectCorrelationIDs(env, task.ID, sessionID)`
- Update `internal/executor/gemini/gemini.go`:
  - Same changes as Claude executor
- Test manually: Run coordinator task, verify TRACEPARENT in subprocess

**Acceptance Criteria:**
- [ ] Claude executor injects TRACEPARENT into subprocess env
- [ ] Gemini executor injects TRACEPARENT into subprocess env
- [ ] Correlation IDs (task_id, session_id) passed to subprocess
- [ ] Existing executor tests still pass
- [ ] `make test` passes

**Files:**
| File | Change | LOC |
|------|--------|-----|
| `internal/executor/claude/claude.go` | Add context injection | +10 |
| `internal/executor/gemini/gemini.go` | Add context injection | +10 |

---

### M3: CLI Context Extraction (4 hours)

**Goal:** Enable `ailang run`, `ailang check`, `ailang eval-suite`, and `ailang repl` to extract parent trace context from environment.

**Estimated:** 100 LOC (25 per command)

**Tasks:**
- Add CLI flags to each command:
  - `--trace-parent` - W3C traceparent header (explicit override)
  - `--trace-id` - Trace ID (alternative to --trace-parent)
  - `--parent-span` - Parent span ID (requires --trace-id)
  - `--task-id` - AILANG task ID for correlation
  - `--session-id` - Session ID for correlation
- Create shared helper `extractTraceContext(cmd *cobra.Command) context.Context`:
  - CLI flags take precedence over environment variables
  - Falls back to `telemetry.ExtractTraceContext()`
- Update commands to use extracted context for root spans
- Record correlation IDs as span attributes

**Acceptance Criteria:**
- [ ] `ailang run` creates child span under parent trace when TRACEPARENT set
- [ ] `ailang check` creates child span under parent trace
- [ ] `ailang eval-suite` creates child span under parent trace
- [ ] `ailang repl` creates child span under parent trace
- [ ] CLI flags override environment variables
- [ ] Correlation IDs recorded as span attributes
- [ ] Commands work normally without trace context (no regression)

**Files:**
| File | Change | LOC |
|------|--------|-----|
| `cmd/ailang/run.go` | Extract context, add flags | +25 |
| `cmd/ailang/check.go` | Extract context, add flags | +25 |
| `cmd/ailang/eval_suite.go` | Extract context, add flags | +25 |
| `cmd/ailang/repl.go` | Extract context, add flags | +25 |

---

### M4: Verification & Documentation (3 hours)

**Goal:** Verify end-to-end trace linking works and document the feature.

**Estimated:** 60 LOC documentation + 40 LOC integration test = 100 LOC

**Tasks:**
- Create integration test script:
  - Start coordinator with telemetry enabled
  - Send task that triggers `ailang run`
  - Verify traces appear in GCP Cloud Trace
  - Verify parent-child relationship (executor → ailang run)
- Test CLI tools:
  - Test Claude Code with TRACEPARENT in env
  - Test Gemini CLI with TRACEPARENT in env
  - Document which scenario (A/B/C) applies
- Update `docs/docs/guides/telemetry.md`:
  - Add "Cross-Process Trace Linking" section
  - Document env vars (TRACEPARENT, AILANG_TASK_ID, AILANG_SESSION_ID)
  - Document CLI flags
  - Add CI/CD examples (GitHub Actions, Cloud Build)

**Acceptance Criteria:**
- [ ] End-to-end test passes: coordinator → executor → ailang run linked
- [ ] CLI tool behavior documented (scenario A/B/C)
- [ ] Telemetry guide updated with cross-process linking
- [ ] CI/CD examples included

**Files:**
| File | Change | LOC |
|------|--------|-----|
| `docs/docs/guides/telemetry.md` | Add cross-process section | +60 |
| `scripts/test_trace_propagation.sh` | Integration test script | +40 |

---

## Day-by-Day Schedule

### Day 1 (8 hours)

| Time | Milestone | Task |
|------|-----------|------|
| Morning (3h) | M1 | Create context_propagation.go + tests |
| Midday (2h) | M2 | Update Claude + Gemini executors |
| Afternoon (3h) | M3 (partial) | Add CLI flags to run.go, check.go |

**Day 1 Deliverables:**
- `internal/telemetry/context_propagation.go` complete
- Both executors injecting trace context
- `ailang run` and `ailang check` extracting context

### Day 2 (4 hours)

| Time | Milestone | Task |
|------|-----------|------|
| Morning (1h) | M3 (finish) | Add CLI flags to eval_suite.go, repl.go |
| Midday (2h) | M4 | Integration testing, CLI tool verification |
| Afternoon (1h) | M4 | Documentation update |

**Day 2 Deliverables:**
- All CLI commands with trace context extraction
- End-to-end verification complete
- Documentation updated

---

## Success Metrics

- [ ] Unit test coverage: >90% for context_propagation.go
- [ ] All existing tests passing: `make test`
- [ ] Linting clean: `make lint`
- [ ] End-to-end trace linking verified in GCP Cloud Trace
- [ ] Documentation: telemetry.md updated with cross-process section

## Dependencies

- ✅ Telemetry infrastructure (v0.6.1) - Complete
- ✅ W3C Trace Context propagators registered in otel.go - Complete
- ✅ Executor spans exist (claude.execute, gemini.execute) - Complete
- ⏳ M-OTEL-EXTENDED M1 (compiler pipeline spans) - Complete

## Open Questions

1. **File feature requests to CLI tools?**
   - Recommendation: Yes, after verifying their behavior
   - Nice-to-have: CLI internal spans in our trace (Layer 2)

2. **REPL session handling?**
   - Session span should use extracted trace context
   - Each input span is child of session span

## Notes

- All changes are additive - no behavioral changes to existing functionality
- CLI tools (Claude Code, Gemini CLI) likely ignore TRACEPARENT for their own spans, but env vars pass through to `ailang run` via Unix process inheritance
- Correlation IDs (task_id, session_id) provide fallback linking if env sanitization occurs
- Zero overhead when trace context is not present (graceful no-op)

## Total Estimates

| Milestone | LOC | Hours |
|-----------|-----|-------|
| M1: Context Propagation Helpers | 140 | 3 |
| M2: Executor Integration | 20 | 2 |
| M3: CLI Context Extraction | 100 | 4 |
| M4: Verification & Documentation | 100 | 3 |
| **Total** | **360** | **12** |
