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

### How SimHash Works

```
"hello world" → tokenize → ["hello", "world"]
                         → hash each → [0xABC..., 0xDEF...]
                         → aggregate → 64-bit fingerprint
```

- **Deterministic**: Same text always produces same hash
- **Locality-sensitive**: Similar texts have similar hashes
- **Comparison**: Hamming distance (count differing bits)
  - Distance 0 = identical
  - Distance 1-3 = very similar (likely duplicates)
  - Distance 4-10 = somewhat similar
  - Distance 10+ = different

**Note:** Threshold depends on tokenization and text length. `max_dist=3` is a reasonable default but should be tunable.

### Lookup Strategy

```
1. Exact key match?           → Cache hit (O(1))
2. SimHash distance ≤ N?      → Likely duplicate, verify content
3. Embedding similarity > 0.9? → Semantic match (Tier 2 only)
```

This gives **zero-cost near-duplicate detection** without any ML infrastructure.

---

## Sprint Overview (Reordered for Lower Risk)

| Sprint | Focus | LOC Est | Days |
|--------|-------|---------|------|
| S1 | sem_frame Type + bytes + SimHash | ~700 | 6 |
| S2 | SharedMem Effect + In-Memory + find_similar | ~1,100 | 8 |
| **Total** | | **~1,800** | **14** |

**Why this order?**
- sem_frame is pure data (LOW RISK) - validates design before committing to effect
- SharedMem builds on proven sem_frame patterns
- Thread-safety testing has more time allocation

---

## Key Design Decisions

### 1. SimHash Type: Explicit `int64`

AILANG `int` maps to Go `int64`. For cross-platform determinism and JSON round-tripping:

```ailang
type simhash64 = int  -- Documented: must fit signed 64-bit
```

**Rationale:** Avoids platform-dependent int sizes, ensures hamming distance works correctly, prevents float conversion in JSON tooling.

### 2. SimHash Input: Separate `content` Field (Not Raw Bytes)

**Problem:** `bytes_to_string(opaque)` fails for non-UTF8 binary data (gzip, protobuf, etc.)

**Solution:** sem_frame has explicit `content` field for similarity:

```ailang
type sem_frame = {
  id: string,
  ver: int,
  ts: int,
  content: string,                 -- UTF-8 text for SimHash (required)
  simhash: int,                    -- Computed from content
  embedding: option[bytes],        -- Packed float32 (Tier 2)
  embedding_dim: int,              -- 0 if no embedding
  meta: map[string, string],
  opaque: bytes                    -- Arbitrary payload (not hashed)
}
```

**Invariant:** `simhash = simhash(content)`. The `opaque` field is payload-only, never hashed.

### 3. Embedding Storage: Packed Bytes, Not `list[float]`

**Problem:** `list[float]` is allocation-heavy and awkward in codegen.

**Solution:** Store as packed `bytes` with dimension metadata:

```ailang
embedding: option[bytes],   -- Packed float32 (768 floats = 3072 bytes)
embedding_dim: int,         -- 768 for EmbeddingGemma, 0 if none
```

**Benefits:**
- Efficient storage and Go codegen (`[]byte` direct)
- Clean JSON (base64 blob)
- Easy cosine similarity in Go (`unsafe.Slice` to `[]float32`)

### 4. SharedMem API: No Enumeration at Effect Level

**Problem:** `find_similar` needs to iterate the cache, but adding `scan`/`keys` to SharedMem creates perf pitfalls on remote backends.

**Solution (Option B):** Keep SharedMem as pure KV, similarity search is a **backend capability**:

```go
// Effect-level API (language visible)
type SharedMemOps interface {
    Get(key string) ([]byte, bool)
    Put(key string, value []byte)
    CAS(key string, expected, newValue []byte) bool
    Delete(key string)
}

// Backend capability (implementation detail)
type SimilaritySearcher interface {
    FindSimilar(hash int64, maxDist int) []SimilarityMatch
}
```

**In stdlib:** `find_similar` checks if backend implements `SimilaritySearcher`, errors cleanly if not.

**Why:** Avoids baking enumeration into effect semantics. Redis/Firestore can implement or skip similarity search independently.

