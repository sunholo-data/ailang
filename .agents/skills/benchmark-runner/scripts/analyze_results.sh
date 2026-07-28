#!/bin/bash
# Analyze benchmark results: cost, tokens, time, pass rate per trial
# Usage: analyze_results.sh

BENCH_DIR="/Users/mark/dev/sunholo/ai-coding-lang-bench"

echo "════════════════════════════════════════════════════════════"
echo "  AILANG Benchmark Results"
echo "════════════════════════════════════════════════════════════"
echo ""

# Parse all log files for metrics
for f in "$BENCH_DIR"/logs/minigit-ailang*.json; do
  [ -f "$f" ] || continue
  basename "$f"
done | sort | while read fname; do
  echo "--- $fname ---"
  python3 -c "
import json
data = json.loads(open('$BENCH_DIR/logs/$fname').read())
if isinstance(data, list):
    data = [e for e in data if isinstance(e, dict) and e.get('type') == 'result'][-1]
u = data.get('usage', {})
print(f'  Cost:          \${data.get(\"total_cost_usd\", 0):.2f}')
print(f'  Output tokens: {u.get(\"output_tokens\", 0):,}')
print(f'  Cache create:  {u.get(\"cache_creation_input_tokens\", 0):,}')
print(f'  Cache read:    {u.get(\"cache_read_input_tokens\", 0):,}')
print(f'  Turns:         {data.get(\"num_turns\", 0)}')
print(f'  Time:          {data.get(\"duration_ms\", 0)/1000:.1f}s')
" 2>/dev/null
done

echo ""
echo "════════════════════════════════════════════════════════════"
echo "  Summary by Trial"
echo "════════════════════════════════════════════════════════════"
echo ""

python3 - "$BENCH_DIR" <<'PYEOF'
import json, sys, os
bench = sys.argv[1]

# Load results.json
rpath = os.path.join(bench, "results", "results.json")
if not os.path.exists(rpath):
    print("No results.json found")
    sys.exit(0)

results = json.loads(open(rpath).read())
ailang = [r for r in results if r.get("language","").startswith("ailang") and r.get("v1_time",0) > 0]

if not ailang:
    print("No AILANG results found")
    sys.exit(0)

print(f"{'Trial':>8} {'Variant':<16} {'v1 Tests':>10} {'v1 Time':>8} {'v2 Tests':>10} {'v2 Time':>8} {'LOC':>6}")
print("-" * 80)

totals = {"cost": 0, "time": 0, "trials": 0}
for r in ailang:
    lang = r.get("language","?")
    trial = r.get("trial","?")
    v1t = f"{r.get('v1_passed_count',0)}/{r.get('v1_total_count',0)}"
    v1s = f"{r.get('v1_time',0):.0f}s"
    v2t = f"{r.get('v2_passed_count',0)}/{r.get('v2_total_count',0)}"
    v2s = f"{r.get('v2_time',0):.0f}s"
    loc = r.get("v2_loc", r.get("v1_loc", 0))

    # Get cost from logs
    cost = 0
    for phase in ["v1", "v2"]:
        dn = lang.replace("/", "-")
        lf = os.path.join(bench, "logs", f"minigit-{dn}-{trial}-{phase}.json")
        if os.path.exists(lf):
            try:
                ld = json.loads(open(lf).read())
                if isinstance(ld, list):
                    ld = [e for e in ld if isinstance(e, dict) and e.get("type") == "result"][-1]
                cost += ld.get("total_cost_usd", 0)
            except: pass

    totals["cost"] += cost
    totals["time"] += r.get("v1_time",0) + r.get("v2_time",0)
    totals["trials"] += 1

    print(f"{trial:>8} {lang:<16} {v1t:>10} {v1s:>8} {v2t:>10} {v2s:>8} {loc:>6}  ${cost:.2f}")

if totals["trials"] > 0:
    print("-" * 80)
    avg_cost = totals["cost"] / totals["trials"]
    avg_time = totals["time"] / totals["trials"]
    print(f"{'Avg':>8} {'':16} {'':>10} {'':>8} {'':>10} {'':>8} {'':>6}  ${avg_cost:.2f}  {avg_time:.0f}s")

print()
print("Published baselines: Ruby $0.36/73s | Python $0.38/75s | Haskell $0.74/174s")
PYEOF
