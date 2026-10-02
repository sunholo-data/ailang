### Added — `--ai-no-adc` and `--ai-key-file` for the AI effect (#1499)

- `ailang run --ai-no-adc` (or `AILANG_AI_NO_ADC=1`, which also covers every other in-process caller of the
  provider factory) disables the Google provider's silent Application Default Credentials fallback. A missing
  `GOOGLE_API_KEY` / `GEMINI_API_KEY` is then a hard `AuthFailed` error at startup instead of a call billed to
  whatever project `gcloud` points at. Combining it with a pinned Vertex project is refused.
- `ailang run --ai-key-file PATH` (or `AILANG_AI_KEY_FILE`) reads the `--ai` provider's API key from a
  single-line file once at startup. The key never passes through argv or the environment, and errors name
  the path only. A key-less lane (local Ollama, config-driven providers) refuses the flag rather than
  ignoring it.

### Fixed — the Gemini API key no longer rides in the request URL

- The AI Studio client sent the key as `?key=` in the URL, which net/http quotes verbatim in every transport
  error. It now goes in the `x-goog-api-key` header. A custom base URL (tests, proxies) used to receive no
  key at all and now receives the header too.
