# Ollama Embedding Result Variants, std Wrappers, and AI Effect Relabel

**Status**: Planned
**Target**: v0.57.1 (or next minor slot if maintainers prefer a breaking-change window — see High-Impact Decision 2)
**Priority**: P1 (High) — blocks a real consumer (sunholo/crew_lab crew memory) from degrading gracefully; workaround (worker-process isolation) is costly and fragile
**Estimated**: 4 days (2 days implementation + 1 day tests + 0.5 day docs + buffer)
**Dependencies**: None (all required machinery — AI effect ops, AIError classification, stub/fixture handlers — already ships)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Stub/fixture embeddings are deterministic functions of the input; the Result path makes failure outcomes typed values instead of host crashes |
| A2: Replayability | +1 | Fixture-driven embed replay (`--ai-stub --ai-stub-fixtures`) plus `RecordAIEffect` trace events bring embeds into the existing AI replay surface |
| A3: Effect Legibility | +1 | The AI label tells the truth: an embedding is model inference, not generic console/file IO. Today `! {IO}` hides a network call |
| A4: Explicit Authority | +1 | Embeds become gated by the AI capability and charged to the AI budget; today `--caps IO` (or any IO grant) silently authorizes network inference |
| A5: Bounded Verification | +1 | Offline fixture tests make embedding programs verifiable in CI with no Ollama daemon, no key, no spend |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Typed `Result[[float], AIError]` gives agents a machine-decidable degrade path (`code`, `retryable`) instead of parsing a crashed run's stderr |
| A8: Minimal Syntax | 0 | No new syntax; new builtins + std wrappers only |
| A9: Cost Visibility | +1 | Embeds are charged against `[budgets] AI` and appear in budget reports and AI trace events |
| A10: Composability | +1 | The Result variant composes with std/result and mirrors the established `callResult` family shape |
| A11: Structured Failure | +1 | The core ask: typed errors (`ConnectionFailed`, `ModelNotFound`, `Timeout`) instead of aborting the host |
| A12: System Boundary | 0 | Boundary unchanged (local Ollama via `OLLAMA_HOST`) |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced; stub vectors are hash-derived, fixtures are literal
- [x] A3 (Effects): no hidden side effects; the relabel makes an existing hidden network side effect *more* explicit
- [x] A4 (Authority): no ambient access granted; the change *removes* an accidental ambient grant (IO cap covering network inference)
- [x] A7 (Machines First): typed Result + error codes are for machine consumers; the crash variant is retained only for back-compat

## Problem Statement

**Reported by** `stapledons_godot` via ailang messages, against v0.54.5 — verified still present on v0.57.0 (current tree).

`_ollama_embed` / `_ollama_embed_media` (internal/builtins/ollama_embed.go, DX-18) are the only
network-facing builtins that (a) have no Result variant — any Ollama failure returns a raw Go
error that **aborts the whole run** — and (b) are labeled `Effect: "IO"`, so a plain
`--caps IO` grant authorizes network inference calls to Ollama, while the AI machinery
(`[budgets] AI`, `--ai-stub`, AI trace events) does not cover them at all.

Repro (verified in this workspace; Ollama daemon not running):

```
module bad
export func main() -> () ! {IO} = _io_println(show(length(_ollama_embed("no-such-model", "hi"))))

$ ailang run --caps IO --entry main bad.ail
Error: execution failed: _ollama_embed: Ollama embed failed: Post "http://127.0.0.1:11434/api/embed":
  dial tcp 127.0.0.1:11434: connect: connection refused
```

The reporter's variant (model not pulled) fails the same way:
`Ollama embed failed: model "no-such-model" not found` — and in both cases the entire run dies.

**Current State (all claims code-verified; see Verification Log):**

- `_ollama_embed : (string, string) -> list[float] ! {IO}` — no `Result` sibling exists
  (`grep -rn "_ollama_embed_result" internal/ std/ cmd/ examples/` → zero hits).
- No public std wrapper exists: `std/embedding.ail` is pure vector math
  (cosine/dot/normalize/codecs); the underscore builtins are mentioned only in its header comment.
- Runtime gating is by `spec.Effect` via the generic builtin wrapper
  (internal/runtime/builtins.go:136 `ctx.RequireCapWithBudget(builtinSpec.Effect, "")`), so today the
  **IO capability and the IO budget** authorize and pay for embedding calls. Verified empirically:
  a no-caps run fails with `effect 'IO' requires capability, but none provided`.
