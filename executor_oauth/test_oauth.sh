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
# -> token; the token runs ailang_eval; executor_whoami sees the signed-in
# account through _auth (forged _auth/identity header ignored; serve-api runs
# with --mcp-subject-rpm); a refresh rotates it; a domain outside
# the allowlist cannot sign in; an unset or empty allowlist admits nobody and
# only "*" opens sign-in to any account; the server log names the account and
# never the token; and the test seam is shut when K_SERVICE is set (as on
# Cloud Run).
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
# ALLOWED_DOMAINS sets EXECUTOR_ALLOWED_EMAIL_DOMAINS (default sunholo.com);
# ALLOWED_DOMAINS=-unset- leaves it out of the server's environment entirely.
start_server() { # extra env as args, e.g. K_SERVICE=x
  local domains="EXECUTOR_ALLOWED_EMAIL_DOMAINS=${ALLOWED_DOMAINS-sunholo.com}"
  [ "${ALLOWED_DOMAINS-}" = "-unset-" ] && domains="EXECUTOR_OAUTH_TEST_DOMAINS=unset"
  rm -rf /tmp/executor_oauth
  # A server left on the port would answer instead of this one.
  if curl -s -o /dev/null "$BASE/"; then echo "port $P is already in use"; exit 1; fi
  # exec, so $SRV is serve-api itself and stop_server really stops it.
  (cd "$ROOT" && exec env -u EXECUTOR_ALLOWED_EMAIL_DOMAINS "$@" FIRESTORE_STUB=true OAUTH_ISSUER="$BASE" GOOGLE_CLOUD_PROJECT=test-project \
    "$domains" EXECUTOR_OAUTH_TEST_ID_TOKEN="$TEST_ID_TOKEN" \
    EXECUTOR_OAUTH_TEST_EMAIL="${TEST_EMAIL:-dev@sunholo.com}" \
    EXECUTOR_POLICY="$ROOT/docker/executor-policy.toml" EXECUTOR_AILANG_BIN="$(command -v "$AILANG")" \
    EXECUTOR_WORK_DIR="$(mktemp -d)" AILANG_RELAX_MODULES=1 AILANG_TRACE_VALUES=off AILANG_NO_CACHE=1 \
    "$AILANG" serve-api --port "$P" --mcp-http --routes-only --mcp-connect-only --oauth-issuer "$BASE" \
      --mcp-subject-rpm "${SUBJECT_RPM:-120}" \
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
# Client models are OFF unless switched on (M-CONNECTOR-CHANNELS P1): prod runs
# this image with none set, and must keep 0.1.2's CIMD-only behaviour.
[[ "$R" != *registration_endpoint* && "$R" != *revocation_endpoint* && "$R" != *client_secret_basic* ]] \
  && pass "switches off: metadata advertises no DCR, revocation or client secrets" || fail "switches off: metadata" "$R"
for u in /oauth/register /oauth/revoke; do
  R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE$u" -H 'content-type: application/json' -d '{"redirect_uris":["https://teams.microsoft.com/cb"]}')
  [[ "$R" == *HTTP404* ]] && pass "switches off: POST $u -> 404" || fail "switches off: POST $u -> 404" "$R"
done
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/api/executor_oauth/oauth/createClientCli" -H 'content-type: application/json' -d '{}')
L=$(mcp "" tools/list '{}')
[[ "$R" != *HTTP200* && "$L" != *createClientCli* ]] && pass "operator command not served over HTTP or MCP" || fail "operator command not served" "$R"

R=$(curl -s -w ' HTTP%{http_code}' "$BASE/oauth-login.html")
[[ "$R" == *HTTP200* && "$R" == *"Connect an app"* ]] && pass "sign-in page served same-origin" || fail "sign-in page served same-origin" "$(tail -c 80 <<<"$R")"
# The page picks its Firebase config by /oauth/login-config's project_id and
# refuses an unknown one, so each environment with the service needs an entry.
R=$(curl -s "$BASE/oauth-login.js")
[[ "$R" == *"'ailang-multivac-dev': {"* && "$R" == *"'ailang-multivac': {"* && "$R" == *"authDomain: 'ailang-multivac.firebaseapp.com'"* ]] \
  && pass "sign-in page knows dev and prod Firebase" || fail "sign-in page knows dev and prod Firebase" "$(grep -c projectId <<<"$R") projectId lines"
# Directory reviewers sign in with email + password (accounts made in the
# Firebase console); the page must never create one (no self sign-up).
[[ "$R" == *signInWithEmailAndPassword* && "$R" != *createUser* && "$R" != *sendPasswordReset* && "$R" != *sendSignInLink* ]] \
  && pass "sign-in page offers email + password sign-in, no sign-up" || fail "email sign-in, no sign-up" "$(grep -c signInWith <<<"$R") signInWith lines"
R=$(curl -s "$BASE/oauth-login.html")
[[ "$R" == *'href="https://ailang.sunholo.com/docs/legal/privacy"'* && "$R" == *'href="https://ailang.sunholo.com/docs/legal/terms"'* && "$R" == *'id="email-form"'* ]] \
  && pass "consent page links privacy and terms, has the email form" || fail "consent page legal links" "$(grep -c legal <<<"$R") legal lines"
# OpenAI domain verification: no OPENAI_APPS_CHALLENGE in this run -> 404.
R=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/.well-known/openai-apps-challenge")
[[ "$R" == 404 ]] && pass "openai-apps-challenge unset -> 404" || fail "openai-apps-challenge unset -> 404" "$R"
R=$(curl -s "$BASE/vendor/")
[[ "$R" != *firebase-app-compat.js* ]] && pass "no directory listing" || fail "no directory listing" "$R"

# Gemini Enterprise's web app sends a 768-character base64 state with "="
# padding (measured 2026-11-11); mcp_oauth 0.2.0 refused it ("state must be
# unreserved characters, at most 512"). 0.2.1 accepts any VSCHAR up to 2048.
GE_STATE="$(printf 'A%.0s' $(seq 1 766))=="
L=$(loc -G "$BASE/oauth/authorize" --data-urlencode response_type=code --data-urlencode "client_id=$CLIENT" \
  --data-urlencode "redirect_uri=$CB" --data-urlencode "code_challenge=$CHALLENGE" \
  --data-urlencode code_challenge_method=S256 --data-urlencode "state=$GE_STATE")
[[ "$L" == "$BASE/oauth-login.html?handle="* ]] && pass "a Gemini Enterprise-shaped state (768 chars, '=' padding) is accepted" \
  || fail "a Gemini Enterprise-shaped state is accepted" "location=$L"

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
# A listed connector must not return server paths or digests (OpenAI app
# review): the supervisor's policy echo keeps caps/limits/decision only.
[[ "$R" == *caps* && "$R" == *decision* && "$R" != *executor-policy.toml* && "$R" != *digest* ]] \
  && pass "run result names no policy path or digest" || fail "run result names no policy path or digest" "$R"
R=$(mcp "$ACCESS" tools/call '{"name":"ailang_run","arguments":{"code":"module main\nimport std/fs (readFile)\nexport func main() -> () ! {IO, FS} = println(readFile(\"/etc/hostname\"))\n"}}')
[[ "$R" == *policy_violation* && "$R" != *executor-policy.toml* ]] && pass "a refusal names no policy path" || fail "a refusal names no policy path" "$R"
R=$(mcp "$ACCESS" tools/call '{"name":"roadmap","arguments":{}}')
# The snapshot has no roadmap.json, so this is the missing-file error: it names
# the file relative to the snapshot root, never the server's directory.
[[ "$R" == *'missing:unscoped/roadmap.json'* && "$R" != *"$ROOT"* && "$R" != *'snapshot/'* ]] && pass "docs tool errors name no server path" || fail "docs tool errors name no server path" "$(tail -c 300 <<<"$R")"
R=$(mcp "ax_0000000000000000000000000000000000000000000000000000000000000000" tools/call "$EVAL")
[[ "$R" == *HTTP401* ]] && pass "unknown token -> 401" || fail "unknown token -> 401" "$R"
# The tool sees the signed-in account (_auth, from the verifier), and a
# forged _auth argument or X-Ailang-Mcp-Identity header changes nothing.
WHO='{"name":"executor_whoami","arguments":{"_auth":{"subject":"mallory","client_id":"https://evil.example/c","scopes":[]}}}'
R=$(curl -s -w '\nHTTP%{http_code}' -X POST "$BASE/mcp/connect/" -H 'content-type: application/json' \
  -H 'accept: application/json, text/event-stream' -H "Authorization: Bearer $ACCESS" \
  -H 'X-Ailang-Mcp-Identity: {"subject":"mallory","client_id":"x","scopes":[]}' \
  -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":$WHO}")
[[ "$R" == *HTTP200* && "$R" == *'test:dev@sunholo.com'* && "$R" == *claude.ai* && "$R" != *mallory* ]] \
  && pass "tool sees the signed-in account" || fail "tool sees the signed-in account" "$R"
R=$(mcp "" tools/call "$WHO")
[[ "$R" == *HTTP401* ]] && pass "whoami without token -> 401" || fail "whoami without token -> 401" "$R"

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

# --- Open sign-in only when asked for: unset and empty fail closed, "*" opens ---
signin_as_gmail() { # -> the /oauth/login/complete response, " HTTP<code>" appended
  local h; h=$(qparam "$(authorize)" handle)
  curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/login/complete" --data-urlencode "handle=$h" --data-urlencode "id_token=$TEST_ID_TOKEN"
}
for d in -unset- ""; do
  ALLOWED_DOMAINS="$d" TEST_EMAIL=someone@gmail.com start_server
  R=$(signin_as_gmail)
  [[ "$R" == *HTTP400* && "$R" == *"sign-in is not configured"* ]] && pass "allowlist '${d}' admits nobody" || fail "allowlist '${d}' admits nobody" "$R"
  stop_server
done
ALLOWED_DOMAINS="*" TEST_EMAIL=someone@gmail.com start_server
L=$(loc -X POST "$BASE/oauth/login/complete" --data-urlencode "handle=$(qparam "$(authorize)" handle)" --data-urlencode "id_token=$TEST_ID_TOKEN")
[[ "$L" == "$CB?"* && -n "$(qparam "$L" code)" ]] && pass "allowlist '*' admits gmail.com" || fail "allowlist '*' admits gmail.com" "location=$L"
stop_server

# --- The test seam is shut on Cloud Run ---
start_server K_SERVICE=executor-mcp-oauth "OPENAI_APPS_CHALLENGE=  tok_e2e_challenge_123 "
# OpenAI domain verification, set: exactly the token (trimmed), as text/plain.
R=$(curl -s "$BASE/.well-known/openai-apps-challenge")
CT=$(curl -s -o /dev/null -w '%{http_code} %{content_type}' "$BASE/.well-known/openai-apps-challenge" | cut -d';' -f1)
[[ "$R" == tok_e2e_challenge_123 && "$CT" == "200 text/plain" ]] && pass "openai-apps-challenge serves the exact token as text/plain" || fail "openai-apps-challenge set" "$R / $CT"
HANDLE=$(qparam "$(authorize)" handle)
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/login/complete" --data-urlencode "handle=$HANDLE" --data-urlencode "id_token=$TEST_ID_TOKEN")
# The handle exists (storage works), so the refusal is the sign-in itself:
# the token went to real Firebase verification and failed there.
[[ -n "$HANDLE" && "$R" == *HTTP400* && "$R" != *"code="* ]] && pass "test seam shut when K_SERVICE is set" || fail "test seam shut when K_SERVICE is set" "$R"
stop_server

# --- Client models switched on: DCR (Copilot dynamic, Connected Apps),
#     operator clients (Gemini Enterprise, Copilot static), revocation ---
TEAMS_CB="https://teams.microsoft.com/api/platform/v1.0/oAuthRedirect"
GE_CB="https://vertexaisearch.cloud.google.com/oauth-redirect"
handle_for() { # client_id redirect_uri -> the sign-in handle from authorize
  qparam "$(loc -G "$BASE/oauth/authorize" --data-urlencode response_type=code --data-urlencode "client_id=$1" \
    --data-urlencode "redirect_uri=$2" --data-urlencode "code_challenge=$CHALLENGE" --data-urlencode code_challenge_method=S256 --data-urlencode state=st2)" handle
}
code_from() { qparam "$(loc -X POST "$BASE/oauth/login/complete" --data-urlencode "handle=$1" --data-urlencode "id_token=$TEST_ID_TOKEN")" code; }
start_server OAUTH_CONFIDENTIAL_CLIENTS=1 OAUTH_DCR=1 "OAUTH_DCR_REDIRECT_HOSTS=teams.microsoft.com,vscode.dev" OAUTH_REVOCATION=1
R=$(curl -s "$BASE/.well-known/oauth-authorization-server")
[[ "$R" == *"\"registration_endpoint\":\"$BASE/oauth/register\""* && "$R" == *"\"revocation_endpoint\":\"$BASE/oauth/revoke\""* && "$R" == *client_secret_basic* ]] \
  && pass "switches on: metadata advertises registration, revocation, client secrets" || fail "switches on: metadata" "$R"
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/register" -H 'content-type: application/json' \
  -d '{"redirect_uris":["https://evil.example/cb"],"token_endpoint_auth_method":"none"}')
[[ "$R" == *HTTP400* && "$R" == *invalid_redirect_uri* ]] && pass "DCR refuses a redirect host off the allowlist" || fail "DCR host allowlist" "$R"
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/register" -H 'content-type: application/json' \
  -d "{\"redirect_uris\":[\"$TEAMS_CB\"],\"token_endpoint_auth_method\":\"none\",\"client_name\":\"Copilot\"}")
DCR_ID=$(sed -n 's/.*"client_id":"\([^"]*\)".*/\1/p' <<<"$R")
[[ "$R" == *HTTP201* && -n "$DCR_ID" ]] && pass "DCR registers a Copilot-style client" || fail "DCR registers" "$R"
H=$(handle_for "$DCR_ID" "$TEAMS_CB"); CONSENT=$(curl -s "$BASE/oauth/request?handle=$H"); CODE=$(code_from "$H")
[[ "$CONSENT" == *'"registered":"dcr"'* && "$CONSENT" == *'"redirect_host":"teams.microsoft.com"'* ]] \
  && pass "consent marks the client as self-registered (dcr)" || fail "consent marks dcr" "$CONSENT"
TOK=$(curl -s -X POST "$BASE/oauth/token" --data-urlencode grant_type=authorization_code --data-urlencode "code=$CODE" \
  --data-urlencode "code_verifier=$VERIFIER" --data-urlencode "client_id=$DCR_ID" --data-urlencode "redirect_uri=$TEAMS_CB")
DACCESS=$(sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p' <<<"$TOK")
R=$(mcp "$DACCESS" tools/call "$EVAL")
[[ "$DACCESS" == ax_* && "$R" == *HTTP200* && "$R" == *42* ]] && pass "DCR client signs in and runs a tool" || fail "DCR client flow" "$TOK / $R"
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/revoke" --data-urlencode "token=$DACCESS" --data-urlencode "client_id=$DCR_ID")
R2=$(mcp "$DACCESS" tools/call "$EVAL")
[[ "$R" == *HTTP200* && "$R2" == *HTTP401* ]] && pass "revocation kills the access token" || fail "revocation" "$R / $(tail -c 60 <<<"$R2")"
# Operator client (Gemini Enterprise): minted by the CLI into the same stub store.
OUT=$(cd "$ROOT" && env FIRESTORE_STUB=true OAUTH_CONFIDENTIAL_CLIENTS=1 OPERATOR_REDIRECTS="$GE_CB" \
  "$AILANG" run --caps IO,FS,Env,Net,Clock,Rand,SharedMem --entry createClientCli executor_oauth/oauth.ail 2>&1)
GE_ID=$(sed -n 's/^Client ID: *//p' <<<"$OUT"); GE_SECRET=$(sed -n 's/^Client secret: *//p' <<<"$OUT")
[[ "$GE_ID" == mco_* && -n "$GE_SECRET" ]] && pass "operator command mints an mco_ client" || fail "operator command" "$(tail -c 300 <<<"$OUT")"
H=$(handle_for "$GE_ID" "$GE_CB"); CONSENT=$(curl -s "$BASE/oauth/request?handle=$H"); CODE=$(code_from "$H")
[[ "$CONSENT" == *'"registered":"operator"'* ]] && pass "consent marks the operator client" || fail "consent marks operator" "$CONSENT"
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/token" -u "$GE_ID:wrong-secret" --data-urlencode grant_type=authorization_code \
  --data-urlencode "code=$CODE" --data-urlencode "code_verifier=$VERIFIER" --data-urlencode "redirect_uri=$GE_CB")
[[ "$R" == *HTTP401* ]] && pass "operator client: wrong secret -> 401" || fail "operator client wrong secret" "$R"
CODE=$(code_from "$(handle_for "$GE_ID" "$GE_CB")")
TOK=$(curl -s -X POST "$BASE/oauth/token" -u "$GE_ID:$GE_SECRET" --data-urlencode grant_type=authorization_code \
  --data-urlencode "code=$CODE" --data-urlencode "code_verifier=$VERIFIER" --data-urlencode "redirect_uri=$GE_CB")
GACCESS=$(sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p' <<<"$TOK")
R=$(mcp "$GACCESS" tools/call "$EVAL")
[[ "$GACCESS" == ax_* && "$R" == *HTTP200* ]] && pass "operator client (Basic auth) signs in and runs a tool" || fail "operator client flow" "$TOK / $R"
if [ -n "$GE_SECRET" ] && grep -qF "$GE_SECRET" "$LOG"; then fail "log never holds a client secret" "leaked"; else pass "log never holds a client secret"; fi
stop_server
# DCR fails closed without a redirect-host allowlist (the package reads [] as any host).
start_server OAUTH_DCR=1
R=$(curl -s -w ' HTTP%{http_code}' -X POST "$BASE/oauth/register" -H 'content-type: application/json' -d "{\"redirect_uris\":[\"$TEAMS_CB\"]}")
[[ "$R" == *HTTP404* ]] && pass "OAUTH_DCR=1 without OAUTH_DCR_REDIRECT_HOSTS stays off" || fail "DCR fails closed" "$R"
stop_server

[ $rc -eq 0 ] && echo "ok: executor oauth e2e"
exit $rc
