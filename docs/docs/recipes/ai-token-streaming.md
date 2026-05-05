---
title: AI Token Streaming
sidebar_position: 10
---

# AI Token Streaming

**Status**: Available since AILANG **v0.15.0**. The accumulator wrapper [`callStream`](#quick-start-callstream) is **v0.15.1+**.

Stream LLM token-by-token responses through the AI effect with full budget tracking, capability gating, and trace integration. Composes [`std/stream`](../guides/streaming.md)'s SSE-via-POST primitive with the [M-AI-PROVIDER-CONFIG registry](../guides/custom-ai-providers.md) so URL, auth, and request body shape come from `[[ai_provider]]` config in your `ailang.toml` — no caller-supplied URLs or API keys.

## Quick start: `callStream`

If you only need the accumulated final string (the most common case for non-UI agents), use [`callStream`](https://github.com/sunholo-data/ailang/blob/dev/std/ai/streaming.ail). It opens the connection, drives the event loop in Go, accumulates content deltas, and returns `Result[string, AIError]` — no caller-side event handler, no `runEventLoop`, no manual JSON extraction.

```ailang
import std/ai/streaming (callStream, AIError)
import std/io (println)
import std/result (Result, Ok, Err)

export func main() -> () ! {AI, Stream, Net, IO} {
  let body = "[{\"role\":\"user\",\"content\":\"Say hi in five words.\"}]" in
  match callStream("my-openai", "gpt-4o-mini", body) {
    Ok(text) => println(text),
    Err(e) => println("stream failed: " ++ e.code ++ " (retryable=" ++ (if e.retryable then "true" else "false") ++ ")")
  }
}
```

```bash
ailang run --caps AI,Stream,Net,IO --ai my-openai --model my-openai/gpt-4o-mini app.ail
```

`callStream` works against any `[[ai_provider]]` whose `streaming.enabled = true` — OpenAI, OpenRouter, Anthropic, vLLM, llama.cpp, Together, Groq, Anyscale, Fireworks, etc. Provider-specific event taxonomies (`[DONE]` sentinel vs `message_stop`) are handled internally.

For per-token UI updates, drop down to `openaiCompatStream` + the manual event loop — see [Advanced: token-by-token control flow](#advanced-token-by-token-control-flow) below.

## When to use which

| Use case | Path |
|----------|------|
| **Accumulated final string from an LLM (most common)** | [`std/ai/streaming.callStream`](#quick-start-callstream) |
| Per-token UI updates / streaming chunks to a frontend | [`std/ai/streaming.openaiCompatStream`](#advanced-token-by-token-control-flow) + `onEvent` + `runEventLoop` |
| Non-streaming JSON-shaped responses with schemas | [`std/ai`](../guides/ai-effect.mdx) — `callJson`, `callJsonSimple` |
| Generic SSE consumption (non-AI HTTP server, GET-style SSE) | [`std/stream.sseConnect`](../guides/streaming.md) |

`callStream` is the zero-boilerplate option: same wire format, same providers, same effect signature, but you get back a single string instead of having to assemble the event loop yourself. `openaiCompatStream` + event loop is the right choice when you need the *deltas themselves* — for streaming output to a chat UI, for instance.

## API surface

`std/ai/streaming` exports the synchronous accumulator (`callStream`), two stream-opening primitives, and the `AIError` record:

```ailang
-- Accumulator wrapper (v0.15.1+) — opens, drives the loop, returns the joined string.
export func callStream(
  provider: string,
  model: string,
  messagesJson: string
) -> Result[string, AIError] ! {AI, Stream, Net}

-- Stream-opening primitives — return a StreamConn the caller drives.
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

export type AIError = { code: string, message: string, retryable: bool }

-- v1.1 reserved
export type TokenDelta = { text: string, reasoning: string, done: bool }
```

All three streaming functions take a registered provider name + model + a pre-serialised messages JSON array. The Go layer injects `"stream": true` into the request body for `openai_chat`/`simple_completion` request shapes; Anthropic streaming is selected via the `anthropic_messages` request shape and SSE event types in the response (no body flag needed).

Reasoning fields (`reasoning_content`, `thinking`) emitted by reasoning models are READ by `callStream` but not included in the returned string in v0.15.1 — only visible content accumulates. v0.15.2's `callStreamWithReasoning` will surface both.

## Advanced: token-by-token control flow

When you need each delta as it arrives (chat UI, progress indication, early-stop on a sentinel), use the underlying primitive `openaiCompatStream` (or `anthropicStream`) and drive the event loop yourself. `std/ai/streaming` re-exports `onEvent`, `runEventLoop`, and `disconnect` from `std/stream` so a typical streaming program needs only one import for the streaming code path. Pattern-match constructors (`SSEData`, `Closed`, etc.) still come from `std/stream`.

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

For most non-UI agents — code synthesis, tool dispatching, evaluations — `callStream` is simpler and equivalent. Reach for the manual event loop only when you genuinely need per-delta hooks.

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

Then in AILANG, decode each `SSEData` `data` field with `std/json.decode` and read both fields — render the visible text as it streams, surface reasoning in a "thinking" pane. The v1 extraction pattern (working code; v1.1 will wrap this behind `parseDelta`):

```ailang
import std/ai/streaming (openaiCompatStream, onEvent, runEventLoop, disconnect)
import std/stream (StreamEvent, SSEData)
import std/json (decode, getString, asString)
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)

-- Extract a (text, reasoning) pair from one SSE data line. Returns
-- (content_delta, reasoning_delta) — empty strings when the field is
-- absent in this particular delta event.
export func extractDelta(raw: string) -> {text: string, reasoning: string} {
  if raw == "[DONE]" then {text: "", reasoning: ""}
  else match decode(raw) {
    Ok(json) => {
      -- choices[0].delta is the OpenAI-shape envelope; reach in via
      -- nested getString lookups. For deeply-nested paths, std/json's
      -- helper API is the v1 alternative to JSONPath.
      let content = getNestedString(json, ["choices", "0", "delta", "content"]) in
      let reasoning = getNestedString(json, ["choices", "0", "delta", "reasoning_content"]) in
      {text: content, reasoning: reasoning}
    },
    Err(_) => {text: "", reasoning: ""}
  }
}

-- Walk a path of (key | index) strings into a Json value, returning
-- the leaf string or "" if any step misses or yields a non-string.
-- Real production code can use the path-walking helpers in std/json
-- when they land in v1.1; this is the inline equivalent for v1.
func getNestedString(j: Json, path: [string]) -> string {
  -- Implementation: walk the path with std/json.getString for object
  -- keys; for array indices the v1 std/json doesn't yet have a
  -- getIndex helper, so reasoning models with array-typed deltas
  -- need either pattern-matching on JArray + std/list.head or
  -- waiting for v1.1's parseDelta. See std/json source for the
  -- full set of accessors available in v1.
  ""  -- replace with std/json walker for your path shape
}

export func handler(event: StreamEvent) -> bool {
  match event {
    SSEData(_, raw) => {
      let delta = extractDelta(raw) in
      if delta.text != "" then _io_print(delta.text) else ();
      if delta.reasoning != "" then renderReasoning(delta.reasoning) else ();
      true
    },
    _ => false
  }
}

func renderReasoning(text: string) -> () ! {IO} {
  _io_print("[thinking] ");
  _io_println(text)
}
```

The honest v1 story: `std/json.getString` returns `Option[string]` for top-level object keys, but multi-level `choices[0].delta.content` paths require either pattern-matching the JArray manually or composing decoders. v1.1's `parseDelta` will encapsulate this — until then, copy this template and adjust per provider.

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

This is intentionally stricter than `std/stream.sseConnect` (which needs only `Stream`) — without the `AI` cap, AILANG cannot enforce per-provider cost ceilings. The architectural decision is documented in [m-ai-provider-config.md D11](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_15_0/m-ai-provider-config.md) and the [streaming-helper design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_17_0/m-ai-streaming-helper.md).

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