- `ollamaEmbedRequest` never consults `ctx.AI`, so `--ai-stub` / `--ai-stub-fixtures` cannot
  intercept it — every test that touches embeddings needs a live Ollama daemon.
- The sibling pattern already exists and is proven: `_ai_call_result` /
  `_ai_call_json_result` (M-AI-TOOL-LOOP, v0.17.0) return `Result[..., AIError]` through AI
  effect ops, are budget-charged, traced, and stub-covered. The Net builtins likewise return
  `Result[HttpResponse, NetError]`. `_ollama_embed` is the outlier.

**Impact:**

- A game using local embeddings for crew memory (sunholo/crew_lab) cannot degrade gracefully when
  Ollama is stopped or the model is not pulled. The only safe pattern today is isolating embedding
  in a spawned worker process so a crash kills only the worker — process-level error handling as
  a substitute for the language's own `Result` type.
- Security/authority smell: `--caps IO` is the "let it print" grant; it currently also means "let
  it call the network" for this builtin. The AI budget cannot cap embedding spend.

## Goals

**Primary Goal:** Make Ollama embedding failures catchable typed values (`Result[[float], AIError]`), gated by the AI capability with AI-budget accounting and offline stub coverage — mirroring `std/ai.callResult`.

**Success Metrics:**
- The reporter's repro, rewritten with the Result variant, prints a degrade message and exits 0 with Ollama down (instead of aborting).
- `ailang run --caps IO ...` (no AI cap) can no longer reach the embedding path; `--caps AI` is required.
- `[budgets] AI = N` caps embedding calls; the budget report counts them.
- An embedding program runs under `--ai-stub` (and `--ai-stub-fixtures`) with **no Ollama daemon, no key, no network** — CI-runnable.
- Evaluator and strict VM agree on all new fixtures (`--bytecode --strict-bytecode` parity).
- Zero behavior change for programs that do not reference the embedding builtins.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Err payload is the `AIError` record (`{code, message, retryable}`), not `string` | Deviates from the reporter's inline `Result[[float], string]` sketch but matches their stated model ("like std/ai callResult"); locks the match-arm shape for all callers | human | design | med |
| D2: Relabel `_ollama_embed` / `_ollama_embed_media` from `Effect: "IO"` to `Effect: "AI"` (breaking) | The whole A3/A4 fix; breaks every caller's `! {IO}` signature and `--caps IO` invocation, including the reporter's own game (which asked for it) | human | design | high |
| D3: Route embeds through the AI effect op layer with a new optional `EmbedHandler` seam + a default Ollama embedder (not ctx.AI-only) | Preserves "works without `--ai`" (crew_lab runs embeds with no chat model configured) while letting stub/fixture handlers intercept; determines the whole architecture | human | design | high |
| D4: Public wrappers live in `std/embedding` (not `std/ai`) | Reporter's suggestion; std/embedding already documents the underscore builtins; keeps vector math + vector acquisition in one module. std/ai.ail imports only std/result, so `import std/ai (AIError)` from std/embedding creates no cycle | agent | design | low |
| D5: Default stub embedding = deterministic hash-derived vector (fixed dimension) | Byte-exact offline tests (A1) need determinism without Ollama; dimensionality/derivation detail | agent | compile | low |
| D6: Fixture support via a new `"embed"` kind, additive to fixture format version 1 | Keeps old fixture files loading; new key canonicalization (model+text) must be documented alongside speech's precedent | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1 — AIError record as the Err payload (human approval; overrides the reporter's `string` sketch with rationale)
- [ ] D2 — the IO→AI relabel of the two existing crash-variant builtins, and whether it lands in v0.57.1 or waits for a minor window (human approval; the only breaking change in this doc)
- [ ] D3 — op-layer routing with EmbedHandler seam + default Ollama embedder fallback (human approval)

## Solution Design

### Overview

Bring Ollama embeddings into the AI effect family exactly the way `callResult` was brought in for
chat (M-AI-TOOL-LOOP): thin builtin pass-throughs → AI effect ops → handler seam. The ops return
`Result[[float], AIError]` for the new Result variants and propagate Go errors for the retained
crash variants. A new **optional** `EmbedHandler` interface on the AI handler lets `--ai-stub`
and `--ai-stub-fixtures` intercept embedding calls for offline tests; when the bound handler does
not implement it (including `ctx.AI == nil`), the op falls back to a **default Ollama embedder**
moved out of `internal/builtins` into `internal/ai/ollama`, preserving today's "no `--ai` flag
needed, `OLLAMA_HOST` respected" semantics. The crash variants are relabeled `IO` → `AI`.

