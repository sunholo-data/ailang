#!/bin/bash
# Cheap drift guardrail: fails the build if docs/docs/reference/stdlib.md
# omits any module reported by `ailang docs --list`, or lists modules
# that no longer exist. Run automatically during docs build via sync-all.
#
# To update: edit docs/docs/reference/stdlib.md, add the new module under
# its domain group with purpose + capability, then re-run the build.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCS_ROOT="$(dirname "$SCRIPT_DIR")"
INDEX_FILE="$DOCS_ROOT/docs/reference/stdlib.md"

if ! command -v ailang >/dev/null 2>&1; then
  echo "check-stdlib-index: 'ailang' not found in PATH — skipping drift check."
  echo "  (Install with 'make install' from the repo root to enable this guardrail.)"
  exit 0
fi

if [ ! -f "$INDEX_FILE" ]; then
  echo "check-stdlib-index: $INDEX_FILE not found"
  exit 1
fi

# Modules the binary knows about. The list lines look like:
#     std/io         Print to stdout and read from stdin.
listed=$(ailang docs --list 2>/dev/null | awk '/^  std\// {print $1}' | sort -u)

# Modules documented in the index. We grep for backticked entries like
# `std/io` and de-dupe — the prose mentions `std/foo` only in tables.
documented=$(grep -oE '`std/[a-zA-Z_]+`' "$INDEX_FILE" | tr -d '`' | sort -u)

if [ -z "$listed" ]; then
  echo "check-stdlib-index: 'ailang docs --list' returned no modules — refusing to validate."
  echo "  (Is the binary stale? Run 'make quick-install' from the repo root.)"
  exit 1
fi

missing=$(comm -23 <(printf '%s\n' "$listed") <(printf '%s\n' "$documented") || true)
extra=$(comm -13 <(printf '%s\n' "$listed") <(printf '%s\n' "$documented") || true)

if [ -n "$missing" ] || [ -n "$extra" ]; then
  echo ""
  echo "✗ Stdlib Index drift detected — $INDEX_FILE is out of sync with the binary."
  echo ""
  if [ -n "$missing" ]; then
    echo "  Modules in stdlib but missing from the index (add these):"
    printf '    - %s\n' $missing
    echo ""
  fi
  if [ -n "$extra" ]; then
    echo "  Modules listed in the index but no longer in stdlib (remove these):"
    printf '    - %s\n' $extra
    echo ""
  fi
  echo "  Edit $INDEX_FILE under the appropriate domain group and re-run."
  exit 1
fi

count=$(printf '%s\n' "$listed" | wc -l | tr -d ' ')
echo "✓ Stdlib Index in sync ($count modules)"
