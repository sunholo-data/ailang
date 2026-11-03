# Persistent Semantic Store: SharedMem/SharedIndex SQLite Backends + std/ai.embed

**Status**: Planned
**Target**: v0.57.1
**Priority**: P1 (Medium) — unblocks a shipped consumer's headline use case (Stapledon's Voyage cross-run content library); no language-correctness bug
**Estimated**: 5–6 days (~10h flags+wiring, ~8h SQLite SharedIndex, ~8h std/ai.embed, ~6h cache, ~6h docs/tests/examples + buffer)
**Dependencies**: None — reuses three shipped seams: the `effects.OpenSharedCache` registry (M-BRAIN / M-V1-SIMPLIFY-S3), the `messaging` embedder factory, and `ai.ClassifyError` typed errors

**Reported by**: `stapledons_godot` via ailang messages. Use case (design repo `vision/premise-and-loop.md` §6): a **cross-run content library** — content generated in one playthrough (a "universe") is found by semantic near-match in later ones; also crew memories and the Archive's decaying retrieval index.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The SQLite SharedIndex preserves the Strict contract ((score DESC, key ASC), lexicographic `maxScan` prefix); cache keys are content hashes, so cached and fresh paths return identical vectors for identical (model, text) |
| A2: Replayability | 0 | No trace-shape change |
| A3: Effect Legibility | +1 | `std/ai.embed` declares `! {AI}` and passes the AI cap+budget gate like every other AI op; the store flags are operator-side, so the program-visible effect surface is unchanged |
| A4: Explicit Authority | +1 | Flags require their capability (`--sharedmem-db` needs `--caps SharedMem`) — persistence grants no ambient access the program didn't already have |
| A5: Bounded Verification | 0 | No verification-surface change |
| A6: Safe Concurrency | +1 | SQLite WAL + 5s busy timeout (the `sqliteopen` recipe) makes two-process sharing sound, where today's in-memory backends silently cannot share at all |
| A7: Machines First | 0 | No syntax change; one typed stdlib export |
| A8: Minimal Syntax | +1 | Flags + one stdlib function; zero grammar change |
| A9: Cost Visibility | +1 | Embed calls consume the AI budget (`RequireCapWithBudget`); cache hits cost zero provider calls and are countable via `Stats()`/`EmbeddingStats()` |
| A10: Composability | +1 | `embed` composes with `std/embedding` (cosine), `std/sharedindex.upsertWithEmbedding`, and `std/sem` frames — the reporter's exact loop |
| A11: Structured Failure | +1 | `Result[[float], AIError]` replaces `_ollama_embed`'s untyped Go-error crash |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — default (no flags) behavior is byte-identical to today
- [x] A3 (Effects): No hidden side effects — persistence is opt-in via operator flags; `embed` is declared `! {AI}`
- [x] A4 (Authority): No ambient access granted — capability gates unchanged and enforced
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

---

## Problem Statement

The reporter's game wants memories that outlive a process. Four verified gaps block it:

**Current State:**

1. **SharedMem/SharedIndex are always in-memory at runtime.** `SetupSharedMemHandler` unconditionally does `effects.NewSharedMemContext(nil)` (`internal/runner/handlers.go:212`) and `SetupSharedIndexHandler` does `effects.NewSharedIndexContext(nil)` (`handlers.go:308-312`). The code itself says `// Future: could add flags for Redis, memcached, etc.` Nothing survives a restart; two processes cannot share a store. Every "crew memory" or "Archive index" built in one run evaporates at exit.
2. **A production-shaped SQLite backend already exists but is unreachable from a run.** `internal/platform/sharedmem/sqlite.go` (702 LOC) implements `effects.BrainCache` — and therefore the `effects.SharedCache` byte contract SharedMem needs — with namespaces, TTL/GC, WAL, embeddings, deterministic sorts. It registers at binary start (`cmd/ailang/platform_init.go:18` → `effects.RegisterSharedCacheOpener`). But only the **CLI brain** opens it (`ailang cache …` → `effects.NewBrainStore` → `OpenSharedCache`, `cmd/ailang/cache.go:69-73`). AILANG programs — the game sidecar — cannot reach it at all.
3. **The only embedding reachable from AILANG is `_ollama_embed`.** It is a raw builtin (no import, no std wrapper), takes the model per call, has no cache, and fails with a **typed-as-nothing Go error** that crashes the host (`internal/builtins/ollama_embed.go`). The Gemini and OpenAI embedders (`internal/messaging/embedder_gemini.go`, `embedder_openai.go`) serve only the CLI brain and messaging. The reporter's premise: *players bring their own Gemini/OpenRouter key; Ollama is not guaranteed on their machines* — so `_ollama_embed` is not a path players can rely on.
4. **The docs already promise what this design delivers.** `docs/docs/guides/semantic-caching-how-to.mdx:17`: "**SharedMem**: A persistent key-value store where agents park their thoughts" — false today (process-local), true only after this design. Line 18's SharedIndex bullet inherits the same correction.

