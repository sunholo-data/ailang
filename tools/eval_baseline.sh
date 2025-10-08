#!/usr/bin/env bash
#
# eval_baseline.sh - Store current eval results as baseline for comparison
#
# Usage:
#   ./tools/eval_baseline.sh [VERSION]
#
# Example:
#   ./tools/eval_baseline.sh v0.3.0-alpha5
#
# This script:
# 1. Runs full benchmark suite
# 2. Stores results in baselines/VERSION/
# 3. Generates performance matrix
# 4. Creates baseline marker for future comparisons

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Defaults
VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo "dev")}"
MODEL="${MODEL:-claude-sonnet-4-5}"
LANGS="${LANGS:-ailang}"
SELF_REPAIR="${SELF_REPAIR:-true}"

BASELINE_DIR="eval_results/baselines/${VERSION}"
MATRIX_FILE="eval_results/performance_tables/${VERSION}.json"

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Creating Baseline: ${BOLD}${VERSION}${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo "  Version:     $VERSION"
echo "  Model:       $MODEL"
echo "  Languages:   $LANGS"
echo "  Self-repair: $SELF_REPAIR"
echo "  Output:      $BASELINE_DIR"
echo ""

# Check if baseline already exists
if [ -d "$BASELINE_DIR" ]; then
  echo -e "${YELLOW}⚠ Warning: Baseline for $VERSION already exists${NC}"
  echo ""
  read -p "Overwrite existing baseline? (y/N) " -n 1 -r
  echo ""
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 1
  fi
  rm -rf "$BASELINE_DIR"
fi

# Create baseline directory
mkdir -p "$BASELINE_DIR"

# Find all benchmarks
BENCHMARKS=$(find benchmarks -name "*.yml" -type f | sort)
BENCHMARK_COUNT=$(echo "$BENCHMARKS" | wc -l | tr -d ' ')

echo -e "${CYAN}Running ${BENCHMARK_COUNT} benchmarks...${NC}"
echo ""

# Build flags
EVAL_FLAGS="--langs ${LANGS} --model ${MODEL}"
if [ "$SELF_REPAIR" = "true" ]; then
  EVAL_FLAGS="$EVAL_FLAGS --self-repair"
fi

# Run each benchmark
BENCH_NUM=0
SUCCESS_COUNT=0
FAIL_COUNT=0

for BENCH_FILE in $BENCHMARKS; do
  BENCH_NUM=$((BENCH_NUM + 1))
  BENCH_ID=$(basename "$BENCH_FILE" .yml)

  printf "[%2d/%2d] %-30s " "$BENCH_NUM" "$BENCHMARK_COUNT" "$BENCH_ID"

  if bin/ailang eval --benchmark "$BENCH_ID" --output "$BASELINE_DIR" $EVAL_FLAGS 2>&1 | grep -q "✓"; then
    echo -e "${GREEN}✓${NC}"
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  else
    echo -e "${RED}✗${NC}"
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi
done

echo ""
echo -e "${GREEN}✓ Benchmarks complete${NC}"
echo "  Success: $SUCCESS_COUNT"
echo "  Failed:  $FAIL_COUNT"
echo ""

# Generate performance matrix
echo -e "${CYAN}Generating performance matrix...${NC}"
./tools/generate_matrix_json.sh "$BASELINE_DIR" "$VERSION" "$MATRIX_FILE"

# Create baseline metadata
METADATA_FILE="${BASELINE_DIR}/baseline.json"
cat > "$METADATA_FILE" << EOF
{
  "version": "$VERSION",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "model": "$MODEL",
  "languages": "$LANGS",
  "self_repair": $SELF_REPAIR,
  "total_benchmarks": $BENCHMARK_COUNT,
  "success_count": $SUCCESS_COUNT,
  "fail_count": $FAIL_COUNT,
  "matrix_file": "$MATRIX_FILE",
  "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo "unknown")",
  "git_branch": "$(git branch --show-current 2>/dev/null || echo "unknown")"
}
EOF

echo ""
echo -e "${GREEN}✓ Baseline stored successfully${NC}"
echo ""
echo "  Baseline:     $BASELINE_DIR"
echo "  Matrix:       $MATRIX_FILE"
echo "  Metadata:     $METADATA_FILE"
echo ""
echo "Next steps:"
echo "  1. Make code changes"
echo "  2. Run: make eval-validate-fix BENCH=<benchmark_id>"
echo "  3. Compare results with: ./tools/eval_diff.sh $VERSION <new_run>"
echo ""
