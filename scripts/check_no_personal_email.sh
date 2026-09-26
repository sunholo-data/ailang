#!/usr/bin/env bash
# check_no_personal_email.sh — refuse a personal email address in the surface the LOOP writes.
#
# WHY (Mark, attended 2026-09-02). Ledger rows are pasted VERBATIM into the public bookkeeping
# issue by every mission report, so an address written into an evidence cell is published — and
# the loop generated exactly that prose from the provenance rule ("record the commit author"),
# reaching 11 occurrences across 9 tracked files before anyone noticed. The rule now says compare,
# never record; this gate is what makes that stick, because the generator is automated and prose
# review is not.
#
# SCOPE IS DELIBERATELY NARROW: the loop-written surface only. It does NOT police
#   * cmd/ailang/*.go — an owner address there is FUNCTIONAL (access control), not incidental
#   * CODE_OF_CONDUCT.md / GOVERNANCE.md / SECURITY.md / .bestpractices.json — deliberate public
#     contact points; whether those change is a governance decision, not a lint
#   * eval_results/** — banked evidence, never rewritten
# Widening this without ruling on those first would just fail the build.
#
# Allowed: GitHub noreply addresses (attributable, expose nothing), example.com/.invalid/.test
# placeholders (RFC 2606/6761 reserved, cannot be real), and machine identities such as GCP
# service accounts — none of which is a person.
# bash 3.2 safe.

set -u
RED=$'\033[0;31m'; GREEN=$'\033[0;32m'; RESET=$'\033[0m'
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT" || exit 1

# The loop writes here.
SCOPE_RE='^(design_docs/[^/]*mission[^/]*\.md|scripts/.*|\.claude/skills/.*)$'
PAT='[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'

hits=0
while IFS= read -r f; do
    case "$f" in
        *.png|*.jpg|*.gif|*.pdf|*.zip) continue ;;
    esac
    [ -f "$f" ] || continue
    found=$(LC_ALL=C grep -oE "$PAT" "$f" 2>/dev/null \
        | grep -vE 'users\.noreply\.github\.com|noreply@|@example\.(com|org|net)|\.(invalid|test|localhost)$|gserviceaccount\.com|@sentry\.io' \
        | sort -u)
    if [ -n "$found" ]; then
        while IFS= read -r addr; do
            [ -n "$addr" ] || continue
            echo "${RED}✗ personal email in loop-written file:${RESET} $f  ->  $addr"
            hits=$((hits+1))
        done <<< "$found"
    fi
done <<< "$(git ls-files | grep -E "$SCOPE_RE")"

if [ "$hits" -gt 0 ]; then
    echo ""
    echo "${RED}check-no-personal-email FAILED${RESET} ($hits occurrence(s))."
    echo "Ledger rows are pasted verbatim into the public bookkeeping issue — an address here is published."
    echo "Record the provenance VERDICT (\"attended\" / \"fleet\"), never the address."
    echo "Use a GitHub noreply address where an identity is genuinely needed."
    exit 1
fi
echo "${GREEN}✓ check-no-personal-email: no personal addresses in the loop-written surface${RESET}"
exit 0