**Impact:** Stapledon's Voyage (and any agent fleet using cross-run memory — this is also the `ailang docs search` neural index's own pattern) must rebuild its library every process, and AILANG code on player machines cannot embed at all without Ollama. Reported via the message plane by the game's own agent — a first-class consumer, not a hypothetical.

---

## Goals

**Primary Goal:** A run started with `--sharedmem-db`/`--sharedindex-db` (or env equivalents) reads and writes a SQLite store that survives restarts and is shareable across processes, and `std/ai.embed(text)` returns provider-configured embeddings as `Result[[float], AIError]` with an optional content-hash cache — measured by cross-process round-trip integration tests.

**Success Metrics:**
- Frames `put` in run A are `get`-able in run B against the same `--sharedmem-db` (integration test, two sequential `Run` calls and/or two processes)
- `std/sharedindex` upsert→findSimhash/findByEmbedding round-trips across processes against `--sharedindex-db`, with Strict ordering identical to the in-memory backend on the same state
- `std/ai.embed` type-checks (verified with `ailang check`), returns `Err(AIError)` — not a crash — when no embedder is configured, and `Ok([float])` under a real or stub embedder config
- With the cache enabled, embedding the same text twice makes **one** provider call (asserted via a stub embedder call counter)
- Guide line 17-18 corrected; new flags appear in `ailang run --help`

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| **D1: Reuse the registered SQLite opener (`effects.OpenSharedCache`) for SharedMem persistence** — no new backend | The seam, WAL recipe, schema and migration logic all exist (M-BRAIN); a parallel backend would fork pragma/migration code | agent | design | med |
| **D2: SharedIndex persistence = new `SQLiteSharedIndex` behind a new `SharedIndexOpener` seam, in its own table (`sharedindex_entries`), NOT `brain_frames`** | Index entries are `(namespace, key)`-shaped metadata with version+timestamp; `brain_frames` keys are global and frame-shaped. Conflating them would make SharedMem keys and index keys collide and corrupt both | agent | design | high |
| **D3: `std/ai.embed` routes through a provider-configured `effects.Embedder` on `EffContext` (the messaging embed config surface), NOT through `AIHandler`** | `AIHandler` has no `Embed` method and `internal/ai` chat handlers are not embedders; the messaging embedders (`ollama/openai/gemini`) are shipped, provider-configured, and key-driven | agent | design | med |
| **D4: `embed` failure surface = `Result[[float], AIError]` via `ai.ClassifyError`, effect `! {AI}` (cap + budget gated)** | Fixes `_ollama_embed`'s untyped crash; matches the `_ai_call_result` precedent exactly | compiler (precedent) | design | low |
| **D5: Content-hash cache = a namespace in the `--sharedmem-db` file (opt-in `--embed-cache`), not a separate cache flag/file** | One store per run ("the universe"); rows land in the existing `brain_frames` schema (embedding/embed_model/expires_at columns) so `ailang cache` tooling and TTL/GC already manage them | human at review | design | med |
| **D6: No silent fallbacks** — `--sharedmem-db` without `--caps SharedMem` is a hard error; opener/open failures are hard errors; a requested-but-unavailable embedder is a startup error; a `nil` embedder makes `embed` return `Err(AIError{ProviderNotFound})`, never a crash, never a zero vector | Doctrine: a flag that silently does nothing is a false guarantee (see `--net-max-bytes` precedent) | human (doctrine) | design | low |
| **D7: ANN index and a read-only brain effect are out of scope** (Future Work) | Keeps this at flag+std surface; the `SharedIndex` interface hides scan strategy so an ANN can later drop in behind the same flags | human | design | low |

### Design Freeze

All high-change-cost rows are resolved before coding:

- [x] D1 — reuse `effects.OpenSharedCache` + `SQLiteSharedCache` as the SharedMem `Cache` (it satisfies `effects.SharedCache`: Get/Put/Delete/CAS/Keys/Len)
- [x] D2 — new `sharedindex_entries` table + `SQLiteSharedIndex` + `SharedIndexOpener` registry seam in `internal/effects`
- [x] D3 — `EffContext.Embedder` wired from `messaging.LoadEmbedConfigFromEnv()` + `NewEmbedderFromConfig`
- [x] D4 — `Result[[float], AIError]`, `ai.ClassifyError`, effect `! {AI}`
- [x] D5 — cache namespace `"embed-cache"` inside the `--sharedmem-db` file, keyed `sha256(model ∥ text)`
- [x] D6 — loud failures on every misconfiguration
- [x] D7 — ANN + read-only brain effect deferred with named seams

---

## Solution Design

### Overview

Three workstreams on top of shipped seams, plus a docs correction:

