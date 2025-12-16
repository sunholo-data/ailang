# DX-16: SharedIndex — Deterministic Semantic Retrieval for SharedMem

**Status**: Implemented
**Version**: v0.5.11
**Priority**: P0 (High)
**Actual Duration**: ~5 days
**Milestone ID**: M-DX16

**Dependencies** (all satisfied):
- DX-15 (SharedMem + sem_frame MVP) ✅
- Builtin registration pattern (internal/builtins/spec.go) ✅
- TList/TApp type unification fix (DX-17 canonical "list" form) ✅

---

## Executive Summary

**Transform SharedMem from "shared KV store" into "shared cognition".**

This implementation enables agents to retrieve the right shared context by meaning (not hand-rolled key guesses), in a way that is replayable when needed, without cross-domain collisions.

**Delivered:**
1. **Keys only** from similarity search (cheap, bounded, consistent)
2. **Explicit SimHash field** on sem_frame
3. **Determinism modes** (Strict vs BestEffort)
4. **Namespace scoping** (required, prevents collision)
5. **SharedIndex as explicit capability** (separate from SharedMem)
6. **Trace logging** for debugging and replay

---

## Implementation Summary

### Completed Milestones

| Milestone | Description | Status |
|-----------|-------------|--------|
| M1 | sem_frame v2 schema with SimHash | ✅ |
| M2 | Embedding helpers (_simhash builtin) | ✅ |
| M3 | Namespace type (sem_namespace alias) | ✅ |
| M4 | SharedIndex effect declaration | ✅ |
| M5 | SharedIndex Go interface | ✅ |
| M6 | InMemorySharedIndex implementation | ✅ |
| M7 | SharedIndex builtins (5 registered) | ✅ |
| M8 | Determinism modes (Strict/BestEffort) | ✅ |
| M9 | stdlib primitives in std/sem | ✅ |
| M10 | Trace logging infrastructure | ✅ |
| M11 | Working example | ✅ |
| M12 | Tests and polish | ✅ |

---

## Technical Design (As Implemented)

### 1. Namespace Type

```ailang
-- std/sem.ail
type namespace = string  -- Type alias for clarity
```

Namespace is required in all SharedIndex operations. ASCII alphanumeric + hyphen + underscore recommended.

### 2. sem_frame Schema (v2)

```ailang
export type sem_frame = {
  id: string,        -- unique identifier
  content: string,   -- text for SimHash computation
  opaque: string,    -- arbitrary JSON payload
  ver: int,          -- version for CAS
  ts: int,           -- timestamp (Unix millis)
  embedding: string, -- reserved for Tier 2 neural embeddings
  simhash: int       -- cached SimHash of content
}
```

**Invariant:** `simhash == _simhash(content)` must hold.

### 3. SharedIndex Go Interface

```go
// internal/effects/sharedindex.go

type SharedIndex interface {
    Upsert(namespace, key string, simhash, version, timestamp int64)
    Delete(namespace, key string)
    FindSimilarSimHash(
        namespace string,
        simhash int64,
        topK, maxScan int,
        mode DeterminismMode,
    ) []SearchResult
    EntryCount(namespace string) int
    Namespaces() []string
}

type SearchResult struct {
    Key       string
    Score     float64  // 1.0 - (hamming_distance / 64.0)
    Version   int64
    Timestamp int64
}
```

### 4. InMemorySharedIndex Implementation

Thread-safe in-memory implementation using `sync.RWMutex`:
- O(N) scan within namespace, bounded by `maxScan`
- Strict mode: deterministic ordering `(score DESC, key ASC)`
- BestEffort mode: same as Strict for in-memory (placeholder for ANN backends)

### 5. Determinism Modes

```go
type DeterminismMode int

const (
    DeterminismStrict     DeterminismMode = iota  // Replayable given fixed state
    DeterminismBestEffort                          // Faster, approximate
)
```

**Strict semantics:**
- Same query on unchanged index → identical results
- Tie-breaking: lexicographic key order (ASC)
- Scoring: `score = 1.0 - (hamming_distance / 64.0)`

