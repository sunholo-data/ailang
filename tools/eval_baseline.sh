#!/usr/bin/env bash
#
# eval_baseline.sh - Store current eval results as baseline for comparison
#
# Usage:
#   ./tools/eval_baseline.sh [VERSION] [MODEL] [LANGS]
#
# Example:
#   ./tools/eval_baseline.sh v0.3.0-alpha5
#   MODEL=gpt5 ./tools/eval_baseline.sh v0.3.1
#
# This script:
# 1. Runs full benchmark suite (using ailang eval-suite)
# 2. Stores results in baselines/VERSION/
# 3. Generates performance matrix
# 4. Creates git metadata

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
LANGS="${LANGS:-python,ailang}"
PARALLEL="${PARALLEL:-5}"

BASELINE_DIR="eval_results/baselines/${VERSION}"

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Creating Baseline: ${BOLD}${VERSION}${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo "  Version:     $VERSION"
echo "  Model:       $MODEL"
echo "  Languages:   $LANGS"
echo "  Parallel:    $PARALLEL"
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

# Run benchmark suite with parallel execution
echo -e "${CYAN}Running benchmark suite...${NC}"
echo ""

bin/ailang eval-suite \
  --models "$MODEL" \
  --langs "$LANGS" \
  --parallel "$PARALLEL" \
  --output "$BASELINE_DIR"

# Check results
SUCCESS_COUNT=$(find "$BASELINE_DIR" -name "*.json" -type f -exec jq -r 'select(.stdout_ok == true) | .id' {} \; 2>/dev/null | sort -u | wc -l | tr -d ' ')
TOTAL_COUNT=$(find "$BASELINE_DIR" -name "*.json" -type f | wc -l | tr -d ' ')
FAIL_COUNT=$((TOTAL_COUNT - SUCCESS_COUNT))

echo ""
echo -e "${GREEN}✓ Benchmarks complete${NC}"
echo "  Success: $SUCCESS_COUNT"
echo "  Failed:  $FAIL_COUNT"
echo "  Total:   $TOTAL_COUNT"
echo ""

# Generate performance matrix
echo -e "${CYAN}Generating performance matrix...${NC}"
if bin/ailang eval-matrix "$BASELINE_DIR" "$VERSION" 2>/dev/null; then
  echo -e "${GREEN}✓ Matrix generated${NC}"
else
  echo -e "${YELLOW}⚠ Matrix generation skipped${NC}"
fi

# Create baseline metadata
METADATA_FILE="${BASELINE_DIR}/baseline.json"
cat > "$METADATA_FILE" << EOF
{
  "version": "$VERSION",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "model": "$MODEL",
  "languages": "$LANGS",
  "parallel": $PARALLEL,
  "total_runs": $TOTAL_COUNT,
  "success_count": $SUCCESS_COUNT,
  "fail_count": $FAIL_COUNT,
  "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo "unknown")",
  "git_branch": "$(git branch --show-current 2>/dev/null || echo "unknown")",
  "git_dirty": $(git diff-index --quiet HEAD -- 2>/dev/null && echo "false" || echo "true")
}
EOF

echo ""
echo -e "${GREEN}✓ Baseline stored successfully${NC}"
echo ""
echo "  Baseline:  $BASELINE_DIR"
echo "  Metadata:  $METADATA_FILE"
echo "  Files:     $TOTAL_COUNT result files"
echo ""
echo "Next steps:"
echo "  1. Make code changes"
echo "  2. Run: make eval-validate-fix BENCH=<benchmark_id>"
echo "  3. Compare: ailang eval-compare $BASELINE_DIR eval_results/current"
echo ""
