#!/bin/bash
# Real process-tree regression for the mission controller watchdogs.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
DRIVER="${MC_KILL_TREE_DRIVER:-$HERE/mission-control.sh}"
ARM="${MC_KILL_TREE_ARM:-all}"
TMP="$(mktemp -d)" || exit 1
PIDS="$TMP/pids"
: > "$PIDS"

alive() {
  local stat
  kill -0 "$1" 2>/dev/null || return 1
  stat=$(ps -p "$1" -o stat= 2>/dev/null) || return 1
  case "$stat" in *Z*) return 1 ;; esac
  return 0
}
cleanup() {
  local pid deadline file
  # Collect every PID written by fixtures, including a child born during TERM.
  for file in "$TMP"/*.pid; do
    [ -s "$file" ] && cat "$file" >> "$PIDS"
  done
  for pid in $(cat "$PIDS" 2>/dev/null); do
    pgrep -P "$pid" 2>/dev/null >> "$PIDS" || true
  done
  # KILL directly: a polite TERM here can itself create a new late child.
  for pid in $(cat "$PIDS" 2>/dev/null); do kill -KILL "$pid" 2>/dev/null || true; done
  deadline=$(( $(date +%s) + 2 ))
  while [ "$(date +%s)" -lt "$deadline" ]; do
    local any=0
    for pid in $(cat "$PIDS" 2>/dev/null); do alive "$pid" && any=1; done
    [ "$any" -eq 0 ] && break
    /bin/sleep 0.1
  done
  for file in "$TMP"/*.pid; do
    [ -s "$file" ] && kill -KILL "$(cat "$file")" 2>/dev/null || true
  done
  rm -rf "$TMP"
}
trap cleanup EXIT

awk '/^_mc_descendants\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/descendants.sh"
awk '/^_mc_kill_tree\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/kill_tree.sh"
awk '/^_mc_run_once\(\) \{/,/^\}$/' "$DRIVER" > "$TMP/run_once.sh"
[ -s "$TMP/descendants.sh" ] && [ -s "$TMP/kill_tree.sh" ] && [ -s "$TMP/run_once.sh" ] || {
  echo "FAIL extraction: missing driver function"; exit 1;
}
# shellcheck source=/dev/null
. "$TMP/descendants.sh"
# shellcheck source=/dev/null
. "$TMP/kill_tree.sh"
# shellcheck source=/dev/null
. "$TMP/run_once.sh"

pass=0; fail=0
fixture_root=0; fixture_child=0; fixture_grand=0; fixture_late=0; fixture_sibling=0
ok() { pass=$((pass+1)); echo "ok - $1"; }
bad() { fail=$((fail+1)); echo "not ok - $1"; }
selected() { [ "$ARM" = all ] || [ "$ARM" = "$1" ]; }
wait_file() {
  local path="$1" deadline=$(( $(date +%s) + 5 ))
  while [ ! -s "$path" ] && [ "$(date +%s)" -lt "$deadline" ]; do /bin/sleep 0.1; done
  [ -s "$path" ]
}
wait_dead() {
  local pid="$1" deadline=$(( $(date +%s) + 5 ))
  while alive "$pid" && [ "$(date +%s)" -lt "$deadline" ]; do /bin/sleep 0.1; done
  ! alive "$pid"
}
wait_job() {
  local pid="$1" deadline=$(( $(date +%s) + 7 ))
  while kill -0 "$pid" 2>/dev/null && [ "$(date +%s)" -lt "$deadline" ]; do
    # A completed shell job may still answer kill -0 until wait reaps it.
    [ "$(jobs -pr | grep -c "^${pid}$")" -eq 0 ] && break
    /bin/sleep 0.1
  done
  if [ "$(date +%s)" -ge "$deadline" ]; then return 1; fi
  wait "$pid" 2>/dev/null || true
  return 0
}

if selected structural || [ "$ARM" = all ]; then
  if grep -Fq '_mc_kill_tree "$CONTROLLER_PID" 60' "$TMP/run_once.sh" &&
     grep -Fq '_mc_kill_tree "$CONTROLLER_PID" 30' "$TMP/run_once.sh"; then
    ok "both watchdog sites call the helper with 60/30"
  else
    bad "both watchdog sites call the helper with 60/30"
  fi
fi
if [ "$ARM" = structural ]; then
  echo "kill tree structural: $pass passed, $fail failed"
  [ "$fail" -eq 0 ]; exit $?
fi
if ! ps -p "$$" -o stat= > "$TMP/ps.out" 2> "$TMP/ps.err" ||
   [ ! -s "$TMP/ps.out" ]; then
  echo "UNINFORMATIVE UNDER SANDBOX: ps cannot inspect fixture processes"
  exit 2
fi
pgrep -P "$$" > "$TMP/pgrep.out" 2> "$TMP/pgrep.err" || true
if [ -s "$TMP/pgrep.err" ]; then
  echo "UNINFORMATIVE UNDER SANDBOX: pgrep cannot inspect fixture processes"
  exit 2
fi

cat > "$TMP/grand.sh" <<'EOF'
#!/bin/sh
echo "$$" > "$1/$2.pid"
trap '' TERM
while :; do sleep 1; done
EOF
cat > "$TMP/child.sh" <<'EOF'
#!/bin/sh
dir="$1"; name="$2"
sh "$dir/grand.sh" "$dir" "${name}-grand" &
echo "$$" > "$dir/$name.pid"
if [ "$name" = late ]; then
  trap 'sh "$dir/grand.sh" "$dir" late-born & echo "$!" > "$dir/late-born-start.pid"' TERM
else
  trap '' TERM
fi
while :; do sleep 1; done
EOF
cat > "$TMP/root.sh" <<'EOF'
#!/bin/sh
dir="$1"
sh "$dir/child.sh" "$dir" resistant &
sh "$dir/child.sh" "$dir" late &
echo "$$" > "$dir/root.pid"
while :; do sleep 1; done
EOF
cat > "$TMP/sibling.sh" <<'EOF'
#!/bin/sh
echo "$$" > "$1/sibling.pid"
while :; do sleep 1; done
EOF

if [ "$ARM" = all ] || [ "$ARM" = root ] || [ "$ARM" = resistant ] ||
   [ "$ARM" = late ] || [ "$ARM" = sibling ]; then
  sh "$TMP/root.sh" "$TMP" & root_job=$!
  sh "$TMP/sibling.sh" "$TMP" & sibling_job=$!
  for name in root resistant resistant-grand late late-grand sibling; do
    if ! wait_file "$TMP/$name.pid"; then bad "fixture $name started"; exit 1; fi
    cat "$TMP/$name.pid" >> "$PIDS"
    case "$name" in
      root) fixture_root=$((fixture_root+1)) ;;
      resistant|late) fixture_child=$((fixture_child+1)) ;;
      resistant-grand|late-grand) fixture_grand=$((fixture_grand+1)) ;;
      sibling) fixture_sibling=$((fixture_sibling+1)) ;;
    esac
  done
  export MC_TEST_SIBLING_PID="$sibling_job"
  _mc_kill_tree "$root_job" 1 & reap_job=$!
  if ! wait_job "$reap_job"; then bad "tree helper completed by deadline"; exit 1; fi
  if selected late || [ "$ARM" = all ]; then
    if wait_file "$TMP/late-born.pid"; then
      cat "$TMP/late-born.pid" >> "$PIDS"
      fixture_late=$((fixture_late+1))
    else bad "late descendant started"; fi
  fi
  if selected root || [ "$ARM" = all ]; then
    result=0
    for name in root resistant late; do wait_dead "$(cat "$TMP/$name.pid")" || result=1; done
    [ "$result" -eq 0 ] && ok "root and children dead" || bad "root and children dead"
  fi
  if selected resistant || [ "$ARM" = all ]; then
    result=0
    for name in resistant resistant-grand late-grand; do wait_dead "$(cat "$TMP/$name.pid")" || result=1; done
    [ "$result" -eq 0 ] && ok "TERM-resistant descendants dead after KILL" || bad "TERM-resistant descendants dead after KILL"
  fi
  if selected late || [ "$ARM" = all ]; then
    if [ -s "$TMP/late-born.pid" ] && wait_dead "$(cat "$TMP/late-born.pid")"; then
      ok "grace-born descendant dead"
    else bad "grace-born descendant dead"; fi
  fi
  if selected sibling || [ "$ARM" = all ]; then
    alive "$sibling_job" && ok "unrelated sibling alive" || bad "unrelated sibling alive"
  fi
fi

if selected triggered || [ "$ARM" = all ]; then
  # The real run function is extracted. Only the helper's grace is shortened;
  # its production call sites remain 60/30 and are checked above.
  eval "$(declare -f _mc_kill_tree | sed '1s/_mc_kill_tree/_mc_real_kill_tree/')"
  _mc_kill_tree() { _mc_real_kill_tree "$1" 1; }
  # Keep the long idle stall watchdog in-shell, so cancelling it leaves no
  # orphan sleep process in the fixture.
  sleep() { case "$1" in 10) while :; do :; done ;; *) /bin/sleep "$1" ;; esac; }
  claude() {
    sh "$TMP/grand.sh" "$TMP" trigger-child &
    echo "$!" > "$TMP/trigger-child-start.pid"
    while :; do sleep 1; done
  }
  _mc_slot_state="$TMP/state"; MISSION_NAME=test; attempt=1
  CONTROLLER_PROVIDER=claude; MODEL=test; PROMPT=test
  LOG="$TMP/controller.log"; PIDFILE="$TMP/controller.pid"
  HARD_TIMEOUT=1; STALL_GRACE=10; STALL_SAMPLES=3; STALL_INTERVAL=1; STALL_CHILD_AGE=1
  mkdir -p "$_mc_slot_state"
  _mc_run_once & run_job=$!
  if ! wait_file "$TMP/trigger-child.pid"; then bad "trigger fixture started"; exit 1; fi
  cat "$TMP/trigger-child.pid" >> "$PIDS"
  if wait_job "$run_job" && wait_dead "$(cat "$TMP/trigger-child.pid")"; then
    ok "triggered watchdog finishes reap before return"
  else bad "triggered watchdog finishes reap before return"; fi
fi

if selected normal || [ "$ARM" = all ]; then
  # A finished controller must not wait for either watchdog's sleep interval.
  claude() { return 0; }
  sleep() { case "$1" in 10) while :; do :; done ;; *) /bin/sleep "$1" ;; esac; }
  _mc_slot_state="$TMP/normal-state"; MISSION_NAME=test; attempt=2
  CONTROLLER_PROVIDER=claude; MODEL=test; PROMPT=test
  LOG="$TMP/normal.log"; PIDFILE="$TMP/normal.pid"
  HARD_TIMEOUT=10; STALL_GRACE=10; STALL_SAMPLES=3; STALL_INTERVAL=1; STALL_CHILD_AGE=1
  mkdir -p "$_mc_slot_state"
  _mc_run_once & run_job=$!
  if wait_job "$run_job"; then ok "normal completion cancels watchdogs promptly"
  else bad "normal completion cancels watchdogs promptly"; fi
fi

echo "kill tree fixtures: root=$fixture_root child=$fixture_child grandchild=$fixture_grand late=$fixture_late sibling=$fixture_sibling; $pass passed, $fail failed"
[ "$fail" -eq 0 ]
