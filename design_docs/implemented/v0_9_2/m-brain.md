# M-BRAIN: AILANG Persistent Semantic Cache ("The Brain")

**Status**: Planned
**Target**: v0.9.3
**Priority**: P1 (High — foundational for daily workflow value)
**Estimated**: 8 days (32h implementation + 12h testing + 4h docs + buffer)
**Dependencies**: DX-15/16/17 (SharedMem/SharedIndex/sem_frame — all ✅ implemented v0.5.11)
**Milestone ID**: M-BRAIN
**Created**: 2026-03-13

---

## Problem Statement

AILANG has a complete semantic caching infrastructure (SharedMem, SharedIndex, sem_frame) built across DX-15/16/17, but it's **unused in practice**. Only 2 demo files reference it. The cache is in-memory-only, so all knowledge is lost on restart. No workflow integrates it.

**Current State:**
- SharedMem/SharedIndex: fully implemented, 13 builtins, 65+ tests — **zero real consumers**
- sem_frame: 568 LOC of high-level operations in `std/sem.ail` — **only imported by 2 example files**
- In-memory backend only: all cached knowledge lost on process exit
- No CLI for the cache: only accessible by writing `.ail` programs with `--caps`
- No hook integration: sessions start cold every time, nothing captured during work
- M-SEMANTIC-ENVELOPE's resolution feedback loop: designed but never wired up

**Impact:**
- Every Claude Code session starts from zero context (except MEMORY.md and CLAUDE.md)
- Bug fixes, architectural decisions, file relationships — all rediscovered each session
- The "system gets smarter with use" vision (M-SEMANTIC-ENVELOPE §4) remains unrealized
- Cloud deployment has no shared knowledge layer between agent sessions

---

## Goals

**Primary Goal:** Make AILANG's semantic cache a living knowledge base that accumulates intelligence across coding sessions, starting with SQLite persistence and Claude Code hook integration.

**Success Metrics:**
1. `ailang cache search "type inference bug"` returns relevant prior knowledge in <100ms
2. Session start hook injects top-3 relevant cached frames based on recent git diff
3. Post-commit hook auto-captures resolution frames (problem → solution pairs)
4. Cache persists across sessions (SQLite at `~/.ailang/state/brain.db`)
5. Same `SharedCache` interface works for both in-memory and SQLite backends
6. Cloud upgrade path: SQLite → Firestore requires only backend swap, no API changes

---

## Non-Goals

**Not in this feature:**
- Redis backend — SQLite is sufficient for single-developer; cloud uses Firestore (deferred)
- Multi-agent shared cache — focus on single-developer workflow first
- Full M-SEMANTIC-ENVELOPE (5-slot envelopes) — start with intent + resolution slots only
- ANN/vector database — SQLite scan is fine for <100K frames
- In-process embedder (llama.cpp/MLX) — Ollama works, optional dependency
- Custom embedding models — use off-the-shelf (EmbeddingGemma, nomic-embed-text)
- Fixed-size vector type (`vector[float; 768]`) — language feature, separate design doc

---

## Solution Design

### Overview

Three layers, each independently useful:

1. **SQLite Backend** — Persistent `SharedCache` implementation following `internal/messaging/schema.go` patterns
2. **CLI Surface** — `ailang cache` commands for humans to query/browse/manage the brain
3. **Hook Integration** — Claude Code hooks that capture knowledge during work and inject it at session start

### Two-Tier Scoping (User + Project)

Following the same pattern as Claude Code settings (`~/.claude/` + `.claude/`) and `ailang messages` (global DB, project-scoped inboxes), the brain operates at two levels:

| Level | Location | Contains | Scope |
|-------|----------|----------|-------|
| **User** | `~/.ailang/state/brain.db` | Cross-project knowledge: Go patterns, debugging techniques, personal conventions | All projects |
| **Project** | `.ailang/state/brain.db` | Project-specific: architecture decisions, bug history, file invariants, resolutions | This repo only |

