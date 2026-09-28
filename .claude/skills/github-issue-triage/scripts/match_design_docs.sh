#!/usr/bin/env bash
# Match open GitHub issues to design documents on the pushed branch.
# Usage: match_design_docs.sh [--planned] [--implemented] [--all] [--json] [--ref origin/dev]
#
# Matching (read from REF, default origin/dev, never the working tree):
# 1. Exact issue reference (#123) in a design doc
# 2. M-ID (M-FOO-BAR) from the issue title matched to a doc filename
#
# A match means "a doc mentions this issue", not "the doc covers it". Title
# keyword matching was removed: it took the first doc containing any two title
# words and reported almost everything as covered.

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

CHECK_PLANNED=true
CHECK_IMPLEMENTED=true
JSON_OUTPUT=false
REF="origin/dev"
REPO="sunholo-data/ailang"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --planned) CHECK_PLANNED=true; CHECK_IMPLEMENTED=false; shift ;;
        --implemented) CHECK_PLANNED=false; CHECK_IMPLEMENTED=true; shift ;;
        --all) CHECK_PLANNED=true; CHECK_IMPLEMENTED=true; shift ;;
        --json) JSON_OUTPUT=true; shift ;;
        --ref) REF="$2"; shift 2 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

cd "$PROJECT_ROOT"
"$SCRIPT_DIR/check_auth.sh" --quiet || exit 1

if [[ "$REF" == origin/* ]]; then
    git fetch -q origin "${REF#origin/}" || { echo "error: git fetch of $REF failed; refusing to match against a stale ref" >&2; exit 1; }
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

gh issue list --repo "$REPO" --state open --limit 1000 --json number,title > "$TMP/issues.json"

# "<issue> <path>" rows for #N references, and the doc file list, per directory.
for d in planned implemented; do
    git grep -oE '#[0-9]+' "$REF" -- "design_docs/$d" 2>/dev/null \
      | awk -F: '{n=$NF; sub(/^#/, "", n); print n, $2}' | sort -u > "$TMP/ref_$d"
    git ls-tree -r --name-only "$REF" "design_docs/$d" > "$TMP/files_$d"
done

LATEST_PLANNED=$(git ls-tree -d --name-only "$REF" design_docs/planned/ | sed 's#.*/##' | grep -E '^v[0-9]' | sort -V | tail -1)

if ! $JSON_OUTPUT; then
    echo "Matching issues to design docs on $REF" >&2
    echo "" >&2
fi

jq -r '.[] | [.number, .title] | @tsv' "$TMP/issues.json" | while IFS=$'\t' read -r num title; do
    doc=""; status=""; type=""
    for d in planned implemented; do
        [[ -n "$doc" ]] && break
        if [[ $d == planned ]] && ! $CHECK_PLANNED; then continue; fi
        if [[ $d == implemented ]] && ! $CHECK_IMPLEMENTED; then continue; fi
        hit=$(awk -v n="$num" '$1==n{print $2; exit}' "$TMP/ref_$d")
        if [[ -n "$hit" ]]; then doc=$hit; status=$(echo "$d" | tr '[:lower:]' '[:upper:]'); type=exact_reference; continue; fi
        mid=$(echo "$title" | grep -oE 'M-[A-Z0-9-]+' | head -1 | tr '[:upper:]' '[:lower:]' || true)
        if [[ -n "$mid" ]]; then
            hit=$(grep -iE "/${mid}(-sprint-plan)?\.md$" "$TMP/files_$d" | head -1 || true)
            if [[ -n "$hit" ]]; then doc=$hit; status=$(echo "$d" | tr '[:lower:]' '[:upper:]'); type=m_id_match; fi
        fi
    done

    if $JSON_OUTPUT; then
        jq -cn --argjson n "$num" --arg t "$title" --arg s "${status:-NOT_COVERED}" --arg d "$doc" --arg m "$type" \
          '{number:$n, title:$t, status:$s, doc:(if $d=="" then null else $d end), match_type:(if $m=="" then null else $m end)}'
    else
        echo "Issue #$num: $title"
        if [[ -n "$doc" ]]; then
            echo "  Status: $status ($type)"
            echo "  Design doc: $doc"
        else
            echo "  Status: NOT COVERED (no doc on $REF references #$num)"
            [[ -n "$LATEST_PLANNED" ]] && echo "  If a doc is warranted: design_docs/planned/$LATEST_PLANNED/"
        fi
        echo ""
    fi
done

if ! $JSON_OUTPUT; then
    echo "Total issues analyzed: $(jq length "$TMP/issues.json")" >&2
fi
