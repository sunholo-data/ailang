#!/usr/bin/env bash
# Analyze failures from eval baseline results
#
# Usage:
#   analyze_failures.sh <baseline_dir> [language]
#
# Example:
#   analyze_failures.sh eval_results/baselines/v0.3.16
#   analyze_failures.sh eval_results/baselines/v0.3.16 ailang

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <baseline_dir> [language]" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 eval_results/baselines/v0.3.16" >&2
    echo "  $0 eval_results/baselines/v0.3.16 ailang" >&2
    exit 1
fi

BASELINE_DIR="$1"
LANG="${2:-ailang}"

if [ ! -d "$BASELINE_DIR" ]; then
    echo "Error: Directory not found: $BASELINE_DIR" >&2
    exit 1
fi

SUMMARY="$BASELINE_DIR/summary.jsonl"

# Generate summary if it doesn't exist
if [ ! -f "$SUMMARY" ]; then
    echo "→ Generating summary.jsonl..." >&2
    bin/ailang eval-summary "$BASELINE_DIR" >&2
fi

echo "=== Failure Analysis for $LANG in $(basename $BASELINE_DIR) ==="
echo

# Overall stats
echo "Overall Statistics:"
jq -s --arg lang "$LANG" '
  map(select(.lang == $lang)) as $filtered |
  $filtered | length as $total |
  ($filtered | map(select(.stdout_ok)) | length) as $success |
  {
    total: $total,
    success: $success,
    failed: ($total - $success),
    success_rate: (($success / $total * 100) | round)
  }
' "$SUMMARY"
echo

# Error categories
echo "Error Categories:"
jq -s --arg lang "$LANG" '
  map(select(.lang == $lang and .stdout_ok == false)) as $failures |
  $failures |
  group_by(.error_category) |
  map({
    category: .[0].error_category,
    count: length,
    percentage: ((length / ($failures | length) * 100) | floor)
  }) |
  sort_by(-.count)
' "$SUMMARY"
echo

# Top failing benchmarks
echo "Top 20 Failing Benchmarks:"
jq -s --arg lang "$LANG" '
  map(select(.lang == $lang and .stdout_ok == false)) |
  group_by(.id) |
  map({
    benchmark: .[0].id,
    failures: length,
    success_rate: ((6 - length) * 100 / 6 | floor)
  }) |
  sort_by(.success_rate, -.failures) |
  .[:20]
' "$SUMMARY"
echo

# Model performance
echo "Model Performance:"
jq -s --arg lang "$LANG" '
  map(select(.lang == $lang)) |
  group_by(.model) |
  map(
    . as $model_runs |
    ($model_runs | map(select(.stdout_ok)) | length) as $success |
    ($model_runs | length) as $total |
    {
      model: .[0].model,
      success: $success,
      failed: ($total - $success),
      total: $total,
      success_rate: (($success / $total * 100) | round)
    }
  ) |
  sort_by(-.success_rate)
' "$SUMMARY"
echo

# Error code distribution
echo "Error Codes:"
jq -s --arg lang "$LANG" '
  map(select(.lang == $lang and .err_code != null)) |
  group_by(.err_code) |
  map({
    code: .[0].err_code,
    count: length
  }) |
  sort_by(-.count)
' "$SUMMARY"
