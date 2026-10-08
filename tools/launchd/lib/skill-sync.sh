#!/bin/bash
# Shared skill checkout refresh, portable to bash 3.2. Source pin-root.sh first.
# mc_skill_sync apply|report always returns 0; outputs SKILL_SYNC_STATUS/NOTE.
# AILANG_SKILL_SYNC=0 disables all Git inspection (default: enabled).
# AILANG_SKILL_SYNC_CHECKOUT overrides the mission-control skill symlink target.
# No fetch: pin-root refreshes shared refs. Every Git call has a fixed 5s cap.
# Report uses optional-lock-free reads and writes only private temporary scratch.
# Final preflight is not atomic: Git's own locks/refusals remain the last defence.

_ss_verdict() { SKILL_SYNC_STATUS="$1"; SKILL_SYNC_NOTE="$2"; }
_ss_git() {
  local rc=0
  # Raw output stays in scratch: command substitution would destroy NUL records.
  _pin_bounded 5 /bin/bash -c 'out=$1; err=$2; shift 2; exec env GIT_OPTIONAL_LOCKS=0 git "$@" >"$out" 2>"$err"' bash "$ss_out" "$ss_err" -C "$ss_checkout" "$@" || rc=$?
  if [ "$rc" -ne 0 ]; then
    if [ "$rc" -eq 124 ]; then
      _ss_verdict error:git-timeout 'bounded Git call expired; checkout may have advanced if merge started'
    else
      _ss_verdict error:git-failed 'Git inspection failed'
    fi
    return "$rc"
  fi
  return 0
}
_ss_path() {
  _ss_git rev-parse --git-path "$1" || return 1
  ss_path=$(cat "$ss_out")
  case "$ss_path" in /*) ;; *) ss_path="$ss_checkout/$ss_path" ;; esac
}
_ss_safety() {
  local marker
  for marker in rebase-merge rebase-apply MERGE_HEAD CHERRY_PICK_HEAD REVERT_HEAD sequencer BISECT_LOG BISECT_START; do
    _ss_path "$marker" || return 1
    if [ -e "$ss_path" ]; then
      _ss_verdict skip:operation-in-progress 'worktree operation in progress'; return 1
    fi
  done
  _ss_path index.lock || return 1
  ss_lock="$ss_path"
  if [ -e "$ss_lock" ]; then
    _ss_verdict skip:index-locked 'index lock held; no retry'; return 1
  fi
}
_ss_snapshot() {
  _ss_git symbolic-ref --quiet --short HEAD || { [ "$SKILL_SYNC_STATUS" = error:git-timeout ] || _ss_verdict skip:not-dev 'branch is not dev'; return 1; }
  ss_branch=$(cat "$ss_out")
  if [ "$ss_branch" != dev ]; then
    _ss_verdict skip:not-dev 'branch is not dev'; return 1
  fi
  _ss_git rev-parse --verify 'HEAD^{commit}' || return 1
  ss_head=$(cat "$ss_out")
  _ss_git rev-parse --verify 'origin/dev^{commit}' || { [ "$SKILL_SYNC_STATUS" = error:git-timeout ] || _ss_verdict error:ref-unavailable 'origin/dev is unavailable'; return 1; }
  ss_tip=$(cat "$ss_out")
  _ss_git rev-list --count origin/dev..HEAD || return 1
  ss_ahead=$(cat "$ss_out")
  if [ "$ss_ahead" != 0 ]; then
    _ss_verdict skip:ahead 'checkout has local commits'; return 1
  fi
  _ss_git rev-list --count HEAD..origin/dev || return 1
  ss_behind=$(cat "$ss_out")
}
_ss_collisions() {
  local record path other code d incoming hit prior
  local incoming_paths=() dirty_paths=() collisions=()
  _ss_git diff --name-status -z -M -C "$ss_head" "$ss_tip" || return 1
  while IFS= read -r -d '' code; do
    IFS= read -r -d '' path || { _ss_verdict error:path-stream 'incomplete diff path stream'; return 1; }
    incoming_paths+=("$path") # incoming first side
    case "$code" in R*|C*)
      IFS= read -r -d '' other || { _ss_verdict error:path-stream 'incomplete diff rename stream'; return 1; }
      incoming_paths+=("$other") # incoming second side
      ;; esac
  done < "$ss_out"
  _ss_git status --porcelain=v1 -z --untracked-files=all || return 1
  while IFS= read -r -d '' record; do
    code=${record:0:2}; path=${record:3}
    dirty_paths+=("$path") # dirty destination (also ordinary paths)
    case "$code" in *R*|*C*)
      IFS= read -r -d '' other || { _ss_verdict error:path-stream 'incomplete status rename stream'; return 1; }
      dirty_paths+=("$other") # dirty source
      ;; esac
  done < "$ss_out"
  for d in ${dirty_paths[@]+"${dirty_paths[@]}"}; do
    hit=0
    for incoming in ${incoming_paths[@]+"${incoming_paths[@]}"}; do
      if [ "$d" = "$incoming" ] || [[ "$d" = "$incoming/"* || "$incoming" = "$d/"* ]]; then hit=1; fi
    done
    [ "$hit" = 1 ] || continue
    prior=0
    for path in ${collisions[@]+"${collisions[@]}"}; do [ "$path" != "$d" ] || prior=1; done
    [ "$prior" = 0 ] || continue
    collisions+=("$d")
  done
  if [ "${#collisions[@]}" -gt 0 ]; then
    _ss_verdict skip:dirty-range "colliding paths=${#collisions[@]}"; return 1
  fi
}
_ss_run() {
  local mode="$1" link root before_head before_tip before_branch rc after
  if [ "$mode" != apply ] && [ "$mode" != report ]; then
    _ss_verdict error:invalid-mode 'expected apply or report'; return 1
  fi
  if [ "${AILANG_SKILL_SYNC:-1}" = 0 ]; then
    _ss_verdict skip:disabled 'disabled by AILANG_SKILL_SYNC=0'; return 1
  fi
  if ! type _pin_bounded >/dev/null 2>&1; then
    _ss_verdict error:bounded-unavailable 'pin-root bounded primitive unavailable'; return 1
  fi
  ss_checkout=${AILANG_SKILL_SYNC_CHECKOUT:-}
  if [ -z "$ss_checkout" ]; then
    link="$HOME/.claude/skills/mission-control"
    if [ ! -L "$link" ]; then
      _ss_verdict skip:symlink-absent 'mission-control skill symlink absent'; return 1
    fi
    ss_checkout=$(readlink -f "$link" 2>/dev/null) || ss_checkout=''
    if [ -z "$ss_checkout" ] || [ ! -d "$ss_checkout" ]; then
      _ss_verdict skip:symlink-broken 'mission-control skill symlink broken'; return 1
    fi
  fi
  _ss_git rev-parse --show-toplevel || { [ "$SKILL_SYNC_STATUS" = error:git-timeout ] || _ss_verdict skip:not-worktree 'target is not a worktree'; return 1; }
  ss_checkout=$(cat "$ss_out")
  ss_checkout=$(cd "$ss_checkout" && pwd -P) || return 1
  if [ -n "${MC_DRIVER_ROOT:-}" ]; then
    root=$(cd "$MC_DRIVER_ROOT" && pwd -P) || { _ss_verdict error:driver-root 'driver root unreadable'; return 1; }
  else
    _ss_git -C "$PWD" rev-parse --show-toplevel || return 1
    root=$(cat "$ss_out")
    root=$(cd "$root" && pwd -P) || return 1
  fi
  if [ "$ss_checkout" = "$root" ]; then
    _ss_verdict skip:self-target 'target is the driver worktree'; return 1
  fi
  _ss_snapshot || return 1
  _ss_safety || return 1
  if [ "$ss_behind" = 0 ]; then
    _ss_verdict current 'checkout already at origin/dev'; return 1
  fi
  _ss_collisions || return 1
  if [ "$mode" = report ]; then
    _ss_verdict "synced:$ss_behind" "would-sync $ss_behind commits (report mode; checkout unchanged)"; return 1
  fi
  before_head=$ss_head; before_tip=$ss_tip; before_branch=$ss_branch
  _ss_snapshot || { case "$SKILL_SYNC_STATUS" in skip:*) _ss_verdict skip:state-changed 'snapshot changed before merge';; esac; return 1; }
  if [ "$before_head" != "$ss_head" ] || [ "$before_tip" != "$ss_tip" ] || [ "$before_branch" != "$ss_branch" ]; then
    _ss_verdict skip:state-changed 'snapshot changed before merge'; return 1
  fi
  _ss_safety || return 1
  _ss_collisions || return 1
  rc=0
  _ss_git merge --ff-only origin/dev || rc=$?
  if [ "$rc" -ne 0 ]; then
    [ "$rc" -ne 124 ] || return 1
    if [ -e "$ss_lock" ]; then
      _ss_verdict skip:index-locked 'Git refused held index lock; no retry'
    elif LC_ALL=C grep -Eq 'cannot lock ref|not possible to fast-forward|refusing to merge unrelated histories|local changes|untracked working tree|would be overwritten|another git process' "$ss_err"; then
      _ss_verdict skip:git-refused 'Git refused concurrent checkout/ref state; no retry'
    else
      _ss_verdict error:merge-failed 'ff-only merge failed'
    fi
    return 1
  fi
  _ss_git rev-parse --verify 'HEAD^{commit}' || return 1
  after=$(cat "$ss_out")
  _ss_git rev-list --count "$before_head..HEAD" || return 1
  if [ "$after" != "$before_tip" ] || [ "$(cat "$ss_out")" != "$ss_behind" ]; then
    _ss_verdict error:merge-unverified 'merge returned success but resulting HEAD/count changed'; return 1
  fi
  _ss_verdict "synced:$ss_behind" "fast-forwarded $ss_behind commits"
}
mc_skill_sync() {
  local ss_checkout ss_out ss_err ss_path ss_lock ss_branch ss_head ss_tip ss_ahead ss_behind scratch
  local pin_had=${PIN_BOUNDED_OUT+x} pin_saved=${PIN_BOUNDED_OUT:-}
  _ss_verdict error:setup-failed 'scratch setup failed'
  scratch=$(mktemp -d "${TMPDIR:-/tmp}/skill-sync.XXXXXX") || return 0
  ss_out="$scratch/out"; ss_err="$scratch/err"
  _ss_run "${1:-}" || :
  rm -rf "$scratch"
  if [ "$pin_had" = x ]; then PIN_BOUNDED_OUT=$pin_saved; else unset PIN_BOUNDED_OUT; fi
  return 0
}
