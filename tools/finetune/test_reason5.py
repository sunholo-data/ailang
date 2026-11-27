#!/usr/bin/env python3
"""Unit tests for tools/finetune/reason5.py (sandbox-safe, all CLIs mocked)."""
import hashlib
import json
import os
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

HERE = os.path.dirname(os.path.abspath(__file__))
if HERE not in sys.path:
    sys.path.insert(0, HERE)

import reason5  # noqa: E402


def make_valid_reasoning(word_count: int = 140, extra: str = "") -> str:
    base = (
        "I need to read the task specification carefully and derive the input and output contract. "
        "First, I will define an algebraic data type or record structure to represent the core domain state cleanly, "
        "keeping all helper functions pure and using explicit pattern matching with recursion base and step cases. "
        "Since AILANG is purely functional, every transformation will recurse over lists or trees directly without "
        "mutable state. Let us check which standard library functions and explicit module imports are required, and "
        "make sure the entry point declares the appropriate effect annotations such as IO when printing output. "
        "Edge cases include empty inputs, single-element lists, negative numbers, and boundary values. "
        "I will structure the helper functions before main so type inference and module checking succeed cleanly."
    )
    words = base.split()
    out = []
    while len(out) < word_count:
        out.extend(words)
    text = " ".join(out[:word_count])
    if extra:
        text = f"{text} {extra}"
    return text


SAMPLE_CODE = """module benchmark/solution
import std/list (map, foldl)

pure func sumDoubled(xs: [int]) -> int =
  foldl(\\acc x. acc + x * 2, 0, xs)

export func main() -> () ! {IO} =
  println(show(sumDoubled([1, 2, 3])))
"""


class TestHeldoutGate(unittest.TestCase):
    def test_real_heldout_loaded_and_variant_matching(self):
        held = reason5.load_heldout()
        self.assertIn("merge_sort", held)
        self.assertIn("run_length_encode_v2", held)
        self.assertIn("run_length_encode", held)

        self.assertTrue(reason5.is_heldout("merge_sort", held))
        self.assertTrue(reason5.is_heldout("merge_sort_v2", held))
        self.assertTrue(reason5.is_heldout("merge_sort_v12", held))
        self.assertTrue(reason5.is_heldout("run_length_encode_v3", held))
        self.assertTrue(reason5.is_heldout("bench:pipeline", held))

        self.assertFalse(reason5.is_heldout("fizzbuzz", held))
        self.assertFalse(reason5.is_heldout("emit_exact_bytes_varied", held))
        self.assertFalse(reason5.is_heldout("", held))
        self.assertFalse(reason5.is_heldout(None, held))


class TestValidateReasoning(unittest.TestCase):
    def test_valid_reasoning_passes(self):
        ok, why = reason5.validate_reasoning(make_valid_reasoning(150), SAMPLE_CODE)
        self.assertTrue(ok, why)
        self.assertEqual(why, "")

    def test_word_count_bounds(self):
        ok, why = reason5.validate_reasoning("short plan only ten words here total", SAMPLE_CODE)
        self.assertFalse(ok)
        self.assertEqual(why, "too_short")

        ok, why = reason5.validate_reasoning(make_valid_reasoning(605), SAMPLE_CODE)
        self.assertFalse(ok)
        self.assertEqual(why, "too_long")

    def test_control_tokens_rejected(self):
        for tok in ("<|channel>", "<channel|>", "<|tool_call>", "<|turn>"):
            ok, why = reason5.validate_reasoning(make_valid_reasoning(120, extra=tok), SAMPLE_CODE)
            self.assertFalse(ok, tok)
            self.assertEqual(why, "control_token")

    def test_code_fence_rejected(self):
        ok, why = reason5.validate_reasoning(
            make_valid_reasoning(120, extra="```ailang\nfunc f() = 1\n```"), SAMPLE_CODE
        )
        self.assertFalse(ok)
        self.assertEqual(why, "code_fence")

    def test_meta_leakage_rejected(self):
        for phrase in ("the given program", "in the provided code", "solution above", "looking at the code"):
            ok, why = reason5.validate_reasoning(make_valid_reasoning(120, extra=phrase), SAMPLE_CODE)
            self.assertFalse(ok, phrase)
            self.assertEqual(why, "meta_leakage")

    def test_banned_syntax_rejected(self):
        for phrase in ("use a for loop here", "implement a while loop", "update a mutable variable"):
            ok, why = reason5.validate_reasoning(make_valid_reasoning(120, extra=phrase), SAMPLE_CODE)
            self.assertFalse(ok, phrase)
            self.assertEqual(why, "banned_syntax")

    def test_hallucinated_import_rejected(self):
        ok, why = reason5.validate_reasoning(
            make_valid_reasoning(120, extra="I should import std/json to parse input."), SAMPLE_CODE
        )
        self.assertFalse(ok)
        self.assertEqual(why, "hallucinated_import")

        # std/io mentioned alongside println in code is allowed; std/list is in SAMPLE_CODE
        ok2, why2 = reason5.validate_reasoning(
            make_valid_reasoning(120, extra="Use std/list and std/io println."), SAMPLE_CODE
        )
        self.assertTrue(ok2, why2)


