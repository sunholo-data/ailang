### Added — Claude Sonnet 5.5 in the model registry (2026-09-29)

`claude-sonnet-5-5` ($2/$10 per 1M, cache reads 0.1x). It thinks adaptively by default, and
`{type:"disabled"}` returns 400, so it is registered as `CanDisable:false`. It is also in the
min-cacheable-prefix table (512) and the headroom ceilings. Standard placement: 28/29, anchored
ELO **2295.8**, $0.75, level with gpt6-sol (2288.7). Agent smoke+core on the subscription lane:
41/42. It is not in any eval suite yet.

**Executor rung 2 (Mark, attended).** When codex is dry, the executor now tries
`claude:claude-sonnet-5-5` before the pi chain, in both `roles.executor` and the mission driver.
The rung is probed through `_mc_probe`, which applies the Anthropic ration gate, so a drought or
an over-ration week still walks pi. Set `MISSION_EXECUTOR_ANTHROPIC_RUNG=''` to turn it off for
a mission. `resolve-role-spawn.sh` now compares model *families* for generator≠judge. Before
this, the `sonnet` evaluator would have judged a `claude:claude-sonnet-5-5` executor unnoticed.
The docs mission keeps its own executor chain, and world, a fork, is not changed.
