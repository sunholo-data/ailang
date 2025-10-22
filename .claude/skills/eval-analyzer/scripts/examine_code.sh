#!/usr/bin/env bash
# Examine generated code from failed benchmarks
#
# Usage:
#   examine_code.sh <baseline_dir> <benchmark_id> [model]
#
# Example:
#   examine_code.sh eval_results/baselines/0.3.16 api_call_json
#   examine_code.sh eval_results/baselines/0.3.16 api_call_json gpt5

set -euo pipefail

if [ $# -lt 2 ]; then
    echo "Usage: $0 <baseline_dir> <benchmark_id> [model]" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 eval_results/baselines/0.3.16 api_call_json" >&2
    echo "  $0 eval_results/baselines/0.3.16 api_call_json gpt5" >&2
    exit 1
fi

BASELINE_DIR="$1"
BENCH_ID="$2"
MODEL="${3:-}"

if [ ! -d "$BASELINE_DIR" ]; then
    echo "Error: Directory not found: $BASELINE_DIR" >&2
    exit 1
fi

# Find matching result files
if [ -n "$MODEL" ]; then
    PATTERN="${BASELINE_DIR}/${BENCH_ID}_ailang_${MODEL}_*.json"
else
    PATTERN="${BASELINE_DIR}/${BENCH_ID}_ailang_*.json"
fi

FILES=$(find "$BASELINE_DIR" -name "${BENCH_ID}_ailang_*.json" 2>/dev/null | head -5)

if [ -z "$FILES" ]; then
    echo "No result files found for benchmark: $BENCH_ID" >&2
    exit 1
fi

for file in $FILES; do
    echo "=========================================="
    echo "File: $(basename "$file")"
    echo "Model: $(jq -r '.model' "$file")"
    echo "Success: $(jq -r '.stdout_ok' "$file")"
    echo "Error Category: $(jq -r '.error_category' "$file")"
    echo "Error Code: $(jq -r '.err_code // "N/A"' "$file")"
    echo ""
    echo "--- Generated Code ---"
    jq -r '.code // "N/A"' "$file"
    echo ""
    echo "--- Compiler Errors ---"
    jq -r '.compile_stderr // "N/A"' "$file" | head -20
    echo ""
done
