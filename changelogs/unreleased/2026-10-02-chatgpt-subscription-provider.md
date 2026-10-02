### Added — `chatgpt/` AI provider: OpenAI models on a ChatGPT subscription

`std/ai` (and so `ailang run --ai`) can drive OpenAI models on the ChatGPT subscription the codex
CLI is logged into: `--ai chatgpt/gpt-6.1-sol`. New package `internal/ai/chatgpt` reads
`$CODEX_HOME/auth.json` (default `~/.codex/auth.json`) read-only on each request (codex owns the
refresh), refuses an API-key login or an expired token with the fix, and speaks the streaming
Responses API of the ChatGPT codex backend: text, reasoning and usage deltas, and tool calls
(`function_call` / `function_call_output` paired by `call_id`). It identifies itself as
`originator: ailang`. The factory labels it the OAuth lane.

motoko uses it through the new `motoko-chatgpt-gpt-6-1-sol` row (`MODEL=chatgpt/gpt-6.1-sol`,
`cloud` profile). The motoko preflight refuses the run without a codex ChatGPT login, and the row's
cost is labelled list-price-equivalent (subscription), not metered. First live run 2026-10-02:
fizzbuzz, json_parse and recursion_fibonacci 3/3 in 55s, $0.00 billed.

Anthropic's subscription OAuth was tried first and is NOT possible: the API answers a non-Claude-Code
client with 429 `rate_limit_error` while `claude -p` on the same account works.