class TestBackends(unittest.TestCase):
    def test_agy_backend_env_flags_and_parsing(self):
        captured = {}

        def fake_run(cmd, **kwargs):
            captured["cmd"] = cmd
            captured["cwd"] = kwargs.get("cwd")
            captured["env"] = kwargs.get("env")
            captured["stdin"] = kwargs.get("stdin")
            inner = json.dumps({"items": [{"id": "abc123", "reasoning": make_valid_reasoning(130)}]})
            outer = json.dumps({"status": "SUCCESS", "response": f"```json\n{inner}\n```"})
            return subprocess.CompletedProcess(cmd, 0, stdout=outer, stderr="")

        fake_env = {
            "PATH": os.environ.get("PATH", ""),
            "GEMINI_API_KEY": "secret-gemini",
            "GOOGLE_API_KEY": "secret-google",
            "AGY_ADC_AUTH": "1",
            "GOOGLE_CLOUD_PROJECT": "billed-proj",
            "AILANG_AGY_SUB_HOME": "/tmp/sub-home-test",
        }
        with mock.patch.dict(os.environ, fake_env, clear=True), \
                mock.patch.object(reason5.os.path, "exists", return_value=False), \
                mock.patch.object(reason5.subprocess, "run", side_effect=fake_run):
            res = reason5.call_agy("gemini-4-neon-medium", "prompt text")

        self.assertIn("abc123", res)
        self.assertGreaterEqual(len(res["abc123"].split()), 120)
        cmd = captured["cmd"]
        self.assertEqual(cmd[0], "agy")
        self.assertIn("--mode", cmd)
        self.assertEqual(cmd[cmd.index("--mode") + 1], "plan")
        self.assertIn("--disable-slash-commands", cmd)
        self.assertIn("--output-format", cmd)
        self.assertEqual(cmd[cmd.index("--output-format") + 1], "json")
        self.assertIn("--json-schema", cmd)
        self.assertIn("--print-timeout", cmd)
        self.assertEqual(cmd[cmd.index("--print-timeout") + 1], "300s")
        self.assertIn("-p", cmd)

        env = captured["env"]
        for stripped in ("GEMINI_API_KEY", "GOOGLE_API_KEY", "AGY_ADC_AUTH", "GOOGLE_CLOUD_PROJECT"):
            self.assertNotIn(stripped, env)
        self.assertEqual(env["HOME"], "/tmp/sub-home-test")
        self.assertEqual(env["AGY_CLI_DISABLE_AUTO_UPDATE"], "true")
        self.assertEqual(captured["stdin"], subprocess.DEVNULL)
        self.assertNotEqual(captured["cwd"], os.getcwd())

    def test_claude_cli_backend_parses_json(self):
        payload = json.dumps({"items": [{"id": "id1", "reasoning": make_valid_reasoning(140)}]})
        with mock.patch.object(reason5.stale_fix, "ask_claude", return_value=(payload, {})) as m:
            res = reason5.call_claude("claude-sonnet-4-6", "test prompt")
        m.assert_called_once_with("claude-sonnet-4-6", "test prompt")
        self.assertIn("id1", res)
        self.assertEqual(len(res["id1"].split()), 140)


