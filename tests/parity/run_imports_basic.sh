#!/usr/bin/env bash
# Test REPL/file parity for imports
# Ensures that importing and using a function produces the same result in both modes

set -euo pipefail

echo "Testing REPL/file parity for imports..."

# Run in file mode
FILE_OUT=$(bin/ailang run examples/v3_3/imports_basic.ail 2>&1 | tail -1)

# Run in REPL mode (heredoc to avoid terminal interaction)
REPL_OUT=$(cat <<'EOF' | bin/ailang repl 2>&1 | tail -1
import examples/v3_3/math/gcd (gcd)
gcd(48, 18)
EOF
)

# Compare outputs
if [ "$FILE_OUT" = "$REPL_OUT" ]; then
    echo "✓ REPL/file parity: PASS"
    echo "  Both modes output: $FILE_OUT"
    exit 0
else
    echo "✗ REPL/file parity: FAIL"
    echo "  File mode:  '$FILE_OUT'"
    echo "  REPL mode:  '$REPL_OUT'"
    exit 1
fi