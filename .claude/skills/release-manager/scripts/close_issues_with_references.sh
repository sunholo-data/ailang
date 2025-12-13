#!/usr/bin/env bash
# Close GitHub issues with proper release references
# Usage: close_issues_with_references.sh <version> <issue_number> [changelog_section]
#
# Creates a closing comment with:
# - Release URL
# - Relevant CHANGELOG excerpt
# - Commit hash (if found)
# - Design doc link (if exists)

set -euo pipefail

VERSION="${1:-}"
ISSUE_NUM="${2:-}"
CHANGELOG_SECTION="${3:-}"

if [ -z "$VERSION" ] || [ -z "$ISSUE_NUM" ]; then
    echo "Usage: $0 <version> <issue_number> [changelog_section]"
    echo ""
    echo "Arguments:"
    echo "  version          Release version (e.g., 0.5.10 or v0.5.10)"
    echo "  issue_number     GitHub issue number to close"
    echo "  changelog_section  Optional: Specific section name from CHANGELOG"
    echo ""
    echo "Examples:"
    echo "  $0 0.5.10 29                    # Auto-detect fix from CHANGELOG"
    echo "  $0 0.5.10 29 'M-STRING-CONVERT' # Use specific section"
    echo ""
    echo "The script will:"
    echo "  1. Find the release URL"
    echo "  2. Extract relevant CHANGELOG entry"
    echo "  3. Find related commit hash"
    echo "  4. Generate a proper closing comment"
    echo "  5. Close the issue with the comment"
    exit 1
fi

# Normalize version
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
cd "$PROJECT_ROOT"

REPO="sunholo-data/ailang"

# ============================================================================
# Get release info
# ============================================================================
echo "Getting release info for $VERSION..."

RELEASE_URL="https://github.com/$REPO/releases/tag/$VERSION"
RELEASE_TAG_COMMIT=$(git rev-parse "$VERSION" 2>/dev/null || echo "")

if [ -z "$RELEASE_TAG_COMMIT" ]; then
    echo "Error: Tag $VERSION not found"
    exit 1
fi

SHORT_COMMIT="${RELEASE_TAG_COMMIT:0:7}"
COMMIT_URL="https://github.com/$REPO/commit/$RELEASE_TAG_COMMIT"

echo "  Release URL: $RELEASE_URL"
echo "  Commit: $SHORT_COMMIT"

# ============================================================================
# Get issue info
# ============================================================================
echo "Getting issue #$ISSUE_NUM info..."

ISSUE_INFO=$(gh issue view "$ISSUE_NUM" --repo "$REPO" --json title,body,state 2>/dev/null || echo "")
if [ -z "$ISSUE_INFO" ]; then
    echo "Error: Issue #$ISSUE_NUM not found"
    exit 1
fi

ISSUE_STATE=$(echo "$ISSUE_INFO" | jq -r '.state')
ISSUE_TITLE=$(echo "$ISSUE_INFO" | jq -r '.title')
ISSUE_BODY=$(echo "$ISSUE_INFO" | jq -r '.body // ""')

echo "  Title: $ISSUE_TITLE"
echo "  State: $ISSUE_STATE"

if [ "$ISSUE_STATE" != "OPEN" ]; then
    echo "Warning: Issue is already $ISSUE_STATE"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
fi

# ============================================================================
# Extract relevant CHANGELOG section
# ============================================================================
echo "Searching CHANGELOG for relevant entry..."

VERSION_UNDERSCORE=$(echo "$VERSION" | tr '.' '_' | sed 's/^v//')

# Get the changelog section for this version
CHANGELOG_FULL=$(awk -v ver="$VERSION" '
    /^## \[/ {
        if (found) exit
        if (index($0, "[" ver "]")) {
            found = 1
            print
            next
        }
    }
    found { print }
' CHANGELOG.md 2>/dev/null || echo "")

# Try to find the specific section if provided
FIX_DESCRIPTION=""
FIX_SECTION=""

if [ -n "$CHANGELOG_SECTION" ]; then
    # Look for the specific section
    FIX_SECTION=$(echo "$CHANGELOG_FULL" | awk -v section="$CHANGELOG_SECTION" '
        /^### / {
            if (found) exit
            if (tolower($0) ~ tolower(section)) {
                found = 1
            }
        }
        found { print }
    ' | head -20)
fi

