## M-AGENT-STEP-CANCELLATION: abort a running `std/ai.step()` — extension + one small enabling fix (#231)

**Status**: PLANNED
**Target**: v0.28.0 (tentative)
**Priority**: P2 (Medium — in-lane agentic need; today a long AI step can only be killed with SIGKILL, losing the run summary)
**Estimated**: ~1 day fix + ~0.5 day extension
**Dependencies**: Concurrent-stdin primitive already exists (`std/stream.asyncReadStdinLines` + `selectEvents`). Needs one scoped `std/ai` runtime fix (Lane A) before the extension (Lane B) is useful.

**Tracks**: [#231](https://github.com/sunholo-data/ailang/issues/231) — *"std/ai.step should support cancellation via stdin or signal"* (reported by `motoko_explore`).
**Verified against**: v0.27.0 (`std/ai.step` has no cancellation arg; `asyncReadStdinLines` present).

---

## First: correcting the record on #231

The Snake feedback brief (Felix) cited #231 as "no stdin support," which made it look like a
*human keyboard input* gap. **It is not.** #231 is an agentic concern: when motoko sends
`{"type":"abort"}` to stdin **during** an AI step, the process can't stop until the LLM HTTP call
returns (30–120s), because `readLine()` is only checked *between* tasks and the Go HTTP client blocks
uninterruptibly. That is squarely in AILANG's lane (it's about the harness controlling its own runs),
not the out-of-lane "raw keyboard for games" thing.

Three distinct stdin stories were being conflated. Keeping them separate is half the value here:

| Story | State | Lane |
|---|---|---|
| **Line input** (`readLine`, `asyncReadStdinLines`) | ✅ **Works today** | done |
| **Abort a running agent step** (this doc / #231) | ❌ the real gap | extension + small fix |
| **Raw-mode keypress for human games** (Felix's actual need, *unfiled*) | ❌ separate | by-design **not in core** (non-deterministic, no WASM, no agentic need) |

---

## What already exists (so we don't rebuild it)

- `std/io.readLine() -> string ! {IO}` — blocking line read. Works; verified.
- `std/stream.asyncReadStdinLines(name, priority) -> StreamSource ! {Stream}`
  ([std/stream.ail:151](../../std/stream.ail#L151)) — a **concurrent, non-blocking** stdin line source
  meant for `selectEvents` multi-source dispatch. **The "watch stdin while something else runs"
  primitive is already here.** No new input plumbing is needed.

## The actual gap

`std/ai.step(model, messages, tools)` ([std/ai.ail:191](../../std/ai.ail#L191)) takes no cancellation
handle, and `_ai_step` issues a blocking HTTP request with no `context.Context` wired through. So even
with `asyncReadStdinLines` telling you "abort arrived," you have nothing to cancel the in-flight call
with. SIGKILL is the only lever, and it skips the graceful `run_summary` event.

---

## Design — two lanes (per PROGRAM.md §4)

### Lane A — AILANG fix: make the in-flight AI call cancellable (the enabling primitive)

The extension can't interrupt a Go HTTP client that has no cancel handle, so the runtime must provide
one. Two layers, smallest-first:

**A1 (floor / smallest): SIGINT cancels the in-flight AI context.** Register a runtime signal handler
that cancels the `context.Context` of any in-flight `_ai_step`, so `step` returns
`Err(Cancelled)` promptly instead of blocking. This is issue-option C, needs **zero AILANG surface
change**, and immediately makes `Ctrl+C` graceful (the loop catches the `Err` and emits `run_summary`).
This is arguably a core-floor correctness fix (a blocking call that ignores cancellation is the floor
being wrong), not a feature.

**A2 (graceful stdin abort): a cancellable step variant.** For motoko's `{"type":"abort"}` protocol
(not a signal), expose cancellation as a value the program already has. Recommended shape — thread a
cancel source the runtime polls:

```ailang
-- std/ai: cancellable sibling of step (same Ok(StepResult)/Err(AIError) shape).
-- `cancel` is a handle the runtime checks; when fired, the HTTP context is cancelled
-- and step returns Err(Cancelled).
export func stepCancellable(
  model: string, messages: [Message], tools: [ToolDef], cancel: CancelToken
) -> Result[StepResult, AIError] ! {AI}
```

`CancelToken` is produced from the same stream the abort arrives on (e.g. a token that fires when an
`asyncReadStdinLines` source yields an `{"abort"}` line), so A2 composes with the *existing*
`selectEvents` machinery rather than inventing a new concurrency model. Add `Cancelled` to `AIError`.

*(Open question for design review: token-handle vs. wiring cancellation implicitly through the Stream
effect's `selectEvents`. A2 may be unnecessary if A1 + an extension that converts a stdin-abort into a
self-`SIGINT` is judged clean enough — decide before building A2.)*

### Lane B — Motoko extension: the graceful-abort orchestrator

A `motoko-ext-abort` package (under `mk-ast/packages/motoko-ext-*`) owns the *policy*, using Lane A's
primitive:
- On run start, open an `asyncReadStdinLines` source for control messages.
- Via `selectEvents`, watch for `{"type":"abort"}` concurrently with the dispatch step.
- On abort: fire the cancel handle (A2) — or raise SIGINT to self (A1) — let the in-flight step return
  `Err(Cancelled)`, then emit `run_summary` and shut down cleanly.

This keeps the smarts (abort protocol, graceful-shutdown sequencing, telemetry) in the extension layer
where PROGRAM.md wants them, and keeps the core change to the minimum that *only the runtime* can do.

---

## Recommendation

Ship **A1 first** — it's the smallest change, fixes the worst symptom (Ctrl+C now graceful), and may be
all that's needed. Build **Lane B** on top for the stdin-`{"abort"}` protocol motoko actually uses,
reaching for **A2** only if routing the abort through a self-signal proves ugly. Do **not** bundle the
raw-keyboard-for-games item — that's a separate, by-design-not-in-core decision (its own note in
limitations.md).

## Testing

- A1: start a slow/mocked `step`, send SIGINT, assert `step` returns `Err(Cancelled)` within ~1s and a
  `run_summary` is emitted (no SIGKILL).
- A2/B: drive an `asyncReadStdinLines` source that yields `{"type":"abort"}` mid-step; assert prompt
  `Err(Cancelled)` + graceful shutdown.
- Regression: a normal (un-aborted) step is bit-for-bit identical to today's `step`.

## Out of scope

- Raw-mode / non-blocking single-keypress input (human games) — separate; by-design not in core.
- A general structured-concurrency cancellation framework — this is scoped to AI-step abort only.
- Replacing `step` — `stepCancellable` is additive; `step` stays.
