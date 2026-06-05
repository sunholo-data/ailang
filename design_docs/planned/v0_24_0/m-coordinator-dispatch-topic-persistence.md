# M-COORD-DISPATCH-TOPIC-PERSISTENCE: Persist & Validate Task Topic at Dispatch

**Status**: Planned
**Target**: v0.24.0
**Priority**: P0 (High — silent, unrecoverable task failures burning 20 min of executor wall-clock each)
**Estimated**: ~3 days (phased — validation gate ships independently of persistence)
**Dependencies**: M-COORD-MULTI-HOST-WORKERS (v0.22.0, shipped), M-MSG-AGENT-MESSAGING (v0.5.6, shipped)

> **How this doc came to exist:** it documents a defect discovered *while a task was failing on it*. Task `task-1dfb8f3b` was dispatched as a `design-doc-creator` task whose prompt contained only a correlation-ID reference and no topic. The original milestone topic for that task is **unrecoverable** (see Problem Statement for the full trace). Rather than fabricate a feature topic, this doc captures the systemic dispatch defect that caused `task-1dfb8f3b` and two sibling tasks to fail — which is exactly what a reviewer of those failed tasks needs. The fix prevents the whole failure class.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is coordinator/orchestration infrastructure (task dispatch, prompt rendering, worker recovery). It does not touch language semantics, so most axioms are neutral; the positives come from making dispatch failures explicit and reconstructable instead of silent timeouts.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language determinism; dispatch rendering becomes *more* deterministic (validated, non-empty) |
| A2: Replayability | +1 | Persisting the topic keyed by correlation-ID makes the message → task → doc chain fully reconstructable; today the topic is lost the moment the executor process exits |
| A3: Effect Legibility | 0 | No new effects |
| A4: Explicit Authority | 0 | No change to authority model |
| A5: Bounded Verification | +1 | Dispatch-time validation is a bounded, local check that rejects a bad task in milliseconds instead of after a 20-minute remote timeout |
| A6: Safe Concurrency | 0 | Reuses existing worktree isolation; no concurrency change |
| A7: Machines First | +1 | A worker that loses its prompt currently has no machine-legible recovery path (the topic is nowhere it can read); this adds one |
| A8: Minimal Syntax | 0 | No language syntax |
| A9: Cost Visibility | +1 | Eliminates the dominant waste mode here — a topic-less task burns a full 20-min executor budget (and its token cost) to produce nothing |
| A10: Composability | +1 | Reuses the existing Firestore-backed message store and `TaskCompletion` envelope rather than adding a new store |
| A11: Structured Failure | +1 | Converts a silent "timeout after 20m0s" into an explicit, early, typed rejection (`ErrEmptyTaskTopic`) routed like any other dispatch error |
| A12: System Boundary | 0 | No new boundary |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No language nondeterminism introduced; rendering is validated
- [x] A3 (Effects): No new implicit effects
- [x] A4 (Authority): No ambient authority added
- [x] A7 (Machines First): Adds a machine-readable recovery path where none existed

## Problem Statement

The coordinator dispatches a task to a worker by rendering an executor **prompt** that is supposed to contain the task's topic/body. When that rendering produces a prompt with **no topic** — only the bookkeeping correlation-ID and a generic "do the task" instruction — the worker has no way to discover what it was asked to do, and **no store it can read to recover it**. The result is not a clean error: the worker (a `design-doc-creator` agent) searches every available source for ~20 minutes and is then killed by the executor timeout.

### Observed Evidence (2026-06-05 batch)

A batch of **7** `design-doc-creator` tasks was dispatched at ~02:55 UTC. Their dispatch correlation IDs (`msg_20260605_0355NN_<taskid>`) order them:

| Dispatch | Task | Outcome | Artifact |
|----------|------|---------|----------|
| `…035526` | task-eeef6785 | completed | `design_docs/planned/v0_25_0/m-list-comprehensions.md` |
| `…035529` | task-91ae3040 | completed | `design_docs/rejected/m-multiline-adt-parser-bug.md` |
| `…035533` | task-0c808c59 | **failed** | — (timeout 20m0s) |
| `…035536` | **task-1dfb8f3b** | **failed** | — (timeout 20m0s) |
| `…035539` | task-49237121 | **failed** | — (timeout 20m0s) |
| `…035542` | task-65639210 | completed | `design_docs/planned/v0_24_0/m-eval-regression-triage-gating.md` |
| `…035545` | task-11e68c7e | completed | `design_docs/planned/v0_24_0/m-prompt-numeric-string-conversions.md` |

