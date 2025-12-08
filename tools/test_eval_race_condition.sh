#!/bin/bash
# Test script to detect race conditions in eval harness
# This runs benchmarks with high parallelism to trigger potential race conditions

set -e

echo "=== Eval Harness Race Condition Test ==="
echo

# Configuration
TEST_OUTPUT_DIR="eval_results/race_condition_test"
MODELS="gemini-2-5-flash"
BENCHMARKS="recursion_fibonacci,referential_transparency,simple_print,exhaustive_pattern_matching"
PARALLELISM=20

echo "Configuration:"
echo "  Models: $MODELS"
echo "  Benchmarks: $BENCHMARKS"
echo "  Parallelism: $PARALLELISM"
echo "  Output: $TEST_OUTPUT_DIR"
echo

# Clean previous test results
if [ -d "$TEST_OUTPUT_DIR" ]; then
    echo "Cleaning previous test results..."
    rm -rf "$TEST_OUTPUT_DIR"
fi

# Run eval suite with high parallelism (should trigger bug if it exists)
echo "Running eval suite with high parallelism ($PARALLELISM concurrent jobs)..."
echo "This should complete in ~10-15 seconds if no issues..."
echo

ailang eval-suite \
    --benchmarks "$BENCHMARKS" \
    --models "$MODELS" \
    --parallel "$PARALLELISM" \
    --output "$TEST_OUTPUT_DIR"

echo
echo "Eval suite completed. Analyzing results..."
echo

# Check for suspicious output patterns
echo "Checking for output corruption..."
CORRUPTED=0

# recursion_fibonacci should output "6765", not "All results equal"
if grep -q "All results equal" "$TEST_OUTPUT_DIR"/standard/recursion_fibonacci*.json 2>/dev/null; then
    echo "❌ CORRUPTION DETECTED: recursion_fibonacci has wrong output!"
    grep -A 3 "stdout" "$TEST_OUTPUT_DIR"/standard/recursion_fibonacci*.json | head -10
    CORRUPTED=1
fi

# referential_transparency should output "All results equal: true", not numbers
if grep -q "\"6765\"" "$TEST_OUTPUT_DIR"/standard/referential_transparency*.json 2>/dev/null; then
    echo "❌ CORRUPTION DETECTED: referential_transparency has wrong output!"
    grep -A 3 "stdout" "$TEST_OUTPUT_DIR"/standard/referential_transparency*.json | head -10
    CORRUPTED=1
fi

# simple_print should not have fibonacci output
if grep -q "\"6765\"" "$TEST_OUTPUT_DIR"/standard/simple_print*.json 2>/dev/null; then
    echo "❌ CORRUPTION DETECTED: simple_print has wrong output!"
    grep -A 3 "stdout" "$TEST_OUTPUT_DIR"/standard/simple_print*.json | head -10
    CORRUPTED=1
fi

if [ $CORRUPTED -eq 0 ]; then
    echo "✅ No output corruption detected!"
    echo "✅ All benchmarks produced expected output patterns"
else
    echo
    echo "❌ TEST FAILED: Output corruption detected"
    echo "❌ This indicates a race condition in the eval harness"
    exit 1
fi

echo
echo "=== Test Summary ==="
cat "$TEST_OUTPUT_DIR"/summary.jsonl | jq -s \
    'map(select(.lang == "ailang")) |
     {total: length, passed: map(select(.stdout_ok)) | length}' | \
    jq -r '"Total runs: \(.total)\nPassed: \(.passed)\nSuccess rate: \((.passed * 100 / .total) | floor)%"'

echo
echo "✅ Race condition test PASSED"
echo "✅ Eval harness output capture is working correctly"
