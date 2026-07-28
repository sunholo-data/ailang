#!/usr/bin/env bash
# categorize_errors.sh - Categorize AILANG failures by error type
#
# Usage: ./categorize_errors.sh <eval_dir>
# Example: ./categorize_errors.sh eval_results/baselines/v0.6.5
#
# Output: Error categories with counts and affected benchmarks

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

echo "=== AILANG Error Categories ==="
echo ""

# Get all AILANG failures grouped by error category
echo "Error Category Breakdown:"
echo ""

jq -rs '
  map(select(.lang == "ailang" and .stdout_ok == false)) |
  group_by(.error_category) |
  map({
    category: .[0].error_category,
    count: length,
    benchmarks: (map(.benchmark) | unique | sort)
  }) |
  sort_by(-.count)
' "$SUMMARY_FILE" | jq -r '
  .[] |
  "[\(.category // "unknown")] - \(.count) failures"
'

echo ""
echo "=== Detailed Breakdown ==="
echo ""

# For each category, show the benchmarks
jq -rs '
  map(select(.lang == "ailang" and .stdout_ok == false)) |
  group_by(.error_category) |
  map({
    category: .[0].error_category,
    count: length,
    benchmarks: (map(.benchmark) | unique | sort)
  }) |
  sort_by(-.count) |
  .[] |
  "### \(.category // "unknown") (\(.count) failures)\n\(.benchmarks | map("  - " + .) | join("\n"))\n"
' "$SUMMARY_FILE"

echo ""
echo "=== Fix Recommendations ==="
echo ""

# Provide recommendations based on error categories
jq -rs '
  map(select(.lang == "ailang" and .stdout_ok == false)) |
  group_by(.error_category) |
  map({category: .[0].error_category, count: length}) |
  sort_by(-.count)
' "$SUMMARY_FILE" | jq -r '
  .[] |
  if .category == "WRONG_LANG" then
    "WRONG_LANG (\(.count)): Model wrote Python instead of AILANG\n  Fix: Strengthen \"NOT Python\" emphasis in prompt\n"
  elif .category == "PAR_001" or .category == "compile_error" then
    "\(.category) (\(.count)): Parser/syntax error\n  Fix: Add more syntax examples to prompt\n"
  elif .category == "type_error" then
    "type_error (\(.count)): Type unification failure\n  Fix: Check for language gaps, add type examples to prompt\n"
  elif .category == "logic_error" then
    "logic_error (\(.count)): Compiles but wrong output\n  Fix: Better algorithm examples in prompt\n"
  elif .category == "EOF" or .category == "eof_error" then
    "EOF (\(.count)): Incomplete code generation\n  Fix: Model limitation, may not be fixable via prompt\n"
  else
    "\(.category // "unknown") (\(.count)): Unknown error type\n  Fix: Manual investigation needed\n"
  end
'

echo ""
echo "=== Raw Error Messages (Sample) ==="
echo ""
echo "Sample stderr from failures (first 5):"
echo ""

# Show sample error messages
jq -rs '
  map(select(.lang == "ailang" and .stdout_ok == false)) |
  .[0:5] |
  .[] |
  "--- \(.benchmark) (\(.model)) ---\n\(.stderr // "no stderr")[0:500]\n"
' "$SUMMARY_FILE" 2>/dev/null || echo "(No detailed error messages available)"
