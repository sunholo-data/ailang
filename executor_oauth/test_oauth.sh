#!/usr/bin/env bash
# End-to-end test of the executor's public OAuth service (M-AILANG-EXECUTOR-GE
# P1b) under a real `serve-api --mcp-connect-only`, with stub storage
# (/tmp/executor_oauth) and the test sign-in seam instead of Firebase.
#
#   executor_oauth/test_oauth.sh                  # from the ailang repo root
#   ROOT=<dir with the three tool dirs> AILANG=<binary> executor_oauth/test_oauth.sh
#
# The CIMD fetch is live: client_id is Claude Code's published metadata
# document, so this needs network access to claude.ai.
#
# Checks: no ungated way to a tool (/mcp/, the @route, the /api/ catch-all);
# the 401 challenge and both metadata documents; authorize -> sign-in -> code
# -> token; the token runs ailang_eval; a refresh rotates it; a domain outside
# the allowlist cannot sign in; the server log names the account and never the
# token; and the test seam is shut when K_SERVICE is set (as on Cloud Run).
set -uo pipefail
ROOT="${ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
AILANG="${AILANG:-ailang}"
P="${PORT:-18931}"
BASE="http://localhost:$P"
LOG=$(mktemp)
SRV=""
rc=0
pass() { echo "PASS $1"; }
fail() { echo "FAIL $1: $2"; rc=1; }

TEST_ID_TOKEN="test-id-token-0123456789abcdef0123456789"
start_server() { # extra env as args, e.g. K_SERVICE=x
  rm -rf /tmp/executor_oauth
  # A server left on the port would answer instead of this one.
  if curl -s -o /dev/null "$BASE/"; then echo "port $P is already in use"; exit 1; fi
  # exec, so $SRV is serve-api itself and stop_server really stops it.
  (cd "$ROOT" && exec env "$@" FIRESTORE_STUB=true OAUTH_ISSUER="$BASE" GOOGLE_CLOUD_PROJECT=test-project \
    EXECUTOR_ALLOWED_EMAIL_DOMAINS=sunholo.com EXECUTOR_OAUTH_TEST_ID_TOKEN="$TEST_ID_TOKEN" \
    EXECUTOR_OAUTH_TEST_EMAIL="${TEST_EMAIL:-dev@sunholo.com}" \
    EXECUTOR_POLICY="$ROOT/docker/executor-policy.toml" EXECUTOR_AILANG_BIN="$(command -v "$AILANG")" \
    EXECUTOR_WORK_DIR="$(mktemp -d)" AILANG_RELAX_MODULES=1 AILANG_TRACE_VALUES=off AILANG_NO_CACHE=1 \
    "$AILANG" serve-api --port "$P" --mcp-http --routes-only --mcp-connect-only --oauth-issuer "$BASE" \
      --caps Clock,FS,Env,IO,Net,Process,Rand --static executor_oauth/static \
      --process-timeout 90s --process-allowlist "$(command -v "$AILANG"):eval,$(command -v "$AILANG"):run,$(command -v "$AILANG"):ai-check,$(command -v "$AILANG"):prompt,$(command -v "$AILANG"):builtins" \
      mcp_tools/ executor_tools/ executor_oauth/ >"$LOG" 2>&1) &
  SRV=$!
  for _ in $(seq 1 60); do curl -s -o /dev/null "$BASE/.well-known/oauth-authorization-server" && return 0; sleep 1; done
  echo "server did not start:"; tail -20 "$LOG"; exit 1
}
stop_server() { [ -n "$SRV" ] && kill "$SRV" 2>/dev/null; wait "$SRV" 2>/dev/null; SRV=""; }
trap 'stop_server; rm -f "$LOG"' EXIT

