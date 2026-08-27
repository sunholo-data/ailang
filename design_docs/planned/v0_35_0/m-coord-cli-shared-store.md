# M-COORD-CLI-SHARED-STORE: control-plane commands must operate on the daemon's actual store

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1 — the operator's approve/reject/list levers are silently inert whenever the daemon runs on shared (gcp) storage, which is the Studio's live configuration
**Estimated**: ~2 days
**Dependencies**: None

## Quorum triggers

All four skip conditions hold: no design-freeze items (the direction is a bug, not a decision),
localized to the CLI's store wiring (fixes shared machinery, overrides nothing), no cost/KPI or
banked-data surface, every premise verified in-repo. **Quorum skipped per checklist.**

## Axiom Compliance

Eval/ops infrastructure; language axioms are neutral except:

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | — |
| A2: Replayability | 0 | — |
| A3: Effect Legibility | 0 | — |
| A4: Explicit Authority | 0 | — |
| A5: Bounded Verification | 0 | — |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | Agents controlling approvals programmatically hit the same dead CLI a human does; fixing it unblocks delegated control (Mark delegated approvals to the attended agent 2026-08-27 and the CLI could not serve it) |
| A8: Minimal Syntax | 0 | — |
| A9: Cost Visibility | +1 | A rejection that silently targets the wrong store lets duplicate/unwanted work continue spending |
| A10: Composability | 0 | — |
| A11: Structured Failure | +1 | "sql: no rows in result set" against the wrong database becomes either correct behavior or a loud wrong-store error |
| A12: System Boundary | 0 | — |

**Net Score: +3** → Move forward. Hard-violation check: none (A1/A3/A4/A7 all ≥ 0).

## Problem Statement

The coordinator daemon on the Studio runs with `AILANG_STORAGE=gcp` (launchd plist), so its
task/approval state lives in Firestore. Every coordinator CLI *action*, however, opens local
SQLite unconditionally. The operator's documented control commands are therefore blind to the
daemon they nominally control.

**Measured 2026-08-27** (all first-party, see Verification Log):

- `ailang coordinator reject task-08f450a0` → `Error: failed to get task: sql: no rows in
  result set` — for a task the daemon had *live in "awaiting approval"* at that moment.
- `ailang coordinator list` shows one stale task from May 27 — identically with and without
  `AILANG_STORAGE=gcp` in the CLI's environment, proving the CLI ignores the selector entirely.
- The rejection had to be issued through the dashboard HTTP API instead
  (`POST /api/approvals/apr-08f450a0/reject`), which worked because the server shares the
  daemon's backend wiring.

**Impact:** with approvals delegated to an agent (or exercised by a human over SSH), the CLI
path fails closed but *confusingly* — "no rows" reads as "task doesn't exist", not "you are
looking at the wrong database". Sub-second operator actions became a 20-minute detour through
handler source to find a working channel.

## Goals

**Primary Goal:** every coordinator control command operates on the same store as the daemon
it controls, or refuses loudly naming both stores.

**Success Metrics:**
- `coordinator list/pending/approve/reject/reopen/retry/cleanup/diff/logs/worktree` all show
  and act on the daemon's live tasks on a gcp-storage machine (measured against a real task).
- A deliberate mismatch (CLI forced local while daemon is gcp) produces an explicit
  wrong-store error naming both modes — never `sql: no rows`.
- A targeted `coordinator cancel <task-id>` exists for a running/queued task (the gap that
  forced "let both duplicates run" on 2026-08-27).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: CLI resolves its store the same way the daemon does (shared resolver honoring `AILANG_STORAGE`/config), instead of `NewSQLiteStore` hardcoding | Defines which plane every operator command hits | agent | design | med |
| D2: Store-identity guard: commands print the resolved store (mode + project) in their header, mirroring `ailang messages list`'s `store:` header — the control that caught the messages-plane version of this trap | Makes the wrong-store failure mode visible forever | agent | design | low |
| D3: New `coordinator cancel <task-id>` (targeted, any non-terminal state) distinct from `cleanup` (stale-only) | Only lever against duplicate/runaway dispatches | agent | design | low |

### Design Freeze

None — all decisions agent-resolvable; no human ratification required.

## Solution Design

### Overview

One shared store-resolution path for daemon and CLI. The systemic fix (per CLAUDE.md §3): audit
**every** direct `NewSQLiteStore` call site in `cmd/`, not just the six in
`coordinator_actions.go` — the audit already found a seventh in `chains_diff.go`.

### Conflict Surface