**Query behavior:**
- **Default**: both levels queried, results merged (project ranked higher via boost)
- `--scope project`: project brain only
- `--scope user`: user brain only

**Write behavior:**
- Hooks write to **project** brain by default (resolutions, code-context are project-specific)
- `--scope user` flag on `ailang cache put` writes to user brain (cross-project learnings)
- CLI `ailang cache promote KEY` copies a frame from project → user brain

**Why two tiers:**
- Matches existing patterns (`ailang messages` inboxes, Claude Code settings hierarchy)
- Namespaces align with `ailang messages`, skills, eval results → enables future joins
- Cloud upgrade: project brain → Firestore collection per project; user brain stays local or syncs separately

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Claude Code Hooks                               │
│                                                                     │
│  SessionStart           PostToolUse(Write/Edit)     PostToolUse     │
│  ┌──────────────┐      ┌──────────────────────┐   (Bash:git commit)│
│  │ Query BOTH    │      │ Capture file context  │   ┌─────────────┐ │
│  │ brains for    │      │ to PROJECT brain      │   │ Capture      │ │
│  │ prior context │      │ (lightweight, async)  │   │ resolution   │ │
│  └──────┬───────┘      └──────────┬───────────┘   │ to PROJECT   │ │
│         │                         │                 └──────┬──────┘ │
└─────────┼─────────────────────────┼────────────────────────┼────────┘
          │                         │                        │
          ▼                         ▼                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     ailang cache CLI                                 │
│  ailang cache search "query"          ailang cache list --recent    │
│  ailang cache put --scope user        ailang cache promote KEY      │
│  ailang cache gc --older-than 30d     ailang cache export/import   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                     ┌─────────┴─────────┐
                     ▼                   ▼
┌────────────────────────────┐ ┌──────────────────────────────────────┐
│   USER Brain               │ │   PROJECT Brain                      │
│   ~/.ailang/state/brain.db │ │   .ailang/state/brain.db             │
│   Cross-project knowledge  │ │   Project-specific knowledge         │
│   Go patterns, conventions │ │   Resolutions, file context, arch    │
└────────────────────────────┘ └──────────────────────────────────────┘
                     │                   │
                     └─────────┬─────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│              SharedCache Interface (unchanged)                       │
│  Get(key) / Put(key, value) / CAS(key, old, new) / Delete / Keys   │
├─────────────────────────┬───────────────────────────────────────────┤
│  InMemorySharedCache    │  SQLiteSharedCache (NEW)                  │
│  (existing, for tests   │  WAL mode, same pragmas as messaging     │
│   and ephemeral use)    │  + SimHash column + embedding column      │
│                         │  + namespace + TTL + created_at           │
└─────────────────────────┴───────────────────────────────────────────┘
```

### The SharedCache Interface (unchanged)

The existing `SharedCache` interface in `internal/effects/sharedmem.go` already supports pluggable backends:

```go
type SharedCache interface {
    Get(key string) ([]byte, bool)
    Put(key string, value []byte)
    Delete(key string)
    CAS(key string, oldValue, newValue []byte) bool
    Keys() []string
    Len() int
}
```

We add a new SQLite implementation alongside the existing `InMemorySharedCache`.

### SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS brain_frames (
    key         TEXT PRIMARY KEY,
    namespace   TEXT NOT NULL DEFAULT 'default',
    value       BLOB NOT NULL,              -- JSON-encoded sem_frame
    simhash     INTEGER,                    -- 64-bit SimHash for fast search
    embedding   BLOB,                       -- packed float32 (optional)
    embed_model TEXT,                        -- e.g. "ollama:embeddinggemma"
    embed_dim   INTEGER DEFAULT 0,
    content     TEXT,                        -- searchable text (for FTS5)
    version     INTEGER DEFAULT 1,
    created_at  INTEGER NOT NULL,           -- unix millis
    updated_at  INTEGER NOT NULL,           -- unix millis
    expires_at  INTEGER,                    -- unix millis (NULL = no expiry)
    source      TEXT                         -- "hook:commit", "hook:edit", "cli", "agent"
);

CREATE INDEX idx_brain_ns ON brain_frames(namespace);
CREATE INDEX idx_brain_simhash ON brain_frames(namespace, simhash);
CREATE INDEX idx_brain_updated ON brain_frames(updated_at DESC);
CREATE INDEX idx_brain_expires ON brain_frames(expires_at) WHERE expires_at IS NOT NULL;

-- FTS5 for keyword search (complements SimHash/embedding similarity)
CREATE VIRTUAL TABLE IF NOT EXISTS brain_frames_fts USING fts5(
    key, namespace, content,
    content=brain_frames,
    content_rowid=rowid
);
```

