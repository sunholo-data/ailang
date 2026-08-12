#!/usr/bin/env bash
# pin-root.sh — make a launchd driver run COMMITTED code, and make failing to do so LOUD.
#
# THE CLASS THIS CLOSES (#558; measured twice — 2026-08-03 and 2026-08-12).
# launchd invokes each driver by absolute path into a shared, mutable clone that nothing
# keeps current, and everything the fire reads hangs off that one root:
#
#     REPO="${MISSION_WORKDIR:-$(cd "$(dirname "$0")/../.." && pwd)}"; cd "$REPO"
#
#   * the DRIVER itself         -> #556 retired qwen3.5 and the nightly ran it 24/24 two days
#                                  later; 564cc4640's lane fix was inert on V1 while both
#                                  sibling missions had it
#   * the SKILL (.claude/skills) -> stale skill, iteration 128
#   * the CHARTER (design_docs)  -> stale charter, iteration 129
#
# Three symptoms, one root. Patching any single artefact fixes one third of a bug, so this
# pins the ROOT: re-exec the driver out of a worktree pinned to a committed ref, which moves
# the script, the skill and the charter together in one move.
#
# WHY THE FAILURE PATH IS LOUD RATHER THAN SILENT. If the fetch fails and we quietly carry on
# from the working tree, we have rebuilt the exact defect 564cc4640 removed one layer up — a
# fallback whose only witness is a log nobody reads (Critical Principle 2). So a failed pin is
# NOT fatal (making network availability a hard dependency of every fire trades a rare silent
# staleness for a common loud outage) but it MUST be reported on the human channel by the
# caller. This file sets PIN_STATUS=STALE and returns 1; emitting is the caller's job, because
# only the caller knows its own early-exit points — a fire that never runs must never post.
#
# CONTRACT
#   source, never execute.  $0 stays the CALLER's path, which is what makes $0-relative
#   resolution below refer to the driver rather than to this file.
#
#   in    $0                          the driver's path (implicit)
#         AILANG_DRIVER_REF           ref to pin to           (default origin/dev)
#         AILANG_DRIVER_PIN=0         opt out entirely        (default on)
#         AILANG_DRIVER_FETCH_TIMEOUT bounded fetch, seconds  (default 120)
#         AILANG_DRIVER_PIN_DIR       override worktree path
#         MISSION_NAME                namespaces the worktree so missions never collide
#   out   PIN_STATUS  pinned | disabled | STALE
#         PIN_NOTE    one-line human summary, safe to log or post
#         PIN_DRIFT   how many commits the source clone is behind the ref ("?" if unknown)
#   exit  re-execs on success and NEVER RETURNS; returns 0 (already pinned / opted out) or
#         1 (STALE — caller continues on the working tree, loudly)
#
# CHICKEN AND EGG, stated rather than hidden: this block lives in the driver it protects, so it
# does nothing until the shared clone receives it once. That first reconcile is human (Principle
# 0 forbids unattended branch ops on a shared dirty tree). Afterwards it is self-maintaining.
#
# Portable to macOS bash 3.2.57 — no associative arrays, no ${v,,}, no GNU timeout.

PIN_STATUS="unknown"
PIN_NOTE=""
PIN_DRIFT="?"

# _pin_bounded SECONDS CMD... — hard wall-clock cap; rc = CMD's rc, or 124 on expiry.
# Deliberately duplicates mission-control.sh's _mc_bounded rather than depending on it: this
# file has to be sourceable by drivers that define no such helper, and the rig has no GNU
# timeout (mission-control.sh:37). An unbounded fetch here would hang the fire before the
# driver's own watchdog exists — the failure mode _mc_bounded was written for.
_pin_bounded() {
  local secs="$1"; shift
  local out_f rc deadline pid
  out_f=$(mktemp -t pin_bounded) || { PIN_BOUNDED_OUT=""; return 125; }
  ( exec "$@" ) >"$out_f" 2>&1 &
  pid=$!
  deadline=$(( $(date +%s) + secs ))
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
      kill "$pid" 2>/dev/null; sleep 1; kill -9 "$pid" 2>/dev/null
      PIN_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
      return 124
    fi
    sleep 1
  done
  wait "$pid"; rc=$?
  PIN_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
  return "$rc"
}

_pin_stale() {  # $1 = reason -> STALE, caller reports and continues on the working tree
  PIN_STATUS="STALE"
  PIN_NOTE="$1"
  return 1
}

