#!/bin/bash
# cloud_setup.sh - Claude Code SessionStart hook for cloud environment setup
#
# This hook automatically sets up the development environment when
# Claude Code starts in a cloud/web environment.
#
# Only runs when CLAUDE_CODE_REMOTE=true (cloud/mobile sessions).
# Local sessions skip this hook entirely.

set -euo pipefail

# Only run in cloud/remote environments
if [ "$CLAUDE_CODE_REMOTE" != "true" ]; then
    exit 0
fi

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_DIR="${HOME}/.ailang/state"
LOG_FILE="${STATE_DIR}/cloud_setup.log"
SETUP_MARKER="${STATE_DIR}/.cloud_setup_done"

# Ensure state directory exists
mkdir -p "$STATE_DIR"

# Log function
log() {
    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" >> "$LOG_FILE"
}

log "=== Cloud Setup Hook Started ==="
log "CLAUDE_CODE_REMOTE=$CLAUDE_CODE_REMOTE"

# Create AILANG config if missing (coordinator needs this)
create_ailang_config() {
    local AILANG_CONFIG="${HOME}/.ailang/config.yaml"

    if [ ! -f "$AILANG_CONFIG" ]; then
        mkdir -p "$(dirname "$AILANG_CONFIG")"
        cat > "$AILANG_CONFIG" << 'YAML'
github:
  expected_user: sunholo-voight-kampff
  default_repo: sunholo-data/ailang
  create_labels:
    - ailang-message
  watch_labels:
    - coordinator:bug
    - coordinator:feature
    - coordinator:docs
  auto_import: true

coordinator:
  default_provider: claude

  agents:
    - id: coordinator
      label: "General Coordinator"
      inbox: coordinator
      workspace: /home/user/ailang
      capabilities: [code, docs, research]

  github_sync:
    enabled: false  # Disabled in cloud - gh auth not available
    interval_secs: 60
    target_inbox: coordinator
YAML
        log "Created default config.yaml (GitHub sync disabled - no auth)"
    fi
}

create_ailang_config

# Check if setup already completed this session
if [ -f "$SETUP_MARKER" ]; then
    MARKER_AGE=$(($(date +%s) - $(stat -c %Y "$SETUP_MARKER" 2>/dev/null || stat -f %m "$SETUP_MARKER" 2>/dev/null || echo 0)))
    # Skip if setup was done in last 6 hours (session typically doesn't last longer)
    if [ "$MARKER_AGE" -lt 21600 ]; then
        log "Setup already completed (${MARKER_AGE}s ago), skipping"
        echo "Cloud environment: Already configured"
        exit 0
    fi
fi

# Quick check if environment is already set up
check_environment() {
    local missing=""

    # Check Go
    if ! command -v go &> /dev/null; then
        missing="$missing Go"
    elif ! go version 2>/dev/null | grep -q "go1.2"; then
        missing="$missing Go(wrong-version)"
    fi

    # Check make
    if ! command -v make &> /dev/null; then
        missing="$missing make"
    fi

    # Check gh
    if ! command -v gh &> /dev/null; then
        missing="$missing gh"
    fi

    # Check AILANG binary
    if [ ! -f "$PROJECT_ROOT/bin/ailang" ]; then
        missing="$missing ailang-binary"
    fi

    echo "$missing"
}

MISSING=$(check_environment)

if [ -z "$MISSING" ]; then
    log "Environment already complete, no setup needed"
    touch "$SETUP_MARKER"
    echo "Cloud environment: Ready (all tools present)"
    echo "Note: gh CLI works for issues/labels but NOT for PRs - use GitHub compare links instead"

    # Export environment variables to CLAUDE_ENV_FILE for session
    if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
        echo "export PATH=/usr/local/go/bin:\$PATH" >> "$CLAUDE_ENV_FILE"
        echo "export GOTOOLCHAIN=local" >> "$CLAUDE_ENV_FILE"
        echo "export GOPROXY=direct" >> "$CLAUDE_ENV_FILE"
        log "Exported Go environment to CLAUDE_ENV_FILE"
    fi

    exit 0
fi

log "Missing tools: $MISSING"
echo "Cloud environment: Setting up (missing:$MISSING)..."

# Run the full setup script
if [ -x "$PROJECT_ROOT/.claude/skills/cloud-setup/scripts/setup.sh" ]; then
    log "Running full setup script..."

    # Run setup, capture output for logging
    if "$PROJECT_ROOT/.claude/skills/cloud-setup/scripts/setup.sh" >> "$LOG_FILE" 2>&1; then
        log "Setup completed successfully"
        touch "$SETUP_MARKER"

        # Export environment variables
        if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
            echo "export PATH=/usr/local/go/bin:\$PATH" >> "$CLAUDE_ENV_FILE"
            echo "export GOTOOLCHAIN=local" >> "$CLAUDE_ENV_FILE"
            echo "export GOPROXY=direct" >> "$CLAUDE_ENV_FILE"
            log "Exported Go environment to CLAUDE_ENV_FILE"
        fi

        echo "Cloud environment: Setup complete"
        echo "Note: gh CLI works for issues/labels but NOT for PRs - use GitHub compare links instead"
    else
        log "Setup failed - see $LOG_FILE for details"
        echo "Cloud environment: Setup FAILED (see ~/.ailang/state/cloud_setup.log)"
        echo "You can retry manually: .claude/skills/cloud-setup/scripts/setup.sh"
    fi
else
    log "Setup script not found or not executable"
    echo "Cloud environment: Setup script missing"
    echo "Missing tools:$MISSING"
    echo "Install manually or use: .claude/skills/cloud-setup/scripts/setup.sh"
fi

log "=== Cloud Setup Hook Completed ==="
exit 0
