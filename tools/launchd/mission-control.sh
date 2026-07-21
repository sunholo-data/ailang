#!/usr/bin/env bash
# mission-control.sh — continuous outer-loop iterations for ONE mission (default: V1).
# PORTABLE (M-MISSION-PORTABILITY M1, 2026-07-21, attended): MISSION_PROFILE=<name>
# sources ~/.config/ailang/mission-<name>.env (MISSION_NAME/REPO/DOC/...). MISSION_NAME=v1
# (the default) keeps the LEGACY state paths + log name EXACTLY as before — bit-for-bit,
# no migration; any other name gets fully namespaced state so two missions never collide.
#
# Fires a headless Claude session that runs the mission-control skill:
# observe mission state → pick top backlog item → route through the inner-loop
# skills (design-doc → sprint-plan → execute → evaluate) → record → retro.
# See design_docs/v1-mission.md for the charter and guardrails.
#
# Scheduled via launchd StartInterval every 2h (see the plist); the overlap
# guard below makes this effectively "back-to-back iterations, ≤2h idle gap".
# Iterations are cloud-model work: they NEVER take rig.lock (GPU mutex only —
# GPU-touching sprint steps take it per-step inside the session).
#
# MODEL SELECTION (fleet Phase A, 2026-07-14): ordered preference probing.
# MISSION_MODEL_PREFS (default "claude-opus-4-8,claude-fable-5" — OPUS-FIRST
# since 2026-07-16, Mark: Fable is reserved for high-cognition ROLES — design
# synthesis + evaluation, both bounded pinned sub-agents — never the long
# orchestration session, which burned the weekly Fable bucket at 2h cadence)
# is walked each iteration with a 1-token probe; first usable model wins. A
# quota-limited probe falls through to the next candidate; transient errors
# retry once. Fable last = emergency fallback only (a controller on Fable
# beats no controller). Semantics of the ordered list follow
# internal/ai/routing.go AIRoutingPolicy.Order (the third-vocabulary rule in
# m-mission-adaptive-multiprovider-routing); it lives in bash because the
# driver must select BEFORE any Go/claude process exists.
# Manual pins still win: MISSION_MODEL env (absolute) or
# ~/.ailang/state/mission-model ("<model> [expiry-epoch]", auto-expires).
#
# Transient Anthropic errors (Overloaded/dropped socket) are retried with backoff
# (TRANSIENT-RETRY block); deliberate watchdog kills are not.
# Kill switch: touch ~/.ailang/state/mission-control.disabled
# Portable to macOS bash 3.2. No GNU timeout on this rig → bash watchdog below.
set -uo pipefail

REPO="${MISSION_WORKDIR:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$REPO" || exit 1

# launchd PATH is restricted; claude lives in ~/.local/bin, go tools in ~/go/bin.
export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"

# --- MISSION PROFILE + STATE NAMESPACE (M1, 2026-07-21) ----------------------
[ -n "${MISSION_PROFILE:-}" ] && [ -f "$HOME/.config/ailang/mission-${MISSION_PROFILE}.env" ] \
  && . "$HOME/.config/ailang/mission-${MISSION_PROFILE}.env"
MISSION_NAME="${MISSION_NAME:-v1}"
MISSION_REPO="${MISSION_REPO:-sunholo-data/ailang}"
MISSION_DOC="${MISSION_DOC:-design_docs/v1-mission.md}"
export MISSION_NAME MISSION_REPO MISSION_DOC
STATE_DIR="$HOME/.ailang/state"
if [ "$MISSION_NAME" = "v1" ]; then
  # LEGACY paths — bit-for-bit compat with the live V1 loop (no migration).
  LOG=/tmp/ailang-mission-control.log
  KILL_SWITCH="$STATE_DIR/mission-control.disabled"
  PIDFILE="$STATE_DIR/mission-control.pid"
  OVERRIDE_FILE="$STATE_DIR/mission-model"
  LAST_MODEL_FILE="$STATE_DIR/mission-model-last"
  EXEC_ONCE_FILE="$STATE_DIR/mission-executor-model-once"
  GH_ISSUE_FILE="$STATE_DIR/mission-gh-issue"
  MSG_FROM="mission-control"
