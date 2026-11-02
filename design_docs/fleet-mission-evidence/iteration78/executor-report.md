```text
 .../mission-control/resources/gate-3-route.md      |   2 +-
 tools/launchd/resolve-role-spawn.sh                |  30 +++++
 tools/launchd/test_mission_routing.sh              | 122 +++++++++++++++++++++
 3 files changed, 153 insertions(+), 1 deletion(-)
```

The untracked changelog fragment adds 3 lines. Including it: **4 files changed, 156 insertions(+), 1 deletion(-)**.

**G1:** rc **0**, no syntax diagnostics, using macOS Bash 3.2.57:

```bash
/bin/bash -n tools/launchd/resolve-role-spawn.sh && /bin/bash -n tools/launchd/test_mission_routing.sh
```

**G2–G4:** each returned rc **0**, with identical summary: `==== 146 passed, 0 failed ====`.

```bash
# G2
env -i HOME="$HOME" PATH="$PATH" TMPDIR="$PWD/.tmp-i78/" /bin/bash tools/launchd/test_mission_routing.sh

# G3
env -i HOME="$HOME" PATH="$PATH" TMPDIR="$PWD/.tmp-i78/" MISSION_ANTHROPIC_AVAILABLE=0 /bin/bash tools/launchd/test_mission_routing.sh

# G4
TMPDIR="$PWD/.tmp-i78/" /bin/bash tools/launchd/test_mission_routing.sh
```

`TMPDIR` was explicitly supplied to keep all suite scratch files inside the worktree. Suite return codes were captured before reading each log’s final line. An additional run with conflicting exported mission variables also passed **146/0**.

**G5:** all **12 direct resolver calls** passed, covering the ten requested behaviors and availability controls. Every command and output was printed during execution; each returned rc **0**, exactly one stdout line, and no stderr.

| Arm | Observed stdout |
|---|---|
| R15 | `reroute pi:openrouter/minimax/minimax-m3 anthropic-unavailable` |
| R16, availability=1 | `agent-tool sonnet declared:alias-pin` |
| R17, availability unset | `agent-tool sonnet declared:alias-pin` |
| R18 | `reroute pi:ollama/minimax-m3:cloud anthropic-unavailable` |
| R19 | `reroute pi:ollama/z:cloud anthropic-unavailable` |
| R20, aliases/Claude only | `refuse fail-closed:anthropic-unavailable-no-fallback` |
| R21, fallback unset | `refuse fail-closed:anthropic-unavailable-no-fallback` |
| R22 | `reroute pi:ollama/deepseek-v4.1-flash:cloud anthropic-unavailable` |
| R23 | `reroute codex:gpt-6.1-sol anthropic-unavailable` |
| R24, generator collision | `reroute pi:openrouter/b/c anthropic-unavailable` |
| R25, whitespace | `reroute pi:ollama/a:cloud anthropic-unavailable` |
| R26, provider pin | `recipe codex:gpt-6.1-sol declared:provider-pin` |

**G6:** mutation runner rc **0**: **6/6 mutants killed**. Each mutant suite returned rc **1**, using scratch copies with `ROOT` preserved and `RESOLVE` redirected:

```bash
env -i HOME="$HOME" PATH="$PATH" TMPDIR="$PWD/.tmp-i78/" /bin/bash ".tmp-i78/$mutant/test_mission_routing.sh"
```

| Mutant | Killing arms | Passed / failed |
|---|---|---|
| m1: remove new block | R15, R18–R25, R27–R32, R35, R37–R38 | 128 / 18 |
| m2: remove Claude skip | R18, R20, R28, R32 | 142 / 4 |
| m3: remove ration skip | R19, R30, R37 | 143 / 3 |
| m4: remove evaluator collision skip | R24, R27, R31, R35 | 142 / 4 |
| m5: availability `!= 1` | R2, R3, R4, R4b, R4c, **R17**, R33, R34 | 138 / 8 |
| m6: remove whitespace trim | R25, R32 | 144 / 2 |

Untouched-base findings: clean and inherited environments passed **122/0**; inherited availability=0 reproduced **113/9**, rc **1**. The base suite also printed missing-helper diagnostics for `_mc_canon_id`, `_mc_is_demoted`, `_mc_ration_refuses`, `_mc_export_mission_registry`, and `_mc_spawn_over_ration` despite passing. Those extracted-controller-test limitations remain unchanged.

`git diff --check` passed. Only the four permitted files remain changed; the documentation edit is exactly one sentence. `.tmp-i78/` was deleted, all started processes finished, and no git writes were performed. Changes remain uncommitted for review.