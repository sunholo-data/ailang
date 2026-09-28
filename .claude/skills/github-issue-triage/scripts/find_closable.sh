#!/usr/bin/env bash
# List CLOSE CANDIDATES among open issues. Never closes anything.
# Usage: find_closable.sh [--json] [--ref origin/dev]
#
# A candidate is a lead to verify, not a verdict. Every signal is read from the
# pushed branch (default origin/dev), never the working tree, which in a
# worktree may be an old branch.
#
# Signals, strongest first:
#   fix_commit   a commit on REF says "fixes/closes/resolves #N"
#   impl_doc     a design doc under design_docs/implemented/ on REF names #N
#                in its header (first 20 lines)
#
# Deliberately NOT a signal: "#N appears in changelogs/" or anywhere in a doc
# body. Changelogs and docs cite issues for triage rows, "Refs #N" and
# cross-references.
#
# Measured against the verified 2026-09-28 triage (13 fixed, 18 live):
#   old heuristic (changelog or any implemented-doc mention): flagged 25;
#     7 fixed, 18 live (several P0/P1), and it missed 6 real fixes.
#   these signals: 2 of the 18 live issues still flagged (a false "closes #N",
#     a partial fix); 3 of the 13 fixes caught.
# Most fix commits never name the issue, so ABSENCE OF A CANDIDATE MEANS
# NOTHING. Full recall only comes from reading every issue against the code
# (SKILL.md step 3). That is also why this script cannot close issues.

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

JSON_OUTPUT=false
REF="origin/dev"
REPO="sunholo-data/ailang"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_OUTPUT=true; shift ;;
        --ref) REF="$2"; shift 2 ;;
        --close|--dry-run)
            echo "error: $1 was removed: this script only lists candidates. Verify each one, then close by hand with evidence (SKILL.md step 3)." >&2
            exit 2 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

cd "$PROJECT_ROOT"
"$SCRIPT_DIR/check_auth.sh" --quiet || exit 1

if [[ "$REF" == origin/* ]]; then
    git fetch -q origin "${REF#origin/}" || { echo "error: git fetch of $REF failed; refusing to report from a stale ref" >&2; exit 1; }
fi
git rev-parse -q --verify "$REF^{commit}" >/dev/null || { echo "error: unknown ref $REF" >&2; exit 1; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

gh issue list --repo "$REPO" --state open --limit 1000 --json number,title,url > "$TMP/issues.json"

# One pass each over history and the implemented-doc tree, as "<issue> <evidence>" rows.
git log "$REF" --format='%h %s%n%b' \
  | awk '/^[0-9a-f]+ /{sha=$1} {line=tolower($0); while (match(line, /(close[sd]?|fix(e[sd])?|resolve[sd]?):? +#[0-9]+/)) { m=substr(line, RSTART, RLENGTH); sub(/.*#/, "", m); print m, sha; line=substr(line, RSTART+RLENGTH) }}' \
  | sort -u > "$TMP/fix_commits"
# Header mentions only (first 20 lines): a doc that closes an issue names it in
# its header. Body mentions are cross-references and flagged live bugs as fixed.
git grep -noE '#[0-9]+' "$REF" -- design_docs/implemented 2>/dev/null \
  | awk -F: '$3<=20 {n=$NF; sub(/^#/, "", n); print n, $2}' | sort -u > "$TMP/impl_docs"

jq -r '.[] | [.number, .title, .url] | @tsv' "$TMP/issues.json" | while IFS=$'\t' read -r num title url; do
    commits=$(awk -v n="$num" '$1==n{print $2}' "$TMP/fix_commits" | paste -sd, -)
    docs=$(awk -v n="$num" '$1==n{print $2}' "$TMP/impl_docs" | head -3 | paste -sd, -)
    [[ -z "$commits$docs" ]] && continue
    if [[ -n "$commits" ]]; then signal=fix_commit; else signal=impl_doc; fi
    if $JSON_OUTPUT; then
        jq -cn --argjson n "$num" --arg t "$title" --arg u "$url" --arg s "$signal" --arg c "$commits" --arg d "$docs" \
          '{number:$n, title:$t, url:$u, signal:$s, fix_commits:($c|split(",")|map(select(.!=""))), impl_docs:($d|split(",")|map(select(.!="")))}'
    else
        echo "CANDIDATE #$num [$signal]: $title"
        [[ -n "$commits" ]] && echo "  fix commits on $REF: $commits"
        [[ -n "$docs" ]] && echo "  mentioned in implemented doc(s): $docs"
        echo "  $url"
        echo ""
    fi
done

if ! $JSON_OUTPUT; then
    echo "These are leads, not verdicts. Verify each against $REF (SKILL.md step 3) before closing." >&2
fi
