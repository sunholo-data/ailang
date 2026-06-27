#!/usr/bin/env python3
"""Decompose where an agentic AILANG run spent its steps, and diff two runs.

A loop tool for M-AGENT-ERGONOMICS: after shipping a diagnostic and re-running a benchmark,
run this against (baseline_session, new_session) to see — with data, not assumptions — where the
step delta came from: which tool actions grew, which AILANG fix-cycle categories changed, the full
type-error messages, and the imports each run actually used (to catch a fix INDUCING a regression).

Usage: tools/analyze_run_steps.py <baseline.jsonl> <new.jsonl>
Sessions live in mk-ast/.motoko/logfile/session_*.jsonl
"""
import json, sys, re
from collections import Counter

def load(p): return [json.loads(l) for l in open(p) if l.strip()]

def results_map(evs):
    res = {}
    for o in evs:
        if o.get('type') == 'native_tool_results':
            for r in o.get('results', []):
                res[r.get('tool_call_id')] = r
    return res

def fail_cat(out):
    if 'undefined variable' in out: return 'import-miss'
    if 'PAR_MATCH_ARROW' in out:    return 'arrow'
    if 'requires capability' in out: return 'effect'
    if 'type error' in out or 'unify' in out: return 'type'
    if 'parse error' in out or re.search(r'PAR_[A-Z]', out): return 'parse'
    return 'other-fail'

IMPORT_RE = re.compile(r'import\s+(std/\w+)')

def decompose(path):
    evs = load(path); res = results_map(evs)
    steps = max([o.get('step', 0) for o in evs] + [0])
    rs = [o for o in evs if o.get('type') == 'run_summary']
    actions, fails = Counter(), Counter()
    type_errs, imports = [], Counter()
    for o in evs:
        if o.get('type') == 'stream_error_retry':
            actions['stream-retry'] += 1
        if o.get('type') != 'native_tool_calls':
            continue
        for c in o.get('tool_calls', []):
            t = c.get('tool'); args = c.get('arguments', {})
            if t in ('WriteFile', 'EditFile'):
                actions['write/edit'] += 1
                for m in IMPORT_RE.findall(str(args.get('content', ''))):
                    imports[m] += 1
            elif t == 'ReadFile':
                actions['read'] += 1
            elif t == 'BashExec':
                cmd = str(args.get('cmd', ''))
                out = str(res.get(c.get('id'), {}).get('stdout', '')) + str(res.get(c.get('id'), {}).get('stderr', ''))
                if 'ailang run' in cmd or 'ailang check' in cmd:
                    if 'Error' in out and 'No errors' not in out:
                        cat = fail_cat(out); actions['check-FAIL'] += 1; fails[cat] += 1
                        if cat == 'type':
                            tl = [l.strip() for l in out.split('\n') if 'type error' in l or 'unify' in l]
                            type_errs.append((o.get('step'), tl[0][:110] if tl else out[:110]))
                    else:
                        actions['check-PASS'] += 1
                elif 'ailang docs' in cmd:
                    actions['docs-lookup'] += 1
                elif any(x in cmd for x in ('find ', 'ls ', 'cat ', 'grep', 'head', 'tail', 'wc ')):
                    actions['explore'] += 1
                else:
                    actions['bash-other'] += 1
            else:
                actions['tool:' + str(t)] += 1
    return dict(steps=steps, finish=(rs[0].get('finish_reason') if rs else 'none'),
                actions=actions, fails=fails, type_errs=type_errs, imports=imports)

a, b = decompose(sys.argv[1]), decompose(sys.argv[2])
print(f"BASELINE {sys.argv[1].split('/')[-1]}: {a['steps']} steps ({a['finish']})")
print(f"NEW      {sys.argv[2].split('/')[-1]}: {b['steps']} steps ({b['finish']})")
print(f"DELTA: {b['steps']-a['steps']:+d} steps\n")
print("=== tool-call histogram: baseline -> new (delta), sorted by delta ===")
for k in sorted(set(a['actions']) | set(b['actions']), key=lambda k: -(b['actions'].get(k, 0) - a['actions'].get(k, 0))):
    print(f"  {k:14} {a['actions'].get(k,0):3} -> {b['actions'].get(k,0):3}  ({b['actions'].get(k,0)-a['actions'].get(k,0):+d})")
print("\n=== failed ailang check/run by category: baseline -> new ===")
for k in sorted(set(a['fails']) | set(b['fails'])):
    print(f"  {k:12} {a['fails'].get(k,0)} -> {b['fails'].get(k,0)}")
print(f"\n=== NEW-run type errors ({len(b['type_errs'])}) — full messages (causal probe) ===")
for s, m in b['type_errs']:
    print(f"  step {s:3}: {m}")
print("\n=== imports written: baseline vs new (catch a suggestion inducing a wrong import) ===")
for k in sorted(set(a['imports']) | set(b['imports'])):
    print(f"  {k:16} base x{a['imports'].get(k,0):2}  new x{b['imports'].get(k,0):2}")
