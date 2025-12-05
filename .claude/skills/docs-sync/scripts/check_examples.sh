#!/bin/bash
# check_examples.sh - Validate that example files work
# Usage: ./check_examples.sh [--quick]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

QUICK_MODE=false
if [[ "$1" == "--quick" ]]; then
    QUICK_MODE=true
fi

echo "=== Example Files Validation ==="
echo ""

# Check examples directory
EXAMPLES_DIR="$PROJECT_ROOT/examples"
RUNNABLE_DIR="$PROJECT_ROOT/examples/runnable"

if [[ ! -d "$EXAMPLES_DIR" ]]; then
    echo "ERROR: Examples directory not found: $EXAMPLES_DIR"
    exit 1
fi

# Count examples
TOTAL_EXAMPLES=$(find "$EXAMPLES_DIR" -name "*.ail" | wc -l | tr -d ' ')
RUNNABLE_EXAMPLES=$(find "$RUNNABLE_DIR" -name "*.ail" 2>/dev/null | wc -l | tr -d ' ')

echo "Examples Summary:"
echo "  Total .ail files: $TOTAL_EXAMPLES"
echo "  Runnable examples: $RUNNABLE_EXAMPLES"
echo ""

# Check which examples are used in website
echo "=== Examples Used in Website ==="
DOCS_DIR="$PROJECT_ROOT/docs/docs"

# Find raw-loader imports
echo "Files using raw-loader imports:"
grep -r "raw-loader.*examples" "$DOCS_DIR" 2>/dev/null | while read -r line; do
    file=$(echo "$line" | cut -d: -f1)
    example=$(echo "$line" | grep -o "examples/[^'\"]*" | head -1)
    echo "  $file → $example"
done
echo ""

# Check if runnable examples actually run
echo "=== Testing Runnable Examples ==="
if [[ "$QUICK_MODE" == "true" ]]; then
    echo "(Quick mode - testing first 5 only)"
    files=$(find "$RUNNABLE_DIR" -name "*.ail" 2>/dev/null | head -5)
else
    files=$(find "$RUNNABLE_DIR" -name "*.ail" 2>/dev/null)
fi

PASSED=0
FAILED=0
FAILED_FILES=""

for file in $files; do
    basename=$(basename "$file")

    # Try to compile/check the file
    if "$PROJECT_ROOT/bin/ailang" run --entry main --caps IO "$file" >/dev/null 2>&1; then
        echo "  [OK] $basename"
        ((PASSED++))
    elif "$PROJECT_ROOT/bin/ailang" run --entry main "$file" >/dev/null 2>&1; then
        echo "  [OK] $basename (no caps)"
        ((PASSED++))
    else
        # Try without entry point (expression mode)
        if head -1 "$file" | grep -q "^module"; then
            echo "  [FAIL] $basename"
            ((FAILED++))
            FAILED_FILES="$FAILED_FILES $basename"
        else
            echo "  [SKIP] $basename (no module declaration)"
        fi
    fi
done

echo ""
echo "=== Summary ==="
echo "Passed: $PASSED"
echo "Failed: $FAILED"
if [[ -n "$FAILED_FILES" ]]; then
    echo "Failed files:$FAILED_FILES"
fi

# Check for embedded code blocks (anti-pattern)
echo ""
echo "=== Anti-Pattern Check ==="
echo "Checking for embedded code blocks (should use raw-loader instead):"

EMBEDDED_COUNT=$(grep -r '```ailang' "$DOCS_DIR" 2>/dev/null | wc -l | tr -d ' ')
EMBEDDED_COUNT2=$(grep -r '```typescript' "$DOCS_DIR" 2>/dev/null | grep -v "CodeBlock" | wc -l | tr -d ' ')

if [[ "$EMBEDDED_COUNT" -gt 0 ]]; then
    echo "  WARNING: Found $EMBEDDED_COUNT \`\`\`ailang code blocks"
    echo "  These may drift out of sync - consider using raw-loader"
fi
if [[ "$EMBEDDED_COUNT2" -gt 5 ]]; then
    echo "  INFO: Found $EMBEDDED_COUNT2 \`\`\`typescript code blocks"
    echo "  Some may be examples that should use raw-loader"
fi

echo ""
echo "=== Example Check Complete ==="
