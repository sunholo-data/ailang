# Mission Dashboard — Motoko

_Snapshot, overwritten each iteration. History lives in the charter STATUS block and the mission log._

**Last iteration**: 34 — 2026-09-03 — row 6o · PR [#1034](https://github.com/sunholo-data/ailang/pull/1034) · evaluator **PASS 96/100, zero blocking** · **NOT LANDED — blocked on dev's own required reds**

## Where the goal is
- Row 6o's two defects are **fixed and independently verified**; the suite goes 43 → 46 arms, green 4/4 local runs.
- The work is **not merged**. `test` and `lint` are required on `dev` and both are red **on dev itself**, for causes this diff cannot touch (0 Go files changed).

## In flight / next
- **PR #1034** — open, rebased onto `267a94e92`, verified. Resume predicate, run as a command: `test` AND `lint` green on `dev`.
- **Next pick**: row **6p** — derive the suite's wall-clock/node-ceiling bounds from a stimulus measured in-test, rather than hardcoding constants calibrated on one machine at one load.
- Rows 10/11/12 stay Phase-0 parked: upstream `arniwesth/motoko_agent#154` still OPEN (re-measured this iteration, controls firing).

## Dev CI — two independent reds, neither ours, handed to V1
1. `c8c841e24` deleted the `FMT_AB_TESTABLE_FUNCTIONS` markers from `tools/launchd/nightly-eval.sh` → reds `test` and `launchd drivers (bash 3.2)`. V1's #1030 covers it (now MERGEABLE).
2. **New today**: `lint` fails `make fmt` on 7 Go files from the coordinator work. #1030 does NOT cover it. One command to fix; blocks every PR in the repo.
- Consequence worth naming: the `launchd drivers` target dies at the fmt_ab script **before** reaching the probe suite, so that leg is not just red, it is **blind** — motoko's arms have never run in CI.

## Loop / routing
- Controller `claude:claude-opus-5` · designer **`fable`** (rotation entry after the pointer's deepseek; Agent-tool pin) · planner **`opus`** (`derive-planner-lane.sh` → `opus fail-closed:planner-lane-field-missing`, verbatim) · executor **`codex:gpt-5.6-sol`** · evaluator **`sonnet`**, own worktree. generator≠judge holds.
- **Metered $0.47** of the $5 ceiling (2 quorum rounds + 3 restored-reviewer re-runs).

## Parked on Mark
**Nothing.** No open decision rows (ledger valid, 6 rows, 0 OPEN). The landing block is a predicate, not a decision.
