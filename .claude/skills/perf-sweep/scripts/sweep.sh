#!/bin/bash
# sweep.sh — monthly perf sweep: re-measure, diff against the last banked
# snapshot, name what got slower or fatter, bank, exit 2 on a regression.
#
# Usage:
#   sweep.sh                 # full (5 workload runs, count=3 benches; ~4 min)
#   sweep.sh --quick         # 3 runs, count=1 (~2 min)
#   sweep.sh --against <snapshot.json>
#   sweep.sh --control       # positive control: must print WORSE and exit 2, banks nothing
#   sweep.sh --no-hotspots   # skip the pprof top-allocators listing
#
# bash 3.2 / BSD grep compatible.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

QUICK=""; AGAINST=""; CONTROL=""; HOTSPOTS=1
while [ $# -gt 0 ]; do
  case "$1" in
    --quick) QUICK="--quick" ;;
    --against) shift; AGAINST="$1" ;;
    --control) CONTROL="--control" ;;
    --no-hotspots) HOTSPOTS=0 ;;
    -h|--help) sed -n '2,11p' "$0"; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 64 ;;
  esac
  shift
done

SNAPDIR=.ailang/state/perf
mkdir -p "$SNAPDIR"
today="$(date +%Y-%m-%d)"
if [ -z "$AGAINST" ]; then
  AGAINST="$(find "$SNAPDIR" -maxdepth 1 -name '*.json' ! -name "$today.json" | sort | tail -1 || true)"
fi
if [ -z "$AGAINST" ] || [ ! -f "$AGAINST" ]; then
  echo "no previous snapshot to diff against; banking a baseline instead" >&2
  # shellcheck disable=SC2086
  exec /bin/bash tools/perf_sweep.sh $QUICK
fi

NOW="$(mktemp -t perf-now.XXXXXX)"
trap 'rm -f "$NOW"' EXIT
# shellcheck disable=SC2086
/bin/bash tools/perf_sweep.sh $QUICK $CONTROL --json --out "$NOW" >/dev/null

echo "perf sweep — $(jq -r .date "$NOW") @ $(jq -r .commit "$NOW")  vs  $(jq -r .date "$AGAINST") @ $(jq -r .commit "$AGAINST")"
echo "  now:  $(jq -r '.hardware | "\(.cpu) \(.os)/\(.arch) \(.go)"' "$NOW") — $(jq -r .ailang_version "$NOW")"
echo "  base: $(jq -r '.hardware | "\(.cpu) \(.os)/\(.arch) \(.go)"' "$AGAINST") — $(jq -r .ailang_version "$AGAINST")"
same_class=1
if [ "$(jq -r .hardware.cpu "$NOW")" != "$(jq -r .hardware.cpu "$AGAINST")" ] || [ "$(jq -r .hardware.os "$NOW")" != "$(jq -r .hardware.os "$AGAINST")" ]; then
  same_class=0
  echo "  !! different machine class — numbers are not comparable; the diff is shown but does not gate"
fi
echo

# One row per metric: before, now, delta %, verdict against the metric's tolerance.
jq -r --slurpfile prev "$AGAINST" '
  ($prev[0].metrics) as $p
  | .metrics | to_entries[]
  | .key as $k | .value as $m
  | ($p[$k].value // null) as $before
  | (if ($m.value|type)=="number" and ($before|type)=="number" and $before != 0 then (($m.value - $before) / $before * 100) else null end) as $pct
  | (if $pct == null then "?"
     elif $m.tol == null then (if $pct == 0 then "=" else "info" end)
     elif ($m.dir == "le" and $pct > ($m.tol*100)) or ($m.dir == "ge" and $pct < -($m.tol*100)) then "WORSE"
     elif ($m.dir == "le" and $pct < -($m.tol*100)) or ($m.dir == "ge" and $pct > ($m.tol*100)) then "better"
     else "=" end) as $v
  | [$k, ($before|tostring), ($m.value|tostring), $m.unit, (if $pct==null then "" else (if $pct>0 then "+" else "" end) + ($pct|floor|tostring) + "%" end), (if $m.tol==null then "" else "±" + (($m.tol*100)|tostring) + "%" end), $v]
  | @tsv' "$NOW" \
  | awk -F'\t' 'BEGIN{printf "%-32s %12s %12s %-6s %7s %6s  %s\n","metric","before","now","unit","delta","tol",""} {printf "%-32s %12s %12s %-6s %7s %6s  %s\n",$1,$2,$3,$4,$5,$6,$7}'

regressions="$(jq -r --slurpfile prev "$AGAINST" '
  ($prev[0].metrics) as $p
  | .metrics | to_entries[]
  | select(.value.tol != null)
  | select((.value.value|type)=="number" and (($p[.key].value // null)|type)=="number" and $p[.key].value != 0)
  | select((.value.dir=="le" and .value.value > $p[.key].value * (1 + .value.tol)) or (.value.dir=="ge" and .value.value < $p[.key].value * (1 - .value.tol)))
  | .key' "$NOW")"

echo
echo "── what to look at ──────────────────────────────────────────────"
echo "• workloads over their ledger target (benchmarks/budget_ledger.md):"
jq -r '.metrics | to_entries[] | select(.key|startswith("lat_")) | select(.value.how|test("target [0-9]+ms")) | [.key, .value.value, (.value.how|capture("target (?<t>[0-9]+)ms").t|tonumber)] | select(.[1] > .[2]) | "    \(.[0]) p95 \(.[1]) ms > target \(.[2]) ms"' "$NOW"
echo "• regressions, with the command that produced each number:"
if [ -n "$regressions" ]; then
  for k in $regressions; do
    jq -r --arg k "$k" '.metrics[$k] | "    \($k): \(.value) \(.unit) — \(.how)"' "$NOW"
  done
else
  echo "    none"
fi
if [ "$HOTSPOTS" = 1 ]; then
  echo "• top allocation sites, list_large workload (go tool pprof, alloc_space):"
  bin="$(mktemp -t ailang-prof.XXXXXX)"; prof="$(mktemp -t ailang-mem.XXXXXX)"
  if go build -o "$bin" ./cmd/ailang 2>/dev/null && AILANG_NO_TRACE=1 "$bin" run --quiet --caps IO --memprofile "$prof" benchmarks/workloads/list_large.ail >/dev/null 2>&1; then
    go tool pprof -top -sample_index=alloc_space "$bin" "$prof" 2>/dev/null | sed -n '7,14p' | sed 's/^/    /'
  else
    echo "    (profile unavailable)"
  fi
  rm -f "$bin" "$prof"
fi

if [ -n "$CONTROL" ]; then
  echo
  if [ -n "$regressions" ]; then echo "positive control: WORSE reported as expected (not banked)"; exit 2; fi
  echo "positive control FAILED: the instrument did not see a planted regression"; exit 1
fi

cp "$NOW" "$SNAPDIR/$today.json"
echo
echo "banked: $SNAPDIR/$today.json"
if [ -n "$regressions" ] && [ "$same_class" = 1 ]; then
  echo "REGRESSIONS since $(jq -r .date "$AGAINST"):"; for k in $regressions; do echo "  - $k"; done
  exit 2
fi
