Follow-up to #2413 (the scope-guard arm, merged as `55d336df7`). Carries out the two attended rulings of 2026-11-16:

- **D-FLEET-36 A** (`skill:design-doc-creator-grep-portability`): the 8 `grep -E "^\d+\."` sites in both `create_planned_doc.sh` copies become `^[0-9]+\.`. GNU grep reads `\d` as a literal `d`, so those lines selected nothing on Linux.
- **D-FLEET-38 A** (`skills:agents-copies-stale-non-mission`): `sync-agents-skills.sh` gains exactly the 16 rename-only or identical skills the ruling names, and their `.agents/skills/*/SKILL.md` mirrors are regenerated. `test_agents_skills_sync.sh` pins the exact list and refuses `\d` coming back in either design-doc-creator copy.

It could not ship in #2413: the guard that judges a push is the pinned `origin/dev` copy, which did not admit these paths until #2413 merged (fleet charter, iteration-16 rule). The pinned guard at `18e848db6` admits all 19 paths for `fleet` (dry-run rc 0; a `v1` control refuses the two `tools/launchd` scripts, rc 1).

Built and judged in fleet iteration 121: planner opus, executor haiku, judge sonnet **PASS 93**, 0 blocking. Evidence: `design_docs/fleet-mission-evidence/iteration121/` on the record branch (#2417). Landed in fleet iteration 122, with a fresh independent review of the pushed head.

Tickets close after merge, once the done-gate runs on `origin/dev`.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
