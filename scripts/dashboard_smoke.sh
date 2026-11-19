#!/bin/bash
# dashboard_smoke.sh — post-deploy smoke check for the headless dashboard API
# (`ailang serve`; design_docs/planned/m-dashboard-headless-api.md).
#
#   scripts/dashboard_smoke.sh <base-url> [--expect-benchmarks]
#
# Checks STATUS CODES ONLY and never prints a response body: a body can carry
# message payloads, spans or approval records, and this runs in build logs.
#
#   /health                         GET    200
#   /                               GET    200, Content-Type text/plain (the banner, not an HTML page)
#   /api/version                    GET    200   (public; the rig's launchd drivers read it)
#   /api/observatory/spans          GET    401   without a token (the read gate is on); a
#                                                loopback base URL is exempt from the gate, so
#                                                there it must be 200
#   /api/observatory/spans          GET    200   WITH $AILANG_DASHBOARD_TOKEN, when it is set
#   /v1/traces                      POST   200   for an empty OTLP/JSON body {"resourceSpans":[]}
#                                                (writes nothing; 401/404/405 = ingest is broken)
#   /benchmarks/latest.json         POST   405   (the read-through route is registered; GET-only)
#   /benchmarks/latest.json         GET    200   only with --expect-benchmarks (dev serves the
#                                                docs-site data; test and prod have no bucket)
#   /v1/bogus                       GET    404   (the control: an unrouted path is not a 200)
#
# Exit 0 = all pass, 1 = a check failed, 2 = usage.
# Bash 3.2 compatible (the rig's /bin/bash).
set -u

usage() { echo "usage: $0 <base-url> [--expect-benchmarks]" >&2; exit 2; }

BASE="${1:-}"
[ -n "$BASE" ] || usage
shift
EXPECT_BENCH=0
for arg in "$@"; do
  case "$arg" in
    --expect-benchmarks) EXPECT_BENCH=1 ;;
    *) usage ;;
  esac
done
BASE="${BASE%/}"
case "$BASE" in
  http://*|https://*) ;;
  *) echo "dashboard_smoke: base URL must start with http:// or https://" >&2; exit 2 ;;
esac

LOOPBACK=0
case "$BASE" in
  http://127.*|http://localhost*|http://\[::1\]*) LOOPBACK=1 ;;
esac

TIMEOUT="${DASHBOARD_SMOKE_TIMEOUT:-20}"
FAILS=0

# code METHOD PATH [extra curl args...] -> prints the HTTP status ("000" on a
# transport error). The body always goes to /dev/null.
code() {
  local method="$1" path="$2"
  shift 2
  curl -sS -o /dev/null -w '%{http_code}' --max-time "$TIMEOUT" -X "$method" "$@" "$BASE$path" 2>/dev/null || true
}

check() { # check NAME WANT GOT
  if [ "$3" = "$2" ]; then
    echo "  ok    $1 -> $3"
  else
    echo "  FAIL  $1 -> $3 (want $2)"
    FAILS=$((FAILS + 1))
  fi
}

echo "dashboard smoke: $BASE"

check "GET /health" 200 "$(code GET /health)"

ROOT_CT=$(curl -sS -o /dev/null -w '%{http_code} %{content_type}' --max-time "$TIMEOUT" "$BASE/" 2>/dev/null || true)
case "$ROOT_CT" in
  "200 text/plain"*) echo "  ok    GET / -> $ROOT_CT" ;;
  *) echo "  FAIL  GET / -> ${ROOT_CT:-000} (want 200 text/plain: the headless banner, not a UI page)"; FAILS=$((FAILS + 1)) ;;
esac

check "GET /api/version" 200 "$(code GET /api/version)"

if [ "$LOOPBACK" -eq 1 ]; then
  check "GET /api/observatory/spans (loopback: gate exempt)" 200 "$(code GET '/api/observatory/spans?limit=1')"
else
  check "GET /api/observatory/spans without a token" 401 "$(code GET '/api/observatory/spans?limit=1')"
fi
if [ -n "${AILANG_DASHBOARD_TOKEN:-}" ]; then
  check "GET /api/observatory/spans with AILANG_DASHBOARD_TOKEN" 200 \
    "$(code GET '/api/observatory/spans?limit=1' -H "Authorization: Bearer ${AILANG_DASHBOARD_TOKEN}")"
fi

check "POST /v1/traces (empty OTLP/JSON)" 200 \
  "$(code POST /v1/traces -H 'Content-Type: application/json' --data '{"resourceSpans":[]}')"

check "POST /benchmarks/latest.json (route registered, GET-only)" 405 "$(code POST /benchmarks/latest.json)"
if [ "$EXPECT_BENCH" -eq 1 ]; then
  check "GET /benchmarks/latest.json" 200 "$(code GET /benchmarks/latest.json)"
fi

check "GET /v1/bogus (control)" 404 "$(code GET /v1/bogus)"

if [ "$FAILS" -gt 0 ]; then
  echo "dashboard smoke: $FAILS check(s) FAILED"
  exit 1
fi
echo "dashboard smoke: all checks passed"