### Hook Integration Points

**1. SessionStart Hook — "What do I know about this codebase right now?"**

```bash
# scripts/hooks/brain_session_start.sh
# Runs at the start of every Claude Code session

# Get recent git changes as context signal
RECENT_DIFF=$(git diff HEAD~3..HEAD --stat 2>/dev/null | head -20)
RECENT_FILES=$(git diff HEAD~3..HEAD --name-only 2>/dev/null)

# Query the brain for relevant prior knowledge
ailang cache search --context "$RECENT_FILES" --limit 5 --format summary

# Output injected into system reminders (like current session_start.sh)
```

Output example:
```
━━━ BRAIN: Relevant Prior Knowledge ━━━
1. [2d ago] internal/types/unify.go — "TypeVar unification requires
   occurs check; missing case caused infinite loop in v0.8.1"
2. [5d ago] internal/parser/parser_effect.go — "Effect tracking order
   matters: SharedMem must register before SharedIndex"
3. [1w ago] Bug resolution: "Record field access codegen panic fixed
   by adding nil check in emitFieldAccess (commit abc123)"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**2. PostToolUse Hook (Write/Edit) — "Remember what I learned about this file"**

Lightweight capture — when files are edited, note the file path and a summary. This doesn't need to be synchronous or blocking; it can queue for async processing.

```bash
# scripts/hooks/brain_capture.sh
# Triggered on Write/Edit tool use

FILE_PATH="$TOOL_INPUT_FILE_PATH"
if [ -n "$FILE_PATH" ]; then
    # Async: queue file context for brain storage
    ailang cache put-context --file "$FILE_PATH" --source "hook:edit" --async &
fi
```

**3. PostToolUse Hook (Bash: git commit) — "Remember the problem→solution"**

This is the resolution feedback loop from M-SEMANTIC-ENVELOPE:

```bash
# scripts/hooks/brain_resolution.sh
# Triggered on Bash tool use containing "git commit"

if echo "$TOOL_INPUT_COMMAND" | grep -q "git commit"; then
    COMMIT_MSG=$(git log -1 --format=%B 2>/dev/null)
    DIFF_SUMMARY=$(git diff HEAD~1..HEAD --stat 2>/dev/null)
    FILES_CHANGED=$(git diff HEAD~1..HEAD --name-only 2>/dev/null)

    ailang cache put-resolution \
        --commit-msg "$COMMIT_MSG" \
        --diff-summary "$DIFF_SUMMARY" \
        --files "$FILES_CHANGED" \
        --source "hook:commit" &
fi
```

### CLI Commands

```bash
# Search (both brains by default, merged results)
ailang cache search "type inference"              # SimHash + FTS5 search
ailang cache search "type inference" --neural      # Neural embedding search
ailang cache search --context FILE1,FILE2          # Find knowledge about these files
ailang cache search "Go patterns" --scope user     # User brain only
ailang cache search "parser bug" --scope project   # Project brain only

