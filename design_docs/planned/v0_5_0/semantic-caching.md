DX-15: Shared Semantic Cache (SharedMem + sem_frame)

Status: Draft
Target: v0.5.x
Owner: Mark / Sunholo
Last updated: 2025-11-28

1. Summary

Introduce a primary cache mechanism for AILANG based on:
	1.	A semantic shared-memory effect: SharedMem
	2.	A canonical semantic frame type: sem_frame
	3.	A standard embedding interface: AI.embed with fixed dimension SEM_EMBED_DIM

This cache is:
	•	Typed: represented as sem_frame
	•	Deterministic (by construction): via an algebraic effect + CAS
	•	Multi-agent-native: same mechanism for caching, shared state, and agent collaboration
	•	Backend-agnostic: Redis / Firestore / local map, etc.

From AILANG program’s POV:
	•	persistent/shared state ⇒ SharedMem
	•	cached intermediate / expensive computations ⇒ with_sem_cache
	•	multi-agent shared plans/world state ⇒ sem_frame keyed by sem_key

This replaces ad-hoc “misc caches” with one canonical semantic cache substrate.

⸻

2. Motivation

AILANG is targeting AI-native systems:
	•	multiple agents
	•	expensive LLM calls
	•	shared world models
	•	self-improving agents via trace replay

In that world, “cache” is not just a performance hack:
	•	it is working memory
	•	it is world state
	•	it is plan store
	•	it is cross-agent communication fabric

Today, these are usually implemented as:
	•	ad-hoc Redis keys
	•	separate “agent memory” stores
	•	separate messaging queues
	•	hidden language-/library-specific caches

This yields:
	•	fragmented semantics
	•	non-deterministic coordination
	•	poor observability (caches invisible to trace)
	•	hard-to-train agents (hidden state)

We want
	•	single conceptual model: semantic shared cache
	•	typed, effect-tracked interaction
	•	deterministic semantics on top of non-deterministic substrates
	•	easy offline/local mode with bundled embedding model
	•	a substrate for “agents thinking together”, not just messaging

Hence: SharedMem + sem_frame as the primary caching paradigm.

⸻

3. Goals / Non-goals

Goals
	1.	Language-level abstraction for persistent/shared state (SharedMem effect).
	2.	Canonical data model for semantic cache entries (sem_frame).
	3.	Embedding integration via AI.embed and SEM_EMBED_DIM, but optional at runtime.
	4.	Deterministic update semantics via CAS-based updates.
	5.	Multi-agent coordination via shared keys and frames.
	6.	Backend-agnostic runtime (SharedCache interface in Go).
	7.	Traceability: cache ops visible in effect trace for eval / training.

Non-goals (for this iteration)
	1.	General-purpose distributed transactions across many keys.
	2.	Full pub/sub or event streams (can be layered later).
	3.	Locking primitives beyond CAS-based optimistic concurrency.
	4.	High-performance clustering/sharding strategies (initial impl can be naive).

⸻

4. High-level Architecture

Conceptual stack:
	•	AILANG language
	•	effect SharedMem { get, put, cas }
	•	type sem_frame
	•	with_sem_cache, load_frame, update_frame helpers
	•	AI stdlib
	•	AI.embed(texts: list[string]) -> list[vector[float; SEM_EMBED_DIM]]
	•	Go runtime
	•	Embedder interface (local / remote)
	•	SharedCache interface (Redis / Firestore / in-memory)
	•	Effect handlers wiring AI + SharedMem to runtime

Key design choice: every “proper cache” visible to AILANG uses SharedMem.
Other caches (purely local in Go) are runtime-only optimisations, not semantic.

⸻

5. Language-Level Spec

5.1 Semantic key and frame

module shared/sem

// Key format convention: "<namespace>:<entity>:<id>"
type sem_key = string

// Dimension configured at build/runtime initialisation
import std/ai (SEM_EMBED_DIM)

// Canonical shared semantic object
type sem_frame = {
  id: string,                                     // logical ID within domain
  ver: int,                                       // monotone version per key
  ts: int,                                        // logical timestamp (e.g., ms since epoch)
  embedding: option[vector[float; SEM_EMBED_DIM]],// semantic anchor (optional)
  meta: map[string, string],                      // lightweight annotations
  opaque: bytes                                   // compressed domain-specific payload
}

Notes:
	•	embedding is optional ⇒ SharedMem usable before embedding is configured.
	•	meta is intentionally simple; domain-specific structure stays in opaque.
	•	ver and ts are for human tooling, debugging, and coarse conflict reasoning.

⸻

5.2 SharedMem effect