1. **`internal/storage/backend.go` resolver** (the `AILANG_STORAGE` switch, backend.go:83) —
   REUSE, not reimplement: the CLI must call the same resolver the daemon uses so the two can
   never diverge again.
2. **Scripts/automation invoking the CLI** (`tools/launchd/*`, skills) — behavior change is
   additive on gcp machines (commands start *working*); on local-only machines nothing changes
   (resolver yields local SQLite as today).
3. **`chains_diff.go`** shares the same defect class — include in the audit and fix.
4. **Dashboard server channel** (`internal/server/handlers_approvals.go`) — unchanged; it is
   already correct and stays the remote channel. This doc fixes the local CLI channel.
5. **Programs that MUST still work**: `coordinator start` (daemon path untouched), all
   commands on a pure-local machine, `coordinator status`/`watcher-status`.

### Implementation Plan

**M1 — shared store resolution (~1 day)**
- Extract/reuse the daemon's backend resolution for task/approval stores; replace all 7
  audited call sites; add the store-identity header line to every command's output (D2).
- Mismatch guard: if an explicit `--state-dir`/local flag contradicts the resolved daemon
  store, refuse with both identities named.
- **VERIFY**: on the Studio (daemon on gcp): `coordinator list` shows the daemon's live
  tasks; `reject` on a real pending approval succeeds end-to-end and the daemon processes it;
  forced-local control run produces the loud wrong-store error; `make ci` green.

**M2 — targeted cancel (~0.5 day)**
- `coordinator cancel <task-id> [--reason]`: marks a queued/running task cancelled through the
  shared store; the daemon's executor observes cancellation at its next checkpoint (reuse the
  existing cleanup cancellation state; no new state machine).
- **VERIFY**: dispatch a throwaway task, cancel it mid-run, observe the daemon abort + status
  `cancelled` + worktree preserved; `cancel` on a terminal task errors cleanly.

### Files to Modify/Create

- `cmd/ailang/coordinator_actions.go` — 6 call sites → shared resolver; add cancel (+~120 LOC)
- `cmd/ailang/chains_diff.go` — 7th call site (+~10 LOC)
- `internal/coordinator/` — store resolver export if not already public (+~40 LOC)
- tests: store-resolution unit tests + a mismatch-guard test (+~100 LOC)

## Success Criteria

- [ ] All Verification-Log defects reproduced as failing tests first, then green
- [ ] M1 + M2 VERIFY blocks executed and results recorded in this doc
- [ ] Zero remaining direct `NewSQLiteStore` calls in `cmd/` (grep in CI or lint note)

## Non-Goals

- Fixing dispatch/ack semantics, retrigger latency, or handoff observability — that is
  [m-coord-dispatch-integrity](m-coord-dispatch-integrity.md).
- Any change to daemon-side storage wiring (it is correct).
- Remote/dashboard channel changes.

## Verification Log (first-party, 2026-08-27)

| # | Claim | Evidence |
|---|-------|----------|
| V1 | CLI hardcodes local SQLite in all coordinator actions | `grep -n NewSQLiteStore cmd/ailang/coordinator_actions.go` → lines 52, 180, 295, 363, 445, 556; plus `chains_diff.go:64` |
| V2 | Daemon runs on gcp storage | plist: `AILANG_STORAGE=gcp`, `AILANG_CLOUD_PROJECT=ailang-multivac`; CLI shell control: `ailang storage status` → `Mode: local` |
| V3 | Live task invisible to CLI | `coordinator reject task-08f450a0` → `sql: no rows in result set` while daemon log showed the task awaiting approval; `coordinator list` → one May-27 task, byte-identical output with and without `AILANG_STORAGE=gcp` prefix |
| V4 | Dashboard channel works against the same task | `POST /api/approvals/apr-08f450a0/reject` → `{"status":"success"}`; subsequent GET → `"status":"rejected"` |
| V5 | No targeted cancel exists | `ailang coordinator` help: `cleanup` = "Cancel stale running/queued tasks" only; no `cancel <task-id>` |
| V6 | `AILANG_STORAGE` moves all three backends (why the resolver must be shared, not copied) | `internal/storage/backend.go:83` (per message-plane topology doc) |

## Related Documents

- [m-coord-dispatch-integrity](m-coord-dispatch-integrity.md) — sibling doc, same incident
- M-MESSAGE-PLANE-FAIL-LOUD (changelog [Unreleased], 2026-08-26) — the fail-loud family this
  continues; its M1 (standby exits non-zero) is the same "inert but healthy-looking" class
- `docs/internal/message-plane-topology.md` — store-selection topology and the `store:` header
  control this doc extends to the coordinator CLI
