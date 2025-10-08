#!/usr/bin/env bash
#
# eval_validate_fix.sh - Validate a fix by comparing against baseline
#
# Usage:
#   ./tools/eval_validate_fix.sh BENCHMARK_ID [BASELINE_VERSION]
#
# Example:
#   ./tools/eval_validate_fix.sh float_eq v0.3.0-alpha5
#
# This script:
# 1. Checks if baseline exists for the benchmark
# 2. Runs the benchmark with current code
# 3. Compares results and shows if the fix worked

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

if [ $# -lt 1 ]; then
  echo -e "${RED}Error: Missing benchmark ID${NC}"
  echo "Usage: $0 BENCHMARK_ID [BASELINE_VERSION]"
  echo ""
  echo "Example:"
  echo "  $0 float_eq v0.3.0-alpha5"
  echo ""
  echo "Available baselines:"
  if [ -d "eval_results/baselines" ]; then
    ls -1 eval_results/baselines/ 2>/dev/null | sed 's/^/  • /'
  else
    echo "  (none found - run: make eval-baseline)"
  fi
  exit 1
fi

BENCHMARK_ID="$1"
BASELINE_VERSION="${2:-$(ls -1 eval_results/baselines/ 2>/dev/null | tail -1)}"

if [ -z "$BASELINE_VERSION" ]; then
  echo -e "${RED}Error: No baseline version specified and no baselines found${NC}"
  echo "Run: make eval-baseline"
  exit 1
fi

BASELINE_DIR="eval_results/baselines/${BASELINE_VERSION}"
VALIDATION_DIR="eval_results/validation/${BENCHMARK_ID}_$(date +%Y%m%d_%H%M%S)"

# Check if baseline exists
if [ ! -d "$BASELINE_DIR" ]; then
  echo -e "${RED}Error: Baseline not found: $BASELINE_DIR${NC}"
  echo ""
  echo "Available baselines:"
  ls -1 eval_results/baselines/ 2>/dev/null | sed 's/^/  • /'
  exit 1
fi

# Check if benchmark exists in baseline
BASELINE_FILE=$(find "$BASELINE_DIR" -name "${BENCHMARK_ID}_*.json" | head -1)
if [ -z "$BASELINE_FILE" ]; then
  echo -e "${RED}Error: Benchmark $BENCHMARK_ID not found in baseline${NC}"
  echo ""
  echo "Available benchmarks in baseline:"
  find "$BASELINE_DIR" -name "*.json" -exec basename {} \; | sed 's/_.*$//' | sort -u | sed 's/^/  • /'
  exit 1
fi

# Extract baseline results
BASELINE_SUCCESS=$(jq -r '.stdout_ok' "$BASELINE_FILE")
BASELINE_ERROR=$(jq -r '.error_category' "$BASELINE_FILE")
BASELINE_STDERR=$(jq -r '.stderr' "$BASELINE_FILE")

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Validating Fix: ${BOLD}${BENCHMARK_ID}${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo "  Benchmark:   $BENCHMARK_ID"
echo "  Baseline:    $BASELINE_VERSION"
echo ""

# Show baseline status
echo -e "${BOLD}Baseline Status:${NC}"
if [ "$BASELINE_SUCCESS" = "true" ]; then
  echo -e "  ${GREEN}✓ Passing${NC}"
else
  echo -e "  ${RED}✗ Failing${NC} (${BASELINE_ERROR})"
  if [ "$BASELINE_STDERR" != "null" ] && [ -n "$BASELINE_STDERR" ]; then
    echo ""
    echo "  Error:"
    echo "$BASELINE_STDERR" | head -3 | sed 's/^/    /'
  fi
fi

echo ""
echo -e "${CYAN}Running benchmark with current code...${NC}"
echo ""

# Run benchmark with current code
mkdir -p "$VALIDATION_DIR"

if ! bin/ailang eval --benchmark "$BENCHMARK_ID" --output "$VALIDATION_DIR" --self-repair 2>&1 | tee /dev/tty | grep -q "✓"; then
  echo ""
  echo -e "${YELLOW}⚠ Benchmark execution completed with warnings/errors${NC}"
fi

# Find result file
RESULT_FILE=$(find "$VALIDATION_DIR" -name "${BENCHMARK_ID}_*.json" | head -1)

if [ -z "$RESULT_FILE" ]; then
  echo -e "${RED}Error: No result file found${NC}"
  exit 1
fi

# Extract new results
NEW_SUCCESS=$(jq -r '.stdout_ok' "$RESULT_FILE")
NEW_ERROR=$(jq -r '.error_category' "$RESULT_FILE")
NEW_STDERR=$(jq -r '.stderr' "$RESULT_FILE")
NEW_FIRST_OK=$(jq -r '.first_attempt_ok' "$RESULT_FILE")
NEW_REPAIR_USED=$(jq -r '.repair_used' "$RESULT_FILE")

echo ""
echo -e "${BOLD}New Status:${NC}"
if [ "$NEW_SUCCESS" = "true" ]; then
  echo -e "  ${GREEN}✓ Passing${NC}"
  if [ "$NEW_FIRST_OK" = "false" ] && [ "$NEW_REPAIR_USED" = "true" ]; then
    echo -e "    (${CYAN}ℹ${NC} Required self-repair)"
  fi
else
  echo -e "  ${RED}✗ Failing${NC} (${NEW_ERROR})"
  if [ "$NEW_STDERR" != "null" ] && [ -n "$NEW_STDERR" ]; then
    echo ""
    echo "  Error:"
    echo "$NEW_STDERR" | head -3 | sed 's/^/    /'
  fi
fi

echo ""
echo "═══════════════════════════════════════════════"

# Determine validation result
if [ "$BASELINE_SUCCESS" = "false" ] && [ "$NEW_SUCCESS" = "true" ]; then
  echo -e "${GREEN}✓ FIX VALIDATED: Benchmark now passing!${NC}"
  echo ""
  echo "The fix successfully resolved the issue."
  EXIT_CODE=0

elif [ "$BASELINE_SUCCESS" = "true" ] && [ "$NEW_SUCCESS" = "false" ]; then
  echo -e "${RED}✗ REGRESSION: Benchmark was passing, now failing!${NC}"
  echo ""
  echo "The change broke a previously passing benchmark."
  EXIT_CODE=1

elif [ "$BASELINE_SUCCESS" = "false" ] && [ "$NEW_SUCCESS" = "false" ]; then
  echo -e "${YELLOW}⚠ STILL FAILING: Benchmark remains broken${NC}"
  echo ""
  if [ "$BASELINE_ERROR" != "$NEW_ERROR" ]; then
    echo "Error category changed: $BASELINE_ERROR → $NEW_ERROR"
    echo "The fix may have partially helped, but more work needed."
  else
    echo "The fix did not resolve the issue."
  fi
  EXIT_CODE=1

else
  echo -e "${CYAN}ℹ NO CHANGE: Benchmark still passing${NC}"
  echo ""
  echo "The benchmark was already working."
  EXIT_CODE=0
fi

echo ""
echo "Results saved to: $VALIDATION_DIR"
echo ""

exit $EXIT_CODE
