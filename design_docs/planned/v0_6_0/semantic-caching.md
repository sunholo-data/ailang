# DX-15: Semantic Caching (SharedMem + sem_frame)

**Status:** Approved
**Target:** v0.5.x
**Owner:** Mark / Sunholo
**Last updated:** 2025-11-28

---

## 1. Problem & Goal

AILANG targets AI-native systems:
- multiple cooperating agents
- expensive LLM calls
- shared world models and plans
- trace-driven self-improvement

In this setting, "cache" is not just a performance tweak; it is:
- persistent working memory
- world state
- plan store
- multi-agent coordination fabric

Today, these are typically implemented as fragmented infra (ad-hoc Redis keys, agent memory stores, queues, hidden caches). That breaks determinism, observability, and learnability.

**Goal:** introduce a single, typed, effect-tracked semantic cache that:
- is used by all AILANG code for persistent/shared state,
- supports deterministic CAS-style updates,
- is backend-agnostic (memory, Redis, Firestore, ...),
- integrates embeddings for semantic retrieval when available,
- and is explicitly designed for multi-agent cooperation.

We do this via:
- `SharedMem` effect (primary cache mechanism),
- canonical `sem_frame` data model,
- `AI.embed` effect and `SEM_EMBED_DIM` invariant,
- Go `SharedCache` + `Embedder` interfaces under the hood.

---

## 2. High-Level Design

### 2.1 Core abstractions

- **SharedMem effect**
  - abstract access to shared/persistent state: `get`, `put`, `cas`
- **sem_frame** (semantic frame)
  - canonical payload type stored under a `sem_key`
- **AI.embed**
  - returns fixed-dimensional float vectors used as semantic anchors
- **Go runtime interfaces**
  - `SharedCache` (different backends)
  - `Embedder` (local/remote embedding implementations)

All "proper caches" visible to AILANG code go through `SharedMem`.
Local Go-level caches are allowed but considered non-semantic implementation details.

---

## 3. Language-Level Specification

### 3.1 Types

```ailang
module shared/sem

-- Cache key; convention: "<namespace>:<entity>:<id>"
type sem_key = string
```

```ailang
module std/ai

const SEM_EMBED_DIM: int

type embedding = list[float]  -- invariant: length(embedding) == SEM_EMBED_DIM
```

```ailang
type sem_frame = {
  id: string,
  ver: int,
  ts: int,
  embedding: option[embedding],
  meta: map[string, string],
  opaque: bytes
}
```

Field descriptions:
- `id` - domain identifier (e.g., goal id, doc id, region id)
- `ver` - monotone version counter for this key
- `ts` - logical timestamp (e.g. ms since epoch)
- `embedding` - optional semantic anchor; absent if no embedder configured
- `meta` - small, stringly-typed annotations (status, scores, tags)
- `opaque` - domain payload, arbitrary binary (e.g. encoded plan, KV slice)

**Design Decision: `bytes` type**

The `bytes` type is a **new builtin**, mapped to `[]byte` in Go.
- Minimal surface: `bytes` equality, `length`, `string_to_bytes`, `bytes_to_string`
- JSON encoding base64-encodes `bytes` fields automatically

**Design Decision: `embedding` as `list[float]`**

For DX-15, use `list[float]` with dimension invariant enforced at runtime.
- Does not block on M-ARRAY-TYPE
- Runtime enforces: `len(embedding) == SEM_EMBED_DIM`
- Migration path: change to `vector[float; N]` when array type lands

---

### 3.2 SharedMem effect

```ailang
module std/shared_mem

import shared/sem (sem_key)

effect SharedMem {
  get(key: sem_key) -> option[bytes]
  put(key: sem_key, value: bytes) -> unit
  cas(key: sem_key, expected: bytes, next: bytes) -> bool
}
```

Semantics:
- `get` - returns `None` if key not present; otherwise `Some(bytes)`
- `put` - blind overwrite; intended for initialisation or non-contended updates
- `cas` - compare-and-swap; atomic per key:
  - if current value == expected -> overwrite with next, return `true`
  - else -> do nothing, return `false`

All functions touching shared state must declare `! {SharedMem, ...}` in their effect row.

---

