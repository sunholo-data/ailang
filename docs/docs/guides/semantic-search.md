---
sidebar_position: 7
title: Semantic Search
description: Two-tier semantic search with SimHash and neural embeddings for AI agents
---

# Semantic Search in AILANG

AILANG provides a two-tier semantic search system for building AI agents with memory and retrieval capabilities:

- **Tier 1: SimHash** - Fast, deterministic text similarity (no external dependencies)
- **Tier 2: Neural Embeddings** - True semantic similarity via Ollama + EmbeddingGemma

This guide walks you through setting up both tiers, from basic to advanced configurations.

## Quick Start (5 minutes)

### Tier 1: SimHash-based Search (No Setup Required)

SimHash provides immediate semantic search without any external dependencies:

```ailang
module examples/simhash_search

func main() -> string ! {IO, SharedIndex} {
  -- Store beliefs with SimHash
  let _ = _sharedindex_upsert("beliefs", "b1", _simhash("The sky is blue"), 1, 1000)
  in let _ = _sharedindex_upsert("beliefs", "b2", _simhash("Stars shine at night"), 1, 2000)
  in let _ = _sharedindex_upsert("beliefs", "b3", _simhash("Clouds contain water"), 1, 3000)

  -- Find similar to "What color is the sky?"
  in let query_hash = _simhash("What color is the sky?")
  in let results = _sharedindex_find_simhash("beliefs", query_hash, 3, 100, true)

  in "Found " ++ intToStr(_array_length(results)) ++ " results"
}
```

Run it:
```bash
ailang run --caps IO,SharedIndex --entry main examples/simhash_search.ail
```

**How SimHash Works:**
- Converts text to a 64-bit fingerprint
- Similar texts have similar fingerprints
- Similarity = 1.0 - (hamming_distance / 64)
- Fast, deterministic, no external dependencies
- Best for: lexical similarity, near-duplicate detection

---

## Setting Up Neural Embeddings (Tier 2)

For true semantic understanding ("cat" matches "kitten"), use neural embeddings.

### Step 1: Install Ollama

Ollama runs AI models locally on your machine.

**macOS:**
```bash
brew install ollama
```

**Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**Windows:**
Download from [ollama.com](https://ollama.com/download)

### Step 2: Download an Embedding Model

We recommend **EmbeddingGemma** as the default:

```bash
ollama pull embeddinggemma
```

**Model specs:**
- **Size:** 622MB
- **Dimensions:** 768
- **Parameters:** 300M
- **Languages:** 100+ (multilingual)
- **Latency:** ~160ms per embedding

### Step 3: Start Ollama

```bash
ollama serve
```

Ollama runs on `http://localhost:11434` by default.

### Step 4: Verify Setup

Test with curl:
```bash
curl http://localhost:11434/api/embed -d '{
  "model": "embeddinggemma",
  "input": "Hello world"
}'
```

Or test with AILANG:
```ailang
module test_embedding

func main() -> string ! {IO} {
  let emb = _ollama_embed("embeddinggemma", "Hello world")
  in "Got " ++ intToStr(_array_length(emb)) ++ "-dimensional embedding"
}
```

---

## Neural Semantic Search Example

Full example using embeddings for semantic search:

```ailang
module examples/neural_search

import std/string (floatToStr, intToStr)

-- Store a belief with its embedding
func store_belief(ns: string, key: string, text: string, ver: int, ts: int) -> unit ! {IO, SharedIndex} {
  let embedding = _ollama_embed("embeddinggemma", text)
  in let simhash = _simhash(text)
  in _sharedindex_upsert_emb(ns, key, simhash, embedding, ver, ts)
}

-- Search by semantic similarity
func search(ns: string, query: string, top_k: int) -> list[{key: string, score: float, version: int, timestamp: int}] ! {IO, SharedIndex} {
  let query_emb = _ollama_embed("embeddinggemma", query)
  in _sharedindex_find_by_embedding(ns, query_emb, top_k, 100, true)
}

func main() -> string ! {IO, SharedIndex} {
  -- Store beliefs
  let _ = store_belief("beliefs", "b1", "The sky is blue during daytime", 1, 1000)
  in let _ = store_belief("beliefs", "b2", "Stars are visible at night", 1, 2000)
  in let _ = store_belief("beliefs", "b3", "Clouds are made of water vapor", 1, 3000)
  in let _ = store_belief("beliefs", "b4", "Rain falls from clouds", 1, 4000)
  in let _ = store_belief("beliefs", "b5", "The ocean looks blue", 1, 5000)

  -- Query: "What color is the sky?"
  in let results = search("beliefs", "What color is the sky?", 3)
  in let _ = print_results(results)

  in "Demo complete"
}

func print_results(results: list[{key: string, score: float, version: int, timestamp: int}]) -> unit ! {IO} {
  match results {
    [] => _io_println("No results"),
    [first, ...rest] =>
      let _ = _io_println(first.key ++ " (score: " ++ floatToStr(first.score) ++ ")")
      in print_results(rest)
  }
}
```

Run:
```bash
ailang run --caps IO,SharedIndex --entry main examples/neural_search.ail
```

---

## Recommended Embedding Models

### General Purpose

| Model | Dims | Size | Best For |
|-------|------|------|----------|
| `embeddinggemma` | 768 | 622MB | **Default choice** - great quality, multilingual |
| `nomic-embed-text` | 768 | 274MB | Smaller, English-focused |
| `mxbai-embed-large` | 1024 | 669MB | Higher quality, larger dims |

### Specialized

| Model | Dims | Size | Best For |
|-------|------|------|----------|
| `all-minilm` | 384 | 46MB | Lightweight, fast, low memory |
| `snowflake-arctic-embed` | 1024 | 669MB | Retrieval-focused |
| `bge-large` | 1024 | 1.3GB | Chinese + English, high quality |

### Pull and Try Different Models

```bash
# Try different models
ollama pull nomic-embed-text
ollama pull mxbai-embed-large
ollama pull all-minilm

# Use in AILANG
let emb = _ollama_embed("nomic-embed-text", "Hello world")
```

---

## Configuration Options

### Environment Variables

```bash
# Custom Ollama endpoint
export OLLAMA_HOST=http://localhost:11434

# For remote Ollama server
export OLLAMA_HOST=http://your-server:11434
```

### Timeout Configuration

The `_ollama_embed` builtin uses a 30-second timeout by default. For large batches or slow networks, ensure Ollama is running locally.

---

## Performance Tips

### 1. Batch Your Embeddings

Generate embeddings once at insert time, not at query time:

```ailang
-- GOOD: Store embedding with data
func store_item(text: string) -> unit ! {IO, SharedIndex} {
  let emb = _ollama_embed("embeddinggemma", text)
  in _sharedindex_upsert_emb("items", key, _simhash(text), emb, 1, now())
}
```

### 2. Use SimHash for Pre-filtering

For large datasets, use SimHash as a fast first-pass filter:

```ailang
-- 1. Fast SimHash pre-filter (no Ollama call)
let candidates = _sharedindex_find_simhash("items", _simhash(query), 100, 1000, true)

-- 2. Re-rank top candidates with embeddings (few Ollama calls)
-- ... load full items and compute embedding similarity
```

### 3. Keep Ollama Running

Don't restart Ollama between queries - model loading is the slowest part:

```bash
# Start once, keep running
ollama serve &

# Models stay loaded in memory
```

### 4. Use `maxScan` to Bound Search Time

```ailang
-- Limit to scanning 500 entries max
let results = _sharedindex_find_by_embedding("items", query_emb, 10, 500, true)
```

---

## SimHash vs Embeddings: When to Use Each

| Feature | SimHash (Tier 1) | Embeddings (Tier 2) |
|---------|------------------|---------------------|
| **Similarity type** | Lexical (word overlap) | Semantic (meaning) |
| **Setup** | None | Ollama + model |
| **Speed** | ~1μs | ~160ms |
| **Memory** | 8 bytes per entry | ~6KB per entry (768 floats) |
| **Determinism** | Perfect | Model-dependent |
| **"cat" ↔ "kitten"** | Low score | High score |
| **Best for** | Near-duplicates, fast search | Semantic understanding |

**Use SimHash when:**
- You need instant results
- No external dependencies allowed
- Searching for near-duplicates
- Memory is constrained

**Use Embeddings when:**
- Semantic understanding matters
- "What color is the sky?" should match "The sky appears blue"
- Building conversational agents
- Quality > speed

---

## Hybrid Search Pattern

Combine both tiers for best results:

```ailang
func hybrid_search(ns: string, query: string, top_k: int) -> list[search_result] ! {IO, SharedIndex} {
  -- Tier 1: Fast SimHash pre-filter (get top 100 candidates)
  let simhash_results = _sharedindex_find_simhash(ns, _simhash(query), 100, 1000, true)

  -- Tier 2: Re-rank with embeddings (slower, more accurate)
  in let query_emb = _ollama_embed("embeddinggemma", query)
  in let emb_results = _sharedindex_find_by_embedding(ns, query_emb, top_k, 100, true)

  in emb_results  -- Return embedding-ranked results
}
```

---

## Troubleshooting

### "Connection refused" error

Ollama isn't running:
```bash
ollama serve
```

### "Model not found" error

Pull the model first:
```bash
ollama pull embeddinggemma
```

### Slow embeddings

1. Check if model is loaded: `ollama list`
2. Ensure running locally (not over network)
3. Use a smaller model: `all-minilm` (46MB)

### Out of memory

Use a smaller model:
```bash
ollama pull all-minilm  # Only 46MB
```

Or increase Ollama memory:
```bash
# macOS
export OLLAMA_MAX_VRAM=4096

# Start with more memory
ollama serve
```

---

## API Reference

### SimHash Builtins

```ailang
-- Generate 64-bit SimHash
_simhash(text: string) -> int

-- Hamming distance between two hashes
_hamming_distance(a: int, b: int) -> int

-- Store with SimHash
_sharedindex_upsert(ns: string, key: string, simhash: int, ver: int, ts: int) -> unit ! {SharedIndex}

-- Find by SimHash similarity
_sharedindex_find_simhash(ns: string, query_hash: int, top_k: int, max_scan: int, deterministic: bool)
  -> list[{key: string, score: float, version: int, timestamp: int}] ! {SharedIndex}
```

### Embedding Builtins

```ailang
-- Generate embedding vector (requires Ollama)
_ollama_embed(model: string, text: string) -> list[float] ! {IO}

-- Store with SimHash + embedding
_sharedindex_upsert_emb(ns: string, key: string, simhash: int, embedding: list[float], ver: int, ts: int)
  -> unit ! {SharedIndex}

-- Find by embedding similarity (cosine)
_sharedindex_find_by_embedding(ns: string, query_emb: list[float], top_k: int, max_scan: int, deterministic: bool)
  -> list[{key: string, score: float, version: int, timestamp: int}] ! {SharedIndex}
```

### Utility Builtins

```ailang
-- Count entries in namespace
_sharedindex_entry_count(ns: string) -> int ! {SharedIndex}

-- List all namespaces
_sharedindex_namespaces(()) -> list[string] ! {SharedIndex}

-- Delete entry
_sharedindex_delete(ns: string, key: string) -> unit ! {SharedIndex}
```

---

## Next Steps

- [AI Effect Guide](/docs/guides/ai-effect) - Using the AI effect for LLM calls
- [Agent Messaging](/docs/guides/agent-messaging) - Building multi-agent systems
- [Examples](/docs/examples) - More working examples