module std/shared_mem

import shared/sem (sem_key)

effect SharedMem {
  get(key: sem_key) -> option[bytes]
  put(key: sem_key, value: bytes) -> unit
  cas(key: sem_key, expected: bytes, next: bytes) -> bool
}

Semantics:
	•	get returns the latest stable value (if any).
	•	put blindly overwrites (used rarely; usually initialisation).
	•	cas ensures per-key linearisation:
	•	if current value bytes == expected ⇒ set to next and return true
	•	otherwise ⇒ return false, no state change

All SharedMem usage is explicit via effect typing:

func f(...) -> T ! {SharedMem, ...} { ... }


⸻

5.3 Standard helpers for frames

module shared/sem_io

import shared/sem (sem_frame, sem_key)
import std/shared_mem (SharedMem)
import std/json (encode_json, decode_json)

export func load_frame(key: sem_key)
  -> option[sem_frame] ! {SharedMem, Json} {
  match SharedMem.get(key) {
    None => None,
    Some(bytes) => {
      let frame = decode_json(bytes) as sem_frame
      Some(frame)
    }
  }
}

export func store_frame(key: sem_key, frame: sem_frame)
  -> unit ! {SharedMem, Json} {
  let bytes = encode_json(frame)
  SharedMem.put(key, bytes)
}

// CAS-based atomic update with retry
export func update_frame(
  key: sem_key,
  f: func(sem_frame) -> sem_frame
) -> option[sem_frame] ! {SharedMem, Json} {
  match SharedMem.get(key) {
    None => None,
    Some(bytes) => {
      let cur = decode_json(bytes) as sem_frame
      let next = f(cur)

      let cur_bytes = encode_json(cur)
      let next_bytes = encode_json(next)

      if SharedMem.cas(key, cur_bytes, next_bytes) {
        Some(next)
      } else {
        // Implementation may choose bounded retry or explicit backoff.
        update_frame(key, f)
      }
    }
  }
}

5.4 “Semantic cache” primitive: with_sem_cache

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

Patterns:
	•	Cached plan for goal_id: with_sem_cache("plan:" ++ goal_id, build_plan)
	•	Cached document summary: with_sem_cache("doc:" ++ doc_id, summarise)
	•	Cached world region: with_sem_cache("world:" ++ region_id, build_region_state)

⸻

5.5 AI embedding interface

module std/ai

// Fixed at runtime initialisation; used in type signatures.
const SEM_EMBED_DIM: int

effect AI {
  // Other operations: call_model, etc.
  embed(texts: list[string]) -> list[vector[float; SEM_EMBED_DIM]]
}

Usage in semantic frame construction:

import std/ai (AI)
import shared/sem (sem_frame)

export func make_frame_with_embedding(id: string, payload: bytes)
  -> sem_frame ! {AI} {
  let embs = AI.embed([ "id=" ++ id ])
  let emb = head(embs)

  {
    id = id,
    ver = 0,
    ts = now_ms(),
    embedding = Some(emb),
    meta = {},
    opaque = payload
  }
}

Embedding is optional: you can also create frames with embedding = None if no embedder is configured.

⸻

6. Multi-Agent Example (Sketch)

Three agents collaborate around plan:<goal_id>:
	•	Planner: creates plan frame via with_sem_cache.
	•	Critic: reads + annotates plan via update_frame.
	•	Executor: reads + marks progress via update_frame.

Key points:
	•	All use same key and sem_frame.
	•	No messaging queues; coordination via SharedMem.
	•	Cache and world state are the same substrate.

You already have enough in the spec to write this as real AILANG code when the compiler is ready.

⸻

7. Runtime Design (Go)

7.1 Embedding provider

Define an Embedder interface:

type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dim() int
}

Implementations:
	•	LocalFileEmbedder (GGUF/ONNX on CPU; model path from config)
	•	EmbeddedModelEmbedder (weights bundled in binary via //go:embed)
	•	RemoteHTTPEmbedder (OpenAI / Vertex / etc.)

On runtime init:
	•	Read config (embedding.provider, model_path, etc.).
	•	Instantiate concrete Embedder.
	•	Check Embedder.Dim() == SEM_EMBED_DIM (configured constant).
	•	Wire into AI effect handler.

This keeps the language agnostic to local vs remote; determinism guarantees are a property of the runtime configuration, not the language spec.

⸻

7.2 Shared cache provider

Define a SharedCache interface:

type SharedCache interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Put(ctx context.Context, key string, value []byte) error
    CAS(ctx context.Context, key string, expected, next []byte) (bool, error)
}

