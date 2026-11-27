#!/usr/bin/env python3
"""Pilot-5d training mix: joint single-turn language + repair + lean-pi agent episodes.

Combines up to three sources into a single thinking-ON mlx-lm dataset (train.jsonl, valid.jsonl,
and mix_report.json) for tools/finetune/train2.py:

  --single    reasoned_single.jsonl from reason5.py --mode single:
                {id/hash, group, kind, task, reasoning, code}
              Rendered with 3 prompt prefixes (~35% none, ~50% slim, ~15% compact);
              kind == "example" is repeated --example-repeat times with distinct prompt arms.
  --episodes  optional reasoned_episodes.jsonl from reason5.py --mode episodes,
              rendered via build_mix2.render in lean-pi format, capped at --agent-cap 450,
              --max-steps 20, --max-write-chars 6500.
  --repairs   optional repairs.jsonl single-turn compiler error -> fix pairs:
                {benchmark/group, code, error, fixed, reasoning}
              Rendered with slim/none prefix and the check-repair task template.

Safety gates enforced unconditionally:
  1. Held-out isolation: refuses any record whose benchmark or group matches
     extract_lang.load_heldout() (core + stretch + retired_excluded).
  2. Leakage-free split: 5% of canonical group/benchmark keys go to valid.jsonl, 95% to
     train.jsonl, so no group or benchmark straddles train and valid.
  3. Hard non-empty reasoning gate: every assistant message with weight != 0 must have
     >= 40 reasoning words for single-turn code generation (--single, --repairs) and
     >= 8 reasoning words for multi-turn tactical steps (--episodes).
  4. Token/char length gate: exact token count via train2._render when --tokenizer is passed
     and --max-tokens > 0; otherwise falls back to --max-chars 32000.
"""
import argparse
import collections
import hashlib
import json
import os
import random
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
if HERE not in sys.path:
    sys.path.insert(0, HERE)

import build_mix2  # noqa: E402
try:
    import extract_lang  # noqa: E402
except ModuleNotFoundError as _e:
    if _e.name != "yaml":
        raise
    import types as _types
    sys.modules["yaml"] = _types.ModuleType("yaml")
    try:
        import extract_lang  # noqa: E402
    finally:
        sys.modules.pop("yaml", None)

PROMPT_FILES = {"none": "ft-none.md", "slim": "ft-slim-v1.md"}
ARM_WEIGHTS = (("none", 35), ("slim", 50), ("compact", 15))
SOLUTION_TEMPLATE = (
    "module benchmark/solution\n"
    "// DO NOT CHANGE THE MODULE DECLARATION ABOVE!\n"
    "// TODO: Add your solution code below\n\n"
)


def _fake_ws(rnd: random.Random, bid: str) -> str:
    safe_bid = re.sub(r"[^\w]+", "_", bid or "repair")
    a = "".join(rnd.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(2))
    b = "".join(rnd.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(30))
    pid = rnd.randint(1000, 99999)
    ts = f"2026{rnd.randint(1, 12):02d}{rnd.randint(1, 28):02d}_{rnd.randint(0, 235959):06d}"
    return (
        f"/var/folders/{a}/{b}/T/ailang_eval/"
        f"{safe_bid}_pi-gemma4-12b-ft_{ts}_{pid}_t1_{rnd.randint(10**8, 10**9)}/"
        f"{safe_bid}_ailang_pi_{pid}"
    )


def _percentile(sorted_vals: list[int], pct: float) -> int:
    if not sorted_vals:
        return 0
    idx = min(len(sorted_vals) - 1, max(0, int(round((pct / 100.0) * (len(sorted_vals) - 1)))))
    return sorted_vals[idx]


def _word_stats(vals: list[int]) -> dict:
    if not vals:
        return {"count": 0, "min": 0, "p50": 0, "p95": 0, "max": 0}
    s = sorted(vals)
    return {
        "count": len(s),
        "min": s[0],
        "p50": _percentile(s, 50),
        "p95": _percentile(s, 95),
        "max": s[-1],
    }


