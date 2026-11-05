#!/usr/bin/env python3
"""Pre-training gate for a mix directory (run with the finetune venv python).

Renders every sample exactly as train.py will (inference-aligned masking), drops
samples longer than --max-seq (mlx-lm would truncate the END, i.e. the target),
rewrites train.jsonl / valid.jsonl in place, and exits non-zero if any sample
fails to render or has an empty target. Prints the real token totals that the
training-time estimate should be based on.
"""
import argparse
import json
import os
import statistics
import sys
from collections import Counter

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import train  # noqa: E402,F401  (installs the aligned ChatDataset.process)
from mlx_lm.tuner.datasets import ChatDataset  # noqa: E402
from transformers import AutoTokenizer  # noqa: E402


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mix", required=True)
    ap.add_argument("--model", required=True)
    ap.add_argument("--max-seq", type=int, default=8192)
    a = ap.parse_args()
    tok = AutoTokenizer.from_pretrained(a.model)
    failed = 0
    for split in ("train", "valid"):
        path = os.path.join(a.mix, f"{split}.jsonl")
        data = [json.loads(ln) for ln in open(path)]
        ds = ChatDataset(data, tok, mask_prompt=True)
        keep, lens, tgt, drop = [], [], [], Counter()
        for d in data:
            try:
                t, off = ds.process(d)
            except Exception as e:  # report, never silently keep a sample that cannot render
                drop["render_error:" + type(e).__name__] += 1
                failed += 1
                continue
            if len(t) - off <= 0:
                drop["empty_target"] += 1
                failed += 1
                continue
            if len(t) > a.max_seq:
                drop["too_long"] += 1
                continue
            keep.append(d)
            lens.append(len(t))
            tgt.append(len(t) - off)
        with open(path, "w") as f:
            for d in keep:
                f.write(json.dumps(d) + "\n")
        print(f"{split}: kept {len(keep)} dropped {dict(drop)} tokens {sum(lens)} "
              f"target_tokens {sum(tgt)} median_len {int(statistics.median(lens))} max_len {max(lens)}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
