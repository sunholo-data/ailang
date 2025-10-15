#!/bin/bash
# verify-stdlib.sh - Verify stdlib interfaces haven't changed

set -e

STDLIB_DIR="stdlib/std"
GOLDEN_DIR=".stdlib-golden"

# List of stdlib modules
MODULES="io list option result string"

echo "Verifying stdlib interface stability..."
echo

FAILED=0

for module in $MODULES; do
    MODULE_PATH="$STDLIB_DIR/$module.ail"
    GOLDEN_JSON="$GOLDEN_DIR/$module.json"
    GOLDEN_HASH="$GOLDEN_DIR/$module.sha256"

    if [ ! -f "$GOLDEN_HASH" ]; then
        echo "✗ $module: No golden file found"
        echo "  Run 'make freeze-stdlib' to create golden files"
        FAILED=1
        continue
    fi

    EXPECTED_HASH=$(cat "$GOLDEN_HASH")

    # Generate current JSON
    CURRENT_JSON=$(mktemp)
    ailang iface "$MODULE_PATH" 2>/dev/null | grep -A 10000 '^{' > "$CURRENT_JSON"

    # Compute current hash
    if command -v sha256sum >/dev/null 2>&1; then
        CURRENT_HASH=$(sha256sum "$CURRENT_JSON" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        CURRENT_HASH=$(shasum -a 256 "$CURRENT_JSON" | awk '{print $1}')
    else
        echo "Error: No SHA256 tool found" >&2
        exit 1
    fi

    if [ "$CURRENT_HASH" = "$EXPECTED_HASH" ]; then
        echo "✓ $module (SHA256: ${EXPECTED_HASH:0:16}...)"
    else
        echo "✗ $module: Interface changed!"
        echo "  Expected: $EXPECTED_HASH"
        echo "  Got:      $CURRENT_HASH"
        echo
        echo "  Diff:"
        diff -u "$GOLDEN_JSON" "$CURRENT_JSON" || true
        echo
        FAILED=1
    fi

    rm -f "$CURRENT_JSON"
done

echo

if [ $FAILED -eq 0 ]; then
    echo "✓ All stdlib interfaces stable"
    exit 0
else
    echo "✗ Stdlib interface verification failed"
    echo "  If changes are intentional, run 'make freeze-stdlib' to update golden files"
    exit 1
fi