1. **Persistence flags.** `--sharedmem-db <path>` and `--sharedindex-db <path>` (env fallbacks `AILANG_SHAREDMEM_DB` / `AILANG_SHAREDINDEX_DB`, flag wins — the `--fs-max-bytes` / `AILANG_FS_MAX_BYTES` pattern). Empty → today's in-memory default, byte-identical. Set → the runner opens the persistent backend and installs it as the effect context's store. Each flag requires its capability grant. Both may point at the **same file** (separate tables; SharedMem uses `brain_frames`, SharedIndex uses `sharedindex_entries`) or different files.

2. **`std/ai.embed`.** New `_ai_embed` builtin (module `std/ai`), type `(text: string) -> Result[[float], AIError] ! {AI}`, exported as `std/ai.embed(text)`. The runner builds an `effects.Embedder` from the same embedding config surface the CLI brain uses (`embeddings.provider` in `~/.ailang/config.yaml` + `AILANG_EMBED_*` env vars; providers `ollama | openai | gemini | none`), stores it on `EffContext`, and the builtin classifies failures through `ai.ClassifyError` into the `AIError` record — the same shape `callResult` already returns.

3. **Content-hash cache (opt-in).** `--embed-cache` enables it (requires a persistent SharedMem store; hard error otherwise). Key: `sha256(model ∥ "\x00" ∥ text)` in namespace `"embed-cache"`. Hit → return the stored vector (zero provider calls); miss → call the embedder, then `PutVector`. `--embed-cache-ttl <duration>` stamps `expires_at` so the Archive's *decaying* retrieval index falls out of the existing GC path (`GarbageCollect`).

### Architecture

**Components:**

1. **Persistent SharedMem (wiring only).** `SQLiteSharedCache` already satisfies `effects.SharedCache`. `SetupSharedMemHandler(effCtx, dbPath)`: `dbPath == ""` → `NewSharedMemContext(nil)` (today); else `effects.OpenSharedCache(dbPath)` → `NewSharedMemContext(cache)`. Opened through the registered platform opener — the core never links cgo sqlite.

2. **`SQLiteSharedIndex`** (`internal/platform/sharedmem/sqlite_index.go`, new). Implements `effects.SharedIndex` over:
   ```sql
   CREATE TABLE IF NOT EXISTS sharedindex_entries (
     namespace TEXT NOT NULL,
     key       TEXT NOT NULL,
     simhash   INTEGER NOT NULL,
     embedding BLOB,
     version   INTEGER NOT NULL,
     timestamp INTEGER NOT NULL,
     PRIMARY KEY (namespace, key)
   );
   ```
   `FindSimilarSimHash`/`FindSimilarByEmbedding` load the namespace's rows (`ORDER BY key` + `LIMIT maxScan` when `maxScan > 0`, preserving the lexicographic-prefix contract from M-SHAREDINDEX-MAXSCAN) and score in Go with the same deterministic sort as the in-memory backend: score DESC, key ASC. Cosine scoring reuses `simhash.Cosine`. Registered via a new `SharedIndexOpener` seam mirroring `sharedmem_backend.go` (`RegisterSharedIndexOpener` / `OpenSharedIndex` / `ErrBackendNotRegistered`).

3. **Embed wiring.** `EffContext` gains `Embedder effects.Embedder` (nil = unconfigured). Runner `SetupEmbedder`: `messaging.LoadEmbedConfigFromEnv()` → `NewEmbedderFromConfig`; `provider == "none"`/empty → nil embedder (no warning spam for runs that never call `embed`); explicit `--embed-provider` with an unknown value or missing key → startup error. A new `RegisterOp("AI", "embed", …)` op (in the `internal/effects` AI op registry, next to `callResult`) performs the call so the existing cap + budget + trace machinery applies unchanged.

4. **`_ai_embed` builtin + `std/ai.ail` export.** Registration follows `_ai_call_result` (`internal/builtins/ai_step.go:104-142`) with `aiErrorRecordType(T)`; stdlib wrapper `export func embed(text: string) -> Result[[float], AIError] ! {AI} = _ai_embed(text)` with the standard `@requires` doc comment.

5. **Embed cache.** Lives in builtin-adjacent code (`internal/builtins/ai_embed.go`), using `BrainFrame` columns that already exist: `key`, `namespace`, `embedding`, `embedding_dim`, `embed_model`, `expires_at`. No new table, no new migration.

### Implementation Plan

**Phase 1: SharedIndex persistence seam** (~8 hours)
- [ ] `internal/effects/sharedindex_backend.go`: `SharedIndexOpener`, `RegisterSharedIndexOpener`, `OpenSharedIndex`, reuse `ErrBackendNotRegistered`
- [ ] `internal/platform/sharedmem/sqlite_index.go`: schema + all 8 `SharedIndex` methods (Upsert, Delete, FindSimilarSimHash, EntryCount, Namespaces, UpsertWithEmbedding, FindSimilarByEmbedding)
- [ ] Unit tests: round-trip, Strict ordering vs in-memory on same state, namespace isolation, `maxScan` lexicographic-prefix semantics, two openers on one file

