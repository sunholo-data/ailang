#!/bin/bash
# Analyze compilation timing for a file
# Usage: analyze_compilation.sh <file.ail>

set -e

FILE="${1:-}"

if [ -z "$FILE" ]; then
    echo "Usage: analyze_compilation.sh <file.ail>"
    echo ""
    echo "Example:"
    echo "  analyze_compilation.sh examples/runnable/factorial.ail"
    exit 1
fi

if [ ! -f "$FILE" ]; then
    echo "Error: File not found: $FILE"
    exit 1
fi

# Check telemetry is configured
if [ -z "$GOOGLE_CLOUD_PROJECT" ] && [ -z "$OTEL_EXPORTER_OTLP_ENDPOINT" ]; then
    echo "Warning: No telemetry configured"
    echo "Set GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT for tracing"
    echo ""
fi

echo "=== Compiling with Tracing: $FILE ==="
echo ""

# Run with timing
START=$(date +%s%N)
ailang check "$FILE" 2>&1 || true
END=$(date +%s%N)

DURATION_MS=$(( (END - START) / 1000000 ))
echo ""
echo "Total wall time: ${DURATION_MS}ms"
echo ""

# If telemetry is configured, try to fetch the trace
if [ -n "$GOOGLE_CLOUD_PROJECT" ]; then
    echo "=== Waiting for trace to appear (5s) ==="
    sleep 5

    echo ""
    echo "=== Most Recent Compilation Trace ==="
    # Get most recent compile trace
    TRACE_ID=$(ailang trace list --hours 1 --limit 1 --filter "compile" --json 2>/dev/null | jq -r '.[0].trace_id' 2>/dev/null || echo "")

    if [ -n "$TRACE_ID" ] && [ "$TRACE_ID" != "null" ]; then
        ailang trace view "$TRACE_ID"
    else
        echo "No compilation trace found yet. Try again in 30-60 seconds:"
        echo "  ailang trace list --filter 'compile'"
    fi
fi
