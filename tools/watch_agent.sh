#!/bin/bash
# Watch an agent workspace in real-time
# Usage: ./tools/watch_agent.sh <workspace-path>

WORKSPACE="$1"

if [ -z "$WORKSPACE" ]; then
    echo "Usage: $0 <workspace-path>"
    echo ""
    echo "Example:"
    echo "  $0 /tmp/ailang_eval/cli_args_52074"
    echo ""
    echo "This script monitors an agent workspace in real-time, showing:"
    echo "  - File changes (solution.ail, README.md, etc.)"
    echo "  - Bash commands Claude runs"
    echo "  - Tool usage"
    exit 1
fi

if [ ! -d "$WORKSPACE" ]; then
    echo "Error: Workspace directory not found: $WORKSPACE"
    exit 1
fi

echo "🔍 Watching agent workspace: $WORKSPACE"
echo "Press Ctrl+C to stop"
echo ""

# Watch for file changes
fswatch -r "$WORKSPACE" | while read file; do
    filename=$(basename "$file")
    timestamp=$(date +"%H:%M:%S")

    case "$filename" in
        solution.ail)
            echo "[$timestamp] 📝 solution.ail modified"
            echo "--- Current content (last 10 lines) ---"
            tail -10 "$file" 2>/dev/null | sed 's/^/    /'
            echo ""
            ;;
        README.md)
            echo "[$timestamp] 📄 README.md accessed"
            ;;
        syntax_reference.md)
            echo "[$timestamp] 📚 syntax_reference.md accessed"
            ;;
        *.log)
            echo "[$timestamp] 📋 Log file updated: $filename"
            ;;
        *)
            echo "[$timestamp] 📁 File changed: $filename"
            ;;
    esac
done
