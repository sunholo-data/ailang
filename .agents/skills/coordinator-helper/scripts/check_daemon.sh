#!/usr/bin/env bash
# Check if the coordinator daemon is running and show status
# Usage: check_daemon.sh

set -euo pipefail

echo "Checking coordinator daemon status..."
echo ""

# Check if ailang is available
if ! command -v ailang &> /dev/null; then
    echo "Error: ailang command not found"
    echo "Run: make install"
    exit 1
fi

# Get coordinator status
status_output=$(ailang coordinator status 2>&1) || true

if echo "$status_output" | grep -q "running"; then
    echo "✓ Coordinator daemon is RUNNING"
    echo ""
    echo "$status_output"
else
    echo "✗ Coordinator daemon is NOT RUNNING"
    echo ""
    echo "To start:"
    echo "  make services-start    # Start with dashboard server"
    echo "  ailang coordinator start  # Start daemon only"
fi

echo ""

# Check if server is running (for dashboard)
if curl -s http://localhost:1957/health &> /dev/null; then
    echo "✓ Dashboard server is running at http://localhost:1957"
else
    echo "○ Dashboard server is not running"
    echo "  Start with: make services-start"
fi
