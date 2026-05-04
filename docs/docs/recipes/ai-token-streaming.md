---
title: AI Token Streaming
sidebar_position: 10
---

# AI Token Streaming

**Status**: Available since AILANG **v0.15.0**.

Stream LLM token-by-token responses through the AI effect with full budget tracking, capability gating, and trace integration. Composes [`std/stream`](../guides/streaming.md)'s SSE-via-POST primitive with the [M-AI-PROVIDER-CONFIG registry](../guides/custom-ai-providers.md) so URL, auth, and request body shape come from `[[ai_provider]]` config in your `ailang.toml` — no caller-supplied URLs or API keys.

## Quick start

`std/ai/streaming` re-exports the connection-driving functions from `std/stream` (`onEvent`, `runEventLoop`, `disconnect`) so a typical streaming program needs only one import for the streaming code path. Pattern-match constructors (`SSEData`, `Closed`, etc.) still come from `std/stream`.

```ailang
import std/ai/streaming (openaiCompatStream, onEvent, runEventLoop, disconnect)
import std/stream (SSEData, Closed, StreamError)   -- pattern-match constructors
import std/io (println)
import std/result (Result, Ok, Err)

export func handler(event: StreamEvent) -> bool {
  match event {
    SSEData(_, data) => if data == "[DONE]" then false else { _io_print(data); true },
    Closed(_, _) => false,
    StreamError(_) => false,
    _ => true
  }
}

export func main() -> () ! {AI, Stream, Net, IO} {
  let body = "[{\"role\":\"user\",\"content\":\"Say hi in five words.\"}]" in
  match openaiCompatStream("my-openai", "gpt-4o-mini", body) {
    Ok(conn) => {
      onEvent(conn, handler);
      runEventLoop(conn);
      disconnect(conn)
    },
    Err(_) => println("stream failed")
  }
}
```

```bash
ailang run --caps AI,Stream,Net,IO --ai my-openai --model my-openai/gpt-4o-mini app.ail
```

## When to use which

| Use case | Path |
|----------|------|
| AI provider streaming (OpenAI / OpenRouter / Anthropic / vLLM / llama.cpp / Together / Groq / Anyscale / Fireworks) | **`std/ai/streaming`** (this guide) |
| Generic SSE consumption (non-AI HTTP server, GET-style SSE) | [`std/stream.sseConnect`](../guides/streaming.md) |
| Built-in AI calls without streaming | [`std/ai`](../guides/ai-effect.mdx) — `call`, `callJson`, `callJsonSimple` |

The dispatch differs: `std/ai/streaming` requires the `AI` capability and routes through the `[[ai_provider]]` registry; `std/stream.sseConnect` requires only the `Stream` capability and takes a caller-supplied URL.

## API surface

`std/ai/streaming` exports two stream-opening functions plus types reserved for the v1.1 typed-extraction API:

```ailang
export func openaiCompatStream(
  provider: string,
  model: string,
  messagesJson: string
) -> Result[StreamConn, StreamErrorKind] ! {AI, Stream, Net}

export func anthropicStream(
  provider: string,
  model: string,
  messagesJson: string
) -> Result[StreamConn, StreamErrorKind] ! {AI, Stream, Net}

-- v1.1 reserved
export type TokenDelta = { text: string, reasoning: string, done: bool }
export type AIError = { code: string, message: string, retryable: bool }
```

Both functions take a registered provider name + model + a pre-serialised messages JSON array. The Go layer injects `"stream": true` into the request body for `openai_chat`/`simple_completion` request shapes; Anthropic streaming is selected via the `anthropic_messages` request shape and SSE event types in the response (no body flag needed).

## Recipe 1: OpenAI

```toml
# ailang.toml
[[ai_provider]]
schema_version = 1
name = "openai-stream"
endpoint = "https://api.openai.com/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "bearer", env = "OPENAI_API_KEY" }
cost = { input_per_1m_usd = 0.15, output_per_1m_usd = 0.6 }
capabilities = { tool_calling = false, json_mode = true, streaming = true, vision = false, structured_outputs = false }

[ai_provider.streaming]
enabled = true
delta_path = "$.choices[0].delta.content"
reasoning_path = "$.choices[0].delta.reasoning_content"
done_sentinel = "[DONE]"
```

Usage in AILANG:

```ailang
let body = "[{\"role\":\"user\",\"content\":\"Hello!\"}]" in
openaiCompatStream("openai-stream", "gpt-4o-mini", body)
```

Each `SSEData(eventType, data)` event's `data` is a JSON string like `{"choices":[{"delta":{"content":"Hello"}}]}`. The recipe-level extraction pattern is:

```ailang
match event {
  SSEData(_, raw) =>
    if raw == "[DONE]" then false
    else {
      -- parse raw via std/json.decode and extract delta.content
      _io_print(raw); true
    },
  ...
}
```

## Recipe 2: OpenRouter

OpenRouter speaks an OpenAI-compatible wire format, so the same `request_shape = "openai_chat"` works:

```toml
[[ai_provider]]
schema_version = 1
name = "openrouter-stream"
endpoint = "https://openrouter.ai/api/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "bearer", env = "OPENROUTER_API_KEY" }
capabilities = { tool_calling = false, json_mode = true, streaming = true, vision = false, structured_outputs = false }

[ai_provider.streaming]
enabled = true
delta_path = "$.choices[0].delta.content"
reasoning_path = "$.choices[0].delta.reasoning_content"
done_sentinel = "[DONE]"
```

```ailang
openaiCompatStream("openrouter-stream", "anthropic/claude-sonnet-4-5", body)
```

OpenRouter routes to the underlying provider per the model identifier; the AILANG-side surface is identical.

### Reasoning models (DeepSeek-R1, o1)

Reasoning models emit two delta streams: the visible `content` and the hidden `reasoning_content`. Configure both paths in your provider config:

```toml
[ai_provider.streaming]
enabled = true
delta_path = "$.choices[0].delta.content"
reasoning_path = "$.choices[0].delta.reasoning_content"
```

Then in AILANG, decode each `SSEData` `data` field with `std/json.decode` and read both fields — render the visible text as it streams, surface reasoning in a "thinking" pane:

```ailang
-- pseudocode (v1.1 will expose this via parseDelta)
match decode(raw) {
  Ok(json) => {
    let content = readPath(json, "$.choices[0].delta.content") in
    let reasoning = readPath(json, "$.choices[0].delta.reasoning_content") in
    -- render content + reasoning separately
    true
  },
  Err(_) => true
}
```

## Recipe 3: Anthropic native

Anthropic's Messages API uses a different SSE event taxonomy (`message_start`, `content_block_start`, `content_block_delta`, `message_stop`):

```toml
[[ai_provider]]
schema_version = 1
name = "anthropic-stream"
endpoint = "https://api.anthropic.com/v1/messages"
request_shape = "anthropic_messages"
response_path = "$.content[0].text"
auth = { type = "x-api-key", env = "ANTHROPIC_API_KEY" }
auth_headers = { anthropic-version = "2023-06-01" }
capabilities = { tool_calling = true, json_mode = false, streaming = true, vision = true, structured_outputs = false }

[ai_provider.streaming]
enabled = true
delta_path = "$.delta.text"
```

```ailang
anthropicStream("anthropic-stream", "claude-sonnet-4-5", body)
```

Each `SSEData(eventType, data)` carries the event type as `eventType` (e.g. `"content_block_delta"`) and the JSON payload as `data`. Filter by `eventType` to extract only deltas:

```ailang
match event {
  SSEData("content_block_delta", raw) => { _io_print(raw); true },
  SSEData("message_stop", _) => false,
  _ => true
}
```

## Effect signature

All streaming functions require `! {AI, Stream, Net}`:
- **`AI`**: budget tracking + cap gating (the AI cap is the gate for LLM access)
- **`Stream`**: SSE event-loop machinery
- **`Net`**: underlying HTTP POST

This is intentionally stricter than `std/stream.sseConnect` (which needs only `Stream`) — without the `AI` cap, AILANG cannot enforce per-provider cost ceilings. The architectural decision is documented in [m-ai-provider-config.md D11](../../design_docs/planned/v0_15_0/m-ai-provider-config.md) and the [streaming-helper design doc](../../design_docs/planned/v0_17_0/m-ai-streaming-helper.md).

## v1 limitations

- **Pre-serialised messages**: the `messagesJson` parameter is a string, not a typed `[Message]` list. Caller is responsible for JSON serialisation. v1.1 will accept a typed list.
- **No typed `parseDelta`**: extracting `TokenDelta` from raw SSE events is left to caller code in v1. The `TokenDelta` type is exported as a target shape; v1.1 ships a typed extractor that consumes the provider's `streaming.delta_path`/`streaming.reasoning_path` JSONPaths automatically.
- **`query-param` auth not supported for streaming**: bearer / `x-api-key` / `auth_headers` cover the universe of streaming-capable providers; query-param auth (Gemini-style) does not appear in any streaming endpoint we've found. If you need it, file feedback.
- **Built-in providers not routable**: streaming via this helper supports only `[[ai_provider]]`-declared providers in v1. Built-in providers (`openai`/`anthropic`/`gemini`/`ollama`/`openrouter`) have their own streaming code paths in future milestones; for now, declare a config-driven mirror if you want streaming through the unified helper.
- **`request_shape = "custom"`**: schema-reserved, not yet runtime-supported. Use one of the three v1 named shapes.

## See also

- [`std/ai/streaming`](https://github.com/sunholo-data/ailang/blob/dev/std/ai/streaming.ail) — module source
- [`std/stream`](../guides/streaming.md) — generic SSE/WebSocket primitives
- [Custom AI Providers guide](../guides/custom-ai-providers.md) — full `[[ai_provider]]` schema reference
- [`examples/runnable/ai_stream_openai.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/runnable/ai_stream_openai.ail) — runnable demo
- [Design doc: M-AI-STREAMING-HELPER](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_17_0/m-ai-streaming-helper.md)
- [Design doc: motoko integration sequence](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/motoko-integration-sequence.md) — external-consumer evidence
