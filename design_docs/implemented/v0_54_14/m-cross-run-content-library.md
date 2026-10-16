# Cross-Run Content Library: Persistent SharedMem/SharedIndex + std/ai.embed

**Status**: Implemented (independently evaluated, round 1)
**Target**: v0.54.8 (original approved target; implemented on v0.54.13 checkout with v0.54.14 builtin metadata; no release performed)
**Priority**: P1
**Estimated**: ~2 weeks (~30 hours: 20h implementation + 6h testing + 4h docs, 2x'd)
**Dependencies**: None hard. Builds on M-DX15/DX16 (SharedMem/SharedIndex effects, v0.5.11), M-BRAIN
(SQLite BrainCache seam, M-V1-SIMPLIFICATION Phase 1.3), and the M-AI-PROVIDER-CONFIG surface.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Persistence + the content-hash embed cache make cross-run retrieval reproducible: the same text under the same model yields the identical vector on every run. Strict-mode ordering (score DESC, key ASC) is preserved by the SQLite backend. Flag-driven config, no ambient defaults. |
| A2: Replayability | 0 | No trace/replay changes; SharedIndex trace entries are unchanged. |
| A3: Effect Legibility | +1 | `embed` rides the existing `{AI}` effect row; persistent stores ride the existing `SharedMem`/`SharedIndex` capabilities. No new effect, no hidden side effects — persistence exists only when the operator passes a flag. |
| A4: Explicit Authority | +1 | No capability is granted implicitly: `--sharedmem-db` without `--caps SharedMem` is a hard error (mirrors `--stream-max-message`); `_ai_embed` with no configured embedder returns a typed `ProviderNotFound`, never a silent fallback. |
| A5: Bounded Verification | 0 | No local-check surface changes. |
| A6: Safe Concurrency | +1 | The sqliteopen recipe (WAL + 5s busy timeout + single writer) makes two processes sharing one DB file safe by construction; CAS stays transactional. Today two processes cannot share a store at all. |
| A7: Machines First | +1 | The surface is a typed `Result[[float], AIError]` — machine-decidable retry logic, no string parsing; the storage config is flags + env vars, not prose. |
| A8: Minimal Syntax | 0 | One stdlib wrapper function (`std/ai.embed`); no syntax changes. |
| A9: Cost Visibility | +1 | `embed` records an AI effect op (budget + trace) exactly like `callResult`; the content-hash cache makes repeat-embedding spend visible and avoidable. |
| A10: Composability | +1 | `std/ai.embed` composes directly with `std/sharedindex.upsertWithEmbedding`/`findByEmbedding` and `std/embedding.cosine` — the exact loop the Stapledon's Voyage use case needs. |
| A11: Structured Failure | +1 | Provider/network failures surface as `Err(AIError{code, message, retryable})` instead of crashing the host; DB open failures are loud run errors. |
| A12: System Boundary | +1 | External provider boundary stays explicit (bring-your-own key); persistent storage stays behind operator flags rather than ambient filesystem writes. |

**Net Score: +10** → **Decision: Proceed to implementation** (pending human approval per work-routing gate)

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced. External embeddings are nondeterministic across runs *today* (`_ollama_embed`, raw builtin); the cache strictly improves this. Runs without the new flags are bit-identical to today.
- [x] A3 (Effects): no hidden side effects; both new surfaces require already-declared capabilities.
- [x] A4 (Authority): no ambient access granted; flag-without-cap is an error, unconfigured embed is a typed error.
- [x] A7 (Machines First): typed Result surface, flag/env config, no human-only affordances.

## Problem Statement

**Use case** (Stapledon's Voyage, design repo `vision/premise-and-loop.md` §6): a cross-run content
library. Content generated in one playthrough (a "universe") is found by semantic near-match in later
ones. The same machinery serves crew memories and the Archive's decaying retrieval index. The game
repo is `sunholo-data/stapledons-godot` (see [stapledon-v0-54-6-workarounds.md](../../implemented/v0_54_7/stapledon-v0-54-6-workarounds.md)
for the standing game↔engine friction register).

AILANG already ships the pieces — SharedMem (flat cache), SharedIndex (namespace-partitioned
SimHash + embedding similarity), `sem_frame`, `std/embedding` vector math — but three gaps keep them
from composing into that library:

**Gap 1 — nothing survives a restart, and no two processes can share a store.**
`SetupSharedMemHandler`/`SetupSharedIndexHandler` always construct in-memory contexts
(`internal/runner/handlers.go:171-176` and `:269-274`, both carrying the comment
"Future: could add flags"). A universe's content library dies with the process; a second process
(host orchestrator, serve-api worker, another playthrough) cannot see it. The SQLite backend
(`internal/platform/sharedmem/sqlite.go`, registered via `effects.RegisterSharedCacheOpener` in
`cmd/ailang/platform_init.go`) already satisfies the `SharedCache` seam — but only the CLI brain
(`cmd/ailang/cache.go`) opens it. There is **no** persistent SharedIndex backend at all (only
`InMemorySharedIndex`).

**Gap 2 — the only embedding reachable from AILANG is `_ollama_embed`.**
It is a raw builtin (module `std/ai`, effect `{IO}`) with no cache, and it hard-requires a local
Ollama. The Gemini/OpenAI/Ollama embedders that already exist (`internal/messaging/embedder*.go`)
and the embed config surface (`AILANG_EMBED_PROVIDER`, `embeddings:` in `~/.ailang/config.yaml`,
`internal/config/embed.go`) serve only the CLI brain and messaging. Players bring their own
Gemini/OpenRouter key; Ollama is not guaranteed on their machines. No `internal/ai` provider
implements any embed path (verified: no `Embed` method in `internal/ai/{gemini,openai,openrouter,ollama,configdriven}`),
so there is no `std/ai`-level wrapper to call.

**Gap 3 — the docs lie.**
`docs/docs/guides/semantic-caching-how-to.mdx:17` calls SharedMem "A persistent key-value store".
It is in-memory. A player following the guide loses their universe on exit.

**Impact:** the flagship use case for the semantic-caching stack (a game whose whole premise is
cross-run memory) is unrealizable on stock AILANG. Any game or agent that wants durable
semantic memory must shell out to Ollama or leave the language. This is a feature/semantics gap,
routed through design-doc-creator per the work-routing gate.

## Goals

**Primary Goal:** Make one playthrough's generated content findable by semantic near-match in later
playthroughs, using only stock AILANG: `std/ai.embed` through the configured provider, and
SharedMem/SharedIndex backed by a shared SQLite file.

**Success Metrics:**
- `ailang run --sharedmem-db X` in process A, then process B: B sees A's keys (integration test).
- `ailang run --sharedindex-db X` across two runs: upserted entries from run 1 are found by
  `findSimhash`/`findByEmbedding` in run 2 (integration test).
- `std/ai.embed("text")` returns `Ok([float...])` under `--ai gemini-2-5-flash` + `GOOGLE_API_KEY`
  with **no Ollama installed**, and `Err(AIError{code:"ProviderNotFound"})` when nothing is
  configured (unit tests).
- Identical `(model, text)` pair embeds twice → second call served from the content-hash cache
  (zero provider calls; verified by stub-embedder call counter).
- `docs/docs/guides/semantic-caching-how-to.mdx` no longer claims default persistence.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| SQLite files via the existing `sqliteopen` recipe — not Redis/vector DBs | The backend already exists and satisfies the seam; adding a second store technology multiplies ops surface (the old `semantic-caching-future.md` sketch proposed Redis — rejected as new infrastructure) | human | design | high |
| DB open failure is fatal; never silently fall back to in-memory | A "persisted" library that quietly forgets is silent data loss (guardrail: no silent fallbacks) | human | design | med |
| `--sharedmem-db` without `--caps SharedMem` is an error (mirror `--stream-max-message`) | An ignored flag on a persistence feature looks identical to a working save | human | design | low |
| `std/ai.embed(text) -> Result[[float], AIError] ! {AI}` — single text arg, provider from config; NOT `_ollama_embed`-style model-per-call | Fixes the vector geometry to the operator's config; per-call model mixing silently corrupts search (different models = different dimensions/space) | human | design | med |
| Embedder resolution: explicit `embeddings:` config first, else derived from the bound `--ai` provider when that provider has an embed path + credentials, else typed `Err`; `--ai-stub` gets a deterministic stub embedder | Derivation is the only way "players bring a Gemini key and it just works"; but it must never be silent (one stderr notice when derived) | human | design | med |
| Persistent embed cache lives in a dedicated `embed_cache` table, not `brain_frames` | `brain_frames.key` is `PRIMARY KEY` and the flat SharedCache ops see the whole table — cache rows would leak into `_sharedmem_get`/`_sharedmem_keys` | agent | design | low |
| SharedIndex persistence = new `RegisterSharedIndexOpener` seam + `SQLiteSharedIndex` in `internal/platform/sharedmem` (same package, same recipe) | Mirrors the M-V1-SIMPLIFY core-seam/platform-impl split; keeps cgo sqlite out of the core | agent | design | low |

### Design Freeze

- [x] SQLite-only persistence via the existing opener (D1)
- [x] Fatal open errors, no in-memory fallback (D2)
- [x] Flag-without-cap is a hard error (D3)
- [x] `embed` surface: `(string) -> Result[[float], AIError] ! {AI}` (D4)
- [x] Embedder resolution order incl. derived-from-bound-provider and `--ai-stub` stub (D5)
- [x] Namespace scope: a DB file owns its namespace set; both flags may point at one file (tables
      coexist) or two files (isolated universes). Nothing crosses files.

## Solution Design

### Overview

Two additive wires, one doc fix:

1. **Persistence flags.** `ailang run` gains `--sharedmem-db <path>` and `--sharedindex-db <path>`
   (env `AILANG_SHAREDMEM_DB` / `AILANG_SHAREDINDEX_DB`; flag beats env). SharedMem reuses the
   already-registered SQLite opener (`effects.OpenSharedCache` → `BrainCache`, which satisfies
   `SharedCache`). SharedIndex gets a matching core seam (`effects.RegisterSharedIndexOpener`) and
   a new `SQLiteSharedIndex` in `internal/platform/sharedmem`.
2. **`std/ai.embed`.** A new AI op + builtin + stdlib wrapper that embeds text through the
   configured provider (Gemini/OpenAI/Ollama embedders reused from `internal/messaging`), returns
   `Result[[float], AIError]`, and consults an optional content-hash cache.

Default runs (no flags, no embed call) are byte-identical to today.

### Architecture

**Components:**

1. **Persistent SharedMem** — `SetupSharedMemHandler(effCtx, opts)` opens the DB through
   `effects.OpenSharedCache` when a path is configured and wraps it in `NewSharedMemContext`
   (which takes any `SharedCache`; `SQLiteSharedCache` already satisfies it — compile-time
   checked in `sqlite.go`). Open failure or unregistered backend → hard run error.
   The AILANG surface stays flat-keyed; the file's `brain_frames.namespace` stays `'default'`
   for flat ops (schema unchanged).
2. **Persistent SharedIndex** — new `SQLiteSharedIndex` (own table, below). Core gains
   `effects.RegisterSharedIndexOpener` / `effects.OpenSharedIndex` mirroring the SharedCache seam;
   `cmd/ailang/platform_init.go` registers it next to `sharedmem.Register()`.
3. **`std/ai.embed`** — `RegisterOp("AI", "embed", aiEmbed)` in `internal/effects/ai_embed.go`;
   builtin `_ai_embed` in `internal/builtins` (module `std/ai`) delegating via
   `effects.Call(ctx, "AI", "embed", args)` — exactly the `_ai_call_result` pattern
   (`internal/builtins/ai_step.go:104-145`). `AIContext` gains an `embedder effects.Embedder`
   field + `WithEmbedder` setter; the lazy path flows it automatically (the initializer builds a
   complete `AIContext`, `internal/effects/ai_lazy.go:19`). Nil embedder →
   `Err(AIError{code:"ProviderNotFound", message:"…set AILANG_EMBED_PROVIDER or bind --ai gemini/openai…", retryable:false})`.
   Other errors classified via the existing `classifyOpError` (`internal/effects/ai_step.go:31`).
   Every call records `RecordAIEffect("embed", …)` so budgets and traces see it.
4. **Embedder wiring** (`cmd/ailang/ai_handlers.go`, the point that already knows
   `model.Provider`):
   1. explicit embeddings config (`messaging.LoadEmbedConfigFromEnv` + `NewEmbedderFromConfig`)
      when `embeddings.provider`/`AILANG_EMBED_PROVIDER` is set;
   2. else derive from the bound provider: `gemini` → `GeminiEmbedder` (GOOGLE_API_KEY/ADC),
      `openai` → `OpenAIEmbedder` (OPENAI_API_KEY), `ollama` → `OllamaEmbedder`;
      `openrouter`/`anthropic`/`configdriven` → no embedder (no embed path exists — verified).
      Derivation prints one stderr notice naming the derived provider + model;
   3. `--ai-stub` → deterministic stub embedder (fixed-dimension pseudo-vector derived from a
      hash of the text) so tests/evals run keyless.
   `messaging.Embedder` satisfies `effects.Embedder` structurally (identical 4-method surface:
   `Embed`, `EmbedBatch`, `Dimension`, `ModelName`).
5. **Content-hash embed cache** — a decorator around `effects.Embedder`: in-process map always;
   persistent SQLite layer when `--embed-cache-db <path>` / `AILANG_EMBED_CACHE_DB` is set. Key:
   `ModelName() + "\x00" + sha256(text)`. Table `embed_cache(model TEXT, text_hash TEXT, dim INT,
   vector BLOB, created_at INT, PRIMARY KEY(model, text_hash))` in its own table — NOT
   `brain_frames`, whose flat `key`-primary-key keyspace is program-visible through
   `_sharedmem_get`/`_sharedmem_keys`.

**SharedIndex schema** (new table, same file is fine):

```sql
CREATE TABLE IF NOT EXISTS sharedindex_entries (
  namespace     TEXT NOT NULL,
  key           TEXT NOT NULL,
  simhash       INTEGER NOT NULL,
  version       INTEGER NOT NULL,
  timestamp     INTEGER NOT NULL,
  embedding     BLOB,
  embedding_dim INTEGER DEFAULT 0,
  PRIMARY KEY(namespace, key)
);
CREATE INDEX IF NOT EXISTS idx_sharedindex_ns ON sharedindex_entries(namespace);
```

`Find*` methods `SELECT` the namespace's rows and score in Go (hamming for SimHash, cosine for
embeddings via `internal/simhash.Cosine`), preserving the `SharedIndex` contract: score DESC,
key ASC in Strict mode, `maxScan` bound, empty-slice-on-no-hits. This is the same O(N) scan as the
in-memory backend — persistence, not acceleration (see Non-Goals). Cross-process safety comes from
the recipe: WAL, 5s busy timeout, single-writer pool (`internal/sqliteopen/sqliteopen.go:14-19`).
`IndexEntry.Embedding` is `[]float64`; storage uses the existing `effects.EncodeEmbedding`
(float32 LE) with conversion at the seam — recorded dimension guards consumers.

**Cross-process semantics** (the point of Gap 1): two `ailang run` processes pointing at the same
file share state. Writes serialize through SQLite's single-writer model; `cas` remains atomic
(`SQLiteSharedCache.CAS` is transactional today). Read-after-write is WAL-immediate across
processes.

**Namespaces per file**: a DB file owns its namespace set. Pointing `--sharedmem-db` and
`--sharedindex-db` at one path gives one file with coexisting tables (`brain_frames`,
`sharedindex_entries`, and optionally `embed_cache`); pointing them at different paths isolates
stores. Do **not** point `--sharedmem-db` at the CLI brain's `~/.ailang/state/brain.db` /
`.ailang/state/brain.db`: program flat keys would collide with `ailang cache` frames in the same
`brain_frames` table (documented, not enforced).

### Implementation Plan

**Phase 1: Persistent SharedMem** (~6 hours)
- [x] `SharedMemOptions{DBPath string}` on `runner.Options`; `--sharedmem-db` flag +
      `AILANG_SHAREDMEM_DB` env (flag wins); registry row in `internal/config`
- [x] `SetupSharedMemHandler(effCtx, opts) error`: open via `effects.OpenSharedCache` when path
      set; hard error on open failure; error when flag set without `--caps SharedMem`
- [x] Update call sites: `internal/runner/run.go:477`, `internal/runner/batch.go:54`,
      `cmd/ailang/serve_api.go` `initServeAPIStores` (keep the serve-api SharedMem path working —
      it was broken once before, found 2026-10-02 by the sunholo/mcp_oauth e2e)
- [x] Cross-process + restart integration tests (two processes, one file)

**Phase 2: Persistent SharedIndex** (~8 hours)
- [x] `effects.RegisterSharedIndexOpener` / `effects.OpenSharedIndex` seam (mirror
      `sharedmem_backend.go`; `ErrBackendNotRegistered` when nothing registered)
- [x] `SQLiteSharedIndex` in `internal/platform/sharedmem/sqlite_sharedindex.go`: schema above,
      full `SharedIndex` interface, Strict ordering, `maxScan`
- [x] Register in `cmd/ailang/platform_init.go`; `SharedIndexOptions{DBPath}` + `--sharedindex-db`
      flag + env; same cap-gate and fatal-open rules as Phase 1
- [x] Restart + namespace-isolation tests; decode/encode round-trip for embeddings

**Phase 3: std/ai.embed** (~10 hours)
- [x] `AIContext.embedder` + `WithEmbedder`; op `aiEmbed` in `internal/effects/ai_embed.go`
      (Result construction via the `ai_encode.go` helpers; `classifyOpError`)
- [x] `_ai_embed` builtin in `internal/builtins/ai_embed.go`: type
      `(string) -> Result[list[float], AIError] ! {AI}`, metadata per `BuiltinMetadata` conventions
- [x] `std/ai.ail`: `export func embed(text: string) -> Result[[float], AIError] ! {AI} = _ai_embed(text)`
      (+ doc comment); bump `std/VERSION` in lockstep with the binary
- [x] Wiring in `cmd/ailang/ai_handlers.go` per resolution order; `--ai-stub` stub embedder
- [x] Unit tests: nil embedder → `ProviderNotFound`; provider error → classified `Err`;
      stub determinism (same text → same vector)

**Phase 4: embed cache + docs** (~6 hours)
- [x] Caching decorator (in-process map + optional `embed_cache` table via sqliteopen);
      `--embed-cache-db` flag + env; stub-call-counter test proving cache hits
- [x] Fix `semantic-caching-how-to.mdx:17`; add a "Persistence" section with the flags; update
      `cmd/ailang/help.go` and the env-var registry rows
- [x] New example: `examples/reference/cross_run_library.ail` (embed → upsertWithEmbedding →
      findByEmbedding round-trip); add to `examples/manifest.json`

### Files to Modify/Create

**New files:**
- `internal/platform/sharedmem/sqlite_sharedindex.go` (~220 LOC) — persistent SharedIndex backend
- `internal/effects/sharedindex_backend.go` (~60 LOC) — opener seam (mirrors `sharedmem_backend.go`)
- `internal/effects/ai_embed.go` (~90 LOC) — `aiEmbed` op
- `internal/builtins/ai_embed.go` (~110 LOC) — `_ai_embed` builtin + type + metadata
- `internal/platform/sharedmem/embed_cache.go` (~120 LOC) — embed cache table + decorator
- `examples/reference/cross_run_library.ail` (~60 LOC) — the use-case loop, runnable

**Modified files:**
- `internal/runner/handlers.go` (+40/-8) — options params on both Setup funcs, opener wiring
- `internal/runner/run.go` (+8) — `Options` fields; pass-through
- `internal/runner/batch.go` (+4) — call-site update
- `cmd/ailang/serve_api.go` (+4) — `initServeAPIStores` signature
- `cmd/ailang/main_run.go` (+12) — `--sharedmem-db`, `--sharedindex-db`, `--embed-cache-db` flags
- `cmd/ailang/platform_init.go` (+2) — register the index opener
- `cmd/ailang/ai_handlers.go` (+50) — embedder resolution + `WithEmbedder`
- `cmd/ailang/help.go` (+6) — flag docs
- `internal/config/embed.go` (+6) — env registry rows for the three new vars
- `internal/effects/ai.go` (+15) — `AIContext.embedder` + setter
- `std/ai.ail` (+8) — `embed` wrapper; `std/VERSION` bump
- `docs/docs/guides/semantic-caching-how-to.mdx` (+25/-2) — truth + persistence section

## Examples

### Example 1: the cross-run content library (the use case)

**Before (impossible without local Ollama; dies at process exit):**
```ailang
-- Requires Ollama running locally; effect {IO}; nothing persists.
let emb = _ollama_embed("embeddinggemma", content) in
upsertWithEmbedding("archive", entry_id, simhash(content), emb, 1, now())
-- ...process exits; the universe is gone
```

**After:**
```ailang
import std/ai (embed)
import std/simhash (simhash)
import std/sharedindex (upsertWithEmbedding, findByEmbedding)

-- Write path (run N): index generated content through the configured provider
match embed(entry.content) {
  Ok(v)    => upsertWithEmbedding("universe-3:archive", entry.id, simhash(entry.content), v, entry.version, now()),
  Err(e)   => log_and_fallback(entry, e)   -- typed AIError; retryable says whether to retry
}

-- Read path (run N+1, or a second process): find it by meaning
match embed(query) {
  Ok(qv) => findByEmbedding("universe-3:archive", qv, 5, 0, true),
  Err(e) => []
}
```
```bash
# Two playthroughs share one library file; player brings a Gemini key:
ailang run --caps SharedMem,SharedIndex,AI \
  --sharedmem-db saves/library.db --sharedindex-db saves/library.db \
  --embed-cache-db saves/library.db \
  --ai gemini-2-5-flash game/main.ail
```

### Example 2: cross-process SharedMem (host orchestrator + game)

**Before:** process A `put`s; process B `get`s → `None`. Always.
**After:** with `--sharedmem-db /srv/game/shared.db` in both processes, B sees A's writes (WAL);
`cas` remains atomic across processes, so the "create-if-absent" pattern is a real lock.

## Success Criteria

- [x] Two-process SharedMem integration test passes (A puts, B gets, across `exec.Command` boundary)
- [x] Restart test: index entries upserted in run 1 are found in run 2 via the same DB path
- [x] `std/ai.embed` returns `Ok` under a Gemini/OpenAI embedder with no Ollama installed; typed
      `Err(AIError{code:"ProviderNotFound"})` when unconfigured; stub is deterministic
- [x] Cache: second embed of the same text makes zero provider calls (stub counter)
- [x] `--sharedmem-db` without `--caps SharedMem` exits non-zero with a message naming the flag
- [x] Default runs (no flags) behave byte-identically (existing suites green)
- [x] All tests passing (`make test`; `make check-boundaries` clean — core gains no platform import)
- [x] Documentation updated (guide fixed, help text, env registry, example in manifest)

## Conflict Surface

This touches `internal/effects` (new op + context field), `internal/builtins` (new builtin),
`internal/runner` (handler signatures) — no parser/lexer/typechecker/effect-row-algebra changes.

### Positions touched

| Position | What's new | What already lives there |
|----------|-----------|--------------------------|
| AI op table (`RegisterOp("AI", …)`) | op name `"embed"` | `"call"`, `"callResult"`, `"step"`, `"callSpeech"`, `"callImage*"` … — additive map entry, no dispatch change |
| Builtin registry, module `std/ai` | `_ai_embed` | `_ai_call`, `_ai_call_result`, `_ollama_embed`, `_ollama_embed_media`, … — no name collision (verified: no `_ai_embed` anywhere) |
| `std/ai.ail` exports | `embed` | `call`, `callResult`, `step`, `callSpeech`, `runTools` … — no existing `embed` export (verified by reading the module) |
| `runner.Options` struct | 3 option fields | additive; zero-value = today's behavior |
| `run` flag set | `--sharedmem-db`, `--sharedindex-db`, `--embed-cache-db` | verified unallocated (grep across repo: no hits) |

### Disambiguation

No grammar or type-inference ambiguity: all additions are new identifiers in existing registries.
`std/ai.embed` vs module `std/embedding` — different modules, import-qualified; no shadowing.
`_ai_embed` vs `_ollama_embed` — distinct names, distinct effects (`{AI}` vs `{IO}`); both remain.

### Programs that MUST still work

1. `examples/reference/sharedmem_cache.ail` — SharedMem basics (in-memory default)
2. `examples/reference/semantic_retrieval.ail` — sem_frame + SharedIndex loop
3. `examples/reference/neural_semantic_search.ail` — `_ollama_embed` + index (unchanged)
4. `examples/runnable/stdlib_sharedindex.ail` — stdlib coverage
5. `ailang serve-api --caps SharedMem` end-to-end (the path fixed 2026-10-02 after the
   sunholo/mcp_oauth e2e caught it) — `initServeAPIStores` signature change must keep it live

### What deliberately changes

Nothing at the language level. Go-level: `SetupSharedMemHandler`/`SetupSharedIndexHandler` gain
parameters (internal call sites only: run.go, batch.go, serve_api.go). No existing program,
flag, or behavior changes unless a new flag is passed.

## Testing Strategy

**Unit tests:**
- `SQLiteSharedIndex`: upsert/delete/find round-trip, Strict ordering (score DESC, key ASC),
  `maxScan`, namespace isolation, embedding blob round-trip, empty-namespace edge cases
- `aiEmbed` op: nil embedder → `ProviderNotFound`; embedder error → `classifyOpError` mapping;
  Ok path builds `Ok(list[float])`; `RecordAIEffect` entry present
- Embedder wiring: resolution order (explicit config wins; derivation prints notice;
  openrouter → nil; stub deterministic)
- Embed cache: same (model, text) → single provider call; different model → miss

**Integration tests:**
- Two-process SharedMem share (subprocess or two connections + WAL flush)
- Restart persistence for both stores (open, close, reopen)
- `ailang run` full loop with flags: `--caps SharedIndex,AI --sharedindex-db … --ai gemini…` with
  a stub embedder against a temp DB

**Regression-surface tests:** the five fixtures above run green under default flags (no new flags).

**Manual testing:** `ailang run` the new example twice against one DB; verify second run finds
run 1's entries; verify `--help` lists the flags.

## Deferred Decisions

- In-process embed-cache bound (entry count or bytes) and eviction policy — agent may choose
- Exact stub-embedder dimension/hash (fixed at implementation, must be documented in the builtin
  metadata) — agent may choose
- Whether `serve-api` grows its own `--shared*-db` flags now or inherits env only — agent may
  choose (env-only is acceptable for v0.54.8)
- `SQLiteSharedIndex` close/teardown ergonomics (the `SharedIndex` interface has no `Close`;
  WAL makes process exit safe) — agent may choose; document the choice
- Whether batch mode (`--batch`) exposes the flags or only env — agent may choose

## Non-Goals

- **ANN / sub-linear search** — all `Find*` paths remain O(N) scans (in-memory and SQLite both,
  verified). The `SharedIndex` interface is already the seam a future sqlite-vec/HNSW backend
  slots into; not measured as needed at game scale yet (see Future Work).
- **Read-only SharedMem/SharedIndex capability** (a "brain read" grant without write) — real
  request, but a capability-surface change; needs its own conflict-surface pass (see Future Work).
- **Redis/external stores** — rejected in D1; the SQLite opener already exists.
- **`_ollama_embed` changes** — untouched; it stays the raw local-model builtin.
- **Vector-dimension migration tooling** (re-embedding a library after a model switch) —
  `GarbageCollect`/`DeleteNamespace` exist on the cache tier; a dedicated tool is future work.

## Timeline

**Week 1** (~16 hours): Phases 1–2 — persistent SharedMem + SharedIndex, flags, integration tests.
**Week 2** (~14 hours): Phase 3 — `std/ai.embed` + wiring; Phase 4 — cache + docs + example;
full `make test`, `make check-boundaries`, `make fmt/lint`.

**Total: ~30 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Mixed embedding models in one namespace → silently wrong cosine scores (768-dim vs 1536-dim spaces) | High | Cache keys include model; resolution order fixes one model per run; guide documents pinning (`AILANG_EMBED_GEMINI_MODEL` etc.); `embed_model` is stored with vectors for forensics |
| Two writers on one SQLite file contending | Medium | Recipe already enforces single-writer-per-connection + 5s busy timeout; game loop is low-write; document "one run at a time per file" as the supported shape |
| Derived-embedder surprises (operator set `--ai openrouter/…`, expected embed to work) | Medium | One stderr notice naming the derived provider; openrouter/configdrained derive nothing and `embed` returns a typed `ProviderNotFound` naming the env vars to set |
| `--ai-stub` embedder leaks into production paths | Low | Stub only under explicit `--ai-stub`; asserted in wiring tests |
| serve-api SharedMem path regresses on the signature change | Medium | It broke before (2026-10-02 e2e); its call site is listed in the plan and covered by a regression test |
| std/VERSION drift (binary vs std mismatch warning seen in verification) | Low | Bump in lockstep; `--strict` run in CI catches it |

## Verification Log

Claims verified against dev `329f4dd2` (binary in PATH reports v0.54.6/71f4cf6; std reports v0.54.7 —
the known lockstep warning, see Risks):

| ID | Claim | Evidence | Verdict |
|----|-------|----------|---------|
| V1 | SharedMem/SharedIndex are always in-memory with "Future: could add flags" comments | `internal/runner/handlers.go:171-176`, `:269-274` — `NewSharedMemContext(nil)` / `NewSharedIndexContext(nil)` | Confirmed |
| V2 | The SQLite backend exists and only the CLI brain opens it | `internal/platform/sharedmem/sqlite.go` (`var _ effects.BrainCache = (*SQLiteSharedCache)(nil)`); only non-test `OpenSharedCache`/`NewBrainStore` caller is `cmd/ailang/cache.go`; registered at `cmd/ailang/platform_init.go:18` | Confirmed |
| V3 | No persistent SharedIndex backend exists | `grep -rn "SharedIndex" internal/platform/` → no hits outside tests; only `InMemorySharedIndex` implements `effects.SharedIndex` (`internal/effects/sharedindex_inmemory.go:21`) | Confirmed (negative-existence) |
| V4 | No `--sharedmem-db`/`--sharedindex-db` flags or `AILANG_SHAREDMEM_DB`/`AILANG_SHAREDINDEX_DB` env vars exist | repo-wide grep (go/md/mdx) → zero hits | Confirmed (negative-existence) |
| V5 | `_ollama_embed` is the only AILANG-reachable embed builtin; raw, `{IO}`-effect, uncached, Ollama-only | `internal/builtins/ollama_embed.go` (`Module: "std/ai"`, `Effect: "IO"`, `ollamaEmbedRequest`); `std/ai.ail` contains no `embed` export (read in full) | Confirmed |
| V6 | No `internal/ai` provider implements an embed path | `grep -rn "Embed\b" internal/ai/{gemini,openai,openrouter,ollama,configdriven}/` → empty; embedders live only in `internal/messaging` | Confirmed (negative-existence) |
| V7 | `Result[[float], AIError] ! {AI}` annotation + `Ok`/`Err`/AIError-record construction are valid | `ailang check` on a program declaring `func embed_ok(text: string) -> Result[[float], AIError] ! {AI} = Ok([0.1, 0.2])` + record literal `{ code: "ProviderNotFound", … }` → `✓ No errors found!` (exit 0) | Confirmed |
| V8 | The `_ai_call_result` pattern to mirror exists (op + builtin + Result helpers + `classifyOpError`) | `internal/effects/ai_step.go:31-93` (`classifyOpError`, `RegisterOp("AI","callResult",…)`), `internal/builtins/ai_step.go:104-145`, helpers in `internal/effects/ai_encode.go:48,104` | Confirmed |
| V9 | `AIContext` has no embedder field today; lazy path would carry one automatically | `internal/effects/ai.go:135-140` (fields: `lazy`, `handler`, `modelResolver`); `NewLazyAIContext(initialize)` builds a full `AIContext` (`ai_lazy.go:19`) | Confirmed (negative-existence + mechanism) |
| V10 | `messaging.Embedder` satisfies `effects.Embedder` structurally | Both declare `Embed(string) ([]float32, error)`, `EmbedBatch([]string) ([][]float32, error)`, `Dimension() int`, `ModelName() string` (`internal/messaging/embedder.go:16-20`, `internal/effects/sharedmem_backend.go:14-18`) | Confirmed |
| V11 | Embed config surface + provider env vars exist for reuse | `internal/config/embed.go:8-20` (`AILANG_EMBED_PROVIDER`, models, deprecated defaults); `messaging.NewEmbedderFromConfig` dispatches ollama/openai/gemini/none | Confirmed |
| V12 | Docs call SharedMem "persistent" today | `docs/docs/guides/semantic-caching-how-to.mdx:17`: "**SharedMem**: A persistent key-value store where agents park their thoughts" | Confirmed |
| V13 | sqliteopen recipe makes cross-process sharing safe | `internal/sqliteopen/sqliteopen.go:14-19`: WAL, `busy_timeout=5000` ("a second process waits instead of failing"), synchronous=NORMAL, pool capped at one connection | Confirmed |
| V14 | `brain_frames.key` is `TEXT PRIMARY KEY` (flat keyspace); flat SharedCache ops see the whole table; namespace is metadata, default `'default'` | `internal/platform/sharedmem/sqlite.go` schema (~line 88) + `Get`/`Put`/`Keys` query on `key` alone | Confirmed |
| V15 | All similarity scans are linear today | `internal/effects/sharedindex_inmemory.go:91` iterates the namespace map; `SQLiteSharedCache.SearchBySimHash`/`SearchByEmbedding` `SELECT` all namespace rows and score in Go | Confirmed |
| V16 | Both Setup handlers have exactly three call sites | `internal/runner/run.go:477-478`, `internal/runner/batch.go:54-55`, `cmd/ailang/serve_api.go` `initServeAPIStores` (`:329-330`) — grep, non-test | Confirmed |
| V17 | `AIError` supports the codes this design cites (`ProviderNotFound`, `RateLimit`, `Timeout`, …) | `std/ai.ail` AIError comment enumerates the code set incl. `ProviderNotFound` | Confirmed |
| V18 | std/sharedindex export surface used in examples (`upsertWithEmbedding`, `findByEmbedding`) | `std/sharedindex.ail:53,60` (read) | Confirmed |
| V19 | Regression fixtures exist | `examples/reference/{sharedmem_cache,semantic_retrieval,neural_semantic_search}.ail`, `examples/runnable/stdlib_sharedindex.ail`, `examples/reference/ollama_embed_test.ail` — grep/ls | Confirmed |
| V20 | Flag-without-cap precedent exists (`--stream-max-message needs --caps Stream`) | `internal/runner/handlers.go:222-228` | Confirmed |
| V21 | Embedding blob encode/decode helpers exist for the index backend | `effects.EncodeEmbedding`/`DecodeEmbedding` (`internal/effects/sharedmem_backend.go:145+`, used in `brain.go:257,268`) | Confirmed |

## Related Documents

- [semantic-caching-future.md](../../implemented/v0_5_11/semantic-caching-future.md) (v0.5.11,
  status "Planned" inside implemented/) — proposed Redis/etcd backends as future work. **Distinct
  from this doc**: it predates the SQLite BrainCache seam that shipped in M-BRAIN/M-V1-SIMPLIFY;
  this design reuses that existing opener instead (D1) and adds the missing std-level embed. Its
  "persistent backends" item is hereby superseded by this doc.
- [m-brain-sprint-plan.md](../../implemented/v0_9_2/m-brain-sprint-plan.md) / M-BRAIN — built the
  SQLite backend + CLI brain this design wires into the language runtime.
- [DX-15-semantic-caching-MVP.md](../../implemented/v0_6_0/DX-15-semantic-caching-MVP.md),
  [semantic-caching-complete.md](../../implemented/v0_6_0/semantic-caching-complete.md) — the
  SharedMem/SharedIndex effects themselves.
- [m-embed-task-prefix.md](../../implemented/v0_26_0/m-embed-task-prefix.md) — embedprefix
  role-prefixing; the SQLite auto-embed path already uses it (relevant if the game later wants
  auto-embedding on frame writes).
- [stapledon-v0-54-6-workarounds.md](../../implemented/v0_54_7/stapledon-v0-54-6-workarounds.md) — the standing
  game↔engine friction register for Stapledon's Voyage (game repo
  `sunholo-data/stapledons-godot`, design repo `vision/premise-and-loop.md` §6, external).
