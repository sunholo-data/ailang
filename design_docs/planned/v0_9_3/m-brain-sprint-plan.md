# Sprint Plan: M-BRAIN — AILANG Persistent Semantic Cache

## Summary

Wire up AILANG's existing semantic caching infrastructure (SharedMem/SharedIndex/sem_frame) with SQLite persistence and Claude Code hook integration, creating a knowledge base that accumulates intelligence across coding sessions.

**Duration:** 4 days (estimated 32h implementation + testing)
**Dependencies:** DX-15/16/17 (all ✅), M-PERF5 finalized (✅)
**Risk Level:** Medium (new SQLite backend + hook integration, but follows established patterns)
**Design Doc:** [m-brain.md](m-brain.md)

## Current Status Analysis

### Completed Infrastructure (v0.5.11–v0.8.1)
- ✅ SharedMem effect: 6 builtins, 12 tests — `internal/effects/sharedmem.go`
- ✅ SharedIndex effect: 7 builtins, 14 tests — `internal/effects/sharedindex.go`
- ✅ sem_frame: 568 LOC in `std/sem.ail` (store/load/search/CAS helpers)
- ✅ SimHash + hamming_distance: `internal/builtins/simhash.go`
- ✅ Ollama embeddings: `internal/builtins/ollama_embed.go`
- ✅ Messaging SQLite patterns: `internal/messaging/schema.go` (WAL, migrations, pragmas)
- ✅ Claude Code hooks: SessionStart, PreToolUse, PostToolUse all active in `.claude/settings.json`
- ✅ Semantic envelope: `internal/messaging/envelope.go` (5-slot system)

### Velocity (last 14 days)
- M-PERF5: ~310 LOC in 1 day (3 milestones, focused Go work)
- M-CLOUD-DUAL-AUTH: ~400 LOC in 1 day (design doc + implementation)
- Recent average: ~200-400 LOC/day for implementation work
- Estimated capacity: ~1,600 LOC across 4 days (conservative, includes testing)

### What We're Building On
- `SharedCache` interface already supports pluggable backends (just add SQLite impl)
- `messaging/schema.go` has exact SQLite patterns to copy (WAL, pragmas, migrations)
- Hook infrastructure already active (3 hook types wired in `.claude/settings.json`)
- `ailang cache` CLI follows same patterns as `ailang messages` commands

## Proposed Milestones

### M1: SQLite Backend for SharedMem
**Goal:** Persistent `SharedCache` implementation with search capabilities
**Estimated:** ~400 LOC implementation + ~350 LOC tests = ~750 LOC
**Duration:** 1.5 days

**Tasks:**
- Create `internal/effects/sharedmem_sqlite.go`:
  - `SQLiteSharedCache` struct implementing `SharedCache` interface
  - `brain_frames` table: key, namespace, value, simhash, content, version, timestamps, TTL, source
  - FTS5 virtual table for keyword search
  - `NewSQLiteSharedCache(dbPath)` constructor with WAL mode + messaging pragmas
  - Extended methods: `SearchBySimHash()`, `SearchByText()`, `GarbageCollect()`, `Stats()`
- Create `internal/effects/brain.go`:
  - `BrainStore` struct wrapping two `SQLiteSharedCache` instances (user + project)
  - `NewBrainStore(userDBPath, projectDBPath)` — opens both, project may be nil
  - `Search()` queries both, merges results with project-local boost
  - `Put()` writes to project by default, user with `scope` option
  - `Promote(key)` copies frame from project → user brain
- Create `internal/effects/sharedmem_sqlite_test.go`:
  - Full `SharedCache` interface compliance (Get/Put/CAS/Delete/Keys/Len)
  - SimHash search correctness + ordering
  - FTS5 keyword search
  - GC with TTL expiry
  - Concurrent read/write stress test
  - Two-tier merge: project results ranked above user results
- Wire backend selection in `cmd/ailang/run_helpers.go`:
  - Check `~/.ailang/config.yaml` for `brain.backend` setting
  - User brain: `~/.ailang/state/brain.db`
  - Project brain: `.ailang/state/brain.db` (relative to repo root)
  - Fallback: in-memory (existing behavior) when config absent

**Acceptance Criteria:**
- [ ] `SQLiteSharedCache` passes all existing `SharedCache` interface tests
- [ ] SimHash search returns top-K results with deterministic ordering
- [ ] FTS5 keyword search returns relevant results
- [ ] GC removes expired frames based on TTL
- [ ] Concurrent stress test passes (100 goroutines, following `sharedmem_test.go` pattern)
- [ ] Data persists across process restarts
- [ ] `make test` and `make lint` pass

**Risks:**
- CGO requirement for go-sqlite3 — already a dependency via messaging
- Schema mismatch with existing in-memory interface — keep interface identical, extend with composition

---

### M2: CLI Surface (`ailang cache`)
**Goal:** Human-friendly CLI for searching, browsing, and managing the brain
**Estimated:** ~500 LOC implementation + ~100 LOC tests = ~600 LOC
**Duration:** 1 day

