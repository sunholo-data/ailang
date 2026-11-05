#!/usr/bin/env bash
# test_rig_friction_digest.sh — fixture test for tools/rig-friction-digest.sh.
# Run: bash tools/launchd/test_rig_friction_digest.sh   (DIGEST=path overrides the script under test)
# Uses real rotation benchmark ids (the digest filters to core/stretch/frontier tiers).
set -uo pipefail
REPO="$(cd "$(dirname "$0")/../.." && pwd)"
DIGEST="${DIGEST:-$REPO/tools/rig-friction-digest.sh}"
T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
fail=0
O=opencode-qwen3-8-27b; P=pi-qwen3-8-27b; K=motoko-local-qwen3-8-27b-microrag
row() { # row VERSION BENCH MODEL OK CATEGORY [STDERR]
  mkdir -p "$T/roll/$1/agent"
  python3 -c 'import json,sys; v,b,m,ok,cat,err=sys.argv[1:7]; print(json.dumps({"id":b,"lang":"ailang","model":m,"stdout_ok":ok=="true","error_category":cat,"stderr":err}))' \
    "$1" "$2" "$3" "$4" "$5" "${6:-}" > "$T/roll/$1/agent/$2_ailang_$3_$RANDOM$RANDOM.json"
}
for i in 1 2 3; do                                              # motoko-gap: motoko 0/3, pi 3/3
  row v0.9.0 pipeline $K false logic_error; row v0.9.0 pipeline $P true none; row v0.9.0 pipeline $O true none
done
for i in 1 2 3; do                                              # harness-gap: opencode 0/3, pi 3/3
  row v0.9.0 graph_bfs $O false timeout; row v0.9.0 graph_bfs $P true none; row v0.9.0 graph_bfs $K true none
done
for m in $O $P $K; do row v0.9.0 cp_60159 $m false timeout; row v0.9.0 cp_60159 $m false timeout; done   # budget
# the SAME error kind in two benchmarks, two harnesses, different positions -> ONE ailang item, 2x
row v0.9.0 type_unify $P false compile_error "Error: type error in benchmark/solution (decl 8): ++ operator at benchmark/solution.ail:42:54: '++' is for lists only."
row v0.9.0 lfu_cache_trace $K false compile_error "=== banner ===
Error: type error in benchmark/solution (decl 3): ++ operator at benchmark/solution.ail:7:1: '++' is for lists only."
row v0.9.0 type_unify $O true none; row v0.9.0 lfu_cache_trace $O true none
row v0.9.0 pipeline $O false runtime_error "Error: a one-off slip"                 # 1x kind: digest only, not routed
row v0.9.0 quine $P false compile_error "Error: not a rotation benchmark"          # quine: not in rotation tiers
row v0.1.0 merge_sort $P false compile_error "Error: stale evidence only"           # newest bank too old
for v in v0.2.0 v0.3.0 v0.4.0 v0.5.0 v0.6.0 v0.7.0; do row $v float_eq $P true none; done   # 6 newer banks

J="$(bash "$DIGEST" --roll "$T/roll" --json 2>&1)"
lane() { python3 -c 'import json,sys; d=json.loads(sys.argv[1]); print(" ".join(sorted(i["lane"] for i in d["items"] if i["id"]==sys.argv[2])) or "-")' "$J" "$1"; }
expect() { local got; got="$(lane "$1")"; if [ "$got" = "$2" ]; then echo "PASS $1 -> $2"; else echo "FAIL $1: got [$got] want [$2]"; fail=1; fi; }
expect pipeline motoko-gap
expect graph_bfs harness-gap
expect cp_60159 budget
expect quine -
expect merge_sort -
k="$(python3 -c 'import json,sys; d=json.loads(sys.argv[1]); a=[i for i in d["items"] if i["lane"]=="ailang"]; print(len(a), a[0]["count"] if a else 0, ",".join(a[0]["benchmarks"]) if a else "")' "$J")"
k="$(python3 -c 'import json,sys; d=json.loads(sys.argv[1]); a=sorted([i for i in d["items"] if i["lane"]=="ailang"], key=lambda i:-i["count"]); print(len(a), a[0]["count"] if a else 0, ",".join(a[0]["benchmarks"]) if a else "")' "$J")"
if [ "$k" = "2 2 lfu_cache_trace,type_unify" ]; then echo "PASS error kind grouped across benchmarks [$k]"; else echo "FAIL error grouping: [$k]"; fail=1; fi

# routing: stub ailang, isolated HOME
mkdir -p "$T/bin" "$T/home/.ailang/state"; : > "$T/home/.ailang/state/mission-motoko.disabled"
printf '#!/bin/bash\n[ -f "%s/sendfail" ] && exit 1\necho "$3" >> "%s/sent"\n' "$T" "$T" > "$T/bin/ailang"; chmod +x "$T/bin/ailang"
route() { PATH="$T/bin:$PATH" HOME="$T/home" bash "$DIGEST" --roll "$T/roll" --route >/dev/null 2>&1; }
: > "$T/sendfail"; route; rc=$?
if [ "$rc" = 1 ] && [ ! -s "$T/home/.ailang/state/rig-friction-routed" ]; then echo "PASS failed send exits 1, remembers nothing"; else echo "FAIL failed send: rc=$rc state=[$(cat "$T/home/.ailang/state/rig-friction-routed" 2>/dev/null)]"; fail=1; fi
rm -f "$T/sendfail"; route
sent="$(sort "$T/sent" 2>/dev/null | tr '\n' ' ')"
if [ "$sent" = "ailang-core user " ]; then echo "PASS first route -> ailang-core + user (motoko paused)"; else echo "FAIL first route sent to [$sent]"; fail=1; fi
if grep -q "one-off" "$T/home/.ailang/state/rig-friction-routed"; then echo "FAIL a 1x error kind was routed"; fail=1; else echo "PASS 1x error kind not routed"; fi
: > "$T/sent"; route
if [ ! -s "$T/sent" ]; then echo "PASS second route re-sends nothing"; else echo "FAIL re-sent: $(cat "$T/sent")"; fail=1; fi
bash "$DIGEST" --bogus >/dev/null 2>&1; rc=$?
if [ "$rc" = 2 ]; then echo "PASS usage error exits 2"; else echo "FAIL usage rc=$rc"; fail=1; fi
exit $fail
