#!/usr/bin/env bash
# Run a quick single-model test of a benchmark
# Usage: test_benchmark.sh <benchmark_id> [model]

set -euo pipefail

BENCHMARK_ID="${1:-}"
MODEL="${2:-claude-haiku-4-5}"

if [[ -z "$BENCHMARK_ID" ]]; then
    echo "Usage: test_benchmark.sh <benchmark_id> [model]"
    echo ""
    echo "Arguments:"
    echo "  benchmark_id  The benchmark ID (e.g., json_parse)"
    echo "  model         Model to test with (default: claude-haiku-4-5)"
    echo ""
    echo "Available models (from models.yml):"
    echo "  - claude-haiku-4-5 (cheap, fast - good for debugging)"
    echo "  - claude-sonnet-4-5"
    echo "  - gpt5-mini (cheap)"
    echo "  - gpt5"
    echo "  - gemini-2-5-flash (cheap)"
    echo "  - gemini-2-5-pro"
    exit 1
fi

BENCHMARK_FILE="benchmarks/${BENCHMARK_ID}.yml"

if [[ ! -f "$BENCHMARK_FILE" ]]; then
    echo "Error: Benchmark file not found: $BENCHMARK_FILE"
    exit 1
fi

echo "=== Testing Benchmark: $BENCHMARK_ID ==="
echo "Model: $MODEL"
echo ""

# Run the eval
ailang eval-suite --models "$MODEL" --benchmarks "$BENCHMARK_ID"

echo ""
echo "=== Results ==="

# Find the most recent result file
RESULT_DIR="eval_results"
if [[ -d "$RESULT_DIR" ]]; then
    RESULT_FILE=$(find "$RESULT_DIR" -name "${BENCHMARK_ID}_ailang_${MODEL}*.json" -type f -mmin -5 | head -1)

    if [[ -n "$RESULT_FILE" && -f "$RESULT_FILE" ]]; then
        echo "Result file: $RESULT_FILE"
        echo ""

        # Extract key fields
        PASSED=$(jq -r '.passed // false' "$RESULT_FILE")
        ERROR_TYPE=$(jq -r '.error_type // "none"' "$RESULT_FILE")

        if [[ "$PASSED" == "true" ]]; then
            echo "Status: PASSED"
        else
            echo "Status: FAILED"
            echo "Error type: $ERROR_TYPE"
            echo ""
            echo "=== Generated Code ==="
            jq -r '.generated_code // "N/A"' "$RESULT_FILE"
            echo ""
            echo "=== Actual Output ==="
            jq -r '.actual_stdout // "N/A"' "$RESULT_FILE"
            echo ""
            echo "=== Compiler/Runtime Error ==="
            jq -r '.error_message // "N/A"' "$RESULT_FILE"
        fi
    else
        echo "Could not find recent result file"
    fi
else
    echo "Results directory not found: $RESULT_DIR"
fi
