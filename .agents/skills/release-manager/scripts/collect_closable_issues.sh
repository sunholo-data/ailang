#!/usr/bin/env bash
# Collect GitHub issues that can be closed with this release
# Usage: collect_closable_issues.sh <version> [--close] [--json]
#
# Uses ailang messages integration to find related issues.
# Scans commits, changelog, and design docs to find issues that should be closed.
# Outputs suggested closures and optionally closes them via `gh issue close`.

set -euo pipefail

VERSION="${1:-}"
CLOSE_ISSUES=false
JSON_OUTPUT=false

# Parse args
shift || true
while [[ $# -gt 0 ]]; do
    case "$1" in
        --close)
            CLOSE_ISSUES=true
            shift
            ;;
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [--close] [--json]"
    echo ""
    echo "Options:"
    echo "  --close   Actually close the issues (default: just list them)"
    echo "  --json    Output in JSON format for release notes"
    echo ""
    echo "Examples:"
    echo "  $0 0.5.9              # List issues to close"
    echo "  $0 0.5.9 --close      # List and close issues"
    echo "  $0 0.5.9 --json       # Output JSON for release notes"
    exit 1
fi

# Normalize version (add v prefix if missing)
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
cd "$PROJECT_ROOT"

# Get previous tag to determine commit range
PREV_TAG=$(git tag --sort=-version:refname | grep -v "$VERSION" | head -1 || echo "")
if [ -z "$PREV_TAG" ]; then
    COMMIT_RANGE="HEAD"
else
    COMMIT_RANGE="${PREV_TAG}..HEAD"
fi

# Temp files for collecting issues (avoid associative arrays for bash 3.x compat)
TMPDIR="${TMPDIR:-/tmp}"
ISSUES_FILE="$TMPDIR/closable_issues_$$"
trap "rm -f $ISSUES_FILE*" EXIT

# ============================================================================
# 0. Sync GitHub issues via ailang messages integration
# ============================================================================
echo "Syncing GitHub issues via ailang messages..." >&2

if command -v ailang &> /dev/null; then
    # Import latest issues from GitHub (silent, just sync)
    ailang messages import-github --labels bug,feature,ailang-message 2>/dev/null || true
    echo "  ✓ Synced issues from GitHub" >&2
else
    echo "  Warning: ailang not found, skipping GitHub sync" >&2
fi

# ============================================================================
# 1. Find issue references in commits
# ============================================================================
echo "Scanning commits since $PREV_TAG for issue references..." >&2

# Match patterns: Fixes #123, Closes #123, Resolves #123, refs #123
git log "$COMMIT_RANGE" --oneline 2>/dev/null | grep -oiE "(fix|close|resolve|refs?)\s*#[0-9]+" | grep -oE "#[0-9]+" | tr -d '#' | sort -u > "$ISSUES_FILE.commits" 2>/dev/null || true

COMMIT_COUNT=$(wc -l < "$ISSUES_FILE.commits" 2>/dev/null | tr -d ' ' || echo "0")
echo "  Found $COMMIT_COUNT issue(s) referenced in commits" >&2

# ============================================================================
# 2. Find issue references in CHANGELOG for this version
# ============================================================================
echo "Scanning CHANGELOG.md for issue references..." >&2

# Extract changelog section for this version and find issue numbers
awk -v ver="$VERSION" '
    /^## \[/ {
        if (found) exit
        if (index($0, "[" ver "]")) {
            found = 1
            next
        }
    }
    found { print }
' CHANGELOG.md 2>/dev/null | grep -oE "#[0-9]+" | tr -d '#' | sort -u > "$ISSUES_FILE.changelog" 2>/dev/null || true

CHANGELOG_COUNT=$(wc -l < "$ISSUES_FILE.changelog" 2>/dev/null | tr -d ' ' || echo "0")
echo "  Found $CHANGELOG_COUNT issue(s) referenced in CHANGELOG" >&2

