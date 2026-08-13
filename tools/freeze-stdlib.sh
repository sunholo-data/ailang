#!/bin/bash
# freeze-stdlib.sh - Generate SHA256 golden files for stdlib interfaces
#
# RE-FREEZING ACCEPTS WHATEVER THE INTERFACES CURRENTLY ARE. It is not a check.
# Run it only when you have decided the current interfaces are correct, and say
# so in the commit — a silent re-freeze turns the gate into a rubber stamp.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=tools/stdlib-iface-lib.sh
source "$SCRIPT_DIR/stdlib-iface-lib.sh"

resolve_ailang
mkdir -p "$GOLDEN_DIR"

echo "Freezing stdlib interfaces (binary: $AILANG)..."
echo

# Drop goldens for modules that no longer exist, or the gate quietly keeps
# verifying a deleted module against a stale snapshot forever.
CURRENT_MODULES="$(stdlib_modules)"
for existing in "$GOLDEN_DIR"/*.json; do
    [ -e "$existing" ] || continue
    name="$(basename "$existing" .json)"
    if ! echo "$CURRENT_MODULES" | grep -qx "$name"; then
        echo "  - $name (module gone — removing stale golden)"
        rm -f "$GOLDEN_DIR/$name.json" "$GOLDEN_DIR/$name.sha256"
    fi
done

COUNT=0
for module in $CURRENT_MODULES; do
    JSON_FILE="$GOLDEN_DIR/$module.json"
    HASH_FILE="$GOLDEN_DIR/$module.sha256"

    iface_json "$module" "$JSON_FILE"
    sha256_of "$JSON_FILE" > "$HASH_FILE"

    HASH="$(cat "$HASH_FILE")"
    echo "  ✓ $module (SHA256: ${HASH:0:16}...)"
    COUNT=$((COUNT + 1))
done

echo
echo "✓ Froze $COUNT stdlib interfaces in $GOLDEN_DIR/"
echo "  Run 'make verify-stdlib' to check for API changes"
