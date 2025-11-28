# Sprint Plan: DX-15 Semantic Caching (SharedMem + sem_frame)

## Summary

Introduce AILANG's primary cache mechanism: a semantic shared-memory effect (`SharedMem`) with canonical data model (`sem_frame`), optional embedding support (`AI.embed`), and backend-agnostic runtime (Redis/Firestore/in-memory). This enables multi-agent coordination, deterministic caching, and AI-native persistent state.

**Duration:** 4 sprints (15-20 days total)
**Dependencies:**
- Effect system (already implemented: IO, FS, Clock, Net, Env)
- JSON encode/decode (std/json exists)
- M-ARRAY-TYPE (parallel development, not blocking)
**Risk Level:** High (new paradigm, multiple integration points)

---

## Current Status Analysis

### Existing Foundation
- **Effects infrastructure**: IO, FS, Clock, Net, Env effects working ([internal/effects/](internal/effects/))
- **Effect context**: 21KB context.go with capability tracking
- **JSON support**: std/json.encode and std/json.decode available
- **Test harness**: MockEffContext for hermetic testing

### Velocity (Last 14 Days)
- M-TESTING-DEPS: ~400 LOC (cross-function dependencies)
- Aliasing feature: ongoing work
- Average: ~150-250 LOC/day for complex features
- Estimated capacity: 200 LOC/day with tests

### What Doesn't Exist Yet
- SharedMem effect (no persistent/shared state)
- sem_frame type (no canonical cache entry)
- AI.embed (no embedding interface)
- SharedCache Go interface (no cache abstraction)
- No multi-agent coordination primitives

---

## Sprint Overview

| Sprint | Focus | LOC Est | Days |
|--------|-------|---------|------|
| S1 | SharedMem + In-Memory | ~1,200 | 5-6 |
| S2 | sem_frame + Helpers | ~800 | 4-5 |
| S3 | AI.embed + Embedder | ~700 | 4-5 |
| S4 | Production Backends + Demo | ~800 | 4-5 |
| **Total** | | **~3,500** | **17-20** |

---

## Sprint 1: SharedMem Effect + In-Memory Backend

**Goal:** Implement core SharedMem effect with get/put/cas operations and in-memory backend for local development.

**Estimated:** 800 LOC implementation + 400 LOC tests = ~1,200 LOC
**Duration:** 5-6 days

### Day 1: Effect Declaration + Type Foundation

**Tasks:**
- [ ] Define `sem_key` type alias in types system
- [ ] Add SharedMem effect to effect registry
- [ ] Define effect operations: get, put, cas
- [ ] Wire effect capability to capability system

**Files:**
- `internal/effects/shared_mem.go` (new, ~150 LOC)
- `internal/types/types.go` (extend, ~30 LOC)
- `internal/effects/capability.go` (extend, ~20 LOC)

**Acceptance Criteria:**
- [ ] `SharedMem` appears in capability list
- [ ] Effect can be declared: `func f() -> T ! {SharedMem}`
- [ ] Parser accepts SharedMem in effect row

### Day 2: SharedCache Interface + In-Memory Implementation

**Tasks:**
- [ ] Define SharedCache Go interface
- [ ] Implement InMemorySharedCache (thread-safe map)
- [ ] Add CAS semantics (compare-and-swap)
- [ ] Wire to effect context

**Files:**
- `internal/effects/shared_cache.go` (new, ~200 LOC)
- `internal/effects/shared_cache_memory.go` (new, ~150 LOC)

**Acceptance Criteria:**
- [ ] InMemorySharedCache passes thread-safety tests
- [ ] CAS returns true only on exact match
- [ ] Get returns `(val, ok=false)` for missing keys at Go layer; maps to `None` at AILANG level

### Day 3: Effect Handlers + Evaluator Integration

**Tasks:**
- [ ] Implement SharedMem.get handler
- [ ] Implement SharedMem.put handler
- [ ] Implement SharedMem.cas handler
- [ ] Add SharedMem to effect dispatch in evaluator

**Files:**
- `internal/effects/shared_mem_handlers.go` (new, ~150 LOC)
- `internal/eval/eval.go` (extend, ~50 LOC)