class TestSingleMode(unittest.TestCase):
    def test_single_mode_filters_merges_examples_batches_and_resumes(self):
        with tempfile.TemporaryDirectory() as td:
            desc_path = os.path.join(td, "desc.jsonl")
            eval_path = os.path.join(td, "eval.jsonl")
            ex_dir = os.path.join(td, "examples")
            os.makedirs(os.path.join(ex_dir, "sub"), exist_ok=True)
            out_path = os.path.join(td, "out.jsonl")

            # desc.jsonl: 2 train (1 heldout group), 1 test, 1 valid
            desc_rows = [
                {
                    "hash": "train1",
                    "path": "pkg/a.ail",
                    "group": "pkg:a",
                    "split": "train",
                    "lines": 8,
                    "task": "Write sumDoubled on a list.",
                    "code": SAMPLE_CODE,
                },
                {
                    "hash": "held_desc",
                    "path": "pkg/ms.ail",
                    "group": "bench:merge_sort",
                    "split": "train",
                    "lines": 5,
                    "task": "Sort a list.",
                    "code": "module benchmark/solution\nexport func main() -> () = ()\n",
                },
                {
                    "hash": "test1",
                    "path": "pkg/test.ail",
                    "group": "pkg:test",
                    "split": "test",
                    "lines": 5,
                    "task": "Held back test item.",
                    "code": "module pkg/test\nexport func main() -> () = ()\n",
                },
                {
                    "hash": "valid1",
                    "path": "pkg/valid.ail",
                    "group": "pkg:valid",
                    "split": "valid",
                    "lines": 5,
                    "task": "Valid split item.",
                    "code": "module pkg/valid\nexport func main() -> () = ()\n",
                },
            ]
            with open(desc_path, "w") as f:
                for r in desc_rows:
                    f.write(json.dumps(r) + "\n")

            # eval.jsonl: 1 heldout (_v2 variant), 3 non-heldout for fizzbuzz with --eval-cap 2 (preferring distinct models)
            eval_rows = [
                {
                    "source": "eval",
                    "benchmark": "merge_sort_v2",
                    "model": "m1",
                    "task": "Merge sort",
                    "code": "module benchmark/solution\nexport func main() -> () = ()\n",
                },
                {
                    "source": "eval",
                    "benchmark": "fizzbuzz",
                    "model": "m1",
                    "task": "Print fizzbuzz 1..15",
                    "code": "module benchmark/solution\nexport func main() -> () ! {IO} = println(\"1\")\n",
                },
                {
                    "source": "eval",
                    "benchmark": "fizzbuzz",
                    "model": "m1",
                    "task": "Print fizzbuzz 1..15",
                    "code": "module benchmark/solution\nexport func main() -> () ! {IO} = println(\"2\")\n",
                },
                {
                    "source": "eval",
                    "benchmark": "fizzbuzz",
                    "model": "m2",
                    "task": "Print fizzbuzz 1..15",
                    "code": "module benchmark/solution\nexport func main() -> () ! {IO} = println(\"3\")\n",
                },
            ]
            with open(eval_path, "w") as f:
                for r in eval_rows:
                    f.write(json.dumps(r) + "\n")

            # examples: 1 with header comment, 1 with module+exports only, 1 empty/no-meta skipped
            with open(os.path.join(ex_dir, "header.ail"), "w") as f:
                f.write("-- Compute factorial recursively\nmodule examples/header\nexport func fact(n: int) -> int = 1\n")
            with open(os.path.join(ex_dir, "sub", "no_header.ail"), "w") as f:
                f.write("module examples/sub/no_header\nexport func compute() -> int = 42\n")
            with open(os.path.join(ex_dir, "bare.ail"), "w") as f:
                f.write("1 + 2\n")

            batches_seen = []

            def fake_backend(backend, model, items, mode):
                batches_seen.append(([it["id"] for it in items], mode))
                return {it["id"]: make_valid_reasoning(140) for it in items}

            with mock.patch.object(reason5, "call_backend", side_effect=fake_backend):
                rc = reason5.main(
                    [
                        "--mode",
                        "single",
                        "--in",
                        desc_path,
                        "--out",
                        out_path,
                        "--eval",
                        eval_path,
                        "--eval-cap",
                        "2",
                        "--examples-dir",
                        ex_dir,
                        "--batch-size",
                        "2",
                        "--workers",
                        "1",
                        "--seed",
                        "7",
                    ]
                )
            self.assertEqual(rc, 0)

            with open(out_path) as f:
                out_recs = [json.loads(ln) for ln in f]
            # Expected: train1 (1) + fizzbuzz capped at 2 (m1 and m2 distinct models!) + 2 examples = 5
            self.assertEqual(len(out_recs), 5)
            # 5 items at batch-size 2 => 3 batches (2, 2, 1)
            self.assertEqual(sorted(len(b[0]) for b in batches_seen), [1, 2, 2])
            self.assertTrue(all(b[1] == "single" for b in batches_seen))

            bench_recs = [r for r in out_recs if r.get("kind") == "bench"]
            self.assertEqual(len(bench_recs), 2)
            self.assertEqual({r["model"] for r in bench_recs}, {"m1", "m2"})
            self.assertTrue(all(r["benchmark"] == "fizzbuzz" for r in bench_recs))

            ex_recs = [r for r in out_recs if r.get("kind") == "example"]
            self.assertEqual({r["group"] for r in ex_recs}, {"example:header.ail", "example:sub/no_header.ail"})

            # Resume test: running again should skip all 5 and make 0 backend calls
            batches_seen.clear()
            with mock.patch.object(reason5, "call_backend", side_effect=fake_backend):
                reason5.main(
                    [
                        "--mode",
                        "single",
                        "--in",
                        desc_path,
                        "--out",
                        out_path,
                        "--eval",
                        eval_path,
                        "--eval-cap",
                        "2",
                        "--examples-dir",
                        ex_dir,
                    ]
                )
            self.assertEqual(batches_seen, [])
            with open(out_path) as f:
                self.assertEqual(len(f.readlines()), 5)

    def test_single_mode_rejects_bad_reasoning_and_retries_invalid_out(self):
        with tempfile.TemporaryDirectory() as td:
            desc_path = os.path.join(td, "desc.jsonl")
            out_path = os.path.join(td, "out.jsonl")
            with open(desc_path, "w") as f:
                f.write(json.dumps({"hash": "h1", "group": "g1", "split": "train", "task": "t1", "code": SAMPLE_CODE}) + "\n")
                f.write(json.dumps({"hash": "h2", "group": "g2", "split": "train", "task": "t2", "code": SAMPLE_CODE}) + "\n")

            # Seed out_path with h1 having invalid (too short) reasoning: resume must NOT skip h1
            with open(out_path, "w") as f:
                f.write(json.dumps({"id": "h1", "hash": "h1", "code": SAMPLE_CODE, "reasoning": "too short"}) + "\n")

            # Backend returns invalid fence for h1 and valid reasoning for h2
            def fake_backend(backend, model, items, mode):
                return {
                    "h1": make_valid_reasoning(130, extra="```ailang\nbad\n```"),
                    "h2": make_valid_reasoning(130),
                }

            with mock.patch.object(reason5, "call_backend", side_effect=fake_backend):
                reason5.main(["--mode", "single", "--in", desc_path, "--out", out_path, "--workers", "1"])

            with open(out_path) as f:
                rows = [json.loads(ln) for ln in f]
            # Original bad row + newly accepted h2 row (rejected h1 not written)
            self.assertEqual([r["id"] for r in rows], ["h1", "h2"])

    def test_single_mode_limit(self):
        with tempfile.TemporaryDirectory() as td:
            desc_path = os.path.join(td, "desc.jsonl")
            out_path = os.path.join(td, "out.jsonl")
            with open(desc_path, "w") as f:
                for i in range(5):
                    f.write(json.dumps({"hash": f"h{i}", "group": f"g{i}", "split": "train", "task": f"t{i}", "code": SAMPLE_CODE}) + "\n")

            with mock.patch.object(
                reason5,
                "call_backend",
                side_effect=lambda b, m, items, mode: {it["id"]: make_valid_reasoning(130) for it in items},
            ):
                reason5.main(["--mode", "single", "--in", desc_path, "--out", out_path, "--limit", "3", "--workers", "1"])
            with open(out_path) as f:
                self.assertEqual(len(f.readlines()), 3)

    def test_verify_compile_and_include_valid(self):
        with tempfile.TemporaryDirectory() as td:
            desc_path = os.path.join(td, "desc.jsonl")
            eval_path = os.path.join(td, "eval.jsonl")
            out_path = os.path.join(td, "out.jsonl")
            with open(desc_path, "w") as f:
                f.write(json.dumps({"hash": "v1", "group": "g", "split": "valid", "task": "t", "code": SAMPLE_CODE}) + "\n")
                f.write(json.dumps({"hash": "t1", "group": "g", "split": "test", "task": "t", "code": SAMPLE_CODE}) + "\n")
            with open(eval_path, "w") as f:
                f.write(json.dumps({"benchmark": "fizzbuzz", "model": "m1", "task": "good", "code": SAMPLE_CODE}) + "\n")
                f.write(json.dumps({"benchmark": "fizzbuzz", "model": "m2", "task": "bad", "code": "broken"}) + "\n")

            def fake_verify(code):
                return ("broken" not in code, "err" if "broken" in code else "")

            with mock.patch.object(reason5, "verify_compile", side_effect=fake_verify), \
                    mock.patch.object(
                        reason5,
                        "call_backend",
                        side_effect=lambda b, m, items, mode: {it["id"]: make_valid_reasoning(140) for it in items},
                    ):
                reason5.main(
                    [
                        "--mode",
                        "single",
                        "--in",
                        desc_path,
                        "--out",
                        out_path,
                        "--eval",
                        eval_path,
                        "--include-valid",
                        "--verify-compile",
                        "--workers",
                        "1",
                    ]
                )
            with open(out_path) as f:
                out_recs = [json.loads(ln) for ln in f]
            self.assertEqual(len(out_recs), 2)  # v1 (valid) + good eval; test1 and bad eval excluded