### 5. CAS Semantics: Explicit Nil Handling

```ailang
-- Create if absent (expected = None)
cas(key, None, new_value)     -- Returns true if key didn't exist

-- Update if matches (expected = Some(old))
cas(key, Some(old), new_value) -- Returns true only if current == old
```

**At Go level:**
```go
func (c *Cache) CAS(key string, expected, newValue []byte) bool {
    // expected == nil means "create if absent"
    // expected != nil means "swap only if exact byte match"
}
```

### 6. Clock Dependency: Pure Constructor Variant

**Problem:** `make_frame` uses `Clock.now()`, forcing Clock effect in pure test code.

**Solution:** Two constructors:

```ailang
-- Requires Clock effect
func make_frame(id: string, content: string, opaque: bytes) -> sem_frame ! {Clock}

-- Pure (for tests, deterministic replay)
func make_frame_at(id: string, content: string, opaque: bytes, ts: int) -> sem_frame
```

### 7. Thread Safety: Copy-on-Read/Write

**Critical invariant:** No shared backing arrays between cache and callers.

```go
func (c *Cache) Put(key string, value []byte) {
    c.mu.Lock()
    defer c.mu.Unlock()
    // COPY value - caller may mutate their slice
    c.store[key] = append([]byte(nil), value...)
}

func (c *Cache) Get(key string) ([]byte, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    v, ok := c.store[key]
    if !ok {
        return nil, false
    }
    // COPY on read - caller can't corrupt cache
    return append([]byte(nil), v...), true
}

func (c *Cache) CAS(key string, expected, newValue []byte) bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    current := c.store[key]
    // Compare copies, store copy
    if bytes.Equal(current, expected) {
        c.store[key] = append([]byte(nil), newValue...)
        return true
    }
    return false
}
```

---

## Sprint 1: sem_frame Type + bytes + SimHash

**Goal:** Define canonical cache entry type, bytes utilities, and SimHash fingerprinting. No effects yet - pure data and functions.

**Estimated:** 500 LOC implementation + 200 LOC tests = ~700 LOC
**Duration:** 6 days

### Day 1: bytes Builtins

**Tasks:**
- [ ] Verify TBytes type works in user code
- [ ] Add bytes conversion builtins to registry
- [ ] Implement string↔bytes conversions (UTF-8)
- [ ] Implement bytes↔base64 conversions

**Builtins to add:**
```go
// internal/builtins/bytes.go
"bytes_from_string": func(s string) []byte      // UTF-8 encode
"bytes_to_string":   func(b []byte) string      // UTF-8 decode
"bytes_to_base64":   func(b []byte) string      // Base64 encode
"bytes_from_base64": func(s string) []byte      // Base64 decode
"bytes_length":      func(b []byte) int         // Length in bytes
```

**Files:**
- `internal/builtins/bytes.go` (new, ~100 LOC)
- `internal/builtins/bytes_test.go` (new, ~80 LOC)

**Acceptance Criteria:**
- [ ] `bytes_from_string("hello")` returns 5-byte slice
- [ ] Round-trip: `bytes_to_string(bytes_from_string(s)) == s`
- [ ] Base64 encoding matches standard Go encoding

### Day 2: sem_frame Type Definition

**Tasks:**
- [ ] Define sem_frame record type in stdlib
- [ ] Define sem_key type alias (string)
- [ ] Define update_result ADT
- [ ] Test type parsing and inference

**Type definitions:**
```ailang
-- stdlib/shared/sem.ail
module shared/sem

type sem_key = string
type simhash64 = int   -- Documented: signed 64-bit for cross-platform determinism

type sem_frame = {
  id: string,
  ver: int,
  ts: int,
  content: string,              -- UTF-8 text for SimHash (required, never empty)
  simhash: simhash64,           -- Computed from content, always present
  embedding: option[bytes],     -- Packed float32 (Tier 2, None for MVP)
  embedding_dim: int,           -- 768 for EmbeddingGemma, 0 if no embedding
  meta: map[string, string],
  opaque: bytes                 -- Arbitrary payload (not hashed)
}

type update_result[T] =
  | Missing
  | Updated(T)
  | Conflict(T)

-- Similarity result for find_similar queries
type similarity_match = {
  key: sem_key,
  frame: sem_frame,
  distance: int        -- Hamming distance (0 = identical)
}
```

