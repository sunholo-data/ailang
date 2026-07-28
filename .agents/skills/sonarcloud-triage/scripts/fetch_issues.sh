#!/usr/bin/env bash
# List open bugs + vulnerabilities at the given severities (default BLOCKER,CRITICAL).
# Read-only; no token needed.
#
# Usage:
#   fetch_issues.sh                     # BLOCKER,CRITICAL
#   fetch_issues.sh BLOCKER,CRITICAL,MAJOR
#   fetch_issues.sh BLOCKER             # BLOCKER only

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

severities="${1:-BLOCKER,CRITICAL}"

sc_get "/api/issues/search?componentKeys=${SONAR_PROJECT}&severities=${severities}&types=BUG,VULNERABILITY&resolved=false&ps=500" \
  | python3 -c "
import json, sys
d = json.load(sys.stdin)
total = d.get('total', 0)
issues = d.get('issues', [])
print(f'Total open BUG+VULNERABILITY at $severities: {total}')
if total > len(issues):
    print(f'(showing first {len(issues)} — paginate for more)')
print()
print(f'{\"severity\":8} {\"type\":14} {\"file:line\":65} {\"rule\":35} {\"key\"}')
print(f'{\"-\"*8} {\"-\"*14} {\"-\"*65} {\"-\"*35} {\"-\"*20}')
prefix = '${SONAR_PROJECT}:'
for i in issues:
    comp = i['component']
    if comp.startswith(prefix):
        comp = comp[len(prefix):]
    loc = f\"{comp}:{i.get('line','?')}\"
    print(f'{i[\"severity\"]:8} {i[\"type\"]:14} {loc:65} {i[\"rule\"]:35} {i[\"key\"]}')
    print(f'         {i[\"message\"]}')
"
