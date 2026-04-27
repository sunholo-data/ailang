#!/usr/bin/env bash
# identify_python_only.sh — Find AILANG/Python gaps across all eval modes.
#
# Usage: ./identify_python_only.sh <eval_dir>
# Example: ./identify_python_only.sh eval_results/baselines/v0.14.2
#
# v0.14.2-aware: breaks the analysis down across the three eval modes
# (0-shot, with-repair, agent) and per-agent-harness, since a "gap" can be:
#   - language gap: AILANG fails everywhere Python passes
#   - prompt gap: 0-shot fails but with-repair fixes it (and repair is rare)
#   - harness gap: model fails only via certain agent harnesses
#
# api_error runs are excluded (infra noise, not language signal).

set -euo pipefail

EVAL_DIR="${1:-}"

if [[ -z "$EVAL_DIR" ]]; then
    echo "Usage: $0 <eval_dir>"
    echo "Example: $0 eval_results/baselines/v0.14.2"
    exit 1
fi

[[ -d "$EVAL_DIR" ]] || { echo "ERROR: $EVAL_DIR not found" >&2; exit 1; }

SUMMARY_FILE="$EVAL_DIR/summary.jsonl"
if [[ ! -f "$SUMMARY_FILE" ]]; then
    echo "Generating summary.jsonl..."
    ailang eval-summary "$EVAL_DIR" > /dev/null
fi

# All analysis driven by Python — handles the 3 modes + per-harness cleanly.
python3 - "$SUMMARY_FILE" << 'PYEOF'
import json, sys
from collections import defaultdict

f = sys.argv[1]
records = []
with open(f) as fh:
    for line in fh:
        line = line.strip()
        if not line: continue
        try: records.append(json.loads(line))
        except: pass

# Filter to AILANG + Python pairs (the headline comparison)
records = [r for r in records if r.get('lang') in ('ailang', 'python')]

# Index by (benchmark, lang, mode) where mode ∈ {standard-0shot, standard-final, agent}.
# - standard-0shot: standard run, first_attempt_ok (no repair)
# - standard-final: standard run, stdout_ok after any repair
# - agent: agent eval run
def mode_of(r):
    return 'agent' if r.get('eval_mode') == 'agent' else 'standard'

# Gather per-(benchmark, lang) stats by mode
b2 = defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: {
    'total': 0, 'pass': 0, 'real_runs': 0, 'real_fails': 0,
    'errors': defaultdict(int)
})))
# also harness sub-bucket (agent only)
b3 = defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: {
    'total': 0, 'pass': 0, 'real_runs': 0, 'real_fails': 0,
})))

for r in records:
    bid, lang = r['id'], r['lang']
    mode = mode_of(r)
    cell = b2[bid][lang][mode]
    cell['total'] += 1
    is_api = r.get('error_category') == 'api_error'
    if not is_api:
        cell['real_runs'] += 1
        if r.get('stdout_ok'):
            cell['pass'] += 1
        else:
            cell['real_fails'] += 1
            cell['errors'][r.get('error_category', 'none')] += 1

    # Track 0-shot success separately for standard runs
    if mode == 'standard':
        zsh = b2[bid][lang]['standard-0shot']
        zsh['total'] += 1
        if not is_api:
            zsh['real_runs'] += 1
            if r.get('first_attempt_ok'):
                zsh['pass'] += 1

    if mode == 'agent':
        h = r.get('executor', 'unknown')
        hcell = b3[bid][lang][h]
        hcell['total'] += 1
        if not is_api:
            hcell['real_runs'] += 1
            if r.get('stdout_ok'):
                hcell['pass'] += 1
            else:
                hcell['real_fails'] += 1

def pct(p, total):
    if total == 0: return None
    return p/total*100

# === Section 1: Language gaps (AILANG fails where Python passes, by mode) ===
print("=== Section 1: AILANG Language Gaps by Mode ===")
print()
print("(Python passes ≥1 run, AILANG passes 0 of N real runs in that mode)")
print()

for mode_label, mode_key in [
    ("0-shot (no repair)",     "standard-0shot"),
    ("With Repair (final)",    "standard"),
    ("Agent (any harness)",    "agent"),
]:
    print(f"  --- Mode: {mode_label} ---")
    gaps = []
    for bid in sorted(b2.keys()):
        ai = b2[bid].get('ailang', {}).get(mode_key, {'pass':0,'real_runs':0,'real_fails':0,'errors':{}})
        py = b2[bid].get('python', {}).get(mode_key, {'pass':0,'real_runs':0})
        if ai['real_runs'] == 0 or py['real_runs'] == 0: continue
        if py['pass'] > 0 and ai['pass'] == 0:
            errs = ", ".join(f"{k}={v}" for k,v in sorted(ai['errors'].items())) if ai['errors'] else "—"
            gaps.append((bid, ai['real_fails'], ai['real_runs'], errs))
    if gaps:
        for bid, f, t, errs in gaps:
            print(f"    🔴 {bid}: {f}/{t} AILANG fails ({errs})")
    else:
        print("    ✓ no gaps in this mode")
    print()

