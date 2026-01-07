#!/bin/bash
# echo_payload.sh - Demo script for coordinator script invoke type
#
# This script demonstrates how the coordinator passes JSON payloads to scripts.
# It shows both the raw JSON and the converted environment variables.
#
# Usage:
#   Configure as a coordinator agent with invoke.type: script
#
# Example config in ~/.ailang/config.yaml:
#   coordinator:
#     agents:
#       - id: echo-demo
#         inbox: echo-demo
#         invoke:
#           type: script
#           command: "./scripts/coordinator/echo_payload.sh"
#           env_from_payload: true
#
# Send a message:
#   ailang messages send echo-demo '{"model": "gpt5", "benchmark": "fizzbuzz"}' \
#     --title "Echo test" --from "user"

set -e

echo "========================================"
echo " AILANG Script Invoke Demo"
echo "========================================"
echo ""
echo "Task Context (auto-injected):"
echo "  AILANG_TASK_ID:     ${AILANG_TASK_ID:-<not set>}"
echo "  AILANG_MESSAGE_ID:  ${AILANG_MESSAGE_ID:-<not set>}"
echo "  AILANG_WORKSPACE:   ${AILANG_WORKSPACE:-<not set>}"
echo ""
echo "Raw JSON Payload:"
echo "----------------------------------------"
if [ -n "$AILANG_PAYLOAD" ]; then
    # Pretty print if jq is available, otherwise just echo
    if command -v jq &> /dev/null; then
        echo "$AILANG_PAYLOAD" | jq . 2>/dev/null || echo "$AILANG_PAYLOAD"
    else
        echo "$AILANG_PAYLOAD"
    fi
else
    echo "  (no payload)"
fi
echo ""
echo "Converted Environment Variables:"
echo "----------------------------------------"
echo "  (JSON keys are converted to UPPER_SNAKE_CASE)"
echo "  (Nested objects flatten: db.host → DB_HOST)"
echo ""

# Show common demo payload variables
show_var() {
    local var="$1"
    local value="${!var}"
    if [ -n "$value" ]; then
        echo "  $var=$value"
    fi
}

# Common test payload variables
show_var MODEL
show_var BENCHMARK
show_var PARALLEL
show_var VERBOSE
show_var COUNT
show_var RATE
show_var GREETING
show_var STEP
show_var MODELS
show_var OPTIONAL

# Nested object variables (flattened with underscore)
show_var CONFIG_PARALLEL
show_var CONFIG_VERBOSE
show_var DB_HOST
show_var DB_PORT
show_var DB_NAME
show_var DB_USER

echo ""
echo "----------------------------------------"
echo ""
echo "Script completed successfully!"
echo ""
echo "ECHO_COMPLETE: true"
echo "TIMESTAMP: $(date -Iseconds)"