else
  LOG="/tmp/ailang-mission-${MISSION_NAME}.log"
  KILL_SWITCH="$STATE_DIR/mission-${MISSION_NAME}.disabled"
  PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"
  OVERRIDE_FILE="$STATE_DIR/mission-${MISSION_NAME}-model"
  LAST_MODEL_FILE="$STATE_DIR/mission-${MISSION_NAME}-model-last"
  EXEC_ONCE_FILE="$STATE_DIR/mission-${MISSION_NAME}-executor-model-once"
  GH_ISSUE_FILE="$STATE_DIR/mission-${MISSION_NAME}-gh-issue"
  MSG_FROM="mission-${MISSION_NAME}"
fi
# -----------------------------------------------------------------------------
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"

# BILLING GUARD (2026-07-10): the mission MUST bill the Claude subscription,
# never API credits. secrets.env exports ANTHROPIC_API_KEY for other tools —
# strip it so claude's only auth paths are subscription ones (keychain OAuth,
# or CLAUDE_CODE_OAUTH_TOKEN if set). Subscription-or-nothing by construction.
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN

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

# --- model selection (fleet Phase A) -----------------------------------------
PREFS="${MISSION_MODEL_PREFS:-claude-opus-4-8,claude-fable-5}"
QUOTA_SIG="usage limit|rate.?limit|quota|exceeded|too many requests|weekly limit"

# _mc_probe MODEL → 0 usable | 1 quota-limited | 2 unusable (auth/transient×2)
_mc_probe() {
  local m="$1" out rc
  out=$(claude -p 'reply with exactly: ok' --model "$m" 2>&1); rc=$?
  [ "$rc" -eq 0 ] && return 0
  if printf '%s' "$out" | grep -qiE "$QUOTA_SIG"; then return 1; fi
  # transient? retry once
  sleep 5
  out=$(claude -p 'reply with exactly: ok' --model "$m" 2>&1); rc=$?
  [ "$rc" -eq 0 ] && return 0
  printf '%s' "$out" | grep -qiE "$QUOTA_SIG" && return 1
  return 2
}

select_model() {
  # 1. absolute pin
  if [ -n "${MISSION_MODEL:-}" ]; then MODEL="$MISSION_MODEL"; MODEL_WHY="env pin"; return 0; fi
  # 2. override file pin (optional expiry epoch)
  if [ -f "$OVERRIDE_FILE" ]; then
    local ov_model ov_until now
    read -r ov_model ov_until < "$OVERRIDE_FILE" 2>/dev/null || true
    now=$(date +%s)
    if [ -n "${ov_until:-}" ] && [ "$now" -ge "${ov_until:-0}" ]; then
      rm -f "$OVERRIDE_FILE"
      log "model override expired — resuming preference probing"
    elif [ -n "${ov_model:-}" ]; then
      MODEL="$ov_model"; MODEL_WHY="override file"; return 0
    fi
  fi
  # 3. ordered preference probing
  local m why rcode
  for m in $(printf '%s' "$PREFS" | tr ',' ' '); do
    _mc_probe "$m"; rcode=$?
    case "$rcode" in
      0) MODEL="$m"; MODEL_WHY="probe ok"; return 0 ;;
      1) log "model $m quota-limited — falling through" ;;
      2) log "model $m unusable (auth/transient) — falling through" ;;
    esac
  done
  return 1
}
# ----------------------------------------------------------------------------

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

# TRANSIENT-RETRY (2026-07-14): Anthropic capacity is flaky some evenings —
# `claude -p` does its own internal retries then exits rc=1 on a persistent
# "API Error: Overloaded" / dropped socket, losing the whole iteration (2 lost
# 2026-07-14). Retry the run on a TIGHTLY-ANCHORED transient signature (claude's
# own "API Error:" emissions + socket-closed), with backoff. NEVER retried:
# watchdog kills (rc 143/137 = deliberate stall/timeout), quota/429 (that's
# Phase A's start-probe fall-through job, not a same-model retry), or any other
# genuine rc. Signature is anchored so an unrelated "503" in a test's output
# (e.g. the httpbin fixture) cannot trigger a false retry.
TRANSIENT_RETRIES="${MISSION_TRANSIENT_RETRIES:-3}"   # total attempts incl. the first
TRANSIENT_BACKOFF="${MISSION_TRANSIENT_BACKOFF:-45}"  # base seconds, ×attempt (45s,90s)
TRANSIENT_SIG="API Error: Overloaded|socket connection was closed|overloaded_error|API Error: 5[0-9][0-9]|API Error: Internal|API Error: Connection|API Error: Request timed out"

