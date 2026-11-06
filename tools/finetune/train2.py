#!/usr/bin/env python3
"""mlx-lm LoRA wrapper for pilot-2: whole-episode samples, loss on EVERY assistant turn,
rendered in the format Ollama serves (thinking ON).

Why (2026-11-06):
- Pilot-2 episodes carry pi's real ~7.3k-token system prompt + tool schemas. Training one
  sample per assistant turn (pilot-1's train.py) would repeat that prefix ~5x per episode.
  Here each episode is ONE sequence and the loss mask covers every span the model generates.
- Pilot-1 was trained thinking-OFF (an empty, closed thought pre-filled by the generation
  prompt); Ollama serves thinking-ON, and the fine-tune's tool calls landed in the thinking
  channel. Rendering with enable_thinking=True and a `reasoning` field on each assistant step
  trains `<|channel>thought ... <channel|>` + action, exactly the served format.

Span = render(messages[:k+1]) minus its longest common prefix with
render(messages[:k], add_generation_prompt=True): the thought, the call and the stop token.
Every partial render must be a token prefix of the full one (asserted), so the mask lines
up with the sequence actually trained.

Usage: python tools/finetune/train2.py <mlx_lm.lora flags...>   (data: {"messages", "tools"})
"""
import sys

import mlx.core as mx
import mlx.nn as nn
import numpy as np
from mlx_lm import lora
from mlx_lm.tuner import datasets, trainer

KW = {"enable_thinking": True}


def _render(tok, msgs, tools, gen=False):
    return list(tok.apply_chat_template(msgs, tools=tools, add_generation_prompt=gen, return_dict=False, **KW))


def _episode_process(self, d):
    msgs, tools = d[self.chat_key], d.get("tools")
    full = _render(self.tokenizer, msgs, tools)
    mask = np.zeros(len(full), np.int8)
    for k, m in enumerate(msgs):
        if m.get("role") != "assistant":
            continue
        prompt = _render(self.tokenizer, msgs[:k], tools, gen=True)
        upto = _render(self.tokenizer, msgs[: k + 1], tools)
        lcp = 0
        for a, b in zip(upto, prompt):
            if a != b:
                break
            lcp += 1
        if full[: len(upto)] != upto:
            raise ValueError("partial render is not a prefix of the full render; mask would be misaligned")
        mask[lcp: len(upto)] = 1
    return (full, mask)


datasets.ChatDataset.process = _episode_process


def masked_iterate_batches(dataset, batch_size, max_seq_length, loop=False, seed=None, comm_group=None):
    idx = sorted(range(len(dataset)), key=lambda i: dataset.itemlen(i) if hasattr(dataset, "itemlen") else len(dataset[i][0]))
    batches = [idx[i: i + batch_size] for i in range(0, len(idx) - batch_size + 1, batch_size)]
    if seed is not None:
        np.random.seed(seed)
    while True:
        for i in np.random.permutation(len(batches)):
            items = [dataset[j] for j in batches[i]]
            L = min(max(len(t) for t, _ in items), max_seq_length)
            arr = np.zeros((len(items), L), np.int32)
            msk = np.zeros((len(items), L), np.float32)
            for r, (t, m) in enumerate(items):
                n = min(len(t), L)
                arr[r, :n], msk[r, :n] = t[:n], m[:n]
            yield mx.array(arr), mx.array(msk)
        if not loop:
            break


def masked_loss(model, batch, mask):
    logits = model(batch[:, :-1])
    m = mask[:, 1:]  # token t is predicted from t-1; mask marks generated tokens
    ce = nn.losses.cross_entropy(logits, batch[:, 1:]) * m
    ntoks = m.sum()
    return ce.astype(mx.float32).sum() / ntoks, ntoks


_train, _evaluate = trainer.train, trainer.evaluate
lora.train = lambda *a, **k: _train(*a, loss=masked_loss, iterate_batches=masked_iterate_batches, **k)
lora.evaluate = lambda *a, **k: _evaluate(*a, loss=masked_loss, iterate_batches=masked_iterate_batches, **k)

if __name__ == "__main__":
    sys.argv[0] = "mlx_lm.lora"
    lora.main()
