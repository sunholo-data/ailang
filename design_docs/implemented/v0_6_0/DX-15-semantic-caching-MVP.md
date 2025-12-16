# DX-15: Shared Memory + Near-Duplicate Caching MVP

**Status:** IMPLEMENTED
**Version:** v0.5.11
**Completed:** 2025-12-16
**Duration:** 3 days (vs 14 estimated)

---

## Implementation Summary

### What Was Built

| Feature | Status | Key Files |
|---------|--------|-----------|
| bytes builtins (5 functions) | ✅ | `internal/builtins/bytes.go` |
| SimHash + hamming_distance | ✅ | `internal/builtins/simhash.go` |
| sem_frame type | ✅ | `std/sem.ail` |
| JSON encode/decode for frames | ✅ | `std/sem.ail` (encode_frame, decode_frame) |
| SharedMem effect | ✅ | `internal/effects/sharedmem.go` |
| InMemorySharedCache | ✅ | `internal/effects/sharedmem.go` |
| Thread-safety stress tests | ✅ | `internal/effects/sharedmem_test.go` (12 tests) |
| SharedMem builtins | ✅ | `internal/builtins/sharedmem.go` (5 builtins) |
| Frame helpers | ✅ | `std/sem.ail` (load_frame, store_frame, update_frame) |
| find_similar backend | ⏸️ | Stretch goal - using `is_similar()` pairwise |
| Working example | ✅ | `examples/sharedmem_cache.ail` |

### Metrics

| Metric | Estimated | Actual |
|--------|-----------|--------|
| Total LOC | 1,800 | ~1,100 |
| Duration | 14 days | 3 days |
| Test coverage | 90%+ | 21 tests pass |
| Milestones | 14 | 13/14 (93%) |

### Key Deviations from Plan

1. **find_similar as builtin deferred** - Using `is_similar(frame1, frame2, threshold)` for pairwise comparison instead of backend scan. This is sufficient for MVP use cases.

2. **No meta field in sem_frame** - Removed map[string, string] metadata field to simplify JSON encoding. Can be added in v0.5.12 if needed.

3. **M-DX10 workaround for keys()** - `_sharedmem_keys` takes unit parameter due to nullary function bug.

4. **Transitive import requirement** - Example file must import `std/json` and `std/result` because transitive dependencies aren't auto-loaded at runtime.

### Files Created/Modified

**New Files:**
- `internal/effects/sharedmem.go` - SharedCache interface + InMemorySharedCache (202 LOC)
- `internal/effects/sharedmem_test.go` - Thread-safety stress tests (394 LOC)
- `internal/builtins/sharedmem.go` - 5 SharedMem builtins (360 LOC)
- `internal/builtins/sharedmem_test.go` - 9 builtin tests (300 LOC)
- `internal/builtins/bytes.go` - bytes conversion builtins (previously created)
- `internal/builtins/simhash.go` - SimHash + hamming distance (previously created)
- `std/sharedmem.ail` - AILANG wrapper module (63 LOC)
- `examples/sharedmem_cache.ail` - Working demo (55 LOC)

**Modified Files:**
- `internal/effects/context.go` - Added SharedMem field
- `internal/parser/parser_effect.go` - Added SharedMem to known effects
- `internal/types/builder.go` - Changed List to lowercase `list`
- `cmd/ailang/run_helpers.go` - Added setupSharedMemHandler
- `std/sem.ail` - Added frame helpers (load_frame, store_frame, update_frame)

### Usage Example

```bash
./bin/ailang run --caps IO,SharedMem --entry main examples/sharedmem_cache.ail
```

Output:
```
=== SharedMem Cache Demo ===
1. Basic cache operations:
   Stored and retrieved: Hello, World!
2. Semantic frame creation:
   Stored frame with simhash: -3839431810354909714
3. Near-duplicate detection:
   Frame1 simhash: -3839431810354909714
   Frame2 simhash: 5095709833168284012
   Are similar (threshold 10): true
4. Load frame from cache:
   Loaded: id=doc-001, ver=1
```

