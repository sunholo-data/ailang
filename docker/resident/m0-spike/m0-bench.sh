#!/bin/bash
# M0 gate benchmark. Runs once at boot, prints a report to stdout (Cloud
# Logging), then serves $PORT so the instance stays up for inspection.
set -uo pipefail
say() { echo "M0 | $*"; }

# Bind $PORT FIRST. The benchmark runs for minutes and a startup probe that
# finds nothing listening would kill the container before it reports anything.
mkdir -p /tmp/serve && cd /tmp/serve
python3 -m http.server "${PORT:-8080}" --bind 0.0.0.0 >/tmp/http.log 2>&1 &
HTTP_PID=$!
say "http server pid=$HTTP_PID on port ${PORT:-8080}"

say "=============== ENVIRONMENT ==============="
say "date: $(date -u +%FT%TZ)"
say "nproc: $(nproc)"
say "cpu model: $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2- | xargs)"
for f in /sys/fs/cgroup/cpu.max /sys/fs/cgroup/cpu.weight /sys/fs/cgroup/memory.max; do
  [ -r "$f" ] && say "$(basename $f): $(cat $f)"
done
say "MemTotal: $(grep MemTotal /proc/meminfo | awk '{print $2/1024" MiB"}')"
say "tty check: $(tty 2>&1 || true)   (expect 'not a tty')"
say "agent-home mount: $(mount | grep -c agent-home) entries"
ls -la /agent-home 2>&1 | head -3 | while read -r l; do say "  $l"; done

say "=============== HERDR HEADLESS ==============="
say "herdr version: $(herdr --version 2>&1)"
# Bare `herdr` launches the TUI — never call it. Start the server explicitly,
# detached from any terminal, stdin from /dev/null.
# NOTE: `setsid herdr server` fails silently (empty log, server never starts).
# Plain background works — verified in a TTY-less Cloud Build container.
herdr server >/tmp/herdr-server.log 2>&1 &
HERDR_PID=$!
say "herdr server pid=$HERDR_PID, waiting for socket..."
for i in $(seq 1 30); do
  if herdr status >/tmp/herdr-status.txt 2>&1; then break; fi
  sleep 1
done
say "--- herdr status ---"
sed 's/^/M0 |   /' /tmp/herdr-status.txt 2>/dev/null | head -14
say "--- server log (first 10) ---"
head -10 /tmp/herdr-server.log 2>/dev/null | sed 's/^/M0 |   /'
say "--- api snapshot (first 20 lines) ---"
herdr api snapshot 2>&1 | head -20 | sed 's/^/M0 |   /'
say "--- workspace/pane probe ---"
herdr workspace list 2>&1 | head -8 | sed 's/^/M0 |   ws: /'
herdr pane list 2>&1 | head -8 | sed 's/^/M0 |   pane: /'

say "=============== BURST BUDGET ==============="
# `go build -a std` forces a full stdlib rebuild: CPU-bound, parallel, no
# network, and reproducible on any machine with the same Go version. Three
# consecutive runs so burst-budget exhaustion shows up as a rising trend.
export GOFLAGS=-p=$(nproc)
for run in 1 2 3; do
  go clean -cache >/dev/null 2>&1
  START=$(date +%s.%N)
  go build -a std >/tmp/build-$run.log 2>&1
  RC=$?
  END=$(date +%s.%N)
  say "go build -a std  run $run: $(echo "$END - $START" | bc)s  (exit $RC)"
  [ $RC -ne 0 ] && head -5 /tmp/build-$run.log | sed 's/^/M0 |   ERR /'
done

say "=============== SINGLE-CORE ==============="
START=$(date +%s.%N)
python3 -c "
x=0
for i in range(8000000): x+=i*i
print(x)
" >/dev/null 2>&1
END=$(date +%s.%N)
say "single-core python loop: $(echo "$END - $START" | bc)s"

say "=============== DONE — idling, http pid=$HTTP_PID ==============="
wait $HTTP_PID
