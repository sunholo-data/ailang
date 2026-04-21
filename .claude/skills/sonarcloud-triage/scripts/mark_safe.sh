#!/usr/bin/env bash
# Bulk mark every TO_REVIEW hotspot matching a rule key as REVIEWED+SAFE with the given comment.
# Idempotent: hotspots already REVIEWED are skipped (API returns 400 on re-transition, which we ignore).
#
# Usage: mark_safe.sh RULE_KEY "comment explaining why this rule is safe here"
#
# Example:
#   mark_safe.sh go:S2245 "Deterministic seeding for reproducible benchmarks; crypto/rand would break reproducibility."

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

if [[ $# -lt 2 ]]; then
    echo "usage: $(basename "$0") RULE_KEY \"comment\"" >&2
    exit 64
fi

rule_key="$1"
comment="$2"

require_token

# Fetch all TO_REVIEW hotspots matching this rule.
# The hotspots/search endpoint doesn't filter by rule server-side, so we filter client-side.
echo "Fetching TO_REVIEW hotspots for rule ${rule_key}..."
keys_tmp="$(mktemp)"
trap 'rm -f "$keys_tmp"' EXIT

RULE="$rule_key" sc_get "/api/hotspots/search?projectKey=${SONAR_PROJECT}&status=TO_REVIEW&ps=500" \
  | RULE="$rule_key" python3 -c "
import json, sys, os
rule = os.environ['RULE']
d = json.load(sys.stdin)
keys = [h['key'] for h in d.get('hotspots', []) if h.get('ruleKey') == rule]
total = d.get('paging', {}).get('total', 0)
sys.stderr.write(f'  {len(keys)} of {total} TO_REVIEW hotspots match rule {rule}\n')
if total > len(d.get('hotspots', [])):
    sys.stderr.write(f'  (only first page scanned — rerun after this batch if more remain)\n')
print('\n'.join(keys))
" > "$keys_tmp"

count="$(wc -l < "$keys_tmp" | tr -d ' ')"
if [[ "$count" == "0" ]]; then
    echo "No TO_REVIEW hotspots found for rule ${rule_key}. Nothing to do."
    exit 0
fi

echo "Marking ${count} hotspot(s) as REVIEWED+SAFE..."
ok=0
fail=0
while IFS= read -r key; do
    [[ -z "$key" ]] && continue
    if sc_post "/api/hotspots/change_status" \
            "hotspot=$key" \
            "status=REVIEWED" \
            "resolution=SAFE" \
            "comment=$comment" > /dev/null 2>&1; then
        ok=$((ok + 1))
        printf '.'
    else
        fail=$((fail + 1))
        printf 'x'
    fi
done < "$keys_tmp"
echo
echo "Done: ${ok} marked SAFE, ${fail} failed (may already be REVIEWED)."
