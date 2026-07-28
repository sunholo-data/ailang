#!/usr/bin/env bash
# Show the full prompt that models receive for a benchmark
# Usage: show_full_prompt.sh <benchmark_id>

set -euo pipefail

BENCHMARK_ID="${1:-}"

if [[ -z "$BENCHMARK_ID" ]]; then
    echo "Usage: show_full_prompt.sh <benchmark_id>"
    echo "Example: show_full_prompt.sh json_parse"
    exit 1
fi

BENCHMARK_FILE="benchmarks/${BENCHMARK_ID}.yml"

if [[ ! -f "$BENCHMARK_FILE" ]]; then
    echo "Error: Benchmark file not found: $BENCHMARK_FILE"
    exit 1
fi

echo "=== Benchmark: $BENCHMARK_ID ==="
echo ""

# Check which prompt field is used
if grep -q "^prompt:" "$BENCHMARK_FILE"; then
    echo "WARNING: This benchmark uses 'prompt:' which REPLACES the teaching prompt!"
    echo "Models will NOT see AILANG syntax. Consider changing to 'task_prompt:'."
    echo ""
    echo "=== Custom Prompt (replaces teaching prompt) ==="
    # Extract the prompt field
    sed -n '/^prompt:/,/^[a-z_]*:/{ /^prompt:/d; /^[a-z_]*:/d; p; }' "$BENCHMARK_FILE"
elif grep -q "^task_prompt:" "$BENCHMARK_FILE"; then
    echo "This benchmark uses 'task_prompt:' (correct - appends to teaching prompt)"
    echo ""
    echo "=== Teaching Prompt (from ailang prompt) ==="
    echo "[Run 'ailang prompt' to see full teaching prompt]"
    echo ""
    echo "=== Task Prompt (appended to above) ==="
    echo ""
    echo "## Task"
    echo ""
    # Extract the task_prompt field
    sed -n '/^task_prompt:/,/^[a-z_]*:/{ /^task_prompt:/d; /^[a-z_]*:/d; p; }' "$BENCHMARK_FILE"
else
    echo "WARNING: No prompt or task_prompt field found!"
    echo "This benchmark will only use the teaching prompt with no specific task."
fi

echo ""
echo "=== Expected Output ==="
if grep -q "^expected_stdout:" "$BENCHMARK_FILE"; then
    sed -n '/^expected_stdout:/,/^[a-z_]*:/{ /^expected_stdout:/d; /^[a-z_]*:/d; p; }' "$BENCHMARK_FILE"
else
    echo "[No expected_stdout defined]"
fi
