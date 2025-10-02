#!/bin/bash
# freeze-stdlib.sh - Generate SHA256 golden files for stdlib interfaces

set -e

STDLIB_DIR="stdlib/std"
GOLDEN_DIR=".stdlib-golden"

# Create golden directory if it doesn't exist
mkdir -p "$GOLDEN_DIR"

# List of stdlib modules
MODULES="io list option result string"

echo "Freezing stdlib interfaces..."
echo

for module in $MODULES; do
    MODULE_PATH="$STDLIB_DIR/$module.ail"
    JSON_FILE="$GOLDEN_DIR/$module.json"
    HASH_FILE="$GOLDEN_DIR/$module.sha256"

    echo "Processing $module..."

    # Generate normalized JSON (strip debug output)
    ailang iface "$MODULE_PATH" 2>/dev/null | grep -A 10000 '^{' > "$JSON_FILE"

    # Compute SHA256 hash
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$JSON_FILE" | awk '{print $1}' > "$HASH_FILE"
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$JSON_FILE" | awk '{print $1}' > "$HASH_FILE"
    else
        echo "Error: No SHA256 tool found (sha256sum or shasum)" >&2
        exit 1
    fi

    HASH=$(cat "$HASH_FILE")
    echo "  ✓ $module.json (SHA256: ${HASH:0:16}...)"
done

echo
echo "✓ Stdlib interfaces frozen in $GOLDEN_DIR/"
echo "  Run 'make verify-stdlib' to check for API changes"
