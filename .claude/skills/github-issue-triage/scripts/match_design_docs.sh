#!/usr/bin/env bash
# Match open GitHub issues to design documents
# Usage: match_design_docs.sh [--planned] [--implemented] [--all] [--json]
#
# Matching strategies:
# 1. Exact issue number (#123) in design doc
# 2. Bug ID (M-BUG-XXX) matching
# 3. Keyword matching from issue title

set -eo pipefail  # Note: removed -u for unset variable handling

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

# Defaults
CHECK_PLANNED=true
CHECK_IMPLEMENTED=true
JSON_OUTPUT=false
REPO="sunholo-data/ailang"

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --planned)
            CHECK_PLANNED=true
            CHECK_IMPLEMENTED=false
            shift
            ;;
        --implemented)
            CHECK_PLANNED=false
            CHECK_IMPLEMENTED=true
            shift
            ;;
        --all)
            CHECK_PLANNED=true
            CHECK_IMPLEMENTED=true
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
ISSUES=$(gh issue list --repo "$REPO" --state open --limit 100 --json number,title,labels 2>/dev/null || echo "[]")

if [[ "$ISSUES" == "[]" ]]; then
    echo "No open issues found."
    exit 0
fi

# Prepare output
declare -a RESULTS

echo "Matching Issues to Design Docs" >&2
echo "===============================" >&2
echo "" >&2

# Process each issue
echo "$ISSUES" | jq -c '.[]' | while read -r issue; do
    NUMBER=$(echo "$issue" | jq -r '.number')
    TITLE=$(echo "$issue" | jq -r '.title')
    LABELS=$(echo "$issue" | jq -r '.labels | map(.name) | join(",")')

    MATCH_DOC=""
    MATCH_TYPE=""
    MATCH_STATUS=""

    # Strategy 1: Search for exact issue number reference (#123)
    if $CHECK_PLANNED; then
        FOUND=$(grep -rl "#$NUMBER" design_docs/planned/ 2>/dev/null | head -1 || echo "")
        if [[ -n "$FOUND" ]]; then
            MATCH_DOC="$FOUND"
            MATCH_TYPE="exact_reference"
            MATCH_STATUS="PLANNED"
        fi
    fi

    if [[ -z "$MATCH_DOC" ]] && $CHECK_IMPLEMENTED; then
        FOUND=$(grep -rl "#$NUMBER" design_docs/implemented/ 2>/dev/null | head -1 || echo "")
        if [[ -n "$FOUND" ]]; then
            MATCH_DOC="$FOUND"
            MATCH_TYPE="exact_reference"
            MATCH_STATUS="IMPLEMENTED"
        fi
    fi

    # Strategy 2: Search by M-ID patterns in title
    if [[ -z "$MATCH_DOC" ]]; then
        # Extract M-XXX patterns from title
        M_ID=$(echo "$TITLE" | grep -oE "M-[A-Z0-9-]+" | head -1 || echo "")
        if [[ -n "$M_ID" ]]; then
            M_ID_LOWER=$(echo "$M_ID" | tr '[:upper:]' '[:lower:]' | tr '-' '_')

            if $CHECK_PLANNED; then
                FOUND=$(find design_docs/planned/ -name "*${M_ID_LOWER}*" -o -name "*${M_ID}*" 2>/dev/null | head -1 || echo "")
                if [[ -n "$FOUND" ]]; then
                    MATCH_DOC="$FOUND"
                    MATCH_TYPE="m_id_match"
                    MATCH_STATUS="PLANNED"
                fi
            fi

            if [[ -z "$MATCH_DOC" ]] && $CHECK_IMPLEMENTED; then
                FOUND=$(find design_docs/implemented/ -name "*${M_ID_LOWER}*" -o -name "*${M_ID}*" 2>/dev/null | head -1 || echo "")
                if [[ -n "$FOUND" ]]; then
                    MATCH_DOC="$FOUND"
                    MATCH_TYPE="m_id_match"
                    MATCH_STATUS="IMPLEMENTED"
                fi
            fi
        fi
    fi

    # Strategy 3: Keyword matching (3+ significant words)
    if [[ -z "$MATCH_DOC" ]]; then
        # Extract significant words (4+ chars, not common words)
        KEYWORDS=$(echo "$TITLE" | tr '[:upper:]' '[:lower:]' | grep -oE '\b[a-z]{4,}\b' | \
            grep -vE '^(the|and|for|with|this|that|from|have|been|will|are|was|should|when|what|where|which|into|than|them|then|there|these|those|would|could|about|after|before|between|other|over|under|some|such|only|also|just|like|make|well|even|most|more|very|back|down|much|good|want|come|does|made|find|here|take|give|tell|work|call|first|last|long|time|look|think|know|want|need|must)$' | \
            head -5 || echo "")

        for word in $KEYWORDS; do
            if $CHECK_PLANNED && [[ -z "$MATCH_DOC" ]]; then
                FOUND=$(grep -ril "$word" design_docs/planned/ 2>/dev/null | head -1 || echo "")
                if [[ -n "$FOUND" ]]; then
                    # Verify with a second keyword
                    for word2 in $KEYWORDS; do
                        if [[ "$word" != "$word2" ]] && grep -qi "$word2" "$FOUND" 2>/dev/null; then
                            MATCH_DOC="$FOUND"
                            MATCH_TYPE="keyword_match"
                            MATCH_STATUS="PLANNED"
                            break
                        fi
                    done
                fi
            fi

            if [[ -z "$MATCH_DOC" ]] && $CHECK_IMPLEMENTED; then
                FOUND=$(grep -ril "$word" design_docs/implemented/ 2>/dev/null | head -1 || echo "")
                if [[ -n "$FOUND" ]]; then
                    for word2 in $KEYWORDS; do
                        if [[ "$word" != "$word2" ]] && grep -qi "$word2" "$FOUND" 2>/dev/null; then
                            MATCH_DOC="$FOUND"
                            MATCH_TYPE="keyword_match"
                            MATCH_STATUS="IMPLEMENTED"
                            break
                        fi
                    done
                fi
            fi

            [[ -n "$MATCH_DOC" ]] && break
        done
    fi

    # Output result
    if $JSON_OUTPUT; then
        if [[ -n "$MATCH_DOC" ]]; then
            echo "{\"number\":$NUMBER,\"title\":\"$TITLE\",\"status\":\"$MATCH_STATUS\",\"doc\":\"$MATCH_DOC\",\"match_type\":\"$MATCH_TYPE\"}"
        else
            echo "{\"number\":$NUMBER,\"title\":\"$TITLE\",\"status\":\"NOT_COVERED\",\"doc\":null,\"match_type\":null}"
        fi
    else
        echo "Issue #$NUMBER: $TITLE"
        if [[ -n "$MATCH_DOC" ]]; then
            echo "  Status: $MATCH_STATUS"
            echo "  Design doc: $MATCH_DOC"
            echo "  Match type: $MATCH_TYPE"
        else
            echo "  Status: NOT COVERED"
            echo "  No matching design doc found"
            # Suggest version directory
            LATEST_PLANNED=$(ls -1 design_docs/planned/ 2>/dev/null | sort -V | tail -1 || echo "v0_7_0")
            echo "  Suggestion: Create design doc in design_docs/planned/$LATEST_PLANNED/"
        fi
        echo ""
    fi
done

# Summary (only for non-JSON output)
if ! $JSON_OUTPUT; then
    echo "===============================" >&2
    TOTAL=$(echo "$ISSUES" | jq 'length')
    echo "Total issues analyzed: $TOTAL" >&2
fi