**Acceptance Criteria:**
- [ ] `SharedMem.get("key")` returns `option[bytes]`
- [ ] `SharedMem.put("key", bytes)` succeeds
- [ ] `SharedMem.cas("key", old, new)` returns bool

### Day 4: Builtins Registration + AILANG Integration

**Tasks:**
- [ ] Register SharedMem builtins in builtin registry
- [ ] Add type signatures for get/put/cas
- [ ] Create `std/shared_mem.ail` module skeleton
- [ ] Wire handlers through pipeline

**Files:**
- `internal/builtins/shared_mem.go` (new, ~100 LOC)
- `stdlib/std/shared_mem.ail` (new, ~50 LOC)

**Acceptance Criteria:**
- [ ] `import std/shared_mem` works
- [ ] SharedMem operations callable from AILANG
- [ ] Types inferred correctly

### Day 5-6: Testing + Configuration

**Tasks:**
- [ ] Write comprehensive unit tests for SharedCache
- [ ] Write integration tests for SharedMem effect
- [ ] Add configuration schema for cache provider
- [ ] Create example: basic key-value store

**Files:**
- `internal/effects/shared_mem_test.go` (new, ~300 LOC)
- `internal/effects/shared_cache_test.go` (new, ~200 LOC)
- `examples/shared_mem_basic.ail` (new, ~40 LOC)

**Acceptance Criteria:**
- [ ] 90%+ test coverage for new code
- [ ] Example runs with `--caps SharedMem`
- [ ] Config allows switching providers

**Sprint 1 Risks:**
| Risk | Impact | Mitigation |
|------|--------|-----------|
| CAS semantics complexity | Medium | Follow Redis WATCH/MULTI pattern |
| Thread-safety bugs | High | Use sync.RWMutex + extensive testing |
| Effect wiring issues | Medium | Follow existing IO/FS pattern |

---

## Sprint 2: sem_frame Type + Helper Functions

**Goal:** Implement canonical sem_frame type and standard helpers for frame operations.

**Estimated:** 500 LOC implementation + 300 LOC tests = ~800 LOC
**Duration:** 4-5 days

### Day 1: sem_frame Type Definition

**Tasks:**
- [ ] Define sem_frame record type
- [ ] Add vector type for embeddings (or use list[float])
- [ ] Define SEM_EMBED_DIM constant
- [ ] Add option type for embedding field

**Design (from spec):**
```ailang
type sem_frame = {
  id: string,
  ver: int,
  ts: int,
  embedding: option[list[float]],  -- or vector[float; SEM_EMBED_DIM]
  meta: map[string, string],
  opaque: bytes
}
```

**Files:**
- `stdlib/shared/sem.ail` (new, ~50 LOC)
- `internal/types/types.go` (extend if needed, ~30 LOC)

**Acceptance Criteria:**
- [ ] sem_frame type parses and type-checks
- [ ] Fields accessible via record syntax
- [ ] embedding is optional (None works)

### Day 2: JSON Encoding for sem_frame

**Tasks:**
- [ ] Implement sem_frame JSON serialization
- [ ] Handle option[embedding] encoding
- [ ] Handle bytes encoding (base64)
- [ ] Test round-trip encode/decode

**Files:**
- `stdlib/shared/sem_json.ail` (new, ~80 LOC)
- Test file with encoding tests

**Acceptance Criteria:**
- [ ] encode_json(frame) produces valid JSON
- [ ] decode_json(bytes) restores exact frame
- [ ] bytes field uses base64

### Day 3: load_frame + store_frame Helpers

**Tasks:**
- [ ] Implement load_frame(key) -> option[sem_frame]
- [ ] Implement store_frame(key, frame)
- [ ] Wire through SharedMem effect
- [ ] Handle JSON errors gracefully

**Files:**
- `stdlib/shared/sem_io.ail` (new, ~100 LOC)

**Acceptance Criteria:**
- [ ] load_frame returns None for missing keys
- [ ] store_frame persists to SharedMem
- [ ] Round-trip: store then load returns same frame

### Day 4: update_frame with CAS Retry

**Tasks:**
- [ ] Implement update_frame(key, f) with CAS
- [ ] Add bounded retry logic (MAX_UPDATE_RETRIES = 8-10)
- [ ] Implement `update_result` ADT: `Missing | Updated(T) | Conflict(T)`
- [ ] Log retry attempts for debugging

