# Iteration 89 — gates (controller, outside the sandbox)

## Base `4ea46f05e` (#2141 tip), measured before the directive was sent (false-green 4)
These 11 suites returned rc 0:
- `test_secrets_scope.sh`, `test_codex_controller_env.sh`, `test_controller_capacity.sh`, `test_mission_kill_tree.sh`, `test_mission_stall.sh`;
- `test_pi_controller_note.sh`, `test_quorum_artifacts.sh`, `scripts/test_mission_pi_run.sh`;
- `test_mission_pi_run_commits.sh`, `test_mission_pi_run_sandbox.sh`, `test_mission_pi_run_provider_quota.sh`.

`sync-agents-skills.sh --check` also returned rc 0.

**Controller gap:** the directive did not list `make test-launchd-drivers`. Because of that, the executor never ran `test_pi_ext_args.sh`, which the M2c prose broke (see below).

## Executor tree, first out-of-sandbox run
- All 11 suites above returned rc 0. That included the four pi-runner suites the executor had not run, and three it had labelled UNINFORMATIVE under the sandbox.
- Mutations: 68 killed, 0 survivors.
- shellcheck on new files: rc 0.
- `git diff --check`: rc 0.

## Defects found by the controller, all fixed before commit `7f1e13d5f`
1. **`$!` is not the controller PID.**
   - Measured on bash 3.2.57: `( export …; f || exit $?; sleep 7 ) &` gives `$!` = the bash subshell, with `sleep` as its child. A bare `sleep 7 &` gives `$!` = sleep.
   - Base launched claude as a bare command, so `_mc_stalled` (which skips its root and counts older descendants as long children) would have changed meaning for the primary lane.
   - Fix: `exec claude …` and `cd … && exec pi …`. Measured after the fix: `$!` is the controller in both shapes, and a refused apply still exits with its own rc (7 in the probe).
   - The kill-tree and stall fixtures were coupled to the old form. One faked `claude` as a shell function, which `exec` cannot run; the other grepped a literal. They now use PATH stubs, plus a new assertion that the pidfile names the controller process.
   - Mutation drill: with `exec` reverted in a scratch copy, the kill-tree test returns rc 1 (`not ok - controller pid is the controller process`). With `exec` in place it returns rc 0.
2. **`test_pi_ext_args.sh` red at head (1 failure plus an instrument failure), against 17/17 at base.** The M2c prose had replaced the raw pi sandbox snippet that the test pins. My first base control was invalid because the archive left out the skill dirs; the corrected control was 17/17. Fix: restore the snippet and its `--mode json` teaching, and add one line naming the runner as the preferred route. After the fix: 17/17.
3. **Repo-local scratch.** `test_secrets_scope.sh` and `test_secrets_scope_spawn.sh` exported `TMPDIR=<repo>/.snap/test-tmp`, a sandbox workaround that would leave files in any checkout. Fix: removed it from the first; the second now uses `mktemp -d` with a trap. After the fix, no `.snap/test-tmp` appears.

## Final tree `7f1e13d5f`
- `env -i … make test-launchd-drivers`: **54/54 suites** (`/tmp/iter89_ltd2.log`; excerpt in `launchd-drivers-final.log`). An earlier run raced the controller's edits and is void.
- `test_secrets_scope.sh`: 58 passed, 0 failed.
- `--mutations`: 68 killed, 0 survivors (`mutations-final.log`).
- kill-tree, stall, pi_ext_args, capacity, pi-note, codex-env and quorum_artifacts: rc 0.
- shellcheck: new files rc 0. The changed-file comparator against the parent over 14 files found no new findings.
- `sync-agents-skills.sh --check`: rc 0. The regenerated `.agents` mirror matched the executor's snapshot byte for byte before the snippet restore, and was regenerated again after it.
- `check-changelog`, `check-file-sizes`, `check-context-docs`, `check-no-personal-email` and `git diff --check`: all rc 0.
- Safety observation, with no content read or printed: the real `secrets.env` had the same sha256 and line count before and after every gate run, including both `make test-launchd-drivers` runs. `~/.ailang/state` entries: 491 before and 491 after the gate runs. The pre-executor count was not taken; iteration 88 recorded 496, and siblings write there.
- `git diff --check` and the closing-keyword scan of the PR body found 0 matches. Control: `Fixes #1` matched 1.
