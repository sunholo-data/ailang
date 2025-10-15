---
id: wasm-integration
title: WebAssembly Integration Guide
sidebar_label: WASM Integration
---

# WebAssembly Integration Guide

AILANG can run entirely in the browser using WebAssembly, enabling interactive demonstrations and online playgrounds without requiring server-side execution.

## Overview

The AILANG WebAssembly build provides:

- **Full Language Support**: Complete AILANG interpreter compiled to WASM
- **Client-Side Execution**: No server needed after initial load
- **Small Bundle Size**: ~5.7MB uncompressed (~1-2MB with gzip)
- **React Integration**: Ready-made component for easy integration
- **Offline Capable**: Works offline after first load

## Quick Start

### 1. Build WASM Binary

```bash
cd ailang
make build-wasm
```

This produces `bin/ailang.wasm`.

### 2. Integration Options

#### Option A: Docusaurus (Recommended)

1. Copy assets:
```bash
cp bin/ailang.wasm docs/static/wasm/
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" docs/static/wasm/
cp web/ailang-repl.js docs/src/components/
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
| Expression evaluation | ✅ | ✅ |
| Type inference | ✅ | ✅ |
| Pattern matching | ✅ | ✅ |
| Type classes | ✅ | ✅ |
| File I/O (`FS` effect) | ✅ | ❌ |
| Module imports | ✅ | ❌ |
| History persistence | ✅ | ❌ |

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

WASM is automatically built and released:

```yaml
# .github/workflows/release.yml (excerpt)
- name: Build WASM binary
  run: make build-wasm

- name: Create Release
  uses: softprops/action-gh-release@v2
  with:
    files: bin/ailang-wasm.tar.gz
```

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

## Examples

See:
- [Live Playground](/docs/playground) - Try it now
- [Integration Example](https://github.com/sunholo-data/ailang/blob/main/web/example.mdx)
- [Component Source](https://github.com/sunholo-data/ailang/blob/main/web/AilangRepl.jsx)

## Next Steps

- [Try the Playground](/docs/playground)
- [Download Latest Release](https://github.com/sunholo-data/ailang/releases/latest)
- [Report Issues](https://github.com/sunholo-data/ailang/issues)
