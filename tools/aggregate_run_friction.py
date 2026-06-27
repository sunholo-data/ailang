#!/usr/bin/env python3
"""Aggregate AILANG agentic-run friction across N session logs (mean ± variance per category).

The signal-finder for the M-AGENT-ERGONOMICS loop. Single agentic runs have huge path variance, so
a one-run "X got worse" is almost always noise. To decide what to smooth next, run a benchmark N
times on the rig (local ollama = free tokens) and aggregate here BEFORE concluding a cause. Pair
with tools/analyze_run_steps.py (which diffs two single runs in detail).

Usage:
  tools/aggregate_run_friction.py <session1.jsonl> <session2.jsonl> ...
  tools/aggregate_run_friction.py "$(ls mk-ast/.motoko/logfile/session_docx_*.jsonl)"

Only aggregate runs from the SAME condition (same code state). Mixing conditions is exactly the
un-rigorous mistake this tool exists to prevent.
"""
import json, sys, re, glob, statistics
from collections import Counter

def load(p): return [json.loads(l) for l in open(p) if l.strip()]

def fail_cat(out):
    if 'undefined variable' in out: return 'import-miss'
    if 'PAR_MATCH_ARROW' in out:    return 'arrow'
    if 'requires capability' in out: return 'effect'
    if 'type error' in out or 'unify' in out: return 'type'
    if 'parse error' in out or re.search(r'PAR_[A-Z]', out): return 'parse'
    return 'other-fail'

def run_counts(path):
    evs = load(path); res = {}
    for o in evs:
        if o.get('type') == 'native_tool_results':
            for r in o.get('results', []):
                res[r.get('tool_call_id')] = r
    c = Counter()
    c['steps'] = max([o.get('step', 0) for o in evs] + [0])
    rs = [o for o in evs if o.get('type') == 'run_summary']
    c['finished_stop'] = 1 if (rs and rs[0].get('finish_reason') == 'stop') else 0
    for o in evs:
        if o.get('type') == 'stream_error_retry':
            c['stream-retry'] += 1
        if o.get('type') != 'native_tool_calls':
            continue
        for tc in o.get('tool_calls', []):
            t = tc.get('tool')
            if t in ('WriteFile', 'EditFile'):
                c['write/edit'] += 1
            elif t == 'ReadFile':
                c['read'] += 1
            elif t == 'BashExec':
                cmd = str(tc.get('arguments', {}).get('cmd', ''))
                out = str(res.get(tc.get('id'), {}).get('stdout', '')) + str(res.get(tc.get('id'), {}).get('stderr', ''))
                if 'ailang run' in cmd or 'ailang check' in cmd:
                    if 'Error' in out and 'No errors' not in out:
                        c['check-FAIL'] += 1; c['fail:' + fail_cat(out)] += 1
                    else:
                        c['check-PASS'] += 1
                elif 'ailang docs' in cmd:
                    c['docs-lookup'] += 1
                elif '<<' in cmd or re.search(r'\s>>?\s*\S', cmd):
                    c['bash-write'] += 1  # `cat > f << EOF` / redirects are WRITES, not exploration
                elif any(x in cmd for x in ('find ', 'ls ', 'cat ', 'grep', 'head', 'tail', 'wc ')):
                    c['explore'] += 1
                else:
                    c['bash-other'] += 1
    return c

paths = []
for a in sys.argv[1:]:
    g = glob.glob(a)
    paths += g if g else ([a] if a.endswith('.jsonl') else a.split())
paths = [p for p in paths if p.endswith('.jsonl')]
if not paths:
    print(__doc__); sys.exit(1)

runs = [run_counts(p) for p in paths]
N = len(runs)
keys = set().union(*[set(r) for r in runs])
print(f"=== N={N} runs (same condition) ===")
for p in paths:
    print(f"  - {p.split('/')[-1]}")
print(f"\n  {'metric':18} {'mean':>7} {'std':>7} {'min':>4} {'max':>4}   (CV = std/mean: high = noisy)")
for k in sorted(keys, key=lambda k: -statistics.mean([r.get(k, 0) for r in runs])):
    vals = [r.get(k, 0) for r in runs]
    mean = statistics.mean(vals)
    std = statistics.pstdev(vals) if N > 1 else 0.0
    cv = (std / mean) if mean else 0.0
    print(f"  {k:18} {mean:7.1f} {std:7.1f} {min(vals):4d} {max(vals):4d}   {('CV=%.2f' % cv) if mean else ''}")
print("\nRead: high-mean LOW-CV categories are the reliable friction to target next;")
print("      high-CV categories are noise — do NOT target them off a single run.")
