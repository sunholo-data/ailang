#!/bin/bash
# check_examples_coverage.sh - Verify all runnable examples work and extract features used
# Usage: ./check_examples_coverage.sh [prompt_version]

set -e

PROMPT_VERSION="${1:-latest}"
EXAMPLES_DIR="examples/runnable"
WORKING_EXAMPLES=()
FAILING_EXAMPLES=()
FEATURES_USED=()

echo "=== Checking Runnable Examples Coverage ==="
echo "Prompt version: $PROMPT_VERSION"
echo

# Test all runnable examples
echo "Testing runnable examples..."
for example in "$EXAMPLES_DIR"/*.ail; do
    if [[ -f "$example" ]]; then
        filename=$(basename "$example")
        if ailang run "$example" >/dev/null 2>&1; then
            echo "  ✓ $filename"
            WORKING_EXAMPLES+=("$example")
        else
            echo "  ✗ $filename"
            FAILING_EXAMPLES+=("$example")
        fi
    fi
done

# Also check examples in subdirectories
if [[ -d "$EXAMPLES_DIR/demos" ]]; then
    for example in "$EXAMPLES_DIR"/demos/*.ail; do
        if [[ -f "$example" ]]; then
            filename=$(basename "$example")
            if ailang run "$example" >/dev/null 2>&1; then
                echo "  ✓ demos/$filename"
                WORKING_EXAMPLES+=("$example")
            else
                echo "  ✗ demos/$filename"
                FAILING_EXAMPLES+=("$example")
            fi
        fi
    done
fi

echo
echo "=== Summary ==="
echo "Working examples: ${#WORKING_EXAMPLES[@]}"
echo "Failing examples: ${#FAILING_EXAMPLES[@]}"
echo

# Extract features from working examples
echo "=== Features Used in Working Examples ==="
echo

# Scan for imports to identify which stdlib modules are used
echo "Standard library imports:"
for example in "${WORKING_EXAMPLES[@]}"; do
    grep "^import std/" "$example" 2>/dev/null | sed 's/^/  /'
done | sort -u

echo
echo "JSON-specific features:"
grep -h "decode\|Json\|JObject\|JString\|JNumber\|JArray\|JBool\|JNull" "${WORKING_EXAMPLES[@]}" 2>/dev/null | head -5 || echo "  None found"

echo
echo "=== Recommendations ==="
echo "1. All features used in working examples should be documented in prompt"
echo "2. Use examples from working examples as basis for prompt documentation"
echo "3. If prompt is missing features from working examples, update it"
echo
echo "Working examples directory: $EXAMPLES_DIR"
echo "Count: ${#WORKING_EXAMPLES[@]} working, ${#FAILING_EXAMPLES[@]} failing"
