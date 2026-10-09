### Fixed — Codex routing refusal and interim ChatGPT deadline

Refs #903; interim HTTP deadline portion of #1259 absorbed.

- AI-effect codex* guesses and explicit codex providers now fail loudly with
  guidance to use chatgpt/<model>, instead of silently choosing metered OpenAI.
  Factory refusal precedes config-driven fallback. Explicit provider: openai
  registry rows and mission Codex executor pins retain their chosen lanes.
- Motoko codex preflight requires a ChatGPT subscription login and classifies
  its cost as subscription; an OPENAI_API_KEY alone cannot admit the run.
- Interim ChatGPT HTTP requests have a 10-minute total deadline, including
  continuous SSE response output. WithTimeout overrides it; earlier caller
  cancellation remains effective. No new timeout flag or environment variable.

Phase 1 only. D8 measurement, registry migration, CLI-backed AI-effect provider,
and direct-backend removal remain separate gated work.
