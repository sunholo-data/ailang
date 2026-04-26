#!/usr/bin/env bash
# Dual-mode + per-harness saturation audit with API-error adjustment.
#
# Reports per-(benchmark × harness × language):
#   - Raw success rate (count successes / total runs)
#   - API-error rate (infrastructure failures, NOT model wrong-answers)
#   - Adjusted success rate (count successes / non-api-error runs) — model strength
#
# Why: across v0.14.x baselines, ~45% of agent failures are API errors
# (gemini quota, codex version, opencode infra) which deflate raw pass rates
# without indicating model weakness. Saturation decisions should use adjusted rates.
#
# Demote candidate = ≥95% on:
#   - std-AI, std-Py (standard mode)
#   - claude/codex/gemini/opencode × ailang/python — *adjusted* rate (excluding API errors)
#
# Usage:
#   audit_saturation.sh [baseline_version]
# Default version: latest in eval_results/baselines/

set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
    VERSION=$(ls eval_results/baselines/ | grep -E '^v' | sort -V | tail -1)
    echo "No version specified, using latest: $VERSION"
fi

BASELINE_DIR="eval_results/baselines/${VERSION}"
[[ -d "$BASELINE_DIR/standard" ]] || { echo "ERROR: $BASELINE_DIR/standard not found" >&2; exit 1; }
[[ -d "$BASELINE_DIR/agent" ]]    || { echo "ERROR: $BASELINE_DIR/agent not found" >&2; exit 1; }

python3 - "$BASELINE_DIR" << 'PYEOF'
import json, glob, os, sys
from collections import defaultdict

BASELINE_DIR = sys.argv[1]
HARNESSES = ['claude', 'codex', 'gemini', 'opencode']
THRESHOLD = 95.0

# 1. Standard pass rates (no api_error possible — API call is the run)
std = defaultdict(lambda: defaultdict(list))
for f in glob.glob(f'{BASELINE_DIR}/standard/*.json'):
    try:
        with open(f) as fh: d = json.load(fh)
        bid, lang = d.get('id'), d.get('lang')
        if bid and lang:
            std[bid][lang].append(bool(d.get('stdout_ok', False)))
    except Exception:
        pass

# 2. Agent runs: track success + error_category per (benchmark, harness, language)
agent = defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: {'total': 0, 'success': 0, 'api_error': 0})))
for f in glob.glob(f'{BASELINE_DIR}/agent/*.json'):
    try:
        with open(f) as fh: d = json.load(fh)
        bid, lang = d.get('id'), d.get('lang')
        h = d.get('executor', 'unknown')  # the harness, set directly in eval output
        if not (bid and lang and h in HARNESSES): continue
        cell = agent[bid][h][lang]
        cell['total'] += 1
        if d.get('stdout_ok'): cell['success'] += 1
        if d.get('error_category') == 'api_error': cell['api_error'] += 1
    except Exception:
        pass

# 3. Tier per benchmark
tier_of = {}
for yml in glob.glob('benchmarks/*.yml'):
    bid = os.path.basename(yml)[:-4]
    with open(yml) as fh:
        for line in fh:
            if line.startswith('tier:'):
                tier_of[bid] = line.split(':',1)[1].strip().split()[0]
                break

def pct(num, den):
    if den == 0: return None
    return num / den * 100

def fmt(p):
    if p is None: return "  -  "
    return f"{p:>4.0f}%"

# Per-benchmark report
rows = []
for bid in sorted(set(list(std.keys()) + list(agent.keys()))):
    if tier_of.get(bid) not in ('core', 'stretch'):
        continue
    sa = pct(sum(std[bid].get('ailang', [])), len(std[bid].get('ailang', [])))
    sp = pct(sum(std[bid].get('python', [])), len(std[bid].get('python', [])))
    h_data = {}
    for h in HARNESSES:
        h_data[h] = {}
        for lang in ('ailang', 'python'):
            cell = agent[bid][h][lang]
            t = cell['total']
            if t == 0:
                h_data[h][lang] = {'raw': None, 'adj': None, 'api_err_pct': None, 'total': 0}
            else:
                non_api = t - cell['api_error']
                h_data[h][lang] = {
                    'raw': pct(cell['success'], t),
                    'adj': pct(cell['success'], non_api) if non_api > 0 else None,
                    'api_err_pct': pct(cell['api_error'], t),
                    'total': t,
                }
    rows.append((bid, tier_of[bid], sa, sp, h_data))

