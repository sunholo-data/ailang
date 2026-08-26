# Sprint Plan: M-PIPELINE-RECONCILIATION

**Design doc**: [m-pipeline-reconciliation.md](./m-pipeline-reconciliation.md) — D1–D4 ratified (Mark, attended, 2026-08-26)
**Created**: 2026-08-26
**Estimated**: 6 milestones, ~1,040 LOC + two CAS config changes + one plugin PR
**Risk**: Medium — M1 ripples through both store implementations; M4 touches the registry all 39 agents load through

## Pre-flight

| Check | Result |
|---|---|
| D1–D4 frozen | ✅ ratified in-session |
| Store implementations affected by M1 | 2 — `store_sqlite_approvals.go`, `storage/firestore/coordinator_approvals.go` (+ `coordinator_convert.go`) |
| Migration pattern exists | ✅ guarded `ALTER TABLE` list in `store_sqlite.go` |
| Evaluator verdict source | ✅ sprint-evaluator skill: 100-point rubric, PASS ≥ 70, JSON report |
| CI usable | ✅ green again since `db9cb0fe7` |

## Milestones

**M1 — Closed verdict type + `Evaluation` on approvals (~220 LOC)**
`EvaluationVerdict` (PASS/FAIL/UNAVAILABLE, parse+format, absence unrepresentable), `Evaluation` field on `ApprovalRequestRecord`, guarded SQLite migration, Firestore map round-trip, `UpdateApprovalEvaluationByTask` on the store interface + both impls.
AC: parse/format round-trips incl. unparsable→UNAVAILABLE; both stores build; update-by-task works (SQLite test); existing approval tests green.

**M2 — Per-edge auto-approve + verdict attachment (~250 LOC)**
`AutoApproveHandoffTo []string` on AgentConfig (consulted by `handleAgentHandoffs` even when the bool is false); `EvaluatesParent bool` — an evaluator agent's completion parses `EVALUATION_VERDICT:` from output and writes the verdict to the PARENT task's approval; the evaluator task FAILING writes `UNAVAILABLE(reason)` (the failure branch is the point); FAIL/UNAVAILABLE block auto_merge and auto-approved downstream handoffs, never the human gate.
AC: per-edge auto-approve fires only for listed targets; verdict lands on parent approval on success AND on evaluator failure; blocking test; unparsable output → UNAVAILABLE.

**M3 — Lane B chain gains the evaluator stage (config, CAS + plugin PR)**
`sprint-evaluator` agent (skill invoke, `skip_approval: true` — read-only, its value is the verdict, `evaluates_parent: true`); `sprint-executor` gains `trigger_on_complete: [sprint-evaluator]` + `auto_approve_handoff_to: [sprint-evaluator]`. Sync sprint-evaluator skill to ailang_bootstrap.
AC: config validates + CAS-written; plugin PR open.

**M4 — Chain-as-data (~250 LOC + config)**
`pipelines:` section (stage list + per-project bindings) expanded by the registry into the same effective AgentConfigs; fixture test proves expansion ≡ today's clones BEFORE the stapledon/twilight sextet is deleted from live config.
AC: expansion-equivalence fixture green; live config loses 6 entries; 39-agent lane fixture still green.

**M5 — Shared model routing (~200 LOC + config)**
`model_routing:` (role → ordered chain) in the CAS-managed config; Lane B resolves empty `model:` through it; `ailang coordinator routing <role>` CLI for Lane A's driver; static opus pins deleted.
**Explicitly deferred**: wiring `tools/launchd/mission-control.sh` to the CLI — the mission loops are live tonight; follow-up flagged in the doc.
AC: resolution unit tests; pins removed from live config; CLI prints the chain.

**M6 — Decision-ledger spine (~120 LOC + skill edit)**
`ailang coordinator pending` unions store approvals with unread `approvals`-inbox messages (Lane A items); mission-control Gate 3 posts `awaiting_approval` stages to the `approvals` inbox (Discord pings via `6345f2dc1`).
AC: pending shows both classes; skill edit committed.

## Order
M1 → M2 → M3 (Lane B evaluation live) → M4 → M5 → M6. Per-milestone commit + per-package tests; CI watched on push.
