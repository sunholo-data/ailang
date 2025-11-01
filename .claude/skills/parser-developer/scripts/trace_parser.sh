#!/usr/bin/env bash
set -euo pipefail

# Trace AILANG parser with DEBUG_PARSER=1

if [ $# -lt 1 ]; then
    echo "Usage: $0 <file.ail>"
    echo ""
    echo "Traces parser token positions for debugging."
    echo "Shows ENTER/EXIT for parse functions with cur/peek tokens."
    exit 1
fi

FILE="$1"

if [ ! -f "$FILE" ]; then
    echo "✗ File not found: $FILE"
    exit 1
fi

echo "Tracing parser for: $FILE"
echo "========================="
echo ""

# Run with DEBUG_PARSER=1
DEBUG_PARSER=1 ailang run "$FILE" 2>&1 || true

echo ""
echo "Legend:"
echo "  cur  = current token (parser is AT this token)"
echo "  peek = next token (parser can see ahead)"
echo ""
echo "Convention: Parser functions leave cursor AT last token (not after)"
