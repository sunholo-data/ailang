# Codex fail-loud routing and ChatGPT deadline

Refs #903; related #1259.

- AI-effect `codex*` models now refuse construction with guidance pointing to the interim `chatgpt/` subscription lane. They no longer infer the metered OpenAI provider. Explicit `provider: openai` registry entries and Codex CLI executor routing are unchanged.
- Motoko preflight requires subscription credentials for `codex*`, and reports subscription cost provenance.
- The interim ChatGPT client limits the entire HTTP exchange, including streamed bodies, to 10 minutes; `WithTimeout` overrides it and earlier context deadlines apply. #1259 will absorb the backstop; no new flag or environment variable is added.
- Codex executor documentation now describes subscription-first authentication and the explicitly metered alternative.

Phase 2 replaces the direct backend with the sanctioned Codex CLI provider in a separate release, after the D8 migration ruling.
