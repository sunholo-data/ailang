#!/usr/bin/env bash
# Run evaluation baseline for a release version

set -euo pipefail

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <version> [--full]" >&2
    echo "Example: $0 0.3.14 --full" >&2
    echo "" >&2
    echo "Options:" >&2
    echo "  --full    Run with all production models (default: dev models only)" >&2
    exit 1
fi

VERSION="$1"
FULL_FLAG=""

if [[ $# -gt 1 ]] && [[ "$2" == "--full" ]]; then
    FULL_FLAG="FULL=true"
fi

echo "Running eval baseline for v$VERSION..."
if [[ -n "$FULL_FLAG" ]]; then
    echo "Mode: FULL (all 6 production models)"
    echo "Expected cost: ~\$0.50-1.00"
    echo "Expected time: ~15-20 minutes"
else
    echo "Mode: DEV (3 cheap models only)"
    echo "Expected cost: ~\$0.10-0.20"
    echo "Expected time: ~5-10 minutes"
fi
echo

# Run baseline
if [[ -n "$FULL_FLAG" ]]; then
    make eval-baseline EVAL_VERSION="$VERSION" FULL=true
else
    make eval-baseline EVAL_VERSION="$VERSION"
fi

# Show results location
RESULTS_DIR="eval_results/baselines/v$VERSION"
if [[ -d "$RESULTS_DIR" ]]; then
    FILE_COUNT=$(find "$RESULTS_DIR" -name "*.json" | wc -l | tr -d ' ')
    echo
    echo "✓ Baseline complete"
    echo "  Results: $RESULTS_DIR"
    echo "  Files: $FILE_COUNT result files"
else
    echo "✗ Results directory not found: $RESULTS_DIR" >&2
    exit 1
fi
