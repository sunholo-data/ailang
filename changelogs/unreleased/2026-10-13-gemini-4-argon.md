### Added — Gemini 4 Argon model row

- `gemini-4-argon` (Vertex, `ailang-dev`/global) registered opt-in, in no suite. Introductory
  $2/$10 per 1M (cache read $0.10) expiring 2026-11-30, scheduled to $4/$20 ($0.20) from
  2026-12-01. Thinking is adaptive (default MEDIUM); `thinking_budget` is rejected by the API, so
  the row sets no reasoning budget. `max_output_tokens` 32768 per Vertex's non-streaming
  per-request limit. Standard smoke+core+frontier: 49/49 pass at $0.068/run with zero thinking tokens.
