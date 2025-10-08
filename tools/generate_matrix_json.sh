#!/usr/bin/env bash
#
# generate_matrix_json.sh - Generate performance matrix JSON for historical tracking
#
# Usage:
#   ./tools/generate_matrix_json.sh INPUT_DIR VERSION [OUTPUT_FILE]
#
# Example:
#   ./tools/generate_matrix_json.sh eval_results/baseline v0.3.0-alpha5 performance_tables/v0.3.0-alpha5.json
#
# Output: Structured JSON with aggregates by model, benchmark, error code, etc.

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

if [ $# -lt 2 ]; then
  echo -e "${RED}Error: Missing arguments${NC}"
  echo "Usage: $0 INPUT_DIR VERSION [OUTPUT_FILE]"
  echo ""
  echo "Example:"
  echo "  $0 eval_results/baseline v0.3.0-alpha5 performance_tables/v0.3.0-alpha5.json"
  exit 1
fi

INPUT_DIR="$1"
VERSION="$2"
OUTPUT_FILE="${3:-eval_results/performance_tables/${VERSION}.json}"

# Verify input directory exists
if [ ! -d "$INPUT_DIR" ]; then
  echo -e "${RED}Error: Directory not found: $INPUT_DIR${NC}"
  exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
  echo -e "${RED}Error: jq is required but not installed${NC}"
  echo "Install with: brew install jq"
  exit 1
fi

echo -e "${CYAN}Generating performance matrix for ${BOLD}${VERSION}${NC}${CYAN} from $INPUT_DIR${NC}"
echo ""

# Create output directory
OUTPUT_DIR=$(dirname "$OUTPUT_FILE")
mkdir -p "$OUTPUT_DIR"

# Generate JSONL summary first (temporary)
TEMP_JSONL=$(mktemp)
trap "rm -f $TEMP_JSONL" EXIT

for file in "$INPUT_DIR"/*.json; do
  if [ -f "$file" ]; then
    jq -c . "$file" >> "$TEMP_JSONL"
  fi
done

# Calculate aggregates
MATRIX=$(jq -s '
if length == 0 then
  {
    version: "'$VERSION'",
    timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
    total_runs: 0,
    error: "No results found",
    aggregates: {},
    models: {},
    benchmarks: {},
    error_codes: []
  }
else
  {
    version: "'$VERSION'",
    timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
    total_runs: length,

    # Overall aggregates
    aggregates: {
      "0-shot_success": (map(select(.first_attempt_ok == true)) | length / length),
      "final_success": (map(select(.stdout_ok == true)) | length / length),
      repair_used: (map(select(.repair_used == true)) | length),
      repair_success_rate: (
        (map(select(.repair_used == true and .repair_ok == true)) | length) /
        ((map(select(.repair_used == true)) | length) + 0.0001)  # Avoid div by zero
      ),
      total_tokens: (map(.total_tokens) | add),
      total_cost_usd: (map(.cost_usd) | add),
      avg_duration_ms: (map(.duration_ms) | add / length)
    },

  # By model
  models: (
    group_by(.model) | map({
      key: .[0].model,
      value: {
        total_runs: length,
        aggregates: {
          "0-shot_success": (map(select(.first_attempt_ok == true)) | length / length),
          "final_success": (map(select(.stdout_ok == true)) | length / length),
          repair_used: (map(select(.repair_used == true)) | length),
          repair_success_rate: (
            (map(select(.repair_used == true and .repair_ok == true)) | length) /
            ((map(select(.repair_used == true)) | length) + 0.0001)
          ),
          avg_tokens: (map(.total_tokens) | add / length),
          avg_cost_usd: (map(.cost_usd) | add / length)
        },
        benchmarks: (
          group_by(.id) | map({
            key: .[0].id,
            value: {
              success: (map(select(.stdout_ok == true)) | length > 0),
              first_attempt_ok: (.[0].first_attempt_ok),
              repair_used: (.[0].repair_used // false),
              tokens: (.[0].total_tokens)
            }
          }) | from_entries
        )
      }
    }) | from_entries
  ),

  # By benchmark
  benchmarks: (
    group_by(.id) | map({
      key: .[0].id,
      value: {
        total_runs: length,
        success_rate: (map(select(.stdout_ok == true)) | length / length),
        avg_tokens: (map(.total_tokens) | add / length),
        languages: (map(.lang) | unique)
      }
    }) | from_entries
  ),

  # By error code (failures only)
  error_codes: (
    map(select(.err_code != null and .err_code != "")) |
    group_by(.err_code) | map({
      code: .[0].err_code,
      count: length,
      repair_success: (map(select(.repair_ok == true)) | length / length)
    })
  ),

  # By language
  languages: (
    group_by(.lang) | map({
      key: .[0].lang,
      value: {
        total_runs: length,
        success_rate: (map(select(.stdout_ok == true)) | length / length),
        avg_tokens: (map(.total_tokens) | add / length)
      }
    }) | from_entries
  ),

  # By prompt version (if available)
  prompt_versions: (
    map(select(.prompt_version != null and .prompt_version != "")) |
    group_by(.prompt_version) | map({
      key: .[0].prompt_version,
      value: {
        total_runs: length,
        "0-shot_success": (map(select(.first_attempt_ok == true)) | length / length),
        "final_success": (map(select(.stdout_ok == true)) | length / length),
        avg_tokens: (map(.total_tokens) | add / length)
      }
    }) | from_entries
  )
}
end
' < "$TEMP_JSONL")

# Write output
echo "$MATRIX" | jq '.' > "$OUTPUT_FILE"

# Extract summary stats
TOTAL_RUNS=$(echo "$MATRIX" | jq -r '.total_runs')
ZERO_SHOT=$(echo "$MATRIX" | jq -r '.aggregates."0-shot_success" * 100 | round')
FINAL_SUCCESS=$(echo "$MATRIX" | jq -r '.aggregates."final_success" * 100 | round')
TOTAL_COST=$(echo "$MATRIX" | jq -r '.aggregates.total_cost_usd')

echo -e "${GREEN}✓ Performance matrix generated${NC}"
echo ""
echo "  Version:       $VERSION"
echo "  Total runs:    $TOTAL_RUNS"
echo "  0-shot:        ${ZERO_SHOT}%"
echo "  Final success: ${FINAL_SUCCESS}%"
echo "  Total cost:    \$${TOTAL_COST}"
echo ""
echo "  Output: $OUTPUT_FILE"
echo ""

echo "Example queries:"
echo ""
echo "  # Show model comparison"
echo "  jq '.models | to_entries | map({model: .key, success: (.value.aggregates.\"final_success\" * 100 | round)})' $OUTPUT_FILE"
echo ""
echo "  # Show error distribution"
echo "  jq '.error_codes | map(\"{\\(.code)}: \\(.count) (repair: \\(.repair_success * 100 | round)%)\")' $OUTPUT_FILE"
echo ""
echo "  # Compare prompt versions"
echo "  jq '.prompt_versions' $OUTPUT_FILE"
echo ""
