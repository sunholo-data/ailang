#!/usr/bin/env bash
set -euo pipefail

# Quick API discovery using make doc

if [ $# -lt 2 ]; then
    echo "Usage: $0 <package> <symbol>"
    echo ""
    echo "Examples:"
    echo "  $0 internal/parser New"
    echo "  $0 internal/testing NewCollector"
    echo "  $0 internal/ast FuncDecl"
    exit 1
fi

PKG="$1"
SYMBOL="$2"

echo "Searching for '$SYMBOL' in $PKG..."
echo "======================================"
echo ""

if make doc PKG="$PKG" | grep -i "$SYMBOL"; then
    echo ""
    echo "✓ Found matches"
else
    echo "✗ No matches found"
    echo ""
    echo "Try:"
    echo "  grep -r '$SYMBOL' $PKG/"
fi
