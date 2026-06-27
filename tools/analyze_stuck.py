#!/usr/bin/env python3
"""Find WHERE an agentic AILANG run got stuck — for max_steps spirals.

Loop tool (M-AGENT-ERGONOMICS): there's idle time between trials of an N-run study; use it to
analyze each max_steps session and look for the pattern that loops the model:
  1. REPEATED errors — the same (path/line-normalized) check failure the model can't fix.
  2. The PROGRESS WALL — last passing check / last write vs where the run actually ended.
  3. The TAIL — the last actions, to see the edit<->check-fail (or re-explore) loop.

Usage: tools/analyze_stuck.py <session.jsonl> [<session2.jsonl> ...]
"""
import json, sys, re
from collections import Counter

def norm_err(out):
    for l in out.split('\n'):
        if 'Error' in l and 'No errors' not in l:
            e = l.strip()
            e = re.sub(r'/\S+?\.ail', 'F', e)
            e = re.sub(r':\d+:\d+', '', e)
            e = re.sub(r'\(decl \d+\)', '(decl N)', e)
            e = re.sub(r'/var/folders/\S+', '', e)
            return e[:130]
    return out.strip()[:100]

def analyze(path):
    evs = [json.loads(l) for l in open(path) if l.strip()]
    res = {}
    for o in evs:
        if o.get('type') == 'native_tool_results':
            for r in o.get('results', []):
                res[r.get('tool_call_id')] = r
    seq = []   # (step, action, detail)
    fails = [] # (step, normalized error)
    for o in evs:
        if o.get('type') != 'native_tool_calls':
            continue
        s = o.get('step')
        for c in o.get('tool_calls', []):
            t, a = c.get('tool'), c.get('arguments', {})
            if t == 'BashExec':
                cmd = str(a.get('cmd', '')).strip()
                out = str(res.get(c.get('id'), {}).get('stdout', '')) + str(res.get(c.get('id'), {}).get('stderr', ''))
                failed = 'Error' in out and 'No errors' not in out
                if 'ailang run' in cmd or 'ailang check' in cmd:
                    if failed:
                        e = norm_err(out); fails.append((s, e)); seq.append((s, 'CHECK-FAIL', e[:58]))
                    else:
                        seq.append((s, 'check-pass', ''))
                elif '<<' in cmd or re.search(r'\s>>?\s*\S', cmd):
                    m = re.search(r'(\S+\.ail)', cmd); seq.append((s, 'write', m.group(1).split('/')[-1] if m else ''))
                elif cmd.startswith(('find', 'ls', 'cat', 'grep')):
                    seq.append((s, 'explore', cmd.split()[0]))
            elif t in ('WriteFile', 'EditFile'):
                seq.append((s, t, str(a.get('path', '')).split('/')[-1]))
            elif t == 'ReadFile':
                seq.append((s, 'read', str(a.get('path', '')).split('/')[-1]))

    rs = [o for o in evs if o.get('type') == 'run_summary']
    maxstep = max([o.get('step', 0) for o in evs] + [0])
    print(f"\n========== {path.split('/')[-1]}  ({maxstep} steps, {rs[0].get('finish_reason') if rs else '?'}) ==========")
    ec = Counter(e for _, e in fails)
    print(f"{len(fails)} failed checks. REPEATED (stuck on the same bug):")
    any_rep = False
    for e, n in ec.most_common():
        if n >= 2:
            any_rep = True
            steps = [s for s, ee in fails if ee == e]
            print(f"  x{n}  steps {steps[0]}..{steps[-1]}: {e}")
    if not any_rep:
        print("  (no error repeated >=2x — not a single-bug stall)")
    last_pass = max([s for s, a, _ in seq if a == 'check-pass'], default=None)
    last_write = max([s for s, a, _ in seq if a in ('WriteFile', 'write')], default=None)
    print(f"\nPROGRESS WALL: last check-PASS=step {last_pass} | last write=step {last_write} | ended=step {maxstep}")
    print("TAIL (last 16 actions — the loop):")
    for s, a, d in seq[-16:]:
        print(f"  {s:3} {a:11} {d}")

for p in sys.argv[1:]:
    analyze(p)
