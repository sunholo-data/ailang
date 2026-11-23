#!/bin/bash
# Sign the agy executor's SUBSCRIPTION route in, once, attended.
#
# The agy executor runs every subscription-route task under an isolated HOME
# ($AILANG_AGY_HOME_ROOT/subscription, default ~/.ailang/state/agy/subscription),
# never the operator's real HOME. This script opens an interactive agy session
# under that same HOME so the Google sign-in lands where the executor will look.
#
# Sign in with the DEDICATED personal Google account that holds the Google AI
# Pro/Ultra plan (Workspace accounts get no plan quota in agy). Its password
# and 2FA live in 1Password; the OAuth token agy caches does NOT go there.
# One sign-in per machine (laptop, rig, cloud) — never copy this HOME's token
# to another holder: a shared refresh token is what killed the Claude loops
# (memory: reference_claude_credentials_three_holders).
#
# Usage: tools/agy/signin.sh            interactive sign-in, then verify
#        tools/agy/signin.sh --verify   verify only (non-interactive, safe in a loop)
set -euo pipefail

ROOT="${AILANG_AGY_HOME_ROOT:-$HOME/.ailang/state/agy}"
SUB_HOME="$ROOT/subscription"
AGY="${AGY:-$(command -v agy || echo "$HOME/.local/bin/agy")}"
HERE="$(cd "$(dirname "$0")" && pwd)"

"$HERE/install.sh" --check

run_agy() {
  env -u GEMINI_API_KEY -u GOOGLE_API_KEY -u AGY_ADC_AUTH -u GOOGLE_CLOUD_PROJECT \
    HOME="$SUB_HOME" AGY_CLI_DISABLE_AUTO_UPDATE=true "$AGY" "$@"
}

mkdir -p "$SUB_HOME/.gemini/antigravity-cli"
settings="$SUB_HOME/.gemini/antigravity-cli/settings.json"
[ -s "$settings" ] || echo '{}' > "$settings"
chmod 600 "$settings"

verify() {
  echo "== agy models (subscription HOME: $SUB_HOME)"
  local out
  out=$(run_agy models </dev/null 2>&1 || true)
  printf '%s\n' "$out" | head -20
  if printf '%s' "$out" | grep -qiE 'sign in|authentication'; then
    echo "NOT SIGNED IN for the subscription route." >&2
    return 1
  fi
  if ! printf '%s' "$out" | grep -q 'gemini-4-neon'; then
    echo "Signed in, but gemini-4-neon is not listed — is this account on a Google AI Pro/Ultra plan?" >&2
    return 1
  fi
  echo "== 1-reply probe on gemini-4-neon-low"
  local probe
  probe=$(cd "$(mktemp -d)" && run_agy --model gemini-4-neon-low --output-format json \
    --print-timeout 90s -p 'reply with exactly: ok' </dev/null 2>&1 || true)
  printf '%s\n' "$probe" | tail -c 600
  if ! printf '%s' "$probe" | grep -q '"status": *"SUCCESS"'; then
    echo "probe did not return status SUCCESS" >&2
    return 1
  fi
  echo "OK: subscription route signed in and serving gemini-4-neon."
}

if [ "${1:-}" = "--verify" ]; then
  verify
  exit $?
fi

if [ -n "${SSH_CONNECTION:-}" ]; then
  cat <<'EOF'
NOTE: this is an SSH session. agy stores the sign-in in the macOS keychain, which
is locked over SSH. Unlock it first, then re-run this script:
    security unlock-keychain ~/Library/Keychains/login.keychain-db
EOF
fi

cat <<EOF
Opening agy interactively under HOME=$SUB_HOME.
  1. Sign in with the dedicated Google AI account (personal @gmail.com, not Workspace).
  2. Once the prompt appears, type /quit (or Ctrl-C) to leave.
This script then verifies the route non-interactively.
EOF
(cd "$(mktemp -d)" && run_agy) || true
verify