# === Summary: harness-wide error rates ===
hwide = defaultdict(lambda: {'total': 0, 'success': 0, 'api_error': 0, 'logic_error': 0, 'runtime_error': 0})
for f in glob.glob(f'{BASELINE_DIR}/agent/*.json'):
    try:
        with open(f) as fh: d = json.load(fh)
        h = d.get('executor', '?')
        if h not in HARNESSES: continue
        hwide[h]['total'] += 1
        if d.get('stdout_ok'): hwide[h]['success'] += 1
        cat = d.get('error_category', 'none')
        if cat in ('api_error', 'logic_error', 'runtime_error'):
            hwide[h][cat] += 1
    except Exception:
        pass

print(f"\n=== Harness-level error rates ({BASELINE_DIR}) ===")
print(f"{'Harness':<10} {'Total':<6} {'Pass':<8} {'API-err':<10} {'Logic':<8} {'Runtime':<8} {'Adjusted-pass':<14}")
for h in HARNESSES:
    s = hwide[h]
    if s['total'] == 0: continue
    p = pct(s['success'], s['total'])
    a = pct(s['api_error'], s['total'])
    non_api = s['total'] - s['api_error']
    adj = pct(s['success'], non_api) if non_api > 0 else None
    print(f"{h:<10} {s['total']:<6} {p:>4.0f}%   {a:>4.0f}% ({s['api_error']:>3}) {s['logic_error']:<8} {s['runtime_error']:<8} {fmt(adj)} (non-api runs only)")

# === Per-benchmark RAW pass rate table ===
print(f"\n=== Per-benchmark RAW pass rates (raw includes api_error failures) ===")
header = f"{'Benchmark':<32} {'Tier':<8} {'std-AI':<6} {'std-Py':<6}"
for h in HARNESSES:
    header += f" {h[:4]+'-AI':<7} {h[:4]+'-Py':<7}"
print(header)
print("-" * len(header))
for bid, tier, sa, sp, h_data in rows:
    line = f"{bid:<32} {tier:<8} {fmt(sa):<6} {fmt(sp):<6}"
    for h in HARNESSES:
        line += f" {fmt(h_data[h]['ailang']['raw']):<7} {fmt(h_data[h]['python']['raw']):<7}"
    print(line)

# === Per-benchmark ADJUSTED pass rate table (excluding API errors) ===
print(f"\n=== Per-benchmark ADJUSTED pass rates (api_error runs excluded) ===")
print(f"=== This is the real model-strength signal — saturation decisions should use this ===")
print(header)
print("-" * len(header))
for bid, tier, sa, sp, h_data in rows:
    line = f"{bid:<32} {tier:<8} {fmt(sa):<6} {fmt(sp):<6}"
    for h in HARNESSES:
        line += f" {fmt(h_data[h]['ailang']['adj']):<7} {fmt(h_data[h]['python']['adj']):<7}"
    print(line)

# === Demote recommendations using ADJUSTED rates ===
print(f"\n{'='*40}\nDemote recommendations (adjusted ≥{THRESHOLD:.0f}% on every dim with data)\n{'='*40}")
candidates_strict, candidates_adj = [], []
for bid, tier, sa, sp, h_data in rows:
    raw_dims = [v for v in (sa, sp)]
    raw_dims += [h_data[h][lang]['raw'] for h in HARNESSES for lang in ('ailang','python')]
    raw_dims = [v for v in raw_dims if v is not None]
    adj_dims = [v for v in (sa, sp)]
    adj_dims += [h_data[h][lang]['adj'] for h in HARNESSES for lang in ('ailang','python')]
    adj_dims = [v for v in adj_dims if v is not None]

    if raw_dims and all(v >= THRESHOLD for v in raw_dims):
        candidates_strict.append(bid)
    if adj_dims and all(v >= THRESHOLD for v in adj_dims):
        candidates_adj.append(bid)

print(f"\nSTRICT (raw ≥{THRESHOLD:.0f}% — counts API errors as failures):")
print("  " + (", ".join(candidates_strict) if candidates_strict else "(none)"))
print(f"\nADJUSTED (excludes api_error runs — true model strength):")
print("  " + (", ".join(candidates_adj) if candidates_adj else "(none)"))
PYEOF
