#!/usr/bin/env bash
# rig-lock.sh — shared mutual-exclusion for ALL local-rig eval jobs
# (M-EVAL-OS-CONTINUOUS-ROTATION). The rig is a single GPU / bandwidth-bound box;
# concurrent runs thrash and an ollama model reload mid-run silently kills a stream.
# Every rig job (nightly-eval, nightly-lang-eval, os-rotation-filler) takes this
# lock so they never overlap. Scheduled jobs wait; the background filler yields.
#
# macOS has no flock(1), so this uses an atomic mkdir lock with dead-owner recovery.
#
# Usage:
#   source "$(dirname "$0")/rig-lock.sh"
#   rig_lock_acquire wait    # block until free (scheduled jobs)
#   rig_lock_acquire nowait  # return 1 immediately if held (background filler)
# The lock is auto-released on process exit (EXIT trap).

RIG_LOCK_DIR="${RIG_LOCK_DIR:-$HOME/.ailang/state/rig.lock.d}"
# Priority requests live in the sibling .priority directory, named by requester PID.

rig_lock_eval_binary() {
  if [ -n "${AILANG_EVAL_BIN:-}" ]; then printf '%s\n' "$AILANG_EVAL_BIN"
  elif [ -x "$HOME/.local/share/ailang/rig-priority/ailang" ]; then
    local pinned current
    current=$(cat std/VERSION 2>/dev/null) || return 1
    if [ -x "$HOME/.local/share/ailang/rig-priority/$current/ailang" ]; then
      printf '%s\n' "$HOME/.local/share/ailang/rig-priority/$current/ailang"
      return 0
    fi
    pinned=$(cat "$HOME/.local/share/ailang/rig-priority/VERSION" 2>/dev/null) || return 1
    if [ "$pinned" != "$current" ]; then
      echo "riglock: priority runner version $pinned differs from checkout $current; rebuild the local runner before evaluating" >&2
      return 1
    fi
    printf '%s\n' "$HOME/.local/share/ailang/rig-priority/ailang"
  else printf 'ailang\n'; fi
}

rig_lock_priority_pending() {
  local request pid
  for request in "$RIG_LOCK_DIR.priority"/*; do
    [ -f "$request" ] || continue
    pid=${request##*/}
    case "$pid" in ''|*[!0-9]*) continue;; esac
    if kill -0 "$pid" 2>/dev/null; then return 0; fi
    rm -f "$request"
  done
  return 1
}

rig_lock_release() {
  # A child may be lending this lock to priority work. Never remove its lock
  # if the parent is terminated during that handoff.
  local owner
  read -r owner _ < "$RIG_LOCK_DIR/holder" 2>/dev/null || owner=""
  [ "$owner" != "$$" ] || rm -rf "$RIG_LOCK_DIR"
  unset AILANG_RIG_LOCK_HELD AILANG_RIG_LOCK_OWNER_PID
}

# Call only between completed units, never while an inference/tool is in flight.
rig_lock_yield() {
  rig_lock_priority_pending || return 0
  local owner
  read -r owner _ < "$RIG_LOCK_DIR/holder" || return 1
  [ "$owner" = "$$" ] || return 1
  echo "riglock: yielded at a safe boundary for priority work" >&2
  rig_lock_release
  rig_lock_acquire wait
  echo "riglock: priority work finished; evaluation resumed" >&2
}

rig_lock_acquire() {
  local mode="${1:-wait}"
  mkdir -p "$(dirname "$RIG_LOCK_DIR")" 2>/dev/null
  while true; do
    if rig_lock_priority_pending; then
      [ "$mode" != "nowait" ] || return 1
      sleep 1
      continue
    fi
    if mkdir "$RIG_LOCK_DIR" 2>/dev/null; then
      if rig_lock_priority_pending; then
        rmdir "$RIG_LOCK_DIR"
        continue
      fi
      break
    fi
    # Multi-hour evals are legitimate. Reclaim only a positively dead owner.
    local holder_pid
    holder_pid=$(awk '{for(i=1;i<=NF;i++) if($i ~ /^pid=[0-9]+$/){sub(/^pid=/,"",$i);print $i;exit} if($1 ~ /^[0-9]+$/)print $1}' "$RIG_LOCK_DIR/holder" 2>/dev/null || true)
    if [ -n "$holder_pid" ] && ! kill -0 "$holder_pid" 2>/dev/null; then
      rm -rf "$RIG_LOCK_DIR"
      continue
    fi
    if [ "$mode" = "nowait" ]; then
      return 1
    fi
    sleep 30
  done
  echo "$$ $(date -u +%FT%TZ)" > "$RIG_LOCK_DIR/holder" 2>/dev/null || true
  # Tell descendant `ailang eval-suite` processes that an ancestor already holds
  # the rig lock, so its native riglock.Acquire (internal/riglock) is a no-op and
  # does not deadlock against this wrapper's lock. Must match riglock.EnvHeld.
  export AILANG_RIG_LOCK_HELD=1
  export AILANG_RIG_LOCK_OWNER_PID=$$
  trap rig_lock_release EXIT
  return 0
}

# rig_in_blackout START END  — true if the current HH:MM is within [START,END)
# (24h "HH:MM"). Used by the filler to avoid the nightly windows + model reloads.
rig_in_blackout() {
  local now start end
  now=$(date +%H%M); start=${1//:/}; end=${2//:/}
  if [ "$start" -le "$end" ]; then
    [ "$now" -ge "$start" ] && [ "$now" -lt "$end" ]
  else # window wraps midnight
    [ "$now" -ge "$start" ] || [ "$now" -lt "$end" ]
  fi
}
