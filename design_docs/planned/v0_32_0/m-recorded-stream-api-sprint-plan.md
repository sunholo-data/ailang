# Sprint Plan — M-RECORDED-STREAM-API (sprint 1 of 2)

**Design doc**: [`m-recorded-stream-api.md`](m-recorded-stream-api.md) (R3, quorum-cleared, Mark's
option-(c) ruling folded in, planner ruling on the sentinel appended)
**Sprint ID**: `M-RECORDED-STREAM-API-S1`
**Branch**: `sprint/m-recorded-stream-api` (worktree `.wt-iter133`, branched from `origin/dev` at `a929ec452`)
**Target version**: v0.32.0
**GitHub issue**: [#546](https://github.com/sunholo-data/ailang/issues/546)
**Original author of the adopted implementation**: **@arniwesth** — credit is mandatory in the
adopting commit and the CHANGELOG entry.
**Planner**: opus, mission iteration 133
**Executor**: `codex:gpt-5.6-sol` under `--sandbox workspace-write`
**Estimate**: **3.75 days** (range 3.5–4.0) · **risk: medium**

---

## 0. Read this before touching code

### 0.1 The sandbox denies loopback socket binds

Any `httptest`/net-listener test that fails with a bind/permission error under the executor's
sandbox is **UNINFORMATIVE UNDER SANDBOX — not a product failure**. Label it exactly that way in
the progress notes and move on; do **not** report it as a red gate. The controller re-runs all
gates outside the sandbox. The known instance in this area is
`TestNetHTTPRequestBytes_RoundTripSHA` in `internal/effects`.

**Always run Go gates with `SKIP_NET_TESTS=1`.** There is a separate live-network flake filed as
**#561** (`TestNetHttpPost/httpPost_to_httpbin.org` fails on an upstream httpbin 503; the test
tolerates transport errors but not a 5xx). Do not chase it.

Isolate with `-run` when you want a specific answer; **the narrowing travels with the finding**.
If you write "tests pass", say which package and which `-run`/`-skip` filter produced it.

### 0.2 The file-size gate is the tightest constraint in this sprint

`make check-file-sizes` fails any file in `internal/` or `cmd/` above **800 lines**
(`make/code-health.mk:127`). Measured first-party by the planner at `2f12ddacd`:

| file | HEAD | after `/tmp/arni-546-dev.patch` | headroom |
|---|---:|---:|---:|
| `internal/effects/ai_step.go` | 665 | **778** | **22 lines** |
| `internal/builtins/ai_step.go` | 550 | 619 | 181 |
| `std/ai.ail` | 446 | 471 | n/a (not gated) |

Verified: patch applied → `wc -l internal/effects/ai_step.go` = **778**; `make check-file-sizes` →
green ("All files within 800 line limit"); patch reversed → tree clean.

**22 lines is not enough for the shared core, the drain bound, the counters and the trace
plumbing this sprint adds.** That is why **S1-M1 (the split) is the FIRST milestone, not a
cleanup afterthought**. This is the same wall that stopped `m-property-generator-coverage`
Lane B1 at 790/800.

### 0.3 The reference implementation

`/tmp/arni-546-dev.patch` (20,402 B; +452 / 5 files / **0 deletions**). Verified first-party by
the planner in this worktree at `2f12ddacd`:

- `git apply --check` → **rc=0**; `git apply --check --reverse` → **rc=1** (non-vacuous control).
- Applied, then `SKIP_NET_TESTS=1 go test ./internal/effects/ -run 'Recorded|StreamRecorded' -v`
  → **rc=0, 4/4 PASS**, `ok github.com/sunholo-data/ailang/internal/effects 0.413s`.
- Reversed; `git status --porcelain` empty.

Adopt it as the base of S1-M2. **Do not rewrite it.** The design doc's verdict is ADOPT, not
reinvent, and it was reaffirmed through two quorum rounds.

### 0.4 The sentinel-panic abort is CANCELLED

The design doc originally specified a sentinel-panic abort on drain-budget exhaustion. The
planner **ruled it out** — see `§PLANNER RULING` in the design doc for the four reasons (Go's
`js.FuncOf` per-goroutine `recover` contract; the async-Promise WASM path; PR CI never compiling
`js && wasm`; and the covert `AIHandler` contract change).

**Do not implement a panic/recover anywhere in this sprint.** On budget exhaustion the drain goes
**inert**: one `drainExhausted bool`, checked as the first statement of the callback. There is no
sentinel type, no `recover`, no containment test, no build-tag split.

