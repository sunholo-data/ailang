#!/bin/bash
# --tool-stall-seconds: an OPEN tool call gets its own no-progress bound; model silence keeps
# --stall-seconds. Ticket tool-hang:pi-evaluator:stall-bound-below-single-test-run (fleet i105).
#
# Every arm stubs `pi` on PATH, so no model is called. Timing arms use stall 4 / tool 6-20 s,
# so the suite stays short; arm 6 is the only timing assertion on the poll clamp.
set -u

SUT="$(cd "$(dirname "$0")/../.." && pwd)/scripts/mission_pi_run.sh"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/node_modules/@anthropic-ai/sandbox-runtime"
echo '{}' > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules"
MISSION_PI_SANDBOX_STAGE_PARENT="$TMP"
MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude"
export MISSION_PI_SANDBOX_NODE_MODULES MISSION_PI_SANDBOX_STAGE_PARENT MISSION_PI_CLAUDE_TMP_DIR
mkdir -p "$TMP/bin"
PATH="$TMP/bin:$PATH"; export PATH
unset MISSION_PI_STALL_SECONDS MISSION_PI_TOOL_STALL_SECONDS MISSION_PI_MAX_SECONDS
MISSION_PI_POLL_SECONDS=1; export MISSION_PI_POLL_SECONDS

passed=0; failed=0
field() { sed -n "s/.*\"$2\": \"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p" "$1" 2>/dev/null; }
ok()   { echo "PASS: $1"; passed=$((passed+1)); }
bad()  { echo "FAIL: $1"; failed=$((failed+1)); }

stub() { # stub <shell body run as pi>
  { echo '#!/bin/bash'; echo 'cat >/dev/null'; echo ': > "$PI_SANDBOX_READY_FILE"'; echo "$1"; } > "$TMP/bin/pi"
  chmod +x "$TMP/bin/pi"
}
repo() { # fresh git worktree with one base commit
  rm -rf "$1"; mkdir -p "$1"
  git -C "$1" init -q
  echo base > "$1/f.txt"
  git -C "$1" add f.txt
  git -C "$1" -c user.email=t@t -c user.name=t commit -qm base
}
# arm NAME EXPECT_RC EXPECT_VERDICT EXPECT_BOUND|- [runner args...]   (stub already set)
arm() {
  local name=$1 erc=$2 ever=$3 ebound=$4; shift 4
  repo "$TMP/$name.wt"
  echo directive > "$TMP/$name.directive"
  "$SUT" --model m --directive "$TMP/$name.directive" --workdir "$TMP/$name.wt" \
    --out "$TMP/$name.ndjson" --max-seconds 30 "$@" >"$TMP/$name.log" 2>&1
  local rc=$?
  ARM_VF="$TMP/$name.ndjson.verdict.json"
  local v; v=$(field "$ARM_VF" verdict 2>/dev/null)
  if [ "$rc" = "$erc" ] && [ "$v" = "$ever" ] && \
     { [ "$ebound" = - ] || [ "$(field "$ARM_VF" stall_bound_seconds)" = "$ebound" ]; }; then
    ok "$name (rc=$rc verdict=$v bound=${ebound})"
  else
    bad "$name (rc=$rc verdict=${v:-<missing>} bound=$(field "$ARM_VF" stall_bound_seconds 2>/dev/null); expected rc=$erc verdict=$ever bound=$ebound)"
  fi
}
TOOL_OPEN='printf "{\"type\":\"tool_execution_start\",\"toolName\":\"bash\",\"args\":{\"command\":\"make slow-test\"}}\n"'
TOOL_DONE='echo changed > f.txt; printf "{\"type\":\"tool_execution_end\"}\n"; printf "{\"type\":\"agent_end\"}\n"'

# Arm 1 — default: no tool flag, so the tool bound IS the stall bound (4). The open tool
# is silent for 8 s and must be killed at 4 s as tool_hang.
stub "$TOOL_OPEN; sleep 8; $TOOL_DONE"
arm 1_default 18 tool_hang 4 --stall-seconds 4
if [ "$(field "$ARM_VF" tool_stall_seconds)" = 4 ] && [ "$(field "$ARM_VF" stall_seconds)" = 4 ] && \
   grep -q 'make slow-test' "$ARM_VF"; then
  ok "1_default verdict keys (stall_seconds=4 tool_stall_seconds=4, hung_tool named)"
