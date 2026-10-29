# Iteration 65: done-gate for `planner-lane:files-section-parses-fenced-bullets` + `planner-lane:fenced-code-in-files-section`

Candidate head `c50fe663b` on `fleet/i65-planner-lane-fences`, base `58a41bad4`. Every command ran out of sandbox, from the sprint worktree.

## Reproduction at base (controller, before routing)
- `MISSION_PLANNER_MODEL=codex:gpt-6.1-sol`, default allowlist:
  - control doc: `codex declared:codex-ok`
  - plus a fenced example bullet: `opus fail-closed:path-not-in-codex-allowlist` (the ticketed fail-safe direction)
  - Files: infra bullet, a ```bash fence holding `# run this`, then `- internal/parser/parser.go`: **`codex declared:codex-ok`** (fail-OPEN; not in either ticket). Its control without the fence gives `opus fail-closed:path-not-in-codex-allowlist`.
- At head the same four docs give codex, codex, opus, opus.

## Gates
- `/bin/bash -n tools/launchd/derive-planner-lane.sh`: rc 0.
- `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR /bin/bash tools/launchd/test_mission_routing.sh`: base rc 0 108/0; round 1 (`c230c7555`) 115/0; head 122/0. Without `env -i` one pre-existing arm fails at base, because the driver env leaks `MISSION_PLANNER_ANTHROPIC_FALLBACK`. The 14 `_mc_ration_refuses: command not found` lines appear identically at base.
- `make check-changelog`: rc 0 at base and head.
- `env -i … make test-launchd-drivers`: rc 0 at `c230c7555` and rc 0 at `c50fe663b` ("run-suites: all 49 suites passed").
- The 14 pre-existing fixtures a–p give byte-identical stdout+stderr at base and head under a codex pin and a pi pin (28 comparisons, 0 diffs). The judge also compared the first 600 tracked `design_docs/*.md` files: 0 diffs.

## Mutation drills at head (controller; each applied, `cmp` proved it changed the file, then restored with `cp` + `cmp` 0)
| Mutant | Suite | Red arms |
|---|---|---|
| A: no fence skip in the path scan | 113/9 | q r s t u v w y z |
| B: no fence skip in the count scan | 121/1 | s |
| C: any 3+ run closes any fence | 119/3 | t u z |
| D: closer may carry trailing text | 121/1 | z |
| E: no 0–3 indent limit | 121/1 | za |
| F: backtick info string may hold a backtick | 121/1 | zb |
| G: two-character fences | 121/1 | zc |
| H: no CR strip | 120/2 | x y |
| I: no end-of-file sentinel | 121/1 | v |

## Dry-runs (`MISSION_PROFILE=fleet`, synthetic HOME, `AILANG_DRIVER_PINNED=c50fe66`)
- Healthy, attempt 1: rc 0, `lanes=DEGRADED(2)`. The codex probe returned `Selected model is at capacity` (a transient provider state, the D-FLEET-26 string), and planner/executor were handed to opus/haiku with named reasons. Attempt 2: rc 0, `lanes=ok`, `DRY RUN ok`.
- Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i65`): rc 0, `lanes=DEGRADED(1)`, `model_not_found`, executor handed to `claude:claude-haiku-5-5`.
- The real `~/.ailang/state/mission-fleet.pid` sha was unchanged across all three runs.