# ============================================================================
# 3. Find issues referenced in design docs for this version
# ============================================================================
VERSION_DIR=$(echo "$VERSION" | tr '.' '_')
DESIGN_DOC_PATH="design_docs/implemented/${VERSION_DIR}"
PLANNED_PATH="design_docs/planned/${VERSION_DIR}"

echo "Scanning design docs ($DESIGN_DOC_PATH, $PLANNED_PATH) for issue references..." >&2

touch "$ISSUES_FILE.design"
for doc_dir in "$DESIGN_DOC_PATH" "$PLANNED_PATH"; do
    if [ -d "$doc_dir" ]; then
        grep -rhoE "#[0-9]+" "$doc_dir" 2>/dev/null | tr -d '#' >> "$ISSUES_FILE.design" || true
    fi
done
sort -u "$ISSUES_FILE.design" -o "$ISSUES_FILE.design" 2>/dev/null || true

DESIGN_COUNT=$(wc -l < "$ISSUES_FILE.design" 2>/dev/null | tr -d ' ' || echo "0")
echo "  Found $DESIGN_COUNT issue(s) referenced in design docs" >&2

# ============================================================================
# 4. Query ailang messages for GitHub-linked issues
# ============================================================================
echo "Querying ailang messages for GitHub-linked issues..." >&2

touch "$ISSUES_FILE.messages"
touch "$ISSUES_FILE.messages_detail"

if command -v ailang &> /dev/null; then
    # Get all messages with github_issue field, extract issue numbers
    MESSAGES_JSON=$(ailang messages list --json --limit 100 2>/dev/null || echo "[]")

    # Extract github_issue numbers and their details
    echo "$MESSAGES_JSON" | jq -r '.[] | select(.github_issue != null and .github_issue > 0) | "\(.github_issue)"' 2>/dev/null | sort -u > "$ISSUES_FILE.messages" || true

    # Also store full message details for later lookup
    echo "$MESSAGES_JSON" | jq -c '.[] | select(.github_issue != null and .github_issue > 0)' 2>/dev/null > "$ISSUES_FILE.messages_detail" || true

    MESSAGES_COUNT=$(wc -l < "$ISSUES_FILE.messages" 2>/dev/null | tr -d ' ' || echo "0")
    echo "  Found $MESSAGES_COUNT issue(s) tracked in ailang messages" >&2
else
    echo "  Warning: ailang not found, skipping message query" >&2
fi

# ============================================================================
# 5. Match open issues against CHANGELOG keywords
# ============================================================================
echo "Matching issues against CHANGELOG keywords..." >&2

touch "$ISSUES_FILE.keywords"

# Get changelog section text for keyword matching
CHANGELOG_TEXT=$(awk -v ver="$VERSION" '
    /^## \[/ {
        if (found) exit
        if (index($0, "[" ver "]")) {
            found = 1
            next
        }
    }
    found { print }
' CHANGELOG.md 2>/dev/null || true)

# Match messages against changelog keywords
if [ -s "$ISSUES_FILE.messages_detail" ]; then
    while IFS= read -r msg_json; do
        issue_num=$(echo "$msg_json" | jq -r '.github_issue' 2>/dev/null || echo "")
        title=$(echo "$msg_json" | jq -r '.title // ""' 2>/dev/null || echo "")
        payload=$(echo "$msg_json" | jq -r '.payload // ""' 2>/dev/null || echo "")

        [ -z "$issue_num" ] && continue

        # Check if any significant word from issue title/payload appears in changelog
        for word in $(echo "$title $payload" | tr '[:upper:]' '[:lower:]' | grep -oE '\b[a-z]{4,}\b' | grep -vE '^(the|and|for|with|this|that|from|have|been|will|are|was|bug|fix|issue|error|when|should|expected|context)$' | head -10); do
            if echo "$CHANGELOG_TEXT" | grep -qi "$word" 2>/dev/null; then
                echo "$issue_num" >> "$ISSUES_FILE.keywords"
                break
            fi
        done
    done < "$ISSUES_FILE.messages_detail"

    sort -u "$ISSUES_FILE.keywords" -o "$ISSUES_FILE.keywords" 2>/dev/null || true
