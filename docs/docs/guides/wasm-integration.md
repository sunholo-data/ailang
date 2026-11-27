---
id: wasm-integration
title: WebAssembly Integration Guide
sidebar_label: WASM Integration
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# WebAssembly Integration Guide

AILANG compiles to WebAssembly (`ailang.wasm`, ~33 MB uncompressed / ~7 MB gzipped with embedded stdlib) so browsers can type-check, evaluate, and run effectful AILANG modules offline with zero backend.

:::tip See WASM in Action
Try the live browser demos: [DocParse](https://www.sunholo.com/ailang-demos/docparse.html), [Document Extractor](https://www.sunholo.com/ailang-demos/extractor.html), and [All 9 Demos](https://www.sunholo.com/ailang-demos/).
:::

## Quick Start

Download the release bundle (`ailang.wasm`, `wasm_exec.js`, `ailang-repl.js`) or build from source with `make build-wasm`:

```bash
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-wasm.tar.gz | tar -xz
# Or build from source:
make build-wasm && cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
```

<details>
<summary>HTML, Docusaurus, and React setup snippets</summary>

**Vanilla HTML:**
```html
<script src="wasm_exec.js"></script>
<script src="ailang-repl.js"></script>
<script>
  const repl = new AilangREPL();
  repl.init('/path/to/ailang.wasm').then(() => {
    console.log(repl.eval('1 + 2')); // "3 :: Int"
  });
</script>
```

**Docusaurus / React:** Copy `ailang.wasm` + `wasm_exec.js` to `docs/static/wasm/`, copy `web/ailang-repl.js` + `web/AilangRepl.jsx` to `docs/src/components/`, add `/wasm/wasm_exec.js` (`async: false`) to `scripts` in `docusaurus.config.js`, and render `<AilangRepl />` in MDX.

</details>

## JavaScript API (`AilangREPL`)

| Method | Returns | Purpose |
|---|---|---|
| `init(wasmPath)` / `onReady(cb)` | `Promise` / `void` | Initialize WASM binary and embedded stdlib |
| `eval(input)` / `command(cmd)` / `reset()` | `string` | Evaluate expression, run REPL command (`:type`, `:help`), or reset state |
| `getVersion()` / `getVersionInfo()` / `needsContinuation(line)` | `string \| Object \| bool` | Version info and multi-line input check |
| `loadModule(name, code)` / `listModules()` / `importModule(name)` | `{success, exports?, error?}` | Compile and register AILANG modules (explicit `export` or REPL-style) |
| `call(mod, fn, ...args)` | `{success, result?, error?}` | Call exported function directly (supports `number`, `string`, `boolean`, `array`) |
| `setEffectHandler(cap, op, fn)` / `setAIHandler(fn)` / `grantCapability(cap)` | `{success, error?}` | Bridge AILANG capabilities (`IO`, `FS`, `Net`, `AI`, `Clock`) to JS callbacks |
| `evalAsync(input)` / `callAsync(mod, fn, ...args)` | `Promise` | Async variants required when JS effect handlers return `Promise`s |

```javascript
const repl = new AilangREPL();
await repl.init('/wasm/ailang.wasm');

repl.loadModule('math', `
let add: Int -> Int -> Int = \\x. \\y. x + y
let greet = \\name. "Hello, " <> name
`);
console.log(repl.call('math', 'add', 2, 3).result); // "5 :: Int"
```

<details>
<summary>Complete invoice processor HTML example</summary>

```html
<script src="wasm_exec.js"></script>
<script src="ailang-repl.js"></script>
<script>
  const repl = new AilangREPL();
  repl.init('/wasm/ailang.wasm').then(() => {
    const result = repl.loadModule('invoice', `
let calculateTotal: Int -> Float -> Float = \\quantity. \\unitPrice.
  int_to_float(quantity) * unitPrice
let applyDiscount: Float -> Float -> Float = \\total. \\discountPercent.
  total * (1.0 - discountPercent / 100.0)
    `);
    if (result.success) {
      console.log(repl.call('invoice', 'applyDiscount', 99.95, 10).result);
    }
  });
</script>
```

</details>

## Effect Handlers in the Browser

In the browser, JavaScript callbacks back AILANG's `! {IO}`, `! {Net}`, `! {Clock}`, and `! {AI}` effects. Use `ailangEvalAsync` / `ailangCallAsync` (or `repl.evalAsync` / `repl.callAsync`) whenever any handler is `async`:

```javascript
ailangSetEffectHandler("IO", {
  print:    (msg) => { output.textContent += msg; },
  println:  (msg) => { output.textContent += msg + '\n'; },
  readLine: ()    => prompt("Enter input:")
});

ailangSetAIHandler(async (input) => {
  const resp = await fetch('/api/ai', { method: 'POST', body: input });
  return await resp.text(); // auto-grants AI capability
});

repl.loadModule('demo', `
module demo
import std/ai as AI
export func ask(question: string) -> string ! {AI} = AI.call(question)
`);

const answer = await ailangCallAsync('demo', 'ask', 'What is 2+2?');
```

For typed multi-turn `std/ai` (`step`, `stepWithCache`, `stepWithStream`), see [Browser `ai.step` (BYO API key)](./wasm-ai-step-byo-key.md).

## Cognitive OS Substrate

The WASM build includes the Go-side **Cognitive OS substrate** for deterministic, replayable browser agents (`examples/cognitive_os/single_agent_replay.ail`):

| Effect | Stdlib | JS Bridge / Status |
|---|---|---|
| `DOM` | `std/dom` | Scoped DOM mutations (`AddPanel`, `UpdateNode`, `RemoveNode`, `AddTimeline`, `applyPatchResult`); `ailangSetDOMApplyPatchHandler` / `ailangSetDOMApplyBatchHandler` |
| `Msg` | `std/cognition` | Transport-independent mailbox messaging (`sendMsgResult`); `ailangSetMsgSendHandler` / `ailangSetMsgRecvHandler` |
| `Trace` | `std/trace` | Cognitive event log with Lamport-clock JSONL replay |

```ailang
module myagent

import std/dom (DOMPatch, AddPanel, applyPatchResult)
import std/cognition (sendMsgResult)
import std/result (Result, Ok, Err)

export func renderHeatmap(region: string) -> string ! {DOM} = {
  match applyPatchResult(region, AddPanel("Status", "Running...")) {
    Ok(r)  => "node:${r.node_id}",
    Err(e) => "failed:${e.code}"
  }
}

export func relay(to: string, payload: string) -> string ! {Msg} = {
  match sendMsgResult(to, payload) {
    Ok(r)  => "sent:${r.msg_id}@${show(r.clock)}",
    Err(e) => "send_failed:${e.code}"
  }
}
```

## Capabilities, Deployment & Troubleshooting

| Feature | CLI | WASM |
|---|---|---|
| Stdlib (`std/*`) | Yes | Embedded (`importModule('std/list')`) |
| Effects (`IO`, `Net`, `FS`, `AI`, `Clock`) | `--caps` | JS callbacks (`Net` subject to CORS; `FS` via JS storage) |
| Custom file imports | Yes | Use `loadModule()` |
| REPL history persistence | Yes | No |

<details>
<summary>Static hosting, CI/CD bundling, and common troubleshooting</summary>

- **Nginx / CDN**: Enable `gzip_types application/wasm;`, set `Cache-Control "public, max-age=31536000, immutable"`, and add `<link rel="preload" href="/wasm/ailang.wasm" as="fetch" crossorigin>`.
- **"ambiguous type variable α with classes [Num]"**: Add explicit parameter/return type annotations on numeric REPL functions (`let add: Int -> Int -> Int = ...`).
- **"module X not loaded"**: Call `repl.loadModule(name, code)` before `importModule(name)` or `call(name, ...)`.
- **Effects silent or failing**: Register handlers or call `repl.grantCapability(...)` before running, and use `evalAsync`/`callAsync` for Promise-returning callbacks.

</details>

## Next Steps

- [Live Playground](/docs/playground) · [Browser `ai.step` (BYO key)](./wasm-ai-step-byo-key) · [Web Example Source](https://github.com/sunholo-data/ailang/blob/main/web/example.mdx)