### 3.3 Frame helpers (load_frame, store_frame, update_frame)

**Design Decision: `update_result` ADT for retry semantics**

```ailang
module shared/sem_io

import shared/sem (sem_frame, sem_key)
import std/shared_mem (SharedMem)
import std/json (encode_json, decode_json)

type update_result[T] =
  | Missing
  | Updated(T)
  | Conflict(T)  -- gave up due to CAS contention

export func load_frame(key: sem_key)
  -> option[sem_frame] ! {SharedMem, Json} {
  match SharedMem.get(key) {
    None => None,
    Some(bytes) => Some(decode_json(bytes) as sem_frame)
  }
}

export func store_frame(key: sem_key, frame: sem_frame)
  -> unit ! {SharedMem, Json} {
  SharedMem.put(key, encode_json(frame))
}

export func update_frame(
  key: sem_key,
  f: func(sem_frame) -> sem_frame
) -> update_result[sem_frame] ! {SharedMem, Json} {
  let rec loop(remaining: int, last: option[sem_frame])
    -> update_result[sem_frame] ! {SharedMem, Json} {
    if remaining <= 0 {
      match last {
        Some(frame) => Conflict(frame),
        None => Missing
      }
    } else {
      match SharedMem.get(key) {
        None => Missing,
        Some(bytes) => {
          let cur = decode_json(bytes) as sem_frame
          let next = f(cur)

          let cur_bytes = encode_json(cur)
          let next_bytes = encode_json(next)

          if SharedMem.cas(key, cur_bytes, next_bytes) {
            Updated(next)
          } else {
            loop(remaining - 1, Some(cur))  -- backoff in runtime
          }
        }
      }
    }
  }

  loop(MAX_UPDATE_RETRIES, None)  -- configured constant, e.g. 8-10
}
```

Retry semantics:
- `MAX_UPDATE_RETRIES` is a small constant (8-10), configured in stdlib or runtime
- `Conflict` includes the last seen frame for caller decision-making
- Backoff (sleep/jitter) is implemented in the runtime, not AILANG source
- No infinite loops - bounded retry with clear result

Caller usage:
```ailang
match update_frame(key, f) {
  Missing => ...,
  Updated(frame) => ...,
  Conflict(frame) => ...,  -- caller decides: retry, abort, or escalate
}
```

---

### 3.4 Semantic cache primitive: with_sem_cache

```ailang
module shared/sem_cache

import shared/sem (sem_frame, sem_key)
import shared/sem_io (load_frame, store_frame)

export func with_sem_cache(
  key: sem_key,
  compute: func(unit) -> sem_frame ! {AI, IO}
) -> sem_frame ! {SharedMem, Json, AI, IO} {
  match load_frame(key) {
    Some(frame) => frame,
    None => {
      let frame = compute(())
      store_frame(key, frame)
      frame
    }
  }
}
```

Idioms:
- Plan cache: `with_sem_cache("plan:" ++ goal_id, build_plan_for_goal)`
- Doc summary cache: `with_sem_cache("doc:" ++ doc_id, summarise_doc)`
- World region cache: `with_sem_cache("world:" ++ region_id, build_region_state)`

This is the **primary user-facing cache API** in AILANG.

---

### 3.5 AI.embed interface

```ailang
module std/ai

const SEM_EMBED_DIM: int

type embedding = list[float]

effect AI {
  embed(texts: list[string]) -> list[embedding]
  -- other AI ops: call_model, etc.
}
```

Constraints:
- For any returned embedding, `len(embedding) == SEM_EMBED_DIM`
- The runtime enforces this invariant when wiring Embedder

Typical use in frame construction:

```ailang
import std/ai (AI)
import shared/sem (sem_frame)

export func make_frame_with_embedding(id: string, payload: bytes)
  -> sem_frame ! {AI} {
  let [emb] = AI.embed([ "id=" ++ id ])
  {
    id = id,
    ver = 0,
    ts = now_ms(),
    embedding = Some(emb),
    meta = {},
    opaque = payload
  }
}
```

**Design Decision: Remote-only for DX-15**

If no embedder is configured, `AI.embed` is unavailable or fails clearly at configuration time; callers can construct frames with `embedding = None`.

