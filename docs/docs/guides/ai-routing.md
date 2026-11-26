---
title: AI Provider Routing (OpenRouter)
description: Replayable multi-vendor LLM routing through OpenRouter with explicit opt-in flags and trace resolution payloads.
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# AI Provider Routing (OpenRouter)

[OpenRouter](https://openrouter.ai) proxies ~100 models behind one Chat Completions API. AILANG's OpenRouter adapter supports `<vendor>/<model>` targets (such as `anthropic/claude-sonnet-4.5` or `openrouter/auto`) while preserving two replayability guarantees:

1. **Replayable resolution**: Every routed call records `requested_model`, `resolved_model`, `fallback_chain`, token counts, and `cost_usd` in the trace's `ResolvedRoute` payload.
2. **Explicit opt-in**: Multi-vendor routing requires `--allow-routing` (or `! {AI[mode=routeable]}`), preventing accidental fallback chains.

## Quick Start

```bash
export OPENROUTER_API_KEY=sk-or-...

# Pinned model — no --allow-routing required
ailang run --caps IO,AI --ai openrouter/anthropic/claude-sonnet-4.5 \
  --entry main examples/ai_openrouter_routing.ail

# Routing policy — ordered fallback + structured output + cheapest
ailang run --caps IO,AI --ai openrouter/auto \
  --routing-fallback "anthropic,openai,google" \
  --routing-require structured_outputs \
  --routing-prefer cheapest \
  --allow-routing \
  --entry main examples/ai_openrouter_routing.ail
```

## Routing Policy Flags

| Flag | OpenRouter `provider` Mapping | Notes |
|---|---|---|
| `--routing-fallback "v1,v2,..."` | `order: [...]` + `allow_fallbacks: true` | Ordered upstream fallback chain |
| `--routing-require "cap1,cap2"` | `require_parameters: [...]` | Skips models lacking capabilities (`structured_outputs`, `tool_calling`, `vision`) |
| `--routing-prefer cheapest\|fastest\|most_reliable` | `sort: ...` | Tie-breaker across eligible models |
| `--routing-max-price <usd>` | Recorded in IR/trace only | Accepted and logged, not forwarded to OpenRouter (`transforms` deferred) |
| `--allow-routing` | Runtime safety gate | Required whenever any `--routing-*` flag is set unless entry uses `! {AI[mode=routeable]}` |

## Effect Modes & `Parameterised Effects`

| Effect Annotation | Meaning |
|---|---|
| `! {AI}` | Bare; desugars to `! {AI[mode=fixed]}` (direct provider call) |
| `! {AI[mode=fixed]}` | Rejects routed providers via invariant unification |
| `! {AI[mode=routeable]}` | Opts into routing without needing `--allow-routing` |
| `! {AI[mode=replay-only]}` / `! {AI[scope=byok]}` | Reserved (tracked in [M-EFFECT-REFINEMENT](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_1_0/m-effect-refinement.md)) |

See [`examples/ai_modes.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/ai_modes.ail) and [Parameterised Effects](./parameterised-effects.md).

## Trace & Replay

```json
{
  "event": "effect",
  "effect": {
    "effect_name": "AI",
    "op": "call",
    "route": {
      "requested_model": "openrouter/auto",
      "resolved_model": "anthropic/claude-sonnet-4.5",
      "resolved_provider": "anthropic",
      "fallback_chain": ["anthropic/claude-sonnet-4.5"],
      "prompt_tokens": 1234,
      "completion_tokens": 312,
      "cached_tokens": 1000,
      "reasoning_tokens": 0,
      "cost_usd": "0.00428"
    }
  }
}
```

```bash
# Inspect resolved models and total cost from an --emit-trace jsonl capture
jq -c 'select(.event=="effect" and (.effect.effect_name//"")=="AI") | .effect.route' trace.jsonl
jq -s '[.[] | select(.event=="effect" and (.effect.effect_name//"")=="AI") | .effect.route.cost_usd | tonumber] | add' trace.jsonl
```

*Note*: Trace data captures `resolved_model` today; trace-pinning replay (`--reroute`) and AILANG-side `ai.complete(...)` constructors are deferred to [M-EFFECT-REFINEMENT](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_1_0/m-effect-refinement.md).

## Supported Models (`models.yml`)

Use native providers for frontier Google/Anthropic/OpenAI models (`claude-sonnet-4-6`, `gpt-5`, `gemini-2-5-pro`). Reserve `or-*` routes in [`internal/eval_harness/models.yml`](https://github.com/sunholo-data/ailang/blob/dev/internal/eval_harness/models.yml) for models without a direct API (`or-glm-5`, `or-minimax-m2-7`, `or-deepseek-v3`, `or-qwen-2-5-72b`, `or-gemma-4-26b`, `or-mistral-large`, `or-auto`).

```ailang sketch
func summarise(text: string) -> Result[string, AIError]
  ! {AI[mode=routeable]} =
    ai.complete({ ... })
```

## Worked Example

```ailang
module examples/ai_openrouter_routing

import std/ai (call)
import std/io (println)

export func main() -> () ! {AI, IO} {
  println("Calling AI through configured provider (routing-aware)...");
  let response = call("Summarize the AILANG language in one sentence.");
  println("Response:");
  println(response)
}
```

```bash
ailang run --caps IO,AI --ai-stub --entry main examples/ai_openrouter_routing.ail
```

## Related

- [AI Effect Guide](./ai-effect.mdx) — `std/ai` (`callJsonResult`, `step`, `runTools`)
- [Custom AI Providers](./custom-ai-providers.md) — `[[ai_provider]]` TOML configuration
- [M-AI-OPENROUTER design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_16_x/m-ai-openrouter-provider.md)
