#!/bin/bash
# Test script for hierarchy tracking
# This script is invoked by the coordinator and calls ailang exec
# to verify the full hierarchy chain works:
#   message → coordinator → script → ailang exec → tools → ailang run/check

set -e

# Environment variables provided by coordinator:
# - AILANG_TASK_ID: Current task ID
# - AILANG_PARENT_TASK_ID: Parent task ID for hierarchy linking
# - AILANG_WORKSPACE: Workspace directory
# - AILANG_MESSAGE_ID: Original message ID
# Plus any env vars from JSON payload (if env_from_payload: true)

echo "=== Hierarchy Test Script ==="
echo "AILANG_TASK_ID: ${AILANG_TASK_ID:-not set}"
echo "AILANG_PARENT_TASK_ID: ${AILANG_PARENT_TASK_ID:-not set}"
echo "AILANG_WORKSPACE: ${AILANG_WORKSPACE:-not set}"
echo "AILANG_MESSAGE_ID: ${AILANG_MESSAGE_ID:-not set}"

# Get directive from payload or use default
DIRECTIVE="${DIRECTIVE:-List files in the current directory using the Bash tool}"
PROVIDER="${PROVIDER:-claude}"
MODEL="${MODEL:-haiku}"

echo ""
echo "Provider: $PROVIDER"
echo "Model: $MODEL"
echo "Directive: $DIRECTIVE"
echo ""

# Generate a unique task ID for the exec call
EXEC_TASK_ID="exec-from-script-$(date +%s)"

echo "Calling ailang exec with:"
echo "  --task-id=$EXEC_TASK_ID"
echo "  --parent-task-id=$AILANG_PARENT_TASK_ID"
echo ""

# Call ailang exec with parent task ID for hierarchy linking
# The parent task ID links this exec to the coordinator task
ailang exec "$PROVIDER" "$DIRECTIVE" \
    --task-id="$EXEC_TASK_ID" \
    --parent-task-id="${AILANG_PARENT_TASK_ID:-root}" \
    --model="$MODEL" \
    --timeout=2m \
    --stream-json 2>&1 | while read -r line; do
        echo "[exec] $line"
    done

EXEC_EXIT_CODE=${PIPESTATUS[0]}

echo ""
echo "=== Hierarchy Test Complete ==="
echo "Exit code: $EXEC_EXIT_CODE"

# Output markers for coordinator
echo ""
echo "HIERARCHY_TEST_RESULT: success"
echo "EXEC_TASK_ID: $EXEC_TASK_ID"
echo "PARENT_TASK_ID: ${AILANG_PARENT_TASK_ID:-root}"

exit $EXEC_EXIT_CODE
