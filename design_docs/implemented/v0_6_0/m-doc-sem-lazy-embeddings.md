# M-DOC-SEM: Lazy Embeddings for Neural Doc Search

**Status**: IMPLEMENTED
**Target**: v0.5.11
**Priority**: P1 (Medium)
**Estimated**: 2-3 days → **Actual**: 0.75 days (1005 LOC)
**Dependencies**: M-DX15 (SimHash builtins - COMPLETE), SharedIndex infrastructure
**Implemented**: 2025-12-16

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes need for separate indexing commands |
| Preserve Semantic Clarity | 0 | 0 | Neural search is explicit via `--neural` flag |
| Increase Determinism | + | +1 | Bounded candidate set, deterministic pipeline |
| Lower Token Cost | + | +1 | Caching effect reduces repeated embedding work |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

Neural search via `--neural` flag currently has unclear behavior around when embeddings are computed. A naive implementation would trigger "embed the whole repo" on first search, which is:

**Current State:**
- No lazy embedding strategy defined
- Risk of unbounded compute on first neural search
- No caching of embeddings between searches
- Model changes would require full re-embedding

**Impact:**
- AI agents using `ailang docs search --neural` could face unexpected latency
- No clear cost bounds for embedding operations
- Poor developer experience when changing embedding models

## Goals

**Primary Goal:** Compute embeddings lazily only for bounded SimHash candidate sets, enabling fast neural search with predictable resource usage.

**Success Metrics:**
- Neural search NEVER embeds entire doc corpus (bounded to `topK` candidates)
- Embeddings cached in SharedMem with model version tagging
- First neural search completes in <10s for typical queries
- Subsequent searches reuse cached embeddings (near-instant)

## Solution Design

### Overview

Lazy embeddings use a two-stage pipeline:
1. **SimHash shortlist** - Fast, deterministic candidate filtering (always runs first)
2. **Embed missing candidates only** - Compute embeddings lazily for shortlisted frames

This gives us:
- Deterministic behavior
- Bounded compute
- Caching effect (once embedded, future searches are fast)

### Architecture

**Two-Stage Pipeline:**

```
Query → SimHash → Bounded Candidates → Embed Missing → Embedding Search → Results
         ↓                ↓                  ↓
      (always)        (topK limit)     (cached in SharedMem)
```

**Components:**

1. **SimHash Shortlist (Stage 1)**
   - Compute query simhash
   - Use SharedIndex to find `topK` candidates (default: `max(200, 20*limit)`)
   - Deterministic: same query → same candidates

2. **Lazy Embedding (Stage 2)**
   - For each candidate: load frame from SharedMem
   - If `frame.embedding` missing/empty OR model mismatch → compute `_ollama_embed(model, frame.content)`
   - Store updated frame back (CAS update)
   - Upsert embedding into SharedIndex

3. **Query Embedding**
   - Compute `qemb = _ollama_embed(model, query_text)` once

4. **Embedding Search (Stage 3)**
   - Call `_sharedindex_find_by_embedding(namespace, qemb, limit, max_scan, deterministic=true)`
   - Return top results; load frames; print path/title

### Model/Version Tagging (MUST)

Store in frame opaque or dedicated fields:
- `embedding_model`: e.g. "embeddinggemma"
- `embedding_updated_at`: unix millis
- `embedding_dim`: optional

**Model change behavior:** If model changes, treat existing embeddings as invalid and recompute lazily on next search.

### Implementation Plan

**Phase 1: CLI Foundation** (COMPLETE ✅)
- [x] Add `ailang docs search` command skeleton
- [x] Add `--neural`, `--neural-candidates`, `--stream`, `--limit`, `--json` flags
- [x] Frame metadata with `embedding_model`, `embedding_updated_at` fields

**Phase 2: SimHash Shortlist** (COMPLETE ✅)
- [x] Implement SimHash-based candidate filtering
- [x] Bound candidates by `--neural-candidates` (default: `max(200, 20*limit)`)

**Phase 3: Lazy Embedding Pipeline** (COMPLETE ✅)
- [x] Implement lazy embedding with Ollama
- [x] Cache embeddings to `~/.ailang/cache/doc_embeddings.json`
- [x] Model version tagging - recompute on model change

