#!/bin/bash
# Record that a role's lane is DEAD for the current mission fire (D-FLEET-2).
#
#   tools/launchd/mission-lane-dead.sh <role> <lane> <evidence>
#
# The spawn-pin hook allows an Agent-tool spawn on a provider-pinned role only when
# the requested model is a bare entry of the role's DECLARED fallback chain AND the
# pin plus every chain entry before it is dead this fire (or over ration). This
# script is the only writer of that ledger. It refuses a lane that is neither the
# role's pin nor in its declared chain, and refuses empty evidence — so a false
# record can at most reach the NEXT declared lane, never an undeclared one.
#
# Ledger: ${AILANG_STATE_DIR:-$HOME/.ailang/state}/mission-lane-dead/<MISSION_FIRE_ID>.tsv
# (one file per fire; a new fire starts clean). Exit 0 recorded, 2 refused.
set -u

role=${1:-}; lane=${2:-}; evidence=${3:-}
refuse() { printf 'mission-lane-dead: REFUSED — %s\n' "$1" >&2; exit 2; }

[ -n "${MISSION_FIRE_ID:-}" ] || refuse "MISSION_FIRE_ID is unset (not inside a mission fire)"
case "$role" in designer|planner|executor|evaluator) ;; *) refuse "role '$role' is not designer|planner|executor|evaluator" ;; esac
[ -n "$lane" ] || refuse "no lane given"
[ -n "$evidence" ] || refuse "no evidence given (a verdict path, an rc, an error line)"

role_uc=$(printf '%s' "$role" | tr 'a-z' 'A-Z')
pv="MISSION_${role_uc}_MODEL"; fv="MISSION_${role_uc}_FALLBACK"
pin="${!pv:-}"; chain="${!fv:-}"
declared=0
[ "$lane" = "$pin" ] && declared=1
case ",$chain," in *",$lane,"*) declared=1 ;; esac
[ "$declared" -eq 1 ] || refuse "'$lane' is neither $pv ('$pin') nor in $fv"

dir="${AILANG_STATE_DIR:-$HOME/.ailang/state}/mission-lane-dead"
mkdir -p "$dir" || refuse "cannot create $dir"
printf '%s\t%s\t%s\t%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$role" "$lane" \
  "$(printf '%s' "$evidence" | tr '\t\n' '  ' | cut -c1-500)" >> "$dir/${MISSION_FIRE_ID}.tsv" \
  || refuse "cannot write ledger"
printf 'recorded: %s lane %s dead for fire %s\n' "$role" "$lane" "$MISSION_FIRE_ID"
