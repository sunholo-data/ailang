#!/bin/bash
# run-suites.sh — run launchd driver suites through suite-env.sh, N at a time.
#
#   run-suites.sh <jobs> <suite.sh>... [--serial <suite.sh>...]
#
# The suites are sleep/timeout-bound, not CPU-bound: run one after another they
# took 803s of the bash-3.2 CI job (2026-10-26), most of it waiting on bounds.
# Suites before --serial run <jobs> at a time; suites after it run alone,
# afterwards — for arms that measure the machine itself (the motoko probe's
# fork-rate arm reads CPU contention as its signal).
#
# Each suite's output is buffered and printed whole when it finishes, so parallel
# suites never interleave; then a per-suite timing table. Exit 1 if any suite
# failed OR never reported (a worker that died silently is not a pass).
# bash 3.2: no `wait -n`, no assoc arrays, and "${a[@]}" on an empty array
# trips set -u — hence xargs -P, flat files, and no set -u.

here=$(cd "$(dirname "$0")" && pwd)

if [ "${1-}" = --one ]; then
  # Worker mode (invoked by xargs or the serial loop): <outdir> <suite>.
  out=$2 suite=$3
  log="$out/$(printf '%s' "$suite" | tr '/' '_').log"
  start=$(date +%s)
  /bin/bash "$here/suite-env.sh" "$suite" > "$log" 2>&1 < /dev/null
  rc=$?
  secs=$(( $(date +%s) - start ))
  if [ "$rc" -eq 0 ]; then verdict=PASS; else verdict=FAIL; fi
  until mkdir "$out/.print.lock" 2>/dev/null; do sleep 0.1; done
  echo "──── $verdict $suite (${secs}s, rc=$rc)"
  cat "$log"
  printf '%s\t%s\t%s\n' "$secs" "$rc" "$suite" >> "$out/results.tsv"
  rmdir "$out/.print.lock"
  exit 0
fi

jobs=${1-}
case "$jobs" in ''|*[!0-9]*|0) echo "usage: run-suites.sh <jobs> <suite.sh>... [--serial <suite.sh>...]" >&2; exit 2;; esac
shift

parallel_list= serial_list= mode=parallel expected=0
for arg in "$@"; do
  if [ "$arg" = --serial ]; then mode=serial; continue; fi
  [ -f "$arg" ] || { echo "run-suites: no such suite: $arg" >&2; exit 2; }
  expected=$((expected + 1))
  if [ "$mode" = parallel ]; then
    parallel_list="$parallel_list$arg
"
  else
    serial_list="$serial_list$arg
"
  fi
done
[ "$expected" -gt 0 ] || { echo "run-suites: no suites given" >&2; exit 2; }

out=$(mktemp -d "${TMPDIR:-/tmp}/launchd-suites.XXXXXX") || exit 2
trap 'rm -rf "$out"' EXIT
: > "$out/results.tsv"
start=$(date +%s)

if [ -n "$parallel_list" ]; then
  printf '%s' "$parallel_list" | tr '\n' '\0' | xargs -0 -n 1 -P "$jobs" /bin/bash "$here/run-suites.sh" --one "$out"
fi
printf '%s' "$serial_list" | while IFS= read -r suite; do
  [ -n "$suite" ] && /bin/bash "$here/run-suites.sh" --one "$out" "$suite"
done

wall=$(( $(date +%s) - start ))
reported=$(wc -l < "$out/results.tsv" | tr -d ' ')
failed=$(awk -F'\t' '$2 != 0 { n++ } END { print n + 0 }' "$out/results.tsv")

echo "──── suite timings (slowest first; wall ${wall}s, ${jobs} parallel)"
sort -rn "$out/results.tsv" | awk -F'\t' '{ printf "%5ss  %s  %s\n", $1, ($2 == 0 ? "ok  " : "FAIL"), $3 }'

if [ "$reported" -ne "$expected" ]; then
  echo "run-suites: $reported of $expected suites reported — a worker died without a verdict" >&2
  exit 1
fi
if [ "$failed" -ne 0 ]; then
  echo "run-suites: $failed of $expected suites FAILED:" >&2
  awk -F'\t' '$2 != 0 { print "  " $3 " (rc=" $2 ")" }' "$out/results.tsv" >&2
  exit 1
fi
echo "run-suites: all $expected suites passed"
