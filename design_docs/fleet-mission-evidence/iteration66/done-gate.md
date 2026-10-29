# Iteration 66: done-gate for `gate5:controlplane-send-refused-local-store`

Candidate head `ffde0469a` on `fleet/i66-controlplane-store-scope` (executor `60f60367c` + controller `ffde0469a`), base `138da7053`. Every command ran from the sprint worktree.

## Reproduction at base (controller, before routing)
- `ailang messages inboxes` (gcp store, registry `gs://ailang-multivac-ailang-config/config.yaml`): `TRIAGE controlplane … filed for a human, on purpose`.
- `AILANG_STORAGE_MESSAGING=local ailang messages inboxes` (registry `~/.ailang/config.yaml`): `! NOTHING controlplane … accepted and never dispatched`, so `checkSendInbox` (`cmd/ailang/messages_send_inbox_guard.go`) refuses the send.
- `~/.zshenv:38-39` exports the gcp pair, which is why the controller's Bash-tool sends have worked since the ticket was filed. The driver scopes its own sends per call (`mission-control.sh` notices).
- Audit of `.claude/skills/mission-control`: 4 unscoped prescribed commands (gate-5 send ×2, gate-5 3(c) `read --peek`, gate-3 `approvals` send); 3 already scoped by `gate-3-triage.md`'s `$MSG`.

## Gates
- `/bin/bash tools/launchd/sync-agents-skills.sh --check`: rc 0 at `60f60367c` and `ffde0469a`.
- `/bin/bash tools/launchd/test_agents_skills_sync.sh`: 5/0 at both heads.
- `make test-launchd-drivers`: rc 0 at `60f60367c` and at `ffde0469a` ("run-suites: all 49 suites passed"). The executor's own first run read rc 2 on three unrelated suites (`test_skill_sync.sh` git-timeout, `test_controller_capacity.sh`, `tools/eval/test_motoko_connection_probe.sh`), each green alone and green on its re-run; the controller's two runs were clean.

## Mutation drills (each applied, then restored with `cp` + `cmp`)
| Mutant | Pre-fix arm (`60f60367c`) | Final arm (`ffde0469a`) |
|---|---|---|
| D1 unscope the gate-5 `controlplane` send | FAIL (executor) | FAIL `gate-5-retro.md:48` |
| D2 unscope the gate-3 `approvals` send | FAIL (executor) | FAIL (judge D5) |
| D3 unscope the 3(c) read | FAIL (executor) | FAIL (judge D6) |
| D4 delete `MSG=` in gate-3-triage | FAIL (executor) | FAIL (judge D10/D17) |
| D8 scope pair only in a trailing comment | **PASS** (judge) | FAIL `gate-3-route.md:874` |
| D9b `ailang messages send` at end of line, args on the next | **PASS** | FAIL `gate-3-route.md:874` |
| D13 wrong project on `approvals` | — | FAIL `gate-3-route.md:829` |

Judge round 2 added D14–D17 (pair then `true;`, scoped `list;` then unscoped send, end-of-line `read`, undefined `$MSG`): all FAIL. Residual, not fixed: a double space or tab between `messages` and `send` is not matched.

## Dry-runs (`MISSION_PROFILE=fleet`, synthetic HOME, `AILANG_DRIVER_PINNED=ffde046`)
- Healthy: rc 0, `DRY RUN ok`, `lanes=DEGRADED(2)`: planner and executor codex probes `rc=75` (over daily ration, the same state this fire's driver recorded in `MISSION_ROUTING_NOTE`), handed to opus and `claude:claude-haiku-5-5`.
- Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i66`): rc 0, `DRY RUN ok`, `lanes=DEGRADED(2)`; quota admission refused the bogus codex lane before any probe, and a named fallback was recorded.
- The real `~/.ailang/state/mission-fleet.pid` sha was unchanged across both runs.
- The change touches no driver code; the dry-runs confirm the driver still boots and routes on this head.