def build_compact_prompt(src_path: str, max_chars: int = 6400) -> str:
    """Build deterministic compact syntax+stdlib reference card from prompts/v0.16.20.md."""
    with open(src_path, encoding="utf-8") as f:
        raw = f.read()
    secs = re.split(r"(?=^## )", raw, flags=re.M)
    by_heading = {}
    for sec in secs:
        stripped = sec.strip()
        if not stripped:
            continue
        first = stripped.splitlines()[0].strip()
        if first.startswith("## ") and first not in by_heading:
            by_heading[first] = stripped

    parts = []
    for h in ("## Required Program Structure", "## What AILANG Does NOT Have"):
        if h in by_heading:
            parts.append(by_heading[h])
    if "## Quick Syntax Reference" in by_heading:
        parts.append(by_heading["## Quick Syntax Reference"])
    elif "## Syntax Reference" in by_heading:
        parts.append(by_heading["## Syntax Reference"])

    if "## Standard Library" in by_heading:
        std_head = by_heading["## Standard Library"].split("**Renamed stdlib functions.**")[0].strip()
        budget = max_chars - sum(len(p) + 2 for p in parts)
        if budget > 300:
            if len(std_head) <= budget:
                parts.append(std_head)
            else:
                # Keep complete paragraphs/lines cleanly without mid-sentence truncation.
                chunks = std_head.split("\n\n")
                kept_chunks, used = [], 0
                for ch in chunks:
                    if "```" in ch:
                        # Take complete import lines inside the code fence up to budget.
                        fence_lines, f_used, in_fence = [], 0, False
                        for ln in ch.splitlines():
                            reserve = 4 if in_fence else 0
                            if used + f_used + len(ln) + 1 + reserve > budget:
                                break
                            fence_lines.append(ln)
                            f_used += len(ln) + 1
                            if ln.strip().startswith("```"):
                                in_fence = not in_fence
                        if in_fence:
                            fence_lines.append("```")
                        if len(fence_lines) >= 3:
                            kept_chunks.append("\n".join(fence_lines))
                        break
                    if used + len(ch) + 2 > budget:
                        break
                    kept_chunks.append(ch)
                    used += len(ch) + 2
                if len(kept_chunks) >= 2:
                    parts.append("\n\n".join(kept_chunks).strip())

    if not parts:
        raise ValueError(f"no compact reference headings found in {src_path}")
    return "\n\n".join(parts).strip()


def load_prompts(prompts_dir: str) -> dict[str, str]:
    """Load none, slim, and compact prompt prefixes from prompts_dir (with v0.16.20.md fallback)."""
    out = {}
    for key, fn in PROMPT_FILES.items():
        p = os.path.join(prompts_dir, fn)
        with open(p, encoding="utf-8") as f:
            out[key] = f.read().strip()

    compact_path = os.path.join(prompts_dir, "ft-compact-v1.md")
    if os.path.exists(compact_path):
        with open(compact_path, encoding="utf-8") as f:
            out["compact"] = f.read().strip()
        return out

    candidates = [
        os.path.join(prompts_dir, "v0.16.20.md"),
        os.path.join(extract_lang.REPO, "prompts", "v0.16.20.md"),
    ]
    for cand in candidates:
        if os.path.exists(cand):
            out["compact"] = build_compact_prompt(cand)
            return out

    raise FileNotFoundError(
        f"neither {compact_path} nor prompts/v0.16.20.md found for compact prompt"
    )


def _pick_arms(rnd: random.Random, n: int) -> list[str]:
    """Pick n prompt arms weighted (~35% none, ~50% slim, ~15% compact), distinct while pool allows."""
    chosen = []
    avail = list(ARM_WEIGHTS)
    for _ in range(n):
        if not avail:
            avail = list(ARM_WEIGHTS)
        names = [x[0] for x in avail]
        weights = [x[1] for x in avail]
        pick = rnd.choices(names, weights=weights, k=1)[0]
        chosen.append(pick)
        avail = [(nm, w) for nm, w in avail if nm != pick]
    return chosen