### Architecture

**Components:**

1. **Default Ollama embedder** (`internal/ai/ollama/embed.go`, new ~120 LOC): move
   `ollamaEmbedRequest`/`ollamaMediaInput` out of `internal/builtins/ollama_embed.go`. Same wire
   behavior: `api.ClientFromEnvironment()`, `client.Embed`, 30 s text / 120 s media timeouts,
   float32→float64 conversion. Adds the one thing the builtin lacked: error classification into
   `*ai.AIError` —
   - client-creation failure / `connection refused` / EOF → `ConnectionFailed` (retryable) —
     `ai.ClassifyError` already maps these;
   - context deadline → `Timeout` (retryable);
   - Ollama 404 "model … not found" → `ModelNotFound` (NOT retryable). This mapping must be
     explicit: `classify()` has no "not found" rule, so the reporter's exact failure would
     otherwise fall through to `Internal` with `retryable=true`, telling callers to retry a
     missing model. Match the Ollama api error type (an `*api.StatusError` carries the HTTP
     status; the implementer verifies the exact shape against the vendored module source at
     `github.com/ollama/ollama v0.40.13`) or the `"not found"` message;
   - empty-embeddings response → `Internal` (not retryable — protocol oddity).
   No new AIError codes are needed; the existing vocabulary covers every case.
2. **EmbedHandler seam** (`internal/effects/ai_embed.go`, new ~200 LOC): optional interface
   following the `AIHandlerWithImageRefs` / `AIHandlerWithRouting` precedent —
   ```go
   type EmbedHandler interface {
       Embed(model, text string) ([]float64, error)
       EmbedMedia(model, text string, image, audio []byte) ([]float64, error)
   }
   ```
   Resolution order inside the op: if `ctx.AI`'s handler implements `EmbedHandler` → use it
   (this is how stubs/fixtures intercept); otherwise use the default Ollama embedder. This is a
   documented, loud default — not a silent fallback: embedding is Ollama-backed by design
   (DX-18), independent of the chat provider, and the docs state it.
3. **AI effect ops**: `RegisterOp("AI", "embed", …)`, `"embedMedia"`, `"embedResult"`,
   `"embedMediaResult"` (all verified unallocated — see Verification Log). The Result ops
   construct `Ok(list[float])` via a new `makeOkFloatListResult` helper and `Err(AIError)` via the
   existing `makeAIErrorResultRecord` (internal/effects/ai_encode.go:104). All four ops record
   `RecordAIEffect(opName, [model, text-len], "ok:dim=N" | "err:<code>", nil)` so embeds show up
   in traces/budget reports like every other AI call.
4. **Builtins** (`internal/builtins/ollama_embed.go`, rewrite ~-60/+120 LOC): the two existing
   builtins become thin pass-throughs (`ctx.RequireCapWithBudget("AI", "")` +
   `effects.Call(ctx, "AI", "embed"/"embedMedia", args)`, exactly the `ai_step.go` pattern) with
   `Effect: "AI"`. Two new registrations:
   - `_ollama_embed_result : (string, string) -> Result[list[float], AIError] ! {AI}`
   - `_ollama_embed_media_result : (string, string, bytes, bytes) -> Result[list[float], AIError] ! {AI}`
   Both `Module: "std/ai"`, `Stability: StabilityExperimental`, `Since` set to the release version.
5. **Stub + fixtures**:
   - `StubAIHandler` gains `Embed`/`EmbedMedia` returning a deterministic vector (D5): fixed
     dimension (suggest 8; agent may choose), each component derived from the sha256 of
     `"model\x00text"` — byte-stable across runs and platforms. Media mixes in the bytes' sha256.
   - `FixtureAIHandler`: new fixture kind `"embed"` — response field `"embedding": [0.1, …]`
     (float array) or the existing `"error": {code, message, retryable}` replay. Canonical key
     (D6): compact JSON `{"model":M,"text":T}`, keys sorted, no HTML escaping — the
     `SpeechFixtureCanonical` recipe (internal/effects/ai_stub_fixtures.go). Media fixtures add
     `"image"`/`"audio"` sha256 hexes to the canonical object. Fixture format **stays version 1**:
     the change is additive (new kind value + new optional field); old files load unchanged and
     unknown-kind rejection still guards typos.