### 6. Registered Builtins

| Builtin | Signature | Effect |
|---------|-----------|--------|
| `_sharedindex_upsert` | `(string, string, int, int, int) -> unit` | SharedIndex |
| `_sharedindex_delete` | `(string, string) -> unit` | SharedIndex |
| `_sharedindex_find_simhash` | `(string, int, int, int, bool) -> list[{key, score, version, timestamp}]` | SharedIndex |
| `_sharedindex_entry_count` | `string -> int` | SharedIndex |
| `_sharedindex_namespaces` | `unit -> list[string]` | SharedIndex |

### 7. Stdlib API (std/sem)

```ailang
-- Store frame with automatic indexing
export func store_frame_ns(ns: namespace, frame: sem_frame) -> unit ! {SharedMem, SharedIndex}

-- Find similar by query text
export func find_similar(ns: namespace, query: string, top_k: int) -> list[search_result] ! {SharedIndex}

-- Find similar with scan limit and determinism control
export func find_similar_bounded(ns: namespace, query: string, top_k: int, max_scan: int, deterministic: bool) -> list[search_result] ! {SharedIndex}

-- Resolve best match and load full frame
export func resolve_best_match(ns: namespace, query: string) -> Option[sem_frame] ! {SharedMem, SharedIndex}

-- Resolve with score threshold
export func resolve_best_match_threshold(ns: namespace, query: string, min_score: float) -> Option[sem_frame] ! {SharedMem, SharedIndex}

-- Delete from both storage and index
export func delete_frame_ns(ns: namespace, key: string) -> unit ! {SharedMem, SharedIndex}

-- Count indexed frames
export func count_frames(ns: namespace) -> int ! {SharedIndex}

-- List all namespaces
export func list_namespaces(u: unit) -> list[string] ! {SharedIndex}
```

### 8. Trace Logging

```go
type TraceEntry struct {
    Operation   string          `json:"op"`
    Namespace   string          `json:"namespace"`
    Key         string          `json:"key,omitempty"`
    QueryHash   int64           `json:"query_hash,omitempty"`
    TopK        int             `json:"top_k,omitempty"`
    MaxScan     int             `json:"max_scan,omitempty"`
    Mode        DeterminismMode `json:"mode,omitempty"`
    ResultCount int             `json:"result_count,omitempty"`
    ChosenKey   string          `json:"chosen_key,omitempty"`
    Timestamp   int64           `json:"ts"`
}
```

Enable via `ctx.SharedIndex.EnableTracing()`, retrieve via `ctx.SharedIndex.GetTrace()`.

---

## Files Created/Modified

### New Files

| File | LOC | Description |
|------|-----|-------------|
| `internal/effects/sharedindex.go` | ~260 | SharedIndex interface + InMemorySharedIndex + trace |
| `internal/effects/sharedindex_test.go` | ~300 | 7 tests (basic ops, similarity, determinism, concurrency, tracing) |
| `internal/builtins/sharedindex.go` | ~415 | 5 SharedIndex builtins with metadata |
| `internal/builtins/sharedindex_test.go` | ~260 | 7 tests for builtin implementations |
| `examples/semantic_retrieval.ail` | ~92 | Working example demonstrating full workflow |

### Modified Files

| File | Changes |
|------|---------|
| `std/sem.ail` | +130 LOC: search_result type, store_frame_ns, find_similar, resolve_best_match, etc. |
| `internal/builtins/simhash.go` | Added _simhash, _hamming_distance builtins |
| `internal/types/type_head.go` | DX-17 fix: canonical lowercase "list" |
| `internal/builtins/array.go` | Updated to use T.List() |
| `internal/builtins/list.go` | Updated to use T.List() |
| `internal/pipeline/op_lowering.go` | Updated for lowercase "list" |

**Total:** ~1,457 new LOC, ~200 modified LOC

---

## Test Coverage

