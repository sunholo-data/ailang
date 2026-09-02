# Mission Dashboard — V1

_Snapshot, overwritten each iteration. History lives in `v1-mission.md` STATUS + `v1-mission-log.md`._

**Last iteration:** 319 · 2026-09-02 · LANDED · [HARNESS]
**Latest release:** v0.34.0

## Goal distance
**N = 10 design docs remaining before v1.0.0** — unmoved this iteration (HARNESS work moves the goal by 0).
D-53's **UNCLASSIFIED bucket of 4** (would make it 14) is still named and unruled.

## Just landed
`m-probe-derace-has-no-killer` — PR [#1020](https://github.com/sunholo-data/ailang/pull/1020) → `f5d031161`, 21 checks, zero not-green, CLEAN.
The de-race that cleared a dev red at iteration 317 was pinned by **nothing**: a full revert passed all 42 arms rc=0 (112s vs a 50s baseline), and wall time is the only thing that moved. Now a sole killer. Also adds the missing suite-scope leak guard for `PROBE_TREE_DISCOVERY_SECS`.

## Up next (banked queue head)
1. `m-probe-discovery-default-30s-unpinned` — a production-path tightening nobody chose and no test pins (mutant 30→5 passes 42/42).
2. `m-docparse-v0340-reports-2026-09-01` — VERIFY-then-route; a live consumer's silent export drop, already failed to reproduce in two shapes.
3. `m-changeclass-unknown-consumers` — `U` is a fourth enum value in switches written for three; a PRE-CONDITION for Sprint 2.

## Loop health / routing
- Controller `claude:claude-opus-5` · executor `codex:gpt-5.6-sol` (probe rc=0, one sandboxed run) · evaluator `sonnet` in its **own** worktree (generator≠judge holds) · designer rotation at `claude:claude-fable-5`, **did not run**.
- Per-gate `mission-heartbeat.sh` stamps in use (D-52).
- Running skill was **7 commits / 186 lines behind origin**, plus 64 uncommitted lines; delta read and both missing rules applied before proceeding.
- **CI corrected this iteration once.** The judge and I removed two executor stabilizers as unnecessary on 8 local runs; the GitHub macOS runner reddened deterministically and both are restored. Iteration 318's lesson in mirror image — a scheduling property is platform-scoped and cannot be declared *unnecessary* from one host any more than *unreachable*.

## Parked on Mark (both OPEN, neither answered)
- **D-53** — rule on the 4 UNCLASSIFIED docs (N=10 vs N=14). Loop recommends N=12. Default: keep reporting 10 with the bucket named.
- **D-54 — ESCALATED, re-measure it before quoting the row.** The main checkout went **9 ahead → 22 ahead** and **27 → 31 behind** *within this one iteration*: 13 new unpushed commits, including standard-mode Anthropic OAuth, Fable 5.1 registration and a full design doc through three quorum rounds. The row was written about 9 stranded bookkeeping commits; it is now 22 commits of substantive feature work that no unattended pick can see. Loop recommends option (b): authorise a PR from the main checkout's `dev`. Default: keep flagging and skipping.

## Cost posture
Metered **$0.00** of the $5 iteration ceiling. Every lane a quota bucket; no quorum round, no designer, no planner.
