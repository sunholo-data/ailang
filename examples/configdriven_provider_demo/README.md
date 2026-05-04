# Config-driven AI provider demo

Reference example for [M-AI-PROVIDER-CONFIG](../../design_docs/planned/v0_15_0/m-ai-provider-config.md). Demonstrates registering a custom AI provider via a `[[ai_provider]]` block in `ailang.toml` — no Go code, no binary fork.

## What this proves

Two providers are declared in [ailang.toml](ailang.toml):

- `demo-vllm` — local OpenAI-compatible server (vLLM, llama.cpp, etc.) with no auth
- `demo-openrouter-via-config` — OpenRouter via the generic config-driven path (illustrative; the built-in `openrouter` provider is recommended for production since the `[[ai_provider]]` schema doesn't yet cover OpenRouter's routing-policy features)

Both flow through the same AI effect machinery as built-in providers: budget tracked, `--caps AI` gated, trace spans emitted with the same shape.

## Run

**With a local vLLM/llama.cpp server on port 8000:**

```bash
cd examples/configdriven_provider_demo
ailang run --caps AI,IO --ai demo-vllm --model demo-vllm/llama-3.1-8b main.ail
```

**With OpenRouter via the config-driven path:**

```bash
export OPENROUTER_API_KEY=sk-or-...
cd examples/configdriven_provider_demo
ailang run --caps AI,IO \
  --ai demo-openrouter-via-config \
  --model demo-openrouter-via-config/anthropic/claude-sonnet-4-5 \
  main.ail
```

## What the CLI does at startup

1. Walks up from the working directory to find `ailang.toml` (the existing `pkg.FindManifest` lookup).
2. Loads it, harvests every `[[ai_provider]]` block, and registers each with the global provider registry.
3. Walks the lock file's dependencies and harvests their `[[ai_provider]]` blocks too.
4. Cross-package duplicate-name conflicts produce a structured error naming both source manifests.
5. Built-in provider names (`openai`, `anthropic`, `gemini`, `ollama`, `openrouter`) win at dispatch over config-driven providers that share their name (with a warning to stderr).

When `std/ai.call` runs in [main.ail](main.ail), the dispatch in `cmd/ailang/ai_handlers.go` consults the registry after the built-in switch falls through, finds the config-driven provider, and constructs a handler that wraps it with the standard `ai.NewHandler`.

## Adding your own provider

Copy `[[ai_provider]]` from this `ailang.toml` and adjust:

- `name` — your routing prefix (e.g. `"my-llm"` for `call("my-llm/<model>", ...)`)
- `endpoint` — your HTTP endpoint
- `request_shape` — `openai_chat` (most common), `anthropic_messages`, or `simple_completion` (Ollama-style)
- `response_path` — JSONPath to extract the response text on 2xx
- `auth` — `bearer` / `x-api-key` / `query-param` / `none`, or use `auth_headers` for custom headers with `${ENV_VAR}` interpolation
- `cost` — per-token and/or per-call USD rates for budget tracking
- `capabilities` — flag the features your provider supports

See [docs/docs/guides/custom-ai-providers.md](../../docs/docs/guides/custom-ai-providers.md) for the full schema reference and three real-world recipes (vLLM, llama.cpp, Anthropic).
