#!/usr/bin/env bash
# Find issues that have been implemented and can be closed
# Usage: find_closable.sh [--close] [--dry-run] [--json]
#
# Checks:
# 1. Issue referenced in design_docs/implemented/
# 2. Issue mentioned in CHANGELOG.md
# 3. Issue keywords match implemented features

set -eo pipefail  # Note: removed -u for unset variable handling

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

# Defaults
DO_CLOSE=false
DRY_RUN=false
JSON_OUTPUT=false
REPO="sunholo-data/ailang"

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --close)
            DO_CLOSE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --json)
            JSON_OUTPUT=true
            shift
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

cd "$PROJECT_ROOT"

# Check auth
"$SCRIPT_DIR/check_auth.sh" --quiet || exit 1

# Get all open issues
ISSUES=$(gh issue list --repo "$REPO" --state open --limit 100 --json number,title,labels,url 2>/dev/null || echo "[]")

if [[ "$ISSUES" == "[]" ]]; then
    echo "No open issues found."
    exit 0
fi

# Temp file for closable issues
TMPFILE=$(mktemp)
trap "rm -f $TMPFILE" EXIT

echo "Finding Closable Issues" >&2
echo "=======================" >&2
echo "" >&2

# Process each issue
CLOSABLE_COUNT=0

echo "$ISSUES" | jq -c '.[]' | while read -r issue; do
    NUMBER=$(echo "$issue" | jq -r '.number')
    TITLE=$(echo "$issue" | jq -r '.title')
    URL=$(echo "$issue" | jq -r '.url')

    CLOSABLE=false
    REASON=""
    DOC=""

    # Check 1: Referenced in implemented design docs
    FOUND=$(grep -rl "#$NUMBER" design_docs/implemented/ 2>/dev/null | head -1 || echo "")
    if [[ -n "$FOUND" ]]; then
        CLOSABLE=true
        REASON="design_doc_implemented"
        DOC="$FOUND"
    fi

    # Check 2: Referenced in CHANGELOG.md
    if ! $CLOSABLE && grep -qE "#$NUMBER\b" CHANGELOG.md 2>/dev/null; then
        CLOSABLE=true
        REASON="in_changelog"
    fi

    # Check 3: M-ID match in implemented docs
    if ! $CLOSABLE; then
        M_ID=$(echo "$TITLE" | grep -oE "M-[A-Z0-9-]+" | head -1 || echo "")
        if [[ -n "$M_ID" ]]; then
            M_ID_LOWER=$(echo "$M_ID" | tr '[:upper:]' '[:lower:]' | tr '-' '_')
            FOUND=$(find design_docs/implemented/ -name "*${M_ID_LOWER}*" -o -name "*${M_ID}*" 2>/dev/null | head -1 || echo "")
            if [[ -n "$FOUND" ]]; then
                CLOSABLE=true
                REASON="m_id_implemented"
                DOC="$FOUND"
            fi
        fi
    fi

    if $CLOSABLE; then
        CLOSABLE_COUNT=$((CLOSABLE_COUNT + 1))

        if $JSON_OUTPUT; then
            echo "{\"number\":$NUMBER,\"title\":\"$TITLE\",\"reason\":\"$REASON\",\"doc\":\"$DOC\",\"url\":\"$URL\"}"
        else
            echo "Issue #$NUMBER: $TITLE"
            echo "  Reason: $REASON"
            [[ -n "$DOC" ]] && echo "  Doc: $DOC"
            echo "  URL: $URL"
            echo ""
        fi

        # Record for closing
        echo "$NUMBER|$TITLE|$REASON|$DOC" >> "$TMPFILE"
    fi
done

# Summary
if ! $JSON_OUTPUT; then
    echo "=======================" >&2
    CLOSABLE_COUNT=$(wc -l < "$TMPFILE" | tr -d ' ')
    echo "Found $CLOSABLE_COUNT closable issue(s)" >&2
fi

# Close issues if requested
if $DO_CLOSE || $DRY_RUN; then
    echo "" >&2

    while IFS='|' read -r num title reason doc; do
        [[ -z "$num" ]] && continue

        # Build closing comment
        COMMENT="This issue has been addressed."

        case "$reason" in
            design_doc_implemented)
                COMMENT="$COMMENT\n\nImplemented via design doc: \`$doc\`"
                ;;
            in_changelog)
                COMMENT="$COMMENT\n\nReferenced in CHANGELOG.md."
                ;;
            m_id_implemented)
                COMMENT="$COMMENT\n\nImplemented in: \`$doc\`"
                ;;
        esac

        COMMENT="$COMMENT\n\nClosed by github-issue-triage skill."

        if $DRY_RUN; then
            echo "[DRY RUN] Would close #$num: $title" >&2
            echo "  Comment: $(echo -e "$COMMENT")" >&2
        elif $DO_CLOSE; then
            echo "Closing #$num: $title" >&2
            if gh issue close "$num" --repo "$REPO" --comment "$(echo -e "$COMMENT")" 2>/dev/null; then
                echo "  Closed." >&2
            else
                echo "  FAILED to close." >&2
            fi
        fi
    done < "$TMPFILE"
fi

if [[ -s "$TMPFILE" ]] && ! $DO_CLOSE && ! $DRY_RUN && ! $JSON_OUTPUT; then
    echo "" >&2
    echo "To close these issues:" >&2
    echo "  $0 --close" >&2
    echo "" >&2
    echo "To preview close actions:" >&2
    echo "  $0 --dry-run" >&2
fi
