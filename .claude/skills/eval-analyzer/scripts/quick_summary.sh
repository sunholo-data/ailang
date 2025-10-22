#!/usr/bin/env bash
# Quick summary of eval baseline results using eval-matrix
#
# Usage:
#   quick_summary.sh <baseline_dir>
#
# Example:
#   quick_summary.sh eval_results/baselines/v0.3.16

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <baseline_dir>" >&2
    echo "" >&2
    echo "Example:" >&2
    echo "  $0 eval_results/baselines/v0.3.16" >&2
    exit 1
fi

BASELINE_DIR="$1"

if [ ! -d "$BASELINE_DIR" ]; then
    echo "Error: Directory not found: $BASELINE_DIR" >&2
    exit 1
fi

VERSION=$(basename "$BASELINE_DIR")

# Check if ailang binary exists
if [ ! -f "bin/ailang" ]; then
    echo "Error: bin/ailang not found. Run 'make build' first." >&2
    exit 1
fi

echo "╔═══════════════════════════════════════════════╗"
echo "║  Quick Summary: $VERSION"
echo "╚═══════════════════════════════════════════════╝"
echo

# Use ailang eval-matrix for quick stats
if ! bin/ailang eval-matrix "$BASELINE_DIR" "$VERSION" 2>/dev/null | head -40; then
    echo "Error: Failed to run eval-matrix. Check that result files exist in $BASELINE_DIR" >&2
    exit 1
fi

echo
echo "For detailed analysis, run:"
echo "  .claude/skills/eval-analyzer/scripts/analyze_failures.sh $BASELINE_DIR"
echo "  .claude/skills/eval-analyzer/scripts/compare_models.sh $BASELINE_DIR"
echo "  .claude/skills/eval-analyzer/scripts/examine_code.sh $BASELINE_DIR <benchmark_id>"

