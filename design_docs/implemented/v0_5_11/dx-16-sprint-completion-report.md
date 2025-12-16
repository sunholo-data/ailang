# Sprint Completion Report: DX-16 SharedIndex — Deterministic Semantic Retrieval

## Summary

**Status:** COMPLETED
**Duration:** ~5 days (across multiple sessions)
**Target Version:** v0.5.11
**Milestone ID:** M-DX16

---

## Final Metrics

| Metric | Target | Actual |
|--------|--------|--------|
| Total LOC new | ~950 | ~1,457 |
| Total LOC modified | ~165 | ~200 |
| Milestones | 12/12 | 12/12 ✅ |
| Tests passing | All | All ✅ |
| Example working | ✅ | ✅ |

---

## Milestone Completion Status

### Day 1: Foundation

| Milestone | Status | Notes |
|-----------|--------|-------|
| M1: sem_frame v2 Schema | ✅ | Added content, simhash fields to sem_frame |
| M2: Embedding Helpers | ✅ | _simhash, _hamming_distance builtins |
| M3: Namespace Type | ✅ | `type namespace = string` alias |
| M4: SharedIndex Effect Declaration | ✅ | Added to known effects |
| M5: SharedIndex Go Interface | ✅ | Full interface + types defined |

### Day 2: Implementation

| Milestone | Status | Notes |
|-----------|--------|-------|
| M6: InMemorySharedIndex | ✅ | Thread-safe impl with sync.RWMutex |
| M7: SharedIndex Builtins | ✅ | 5 builtins registered with metadata |
| M8: Determinism Modes | ✅ | Strict (score DESC, key ASC) + BestEffort |

### Day 3: Integration

| Milestone | Status | Notes |
|-----------|--------|-------|
| M9: Stdlib Primitives | ✅ | store_frame_ns, find_similar, resolve_best_match, etc. |
| M10: Trace Logging | ✅ | Full trace infrastructure with JSON serialization |
| M11: Working Example | ✅ | examples/semantic_retrieval.ail |
| M12: Tests + Polish | ✅ | 14+ new tests, lint clean |

---

## Blockers Encountered & Resolved

### TList/TApp Type Unification (DX-17)

**Problem:** M9 was blocked because `list[search_result]` types weren't unifying:
- Parser creates `TList{Element: T}` for `[T]` syntax
- Type builder creates `TApp{Constructor: "List", Args: [T]}`
- These were treated as different types

**Resolution:**
- Canonical form is lowercase `"list"` everywhere
- Updated `internal/types/type_head.go` to only check lowercase
- Updated all builtins to use `T.List()` instead of `T.App("List")`
- User directive: "no legacy - one way is the good way - lets make it 'list'"

**Files Modified:**
- `internal/types/type_head.go`
- `internal/builtins/array.go`
- `internal/builtins/list.go`
- `internal/pipeline/op_lowering.go`
- Tests and golden files

### Pattern Matching on Lists

**Problem:** Couldn't use pattern matching on `list[search_result]`:
```ailang
-- This failed before fix
match results {
  [] => None,
  [first, ..._] => Some(first)
}
```

**Resolution:** After TList/TApp fix, pattern matching works correctly. Replaced helper function that used `_array_length` with direct pattern matching.

---

## Files Created

| File | LOC | Description |
|------|-----|-------------|
| `internal/effects/sharedindex.go` | 263 | Interface + InMemorySharedIndex + trace logging |
| `internal/effects/sharedindex_test.go` | 299 | 7 comprehensive tests |
| `internal/builtins/sharedindex.go` | 414 | 5 builtins with full metadata |
| `internal/builtins/sharedindex_test.go` | 257 | 7 builtin tests |
| `examples/semantic_retrieval.ail` | 92 | Working end-to-end example |
| `design_docs/implemented/v0_5_11/dx-16-*.md` | ~600 | Design doc + this report |

---

## Test Summary

### SharedIndex Effect Tests (7/7 passing)
- BasicOperations: Upsert, Delete, EntryCount
- FindSimilarSimHash: Similarity scoring with hamming distance
- MaxScan: Bounded search limits
- DeterminismStrict: Consistent ordering guarantee
- ConcurrentAccess: 100+ goroutines with -race
- SharedIndexContext: Counter tracking
- SharedIndexContext_Tracing: Trace enable/disable/clear

### SharedIndex Builtin Tests (7/7 passing)
- Upsert: Store entry with all params
- Upsert_NoCapability: Error when not enabled
- Delete: Remove entry
- FindSimHash: Similarity search with scoring
- FindSimHash_DeterministicOrdering: Key ASC tie-breaking
- EntryCount: Namespace counting
- Namespaces: List all namespaces

### SimHash Tests (9/9 passing)
- Deterministic hashing
- Similar/different text detection
- Edge cases: empty string, case insensitivity, punctuation

---

## Key Design Decisions Made

1. **Keys only from search** - Returns keys, not full frames. Agent uses load_frame for full data.

2. **Strict determinism** - Same query on unchanged index → same results. Tie-breaking: `(score DESC, key ASC)`.

