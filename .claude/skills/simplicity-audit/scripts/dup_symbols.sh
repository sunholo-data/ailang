#!/bin/bash
# dup_symbols.sh — which top-level function names are declared in several
# packages, and where. The "multiple implementations of the same thing" census.
#
# A name in >= N packages is not automatically a duplicate (Load, Parse, Open
# are honest names), but it is where duplicates hide: truncate ×11 with three
# semantics, fileExists ×4, writeJSON ×4 were all found this way.
#
# Usage: dup_symbols.sh [min-packages]   (default 3)
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
MIN="${1:-3}"
grep -rn --include='*.go' -E '^func [a-zA-Z_][a-zA-Z0-9_]*\(' internal cmd 2>/dev/null \
  | grep -v '_test\.go:' \
  | sed -E 's#^([^:]+)/[^/:]+:[0-9]+:func ([a-zA-Z_][a-zA-Z0-9_]*)\(.*#\2 \1#' \
  | grep -vE '^(init|main|New|Run|Name|String) ' \
  | sort -u \
  | awk -v min="$MIN" '
      { n[$1]++; where[$1] = where[$1] " " $2 }
      END { for (k in n) if (n[k] >= min) printf "%3d  %-28s%s\n", n[k], k, where[k] }' \
  | sort -rn
