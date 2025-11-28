#!/usr/bin/env bash
# Validate a benchmark YAML file for common issues
# Usage: check_benchmark.sh <benchmark_file>

set -euo pipefail

BENCHMARK_FILE="${1:-}"

if [[ -z "$BENCHMARK_FILE" ]]; then
    echo "Usage: check_benchmark.sh <benchmark_file>"
    echo "Example: check_benchmark.sh benchmarks/json_parse.yml"
    exit 1
fi

if [[ ! -f "$BENCHMARK_FILE" ]]; then
    echo "Error: File not found: $BENCHMARK_FILE"
    exit 1
fi

ERRORS=0
WARNINGS=0

echo "Checking: $BENCHMARK_FILE"
echo ""

# Check for required fields
REQUIRED_FIELDS=("id" "description" "languages" "entrypoint" "caps" "expected_stdout")

for field in "${REQUIRED_FIELDS[@]}"; do
    if ! grep -q "^${field}:" "$BENCHMARK_FILE"; then
        echo "ERROR: Missing required field: $field"
        ((ERRORS++))
    fi
done

# Check for prompt vs task_prompt
if grep -q "^prompt:" "$BENCHMARK_FILE"; then
    echo "WARNING: Uses 'prompt:' instead of 'task_prompt:'"
    echo "  -> Models will NOT see the AILANG teaching prompt!"
    echo "  -> Change to 'task_prompt:' unless intentionally testing without teaching"
    ((WARNINGS++))
fi

if ! grep -q "^task_prompt:" "$BENCHMARK_FILE" && ! grep -q "^prompt:" "$BENCHMARK_FILE"; then
    echo "WARNING: No prompt or task_prompt field"
    echo "  -> Models will only see teaching prompt with no specific task"
    ((WARNINGS++))
fi

# Check for valid capabilities
if grep -q "^caps:" "$BENCHMARK_FILE"; then
    CAPS=$(grep "^caps:" "$BENCHMARK_FILE" | sed 's/caps:\s*//')
    VALID_CAPS=("IO" "FS" "Clock" "Net")

    for cap in "${VALID_CAPS[@]}"; do
        # Remove this cap from consideration
        :
    done

    # Simple check - look for common mistakes
    if echo "$CAPS" | grep -qi "io" && ! echo "$CAPS" | grep -q "IO"; then
        echo "WARNING: Capability 'io' should be 'IO' (case-sensitive)"
        ((WARNINGS++))
    fi
    if echo "$CAPS" | grep -qi "fs" && ! echo "$CAPS" | grep -q "FS"; then
        echo "WARNING: Capability 'fs' should be 'FS' (case-sensitive)"
        ((WARNINGS++))
    fi
fi

# Check YAML syntax (basic) - only if pyyaml is available
if python3 -c "import yaml" 2>/dev/null; then
    if ! python3 -c "import yaml; yaml.safe_load(open('$BENCHMARK_FILE'))" 2>/dev/null; then
        echo "ERROR: Invalid YAML syntax"
        ((ERRORS++))
    fi
else
    # Fallback: basic syntax check with Go
    if command -v yq &>/dev/null; then
        if ! yq '.' "$BENCHMARK_FILE" >/dev/null 2>&1; then
            echo "ERROR: Invalid YAML syntax"
            ((ERRORS++))
        fi
    else
        echo "INFO: Skipping YAML syntax check (pyyaml/yq not installed)"
    fi
fi

# Check for <LANG> placeholder in task_prompt
if grep -q "^task_prompt:" "$BENCHMARK_FILE"; then
    if ! grep -A 20 "^task_prompt:" "$BENCHMARK_FILE" | grep -q "<LANG>"; then
        echo "INFO: task_prompt doesn't use <LANG> placeholder"
        echo "  -> Consider using 'Write a program in <LANG> that...' for clarity"
    fi
fi

echo ""
echo "=== Summary ==="
echo "Errors: $ERRORS"
echo "Warnings: $WARNINGS"

if [[ $ERRORS -gt 0 ]]; then
    exit 1
fi

exit 0
