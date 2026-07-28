#!/usr/bin/env bash
# Delegate a task to the coordinator for autonomous execution
# Usage: delegate_task.sh <type> <title> <description>
#
# Types: bug, feature, docs, research, refactor, test
#
# Examples:
#   delegate_task.sh bug "Parser NPE" "Fix null pointer in parser.go line 42"
#   delegate_task.sh feature "Add verbose flag" "Add --verbose flag to CLI"

set -euo pipefail

if [[ $# -lt 3 ]]; then
    echo "Usage: delegate_task.sh <type> <title> <description>"
    echo ""
    echo "Task types:"
    echo "  bug       - Bug fix (uses Claude Code)"
    echo "  feature   - New feature (uses Claude Code)"
    echo "  refactor  - Code restructuring (uses Claude Code)"
    echo "  test      - Writing/fixing tests (uses Claude Code)"
    echo "  docs      - Documentation (uses Gemini)"
    echo "  research  - Investigation (uses Gemini)"
    echo ""
    echo "Examples:"
    echo "  delegate_task.sh bug \"Parser NPE\" \"Fix null pointer in parser.go line 42\""
    echo "  delegate_task.sh feature \"Add flag\" \"Add --verbose flag to the CLI\""
    exit 1
fi

TYPE="$1"
TITLE="$2"
DESCRIPTION="$3"

# Validate type
case "$TYPE" in
    bug|feature|refactor|test|docs|research)
        ;;
    *)
        echo "Error: Invalid type '$TYPE'"
        echo "Valid types: bug, feature, refactor, test, docs, research"
        exit 1
        ;;
esac

# Check if coordinator is running
if ! ailang coordinator status 2>&1 | grep -q "running"; then
    echo "Warning: Coordinator daemon is not running"
    echo "Start with: make services-start"
    echo ""
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "Delegating task to coordinator..."
echo ""
echo "Type:        $TYPE"
echo "Title:       $TITLE"
echo "Description: $DESCRIPTION"
echo ""

# Send the message to coordinator inbox
ailang messages send coordinator "$DESCRIPTION" \
    --title "$TITLE" \
    --from "claude-code" \
    --type "$TYPE"

echo ""
echo "✓ Task delegated successfully"
echo ""
echo "Monitor with: ailang coordinator list"
