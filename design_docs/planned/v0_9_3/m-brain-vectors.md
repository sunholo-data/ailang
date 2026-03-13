# M-BRAIN-VECTORS: Vector-Native Brain with Embedding Storage and Cosine Search

**Status**: Planned
**Target**: v0.9.4
**Priority**: P1 (High — bridges existing infra to real daily value)
**Estimated**: 5 days (20h implementation + 8h testing + 4h docs + buffer)
**Dependencies**: M-BRAIN (SQLite backend + CLI — ✅ completed v0.9.3), DX-16/17 (SharedIndex + sem_frame — ✅ v0.5.11)
**Milestone ID**: M-BRAIN-VECTORS
**Created**: 2026-03-13

---

## Problem Statement

M-BRAIN shipped a persistent semantic cache with text + SimHash search. But SimHash is a crude 64-bit fingerprint — it captures lexical similarity (word overlap) not semantic similarity. Two descriptions of the same concept using different words score poorly.

**Current limitations:**
- SimHash: 64-bit → only 64 bits of discrimination. "Fix race in scheduler" vs "Resolve concurrency bug in task runner" = low similarity despite identical meaning
- Text search: SQL LIKE is keyword-exact. "type inference" won't find "Hindley-Milner unification"
- No cross-modal matching: code, error traces, commit messages are different "languages" that can't be compared
- Machine-to-machine communication requires text intermediary — agents write English to explain patterns instead of working in vector space directly

**What exists but is disconnected:**
- `SharedIndex` has `UpsertWithEmbedding()` + `FindSimilarByEmbedding()` with cosine similarity — **in-memory only**
- `messaging.Embedder` interface supports Ollama, OpenAI, Gemini — **not wired to brain**
- `std/embedding.ail` has pure vector math (cosine, dot, normalize) — **unused**
- `sem_frame.embedding: Option[bytes]` field — **always None**
- Google's [Gemini Embedding 2](https://ai.google.dev/gemini-api/docs/models/gemini-embedding-2-preview) (March 2026): natively multimodal (text, image, video, audio, PDF), 3072-dim, flexible output (128–3072), task-specific instructions — **ideal cloud upgrade**

**The gap:** The brain stores text and uses it as the communication medium. But embeddings ARE the semantic content — text is just one way to produce them. Two code patterns that solve the same problem should be "near each other" in vector space even if described completely differently.

---

## Goals

**Primary Goal:** Add real embedding vectors to the brain's persistent store, enabling cosine similarity search as the primary retrieval mechanism — with SimHash remaining as the fast/cheap fallback.

**Success Metrics:**
1. `ailang cache search "concurrency bug"` finds a frame stored as "fix race condition in scheduler" (score > 0.7 via cosine, vs 0.4 via SimHash)
2. Embedding generation is async/optional — brain works without an embedder running
3. `ailang cache put --embed` auto-generates and stores embedding alongside frame
4. Cosine similarity search on 10K frames completes in <200ms (SQLite scan + Go-side cosine)
5. Embedding model is pluggable: Ollama locally, Gemini Embedding 2 in cloud, OpenAI as alternative
6. Vector-only frames: store embedding + opaque payload with no text content (machine-to-machine)

---

## Non-Goals

- **ANN index (HNSW, IVF)** — brute-force cosine scan is fine for <100K frames. ANN adds complexity for marginal gain at this scale.
- **In-process model (llama.cpp, MLX)** — external embedder (Ollama, API) keeps the brain lightweight
- **Fixed-size vector type** (`vector[float; 768]`) — language-level type is a separate design doc
- **Multi-agent shared vector DB** — single-developer two-tier (user + project) first
- **Training/fine-tuning** — use off-the-shelf models
- **Image/video embedding** — text + code first; multimodal embedding is the cloud upgrade path

---

## Solution Design

### Architecture

```
                         ┌──────────────────┐
                         │  ailang cache put │
                         │    --embed        │
                         └────────┬─────────┘
                                  │
                    ┌─────────────▼──────────────┐
                    │    BrainStore (two-tier)    │
                    │   user.db  +  project.db   │
                    └─────────────┬──────────────┘
                                  │
            ┌─────────────────────┼──────────────────────┐
            │                     │                      │
     ┌──────▼──────┐    ┌───────▼────────┐    ┌────────▼────────┐
     │  SimHash     │    │  Embedding     │    │  Text/LIKE      │
     │  (fast,      │    │  BLOB column   │    │  (keyword       │
     │   always)    │    │  + cosine scan │    │   fallback)     │
     └─────────────┘    └───────┬────────┘    └─────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │   Embedder interface  │
                    │   (async, optional)   │
                    ├───────────────────────┤
                    │ Local:  Ollama/Gemma  │
                    │ Cloud:  Gemini Emb 2  │
                    │ Alt:    OpenAI        │
                    │ None:   SimHash-only  │
                    └───────────────────────┘
```

