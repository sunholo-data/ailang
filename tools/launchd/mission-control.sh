#!/usr/bin/env bash
# mission-control.sh — continuous outer-loop iterations for the V1 mission.
#
# Fires a headless Claude (Fable) session that runs the mission-control skill:
# observe mission state → pick top backlog item → route through the inner-loop
# skills (design-doc → sprint-plan → execute → evaluate) → record → retro.
# See design_docs/v1-mission.md for the charter and guardrails.
#
# Scheduled via launchd StartInterval every 2h (see the plist); the overlap
# guard below makes this effectively "back-to-back iterations, ≤2h idle gap".
# Iterations are cloud-model work: they NEVER take rig.lock (GPU mutex only —
# GPU-touching sprint steps take it per-step inside the session).
#
# Kill switch: touch ~/.ailang/state/mission-control.disabled
# Portable to macOS bash 3.2. No GNU timeout on this rig → bash watchdog below.
set -uo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO" || exit 1

# launchd PATH is restricted; claude lives in ~/.local/bin, go tools in ~/go/bin.
export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"

# BILLING GUARD (2026-07-10): the mission MUST bill the Claude subscription,
# never API credits. secrets.env exports ANTHROPIC_API_KEY for other tools —
# strip it so claude's only auth paths are subscription ones (keychain OAuth,
# or CLAUDE_CODE_OAUTH_TOKEN if set). Subscription-or-nothing by construction.
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN

LOG=/tmp/ailang-mission-control.log
log() { echo "[$(date '+%F %H:%M:%S')] $*" | tee -a "$LOG"; }

# Controller model. Steady state is Fable. A time-boxed override file lets us
# borrow a different model (e.g. Opus during a Fable-quota window) and REVERT
# AUTOMATICALLY when it expires — no session or human needed. Format of
# ~/.ailang/state/mission-model:  "<model> <expiry-epoch>". Past expiry, the
# next iteration deletes it and falls back to the default, notifying #329.
DEFAULT_MODEL="claude-fable-5"
OVERRIDE_FILE="$HOME/.ailang/state/mission-model"
MODEL="${MISSION_MODEL:-$DEFAULT_MODEL}"
if [ -z "${MISSION_MODEL:-}" ] && [ -f "$OVERRIDE_FILE" ]; then
  read -r OV_MODEL OV_UNTIL < "$OVERRIDE_FILE" 2>/dev/null || true
  NOW=$(date +%s)
  if [ -n "${OV_UNTIL:-}" ] && [ "$NOW" -ge "$OV_UNTIL" ]; then
    rm -f "$OVERRIDE_FILE"
    echo "[$(date '+%F %H:%M:%S')] model override expired — reverting to $DEFAULT_MODEL" | tee -a /tmp/ailang-mission-control.log
    gh issue comment "${MISSION_GH_ISSUE:-329}" --repo sunholo-data/ailang \
      --body "🔁 Controller model override expired — the loop is back on **$DEFAULT_MODEL** (Fable) as of $(date '+%F %H:%M %Z'). Automatic revert, no action taken." 2>/dev/null || true
  elif [ -n "${OV_MODEL:-}" ]; then
    MODEL="$OV_MODEL"
  fi
fi
HARD_TIMEOUT="${MISSION_TIMEOUT:-21600}"   # 6h wall-clock kill per iteration
KILL_SWITCH="$HOME/.ailang/state/mission-control.disabled"

# 1. Kill switch — the intended "off" state, exit silently.
if [ -f "$KILL_SWITCH" ]; then
  log "kill switch present ($KILL_SWITCH) — skip"; exit 0
fi

# 1b. ONE iteration at a time (2026-07-10, continuous mode): two concurrent
#     controllers would stomp the charter/log in the main tree and could pick
#     the same queue item. If one is still running, yield this slot.
if pgrep -f "claude -p Run one mission" >/dev/null 2>&1; then
  log "previous iteration still running — yield (next interval retries)"; exit 0
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
      --title "Mission iteration blocked: auth probe failed" --from mission-control 2>/dev/null
    gh issue comment "${MISSION_GH_ISSUE:-329}" --repo sunholo-data/ailang \
      --body "⚠️ Mission iteration did not start: subscription auth probe failed on the rig (API keys are deliberately stripped; keychain likely unavailable). Zero tokens spent. Will retry next interval." 2>/dev/null
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
design_docs/v1-mission.md and follow its gates. You are a scheduled run; \
there is no human present — park anything needing human input and report via \
ailang messages and the GitHub bookkeeping issue, per the skill."

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
    --body "⚠️ Mission iteration **FAILED to complete** (rc=$RC — timeout or crash) at $(date '+%F %H:%M %Z'). Log on the rig: \`$LOG\`. The queue is untouched; the next interval will retry." 2>/dev/null
else
  log "iteration complete (rc=0)"
  # The skill itself sends the substantive report (Gate 5, both channels).
fi
exit "$RC"
