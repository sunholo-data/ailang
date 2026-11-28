# mission-slot.sh — reusable mission worktree slots (M-MISSION-WORKTREE-SLOTS). bash 3.2.
#
# Sourced by tools/launchd/mission-worktree.sh; defines functions only. Design:
# design_docs/planned/v0_64_2/m-mission-worktree-slots.md
#
# A slot is a worktree at a FIXED path, $MISSION_SLOT_ROOT/<repo>/<mission>/<role>-<n>, reused by
# every iteration so checkouts are incremental and Go build-cache keys (which include the package
# dir without -trimpath) stay stable.
#
# OWNERSHIP. A slot is owned by the FIRE, i.e. the mission driver process whose PID ends
# $MISSION_FIRE_ID (<mission>-<epoch>-<pid>, mission-control.sh L2431). Neither the acquire call
# nor the Bash tool call that ran it nor the sub-agent working in the tree lives as long as the
# iteration; the driver does (it `wait`s on the controller). The lock records the driver's PID and
# birth time; it is HELD while that process is alive with the same birth, and an unreadable birth
# counts as held. Only the driver releases (release-fire, from its own EXIT trap).
#
# LOCKING. Taking a slot lock, or stealing one whose driver is dead, happens inside a short
# per-mission mutex (mkdir). The mutex holder is the acquire process itself, alive for the few
# milliseconds it holds it; a mutex whose holder is dead is taken over with rename(2), which only
# one mover can win.

SLOT_BOUND_S=30

slot_root() { printf '%s\n' "${MISSION_SLOT_ROOT:-$HOME/.ailang-slots}"; }

# Physical path of an existing or not-yet-existing directory (resolves the deepest existing parent).
slot_phys() {
  local p=$1 tail=""
  while [ ! -d "$p" ]; do tail="/$(basename "$p")$tail"; p=$(dirname "$p"); done
  printf '%s%s\n' "$(cd "$p" && pwd -P)" "$tail"
}

# Prefixes a slot root must never sit under. The sweep's PROTECT list (scripts/worktree_sweep.sh)
# is mirrored here and pinned by test_mission_slots.sh (a drift there fails the suite); the
# checkout-audit scan root (~/dev/sunholo-data, weekly --delete of idle trees) is refused whole.
slot_forbidden_prefixes() {
  local h; h=$(slot_phys "$HOME")
  printf '%s\n' /tmp /private/tmp "$h/.ailang-driver-pin" "$h/.ailang-ab" "$h/.ailang-nightly" \
    "$h/.ailang/state/worktrees" "$h/dev/sunholo-data"
}

slot_root_ok() {
  local root p; root=$(slot_phys "$1")
  while IFS= read -r p; do
    case "$root/" in "$p/"*) return 1 ;; esac
  done <<EOF
$(slot_forbidden_prefixes)
EOF
  return 0
}

slot_count() { case "$1" in exec|judge) echo 2 ;; *) echo 1 ;; esac; }

slot_bounded() { local s=$1; shift; perl -e 'alarm(shift @ARGV); exec @ARGV' "$s" "$@"; }

# 0 = alive with the recorded birth (or inconclusive), 1 = provably dead.
slot_driver_alive() {
  local pid=$1 saved=$2 now st
  case "$pid" in ''|*[!0-9]*) return 1 ;; esac
  now=$(birth "$pid")
  if [ -z "$now" ]; then
    if kill -0 "$pid" 2>/dev/null; then return 0; else return 1; fi
  fi
  st=$(ps -p "$pid" -o stat= 2>/dev/null) || return 0
  case "$st" in *Z*) return 1 ;; esac
  [ "$now" = "$saved" ]
}

# 0 = some process has its cwd at or below $1 (or lsof could not answer), 1 = none.
slot_cwd_busy() {
  local p out; p=$(slot_phys "$1")
  out=$(slot_bounded "$SLOT_BOUND_S" lsof -n -d cwd -Fn 2>/dev/null) || return 0
  [ -n "$out" ] || return 0
  printf '%s\n' "$out" | sed -n 's/^n//p' | awk -v p="$p" '$0==p || index($0, p"/")==1 {f=1} END {exit !f}'
}

