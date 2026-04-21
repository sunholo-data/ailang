#!/usr/bin/env bash
# benchmark_health.sh — rotation-decision report for a baseline.
#
# Identifies benchmarks that should be retired, promoted, or demoted:
#   - Saturated benchmarks (100% pass across all models × languages)
#   - Model refusals (by agent-mode transcript, if available)
#   - Tier-level headroom (benchmarks blocking tier promotion)
#
# Usage:
#   benchmark_health.sh <baseline_dir>
#
# Example:
#   benchmark_health.sh eval_results/baselines/v0.13.0
#
# Output is designed to be pasted into the release-review doc.

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

if command -v ailang >/dev/null 2>&1; then
  AILANG="ailang"
elif [ -x "bin/ailang" ]; then
  AILANG="bin/ailang"
else
  echo "Error: ailang not found. Run 'make build' or 'make install'." >&2
  exit 1
fi

echo "╔════════════════════════════════════════════════════════════╗"
echo "║  Benchmark Health Report: $VERSION"
echo "╚════════════════════════════════════════════════════════════╝"
echo

# ─── 1. Saturated benchmarks (rotation candidates) ───────────────────────────
echo "── Saturated benchmarks (rotation candidates) ──────────────"
echo "   Retire if still saturated for 2 consecutive baselines."
echo
"$AILANG" eval-matrix "$BASELINE_DIR" "$VERSION" --show-saturated 2>/dev/null \
  | awk '/[Ss]aturated/{p=1} p' \
  || echo "  (no saturated benchmarks)"
echo

# ─── 2. Refusal detection (agent transcripts) ────────────────────────────────
echo "── Model refusals (task declined, not a code failure) ──────"
AGENT_DIR="$BASELINE_DIR/agent"
if [ -d "$AGENT_DIR" ]; then
  # Refusal heuristic: non-empty transcript but stdout_ok=false AND
  # transcript mentions typical refusal phrasing. This mirrors
  # internal/eval_analysis/refusal.go's DetectRefusal primitive.
  REFUSAL_COUNT=0
  for f in "$AGENT_DIR"/*.json; do
    [ -f "$f" ] || continue
    if jq -e '
      select(.stdout_ok == false) |
      select(
        (.stderr // "" | test("(?i)I (can’t|cannot|won’t|will not) (help|comply|assist|do)"))
        or (.stderr // "" | test("(?i)against my (guidelines|principles)"))
        or (.stderr // "" | test("(?i)decline (to|this)"))
      )
    ' "$f" >/dev/null 2>&1; then
      BENCH=$(jq -r '.benchmark_id // .benchmark // "?"' "$f")
      MODEL=$(jq -r '.model // "?"' "$f")
      LANG=$(jq -r '.lang // "?"' "$f")
      echo "  ⚠  $BENCH [$MODEL/$LANG] — refusal detected"
      REFUSAL_COUNT=$((REFUSAL_COUNT + 1))
    fi
  done
  if [ "$REFUSAL_COUNT" -eq 0 ]; then
    echo "  (no refusals detected)"
  else
    echo
    echo "  $REFUSAL_COUNT refusal(s) total. Run:"
    echo "    $AILANG eval-analyze -results $BASELINE_DIR -dry-run"
    echo "  for the full refusal breakdown from internal/eval_analysis."
  fi
else
  echo "  (no agent/ directory — skip refusal detection for standard eval)"
fi
echo

# ─── 3. Tier promotion candidates ────────────────────────────────────────────
echo "── Tier promotion signal (from dashboard JSON) ─────────────"
DASHBOARD_JSON="docs/static/benchmarks/latest.json"
if [ -f "$DASHBOARD_JSON" ] && jq -e '.tiers' "$DASHBOARD_JSON" >/dev/null 2>&1; then
  jq -r '
    .tiers | to_entries | .[] |
    . as $t |
    (if $t.key == "smoke" then
       if $t.value.ailang_success_rate < 0.95 then
         "  ⚠  smoke tier dropped below 95% — investigate before releasing"
       else empty end
     elif $t.key == "core" then
       if $t.value.ailang_success_rate >= 0.95 then
         "  ↑  core tier above 95% — consider promoting easier core benchmarks to smoke"
       elif $t.value.ailang_success_rate < 0.5 then
         "  ↓  core tier below 50% — investigate language regression or demote weakest benchmarks"
       else empty end
     elif $t.key == "stretch" then
       if $t.value.ailang_success_rate >= 0.7 then
         "  ↑  stretch tier above 70% — promotion candidates to core (see CURATION.md §5)"
       else empty end
     elif $t.key == "vision" then
       if $t.value.ailang_success_rate >= 0.3 then
         "  ↑  vision tier above 30% — some benchmarks ready for stretch"
       else empty end
     else empty end)
  ' "$DASHBOARD_JSON" || true

  # If jq produced no signal lines, print a reassuring note.
  SIGNALS=$(jq -r '
    .tiers | to_entries | .[] |
    select(
      (.key == "smoke"   and .value.ailang_success_rate < 0.95) or
      (.key == "core"    and (.value.ailang_success_rate >= 0.95 or .value.ailang_success_rate < 0.5)) or
      (.key == "stretch" and .value.ailang_success_rate >= 0.7) or
      (.key == "vision"  and .value.ailang_success_rate >= 0.3)
    )
  ' "$DASHBOARD_JSON")
  if [ -z "$SIGNALS" ]; then
    echo "  (all tiers within expected ranges — no promotion/demotion action needed)"
  fi
else
  echo "  (dashboard JSON missing .tiers — regenerate with:"
  echo "     $AILANG eval-report $BASELINE_DIR $VERSION --format=json)"
fi
echo

echo "╚════════════════════════════════════════════════════════════╝"
echo "Next steps: cross-check against previous baseline before retiring"
echo "or promoting. See benchmarks/CURATION.md §4 (rotation) and §5"
echo "(promotion/demotion criteria)."