# === Section 2: Per-benchmark mode comparison ===
print("=== Section 2: Per-Benchmark Pass-Rate Matrix (AILANG, adjusted) ===")
print()
print(f"  {'Benchmark':<32}  {'0-shot':<8} {'+Repair':<8} {'Agent':<8}  Repair-Δ")
for bid in sorted(b2.keys()):
    ai = b2[bid]['ailang']
    zs = pct(ai.get('standard-0shot', {}).get('pass', 0), ai.get('standard-0shot', {}).get('real_runs', 0))
    fn = pct(ai.get('standard', {}).get('pass', 0), ai.get('standard', {}).get('real_runs', 0))
    ag = pct(ai.get('agent', {}).get('pass', 0), ai.get('agent', {}).get('real_runs', 0))
    def fmt(p): return f"{p:>5.0f}%" if p is not None else "  -  "
    delta = (fn - zs) if (zs is not None and fn is not None) else None
    delta_str = f"{delta:+.0f}pp" if delta is not None else ""
    print(f"  {bid:<32}  {fmt(zs):<8} {fmt(fn):<8} {fmt(ag):<8}  {delta_str}")
print()

# === Section 3: Per-harness AILANG gaps ===
print("=== Section 3: AILANG Per-Harness Gaps (0-pass agent harnesses) ===")
print()
print("  (real_runs > 0 but no successful AILANG run for that harness — likely")
print("   a harness-language interaction, not a benchmark issue)")
print()
HARNESSES = ['claude','codex','gemini','opencode','pi']
for bid in sorted(b3.keys()):
    rows = []
    for h in HARNESSES:
        cell = b3[bid].get('ailang', {}).get(h, {'pass':0,'real_runs':0})
        if cell['real_runs'] > 0 and cell['pass'] == 0:
            rows.append(h)
    if rows:
        print(f"  - {bid}: AILANG 0-pass harnesses: {', '.join(rows)}")

# === Section 4: Where repair helps the most (prompt-improvement signal) ===
print()
print("=== Section 4: Where Self-Repair Recovers AILANG (Prompt Gap Signal) ===")
print()
print("  Big repair-Δ on AILANG = 0-shot prompt is the bottleneck. Add example to teaching prompt.")
print()
recovery = []
for bid in sorted(b2.keys()):
    ai = b2[bid]['ailang']
    zs_runs = ai.get('standard-0shot', {}).get('real_runs', 0)
    zs_pass = ai.get('standard-0shot', {}).get('pass', 0)
    fn_runs = ai.get('standard', {}).get('real_runs', 0)
    fn_pass = ai.get('standard', {}).get('pass', 0)
    if zs_runs == 0 or fn_runs == 0: continue
    delta = (fn_pass/fn_runs - zs_pass/zs_runs) * 100
    if delta >= 20:
        recovery.append((bid, zs_pass/zs_runs*100, fn_pass/fn_runs*100, delta))
recovery.sort(key=lambda x: -x[3])
for bid, zs, fn, delta in recovery[:10]:
    print(f"  📝 {bid}:  {zs:.0f}% → {fn:.0f}%  (+{delta:.0f}pp from repair)")
if not recovery:
    print("  (none — repair isn't recovering AILANG meaningfully on any benchmark)")

# === Section 5: AILANG-only wins (preserve these) ===
print()
print("=== Section 5: AILANG-Only Wins (preserve as value evidence) ===")
print()
print("  (Python real_runs > 0 with 0 passes; AILANG ≥ 1 pass)")
print()
wins = []
for bid in sorted(b2.keys()):
    ai = b2[bid]['ailang'].get('agent', {'pass':0,'real_runs':0})
    py = b2[bid]['python'].get('agent', {'pass':0,'real_runs':0})
    if ai['real_runs'] == 0 or py['real_runs'] == 0: continue
    if ai['pass'] > 0 and py['pass'] == 0:
        wins.append((bid, ai['pass'], ai['real_runs']))
for bid, p, t in wins:
    print(f"  💚 {bid} (AILANG {p}/{t}, Python 0/{t})")
if not wins:
    print("  (none in agent mode — AILANG and Python passing sets overlap on every benchmark)")

PYEOF
