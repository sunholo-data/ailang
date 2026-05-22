# M-EVAL-LOCAL-OBSERVABILITY — Reliable local eval observability

**Status**: Planned
**Target**: v0.22.0
**Priority**: P1 — Blocks the 24/7 local eval rotation from being trustworthy
**Estimated**: 2–3 days
**Dependencies**: None (companion to [M-EVAL-LOCAL-OLLAMA](../m-eval-local-ollama.md))

## TL;DR

Sibling milestone to [M-EVAL-LOCAL-OLLAMA](../m-eval-local-ollama.md). That doc is about making local Ollama models *run* correctly. This doc is about making local Ollama eval runs *observable* — specifically about telling "model is thinking hard" from "model is genuinely stuck" without `ps` + `tail -f` gymnastics.

The investigation on 2026-05-22 showed the eval-suite has zero per-turn live visibility today, even though all the necessary OTEL plumbing exists. Three concrete bugs gate this:

1. **FK constraint drops opencode spans** at the OTLP receiver — when no parent `task` row exists, the receiver rejects the entire batch instead of inserting with `task_id=NULL`.
2. **No benchmark-id label on chain stages** — `ailang chains view` shows "4 stages running" but not which benchmark each is on.
3. **No `ailang chains live` command** — even with spans landing, there's no single-page live view that joins chain + opencode-session + ollama-runtime state.

Fix all three and we have a reliable monitoring story for the 24/7 rotation.

## Problem Statement

**The investigation:** while running M-EVAL-LOCAL-OLLAMA's smoke-tier experiments (8 runs across 2026-05-22), the only reliable in-flight signals we had were:

| Signal | Reliability | What it tells us |
|---|---|---|
| `ollama runner` process CPU% | ✓ good | "GPU is doing work" (20–30%) vs "stuck" (<1%) |
| `pgrep -fc 'opencode run'` | ✓ good | Number of active sessions |
| Result file count in output dir | ✓ good | Number of completed benchmarks |
| opencode.db message count | ⚠ misleading | Long thinking phases look identical to "stuck" — no DB write during reasoning |
| `ailang chains view <id>` | ⚠ limited | High-level "N stages running" but no benchmark labels |
| Observatory spans (`ailang trace list`) | ✗ broken | Spans get sent, **dropped by FK constraint**, never persist |

The most critical gap: gemma4:26b is a **thinking model**. It can spend 5–10 minutes in pure internal reasoning before producing any visible output. Without per-turn spans, this is indistinguishable from a genuinely-stuck process, and we've spent real human time on the "is it stuck?" question multiple times today.

**The good news:** opencode (the npm CLI) already emits OTEL spans via `@effect/opentelemetry`. AILANG's opencode executor already emits `opencode.execute` + `opencode.step` spans. The plumbing exists; it just doesn't land because of bug #1.

**The bad news:** without all three fixes, we cannot stand up a reliable 24/7 rotation. We'll keep being unable to triage "thinking" vs "stuck."

## Goals

**Primary Goal:** Make `ailang chains live <id>` (new command) display, with ≤5 second staleness:
- Which benchmark each concurrent stage is running
- How many agent turns each stage has completed
- Per-turn token count and timing
- Whether the model is *generating* (ollama runner CPU >5%) or *idle*

**Success Metrics:**

