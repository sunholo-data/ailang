---
id: wasm-integration
title: WebAssembly Integration Guide
sidebar_label: WASM Integration
---

# WebAssembly Integration Guide

AILANG can run entirely in the browser using WebAssembly, enabling interactive demonstrations and online playgrounds without requiring server-side execution.

:::tip See WASM in Action
All AILANG browser demos run on WebAssembly — try them live:
- **[DocParse](https://www.sunholo.com/ailang-demos/docparse.html)** — Parse DOCX, PPTX, XLSX, PDF entirely in-browser
- **[Document Extractor](https://www.sunholo.com/ailang-demos/extractor.html)** — AI-powered extraction using WASM modules
- **[All 9 Demos](https://www.sunholo.com/ailang-demos/)** — Full demo gallery
:::

## Overview

The AILANG WebAssembly build provides:

- **Full Language Support**: Complete AILANG interpreter compiled to WASM
- **Standard Library Included**: All 20 stdlib modules (`std/list`, `std/json`, etc.) embedded and auto-loaded
- **Client-Side Execution**: No server needed after initial load
- **Small Bundle Size**: ~33MB uncompressed (~7MB with gzip)
- **React Integration**: Ready-made component for easy integration
- **Offline Capable**: Works offline after first load

## Quick Start

### Option 1: From Release (Recommended)

Download the pre-built WASM bundle from the latest release. All three files are version-matched and built from the same commit:

```bash
# Download and extract — includes ailang.wasm, wasm_exec.js, ailang-repl.js
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-wasm.tar.gz | tar -xz
```

The archive contains:

| File | Size | Description |
|------|------|-------------|
| `ailang.wasm` | ~33MB | Compiled WASM binary with embedded stdlib |
| `wasm_exec.js` | ~17KB | Go WASM runtime (version-matched to build toolchain) |
| `ailang-repl.js` | ~5KB | JavaScript wrapper class (`AilangREPL`) |

### Option 2: Build from Source

```bash
cd ailang
make build-wasm

# Copy wasm_exec.js from your Go installation
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
```

This produces `bin/ailang.wasm`. You'll also need [ailang-repl.js](https://github.com/sunholo-data/ailang/blob/main/web/ailang-repl.js) from the `web/` directory.

### Integration Options

#### Option A: Docusaurus (Recommended)

1. Copy assets:
```bash
cp ailang.wasm docs/static/wasm/
cp wasm_exec.js docs/static/wasm/
cp ailang-repl.js docs/src/components/
cp web/AilangRepl.jsx docs/src/components/
```

2. Add to `docusaurus.config.js`:
```javascript
module.exports = {
  scripts: [
    {
      src: '/wasm/wasm_exec.js',
      async: false,
    },
  ],
  // ... rest of config
};
```

3. Use in MDX:
```mdx
---
title: Try AILANG
---

import AilangRepl from '@site/src/components/AilangRepl';

<AilangRepl />
```

#### Option B: Vanilla HTML

```html
<!DOCTYPE html>
<html>
<head>
  <title>AILANG REPL</title>
  <script src="wasm_exec.js"></script>
  <script src="ailang-repl.js"></script>
</head>
<body>
  <div id="repl-container"></div>

  <script>
    const repl = new AilangREPL();

    repl.init('/path/to/ailang.wasm').then(() => {
      console.log('AILANG ready!');

      // Evaluate expressions
      const result = repl.eval('1 + 2');
      console.log(result); // "3 :: Int"
    });
  </script>
</body>
</html>
```

#### Option C: React (Custom)

```jsx
import { useEffect, useState } from 'react';
import AilangREPL from './ailang-repl';

export default function MyReplComponent() {
  const [repl, setRepl] = useState(null);
  const [result, setResult] = useState('');

  useEffect(() => {
    const replInstance = new AilangREPL();
    replInstance.init('/wasm/ailang.wasm').then(() => {
      setRepl(replInstance);
    });
  }, []);

  const handleEval = (input) => {
    if (repl) {
      const output = repl.eval(input);
      setResult(output);
    }
  };

  return (
    <div>
      <input onKeyDown={(e) => {
        if (e.key === 'Enter') handleEval(e.target.value);
      }} />
      <pre>{result}</pre>
    </div>
  );
}
```

## JavaScript API

### `AilangREPL` Class

```javascript
const repl = new AilangREPL();
```

#### Methods

##### `init(wasmPath)`

Initialize the WASM module.

```javascript
await repl.init('/wasm/ailang.wasm');
```

**Parameters:**
- `wasmPath` (string): Path to `ailang.wasm` file

**Returns:** Promise that resolves when REPL is ready

##### `eval(input)`

Evaluate an AILANG expression.

```javascript
const result = repl.eval('1 + 2');
// Returns: "3 :: Int"
```

**Parameters:**
- `input` (string): AILANG code to evaluate

**Returns:** Result string (includes value and type)

##### `command(cmd)`

Execute a REPL command.

```javascript
const type = repl.command(':type \x. x');
// Returns: "\x. x :: a -> a"
```

**Parameters:**
- `cmd` (string): REPL command (e.g., `:type`, `:help`)

**Returns:** Command output string

##### `reset()`

Reset the REPL environment.

```javascript
repl.reset();
```

**Returns:** Status message

##### `onReady(callback)`

Register callback for when REPL is ready.

```javascript
repl.onReady(() => {
  console.log('REPL initialized!');
});
```

##### `getVersion()`

Get version information.

```javascript
const version = repl.getVersion();
// Returns: "v0.7.2" (or null if not ready)
```

**Returns:** Version string or `null`

##### `getVersionInfo()`

Get detailed version information.

```javascript
const info = repl.getVersionInfo();
// Returns: { version: "v0.7.2", buildTime: "2026-02-06T...", platform: "js/wasm" }
```

**Returns:** Object with `version`, `buildTime`, `platform` fields, or `null`

##### `needsContinuation(line)`

Check if a line needs continuation (for multi-line input UIs).

```javascript
repl.needsContinuation('let x = 5 in');  // true
repl.needsContinuation('1 + 2');          // false
```

**Returns:** `boolean`

### Module Loading API (v0.7.1+)

The WASM REPL supports loading complete AILANG modules, enabling complex browser-based demos with multiple function definitions.

:::tip Why Module Loading?
The REPL evaluates expressions line-by-line, so function definitions like `let add = \x. \y. x + y` don't persist in scope for subsequent calls. The Module Loading API solves this by compiling entire modules and storing their exports for later use.
:::

##### `loadModule(name, code)`

Load an AILANG module into the registry.

```javascript
const result = repl.loadModule('math', `
let add: Int -> Int -> Int = \\x. \\y. x + y
let mul: Int -> Int -> Int = \\x. \\y. x * y
`);

if (result.success) {
  console.log('Exports:', result.exports);
  // Exports: ["add", "mul"]
} else {
  console.error('Error:', result.error);
}
```

**Parameters:**
- `name` (string): Module name (e.g., `"math"`, `"invoice_processor"`)
- `code` (string): AILANG source code

**Returns:** Object with:
- `success` (boolean): Whether loading succeeded
- `exports` (string[]): List of exported function names (on success)
- `error` (string): Error message (on failure)

:::warning Type Annotations Required
For functions with numeric operations, you must include explicit type annotations:
```javascript
// ✅ Works
let add: Int -> Int -> Int = \\x. \\y. x + y

// ❌ Fails with "ambiguous type variable"
let add = \\x. \\y. x + y
```
:::

:::info Export Behavior (v0.7.1.2+)
Modules can use explicit exports or export all bindings:

**Explicit exports** (recommended for libraries):
```javascript
// Only public_func is exported, private_func is hidden
repl.loadModule('mylib', `
module mylib
export pure func public_func(x: int) -> int = x * 2
pure func private_func(x: int) -> int = x * 3
`);
// result.exports: ["public_func"]
```

**Implicit exports** (for REPL-style code):
```javascript
// All top-level bindings are exported
repl.loadModule('utils', `
let double: Int -> Int = \\x. x * 2
let triple: Int -> Int = \\x. x * 3
`);
// result.exports: ["double", "triple"]
```
:::

##### `listModules()` (v0.7.1+)

List all loaded modules.

```javascript
const modules = repl.listModules();
// Returns: ["math", "utils"]
```

**Returns:** Array of module names

##### `importModule(moduleName)` (v0.7.1+)

Import a module's exports into the REPL environment.

```javascript
repl.importModule('math');
// Now you can use: repl.eval('add(2)(3)')
```

**Parameters:**
- `moduleName` (string): Name of a loaded module

**Returns:** Import result message

##### `call(moduleName, funcName, ...args)`

Call a function from a loaded module. This is a convenience method that:
1. Looks up the function from the module registry
2. Converts JavaScript arguments to AILANG values
3. Invokes the function directly (no eval)

```javascript
// Load a module
repl.loadModule('math', `
let add: Int -> Int -> Int = \\x. \\y. x + y
let greet = \\name. "Hello, " <> name
`);

// Call functions
const sum = repl.call('math', 'add', 2, 3);
if (sum.success) {
  console.log(sum.result); // "5 :: Int"
}

const greeting = repl.call('math', 'greet', 'World');
if (greeting.success) {
  console.log(greeting.result); // "\"Hello, World\" :: String"
}
```

**Parameters:**
- `moduleName` (string): Module containing the function
- `funcName` (string): Function to call
- `...args` (any): Arguments (converted to AILANG values)

**Returns:** Object with:
- `success` (boolean): Whether the call succeeded
- `result` (string): Result with value and type (on success)
- `error` (string): Error message (on failure)

**Supported argument types:**
| JavaScript Type | AILANG Syntax |
|-----------------|---------------|
| `number` | `42` or `3.14` |
| `string` | `"text"` |
| `boolean` | `true` / `false` |
| `array` | `[1, 2, 3]` |

### Module Loading Example

Here's a complete example building an invoice processor demo:

```html
<!DOCTYPE html>
<html>
<head>
  <title>Invoice Processor Demo</title>
  <script src="wasm_exec.js"></script>
  <script src="ailang-repl.js"></script>
</head>
<body>
  <script>
    const repl = new AilangREPL();

    repl.init('/wasm/ailang.wasm').then(() => {
      // Load the invoice processor module
      const result = repl.loadModule('invoice', `
-- Invoice processing functions
let calculateTotal: Int -> Float -> Float = \\quantity. \\unitPrice.
  int_to_float(quantity) * unitPrice

let applyDiscount: Float -> Float -> Float = \\total. \\discountPercent.
  total * (1.0 - discountPercent / 100.0)

let formatCurrency: Float -> String = \\amount.
  "$" <> float_to_string(amount)
      `);

      if (!result.success) {
        console.error('Failed to load module:', result.error);
        return;
      }

      console.log('Loaded exports:', result.exports);
      // ["calculateTotal", "applyDiscount", "formatCurrency"]

      // Process an invoice
      const subtotal = repl.call('invoice', 'calculateTotal', 5, 19.99);
      if (subtotal.success) {
        console.log('Subtotal:', subtotal.result);
        // "99.95 :: Float"
      }

      const total = repl.call('invoice', 'applyDiscount', 99.95, 10);
      if (total.success) {
        console.log('After 10% discount:', total.result);
        // "89.955 :: Float"
      }

      // Or use eval directly after importing
      repl.importModule('invoice');
      const formatted = repl.eval('formatCurrency(89.95)');
      console.log('Formatted:', formatted);
      // "\"$89.95\" :: String"
    });
  </script>
</body>
</html>
```

### Effect Handlers API (v0.7.1+)

The WASM REPL supports configuring AILANG's algebraic effect system from JavaScript. This enables browser-based programs to use effects like AI, IO, Net, FS, and more by providing JavaScript callback implementations.

:::tip Why Effect Handlers?
AILANG tracks side effects in the type system. A function declared as `func ask(q: string) -> string ! {AI}` requires the `AI` capability to run. In the CLI, capabilities are granted via `--caps`. In the browser, you provide JavaScript implementations for each effect operation.
:::

#### Low-Level Globals

These functions are registered on the global `window` object when the WASM module loads:

| Global Function | Purpose |
|----------------|---------|
| `ailangSetEffectHandler(name, ops)` | Override any effect with JS callbacks |
| `ailangSetAIHandler(fn)` | Convenience wrapper for AI effect |
| `ailangGrantCapability(name)` | Grant a capability without a handler |
| `ailangEvalAsync(expr)` | Evaluate with async effect support |
| `ailangCallAsync(mod, fn, ...args)` | Call module function with async effects |

#### `AilangREPL` Wrapper Methods

The `AilangREPL` class wraps all low-level globals with error handling:

```javascript
const repl = new AilangREPL();
await repl.init('/wasm/ailang.wasm');

// Register effect handlers
repl.setEffectHandler('IO', 'print', (msg) => { output.textContent += msg; });
repl.setAIHandler(async (input) => { /* call LLM API */ });
repl.grantCapability('IO');

// Async evaluation (returns Promise)
const result = await repl.evalAsync('perform IO.print("hello")');
const output = await repl.callAsync('demo', 'processData', 'input');
```

:::info Wrapper vs Low-Level API
The wrapper methods (`repl.setEffectHandler()`) provide error handling and readiness checks. The low-level globals (`ailangSetEffectHandler()`) are called directly on `window` and skip these checks. For most use cases, prefer the wrapper methods.
:::

#### `ailangSetEffectHandler(effectName, handlers)`

Override any effect's operations with JavaScript callbacks. Auto-grants the named capability.

```javascript
// IO effect — route print/println to DOM
ailangSetEffectHandler("IO", {
  print:    (msg) => { output.textContent += msg; },
  println:  (msg) => { output.textContent += msg + '\n'; },
  readLine: ()    => prompt("Enter input:")
});

// Net effect — use browser fetch
ailangSetEffectHandler("Net", {
  httpRequest: async (method, url, headers, body) => {
    const resp = await fetch(url, {
      method,
      headers: JSON.parse(headers),
      body
    });
    return JSON.stringify({
      status: resp.status,
      body: await resp.text(),
      ok: resp.ok
    });
  }
});

// Clock effect — use JS Date
ailangSetEffectHandler("Clock", {
  now:   () => Date.now(),
  sleep: (ms) => new Promise(resolve => setTimeout(resolve, ms))
});
```

**Parameters:**
- `effectName` (string): Effect name matching the AILANG capability (e.g., `"IO"`, `"AI"`, `"Net"`)
- `handlers` (object): Map of operation names to JS functions

**Returns:** `{ success: true }` or `{ success: false, error: "..." }`

**Supported handler return types:**
- Synchronous values (string, number, boolean, null)
- Promises (resolved value is converted to AILANG)

#### `ailangSetAIHandler(callback)`

Convenience wrapper for the AI effect. Sets both the effect handler and the dedicated `AIHandler` on the effect context. This is the recommended way to enable `std/ai.call()` in the browser.

```javascript
// Connect to Gemini Flash API
ailangSetAIHandler(async function(input) {
  const resp = await fetch(
    `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=${API_KEY}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        contents: [{ parts: [{ text: input }] }]
      })
    }
  );
  const data = await resp.json();
  return data.candidates[0].content.parts[0].text;
});
```

**Parameters:**
- `callback` (function): JS function that takes a string and returns a string (or Promise\<string\>)

**Returns:** `{ success: true }` or `{ success: false, error: "..." }`

:::info Auto-Grant
`ailangSetAIHandler` automatically grants the `AI` capability and registers the handler in the effect registry. You don't need to call `ailangGrantCapability("AI")` separately.
:::

#### `ailangGrantCapability(name)`

Grant a named capability without providing a handler. Useful when the effect operations are already registered (e.g., built-in effects that work in WASM).

```javascript
ailangGrantCapability("IO");
ailangGrantCapability("Debug");
```

**Parameters:**
- `name` (string): Capability name

**Returns:** `{ success: true }` or `{ success: false, error: "..." }`

#### Valid Capabilities

| Capability | Description | Common Operations |
|------------|-------------|-------------------|
| `IO` | Console/display I/O | `print`, `println`, `readLine` |
| `FS` | File system access | `readFile`, `writeFile`, `exists` |
| `Net` | Network requests | `httpGet`, `httpPost`, `httpRequest` |
| `AI` | AI model completion | `complete` |
| `Clock` | Time operations | `now`, `sleep` |

#### `ailangEvalAsync(expression)`

Evaluate an AILANG expression asynchronously. Returns a JavaScript Promise. **Required when the expression may trigger effect handlers that return Promises** (e.g., AI calls, network requests).

```javascript
// Sync handler — ailangEval() works fine
ailangSetAIHandler((input) => "mock: " + input);
const result = repl.eval('import std/ai as AI in AI.call("hello")');

