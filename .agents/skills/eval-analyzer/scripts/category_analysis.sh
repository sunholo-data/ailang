#!/usr/bin/env bash
# category_analysis.sh — end-to-end per-tag analysis of a baseline.
#
# Surfaces:
#   - Per-tag AILANG vs Python delta (from `eval-matrix --by-tags`)
#   - Tier-level pass rates pulled from the published dashboard JSON
#   - Benchmarks AILANG wins on but Python fails
#
# Usage:
#   category_analysis.sh <baseline_dir>
#
# Example:
#   category_analysis.sh eval_results/baselines/v0.13.0
#
# Requires:
#   - `ailang` on PATH (or bin/ailang built via `make build`)
#   - `jq`
#   - `docs/static/benchmarks/latest.json` regenerated for the same baseline
#     (via `ailang eval-report <dir> <ver> --format=json`)

set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <baseline_dir>" >&2
  echo "Example: $0 eval_results/baselines/v0.13.0" >&2
  exit 1
fi

BASELINE_DIR="$1"

if [ ! -d "$BASELINE_DIR" ]; then
  echo "Error: Directory not found: $BASELINE_DIR" >&2
  exit 1
fi

VERSION="$(basename "$BASELINE_DIR")"
DASHBOARD_JSON="docs/static/benchmarks/latest.json"

# Resolve ailang binary
if command -v ailang >/dev/null 2>&1; then
  AILANG="ailang"
elif [ -x "bin/ailang" ]; then
  AILANG="bin/ailang"
else
  echo "Error: ailang not found. Run 'make build' or 'make install'." >&2
  exit 1
fi

echo "╔════════════════════════════════════════════════════════════╗"
echo "║  Category Analysis: $VERSION"
echo "╚════════════════════════════════════════════════════════════╝"
echo

# ─── 1. Tier-level pass rates from the dashboard JSON ────────────────────────
echo "── Tier pass rates (from $DASHBOARD_JSON) ───────────────────"
if [ -f "$DASHBOARD_JSON" ] && jq -e '.tiers' "$DASHBOARD_JSON" >/dev/null 2>&1; then
  jq -r '
    .tiers
    | to_entries
    | sort_by(.key)
    | .[]
    | "  \(.key | ascii_upcase)\t  AILANG \(.value.ailang_success_rate * 100 | . * 10 | round / 10)%\tPython \(.value.python_success_rate * 100 | . * 10 | round / 10)%\t(\(.value.benchmark_count) benchmarks)"
  ' "$DASHBOARD_JSON"
  echo
  CORE_AILANG=$(jq -r '.tiers.core.ailang_success_rate * 100 | . * 10 | round / 10' "$DASHBOARD_JSON")
  CORE_PY=$(jq -r '.tiers.core.python_success_rate * 100 | . * 10 | round / 10' "$DASHBOARD_JSON")
  echo "  ★ HEADLINE: Core tier is ${CORE_AILANG}% AILANG vs ${CORE_PY}% Python"
else
  echo "  (Dashboard JSON missing or lacks a .tiers block — regenerate with:"
  echo "     $AILANG eval-report $BASELINE_DIR $VERSION --format=json)"
fi
echo

# ─── 2. Per-tag delta table ──────────────────────────────────────────────────
echo "── Per-tag AILANG vs Python delta ───────────────────────────"
"$AILANG" eval-matrix "$BASELINE_DIR" "$VERSION" --by-tags 2>/dev/null \
  | awk '/^## By Tags/{p=1} p'
echo

# ─── 3. AILANG-only wins ─────────────────────────────────────────────────────
echo "── AILANG-only wins (benchmark × model) ─────────────────────"
"$AILANG" eval-matrix "$BASELINE_DIR" "$VERSION" --ailang-wins 2>/dev/null \
  | awk '/^## AILANG-Only Wins/{p=1} p' \
  || echo "  (no AILANG-only wins reported)"
echo

echo "╚════════════════════════════════════════════════════════════╝"
echo "Done. For promotion/demotion rules see benchmarks/CURATION.md §5."
