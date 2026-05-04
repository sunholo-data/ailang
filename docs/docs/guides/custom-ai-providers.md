---
title: Custom AI Providers
sidebar_position: 50
---

# Custom AI Providers (Config-Driven)

**Status**: Available since AILANG **v0.15.0**.

Add a new AI provider as a **package** — no Go code, no binary fork. Declare an `[[ai_provider]]` block in your `ailang.toml`, and AILANG registers it as a first-class provider with full budget tracking, AI capability gating, and trace integration. Same machinery as the hardcoded built-ins (`openai`, `anthropic`, `gemini`, `ollama`, `openrouter`).

## Quick start

```toml
# ailang.toml in your project root
[[ai_provider]]
schema_version = 1
name = "my-llm"
endpoint = "http://localhost:8000/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "none" }
```

Then call it from AILANG:

```ailang
import std/ai (call)

export func main() -> string ! {AI} = call("Hello!")
```

```bash
ailang run --caps AI --ai my-llm --model my-llm/some-model main.ail
```

That's it. Budget, AI cap, and trace spans all flow through the standard AILANG effect machinery.

## When to use this vs a built-in provider

| Use case | Path |
|----------|------|
| OpenAI / Anthropic / Gemini / Ollama / OpenRouter | Built-in providers — already wired |
| vLLM, llama.cpp, Together, Groq, Anyscale, Fireworks, DeepInfra, Perplexity, Mistral native, any OpenAI-compatible HTTP endpoint | **Config-driven** (this guide) |
| Bedrock, Vertex AI, Azure OpenAI (custom auth flows like SigV4 / Azure AD / OAuth) | Stay built-in or escape via `auth_headers` |
| Non-HTTP transports (gRPC, WebSocket-only) | Stay built-in |

## Schema reference

```toml
[[ai_provider]]
# REQUIRED FIELDS
schema_version = 1                        # only "1" accepted in v0.15.0
name           = "my-llm"                 # routing prefix; call("my-llm/<model>", ...) routes here
endpoint       = "https://api.example.com/v1/chat/completions"
request_shape  = "openai_chat"            # "openai_chat" | "anthropic_messages" | "simple_completion" | "custom"
response_path  = "$.choices[0].message.content"  # JSONPath to extract text on 2xx
auth           = { type = "none" }        # required — see auth shapes below

# OPTIONAL FIELDS
error_path     = "$.error.message"        # JSONPath to extract error text on 4xx/5xx
cost           = { input_per_1m_usd = 1.0, output_per_1m_usd = 2.0, currency = "USD" }
# OR
cost           = { per_call_usd = 0.01 }

capabilities   = { tool_calling = false, json_mode = true, streaming = true, vision = false, structured_outputs = false }

# CUSTOM AUTH ESCAPE (use either `auth.type` OR `auth_headers`, or both)
auth_headers   = { Authorization = "Bearer ${MY_TOKEN}", X-Org = "${MY_ORG}" }

# CUSTOM REQUEST SHAPE ESCAPE (only when request_shape = "custom")
request_template = "{ \"model\": \"{{model}}\", \"input\": \"{{prompt}}\" }"

# STREAMING (consumed by M-AI-STREAMING-HELPER, v0.17.0)
[ai_provider.streaming]
enabled         = true
delta_path      = "$.choices[0].delta.content"
reasoning_path  = "$.choices[0].delta.reasoning_content"  # optional; o1, DeepSeek-R1
done_sentinel   = "[DONE]"

# MODELS ALLOW-LIST (optional; if absent, prefix-match accepts any model)
[ai_provider.models]
allowed = ["llama-3.1-70b", "llama-3.1-8b"]
```

### Request shapes (v1 catalog)

| Shape | Wire format | Use for |
|-------|-------------|---------|
| `openai_chat` | OpenAI Chat Completions (`{messages, model, max_tokens, ...}`) | OpenAI, OpenRouter, Together, Groq, Anyscale, Fireworks, vLLM, llama.cpp openai-compat, anything else OpenAI-shaped |
| `anthropic_messages` | Anthropic Messages API (`{messages: [{role, content: [{type:"text", text}]}], model, max_tokens, system}`) | Anthropic native, Anthropic-shaped proxies |
| `simple_completion` | Single-prompt format (`{prompt, model, max_tokens, ...}`) | Ollama-style endpoints |
| `custom` | Your `request_template` Go template | Anything else (deferred to schema v2 — not implemented in v0.16) |

### Auth shapes

| `auth.type` | Wire effect | Required fields |
|-------------|-------------|-----------------|
| `bearer` | `Authorization: Bearer ${env}` | `env` |
| `x-api-key` | `x-api-key: ${env}` | `env` |
| `query-param` | `?<name>=${env}` appended to URL | `env`, `name` |
| `none` | No auth header | — |

For anything else (additional headers, custom prefixes, multi-header schemes), use `auth_headers` — a literal header dict with `${VAR}` interpolation. `${VAR}` patterns are restricted to uppercase-snake-case (`[A-Z_][A-Z0-9_]*`) for safety; literal substitution only, no shell expansion.

### Capabilities

