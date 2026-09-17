#!/bin/bash
# perf_sweep.sh — the runtime performance and memory table, as one JSON snapshot.
#
# Measures what the latency ledger and the file-size gate cannot see together:
# wall-clock p95 of the canonical workloads, PEAK RSS of five memory-shaped
# programs (live data, effect results, log lines, string and map building),
# and allocation per op of the evaluator's hot paths. Every number is
# reproducible from the command in its "how" field.
#
# Measurement rules baked in, each one learned the hard way (M-V1-MEMORY-FOOTPRINT):
#   - AILANG_NO_TRACE=1 on every timed/measured run: the default `standard` tier is
#     load-bearing on any number anyone reports (two reporters attributed the
#     tracer to `concat` and to `cons` in 2026-09).
#   - GOGC=100 on every RSS run: under the CLI's GOGC=500 peak RSS is where the GC
#     cycles land (0.96x on one machine, 1.37x on another, same binary).
#   - RSS only for LIVE-data shapes; allocation counts (B/op) for "does it copy".
#   - hardware is recorded and the diff refuses to gate across machine classes.
#
# Usage:
#   tools/perf_sweep.sh                  # print table, bank .ailang/state/perf/<date>.json
#   tools/perf_sweep.sh --json           # JSON only
#   tools/perf_sweep.sh --out <file>     # bank elsewhere
#   tools/perf_sweep.sh --quick          # 3 workload runs, count=1 benches (~2 min)
#   tools/perf_sweep.sh --control        # positive control: cons probe at deep tier
#
# bash 3.2 / BSD grep / macOS awk compatible (the rig).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

JSON_ONLY=0; OUT=""; RUNS=5; BENCH_COUNT=3; CONTROL=0
while [ $# -gt 0 ]; do
  case "$1" in
    --json) JSON_ONLY=1 ;;
    --out) shift; OUT="$1" ;;
    --quick) RUNS=3; BENCH_COUNT=1 ;;
    --control) CONTROL=1 ;;
    -h|--help) sed -n '2,26p' "$0"; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 64 ;;
  esac
  shift
done

command -v jq >/dev/null || { echo "perf_sweep: jq is required" >&2; exit 1; }

# The binary under test is built from THIS tree, never the PATH one (stale-binary trap).
BIN="$(mktemp -t ailang-sweep.XXXXXX)"
WORK="$(mktemp -d -t perf-sweep.XXXXXX)"
trap 'rm -rf "$BIN" "$WORK"' EXIT
go build -o "$BIN" ./cmd/ailang

DATE="$(date +%Y-%m-%d)"
COMMIT="$(git rev-parse --short HEAD)$(git diff --quiet HEAD -- . ':!.ailang/state' 2>/dev/null || echo -dirty)"
VERSION="$("$BIN" --version 2>/dev/null | head -1)"
case "$(uname -s)" in
  Darwin) CPU="$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo unknown)" ;;
  *) CPU="$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | sed 's/.*: //' || echo unknown)" ;;
esac
GOVER="$(go version | awk '{print $3}')"

# peak_rss_mb <cmd...>: peak resident set of one child, in MB, via /usr/bin/time.
# Darwin prints bytes ("maximum resident set size"); Linux prints KB.
peak_rss_mb() {
  local log="$WORK/time.$$"
  case "$(uname -s)" in
    Darwin) /usr/bin/time -l "$@" >/dev/null 2>"$log" || true
            awk '/maximum resident set size/{printf "%d", $1/1048576}' "$log" ;;
    *)      /usr/bin/time -v "$@" >/dev/null 2>"$log" || true
            awk '/Maximum resident set size/{printf "%d", $NF/1024}' "$log" ;;
  esac
}

# ---- 1. latency workloads (p95 ms) -----------------------------------------
LAT="$WORK/latency.json"
# --no-write prints the JSON to stdout instead of writing --output; keep the
# ledger file untouched and capture the document from stdout (it starts at "{").
tools/bench_workloads.sh --runs "$RUNS" --no-write 2>/dev/null | awk 'f||/^\{/{f=1; print}' > "$LAT" || true
jq -e '.workloads' "$LAT" >/dev/null 2>&1 || { echo "perf_sweep: bench_workloads produced no JSON" >&2; exit 1; }
lat_p95() { jq -r --arg w "$1" '.workloads[$w].p95_ms // empty' "$LAT"; }
# target p95 from the ledger row "| `name` | baseline | target | ..."
lat_target() { grep -E "^\| \`$1\`" benchmarks/budget_ledger.md | head -1 | awk -F'|' '{gsub(/ /,"",$4); print $4}'; }

