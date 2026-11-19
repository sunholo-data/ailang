### Changed — Fleet scope guard admits the design-doc-creator skill (D-FLEET-36, ruled A 2026-11-16)

The fleet support allowlist (`tools/launchd/githooks/pre-push`, `_scope_is_fleet_support`) now admits
`.claude/skills/design-doc-creator/**` and `.agents/skills/design-doc-creator/**`, per Mark's attended
ruling D-FLEET-36 = A (2026-11-16). They are support paths, not harness, so product missions keep
pushing them; every other `.claude/skills` path stays refused for the fleet. Guard arm, Authority line
and regression arms only; under the iteration-16 rule the skill's eight `grep -E "^\d+\."` sites change
in a separate fleet change once this lands on the pinned `origin/dev` guard.
