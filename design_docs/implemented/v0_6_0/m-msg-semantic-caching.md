# M-MSG-SEMANTIC: Semantic Search + Safe Deduplication for AILANG Messages

**Status:** IMPLEMENTED
**Target:** v0.5.11
**Estimated LOC:** ~600 → **Actual:** ~650 LOC
**Implemented:** 2025-12-16
**Prerequisites:** DX-15/16/17 (SharedMem/SharedIndex + SimHash + embeddings + Ollama Embedder) ✅ in v0.5.11
**Dogfood Goal:** Make semantic caching the default way AILANG developers and agents triage and reuse message context.

---

## Problem Statement

The AILANG messaging system (`ailang messages`) currently supports only exact-match filters (inbox/from/unread/ack):

```bash
ailang messages list --inbox ailang_core
ailang messages list --from ailang
ailang messages list --unread
```

**Missing capabilities:**

1. **Semantic search:** find messages by meaning ("messages about parser bugs")
2. **Near-duplicate detection:** detect repeated agent outputs or repeated bug reports
3. **Similarity grouping:** cluster related messages for triage
4. **Content search:** search within title/payload without exact matching
5. **Collaboration safety:** allow dedupe in shared inboxes without destructive operations

---

## Goals

### Primary Goals

- Add `ailang messages search "<query>"` using SimHash (fast) and optional embeddings (semantic).
- Add `--similar-to MSG_ID` listing of similar messages.
- Add safe, reversible deduplication for shared inboxes.

### Success Metrics

1. **Latency:** <10ms SimHash search on 10K messages (SharedIndex path), acceptable fallback on SQLite scan
2. **Relevance:** top-3 contains relevant message for common dev queries
3. **Safety:** dedupe is reversible; no deletions; false positives minimized by high thresholds + report-first UX
4. **Adoption:** `messages search` becomes a primary workflow for triage

---

## Non-Goals (v0.5.12)

- True ANN indexes (BestEffort ANN backends)
- Cross-inbox/global search by default
- Cloud message backend (planned later; design must be compatible)

---

## Key Design Decisions

### 1) Inbox as project-scoped collaboration stream

Inbox names should derive from projects and be stable, e.g.:
- `ailang_core`
- `stapledon`

These inboxes are shared among humans and agents.

### 2) Namespace mapping (SharedIndex)

Use per-inbox namespaces to prevent cross-project collisions:
- `namespace = "messages:" + inbox`
- `key = "msg:" + msg_id`

### 3) Source of truth and acceleration

- **SQLite** remains the source of truth for message content and semantic fields.
- **SharedIndex** is an accelerator when capability is enabled.
- CLI must work even without SharedIndex (fallback).

### 4) Safe dedupe (no delete)

Dedupe must be reversible and collaboration-safe.

- **Never delete messages.**
- Mark duplicates by:
  - setting `dup_of = <representative_msg_id>`
  - setting `ack = true` (or equivalent ack marker)
- **Undo:** clear `dup_of` (and optionally restore ack state if tracked)

### 5) Role gating

Shared inboxes require guardrails.

- Report-only commands allowed to any reader.
- Mutating dedupe (`--apply`) requires maintainer/human role (default).

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         AILANG Messages CLI                          │
│  ailang messages search "parser bugs" --inbox ailang_core            │
│  ailang messages list --similar-to MSG_ID --inbox ailang_core        │
│  ailang messages dedupe --threshold 0.95 --inbox ailang_core         │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    internal/messaging/search.go                      │
│  - Search(query) -> []SearchHit                                      │
│  - FindSimilar(msgID) -> []SearchHit                                 │
│  - FindDuplicates(threshold) -> []DuplicateGroup                     │
│  - ApplyDuplicates(groups) -> writes dup_of + ack                    │
└─────────────────────────────────────────────────────────────────────┘
                      │                              │
                      ▼                              ▼