**Phase 4: Observability** (COMPLETE ✅)
- [x] Print stats: `embedded_now=K reused=N-K`
- [x] Add `--json` output format
- [x] Update CLAUDE.md with agent instructions

**Phase 5: General Corpus Support** (COMPLETE ✅)
- [x] Add `--path <dir>` flag (default: `design_docs`)
- [x] `--subdir` filter for general filtering (`--stream` kept as alias)
- [x] Per-corpus cache files: `~/.ailang/cache/embeddings/<corpus-hash>.json`
- [x] Update help text for general-purpose usage

**Phase 6: Content Hash Caching** (COMPLETE ✅)
- [x] Add `content_hash` (SHA256) to `CachedEmbedding` struct
- [x] On cache lookup: verify content hash matches current file
- [x] Hash mismatch → automatic recompute (content changed)
- [x] Store absolute paths for reliable cache key matching

**Phase 7: Cache Management Commands** (COMPLETE ✅)
- [x] Add `--cleanup` flag to remove orphaned entries (paths that no longer exist)
- [x] Add `--cache-info` flag to show cache stats (entries, size, model, corpus)
- [x] Add `--rebuild` flag to force full cache rebuild

### Files to Modify/Create

**Modified files:**
- `internal/docs/search.go` - Add lazy embedding pipeline, ~150 LOC
- `internal/docs/frame.go` - Add embedding metadata fields, ~20 LOC
- `cmd/ailang/docs.go` - Add `--neural-candidates` flag, ~10 LOC

**New files:**
- `internal/docs/embed_lazy.go` - Lazy embedding logic, ~100 LOC
- `internal/docs/embed_lazy_test.go` - Tests, ~150 LOC

## Examples

### Example 1: Neural Search with Lazy Embedding

**Command:**
```bash
ailang docs search "parser error handling" --stream planned --neural
```

**Output:**
```
🔍 Neural search: "parser error handling"
   SimHash candidates: 200 (from 847 total docs)
   Embeddings: 45 computed, 155 reused (model: embeddinggemma)

1. design_docs/planned/v0_5_9/m-dx9-parser-developer-experience.md (0.89)
2. design_docs/implemented/v0_3_10/error-handling.md (0.85)
3. internal/parser/README.md (0.82)
```

### Example 2: Subsequent Search (Cached)

**Command:**
```bash
ailang docs search "parser error" --stream planned --neural
```

**Output:**
```
🔍 Neural search: "parser error"
   SimHash candidates: 200 (from 847 total docs)
   Embeddings: 0 computed, 200 reused (model: embeddinggemma)

1. design_docs/planned/v0_5_9/m-dx9-parser-developer-experience.md (0.91)
...
```

### Example 3: Custom Candidate Limit

**Command:**
```bash
ailang docs search "types" --neural --neural-candidates 50 --limit 5
```

**Output:**
```
🔍 Neural search: "types"
   SimHash candidates: 50 (bounded by --neural-candidates)
   Embeddings: 12 computed, 38 reused (model: embeddinggemma)

1-5. [results...]
```

### Example 4: Search Different Corpus (Phase 5)

**Command:**
```bash
# Search website documentation instead of design docs
ailang docs search --path docs "semantic caching"

# Search with subdirectory filter
ailang docs search --path docs --subdir guides "getting started"
```

**Output:**
```
🔍 Search: "semantic caching" (corpus: docs)
   Total docs: 45
   SimHash candidates: 45

1. docs/guides/semantic-caching-how-to.mdx (0.92)
2. docs/guides/semantic-search.md (0.87)
3. docs/intro.mdx (0.71)
```

### Example 5: Cache Management (Phase 7)

**Command:**
```bash
# Show cache statistics
ailang docs search --cache-info

# Clean orphaned entries (files that moved/deleted)
ailang docs search --cleanup

# Force rebuild of all embeddings
ailang docs search --rebuild "query"
```

**Output (`--cache-info`):**
```
📦 Embedding Cache Info:
   Corpus: design_docs (hash: a1b2c3d4)
   Model: embeddinggemma
   Entries: 156
   Cache size: 2.4 MB
   Last updated: 2025-12-16 15:43:00

   Orphaned entries: 3 (use --cleanup to remove)
```