# If no specific section, try to auto-detect from issue keywords
if [ -z "$FIX_SECTION" ]; then
    # Extract keywords from issue title and body
    KEYWORDS=$(echo "$ISSUE_TITLE $ISSUE_BODY" | tr '[:upper:]' '[:lower:]' | grep -oE '\b[a-z]{4,}\b' | sort -u | head -20)

    # Try to match keywords against changelog
    for keyword in $KEYWORDS; do
        # Skip common words
        case "$keyword" in
            the|and|for|with|this|that|from|have|been|will|are|was|bug|fix|issue|error|when|should|expected|context|code|type|function|variable)
                continue
                ;;
        esac

        MATCH=$(echo "$CHANGELOG_FULL" | grep -i "$keyword" | head -1 || echo "")
        if [ -n "$MATCH" ]; then
            # Find the section containing this match
            FIX_SECTION=$(echo "$CHANGELOG_FULL" | awk -v kw="$keyword" '
                /^### / { section = $0; content = "" }
                { content = content "\n" $0 }
                tolower($0) ~ tolower(kw) { found = 1; exit }
                END { if (found) print section content }
            ' | head -25)
            break
        fi
    done
fi

# Extract just the description (first paragraph after section header)
if [ -n "$FIX_SECTION" ]; then
    FIX_TITLE=$(echo "$FIX_SECTION" | head -1 | sed 's/^### //')
    FIX_DESCRIPTION=$(echo "$FIX_SECTION" | tail -n +2 | sed '/^$/q' | head -5 | tr '\n' ' ' | sed 's/  */ /g; s/^ *//; s/ *$//')
    echo "  Found section: $FIX_TITLE"
else
    echo "  Warning: Could not find specific fix in CHANGELOG"
    FIX_TITLE="See CHANGELOG"
    FIX_DESCRIPTION=""
fi

# ============================================================================
# Check for design doc
# ============================================================================
DESIGN_DOC_LINK=""

# Look for design doc in implemented folder
for doc in design_docs/implemented/v${VERSION_UNDERSCORE}/*.md; do
    if [ -f "$doc" ]; then
        # Check if doc name relates to the fix
        doc_name=$(basename "$doc" .md)
        if echo "$FIX_TITLE $ISSUE_TITLE" | grep -qi "${doc_name//-/ }" 2>/dev/null; then
            DESIGN_DOC_LINK="https://github.com/$REPO/blob/main/$doc"
            echo "  Found design doc: $doc"
            break
        fi
    fi
done

# ============================================================================
# Build closing comment
# ============================================================================
echo ""
echo "Building closing comment..."

COMMENT="Fixed in [$VERSION]($RELEASE_URL)"

if [ -n "$FIX_TITLE" ] && [ "$FIX_TITLE" != "See CHANGELOG" ]; then
    # Extract M-* code if present
    M_CODE=$(echo "$FIX_TITLE" | grep -oE '\(M-[A-Z0-9-]+\)' | head -1 || echo "")
    CLEAN_TITLE=$(echo "$FIX_TITLE" | sed 's/ *(M-[A-Z0-9-]*)//g')

    COMMENT="$COMMENT - $CLEAN_TITLE"
    if [ -n "$M_CODE" ]; then
        COMMENT="$COMMENT $M_CODE"
    fi
fi

COMMENT="$COMMENT."

if [ -n "$FIX_DESCRIPTION" ]; then
    COMMENT="$COMMENT

$FIX_DESCRIPTION"
fi

if [ -n "$DESIGN_DOC_LINK" ]; then
    COMMENT="$COMMENT

See: [Design Doc]($DESIGN_DOC_LINK)"
fi

# Add commit reference
COMMENT="$COMMENT

Release commit: [\`$SHORT_COMMIT\`]($COMMIT_URL)"

echo ""
echo "============================================"
echo "Closing comment preview:"
echo "============================================"
echo "$COMMENT"
echo "============================================"
echo ""

# ============================================================================
# Close the issue
# ============================================================================
read -p "Close issue #$ISSUE_NUM with this comment? (y/N) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Closing issue #$ISSUE_NUM..."
    if gh issue close "$ISSUE_NUM" --repo "$REPO" --comment "$COMMENT"; then
        echo "✓ Issue #$ISSUE_NUM closed successfully"
        echo "  URL: https://github.com/$REPO/issues/$ISSUE_NUM"
    else
        echo "✗ Failed to close issue"
        exit 1
    fi
else
    echo "Cancelled. Issue not closed."
    echo ""
    echo "To close manually:"
    echo "gh issue close $ISSUE_NUM --repo $REPO --comment \"\$COMMENT\""
fi
