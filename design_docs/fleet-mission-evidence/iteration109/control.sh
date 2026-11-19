#!/bin/bash
# i109 repro: a pi child that setpgrp()s into its own group survives the runner (ok AND wall_timeout).
# WARNING (planner, iteration 109): the cleanup lines kill EVERY /bin/sleep 300 on the host. Do not run on a shared rig;
# the landed suite tools/launchd/test_mission_pi_run_orphan_reap.sh identifies its fixtures by pid file + birth instead.
set -u
SUT="${1:?runner path}"
TMP=$(mktemp -d)
mkdir -p "$TMP/node_modules/@anthropic-ai/sandbox-runtime" "$TMP/bin"
echo '{}' > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
export MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules" MISSION_PI_SANDBOX_STAGE_PARENT="$TMP" MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude" MISSION_PI_POLL_SECONDS=1
PATH="$TMP/bin:$PATH"; export PATH
TAG="i109orphan$$"
run() { # name tailbody max
  cat > "$TMP/bin/pi" <<STUB
#!/bin/bash
cat >/dev/null; : > "\$PI_SANDBOX_READY_FILE"
perl -e 'exec "/bin/sleep", "300"' $TAG >/dev/null 2>&1 </dev/null &
sleep 2
$2
STUB
  chmod +x "$TMP/bin/pi"
  rm -rf "$TMP/$1.wt"; mkdir -p "$TMP/$1.wt"; git -C "$TMP/$1.wt" init -q; echo b > "$TMP/$1.wt/f"; git -C "$TMP/$1.wt" add f; git -C "$TMP/$1.wt" -c user.email=t@t -c user.name=t commit -qm b
  echo d > "$TMP/$1.d"
  "$SUT" --model m --directive "$TMP/$1.d" --workdir "$TMP/$1.wt" --out "$TMP/$1.ndjson" --max-seconds "$3" >"$TMP/$1.log" 2>&1; rc=$?
  sleep 1
  # perl exec keeps argv0 'sleep 300' — match the sleep whose parent chain we created: grep by pgid != runner
  n=$(ps -A -o pid=,ppid=,pgid=,command= | awk '$4=="/bin/sleep" && $5=="300"' | wc -l | tr -d ' ')
  echo "$1: runner rc=$rc verdict=$(sed -n 's/.*"verdict": "\([^"]*\)".*/\1/p' "$TMP/$1.ndjson.verdict.json") surviving_sleep300=$n"
  ps -A -o pid=,ppid=,pgid=,command= | awk '$4=="/bin/sleep" && $5=="300" {print "   survivor pid="$1" ppid="$2" pgid="$3}'
  ps -A -o pid=,command= | awk '$2=="/bin/sleep" && $3=="300" {print $1}' | xargs -n1 kill 2>/dev/null
}
# control: no orphan should exist before
echo "pre-existing sleep300: $(ps -A -o command= | awk '$1=="/bin/sleep" && $2=="300"' | wc -l | tr -d ' ')"

run wall 'sleep 60' 6
rm -rf "$TMP"
