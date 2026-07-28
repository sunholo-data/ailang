#!/usr/bin/env bash
# test_example.sh - Test if an AILANG code example compiles and runs correctly
#
# Usage: ./test_example.sh <file.ail>
# Example: ./test_example.sh /tmp/test.ail
#
# Creates a test file if code is passed via stdin:
#   echo 'module test/example...' | ./test_example.sh -
#
# Output: Success/failure with any error messages

set -euo pipefail

FILE="${1:-}"

if [[ -z "$FILE" ]]; then
    echo "Usage: $0 <file.ail>"
    echo "       echo 'code...' | $0 -"
    echo ""
    echo "Example:"
    echo "  $0 /tmp/test.ail"
    echo "  echo 'module test/ex' | $0 -"
    exit 1
fi

# Handle stdin
if [[ "$FILE" == "-" ]]; then
    FILE=$(mktemp /tmp/ailang_test_XXXXXX.ail)
    cat > "$FILE"
    trap "rm -f '$FILE'" EXIT
fi

if [[ ! -f "$FILE" ]]; then
    echo "Error: File not found: $FILE"
    exit 1
fi

echo "=== Testing AILANG Example ==="
echo "File: $FILE"
echo ""

# Show the code
echo "--- Code ---"
cat "$FILE"
echo ""
echo "--- End Code ---"
echo ""

# Step 1: Type check
echo "Step 1: Type checking..."
if ailang check "$FILE" 2>&1; then
    echo "  Type check passed"
else
    echo "  Type check FAILED"
    echo ""
    echo "=== RESULT: COMPILE ERROR ==="
    echo "This example has a type/syntax error and should NOT be added to prompts."
    echo "Consider creating a design doc for the language limitation."
    exit 1
fi

# Step 2: Check if it has a main function
HAS_MAIN=$(grep -c 'func main\|export func main' "$FILE" || true)

if [[ "$HAS_MAIN" -gt 0 ]]; then
    echo ""
    echo "Step 2: Running (has main function)..."

    # Determine required capabilities
    CAPS=""
    if grep -q 'IO\|println\|print\|readLine' "$FILE"; then
        CAPS="${CAPS}IO,"
    fi
    if grep -q 'FS\|readFile\|writeFile' "$FILE"; then
        CAPS="${CAPS}FS,"
    fi
    if grep -q 'Clock\|now\|sleep' "$FILE"; then
        CAPS="${CAPS}Clock,"
    fi
    if grep -q 'Net\|http\|fetch' "$FILE"; then
        CAPS="${CAPS}Net,"
    fi

    # Remove trailing comma
    CAPS="${CAPS%,}"

    if [[ -z "$CAPS" ]]; then
        CAPS="IO"  # Default to IO for println
    fi

    echo "  Using capabilities: $CAPS"
    echo ""
    echo "--- Output ---"

    if OUTPUT=$(ailang run --caps "$CAPS" --entry main "$FILE" 2>&1); then
        echo "$OUTPUT"
        echo "--- End Output ---"
        echo ""
        echo "=== RESULT: SUCCESS ==="
        echo "This example compiles and runs correctly."
        echo "Safe to add to prompts!"
        exit 0
    else
        echo "$OUTPUT"
        echo "--- End Output ---"
        echo ""
        echo "=== RESULT: RUNTIME ERROR ==="
        echo "This example compiles but fails at runtime."
        echo "Check for logic errors or missing capabilities."
        exit 1
    fi
else
    echo ""
    echo "Step 2: No main function found (pure module)"
    echo ""
    echo "=== RESULT: TYPE CHECK ONLY ==="
    echo "Example type-checks but cannot be run (no main)."
    echo "Add a main function to test full behavior."
    exit 0
fi
