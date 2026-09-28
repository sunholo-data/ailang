#!/usr/bin/env bash
# Starting-point triage report: where to look, not what to decide.
# Usage: triage_report.sh [--output FILE] [--stale-days N] [--ref origin/dev]
#
# Sections: summary, close candidates (unverified leads from find_closable.sh),
# issues missing a priority label, needs-decision, doc coverage, stale.
# Nothing here is a verdict. Verify every issue against the code (SKILL.md
# step 3) before closing, relabelling or transferring it.

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

OUTPUT_FILE=""
STALE_DAYS=30
REF="origin/dev"
REPO="sunholo-data/ailang"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --output) OUTPUT_FILE="$2"; shift 2 ;;
        --stale-days) STALE_DAYS="$2"; shift 2 ;;
        --ref) REF="$2"; shift 2 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

cd "$PROJECT_ROOT"
"$SCRIPT_DIR/check_auth.sh" --quiet || exit 1

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

gh issue list --repo "$REPO" --state open --limit 1000 \
  --json number,title,labels,updatedAt > "$TMP/issues.json"
"$SCRIPT_DIR/find_closable.sh" --json --ref "$REF" > "$TMP/candidates.jsonl" 2>/dev/null
"$SCRIPT_DIR/match_design_docs.sh" --json --ref "$REF" > "$TMP/docs.jsonl" 2>/dev/null

if date -v-1d >/dev/null 2>&1; then
    STALE_DATE=$(date -u -v-"${STALE_DAYS}"d +%Y-%m-%dT%H:%M:%SZ)
else
    STALE_DATE=$(date -u -d "$STALE_DAYS days ago" +%Y-%m-%dT%H:%M:%SZ)
fi

total=$(jq length "$TMP/issues.json")
issue_rows() { jq -r "$1" "$TMP/issues.json"; }
no_priority=$(issue_rows '[.[] | select([.labels[].name] | any(startswith("priority:")) | not)] | length')
needs_decision=$(issue_rows '[.[] | select([.labels[].name] | index("needs-decision"))] | length')
stale=$(jq --arg d "$STALE_DATE" '[.[] | select(.updatedAt < $d)] | length' "$TMP/issues.json")
candidates=$(wc -l < "$TMP/candidates.jsonl" | tr -d ' ')
covered=$(jq -s '[.[] | select(.status=="PLANNED")] | length' "$TMP/docs.jsonl")
uncovered=$(jq -s '[.[] | select(.status=="NOT_COVERED")] | length' "$TMP/docs.jsonl")

{
echo "# Issue triage starting point: $REPO"
echo ""
echo "Generated $(date '+%Y-%m-%d %H:%M'), docs and history read from \`$REF\` ($(git rev-parse --short "$REF"))."
echo ""
echo "| | Count |"
echo "|---|---|"
echo "| Open issues | $total |"
echo "| Close candidates (unverified leads) | $candidates |"
echo "| Missing a \`priority:*\` label | $no_priority |"
echo "| \`needs-decision\` | $needs_decision |"
echo "| Referenced by a planned doc | $covered |"
echo "| No design doc references it | $uncovered |"
echo "| No activity in ${STALE_DAYS}+ days | $stale |"
echo ""
echo "## Close candidates (unverified)"
echo ""
echo "Leads only. Most real fixes never name their issue, so an issue missing here may still be fixed."
echo ""
if [[ "$candidates" -gt 0 ]]; then
    jq -r '"- #\(.number) [\(.signal)]: \(.title)\n  - evidence: \((.fix_commits + .impl_docs) | join(", "))"' "$TMP/candidates.jsonl"
else
    echo "_None._"
fi
echo ""
echo "## Missing a priority label"
echo ""
issue_rows '.[] | select([.labels[].name] | any(startswith("priority:")) | not) | "- #\(.number): \(.title)"'
echo ""
echo "## Needs a maintainer decision"
echo ""
issue_rows '.[] | select([.labels[].name] | index("needs-decision")) | "- #\(.number): \(.title)"'
echo ""
echo "## Stale (no activity in ${STALE_DAYS}+ days)"
echo ""
jq -r --arg d "$STALE_DATE" '.[] | select(.updatedAt < $d) | "- #\(.number) (last \(.updatedAt[:10])): \(.title)"' "$TMP/issues.json"
echo ""
echo "Next: verify every open issue against \`$REF\` (SKILL.md step 3); this report only says where to start."
} > "$TMP/report.md"

if [[ -n "$OUTPUT_FILE" ]]; then
    cp "$TMP/report.md" "$OUTPUT_FILE"
    echo "Report written to: $OUTPUT_FILE"
else
    cat "$TMP/report.md"
fi