6. **std wrappers** (`std/embedding.ail`, +~40 LOC): `import std/ai (AIError)` plus —
   ```ailang
   export func ollamaEmbed(model: string, text: string) -> [float] ! {AI} = _ollama_embed(model, text)
   export func ollamaEmbedResult(model: string, text: string) -> Result[[float], AIError] ! {AI} = _ollama_embed_result(model, text)
   export func ollamaEmbedMedia(model: string, text: string, image: bytes, audio: bytes) -> [float] ! {AI} = _ollama_embed_media(model, text, image, audio)
   export func ollamaEmbedMediaResult(model: string, text: string, image: bytes, audio: bytes) -> Result[[float], AIError] ! {AI} = _ollama_embed_media_result(model, text, image, audio)
   ```
   User code never touches an underscore builtin. (The signature syntax is verified — see
   Verification Log entry V-check-1.)
7. **Warning text** (`cmd/ailang/ai_handlers.go`, ~1 line): the `--caps AI requires --ai` warning
   gains a caveat that Ollama embeds work without `--ai`, so embed-only programs are not told to
   configure a chat model they do not need.

### Implementation Plan

**Phase 1: Op layer + default embedder** (~1 day)
- [ ] Move `ollamaEmbedRequest`/`ollamaMediaInput` to `internal/ai/ollama/embed.go`; add AIError classification incl. the ModelNotFound mapping
- [ ] `internal/effects/ai_embed.go`: EmbedHandler interface, 4 ops, `makeOkFloatListResult`, RecordAIEffect
- [ ] Unit tests: classification table (connection refused → ConnectionFailed/retryable; 404 model-not-found → ModelNotFound/not-retryable; deadline → Timeout), op Result construction

**Phase 2: Builtins + relabel** (~0.5 day)
- [ ] Rewrite `internal/builtins/ollama_embed.go`: pass-throughs, `Effect: "AI"`, two new Result builtins with metadata
- [ ] Confirm `ailang doctor builtins` / validator passes (registry validation catches Effect/arity mismatches)

**Phase 3: Stub + fixtures** (~1 day)
- [ ] `StubAIHandler.Embed/EmbedMedia` deterministic vectors
- [ ] Fixture kind `"embed"`: format struct, validation, canonical keys, `FixtureAIHandler` overrides; docs in `docs/docs/guides/ai-effect.mdx`
- [ ] Tests: fixture hit/miss/dup/error paths, old-format files still load

**Phase 4: std wrappers + docs + migration** (~1 day)
- [ ] `std/embedding.ail` wrappers; `ailang check` on a program importing them
- [ ] Update `examples/reference/neural_semantic_search.ail` and `examples/reference/ollama_embed_test.ail`: `! {IO}` → `! {AI}`, run lines `--caps IO,SharedIndex` → `--caps AI,SharedIndex`
- [ ] Update `docs/docs/guides/semantic-caching-how-to.mdx` + `docs/docs/guides/semantic-search.md` (same relabel)
- [ ] Changelog entry under `changelogs/v0.32-current.md` with an explicit breaking-change note and the one-line migration (`! {IO}`→`! {AI}`, `--caps` add `AI`)
- [ ] End-to-end acceptance (see Success Criteria): degrade repro, no-AI-cap denial, budget charge, offline stub run, `--bytecode --strict-bytecode` parity

### Files to Modify/Create

**New files:**
- `internal/ai/ollama/embed.go` (~120 LOC) — default Ollama embedder + AIError classification
- `internal/effects/ai_embed.go` (~200 LOC) — EmbedHandler seam, 4 AI ops, Ok-list encoder
- `internal/ai/ollama/embed_test.go` (~150 LOC) — classification + wire-shape tests
- `internal/effects/ai_embed_test.go` (~200 LOC) — op tests (stub handler, default embedder, Result shapes)

**Modified files:**
- `internal/builtins/ollama_embed.go` (−60/+120 LOC) — pass-throughs, AI effect, 2 new Result builtins
- `internal/effects/ai.go` (~+12 LOC) — StubAIHandler gains Embed/EmbedMedia
- `internal/effects/ai_stub_fixtures.go` (~+90 LOC) — `"embed"` fixture kind
- `std/embedding.ail` (+~40 LOC) — 4 public wrappers + header comment update
- `cmd/ailang/ai_handlers.go` (+1 line) — warning caveat
- `examples/reference/neural_semantic_search.ail`, `examples/reference/ollama_embed_test.ail` (effect rows + run lines)
- `docs/docs/guides/semantic-caching-how-to.mdx`, `docs/docs/guides/semantic-search.md`, `docs/docs/guides/ai-effect.mdx` (stub/embed fixture docs)
- `changelogs/v0.32-current.md` — entry + breaking-change note

