# Mission Dashboard — Motoko

_Snapshot, overwritten each iteration. History lives in the charter STATUS block and the mission log._

**Last iteration**: 34 — 2026-09-03 — row 6o · **LANDED** as `b97cbf83c`..`684ab8331` (PR [#1034](https://github.com/sunholo-data/ailang/pull/1034), rebase-merged) · evaluator **PASS 96/100, zero blocking** · Gate 3b **GREEN 21/21**

## Where the goal is
- Row 6o **CLOSED**. Its two defects are fixed, independently judged, and **CI-verified**: the suite goes 43 → 46 arms, and all three new arms run green on the macOS runner, not just locally.
- The headline mutant (`kill -9 "-$pid"` → single-PID at probe:261) **survived at base** and is now the sole `not ok`. Row 6i's older coverage is intact.

## In flight / next
- **Nothing in flight.** #1034 merged; the branch is deleted.
- **Next pick**: row **6p** — derive the suite's wall-clock/node-ceiling bounds from a stimulus measured in-test, rather than hardcoding constants calibrated on one machine at one load.
- Rows 10/11/12 stay Phase-0 parked: upstream `arniwesth/motoko_agent#154` still OPEN (re-measured this iteration, controls firing).

## Dev CI — was red ~24h on five stacked defects; FIXED mid-iteration by V1 (`b51e53f78`)
- Motoko blocked on it, handed both known causes to V1, and kept its own pick. The second cause (a `lint`/`make fmt` red on 7 Go files) was **net-new** — V1's in-flight #1030 did not cover it.
- V1's fix found three further defects behind the first two. dev is green again; this iteration landed on the fixed base.
- Retired: the `launchd drivers` leg is no longer blind — the probe suite is reached and motoko's arms run there.

## Loop / routing
- Controller `claude:claude-opus-5` · designer **`fable`** (rotation entry after the pointer's deepseek; Agent-tool pin) · planner **`opus`** (`derive-planner-lane.sh` → `opus fail-closed:planner-lane-field-missing`, verbatim) · executor **`codex:gpt-5.6-sol`** · evaluator **`sonnet`**, own worktree. generator≠judge holds.
- **Metered $0.50** of the $5 ceiling (2 quorum rounds + 3 restored-reviewer re-runs). Fable: 2 bounded designer runs on ONE doc — within the diet.

## Parked on Mark
**Nothing.** No open decision rows (ledger valid, 6 rows, 0 OPEN).
