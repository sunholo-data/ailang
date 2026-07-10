---
name: mission-control
description: Run ONE outer-loop iteration of a long-running mission (default: the V1 mission) — observe mission state, pick the top backlog item, route it through design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator with the mission's model routing policy, record a log entry, and run the retro. Use when user says "run mission control", "mission iteration", "work the v1 backlog", or when fired nightly by the dev.ailang.mission-control launchd job.
---

# Mission Control — one outer-loop iteration

Run ONE iteration of the mission defined in [`design_docs/v1-mission.md`](../../../design_docs/v1-mission.md)
(or the mission doc passed as argument). The gates run in order and are not skippable; earlier
gates are cheap and prevent expensive mistakes. This is the outer loop around the four honed
inner-loop skills — it does not duplicate them.

## Current State

- **Kill switch**: !'test -f ~/.ailang/state/mission-control.disabled && echo "DISABLED — STOP" || echo "armed"'
- **Branch / tree**: !'git branch --show-current && git status --porcelain | head -5'
- **gh account**: !'gh auth status 2>&1 | grep -E "Active account|Logged in" | head -2'
- **Queue head**: !'grep -A2 "^## Queue" design_docs/v1-mission.md | tail -2'
- **Last log entry**: !'grep "^## " design_docs/v1-mission-log.md | tail -1'
- **Unread inbox**: !'ailang messages list --unread 2>/dev/null | head -8 || echo "none"'
- **Parked evaluations**: !'ls .ailang/state/evaluations/ 2>/dev/null | tail -3 || echo "none"'

> Use the injected data above first; re-run only if empty or stale.

## Gate 0 — PREFLIGHT (deterministic; abort = exit silently with a controlplane message)

1. Kill switch set → STOP (no message needed; this is the intended off state).
2. `gh auth status` must show `sunholo-voight-kampff` before any push. Wrong account → fix with
   `gh auth switch --user sunholo-voight-kampff` or park all push steps.
3. Dirty working tree in the main checkout → do NOT stash/checkout (Critical Principle 0).
   Doc-only edits (mission doc, log) may proceed; sprint work goes to a coordinator worktree anyway.
4. Unread inbox messages: triage per agent-inbox skill. A genuine regression or human directive
   OUTRANKS the queue — it becomes this iteration's pick.

## Gate 1 — OBSERVE (cheap, read-only)

Read: the mission doc (queue, guardrails, routing policy — they may have changed), the last 1–2
log entries (especially **Next** and **Ruled out** — do not re-chase), any parked
`needs-human-review` items that got human answers in the inbox.

## Gate 2 — PICK + REALITY-CHECK

Take the top `[NEXT]` queue item. **Before any work, verify the doc's claimed status against repo
reality**: `git log --grep`, does the code/test already exist, does `make test` already cover it.
A design doc's status header is a claim, not a fact (M-EVAL-BENCH-UI shipped fully while its doc
said Planned for a month). If already done → the iteration's deliverable is the bookkeeping
(move doc to implemented/, update queue, log it) and you pick the NEXT item too.

**Verification protocol** (added iteration 1 after three same-class frictions):
1. **Rebuild before any live check**: `make quick-install && make build` — BOTH binaries.
   `~/go/bin/ailang` (PATH) and `bin/ailang` (preferred by test helpers when present) go stale
   independently; a stale one silently falsifies results (1a: stale installed binary showed
   pre-fix behavior; 1b-eval: Jun-26 `bin/ailang` v0.26.0 broke `make test` with a phantom
   `_io_flush` error). Confirm `--version` matches `git describe` before trusting output.
2. **A parked test is a claim, not evidence**: `t.Skip`-ed / disabled tests say "nobody
   re-checked", not "still broken". Un-skip and RUN before treating the bug as open — the
   M-TYPEENV-SUB "open P0" was already fixed; only un-skipping revealed it.
3. **Exit codes through pipes lie**: `cmd | tail; echo $?` reports tail's status. Use direct
   invocation or PIPESTATUS.

## Gate 3 — ROUTE + EXECUTE (the inner loop, with the routing policy)

Apply the mission doc's **model routing policy** (currently: Fable = design docs + evaluation +
this loop; Opus = sprint-planner + sprint-executor; Sonnet = deterministic mechanical work only).

- No design doc yet → invoke **design-doc-creator** (its hard gates apply: live `ailang check`
  verification, Conflict Surface for parser/types/codegen).
- Design doc but no plan → **sprint-planner** → sprint JSON + handoff.
- Plan exists → **sprint-executor** in an isolated worktree (coordinator-managed or
  `git worktree add` — NEVER the shared main tree; concurrent agents stomp uncommitted work).
- Execution complete → **sprint-evaluator**. Max 3 rounds; on round-3 fail →
  `needs-human-review`, park, message controlplane.

**GPU rule (two-tier)**: default iterations never touch `rig.lock` — it is a GPU mutex only.
If (and only if) a step drives ollama/local models: `source tools/launchd/rig-lock.sh &&
rig_lock_acquire wait` around THAT STEP, release immediately after. Ask explicitly at routing
time: "does this step touch the GPU?" — never let a test reach it by accident.

**Multi-week strategic items**: do not execute — the iteration's deliverable is DECOMPOSITION
into sprint-sized design docs (≤3–4 days each), queued individually.

## Gate 4 — RECORD (append-only; the log is the mission's memory)

Append an entry to `design_docs/v1-mission-log.md` using its fixed template — every section,
"none" over omission. The **Routing evidence** row and **Ruled out** ledger are the two highest-
value fields: evidence drives routing-policy changes; ruled-out stops re-chasing. Update the
mission doc's queue tags ([LANDED], [PARKED], etc.) and STATUS stamp.

## Gate 5 — RETRO + REPORT

1. Scan this iteration's friction (evaluator feedback, executor corrections, your own dead ends)
   plus unread `docs/sprint-retros/` material. Route each item to exactly ONE lane:
   - **skill fix** — edit the offending SKILL.md. Max ONE skill edit per iteration; requires ≥2
     recorded frictions pointing at the same gap; state both in the commit message.
   - **process fix** — edit the mission doc (guardrails/ordering/routing policy per its rules).
   - **backlog** — new design doc via design-doc-creator, or re-prioritize the queue.
2. Routing-policy change? Only with ≥3 evidence rows; stamp it in the mission doc.
3. Morning report: `ailang messages send controlplane "<summary>" --title "Mission iteration N:
   <headline>" --from mission-control` — what shipped, score, parked items needing Mark,
   any skill/process edit made.

## Standing rules

1. **One backlog item per iteration** (a bookkeeping-only pick allows taking a second).
2. **Never force through a guardrail** — park and report; the queue always has a next item.
3. **Commit per milestone** on `dev` (or the worktree branch); no pushes on the wrong gh account;
   NEVER release — stop at ready-to-release and report.
4. **The inner-loop skills are the contract** — improve them via Gate 5, don't bypass them
   mid-iteration because one is annoying. If a skill blocks you, that IS the retro finding.
5. **Data before conclusions** (PROGRAM.md invariant): no fix without a measured/reproduced
   failure; record refuted hypotheses in the log's Ruled out field.
