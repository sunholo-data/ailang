#!/bin/bash
# gemini_run.sh - Run Gemini CLI with correct Node version
set -e

usage() {
    echo "Usage: $0 <prompt> [--json] [--timeout <seconds>]"
    echo ""
    echo "Arguments:"
    echo "  prompt          The prompt to send to Gemini"
    echo "  --json          Output in JSON format"
    echo "  --timeout N     Timeout in seconds (default: 120)"
    echo ""
    echo "Examples:"
    echo "  $0 \"Say hello\""
    echo "  $0 \"Explain recursion\" --json"
    exit 1
}

if [[ $# -lt 1 ]]; then
    usage
fi

PROMPT="$1"
shift

JSON_OUTPUT=""
TIMEOUT=120

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json)
            JSON_OUTPUT="--output-format json"
            shift
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Find Node v20+ and Gemini CLI
NODE_PATH=""
GEMINI_PATH=""

# Check nvm locations (prefer v22)
for node_dir in ~/.nvm/versions/node/v22* ~/.nvm/versions/node/v21* ~/.nvm/versions/node/v20*; do
    if [[ -d "$node_dir" ]]; then
        if [[ -f "$node_dir/lib/node_modules/@google/gemini-cli/dist/index.js" ]]; then
            NODE_PATH="$node_dir/bin/node"
            GEMINI_PATH="$node_dir/lib/node_modules/@google/gemini-cli/dist/index.js"
            break
        fi
    fi
done

if [[ -z "$NODE_PATH" || -z "$GEMINI_PATH" ]]; then
    echo "Error: Could not find Node v20+ with Gemini CLI installed"
    echo "Install with:"
    echo "  nvm install 22"
    echo "  npm install -g @google/gemini-cli"
    exit 1
fi

# Run Gemini CLI (timeout not used - macOS doesn't have it by default)
exec "$NODE_PATH" "$GEMINI_PATH" -p "$PROMPT" $JSON_OUTPUT