# PER-ROLE MODEL ROUTING (2026-07-15, m-mission-agentic-provider-routing M1): the charter's routing
# table was never enforced — every inner role ran on the controller's single session --model, so with
# the driver on Fable 100% of each iteration billed Fable (memory:
# project-mission-routing-table-never-enforced). Fix: the controller session keeps $MODEL; the HEAVY
# roles are spawned by mission-control Gate 3 as model-PINNED sub-agents that read these env vars.
# Defaults track the charter routing table; M3 will A/B the planner down-tier — keep it at the proven
# Opus until there's evidence. Cross-provider AGENT executors (codex/motoko) ride the same env once
# fleet Phase C wires them into the spawn (a value like "codex:gpt-5.6" is resolved by the skill).
# 2026-07-16 (Mark): Fable = high-cognition ROLES only. The controller session is opus-first (see
# PREFS above); Fable bills exactly two BOUNDED pinned sub-agents per iteration: the designer
# (deep spec synthesis, fired only when a new doc is needed) and the evaluator (adversarial judge,
# ≠ the opus executor → generator≠judge holds).
# NB: these are in-session Agent/Task-tool model ALIASES (opus|fable|sonnet|haiku) — NOT the full
# IDs (claude-opus-4-8) the driver's own `claude -p --model` flag takes. Two different interfaces:
# the controller session is launched with a full ID; the sub-agents it spawns are pinned by alias.
# A "provider:model" value (e.g. codex:gpt-5.6-sol) instead signals cross-provider agent routing via
# provider_executor (fleet Phase C), which the skill resolves — not the Agent tool.
# MISSION_EXECUTOR_MODEL specifically accepts EITHER form: an Agent alias (opus, the default) OR a
# provider:model value (codex:gpt-5.6-sol) which the mission-control Gate-3 recipe routes to a bounded
# `codex exec` run in the sprint worktree (M1b). The default stays "opus" — codex is OPT-IN per
# iteration via an env override, so a normal fire never bills metered OpenAI $ by accident.
# Weekly rolling bookkeeping issue (2026-07-16, Mark): the issue number lives in a state file so
# the skill's Monday-07:00 rotation (aligned to the quota reset) moves threads without a driver
# edit. Precedence: env pin > state file > 329 (the original thread). Exported so the skill's
# gh snippets see the same number the driver reports to.
MISSION_GH_ISSUE="${MISSION_GH_ISSUE:-$(head -1 "$GH_ISSUE_FILE" 2>/dev/null)}"
[ -z "${MISSION_GH_ISSUE:-}" ] && [ "$MISSION_NAME" = "v1" ] && MISSION_GH_ISSUE=329
export MISSION_GH_ISSUE

# designer default is the claude-CLI lane (claude:<full-id>), NOT the bare "fable" alias: the
# Agent tool pins only sonnet|opus|haiku (F1, iteration 31), so under an opus-first controller a
# bare "fable" would silently fall back to opus. claude:claude-fable-5 = a REAL bounded Fable run.
export MISSION_DESIGNER_MODEL="${MISSION_DESIGNER_MODEL:-claude:claude-fable-5}"
# Per-iteration METERED-spend ceiling (2026-07-18, Mark: "make sure costs don't go crazy"):
# the sum of all metered-API spend (codex $ + gemini $) within ONE iteration must stay under
# this. Enforced by the skill's Gate-3 metered ledger; quota-bucket (subscription) spend is
# NOT counted — this caps dollars, not tokens.
export MISSION_METERED_BUDGET_USD="${MISSION_METERED_BUDGET_USD:-5}"
export MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-opus}"
export MISSION_EXECUTOR_MODEL="${MISSION_EXECUTOR_MODEL:-opus}"
# evaluator default = sonnet (2026-07-16, Mark directive on #399: "default can be gemini (if able
# to git clone the codebase etc)? otherwise sonnet-5"). gemini managed_agents is NOT viable as the
# evaluator today — VERIFIED iteration 38: (1) architecturally the request body carries only
# Directive+SystemPrompt over a server-side CapRemoteSandbox (managed_agents.go:164), so it cannot
# see the sprint's UNCOMMITTED worktree changes nor re-run local tests — at most it could clone the
# public origin/dev, which lacks the changes; (2) the backend live-timed-out (http2 timeout, same
# class as iters 36-37). So the ladder resolves to sonnet-5: pinnable via the Agent tool (fable is
# not — F1), distinct from the opus executor (generator≠judge holds), cheap, behavioral (re-runs
# tests locally). This also RETIRES the per-iteration fable→sonnet re-route (iters 31/36) into a
# standing default. gemini-as-evaluator is a queued follow-up (diff-bridge + backend reliability).
export MISSION_EVALUATOR_MODEL="${MISSION_EVALUATOR_MODEL:-sonnet}"

