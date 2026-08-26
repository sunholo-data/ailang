#!/usr/bin/env bash
# mission-control.sh — continuous outer-loop iterations for ONE mission (default: V1).
# PORTABLE (M-MISSION-PORTABILITY M1, 2026-07-21, attended): MISSION_PROFILE=<name>
# sources ~/.config/ailang/mission-<name>.env (MISSION_NAME/REPO/DOC/...). MISSION_NAME=v1
# (the default) keeps the LEGACY state paths + log name EXACTLY as before — bit-for-bit,
# no migration; any other name gets fully namespaced state so two missions never collide.
#
# Fires a headless controller session that runs the mission-control skill:
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
# MISSION_MODEL_PREFS (default "claude-opus-5,claude-opus-4-8,claude-fable-5"
# — Opus 5 first since 2026-07-27 (Mark), 4.8 kept as probe fallback — OPUS-FIRST
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
# D6 rollback pin: always sourced for the RESOLVED mission name, so the documented
# `MISSION_PLANNER_MODEL=opus` rollback works for V1 (whose plist sets no MISSION_PROFILE).
# Convention: entries use ${VAR:-value} so a command-line env pin still wins (the AC fixtures
# depend on that). Double-source for World is idempotent.
[ -f "$HOME/.config/ailang/mission-${MISSION_NAME}.env" ] \
  && . "$HOME/.config/ailang/mission-${MISSION_NAME}.env"
STATE_DIR="$HOME/.ailang/state"
if [ "$MISSION_NAME" = "v1" ]; then
  # LEGACY paths — bit-for-bit compat with the live V1 loop (no migration).
  LOG=/tmp/ailang-mission-control.log
  KILL_SWITCH="$STATE_DIR/mission-control.disabled"
  PIDFILE="$STATE_DIR/mission-control.pid"
  OVERRIDE_FILE="$STATE_DIR/mission-model"
  LAST_MODEL_FILE="$STATE_DIR/mission-model-last"
  EXEC_ONCE_FILE="$STATE_DIR/mission-executor-model-once"
  # NAMESPACED, deliberately breaking the legacy-path rule above (iteration 282 found it).
  # Every OTHER mission uses mission-<name>-gh-issue; V1 alone read the fleet-shared bare
  # file, which holds a CLOSED thread (745) while mission-v1-gh-issue holds the live one
  # (852). Gate 0 reads Mark's directives from this issue, so a stale value silently
  # reports "0 directives" against a dead thread — the loop only ever saw real directives
  # because the controller distrusted the env and re-derived the namespaced path by hand.
  GH_ISSUE_FILE="$STATE_DIR/mission-v1-gh-issue"
  BLOCKED_FILE="$STATE_DIR/mission-control.blocked"
  PIN_DRIFT_FILE="$STATE_DIR/mission-control.pin-drift"
  MSG_FROM="mission-control"
else
  LOG="/tmp/ailang-mission-${MISSION_NAME}.log"
  KILL_SWITCH="$STATE_DIR/mission-${MISSION_NAME}.disabled"
  PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"
  OVERRIDE_FILE="$STATE_DIR/mission-${MISSION_NAME}-model"
  LAST_MODEL_FILE="$STATE_DIR/mission-${MISSION_NAME}-model-last"
  EXEC_ONCE_FILE="$STATE_DIR/mission-${MISSION_NAME}-executor-model-once"
  GH_ISSUE_FILE="$STATE_DIR/mission-${MISSION_NAME}-gh-issue"
  BLOCKED_FILE="$STATE_DIR/mission-${MISSION_NAME}.blocked"
  PIN_DRIFT_FILE="$STATE_DIR/mission-${MISSION_NAME}.pin-drift"
  MSG_FROM="mission-${MISSION_NAME}"
fi
# -----------------------------------------------------------------------------
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"

# BILLING GUARD (2026-07-10): the mission MUST bill the Claude subscription,
# never API credits. secrets.env exports ANTHROPIC_API_KEY for other tools —
# strip it so claude's only auth paths are subscription ones (keychain OAuth,
# or CLAUDE_CODE_OAUTH_TOKEN if set). Subscription-or-nothing by construction.
# 2026-07-27 extension (same construction, codex lane): secrets.env also exports the
# METERED OPENAI_API_KEY — strip it so codex's only auth is the ChatGPT-subscription
# OAuth in ~/.codex/auth.json (auth_mode=chatgpt). Metered OpenAI runs happen outside
# mission iterations, deliberately.
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY

log() { echo "[$(date '+%F %H:%M:%S')] $*" | tee -a "$LOG"; }

# _mc_notify TITLE BODY LABEL — report a degradation on BOTH human channels.
# Extracted from the lane-degradation block (564cc4640) when the driver-pin notice (#558) needed
# the identical shape: two near-identical emit blocks is how one of them silently rots.
# NOT fail-closed: aborting on a failed post would make GitHub/controlplane availability a hard
# dependency of every fire. A failed post is LOUD in the driver log instead — the one thing the
# silent-fallback class never was (Critical Principle 2).
_mc_notify() {
  local title="$1" body="$2" label="$3"
  ailang messages send controlplane "$body" \
    --title "$title" --from "$MSG_FROM" 2>/dev/null \
    || log "WARNING: ${label} notice FAILED to send via ailang messages"
  if [ -n "${MISSION_GH_ISSUE:-}" ]; then
    gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" --body "$body" >/dev/null 2>&1 \
      || log "WARNING: ${label} notice FAILED to post to issue #${MISSION_GH_ISSUE}"
  else
    log "WARNING: ${label} notice needed but MISSION_GH_ISSUE is unset — no issue notice possible"
  fi
}

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
PREFS="${MISSION_MODEL_PREFS:-claude-opus-5,claude-opus-4-8,claude-fable-5}"
CONTROLLER_FALLBACK="${MISSION_CONTROLLER_FALLBACK:-codex:gpt-5.6-sol}"
QUOTA_SIG="usage limit|rate.?limit|quota|exceeded|too many requests|weekly limit"
PROBE_TIMEOUT="${MISSION_PROBE_TIMEOUT:-120}"   # per-probe wall-clock cap, seconds