Three **consecutive** middle tasks failed identically. The completion notification for each carries:

```json
{"agent_id":"design-doc-creator","branch_name":"coordinator/task-1dfb8f3b",
 "changed_files":null,"error_msg":"executor failed: claude task failed: timeout after 20m0s",
 "status":"failed","task_id":"task-1dfb8f3b"}
```

The session transcript for `task-1dfb8f3b` shows the **entire** prompt the worker received was:

```
msg_20260605_035536_1dfb8f3b

Invoke the design-doc-creator skill to complete this task.

## CRITICAL: Coordinator Output Requirements
... DESIGN_DOC_PATH: <path-to-file> ...
```

There is **no topic** — no feature name, problem statement, or milestone reference — only the correlation ID and the output-marker boilerplate.

### Why it is unrecoverable today

Every avenue a worker could use to recover the topic is a dead end:

1. **The topic is not in the message store.** `ailang messages read msg_20260605_035536_1dfb8f3b` → `Error: no message found`. Critically, the **same is true for the completed siblings** — `ailang messages read msg_20260605_035542_65639210` (the one that produced a doc) also returns "no message found." So the topic was never a stored message for *any* task in the batch; it was interpolated **inline** into the executor prompt at dispatch time. The completed tasks succeeded only because their inline prompt still carried the topic; the failed tasks' inline prompt had been reduced to the bare correlation ID.

2. **The completion notifications don't carry the request body.** The `design-doc-creator` inbox holds only `completion`-type messages (payload = `TaskCompletion` JSON); none preserve the original topic.

3. **The coordinator task store is unreachable from the worker.** `ailang coordinator logs task-1dfb8f3b` and `ailang coordinator list` both fail with:
   `failed to open coordinator database: ... go-sqlite3 requires cgo ... This is a stub` — the distributed binary is built `CGO_ENABLED=0`, so the local SQLite task store cannot be opened on a worker. The full task record (which presumably still holds the topic) exists only on the coordinator host's disk.

4. **Failure is silent and slow.** Because none of the above returns the topic, the worker keeps searching until the **20-minute** executor timeout fires. The only signal that reaches a human is `timeout after 20m0s` — which looks like a hung/slow agent, not a malformed dispatch. The 20 minutes of executor time and tokens are pure waste.

### Root Cause

The dispatch path renders the executor prompt by interpolating the task topic/body, but **(a)** does not validate that the rendered prompt actually contains a non-empty topic before sending it, and **(b)** persists the topic nowhere the worker can later read. When the topic interpolation yields empty (the exact mechanism that emptied these 3 — truncation, a nil body field, or a templating miss — lives in coordinator dispatch code on the host and should be confirmed there), the system dispatches an unrunnable task and only learns of it via timeout.

This is the anti-pattern the project's standards warn against: a failure mode with **no loud signal and no recovery path**.

### Impact

- **Who:** Every coordinator-dispatched task whose prompt is rendered from a topic field — `design-doc-creator`, `sprint-executor`, `website-builder`, etc. The 2026-06-05 batch shows the failure is not hypothetical; 3 of 7 tasks (43%) hit it in a single batch.
- **How significant:** Each occurrence burns a full 20-minute executor slot + token budget to produce nothing, marks the task `failed`, and leaves no diagnostic beyond a generic timeout. A retry re-dispatches the *same* topic-less prompt, so retries fail identically — the task is permanently stuck until a human intervenes.

## Goals

**Primary goal:** No coordinator task is ever dispatched with an empty topic, and any worker that nonetheless receives a topic-less prompt fails **fast and loud** with a recoverable diagnostic — never a 20-minute silent timeout.

**Success metrics:**

1. **Dispatch-time gate:** A task whose rendered prompt has no non-empty topic is rejected at dispatch in <100 ms with a typed error (`ErrEmptyTaskTopic`), not sent to a worker. Target: 0 topic-less prompts reach an executor.
2. **Recovery path:** The task topic is persisted keyed by correlation-ID in a store the worker *can* read (the Firestore-backed message store, which works on `CGO_ENABLED=0` binaries — `ailang messages list` succeeds where `coordinator list` fails). A worker can fetch its own topic by correlation-ID.
3. **Fast worker-side failure:** If a worker detects a design-doc/sprint prompt with no topic and cannot recover it from the store, it exits within ~60 s with a structured error, not at the 20-minute timeout. Target: failure latency drops from 1200 s → <60 s.
4. **Regression guard:** A test reproduces the 2026-06-05 condition (render a dispatch from a task with an empty topic field) and asserts the gate rejects it.

