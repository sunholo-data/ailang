#!/bin/bash
# Broadcast release notification with changelog changes and closed issues
# Usage: broadcast_release.sh <version> [--include-issues]
#
# Extracts the changelog entry for the given version and broadcasts
# it to all listening projects via the agent messaging system.
# Optionally includes list of closed GitHub issues in the notification.

set -e

VERSION="${1:-}"
INCLUDE_ISSUES=false

# Parse args
shift 2>/dev/null || true
while [[ $# -gt 0 ]]; do
    case "$1" in
        --include-issues)
            INCLUDE_ISSUES=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [--include-issues]"
    echo "Example: $0 0.4.5"
    echo "Example: $0 0.4.5 --include-issues"
    exit 1
fi

# Normalize version (add v prefix if missing)
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

cd "$PROJECT_ROOT"

# Resolve active changelog. Root CHANGELOG.md is an index — themed entries live
# in changelogs/v*.*-current.md (or archived files). Search both, in order.
CHANGELOG_FILE=""
for candidate in changelogs/v*-current.md changelogs/*.md CHANGELOG.md; do
    if [ -f "$candidate" ] && grep -q "^## \[$VERSION\]" "$candidate" 2>/dev/null; then
        CHANGELOG_FILE="$candidate"
        break
    fi
done

if [ -z "$CHANGELOG_FILE" ]; then
    # Fall back to root so downstream "no entry" warning still fires with context
    CHANGELOG_FILE="CHANGELOG.md"
fi

if [ ! -f "$CHANGELOG_FILE" ]; then
    echo "Error: no changelog file found in $PROJECT_ROOT"
    exit 1
fi

echo "Using changelog file: $CHANGELOG_FILE"

echo "Extracting changelog for $VERSION..."

# Extract the changelog section for this version
# Pattern: ## [vX.X.X] - YYYY-MM-DD until next ## [v or end of file
CHANGELOG_CONTENT=$(awk -v ver="$VERSION" '
    /^## \[/ {
        if (found) exit
        if (index($0, "[" ver "]")) {
            found = 1
            print $0
            next
        }
    }
    found { print }
' "$CHANGELOG_FILE")

if [ -z "$CHANGELOG_CONTENT" ]; then
    echo "Warning: No changelog entry found for $VERSION"
    echo "Using generic release message..."
    CHANGELOG_CONTENT="## [$VERSION]

New release available. See CHANGELOG.md for details."
fi

# Strip control characters and problematic Unicode (keep ASCII + basic extended)
# This ensures JSON encoding works reliably across platforms
CHANGELOG_CONTENT=$(echo "$CHANGELOG_CONTENT" | LC_ALL=C tr -cd '[:print:]\n\t' | sed 's/[^[:print:]\t\n]//g')

# Truncate if too long (keep under 2000 chars for message readability)
if [ ${#CHANGELOG_CONTENT} -gt 2000 ]; then
    CHANGELOG_CONTENT="${CHANGELOG_CONTENT:0:1950}

... (truncated - see full changelog at https://github.com/sunholo-data/ailang/blob/main/CHANGELOG.md)"
fi

# Collect closed issues if requested
CLOSED_ISSUES_TEXT=""
CLOSED_ISSUES_JSON="[]"

if $INCLUDE_ISSUES; then
    echo "Collecting closed issues..."

    # Get issues closed in this release (recently closed with the release label or referenced in commits)
    COLLECT_SCRIPT="$SCRIPT_DIR/collect_closable_issues.sh"

    if [ -x "$COLLECT_SCRIPT" ]; then
        # Get JSON output of closable issues
        CLOSED_ISSUES_JSON=$("$COLLECT_SCRIPT" "$VERSION" --json 2>/dev/null || echo "[]")

        # Create human-readable text from JSON
        if command -v jq &> /dev/null && [ "$CLOSED_ISSUES_JSON" != "[]" ]; then
            CLOSED_ISSUES_TEXT=$(echo "$CLOSED_ISSUES_JSON" | jq -r '.[] | "- #\(.number): \(.title)"' 2>/dev/null || echo "")

            if [ -n "$CLOSED_ISSUES_TEXT" ]; then
                CHANGELOG_CONTENT="$CHANGELOG_CONTENT

### Issues Closed
$CLOSED_ISSUES_TEXT"
                echo "  Added $(echo "$CLOSED_ISSUES_JSON" | jq 'length') issue(s) to release notes"
            fi
        fi
    else
        echo "Warning: collect_closable_issues.sh not found, skipping issue collection"
    fi
fi

echo "Broadcasting release notification..."

# Create the release notification message
MESSAGE_TITLE="AILANG $VERSION Released"
RELEASE_URL="https://github.com/sunholo-data/ailang/releases/tag/$VERSION"

# Build JSON payload using jq for proper escaping (handles newlines, quotes, etc.)
if command -v jq &> /dev/null; then
    JSON_PAYLOAD=$(jq -n \
        --arg type "release_notification" \
        --arg title "$MESSAGE_TITLE" \
        --arg version "$VERSION" \
        --arg description "$CHANGELOG_CONTENT" \
        --arg priority "high" \
        --arg release_url "$RELEASE_URL" \
        --argjson closed_issues "$CLOSED_ISSUES_JSON" \
        '{type: $type, title: $title, version: $version, description: $description, priority: $priority, release_url: $release_url, closed_issues: $closed_issues}')
else
    # Fallback: use Python for JSON encoding if jq not available
    if command -v python3 &> /dev/null; then
        JSON_PAYLOAD=$(python3 -c "
import json
import sys
print(json.dumps({
    'type': 'release_notification',
    'title': sys.argv[1],
    'version': sys.argv[2],
    'description': sys.argv[3],
    'priority': 'high',
    'release_url': sys.argv[4],
    'closed_issues': json.loads(sys.argv[5]) if sys.argv[5] else []
}))
" "$MESSAGE_TITLE" "$VERSION" "$CHANGELOG_CONTENT" "$RELEASE_URL" "$CLOSED_ISSUES_JSON")
    else
        echo "Error: Neither jq nor python3 available for JSON encoding"
        echo "Install jq with: brew install jq"
        exit 1
    fi
fi

# Send to user inbox (broadcast point for all projects)
if command -v ailang &> /dev/null; then
    # Use ailang messages send to broadcast to user inbox
    ailang messages send user "$JSON_PAYLOAD" --title "$MESSAGE_TITLE" --from "release-manager"

    echo ""
    echo "Release notification broadcast for $VERSION"
    echo "Projects can check their inbox with: ailang messages list --unread"
else
    echo "Warning: ailang not found in PATH"
    echo "Install with: make install"
    echo ""
    echo "Message that would be sent:"
    echo "$JSON_PAYLOAD"
    exit 1
fi

echo ""
echo "Changelog excerpt:"
echo "---"
echo "$CHANGELOG_CONTENT" | head -20
if [ $(echo "$CHANGELOG_CONTENT" | wc -l) -gt 20 ]; then
    echo "..."
fi
echo "---"
