# AILANG Web REPL

Browser-based AILANG REPL powered by WebAssembly.

## Quick Start

### From Release (Recommended)

Download the pre-built WASM bundle from the latest release:

```bash
# Download and extract — includes ailang.wasm, wasm_exec.js, ailang-repl.js
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-wasm.tar.gz | tar -xz
```

All three files are version-matched and ready to use.

### From Source

```bash
# Build WASM binary
make build-wasm

# Copy wasm_exec.js from your Go installation
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
```

## Usage

### Minimal HTML Example

```html
<!DOCTYPE html>
<html>
<head>
  <script src="wasm_exec.js"></script>
  <script src="ailang-repl.js"></script>
</head>
<body>
  <script>
    const repl = new AilangREPL();
    repl.init('/ailang.wasm').then(() => {
      console.log(repl.eval('1 + 2'));       // "3 :: Int"
      console.log(repl.getVersion());        // "v0.7.2"
    });
  </script>
</body>
</html>
```

### Docusaurus Integration

```bash
# Copy files to your Docusaurus project
cp ailang.wasm <docusaurus-site>/static/wasm/
cp wasm_exec.js <docusaurus-site>/static/wasm/
cp ailang-repl.js <docusaurus-site>/src/components/
cp web/AilangRepl.jsx <docusaurus-site>/src/components/
```

Load WASM support in `docusaurus.config.js`:

```js
module.exports = {
  scripts: [
    { src: '/wasm/wasm_exec.js', async: false },
  ],
};
```

Use in MDX pages:

```mdx
import AilangRepl from '@site/src/components/AilangRepl';

<AilangRepl />
```

## API Reference

### Core Methods

```js
const repl = new AilangREPL();

// Initialize with WASM binary path
await repl.init('/wasm/ailang.wasm');

// Evaluate expressions
repl.eval('1 + 2');                    // "3 :: Int"
repl.eval('let f = \\x. x * 2 in f(21)');  // "42 :: Int"

// Execute REPL commands
repl.command(':type \\x. x');          // "forall a. a -> a"
repl.command(':help');                 // Help text

// Reset environment (reloads stdlib)
repl.reset();                          // "Environment reset"

// Version info
repl.getVersion();                     // "v0.7.2"
repl.getVersionInfo();                 // { version, buildTime, platform }

// Multi-line input helper
repl.needsContinuation('let x = 5 in');  // true
```

### Module API (v0.7.1+)

Load and call AILANG modules from JavaScript:

```js
// Load a module
const result = repl.loadModule('math_utils', `
  module math_utils
  import std/math (intToFloat)
  export func circleArea(r: int) -> float =
    let rf = intToFloat(r)
    in 3.14159 * rf * rf
`);
// result: { success: true, exports: ['circleArea'] }

// List loaded modules
repl.listModules();
// ['std/prelude', 'std/math', ..., 'math_utils']

// Call an exported function
repl.call('math_utils', 'circleArea', 5);
// { success: true, result: '78.53975' }

// Import module exports into REPL environment
repl.importModule('math_utils');
// Now you can use: repl.eval('circleArea(10)')
```

### Effect Handlers (v0.7.2+)

Register JavaScript functions as AILANG effect handlers:

```js
// Grant capabilities
repl.grantCapability('IO');
repl.grantCapability('AI');

// Register IO.print handler
repl.setEffectHandler('IO', 'print', (msg) => {
  document.getElementById('output').textContent += msg + '\n';
  return msg;
});

// Register AI completion handler
repl.setAIHandler(async (prompt) => {
  const resp = await fetch('/api/chat', {
    method: 'POST',
    body: JSON.stringify({ prompt })
  });
  const data = await resp.json();
  return data.response;
});
```

### Async Methods (v0.7.2+)

Use async variants when effects may return Promises:

```js
// Async expression evaluation
const result = await repl.evalAsync('perform IO.print("hello")');

// Async module function call
const output = await repl.callAsync('my_module', 'processData', 'input');
// output: { success: true, result: '...' }
```

### Complete Method Reference

| Method | Returns | Since | Description |
|--------|---------|-------|-------------|
| `init(wasmPath)` | `Promise<this>` | v0.7.0 | Initialize WASM module |
| `eval(input)` | `string` | v0.7.0 | Evaluate expression |
| `command(cmd)` | `string` | v0.7.0 | Execute REPL command |
| `reset()` | `string` | v0.7.0 | Reset environment + reload stdlib |
| `getVersion()` | `string\|null` | v0.7.0 | Get version string |
| `getVersionInfo()` | `Object\|null` | v0.7.0 | Get `{version, buildTime, platform}` |
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

### Low-Level Window Globals

These are registered by the Go WASM binary. The `AilangREPL` class wraps all of them:

| Global | Description |
|--------|-------------|
| `ailangEval(input)` | Evaluate expression |
| `ailangReset()` | Reset REPL |
| `ailangVersion()` | Get version info object |
| `ailangLoadModule(name, code)` | Load module |
| `ailangListModules()` | List modules |
| `ailangCall(mod, func, ...args)` | Call export |
| `ailangSetEffectHandler(cap, op, fn)` | Register effect handler |
| `ailangSetAIHandler(fn)` | Register AI handler |
| `ailangGrantCapability(cap)` | Grant capability |
| `ailangEvalAsync(input)` | Async eval (returns Promise) |
| `ailangCallAsync(mod, func, ...args)` | Async call (returns Promise) |

## Valid Capabilities

| Capability | Description | Common Operations |
|------------|-------------|-------------------|
| `IO` | Console/display I/O | `print`, `readLine` |
| `FS` | File system access | `readFile`, `writeFile` |
| `Net` | Network requests | `httpGet`, `httpPost` |
| `AI` | AI model completion | `complete` |
| `Clock` | Time operations | `now` |

## Release Artifact

The `ailang-wasm.tar.gz` release asset contains:

| File | Size | Description |
|------|------|-------------|
| `ailang.wasm` | ~33MB | Compiled WASM binary |
| `wasm_exec.js` | ~17KB | Go WASM runtime (version-matched) |
| `ailang-repl.js` | ~5KB | JavaScript wrapper class |

All three files are version-matched and built from the same commit.

## Limitations

- **Effects require handlers**: IO/FS/Net/AI effects need JS handlers registered
- **No persistent storage**: State lost on page reload (use `loadModule` to restore)
- **Binary size**: ~33MB uncompressed (~7MB with gzip)
- **First load**: May take 2-3 seconds to initialize
- **Sync effects only**: Effect handlers must return synchronously unless using async methods

## Troubleshooting

### "WebAssembly not supported"
Use a modern browser (Chrome 57+, Firefox 52+, Safari 11+).

### "Failed to load AILANG WASM"
Check browser console. Verify `ailang.wasm` path is correct and CORS allows loading.

### "REPL not initialized"
Wait for `init()` to resolve or use `onReady()` callback.

### "undefined global variable: X from std/Y"
Module's stdlib import failed. Check browser console for warnings during stdlib loading.

### Effects don't work
1. Grant the capability first: `repl.grantCapability('IO')`
2. Register handlers: `repl.setEffectHandler('IO', 'print', fn)`
3. Use async methods: `await repl.evalAsync(...)` if handlers return Promises
