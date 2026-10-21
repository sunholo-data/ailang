### Added — `gemini-agents-argon`: Gemini 4 Argon as an agent-mode model

New registry row on the existing `gemini_agents` executor (Gemini Developer API), with `agent_model_name: antigravity-preview-09-2026/gemini-4-argon`, so the pin travels as `agent_config {type: antigravity, model: gemini-4-argon}`. `TestLiveArgonPin` (2026-10-21) showed the Developer API now rejects an unknown model with 404, so the pin is validated there. First run: 2/2 agent benchmarks pass, $0.29. Not in any suite yet.

### Fixed — `managed_agents` (Vertex) refuses a model pin instead of ignoring it

Vertex runs any `agent_config.model`, including a bogus one, so a pinned row on the Vertex executor would have run the agent's default model under the pinned row's name. It now fails before sending a request and names `gemini_agents` as the executor that honours pins.
