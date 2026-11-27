#!/usr/bin/env python3
"""Generate genuine, structured AILANG planning thoughts (`reasoning`) in batches (Pilot-5).

Pilot-5 Runs a & c omitted `reasoning`, training an empty `<|channel>thought\\n<channel|>`
block that collapsed test-time thinking (~430 tokens vs 3,000-6,000) and failed the
pass-rate gate (17% vs 39%). `reason5.py` backfills validated 120-400 word planning
thoughts for verified training items in batches via subscription CLIs (`agy` or `claude-cli`).

Modes:
  single    Enrich `desc.jsonl` (train split), optional `--eval` solutions, and optional
            `--examples-dir` runnable examples with `reasoning` for single-turn SFT.
  episodes  Enrich `episodes_thought.jsonl` by replacing the initial substantive `write`
            (or `build`) step's template thought with a genuine planning `thought`, while
            preserving tactical thoughts on `read`/`check`/`run`/lookup steps.
  repairs   Enrich verified compiler-error -> fix pairs (`stale_pairs.jsonl`,
            `onpolicy_pairs_v2.jsonl`, etc.) with concise 60-250 word diagnostic `reasoning`
            for `build_mix5d.py --repairs`.

Held-out safety gate: any record whose `benchmark` matches `extract_lang.load_heldout()`
(including `_vN` variants) is refused immediately and never sent to any backend or output.
"""
import argparse
import collections
import concurrent.futures as cf
import copy
import hashlib
import json
import os
import random
import re
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
if HERE not in sys.path:
    sys.path.insert(0, HERE)


def _import_finetune_helpers():
    """Import `extract_lang.load_heldout` and `stale_fix` cleanly even under bare `python3 -I`.

    `extract_lang` and `stale_fix` import PyYAML at module top level for benchmark YAML
    parsing, which `load_heldout` and `ask_claude` do not use. If `yaml` is absent in a
    minimal Python environment, inject a failing-on-use stub only during import.
    """
    try:
        from extract_lang import load_heldout as _load_heldout
        import stale_fix as _stale_fix
        return _load_heldout, _stale_fix
    except ModuleNotFoundError as exc:
        if exc.name != "yaml":
            raise
        import types

        stub = types.ModuleType("yaml")
        stub.YAMLError = RuntimeError

        def _no_yaml(*_a, **_kw):
            raise RuntimeError("PyYAML is not installed in this Python environment")

        stub.safe_load = _no_yaml
        sys.modules["yaml"] = stub
        try:
            from extract_lang import load_heldout as _load_heldout
            import stale_fix as _stale_fix
            return _load_heldout, _stale_fix
        finally:
            sys.modules.pop("yaml", None)


load_heldout, stale_fix = _import_finetune_helpers()

SCHEMA = {
    "type": "object",
    "properties": {
        "items": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "reasoning": {"type": "string"},
                },
                "required": ["id", "reasoning"],
                "additionalProperties": False,
            },
        }
    },
    "required": ["items"],
    "additionalProperties": False,
}
SCHEMA_STR = json.dumps(SCHEMA, separators=(",", ":"))

CONTROL_TOKENS = (
    "<|channel>",
    "<channel|>",
    "<|tool_call>",
    "<|turn>",
    "<|thought>",
    "<|tool_response>",
)

META_LEAK_RE = re.compile(
    r"\b(?:"
    r"given program|provided code|provided program|provided solution|"
    r"solution above|looking at the code|in the code above|"
    r"from the solution|above solution|the given solution"
    r")\b",
    re.I,
)

BANNED_SYNTAX_RE = re.compile(
    r"\b(?:"
    r"(?:for|while)\s+loops?|"
    r"do[-\s]while|"
    r"mutable\s+variables?|"
    r"try\s*/\s*catch|try-catch"
    r")\b",
    re.I,
)

STD_MOD_RE = re.compile(r"\b(std/[a-z][a-z0-9_]*(?:/[a-z][a-z0-9_]*)?)\b")
MOD_RE = re.compile(r"^\s*module\s+([\w/.\-]+)", re.M)
EXPORT_SIG_RE = re.compile(r"^\s*(export\s+(?:pure\s+)?(?:func|type)\s+[^\n={]+)", re.M)
REPAIR_ERROR_KIND_RE = re.compile(
    r"\b(IMP\d+|MOD\d+|PAR(?:\d+|_[A-Z0-9_]+)|TC_[A-Z0-9_]+)\b"
    r"|(`\+\+`|undefined variable|type unification|effect checking)"
)


