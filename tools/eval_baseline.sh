#!/usr/bin/env bash
#
# eval_baseline.sh - Store current eval results as baseline for comparison
#
# Usage:
#   VERSION=v0.3.10 ./tools/eval_baseline.sh
#   VERSION=v0.3.10 FULL=true ./tools/eval_baseline.sh
#
# Required:
#   VERSION - Explicit version string (e.g., v0.3.10)
#
# Optional:
#   FULL=true - Use expensive models (gpt5, claude-sonnet-4-5, gemini-2-5-pro)
#   MODELS=... - Custom model list (comma-separated)
#   LANGS=... - Languages to test (default: python,ailang)
#   PARALLEL=N - Number of parallel jobs (default: 5)
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

# VERSION is now REQUIRED (no default from git describe)
VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
  echo -e "${RED}Error: VERSION environment variable is required${NC}"
  echo ""
  echo "Usage:"
  echo "  VERSION=v0.3.10 ./tools/eval_baseline.sh"
  echo "  VERSION=v0.3.10 FULL=true ./tools/eval_baseline.sh"
  echo ""
  echo "Or use make target:"
  echo "  make eval-baseline VERSION=v0.3.10"
  echo ""
  exit 1
fi
FULL_SUITE="${FULL:-false}"  # Set FULL=true for full expensive suite
MODELS="${MODELS:-}"  # Custom model list (comma-separated)
LANGS="${LANGS:-python,ailang}"
PARALLEL="${PARALLEL:-5}"

BASELINE_DIR="eval_results/baselines/${VERSION}"

# Determine model description for display
if [ -n "$MODELS" ]; then
  MODEL_DESC="$MODELS (custom)"
elif [ "$FULL_SUITE" = "true" ]; then
  MODEL_DESC="gpt5, claude-sonnet-4-5, gemini-2-5-pro (--full)"
else
  MODEL_DESC="gpt5-mini, gemini-2-5-flash (dev default)"
fi

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Creating Baseline: ${BOLD}${VERSION}${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo "  Version:     $VERSION"
echo "  Models:      $MODEL_DESC"
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

# Build command with conditional flags
CMD=(bin/ailang eval-suite --langs "$LANGS" --parallel "$PARALLEL" --output "$BASELINE_DIR")

if [ -n "$MODELS" ]; then
  # User specified custom models
  CMD+=(--models "$MODELS")
elif [ "$FULL_SUITE" = "true" ]; then
  # Full expensive suite
  CMD+=(--full)
fi
# Otherwise, use default (dev models)

"${CMD[@]}"

# Check results
TOTAL_COUNT=$(find "$BASELINE_DIR" -name "*.json" -type f | wc -l | tr -d ' ')

# Note: We don't cache success_count anymore - it's calculated dynamically from result files
# This prevents the "wrong success_count" bug (e.g., 20 vs actual 74 in v0.3.9)

echo ""
echo -e "${GREEN}✓ Benchmarks complete${NC}"
echo "  Total runs: $TOTAL_COUNT"
echo "  (Success count calculated dynamically from result files)"
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

# Determine actual models used (extract from result files)
ACTUAL_MODELS=$(find "$BASELINE_DIR" -name "*.json" -type f -exec jq -r '.model' {} \; 2>/dev/null | sort -u | paste -sd "," -)

# Get git describe for reference (but keep version separate)
GIT_DESCRIBE="$(git describe --tags --always 2>/dev/null || echo "unknown")"

cat > "$METADATA_FILE" << EOF
{
  "version": "$VERSION",
  "git_describe": "$GIT_DESCRIBE",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "models": "$ACTUAL_MODELS",
  "full_suite": $FULL_SUITE,
  "languages": "$LANGS",
  "parallel": $PARALLEL,
  "total_runs": $TOTAL_COUNT,
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
