#!/usr/bin/env python3
"""motoko_diag.py <session.jsonl | session_id> — reliable diagnostic for a motoko run.
Reads what the harness ACTUALLY logs (tool via 'tool' key, input_tokens via 'thinking'),
and makes the OBSERVABILITY GAPS explicit (token drops with no compaction event = silent compaction)."""
import json, sys, glob
from collections import Counter, defaultdict

arg = sys.argv[1]
path = arg if arg.endswith(".jsonl") else None
if not path:
    hits = glob.glob(f"/Users/voightkampff/dev/*/.motoko/logfile/*{arg}*.jsonl")
    path = hits[0] if hits else None
if not path: sys.exit(f"no jsonl for {arg}")
evs = [json.loads(l) for l in open(path) if l.strip()]
print(f"== {path.split('/')[-1]} :: {len(evs)} events ==\n")

# outcome
rs = next((o for o in evs if o.get("type")=="run_summary"), {})
print(f"OUTCOME: finish={rs.get('finish_reason')} steps={rs.get('steps_executed')} err={rs.get('error')!r}")

# tool histogram (correct 'tool' key) + per-step tool sequence
calls = []  # (step, tool, target)
for o in evs:
    if o.get("type")=="native_tool_calls":
        for c in o.get("tool_calls", []):
            a = c.get("arguments", {})
            tgt = str(a.get("path") or a.get("file") or a.get("cmd") or a.get("command") or "")[:70]
            calls.append((o.get("step"), c.get("tool"), tgt))
print(f"\nTOOLS ({len(calls)} calls): {dict(Counter(t for _,t,_ in calls))}")

# input_tokens trajectory + UNEXPLAINED DROPS (the observability gap)
toks = {o["step"]: o["input_tokens"] for o in evs if o.get("type")=="thinking" and "input_tokens" in o}
comp_steps = {o.get("step") for o in evs if "compact" in o.get("type","").lower()}
print(f"\nINPUT_TOKENS: peak={max(toks.values()) if toks else 0}  (limit 262144)")
prev=None; silent=0
for s in sorted(toks):
    v=toks[s]
    if prev and v < prev-20000:
        tag = "compaction logged" if s in comp_steps else ">>> SILENT (no compaction event) <<<"
        if s not in comp_steps: silent+=1
        print(f"   step {s}: {prev:>7} -> {v:>7}  drop {prev-v:>7}  [{tag}]")
    prev=v
print(f"   => {silent} context-drops with NO logged cause  (THIS is the observability gap)")

# edit -> typecheck correlation
edits=[(s,t,tgt) for s,t,tgt in calls if t in ("WriteFile","EditFile","EditDecl")]
txt=open(path).read()
print(f"\nEDITS: {len(edits)} (Write/Edit/EditDecl) vs typecheck events: OK={txt.count('ailang check: OK')} FAILED={txt.count('ailang check: FAILED')}")
if edits: print(f"   edit steps: {[s for s,_,_ in edits]}")

# stuck-loop detector: consecutive identical (tool,target)
print("\nSTUCK-LOOP (>=4 consecutive identical tool calls):")
runs=[]; cur=None; n=0
for s,t,tgt in calls:
    key=(t,tgt)
    if key==cur: n+=1
    else:
        if n>=4: runs.append((cur,n,laststep))
        cur=key; n=1
    laststep=s
if n>=4: runs.append((cur,n,laststep))
for (t,tgt),n,ls in runs: print(f"   {n}x {t} {tgt!r} (ending ~step {ls})")
if not runs: print("   none")
