#!/usr/bin/env bash
# Examine prompts used for code generation
#
# Usage:
#   examine_prompts.sh <baseline_dir> <benchmark_id> [model]
#
# Example:
#   examine_prompts.sh eval_results/baselines/0.3.16 api_call_json
#   examine_prompts.sh eval_results/baselines/0.3.16 api_call_json gpt5

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
FILES=$(find "$BASELINE_DIR" -name "${BENCH_ID}_ailang_*.json" 2>/dev/null | head -3)

if [ -z "$FILES" ]; then
    echo "No result files found for benchmark: $BENCH_ID" >&2
    exit 1
fi

for file in $FILES; do
    echo "=========================================="
    echo "File: $(basename "$file")"
    echo "Model: $(jq -r '.model' "$file")"
    echo ""
    echo "--- System Prompt ---"
    jq -r '.system_prompt // "N/A"' "$file"
    echo ""
    echo "--- User Prompt ---"
    jq -r '.user_prompt // "N/A"' "$file"
    echo ""
    echo "--- Success: $(jq -r '.stdout_ok' "$file") ---"
    echo ""
done
