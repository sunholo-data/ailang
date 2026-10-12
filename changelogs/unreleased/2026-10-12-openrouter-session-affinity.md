### Added — OpenRouter session affinity (`x-session-id`)

- Every OpenRouter request now carries `x-session-id`, so OpenRouter keeps one conversation on one
  upstream host (warm prompt cache, consistent model behaviour). The value comes from
  `OPENROUTER_SESSION_ID`, or else one id per process. motoko's launcher sets it to the motoko
  session id.
- Why: on 2026-10-12, with GLM-5.3-Flash on the same task at the same time, pi (which sends the
  header) stayed on one host for all 81 calls with 97% of input cached. motoko via `std/ai` was
  load-balanced over three hosts, one of which cached 8% of input and returned tool calls that
  contradicted the model's own reasoning.
- motoko pin `aa23bbd1`: its launcher passes the motoko session id as `OPENROUTER_SESSION_ID`.
