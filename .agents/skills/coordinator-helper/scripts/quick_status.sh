#!/usr/bin/env bash
# Show a quick summary of coordinator task status
# Usage: quick_status.sh

set -euo pipefail

echo "Coordinator Task Summary"
echo "========================"
echo ""

# Check if coordinator is running
if ! ailang coordinator status 2>&1 | grep -q "running"; then
    echo "⚠ Coordinator daemon is not running"
    echo "  Start with: make services-start"
    echo ""
fi

# Get task counts using JSON output
pending=$(ailang coordinator list --status pending,queued --json 2>/dev/null | jq 'length' 2>/dev/null || echo "0")
running=$(ailang coordinator list --status running --json 2>/dev/null | jq 'length' 2>/dev/null || echo "0")
approval=$(ailang coordinator list --status pending_approval --json 2>/dev/null | jq 'length' 2>/dev/null || echo "0")
completed=$(ailang coordinator list --status completed --limit 5 --json 2>/dev/null | jq 'length' 2>/dev/null || echo "0")
failed=$(ailang coordinator list --status failed,rejected --limit 5 --json 2>/dev/null | jq 'length' 2>/dev/null || echo "0")

echo "○ Pending:          $pending"
echo "● Running:          $running"
echo "◐ Awaiting Approval: $approval"
echo "✓ Completed (last 5): $completed"
echo "✗ Failed (last 5):   $failed"
echo ""

# Show tasks awaiting approval if any
if [[ "$approval" -gt 0 ]]; then
    echo "Tasks Awaiting Approval:"
    echo "------------------------"
    ailang coordinator list --status pending_approval --json 2>/dev/null | \
        jq -r '.[] | "  • \(.title) [\(.id[:12])]"' 2>/dev/null || true
    echo ""
    echo "Review with: ailang coordinator list"
fi

# Show running tasks if any
if [[ "$running" -gt 0 ]]; then
    echo "Currently Running:"
    echo "------------------"
    ailang coordinator list --status running --json 2>/dev/null | \
        jq -r '.[] | "  • \(.title) [\(.provider // "unknown")]"' 2>/dev/null || true
    echo ""
fi
