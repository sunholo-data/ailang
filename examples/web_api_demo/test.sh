#!/bin/bash
# Test script for web_api_demo example
# Usage: ./test.sh
#
# Starts ailang serve-api, runs curl tests against all endpoints,
# validates responses, and reports results.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PORT=19876  # Use unusual port to avoid conflicts
PASS=0
FAIL=0
ERRORS=""

# Build ailang if needed
if ! command -v ailang &> /dev/null; then
  echo "Error: ailang not found in PATH. Run 'make install' first."
  exit 1
fi

cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# Start the API server
echo "Starting ailang serve-api on port $PORT..."
ailang serve-api --port "$PORT" "$SCRIPT_DIR/api/" > /dev/null 2>&1 &
SERVER_PID=$!
sleep 3

# Check server started
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
  echo "FAIL: Server failed to start"
  exit 1
fi

# Test helper: check response contains expected string
check() {
  local label="$1"
  local response="$2"
  local expected="$3"

  if echo "$response" | grep -qF "$expected"; then
    PASS=$((PASS + 1))
    echo "  PASS: $label"
  else
    FAIL=$((FAIL + 1))
    ERRORS="$ERRORS\n  FAIL: $label\n    Expected: $expected\n    Got: $response"
    echo "  FAIL: $label"
    echo "    Expected: $expected"
    echo "    Got: $response"
  fi
}

echo ""
echo "=== Health Check ==="
RESP=$(curl -s "http://localhost:$PORT/api/_health")
check "health status ok" "$RESP" '"status":"ok"'
check "health modules count 2" "$RESP" '"modules_count":2'

echo ""
echo "=== Module Listing ==="
RESP=$(curl -s "http://localhost:$PORT/api/_meta/modules")
check "lists api/math module" "$RESP" '"path":"api/math"'
check "lists api/greet module" "$RESP" '"path":"api/greet"'
check "count is 2" "$RESP" '"count":2'

echo ""
echo "=== Math Module ==="
RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/math/add" \
  -H "Content-Type: application/json" -d '{"args": [3, 4]}')
check "add(3,4) = 7" "$RESP" '"result":7'

RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/math/multiply" \
  -H "Content-Type: application/json" -d '{"args": [5, 6]}')
check "multiply(5,6) = 30" "$RESP" '"result":30'

RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/math/factorial" \
  -H "Content-Type: application/json" -d '{"args": [5]}')
check "factorial(5) = 120" "$RESP" '"result":120'

RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/math/fibonacci" \
  -H "Content-Type: application/json" -d '{"args": [10]}')
check "fibonacci(10) = 55" "$RESP" '"result":55'

echo ""
echo "=== Greet Module ==="
RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/greet/hello" \
  -H "Content-Type: application/json" -d '{"args": ["World"]}')
check "hello(World)" "$RESP" '"result":"Hello, World!"'

RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/greet/farewell" \
  -H "Content-Type: application/json" -d '{"args": ["Alice"]}')
check "farewell(Alice)" "$RESP" '"result":"Goodbye, Alice. Until next time!"'

# Single value body (no args wrapper)
RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/greet/hello" \
  -H "Content-Type: application/json" -d '"Bob"')
check "hello(Bob) single value" "$RESP" '"result":"Hello, Bob!"'

# JSON-returning function
RESP=$(curl -s -X POST "http://localhost:$PORT/api/api/greet/welcome" \
  -H "Content-Type: application/json" -d '{"args": ["Charlie"]}')
check "welcome(Charlie) has message" "$RESP" 'Welcome, Charlie!'

echo ""
echo "=== Error Handling ==="
# Unknown function
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  "http://localhost:$PORT/api/api/math/nope" -d '{}')
check "unknown function returns 404" "$RESP" "404"

# Unknown module
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  "http://localhost:$PORT/api/no/such/module/func" -d '{}')
check "unknown module returns 404" "$RESP" "404"

# GET not allowed
RESP=$(curl -s -o /dev/null -w "%{http_code}" \
  "http://localhost:$PORT/api/api/math/add")
check "GET returns 405" "$RESP" "405"

# CORS preflight
RESP=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS \
  "http://localhost:$PORT/api/_health")
check "OPTIONS returns 204" "$RESP" "204"

echo ""
echo "================================"
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo -e "\nFailures:$ERRORS"
  exit 1
else
  echo "All tests passed!"
fi