**Key invariants:**
- `simhash = simhash(content)` - always computed from content, never from opaque
- `content` is UTF-8 normalized text (caller responsibility)
- `opaque` can be any binary data (gzip, protobuf, etc.)

**Files:**
- `stdlib/shared/sem.ail` (new, ~50 LOC)

**Acceptance Criteria:**
- [ ] `import shared/sem` works
- [ ] sem_frame type-checks correctly
- [ ] Can construct: `{ id = "x", ver = 1, ts = 0, embedding = None, meta = {}, opaque = bytes_from_string("") }`

### Day 3: sem_frame JSON Encoding

**Tasks:**
- [ ] Implement sem_frame → JSON encoding
- [ ] Handle option[embedding] (null if None)
- [ ] Handle bytes as base64
- [ ] Implement JSON → sem_frame decoding

**Files:**
- `stdlib/shared/sem_json.ail` (new, ~100 LOC)
- Test file with round-trip tests

**Acceptance Criteria:**
- [ ] `encode_frame(f)` produces valid JSON string
- [ ] `decode_frame(json)` returns `option[sem_frame]`
- [ ] Round-trip preserves all fields exactly
- [ ] bytes field serializes as base64 string

### Day 4: SimHash Implementation

**Tasks:**
- [ ] Implement SimHash algorithm in Go
- [ ] Add `simhash` builtin function
- [ ] Add `hamming_distance` builtin function
- [ ] Write unit tests with known test vectors

**SimHash Implementation (pure Go):**
```go
// internal/builtins/simhash.go
func SimHash(text string) int64 {
    tokens := tokenize(text)  // Split on whitespace/punctuation
    var v [64]int

    for _, token := range tokens {
        h := fnv.New64a()
        h.Write([]byte(strings.ToLower(token)))
        hash := h.Sum64()

        for i := 0; i < 64; i++ {
            if (hash>>i)&1 == 1 {
                v[i]++
            } else {
                v[i]--
            }
        }
    }

    var fingerprint uint64
    for i := 0; i < 64; i++ {
        if v[i] > 0 {
            fingerprint |= 1 << i
        }
    }
    return int64(fingerprint)
}

func HammingDistance(a, b int64) int {
    return bits.OnesCount64(uint64(a ^ b))
}
```

**Builtins:**
```go
"simhash":          func(text: string) -> int
"hamming_distance": func(a: int, b: int) -> int
```

**Files:**
- `internal/builtins/simhash.go` (new, ~80 LOC)
- `internal/builtins/simhash_test.go` (new, ~100 LOC)

**Acceptance Criteria:**
- [ ] `simhash("hello world")` returns consistent 64-bit int
- [ ] `simhash("hello world") == simhash("hello world")` (deterministic)
- [ ] `hamming_distance(simhash("hello"), simhash("helo"))` is small (1-3)
- [ ] `hamming_distance(simhash("hello"), simhash("goodbye"))` is large (15+)

### Day 5: Frame Constructors + Tests

**Tasks:**
- [ ] Add `make_frame(id, content, opaque)` constructor with Clock effect
- [ ] Add `make_frame_at(id, content, opaque, ts)` pure variant for tests
- [ ] Add `make_frame_with_meta(...)` variants
- [ ] SimHash computed automatically from content field
- [ ] Write comprehensive unit tests
- [ ] Create example file

**Frame constructor logic:**
```ailang
-- With Clock effect (production use)
func make_frame(id: string, content: string, opaque: bytes) -> sem_frame ! {Clock} =
  {
    id = id,
    ver = 1,
    ts = Clock.now(),
    content = content,
    simhash = simhash(content),
    embedding = None,
    embedding_dim = 0,
    meta = {},
    opaque = opaque
  }

-- Pure variant (tests, deterministic replay)
func make_frame_at(id: string, content: string, opaque: bytes, ts: int) -> sem_frame =
  {
    id = id,
    ver = 1,
    ts = ts,
    content = content,
    simhash = simhash(content),
    embedding = None,
    embedding_dim = 0,
    meta = {},
    opaque = opaque
  }
```

