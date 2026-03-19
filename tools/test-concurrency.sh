#!/usr/bin/env bash
# test-concurrency.sh — Verify serve-api handles concurrent requests correctly.
#
# IMPORTANT: Do NOT use `2>&1 | tee` with the server — it interferes with
# Go's HTTP response flushing and makes responses appear to hang.
# Redirect server output to /dev/null or a file with `> file 2>&1`.
#
# Usage:
#   ./tools/test-concurrency.sh [module-dir] [port]
#
# Examples:
#   ./tools/test-concurrency.sh examples/web_api_demo/api/     # simple modules
#   ./tools/test-concurrency.sh ../docparse/docparse/ 8081     # DocParse
#
# Environment:
#   DEBUG_CONCURRENCY=1  — Enable per-request evaluator tracing
#   CAPS="IO,FS,Env"     — Capabilities to grant (default: none)
#   AI_STUB=1            — Use stub AI handler

set -euo pipefail

MODULE_DIR="${1:?Usage: $0 <module-dir> [port]}"
PORT="${2:-8200}"
CAPS="${CAPS:-}"
TIMEOUT=3  # seconds per request — if sequential takes <100ms, 3s is generous

# Build flags
FLAGS="--port $PORT"
if [ -n "$CAPS" ]; then FLAGS="$FLAGS --caps $CAPS"; fi
if [ "${AI_STUB:-}" = "1" ]; then FLAGS="$FLAGS --ai-stub"; fi

echo "╔══════════════════════════════════════╗"
echo "║  AILANG Concurrency Test             ║"
echo "╠══════════════════════════════════════╣"
echo "║  Module: $MODULE_DIR"
echo "║  Port:   $PORT"
echo "║  Caps:   ${CAPS:-none}"
echo "╚══════════════════════════════════════╝"
echo ""

# Start server (NO tee/pipe — redirect to file)
ailang serve-api $FLAGS "$MODULE_DIR" > /tmp/concurrency_server.log 2>&1 &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null; wait $SERVER_PID 2>/dev/null" EXIT

# Wait for ready
echo -n "Starting server..."
for i in $(seq 1 30); do
  if curl -s --max-time 1 "http://localhost:$PORT/api/_health" > /dev/null 2>&1; then
    echo " ready (${i}s)"
    break
  fi
  if [ $i -eq 30 ]; then echo " TIMEOUT"; cat /tmp/concurrency_server.log | tail -5; exit 1; fi
  sleep 1
done

# Discover a function to test
MODULES=$(curl -s --max-time 3 "http://localhost:$PORT/api/_meta/modules")
FIRST_MOD=$(echo "$MODULES" | python3 -c "import sys,json; m=json.loads(sys.stdin.read())['modules'][0]; print(m['path'])" 2>/dev/null)
FIRST_FN=$(echo "$MODULES" | python3 -c "import sys,json; e=json.loads(sys.stdin.read())['modules'][0]['exports'][0]; print(e['name'])" 2>/dev/null)
ARITY=$(echo "$MODULES" | python3 -c "import sys,json; e=json.loads(sys.stdin.read())['modules'][0]['exports'][0]; print(e['arity'])" 2>/dev/null)

echo "Testing: $FIRST_MOD/$FIRST_FN (arity=$ARITY)"
echo ""

# Build request body based on arity
if [ "$ARITY" = "0" ] || [ "$ARITY" = "-1" ]; then
  BODY=""
  METHOD="GET"
  URL="http://localhost:$PORT/api/$FIRST_MOD/$FIRST_FN"
else
  BODY='{"args":["test"]}'
  METHOD="POST"
  URL="http://localhost:$PORT/api/$FIRST_MOD/$FIRST_FN"
fi