**Phase 2: Runner flags + wiring** (~10 hours)
- [ ] `cmd/ailang/main_run.go`: `--sharedmem-db`, `--sharedindex-db`, `--embed-cache`, `--embed-cache-ttl`, `--embed-provider`/`--embed-model` flags
- [ ] `internal/config`: `AILANG_SHAREDMEM_DB` / `AILANG_SHAREDINDEX_DB` accessors (flag-over-env precedence, `FSMaxBytes` pattern)
- [ ] `internal/runner/handlers.go`: `SetupSharedMemHandler(effCtx, dbPath)` / `SetupSharedIndexHandler(effCtx, dbPath)` — cap check, open, loud failure; delete the stale `Future: could add flags` comments
- [ ] `internal/runner/run.go` (`Options` fields) + `cmd/ailang/serve_api.go` + `internal/runner/batch.go` callsites
- [ ] `cmd/ailang/help.go` + `cmd/ailang/doctor_memory.go` flag docs; decide `cmd/ailang/run_policy.go` surface (see Deferred Decisions)
- [ ] Integration test: two sequential `Run` calls against one temp db file share frames; flag-without-cap errors

**Phase 3: `std/ai.embed`** (~8 hours)
- [ ] `internal/effects/context.go`: `Embedder` field + setter
- [ ] Runner `SetupEmbedder` from messaging config; explicit-flag validation
- [ ] `internal/effects` AI op `embed`; `internal/builtins/ai_embed.go`: `_ai_embed` builtin, `ClassifyError` → `AIError` mapping, `Result` construction
- [ ] `std/ai.ail`: export `embed` (doc comment: provider config, cache behavior, error codes)
- [ ] Tests: nil embedder → `Err(AIError{ProviderNotFound, retryable=false})`; stub embedder → `Ok` vector; `ailang check` on a program importing `std/ai (embed)`

**Phase 4: Content-hash cache** (~6 hours)
- [ ] Cache lookup/store under namespace `"embed-cache"` when `--embed-cache` + `--sharedmem-db`; hard error when `--embed-cache` set without a persistent store
- [ ] TTL via `expires_at` + GC-on-open (`GarbageCollect`) — the Archive decay knob
- [ ] Tests: hit/miss/expiry; one provider call for repeated text (stub call counter); cross-run reuse

**Phase 5: Docs + examples** (~6 hours)
- [ ] Fix `docs/docs/guides/semantic-caching-how-to.mdx:17-18` + add a Persistence section (flags, env vars, same-file/different-file, cache)
- [ ] New example `examples/reference/persistent_semantic_store.ail` (cross-run library: embed → upsertWithEmbedding in run A; findByEmbedding in run B)
- [ ] `ailang run --help` shows the new flags; stale comments removed

### Pipeline Pass Coverage

Not a compiler pipeline pass change — no `prog.Decls` / contract-expression lowering involved. (Checked: this design touches runtime wiring and builtins only.)

### Files to Modify/Create

**New files:**
- `internal/platform/sharedmem/sqlite_index.go` (~260 LOC) — `SQLiteSharedIndex` + `sharedindex_entries` schema
- `internal/platform/sharedmem/sqlite_index_test.go` (~200 LOC)
- `internal/effects/sharedindex_backend.go` (~70 LOC) — opener seam (mirror of `sharedmem_backend.go`)
- `internal/builtins/ai_embed.go` (~140 LOC) — `_ai_embed` + cache logic
- `examples/reference/persistent_semantic_store.ail` (~70 LOC)

**Modified files:**
- `internal/runner/handlers.go` (+45/−8) — persistent setup + signature change
- `internal/runner/run.go` (+12) — `Options` fields, pass-through
- `cmd/ailang/main_run.go` (+18) — flag definitions
- `cmd/ailang/ai_handlers.go` or a sibling (+25) — `SetupEmbedder` wiring (messaging config → `EffContext.Embedder`)
- `internal/effects/context.go` (+5) — `Embedder` field
- `internal/effects/` AI op registration (+20, in the `RegisterOp("AI", …)` file) — `embed` op
- `internal/config/` (+15) — env accessors
- `cmd/ailang/serve_api.go`, `internal/runner/batch.go` (+4 each) — callsite updates
- `cmd/ailang/help.go` (+10), `cmd/ailang/doctor_memory.go` (+2), `cmd/ailang/run_policy.go` (+TBD) — docs/policy
- `std/ai.ail` (+12) — `embed` export
- `docs/docs/guides/semantic-caching-how-to.mdx (−2/+25)` — truth + persistence section

---

## Examples

### Example 1: Cross-run content library (the reporter's loop)

