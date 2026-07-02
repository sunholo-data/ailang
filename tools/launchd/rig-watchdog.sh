#!/usr/bin/env bash
# rig-watchdog.sh — Poll-based reliability backstop for the local-Ollama
# eval rig. Called every 60s by dev.ailang.rig-watchdog.plist.
#
# Checks two services:
#   - ollama serve at http://localhost:11434/api/tags
#   - ailang OTLP receiver at http://localhost:1957/health
#
# If either is unreachable, kickstart the corresponding launchd job.
# Logs each restart event to stdout (captured by the plist).
#
# Why this exists: macOS launchd's KeepAlive directive does NOT reliably
# respawn jobs after SIGKILL on macOS 26.x Tahoe in the gui/<uid> context.
# Empirically verified 2026-05-23: 30+ seconds after SIGKILL'ing the ollama
# serve process, launchd's "pended nondemand spawn = semaphore" gating
# prevents respawn. This watchdog catches the dropped case.

set -u

TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
UID_NUMBER=$(id -u)

# Check ollama
if ! curl --max-time 2 -s http://localhost:11434/api/tags >/dev/null 2>&1; then
    echo "${TIMESTAMP} [WATCHDOG] ollama unreachable — kickstart dev.ollama.serve"
    launchctl kickstart "gui/${UID_NUMBER}/dev.ollama.serve" 2>&1
fi

# Check ailang server (OTLP receiver). Not yet a launchd job by default — only
# kickstart if the plist exists (i.e. user opted into the launchd setup).
if ! curl --max-time 2 -s http://localhost:1957/health | grep -q healthy 2>/dev/null; then
    if launchctl print "gui/${UID_NUMBER}/dev.ailang.server" >/dev/null 2>&1; then
        echo "${TIMESTAMP} [WATCHDOG] ailang server unreachable — kickstart dev.ailang.server"
        launchctl kickstart "gui/${UID_NUMBER}/dev.ailang.server" 2>&1
    else
        # ailang server isn't launchd-managed on this rig; skip silently.
        # (User can install dev.ailang.server.plist for full automation.)
        :
    fi
fi

# ── WEDGE KILLER (M-RIG-WATCHDOG-WEDGE, 2026-07-01) ──────────────────────────────────
# The checks above only verify SERVICE availability (ollama/OTLP). They let a 9h-wedged
# os-rotation chunk sit, blocking every experiment (the rig-flakiness that ate days). Kill a
# rotation chunk — an `ailang eval-suite` spawned by os-rotation-filler — that is EITHER past a
# hard wall-clock OR alive-but-STALLED (no session-log write ⇒ no step-progress; distinguishes a
# wedge from a legitimately-slow docx run). Only FILLER-parented chunks are ever touched, so
# manual runs and the A/B daemons (ab_*.sh parents) are never killed. Also reap orphaned :8080
# env-servers (the port-8080-zombie that crashes the next run with "no run_summary").
LOGDIR="$HOME/dev/mk-ast/.motoko/logfile"
HARD_SECS=$(( ${RIG_WATCHDOG_HARD_HOURS:-8} * 3600 ))
SOFT_SECS=$(( ${RIG_WATCHDOG_SOFT_HOURS:-2} * 3600 ))
STALL_MIN=${RIG_WATCHDOG_STALL_MIN:-30}

etime_secs() {  # PID → elapsed seconds (0 on any miss). Takes the pid and reads `ps etime`
    # ITSELF — the caller passes $pid, so a version that parsed $1 as an etime string was
    # actually parsing the pid (a colonless number) as seconds. Robust under `set -u`: a pid
    # can die between pgrep and ps (empty etime), and a fresh process shows "MM:SS" (2 fields)
    # not "HH:MM:SS" — dereferencing $2/$3 unguarded aborted the whole watchdog in the
    # command-substitution subshell, so it NEVER reached the kill.
    local et days=0 hms h=0 m=0 s=0
    et=$(ps -o etime= -p "${1:-}" 2>/dev/null | tr -d ' ')
    [ -n "$et" ] || { echo 0; return; }
    case "$et" in *-*) days="${et%%-*}"; hms="${et#*-}";; *) hms="$et";; esac
    local IFS=:
    # shellcheck disable=SC2086
    set -- $hms
    if   [ "$#" -eq 3 ]; then h="${1:-0}"; m="${2:-0}"; s="${3:-0}"
    elif [ "$#" -eq 2 ]; then h=0;         m="${1:-0}"; s="${2:-0}"
    else                      h=0; m=0;    s="${1:-0}"; fi
    echo $(( days*86400 + 10#${h:-0}*3600 + 10#${m:-0}*60 + 10#${s:-0} ))
}

newest=$(ls -t "$LOGDIR"/session_*.jsonl 2>/dev/null | head -1)
stall_min=999
if [ -n "$newest" ]; then
    mtime=$(stat -f %m "$newest" 2>/dev/null)
    case "$mtime" in ''|*[!0-9]*) ;; *) stall_min=$(( ( $(date +%s) - mtime ) / 60 ));; esac
fi

for pid in $(pgrep -f "ailang eval-suite" 2>/dev/null); do
    ppid=$(ps -o ppid= -p "$pid" 2>/dev/null | tr -d ' ')
    pcmd=$(ps -o command= -p "$ppid" 2>/dev/null)
    case "$pcmd" in *os-rotation-filler*) ;; *) continue ;; esac   # ONLY rotation chunks
    secs=$(etime_secs "$pid")
    case "$secs" in ''|*[!0-9]*) secs=0;; esac   # never let a bad parse crash the compare
    reason=""
    if [ "$secs" -gt "$HARD_SECS" ]; then reason="hard-max (${secs}s alive)"
    elif [ "$secs" -gt "$SOFT_SECS" ] && [ "$stall_min" -gt "$STALL_MIN" ]; then
        reason="no-progress (${stall_min}m since last session write, ${secs}s alive)"; fi
    if [ -n "$reason" ]; then
        pgid=$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ')
        echo "${TIMESTAMP} [WATCHDOG] WEDGED rotation chunk pid $pid — $reason — killing pgroup $pgid"
        { [ -n "$pgid" ] && kill -TERM "-$pgid"; } 2>/dev/null || kill -TERM "$pid" 2>/dev/null
    fi
done

for zp in $(lsof -ti :8080 2>/dev/null); do
    zpp=$(ps -o ppid= -p "$zp" 2>/dev/null | tr -d ' ')
    if [ -z "$zpp" ] || [ "$zpp" = "1" ] || ! ps -p "$zpp" >/dev/null 2>&1; then
        echo "${TIMESTAMP} [WATCHDOG] orphaned :8080 listener pid $zp (parent ${zpp:-gone}) — killing"
        kill -TERM "$zp" 2>/dev/null
    fi
done

# Exit 0 always — the next tick will re-check. Non-zero exit would make
# launchd consider the watchdog itself broken.
exit 0
