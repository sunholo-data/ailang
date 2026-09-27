#!/usr/bin/env bash
# test_rig_watchdog_kill_tree.sh — kill_tree must reap a descendant that left its
# parent's process group (the opencode/pi shape that leaked the GPU on 2026-09-27).
#
# Builds: parent (this test's child group) → worker that calls setpgrp → sleeper.
# Killing only the parent's pgroup leaves worker+sleeper alive; kill_tree must not.
set -u
here=$(cd "$(dirname "$0")" && pwd)
fns=$(mktemp); trap 'rm -f "$fns"' EXIT
# Pull just the helper functions out of the watchdog (it acts on source otherwise).
sed -n '/^descendants() {/,/^}/p; /^kill_tree() {/,/^}/p' "$here/rig-watchdog.sh" > "$fns"
# shellcheck disable=SC1090
. "$fns"
command -v kill_tree >/dev/null || { echo "FAIL: kill_tree not found in rig-watchdog.sh"; exit 1; }

fail=0
# parent: a bash that spawns a worker in its OWN process group, then waits.
bash -c 'perl -e "setpgrp(0,0); system(q{sleep 300}); sleep 300" & wait' &
parent=$!
sleep 1
worker=$(pgrep -P "$parent" | head -1)
sleeper=$(pgrep -P "$worker" | head -1)
pg_parent=$(ps -o pgid= -p "$parent" | tr -d ' ')
pg_worker=$(ps -o pgid= -p "$worker" | tr -d ' ')
if [ -z "$worker" ] || [ -z "$sleeper" ] || [ "$pg_parent" = "$pg_worker" ]; then
    echo "FAIL: fixture did not build a split-group tree (worker=$worker sleeper=$sleeper pg=$pg_parent/$pg_worker)"
    kill -9 "$parent" "$worker" "$sleeper" 2>/dev/null; exit 1
fi

kill_tree "$parent" >/dev/null
sleep 1
for p in "$parent" "$worker" "$sleeper"; do
    if ps -p "$p" >/dev/null 2>&1 && [ "$(ps -o stat= -p "$p" | cut -c1)" != "Z" ]; then
        echo "FAIL: pid $p survived kill_tree"; fail=1; kill -9 "$p" 2>/dev/null
    fi
done
# the test's own shell must survive (kill_tree never signals its caller's group)
[ "$fail" = 0 ] && echo "PASS: kill_tree reaped a split-group tree (parent $parent, worker $worker, sleeper $sleeper)"
exit "$fail"
