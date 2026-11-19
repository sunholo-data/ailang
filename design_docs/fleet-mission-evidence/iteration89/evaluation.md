# Iteration 89 — diff judge (sonnet, Agent tool, foreground, own detached worktree at `7f1e13d5f`)

The judge reviewed `git diff 4ea46f05e 7f1e13d5f`. All five controller edits were disclosed to it beforehand: `exec` on the claude and pi children, kill-tree PATH stubs plus a pid assertion, the stall a7c pattern, the raw pi snippet restored, and the repo-local `TMPDIR` removed. 89,376 subagent tokens; 17 tool uses; 514 s.

## Verdict: PASS, 92/100, 0 blocking

| Row | Score |
|---|---|
| Tests Pass | 20/20 |
| Lint Clean | 8/10 |
| Acceptance Criteria | 28/30 |
| Code Quality | 13/15 |
| Documentation | 14/15 |
| Design Fidelity | 9/10 |
| Regression Surface / Performance | not triggered |
| **Total** | **92** (20+8+28+13+14+9) |

The judge's headline line first said 90. In the same message, it corrected that to the row sum of 92.

## What the judge re-ran at `7f1e13d5f`
These suites passed:
- `test_secrets_scope.sh`: 58/58.
- stall: 50.
- kill-tree: 8.
- pi-note: 36.
- codex-env: 18.
- capacity: 16.
- quorum_artifacts.
- `test_mission_pi_run.sh`: 36.
- `sync-agents-skills.sh --check`.

Counts match the parent, except that kill-tree gained the disclosed pid assertion.

## Probes, all confirmed
- The parent driver stays unscoped. A mutant applying the scope in the parent failed all three controller arms.
- The billing unset at `:171` is untouched.
- The codex marker exception is the single case label, and the value must be one of the six lanes. An empty marker is refused.
- A failed apply refuses the child (`|| exit $?`), and the driver sees it through `wait "$CONTROLLER_PID"`.
- `exec` keeps the log redirection and the watchdogs.
- `$!` is now the controller. The pi lane was already off before this change.
- The role wrapper derives its own root. The installed codex parses the appended `-c` flags.
- No fixture assertion was removed.
- All M2 arms in the plan are present.

## Non-blocking findings
1. A missing `exec` on the pi controller is caught only by `test_quorum_artifacts.sh`. Kill-tree and stall exercise only claude.
2. The `include_only`-warns mutant and the ignored `mc_codex_env_args` failure mutant are caught only by `test_quorum_artifacts.sh`, not by the spawn suite.
3. The codex controller is not `exec`'d. That is unchanged from base, because it was already a backgrounded function. A comment could say so.
4. The raw pi recipe snippet lacks `pi-role` and the `BASH_ENV` clear. This is documented, and the runner is the preferred route.
5. The codex `-c` flags are appended after the caller's arguments. They parse, but have not been tried in a real inference run.

## Judge mutation table (scratch copies)
All 18 mutants were killed, 4 of them only by a sibling suite:
- **Killed by the spawn suite:**
  - scope applied in the parent;
  - `*SECRETS*` exempted from the deny rule;
  - marker validation dropped;
  - apply failure ignored in each of the codex, claude and pi controllers;
  - `BASH_ENV` kept in the pi controller, and kept in the pi runner;
  - the runner ignoring apply failure;
  - the wrapper ignoring apply failure;
  - the wrapper taking its root from the caller;
  - the wrapper skipping env args;
  - a wrong claude lane name;
  - an unencodable root accepted.
- **Killed only by a sibling suite:**
  - no `exec` on claude (kill_tree, stall, quorum_artifacts);
  - no `exec` on pi (quorum_artifacts only);
  - the ignored `mc_codex_env_args` failure (quorum_artifacts only);
  - `include_only` warning instead of refusing (quorum_artifacts only).

## Strongest objection
Everything was exercised with fake providers and synthetic inventories. Nothing shows that a real controller still authenticates and reads its quota gauges once the scope is applied. M3's six dry-runs are deferred. So PASS means correct and safe M2 wiring, not a validated rollout.
