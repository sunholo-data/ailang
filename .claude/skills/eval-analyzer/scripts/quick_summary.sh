#!/usr/bin/env bash
# Quick summary of eval baseline results
#
# Usage:
#   quick_summary.sh <baseline_dir>
#
# Example:
#   quick_summary.sh eval_results/baselines/v0.3.16

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <baseline_dir>" >&2
    exit 1
fi

BASELINE_DIR="$1"
VERSION=$(basename "$BASELINE_DIR")

echo "╔═══════════════════════════════════════════════╗"
echo "║  Quick Summary: $VERSION"
echo "╚═══════════════════════════════════════════════╝"
echo

# Use ailang eval-matrix for quick stats
bin/ailang eval-matrix "$BASELINE_DIR" "$VERSION" 2>/dev/null | head -40

echo
echo "For detailed analysis, run:"
echo "  .claude/skills/eval-analyzer/scripts/analyze_failures.sh $BASELINE_DIR"
echo "  .claude/skills/eval-analyzer/scripts/compare_models.sh $BASELINE_DIR"

