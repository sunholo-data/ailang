# Mission Dashboard — V1

> Snapshot only; history lives in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 251, 2026-08-22**.

## Where we are

- **Latest release** v0.33.1; the cons-cells programme targets v0.35.0 (LC-1 landed in v0.34.0's window).
- **dev CI** GREEN at `684ebc23e` — 16 checks, zero not-green (control fires at 16).
- **Decision ledger** 23 rows, **ZERO OPEN**. `D-22` was answered this iteration and closed.

## In flight / next

1. **`m-list-accessor-api` (LC-2)** — design doc LANDED and **quorum-cleared** (iter 251), 793
   lines / 28 verification rows. **Next: sprint-planner**, then executor. 3–4 days.
2. **`m-array-show-diverges-run-vs-compile` M4** — deferred exactly one iteration (lost to the
   directive, not to a problem). First task `m-array-typed-boundary-lines-unpinned`, then CHANGELOG,
   doc move to `implemented/v0_34/`, VL-9 correction.
3. Then LC-3a/3b/3c (mechanical, parallelizable) → LC-4 (the swap, riskiest) → LC-5 (tuning).

## The programme this unblocked

`D-19` = true cons cells; `D-22` = **`C1`, plain cons cells, not chunked**. Permanent fix for
[#676](https://github.com/sunholo-data/ailang/issues/676) (live user-reported OOM). 8 pieces,
15.5–21.5 person-days — **unchanged**: the roadmap was already scoped for C1.

## Loop cadence + routing

Controller `claude:claude-opus-5` · planner/executor `codex:gpt-5.6-sol` · evaluator `sonnet`.
**The designer rotation is degraded and it is worth knowing**: of three entries, gemini is
read-only (`CapRemoteSandbox`, cannot author) and `codex` collides with quorum reviewer
`gpt5-6-sol` on author-independence — so Fable is effectively the only clean authoring lane, which
is why iter 251 spent **two** Fable runs against a one-run diet (FLAGGED; 2nd instance after 228).
Every wait bounded; worktrees are siblings of the repo, never `/tmp`.

## Cost posture · Parked on Mark

Iteration 251 metered **$0.237173** of $5 — all quorum (two rounds, 2-of-2); the rest rides quota
buckets (opus ×1, fable ×2). **Parked on Mark: nothing** — zero OPEN decisions; `D-22` was the last,
answered `2026-08-22T11:36:26Z`.