Implementations:
	•	InMemorySharedCache (for tests / dev)
	•	RedisSharedCache
	•	FirestoreSharedCache
	•	later: AlloyDBSharedCache, etc.

The SharedMem effect handler maps directly to these calls.

7.3 Configuration schema (example)

embedding:
  provider: local     # local | remote
  model_path: ./models/ailang-embed-small.gguf
  dim: 384

shared_cache:
  provider: redis     # redis | firestore | memory
  redis:
    addr: localhost:6379
    db: 0
  firestore:
    project_id: my-gcp-project
    collection: ailang_shared_cache

Startup:
	•	Instantiate Embedder using embedding.*.
	•	Instantiate SharedCache using shared_cache.*.
	•	Register handlers for AI.embed and SharedMem.{get,put,cas}.

⸻

8. Determinism & Observability

8.1 Determinism model

Given:
	•	A fixed runtime build (including SEM_EMBED_DIM),
	•	A fixed configuration (embedding provider + model, shared cache backend),
	•	A recorded sequence of effects (AI.call_model, AI.embed, SharedMem ops),

AILANG program behaviour is:
	•	Deterministic relative to that “world description”.
	•	SharedMem updates are linearizable per key via CAS.
	•	Effect trace contains all cache interactions.

Two modes:
	•	Strong determinism (local mode):
	•	local embedding model; CPU-only; deterministic backend (e.g. in-memory or single-node Redis).
	•	Config-deterministic (remote mode):
	•	remote embedding; stability depends on provider, but configuration is logged with each run.

8.2 Effect trace

SharedMem operations should be logged in the trace:
	•	SharedMem.get(key, hit|miss, size_bytes)
	•	SharedMem.cas(key, success|fail, size_bytes)
	•	Possibly a derived sem_frame view for tooling (decode meta).

This enables:
	•	profiling cache hit/miss,
	•	debugging multi-agent interactions,
	•	training agents on “good cache usage” behaviour.

⸻

9. Why this is the Primary Cache Mechanism
	1.	Unification
	•	Caching, shared state, agent memory, and inter-agent coordination all use SharedMem.
	•	No separate APIs for “HTTP cache”, “agent memory”, etc.: all are namespaces over sem_key.
	2.	Semantically rich
	•	sem_frame.embedding gives semantic similarity and clustering.
	•	meta and opaque allow arbitrary domain-specific payload.
	3.	Effect-typed and observable
	•	Cache behaviour is explicit and recordable; no invisible performance magic.
	•	Supports replay and evaluation.
	4.	Multi-agent-native
	•	Any two agents that share a key share cognitive state.
	•	No need to marshal through text messages for collaboration.
	5.	Backend-agnostic
	•	Same language semantics over Redis, Firestore, in-memory, etc.
	6.	Optional embedding
	•	Systems can start with embedding = None and still get shared-cache benefits.
	•	Embeddings become a powerful upgrade, not a hard dependency.

Result: one mental model for “where does persistent state live?” and “how do agents collaborate?” rather than a zoo of orthogonal caches.

⸻

10. Rollout Plan
	1.	Phase 1 – Runtime + stdlib skeleton
	•	Implement SharedCache + SharedMem handler in Go.
	•	Implement sem_key, sem_frame, shared/sem_io, shared/sem_cache.
	•	Implement Embedder interface and a simple RemoteHTTPEmbedder (OpenAI).
	2.	Phase 2 – Local embedding
	•	Add LocalFileEmbedder with a small GGUF/ONNX model.
	•	Wire configuration + SEM_EMBED_DIM enforcement.
	3.	Phase 3 – Use in a small multi-agent demo
	•	Planner/Critic/Executor for plans or a tiny “world region” simulation.
	•	Measure latency/token savings vs pure text/messaging baseline.
	4.	Phase 4 – Adopt across stdlib / examples
	•	LLM result cache built on SharedMem.
	•	Document summary cache built on SharedMem.
	•	Example Emissary/VAC integration.

⸻

11. Open Questions
	1.	Retry semantics for update_frame
	•	Bounded retries vs explicit “conflict” error.
	•	Backoff strategy (especially under heavy contention).
	2.	Scanning / listing
	•	Do we want SharedMem.scan(prefix: string) in v1, or keep v1 per-key only?
	•	Useful for “list all frames under plan: namespace”.
	3.	Embedding model standardisation
	•	Ship an official “AILANG semantic small” model?
	•	Or rely on existing open source embeddings (e.g., BGE-like)?
	4.	Versioning semantics
	•	Do we give ver any semantics beyond “monotone counter” (e.g., tie into eval)?