1. After enabling `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957`, observatory.db span count grows during an eval-suite run (currently: stays at 0).
2. `ailang trace list --hours 1` (or local equivalent that doesn't require GCP) shows per-step spans for in-flight runs.
3. `ailang chains live <id>` displays the columns above and refreshes every 2–5 seconds.
4. We can distinguish thinking-hard from stuck on a running benchmark within 30 seconds.
5. The 24/7 rotation logs include per-benchmark span timing that can answer "did this run thrash" without re-reading 300K-token transcripts.

## Findings from 2026-05-22 Investigation

### Finding 1: opencode emits per-step spans (already!)

[opencode.go:263](../../../internal/executor/opencode/opencode.go#L263):

```go
case "step_start":
    numSteps++
    _, stepSpan = telemetry.StartSpan(ctx, opencodeTracer, "opencode.step",
        trace.WithAttributes(
            attribute.Int("opencode.step_num", numSteps),
            attribute.String("opencode.message_id", ev.Part.MessageID),
        ),
    )
```

These spans exist. They have step number, message ID, parent (`opencode.execute`). If they landed in observatory.db, per-turn live progress would be trivial to query.

Additionally, the opencode npm package itself emits OTEL spans via `@effect/opentelemetry` (service name `opencode`, SDK `nodejs`). We observed these in the server log:

- `SyncEvent.run`
- `Truncate.cleanup`
- Plus more during a full session (need to capture; FK rejects them all today)

These are *bonus* visibility we'd get for free if FK constraint stopped rejecting them.

### Finding 2: The FK-constraint bug at the OTLP receiver

`~/.ailang/logs/server.log` excerpt while running `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957 ailang eval-suite ...`:

```
observatory: received traces request with 1 resource spans
observatory: WARNING: span 735e9bdf9e74ef569f75cdf7 has task_id=eval-1779474162154108000 but task not found
observatory: WARNING: span 735e9bdf9e74ef569f75cdf7 has assignment_id=aa_1779474162154108000 but assignment not found
observatory: storing span name=Truncate.cleanup, id=735e9bdf9e74ef569f75cdf7
observatory: failed to process resource spans: store span ...: insert span: FOREIGN KEY constraint failed
```

The receiver:
1. Receives spans with `task_id` and `assignment_id` attributes set by AILANG
2. Detects the parent rows don't exist (`task not found`, `assignment not found`)
3. **Logs a warning but proceeds to INSERT anyway** with the FK references intact
4. SQLite rejects the INSERT due to the foreign key constraint
5. Span is dropped; nothing persists

The schema (`internal/observatory/schema.sql`):

```sql
CREATE TABLE spans (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    parent_span_id TEXT,
    task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL,
    agent_assignment_id TEXT REFERENCES agent_assignments(id) ON DELETE SET NULL,
    ...
);
```

The `REFERENCES` constraint is the right pattern for relational integrity, but the INSERT path doesn't handle missing parents gracefully. The detection logic exists (the warning fires); it just doesn't translate "warning detected" into "set the offending column to NULL before insert."

**The fix is small** (~10 LOC in [backend_sqlite.go:140-200](../../../internal/observatory/backend_sqlite.go#L140-L200)):

```go
// In createSpanWithAggregation, before INSERT:
if span.TaskID != "" {
    if !b.taskExists(ctx, span.TaskID) {
        log.Printf("observatory: task %s not found, storing span with task_id=NULL", span.TaskID)
        span.TaskID = ""  // Triggers NULL via the existing taskID interface{} pattern
    }
}
// Same for AgentAssignmentID
```

Even simpler: drop the FK constraint to `ON DELETE SET NULL` semantics during INSERT too (i.e., allow orphan references at insert time, clean them up later). Or remove the FK entirely — the data model treats task_id as a soft reference everywhere else (see the `M-CHAINS-SIMPLIFY` comment in the schema about chain_id being a "soft reference").

### Finding 3: Eval-suite doesn't register tasks/assignments

The eval-suite path generates `task_id=eval-<timestamp>` for telemetry attribution but never inserts a row into the `tasks` table. The coordinator path does, which is why coordinator-driven spans land successfully but eval-suite spans don't.

Two solution shapes:

**(A) Have eval-suite create a stub task row** at run start. Pros: keeps FK semantics intact. Cons: adds eval/coordinator schema entanglement; eval task rows are noise in the tasks UI.

**(B) Drop the FK or accept orphans on insert.** Pros: cleaner separation (eval and coordinator are different concerns). Cons: tasks table can have spans pointing at non-existent IDs.

**Recommendation: (B).** Eval-suite tasks are ephemeral; coordinator tasks are durable. Treating eval task_ids as soft labels is the right model.

### Finding 4: Chain stages have no benchmark label

When 4 stages are running concurrently, `ailang chains view <id>` shows:

```
Stages:
  1. eval-agent [running]
  2. eval-agent [running]
  3. eval-agent [running]
  4. eval-agent [running]
```

We need:

```
Stages:
  1. eval-agent fizzbuzz        [running, turn 12, 47K tokens, 18m]
  2. eval-agent adt_option      [running, turn 8,  31K tokens, 18m]
  3. eval-agent balanced_parens [running, turn 0, 0 tokens (TTFT), 18m]  ← stuck!
  4. eval-agent recursion_fib   [running, turn 14, 52K tokens, 18m]
```

The chain stage attributes table can hold arbitrary key-value pairs already. We just need to record `benchmark_id` when the stage is registered. ~10 LOC change.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix shape: insert with NULL on missing parent (B) vs auto-create task row (A) | Determines whether tasks table accumulates eval noise | human | design | med |
| Whether to start `ailang server` at boot via launchd | Required for OTLP receiver to be available 24/7 | human | design | low |
| Should `OTEL_EXPORTER_OTLP_ENDPOINT` be set automatically in eval-suite invocations, or always required by env? | Affects user experience and rotation scripting | human | design | low |
| `ailang chains live <id>` UI format: TUI (ncurses-style) vs auto-refreshing text vs JSON | Implementation cost varies | agent | design | med |

### Design Freeze

Before implementation:

- [ ] Decide A vs B for the FK fix
- [ ] Decide on `ailang chains live` UI format
- [ ] Decide whether eval-suite auto-detects and uses local OTLP receiver if running

### Deferred Decisions

- Whether to also fix the `claude-hooks: PreToolUse error: failed to insert tool start: FOREIGN KEY constraint failed` (same class of bug, but Claude-Code hooks not eval-suite).
- Whether observatory.db should grow a `eval_runs` table (separate from `tasks`) to give eval spans a proper parent surface.
- Whether to emit additional eval-specific spans (e.g., `eval.benchmark`, `eval.session`) that wrap the existing `opencode.execute` for clearer query semantics.

## Solution Design

### Overview

Three small fixes + one new CLI command:

1. **FK-tolerant span insert** (~15 LOC): if `task_id` or `assignment_id` refers to a missing row, set to NULL and proceed.
2. **Eval-suite chain-stage labeling** (~10 LOC): record `benchmark_id` as a stage attribute when registering an eval-agent stage.
3. **launchd plist for `ailang server`** (~50 lines XML): auto-start at boot.
4. **`ailang chains live <id>`** (~150 LOC new subcommand): refreshing text view that joins chain_stages + spans + opencode.db.

### Implementation Plan

**Phase 1 (1 day): FK fix + basic span landing**

- [ ] Fix `createSpanWithAggregation` to set NULL on missing parents
- [ ] Verify with `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957 make eval-smoke MODELS=opencode-gemma4-26b -agent ...` — spans should land
- [ ] Add a regression test: insert span with non-existent task_id, expect NULL not error
- [ ] Document in `docs/docs/guides/evaluation/local-ollama.md`

**Phase 2 (~half day): chain-stage labeling**

- [ ] Pass `benchmark_id` into the chain stage when each eval-agent goroutine starts
- [ ] Update `ailang chains view` formatting to surface the attribute
- [ ] Verify diagnose no longer flags "No session ID recorded" — or fix that too

**Phase 3 (1 day): `ailang chains live <id>`**

- [ ] New CLI subcommand
- [ ] Refresh loop (default every 3s)
- [ ] Single-page output: stages × {benchmark, turn, tokens, age, model_busy?}
- [ ] Exits cleanly on chain completion

**Phase 4 (1 hour): launchd plist**

- [ ] Write plist with `RunAtLoad: true`, `KeepAlive: true`
- [ ] Test with `launchctl load`/`launchctl start`/reboot
- [ ] Document in the local-ollama eval guide

### Files to Modify

| File | LOC delta | Why |
|---|---|---|
| `internal/observatory/backend_sqlite.go` | +15 | FK-tolerant insert |
| `internal/observatory/store.go` | +5 | `taskExists()` helper (or use existing query) |
| `internal/eval_harness/agent_runner_multi.go` | +5 | Set benchmark_id on chain stage |
| `cmd/ailang/chains_view.go` | +10 | Surface benchmark_id in output |
| `cmd/ailang/chains_live.go` | +150 (new) | New subcommand |
| `cmd/ailang/main.go` | +3 | Register chains-live subcommand |
| `~/Library/LaunchAgents/dev.ailang.server.plist` | +50 (new) | Boot-time server start |
| `docs/docs/guides/evaluation/local-ollama.md` | +80 | User-facing guide |
| `internal/observatory/backend_sqlite_test.go` | +30 | Regression test for FK fix |

**Total estimated impact:** ~350 LOC.

## Conflict Surface

Not a parser/typechecker change, so no language-semantics conflict. But:

**observatory.db schema:** the FK fix changes the *behavior* of span inserts, not the schema. Existing spans (currently 0) won't be affected. Future spans get NULL task_id when the parent doesn't exist. The `ailang chains view` joins still work — they LEFT JOIN tasks on task_id, so NULL is handled.

**`ailang chains` CLI surface:** adding `chains live` doesn't affect existing subcommands. The `view` formatting change (showing benchmark_id) is additive.

**Programs that must keep working:**
1. Coordinator-driven runs (task_id always populated): unchanged.
2. Eval-suite runs without OTLP set: unchanged (no spans emitted, no FK to fail).
3. Existing `ailang chains view`/`tree`/`diagnose` queries: still work; benchmark_id is an additional column when present.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No determinism impact |
| A2: Replayability | +1 | Captured spans enable better post-hoc replay analysis |
| A3: Effect Legibility | +1 | Eval IO/network effects become visible in observatory.db |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type system unchanged |
| A6: Safe Concurrency | +1 | Diagnosing concurrent-eval contention requires this observability |
| A7: Machines First | +1 | Live span data is machine-readable; supports automated regression detection |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Per-turn token + timing data lands in observatory.db, queryable |
| A10: Composability | 0 | New CLI subcommand composes with existing chains tooling |
| A11: Structured Failure | +1 | "FK failed → drop span" is replaced with "FK missing → store with NULL"; structured graceful degradation |
| A12: System Boundary | +1 | OTLP receiver is the system boundary; this fix makes it forgiving of upstream variance |

**Net Score: +7** → **Proceed.**

**Hard violation check:** None. No `-1` on A1/A3/A4/A7.

## Success Criteria

- [ ] `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957 make eval-smoke MODELS=opencode-gemma4-26b -agent ...` produces spans in `~/.ailang/state/observatory.db`
- [ ] Span count after a 17-benchmark smoke tier: ≥100 (opencode emits multiple per step × multiple steps per benchmark)
- [ ] `ailang chains view <id>` shows benchmark name per stage
- [ ] `ailang chains live <id>` displays a working live view
- [ ] `make services-start` is robust enough to run from launchd at boot
- [ ] Regression test added: insert with missing FK should succeed with NULL
- [ ] CHANGELOG entry for v0.22.0

## Timeline

| Day | Work |
|---|---|
| 2026-05-23 | Phase 1: FK fix + verify spans land |
| 2026-05-24 | Phase 2 + 3: stage labeling + `ailang chains live` |
| 2026-05-25 | Phase 4 + docs + tests |
| 2026-05-26 | Buffer / merge / release in v0.22.0 |

## Related Documents

- [M-EVAL-LOCAL-OLLAMA](../m-eval-local-ollama.md) — the operational milestone this observability work supports
- [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md) — uniform executor contract; this work doesn't change it
- [`internal/observatory/schema.sql`](../../../internal/observatory/schema.sql) — the FK constraints in question
- CLAUDE.md `coordinator.md` rule — notes that TRACEPARENT is NOT propagated to subprocesses; this is fine for us because the spans we want come from the AILANG opencode executor process, not the opencode subprocess itself

## Open Questions

1. Should we also surface opencode-npm-emitted spans (e.g. `SyncEvent.run`, `Truncate.cleanup`) in the `ailang chains live` view, or filter them out as noise? Initial inclination: filter by `service.name=opencode` and only show our `executor.opencode.*` spans plus opencode's `step_*` events.
2. Once spans land, the 24/7 rotation generates lots of them. What's the disk pressure? observatory.db could grow ~MB/day. Worth thinking about retention policy (auto-delete spans older than 30 days?).
3. Does the existing `ailang trace list` work locally without GCP if local spans exist? Right now it errors with "GOOGLE_CLOUD_PROJECT not set" even when there's no need for cloud. Worth a flag like `--local`.
