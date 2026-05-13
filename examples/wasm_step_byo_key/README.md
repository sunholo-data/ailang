# wasm_step_byo_key — browser ai.step demo

Demonstrates `ai.step`, `ai.stepWithCache`, and `ai.stepWithStream` running
inside browser-AILANG (WASM) against a direct provider HTTP endpoint.
The user's API key is held in `localStorage` and sent only to the chosen
provider — there is no AILANG coordinator in the loop.

## Files

- `chat.ail` — three exports: `ask`, `askCached`, `askStreaming`
- `index.html` — JS host that registers `ailangSetAIStepHandler`,
  `ailangSetAIStepWithCacheHandler`, `ailangSetAIStepWithStreamHandler`
  and wires them to either Anthropic or OpenRouter

## Type-checking

```bash
ailang check examples/wasm_step_byo_key/chat.ail
```

## Running

The WASM blob and `wasm_exec.js` ship via the docs site. To preview locally:

```bash
make wasm-serve   # serves docs/static/{wasm,js} on http://localhost:3000
# Then open http://localhost:3000/examples/wasm_step_byo_key/index.html
```

Paste a key (`sk-ant-...` for Anthropic, `sk-or-...` for OpenRouter), pick
a model, click one of the three buttons.

## What this verifies end-to-end

- `WasmAIHandler.Step` invokes `ailangSetAIStepHandler` callback
- `WasmAIHandler.StepWithCache` forwards `CacheBreakpoint{Position, TTL}`
  → JS handler maps to Anthropic `cache_control: ephemeral` on the system
  block
- `WasmAIHandler.StepWithStream` runs the per-chunk callback via
  `js.FuncOf` — `ContentDelta` / `ThinkingDelta` / `Usage` chunks land in
  AILANG's `renderChunk` with the right `kind` discriminator
- `Response` round-trip: AILANG sees `cache_read_input_tokens` from the
  warm-cache turn

## Limitations

- Tool calls (`tools` parameter) are not exercised — pass empty `[]`. Wiring
  tool dispatch from a browser is out of scope for this BYO-key path; the
  message-bus path (M-WASM-AI-STEP-VIA-MESSAGES) is the place for that.
- Anthropic's CORS policy requires the
  `anthropic-dangerous-direct-browser-access: true` header. Production
  deployments should proxy through their own domain.