// Async handler — MUST use ailangEvalAsync
ailangSetAIHandler(async (input) => {
  const resp = await fetch('/api/ai', { method: 'POST', body: input });
  return await resp.text();
});
const result = await ailangEvalAsync('import std/ai as AI in AI.call("hello")');
```

**Parameters:**
- `expression` (string): AILANG expression or REPL command

**Returns:** Promise\<string\> — resolves with result string, rejects on error

#### `ailangCallAsync(moduleName, funcName, ...args)`

Call a module export asynchronously. Returns a JavaScript Promise. Same as `repl.call()` but supports async effect handlers.

```javascript
// Load a module that uses AI
repl.loadModule('demo', `
module demo
import std/ai as AI

export func askAI(question: string) -> string ! {AI} =
    AI.call(question)
`);

// Call with async AI handler
ailangSetAIHandler(async (input) => {
  // ... fetch from LLM API
  return "AI response";
});

const result = await ailangCallAsync('demo', 'askAI', 'What is 2+2?');
// result: { success: true, result: '"AI response" :: String' }
```

**Parameters:**
- `moduleName` (string): Module containing the function
- `funcName` (string): Function to call
- `...args` (any): Arguments (number, string, boolean)

**Returns:** `Promise<{ success, result?, error? }>`

### Effect Handler Example

Complete example using AI and IO effects in the browser:

```html
<!DOCTYPE html>
<html>
<head>
  <title>AILANG Effects Demo</title>
  <script src="wasm_exec.js"></script>
  <script src="ailang-repl.js"></script>
