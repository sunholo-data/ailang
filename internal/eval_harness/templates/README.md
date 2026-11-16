# Agent Task Templates

`agent_task_<lang>.txt` is the task message sent to the agent in agent mode, one file
per language (the path comes from `langreg.Language.TaskTemplatePath()`). It says what to
solve; the language itself is taught by the system prompt (`prompts/<lang>.md`, loaded
through langreg).

`GenerateAgentPromptsWithSystemPrompt` (`../agent_prompt.go`) loads the template and fills
the placeholders. The path is relative to the repo root, which is where evals run from;
if the file cannot be read it falls back to a hardcoded template.

## Placeholders

| Placeholder | Filled with |
|---|---|
| `{{DESCRIPTION}}` | spec `description` + `task_prompt` |
| `{{EXPECTED_OUTPUT}}` | spec `expected_stdout` (the visible test only) |
| `{{SOLUTION_PATH}}` | path of the solution file in the agent's workspace |
| `{{CAPS}}` | comma-separated capabilities |
| `{{TIMEOUT}}` | agent timeout in seconds |
| `{{PYTHON_VERSION}}` | the pinned Python version the grader runs |
| `{{CONTRACT_SPEC}}`, `{{TOOL_GUIDANCE}}`, `{{VERIFICATION_STEPS}}`, `{{EXTRA_TOOLS}}` | set by the eval condition (`baseline`, `contract`, `tool_aware`, ...) |

`<LANG>` in a benchmark's task prompt is replaced with the language's display name.

Hidden tests (`hidden_tests_dir`) must never reach a template or the workspace:
`TestHiddenTests_NeverReachPromptOrWorkspace` drives this generator and
`prepareAgentWorkspace` under every condition and fails on any leak.
