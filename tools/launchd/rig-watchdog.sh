#!/usr/bin/env bash
# rig-watchdog.sh — Poll-based reliability backstop for the local-Ollama
# eval rig. Called every 60s by dev.ailang.rig-watchdog.plist.
#
# Checks three services:
#   - ollama serve at http://127.0.0.1:11435/api/tags (its private port)
#   - the rig GPU gateway (dev.ailang.rig-gate) at 127.0.0.1:11434, ollama's
#     usual address, when that job is loaded (M-RIG-GPU-ADMISSION-GATEWAY)
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

# Check ollama. The probe address is PINNED to 127.0.0.1 — it must name the same
# server this block restarts. dev.ollama.serve binds OLLAMA_HOST=127.0.0.1:11435,
# but "localhost" is dual-stack and resolves ::1 first, so the probe and the
# kickstart target can be DIFFERENT servers. Observed 2026-07-21..08-03 (#557):
# a second, GUI-launched `ollama serve` held [::1]:11434 for 13 days, so this
# watchdog probed the app's server while restarting launchd's — meaning a dead
# dev.ollama.serve would never have been noticed. Do not relax this to localhost.
# The probe goes to ollama's PRIVATE port, directly: through the gateway, a dead
# ollama would look alive whenever the gateway answered.
if ! curl --max-time 2 -s http://127.0.0.1:11435/api/tags >/dev/null 2>&1; then
    echo "${TIMESTAMP} [WATCHDOG] ollama unreachable — kickstart dev.ollama.serve"
    launchctl kickstart "gui/${UID_NUMBER}/dev.ollama.serve" 2>&1
fi

# Check the rig gateway. It answers every request itself (a 502 when ollama is
# down, a 423 when it refuses), so ANY HTTP status means it is alive; only "000"
# (nothing listening) means it is dead. There is no fallback to direct ollama:
# a dead gateway is restarted, never bypassed (design doc D4).
if launchctl print "gui/${UID_NUMBER}/dev.ailang.rig-gate" >/dev/null 2>&1; then
    gate_code=$(curl --max-time 2 -s -o /dev/null -w '%{http_code}' http://127.0.0.1:11434/api/version 2>/dev/null)
    if [ "${gate_code:-000}" = "000" ]; then
        echo "${TIMESTAMP} [WATCHDOG] rig gateway unreachable — kickstart dev.ailang.rig-gate"
        launchctl kickstart "gui/${UID_NUMBER}/dev.ailang.rig-gate" 2>&1
    fi
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

# descendants PID — every descendant of PID, deepest first. Must be walked BEFORE any
# kill: once a parent dies its children are reparented to launchd and the tree is gone.
descendants() {
    local c
    for c in $(pgrep -P "${1:-}" 2>/dev/null); do descendants "$c"; echo "$c"; done
}

# kill_tree PID — TERM the whole process tree under PID, not just PID's group.
#
# Killing only the chunk's process group LEAKED the GPU. opencode (and pi, via
# proctree.Configure) start in their OWN process group, so `kill -TERM -$pgid` left
# them running: reparented to launchd, still streaming prompts at the 27B model with
# no rig lock held. Measured 2026-09-27: two such `opencode run` orphans, 15h and 18h
# old, born at the 00:26 and 03:11 wedge-kills below. With OLLAMA_NUM_PARALLEL=1 every
# real eval queued behind them — 92 chat requests >15m that day, nightly 0/10 INVALID.
# So collect every descendant, kill each one's group (catching grandchildren we did not
# enumerate) and each pid, and never our own group.
kill_tree() {
    local root="$1" own p g groups=""
    own=$(ps -o pgid= -p $$ 2>/dev/null | tr -d ' ')
    for p in $(descendants "$root") "$root"; do
        g=$(ps -o pgid= -p "$p" 2>/dev/null | tr -d ' ')
        case " $groups " in *" $g "*) ;; *) [ -n "$g" ] && groups="$groups $g" ;; esac
        kill -TERM "$p" 2>/dev/null
    done
    for g in $groups; do
        [ "$g" = "$own" ] || [ "$g" = "1" ] || kill -TERM "-$g" 2>/dev/null
    done
    echo "$groups"
}

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
        groups=$(kill_tree "$pid")
        echo "${TIMESTAMP} [WATCHDOG] WEDGED rotation chunk pid $pid — $reason — killed tree (pgroups:${groups})"
    fi
done

# ── ORPHANED GPU CLIENTS (opt-in: RIG_WATCHDOG_REAP_ORPHANS=1) ────────────────────────
# Backstop for trees kill_tree never saw: an eval-suite SIGKILLed by someone else, or a
# crash, leaves its agent reparented to launchd (PPID 1) and still talking to ollama.
# Selected by what matters, a live connection to ollama: a process holding a socket to
# 127.0.0.1:11434, with PPID 1, whose binary is an agent CLI, older than the grace
# period. NOT by command line — pi rewrites its process title to a bare "pi" (measured
# 2026-09-27: `ps -o command=` shows no arguments), so an argv match never sees it.
# A live eval's agent always has its eval-suite as parent, a human's session has a
# shell, and short-lived hooks (`ailang cache put-resolution`, PPID 1, embeds) are not
# agent CLIs. Opt-in because it acts on processes this watchdog did not start.
if [ "${RIG_WATCHDOG_REAP_ORPHANS:-0}" = "1" ]; then
    ORPHAN_GRACE=$(( ${RIG_WATCHDOG_ORPHAN_GRACE_MIN:-10} * 60 ))
    for op in $(lsof -t -nP -iTCP@127.0.0.1:11434 -sTCP:ESTABLISHED 2>/dev/null | sort -u); do
        [ "$(ps -o ppid= -p "$op" 2>/dev/null | tr -d ' ')" = "1" ] || continue
        ocomm=$(basename "$(ps -o comm= -p "$op" 2>/dev/null)" 2>/dev/null)
        case "$ocomm" in opencode|pi|node) ;; *) continue ;; esac
        ocmd=$(ps -o command= -p "$op" 2>/dev/null)
        osecs=$(etime_secs "$op")
        case "$osecs" in ''|*[!0-9]*) continue ;; esac
        [ "$osecs" -gt "$ORPHAN_GRACE" ] || continue
        groups=$(kill_tree "$op")
        echo "${TIMESTAMP} [WATCHDOG] ORPHANED GPU client pid $op (PPID 1, ${osecs}s) — killed tree (pgroups:${groups}): $(printf '%s' "$ocmd" | cut -c1-80)"
    done
