# M-STATE-DISK-RETENTION: Bounded On-Disk State and a `ailang disk` Command

**Status**: Planned
**Target**: v0.36.0
**Priority**: P1 (Medium — unbounded growth, no current outage; precedent incident reached 64GB)
**Estimated**: 3 days
**Dependencies**: None (extends M-OBS-RETENTION machinery, does not replace it)
**Planner-Lane**: codex-ok

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics; reclamation is out-of-band |
| A2: Replayability | 0 | Recent traces preserved under existing 7d/30d TTLs; this doc adds no new deletion of trace data |
| A3: Effect Legibility | +1 | Disk as an effect becomes explicit — every writer of `~/.ailang` is enumerated and attributed to an owner |
| A4: Explicit Authority | 0 | No capability changes; destructive paths require an explicit flag |
| A5: Bounded Verification | +1 | Bounded state means bounded analysis surface; report output is a fixed schema |
| A6: Safe Concurrency | 0 | VACUUM is lock-sensitive but gated behind an explicit, non-default subcommand (see Risks) |
| A7: Machines First | +1 | `--json` report is the primary interface; auto-reclamation needs no human |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | Disk cost is currently unobservable through any CLI path; this makes it a first-class readout |
| A10: Composability | 0 | Reuses existing `RunRetention` / `GarbageCollect` rather than adding a parallel mechanism |
| A11: Structured Failure | +1 | Dry-run is the default; every reclaimed byte is attributed and reported; no silent deletion |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): Reclamation is explicit and reported, never silent
- [x] A4 (Authority): No ambient access granted; `clean`/`vacuum` require explicit invocation
- [x] A7 (Machines First): Report is machine-readable first, human-formatted second

### Decision Thresholds

Net +5 ≥ +2 → proceed to draft.

---

## Verification Log

Every load-bearing claim below was checked against the code or the running system on
**2026-09-08** against `dev` at v0.35.3. Negative-existence claims (the design's premises rest
on things *not* existing) each carry their own row, per the design-doc-creator hard gate.

