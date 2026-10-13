# M-OLLAMA-EMBED-RESULT-AI-EFFECT

**Status**: Planned
**Target**: v0.54.8 (breaking-change window closes at v1.0.0 — see Risks)
**Priority**: P1 (DX — blocks graceful degradation in real user code)
**Estimated**: 3 days
**Dependencies**: None (extends the existing `--ai-stub` / `--ai-stub-fixtures` machinery, no changes to it beyond a new fixture kind)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No new nondeterminism; the stub vector is deterministic, the real call was already effectful |
| A2: Replayability | +1 | Fixture kind `embed` makes embedding-dependent runs replayable byte-for-byte offline; today they need a live Ollama |
| A3: Effect Legibility | +1 | The declared effect now matches the actual side effect (AI model inference over the network), not a generic IO label |
| A4: Explicit Authority | +1 | An IO grant no longer reaches the network; embedding demands an explicit AI capability and charges the AI budget |
| A5: Bounded Verification | +1 | Embedding behavior (including failure paths) becomes verifiable offline via --ai-stub / fixtures instead of requiring a live model server |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Result variant gives AI-generated code a decidable Err branch instead of a host-aborting Go error — the reported failure mode was an agent's game crashing whole-process |
| A8: Minimal Syntax | 0 | No syntax changes; additive builtins/wrappers only |
| A9: Cost Visibility | +1 | Embedding calls now charge the AI budget and operator ceiling; previously invisible under IO |
| A10: Composability | +1 | Result composes with std/result and the match idiom; wrappers compose with std/embedding's vector math and SharedIndex |
| A11: Structured Failure | +1 | Err(message) instead of process abort; capability failures remain typed CapabilityError |
| A12: System Boundary | +1 | Boundary crossing (local model server) now crosses the AI boundary with cap, budget, trace — uniform with every other AI call |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects — the change *removes* a mislabeled one
- [x] A4 (Authority): no ambient access granted — the change *narrows* IO's reach
- [x] A7 (Machines First): the primary beneficiary is machine-written code that must degrade gracefully

## Problem Statement

`_ollama_embed(model, text) -> [float] ! {IO}` and its media variant are the only
neural-embedding surface in the language. They have two defects, both reported from real use
(sunholo/crew_lab: crew memory via local Ollama embeddings, embeddinggemma-2, 768 dims, ~0.4 s):

1. **No Result variant.** Any Ollama failure (server stopped, model not pulled, timeout)
   surfaces as a Go-level error that aborts the entire run. A game that wants to degrade
   gracefully (fall back to SimHash retrieval, or skip memory) cannot: the only safe pattern
   today is isolating embedding in a spawned worker process so the crash kills only the
   worker. Every other AI call surface already has a Result variant (`std/ai.callResult`,
   `callJsonResult`, `callJsonSimpleResult`, `callSpeech`) — the embedding surface is the
   one gap.

2. **Mislabeled effect.** The builtins declare `! {IO}`, so `--caps IO` alone authorizes a
   network call to the Ollama host. `--caps AI --ai-stub` does NOT cover them (verified:
   `ailang run --caps AI --ai-stub` on an embedding program fails with
   `effect 'IO' requires capability`), AI budgets do not charge them (the impl bypasses
   `effects.Call`, so `RequireCapWithBudget` never runs), and there is no stub — so tests of
   embedding behavior, including failure paths, need a live Ollama.

**Current State (verified on v0.54.7 binary, this machine):**

```
$ ailang run --caps IO --entry main bad.ail      # Ollama not running
Error: execution failed: _ollama_embed: Ollama embed failed:
  Post "http://127.0.0.1:11434/api/embed": dial tcp 127.0.0.1:11434: connect: connection refused
# whole run aborts, exit 1 — no AILANG-level error to catch

$ ailang run --caps AI --ai-stub --entry main bad.ail
Error: execution failed: effect 'IO' requires capability, but none provided
Hint: Run with --caps IO
# the AI stub path does not reach the embedding builtin at all
```

**Impact:** any program mixing AI inference with local embeddings — the primary embedding
use case — cannot test offline, cannot bound embedding spend with AI budgets, and cannot
survive Ollama outages. Works fine otherwise: embeddinggemma-2 via Ollama returns 768 dims
in ~0.4 s.

