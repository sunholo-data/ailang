### Added — pre-publication secret scan for mission evidence (fleet iteration 136)

- `scripts/mission_evidence_secret_scan.sh <path>...` scans files, directories and `.tar`/`.tar.gz`/`.tgz`/`.zip` archives (by member content) for the literal VALUES of the rig's credentials. It loads them in-process from `~/.config/ailang/secrets.env` (parsed, never sourced) and from credential-named environment variables, and never puts a value in argv or output. It proves its matcher on a loaded value before scanning (`control=ok`). Exit 0 clean, 1 leak (path, member and key name only), 2 instrument failure (no values, unreadable input, nested or corrupt archive, symlink). Suite `scripts/test_mission_evidence_secret_scan.sh` runs in `make test-launchd-drivers`.
- mission-control Gate 4 **Public publish** rule, pointed to from the top of `SKILL.md`: before any public bucket upload, public-repo evidence commit or issue/PR post, assemble from named files (never whole tool-output directories or `make -p`/`env` dumps), scan the exact bytes in their own tool call, and publish in a separate later call only on rc 0, `hits=0`, `control=ok`. Ticket `publish:evidence-pack:no-secret-value-guard` (stapledon iteration 113).

### Fixed — `.agents` sprint skill mirrors regenerated

- `.agents/skills/sprint-{planner,executor,evaluator}` were left stale by `b49dc1c59`, which reddened `tools/launchd/test_agents_skills_sync.sh` and `test_quorum_artifacts.sh` on dev. Regenerated with `tools/launchd/sync-agents-skills.sh`.