**Files:**
- `stdlib/shared/sem_io.ail` (extend, ~80 LOC)

**Design:**
```ailang
type update_result[T] =
  | Missing
  | Updated(T)
  | Conflict(T)  -- gave up due to CAS contention

func update_frame(key: sem_key, f: func(sem_frame) -> sem_frame)
  -> update_result[sem_frame] ! {SharedMem, Json}
```

**Acceptance Criteria:**
- [ ] update_frame retries on CAS failure (bounded, max 8-10)
- [ ] Returns `Updated(frame)` on success
- [ ] Returns `Missing` if key doesn't exist
- [ ] Returns `Conflict(last_seen)` if retries exhausted (caller decides next action)

### Day 5: with_sem_cache Primitive

**Tasks:**
- [ ] Implement with_sem_cache(key, compute)
- [ ] Cache miss triggers compute, stores result
- [ ] Cache hit returns stored frame
- [ ] Create comprehensive tests

**Files:**
- `stdlib/shared/sem_cache.ail` (new, ~80 LOC)
- `examples/sem_cache_example.ail` (new, ~50 LOC)

**Acceptance Criteria:**
- [ ] Cache hit skips compute function
- [ ] Cache miss computes and stores
- [ ] Example demonstrates caching pattern

**Sprint 2 Risks:**
| Risk | Impact | Mitigation |
|------|--------|-----------|
| bytes type complexity | Medium | May need to add bytes builtin type |
| JSON edge cases | Low | Follow std/json patterns |
| CAS retry loops | Medium | Add exponential backoff |

---

## Sprint 3: AI.embed + Embedder Interface

**Goal:** Add embedding integration via AI.embed with pluggable backends.

**Estimated:** 450 LOC implementation + 250 LOC tests = ~700 LOC
**Duration:** 4-5 days

### Day 1: Embedder Go Interface

**Tasks:**
- [ ] Define Embedder interface in Go
- [ ] Add Dim() method for dimension validation
- [ ] Create stub embedder for tests
- [ ] Wire to effect context

**Files:**
- `internal/effects/embedder.go` (new, ~100 LOC)
- `internal/effects/embedder_stub.go` (new, ~50 LOC)

**Interface Design:**
```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dim() int
}
```

**Acceptance Criteria:**
- [ ] Embedder interface defined
- [ ] Stub returns zero vectors for testing
- [ ] Dim() returns configured dimension

### Day 2: AI Effect Extension

**Tasks:**
- [ ] Add embed operation to AI effect
- [ ] Define type: `embed(texts: list[string]) -> list[list[float]]`
- [ ] Wire to Embedder interface
- [ ] Handle dimension validation

**Files:**
- `internal/effects/ai.go` (new or extend, ~150 LOC)
- `stdlib/std/ai.ail` (new, ~40 LOC)

**Acceptance Criteria:**
- [ ] AI.embed callable from AILANG
- [ ] Returns list of embedding vectors
- [ ] Dimension matches SEM_EMBED_DIM

### Day 3: OllamaEmbedder (EmbeddingGemma - Primary)

**Tasks:**
- [ ] Add `github.com/ollama/ollama/api` dependency
- [ ] Implement OllamaEmbedder wrapping `api.Client.Embed()`
- [ ] Handle connection errors and timeouts
- [ ] Validate vector dimension matches config

**Implementation: Use Ollama's Official Go Client**

```go
import "github.com/ollama/ollama/api"

type OllamaEmbedder struct {
    client *api.Client
    model  string
    dim    int
}

func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    resp, err := e.client.Embed(ctx, &api.EmbedRequest{
        Model:      e.model,      // "embeddinggemma"
        Input:      texts,        // batch input
        Dimensions: e.dim,        // 768 (or 512/256/128 for MRL)
    })
    if err != nil {
        return nil, err
    }
    return resp.Embeddings, nil  // Already [][]float32!
}

func (e *OllamaEmbedder) Dim() int { return e.dim }
```