**Files:**
- `stdlib/shared/sem.ail` (extend, ~60 LOC)
- `stdlib/shared/sem_test.ail` (new, ~120 LOC)
- `examples/sem_frame_basic.ail` (new, ~50 LOC)

**Acceptance Criteria:**
- [ ] `make_frame("key", "content text", payload)` creates valid frame
- [ ] `make_frame_at(...)` works without Clock effect
- [ ] SimHash computed from content, not opaque
- [ ] Tests verify simhash consistency
- [ ] Example demonstrates near-duplicate detection

### Day 6: Buffer / Polish

**Tasks:**
- [ ] Address any issues from Days 1-5
- [ ] Ensure 90%+ test coverage
- [ ] Update CLAUDE.md with bytes and simhash builtins
- [ ] Document sem_frame in stdlib README

---

## Sprint 2: SharedMem Effect + In-Memory Backend + find_similar

**Goal:** Implement SharedMem effect with thread-safe in-memory storage and near-duplicate search.

**Estimated:** 800 LOC implementation + 300 LOC tests = ~1,100 LOC
**Duration:** 8 days

### Day 1: SharedMem Effect Declaration

**Tasks:**
- [ ] Add SharedMem to effect registry
- [ ] Add SharedMem capability
- [ ] Define effect operations: get, put, cas, delete
- [ ] Wire to capability system

**Files:**
- `internal/effects/shared_mem.go` (new, ~100 LOC)
- `internal/effects/capability.go` (extend, ~20 LOC)

**Acceptance Criteria:**
- [ ] `SharedMem` appears in `--caps` list
- [ ] Effect can be declared: `func f() -> T ! {SharedMem}`
- [ ] Parser accepts SharedMem in effect row

### Day 2: SharedCache Interface

**Tasks:**
- [ ] Define SharedCache Go interface
- [ ] Design CAS semantics (compare-and-swap)
- [ ] Add EffContext.SharedCache field
- [ ] Wire to pipeline initialization

**Interface:**
```go
// internal/effects/shared_cache.go
type SharedCache interface {
    Get(key string) ([]byte, bool)
    Put(key string, value []byte)
    CAS(key string, expected, newValue []byte) bool  // Returns true if swap succeeded
    Delete(key string)
}
```

**Files:**
- `internal/effects/shared_cache.go` (new, ~80 LOC)
- `internal/effects/context.go` (extend, ~30 LOC)

**Acceptance Criteria:**
- [ ] Interface defined with clear semantics
- [ ] EffContext has SharedCache field
- [ ] Nil SharedCache returns clear error

### Day 3: InMemorySharedCache Implementation

**Tasks:**
- [ ] Implement thread-safe in-memory cache
- [ ] Use sync.RWMutex for thread safety
- [ ] Implement atomic CAS logic
- [ ] Add basic unit tests

**Files:**
- `internal/effects/shared_cache_memory.go` (new, ~150 LOC)
- `internal/effects/shared_cache_memory_test.go` (new, ~100 LOC)

**Implementation:**
```go
type InMemorySharedCache struct {
    mu    sync.RWMutex
    store map[string][]byte
}

func (c *InMemorySharedCache) CAS(key string, expected, newValue []byte) bool {
    c.mu.Lock()
    defer c.mu.Unlock()

    current, exists := c.store[key]
    if !exists && expected == nil {
        c.store[key] = newValue
        return true
    }
    if exists && bytes.Equal(current, expected) {
        c.store[key] = newValue
        return true
    }
    return false
}
```

**Acceptance Criteria:**
- [ ] Get/Put work correctly
- [ ] CAS returns true only on exact match
- [ ] Thread-safe under concurrent access

### Day 4: Thread-Safety Stress Tests

**Tasks:**
- [ ] Write concurrent access tests (100+ goroutines)
- [ ] Test CAS under contention
- [ ] Run with `-race` flag
- [ ] Fix any race conditions found

**Files:**
- `internal/effects/shared_cache_stress_test.go` (new, ~150 LOC)

