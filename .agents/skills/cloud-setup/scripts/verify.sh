#!/usr/bin/env bash
# Verify an Anthropic-hosted Claude Code cloud session is ready for AILANG work.
# Read-only. Exit 0 = ready (warnings allowed), 1 = something setup.sh should have fixed.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$ROOT" || exit 1

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $1"; }
bad()  { echo -e "${RED}✗${NC} $1"; FAILURES=$((FAILURES + 1)); }
warn() { echo -e "${YELLOW}⚠${NC} $1"; WARNINGS=$((WARNINGS + 1)); }
FAILURES=0; WARNINGS=0

echo "== Toolchain"
want="go$(awk '/^go /{print $2; exit}' go.mod)"
have="$(go env GOVERSION 2>/dev/null || echo none)"
[ "$have" = "$want" ] && ok "Go $have" || bad "Go $have, go.mod wants $want"
[ "${GOTOOLCHAIN:-}" = "local" ] && [ "$have" != "$want" ] && \
  bad "GOTOOLCHAIN=local blocks the toolchain go.mod asks for — unset it"
for t in make git gh jq gcloud node; do
  command -v "$t" >/dev/null 2>&1 && ok "$t" || bad "$t missing"
done
command -v z3 >/dev/null 2>&1 && ok "z3" || warn "z3 missing — ailang ai-check skips contract verification"

echo "== ailang"
if command -v ailang >/dev/null 2>&1; then
  v="$(ailang --version 2>/dev/null | head -1)"
  case "$v" in *dev*) warn "ailang reports '$v' — unstamped build; use make quick-install" ;; *) ok "ailang on PATH: $v" ;; esac
  ailang run --caps IO examples/runnable/hello.ail 2>&1 | grep -q Hello && ok "runs hello.ail" || bad "cannot run hello.ail"
else
  bad "ailang not on PATH — run scripts/setup.sh"
fi

echo "== GCP"
# Not `print-access-token`: the harness sets a CLOUDSDK_AUTH_ACCESS_TOKEN placeholder, so that
# succeeds while every API call fails CREDENTIALS_MISSING (measured 2026-10-16).
acct="$(gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null | head -1)"
if [ -n "$acct" ]; then
  ok "gcloud active account: $acct"
else
  warn "gcloud has no usable credential — set GCP_SA_KEY_JSON in the environment, rerun setup.sh"
fi
[ -f "$HOME/.config/gcloud/application_default_credentials.json" ] || [ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ] \
  && ok "application default credentials present" \
  || warn "no ADC — Go clients (Firestore inbox, observatory) cannot authenticate"

echo "== Environment"
[ "${AILANG_STORAGE_MESSAGING:-}" = "gcp" ] && ok "AILANG_STORAGE_MESSAGING=gcp" \
  || warn "AILANG_STORAGE_MESSAGING unset — 'ailang messages' reads an empty local store"
[ "${AILANG_STORAGE:-}" = "gcp" ] && bad "AILANG_STORAGE=gcp moves ALL stores — use AILANG_STORAGE_MESSAGING"
[ -n "${AILANG_MESSAGES_PROJECT:-}" ] && ok "AILANG_MESSAGES_PROJECT=$AILANG_MESSAGES_PROJECT" \
  || warn "AILANG_MESSAGES_PROJECT unset (want ailang-multivac)"
[ "${AILANG_AUTOPUSH:-1}" = "0" ] && ok "AILANG_AUTOPUSH=0 (Stop hook will not push dev)" \
  || warn "Stop hook will rebase+push dev from this ephemeral clone — set AILANG_AUTOPUSH=0"
for k in OPENROUTER_API_KEY AILANG_REGISTRY_API_KEY; do
  [ -n "${!k:-}" ] && ok "$k set" || echo "  · $k not set (only needed for evals / publishing)"
done

echo "== Quick tests"
go test ./internal/lexer/... ./internal/parser/... -count=1 >/tmp/ailang-cloud-verify.log 2>&1 \
  && ok "lexer + parser tests" || bad "tests failed — see /tmp/ailang-cloud-verify.log"

echo
if [ "$FAILURES" -gt 0 ]; then
  echo -e "${RED}$FAILURES failure(s), $WARNINGS warning(s).${NC} Run .claude/skills/cloud-setup/scripts/setup.sh"
  exit 1
fi
echo -e "${GREEN}Ready${NC} ($WARNINGS warning(s))."
