#!/usr/bin/env python3
"""Find WHERE an agentic AILANG run got stuck, and WHY it couldn't fix it — for max_steps spirals.

The loop's deep-analysis step (M-AGENT-ERGONOMICS). For each max_steps session it surfaces:
  1. REPEATED errors — the same (path/line-normalized) check failure the model never fixed.
  2. The PROGRESS WALL — last passing check / last write vs where the run actually ended.
  3. The TAIL — the last actions (the edit<->check-fail or re-explore loop).
  4. A DOSSIER per repeated error — the full UNTRUNCATED error PLUS the model's own edits across
     each failed attempt, so the *mechanism* is visible: what it kept changing vs the invariant it
     never touched (the blind spot). This is what turns "show failed x4" into "it reshuffled the
     import 4x and never removed show" — the two manual drill-downs, automated, every time.

Usage:
  tools/analyze_stuck.py <session.jsonl> [more.jsonl ...]
  tools/analyze_stuck.py <dir>     # scan dir for session_*.jsonl; deep-analyze the max_steps ones
                                   # e.g. tools/analyze_stuck.py ~/dev/mk-ast/.motoko/logfile
"""
import json, sys, re, glob, os
from collections import Counter


def load(p):
    return [json.loads(l) for l in open(p) if l.strip()]


def results_index(evs):
    res = {}
    for o in evs:
        if o.get('type') == 'native_tool_results':
            for r in o.get('results', []):
                res[r.get('tool_call_id')] = r
    return res


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


def err_lines(out):
    """The full, untruncated error lines (the drill-down step 2: exact code/message/location)."""
    return [l.strip() for l in out.split('\n')
            if re.search(r'(PAR|IMP|TC|LDR|MOD)\d|type error|parser panic|not exported|undefined', l)][:3]


def model_writes(evs):
    """Every file-mutating action: (step, target_path, content). Heredoc/redirect writes carry the
    whole cmd as content, so a symbol search over it still finds the written lines."""
    w = []
    for o in evs:
        if o.get('type') != 'native_tool_calls':
            continue
        s = o.get('step')
        for c in o.get('tool_calls', []):
            t, a = c.get('tool'), c.get('arguments', {})
            if t in ('WriteFile', 'EditFile'):
                w.append((s, str(a.get('path', '')), str(a.get('content', ''))))
            elif t == 'BashExec':
                cmd = str(a.get('cmd', ''))
                if '<<' in cmd or re.search(r'\s>>?\s*\S', cmd):
                    m = re.search(r'>\s*(\S+\.ail)', cmd)
                    w.append((s, m.group(1) if m else '', cmd))
    return w


def dossier(occ, writes):
    """occ: [(step, full_out)] for one repeated error. Emit the full error + the per-attempt
    mechanism, so the invariant the model never changed (its blind spot) is visible."""
    steps = [s for s, _ in occ]
    full = occ[0][1]
    print(f"\n  -- DOSSIER: repeated x{len(occ)} (steps {steps[0]}..{steps[-1]}) --")
    for l in err_lines(full):
        print(f"     {l[:116]}")
    in_window = lambda s: steps[0] - 2 <= s <= steps[-1] + 2

    msym = re.search(r"symbol '(\w+)'", full) or re.search(r"variable:?\s*'?(\w+)'?", full)
    if msym:
        sym = msym.group(1)
        seen, dist = set(), []
        for s, _path, content in writes:
            if not in_window(s):
                continue
            for l in content.split('\n'):
                ls = l.strip()
                if sym in ls and ('import' in ls or f'{sym}(' in ls or f'{sym},' in ls or f', {sym}' in ls):
                    if ls not in seen:
                        seen.add(ls)
                        dist.append((s, ls))
                    break
        if dist:
            print(f"     subject '{sym}' — the model's distinct edits that mention it, in order:")
            for s, l in dist[:8]:
                print(f"       step {s:3}: {l[:72]}")
            print(f"     -> {len(dist)} distinct attempts, every one keeps '{sym}'. That invariant is")
            print(f"        the blind spot. Fix question: what error message makes it remove/move '{sym}'?")
        else:
            print(f"     subject '{sym}' (no matching model edit found in window — check exploration loop)")
        return

    mloc = re.search(r'([\w./-]+\.ail):(\d+):', full)
    if mloc:
        fpath, ln = mloc.group(1), int(mloc.group(2))
        base = fpath.split('/')[-1]
        seen = set()
        for s, path, content in writes:
            if not in_window(s) or base not in path:
                continue
            lines = content.split('\n')
            if ln - 1 < len(lines):
                cur = lines[ln - 1].strip()
                if cur and cur not in seen:
                    seen.add(cur)
        print(f"     offending construct at {fpath}:{ln} — distinct content of that line across writes: {len(seen)}")
        for cur in list(seen)[:6]:
            print(f"       L{ln}: {cur[:70]}")
        if len(seen) <= 1:
            print(f"     -> the offending line never changed — the model edited elsewhere (wrong target).")
        return
    print("     (no symbol/location subject parsed — inspect the full error above)")


