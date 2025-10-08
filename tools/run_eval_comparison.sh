#!/usr/bin/env bash
# Run eval comparison across 3 models
set -euo pipefail

MODELS=("claude-sonnet-4-5" "gpt-4o-mini" "gemini-2.0-flash-exp")
OUTPUT_DIR="eval_results/comparison_$(date +%Y-%m-%d)"
mkdir -p "$OUTPUT_DIR"

echo "Running eval comparison across ${#MODELS[@]} models..."
echo "Output directory: $OUTPUT_DIR"
echo ""

# Get all benchmarks
BENCHMARKS=$(ls benchmarks/*.yml | xargs -n1 basename | sed 's/\.yml$//')

for MODEL in "${MODELS[@]}"; do
    echo "======================================"
    echo "Model: $MODEL"
    echo "======================================"

    for BENCH in $BENCHMARKS; do
        echo "  Running: $BENCH"
        bin/ailang eval --benchmark "$BENCH" --model "$MODEL" --langs ailang --self-repair --output "$OUTPUT_DIR" 2>&1 | grep -E "(✓|✗)" || true
    done

    echo ""
done

echo "✓ Eval comparison complete"
echo "Results in: $OUTPUT_DIR"
echo ""
echo "Summary:"
ls -1 "$OUTPUT_DIR"/*ailang*.json | wc -l | xargs echo "Total results:"
