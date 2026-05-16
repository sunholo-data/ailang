#!/usr/bin/env bash
# Bundles the AILANG VS Code extension source into a single extension.js
# committed at cmd/ailang/editor_assets/vscode/extension.js.
#
# Run after editing extension-src.js OR bumping the vscode-languageclient
# dependency. The committed extension.js is embedded into the ailang binary
# via //go:embed; users get it copied to ~/.vscode/extensions/ailang/ via
# `ailang editor install vscode`.
#
# Requires: node + npm on PATH. esbuild is fetched on demand if absent.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HERE/../.." && pwd)"
OUT="$REPO_ROOT/cmd/ailang/editor_assets/vscode/extension.js"

cd "$HERE"

echo "→ Installing build deps (vscode-languageclient + esbuild)..." >&2
npm install --silent --no-audit --no-fund

echo "→ Bundling extension.js → $OUT ..." >&2
mkdir -p "$(dirname "$OUT")"
npx --yes esbuild extension-src.js \
  --bundle \
  --platform=node \
  --target=node18 \
  --format=cjs \
  --external:vscode \
  --minify \
  --legal-comments=none \
  --outfile="$OUT" >&2

# Add a header so anyone reading the committed file knows where to regenerate.
TMP="$(mktemp)"
{
  echo "// AILANG VS Code extension — esbuild bundle."
  echo "// SOURCE: tools/vscode-extension-build/extension-src.js"
  echo "// REBUILD: bash tools/vscode-extension-build/bundle.sh"
  echo "// Bundles vscode-languageclient with the AILANG-specific activation logic."
  cat "$OUT"
} > "$TMP"
mv "$TMP" "$OUT"

SIZE=$(wc -c < "$OUT" | tr -d ' ')
echo "✓ Bundle ready ($SIZE bytes)" >&2
