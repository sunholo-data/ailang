#!/bin/bash
SUT="/Users/voightkampff/.ailang-driver-pin/fleet/scripts/mission_pi_run.sh"; TMP=/tmp/fleet_i105/repro
mkdir -p "$TMP/node_modules/@anthropic-ai/sandbox-runtime"; echo "{}" > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
export MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules" MISSION_PI_SANDBOX_STAGE_PARENT="$TMP" MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude" MISSION_PI_POLL_SECONDS=1
mkdir -p "$TMP/bin"; export PATH="$TMP/bin:$PATH"
for case in long short; do
 if [ $case = long ]; then S=8; else S=1; fi
 cat > "$TMP/bin/pi" <<EOF
#!/bin/bash
cat >/dev/null; : > "\$PI_SANDBOX_READY_FILE"
echo '{"type":"tool_execution_start","toolName":"bash","args":{"command":"make slow-test"}}'
sleep $S; echo changed > f.txt
echo '{"type":"tool_execution_end","toolName":"bash"}'
echo '{"type":"agent_end"}'
EOF
 chmod +x "$TMP/bin/pi"; W="$TMP/w_$case"; mkdir -p $W; git -C $W init -q; echo b > $W/f.txt; git -C $W add f.txt; git -C $W -c user.email=t@t -c user.name=t commit -qm b
 echo d > $TMP/d
 "$SUT" --model m --directive $TMP/d --workdir $W --out $TMP/$case.ndjson --max-seconds 30 --stall-seconds 4 >/dev/null 2>&1; echo "$case (tool runs ${S}s, stall 4s): rc=$? $(jq -c '{verdict,hung_tool,elapsed_seconds}' $TMP/$case.ndjson.verdict.json)"
done
