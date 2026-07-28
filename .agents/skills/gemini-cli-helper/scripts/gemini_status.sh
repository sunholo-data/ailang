#!/bin/bash
# gemini_status.sh - Check Gemini CLI installation and configuration status
set -e

echo "Gemini CLI Status"
echo "━━━━━━━━━━━━━━━━━"

# Find Node v20+ installation
NODE_PATH=""
GEMINI_PATH=""

# Check common nvm locations
for node_dir in ~/.nvm/versions/node/v22* ~/.nvm/versions/node/v21* ~/.nvm/versions/node/v20*; do
    if [[ -d "$node_dir" ]]; then
        NODE_PATH="$node_dir/bin/node"
        # Check if Gemini CLI is installed here
        if [[ -f "$node_dir/lib/node_modules/@google/gemini-cli/dist/index.js" ]]; then
            GEMINI_PATH="$node_dir/lib/node_modules/@google/gemini-cli/dist/index.js"
            break
        fi
    fi
done

# Check Homebrew Node
if [[ -z "$NODE_PATH" ]]; then
    for node_dir in /opt/homebrew/opt/node@22 /opt/homebrew/opt/node@20; do
        if [[ -d "$node_dir" ]]; then
            NODE_PATH="$node_dir/bin/node"
            break
        fi
    done
fi

# Node version
if [[ -n "$NODE_PATH" && -x "$NODE_PATH" ]]; then
    NODE_VERSION=$("$NODE_PATH" --version 2>/dev/null || echo "unknown")
    echo "Node Version: $NODE_VERSION (required: v20+)"
    echo "Node Path: $NODE_PATH"
else
    echo "Node Version: NOT FOUND (v20+ required)"
    echo "  Install with: nvm install 22"
    exit 1
fi

# Gemini CLI
if [[ -n "$GEMINI_PATH" && -f "$GEMINI_PATH" ]]; then
    echo "Gemini CLI: $GEMINI_PATH"
    GEMINI_VERSION=$("$NODE_PATH" "$GEMINI_PATH" --version 2>&1 | grep -v "Creating GCP" || echo "unknown")
    echo "Version: $GEMINI_VERSION"
else
    # Try to find via which
    GEMINI_BIN=$(which gemini 2>/dev/null || true)
    if [[ -n "$GEMINI_BIN" ]]; then
        # Resolve symlink to find actual path
        REAL_PATH=$(readlink "$GEMINI_BIN" 2>/dev/null || echo "")
        if [[ -n "$REAL_PATH" ]]; then
            GEMINI_DIR=$(dirname "$(dirname "$GEMINI_BIN")")
            GEMINI_PATH="$GEMINI_DIR/lib/node_modules/@google/gemini-cli/dist/index.js"
            if [[ -f "$GEMINI_PATH" ]]; then
                echo "Gemini CLI: $GEMINI_PATH"
            else
                echo "Gemini CLI: NOT FOUND"
                echo "  Install with: npm install -g @google/gemini-cli"
                exit 1
            fi
        fi
    else
        echo "Gemini CLI: NOT FOUND"
        echo "  Install with: npm install -g @google/gemini-cli"
        exit 1
    fi
fi

# GCP Project
GCP_PROJECT="${GOOGLE_CLOUD_PROJECT:-}"
if [[ -n "$GCP_PROJECT" ]]; then
    echo "GCP Project: $GCP_PROJECT"
    echo "Telemetry: Enabled (GCP Cloud Trace)"
else
    echo "GCP Project: NOT SET"
    echo "Telemetry: Disabled"
    echo "  Set with: export GOOGLE_CLOUD_PROJECT=your-project"
fi

# Test connection
echo ""
echo "Testing Gemini CLI..."
if "$NODE_PATH" "$GEMINI_PATH" --version &>/dev/null; then
    echo "✓ Gemini CLI is working"
else
    echo "✗ Gemini CLI test failed"
    exit 1
fi