### 0.5 Testability corollary — do not go looking for a third route

`ai.StreamChunk` is sealed by the unexported `streamChunkMarker()` (`internal/ai/provider.go:159-163`).
A test in `internal/effects` **cannot** declare a genuine fourth variant. The only two
constructible unencodable inputs are:

1. a fake handler calling `onChunk(nil)` (untyped nil hits `encodeStreamChunk`'s `default` at
   `internal/effects/ai_step.go:447`); or
2. a test-local `struct{ ai.StreamChunk }` embedding a nil interface (same effect; the marker
   method is never called by the type switch, so it does not panic).

The fake-handler pattern already exists: `fakeStepHandler`, `internal/effects/ai_step_test.go:53`.
No `onChunk(nil)` call site exists anywhere in the repo yet. **There is no third route. Do not
burn hours looking.**

---

## 1. The sprint cut, and why

The design doc's M1–M4 total **4.5–5.75 days** after the sentinel ruling. The mission charter caps
a sprint at **3–4 days**. So the work is two sprints.

### Sprint 1 — "merge-ready recorded stream core" (this plan, 3.75 d)

Everything that must be true for the builtin to be *mergeable and honestly self-describing*:
the file split, the adopted patch on a shared core, the fail-loud invariant with the bounded
inert drain, the full Go test matrix, and the contract text (LongDesc / `Since:` metadata /
CHANGELOG / credit).

### Sprint 2 — "surface truth + exhaustiveness guard" (~1.5–2.0 d, separate plan)

`examples/ai_streaming_recorded.ail` + manifest regen + `--ai-stub` run; the two false "open row"
doc claims + the CI text guard; prompt/μRAG discovery + generated builtin docs + website;
and the M4 `StreamChunk` exhaustiveness guard.

### Why this boundary and not the doc's M1+M2 | M3+M4

Three deliberate moves off the doc's milestone lines:

1. **The file split (S1-M1) is new work the doc never budgeted.** It costs 0.5 d and it is
   load-bearing; without it the shared-core refactor hits the 800-line wall mid-sprint. The 0.5 d
   the sentinel ruling freed from M1 is reallocated here — the sprint neither grew nor shrank.
2. **`Since:` metadata, LongDesc, CHANGELOG and credit move UP from doc-M3 into sprint 1.** You
   cannot merge a builtin registered `Since: "prototype"`; CHANGELOG is required for every change
   (`.claude/rules/coding-standards.md`); and shipping a *fail-loud* API whose deliberate
   divergence from its sibling is undocumented is precisely the silent-fallback shape this repo
   forbids. These are contract text, not documentation polish.
3. **The example, discovery, website and the M4 guard stay in sprint 2.** They are real and
   required before the feature is "done", but none of them changes the merged semantics, and
   bundling them pushes sprint 1 past the charter ceiling.

### What the sprint-1 PR CAN honestly claim

- `std/ai.stepWithStreamRecorded` exists, is registered, typed, `StabilityExperimental`,
  `Since: "v0.32.0"`, and returns `{chunks, outcome}` exactly as @arniwesth specified.
- Both stream operations share **one** validation/decode/dispatch core; no duplicated block remains.
- The fail-loud invariant holds with the stable public message prefix `unencodable stream chunk`;
  `delivered_chunks == len(recorded)` is invariant; the post-failure drain is bounded in
  per-chunk work and retains nothing.
- The existing `stepWithStream` is externally unchanged (parity test).
- Evaluator ↔ non-strict `--bytecode` parity for the new nested `RecordedStream`.
- @arniwesth is credited as original author.

### What the sprint-1 PR MUST NOT claim

- ❌ **"ADR-009 / Motoko Project 009 is unblocked."** Sprint 1 passes a *Go-level* ordering gate.
  The external consumer probe is @arniwesth's, against a merged build. Say "the substrate is
  merged; the consumer probe is theirs to run."
- ❌ **"The streaming docs no longer lie."** Both false "the callback's effect row is open" claims
  (`std/ai.ail:324`, `examples/runnable/ai_streaming.ail:40-42`) are still live after sprint 1.
- ❌ **"There is a worked example."** No `examples/ai_streaming_recorded.ail` yet; `ailang prompt`
  and μRAG discovery do not know the function exists.
- ❌ **"An unencodable `StreamChunk` variant is impossible."** That is the M4 guard, sprint 2. The
  correct sprint-1 sentence is: *unreachable for any `StreamChunk` constructible today (sealed
  interface); not yet unreachable-by-CI at variant-introduction time.*
