#!/usr/bin/env bash
# Check GitHub CLI authentication matches expected user
# Usage: check_auth.sh [--quiet]
#
# Verifies:
# 1. gh CLI is installed
# 2. User is authenticated
# 3. Active account matches github.expected_user in config
#
# Exit codes:
#   0 - Auth OK
#   1 - Auth failed or wrong user

set -euo pipefail

QUIET=false
if [[ "${1:-}" == "--quiet" ]]; then
    QUIET=true
fi

log() {
    if ! $QUIET; then
        echo "$@"
    fi
}

error() {
    echo "ERROR: $@" >&2
}

# Check gh CLI installed
if ! command -v gh &> /dev/null; then
    error "gh CLI not installed"
    error "Install: brew install gh"
    exit 1
fi

# Check authenticated
if ! gh auth status &> /dev/null; then
    error "Not authenticated to GitHub"
    error "Run: gh auth login"
    exit 1
fi

# Get active account
ACTIVE_USER=$(gh auth status 2>&1 | grep -E "Active account: true" -B3 | grep -oE "account [^ ]+" | head -1 | awk '{print $2}' || echo "")

if [[ -z "$ACTIVE_USER" ]]; then
    # Try alternative parsing
    ACTIVE_USER=$(gh auth status 2>&1 | grep -E "Logged in to github.com" | grep -oE "account [^ ]+" | head -1 | awk '{print $2}' || echo "")
fi

if [[ -z "$ACTIVE_USER" ]]; then
    error "Could not determine active GitHub user"
    error "Run: gh auth status"
    exit 1
fi

# Get expected user from config
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
CONFIG_FILE="$HOME/.ailang/config.yaml"

EXPECTED_USER=""
if [[ -f "$CONFIG_FILE" ]]; then
    EXPECTED_USER=$(grep -E "^\s*expected_user:" "$CONFIG_FILE" 2>/dev/null | awk '{print $2}' | tr -d '"' | tr -d "'" || echo "")
fi

if [[ -z "$EXPECTED_USER" ]]; then
    log "Warning: github.expected_user not set in $CONFIG_FILE"
    log "Active user: $ACTIVE_USER"
    log ""
    log "To set expected user, add to $CONFIG_FILE:"
    log "  github:"
    log "    expected_user: $ACTIVE_USER"
    exit 0
fi

# Compare
if [[ "$ACTIVE_USER" != "$EXPECTED_USER" ]]; then
    error "GitHub account mismatch!"
    error "  Active:   $ACTIVE_USER"
    error "  Expected: $EXPECTED_USER"
    error ""
    error "To switch accounts:"
    error "  gh auth switch --user $EXPECTED_USER"
    exit 1
fi

log "GitHub auth OK"
log "  Active user: $ACTIVE_USER"
log "  Config user: $EXPECTED_USER"
exit 0