**Before (today):** impossible — run B starts with an empty in-memory store; run A's frames are gone. The only embedding path assumes local Ollama and crashes the host on any error:

```ailang
-- run A: index a generated universe description (in-memory only, dies at exit)
upsert("universe-A", "planet-3", simhash(text), 1, now)
let emb = _ollama_embed("embeddinggemma", text)   -- raw builtin; Go error crashes host
upsertWithEmbedding("universe-A", "planet-3", simhash(text), emb, 1, now)
```

**After (two runs, one store):**

**Run A (index a universe; literal timestamps as in shipped examples):**

```ailang
import std/ai (embed)
import std/sharedindex (upsertWithEmbedding)
import std/simhash (simhash)
import std/io (println)

export func main() -> () ! {AI, IO, SharedIndex} =
  let text = "Ocean world: storm belts, drowned cities, a shy archivist species" in
  match embed(text) {
    Ok(vec) => upsertWithEmbedding("universe-A", "planet-3", simhash(text), vec, 1, 1000),
    Err(e) => println("embed failed: ${e.code} — retryable=${show(e.retryable)}")
  }
```

**Run B (next playthrough, same `--sharedindex-db universe.db`):**

```ailang
import std/sharedindex (findByEmbedding)
import std/ai (embed)
import std/list (length)
import std/io (println)

export func main() -> () ! {AI, IO, SharedIndex} =
  match embed("watery world with ruins and a librarian creature") {
    Ok(query) => {
      let hits = findByEmbedding("universe-A", query, 5, 0, true)
      println("found ${show(length(hits))} echoes of a previous universe")
    },
    Err(e) => println("no embedding: ${e.message}")
  }
```

### Example 2: Decaying Archive index (TTL cache)

```bash
# Embed results cached in the same store, entries decay after 30 days:
ailang run --caps AI,SharedMem --sharedmem-db archive.db \
  --embed-provider gemini --embed-cache --embed-cache-ttl 720h game.ail
# archive.db rows in namespace "embed-cache" expire via GarbageCollect on next open
```

### Example 3: Flag misuse fails loudly (no silent fallback)

```bash
$ ailang run --sharedmem-db store.db game.ail
Error: --sharedmem-db needs --caps SharedMem        # not an ignored flag

$ ailang run --caps SharedMem --embed-cache game.ail   # but no --sharedmem-db
Error: --embed-cache needs --sharedmem-db (the cache is a namespace in that store)
```

---

## Success Criteria

- [ ] Two sequential runs against one `--sharedmem-db` file share frames (integration test)
- [ ] `std/sharedindex` upsert/find round-trips across runs against `--sharedindex-db`; Strict ordering matches the in-memory backend on identical state
- [ ] `std/ai.embed` type-checks (`ailang check` on a program importing it) and returns `Result[[float], AIError]`
- [ ] No embedder configured → `Err(AIError{code:"ProviderNotFound", retryable:false})`, not a crash; provider errors classify via `ai.ClassifyError`
- [ ] Cache enabled: repeated text = 1 provider call (stub-counter test); TTL rows expire via GC
- [ ] Default (no flags) behavior byte-identical to today (regression: full existing SharedMem/SharedIndex suites pass unchanged)
- [ ] `make simplicity-audit` shows no command/env-route regression (4 new flags + 2 env vars documented in help)
- [ ] All tests passing (`make test`); documentation updated (guide line 17-18 + persistence section); example added and `ailang check`-clean
- [ ] Engine parity: evaluator and strict VM agree (the strict VM resolves the same builtin specs — verified pattern, no VM-side index code exists)

---

## Conflict Surface

This design touches **runtime wiring, one builtin, and flags** — no parser, lexer, AST, typechecker, codegen, or effect-row-algebra change. For completeness, the surfaces it does extend:

### Positions touched

- **`std/ai` export list**: adds `embed` alongside `callResult`/`callJsonResult`/`step`/…. Disambiguation: ordinary name binding; no grammar shared. Existing importers unaffected (records are open at the value level; the module adds a new export).
- **Builtin namespace `std/ai`**: adds `_ai_embed` (raw name). No existing builtin named `_ai_embed` (verified: `grep -rn '"_ai_embed"' internal/` → empty; nearest are `_ai_call_result`/`_ai_step`).
- **`ailang run` flag set**: adds 6 flags. Verified unclaimed: `grep sharedmem-db|sharedindex-db|embed-cache|embed-provider` → empty today.
- **`EffContext` fields**: adds `Embedder` beside `AI`/`SharedMem`/`SharedIndex`; `SetupSharedMemHandler`/`SetupSharedIndexHandler` gain one parameter each — all three callsites (`run.go`, `batch.go`, `serve_api.go`) updated in the same change.

### Programs that MUST still work