## Solution Design

### Overview

Three independent, individually-shippable layers, defense-in-depth from cheapest/earliest to last-resort:

1. **Validate at dispatch (the gate).** Before the coordinator hands a rendered prompt to an executor, assert the prompt contains a non-empty topic segment. On failure, reject the task with `ErrEmptyTaskTopic` and route it through the normal dispatch-error path (dead-letter / failed-with-reason) instead of dispatching.
2. **Persist the topic where the worker can read it.** At dispatch, write the task topic to the Firestore-backed message store keyed by the correlation-ID the prompt already references. This makes the existing `msg_…` reference in every prompt a *resolvable* pointer rather than dead bookkeeping.
3. **Fail fast at the worker.** Give workers a tiny preamble check: if the prompt is a known agent-task shape but carries no topic, attempt one `messages read <correlation-id>` recovery; if that also yields nothing, exit immediately with a structured error.

### Architecture

```
                    ┌─────────────────────────── coordinator host ───────────────────────────┐
  task record  ───► render prompt ──► [GATE: topic non-empty?] ──no──► reject(ErrEmptyTaskTopic)
  (topic, id)            │                     │ yes                         └─► failed-with-reason
                         │                     ▼
                         └──► persist topic to message store (key = correlation_id)  ◄── Firestore
                                               │
                                               ▼  dispatch prompt (contains correlation_id)
                    ┌──────────────────────────┴─ worker (CGO_ENABLED=0) ──────────────────────┐
                    │  preamble: topic present in prompt? ──yes──► run normally                 │
                    │                 │ no                                                       │
                    │                 ▼                                                          │
                    │   messages read <correlation_id> ──found──► inject topic, run             │
                    │                 │ not found                                                │
                    │                 ▼                                                          │
                    │       exit <60s with ErrTopicUnrecoverable (NOT a 20-min timeout)         │
                    └────────────────────────────────────────────────────────────────────────┘
```

The seam that makes layer 2 work: the prompt **already** embeds the correlation-ID (`msg_20260605_035536_1dfb8f3b`). Today it points to nothing. Persisting the topic under that key turns the existing reference into a working recovery handle with no change to prompt format.

### Why the message store, not the coordinator DB

`ailang messages …` is Firestore-backed and **works on the distributed `CGO_ENABLED=0` binary** (confirmed: `ailang messages list` succeeds in the same worktree where `ailang coordinator list` fails with the cgo-stub error). The SQLite coordinator task store is host-local and cgo-gated, so it can never be the worker's recovery source. Reusing the message store avoids standing up a new store (A10).

### Implementation Plan

**Phase 1 — Dispatch-time validation gate (Day 1, ships alone)**
- In the coordinator dispatch path (where the executor prompt is assembled — `internal/coordinator/`, the prompt-render step that interpolates the task topic), add `validateRenderedPrompt(taskID, prompt, topic)`.
- Define `ErrEmptyTaskTopic`; treat a prompt whose topic segment is empty/whitespace as a hard reject.
- Route the rejection through the existing failed-task path with `error_msg = "dispatch rejected: empty task topic"` so it surfaces in the completion notification instead of a timeout.
- Files: `internal/coordinator/<dispatch>.go` (+~60 LOC), error defs (+~10 LOC).

**Phase 2 — Topic persistence + worker recovery (Day 2)**
- At dispatch (after the gate passes), write the topic to the message store keyed by `correlation_id` (a `task-topic`-typed message, or a dedicated `task_topics` collection read via the messages API).
- Add a worker preamble helper: detect the agent-task prompt shape, and if topic-less, run one `messages read <correlation_id>`; inject on success.
- Files: `internal/coordinator/<dispatch>.go` (+~40 LOC), worker preamble (+~50 LOC).

**Phase 3 — Fast worker-side timeout (Day 3)**
- When a topic-less prompt cannot be recovered, exit within a short bound (~60 s) with `ErrTopicUnrecoverable` rather than running to the 20-min executor timeout.
- Optionally make `coordinator retry` refuse to re-dispatch a task that previously failed `ErrEmptyTaskTopic` until the topic is supplied (prevents identical retry loops).
- Files: worker entry (+~30 LOC), `internal/coordinator/retry` (+~20 LOC).

