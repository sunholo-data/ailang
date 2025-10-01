#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EX_DIR="$ROOT/examples"
OK=0

shopt -s nullglob
for src in "$EX_DIR"/*.ail; do
  base=$(basename "$src" .ail)
  golden="$EX_DIR/${base}.golden"
  out=$(mktemp)
  if ! ailang run "$src" > "$out"; then
    echo "RUN FAIL: $src"; OK=1; continue
  fi
  if [ ! -f "$golden" ]; then
    echo "MISSING GOLDEN: $golden"
    echo "Create with: ailang run $src > $golden"
    OK=1; continue
  fi
  if ! diff -u "$golden" "$out" >/dev/null; then
    echo "GOLDEN MISMATCH: $src"
    diff -u "$golden" "$out" || true
    OK=1
  fi
done

exit $OK
