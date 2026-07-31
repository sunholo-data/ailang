# M-RECORDED-STREAM-API — Productionize the offered recorded `std/ai` stream

**Status**: **PARKED — `needs-human-review` (mission iteration 124, 2026-07-31).** Quorum BLOCKED
twice (R1 and R2). The design DIRECTION — adopt @arniwesth's additive sibling — survived both
rounds and is **not** what is contested. Do **not** route to sprint-planner until the scope
question below is answered. See "Quorum verification log".
**Target**: v0.31.0
**Priority**: P0 (blocking external consumer architecture)
**Estimated**: 4–5 engineering days, **plus** whatever the cancellation decision below costs
**Dependencies**: external implementation offered in ailang#546 by @arniwesth; Motoko Project 009 ADR-001 is formally blocked on this API

> ## ⚠ THE PARK — one decision is needed, and it is a scope call, not a detail
>
> R2's surviving objection (`gpt5-6-sol`) is that the fail-loud path has an **unbounded wait**:
> after an unencodable chunk the design says "drain safely" if the provider cannot be cancelled,
> but specifies no deadline, cancellation mechanism, or budget. Its proposed fix is explicit —
> enforce a stream deadline and a finite chunk/byte budget "propagated through a cancellable
> provider context", and, verbatim: *"If the current `AIHandler.StepWithStream` cannot be
> cancelled or bounded, make that provider-interface change a blocking dependency rather than
> draining indefinitely."*
>
> **That conditional FIRES. Verified first-party by the controller at HEAD `130ad1da2`:**
> `internal/effects/ai.go:87` declares
> `StepWithStream(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error)`
> — **no `context.Context`, so the provider call cannot be cancelled.** (Known-positive control:
> the same search finds `context.Context` freely elsewhere in the repo, e.g. `internal/notify/`,
> so this absence is real, not a broken grep.) There are **7 implementers across 6 files**,
> including `cmd/wasm/effects.go`, so adding cancellation is a cross-cutting interface change that
> reaches the WASM target.
>
> **Why this was NOT resolved under the narrow-refinement carve-out:** that carve-out covers only
> objections leaving the design DIRECTION intact. Adding cancellation to `AIHandler` destroys the
> "purely additive, no existing line modified" property — precisely the argument this doc's ADOPT
> verdict and its 1.x-stability claim rest on. Choosing that scope is controller judgement, and
> Standing rule 2 says park rather than force it through.
>
> **The question for a human (and for @arniwesth):** do we
> **(a)** land the recorded sibling now with a documented unbounded-drain caveat and file
> cancellation as a follow-up; **(b)** take the `AIHandler` cancellation change as a blocking
> dependency first, accepting a 7-implementer interface change including WASM; or
> **(c)** bound the drain locally inside the recorded op (chunk/byte budget, no interface change),
> capping the damage without making the provider truly cancellable?
> The controller's read is that **(c)** is the cheapest honest answer and **(a)** is the one to
> avoid — but this is a scope decision for Mark, not for the loop.

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

## Axiom Compliance

| Axiom | Score | Justification |
|---|---:|---|
| A1 Determinism | +1 | Returns the exact encodable prefix in arrival order and deterministically terminates with typed `Internal` failure at the first unencodable chunk; it never skips that chunk and continues successfully |
| A2 Replayability | +2 | A non-encoding-error outcome certifies a complete log; an encoding error explicitly marks the returned prefix incomplete, so replay never mistakes partial evidence for a lossless transcript |
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
- Bounding streams, spilling chunks to disk, backpressure, resumable streams, or concurrency.
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

Retention is intentionally unbounded for v0.31.0: all chunks remain in memory until the call
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

### Count invariant

The patch increments `chunkCount` before `encodeStreamChunk`, but appends only after a non-nil
encoding. An unknown/skipped provider chunk can therefore produce trace `chunks:N` while
`len(recorded) < N`. Production code MUST NOT skip that chunk or continue as though the log were
complete. At the first nil/unknown encoding, the shared core records no value for that chunk,
invokes no callback for it, suppresses delivery and recording of later chunks if the provider
cannot be cancelled, and overrides any provider terminal result with a non-retryable typed
`AIError{code=Internal, message="unencodable stream chunk at provider index N; recorded log is an incomplete prefix"}`.
The returned `chunks` are exactly the values already delivered before the failure.

Trace metadata uses two unambiguous counters: `provider_chunks` (all callbacks observed from the
provider, including callbacks after the fatal representation boundary) and `delivered_chunks`
(successfully encoded prefix length). `delivered_chunks == len(recorded)` is invariant; the typed
error and fatal index explain any difference from `provider_chunks`. There is no `skipped_chunks`
success path.

