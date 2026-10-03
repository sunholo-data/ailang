#!/bin/bash
# mission-arm.sh NAME [--check] [--force --reason TEXT] — the ONE way to arm a mission.
#
# m-mission-light-profile Phase 1 (HD-4, Mark 2026-10-01). Runs mission-lane-check.sh — the
# existing `ailang mission doctor` (configuration) plus runtime readiness of every lane — and
# removes the mission's kill switch only when it reports READY. The kill switch's note is kept as
# `<file>.released-<UTC stamp>` so the pause history survives. --force arms anyway, and only with
# --reason, which is appended to that kept note.
#
#   --check   report only; exit 0 READY, 1 NOT READY. The weekly lane job runs this.
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
NAME=""; CHECK_ONLY=0; FORCE=0; REASON=""
while [ $# -gt 0 ]; do
  case "$1" in
    --check) CHECK_ONLY=1; shift ;;
    --force) FORCE=1; shift ;;
    --reason) REASON="${2:-}"; shift 2 ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    -*) echo "mission-arm.sh: unknown flag $1" >&2; exit 2 ;;
    *) NAME="$1"; shift ;;
  esac
done
[ -n "$NAME" ] || { echo "usage: mission-arm.sh NAME [--check] [--force --reason TEXT]" >&2; exit 2; }
[ "$FORCE" = 1 ] && [ -z "$REASON" ] && { echo "mission-arm.sh: --force needs --reason TEXT" >&2; exit 2; }

KILL="$HOME/.ailang/state/mission-${NAME}.disabled"
[ "$NAME" = v1 ] && KILL="$HOME/.ailang/state/mission-control.disabled"

bash "$HERE/mission-lane-check.sh" "$NAME"; rc=$?
[ "$CHECK_ONLY" = 1 ] && exit "$rc"

if [ ! -f "$KILL" ]; then echo "mission-arm: $NAME is already armed (no $KILL)"; exit "$rc"; fi
if [ "$rc" -ne 0 ] && [ "$FORCE" != 1 ]; then
  echo "mission-arm: NOT arming $NAME — fix the dead rows above, or --force --reason TEXT" >&2
  exit 1
fi
kept="$KILL.released-$(date -u +%Y%m%dT%H%M%SZ)"
mv "$KILL" "$kept" || { echo "mission-arm: could not move $KILL" >&2; exit 1; }
if [ "$FORCE" = 1 ] && [ "$rc" -ne 0 ]; then
  printf '\nFORCE-ARMED %s despite NOT READY: %s\n' "$(date -u +%FT%TZ)" "$REASON" >> "$kept"
fi
echo "mission-arm: $NAME armed (pause note kept at $kept)"