1. `examples/runnable/stdlib_sharedindex.ail` — in-memory SharedIndex path unchanged (verified type-checks today)
2. `examples/reference/sharedmem_cache.ail` — in-memory SharedMem path unchanged
3. `examples/reference/neural_semantic_search.ail` — uses `_sharedindex_find_by_embedding` with Strict+maxScan; the SQLite backend must reproduce its ordering
4. Any program calling `_ollama_embed` — untouched, stays as-is (back-compat; `embed` is additive)
5. `ailang cache …` CLI brain on existing `brain.db` files — `brain_frames` schema unchanged; `sharedindex_entries` is a new table in the same file

### What deliberately changes

- `SetupSharedMemHandler`/`SetupSharedIndexHandler` **signatures** (internal Go API, not language surface) — all in-repo callsites enumerated above.
- Guide line 17-18 wording (docs bug fix).
- Nothing in the language grammar, type system, or wire shapes of existing builtins.

---

## Testing Strategy

**Unit tests:**
- `SQLiteSharedIndex`: upsert/delete/find/count/namespaces, Strict ordering determinism (fixed state → fixed results), namespace isolation, embedding round-trip (encode/decode via `effects.EncodeEmbedding`), `maxScan` lexicographic-prefix semantics, same-file double-open (WAL busy behavior)
- `_ai_embed`: nil embedder → typed `Err`; stub embedder → `Ok`; `ClassifyError` code mapping (auth/rate-limit/timeout cases)
- Cache: hit/miss/expiry; provider-call counting

**Integration tests:**
- Two `Run` invocations over one temp `.db` file (SharedMem get across runs; SharedIndex find across runs)
- Flag misuse matrix (D6): each new flag without its prerequisite → the exact error text
- `ailang check` + `ailang run` (AI stub or stub embedder) on the new example

**Regression-surface tests** (per Conflict Surface list): the four existing example programs above remain check-clean; full existing SharedMem/SharedIndex test suites pass unchanged.