def _canonical_split_key(benchmark: str | None, group: str | None, fallback_id: str) -> str:
    if benchmark:
        return f"bench:{benchmark}"
    if group:
        return str(group)
    return fallback_id


def _check_not_heldout(
    held: set[str],
    source: str,
    benchmark: str | None = None,
    group: str | None = None,
    rec_id: str | None = None,
) -> None:
    cands = []
    if benchmark:
        cands.append(str(benchmark))
    if group:
        g = str(group)
        cands.append(g)
        if g.startswith("bench:"):
            cands.append(g[len("bench:"):])
    if rec_id:
        cands.append(str(rec_id))
    for c in cands:
        if c in held:
            raise ValueError(f"held-out benchmark {c!r} in {source}: refusing")


def _assistant_reasoning_words(sample: dict, min_words: int) -> tuple[bool, list[int]]:
    """Return (all_valid, list_of_word_counts) for all weight != 0 assistant messages."""
    counts = []
    for msg in sample.get("messages", []):
        if msg.get("role") != "assistant" or msg.get("weight", 1) == 0:
            continue
        r = msg.get("reasoning")
        if not isinstance(r, str):
            return False, counts
        wc = len(r.split())
        if wc < min_words:
            return False, counts
        counts.append(wc)
    return len(counts) > 0, counts


def _write_args_content_len(step: dict) -> int:
    call = step.get("call") or {}
    if call.get("name") != "write" or step.get("no_loss"):
        return 0
    args = call.get("args")
    if isinstance(args, str):
        try:
            args = json.loads(args)
        except ValueError:
            return len(args)
    if isinstance(args, dict):
        return len(args.get("content") or "")
    return 0


def _read_jsonl(path: str) -> list[dict]:
    with open(os.path.expanduser(path), encoding="utf-8") as f:
        return [json.loads(line) for line in f if line.strip()]