- [m-doc-sem-lazy-embeddings.md](../../implemented/v0_6_0/m-doc-sem-lazy-embeddings.md) — lazy
  embedding in doc search (prior art for the embedder seam).

## References

- **Motivation:** Stapledon's Voyage cross-run content library (game design repo,
  `vision/premise-and-loop.md` §6); task directive "Persistent SharedMem/SharedIndex backend +
  std-level embed via the configured provider" (coordinator task-70cf8428).
- **Architecture:** `ARCHITECTURE.md` layer map — core seam (`internal/effects`) vs platform impl
  (`internal/platform/sharedmem`) vs wiring (`cmd/ailang`); `make check-boundaries` guards it.
- **Axiom reference:** [Design Axioms](/docs/references/axioms)
- **Guides:** [semantic-caching-how-to](/docs/guides/semantic-caching-how-to) (fixed by Phase 4)

## Future Work

- **ANN index backend** — sqlite-vec or an in-process HNSW behind the existing `SharedIndex`
  seam; only worth it once a measured namespace exceeds ~10⁴ entries with latency complaints.
  The `maxScan` parameter already gives operators a knob today.
- **Read-only storage capability** — e.g. `--caps SharedMem:read` or a `SharedMemRead` grant so
  a sandboxed module can query the library but not mutate it; needs its own conflict-surface pass
  over the caps grammar.
- **Decay/archive helpers** — the Archive's decaying index can already delete by age on the cache
  tier (`GarbageCollectOlderThan`); a std-level `pruneOlderThan(namespace, ms)` wrapper on the
  index tier would complete the loop.
- **Embed cache sharing with the CLI brain** — one `embed_cache` table per file already allows
  `ailang cache` and game runs to share cached vectors when pointed at the same DB.
- **Multimodal embed parity** — `_ollama_embed_media` has no provider-configured equivalent;
  deferred until a provider need is measured.

---

**Document created**: 2026-10-13
**Last updated**: 2026-10-13

## Implementation verification (2026-10-16)

Implemented against the approved sprint plan, which superseded this design's broad local full-suite gate with changed packages plus test-core; CI retains the full suite. All six changed packages, test-core, lint, stdlib (48 interfaces), CLI docs, formatting, file sizes, changelog and architecture boundaries pass. The manifest has zero module drift. The new strict two-process showcase and named regression fixtures pass (the optional Ollama fixture is type-checked only). Aggregate verify-examples retains five unchanged baseline failures for unexported main functions; these are recorded under the approved unattended-run baseline-failure policy. Independent evaluation: 98/100, pass.