- ❌ **"The post-failure drain terminates in bounded wall-clock."** It terminates when the
  provider's stream ends. Bounded *work*, not bounded *time*.
- ❌ **Anything at all about the `js && wasm` build.** PR CI does not compile it
  (`grep -n wasm .github/workflows/ci.yml .github/workflows/build.yml` → rc=1, zero hits; the only
  `build-wasm` job is `release.yml:17`).

---

## 2. Milestones

Five milestones, each ≤ 1 day. Commit at each boundary — concurrent agents share this repo and a
long-lived dirty tree is a hazard. Use `refs #546` on every commit except the last of sprint 2.

---

### S1-M1 — Split `internal/effects/ai_step.go` for headroom (0.5 d, ~0 net LOC)

**Behaviour-free. This must be the first commit and must contain no logic change.**

Move, verbatim, out of `internal/effects/ai_step.go`:

| new file | move these (current line refs at `a929ec452`) | approx |
|---|---|---:|
| `internal/effects/ai_decode.go` | `decodeMessages` (:459), `decodeToolCalls` (:494), `decodeImageParts` (:514), `decodeCacheBreakpoints` (:537), `decodeToolSchemas` (:557), `getStringField` (:576) | ~135 |
| `internal/effects/ai_encode.go` | `encodeStreamChunk` (:422), `makeOkStringResult` (:593), `makeOkStepResult` (:601), `encodeToolCalls` (:629), `makeAIErrorResultRecord` (:646) | ~110 |

Leaves `ai_step.go` at roughly **420 lines**, restoring ~380 lines of headroom before anything is
added.

**Acceptance**
- [ ] `git diff -M --stat` for this commit shows moves only — no changed function bodies. State
      the diff summary in the progress notes.
- [ ] `SKIP_NET_TESTS=1 go test ./internal/effects/ ./internal/builtins/` → rc=0.
- [ ] `make check-file-sizes` → green.
- [ ] `wc -l internal/effects/ai_step.go` ≤ 450.
- [ ] `go build ./...` → rc=0.

**Risks**: none material. Import cycles are impossible (same package). Watch for a helper that is
also referenced from `ai.go` — same package, so it still resolves.

---

### S1-M2 — Adopt @arniwesth's patch onto a shared core (1.0 d, ~200 net LOC)

**Step 1 — adopt.** `git apply /tmp/arni-546-dev.patch`. Commit this *alone*, before refactoring,
with @arniwesth credited:

```
feat(std/ai): adopt stepWithStreamRecorded reference implementation (#546)

Original author: @arniwesth (arniwesth/ailang#2). Adopted verbatim as the
base for the shared-core refactor that follows.

refs #546
Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
```

A verbatim adoption commit makes the subsequent refactor reviewable as a diff against the
author's own code, and makes the credit legible in `git log`.

**Step 2 — shared core.** New file `internal/effects/ai_stream_core.go`. Extract the ~80
duplicated lines (arity/type validation, handler + `FnCaller` guards, message/tool-schema/
cache-breakpoint decode, the single `ctx.AI.StepWithStream` invocation, chunk encode, callback
delivery + fail-soft callback-error tracing, typed operation-error classification, final response
conversion, trace metadata/count construction) into one private core taking an explicit
collection/error policy:

```go
type streamRecordPolicy struct {
    record   bool // append encoded chunks to a returned slice
    failLoud bool // first unencodable chunk => typed Internal, suppress later delivery
}
```

`aiStepWithStream` = core + `{record:false, failLoud:false}`, returning the existing
`Result[StepResult, AIError]` shape. `aiStepWithStreamRecorded` = core + `{record:true,
failLoud:true}`, returning `{chunks, outcome}`.

**Acceptance**
- [ ] `grep -c` shows exactly one copy of the argument-validation and decode blocks across
      `aiStepWithStream` and `aiStepWithStreamRecorded` — quote the command and the count.
- [ ] The four adopted tests still pass: `SKIP_NET_TESTS=1 go test ./internal/effects/ -run 'Recorded|StreamRecorded' -v` → 4/4.
- [ ] `TestAIStepWithStream_UnchangedByRecordedVariant` passes.
- [ ] `SKIP_NET_TESTS=1 go test ./internal/effects/ ./internal/builtins/` → rc=0.
- [ ] `make check-file-sizes` → green; no file in `internal/effects` above 700.
- [ ] `internal/builtins/ai.go` + `ai_step.go` registration unchanged in behaviour;
      `make build && ./bin/ailang doctor builtins` (with `AILANG_BUILTINS_REGISTRY=1`) → rc=0.

