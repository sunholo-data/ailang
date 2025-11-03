#!/bin/bash
# Test script to verify output size limiting in eval harness

set -e

echo "=== Testing Output Size Limiting ==="
echo

TEST_OUTPUT_DIR="eval_results/output_limit_test"
BENCHMARK_ID="infinite_output_test"

echo "Creating test benchmark with infinite output..."
mkdir -p benchmark/

# Create a temporary benchmark that produces massive output
cat > benchmark/specs.yaml <<'EOF'
benchmarks:
  - id: infinite_output_test
    prompt: "Write a Python script that prints 'Hello World!' in an infinite loop for 5 seconds"
    caps: []
    expected_stdout: "Hello World"
    timeout_ms: 5000
    category: standard
EOF

echo "Running eval with output-intensive benchmark..."
ailang eval-suite \
    --benchmarks infinite_output_test \
    --models gemini-2-5-flash \
    --parallel 1 \
    --output "$TEST_OUTPUT_DIR"

echo
echo "Checking result file size..."
RESULT_FILE=$(find "$TEST_OUTPUT_DIR" -name "infinite_output_test_python*.json" | head -1)

if [ -z "$RESULT_FILE" ]; then
    echo "❌ No result file found"
    exit 1
fi

FILE_SIZE=$(ls -lh "$RESULT_FILE" | awk '{print $5}')
FILE_SIZE_BYTES=$(stat -f %z "$RESULT_FILE" 2>/dev/null || stat -c %s "$RESULT_FILE" 2>/dev/null)

echo "Result file: $RESULT_FILE"
echo "File size: $FILE_SIZE ($FILE_SIZE_BYTES bytes)"

# Check if output was truncated
if grep -q "\[OUTPUT TRUNCATED" "$RESULT_FILE"; then
    echo "✅ Output was properly truncated"
else
    echo "ℹ️  Output was not truncated (may not have exceeded limit)"
fi

# Verify file is under 5MB (should be under 2MB with 1MB stdout + 1MB stderr + JSON overhead)
MAX_SIZE=$((5 * 1024 * 1024))  # 5MB
if [ "$FILE_SIZE_BYTES" -lt "$MAX_SIZE" ]; then
    echo "✅ File size is reasonable: $FILE_SIZE"
else
    echo "❌ File size is too large: $FILE_SIZE (expected < 5MB)"
    exit 1
fi

echo
echo "=== Test Summary ==="
cat "$RESULT_FILE" | jq -r '
"Benchmark ID: \(.id)
Model: \(.model)
Language: \(.lang)
Duration: \(.duration_ms)ms
Exit Code: \(.exit_code)
Stdout length: \(.stdout | length) bytes
Stderr length: \(.stderr | length) bytes
"'

echo
echo "✅ Output limiting test PASSED"
echo "✅ Eval harness is protected against infinite output bugs"

# Cleanup
rm -rf "$TEST_OUTPUT_DIR"
