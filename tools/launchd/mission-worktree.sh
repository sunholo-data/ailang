#!/bin/bash
# mission-worktree.sh — create a mission worktree that is never read half-built.
#
#   mission-worktree.sh add BRANCH SHA PATH   start a background checkout; prints PATH
#   mission-worktree.sh wait PATH [MAX_S]     prints ready | failed:<why> | pending
#     exit 0 ready, 1 failed, 75 pending (call again)
#     failures include failed:writer_died and failed:owner_never_published
#   Bare pending expires after MISSION_WORKTREE_OWNER_GRACE seconds (default 60).
#
# A full checkout can outlive the controller's shell tool. Detach the writer and register
# its identity before returning; wait recovers abandoned pending state after writer death.
# /tmp trees are refused because their location breaks tests (gate-3-route.md).
set -uo pipefail
STATE="${MISSION_WORKTREE_STATE:-$HOME/.ailang/state/worktrees}"
mkdir -p "$STATE"
status_file() { printf '%s/%s.status' "$STATE" "$(printf '%s' "$1" | tr '/' '_')"; }
birth() { LC_ALL=C ps -p "$1" -o lstart= 2>/dev/null | awk '{$1=$1;print}'; }
pgid() { ps -p "$1" -o pgid= 2>/dev/null | tr -d ' '; }
cleanup() {
  git -C "$REPO" worktree remove --force "$WT" >/dev/null 2>&1; rm -rf "$WT"
  git -C "$REPO" worktree prune >/dev/null 2>&1; git -C "$REPO" branch -D "$BRANCH" >/dev/null 2>&1
}
consume() { rm -f "$SF" "$SF.owner" "$SF.started" "$SF.child" "$SF.go"; }
fail_pending() { [ "$(cat "$SF" 2>/dev/null)" != pending ] || echo "failed:$1" > "$SF"; }
terminal() {
  case "$1" in
    ready) echo ready; consume; exit 0 ;;
    failed:*) echo "$1"; consume; exit 1 ;;
  esac
}
# Return alive=0, dead=1, inconclusive=2. An inspection failure is not death.
identity() {
  local pid=$1 saved=$2 token=$3 now stat cmd
  now=$(birth "$pid")
  if [ -z "$now" ]; then
    if kill -0 "$pid" 2>/dev/null; then return 2; else return 1; fi
  fi
  stat=$(ps -p "$pid" -o stat= 2>/dev/null) || return 2
  case "$stat" in *Z*) return 1;; esac
  [ "$saved" != - ] || return 2
  [ "$now" = "$saved" ] || return 1
  cmd=$(ps -ww -p "$pid" -o command= 2>/dev/null) || return 2
  [ -n "$cmd" ] || return 2
  case "$cmd" in *"$token"*) return 0;; *) return 1;; esac
}
owner() {
  [ -f "$SF.owner" ] || return 1
  { IFS= read -r PID; IFS= read -r BIRTH; IFS= read -r TOKEN
    IFS= read -r REPO; IFS= read -r BRANCH; IFS= read -r PGID; } < "$SF.owner"
  case "$PID" in ''|*[!0-9]*) return 1;; esac
  [ "$PID" -gt 1 ] && [ -n "$BIRTH" ] && [ -n "$TOKEN" ] && [ -n "$REPO" ] && [ -n "$BRANCH" ]
}
# No owner means checkout was never released: inspect state, never clean a tree.
ownerless_dead() {
  local pid b token stamp
  if [ -f "$SF.started" ]; then
    { IFS= read -r pid; IFS= read -r b; IFS= read -r token; } < "$SF.started"
    case "$pid" in ''|*[!0-9]*) return 1;; esac
    [ "$pid" -gt 1 ] && [ -n "$b" ] && [ -n "$token" ] || return 1
    identity "$pid" "$b" "$token"; [ "$?" -eq 1 ]
  else
    # GNU first: on Linux `stat -f` is filesystem status and prints `?` with rc 0.
    stamp=$(stat -c %Y "$SF" 2>/dev/null) || stamp=$(stat -f %m "$SF" 2>/dev/null) || return 1
    case "$stamp" in ''|*[!0-9]*) return 1;; esac
    [ "$(( $(date +%s) - stamp ))" -ge "${MISSION_WORKTREE_OWNER_GRACE:-60}" ]
  fi
}
# Checkout has a separate, recorded group, so hooks cannot recreate a removed tree.
quiesce() {
  [ -f "$SF.child" ] || return 0
  local cp cb ct cg rc own n=0 members
  { IFS= read -r cp; IFS= read -r cb; IFS= read -r ct; IFS= read -r cg; } < "$SF.child"
  case "$cp" in ''|*[!0-9]*) return 1;; esac
  [ "$cp" -gt 1 ] && [ "$cg" = "$cp" ] || return 1
  own=$(pgid $$); [ -n "$own" ] && [ "$cg" != "$own" ] || return 1
  identity "$cp" "$cb" "$ct"; rc=$?
  if [ "$rc" -eq 0 ]; then
    [ "$(pgid "$cp")" = "$cg" ] || return 1
    kill -KILL -- "-$cg" 2>/dev/null || return 1
  elif [ "$rc" -eq 2 ]; then
    return 1
  elif kill -0 "$cp" 2>/dev/null; then
    # A zombie leader is dead; a replacement process is never ours to signal.
    [ "$(birth "$cp")" = "$cb" ] || return 1
    case "$(ps -p "$cp" -o stat= 2>/dev/null)" in *Z*) :;; *) return 1;; esac
  fi
  while [ "$n" -lt 50 ]; do
    members=$(ps -axo pgid=,stat= 2>/dev/null) || return 1
    if ! printf '%s\n' "$members" | awk -v g="$cg" '$1==g && $2 !~ /Z/ {found=1} END {exit !found}'; then return 0; fi
    sleep .1; n=$((n+1))
  done
  return 1
}
# Child publishes its own final bash identity, then waits for a bounded release.
register() {
  local file=$1 b g
  b=$(birth $$); g=$(pgid $$)
  printf '%s\n%s\n%s\n%s\n' "$$" "${b:--}" "$TOKEN" "${g:--}" > "$file.tmp.$$"
  mv "$file.tmp.$$" "$file"
}
# Bounds are wall-clock (SECONDS): a loaded rig makes each `sleep .1` cost ~0.2 s.
rendezvous() {
  local end=$((SECONDS + 10))
  while [ ! -f "$1" ] && [ "$SECONDS" -lt "$end" ]; do sleep .1; done
  [ -f "$1" ]
}
case "${1:-}" in
  _checkout)
    REPO=$2; BRANCH=$3; SHA=$4; WT=$5; SF=$6; TOKEN=$7
    register "$SF.child"
    rendezvous "$SF.go" || exit 1
    git -C "$REPO" worktree add -q -b "$BRANCH" "$WT" "$SHA" >/dev/null 2>&1
    exit $?
    ;;
  _writer)
    REPO=$2; BRANCH=$3; SHA=$4; WT=$5; SF=$6; TOKEN=$7; LIMIT=$8
    [ "$(cat "$SF" 2>/dev/null)" = pending ] || exit 1
    register "$SF.started" && rendezvous "$SF.owner" && owner && [ "$PID" = "$$" ] || {
      fail_pending owner_never_published; exit 1;
    }
    [ "$(cat "$SF" 2>/dev/null)" = pending ] || exit 1
    set -m
    /bin/bash "$0" _checkout "$REPO" "$BRANCH" "$SHA" "$WT" "$SF" "$TOKEN" & p=$!
    set +m
    rendezvous "$SF.child" || { fail_pending writer_died; exit 1; }
    : > "$SF.go"
    s=0; reason=""
    while kill -0 "$p" 2>/dev/null; do
      if [ "$s" -ge "$LIMIT" ]; then reason="timeout_${LIMIT}s"; break; fi
      sleep 1; s=$((s+1))
    done
    if [ -n "$reason" ]; then
      if ! quiesce; then echo 'mission-worktree: cannot verify checkout group for timeout cleanup' >&2; exit 1; fi
      wait "$p" 2>/dev/null || :
    else
      wait "$p"; rc=$?
      [ "$rc" -eq 0 ] || reason="worktree_add_rc_$rc"
    fi
    if [ -z "$reason" ]; then
      n=$(git -C "$WT" status --porcelain 2>/dev/null | wc -l | tr -d ' ')
      [ "$n" = 0 ] || reason="not_clean_after_add ($n entries)"
    fi
    if [ -n "$reason" ]; then cleanup; echo "failed:$reason" > "$SF"; else echo ready > "$SF"; fi
    ;;
  add)
    BRANCH="${2:-}"; SHA="${3:-}"; WT="${4:-}"
    [ -n "$BRANCH" ] && [ -n "$SHA" ] && [ -n "$WT" ] || { echo 'usage: mission-worktree.sh add BRANCH SHA PATH' >&2; exit 2; }
    case "$WT" in /tmp/*|/private/tmp/*) echo "failed:path_under_tmp ($WT) — use a sibling of the repo" >&2; exit 1;; esac
    [ ! -e "$WT" ] || { echo "failed:path_exists ($WT)" >&2; exit 1; }
    REPO=$(git rev-parse --show-toplevel 2>/dev/null) || { echo failed:not_in_a_repo >&2; exit 1; }
    SF=$(status_file "$WT"); echo pending > "$SF"
    TOKEN=$(mktemp "$STATE/launch.XXXXXX") || exit 1
    rm -f "$TOKEN"
    LIMIT="${MISSION_WORKTREE_TIMEOUT:-900}"
    # Probing the actual POSIX call also catches missing/broken Perl installations.
    set +m
    if perl -MPOSIX -e 'POSIX::setsid() >= 0 or die "setsid: $!"' </dev/null >/dev/null 2>&1; then
      nohup perl -MPOSIX -e 'POSIX::setsid() >= 0 or die "setsid: $!"; exec @ARGV; die "exec: $!"' \
        /bin/bash "$0" _writer "$REPO" "$BRANCH" "$SHA" "$WT" "$SF" "$TOKEN" "$LIMIT" </dev/null >/dev/null 2>&1 & PID=$!
    else
      echo 'mission-worktree: Perl POSIX::setsid unavailable; using bash set -m' >&2
      set -m
      nohup /bin/bash "$0" _writer "$REPO" "$BRANCH" "$SHA" "$WT" "$SF" "$TOKEN" "$LIMIT" </dev/null >/dev/null 2>&1 & PID=$!
      set +m
    fi
    # A failed launcher still gets an owner record; wait can prove its PID absent.
    s=$((SECONDS + 10))
    while [ ! -f "$SF.started" ] && kill -0 "$PID" 2>/dev/null && [ "$SECONDS" -lt "$s" ]; do sleep .1; done
    BIRTH=-; PGID=-
    if [ -f "$SF.started" ]; then
      { IFS= read -r started_pid; IFS= read -r BIRTH; IFS= read -r started_token; IFS= read -r PGID; } < "$SF.started"
      [ "$started_pid/$started_token" = "$PID/$TOKEN" ] || { BIRTH=-; PGID=-; }
    fi
    caller=$(pgid $$)
    if [ "$PGID" = - ] || [ -z "$caller" ] || [ "$PGID" = "$caller" ]; then
      echo 'mission-worktree: detachment unverified; retaining nohup with writer identity monitoring' >&2
    fi
    printf '%s\n%s\n%s\n%s\n%s\n%s\n' "$PID" "$BIRTH" "$TOKEN" "$REPO" "$BRANCH" "$PGID" > "$SF.owner.tmp.$$"
    mv "$SF.owner.tmp.$$" "$SF.owner" || exit 1
    echo "$WT"
    ;;
  wait)
    WT="${2:-}"; MAX="${3:-45}"
    [ -n "$WT" ] || { echo 'usage: mission-worktree.sh wait PATH [MAX_S]' >&2; exit 2; }
    SF=$(status_file "$WT"); [ -f "$SF" ] || { echo failed:unknown_worktree; exit 1; }
    s=0
    while :; do
      v=$(cat "$SF" 2>/dev/null); terminal "$v"
      if [ "$v" = pending ] && owner; then
        saved=$(cat "$SF.owner")
        identity "$PID" "$BIRTH" "$TOKEN"; rc=$?
        if [ "$rc" -eq 1 ]; then
          v=$(cat "$SF" 2>/dev/null); terminal "$v"
          if [ "$v" = pending ] && [ "$(cat "$SF.owner" 2>/dev/null)" = "$saved" ] && quiesce; then
            cleanup; echo failed:writer_died; consume; exit 1
          fi
        fi
      elif [ "$v" = pending ] && [ ! -f "$SF.owner" ]; then
        saved=$(cat "$SF.started" 2>/dev/null)
        if ownerless_dead; then
          v=$(cat "$SF" 2>/dev/null); terminal "$v"
          if [ "$v" = pending ] && [ ! -f "$SF.owner" ] && [ "$(cat "$SF.started" 2>/dev/null)" = "$saved" ]; then
            if [ -n "$saved" ]; then echo failed:writer_died; else echo failed:owner_never_published; fi
            consume; exit 1
          fi
        fi
      fi
      [ "$s" -ge "$MAX" ] && { echo pending; exit 75; }
      sleep 1; s=$((s+1))
    done
    ;;
  *) sed -n '2,8p' "$0" >&2; exit 2;;
esac
