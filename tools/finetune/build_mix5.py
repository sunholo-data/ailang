#!/usr/bin/env python3
"""Pilot-5 training mix: single-turn language examples (docs/docs/finetuning/pilot-5.md).

Each example is one user message and one assistant message, in the shape the single-shot test
sends (cmd/ailang/eval_benchmark.go):

    user       <teaching prompt>\\n\\n## Task\\n\\n<task description>
    assistant  <the verified program>

The teaching prompt is `none` or `slim`, alternating by the program's hash. The full prompt is
about 27,000 tokens, more than twice the trainer's sequence limit, so it is not rendered here;
whether a fine-tune still works with it is measured at test time instead.

Only split == "train" records are used. Validation is 5% of the near-duplicate GROUPS, so no
validation program has a sibling in training.

--eval adds passing solutions to TRAINING-split benchmarks (extract_lang.py's eval.jsonl; held-out
tasks are already excluded there). They are the only examples shaped like the test: read stdin,
print exact output, `module benchmark/solution`. At most --eval-cap per task, each from a different
model where possible, re-checked with the pinned compiler; a task's solutions go to one split.

  python3 -I tools/finetune/build_mix5.py --desc desc.jsonl --prompts DIR --out DIR [--limit N]
      [--eval eval.jsonl --eval-cap 8]
"""
import argparse
import collections
import json
import os
import hashlib
import random
import sys

PROMPTS = {"none": "ft-none.md", "slim": "ft-slim-v1.md"}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--desc", required=True)
    ap.add_argument("--prompts", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--max-chars", type=int, default=24000)  # ~8k tokens, under the 12,288 sequence limit
    ap.add_argument("--seed", type=int, default=17)
    ap.add_argument("--eval")
    ap.add_argument("--eval-cap", type=int, default=8)
    a = ap.parse_args()
    prompts = {k: open(os.path.join(a.prompts, f)).read().strip() for k, f in PROMPTS.items()}
    recs = [json.loads(line) for line in open(a.desc)]
    rep = collections.Counter()
    rep["heldback_test"] = sum(r["split"] != "train" for r in recs)
    recs = [r for r in recs if r["split"] == "train"]
    rnd = random.Random(a.seed)
    rnd.shuffle(recs)
    if a.limit:
        recs = recs[:a.limit]
    if a.eval:
        sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
        from indist_test import check
        per = collections.defaultdict(list)
        for line in open(a.eval):
            d = json.loads(line)
            if d.get("task"):
                per[d["benchmark"]].append(d)
        for bid in sorted(per):
            items = per[bid]
            rnd.shuffle(items)
            kept, models, codes = 0, set(), set()
            for want_new_model in (True, False):
                for d in items:
                    h = hashlib.sha256(d["code"].encode()).hexdigest()[:16]
                    if kept >= a.eval_cap or h in codes or (want_new_model and d["model"] in models):
                        continue
                    if not check(d["code"])[0]:
                        rep["eval_fails_pinned_check"] += 1
                        codes.add(h)
                        continue
                    codes.add(h)
                    models.add(d["model"])
                    kept += 1
                    recs.append({"hash": h, "group": "bench:" + bid, "split": "train", "task": d["task"], "code": d["code"], "kind": "bench"})
        rnd.shuffle(recs)
    groups = sorted({str(r["group"]) for r in recs})
    rnd.shuffle(groups)
    valid_groups = set(groups[:max(1, len(groups) // 20)])
    os.makedirs(a.out, exist_ok=True)
    out = {"train": open(os.path.join(a.out, "train.jsonl"), "w"), "valid": open(os.path.join(a.out, "valid.jsonl"), "w")}
    chars = collections.Counter()
    for r in recs:
        kind = "none" if int(r["hash"], 16) % 2 == 0 else "slim"
        pk = kind
        user = f"{prompts[pk]}\n\n## Task\n\n{r['task']}"
        if len(user) + len(r["code"]) > a.max_chars:
            rep["too_long"] += 1
            continue
        split = "valid" if str(r["group"]) in valid_groups else "train"
        kind = ("bench-" if r.get("kind") == "bench" else "") + kind
        out[split].write(json.dumps({"messages": [{"role": "user", "content": user},
                                                  {"role": "assistant", "content": r["code"].strip()}]}) + "\n")
        rep[f"{split}:{kind}"] += 1
        chars[split] += len(user) + len(r["code"])
    for f in out.values():
        f.close()
    report = {"samples": dict(rep), "approx_tokens": {k: v // 3 for k, v in chars.items()}}
    json.dump(report, open(os.path.join(a.out, "mix_report.json"), "w"), indent=1)
    print(json.dumps(report))


if __name__ == "__main__":
    main()
