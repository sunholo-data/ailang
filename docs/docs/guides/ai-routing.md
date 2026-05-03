# AI Provider Routing (OpenRouter)

> Available since v0.16.0 (M-AI-OPENROUTER).

[OpenRouter](https://openrouter.ai) is a meta-provider that proxies one
HTTP API to ~100 frontier models behind a unified Chat Completions
shape. AILANG ships an OpenRouter adapter so you can target
`<vendor>/<model>` strings (e.g. `anthropic/claude-sonnet-4.5`,
`google/gemini-2.5-flash`, `meta-llama/llama-3.3-70b-instruct`) without
adding per-vendor SDK code.

The interesting part is **how AILANG keeps dynamic provider routing
replayable** without violating the determinism / replayability axioms.
Two safety properties hold:

1. **Replayable resolution.** Every routed call records the resolved
   model, fallback chain, prompt/completion/cached tokens and USD cost
   in the trace's `ResolvedRoute` payload. Replaying a trace can pin
   the call to the exact model that ran originally.
2. **Explicit opt-in.** Routing-capable behaviour is gated behind an
   explicit `--allow-routing` runtime flag. Programs that don't
   declare routing intent cannot accidentally pick up a multi-vendor
   fallback chain.

## Quick start

```bash
export OPENROUTER_API_KEY=sk-or-...

# Pinned model — no routing
ailang run \
  --caps IO,AI \
  --ai openrouter/anthropic/claude-sonnet-4.5 \
  --entry main \
  examples/ai_openrouter_routing.ail

# Routing policy — ordered fallback, structured-output capable, cheapest
ailang run \
  --caps IO,AI \
  --ai openrouter/auto \
  --routing-fallback "anthropic,openai,google" \
  --routing-require structured_outputs \
  --routing-prefer cheapest \
  --allow-routing \
  --entry main \
  examples/ai_openrouter_routing.ail
```

The first invocation pins to one model — no `--allow-routing` needed.
The second declares a routing policy and therefore must include
`--allow-routing` (otherwise the CLI errors before submitting the
request).

## Routing policies

OpenRouter accepts a `provider` block on each call that constrains how
it picks an upstream. AILANG exposes that surface as four CLI flags
plus the safety gate.

| Flag | Maps to OpenRouter `provider`... | Notes |
|---|---|---|
| `--routing-fallback "vendor1,vendor2,..."` | `order: [...]` + `allow_fallbacks: true` | Ordered list of preferred upstreams. OpenRouter tries them in order and falls through on failure. |
| `--routing-require "cap1,cap2,..."` | `require_parameters: [...]` | Hard capability requirements (e.g. `structured_outputs`, `tool_calling`, `vision`). Models that lack a required capability are skipped. |
| `--routing-prefer cheapest\|fastest\|most_reliable` | `sort: ...` | Tie-breaker among models that satisfy the requirements. `cheapest` is the most common choice. |
| `--routing-max-price <usd>` | (carried in IR; not yet forwarded) | The IR carries this value but the OpenRouter `provider` field doesn't accept it on the standard path — OpenRouter's per-call max-price filter lives behind their `transforms` API, which is deferred. The flag is accepted today and surfaced in the trace, but does not bind the upstream. |
| `--allow-routing` | (no upstream mapping) | **Required** whenever any other `--routing-*` flag is set. Runtime safety gate — see below. |

The CLI rejects a call with `--routing-fallback "anthropic,..."` but no
`--allow-routing` flag with a typed error before any HTTP request goes
out. This is the runtime equivalent of the `!{AI[mode=routeable]}`
type-level marker that is planned but deferred (see [BYOK and
AI[Routeable] (future)](#byok-and-airouteable-future) below).

## Trace and replay

When a routed call completes, the AI effect op emits a trace event
with a `route` payload. This payload is the M3 deliverable: AI ops
previously did not write trace events at all; they now do, and the
OpenRouter handler populates `ResolvedRoute` via the
`AIHandlerWithRouting` interface.

A complete trace event for a routed call looks like:

```json
{
  "event": "effect",
  "effect": {
    "effect_name": "AI",
    "op": "call",
    "route": {
      "requested_model":   "openrouter/auto",
      "resolved_model":    "anthropic/claude-sonnet-4.5",
      "resolved_provider": "anthropic",
      "fallback_chain":    ["anthropic/claude-sonnet-4.5"],
      "prompt_tokens":     1234,
      "completion_tokens": 312,
      "cached_tokens":     1000,
      "reasoning_tokens":  0,
      "cost_usd":          "0.00428"
    }
  }
}
```

Useful `jq` recipes against an `--emit-trace jsonl` capture:

```bash
# Show all routed calls and their resolved model
jq -c 'select(.event=="effect" and (.effect.effect_name//"")=="AI") | .effect.route' trace.jsonl

# Sum cost across all AI calls in a trace
jq -s '[.[] | select(.event=="effect" and (.effect.effect_name//"")=="AI") | .effect.route.cost_usd | tonumber] | add' trace.jsonl

# Show the fallback chain that actually fired (more than one model = a fallback happened)
jq -c 'select(.event=="effect" and (.effect.effect_name//"")=="AI") | {requested: .effect.route.requested_model, chain: .effect.route.fallback_chain}' trace.jsonl
```

**Replay status:** the trace data is captured in v0.16.0, but the
replay engine itself is currently trace-naive — it reruns the source
file with a fresh AI handler instead of pinning to the trace's
`resolved_model`. A trace-aware replay handler that consults the
baseline trace per call (and an optional `--reroute` flag for
"what would happen now") is tracked as deferred work. The trace data
is now there for the replay engine to consume when implemented.

## Models supported

Any model OpenRouter exposes is callable via the
`openrouter/<vendor>/<model>` form (e.g. `openrouter/anthropic/claude-sonnet-4.5`).

The eval harness pre-registers ten representative models in
[`internal/eval_harness/models.yml`](https://github.com/sunholo-data/ailang/blob/dev/internal/eval_harness/models.yml)
under the `or-*` prefix:

- `or-claude-sonnet-4-5` — `anthropic/claude-sonnet-4.5` (frontier)
- `or-gpt5` / `or-gpt5-mini` — `openai/gpt-5` / `openai/gpt-5-mini`
- `or-gemini-2-5-pro` / `or-gemini-2-5-flash` — Google's tiers
- `or-llama-3-3-70b` — `meta-llama/llama-3.3-70b-instruct` (open weights)
- `or-deepseek-chat`, `or-mistral-large`, `or-qwen-2-5-72b` — long-tail
- `or-auto` — `openrouter/auto` (OpenRouter picks for you)

Adding new entries is YAML-only; no Go code changes are required to
benchmark a new model.

## BYOK and AI[Routeable] (future)

The `--allow-routing` runtime flag is a **back-compat fallback** for a
type-level marker that is planned but not yet shipped. The canonical
home for that work is
[M-EFFECT-REFINEMENT (v1.0.0)](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md),
which generalises `!{E[mode=...]}` across `Rand`, `Clock`, `Net`,
`FS`, and `AI`. See [Example 4: Modal AI](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md#example-4-modal-ai)
for the design.

When that lands, programs will be able to write:

```ailang
func summarise(text: string) -> Result[string, AIError]
  ! {AI[mode=routeable]} =
    ai.complete({ ... })
```

and the type system will reject any attempt to use a routing-capable
provider under plain `!{AI}`. Until then, the runtime gate carries the
same explicit-opt-in intent — programs that don't pass
`--allow-routing` cannot accidentally inherit a fallback chain.

`AI[scope=byok]` (per-call BYOK key passing) and
`AI[mode=replay-only]` (forbid live calls; force read-from-trace) live
in the same M-EFFECT-REFINEMENT milestone.

## Deferred (intentionally)

The M-AI-OPENROUTER sprint deferred the following items by design.
None of them block usage of OpenRouter today; they are listed here so
you know what to expect.

- **Type-level effect-row markers** — `!{AI[mode=routeable]}`,
  `!{AI[mode=replay-only]}`, `!{AI[scope=byok]}`. AILANG effects are
  flat label-strings today, not parameterised rows. Adding them needs
  parser + AST + elaborator + typechecker work — a multi-day language
  feature scoped to
  [M-EFFECT-REFINEMENT](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md).
- **Replay-engine pin-to-resolved + `--reroute` flag** — Trace data is
  captured today; the replay engine itself is still trace-naive. The
  pin-to-resolved replay handler is a separate replay-engine refactor.
- **AILANG-side `ai.complete(req)` entry point** —
  `stdlib/std/ai.ail` exposes `call`/`callJson` only; a value-level
  `provider({...route: {...}})` constructor needs new builtins.
  Today, routing is configured via CLI flags only.
- **Streaming** — Phase 1 is request/response only.
- **OpenRouter `transforms` / web-search / image features** —
  OpenRouter's preview features (middle-out compression, web-search
  tool, image inputs) are not wired through the adapter.
- **Cost budget enforcement** — `--routing-max-price` is captured in
  the policy but not forwarded to OpenRouter (their max-price filter
  lives under `transforms`). AILANG-side budget enforcement (halt on
  exceed) is also out of scope; the trace records cost so a follow-up
  can build budgets on top.

## Worked example

The shipped example file uses CLI-flag routing — the AILANG-side
`provider({...})` constructor is deferred (see above). See
[`examples/ai_openrouter_routing.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/ai_openrouter_routing.ail):

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

Run it under stub mode (no API key, deterministic for tests):

```bash
ailang run --caps IO,AI --ai-stub --entry main \
  examples/ai_openrouter_routing.ail
```

Run it against real OpenRouter with a routing policy:

```bash
export OPENROUTER_API_KEY=sk-or-...
ailang run \
  --caps IO,AI \
  --ai openrouter/auto \
  --routing-fallback "anthropic,openai,google" \
  --routing-require structured_outputs \
  --routing-prefer cheapest \
  --allow-routing \
  --emit-trace jsonl \
  --entry main \
  examples/ai_openrouter_routing.ail \
  > trace.jsonl

jq 'select(.event=="effect" and (.effect.effect_name//"")=="AI") | .effect.route' trace.jsonl
```

The expected `route` payload shape is the JSON shown in [Trace and
replay](#trace-and-replay) above. The `resolved_model` reflects which
upstream OpenRouter actually used; the `fallback_chain` lists every
model tried in order.

## Cost-comparison benchmark

The sprint ships a small benchmark that runs the same prompt across
several models and emits a CSV with per-model cost, tokens, and
latency:

```bash
export OPENROUTER_API_KEY=sk-or-...
./benchmarks/openrouter_cost_compare/run.sh > results.csv
column -t -s, results.csv
```

This benchmark is **not** part of `make ci` — it makes real billable
calls. When `OPENROUTER_API_KEY` is unset, `run.sh` exits 0 with a
skip message. See
[`benchmarks/openrouter_cost_compare/README.md`](https://github.com/sunholo-data/ailang/tree/dev/benchmarks/openrouter_cost_compare)
for details.

## Related

- [AI Effect: calling LLMs from AILANG](ai-effect.mdx) — the underlying
  `std/ai` effect this guide builds on.
- [M-AI-OPENROUTER design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_16_x/m-ai-openrouter-provider.md)
  — the canonical design, including the implementation report.
- [M-EFFECT-REFINEMENT design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md)
  — the v1.0.0 milestone where the type-level mode markers will land.