| # | Claim | Kind | How verified | Result |
|---|-------|------|--------------|--------|
| V1 | Observatory TTLs are 7d (spans, trace_summaries, metrics) and 30d (chat_messages, session_tools) | positive | Read `internal/observatory/retention.go:39-58` doc comment + the five `steps` DELETE templates | Confirmed |
| V2 | `CheckHealth` size ladder is 200MB warn / 500MB cleanup / 2048MB refuse | positive | Read `internal/observatory/health.go:23-31,66-104` | Confirmed |
| V3 | `RunRetention` has exactly two callers | positive | `grep -rn "RunRetention(" --include="*.go" .` → `health.go:86`, `daemon.go:544` only (rest are decls/tests) | Confirmed |
| V4 | `DeleteOldTasks` is **never called** anywhere in the codebase | **negative** | `grep -rn "DeleteOldTasks" --include="*.go" . \| grep -v _test.go` → 3 hits, all declarations (`store.go:276` interface, `store_sqlite_queries.go:114` impl, `storage/firestore/coordinator_approvals.go:223` impl). Zero call sites | Confirmed — dead code |
| V5 | `DeleteOldTaskEvents(30d)` is called only from the coordinator daemon | positive | `grep -rn "DeleteOldTaskEvents"` → `daemon.go:557` sole caller | Confirmed |
| V6 | The compile cache has **no eviction, no TTL, and no size cap** | **negative** | `grep -niE "evict\|gc\|prune\|clean\|maxSize\|budget\|remove\|delete" internal/pipeline/cache_store.go` → only `Clear()` at :118 (all-or-nothing, removes everything) | Confirmed |
| V7 | Compile cache path is `<projectDir>/.ailang/cache/compile`, with an `AILANG_CACHE_DIR` override that already exists | positive | Read `internal/pipeline/cache_runtime.go:45-50` (`cacheRootPath`) and `cache_store.go:64-70` | Confirmed — override exists, so relocation needs no new plumbing |
| V8 | The two hook scripts append to their logs with **no rotation or size check** | **negative** | Read `internal/executor/claude_telemetry.sh:31-38` and `scripts/hooks/session_start.sh:48-58`; grepped both for `rotate\|truncate\|wc -c\|stat -f` — the only hits are unrelated (jq payload truncation at :129, curl output parsing at :178, lock-age `stat` at :110) | Confirmed |
| V9 | The telemetry `log()` is unconditional, not behind a debug flag | positive | `claude_telemetry.sh:36` comment reads `# Log function (ALWAYS enabled for debugging coordinator hooks issue)`; `log "RAW_INPUT: …"` at :45 runs on every hook event | Confirmed |
| V10 | `CheckHealth` runs on **every** `ailang` invocation | positive | Read `cmd/ailang/main.go:72-75` — called immediately after `flag.Parse()`, before subcommand dispatch | Confirmed |
| V11 | The name `cache` is **already taken** by the brain command | positive | `grep -n "case \"" cmd/ailang/main.go` → `:356 case "cache", "brain":` → `cacheCommand()`. `ailang cache --help` confirms it is the semantic brain, not the compile cache | Confirmed — **`disk` chosen to avoid collision** |
| V12 | The name `disk` is **unallocated** as a subcommand | **negative** | `grep -n "case \"" cmd/ailang/main.go` (88 cases) — no `disk` case; `storage` (:458) exists but is GCP migration (`migrate`/`verify`/`status`), not local disk | Confirmed |
| V13 | The brain has TTL GC, but only manually invoked | positive | `cmd/ailang/cache.go:309-355` `runCacheGC` calls `GarbageCollect()` (TTL) + optional `GarbageCollectOlderThan`. No automatic caller | Confirmed |
| V14 | The docsearch embeddings cache has orphan-only, manual cleanup — **no age or size policy** | **negative** | `internal/docsearch/embed.go:349` `CleanupCache` removes entries whose corpus file vanished; sole caller `cmd/ailang/docs_search.go:409` (explicit `--cleanup`). Grep for `maxAge\|ttl\|evict` in `embed.go` → none | Confirmed |
| V15 | `~/.ailang/speak/` is written by no Go code and has no retention | **negative** | `grep -rn '"speak"' --include="*.go" .` → no hits. Directory holds `sessions/` (93 dirs, 70MB) plus three logs | Confirmed — owner unidentified in-repo (see Deferred) |
| V16 | No new diagnostic error code is required | **negative** | This adds a CLI subcommand, not a compiler diagnostic. No `MODxxx`/`TCxxx`/`PARxxx` allocation is proposed, so no namespace grep applies | N/A by construction |
| V17 | Measured: coordinator.db is 99.6% free pages | measurement | `PRAGMA page_count/freelist_count/page_size/auto_vacuum` → 36906 pages, 36756 free, 4096B, auto_vacuum=0. Live rows: tasks=231, approval_requests=111, task_events=0 | Confirmed — 143MB of 144MB is reclaimable free space |
| V18 | Measured: observatory.db has 0 free pages and 52,829 spans past TTL | measurement | `PRAGMA freelist_count` → 0. `SELECT COUNT(*) FROM spans WHERE datetime(start_time) < datetime('now','-7 days')` → 52829; trace_summaries → 19605; metrics → 5774 | Confirmed |
| V19 | Measured: retention last ran ~2026-08-31, matching the daemon's last write | measurement | Oldest span `2026-08-31 05:37:53`; `coordinator.db` mtime `2026-08-31 07:37` | Confirmed — consistent with V3/V5: the daemon is retention's only real driver |
| V20 | Measured: 247 repo-local `.ailang` dirs totalling 1.0GB, 0.9GB of it compile cache | measurement | `find ~/dev ~/Documents ~/.gemini -type d -name .ailang -prune \| xargs du -sk` | Confirmed |

**One earlier hypothesis was refuted by V17 and is recorded here so it is not re-derived:**
the coordinator's 144MB was initially attributed to V4 (the never-called `DeleteOldTasks`
letting the `tasks` table grow). The row counts disprove it — 231 tasks. The space is
*deleted-but-unreclaimed*, so the fix is VACUUM, not more deletion. V4 remains a real defect
but is not the cause of the disk usage, and fixing it alone would have reclaimed nothing.

---

## Problem Statement

AILANG writes to disk in eight places across two roots. Nothing enumerates them, and three
distinct failure modes let them grow without bound.

**Measured on one developer laptop, 2026-09-08, before any cleanup: 2.2 GB total** —
1.2 GB in `~/.ailang`, 1.0 GB scattered across 247 repo-local `.ailang/` directories.

**Current State:**

