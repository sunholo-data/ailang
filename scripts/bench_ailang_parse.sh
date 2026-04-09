#!/usr/bin/env bash
# bench_ailang_parse.sh — M5 acceptance benchmark.
#
# Compares wall-clock execution of `ailang run` (evaluator) vs
# `ailang run --bytecode` (VM) on the runnable example corpus.
# Reports per-file and aggregate speedup.
#
# Usage:
#   scripts/bench_ailang_parse.sh              # full corpus
#   scripts/bench_ailang_parse.sh --iters 5    # more samples
#   scripts/bench_ailang_parse.sh --only fib   # filter by name
#   scripts/bench_ailang_parse.sh --min-ms 80  # skip startup-dominated

set -euo pipefail

ITERS=3
ONLY=""
MIN_MS=80
TIMEOUT=30

while [[ $# -gt 0 ]]; do
  case "$1" in
    --iters)  ITERS="$2"; shift 2 ;;
    --only)   ONLY="$2"; shift 2 ;;
    --min-ms) MIN_MS="$2"; shift 2 ;;
    --timeout) TIMEOUT="$2"; shift 2 ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

AILANG="${AILANG:-ailang}"
CORPUS_DIR="examples/runnable"

if [[ ! -d "$CORPUS_DIR" ]]; then
  echo "ERROR: $CORPUS_DIR not found. Run from repo root." >&2
  exit 1
fi

# Collect files
FILES=()
for f in "$CORPUS_DIR"/*.ail; do
  base="$(basename "$f")"
  [[ -n "$ONLY" && "$base" != *"$ONLY"* ]] && continue
  FILES+=("$f")
done

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "No matching .ail files found."
  exit 1
fi

echo "═══════════════════════════════════════════════════════════"
echo " AILANG Bytecode VM Benchmark (iters=$ITERS, min=${MIN_MS}ms)"
echo "═══════════════════════════════════════════════════════════"
echo ""

# best_of_n FILE FLAGS
# Runs N times, returns best wall-clock in ms (integer).
best_of_n() {
  local file="$1"
  shift
  local best=999999
  for ((i=0; i<ITERS; i++)); do
    local start end elapsed
    start=$(python3 -c "import time; print(int(time.time()*1000))")
    timeout "$TIMEOUT" "$AILANG" run "$@" "$file" >/dev/null 2>&1 || true
    end=$(python3 -c "import time; print(int(time.time()*1000))")
    elapsed=$((end - start))
    [[ $elapsed -lt $best ]] && best=$elapsed
  done
  echo "$best"
}

total_eval=0
total_vm=0
counted=0
speedups=()

printf "%-40s %8s %8s %8s  %s\n" "FILE" "EVAL(ms)" "VM(ms)" "SPEEDUP" "NOTE"
printf "%-40s %8s %8s %8s  %s\n" "$(printf '%0.s─' {1..40})" "────────" "────────" "────────" "────"

for f in "${FILES[@]}"; do
  base="$(basename "$f")"

  eval_ms=$(best_of_n "$f")
  vm_ms=$(best_of_n "$f" --bytecode)

  note=""
  if [[ $eval_ms -lt $MIN_MS ]]; then
    note="startup-dominated"
    printf "%-40s %8d %8d %8s  %s\n" "$base" "$eval_ms" "$vm_ms" "—" "$note"
    continue
  fi

  if [[ $vm_ms -gt 0 ]]; then
    # Use bc for floating point division
    speedup=$(echo "scale=2; $eval_ms / $vm_ms" | bc)
  else
    speedup="inf"
  fi

  total_eval=$((total_eval + eval_ms))
  total_vm=$((total_vm + vm_ms))
  counted=$((counted + 1))
  speedups+=("$speedup")

  printf "%-40s %8d %8d %8sx  %s\n" "$base" "$eval_ms" "$vm_ms" "$speedup" "$note"
done

echo ""
echo "═══════════════════════════════════════════════════════════"
echo " SUMMARY ($counted files above ${MIN_MS}ms threshold)"
echo "═══════════════════════════════════════════════════════════"

if [[ $counted -gt 0 && $total_vm -gt 0 ]]; then
  agg_speedup=$(echo "scale=2; $total_eval / $total_vm" | bc)
  echo "  Total eval time: ${total_eval}ms"
  echo "  Total VM time:   ${total_vm}ms"
  echo "  Aggregate speedup: ${agg_speedup}x"

  # Median speedup
  sorted=($(printf '%s\n' "${speedups[@]}" | sort -n))
  mid=$((counted / 2))
  echo "  Median speedup:    ${sorted[$mid]}x"

  # Gate check
  pass=$(echo "$agg_speedup >= 3.0" | bc)
  if [[ "$pass" == "1" ]]; then
    echo ""
    echo "  ✓ GATE PASSED: aggregate speedup ≥ 3×"
  else
    echo ""
    echo "  ✗ GATE FAILED: aggregate speedup < 3× (target: ≥3×, ideal: 5×)"
    echo "    Re-open disasm and find the next bottleneck."
  fi
else
  echo "  No files above threshold — cannot compute speedup."
fi
echo "═══════════════════════════════════════════════════════════"