**Risks**: the core has two return shapes. Prefer returning `([]eval.Value, *ai.Response, *ai.AIError)`
from the core and letting each wrapper build its own `eval.Value`, rather than threading a
constructor callback — simpler to read and it keeps the two public shapes obviously separate.

---

### S1-M3 — Fail-loud invariant + bounded inert drain (0.75 d, ~120 LOC)

Implement in the core, active only under `failLoud`:

1. **First unencodable chunk** (`encodeStreamChunk` returns nil): record nothing for it, invoke no
   AILANG callback for it, and latch a non-retryable typed
   `AIError{code: Internal, message: "unencodable stream chunk at provider index N; recorded log is an incomplete prefix"}`.
   **The prefix `unencodable stream chunk` is part of the public contract** — put it in a named
   constant and assert on it.
2. **The latched error cannot be overwritten** by a later provider terminal success or error.
   First representation failure decides `outcome`; everything after is diagnostics.
3. **Drain mode** — after the latch, the callback does no encoding, no recording, no delivery. It
   advances two counters: post-failure chunk count and post-failure payload bytes (sum of
   text/JSON field lengths; no retention).
4. **Budgets**, named constants in `internal/effects`:
   `recordedDrainMaxChunks = 256`, `recordedDrainMaxBytes = 1 << 20`. Whichever exhausts first
   sets `drainExhausted = true`.