# Browse
ailang cache list --recent --limit 10              # Recent entries (both brains)
ailang cache list --namespace resolutions           # Browse resolutions
ailang cache show FRAME_KEY                         # Full frame detail

# Write (project brain by default)
ailang cache put "lesson learned" --ns learnings --content "Always check occurs..."
ailang cache put "Go tip" --ns patterns --scope user --content "Use sync.Pool for..."
ailang cache put-file internal/types/unify.go --ns code-notes --content "Key invariant..."

# Promote project knowledge to user brain (cross-project reuse)
ailang cache promote FRAME_KEY                     # Copy project frame → user brain

# Maintenance
ailang cache stats                                  # Counts per brain + namespace
ailang cache gc --older-than 90d --namespace ephemeral  # Garbage collect
ailang cache export > brain-backup.jsonl            # Export for cloud sync
ailang cache import < brain-backup.jsonl            # Import from backup
```

### Namespace Convention

Namespaces are shared across both brain tiers (same schema, same names). This enables future joins with `ailang messages` inboxes and skill metadata.

| Namespace | Default Tier | Source | Content | TTL |
|-----------|-------------|--------|---------|-----|
| `resolutions` | project | hook:commit | Problem → solution pairs | Permanent |
| `code-context` | project | hook:edit | File relationships, invariants | 30 days |
| `learnings` | project | cli/manual | Developer notes, project gotchas | Permanent |
| `patterns` | user | cli/manual | Cross-project patterns (Go, debugging) | Permanent |
| `session` | project | hook:session | Session summaries | 90 days |
| `ephemeral` | project | any | Temporary working memory | 7 days |

### Cloud Upgrade Path

The `SharedCache` interface is backend-agnostic. For cloud:

```go
// Future: internal/effects/sharedmem_firestore.go
type FirestoreSharedCache struct {
    client     *firestore.Client
    collection string
}

// Same interface, different storage
func (f *FirestoreSharedCache) Get(key string) ([]byte, bool) { ... }
func (f *FirestoreSharedCache) Put(key string, value []byte)  { ... }
func (f *FirestoreSharedCache) CAS(key, old, new []byte) bool { ... }
```

Configuration:
```yaml
# ~/.ailang/config.yaml (user-level defaults)
brain:
  backend: sqlite           # or "firestore" for cloud
  user_db: ~/.ailang/state/brain.db
  # project_db auto-resolved: .ailang/state/brain.db (relative to repo root)

  firestore:                # cloud upgrade (future)
    project_id: my-project
    collection: ailang_brain
