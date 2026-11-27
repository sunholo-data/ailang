#!/usr/bin/env python3
"""Unit tests for tools/finetune/build_mix5d.py."""
import json
import os
import sys
import tempfile
import types
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", ".."))
if HERE not in sys.path:
    sys.path.insert(0, HERE)

import build_mix5d  # noqa: E402
import extract_lang  # noqa: E402


def _words(n: int, prefix: str = "step") -> str:
    return " ".join(f"{prefix}{i}" for i in range(n))


def _write_jsonl(path: str, rows: list[dict]) -> str:
    with open(path, "w", encoding="utf-8") as f:
        for r in rows:
            f.write(json.dumps(r) + "\n")
    return path


def _make_prompts(d: str, include_compact: bool = False) -> str:
    os.makedirs(d, exist_ok=True)
    with open(os.path.join(d, "ft-none.md"), "w", encoding="utf-8") as f:
        f.write("You are writing code in AILANG.")
    with open(os.path.join(d, "ft-slim-v1.md"), "w", encoding="utf-8") as f:
        f.write("# AILANG TOP DIALECT TRAPS\n1. No xs[0].")
    if include_compact:
        with open(os.path.join(d, "ft-compact-v1.md"), "w", encoding="utf-8") as f:
            f.write("## Compact Reference\nUse `module ...` and `println(show(x))`.")
    return d


def _make_lean_fixtures(d: str) -> tuple[str, str, str]:
    os.makedirs(d, exist_ok=True)
    fmt = os.path.join(d, "pi_format_lean.json")
    slim = os.path.join(d, "arm_slim.json")
    none = os.path.join(d, "arm_none.json")
    with open(fmt, "w", encoding="utf-8") as f:
        json.dump(
            {
                "system": "LeanSystemPrompt\nCurrent working directory: /private/tmp/x\nShell limits: ok",
                "tools": [{"type": "function", "function": {"name": "write", "parameters": {}}}],
            },
            f,
        )
    with open(slim, "w", encoding="utf-8") as f:
        json.dump({"prefix": "SLIM_ARM\n\n"}, f)
    with open(none, "w", encoding="utf-8") as f:
        json.dump({"prefix": "NONE_ARM\n\n"}, f)
    return fmt, slim, none


def _episode(bid: str, kind: str = "solve", n_steps: int = 2, write_chars: int = 100,
             thought_words: int = 10, final_words: int = 10, include_noloss_empty: bool = False) -> dict:
    steps = []
    if include_noloss_empty:
        steps.append(
            {
                "thought": "",
                "no_loss": True,
                "call": {"name": "write", "args": {"path": "benchmark/solution.ail", "content": "bad " * 2000}},
                "result": "error: syntax",
            }
        )
    for i in range(n_steps):
        if i == 0:
            steps.append(
                {
                    "thought": _words(thought_words, "t") if thought_words else "",
                    "call": {"name": "write", "args": {"path": "benchmark/solution.ail", "content": "x" * write_chars}},
                    "result": "ok",
                }
            )
        else:
            steps.append(
                {
                    "thought": _words(thought_words, "r") if thought_words else "",
                    "call": {"name": "bash", "args": {"command": "ailang run benchmark/solution.ail"}},
                    "result": "42",
                }
            )
    return {
        "benchmark": bid,
        "kind": kind,
        "task_prompt": f"Write your complete solution to: /tmp/ws_{bid}/benchmark/solution.ail\nOutput 42.",
        "steps": steps,
        "final": "Done.",
        "final_thought": _words(final_words, "f") if final_words else "",
    }