## Examples

### Example 1: Graceful degradation (the reporter's repro)

**Before** (v0.54.5–v0.57.0): Ollama down or model not pulled → whole run aborts.

**After:**
```ailang
module crew_mem
import std/embedding (ollamaEmbedResult)
import std/io (println)
import std/list (length)

export func main() -> () ! {AI, IO} =
  match ollamaEmbedResult("embeddinggemma-2", "crew memory") {
    Ok(vec)  => println("embedded ${show(length(vec))} dims"),
    Err(e)   =>
      if e.retryable then println("ollama down (${e.code}) — falling back to simhash")
      else println("model missing (${e.message}) — prompt the user to pull it")
  }
```
With Ollama stopped this prints `ollama down (ConnectionFailed) — falling back to simhash` and
exits 0. The match/on `Result`, the `AIError` field access, and the string interpolation are all
verified working syntax (Verification Log V-check-1); only the wrapper target is new code.

### Example 2: Offline test via stub fixtures (no Ollama, no key, no network)

```json
{ "version": 1,
  "fixtures": [
    {"kind": "embed", "model": "embeddinggemma-2", "text": "hello",
     "embedding": [0.1, 0.2, 0.3, 0.4]},
    {"kind": "embed", "model": "embeddinggemma-2", "text": "hi",
     "error": {"code": "ModelNotFound", "message": "model \"embeddinggemma-2\" not found",
               "retryable": false}}
  ]
}
```
`ailang run --caps AI --ai-stub --ai-stub-fixtures embeds.json --entry main test.ail` — the Ok
fixture replays the literal vector, the error fixture replays the reporter's exact failure as a
catchable `Err`.

### Example 3: Authority tightening

Before: `ailang run --caps IO ...` authorized embedding network calls (verified: a no-caps run
today fails on the **IO** capability check). After: the same run fails with
`effect 'AI' requires capability, Hint: Run with --caps AI` — the IO grant no longer reaches the
network, and `[budgets] AI` caps embedding spend.

## Success Criteria

- [ ] Reporter's repro (rewritten per Example 1) prints a degrade message and exits 0 with Ollama down (acceptance: run in a sandbox without Ollama)
- [ ] `--caps IO` alone cannot reach any embedding builtin; runtime error names the AI capability (acceptance: run without AI cap)
- [ ] `[budgets] AI = 1` run with 2 embed calls fails loudly on the second, budget report counts both
- [ ] Embedding program runs offline under `--ai-stub` and under `--ai-stub-fixtures` (acceptance: CI test, no daemon)
- [ ] Evaluator and strict VM agree on all new examples (`--bytecode --strict-bytecode`)
- [ ] `make test-core` green; no behavior change for programs not referencing the embedding builtins
- [ ] Old fixture files (kinds text/image/speech) still load unchanged
- [ ] Documentation updated (guides, std/embedding header, ai-effect.mdx fixture docs, changelog incl. breaking-change note)
- [ ] `ailang doctor builtins` passes on the two new registrations

## Conflict Surface

This design touches `internal/effects/` (new ops + optional interface on the handler family) and
`internal/builtins/` (effect relabel + new registrations). No parser/lexer/typechecker surface
changes; the conflict surface is registry/effect-row based.

### Positions touched

- **Builtin name registry**: adds `_ollama_embed_result`, `_ollama_embed_media_result` (verified
  unallocated). No existing name is redefined.
- **AI op registry**: adds `embed`, `embedMedia`, `embedResult`, `embedMediaResult` (verified
  unallocated — full `RegisterOp("AI", …)` list enumerated in the Verification Log).
- **Effect rows of two existing builtins**: `_ollama_embed` / `_ollama_embed_media` change
  `! {IO}` → `! {AI}` at the type level (both the `BuiltinSpec.Effect` and the `types.Builder`
  `.Effects(...)` must change together — the validator flags drift).
- **AIHandler optional-interface family**: one more optional capability interface alongside
  `AIHandlerWithRouting` / `AIHandlerWithImageRefs`; existing handlers that do not implement it
  are unaffected.