def build_mix(a: argparse.Namespace) -> dict:
    if not a.single and not a.episodes and not a.repairs:
        raise ValueError("at least one of --single, --episodes, or --repairs must be provided")

    held = extract_lang.load_heldout()
    rnd = random.Random(a.seed)

    ntok = None
    verified_single_thought = False
    if a.tokenizer and a.max_tokens:
        from pathlib import Path
        from mlx_lm.utils import load_tokenizer
        from train2 import _render  # exact render training uses

        _tok = load_tokenizer(Path(os.path.expanduser(a.tokenizer)))
        ntok = lambda r: len(_render(_tok, r["messages"], r.get("tools")))  # noqa: E731

    prompts = None
    if a.single or a.repairs:
        if not a.prompts:
            raise ValueError("--prompts is required when --single or --repairs is provided")
        prompts = load_prompts(os.path.expanduser(a.prompts))

    rep = collections.Counter()
    # items: list of (split_key, kind_label, is_single_turn, sample, tok_len_or_none, word_counts)
    accepted = []

    def _length_ok(sample: dict, kind_label: str, is_single: bool = False) -> tuple[bool, int | None]:
        nonlocal verified_single_thought
        if ntok is not None:
            if is_single and not verified_single_thought:
                toks = _render(_tok, sample["messages"], sample.get("tools"))
                if callable(getattr(_tok, "decode", None)):
                    if "<|channel>thought\n" not in _tok.decode(toks):
                        raise ValueError(
                            "rendered single-turn sample missing '<|channel>thought\\n'; check tokenizer chat_template"
                        )
                verified_single_thought = True
                tlen = len(toks)
            else:
                tlen = ntok(sample)
            if tlen > a.max_tokens:
                rep[f"over_max_tokens:{kind_label}"] += 1
                return False, None
            return True, tlen
        clen = len(json.dumps(sample))
        if clen > a.max_chars:
            rep[f"too_long:{kind_label}"] += 1
            return False, None
        return True, None

    # ---- 1. Single-turn code generation (--single)
    if a.single:
        single_recs = _read_jsonl(a.single)
        for r in single_recs:
            code = (r.get("code") or "").strip()
            rec_id = str(r.get("hash") or r.get("id") or hashlib.sha256(code.encode()).hexdigest()[:16])
            bid = r.get("benchmark")
            group = r.get("group")
            _check_not_heldout(held, f"--single ({rec_id})", benchmark=bid, group=group, rec_id=r.get("id"))
        rnd.shuffle(single_recs)
        for r in single_recs:
            if r.get("split", "train") != "train":
                rep["heldback_test"] += 1
                continue
            kind = str(r.get("kind") or "repo")
            task = (r.get("task") or "").strip()
            code = (r.get("code") or "").strip()
            reasoning = (r.get("reasoning") or "").strip()
            if not task or not code:
                rep[f"empty_field:{kind}"] += 1
                continue
            if len(reasoning.split()) < a.min_single_reasoning_words:
                rep[f"short_reasoning:{kind}"] += 1
                continue
            rec_id = str(r.get("hash") or r.get("id") or hashlib.sha256(code.encode()).hexdigest()[:16])
            skey = _canonical_split_key(r.get("benchmark"), r.get("group"), rec_id)
            n_rep = a.example_repeat if kind == "example" else 1
            for pk in _pick_arms(rnd, n_rep):
                user = f"{prompts[pk]}\n\n## Task\n\n{task}"
                sample = {
                    "messages": [
                        {"role": "user", "content": user},
                        {"role": "assistant", "content": code, "reasoning": reasoning},
                    ]
                }
                ok_r, wcounts = _assistant_reasoning_words(sample, a.min_single_reasoning_words)
                if not ok_r:
                    rep[f"short_reasoning:{kind}"] += 1
                    continue
                ok_l, tlen = _length_ok(sample, kind, is_single=True)
                if not ok_l:
                    continue
                accepted.append((skey, f"{kind}-{pk}", True, sample, tlen, wcounts))

    eps = []
    bench_task_templates: dict[str, str] = {}
    if a.episodes:
        eps = _read_jsonl(a.episodes)
        for e in eps:
            _check_not_heldout(
                held,
                "--episodes",
                benchmark=e.get("benchmark"),
                group=e.get("group"),
                rec_id=e.get("id"),
            )
            ebid = str(e.get("benchmark") or "").strip()
            etp = e.get("task_prompt") or ""
            ews = build_mix2.ws_of(etp) if etp else None
            if ebid and ews and ebid not in bench_task_templates:
                bench_task_templates[ebid] = etp.replace(ews, "/__WS__")

    head = tail = tools = None
    arms = None
    if a.episodes or (a.repairs and a.agent_repair_frac > 0.0):
        head, tail, tools = build_mix2.lean_format(a.lean_format)
        with open(os.path.expanduser(a.arm_slim), encoding="utf-8") as f:
            slim_prefix = json.load(f)["prefix"]
        with open(os.path.expanduser(a.arm_none), encoding="utf-8") as f:
            none_prefix = json.load(f)["prefix"]
        arms = [("slim", slim_prefix), ("none", none_prefix)]

    # ---- 2. Compiler error -> fix repairs (--repairs: single-turn + optional lean-pi tool repairs)
    if a.repairs:
        repair_recs = _read_jsonl(a.repairs)
        for r in repair_recs:
            _check_not_heldout(
                held,
                "--repairs",
                benchmark=r.get("benchmark"),
                group=r.get("group"),
                rec_id=r.get("id"),
            )
        rnd.shuffle(repair_recs)
        for r in repair_recs:
            if r.get("split", "train") != "train":
                rep["heldback_test"] += 1
                continue
            code = (r.get("code") or "").strip()
            error = (r.get("error") or "").strip()
            fixed = (r.get("fixed") or "").strip()
            reasoning = (r.get("reasoning") or "").strip()
            if not code or not error or not fixed:
                rep["empty_field:repair"] += 1
                continue
            if len(reasoning.split()) < a.min_single_reasoning_words:
                rep["short_reasoning:repair"] += 1
                continue
            bid = str(r.get("benchmark") or "").strip()
            fallback_id = str(r.get("hash") or r.get("id") or hashlib.sha256(code.encode()).hexdigest()[:16])
            skey = _canonical_split_key(bid or None, r.get("group"), fallback_id)

            use_agent_repair = (
                a.agent_repair_frac > 0.0
                and rnd.random() < a.agent_repair_frac
                and (not a.max_write_chars or len(fixed) <= a.max_write_chars)
            )
            if use_agent_repair:
                ws = _fake_ws(rnd, bid or "repair")
                path = f"{ws}/benchmark/solution.ail"
                tpl = bench_task_templates.get(
                    bid,
                    "Write your complete solution to: /__WS__/benchmark/solution.ail\n\n"
                    "Ensure `ailang check benchmark/solution.ail` passes cleanly.",
                )
                task_prompt = tpl.replace("/__WS__", ws)
                arm_name, arm_prefix = rnd.choice(arms)
                msgs = [
                    {"role": "developer", "content": head + build_mix2.CWD_LINE + "/private" + ws + tail},
                    {"role": "user", "content": arm_prefix + task_prompt},
                    {
                        "role": "assistant",
                        "weight": 0,
                        "content": "",
                        "reasoning": "Read the template file at the given path first.",
                        "tool_calls": [
                            {
                                "id": "call_1",
                                "type": "function",
                                "function": {"name": "read", "arguments": {"path": path}},
                            }
                        ],
                    },
                    {"role": "tool", "tool_call_id": "call_1", "name": "read", "content": SOLUTION_TEMPLATE},
                    {
                        "role": "assistant",
                        "weight": 0,
                        "content": "",
                        "reasoning": "Write the initial implementation to benchmark/solution.ail.",
                        "tool_calls": [
                            {
                                "id": "call_2",
                                "type": "function",
                                "function": {"name": "write", "arguments": {"path": path, "content": code}},
                            }
                        ],
                    },
                    {"role": "tool", "tool_call_id": "call_2", "name": "write", "content": f"Successfully wrote to {path}"},
                    {
                        "role": "assistant",
                        "weight": 0,
                        "content": "",
                        "reasoning": "Run `ailang check` first; the compiler names what is wrong.",
                        "tool_calls": [
                            {
                                "id": "call_3",
                                "type": "function",
                                "function": {
                                    "name": "bash",
                                    "arguments": {"command": "ailang check benchmark/solution.ail"},
                                },
                            }
                        ],
                    },
                    {
                        "role": "tool",
                        "tool_call_id": "call_3",
                        "name": "bash",
                        "content": f"{error}\n\nCommand exited with code 1",
                    },
                    {
                        "role": "assistant",
                        "content": "",
                        "reasoning": reasoning,
                        "tool_calls": [
                            {
                                "id": "call_4",
                                "type": "function",
                                "function": {"name": "write", "arguments": {"path": path, "content": fixed}},
                            }
                        ],
                    },
                ]
                sample = {"messages": msgs, "tools": tools}
                ok_r, wcounts = _assistant_reasoning_words(sample, a.min_single_reasoning_words)
                if not ok_r:
                    rep["short_reasoning:repair"] += 1
                    continue
                ok_l, tlen = _length_ok(sample, "ep-repair-direct")
                if not ok_l:
                    continue
                accepted.append((skey, f"ep-repair-direct-{arm_name}", False, sample, tlen, wcounts))
            else:
                pk = rnd.choice(["slim", "none"])
                user = f"{prompts[pk]}\n\nFix this AILANG module so ailang check passes:\n\n{error}\n\n{code}"
                sample = {
                    "messages": [
                        {"role": "user", "content": user},
                        {"role": "assistant", "content": fixed, "reasoning": reasoning},
                    ]
                }
                ok_r, wcounts = _assistant_reasoning_words(sample, a.min_single_reasoning_words)
                if not ok_r:
                    rep["short_reasoning:repair"] += 1
                    continue
                ok_l, tlen = _length_ok(sample, "repair", is_single=True)
                if not ok_l:
                    continue
                accepted.append((skey, f"repair-{pk}", True, sample, tlen, wcounts))

    # ---- 3. Multi-turn lean-pi episodes (--episodes)
    if a.episodes:
        rnd.shuffle(eps)
        per = collections.Counter()
        ep_kept = 0
        for e in eps:
            ekind = e["kind"]
            if len(e.get("steps", [])) > a.max_steps:
                rep[f"too_long:ep-{ekind}"] += 1
                continue
            if a.max_write_chars and any(_write_args_content_len(st) > a.max_write_chars for st in e.get("steps", [])):
                rep[f"long_write:ep-{ekind}"] += 1
                continue
            # Normalize reasoning/thought keys if coming from reason5.py
            for st in e.get("steps", []):
                if not st.get("thought") and st.get("reasoning"):
                    st["thought"] = st["reasoning"]
            if not e.get("final_thought") and e.get("final_reasoning"):
                e["final_thought"] = e["final_reasoning"]

            # Enforce non-empty reasoning gate before taking a cap slot
            bad_thought = any(
                not st.get("no_loss") and len((st.get("thought") or "").split()) < a.min_step_reasoning_words
                for st in e.get("steps", [])
            )
            bad_final = len((e.get("final_thought") or "").split()) < a.min_step_reasoning_words
            if bad_thought or bad_final:
                rep[f"short_reasoning:ep-{ekind}"] += 1
                continue

            bid = e["benchmark"]
            if ekind == "solve":
                cap_key, cap = ("solve", bid), a.solve_per_bench
            elif ekind == "onpolicy":
                cap_key, cap = ("onpolicy", bid, e.get("error_kind") or build_mix2.error_kind(e)), a.onpolicy_per_kind
            elif ekind == "build":
                cap_key, cap = ("build", bid), a.build_per_bench
            elif ekind == "repair":
                cap_key, cap = ("repair", bid, build_mix2.error_kind(e)), a.repair_per_kind
            else:
                cap_key, cap = (ekind, bid), a.solve_per_bench

            if per[cap_key] >= cap:
                rep[f"capped:ep-{ekind}"] += 1
                continue
            if a.agent_cap > 0 and ep_kept >= a.agent_cap:
                rep[f"agent_cap:ep-{ekind}"] += 1
                continue

            arm_name, arm_prefix = rnd.choice(arms)
            rendered = build_mix2.render(e, head, tail, tools, arm_prefix)
            if rendered is None:
                rep["no_workspace"] += 1
                continue
            ok_r, wcounts = _assistant_reasoning_words(rendered, a.min_step_reasoning_words)
            if not ok_r:
                rep[f"short_reasoning:ep-{ekind}"] += 1
                continue
            ok_l, tlen = _length_ok(rendered, f"ep-{ekind}")
            if not ok_l:
                continue

            per[cap_key] += 1
            ep_kept += 1
            skey = _canonical_split_key(bid, e.get("group"), bid)
            accepted.append((skey, f"ep-{ekind}-{arm_name}", False, rendered, tlen, wcounts))

    # ---- Leakage-free group/benchmark split (5% valid, 95% train)
    groups = sorted({item[0] for item in accepted})
    rnd.shuffle(groups)
    if len(groups) >= 2:
        n_valid = max(1, min(len(groups) - 1, int(len(groups) * a.valid_frac)))
    else:
        n_valid = 0
    valid_groups = set(groups[:n_valid])

    rnd.shuffle(accepted)
    out_dir = os.path.expanduser(a.out)
    os.makedirs(out_dir, exist_ok=True)

    counts = collections.Counter()
    split_totals = collections.Counter()
    chars = collections.Counter()
    exact_toks = collections.Counter()
    all_words, single_words, multi_words = [], [], []

    with open(os.path.join(out_dir, "train.jsonl"), "w", encoding="utf-8") as ft, \
            open(os.path.join(out_dir, "valid.jsonl"), "w", encoding="utf-8") as fv:
        for skey, kind_label, is_single, sample, tlen, wcounts in accepted:
            min_req = a.min_single_reasoning_words if is_single else a.min_step_reasoning_words
            ok_final, _ = _assistant_reasoning_words(sample, min_req)
            if not ok_final:
                raise RuntimeError(f"reasoning gate invariant violated for {kind_label}")

            split = "valid" if skey in valid_groups else "train"
            line = json.dumps(sample)
            (fv if split == "valid" else ft).write(line + "\n")

            counts[(split, kind_label)] += 1
            split_totals[split] += 1
            chars[split] += len(line)
            if tlen is not None:
                exact_toks[split] += tlen
            all_words.extend(wcounts)
            if is_single:
                single_words.extend(wcounts)
            else:
                multi_words.extend(wcounts)

    report = {
        "samples": {f"{k[0]}:{k[1]}": v for k, v in sorted(counts.items())},
        "total_samples": {"train": split_totals["train"], "valid": split_totals["valid"]},
        "reasoning_words": {
            "overall": _word_stats(all_words),
            "single_turn": _word_stats(single_words),
            "multi_turn": _word_stats(multi_words),
        },
        "approx_tokens": {k: int(v / 3.2) for k, v in chars.items()},
        "exact_tokens": dict(exact_toks) if ntok is not None else None,
        "groups": len(groups),
        "valid_groups": len(valid_groups),
        "dropped": dict(sorted(rep.items())),
    }
    with open(os.path.join(out_dir, "mix_report.json"), "w", encoding="utf-8") as rf:
        json.dump(report, rf, indent=1)
    return report


