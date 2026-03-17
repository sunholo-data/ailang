# Sprint Plan: M-BRAIN-VECTORS

## Summary

| Field | Value |
|-------|-------|
| **Sprint ID** | M-BRAIN-VECTORS |
| **Goal** | Add real embedding vectors to the brain, enabling cosine similarity search and machine-to-machine vector communication |
| **Duration** | 3 days (M-BRAIN shipped in 1 session; this is smaller but has external deps) |
| **Risk Level** | Medium (embedder integration depends on Ollama running locally; graceful fallback mitigates) |
| **Total LOC** | ~1,200 (implementation + tests) |
| **Design Doc** | [m-brain-vectors.md](m-brain-vectors.md) |

## Velocity Context

Recent velocity from M-BRAIN sprint:
- M1 (SQLite backend): ~700 LOC in ~45min
- M2 (CLI surface): ~614 LOC in ~30min
- M3 (Hooks): ~80 LOC in ~15min
- M4 (Docs): ~190 LOC in ~15min
- **Total: ~1,584 LOC in ~2 hours**

This sprint is similar in scope (~1,200 LOC) but involves external service integration (Ollama/Gemini), which adds latency for testing.

## Milestones

### M1: Schema Migration + Embedding Storage (~300 LOC)

**Goal**: Add embedding BLOB column to brain_frames, implement encode/decode, cosine scan.

**Tasks**:
1. Schema migration v1 → v2: add `embedding BLOB`, `embedding_dim INTEGER`, `embed_model TEXT`
2. `encodeEmbedding([]float32) []byte` / `decodeEmbedding([]byte) []float32` — IEEE 754 float32 LE
3. `PutFrame()` extended: accepts optional `embedding []float32`
4. `PutVector()` — embedding + opaque payload, no text content
5. `SearchByEmbedding()` — brute-force cosine scan over all frames with embeddings
6. `cosineSimilarity(a, b []float32) float64` helper
7. Tests: encode/decode round-trip, cosine accuracy, schema migration, mixed frames

**Acceptance Criteria**:
- Schema migrates non-destructively (v1 DB opens with v2 code, new columns nullable)
- Embedding encode/decode round-trips exactly (float32 precision)
- Cosine search returns correct top-K results with deterministic ordering
- PutVector stores frame with embedding but no content
- `make test` and `make lint` pass

**Files**:
- `internal/effects/sharedmem_sqlite.go` — schema migration, new methods
- `internal/effects/sharedmem_sqlite_test.go` — new tests

---

### M2: Embedder Wiring + Auto-Embed (~250 LOC)

**Goal**: Wire `messaging.Embedder` into brain, auto-embed on put.

**Tasks**:
1. `CacheOption` pattern: `WithEmbedder(e messaging.Embedder)`
2. `BrainStore` propagates embedder to both tiers
3. `PutFrame()` auto-embeds `content` when embedder present
4. CLI: `ailang cache put --embed` flag
5. CLI: `ailang cache embed [--namespace NS]` — backfill existing frames
6. Config: read `brain.embedding.provider` from config.yaml or env var
7. Tests: mock embedder, auto-embed verify, backfill, nil-embedder fallback

**Acceptance Criteria**:
- `ailang cache put --embed --content "text" key` stores embedding alongside SimHash
- `ailang cache embed` backfills frames that lack embeddings
- Brain works identically when no embedder configured (nil embedder)
- Embedder errors don't block frame storage (fallback to SimHash-only)
- `make test` and `make lint` pass

**Files**:
- `internal/effects/sharedmem_sqlite.go` — WithEmbedder option
- `internal/effects/brain.go` — embedder propagation
- `cmd/ailang/cache.go` — --embed flag, embed subcommand
- `internal/effects/sharedmem_sqlite_test.go` — mock embedder tests

---

### M3: Three-Tier Search Merge + CLI (~250 LOC)

**Goal**: Upgrade search to merge cosine + SimHash + text results.

**Tasks**:
1. `BrainStore.Search()` upgraded: cosine (if available) → SimHash → text, merged by score
2. Cosine results get +0.1 boost over SimHash-only results
3. CLI: `--cosine` and `--simhash` flags to force search path
4. Auto-detect: use cosine when >50% of frames have embeddings, else SimHash
5. `ailang cache put-vector` CLI — reads JSON from stdin
6. `ailang cache stats` shows embedding coverage (% with embeddings, model, dim)
7. `ailang cache export/import` handles embedding BLOB (base64 in JSONL)
8. Hook updates: brain_resolution.sh gains `--embed` when embedder available
9. Tests: three-tier merge ranking, boost correctness, export/import with embeddings

**Acceptance Criteria**:
- Cosine-backed results rank above SimHash-only for semantically similar queries
- `--cosine` forces embedding-only search, returns error if no embeddings
- `--simhash` forces SimHash-only (fast path)
- Stats shows "With embeddings: N (X%)"
- Export/import round-trips embeddings correctly
- `make test` and `make lint` pass

**Files**:
- `internal/effects/brain.go` — three-tier search merge
- `cmd/ailang/cache.go` — flags, put-vector, stats, export/import
- `scripts/hooks/brain_resolution.sh` — --embed flag
- `internal/effects/sharedmem_sqlite_test.go` — merge tests

---

### M4: Docs, CHANGELOG, Polish (~150 LOC)

**Goal**: Update user guide, CHANGELOG, verify everything works end-to-end.

**Tasks**:
1. Update `docs/docs/guides/brain-cache.md` with embedding sections
2. Update `changelogs/v0.9-current.md`
3. End-to-end smoke test: put with embed → search by cosine → verify ranking
4. Update sprint JSON: all milestones pass

**Acceptance Criteria**:
- User guide documents `--embed`, `--cosine`, `ailang cache embed`, `put-vector`
- CHANGELOG has M-BRAIN-VECTORS section
- `make test`, `make lint` pass
- Sprint JSON status = completed

**Files**:
- `docs/docs/guides/brain-cache.md`
- `changelogs/v0.9-current.md`

---

## Day-by-Day Plan

| Day | Milestones | Key Risk |
|-----|-----------|----------|
| 1 | M1 (schema + storage) + M2 (embedder wiring) | Schema migration edge cases |
| 2 | M3 (three-tier search, put-vector, stats) | Cosine ranking correctness |
| 3 | M4 (docs, polish) + buffer | None (documentation) |

## Success Metrics

- [ ] Cosine search finds semantically similar frames SimHash misses
- [ ] Brain works without embedder (graceful fallback)
- [ ] Schema migration is non-destructive
- [ ] Export/import round-trips embeddings
- [ ] All tests pass, lint clean
- [ ] User guide updated
