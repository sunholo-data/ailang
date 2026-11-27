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

try:
    import numpy as np
except ModuleNotFoundError:
    np = None

try:
    import mlx.core as mx
    import mlx.nn as nn
    from mlx_lm import lora
    from mlx_lm.tuner import datasets, trainer
except ModuleNotFoundError as _exc:
    mx = nn = lora = datasets = trainer = None
    _IMPORT_ERROR = _exc
else:
    _IMPORT_ERROR = None

KW = {"enable_thinking": True}
_GEMMA4_THINKING_BUG_PATTERNS = (
    "thinking_text and loop.index0 > ns_turn.last_user_idx and message.get('tool_calls')",
    'thinking_text and loop.index0 > ns_turn.last_user_idx and message.get("tool_calls")',
)
_GEMMA4_THINKING_FIXED = "thinking_text and loop.index0 > ns_turn.last_user_idx"


def ensure_thinking_chat_template(tok):
    """Patch Gemma 4's chat_template in-place so non-tool assistant turns retain reasoning."""
    if tok is None or getattr(tok, "_thinking_template_patched", False):
        return tok
    inner = getattr(tok, "_tokenizer", None)
    if inner is not None and inner is not tok:
        ensure_thinking_chat_template(inner)
    ct = getattr(tok, "chat_template", None)
    if isinstance(ct, str):
        for bad in _GEMMA4_THINKING_BUG_PATTERNS:
            if bad in ct:
                ct = ct.replace(bad, _GEMMA4_THINKING_FIXED)
        if ct != tok.chat_template:
            tok.chat_template = ct
    try:
        tok._thinking_template_patched = True
    except (AttributeError, TypeError):
        pass
    return tok


def _render(tok, msgs, tools=None, gen=False):
    ensure_thinking_chat_template(tok)
    return list(tok.apply_chat_template(msgs, tools=tools, add_generation_prompt=gen, return_dict=False, **KW))


def _episode_process_tokens(tok, msgs, tools=None):
    ensure_thinking_chat_template(tok)
    full = _render(tok, msgs, tools)
    mask = [0] * len(full)
    for k, m in enumerate(msgs):
        if m.get("role") != "assistant" or m.get("weight") == 0:
            continue  # weight 0: the model's own mistake kept as context (pilot-3), no loss
        prompt = _render(tok, msgs[:k], tools, gen=True)
        upto = _render(tok, msgs[: k + 1], tools)
        lcp = 0
        for a, b in zip(upto, prompt):
            if a != b:
                break
            lcp += 1
        if full[: len(upto)] != upto:
            raise ValueError("partial render is not a prefix of the full render; mask would be misaligned")
        reasoning = (m.get("reasoning") or m.get("reasoning_content") or "").strip()
        if reasoning:
            target_text = tok.decode(full[lcp: len(upto)])
            if "<|channel>thought\n" not in target_text or "<channel|>" not in target_text:
                raise ValueError("reasoning field was dropped from rendered target span; check chat_template")
        for i in range(lcp, len(upto)):
            mask[i] = 1
    return (full, mask)


def _episode_process(self, d):
    full, mask_list = _episode_process_tokens(self.tokenizer, d[self.chat_key], d.get("tools"))
    mask = np.asarray(mask_list, dtype=np.int8) if np is not None else mask_list
    return (full, mask)


if datasets is not None:
    datasets.ChatDataset.process = _episode_process


def masked_iterate_batches(dataset, batch_size, max_seq_length, loop=False, seed=None, comm_group=None):
    if _IMPORT_ERROR is not None:
        raise _IMPORT_ERROR
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
    if _IMPORT_ERROR is not None:
        raise _IMPORT_ERROR
    logits = model(batch[:, :-1])
    m = mask[:, 1:]  # token t is predicted from t-1; mask marks generated tokens
    ce = nn.losses.cross_entropy(logits, batch[:, 1:]) * m
    ntoks = m.sum()
    return ce.astype(mx.float32).sum() / ntoks, ntoks


if lora is not None and trainer is not None:
    _train, _evaluate = trainer.train, trainer.evaluate
    lora.train = lambda *a, **k: _train(*a, loss=masked_loss, iterate_batches=masked_iterate_batches, **k)
    lora.evaluate = lambda *a, **k: _evaluate(*a, loss=masked_loss, iterate_batches=masked_iterate_batches, **k)

if __name__ == "__main__":
    if _IMPORT_ERROR is not None:
        raise _IMPORT_ERROR
    sys.argv[0] = "mlx_lm.lora"
    lora.main()
