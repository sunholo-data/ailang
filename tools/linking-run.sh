#!/usr/bin/env bash
# linking-run.sh — the per-release measurement that REPLACES the full baseline
# (M-EVAL-ROLLING-ELO M3, design freeze D3 ratified by Mark 2026-08-27).
#
# Runs the FIXED direction panel against the ratified bridge, banks by version,
# persists the placement fit (+ trial_history), then stamps the release's
# language-direction index.
#
# Usage:
#   bash tools/linking-run.sh <version> [--dry-run]
#
# Cost: est. $12-16 (guard refuses >$25). Full baselines are now QUARTERLY
# re-anchor duty only: make eval-baseline FULL=true
#
# bash 3.2 compatible (the rig has no bash 4.x — no associative arrays, no ${v,,}).
set -euo pipefail

VERSION="${1:-}"
DRY="${2:-}"
if [ -z "$VERSION" ]; then
  echo "usage: bash tools/linking-run.sh <version> [--dry-run]" >&2
  exit 2
fi

# Ratified bridge panel (D3): three majors + two OpenRouter picks. Five members
# so the overlap-swap protocol tolerates any single vendor retirement.
BRIDGE="claude-sonnet-5,gpt5-6-terra,gemini-3-7-flash,or-glm-5-3-flash,or-deepseek-v4-flash"
PANEL_JSON="internal/eval_harness/direction_panel_v1.json"
OBS_DB="${OBSERVATORY_DB:-$HOME/.ailang/state/observatory.db}"
OUT_DIR="eval_results/baselines/${VERSION}"
COST_CAP="${LINKING_RUN_COST_CAP:-25}"

if [ ! -f "$PANEL_JSON" ]; then
  echo "missing $PANEL_JSON — generate it with tools/gen-anchor --direction-out" >&2
  exit 1
fi

# The panel is the benchmark list, verbatim: never confidence-selected, or the
# index averages a different set each release and stops being comparable.
BENCHES="$(python3 -c 'import json,sys; print(",".join(sorted(json.load(open(sys.argv[1]))["benchmarks"])))' "$PANEL_JSON")"
# grep -c counts lines including the last unterminated one (wc -l undercounts by 1).
NBENCH="$(printf '%s' "$BENCHES" | tr ',' '\n' | grep -c .)"

echo "linking run ${VERSION}"
echo "  panel:  ${NBENCH} benchmarks (${PANEL_JSON})"
echo "  bridge: ${BRIDGE}"
echo "  out:    ${OUT_DIR}"

if [ "$DRY" = "--dry-run" ]; then
  ailang eval-suite --models "$BRIDGE" --benchmarks "$BENCHES" --langs ailang,python --dry-run
  exit 0
fi

# 1) Measure. --skip-existing so a resumed run does not re-pay for banked cells.
ailang eval-suite \
  --models "$BRIDGE" \
  --benchmarks "$BENCHES" \
  --langs ailang,python \
  --output "$OUT_DIR" \
  --bank-by-version \
  --skip-existing \
  --parallel 4

# 2) Cost guard AFTER the fact as well as the estimate: a run that blew the cap
#    is a measurement we still keep, but it must be loud, not silent.
SPENT="$(python3 - "$OUT_DIR" <<'PY'
import glob, json, sys
tot = 0.0
for f in glob.glob(sys.argv[1] + "/**/*.json", recursive=True):
    try:
        d = json.load(open(f))
    except Exception:
        continue
    if isinstance(d, dict):
        tot += d.get("cost_usd") or 0
print(f"{tot:.2f}")
PY
)"
echo "  spent: \$${SPENT} (cap \$${COST_CAP})"
if [ "$(python3 -c "print(1 if float('$SPENT') > float('$COST_CAP') else 0)")" = "1" ]; then
  echo "WARNING: linking run exceeded the \$${COST_CAP} cap — investigate before the next release" >&2
fi

# 3) Placement fit + trial_history (standard mode, anchored).
go run ./tools/eval-elo "$OUT_DIR" --mode standard --persist "$OBS_DB"

# 4) Direction index — REFUSES on any missing panel cell (no partial index).
go run ./tools/direction-fit \
  --version "$VERSION" \
  --bridge "$BRIDGE" \
  --db "$OBS_DB" \
  --out "${OUT_DIR}/direction_index.json" \
  "$OUT_DIR"

echo "linking run ${VERSION} complete: ratings persisted, direction index stamped"