- **Fixture file format**: one new kind value and one new optional response field, additive to
  version 1; `DisallowUnknownFields` + unknown-kind rejection stay.

### What else lives there / disambiguation

- The runtime capability check is **generic over `spec.Effect`** (internal/runtime/builtins.go:
  `RequireCapWithBudget(builtinSpec.Effect, "")`) — no dispatch change needed; the relabel is
  data, not code.
- Effect rows: a function calling a relabeled builtin now requires `AI` in its signature row.
  The typechecker's existing "Missing effects: AI / Suggested fix: func main(...) -> T ! {AI}"
  diagnostic (observed live in this workspace for the IO case) is the migration guide — callers
  get a precise edit suggestion. No row-algebra changes.

### Programs that MUST still work

1. `examples/reference/neural_semantic_search.ail` — updated in-tree (effect rows + run line); must run against a live Ollama exactly as before once migrated
2. `examples/reference/ollama_embed_test.ail` — same
3. Any program using `std/ai.callResult` / `step` — zero changes to those ops or the AIError shape
4. Any fixture file using kinds `text`/`image`/`speech` — additive format change only
5. Any program not referencing the embedding builtins — no effect rows, caps, or budgets shift

### What deliberately changes

- `_ollama_embed` / `_ollama_embed_media` signatures: `! {IO}` → `! {AI}` (breaking). Migration:
  add `AI` to the calling functions' effect rows and to `--caps`. In-repo consumers: the two
  reference examples + two guide pages, all updated in this sprint. Known external consumer:
  sunholo/crew_lab (the reporter, who requested the change). Both crash-variant builtins keep
  their crash semantics — only the capability gate and budget lane change.
- The reporter's literal ask `Result[[float], string]` is intentionally delivered as
  `Result[[float], AIError]` (Decision D1); a `string` payload would drop the retryable/code
  signal their own use case (distinguish "Ollama down" from "model not pulled") depends on.

## Testing Strategy

**Unit tests:**
- `internal/ai/ollama/embed_test.go`: error classification table (connection refused / 404 model-not-found / deadline / empty embeddings), float32→float64 conversion, media input assembly (empty-part omission)
- `internal/effects/ai_embed_test.go`: ops return correct Result ADT shapes; EmbedHandler-seam resolution (handler implements vs nil ctx.AI vs non-embedding handler → default embedder); RecordAIEffect emitted; stub vectors deterministic across calls
- `internal/effects/ai_stub_fixtures_test.go`: `"embed"` kind validation (needs `embedding` or `error`; dup keys rejected; media canonical keys), old-format compatibility

**Integration tests:**
- CLI-level: degrade repro exits 0 with Ollama absent; `--caps IO` denial message names AI; `--ai-stub` offline run byte-exact across two invocations (A1); `--bytecode --strict-bytecode` parity on the migrated examples
- Budget: `[budgets] AI` charging via a two-call program

**Manual testing:**
- Live Ollama: embeddinggemma-2 768-dim round trip (reporter measured ~0.4 s); media embed with an image fixture

## Deferred Decisions

- Stub vector dimensionality and derivation (D5 detail) — agent may choose; must be documented in ai-effect.mdx
- Media fixture canonical key details (sha256 field names in the canonical object) — agent may choose, following the speech precedent
- Whether the crash variants get a deprecation timeline after the Result variants ship — human at review, post-release
- Whether `RecordAIEffect` result strings include a truncated text preview like `callResult` does, or only lengths (privacy) — agent may choose
- Trace-tier placement of embed events if `--trace-tier` filtering needs a rule — agent may choose, default to the same tier as other AI ops

## Non-Goals

