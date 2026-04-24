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
#   PARALLEL=N - Number of parallel jobs (default: 15)
#   RESUME=true - Resume interrupted run (skip existing results, don't delete)
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
  echo "  VERSION=v0.3.10 RESUME=true ./tools/eval_baseline.sh"
  echo ""
  echo "Or use make target:"
  echo "  make eval-baseline EVAL_VERSION=v0.3.10"
  echo "  make eval-baseline EVAL_VERSION=v0.3.10 RESUME=true"
  echo ""
  exit 1
fi
FULL_SUITE="${FULL:-false}"  # Set FULL=true for full expensive suite
RESUME="${RESUME:-false}"  # Set RESUME=true to skip existing results
MODELS="${MODELS:-}"  # Custom model list (comma-separated)
LANGS="${LANGS:-python,ailang}"
PARALLEL="${PARALLEL:-15}"
TIER="${TIER:-}"  # Optional tier filter (smoke,core,stretch,vision); empty = all tiers

BASELINE_DIR="eval_results/baselines/${VERSION}"

# Determine model description for display
if [ -n "$MODELS" ]; then
  MODEL_DESC="$MODELS (custom)"
elif [ "$FULL_SUITE" = "true" ]; then
  MODEL_DESC="All 6 models: gpt5-4, gpt5-4-mini, claude-opus-4-7, claude-sonnet-4-6, gemini-3-1-pro, gemini-3-flash (--full)"
else
  MODEL_DESC="Dev models: gpt5-4-mini, claude-haiku-4-5, gemini-3-flash (default)"
fi

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Creating Baseline: ${BOLD}${VERSION}${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo "  Version:     $VERSION"
echo "  Models:      $MODEL_DESC"
echo "  Languages:   $LANGS"
echo "  Parallel:    $PARALLEL"
echo "  Tier:        ${TIER:-all}"
echo "  Self-repair: ENABLED (critical for agentic AI evaluation)"
echo "  Output:      $BASELINE_DIR"
echo ""

# Check if baseline already exists
if [ -d "$BASELINE_DIR" ]; then
  if [ "$RESUME" = "true" ]; then
    EXISTING_COUNT=$(find "$BASELINE_DIR" -name "*.json" -type f | wc -l | tr -d ' ')
    echo -e "${CYAN}→ Resuming existing baseline ($EXISTING_COUNT results found)${NC}"
  else
    echo -e "${YELLOW}⚠ Warning: Baseline for $VERSION already exists${NC}"
    echo ""
    read -p "Overwrite existing baseline? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      echo "Aborted"
      exit 1
    fi
    rm -rf "$BASELINE_DIR"
    # Create baseline directory
    mkdir -p "$BASELINE_DIR"
  fi
else
  # Create baseline directory
  mkdir -p "$BASELINE_DIR"
fi

# Run benchmark suite with parallel execution
echo -e "${CYAN}Running benchmark suite...${NC}"
echo ""

# Build command with conditional flags
# CRITICAL: Always enable self-repair for agentic AI evaluation
CMD=(bin/ailang eval-suite --langs "$LANGS" --parallel "$PARALLEL" --output "$BASELINE_DIR" -self-repair)

if [ -n "$MODELS" ]; then
  # User specified custom models
  CMD+=(--models "$MODELS")
elif [ "$FULL_SUITE" = "true" ]; then
  # Full expensive suite
  CMD+=(--full)
fi
# Otherwise, use default (dev models)

# Add --tier filter if requested (v0.14.0+)
if [ -n "$TIER" ]; then
  CMD+=(--tier "$TIER")
fi

# Add --skip-existing if resuming
if [ "$RESUME" = "true" ]; then
  CMD+=(--skip-existing)
fi

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

# Capture the resolved benchmark list (ground truth from result files, not just
# the --tier spec). This locks in what "this release's baseline" actually
# measured, so longitudinal comparisons can detect set drift across releases.
# Scope to the standard/ subdir so this block covers the standard stage only;
# the agent stage (run separately by the post-release wrapper) augments
# baseline.json with a parallel "agent" object.
STD_RESULTS_DIR="$BASELINE_DIR/standard"
if [ ! -d "$STD_RESULTS_DIR" ]; then
  # Fallback for callers that write results directly to $BASELINE_DIR (no subdir)
  STD_RESULTS_DIR="$BASELINE_DIR"
fi
BENCHMARK_IDS_JSON=$(find "$STD_RESULTS_DIR" -name "*.json" -type f ! -name "baseline.json" \
  -exec jq -r '.id' {} \; 2>/dev/null \
  | sort -u \
  | jq -R . \
  | jq -sc .)
BENCHMARK_IDS_JSON="${BENCHMARK_IDS_JSON:-[]}"
BENCHMARK_COUNT=$(echo "$BENCHMARK_IDS_JSON" | jq 'length')
STD_FILE_COUNT=$(find "$STD_RESULTS_DIR" -name "*.json" -type f ! -name "baseline.json" 2>/dev/null | wc -l | tr -d ' ')

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
  "standard": {
    "tier_spec": "${TIER:-}",
    "count": $BENCHMARK_COUNT,
    "files": $STD_FILE_COUNT,
    "models": "$ACTUAL_MODELS",
    "langs": "$LANGS",
    "resolved": $BENCHMARK_IDS_JSON
  },
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
echo "  2. Analyze: ailang eval-analyze -results eval_results/current -dry-run"
echo "  3. Compare: ailang eval-compare $BASELINE_DIR eval_results/current"
echo ""