Local GGUF/ONNX embedder is **explicitly deferred to DX-16**.

---

## 4. Runtime Design (Go)

### 4.1 Shared cache interface

```go
type SharedCache interface {
    Get(ctx context.Context, key string) (val []byte, ok bool, err error)
    Put(ctx context.Context, key string, val []byte) error
    CAS(ctx context.Context, key string, expected, next []byte) (bool, error)
}
```

Implementations:
- `InMemorySharedCache` - thread-safe map + `sync.RWMutex`
- `RedisSharedCache` - CAS implemented via Lua script or WATCH/MULTI/EXEC
- `FirestoreSharedCache` - CAS via transactions and etag/version fields

The `SharedMem` effect handler delegates to this interface.

Configuration (schematic):

```yaml
shared_cache:
  provider: memory | redis | firestore
  redis:
    addr: localhost:6379
    db: 0
  firestore:
    project_id: my-project
    collection: ailang_cache
```

On startup:
- Load config
- Instantiate appropriate SharedCache
- Attach to effect context

---

### 4.2 Embedder interface

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dim() int
}
```

DX-15 provides:
- `StubEmbedder` for tests (zero vectors)
- `OllamaEmbedder` for local inference (EmbeddingGemma via Ollama - **primary**)
- `VertexAIEmbedder` for Google Cloud production (optional)

**Implementation: Use Ollama's Official Go Client**

```go
import "github.com/ollama/ollama/api"

// EmbedRequest supports batch embedding with MRL truncation
type EmbedRequest struct {
    Model      string         // "embeddinggemma"
    Input      any            // []string for batch
    Dimensions int            // 768, 512, 256, or 128 (MRL)
    KeepAlive  *Duration      // model memory duration
}

// EmbedResponse returns [][]float32 (batch of vectors)
type EmbedResponse struct {
    Model      string
    Embeddings [][]float32    // The embedding vectors!
}

// Usage
client, _ := api.ClientFromEnvironment()
resp, err := client.Embed(ctx, &api.EmbedRequest{
    Model:      "embeddinggemma",
    Input:      []string{"Hello", "World"},
    Dimensions: 768,
})
// resp.Embeddings is [][]float32
```

This means our `OllamaEmbedder` is just a thin wrapper around `api.Client.Embed()`.

**Primary Model: EmbeddingGemma**

[EmbeddingGemma](https://huggingface.co/google/embeddinggemma-300m) is the recommended embedding model:
- **Dimension**: 768 (default), supports MRL truncation to 512/256/128
- **Parameters**: 300M (~200MB RAM quantized)
- **Context**: 2,048 tokens
- **Languages**: 100+ multilingual, SOTA under 500M on MTEB
- **Latency**: Sub-30ms on modern hardware
- **License**: Gemma terms (open weights, commercial use allowed)

**Why EmbeddingGemma?**
- **On-device friendly**: Designed for phones/laptops, runs locally
- **Deterministic**: Same model + same input = reproducible embeddings
- **Quality**: SOTA under 500M params, multilingual
- **Ollama support**: `ollama pull embeddinggemma` just works
- **Matryoshka**: Truncate to 512/256/128 dims with graceful degradation

Configuration:

```yaml
embedding:
  provider: none | local-gemma | vertex
  dim: 768  # EmbeddingGemma default (or 512/256/128 for smaller traces)

  # Option 1: Ollama (local, recommended for dev/offline)
  local_gemma:
    endpoint: "http://127.0.0.1:11434"
    model: embeddinggemma

  # Option 2: Vertex AI (cloud, recommended for production scale)
  vertex:
    project_id: my-gcp-project
    location: us-central1
    model: embeddinggemma
```

**Quick Start (Local):**
```bash
# Install Ollama and pull the model
ollama pull embeddinggemma
ollama serve  # Runs on localhost:11434

# AILANG config
embedding:
  provider: local-gemma
  dim: 768