# _mc_bounded SECONDS CMD... — run CMD with a hard wall-clock cap.
# rc = CMD's rc, or 124 on expiry (mirrors GNU `timeout`, which this rig does not have).
# Combined stdout+stderr lands in $MC_BOUNDED_OUT.
#
# Why (2026-07-27): a model probe is a network call to a third party and CAN hang. Observed that
# day: `codex exec --model <unknown-model>` ran past 180s with no output. Both probes below used
# to be unbounded command substitutions, so one hung probe would burn the whole 6h fire before the
# driver's HARD_TIMEOUT reclaimed it — the exact failure class as mission-control Standing rule 6
# ("every wait is bounded"), which the loop enforces on itself but the driver did not.
_mc_bounded() {
  local secs="$1"; shift
  local out_f rc deadline pid
  out_f=$(mktemp -t mc_bounded) || { MC_BOUNDED_OUT=""; return 125; }
  ( exec "$@" ) >"$out_f" 2>&1 &
  pid=$!
  deadline=$(( $(date +%s) + secs ))
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
      kill "$pid" 2>/dev/null; sleep 2; kill -9 "$pid" 2>/dev/null
      MC_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
      return 124
    fi
    sleep 2
  done
  wait "$pid"; rc=$?
  MC_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
  return "$rc"
}

# _mc_probe MODEL → 0 usable | 1 quota-limited | 2 unusable (auth/transient×2/timeout×2)
_mc_probe() {
  local m="$1" out rc
  _mc_bounded "$PROBE_TIMEOUT" claude -p 'reply with exactly: ok' --model "$m"; rc=$?
  out="$MC_BOUNDED_OUT"
  [ "$rc" -eq 0 ] && return 0
  # Log the captured output tail on timeout: an EMPTY capture (claude -p is
  # silent until completion) means a hang/backoff loop, error text means an
  # actual failure — 2026-08-05's refusals were undiagnosable without this.
  [ "$rc" -eq 124 ] && log "model $m probe timed out after ${PROBE_TIMEOUT}s — captured output: '$(printf '%s' "$out" | tail -c 200 | tr '\n' ' ')'"
  if printf '%s' "$out" | grep -qiE "$QUOTA_SIG"; then return 1; fi
  # transient? retry once
  sleep 5
  _mc_bounded "$PROBE_TIMEOUT" claude -p 'reply with exactly: ok' --model "$m"; rc=$?
  out="$MC_BOUNDED_OUT"
  [ "$rc" -eq 0 ] && return 0
  [ "$rc" -eq 124 ] && log "model $m probe timed out after ${PROBE_TIMEOUT}s (retry) — captured output: '$(printf '%s' "$out" | tail -c 200 | tr '\n' ' ')'"
  printf '%s' "$out" | grep -qiE "$QUOTA_SIG" && return 1
  return 2
}

# _mc_probe_codex MODEL → 0 usable | non-zero unusable. The OpenAI API key is
# stripped above, so a pass proves the ChatGPT-subscription OAuth lane works.
_mc_probe_codex() {
  local m="$1" rc
  _mc_bounded "$PROBE_TIMEOUT" codex exec --skip-git-repo-check --model "$m" 'reply with exactly: ok'
  rc=$?
  [ "$rc" -eq 124 ] && log "controller fallback codex:$m probe timed out after ${PROBE_TIMEOUT}s"
  [ "$rc" -ne 0 ] && log "controller fallback codex:$m probe failed (rc=$rc): $(printf '%s' "$MC_BOUNDED_OUT" | tail -3 | tr '\n' ' ')"
  return "$rc"
}

_mc_set_controller() {
  local requested="$1"
  MODEL_WHY="$2"
  case "$requested" in
    codex:*) CONTROLLER_PROVIDER=codex; MODEL="${requested#codex:}"; MISSION_ANTHROPIC_AVAILABLE=0 ;;
    claude:*) CONTROLLER_PROVIDER=claude; MODEL="${requested#claude:}"; MISSION_ANTHROPIC_AVAILABLE=1 ;;
    *) CONTROLLER_PROVIDER=claude; MODEL="$requested"; MISSION_ANTHROPIC_AVAILABLE=1 ;;
  esac
  CONTROLLER_ID="${CONTROLLER_PROVIDER}:${MODEL}"
  export CONTROLLER_PROVIDER CONTROLLER_ID MODEL MODEL_WHY MISSION_ANTHROPIC_AVAILABLE
}

select_model() {
  # 1. absolute pin
  if [ -n "${MISSION_MODEL:-}" ]; then _mc_set_controller "$MISSION_MODEL" "env pin"; return 0; fi
  # 2. override file pin (optional expiry epoch)
  if [ -f "$OVERRIDE_FILE" ]; then
    local ov_model ov_until now
    read -r ov_model ov_until < "$OVERRIDE_FILE" 2>/dev/null || true
    now=$(date +%s)
    if [ -n "${ov_until:-}" ] && [ "$now" -ge "${ov_until:-0}" ]; then
      rm -f "$OVERRIDE_FILE"
      log "model override expired — resuming preference probing"
    elif [ -n "${ov_model:-}" ]; then
      _mc_set_controller "$ov_model" "override file"; return 0
    fi
  fi
  # 3. ordered preference probing
  local m why rcode
  for m in $(printf '%s' "$PREFS" | tr ',' ' '); do
    _mc_probe "$m"; rcode=$?
    case "$rcode" in
      0) _mc_set_controller "$m" "probe ok"; return 0 ;;
      1) log "model $m quota-limited — falling through" ;;
      2) log "model $m unusable (auth/transient) — falling through" ;;
    esac
  done
  case "$CONTROLLER_FALLBACK" in
    codex:*)
      m="${CONTROLLER_FALLBACK#codex:}"
      log "all Anthropic controller candidates unavailable — probing $CONTROLLER_FALLBACK"
      if _mc_probe_codex "$m"; then
        _mc_set_controller "$CONTROLLER_FALLBACK" "Anthropic unavailable; subscription fallback"
        return 0
      fi
      ;;
    *) log "unsupported MISSION_CONTROLLER_FALLBACK '$CONTROLLER_FALLBACK' (expected codex:<model>)" ;;
  esac
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
# MISSION_EXECUTOR_MODEL specifically accepts EITHER form: an Agent alias (opus) OR a
# provider:model value (codex:gpt-5.6-sol — the DEFAULT since 2026-07-27) which the mission-control
# Gate-3 recipe routes to a bounded `codex exec` run in the sprint worktree (M1b). Default flipped
# to codex 2026-07-27 (Mark, quota relief): codex now authenticates via ChatGPT SUBSCRIPTION
# (auth.json auth_mode=chatgpt; metered API-key backup at ~/.codex/auth.json.apikey.bak), so it is
# a quota lane, not metered $ — the old "never bills metered $ by accident" rationale is moot.
# The pre-flight probe below falls back to opus for the fire when the codex bucket is spent.
# Generator≠judge holds: evaluator stays sonnet regardless of executor lane.
# Weekly rolling bookkeeping issue (2026-07-16, Mark): the issue number lives in a state file so
# the skill's Monday-07:00 rotation (aligned to the quota reset) moves threads without a driver
# edit. Precedence: env pin > state file > 329 (the original thread). Exported so the skill's
# gh snippets see the same number the driver reports to.
MISSION_GH_ISSUE="${MISSION_GH_ISSUE:-$(head -1 "$GH_ISSUE_FILE" 2>/dev/null)}"
[ -z "${MISSION_GH_ISSUE:-}" ] && [ "$MISSION_NAME" = "v1" ] && MISSION_GH_ISSUE=329
export MISSION_GH_ISSUE