class TestEpisodesMode(unittest.TestCase):
    def test_episodes_dedup_heldout_preserve_tactical_and_resume(self):
        with tempfile.TemporaryDirectory() as td:
            in_path = os.path.join(td, "episodes_thought.jsonl")
            out_path = os.path.join(td, "episodes_reason5.jsonl")

            ep_held = {
                "kind": "solve",
                "benchmark": "graph_bfs_v2",
                "task_prompt": "Solve graph BFS",
                "steps": [
                    {"call": {"name": "write", "args": {"path": "x", "content": SAMPLE_CODE}}, "result": "ok", "thought": "old"}
                ],
                "final": "Done",
                "final_thought": "Done",
            }
            ep_solve = {
                "kind": "solve",
                "benchmark": "fizzbuzz",
                "task_prompt": "Solve fizzbuzz",
                "steps": [
                    {"call": {"name": "read", "args": {"path": "/ws/benchmark/solution.ail"}}, "result": "tmpl", "thought": "Read template first."},
                    {"call": {"name": "bash", "args": {"command": "ailang docs std/list.foldl"}}, "result": "sig", "habit": "lookup", "lookup": "std/list.foldl", "thought": "Look up foldl."},
                    {"call": {"name": "write", "args": {"path": "/ws/benchmark/solution.ail", "content": SAMPLE_CODE}}, "result": "ok", "thought": "25-word template"},
                    {"call": {"name": "bash", "args": {"command": "ailang check benchmark/solution.ail"}}, "result": "", "habit": "check", "thought": "Type-check it."},
                    {"call": {"name": "bash", "args": {"command": "ailang run --caps IO benchmark/solution.ail"}}, "result": "12\n", "thought": "Run it."},
                ],
                "final": "Done.",
                "final_thought": "Output matches exactly.",
            }
            # Duplicate of ep_solve (same benchmark + first_write_code, different workspace)
            ep_dup = json.loads(json.dumps(ep_solve))
            ep_dup["steps"][0]["call"]["args"]["path"] = "/ws2/benchmark/solution.ail"

            ep_build = {
                "kind": "build",
                "benchmark": "fizzbuzz",
                "task_prompt": "Solve fizzbuzz in steps",
                "steps": [
                    {"call": {"name": "read", "args": {"path": "/ws/benchmark/solution.ail"}}, "result": "tmpl", "thought": "Read first."},
                    {"call": {"name": "write", "args": {"path": "/ws/benchmark/solution.ail", "content": "module benchmark/solution\nexport func main() -> () = ()\n"}}, "result": "ok", "habit": "build", "thought": "Build part1."},
                    {"call": {"name": "edit", "args": {"path": "/ws/benchmark/solution.ail", "edits": []}}, "result": "ok", "habit": "build_rest", "thought": "Fill main."},
                ],
                "final": "Done.",
                "final_thought": "Done.",
            }

            with open(in_path, "w") as f:
                for ep in (ep_held, ep_solve, ep_dup, ep_build):
                    f.write(json.dumps(ep) + "\n")

            called_ids = []

            def fake_backend(backend, model, items, mode):
                self.assertEqual(mode, "episodes")
                for it in items:
                    called_ids.append(it["id"])
                return {it["id"]: make_valid_reasoning(160) for it in items}

            with mock.patch.object(reason5, "call_backend", side_effect=fake_backend):
                rc = reason5.main(
                    [
                        "--mode",
                        "episodes",
                        "--in",
                        in_path,
                        "--out",
                        out_path,
                        "--batch-size",
                        "8",
                        "--workers",
                        "1",
                    ]
                )
            self.assertEqual(rc, 0)

            expected_solve_id = hashlib.sha256(f"fizzbuzz:{SAMPLE_CODE}".encode()).hexdigest()[:16]
            self.assertEqual(len(called_ids), 2)  # heldout skipped, duplicate deduped, solve + build kept
            self.assertIn(expected_solve_id, called_ids)

            with open(out_path) as f:
                out_eps = [json.loads(ln) for ln in f]
            self.assertEqual(len(out_eps), 2)
            by_kind = {e["kind"]: e for e in out_eps}

            solve_out = by_kind["solve"]
            self.assertEqual(solve_out["episode_id"], expected_solve_id)
            self.assertEqual(solve_out["steps"][0]["thought"], "Read template first.")
            self.assertEqual(solve_out["steps"][1]["thought"], "Look up foldl.")
            self.assertEqual(len(solve_out["steps"][2]["thought"].split()), 160)
            self.assertEqual(solve_out["steps"][3]["thought"], "Type-check it.")
            self.assertEqual(solve_out["steps"][4]["thought"], "Run it.")
            self.assertEqual(solve_out["final_thought"], "Output matches exactly.")

            build_out = by_kind["build"]
            self.assertEqual(build_out["steps"][0]["thought"], "Read first.")
            self.assertEqual(len(build_out["steps"][1]["thought"].split()), 160)
            self.assertEqual(build_out["steps"][2]["thought"], "Fill main.")

            # Resume: re-running makes 0 backend calls
            called_ids.clear()
            with mock.patch.object(reason5, "call_backend", side_effect=fake_backend):
                reason5.main(["--mode", "episodes", "--in", in_path, "--out", out_path])
            self.assertEqual(called_ids, [])
            with open(out_path) as f:
                self.assertEqual(len(f.readlines()), 2)

    def test_onpolicy_no_loss_write_not_overwritten(self):
        with tempfile.TemporaryDirectory() as td:
            in_path = os.path.join(td, "ep.jsonl")
            out_path = os.path.join(td, "out.jsonl")
            ep_onp = {
                "kind": "onpolicy",
                "benchmark": "fizzbuzz",
                "task_prompt": "Solve fizzbuzz",
                "steps": [
                    {"call": {"name": "read", "args": {"path": "x"}}, "result": "tmpl", "thought": "Read."},
                    {"call": {"name": "write", "args": {"path": "x", "content": "bad"}, "no_loss": True}, "no_loss": True, "result": "ok", "thought": "Model mistake."},
                    {"call": {"name": "edit", "args": {"path": "x", "edits": []}}, "result": "ok", "thought": "Apply fix."},
                ],
                "final": "Done.",
                "final_thought": "Done.",
            }
            with open(in_path, "w") as f:
                f.write(json.dumps(ep_onp) + "\n")

            with mock.patch.object(reason5, "call_backend") as m:
                reason5.main(["--mode", "episodes", "--in", in_path, "--out", out_path])
            m.assert_not_called()
            with open(out_path) as f:
                out_eps = [json.loads(ln) for ln in f]
            self.assertEqual(len(out_eps), 1)
            self.assertEqual(out_eps[0]["steps"][1]["thought"], "Model mistake.")


