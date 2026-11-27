# Fleet mission dashboard — iteration 144

Updated 2026-11-27; scheduled unattended. [Bookkeeping #2445](https://github.com/sunholo-data/ailang/issues/2445).

- **Outcome: pick NONE [ADMIN].** 19 open tickets, the same set iteration 143 left, none routable without a ruling or outside fleet Authority. Independent judge (sonnet): **NONE, 88%**.
- **Dev: RED** at `9b35c44d0`, required `checks`. `make check-context-docs` fails on two links that the attended docs overhaul (`d03096c91`) left pointing at moved guides: `.claude/rules/coordinator.md` → `docs/docs/guides/coordinator-workers.md` and `.claude/skills/local-ollama-eval/SKILL.md` → `docs/docs/guides/evaluation/local-ollama.md`. The fix is outside fleet Authority and V1 owns dev reds, but V1's loop is disabled. `Build` (Cloud Build) reads "Couldn't read commit".
- **Open tickets:** 19. D-FLEET-44/45/46 gate the top ones; the rest are outside fleet Authority.
- **Routing:** controller opus. Judge sonnet on the Agent tool, read-only (87k tok). No designer, planner or executor (no pick). Codex is over its daily ration (probe rc 75).
- **Cost:** metered $0.
- **Parked on Mark:** D-FLEET-44, D-FLEET-45, D-FLEET-46 unchanged. D-FLEET-48 is moot; close it when convenient.
- **Next:** pick NONE unless a ruling lands or a new in-scope ticket arrives.
- **Evidence:** [log](fleet-mission-log.md) and [iteration144/](fleet-mission-evidence/iteration144/).