┌───────────────────────────────┐     ┌────────────────────────────────┐
│        SQLite (source)         │     │   SharedIndex (accelerator)    │
│ inbox_messages table           │     │ namespaces = messages:<inbox>  │
│ + simhash INTEGER              │     │ simhash search (deterministic) │
│ + embedding BLOB/TEXT          │     │ embedding search (cosine)      │
│ + embedding_model TEXT         │     │ trace logging (optional)       │
│ + embedding_updated_at INTEGER │     └────────────────────────────────┘
│ + dup_of TEXT                  │
└───────────────────────────────┘
```

---

## Inbox Inference

To reduce friction, `--inbox` may be omitted.

**Resolution order (deterministic):**

1. `--inbox` flag if provided
2. `.ailang/project.toml` (or similar) `project.inbox` if present
3. Git repo root folder name → sanitized inbox name
   - optional alias map: repo `ailang` → inbox `ailang_core`
4. Fallback: `default`

CLI should print the resolved inbox once:

```
Using inbox: ailang_core (inferred from repo)
```

---

## Canonical Search Text (critical)

Messages may contain stack traces and code blocks. Avoid aggressive normalization.

Define a single canonical function used for both simhash and embeddings:

**SearchText(msg):**
- `title`
- plus bounded `payload_excerpt` (default 16KB)
- preserve punctuation and code formatting
- collapse repeated whitespace only (optional)

**EmbeddingText(msg):**
- default: `title + "\n" + payload_excerpt`
- optional future optimization: prefer natural language portion (exclude code blocks) to reduce embedding noise

This prevents "search works differently in different modes."

---

## Data Model and Schema Changes

### inbox_messages additions

**Required for Phase 1 (SimHash):**
- `simhash INTEGER` (nullable during migration/backfill)
- index on `simhash` (helps filtering, not true similarity)

**Required for safe dedupe:**
- `dup_of TEXT NULL` (message ID pointer)
- optional: `dedupe_run_id TEXT NULL` (audit)

**Optional for neural (Phase 3):**
- `embedding BLOB/TEXT NULL` (store float vector as bytes/JSON; depends on current storage conventions)
- `embedding_model TEXT NULL` (e.g. `embeddinggemma`)
- `embedding_updated_at INTEGER NULL` (unix millis)
- optional: `embedding_dim INTEGER NULL`

**Note:** `_ollama_embed` returns `list[float]`. Persisting it can be JSON text (simple) or packed float32 (smaller). For v0.5.12, JSON is acceptable.

---

## API Design

### CLI Commands

**Semantic search:**

```bash
ailang messages search "parser error handling" --inbox ailang_core --limit 10
ailang messages search "parser error handling" --inbox ailang_core --threshold 0.80
ailang messages search "parser error handling" --inbox ailang_core --neural --limit 10
```

**Similar messages:**

```bash
ailang messages list --inbox ailang_core --similar-to msg_20251215_abc123 --limit 10
ailang messages list --inbox ailang_core --similar-to msg_20251215_abc123 --neural --limit 10
```

**Dedupe (safe by default):**

```bash
# Report only (default)
ailang messages dedupe --inbox ailang_core --threshold 0.95

# Apply (role-gated)
ailang messages dedupe --inbox ailang_core --threshold 0.95 --apply
```

**Optional: rebuild accelerator index:**

```bash
ailang messages index rebuild --inbox ailang_core
```

### CLI Output Requirements (explainability)

Every semantic query prints a footer:

- `backend=<SharedIndex|SQLite>`
- `mode=<Strict|BestEffort>`
- `score=<simhash|embedding>`
- `limit=<N> max_scan=<N> threshold=<X>`

Example:

```
backend=SharedIndex mode=Strict score=embedding limit=10 max_scan=500 threshold=0.80
```

---

### Go API

```go
type SearchOptions struct {
    Inbox         string
    Query         string
    Threshold     float64 // default 0.70
    Limit         int     // default 20
    MaxScan       int     // default 1000
    Neural        bool    // if true, embedding search
    Deterministic bool    // if true, Strict
}

type SearchHit struct {
    Msg       InboxMessage
    Score     float64
    ScoreKind string // "simhash" | "embedding"
}

func (s *Store) SemanticSearch(opts SearchOptions) ([]SearchHit, error)
func (s *Store) FindSimilar(inbox, msgID string, opts SearchOptions) ([]SearchHit, error)

type DuplicateGroup struct {
    Representative InboxMessage
    Duplicates     []InboxMessage
    MinScore       float64
    ScoreKind      string
}

// Report-only
func (s *Store) FindDuplicates(inbox string, threshold float64, neural bool) ([]DuplicateGroup, error)

