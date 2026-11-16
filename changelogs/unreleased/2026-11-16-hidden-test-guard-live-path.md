### Fixed — eval: the hidden-test leak guard now drives the live agent path (2026-11-16)

`TestHiddenTests_NeverReachPromptOrWorkspace` is meant to prove that a benchmark's hidden
tests (M-EVAL-HIDDEN-TESTS) never reach the model's prompt or its workspace. It drove
`GenerateAgentPrompt` and `PrepareWorkspaceWithSyntax`, which no eval has called since
2025-10-28 (`153dcc5b2` switched the agent runner to `GenerateAgentPromptsWithSystemPrompt`
and an inline placeholder). The live agent path had no leak guard.

- The workspace setup in `RunAgentBenchmarkWithExecutor` (the `.git` folder, seeded input
  files, the fmt-hook settings and the solution placeholder) is now one function,
  `prepareAgentWorkspace`. Behaviour is unchanged.
- The guard now drives `prepareAgentWorkspace` (fmt hook off and on) and
  `GenerateAgentPromptsWithSystemPrompt` plus the traps card, for Python and AILANG, under
  every eval condition, from the repo root so the real templates load. It also still checks
  the standard-mode prompt. A temporary leak into the task prompt, the system prompt, or a
  workspace file each makes it fail.

### Changed — eval: the legacy agent prompt path is deleted (2026-11-16)

Deleted from `internal/eval_harness/agent_prompt.go`: `GenerateAgentPrompt`,
`EnhancedGenerateAgentPrompt`, `LoadActiveSyntaxReference`, `PrepareWorkspaceWithSyntax`,
and their hardcoded fallback templates. Also deleted: `langreg.Language.PromptTemplatePath`
(it pointed at `agent_prompt*.txt` files that no longer exist), the tests that exercised only
this path, and `templates/INTEGRATION.md` and `templates/TEMPLATE_GUIDE.md`, which described
it. `templates/README.md` now describes the live task templates. The teaching-prompt content
test now loads through `LoadSystemPromptForLanguage`, which is the loader the live path uses.