```

**Two-tier in cloud:** Project brain → Firestore collection per project (shared by all cloud agents working on that project). User brain stays local or syncs to a personal Firestore collection. This matches how `ailang messages` already scopes inboxes per project.

---

## Implementation Plan

### Phase 1: SQLite Backend (~12h)

- [ ] Create `internal/effects/sharedmem_sqlite.go` — `SQLiteSharedCache` implementing `SharedCache`
- [ ] Schema creation with WAL mode, pragmas matching `messaging/schema.go` patterns
- [ ] `brain_frames` table with simhash, embedding, namespace, TTL, source columns
- [ ] FTS5 virtual table for keyword search
- [ ] Migration support (version tracking for future schema changes)
- [ ] `NewSQLiteSharedCache(dbPath string) (*SQLiteSharedCache, error)` constructor
- [ ] Extended interface: `SearchBySimHash()`, `SearchByText()`, `GarbageCollect()`, `Stats()`
- [ ] Wire into `run_helpers.go`: when `SharedMem` cap requested, check config for backend preference
- [ ] Unit tests: all `SharedCache` interface methods + SQLite-specific search + GC
- [ ] Stress tests: concurrent read/write (following `sharedmem_test.go` patterns)

### Phase 2: CLI Surface (~8h)

- [ ] Create `cmd/ailang/cache.go` — `ailang cache` subcommand with search/list/show/put/stats/gc/export/import
- [ ] `ailang cache search` — SimHash + optional neural + FTS5 keyword, with explainability footer
- [ ] `ailang cache search --context FILE1,FILE2` — find frames related to given files
- [ ] `ailang cache list` — browse by namespace, recency, source
- [ ] `ailang cache put` / `ailang cache put-resolution` / `ailang cache put-context` — manual and hook-driven capture
- [ ] `ailang cache stats` — namespace counts, storage size, last activity
- [ ] `ailang cache gc` — TTL-based + age-based garbage collection
- [ ] `ailang cache export/import` — JSONL format for backup and cloud sync
- [ ] Integration tests for CLI commands

### Phase 3: Hook Integration (~8h)

- [ ] Create `scripts/hooks/brain_session_start.sh` — query brain for relevant context on session start
- [ ] Create `scripts/hooks/brain_capture.sh` — lightweight async capture on file edit
- [ ] Create `scripts/hooks/brain_resolution.sh` — capture commit as resolution frame
- [ ] Update `.claude/settings.json` — register new hooks alongside existing ones
- [ ] `ailang cache search --context` smart ranking: weight by file overlap, recency, namespace
- [ ] Session start output format: concise, actionable summaries (not raw frames)
- [ ] Async capture: non-blocking hooks that don't slow down the coding flow
- [ ] End-to-end test: edit file → commit → new session → verify context injected

### Phase 4: Polish & Docs (~4h)

- [ ] Configuration in `~/.ailang/config.yaml` for brain backend, TTL defaults, hook enablement
- [ ] CHANGELOG.md update
- [ ] User guide: `docs/docs/guides/brain-cache.md`
- [ ] Update `docs/docs/guides/semantic-search.md` to reference brain cache
- [ ] Example: `examples/reference/brain_demo.ail` using persistent cache
- [ ] Update CLAUDE.md with brain cache section

---

## Files to Create/Modify

**New files:**
| File | LOC Est. | Purpose |
|------|----------|---------|
| `internal/effects/sharedmem_sqlite.go` | ~400 | SQLite SharedCache implementation |
| `internal/effects/sharedmem_sqlite_test.go` | ~350 | Unit + stress tests |
| `cmd/ailang/cache.go` | ~500 | CLI subcommand |
| `scripts/hooks/brain_session_start.sh` | ~40 | Session start hook |
| `scripts/hooks/brain_capture.sh` | ~30 | Edit capture hook |
| `scripts/hooks/brain_resolution.sh` | ~40 | Commit resolution hook |
| `docs/docs/guides/brain-cache.md` | ~200 | User guide |

**Modified files:**
| File | Changes | LOC Est. |
|------|---------|----------|
| `internal/effects/sharedmem.go` | Add extended search interface | +30 |
| `cmd/ailang/main.go` | Register cache subcommand | +10 |
| `cmd/ailang/run_helpers.go` | Backend selection from config | +20 |
| `.claude/settings.json` | Register brain hooks | +30 |
| `CHANGELOG.md` | Document feature | +20 |

**Total:** ~1,670 LOC new/modified

---

## Examples

### Example 1: Daily Workflow

**Before (every session starts cold):**
```
$ claude
> Read internal/types/unify.go
> [spends 5 minutes re-understanding the occurs check]
> [rediscovers the same bug pattern from last week]
```

**After (brain injects relevant context):**
```
$ claude
━━━ BRAIN: Relevant Prior Knowledge ━━━
1. [3d ago] unify.go — "Occurs check prevents infinite types.
   The recursive case on line 142 must check both TVar2 and TypeApp."
2. [1w ago] Resolution: "Fixed panic in record field codegen by
   adding nil guard at emitFieldAccess:89 (commit b03d79d5)"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Example 2: Searching the Brain

