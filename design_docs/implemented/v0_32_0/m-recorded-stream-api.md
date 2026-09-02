# M-RECORDED-STREAM-API — Productionize the offered recorded `std/ai` stream

**Status**: **DECIDED by Mark 2026-08-03 (attended): OPTION (c) — bound the drain locally in
the recorded op, no interface change (author-endorsed, #546 comment 2026-08-01: sealed
StreamChunk interface makes the fail-loud drain trigger unreachable today) → ROUTABLE to
sprint-planner.** [Was: PARKED — `needs-human-review` (mission iteration 124, 2026-07-31).] Quorum BLOCKED
twice (R1 and R2). The design DIRECTION — adopt @arniwesth's additive sibling — survived both
rounds and is **not** contested. The park is CLEARED; see "✅ THE DECISION" below and the
R3 section of the Quorum verification log.
**PLANNED — iteration 133 (sprint-planner).** Cut into two sprints; execute from
[`m-recorded-stream-api-sprint-plan.md`](m-recorded-stream-api-sprint-plan.md). One design change
was made by the planner under delegated authority: **the sentinel-panic abort is cancelled**
(drain mode itself ships unchanged) — see §PLANNER RULING and §R4. Mark's option-(c) ruling is
preserved in full; the one residual it loses (a bound on post-failure *wall-clock* as opposed to
*work*) is the sprint's single open human question.
**Target**: **v0.32.0** — corrected by the controller, iteration 133. The doc was written against
`v0.31.0`, but **v0.31.0 shipped on 2026-07-29** (`gh release list` → `AILANG v0.31.0  Latest`;
tag `1f6f7dd28`, and current dev is 58 commits past it). Every `Since: "v0.31.0"` /
"experimental since v0.31.0" string below is therefore stale and must read **v0.32.0**. Confirm
the exact string at merge time — releases are Mark's sole decision and this loop never releases —
and move this file to `design_docs/planned/v0_32_0/` when it is next touched.
**Priority**: P0 (blocking external consumer architecture)
**Estimated**: 5.5–6.0 engineering days (was 4–5; grew by the drain bound in M1 and the new
exhaustiveness-guard milestone M4 — see the milestone table)
**Dependencies**: external implementation offered in ailang#546 by @arniwesth; Motoko Project 009 ADR-001 is formally blocked on this API