---

## Original Sprint Plan

*(Preserved below for reference)*

---

# Sprint Plan: DX-15 Shared Memory + Near-Duplicate Caching MVP

## Summary

**MVP scope** for shared-memory caching: SharedMem effect with in-memory backend, sem_frame canonical type with **tiered similarity**:
- **Tier 1 (MVP):** SimHash fingerprints - deterministic near-duplicate prefilter (always-on, zero dependencies)
- **Tier 2 (Optional):** Neural embeddings via Ollama - true semantic similarity

**Duration:** 2 sprints (14 days)
**Target Version:** v0.5.11
**Risk Level:** Medium (reordered to reduce risk)

### What's IN MVP
- SharedMem effect (get/put/cas/delete)
- In-memory backend (thread-safe, copy-on-read/write)
- sem_frame record type with **SimHash fingerprint** (always computed)
- load_frame, store_frame, update_frame helpers
- SimHash-based **near-duplicate detection** (structural, not semantic)
- find_similar as backend capability (not effect-level enumeration)

### What's DEFERRED to v0.5.12+
- Redis backend
- Firestore backend
- Multi-agent demo
- AI.embed / neural embeddings (Tier 2)
- In-process embedder (llama.cpp/MLX)

---

## Tiered Similarity Architecture

### The Problem
Full neural embeddings (768-dim vectors) require:
- External dependency (Ollama)
- Model download (~800MB)
- GPU/CPU inference time
- 3KB storage per frame

### The Solution: Two Tiers

| Tier | Method | Storage | Dependencies | Use Case |
|------|--------|---------|--------------|----------|
| **Tier 1** | SimHash | 8 bytes | None (pure Go) | **Near-duplicate** detection (structural) |
| **Tier 2** | Neural embedding | varies | Ollama | **Semantic** similarity (paraphrases) |

**SimHash** is always computed. **Neural embeddings** are optional (deferred to v0.5.12).

### Important: Tier 1 is NOT Semantic

SimHash catches **structural similarity** (same/similar words), NOT **semantic similarity** (same meaning, different words):

| Query | Match? | Why |
|-------|--------|-----|
| "quick brown fox" vs "quick brown dog" | ✅ Yes | Same structure, 1 word different |
| "car" vs "automobile" | ❌ No | Different words, same meaning |
| "The meeting is at 3pm" vs "We meet at 15:00" | ❌ No | Paraphrase |

**This is intentional.** Tier 1 is a **prefilter** for near-duplicates. True semantic matching requires Tier 2 embeddings.

---

## Key Design Decisions (as implemented)

### 1. SimHash Type: Explicit `int`
AILANG `int` maps to Go `int64`. Implemented as documented.

### 2. Content vs Opaque Separation
sem_frame has explicit `content` field for SimHash, `opaque` for arbitrary binary payload.

### 3. SharedMem API: No Enumeration
SharedMem is pure KV (get/put/cas/delete). Similarity search uses `is_similar()` for pairwise comparison.

### 4. CAS Semantics: Nil = Create-if-absent
`cas(key, None, new_value)` creates if absent, `cas(key, Some(old), new)` updates if matches.

### 5. Thread Safety: Copy-on-Read/Write
All cache operations copy bytes to prevent buffer sharing bugs. Verified with 100+ goroutine stress tests.

---

## Post-MVP (v0.5.12+)

Deferred to future sprints:
1. **find_similar backend scan** - Backend capability for similarity search
2. **Redis backend** - Production-ready persistent cache
3. **Firestore backend** - Cloud-native option
4. **AI.embed / Neural embeddings** - Tier 2 semantic similarity
5. **meta field** - Key-value metadata in sem_frame

---

**Original Plan Created:** 2025-12-14
**Implementation Completed:** 2025-12-16
**Target Version:** v0.5.11
