# M-OTEL-EXTENDED Sprint Plan

**Sprint ID:** M-OTEL-EXTENDED
**Design Doc:** [m-otel-extended-instrumentation.md](m-otel-extended-instrumentation.md)
**Target:** v0.6.3
**Duration:** 3 days (~18 hours remaining)
**Risk Level:** Low

## Goal

Complete OpenTelemetry instrumentation across AILANG CLI commands: Eval Harness, Extended Messages, REPL, and Check command.

## Current Status

- **M1: Compiler Pipeline** - ✅ COMPLETE
- **M2: Eval Harness** - 🔄 IN PROGRESS
- **M3: Extended Messages** - Planned
- **M4: REPL Command** - Planned (NEW)
- **M5: Check Command** - Planned (NEW)

## Velocity Analysis

- Recent telemetry work: ~150 LOC/day
- Estimated capacity for 3 days: ~450 LOC
- Sprint estimate: ~370 LOC (within capacity)

## Milestones

### M1: Compiler Pipeline Instrumentation ✅ COMPLETE

**Status:** Already implemented with spans for parse, elaborate, typecheck, validate, lower.

### M2: Eval Harness Instrumentation (~2 hours, ~100 LOC)

**Description:** Add spans for benchmark suite execution and individual benchmarks.

**Files to modify:**
- `internal/eval_harness/runner.go` (+60 LOC)
- `internal/eval_harness/model_runner.go` (+40 LOC)

**Tasks:**
- [ ] Add tracer import and package-level tracer
- [ ] Add `eval.suite` span with benchmark/model counts
- [ ] Add `eval.benchmark` span with ID and difficulty
- [ ] Add `eval.model_call` span with model/tokens
- [ ] Add `eval.validation` span with pass/fail status
- [ ] Record token counts and costs on spans

**Acceptance Criteria:**
- [ ] Eval runs produce spans visible in GCP/Jaeger
- [ ] Each benchmark is a child span of suite
- [ ] Token/cost data visible in span attributes
- [ ] Existing eval tests pass

### M3: Extended Message System Instrumentation (~1.5 hours, ~60 LOC)

**Description:** Complete messaging observability (send/search already exist, add list/read/ack).

**Files to modify:**
- `internal/messages/store.go` (+40 LOC)
- `internal/messages/github.go` (+20 LOC)

**Tasks:**
- [ ] Add `messages.list` span with query.inbox, query.status, results.count
- [ ] Add `messages.read` span with message.id, message.inbox
- [ ] Add `messages.ack` span with message.id, message.status
- [ ] Add `messages.unack` span with message.id
- [ ] Add `messages.github_sync` span with github.repo, issues.imported
- [ ] Add `messages.cleanup` span with deleted.count

**Acceptance Criteria:**
- [ ] `messages.list` traced with query parameters
- [ ] `messages.read` traced with message ID
- [ ] `messages.ack`/`unack` traced
- [ ] `messages.github_sync` traced with import counts

---

### M4: REPL Command Instrumentation (~2 hours, ~90 LOC)

**Description:** Enable post-hoc analysis of REPL debugging sessions.

**Files to modify:**
- `cmd/ailang/repl.go` (+30 LOC)
- `internal/repl/repl.go` (+40 LOC)
- `internal/repl/repl_eval.go` (+20 LOC)

**Span Hierarchy:**
```
repl.session (session.id, duration_ms)
├── repl.input (input.type=expression, input.text)
│   └── compile.pipeline (from M1)
└── repl.input (input.type=command)
```

**Tasks:**
- [ ] Initialize telemetry in cmd/ailang/repl.go with service "ailang-repl"
- [ ] Create `repl.session` span that lives for entire session duration
- [ ] Add `repl.input` child spans for each user input line
- [ ] Add `repl.eval` spans for expression evaluation results
- [ ] Ensure compilation spans (M1) nest correctly under input spans

**Acceptance Criteria:**
- [ ] `repl.session` spans capture full interactive sessions
- [ ] `repl.input` spans show each user input
- [ ] Compilation phases nest correctly under input spans
- [ ] Session duration and input count captured

---

### M5: Check Command Instrumentation (~1.5 hours, ~40 LOC)

**Description:** Enable type-check performance monitoring and regression detection.

**Files to modify:**
- `cmd/ailang/check.go` (+40 LOC)

**Span Hierarchy:**
```
ailang.check (file.path, file.count)
├── compile.pipeline (from M1)
└── check.result (passed, errors.count)
```

**Tasks:**
- [ ] Initialize telemetry in cmd/ailang/check.go with service "ailang-check"
- [ ] Create `ailang.check` root span with file.path, timeout_ms
- [ ] Add `check.result` span with passed, errors.count, warnings.count
- [ ] Ensure compilation phases from M1 nest as children

**Acceptance Criteria:**
- [ ] `ailang.check` root span with file path
- [ ] Compilation phases nest as children
- [ ] `check.result` shows pass/fail with error counts
- [ ] Works with `--timeout` flag

---

### M6: Documentation Update (~30 min, ~80 LOC)

**Description:** Update telemetry docs with all new instrumented components.

**Files to modify:**
- `docs/docs/guides/telemetry.md` (+80 LOC)

**Tasks:**
- [ ] Add Eval Harness section with span names/attributes
- [ ] Add Extended Messages section
- [ ] Add REPL section with session hierarchy example
- [ ] Add Check Command section
- [ ] Update span reference table

**Acceptance Criteria:**
- [ ] Docs build without errors
- [ ] All new spans documented with attributes

## Day-by-Day Breakdown

| Day | Milestone | Hours | LOC |
|-----|-----------|-------|-----|
| 1 | M2: Eval Harness | 2h | 100 |
| 2 | M3: Messages + M5: Check | 1.5h + 1.5h | 100 |
| 3 | M4: REPL + M6: Docs | 2h + 0.5h | 170 |

## Success Metrics

- [ ] All 5 remaining milestones complete (M2-M6)
- [ ] Tests pass: `make test`
- [ ] Lint clean: `make lint`
- [ ] ~370 LOC added total
- [ ] Zero performance impact when telemetry disabled
- [ ] Spans visible in GCP Cloud Trace

## Dependencies

- Telemetry infrastructure (completed in v0.6.1) ✅
- M1 Compiler Pipeline spans ✅
- Global TracerProvider pattern established ✅

## Notes

- Uses same pattern as existing AI provider instrumentation
- No-op overhead measured at <5ns per span when disabled
- All changes are additive (no behavioral changes)
- REPL input text truncated to 200 chars in span attributes
