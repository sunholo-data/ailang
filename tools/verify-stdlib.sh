#!/bin/bash
# verify-stdlib.sh - Verify stdlib interfaces haven't changed
#
# Fails if any std/ module's exported interface differs from its frozen golden,
# and ALSO if a module has no golden at all — an unfrozen module is uncovered,
# not "passing".

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=tools/stdlib-iface-lib.sh
source "$SCRIPT_DIR/stdlib-iface-lib.sh"

resolve_ailang

echo "Verifying stdlib interface stability (binary: $AILANG)..."
echo

FAILED=0
CHECKED=0

MODULES="$(stdlib_modules)"
if [ -z "$MODULES" ]; then
    echo "✗ no modules found under $STDLIB_DIR/ — refusing to report success" >&2
    exit 1
fi

for module in $MODULES; do
    GOLDEN_JSON="$GOLDEN_DIR/$module.json"
    GOLDEN_HASH="$GOLDEN_DIR/$module.sha256"

    if [ ! -f "$GOLDEN_HASH" ]; then
        echo "✗ $module: no golden file — module is UNCOVERED"
        echo "    Run 'make freeze-stdlib' to snapshot it (and say so in the commit)"
        FAILED=1
        continue
    fi

    EXPECTED_HASH="$(cat "$GOLDEN_HASH")"

    CURRENT_JSON="$(mktemp)"
    if ! iface_json "$module" "$CURRENT_JSON"; then
        rm -f "$CURRENT_JSON"
        FAILED=1
        continue
    fi

    CURRENT_HASH="$(sha256_of "$CURRENT_JSON")"
    CHECKED=$((CHECKED + 1))

    if [ "$CURRENT_HASH" = "$EXPECTED_HASH" ]; then
        echo "✓ $module (SHA256: ${EXPECTED_HASH:0:16}...)"
    else
        echo "✗ $module: interface changed!"
        echo "    Expected: $EXPECTED_HASH"
        echo "    Got:      $CURRENT_HASH"
        echo "    Diff (golden → current):"
        diff -u "$GOLDEN_JSON" "$CURRENT_JSON" | sed 's/^/      /' || true
        echo
        FAILED=1
    fi

    rm -f "$CURRENT_JSON"
done

echo
if [ "$FAILED" -eq 0 ]; then
    echo "✓ All $CHECKED stdlib interfaces stable"
    exit 0
else
    echo "✗ Stdlib interface verification failed ($CHECKED verified before failures)"
    echo "  If the changes are intentional, run 'make freeze-stdlib' and state the"
    echo "  interface change in the commit message."
    exit 1
fi
