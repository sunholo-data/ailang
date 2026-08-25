# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*

**Last iteration**: 276 · 2026-08-25 · controller `opus` · metered **$0.00** of $5

## Now
- **Latest release**: v0.33.2 · dev `f4828cc89` · **16** checks SHA-addressed, **zero** not-green
- **Just done (276)**: `m-make-ci-red-ai-modes` **ADJUDICATED → PARKED on `D-37`**. `make ci` is RED on
  1 of 27 prerequisites; the cause is `examples/ai_modes.ail`, the repo's only `mode=routeable` user.
  It is a **real regression** (green at its 2026-05-04 shipping commit) but **not** from the sprint it
  was blamed on — and it had already been parked for Mark on 2026-07-28 **in a commit message**, never
  as a ledger row. No code changed: the options are language-semantics rulings.
- **Next**: `m-fmt-check-ail-broken-and-red` → `m-ai-modes-regression-window`

## Parked on Mark — 5 OPEN decisions
| ID | One-line ask |
|---|---|
| **D-37** | **NEW** — may `AI[mode=routeable]` call `std/ai.call` (needs `mode=fixed`)? (a) register the subsumption edge · (b) make `call` mode-polymorphic · (c) migrate the example · (d) quarantine. **This is the sole cause of a red `make ci`.** |
| D-36 | Evaluator FAILs all 3 rounds but findings are mechanical — PARK or LAND? |
| D-32 | Should an `inconclusive` verification obligation be exempted from the cost-per-verified-success arm? |
| D-31 | Split the designer rotation into authoring vs review lanes (2 of 3 entries can't author)? |
| D-30 | How to enforce the harness↔`ai-check` version coupling before the `not_applicable` split? |

## Health
- **`make ci` is RED at HEAD** and has been since ~2026-07 — gated on `D-37`, no workaround applied
- Running mission-control skill is **1 commit behind origin** (`065a4f16c`); delta read before use.
  Main checkout is 3 ahead / 10 behind with a *concurrent agent's* unpushed work → reconcile refused
  (Gate-1 obligation 1 fails on `patch-id`). Not repairable by the loop.
- Observatory store **321 MB** vs a 200 MB warn threshold — prints on every `ailang` invocation
- Rotation: `#852` (created 2026-08-24), 15 comments — next rotation Mon 2026-08-31 07:00 CEST
- Weekly external-issue sweep: discharged for this rotation week by iteration 267

## Loop
- Cadence: launchd `dev.ailang.mission-control`, pinned worktree `~/.ailang-driver-pin/v1`
- Routing: controller `opus` · designer rotation `claude:claude-fable-5` (untouched, no doc this iter)
  · planner/executor `codex:gpt-5.6-sol` · evaluator `sonnet`
- Quota posture: no metered spend for 4 consecutive iterations
