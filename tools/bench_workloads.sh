#!/usr/bin/env bash
# bench_workloads.sh — M-LAT-BUDGET latency-budget probe harness
#
# Runs the canonical workloads in benchmarks/workloads/ N times each, captures
# wall-clock p50/p95 with AILANG_NO_TRACE=1, and writes a structured JSON
# baseline at benchmarks/latency_budgets.json.
#
# Usage:
#   tools/bench_workloads.sh                       # 5 runs each, default output
#   tools/bench_workloads.sh --runs 10
#   tools/bench_workloads.sh --workload list_large --runs 20 --verbose
#   tools/bench_workloads.sh --output /tmp/baseline.json --no-write   # dry-run
#
# Always sets AILANG_NO_TRACE=1 — measurements with tracing on are meaningless
# (M-PERF6B regression history). Always uses the project-local ./bin/ailang
# if present so reinstalls don't drift the baseline.
#
# Exit non-zero on missing prerequisites or workload runtime errors. Output
# regressions are NOT computed here — that's the job of `bench-check` against
# the baseline JSON.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKLOADS_DIR="$PROJECT_ROOT/benchmarks/workloads"
DEFAULT_OUTPUT="$PROJECT_ROOT/benchmarks/latency_budgets.json"

# Default workload set — order matters only for output readability
DEFAULT_WORKLOADS=(
    cold_hello
    warm_eval
    typecheck_heavy
    effect_roundtrip
    list_small
    list_large
)

# Defaults
RUNS=5
WORKLOAD=""
OUTPUT="$DEFAULT_OUTPUT"
VERBOSE=0
NO_WRITE=0

usage() {
    sed -n '2,18p' "$0" | sed 's/^# \?//'
    exit "${1:-0}"
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --runs)      RUNS="$2"; shift 2 ;;
        --workload)  WORKLOAD="$2"; shift 2 ;;
        --output)    OUTPUT="$2"; shift 2 ;;
        --verbose|-v) VERBOSE=1; shift ;;
        --no-write)  NO_WRITE=1; shift ;;
        --help|-h)   usage 0 ;;
        *) echo "unknown flag: $1" >&2; usage 1 ;;
    esac
done

if ! [[ "$RUNS" =~ ^[0-9]+$ ]] || [[ "$RUNS" -lt 1 ]]; then
    echo "error: --runs must be a positive integer (got: $RUNS)" >&2
    exit 1
fi

# Locate AILANG binary. Prefer the installed `ailang` in PATH (kept fresh by
# `make quick-install`) over the in-tree `./bin/ailang`, which can lag a
# release behind if the developer hasn't run `make build` recently.
if command -v ailang >/dev/null 2>&1; then
    AILANG="$(command -v ailang)"
elif [[ -x "$PROJECT_ROOT/bin/ailang" ]]; then
    AILANG="$PROJECT_ROOT/bin/ailang"
else
    echo "error: ailang binary not found in PATH or $PROJECT_ROOT/bin/" >&2
    echo "       run 'make quick-install' first" >&2
    exit 1
fi

# All workload runs happen from the project root with relative paths so that
# AILANG's canonical module ID matches the module declaration. Running with
# an absolute path triggers MOD010 ("module path doesn't match file path").
cd "$PROJECT_ROOT"

if ! command -v python3 >/dev/null 2>&1; then
    echo "error: python3 required for millisecond timing and JSON output" >&2
    exit 1
fi

# Resolve workload list
if [[ -n "$WORKLOAD" ]]; then
    WORKLOADS=("$WORKLOAD")
else
    WORKLOADS=("${DEFAULT_WORKLOADS[@]}")
fi

# Verify all workload files exist before doing any work
for w in "${WORKLOADS[@]}"; do
    if [[ ! -f "$WORKLOADS_DIR/$w.ail" ]]; then
        echo "error: workload not found: $WORKLOADS_DIR/$w.ail" >&2
        exit 1
    fi
done

# Hardware/version fingerprint — recorded so future-you knows whether the
# baseline is comparable to a current measurement.
HW_OS="$(uname -s)"
HW_ARCH="$(uname -m)"
case "$HW_OS" in
    Darwin) HW_CPU="$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo unknown)" ;;
    Linux)  HW_CPU="$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | sed 's/.*: //' || echo unknown)" ;;
    *)      HW_CPU="unknown" ;;
esac
GO_VERSION="$(go version 2>/dev/null | awk '{print $3}' || echo unknown)"
AILANG_VERSION="$("$AILANG" --version 2>/dev/null | head -1 || echo unknown)"
GIT_COMMIT="$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"
GIT_DIRTY=""
if ! git -C "$PROJECT_ROOT" diff-index --quiet HEAD 2>/dev/null; then
    GIT_DIRTY="-dirty"
fi
CAPTURED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "AILANG latency-budget harness"
echo "─────────────────────────────────────────────────"
echo "binary:    $AILANG"
echo "version:   $AILANG_VERSION"
echo "commit:    ${GIT_COMMIT}${GIT_DIRTY}"
echo "cpu:       $HW_CPU"
echo "os/arch:   $HW_OS/$HW_ARCH"
echo "go:        $GO_VERSION"
echo "runs:      $RUNS"
echo "workloads: ${WORKLOADS[*]}"
echo "trace:     off (AILANG_NO_TRACE=1)"
echo

