#!/usr/bin/env bash
# identify_python_only.sh - List benchmarks where Python passes but AILANG fails
#
# Usage: ./identify_python_only.sh <eval_dir>
# Example: ./identify_python_only.sh eval_results/baselines/v0.6.5
#
# Output: List of benchmark names where Python succeeds but AILANG fails

set -euo pipefail

EVAL_DIR="${1:-}"

if [[ -z "$EVAL_DIR" ]]; then
    echo "Usage: $0 <eval_dir>"
    echo "Example: $0 eval_results/baselines/v0.6.5"
    exit 1
fi

if [[ ! -d "$EVAL_DIR" ]]; then
    echo "Error: Directory not found: $EVAL_DIR"
    exit 1
fi

# Ensure summary.jsonl exists
SUMMARY_FILE="$EVAL_DIR/summary.jsonl"
if [[ ! -f "$SUMMARY_FILE" ]]; then
    echo "Generating summary.jsonl..."
    ailang eval-summary "$EVAL_DIR"
fi

echo "=== Python-Only Passing Benchmarks ==="
echo ""
echo "Benchmarks where Python passes but AILANG fails:"
echo ""

# Find benchmarks where Python passes for at least one model
# but AILANG fails for all models
jq -rs '
  # Group by benchmark
  group_by(.benchmark) |

  # For each benchmark, check if:
  # - Python passes for at least one model
  # - AILANG fails for all models
  map(
    {
      benchmark: .[0].benchmark,
      python_passes: (map(select(.lang == "python" and .stdout_ok == true)) | length > 0),
      ailang_passes: (map(select(.lang == "ailang" and .stdout_ok == true)) | length > 0),
      ailang_errors: (map(select(.lang == "ailang" and .stdout_ok == false)) | map({model: .model, error: .error_category}) | unique)
    }
  ) |

  # Filter to python-only (python passes, ailang fails for all)
  map(select(.python_passes and (.ailang_passes | not))) |

  # Sort by benchmark name
  sort_by(.benchmark)
' "$SUMMARY_FILE" | jq -r '
  .[] |
  "- \(.benchmark)"
'

echo ""
echo "=== Summary ==="

# Count total gaps
GAP_COUNT=$(jq -rs '
  group_by(.benchmark) |
  map(
    {
      python_passes: (map(select(.lang == "python" and .stdout_ok == true)) | length > 0),
      ailang_passes: (map(select(.lang == "ailang" and .stdout_ok == true)) | length > 0)
    }
  ) |
  map(select(.python_passes and (.ailang_passes | not))) |
  length
' "$SUMMARY_FILE")

TOTAL_BENCHMARKS=$(jq -rs 'map(.benchmark) | unique | length' "$SUMMARY_FILE")
AILANG_PASSING=$(jq -rs 'map(select(.lang == "ailang" and .stdout_ok == true)) | map(.benchmark) | unique | length' "$SUMMARY_FILE")
PYTHON_PASSING=$(jq -rs 'map(select(.lang == "python" and .stdout_ok == true)) | map(.benchmark) | unique | length' "$SUMMARY_FILE")

echo "Total benchmarks: $TOTAL_BENCHMARKS"
echo "Python passing (any model): $PYTHON_PASSING"
echo "AILANG passing (any model): $AILANG_PASSING"
echo "Python-only gaps: $GAP_COUNT"
echo ""
echo "Gap details (with error info):"
echo ""

jq -rs '
  group_by(.benchmark) |
  map(
    {
      benchmark: .[0].benchmark,
      python_passes: (map(select(.lang == "python" and .stdout_ok == true)) | length > 0),
      ailang_passes: (map(select(.lang == "ailang" and .stdout_ok == true)) | length > 0),
      ailang_errors: (map(select(.lang == "ailang" and .stdout_ok == false)) | map(.error_category) | unique)
    }
  ) |
  map(select(.python_passes and (.ailang_passes | not))) |
  sort_by(.benchmark) |
  .[] |
  "  \(.benchmark): \(.ailang_errors | join(", "))"
' "$SUMMARY_FILE"