CLIENT="https://claude.ai/oauth/claude-code-client-metadata"
CB="http://localhost:33418/callback"
VERIFIER="dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
CHALLENGE="E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
loc() { curl -s -o /dev/null -w '%{redirect_url}' "$@"; }
qparam() { sed -n "s/.*[?&]$2=\([^&]*\).*/\1/p" <<<"$1"; }
mcp() { # token method params -> "status body"
  curl -s -w '\nHTTP%{http_code}' -X POST "$BASE/mcp/connect/" -H 'content-type: application/json' \
    -H 'accept: application/json, text/event-stream' ${1:+-H "Authorization: Bearer $1"} \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$2\",\"params\":$3}"
}
EVAL='{"name":"ailang_eval","arguments":{"expression":"6 * 7"}}'
authorize() { loc -G "$BASE/oauth/authorize" --data-urlencode response_type=code --data-urlencode "client_id=$CLIENT" \
  --data-urlencode "redirect_uri=$CB" --data-urlencode "code_challenge=$CHALLENGE" \
  --data-urlencode code_challenge_method=S256 --data-urlencode state=st1; }

start_server

# --- No ungated way to a tool ---
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/mcp/" -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":$EVAL}")
[[ "$R" != *HTTP200* && "$R" != *'42'* ]] && pass "/mcp/ not served" || fail "/mcp/ not served" "$R"
for u in /api/executor/ailang_eval /api/executor_tools/executor/ailangEval; do
  R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE$u" -H 'content-type: application/json' -d '{"expression":"6 * 7"}')
  [[ "$R" != *HTTP200* ]] && pass "POST $u refused" || fail "POST $u refused" "$R"
