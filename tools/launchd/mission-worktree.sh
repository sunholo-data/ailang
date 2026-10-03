#!/bin/bash
# mission-worktree.sh — create a mission worktree that is never read half-built.
#
#   mission-worktree.sh add BRANCH SHA PATH   start the checkout in the background; prints PATH
#   mission-worktree.sh wait PATH [MAX_S]     poll ≤MAX_S (default 45): prints ready | failed:<why> | pending
#                                             exit 0 ready, 1 failed, 75 pending (call again)
#
# m-mission-light-profile Phase 1. A full checkout is ~25,800 files and takes minutes; every
# controller's shell tool kills a long foreground command, and git keeps checking out after the
# tool returns. Docs iteration 17 (2026-10-01) then read a tree with 25,786 staged deletions;
# V1 iteration 234 (2026-08-20) removed the live index.lock and killed a checkout. The skill had
# prose rules for this since 2026-08-20 and a controller still ran the add in the foreground —
# so the guard is this script, with two short calls instead of one long one.
#
# The background job enforces MISSION_WORKTREE_TIMEOUT (default 900 s) and, on timeout, a failed
# add or a non-empty `git status`, removes the partial worktree and its branch and reports
# failed:<why>. A PATH under /tmp is refused (gate-3-route.md: /tmp trees fail tests for their
# location, not their code).
set -uo pipefail
STATE="${MISSION_WORKTREE_STATE:-$HOME/.ailang/state/worktrees}"
mkdir -p "$STATE"
status_file() { printf '%s/%s.status' "$STATE" "$(printf '%s' "$1" | tr '/' '_')"; }

case "${1:-}" in
  add)
    BRANCH="${2:-}"; SHA="${3:-}"; WT="${4:-}"
    [ -n "$BRANCH" ] && [ -n "$SHA" ] && [ -n "$WT" ] || { echo "usage: mission-worktree.sh add BRANCH SHA PATH" >&2; exit 2; }
    case "$WT" in /tmp/*|/private/tmp/*) echo "failed:path_under_tmp ($WT) — use a sibling of the repo, e.g. ../.wt-<mission>-<item>" >&2; exit 1 ;; esac
    [ -e "$WT" ] && { echo "failed:path_exists ($WT)" >&2; exit 1; }
    REPO=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "failed:not_in_a_repo" >&2; exit 1; }
    SF=$(status_file "$WT"); echo pending > "$SF"
    LIMIT="${MISSION_WORKTREE_TIMEOUT:-900}"
    nohup /bin/bash -c '
      repo="$1"; br="$2"; sha="$3"; wt="$4"; sf="$5"; limit="$6"
      cleanup() { git -C "$repo" worktree remove --force "$wt" >/dev/null 2>&1; rm -rf "$wt"
                  git -C "$repo" worktree prune >/dev/null 2>&1; git -C "$repo" branch -D "$br" >/dev/null 2>&1; }
      git -C "$repo" worktree add -q -b "$br" "$wt" "$sha" >/dev/null 2>&1 & p=$!
      s=0; while kill -0 "$p" 2>/dev/null; do
        [ "$s" -ge "$limit" ] && { kill "$p" 2>/dev/null; wait "$p" 2>/dev/null; cleanup; echo "failed:timeout_${limit}s" > "$sf"; exit 0; }
        sleep 2; s=$((s+2)); done
      wait "$p"; rc=$?
      [ "$rc" -eq 0 ] || { cleanup; echo "failed:worktree_add_rc_$rc" > "$sf"; exit 0; }
      n=$(git -C "$wt" status --porcelain 2>/dev/null | wc -l | tr -d " ")
      [ "$n" = 0 ] || { cleanup; echo "failed:not_clean_after_add ($n entries)" > "$sf"; exit 0; }
      echo ready > "$sf"
    ' _ "$REPO" "$BRANCH" "$SHA" "$WT" "$SF" "$LIMIT" >/dev/null 2>&1 &
    echo "$WT"
    ;;
  wait)
    WT="${2:-}"; MAX="${3:-45}"
    [ -n "$WT" ] || { echo "usage: mission-worktree.sh wait PATH [MAX_S]" >&2; exit 2; }
    SF=$(status_file "$WT"); [ -f "$SF" ] || { echo "failed:unknown_worktree"; exit 1; }
    s=0
    while :; do
      v=$(cat "$SF" 2>/dev/null)
      case "$v" in
        ready) echo ready; rm -f "$SF"; exit 0 ;;
        failed:*) echo "$v"; rm -f "$SF"; exit 1 ;;
      esac
      [ "$s" -ge "$MAX" ] && { echo pending; exit 75; }
      sleep 1; s=$((s+1))
    done
    ;;
  *) sed -n '2,6p' "$0" >&2; exit 2 ;;
esac