**Cause A — space is deleted but never returned to the filesystem.** No database sets
`auto_vacuum`, and `RunRetention` deliberately does not VACUUM (`retention.go:50-54`: it needs
an exclusive lock that a running `ailang serve` blocks). Retention therefore converts live
pages into free pages that no process ever reclaims:

| Database | On disk | Free pages | Live rows |
|---|---|---|---|
| `coordinator.db` | 144 MB | **36,756 / 36,906 (99.6%)** | tasks=231, approvals=111, task_events=0 |
| `collaboration.db` | 24 MB | 3,027 / 6,296 (46%) | — |

The coordinator DB is a near-empty 144MB file. Its retention *worked* — `task_events` is at
zero rows, down from the 101K/144MB recorded in M-OBS-RETENTION — and the reward for that
working was a file that never shrank.

**Cause B — retention is implemented, correct, and not driven.** `RunRetention` has exactly
two callers (V3): the coordinator daemon's periodic loop, and `CheckHealth` when the DB
exceeds 500MB. The daemon last ran 2026-08-31 (V19). The DB sits at 290MB — inside the
200–500MB warn band — so the startup path only logs. Result:

- 52,829 spans, 19,605 trace_summaries and 5,774 metrics are past their 7-day TTL and still stored (V18)
- `Observatory: 293MB (warn threshold: 200MB)` prints on **every single `ailang` invocation**
  (V10), warning the operator about a condition the binary has the code to fix and declines to

A warning that fires on every command and never resolves is indistinguishable from noise, and
gets filtered out — which is what happened here for eight days.

**Cause C — subsystems with no policy at all.**

| What | Size | Policy | Evidence |
|---|---|---|---|
| Compile caches (247 dirs) | 0.9 GB | none — no TTL, no cap, no eviction | V6, V20 |
| `state/telemetry_hooks.log` | **475 MB**, 1,684,856 lines | none — unrotated, and `log()` is unconditional | V8, V9 |
| `state/hooks.log` | 37 MB | none — unrotated | V8 |
| `speak/sessions/` (93) | 70 MB | none; no in-repo writer | V15 |
| `cache/embeddings/` | 70 MB | orphan-only, manual | V14 |
| Brain (`brain.db` + project brains) | 19 MB + 13 MB | TTL exists, manual invocation only | V13 |

The single largest file on disk was a debug log left switched on — `# Log function (ALWAYS
enabled for debugging coordinator hooks issue)` — that stopped being written on 2026-08-06 and
was never noticed, because no command reports file sizes.

The compile cache compounds differently: `cacheRootPath` keys on `projectDir` (V7), so one
cache directory appears per directory ever compiled in. On the measured machine that produced
247 of them, including caches under `ailang-parse/_site/` and `ailang-parse/docs/` — build
output directories that were compiled in once and left 32MB behind each. 187 of the 247 had
not been touched in two months and held 652MB.

**Impact:**

- **Developers**: 2.2GB of mostly-dead state with no command to see it. The precedent incident
  (M-OBS-RETENTION, v0.10.0) reached 64GB and rendered a laptop unusable; the mechanisms that
  produced it are fixed, but the classes around it are not.
- **AI agents**: an agent asked to free space has no tool for it and must shell out to `du`,
  `find` and `sqlite3` — exactly the "second toolchain" problem AILANG exists to remove.
- **CI**: the compile-cache-per-directory pattern means ephemeral runners accumulate caches
  they never reuse.

## Goals

**Primary Goal:** Make every byte AILANG writes to disk visible through one command and
reclaimable without manual `sqlite3`, reducing steady-state footprint on the measured machine
from 2.2GB to under 400MB.

**Success Metrics:**

