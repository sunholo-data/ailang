#!/usr/bin/env bash
# Bulk mark every unresolved BUG/VULNERABILITY matching a rule key as Won't Fix with the given comment.
# Idempotent: already-resolved issues are silently skipped by the API.
#
# Usage: mark_wontfix.sh RULE_KEY "comment explaining why we won't fix this rule here"
#
# Example:
#   mark_wontfix.sh typescript:S1082 "Internal visualization component; a11y parity not a product requirement."

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

if [[ $# -lt 2 ]]; then
    echo "usage: $(basename "$0") RULE_KEY \"comment\"" >&2
    exit 64
fi

rule_key="$1"
comment="$2"

require_token

echo "Fetching open issues for rule ${rule_key}..."
keys_tmp="$(mktemp)"
trap 'rm -f "$keys_tmp"' EXIT

sc_get "/api/issues/search?componentKeys=${SONAR_PROJECT}&rules=${rule_key}&resolved=false&ps=500" \
  | python3 -c "
import json, sys
d = json.load(sys.stdin)
keys = [i['key'] for i in d.get('issues', [])]
total = d.get('total', 0)
sys.stderr.write(f'  {len(keys)} of {total} open issues match rule\n')
if total > len(d.get('issues', [])):
    sys.stderr.write(f'  (only first page scanned — rerun after this batch if more remain)\n')
print('\n'.join(keys))
" > "$keys_tmp"

count="$(wc -l < "$keys_tmp" | tr -d ' ')"
if [[ "$count" == "0" ]]; then
    echo "No open issues for rule ${rule_key}. Nothing to do."
    exit 0
fi

echo "Marking ${count} issue(s) Won't Fix..."
ok=0
fail=0
while IFS= read -r key; do
    [[ -z "$key" ]] && continue
    # Comment first for audit trail, then transition.
    if sc_post "/api/issues/add_comment" "issue=$key" "text=$comment" > /dev/null 2>&1 \
       && sc_post "/api/issues/do_transition" "issue=$key" "transition=wontfix" > /dev/null 2>&1; then
        ok=$((ok + 1))
        printf '.'
    else
        fail=$((fail + 1))
        printf 'x'
    fi
done < "$keys_tmp"
echo
echo "Done: ${ok} marked Won't Fix, ${fail} failed (may already be resolved)."
