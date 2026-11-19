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

  python3 -I tools/finetune/build_mix5.py --desc desc.jsonl --prompts DIR --out DIR [--limit N]
"""
import argparse
import collections
import json
import os
import random

PROMPTS = {"none": "ft-none.md", "slim": "ft-slim-v1.md"}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--desc", required=True)
    ap.add_argument("--prompts", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--max-chars", type=int, default=24000)  # ~8k tokens, under the 12,288 sequence limit
    ap.add_argument("--seed", type=int, default=17)
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
    groups = sorted({r["group"] for r in recs})
    rnd.shuffle(groups)
    valid_groups = set(groups[:max(1, len(groups) // 20)])
    os.makedirs(a.out, exist_ok=True)
    out = {"train": open(os.path.join(a.out, "train.jsonl"), "w"), "valid": open(os.path.join(a.out, "valid.jsonl"), "w")}
    chars = collections.Counter()
    for r in recs:
        kind = "none" if int(r["hash"], 16) % 2 == 0 else "slim"
        user = f"{prompts[kind]}\n\n## Task\n\n{r['task']}"
        if len(user) + len(r["code"]) > a.max_chars:
            rep["too_long"] += 1
            continue
        split = "valid" if r["group"] in valid_groups else "train"
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
