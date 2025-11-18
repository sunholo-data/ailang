#!/usr/bin/env bash
# Run a small test benchmark to verify model works end-to-end
# Usage: run_test_benchmark.sh <model-name> [benchmark]

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <model-name> [benchmark]"
    echo ""
    echo "Arguments:"
    echo "  model-name : Friendly model name from models.yml (e.g., gpt5-1)"
    echo "  benchmark  : Optional benchmark name (default: fizzbuzz)"
    echo ""
    echo "Examples:"
    echo "  $0 gpt5-1"
    echo "  $0 gemini-3-pro recursion_factorial"
    exit 1
fi

MODEL="$1"
BENCHMARK="${2:-fizzbuzz}"

echo "Running test benchmark: $BENCHMARK"
echo "Model: $MODEL"
echo ""

# Check if ailang is available
if ! command -v ailang &> /dev/null; then
    echo "✗ ailang command not found"
    echo "Run: make install"
    exit 1
fi

# Create temp directory for results
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "✓ Running benchmark (output: $TEMP_DIR)..."
echo ""

# Run eval suite with single benchmark and model
if ailang eval-suite \
    --models "$MODEL" \
    --benchmarks "$BENCHMARK" \
    --output "$TEMP_DIR" \
    --timeout 30s; then

    echo ""
    echo "✓ Benchmark completed"

    # Check results
    RESULT_FILE=$(find "$TEMP_DIR" -name "${BENCHMARK}_ailang_${MODEL}_*.json" | head -1)

    if [ -z "$RESULT_FILE" ]; then
        echo "⚠ No result file found"
        echo "Check $TEMP_DIR for details"
        exit 1
    fi

    # Parse result
    RESULT=$(python3 -c "import sys, json; r = json.load(open('$RESULT_FILE')); print(r['result'])")
    INPUT_TOKENS=$(python3 -c "import sys, json; r = json.load(open('$RESULT_FILE')); print(r.get('input_tokens', 0))")
    OUTPUT_TOKENS=$(python3 -c "import sys, json; r = json.load(open('$RESULT_FILE')); print(r.get('output_tokens', 0))")
    COST=$(python3 -c "import sys, json; r = json.load(open('$RESULT_FILE')); print(r.get('cost', 0))")

    if [ "$RESULT" = "PASS" ]; then
        echo "✓ Result: PASS"
    else
        echo "✗ Result: $RESULT"
        echo ""
        echo "Generated code:"
        python3 -c "import sys, json; r = json.load(open('$RESULT_FILE')); print(r.get('generated_code', 'N/A'))"
        echo ""
        echo "Error:"
        python3 -c "import sys, json; r = json.load(open('$RESULT_FILE')); print(r.get('error', 'N/A'))"
        exit 1
    fi

    echo "✓ Tokens: $INPUT_TOKENS input, $OUTPUT_TOKENS output"
    echo "✓ Cost: \$$COST"
    echo ""
    echo "✓ Model is ready for production use"

else
    echo "✗ Benchmark failed"
    echo "Check logs for details"
    exit 1
fi
