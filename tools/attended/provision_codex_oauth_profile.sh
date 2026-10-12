#!/usr/bin/env bash
# Attended OAuth provisioning. This never copies an existing auth.json.
set -euo pipefail
set +x
umask 077
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
fail() { printf 'FAIL %s\n' "$1" >&2; exit 1; }
usage() {
  cat <<'USAGE'
Usage: provision_codex_oauth_profile.sh prepare missions|cloud [--config FILE]
       provision_codex_oauth_profile.sh login missions|cloud [--device-auth]
       provision_codex_oauth_profile.sh check
       provision_codex_oauth_profile.sh publish-cloud PROJECT SECRET

Homes: CODEX_INTERACTIVE_HOME (~/.codex), CODEX_MISSION_HOME (~/.codex-missions),
       CODEX_CLOUD_HOME (~/.codex-cloud-staging).
prepare accepts an explicitly audited config.toml only; no automatic copying.
login requires fresh credentials and a user-completed OAuth authorization.
publish-cloud validates all profiles before adding a Secret Manager version.
No mission launchd configuration or running session is changed by this command.
USAGE
}
action=${1:-}; [[ -n $action ]] || { usage; exit 2; }; shift
case $action in
  check) [[ $# == 0 ]] || fail arguments; exec bash "$SCRIPT_DIR/check_codex_oauth_profiles.sh" ;;
  publish-cloud)
    [[ $# == 2 ]] || fail arguments
    bash "$SCRIPT_DIR/check_codex_oauth_profiles.sh" || exit 1
    command -v gcloud >/dev/null 2>&1 || fail dependency-gcloud
    gcloud secrets versions add "$2" --project "$1" --data-file "${CODEX_CLOUD_HOME:-$HOME/.codex-cloud-staging}/auth.json" --quiet >/dev/null 2>&1 || fail cloud-publish
    printf 'PASS cloud-publish\n'; exit 0 ;;
  prepare|login) ;;
  --help|-h) usage; exit 0 ;;
  *) fail action ;;
esac
[[ $# -ge 1 ]] || fail profile
profile=$1; shift
case $profile in
  missions) target=${CODEX_MISSION_HOME:-$HOME/.codex-missions} ;;
  cloud) target=${CODEX_CLOUD_HOME:-$HOME/.codex-cloud-staging} ;;
  *) fail profile ;;
esac
config=
login_args=()
if [[ $action == login && $# == 1 && $1 == --device-auth ]]; then
  login_args=(--device-auth)
  shift
fi
if [[ $# != 0 ]]; then
  [[ $action == prepare && $# == 2 && $1 == --config ]] || fail arguments
  config=$2
  [[ -f $config && ! -L $config ]] || fail config-file
fi
# Normalize absent targets through their existing parent, refusing aliases before
# mkdir/chmod. This protects interactive permissions as well as its credentials.
[[ ! -L $target ]] || fail "$profile-home"
parent=$(dirname "$target"); leaf=$(basename "$target")
[[ $leaf != . && $leaf != .. && -d $parent ]] || fail "$profile-home-parent"
resolved="$(cd "$parent" && pwd -P)/$leaf"
interactive=${CODEX_INTERACTIVE_HOME:-$HOME/.codex}
[[ -d $interactive && ! -L $interactive ]] || fail interactive-home
interactive=$(cd "$interactive" && pwd -P)
[[ $resolved != "$interactive" && $resolved != "$interactive/"* && $interactive != "$resolved/"* ]] || fail distinct-homes
other=${CODEX_CLOUD_HOME:-$HOME/.codex-cloud-staging}
[[ $profile != cloud ]] || other=${CODEX_MISSION_HOME:-$HOME/.codex-missions}
if [[ -e $other ]]; then
  [[ ! -L $other && -d $other ]] || fail distinct-homes
  other=$(cd "$other" && pwd -P)
else
  other_parent=$(dirname "$other")
  [[ -d $other_parent ]] || fail other-home-parent
  other="$(cd "$other_parent" && pwd -P)/$(basename "$other")"
fi
[[ $resolved != "$other" && $resolved != "$other/"* && $other != "$resolved/"* ]] || fail distinct-homes
if [[ $action == prepare ]]; then
  [[ ! -e $target || -d $target ]] || fail "$profile-home"
  [[ ! -L $target/auth.json ]] || fail "$profile-auth-file"
  [[ ! -L $target/config.toml ]] || fail config-file
  mkdir -p "$target"; chmod 700 "$target"
  if [[ -n $config ]]; then
    [[ ! -e $target/config.toml && ! -L $target/config.toml ]] || fail config-exists
    # This is an operator-audited config, not a credential source. Override
    # the top-level credential-store key while preserving model/MCP sections.
    { printf 'cli_auth_credentials_store = "file"\n'; awk '
      /^\[/ { section=1 }
      !section && /^[[:space:]]*cli_auth_credentials_store[[:space:]]*=/ { next }
      { print }
    ' "$config"; } > "$target/config.toml"
    chmod 600 "$target/config.toml"
  fi
  if [[ ! -e $target/config.toml && ! -L $target/config.toml ]]; then
    printf 'cli_auth_credentials_store = "file"\n' > "$target/config.toml"
    chmod 600 "$target/config.toml"
  fi
  awk '
    /^\[/ { section=1 }
    !section && /^[[:space:]]*cli_auth_credentials_store[[:space:]]*=[[:space:]]*"file"[[:space:]]*(#.*)?$/ { found=1 }
    END { exit !found }
  ' "$target/config.toml" || fail config-file-storage
  printf 'PASS %s-prepare\n' "$profile"; exit 0
fi
[[ -d $target ]] || fail "$profile-prepare-required"
[[ ! -e $target/auth.json && ! -L $target/auth.json ]] || fail "$profile-fresh-login-required"
command -v codex >/dev/null 2>&1 || fail dependency-codex
chmod 700 "$target"
# Explicit file storage overrides config/keychain defaults. Codex owns OAuth.
env -u OPENAI_API_KEY -u CODEX_API_KEY CODEX_HOME="$target" codex -c 'cli_auth_credentials_store="file"' login ${login_args[@]+"${login_args[@]}"} || fail "$profile-login"
[[ -f $target/auth.json && ! -L $target/auth.json ]] || fail "$profile-auth-file"
chmod 600 "$target/auth.json"
printf 'PASS %s-login\n' "$profile"
