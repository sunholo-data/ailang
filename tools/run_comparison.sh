#!/usr/bin/env bash
#
# run_comparison.sh - Run AILANG vs Python comparison for marketing table
#

set -euo pipefail

OUTPUT_DIR="${1:-eval_results/marketing_comparison}"
MODEL="${2:-claude-sonnet-4-5}"

echo "Running AILANG vs Python comparison..."
echo "  Model: $MODEL"
echo "  Output: $OUTPUT_DIR"
echo ""

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

BENCHMARKS="adt_option cli_args fizzbuzz float_eq json_parse numeric_modulo pipeline records_person recursion_factorial recursion_fibonacci"

COUNT=0
TOTAL=20  # 10 benchmarks × 2 languages

for BENCH in $BENCHMARKS; do
  for LANG in python ailang; do
    COUNT=$((COUNT + 1))
    printf "[%2d/%2d] %-20s %-6s ... " "$COUNT" "$TOTAL" "$BENCH" "$LANG"

    if bin/ailang eval --benchmark "$BENCH" --model "$MODEL" --langs "$LANG" --self-repair --output "$OUTPUT_DIR" > /dev/null 2>&1; then
      echo "✓"
    else
      echo "✗"
    fi
  done
done

echo ""
echo "✓ Comparison complete: $OUTPUT_DIR"