else
  bad "1_default verdict keys (stall_seconds/tool_stall_seconds/hung_tool)"
fi

# Arm 2 — raised tool bound (20): the same 8 s silent open tool must survive and finish ok.
stub "$TOOL_OPEN; sleep 8; $TOOL_DONE"
arm 2_raised 0 ok - --stall-seconds 4 --tool-stall-seconds 20
if [ "$(field "$ARM_VF" tool_stall_seconds)" = 20 ]; then ok "2_raised verdict records tool_stall_seconds=20"
else bad "2_raised verdict records tool_stall_seconds=20"; fi

# Arm 3 — model silence with no open tool: the stall bound (4) applies even though the tool
# bound is 20. Must be stream_dead at 4 s, not survive to 12 s.
stub 'printf "{\"type\":\"turn_start\"}\n"; sleep 12'
arm 3_model_silence 12 stream_dead 4 --stall-seconds 4 --tool-stall-seconds 20

# Arm 3b — a COMPLETED tool pair then silence: the tool is closed, so the stall bound applies.
stub 'printf "{\"type\":\"tool_execution_start\",\"toolName\":\"bash\",\"args\":{\"command\":\"ls\"}}\n"; printf "{\"type\":\"tool_execution_end\"}\n"; sleep 12'
arm 3b_closed_tool 12 stream_dead 4 --stall-seconds 4 --tool-stall-seconds 20

# Arm 4 — open tool past its raised-but-finite bound (6): killed at 6 s as tool_hang, naming
# the command. A tool that sleeps 15 s must not be exempt from every bound.
stub "$TOOL_OPEN; sleep 15; $TOOL_DONE"
arm 4_hang_past_bound 18 tool_hang 6 --stall-seconds 4 --tool-stall-seconds 6
if grep -q 'make slow-test' "$ARM_VF"; then ok "4_hang_past_bound names hung_tool"
else bad "4_hang_past_bound names hung_tool"; fi

# Arm 5 — bad values are refused rc 14 with a verdict file, by flag and by env.
stub ':'
for v in abc 0 -5 08; do
  repo "$TMP/5_$v.wt"; echo directive > "$TMP/5_$v.directive"
  "$SUT" --model m --directive "$TMP/5_$v.directive" --workdir "$TMP/5_$v.wt" \
    --out "$TMP/5_$v.ndjson" --tool-stall-seconds "$v" >/dev/null 2>&1
  rc=$?
  if [ "$rc" = 14 ] && [ "$(field "$TMP/5_$v.ndjson.verdict.json" verdict)" = launch_failed ]; then
    ok "5 bad value '$v' refused (rc 14, verdict file written)"
  else bad "5 bad value '$v' refused (rc=$rc)"; fi
done
repo "$TMP/5_env.wt"; echo directive > "$TMP/5_env.directive"
MISSION_PI_TOOL_STALL_SECONDS=x "$SUT" --model m --directive "$TMP/5_env.directive" \
  --workdir "$TMP/5_env.wt" --out "$TMP/5_env.ndjson" >/dev/null 2>&1
rc=$?
if [ "$rc" = 14 ] && [ -f "$TMP/5_env.ndjson.verdict.json" ]; then ok "5 bad env MISSION_PI_TOOL_STALL_SECONDS=x refused"
else bad "5 bad env MISSION_PI_TOOL_STALL_SECONDS=x refused (rc=$rc)"; fi

# Arm 6 — poll clamp follows the SMALLER bound. Unset POLL (10 s default): with stall 60 and
# tool 3 the clamp must be a third of 3 (poll 1 s), so the 3 s open-tool bound fires by ~4 s.
# A clamp on stall/3 would first poll at 10 s and report ~12 s elapsed.
unset MISSION_PI_POLL_SECONDS
stub "$TOOL_OPEN; sleep 30; $TOOL_DONE"
arm 6_clamp 18 tool_hang 3 --stall-seconds 60 --tool-stall-seconds 3 --max-seconds 60
el=$(field "$ARM_VF" elapsed_seconds)
if [ -n "$el" ] && [ "$el" -le 9 ]; then ok "6_clamp elapsed ${el}s <= 9"
else bad "6_clamp elapsed ${el:-<missing>}s (expected <= 9)"; fi
MISSION_PI_POLL_SECONDS=1; export MISSION_PI_POLL_SECONDS

echo "passed=$passed failed=$failed"
[ "$failed" -eq 0 ]