**Test scenarios:**
```go
func TestCASContention(t *testing.T) {
    cache := NewInMemorySharedCache()
    cache.Put("counter", []byte("0"))

    var wg sync.WaitGroup
    successes := atomic.Int32{}

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                old, _ := cache.Get("counter")
                newVal := incrementBytes(old)
                if cache.CAS("counter", old, newVal) {
                    successes.Add(1)
                }
            }
        }()
    }
    wg.Wait()
    // Exactly successes.Load() increments should have happened
}
```

**Acceptance Criteria:**
- [ ] All tests pass with `-race`
- [ ] No deadlocks under stress
- [ ] CAS correctness verified

### Day 5: SharedMem Builtins + Handlers

**Tasks:**
- [ ] Register SharedMem builtins (get, put, cas, delete)
- [ ] Implement effect handlers
- [ ] Wire through evaluator
- [ ] Add type signatures

**Builtins:**
```go
// internal/builtins/shared_mem.go
"SharedMem.get":    func(key: string) -> option[bytes] ! {SharedMem}
"SharedMem.put":    func(key: string, value: bytes) -> unit ! {SharedMem}
"SharedMem.cas":    func(key: string, expected: bytes, new: bytes) -> bool ! {SharedMem}
"SharedMem.delete": func(key: string) -> unit ! {SharedMem}
```

**Files:**
- `internal/builtins/shared_mem.go` (new, ~120 LOC)
- `internal/effects/shared_mem_handlers.go` (new, ~100 LOC)
- `internal/eval/eval.go` (extend, ~30 LOC)

**Acceptance Criteria:**
- [ ] `SharedMem.get("key")` returns `option[bytes]`
- [ ] `SharedMem.put("key", data)` stores data
- [ ] `SharedMem.cas("key", old, new)` returns bool
- [ ] Effect requires `--caps SharedMem`

### Day 6: High-Level Helpers (load_frame, store_frame, update_frame)

**Tasks:**
- [ ] Implement load_frame using SharedMem.get + JSON decode
- [ ] Implement store_frame using SharedMem.put + JSON encode
- [ ] Implement update_frame with CAS retry loop
- [ ] Add bounded retries (max 8-10)

**Files:**
- `stdlib/shared/sem_io.ail` (new, ~120 LOC)

**Implementation:**
```ailang
func load_frame(key: sem_key) -> option[sem_frame] ! {SharedMem} =
  match SharedMem.get(key) with
  | None -> None
  | Some(data) -> decode_frame(bytes_to_string(data))

func store_frame(key: sem_key, frame: sem_frame) -> unit ! {SharedMem} =
  let json = encode_frame(frame) in
  SharedMem.put(key, bytes_from_string(json))

func update_frame(key: sem_key, f: func(sem_frame) -> sem_frame)
    -> update_result[sem_frame] ! {SharedMem} =
  update_frame_loop(key, f, 0, 10)  -- max 10 retries
```

**Acceptance Criteria:**
- [ ] load_frame returns None for missing keys
- [ ] store_frame persists correctly
- [ ] update_frame handles CAS contention
- [ ] Returns Conflict after max retries

### Day 7: find_similar + Near-Duplicate Detection

**Tasks:**
- [ ] Implement `find_similar(simhash, max_distance)` helper
- [ ] Scan cache for frames within hamming distance
- [ ] Return list of matches sorted by distance
- [ ] Add index for efficient lookup (optional optimization)

**Implementation:**
```ailang
-- Find frames with similar content (using SimHash)
func find_similar(hash: int, max_dist: int) -> list[similarity_match] ! {SharedMem} =
  -- Scans all cached frames, returns those within max_dist
  -- For MVP: linear scan is fine (optimize with LSH index later)
  SharedMem.scan_similar(hash, max_dist)
```

**Go-side implementation:**
```go
// internal/effects/shared_mem_handlers.go
func (c *InMemorySharedCache) FindSimilar(hash int64, maxDist int) []SimilarityMatch {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var matches []SimilarityMatch
    for key, data := range c.store {
        frame := decodeFrame(data)
        dist := bits.OnesCount64(uint64(hash ^ frame.SimHash))
        if dist <= maxDist {
            matches = append(matches, SimilarityMatch{
                Key:      key,
                Frame:    frame,
                Distance: dist,
            })
        }
    }
    sort.Slice(matches, func(i, j int) bool {
        return matches[i].Distance < matches[j].Distance
    })
    return matches
}
```