```bash
$ ailang cache search "record type codegen panic"
  1. [resolution] Record field access nil panic (score: 0.89)
     Fixed by nil guard in emitFieldAccess. Commit b03d79d5.
  2. [code-context] internal/codegen/records.go (score: 0.74)
     Key invariant: all record fields must be checked for nil before emit.
  3. [learning] Codegen record patterns (score: 0.68)
     Record types need both field access and update codegen paths.

backend=SQLite mode=SimHash+FTS5 results=3 query_ms=12
```

### Example 3: Resolution Auto-Capture

```bash
# Developer fixes a bug and commits
$ git commit -m "Fix: TypeVar unification missing occurs check

The recursive unification of TypeVar with TypeApp was not checking
for occurs (infinite type). Added occurs check at unify.go:142."

# Hook fires automatically, captures:
# - Namespace: resolutions
# - Content: commit message + file list
# - SimHash: computed from message text
# - Files: internal/types/unify.go
# - Source: hook:commit
```

---

## Success Criteria

- [ ] `ailang cache search` returns relevant results from SQLite in <100ms
- [ ] SQLite backend passes all existing `SharedCache` interface tests
- [ ] Session start hook injects relevant context (top-3 frames by file overlap + recency)
- [ ] Post-commit hook captures resolution frames automatically
- [ ] Cache persists across process restarts (`brain.db` survives)
- [ ] `ailang cache stats` shows namespace breakdown and storage size
- [ ] GC removes expired frames based on TTL
- [ ] Export/import enables backup and cloud migration
- [ ] All existing SharedMem/SharedIndex tests still pass
- [ ] No performance regression: hooks are async, don't block coding flow
- [ ] Configuration in `config.yaml` for backend, TTLs, hook enablement
- [ ] CHANGELOG.md updated
- [ ] User guide at `docs/docs/guides/brain-cache.md`

---

## Testing Strategy

**Unit tests:**
- SQLiteSharedCache: all SharedCache interface methods (Get/Put/CAS/Delete/Keys/Len)
- Search: SimHash search, FTS5 keyword search, combined ranking
- GC: TTL expiry, namespace-scoped cleanup
- Schema: migration versioning, fresh creation, upgrade from empty

**Integration tests:**
- CLI: `ailang cache search/list/put/stats/gc/export/import` end-to-end
- Hook simulation: mock tool use → verify frame captured
- Session start: populate brain → query with file context → verify relevant results

**Stress tests:**
- Concurrent read/write (following existing `sharedmem_test.go` patterns)
- 10K frame insertion + search performance benchmarks
- GC under concurrent access

**Manual testing:**
- Full workflow: code → commit → new session → verify brain context
- Search quality: verify SimHash + FTS5 returns useful results

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Hook latency slows coding | Medium | All capture hooks are async (`&` background); session start hook has 2s timeout |
| SQLite WAL conflicts with messaging DB | Low | Separate database file (`brain.db` vs `collaboration.db`); independent connections |
| SimHash quality insufficient for short content | Medium | FTS5 keyword fallback; neural embeddings as opt-in upgrade |
| Brain grows unbounded | Low | Default TTLs per namespace; `ailang cache gc` in session start; configurable limits |
| Ollama unavailable for neural search | Low | Neural is optional; SimHash + FTS5 work without Ollama |
| Hook shell scripts fragile across platforms | Medium | Keep hooks simple (5-10 lines); test on macOS + Linux; provide `--no-brain` escape hatch |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | SQLite provides deterministic storage; SimHash scoring is deterministic; search ordering is stable (score DESC, key ASC) |
| A2: Replayability | +2 | Resolution frames create auditable problem→solution history; brain is exportable/importable |
| A3: Effect Legibility | +1 | SharedMem remains explicit effect; hooks are visible in settings.json; no hidden side effects |
| A4: Explicit Authority | +1 | Cache access requires SharedMem capability; hook registration is explicit in config |
| A5: Bounded Verification | +1 | Frame validation is local (key uniqueness, schema compliance); no global reasoning needed |
| A6: Safe Concurrency | 0 | SQLite WAL handles concurrent access; no new concurrency patterns introduced |
| A7: Machines First | +2 | Core value: accumulated knowledge designed for machine consumption; structured frames not prose |
| A8: Minimal Syntax | 0 | No new language syntax; CLI and hooks only |
| A9: Cost Visibility | +1 | Storage costs visible via `ailang cache stats`; TTLs prevent unbounded growth |
| A10: Composability | +1 | Builds on existing SharedCache interface; SQLite backend composes with all existing sem_frame operations |
| A11: Structured Failure | +1 | Cache miss returns Option[sem_frame]; GC failures are logged, not silent |
| A12: System Boundary | +2 | Clear boundary: hooks capture → SQLite stores → CLI queries. Cloud upgrade is explicit backend swap. |