fi

for zp in $(lsof -ti :8080 2>/dev/null); do
    zpp=$(ps -o ppid= -p "$zp" 2>/dev/null | tr -d ' ')
    if [ -z "$zpp" ] || [ "$zpp" = "1" ] || ! ps -p "$zpp" >/dev/null 2>&1; then
        echo "${TIMESTAMP} [WATCHDOG] orphaned :8080 listener pid $zp (parent ${zpp:-gone}) — killing"
        kill -TERM "$zp" 2>/dev/null
    fi
done

# --- MISSION-JOB RE-BOOTSTRAP (2026-08-18) ---------------------------------
# The scheduled ailang jobs VANISH from the gui/<uid> domain — not disabled, not
# crashed: absent, while their plists sit untouched in ~/Library/LaunchAgents.
# Observed twice: 2026-08-17 (loops silent ~12h, discovered by hand) and
# 2026-08-18 (again, ~4.5h silent, V1 and world each missing fires). Both times
# the SAME seven vanished and dev.ailang.rig-watchdog — this job — survived.
#
# The cause is UNKNOWN and deliberately not guessed at here. It could not be
# identified from the unified log: `log show` returns zero records to this
# context even for the last 2 minutes despite a 1.1G store, so the instrument is
# blind rather than the event absent. Nothing in this repo calls `launchctl
# bootout`/`unload` on a mission label (grepped).
#
# So this is recovery, not a fix, and it is the same bet the ollama block above
# makes: on this box launchd's own guarantees are not dependable, and a 60s
# poller that survives is worth more than a correct theory. If the root cause is
# later found, DELETE this block rather than leaving two mechanisms.
#
# Re-bootstrap is idempotent: an already-loaded label makes `bootstrap` fail
# harmlessly (EALREADY), so the common path is silent and only genuine restores
# log. RunAtLoad=true on the three mission jobs means a restore also FIRES that
# mission — which is the point, since a vanished job has been missing fires.
for label in mission-control mission-motoko mission-world \
             mission-recovery mission-recovery-motoko \
             nightly-eval os-rotation-filler; do
    full="dev.ailang.${label}"
    plist="$HOME/Library/LaunchAgents/${full}.plist"
    [ -f "$plist" ] || continue
    # HOLD = a SANCTIONED stop. Without this, re-bootstrap fights any operator or
    # agent who stopped a job ON PURPOSE — it would resurrect it inside 60s, which
    # is worse than the drift this block exists to fix. Matters most for the GPU
    # consumers: os-rotation-filler fires every 2700s and takes rig.lock, so
    # reviving it mid-eval starts a SECOND job against a device already holding a
    # 33.5GB model. Two LLMs resident is the documented OOM shape that panicked the
    # rig on 2026-07-20. Cloud-only jobs (the missions) never touch the GPU, but
    # they honour the hold too so there is ONE idiom rather than a special case.
    #   hold:    mkdir -p ~/.ailang/state/launchd-hold && \
    #            echo "why + who" > ~/.ailang/state/launchd-hold/dev.ailang.os-rotation-filler
    #   release: rm ~/.ailang/state/launchd-hold/dev.ailang.os-rotation-filler
    # A hold does NOT stop a loaded job — bootout separately. It only stops this
    # watchdog putting it back.
    hold="$HOME/.ailang/state/launchd-hold/${full}"
    if [ -f "$hold" ]; then
        echo "${TIMESTAMP} [WATCHDOG] ${full} on HOLD — not re-bootstrapping ($(head -c 120 "$hold" 2>/dev/null | tr '\n' ' '))"
        continue
    fi
    # Match the LABEL COLUMN of `launchctl list` (tab-separated: PID STATUS LABEL).
    # A substring grep would false-positive: "mission-recovery" is a prefix of
    # "mission-recovery-motoko", so the shorter label would read as loaded
    # whenever only the longer one is — silently skipping the restore.
    if ! launchctl list | awk '{print $3}' | grep -qx "$full"; then
        if launchctl bootstrap "gui/${UID_NUMBER}" "$plist" 2>/dev/null; then
            echo "${TIMESTAMP} [WATCHDOG] ${full} was ABSENT from gui/${UID_NUMBER} — re-bootstrapped"
        else
            echo "${TIMESTAMP} [WATCHDOG] ${full} absent and re-bootstrap FAILED — needs a human"
        fi
    fi
done

# Exit 0 always — the next tick will re-check. Non-zero exit would make
# launchd consider the watchdog itself broken.
exit 0
