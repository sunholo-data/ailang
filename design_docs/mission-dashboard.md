# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-13 ~05:20 local (iteration 189)

## Now
- **v0.33.0** · `dev` @ `fc7fc67b4`, CI green (SHA-addressed `checks=16`, `pending=0`, zero not-green;
  `CI` + `Build and Release` both `success`).
- ✅ **`#665` FIXED + LANDED** — a genuine regression that outranked the human-gated queue.
  `tools/launchd/nightly-eval.sh` never sourced `secrets.env`, so `OPENROUTER_API_KEY` was absent in
  the non-login launchd env → local ollama models failed the motoko canary → `eval-suite` exited
  "No models support agent evaluation" → the nightly banked **ZERO rows**. Fix mirrors the two
  sibling launchd scripts (`os-rotation-filler.sh:22-25` carries the identical rationale). `#665` closed.
- ⚠️ **Not-yet-confirmed end-to-end**: mechanism proven (launchd-shaped env → key SET, value never
  printed; `bash -n` clean; delivery path is the main clone the plist runs + os-rotation-filler's
  45-min `git pull`), but the full bank confirms only at the next **03:00** run.
- ⚠️ **`#649`** stays open — a sustained *capability gap* on `log_file_analyzer` (opencode-qwen local),
  NOT an instrument regression; not this iteration's pick.

## Next
1. Confirm the 03:00 nightly banks rows (self-verifies; no action unless still zero).
2. Queue top (`#558`/`D-13` and `D-1`..`D-12`) is **human-gated end to end** — nothing routable
   without a Mark directive. Next unblocked non-human item: `[SWEEP iter-158]` external-issue triage
   batch (P3, ~0.5d), then `m-dialect-keyword-diagnostics` (`#539`, NEW-DOC + quorum).

## Loop health
- 5 consecutive pinned fires; zero dead slots since iter-184. Held the turn open with chained bounded
  CI polls (Standing rule 7). Gate-1 skill `cmp` SILENT (running skill == `origin/dev`); local==origin.
- Routing: controller **opus** only — a one-line infra fix needs no design doc, so no
  designer/planner/executor/evaluator fired. `metered=$0.00` of the `$5` ceiling.

## Parked on Mark — all on issue #635
`D-1` (#613) · `D-2` (#604/#614) · `D-7` (pi/codex executor) · `D-8` (#618 rig rollout) ·
`D-9` (#619 scope) · `D-10` (#616 fix site) · `D-11` · `D-12` (auto-row on human unblock) ·
`D-13` (#558 scope, one word A/B/C). No new decision item this iteration.

## Quota posture
Subscription buckets only (controller opus); zero metered spend this iteration. Billing tripwire
CLEAN (re-checked inside the tool shell, not just at preflight).