**Output (`--cleanup`):**
```
🧹 Cache cleanup:
   Removed 3 orphaned entries:
   - design_docs/planned/v0_5_10/old-feature.md (deleted)
   - design_docs/implemented/v0_5_8/moved-doc.md (moved)
   - design_docs/planned/v0_5_9/renamed.md (renamed)

   Cache size: 2.4 MB → 2.3 MB
```

## Agent Instruction Block Update

Add to agent docs:

> **When using `--neural`**, embeddings are computed lazily only for the bounded SimHash candidate set; do not attempt to embed the entire doc corpus.

## Success Criteria

**Phase 1-4 (COMPLETE ✅):**
- [x] `ailang docs search --neural` uses two-stage pipeline
- [x] Candidate set bounded by `--neural-candidates` (default: `max(200, 20*limit)`)
- [x] Embeddings cached with model version; mismatch triggers recompute
- [x] Stats printed: `embedded_now=K reused=N-K`
- [x] `--json` output format works
- [x] Agent instruction block updated

**Phase 5-7 (COMPLETE ✅):**
- [x] `--path <dir>` supports arbitrary document corpora
- [x] `--subdir <pattern>` filters by subdirectory (`--stream` kept as backwards-compatible alias)
- [x] Per-corpus cache files (`~/.ailang/cache/embeddings/<corpus-hash>.json`)
- [x] Content hash (SHA256) validates cache freshness (auto-recompute on change)
- [x] `--cleanup` removes orphaned cache entries
- [x] `--cache-info` shows cache statistics
- [x] `--rebuild` forces full cache rebuild

## Implementation Summary

**Files Created:**
- `cmd/ailang/docs_search.go` - CLI command (~200 LOC)
- `internal/docsearch/search.go` - SimHash search (~280 LOC)
- `internal/docsearch/embed.go` - Lazy embedding pipeline (~215 LOC)

**Key Design Decisions:**
1. Used standalone SimHash implementation (not SharedIndex builtins) for simplicity
2. Per-corpus cache files identified by SHA256 hash of corpus path
3. Content hash stored with each embedding for staleness detection
4. Progress indicator on stderr for long embedding operations

**Actual vs Estimated:**
- Estimated: 2-3 days (~12 hours)
- Actual: 0.75 days (~6 hours)
- LOC: 1005 total (vs estimated 840)

## Testing Strategy

**Unit tests:**
- Model mismatch detection
- Candidate bounding logic
- CAS update for frame embeddings

**Integration tests:**
- Full neural search pipeline
- Caching behavior (second search reuses embeddings)
- Model change triggers recompute

**Manual testing:**
- Verify stats output is accurate
- Test with various corpus sizes

## Non-Goals

**Not in this feature:**
- `ailang docs index` changes - Keep as "SimHash only" updater (fast, deterministic)
- Embedding the entire corpus upfront - By design, we avoid this
- Multiple embedding models simultaneously - One model at a time
- GPU/batch embedding optimization - Future work

## Design Questions to Resolve

1. **What content to embed?**
   - **Recommended:** Use extracted `DocSearchText` (same as SimHash) so both tiers align
   - Alternative: title + headings + summary only

2. **Should `ailang docs index` remain no-op for embeddings?**
   - **Recommended:** Yes - keep index as "SimHash only" (fast, deterministic, predictable)
   - Embeddings computed lazily on first `--neural` search

## Timeline

**Day 1** (6 hours):
- Phase 1: Core infrastructure
- Phase 2: Two-stage pipeline implementation

**Day 2** (4 hours):
- Phase 2: Complete and test
- Phase 3: Observability

**Day 3** (2 hours):
- Documentation and agent instruction update
- Final testing and polish

**Total: ~12 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Ollama not running | Med | Clear error message, fallback to SimHash-only results |
| Model name mismatch | Low | Store full model identifier, validate on load |
| Large candidate set slowness | Low | Bounded by default, configurable via flag |

## References

- [M-DX15 SimHash Builtins](../v0_5_11/m-dx15-simhash-builtins.md) - Foundation for Stage 1
- [Semantic Caching Design](../v0_6_0/semantic-caching-future.md) - Related caching concepts
- SharedIndex implementation in `internal/runtime/sharedindex/`

## Future Work

- Batch embedding via Ollama (reduce round trips)
- Multiple embedding model support
- Embedding precompute command (`ailang docs embed --all`)
- GPU acceleration for large corpora

---

**Document created**: 2025-12-16
**Last updated**: 2025-12-16
