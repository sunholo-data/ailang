#!/usr/bin/env bash
# Validate AILANG code syntax using ailang check

set -euo pipefail

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <file.ail>" >&2
    echo "Validates AILANG code syntax" >&2
    exit 1
fi

FILE="$1"

if [[ ! -f "$FILE" ]]; then
    echo "Error: File not found: $FILE" >&2
    exit 1
fi

# Check if ailang binary is available
if ! command -v ailang &> /dev/null; then
    echo "Error: 'ailang' command not found" >&2
    echo "Install with: make install" >&2
    exit 1
fi

# Run type check
echo "Validating $FILE..."
if ailang check "$FILE"; then
    echo "✓ Type check passed"
    exit 0
else
    echo "✗ Type check failed" >&2
    exit 1
fi
