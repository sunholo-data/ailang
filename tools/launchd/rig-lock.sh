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
RIG_HANDOFF_FILE="${RIG_HANDOFF_FILE:-$HOME/.ailang/state/rig.handoff}"
RIG_YIELD_WINDOW_SEC="${RIG_YIELD_WINDOW_SEC:-180}"

# --- cooperative yield (M-RIG-LOCK-YIELD) -----------------------------------
# The shell half of internal/riglock/yield.go; read that file's header for why
# this exists. Same marker, same format, same expiry rule, so a shell requester
# and a Go holder (or the reverse) speak to each other.
#
# The marker deliberately lives OUTSIDE $RIG_LOCK_DIR. A holder that yields
# REMOVES the lock directory, so anything stored inside it would vanish at the
# exact moment it is needed, and the gap would read as an ordinary free lock —
# which the 45-minute background filler would win. NoWait acquirers that are not
# the named requester are refused for as long as a handoff is in force.

# rig_yield_pending [requester] — true if a handoff is in force. With a
# requester argument, true only if the handoff belongs to SOMEONE ELSE.
# An expired marker, or one whose requester pid is gone, is removed and
# reported absent: a stuck marker would refuse every NoWait acquirer until a
# human noticed, which is worse than the contention it prevents.
rig_yield_pending() {
  local mine="${1:-}" line req pid until_s now_s until_epoch
  line=$(head -1 "$RIG_HANDOFF_FILE" 2>/dev/null) || return 1
  [ -n "$line" ] || return 1
  req=$(printf '%s' "$line" | tr ' ' '\n' | sed -n 's/^requester=//p' | head -1) || true
  pid=$(printf '%s' "$line" | tr ' ' '\n' | sed -n 's/^pid=//p' | head -1) || true
  until_s=$(printf '%s' "$line" | tr ' ' '\n' | sed -n 's/^until=//p' | head -1) || true
  # No requester or no deadline = unparseable. Do not guess at an unbounded grant.
  if [ -z "$req" ] || [ -z "$until_s" ]; then
    rm -f "$RIG_HANDOFF_FILE" 2>/dev/null; return 1
  fi
  # `date -j -f ... -u` parses the RFC3339 stamp as UTC, which is what it is.
  # Parsing it in the local zone is the CEST bug daneel already paid for once.
  until_epoch=$(TZ=UTC date -j -u -f '%Y-%m-%dT%H:%M:%SZ' "$until_s" +%s 2>/dev/null) || true
  now_s=$(date -u +%s)
  if [ -z "$until_epoch" ] || [ "$now_s" -ge "$until_epoch" ]; then
    rm -f "$RIG_HANDOFF_FILE" 2>/dev/null; return 1
  fi
  if [ -n "$pid" ] && ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$RIG_HANDOFF_FILE" 2>/dev/null; return 1
  fi
  [ -n "$mine" ] && [ "$req" = "$mine" ] && return 1
  return 0
}

# rig_lock_request_yield REQUESTER [WINDOW_SEC] — ask the holder to step aside
# at its next checkpoint. Does NOT acquire; the caller then races for the lock
# normally, passing its own name to rig_lock_acquire so this guard lets it in.
# Refuses to overwrite a handoff owned by someone else: two short jobs may queue
# behind one holder, but the second must not inherit the first one's grant.
rig_lock_request_yield() {
  local requester="${1:?rig_lock_request_yield needs a requester name}"
  local window="${2:-$RIG_YIELD_WINDOW_SEC}"
  rig_yield_pending "$requester" && return 1
  mkdir -p "$(dirname "$RIG_HANDOFF_FILE")" 2>/dev/null
  printf 'requester=%s pid=%s until=%s\n' "$requester" "$$" \
    "$(date -u -v+"${window}"S +%FT%TZ 2>/dev/null || date -u -d "+${window} seconds" +%FT%TZ)" \
    > "$RIG_HANDOFF_FILE"
}

# rig_lock_clear_yield REQUESTER — hand the GPU back EARLY. The holder is
# blocked polling for this; skipping it costs it the whole remaining window.
rig_lock_clear_yield() {
  local requester="${1:?rig_lock_clear_yield needs a requester name}" req
  req=$(head -1 "$RIG_HANDOFF_FILE" 2>/dev/null | tr ' ' '\n' | sed -n 's/^requester=//p' | head -1) || true
  [ "$req" = "$requester" ] && rm -f "$RIG_HANDOFF_FILE" 2>/dev/null
  return 0
}

# rig_lock_acquire MODE [REQUESTER]
rig_lock_acquire() {
  local mode="${1:-wait}" requester="${2:-}"
  mkdir -p "$(dirname "$RIG_LOCK_DIR")" 2>/dev/null
  # Honour a handoff to someone else BEFORE racing for the directory: the gap a
  # yielding holder opens belongs to the named requester, not to whoever polls
  # fastest.
  while rig_yield_pending "$requester"; do
    [ "$mode" = "nowait" ] && return 1
    sleep 2
  done
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

# rig_time_minus HH:MM MINUTES — print HH:MM shifted MINUTES earlier, wrapping
# at midnight. Lets a caller express "the hour before the nightly starts" from
# the nightly's own start time, instead of hard-coding a second constant that
# then drifts when the schedule moves.
rig_time_minus() {
  local hhmm="$1" mins="$2" h m total
  h=${hhmm%%:*}; m=${hhmm##*:}
  # Strip a leading zero before arithmetic: bash reads 08 and 09 as invalid octal.
  total=$(( 10#$h * 60 + 10#$m - mins ))
  while [ "$total" -lt 0 ]; do total=$(( total + 1440 )); done
  total=$(( total % 1440 ))
  printf '%02d:%02d' $(( total / 60 )) $(( total % 60 ))
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
