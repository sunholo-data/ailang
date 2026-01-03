#!/bin/bash
# Check recent traces with optional filtering
# Usage: check_traces.sh [hours] [filter]

set -e

HOURS="${1:-1}"
FILTER="${2:-}"

echo "=== Telemetry Status ==="
ailang trace status
echo ""

echo "=== Recent Traces (last ${HOURS}h) ==="
if [ -n "$FILTER" ]; then
    echo "Filter: $FILTER"
    ailang trace list --hours "$HOURS" --limit 20 --filter "$FILTER"
else
    ailang trace list --hours "$HOURS" --limit 20
fi
echo ""

echo "=== Trace Summary ==="
# Get JSON output for analysis
TRACES=$(ailang trace list --hours "$HOURS" --limit 50 --json 2>/dev/null || echo "[]")

if [ "$TRACES" = "[]" ] || [ -z "$TRACES" ]; then
    echo "No traces found in the last ${HOURS} hour(s)"
    echo ""
    echo "To generate traces:"
    echo "  1. Set GOOGLE_CLOUD_PROJECT=your-project-id"
    echo "  2. Run an ailang command (run, eval-suite, messages)"
    echo "  3. Wait 30-60 seconds for traces to appear"
else
    # Count by root span name
    echo "Traces by operation:"
    echo "$TRACES" | jq -r '.[].name' 2>/dev/null | sort | uniq -c | sort -rn || echo "  (unable to parse trace names)"
fi