**Not in this feature:**
- Relabeling to a `Net` effect instead of `AI` — considered and rejected for now: Net has no budget/stub/trace integration, and an embedding is model inference, not arbitrary HTTP. The residual (AI cap grants localhost Ollama access without a Net allowlist) is documented under Future Work.
- Batch embeddings (Ollama's API accepts multiple inputs per call) — single-input wrappers only; a batch API is additive later.
- Cloud embedding providers (OpenAI/Gemini embeddings) — the EmbedHandler seam makes this a later, additive provider feature.
- Result-ifying `_sharedindex_*` builtins — local-only, no network; their Go errors are programming errors, which crashing on is correct.
- Changing `_simhash` or the pure vector-math half of `std/embedding` — untouched.

## Timeline

**Week 1** (~2.5 days):
- Phase 1: op layer + default embedder + classification (1 day)
- Phase 2: builtins + relabel (0.5 day)
- Phase 3: stub + fixtures (1 day)

**Week 2** (~1.5 days):
- Phase 4: std wrappers, examples, guides, changelog (1 day)
- End-to-end acceptance + review buffer (0.5 day)

**Total: ~4 days across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking change (IO→AI) hits unknown external consumers | Med | Builtins are `Stability: StabilityExperimental` (the metadata says so); loud changelog note + typechecker "Missing effects: AI" diagnostic gives a one-line migration; the known consumer requested it |
| Default-embedder fallback is mistaken for a silent fallback | Med | Documented loudly (AI-effect guide + builtin LongDesc): "embeds are Ollama-backed by default, `OLLAMA_HOST` respected; stub handlers intercept via EmbedHandler"; the fallback is a defined default, not an error path |
| ModelNotFound mapping depends on Ollama api error shape | Low | Implementer verifies `*api.StatusError` fields against the vendored module (v0.40.13); belt-and-braces: match 404 status OR `"not found"` in the message |
| Stub vectors too "real" (tests overfit to 8 dims) | Low | Documented derivation + dimension; fixtures provide exact vectors for byte-exact tests |
| Fixture format drift confuses older binaries | Low | Format stays version 1; change is additive; old files verified loading unchanged |

## Related Documents

The create-script's SimHash + neural search over "ollama embed result" returned no matches
(neural search needs a live Ollama, which is itself evidence for this doc's ask #3). Manual
sweep found these:

**Implemented (may inform design):**
- [m-ai-tool-loop.md](../implemented/v0_17_x/m-ai-tool-loop.md) (v0.17.0) — established the `_ai_call_result` / AI op / AIError pattern this design copies
- [dx-16-shared-index-deterministic-retrieval.md](../implemented/v0_6_0/dx-16-shared-index-deterministic-retrieval.md) — the SharedIndex + embedding retrieval stack this feeds
- [m-doc-sem-lazy-embeddings.md](../implemented/v0_6_0/m-doc-sem-lazy-embeddings.md) — neural-embedding consumers of this builtin
- [m-eval-failure-attribution.md](../implemented/v0_22_0/m-eval-failure-attribution.md) — the `--caps AI` + `--ai-stub` harness wiring the stub coverage plugs into
- [m-ai-provider-config.md](../implemented/v0_15_0/m-ai-provider-config.md) — provider/handler config layer the EmbedHandler seam extends

**Planned (check for overlap):** none found covering embedding Result variants or effect relabel.

## Verification Log

Every load-bearing claim above was checked against the live tree (commit b23cadce, v0.57.0) —
code citations, greps, or live `ailang` runs — per the design-doc-creator hard gate.

| # | Claim | Verification |
|---|-------|--------------|
| V1 | No Result variant exists for `_ollama_embed` | `grep -rn "_ollama_embed_result\|ollamaEmbed\|embedResult" internal/ std/ cmd/ examples/` → only the builtin's own registration file; zero Result/wrapper hits |
| V2 | No std wrapper exists (user code must call the underscore builtin) | `grep -rn "_ollama_embed" std/*.ail` → header-comment mentions only (std/embedding.ail:16–17) |
| V3 | Effect is labeled `IO` today | internal/builtins/ollama_embed.go:30,94 `Effect: "IO"` + `makeOllamaEmbedType … .Effects("IO")` |
| V4 | Runtime gate + budget charge follow `spec.Effect` (so today the **IO** cap/budget authorize embeds) | internal/runtime/builtins.go:136 `ctx.RequireCapWithBudget(builtinSpec.Effect, "")`; live run without caps: `Error: execution failed: effect 'IO' requires capability, but none provided / Hint: Run with --caps IO` |
| V5 | A failed embed aborts the whole run | Live repro in this workspace (Ollama down): `Error: execution failed: _ollama_embed: Ollama embed failed: Post "http://127.0.0.1:11434/api/embed": dial tcp 127.0.0.1:11434: connect: connection refused` — exit before main's println |
| V6 | `--ai-stub` cannot intercept embeds (no live-Ollama-free test path exists) | internal/builtins/ollama_embed.go `ollamaEmbedRequest` never reads `ctx.AI`; stub/fixture handlers live only behind `ctx.AI` (cmd/ailang/ai_handlers.go `setupAIHandler`) |
| V7 | `callResult` returns `Result[string, AIError]` through an AI op (the pattern being copied) | std/ai.ail:142; internal/builtins/ai_step.go:104–143; internal/effects/ai_step.go:62–100 |
| V8 | Net builtins already return Results (embeds are the outlier) | internal/builtins/net.go:53,81 `Result[HttpResponse, NetError]` |
| V9 | AIError vocabulary covers the needed codes; no new codes required | internal/ai/errors.go constants (`CodeConnectionFailed`, `CodeModelNotFound`, `CodeTimeout`, `CodeInternal`) |
| V10 | `classify()` has no "model not found" rule → the reporter's failure would misclassify as retryable `Internal` | internal/ai/errors.go:262–305 switch read: quota/429/timeout/connection/5xx rules only; `"model … not found"` matches none → falls to `NewAIError(CodeInternal, msg, true)` |
| V11 | Op names `embed`/`embedMedia`/`embedResult`/`embedMediaResult` unallocated | Full `RegisterOp("AI", …)` enumeration grepped: call, callJson, callJsonSimple, callImage(+Base64/WithRefs×2), callSpeech, callResult, callJsonResult, callJsonSimpleResult, step, stepWithCache, stepWithStream, stepWithStreamRecorded |
| V12 | Fixture kind `"embed"` unallocated; format v1 is additively extensible; old files keep loading | internal/effects/ai_stub_fixtures.go: kinds text/image/speech only; `validate()` rejects unknown kinds (additive code change); `DisallowUnknownFields` + version==1 checks unchanged by a new kind/field |
| V13 | In-repo callers of `_ollama_embed` (blast radius) are exactly 2 examples + 2 guides + sharedindex doc-strings | `grep -rln "_ollama_embed" examples/ tests/ bench/ benchmarks/ eval_projects/ goldens/ docs/` → examples/reference/neural_semantic_search.ail, examples/reference/ollama_embed_test.ail, docs/docs/guides/semantic-caching-how-to.mdx, docs/docs/guides/semantic-search.md (+ internal doc-string examples in sharedindex.go metadata, no effect rows) |
| V14 | No std import cycle for `std/embedding` importing `std/ai (AIError)` | std/ai.ail imports only std/result (head of file); nothing in std/ imports std/embedding |
| V15 | Planned signature/match syntax typechecks | V-check-1 (live `ailang check` + run, this workspace): `Result[[float], AIError] ! {AI}` signatures, `match … { Ok(vec) => …, Err(e) => if e.retryable … }`, AIError field access, `${e.message}` interpolation — `✓ No errors found!`, degrade program runs and prints; the `+` string concat error in the first draft was caught and fixed by the checker itself |
| V16 | `--caps AI` without `--ai` currently warns "Programs using AI.call will fail" — misleading for embed-only programs after this change | Live run in this workspace (degrade test): warning text observed verbatim; motivates the ai_handlers.go caveat task |
| V17 | Typechecker produces the "Missing effects" migration diagnostic claimed in Conflict Surface | Live run in this workspace: `Effect checking failed for function 'main' … Missing effects: IO … Suggested fix: func main(...) -> T ! {IO}` |

Note for the implementer: the Ollama api error type (`*api.StatusError`) carries the HTTP status
per the upstream module; the module source is not present in this sandbox (no Go toolchain), so
that one shape claim must be confirmed against the vendored `github.com/ollama/ollama v0.40.13`
source when writing the ModelNotFound mapping (Risk table has the belt-and-braces rule).

## References

- Reporter: `stapledons_godot` via ailang messages — "DX (v0.54.5): _ollama_embed has no Result variant and no std wrapper" (task task-48c3ec66)
- Pattern precedent: [m-ai-tool-loop.md](../implemented/v0_17_x/m-ai-tool-loop.md) (callResult family), fixture stubs (#1497, `--ai-stub-fixtures`)
- Stub/fixture docs: [ai-effect.mdx](../../docs/docs/guides/ai-effect.mdx)
- [Design Axioms](/docs/references/axioms) — A3/A4 are the core of ask #3

## Future Work

- Net-allowlist gating for the embed endpoint (deny-by-default `OLLAMA_HOST` restrictions) if a sandboxing need is reported
- Batch embeddings (one request, N vectors) behind the same EmbedHandler seam
- Cloud embedding providers as EmbedHandler implementations
- Deprecation timeline for the crash variants once callers have migrated
- `ailang doctor`/LSP surface for embedding model presence checks

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