# ---- 2. memory probes (peak RSS MB, GOGC=100, tracing off) ------------------
PROBES="$REPO_ROOT/benchmarks/memprobes"
cp "$PROBES"/*.ail "$WORK/"
head -c 20000000 /dev/zero | tr '\0' 'q' > "$WORK/big.txt"
export AILANG_RELAX_MODULES=1
probe() { # probe <name> <caps>
  ( cd "$WORK" && GOGC=100 AILANG_NO_TRACE=1 peak_rss_mb "$BIN" run --entry main --caps "$2" --log-level error "$1.ail" )
}
HELLO="$WORK/hello.ail"
printf 'module hello\nexport func main() -> int ! {IO} = let _ = println("hi") in 0\n' > "$HELLO"
floor_mb="$( cd "$WORK" && GOGC=100 AILANG_NO_TRACE=1 peak_rss_mb "$BIN" run --entry main --caps IO hello.ail )"
cons_mb="$(probe cons IO)"
effect_mb="$(probe effect_result IO,FS)"
debuglog_mb="$(probe debuglog IO,Debug)"
string_mb="$(probe string_build IO)"
# deep-trace ratio on the live-data probe: the tracer must not multiply live data
cons_deep_mb="$( cd "$WORK" && GOGC=100 AILANG_TRACE=deep peak_rss_mb "$BIN" run --entry main --caps IO --emit-trace jsonl cons.ail )"
if [ "$CONTROL" = 1 ]; then
  # Positive control: the diff must flag a regression. Inflate the live-data
  # probe by running it at 3x depth — a real, measured worse number, not a
  # doctored JSON, so the whole pipeline (probe → snapshot → diff) is exercised.
  sed 's/build(6000, \[\])/build(10400, [])/' "$PROBES/cons.ail" > "$WORK/cons.ail"
  cons_mb="$(probe cons IO)"
fi
cons_deep_ratio="$(awk -v a="$cons_deep_mb" -v b="$cons_mb" 'BEGIN{ if (b>0) printf "%.2f", a/b; else print "0" }')"

# ---- 3. allocation per op (Go benchmarks; min over count) --------------------
BENCHLOG="$WORK/bench.txt"
go test ./internal/eval/ -run '^$' -bench 'BenchmarkEval_(ListMapFilter|PatternMatch|StringPipeline)$|BenchmarkMapInsert5K$' -benchmem -count "$BENCH_COUNT" > "$BENCHLOG" 2>&1 || true
go test ./internal/builtins/ -run '^$' -bench 'Benchmark(ListMap50K|ListFoldl50K|ParseElements_1K)$' -benchmem -count "$BENCH_COUNT" >> "$BENCHLOG" 2>&1 || true
bench_min() { # bench_min <name> <field: ns|B>
  case "$2" in
    ns) awk -v n="$1" '$1 ~ "^"n"-" {print $3}' "$BENCHLOG" | sort -n | head -1 ;;
    B)  awk -v n="$1" '$1 ~ "^"n"-" {print $5}' "$BENCHLOG" | sort -n | head -1 ;;
  esac
}

# ---- assemble --------------------------------------------------------------
metric() { # metric <name> <value> <unit> <dir> <tol|null> <how>
  jq -n --arg k "$1" --arg v "${2:-}" --arg u "$3" --arg d "$4" --arg t "$5" --arg h "$6" \
    '{($k): {value: (if $v=="" then null else ($v|tonumber) end), unit: $u, dir: $d, tol: (if $t=="null" then null else ($t|tonumber) end), how: $h}}'
}
LATHOW="tools/bench_workloads.sh --runs $RUNS (AILANG_NO_TRACE=1, first run discarded), p95 ms; target from benchmarks/budget_ledger.md"
{
  for w in cold_hello warm_eval typecheck_heavy effect_roundtrip list_small list_large; do
    metric "lat_${w}_p95_ms" "$(lat_p95 "$w")" ms le 0.15 "$LATHOW; ledger target $(lat_target "$w")ms"
  done
  metric rss_floor_mb "$floor_mb" MB le 0.10 "peak RSS of println hello, GOGC=100 AILANG_NO_TRACE=1"
  metric rss_cons_6000_mb "$cons_mb" MB le 0.10 "benchmarks/memprobes/cons.ail depth 6000: n :: acc recursion, every frame holds a copy (LIVE data; M-LIST-CONS-QUADRATIC)"
  metric rss_cons_deep_trace_ratio "$cons_deep_ratio" x le 0.10 "cons.ail at AILANG_TRACE=deep --emit-trace jsonl / untraced; render must be bounded (M-V1-MEMORY-FOOTPRINT M1)"
  metric rss_effect_result_mb "$effect_mb" MB le 0.15 "benchmarks/memprobes/effect_result.ail: 20 x readFileResult of 20 MB via foldlE step (FS double copy + effect-trace render)"
  metric rss_debuglog_200k_mb "$debuglog_mb" MB le 0.15 "benchmarks/memprobes/debuglog.ail: 200k structured DEBUG lines under --log-level error (must stream/drop on arrival)"
  metric rss_string_build_20k_mb "$string_mb" MB le 0.15 "benchmarks/memprobes/string_build.ail: fold string accumulator over 20k pieces + join (quadratic idiom; a builder/fusion would flatten it)"
  for b in BenchmarkEval_ListMapFilter BenchmarkEval_PatternMatch BenchmarkEval_StringPipeline; do
    short="$(echo "$b" | sed 's/BenchmarkEval_//' | tr 'A-Z' 'a-z')"
    metric "eval_${short}_ns_op" "$(bench_min "$b" ns)" ns/op le 0.20 "go test ./internal/eval -bench $b -benchmem -count $BENCH_COUNT, min ns/op"
    metric "eval_${short}_b_op" "$(bench_min "$b" B)" B/op le 0.05 "same run, min B/op (deterministic: a copy shows here, not in RSS)"
  done
  metric eval_mapinsert5k_b_op "$(bench_min BenchmarkMapInsert5K B)" B/op le 0.05 "go test ./internal/eval -bench BenchmarkMapInsert5K: 5,000 MapValue.Insert, copy-on-write (RSS of the same loop swung +34% run to run; B/op is the instrument)"
  for b in BenchmarkListMap50K BenchmarkListFoldl50K BenchmarkParseElements_1K; do
    short="$(echo "$b" | sed 's/Benchmark//' | tr 'A-Z' 'a-z')"
    metric "builtin_${short}_b_op" "$(bench_min "$b" B)" B/op le 0.05 "go test ./internal/builtins -bench $b -benchmem -count $BENCH_COUNT, min B/op"
  done
} | jq -s 'add' > "$WORK/metrics.json"

SNAP="$(jq -n --arg date "$DATE" --arg commit "$COMMIT" --arg version "$VERSION" --arg cpu "$CPU" --arg os "$(uname -s)" --arg arch "$(uname -m)" --arg go "$GOVER" \
  --argjson runs "$RUNS" --argjson bench_count "$BENCH_COUNT" --argjson control "$CONTROL" --slurpfile m "$WORK/metrics.json" \
  '{schema: 1, date: $date, commit: $commit, ailang_version: $version, hardware: {cpu: $cpu, os: $os, arch: $arch, go: $go}, config: {runs: $runs, bench_count: $bench_count, gogc: 100, trace: "off", control: ($control==1)}, metrics: $m[0]}')"

if [ "$JSON_ONLY" = 1 ]; then
  if [ -n "$OUT" ]; then printf '%s\n' "$SNAP" > "$OUT"; else printf '%s\n' "$SNAP"; fi
  exit 0
fi

printf '%s\n' "$SNAP" | jq -r '"perf sweep — \(.date) @ \(.commit) — \(.ailang_version) — \(.hardware.cpu) \(.hardware.os)/\(.hardware.arch) \(.hardware.go)"'
echo
printf '%s\n' "$SNAP" | jq -r '.metrics | to_entries[] | [.key, (.value.value|tostring), .value.unit] | @tsv' \
  | awk -F'\t' 'BEGIN{printf "%-34s %14s %s\n","metric","value","unit"} {printf "%-34s %14s %s\n",$1,$2,$3}'

if [ "$CONTROL" = 1 ]; then
  echo; echo "(positive control run: NOT banked)"; exit 0
fi
DEST="${OUT:-.ailang/state/perf/$DATE.json}"
mkdir -p "$(dirname "$DEST")"
printf '%s\n' "$SNAP" > "$DEST"
echo; echo "banked: $DEST"
