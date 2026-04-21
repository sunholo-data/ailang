#!/usr/bin/env bash
# Shared helpers for sonarcloud-triage scripts.
# Source with: source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

set -euo pipefail

SONAR_HOST="${SONAR_HOST:-https://sonarcloud.io}"
SONAR_ORG="${SONAR_ORG:-sunholo-data}"
SONAR_PROJECT="${SONAR_PROJECT:-sunholo-data_ailang}"

require_token() {
    if [[ -z "${SONAR_PAT:-}" ]]; then
        echo "error: SONAR_PAT is not set." >&2
        echo "" >&2
        echo "Generate a user token at https://sonarcloud.io/account/security" >&2
        echo "and export it (zshenv is fine):" >&2
        echo "    export SONAR_PAT=<your-token>" >&2
        exit 64
    fi
}

# sc_get PATH  -> curl GET against SonarCloud API; no auth required for public read
sc_get() {
    local path="$1"
    curl -sS --fail-with-body "${SONAR_HOST}${path}"
}

# sc_post PATH KEY=VAL KEY=VAL... -> authenticated POST with form data
sc_post() {
    require_token
    local path="$1"; shift
    local args=()
    for kv in "$@"; do
        args+=(--data-urlencode "$kv")
    done
    curl -sS --fail-with-body -u "${SONAR_PAT}:" -X POST "${SONAR_HOST}${path}" "${args[@]}"
}
