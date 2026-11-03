Implemented both PROPOSED changes. C1 and C2 also pass the scope suite independently.

Both changes modify:

- `tools/launchd/githooks/pre-push`
- `tools/launchd/test_mission_scope_guard.sh`
- `design_docs/fleet-mission.md` — Authority only

C1 creates `changelogs/unreleased/2026-11-03-fleet-scope-arm-dfleet36.md`; C2 creates `changelogs/unreleased/2026-11-03-fleet-scope-arm-dfleet38.md`.

Full snapshots are saved under `.snap/C1/` and `.snap/final/`.

| Final gate | Result |
|---|---|
| Hook `bash -n` | PASS, rc 0 |
| Test `bash -n` | PASS, rc 0 |
| `git diff --check` | PASS, rc 0 |
| Mission decisions check | PASS, rc 0 |
| Scope guard suite | PASS, rc 0 |
| `make check-changelog` | PASS, rc 0 |
| `make check-context-docs` | PASS, rc 0 |
| `make check-no-personal-email` | PASS, rc 0 |

All four mutations were killed, each returning rc 1:

| Mutation | Failing assertion |
|---|---|
| m1: delete use-ailang arm | Fleet `.agents/skills/use-ailang/SKILL.md`: refused instead of allowed |
| m2: broad `.agents/skills/*` | Fleet `.agents/skills/design-doc-creator-x/SKILL.md`: allowed instead of refused |
| m3: broaden creator prefix | Fleet `.claude/skills/design-doc-creator-x/SKILL.md`: allowed instead of refused |
| m4: move creator into harness | V1 nested creator script: refused instead of allowed |

C2 alone also kills m2 at `.agents/skills/model-manager/SKILL.md`.

Final `pre-push` matches its snapshot via `cmp`. Harness/verdict functions remain unchanged. No scope deviations; test temporary files were redirected under `.snap/tmp/` to keep writes in-tree. No gates were sandbox-uninformative.