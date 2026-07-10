#!/usr/bin/env bash
# mission-control.sh — nightly outer-loop iteration for the V1 mission.
#
# Fires a headless Claude (Fable) session that runs the mission-control skill:
# observe mission state → pick top backlog item → route through the inner-loop
# skills (design-doc → sprint-plan → execute → evaluate) → record → retro.
# See design_docs/v1-mission.md for the charter and guardrails.
#
# Deliberately does NOT take the rig lock: default iterations are cloud-model
# work (no GPU). rig.lock is a GPU mutex only — GPU-touching sprint steps take
# it themselves, per-step, inside the session (two-tier rule in the charter).
#
# Schedule via launchd (see dev.ailang.mission-control.plist). Kill switch:
#   touch ~/.ailang/state/mission-control.disabled
# Portable to macOS bash 3.2. No GNU timeout on this rig → bash watchdog below.
set -uo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO" || exit 1

# launchd PATH is restricted; claude lives in ~/.local/bin, go tools in ~/go/bin.
export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"

# BILLING GUARD (2026-07-10): the nightly MUST bill the Claude subscription, never
# API credits. secrets.env exports ANTHROPIC_API_KEY for other tools — if it leaks
# into claude's env, headless -p silently bills the API. Strip it, and require the
# subscription token (one-time: `claude setup-token`, then put the sk-ant-oat...
# value in secrets.env as CLAUDE_CODE_OAUTH_TOKEN). No token → refuse to run.
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN

LOG=/tmp/ailang-mission-control.log
log() { echo "[$(date '+%F %H:%M:%S')] $*" | tee -a "$LOG"; }

MODEL="${MISSION_MODEL:-claude-fable-5}"
HARD_TIMEOUT="${MISSION_TIMEOUT:-21600}"   # 6h wall-clock kill
KILL_SWITCH="$HOME/.ailang/state/mission-control.disabled"

# 1. Kill switch — the intended "off" state, exit silently.
if [ -f "$KILL_SWITCH" ]; then
  log "kill switch present ($KILL_SWITCH) — skip"; exit 0
fi

# 2. Subscription auth available? With API keys stripped above, claude can only
#    bill the subscription — via CLAUDE_CODE_OAUTH_TOKEN if set, else the Keychain
#    OAuth (works because this rig stays logged in; verified 2026-07-10: probe
#    succeeds from this context with the key stripped). Probe cheaply before the
#    real run so a locked keychain (e.g. rig at login screen post-reboot) fails
#    loudly instead of wasting the iteration slot.
if [ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ] && [ "${MISSION_DRY_RUN:-0}" != "1" ]; then
  if ! claude -p 'reply with exactly: ok' --model claude-haiku-4-5-20251001 >/dev/null 2>&1; then
    log "auth probe FAILED with API keys stripped — no subscription auth available"
    log "(keychain locked? rig at login screen?) Refusing to run."
    ailang messages send controlplane \
      "mission-control refused to start: no subscription auth (keychain probe failed, API keys deliberately stripped). Is the rig logged in? Optionally set CLAUDE_CODE_OAUTH_TOKEN in secrets.env." \
      --title "Mission nightly blocked: auth probe failed" --from mission-control 2>/dev/null
    gh issue comment "${MISSION_GH_ISSUE:-329}" --repo sunholo-data/ailang \
      --body "⚠️ Nightly mission iteration did not start: subscription auth probe failed on the rig (API keys are deliberately stripped; keychain likely unavailable). Zero tokens spent. Will retry next schedule." 2>/dev/null
    exit 1
  fi
  log "auth probe ok (subscription via keychain OAuth)"
fi

# 3. claude CLI reachable?
if ! command -v claude >/dev/null 2>&1; then
  log "claude CLI not on PATH — abort"
  ailang messages send controlplane "mission-control driver: claude CLI not found on PATH" \
    --title "Mission iteration FAILED to start" --from mission-control 2>/dev/null
  exit 1
fi

# 4. Dry run — verify wiring without spending tokens.
if [ "${MISSION_DRY_RUN:-0}" = "1" ]; then
  log "DRY RUN ok: repo=$REPO model=$MODEL timeout=${HARD_TIMEOUT}s"; exit 0
fi

log "=== mission iteration starting (model=$MODEL, timeout=${HARD_TIMEOUT}s) ==="

PROMPT="Run one mission-control iteration: invoke the mission-control skill for \
design_docs/v1-mission.md and follow its gates. You are the nightly scheduled run; \
there is no human present — park anything needing human input and report via \
ailang messages, per the skill."

claude -p "$PROMPT" \
  --model "$MODEL" \
  --permission-mode bypassPermissions \
  >>"$LOG" 2>&1 &
CLAUDE_PID=$!

# Watchdog: TERM at the wall limit, KILL 60s later. (No GNU timeout on macOS.)
(
  sleep "$HARD_TIMEOUT"
  if kill -0 "$CLAUDE_PID" 2>/dev/null; then
    echo "[$(date '+%F %H:%M:%S')] HARD TIMEOUT ${HARD_TIMEOUT}s — killing $CLAUDE_PID" >>"$LOG"
    kill -TERM "$CLAUDE_PID" 2>/dev/null; sleep 60; kill -KILL "$CLAUDE_PID" 2>/dev/null
  fi
) &
WATCHDOG_PID=$!

wait "$CLAUDE_PID"; RC=$?
kill "$WATCHDOG_PID" 2>/dev/null

if [ "$RC" -ne 0 ]; then
  log "iteration exited rc=$RC"
  ailang messages send controlplane \
    "mission-control iteration exited rc=$RC (timeout or crash). Log: $LOG" \
    --title "Mission iteration FAILED (rc=$RC)" --from mission-control 2>/dev/null
  gh issue comment "${MISSION_GH_ISSUE:-329}" --repo sunholo-data/ailang \
    --body "⚠️ Nightly mission iteration **FAILED to complete** (rc=$RC — timeout or crash) at $(date '+%F %H:%M %Z'). Log on the rig: \`$LOG\`. The queue is untouched; next scheduled run will retry." 2>/dev/null
else
  log "iteration complete (rc=0)"
  # The skill itself sends the substantive morning report (Gate 5).
fi
exit "$RC"
