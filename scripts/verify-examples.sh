#!/bin/bash
# verify-examples.sh - Automated example verification for AILANG
# Runs all examples and outputs structured JSON results

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create results directory
RESULTS_DIR="tests/results"
mkdir -p "$RESULTS_DIR"

# Output files
RESULTS_FILE="$RESULTS_DIR/examples.jsonl"
SUMMARY_FILE="$RESULTS_DIR/summary.txt"

# Clear previous results
> "$RESULTS_FILE"
> "$SUMMARY_FILE"

# Counters
TOTAL=0
PASSED=0
FAILED=0
SKIPPED=0

echo "AILANG Example Verification"
echo "============================"
echo ""

# Function to test a single example
test_example() {
    local file=$1
    local entry=${2:-main}

    TOTAL=$((TOTAL + 1))
    local display_name=$(basename "$file")

    # Run the example
    local stdout_file=$(mktemp)
    local stderr_file=$(mktemp)
    local exit_code=0

    if timeout 5s ailang --entry "$entry" run "$file" > "$stdout_file" 2> "$stderr_file"; then
        exit_code=0
    else
        exit_code=$?
    fi

    local stdout=$(cat "$stdout_file")
    local stderr=$(cat "$stderr_file")
    rm -f "$stdout_file" "$stderr_file"

    # Determine status
    local status="unknown"
    local reason=""

    if [ $exit_code -eq 0 ]; then
        if echo "$stderr" | grep -q "Error:"; then
            status="fail"
            reason=$(echo "$stderr" | grep "Error:" | head -1)
            FAILED=$((FAILED + 1))
        else
            status="pass"
            PASSED=$((PASSED + 1))
        fi
    else
        status="fail"
        reason=$(echo "$stderr" | grep "Error:" | head -1 || echo "Exit code: $exit_code")
        FAILED=$((FAILED + 1))
    fi

    # Write JSON result
    stdout_json=$(echo "$stdout" | jq -Rs . 2>/dev/null || echo '""')
    stderr_json=$(echo "$stderr" | jq -Rs . 2>/dev/null || echo '""')
    reason_json=$(echo "$reason" | jq -Rs . 2>/dev/null || echo '""')

    echo "{\"file\":\"$file\",\"entry\":\"$entry\",\"status\":\"$status\",\"exit_code\":$exit_code,\"stdout\":$stdout_json,\"stderr\":$stderr_json,\"reason\":$reason_json}" >> "$RESULTS_FILE"

    # Print status
    if [ "$status" = "pass" ]; then
        echo -e "${GREEN}✓${NC} $display_name"
    else
        echo -e "${RED}✗${NC} $display_name"
        [ -n "$reason" ] && echo "  └─ ${reason:0:80}"
    fi
}

# Test examples
for file in examples/*.ail examples/v3_3/*.ail examples/showcase/*.ail examples/demos/*.ail; do
    [ -f "$file" ] || continue
    
    # Skip if marked as broken
    if head -5 "$file" 2>/dev/null | grep -q "⚠️.*cannot execute"; then
        SKIPPED=$((SKIPPED + 1))
        TOTAL=$((TOTAL + 1))
        echo -e "${YELLOW}⊘${NC} $(basename "$file") (skipped)"
        continue
    fi
    
    # Find entrypoint
    if grep -q "func main" "$file"; then
        test_example "$file" "main"
    else
        first_export=$(grep "export func" "$file" | head -1 | sed 's/.*func \([a-zA-Z_][a-zA-Z0-9_]*\).*/\1/' || echo "")
        if [ -n "$first_export" ]; then
            test_example "$file" "$first_export"
        else
            SKIPPED=$((SKIPPED + 1))
            TOTAL=$((TOTAL + 1))
            echo -e "${YELLOW}⊘${NC} $(basename "$file") (no entrypoint)"
        fi
    fi
done

echo ""
echo "Summary: Total=$TOTAL Passed=$PASSED Failed=$FAILED Skipped=$SKIPPED"
[ $TOTAL -gt 0 ] && echo "Pass rate: $((PASSED * 100 / TOTAL))%"

# Write summary
{
    echo "AILANG Example Verification - $(date)"
    echo "Total=$TOTAL Passed=$PASSED Failed=$FAILED Skipped=$SKIPPED"
} > "$SUMMARY_FILE"

[ $FAILED -eq 0 ]
