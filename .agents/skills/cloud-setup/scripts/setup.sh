#!/usr/bin/env bash
# Provision an Anthropic-hosted Claude Code cloud container for AILANG development.
#
# Run it as the environment's Setup script (see SKILL.md), or by hand inside a session.
# Idempotent: every step checks before acting, and a failed step warns instead of aborting,
# so a partial environment still starts.
#
# What the container already has (measured 2026-10-16): Go matching go.mod, make, git, gh,
# node 22, python3, jq, gcloud/bq/gsutil wired to the egress proxy. What it lacks: the ailang
# binary on PATH, z3, and any GCP credential.
#
# Retired from the 2025 version of this script, and why:
#   - DNS rewrite of /etc/resolv.conf: egress goes through the agent proxy; DNS works.
#   - wget --no-check-certificate: the proxy CA is in the system bundle; never disable TLS.
#   - Pinned Go 1.24 + GOTOOLCHAIN=local: go.mod now asks for 1.26.x, and GOTOOLCHAIN=local
#     turns that into a hard build error. Leave the default (auto) so go fetches the toolchain.
#   - GOPROXY=direct: direct fetches hit GitHub's anonymous throttle. The executor images use
#     'https://proxy.golang.org|direct' (docker/Dockerfile.agent-base, 2026-09-29).

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$ROOT" || exit 1

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $1"; }
bad()  { echo -e "${RED}✗${NC} $1"; FAILURES=$((FAILURES + 1)); }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
FAILURES=0

export GOPROXY="${GOPROXY:-https://proxy.golang.org|direct}"
export PATH="/usr/local/go/bin:$(go env GOPATH 2>/dev/null || echo "$HOME/go")/bin:$HOME/.local/bin:$PATH"

echo "== AILANG cloud setup ($ROOT)"

# 1. Go toolchain — go.mod is the only source of the required version.
want="go$(awk '/^go /{print $2; exit}' go.mod)"
if have="$(go env GOVERSION 2>/dev/null)"; then
  [ "$have" = "$want" ] && ok "Go $have" || warn "Go $have; go.mod wants $want (go will auto-fetch it)"
else
  bad "go not found — the base image should ship it; install from https://go.dev/dl/"
fi

# 2. System packages the executor images carry and this container does not.
#    z3: without it `ailang ai-check` silently skips contract verification.
missing=""
command -v z3 >/dev/null 2>&1 || missing="$missing z3"
command -v ssh >/dev/null 2>&1 || missing="$missing openssh-client"
if [ -n "$missing" ]; then
  if command -v apt-get >/dev/null 2>&1 && \
     { apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq $missing; } >/dev/null 2>&1; then
    ok "installed:$missing"
  else
    warn "could not install:$missing (contract checks / git@ URLs degrade)"
  fi
else
  ok "z3 and ssh present"
fi

# 3. GCP credentials from the GCP_SA_KEY_JSON environment secret.
#    Written to the ADC well-known path, so Go client libraries (Firestore message store,
#    observatory) find it with no GOOGLE_APPLICATION_CREDENTIALS export; also activated for
#    gcloud/bq/gsutil. Without it every GCP-backed command fails with CREDENTIALS_MISSING.
ADC="$HOME/.config/gcloud/application_default_credentials.json"
if [ -n "${GCP_SA_KEY_JSON:-}" ]; then
  mkdir -p "$(dirname "$ADC")"
  ( umask 077; printf '%s' "$GCP_SA_KEY_JSON" > "$ADC" )
  if gcloud auth activate-service-account --key-file="$ADC" --quiet >/dev/null 2>&1; then
    gcloud config set project "${AILANG_CLOUD_PROJECT:-ailang-multivac}" --quiet >/dev/null 2>&1
    ok "GCP service account activated ($(gcloud config get-value account 2>/dev/null))"
  else
    bad "GCP_SA_KEY_JSON is set but gcloud rejected it — is it the full key JSON?"
  fi
elif [ -f "$ADC" ]; then
  ok "GCP credentials already at $ADC"
else
  warn "no GCP_SA_KEY_JSON — Firestore inbox, observatory and cloud logs unavailable"
fi

# 4. Build and install ailang. quick-install stamps the version via LDFLAGS (an unstamped
#    binary reports "AILANG dev"), installs to GOPATH/bin and links ~/.local/bin. Neither is
#    on this container's PATH, so also link /usr/local/bin.
if go mod download && make quick-install >/tmp/ailang-cloud-setup-build.log 2>&1; then
  ln -sf "$(go env GOPATH)/bin/ailang" /usr/local/bin/ailang 2>/dev/null || true
  ok "ailang installed: $(ailang --version 2>/dev/null | head -1)"
else
  bad "build failed — see /tmp/ailang-cloud-setup-build.log"
fi

# 5. Smoke test.
if ailang run --caps IO examples/runnable/hello.ail 2>&1 | grep -q "Hello"; then
  ok "ailang runs examples/runnable/hello.ail"
else
  bad "ailang could not run examples/runnable/hello.ail"
fi

echo
if [ "$FAILURES" -eq 0 ]; then
  echo -e "${GREEN}Setup complete.${NC} Run .claude/skills/cloud-setup/scripts/verify.sh inside a session."
else
  echo -e "${RED}Setup finished with $FAILURES failure(s).${NC}"
fi
# Never fail the environment boot: a session with a partial setup is still useful.
exit 0