# 1. Kill switch — the intended "off" state, exit silently.
if [ -f "$KILL_SWITCH" ]; then
  log "kill switch present ($KILL_SWITCH) — skip"; exit 0
fi

# 1b. ONE iteration at a time (2026-07-10, continuous mode): two concurrent
#     controllers would stomp the charter/log in the main tree and could pick
#     the same queue item. If one is still running, yield this slot.
#     PIDFILE-based (2026-07-16): the old `pgrep -f "claude -p Run one mission"`
#     matched ANY process whose cmdline contained the phrase — including a
#     human's monitoring shell (`pgrep -f "claude -p Run one mission"` itself!),
#     which made a kickstarted fire yield against its own observer. A pidfile
#     + liveness check cannot false-positive.
if [ -f "$PIDFILE" ]; then
  oldpid=$(head -1 "$PIDFILE" 2>/dev/null)
  if [ -n "$oldpid" ] && kill -0 "$oldpid" 2>/dev/null; then
    log "previous iteration still running (pid $oldpid) — yield (next interval retries)"; exit 0
  fi
  rm -f "$PIDFILE"   # stale pidfile from a crashed/killed run — proceed
fi

# 2. claude CLI reachable?
if ! command -v claude >/dev/null 2>&1; then
  log "claude CLI not on PATH — abort"
  ailang messages send controlplane "mission-control driver: claude CLI not found on PATH" \
    --title "Mission iteration FAILED to start" --from "$MSG_FROM" 2>/dev/null
  exit 1
fi

# 3. Dry run — verify wiring without spending tokens (no probes fired).
if [ "${MISSION_DRY_RUN:-0}" = "1" ]; then
  log "DRY RUN ok: mission=$MISSION_NAME repo-slug=$MISSION_REPO doc=$MISSION_DOC workdir=$REPO pidfile=$PIDFILE prefs=$PREFS timeout=${HARD_TIMEOUT}s | roles: designer=$MISSION_DESIGNER_MODEL planner=$MISSION_PLANNER_MODEL executor=$MISSION_EXECUTOR_MODEL evaluator=$MISSION_EVALUATOR_MODEL"; exit 0
fi

