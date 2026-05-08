#!/usr/bin/env bash
#
# snapshot-fs-contention.sh — capture filesystem contention during parallel motoko runs
#
# Created for M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2) Phase 1 investigation.
# Bisects between H1 (cache-write race), H2 (env-server cross-routing), H3 (registry race)
# by snapshotting which files each motoko PID has open + which writes occur.
#
# Usage (run in a separate terminal BEFORE launching the eval-suite parallel run):
#   ./tools/snapshot-fs-contention.sh [interval_ms] [duration_sec]
#     interval_ms   — snapshot interval in milliseconds (default: 200)
#     duration_sec  — total runtime in seconds (default: 120 = 2 min)
#
# Output: /tmp/motoko-fs-contention-<timestamp>/
#   - lsof.log           lsof | grep MOTOKO snapshots, one per interval
#   - dtruss.log         dtruss -e openat,write,unlink for ONE motoko PID (if found)
#   - summary.txt        post-run summary: distinct paths written, port-bind attempts
#   - pids.log           timeline of motoko PIDs detected
#
# What to look for in the output:
#   H1 (cache race):   double-writes to MOTOKO_REPO/.ailang/cache/compile/.../core.gob
#                      from different PIDs within the same interval window
#   H2 (env-server):   bind() attempts to the same port from different PIDs
#                      (lsof TCP entries with same :PORT from multiple PIDs)
#   H3 (registry):     writes to MOTOKO_REPO/src/core/ext/registry_generated.ail or
#                      .motoko/store/ from different PIDs
#
# After the run completes, dump summary.txt for analysis.
#

set -euo pipefail

INTERVAL_MS="${1:-200}"
DURATION_SEC="${2:-120}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
OUTDIR="/tmp/motoko-fs-contention-${TIMESTAMP}"
INTERVAL_FRACTIONAL=$(awk "BEGIN { printf \"%.3f\", ${INTERVAL_MS} / 1000 }")

mkdir -p "$OUTDIR"

echo "[snapshot] writing to $OUTDIR"
echo "[snapshot] interval=${INTERVAL_MS}ms, duration=${DURATION_SEC}s"
echo "[snapshot] interval_fractional=${INTERVAL_FRACTIONAL}s"

LSOF_LOG="$OUTDIR/lsof.log"
PIDS_LOG="$OUTDIR/pids.log"
DTRUSS_LOG="$OUTDIR/dtruss.log"
SUMMARY="$OUTDIR/summary.txt"

# Track the FIRST motoko PID we observe so we can dtruss it (macOS dtruss requires sudo
# AND can only attach to one PID at a time; we cherry-pick one and accept that as the
# representative trace). On Linux, swap dtruss for `strace -f -e openat,write,unlink`.
TARGET_PID=""

# Background loop: snapshot lsof every INTERVAL_MS ms for DURATION_SEC seconds.
# stdout is the event stream; redirect to LSOF_LOG.
deadline=$(($(date +%s) + DURATION_SEC))

while [ "$(date +%s)" -lt "$deadline" ]; do
  ts=$(date +%H:%M:%S.%3N)

  # Capture all motoko-related processes (motoko binary, bun, ailang) and their open files.
  # ail-related grep: motoko (the wrapper), bun (the TS runtime), ailang (the AILANG runtime),
  # env-server-main (auto_start spawn).
  motoko_pids=$(pgrep -f 'bin/motoko|bun.*src/tui|ailang.*supervisor|env-server-main' 2>/dev/null | sort -u | head -50 || true)

  if [ -n "$motoko_pids" ]; then
    # Track newly-seen PIDs in the timeline.
    for pid in $motoko_pids; do
      if ! grep -q "^$pid$" "$PIDS_LOG" 2>/dev/null; then
        echo "$pid" >> "$PIDS_LOG"
        echo "[$ts] NEW PID $pid — $(ps -o command= -p "$pid" 2>/dev/null | head -c 200)" >> "$LSOF_LOG"
        # First motoko/ailang PID we see: target for dtruss (best-effort, requires sudo).
        if [ -z "$TARGET_PID" ] && ps -o command= -p "$pid" 2>/dev/null | grep -qE 'ailang.*supervisor|bin/motoko'; then
          TARGET_PID="$pid"
          echo "[$ts] selecting PID $TARGET_PID for dtruss" >> "$LSOF_LOG"
        fi
      fi
    done

    # Snapshot lsof for these PIDs. Filter to the paths of interest:
    #   - MOTOKO_REPO state (.ailang/cache, .motoko/store, src/core/ext/registry_generated)
    #   - TCP listening sockets (env-server bind attempts)
    pid_args=$(echo "$motoko_pids" | tr '\n' ',' | sed 's/,$//')
    {
      echo "=== $ts ==="
      lsof -p "$pid_args" 2>/dev/null | grep -E '\.ailang/cache|\.motoko/store|registry_generated|TCP.*LISTEN|TCP.*ESTABLISHED' || echo "(no matches)"
    } >> "$LSOF_LOG"
  fi

  # Sleep until next interval.
  sleep "$INTERVAL_FRACTIONAL"
done

echo "[snapshot] capture loop ended"

# Post-process: extract distinct PIDs, distinct paths written, port-bind attempts.
{
  echo "=== Summary: $TIMESTAMP ==="
  echo "Duration: ${DURATION_SEC}s, interval: ${INTERVAL_MS}ms"
  echo ""
  echo "=== Distinct motoko-family PIDs observed ==="
  if [ -f "$PIDS_LOG" ]; then
    wc -l < "$PIDS_LOG" | awk '{print "Total PIDs:", $1}'
    head -30 "$PIDS_LOG"
  else
    echo "(no PIDs observed)"
  fi
  echo ""
  echo "=== H1 candidate: distinct cache-write paths ==="
  grep -oE '/[^ ]*\.ailang/cache/[^ ]*\.gob' "$LSOF_LOG" 2>/dev/null | sort -u | head -30 || echo "(none)"
  echo ""
  echo "=== H2 candidate: TCP listening sockets ==="
  grep 'TCP.*LISTEN' "$LSOF_LOG" 2>/dev/null | awk '{print $2, $9}' | sort -u | head -30 || echo "(none)"
  echo ""
  echo "=== H3 candidate: registry/store writes ==="
  grep -E 'registry_generated|\.motoko/store' "$LSOF_LOG" 2>/dev/null | sort -u | head -30 || echo "(none)"
  echo ""
  echo "=== Same-port-from-multiple-PIDs (H2 race indicator) ==="
  grep 'TCP.*LISTEN' "$LSOF_LOG" 2>/dev/null \
    | awk '{print $9, $2}' \
    | sort -u \
    | awk '{port=$1; pid=$2; pids[port] = pids[port] " " pid; count[port]++} END {for (p in count) if (count[p] > 1) print p, "→ PIDs:", pids[p]}' \
    || echo "(none)"
} > "$SUMMARY"

echo ""
echo "[snapshot] DONE. Output dir: $OUTDIR"
echo "[snapshot] Summary:"
cat "$SUMMARY"
