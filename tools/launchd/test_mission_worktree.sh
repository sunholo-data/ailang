#!/bin/bash
# test_mission_worktree.sh — mission-worktree.sh never hands back a half-built or dirty worktree.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
MW="${MISSION_WORKTREE_UNDER_TEST:-$HERE/mission-worktree.sh}"
PASS=0; FAIL=0; UNINFORMATIVE=0
PS=$(command -v ps)
RUNNER_PGID=$("$PS" -p $$ -o pgid= 2>/dev/null | tr -d ' ')
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1 — got: $2"; }

T=$(cd "$(mktemp -d)" && pwd -P); trap 'rm -rf "$T"' EXIT
case "$T" in /tmp/*|/private/tmp/*) echo "SKIP: TMPDIR is under /tmp; this suite needs a non-/tmp scratch dir"; exit 0 ;; esac
R="$T/repo"; mkdir -p "$R" "$T/home"; export HOME="$T/home" MISSION_WORKTREE_STATE="$T/state"
git -C "$R" init -q && echo a > "$R/a" && git -C "$R" add a && git -C "$R" -c user.name=t -c user.email=t@t commit -qm i
SHA=$(git -C "$R" rev-parse HEAD)
waitfor() { local o; for _ in 1 2 3 4 5 6 7 8 9 10; do o=$(cd "$R" && bash "$MW" wait "$1" 5); [ "$o" != pending ] && break; done; echo "$o"; }

if [ -z "${MISSION_WORKTREE_TEST_ARM:-}" ]; then
echo "== happy path: add returns at once, wait reports ready, tree is clean"
(cd "$R" && bash "$MW" add br1 "$SHA" "$T/wt1") >/dev/null
o=$(waitfor "$T/wt1"); [ "$o" = ready ] && ok "ready" || bad "ready" "$o"
[ -f "$T/wt1/a" ] && [ -z "$(git -C "$T/wt1" status --porcelain)" ] && ok "checked out and clean" || bad "tree" "$(ls "$T/wt1")"

echo "== /tmp is refused"
o=$(cd "$R" && bash "$MW" add br2 "$SHA" /tmp/mw-test-refuse 2>&1); rc=$?
[ "$rc" = 1 ] && case "$o" in *path_under_tmp*) true;; *) false;; esac && ok "/tmp refused" || bad "/tmp" "$o rc=$rc"

echo "== timeout: a checkout that outlives the bound is removed, branch included"
if [ -n "$RUNNER_PGID" ]; then
mkdir -p "$T/slowbin"
printf '#!/bin/bash\nif [ "$3" = "worktree" ] && [ "$4" = "add" ]; then sleep 30; fi\nexec %s "$@"\n' "$(command -v git)" > "$T/slowbin/git"
chmod +x "$T/slowbin/git"
(cd "$R" && PATH="$T/slowbin:$PATH" MISSION_WORKTREE_TIMEOUT=3 bash "$MW" add br3 "$SHA" "$T/wt3") >/dev/null
o=$(waitfor "$T/wt3"); case "$o" in failed:timeout_3s) ok "timeout reported";; *) bad "timeout" "$o";; esac
[ ! -e "$T/wt3" ] && ok "partial worktree removed" || bad "partial worktree left" "$(ls "$T/wt3")"
git -C "$R" rev-parse --verify -q br3 >/dev/null && bad "branch left behind" "br3" || ok "branch removed"

else
  echo "  UNINFORMATIVE UNDER SANDBOX: timeout group cleanup (ps denied)"
  UNINFORMATIVE=$((UNINFORMATIVE+1))
fi

echo "== a tree that is not clean after checkout is rejected"
mkdir -p "$T/hooks"; printf '#!/bin/bash\necho junk > stray.txt\n' > "$T/hooks/post-checkout"; chmod +x "$T/hooks/post-checkout"
git -C "$R" config core.hooksPath "$T/hooks"
(cd "$R" && bash "$MW" add br4 "$SHA" "$T/wt4") >/dev/null
o=$(waitfor "$T/wt4"); case "$o" in failed:not_clean_after_add*) ok "dirty tree rejected";; *) bad "dirty" "$o";; esac
[ ! -e "$T/wt4" ] && ok "dirty worktree removed" || bad "dirty worktree left" ""
git -C "$R" config --unset core.hooksPath

echo "== wait on an unknown path fails rather than hanging"
o=$(cd "$R" && bash "$MW" wait "$T/nope" 1); [ "$o" = failed:unknown_worktree ] && ok "unknown worktree" || bad "unknown" "$o"


fi

# All new fixtures, shims and process records live below this same scratch root.
MV=$(command -v mv)
CHILDREN=""
finish() {
  local p record rp rb rt rg current
  # Recover fixture-owned detached jobs even after an assertion fails.
  for record in "$T"/arm-*/state/*.status.owner "$T"/arm-*/state/*.status.child; do
    [ -f "$record" ] || continue
    { IFS= read -r rp; IFS= read -r rb; IFS= read -r rt; } < "$record"
    case "$rp" in ''|*[!0-9]*) continue;; esac
    current=$(birth "$rp")
    [ -n "$current" ] && [ "$current" = "$rb" ] || continue
    "$PS" -ww -p "$rp" -o command= 2>/dev/null | grep -F -- "$rt" >/dev/null || continue
    rg=$("$PS" -p "$rp" -o pgid= 2>/dev/null | tr -d ' ')
    if [ "$rg" = "$rp" ] && [ -n "$RUNNER_PGID" ] && [ "$rg" != "$RUNNER_PGID" ]; then
      kill -KILL -- "-$rg" 2>/dev/null || :
    fi
  done
  for p in $CHILDREN; do
    if jobs -pr | grep -x "$p" >/dev/null; then
      rg=$("$PS" -p "$p" -o pgid= 2>/dev/null | tr -d ' ')
      if [ "$rg" = "$p" ] && [ -n "$RUNNER_PGID" ] && [ "$rg" != "$RUNNER_PGID" ]; then
        kill -KILL -- "-$rg" 2>/dev/null || :
      fi
      kill -KILL "$p" 2>/dev/null || :
    fi
    wait "$p" 2>/dev/null || :
  done
  rm -rf "$T"
}
trap finish EXIT
trap 'exit 130' INT TERM
# Assertions take commands, never metadata or shell text.
check() { local label=$1; shift; if "$@" >/dev/null; then ok "$label"; else bad "$label" "stdout=${OUT:-} rc=${RC:-}"; fi; }
await_file() { local n=0; while [ ! -f "$1" ] && [ "$n" -lt 100 ]; do sleep .1; n=$((n+1)); done; [ -f "$1" ]; }
result() { OUT=$(/bin/bash "$MW" wait "$WT" "${1:-15}"); RC=$?; }
verdict() { check "$1 stdout/rc" test "$OUT/$RC" = "$2/$3"; }
fixture() {
  F="$T/arm-$1"; R="$F/repo"; WT="$F/tree"; BRANCH="arm-$1"
  mkdir -p "$R" "$F/bin" "$F/hooks"
  export MISSION_WORKTREE_STATE="$F/state" FIXTURE="$F" REAL_PS="$PS" REAL_MV="$MV"
  git -C "$R" init -q
  echo a > "$R/a"; git -C "$R" add a
  git -C "$R" -c user.name=t -c user.email=t@t commit -qm i
  SHA=$(git -C "$R" rev-parse HEAD)
  SF="$MISSION_WORKTREE_STATE/$(printf '%s' "$WT" | tr / _).status"
}
hold_hook() {
  cat > "$F/hooks/post-checkout" <<'HOOK'
#!/bin/bash
: > "$FIXTURE/entered"
n=0
while [ ! -f "$FIXTURE/release" ] && [ "$n" -lt 200 ]; do sleep .1; n=$((n+1)); done
# A surviving hook would recreate the tree after the waiter's cleanup.
if [ -f "$FIXTURE/release" ]; then mkdir -p "$FIXTURE/tree"; : > "$FIXTURE/late"; fi
HOOK
  chmod +x "$F/hooks/post-checkout"; git -C "$R" config core.hooksPath "$F/hooks"
}
add() { (cd "$R" && /bin/bash "$MW" add "$BRANCH" "$SHA" "$WT") > "$F/add.out" 2> "$F/add.err"; }
owner() {
  [ -f "$SF.owner" ] || return 1
  { IFS= read -r PID; IFS= read -r BIRTH; IFS= read -r TOKEN; IFS= read -r REPO; IFS= read -r BR; IFS= read -r PGID; } < "$SF.owner"
}
birth() { LC_ALL=C "$PS" -p "$1" -o lstart= 2>/dev/null | awk '{$1=$1;print}'; }
identity() {
  [ "$(birth "$PID")" = "$BIRTH" ] && [ -n "$BIRTH" ] &&
    "$PS" -ww -p "$PID" -o command= 2>/dev/null | grep -F -- "$TOKEN" >/dev/null
}
consumed() { check 'terminal status consumed' test ! -e "$SF"; check 'owner consumed' test ! -e "$SF.owner"; }
removed() {
  check 'partial path removed' test ! -e "$WT"
  if git -C "$R" rev-parse --verify -q "$BRANCH" >/dev/null; then bad 'branch removed' "$BRANCH exists"; else ok 'branch removed'; fi
  if git -C "$R" worktree list --porcelain | grep -F "worktree $WT" >/dev/null; then bad 'registration pruned' "$WT"; else ok 'registration pruned'; fi
}
process_arm() {
  if [ -z "$RUNNER_PGID" ]; then
    echo "  UNINFORMATIVE UNDER SANDBOX: $1 (ps denied)"; UNINFORMATIVE=$((UNINFORMATIVE+1)); return 1
  fi
}
run_arm() {
  [ -z "${MISSION_WORKTREE_TEST_ARM:-}" ] || [ "$MISSION_WORKTREE_TEST_ARM" = "$1" ] || return 0
  echo "== $1"; "$2"
}
dead_writer_mode() {
  fixture "dead-$1"; hold_hook; add
  if ! { await_file "$F/entered" && owner && identity; }; then bad 'writer rendezvous/identity' ''; return; fi
  check 'partial checkout exists' test -f "$WT/a"
  check 'branch exists during checkout' git -C "$R" rev-parse --verify -q "$BRANCH"
  # The checkout leader must be in its own verified group before any signal.
  local cp cb ct cg
  { IFS= read -r cp; IFS= read -r cb; IFS= read -r ct; IFS= read -r cg; } < "$SF.child"
  [ "$cg" = "$cp" ] && [ "$cg" != "$RUNNER_PGID" ] && [ "$(birth "$cp")" = "$cb" ] && [ "$ct" = "$TOKEN" ] && [ "$("$PS" -p "$cp" -o pgid= | tr -d ' ')" = "$cg" ] || { bad 'safe checkout group' "$cg"; return; }
  kill -KILL "$PID" || { echo '  UNINFORMATIVE UNDER SANDBOX: writer signal denied'; UNINFORMATIVE=$((UNINFORMATIVE+1)); return; }
  if [ "$1" = group-gone ]; then
    # This group was created by this fixture and its leader identity checked above.
    kill -KILL -- "-$cg" || { echo '  UNINFORMATIVE UNDER SANDBOX: checkout group signal denied'; UNINFORMATIVE=$((UNINFORMATIVE+1)); return; }
  fi
  result 3; verdict 'writer death' failed:writer_died 1
  removed; consumed
  local st
  st=$("$PS" -p "$cp" -o stat= 2>/dev/null)
  case "$st" in ''|*Z*) ok 'owned checkout quiesced';; *) bad 'owned checkout quiesced' "$st";; esac
  : > "$F/release"
  local n=0
  while [ "$n" -lt 100 ]; do
    st=$("$PS" -p "$cp" -o stat= 2>/dev/null)
    case "$st" in ''|*Z*) break;; esac
    [ ! -f "$F/late" ] || break
    sleep .1; n=$((n+1))
  done
  check 'no late hook recreation' test ! -e "$F/late"; check 'path stays removed' test ! -e "$WT"
  if git -C "$R" rev-parse --verify -q "$BRANCH" >/dev/null; then bad 'branch stays removed' "$BRANCH exists"; else ok 'branch stays removed'; fi
}
dead_writer() {
  process_arm 'writer killed mid-checkout' || return 0
  dead_writer_mode writer-only
  dead_writer_mode group-gone
}
caller_term() {
  local fallback=${1:-no} harness hg wg hp
  process_arm 'caller group TERM' || return 0
  fixture "caller-$fallback"; hold_hook
  if [ "$fallback" = yes ]; then
    printf '#!/bin/bash\nexit 127\n' > "$F/bin/perl"; chmod +x "$F/bin/perl"
  fi
  # shellcheck disable=SC2016 # Expanded by the isolated harness.
  harness='cd "$1" || exit; /bin/bash "$2" add "$3" "$4" "$5" > "$6/add.out" 2> "$6/add.err"; echo $? > "$6/add.rc"; : > "$6/returned"; n=0; while [ ! -f "$6/caller-release" ] && [ "$n" -lt 200 ]; do sleep .1; n=$((n+1)); done'
  # Monitor mode isolates this fixture independently of the production launcher.
  set -m
  PATH="$F/bin:$PATH" /bin/bash -c "$harness" _ "$R" "$MW" "$BRANCH" "$SHA" "$WT" "$F" & hp=$!
  set +m
  CHILDREN="$CHILDREN $hp"
  if ! { await_file "$F/returned" && await_file "$F/entered" && owner; }; then bad 'caller rendezvous' ''; return; fi
  check 'add PATH/rc before caller teardown' test "$(cat "$F/add.out")/$(cat "$F/add.rc")" = "$WT/0"
  hg=$("$PS" -p "$hp" -o pgid= | tr -d ' '); wg=$("$PS" -p "$PID" -o pgid= | tr -d ' ')
  [ "$hg" = "$hp" ] && [ "$hg" != "$RUNNER_PGID" ] || { bad 'private caller group' "$hg"; return; }
  check 'writer detached from caller PGID' test "$wg" != "$hg"
  check 'recorded writer PGID matches' test "$wg" = "$PGID"
  if grep -q 'using bash set -m' "$F/add.err"; then echo '  launcher: bash set -m'; else echo '  launcher: Perl POSIX::setsid'; fi
  if ! kill -TERM -- "-$hg"; then echo '  UNINFORMATIVE UNDER SANDBOX: caller group signal denied'; UNINFORMATIVE=$((UNINFORMATIVE+1)); return; fi
  wait "$hp" 2>/dev/null || :
  : > "$F/release"; result; verdict 'caller teardown' ready 0
  check 'clean complete tree' test -z "$(git -C "$WT" status --porcelain 2>/dev/null)"
  check 'tree preserved' test -f "$WT/a"; check 'branch preserved' git -C "$R" rev-parse --verify -q "$BRANCH"; consumed
}
fallback() {
  if [ -n "$RUNNER_PGID" ]; then caller_term yes; return; fi
  process_arm 'caller group TERM' || :
  # The no-signal fallback smoke remains useful when group inspection is denied.
  fixture fallback-smoke
  printf '#!/bin/bash\nexit 127\n' > "$F/bin/perl"; chmod +x "$F/bin/perl"
  PATH="$F/bin:$PATH" add
  echo '  launcher selected (forced unavailable Perl): bash set -m; detachment UNINFORMATIVE UNDER SANDBOX'
  result; verdict 'bash fallback smoke' ready 0; consumed
}
live_writer() {
  process_arm 'live slow writer' || return 0
  fixture live; hold_hook; add
  if ! { await_file "$F/entered" && owner && identity; }; then bad 'live writer identity' ''; return; fi
  result 1; verdict 'live pending' pending 75
  check 'pending state retained' test "$(cat "$SF")" = pending
  check 'live path retained' test -f "$WT/a"; check 'owner still alive' identity
  check 'live branch retained' git -C "$R" rev-parse --verify -q "$BRANCH"
  : > "$F/release"; result; verdict 'released writer' ready 0; consumed
}
race() {
  local terminal
  for terminal in ready failed:worktree_add_rc_1; do
    fixture "race-${terminal//:/-}"; mkdir -p "$MISSION_WORKTREE_STATE"
    git -C "$R" worktree add -qb "$BRANCH" "$WT" "$SHA"
    printf '99999999\nold birth\nrace-token\n%s\n%s\n99999999\n' "$R" "$BRANCH" > "$SF.owner"
    echo pending > "$SF"; export RACE_SF="$SF" RACE_TERMINAL="$terminal"
    cat > "$F/bin/ps" <<'SHIM'
#!/bin/bash
if [ "$2" = 99999999 ]; then printf '%s\n' "$RACE_TERMINAL" > "$RACE_SF.tmp"; "$REAL_MV" "$RACE_SF.tmp" "$RACE_SF"; exit 1; fi
exec "$REAL_PS" "$@"
SHIM
    chmod +x "$F/bin/ps"
    PATH="$F/bin:$PATH" result 1
    if [ "$terminal" = ready ]; then verdict 'completion re-read' ready 0; else verdict 'failure re-read' "$terminal" 1; fi
    check 'race tree preserved' test -f "$WT/a"; check 'race branch preserved' git -C "$R" rev-parse --verify -q "$BRANCH"; consumed
  done
}
sentinel_case() {
  process_arm 'PID identity sentinel' || return 0
  fixture "$1"; mkdir -p "$MISSION_WORKTREE_STATE"
  /bin/bash -c 'n=0; while [ "$n" -lt 200 ]; do sleep .1; n=$((n+1)); done' sentinel-token & local sentinel=$!
  CHILDREN="$CHILDREN $sentinel"
  local saved token=sentinel-token
  saved=$(birth "$sentinel"); [ "$1" != birth ] || saved='different birth'
  [ "$1" != token ] || token=wrong-launch-token
  git -C "$R" worktree add -qb "$BRANCH" "$WT" "$SHA"
  printf '%s\n%s\n%s\n%s\n%s\n0\n' "$sentinel" "$saved" "$token" "$R" "$BRANCH" > "$SF.owner"
  echo pending > "$SF"; result 1; verdict 'identity mismatch' failed:writer_died 1; removed; consumed
  check 'replacement sentinel survives' kill -0 "$sentinel"
  kill "$sentinel"; wait "$sentinel" 2>/dev/null || :
}
birth_case() { sentinel_case birth; }
token_case() { sentinel_case token; }
publication() {
  fixture publication; hold_hook
  cat > "$F/bin/mv" <<'SHIM'
#!/bin/bash
case "$2" in *.status.owner)
  : > "$FIXTURE/publishing"; n=0
  while [ ! -f "$FIXTURE/publish-release" ] && [ "$n" -lt 100 ]; do sleep .1; n=$((n+1)); done ;;
esac
exec "$REAL_MV" "$@"
SHIM
  chmod +x "$F/bin/mv"
  PATH="$F/bin:$PATH" add & local ap=$!; CHILDREN="$CHILDREN $ap"
  if await_file "$F/publishing"; then
    if [ ! -s "$F/add.out" ]; then ok 'no PATH before owner publication'; else bad 'no PATH before owner publication' "$(cat "$F/add.out") (owner absent)"; fi
  else bad 'owner publication rendezvous' 'not reached'; fi
  : > "$F/publish-release"
  await_file "$F/entered" || bad 'checkout released after owner publication' ''
  wait "$ap" || bad 'registered add rc' "$?"
  if grep -q 'using bash set -m' "$F/add.err"; then echo '  launcher selected on host: bash set -m'; else echo '  launcher selected on host: Perl POSIX::setsid'; fi
  if owner; then
    check 'metadata repo' test "$REPO" = "$R"; check 'metadata branch' test "$BR" = "$BRANCH"
    check 'metadata PID' test "$PID" -gt 1; check 'metadata token' test -n "$TOKEN"; check 'metadata birth' test -n "$BIRTH"
    if [ -n "$RUNNER_PGID" ]; then check 'published live identity' identity; else echo '  UNINFORMATIVE UNDER SANDBOX: published live identity (ps denied)'; UNINFORMATIVE=$((UNINFORMATIVE+1)); fi
  else bad 'complete owner at add return' ''; fi
  : > "$F/release"; result; verdict 'published writer completes' ready 0; consumed
  fixture startup; mkdir -p "$F/bin"
  # Probe succeeds, actual launcher fails before exec: the parent must retain an owner.
  cat > "$F/bin/perl" <<'SHIM'
#!/bin/bash
if [ -e "$FIXTURE/probed" ]; then exit 1; fi
: > "$FIXTURE/probed"; exit 0
SHIM
  chmod +x "$F/bin/perl"
  PATH="$F/bin:$PATH" add
  result 1; verdict 'startup failure terminalized' failed:writer_died 1; consumed
}
run_arm 'writer killed mid-checkout => failed:writer_died rc=1 and cleanup' dead_writer
run_arm 'caller group TERM after add => ready rc=0' caller_term
run_arm 'live slow writer => pending rc=75 without cleanup' live_writer
run_arm 'completion between pending read and dead check => ready rc=0' race
run_arm 'reused PID with different birth => writer_died without signaling replacement' birth_case
run_arm 'same-second PID identity needs launch token' token_case
run_arm 'missing Perl => detached bash fallback reaches ready' fallback
run_arm 'add publication => owner recorded before returned PATH' publication

echo "==== $PASS passed, $FAIL failed, $UNINFORMATIVE UNINFORMATIVE UNDER SANDBOX ===="
[ "$FAIL" -eq 0 ]
