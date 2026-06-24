#!/usr/bin/env bash
# docx best-of-N: run docx up to N times, verify each vs golden, STOP at the first pass.
# Demonstrates the best-of-N lever capturing qwen3.6's high-variance syntax adherence
# (base ~1/3 -> best-of-6 ~91%). Blackout-aware (waits out GPU blackout 04:00-07:00).
set -u
N=${1:-6}
A=/Users/voightkampff/dev/sunholo-data/ailang
SUM=/tmp/docx_bestof_summary.txt; : > "$SUM"
echo "[docx best-of-$N start] $(date '+%F %H:%M')" >> "$SUM"
while h=$((10#$(date +%H))); [ "$h" -ge 4 ] && [ "$h" -lt 7 ]; do echo "  blackout, waiting ($(date '+%H:%M'))" >> "$SUM"; sleep 300; done
PASS=0
for k in $(seq 1 "$N"); do
  L=/tmp/docx_bo_$k.log; : > "$L"
  ( cd "$A" && MOTOKO_REPO=/Users/voightkampff/dev/mk-integration bash eval_projects/docx_reimplement/run.sh motoko >> "$L" 2>&1 ) &
  RUN=$!
  ( sleep 2700; kill -0 "$RUN" 2>/dev/null && pkill -9 -f 'bun.*src/tui'; ) &
  WD=$!
  wait "$RUN"; kill "$WD" 2>/dev/null
  WS=$(grep -oE 'ws=/tmp/docx-motoko-[0-9-]+' "$L"|head -1|cut -d= -f2)
  vr=$("$A/eval_projects/docx_reimplement/verify.sh" "$WS" 2>&1 | tail -1)
  echo "run$k: $vr @$(date '+%H:%M')" >> "$SUM"
  if echo "$vr" | grep -q "17 passed"; then PASS=1; echo ">>> best-of-N PASS captured at run $k" >> "$SUM"; break; fi
done
echo "[docx best-of-$N DONE] best_of_n_pass=$PASS @$(date '+%F %H:%M')" >> "$SUM"
