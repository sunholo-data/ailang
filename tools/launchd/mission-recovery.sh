#!/usr/bin/env bash
#
# mission-recovery.sh — reclaim the wall-clock a probe stall costs the mission loop.
#
# THE PROBLEM. mission-control.sh probes each candidate controller model before
# spending anything. During the API-side new-session stalls first seen 2026-08-03
# (peak 08-05: 21 probe timeouts), every probe times out at 120s and the driver
# refuses. StartInterval is 5400s, so that slot is gone for 90 minutes even when
# the stall clears in 20 — on 2026-08-10 it refused at 07:59 and the very next fire
# probed OK in 24s at 09:29, i.e. ~85 minutes idle for a stall that was already over.
#
# THE FIX. Poll every few minutes; when the loop is in a blocked episode and nothing
# is running, kickstart it. Recovery drops from "up to 90 min" to "cooldown + poll".
#
# WHY THE DRIVER IS THE PROBE. This script does NOT probe the API itself. During
# these episodes small requests pass while heavy repo-cwd session creation stalls,
# so a cheap probe here would go green and kickstart into a driver that then refuses
# again. Letting the driver's own probe decide is representative by construction, and
# a failed retry costs ZERO tokens (timed-out probes return empty) — only wall time.
#
# SAFETY. Never resurrects a deliberately disabled mission: the kill switch is checked
# first and wins. Acts only while the driver's own .blocked marker is present, so a
# healthy or idle loop is never touched.
#
# Install:
#   cp tools/launchd/dev.ailang.mission-recovery.plist ~/Library/LaunchAgents/
#   launchctl load ~/Library/LaunchAgents/dev.ailang.mission-recovery.plist
# Disable (survives reboot):
#   touch ~/.ailang/state/mission-recovery.disabled
set -uo pipefail

MISSION_NAME="${MISSION_NAME:-v1}"
STATE_DIR="$HOME/.ailang/state"
LOG=/tmp/ailang-mission-recovery.log

if [ "$MISSION_NAME" = "v1" ]; then
	KILL_SWITCH="$STATE_DIR/mission-control.disabled"
	PIDFILE="$STATE_DIR/mission-control.pid"
	BLOCKED_FILE="$STATE_DIR/mission-control.blocked"
	TARGET_LABEL="dev.ailang.mission-control"
else
	KILL_SWITCH="$STATE_DIR/mission-${MISSION_NAME}.disabled"
	PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"
	BLOCKED_FILE="$STATE_DIR/mission-${MISSION_NAME}.blocked"
	TARGET_LABEL="dev.ailang.mission-${MISSION_NAME}"
fi
LAST_KICK_FILE="$STATE_DIR/mission-${MISSION_NAME}-recovery-last"
SELF_KILL_SWITCH="$STATE_DIR/mission-recovery.disabled"

# The driver writes its PIDFILE only AFTER the probe phase (it guards the claude
# run, not the probes), so a fire that is still probing is invisible to the overlap
# check. A refusal cycle is 3 models x 2 attempts x 120s plus the codex/pi lane
# probes -- ~13 min observed, ~16 min worst case. The cooldown must exceed that or
# retries stack into concurrent probe storms.
COOLDOWN="${MISSION_RECOVERY_COOLDOWN:-1200}"   # 20 min

log() { echo "[$(date '+%F %H:%M:%S')] $*" >> "$LOG"; }

# 1. Our own kill switch.
[ -f "$SELF_KILL_SWITCH" ] && exit 0

# 2. The mission's kill switch WINS. A disabled mission stays disabled -- resurrecting
#    a loop Mark deliberately turned off is the one unrecoverable mistake here.
[ -f "$KILL_SWITCH" ] && exit 0

# 3. Only act while the driver says it is blocked. Absent marker = healthy or idle.
[ -f "$BLOCKED_FILE" ] || exit 0

# 4. An iteration already running (post-probe) -- nothing to recover.
if [ -f "$PIDFILE" ]; then
	oldpid=$(head -1 "$PIDFILE" 2>/dev/null)
	if [ -n "$oldpid" ] && kill -0 "$oldpid" 2>/dev/null; then
		exit 0
	fi
fi

# 5. Cooldown since our last kickstart (see COOLDOWN note above).
now=$(date +%s)
if [ -f "$LAST_KICK_FILE" ]; then
	last=$(head -1 "$LAST_KICK_FILE" 2>/dev/null)
	case "$last" in
		''|*[!0-9]*) last=0 ;;   # corrupt/empty -> treat as never
	esac
	elapsed=$((now - last))
	if [ "$elapsed" -lt "$COOLDOWN" ]; then
		exit 0
	fi
fi

# 6. Retry. The driver re-probes; if the stall persists it refuses again, silently
#    (its .blocked marker suppresses repeat announcements) and costs no tokens.
printf '%s\n' "$now" > "$LAST_KICK_FILE"
blocked_age=$(( now - $(stat -f %m "$BLOCKED_FILE" 2>/dev/null || echo "$now") ))
log "mission $MISSION_NAME blocked for ${blocked_age}s — kickstarting $TARGET_LABEL (cooldown ${COOLDOWN}s)"
if launchctl kickstart "gui/$(id -u)/$TARGET_LABEL" >>"$LOG" 2>&1; then
	log "kickstart issued"
else
	log "kickstart FAILED (rc=$?) — is $TARGET_LABEL loaded? (launchctl list | grep $TARGET_LABEL)"
fi
