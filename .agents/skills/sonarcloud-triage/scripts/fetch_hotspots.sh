#!/usr/bin/env bash
# Dump all TO_REVIEW hotspots grouped by rule + top directories. Read-only.
#
# Usage: fetch_hotspots.sh [STATUS]   # default: TO_REVIEW (also accepts REVIEWED)

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

status="${1:-TO_REVIEW}"

sc_get "/api/hotspots/search?projectKey=${SONAR_PROJECT}&status=${status}&ps=500" \
  | python3 -c "
import json, sys
from collections import Counter, defaultdict
d = json.load(sys.stdin)
total = d.get('paging', {}).get('total', 0)
hotspots = d.get('hotspots', [])

print(f'Total hotspots at status=$status: {total}')
if total > len(hotspots):
    print(f'(showing first {len(hotspots)} — paginate for more)')
print()

by_cat = Counter()
by_rule = defaultdict(list)
by_dir = Counter()
prefix = '${SONAR_PROJECT}:'

for h in hotspots:
    comp = h['component']
    if comp.startswith(prefix):
        comp = comp[len(prefix):]
    by_cat[(h['securityCategory'], h['vulnerabilityProbability'])] += 1
    by_rule[(h['securityCategory'], h['ruleKey'])].append((comp, h.get('line', '?'), h['key']))
    d = comp.rsplit('/', 1)[0] if '/' in comp else comp
    by_dir[d] += 1

print('By category / probability:')
for (cat, prob), n in sorted(by_cat.items(), key=lambda x: -x[1]):
    print(f'  {n:4d}  {prob:6} {cat}')
print()

print('By rule (with sample files):')
for (cat, rule), items in sorted(by_rule.items(), key=lambda x: -len(x[1])):
    print(f'  {len(items):4d}  {cat:25} {rule}')
    seen = set()
    for f, ln, _ in items:
        key = f.rsplit('/', 1)[0] if '/' in f else f
        if key not in seen:
            seen.add(key)
            print(f'           └─ {f}:{ln}')
        if len(seen) >= 3:
            break
    if len(items) > 3:
        remaining = len(items) - sum(1 for _,_,_ in items[:3])
print()

print('Top directories:')
for d, n in by_dir.most_common(15):
    print(f'  {n:4d}  {d}')
"
