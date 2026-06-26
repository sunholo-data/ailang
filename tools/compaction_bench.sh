#!/usr/bin/env bash
# compaction_bench.sh [task] [threshold_pct] [n] — force-compaction RELIABILITY test.
#
# Compaction only fires naturally on huge-context tasks (docx), which take ~1hr each.
# This lowers the compaction_ai threshold so a SMALL task fires the AI summarizer
# repeatedly, then checks the compaction PATH directly + repeatably (minutes, not hours):
#   - did compaction FIRE                       (the trigger works)
#   - sys_kept >= 1 on EVERY compaction          (host fix: AILANG reference preserved)
#   - 0 empty summaries                          (/no_think fix: qwen returns content)
#   - the model still CONVERGED                  (working set preserved enough)
#
# On a small task (<112K ctx) the host's structural layer stays dormant, so this is a
# clean test of the AI-compaction path alone.
#
#   tools/compaction_bench.sh stats_probe 8 5
set -uo pipefail
TASK="${1:-stats_probe}"; TH="${2:-8}"; N="${3:-5}"
REPO="${MOTOKO_REPO:-/Users/voightkampff/dev/mk-ast}"
ED="$(cd "$(dirname "$0")/../eval_projects" && pwd)"
CFG="$REPO/.motoko/config/ollama/compaction_ai.json"

cp "$CFG" "$CFG.benchbak"
trap 'mv -f "$CFG.benchbak" "$CFG" 2>/dev/null' EXIT
python3 -c "import json;p='$CFG';d=json.load(open(p));d['threshold_pct']=$TH;json.dump(d,open(p,'w'),indent=2)"
echo "=== compaction reliability: $TASK x$N @ threshold_pct=$TH (repo=$REPO) ==="
printf "%-5s %-7s %-9s %-7s %-6s\n" "run" "fired" "sys_kept" "empty" "pass"

fire_runs=0 sys_ok=0 empty_total=0 pass=0
for i in $(seq 1 "$N"); do
  L="/tmp/cbench_${TASK}_${i}.log"
  MOTOKO_REPO="$REPO" bash "$ED/$TASK/run.sh" >"$L" 2>&1
  S=$(grep -aoE 'session_[a-z]+_motoko_[0-9-]+' "$L" | head -1)
  J="$REPO/.motoko/logfile/${S}.jsonl"
  read -r fired allsys empty < <(python3 -c "
import json,re,sys
try: evs=[json.loads(l) for l in open('$J') if l.strip()]
except: print('0 0 0'); sys.exit()
comp=[o.get('note','') for o in evs if o.get('type')=='compaction_extension']
sk=[int(m.group(1)) for c in comp for m in [re.search(r'sys_kept=(\\d+)',c)] if m]
empty=sum(1 for c in comp if 'AI-summarized' in c and '## Progress' not in c)
allsys=1 if (sk and all(x>=1 for x in sk)) else 0
print(len(comp), allsys, empty)
" 2>/dev/null || echo "0 0 0")
  if grep -aqE '^Result: PASS|^Result: [1-9][0-9]* passed, 0 failed' "$L"; then p=1; else p=0; fi
  [ "${fired:-0}" -gt 0 ] && fire_runs=$((fire_runs+1))
  [ "${allsys:-0}" -eq 1 ] && sys_ok=$((sys_ok+1))
  empty_total=$((empty_total+${empty:-0}))
  pass=$((pass+p))
  printf "%-5s %-7s %-9s %-7s %-6s\n" "#$i" "${fired:-0}" "$([ "${allsys:-0}" -eq 1 ] && echo OK || echo FAIL)" "${empty:-0}" "$([ "$p" -eq 1 ] && echo PASS || echo FAIL)"
done
echo "------------------------------------------------------------"
echo "fired:        $fire_runs/$N runs triggered compaction"
echo "sys_kept OK:  $sys_ok/$N runs preserved the system message on every compaction"
echo "empty sums:   $empty_total total (want 0)"
echo "converged:    $pass/$N runs PASSED"
echo "VERDICT: $([ "$fire_runs" -eq "$N" ] && [ "$sys_ok" -eq "$N" ] && [ "$empty_total" -eq 0 ] && [ "$pass" -eq "$N" ] && echo 'COMPACTION RELIABLE' || echo 'NOT yet — see rows above')"
