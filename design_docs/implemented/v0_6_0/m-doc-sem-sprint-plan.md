# Sprint Plan: M-DOC-SEM Lazy Embeddings for Neural Doc Search

## Summary

Implement lazy neural embeddings for `ailang docs search --neural` that bounds compute to SimHash shortlist candidates only, with model-versioned caching.

**Duration:** 2-3 days
**Dependencies:** SharedIndex builtins (COMPLETE), SimHash builtins (COMPLETE)
**Risk Level:** Low (infrastructure exists, wiring new CLI)

## Current Status Analysis

### Completed Recently
- ✅ SharedIndex builtins: `_sharedindex_upsert_emb`, `_sharedindex_find_by_embedding` (~660 LOC)
- ✅ SimHash builtins: `_simhash`, `_hamming_distance` (~100 LOC)
- ✅ Ollama embedding builtin: `_ollama_embed` (already exists)

### Velocity
- Recent average: ~200 LOC/day from M-DX15 work
- Estimated capacity: ~500 LOC for this sprint

### Remaining from Design Doc
- ⏳ M1: Frame metadata for embedding model/version (~50 LOC)
- ⏳ M2: CLI `docs search` command (~150 LOC)
- ⏳ M3: Two-stage pipeline (SimHash → lazy embed → search) (~250 LOC)
- ⏳ M4: Observability stats (~50 LOC)
- 📋 Tests (~200 LOC)

## Proposed Milestones

### Milestone 1: CLI Foundation + Frame Metadata
**Goal:** Add `ailang docs search` CLI command skeleton and frame embedding metadata
**Estimated:** 100 LOC implementation + 50 LOC tests = 150 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `cmd/ailang/docs_search.go` with basic subcommand routing
- Add `--stream` flag (planned/implemented/all)
- Add `--neural` flag (triggers embedding path)
- Add `--neural-candidates` flag (default: `max(200, 20*limit)`)
- Define frame embedding metadata struct: `embedding_model`, `embedding_updated_at`

**Acceptance Criteria:**
- [ ] `ailang docs search "query"` runs without error (returns no results yet)
- [ ] `ailang docs search --help` shows all flags
- [ ] Frame metadata struct defined with JSON tags
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- None (straightforward CLI wiring)

### Milestone 2: SimHash Shortlist (Stage 1)
**Goal:** Implement fast SimHash-based candidate filtering using existing SharedIndex
**Estimated:** 100 LOC implementation + 50 LOC tests = 150 LOC
**Duration:** 0.5 days

**Tasks:**
- Index design docs into SharedIndex namespace "design_docs"
- Compute query SimHash from search query
- Call `_sharedindex_find_simhash` with bounded `topK`
- Return shortlist with keys (file paths)

**Acceptance Criteria:**
- [ ] `ailang docs search "parser"` returns SimHash matches
- [ ] Results bounded by `--neural-candidates` (or default)
- [ ] Namespace "design_docs" populated on first search (lazy index)
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Index initialization timing - Mitigation: lazy init on first search

### Milestone 3: Lazy Embedding Pipeline (Stage 2-3)
**Goal:** Embed only missing candidates, then run cosine similarity search
**Estimated:** 150 LOC implementation + 80 LOC tests = 230 LOC
**Duration:** 1 day

**Tasks:**
- For each shortlist candidate:
  - Load from SharedMem (or read file)
  - Check if embedding exists AND model matches
  - If missing/mismatch: call `_ollama_embed(model, content)`
  - Store updated embedding with metadata
- Compute query embedding once
- Call `_sharedindex_find_by_embedding` for final ranking
- Format and display results

**Acceptance Criteria:**
- [ ] `ailang docs search "parser" --neural` uses embedding search
- [ ] Second search reuses cached embeddings (0 computed)
- [ ] Model change triggers re-embedding
- [ ] Results sorted by cosine similarity score
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Ollama not running - Mitigation: Clear error, fallback to SimHash-only

### Milestone 4: Observability + Polish
**Goal:** Add stats output and finalize UX
**Estimated:** 50 LOC implementation + 20 LOC tests = 70 LOC
**Duration:** 0.5 days

**Tasks:**
- Print: `neural_candidates=N embedded_now=K reused=N-K`
- Print: `model: <model_name>`
- Add `--json` output format for scripting
- Handle edge cases (empty corpus, no matches)
- Update CLAUDE.md agent instruction block

**Acceptance Criteria:**
- [ ] Stats printed on neural search
- [ ] `--json` returns structured output
- [ ] Error messages helpful (Ollama not running, etc.)
- [ ] Agent instruction block updated
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- None (polish work)

## Success Metrics
- Test coverage: >70% for new code
- Examples: Add `examples/docs_search.ail` showing usage
- Documentation: Update CLAUDE.md with agent guidance
- All tests passing: ✅
- All linting passing: ✅

## Dependencies
- SharedIndex effect enabled (already implemented)
- Ollama running locally (for --neural)
- Design docs exist in `design_docs/` (they do)

## Open Questions
1. **What content to embed?** - Recommend: Use full doc content (same as SimHash input)
2. **Keep `ailang docs index` separate?** - Recommend: Yes, lazy init from search

## Files to Create/Modify

**New files:**
- `cmd/ailang/docs_search.go` - CLI command (~200 LOC)
- `cmd/ailang/docs_search_test.go` - Tests (~150 LOC)
- `internal/docsearch/search.go` - Search logic (~200 LOC)
- `internal/docsearch/embed.go` - Lazy embedding (~100 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Route `docs search` subcommand (~5 LOC)
- `CLAUDE.md` - Agent instruction update (~10 LOC)

## Notes
- All embedding infrastructure exists via SharedIndex builtins
- This is primarily CLI wiring + orchestration logic
- Lazy approach means first search may be slower (building index)
- Conservative estimate: 2-3 days actual work

---

**Document created**: 2025-12-16
**Last updated**: 2025-12-16