def _bench_base(name: str) -> str:
    return re.sub(r"_v\d+$", "", name.strip())


def is_heldout(benchmark: str | None, heldout: set[str]) -> bool:
    """Return True if `benchmark` (or its `_vN` base) matches `heldout`."""
    if not benchmark:
        return False
    b = str(benchmark).strip()
    if b.startswith("bench:"):
        b = b[len("bench:") :]
    if not b:
        return False
    if b in heldout:
        return True
    held_bases = {_bench_base(h) for h in heldout}
    return _bench_base(b) in held_bases


def _record_is_heldout(rec: dict, heldout: set[str]) -> bool:
    if is_heldout(rec.get("benchmark"), heldout):
        return True
    grp = str(rec.get("group") or "")
    if grp.startswith("bench:") and is_heldout(grp[len("bench:") :], heldout):
        return True
    return False


def validate_reasoning(text: str, code: str, extra_code: str = "") -> tuple[bool, str]:
    """Validate word count, formatting hygiene, AILANG constraints, and code grounding."""
    if not isinstance(text, str) or not text.strip():
        return False, "empty"
    words = text.split()
    if len(words) < 50:
        return False, "too_short"
    if len(words) > 600:
        return False, "too_long"
    if any(tok in text for tok in CONTROL_TOKENS):
        return False, "control_token"
    if "```" in text:
        return False, "code_fence"
    if META_LEAK_RE.search(text):
        return False, "meta_leakage"
    if BANNED_SYNTAX_RE.search(text):
        return False, "banned_syntax"
    combined_code = "\n".join(p for p in (code, extra_code) if p)
    if combined_code:
        for mod in STD_MOD_RE.findall(text):
            if mod not in combined_code:
                if mod == "std/io" and ("println" in combined_code or "print" in combined_code):
                    continue
                return False, "hallucinated_import"
    return True, ""


def verify_compile(code: str) -> tuple[bool, str]:
    """Compile-check an AILANG program via `indist_test.check`."""
    from indist_test import check

    return check(code)


def synthesize_example_task(code: str, relpath: str) -> str:
    """Build a concise task prompt from leading `--` comments, module name, and exports."""
    m = MOD_RE.search(code)
    head_region = code[: m.start()] if m else code
    header_lines = []
    for ln in head_region.splitlines():
        s = ln.strip()
        if not s:
            if header_lines and not m:
                break
            continue
        if s.startswith("--"):
            txt = re.sub(r"^--\s?", "", s).strip()
            if txt.lower().startswith("task:"):
                txt = txt[5:].strip()
            if txt:
                header_lines.append(txt)
        elif not m:
            break
    mod_name = m.group(1) if m else None
    exports = [re.sub(r"\s+", " ", sig.strip()) for sig in EXPORT_SIG_RE.findall(code)]
    if not header_lines and not mod_name and not exports:
        return ""
    parts = []
    if header_lines:
        parts.append(" ".join(header_lines))
    if mod_name:
        parts.append(f"Module: `{mod_name}` (file `{relpath}`).")
    elif not header_lines:
        parts.append(f"Write the AILANG program for `{relpath}`.")
    if exports:
        parts.append("Exported declarations:\n" + "\n".join(f"- `{e}`" for e in exports[:12]))
    return "\n\n".join(parts).strip()


def compute_single_id(rec: dict) -> str:
    return str(rec.get("hash") or hashlib.sha256(rec["code"].encode("utf-8")).hexdigest()[:16])


def compute_repair_id(benchmark: str, broken: str, fixed: str) -> str:
    return hashlib.sha256(f"{benchmark}:{broken}:{fixed}".encode("utf-8")).hexdigest()[:16]


def repair_error_kind(error: str) -> str:
    m = REPAIR_ERROR_KIND_RE.search(error or "")
    if not m:
        return "other"
    return m.group(1) or m.group(2)


def find_first_substantive_write(ep: dict) -> tuple[int | None, str]:
    """Return `(index, content)` of the first trained `write` (or `build`) step."""
    for i, s in enumerate(ep.get("steps") or []):
        if s.get("no_loss"):
            continue
        call = s.get("call") or {}
        if call.get("name") == "write":
            content = (call.get("args") or {}).get("content") or ""
            if content.strip():
                return i, content
    return None, ""


def compute_episode_id(ep: dict, first_write_code: str) -> str:
    bid = str(ep.get("benchmark") or "")
    return hashlib.sha256(f"{bid}:{first_write_code}".encode("utf-8")).hexdigest()[:16]


