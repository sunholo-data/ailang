#!/usr/bin/env bash
# os-release-snapshot.sh <ailang-version> [--reset]
#
# Records the local-rig OS leaderboard for one AILANG release into the
# longitudinal history, so the website can chart version-over-version evolution
# ("is each AILANG release moving the needle for local models?" —
# M-EVAL-OS-LONGITUDINAL).
#
#   1. Re-publishes os/latest.json from the rolling accumulator (current numbers).
#   2. Appends it to docs/static/benchmarks/os/history.json as an entry keyed by
#      <ailang-version>, deduped/replaced (re-running for the same version updates
#      it — useful while a version's numbers are still filling in).
#   3. With --reset: clears the ACTIVE-model files from the rolling accumulator so
#      the rotation re-measures fresh against the NEXT release. RETIRED models
#      (not matching ACTIVE_PATTERN) stay frozen at their last version.
#
# Convention: the post-release skill runs `os-release-snapshot.sh <version> --reset`
# once the rig is on the released version, attributing the accumulated numbers to
# that release and resetting for the next cycle. Without --reset it just snapshots
# (safe to re-run as numbers fill).
#
# ACTIVE_PATTERN (env, default "qwen3-6") selects which models --reset clears.
set -uo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:?usage: os-release-snapshot.sh <ailang-version> [--reset]}"
RESET=0; [ "${2:-}" = "--reset" ] && RESET=1
ACTIVE_PATTERN="${ACTIVE_PATTERN:-qwen3-6}"
ROLL="eval_results/rotation/os-rolling"
LATEST="docs/static/benchmarks/os/latest.json"
HISTORY="docs/static/benchmarks/os/history.json"

command -v ailang >/dev/null || { echo "ailang not on PATH"; exit 1; }

# 1+2. Publish latest.json AND the history-entry source from the VERSION-scoped
#    bank dir ONLY ($ROLL/$VERSION, created by --bank-by-version). --summarize
#    regenerates the dir's summary.json from raw result files first: eval-suite
#    only finalizes a summary when a suite COMPLETES, so an interrupted rotation
#    leaves the dir summary-less — and the old silent fallback then published the
#    ROOT accumulator, which findSummaryFiles walks RECURSIVELY, pooling EVERY
#    version's summaries into rows attributed to one version (found 2026-07-20:
#    the v0.30.0 history entry was actually multi-version pooled data).
#    NO fallback: failing loudly beats publishing misattributed numbers.
#    --ailang-version pins attribution to $VERSION even when the checkout has
#    already moved to the next release (the release-pickup snapshot path).
VER_DIR="$ROLL/$VERSION"
if [ ! -d "$VER_DIR" ]; then
  echo "ERROR: no per-version rotation dir $VER_DIR — nothing banked for $VERSION yet." >&2
  echo "       History entry NOT written, latest.json NOT touched. Re-run after the rotation banks results." >&2
  exit 1
fi
VER_JSON="$(mktemp)"
if ! ailang eval-publish "rolling-$(date +%Y%m%d)" --rotation "$VER_DIR" --summarize \
     --ailang-version "$VERSION" --os-json "$VER_JSON" >/dev/null; then
  echo "ERROR: eval-publish failed for $VER_DIR — history entry NOT written (no root fallback)." >&2
  exit 1
fi
cp "$VER_JSON" "$LATEST"
HIST_SRC="$VER_JSON"

# 3. Append/replace the <version> entry in history.json (from the per-version snapshot).
python3 - "$VERSION" "$HIST_SRC" "$HISTORY" <<'PY'
import json, sys, os
version, latest_p, hist_p = sys.argv[1], sys.argv[2], sys.argv[3]
latest = json.load(open(latest_p))
entry = {
    "ailang_version": version,
    "generated": latest.get("generated"),
    "trials": latest.get("trials"),
    "languages": latest.get("languages"),
    "rows": latest.get("rows", []),
}
hist = []
if os.path.exists(hist_p):
    try:
        hist = json.load(open(hist_p))
        if not isinstance(hist, list):
            hist = []
    except Exception:
        hist = []
hist = [e for e in hist if e.get("ailang_version") != version]  # dedupe/replace
hist.append(entry)
# Sort newest-first by version string (good enough for vMAJOR.MINOR.PATCH).
hist.sort(key=lambda e: e.get("ailang_version", ""), reverse=True)
os.makedirs(os.path.dirname(hist_p), exist_ok=True)
json.dump(hist, open(hist_p, "w"), indent=2)
print("history.json: %d version(s); %s has %d model rows" % (
    len(hist), version, len(entry["rows"])))
PY

# 3. Optional reset of active-model files in the ROOT accumulator working set
#    ONLY (retired models kept). Version bank dirs (v*/) are PERMANENT archives —
#    the old unrestricted find deleted active-model files from them too, which is
#    what destroyed v0.29.2's raw per-run files on the v0.30.0 release day
#    (2026-07-19) and would have wiped each version's history at the next release.
if [ "$RESET" = "1" ]; then
  n=$(find "$ROLL" -type d -name "v[0-9]*" -prune -o -type f -name "*${ACTIVE_PATTERN}*" -print 2>/dev/null | wc -l | tr -d ' ')
  find "$ROLL" -type d -name "v[0-9]*" -prune -o -type f -name "*${ACTIVE_PATTERN}*" -exec rm -f {} + 2>/dev/null || true
  echo "reset: cleared $n active-model (*${ACTIVE_PATTERN}*) files from the root accumulator; version banks + retired models kept"
  # latest.json intentionally NOT republished here: it keeps showing this (old)
  # version's final snapshot until the rotation banks the first results for the
  # NEW version and the filler's 7b snapshot flips it — always honest about what
  # the rows were measured on.
fi

echo "✓ snapshot: $VERSION archived to $HISTORY$([ "$RESET" = 1 ] && echo '; accumulator reset for next release')"
