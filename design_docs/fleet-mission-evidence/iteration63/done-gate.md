# Done-gate (iteration 63, out of sandbox, at bfd312a8a)

- Base reds reproduced at `5f56181a1` (controller, out of sandbox): `sync-agents-skills.sh --check` rc 1, `test_agents_skills_sync.sh` rc 1, `test_quorum_artifacts.sh` rc 1, `go test ./internal/mission/quorum/ -run TestQuorumArtifactShellSuite` rc 1 (all `STALE: .agents/skills/sprint-evaluator`); `make check-changelog` rc 0.
- At `bfd312a8a`: the same four rc 0, `go test ./internal/mission/quorum/ -count=1` ok, `make check-changelog` rc 0.
- `env -i HOME PATH USER TMPDIR make test-launchd-drivers`: rc 0; 1113 PASS lines, 0 FAIL; `run-suites: all 49 suites passed`; `launchd drivers: tests + bash 3.2 syntax OK`.
- Dry-run healthy, `MISSION_PROFILE=triage` (idle, armed, no pidfile before or after), `AILANG_DRIVER_PINNED=bfd312a8a`: rc 0, `lanes=ok`.
- Dry-run degraded, plus `MISSION_EXECUTOR_MODEL=codex:bogus-model-i63`: rc 0, `lanes=DEGRADED(1)`, `codex model 'bogus-model-i63' unusable: probe failed (rc=1)`, executor handed to `claude:claude-haiku-5-5`.
- Controller replay of judge mutant 1 (`.agents` sprint-evaluator reverted to base content): `test_agents_skills_sync.sh` rc 1; restored by `cp`, `cmp` 0, post rc 0; judge tree clean.
