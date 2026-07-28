#!/usr/bin/env bash
# List all open GitHub issues with details
# Usage: list_open_issues.sh [--json] [--labels LABELS] [--limit N]
#
# Options:
#   --json          Output as JSON
#   --labels LABELS Filter by labels (comma-separated)
#   --limit N       Limit number of issues (default: 100)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

# Defaults
JSON_OUTPUT=false
LABELS=""
LIMIT=100
REPO="sunholo-data/ailang"

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        --labels)
            LABELS="$2"
            shift 2
            ;;
        --limit)
            LIMIT="$2"
            shift 2
            ;;
        --repo)
            REPO="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Check auth
"$SCRIPT_DIR/check_auth.sh" --quiet || exit 1

# Build gh command
GH_ARGS="--repo $REPO --state open --limit $LIMIT"
if [[ -n "$LABELS" ]]; then
    GH_ARGS="$GH_ARGS --label $LABELS"
fi

if $JSON_OUTPUT; then
    # JSON output
    gh issue list $GH_ARGS --json number,title,labels,createdAt,updatedAt,assignees,author,url
else
    # Human-readable output
    echo "Open Issues for $REPO"
    echo "=================================================="
    echo ""

    # Get issues as JSON and format nicely
    ISSUES=$(gh issue list $GH_ARGS --json number,title,labels,createdAt,updatedAt,assignees 2>/dev/null || echo "[]")

    if [[ "$ISSUES" == "[]" ]]; then
        echo "No open issues found."
        exit 0
    fi

    # Calculate age in days
    NOW=$(date +%s)

    echo "$ISSUES" | jq -r '.[] |
        "Issue #\(.number): \(.title)\n" +
        "  Labels: \(if .labels | length > 0 then (.labels | map(.name) | join(", ")) else "none" end)\n" +
        "  Created: \(.createdAt | split("T")[0])\n" +
        "  Updated: \(.updatedAt | split("T")[0])\n" +
        "  Assignee: \(if .assignees | length > 0 then (.assignees | map(.login) | join(", ")) else "unassigned" end)\n"'

    # Count by label
    echo ""
    echo "Summary"
    echo "-------"
    TOTAL=$(echo "$ISSUES" | jq 'length')
    echo "Total open: $TOTAL"

    # Count bugs
    BUGS=$(echo "$ISSUES" | jq '[.[] | select(.labels[]?.name == "bug")] | length')
    echo "Bugs: $BUGS"

    # Count enhancements
    ENHANCEMENTS=$(echo "$ISSUES" | jq '[.[] | select(.labels[]?.name == "enhancement")] | length')
    echo "Enhancements: $ENHANCEMENTS"

    # Count stale (no update in 30+ days)
    THIRTY_DAYS_AGO=$(date -v-30d +%Y-%m-%d 2>/dev/null || date -d "30 days ago" +%Y-%m-%d 2>/dev/null || echo "")
    if [[ -n "$THIRTY_DAYS_AGO" ]]; then
        STALE=$(echo "$ISSUES" | jq --arg date "$THIRTY_DAYS_AGO" '[.[] | select(.updatedAt < $date)] | length')
        echo "Stale (30+ days): $STALE"
    fi
fi