# --- DRIVER PIN (#558) -------------------------------------------------------
# Re-exec this driver out of a worktree pinned to committed origin/dev, so the script, the
# skill and the charter all come from the same reviewed commit rather than from whatever the
# shared clone happens to hold. See tools/launchd/lib/pin-root.sh for the full rationale.
#
# PLACED HERE, and the position is load-bearing in both directions:
#   * BEFORE the model probes below — a re-exec restarts the script from the top, so pinning
#     any later would probe every lane twice and bill it twice;
#   * BEFORE the pidfile is written (line ~583) — the re-exec'd copy would otherwise read its
#     own parent's pid and yield to itself as an overlap, turning every fire into a no-op;
#   * AFTER the state block, so a failed pin has LOG, log(), MSG_FROM and MISSION_GH_ISSUE
#     available to report with. Reporting a stale driver on a channel that needs the stale
#     driver's own config to be resolved is not reporting.
# Sourced from $REPO, which on the first pass is the UNPINNED clone: the stale helper re-execs
# into the pinned driver, which then sources the pinned helper. Two passes by construction.
if [ -f "$REPO/tools/launchd/lib/pin-root.sh" ]; then
  . "$REPO/tools/launchd/lib/pin-root.sh"
  pin_root_to_committed_ref "$@"
else
  # PIN_DRIFT is normally initialised by the helper. It must be set HERE too: this branch is the
  # pre-helper clone, i.e. exactly the case it exists to survive, and `set -u` would otherwise
  # abort at the DRY RUN line below — the fallback crashing only on the fallback path.
  PIN_STATUS="STALE"
  PIN_DRIFT="?"
  PIN_NOTE="$REPO/tools/launchd/lib/pin-root.sh is absent — this clone predates the driver pin (#558)"
fi
# --- DRIVER PIN DECISION START ---
_pin_degraded=""
_pin_drift_degraded=""
if [ "$PIN_STATUS" = "STALE" ]; then
  # Deliberately NOT fatal: aborting would make network/git availability a hard dependency of
  # every fire, trading rare silent staleness for common loud outage. Loud instead of fatal.
  log "DRIVER PIN FAILED — this fire runs the WORKING TREE at $(git -C "$REPO" rev-parse --short HEAD 2>/dev/null), not committed code: $PIN_NOTE"
  _pin_degraded="
