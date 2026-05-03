# OpenRouter Cost-Comparison Benchmark

Runs the same prompt against several OpenRouter-routed models and emits
a CSV with per-model cost, tokens, and latency. Demonstrates the
cost-visibility win that justifies the M-AI-OPENROUTER milestone.

## Files

| File | Purpose |
|------|---------|
| [`scenario.md`](scenario.md) | Full scenario description, prompt, model list, expected output shape |
| [`run.sh`](run.sh) | Runner — issues one call per model, extracts cost/tokens from the trace, emits CSV |
| [`README.md`](README.md) | This file |

## Quick start

```bash
export OPENROUTER_API_KEY=sk-or-...
./benchmarks/openrouter_cost_compare/run.sh > results.csv
column -t -s, results.csv
```

## CI behaviour

This benchmark is **not** part of `make ci`. It makes real, billable
HTTP calls to OpenRouter. When `OPENROUTER_API_KEY` is unset, `run.sh`
exits 0 with a skip message — safe to invoke from any context.

## Dependencies

- `ailang` on PATH (run `make install` or `make quick-install`)
- `jq` (for parsing the trace JSONL)
- `python3` (used for portable millisecond clock and float arithmetic)

If any of the above are missing, the script prints a `skip:` message
and exits 0.

## What it exercises

The benchmark exercises the M-AI-OPENROUTER M1 + M2 + M3 surface as
shipped in v0.16.0:

- **M1** — Each call goes through `internal/ai/openrouter/`, which
  populates `route.cost_usd` and `route.cached_tokens`.
- **M2** — Pinned-model calls; routing flags (`--routing-fallback`,
  `--routing-prefer`, `--allow-routing`) can be added to the runner to
  also benchmark a routing policy.
- **M3** — The `route` payload extracted by `jq` is the M3 deliverable
  — AI effect ops emit trace events and the OpenRouter handler
  populates `ResolvedRoute`.

## Adding more models

Edit the `MODELS=(...)` array near the top of `run.sh`. Use the
OpenRouter `<vendor>/<model>` form. To view available models, see
`internal/eval_harness/models.yml` (the `or-*` entries).

## Caveats

- Cost values come from OpenRouter's reported `cost` field. They
  represent what OpenRouter billed for the call, not your actual
  invoice (rounding, credits, and BYOK can shift the latter).
- `cached_tokens` is reported when the upstream provider supports
  prompt caching and OpenRouter forwards it. Many models report 0.
- A two-sentence summarisation task is small (~40 prompt tokens). Use
  it as a relative comparison; absolute pennies will dominate token
  costs at this scale.