done
R=$(mcp "" tools/list '{}')
[[ "$R" == *HTTP200* && "$R" == *'"ailang_run"'* ]] && pass "tools/list open" || fail "tools/list open" "$R"
H=$(curl -s -D - -o /dev/null -X POST "$BASE/mcp/connect/" -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' \
  -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":$EVAL}")
[[ "$H" == *" 401"* && "$H" == *"resource_metadata="* ]] && pass "tool call without token -> 401 challenge" || fail "tool call without token -> 401 challenge" "$H"
R=$(curl -s "$BASE/.well-known/oauth-protected-resource/mcp/connect/")
[[ "$R" == *"\"authorization_servers\":[\"$BASE\"]"* ]] && pass "protected-resource metadata names the issuer" || fail "protected-resource metadata" "$R"
R=$(curl -s "$BASE/.well-known/oauth-authorization-server")
[[ "$R" == *"\"issuer\":\"$BASE\""* && "$R" == *S256* ]] && pass "authorization-server metadata" || fail "authorization-server metadata" "$R"

R=$(curl -s -w ' HTTP%{http_code}' "$BASE/oauth-login.html")
[[ "$R" == *HTTP200* && "$R" == *"Connect an app"* ]] && pass "sign-in page served same-origin" || fail "sign-in page served same-origin" "$(tail -c 80 <<<"$R")"
R=$(curl -s "$BASE/vendor/")
[[ "$R" != *firebase-app-compat.js* ]] && pass "no directory listing" || fail "no directory listing" "$R"

# --- Sign in, get a token, use it ---
L1=$(authorize); HANDLE=$(qparam "$L1" handle)
[[ "$L1" == "$BASE/oauth-login.html?handle="* && -n "$HANDLE" ]] && pass "authorize -> same-origin sign-in page" || fail "authorize -> sign-in page" "location=$L1"
R=$(curl -s "$BASE/oauth/request?handle=$HANDLE")
[[ "$R" == *'"client_host":"claude.ai"'* && "$R" == *'"redirect_host":"localhost:33418"'* ]] && pass "consent details" || fail "consent details" "$R"
L2=$(loc -X POST "$BASE/oauth/login/complete" --data-urlencode "handle=$HANDLE" --data-urlencode "id_token=$TEST_ID_TOKEN")
CODE=$(qparam "$L2" code)
[[ "$L2" == "$CB?"* && -n "$CODE" ]] && pass "sign-in returns a code" || fail "sign-in returns a code" "location=$L2"
TOK=$(curl -s -X POST "$BASE/oauth/token" --data-urlencode grant_type=authorization_code --data-urlencode "code=$CODE" \
  --data-urlencode "code_verifier=$VERIFIER" --data-urlencode "client_id=$CLIENT" --data-urlencode "redirect_uri=$CB")
ACCESS=$(sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p' <<<"$TOK")
REFRESH=$(sed -n 's/.*"refresh_token":"\([^"]*\)".*/\1/p' <<<"$TOK")
[[ "$ACCESS" == ax_* && ${#ACCESS} -eq 67 && -n "$REFRESH" && "$TOK" == *'"expires_in":3600'* ]] && pass "token issued (ax_, 1h)" || fail "token issued" "$TOK"
R=$(mcp "$ACCESS" tools/call "$EVAL")
[[ "$R" == *HTTP200* && "$R" == *42* ]] && pass "token runs ailang_eval" || fail "token runs ailang_eval" "$R"
R=$(mcp "ax_0000000000000000000000000000000000000000000000000000000000000000" tools/call "$EVAL")
[[ "$R" == *HTTP401* ]] && pass "unknown token -> 401" || fail "unknown token -> 401" "$R"

# --- Refresh rotates ---
TOK2=$(curl -s -X POST "$BASE/oauth/token" --data-urlencode grant_type=refresh_token --data-urlencode "refresh_token=$REFRESH" --data-urlencode "client_id=$CLIENT")
ACCESS2=$(sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p' <<<"$TOK2")
R=$(mcp "$ACCESS2" tools/call "$EVAL")
[[ -n "$ACCESS2" && "$ACCESS2" != "$ACCESS" && "$R" == *HTTP200* ]] && pass "refresh issues a working new token" || fail "refresh" "$TOK2 / $R"
R=$(curl -s -X POST "$BASE/oauth/token" --data-urlencode grant_type=refresh_token --data-urlencode "refresh_token=$REFRESH" --data-urlencode "client_id=$CLIENT")
[[ "$R" == *invalid_grant* ]] && pass "refresh token reuse refused" || fail "refresh token reuse refused" "$R"

# --- Log: account named, token never ---
sleep 1
grep -q '"event":"executor_auth","outcome":"ok","account":"test:dev@sunholo.com","client":"claude.ai"' "$LOG" && pass "auth log names the account" || fail "auth log names the account" "$(grep executor_auth "$LOG" | head -2)"
if [ -z "$ACCESS" ] || [ -z "$REFRESH" ] || [ -z "$CODE" ]; then fail "log never holds a token or code" "nothing was issued to look for"
elif grep -qF "$ACCESS" "$LOG" || grep -qF "$REFRESH" "$LOG" || grep -qF "$CODE" "$LOG"; then fail "log never holds a token or code" "leaked"
else pass "log never holds a token or code"; fi
stop_server

# --- A domain outside the allowlist cannot sign in ---
TEST_EMAIL=someone@gmail.com start_server
HANDLE=$(qparam "$(authorize)" handle)
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/login/complete" --data-urlencode "handle=$HANDLE" --data-urlencode "id_token=$TEST_ID_TOKEN")
[[ "$R" == *HTTP400* && "$R" == *"not allowed to sign in"* ]] && pass "gmail.com refused under a sunholo.com allowlist" || fail "domain allowlist" "$R"
stop_server

# --- The test seam is shut on Cloud Run ---
start_server K_SERVICE=executor-mcp-oauth
HANDLE=$(qparam "$(authorize)" handle)
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/login/complete" --data-urlencode "handle=$HANDLE" --data-urlencode "id_token=$TEST_ID_TOKEN")
# The handle exists (storage works), so the refusal is the sign-in itself:
# the token went to real Firebase verification and failed there.
[[ -n "$HANDLE" && "$R" == *HTTP400* && "$R" != *"code="* ]] && pass "test seam shut when K_SERVICE is set" || fail "test seam shut when K_SERVICE is set" "$R"
stop_server

[ $rc -eq 0 ] && echo "ok: executor oauth e2e"
exit $rc