- driver pin: **FAILED** — \`$PIN_NOTE\`. This fire ran \`$REPO\` at \`$(git -C "$REPO" rev-parse --short HEAD 2>/dev/null)\` ($(git -C "$REPO" rev-list --count HEAD..origin/dev 2>/dev/null || echo '?') behind \`origin/dev\`), so any landed driver/skill/charter fix newer than that commit was NOT in effect."
else
  log "driver pin: $PIN_NOTE"
  if [ "$PIN_STATUS" = "pinned" ]; then
    # Normalise once: `set -u` is in force and every reference below is bare. pin-root.sh sets
    # PIN_DRIFT on the line after PIN_STATUS, so an unset value is unreachable today — pinned
    # so the invariant is enforced rather than assumed.
    PIN_DRIFT="${PIN_DRIFT:-}"
    case "$PIN_DRIFT" in
      ''|*[!0-9]*)
        log "driver pin drift: unknown ($PIN_DRIFT); notice suppressed"
        ;;
      *)
        _pin_drift_warn="${AILANG_DRIVER_DRIFT_WARN:-25}"
        # A threshold of 0 persists a previous of 0, and `-ge $((0 * 2))` is then true on
        # every fire — the notice-every-90-minutes outcome this whole block exists to avoid,
        # arriving through its own knob. A default is an instrument too: floor it, loudly.
        case "$_pin_drift_warn" in
          ''|*[!0-9]*|0)
            log "driver pin drift: AILANG_DRIVER_DRIFT_WARN='$_pin_drift_warn' is not a positive integer; using 25"
            _pin_drift_warn=25
            ;;
        esac
        if [ "$PIN_DRIFT" -lt "$_pin_drift_warn" ]; then
          rm -f "$PIN_DRIFT_FILE"
          log "driver pin drift: $PIN_DRIFT below warning threshold $_pin_drift_warn; notice re-armed"
        else
          _pin_drift_previous=""
          [ -r "$PIN_DRIFT_FILE" ] && _pin_drift_previous="$(head -1 "$PIN_DRIFT_FILE" 2>/dev/null)"
          case "$_pin_drift_previous" in
            ''|*[!0-9]*) _pin_drift_emit=1 ;;
            *)
              if [ "$PIN_DRIFT" -ge $((_pin_drift_previous * 2)) ]; then
                _pin_drift_emit=1
              else
                _pin_drift_emit=0
              fi
              ;;
          esac
          if [ "$_pin_drift_emit" -eq 1 ]; then
            _pin_drift_degraded="$PIN_DRIFT"
            printf '%s\n' "$PIN_DRIFT" > "$PIN_DRIFT_FILE"
            log "driver pin drift: $PIN_DRIFT at/above threshold $_pin_drift_warn; notice armed (previous=${_pin_drift_previous:-none})"
          else
            log "driver pin drift: $PIN_DRIFT at/above threshold $_pin_drift_warn; deduped until doubling from $_pin_drift_previous"
          fi
        fi
        ;;
    esac
  else
    log "driver pin drift: skipped (status=$PIN_STATUS)"
  fi
fi
# --- DRIVER PIN DECISION END ---

# designer default is the claude-CLI lane (claude:<full-id>), NOT the bare "fable" alias: the
# Agent tool pins only sonnet|opus|haiku (F1, iteration 31), so under an opus-first controller a
# bare "fable" would silently fall back to opus. claude:claude-fable-5 = a REAL bounded Fable run.
export MISSION_DESIGNER_MODEL="${MISSION_DESIGNER_MODEL:-claude:claude-fable-5}"
# Per-iteration METERED-spend ceiling (2026-07-18, Mark: "make sure costs don't go crazy"):
# the sum of all metered-API spend (codex $ + gemini $) within ONE iteration must stay under
# this. Enforced by the skill's Gate-3 metered ledger; quota-bucket (subscription) spend is
# NOT counted — this caps dollars, not tokens.
export MISSION_METERED_BUDGET_USD="${MISSION_METERED_BUDGET_USD:-5}"
# THE FLIP (m-planner-codex-lane M4, mission iteration 136): the sprint-planner default
# moves to the ChatGPT-subscription codex bucket so opus stays controller-only (Mark
# quota-offload #1). The CONFIGURED default is not the EFFECTIVE lane: the skill's Gate-3
# step 1b runs tools/launchd/derive-planner-lane.sh on the picked design doc and fails
# CLOSED to opus unless that doc declares **Planner-Lane**: codex-ok AND every path it
# declares is inside the D2 infra allowlist. Rollback = uncomment MISSION_PLANNER_MODEL
# in ~/.config/ailang/mission-<name>.env (delivery mechanism added by M2 above).
# FLEET — Ollama Cloud sits at the FIRST FALLBACK, not primary (Mark 2026-08-26,
# attended; corrected from an earlier over-reach that made these the defaults).
#
# codex keeps both primary roles: it has months of track record in them, whereas
# the cloud lanes have external benchmarks, one fizzbuzz agent eval and an rc=0
# reachability probe — and NO evidence in the planner/executor roles specifically.
# An unattended nightly loop is the wrong place to find that out. The fallback
# slot exercises them for real whenever the codex bucket is spent, so evidence
# accumulates in the actual roles before anything is promoted.
#
# Same fleet for every mission: they all source THIS file and no plist overrides
# these vars, so there is one definition, not four kept in sync.
export MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-codex:gpt-5.6-sol}"
# executor = deepseek-v4-flash on the FLAT-RATE ollama route. These are the same
#            weights the fallback chain below already reaches through OpenRouter,
#            so this is a route change, not a capability change — and it degrades
#            to that metered twin if the ollama quota runs out. Draws 4.2x
#            gpt-oss (0.029 units/M, measured), ~4x cheaper per token than the
#            planner, which is what the high-volume role needs.
export MISSION_EXECUTOR_MODEL="${MISSION_EXECUTOR_MODEL:-codex:gpt-5.6-sol}"
# EXECUTOR FALLBACK CHAIN — ailang#611 (2026-08-11).
#
# RATIFIED SEMANTICS (Mark 2026-08-06, restated attended 2026-08-10 and 2026-08-11):
# "codex as default but deepseek to be replacement when codex out of quota", then
# opus as the last resort. Until now NEITHER driver implemented it: the codex and
# pi pre-flight loops are independent and each degraded straight to a hardcoded
# `opus`, so deepseek was only ever reached by hard-pinning it in a mission env
# file — which made it the DEFAULT, the opposite of the ratified policy, and left
# the pin running unprobed on World (whose driver has only the codex loop).
#
# Implemented by retargeting the codex loop's fallback rather than adding a third
# probe loop: the pi loop already runs immediately AFTER the codex loop, so handing
# the role to a `pi:*` model here means the very next loop probes it and degrades to
# opus if it is unusable too. One chain, no new probe machinery, and the per-role
# fallback stays overridable for a mission that wants a different tail.
# `:floor` DROPPED 2026-08-18 (was `...-0731:floor`). `:floor` is OpenRouter
# provider.sort=price, so it pins the executor to the CHEAPEST endpoint — and the two
# cheapest for this model carry NEGATIVE health status (StreamLake -2, Decart -5 of 28
# endpoints, measured today). The lane was routed to the least-healthy host by
# construction. World iteration 91 then returned `rc=0` with ZERO BYTES CHANGED twice,
# at 625 output tokens against the 65,536 budget with `stopReason=stop` — the normal
# success state, so the guard's stopReason assertion passes on a total failure and only
# the worktree-diff assertion caught it.
#
# NOT a proven root cause, stated honestly: a live A/B probe today had BOTH `:floor`
# (-> StreamLake) and the bare id (-> DeepInfra) return a correct tool_call, so tool
# support is not categorically broken on either route, and billing is not implicated
# either (OpenRouter credits were $8.33 remaining, and exhaustion returns 402, not a
# clean stop). What survives is a correlation worth settling: the lane's one clear
# big-milestone success (V1 iteration 156) was 2026-08-07, and `:floor` was pinned
# 2026-08-11 — every failure since is on `:floor`.
#
# KNOWN COST OF THIS CHANGE: prompt caches are PER-PROVIDER, so a bare id that
# load-balances across 26 hosts caches far less (measured 2026-08-11: unpinned
# cacheRead=0 on two identical ~27.7k prompts vs :floor caching 27,392 of 27,673 on
# call 2 => ~4.8x cheaper per repeat call). Accepted deliberately: a lane that returns
# zero bytes has worse economics than any cache multiplier can repair. ROLLBACK: re-add
# the `:floor` suffix here, or override MISSION_EXECUTOR_FALLBACK per mission env file.
# ROUTE CHANGE, NOT MODEL CHANGE (2026-08-26): the ratified semantics — "codex as
# default but deepseek to be replacement when codex out of quota", then opus last —
# are preserved exactly. Same deepseek-v4-flash weights; the flat-rate ollama route
# replaces the metered OpenRouter one. Measured 0.029 ollama usage-units/M tokens.
# ROLLBACK: restore pi:openrouter/deepseek/deepseek-v4-flash-0731 here.
export MISSION_EXECUTOR_FALLBACK="${MISSION_EXECUTOR_FALLBACK:-pi:ollama/deepseek-v4-flash:0731-cloud}"
# kimi-k3 sits between codex and opus rather than degrading straight to opus:
# strongest open-weight model measured externally (88.3 Terminal-Bench 2.1), and
# a flat-rate lane is the right thing to try before spending Anthropic quota.
# Draws 18x gpt-oss per token (0.124 units/M) — affordable because planning is ONE
# run per iteration. The pi probe loop degrades it to opus if unusable, so opus
# remains the last resort exactly as before. ROLLBACK: set this back to `opus`.
export MISSION_PLANNER_FALLBACK="${MISSION_PLANNER_FALLBACK:-pi:ollama/kimi-k3:cloud}"
# When a design doc requires the Anthropic planner lane but the controller probe
# has proved the Anthropic subscription unavailable, use Codex Sol rather than
# wedging or silently inheriting the failed controller. derive-planner-lane.sh
# applies this only when MISSION_ANTHROPIC_AVAILABLE=0.
export MISSION_PLANNER_ANTHROPIC_FALLBACK="${MISSION_PLANNER_ANTHROPIC_FALLBACK:-codex:gpt-5.6-sol}"
# Codex-lane pre-flight, ROLE-GENERIC (m-planner-codex-lane): probe once per DISTINCT
# codex model, fall back per-role on ANY non-zero rc (#486: probe MUST carry --model;
# an unusable pin is exactly as fatal as spent quota). Export AFTER fallback so the
# EXPORTED env — what the routing-evidence row reports — stays honest.
# BASH 3.2 (L19): ':'-delimited string sets, NOT associative arrays; no ${var,,}.
#
# The probe MUST carry --model (#486, 2026-07-27): without it codex exercises its DEFAULT model,
# so a pinned-but-unreachable model false-greens the lane. Live evidence that day: codex-cli
# 0.137.0 answered the model-less probe on gpt-5.5 (rc=0) while `--model gpt-5.6-sol` returned a
# 400 "requires a newer version of Codex" — the driver exported the codex pin as healthy and the
# failure only surfaced inside the skill's Gate-3 recipe, one silent fallback later.
#
# Fall back on ANY non-zero rc, not just quota signatures: an unusable model pin is exactly as
# fatal to the lane as a spent quota, and the old quota-only gate is what let #486 through. The
# skill's Gate-3 recipe re-probes and would fall back anyway; doing it here keeps the EXPORTED
# env honest, which is what the routing-evidence row reports.
_cx_probed=":"   # models probed this fire (dedupe: planner+executor share the default model)
_cx_failed=":"   # models whose probe failed
# LANE-DEGRADATION LEDGER (motoko mission iteration 0, 2026-08-12; Mark ratified the fix).
# Until now a lane demotion was `log`ged here and NOWHERE ELSE — none of this driver's four
# `gh issue comment` sites covers it, so the human channel saw nothing. That is exactly how the
# World mission spent FIVE iterations (18/19/21/22) silently demoted from codex to opus, each
# mis-attributed to a spent quota, before iter-23 found the real cause. A fallback visible only in
# a routing-evidence row written AFTER the fact is still a silent fallback (Critical Principle 2):
# by then the iteration has already run on the wrong lane.
# Accumulate here; emit ONCE below, AFTER every early exit and BEFORE the iteration starts.
# bash 3.2 (L19/L21): no associative arrays — ';'-delimited "model=rc", newline-delimited ledger.
_lane_degraded=""   # newline-delimited markdown bullets, one per degraded role
_cx_rcmap=""        # "model=rc;" so the emit site names the probe's exit code, not just the lane
_pi_rcmap=""
for role in PLANNER EXECUTOR; do
  var="MISSION_${role}_MODEL"; val="${!var}"
  case "$val" in codex:*)
    cx_model="${val#codex:}"
    case "$_cx_probed" in *":${cx_model}:"*) : ;; *)   # not yet probed
      _cx_probed="${_cx_probed}${cx_model}:"
      _mc_bounded "$PROBE_TIMEOUT" codex exec --skip-git-repo-check --model "$cx_model" 'reply with exactly: ok'
      cx_rc=$?; cx_out="$MC_BOUNDED_OUT"
      if [ "$cx_rc" -ne 0 ]; then
        _cx_failed="${_cx_failed}${cx_model}:"
        # why-classification happens ONCE, at probe time (timeout / quota-sig / other)
        if [ "$cx_rc" -eq 124 ]; then cx_why="probe timed out after ${PROBE_TIMEOUT}s"
        elif printf '%s' "$cx_out" | grep -qiE "$QUOTA_SIG"; then cx_why="quota-limited"
        else cx_why="probe failed (rc=$cx_rc)"; fi
        log "codex model '$cx_model' unusable: $cx_why"
        log "codex probe output: $(printf '%s' "$cx_out" | tail -3 | tr '\n' ' ')"
        _cx_rcmap="${_cx_rcmap}${cx_model}=${cx_rc};"
      fi
    ;; esac
    case "$_cx_failed" in *":${cx_model}:"*)
      role_lc=$(printf '%s' "$role" | tr 'A-Z' 'a-z')   # ${role,,} is bash-4.0-only (L21)
      # Hand off to the NEXT link, not straight to opus (#611). A `pi:*` value here
      # is probed by the pi loop below, which degrades to opus on its own failure —
      # that is what makes codex -> deepseek -> opus a real chain. `%s` rather than
      # a bare format string: the value is data, and a stray % would be a directive.
      fbvar="MISSION_${role}_FALLBACK"; fb="${!fbvar:-opus}"
      log "codex ${role_lc} lane -> falling back to '$fb' for this fire (model '$cx_model')"
      _cx_rc_for=$(printf '%s' "$_cx_rcmap" | tr ';' '\n' | grep "^${cx_model}=" | head -1 | cut -d= -f2)
      [ -n "$_cx_rc_for" ] || _cx_rc_for="unknown"
      _lane_degraded="${_lane_degraded}
- \`${role_lc}\`: **codex** lane \`${cx_model}\` unusable (probe rc=\`${_cx_rc_for}\`$([ "$_cx_rc_for" = "124" ] && printf ' — TIMEOUT after %ss' "$PROBE_TIMEOUT")) → handed to \`${fb}\`"
      printf -v "$var" '%s' "$fb"; export "$var"
    ;; esac
  ;; esac
done
# pi-lane pre-flight, ROLE-GENERIC (mirrors the codex loop above; added 2026-08-06,
# Mark: DeepSeek executor lane — trial record in models.yml pi-or-deepseek-v4-flash).
# Probe once per DISTINCT pi model, fall back per-role on ANY non-zero rc — an
# unusable pin is exactly as fatal as a spent bucket (#486). The OpenRouter key
# rides ~/.pi/agent/models.json (custom provider), not env, so this probe is
# headless-safe. --no-tools keeps it ~1 reply-token; --no-session avoids polluting
# ~/.pi/sessions. BASH 3.2 (L19): ':'-delimited string sets, NOT associative arrays.
_pi_probed=":"   # models probed this fire (dedupe: planner+executor could share one)
_pi_failed=":"   # models whose probe failed
for role in PLANNER EXECUTOR; do
  var="MISSION_${role}_MODEL"; val="${!var}"
  case "$val" in pi:*)
    pi_model="${val#pi:}"
    case "$_pi_probed" in *":${pi_model}:"*) : ;; *)   # not yet probed
      _pi_probed="${_pi_probed}${pi_model}:"
      _mc_bounded "$PROBE_TIMEOUT" pi --mode json --no-session --no-tools --model "$pi_model" -p 'reply with exactly: ok'
      pi_rc=$?; pi_out="$MC_BOUNDED_OUT"
      if [ "$pi_rc" -ne 0 ]; then
        _pi_failed="${_pi_failed}${pi_model}:"
        if [ "$pi_rc" -eq 124 ]; then pi_why="probe timed out after ${PROBE_TIMEOUT}s"
        else pi_why="probe failed (rc=$pi_rc)"; fi
        log "pi model '$pi_model' unusable: $pi_why"
        log "pi probe output: $(printf '%s' "$pi_out" | tail -3 | tr '\n' ' ')"
        _pi_rcmap="${_pi_rcmap}${pi_model}=${pi_rc};"
      fi
    ;; esac
    case "$_pi_failed" in *":${pi_model}:"*)
      role_lc=$(printf '%s' "$role" | tr 'A-Z' 'a-z')   # ${role,,} is bash-4.0-only (L21)
      log "pi ${role_lc} lane -> falling back to opus for this fire (model '$pi_model')"
      _pi_rc_for=$(printf '%s' "$_pi_rcmap" | tr ';' '\n' | grep "^${pi_model}=" | head -1 | cut -d= -f2)
      [ -n "$_pi_rc_for" ] || _pi_rc_for="unknown"
      _lane_degraded="${_lane_degraded}
- \`${role_lc}\`: **pi** lane \`${pi_model}\` unusable (probe rc=\`${_pi_rc_for}\`$([ "$_pi_rc_for" = "124" ] && printf ' — TIMEOUT after %ss' "$PROBE_TIMEOUT")) → handed to \`opus\` (end of chain)"
      printf -v "$var" 'opus'; export "$var"
    ;; esac
  ;; esac
done
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

# 3. Dry run — verify wiring without spending tokens (probes DO fire; they are ~1 reply-token).
# `lanes=` reports the degradation ledger's state HERE, ~90 lines before the notice's own emit
# site. That is deliberate: it makes the seam between accumulation (the probe loops) and emission
# testable without spending an iteration, which is otherwise the one part of the notice path a
# cheap test cannot reach. Prove BOTH arms:
#   MISSION_PROFILE=<m> MISSION_DRY_RUN=1                                      -> lanes=ok
#   MISSION_PROFILE=<m> MISSION_DRY_RUN=1 MISSION_EXECUTOR_MODEL=codex:bogus \
#     MISSION_PROBE_TIMEOUT=10                                                 -> lanes=DEGRADED(...)
if [ "${MISSION_DRY_RUN:-0}" = "1" ]; then
  if [ -n "$_lane_degraded" ]; then
    _dry_lanes="DEGRADED($(printf '%s' "$_lane_degraded" | grep -c '^- '))$(printf '%s' "$_lane_degraded" | tr '\n' ' ')"
  else
    _dry_lanes="ok"
  fi
  log "DRY RUN ok: mission=$MISSION_NAME repo-slug=$MISSION_REPO doc=$MISSION_DOC workdir=$REPO pidfile=$PIDFILE prefs=$PREFS timeout=${HARD_TIMEOUT}s | roles: designer=$MISSION_DESIGNER_MODEL planner=$MISSION_PLANNER_MODEL executor=$MISSION_EXECUTOR_MODEL evaluator=$MISSION_EVALUATOR_MODEL | lanes=$_dry_lanes | pin=$PIN_STATUS($PIN_DRIFT behind)"; exit 0
fi

# 4. Select the model (probe doubles as the subscription-auth check: API keys
#    are stripped above, so a passing probe proves keychain/token auth too).
if ! select_model; then
  log "NO usable controller in Anthropic prefs ($PREFS) or fallback ($CONTROLLER_FALLBACK). Refusing."
  # Announce ONCE per blocked episode, not once per refusal. mission-recovery.sh
  # retries every ~20 min while a stall lasts (instead of waiting out the 90-min
  # StartInterval), so notifying per refusal would post a dozen identical GH
  # comments and controlplane messages during a single API incident. The marker
  # is cleared the moment a probe succeeds, so the NEXT episode announces again.
  if [ -f "$BLOCKED_FILE" ]; then
    log "refusal already announced this episode ($BLOCKED_FILE) — staying quiet"
  else
    : > "$BLOCKED_FILE"
    ailang messages send controlplane \
      "mission-control refused to start: no usable controller in Anthropic prefs ($PREFS) or fallback ($CONTROLLER_FALLBACK). Per-model reasons are in the driver log. Zero tokens spent beyond probes. Further refusals in this episode are silent; mission-recovery retries automatically." \
      --title "Mission iteration blocked: no usable model" --from "$MSG_FROM" 2>/dev/null
    [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
      --body "⚠️ Mission iteration did not start: **no usable controller** in Anthropic preferences (\`$PREFS\`) or fallback (\`$CONTROLLER_FALLBACK\`). Per-model detail is in the driver log. \`mission-recovery\` retries automatically; further refusals in this episode are silent to avoid comment spam." 2>/dev/null
  fi
  exit 1
fi
rm -f "$BLOCKED_FILE"   # a probe succeeded — the blocked episode (if any) is over

# Announce model CHANGES on #329 (not every iteration — only transitions).
PREV_MODEL=$(cat "$LAST_MODEL_FILE" 2>/dev/null || true)
if [ -n "$CONTROLLER_ID" ] && [ "$CONTROLLER_ID" != "${PREV_MODEL:-}" ]; then
  printf '%s\n' "$CONTROLLER_ID" > "$LAST_MODEL_FILE"
  if [ -n "${PREV_MODEL:-}" ]; then
    log "controller model change: ${PREV_MODEL} → ${CONTROLLER_ID} (${MODEL_WHY})"
    [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
      --body "🔁 Controller model: **${PREV_MODEL} → ${CONTROLLER_ID}** (${MODEL_WHY}) at $(date '+%F %H:%M %Z'). Automatic — Anthropic preference order \`$PREFS\`, then \`$CONTROLLER_FALLBACK\`; reverts when a higher-preference probe succeeds again." 2>/dev/null || true
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

# Background-task ceiling (2026-08-08, mission-world iter-65 + V1 iter-167). `claude -p` terminates
# still-running BACKGROUND tasks 600s after the assistant's last turn ends, prints
# "Background tasks still running after 600s; terminating." and exits **rc=0**. The controller
# spawns its planner/executor as a background Agent and stops its turn to wait, so the slot dies
# with a plausible transcript, zero commits, zero charter rows — and NEITHER watchdog fires,
# because rc=0 is a clean exit. Attribution is exact and first-party: 2 hits in this driver's own
# log (/tmp/ailang-mission-control.log lines 3193, 3420) = the 2026-08-07 12:26 fire (iteration
# 159) and the 2026-08-08 09:09 fire (iteration 167 attempt 1); 2 hits in mission-world's log = its
# only 2 orphaned slots in 67 iterations. Zero misses, zero false positives across both missions.
# 0 = wait indefinitely — that is not an UNBOUNDED wait (Standing rule 6): it hands the bound to
# HARD_TIMEOUT (${HARD_TIMEOUT}s, logs "HARD TIMEOUT") and the stall watchdog (idle tree + a
# descendant ≥${STALL_CHILD_AGE}s → "STALL … killing early"), BOTH of which are LOUD. It replaces a
# silent 10-minute rc=0 with two noisy bounds that already exist. A live background agent keeps the
# tree non-idle, so the stall watchdog cannot false-fire on it.
export CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS="${CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS:-0}"

# LANE-DEGRADATION NOTICE — emit the ledger accumulated by the probe loops (see their header).
# HERE, not at the probe: this point is after EVERY early exit (kill switch, overlap yield, dry
# run, no-model refusal) and after the one-shot override, so a fire that does not actually run
# can never post a notice, and the notice names the model the iteration is ABOUT to use rather
# than the one it was configured with. Two channels, mirroring the driver's four existing
# report sites: `ailang messages` (controlplane) + the bookkeeping issue.
# NOT fail-closed on the post: aborting the iteration would make GitHub availability a hard
# dependency of every fire. Instead, a post failure is itself LOUD in the driver log — the one
# thing the old code never was.
if [ -n "$_lane_degraded" ]; then
  _deg_body="**Executor/planner lane degraded on this fire** — recorded before the iteration ran.
${_lane_degraded}

Controller: \`${MODEL}\` (${MODEL_WHY}). Effective roles now: designer=\`${MISSION_DESIGNER_MODEL}\` planner=\`${MISSION_PLANNER_MODEL}\` executor=\`${MISSION_EXECUTOR_MODEL}\` evaluator=\`${MISSION_EVALUATOR_MODEL}\`.
Driver log: \`${LOG}\`. If this repeats across fires, the lane is down — check the bucket, and check that this mission's plist carries a PATH that reaches the CLI (the World mission lost five iterations to exactly that)."
  log "LANE DEGRADED this fire:$(printf '%s' "$_lane_degraded" | tr '\n' ' ')"
  _mc_notify "Mission ${MISSION_NAME}: executor/planner lane degraded" "$_deg_body" "lane-degradation"
fi

# DRIVER-PIN NOTICE — same site and same reasoning as the lane notice above: after every early
# exit, so a fire that does not run cannot post. Emitted only when the pin actually FAILED, i.e.
# only when stale code really did run. Source-clone drift is reported separately below because it
# is hazardous to interactive sessions; a persisted doubling threshold replaces posting it every
# 90 minutes, which would train the channel to be ignored.
if [ -n "$_pin_degraded" ]; then
  _pin_body="**Driver ran UNPINNED on this fire** — recorded before the iteration ran.
${_pin_degraded}

Mission \`${MISSION_NAME}\`. Driver log: \`${LOG}\`. The fire still ran; only its code provenance is
unknown. Fix: reconcile that clone with \`origin/dev\`, or find why the fetch failed. Until then
every fire silently runs whatever that working tree happens to hold — the class \`#558\` tracks,
measured twice (2026-08-03 \`#556\`, 2026-08-12 \`564cc4640\`)."
  _mc_notify "Mission ${MISSION_NAME}: driver ran UNPINNED (code provenance unknown)" "$_pin_body" "driver-pin"
fi

if [ -n "$_pin_drift_degraded" ]; then
  _pin_drift_body="**Pinned driver held, but the source clone drifted** — this fire ran correctly pinned.

Source clone: \`${AILANG_DRIVER_SRC:-$REPO}\`. Drift: **${_pin_drift_degraded} commits behind** \`origin/dev\`.
The hazard is an interactive session started in that clone: it resolves that clone's own
\`.claude/skills/\` and \`design_docs/\`, not the pinned driver's copies. Fix: reconcile
\`${AILANG_DRIVER_SRC:-$REPO}\` with \`origin/dev\`. This notice repeats only when the measured
drift doubles.

\`\$REPO\` is NOT the path to reconcile: pin-root.sh exports MISSION_WORKDIR=<pin worktree> before
re-execing, and REPO is derived from it, so on the pinned pass REPO names the throwaway worktree —
whose drift is 0 by construction. AILANG_DRIVER_SRC is the source clone."
  _mc_notify "Mission ${MISSION_NAME}: pinned source clone drifted (${_pin_drift_degraded} behind)" "$_pin_drift_body" "pin-drift"
fi

log "=== mission iteration starting (controller=$CONTROLLER_ID via ${MODEL_WHY}, timeout=${HARD_TIMEOUT}s | bg-wait-ceiling=${CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS}ms | roles: designer=$MISSION_DESIGNER_MODEL planner=$MISSION_PLANNER_MODEL executor=$MISSION_EXECUTOR_MODEL evaluator=$MISSION_EVALUATOR_MODEL) ==="

PROMPT="Run one mission-control iteration: invoke the mission-control skill for \
${MISSION_DOC} and follow its gates. You are a scheduled run; \
there is no human present — park anything needing human input and report via \
ailang messages and the GitHub bookkeeping issue, per the skill. \
The authoritative runtime instructions are .claude/skills/mission-control/SKILL.md; \
read and follow that file even when the controller provider is Codex. \
This prompt carries the operator's standing request for this run, written in advance \
because the run is unattended: USE THE AGENT TOOL to spawn the designer, planner, executor \
and evaluator roles exactly as the skill's routing table specifies. Treat that as the user \
asking for sub-agents, so any standing instruction to avoid the Agent tool unless the user \
asks is satisfied here and does not apply. The evaluator in particular is REQUIRED: \
generator-not-equal-judge is a non-negotiable property of this loop, and an iteration that \
lands work on the controller's own verdict has no independent review at all. If a role \
genuinely cannot be spawned, record WHICH role, the error, and the fallback you used in \
the routing block; do not silently proceed without a judge."

# _mc_run_once → runs the selected provider with BOTH watchdogs, waits, sets global RC.
# Watchdogs are per-attempt (fresh PIDs each retry).
_mc_run_once() {
  if [ "$CONTROLLER_PROVIDER" = "codex" ]; then
    codex exec --skip-git-repo-check \
      --dangerously-bypass-approvals-and-sandbox \
      --model "$MODEL" -C "$REPO" "$PROMPT" >>"$LOG" 2>&1 &
  else
    claude -p "$PROMPT" \
      --model "$MODEL" \
      --permission-mode bypassPermissions \
      >>"$LOG" 2>&1 &
  fi
  CONTROLLER_PID=$!
  printf '%s\n' "$CONTROLLER_PID" > "$PIDFILE"   # overlap guard reads this (per-attempt: retries refresh it)

  # Watchdog: TERM at the wall limit, KILL 60s later. (No GNU timeout on macOS.)
  (
    sleep "$HARD_TIMEOUT"
    if kill -0 "$CONTROLLER_PID" 2>/dev/null; then
      echo "[$(date '+%F %H:%M:%S')] HARD TIMEOUT ${HARD_TIMEOUT}s — killing $CONTROLLER_PID" >>"$LOG"
      kill -TERM "$CONTROLLER_PID" 2>/dev/null; sleep 60; kill -KILL "$CONTROLLER_PID" 2>/dev/null
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
    while kill -0 "$CONTROLLER_PID" 2>/dev/null; do
      if _mc_stalled "$CONTROLLER_PID"; then hits=$((hits + 1)); else hits=0; fi
      if [ "$hits" -ge "$STALL_SAMPLES" ]; then
        echo "[$(date '+%F %H:%M:%S')] STALL: $CONTROLLER_PROVIDER $CONTROLLER_PID idle with a descendant alive ≥${STALL_CHILD_AGE}s across $STALL_SAMPLES samples (unbounded poll loop?) — killing early" >>"$LOG"
        kill -TERM "$CONTROLLER_PID" 2>/dev/null; sleep 30; kill -KILL "$CONTROLLER_PID" 2>/dev/null
        break
      fi
      sleep "$STALL_INTERVAL"
    done
  ) &
  STALL_PID=$!

  wait "$CONTROLLER_PID"; RC=$?
  kill "$WATCHDOG_PID" "$STALL_PID" 2>/dev/null
  return "$RC"
}

# Snapshot the mission log's last record heading before the run, so the rc!=0
# notice below can tell "iteration lost" from "iteration recorded itself, then a
# watchdog killed a lingering child" (iter-145, 2026-08-05: report landed 14:35,
# SIGTERM 14:41, and the FAILED comment sent a human to re-check landed work).
MISSION_LOG_FILE="${MISSION_DOC%.md}-log.md"
pre_last_record=$(grep '^## ' "$MISSION_LOG_FILE" 2>/dev/null | tail -1)

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
  post_last_record=$(grep '^## ' "$MISSION_LOG_FILE" 2>/dev/null | tail -1)
  if [ -n "$post_last_record" ] && [ "$post_last_record" != "$pre_last_record" ]; then
    # The mission log gained a record during this run: the work landed and the
    # non-zero exit is a late kill of a lingering child, not a lost iteration.
    log "iteration exited rc=$RC AFTER recording itself — late kill, work landed"
    ailang messages send controlplane \
      "mission-control iteration exited rc=$RC AFTER its mission-log record landed (late watchdog kill of a lingering child, not a lost iteration). Record: ${post_last_record:0:160}. Log: $LOG" \
      --title "Mission iteration killed post-record (rc=$RC) — work landed" --from "$MSG_FROM" 2>/dev/null
    [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
      --body "ℹ️ Mission iteration exited **rc=$RC after landing its record** at $(date '+%F %H:%M %Z') — the mission log gained an entry during this run, so this was a late watchdog kill of a lingering child, not a lost iteration. The queue advanced normally. Log on the rig: \`$LOG\`." 2>/dev/null
  else
    log "iteration exited rc=$RC"
    ailang messages send controlplane \
      "mission-control iteration exited rc=$RC (timeout or crash). Log: $LOG" \
      --title "Mission iteration FAILED (rc=$RC)" --from "$MSG_FROM" 2>/dev/null
    [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
      --body "⚠️ Mission iteration **FAILED to complete** (rc=$RC — timeout or crash) at $(date '+%F %H:%M %Z'). Log on the rig: \`$LOG\`. The queue is untouched; the next interval will retry." 2>/dev/null
  fi
else
  log "iteration complete (rc=0)"
  # The skill itself sends the substantive report (Gate 5, both channels).
fi
exit "$RC"
