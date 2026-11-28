#!/bin/bash
# test_mission_slots.sh — reusable mission worktree slots (M-MISSION-WORKTREE-SLOTS).
# Design: design_docs/planned/v0_64_2/m-mission-worktree-slots.md (TST-SLOT-* rows). bash 3.2.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
MW="${MISSION_WORKTREE_UNDER_TEST:-$HERE/mission-worktree.sh}"
SWEEP="$HERE/../../scripts/worktree_sweep.sh"
PASS=0; FAIL=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1 — got: $2"; }

T=$(cd "$(mktemp -d)" && pwd -P)
KIDS=""
TMPROOT="/tmp/ailang-slot-test.$$"   # must be under /tmp to test the refusal; removed if a bug creates it
finish() { { for k in $KIDS; do kill "$k"; done; wait; } 2>/dev/null; rm -rf "${T:?}" "${TMPROOT:?}"; }
trap finish EXIT
case "$T" in /tmp/*|/private/tmp/*) echo "SKIP: TMPDIR is under /tmp; this suite needs a non-/tmp scratch dir"; exit 0 ;; esac
R="$T/repo"; mkdir -p "$R" "$T/home"
export HOME="$T/home" MISSION_WORKTREE_STATE="$T/state"
unset MISSION_SLOT_ROOT
g() { git -C "$R" -c user.name=t -c user.email=t@t "$@"; }
g init -q && echo a > "$R/a" && g add a && g commit -qm one
SHA1=$(g rev-parse HEAD)
echo b > "$R/b" && g add b && g commit -qm two
SHA2=$(g rev-parse HEAD)
ROOT="$HOME/.ailang-slots/repo/m"

# Background sleeps drop the EXIT trap: bash 3.2, once `set -m` has been on, runs an inherited EXIT
# trap in a forked job that is killed, which would delete $T mid-suite (seen while writing this).
bg_sleep() { ( trap - EXIT; exec sleep "$1" ) & }
# A stand-in mission driver: long-lived, its PID ends the fire id (as mission-control.sh L2431).
start_driver() { bg_sleep 600; D=$!; KIDS="$KIDS $D"; FIRE="m-$(date +%s)-$D"; }
acq() { (cd "$R" && MISSION_FIRE_ID="$1" bash "$MW" acquire m "$2" "$3" ${4:+"$4"} 2>/dev/null); }
waitfor() { local o; for _ in 1 2 3 4 5 6 7 8 9 10; do o=$(cd "$R" && bash "$MW" wait "$1" 5); [ "$o" != pending ] && break; done; echo "$o"; }
acq_ready() { # FIRE ROLE SHA [BRANCH] -> prints the slot path, or the failure
  local o p; o=$(acq "$@")
  case "$o" in "pending "*) p=${o#pending }; [ "$(waitfor "$p")" = ready ] && echo "$p" || echo "notready:$p" ;; *) echo "$o" ;; esac
}
fhash() { git hash-object "$1" 2>/dev/null || echo absent; }

echo "== TST-SLOT-01: first acquire creates the slot at the exact SHA, clean"
start_driver; F1=$FIRE; D1=$D
o=$(acq_ready "$F1" exec "$SHA1")
[ "$o" = "$ROOT/exec-1" ] && ok "slot path is fixed: $o" || bad "slot path" "$o"
[ "$(git -C "$ROOT/exec-1" rev-parse HEAD 2>/dev/null)" = "$SHA1" ] && ok "HEAD = SHA" || bad "HEAD" "$(git -C "$ROOT/exec-1" rev-parse HEAD 2>&1)"
[ -z "$(git -C "$ROOT/exec-1" status --porcelain)" ] && ok "clean" || bad "clean" "$(git -C "$ROOT/exec-1" status --porcelain)"
[ "$(cd "$R" && bash "$MW" held "$F1" exec)" = "$ROOT/exec-1" ] && ok "held FIRE exec -> the slot" || bad "held" "$(cd "$R" && bash "$MW" held "$F1" exec)"

echo "== acquire with a new BRANCH switches the slot onto it"
o=$(acq_ready "$F1" record "$SHA1" rec/b1)
[ "$(git -C "$ROOT/record-1" branch --show-current 2>/dev/null)" = rec/b1 ] && ok "on rec/b1" || bad "branch" "$o"

echo "== TST-SLOT-07: an existing branch name is refused and never moved"
before=$(g rev-parse rec/b1)
o=$(acq "$F1" plan "$SHA2" rec/b1)
case "$o" in failed:branch_exists*) ok "refused: $o" ;; *) bad "branch_exists" "$o" ;; esac
[ "$(g rev-parse rec/b1)" = "$before" ] && ok "rec/b1 unchanged" || bad "rec/b1 moved" "$(g rev-parse rec/b1)"
[ ! -e "$ROOT/plan-1" ] && [ ! -e "$ROOT/plan-1.lock" ] && ok "no plan slot or lock created" || bad "plan created" "$(ls "$ROOT")"

echo "== TST-SLOT-04: acquired in shell A (exited), nothing cwd inside, live driver -> never taken"
o=$(acq_ready "$F1" judge "$SHA1")             # shell A: a subshell that has exited
[ "$o" = "$ROOT/judge-1" ] || bad "judge-1 setup" "$o"
echo marker > "$ROOT/judge-1/untracked-marker"
h_before="$(fhash "$ROOT/judge-1/a")/$(fhash "$ROOT/judge-1/untracked-marker")"
set -m
( acq "$F1" judge "$SHA1" > "$T/b.out" ) & bpid=$!
set +m
wait "$bpid"
case "$(cat "$T/b.out")" in "pending $ROOT/judge-2") ok "shell B (own process group) got judge-2" ;; *) bad "shell B" "$(cat "$T/b.out")" ;; esac
waitfor "$ROOT/judge-2" >/dev/null
o=$(acq "$F1" judge "$SHA1")
[ "$o" = busy ] && ok "all judge slots held -> busy" || bad "busy" "$o"
[ "$(fhash "$ROOT/judge-1/a")/$(fhash "$ROOT/judge-1/untracked-marker")" = "$h_before" ] && ok "judge-1 untouched (tracked + untracked)" || bad "judge-1 touched" ""

echo "== TST-SLOT-05: a dead driver's slot is reclaimed; PID reuse with a new birth is reclaimed; unreadable birth is held"
rm -f "$ROOT/judge-1/untracked-marker"
(cd "$R" && bash "$MW" release-fire "$F1") >/dev/null
o=$(acq_ready "$F1" exec "$SHA1"); [ "$o" = "$ROOT/exec-1" ] || bad "re-take exec-1" "$o"
kill "$D1"; wait "$D1" 2>/dev/null          # SIGKILL-like: no trap runs, the lock stays
[ -d "$ROOT/exec-1.lock" ] && ok "dead driver's lock still on disk" || bad "lock vanished" ""
start_driver; F2=$FIRE
o=$(acq_ready "$F2" exec "$SHA1")
[ "$o" = "$ROOT/exec-1" ] && ok "reclaimed exec-1 from the dead driver" || bad "reclaim" "$o"
[ "$(sed -n 1p "$ROOT/exec-1.lock/owner")" = "$F2" ] && ok "owner is the new fire" || bad "owner" "$(sed -n 1p "$ROOT/exec-1.lock/owner")"
# PID reuse: a live PID whose recorded birth differs.
(cd "$R" && bash "$MW" release-fire "$F2") >/dev/null
mkdir "$ROOT/exec-1.lock"; printf '%s\n%s\n%s\n%s\n%s\n' "x-1-$$" "$$" "Mon Jan  1 00:00:00 2001" exec 0 > "$ROOT/exec-1.lock/owner"
o=$(acq_ready "$F2" exec "$SHA1")
[ "$o" = "$ROOT/exec-1" ] && ok "live PID with another birth = dead owner, reclaimed" || bad "pid reuse" "$o"
# Unreadable birth for the owner's PID: ps shim answers nothing for that one PID.
(cd "$R" && bash "$MW" release-fire "$F2") >/dev/null
start_driver; F3=$FIRE; D3=$D
o=$(acq_ready "$F3" exec "$SHA1"); [ "$o" = "$ROOT/exec-1" ] || bad "F3 exec-1" "$o"
mkdir -p "$T/psbin"
printf '#!/bin/bash\nfor a in "$@"; do [ "$a" = "%s" ] && exit 1; done\nexec /bin/ps "$@"\n' "$D3" > "$T/psbin/ps"; chmod +x "$T/psbin/ps"
o=$(cd "$R" && PATH="$T/psbin:$PATH" MISSION_FIRE_ID="$F2" bash "$MW" acquire m exec "$SHA1" 2>/dev/null)
[ "$o" = "pending $ROOT/exec-2" ] && ok "unreadable birth = held (got exec-2, not exec-1)" || bad "unreadable birth" "$o"
waitfor "$ROOT/exec-2" >/dev/null
(cd "$R" && bash "$MW" release-fire "$F2") >/dev/null; (cd "$R" && bash "$MW" release-fire "$F3") >/dev/null

echo "== TST-SLOT-09: a slot root under a protected prefix is refused, nothing created"
for bad_root in "$TMPROOT" "$HOME/.ailang-driver-pin/x" "$HOME/dev/sunholo-data/slots" "$HOME/dev/sunholo-data/foo-worktrees"; do
  o=$(cd "$R" && MISSION_SLOT_ROOT="$bad_root" MISSION_FIRE_ID="$F2" bash "$MW" acquire m exec "$SHA1" 2>/dev/null)
  case "$o" in failed:slot_root_protected*) ok "refused $bad_root" ;; *) bad "root $bad_root" "$o" ;; esac
  [ ! -e "$bad_root" ] || bad "created $bad_root" "$(ls "$bad_root")"
done
# Seam: every prefix the sweep protects must be refused as a slot root (except the slot root itself).
line=$(grep '^PROTECT=' "$SWEEP")
# shellcheck disable=SC2034  # read by the eval of the PROTECT line
prot=$(AILANG_SWEEP_PROTECT=""; eval "$line"; printf '%s' "$PROTECT")
n=0; seam_bad=0
IFS=':'; for p in $prot; do
  [ -n "$p" ] || continue
  case "$p" in "$HOME/.ailang-slots"|"$HOME/.ailang-slots/") continue ;; esac
  n=$((n+1))
  o=$(cd "$R" && MISSION_SLOT_ROOT="${p%/}/slots" MISSION_FIRE_ID="$F2" bash "$MW" acquire m exec "$SHA1" 2>/dev/null)
  case "$o" in failed:slot_root_protected*) : ;; *) seam_bad=1; bad "sweep-protected prefix accepted as slot root: $p" "$o" ;; esac
done; unset IFS
[ "$n" -ge 4 ] && [ "$seam_bad" = 0 ] && ok "all $n sweep PROTECT prefixes refused" || { [ "$n" -ge 4 ] || bad "seam control: parsed too few PROTECT prefixes" "$n"; }

echo "== TST-SLOT-13: no fire owner -> refused, nothing created"
o=$(cd "$R" && env -u MISSION_FIRE_ID bash "$MW" acquire m plan "$SHA1" 2>/dev/null)
case "$o" in failed:no_fire_owner*) ok "unset MISSION_FIRE_ID refused" ;; *) bad "no owner" "$o" ;; esac
bg_sleep 30; dead=$!; kill "$dead"; wait "$dead" 2>/dev/null
o=$(acq "m-1-$dead" plan "$SHA1")
case "$o" in failed:fire_owner_not_running*) ok "dead driver PID refused" ;; *) bad "dead owner" "$o" ;; esac
[ ! -e "$ROOT/plan-1.lock" ] && [ ! -e "$ROOT/plan-1" ] && ok "no plan lock or tree" || bad "plan created" "$(ls "$ROOT")"

echo "== TST-SLOT-12: the driver's EXIT trap releases exactly its own fire's locks; trees stay"
start_driver; FO=$FIRE
o=$(acq_ready "$FO" plan "$SHA1"); [ "$o" = "$ROOT/plan-1" ] || bad "other fire plan-1" "$o"
cat > "$T/driver.sh" <<EOF
trap 'bash "$MW" release-fire "\$MISSION_FIRE_ID" > "$T/released"' EXIT
export MISSION_FIRE_ID="m-\$(date +%s)-\$\$"; echo "\$MISSION_FIRE_ID" > "$T/fire"
while [ ! -f "$T/stop" ]; do sleep .2; done
EOF
(cd "$R" && /bin/bash "$T/driver.sh") & DD=$!; KIDS="$KIDS $DD"
for _ in $(seq 1 50); do [ -s "$T/fire" ] && break; sleep .1; done
FD=$(cat "$T/fire")
a=$(acq_ready "$FD" exec "$SHA1"); r=$(acq_ready "$FD" record "$SHA1")
[ "$a" = "$ROOT/exec-1" ] && [ "$r" = "$ROOT/record-1" ] && ok "driver fire holds exec-1 + record-1" || bad "driver holds" "$a / $r"
touch "$T/stop"; wait "$DD"
[ "$(cat "$T/released" 2>/dev/null)" = "released 2" ] && ok "EXIT trap released 2" || bad "released" "$(cat "$T/released" 2>&1)"
[ ! -d "$ROOT/exec-1.lock" ] && [ ! -d "$ROOT/record-1.lock" ] && ok "its locks are gone" || bad "locks left" "$(ls -d "$ROOT"/*.lock)"
[ -e "$ROOT/exec-1/.git" ] && [ -e "$ROOT/record-1/.git" ] && ok "trees stay" || bad "trees gone" ""
[ -d "$ROOT/plan-1.lock" ] && ok "another fire's lock untouched" || bad "other fire lock removed" ""
(cd "$R" && bash "$MW" held "$FD" exec) >/dev/null 2>&1 && bad "held after release" "still held" || ok "held -> none after release"
(cd "$R" && bash "$MW" release-fire "$FO") >/dev/null

echo "== TST-SLOT-14: two acquirers race to steal the same dead lock -> exactly one gets exec-1"
rounds=3; good=0
for i in $(seq 1 $rounds); do
  start_driver; FA=$FIRE; start_driver; FB=$FIRE
  rm -rf "${ROOT:?}/exec-1.lock" "${ROOT:?}/exec-2.lock"
  mkdir "$ROOT/exec-1.lock"; printf '%s\n%s\n%s\n%s\n%s\n' "dead-1-$dead" "$dead" "x" exec 0 > "$ROOT/exec-1.lock/owner"
  rm -f "$T/go" "$T/ra" "$T/rb"
  set -m
  ( n=0; while [ ! -f "$T/go" ] && [ $n -lt 200 ]; do sleep .05; n=$((n+1)); done; acq "$FA" exec "$SHA1" > "$T/ra" ) & pa=$!
  ( n=0; while [ ! -f "$T/go" ] && [ $n -lt 200 ]; do sleep .05; n=$((n+1)); done; acq "$FB" exec "$SHA1" > "$T/rb" ) & pb=$!
  set +m
  sleep .3; touch "$T/go"; wait "$pa" "$pb"
  A=$(cat "$T/ra"); B=$(cat "$T/rb")
  e1=0; case "$A" in *"/exec-1") e1=$((e1+1)) ;; esac; case "$B" in *"/exec-1") e1=$((e1+1)) ;; esac
  own=$(sed -n 1p "$ROOT/exec-1.lock/owner" 2>/dev/null)
  if [ "$e1" = 1 ] && [ "$A" != "$B" ] && { [ "$own" = "$FA" ] || [ "$own" = "$FB" ]; }; then good=$((good+1)); else bad "round $i" "A=$A B=$B owner=$own"; fi
  for s in exec-1 exec-2; do waitfor "$ROOT/$s" >/dev/null 2>&1; done
  (cd "$R" && bash "$MW" release-fire "$FA"; bash "$MW" release-fire "$FB") >/dev/null
done
[ "$good" = "$rounds" ] && ok "$rounds/$rounds rounds: exactly one winner of exec-1, distinct slots, owner recorded"

mtime() { perl -e '@s = stat shift or exit; print $s[9]' "$1" 2>/dev/null; }
release() { (cd "$R" && bash "$MW" release-fire "$1") >/dev/null; }
SV="$HOME/.ailang/state/slots"

echo "== TST-SLOT-02: reset to a newer SHA rewrites only the changed files"
start_driver; F9=$FIRE
o=$(acq_ready "$F9" exec "$SHA1"); [ "$o" = "$ROOT/exec-1" ] || bad "exec-1 at SHA1" "$o"
release "$F9"
a_before=$(mtime "$ROOT/exec-1/a"); sleep 1.2
o=$(acq_ready "$F9" exec "$SHA2")
[ "$o" = "$ROOT/exec-1" ] && ok "same slot reused at SHA2" || bad "reuse" "$o"
[ "$(git -C "$ROOT/exec-1" rev-parse HEAD)" = "$SHA2" ] && [ -f "$ROOT/exec-1/b" ] && ok "HEAD = SHA2, new file present" || bad "HEAD" "$(git -C "$ROOT/exec-1" rev-parse HEAD)"
[ "$(mtime "$ROOT/exec-1/a")" = "$a_before" ] && ok "unchanged file a not rewritten (incremental checkout)" || bad "a rewritten" "$a_before -> $(mtime "$ROOT/exec-1/a")"
[ -z "$(git -C "$ROOT/exec-1" status --porcelain)" ] && ok "clean" || bad "clean" "$(git -C "$ROOT/exec-1" status --porcelain)"
[ ! -d "$SV/repo_m_exec-1" ] && ok "clean slot: no salvage written" || bad "salvage for a clean slot" "$(ls "$SV/repo_m_exec-1")"
release "$F9"

echo "== TST-SLOT-03: a dirty slot is salvaged (staged, unstaged, index, untracked) and then reset"
W="$ROOT/exec-1"
git -C "$W" checkout -q --detach "$SHA1"
echo b2 > "$W/c-unstaged"; git -C "$W" add c-unstaged; git -C "$W" -c user.name=t -c user.email=t@t commit -qm local-detached
echo B > "$W/a"; git -C "$W" add a; echo a > "$W/a"   # stage B, restore the file to HEAD's content
echo b3 > "$W/c-unstaged"                              # unstaged change on a tracked file
echo U > "$W/untracked.txt"
OLD=$(git -C "$W" rev-parse HEAD)
[ -n "$(git -C "$W" diff HEAD -- a)" ] && bad "fixture" "git diff HEAD sees a" || ok "fixture: the staged B is invisible to git diff HEAD"
GD=$(git -C "$W" rev-parse --path-format=absolute --git-dir); cp "$GD/index" "$T/index.before"
o=$(acq_ready "$F9" exec "$SHA2")
[ "$o" = "$W" ] && ok "reset to SHA2" || bad "reset" "$o"
[ -z "$(git -C "$W" status --porcelain)" ] && [ "$(git -C "$W" rev-parse HEAD)" = "$SHA2" ] && ok "clean at SHA2" || bad "after reset" "$(git -C "$W" status --porcelain)"
d=$(find "$SV/repo_m_exec-1" -mindepth 1 -maxdepth 1 -name 'salvage-*' -type d | head -1)
[ -n "$d" ] && ok "salvage dir written" || bad "salvage" "none"
cmp -s "$d/index" "$T/index.before" && ok "index copy is byte-identical" || bad "index copy" "differs"
ref=$(sed -n 's/^ref=//p' "$d/INFO")
[ -n "$ref" ] && [ "$(git -C "$R" rev-parse "$ref" 2>/dev/null)" = "$OLD" ] && ok "unreachable detached commit pinned at $ref" || bad "pin" "ref=$ref"
# Restore into a fresh tree at the old HEAD and compare with what was there.
git -C "$R" worktree add -q --detach "$T/restore" "$OLD"
(cd "$T/restore" && git apply --index "$d/staged.patch" && git apply "$d/unstaged.patch" && tar -xf "$d/untracked.tar")
[ "$(git -C "$T/restore" show :a)" = B ] && ok "staged B restored into the index" || bad "staged B" "$(git -C "$T/restore" show :a 2>&1)"
[ "$(cat "$T/restore/a")" = a ] && [ "$(cat "$T/restore/c-unstaged")" = b3 ] && [ "$(cat "$T/restore/untracked.txt")" = U ] \
  && ok "working file, unstaged change and untracked file restored" || bad "restore" "$(ls "$T/restore")"
git -C "$R" worktree remove --force "$T/restore"
release "$F9"

echo "== slot on a branch from an earlier fire: reset onto a new branch keeps the old one"
o=$(acq_ready "$F9" record "$SHA2" rec/b2)
[ "$(git -C "$ROOT/record-1" branch --show-current)" = rec/b2 ] && [ "$(git -C "$ROOT/record-1" rev-parse HEAD)" = "$SHA2" ] && ok "record-1 on rec/b2 at SHA2" || bad "record-1" "$o"
[ "$(g rev-parse rec/b1)" = "$SHA1" ] && ok "rec/b1 untouched" || bad "rec/b1" "$(g rev-parse rec/b1)"
release "$F9"

echo "== TST-SLOT-08: a reset that does not prove clean deletes the slot"
# Inject AFTER clean: a git shim that leaves a stray file once `clean` has run (a hook would be
# undone by the clean itself).
mkdir -p "$T/gitbin"
printf '#!/bin/bash\n%s "$@"; rc=$?\ncase " $* " in *" clean "*) echo junk > "$2/stray.txt";; esac\nexit $rc\n' "$(command -v git)" > "$T/gitbin/git"; chmod +x "$T/gitbin/git"
o=$(cd "$R" && PATH="$T/gitbin:$PATH" MISSION_FIRE_ID="$F9" bash "$MW" acquire m exec "$SHA1" 2>/dev/null); p=${o#pending }; r=$(waitfor "$p")
case "$r" in failed:not_clean_after_reset*) ok "failed:not_clean_after_reset" ;; *) bad "unclean reset" "$r" ;; esac
[ ! -e "$ROOT/exec-1" ] && ok "slot dir removed" || bad "slot left" "$(ls "$ROOT/exec-1" | head -3)"
release "$F9"   # the fire still holds the lock of its failed slot; the next fire reclaims it
o=$(acq_ready "$F9" exec "$SHA1"); [ "$o" = "$ROOT/exec-1" ] && ok "next acquire recreates it" || bad "recreate" "$o"
release "$F9"

echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" = 0 ]