// Mutating (role-gated by caller/CLI)
func (s *Store) ApplyDuplicates(inbox string, groups []DuplicateGroup, runID string) error
```

---

## Search Semantics

### SimHash scoring

```
score = 1.0 - (hamming_distance / 64.0)
```

### Embedding scoring

- cosine similarity computed inside SharedIndex (`_sharedindex_find_by_embedding`)

### Determinism

When `Deterministic=true`:
- stable ordering `(score DESC, key ASC)`
- if equal, key ordering applies
- guarantee holds given fixed inbox state

---

## Accelerator Usage and Fallback

### If SharedIndex capability is available

Use:
- `_sharedindex_find_simhash(namespace, simhash, limit, max_scan, deterministic)`
- or `_sharedindex_find_by_embedding(namespace, embedding, limit, max_scan, deterministic)` when `--neural`

Then join keys back to SQLite to fetch full messages.

### If SharedIndex is NOT available

Fallback to SQLite scan:
- compute query simhash
- scan messages in inbox (bounded by `max_scan` or full inbox)
- compute hamming distance
- return top-K with deterministic ordering

**Neural fallback:**
- optional: disallow without SharedIndex, or do "compute query embedding + brute cosine vs stored embeddings"
- **Recommendation for v0.5.12:** require SharedIndex for `--neural` unless you already have a cosine implementation in Go here.

---

## Deduplication Semantics (safe + reversible)

### Dedupe definition

A message B is a duplicate of A if:
- `similarity(A, B) >= threshold`
- similarity kind is simhash unless `--neural` is specified

### Representative selection (deterministic)

- choose the oldest by `(timestamp ASC, id ASC)`

### Apply operation (no deletion)

For each duplicate:
- set `dup_of = rep_id`
- set `ack = true`
- optionally set `dedupe_run_id = runID`

### Listing behavior

Add:
- `--collapsed`: hide messages where `dup_of IS NOT NULL`
- `--duplicates-of <id>`: list messages where `dup_of = <id>`

### Role gating

- `dedupe` (report) is allowed for readers
- `dedupe --apply` requires maintainer/human role

---

## Implementation Plan

### Phase 1: SimHash + safe dedupe schema (4 hours)

**Files:**
- `internal/messaging/schema.go` (migration)
- `internal/messaging/inbox.go` (compute simhash on insert/update)
- `internal/messaging/search.go` (SimHash search + dedupe report)

**Tasks:**
1. Add `simhash` + `dup_of` columns (+ indexes)
2. Backfill simhash for existing messages
3. Implement SimHash search (SharedIndex path + fallback scan)
4. Implement dedupe report (cluster groups)

### Phase 2: CLI integration + inbox inference + role gating (2-3 hours)

**Files:**
- `cmd/ailang/messages_search.go` (new)
- `cmd/ailang/messages_list.go` (add `--similar-to`, `--collapsed`, `--duplicates-of`)
- `cmd/ailang/messages_dedupe.go` (new)
- `cmd/ailang/messages_inbox_infer.go` (new helper)

**Tasks:**
1. Add `messages search`
2. Add `--similar-to`, `--collapsed`, `--duplicates-of`
3. Add `messages dedupe` report + `--apply` (role check)
4. Implement inbox inference (repo/config)
5. Print explainability footer

### Phase 3: Neural search (optional, 3 hours)

**Files:**
- `internal/messaging/schema.go` (embedding columns)
- `internal/messaging/embedder.go` (lazy embedding compute)
- `internal/messaging/search.go` (embedding search via SharedIndex)

**Tasks:**
1. Store embeddings in SQLite (JSON text acceptable for v0.5.12)
2. On `--neural`, compute missing embeddings for candidate set (lazy) and persist
3. Query embeddings: compute query embedding once, call `_sharedindex_find_by_embedding`

### Optional: index rebuild command (1-2 hours)

**Files:**
- `cmd/ailang/messages_index.go`
- `internal/messaging/index.go`

**Tasks:**
- `messages index rebuild --inbox X` to repopulate SharedIndex from SQLite

---

## Migration Strategy

- Add columns with `ALTER TABLE`.
- Backfill simhash for existing rows in a single transaction if feasible.
- Embeddings are backfilled lazily (only when requested with `--neural`).

---

## Testing Strategy

### Unit tests

- simhash computed consistently for message insert/update
- search returns expected results and stable ordering
- dedupe grouping stable and representative selection deterministic
- apply sets `dup_of` and `ack` correctly
- role gating denies non-maintainer apply

### Integration tests

- `messages search` works with and without `--caps SharedIndex`
- `messages search --neural` computes embeddings and persists them
- `messages dedupe` default is report-only
- `messages dedupe --apply` changes state and `--collapsed` hides duplicates

---

## Performance Notes

- SimHash scan on 10K messages is acceptable, especially bounded by `max_scan`.
- SharedIndex accelerator should provide sub-10ms similarity search for typical inbox sizes.
- Embedding computation is the dominant cost; keep it:
  - opt-in (`--neural`)
  - lazy (compute missing only)
  - bounded (don't embed entire archive unless requested)

---

## Success Criteria (v0.5.12)

- [ ] `messages search` returns relevant results by meaning in a shared inbox
- [ ] Works without SharedIndex (SimHash fallback)
- [ ] `--neural` works when SharedIndex is enabled and stores embeddings for reuse
- [ ] Safe dedupe: report-only default; apply is role-gated; reversible via clearing `dup_of`
- [ ] Inbox inference works and is explicitly printed
- [ ] Deterministic ordering and explainability footer present

---

## Local Embedding Configuration (Ollama)

### Config File: `~/.ailang/config.yaml`

```yaml
embeddings:
  # Provider: "ollama" (local) or "none" (SimHash only)
  provider: ollama

  # Ollama-specific settings
  ollama:
    # Model name - gemma2:2b has good quality/speed balance
    model: gemma2:2b

    # Ollama API endpoint (default: localhost)
    endpoint: http://localhost:11434

    # Embedding dimension (auto-detected from model, but can override)
    # gemma2:2b = 2048, nomic-embed-text = 768, mxbai-embed-large = 1024
    dimension: 2048

    # Timeout for embedding requests (default: 30s)
    timeout: 30s

    # Batch size for bulk embedding operations (default: 10)
    batch_size: 10

  # Search behavior
  search:
    # Default search mode: "simhash" or "neural"
    default_mode: simhash

    # Auto-compute embeddings on message insert (default: false for perf)
    auto_embed_on_insert: false

    # Threshold defaults by mode
    simhash_threshold: 0.70
    neural_threshold: 0.75