- `ailang disk` reports every location in the table above, with size, age and owning subsystem — currently 0 of 8 are reachable from any CLI path
- Free-page waste reclaimed: coordinator.db 144MB → ~1MB, collaboration.db 24MB → ~13MB
- Observatory converges to its own TTL without the daemon running: 290MB → ~190MB, and the every-invocation warning stops firing on a DB the binary could have pruned
- Hook logs bounded: unbounded → ≤10MB each by construction
- Compile cache directories: 247 → 1 per project root, with stale entries aged out
- Zero silent deletions — every reclaimed byte attributed in the report

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Command is `disk`, not `cache` or `storage` | Both names are taken with unrelated meanings (V11, V12); picking either would overload a live command and break agent expectations | human | design | high |
| Reclamation defaults to dry-run; `--yes` required to delete | Determines whether an agent can invoke this unattended without risk. Inverting it later is a breaking behavioural change | human | design | high |
| `auto_vacuum=INCREMENTAL` on new DBs + explicit `disk vacuum` for existing ones | Auto-vacuum cannot be enabled on an existing DB without a full VACUUM; choosing incremental now avoids a second migration later | human | design | high |
| Compile cache relocates to a single content-addressed root under `~/.ailang/cache/compile/<project-hash>` | Changes on-disk layout other lanes may assume; conflicts with a PARKED doc on the same file (see Conflict Surface) | human | design | high |
| Retention gains a time-based trigger (stamp file) alongside the size ladder | Fixes Cause B, but means retention can now run on a path that previously only logged | agent | design | med |
| Telemetry `log()` moves behind `AILANG_HOOK_DEBUG=1` | Removes the largest single growth source; loses always-on forensics for the hooks path | agent | implementation | low |
| Report emits a fixed JSON schema | Downstream tooling and agents will bind to it | agent | implementation | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Subcommand name `disk` ratified (vs. extending `doctor`, which already hosts diagnostics)
- [ ] Dry-run-by-default confirmed as the destructive-path contract
- [ ] `auto_vacuum=INCREMENTAL` vs. periodic explicit VACUUM chosen
- [ ] Compile cache relocation approved, **and sequenced against the PARKED
      `m-cache-module-id-encoding` doc** — both edit `internal/pipeline/cache_store.go`
- [ ] Default staleness threshold for compile caches (proposed: 30 days)

## Solution Design

### Overview

Three independent fixes plus one new command. The fixes are ordered by payoff-to-risk and are
separately shippable; the command is what makes any of it observable.

### Architecture

**Components:**

1. **`internal/diskreport`** (new) — walks the known locations, returns a typed report. Pure
   enumeration and sizing; performs no deletion. This is the contract-bearing core.
2. **`cmd/ailang/disk.go`** (new) — `report` / `clean` / `vacuum` subcommands over that core.
3. **Retention trigger** — a `state/.retention-stamp` file lets `CheckHealth` run retention when
   it last ran more than 7 days ago, regardless of which size band the DB is in.
4. **Reclamation** — `PRAGMA auto_vacuum=INCREMENTAL` for newly created DBs; `disk vacuum` for
   the existing ones, which refuses to run while `ailang serve` holds the DB.
5. **Bounded hook logs** — a shared `rotate_if_large()` in the hook scripts, plus gating the
   telemetry `log()` behind `AILANG_HOOK_DEBUG=1`.
6. **Compile cache aging** — a `last_used` timestamp in the manifest; entries older than the
   threshold are dropped on load. Default `AILANG_CACHE_DIR` to a single per-project root.

### Implementation Plan

**Phase 1: Visibility** (~6 hours)

- `internal/diskreport` enumerating all eight locations, with sizes, ages and owners
- `ailang disk` / `ailang disk --json`
- Unit tests over a synthetic tree; contract: reported total equals sum of parts

**Phase 2: Reclaim what is already dead** (~8 hours)

- `ailang disk vacuum` — checkpoint + VACUUM, refusing while the server holds a lock
- `PRAGMA auto_vacuum=INCREMENTAL` in the create path for all three DBs
- Time-based retention trigger via stamp file in `CheckHealth`
- Wire `DeleteOldTasks` into `runRetentionCleanup` next to the existing `DeleteOldTaskEvents`
  call (fixes V4 — one line, already implemented and tested)

**Phase 3: Stop the growth** (~10 hours)

- `rotate_if_large()` in `scripts/hooks/` shared helper; both scripts adopt it
- Telemetry `log()` behind `AILANG_HOOK_DEBUG=1`
- Compile cache `last_used` + age-out on load
- `ailang disk clean [--older-than 30d] [--yes]`, dry-run by default

### Files to Modify/Create

**New files:**

- `internal/diskreport/report.go` — location enumeration and sizing (~180 LOC)
- `internal/diskreport/report_test.go` — synthetic-tree tests (~140 LOC)
- `cmd/ailang/disk.go` — `report`/`clean`/`vacuum` subcommands (~220 LOC)
- `scripts/hooks/lib/rotate.sh` — shared `rotate_if_large()` helper (~25 LOC)

**Modified files:**

