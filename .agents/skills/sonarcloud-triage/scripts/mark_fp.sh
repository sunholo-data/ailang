#!/usr/bin/env bash
# Mark a single issue (BUG or VULNERABILITY) as False Positive, with an explanatory comment.
#
# Usage: mark_fp.sh ISSUE_KEY "comment explaining why this is a false positive"
#
# Issue keys come from fetch_issues.sh — last column.
# Example:
#   mark_fp.sh AY-abc123... "PKCS1v15 used for signature verification only (RS256 JWT interop), not encryption."

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

if [[ $# -lt 2 ]]; then
    echo "usage: $(basename "$0") ISSUE_KEY \"comment\"" >&2
    exit 64
fi

issue_key="$1"
comment="$2"

require_token

# Order matters: add the comment first so the audit trail carries the reason,
# then perform the transition. do_transition silently no-ops if the issue is
# already resolved, which keeps this idempotent.
echo "Adding comment to ${issue_key}..."
sc_post "/api/issues/add_comment" "issue=$issue_key" "text=$comment" > /dev/null

echo "Transitioning ${issue_key} → False Positive..."
sc_post "/api/issues/do_transition" "issue=$issue_key" "transition=falsepositive" > /dev/null

echo "Done."