def _default_tactical_thought(step: dict) -> str:
    call = step.get("call") or {}
    name = call.get("name", "")
    args = call.get("args") or {}
    cmd = str(args.get("command") or "")
    if name == "read":
        return "Read the template file at the given path first."
    if step.get("habit") == "lookup":
        return f"Check the exact stdlib signature for `{step.get('lookup', '')}` before writing."
    if step.get("habit") == "examples":
        return f"Search for a working example using `{step.get('lookup', '')}`."
    if name == "bash" and "ailang check" in cmd:
        return "Run `ailang check` to verify types, imports, and effects before running."
    if name == "bash" and "ailang run" in cmd:
        return "Run the program and verify its output matches the expected output exactly."
    return "Verify the step output and continue."


BATCH_INSTRUCTIONS = """You are generating internal planning scratchpads (`reasoning`) for an AILANG training dataset.
For EACH item below, write the first-person internal thinking scratchpad an expert AILANG developer produces BEFORE writing the target AILANG program.

Requirements for each `reasoning`:
1. Derive the input/output contract, parsing/formatting rules, and subtle edge cases (empty inputs, negative/zero bounds, ties, escaping, exact newline/stdout format) from the task.
2. For algorithmic or data-transformation tasks, walk through the algorithm step by step and trace a small concrete example or boundary case by hand to verify recursion termination, accumulator invariants, and pattern-matching exhaustiveness before settling on the helper structure.
3. Plan the ADTs/records/types, pure helper functions with recursion base and step cases (AILANG is purely functional: NO `for` or `while` loops, NO mutable variables), explicit `std/...` imports, and effect annotations (e.g. `! {IO}`, `! {FS}`) required.
4. Write in first person ("Let's break down...", "First, let's trace...") as genuine pre-implementation thinking, not a dry summary.
5. Do NOT include markdown code fences (```) or paste multi-line code blocks.
6. Do NOT use meta-phrases such as "given program", "provided code", "provided solution", "solution above", or "looking at the code".
7. ONLY mention types, functions, imports, and effects that actually appear in the target AILANG program so the plan aligns 100% with the code.
8. Length: 180 to 450 words on algorithmic/benchmark tasks; 100 to 250 words on simple declarative modules (strictly between 60 and 550 words).

Return ONLY a strict JSON object of the form:
{"items": [{"id": "<item id>", "reasoning": "<scratchpad>"}]}
Include one entry for every item id below."""

REPAIR_BATCH_INSTRUCTIONS = """You are generating concise diagnostic reasoning scratchpads (`reasoning`) for an AILANG compiler-repair training dataset.
For EACH item below, write a 60 to 250 word first-person diagnostic scratchpad as an AILANG developer reading the compiler error on their broken module and planning the minimal fix BEFORE outputting the corrected module.

Requirements for each `reasoning`:
1. Quote or identify the specific compiler error code or message (e.g., `IMP010`, `PAR017`, `PAR018`, `MOD010`, `++`, undefined variable, type mismatch) and pinpoint the line or construct causing it.
2. State the AILANG language or stdlib rule that was violated (e.g., `++` is for lists only rather than strings, `let rec` is disallowed in AILANG, `;` is invalid inside expression bodies, `module` header must match the module path, or a symbol is not exported by that `std/...` module).
3. Explain the minimal change needed to make `ailang check` pass while preserving the program's behaviour identically.
4. Write in first person ("The compiler reports...", "To fix this without changing behavior, I will...") as genuine pre-fix diagnosis.
5. Do NOT include markdown code fences (```) or paste multi-line code blocks.
6. Do NOT use meta-phrases such as "given program", "provided code", "provided program", "provided solution", "solution above", or "looking at the code".
7. ONLY mention `std/...` modules that actually appear in the broken or fixed AILANG module.
8. Length: strictly 60 to 250 words.

Return ONLY a strict JSON object of the form:
{"items": [{"id": "<item id>", "reasoning": "<scratchpad>"}]}
Include one entry for every item id below."""


def build_batch_prompt(items: list[dict], mode: str) -> str:
    if mode == "repairs":
        sections = [REPAIR_BATCH_INSTRUCTIONS]
        for it in items:
            sections.append(
                f"=== ITEM {it['id']} ===\n"
                f"Compiler error:\n{it['error'].strip()}\n\n"
                f"Broken AILANG module:\n{it['code'].strip()}\n\n"
                f"Target fixed AILANG module (reference only — diagnose the error and plan this minimal fix as if about to write it):\n{it['fixed'].strip()}"
            )
        return "\n\n".join(sections)

    sections = [BATCH_INSTRUCTIONS]
    for it in items:
        note = ""
        if mode == "episodes" and it.get("is_build"):
            note = "\nContext: This is the initial build step (helpers + stub `main`) that must compile cleanly before completing `main` in a follow-up edit."
        sections.append(
            f"=== ITEM {it['id']} ==={note}\n"
            f"Task:\n{it['task'].strip()}\n\n"
            f"Target AILANG program (reference only — plan it as if about to write it):\n{it['code'].strip()}"
        )
    return "\n\n".join(sections)


