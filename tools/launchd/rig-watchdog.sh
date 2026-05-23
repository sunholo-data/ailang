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

# Exit 0 always — the next tick will re-check. Non-zero exit would make
# launchd consider the watchdog itself broken.
exit 0
