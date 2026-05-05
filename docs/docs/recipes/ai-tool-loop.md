---
title: AI Multi-Turn Tool Loop
sidebar_position: 12
---

import CodeBlock from '@theme/CodeBlock';
import AIToolLoopExample from '!!raw-loader!@site/../examples/runnable/ai_tool_loop.ail';

# AI Multi-Turn Tool Loop

**Status**: Available since AILANG **v0.17.0** (M-AI-TOOL-LOOP).

Drive an OpenAI-style tool-use loop in pure AILANG: the model emits tool
calls, your `dispatch` callback runs them, results are threaded back, the
model continues. The whole agent loop — provider HTTP, tool-call parsing,
message threading, typed errors — fits in a few lines.

This recipe complements [AI Token Streaming](./ai-token-streaming.md): use
`callStream` when you want token-by-token output without tools, use
`runTools` when you want multi-turn dispatch (with or without streaming —
streaming `runTools` may land in a future release).

## Quick start: `runTools`

`std/ai.runTools` is the convenience driver. Pass a model, an initial
conversation, a tool catalog, a dispatch callback, and a step budget.
Get back the full final transcript on success or a typed `AIError` on
failure.

```ailang
import std/ai (runTools, Message, ToolSchema, ToolCall, AIError)
import std/io (println)
import std/result (Result, Ok, Err)
import std/list (length)

export pure func dispatch(call: ToolCall) -> string {
  if call.name == "read_doc"
  then "[NDA between Acme and Beta, 2-year term, governed by Delaware law]"
  else "unknown tool: ${call.name}"
}

export func main() -> () ! {AI, IO} {
  let tools: [ToolSchema] = [{
    name: "read_doc",
    description: "Read the named document",
    parameters: "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"}},\"required\":[\"name\"]}"
  }];
  let messages: [Message] = [{
    role: "user", content: "Summarize nda.docx",
    tool_calls: [], tool_call_id: ""
  }];
  match runTools("", messages, tools, dispatch, 8) {
    Ok(transcript) =>
      println("=== ${show(length(transcript))} turns ==="),
    Err(e) =>
      if e.retryable
      then println("transient ${e.code}: ${e.message}")
      else println("fatal ${e.code}: ${e.message}")
  }
}
```

```bash
ailang run --caps AI,IO --ai gemini-3-flash-preview \
  examples/runnable/ai_tool_loop.ail
```

## How the loop works

`runTools` calls `step` repeatedly:

1. **Step 1.** Call `step(model, messages, tools)` → `Ok(StepResult)`.
2. **If `result.finish_reason != "tool_calls"`** → loop terminates with
   `Ok(messages ++ [result.message])`.
3. **Otherwise** for each `result.tool_calls[i]`:
   - Call `dispatch(toolCall)` → result string
   - Build a tool-role `Message {tool_call_id: toolCall.id, content: result}`
4. **Append** `[result.message] ++ tool_messages` to the conversation.
5. **Recurse** with `budget - 1`.
6. **If `budget <= 0`** → `Err(AIError{Internal, "step budget exhausted",
   retryable: false})`.

The full transcript is returned, so you can extract the last assistant
message, the full conversation, the tool dispatches, or whatever shape
your application needs.

## Effect-polymorphic dispatch

The `dispatch` callback type is `(ToolCall) -> string` — but AILANG
infers an open effect row, so a callback with effects (`! {FS, Process}`,
`! {Net}`, etc.) composes without changing `runTools`'s signature.

```ailang
-- Real dispatch with file I/O + shell exec
func dispatchReal(call: ToolCall) -> string ! {FS, Process} {
  if call.name == "read_doc"
  then read_file(extract_name(call.arguments))
  else if call.name == "run_command"
  then exec_shell(call.arguments)
  else "unknown tool: ${call.name}"
}

-- runTools picks up the {FS, Process} from dispatch automatically
runTools("", messages, tools, dispatchReal, 8)  -- inferred ! {AI, FS, Process}
```

This is the pattern [`motoko_agent`](https://github.com/arniwesth/motoko_agent)
uses — its tool runtime has `! {FS, Process}` effects and composes with
`runTools` with no signature change.

## Lower-level: just `step`

If you need to interpose between turns (per-turn approval gates, cost
ceilings, custom logging), drop down to `step` and write your own loop:

```ailang
import std/ai (step, Message, ToolSchema, AIError)

func myLoop(messages: [Message], budget: int)
  -> Result[[Message], AIError] ! {AI, IO} {
  if budget <= 0
  then Err({code: "Internal", message: "out of budget", retryable: false})
  else match step("", messages, []) {
    Err(e) => Err(e),
    Ok(result) => {
      println("turn cost: ${show(result.input_tokens + result.output_tokens)} tokens");
      -- ... custom dispatch / approval / logging here ...
      myLoop(messages ++ [result.message], budget - 1)
    }
  }
}
```

## Provider parity

| Provider | Tool support | Notes |
|---|---|---|
| **Anthropic** | ✅ | Messages API `tool_use` content blocks; `arguments` is JSON object |
| **Gemini** | ✅ | `functionCall` parts; adapter generates stable `<turn>_<call>` IDs (Gemini lacks native IDs) |
| **OpenAI** | ✅ | Chat Completions `tool_calls`; `arguments` is JSON STRING (OpenAI quirk) |
| **OpenRouter** | ✅ | Passthrough; composes with `Routing` policies for cost/capability routing |
| **Ollama** | ❌ | Returns `AIError{ToolsNotSupported}`; for no-tools calls falls through to chat |
| **Configdriven** | ❌ in v1 | Same typed reject; v2 may add tool support to the `[[ai_provider]]` schema |

The wire-format differences are normalized at the adapter layer — your
AILANG code sees the same `ToolSchema` / `ToolCall` / `Message` records
regardless of provider.

## Typed errors

Every Result-returning call (`callResult`, `callJsonResult`, `step`,
`runTools`) returns the same `AIError` shape on failure:

```ailang
type AIError = {
  code: string,        -- "RateLimit" | "AuthFailed" | "ContextLength" | ...
  message: string,
  retryable: bool      -- agents can route on this for retry decisions
}
```

See the [AI Effect guide](../guides/ai-effect.mdx#typed-errors-with-callresult--calljsonresult-v0170)
for the full code vocabulary and retry-vs-fatal taxonomy.

## Worked example

The full [`examples/runnable/ai_tool_loop.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/runnable/ai_tool_loop.ail)
demonstrates a 2-tool catalog (`read_doc`, `list_docs`) with an offline-
deterministic dispatch — runs against `--ai-stub` for testing without a
live model.

<CodeBlock language="ailang">{AIToolLoopExample}</CodeBlock>

## When to use which

| You want... | Use |
|---|---|
| One-shot text response, can crash on failure | `call(input)` |
| One-shot text response, typed errors | `callResult(input)` |
| One-shot JSON response with schema | `callJsonResult(input, schema)` |
| Token-by-token streaming, no tools | [`callStream`](./ai-token-streaming.md) |
| One agent turn with tool catalog | `step(model, messages, tools)` |
| Full agent loop with tool dispatch | `runTools(model, messages, tools, dispatch, budget)` |

## Related

- [AI Effect guide](../guides/ai-effect.mdx) — full reference for the `std/ai` surface
- [AI Token Streaming](./ai-token-streaming.md) — streaming companion recipe
- [AI Provider Routing](../guides/ai-routing.md) — OpenRouter `Routing` policies
- [Custom AI Providers](../guides/custom-ai-providers.md) — `[[ai_provider]]` config