**Files:**
- `stdlib/shared/sem_search.ail` (new, ~60 LOC)
- `internal/effects/shared_mem_handlers.go` (extend, ~50 LOC)

**Acceptance Criteria:**
- [ ] `find_similar(hash, 3)` returns near-duplicates
- [ ] Results sorted by distance (closest first)
- [ ] Empty list if no matches

### Day 8: Integration Tests + Example

**Tasks:**
- [ ] Write end-to-end integration tests
- [ ] Create working example with duplicate detection
- [ ] Test with `ailang run --caps SharedMem`
- [ ] Document usage in CLAUDE.md

**Files:**
- `internal/effects/shared_mem_integration_test.go` (new, ~150 LOC)
- `examples/shared_mem_basic.ail` (new, ~80 LOC)
- `stdlib/std/shared_mem.ail` (new, ~30 LOC) - re-exports

**Example:**
```ailang
module examples/shared_mem_basic

import shared/sem
import std/io

func main() -> unit ! {IO, SharedMem, Clock} =
  -- Store some frames (content is hashed, opaque is payload)
  let f1 = make_frame("doc1", "The quick brown fox", bytes_from_string("payload1")) in
  let f2 = make_frame("doc2", "The quick brown dog", bytes_from_string("payload2")) in  -- Similar!
  let f3 = make_frame("doc3", "Hello world", bytes_from_string("payload3")) in          -- Different
  let _ = store_frame("cache:doc1", f1) in
  let _ = store_frame("cache:doc2", f2) in
  let _ = store_frame("cache:doc3", f3) in

  -- Find similar to "The quick brown cat"
  let query_hash = simhash("The quick brown cat") in
  let matches = find_similar(query_hash, 5) in

  IO.println("Found " ++ int_to_string(length(matches)) ++ " similar frames:")
  -- Should find doc1 and doc2 (structural similarity), not doc3

  -- Note: "car" vs "automobile" would NOT match - that requires Tier 2 embeddings
```

**Acceptance Criteria:**
- [ ] Example runs with `ailang run --caps IO,SharedMem,Clock --entry main`
- [ ] Near-duplicate detection works correctly
- [ ] Integration tests pass
- [ ] 90%+ coverage on new code

---

## Optional: AI.embed (Add if time permits)

If Sprints 1-2 complete early, add embeddings:

### Day 8-9: Embedder Interface + Ollama

**Tasks:**
- [ ] Define Embedder interface
- [ ] Add stub embedder for tests
- [ ] Implement OllamaEmbedder
- [ ] Wire to EffContext

**Files:**
- `internal/effects/embedder.go` (new, ~80 LOC)
- `internal/effects/embedder_stub.go` (new, ~40 LOC)
- `internal/effects/embedder_ollama.go` (new, ~80 LOC)

**Config:**
```yaml
embedding:
  provider: none | ollama    # Default: none
  ollama:
    endpoint: "http://127.0.0.1:11434"
    model: embeddinggemma
    dim: 768
```

### Day 10: AI.embed Builtin

**Tasks:**
- [ ] Add AI.embed builtin
- [ ] Add make_frame_with_embedding helper
- [ ] Create embedding example

**Acceptance Criteria:**
- [ ] `AI.embed(["text"])` returns `list[list[float]]`
- [ ] Works with local Ollama + EmbeddingGemma
- [ ] Clear error if Ollama not running

---

## Success Criteria

### Must Have (MVP Complete)
- [ ] bytes builtins working (5 functions)
- [ ] SimHash builtins working (simhash, hamming_distance)
- [ ] sem_frame type with content/opaque separation
- [ ] simhash64 type alias documented
- [ ] make_frame (with Clock) and make_frame_at (pure) constructors
- [ ] SharedMem effect with in-memory backend (copy-on-read/write)
- [ ] CAS with explicit nil semantics (create-if-absent)
- [ ] load_frame, store_frame, update_frame helpers
- [ ] find_similar as backend capability (not effect-level)
- [ ] Thread-safe under stress (100+ goroutines, -race clean)
- [ ] 90%+ test coverage
- [ ] Working example demonstrating near-duplicate detection

