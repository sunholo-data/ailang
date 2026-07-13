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

# --- stall detection (see the stall watchdog below) -------------------------
# _mc_descendants PID → echoes PID and every descendant PID (one per line).
_mc_descendants() {
  local pid="$1"; echo "$pid"
  local kids k; kids=$(pgrep -P "$pid" 2>/dev/null)
  for k in $kids; do _mc_descendants "$k"; done
}
# _mc_etime_secs "[[DD-]HH:]MM:SS" → seconds (macOS ps has no `etimes`).
_mc_etime_secs() {
  local t="${1// /}" dd=0 hh=0 mm=0 ss=0 rest nf
  [ -n "$t" ] || { echo 0; return; }
  case "$t" in *-*) dd=${t%%-*}; rest=${t#*-} ;; *) rest="$t" ;; esac
  nf=$(( $(printf '%s' "$rest" | tr -cd ':' | wc -c) + 1 ))
  if [ "$nf" -ge 3 ]; then hh=${rest%%:*}; rest=${rest#*:}; fi
  mm=${rest%%:*}; ss=${rest##*:}
  echo $(( 10#${dd:-0}*86400 + 10#${hh:-0}*3600 + 10#${mm:-0}*60 + 10#${ss:-0} ))
}
# _mc_stalled PID → true when the tree is IDLE (<2% CPU across the tree) AND has
# a descendant that has itself been alive ≥ STALL_CHILD_AGE. That pair is the
# fingerprint of a wedged tool call (iteration 13: a `until …; do sleep 30; done`
# whose zsh child sat alive 4h+ at 0% CPU). We key on the LONG-LIVED CHILD, not a
# live `sleep` — after hours of polling `gh` is rate-limited/slow, so a `sleep`
# descendant is only intermittently present and a naive sleep-catcher misses it.
# Errs safe: macOS `ps %cpu` is a lifetime-decaying average, so a session doing
# real work reads non-idle and is NOT flagged (we miss late stalls, never kill
# live work); and STALL_CHILD_AGE is set past the skill's 30-min bounded-wait cap
# so a COMPLIANT wait can never trip it.
_mc_stalled() {
  local root="$1" pids p secs cpu long=0
  pids=$(_mc_descendants "$root")
  for p in $pids; do
    [ "$p" = "$root" ] && continue
    secs=$(_mc_etime_secs "$(ps -o etime= -p "$p" 2>/dev/null)")
    [ "${secs:-0}" -ge "${STALL_CHILD_AGE:-2400}" ] && { long=1; break; }
  done
  [ "$long" -eq 1 ] || return 1
  cpu=$(ps -o %cpu= -p "$(echo $pids | tr ' ' ',')" 2>/dev/null | awk '{s+=$1} END{printf "%d", s+0}')
  [ "${cpu:-0}" -lt 2 ] || return 1
  return 0
}
# ----------------------------------------------------------------------------

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
# Stall watchdog (2026-07-12): a wedged unbounded poll loop (iteration 13's
# `until COND; do sleep 30; done`) otherwise burns the whole 6h slot before
# HARD_TIMEOUT. Kill early once the session is IDLE (<2% CPU) with a descendant
# that has itself been alive ≥ STALL_CHILD_AGE — a wedged tool call. Both the
# grace and the child-age gate sit past the skill's 30-min bounded-wait cap so a
# COMPLIANT wait can never trip it. All env-overridable.
STALL_GRACE="${MISSION_STALL_GRACE:-2400}"       # 40m before the first check
STALL_CHILD_AGE="${MISSION_STALL_CHILD_AGE:-2400}" # a descendant alive ≥40m = wedged
STALL_INTERVAL="${MISSION_STALL_INTERVAL:-120}"  # 2m between samples
STALL_SAMPLES="${MISSION_STALL_SAMPLES:-3}"      # consecutive idle+long-child hits → kill
export STALL_CHILD_AGE
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

# Stall watchdog: after the grace window, sample for the wedged-tool fingerprint
# (idle tree + a descendant alive ≥ STALL_CHILD_AGE). STALL_SAMPLES consecutive
# hits → kill early so the slot recycles instead of idling to HARD_TIMEOUT. hits
# resets on any non-idle/no-long-child sample, so live work is never killed.
(
  sleep "$STALL_GRACE"
  hits=0
  while kill -0 "$CLAUDE_PID" 2>/dev/null; do
    if _mc_stalled "$CLAUDE_PID"; then hits=$((hits + 1)); else hits=0; fi
    if [ "$hits" -ge "$STALL_SAMPLES" ]; then
      echo "[$(date '+%F %H:%M:%S')] STALL: claude $CLAUDE_PID idle with a descendant alive ≥${STALL_CHILD_AGE}s across $STALL_SAMPLES samples (unbounded poll loop?) — killing early" >>"$LOG"
      kill -TERM "$CLAUDE_PID" 2>/dev/null; sleep 30; kill -KILL "$CLAUDE_PID" 2>/dev/null
      break
    fi
    sleep "$STALL_INTERVAL"
  done
) &
STALL_PID=$!

wait "$CLAUDE_PID"; RC=$?
kill "$WATCHDOG_PID" "$STALL_PID" 2>/dev/null

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
