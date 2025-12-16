# M-MSG-SEMANTIC Sprint Plan: Semantic Search for AILANG Messages

**Sprint ID:** M-MSG-SEMANTIC
**Status:** COMPLETED
**Design Doc:** [m-msg-semantic-caching.md](./m-msg-semantic-caching.md)
**Target Version:** v0.5.11
**Estimated Duration:** 2-3 days (9-12 hours) → **Actual:** ~0.5 days
**Risk Level:** Low (builds on existing SimHash infrastructure from v0.5.11)

---

## Sprint Summary

**Goal:** Add semantic search capabilities to `ailang messages` CLI, enabling developers and agents to find messages by meaning, detect near-duplicates, and safely deduplicate shared inboxes.

**Key Deliverables:**
1. `ailang messages search "query"` - Find messages by semantic similarity
2. `ailang messages list --similar-to MSG_ID` - Find similar messages
3. `ailang messages dedupe --threshold 0.95` - Safe deduplication report/apply
4. Inbox inference from repo/config (reduce CLI friction)

**Prerequisites (all complete in v0.5.11):**
- `_simhash` builtin - SimHash algorithm for text fingerprinting
- `_hamming_distance` builtin - Compare SimHash values
- `_ollama_embed` builtin - Neural embeddings (optional, Phase 3)
- SharedIndex infrastructure - Accelerated similarity search

---

## Current Implementation Status

### Existing Infrastructure (Ready to Use)

| Component | File | LOC | Status |
|-----------|------|-----|--------|
| SimHash algorithm | `internal/builtins/simhash.go` | 257 | SimHash(), HammingDistance() |
| Messaging schema | `internal/messaging/schema.go` | 481 | inbox_messages table |
| Message CRUD | `internal/messaging/inbox.go` | 387 | Insert/List/Get/Update |
| Messages CLI | `cmd/ailang/messages*.go` | ~600 | send/list/read/ack/cleanup |

### Velocity Analysis (Last 7 Days)

Recent milestones show ~300-800 LOC per feature with 3-6 hour implementation times:
- M-DX11-PHASE2: Debug event wiring (~400 LOC, 4 hours)
- M-DX15 SimHash: SimHash builtins (~300 LOC, 3 hours)
- Unified AI providers: AI package consolidation (~1300 LOC, 8 hours)

**Conservative estimate:** 150 LOC/day for well-scoped changes

---

## Milestone Breakdown

### M1: Schema Migration + SimHash on Insert (~150 LOC, 2 hours)

**Files:**
- `internal/messaging/schema.go` (modify) - Add columns + migration
- `internal/messaging/inbox.go` (modify) - Compute simhash on insert

**Tasks:**
1. Add `simhash INTEGER` column to `inbox_messages` table
2. Add `dup_of TEXT NULL` column for safe deduplication
3. Create index: `CREATE INDEX idx_inbox_messages_simhash ON inbox_messages(simhash)`
4. Implement `migrateV110ToV112()` migration function
5. Modify `InsertInboxMessage()` to compute simhash from `title + " " + payload`
6. Backfill simhash for existing messages during migration

**Acceptance Criteria:**
- [ ] New columns exist after migration
- [ ] SimHash computed and stored on message insert
- [ ] Existing messages backfilled with simhash values
- [ ] Migration is idempotent (safe to run multiple times)
- [ ] `make test` passes

### M2: Search Implementation (~200 LOC, 3 hours)

**Files:**
- `internal/messaging/search.go` (NEW) - Semantic search functions

**Tasks:**
1. Define `SearchOptions` struct with Query, Threshold, Limit, Inbox, UseNeural fields
2. Define `SearchHit` struct with Message, Score, ScoreKind fields
3. Implement `SemanticSearch(opts SearchOptions) ([]SearchHit, error)`
   - Compute query simhash
   - Scan messages in target inbox (bounded by MaxScan)
   - Compute hamming distance, convert to similarity score: `1.0 - (dist/64.0)`
   - Filter by threshold, sort by score DESC, return top-K
4. Implement `FindSimilar(msgID string, threshold float64, limit int) ([]SearchHit, error)`
   - Look up message by ID
   - Use its simhash for similarity search
5. Add deterministic ordering: `(score DESC, message_id ASC)`

**Acceptance Criteria:**
- [ ] `SemanticSearch()` returns messages sorted by similarity
- [ ] `FindSimilar()` finds messages similar to a given message
- [ ] Threshold filtering works correctly
- [ ] Results are deterministically ordered
- [ ] Unit tests cover typical queries

### M3: Deduplication Report (~120 LOC, 2 hours)

**Files:**
- `internal/messaging/search.go` (extend) - Add deduplication functions

**Tasks:**
1. Define `DuplicateGroup` struct: Representative, Duplicates, MinScore, ScoreKind
2. Implement `FindDuplicates(inbox string, threshold float64) ([]DuplicateGroup, error)`
   - Scan all messages in inbox
   - Build clusters of messages with similarity >= threshold
   - Select representative: oldest message (by timestamp, then ID)
3. Implement `ApplyDuplicates(groups []DuplicateGroup, runID string) error`
   - For each duplicate: set `dup_of = representative.ID`, set status = "read"
   - Record `runID` for audit trail
4. Add `--collapsed` filter to `ListInboxMessages()` to hide duplicates

**Acceptance Criteria:**
- [ ] `FindDuplicates()` returns correct clusters
- [ ] Representative selection is deterministic (oldest message)
- [ ] `ApplyDuplicates()` marks duplicates without deleting
- [ ] Duplicates can be restored by clearing `dup_of`
- [ ] `--collapsed` hides messages where `dup_of IS NOT NULL`