</head>
<body>
  <div id="output"></div>

  <script>
    const repl = new AilangREPL();
    const output = document.getElementById('output');

    repl.init('/wasm/ailang.wasm').then(async () => {
      // Set up IO effect — route output to DOM
      ailangSetEffectHandler("IO", {
        print:   (msg) => { output.textContent += msg; },
        println: (msg) => { output.textContent += msg + '\n'; }
      });

      // Set up AI effect — mock handler for demo
      ailangSetAIHandler((input) => {
        return "The answer is 42";
      });

      // Load a module that uses both effects
      repl.loadModule('demo', `
module demo
import std/ai as AI

export func greet(name: string) -> () ! {IO} =
    println("Hello, " ++ name ++ "!")

export func ask(question: string) -> string ! {AI} =
    AI.call(question)
      `);

      // Call IO function (sync is fine for sync handlers)
      repl.call('demo', 'greet', 'World');
      // DOM shows: "Hello, World!"

      // Call AI function
      const answer = await ailangCallAsync('demo', 'ask', 'What is the meaning of life?');
      console.log(answer.result);
      // '"The answer is 42" :: String'
    });
  </script>
</body>
</html>
```

### Complete Method Reference

| Method | Returns | Since | Description |
|--------|---------|-------|-------------|
| `init(wasmPath)` | `Promise<this>` | v0.7.0 | Initialize WASM module |
| `eval(input)` | `string` | v0.7.0 | Evaluate expression |
| `command(cmd)` | `string` | v0.7.0 | Execute REPL command |
| `reset()` | `string` | v0.7.0 | Reset environment + reload stdlib |
| `getVersion()` | `string` or `null` | v0.7.0 | Get version string |
| `getVersionInfo()` | `Object` or `null` | v0.7.0 | Get version, buildTime, platform |
| `needsContinuation(line)` | `boolean` | v0.7.0 | Check if line needs more input |
| `onReady(callback)` | `void` | v0.7.0 | Register ready callback |
| `loadModule(name, code)` | `{success, exports?, error?}` | v0.7.1 | Compile and register module |
| `listModules()` | `string[]` | v0.7.1 | List loaded module names |
| `call(mod, func, ...args)` | `{success, result?, error?}` | v0.7.1 | Call module export |
| `importModule(name)` | `string` | v0.7.1 | Import into REPL env |
| `setEffectHandler(cap, op, fn)` | `{success, error?}` | v0.7.2 | Register effect handler |
| `setAIHandler(fn)` | `{success, error?}` | v0.7.2 | Register AI handler |
| `grantCapability(cap)` | `{success, error?}` | v0.7.2 | Grant effect capability |
| `evalAsync(input)` | `Promise<string>` | v0.7.2 | Async eval (for effects) |
| `callAsync(mod, func, ...args)` | `Promise<{success, result?, error?}>` | v0.7.2 | Async call (for effects) |

## REPL Commands

The WebAssembly REPL supports the same commands as the CLI:

| Command | Description |
|---------|-------------|
| `:help` | Show available commands |
| `:type <expr>` | Display expression type |
| `:instances` | Show type class instances |
| `:reset` | Clear environment |

## Limitations

The browser version has these limitations compared to the CLI:

| Feature | CLI | WASM |
|---------|-----|------|
| Expression evaluation | Yes | Yes |
| Type inference | Yes | Yes |
| Pattern matching | Yes | Yes |
| Type classes | Yes | Yes |
| Module loading | Yes | Yes (v0.7.1+) |
| Standard library | Yes | Yes (v0.7.1+)* |
| Explicit exports | Yes | Yes (v0.7.1.2+) |
| Effect handlers | Yes (`--caps`) | Yes (v0.7.1+, via JS callbacks) |
| AI effect (`std/ai`) | Yes | Yes (v0.7.1+, via `ailangSetAIHandler`) |
| IO effect (print/read) | Yes (terminal) | Yes (v0.7.1+, via JS callbacks) |
| Net effect (HTTP) | Yes | Via JS callbacks (CORS applies) |
| FS effect (files) | Yes (filesystem) | Via JS callbacks (localStorage/IndexedDB) |
| Async evaluation | N/A | Yes (v0.7.1+, `ailangEvalAsync`) |
| Custom file imports | Yes | No (use `loadModule()`) |
| History persistence | Yes | No |

\* The standard library (20 modules including `std/list`, `std/json`, `std/string`) is embedded in the WASM binary and auto-loaded on init. Use `importModule('std/list')` to bring stdlib exports into scope. Custom user file imports require `loadModule()` instead.

## Deployment

### Static Hosting

WASM files work on any static host:

```bash
# Build and deploy
make build-wasm
cp bin/ailang.wasm your-site/static/wasm/
# Deploy your-site/ to Netlify/Vercel/GitHub Pages
```

### CDN Optimization

1. **Enable Compression:**
```nginx
# nginx.conf
gzip_types application/wasm;
```

2. **Set Cache Headers:**
```nginx
location ~* \.wasm$ {
  add_header Cache-Control "public, max-age=31536000, immutable";
}
```

3. **Use HTTP/2:**
WASM benefits from HTTP/2 multiplexing for faster loading.

### Performance Tips

- **Lazy Loading**: Only load WASM when user navigates to playground
- **Service Worker**: Cache WASM for offline use
- **CDN**: Serve from edge locations
- **Preload**: Add `<link rel="preload" href="ailang.wasm" as="fetch">`

## CI/CD Integration

### GitHub Actions

The release workflow automatically builds and bundles the WASM archive with all dependencies:

```yaml
# .github/workflows/release.yml (excerpt)
- name: Build WASM binary
  run: make build-wasm