def _strip_fences(txt: str) -> str:
    txt = txt.strip()
    m = re.match(r"^```(?:json)?\s*\n?(.*?)\n?```\s*$", txt, re.S)
    return m.group(1).strip() if m else txt


_JSON_ESCAPE_TOKEN_RE = re.compile(r'\\(["\\/bfnrtu]|u[0-9a-fA-F]{4})|(\\)')


def _fix_json_escapes(txt: str) -> str:
    r"""Escape lone backslashes (e.g. AILANG `\x.` lambdas) while keeping valid JSON escapes intact."""
    return _JSON_ESCAPE_TOKEN_RE.sub(lambda m: m.group(0) if m.group(1) else r"\\", txt)


def _first_json_dict(txt: str) -> dict:
    """Extract the first JSON object from `txt` cleanly even if trailing text follows."""
    dec = json.JSONDecoder()
    for candidate in (txt, _fix_json_escapes(txt)):
        idx = 0
        while True:
            lo = candidate.find("{", idx)
            if lo == -1:
                break
            try:
                obj, _ = dec.raw_decode(candidate, lo)
                if isinstance(obj, dict):
                    return obj
            except ValueError:
                pass
            idx = lo + 1
    raise ValueError("no JSON object found")


def parse_batch_response(raw_text: str) -> dict[str, str]:
    """Parse `{"items": [{"id": ..., "reasoning": ...}]}` into `{id: reasoning}`."""
    txt = _strip_fences(raw_text)
    try:
        data = json.loads(txt)
    except ValueError:
        try:
            data = json.loads(_fix_json_escapes(txt))
        except ValueError:
            data = _first_json_dict(txt)
    if not isinstance(data, dict) or not isinstance(data.get("items"), list):
        raise ValueError("response JSON missing 'items' list")
    out: dict[str, str] = {}
    for entry in data["items"]:
        if isinstance(entry, dict) and entry.get("id") is not None and isinstance(entry.get("reasoning"), str):
            out[str(entry["id"])] = entry["reasoning"].strip()
    return out


def call_agy(model: str, prompt: str) -> dict[str, str]:
    """Invoke `agy` via the isolated subscription profile."""
    default_bin = os.path.expanduser("~/.ailang/state/agy/bin/agy")
    agy_bin = default_bin if os.path.exists(default_bin) else "agy"
    env = dict(os.environ)
    for key in ("GEMINI_API_KEY", "GOOGLE_API_KEY", "AGY_ADC_AUTH", "GOOGLE_CLOUD_PROJECT"):
        env.pop(key, None)
    env["HOME"] = os.environ.get(
        "AILANG_AGY_SUB_HOME", os.path.expanduser("~/.ailang/state/agy/subscription")
    )
    env["AGY_CLI_DISABLE_AUTO_UPDATE"] = "true"
    cmd = [
        agy_bin,
        "--model",
        model,
        "--mode",
        "plan",
        "--disable-slash-commands",
        "--output-format",
        "json",
        "--json-schema",
        SCHEMA_STR,
        "--print-timeout",
        "300s",
        "-p",
        prompt,
    ]
    with tempfile.TemporaryDirectory() as td:
        r = subprocess.run(
            cmd,
            cwd=td,
            env=env,
            stdin=subprocess.DEVNULL,
            capture_output=True,
            text=True,
            timeout=360,
        )
    if r.returncode != 0:
        msg = (r.stderr or r.stdout or "").strip()[:200]
        raise RuntimeError(f"agy rc={r.returncode}: {msg}")
    try:
        d = json.loads(r.stdout)
    except ValueError:
        # Handle multi-line NDJSON or diagnostic lines preceding the final JSON envelope.
        d = None
        for ln in reversed(r.stdout.splitlines()):
            ln = ln.strip()
            if ln.startswith("{"):
                try:
                    cand = json.loads(ln)
                    if isinstance(cand, dict) and ("response" in cand or "status" in cand):
                        d = cand
                        break
                except ValueError:
                    continue
        if d is None:
            d = _first_json_dict(r.stdout)
    if d.get("is_error") or str(d.get("status", "")).upper() == "ERROR":
        raise RuntimeError(f"agy error: {d.get('error') or d.get('status')}")
    resp = d.get("response")
    if resp is None:
        raise RuntimeError("agy response JSON missing 'response' field")
    if isinstance(resp, dict):
        return parse_batch_response(json.dumps(resp))
    return parse_batch_response(str(resp))


