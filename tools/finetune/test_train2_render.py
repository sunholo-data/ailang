#!/usr/bin/env python3
"""Unit tests for Gemma 4 chat_template patching and span masking in tools/finetune/train2.py."""
import json
import os
import re
import sys
import tempfile
import types
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
if HERE not in sys.path:
    sys.path.insert(0, HERE)

import build_mix5d  # noqa: E402
import extract_lang  # noqa: E402
import train2  # noqa: E402

try:
    import jinja2
except ModuleNotFoundError:
    jinja2 = None


GEMMA4_CHAT_TEMPLATE = (
    "{{- bos_token -}}"
    "{%- set ns_turn = namespace(last_user_idx=-1) -%}"
    "{%- for message in messages -%}"
    "{%- if message['role'] == 'user' -%}{%- set ns_turn.last_user_idx = loop.index0 -%}{%- endif -%}"
    "{%- endfor -%}"
    "{%- for message in messages -%}"
    "{%- if message['role'] in ['system', 'developer'] -%}"
    "{{- '<|turn>system\\n' + (message.get('content') or '') + '<turn|>\\n' -}}"
    "{%- elif message['role'] == 'user' -%}"
    "{{- '<|turn>user\\n' + (message.get('content') or '') + '<turn|>\\n' -}}"
    "{%- elif message['role'] == 'tool' -%}"
    "{{- '<|turn>tool\\n' + (message.get('content') or '') + '<turn|>\\n' -}}"
    "{%- elif message['role'] == 'assistant' -%}"
    "{{- '<|turn>model\\n' -}}"
    "{%- set thinking_text = message.get('reasoning') or message.get('reasoning_content') -%}"
    "{%- if thinking_text and loop.index0 > ns_turn.last_user_idx and message.get('tool_calls') -%}"
    "{{- '<|channel>thought\\n' + thinking_text + '\\n<channel|>' -}}"
    "{%- endif -%}"
    "{%- if message.get('tool_calls') -%}"
    "{%- for tc in message['tool_calls'] -%}"
    "{{- '<|call:' + tc['function']['name'] + '<call|>' -}}"
    "{%- endfor -%}"
    "{%- endif -%}"
    "{%- if message.get('content') -%}{{- message['content'] -}}{%- endif -%}"
    "{{- '<turn|>\\n' -}}"
    "{%- endif -%}"
    "{%- endfor -%}"
    "{%- if add_generation_prompt -%}"
    "{{- '<|turn>model\\n' -}}"
    "{%- if not enable_thinking -%}{{- '<|channel>thought\\n<channel|>' -}}{%- endif -%}"
    "{%- endif -%}"
)

_TOKEN_RE = re.compile(
    r"<bos>|<\|turn>|<turn\|>|<\|channel>thought\n|<channel\|>|<\|call:[^<]+<call\|>|\s+|[^\s<]+|<"
)


class _SimpleVocabTokenizer:
    """Reversible word/tag tokenizer for testing token spans and decode()."""

    def __init__(self, chat_template: str | None):
        self.chat_template = chat_template
        self._tok2id: dict[str, int] = {}
        self._id2tok: list[str] = []

    def encode(self, text: str) -> list[int]:
        ids = []
        for piece in _TOKEN_RE.findall(text):
            tid = self._tok2id.get(piece)
            if tid is None:
                tid = len(self._id2tok)
                self._tok2id[piece] = tid
                self._id2tok.append(piece)
            ids.append(tid)
        return ids

    def decode(self, ids: list[int]) -> str:
        return "".join(self._id2tok[i] for i in ids)


