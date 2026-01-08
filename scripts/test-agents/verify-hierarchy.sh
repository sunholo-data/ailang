#!/bin/bash
# Verify hierarchical trace tracking works correctly.
# This script tests the full flow:
#   message → coordinator → hierarchy-test script → ailang exec → claude → tools
#
# Prerequisites:
#   - ailang coordinator must be running (make services-start)
#   - ailang serve must be running (make services-start)

set -e

echo "=== Hierarchy Verification Test ==="
echo ""

# Check services are running
if ! ailang coordinator status 2>&1 | grep -q "running"; then
    echo "ERROR: Coordinator not running. Run 'make services-start' first."
    exit 1
fi

if ! curl -s http://localhost:1957/health | grep -q "ok"; then
    echo "ERROR: Server not running. Run 'make services-start' first."
    exit 1
fi

echo "✓ Services running"
echo ""

# Send test message
TIMESTAMP=$(date +%s)
MESSAGE_ID="msg_hierarchy_test_${TIMESTAMP}"
DIRECTIVE="Run 'ls examples/*.ail | head -3' and report results"

echo "Sending test message to hierarchy-test agent..."
SEND_OUTPUT=$(ailang messages send hierarchy-test "{\"DIRECTIVE\":\"${DIRECTIVE}\"}" \
    --title "Hierarchy Test ${TIMESTAMP}" --from "verify-script" 2>&1)

# Extract message ID from output
if echo "$SEND_OUTPUT" | grep -q "Message sent"; then
    MSG_ID=$(echo "$SEND_OUTPUT" | grep -oE 'msg_[a-z0-9_]+')
    echo "✓ Message sent: ${MSG_ID}"
else
    echo "ERROR: Failed to send message"
    echo "$SEND_OUTPUT"
    exit 1
fi

# Wait for task to complete
echo ""
echo "Waiting for task to complete (max 60s)..."
for i in {1..12}; do
    sleep 5
    COMPLETED=$(ailang coordinator status 2>&1 | grep "Completed:" | awk '{print $2}')
    echo "  ... $((i*5))s elapsed, completed tasks: $COMPLETED"

    # Check if task is done by looking for the task ID in recent spans
    TASK_ID="task-$(echo $MSG_ID | sed 's/msg_//')"
    SPAN_COUNT=$(sqlite3 ~/.ailang/state/observatory.db "
        SELECT COUNT(*) FROM spans
        WHERE task_id = '${TASK_ID}'
        AND start_time > datetime('now', '-2 minutes')
    " 2>/dev/null || echo "0")

    if [ "$SPAN_COUNT" -gt 0 ]; then
        echo ""
        echo "✓ Task completed with ${SPAN_COUNT} linked spans"
        break
    fi
done

if [ "$SPAN_COUNT" -eq "0" ]; then
    echo ""
    echo "WARNING: No spans found. Task may still be running or failed."
fi

# Extract task ID from message ID
TASK_ID="task-$(echo $MSG_ID | sed 's/msg_[0-9]*_[0-9]*_//')"
echo ""
echo "Task ID: ${TASK_ID}"

# Query hierarchy
echo ""
echo "=== Hierarchy Structure ==="
curl -s "http://localhost:1957/api/observatory/tasks/${TASK_ID}/hierarchy" 2>&1 | \
    jq -r '.agents[0].traces[] | "Trace \(.trace_id[:8])... (\(.spans | length) spans): \(.root_span.span.name)"' 2>/dev/null || \
    echo "  (No hierarchy data yet)"

# Query spans
echo ""
echo "=== Spans in Task ==="
sqlite3 ~/.ailang/state/observatory.db "
SELECT
  SUBSTR(name, 1, 25) as name,
  SUBSTR(task_id, 1, 15) as task_id,
  CASE WHEN parent_span_id IS NULL OR parent_span_id = '' THEN 'root' ELSE SUBSTR(parent_span_id, 1, 8) END as parent
FROM spans
WHERE task_id LIKE '${TASK_ID}%'
   OR task_id LIKE 'exec-from-script-%'
ORDER BY start_time ASC
LIMIT 15
" 2>/dev/null || echo "  (No spans found)"

# Check timestamp correlation
echo ""
echo "=== Timestamp Correlation Check ==="
# Look for ailang.* spans that should be correlated
AILANG_SPANS=$(sqlite3 ~/.ailang/state/observatory.db "
SELECT name FROM spans
WHERE name LIKE 'ailang.%' OR name LIKE 'compile.%'
AND start_time > datetime('now', '-5 minutes')
LIMIT 5
" 2>/dev/null || echo "")

if [ -n "$AILANG_SPANS" ]; then
    echo "Found ailang/compile spans that can be correlated:"
    echo "$AILANG_SPANS" | while read name; do echo "  - $name"; done
else
    echo "  No ailang.check or compile.* spans in recent activity."
    echo "  (The test directive didn't trigger ailang commands)"
fi

echo ""
echo "=== Test Complete ==="
echo "View full hierarchy at: http://localhost:1957/"
echo "Task details: curl -s 'http://localhost:1957/api/observatory/tasks/${TASK_ID}/hierarchy' | jq"