**Net Score: +13** → **Decision: Strong Accept**

### Hard Violation Check

- [x] A1 (Determinism): SQLite + SimHash deterministic; no implicit nondeterminism
- [x] A3 (Effects): SharedMem effect explicit; hooks visible in config
- [x] A4 (Authority): No ambient access; capability-gated
- [x] A7 (Machines First): Structured frames for machine consumption; this IS the axiom

---

## Related Documents

**Direct predecessors (this doc builds on):**
- [DX-15-semantic-caching-MVP.md](../../implemented/v0_6_0/DX-15-semantic-caching-MVP.md) — SharedMem effect (v0.5.11)
- [dx-16-shared-index-deterministic-retrieval.md](../../implemented/v0_6_0/dx-16-shared-index-deterministic-retrieval.md) — SharedIndex effect (v0.5.11)
- [semantic-caching-complete.md](../../implemented/v0_6_0/semantic-caching-complete.md) — Implementation status
- [semantic-caching-future.md](../../implemented/v0_5_11/semantic-caching-future.md) — Future work roadmap (Redis, Firestore, hybrid search)

**Vision documents:**
- [m-sem-kernel-vision.md](../../implemented/v0_8_0/m-sem-kernel-vision.md) — AILANG as symbolic reasoning kernel (pillar 1: stable symbolic memory)
- [m-semantic-envelope.md](../../implemented/v0_8_1/m-semantic-envelope.md) — Multi-aspect semantic embeddings for messaging

**Infrastructure this builds on:**
- [m-msg-semantic-caching.md](../../implemented/v0_6_0/m-msg-semantic-caching.md) — Messaging semantic search (SQLite patterns)
- [cloud-messaging-integration.md](../../../docs/docs/guides/cloud-messaging-integration.md) — Cloud architecture (Firestore patterns)

**Cloud evolution:**
- [m-cloud-progress-tracking.md](../v0_9_2/m-cloud-progress-tracking.md) — Cloud Run progress streaming
- [m-agent-orchestration.md](../../planned/v1_0_0/m-agent-orchestration.md) — Multi-agent orchestration (shared brain = shared Firestore)

---

## Future Work

1. **Firestore backend** — Cloud agents share a Firestore brain; knowledge accumulates across all sessions (the "AILANG brain in the cloud")
2. **Multi-agent brain** — Coordinator writes resolution frames that all agents can query; agents learn from each other's fixes
3. **Hybrid search pipeline** — SimHash prefilter (100 candidates) → FTS5 keyword boost → embedding rerank (top-K)
4. **Envelope integration** — Wire brain frames into M-SEMANTIC-ENVELOPE's 5-slot system for richer message context
5. **Brain analytics** — "What has the brain learned this week?" summaries; knowledge gap detection
6. **Auto-summarization** — Periodically summarize clusters of related frames into consolidated knowledge entries
7. **Cross-project brains** — Share brain frames between related projects (e.g., ailang + ailang-multivac)

---

**Document created**: 2026-03-13
**Last updated**: 2026-03-13