# Owner file: fire_id, driver pid, driver birth, role, epoch taken.
slot_write_owner() {
  local lock=$1 role=$2
  printf '%s\n%s\n%s\n%s\n%s\n' "$MISSION_FIRE_ID" "$SLOT_DRIVER_PID" "$SLOT_DRIVER_BIRTH" "$role" "$(date +%s)" \
    > "$lock/owner.tmp.$$" && mv "$lock/owner.tmp.$$" "$lock/owner"
}

slot_lock_held() { # LOCK -> 0 held, 1 free-or-dead
  local fid pid b m
  if [ ! -f "$1/owner" ]; then
    # Only an acquire killed between mkdir and the owner write leaves this (both happen inside
    # the mutex). Held for a grace period, then reclaimable — never held forever.
    m=$(perl -e '@s = stat shift or exit; print $s[9]' "$1" 2>/dev/null)
    case "$m" in ''|*[!0-9]*) return 0 ;; esac
    [ "$(( $(date +%s) - m ))" -lt "${MISSION_SLOT_OWNERLESS_GRACE:-60}" ]; return $?
  fi
  { IFS= read -r fid; IFS= read -r pid; IFS= read -r b; } < "$1/owner"
  slot_driver_alive "$pid" "$b"
}

slot_mutex_take() { # BASE -> 0 taken (sets SLOT_MUTEX), 1 gave up after ~10 s
  local m="$1/.mutex" end=$((SECONDS + 10)) hp hb
  while [ "$SECONDS" -lt "$end" ]; do
    if mkdir "$m" 2>/dev/null; then
      printf '%s\n%s\n' "$$" "$(birth $$)" > "$m/holder"; SLOT_MUTEX=$m; return 0
    fi
    hp=""; hb=""
    [ -f "$m/holder" ] && { IFS= read -r hp; IFS= read -r hb; } < "$m/holder"
    if [ -n "$hp" ] && ! slot_driver_alive "$hp" "$hb"; then
      # Dead holder: exactly one rename of the same source succeeds.
      mv "$m" "$m.dead.$$" 2>/dev/null && rm -rf "${m:?}.dead.$$"
      continue
    fi
    sleep .1
  done
  return 1
}
slot_mutex_drop() { [ -n "${SLOT_MUTEX:-}" ] && rm -rf "${SLOT_MUTEX:?}"; SLOT_MUTEX=""; }

# Take the first free <role>-<n> under BASE; prints its path. Caller holds no mutex.
slot_take() {
  local base=$1 role=$2 n=1 max slot lock
  max=$(slot_count "$role")
  slot_mutex_take "$base" || return 1
  while [ "$n" -le "$max" ]; do
    slot="$base/$role-$n"; lock="$slot.lock"; n=$((n+1))
    if [ -d "$lock" ]; then
      slot_lock_held "$lock" && continue
      rm -rf "${lock:?}"   # dead driver: reclaim (the reset salvages any dirt)
    fi
    mkdir "$lock" 2>/dev/null || continue
    slot_write_owner "$lock" "$role"
    if [ -e "$slot" ] && slot_cwd_busy "$slot"; then rm -rf "${lock:?}"; continue; fi
    slot_mutex_drop; printf '%s\n' "$slot"; return 0
  done
  slot_mutex_drop; return 1
}

# Every lock under the root owned by FIRE_ID [and ROLE]; prints "<lock-dir>".
slot_locks_of() {
  local fid=$1 role=${2:-} root f lf lr
  root=$(slot_root); [ -d "$root" ] || return 0
  find "$root" -mindepth 4 -maxdepth 4 -path '*.lock/owner' -type f 2>/dev/null | while IFS= read -r f; do
    { IFS= read -r lf; IFS= read -r _; IFS= read -r _; IFS= read -r lr; } < "$f"
    [ "$lf" = "$fid" ] || continue
    [ -z "$role" ] || [ "$lr" = "$role" ] || continue
    dirname "$f"
  done
}

slot_fire_owner() { # sets SLOT_DRIVER_PID / SLOT_DRIVER_BIRTH from MISSION_FIRE_ID
  local fid=${MISSION_FIRE_ID:-}
  [ -n "$fid" ] || { echo failed:no_fire_owner; return 1; }
  SLOT_DRIVER_PID=${fid##*-}
  case "$SLOT_DRIVER_PID" in ''|*[!0-9]*) echo "failed:no_fire_owner (MISSION_FIRE_ID=$fid has no pid)"; return 1 ;; esac
  SLOT_DRIVER_BIRTH=$(birth "$SLOT_DRIVER_PID")
  [ -n "$SLOT_DRIVER_BIRTH" ] || { echo "failed:fire_owner_not_running (pid $SLOT_DRIVER_PID)"; return 1; }
}