fi

KEYWORD_COUNT=$(wc -l < "$ISSUES_FILE.keywords" 2>/dev/null | tr -d ' ' || echo "0")
echo "  Found $KEYWORD_COUNT issue(s) potentially related to CHANGELOG entries" >&2

# ============================================================================
# 6. Combine and deduplicate all found issues
# ============================================================================
cat "$ISSUES_FILE.commits" "$ISSUES_FILE.changelog" "$ISSUES_FILE.design" "$ISSUES_FILE.messages" "$ISSUES_FILE.keywords" 2>/dev/null | sort -u > "$ISSUES_FILE.all"

echo "" >&2
echo "============================================" >&2
echo "Issues to close for $VERSION" >&2
echo "============================================" >&2

if [ ! -s "$ISSUES_FILE.all" ]; then
    echo "No issues found to close." >&2
    if $JSON_OUTPUT; then
        echo "[]"
    fi
    exit 0
fi

# ============================================================================
# 7. Enrich with details and filter to open only
# ============================================================================

# Create output files
> "$ISSUES_FILE.details"
> "$ISSUES_FILE.open_nums"

while read -r issue_num; do
    [ -z "$issue_num" ] && continue

    # Determine sources
    sources=""
    if grep -qx "$issue_num" "$ISSUES_FILE.commits" 2>/dev/null; then
        sources="commit"
    fi
    if grep -qx "$issue_num" "$ISSUES_FILE.changelog" 2>/dev/null; then
        sources="${sources:+$sources,}changelog"
    fi
    if grep -qx "$issue_num" "$ISSUES_FILE.design" 2>/dev/null; then
        sources="${sources:+$sources,}design_doc"
    fi
    if grep -qx "$issue_num" "$ISSUES_FILE.messages" 2>/dev/null; then
        sources="${sources:+$sources,}ailang_message"
    fi
    if grep -qx "$issue_num" "$ISSUES_FILE.keywords" 2>/dev/null; then
        sources="${sources:+$sources,}keyword_match"
    fi

    # First try to get details from ailang messages (local, fast)
    title=""
    labels=""
    state="unknown"

    if [ -s "$ISSUES_FILE.messages_detail" ]; then
        msg_info=$(grep "\"github_issue\":$issue_num[,}]" "$ISSUES_FILE.messages_detail" 2>/dev/null | head -1 || echo "")
        if [ -n "$msg_info" ]; then
            title=$(echo "$msg_info" | jq -r '.title // ""' 2>/dev/null | tr '\n' ' ' | sed 's/  */ /g' | sed 's/ *$//' || echo "")
            # Messages don't have labels, but we can infer from category
            category=$(echo "$msg_info" | jq -r '.category // ""' 2>/dev/null || echo "")
            labels="$category"
        fi
    fi

    # Fall back to GitHub API for state check (needed to verify still open)
    if command -v gh &> /dev/null; then
        issue_info=$(gh issue view "$issue_num" --json number,title,state,labels 2>/dev/null || echo "")
        if [ -n "$issue_info" ]; then
            state=$(echo "$issue_info" | jq -r '.state' 2>/dev/null || echo "unknown")
            # Use GitHub title if we didn't get one from messages
            if [ -z "$title" ]; then
                title=$(echo "$issue_info" | jq -r '.title' 2>/dev/null | tr '\n' ' ' | sed 's/  */ /g' | sed 's/ *$//' || echo "Unknown")
            fi
            labels=$(echo "$issue_info" | jq -r '.labels | map(.name) | join(",")' 2>/dev/null || echo "$labels")
        fi
    fi

    if [ "$state" = "OPEN" ]; then
        echo "$issue_num|$title|$sources|$labels" >> "$ISSUES_FILE.details"
        echo "$issue_num" >> "$ISSUES_FILE.open_nums"
        if ! $JSON_OUTPUT; then
            echo "  #$issue_num: $title" >&2
            echo "       Sources: $sources" >&2
            echo "       Labels: ${labels:-none}" >&2
            echo "" >&2
        fi
    else
        if ! $JSON_OUTPUT; then
            echo "  #$issue_num: (already $state, skipping)" >&2
        fi
    fi
