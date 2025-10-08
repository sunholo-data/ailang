#!/usr/bin/env bash
#
# eval_diff.sh - Compare two eval runs and show what changed
#
# Usage:
#   ./tools/eval_diff.sh BASELINE_DIR NEW_DIR [LABEL_BASELINE] [LABEL_NEW]
#
# Example:
#   ./tools/eval_diff.sh eval_results/baselines/v0.3.0 eval_results/after_fix "Before" "After Fix"

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

if [ $# -lt 2 ]; then
  echo -e "${RED}Error: Missing arguments${NC}"
  echo "Usage: $0 BASELINE_DIR NEW_DIR [LABEL_BASELINE] [LABEL_NEW]"
  echo ""
  echo "Example:"
  echo "  $0 eval_results/baselines/v0.3.0 eval_results/after_fix \"Before\" \"After Fix\""
  exit 1
fi

BASELINE_DIR="$1"
NEW_DIR="$2"
LABEL_BASELINE="${3:-Baseline}"
LABEL_NEW="${4:-New}"

# Verify directories exist
if [ ! -d "$BASELINE_DIR" ]; then
  echo -e "${RED}Error: Baseline directory not found: $BASELINE_DIR${NC}"
  exit 1
fi

if [ ! -d "$NEW_DIR" ]; then
  echo -e "${RED}Error: New directory not found: $NEW_DIR${NC}"
  exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
  echo -e "${RED}Error: jq is required but not installed${NC}"
  echo "Install with: brew install jq"
  exit 1
fi

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Eval Diff: ${LABEL_BASELINE} → ${LABEL_NEW}${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""

# Build maps of benchmark results
declare -A baseline_results
declare -A new_results

for file in "$BASELINE_DIR"/*.json; do
  if [ -f "$file" ]; then
    id=$(jq -r '.id' "$file")
    ok=$(jq -r '.stdout_ok' "$file")
    baseline_results["$id"]="$ok"
  fi
done

for file in "$NEW_DIR"/*.json; do
  if [ -f "$file" ]; then
    id=$(jq -r '.id' "$file")
    ok=$(jq -r '.stdout_ok' "$file")
    new_results["$id"]="$ok"
  fi
done

# Find changes
FIXED=()
BROKEN=()
STILL_PASSING=()
STILL_FAILING=()
NEW_BENCHMARKS=()
REMOVED_BENCHMARKS=()

for id in "${!baseline_results[@]}"; do
  baseline_ok="${baseline_results[$id]}"

  if [ -n "${new_results[$id]:-}" ]; then
    new_ok="${new_results[$id]}"

    if [ "$baseline_ok" = "false" ] && [ "$new_ok" = "true" ]; then
      FIXED+=("$id")
    elif [ "$baseline_ok" = "true" ] && [ "$new_ok" = "false" ]; then
      BROKEN+=("$id")
    elif [ "$baseline_ok" = "true" ] && [ "$new_ok" = "true" ]; then
      STILL_PASSING+=("$id")
    else
      STILL_FAILING+=("$id")
    fi
  else
    REMOVED_BENCHMARKS+=("$id")
  fi
done

for id in "${!new_results[@]}"; do
  if [ -z "${baseline_results[$id]:-}" ]; then
    NEW_BENCHMARKS+=("$id")
  fi
done

# Print summary
echo -e "${BOLD}Summary${NC}"
echo "═══════════════════════════════════════════════"
printf "%-30s %10s %10s\n" "" "$LABEL_BASELINE" "$LABEL_NEW"
printf "%-30s %10d %10d\n" "Total benchmarks" "${#baseline_results[@]}" "${#new_results[@]}"
echo ""

# Print fixed benchmarks
if [ ${#FIXED[@]} -gt 0 ]; then
  echo -e "${GREEN}✓ Fixed (${#FIXED[@]}):${NC}"
  for id in "${FIXED[@]}"; do
    echo "  • $id"
  done
  echo ""
fi

# Print broken benchmarks
if [ ${#BROKEN[@]} -gt 0 ]; then
  echo -e "${RED}✗ Broken (${#BROKEN[@]}):${NC}"
  for id in "${BROKEN[@]}"; do
    echo "  • $id"
  done
  echo ""
fi

# Print still passing
if [ ${#STILL_PASSING[@]} -gt 0 ]; then
  echo -e "${CYAN}→ Still passing (${#STILL_PASSING[@]}):${NC}"
  if [ ${#STILL_PASSING[@]} -le 10 ]; then
    for id in "${STILL_PASSING[@]}"; do
      echo "  • $id"
    done
  else
    echo "  (${#STILL_PASSING[@]} benchmarks - too many to list)"
  fi
  echo ""
fi

# Print still failing
if [ ${#STILL_FAILING[@]} -gt 0 ]; then
  echo -e "${YELLOW}⚠ Still failing (${#STILL_FAILING[@]}):${NC}"
  if [ ${#STILL_FAILING[@]} -le 10 ]; then
    for id in "${STILL_FAILING[@]}"; do
      echo "  • $id"
    done
  else
    echo "  (${#STILL_FAILING[@]} benchmarks - too many to list)"
  fi
  echo ""
fi

# Print new benchmarks
if [ ${#NEW_BENCHMARKS[@]} -gt 0 ]; then
  echo -e "${CYAN}+ New benchmarks (${#NEW_BENCHMARKS[@]}):${NC}"
  for id in "${NEW_BENCHMARKS[@]}"; do
    ok="${new_results[$id]}"
    if [ "$ok" = "true" ]; then
      echo -e "  • $id ${GREEN}(passing)${NC}"
    else
      echo -e "  • $id ${RED}(failing)${NC}"
    fi
  done
  echo ""
fi

# Print removed benchmarks
if [ ${#REMOVED_BENCHMARKS[@]} -gt 0 ]; then
  echo -e "${YELLOW}- Removed benchmarks (${#REMOVED_BENCHMARKS[@]}):${NC}"
  for id in "${REMOVED_BENCHMARKS[@]}"; do
    echo "  • $id"
  done
  echo ""
fi

# Calculate metrics
BASELINE_SUCCESS=0
NEW_SUCCESS=0

for ok in "${baseline_results[@]}"; do
  if [ "$ok" = "true" ]; then
    BASELINE_SUCCESS=$((BASELINE_SUCCESS + 1))
  fi
done

for ok in "${new_results[@]}"; do
  if [ "$ok" = "true" ]; then
    NEW_SUCCESS=$((NEW_SUCCESS + 1))
  fi
done

if [ ${#baseline_results[@]} -gt 0 ]; then
  BASELINE_RATE=$(echo "scale=1; $BASELINE_SUCCESS * 100 / ${#baseline_results[@]}" | bc)
else
  BASELINE_RATE="0.0"
fi

if [ ${#new_results[@]} -gt 0 ]; then
  NEW_RATE=$(echo "scale=1; $NEW_SUCCESS * 100 / ${#new_results[@]}" | bc)
else
  NEW_RATE="0.0"
fi

echo -e "${BOLD}Success Rates${NC}"
echo "═══════════════════════════════════════════════"
printf "%-30s %10s %10s\n" "$LABEL_BASELINE" "$BASELINE_SUCCESS/${#baseline_results[@]}" "(${BASELINE_RATE}%)"
printf "%-30s %10s %10s\n" "$LABEL_NEW" "$NEW_SUCCESS/${#new_results[@]}" "(${NEW_RATE}%)"
echo ""

# Calculate delta
if [ "$BASELINE_RATE" != "0.0" ] && [ "$NEW_RATE" != "0.0" ]; then
  DELTA=$(echo "$NEW_RATE - $BASELINE_RATE" | bc)
  if (( $(echo "$DELTA > 0" | bc -l) )); then
    echo -e "Change: ${GREEN}+${DELTA}%${NC} improvement"
  elif (( $(echo "$DELTA < 0" | bc -l) )); then
    echo -e "Change: ${RED}${DELTA}%${NC} regression"
  else
    echo "Change: No change in success rate"
  fi
else
  echo "Change: Unable to calculate (missing data)"
fi

echo ""