**Tasks:**
- Create `cmd/ailang/cache.go` with subcommands:
  - `ailang cache search "query"` — both brains, SimHash + FTS5, optional `--neural`
  - `ailang cache search --context FILE1,FILE2` — find by file overlap
  - `ailang cache search --scope user|project` — filter to one brain tier
  - `ailang cache list --recent --limit N` — browse recent entries (both brains)
  - `ailang cache list --namespace NAME` — browse by namespace
  - `ailang cache show KEY` — full frame detail
  - `ailang cache put --ns NAME --content "text"` — write to project brain (default)
  - `ailang cache put --scope user --ns patterns --content "..."` — write to user brain
  - `ailang cache put-resolution --commit-msg "..." --files "..."` — resolution frame
  - `ailang cache promote KEY` — copy frame from project → user brain
  - `ailang cache stats` — namespace counts per brain tier, storage sizes
  - `ailang cache gc --older-than 30d` — garbage collect (both brains)
  - `ailang cache export > backup.jsonl` / `ailang cache import < backup.jsonl`
- Register in `cmd/ailang/main.go`
- Explainability footer on search results (brain tier, backend, mode, timing)

**Acceptance Criteria:**
- [ ] `ailang cache search` returns relevant results in <100ms
- [ ] `ailang cache list --recent` shows newest frames
- [ ] `ailang cache put` stores frame with correct namespace and SimHash
- [ ] `ailang cache put-resolution` creates resolution frame from commit info
- [ ] `ailang cache stats` shows namespace breakdown and storage size
- [ ] `ailang cache gc` removes expired frames
- [ ] `ailang cache export | ailang cache import` round-trips cleanly
- [ ] `make test` and `make lint` pass

**Risks:**
- CLI scope creep — keep MVP: search, put, list, stats, gc. Defer fancy features.

---

### M3: Hook Integration
**Goal:** Automatic knowledge capture during coding and context injection at session start
**Estimated:** ~150 LOC (hooks + config) + ~50 LOC tests = ~200 LOC
**Duration:** 1 day

**Tasks:**
- Create `scripts/hooks/brain_session_start.sh`:
  - Get recent git changes (`git diff HEAD~3..HEAD --name-only`)
  - Query brain via `ailang cache search --context <files> --limit 5 --format summary`
  - Output concise summary for system reminder injection
  - 2-second timeout to avoid blocking session start
- Create `scripts/hooks/brain_resolution.sh`:
  - Detect `git commit` in Bash tool use
  - Extract commit message + diff summary + changed files
  - Call `ailang cache put-resolution` async (background `&`)
- Create `scripts/hooks/brain_capture.sh`:
  - On Write/Edit tool use, note file path
  - Lightweight async context capture (queue, don't block)
- Update `.claude/settings.json`:
  - Add brain hooks alongside existing hooks
  - Brain hooks should be additive, not replace existing
- Add `brain.enabled` config toggle in `~/.ailang/config.yaml`

**Acceptance Criteria:**
- [ ] Session start hook injects relevant brain context (top-3 frames)
- [ ] Post-commit hook captures resolution frames automatically
- [ ] Hooks are async — don't add >200ms to any tool call
- [ ] Brain hooks can be disabled via config (`brain.enabled: false`)
- [ ] Hooks work when brain.db doesn't exist yet (graceful bootstrap)
- [ ] `make test` and `make lint` pass

**Risks:**
- Hook fragility on different platforms — keep scripts minimal (5-10 lines), test on macOS
- Noisy session start output — cap at 3 frames, concise format, skip if empty

---

### M4: Polish, Docs & Config
**Goal:** Configuration, documentation, example file, CHANGELOG
**Estimated:** ~120 LOC
**Duration:** 0.5 day

**Tasks:**
- Add `brain:` section to `~/.ailang/config.yaml` support:
  ```yaml
  brain:
    enabled: true
    backend: sqlite
    sqlite:
      path: ~/.ailang/state/brain.db
    ttl:
      resolutions: 0        # permanent
      code-context: 30d
      session: 90d
      ephemeral: 7d
  ```
- Create `docs/docs/guides/brain-cache.md` — user guide
- Create `examples/reference/brain_demo.ail` — using persistent SharedMem
- Update `CHANGELOG.md` with M-BRAIN feature
- Update `docs/docs/guides/semantic-search.md` to reference brain
- Update design doc status to Implemented

**Acceptance Criteria:**
- [ ] Config file controls backend, TTLs, enabled state
- [ ] User guide covers search, capture, hooks, cloud upgrade path
- [ ] Example file runs successfully with `--caps IO,SharedMem`
- [ ] CHANGELOG.md updated
- [ ] Design doc moved to implemented/

---

## Success Metrics

| Metric | Target |
|--------|--------|
| `ailang cache search` latency | <100ms on 1K frames |
| SharedCache interface compliance | All existing tests pass |
| Hook latency overhead | <200ms per tool call |
| Session start context injection | Top-3 relevant frames |
| Resolution auto-capture | Works on every `git commit` |
| Test coverage (new code) | >85% |
| Total LOC | ~1,670 |

## Dependencies

- `go-sqlite3` — already a dependency (used by messaging)
- Ollama — optional, only for `--neural` search
- Claude Code hooks — already configured and active

## Open Questions

1. **Namespace defaults** — Should resolution frames include the project name in the namespace? e.g., `resolutions:ailang` vs just `resolutions`? (Matters for future cross-project brains)
2. **Session start format** — How verbose should the brain context injection be? Current proposal: 3-line summaries per frame, max 5 frames.
3. **Embedding auto-compute** — Should `ailang cache put` auto-compute embeddings when Ollama is available? Or keep it opt-in via `--embed`?

## Notes

- M-PERF5 must be fully finalized (M4 benchmark) before starting M-BRAIN
- This sprint focuses on the **daily workflow** use case; multi-agent and cloud are future work
- The SQLite backend is designed for single-developer use; Firestore upgrade is a separate sprint
- Hook scripts should be simple enough to debug with `bash -x`