done < "$ISSUES_FILE.all"

# ============================================================================
# 8. Output results
# ============================================================================

if $JSON_OUTPUT; then
    # Output as JSON array
    echo "["
    first=true
    while IFS='|' read -r num title sources labels; do
        [ -z "$num" ] && continue
        if ! $first; then
            echo ","
        fi
        first=false
        # Escape JSON strings properly
        title_escaped=$(echo "$title" | sed 's/\\/\\\\/g; s/"/\\"/g')
        echo "  {"
        echo "    \"number\": $num,"
        echo "    \"title\": \"$title_escaped\","
        echo "    \"sources\": \"$sources\","
        echo "    \"labels\": \"$labels\","
        echo "    \"url\": \"https://github.com/sunholo-data/ailang/issues/$num\""
        echo -n "  }"
    done < "$ISSUES_FILE.details"
    echo ""
    echo "]"
    exit 0
fi

# ============================================================================
# 9. Close issues if requested
# ============================================================================

REPO="sunholo-data/ailang"
RELEASE_URL="https://github.com/$REPO/releases/tag/$VERSION"
RELEASE_COMMIT=$(git rev-parse "$VERSION" 2>/dev/null || echo "")
SHORT_COMMIT="${RELEASE_COMMIT:0:7}"
COMMIT_URL="https://github.com/$REPO/commit/$RELEASE_COMMIT"

if $CLOSE_ISSUES; then
    echo "" >&2
    echo "Closing issues..." >&2

    CLOSED_COUNT=0
    while IFS='|' read -r num title sources labels; do
        [ -z "$num" ] && continue
        echo "  Closing #$num..." >&2

        # Build a better closing comment with release URL and commit
        COMMENT="Fixed in [$VERSION]($RELEASE_URL)."

        # Add source info if available
        if [ -n "$sources" ]; then
            COMMENT="$COMMENT

Identified via: $sources"
        fi

        # Add commit reference
        if [ -n "$SHORT_COMMIT" ]; then
            COMMENT="$COMMENT

Release commit: [\`$SHORT_COMMIT\`]($COMMIT_URL)"
        fi

        if gh issue close "$num" --comment "$COMMENT" 2>/dev/null; then
            echo "    ✓ Closed #$num" >&2
            CLOSED_COUNT=$((CLOSED_COUNT + 1))

            # Also mark the corresponding ailang message as read
            if command -v ailang &> /dev/null; then
                msg_id=$(ailang messages list --json --limit 100 2>/dev/null | jq -r ".[] | select(.github_issue == $num) | .id" 2>/dev/null | head -1 || echo "")
                if [ -n "$msg_id" ]; then
                    ailang messages ack "$msg_id" 2>/dev/null || true
                fi
            fi
        else
            echo "    ✗ Failed to close #$num" >&2
        fi
    done < "$ISSUES_FILE.details"

    echo "" >&2
    echo "✓ Closed $CLOSED_COUNT issue(s)" >&2
else
    if [ -s "$ISSUES_FILE.open_nums" ]; then
        echo "" >&2
        echo "To close these issues, run:" >&2
        echo "  $0 $VERSION --close" >&2
        echo "" >&2
        echo "For more detailed closing comments, use:" >&2
        echo "  .claude/skills/release-manager/scripts/close_issues_with_references.sh $VERSION <issue_num>" >&2
    fi
fi

# Output issue numbers (one per line) for scripting
cat "$ISSUES_FILE.open_nums" 2>/dev/null || true