### Three-Tier Search Strategy

Search now has three paths, merged by score:

1. **Cosine similarity** (best quality, requires embeddings) — `embedding BLOB` column, Go-side `cosineSimilarity()` over all frames. Score: `(cosine + 1.0) / 2.0` normalized to [0, 1].
2. **SimHash similarity** (fast, always available) — existing `hammingDistance64`. Score: `1.0 - (hamming / 64.0)`.
3. **Text keyword** (fallback) — existing SQL LIKE. Score: 1.0 for matches.

Merge: deduplicate by key, prefer cosine score when available, boost cosine results by +0.1 over SimHash-only results. This means embedding-backed results naturally rank higher.

### Schema Migration (v1 → v2)

```sql
-- Schema v2.0.0: Add embedding columns
ALTER TABLE brain_frames ADD COLUMN embedding     BLOB;      -- packed float32 LE
ALTER TABLE brain_frames ADD COLUMN embedding_dim  INTEGER DEFAULT 0;
ALTER TABLE brain_frames ADD COLUMN embed_model    TEXT;      -- e.g. "ollama:nomic-embed-text", "gemini:gemini-embedding-2-preview"

-- Index on embedding presence (for skipping non-embedded frames in cosine scan)
CREATE INDEX IF NOT EXISTS idx_brain_has_embedding
  ON brain_frames(namespace) WHERE embedding IS NOT NULL;
```

Migration is additive — existing v1 databases work without modification. New columns are nullable. The schema version bumps to `2.0.0`.

### Embedding Storage Format

Embeddings are stored as **packed IEEE 754 float32 little-endian** (same as `_embedding_encode` builtin):
- 768-dim embedding → 3072 bytes
- 1536-dim → 6144 bytes
- 3072-dim (Gemini Embedding 2 full) → 12288 bytes

This is compact and matches the existing `sem_frame.embedding` convention. No JSON encoding overhead.

### Embedder Integration

Reuse the existing `messaging.Embedder` interface:

```go
type Embedder interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    Dimension() int
    ModelName() string
}
```

The brain accepts an optional `Embedder` at construction:

```go
type SQLiteSharedCache struct {
    db       *sql.DB
    embedder messaging.Embedder // nil = SimHash-only mode
}

func NewSQLiteSharedCache(dbPath string, opts ...CacheOption) (*SQLiteSharedCache, error)

type CacheOption func(*SQLiteSharedCache)

func WithEmbedder(e messaging.Embedder) CacheOption { ... }
```

When `embedder` is non-nil:
- `PutFrame()` auto-generates embedding from `content` field
- `SearchByEmbedding()` becomes available
- `SearchBySimHash()` still works (SimHash is always computed)

When `embedder` is nil:
- Everything works as before (SimHash + text search)
- Embedding column stays NULL
- No external dependencies required

### Vector-Only Frames

New frame type for machine-to-machine communication:

```go
// PutVector stores a frame with embedding but no human-readable content.
// The opaque payload is arbitrary bytes (e.g., serialized AST, binary data).
func (c *SQLiteSharedCache) PutVector(key, namespace string, embedding []float32, payload []byte, source string) error
```

CLI:
```bash
# Store embedding directly (piped from external tool)
echo '{"embedding": [0.1, 0.2, ...], "payload": "base64..."}' | ailang cache put-vector --ns code-fingerprints my_key
```

Use cases:
- Code fingerprints: embed function ASTs, find similar implementations across projects
- Error signatures: embed stack traces, find prior resolutions without text description
- Pattern transfer: agent A computes embedding of a solution, agent B finds it by proximity

### CLI Changes

```bash
# Existing commands gain --embed flag:
ailang cache put --content "..." --embed my_key         # Auto-embed content
ailang cache put --content "..." --embed --model gemini  my_key  # Specific model

# New command: embed existing frames that lack embeddings
ailang cache embed [--namespace NS] [--model MODEL]     # Backfill embeddings

# Search now uses cosine when embeddings available:
ailang cache search "query"              # Auto: cosine if available, SimHash fallback
ailang cache search "query" --simhash    # Force SimHash-only (fast)
ailang cache search "query" --cosine     # Force cosine-only (accurate)

# New: vector-only storage
ailang cache put-vector --ns NS KEY      # Reads JSON from stdin

# Stats shows embedding coverage:
ailang cache stats
# ━━━ project brain ━━━
#   Total frames: 150
#   With embeddings: 120 (80%)
#   Embedding model: ollama:nomic-embed-text (768-dim)
#   ...
```

### Embedding Model Configuration

Extends existing `~/.ailang/config.yaml`:

