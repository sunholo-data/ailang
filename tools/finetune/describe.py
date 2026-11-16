#!/usr/bin/env python3
"""Write a task description for each verified AILANG program (pilot-5 stage 1).

A repository program has no prompt, so it cannot be a training example. A strong model reads the
program and writes the task a developer would have been given; the program is the answer and is
already verified, so nothing here needs checking by the compiler. Claude runs through the CLI on
the subscription (stale_fix.ask_claude), never the API.

  python3 -I tools/finetune/describe.py --src repo.split.jsonl --out desc.jsonl \
      [--max-lines 150] [--limit N] [--seed 17] [--workers 4] [--model claude-opus-5-5]

Resumable: records already in --out (by content hash) are skipped. A description that quotes a
long run of the program's own code is rejected (`leak`) and not written.
"""
import argparse
import concurrent.futures as cf
import hashlib
import json
import os
import random
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from stale_fix import ask_claude  # noqa: E402

PROMPT = """Below is a complete, working program in AILANG, a functional language. Write the task \
description a developer would have been given before writing it, so that someone who has never \
seen this program could write an equivalent one.

Rules:
- State what the program is for, then the exact behaviour: inputs, outputs, formats, edge cases, \
and any constants or messages the output depends on.
- Name the module path, and every exported function or type with its signature, because other \
code depends on those names. Do not name private helpers.
- Name the effects the entry points need (for example IO or FS) if there are any.
- Describe behaviour, not implementation. Do not include AILANG code other than the signatures, \
and do not describe how the body is written.
- Plain prose and short lists. No preamble, no title, no closing remark. Under 350 words.

Program ({path}):

{code}
"""
SIG = re.compile(r"^\s*(export\s+)?(pure\s+)?(func|type)\b")


def leak(desc, code):
    """True when the description copies a body line of 40+ characters (signatures are allowed)."""
    body = {ln.strip() for ln in code.split("\n") if len(ln.strip()) >= 40 and not SIG.match(ln)}
    return any(b in desc for b in body)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--src", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--max-lines", type=int, default=150)
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--seed", type=int, default=17)
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--model", default="claude-opus-5-5")
    a = ap.parse_args()
    done = set()
    if os.path.exists(a.out):
        done = {json.loads(line)["hash"] for line in open(a.out)}
    recs = [json.loads(line) for line in open(a.src)]
    recs = [r for r in recs if r["lines"] <= a.max_lines]
    random.Random(a.seed).shuffle(recs)
    todo = []
    for r in recs:
        r["hash"] = hashlib.sha256(r["code"].encode()).hexdigest()[:16]
        if r["hash"] not in done:
            todo.append(r)
    if a.limit:
        todo = todo[:max(0, a.limit - len(done))]
    stats = {"asked": 0, "written": 0, "leak": 0, "error": 0, "in": 0, "out": 0}

    def one(r):
        try:
            txt, u = ask_claude(a.model, PROMPT.format(path=r.get("path", "?").split("/dev/")[-1], code=r["code"]))
        except Exception as e:  # noqa: BLE001 -- one failed request must not stop the batch
            return r, None, str(e)[:80], {}
        return r, txt.strip(), None, u

    with cf.ThreadPoolExecutor(a.workers) as ex, open(a.out, "a") as f:
        for r, txt, err, u in ex.map(one, todo):
            stats["asked"] += 1
            if err or not txt:
                stats["error"] += 1
                continue
            stats["in"] += u.get("prompt_tokens", 0)
            stats["out"] += u.get("completion_tokens", 0)
            if leak(txt, r["code"]):
                stats["leak"] += 1
                continue
            f.write(json.dumps({"hash": r["hash"], "path": r.get("path"), "group": r.get("group"),
                                "split": r.get("split"), "lines": r["lines"], "task": txt, "code": r["code"]}) + "\n")
            f.flush()
            stats["written"] += 1
    print(json.dumps(stats))


if __name__ == "__main__":
    main()