- name: Bundle WASM archive with JS dependencies
  run: |
    # Get version-matched wasm_exec.js from Go toolchain
    cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" bin/
    cp web/ailang-repl.js bin/
    cd bin && tar -czf ailang-wasm.tar.gz ailang.wasm wasm_exec.js ailang-repl.js
```

The archive is verified (size checks for all 3 files) and attached to the GitHub Release as `ailang-wasm.tar.gz`.

### Docusaurus Deployment

WASM is included in documentation builds:

```yaml
# .github/workflows/docusaurus-deploy.yml (excerpt)
- name: Build WASM binary
  run: make build-wasm

- name: Copy static assets
  run: |
    cp bin/ailang.wasm docs/static/wasm/
    cp web/ailang-repl.js docs/src/components/
```

## Troubleshooting

### "WebAssembly not supported"

**Solution**: Use a modern browser:
- Chrome 57+
- Firefox 52+
- Safari 11+
- Edge 16+

### "Failed to load AILANG WASM"

**Solutions**:
1. Check browser console for network errors
2. Verify `ailang.wasm` path is correct
3. Ensure `wasm_exec.js` loaded first
4. Check CORS headers if serving from different domain

### "REPL not initialized"

**Solution**: Wait for `init()` promise or use `onReady()`:

```javascript
repl.init('/wasm/ailang.wasm').then(() => {
  // Safe to use repl here
  repl.eval('1 + 2');
});
```

### Slow Loading

**Solutions**:
1. Enable gzip compression (reduces to ~1-2MB)
2. Use CDN
3. Add preload hints:
   ```html
   <link rel="preload" href="/wasm/ailang.wasm" as="fetch" crossorigin>
   ```

### Effects Don't Work

**Solutions**:
1. Grant the capability first: `repl.grantCapability('IO')`
2. Register handlers: `repl.setEffectHandler('IO', 'print', fn)`
3. Use async methods if handlers return Promises: `await repl.evalAsync(...)`
4. Check that `ailangSetAIHandler` was called before `AI.call()` — it auto-grants the AI capability

### "undefined global variable: X from std/Y"

**Cause**: Module's stdlib import failed during initialization.

**Solution**: Check browser console for warnings during stdlib loading. Try `repl.reset()` to reload the standard library.

### Module Loading Errors

#### "ambiguous type variable α with classes [Num]"

**Cause**: Functions with numeric operations need explicit type annotations.

**Solution**: Add type annotations to function definitions:
```javascript
// ❌ Fails
let add = \\x. \\y. x + y

// ✅ Works
let add: Int -> Int -> Int = \\x. \\y. x + y
```

#### "module X not loaded (use loadModule first)"

**Cause**: Trying to call or import a module that hasn't been loaded.

**Solution**: Load the module before using it:
```javascript
// First load
repl.loadModule('mymodule', code);
// Then import or call
repl.importModule('mymodule');
```

#### "parse error" or "type error" from loadModule

**Cause**: Invalid AILANG syntax or type errors in module code.

**Solution**: Check the error message and fix the module code:
```javascript
const result = repl.loadModule('test', badCode);
if (!result.success) {
  console.error('Module error:', result.error);
  // e.g., "parse error: expected expression at line 3"
}
```

## Examples

See:
- [Live Playground](/docs/playground) - Try it now
- [Integration Example](https://github.com/sunholo-data/ailang/blob/main/web/example.mdx)
- [Component Source](https://github.com/sunholo-data/ailang/blob/main/web/AilangRepl.jsx)

## Next Steps

- [Try the Playground](/docs/playground)
- [Download Latest Release](https://github.com/sunholo-data/ailang/releases/latest)
- [Report Issues](https://github.com/sunholo-data/ailang/issues)