## Goals

**Primary Goal:** Give the Ollama embedding surface the same failure semantics, capability
labeling, budget charging and offline-testability that `std/ai` call variants already have:
`Result`-returning variants, public std wrappers, and AI-effect gating with stub/fixture
coverage.

**Success Metrics:**
1. A program can catch `ollamaEmbedResult("embeddinggemma-2", text)` returning `Err` with
   Ollama down and continue running (exit 0, fallback branch taken).
2. `ailang run --caps IO` on an embedding-only program FAILS the capability gate;
   `--caps AI` is what authorizes embedding calls.
3. `ailang run --caps AI --ai-stub` runs an embedding program to completion offline,
   deterministically, no network.
4. `[budgets] AI = N` (and the policy operator ceiling) charges embedding calls.
5. An offline test exercises the Err path via `--ai-stub-fixtures` (error fixture), no
   Ollama involved.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Relabel `_ollama_embed` / `_ollama_embed_media` from `! {IO}` to `! {AI}` (breaking), rather than keeping legacy on IO and adding AI-only new builtins | Closes the "IO grant reaches the network" hole; two labels for one wire call would permanently keep the hole open and duplicate the surface (simplicity gates). Breaking: 2 example files, 2 doc guides, std/embedding header | human (breaking change) | design | high |
| D2: Err payload is a `string`, not an `AIError` record | The issue asks for `Result[[float], string]`; matches std/embedding's own `Result[[float], string]` decoders (`decodeF32LE`/`decodeF64LE`); keeps degradation `match` sites minimal. Deviates from std/ai `callResult`'s `AIError` | human (deviates from std/ai precedent) | design | med |
| D3: The real (non-stub) path keeps the direct Ollama client call, independent of the bound `--ai` chat handler — no `AIEmbedHandler` interface on real providers in this version | A game bound to Gemini for chat must still embed via local Ollama. Routing embeds through the single bound handler would break multi-provider setups, the primary use case | agent (design rationale fixed here) | design | med |
| D4: Stub coverage rides the existing `--ai-stub` / `--ai-stub-fixtures` machinery with a new fixture kind `"embed"` | Reuses a proven, documented mechanism (kinds text/image/speech) instead of a new flag; gives offline tests both success and error paths | agent | compile | low |
| D5: Public wrappers live in `std/embedding` as `ollamaEmbed` / `ollamaEmbedResult` / `ollamaEmbedMedia` / `ollamaEmbedMediaResult` | The module's header already points at these builtins and hosts the vector math they feed; the `ollama` prefix keeps the name honest (a bare `embedResult` would imply provider-agnostic, which it is not) | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1 approved: relabel both existing builtins IO→AI (accepting the breaking change in v0.54.8, before the v1.0.0 compat freeze) — **needs human approval**
- [ ] D2 approved: `Result[[float], string]` (string error) rather than `Result[[float], AIError]` — **needs human approval**

D3–D5 are agent-decidable within the rationale stated above.

## Solution Design

### Overview

Four builtins, all `! {AI}`, all routed through the AI effect op registry so they get the
capability gate, budget charge, OTEL span and trace recording every other AI call gets:

```
_ollama_embed(model, text)                                   -> [float] ! {AI}
_ollama_embed_result(model, text)                            -> Result[[float], string] ! {AI}
_ollama_embed_media(model, text, image, audio)               -> [float] ! {AI}
_ollama_embed_media_result(model, text, image, audio)         -> Result[[float], string] ! {AI}
```

Public wrappers in `std/embedding` so user code never touches underscore names:

```
ollamaEmbed, ollamaEmbedResult, ollamaEmbedMedia, ollamaEmbedMediaResult
```

Failure semantics: the `*_result` variants return `Err(message)` where message carries the
underlying Ollama error text verbatim (prefixed with the builtin name, exactly like today's
aborting errors — `_ollama_embed: Ollama embed failed: …`). The non-Result variants keep
today's abort-on-error behavior (now as a proper effect error under the AI gate) — identical
contract to `_ai_call` vs `_ai_call_result`.

### Architecture

