#!/usr/bin/env bash
#
# eval_prompt_ab.sh - Run A/B comparison of two prompt versions
#
# Usage:
#   ./tools/eval_prompt_ab.sh PROMPT_A PROMPT_B [OPTIONS]
#
# Example:
#   ./tools/eval_prompt_ab.sh v0.3.0-baseline v0.3.0-hints --langs ailang --model claude-sonnet-4-5
#
# This script:
# 1. Runs all benchmarks with prompt version A
# 2. Runs all benchmarks with prompt version B
# 3. Saves results to separate directories
# 4. Optionally runs comparison analysis with compare_results.sh

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Defaults
LANGS="ailang"
MODEL="claude-sonnet-4-5"
SEED=42
SELF_REPAIR="false"
AUTO_COMPARE="true"
OUTPUT_BASE="eval_results/ab_test"

# Parse arguments
if [ $# -lt 2 ]; then
  echo -e "${RED}Error: Missing prompt versions${NC}"
  echo "Usage: $0 PROMPT_A PROMPT_B [--langs LANGS] [--model MODEL] [--seed SEED] [--self-repair] [--no-compare]"
  echo ""
  echo "Example:"
  echo "  $0 v0.3.0-baseline v0.3.0-hints --langs ailang --model claude-sonnet-4-5"
  echo ""
  echo "Options:"
  echo "  --langs LANGS        Languages to test (default: ailang)"
  echo "  --model MODEL        AI model to use (default: claude-sonnet-4-5)"
  echo "  --seed SEED          Random seed (default: 42)"
  echo "  --self-repair        Enable self-repair for both runs"
  echo "  --no-compare         Don't auto-run comparison analysis"
  exit 1
fi

PROMPT_A="$1"
PROMPT_B="$2"
shift 2

# Parse optional arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --langs)
      LANGS="$2"
      shift 2
      ;;
    --model)
      MODEL="$2"
      shift 2
      ;;
    --seed)
      SEED="$2"
      shift 2
      ;;
    --self-repair)
      SELF_REPAIR="true"
      shift
      ;;
    --no-compare)
      AUTO_COMPARE="false"
      shift
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      exit 1
      ;;
  esac
done

# Create output directories
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
OUTPUT_A="${OUTPUT_BASE}/${TIMESTAMP}_${PROMPT_A}"
OUTPUT_B="${OUTPUT_BASE}/${TIMESTAMP}_${PROMPT_B}"

mkdir -p "$OUTPUT_A"
mkdir -p "$OUTPUT_B"

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  A/B Prompt Testing${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo -e "  Prompt A:    ${GREEN}${PROMPT_A}${NC}"
echo -e "  Prompt B:    ${GREEN}${PROMPT_B}${NC}"
echo -e "  Languages:   ${LANGS}"
echo -e "  Model:       ${MODEL}"
echo -e "  Seed:        ${SEED}"
echo -e "  Self-repair: ${SELF_REPAIR}"
echo ""

# Find all benchmark files
BENCHMARKS=$(find benchmarks -name "*.yml" -type f | sort)
BENCHMARK_COUNT=$(echo "$BENCHMARKS" | wc -l | tr -d ' ')

echo -e "${CYAN}Found ${BENCHMARK_COUNT} benchmarks to test${NC}"
echo ""

# Build flags
EVAL_FLAGS="--langs ${LANGS} --model ${MODEL} --seed ${SEED}"
if [ "$SELF_REPAIR" = "true" ]; then
  EVAL_FLAGS="$EVAL_FLAGS --self-repair"
fi

# Run benchmarks with Prompt A
echo -e "${CYAN}════ Running with ${PROMPT_A} ════${NC}"
BENCH_NUM=0
for BENCH_FILE in $BENCHMARKS; do
  BENCH_NUM=$((BENCH_NUM + 1))
  BENCH_ID=$(basename "$BENCH_FILE" .yml)
  echo -e "[${BENCH_NUM}/${BENCHMARK_COUNT}] ${BENCH_ID}..."

  if bin/ailang eval --benchmark "$BENCH_ID" --prompt-version "$PROMPT_A" --output "$OUTPUT_A" $EVAL_FLAGS 2>&1 | grep -q "✓"; then
    echo -e "  ${GREEN}✓${NC} Success"
  else
    echo -e "  ${YELLOW}⚠${NC} Failed or partial success"
  fi
done

echo ""
echo -e "${CYAN}════ Running with ${PROMPT_B} ════${NC}"
BENCH_NUM=0
for BENCH_FILE in $BENCHMARKS; do
  BENCH_NUM=$((BENCH_NUM + 1))
  BENCH_ID=$(basename "$BENCH_FILE" .yml)
  echo -e "[${BENCH_NUM}/${BENCHMARK_COUNT}] ${BENCH_ID}..."

  if bin/ailang eval --benchmark "$BENCH_ID" --prompt-version "$PROMPT_B" --output "$OUTPUT_B" $EVAL_FLAGS 2>&1 | grep -q "✓"; then
    echo -e "  ${GREEN}✓${NC} Success"
  else
    echo -e "  ${YELLOW}⚠${NC} Failed or partial success"
  fi
done

echo ""
echo -e "${GREEN}✓ A/B testing complete${NC}"
echo ""
echo -e "Results saved to:"
echo -e "  A: ${OUTPUT_A}"
echo -e "  B: ${OUTPUT_B}"
echo ""

# Run comparison analysis if requested
if [ "$AUTO_COMPARE" = "true" ]; then
  echo -e "${CYAN}Running comparison analysis...${NC}"
  echo ""
  if [ -f "tools/compare_results.sh" ]; then
    ./tools/compare_results.sh "$OUTPUT_A" "$OUTPUT_B" "$PROMPT_A" "$PROMPT_B"
  else
    echo -e "${YELLOW}⚠ compare_results.sh not found, skipping analysis${NC}"
    echo "  View results with:"
    echo "    ./tools/compare_results.sh \"$OUTPUT_A\" \"$OUTPUT_B\" \"$PROMPT_A\" \"$PROMPT_B\""
  fi
else
  echo "To compare results, run:"
  echo "  ./tools/compare_results.sh \"$OUTPUT_A\" \"$OUTPUT_B\" \"$PROMPT_A\" \"$PROMPT_B\""
fi

echo ""