> ## ✅ THE DECISION — the park is cleared (Mark, attended, 2026-08-03)
>
> **The question that parked this doc:** R2's surviving objection (`gpt5-6-sol`) was that the
> fail-loud path had an **unbounded wait** — after an unencodable chunk the design said "drain
> safely" if the provider cannot be cancelled, with no deadline, cancellation mechanism, or
> budget. The reviewer's own prescribed fix (a cancellable provider context) was checked against
> the code and requires a 7-implementer `AIHandler` interface change reaching
> `cmd/wasm/effects.go` (`internal/effects/ai.go:87` takes no `context.Context` —
> VERIFIED BY CONTROLLER at `130ad1da2`, re-confirmed at `a929ec452`). The options put to Mark:
> **(a)** land now with a documented unbounded-drain caveat; **(b)** take the `AIHandler`
> cancellation change as a blocking dependency; **(c)** bound the drain locally inside the
> recorded op.
>
> **THE RULING: OPTION (c) — bound the drain LOCALLY inside the recorded op, with a finite
> chunk/byte budget, NO interface change.** Ruled by Mark, attended, 2026-08-03.
>
> **The author independently endorsed (c)** (@arniwesth, ailang#546 comment, 2026-08-01) with an
> argument stronger than the one the controller made: the `StreamChunk` interface is **sealed**
> by the unexported `streamChunkMarker()` method (`internal/ai/provider.go:159-163`), exactly
> three variants implement it, and `encodeStreamChunk` handles all three — so **the fail-loud
> drain trigger is unreachable for any value constructible today** (VERIFIED BY CONTROLLER at
> `a929ec452`, four independent legs including the no-embedding and untyped-nil-in-WASM checks).
> The drain path this bound protects is a forward-compatibility guard, not a live hazard.
>
> **Option (b) is REJECTED**, with the author's reason recorded: a 7-implementer interface
> change reaching WASM forfeits the purely-additive property that the ADOPT verdict and the
> 1.x-stability claim rest on — to make cancellable a path that has **no reachable trigger**.
> Option (a) is likewise not taken: an undocumented-bound caveat is exactly the silent-fallback
> shape this repo forbids. The concrete bound is specified in §Count invariant and drain bound.

> **Provenance and evidence labels.** The implementation under review is the complete
> `arni-546.patch` supplied with this iteration. “VERIFIED HERE” means a command or source read
> performed in this worktree and recorded in the Verification Log. “VERIFIED BY ME (controller)”
> means the first-party evidence supplied in the iteration charter at HEAD `130ad1da2`; it was not
> rerun in this design-doc session. “VERIFIED BY CONTROLLER (outside sandbox, `-run` isolated)”
> means the controller supplied the exact isolated command and transcript after the first quorum.
> The pre-read is supporting evidence, not a quorum verdict.

## Verification Log

| Claim | Method actually run | Result |
|---|---|---|
| Patch applies to this worktree and is purely additive | `git apply --check arni-546.patch`; `git apply --stat arni-546.patch` | **VERIFIED HERE** — check passed; 452 insertions, 5 files, 0 deletions |
| Offered public shape | Read the `std/ai.ail` hunk and builtin type in `arni-546.patch` | **VERIFIED HERE** — sibling `stepWithStreamRecorded`; `RecordedStream = {chunks: [StreamChunk], outcome: Result[StepResult, AIError]}`; callback remains `! {IO}`; existing function is not edited |
| Capture order and error preservation | Read `aiStepWithStreamRecorded` and its partial-error test in the patch | **VERIFIED HERE** — encoded value is appended before `FnCaller`; both success and error paths call `makeRecordedStream(recorded, outcome)` |
| Duplication and counter mismatch | Compared patch handler with current `internal/effects/ai_step.go`; searched `chunkCount`/`recorded` | **VERIFIED HERE** — argument checks/decodes and dispatch/trace handling are duplicated; count increments before encode-nil, recording occurs only after non-nil encode |
| Existing callback documentation is false | Read `std/ai.ail:324` and `examples/runnable/ai_streaming.ail:40-42`; controller's live positive/negative probes | **VERIFIED HERE** for the contradictory declarations; **VERIFIED BY ME (controller)** for compiler behavior and exact closed-row error |
| Current streaming example syntax checks | `ailang prompt --version v0.16.2`; `ailang check examples/runnable/ai_streaming.ail` | **VERIFIED HERE** — prompt loaded; `No errors found!` |
| Existing focused Go packages | `go test ./internal/builtins ./internal/effects` | **VERIFIED HERE** — builtins passed; effects was blocked by sandbox port denial in unrelated `TestNetHTTPRequestBytes_RoundTripSHA` |
| Four offered tests pass | Controller ran `cd .claude/worktrees/arni-546-eval` then `go test ./internal/effects/ -run 'Recorded\|StreamRecorded' -v` outside the sandbox | **VERIFIED BY CONTROLLER (outside sandbox, `-run` isolated)** — `--- PASS: TestAIStepWithStreamRecorded_ReturnsDeliveredChunksOnSuccess (0.00s)`; `--- PASS: TestAIStepWithStreamRecorded_ReturnsChunksOnErrorPath (0.00s)`; `--- PASS: TestAIStepWithStreamRecorded_ContentDeltaConcatEqualsMessageContent (0.00s)`; `--- PASS: TestAIStepWithStream_UnchangedByRecordedVariant (0.00s)`; package PASS (`ok github.com/sunholo-data/ailang/internal/effects 0.364s`). General lesson: a sandbox port-bind denial in an unrelated network test is **UNINFORMATIVE**, not a product failure; isolate the relevant tests with `-run` instead of inheriting another run's claim. |
| Bytecode bridge uses the same route | Read `cmd/ailang/run_helpers.go`, `internal/effects/ops.go`, current streaming builtin/effect registration, and patch registration | **VERIFIED HERE** — evaluator-only bytecode entries dispatch through `runtime.CallEntrypoint`; both stream operations use `RegisterEffectBuiltin` → `effects.Call` → `Registry["AI"]`; no op-specific VM bridge is required |
| ADR-009 needs this exact semantic substrate | Read ADR-009 D1, acceptance gate, consequences, and handoff; read accepted ADR-007 D1–D4 | **VERIFIED HERE** — ordered emissions must survive success or partial failure alongside final outcome, while immediate live projection remains |

The controller additionally verified at HEAD `130ad1da2` with freshly rebuilt
`v0.31.0-31-g130ad1da2` that `StepResult` carries no chunks, an `{IO}` callback checks, and a
recording `{FS}` callback fails closed-row unification. ADR-009 line 134 independently records the
same result against v0.30.0. These are **VERIFIED BY ME (controller)**, not rerun here.

### R3 revision rows (iteration 133, this worktree at HEAD `a929ec452`)

The five controller-supplied facts for this revision (gap-at-HEAD probes; sealed
`StreamChunk`/three-implementer/no-embed/untyped-nil-WASM legs; `ai.go:87` no-context with
7 implementers; both false doc claims live; `encodeStreamChunk` coverage) are **VERIFIED BY
CONTROLLER** at `a929ec452` with rebuilt `v0.31.0-58-ga929ec452`, not rerun here — except where
a row below independently re-observed the same site. The controller flags one leg — "no struct
anywhere embeds `ai.StreamChunk`" — as an empty search for which no positive control could be
built; treat it accordingly. The author's three adapter-loss claims were NOT controller-verified
and are re-verified first-party below.

| Claim | Method actually run | Result |
|---|---|---|
| Anthropic delta switch has NO `default`; a future delta type is silently ignored, `onChunk` never called | Read `internal/ai/anthropic/streamstep.go:300-347` (full switch body); `grep -n 'default' internal/ai/anthropic/streamstep.go` | **VERIFIED HERE** — grep rc=1 (zero hits in the whole file); known-positive control in the same call: `grep -n 'default:' internal/effects/ai_step.go` → `447:	default:`. Switch at `:330` has exactly `text_delta`/`input_json_delta`/`thinking_delta` cases |
| `input_json_delta` is accumulated but deliberately NOT emitted — tool-call stream content absent from any recorded log by design | Read `streamstep.go:336-337` and `:361-369` | **VERIFIED HERE** — `b.inputJSON.WriteString(ev.Delta.PartialJSON)` with no `onChunk` call; content surfaces only as `ai.ToolCall` on the final response at `content_block_stop` |
| WASM: unknown JS `kind` → nil → filtered → no callback, no record, no counter increment; loss invisible to the effects layer | Read `cmd/wasm/effects.go:238-247` and `:345-366` | **VERIFIED HERE** — `jsToStreamChunk` returns nil for unknown kinds (comment at `:347` says so explicitly); `if chunk != nil` at `:243` filters; the effects-layer `onChunk` (where any counter lives) is never invoked |
| Existing `aiStepWithStream` silently skips an unencodable chunk (shipped), and counts before encoding | Read `internal/effects/ai_step.go:376-392` | **VERIFIED HERE** — `chunkCount++` at `:378` precedes `encodeStreamChunk`; `if encoded == nil { return }` at `:380-382`: no callback, no trace event, stream continues |
| `encodeStreamChunk`'s unencodable path is its `default: return nil` | `grep -n 'default:' internal/effects/ai_step.go` | **VERIFIED HERE** — `447:	default:` (function begins `:422`) |
| Anthropic adapter cleans up via `defer`, so a sentinel unwind releases the connection | `grep -n 'defer' internal/ai/anthropic/streamstep.go` | **VERIFIED HERE** — `:51 defer span.End()`, `:91 defer func() { _ = httpResp.Body.Close() }()`. **MOOT since the PLANNER RULING** — no sentinel ships, so nothing unwinds. Row retained as history |
| Fake-handler test pattern exists; NO `onChunk(nil)` call site exists yet anywhere | `grep -rn 'StepWithStream' internal/effects/*_test.go`; `grep -rn 'onChunk(nil)' internal/ cmd/` | **VERIFIED HERE** — positives: `fakeStepHandler` at `internal/effects/ai_step_test.go:53`, `routingStubHandler` at `ai_routing_trace_test.go:48` (instrument sees positives in the same call); the `onChunk(nil)` search returned rc=1/empty |
| Both false "open row" doc claims still live at this HEAD | `sed -n '38,44p' examples/runnable/ai_streaming.ail`; Read `std/ai.ail:315-337` | **VERIFIED HERE** — example `:40-42`: "you can wire whatever side-channel you need (a websocket, a TUI buffer, a metrics sink)"; `std/ai.ail:324`: "The callback's effect row is open"; closed `! {IO}` declaration at `:335` (matches CONTROLLER's facts 1 and 4) |

## Axiom Compliance

| Axiom | Score | Justification |
|---|---:|---|
| A1 Determinism | +1 | Returns the exact encodable prefix in arrival order and deterministically terminates with typed `Internal` failure at the first unencodable chunk; it never skips that chunk and continues successfully |
| A2 Replayability | +2 | A non-encoding-error outcome certifies a complete log **of adapter-EMITTED chunks**; an encoding error explicitly marks the returned prefix incomplete, so replay never mistakes partial evidence for a complete transcript. The guarantee is scoped to the `onChunk` boundary, not the provider wire: adapters already filter below it (no-default delta switch, unemitted `input_json_delta`, WASM unknown-kind nil — see §What "lossless" means), so replay reproduces what the effects layer observed, which is the substrate ADR-009 needs, not a wire capture |
| A3 Effect Legibility | 0 | Callback stays closed at `{IO}` and the call stays `{AI}`; no hidden capture effect is exposed |
| A4 Explicit Authority | 0 | No new capability or authority |
| A5 Bounded Verification | +1 | Ordered callback/return parity is directly testable with finite fakes |
| A6 Safe Concurrency | 0 | No concurrency semantics change; retention is call-local |
| A7 Machines First | +1 | A typed emission log is machine-consumable without scraping IO |
| A8 Minimal Syntax | +1 | One additive sibling and one record type; no language syntax change |
| A9 Cost Visibility | -1 | Unbounded in-memory retention must be explicit to callers |
| A10 Composability | +1 | Live rendering and deterministic recording coexist without widening the callback row |
| A11 Structured Failure | +1 | Provider failures and representation failures are typed in `outcome`; partial chunks remain available but cannot masquerade as a complete log |
| A12 System Boundary | +1 | Capture occurs at the provider/effect boundary where arrival order is known |

**Net: +8 — proceed with the offered design after the productionization gates below.**

## Problem Statement

At current HEAD, `stepWithStream` accepts
`on_chunk: (StreamChunk) -> () ! {IO}` and returns `Result[StepResult, AIError]`; `StepResult` has
no chunks. This was **VERIFIED BY ME (controller)** by source inspection and live compiler probes.
The positive `{IO}` rendering callback checks, but a callback that appends chunks using `{FS}`
fails with incompatible closed rows. `std/io` has no file-write operation, also **VERIFIED BY ME
(controller)**. Therefore live streaming and lossless recording are mutually exclusive through the
current API. ailang#546 is a real API gap, not a ghost issue.

This blocks ADR-009's deterministic test-world architecture. That ADR requires the live adapter to
project each chunk immediately and return the identical lossless, ordered emission log beside the
final `StepResult` or `AIError`. The driver then appends that log to its authoritative trace. A
final response alone cannot reconstruct thinking, usage, or partial content after a transport
failure. ADR-007's accepted scope confirms this is infrastructure for a single-actor logical-fault
deterministic environment; it does not turn the AILANG API itself into a Motoko extension.

Two current comments worsen the gap by teaching a nonexistent workaround:

- `std/ai.ail:324` says the callback row is open and `{IO}` is only typical.
- `examples/runnable/ai_streaming.ail:40-42` says arbitrary websocket, TUI-buffer, or metrics
  side-channels can be wired.

The declarations directly below those comments are closed at `{IO}`. The controller's `{FS}` probe
demonstrates the contradiction. Adding the recorded sibling does not widen that row, so adoption
without documentation repair would leave the trap in place.

## Goals

- Adopt the offered additive `stepWithStreamRecorded` semantics without breaking or changing
  `stepWithStream`.
- Preserve every provider chunk in arrival order whenever the public `StreamChunk` representation
  can encode the stream; otherwise stop recording/delivery at the first unencodable chunk and
  return a typed, non-retryable `AIError{code=Internal}` that explicitly marks the log incomplete.
- Refactor the existing and recorded operations onto one validation/decode/dispatch core so fixes
  cannot drift between siblings.
- Ship the feature as a complete experimental standard-library surface: checked example,
  teaching/discovery entries, generated builtin docs, website, and changelog.
- Clear ADR-009's upstream substrate blocker with a direct consumer integration probe.

## Non-Goals

- Widening the callback effect row or adding `{FS}`, `{Net}`, `{SharedMem}`, or `{Trace}` authority.
- Changing the return type, behavior, or stability of existing `stepWithStream`.
- Building Motoko Project 009, its replay engine, or its ledger.
- Bounding healthy-stream retention, spilling chunks to disk, backpressure, resumable streams,
  or concurrency. (The POST-FAILURE drain budget of §Count invariant is not this: it bounds
  wasted work after the op has already failed, never the recorded log of a healthy stream.)
- Promoting either stream API beyond `StabilityExperimental`.

## Decision: ADOPT, Do Not Reinvent

The independent verdict is **ADOPT with productionization**, agreeing with the pre-read after
reviewing the patch and ADRs directly.

The offered design gets the high-impact choices right:

1. It is a sibling API, so the existing experimental streaming contract and callers remain
   unchanged.
2. It tees one encoded value to an in-memory ordered list and the callback, appending before
   callback invocation. Callback failure remains fail-soft and trace-visible, matching the current
   stream operation.
3. It returns chunks on both terminal outcomes, which is the distinguishing requirement rather
   than a convenience.
4. It leaves the callback closed at `{IO}`. Recording is the operation's responsibility, not a
   capability escape through consumer code.
5. It follows the existing builtin/effect registration and typed-error conventions.

Rejecting would preserve a reproduced, externally blocking gap. Reinventing would spend design
risk on choices already represented by working code and focused tests. However, the patch is a
spike, not merge-ready: roughly 80 lines of validation/decode/dispatch logic duplicate
`aiStepWithStream`, metadata says `Since: prototype`, documentation and discovery are incomplete,
and the count invariant needs resolution. Productionization is a condition of adoption, not a
follow-up.

## Routing Call: Core, Not Extension

Route this as a **core-floor addition** to `std/ai`, `internal/builtins`, and `internal/effects`,
then re-freeze the surface.

The standing mission bias—if it can be an extension, it is an extension—does not fit this change.
An extension cannot recover chunks that the core provider callback discards from the return path,
cannot widen the closed callback row, and cannot register a new `std/ai` typed builtin without
forking the compiler/runtime boundary. The missing information exists only while the core
`AIHandler.StepWithStream` call drains. Capture must happen at that boundary.

The exception is narrow: one additive experimental sibling, demanded by a real external consumer,
with no syntax, capability, provider protocol, or existing-contract change. Motoko-specific replay
and ledger policy remain extension-resident. Thus core owns the complete-or-explicitly-incomplete
recording primitive; extensions own what the log means and how it is replayed.

## API and Return-Shape Decision

Adopt the offered public contract:

```text
RecordedStream = {
  chunks: [StreamChunk],
  outcome: Result[StepResult, AIError]
}

stepWithStreamRecorded(model, messages, tools, cache_breakpoints, on_chunk)
  -> RecordedStream ! {AI}
```

The callback contract remains `(StreamChunk) -> () ! {IO}`. This block is interface notation
transcribed from the patch, not a claim that an unpatched HEAD accepts the new symbol.

Do **not** use `Result[{result, chunks}, AIError]`. On a mid-stream failure, `Err` has nowhere to
carry already observed chunks, exactly when replay evidence is most valuable. `{chunks, outcome}`
keeps a complete ordered emission log orthogonal to the final status for every representable
provider stream. A representation failure instead returns the exact delivered prefix with a typed
terminal error, so completeness is explicit rather than falsely promised. This satisfies ADR-009
D1 for supported provider streams. If representation fails, ADR-009 must reject the turn as
incomplete after preserving the prefix; it must not replay it as a complete stream.

Retention is intentionally unbounded for v0.32.0: all chunks remain in memory until the call
returns, so memory grows linearly with stream length. This is acceptable for bounded agent turns,
but unsuitable for arbitrarily long streams. The builtin LongDesc and public docstring MUST state
that cost; future bounded/spooled variants require a separate design.

## Solution Design

### Shared streaming core

Refactor `aiStepWithStream` and the recorded variant onto one private core responsible for:

- arity/type validation, handler and `FnCaller` guards;
- message, tool-schema, and cache-breakpoint decoding;
- one `AI.StepWithStream` invocation;
- encoding each provider chunk exactly once;
- immediate callback delivery and fail-soft callback-error tracing;
- typed operation-error classification and final response conversion; and
- trace metadata/count construction.

The core accepts an explicit collection/error policy. `aiStepWithStream` preserves its current
external behavior; `aiStepWithStreamRecorded` selects fail-loud recording and returns
`{chunks, outcome}`. There must not be two copies of validation or decoding logic. Regression tests
must prove both wrappers produce the same outcome and callback sequence for all representable
chunks, while an unencodable-chunk test proves only recorded mode gains the new completeness guard.

### Count invariant and drain bound

The patch increments `chunkCount` before `encodeStreamChunk`, but appends only after a non-nil
encoding. An unknown/skipped provider chunk can therefore produce trace `chunks:N` while
`len(recorded) < N`. Production code MUST NOT skip that chunk or continue as though the log were
complete. At the first nil/unknown encoding, the shared core records no value for that chunk,
invokes no callback for it, and overrides any provider terminal result with a non-retryable typed
`AIError{code=Internal, message="unencodable stream chunk at provider index N; recorded log is an incomplete prefix"}`.
The returned `chunks` are exactly the values already delivered before the failure.

**The drain bound (Mark's option (c), 2026-08-03 — replaces the previous unbounded "drain
safely" wording).** After the fatal chunk, the recorded op's callback enters **drain mode**,
implemented entirely inside `aiStepWithStreamRecorded` — no `AIHandler` change, no goroutine
that outlives the call:

- In drain mode the callback does no encoding, no recording, no AILANG-callback delivery; it
  only advances two O(1) counters: post-failure chunk count and post-failure payload bytes
  (sum of chunk text/JSON field lengths — cheap, no retention).
- The budget is **both** a chunk count AND a byte count, whichever exhausts first:
  `recordedDrainMaxChunks = 256`, `recordedDrainMaxBytes = 1 MiB` (named constants in
  `internal/effects`). Two dimensions because they bound different costs: the chunk count bounds
  callback-loop iterations against a provider emitting unbounded tiny chunks; the byte count
  bounds bandwidth-proportional work against a provider emitting few enormous chunks. Neither
  bound affects memory (drain mode retains nothing); both bound wasted wall-work.
- **On budget exhaustion the drain goes INERT** — ~~the callback panics with a private sentinel
  type recovered around `ctx.AI.StepWithStream`~~. **PLANNER RULING, iteration 133: the
  sentinel-panic abort is NOT taken** (option (i) of the controller refutation below; four
  reasons in §PLANNER RULING). Instead a single `drainExhausted bool` is set; every later
  callback returns on its first statement after one boolean load — no counting, no encoding, no
  recording, no delivery, no allocation. The two counters saturate rather than continuing to
  add. The op returns when the provider's stream ends. **No panic crosses the `onChunk`
  boundary, so no new obligation is imposed on any `AIHandler` implementer, in-repo or
  external, on any build target.**
- **What this does and does not bound.** Bounded: total post-failure *work* performed by AILANG
  (O(1) per chunk before exhaustion, one boolean load after; zero retention throughout).
  NOT bounded: post-failure *wall-clock*, which remains the provider's stream duration. That
  residual is stated in the LongDesc rather than papered over, and the honest fix — a
  cancellable provider context across the whole AI surface, on its own merits and not as a
  dependency of this — is filed as a follow-up (§Open Questions). That is also @arniwesth's own
  recommendation (#546, 2026-08-01: *"if cancellation is worth doing, I'd suggest it's worth
  doing on its own merits across the whole AI surface … not as a blocking dependency of this"*).
- **The already-decided typed `Internal` error is preserved and cannot be overwritten** —
  not by a provider terminal success/error arriving after the representation failure, and not
  by drain exhaustion. The first representation failure decides `outcome`; everything after it
  is diagnostics.
- **Observability**: trace metadata records `drain_exhausted:true` when the budget tripped.
  After an exhausted drain, `provider_chunks` is a **floor** (chunks counted until exhaustion),
  not an exact wire count — the flag is what tells a trace consumer which reading applies.

Acceptance (this row is merge-blocking and appears in sprint 1 milestone S1-M3): a fake handler
that keeps emitting chunks after the unencodable one stops being counted after exactly
`recordedDrainMaxChunks` further callbacks (and symmetrically for the byte budget);
`drain_exhausted:true` is traced; every post-exhaustion callback is observably inert (no
encode, no append, no AILANG-callback invocation — asserted via instrumented fake); the op
returns the exact delivered prefix with the original typed `Internal` error; no goroutine or
timer is created at any point; the existing `stepWithStream` path is externally unchanged.
There is no sentinel-containment test, because there is no sentinel.

> ### ⚠ CONTROLLER REFUTATION — the SENTINEL-PANIC ABORT IS UNSOUND ON THE WASM TARGET
>
> Added by the controller, iteration 133, AFTER the designer wrote the drain-bound section above.
> The two-budget **drain mode** is not in question — it is O(1) per chunk, retains nothing, needs
> no interface change, and is safe everywhere. **What is refuted is the `recover`-scoped
> sentinel-panic abort**, and the planner must adjudicate it before M1 is executable.
>
> **Measured first-party at `a929ec452`, two legs:**
>
> 1. **The WASM `onChunk` may run on a DIFFERENT GOROUTINE from the one that would `recover`.**
>    `cmd/wasm/effects.go:238-247` hands the Go callback to JS as a `js.FuncOf` wrapper, and the
>    JS handler's result is awaited via `awaitJSResult` → `awaitPromise` (`:43-53`) whenever it
>    returns a thenable. That function's own comment states the mechanism: *"js.FuncOf callbacks
>    run as goroutines, so blocking one goroutine doesn't freeze the browser event loop."* Go's
>    `recover` is **per-goroutine**, so a sentinel panic raised inside the JS-invoked callback
>    cannot be caught by a `recover` scoped to the `ctx.AI.StepWithStream` call — it is an
>    unrecovered panic on another goroutine, which is **fatal to the whole WASM module**. The
>    design's claim that "the sentinel never crosses the recorded op's boundary" holds on the
>    native path and fails here.
> 2. **No host test can catch this.** `cmd/wasm/effects.go` is `//go:build js && wasm`
>    (line 1, read directly). The file's own sibling comment says the pure-Go helpers were split
>    into `effects_helpers.go` *specifically* so they could be unit-tested on the host — which
>    is positive confirmation that the tagged file is not host-testable. So M1's proposed
>    "sentinel containment" test would pass **green** while the WASM path crashes: the
>    vacuous-pass shape this repo has now hit four times.
>
> **What is NOT refuted:** no provider adapter would swallow the sentinel — `grep -rn 'recover()'
> internal/ai/ internal/effects/ cmd/wasm/` finds **zero** hits under `internal/ai/`
> (known-positive control: the same pattern returns **37** hits repo-wide, incl.
> `internal/effects/stream.go:751` and four sites in `cmd/wasm/main.go`). `defer` hygiene is also
> real: anthropic 2, openai 2, gemini 1, `cmd/wasm/effects.go` 3; `internal/ai/handler.go` has 0
> but performs no stream I/O of its own.
>
> **The decision the planner owes (do not let the executor improvise it).** Note the cost/benefit
> has already shifted: drain mode alone bounds *work per chunk* to O(1); the sentinel only
> additionally bounds *callback count / wall-clock*, on a path the sealed-interface analysis shows
> is unreachable for any value constructible today. Options, in the controller's order of
> preference — but this is the planner's call, with reasons:
> **(i)** ship drain mode WITHOUT the sentinel abort, and document that the post-failure drain is
> bounded in work-per-chunk but runs to the provider's natural end;
> **(ii)** keep the sentinel on the native path only, gated so it can never be reached from a
> `js && wasm` build, and state the WASM behaviour explicitly;
> **(iii)** keep it as designed — only with a stated, testable argument that the WASM callback is
> always synchronous within the `Invoke` frame, which the `awaitPromise` path above appears to
> refute.
> Whatever is chosen, the M1 acceptance row above must be rewritten to match, and any
> containment test must declare that it does **not** cover `js && wasm`.

> ### ⚖ PLANNER RULING — option (i): DRAIN MODE SHIPS, THE SENTINEL DOES NOT
>
> Ruled by the sprint-planner, iteration 133, at `2f12ddacd`. **The controller's refutation is
> CONFIRMED and strengthened against a primary source.** The acceptance row and §Count invariant
> above have been rewritten to match. Four independent reasons, in decreasing order of weight:
>
> **R1 — The sentinel is unsound on `js && wasm`, per Go's own contract, not an inference.**
> Go 1.26.5 `syscall/js`, `FuncOf` doc comment (read at
> `$(go env GOROOT)/src/syscall/js/func.go`): *"Invoking the wrapped Go function from JavaScript
> will pause the event loop and **spawn a new goroutine**. Other wrapped functions which are
> triggered **during a call from Go to JavaScript** get executed on the same goroutine."*
> The exception clause is the whole question, and it does not cover us:
> `WasmAIHandler.StepWithStream` (`cmd/wasm/effects.go:231-260`) returns from
> `stepWithStreamCallback.Invoke(...)` with a thenable and then parks the calling goroutine in
> `awaitPromise`'s `select` (`:53-88`). Every `funcWrapper` invocation after that arrives from
> the JS event loop — *not* during a Go→JS call — and therefore runs on a **new goroutine**.
> `recover` is per-goroutine. A sentinel panic there is unrecovered and fatal to the module.
>
> **R2 — Option (iii) is refuted, and would be unsound-by-delegation even if it weren't.**
> (iii) needs the JS handler to deliver every chunk synchronously inside `Invoke`. The repo's own
> `StepWithStream` doc comment (`cmd/wasm/effects.go:222-229`) says the wrapper is released
> *"once the JS handler resolves its Promise"* — the async path is the designed path, and any
> real streaming handler (fetch + `ReadableStream`) is async by construction. Worse: (iii) makes
> AILANG's memory safety contingent on a third-party JS caller's implementation choice. A
> soundness property a caller can revoke is not a soundness property.
>
> **R3 — Option (ii)'s gate would be enforced by nothing, and its benefit has the same trigger as
> its hazard.**
> (a) **PR CI never compiles the `js && wasm` target.**
> `grep -n 'wasm' .github/workflows/ci.yml .github/workflows/build.yml` → **rc=1, zero hits**
> (known-positive control in the same call: the identical pattern returns 20+ hits in
> `.github/workflows/release.yml`, whose `build-wasm` job is at `:17`). A build-tag slip in a
> WASM-gated sentinel would therefore surface at **release**, not on the PR. Gating an unsound
> branch behind a tag that CI never exercises is a weaker instrument than not having the branch.
> (b) **On WASM the drain branch is unreachable *by construction*, so today the gated sentinel is
> dead code.** `jsToStreamChunk` (`cmd/wasm/effects.go:345-366`) returns exactly
> `{StreamContentDelta, StreamThinkingDelta, StreamUsage, untyped nil}`, and `:243` filters the
> nil before `onChunk` — so the WASM `onChunk` can only ever receive an encodable variant. This
> is a *local structural* guarantee in the very file at risk, strictly stronger than the sealed
> `streamChunkMarker()` argument. The only way the sentinel ever starts mattering on WASM is the
> forward-compat scenario (a fourth variant added to `jsToStreamChunk` without a matching
> `encodeStreamChunk` case) — and in exactly that scenario option (ii) would upgrade the
> outcome from *"typed `Internal` + a long drain"* to *"**fatal WASM module crash**"* if the gate
> were wrong, on a target no PR build exercises. Under (i) that same scenario yields a typed
> error on every target.
>
> **R4 — The sentinel is a covert version of the interface change Mark already rejected.**
> `AIHandler.StepWithStream` has 7 implementers — 4 production (`internal/ai/handler.go:351`,
> `internal/effects/ai.go:246` (decorator), `internal/effects/ai.go:341` (stub),
> `cmd/wasm/effects.go:231`) and 3 test doubles (`ai_routing_trace_test.go:48`,
> `ai_step_test.go:53`, `ai_step_with_stream_test.go:282`) — **plus out-of-repo implementers**,
> including @arniwesth's own driver. Panicking out of the `onChunk` we hand them imposes a new,
> unstated requirement on all of them: be panic-transparent — `defer`-clean and carrying no
> blanket `recover()` in the stream loop. That is a contract change with no compiler signal, no
> interface doc, and no version bump. Mark rejected option (b) precisely because an interface
> change reaching WASM forfeits the purely-additive property the ADOPT verdict rests on; the
> sentinel forfeits the same property **invisibly**. If we would not take it openly we must not
> take it covertly.
> Our *own* adapters happen to be sentinel-safe, which is why R4 is about the contract rather
> than about us: `grep -rn 'recover()' internal/ai/` → **rc=1, zero hits** (control: 37 hits
> across `internal/ cmd/`, incl. `internal/effects/stream.go:751` and 4 in `cmd/wasm/main.go`);
> `grep -rn 'go func' internal/ai/` (non-test) → **rc=1, zero hits** (control: 23 in
> `internal/effects/`). So every native adapter calls `onChunk` synchronously on the caller's
> goroutine — the sentinel *would* have worked natively. It is being declined on contract and
> instrument grounds, not because it cannot work.
>
> **REFUTED doc argument.** §Count invariant claimed the sentinel is *"the established Go
> control-flow escape for callback-driven APIs (`encoding/json`, `fmt` use it internally)."*
> The analogy fails in the load-bearing respect. `encoding/json`'s only `recover` is
> `encode.go:335` in `(*encodeState).marshal`, and the `jsonError` it catches is raised by
> json's **own** `encodeState` recursion; a panic from a *user-supplied* `MarshalJSON` is
> explicitly **re-raised** at `encode.go:339`. So the precedent is "a package panics through its
> own frames", never "a package panics out of a callback it handed to foreign code" — which is
> exactly what is proposed here. The precedent does not cover the case.
>
> **What survives, unchanged:** drain mode itself, both named budgets, `drain_exhausted` trace
> metadata, the preserved typed `Internal` error, locality, and no interface change — i.e. every
> element of Mark's option-(c) ruling. **What does not survive:** a bound on post-failure
> wall-clock. That residual is flagged to Mark as the sprint's one open human question and is
> written into the LongDesc, not hidden.

Trace metadata uses two unambiguous counters: `provider_chunks` (all callbacks observed from the
provider, including callbacks after the fatal representation boundary, up to drain exhaustion)
and `delivered_chunks` (successfully encoded prefix length). `delivered_chunks == len(recorded)`
is invariant; the typed error and fatal index explain any difference from `provider_chunks`.
There is no `skipped_chunks` success path.

**Decision: option (b), fail the call loudly.** This preserves the current public ADT and prevents
a consumer from treating a partial log as complete. Option (a), a total `Unknown/Raw` variant, is
rejected for v0.32.0 because the provider interface supplies typed Go chunk values, not a canonical
raw wire payload; inventing serialization would not guarantee faithful replay and would enlarge the
public protocol. It can be reconsidered only with a provider-level opaque-byte contract. Option (c),
skip and call the log lossy, is rejected because it violates the repository's no-silent-fallback
principle and ADR-009's ordered-emission replay requirement. A diagnostic or counter cannot restore
the missing event.

### What "lossless" means — exact with respect to EMITTED chunks, not the wire

The author's most load-bearing post-quorum point (ailang#546, @arniwesth, after R2): chunks are
also lost ONE LAYER BELOW the encoder, in the provider adapters, invisibly to the effects layer.
All three cited sites were re-verified first-party in this worktree at `a929ec452`
(VERIFIED HERE; see Verification Log R3 rows):

1. `internal/ai/anthropic/streamstep.go:330-347` — the `switch ev.Delta.Type` has cases for
   `text_delta`, `input_json_delta`, `thinking_delta` and **no `default`**: a future Anthropic
   delta type is silently ignored and `onChunk` is never called for it.
2. Same switch, `:336-337` — `input_json_delta` is accumulated into the block's `inputJSON`
   builder and **deliberately not emitted** as a chunk; tool-call stream content is therefore
   already absent from any recorded log by design (it surfaces only as `ToolCalls` on the final
   response).
3. `cmd/wasm/effects.go:348-366` + `:242-245` — an unknown JS `kind` makes `jsToStreamChunk`
   return nil; the `if chunk != nil` guard filters it, so there is no callback, no record, and
   **no increment of `provider_chunks`** (that counter lives in the effects-layer `onChunk`,
   which is never invoked) — the loss is entirely invisible to the effects layer.

**Therefore the guarantee this design delivers is: the returned log is exact — complete and in
arrival order — with respect to the chunks the provider adapter EMITS through `onChunk`, not
with respect to the provider wire protocol.** The fail-loud invariant and the counters police
the encoder boundary and everything above it; they cannot see below it. Per the author's
explicit framing, moving the guard down into the adapters is NOT in scope — it would need the
opaque-byte provider contract that the option-(a) rejection above already declined. This
scoping MUST appear (adapted per audience) in three places: the builtin LongDesc, the
Success Criteria, and the A2 axiom justification — all updated in this revision.

### Sibling divergence on unencodable input — deliberate, not accidental

The EXISTING `aiStepWithStream` already skips an unencodable chunk silently — SHIPPED behavior,
VERIFIED HERE at `a929ec452`: `internal/effects/ai_step.go:377-382` increments `chunkCount`,
then `if encoded == nil { return }` — no callback, no trace event, stream continues. The
author's patch inherits that verbatim on purpose, and this doc's own Non-Goals forbid changing
`stepWithStream`'s behavior. Adopting fail-loud in the recorded op alone therefore means **the
two siblings behave DIFFERENTLY on identical provider input containing an unencodable chunk**:
the existing op silently drops it and reports success; the recorded op stops and returns typed
`Internal`. The equivalence "the recorded sibling is the existing op plus a log" holds ONLY for
streams in which every chunk is encodable — which, per the sealed-interface facts above, is
every stream constructible today. The divergence is thereby confined to the same
forward-compatibility scenario the drain bound guards (a fourth variant introduced without
encoder support), and the M4 exhaustiveness guard exists to keep that scenario unreachable.
This divergence is stated here deliberately (per the author: "either apply the invariant to
both, or state the divergence in the LongDesc"), MUST be stated in the builtin LongDesc, and
carries its own Risks row.

### Where the completeness contract lives — on the record, via `outcome` (no widening)

The author's objection: `provider_chunks`/`delivered_chunks` live in TRACE METADATA, which is
invisible at the AILANG level — a consumer holding `{chunks, outcome}` cannot read trace
metadata, so (the objection goes) it cannot tell a complete log from an incomplete one.

**Ruling: the discrepancy IS already surfaced on the returned record, and `RecordedStream` is
NOT widened.** The fail-loud invariant makes `outcome` itself the completeness signal, as a
machine-checkable case analysis on the record alone:

- `outcome = Ok(_)` → `chunks` is the complete ordered log of emitted chunks;
- `outcome = Err(e)` where `e` is a provider error → `chunks` is the complete ordered log of
  chunks emitted before the failure;
- `outcome = Err(e)` where `e` is the representation failure
  (`code=Internal`, message beginning with the stable prefix `unencodable stream chunk`) →
  `chunks` is an explicitly incomplete prefix.

Because a representation failure ALWAYS overrides the terminal result (§Count invariant, "cannot
be overwritten"), no fourth case exists in which the log is incomplete but `outcome` does not
say so — which is exactly what no-silent-fallback requires. The trace counters are redundant
diagnostics for dashboards, not the contract. **The message prefix `unencodable stream chunk`
is hereby part of the public contract** and gets its own test row; that is the one part of this
ruling that is genuinely weaker than a first-class discriminator. A dedicated `AIError` code
(e.g. `IncompleteStream`) would be cleaner but widens the public error vocabulary that existing
consumers may match exhaustively — that is a shape change requiring @arniwesth's sign-off, so
it is filed as an explicit follow-up question on #546 (NOT assumed), and `RecordedStream` keeps
exactly the shape the author's consumer code already targets. If the author asks for the code,
it is a small additive amendment, not a redesign.

### Exhaustiveness guard over the sealed variant set — IN SCOPE (M4)

The author's final point: neither (b) nor (c) addresses the only way the unencodable branch can
ever open — someone adds a fourth `StreamChunk` variant in `internal/ai` without updating
`encodeStreamChunk`, and Go does not complain because the type switch stays well-typed and the
new variant falls to `default` (`internal/effects/ai_step.go:447`, VERIFIED HERE). A
compile-time-or-CI exhaustiveness guard fails at the point the variant is INTRODUCED rather
than at runtime in someone's stream, and it protects the existing `stepWithStream` (whose
silent skip it makes unreachable again) as much as the recorded sibling.

**Ruling: IN SCOPE, as milestone M4 (0.5 day).** Reasoning made explicit rather than silently
absorbed: (i) this doc's own Conflict Surface item 4 already declared encoder exhaustiveness "a
production concern" — M4 is the implementation of a scope the doc already claimed, not new
scope; (ii) it is orthogonal and additive to Mark's (a)/(b)/(c) ruling, which concerned only
how to bound the drain, so taking it does not touch that ruling; (iii) it converts both the
drain path AND the sibling divergence from "guarded hazards" into "unreachable without a CI
failure first", which is the cheapest risk retirement in this doc. Mechanism (executor's
choice, both acceptable): a canonical `allStreamChunkVariants` registry slice in `internal/ai`
beside the variant declarations, with a test asserting `encodeStreamChunk(v) != nil` for every
entry, and/or a CI parity check comparing `streamChunkMarker()` implementer count against
`encodeStreamChunk` case count. @arniwesth has offered this as a separate small patch on #546 —
if that patch arrives first, M4 collapses to review-and-adopt with credit, which is the
preferred path.

### Documentation correction

Replace the false `std/ai.ail` wording with:

> The callback effect row is closed at `{IO}`. It supports immediate rendering only; callbacks
> requiring other effects such as `{FS}` or `{Net}` do not type-check. Use
> `stepWithStreamRecorded` when the exact ordered chunks must also be retained; its typed outcome
> explicitly reports any representation failure that makes the returned log incomplete.

Replace the example wording with:

> The callback is restricted to `{IO}` for immediate rendering. Other side-channel effects do
> not type-check; use `stepWithStreamRecorded` to receive an ordered chunk log after the call, and
> check its outcome before treating that log as complete.

Add a lightweight CI text guard (or a focused docs test) that fails if the streaming docs again
describe this callback row as “open.” Do not alter the historical changelog occurrence.

The builtin LongDesc for `stepWithStreamRecorded` MUST state all four of (per §What "lossless"
means and §Sibling divergence): (1) linear unbounded retention until the call returns; (2) the
log is exact with respect to chunks the provider adapter emits, not the wire — in particular,
tool-call `input_json` stream content is never emitted as chunks by design; (3) an unencodable
chunk terminates the op with typed `Internal` (stable message prefix `unencodable stream
chunk`) and an explicitly incomplete prefix; after that point AILANG records, encodes and
delivers nothing more, and its per-chunk work is bounded by `recordedDrainMaxChunks`/
`recordedDrainMaxBytes` — **but the call still returns only when the provider's stream ends,
so post-failure wall-clock is the provider's, not AILANG's**; (4) this
fail-loud behavior deliberately diverges from `stepWithStream`, which silently skips an
unencodable chunk — the siblings are equivalent only for fully-encodable streams (every stream
constructible today).

### Runtime and bytecode bridge

No dedicated bytecode op is needed. **VERIFIED HERE:** current evaluator-only bytecode functions
route through `runtime.CallEntrypoint`; both existing and offered builtins use the same
`RegisterEffectBuiltin` → `effects.Call(ctx, "AI", op, args)` → `RegisterOp` registry path. The
implementation must nevertheless add an end-to-end evaluator/`--bytecode` parity test for the new
result's nested record/list/ADT conversion. `--strict-bytecode` behavior stays identical to the
existing callback-bearing stream function.

### Stability, versioning, and credit

- Keep `StabilityExperimental`.
- Change metadata from `Since: "prototype"` to the actual release string `Since: "v0.32.0"`.
- Credit `@arniwesth` as original author in the adopting commit and changelog entry; preserve useful
  provenance in code comments without retaining “NOT an upstream feature” spike language.

## Productionization Milestones

> **⚠ SUPERSEDED AS AN EXECUTION PLAN (planner, iteration 133).** M1–M4 below remain the
> authoritative *acceptance* definitions, but they are **not** the sprint cut. At 4.5–5.75 days
> post-ruling they exceed the mission charter's 3–4 day sprint ceiling, so the work is split
> across two sprints. Execute from
> **[`m-recorded-stream-api-sprint-plan.md`](m-recorded-stream-api-sprint-plan.md)**, which
> re-cuts M1–M4 into five ≤1-day milestones for sprint 1 (merge-ready core) and a scoped
> sprint 2 (surface truth + exhaustiveness guard).

| Milestone | Estimate | Checkable acceptance |
|---|---:|---|
| **M1 — Shared core, fail-loud invariant, and bounded drain** | 2.0 days (was 2.5; −0.5 sentinel machinery dropped by the PLANNER RULING, +0.5 reallocated to the unbudgeted file-size split → see sprint plan S1-M1) | Patch adopted; both stream operations use one validation/decode/dispatch core; no duplicated block remains; capture precedes callback; partial provider-error chunks survive; first unencodable chunk deterministically yields typed `Internal` (stable message prefix `unencodable stream chunk`), no later delivery occurs, counters are explicit, and legacy behavior remains unchanged. **Drain bound (Mark's option (c), as ruled)**: post-failure drain mode with `recordedDrainMaxChunks`/`recordedDrainMaxBytes` budgets, entirely inside the recorded op; on exhaustion the drain goes **inert** (one boolean load per later callback — **no sentinel panic**, per §PLANNER RULING); the op returns the exact prefix with the original `Internal` error preserved, `drain_exhausted:true` traced, no interface change, no goroutine or timer created; a keeps-emitting fake handler stops being counted at exactly the budget and every later callback is observably inert |
| **M2 — Surface, bridge, and complete tests** | 1.0–1.5 days | `RecordedStream` and sibling registered as experimental since v0.32.0; offered four tests retained; added validation parity, callback-error, nil handler/`FnCaller`, every chunk variant, unencodable-first/middle plus provider-continues-after-failure cases (bounded per M1), drain-budget exhaustion for both budget dimensions, capability/budget/trace, registry/type-metadata, evaluator/bytecode parity, and direct ADR-009 integration tests pass |
| **M3 — Example, truth repair, and discovery/docs** | 1.0–1.5 days | `examples/ai_streaming_recorded.ail` checks with the freshly built target compiler and runs with `--ai-stub`; both false “open row” claims use the exact corrected wording above; CI guard added; teaching prompt and μRAG/builtin discovery entry updated; CHANGELOG and docs site updated; LongDesc states all four required items (retention, emitted-not-wire scope, fail-loud + bounded drain, sibling divergence); historical changelog untouched; @arniwesth credited |
| **M4 — StreamChunk exhaustiveness guard** | 0.5 days | Introducing a fourth `StreamChunk` variant in `internal/ai` without an `encodeStreamChunk` case fails CI/tests at introduction time (registry-slice test and/or marker-vs-case parity check); guard covers both siblings; if @arniwesth's offered patch lands first, M4 is review-and-adopt with credit |

Total (planner-revised, iteration 133): **4.5–5.75 days**. Trail: R3 raised 4.0–5.0 → 5.5–6.0
(+0.5 M1 drain machinery, +0.5 M4); the PLANNER RULING then removed the sentinel machinery
(−0.5 from M1) and the sprint plan added the **unbudgeted** `internal/effects/ai_step.go` split
(+0.5, absorbed into sprint 1). M1 is the merge-critical engineering work; the spike must not be
cherry-picked without it.

**This exceeds the mission charter's 3–4 day sprint ceiling, which is why it is two sprints.**
See [`m-recorded-stream-api-sprint-plan.md`](m-recorded-stream-api-sprint-plan.md) §1 for the cut
and for what the first PR may and may not claim.

## Test Plan

Retain the offered four tests:

1. success returns the same chunks delivered to the callback;
2. partial-stream failure returns preceding chunks with `Err`;
3. concatenated `ContentDelta` values equal final message content; and
4. the recorded variant does not change `stepWithStream`'s direct `Result` shape.

Merging additionally requires:

- table-driven parity for every argument/type/decode failure across both siblings;
- no-handler and no-`FnCaller` typed-error paths returning empty chunks for recorded mode;
- callback failure: callback error is traced, stream continues, and recorded chunks remain exact;
- `ContentDelta`, `ThinkingDelta`, and full `Usage` field ordering/encoding;
- unencodable first and middle chunks produce typed non-retryable `Internal` whose message
  carries the stable contract prefix `unencodable stream chunk`, return only the exact
  delivered prefix, never invoke the callback for the bad or later chunks, and cannot be
  overwritten by a later provider success/error; a provider that continues callbacks after the
  boundary is drained under the M1 budget while `provider_chunks` remains diagnostic and
  `delivered_chunks == len(recorded)`;
- drain-budget exhaustion, both dimensions independently: a fake handler emitting unbounded
  post-failure chunks stops being counted after exactly `recordedDrainMaxChunks` further
  callbacks; a fake handler emitting few oversized post-failure chunks trips
  `recordedDrainMaxBytes`; in both cases the op returns with the ORIGINAL `Internal` error,
  `drain_exhausted:true` is traced, and an instrumented fake asserts every post-exhaustion
  callback is inert (no encode, no append, no AILANG-callback invocation). **No
  sentinel-containment or panic-re-raise test exists — the PLANNER RULING dropped the
  sentinel**;
- **testability corollary (author's, verified here)**: because `StreamChunk` is sealed by
  unexported `streamChunkMarker()`, a test in `internal/effects` CANNOT declare a genuine
  fourth variant — the ONLY constructible inputs for every unencodable-chunk test above are a
  fake handler calling `onChunk(nil)` (an untyped nil hits the encoder's `default`), or a
  test-local `struct{ ai.StreamChunk }` embedding a nil interface (same effect; the marker
  method is never called by the type switch, so it does not panic). The fake-handler pattern
  already exists (`fakeStepHandler`, `internal/effects/ai_step_test.go:53`); no `onChunk(nil)`
  call site exists yet anywhere in the repo. Do not burn time looking for a third route — there
  is none;
- M4 exhaustiveness guard: adding a stub fourth variant in a guard-test fixture (or mutating
  the registry slice) fails the guard; removing the stub restores green;
- empty stream success and error; multiple chunks with stable order and no duplicate callback;
- capability denial and AI budget accounting parity;
- trace operation name, routing metadata, final outcome, fatal provider index, `provider_chunks`,
  and `delivered_chunks` assertions;
- builtin registry, public type, metadata (`v0.32.0`, experimental), and generated-doc coverage;
- `ailang check` of the new example with the freshly rebuilt compiler plus `--ai-stub` execution;
- evaluator versus non-strict `--bytecode` output parity for nested `RecordedStream`; and
- ADR-009's direct positive gate: immediate projection, exact ordered returned-log parity, success,
  partial-stream-then-error, and no duplicate delivery.

Run focused packages first, then `go test ./...` and `go build ./...`. Network-listener tests may
need the normal CI environment; this session's focused effects run was sandbox-blocked, not a
product failure.

## Conflict Surface

This change touches the type/effect/runtime path and must be rebased consciously against:

1. **`std/ai` public types and wrappers.** Concurrent edits to `Message`, `StreamChunk`,
   `StepResult`, cache breakpoints, structured-step options, or stability metadata can make the Go
   type builder diverge from AILANG declarations.
2. **Builtin registration and generated docs.** Name/arity/type collisions or missed init
   registration can leave a declared function uncallable or absent from prompt/μRAG discovery.
3. **Effect registry and callback integration.** `FnCaller`, capability/budget checks,
   `RecordAIEffect`, operation classification, and routing metadata must remain identical between
   both wrappers.
4. **Provider stream vocabulary.** Adding a `StreamChunk` variant makes `encodeStreamChunk`
   exhaustiveness a production concern; an unencodable value must take the typed fatal path and
   can never be skipped.
5. **Bytecode/evaluator interop.** The nested record/list/ADT result crosses eval↔bytecode value
   conversion on evaluator-only fallback. Strict-bytecode behavior and fallback diagnostics must
   not change.
6. **WASM AI bridge.** `cmd/wasm/effects.go` provides its own JS `StepWithStream` handler. The new
   op uses the same Go `AIHandler` method, but WASM build-tag tests must confirm callback and nested
   return compatibility; no second JS API should be invented unless tests prove it necessary.
7. **Tracing and telemetry consumers.** A new operation name and chunk-count meaning may affect
   dashboards, golden traces, budget reports, or exhaustive operation filters.
8. **Examples and documentation truth.** The old “open row” claim occurs in two live sources and
   one historical changelog. Fix only the live sources and guard against regression.
9. **Consumer ordering contract.** ADR-009 relies on append-before-terminal-transition and exact
   arrival order. Refactors that reconstruct content from `StepResult` or record after callback
   filtering violate the contract.
10. **Memory behavior.** Accumulating all chunks changes peak memory for callers selecting the new
    API; the existing API remains allocation-light and must not accidentally start retaining.

## Success Criteria

- [ ] Offered semantics are adopted, not redesigned, and original authorship is credited.
- [ ] Existing `stepWithStream` source/API/behavior remains unchanged externally.
- [ ] Recorded mode returns the complete exact ordered log **of adapter-emitted chunks** (the
      guarantee is scoped to the `onChunk` boundary, not the provider wire — see §What
      "lossless" means) on ordinary `Ok`/provider `Err`; an unencodable chunk instead returns
      typed `Internal` (stable message prefix `unencodable stream chunk`) with only the
      explicitly incomplete prefix.
- [ ] The post-failure drain is bounded by `recordedDrainMaxChunks`/`recordedDrainMaxBytes`
      entirely inside the recorded op: on exhaustion the drain goes inert, the original typed
      error is preserved, `drain_exhausted` is traced, and no interface change, no created
      goroutine/timer, and no panic crossing `onChunk` exists (Mark's option (c), 2026-08-03,
      as ruled by the planner in §PLANNER RULING — work is bounded, wall-clock is not).
- [ ] The deliberate sibling divergence on unencodable input is stated in the LongDesc, and the
      M4 exhaustiveness guard fails CI when a `StreamChunk` variant is introduced without an
      `encodeStreamChunk` case.
- [ ] Both siblings share one validation/decode/dispatch implementation.
- [ ] `delivered_chunks == len(recorded)` always; `provider_chunks` divergence is permitted only
      with the explicit typed representation failure and fatal index.
- [ ] Callback row remains closed `{IO}` and both false documentation claims are corrected.
- [ ] Experimental v0.32.0 metadata, memory warning, example, prompt/μRAG, changelog, and docs ship.
- [ ] Evaluator, bytecode fallback, WASM build path, focused tests, full Go tests/build, and the
      ADR-009 direct positive integration gate pass in an environment that permits their fixtures.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Shared-core refactor regresses existing streaming | High | Characterization tests before refactor; legacy wrapper parity and unchanged-shape test |
| Mid-stream error loses evidence | High | Keep `{chunks, outcome}`; partial-then-fail test is merge-blocking |
| New provider chunk is unencodable | High | Fail with typed non-retryable `Internal`; return only the explicit prefix; bounded post-failure drain (M1); M4 guard makes the scenario fail CI at variant introduction; exhaustive/future-variant tests |
| Provider cannot be cancelled after representation failure | Medium | Bounded drain mode (Mark's option (c) as ruled): chunk+byte budget inside the recorded op, drain goes inert on exhaustion, original error preserved, `drain_exhausted` traced. **Residual, stated not hidden**: post-failure wall-clock is still the provider's stream duration. Fix filed as a standalone follow-up (cancellable provider context across the whole AI surface), per @arniwesth #546 |
| ~~Sentinel abort unwinds through an adapter lacking `defer` cleanup~~ | ~~Low~~ | **RETIRED by the PLANNER RULING** — no sentinel is shipped, so no panic crosses the `onChunk` boundary and no `AIHandler` implementer (in-repo or external) acquires a panic-transparency obligation. The `defer`-hygiene audit is no longer needed |
| Siblings diverge on unencodable input (existing op skips silently — shipped `ai_step.go:377-382`; recorded op fails loud) | Medium | Deliberate: Non-Goals forbid changing `stepWithStream`; divergence stated in LongDesc and §Sibling divergence; confined to fully-unreachable-today input; M4 guard keeps it unreachable by failing CI at variant introduction |
| Retention exhausts memory on very long streams | Medium | Experimental marking and prominent linear/unbounded LongDesc; bounded API deferred |
| Callback and returned log diverge | High | Encode once, append before callback, compare identical values and order in tests |
| Docs reintroduce effect-row fiction | Medium | Focused CI text guard and checked examples |
| VM/WASM path differs from evaluator | Medium | Targeted bridge/build-tag tests rather than assuming registry uniformity |

## Related Documents

- [Recorded-stream pre-read](m-recorded-stream-api-preread.md) — attended evidence; not a quorum verdict.
- Motoko Project 009 `adr009.md` — deterministic test-world architecture blocked on this API.
- Motoko Project 007 `adr007.md` — accepted DST definition and scope taxonomy.
- [M-AI-STRUCTURED-STEP](m-ai-structured-step.md) — concurrent `std/ai`/builtin/effect work and a likely rebase conflict.
- M-AI-STEP-STREAMING (v0.18.7) — original streaming contract and parity requirements.

## Open Questions / Deferred Decisions

- A bounded, iterator, spill-to-file, or caller-supplied recorder is deferred; it would change
  authority and backpressure semantics.
- A future total `Unknown/Raw` variant is deferred until providers expose canonical opaque payload
  bytes; v0.32.0's required behavior is the fail-loud decision above, not an implementation choice.
- General callback effect-row polymorphism is explicitly outside this adoption and would need a
  separate language/effect-system design.
- **A cancellable provider context across the whole AI surface** (deadlines/cancellation on live
  streams, `AIHandler` v2). This is the honest fix for the residual the PLANNER RULING leaves
  open — post-failure wall-clock is the provider's, not AILANG's — and it is also the standing
  answer to R2's `gpt5-6-sol` objection. Deliberately NOT a dependency of this sprint: it is a
  7-implementer interface change reaching WASM, worth doing on its own merits (Mark's rejected
  option (b); @arniwesth's own recommendation, #546 2026-08-01). **File as a standalone issue
  during sprint 1 and link it from #546.**
- A first-class `AIError` code for the representation failure (e.g. `IncompleteStream`) instead
  of the stable message prefix on `Internal`: cleaner for consumers, but widens the public error
  vocabulary — filed as an explicit follow-up question to @arniwesth on #546 (see §Where the
  completeness contract lives); NOT assumed by this design.

## Quorum verification log

- **R1 verdict: BLOCKED.** Controller passed; `gpt5-6-sol` and `gemini-3-1-pro` blocked.
- **Objection 1:** skipping unknown/unencodable chunks contradicted the lossless contract,
  deterministic replay, and no-silent-fallback. **Resolved:** chose option (b): first encoding
  failure deterministically produces typed non-retryable `Internal`, returns an explicitly
  incomplete exact prefix, suppresses later delivery, and uses separate provider/delivered counts.
  Options (a) and (c), axioms A1/A2/A11, goals, milestones, tests, risks, and success criteria were
  updated consistently.
- **Objection 2:** the four offered tests were inherited rather than independently verified.
  **Resolved:** the Verification Log now records the controller's outside-sandbox isolated command
  and all four PASS lines, plus the lesson that unrelated sandbox port denial is uninformative and
  focused `-run` isolation is the remedy.

---

**Document created**: 2026-07-31
**Last updated**: 2026-08-03 (R3 revision — park cleared per Mark's option-(c) ruling; five
post-quorum author points folded in; see Quorum verification log §R3)

### R2 — the re-quorum (appended by the controller, iteration 124)

- **R2 verdict: BLOCKED.** Controller passed; both model reviewers blocked again. Metered cost
  R1 $0.0508 + R2 $0.0578 = **$0.1086** total, against the $5 ceiling.
- **`gemini-3-1-pro` — RESOLVED, no design change needed.** It rejected two premises as
  unverified: the syntax check had been run via `ailang prompt --version v0.16.2` rather than the
  v0.31.0 target, and "the patch does not break existing effects tests" rested on the four NEW
  tests alone (the controller's own earlier `-run` isolation was too narrow — a fair hit). Its
  proposed fix was executed verbatim by the controller, outside the sandbox:
  - `go test ./internal/effects -skip TestNetHTTPRequestBytes_RoundTripSHA` → **rc=0, 658 `--- PASS:`
    lines, 23.09s.** The patch breaks nothing in the existing effects suite.
  - `ailang check examples/runnable/ai_streaming.ail` on the freshly built
    `v0.31.0-31-g130ad1da2` → **rc=0, "✓ No errors found!"**.
  Both recorded here so the next iteration does not re-litigate them.
- **`gpt5-6-sol` — NOT resolved; this is the park.** See "⚠ THE PARK" at the top of this doc. Its
  conditional clause was checked against the code and fires: `AIHandler.StepWithStream` takes no
  `context.Context`, so the fix it prescribes requires a 7-implementer provider-interface change
  (including `cmd/wasm/effects.go`). That contradicts the "purely additive" property the ADOPT
  verdict rests on, so it is a scope decision for a human rather than a narrow refinement the
  controller may apply.
- **Not force-passed, not silently downgraded.** Standing rule 2: park and report. The design
  direction was never contested by either reviewer in either round — what is unresolved is
  exclusively how to bound the drain.

### R3 — Mark's ruling and the author-response revision (appended iteration 133, 2026-08-03)

- **The park is CLEARED — no quorum rerun.** This is a bounded revision folding in a human
  ruling plus author feedback, not a redesign; the twice-surviving design direction is untouched
  per the revision charter. R1/R2 sections above are preserved verbatim as history.
- **Mark's ruling (attended, 2026-08-03): OPTION (c)** — bound the drain locally inside the
  recorded op, chunk/byte budget, no interface change. Independently endorsed by @arniwesth
  (#546, 2026-08-01) on sealed-interface grounds: the fail-loud drain trigger is unreachable for
  any value constructible today (VERIFIED BY CONTROLLER at `a929ec452`, four legs). **Option (b)
  REJECTED**: a 7-implementer interface change reaching WASM forfeits the purely-additive
  property the ADOPT verdict rests on, to cancel a path with no reachable trigger. The stale
  "do not route to sprint-planner" sentence was removed from the Status header.
- **Five post-quorum author points (@arniwesth, #546) folded in:**
  1. Concrete drain bound specified (§Count invariant and drain bound): drain mode,
     `recordedDrainMaxChunks`/`recordedDrainMaxBytes`, sentinel-panic abort scoped to the
     recorded op, original error preserved, `drain_exhausted` traced, merge-blocking acceptance
     row in M1. **[SUPERSEDED by the R4 planner ruling below — the sentinel-panic abort is
     cancelled; everything else in this point stands.]**
  2. Guarantee honestly scoped to EMITTED chunks, not the wire (§What "lossless" means) — all
     three adapter-loss sites re-verified first-party here (no-default delta switch `:330-347`,
     unemitted `input_json_delta` `:336-337`, WASM nil-filter `:242-245`/`:348-366`); wording
     landed in LongDesc requirements, Success Criteria, and A2.
  3. Sibling divergence on unencodable input made deliberate and documented (§Sibling
     divergence; `ai_step.go:377-382` verified) with LongDesc requirement and Risks row —
     forced by this doc's own Non-Goals, as the author anticipated.
  4. Completeness contract ruled to live ON the record via `outcome` (§Where the completeness
     contract lives): no `RecordedStream` widening, stable message prefix
     `unencodable stream chunk` promoted to public contract, first-class `IncompleteStream`
     code filed as a follow-up question to the author, not assumed.
  5. Exhaustiveness guard ruled IN SCOPE as new M4 (0.5 day) — implementation of Conflict
     Surface item 4's already-claimed concern; orthogonal to Mark's (a)/(b)/(c) ruling; if the
     author's offered patch lands first, M4 becomes review-and-adopt with credit.
- **Testability corollary recorded in the Test Plan**: the seal means the only constructible
  unencodable inputs are `onChunk(nil)` from a fake handler or a test-local
  `struct{ ai.StreamChunk }` nil-embed — noted so the executor does not burn hours discovering
  it.
- **Estimate updated honestly: 4.0–5.0 → 5.5–6.0 days** (+0.5 drain machinery in M1, +0.5 M4).

### R4 — the planner ruling and the sprint cut (appended iteration 133, 2026-08-03, sprint-planner)

- **The sentinel-panic abort is CANCELLED; drain mode ships.** Option (i) of the controller's
  refutation. Four reasons, with commands and outputs, in §PLANNER RULING above: Go 1.26.5's own
  `syscall/js.FuncOf` contract (new goroutine unless the callback fires *during* a Go→JS call —
  and the WASM streaming path is Promise-async by design); option (iii) refuted; option (ii)
  refuted because PR CI never compiles `js && wasm` (`ci.yml`/`build.yml` → rc=1 zero hits; only
  `release.yml:17`) *and* because on WASM the drain branch is unreachable by construction
  (`jsToStreamChunk` returns only encodable variants or a filtered untyped nil); and R4, that a
  panic through `onChunk` is a covert version of the 7-implementer contract change Mark rejected
  as option (b).
- **One doc argument REFUTED**: the `encoding/json`/`fmt` precedent. json's only `recover`
  (`encode.go:335`) catches a `jsonError` raised by json's *own* recursion and **re-raises**
  user-callback panics at `:339`. It is not a precedent for panicking out of a callback handed to
  foreign code.
- **What Mark's option (c) keeps**: locality, no interface change, both named budgets,
  `drain_exhausted` trace metadata, the preserved typed `Internal` error. **What it loses**: a
  bound on post-failure wall-clock. Flagged to Mark as the sprint's single one-word human
  question, and written into the LongDesc rather than hidden.
- **Cut into two sprints** (4.5–5.75 d exceeds the charter's 3–4 d ceiling). Sprint 1 = five
  ≤1-day milestones: the unbudgeted `internal/effects/ai_step.go` split (22 lines of headroom
  measured against the 800-line gate), verbatim adoption + shared core, fail-loud + inert drain,
  the test matrix, and the merge-blocking contract text (LongDesc / `Since: "v0.32.0"` /
  CHANGELOG / @arniwesth credit). Sprint 2 = example + manifest, the two doc-truth repairs + CI
  guard, discovery/website, and M4.
- **M4 status checked, not assumed**: `gh issue view 546 --json comments` on 2026-08-03 → 4
  comments, the last ours at 08:20Z. @arniwesth **offered** the exhaustiveness guard on
  2026-08-01 but has **not** posted a patch. Budget it as write-from-scratch; re-check at the
  start of sprint 2.
- **Every controller fact re-verified first-party at `2f12ddacd`**, all confirmed: patch applies
  (rc=0) and reverses (rc=1); 4/4 offered tests PASS under `SKIP_NET_TESTS=1`; post-patch
  `internal/effects/ai_step.go` = **778** lines with `make check-file-sizes` green; the stale
  `v0.31.0` target strings existed and are now swept to **v0.32.0**.
- **Execution plan**: [`m-recorded-stream-api-sprint-plan.md`](m-recorded-stream-api-sprint-plan.md)
  and `.ailang/state/sprints/sprint_M-RECORDED-STREAM-API-S1.json`.