Every builtin impl becomes the standard three-liner used by the other AI builtins
(mirror of `aiCallResultImpl`, internal/builtins/ai_step.go):

```go
if err := ctx.RequireCapWithBudget("AI", ""); err != nil { return nil, err }
return effects.Call(ctx, "AI", "embedOllama", args)   // or "embedOllamaMedia", wrapped to Result
```

The two new ops live in `internal/effects/ai_embed.go`:

**Component 1 — op dispatch (`internal/effects/ai_embed.go`, new):**
`RegisterOp("AI", "embedOllama", …)` and `RegisterOp("AI", "embedOllamaMedia", …)`.
Each op:
1. Resolves the stub condition the way `aiCallSpeech` resolves optional capabilities: if
   the resolved handler (`ctx.AI` through `AIContext.ready()`) is `*StubAIHandler` or
   `*FixtureAIHandler`, serve the stub/fixture path (below) — no network.
2. Otherwise call the shared Ollama embed helper (Component 2) with the per-call model
   argument and the existing timeouts (30 s text / 120 s media — unchanged).
3. Record a trace event (`ctx.RecordAIEffect`) with truncated args, like `aiCallResult`
   does, so the observatory sees embeddings in the uniform AI view.

**Component 2 — single Ollama request path (`internal/ai/ollama`, extended):**
Move `ollamaEmbedRequest` (and `ollamaMediaInput`) from
`internal/builtins/ollama_embed.go` into the existing Ollama client package
(`internal/ai/ollama`), home of the ollama Go client usage this repository already has.
Layering: `internal/effects` already imports `internal/ai`, so effects→ai/ollama is a legal
edge; builtins keeps no Ollama client code of its own — one wire path, one place.
The function returns `([]float64, error)`; ops wrap in `ListValue` / `resultOk` / `resultErr`.

**Component 3 — stub + fixtures (`internal/effects`):**
- `StubAIHandler` gains a deterministic `StubEmbed(model, text) []float64`: an 8-dim
  unit vector derived from FNV-64 of `(model, text)` — input-dependent (two texts give
  two vectors, so downstream cosine logic is exercised) but semantically meaningless and
  documented as such. Mirrors `StubSpeechPCM`'s "proves the wiring and the format" contract.
