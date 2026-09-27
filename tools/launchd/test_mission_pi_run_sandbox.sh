#!/bin/bash
# Runner sandbox preflight and handshake integration, with no real pi invocation.
set -u
ROOT=$(cd "$(dirname "$0")/../.." && pwd -P)
SUT="${MISSION_PI_TEST_SUT:-$ROOT/scripts/mission_pi_run.sh}"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/node_modules/@anthropic-ai/sandbox-runtime"
echo '{}' > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
export MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules"
export MISSION_PI_SANDBOX_STAGE_PARENT="$TMP"
export MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude"
export MISSION_PI_POLL_SECONDS=1
PATH="$TMP/bin:$PATH"; export PATH
printf 'directive\n' > "$TMP/directive"

cat > "$TMP/bin/pi" <<'EOF'
#!/bin/bash
printf '%s\n' "$@" > "$PI_TEST_RECORD.args"
printf '%s\n' "$PWD" "$PI_FENCE_ROOT" "$PI_SANDBOX_POLICY_FILE" "$PI_SANDBOX_READY_FILE" > "$PI_TEST_RECORD.env"
cp "$PI_SANDBOX_POLICY_FILE" "$PI_TEST_RECORD.policy"
cat >/dev/null
case "$PI_TEST_MODE" in
  no_ready) echo changed > f.txt ;;
  nonzero) : > "$PI_SANDBOX_READY_FILE"; exit 23 ;;
  commit) : > "$PI_SANDBOX_READY_FILE"; echo changed > f.txt; git add f.txt; git -c user.email=t@t -c user.name=t commit -qm changed ;;
  clean) : > "$PI_SANDBOX_READY_FILE" ;;
  *) : > "$PI_SANDBOX_READY_FILE"; echo changed > f.txt ;;
esac
printf '%s\n' '{"type":"agent_end"}'
EOF
chmod +x "$TMP/bin/pi"

git -C "$TMP" init -q main
echo base > "$TMP/main/f.txt"
git -C "$TMP/main" add f.txt
git -C "$TMP/main" -c user.email=t@t -c user.name=t commit -qm base

passed=0; failed=0
check() { # label condition
  if [ "$2" = true ]; then echo "PASS: $1"; passed=$((passed+1));
  else echo "FAIL: $1"; failed=$((failed+1)); fi
}
field() { jq -r "$2" "$1" 2>/dev/null; }
run() { # id mode [extra env supplied by caller]
  id="$1"; PI_TEST_MODE="$2"; PI_TEST_RECORD="$TMP/$id"
  export PI_TEST_MODE PI_TEST_RECORD
  git -C "$TMP/main" worktree add -q --detach "$TMP/wt-$id" HEAD
  "$SUT" --model m --directive "$TMP/directive" --workdir "$TMP/wt-$id" \
    --out "$TMP/$id.ndjson" --max-seconds 30 --stall-seconds 10 >/dev/null 2>&1
  actual_rc=$?
  verdict="$TMP/$id.ndjson.verdict.json"
}
is_verdict() { [ "$actual_rc" -eq "$1" ] && [ "$(field "$verdict" .verdict)" = "$2" ] && [ "$(field "$verdict" .fenced)" = "$3" ]; }
not_called() { [ ! -e "$PI_TEST_RECORD.args" ]; }

run happy changed
check 'happy verdict and readiness' "$(is_verdict 0 ok true && echo true || echo false)"
args="$TMP/happy.args"
stage=$(sed -n '5p' "$args" | sed 's@/sandbox/index.ts$@@')
check 'two staged absolute extensions' "$(
  [ "$(sed -n '4p' "$args")" = -e ] && [ "$(sed -n '5p' "$args")" = "$stage/sandbox/index.ts" ] &&
  [ "$(sed -n '6p' "$args")" = -e ] && [ "$(sed -n '7p' "$args")" = "$stage/worktree-fence.ts" ] && echo true || echo false)"
check 'worktree cwd and fence root' "$(
  [ "$(sed -n '1p' "$TMP/happy.env")" = "$(cd "$TMP/wt-happy" && pwd -P)" ] &&
  [ "$(sed -n '2p' "$TMP/happy.env")" = "$(cd "$TMP/wt-happy" && pwd -P)" ] && echo true || echo false)"
