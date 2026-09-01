#!/bin/bash
# Append one durable gate stamp for the current scheduled mission attempt.
set -u

if [ "${1:-}" != "stamp" ]; then
  echo "usage: mission-heartbeat.sh stamp <label> [note]" >&2
  exit 2
fi

label="${2:-}"
case "$label" in
  fired|gate-0|gate-1|gate-2|gate-3|gate-3b|gate-4|gate-5|complete|abort) ;;
  *) echo "mission-heartbeat: unknown label: $label" >&2; exit 2 ;;
esac

if [ -z "${MISSION_NAME:-}" ]; then
  echo "mission-heartbeat: MISSION_NAME unset — not a scheduled mission slot; no stamp written"
  exit 0
fi

state_dir="${AILANG_STATE_DIR:-$HOME/.ailang/state}"
mkdir -p "$state_dir" || exit 1
heartbeat="$state_dir/mission-${MISSION_NAME}-heartbeat"
attempt="${MISSION_ATTEMPT:-1}"
epoch=$(date +%s)
iso=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
note="${3:-}"
printf '%s\t%s\t%s\t%s\t%s\n' "$epoch" "$iso" "$label" "$attempt" "$note" >> "$heartbeat"
