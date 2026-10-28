# M-SWEEP-ACCOUNT-HOME — fleet iteration 67

Status: planned. Ticket: `pin-root:worktree-removed-mid-fire` (stapledon iteration 55, filed 2026-10-28T15:27Z; blocking=none).
Authority: controller-approved direction in the iteration-67 planner task; this artifact plans implementation only.
Goal: protect real account pins when SessionStart runs under an overridden HOME, and disable sweep side jobs in dry-run recipes.
Estimate: one session, about 3 hours including mutation drills; M1 ~70 changed lines, M2 ~20. Estimates reflect the inspected small scripts, not a measured LOC/day rate.

## Problem and measured evidence

Accept the controller's first-party root cause; do not repeat the destructive reproduction.
1. Iteration 66 invoked `env -i HOME=/tmp/fleet_i66_home.3S7K ... MISSION_PROFILE=fleet MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh` at 2026-10-28T14:50:33Z. The recipe linked tool configuration but neither real state nor real pins ([Guardrails](../fleet-mission.md#guardrails-on-top-of-the-skills-standing-rules)).
2. Lane probes started real Claude sessions. `env -i` removed unattended markers, so the [SessionStart hook registration](../../.claude/settings.json) invoked [git_health.sh](../../scripts/hooks/git_health.sh) without its mission/task early exit.
3. The hook's HOME-relative stamp was absent, triggering a real background sweep. The [sweeper](../../scripts/worktree_sweep.sh) derives both protection defaults and its log from HOME (lines 53 and 42 respectively).
4. Clean, detached real pins on origin/dev, idle >2h with no process cwd inside, were consequently eligible for removal. Controller evidence: `/tmp/fleet_i66_home.3S7K/.ailang/state/worktree-sweep.log` records removals of `/Users/voightkampff/.ailang-driver-pin/stapledon` at 15:52:00, `triage` at 15:52:08, and `world` at 15:52:24 local CET (14:52Z). The real account log has no corresponding lines.
5. The same controller measured `HOME=/tmp/x perl -e 'print((getpwuid($<))[7])'` returning `/Users/voightkampff`. Sources for timestamps and log excerpts: the controller's measured-evidence items 1–6 in this ticket handoff, not a new planner experiment.

The iteration-66 STATUS paragraph's earlier non-attribution is superseded by this measured ticket evidence; M2 edits the Guardrails recipes, not historical STATUS prose.
Baseline gates were measured GREEN by the controller (hook self-test: 9 passed, rc 0); this planner does not claim to have rerun them.

## Direction and boundaries

- M1 owns only `scripts/hooks/git_health.sh` and `scripts/hooks/test_git_health.sh`.
- M2 owns only `design_docs/fleet-mission.md` and `changelogs/unreleased/2026-10-28-sweep-account-home.md`.
- The fleet scope guard forbids editing `scripts/worktree_sweep.sh` and `scripts/test_worktree_sweep.sh`. Reading them is allowed; never invoke a real sweep with `--apply`.
- No git writes, real-HOME writes, inbox actions, reloads, or implementation are part of this planner task. Executor/controller routing remains with the fleet controller.
- Registry reuse: M1 `none` (existing local hook and shell test); M2 `none` (charter and changelog). Neither is a package-like capability; registry search is inapplicable. No dependencies or `.ail` examples are added.

## M1_HOOK_ACCOUNT_HOME — hook protection and regression tests

Dependency: none. Budget: ~2 hours, ~70 changed lines including tests.
1. Within the existing eligible-sweep branch, before stamp touch/background launch, resolve shell variable `ACCT` using `perl -e 'print((getpwuid($<))[7])'`, suppressing lookup diagnostics. Require successful resolution and an existing directory; empty, failed, or nonexistent results skip the sweep and still exit 0. Never fall back to HOME.
2. Pass a command-scoped `AILANG_SWEEP_PROTECT` to the detached sweep, beginning with `$ACCT/.ailang-driver-pin/:$ACCT/.ailang-ab/:$ACCT/.ailang-nightly/:$ACCT/.ailang/state/worktrees/`, then `:` plus any nonempty caller-supplied `AILANG_SWEEP_PROTECT`, preserving its contents. Introduce no new environment variable; `ACCT` is an unexported shell variable.
3. Retain HOME for stamp/log destinations, existing opt-out and unattended exits, throttling, asynchronous launch, bash 3.2 compatibility, always-zero exit, and the registered 5s hook budget. Add one or two header sentences explaining the 2026-10-28 synthetic-HOME deletion incident and account-home protection.
4. Keep the test's exported fake HOME and throwaway repos. Extend its stub sweep to retain the separate `ran` counter and also record the received protection string in a separate throwaway file. It must never execute the real sweep.
5. Add exactly three `ck` arms, retain the original nine, and change the final pass-count assertion to 12:
   - A: immediately after the first launch, compute account home by the same perl call under fake HOME; assert the recorded list contains the account-home pin prefix (preferably all four prefixes). Do not derive the expectation from HOME.
   - B: after the original nine checks, remove only the fake stamp, invoke with a caller sentinel such as `$T/caller-protect/`, and assert it survives after the account prefixes in the received list.
   - C: remove only the fake stamp again, put an executable `perl` stub that exits 1 first on PATH for this invocation, and assert the sweep count does not increase and hook rc remains 0 (one combined `ck`). Restore PATH afterward. The missing stamp ensures failure reaches account lookup rather than being blocked by freshness testing.

Acceptance commands (controller can run outside the sandbox; stub-only tests):

| Exact command | Expected result |
|---|---|
| `/bin/bash scripts/hooks/test_git_health.sh` | rc 0; final `12 passed, 0 failed`; original nine plus A/B/C green |
| `/bin/bash -n scripts/hooks/git_health.sh` | rc 0, no output |
| `shellcheck -S warning scripts/hooks/git_health.sh scripts/hooks/test_git_health.sh` | rc 0, no warnings |

Review also confirms four account prefixes, unchanged caller suffix, directory/status guard, incident header, and unchanged 5s registration; no real sweep is a timing test.

Mutation drills: change only a disposable copy of the hook and its sibling test under this worktree; run `/bin/bash scripts/hooks/test_git_health.sh` from that copy. Restore the unmutated copy between drills; no git restore/reset and no real sweeper.

| Arm | Named mutation | Required red result |
|---|---|---|
| A | `HOME_PREFIXES`: replace account prefixes with fake-HOME prefixes | A prints FAIL and test exits nonzero |
| B | `DROP_CALLER_PROTECT`: omit the caller suffix from the launch environment | B prints FAIL and test exits nonzero |
| C | `LOOKUP_FAIL_OPEN`: bypass the account lookup success/directory guard so failed lookup still launches the stub | C prints FAIL (counter increased) and test exits nonzero |

Record each named FAIL and rc, not merely a syntax error or a red total caused by a different arm. Unmutated rerun must return `12 passed, 0 failed`.

## M2_RECIPE_CHANGELOG — safe dry-run recipes and release note

Dependency: M1. Budget: ~1 hour including final gates, ~20 changed lines.
1. Add `AILANG_WORKTREE_SWEEP=0` inside BOTH Guardrails `env -i` commands: idle armed sibling and synthetic-HOME fleet. The current synthetic recipe refers to the earlier command; expand it into an explicit second command with synthetic HOME, existing PATH/USER/TMPDIR, pin SHA, fleet profile, and dry-run flag so neither recipe relies on an implicit setting.
2. Add one sentence: a dry-run must start no side jobs; the 2026-10-28 synthetic-HOME dry-run deleted three live pins. Retain the existing synthetic configuration symlink instructions and state isolation.
3. Create `changelogs/unreleased/2026-10-28-sweep-account-home.md`, following the [fragment convention](../../changelogs/unreleased/README.md). Open with `### Fixed — Synthetic-HOME sessions could delete live mission pins (2026-10-28)`; explain account-home protection, fail-closed lookup, dry-run opt-out, and link this plan. No new root changelog edit.

Acceptance commands:

| Exact command | Expected result |
|---|---|
| `rg -c 'AILANG_WORKTREE_SWEEP=0' design_docs/fleet-mission.md` | rc 0; `2` |
| `rg -n -A 4 'env -i' design_docs/fleet-mission.md` | shows both executable recipe commands carrying the opt-out, sibling/fleet profiles and existing dry-run/pin flags |
| `head -n 1 changelogs/unreleased/2026-10-28-sweep-account-home.md` | exactly the `### Fixed — ...` heading specified above |
| `make check-changelog` | rc 0; fragment naming/heading checks pass |
| `make test-launchd-drivers` | rc 0; ends `launchd drivers: tests + bash 3.2 syntax OK`; hook suite reports 12 passes |

Final validation includes rerunning all three M1 commands. Do not require a live lane-probe dry-run or launch a real sweep to validate this patch.

## Real-state acceptance: OBSERVATIONS only

Controller records these read-only observations before and after gates; do not seed, touch, truncate, redirect into, or recreate real state. Use account home rather than the possibly synthetic HOME:

```bash
ACCT=$(perl -e 'print((getpwuid($<))[7])')
shasum -a 256 "$ACCT/.ailang/state/worktree-sweep.log"
wc -l "$ACCT/.ailang/state/worktree-sweep.log"
ls -1 "$ACCT/.ailang-driver-pin"
```

Expected OBSERVATION: log SHA/line-count and pin listing unchanged across these isolated gates. Record an absent path as absent; never create it. If concurrent fleet activity changes observations, report it separately rather than manufacturing a clean comparison. The planner does not perform or claim these controller observations.

## Risks and rejected alternatives

- Residual outside fleet scope: under synthetic HOME the sweeper still logs any OTHER removal to that throwaway HOME. Prefix protection fixes account pins, not global log visibility; name this residual and leave it unfixed.
- Editing the sweeper or its test is outside fleet authority. Reusing its existing protection input is sufficient and reviewable.
- Skipping every sweep whenever HOME differs from account home makes the fake-HOME self-test unable to exercise a positive launch without writing the real stamp; reject that destructive test requirement.
- Resetting HOME to account home for the sweep similarly defeats isolated testing and writes real stamp/log state from sandbox sessions; reject it.
- A missing/broken perl or invalid account directory disables sweeping, intentionally failing closed. Preserve quiet always-zero hook behavior.
- Detached stub scheduling can race assertions; retain existing wait behavior and ensure each counter/protect record is read after the stub completes. No waits belong in the production hook.
- Retain caller colon-separated prefixes verbatim and quote paths; use no bash-4 syntax. Controller runs the gates outside sandbox restrictions; no planner claim of gate success substitutes for execution.