### Nice to Have (Deferred to v0.5.12)
- [ ] AI.embed with Ollama (Tier 2 semantic similarity)
- [ ] Embedding storage as packed bytes
- [ ] with_sem_cache caching primitive
- [ ] LSH index for faster similarity search
- [ ] Redis/Firestore backends

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Thread-safety bugs | High | Day 4-5 stress tests + semantic invariant tests |
| Torn writes ([]byte sharing) | High | Copy-on-read/write in all cache ops |
| bytes type issues | Medium | Day 1 validates bytes work end-to-end |
| CAS semantics confusion | Medium | Explicit nil handling documented |
| SimHash false positives | Low | Hamming distance threshold is tunable |
| SimHash not semantic | N/A | Expected - clearly documented as Tier 1 |
| Users expect semantic matching | Medium | Clear docs: "near-duplicate" not "semantic" |

### Thread Safety: Semantic Correctness

Beyond race detection, tests must verify **semantic invariants**:

1. **No torn writes:** Value stored must equal value retrieved
2. **CAS compares stable values:** Must compare copies, not shared arrays
3. **Get returns independent copy:** Caller mutation can't corrupt cache
4. **Put copies input:** Caller can safely reuse their buffer

```go
// Test semantic correctness
func TestNoTornWrites(t *testing.T) {
    cache := NewInMemorySharedCache()
    value := []byte("original")
    cache.Put("key", value)

    // Mutate caller's buffer
    value[0] = 'X'

    // Cache should still have original
    got, _ := cache.Get("key")
    assert.Equal(t, []byte("original"), got)
}
```

### SimHash Tradeoffs

**What SimHash catches:**
- Near-duplicates (typos, minor edits)
- Reordered text (bag-of-words similarity)
- Copy-paste with small changes

**What SimHash misses:**
- Semantic similarity ("car" vs "automobile")
- Paraphrased content
- Different wording, same meaning

**This is intentional** - Tier 1 is structural prefilter, Tier 2 adds semantic matching.

---

## Post-MVP (v0.5.12+)

Deferred to future sprints:
1. **Redis backend** - Production-ready persistent cache
2. **Firestore backend** - Cloud-native option
3. **Multi-agent demo** - Planner/Critic/Executor coordination
4. **In-process embedder** - llama.cpp/MLX for zero-dependency embeddings

---

## Open Questions (For Future Sprints)

These are deferred to v0.5.12+ but worth considering now:

### 1. Namespace Convention
Should SharedMem support namespaces at the effect level, or keep it as a key convention?

**Options:**
- **A) Convention:** `project:collection:key` - flexible, no API change
- **B) Effect-level:** `SharedMem.get_ns("project", "key")` - type-safe, verbose

**Recommendation:** Start with convention (A), formalize later if needed.

### 2. Similarity Search Backend Strategy
For cloud backends (Redis, Firestore), should similarity search be:

**Options:**
- **A) Client-side:** Fetch all keys, filter locally (simple, but O(n))
- **B) Backend-native:** Use Redis modules or Firestore indexes (complex, but efficient)
- **C) Hybrid:** Client-side for small caches, backend for large

**Recommendation:** Client-side (A) for MVP, add backend-native as optimization.

### 3. Embedding Dimension Flexibility
Should `embedding_dim` be configurable per-frame, or global?

**Options:**
- **A) Per-frame:** Flexible but requires runtime dimension checking
- **B) Global config:** Simpler but less flexible

**Recommendation:** Per-frame (A) - already in sem_frame type.

### 4. Content vs Opaque Relationship
When should opaque be derived from content?

**Use cases:**
- **Cache:** opaque = serialized response, content = request text
- **Document store:** opaque = full doc, content = title + summary
- **Agent state:** opaque = state blob, content = description

**Recommendation:** Document the pattern but don't enforce - caller decides.

---

**Sprint Plan Created:** 2025-12-14
**Based on:** DX-15 full sprint plan analysis + design review feedback
**Target Version:** v0.5.11