### SharedIndex Effect Tests (7 passing)
- `TestInMemorySharedIndex_BasicOperations`
- `TestInMemorySharedIndex_FindSimilarSimHash`
- `TestInMemorySharedIndex_MaxScan`
- `TestInMemorySharedIndex_DeterminismStrict`
- `TestInMemorySharedIndex_ConcurrentAccess`
- `TestSharedIndexContext`
- `TestSharedIndexContext_Tracing`

### SharedIndex Builtin Tests (7 passing)
- `TestSharedIndexUpsert`
- `TestSharedIndexUpsert_NoCapability`
- `TestSharedIndexDelete`
- `TestSharedIndexFindSimHash`
- `TestSharedIndexFindSimHash_DeterministicOrdering`
- `TestSharedIndexEntryCount`
- `TestSharedIndexNamespaces`

### SimHash Tests (9 passing)
- Determinism, similar texts, different texts, empty string, case insensitivity, punctuation handling, near-duplicate detection

---

## Usage Example

```ailang
module examples/semantic_retrieval

import std/sem (
  sem_frame, search_result,
  store_frame_ns, find_similar, resolve_best_match,
  count_frames
)
import std/string (floatToStr, intToStr)

pure func make_frame(id: string, content: string, ver: int, ts: int) -> sem_frame {
  let hash = _simhash(content)
  in { id: id, content: content, opaque: "{}", ver: ver, ts: ts, embedding: "", simhash: hash }
}

func main() -> string ! {IO, SharedMem, SharedIndex} {
  let _ = _io_println("=== Semantic Retrieval Demo ===")

  -- Store beliefs
  in let belief1 = make_frame("belief_sky", "The sky is blue during the day", 1, 1000)
  in let _ = store_frame_ns("beliefs", belief1)

  -- Query for similar
  in let results = find_similar("beliefs", "What color is the sky?", 3)

  -- Resolve best match
  in match resolve_best_match("beliefs", "Where does the sun come up?") {
    None => _io_println("No match"),
    Some(frame) => _io_println("Best: " ++ frame.content)
  }

  in "Demo complete"
}
```

Run with:
```bash
ailang run --caps IO,SharedMem,SharedIndex --entry main examples/semantic_retrieval.ail
```

---

## Success Criteria (All Met)

- [x] `find_similar` returns (key, score, ver, ts), not full frames
- [x] Strict mode produces identical results on repeated calls (given fixed index state)
- [x] Namespace isolation: frames in "beliefs" not visible in "plans" search
- [x] `--caps SharedIndex` required for index operations
- [x] Trace logs include mode, namespace, top_k, max_scan, result_count
- [x] `chosen_key` only in `resolve_best_match` trace
- [x] `resolve_best_match` works end-to-end
- [x] Round-trip determinism test passes
- [x] All existing tests still pass
- [x] simhash invariant: `simhash == _simhash(content)`

---

## Implementation Notes

### DX-17 Fix (TList/TApp Unification)

During implementation, discovered that `list[T]` types weren't unifying correctly:
- Parser creates `TList{Element: T}` for `[T]` syntax
- Type builder creates `TApp{Constructor: "list", Args: [T]}`
- Fix: Canonical form is lowercase `"list"` everywhere
- `AsList()` helper recognizes both representations

### Pattern Matching on Lists

The TList/TApp fix enabled pattern matching on search results:
```ailang
pure func get_first_result(results: list[search_result]) -> Option[search_result] {
  match results {
    [] => None,
    [first, ..._] => Some(first)
  }
}
```

---

## Future Work (Deferred)

1. **DX-17: Neural Embeddings** - Tier 2 semantic similarity via Ollama
2. **DX-18: Persistent Index Backends** - Redis, Firestore with Strict mode support
3. **DX-19: Index Compaction** - Garbage collection for deleted frames
4. **Bucketed SimHash Index** - Sub-linear search via prefix bucketing

---

## References

- [DX-15: Shared Memory + Near-Duplicate Caching MVP](./DX-15-semantic-caching-MVP.md)
- SimHash paper: Charikar (2002) "Similarity Estimation Techniques from Rounding Algorithms"

---

**Document created**: 2025-12-16
**Implementation completed**: 2025-12-16
**Last updated**: 2025-12-16