def parse_args(argv=None) -> argparse.Namespace:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--single", default="", help="reason5.py --mode single output JSONL")
    ap.add_argument("--episodes", default="", help="optional reason5.py --mode episodes output JSONL")
    ap.add_argument("--repairs", default="", help="optional single-turn compiler error -> fix JSONL")
    ap.add_argument(
        "--agent-repair-frac",
        type=float,
        default=0.0,
        help="fraction of --repairs rendered as lean-pi check->write(fixed) tool trajectories (0.0-1.0)",
    )
    ap.add_argument("--prompts", default="", help="directory containing ft-none.md, ft-slim-v1.md, [ft-compact-v1.md]")
    ap.add_argument("--out", required=True, help="output directory for train.jsonl, valid.jsonl, mix_report.json")
    ap.add_argument("--example-repeat", type=int, default=2)
    ap.add_argument("--lean-format", "--format", dest="lean_format", default="~/models/finetune/traces/pi_format_lean.json")
    ap.add_argument("--arm-slim", default="~/models/finetune/traces/arm_slim.json")
    ap.add_argument("--arm-none", default="~/models/finetune/traces/arm_none.json")
    ap.add_argument("--agent-cap", type=int, default=450)
    ap.add_argument("--max-steps", type=int, default=20)
    ap.add_argument("--max-write-chars", type=int, default=6500)
    ap.add_argument("--solve-per-bench", type=int, default=4)
    ap.add_argument("--repair-per-kind", type=int, default=2)
    ap.add_argument("--onpolicy-per-kind", type=int, default=8)
    ap.add_argument("--build-per-bench", type=int, default=4)
    ap.add_argument("--tokenizer", default="", help="model dir; with --max-tokens, measure exact tokens via train2._render")
    ap.add_argument("--max-tokens", type=int, default=12288)
    ap.add_argument("--max-chars", type=int, default=32000)
    ap.add_argument("--min-single-reasoning-words", type=int, default=40)
    ap.add_argument("--min-step-reasoning-words", type=int, default=8)
    ap.add_argument("--valid-frac", type=float, default=0.05)
    ap.add_argument("--seed", type=int, default=17)
    return ap.parse_args(argv)


def main(argv=None):
    a = parse_args(argv)
    try:
        report = build_mix(a)
    except ValueError as e:
        sys.exit(str(e))
    print(json.dumps(report, indent=1))


if __name__ == "__main__":
    main()