```yaml
brain:
  enabled: true
  embedding:
    provider: "ollama"           # "ollama", "gemini", "openai", "none"
    ollama:
      model: "nomic-embed-text"  # or "embeddinggemma"
      endpoint: "http://localhost:11434"
      dimension: 768
    gemini:
      model: "gemini-embedding-2-preview"
      dimension: 768             # can go up to 3072
      task_type: "RETRIEVAL_DOCUMENT"  # Gemini Embedding 2 supports task instructions
    openai:
      model: "text-embedding-3-small"
      dimension: 1536
```

Environment variable override: `AILANG_BRAIN_EMBED_PROVIDER=gemini`

### Cloud Path: Gemini Embedding 2

[Gemini Embedding 2](https://blog.google/innovation-and-ai/models-and-research/gemini-models/gemini-embedding-2/) (released March 10, 2026) is the ideal cloud embedding model:

- **Natively multimodal**: text, image, video, audio, PDF in one unified embedding space
- **Flexible dimensions**: 128 to 3072 (choose cost/quality tradeoff)
- **Task-specific instructions**: `task:code_retrieval`, `task:search_document` improve relevance
- **8192 token input**: handles large code blocks
- Available via Gemini API (`gemini-embedding-2-preview`) and Vertex AI

This means future brain frames could embed:
- Code files → find semantically similar implementations
- Error screenshots → match to known resolutions
- Architecture diagrams → retrieve related patterns
- All in the same vector space, cross-modal search for free

### Hook Integration Updates

The SessionStart brain context hook upgrades from SimHash to cosine search when embeddings are available:

```bash
# brain_session_start logic (pseudocode):
if embedder_available:
    query_embedding = embed(recent_files_as_text)
    results = ailang cache search --cosine "$recent_files_text" --limit 3
else:
    results = ailang cache search "$recent_files_text" --limit 3  # SimHash fallback
```

The post-commit resolution hook gains `--embed` flag to auto-embed resolutions:

```bash
ailang cache put-resolution --commit-msg "$msg" --files "$files" --embed
```

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Cosine similarity is deterministic for same embeddings. Embedding generation is deterministic per model. Deterministic sort: (score DESC, tier, key ASC) |
| A2: Replayability | +1 | Embedding model + version stored per-frame (`embed_model` column). Replay possible by re-embedding with same model |
| A3: Effect Legibility | +1 | Embedding generation is an explicit IO effect (requires `Embedder`). SimHash-only mode has no external deps. `--embed` flag is opt-in |
| A4: Explicit Authority | +1 | Embedder requires configuration (API key, endpoint). No ambient access. `provider: "none"` explicitly disables |
| A5: Bounded Verification | +1 | Cosine similarity is locally computable. No global reasoning needed. Embedding quality verifiable via `ailang cache search --cosine` |
| A6: Safe Concurrency | 0 | Single-writer SQLite (unchanged from M-BRAIN). No new concurrency concerns |
| A7: Machines First | +1 | **Core of this feature.** Vectors ARE the machine-native semantic representation. Text is one input modality, not the storage format |
| A8: Minimal Syntax | 0 | No new language syntax. CLI flags only (`--embed`, `--cosine`, `--simhash`) |
| A9: Cost Visibility | +1 | Embedding dimension stored per-frame. `stats` shows storage size. API-based embedders have explicit cost (tokens/request) |
| A10: Composability | +1 | Three search paths compose cleanly (cosine + SimHash + text). Vector-only frames compose with text frames in same index |
| A11: Structured Failure | +1 | Embedder unavailable → graceful fallback to SimHash. No silent degradation — `stats` shows "0% embedded" |
| A12: System Boundary | +1 | Embedding generation crosses system boundary (Ollama/API). Explicitly marked by `embed_model` field. `--embed` flag makes the crossing visible |

**Net Score: +10** (no violations, strong alignment with A7 Machines First)

---

## Implementation Plan

### M1: Schema Migration + Embedding Storage (~300 LOC, 1 day)

- Add `embedding BLOB`, `embedding_dim INTEGER`, `embed_model TEXT` to brain_frames
- Schema migration: detect v1 → add columns, bump to v2
- `PutFrame()` accepts optional `[]float32` embedding
- `PutVector()` for embedding-only frames
- `SearchByEmbedding()` with brute-force cosine scan
- `encodeEmbedding()`/`decodeEmbedding()` using IEEE 754 float32 LE
- Tests: round-trip encode/decode, cosine search accuracy, mixed frames (some with/without embeddings)

### M2: Embedder Wiring + Auto-Embed (~250 LOC, 1 day)

- `CacheOption WithEmbedder(e messaging.Embedder)` on `NewSQLiteSharedCache`
- `BrainStore` passes embedder to both tiers
- `PutFrame()` auto-embeds `content` when embedder present
- `ailang cache put --embed` flag triggers embedding
- `ailang cache embed` command: backfill embeddings on existing frames
- Config: `brain.embedding.provider` in config.yaml
- Tests: mock embedder, auto-embed on put, backfill, provider selection

### M3: Three-Tier Search Merge (~200 LOC, 1 day)

- `BrainStore.Search()` upgraded: cosine → SimHash → text, merged by score
- Cosine results get +0.1 boost over SimHash-only
- `--cosine` and `--simhash` flags for forcing search path
- `ailang cache search` auto-selects best path based on embedding coverage
- Hook: session_start uses cosine when available
- Hook: brain_resolution gains `--embed`
- Tests: three-tier merge correctness, boost ranking, fallback behavior

### M4: Vector-Only Frames + Polish (~250 LOC, 1 day)

- `ailang cache put-vector` CLI command
- `ailang cache stats` shows embedding coverage, model, dimensions
- `ailang cache export/import` handles embedding BLOB (base64 in JSONL)
- User guide update
- CHANGELOG update
- Tests: vector-only put/search, export/import round-trip with embeddings

### M5: Gemini Embedding 2 Provider (~200 LOC, 1 day)

- `GeminiBrainEmbedder` using Gemini API `gemini-embedding-2-preview`
- Task-type support (`RETRIEVAL_DOCUMENT`, `CODE_RETRIEVAL`, `SEARCH_QUERY`)
- Dimension selection (768 default, configurable up to 3072)
- Integration test with real API (skipped in CI unless `GOOGLE_API_KEY` set)
- Tests: mock Gemini API, dimension handling, task-type routing

---

## Success Criteria

- [ ] Cosine similarity search finds semantically similar frames that SimHash misses
- [ ] Brain works with zero embedder configured (SimHash fallback)
- [ ] `ailang cache put --embed` stores embedding alongside SimHash
- [ ] `ailang cache embed` backfills existing frames
- [ ] `ailang cache stats` shows embedding coverage percentage
- [ ] Vector-only frames work (no text content required)
- [ ] Schema migration from v1 → v2 is non-destructive
- [ ] Export/import round-trips embeddings correctly
- [ ] Cosine search on 10K frames < 200ms
- [ ] Ollama, Gemini, OpenAI providers all functional
- [ ] All existing M-BRAIN tests still pass
- [ ] `make test` and `make lint` pass
- [ ] User guide updated

---

## Timeline

| Day | Milestone | LOC | Key Deliverable |
|-----|-----------|-----|-----------------|
| 1 | M1: Schema + Storage | ~300 | Embedding BLOB, cosine scan, encode/decode |
| 2 | M2: Embedder Wiring | ~250 | Auto-embed on put, backfill command |
| 3 | M3: Three-Tier Search | ~200 | Cosine + SimHash + text merge |
| 4 | M4: Vector Frames | ~250 | put-vector, stats, export/import |
| 5 | M5: Gemini Provider | ~200 | Gemini Embedding 2 integration |

**Total: ~1,200 LOC estimated**

---

## Related Documents

- [M-BRAIN: Persistent Semantic Cache](m-brain.md) — prerequisite, SQLite backend + CLI
- [M-BRAIN Sprint Plan](m-brain-sprint-plan.md) — completed sprint
- [DX-15: Semantic Caching MVP](../../../design_docs/implemented/v0_6_0/DX-15-semantic-caching-MVP.md) — original SharedMem
- [DX-16: Deterministic Semantic Retrieval](../../../design_docs/implemented/v0_5_11/semantic-caching-future.md) — SharedIndex
- [M-SEMANTIC-ENVELOPE](../../../design_docs/implemented/v0_8_1/m-semantic-envelope.md) — the ambitious multi-aspect vision
- [Gemini Embedding 2 Docs](https://ai.google.dev/gemini-api/docs/models/gemini-embedding-2-preview) — Google's multimodal embedding model
- [Gemini Embedding 2 Blog](https://blog.google/innovation-and-ai/models-and-research/gemini-models/gemini-embedding-2/) — Announcement + capabilities

---

## Open Questions

1. **Dimension default**: 768 (compatible with local Gemma/nomic) or 1536 (better quality, 2x storage)?
   - Recommendation: 768 default, configurable. Most use cases don't need > 768 for code/text.

2. **Mixed-dimension frames**: If user switches model (768 → 1536), old embeddings are incompatible.
   - Recommendation: `embed_model` column tracks provenance. `ailang cache embed --force` re-embeds all. Search only compares frames with matching dimensions.

3. **Async embedding**: Should `put --embed` block until embedding is generated, or fire-and-forget?
   - Recommendation: Sync for CLI (user expects confirmation), async for hooks (must not block Claude Code).

4. **Vector-only search UX**: How to present results that have no text content?
   - Recommendation: Show key, namespace, score, dimension, and first N bytes of opaque payload as hex. Rich display deferred.
