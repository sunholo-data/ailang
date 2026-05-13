---
title: Browser ai.step (BYO API key)
description: Call the typed std/ai Step API from browser-AILANG via direct provider HTTP fetch with the user's localStorage API key
sidebar_label: WASM ai.step (BYO key)
---

# Browser `ai.step` with BYO API key

:::tip Try it live
**[→ Open the live demo](pathname:///demos/wasm-step-byo-key/index.html)** — paste your API key, pick a provider (Anthropic / OpenRouter / Gemini), and click `ask` / `askCached` / `askStreaming`. Keys are stored per-provider in your browser's `localStorage` and sent only to the provider you select. No backend.
:::

Browser-AILANG (the WASM build) can call the typed [`std/ai`](/docs/reference/stdlib) Step family — `ai.step`, `ai.stepWithCache`, `ai.stepWithStream` — against a real LLM provider directly from the browser, with no AILANG coordinator or backend in the loop. The user's API key lives in `localStorage` and is sent only to the provider they pick.

This is the proven [`ai.call` BYO-key pattern](./wasm-integration#aihandler) extended to the typed multi-turn Step API. Use it when you want a "try this AI agent in your browser, no signup" landing page that talks to OpenRouter or Anthropic directly.

For server-mediated calls (centralized cost tracking, no CORS limitations, full tool dispatch on backend), see [M-WASM-AI-STEP-VIA-MESSAGES](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_19_0/m-wasm-ai-step-via-messages.md) — the complementary message-bus path.

## What ships in v0.19.0

Three new global JS hooks register the JS-callback handlers that fetch the provider:

| Hook | Wires | Callback signature |
|---|---|---|
| `ailangSetAIStepHandler(fn)` | `ai.step` | `(model, messages, tools) => Promise<Response>` |
| `ailangSetAIStepWithCacheHandler(fn)` | `ai.stepWithCache` | `(model, messages, tools, breakpoints) => Promise<Response>` |
| `ailangSetAIStepWithStreamHandler(fn)` | `ai.stepWithStream` | `(model, messages, tools, breakpoints, onChunk) => Promise<Response>` |

All three coexist with the existing `ailangSetAIHandler(fn)` for `ai.call` — internally they share a single `WasmAIHandler` so the four setters can be installed in any order without clobbering each other.

## The `Response` shape contract

Every handler must return (or resolve to) an object matching this shape. AILANG decodes it into a typed `StepResult`:

```javascript
{
  message: {
    role: "assistant",
    content: "the assistant's text response",
    tool_calls: [],          // see ToolCall shape below
    tool_call_id: ""
  },
  tool_calls: [],            // top-level OR under message — both work
  input_tokens: 42,
  output_tokens: 18,
  cache_read_input_tokens: 0,
  cache_creation_input_tokens: 0,
  finish_reason: "stop",     // or "tool_calls", "length", etc.
  model: "claude-3-5-haiku-latest"
}
```

Token-count fields default to `0` if absent. `finish_reason` and `model` default to empty strings.

## The `ToolCall` shape

Two on-wire shapes are accepted (decoded by `jsToToolCalls` in `cmd/wasm/effects.go`):

```javascript
// Canonical AILANG (flat) — preferred
{ id: "call_1", name: "search", arguments: '{"q":"go"}' }

// OpenAI nested — also accepted
{ id: "call_1", function: { name: "search", arguments: '{"q":"go"}' } }
```

`arguments` must be a JSON string (not a parsed object) — the AILANG side decodes it per the tool's parameters schema.

## The `StreamChunk` shape

`stepWithStream`'s `onChunk` callback fires with one of three discriminated objects:

```javascript
{ kind: "ContentDelta",  text: "fragment of assistant text" }
{ kind: "ThinkingDelta", text: "fragment of model reasoning (Anthropic extended thinking, OpenAI o1/o3 reasoning_content, Gemini thought parts)" }
{ kind: "Usage", input_tokens: 42, output_tokens: 18, cache_read_input_tokens: 0, cache_creation_input_tokens: 0 }
```

Concatenating all `ContentDelta.text` payloads equals `Response.message.content`. `ThinkingDelta` is only fired when the underlying provider+model emits API-level reasoning. `Usage` fires once at end-of-stream.

## The `CacheBreakpoint` shape

`stepWithCache` (and `stepWithStream`) pass cache hints through:

```javascript
{ position: "system", ttl: "ephemeral" }
```

Empty `breakpoints` is a JS empty array (not null) — the JS shim can iterate without a null check. The handler maps these to whatever its provider supports — for Anthropic that's `cache_control: { type: "ephemeral" }` on the system block:

```javascript
const wantsCache = (breakpoints || []).some(bp => bp.position === "system");
body.system = wantsCache
  ? [{ type: "text", text: sys.content, cache_control: { type: "ephemeral" } }]
  : sys.content;
```

## Provider notes

| Provider | Direct browser fetch | Notes |
|---|---|---|
| **Anthropic** | ✅ With `anthropic-dangerous-direct-browser-access: true` header. Production deployments should proxy through their own domain. |
| **OpenRouter** | ✅ Standard `Authorization: Bearer ${key}`. Routes to all backend providers (`openrouter/auto` picks the best per request). |
| **Gemini** (AI Studio) | ✅ `?key=${key}` URL parameter. |
| **OpenAI direct** | ❌ CORS rejects browser-origin requests. Use OpenRouter (`openai/gpt-4o`, etc.) or a tiny CORS proxy. |
| **Ollama** | ✅ When the user has Ollama running on `localhost:11434`. |

## End-to-end walkthrough

The reference demo lives at [`examples/wasm_step_byo_key/`](https://github.com/sunholo-data/ailang/tree/dev/examples/wasm_step_byo_key). It has three buttons (`ask`, `askCached`, `askStreaming`) wired to a `chat.ail` module's three exports.

### `chat.ail` (excerpt)

```ailang
module examples/wasm_step_byo_key/chat

import std/ai (step, stepWithCache, stepWithStream, Message, ToolSchema, CacheBreakpoint, StreamChunk, ContentDelta, ThinkingDelta, Usage)
import std/result (Result, Ok, Err)
import std/io (println)

export func ask(model: string, prompt: string) -> string ! {AI} = {
  let messages: [Message] = [
    { role: "user", content: prompt, tool_calls: [], tool_call_id: "" }
  ];
  let tools: [ToolSchema] = [];
  match step(model, messages, tools) {
    Ok(result) => result.message.content,
    Err(e) => "ERROR(${e.code}): ${e.message}"
  }
}
```

### `index.html` (excerpt)

```html
<script src="/wasm/wasm_exec.js"></script>
<script src="/js/ailang-repl.js"></script>
<script>
  const repl = new AilangREPL();
  await repl.init('/wasm/ailang.wasm');

  ailangSetAIStepHandler(async (model, messages, tools) => {
    const apiKey = localStorage.getItem('ailang-step-byo-key');
    const resp = await fetch('https://api.anthropic.com/v1/messages', {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        'x-api-key': apiKey,
        'anthropic-version': '2023-06-01',
        'anthropic-dangerous-direct-browser-access': 'true',
      },
      body: JSON.stringify({ model, max_tokens: 1024, messages }),
    });
    const data = await resp.json();
    return {
      message: { role: 'assistant', content: data.content[0].text, tool_calls: [], tool_call_id: '' },
      tool_calls: [],
      input_tokens: data.usage?.input_tokens || 0,
      output_tokens: data.usage?.output_tokens || 0,
      cache_read_input_tokens: data.usage?.cache_read_input_tokens || 0,
      cache_creation_input_tokens: 0,
      finish_reason: data.stop_reason || 'end_turn',
      model: data.model || model,
    };
  });

  await repl.loadModule('examples/wasm_step_byo_key/chat', source);
  const result = await ailangCallAsync('examples/wasm_step_byo_key/chat', 'ask', 'claude-3-5-haiku-latest', 'Compose a haiku.');
</script>
```

The full demo at [`examples/wasm_step_byo_key/index.html`](https://github.com/sunholo-data/ailang/blob/dev/examples/wasm_step_byo_key/index.html) covers OpenRouter, streaming SSE drain, and the system-prompt cache path.

## Streaming SSE drain pattern

When `ai.stepWithStream` is called, the JS handler's 5th argument is a JS function the AILANG side wired through `js.FuncOf`. Invoke it once per parsed SSE chunk:

```javascript
ailangSetAIStepWithStreamHandler(async (model, messages, tools, breakpoints, onChunk) => {
  const resp = await fetch('https://api.anthropic.com/v1/messages', {
    method: 'POST',
    headers: { /* ... */ },
    body: JSON.stringify({ model, max_tokens: 1024, messages, stream: true }),
  });
  const reader = resp.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  let text = '';

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const lines = buf.split('\n');
    buf = lines.pop();

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      const payload = line.slice(6).trim();
      if (!payload || payload === '[DONE]') continue;
      const evt = JSON.parse(payload);

      if (evt.type === 'content_block_delta' && evt.delta?.type === 'text_delta') {
        text += evt.delta.text;
        onChunk({ kind: 'ContentDelta', text: evt.delta.text });
      } else if (evt.type === 'message_delta' && evt.usage) {
        onChunk({
          kind: 'Usage',
          input_tokens: evt.usage.input_tokens || 0,
          output_tokens: evt.usage.output_tokens || 0,
          cache_read_input_tokens: 0,
          cache_creation_input_tokens: 0,
        });
      }
    }
  }
  return { message: { content: text, tool_calls: [], tool_call_id: '' }, tool_calls: [], input_tokens: 0, output_tokens: 0, cache_read_input_tokens: 0, cache_creation_input_tokens: 0, finish_reason: 'end_turn', model };
});
```

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `Err(AIError{code: "no_handler", ...})` returned to AILANG | Called `ai.step` before `ailangSetAIStepHandler(fn)` was registered | Register the handler at page load, before any `ailangCallAsync` |
| Browser console: `CORS error` from `api.openai.com` | OpenAI rejects direct browser fetch | Use OpenRouter (`openrouter/openai/gpt-4o`) or proxy through your own domain |
| Anthropic returns `400` with "anthropic-dangerous-direct-browser-access required" | Missing the explicit opt-in header | Add `'anthropic-dangerous-direct-browser-access': 'true'` to fetch headers |
| Handler returns string instead of object | AILANG sees `step response must be a JS object, got string` error | The handler must return the `Response` shape — wrap text in `{ message: { content: text, ... }, ... }` |
| `ContentDelta` chunks fire but final `Response.message.content` is empty | The handler stopped accumulating text into the final return | Build up `text` while streaming and put it in the final returned `message.content` |
| API key visible in DevTools network panel | Expected — same risk as every existing BYO-key demo | Document this in your demo; production keys should never be BYO |

## What was tested

The pure-Go conversion helpers (`messagesToJSCompat`, `toolsToJSCompat`, `cacheBreakpointsToJSCompat`) have unit tests at `cmd/wasm/effects_helpers_test.go` covering empty inputs, single/multi-message round-trips, tool-call serialization, and cache-breakpoint shape. The `js.Value`-consuming helpers (`jsToResponse`, `jsToToolCalls`, `jsToStreamChunk`) live behind the WASM build tag and are verified end-to-end via the [`examples/wasm_step_byo_key/`](https://github.com/sunholo-data/ailang/tree/dev/examples/wasm_step_byo_key) demo against real provider endpoints.

## See also

- [WASM Integration](./wasm-integration) — the broader browser-AILANG embed guide, including the legacy `ailangSetAIHandler` for `ai.call`
- [`std/ai`](/docs/reference/stdlib) — Step / StepWithCache / StepWithStream signatures and StreamChunk variants
- [Design doc: M-WASM-AI-STEP-BYO-KEY](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_19_0/m-wasm-ai-step-byo-key.md)
- [Sister design: M-WASM-AI-STEP-VIA-MESSAGES](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_19_0/m-wasm-ai-step-via-messages.md) — message-bus mediated path with centralized cost tracking