### Files to Modify

| File | Change | Est. LOC |
|------|--------|----------|
| `internal/coordinator/<dispatch>.go` | Validation gate + topic persistence | +100 |
| `internal/coordinator/errors.go` | `ErrEmptyTaskTopic`, `ErrTopicUnrecoverable` | +15 |
| worker entrypoint (executor preamble) | Topic-shape detection + one-shot recovery + fast exit | +80 |
| `internal/coordinator/<retry>.go` | Refuse identical re-dispatch on empty-topic failure | +20 |
| coordinator dispatch tests | Reproduce 2026-06-05 empty-topic condition | +80 |

> **Note for the implementer:** the exact dispatch/render function that emptied the topic for `task-0c808c59`, `task-1dfb8f3b`, `task-49237121` must be located and confirmed on the coordinator host (it is not reproducible from the worktree alone, since the task store is host-local). The gate (Phase 1) is valuable regardless of that root cause and should land first.

## Examples

**Before (observed):**
```
$ # task-1dfb8f3b dispatched with topic-less prompt
$ # worker searches messages, coordinator DB (cgo stub), session artifacts...
$ # ... 20 minutes elapse ...
TaskCompletion{status: failed, error_msg: "executor failed: claude task failed: timeout after 20m0s"}
# Retry re-dispatches the SAME empty prompt → fails identically.
```

**After (Phase 1 gate):**
```
$ # coordinator renders prompt for task-1dfb8f3b, topic is empty
dispatch: task-1dfb8f3b rejected — ErrEmptyTaskTopic (no executor launched)
TaskCompletion{status: failed, error_msg: "dispatch rejected: empty task topic"}
# Cost: <100 ms, zero executor tokens. Signal is explicit and actionable.
```

**After (Phase 2 recovery, if a topic-less prompt still reaches a worker):**
```
$ # worker sees agent-task shape, no topic
worker: topic missing in prompt; resolving correlation_id msg_20260605_035536_1dfb8f3b
worker: recovered topic from message store → "M-… <milestone>"
worker: proceeding normally
```

## Success Criteria

- [ ] Dispatch gate rejects any rendered prompt with an empty topic segment (<100 ms, typed error)
- [ ] No executor is launched for a topic-less task
- [ ] Task topic is persisted to the message store keyed by correlation-ID at dispatch
- [ ] A worker can recover its topic via `messages read <correlation_id>` on a `CGO_ENABLED=0` binary
- [ ] A topic-less, unrecoverable task fails in <60 s, not 20 min
- [ ] `coordinator retry` does not re-dispatch an identical empty-topic prompt
- [ ] Regression test reproduces the 2026-06-05 empty-topic dispatch and asserts rejection
- [ ] All tests passing (`make test`)
- [ ] Documentation updated (CHANGELOG.md; coordinator guide)

## Timeline

| Day | Work |
|-----|------|
| 1 | Phase 1 — validation gate + typed error + regression test (ships alone, eliminates the silent-timeout class immediately) |
| 2 | Phase 2 — topic persistence to message store + worker one-shot recovery |
| 3 | Phase 3 — fast worker-side failure + retry guard; docs + CHANGELOG |

(Estimates doubled from the initial guess per skill guidance; Phase 1 is the high-value half and is independently shippable.)

## Related Documents

- [m-arch3-task-classification.md](../v0_13_0/m-arch3-task-classification.md) — task model the dispatch path builds on
- [m-arch4-executor-stream-processor.md](../v0_13_0/m-arch4-executor-stream-processor.md) — executor/stream boundary where the timeout fires
- [m-arch5-error-handling-strategy.md](../v0_13_0/m-arch5-error-handling-strategy.md) — error-routing conventions this reuses (dead-letter / failed-with-reason)
- [m-msg-auto-triage-pipeline.md](m-msg-auto-triage-pipeline.md) — message-store + Pub/Sub event bus this leverages for persistence/recovery

---

**Provenance note:** Created by `design-doc-creator` while handling `task-1dfb8f3b`, whose own dispatch exhibited this exact defect (topic-less prompt → 20-min timeout). The original milestone topic intended for `task-1dfb8f3b` could not be recovered from any worker-reachable source; this doc captures the systemic dispatch defect responsible instead of substituting an invented feature topic.