5. **On exhaustion the drain goes INERT**: the callback's first statement is
   `if drainExhausted { return }`. Counters saturate. **No panic. No `recover`. No sentinel.**
   (See §0.4 and the design doc's `§PLANNER RULING`.)
6. **Counters**: `provider_chunks` (all callbacks observed, a *floor* once `drain_exhausted`) and
   `delivered_chunks` (encoded prefix length, `== len(recorded)` always). Trace metadata gains
   `drain_exhausted: true` when the budget tripped. There is **no** `skipped_chunks` success path.

**Explicitly out of scope**: any change to `aiStepWithStream`'s behaviour. It keeps its shipped
silent skip (`internal/effects/ai_step.go:377-382` at `a929ec452`). The divergence is deliberate
and gets documented in S1-M5.

**Acceptance**
- [ ] Unencodable first chunk → `outcome = Err`, `code = Internal`, message begins with the
      constant; `chunks` is empty; the AILANG callback was never invoked for it.
- [ ] Unencodable middle chunk → `chunks` is exactly the delivered prefix; later chunks are
      neither recorded nor delivered.
- [ ] A fake handler emitting 1000 post-failure chunks: exactly `recordedDrainMaxChunks` are
      counted, `drain_exhausted:true` is traced, and an instrumented fake asserts **zero**
      encodes / appends / AILANG-callback invocations after exhaustion.
- [ ] A fake handler emitting 3 post-failure chunks of 512 KiB each trips `recordedDrainMaxBytes`
      independently of the chunk budget.
- [ ] A provider terminal `Ok` arriving after the latch does **not** overwrite the `Internal` error.
- [ ] `grep -rn 'recover()\|panic(' internal/effects/ai_stream_core.go` → **zero hits** (this is a
      merge gate for the ruling; pair it with a known-positive control in the same call, e.g. the
      same pattern against `internal/effects/stream.go`).
- [ ] No goroutine or timer is created anywhere in the new code.

---

### S1-M4 — Test matrix (1.0 d, ~350 test LOC)

Put new tests in `internal/effects/ai_stream_core_test.go` (keep
`ai_step_with_stream_recorded_test.go` as the author's four, unmodified, so the adoption stays
legible).

| # | test | notes |
|---|---|---|
| 1 | table-driven argument/type/decode failure parity across both siblings | one table, both wrappers |
| 2 | no-handler and no-`FnCaller` typed errors; recorded mode returns **empty** `chunks` | |
| 3 | callback failure: error is traced, stream continues, recorded chunks stay exact | fail-soft, matches existing op |
| 4 | all three chunk variants: `ContentDelta`, `ThinkingDelta`, full `Usage` field encoding + order | |
| 5 | unencodable first / middle (see S1-M3) | only route is `onChunk(nil)` — §0.5 |
| 6 | both drain budgets, independently (see S1-M3) | |
| 7 | latched error not overwritten by later provider success/error | |
| 8 | empty stream, success and error | |
| 9 | multiple chunks: stable order, no duplicate callback, callback value ≡ recorded value | encode once, append before callback |
| 10 | capability denial + AI budget accounting parity between siblings | |
| 11 | trace assertions: operation name, routing metadata, outcome, fatal provider index, `provider_chunks`, `delivered_chunks`, `drain_exhausted` | |
| 12 | builtin registry + public type + metadata (`v0.32.0`, `StabilityExperimental`) | |
| 13 | **evaluator ↔ non-strict `--bytecode` parity for nested `RecordedStream`** | see risk note below |
| 14 | ADR-009 ordering gate (Go level): immediate projection, exact ordered returned-log parity, success, partial-then-error, no duplicate delivery | this is the P0 justification — but see "MUST NOT claim" |

**Risk note on #13.** The design doc's "no dedicated bytecode op is needed" is **corroborated but
not proven** for the *new* nested record/list/ADT return: the controller confirmed
`runtime.CallEntrypoint` is the evaluator-only dispatch path (`cmd/ailang/run_helpers.go:473-483`)
and that both ops register via `RegisterEffectBuiltin` → `effects.Call` → `RegisterOp("AI", …)`,
but nobody has tested value conversion for `{chunks: [StreamChunk], outcome: Result[...]}` across
the eval↔bytecode boundary. **If #13 fails, that is a real finding, not a test bug — stop and
report it, do not paper over it.** `--strict-bytecode` behaviour must stay identical to the
existing callback-bearing stream function.

**Acceptance**
- [ ] `SKIP_NET_TESTS=1 go test ./internal/effects/ ./internal/builtins/ -count=1` → rc=0; report
      the `--- PASS:` count.
- [ ] `SKIP_NET_TESTS=1 go test ./... ` and `go build ./...` → rc=0, modulo sandbox-denied
      listener tests, which are labelled **UNINFORMATIVE UNDER SANDBOX** with the exact test name.
- [ ] `make check-file-sizes` → green.
- [ ] `make lint` → rc=0.

---

### S1-M5 — Contract text: LongDesc, metadata, CHANGELOG, credit (0.5 d, ~60 LOC + docs)

1. **Builtin `LongDesc` for `stepWithStreamRecorded` MUST state all four** (design doc
   §Documentation correction):
   1. linear, **unbounded** in-memory retention until the call returns;
   2. the log is exact with respect to chunks the provider **adapter emits**, not the provider
      wire — in particular tool-call `input_json` stream content is never emitted as chunks by
      design (`internal/ai/anthropic/streamstep.go:336-337`);
   3. an unencodable chunk terminates the op with typed `Internal` (stable prefix
      `unencodable stream chunk`) and an explicitly incomplete prefix; after that point AILANG
      records/encodes/delivers nothing and its per-chunk work is bounded — **but the call still
      returns only when the provider's stream ends**;
   4. this fail-loud behaviour **deliberately diverges** from `stepWithStream`, which silently
      skips an unencodable chunk; the siblings are equivalent only for fully-encodable streams
      (which is every stream constructible today).
2. **Metadata**: `Since: "prototype"` → `Since: "v0.32.0"`; keep `StabilityExperimental`.
   `grep -rn 'prototype' internal/builtins/ai*.go` must return zero hits for this symbol.
3. **CHANGELOG.md**: new entry under v0.32.0 (unreleased), crediting **@arniwesth** as original
   author and linking #546 + `arniwesth/ailang#2`. **Do not touch the historical changelog
   occurrence of the "open row" wording.**
4. **File the follow-up issue**: "cancellable provider context across the AI surface
   (`AIHandler` v2)" — the honest fix for the wall-clock residual the planner ruling leaves open,
   and the standing answer to R2's `gpt5-6-sol` objection. Link it from #546. Do **not** start it.
5. **Ask @arniwesth on #546** (do not assume the answer): would he prefer a first-class
   `AIError` code (`IncompleteStream`) over the stable message prefix on `Internal`? It is a
   public-vocabulary widening, so it is his call as the consumer.

**Acceptance**
- [ ] `./bin/ailang doctor builtins` (with `AILANG_BUILTINS_REGISTRY=1`) → rc=0.
- [ ] LongDesc text contains all four items — quote it in the progress notes.
- [ ] `grep -rn 'Since: *"prototype"' internal/builtins/` → zero hits for this symbol (with a
      known-positive control proving the grep sees `Since:` lines at all).
- [ ] CHANGELOG entry present and names @arniwesth.
- [ ] Follow-up issue filed; #546 question posted.

---

## 3. Success metrics for sprint 1

- [ ] `SKIP_NET_TESTS=1 go test ./... -count=1` rc=0 (sandbox-denied listener tests labelled).
- [ ] `go build ./...` rc=0.
- [ ] `make check-file-sizes` green; **no file in `internal/effects` above 700 lines**.
- [ ] `make lint` rc=0.
- [ ] `make verify-examples` green (sprint 1 adds no example, so this should be untouched — if it
      goes red, suspect **manifest drift**, not a type regression).
- [ ] Zero `panic(`/`recover()` in the new code.
- [ ] @arniwesth credited in a commit message and in CHANGELOG.

**Not** in sprint 1's metrics, by design: example file, `ailang prompt` / μRAG discovery, website,
the two doc-truth repairs, the CI text guard, and the M4 exhaustiveness guard.

---

## 4. Sprint 2 (scoped now, planned separately)

~1.5–2.0 days. Contents:

1. `examples/ai_streaming_recorded.ail` — `ailang check` with the freshly built target compiler,
   then run with `--ai-stub`. **Regenerate `examples/manifest.json`** — `make verify-examples`
   enforces manifest/module parity (`make/examples.mk:18-27`) and drift is the usual cause of a
   red gate here.
2. Replace both false "the callback's effect row is open" claims with the exact wording in the
   design doc §Documentation correction: `std/ai.ail:324` and
   `examples/runnable/ai_streaming.ail:40-42`. Historical CHANGELOG untouched.
3. A focused CI text guard that fails if the streaming docs again describe this callback row as
   "open".
4. Teaching prompt + μRAG/builtin discovery entry; generated builtin docs; docs website.
5. **M4 — `StreamChunk` exhaustiveness guard.** **CHECK #546 FIRST.** As of 2026-08-03T08:20 the
   issue has 4 comments and @arniwesth has **offered** the guard but **not yet posted a patch**
   (verified by the planner: `gh issue view 546 --json comments`). If a patch has landed by the
   time sprint 2 starts, M4 collapses to review-and-adopt with credit — the preferred path. If
   not, budget 0.5 d and implement it: a canonical `allStreamChunkVariants` registry slice in
   `internal/ai` beside the variant declarations plus a test asserting
   `encodeStreamChunk(v) != nil` for every entry, and/or a CI parity check comparing
   `streamChunkMarker()` implementer count against `encodeStreamChunk` case count. The guard must
   cover both siblings. Note it should also cover `cmd/wasm/effects.go`'s `jsToStreamChunk`, which
   is a second, independent place a fourth variant can go missing.

---

## 5. Top risks, in the order the executor will actually hit them

1. **The 800-line wall (S1-M1/M2).** 22 lines of headroom at HEAD+patch. Mitigated by making the
   split the first commit; if `make check-file-sizes` ever goes red mid-sprint, split further
   rather than compressing code.
2. **The shared-core refactor silently changes `stepWithStream`.** The doc's binding Non-Goal.
   Mitigated by adopting the patch verbatim in its own commit first (so the refactor is a
   reviewable diff), and by `TestAIStepWithStream_UnchangedByRecordedVariant` plus the parity
   table. Characterization first, refactor second.
3. **Evaluator↔bytecode parity for the nested `RecordedStream` (S1-M4 #13).** The one design-doc
   claim nobody has tested. A failure here is a real finding — report it, do not work around it.

Lower but real: sandbox listener denials misread as failures (§0.1); `examples/manifest.json`
drift in sprint 2; and rebase conflict with `M-AI-STRUCTURED-STEP`, which touches the same
`std/ai` + builtin + effects surface (design doc §Conflict Surface).

---

## 6. Open question for a human (one-word answerable)

> **The sentinel-panic abort is dropped, so the drain budget now bounds post-failure WORK
> (O(1)/chunk, zero retention) but NOT post-failure WALL-CLOCK — the call returns when the
> provider's stream ends. This is on a path unreachable for any `StreamChunk` constructible
> today. Accept? (yes / no)**

Reasons are in the design doc's `§PLANNER RULING`. If the answer is **no**, the only remaining
mechanism is a cancellable provider context — Mark's already-rejected option (b) — which becomes
a blocking dependency and pushes this to a multi-sprint item.

---

**Created**: 2026-08-03 (mission iteration 133)
**Supersedes as an execution plan**: the design doc's M1–M4 milestone table (which remains the
authoritative acceptance definition).
