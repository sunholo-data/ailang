# Fleet iteration 91: controller gates, run outside the sandbox

## Base `01baf90c2` (untouched origin/dev)
- `make test-launchd-drivers`: 52 of 53 suites passed. The failure was `test_cloud_branch_sync.sh` arm A20 (`assertion failed: test 5 -le 4`, a deadline check).
- That run overlapped the codex executor's work in the same worktree, so it ran under load.
- The same suite passed on both later head runs (28 s). It is load-sensitive, not a defect, and not related to this change.

## Head `635ea94da` (executor build plus controller store pin)
- `make test-launchd-drivers`: all 54 suites passed.
- `make check-context-docs check-changelog check-tmpfile-hygiene check-home-isolation check-skills`: rc 0.
- `git diff --check`: rc 0.
- `test_mission_idle_fingerprint.sh`: PASS. `--mutations`: 8/8 killed, including the controller's `store-pin` mutant.
- shellcheck: 0 findings on the new files. The driver had 30 findings at base and 30 at head.

## Head `33190af48` (round-2 fix)
- `make test-launchd-drivers`: all 54 suites passed.
- The same check targets: rc 0. `git diff --check 01baf90c2 HEAD`: rc 0.
- `test_mission_idle_fingerprint.sh`: 3 PASS groups. `--mutations`: 12/12 killed.
- shellcheck: 0 findings on the new files after two fixes (SC2209 and SC2015). The driver had 30 findings at base and 30 at head.
- `test_mission_triage_idle.sh`: PASS.
- Real `~/.ailang/state`: 495 entries before and after the suites, and 0 idle-fingerprint files. This was checked by observation; nothing was seeded.

## Seam defect caught by the controller, before the judge
- `AILANG_STORAGE_MESSAGING=gcp` comes only from `~/.zshenv`.
- The controller's zsh tool shell runs `--record` and so would read the gcp store.
- The launchd bash driver runs the pre-check and would read local SQLite.
- The hashes would then never match, and the feature would never fire. That failure is safe but useless.
- The fix pins the store inside the helper, as the driver's own sends do (driver line 248). It has a test arm with a hostile caller store and a `store-pin` mutant.

## Done-gate dry-runs
- Profile: `world`, idle and armed (no kill switch, no pidfile).
- Environment: `env -i`, `AILANG_DRIVER_PINNED=33190af48`, `AILANG_WORKTREE_SWEEP=0`, stdin `/dev/null`.
- Healthy: rc 0, `DRY RUN ok`, roles as configured.
- Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i91`): rc 0. The probe returned 400, the executor fell back to `claude:claude-haiku-5-5`, and the run printed `DRY RUN ok`.
- Unchanged before and after: the pin-dir listing, the world pidfile, the idle files and the sweep-log hash.
- A dry run never reaches the snapshot block, and the pre-check skips dry runs by design.

## Live parity against World (read-only, scratch `IDLE_STATE_DIR`)
- `--snapshot` under a launchd-like `env -i` bash (no messaging-store env) gave `c099527db06f…`, with head e853ca3, ledger 96, open 1, directives 0, inbox 54, red 0.
- `--record` promoted it, and a bare read under zsh (store gcp) gave the identical hash.
- The judge's round-1 independent reading was also `c099527d…`.
