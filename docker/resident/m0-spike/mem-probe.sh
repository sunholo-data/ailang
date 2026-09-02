#!/bin/bash
# Agent-workload probe: what a Node-based coding agent CLI would see and survive.
set -uo pipefail
say() { echo "MEM | $*"; }
mkdir -p /tmp/serve && cd /tmp/serve
python3 -m http.server "${PORT:-8080}" --bind 0.0.0.0 >/dev/null 2>&1 &
HTTP_PID=$!

say "nproc=$(nproc)  MemTotal=$(grep MemTotal /proc/meminfo | awk '{print int($2/1024)}')MiB"
for f in /sys/fs/cgroup/memory.max /sys/fs/cgroup/memory.high /sys/fs/cgroup/cpu.max; do
  [ -r "$f" ] && say "$(basename $f)=$(cat $f)" || say "$(basename $f)=<unreadable>"
done
say "node: $(node --version 2>&1)"
say "node heap_size_limit: $(node -e 'console.log(Math.round(require("v8").getHeapStatistics().heap_size_limit/1048576)+" MiB")' 2>&1)"
say "node reports os.totalmem: $(node -e 'console.log(Math.round(require("os").totalmem()/1048576)+" MiB")' 2>&1)"

say "--- progressive allocation, 64MiB steps ---"
python3 - <<'PY' 2>&1 | while read -r l; do echo "MEM |   $l"; done
import sys
blocks=[]
try:
    for i in range(1, 200):
        blocks.append(bytearray(64*1024*1024))
        if i % 4 == 0:
            print(f"allocated {i*64} MiB", flush=True)
except MemoryError:
    print(f"MemoryError at {len(blocks)*64} MiB", flush=True)
PY
say "--- allocation loop returned (rc=$?) — if the container was OOM-killed you will not see this ---"
say "DONE"
wait $HTTP_PID