class FakeGemma4Tokenizer(_SimpleVocabTokenizer):
    """Tokenizer holding Gemma 4's exact Jinja template and evaluating it dynamically."""

    def __init__(self, chat_template: str | None = GEMMA4_CHAT_TEMPLATE):
        super().__init__(chat_template)

    def _render_text(self, messages, tools=None, add_generation_prompt=False, enable_thinking=True) -> str:
        if jinja2 is not None and isinstance(self.chat_template, str):
            env = jinja2.Environment()
            tmpl = env.from_string(self.chat_template)
            return tmpl.render(
                bos_token="<bos>",
                messages=messages,
                tools=tools,
                add_generation_prompt=add_generation_prompt,
                enable_thinking=enable_thinking,
            )

        # Pure-stdlib evaluation of self.chat_template's thinking condition so tests
        # fail if ensure_thinking_chat_template does not remove `and message.get('tool_calls')`.
        ct = self.chat_template or ""
        require_tool_calls = (
            "thinking_text and loop.index0 > ns_turn.last_user_idx and message.get('tool_calls')" in ct
        )
        allow_plain_thinking = "thinking_text and loop.index0 > ns_turn.last_user_idx" in ct

        last_user_idx = -1
        for idx, msg in enumerate(messages):
            if msg.get("role") == "user":
                last_user_idx = idx

        out = ["<bos>"]
        if tools:
            # Deterministic tool schema prefix so `tools` is exercised identically across partial/full renders.
            out.append(f"<|tools>{json.dumps(tools, sort_keys=True)}<tools|>")

        for idx, msg in enumerate(messages):
            role = msg.get("role")
            if role in ("system", "developer"):
                out.append(f"<|turn>system\n{msg.get('content') or ''}<turn|>\n")
            elif role == "user":
                out.append(f"<|turn>user\n{msg.get('content') or ''}<turn|>\n")
            elif role == "tool":
                out.append(f"<|turn>tool\n{msg.get('content') or ''}<turn|>\n")
            elif role == "assistant":
                out.append("<|turn>model\n")
                thinking_text = msg.get("reasoning") or msg.get("reasoning_content")
                if enable_thinking and thinking_text and idx > last_user_idx and allow_plain_thinking:
                    if not require_tool_calls or msg.get("tool_calls"):
                        out.append(f"<|channel>thought\n{thinking_text}\n<channel|>")
                if msg.get("tool_calls"):
                    for tc in msg["tool_calls"]:
                        fn = tc.get("function", {})
                        args = fn.get("arguments", {})
                        out.append(f"<|call:{fn.get('name', '')}({json.dumps(args, sort_keys=True)})<call|>")
                if msg.get("content"):
                    out.append(msg["content"])
                out.append("<turn|>\n")

        if add_generation_prompt:
            out.append("<|turn>model\n")
            if not enable_thinking:
                out.append("<|channel>thought\n<channel|>")
        return "".join(out)

    def apply_chat_template(
        self,
        messages,
        tools=None,
        add_generation_prompt=False,
        return_dict=False,
        enable_thinking=True,
        **kwargs,
    ):
        text = self._render_text(
            messages,
            tools=tools,
            add_generation_prompt=add_generation_prompt,
            enable_thinking=enable_thinking,
        )
        return self.encode(text)


class ReadOnlyTemplateWrapper:
    """Mimics mlx_lm's TokenizerWrapper delegating to an inner HF tokenizer."""

    def __init__(self, inner):
        self._tokenizer = inner

    @property
    def chat_template(self):
        return self._tokenizer.chat_template

    def apply_chat_template(self, *args, **kwargs):
        return self._tokenizer.apply_chat_template(*args, **kwargs)

    def decode(self, ids):
        return self._tokenizer.decode(ids)


class BrokenDropThoughtTokenizer(_SimpleVocabTokenizer):
    """Tokenizer whose chat_template drops `<|channel>thought` even after patching."""

    def __init__(self):
        super().__init__("{%- for m in messages -%}{{ m.content }}{%- endfor -%}")

    def apply_chat_template(self, messages, tools=None, add_generation_prompt=False, return_dict=False, **kwargs):
        parts = ["<bos>"]
        for m in messages:
            parts.append(f"<|turn>{m.get('role')}\n{m.get('content') or ''}<turn|>\n")
        if add_generation_prompt:
            parts.append("<|turn>model\n")
        return self.encode("".join(parts))