**Primary Model: [EmbeddingGemma](https://huggingface.co/google/embeddinggemma-300m)**
- 768-dimensional vectors (Matryoshka: truncate to 512/256/128)
- 300M parameters (~200MB RAM quantized)
- 2048 token context, sub-30ms latency
- 100+ languages, SOTA under 500M on MTEB

**Quick Start:**
```bash
ollama pull embeddinggemma
ollama serve  # localhost:11434
```

**Files:**
- `internal/effects/embedder_ollama.go` (new, ~80 LOC) - **much simpler with official client!**
- `go.mod` - add `github.com/ollama/ollama/api`

**Acceptance Criteria:**
- [ ] Works with local Ollama server via official Go client
- [ ] Returns 768-dimensional vectors (default) or configured MRL size (512/256/128)
- [ ] Validates `len(vector) == embedding.dim` on every call
- [ ] Passes `Dimensions` field to Ollama for MRL truncation
- [ ] Clear error if Ollama not running

### Day 4: Configuration + Provider Selection

**Tasks:**
- [ ] Add embedding config schema
- [ ] Implement provider selection logic
- [ ] Wire config to runtime initialization
- [ ] Add validation for SEM_EMBED_DIM match

**Config Schema:**
```yaml
embedding:
  provider: none | local-gemma | vertex
  dim: 768  # EmbeddingGemma default (or 512/256/128 for smaller traces)

  # Option 1: Ollama (local, recommended for dev/offline)
  local_gemma:
    endpoint: "http://127.0.0.1:11434"
    model: embeddinggemma

  # Option 2: Vertex AI (cloud, for production scale)
  vertex:
    project_id: my-gcp-project
    location: us-central1
    model: embeddinggemma
```

**Files:**
- `internal/config/embedding.go` (new, ~100 LOC)
- Config file updates

**Acceptance Criteria:**
- [ ] Provider selected from config
- [ ] Dimension validated at startup (768 for EmbeddingGemma)
- [ ] Clear error if misconfigured
- [ ] Works offline with Ollama

### Day 5: Testing + make_frame_with_embedding

**Tasks:**
- [ ] Create make_frame_with_embedding helper
- [ ] Write embedding integration tests
- [ ] Test with stub and mock Ollama
- [ ] Create example: semantic similarity

**Files:**
- `stdlib/shared/sem.ail` (extend, ~50 LOC)
- `internal/effects/embedder_test.go` (new, ~200 LOC)
- `examples/semantic_embedding.ail` (new, ~60 LOC)

**Acceptance Criteria:**
- [ ] make_frame_with_embedding produces valid frames
- [ ] Tests pass with stub embedder
- [ ] Example shows embedding workflow with 768-dim vectors
- [ ] Determinism: same input = same output (within FP rounding)

**Sprint 3 Risks:**
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Ollama not installed | Low | Clear error message + install instructions |
| Model not pulled | Low | `ollama pull embeddinggemma` in docs |
| In-process embedding | High | Defer to DX-16 (llama.cpp/MLX) |

---

## Sprint 4: Production Backends + Multi-Agent Demo

**Goal:** Add Redis/Firestore backends and demonstrate multi-agent coordination.

**Estimated:** 500 LOC implementation + 300 LOC tests = ~800 LOC
**Duration:** 4-5 days

### Day 1: RedisSharedCache

**Tasks:**
- [ ] Implement RedisSharedCache
- [ ] Add Redis connection pooling
- [ ] Implement atomic CAS with Lua script
- [ ] Add connection configuration

**Files:**
- `internal/effects/shared_cache_redis.go` (new, ~200 LOC)

**Acceptance Criteria:**
- [ ] Works with local Redis
- [ ] CAS is truly atomic
- [ ] Handles connection errors gracefully

### Day 2: FirestoreSharedCache

**Tasks:**
- [ ] Implement FirestoreSharedCache
- [ ] Use Firestore transactions for CAS
- [ ] Add GCP configuration
- [ ] Handle Firestore-specific errors

**Files:**
- `internal/effects/shared_cache_firestore.go` (new, ~200 LOC)

**Acceptance Criteria:**
- [ ] Works with Firestore emulator
- [ ] Transactions provide CAS semantics
- [ ] Handles quota/rate limits

### Day 3: Backend Provider Selection

**Tasks:**
- [ ] Add shared_cache config schema
- [ ] Implement provider factory pattern
- [ ] Wire to effect context initialization
- [ ] Add health checks for backends

**Config Schema:**
```yaml
shared_cache:
  provider: redis  # redis | firestore | memory
  redis:
    addr: localhost:6379
    db: 0
  firestore:
    project_id: my-project
    collection: ailang_cache
```

**Files:**
- `internal/config/shared_cache.go` (new, ~100 LOC)

**Acceptance Criteria:**
- [ ] Provider selected from config
- [ ] Graceful fallback to memory
- [ ] Health check on startup

### Day 4-5: Multi-Agent Demo

**Tasks:**
- [ ] Create Planner agent example
- [ ] Create Critic agent example
- [ ] Create Executor agent example
- [ ] Demonstrate shared plan coordination

**Demo Scenario:**
```
1. Planner: Creates plan frame at "plan:goal-123"
2. Critic: Reads plan, adds annotations via update_frame
3. Executor: Reads annotated plan, marks progress
4. All share same sem_frame via SharedMem
```

**Files:**
- `examples/agents/planner.ail` (new, ~80 LOC)
- `examples/agents/critic.ail` (new, ~80 LOC)
- `examples/agents/executor.ail` (new, ~80 LOC)
- `examples/agents/README.md` (new, ~50 lines)

**Acceptance Criteria:**
- [ ] All three agents can read/write shared frame
- [ ] CAS prevents lost updates
- [ ] Demo runs end-to-end
- [ ] Documentation explains pattern

**Sprint 4 Risks:**
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Redis CAS complexity | Medium | Use SET NX + Lua script |
| Firestore transaction limits | Medium | Batch updates |
| Demo complexity | Low | Keep simple, document well |

---

## Success Metrics

### Must Have (Sprint 1-2)
- [ ] SharedMem effect working with in-memory backend
- [ ] sem_frame type with all fields
- [ ] load_frame, store_frame, update_frame helpers
- [ ] with_sem_cache caching primitive
- [ ] 90%+ test coverage for new code

### Should Have (Sprint 3)
- [ ] AI.embed interface working
- [ ] OllamaEmbedder working with EmbeddingGemma
- [ ] Embedding dimension validation
- [ ] Example showing semantic frame creation

### Nice to Have (Sprint 4)
- [ ] Redis backend
- [ ] Firestore backend
- [ ] Multi-agent demo
- [ ] In-process embedder (llama.cpp/MLX) - DEFER to DX-16

### Documentation Updates
- [ ] Update CLAUDE.md with SharedMem capability
- [ ] Add semantic caching guide to docs/
- [ ] Document effect trace for SharedMem ops
- [ ] Add configuration reference

---

## Resolved Design Decisions

All major design questions for DX-15 have been resolved:

| Decision | Resolution |
|----------|-----------|
| **bytes type** | New builtin `bytes`, maps to `[]byte` in Go |
| **embedding type** | `list[float]` with runtime dimension check; migration to `vector[float; N]` when M-ARRAY-TYPE lands |
| **CAS retry** | `update_result[T]` ADT: `Missing \| Updated(T) \| Conflict(T)`; bounded retries (8-10) |
| **Embedder backend** | Ollama (primary) + Vertex AI (optional); in-process deferred to DX-16 |
| **Effect trace** | Metadata only (key, size, hit/success); no payloads for PII safety |

**Deferred to DX-16:**
- In-process embedder (EmbeddingGemma via llama.cpp/MLX, no Ollama dependency)

---

## Dependencies on Other v0.5.0 Work

| Feature | Dependency Type | Notes |
|---------|-----------------|-------|
| M-ARRAY-TYPE | None (parallel) | Could use arrays for embedding vectors |
| Consumer Contract | None | SharedMem is internal, not exposed to consumers |
| Go Codegen | Future | SharedMem might need codegen support later |

---

## Notes

- This is AILANG's most ambitious effect system extension
- "Cache" here is really "shared semantic state" - much more than HTTP caching
- Multi-agent coordination is the killer feature - no separate message queues needed
- Start with in-memory backend, production backends are optimization
- EmbeddingGemma via Ollama for MVP; in-process embedding deferred to DX-16

---

**Sprint Plan Created:** 2025-11-28
**Design Doc:** [semantic-caching.md](semantic-caching.md)
**Target Version:** v0.5.0 - v0.5.3
