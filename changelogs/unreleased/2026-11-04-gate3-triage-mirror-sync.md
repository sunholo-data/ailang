### Fixed — dev CI red: `.agents` mission-control mirror resynced after gate-3-triage edit (2026-11-04)

Commit f22bf1f4b edited the canonical `.claude/skills/mission-control/resources/gate-3-triage.md` (+49 lines) but did not regenerate its `.agents` mirror. From that commit, dev's required checks `test`, `launchd drivers (bash 3.2)` and `Build ubuntu-latest` were red, because the mirror no longer matched the canonical file. The mirror is now regenerated with `tools/launchd/sync-agents-skills.sh`; no hand edits. Resolves harness ticket `ci:skill-mirror-not-synced-after-gate3-triage-edit`.
