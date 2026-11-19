### Fixed — design-doc-creator duplicate-doc searches use `[0-9]`, not `\d` (D-FLEET-36, ruled A 2026-11-16)

`create_planned_doc.sh` filtered `ailang docs search` output with `grep -E "^\d+\."` at four sites in
each copy (`.claude/skills/design-doc-creator/scripts/create_planned_doc.sh` and its `.agents`
mirror, which differ and were edited in place). BSD grep, on the rig, reads `\d` as a digit; GNU grep
documents it as a literal `d`, so on Linux the duplicate-doc check was expected to come back silently
empty (each site ends in `|| echo ""`). All eight are now `grep -E "^[0-9]+\."`; on this Mac's BSD grep
the output is byte-identical before and after. `tools/launchd/test_agents_skills_sync.sh` now fails if
`\d` returns to either copy, or if either stops carrying its four `[0-9]` search sites.

### Changed — `sync-agents-skills.sh` mirrors 16 more `.agents` skills (D-FLEET-38, ruled A 2026-11-16)

`tools/launchd/sync-agents-skills.sh` now also copies benchmark-runner, builtin-developer,
cloud-cascade-debug, collaboration-hub, coordinator-helper, eval-analyzer, eval-gap-finder,
gemini-cli-helper, github-issue-triage, local-model-onboarding, local-ollama-eval, parser-developer,
pr-monitor, prompt-manager, trace-debugger and use-ailang from `.claude/skills` to `.agents/skills`.
Re-measured before the copy: 14 of the `.agents` copies differed only by the 2026-07-28
claude-to-Codex rename (which pointed codex and pi at `.Codex/skills/...` paths that do not exist), two
were already identical, and none held an `.agents`-only file, so the `rsync --delete` lost nothing. The
14 regenerated `SKILL.md` files now read the `.claude` wording. `test_agents_skills_sync.sh` pins the
synced list to exactly these 23 named skills (read from the list the script actually iterates), with
no wildcard. Ten paired skills still differ and stay off the list: design-doc-creator (D-FLEET-36)
and agent-inbox, ailang-packages, benchmark-manager, cli-doc-maintainer, docs-sync, microrag,
model-manager, motoko-analyzer and post-release, which are outside the ruling.
