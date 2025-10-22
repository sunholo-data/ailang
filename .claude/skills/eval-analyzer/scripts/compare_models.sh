#!/usr/bin/env bash
# Compare model performance on AILANG benchmarks
#
# Usage:
#   compare_models.sh <baseline_dir>
#
# Example:
#   compare_models.sh eval_results/baselines/v0.3.16

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <baseline_dir>" >&2
    exit 1
fi

BASELINE_DIR="$1"
SUMMARY="$BASELINE_DIR/summary.jsonl"

if [ ! -f "$SUMMARY" ]; then
    echo "→ Generating summary.jsonl..." >&2
    bin/ailang eval-summary "$BASELINE_DIR" >&2
fi

echo "=== Model Comparison for AILANG in $(basename $BASELINE_DIR) ==="
echo

# Overall model performance
echo "Model Success Rates:"
jq -s '
  map(select(.lang == "ailang")) | 
  group_by(.model) | 
  map({
    model: .[0].model,
    success_rate: (map(select(.stdout_ok)) | length) / length * 100 | round,
    total: length,
    success: map(select(.stdout_ok)) | length,
    failed: map(select(.stdout_ok == false)) | length
  }) | 
  sort_by(-.success_rate)
' "$SUMMARY"
echo

# First attempt vs repaired
echo "First Attempt vs Final Success:"
jq -s '
  map(select(.lang == "ailang")) | 
  group_by(.model) | 
  map({
    model: .[0].model,
    first_attempt: (map(select(.first_attempt_ok)) | length) / length * 100 | round,
    final_success: (map(select(.stdout_ok)) | length) / length * 100 | round,
    repair_helped: ((map(select(.stdout_ok)) | length) - (map(select(.first_attempt_ok)) | length))
  }) | 
  sort_by(-.final_success)
' "$SUMMARY"
echo

# Cost analysis
echo "Cost per Model:"
jq -s '
  map(select(.lang == "ailang")) | 
  group_by(.model) | 
  map({
    model: .[0].model,
    total_cost: (map(.cost_usd) | add * 100 | round / 100),
    avg_cost_per_run: (map(.cost_usd) | add / length * 1000 | round / 1000),
    cost_per_success: (
      if (map(select(.stdout_ok)) | length) > 0 
      then ((map(.cost_usd) | add) / (map(select(.stdout_ok)) | length) * 1000 | round / 1000)
      else null
      end
    )
  }) | 
  sort_by(-.total_cost)
' "$SUMMARY"
echo

# Token usage
echo "Token Usage:"
jq -s '
  map(select(.lang == "ailang")) | 
  group_by(.model) | 
  map({
    model: .[0].model,
    avg_tokens: (map(.total_tokens) | add / length | round),
    avg_input: (map(.input_tokens) | add / length | round),
    avg_output: (map(.output_tokens) | add / length | round)
  })
' "$SUMMARY"
echo

# Best model for each benchmark
echo "Best Model per Benchmark (top 10 by variance):"
jq -s '
  map(select(.lang == "ailang")) | 
  group_by(.id) | 
  map({
    benchmark: .[0].id,
    models: group_by(.model) | map({
      model: .[0].model,
      success: map(select(.stdout_ok)) | length > 0
    }),
    variance: (
      group_by(.model) | 
      map(map(select(.stdout_ok)) | length) | 
      (max - min)
    )
  }) |
  sort_by(-.variance) |
  .[:10]
' "$SUMMARY"

