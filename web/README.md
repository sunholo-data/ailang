# AILANG Web REPL

Browser-based AILANG REPL powered by WebAssembly.

## Quick Start

### 1. Build WASM Binary

```bash
# From ailang root directory
make build-wasm
```

This creates `bin/ailang.wasm`.

### 2. Setup for Docusaurus

```bash
# Copy files to your Docusaurus project
cp bin/ailang.wasm <docusaurus-site>/static/wasm/
cp web/ailang-repl.js <docusaurus-site>/src/components/
cp web/AilangRepl.jsx <docusaurus-site>/src/components/

# Copy Go's WASM support file
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" <docusaurus-site>/static/wasm/
```

### 3. Load WASM Support in Docusaurus

Edit `<docusaurus-site>/docusaurus.config.js`:

```js
module.exports = {
  // ... other config
  scripts: [
    {
      src: '/wasm/wasm_exec.js',
      async: false,
    },
  ],
};
```

### 4. Use in MDX Pages

```mdx
---
title: Try AILANG
---

import AilangRepl from '@site/src/components/AilangRepl';

## Interactive REPL

Try AILANG right in your browser:

<AilangRepl />

## Examples

Try these expressions:
- `1 + 2`
- `:type \x. x + x`
- `let f = \x. x * 2 in f(21)`
```

## Build Target

Add to your `Makefile`:

```makefile
build-wasm:
	@echo "Building WASM binary..."
	GOOS=js GOARCH=wasm go build -o bin/ailang.wasm cmd/wasm/main.go
	@echo "✓ WASM binary: bin/ailang.wasm"
```

## API Reference

### JavaScript API

```js
import AilangREPL from './ailang-repl.js';

// Initialize
const repl = new AilangREPL();
await repl.init('/wasm/ailang.wasm');

// Evaluate expressions
const result = repl.eval('1 + 2');
console.log(result); // "3 :: Int"

// Execute commands
repl.command(':type \x. x');

// Reset environment
repl.reset();

// Get version
const version = repl.getVersion();
```

### React Component Props

```jsx
<AilangRepl
  // No props needed - fully self-contained
/>
```

## Customization

### Styling

Edit the `styles` object in `AilangRepl.jsx`:

```jsx
const styles = {
  container: {
    backgroundColor: '#1e1e1e',  // Change background
    color: '#d4d4d4',            // Change text color
    // ... more styles
  },
};
```

### Theme Integration

For Docusaurus dark/light theme support:

```jsx
import { useColorMode } from '@docusaurus/theme-common';

export default function AilangRepl() {
  const { colorMode } = useColorMode();

  const styles = {
    container: {
      backgroundColor: colorMode === 'dark' ? '#1e1e1e' : '#f5f5f5',
      // ...
    },
  };
  // ...
}
```

## Limitations

- **No File I/O**: File system effects disabled in browser
- **No History Persistence**: History lost on page reload
- **Binary Size**: ~10MB (compresses well with gzip/brotli)
- **First Load**: May take 2-3 seconds to initialize

## Troubleshooting

### "WebAssembly not supported"
- Use a modern browser (Chrome 57+, Firefox 52+, Safari 11+)

### "Failed to load AILANG WASM"
- Check browser console for details
- Verify `ailang.wasm` is in `/static/wasm/`
- Ensure `wasm_exec.js` is loaded first

### "REPL not initialized"
- Wait for `onReady()` callback
- Check network tab for 404s

### "Module not found: './ailang-repl.js'"
- Ensure `ailang-repl.js` is in `src/components/`
- Check import path matches your directory structure

## Advanced: Building for Production

### Optimize Binary Size

```bash
# Build with size optimization
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o bin/ailang.wasm cmd/wasm/main.go

# Further compress (optional)
wasm-opt -Oz bin/ailang.wasm -o bin/ailang.wasm
```

### Enable HTTP Compression

Configure your web server to serve `.wasm` with compression:

**Nginx:**
```nginx
gzip_types application/wasm;
```

**Cloudflare/Netlify/Vercel:** Automatic compression enabled by default.

## Examples

See [examples/web/](../examples/web/) for complete Docusaurus integration examples.

## License

Same as AILANG project (see main LICENSE file).
