# Fleet iteration 37 — evidence (2026-10-14)

All measurements are at origin/dev `42652c19eaf975860f4f91a12b191d24738def79`. The judge is sonnet (Agent tool, foreground), and it re-measured each claim with its own commands.

## 1. Dev is red on required checks, so PR #1824 cannot land (resume predicate still false)

- Branch protection on `dev` requires these contexts: `test`, `lint`, `build`, `docs-gate`, `UI build gate`. The judge read the list.
- `test` (job 115510969327, 135 KB log) fails at `make check-git-exec`:
  `git exec site absent from baseline: internal/executor/managed_agents/devapi_workspace.go (2)`.
  The two sites are `exec.CommandContext(ctx, "git", …)` at `devapi_workspace.go:237` and `:259`, from `ffef3402a`. The gate wants these calls to go through `internal/gitexec`, or the baseline `scripts/git_exec_baseline.txt` to list them.
- `docs-gate`/`docs-build` (job 115511110382, 74 KB log) fail with `MDX compilation failed … docs/docs/reference/env-vars.md`, `Could not parse expression with acorn`, at 252:256. That line contains the literal `'{"sort":"throughput"}'`. `tools/gen-env-docs/main.go` generates the line from `internal/config/ai.go:43` (judge).
- `Build windows-latest` (job 115510965954; judge) fails `TestPlanWorkspace_LocalDeltaCoversEverythingAndSparesIndex` and `TestDevAPI_ExecuteRoundTrip` in `internal/executor/managed_agents`. This check is not required.
- **Fleet scope.** `scripts/git_exec_baseline.txt`, `internal/executor/**` and `internal/config/ai.go` are all refused by `tools/launchd/githooks/pre-push`. The `docs/**` page itself is in scope, but it is generated. Fixing it by hand would edit the mirror, not the source, and the next regeneration would revert it.
- PR #1824 is `BLOCKED`. Its failing set (`test`, `docs-build`, `docs-gate`, `Build windows-latest`) equals dev's. Iteration 36's resume predicate is still false.
- V1 owns repo reds, and its kill switch (`mission-control.disabled`) has been set since 2026-09-14. No open PR fixes either red.

## 2. Ticket `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang` — reproduction check

**Window.** The window covers CI workflow runs (id 193349605) created at or after 2026-09-28T13:44Z.
- The controller's listing hit the 1000-run search cap, so it reaches back only to 17:01Z.
- The judge's listing of failed runs (385, no cap) reaches back to 13:08Z.
- A failed launchd job always fails its CI run, so scanning failed runs finds every failed launchd job in the window.

**Counts.** Across the failed runs, the `launchd drivers (bash 3.2)` jobs split as follows:

| Source | Failure | Success | Skipped |
|---|---|---|---|
| Controller | 19 | 190 | 163 |
| Judge, window from 13:08Z | 21 | 200 | 164 |

The judge's two extra failures fall in 13:08–17:01Z.

**Signatures.**

| Signature | Source | Occurrences |
|---|---|---|
| B `FAIL: production: hanging gh comment is cut off and warns` | `tools/launchd/test_driver_notify.sh` | Last FAIL 2026-09-28T13:15Z (job 108941820368, dependabot branch; judge). That is before the ticket was filed. 0 since. The PASS line appears in every later failed job, and 6 jobs from successful runs are clean (judge). |
| A `INSTRUMENT FAILURE: treatment peer set contains OpenRouter endpoint: 203.0.113.8:443` | `tools/eval/motoko_connection_probe.sh:79` | Job 108956303787 (09-28 13:44Z, dev) and job 110985987146 (run 37051534362, 2026-10-02, dev). None after 10-02. |
| NOT TICKETED: `not ok - bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 …)` | `tools/eval/test_motoko_connection_probe.sh:1136` | 4 on dev: jobs 112880823770 (10-07), 114188876102 (10-10), 114764327101 (10-12), 114869478423 (10-12). This is the live launchd-suite flake. |
| Other | — | 12 jobs failed deterministically on 2026-10-02 because the `.agents` sprint-evaluator mirror was stale (since fixed). 2 jobs failed `account-workspace-subscription-isolation` on branch `fix/codex-oauth-profiles`, which is not dev. |

**Controls.** Every zero is paired with a known-present line in the same log (`passed=`, `ok `). Log sizes ran from 20 KB to 120 KB.

**Scope.** `tools/eval/**` is refused for `MISSION=fleet` (`_scope_verdict`: "outside the fleet allowlist"). The suite still runs inside `make test-launchd-drivers` (`make/test.mk:117`), which is the fleet's own done-gate. So the two live flakes in the fleet's done-gate suite are in files the fleet cannot edit. The motoko loop, which owns that probe, has been disabled since 2026-09-08.

## 3. Dispositions

- **#12**: still parked on the dev red. This is an ownership block: it is not a lane-capacity park, and no judgment is pending. It resumes when the `test` and `docs-gate` predicate turns green. D-FLEET-17 asks how to unblock it.
- **#13**: kept OPEN. B is not reproduced since filing, but A is live, as is the untracked timer flake, and both are out of scope. D-FLEET-18 asks about the authority.
- Judge verdict: **PASS 88**. Deductions: the controller first claimed that the `docs/` path was outside scope (only its source is), omitted the Windows red, and over-stated the window caveat.