def analyze(path, deep=True):
    evs = load(path)
    res = results_index(evs)
    seq = []       # (step, action, detail)
    fails = []     # (step, norm, full_out)
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
                        nm = norm_err(out)
                        fails.append((s, nm, out))
                        seq.append((s, 'CHECK-FAIL', nm[:58]))
                    else:
                        seq.append((s, 'check-pass', ''))
                elif '<<' in cmd or re.search(r'\s>>?\s*\S', cmd):
                    m = re.search(r'(\S+\.ail)', cmd)
                    seq.append((s, 'write', m.group(1).split('/')[-1] if m else ''))
                elif cmd.startswith(('find', 'ls', 'cat', 'grep')):
                    seq.append((s, 'explore', cmd.split()[0]))
            elif t in ('WriteFile', 'EditFile'):
                seq.append((s, t, str(a.get('path', '')).split('/')[-1]))
            elif t == 'ReadFile':
                seq.append((s, 'read', str(a.get('path', '')).split('/')[-1]))

    rs = [o for o in evs if o.get('type') == 'run_summary']
    maxstep = max([o.get('step', 0) for o in evs] + [0])
    fr = rs[0].get('finish_reason') if rs else '?'
    print(f"\n========== {path.split('/')[-1]}  ({maxstep} steps, {fr}) ==========")

    groups = {}
    for s, nm, out in fails:
        groups.setdefault(nm, []).append((s, out))
    rep = sorted([(nm, occ) for nm, occ in groups.items() if len(occ) >= 2], key=lambda x: -len(x[1]))
    print(f"{len(fails)} failed checks. REPEATED (stuck on the same bug):")
    for nm, occ in rep:
        st = [s for s, _ in occ]
        print(f"  x{len(occ)}  steps {st[0]}..{st[-1]}: {nm}")
    if not rep:
        print("  (no error repeated >=2x — not a single-bug stall)")

    last_pass = max([s for s, a, _ in seq if a == 'check-pass'], default=None)
    last_write = max([s for s, a, _ in seq if a in ('WriteFile', 'write')], default=None)
    print(f"\nPROGRESS WALL: last check-PASS=step {last_pass} | last write=step {last_write} | ended=step {maxstep}")
    print("TAIL (last 16 actions):")
    for s, a, d in seq[-16:]:
        print(f"  {s:3} {a:11} {d}")

    if deep and rep:
        writes = model_writes(evs)
        print("\nWHY IT COULDN'T FIX IT (per repeated error):")
        for _nm, occ in rep:
            dossier(occ, writes)


def is_maxsteps(path):
    try:
        for l in open(path):
            o = json.loads(l)
            if o.get('type') == 'run_summary':
                return o.get('finish_reason') == 'max_steps'
    except Exception:
        pass
    return False


def main():
    paths = []
    for a in sys.argv[1:]:
        if os.path.isdir(a):
            cand = sorted(glob.glob(os.path.join(a, '**', 'session_*.jsonl'), recursive=True))
            ms = [c for c in cand if is_maxsteps(c)]
            print(f"# {a}: {len(cand)} sessions, {len(ms)} max_steps -> deep-analyzing those")
            paths += ms
        else:
            paths.append(a)
    if not paths:
        print(__doc__)
        sys.exit(1)
    for p in paths:
        analyze(p)


if __name__ == '__main__':
    main()