class TestTrain2Render(unittest.TestCase):
    def test_ensure_thinking_chat_template_idempotent_and_wrapper(self):
        inner = FakeGemma4Tokenizer()
        wrapper = ReadOnlyTemplateWrapper(inner)

        self.assertIn("and message.get('tool_calls')", wrapper.chat_template)
        train2.ensure_thinking_chat_template(wrapper)

        self.assertTrue(getattr(wrapper, "_thinking_template_patched", False))
        self.assertTrue(getattr(inner, "_thinking_template_patched", False))
        self.assertNotIn("and message.get('tool_calls')", inner.chat_template)
        self.assertIn("thinking_text and loop.index0 > ns_turn.last_user_idx", inner.chat_template)

        patched_once = inner.chat_template
        train2.ensure_thinking_chat_template(wrapper)
        self.assertEqual(inner.chat_template, patched_once)

        # Handles None / dict / unrelated template without raising
        self.assertEqual(train2.ensure_thinking_chat_template({"not": "a tokenizer"}), {"not": "a tokenizer"})
        no_ct = _SimpleVocabTokenizer(None)
        train2.ensure_thinking_chat_template(no_ct)
        self.assertTrue(no_ct._thinking_template_patched)

    def test_before_and_after_patch_single_turn_reasoning(self):
        tok = FakeGemma4Tokenizer()
        msgs = [
            {"role": "user", "content": "Write hello world in AILANG."},
            {
                "role": "assistant",
                "reasoning": "Use module main and import std/io (println).",
                "content": "module main\nimport std/io (println)\nexport func main() -> () ! {IO} = println(\"hi\")",
            },
        ]

        # Before patching, Gemma 4's stock template silently drops `reasoning` on non-tool turns.
        raw_before = tok.decode(tok.apply_chat_template(msgs, enable_thinking=True, return_dict=False))
        self.assertNotIn("<|channel>thought\n", raw_before)
        self.assertNotIn("Use module main", raw_before)

        # train2._render auto-patches tok and renders the full thought + content.
        rendered = tok.decode(train2._render(tok, msgs))
        self.assertIn(
            "<|channel>thought\nUse module main and import std/io (println).\n<channel|>"
            "module main\nimport std/io (println)\nexport func main() -> () ! {IO} = println(\"hi\")<turn|>\n",
            rendered,
        )

    def test_single_turn_episode_process_prefix_and_mask(self):
        tok = FakeGemma4Tokenizer()
        sample = {
            "messages": [
                {"role": "user", "content": "Return 42."},
                {
                    "role": "assistant",
                    "reasoning": "Define exported main returning 42.",
                    "content": "module m/ans\nexport func main() -> int = 42",
                },
            ]
        }

        ds = types.SimpleNamespace(tokenizer=tok, chat_key="messages")
        full, mask = train2._episode_process(ds, sample)
        self.assertEqual(len(full), len(mask))

        prompt = train2._render(tok, sample["messages"][:1], gen=True)
        upto = train2._render(tok, sample["messages"][:2])
        self.assertEqual(full[: len(upto)], upto)
        lcp = len(prompt)
        self.assertEqual(full[:lcp], prompt)

        self.assertTrue(all(v == 0 for v in mask[:lcp]))
        self.assertTrue(all(v == 1 for v in mask[lcp : len(upto)]))

        target_text = tok.decode([tid for tid, m in zip(full, mask) if m == 1])
        self.assertEqual(
            target_text,
            "<|channel>thought\nDefine exported main returning 42.\n<channel|>"
            "module m/ans\nexport func main() -> int = 42<turn|>\n",
        )

    def test_multi_turn_episode_weights_tool_calls_and_final_reply(self):
        tok = FakeGemma4Tokenizer()
        tools = [{"type": "function", "function": {"name": "write", "parameters": {}}}]
        msgs = [
            {"role": "developer", "content": "You are an AILANG coding agent."},
            {"role": "user", "content": "Solve task X."},
            {
                "role": "assistant",
                "weight": 0,
                "content": "",
                "reasoning": "Wrong attempt thought that must stay masked.",
                "tool_calls": [{"id": "c1", "type": "function", "function": {"name": "write", "arguments": {"bad": 1}}}],
            },
            {"role": "tool", "tool_call_id": "c1", "name": "write", "content": "TOOL_ERR_MARKER_XYZ"},
            {
                "role": "assistant",
                "content": "",
                "reasoning_content": "Fix the syntax error via write tool.",
                "tool_calls": [{"id": "c2", "type": "function", "function": {"name": "write", "arguments": {"ok": 2}}}],
            },
            {"role": "tool", "tool_call_id": "c2", "name": "write", "content": "TOOL_OK_MARKER_XYZ"},
            {
                "role": "assistant",
                "content": "Solution verified.",
                "reasoning": "All checks passed; reply concisely.",
            },
        ]

        full, mask = train2._episode_process_tokens(tok, msgs, tools=tools)
        unmasked_text = tok.decode([tid for tid, m in zip(full, mask) if m == 1])
        masked_text = tok.decode([tid for tid, m in zip(full, mask) if m == 0])

        # weight == 0 step is completely masked out
        self.assertNotIn("Wrong attempt thought", unmasked_text)
        self.assertIn("Wrong attempt thought", masked_text)

        # tool_calls step (with reasoning_content) is trained
        self.assertIn("<|channel>thought\nFix the syntax error via write tool.\n<channel|>", unmasked_text)
        self.assertIn("<|call:write(", unmasked_text)

        # final non-tool assistant reply is trained (both thought and content)
        self.assertIn("<|channel>thought\nAll checks passed; reply concisely.\n<channel|>Solution verified.<turn|>\n", unmasked_text)

        # developer, user, and tool messages remain masked
        self.assertNotIn("You are an AILANG coding agent.", unmasked_text)
        self.assertNotIn("Solve task X.", unmasked_text)
        self.assertNotIn("TOOL_ERR_MARKER_XYZ", unmasked_text)
        self.assertNotIn("TOOL_OK_MARKER_XYZ", unmasked_text)

    def test_broken_tokenizer_dropping_thought_raises(self):
        broken = BrokenDropThoughtTokenizer()
        ds = types.SimpleNamespace(tokenizer=broken, chat_key="messages")
        sample = {
            "messages": [
                {"role": "user", "content": "Write foo."},
                {"role": "assistant", "reasoning": "Plan foo carefully.", "content": "module foo"},
            ]
        }
        with self.assertRaisesRegex(
            ValueError, "reasoning field was dropped from rendered target span; check chat_template"
        ):
            train2._episode_process(ds, sample)

        # Empty/whitespace reasoning should not raise
        empty_r = {
            "messages": [
                {"role": "user", "content": "Write foo."},
                {"role": "assistant", "reasoning": "   ", "content": "module foo"},
            ]
        }
        full, mask = train2._episode_process(ds, empty_r)
        self.assertTrue(any(mask))
        self.assertEqual(len(full), len(mask))

    def test_build_mix5d_integration_patches_and_verifies_single_turn_thought(self):
        with tempfile.TemporaryDirectory() as td:
            orig_here = extract_lang.HERE
            try:
                held_dir = os.path.join(td, "held")
                os.makedirs(held_dir, exist_ok=True)
                with open(os.path.join(held_dir, "heldout.json"), "w", encoding="utf-8") as f:
                    json.dump({"core": [], "stretch": [], "retired_excluded": []}, f)
                extract_lang.HERE = held_dir

                pdir = os.path.join(td, "prompts")
                os.makedirs(pdir, exist_ok=True)
                with open(os.path.join(pdir, "ft-none.md"), "w", encoding="utf-8") as f:
                    f.write("Write AILANG.")
                with open(os.path.join(pdir, "ft-slim-v1.md"), "w", encoding="utf-8") as f:
                    f.write("AILANG rules.")
                with open(os.path.join(pdir, "ft-compact-v1.md"), "w", encoding="utf-8") as f:
                    f.write("Compact rules.")

                reasoning_45 = " ".join(f"thought{i}" for i in range(45))
                sp = os.path.join(td, "single.jsonl")
                with open(sp, "w", encoding="utf-8") as f:
                    f.write(
                        json.dumps(
                            {
                                "id": "s1",
                                "group": "g1",
                                "kind": "repo",
                                "task": "Compute 1+1",
                                "reasoning": reasoning_45,
                                "code": "module m/s1\nexport func main() -> int = 2",
                            }
                        )
                        + "\n"
                    )

                def _run_with_tok(tok_obj, out_name):
                    mlx_mod = types.ModuleType("mlx_lm")
                    mlx_utils = types.ModuleType("mlx_lm.utils")
                    mlx_utils.load_tokenizer = lambda path: tok_obj
                    mlx_mod.utils = mlx_utils
                    saved = {k: sys.modules.get(k) for k in ("mlx_lm", "mlx_lm.utils")}
                    sys.modules["mlx_lm"] = mlx_mod
                    sys.modules["mlx_lm.utils"] = mlx_utils
                    try:
                        args = build_mix5d.parse_args(
                            [
                                "--single",
                                sp,
                                "--prompts",
                                pdir,
                                "--out",
                                os.path.join(td, out_name),
                                "--tokenizer",
                                td,
                                "--max-tokens",
                                "4096",
                            ]
                        )
                        return build_mix5d.build_mix(args)
                    finally:
                        for k, v in saved.items():
                            if v is None:
                                sys.modules.pop(k, None)
                            else:
                                sys.modules[k] = v

                # Unpatched FakeGemma4Tokenizer is auto-patched via train2._render and passes check
                tok = FakeGemma4Tokenizer()
                rep = _run_with_tok(tok, "out_ok")
                self.assertTrue(getattr(tok, "_thinking_template_patched", False))
                self.assertGreater(rep["exact_tokens"]["train"], 0)

                # Broken tokenizer that drops thought fails loudly in build_mix5d
                with self.assertRaisesRegex(ValueError, "missing '<\\|channel>thought\\\\n'"):
                    _run_with_tok(BrokenDropThoughtTokenizer(), "out_bad")
            finally:
                extract_lang.HERE = orig_here


if __name__ == "__main__":
    unittest.main()