```

Initialization:
- If `provider: none`, `AI.embed` is disabled (compile/config fail if used)
- If `provider: local-gemma`, instantiate `OllamaEmbedder`; POST to `/api/embeddings`
- If `provider: vertex`, instantiate `VertexAIEmbedder`; use Vertex AI Embeddings API

The AI effect handler:
1. Calls `Embedder.Embed(texts)`
2. Validates `len(vector) == embedding.dim` (fail fast if mismatch)
3. Converts `[][]float32` to `list[float]`

**Determinism guarantee:**
- Same model weights + same code + same input = numerically stable output
- Trace records: `embedding.model`, `embedding.provider`, `embedding.dim`
- Runs are reproducible as long as user has same model version

---

### 4.3 Effect trace

**Design Decision: Metadata only, no payloads**

Each SharedMem operation logs a compact event:

| Field | Type | Description |
|-------|------|-------------|
| `op` | string | `"SharedMem.get"` \| `"SharedMem.put"` \| `"SharedMem.cas"` |
| `key` | string | The sem_key |
| `size_bytes` | int | `len(val)` where applicable |
| `hit` | bool | For get operations |
| `cas_success` | bool | For CAS operations |
| `backend` | string | `"memory"` \| `"redis"` \| `"firestore"` |
| `retries` | int | For high-level helpers like update_frame |

Example trace entries:

```json
{
  "op": "SharedMem.get",
  "key": "plan:goal-123",
  "hit": true,
  "size_bytes": 742,
  "backend": "redis"
}
```

```json
{
  "op": "SharedMem.cas",
  "key": "plan:goal-123",
  "cas_success": false,
  "size_bytes": 772,
  "backend": "redis"
}
```

**No values/payloads in traces** - avoids PII leaks and massive logs, but supports:
- cache hit/miss profiling
- contention analysis
- training/eval on cache usage patterns
- debugging multi-agent coordination

---

## 5. Multi-Agent Pattern (Conceptual)

Agents coordinate by sharing `sem_frames` under stable keys:

1. **Planner:**
   - computes an initial frame f0 at `plan:<goal_id>` using `with_sem_cache`

2. **Critic:**
   - reads the same key, annotates `meta`, updates via `update_frame`

3. **Executor:**
   - reads `opaque` as plan state, marks progress via `update_frame`

Because all of this routes through `SharedMem`:
- there is a single, unified notion of "cache / shared state" for agents
- CAS ensures per-key linearizable updates
- effect traces record interactions for learning and eval

This is the **reference pattern** for AILANG multi-agent applications.

---

## 6. Non-Goals / Follow-ups

**Not part of DX-15:**

| Feature | Ticket | Notes |
|---------|--------|-------|
| In-process embedder | DX-16 | EmbeddingGemma via llama.cpp/MLX (no Ollama dep) |
| Higher-level pub/sub | Future | Can layer on SharedMem |
| Cross-key transactions | Future | v1 is per-key only |
| `vector[float; N]` type | M-ARRAY-TYPE | Use `list[float]` for now |
| `SharedMem.scan(prefix)` | Future | v1 is per-key only |

---

## 7. Summary of Design Decisions

| Question | Decision | Rationale |
|----------|----------|-----------|
| bytes type | New builtin `bytes`, maps to `[]byte` in Go | Clean separation from string, natural for binary payloads |
| embedding type | `list[float]` with runtime dimension check | Doesn't block on M-ARRAY-TYPE, easy migration later |
| CAS retry | `update_result` ADT: `Missing \| Updated \| Conflict` | Clear semantics, no infinite loops, caller control |
| Embedding model | EmbeddingGemma (768-dim, Gemma license) | On-device, SOTA quality, Ollama support |
| Embedding backend | Ollama (primary), Vertex AI (optional) | Local-first, HTTP interface, simple setup |
| Effect trace | Metadata only (key, size, hit/success), no payloads | Avoids PII leaks, smaller logs |

---

## 8. Sprint Plan Reference

See [DX-15-semantic-caching-sprint-plan.md](DX-15-semantic-caching-sprint-plan.md) for:
- 4-sprint breakdown (17-20 days total, ~3,500 LOC)
- Day-by-day tasks
- File-level implementation details
- Risk mitigations

---

**Document Created:** 2025-11-28
**Design Doc Status:** Approved, ready to implement
**Sprint Plan:** [DX-15-semantic-caching-sprint-plan.md](DX-15-semantic-caching-sprint-plan.md)