class TestRepairsMode(unittest.TestCase):
    def test_repair_error_kind_extraction(self):
        cases = [
            ("Error [IMP010]: symbol put not exported by std/json", "IMP010"),
            ("MOD010: module header does not match path", "MOD010"),
            ("PAR017: unexpected ';' in expression body", "PAR017"),
            ("PAR018: let rec is not supported", "PAR018"),
            ("PAR_UNEXPECTED_EOF: unexpected end of file", "PAR_UNEXPECTED_EOF"),
            ("TC_UNIFY_MISMATCH: expected int, got string", "TC_UNIFY_MISMATCH"),
            ("Error: `++` is for lists only, not string", "`++`"),
            ("Error: undefined variable: foo", "undefined variable"),
            ("type unification error: int vs bool", "type unification"),
            ("effect checking failed: missing IO", "effect checking"),
            ("PARSER failed somewhere", "other"),
            ("completely unknown error", "other"),
        ]
        for err, want in cases:
            self.assertEqual(reason5.repair_error_kind(err), want, err)

    def test_validate_reasoning_allows_std_in_broken_or_fixed(self):
        broken = "module benchmark/solution\nimport std/json (put)\nexport func main() -> int = 1\n"
        fixed = "module benchmark/solution\nimport std/string (length)\nexport func main() -> int = length(\"a\")\n"
        # std/json appears only in broken; std/string appears only in fixed
        txt = make_valid_reasoning(
            90,
            extra="IMP010 says symbol put is not in std/json, so I will remove std/json put and import std/string instead.",
        )
        ok, why = reason5.validate_reasoning(txt, broken, extra_code=fixed)
        self.assertTrue(ok, why)

        # Hallucinated std/http appears in neither broken nor fixed -> rejected
        bad_txt = make_valid_reasoning(90, extra="I should import std/http instead.")
        ok2, why2 = reason5.validate_reasoning(bad_txt, broken, extra_code=fixed)
        self.assertFalse(ok2)
        self.assertEqual(why2, "hallucinated_import")

    def test_build_batch_prompt_repairs_mode(self):
        prompt = reason5.build_batch_prompt(
            [
                {
                    "id": "r1",
                    "error": "PAR017: unexpected ';'",
                    "code": "module m\nexport func main() -> int = { 1; }",
                    "fixed": "module m\nexport func main() -> int = 1",
                }
            ],
            "repairs",
        )
        self.assertIn("60 to 250 word", prompt)
        self.assertIn("Compiler error:\nPAR017: unexpected ';'", prompt)
        self.assertIn("Broken AILANG module:", prompt)
        self.assertIn("Target fixed AILANG module", prompt)

    def test_repairs_filters_balances_resumes_and_feeds_build_mix5d(self):
        import build_mix5d  # noqa: E402

        with tempfile.TemporaryDirectory() as td:
            stale_path = os.path.join(td, "stale_pairs.jsonl")
            onp_path = os.path.join(td, "onpolicy_pairs_v2.jsonl")
            out_path = os.path.join(td, "repairs_reasoned.jsonl")

            # stale_pairs.jsonl:
            # - heldout (merge_sort_v2) -> skipped
            # - ok: False -> skipped
            # - broken == fixed -> skipped
            # - empty error -> skipped
            # - 4 `++` repairs for fizzbuzz (will be capped at 2 by --repair-per-kind 2)
            # - 1 `PAR017` repair for fizzbuzz (distinct kind -> kept!)
            # - 1 `IMP010` repair for fizzbuzz where std/json is in `old` (broken) and removed in `fixed`
            stale_rows = [
                {
                    "ok": True,
                    "benchmark": "merge_sort_v2",
                    "old": "module benchmark/solution\nexport func main() -> int = 1",
                    "error": "PAR017: ';'",
                    "fixed": "module benchmark/solution\nexport func main() -> int = 2",
                },
                {
                    "ok": False,
                    "benchmark": "fizzbuzz",
                    "old": "bad",
                    "error": "PAR017: ';'",
                    "fixed": "good",
                },
                {
                    "ok": True,
                    "benchmark": "fizzbuzz",
                    "old": "same",
                    "error": "PAR017: ';'",
                    "fixed": "same",
                },
                {
                    "ok": True,
                    "benchmark": "fizzbuzz",
                    "old": "broken",
                    "error": "   ",
                    "fixed": "fixed",
                },
            ]
            for i in range(4):
                stale_rows.append(
                    {
                        "ok": True,
                        "benchmark": "fizzbuzz",
                        "old": f'module benchmark/solution\nexport func main() -> string = "a" ++ "{i}"',
                        "error": "Error: `++` is for lists only",
                        "fixed": f'module benchmark/solution\nexport func main() -> string = "a{i}"',
                    }
                )
            par17_broken = "module benchmark/solution\nexport func main() -> int = { 1; }"
            par17_fixed = "module benchmark/solution\nexport func main() -> int = 1"
            stale_rows.append(
                {
                    "ok": True,
                    "benchmark": "fizzbuzz",
                    "old": par17_broken,
                    "error": "PAR017: unexpected ';' in expression body",
                    "fixed": par17_fixed,
                }
            )
            imp10_broken = "module benchmark/solution\nimport std/json (put)\nexport func main() -> int = 1"
            imp10_fixed = "module benchmark/solution\nexport func main() -> int = 1"
            stale_rows.append(
                {
                    "ok": True,
                    "benchmark": "fizzbuzz",
                    "old": imp10_broken,
                    "error": "Error [IMP010]: symbol put is not exported by std/json",
                    "fixed": imp10_fixed,
                }
            )
            with open(stale_path, "w", encoding="utf-8") as f:
                for r in stale_rows:
                    f.write(json.dumps(r) + "\n")

            # onpolicy_pairs_v2.jsonl:
            # - uses `code` instead of `old`, omits `ok` (still valid!)
            # - 1 duplicate of stale_rows PAR017 -> deduped
            # - 1 `PAR018` for fizzbuzz
            # - 1 `MOD010` for second_bench (ensures leakage-free split has >=2 groups)
            # - 1 failing --verify-compile for second_bench (`++`) so it must not waste a cap slot
            # - 1 retired heldout (group: bench:run_length_encode) -> skipped
            onp_rows = [
                {
                    "benchmark": "fizzbuzz",
                    "code": par17_broken,
                    "error": "PAR017: unexpected ';' in expression body",
                    "fixed": par17_fixed,
                },
                {
                    "benchmark": "fizzbuzz",
                    "code": "module benchmark/solution\nlet rec f = \\x. x\nexport func main() -> int = 1",
                    "error": "PAR018: let rec is not supported",
                    "fixed": "module benchmark/solution\npure func f(x: int) -> int = x\nexport func main() -> int = f(1)",
                },
                {
                    "benchmark": "second_bench",
                    "code": "module wrong/name\nexport func main() -> int = 1",
                    "error": "MOD010: module header must match path",
                    "fixed": "module benchmark/solution\nexport func main() -> int = 1",
                },
                {
                    "benchmark": "second_bench",
                    "code": 'module benchmark/solution\nexport func main() -> string = "x" ++ "y"',
                    "error": "Error: `++` is for lists only",
                    "fixed": "BROKEN_FIXED_CODE",
                },
                {
                    "benchmark": "second_bench",
                    "code": 'module benchmark/solution\nexport func main() -> string = "c" ++ "d"',
                    "error": "Error: `++` is for lists only",
                    "fixed": 'module benchmark/solution\nexport func main() -> string = "cd"',
                },
                {
                    "group": "bench:run_length_encode",
                    "code": "a",
                    "error": "PAR017: ';'",
                    "fixed": "b",
                },
            ]
            with open(onp_path, "w", encoding="utf-8") as f:
                for r in onp_rows:
                    f.write(json.dumps(r) + "\n")

            imp10_id = reason5.compute_repair_id("fizzbuzz", imp10_broken, imp10_fixed)
            # Seed out_path with an INVALID reasoning row for imp10_id so resume must retry it
            with open(out_path, "w", encoding="utf-8") as f:
                f.write(
                    json.dumps(
                        {
                            "id": imp10_id,
                            "hash": imp10_id,
                            "benchmark": "fizzbuzz",
                            "group": "bench:fizzbuzz",
                            "split": "train",
                            "kind": "repair",
                            "code": imp10_broken,
                            "error": "Error [IMP010]: symbol put is not exported by std/json",
                            "fixed": imp10_fixed,
                            "reasoning": "too short",
                        }
                    )
                    + "\n"
                )

            modes_seen = []

            def fake_backend(backend, model, items, mode):
                modes_seen.append(mode)
                out = {}
                for it in items:
                    if it["id"] == imp10_id:
                        # Mentions std/json which is ONLY in `code` (broken), NOT in `fixed`
                        out[it["id"]] = make_valid_reasoning(
                            95,
                            extra="Compiler error IMP010 says symbol put is not exported by std/json, so I will remove the std/json import.",
                        )
                    else:
                        out[it["id"]] = make_valid_reasoning(95)
                return out

            def fake_verify(code):
                return ("BROKEN_FIXED_CODE" not in code, "compile err")

            with mock.patch.object(reason5, "verify_compile", side_effect=fake_verify), \
                    mock.patch.object(reason5, "call_backend", side_effect=fake_backend):
                rc = reason5.main(
                    [
                        "--mode",
                        "repairs",
                        "--in",
                        f"{stale_path},{onp_path}",
                        "--out",
                        out_path,
                        "--repair-per-kind",
                        "2",
                        "--verify-compile",
                        "--batch-size",
                        "4",
                        "--workers",
                        "1",
                        "--seed",
                        "11",
                    ]
                )
            self.assertEqual(rc, 0)
            self.assertTrue(all(m == "repairs" for m in modes_seen))

            # Expected newly written:
            # fizzbuzz: 2 `++` (capped from 4) + 1 PAR017 + 1 IMP010 + 1 PAR018 = 5
            # second_bench: 1 MOD010 + 1 `++` (BROKEN_FIXED_CODE skipped by verify_compile) = 2
            # Total newly written = 7 (+ 1 seeded invalid row in file = 8 lines)
            with open(out_path, encoding="utf-8") as f:
                all_rows = [json.loads(ln) for ln in f]
            self.assertEqual(len(all_rows), 8)
            valid_rows = [r for r in all_rows if len(r["reasoning"].split()) >= 50]
            self.assertEqual(len(valid_rows), 7)

            for r in valid_rows:
                self.assertEqual(r["id"], r["hash"])
                self.assertEqual(r["group"], "bench:" + r["benchmark"])
                self.assertEqual(r["split"], "train")
                self.assertEqual(r["kind"], "repair")
                self.assertTrue(r["code"] and r["error"] and r["fixed"])
                self.assertNotEqual(r["code"], r["fixed"])

            # Clean out the seeded invalid row (simulating fresh clean output) and test resume = 0 backend calls
            clean_path = os.path.join(td, "repairs_clean.jsonl")
            with open(clean_path, "w", encoding="utf-8") as f:
                for r in valid_rows:
                    f.write(json.dumps(r) + "\n")

            modes_seen.clear()
            with mock.patch.object(reason5, "verify_compile", side_effect=fake_verify), \
                    mock.patch.object(reason5, "call_backend", side_effect=fake_backend):
                reason5.main(
                    [
                        "--mode",
                        "repairs",
                        "--in",
                        f"{stale_path},{onp_path}",
                        "--out",
                        clean_path,
                        "--repair-per-kind",
                        "2",
                        "--verify-compile",
                        "--workers",
                        "1",
                        "--seed",
                        "11",
                    ]
                )
            self.assertEqual(modes_seen, [])
            with open(clean_path, encoding="utf-8") as f:
                self.assertEqual(len(f.readlines()), 7)

            # Verify full compatibility with build_mix5d.py --repairs
            pdir = os.path.join(td, "prompts")
            os.makedirs(pdir, exist_ok=True)
            with open(os.path.join(pdir, "ft-none.md"), "w", encoding="utf-8") as f:
                f.write("You are writing AILANG.")
            with open(os.path.join(pdir, "ft-slim-v1.md"), "w", encoding="utf-8") as f:
                f.write("# TOP TRAPS")
            with open(os.path.join(pdir, "ft-compact-v1.md"), "w", encoding="utf-8") as f:
                f.write("## Compact Ref")

            mix_dir = os.path.join(td, "mix_out")
            mix_args = build_mix5d.parse_args(
                ["--repairs", clean_path, "--prompts", pdir, "--out", mix_dir, "--seed", "5"]
            )
            report = build_mix5d.build_mix(mix_args)
            self.assertEqual(report["total_samples"]["train"] + report["total_samples"]["valid"], 7)
            self.assertGreaterEqual(report["reasoning_words"]["single_turn"]["min"], 50)
            self.assertTrue(any(":repair-" in k for k in report["samples"]))


class TestIsolatedModeCLI(unittest.TestCase):
    def test_invalid_backslash_escape_and_extra_data(self):
        raw = '{"items": [{"id": "a1", "reasoning": "use lambda \\x. x + 1 and \\n newline"}]}\ntrailing line'
        parsed = reason5.parse_batch_response(raw)
        self.assertEqual(parsed["a1"], "use lambda \\x. x + 1 and \n newline")

    def test_python_isolated_mode_help(self):
        script = os.path.join(HERE, "reason5.py")
        r = subprocess.run(
            [sys.executable, "-I", script, "--help"],
            capture_output=True,
            text=True,
            timeout=30,
        )
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("--mode", r.stdout)
        self.assertIn("--batch-size", r.stdout)


if __name__ == "__main__":
    unittest.main()
