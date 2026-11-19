#!/usr/bin/env python3
"""mlx-lm LoRA training wrapper with inference-aligned prompt masking.

Why this exists (measured 2026-11-05): Gemma 4's chat template renders the
generation prompt as `<|turn>model\\n<|channel>thought\\n<channel|>` (an empty
thinking block when thinking is off), but renders a finished assistant turn
WITHOUT that block. mlx-lm's ChatDataset computes the --mask-prompt offset from
the generation-prompt rendering, so on 1,497/2,193 pilot samples the mask covered
the first 4 target tokens, including `<|tool_call>`, the very behaviour being
trained.

This wrapper trains on exactly what inference will see:
    tokens = render(messages[:-1], add_generation_prompt=True) + target
where target is the part of the full rendering after its longest common prefix
with that prompt, and the loss mask ends at the prompt. Everything else is stock
`mlx_lm.lora` (same CLI flags).

Usage: python tools/finetune/train.py <mlx_lm.lora flags...>
"""
import sys

from mlx_lm import lora
from mlx_lm.tuner import datasets


def _aligned_process(self, d):
    messages, tools = d[self.chat_key], d.get("tools", None)
    full = self.tokenizer.apply_chat_template(messages, tools=tools, return_dict=False)
    prompt = self.tokenizer.apply_chat_template(messages[:-1], tools=tools,
                                                add_generation_prompt=True, return_dict=False)
    lcp = 0
    for a, b in zip(full, prompt):
        if a != b:
            break
        lcp += 1
    tokens = list(prompt) + list(full[lcp:])
    return (tokens, len(prompt)) if self.mask_prompt else (tokens, 0)


datasets.ChatDataset.process = _aligned_process

if __name__ == "__main__":
    sys.argv[0] = "mlx_lm.lora"
    lora.main()
