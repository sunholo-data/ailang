#!/usr/bin/env bash
# Summarize current SonarCloud quality gate status for the project.
# Shows each condition (pass/fail, current value, threshold) and the
# New Code period, plus top-level counts. Read-only; no token needed.

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

echo "Project:  ${SONAR_PROJECT}"
echo "Org:      ${SONAR_ORG}"
echo "Host:     ${SONAR_HOST}"
echo

# --- Assigned gate ---
gate_json="$(sc_get "/api/qualitygates/get_by_project?project=${SONAR_PROJECT}&organization=${SONAR_ORG}")"
echo "=== Quality gate ==="
echo "$gate_json" | python3 -c "
import json, sys
g = json.load(sys.stdin)['qualityGate']
print(f\"  name:    {g['name']}\")
print(f\"  default: {g.get('default', False)}\")
"
echo

# --- New code period ---
echo "=== New code period ==="
sc_get "/api/settings/values?keys=sonar.leak.period,sonar.leak.period.type&component=${SONAR_PROJECT}" \
  | python3 -c "
import json, sys
settings = {s['key']: s['value'] for s in json.load(sys.stdin).get('settings', [])}
val = settings.get('sonar.leak.period', '<inherited: previous_version>')
typ = settings.get('sonar.leak.period.type', '<inherited>')
print(f'  sonar.leak.period       = {val}')
print(f'  sonar.leak.period.type  = {typ}')
if val.isdigit():
    print(f'  interpreted as          last {val} days')
"
echo

# --- Condition statuses ---
echo "=== Gate conditions ==="
sc_get "/api/qualitygates/project_status?projectKey=${SONAR_PROJECT}" \
  | python3 -c "
import json, sys
d = json.load(sys.stdin)['projectStatus']
mark = {'OK': 'PASS', 'ERROR': 'FAIL', 'WARN': 'WARN'}
print(f'  Overall: {mark.get(d[\"status\"], d[\"status\"])}')
print()
print(f'  {\"status\":6} {\"metric\":40} {\"cmp\":4} {\"threshold\":>10} {\"actual\":>10}')
print(f'  {\"-\"*6} {\"-\"*40} {\"-\"*4} {\"-\"*10} {\"-\"*10}')
for c in d.get('conditions', []):
    s = mark.get(c['status'], c['status'])
    print(f'  {s:6} {c[\"metricKey\"]:40} {c[\"comparator\"]:4} {c[\"errorThreshold\"]:>10} {c[\"actualValue\"]:>10}')
"
echo

# --- Top-level counts ---
echo "=== Counts ==="
sc_get "/api/measures/component?component=${SONAR_PROJECT}&metricKeys=ncloc,coverage,bugs,vulnerabilities,code_smells,security_hotspots,reliability_rating,security_rating" \
  | python3 -c "
import json, sys
d = json.load(sys.stdin)['component']
m = {x['metric']: x['value'] for x in d.get('measures', [])}
labels = [
    ('ncloc',              'Lines of code'),
    ('coverage',           'Coverage'),
    ('bugs',               'Bugs'),
    ('vulnerabilities',    'Vulnerabilities'),
    ('security_hotspots',  'Security hotspots'),
    ('code_smells',        'Code smells'),
    ('reliability_rating', 'Reliability'),
    ('security_rating',    'Security'),
]
rating_name = {'1.0': 'A', '2.0': 'B', '3.0': 'C', '4.0': 'D', '5.0': 'E'}
for k, label in labels:
    v = m.get(k, '-')
    if k.endswith('_rating'):
        v = rating_name.get(v, v)
    elif k == 'coverage':
        v = f'{v}%' if v != '-' else '-'
    print(f'  {label:20} {v}')
"
