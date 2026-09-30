### Changed
- **GPT-6.1 Sol replaces GPT-6 Sol and GPT-6 Astra in the mission pipelines.** New `gpt6-1-sol` registry row ($2/$10 per 1M, codex subscription lane, needs codex-cli >= 0.159.2). It now heads the planner and executor roles, is the controller's codex rung, and takes astra's designer rung. Measured in one agent-mode pool (core+frontier, 31 benchmarks): 30/31 vs astra 29/31 vs sol 27/31, with zero capability failures and 35% fewer tokens than sol. Standard-mode placement is pending — the OpenAI API account is out of credit.

### Fixed
- **GPT-6 rows were cost-killed in agent mode.** `gpt6-sol`, `gpt6-astra` and the new `gpt6-1-sol` had no `budgets:` block, so the default $0.50 cap killed codex runs at $0.49–$0.72 before the model finished. They now carry the standard 3.0M-token work gate with a dollar backstop sized so the token gate binds first.
- **Cloud codex images now pin codex-cli 0.159.2.** `Dockerfile.agent-codex` (and `agent-codex-go`, which inherits it) and `Dockerfile.agent-eval` installed codex unpinned, so the 09-28 images carried 0.158.0, which cannot run gpt-6.1-sol. The build now fails if the installed version differs from the pin.