- `internal/observatory/health.go` — stamp-file time trigger alongside the size ladder (+35 LOC)
- `internal/coordinator/daemon.go` — call `DeleteOldTasks` in `runRetentionCleanup` (+6 LOC)
- `internal/pipeline/cache_store.go` — `last_used` in the manifest, age-out on load (+60 LOC)
- `internal/pipeline/cache_runtime.go` — default `AILANG_CACHE_DIR` to a per-project root (+20 LOC)
- `internal/executor/claude_telemetry.sh` — gate `log()`, adopt rotation (+8 / −2 LOC)
- `scripts/hooks/session_start.sh` — adopt rotation (+4 LOC)
- `cmd/ailang/main.go` — dispatch `case "disk":` (+5 LOC)

## Examples

### Example 1: Finding out where 2.2GB went

**Before:**

```bash
$ ailang disk
ailang: unknown command "disk"
# operator falls back to:
$ du -sh ~/.ailang/* | sort -rh
$ find ~/dev -type d -name .ailang -prune | xargs du -sk | awk '{s+=$1} END {print s}'
$ sqlite3 ~/.ailang/state/coordinator.db "PRAGMA freelist_count;"
```

**After:**

```bash
$ ailang disk
AILANG on-disk state — 2.2 GB total

~/.ailang                                          1.2 GB
  state/telemetry_hooks.log        475 MB  33d old  unrotated  ← largest, stale
  state/observatory.db             290 MB   0% free  52829 spans past 7d TTL
  state/coordinator.db             144 MB  99% free  ← reclaimable via 'disk vacuum'
  state/hooks.log                   37 MB   0d old   unrotated
  state/collaboration.db            24 MB  46% free
  speak/sessions (93)               70 MB            no retention policy
  cache/embeddings                  70 MB            orphan-cleanup only

repo-local compile caches (247)                    1.0 GB
  187 not used in >30d                             652 MB  ← reclaimable via 'disk clean'

Reclaimable now: 1.4 GB   Run 'ailang disk clean' to preview.
```

### Example 2: An agent reclaiming space unattended

```bash
$ ailang disk clean --older-than 30d --json          # dry run, the default
{"would_reclaim_bytes":1503238553,"items":[
  {"path":"~/.ailang/state/telemetry_hooks.log","bytes":498393409,"reason":"unrotated log, last write 33d ago"},
  {"path":"<247 compile caches>","bytes":683671552,"reason":"unused >30d"}]}

$ ailang disk clean --older-than 30d --yes
Reclaimed 1.4 GB (telemetry_hooks.log 475 MB, 187 compile caches 652 MB, vacuum 155 MB)
```

## Success Criteria

- [ ] `ailang disk` enumerates all eight locations from the Problem Statement table
- [ ] `ailang disk --json` emits a stable schema, covered by a golden test
- [ ] `ailang disk clean` is a no-op without `--yes`, proved by a test asserting nothing is removed
- [ ] `ailang disk vacuum` exits non-zero with a clear message while `ailang serve` holds the DB
- [ ] After vacuum, `PRAGMA freelist_count` on coordinator.db is < 100 pages
- [ ] Retention runs from a cold `ailang` invocation with the daemon stopped and a >7d stamp, on a DB inside the 200–500MB band
- [ ] `DeleteOldTasks` has a live call site (V4 closed)
- [ ] Hook logs rotate at 10MB; a test writes 11MB and asserts the file is capped
- [ ] `AILANG_HOOK_DEBUG` unset produces no `RAW_INPUT:` lines
- [ ] Compile cache entries unused >30d are dropped on next load
- [ ] All tests passing; `make quick-install` clean
- [ ] Documentation updated (CLI reference, `ailang --help`)

## Testing Strategy

**Unit tests:**

- `diskreport` over a synthetic tree with known sizes — total equals sum of parts
- Age-classification boundaries (29d/30d/31d) for cache staleness
- `rotate_if_large()` at 9.9MB (no-op) and 10.1MB (rotates)

**Integration tests:**

- Seed a DB with >7d rows, set the stamp file back 8 days, run `ailang version`, assert rows deleted — the Cause B regression test
- Seed a DB with deleted rows, run `disk vacuum`, assert `freelist_count` drops
- `disk clean` without `--yes` leaves the tree byte-identical

**Manual testing:**

- Run against the measured 2.2GB machine; compare `ailang disk` totals against `du -sh`
- Confirm the every-invocation observatory warning stops after one converged retention pass

## Deferred Decisions