# time_run "<workload-name>" → echoes "ms_int" on success, exits 1 on failure.
# Always strips ailang stdout because we only care about wall-clock duration.
time_run() {
    local name="$1"
    # Relative path is critical — see the cd "$PROJECT_ROOT" comment above.
    local file="benchmarks/workloads/$name.ail"
    local start end
    start=$(python3 -c 'import time; print(int(time.time() * 1000))')
    if ! AILANG_NO_TRACE=1 "$AILANG" run --caps IO --entry main "$file" >/dev/null 2>"$TMP_ERR"; then
        echo "  workload $name FAILED:" >&2
        cat "$TMP_ERR" >&2
        return 1
    fi
    end=$(python3 -c 'import time; print(int(time.time() * 1000))')
    echo $((end - start))
}

TMP_ERR="$(mktemp)"
trap 'rm -f "$TMP_ERR"' EXIT

# Build a Python expression that we can pipe results into. We collect
# {workload: [ms,ms,...]} on stdout and let python3 compute p50/p95 + emit
# the JSON. This avoids reinventing percentile math in bash.
declare -a JSON_ROWS=()

for w in "${WORKLOADS[@]}"; do
    printf "  %-18s " "$w"

    # Discard the first run as a warm-up if RUNS >= 3 — the first compile
    # always pays disk-cache miss + JIT-warm costs that aren't representative
    # of steady-state. With RUNS<3 we keep every sample so the user can still
    # smoke-test with --runs 1.
    times=()
    if [[ "$RUNS" -ge 3 ]]; then
        time_run "$w" >/dev/null   # warm-up, discarded
    fi
    for ((i=1; i<=RUNS; i++)); do
        if t=$(time_run "$w"); then
            times+=("$t")
            [[ $VERBOSE -eq 1 ]] && printf "%dms " "$t"
        else
            echo "abort: workload $w failed on run $i" >&2
            exit 1
        fi
    done

    # Format times as a comma-separated list for python3 to consume
    times_csv=$(IFS=,; echo "${times[*]}")
    summary=$(python3 - <<PY
times = sorted([${times_csv}])
n = len(times)
def pct(p):
    if n == 0: return 0
    # Nearest-rank percentile (no interpolation) — small N, simple is right.
    k = max(0, min(n - 1, int(round((p / 100.0) * (n - 1)))))
    return times[k]
print(f"{pct(50)} {pct(95)} {min(times)} {max(times)}")
PY
)
    p50=$(echo "$summary" | awk '{print $1}')
    p95=$(echo "$summary" | awk '{print $2}')
    pmin=$(echo "$summary" | awk '{print $3}')
    pmax=$(echo "$summary" | awk '{print $4}')

    if [[ $VERBOSE -eq 1 ]]; then
        echo
        printf "    → p50=%dms p95=%dms min=%dms max=%dms\n" "$p50" "$p95" "$pmin" "$pmax"
    else
        printf "p50=%4dms  p95=%4dms  (min %dms, max %dms)\n" "$p50" "$p95" "$pmin" "$pmax"
    fi

    # Stash row for JSON emission. Embed times array as JSON literal.
    times_json="[$(IFS=,; echo "${times[*]}")]"
    JSON_ROWS+=("\"$w\":{\"runs_ms\":$times_json,\"p50_ms\":$p50,\"p95_ms\":$p95,\"min_ms\":$pmin,\"max_ms\":$pmax}")
done

# Stitch JSON. Hand-built so we don't pull a JSON dep into the harness.
WORKLOADS_JSON="{$(IFS=,; echo "${JSON_ROWS[*]}")}"
if [[ "$RUNS" -ge 3 ]]; then
    WARMUP_DISCARDED="True"
else
    WARMUP_DISCARDED="False"
fi
JSON=$(python3 - <<PY
import json
print(json.dumps({
    "schema_version": 1,
    "captured_at": "$CAPTURED_AT",
    "ailang_version": "$AILANG_VERSION",
    "git_commit": "${GIT_COMMIT}${GIT_DIRTY}",
    "hardware": {
        "cpu": "$HW_CPU",
        "os": "$HW_OS",
        "arch": "$HW_ARCH",
        "go": "$GO_VERSION",
    },
    "config": {
        "runs": $RUNS,
        "trace": "off",
        "warmup_discarded": $WARMUP_DISCARDED,
    },
    "workloads": json.loads('''$WORKLOADS_JSON'''),
}, indent=2))
PY
)

if [[ $NO_WRITE -eq 1 ]]; then
    echo
    echo "(--no-write) skipping output file; baseline JSON follows:"
    echo "$JSON"
    exit 0
fi

mkdir -p "$(dirname "$OUTPUT")"
echo "$JSON" > "$OUTPUT"
echo
echo "wrote baseline → $OUTPUT"
