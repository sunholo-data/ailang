#!/usr/bin/env bash
# cron-kicker.sh — Drive interval LaunchAgents from cron when gui/<uid> stops
# spawning them itself.
#
# WHY THIS EXISTS (measured 2026-09-04 04:52 → 2026-09-05 12:04, ~31h outage):
# every non-demand spawn in gui/501 stopped. Not one job — the whole domain.
# All four mission loops froze mid-schedule at the same instant, and Tailscale.app
# died with nothing to respawn it, which is how the laptops lost the tailnet.
#
# The two-armed proof, run 2026-09-05:
#   negative — a BRAND NEW label with RunAtLoad=true + StartInterval=20 sat for
#              100+ seconds at `runs = 0`, `pended nondemand spawn = interval`.
#              RunAtLoad is unambiguous: it did not run.
#   positive — `launchctl kickstart` on that same label ran it immediately.
# So: on-demand spawn works, non-demand spawn does not, and it is domain-wide.
# Long-lived KeepAlive jobs already running (coordinator, server, daemon) survived
# only because they never needed a respawn.
#
# cron is driven by com.vix.cron, a LaunchDaemon in the SYSTEM domain, which is
# unaffected. So cron can still reach gui/<uid> by explicit address even when
# gui/<uid> will no longer start anything on its own.
#
# SELF-DISARMING BY CONSTRUCTION. This script never asks "is launchd healthy?" —
# a question whose only honest answer needs the same broken machinery. It tracks
# each job's `runs` counter and the wall-clock at which that counter last moved.
# A job is kicked only when it is OVERDUE: its counter has not moved in longer
# than its own run interval. When launchd is healthy it moves the counter on
# schedule, nothing is ever overdue, and this script does nothing at all. No flag
# to clear, no second state to get wrong after a re-login.
#
# INSTALL (one crontab line, runs every minute):
#   * * * * * /Users/<you>/dev/sunholo-data/ailang/tools/launchd/cron-kicker.sh
#
# The real fix for the wedge is a re-login of the Aqua session. This is the
# backstop that keeps the loops alive until then — and, left installed, the one
# that catches the next occurrence without a human noticing first.
#
# bash 3.2 (this rig ships 3.2.57 and no 4.x). No associative arrays.

set -u

STATE_DIR="${AILANG_KICKER_STATE:-$HOME/.ailang/state/cron-kicker}"
LOG="${AILANG_KICKER_LOG:-$HOME/.ailang/state/cron-kicker.log}"
UID_NUMBER=$(id -u)
NOW=$(date +%s)

# Jobs this kicker drives. Deliberately NOT every interval job on the rig:
# os-rotation-filler and nightly-eval start GPU eval work, and ollama has twice
# taken the box down through memory (kernel panic 2026-09-03 with 177GB rpages on
# a 128GB machine; jetsam kills 2026-09-05 08:29 and 09:23). Auto-resuming that
# unattended, from a backstop whose whole job is to run without supervision, is
# the one thing here that could panic the box again. Opt in explicitly via
# AILANG_KICKER_EXTRA if you want them.
DEFAULT_LABELS="dev.ailang.mission-control
dev.ailang.mission-docs
dev.ailang.mission-world
dev.ailang.mission-motoko
dev.ailang.mission-recovery
dev.ailang.mission-recovery-motoko
dev.ailang.mission-resume
dev.ailang.rig-watchdog"

LABELS="${AILANG_KICKER_LABELS:-$DEFAULT_LABELS}
${AILANG_KICKER_EXTRA:-}"

mkdir -p "$STATE_DIR" || exit 1

log() {
    echo "[$(date '+%F %H:%M:%S')] $*" >> "$LOG"
}

for label in $LABELS; do
    [ -n "$label" ] || continue

    info=$(launchctl print "gui/${UID_NUMBER}/${label}" 2>/dev/null) || continue

    # A job with no run interval is calendar-driven or on-demand; its schedule is
    # not something a "has it moved lately?" test can reason about, so skip it
    # rather than guess an interval and fire at the wrong time.
    interval=$(printf '%s\n' "$info" | sed -n 's/^[[:space:]]*run interval = \([0-9]*\) seconds$/\1/p' | head -1)
    [ -n "$interval" ] || continue

    runs=$(printf '%s\n' "$info" | sed -n 's/^[[:space:]]*runs = \([0-9]*\)$/\1/p' | head -1)
    [ -n "$runs" ] || continue

    # Already running: not overdue, and kicking would be a no-op anyway. Bump the
    # observed time so a job that legitimately runs longer than its own interval
    # (mission-control routinely takes 2-3h on a 90m interval) is never treated as
    # stalled the moment it finishes.
    if printf '%s\n' "$info" | grep -q '^[[:space:]]*state = running$'; then
        echo "$runs $NOW" > "$STATE_DIR/$label"
        continue
    fi

    state_file="$STATE_DIR/$label"
    if [ -r "$state_file" ]; then
        last_runs=$(cut -d' ' -f1 "$state_file")
        last_change=$(cut -d' ' -f2 "$state_file")
    else
        last_runs=""
        last_change=""
    fi

    # First sight of a job, or a counter that just moved (by launchd or by us):
    # record and wait. Nothing is overdue until a full interval passes with no
    # movement, so a freshly-observed job is never kicked on this pass.
    if [ -z "$last_runs" ] || [ -z "$last_change" ] || [ "$runs" != "$last_runs" ]; then
        echo "$runs $NOW" > "$state_file"
        continue
    fi

    age=$(( NOW - last_change ))
    if [ "$age" -ge "$interval" ]; then
        if launchctl kickstart "gui/${UID_NUMBER}/${label}" >/dev/null 2>&1; then
            log "KICKED $label — counter stuck at runs=$runs for ${age}s (interval ${interval}s)"
        else
            log "FAILED to kick $label — counter stuck at runs=$runs for ${age}s"
        fi
        # Record the attempt, not a success: if the kick did not take, the next
        # pass sees an unchanged counter and an interval's worth of age again, so
        # it retries on the job's own cadence instead of every single minute.
        echo "$runs $NOW" > "$state_file"
    fi
done

exit 0
