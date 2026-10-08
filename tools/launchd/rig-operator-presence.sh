#!/usr/bin/env bash
# rig-operator-presence.sh — keep rig.operator in force while someone is using
# this Mac, so unattended GPU work steps aside. Kept alive by
# dev.ailang.rig-operator-presence.plist.
#
# Why: the GPU that runs the local models also draws the desktop. Measured
# 2026-10-06 on the Mac Studio: with the rotation filler generating on
# qwen3.8-27b the GPU sat at 98-100% and focusing a window took ~5s.
#
# Signal: HIDIdleTime, the seconds since the last keyboard/mouse event in any
# GUI session, including input arriving over Screen Sharing. SSH-only work is
# invisible to it.
#
# The marker carries this loop's pid and a short expiry, so if the watcher dies
# the rig resumes within one marker lifetime rather than staying paused. Readers:
# rig_operator_present (rig-lock.sh) and riglock.OperatorPresent (Go).

# shellcheck source=tools/launchd/rig-lock.sh
source "$(dirname "$0")/rig-lock.sh"

IDLE_SEC="${RIG_OPERATOR_IDLE_SEC:-600}"   # present = input within the last 10 min
POLL_SEC="${RIG_OPERATOR_POLL_SEC:-20}"
TTL_SEC=$((POLL_SEC * 3))                   # survives two missed polls

trap 'rm -f "$RIG_OPERATOR_FILE"' EXIT
trap 'exit 0' TERM INT

was=""
while :; do
  idle=$(ioreg -c IOHIDSystem | awk '/HIDIdleTime/ {print int($NF/1000000000); exit}')
  if [ -n "$idle" ] && [ "$idle" -lt "$IDLE_SEC" ]; then
    printf 'requester=operator pid=%s until=%s\n' "$$" \
      "$(date -u -v+"${TTL_SEC}"S +%FT%TZ)" > "$RIG_OPERATOR_FILE.tmp" \
      && mv -f "$RIG_OPERATOR_FILE.tmp" "$RIG_OPERATOR_FILE"
    now="present"
  else
    rm -f "$RIG_OPERATOR_FILE"
    now="idle"
  fi
  [ "$now" != "$was" ] && echo "[$(date '+%F %T')] operator $now (HIDIdleTime=${idle:-?}s, threshold ${IDLE_SEC}s)"
  was="$now"
  sleep "$POLL_SEC"
done