The following are intentionally left open for the implementer:

- Report table formatting and column widths — agent may choose
- Whether `disk` gains a `--path` filter for a single location — agent may choose
- Rotation strategy: single `.1` backup vs. truncate-in-place — agent may choose
- Hash function for the per-project cache root — agent may choose
- Whether `speak/sessions` gets a retention policy in this doc or is only *reported* — **human at review**; V15 could not identify an in-repo writer, so its ownership must be established before anything deletes it
- Whether `disk clean` should also run brain `GarbageCollect()` — human at review

## Non-Goals

**Not attempted in this feature:**

- **Changing any existing TTL value** — 7d/30d are M-OBS-RETENTION's and stay as they are
- **Reducing span ingestion volume** — that is `m-obs-configurable-span-filtering`, complementary
- **Cloud/Firestore storage** — `ailang storage` owns that surface; this doc is local disk only
- **Compression or an alternative store** — same reasoning as M-OBS-RETENTION's non-goals
- **A background daemon for reclamation** — the coordinator daemon already exists and is the wrong dependency to deepen; that dependency is precisely Cause B
- **Deleting anything under `speak/` before its owner is identified** (V15)

## Conflict Surface

The design-doc-creator gate requires this section for parser/typechecker/codegen changes. This
doc touches none of those. It is written anyway because the change **overrides shared
machinery** other lanes depend on, and doing so surfaced two live conflicts:

### Shared machinery touched

| Surface | Who else depends on it | Reuse or override | Note |
|---|---|---|---|
| `internal/pipeline/cache_store.go` | **PARKED** `m-cache-module-id-encoding` (v0_36_0) rewrites `sanitizeModuleID` in this file | override (both edit the manifest path) | **Sequencing conflict** — both docs target v0.36.0 and both change how cache paths are formed. Whichever lands second must rebase. Flagged as a Design Freeze item |
| `internal/observatory/health.go` `CheckHealth` | M-OBS-RETENTION (v0.10.0) owns the size ladder | reuse + extend | The ladder is kept exactly as-is; a time trigger is added alongside it, so the 500MB behaviour is unchanged |
| `internal/coordinator/daemon.go` `runRetentionCleanup` | M-OBS-RETENTION Component 6 | reuse | One added call to an already-implemented method |
| `AILANG_CACHE_DIR` env var | already honoured by `cacheRootPath` (V7) | reuse | Relocation changes only the *default*; an explicit override keeps working |

### What deliberately changes

- **Compile cache location moves.** Anything assuming `<projectDir>/.ailang/cache/compile`
  breaks. Grep before landing: the tests at `cache_artifacts_test.go:362`,
  `cache_invalidation_test.go:365,379,397,714,718` hard-code that path and will need updating.
- **Retention can now run on a startup path that previously only logged.** A user on a 300MB DB
  who upgrades will see one deletion pass they did not see before. This is the intended fix for
  Cause B, but it is a behavioural change, not a pure addition.
- **`RAW_INPUT:` hook logging stops by default.** Anyone debugging the hooks path must now set
  `AILANG_HOOK_DEBUG=1`.

### Programs that MUST still work

- `ailang cache stats` / `ailang cache gc` — the brain command, unrelated to `disk` (V11)
- `ailang storage status` — GCP surface, unrelated (V12)
- `ailang serve` holding observatory.db while `ailang run` invokes `CheckHealth` concurrently
- A build with `AILANG_CACHE_DIR` already set explicitly

## Quorum Review

**Triggers fired — quorum is recommended before planning:**

| # | Trigger | Fired? |
|---|---------|--------|
| 1 | Design-freeze items exist | **Yes** — five, including two layout decisions a human must ratify |
| 2 | Overrides shared machinery | **Yes** — `cache_store.go` collides with a PARKED doc targeting the same version |
| 3 | Cost/KPI semantics or banked-data schema | No |
| 4 | Premises about external systems | No — every premise is in-repo or measured locally, and re-checkable |

```bash
ailang design-quorum design_docs/planned/v0_36_0/m-state-disk-retention.md \
  --reviewers gpt5-6-sol,gemini-3-1-pro \
  --controller-verdict pass \
  --mission-log design_docs/v1-mission-log.md
```

## Timeline

**Day 1** (~6 hours): Phase 1 — `internal/diskreport` + `ailang disk` report + tests