# _checkout body for a slot: create detached at SHA, or (M1) reuse only an identical clean tree.
slot_checkout() {
  local repo=$1 sha=$2 wt=$3 br=$4
  if [ -e "$wt/.git" ]; then
    [ "$(git -C "$wt" rev-parse HEAD 2>/dev/null)" = "$sha" ] || return 3
    [ -z "$(git -C "$wt" status --porcelain 2>/dev/null)" ] || return 3
  else
    mkdir -p "$(dirname "$wt")" || return 4
    git -C "$repo" worktree add -q --detach "$wt" "$sha" >/dev/null 2>&1 || return $?
  fi
  [ -z "$br" ] || git -C "$wt" switch -q -c "$br" >/dev/null 2>&1 || return 5
}

# Placeholder handed to the shared writer's failure cleanup (`git branch -D "$BRANCH"`): not a
# valid ref name, so cleanup can never delete a real branch.
SLOT_NO_BRANCH="..slot-no-branch"

slot_acquire() { # MISSION ROLE SHA [BRANCH]
  local mission=${1:-} role=${2:-} sha=${3:-} br=${4:-} root base cur
  [ -n "$mission" ] && [ -n "$role" ] && [ -n "$sha" ] || { echo 'usage: mission-worktree.sh acquire MISSION ROLE SHA [BRANCH]' >&2; return 2; }
  case "$mission" in *[!A-Za-z0-9_-]*) echo "failed:bad_mission ($mission)"; return 1 ;; esac
  case "$role" in *[!a-z]*) echo "failed:bad_role ($role)"; return 1 ;; esac
  slot_fire_owner || return 1
  local common; common=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || { echo failed:not_in_a_repo; return 1; }
  REPO=$(dirname "$common")
  sha=$(git -C "$REPO" rev-parse --verify -q "$sha^{commit}") || { echo "failed:unknown_sha ($3)"; return 1; }
  if [ -n "$br" ] && git -C "$REPO" rev-parse --verify -q "refs/heads/$br" >/dev/null; then
    echo "failed:branch_exists ($br)"; return 1
  fi
  root=$(slot_root)
  slot_root_ok "$root" || { echo "failed:slot_root_protected ($root)"; return 1; }
  base="$root/$(basename "$REPO")/$mission"
  mkdir -p "$base" || { echo "failed:cannot_create ($base)"; return 1; }
  base=$(cd "$base" && pwd -P)
  WT=$(slot_take "$base" "$role") || { echo busy; return 1; }
  if [ -e "$WT" ] && [ ! -e "$WT/.git" ]; then
    rm -rf "${WT:?}.lock"; echo "failed:slot_occupied_not_worktree ($WT)"; return 1
  fi
  if [ -e "$WT/.git" ]; then
    cur=$(git -C "$WT" rev-parse HEAD 2>/dev/null)
    if [ "$cur" != "$sha" ] || [ -n "$(git -C "$WT" status --porcelain 2>/dev/null)" ]; then
      rm -rf "${WT:?}.lock"; echo "failed:reset_not_implemented ($WT)"; return 1
    fi
  fi
  # Both are read by launch_writer in mission-worktree.sh.
  # shellcheck disable=SC2034
  BRANCH=$SLOT_NO_BRANCH
  # shellcheck disable=SC2034
  SHA=$sha
  launch_writer slot "$br" || return 1
  echo "pending $WT"
}

slot_held() { # FIRE_ID ROLE
  [ -n "${1:-}" ] && [ -n "${2:-}" ] || { echo 'usage: mission-worktree.sh held FIRE_ID ROLE' >&2; return 2; }
  local l out; out=$(slot_locks_of "$1" "$2" | sort)
  [ -n "$out" ] || return 1
  printf '%s\n' "$out" | while IFS= read -r l; do printf '%s\n' "${l%.lock}"; done
}

slot_release_fire() { # FIRE_ID
  [ -n "${1:-}" ] || { echo 'usage: mission-worktree.sh release-fire FIRE_ID' >&2; return 2; }
  local n
  n=$(slot_locks_of "$1" | while IFS= read -r l; do rm -rf "${l:?}" && echo x; done | wc -l | tr -d ' ')
  echo "released $n"
}
