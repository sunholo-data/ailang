#!/usr/bin/env python3
"""Pilot-5 in-distribution test: does the model write a compiling program for a held-back
repository task description?

The held-out benchmarks differ from the training data in kind (stdin/stdout tasks, exact output).
This test asks the narrower question first: on descriptions of the SAME kind as training, from
programs the model never saw (split == "test", whole near-duplicate groups held back), does the
answer pass `ailang check`?

Each answer is written to <tmp>/<module path>.ail and checked alone, with the pinned compiler and
stdlib. A reference program that does not check alone (it imports a sibling module) is dropped
from the test, so every scored task is known to be solvable in this setup.

  python3 -I tools/finetune/indist_test.py --desc desc.jsonl --model NAME --prompt none|slim \
      --prompts DIR --out FILE.jsonl [--limit N] [--no-think]

Needs the rig (ollama). Resumable by task hash.
"""
import argparse
import collections
import json
import os
import re
import subprocess
import tempfile
import urllib.request

FT = os.path.expanduser("~/models/finetune")
AILANG = FT + "/bin/pinned/ailang"
STD = FT + "/std-ft3/std"
PROMPTS = {"none": "ft-none.md", "slim": "ft-slim-v1.md"}
MOD = re.compile(r"^\s*module\s+([\w/.\-]+)", re.M)


def check(code):
    m = MOD.search(code)
    if not m or ".." in m.group(1) or m.group(1).startswith("/"):
        return False, "no module line"
    with tempfile.TemporaryDirectory() as td:
        path = os.path.join(td, m.group(1) + ".ail")
        os.makedirs(os.path.dirname(path), exist_ok=True)
        open(path, "w").write(code)
        env = dict(os.environ, AILANG_STDLIB_PATH=STD)
        try:
            r = subprocess.run([AILANG, "check", m.group(1) + ".ail"], cwd=td, env=env, capture_output=True, text=True, timeout=60)
        except subprocess.TimeoutExpired:
            return False, "check timeout"
        err = (r.stdout + r.stderr).strip()
        return r.returncode == 0, "" if r.returncode == 0 else re.sub(r"\S+\.ail:\d+:\d+", "<loc>", err)[:200]


def ask(model, content, think):
    body = {"model": model, "stream": False, "messages": [{"role": "user", "content": content}],
            "options": {"num_predict": 12000}}
    if not think:
        body["think"] = False
    req = urllib.request.Request("http://127.0.0.1:11434/api/chat", json.dumps(body).encode(), {"Content-Type": "application/json"})
    d = json.load(urllib.request.urlopen(req, timeout=1800))
    txt = (d.get("message") or {}).get("content") or ""
    blocks = re.findall(r"```[\w-]*\n(.*?)```", txt, re.S)
    return (blocks[-1] if blocks else txt).strip(), d.get("eval_count", 0)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--desc", required=True)
    ap.add_argument("--model", required=True)
    ap.add_argument("--prompt", choices=list(PROMPTS), default="none")
    ap.add_argument("--prompts", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--no-think", action="store_true")
    a = ap.parse_args()
    prompt = open(os.path.join(a.prompts, PROMPTS[a.prompt])).read().strip()
    recs = [json.loads(line) for line in open(a.desc)]
    recs = sorted((r for r in recs if r["split"] == "test"), key=lambda r: r["hash"])
    done = {}
    if os.path.exists(a.out):
        done = {json.loads(line)["hash"]: json.loads(line) for line in open(a.out)}
    with open(a.out, "a") as f:
        n = 0
        for r in recs:
            if a.limit and n >= a.limit:
                break
            if r["hash"] in done:
                n += done[r["hash"]]["ref_ok"]
                continue
            ref_ok, _ = check(r["code"])
            row = {"hash": r["hash"], "path": r["path"], "ref_ok": ref_ok}
            if ref_ok:
                code, toks = ask(a.model, f"{prompt}\n\n## Task\n\n{r['task']}", not a.no_think)
                ok, err = check(code)
                row.update(ok=ok, err=err, tokens=toks, code=code)
                n += 1
            f.write(json.dumps(row) + "\n")
            f.flush()
            done[r["hash"]] = row
    scored = [d for d in done.values() if d["ref_ok"]]
    ok = sum(d["ok"] for d in scored)
    errs = collections.Counter(re.sub(r"\(decl \d+\)", "", d["err"])[:70] for d in scored if not d["ok"])
    toks = sorted(d["tokens"] for d in scored) or [0]
    print(json.dumps({"model": a.model, "prompt": a.prompt, "scored": len(scored), "compile": ok,
                      "rate": round(ok / max(len(scored), 1), 3), "dropped_reference_not_standalone": len(done) - len(scored),
                      "median_tokens": toks[len(toks) // 2], "top_errors": errs.most_common(5)}))


if __name__ == "__main__":
    main()
