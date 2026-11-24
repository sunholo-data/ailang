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
#        tools/agy/signin.sh --profile cloud --upload <gcp-project> <secret>
#            Sign in a CLOUD-ONLY holder (design_docs/planned/m-agy-cloud-executor.md,
#            D1): a fresh throwaway HOME, never the loops' HOME and never ~/.gemini.
#            The token file goes straight into Secret Manager as a new version of
#            <secret> (never printed), and the throwaway HOME is deleted, so the
#            cloud is the only holder of that sign-in. Run with the gcloud account
#            that may add secret versions (CLOUDSDK_CORE_ACCOUNT=m@sunholo.com).
set -euo pipefail

ROOT="${AILANG_AGY_HOME_ROOT:-$HOME/.ailang/state/agy}"
SUB_HOME="$ROOT/subscription"
PROFILE=subscription
UPLOAD_PROJECT=""
UPLOAD_SECRET=""
VERIFY_ONLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --verify) VERIFY_ONLY=1; shift ;;
    --profile) PROFILE="${2:-}"; shift 2 ;;
    --upload) UPLOAD_PROJECT="${2:-}"; UPLOAD_SECRET="${3:-}"; shift 3 ;;
    -h|--help) sed -n '2,26p' "$0"; exit 0 ;;
    *) echo "signin: unknown argument: $1" >&2; exit 2 ;;
  esac
done
case "$PROFILE" in
  subscription) ;;
  cloud)
    if [ -z "$UPLOAD_PROJECT" ] || [ -z "$UPLOAD_SECRET" ]; then
      echo "signin: --profile cloud needs --upload <gcp-project> <secret> (a cloud sign-in that is not uploaded has no holder)" >&2; exit 2
    fi
    [ "$VERIFY_ONLY" = 0 ] || { echo "signin: --verify does not apply to --profile cloud" >&2; exit 2; }
    command -v gcloud >/dev/null || { echo "signin: gcloud is required for --upload" >&2; exit 2; }
    # Fail before the sign-in, not after it, when the secret cannot take a version.
    gcloud secrets describe "$UPLOAD_SECRET" --project "$UPLOAD_PROJECT" --format='value(name)' >/dev/null \
      || { echo "signin: cannot read secret $UPLOAD_SECRET in $UPLOAD_PROJECT as $(gcloud config get-value account 2>/dev/null)" >&2; exit 2; }
    mkdir -p "$ROOT"
    SUB_HOME="$(mktemp -d "$ROOT/.cloud-signin.XXXXXX")"
    chmod 700 "$SUB_HOME"
    trap 'rm -rf "$SUB_HOME"' EXIT
    ;;
  *) echo "signin: unknown profile '$PROFILE' (subscription | cloud)" >&2; exit 2 ;;
esac
AGY="${AGY:-${AILANG_AGY_HOME_ROOT:-$HOME/.ailang/state/agy}/bin/agy}"
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

if [ "$VERIFY_ONLY" = 1 ]; then
  verify
  exit $?
fi

cat <<EOF
Opening agy interactively under HOME=$SUB_HOME.
  1. Sign in with the dedicated Google AI account (personal @gmail.com, not Workspace).
  2. Once the prompt appears, type /quit (or Ctrl-C) to leave.
This script then verifies the route non-interactively.
EOF
(cd "$(mktemp -d)" && run_agy) || true
verify

if [ "$PROFILE" = cloud ]; then
  token="$SUB_HOME/.gemini/antigravity-cli/antigravity-oauth-token"
  [ -s "$token" ] || { echo "signin: no token file at $token after sign-in; nothing uploaded" >&2; exit 1; }
  # Shape check without printing a value: the cloud job refuses anything else.
  jq -e '.token.refresh_token | type == "string" and length > 0' "$token" >/dev/null \
    || { echo "signin: token file has no refresh_token; nothing uploaded" >&2; exit 1; }
  gcloud secrets versions add "$UPLOAD_SECRET" --project "$UPLOAD_PROJECT" --data-file="$token" --format='value(name)' \
    | sed 's/^/uploaded: /'
  echo "OK: cloud-only agy sign-in stored in $UPLOAD_PROJECT/$UPLOAD_SECRET; the local copy is deleted on exit."
fi