**Day 2** (~8 hours): Phase 2 — vacuum, auto_vacuum, retention time trigger, `DeleteOldTasks` wiring

**Day 3** (~10 hours): Phase 3 — hook log rotation, telemetry gating, compile cache aging, `disk clean`

**Total: ~24 hours across 3 days** (estimate doubled from the initial 12h guess, per the
skill's estimation guidance)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| VACUUM blocks on a lock held by `ailang serve` | Med | `disk vacuum` detects the lock and exits non-zero with instructions rather than hanging — the failure mode `retention.go:50-54` explicitly warns about |
| VACUUM needs free space equal to the DB size | Low | Pre-flight check; refuse if free space < 2× DB size |
| Time-triggered retention deletes spans a user was about to inspect | Low | Only deletes past the existing 7d TTL, unchanged from M-OBS-RETENTION |
| Cache relocation collides with the PARKED encoding doc | **High** | Design Freeze item; sequence explicitly, do not land both blind |
| Gating telemetry logs hides a future hooks bug | Low | `AILANG_HOOK_DEBUG=1` restores it; the current always-on setting cost 475MB to keep a months-closed investigation warm |
| Aging out a cache entry still in use forces a recompile | Low | 30d default; recompile is a cost, not a correctness failure |
| `disk clean` deletes something with an unidentified owner | Med | `speak/` is reported but never cleaned until V15's ownership question is answered |

## Related Documents

**Implemented (may inform design):**

- [m-obs-retention.md](../../implemented/v0_10_0/m-obs-retention.md) — **direct precedent.** Built the observatory retention this doc extends, after a 64GB incident. Scope was observatory.db spans/metrics/chat + coordinator `task_events`. **Distinction:** that doc bounded *what is written*; this one addresses *what is never reclaimed* (Cause A), *retention that never fires* (Cause B), and the six subsystems it did not cover (Cause C). Its own non-goals list confirms it did not attempt VACUUM automation.
- [m-db-cleanup.md](../../implemented/v0_7_0/m-db-cleanup.md) — schema consolidation across the three DBs. Adjacent but orthogonal: naming and duplication, not size. Its recorded observatory.db size (~320MB, Jan 2026) is a useful baseline against today's 290MB.
- [m-obs-trace-triage.md](../../implemented/v0_12_0/m-obs-trace-triage.md) — references M-OBS-RETENTION's machinery

**Planned (check for overlap):**

- [m-cache-module-id-encoding.md](m-cache-module-id-encoding.md) — **PARKED, same target version, same file.** Concerns correctness of `sanitizeModuleID`'s path encoding, not cache size. No scope overlap, but a direct edit collision on `internal/pipeline/cache_store.go`. See Conflict Surface.
- [m-obs-configurable-span-filtering.md](../v0_9_3/m-obs-configurable-span-filtering.md) — reduces ingestion volume; complementary to bounding what is retained

**Duplicate gate result:** highest neural similarity across `planned/` and `implemented/` was
**0.31** (`m-mission-iteration-reliability`), below the 0.45 proceed threshold. The 1.00 SimHash
hits (`m-pure-prng`, `m-dx17-list-normalization-bug`) are keyword artifacts on unrelated topics.
`m-obs-retention` did not surface in either search and was found by grepping the
`M-OBS-RETENTION` marker at `cmd/ailang/main.go:73`; its distinction is stated above.

## References

- **Measurements**: this document's Verification Log, taken 2026-09-08 on one developer laptop at v0.35.3
- **Prior incident**: M-OBS-RETENTION — observatory.db reached 64GB (24GB DB + 40GB WAL)
- **Prior art**: Go's `go clean -cache` and `GOCACHE` single-root model; `git gc --auto`'s stamp-file trigger is the direct model for the Cause B fix
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **Consolidate the eight locations to two roots.** The split between `~/.ailang/state`,
  `~/.ailang/cache`, `~/.ailang/speak` and 247 repo-local directories is historical, not designed.
- **A size budget per subsystem**, enforced at write time rather than reclaimed after the fact — the
  logical end state, but it needs the visibility this doc builds before it can be calibrated.
- **`ailang doctor disk`** as a health-check integration once `doctor` gains a general reporting shape.
- **Identify and document the `speak/` writer** (V15), then give it a policy.

---

**Document created**: 2026-09-08
**Last updated**: 2026-09-08

---
DESIGN_DOC_PATH: design_docs/planned/v0_36_0/m-state-disk-retention.md
