#!/bin/bash
# VM vs evaluator on HOF folds (M-FOLDL-CONS-COST-MODEL Phase 3, #1501).
#
#   bench/vm_hof_callbacks/run.sh [ailang-binary] [runs]
#
# Prints min-of-N wall seconds per (program, N, engine):
#   consrepro  foldl(\acc x. x :: acc)  -- O(n) step, quadratic; copy-bound
#   sumfold    foldl(\acc x. acc + x)   -- O(1) step; per-callback cost
#   mapfilt    filter(.., map(.., xs))  -- O(1) closures; per-callback cost
# The Go-level counterpart is `go test ./internal/vm/ -bench HOFFoldl`.
cd "$(dirname "$0")" || exit 1
B=${1:-ailang}
RUNS=${2:-3}
now() { python3 -c 'import time; print(time.time())'; }
t() { # program n [--bytecode]
  local f=$1 n=$2 best=999999 s e
  shift 2
  for _ in $(seq "$RUNS"); do
    s=$(now)
    "$B" run --quiet "$@" --caps IO --args-json "$n" "$f.ail" >/dev/null 2>&1 || echo "FAIL $f $n $*"
    e=$(now)
    best=$(python3 -c "print(min($best, $e - $s))")
  done
  printf "%-10s %-8s %-10s %.2f\n" "$f" "$n" "${1:-interp}" "$best"
}
for n in 40000 80000; do t consrepro $n; t consrepro $n --bytecode; done
for n in 1000000 5000000; do t sumfold $n; t sumfold $n --bytecode; done
for n in 1000000 5000000; do t mapfilt $n; t mapfilt $n --bytecode; done