def call_claude(model: str, prompt: str) -> dict[str, str]:
    """Invoke `stale_fix.ask_claude` and parse batched JSON output."""
    txt, _usage = stale_fix.ask_claude(model, prompt)
    return parse_batch_response(txt)


def call_backend(backend: str, model: str, items: list[dict], mode: str) -> dict[str, str]:
    prompt = build_batch_prompt(items, mode)
    if backend == "agy":
        return call_agy(model, prompt)
    if backend == "claude-cli":
        return call_claude(model, prompt)
    raise ValueError(f"unknown backend: {backend}")


def _chunks(seq: list, size: int, max_chars: int = 55000):
    """Yield batches capped by both item count (`size`) and prompt payload size (`max_chars`)
    so CLI `argv` never approaches macOS ARG_MAX (~256 KB)."""
    cur: list = []
    cur_chars = 0
    limit = max(1, size)
    for item in seq:
        est = (
            len(str(item.get("task", "")))
            + len(str(item.get("code", "")))
            + len(str(item.get("error", "")))
            + len(str(item.get("fixed", "")))
            + 200
        )
        if cur and (len(cur) >= limit or cur_chars + est > max_chars):
            yield cur
            cur = []
            cur_chars = 0
        cur.append(item)
        cur_chars += est
    if cur:
        yield cur