gitdir=$(git -C "$TMP/wt-happy" rev-parse --path-format=absolute --git-dir)
common=$(git -C "$TMP/wt-happy" rev-parse --path-format=absolute --git-common-dir)
check 'explicit policy has caches and narrow git metadata' "$(
  jq -e --arg gitdir "$gitdir" --arg objects "$common/objects" --arg heads "$common/refs/heads" --arg logs "$common/logs" \
    '.filesystem.allowWrite | index("~/Library/Caches/go-build") and index($gitdir) and index($objects) and (index($heads) | not) and (index($logs) | not)' "$TMP/happy.policy" >/dev/null &&
  [ "$(sed -n '3p' "$TMP/happy.env")" = "$stage/policy.json" ] &&
  [ "$(sed -n '4p' "$TMP/happy.env")" = "$stage/ready" ] && [ ! -e "$stage" ] && echo true || echo false)"

MISSION_PI_SANDBOX_NODE_MODULES="$TMP/absent"; export MISSION_PI_SANDBOX_NODE_MODULES
run missing_dep changed
check 'missing dependency refuses launch' "$(is_verdict 15 sandbox_unavailable false && not_called && [ -n "$(field "$verdict" .error)" ] && echo true || echo false)"
MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules"; export MISSION_PI_SANDBOX_NODE_MODULES

MISSION_PI_SANDBOX_POLICY_SOURCE="$TMP/missing.json"; export MISSION_PI_SANDBOX_POLICY_SOURCE
run missing_policy changed
check 'missing policy refuses launch' "$(is_verdict 16 sandbox_policy_invalid false && not_called && echo true || echo false)"
echo '{' > "$TMP/malformed.json"
MISSION_PI_SANDBOX_POLICY_SOURCE="$TMP/malformed.json"; export MISSION_PI_SANDBOX_POLICY_SOURCE
run malformed changed
check 'malformed policy refuses launch' "$(is_verdict 16 sandbox_policy_invalid false && not_called && echo true || echo false)"
jq '.enabled=false' "$ROOT/tools/pi-extensions/sandbox/sandbox.mission.json" > "$TMP/disabled.json"
MISSION_PI_SANDBOX_POLICY_SOURCE="$TMP/disabled.json"; export MISSION_PI_SANDBOX_POLICY_SOURCE
run disabled changed
check 'disabled policy refuses launch' "$(is_verdict 16 sandbox_policy_invalid false && not_called && echo true || echo false)"
unset MISSION_PI_SANDBOX_POLICY_SOURCE

run no_ready no_ready
check 'missing readiness defeats dirty tree' "$(is_verdict 17 sandbox_not_ready false && echo true || echo false)"
run nonzero nonzero
check 'nonzero pi status survives filter' "$(is_verdict 14 launch_failed true && [ "$(field "$verdict" .pi_rc)" = 23 ] && echo true || echo false)"
run linked_commit commit
check 'linked worktree commit counts as work' "$(is_verdict 0 ok true && [ "$(field "$verdict" .commits_since_start)" = 1 ] && [ "$(git -C "$TMP/wt-linked_commit" status --porcelain | wc -l | tr -d ' ')" = 0 ] && echo true || echo false)"
MISSION_PI_CLAUDE_TMP_DIR="$TMP/fresh-claude"; export MISSION_PI_CLAUDE_TMP_DIR
run temp_bootstrap clean
check 'temp bootstrap uses test root' "$(is_verdict 10 empty_worktree true && [ -d "$TMP/fresh-claude" ] && echo true || echo false)"
echo blocker > "$TMP/blocked-parent"
MISSION_PI_CLAUDE_TMP_DIR="$TMP/blocked-parent/claude"; export MISSION_PI_CLAUDE_TMP_DIR
run temp_failure changed
check 'temp bootstrap failure refuses launch' "$(is_verdict 15 sandbox_unavailable false && not_called && echo true || echo false)"

echo "passed=$passed failed=$failed"
[ "$failed" -eq 0 ]