```

### Environment Variables (override config)

```bash
AILANG_EMBED_PROVIDER=ollama
AILANG_OLLAMA_MODEL=gemma2:2b
AILANG_OLLAMA_ENDPOINT=http://localhost:11434
```

### CLI Flag

```bash
# Use neural search with local Ollama
ailang messages search "parser bugs" --neural

# Force SimHash even if Ollama is configured
ailang messages search "parser bugs" --simhash
```

### Go API

```go
type EmbedConfig struct {
    Provider string `yaml:"provider"` // "ollama" | "none"
    Ollama   struct {
        Model     string        `yaml:"model"`
        Endpoint  string        `yaml:"endpoint"`
        Dimension int           `yaml:"dimension"`
        Timeout   time.Duration `yaml:"timeout"`
        BatchSize int           `yaml:"batch_size"`
    } `yaml:"ollama"`
}

// Embedder interface for pluggable backends
type Embedder interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    Dimension() int
    ModelName() string
}

// OllamaEmbedder implementation
type OllamaEmbedder struct {
    endpoint  string
    model     string
    dimension int
    timeout   time.Duration
    client    *http.Client
}

func NewOllamaEmbedder(cfg EmbedConfig) (*OllamaEmbedder, error)
func (e *OllamaEmbedder) Embed(text string) ([]float32, error)
func (e *OllamaEmbedder) EmbedBatch(texts []string) ([][]float32, error)
```

### Storage

Embeddings stored in `inbox_messages` table:

```sql
-- Already in schema (nullable)
embedding TEXT,           -- JSON array of float32: "[0.123, -0.456, ...]"
embedding_model TEXT,     -- e.g., "ollama:gemma2:2b"
embedding_updated_at INTEGER  -- unix millis
```

### Similarity Computation (Neural)

```go
// Cosine similarity for embeddings
func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

### Lazy Embedding Strategy

To avoid embedding overhead on every insert:

1. **On insert:** Only compute SimHash (fast, always)
2. **On `--neural` search:**
   - Compute query embedding
   - Check if candidate messages have embeddings
   - Lazy-embed missing messages (bounded by `max_scan`)
   - Store embeddings for future reuse
3. **Bulk embed command:**
   ```bash
   ailang messages embed --inbox ailang_core --limit 1000
   ```

### Ollama Model Recommendations

| Model | Dimension | Speed | Quality | Notes |
|-------|-----------|-------|---------|-------|
| `nomic-embed-text` | 768 | ⚡⚡⚡ | Good | Best balance |
| `mxbai-embed-large` | 1024 | ⚡⚡ | Better | Higher quality |
| `gemma2:2b` | 2048 | ⚡ | Best | Highest quality, slowest |

**Recommended default:** `nomic-embed-text` for speed, `gemma2:2b` for quality.

---

## Future Work

1. **SQLite FTS5 keyword fallback** - for exact keyword matching
2. **Search history + autocomplete** - track common queries
3. **"Similar incoming" warnings** - on message insert
4. **Cloud-backed message store** - with same namespace/role semantics
5. **ANN index in BestEffort mode** - for very large archives

---

## Summary

| Phase | Scope | LOC | Time |
|-------|-------|-----|------|
| Phase 1 | SimHash + safe dedupe schema | ~300 | 4h |
| Phase 2 | CLI + inbox inference + role gating | ~200 | 2-3h |
| Phase 3 | Neural search (optional) | ~150 | 3h |
| Optional | Index rebuild command | ~50 | 1-2h |
| **Total** | | **~700** | **10-12h** |

**Recommended approach:** Ship Phase 1+2 in v0.5.12, Phase 3 as opt-in experimental.