**Manual testing:**
- `ailang run --help` renders the new flags; guide page renders correctly
- A real Gemini-key run embeds and round-trips (operator smoke; can't run in CI)

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact flag-parse location and `Options` field grouping in `cmd/ailang/main_run.go` — agent may choose
- `sharedindex_entries` index set (beyond the PK) — agent may choose; measure before adding
- Whether the `embed` AI op lives in `internal/effects/ai_step.go`-style registration or a new file — agent may choose
- Cache-namespace name (`"embed-cache"` is suggested, not mandated) — agent may choose
- Whether `--embed-cache-ttl` accepts Go durations only or also `Nd` forms — agent may choose (precedent: `--net-timeout` Go durations)
- `run_policy.go` surface: whether the new flags are policy-overridable or operator-only — **human at review** (check the hardening test's flag enumeration)
- OpenRouter embeddings path: the openai embedder has no base-URL override today; if players must use an OpenRouter key, either add `openai.base_url` to the embeddings config or an `openrouter` provider case — **human at review** (depends on whether OpenRouter exposes an OpenAI-compatible embeddings endpoint at review time; do not assert)
- Batch-embed API (`embedBatch`) — agent may choose to expose later via the same op (out of scope here)

## Non-Goals

**Not in this feature:**
- **ANN / vector index (sqlite-vec, HNSW)** — all scans stay linear-in-namespace; the `SharedIndex` interface hides scan strategy, so an ANN backend can drop in behind the same flag later (reporter's "nice to have", sequenced as Future Work)
- **A read-only brain effect** (exposing `ailang cache` tiers to AILANG programs) — new effect surface, separate doc (reporter's "nice to have")
- **Redis/Firestore backends** — the old DX-18 sketch (`design_docs/implemented/v0_5_11/semantic-caching-future.md`) proposed these; SQLite behind the existing opener seam supersedes it
- **`_ollama_embed` deprecation** — untouched, back-compat
- **SharedMem/SharedIndex wire shapes** — `std/sharedmem.ail` and `std/sharedindex.ail` gain nothing; persistence is invisible to the language

## Timeline

**Week 1** (20 hours):
- Phase 1: SharedIndex persistence seam (8h)
- Phase 2: Runner flags + wiring (10h)
- Buffer (2h)

**Week 2** (20 hours):
- Phase 3: `std/ai.embed` (8h)
- Phase 4: content-hash cache (6h)
- Phase 5: docs + examples (6h)

**Total: ~40 hours across 2 weeks** (initial estimates ×2 per house rules)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| SQLite SharedIndex ordering diverges from in-memory backend | High (silent wrong results) | Differential test: same fixture through both backends, assert identical Strict output |
| Two processes contend on one db file | Medium | WAL + 5s busy timeout already in `sqliteopen.Open`; document one-writer pattern; test two openers |
| Embed cache returns vectors from a *different model* after operator changes `--embed-model` | High (silently wrong neighbors) | Cache key includes the model name (`sha256(model ∥ text)`); dimension mismatch also scored 0 by `Cosine` |
| New flags silently ignored under policy runs | Medium | Explicit decision in `run_policy.go` surface (Deferred, human at review); hardening test enumerates flags |
| Scope creep into ANN/brain effect | Medium | D7 freeze; Future Work section names the seams |
| `ClassifyError` misclassifies embedder transport errors | Low | Reuse the same classifier the AI effect already exercises; unit-test the three common codes |

## Verification Log

*(Every load-bearing claim, checked against code or a live run at dev `b23cadc` / `ailang` v0.57.0 / std v0.57.0 — matching the reporter's dev-329f4dd baseline.)*

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| 1 | `SetupSharedMemHandler` always installs an in-memory cache | Read `internal/runner/handlers.go:208-215` — `effCtx.SharedMem = effects.NewSharedMemContext(nil)` with comment `// Future: could add flags for Redis, memcached, etc.` | Confirmed |
| 2 | `SetupSharedIndexHandler` likewise | Read `internal/runner/handlers.go:306-312` — `effects.NewSharedIndexContext(nil)` | Confirmed |
| 3 | The SQLite backend exists and satisfies the byte contract SharedMem needs | Read `internal/platform/sharedmem/sqlite.go` (702 LOC): implements `effects.BrainCache` (compile-time `var _ effects.BrainCache`), which embeds `SharedCache` (Get/Put/Delete/CAS/Keys/Len all present); registered via `cmd/ailang/platform_init.go:18` → `sharedmem.Register()` → `effects.RegisterSharedCacheOpener(Open)` | Confirmed |
| 4 | Only the CLI brain opens it today | `grep -rn "OpenSharedCache\|NewSQLiteSharedCache" internal/ cmd/ --include="*.go"` (excl. platform pkg/tests) → only `internal/effects/brain.go:39,46` (NewBrainStore, called from `cmd/ailang/cache.go`) | Confirmed |
| 5 | `std/ai` does **not** export `embed` today | Live `ailang check` on a program with `import std/ai (embed)` → `Error: IMP010: symbol 'embed' not exported by 'std/ai'` | Confirmed (gap real) |
| 6 | `_ollama_embed` is a raw builtin, no cache, untyped failure | Read `internal/builtins/ollama_embed.go`: registered module `std/ai`, `Effect: "IO"`, `Impl` returns Go error from `ollamaEmbedRequest` (no Result, no cache); live `ailang check` on a program calling `_ollama_embed("m","t")` with `! {IO}` → clean (typechecks without import) | Confirmed |
| 7 | `AIHandler` has no `Embed` method | `grep -n "Embed" internal/effects/ai.go` → empty (interface: Call/CallJson/CallImage/CallImageBase64/Step/StepWithCache/StepWithStream) | Confirmed (D3 premise) |
| 8 | Gemini/OpenAI embedders exist but serve only CLI brain + messaging | `internal/messaging/embedder_gemini.go`, `embedder_openai.go` exist; `NewEmbedderFromConfig` dispatches `ollama\|openai\|gemini\|none` (read `embedder.go:215-241`); no other callers outside `messaging` + `cmd/ailang/cache.go:83-92` (createEmbedder) | Confirmed |
| 9 | No `--sharedmem-db` / `--sharedindex-db` flags or env vars exist | `grep -rn "sharedmem-db\|sharedindex-db\|SharedMemDB\|SharedIndexDB\|AILANG_SHAREDMEM\|AILANG_SHAREDINDEX" cmd/ internal/ docs/` → empty | Confirmed (names free) |
| 10 | `_ai_embed` builtin name is unallocated | `grep -rn '"_ai_embed"' internal/` → empty | Confirmed (name free) |
| 11 | `Result[[float], AIError]` is a well-formed AILANG annotation; `_ai_call_result` returns `Result[string, AIError]` | Live `ailang check` on `export func f(x: string) -> Result[[float], AIError] ! {AI} = _ai_call_result("x")` → parses/elaborates; fails only at return-type unification: "cannot unify list[float] with string" | Confirmed (shape valid; builtin precedent read in `internal/builtins/ai_step.go:104-152`) |
| 12 | The guide calls SharedMem persistent (docs bug) | `sed -n '15,19p' docs/docs/guides/semantic-caching-how-to.mdx` → line 17: `- **SharedMem**: A persistent key-value store where agents park their thoughts` | Confirmed (false today) |
| 13 | `effects.SharedIndex` is the seam a persistent backend must satisfy; only an in-memory impl exists | Read `internal/effects/sharedindex.go:82-149` (interface) + `sharedindex_inmemory.go`; `ls internal/platform/sharedmem/` → `sqlite.go` only (no index backend) | Confirmed (D2 gap) |
| 14 | `std/sharedindex` exports upsert/findSimhash/upsertWithEmbedding/findByEmbedding/namespaces/entryCount | Read `std/sharedindex.ail` (all exported, `! {SharedIndex}`); live `ailang check examples/runnable/stdlib_sharedindex.ail` → `✓ No errors found!` | Confirmed |
| 15 | All similarity scans are linear today (no ANN anywhere) | `grep -rn "vec0\|sqlite-vec\|hnsw\|faiss\|ANN index" internal/` → empty; `SearchByEmbedding`/`SearchBySimHash` in `sqlite.go` and both in-memory backends scan every row | Confirmed (reporter's "nice to have" is real) |
| 16 | No `Brain` capability/effect exists (read-only brain effect is net-new surface) | `grep -rn '"Brain"' internal/effects/ cmd/ailang/` → empty; `ls std/` → no brain module | Confirmed |
| 17 | Flag-then-env precedent for the new flags | Read `internal/runner/handlers.go:146-170` (`SetupFSLimit`: flag else `AILANG_FS_MAX_BYTES`) and `cmd/ailang/main_run.go:135` | Confirmed (pattern exists) |
| 18 | AI ops register via `RegisterOp("AI", name, op)` and builtins call `effects.Call` (cap+budget+trace machinery) | Read `internal/effects/ai_step.go:49` (`RegisterOp("AI", "callResult", …)`), `ops.go:83-103` (`Call` → `RequireCapWithBudget` → `Registry`), `internal/builtins/ai_step.go:151-156` (`aiCallResultImpl`) | Confirmed (D4 precedent) |
| 19 | `ClassifyError` + `AIError` codes exist for typed mapping | `internal/ai/errors.go:23,28,33` (`CodeAuthFailed`, `CodeRateLimit`, …); `std/ai.ail` exports `AIError = { code, message, retryable }` (read) | Confirmed |
| 20 | `BrainFrame` already carries embedding + model + TTL columns (cache needs no migration) | Read `internal/effects/sharedmem_backend.go:58-73` (`Embedding`, `EmbeddingDim`, `EmbedModel`, `ExpiresAt`) + `sqlite.go` schema/migration (`migrateBrainV2`) | Confirmed (D5 premise) |
| 21 | Two-process sharing is the WAL recipe's designed mode | `sqliteopen.Open` used by `NewSQLiteSharedCache` (WAL, NORMAL sync, 5s busy timeout, foreign keys — read `sqlite.go:74-82` doc + call) | Confirmed |

---

## Related Documents

**Historical (this topic's lineage — read, and deliberately superseded):**
- [Semantic Caching: Future Work](../implemented/v0_5_11/semantic-caching-future.md) — DX-18 proposed Redis/Firestore backends as *new* code; this design instead **wires the shipped SQLite backend** (`internal/platform/sharedmem`, M-BRAIN) into the runner. Distinct topic: no planned or implemented doc proposes flags/runner wiring or `std/ai.embed` (duplicate gate: searches on "persistent semantic store" and "sharedmem sharedindex semantic caching" returned no ≥0.65 neural match).
- [DX-15: Semantic Caching MVP](../implemented/v0_6_0/DX-15-semantic-caching-MVP.md) — shipped the in-memory SharedMem/SharedIndex effects this design persists
- [M-BRAIN / brain context](../implemented/v0_10_0/m-brain-context.md) — the CLI brain's two-tier persistent store this design opens to AILANG runs
- [m-sharedindex-maxscan-determinism](../implemented/v0_54_13/m-sharedindex-maxscan-determinism.md) — the Strict/maxScan ordering contract the SQLite backend MUST reproduce (differential test)

**Planned (checked for overlap):**
- `design_docs/planned/m-ai-call-context.md` — read; different topic (cancellable, deadline-carrying AI provider calls) but directly relevant: the new `embed` AI op should inherit the same bounded-call context once M-AI-CALL-CONTEXT lands

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Reporter's message: `ailang messages` (stapledons_godot) — cross-run content library, crew memories, decaying Archive index
- Design repo `vision/premise-and-loop.md` §6 — the use case (external repo)
- `docs/docs/guides/semantic-caching-how-to.mdx` — the guide this design makes true

## Future Work

- **ANN index backend** (sqlite-vec or in-process HNSW) behind the same `--sharedindex-db` flag — the `SharedIndex` interface hides scan strategy, so this is a drop-in backend change plus a flag like `--sharedindex-ann`
- **Read-only brain effect** — `! {Brain}` capability exposing `Get`/`ListRecent`/`SearchByText` over the CLI brain's two-tier store (read-only first; write via SharedMem store flags)
- **Batch embed** (`embedBatch`) through the same AI op — `Embedder.EmbedBatch` already exists
- **`std/sem` frame TTLs surfaced in AILANG** — the decaying-Archive knob as a stdlib function instead of a flag
- **OpenRouter embeddings provider case** (see Deferred Decisions) pending an OpenAI-compatible embeddings endpoint

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
**Author**: design-doc-creator (coordinator task `task-d812326a`)