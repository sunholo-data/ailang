# OpenRouter Cost-Comparison Benchmark

Reference benchmark for the M-AI-OPENROUTER milestone. Demonstrates the
**cost-visibility win** that justifies adding OpenRouter as a provider:
the same prompt is run against several routed models and the per-call
cost (USD), token usage, and latency are captured side-by-side from the
trace's `ResolvedRoute` payload.

This benchmark is **live-API only**. It is not part of `make ci` and
does not run in CI — it requires `OPENROUTER_API_KEY` and makes real,
billable calls to OpenRouter. When the env var is unset, `run.sh` exits
0 with a friendly skip message.

## The prompt

A simple, deterministic-ish summarisation task that every model can
handle:

> Summarize the AILANG language in exactly two sentences.

Using a small prompt keeps the per-call cost bounded (typically well
under $0.01/model in 2026) while still surfacing meaningful
input/output token differences across models.

## Models compared

| Model string                              | Vendor    | Tier              |
|-------------------------------------------|-----------|-------------------|
| `anthropic/claude-sonnet-4.5`             | Anthropic | Frontier          |
| `openai/gpt-5-mini`                       | OpenAI    | Mid               |
| `google/gemini-2.5-flash`                 | Google    | Cheap, fast       |
| `meta-llama/llama-3.3-70b-instruct`       | Meta      | Open weights      |

These four span the price/quality space — frontier proprietary, a
mid-tier proprietary, a fast cheap proprietary, and an open-weights
hosted option. Adjust the list in `run.sh` to swap or extend.

## What the benchmark measures

For each model, one Chat Completions call is made via OpenRouter. The
trace JSONL is captured via `--emit-trace jsonl`, and `jq` extracts the
`route` payload that the AILANG OpenRouter handler populates:

- `model` — the requested model string (the `<vendor>/<model>` form)
- `input_tokens` — prompt tokens (from `route.prompt_tokens`)
- `output_tokens` — completion tokens (from `route.completion_tokens`)
- `cached_tokens` — cached prompt tokens, if the model/provider reports
  them (from `route.cached_tokens`; many providers report 0)
- `cost_usd` — USD billed for the call (from `route.cost_usd`)
- `latency_ms` — wall time of the call (measured client-side)

A CSV row per model is emitted, and a final summary row sums the total.

## Expected output shape

Stdout:

```
model,input_tokens,output_tokens,cached_tokens,cost_usd,latency_ms
anthropic/claude-sonnet-4.5,42,68,0,0.001530,2103
openai/gpt-5-mini,42,71,0,0.000142,1894
google/gemini-2.5-flash,42,65,0,0.000041,1271
meta-llama/llama-3.3-70b-instruct,42,72,0,0.000076,1987
TOTAL,168,276,0,0.001789,7255
```

(Numbers above are illustrative — actual values vary per call.)

## Cost-visibility story

The CSV makes the cost-spread across vendors immediately legible. In
the example run above, claude-sonnet-4.5 is ~37x more expensive per
call than gemini-2.5-flash for the same prompt. Without
M-AI-OPENROUTER, you would need to integrate four vendor SDKs and
reconcile four different usage payloads to get this number; with the
OpenRouter adapter and the `ResolvedRoute` trace payload it is one
script.

## Deferral status this benchmark exercises

This benchmark exercises the M-AI-OPENROUTER M1+M2+M3 surface as
shipped:

- **M1 (adapter)**: Each call goes through `internal/ai/openrouter/`,
  which forwards the OpenRouter-extended `usage` block (cached_tokens,
  cost_usd) into the AILANG response and trace.
- **M2 (routing IR)**: This benchmark uses pinned models, not routing
  policies. The optional routing flags (`--routing-*`) would be added
  to the `run.sh` invocations to compare a routing policy against
  pinned models.
- **M3 (trace integration)**: The `route` payload extracted by `jq` is
  the M3 deliverable — AI effect ops emit trace events and the
  OpenRouter handler populates `ResolvedRoute` with the resolution
  metadata.

The deferred type-level work (`!{AI[mode=routeable]}`) does not affect
this benchmark — pinned-model calls are valid under plain `!{AI}` per
the runtime safety gate's policy.

## See also

- [run.sh](run.sh) — the runner
- [README.md](README.md) — usage instructions
- [examples/ai_openrouter_routing.ail](../../examples/ai_openrouter_routing.ail) — the M3 example
- [docs/docs/guides/ai-routing.md](../../docs/docs/guides/ai-routing.md) — the user-facing guide