# 4. Select the model (probe doubles as the subscription-auth check: API keys
#    are stripped above, so a passing probe proves keychain/token auth too).
if ! select_model; then
  log "NO usable model in prefs ($PREFS) — quota-exhausted across candidates or auth dead. Refusing."
  ailang messages send controlplane \
    "mission-control refused to start: no usable model in prefs ($PREFS). Either every candidate is quota-limited or subscription auth is unavailable (keychain locked / rig at login screen?). Zero tokens spent beyond probes." \
    --title "Mission iteration blocked: no usable model" --from "$MSG_FROM" 2>/dev/null
  [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
    --body "⚠️ Mission iteration did not start: **no usable model** in preference list (\`$PREFS\`) — all candidates quota-limited or auth unavailable. Will retry next interval; recovery is automatic when any candidate's probe succeeds." 2>/dev/null
  exit 1
fi

# Announce model CHANGES on #329 (not every iteration — only transitions).
PREV_MODEL=$(cat "$LAST_MODEL_FILE" 2>/dev/null || true)
if [ -n "$MODEL" ] && [ "$MODEL" != "${PREV_MODEL:-}" ]; then
  printf '%s\n' "$MODEL" > "$LAST_MODEL_FILE"
  if [ -n "${PREV_MODEL:-}" ]; then
    log "controller model change: ${PREV_MODEL} → ${MODEL} (${MODEL_WHY})"
    [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
      --body "🔁 Controller model: **${PREV_MODEL} → ${MODEL}** (${MODEL_WHY}) at $(date '+%F %H:%M %Z'). Automatic — preference order \`$PREFS\`; reverts when a higher-preference probe succeeds again." 2>/dev/null || true
  fi
fi

# ONE-SHOT executor override (2026-07-16, Mark: fleet live-fire tests). If armed, the file's value
# overrides MISSION_EXECUTOR_MODEL for exactly THIS iteration and is deleted on consumption.
# Placed after every early-exit (kill switch, overlap yield, no-model refusal) so a fire that does
# not actually run can never burn the shot. Arm with e.g.:
#   echo "codex:gpt-5.6-sol" > ~/.ailang/state/mission-executor-model-once
if [ -f "$EXEC_ONCE_FILE" ]; then
  once=$(head -1 "$EXEC_ONCE_FILE" 2>/dev/null)
  rm -f "$EXEC_ONCE_FILE"
  if [ -n "$once" ]; then
    export MISSION_EXECUTOR_MODEL="$once"
    log "one-shot executor override consumed: executor=$once (this iteration only)"
  fi
fi

log "=== mission iteration starting (controller=$MODEL via ${MODEL_WHY}, timeout=${HARD_TIMEOUT}s | roles: designer=$MISSION_DESIGNER_MODEL planner=$MISSION_PLANNER_MODEL executor=$MISSION_EXECUTOR_MODEL evaluator=$MISSION_EVALUATOR_MODEL) ==="

PROMPT="Run one mission-control iteration: invoke the mission-control skill for \
${MISSION_DOC} and follow its gates. You are a scheduled run; \
there is no human present — park anything needing human input and report via \
ailang messages and the GitHub bookkeeping issue, per the skill."

# _mc_run_once → runs claude -p with BOTH watchdogs, waits, sets global RC.
# Watchdogs are per-attempt (fresh PIDs each retry).
_mc_run_once() {
  claude -p "$PROMPT" \
    --model "$MODEL" \
    --permission-mode bypassPermissions \
    >>"$LOG" 2>&1 &
  CLAUDE_PID=$!
  printf '%s\n' "$CLAUDE_PID" > "$PIDFILE"   # overlap guard reads this (per-attempt: retries refresh it)

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
  return "$RC"
}

# Run with transient-retry. On a non-zero exit that is NOT a deliberate watchdog
# kill (143/137) AND whose THIS-attempt output carries a transient signature,
# back off and re-run — up to TRANSIENT_RETRIES total attempts.
attempt=1
while : ; do
  logpos=$(wc -l < "$LOG" 2>/dev/null || echo 0)
  _mc_run_once; RC=$?
  [ "$RC" -eq 0 ] && break
  case "$RC" in 143|137) break ;; esac   # watchdog kill — never retry
  if [ "$attempt" -lt "$TRANSIENT_RETRIES" ] \
     && tail -n +$((logpos + 1)) "$LOG" 2>/dev/null | grep -qiE "$TRANSIENT_SIG"; then
    backoff=$(( TRANSIENT_BACKOFF * attempt ))
    log "transient API error (rc=$RC) attempt $attempt/$TRANSIENT_RETRIES — retrying in ${backoff}s (Anthropic capacity)"
    sleep "$backoff"
    attempt=$((attempt + 1))
    continue
  fi
  break
done

rm -f "$PIDFILE"   # this instance owns the run; yield paths above never reach here

if [ "$RC" -ne 0 ]; then
  log "iteration exited rc=$RC"
  ailang messages send controlplane \
    "mission-control iteration exited rc=$RC (timeout or crash). Log: $LOG" \
    --title "Mission iteration FAILED (rc=$RC)" --from "$MSG_FROM" 2>/dev/null
  [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
    --body "⚠️ Mission iteration **FAILED to complete** (rc=$RC — timeout or crash) at $(date '+%F %H:%M %Z'). Log on the rig: \`$LOG\`. The queue is untouched; the next interval will retry." 2>/dev/null
else
  log "iteration complete (rc=0)"
  # The skill itself sends the substantive report (Gate 5, both channels).
fi
exit "$RC"
