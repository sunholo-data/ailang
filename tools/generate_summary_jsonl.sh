#!/usr/bin/env bash
#
# generate_summary_jsonl.sh - Convert eval results to JSONL format for AI analysis
#
# Usage:
#   ./tools/generate_summary_jsonl.sh INPUT_DIR [OUTPUT_FILE]
#
# Example:
#   ./tools/generate_summary_jsonl.sh eval_results/baseline results/summary.jsonl
#
# Output format: One JSON object per line with key metrics for easy AI analysis

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

if [ $# -lt 1 ]; then
  echo -e "${RED}Error: Missing input directory${NC}"
  echo "Usage: $0 INPUT_DIR [OUTPUT_FILE]"
  echo ""
  echo "Example:"
  echo "  $0 eval_results/baseline results/summary.jsonl"
  exit 1
fi

INPUT_DIR="$1"
OUTPUT_FILE="${2:-summary.jsonl}"

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

echo -e "${CYAN}Generating JSONL summary from $INPUT_DIR${NC}"
echo ""

# Create output directory if needed
OUTPUT_DIR=$(dirname "$OUTPUT_FILE")
mkdir -p "$OUTPUT_DIR"

# Initialize output file
> "$OUTPUT_FILE"

# Process each JSON file
TOTAL_FILES=0
for file in "$INPUT_DIR"/*.json; do
  if [ -f "$file" ]; then
    TOTAL_FILES=$((TOTAL_FILES + 1))

    # Extract key fields and write as single-line JSON
    jq -c '{
      id: .id,
      lang: .lang,
      model: .model,
      seed: .seed,
      prompt_version: .prompt_version,
      first_attempt_ok: .first_attempt_ok,
      repair_used: .repair_used,
      repair_ok: .repair_ok,
      err_code: .err_code,
      compile_ok: .compile_ok,
      runtime_ok: .runtime_ok,
      stdout_ok: .stdout_ok,
      error_category: .error_category,
      input_tokens: .input_tokens,
      output_tokens: .output_tokens,
      total_tokens: .total_tokens,
      cost_usd: .cost_usd,
      duration_ms: .duration_ms,
      timestamp: .timestamp,
      stderr: .stderr
    }' "$file" >> "$OUTPUT_FILE"
  fi
done

# Count lines in output
LINE_COUNT=$(wc -l < "$OUTPUT_FILE" | tr -d ' ')

echo -e "${GREEN}✓ Generated JSONL summary${NC}"
echo ""
echo "  Input:  $INPUT_DIR ($TOTAL_FILES JSON files)"
echo "  Output: $OUTPUT_FILE ($LINE_COUNT lines)"
echo ""
echo "Example queries:"
echo ""
echo "  # Count successes"
echo "  jq -s 'map(select(.stdout_ok == true)) | length' $OUTPUT_FILE"
echo ""
echo "  # Average tokens by model"
echo "  jq -s 'group_by(.model) | map({model: .[0].model, avg_tokens: (map(.total_tokens) | add / length)})' $OUTPUT_FILE"
echo ""
echo "  # Error distribution"
echo "  jq -s 'group_by(.err_code) | map({code: .[0].err_code, count: length})' $OUTPUT_FILE"
echo ""
echo "  # Repair effectiveness"
echo "  jq -s 'map(select(.repair_used == true)) | {total: length, success: map(select(.repair_ok == true)) | length}' $OUTPUT_FILE"
echo ""

# Print sample line
echo "Sample output (first line):"
echo ""
head -1 "$OUTPUT_FILE" | jq .
echo ""