**Decision: option (b), fail the call loudly.** This preserves the current public ADT and prevents
a consumer from treating a partial log as complete. Option (a), a total `Unknown/Raw` variant, is
rejected for v0.31.0 because the provider interface supplies typed Go chunk values, not a canonical
raw wire payload; inventing serialization would not guarantee faithful replay and would enlarge the
public protocol. It can be reconsidered only with a provider-level opaque-byte contract. Option (c),
skip and call the log lossy, is rejected because it violates the repository's no-silent-fallback
principle and ADR-009's ordered-emission replay requirement. A diagnostic or counter cannot restore
the missing event.

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

### Runtime and bytecode bridge

No dedicated bytecode op is needed. **VERIFIED HERE:** current evaluator-only bytecode functions
route through `runtime.CallEntrypoint`; both existing and offered builtins use the same
`RegisterEffectBuiltin` → `effects.Call(ctx, "AI", op, args)` → `RegisterOp` registry path. The
implementation must nevertheless add an end-to-end evaluator/`--bytecode` parity test for the new
result's nested record/list/ADT conversion. `--strict-bytecode` behavior stays identical to the
existing callback-bearing stream function.

### Stability, versioning, and credit

- Keep `StabilityExperimental`.
- Change metadata from `Since: "prototype"` to the actual release string `Since: "v0.31.0"`.
- Credit `@arniwesth` as original author in the adopting commit and changelog entry; preserve useful
  provenance in code comments without retaining “NOT an upstream feature” spike language.

## Productionization Milestones

| Milestone | Estimate | Checkable acceptance |
|---|---:|---|
| **M1 — Shared core and fail-loud invariant hardening** | 2.0 days | Patch adopted; both stream operations use one validation/decode/dispatch core; no duplicated block remains; capture precedes callback; partial provider-error chunks survive; first unencodable chunk deterministically yields typed `Internal`, no later delivery occurs, counters are explicit, and legacy behavior remains unchanged |
| **M2 — Surface, bridge, and complete tests** | 1.0–1.5 days | `RecordedStream` and sibling registered as experimental since v0.31.0; offered four tests retained; added validation parity, callback-error, nil handler/`FnCaller`, every chunk variant, unencodable-first/middle plus provider-continues-after-failure cases, capability/budget/trace, registry/type-metadata, evaluator/bytecode parity, and direct ADR-009 integration tests pass |
| **M3 — Example, truth repair, and discovery/docs** | 1.0–1.5 days | `examples/ai_streaming_recorded.ail` checks with the freshly built target compiler and runs with `--ai-stub`; both false “open row” claims use the exact corrected wording above; CI guard added; teaching prompt and μRAG/builtin discovery entry updated; CHANGELOG and docs site updated; LongDesc states linear unbounded retention; historical changelog untouched; @arniwesth credited |

Total: **4.0–5.0 days**. M1 is the merge-critical engineering
work; the spike must not be cherry-picked without it.

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
- unencodable first and middle chunks produce typed non-retryable `Internal`, return only the exact
  delivered prefix, never invoke the callback for the bad or later chunks, and cannot be overwritten
  by a later provider success/error; a provider that continues callbacks after the boundary is
  drained safely while `provider_chunks` remains diagnostic and
  `delivered_chunks == len(recorded)`;
- empty stream success and error; multiple chunks with stable order and no duplicate callback;
- capability denial and AI budget accounting parity;
- trace operation name, routing metadata, final outcome, fatal provider index, `provider_chunks`,
  and `delivered_chunks` assertions;
- builtin registry, public type, metadata (`v0.31.0`, experimental), and generated-doc coverage;
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
- [ ] Recorded mode returns the complete exact ordered log on ordinary `Ok`/provider `Err`; an
      unencodable chunk instead returns typed `Internal` with only the explicitly incomplete prefix.
- [ ] Both siblings share one validation/decode/dispatch implementation.
- [ ] `delivered_chunks == len(recorded)` always; `provider_chunks` divergence is permitted only
      with the explicit typed representation failure and fatal index.
- [ ] Callback row remains closed `{IO}` and both false documentation claims are corrected.
- [ ] Experimental v0.31.0 metadata, memory warning, example, prompt/μRAG, changelog, and docs ship.
- [ ] Evaluator, bytecode fallback, WASM build path, focused tests, full Go tests/build, and the
      ADR-009 direct positive integration gate pass in an environment that permits their fixtures.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Shared-core refactor regresses existing streaming | High | Characterization tests before refactor; legacy wrapper parity and unchanged-shape test |
| Mid-stream error loses evidence | High | Keep `{chunks, outcome}`; partial-then-fail test is merge-blocking |
| New provider chunk is unencodable | High | Fail with typed non-retryable `Internal`; return only the explicit prefix; suppress later delivery; exhaustive/future-variant tests |
| Provider cannot be cancelled after representation failure | Medium | Drain safely, suppress subsequent callbacks, preserve first fatal error, and expose provider/delivered counters; future cancellable provider API may reduce wasted work |
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
  bytes; v0.31.0's required behavior is the fail-loud decision above, not an implementation choice.
- General callback effect-row polymorphism is explicitly outside this adoption and would need a
  separate language/effect-system design.

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
**Last updated**: 2026-07-31

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
