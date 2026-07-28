#!/usr/bin/env bash
# watch_active.sh — Find the most recent active chain and launch `ailang chains live` on it.
#
# Usage:
#   watch_active.sh             # most recent active chain, default 3s refresh
#   watch_active.sh 1           # 1s refresh
#   watch_active.sh 5 --once    # single render then exit (script-friendly)

set -u

# Self-discover ailang if not on PATH (common when launchd starts a script
# without inheriting the user's PATH).
if ! command -v ailang >/dev/null 2>&1; then
  if [[ -x "$HOME/go/bin/ailang" ]]; then
    export PATH="$HOME/go/bin:$PATH"
  fi
fi

INTERVAL="${1:-3}"
shift 2>/dev/null || true
EXTRA_FLAGS="${*:-}"

if ! command -v ailang >/dev/null 2>&1; then
  echo "✗ ailang not on PATH. Build with 'make install' or export PATH=\$HOME/go/bin:\$PATH"
  exit 1
fi

# Find the most recent ACTIVE chain (status not in terminal state)
CHAIN_LINE=$(ailang chains list --since 1h --limit 5 2>/dev/null | grep -E "\bactive\b|\brunning\b" | head -1)

if [[ -z "$CHAIN_LINE" ]]; then
  # Fallback: most recent any-status chain
  CHAIN_LINE=$(ailang chains list --since 30m --limit 1 2>/dev/null | tail -1)
  if [[ -z "$CHAIN_LINE" || "$CHAIN_LINE" =~ "^ID" ]]; then
    echo "⚠ No recent chains found. Has an eval run been started?"
    echo "  Try: .claude/skills/local-ollama-eval/scripts/run_smoke.sh opencode-gemma4-26b fizzbuzz"
    exit 1
  fi
  echo "ℹ No active chain — showing most recent (may already be completed)"
fi

# Strip the trailing "..." ellipsis and any leftover chars; chain ids are
# 36-char UUIDs so use the first 8 chars as the prefix which `ailang chains
# live` will resolve.
CHAIN_ID=$(echo "$CHAIN_LINE" | awk '{print $1}' | sed 's/\.\.\.$//' | cut -c1-8)

if [[ -z "$CHAIN_ID" ]]; then
  echo "✗ Could not parse chain id from: $CHAIN_LINE"
  exit 1
fi

echo "Watching chain: $CHAIN_ID  (interval ${INTERVAL}s)"
echo "Ctrl-C to exit."
echo ""

exec ailang chains live "$CHAIN_ID" --interval "$INTERVAL" $EXTRA_FLAGS
