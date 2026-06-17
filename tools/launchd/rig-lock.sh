#!/usr/bin/env bash
# rig-lock.sh — shared mutual-exclusion for ALL local-rig eval jobs
# (M-EVAL-OS-CONTINUOUS-ROTATION). The rig is a single GPU / bandwidth-bound box;
# concurrent runs thrash and an ollama model reload mid-run silently kills a stream.
# Every rig job (nightly-eval, nightly-lang-eval, os-rotation-filler) takes this
# lock so they never overlap. Scheduled jobs wait; the background filler yields.
#
# macOS has no flock(1), so this uses an atomic mkdir lock with a staleness steal.
#
# Usage:
#   source "$(dirname "$0")/rig-lock.sh"
#   rig_lock_acquire wait    # block until free (scheduled jobs)
#   rig_lock_acquire nowait  # return 1 immediately if held (background filler)
# The lock is auto-released on process exit (EXIT trap).

RIG_LOCK_DIR="${RIG_LOCK_DIR:-$HOME/.ailang/state/rig.lock.d}"
RIG_LOCK_STALE_MIN="${RIG_LOCK_STALE_MIN:-360}" # steal a lock older than 6h (crash recovery)

rig_lock_acquire() {
  local mode="${1:-wait}"
  mkdir -p "$(dirname "$RIG_LOCK_DIR")" 2>/dev/null
  while ! mkdir "$RIG_LOCK_DIR" 2>/dev/null; do
    # Steal a stale lock (holder crashed without releasing).
    if [ -d "$RIG_LOCK_DIR" ] && [ -n "$(find "$RIG_LOCK_DIR" -maxdepth 0 -mmin +"$RIG_LOCK_STALE_MIN" 2>/dev/null)" ]; then
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
  # shellcheck disable=SC2064
  trap "rm -rf '$RIG_LOCK_DIR'; unset AILANG_RIG_LOCK_HELD" EXIT
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