# pin_root_to_committed_ref "$@" — pass the driver's own args so the re-exec forwards them.
pin_root_to_committed_ref() {
  if [ -n "${AILANG_DRIVER_PINNED:-}" ]; then
    # we ARE the re-exec'd copy; the values below crossed the exec in the environment
    PIN_STATUS="pinned"
    PIN_DRIFT="${AILANG_DRIVER_DRIFT:-?}"
    PIN_NOTE="running committed ${AILANG_DRIVER_REF:-origin/dev} @ ${AILANG_DRIVER_PINNED} from ${MISSION_WORKDIR:-?} (source clone ${AILANG_DRIVER_SRC:-?} was ${PIN_DRIFT} behind)"
    return 0
  fi

  if [ "${AILANG_DRIVER_PIN:-1}" = "0" ]; then
    PIN_STATUS="disabled"
    PIN_NOTE="pin disabled via AILANG_DRIVER_PIN=0 — running the working tree as-is"
    return 0
  fi

  local ref src script wt target short drift fetch_s rc
  ref="${AILANG_DRIVER_REF:-origin/dev}"
  fetch_s="${AILANG_DRIVER_FETCH_TIMEOUT:-120}"
  script=$(basename "$0")

  src=$(cd "$(dirname "$0")/../.." 2>/dev/null && pwd)
  if [ -z "$src" ]; then
    _pin_stale "cannot resolve the source clone from \$0=$0"; return 1
  fi
  if ! git -C "$src" rev-parse --git-dir >/dev/null 2>&1; then
    _pin_stale "source clone $src is not a git repository"; return 1
  fi

  _pin_bounded "$fetch_s" git -C "$src" fetch --quiet origin; rc=$?
  if [ "$rc" -eq 124 ]; then
    _pin_stale "git fetch origin exceeded ${fetch_s}s in $src"; return 1
  elif [ "$rc" -ne 0 ]; then
    _pin_stale "git fetch origin failed (rc=$rc) in $src: $(printf '%s' "${PIN_BOUNDED_OUT:-}" | tail -c 200 | tr '\n' ' ')"; return 1
  fi

  target=$(git -C "$src" rev-parse "$ref" 2>/dev/null)
  if [ -z "$target" ]; then
    _pin_stale "cannot resolve $ref in $src"; return 1
  fi
  short=$(git -C "$src" rev-parse --short "$target" 2>/dev/null)
  drift=$(git -C "$src" rev-list --count "HEAD..$ref" 2>/dev/null)
  [ -n "$drift" ] || drift="?"
  PIN_DRIFT="$drift"

  wt="${AILANG_DRIVER_PIN_DIR:-$HOME/.ailang-driver-pin/${MISSION_NAME:-$(basename "$script" .sh)}}"

  # Refresh (or create) the pin worktree. It is a throwaway checkout with no user work in it,
  # kept outside every dev clone, so --force can never overwrite in-progress changes.
  git -C "$src" worktree prune >/dev/null 2>&1
  if git -C "$src" worktree list --porcelain 2>/dev/null | grep -qx "worktree $wt"; then
    if ! git -C "$wt" checkout --quiet --detach --force "$target" 2>/dev/null; then
      _pin_stale "pin worktree checkout $short failed at $wt"; return 1
    fi
  else
    mkdir -p "$(dirname "$wt")"
    if ! git -C "$src" worktree add --quiet --detach --force "$wt" "$target" 2>/dev/null; then
      _pin_stale "pin worktree add $short failed at $wt"; return 1
    fi
  fi

  # Refuse to exec into a driver the ref does not have — a rename on dev would otherwise turn
  # a pin into a silent no-run. Control for the whole worktree step in one assertion.
  if [ ! -f "$wt/tools/launchd/$script" ]; then
    _pin_stale "$ref has no tools/launchd/$script — refusing to re-exec into a missing driver"; return 1
  fi

  # MISSION_WORKDIR moves too, and that is the point: mission-control.sh:40 reads it AHEAD of
  # $0-relative resolution, so pinning only the script would leave motoko and World (whose env
  # files pin MISSION_WORKDIR — mission-motoko.env:8, mission-world.env:5) running a pinned
  # driver against a stale charter and skill. That half-fix reports green, which is worse than
  # no fix. Sprint worktrees are unaffected: the skill creates them by absolute path
  # (mission-control SKILL.md:1662), not relative to cwd.
  AILANG_DRIVER_PINNED="$short"
  AILANG_DRIVER_SRC="$src"
  AILANG_DRIVER_DRIFT="$drift"
  AILANG_DRIVER_REF="$ref"
  MISSION_WORKDIR="$wt"
  export AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_DRIFT AILANG_DRIVER_REF MISSION_WORKDIR

  exec /bin/bash "$wt/tools/launchd/$script" "$@"
}
