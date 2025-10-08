#!/usr/bin/env bash
#
# compare_results.sh - Compare eval results from two prompt versions
#
# Usage:
#   ./tools/compare_results.sh DIR_A DIR_B LABEL_A LABEL_B
#
# Example:
#   ./tools/compare_results.sh eval_results/baseline eval_results/hints "Baseline" "With Hints"

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

if [ $# -lt 4 ]; then
  echo -e "${RED}Error: Missing arguments${NC}"
  echo "Usage: $0 DIR_A DIR_B LABEL_A LABEL_B"
  echo ""
  echo "Example:"
  echo "  $0 eval_results/baseline eval_results/hints \"Baseline\" \"With Hints\""
  exit 1
fi

DIR_A="$1"
DIR_B="$2"
LABEL_A="$3"
LABEL_B="$4"

# Verify directories exist
if [ ! -d "$DIR_A" ]; then
  echo -e "${RED}Error: Directory not found: $DIR_A${NC}"
  exit 1
fi

if [ ! -d "$DIR_B" ]; then
  echo -e "${RED}Error: Directory not found: $DIR_B${NC}"
  exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
  echo -e "${RED}Error: jq is required but not installed${NC}"
  echo "Install with: brew install jq"
  exit 1
fi

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Prompt A/B Comparison${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${BOLD}${LABEL_A}${NC}"
echo -e "    Directory: $DIR_A"
echo ""
echo -e "  ${BOLD}${LABEL_B}${NC}"
echo -e "    Directory: $DIR_B"
echo ""

# Extract metrics from JSON files
extract_metrics() {
  local dir=$1
  local total=0
  local success_first=0
  local success_final=0
  local repair_used=0
  local repair_success=0
  local total_input_tokens=0
  local total_output_tokens=0
  local total_cost=0

  for file in "$dir"/*.json; do
    if [ -f "$file" ]; then
      total=$((total + 1))

      # Extract fields with jq
      first_ok=$(jq -r '.first_attempt_ok // .stdout_ok' "$file")
      final_ok=$(jq -r '.stdout_ok' "$file")
      repair_used_flag=$(jq -r '.repair_used // false' "$file")
      repair_ok=$(jq -r '.repair_ok // false' "$file")
      input_tokens=$(jq -r '.input_tokens // 0' "$file")
      output_tokens=$(jq -r '.output_tokens // 0' "$file")
      cost=$(jq -r '.cost_usd // 0' "$file")

      # Count successes
      if [ "$first_ok" = "true" ]; then
        success_first=$((success_first + 1))
      fi
      if [ "$final_ok" = "true" ]; then
        success_final=$((success_final + 1))
      fi
      if [ "$repair_used_flag" = "true" ]; then
        repair_used=$((repair_used + 1))
        if [ "$repair_ok" = "true" ]; then
          repair_success=$((repair_success + 1))
        fi
      fi

      # Sum tokens and cost
      total_input_tokens=$(echo "$total_input_tokens + $input_tokens" | bc)
      total_output_tokens=$(echo "$total_output_tokens + $output_tokens" | bc)
      total_cost=$(echo "$total_cost + $cost" | bc)
    fi
  done

  # Calculate rates
  if [ $total -gt 0 ]; then
    first_rate=$(echo "scale=1; $success_first * 100 / $total" | bc)
    final_rate=$(echo "scale=1; $success_final * 100 / $total" | bc)
  else
    first_rate="0.0"
    final_rate="0.0"
  fi

  if [ $repair_used -gt 0 ]; then
    repair_rate=$(echo "scale=1; $repair_success * 100 / $repair_used" | bc)
  else
    repair_rate="N/A"
  fi

  # Output results
  echo "$total"
  echo "$success_first"
  echo "$success_final"
  echo "$first_rate"
  echo "$final_rate"
  echo "$repair_used"
  echo "$repair_success"
  echo "$repair_rate"
  echo "$total_input_tokens"
  echo "$total_output_tokens"
  echo "$total_cost"
}

# Get metrics for both directories
METRICS_A=($(extract_metrics "$DIR_A"))
METRICS_B=($(extract_metrics "$DIR_B"))

# Unpack metrics
TOTAL_A=${METRICS_A[0]}
FIRST_SUCCESS_A=${METRICS_A[1]}
FINAL_SUCCESS_A=${METRICS_A[2]}
FIRST_RATE_A=${METRICS_A[3]}
FINAL_RATE_A=${METRICS_A[4]}
REPAIR_USED_A=${METRICS_A[5]}
REPAIR_SUCCESS_A=${METRICS_A[6]}
REPAIR_RATE_A=${METRICS_A[7]}
INPUT_TOKENS_A=${METRICS_A[8]}
OUTPUT_TOKENS_A=${METRICS_A[9]}
COST_A=${METRICS_A[10]}

TOTAL_B=${METRICS_B[0]}
FIRST_SUCCESS_B=${METRICS_B[1]}
FINAL_SUCCESS_B=${METRICS_B[2]}
FIRST_RATE_B=${METRICS_B[3]}
FINAL_RATE_B=${METRICS_B[4]}
REPAIR_USED_B=${METRICS_B[5]}
REPAIR_SUCCESS_B=${METRICS_B[6]}
REPAIR_RATE_B=${METRICS_B[7]}
INPUT_TOKENS_B=${METRICS_B[8]}
OUTPUT_TOKENS_B=${METRICS_B[9]}
COST_B=${METRICS_B[10]}

# Print comparison table
echo -e "${BOLD}Metric                    ${LABEL_A}            ${LABEL_B}            Difference${NC}"
echo "═══════════════════════════════════════════════════════════════════════════"

printf "%-26s %-15s %-15s %-15s\n" \
  "Total Benchmarks" \
  "$TOTAL_A" \
  "$TOTAL_B" \
  ""

printf "%-26s %-15s %-15s " \
  "0-shot Success" \
  "${FIRST_SUCCESS_A}/${TOTAL_A} (${FIRST_RATE_A}%)" \
  "${FIRST_SUCCESS_B}/${TOTAL_B} (${FIRST_RATE_B}%)"

# Calculate difference
if [ "$FIRST_RATE_A" != "0.0" ] && [ "$FIRST_RATE_B" != "0.0" ]; then
  DIFF=$(echo "$FIRST_RATE_B - $FIRST_RATE_A" | bc)
  if (( $(echo "$DIFF > 0" | bc -l) )); then
    echo -e "${GREEN}+${DIFF}%${NC}"
  elif (( $(echo "$DIFF < 0" | bc -l) )); then
    echo -e "${RED}${DIFF}%${NC}"
  else
    echo "±0.0%"
  fi
else
  echo ""
fi

printf "%-26s %-15s %-15s " \
  "Final Success" \
  "${FINAL_SUCCESS_A}/${TOTAL_A} (${FINAL_RATE_A}%)" \
  "${FINAL_SUCCESS_B}/${TOTAL_B} (${FINAL_RATE_B}%)"

# Calculate difference
if [ "$FINAL_RATE_A" != "0.0" ] && [ "$FINAL_RATE_B" != "0.0" ]; then
  DIFF=$(echo "$FINAL_RATE_B - $FINAL_RATE_A" | bc)
  if (( $(echo "$DIFF > 0" | bc -l) )); then
    echo -e "${GREEN}+${DIFF}%${NC}"
  elif (( $(echo "$DIFF < 0" | bc -l) )); then
    echo -e "${RED}${DIFF}%${NC}"
  else
    echo "±0.0%"
  fi
else
  echo ""
fi

printf "%-26s %-15s %-15s %-15s\n" \
  "Repair Attempts" \
  "$REPAIR_USED_A" \
  "$REPAIR_USED_B" \
  ""

if [ "$REPAIR_RATE_A" != "N/A" ] && [ "$REPAIR_RATE_B" != "N/A" ]; then
  printf "%-26s %-15s %-15s " \
    "Repair Success Rate" \
    "${REPAIR_SUCCESS_A}/${REPAIR_USED_A} (${REPAIR_RATE_A}%)" \
    "${REPAIR_SUCCESS_B}/${REPAIR_USED_B} (${REPAIR_RATE_B}%)"

  DIFF=$(echo "$REPAIR_RATE_B - $REPAIR_RATE_A" | bc)
  if (( $(echo "$DIFF > 0" | bc -l) )); then
    echo -e "${GREEN}+${DIFF}%${NC}"
  elif (( $(echo "$DIFF < 0" | bc -l) )); then
    echo -e "${RED}${DIFF}%${NC}"
  else
    echo "±0.0%"
  fi
else
  printf "%-26s %-15s %-15s %-15s\n" \
    "Repair Success Rate" \
    "${REPAIR_RATE_A}" \
    "${REPAIR_RATE_B}" \
    ""
fi

echo "───────────────────────────────────────────────────────────────────────────"

printf "%-26s %-15s %-15s %-15s\n" \
  "Total Input Tokens" \
  "$INPUT_TOKENS_A" \
  "$INPUT_TOKENS_B" \
  ""

printf "%-26s %-15s %-15s %-15s\n" \
  "Total Output Tokens" \
  "$OUTPUT_TOKENS_A" \
  "$OUTPUT_TOKENS_B" \
  ""

printf "%-26s %-15s %-15s %-15s\n" \
  "Total Cost (USD)" \
  "\$$COST_A" \
  "\$$COST_B" \
  ""

echo ""
echo -e "${CYAN}Key Metrics:${NC}"
echo -e "  • ${BOLD}0-shot Success${NC}: First attempt without error feedback"
echo -e "  • ${BOLD}Final Success${NC}: Success after optional self-repair"
echo -e "  • ${BOLD}Repair Success Rate${NC}: How often self-repair fixed errors"
echo ""

# Provide recommendation
if [ "$FIRST_RATE_A" != "0.0" ] && [ "$FIRST_RATE_B" != "0.0" ]; then
  DIFF=$(echo "$FIRST_RATE_B - $FIRST_RATE_A" | bc)
  if (( $(echo "$DIFF > 5" | bc -l) )); then
    echo -e "${GREEN}✓ Recommendation: ${LABEL_B} shows significant improvement (+${DIFF}% first-attempt success)${NC}"
  elif (( $(echo "$DIFF < -5" | bc -l) )); then
    echo -e "${YELLOW}⚠ Warning: ${LABEL_B} performs worse (${DIFF}% first-attempt success)${NC}"
  else
    echo -e "${CYAN}ℹ No significant difference between prompt versions${NC}"
  fi
fi

echo ""
