### Changed — Fleet scope guard admits 16 named `.agents` skill mirrors (D-FLEET-38, ruled A 2026-11-16)

The fleet support allowlist (`tools/launchd/githooks/pre-push`, `_scope_is_fleet_support`) now admits
exactly 16 enumerated `.agents/skills/<name>/**` trees, per Mark's attended ruling D-FLEET-38 = A
(2026-11-16): benchmark-runner, builtin-developer, cloud-cascade-debug, collaboration-hub,
coordinator-helper, eval-analyzer, eval-gap-finder, gemini-cli-helper, github-issue-triage,
local-model-onboarding, local-ollama-eval, parser-developer, pr-monitor, prompt-manager,
trace-debugger and use-ailang. Each differs from its `.claude/skills` source only by the 2026-07-28
claude-to-Codex renames, or (collaboration-hub, trace-debugger, since 2026-11-10) not at all. They
are support paths, not harness, so product missions are unaffected; there is no `.agents/skills/*`
wildcard and no `.claude/skills` path. Guard arm, Authority lines and regression arms only; extending
`tools/launchd/sync-agents-skills.sh` to these skills is a separate fleet change once this lands on
the pinned `origin/dev` guard (iteration-16 rule).