- `--ai-stub-fixtures` gains kind `"embed"`. Canonical request bytes (keyed like the other
  kinds, literally or by sha256): the compact JSON `{"model":M,"text":T}` sorted-keys,
  for the media variant `{"audio":A,"image":I,"model":M,"text":T}` with booleans for
  part-presence (media fixtures may instead replay for the text-only op by omitting them —
  implementer's latitude, D4). Response shape: `{"vector": [..floats..]}`. An `"error"`
  entry (`{"error": {"message": "…"}}`) replays the Err path — this is what makes the
  graceful-degradation test possible with no Ollama at all. A request with no fixture is a
  loud error, never a quiet fallback — same rule as kinds text/image/speech.

**Component 4 — std wrappers (`std/embedding.ail`):**

```ail
-- Ollama neural embeddings (effectful) — see header note on stubs/budgets
export func ollamaEmbed(model: string, text: string) -> [float] ! {AI} = _ollama_embed(model, text)
export func ollamaEmbedResult(model: string, text: string) -> Result[[float], string] ! {AI} = _ollama_embed_result(model, text)
export func ollamaEmbedMedia(model: string, text: string, image: bytes, audio: bytes) -> [float] ! {AI} = _ollama_embed_media(model, text, image, audio)
export func ollamaEmbedMediaResult(model: string, text: string, image: bytes, audio: bytes) -> Result[[float], string] ! {AI} = _ollama_embed_media_result(model, text, image, audio)
```

(std/embedding.ail adds `import std/result (Result)`; its pure math functions are untouched.)

### Migration (the breaking part, D1)

Programs calling `_ollama_embed*` declare `! {IO}` rows today; after relabel their inferred
rows contain `AI`, so explicit `-> … ! {IO}` signatures fail type/effect checking with a
clear error, and `--caps IO` runs fail the capability gate with the existing hint
(`Hint: Run with --caps AI`). Migration is mechanical: add `AI` to the row, swap the cap.
`--caps auto` needs no change (it grants from the entry's declared row). Affected in-repo
callers (complete list, see Verification Log V8): `examples/reference/ollama_embed_test.ail`,
`examples/reference/neural_semantic_search.ail` (rows `! {IO}` / `! {IO, SharedIndex}` →
`! {AI}` / `! {AI, SharedIndex}` + run instructions), `docs/docs/guides/semantic-search.md`,
`docs/docs/guides/semantic-caching-how-to.mdx`, `std/embedding.ail` header comments,
`internal/builtins/sharedindex.go` metadata examples. Historical prompt files
(`cmd/ailang/prompts/v0.5.x–v0.9.x`) are frozen snapshots and are NOT edited; the current
teaching prompt (v0.16.7) does not mention `_ollama_embed` and needs no change.

One UX touchpoint: `cmd/ailang/ai_handlers.go:67` warns
`--caps AI requires --ai <model> flag … Programs using AI.call will fail.` Under the new
design an embedding-only program legitimately runs with `--caps AI` and no `--ai`. Reword
the warning to name `AI.call`-style oracle calls specifically (embedding calls with
`--caps AI` alone are fine) — implementer's latitude on wording, but the "will fail"
blanket claim must not survive.

### Implementation Plan

**Phase 1: Go core** (~1 day)
- [ ] Move `ollamaEmbedRequest` + `ollamaMediaInput` to `internal/ai/ollama`; keep behavior
      byte-identical (timeouts, error strings).
- [ ] New `internal/effects/ai_embed.go`: ops `embedOllama` / `embedOllamaMedia`, stub
      detection, trace recording, `StubEmbed` deterministic vector.
- [ ] Rework `internal/builtins/ollama_embed.go`: all four builtins `Effect: "AI"`, impls via
      `RequireCapWithBudget` + `effects.Call`; add the two `*_result` registrations
      (`T.App("Result", T.List(T.Float()), T.String())`, built with the existing
      `resultOk`/`resultErr` helpers — same pattern as `_vec_decode_f32le`).
- [ ] Fixture kind `"embed"` in `internal/effects/ai_stub_fixtures.go` (success + error
      entries), documented in the file's format header.

**Phase 2: std + examples + docs** (~0.5 day)
- [ ] `std/embedding.ail`: four wrappers + `Result` import + header update (AI cap, budgets,
      stub note replacing the `! {IO}`-implying comment).
- [ ] Migrate the two `examples/reference/*.ail` files (rows + run comments).
- [ ] Update `docs/docs/guides/semantic-search.md`, `semantic-caching-how-to.mdx`,
      `docs/docs/reference/effects.md` (AI row for the embed builtins),
      `--caps AI` warning text in `cmd/ailang/ai_handlers.go`.

**Phase 3: tests + changelog** (~1 day)
- [ ] Offline: stub vector determinism; fixtures replay (success + Err path); capability gate
      (`--caps IO` denied, `--caps AI` granted) at op level via `effects.Call` unit tests
      (mirror `internal/effects/ai_image_test.go` patterns).
- [ ] Existing `internal/builtins/ollama_embed_test.go`: keep the live-Ollama tests as
      skips (they call `ollamaEmbedImpl` directly today — re-point them at the op/helper;
      the skip-when-Ollama-down behavior stays for the true integration assertions).
- [ ] One runnable degradation example under `examples/` (see Examples).
- [ ] Changelog entry under `changelogs/v0.32-current.md` with the migration note.

### Files to Modify/Create

**New files:**
- `internal/effects/ai_embed.go` — the two AI ops + stub vector (~150 LOC)
- `examples/reference/ollama_embed_fallback.ail` — degradation example (~60 LOC)

**Modified files:**
- `internal/ai/ollama/embed.go` (new home; ~80 LOC moved from builtins, byte-identical logic)
- `internal/builtins/ollama_embed.go` — relabel + route + two new Result builtins (~+90 LOC)
- `internal/effects/ai_stub_fixtures.go` — kind `"embed"` (~+60 LOC)
- `std/embedding.ail` — 4 wrappers + import (~+30 LOC)
- `examples/reference/ollama_embed_test.ail`, `examples/reference/neural_semantic_search.ail` — row/cap migration (~10 LOC)
- `docs/docs/guides/semantic-search.md`, `docs/docs/guides/semantic-caching-how-to.mdx`, `docs/docs/reference/effects.md` — signatures + cap guidance
- `cmd/ailang/ai_handlers.go` — warning rewording (~3 LOC)
- `internal/builtins/sharedindex.go` — metadata example strings only
- `changelogs/v0.32-current.md` — entry + migration note

## Conflict Surface

This change touches `internal/effects/` (op registry, stub handler) and `internal/builtins/`
(builtin registry), so the surface is enumerated per the design-doc gate:

1. **Positions extended:** (a) builtin name namespace: `_ollama_embed_result`,
   `_ollama_embed_media_result` — both verified unallocated; (b) AI op namespace:
   `embedOllama`, `embedOllamaMedia` — verified absent from every `RegisterOp` call;
   (c) fixture kind namespace: `"embed"` alongside text/image/speech; (d) the `Effect`
   field of two existing builtins.
2. **Other constructs already in those positions:** the AI op registry holds call,
   callJson, callJsonSimple(+Result variants), callImage*(4), callSpeech, step*(4),
   callStream — none embed-shaped; the stub handler implements Call/CallJson/Step/
   StepWithCache/StepWithStream/CallSpeech — an added `StubEmbed` method extends an
   interface-satisfying struct, breaking no other implementer (only `*StubAIHandler` and
   `*FixtureAIHandler` are asserted on, by type, not by interface).
3. **Disambiguation:** no parser surface is touched (builtin calls parse identically);
   op and builtin names are unique so registry lookup cannot collide; `--caps` names are
   unchanged (`AI` already in `CapsList`, verified — no new capability name).
4. **Programs that MUST still work post-change (fixtures):**
   - `std/ai` Result calls: `ailang run --caps AI --ai-stub` on a `callResult` program
     verified working today (exit 0) — must remain byte-identical behavior.
   - `std/embedding` pure math + `decodeF64LE(encodeF64LE(v))` round-trip — untouched code.
   - `examples/reference/neural_semantic_search.ail` and `ollama_embed_test.ail` — must run
     against a live Ollama with `--caps AI[,SharedIndex]` after the mechanical row edit.
   - Speech stub: `--ai-stub` `callSpeech` tone (the new embed kind must not disturb
     fixture file validation for kinds text/image/speech).
   - `--caps auto` on any program whose entry row is now `! {AI}` — auto-grant path is
     row-driven (`ResolveAutoCaps`), unaffected.
5. **Deliberate incompatibilities (the whole point of D1):**
   - Explicit `! {IO}` rows over embedding calls stop type/effect-checking.
   - `--caps IO` no longer authorizes embedding; runs fail the capability gate with the
     standard hint.
   - Embedding calls now charge the AI budget / operator ceiling — a program at
     `AI = 0` budget now fails where it previously ran unlimited embeds under IO.
   - `--caps AI` without `--ai`/`--ai-stub` now WORKS for embedding-only programs (direct
     local-Ollama path, D3) — a behavior widening that the reworded warning must state.

## Examples

### Example 1: Graceful degradation (the reported use case)

**Before (aborts the whole run when Ollama is down or the model isn't pulled):**
```ail
-- only safe pattern today: run this in a spawned worker so the crash kills just the worker
let emb = _ollama_embed("embeddinggemma-2", memory_text) in
_sharedindex_upsert_emb("beliefs", id, hash, emb, 1, ts)
```

**After:**
```ail
import std/embedding (ollamaEmbedResult)
import std/result (Result)

export func store_memory(id: string, hash: int, memory_text: string) -> string ! {AI, SharedIndex} =
  match ollamaEmbedResult("embeddinggemma-2", memory_text) {
    Ok(emb) => { _sharedindex_upsert_emb("beliefs", id, hash, emb, 1, ts()); "neural" },
    Err(e) => { _sharedindex_upsert("beliefs", id, hash, 1, ts()); "simhash-fallback: " ++ e }
  }
-- run: ailang run --caps AI,SharedIndex --entry main game.ail
-- offline test: ailang run --caps AI --ai-stub --ai-stub-fixtures embed_fixtures.json ...
```

### Example 2: Offline CI test of the Err path

```json
{ "version": 1, "fixtures": [
  {"kind": "embed", "model": "embeddinggemma-2", "text": "hello",
   "error": {"code": "ConnectionFailed", "message": "connect: connection refused"}}
]}
```
Under `--ai-stub --ai-stub-fixtures`, `ollamaEmbedResult("embeddinggemma-2", "hello")`
returns `Err("connect: connection refused")` — the fallback branch runs in CI with no
Ollama, no network, deterministically.

## Success Criteria

- [ ] Repro from the issue: with Ollama down, a program matching on
      `ollamaEmbedResult(...)` takes its `Err` branch and exits 0 (acceptance: the new
      `examples/reference/ollama_embed_fallback.ail` run twice — once live-down, once with
      fixtures — both green in CI without a live Ollama).
- [ ] `ailang run --caps IO` on an embedding program fails the capability gate;
      `--caps AI` grants it (acceptance: gate unit test at op level + manual CLI check).
- [ ] `--caps AI --ai-stub` completes an embedding program offline; two different texts
      yield two different deterministic stub vectors (acceptance: effects unit test).
- [ ] `--ai-stub-fixtures` kind `embed` replays success and error entries; a fixture miss
      is a loud error (acceptance: fixtures unit test, mirroring speech-fixture tests).
- [ ] AI budget: an op-level test with `AI` budget 0 denies the call (mirror
      `internal/effects/ai_lazy_test.go` budget-zero pattern).
- [ ] All tests passing (`make test`), `make check-boundaries` clean (new effects→ai/ollama
      edge must be legal), `make fmt`/`make lint` clean.
- [ ] Documentation updated (guides + effects reference + changelog migration note).

## Testing Strategy

**Unit tests (offline, no Ollama):**
- Stub determinism: `StubEmbed(m, t)` byte-identical across calls/machines; distinct for
  distinct `(m, t)`; magnitude 1.
- Op gating: `effects.Call(ctx, "AI", "embedOllama", …)` without the AI cap →
  `CapabilityError`; with cap + budget 0 → budget denial.
- Fixtures: kind `embed` success/error replay; miss → loud error; media canonical key shape.
- Result construction: `Ok` carries `ListValue` of `FloatValue`; `Err` carries the message
  verbatim (no swallowing, no fabricated codes).

**Integration tests (live Ollama, keep-skip semantics):**
- Existing dimensionality assertions (768 dims for embeddinggemma) remain in
  `internal/builtins/ollama_embed_test.go`-successor, skipped when Ollama is down — the
  live server stays a skip-gated integration check, never a CI requirement.

**Manual testing:**
- The issue's repro commands, inverted: `--caps IO` denied, `--caps AI` with Ollama down
  aborts on the non-Result variant (unchanged contract) and degrades on the Result variant.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact stub vector derivation (FNV seed → 8 dims is the suggested shape; any deterministic,
  input-dependent, unit-magnitude definition is acceptable) — agent may resolve.
- Fixture canonical-key details for the media variant (sorted compact JSON vs. a simpler
  separator scheme) and whether media fixtures may serve the text-only op — agent may
  resolve, but the key format must be documented in the fixtures file header and the
  guides like the other kinds are.
- Warning-text wording in `cmd/ailang/ai_handlers.go` — agent may resolve; the blanket
  "will fail" claim must not survive.
- Whether `RecordAIEffect` carries a routing-metadata nil (embeds don't route) or a stub
  marker — agent may resolve.

## Non-Goals

**Not attempted in this feature:**
- Provider-agnostic embeddings (OpenAI/Gemini embed models, `AIEmbedHandler` on real
  providers) — no second provider exists in-repo; adding the interface now would be
  speculative. Revisit when one lands (D3 records the seam).
- Net-effect labeling — the AI label matches precedent: all AI-provider network calls
  (Gemini, OpenRouter, Ollama chat) are gated by `AI` alone, and Net's secure defaults
  BLOCK localhost (`NetContext.AllowLocalhost = false`), which would break local Ollama by
  default. Deliberately not taken.
- Batch embedding (multiple inputs per call) — Ollama supports it; no caller needs it yet.
- Changing `_embedding_encode`/`_embedding_decode` or the pure vector math in std/embedding.
- Consolidating `internal/messaging/embedder.go`'s separate Ollama embed usage — different
  plane (CLI message embedding, env-configured), noted in Future Work only.

## Timeline

**Week 1** (3 days):
- Day 1: Phase 1 (Go core: helper move, ops, builtins, fixtures kind).
- Day 2: Phase 2 (std wrappers, examples, docs) + Phase 3 tests start.
- Day 3: Finish tests, changelog, `make test`/`lint`/`check-boundaries`, sprint-evaluator.

**Total: ~3 days across 1 week** (2× the naive estimate of 1.5 days, per skill guidance).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking change lands on existing embedding users (crew_lab is the known one; the issue author is that user and asked for the relabel) | Med | Ship in v0.54.8 with a changelog migration note + mechanical example migrations; BEFORE v1.0.0's compat freeze, when the blast radius is still this enumerable (2 example files, 2 guides, verified V8) |
| Stub vectors mistaken for real semantic similarity in user tests | Low | Document in std/embedding header and the stub doc comment: wiring/format proof only, fixtures supply realism — same contract as `StubSpeechPCM` |
| Effect relabel misses a caller (motoko fork, eval projects) | Low | The type/effect checker converts every miss into a clear compile error naming `AI`; grep of the repo shows no other callers (V8); motoko fork is on its own cadence and gets the same clear error |
| `RequireCapWithBudget` interplay with frame scoping double-charges an embed | Low | The existing `budgetChargeDepth` re-entrancy guard covers builtin-wrapper-then-op dispatch (same shape as `_ai_call_result` → `callResult`); an op-level test asserts exactly one charge |
| Fixture kind `embed` changes validation of existing fixture files | Low | New kind is additive; existing kinds' validation paths untouched; fixture-file version stays 1 (verified: kind dispatch is per-entry, `fixtureKind*` constants) |

## Verification Log

Every "does/doesn't support" premise above, verified live on this machine (ailang v0.54.6
binary, repo at v0.54.7) unless the row cites code:

| # | Claim | Verification |
|---|-------|--------------|
| V1 | `_ollama_embed` failure aborts the whole run (no catchable error) | Reproduced: `ailang run --caps IO` → `Error: execution failed: _ollama_embed: Ollama embed failed: … connection refused`, exit 1 |
| V2 | No Result variant exists | `grep -rn "_ollama_embed" internal/ std/`: only `_ollama_embed` + `_ollama_embed_media` (internal/builtins/ollama_embed.go) |
| V3 | No public std wrapper exists | `std/embedding.ail` mentions the builtins in header comments only; no wrapper function calls them |
| V4 | AI stub does not cover embedding today | `ailang run --caps AI --ai-stub` on the repro → `effect 'IO' requires capability` (the builtin declares IO, so the AI stub path never reaches it) |
| V5 | AI budgets don't charge embedding today | `ollamaEmbedImpl` calls `ollamaEmbedRequest` directly — no `effects.Call`, no `RequireCapWithBudget` (internal/builtins/ollama_embed.go:98-127) |
| V6 | Capability gating is row-driven: entry `! {IO}` demands `--caps IO` | No-caps run of `! {IO}` entry → `effect 'IO' requires capability … Hint: Run with --caps IO`; `! {AI}` entry (std/ai callResult) → same shape for AI; `--caps AI --ai-stub` on the callResult program exits 0 |
| V7 | The Result-variant + wrapper pattern is established | internal/builtins/ai_step.go (`_ai_call_result`: `RequireCapWithBudget("AI","")` + `effects.Call(ctx,"AI","callResult")`); `resultOk`/`resultErr` helpers at internal/builtins/float_codec.go:36-42; `ailang check` passes on the planned wrapper signature `-> Result[[float], string] ! {AI}` (scratch module, this session) |
| V8 | Complete caller list for the relabel | Repo-wide grep `_ollama_embed`: examples/reference/{neural_semantic_search,ollama_embed_test}.ail, docs/docs/guides/{semantic-search.md,semantic-caching-how-to.mdx}, std/embedding.ail header, internal/builtins/sharedindex.go metadata, changelogs (history), prompts v0.5.x–v0.9.x (frozen). Current prompt v0.16.7: no reference |
| V9 | Op names `embedOllama`/`embedOllamaMedia` and builtin names `_ollama_embed_result`/`_ollama_embed_media_result` are unallocated | `grep -rn "RegisterOp(" internal/effects` lists all AI ops (call*, step*, callImage*, callSpeech, callStream…) — none embed-shaped; V2 for builtin names |
| V10 | Fixture kind namespace is text/image/speech only; version-1 files validate per-entry | internal/effects/ai_stub_fixtures.go:43-46 (`fixtureKindText/Image/Speech` constants), file-format header lists the kinds |
| V11 | `AI` is a valid `--caps` name; no new cap needed | internal/runner/caps.go:36 `CapsList = "IO,FS,Net,Env,Process,Clock,AI,Stream,SharedMem,SharedIndex,Secret,Trace,DOM,Msg,Cog"` |
| V12 | Stub/fixture wiring exists as claimed | cmd/ailang/ai_setup.go:28-44 (`stubHandler`: `NewStubAIHandler` or `NewFixtureAIHandler(LoadStubFixtures)`); optional-capability + stub precedent: internal/effects/ai_speech.go (`AISpeechHandler`, `StubSpeechPCM`, `RegisterOp("AI","callSpeech")`) |
| V13 | Net-effect labeling would break local Ollama by default (Non-Goal rationale) | internal/effects/context.go `NetContext`: `AllowLocalhost: false` secure default; docs/docs/reference/effects.md:270 confirms loopback needs `--net-allow-localhost` |
| V14 | effects→ai import edge is legal for the helper move | internal/effects/ai_step.go already imports `internal/ai`; `internal/ai/ollama` already owns the ollama client usage (client.go, step.go) |
| V15 | `--caps auto` survives the relabel unchanged | internal/runner/caps.go `ResolveAutoCaps` grants from the entry's declared effect row — row now contains AI, auto-grant follows |
| V16 | `--caps AI` warning text claims all AI programs fail without `--ai` | cmd/ailang/ai_handlers.go:67 — must be reworded (Component note in Phase 2) |

## Related Documents

**Implemented (may inform design):**
- [m-ai-tool-loop](../implemented/v0_17_x/m-ai-tool-loop.md) — established the Result-returning AI variants (`callResult` et al.) this design brings to the embedding surface
- [dx-16-sprint-completion-report](../implemented/v0_6_0/dx-16-sprint-completion-report.md) — shipped `_ollama_embed` (DX-18 neural embeddings); documents the original `! {IO}` signature being changed here
- [m-doc-sem-lazy-embeddings](../implemented/v0_6_0/m-doc-sem-lazy-embeddings.md) — downstream consumer pattern (lazy embed + cosine) the wrappers compose with
- [semantic-caching-future](../implemented/v0_5_11/semantic-caching-future.md) — the Ollama-integration status table this feature completes

**Planned (checked for overlap):**
- [m-ollama-quota-observation](m-ollama-quota-observation.md) — checked, DISTINCT: coordinator/CLI quota accounting for ollama.com cloud subscriptions; does not touch the language builtin, its effects, or its error semantics. No overlap.

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Issue: "DX (v0.54.5): _ollama_embed has no Result variant and no std wrapper" (this task)
- `docs/docs/guides/semantic-search.md` — current embedding guide (migrated by this feature)
- `changelogs/v0.32-current.md` — `_ollama_embed_media` entry (v0.53.0), the last change to this surface

## Future Work

- `AIEmbedHandler` on real providers (OpenAI/Gemini embeddings) once a second embedding
  provider exists — the op's handler-check seam (D3) is where it plugs in.
- Consolidate `internal/messaging/embedder.go`'s Ollama usage with the shared
  `internal/ai/ollama` embed helper (separate plane today: CLI message embeddings).
- Batch embedding inputs (Ollama accepts a list) if a caller needs throughput.
- Consider an `ailang test` recipe cookbook entry: "testing AI-adjacent effects offline
  with --ai-stub-fixtures", using this feature's embed fixtures as the worked example.

---

**Document created**: 2026-10-13
**Last updated**: 2026-10-13