def _load_done_ids(out_path: str, id_field: str, require_valid_reasoning: bool = False) -> set[str]:
    done: set[str] = set()
    if not os.path.exists(out_path):
        return done
    with open(out_path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                d = json.loads(line)
            except ValueError:
                continue
            val = d.get(id_field) or d.get("hash")
            if not val:
                continue
            if require_valid_reasoning:
                ok, _ = validate_reasoning(
                    d.get("reasoning", ""),
                    d.get("code", ""),
                    extra_code=d.get("fixed", ""),
                )
                if not ok:
                    continue
            done.add(str(val))
    return done


def load_single_items(args, heldout: set[str], rnd: random.Random) -> tuple[list[dict], collections.Counter]:
    stats: collections.Counter = collections.Counter()
    candidates: list[dict] = []
    seen_ids: set[str] = set()

    allowed_splits = {"train", "valid"} if args.include_valid else {"train"}
    with open(os.path.expanduser(args.in_path), encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            r = json.loads(line)
            if _record_is_heldout(r, heldout):
                stats["heldout_skipped"] += 1
                continue
            split = r.get("split", "train")
            if split not in allowed_splits:
                stats["split_skipped"] += 1
                continue
            if not r.get("task") or not r.get("code"):
                stats["incomplete_skipped"] += 1
                continue
            rid = compute_single_id(r)
            if rid in seen_ids:
                stats["dedup_skipped"] += 1
                continue
            seen_ids.add(rid)
            rec = dict(r)
            rec["id"] = rid
            rec["hash"] = r.get("hash") or rid
            rec.setdefault("kind", "repo")
            candidates.append(rec)

    if args.eval:
        per = collections.defaultdict(list)
        with open(os.path.expanduser(args.eval), encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                d = json.loads(line)
                if is_heldout(d.get("benchmark"), heldout):
                    stats["heldout_skipped"] += 1
                    continue
                if d.get("task") and d.get("code") and d.get("benchmark"):
                    per[d["benchmark"]].append(d)
        for bid in sorted(per):
            items = per[bid]
            rnd.shuffle(items)
            kept, models, codes = 0, set(), set()
            for want_new_model in (True, False):
                for d in items:
                    h = hashlib.sha256(d["code"].encode("utf-8")).hexdigest()[:16]
                    model_name = d.get("model", "")
                    if kept >= args.eval_cap or h in codes or (want_new_model and model_name in models):
                        continue
                    if args.verify_compile and not verify_compile(d["code"])[0]:
                        stats["eval_fails_pinned_check"] += 1
                        codes.add(h)
                        continue
                    codes.add(h)
                    models.add(model_name)
                    kept += 1
                    if h in seen_ids:
                        stats["dedup_skipped"] += 1
                        continue
                    seen_ids.add(h)
                    candidates.append(
                        {
                            "id": h,
                            "hash": h,
                            "group": "bench:" + bid,
                            "split": "train",
                            "lines": d["code"].count("\n"),
                            "kind": "bench",
                            "benchmark": bid,
                            "model": model_name,
                            "task": d["task"],
                            "code": d["code"],
                        }
                    )

    if args.examples_dir:
        ex_root = os.path.expanduser(args.examples_dir)
        ail_files = []
        for dp, _, fns in os.walk(ex_root):
            for fn in fns:
                if fn.endswith(".ail"):
                    full = os.path.join(dp, fn)
                    rel = os.path.relpath(full, ex_root).replace(os.sep, "/")
                    ail_files.append((rel, full))
        ail_files.sort()
        for rel, full in ail_files:
            try:
                with open(full, encoding="utf-8") as ef:
                    code = ef.read()
            except (OSError, UnicodeDecodeError):
                stats["example_unreadable"] += 1
                continue
            task = synthesize_example_task(code, rel)
            if not task:
                stats["example_no_metadata"] += 1
                continue
            if args.verify_compile and not verify_compile(code)[0]:
                stats["example_fails_pinned_check"] += 1
                continue
            h = hashlib.sha256(code.encode("utf-8")).hexdigest()[:16]
            if h in seen_ids:
                stats["dedup_skipped"] += 1
                continue
            seen_ids.add(h)
            candidates.append(
                {
                    "id": h,
                    "hash": h,
                    "path": full,
                    "group": "example:" + rel,
                    "split": "train",
                    "lines": code.count("\n"),
                    "kind": "example",
                    "task": task,
                    "code": code,
                }
            )

    done_ids = _load_done_ids(os.path.expanduser(args.out), "id", require_valid_reasoning=True)
    stats["resumed_skipped"] = sum(1 for c in candidates if c["id"] in done_ids)
    todo = [c for c in candidates if c["id"] not in done_ids]
    rnd.shuffle(todo)
    if args.limit > 0:
        todo = todo[: args.limit]
    return todo, stats


def run_single(args, heldout: set[str], rnd: random.Random) -> dict:
    todo, stats = load_single_items(args, heldout, rnd)
    out_path = os.path.expanduser(args.out)
    os.makedirs(os.path.dirname(os.path.abspath(out_path)), exist_ok=True)
    batches = list(_chunks(todo, args.batch_size))
    rejects: collections.Counter = collections.Counter()

    def process_batch(batch: list[dict]):
        res = call_backend(args.backend, args.model, batch, "single")
        return batch, res

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex, open(out_path, "a", encoding="utf-8") as out_f:
        futures = [ex.submit(process_batch, b) for b in batches]
        for fut in cf.as_completed(futures):
            try:
                batch, res = fut.result()
            except Exception as exc:  # noqa: BLE001
                stats["batch_errors"] += 1
                print(f"[reason5] batch error: {exc}", file=sys.stderr)
                continue
            stats["batches_ok"] += 1
            for rec in batch:
                reasoning = res.get(rec["id"], "")
                if not reasoning:
                    stats["missing_in_response"] += 1
                    continue
                ok, why = validate_reasoning(reasoning, rec["code"])
                if not ok:
                    stats["rejected"] += 1
                    rejects[why] += 1
                    continue
                out_rec = dict(rec)
                out_rec["reasoning"] = reasoning
                out_f.write(json.dumps(out_rec) + "\n")
                out_f.flush()
                stats["written"] += 1

    return {"mode": "single", "todo": len(todo), "stats": dict(stats), "reject_reasons": dict(rejects)}


def run_episodes(args, heldout: set[str], rnd: random.Random) -> dict:
    stats: collections.Counter = collections.Counter()
    rejects: collections.Counter = collections.Counter()
    out_path = os.path.expanduser(args.out)
    done_ids = _load_done_ids(out_path, "episode_id")
    seen_ids: set[str] = set(done_ids)

    todo_jobs: list[dict] = []
    passthrough_eps: list[dict] = []

    with open(os.path.expanduser(args.in_path), encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            ep = json.loads(line)
            if is_heldout(ep.get("benchmark"), heldout):
                stats["heldout_skipped"] += 1
                continue
            idx, first_code = find_first_substantive_write(ep)
            if idx is None:
                fallback_src = json.dumps(ep.get("steps") or [], sort_keys=True)
                ep_id = compute_episode_id(ep, fallback_src)
                if ep_id in seen_ids:
                    if ep_id in done_ids:
                        stats["resumed_skipped"] += 1
                    else:
                        stats["dedup_skipped"] += 1
                    continue
                seen_ids.add(ep_id)
                ep_copy = copy.deepcopy(ep)
                ep_copy["episode_id"] = ep_id
                for st in ep_copy.get("steps") or []:
                    if not st.get("thought"):
                        st["thought"] = _default_tactical_thought(st)
                passthrough_eps.append(ep_copy)
                continue

            ep_id = compute_episode_id(ep, first_code)
            if ep_id in done_ids:
                stats["resumed_skipped"] += 1
                continue
            if ep_id in seen_ids:
                stats["dedup_skipped"] += 1
                continue
            seen_ids.add(ep_id)
            step = ep["steps"][idx]
            todo_jobs.append(
                {
                    "id": ep_id,
                    "episode_id": ep_id,
                    "step_idx": idx,
                    "task": ep.get("task_prompt") or "",
                    "code": first_code,
                    "is_build": step.get("habit") == "build",
                    "episode": ep,
                }
            )

    rnd.shuffle(todo_jobs)
    if args.limit > 0:
        todo_jobs = todo_jobs[: args.limit]
        if len(todo_jobs) >= args.limit:
            passthrough_eps = []

    os.makedirs(os.path.dirname(os.path.abspath(out_path)), exist_ok=True)
    batches = list(_chunks(todo_jobs, args.batch_size))

    def process_batch(batch: list[dict]):
        res = call_backend(args.backend, args.model, batch, "episodes")
        return batch, res

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex, open(out_path, "a", encoding="utf-8") as out_f:
        for ep_copy in passthrough_eps:
            out_f.write(json.dumps(ep_copy) + "\n")
            out_f.flush()
            stats["passthrough_written"] += 1

        futures = [ex.submit(process_batch, b) for b in batches]
        for fut in cf.as_completed(futures):
            try:
                batch, res = fut.result()
            except Exception as exc:  # noqa: BLE001
                stats["batch_errors"] += 1
                print(f"[reason5] batch error: {exc}", file=sys.stderr)
                continue
            stats["batches_ok"] += 1
            for job in batch:
                reasoning = res.get(job["id"], "")
                if not reasoning:
                    stats["missing_in_response"] += 1
                    continue
                ok, why = validate_reasoning(reasoning, job["code"])
                if not ok:
                    stats["rejected"] += 1
                    rejects[why] += 1
                    continue
                ep_out = copy.deepcopy(job["episode"])
                ep_out["episode_id"] = job["episode_id"]
                for i, st in enumerate(ep_out.get("steps") or []):
                    if i == job["step_idx"]:
                        st["thought"] = reasoning
                    elif not st.get("thought"):
                        st["thought"] = _default_tactical_thought(st)
                out_f.write(json.dumps(ep_out) + "\n")
                out_f.flush()
                stats["written"] += 1

    return {"mode": "episodes", "todo": len(todo_jobs), "stats": dict(stats), "reject_reasons": dict(rejects)}


def load_repair_items(args, heldout: set[str], rnd: random.Random) -> tuple[list[dict], collections.Counter]:
    stats: collections.Counter = collections.Counter()
    candidates: list[dict] = []
    seen_ids: set[str] = set()

    in_paths = [os.path.expanduser(p.strip()) for p in args.in_path.split(",") if p.strip()]
    if not in_paths:
        raise ValueError("--in must specify at least one JSONL path")

    for in_path in in_paths:
        with open(in_path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                r = json.loads(line)
                if "ok" in r and not r["ok"]:
                    stats["not_ok_skipped"] += 1
                    continue
                if _record_is_heldout(r, heldout):
                    stats["heldout_skipped"] += 1
                    continue
                broken = r.get("old") or r.get("code") or ""
                error = r.get("error") or ""
                fixed = r.get("fixed") or ""
                if (
                    not isinstance(broken, str)
                    or not isinstance(error, str)
                    or not isinstance(fixed, str)
                    or not broken.strip()
                    or not error.strip()
                    or not fixed.strip()
                    or broken == fixed
                ):
                    stats["incomplete_skipped"] += 1
                    continue
                benchmark = str(r.get("benchmark") or "").strip()
                if not benchmark:
                    grp = str(r.get("group") or "")
                    if grp.startswith("bench:"):
                        benchmark = grp[len("bench:") :].strip()
                rid = compute_repair_id(benchmark, broken, fixed)
                if rid in seen_ids:
                    stats["dedup_skipped"] += 1
                    continue
                seen_ids.add(rid)
                candidates.append(
                    {
                        "id": rid,
                        "benchmark": benchmark,
                        "error_kind": repair_error_kind(error),
                        "code": broken,
                        "error": error,
                        "fixed": fixed,
                    }
                )

    done_ids = _load_done_ids(os.path.expanduser(args.out), "id", require_valid_reasoning=True)
    already_done = [c for c in candidates if c["id"] in done_ids]
    pending = [c for c in candidates if c["id"] not in done_ids]
    rnd.shuffle(pending)

    per: collections.Counter = collections.Counter()
    todo: list[dict] = []
    cap = args.repair_per_kind
    for c in already_done + pending:
        cap_key = (c["benchmark"], c["error_kind"])
        if c["id"] in done_ids:
            if cap <= 0 or per[cap_key] < cap:
                per[cap_key] += 1
            stats["resumed_skipped"] += 1
            continue
        if cap > 0 and per[cap_key] >= cap:
            stats["capped_skipped"] += 1
            continue
        if args.verify_compile and not verify_compile(c["fixed"])[0]:
            stats["repair_fails_pinned_check"] += 1
            continue
        per[cap_key] += 1
        todo.append(c)

    rnd.shuffle(todo)
    if args.limit > 0:
        todo = todo[: args.limit]
    return todo, stats


def run_repairs(args, heldout: set[str], rnd: random.Random) -> dict:
    todo, stats = load_repair_items(args, heldout, rnd)
    out_path = os.path.expanduser(args.out)
    os.makedirs(os.path.dirname(os.path.abspath(out_path)), exist_ok=True)
    batches = list(_chunks(todo, args.batch_size))
    rejects: collections.Counter = collections.Counter()

    def process_batch(batch: list[dict]):
        res = call_backend(args.backend, args.model, batch, "repairs")
        return batch, res

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex, open(out_path, "a", encoding="utf-8") as out_f:
        futures = [ex.submit(process_batch, b) for b in batches]
        for fut in cf.as_completed(futures):
            try:
                batch, res = fut.result()
            except Exception as exc:  # noqa: BLE001
                stats["batch_errors"] += 1
                print(f"[reason5] batch error: {exc}", file=sys.stderr)
                continue
            stats["batches_ok"] += 1
            for rec in batch:
                reasoning = res.get(rec["id"], "")
                if not reasoning:
                    stats["missing_in_response"] += 1
                    continue
                ok, why = validate_reasoning(reasoning, rec["code"], extra_code=rec["fixed"])
                if not ok:
                    stats["rejected"] += 1
                    rejects[why] += 1
                    continue
                out_rec = {
                    "id": rec["id"],
                    "hash": rec["id"],
                    "benchmark": rec["benchmark"],
                    "group": "bench:" + rec["benchmark"],
                    "split": "train",
                    "kind": "repair",
                    "code": rec["code"],
                    "error": rec["error"],
                    "fixed": rec["fixed"],
                    "reasoning": reasoning,
                }
                out_f.write(json.dumps(out_rec) + "\n")
                out_f.flush()
                stats["written"] += 1

    return {"mode": "repairs", "todo": len(todo), "stats": dict(stats), "reject_reasons": dict(rejects)}


def build_arg_parser() -> argparse.ArgumentParser:
    ap = argparse.ArgumentParser(description="Generate batched AILANG planning reasoning traces.")
    ap.add_argument("--mode", choices=["single", "episodes", "repairs"], required=True)
    ap.add_argument("--in", dest="in_path", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--eval", default="")
    ap.add_argument("--eval-cap", type=int, default=8)
    ap.add_argument("--examples-dir", default="")
    ap.add_argument("--include-valid", action="store_true")
    ap.add_argument("--repair-per-kind", type=int, default=10)
    ap.add_argument("--verify-compile", action="store_true")
    ap.add_argument("--backend", choices=["agy", "claude-cli"], default="agy")
    ap.add_argument("--model", default="gemini-4-neon-medium")
    ap.add_argument("--batch-size", type=int, default=8)
    ap.add_argument("--workers", type=int, default=3)
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--seed", type=int, default=17)
    return ap


def main(argv=None) -> int:
    ap = build_arg_parser()
    args = ap.parse_args(argv)
    heldout = load_heldout()
    rnd = random.Random(args.seed)
    if args.mode == "single":
        summary = run_single(args, heldout, rnd)
    elif args.mode == "episodes":
        summary = run_episodes(args, heldout, rnd)
    else:
        summary = run_repairs(args, heldout, rnd)
    print(json.dumps(summary))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