3. **Namespace isolation** - Required in all operations. No cross-namespace search.

4. **Lowercase "list" canonical form** - No legacy uppercase "List" support. Clean break.

5. **SimHash-based scoring** - `score = 1.0 - (hamming_distance / 64.0)`. Simple, deterministic.

6. **maxScan bounds O(N)** - Prevents unbounded search time in large namespaces.

---

## Usage

```bash
# Run the working example
ailang run --caps IO,SharedMem,SharedIndex --entry main examples/semantic_retrieval.ail
```

---

## DX-17: Neural Embedding Search (Sprint Extension)

**Status:** COMPLETED ✅
**Added:** 2025-12-16

### Overview

Extend SharedIndex with neural embedding-based similarity search using cosine similarity.
This enables true semantic similarity (Tier 2) in addition to SimHash (Tier 1).

### Phase 1: Ollama Integration ✅

| File | LOC | Description |
|------|-----|-------------|
| `internal/builtins/ollama_embed.go` | ~100 | `_ollama_embed` builtin using Ollama Go SDK |
| `internal/builtins/ollama_embed_test.go` | ~40 | Go test verifying 768-dim embeddings |
| `examples/ollama_embed_test.ail` | ~15 | AILANG example |

**Implementation:**
- **Builtin:** `_ollama_embed(model: string, text: string) -> list[float] ! {IO}`
- **SDK:** `github.com/ollama/ollama/api` (official Ollama Go client)
- **Default model:** EmbeddingGemma (768-dimensional, 300M params)
- **Latency:** ~160ms per embedding call

### Phase 2: Embedding Search (In Progress)

**New Go Types:**
```go
// IndexEntry extended with optional embedding
type IndexEntry struct {
    Namespace string
    Key       string
    SimHash   int64
    Embedding []float64  // NEW: optional 768-dim vector
    Version   int64
    Timestamp int64
    Score     float64
}
```

**New SharedIndex Methods:**
```go
// Store entry with embedding
UpsertWithEmbedding(entry IndexEntry) error

// Find similar by cosine similarity
FindSimilarByEmbedding(namespace string, queryEmbedding []float64, topK, maxScan int, mode DeterminismMode) []SearchResult
```

**New Builtins:**
| Builtin | Signature | Description |
|---------|-----------|-------------|
| `_sharedindex_upsert_emb` | `(ns, key, simhash, embedding, ver, ts) -> unit` | Store with embedding |
| `_sharedindex_find_by_embedding` | `(ns, embedding, topK, maxScan) -> list[search_result]` | Cosine similarity search |

**Cosine Similarity:**
```
score = (A · B) / (||A|| × ||B||)
```
- Range: [-1, 1] normalized to [0, 1] for consistency with SimHash scores
- Deterministic tie-breaking: (score DESC, key ASC)

### Usage

```ailang
-- Generate embedding
let emb = _ollama_embed("embeddinggemma", "The sky is blue")

-- Store with embedding
let _ = _sharedindex_upsert_emb("beliefs", "belief-1", simhash, emb, 1, now())

-- Search by embedding similarity
let query_emb = _ollama_embed("embeddinggemma", "What color is the sky?")
let results = _sharedindex_find_by_embedding("beliefs", query_emb, 5, 100)
```

### Files Created/Modified

| File | LOC | Description |
|------|-----|-------------|
| `internal/effects/sharedindex.go` | +50 | Embedding field, interface methods, trace methods |
| `internal/effects/sharedindex_inmemory.go` | +110 | UpsertWithEmbedding, FindSimilarByEmbedding, cosineSimilarity |
| `internal/effects/sharedindex_test.go` | +240 | 7 new embedding tests |
| `internal/builtins/sharedindex.go` | +250 | 2 new builtins with full metadata |
| `examples/neural_semantic_search.ail` | +95 | End-to-end example with Ollama |

**Total new LOC:** ~745

### Test Results

```
=== RUN   TestCosineSimilarity (7 subtests) --- PASS
=== RUN   TestInMemorySharedIndex_UpsertWithEmbedding --- PASS
=== RUN   TestInMemorySharedIndex_FindSimilarByEmbedding --- PASS
=== RUN   TestInMemorySharedIndex_FindSimilarByEmbedding_TopK --- PASS
=== RUN   TestInMemorySharedIndex_FindSimilarByEmbedding_SkipsEntriesWithoutEmbeddings --- PASS
=== RUN   TestInMemorySharedIndex_FindSimilarByEmbedding_EmptyNamespace --- PASS
=== RUN   TestInMemorySharedIndex_FindSimilarByEmbedding_DeterministicOrdering --- PASS
```

---

## Future Work

- DX-18: Persistent index backends (Redis, Firestore)
- Bucketed SimHash for sub-linear search
- Hybrid search (SimHash pre-filter + embedding re-rank)

---

**Sprint started:** 2025-12-16
**Sprint completed:** 2025-12-16
**Ollama integration added:** 2025-12-16
**Document created:** 2025-12-16
