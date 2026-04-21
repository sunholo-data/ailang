#!/usr/bin/env bash
# Set the New Code period for the project.
#
# Accepted VALUE forms (SonarCloud legacy setting sonar.leak.period):
#   N                    — rolling last N days (e.g. "30")
#   date:YYYY-MM-DD      — everything since that date
#   previous_version     — compared against previous analyzed version
#   <version-tag>        — a specific analyzed version name (e.g. "v0.12.0")
#
# Usage: set_new_code.sh VALUE
# Examples:
#   set_new_code.sh 30
#   set_new_code.sh previous_version
#   set_new_code.sh date:2026-01-01
#   set_new_code.sh v0.12.0

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

if [[ $# -lt 1 ]]; then
    echo "usage: $(basename "$0") VALUE   # N | date:YYYY-MM-DD | previous_version | <version-tag>" >&2
    exit 64
fi

value="$1"
require_token

echo "Setting sonar.leak.period=${value} on project ${SONAR_PROJECT}..."
sc_post "/api/settings/set" \
    "key=sonar.leak.period" \
    "value=$value" \
    "component=$SONAR_PROJECT" > /dev/null
echo "OK. Re-analysis will recompute New Code metrics on the next scan."
echo
echo "Verify with:"
echo "    $(dirname "${BASH_SOURCE[0]}")/gate_status.sh"
