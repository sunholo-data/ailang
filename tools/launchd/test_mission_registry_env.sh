#!/bin/bash
# test_mission_registry_env.sh — pins AILANG_MISSION_REGISTRY export for both branches.
#
# WHY THIS EXISTS. The binary branch exported the registry inline; the legacy branch —
# the one every mission actually runs — did NOT, so a legacy child could discover a
# registry by CWD walk instead of the pinned driver root. Arm 3 kills that regression;
# arm 4 kills an export hoisted above branch selection (which would run on every fire).
#
# Extraction, not duplication: the function under test is awk'd out of
# mission-control.sh itself, so this suite cannot drift green against an edited driver.
set -eu
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="$HERE/mission-control.sh"

TMP="${TMPDIR:-/tmp}/mc-registry-$$"
mkdir -p "$TMP"
trap 'rm -rf "$TMP"' EXIT

awk '/^_mc_export_mission_registry\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/fn_registry.sh"
# Guard against a vacuous extraction (the seam guard test_mission_stall.sh uses).
[ -s "$TMP/fn_registry.sh" ] && grep -q "AILANG_MISSION_REGISTRY" "$TMP/fn_registry.sh" \
  || { echo "FAIL extraction: _mc_export_mission_registry not found in $DRIVER"; exit 1; }

# Arm 1: the helper exports the pinned registry root.
log() { :; }
export MC_DRIVER_ROOT="$TMP/driver"
mkdir -p "$MC_DRIVER_ROOT/missions"
. "$TMP/fn_registry.sh"
_mc_export_mission_registry
[ "${AILANG_MISSION_REGISTRY:-}" = "$TMP/driver/missions" ] \
  || { echo "FAIL arm1: exported '${AILANG_MISSION_REGISTRY:-}' want '$TMP/driver/missions'"; exit 1; }

# Arm 2: a missing missions/ root refuses (exit 2) and names the missing path.
# Kills validation removed: a silent success would let children walk the CWD.
unset AILANG_MISSION_REGISTRY
export MC_DRIVER_ROOT="$TMP/missing"
log() { printf '%s\n' "$*"; }
set +e
msg=$( ( . "$TMP/fn_registry.sh"; _mc_export_mission_registry ) 2>&1 )
rc=$?
set -e
[ "$rc" = 2 ] || { echo "FAIL arm2: rc=$rc want 2"; exit 1; }
case "$msg" in
  *"$TMP/missing/missions"*) ;;
  *) echo "FAIL arm2: message does not name $TMP/missing/missions: $msg"; exit 1 ;;
esac

# Arm 3: both branches are wired — def + binary call + legacy call.
n=$(grep -c "_mc_export_mission_registry" "$DRIVER")
[ "$n" = 3 ] || { echo "FAIL arm3: _mc_export_mission_registry occurs $n times, want 3"; exit 1; }
m=$(awk '/AILANG_MISSION_WORK_ITEM/,/^fi$/' "$DRIVER" | grep -c "_mc_export_mission_registry")
[ "$m" = 1 ] || { echo "FAIL arm3: binary-branch call count $m, want 1"; exit 1; }

# Arm 4: the legacy call sits BELOW the binary branch's closing fi, so the export cannot
# run before branch selection.
bin_start=$(grep -nF 'if [ -n "${AILANG_MISSION_WORK_ITEM:-}" ]; then' "$DRIVER" | head -1 | cut -d: -f1)
bin_fi=$(awk -v s="$bin_start" 'NR>s && /^fi$/{print NR; exit}' "$DRIVER")
legacy=$(grep -n '^_mc_export_mission_registry$' "$DRIVER" | cut -d: -f1)
[ -n "$bin_fi" ] && [ -n "$legacy" ] && [ "$legacy" -gt "$bin_fi" ] \
  || { echo "FAIL arm4: legacy call at '$legacy' is not below binary fi at '$bin_fi'"; exit 1; }

echo 'PASS mission registry env: export arm, missing-root refusal, both branches wired, ordering'
