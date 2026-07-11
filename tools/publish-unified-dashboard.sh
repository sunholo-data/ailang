#!/usr/bin/env bash
# publish-unified-dashboard.sh — regenerate the MAIN dashboard JSON
# (docs/static/benchmarks/latest.json) with CLOUD baseline results AND
# LOCAL on-device rotation results UNIFIED into a single leaderboard.
#
# The on-device roster (opencode/pi/motoko qwen3.6, gemma4) exists to be
# compared against the cloud frontier — but it banks into a separate rotation
# dir (eval_results/rotation/os-rolling/<version>/) and never reaches the main
# tables on its own. This script is the single command that publishes both
# together, using `eval-report --merge` to concatenate the two result sets
# before aggregation (cloud + local models share one ratings.agent leaderboard).
#
# Usage:
#   tools/publish-unified-dashboard.sh <version>
#   tools/publish-unified-dashboard.sh v0.29.2
#
# If no version is given, falls back to std/VERSION (the current build's
# release tag). The rotation dir is included via --merge ONLY when it exists,
# so a release with no local data yet still publishes the cloud-only view.
#
# IMPORTANT: eval-report writes latest.json itself and preserves history — do
# NOT redirect its stdout (see .claude/rules/eval.md). This script therefore
# lets stdout flow to the terminal.
set -uo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO" || exit 1

# Prefer the freshly-built binary if present, else fall back to PATH `ailang`.
AILANG="$REPO/bin/ailang"
[ -x "$AILANG" ] || AILANG="ailang"

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(tr -d '[:space:]' < std/VERSION 2>/dev/null || true)"
fi
if [ -z "$VERSION" ]; then
  echo "error: no version given and std/VERSION is empty" >&2
  echo "usage: $0 <version>" >&2
  exit 1
fi

BASELINE_DIR="eval_results/baselines/${VERSION}"
ROTATION_DIR="eval_results/rotation/os-rolling/${VERSION}"

if [ ! -d "$BASELINE_DIR" ]; then
  echo "error: cloud baseline dir not found: $BASELINE_DIR" >&2
  echo "       (run the post-release eval baselines first)" >&2
  exit 1
fi

echo "== publishing unified dashboard for ${VERSION} =="
echo "   cloud baseline : $BASELINE_DIR"

if [ -d "$ROTATION_DIR" ]; then
  echo "   local rotation : $ROTATION_DIR (merging)"
  "$AILANG" eval-report "$BASELINE_DIR" "$VERSION" \
    --merge "$ROTATION_DIR" --format=json
else
  echo "   local rotation : (none for ${VERSION} — publishing cloud-only)"
  "$AILANG" eval-report "$BASELINE_DIR" "$VERSION" --format=json
fi

status=$?
if [ "$status" -ne 0 ]; then
  echo "error: eval-report failed (exit $status)" >&2
  exit "$status"
fi

echo "== done: docs/static/benchmarks/latest.json refreshed (cloud + local) =="
