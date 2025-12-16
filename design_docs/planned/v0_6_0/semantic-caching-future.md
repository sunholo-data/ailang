# Semantic Caching: Future Work (v0.6.0+)

**Status:** Planned
**Target:** v0.6.0+
**Prerequisites:** DX-15, DX-16, DX-17 (all complete in v0.5.11)

---

## What's Already Implemented (v0.5.11)

See [implemented/v0_5_11/semantic-caching-complete.md](../../implemented/v0_5_11/semantic-caching-complete.md)

| Component | Status |
|-----------|--------|
| SharedMem effect (get/put/cas/delete/keys) | ✅ |
| SharedIndex effect (SimHash + embeddings) | ✅ |
| sem_frame type | ✅ |
| Ollama integration (_ollama_embed) | ✅ |
| SimHash + hamming_distance | ✅ |
| bytes type | ✅ |
| In-memory backends | ✅ |

---

## Remaining Work

### 1. Persistent Backends (DX-18)

**Priority:** High
**Estimated LOC:** ~800

#### Redis Backend

```go
type RedisSharedCache struct {
    client *redis.Client
}

func (r *RedisSharedCache) CAS(ctx context.Context, key string, expected, next []byte) (bool, error) {
    // Use WATCH/MULTI/EXEC or Lua script for atomicity
}
```

**Configuration:**
```yaml
shared_cache:
  provider: redis
  redis:
    addr: localhost:6379
    db: 0
    password: ""
```

#### Firestore Backend

```go
type FirestoreSharedCache struct {
    client *firestore.Client
    collection string
}

func (f *FirestoreSharedCache) CAS(ctx context.Context, key string, expected, next []byte) (bool, error) {
    // Use transactions with version/etag fields
}
```

**Configuration:**
```yaml
shared_cache:
  provider: firestore
  firestore:
    project_id: my-project
    collection: ailang_cache
```

---

### 2. Hybrid Search Optimization (DX-19)

**Priority:** Medium
**Estimated LOC:** ~400

Combine SimHash pre-filtering with embedding re-ranking:

```ailang
func hybrid_search(ns: string, query: string, top_k: int) -> list[search_result] ! {IO, SharedIndex} {
  -- Phase 1: Fast SimHash pre-filter (100 candidates)
  let candidates = _sharedindex_find_simhash(ns, _simhash(query), 100, 1000, true)

  -- Phase 2: Re-rank with embeddings (top_k results)
  in let query_emb = _ollama_embed("embeddinggemma", query)
  in rerank_by_embedding(candidates, query_emb, top_k)
}
```

**Benefits:**
- O(N) SimHash scan is fast (8-byte comparison)
- Embedding comparison limited to 100 candidates instead of all entries
- Best of both: speed of SimHash + accuracy of embeddings

---

### 3. In-Process Embedder (DX-20)

**Priority:** Low (Ollama works well)
**Estimated LOC:** ~600

Run embeddings without Ollama dependency:

```go
type LocalEmbedder struct {
    model *llama.Model  // llama.cpp bindings
}

func (e *LocalEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // Direct inference via MLX or llama.cpp
}
```

**Options:**
- `llama.cpp` with Go bindings
- MLX for Apple Silicon
- ONNX Runtime

**Trade-offs:**
- Pro: No external dependency
- Con: Larger binary, CGO complexity

---

### 4. SharedMem.scan(prefix) (Future)

**Priority:** Low
**Estimated LOC:** ~200

Enumerate keys by prefix:

```ailang
-- Current: must know exact keys
let keys = _sharedmem_keys(())  -- returns ALL keys

-- Future: prefix enumeration
let plan_keys = _sharedmem_scan("plan:")  -- returns keys matching prefix
```

**Use Case:** Multi-agent scenarios where agents need to discover related frames.

---

### 5. Fixed-Size Vector Type (Future Language Feature)

**Priority:** Low
**Estimated LOC:** ~500

Replace `list[float]` with **fixed-size** vector type (like Rust's `[T; N]`):

```ailang
-- Current (v0.5.11)
type embedding = list[float]  -- runtime dimension check only

-- Note: Array[float] also works but is still DYNAMIC-sized []float64

-- Future: Fixed-size with compile-time guarantee
type embedding = vector[float; 768]  -- COMPILE-TIME size check
```

**Why this is different from Array[T]:**
- `Array[T]` / `list[T]` → Go `[]T` (dynamic slice, variable length) ✅ IMPLEMENTED
- `vector[T; N]` → Go `[N]T` (fixed-size array, compile-time size) ❌ NOT IMPLEMENTED

**Benefits:**
- Compile-time dimension mismatch detection
- Better memory layout (no slice header)
- Enables optimized vector math operations
- Type-safe: `vector[float; 768]` ≠ `vector[float; 512]`

**Requires:**
- New parser syntax for size literals in types: `vector[float; 768]`
- New `TVector{Element, Size}` type constructor
- Codegen to Go fixed arrays `[N]T`

---

### 6. Default TTL for Messages (DX-21)

**Priority:** Medium
**Estimated LOC:** ~200

Messages currently persist indefinitely. Add configurable default TTL:

**Configuration:**
```yaml
messages:
  # Default TTL for new messages (0 = no expiration)
  default_ttl: 7d

  # Auto-cleanup on session start
  auto_cleanup: true

  # Cleanup strategy: "expired" or "age"
  cleanup_strategy: expired
```

**Implementation:**
```go
func (i *Inbox) SendMessage(msg *InboxMessage) error {
    // Apply default TTL if not explicitly set
    if msg.ExpiresAt == nil && cfg.Messages.DefaultTTL > 0 {
        expires := time.Now().Add(cfg.Messages.DefaultTTL)
        msg.ExpiresAt = &expires
    }
    // ... rest of send logic
}
```

**Benefits:**
- Automatic inbox hygiene (no manual cleanup needed)
- Prevents unbounded database growth
- Configurable per-deployment
- Backward compatible (0 = no expiration, current default)

**Current state (v0.5.11):**
- `ExpiresAt` field exists in schema ✅
- `cleanup --expired` command works ✅
- No default TTL on creation ❌
- No auto-cleanup hook ❌

---

### 7. Cross-Key Transactions (Future)

**Priority:** Low
**Estimated LOC:** ~500

Atomic operations across multiple keys:

```ailang
-- Future API
transaction {
  let frame1 = load_frame("key1")
  let frame2 = load_frame("key2")
  -- Both updates succeed or fail together
  store_frame("key1", update1(frame1))
  store_frame("key2", update2(frame2))
}
```

**Complexity:** Requires two-phase commit or similar distributed transaction protocol.

---

## Summary

| Feature | Priority | LOC | Notes |
|---------|----------|-----|-------|
| Redis backend | High | ~400 | Production persistence |
| Firestore backend | High | ~400 | Cloud-native option |
| Hybrid search | Medium | ~400 | SimHash + embedding combo |
| Default TTL (DX-21) | Medium | ~200 | Auto-expiring messages |
| In-process embedder | Low | ~600 | Remove Ollama dependency |
| SharedMem.scan | Low | ~200 | Prefix enumeration |
| Fixed-size vector | Low | ~500 | New language feature |
| Cross-key transactions | Low | ~500 | Distributed protocol |

**Total remaining:** ~3,200 LOC

---

**Original Design Doc:** [archived version](https://github.com/sunholo-data/ailang/blob/v0.5.10/design_docs/planned/v0_6_0/semantic-caching.md)
**Implementation Status:** [semantic-caching-complete.md](../../implemented/v0_5_11/semantic-caching-complete.md)
**Last Updated:** 2025-12-16
