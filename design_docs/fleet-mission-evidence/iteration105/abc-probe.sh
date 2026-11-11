#!/bin/bash
TMP=/tmp/fleet_i105/abcprobe; mkdir -p "$TMP/node_modules/@anthropic-ai/sandbox-runtime" "$TMP/bin"; echo "{}" > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
export MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules" MISSION_PI_SANDBOX_STAGE_PARENT="$TMP" MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude" MISSION_PI_POLL_SECONDS=1 PATH="$TMP/bin:$PATH"
printf '#!/bin/bash\ncat >/dev/null; : > "$PI_SANDBOX_READY_FILE"; echo changed > f.txt; echo "{\\"type\\":\\"agent_end\\"}"\n' > "$TMP/bin/pi"; chmod +x "$TMP/bin/pi"
for side in base head; do
  if [ $side = base ]; then SUT=/Users/voightkampff/.ailang-driver-pin/fleet/scripts/mission_pi_run.sh; else SUT=/Users/voightkampff/.ailang-driver-pin/.fleet-i105-tool-stall/scripts/mission_pi_run.sh; fi
  for v in abc 600; do
    W="$TMP/w_${side}_$v"; mkdir -p $W; git -C $W init -q 2>/dev/null; echo b > $W/f.txt; git -C $W add f.txt; git -C $W -c user.email=t@t -c user.name=t commit -qm b 2>/dev/null
    echo d > $TMP/d; OUT="$TMP/${side}_${v}_$(date +%s%N).ndjson"
    "$SUT" --model m --directive $TMP/d --workdir $W --out $OUT --max-seconds 20 --stall-seconds $v > $TMP/${side}_$v.stdout 2> $TMP/${side}_$v.stderr; rc=$?
    if [ -f "$OUT.verdict.json" ]; then j=$(jq -e -c '{verdict,rc}' "$OUT.verdict.json" 2>/dev/null || echo INVALID_JSON); else j=NO_VERDICT_FILE; fi
    echo "$side --stall-seconds $v: rc=$rc verdict=$j stderr_tail=$(tail -1 $TMP/${side}_$v.stderr | cut -c1-120)"
  done
done