The TOML keys match the wire identifiers used in [`internal/ai/routing.go`](https://github.com/sunholo-data/ailang/blob/dev/internal/ai/routing.go) (the `AICapability` type). Routing-policy requirements (when an OpenRouter call needs `tool_calling`, etc.) and registration-time declarations share the same vocabulary.

| Key | Meaning |
|-----|---------|
| `tool_calling` | Function/tool calling supported |
| `json_mode` | JSON-mode output supported |
| `streaming` | SSE token streaming supported (consumed in v0.17.0 by M-AI-STREAMING-HELPER) |
| `vision` | Multimodal image input/output supported |
| `structured_outputs` | Schema-enforced structured output (`response_format = json_schema`) supported |

Calls requiring an unsupported capability fail fast with `AIError{ code: "CapabilityNotSupported" }`.

## Recipe 1: vLLM (local, no auth)

```toml
# ailang.toml
[[ai_provider]]
schema_version = 1
name = "vllm"
endpoint = "http://localhost:8000/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "none" }
cost = { input_per_1m_usd = 0.0, output_per_1m_usd = 0.0 }
capabilities = { tool_calling = false, json_mode = true, streaming = true, vision = false, structured_outputs = false }
```

```bash
ailang run --caps AI --ai vllm --model vllm/llama-3.1-70b my_app.ail
```

## Recipe 2: llama.cpp (local server with custom auth header)

```toml
[[ai_provider]]
schema_version = 1
name = "llamacpp"
endpoint = "http://localhost:8080/completion"
request_shape = "simple_completion"
response_path = "$.content"
auth_headers = { X-Internal-Token = "${LLAMACPP_TOKEN}" }
cost = { per_call_usd = 0.0 }
```

The `X-Internal-Token` header is constructed at call time from `$LLAMACPP_TOKEN`. If the env var is unset the call fails with a clear "missing credential" error before any HTTP request happens.

## Recipe 3: Anthropic native via the config-driven path

This works but the **built-in `anthropic` provider is recommended for production** — it has features (tool use, image input, native streaming, error normalization) that the v1 schema doesn't yet cover. Use config-driven Anthropic when you specifically need to test the schema's expressiveness, or when wrapping a proxy that's Anthropic-shaped but doesn't have a dedicated provider.

```toml
[[ai_provider]]
schema_version = 1
name = "anthropic-via-config"
endpoint = "https://api.anthropic.com/v1/messages"
request_shape = "anthropic_messages"
response_path = "$.content[0].text"
auth = { type = "x-api-key", env = "ANTHROPIC_API_KEY" }
auth_headers = { anthropic-version = "2023-06-01" }
cost = { input_per_1m_usd = 3.0, output_per_1m_usd = 15.0 }
capabilities = { tool_calling = true, json_mode = false, streaming = true, vision = true, structured_outputs = false }
```

## Multiple providers per package

A single `ailang.toml` may declare multiple `[[ai_provider]]` blocks — they're independent registrations. Useful when wrapping a multi-tenant proxy or shipping a curated set of related providers.

## Conflicts and dispatch order

When multiple installed packages declare `[[ai_provider]]` blocks:

- **Within one package**: duplicate names error at manifest load (per-manifest validation).
- **Across packages**: duplicate names error at registration time, naming both source manifests so the user can resolve.
- **Built-in vs config-driven**: built-in always wins at dispatch with a warning to stderr. Don't shadow built-in names.

## What lives in the registry

The runtime tracks the harvested providers in a global registry (see [`internal/ai/registry.go`](https://github.com/sunholo-data/ailang/blob/dev/internal/ai/registry.go)). The dispatch chain in [`cmd/ailang/ai_handlers.go`](https://github.com/sunholo-data/ailang/blob/dev/cmd/ailang/ai_handlers.go) consults this registry after the built-in switch falls through.

## Limitations (v0.15.0)

- **Custom auth flows** (AWS SigV4, Azure AD, OAuth) — not supported; use the `auth_headers` escape with `${ENV_VAR}` for static tokens, or rely on built-in providers for full custom auth.
- **Non-HTTP transports** (gRPC, persistent WebSocket) — Go-side; not in v1 schema.
- **Streaming** — schema accepts `[ai_provider.streaming]` blocks but the runtime hook lands in v0.17.0 with M-AI-STREAMING-HELPER.
- **Tool-use templating** — `capabilities.tool_calling` flag works, but the request-shape templating for tool definitions is deferred to schema v2.
- **Image input templating** — same.
- **`request_shape = "custom"` + `request_template`** — reserved in the schema; runtime support deferred to schema v2.
- **Go plugin / WASM provider extensions** — explicitly rejected (see [m-ai-provider-config.md](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_15_0/m-ai-provider-config.md) Decision D2).

If you hit a limitation, file feedback via the MCP `submit_feedback` tool or open an issue at [github.com/sunholo-data/ailang/issues](https://github.com/sunholo-data/ailang/issues).

## See also

- [examples/configdriven_provider_demo](https://github.com/sunholo-data/ailang/tree/dev/examples/configdriven_provider_demo) — runnable reference example
- [Design doc: M-AI-PROVIDER-CONFIG](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_15_0/m-ai-provider-config.md) — architectural reasoning
- [Design doc: motoko integration sequence](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/motoko-integration-sequence.md) — the external-consumer evidence base for this milestone
- [`std/ai`](./ai-effect.mdx) — the `AI` effect this dispatches through
- [`std/stream`](../runnable/stream_sse.ail) — the SSE infrastructure consumed by streaming providers (v0.17.0)