# Test 1: Sequential baseline
echo "── Sequential baseline ──"
START=$(python3 -c "import time; print(int(time.time()*1000))")
for i in $(seq 1 3); do
  if [ "$METHOD" = "GET" ]; then
    curl -s --max-time $TIMEOUT "$URL" > /dev/null
  else
    curl -s --max-time $TIMEOUT -X POST "$URL" -H 'Content-Type: application/json' -d "$BODY" > /dev/null
  fi
done
END=$(python3 -c "import time; print(int(time.time()*1000))")
SEQ_MS=$((END-START))
echo "  3 sequential: ${SEQ_MS}ms ($((SEQ_MS/3))ms each)"
echo ""

# Test 2: Concurrent (same count)
echo "── 5x concurrent ──"
rm -f /tmp/conc_test_{1,2,3,4,5}.out
START=$(python3 -c "import time; print(int(time.time()*1000))")
for i in 1 2 3 4 5; do
  if [ "$METHOD" = "GET" ]; then
    curl -s --max-time $TIMEOUT -o "/tmp/conc_test_$i.out" "$URL" &
  else
    curl -s --max-time $TIMEOUT -o "/tmp/conc_test_$i.out" -X POST "$URL" -H 'Content-Type: application/json' -d "$BODY" &
  fi
  eval "P$i=\$!"
done
wait $P1 $P2 $P3 $P4 $P5
END=$(python3 -c "import time; print(int(time.time()*1000))")
CONC_MS=$((END-START))

PASS=0; FAIL=0
for i in 1 2 3 4 5; do
  SZ=$(cat "/tmp/conc_test_$i.out" 2>/dev/null | wc -c | tr -d ' ')
  if [ "$SZ" -gt 10 ]; then
    PASS=$((PASS+1))
  else
    FAIL=$((FAIL+1))
    echo "  FAIL: request $i returned ${SZ}b"
  fi
done

echo "  5 concurrent: ${CONC_MS}ms"
echo "  Result: $PASS/5 passed"
if [ $SEQ_MS -gt 0 ]; then
  SPEEDUP=$(python3 -c "print(f'{($SEQ_MS/3*5)/$CONC_MS:.1f}')")
  echo "  Speedup: ${SPEEDUP}x vs sequential"
fi
echo ""

# Test 3: 10x stress
echo "── 10x stress ──"
rm -f /tmp/stress_test_{1,2,3,4,5,6,7,8,9,10}.out
START=$(python3 -c "import time; print(int(time.time()*1000))")
for i in $(seq 1 10); do
  if [ "$METHOD" = "GET" ]; then
    curl -s --max-time $TIMEOUT -o "/tmp/stress_test_$i.out" "$URL" &
  else
    curl -s --max-time $TIMEOUT -o "/tmp/stress_test_$i.out" -X POST "$URL" -H 'Content-Type: application/json' -d "$BODY" &
  fi
  eval "S$i=\$!"
done
wait $S1 $S2 $S3 $S4 $S5 $S6 $S7 $S8 $S9 $S10
END=$(python3 -c "import time; print(int(time.time()*1000))")
STRESS_MS=$((END-START))

PASS=0
for i in $(seq 1 10); do
  SZ=$(cat "/tmp/stress_test_$i.out" 2>/dev/null | wc -c | tr -d ' ')
  if [ "$SZ" -gt 10 ]; then PASS=$((PASS+1)); fi
done
echo "  10 concurrent: ${STRESS_MS}ms ($PASS/10 passed)"
echo ""

# Summary
echo "╔══════════════════════════════════════╗"
echo "║  Results                             ║"
echo "╠══════════════════════════════════════╣"
echo "║  Sequential: ${SEQ_MS}ms (3 requests)"
echo "║  Concurrent: ${CONC_MS}ms (5 requests)"
echo "║  Stress:     ${STRESS_MS}ms (10 requests)"
if [ $PASS -eq 10 ]; then
  echo "║  Status:     ✓ ALL PASSED"
else
  echo "║  Status:     ✗ $((10-PASS)) FAILED"
fi
echo "╚══════════════════════════════════════╝"