class TestBuildMix5d(unittest.TestCase):
    def setUp(self):
        self.td = tempfile.TemporaryDirectory()
        self.addCleanup(self.td.cleanup)
        self.orig_here = extract_lang.HERE
        self.orig_repo = extract_lang.REPO
        self.addCleanup(self._restore_extract_lang)

        self.held_dir = os.path.join(self.td.name, "held")
        os.makedirs(self.held_dir, exist_ok=True)
        with open(os.path.join(self.held_dir, "heldout.json"), "w", encoding="utf-8") as f:
            json.dump({"core": ["held_core"], "stretch": ["held_v2"], "retired_excluded": ["held_old"]}, f)
        extract_lang.HERE = self.held_dir

    def _restore_extract_lang(self):
        extract_lang.HERE = self.orig_here
        extract_lang.REPO = self.orig_repo

    def test_compact_prompt_extracted_from_v0_16_20(self):
        pdir = _make_prompts(os.path.join(self.td.name, "prompts"), include_compact=False)
        prompts = build_mix5d.load_prompts(pdir)
        compact = prompts["compact"]
        self.assertIn("## Required Program Structure", compact)
        self.assertIn("## What AILANG Does NOT Have", compact)
        self.assertTrue("## Quick Syntax Reference" in compact or "## Syntax Reference" in compact)
        self.assertGreater(len(compact), 2500)
        self.assertLess(len(compact), 7000)
        # Determinism check
        self.assertEqual(compact, build_mix5d.load_prompts(pdir)["compact"])

    def test_compact_prompt_uses_ft_compact_when_present(self):
        pdir = _make_prompts(os.path.join(self.td.name, "prompts_c"), include_compact=True)
        prompts = build_mix5d.load_prompts(pdir)
        self.assertEqual(prompts["compact"], "## Compact Reference\nUse `module ...` and `println(show(x))`.")

    def test_single_arms_example_repeat_and_reasoning_gate(self):
        pdir = _make_prompts(os.path.join(self.td.name, "prompts"))
        rows = []
        # 40 valid repo items across 20 groups + 10 valid examples + 2 malformed reasoning
        for i in range(40):
            rows.append(
                {
                    "id": f"r{i}",
                    "group": f"g{i // 2}",
                    "kind": "repo",
                    "task": f"Task {i}",
                    "reasoning": _words(45, "plan"),
                    "code": f"module m/r{i}\nexport func main() -> int = {i}",
                }
            )
        for i in range(10):
            rows.append(
                {
                    "id": f"ex{i}",
                    "group": f"exg{i}",
                    "kind": "example",
                    "task": f"Example {i}",
                    "reasoning": _words(50, "ex"),
                    "code": f"module examples/e{i}\nexport func main() -> int = {i}",
                }
            )
        # Empty and too-short reasoning must be dropped
        rows.append({"id": "bad_empty", "group": "g_bad1", "kind": "repo", "task": "Bad1", "reasoning": "", "code": "module m/b1"})
        rows.append({"id": "bad_short", "group": "g_bad2", "kind": "repo", "task": "Bad2", "reasoning": _words(39), "code": "module m/b2"})
        # Held-back split != train must be ignored
        rows.append({"id": "test_split", "group": "g_test", "kind": "repo", "split": "test", "task": "T", "reasoning": _words(50), "code": "module m/t"})

        single_path = _write_jsonl(os.path.join(self.td.name, "single.jsonl"), rows)
        out_dir = os.path.join(self.td.name, "out_single")
        args = build_mix5d.parse_args(["--single", single_path, "--prompts", pdir, "--out", out_dir, "--example-repeat", "2", "--seed", "7"])
        report = build_mix5d.build_mix(args)

        self.assertEqual(report["total_samples"]["train"] + report["total_samples"]["valid"], 40 + 20)
        self.assertEqual(report["dropped"]["short_reasoning:repo"], 2)
        self.assertEqual(report["dropped"]["heldback_test"], 1)
        self.assertGreaterEqual(report["reasoning_words"]["single_turn"]["min"], 40)

        # Verify examples repeated twice with DIFFERENT prompt arms
        all_samples = []
        for fn in ("train.jsonl", "valid.jsonl"):
            with open(os.path.join(out_dir, fn), encoding="utf-8") as f:
                all_samples.extend(json.loads(ln) for ln in f)

        ex0_prompts = [s["messages"][0]["content"] for s in all_samples if "## Task\n\nExample 0" in s["messages"][0]["content"]]
        self.assertEqual(len(ex0_prompts), 2)
        self.assertNotEqual(ex0_prompts[0], ex0_prompts[1])

        # Verify all 3 prompt arms appeared across the 60 samples
        sample_keys = set(report["samples"].keys())
        self.assertTrue(any(k.endswith("-none") for k in sample_keys))
        self.assertTrue(any(k.endswith("-slim") for k in sample_keys))
        self.assertTrue(any(k.endswith("-compact") for k in sample_keys))

    def test_joint_episodes_repairs_gates_and_leakage_free_split(self):
        pdir = _make_prompts(os.path.join(self.td.name, "prompts"))
        fmt, slim, none = _make_lean_fixtures(os.path.join(self.td.name, "traces"))

        # Single items sharing bench:b0 .. bench:b19 with episodes and repairs
        single_rows = [
            {
                "hash": f"h{i}",
                "group": f"bench:b{i}",
                "benchmark": f"b{i}",
                "kind": "bench",
                "task": f"Bench task {i}",
                "reasoning": _words(42, "s"),
                "code": f"module benchmark/solution\nexport func main() -> int = {i}",
            }
            for i in range(20)
        ]
        repair_rows = [
            {
                "benchmark": f"b{i}",
                "code": f"module benchmark/solution\nexport func main() = x{i}",
                "error": "undefined variable",
                "fixed": f"module benchmark/solution\nexport func main() -> int = {i}",
                "reasoning": _words(45, "fix"),
            }
            for i in range(10)
        ]
        # Malformed repair (< 40 words)
        repair_rows.append(
            {
                "benchmark": "b0",
                "code": "bad",
                "error": "err",
                "fixed": "good",
                "reasoning": _words(15, "shortfix"),
            }
        )

        ep_rows = [_episode(f"b{i}", kind="solve", include_noloss_empty=(i == 0)) for i in range(10)]
        # Bad episodes: too many steps, too long write, short step reasoning, short final reasoning
        ep_rows.append(_episode("b11", kind="solve", n_steps=21))
        ep_rows.append(_episode("b12", kind="solve", write_chars=6600))
        ep_rows.append(_episode("b13", kind="solve", thought_words=5))
        ep_rows.append(_episode("b14", kind="solve", final_words=0))

        sp = _write_jsonl(os.path.join(self.td.name, "single.jsonl"), single_rows)
        rp = _write_jsonl(os.path.join(self.td.name, "repairs.jsonl"), repair_rows)
        ep = _write_jsonl(os.path.join(self.td.name, "episodes.jsonl"), ep_rows)
        out_dir = os.path.join(self.td.name, "out_joint")

        args = build_mix5d.parse_args(
            [
                "--single", sp,
                "--repairs", rp,
                "--episodes", ep,
                "--prompts", pdir,
                "--lean-format", fmt,
                "--arm-slim", slim,
                "--arm-none", none,
                "--agent-cap", "8",
                "--max-steps", "20",
                "--max-write-chars", "6500",
                "--out", out_dir,
                "--seed", "42",
            ]
        )
        report = build_mix5d.build_mix(args)

        self.assertEqual(report["dropped"]["short_reasoning:repair"], 1)
        self.assertEqual(report["dropped"]["too_long:ep-solve"], 1)
        self.assertEqual(report["dropped"]["long_write:ep-solve"], 1)
        self.assertEqual(report["dropped"]["short_reasoning:ep-solve"], 2)
        self.assertEqual(report["dropped"]["agent_cap:ep-solve"], 2)
        self.assertGreaterEqual(report["reasoning_words"]["multi_turn"]["min"], 8)
        self.assertGreaterEqual(report["reasoning_words"]["single_turn"]["min"], 40)

        # Verify no benchmark straddles train and valid, and check repair format + reasoning invariant
        def _benches_in(path: str) -> set[str]:
            found = set()
            with open(path, encoding="utf-8") as f:
                for ln in f:
                    d = json.loads(ln)
                    msgs = d["messages"]
                    is_ep = len(msgs) > 2
                    for m in msgs:
                        if m.get("role") == "assistant" and m.get("weight", 1) != 0:
                            min_w = 8 if is_ep else 40
                            self.assertGreaterEqual(len(m["reasoning"].split()), min_w)
                    txt = msgs[0]["content"] + "\n" + msgs[1]["content"]
                    for i in range(20):
                        if f"ws_b{i}/" in txt or f"Bench task {i}" in txt or f"func main() = x{i}" in txt:
                            found.add(f"b{i}")
            return found

        train_b = _benches_in(os.path.join(out_dir, "train.jsonl"))
        valid_b = _benches_in(os.path.join(out_dir, "valid.jsonl"))
        self.assertTrue(train_b)
        self.assertTrue(valid_b)
        self.assertEqual(train_b & valid_b, set())

    def test_refuses_heldout_including_retired(self):
        pdir = _make_prompts(os.path.join(self.td.name, "prompts"))
        for bad_bid, source_kind in [("held_core", "single"), ("held_v2", "repair"), ("held_old", "episode")]:
            with self.subTest(bad_bid=bad_bid, source_kind=source_kind):
                out_dir = os.path.join(self.td.name, f"out_{bad_bid}")
                if source_kind == "single":
                    p = _write_jsonl(
                        os.path.join(self.td.name, f"s_{bad_bid}.jsonl"),
                        [{"id": "x", "group": f"bench:{bad_bid}", "kind": "bench", "task": "t", "reasoning": _words(45), "code": "c"}],
                    )
                    argv = ["--single", p, "--prompts", pdir, "--out", out_dir]
                elif source_kind == "repair":
                    p = _write_jsonl(
                        os.path.join(self.td.name, f"r_{bad_bid}.jsonl"),
                        [{"benchmark": bad_bid, "code": "c", "error": "e", "fixed": "f", "reasoning": _words(45)}],
                    )
                    argv = ["--repairs", p, "--prompts", pdir, "--out", out_dir]
                else:
                    fmt, slim, none = _make_lean_fixtures(os.path.join(self.td.name, "traces_h"))
                    p = _write_jsonl(os.path.join(self.td.name, f"e_{bad_bid}.jsonl"), [_episode(bad_bid)])
                    argv = ["--episodes", p, "--lean-format", fmt, "--arm-slim", slim, "--arm-none", none, "--out", out_dir]
                with self.assertRaisesRegex(ValueError, "held-out benchmark"):
                    build_mix5d.build_mix(build_mix5d.parse_args(argv))

    def test_tokenizer_max_tokens_and_max_chars_fallback(self):
        pdir = _make_prompts(os.path.join(self.td.name, "prompts"))
        rows = [
            {"id": "short", "group": "g1", "kind": "repo", "task": "Short task", "reasoning": _words(45), "code": "module m/s\nexport func main() -> int = 1"},
            {"id": "long", "group": "g2", "kind": "repo", "task": "Long " + ("x" * 2000), "reasoning": _words(45), "code": "module m/l\nexport func main() -> int = 2"},
        ]
        sp = _write_jsonl(os.path.join(self.td.name, "tok_single.jsonl"), rows)

        # Fallback --max-chars (no tokenizer)
        out_chars = os.path.join(self.td.name, "out_chars")
        rep_chars = build_mix5d.build_mix(
            build_mix5d.parse_args(["--single", sp, "--prompts", pdir, "--out", out_chars, "--max-chars", "1500"])
        )
        self.assertIsNone(rep_chars["exact_tokens"])
        self.assertEqual(rep_chars["dropped"]["too_long:repo"], 1)

        # Exact tokenizer check using stubbed mlx_lm.utils + train2._render
        mlx_mod = types.ModuleType("mlx_lm")
        mlx_utils = types.ModuleType("mlx_lm.utils")
        mlx_utils.load_tokenizer = lambda path: {"path": str(path)}
        mlx_mod.utils = mlx_utils
        train2_stub = types.ModuleType("train2")
        train2_stub._render = lambda tok, msgs, tools=None: list(
            range(sum(len((m.get("content") or "") + (m.get("reasoning") or "")) for m in msgs))
        )
        saved = {k: sys.modules.get(k) for k in ("mlx_lm", "mlx_lm.utils", "train2")}
        sys.modules["mlx_lm"] = mlx_mod
        sys.modules["mlx_lm.utils"] = mlx_utils
        sys.modules["train2"] = train2_stub
        try:
            out_tok = os.path.join(self.td.name, "out_tok")
            rep_tok = build_mix5d.build_mix(
                build_mix5d.parse_args(
                    ["--single", sp, "--prompts", pdir, "--out", out_tok, "--tokenizer", self.td.name, "--max-tokens", "1500"]
                )
            )
        finally:
            for k, v in saved.items():
                if v is None:
                    sys.modules.pop(k, None)
                else:
                    sys.modules[k] = v

        self.assertIsNotNone(rep_tok["exact_tokens"])
        self.assertEqual(rep_tok["dropped"]["over_max_tokens:repo"], 1)
        self.assertEqual(rep_tok["total_samples"]["train"] + rep_tok["total_samples"]["valid"], 1)

    def test_agent_repair_frac_renders_lean_pi_tool_repairs(self):
        pdir = _make_prompts(os.path.join(self.td.name, "prompts"))
        fmt, slim, none = _make_lean_fixtures(os.path.join(self.td.name, "traces_ar"))
        repair_rows = [
            {
                "benchmark": f"b{i}",
                "code": f"module benchmark/solution\nexport func main() = bad{i}",
                "error": "Error: undefined variable: bad",
                "fixed": f"module benchmark/solution\nexport func main() -> int = {i}",
                "reasoning": _words(55, "diag"),
            }
            for i in range(12)
        ]
        ep_rows = [_episode(f"b{i}", kind="solve") for i in range(6)]
        rp = _write_jsonl(os.path.join(self.td.name, "repairs_ar.jsonl"), repair_rows)
        ep = _write_jsonl(os.path.join(self.td.name, "eps_ar.jsonl"), ep_rows)
        out_dir = os.path.join(self.td.name, "out_ar")

        args = build_mix5d.parse_args(
            [
                "--repairs", rp,
                "--episodes", ep,
                "--prompts", pdir,
                "--lean-format", fmt,
                "--arm-slim", slim,
                "--arm-none", none,
                "--agent-repair-frac", "1.0",
                "--out", out_dir,
                "--seed", "9",
            ]
        )
        report = build_mix5d.build_mix(args)
        sample_keys = set(report["samples"].keys())
        self.assertTrue(any("ep-repair-direct-" in k for k in sample_keys))
        self.assertFalse(any(":repair-" in k for k in sample_keys))

        direct_found = 0
        for fn in ("train.jsonl", "valid.jsonl"):
            with open(os.path.join(out_dir, fn), encoding="utf-8") as f:
                for ln in f:
                    d = json.loads(ln)
                    self.assertIn("tools", d)
                    msgs = d["messages"]
                    if len(msgs) == 9 and msgs[-1].get("tool_calls", [{}])[0].get("function", {}).get("name") == "write":
                        direct_found += 1
                        # Broken code and check steps must be weight=0; only final write(fixed) has loss
                        weighted = [m for m in msgs if m.get("role") == "assistant" and m.get("weight", 1) != 0]
                        self.assertEqual(len(weighted), 1)
                        self.assertEqual(len(weighted[0]["reasoning"].split()), 55)
                        self.assertIn("export func main() -> int =", weighted[0]["tool_calls"][0]["function"]["arguments"]["content"])
        self.assertEqual(direct_found, 12)


if __name__ == "__main__":
    unittest.main()
