#!/bin/bash
# echo_payload.sh - Demo script for coordinator script invoke type
#
# This script echoes back all environment variables that were passed
# from a JSON payload via the coordinator's env_from_payload feature.
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
echo "Task Context:"
echo "  AILANG_TASK_ID:     ${AILANG_TASK_ID:-<not set>}"
echo "  AILANG_MESSAGE_ID:  ${AILANG_MESSAGE_ID:-<not set>}"
echo "  AILANG_WORKSPACE:   ${AILANG_WORKSPACE:-<not set>}"
echo ""
echo "Payload Variables (from JSON):"
echo "----------------------------------------"

# Print all non-standard environment variables that might be from payload
# Filter out common system vars to show just payload-derived vars
env | grep -v "^PATH=" | grep -v "^HOME=" | grep -v "^USER=" | grep -v "^SHELL=" | \
     grep -v "^TERM=" | grep -v "^PWD=" | grep -v "^OLDPWD=" | grep -v "^LANG=" | \
     grep -v "^LC_" | grep -v "^SSH_" | grep -v "^XDG_" | grep -v "^_=" | \
     grep -v "^AILANG_" | grep -v "^SHLVL=" | grep -v "^LOGNAME=" | \
     grep -v "^TMPDIR=" | grep -v "^__" | grep -v "^GOOGLE_" | grep -v "^GH_" | \
     sort || echo "  (no payload variables detected)"

echo ""
echo "----------------------------------------"
echo ""
echo "Script completed successfully!"
echo ""
echo "ECHO_COMPLETE: true"
echo "TIMESTAMP: $(date -Iseconds)"
