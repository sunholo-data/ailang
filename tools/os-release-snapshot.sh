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

# 1. Refresh latest.json from the accumulator (current rolling snapshot).
ailang eval-publish "rolling-$(date +%Y%m%d)" --rotation "$ROLL" --os-json "$LATEST" >/dev/null \
  || { echo "eval-publish failed"; exit 1; }

# 2. Snapshot THIS version for the history entry. Prefer the version-scoped
#    subdir ($ROLL/$VERSION, created by --bank-by-version) so the history point
#    reflects that version's own numbers, not the mixed multi-version root
#    accumulator (fixed 2026-07-11 — the mixed-root read was silently
#    cross-contaminating per-version history entries).
VER_DIR="$ROLL/$VERSION"
HIST_SRC="$LATEST"
if [ -d "$VER_DIR" ]; then
  VER_JSON="$(mktemp)"
  if ailang eval-publish "$VERSION" --rotation "$VER_DIR" --os-json "$VER_JSON" >/dev/null 2>&1; then
    HIST_SRC="$VER_JSON"
  fi
else
  echo "note: no per-version dir $VER_DIR — attributing the mixed accumulator to $VERSION" >&2
fi

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

# 3. Optional reset of active-model accumulator files (retired models kept).
if [ "$RESET" = "1" ]; then
  n=$(find "$ROLL" -name "*${ACTIVE_PATTERN}*" 2>/dev/null | wc -l | tr -d ' ')
  find "$ROLL" -name "*${ACTIVE_PATTERN}*" -delete 2>/dev/null || true
  echo "reset: cleared $n active-model (*${ACTIVE_PATTERN}*) files; retired models kept frozen"
  # 4. Re-publish so latest.json reflects the cleared (next-version) state.
  ailang eval-publish "rolling-$(date +%Y%m%d)" --rotation "$ROLL" --os-json "$LATEST" >/dev/null
fi

echo "✓ snapshot: $VERSION archived to $HISTORY$([ "$RESET" = 1 ] && echo '; accumulator reset for next release')"
