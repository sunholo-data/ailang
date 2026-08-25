# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History lives in `v1-mission.md` (STATUS) and
`v1-mission-log.md`. Namespaced on purpose — never write the bare `mission-dashboard.md`.*

**Last iteration:** 275 · 2026-08-25 · `m-verify-targets-unwired` **LANDED** (`fde5ea067`, PR #878)
**Latest release:** v0.33.2 · dev CI green (21 checks on the merge, 4/4 required)

## In flight / next
- **`m-make-ci-red-ai-modes`** ← next. **`make ci` is RED at HEAD**, on one prerequisite:
  `examples/ai_modes.ail` fails effect checking (*"AI requires mode=fixed; declaration provides
  mode=routeable"*). `.claude/rules/dev-workflow.md:22` tells every agent to run `make ci` as
  "full CI verification locally", so this costs every agent, every day.
- `m-fmt-check-ail-broken-and-red` — unwired AND broken: its enumerator scans a `stdlib/` that has
  never existed (46 `.ail` files invisible), plus real drift, plus an `ailang fmt` crash.
- `m-cli-examples-fixture-rot` — 9 of 26 documented CLI commands fail; one may be a real regression
  (`list_sum` documented `(15, 15)`, produces `(15, 5)`).
- `m-verify-examples-trace-suppressed` · `m-gemini-verdict-score-threshold` ·
  `m-codex-streaming-test-flake`

## Loop cadence + routing
- launchd `dev.ailang.mission-control`, pinned worktree `~/.ailang-driver-pin/v1` at `origin/dev`.
- controller `opus` · executor `codex:gpt-5.6-sol` · evaluator `sonnet` (own worktree) ·
  designer rotation seeded `claude:claude-fable-5` (not spawned since iter-271 — direct-fix runs).
- metered spend iter-275: **$0.00** of $5. Quota buckets only.

## Parked on Mark — 4 OPEN decisions
- **`D-36` (new)** — an evaluator FAILED all 3 permitted rounds (63/45/38), every finding real;
  the last 2 were fixed by the controller with the judge's own repros as controls but **no
  independent 4th review**, and it LANDED anyway. Should a round-3 FAIL with mechanical findings
  **land-and-flag**, **park strictly**, or **raise the budget**? The loop cannot decide this.
- `D-30` — harness↔`ai-check` version coupling: schema / same-binary / accept.
- `D-31` — split the designer rotation into authoring vs review lanes (4 instances now).
- `D-32` — exempt `inconclusive` from the effective `cost_per_verified_success` arm?

## Health
- Bookkeeping issue **#852** (rotates Mondays 07:00 CEST; 13 comments, no rotation owed).
- Main checkout is **3 ahead / 7 behind** `origin/dev` on a *concurrent agent's* commits — not
  duplicates, so no reconcile is safe. The running skill therefore lags origin by one Gate-5 edit.