### M4: CLI Integration (~200 LOC, 3 hours)

**Files:**
- `cmd/ailang/messages_search.go` (NEW) - Search command
- `cmd/ailang/messages_crud.go` (modify) - Add `--similar-to`, `--collapsed` flags
- `cmd/ailang/messages_dedupe.go` (NEW) - Dedupe command
- `cmd/ailang/messages_util.go` (modify) - Inbox inference helper

**Tasks:**
1. Add `ailang messages search "query"` subcommand
   - Flags: `--inbox`, `--threshold` (default 0.70), `--limit` (default 20)
   - Print results with similarity scores
   - Print explainability footer: `backend=SQLite mode=Strict score=simhash limit=N threshold=X`
2. Add `--similar-to MSG_ID` flag to `messages list`
3. Add `--collapsed` flag to hide duplicates
4. Add `ailang messages dedupe` subcommand
   - Flags: `--inbox`, `--threshold` (default 0.95), `--apply`, `--dry-run`
   - Default: report-only (show clusters without applying)
   - `--apply`: actually mark duplicates (role-gated to maintainer)
5. Implement inbox inference:
   - Check `--inbox` flag → use if provided
   - Check `.ailang/project.toml` for `project.inbox` → use if present
   - Use git repo root folder name → sanitize as inbox name
   - Fallback: "default"
   - Print: `Using inbox: X (inferred from Y)`

**Acceptance Criteria:**
- [ ] `messages search "query"` returns relevant messages
- [ ] `messages list --similar-to MSG_ID` works
- [ ] `messages dedupe` shows clusters without modifying data
- [ ] `messages dedupe --apply` marks duplicates
- [ ] Inbox inference works and is printed
- [ ] Explainability footer present on all semantic queries

---

## Testing Strategy

### Unit Tests (~100 LOC)

**File:** `internal/messaging/search_test.go` (NEW)

1. `TestSemanticSearch_FindsSimilarMessages` - Basic search
2. `TestSemanticSearch_ThresholdFiltering` - Low/high threshold behavior
3. `TestSemanticSearch_DeterministicOrdering` - Same query = same order
4. `TestFindSimilar_ByMessageID` - Find similar to existing message
5. `TestFindDuplicates_ClustersCorrectly` - Clustering accuracy
6. `TestApplyDuplicates_SetsDupOf` - Apply marks duplicates
7. `TestListWithCollapsed_HidesDuplicates` - Collapsed filter works

### Integration Tests (~50 LOC)

**File:** `internal/messaging/search_integration_test.go` (NEW)

1. Full workflow: insert → search → dedupe → verify
2. Migration test: backfill existing messages

### Manual Testing

```bash
# After implementation:
ailang messages send user "Parser fails on ADT patterns" --title "Parser bug"
ailang messages send user "ADT parsing causes crash" --title "Parser issue"
ailang messages search "parser ADT" --threshold 0.6
ailang messages list --similar-to <MSG_ID>
ailang messages dedupe --threshold 0.90 --dry-run
```

---

## Day-by-Day Implementation Plan

### Day 1 (4-5 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 1h | Schema migration (simhash + dup_of columns) | M1 |
| 1h | InsertInboxMessage simhash computation | M1 |
| 2h | SemanticSearch + FindSimilar implementation | M2 |
| 1h | Unit tests for search functions | M2 |

**Checkpoint:** `make test` passes, simhash stored on insert

### Day 2 (4-5 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 2h | FindDuplicates + ApplyDuplicates | M3 |
| 1h | Unit tests for deduplication | M3 |
| 2h | CLI: `messages search` + `--similar-to` | M4 |

**Checkpoint:** Search CLI works, dedupe report works

### Day 3 (2-3 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 1h | CLI: `messages dedupe` command | M4 |
| 1h | Inbox inference + explainability footer | M4 |
| 1h | Integration tests + final verification | M4 |

**Checkpoint:** All features working, ready for merge

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Search latency | <10ms | Time 10K message search |
| Relevance | Top-3 contains match | Manual testing |
| Dedupe FP rate | <5% | Test with known duplicates |
| Code quality | No lint warnings | `make lint` |
| Test coverage | 80%+ for new code | `go test -cover` |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| SimHash inaccurate for short messages | Medium | Low | Document threshold recommendations |
| Performance on large inboxes | Low | Medium | MaxScan parameter, future SharedIndex integration |
| Migration breaks existing messages | Low | High | Idempotent migration, backup recommendation |

---

## Dependencies

| Dependency | Status | Notes |
|------------|--------|-------|
| SimHash builtin | Complete | `internal/builtins/simhash.go` |
| HammingDistance builtin | Complete | `internal/builtins/simhash.go` |
| Messaging schema | Complete | `internal/messaging/schema.go` |
| Messages CLI | Complete | `cmd/ailang/messages*.go` |

---

## Future Work (Not in Scope)

- **Phase 3: Neural Search** - Embedding-based semantic search (optional, ~150 LOC)
- **SharedIndex acceleration** - Use SharedIndex for sub-ms search on large inboxes
- **SQLite FTS5** - Full-text keyword search as fallback
- **Cross-inbox search** - Search across all inboxes (requires scoping)

---

## Summary

| Phase | Scope | LOC | Time |
|-------|-------|-----|------|
| M1 | Schema + SimHash on insert | ~150 | 2h |
| M2 | Search implementation | ~200 | 3h |
| M3 | Deduplication report/apply | ~120 | 2h |
| M4 | CLI integration | ~200 | 3h |
| **Total** | | **~670** | **10h** |

**Recommended approach:** Complete all 4 milestones in v0.5.12. Phase 3 (Neural Search) deferred to v0.5.13 as experimental.
