#!/usr/bin/env python3
"""Synthesize eval-format agent episodes from VERIFIED programs (pilot-2, design doc §Pilot-2).

Pilot-1's trajectories came from other harnesses and tasks: 181 of 1,083 samples used the
eval's task format, none used pi's real system prompt, and 55% carried stale workspace paths.
This builds episodes in the exact shape a pi agent eval has, from programs we can run:

  solve   training-split eval passes: read the template -> write the solution to the path
          the task names -> run it -> the output matches -> a one-line done message.
  repair  staleness pairs (stale_fix.py): write the old code -> `ailang check` -> the real
          error -> edit (single hunk) or write (otherwise) per round of the repair chain ->
          run -> done. Habit 4: follow the compiler.
  example examples/ programs with pinned stdout, retargeted to `module benchmark/solution`,
          as a task written from the example's `-- Task:` header.

Every tool result is REAL: the program is executed with the current `ailang` and an episode
is kept only if its output equals the expected output its task prompt shows. The task prompt
comes from the harness itself (tools/finetune/agenttask). The output is prefix-independent:
build_mix2.py decides how to render the system prompt, tools and arm prompt.

  uv run --with pyyaml python tools/finetune/synth_episodes.py --out ~/models/finetune/episodes-v1
"""
import argparse
import concurrent.futures as cf
import difflib
import glob
import json
import os
import random
import re
import subprocess
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from extract_lang import AILANG, REPO, load_heldout  # noqa: E402

import yaml  # noqa: E402

TEMPLATE = ("module benchmark/solution\n// DO NOT CHANGE THE MODULE DECLARATION ABOVE!\n"
            "// TODO: Add your solution code below\n\n")
WS_TOKEN = "/__WS__"
UNVERIFIABLE = ("stdin", "input_files", "hidden_tests_dir", "cli_args", "source_constraints", "solution_files", "grading")
DONE_LINES = ["The solution is in {rel} and its output matches the expected output exactly.",
              "Done: {rel} runs and prints exactly the expected output.",
              "Verified: `ailang run` on {rel} prints the expected output."]


def render_tasks(bench_ids):
    """Exact agent task prompts from the harness, with a placeholder workspace."""
    files = [os.path.join(REPO, "benchmarks", f"{b}.yml") for b in bench_ids]
    r = subprocess.run(["go", "run", "./tools/finetune/agenttask", "-solution", f"{WS_TOKEN}/benchmark/solution.ail", *files],
                       cwd=REPO, capture_output=True, text=True, timeout=900)
    out = {}
    for ln in r.stdout.splitlines():
        d = json.loads(ln)
        out[d["id"]] = d
    return out


def fake_ws(rnd, bid):
    """A workspace path shaped like the harness's, different per episode (so the model copies it)."""
    bid = re.sub(r"[^\w]+", "_", bid)
    a = "".join(rnd.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(2))
    b = "".join(rnd.choice("abcdefghijklmnopqrstuvwxyz0123456789") for _ in range(30))
    pid, ts = rnd.randint(1000, 99999), f"2026{rnd.randint(1, 12):02d}{rnd.randint(1, 28):02d}_{rnd.randint(0, 235959):06d}"
    return f"/var/folders/{a}/{b}/T/ailang_eval/{bid}_pi-gemma4-12b-ft_{ts}_{pid}_t1_{rnd.randint(10**8, 10**9)}/{bid}_ailang_pi_{pid}"


def run(code, caps, entry="main"):
    with tempfile.TemporaryDirectory() as td:
        os.makedirs(os.path.join(td, "benchmark"))
        with open(os.path.join(td, "benchmark", "solution.ail"), "w") as f:
            f.write(code)
        try:
            r = subprocess.run([AILANG, "run", "--entry", entry, "--caps", caps, "benchmark/solution.ail"],
                               cwd=td, capture_output=True, text=True, timeout=60)
        except subprocess.TimeoutExpired:
            return None
        return r.stdout if r.returncode == 0 else None


def check_out(code):
    with tempfile.TemporaryDirectory() as td:
        os.makedirs(os.path.join(td, "benchmark"))
        with open(os.path.join(td, "benchmark", "solution.ail"), "w") as f:
            f.write(code)
        r = subprocess.run([AILANG, "check", "benchmark/solution.ail"], cwd=td, capture_output=True, text=True, timeout=60)
        return r.returncode == 0, (r.stderr + r.stdout).strip()


def run_cmd(caps):
    return f"ailang run --entry main --caps {caps} benchmark/solution.ail"


def edit_or_write(path, old, new):
    """One contiguous block of changed LINES -> an `edit` call (pi's edits[] schema); else rewrite."""
    a, b = old.splitlines(keepends=True), new.splitlines(keepends=True)
    ops = [o for o in difflib.SequenceMatcher(None, a, b, autojunk=False).get_opcodes() if o[0] != "equal"]
    if ops:
        i1, i2, j1, j2 = ops[0][1], ops[-1][2], ops[0][3], ops[-1][4]
        old_txt, new_txt = "".join(a[i1:i2]), "".join(b[j1:j2])
        ot, nt = old_txt.rstrip("\n"), new_txt.rstrip("\n")
        # pi requires a unique oldText; and the edit must reproduce the fixed file exactly
        if ot and (i2 - i1) <= 12 and old.count(ot) == 1 and old.replace(ot, nt) == new:
            return ({"name": "edit", "args": {"path": path, "edits": [{"oldText": ot, "newText": nt}]}},
                    f"Successfully replaced 1 block(s) in {path}.")
    return {"name": "write", "args": {"path": path, "content": new}}, f"Successfully wrote to {path}"


COMMON = {"println", "print", "show", "map", "filter", "Some", "None", "Ok", "Err", "length", "foldl"}
_LOOKUP = {}


def lookup(cmd):
    """Real output of a lookup command (cached): what the agent would see."""
    if cmd not in _LOOKUP:
        r = subprocess.run(["/bin/bash", "-c", cmd], cwd=tempfile.gettempdir(), capture_output=True, text=True, timeout=60)
        out = "\n".join(ln for ln in (r.stdout + r.stderr).splitlines() if not ln.startswith("note: ignoring"))
        _LOOKUP[cmd] = out.strip() + "\n"
    return _LOOKUP[cmd]


def lookup_steps(rnd, code):
    """Habit 3 (look up before guessing): targeted signature lookups for functions the solution
    imports, and sometimes an examples search. Only the less common names; outputs are real."""
    pairs = []
    for mod, names in re.findall(r"^import\s+(std/[\w/]+)\s*\(([^)]*)\)", code, re.M):
        for n in names.split(","):
            n = n.strip().split(" as ")[0].strip()
            if n and n[0].islower() and n not in COMMON:
                pairs.append((mod, n))
    rnd.shuffle(pairs)
    steps = []
    for mod, n in pairs[: rnd.choice([1, 2])]:
        cmd = f"ailang docs --all-functions {mod}.{n}"
        out = lookup(cmd)
        if f"{mod}.{n}" in out:
            steps.append({"call": {"name": "bash", "args": {"command": cmd}}, "result": out, "habit": "lookup",
                          "lookup": f"{mod}.{n}"})
    if pairs and rnd.random() < 0.4:
        mod, n = pairs[0]
        cmd = f'ailang examples search "{n}" | head -12'
        out = lookup(cmd)
        if "Found" in out:
            steps.append({"call": {"name": "bash", "args": {"command": cmd}}, "result": out, "habit": "examples",
                          "lookup": n})
    return steps


def solve_episode(rnd, task, code, stdout):
    """read -> [habit: targeted lookups] -> write -> [habit: check] -> run -> done."""
    ws = fake_ws(rnd, task["id"])
    path, rel = f"{ws}/benchmark/solution.ail", "benchmark/solution.ail"
    caps = ",".join(task["caps"] or ["IO"])
    steps = [{"call": {"name": "read", "args": {"path": path}}, "result": TEMPLATE}]
    if rnd.random() < 0.5:
        steps += lookup_steps(rnd, code)
    steps.append({"call": {"name": "write", "args": {"path": path, "content": code}}, "result": f"Successfully wrote to {path}"})
    if rnd.random() < 0.7:
        ok, out = check_out(code)
        steps.append({"call": {"name": "bash", "args": {"command": f"ailang check {rel}"}}, "result": out, "habit": "check"})
    steps.append({"call": {"name": "bash", "args": {"command": run_cmd(caps)}}, "result": stdout})
    return {"kind": "solve", "benchmark": task["id"], "task_prompt": task["task_prompt"].replace(WS_TOKEN, ws),
            "steps": steps, "final": rnd.choice(DONE_LINES).format(rel=rel)}


def repair_episode(rnd, task, pair, stdout):
    ws = fake_ws(rnd, task["id"])
    path, rel = f"{ws}/benchmark/solution.ail", "benchmark/solution.ail"
    caps = ",".join(task["caps"] or ["IO"])
    steps = [{"call": {"name": "read", "args": {"path": path}}, "result": TEMPLATE},
             {"call": {"name": "write", "args": {"path": path, "content": pair["old"]}}, "result": f"Successfully wrote to {path}"}]
    chain = pair.get("chain") or [{"code": pair["old"], "error": pair["error"], "fixed": pair["fixed"]}]
    cur = pair["old"]
    for link in chain:
        ok, err = check_out(cur)
        if ok:  # an output-mismatch round: show the run, not a check
            steps.append({"call": {"name": "bash", "args": {"command": run_cmd(caps)}}, "result": run(cur, caps) or ""})
        else:
            steps.append({"call": {"name": "bash", "args": {"command": f"ailang check {rel}"}}, "result": err})
        call, res = edit_or_write(path, cur, link["fixed"])
        steps.append({"call": call, "result": res})
        cur = link["fixed"]
    ok, _ = check_out(cur)
    if not ok:
        return None
    steps.append({"call": {"name": "bash", "args": {"command": run_cmd(caps)}}, "result": stdout})
    tp = task["task_prompt"].replace(WS_TOKEN, ws)
    if pair.get("expected_stdout") is not None and task["expected"] and pair["expected_stdout"] != task["expected"]:
        tp = tp.replace(task["expected"], pair["expected_stdout"])  # the behaviour the program had when banked
    return {"kind": "repair", "benchmark": task["id"], "task_prompt": tp, "steps": steps,
            "final": rnd.choice(DONE_LINES).format(rel=rel)}


# Habit 5 (pilot-3): after an error, and above all after a FIX THAT FAILED, read the reference
# section the error points to instead of guessing again. Error shape -> teaching-prompt section.
# The section text is real `ailang prompt` output; a lookup is kept only if it is non-empty.
SECTIONS = [
    (r"PAR_MATCH_ARROW|next token to be =>", "Pattern Matching", 45),
    (r"PAR015|bare assignment|PAR017|only valid inside|next token to be \{|next token to be else|got \} instead|"
     r"in expression: \}", "Let Bindings: Block Style vs Expression Style", 42),
    (r"lambda parameter", "Syntax Reference", 22),
    (r"INTERP_END|interpolation", "String Interpolation", 28),
    (r"in expression: as\b|Fractional", "Number Conversions", 17),
    (r"`\+\+`", "String and List Concatenation", 18),
    (r"MOD010|doesn't match file path", "Required Program Structure", 20),  # rule 1: module matches the path
    (r"belongs to ADT 'Result', not 'Option'", "Constructing Option Values", 40),
    (r"\bPAR[_A-Z0-9]+", "What AILANG Does NOT Have", 22),
]


def signature_lookup(err, code):
    """A type mismatch in a call to an imported stdlib function: look up its exact signature
    (habit 3 applied after the fact). Only for names the code imports from a std/ module."""
    m = re.search(r"function application of (\w+) at", err)
    if not m:
        return None
    name = m.group(1)
    for mod, names in re.findall(r"^import\s+(std/[\w/]+)\s*\(([^)]*)\)", code or "", re.M):
        if name in [n.strip().split(" as ")[0].strip() for n in names.split(",")]:
            cmd = f"ailang docs --all-functions {mod}.{name}"
            out = lookup(cmd)
            if f"{mod}.{name}" in out:
                line = next((ln for ln in err.splitlines() if "function application of" in ln), "")
                return {"call": {"name": "bash", "args": {"command": cmd}}, "result": out, "habit": "reference",
                        "lookup": f"{mod}.{name}", "lookup_kind": "signature", "about": re.sub(r"\s+", " ", line.strip())[:200]}
    return None


def section_lookup(err, code=None):
    """The first error LINE that a specific section covers (else the first parse error, with the
    general "does not have" table). Only error lines count: the Hint text mentions other constructs.
    Returns the lookup step, recording which error line it is about."""
    lines = [ln.strip() for ln in err.splitlines()
             if re.search(r"\b(PAR\w*|TC_\w+|IMP\d+|MOD\d+) at |MOD\d+: |type error|expected next token|unexpected token", ln)]
    for specific in (True, False):
        for ln in lines:
            for pat, sec, n in (SECTIONS[:-1] if specific else SECTIONS[-1:]):
                if not re.search(pat, ln):
                    continue
                cmd = f'ailang prompt | grep -A{n} "^## {sec}"'
                out = lookup(cmd)
                if out.strip().startswith("## " + sec):
                    return {"call": {"name": "bash", "args": {"command": cmd}}, "result": out, "habit": "reference",
                            "lookup": sec, "about": re.sub(r"\s+", " ", ln)[:200]}
    return signature_lookup(err, code)


def onpolicy_episode(rnd, task, pair, stdout, p_first_lookup=0.35):
    """The fine-tune's OWN failing attempt (no loss) -> the real error -> [its own failed fix
    (no loss) -> the same kind of error] -> look up the reference section -> the verified fix
    chain (loss) -> run -> done."""
    ws = fake_ws(rnd, task["id"])
    path, rel = f"{ws}/benchmark/solution.ail", "benchmark/solution.ail"
    caps = ",".join(task["caps"] or ["IO"])

    def observe(code):
        ok, err = check_out(code)
        if ok:
            return {"call": {"name": "bash", "args": {"command": run_cmd(caps)}}, "result": run(code, caps) or ""}, err
        return {"call": {"name": "bash", "args": {"command": f"ailang check {rel}"}}, "result": err}, err

    steps = [{"call": {"name": "read", "args": {"path": path}}, "result": TEMPLATE}]
    start = pair["old"]
    # The model's own earlier attempts at this same kind of error, oldest first (pilot-4). Two or
    # more is a loop: fix after fix that never worked. All of them are context, never a target.
    hist = pair.get("history") or ([pair["prev"]] if pair.get("prev") else [])
    if hist:
        steps.append({"call": {"name": "write", "args": {"path": path, "content": hist[0]["code"]}},
                      "result": f"Successfully wrote to {path}", "no_loss": True})
        cur_h = hist[0]["code"]
        for h in [x["code"] for x in hist[1:]] + [start]:
            obs, _ = observe(cur_h)
            steps.append(obs)
            call, res = edit_or_write(path, cur_h, h)
            steps.append({"call": call, "result": res, "no_loss": True, "failed_fix": True})
            cur_h = h
    else:
        steps.append({"call": {"name": "write", "args": {"path": path, "content": start}},
                      "result": f"Successfully wrote to {path}", "no_loss": True})
    obs, err = observe(start)
    steps.append(obs)
    look = section_lookup(err, start) if (hist or rnd.random() < p_first_lookup) else None
    if look:
        look["after_failed"] = len(hist)  # failed fixes of this kind before the lookup
        steps.append(look)
    cur = start
    for k, link in enumerate(pair["chain"]):
        if k:
            obs, _ = observe(cur)
            steps.append(obs)
        call, res = edit_or_write(path, cur, link["fixed"])
        steps.append({"call": call, "result": res})
        cur = link["fixed"]
    ok, _ = check_out(cur)
    if not ok:
        return None
    steps.append({"call": {"name": "bash", "args": {"command": run_cmd(caps)}}, "result": stdout})
    return {"kind": "onpolicy", "benchmark": task["id"], "task_prompt": task["task_prompt"].replace(WS_TOKEN, ws),
            "steps": steps, "final": rnd.choice(DONE_LINES).format(rel=rel), "error_kind": pair.get("kind"),
            "loop": len(hist) >= 2}


TOP_DECL = re.compile(r"^(?:export\s+)?(?:pure\s+)?(?:func|type)\s", re.M)
MAIN_SIG = re.compile(r"^(?:export\s+)?func\s+main\s*\(\s*\)\s*(?:->\s*\(\s*\))?\s*(?:!\s*\{[^}]*\})?", re.M)


def build_split(code):
    """Split a verified program into (part1, rest, final) for a build-up episode, or None.
    part1 = header + the first ~half of the non-main declarations + a stub `main` (body `()`);
    final = header + every non-main declaration in order + the real `main` LAST, so that one
    `edit` (stub main -> rest) turns part1 into final. Caller verifies part1 checks and final runs."""
    starts = [m.start() for m in TOP_DECL.finditer(code)]
    if len(starts) < 3:
        return None
    header = code[: starts[0]]
    decls = [code[a:b].rstrip("\n") + "\n" for a, b in zip(starts, starts[1:] + [len(code)])]
    mains = [d for d in decls if re.match(r"(?:export\s+)?func\s+main\b", d)]
    if len(mains) != 1:
        return None
    sig = MAIN_SIG.match(mains[0])
    if not sig:
        return None  # main returns a value: no `()` stub
    others = [d for d in decls if d is not mains[0]]
    total, acc, k = sum(len(d) for d in others), 0, 0
    while k < len(others) - 1 and acc + len(others[k]) <= total * 0.6:
        acc += len(others[k])
        k += 1
    if k == 0:
        return None
    stub = sig.group(0).rstrip() + " {\n  ()\n}\n"
    part1 = header + "\n".join(others[:k]) + "\n" + stub
    rest = "\n".join(others[k:]) + "\n" + mains[0]
    final = header + "\n".join(others[:k]) + "\n" + rest
    return part1, stub, rest, final


def build_episode(rnd, task, code, stdout):
    """Habit (pilot-4): build a long file up in steps instead of one huge write. Pilot-3's long
    `write` calls sometimes ran away or ended malformed and were dropped by the tool-call parser.
    read -> write helpers + stub main -> ailang check -> edit: stub -> rest + real main -> run."""
    sp = build_split(code)
    if not sp:
        return None
    part1, stub, rest, final = sp
    ok, out = check_out(part1)
    caps = ",".join(task["caps"] or ["IO"])
    if not ok or final.count(stub.rstrip("\n")) or part1.count(stub.rstrip("\n")) != 1 or \
            part1.replace(stub.rstrip("\n"), rest.rstrip("\n")) != final:  # the edit must reproduce final exactly
        return None
    got = run(final, caps)
    if got is None or got.rstrip("\n") != stdout.rstrip("\n"):
        return None
    ws = fake_ws(rnd, task["id"])
    path, rel = f"{ws}/benchmark/solution.ail", "benchmark/solution.ail"
    steps = [{"call": {"name": "read", "args": {"path": path}}, "result": TEMPLATE},
             {"call": {"name": "write", "args": {"path": path, "content": part1}},
              "result": f"Successfully wrote to {path}", "habit": "build"},
             {"call": {"name": "bash", "args": {"command": f"ailang check {rel}"}}, "result": out, "habit": "check"},
             {"call": {"name": "edit", "args": {"path": path, "edits": [{"oldText": stub.rstrip("\n"), "newText": rest.rstrip("\n")}]}},
              "result": f"Successfully replaced 1 block(s) in {path}.", "habit": "build_rest"},
             {"call": {"name": "bash", "args": {"command": run_cmd(caps)}}, "result": got}]
    return {"kind": "build", "benchmark": task["id"], "task_prompt": task["task_prompt"].replace(WS_TOKEN, ws),
            "steps": steps, "final": rnd.choice(DONE_LINES).format(rel=rel)}


def example_task(header, expected, caps, ws):
    path = f"{ws}/benchmark/solution.ail"
    return ("You are solving an AILANG benchmark.\n\n## Task\n\n" + header +
            f"\n\n\n**IMPORTANT: Write your complete solution to: {path}**\n\nExpected output:\n{expected}\n\n"
            "## Requirements\n\n- Write your solution to: **" + path + "**\n"
            "- Your solution must produce output matching the expected output **exactly**\n"
            f"- Read {path} file first so you know the template to write to\n\n"
            f"## Verification Steps (REQUIRED!)\n\n1. **Run your solution:** `{run_cmd(caps)}`\n"
            "2. Compare the output with the expected output; fix and re-run until it matches.\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--lang-data", default="~/models/finetune/data-v1")
    ap.add_argument("--stale", default="~/models/finetune/stale")
    ap.add_argument("--per-bench", type=int, default=25)
    ap.add_argument("--onpolicy", default="", help="on-policy repair pairs (stale_fix.py --prompt onpolicy output); "
                                                   "comma-separate several rounds to pool them")
    ap.add_argument("--build-min-chars", type=int, default=0,
                    help="pilot-4: also make a build-up episode from every solve program at least this long (0 = off)")
    ap.add_argument("--workers", type=int, default=8)
    ap.add_argument("--seed", type=int, default=11)
    a = ap.parse_args()
    out = os.path.expanduser(a.out)
    os.makedirs(out, exist_ok=True)
    rnd = random.Random(a.seed)
    held = load_heldout()

    evals = [json.loads(l) for l in open(os.path.join(os.path.expanduser(a.lang_data), "eval.jsonl"))]
    pairs = []
    for fn in ("stale_pairs.jsonl", "stale_pairs_opus.jsonl"):
        p = os.path.join(os.path.expanduser(a.stale), fn)
        if os.path.exists(p):
            pairs += [json.loads(l) for l in open(p) if '"ok": true' in l]
    stale_exp = {}
    sp = os.path.join(os.path.expanduser(a.stale), "stale.jsonl")
    if os.path.exists(sp):
        from extract_lang import norm_hash
        for l in open(sp):
            d = json.loads(l)
            stale_exp[norm_hash(d["code"])] = d.get("expected_stdout")
        for p in pairs:
            p["expected_stdout"] = stale_exp.get(p.get("code_hash"))

    def ok_spec(bid):
        try:
            b = yaml.safe_load(open(os.path.join(REPO, "benchmarks", f"{bid}.yml")))
        except (OSError, yaml.YAMLError):
            return False
        return bool(b) and bid not in held and not any(k in b for k in UNVERIFIABLE)

    onp = []
    if a.onpolicy:
        seen_h = set()
        for fn in a.onpolicy.split(","):
            for l in open(os.path.expanduser(fn)):
                if '"ok": true' not in l:
                    continue
                p = json.loads(l)
                if p["code_hash"] not in seen_h:  # the same broken state in two rounds counts once
                    seen_h.add(p["code_hash"])
                    onp.append(p)
    bids = sorted({e["benchmark"] for e in evals} | {p["benchmark"] for p in pairs} | {p["benchmark"] for p in onp})
    bids = [b for b in bids if ok_spec(b)]
    tasks = render_tasks(bids)
    rep = {"benchmarks": len(tasks)}

    # ---- solve episodes (cap per benchmark, prefer distinct authors)
    per = {}
    for e in evals:
        if e["benchmark"] in tasks:
            per.setdefault(e["benchmark"], []).append(e)
    jobs = []
    for bid, items in per.items():
        rnd.shuffle(items)
        jobs += [(bid, it["code"]) for it in items[: a.per_bench]]

    def do_solve(j):
        bid, code = j
        t = tasks[bid]
        got = run(code, ",".join(t["caps"] or ["IO"]))
        if got is None or got.rstrip("\n") != (t["expected"] or "").rstrip("\n"):
            return None  # the program no longer prints what today's task asks for
        return (bid, code, got)

    with cf.ThreadPoolExecutor(a.workers) as ex:
        solved = [r for r in ex.map(do_solve, jobs) if r]
    eps = [solve_episode(rnd, tasks[b], c, g) for b, c, g in solved]
    rep["solve"] = {"candidates": len(jobs), "kept": len(eps)}

    # ---- repair episodes
    def do_repair(p):
        t = tasks.get(p["benchmark"])
        if not t:
            return None
        got = run(p["fixed"], ",".join(t["caps"] or ["IO"]))
        want = p.get("expected_stdout") if p.get("expected_stdout") is not None else t["expected"]
        if got is None or got.rstrip("\n") != (want or "").rstrip("\n"):
            return None
        return (p, got)

    with cf.ThreadPoolExecutor(a.workers) as ex:
        rp = [r for r in ex.map(do_repair, pairs) if r]
    reps = [e for e in (repair_episode(rnd, tasks[p["benchmark"]], p, g) for p, g in rp) if e]
    rep["repair"] = {"candidates": len(pairs), "kept": len(reps)}

    # ---- on-policy repair episodes (pilot-3)
    with cf.ThreadPoolExecutor(a.workers) as ex:
        op = [r for r in ex.map(do_repair, onp) if r]
    oeps = [e for e in (onpolicy_episode(rnd, tasks[p["benchmark"]], p, g) for p, g in op) if e]
    rep["onpolicy"] = {"candidates": len(onp), "kept": len(oeps), "loop": sum(1 for e in oeps if e.get("loop")),
                       "with_failed_fix": sum(1 for e in oeps if any(s.get("failed_fix") for s in e["steps"])),
                       "with_lookup": sum(1 for e in oeps if any(s.get("habit") == "reference" for s in e["steps"]))}
    reps += oeps

    # ---- build-up episodes (pilot-4): long programs written in two steps
    if a.build_min_chars:
        longs = [(b, c, g) for b, c, g in solved if len(c) >= a.build_min_chars]
        beps = [e for e in (build_episode(rnd, tasks[b], c, g) for b, c, g in longs) if e]
        rep["build"] = {"candidates": len(longs), "kept": len(beps)}
        eps += beps

    # ---- example episodes (manifest-pinned stdout; retargeted module)
    man = json.load(open(os.path.join(REPO, "examples", "manifest.json")))
    exs = []
    for ent in man["examples"]:
        exp = (ent.get("expected") or {}).get("stdout")
        if not exp or ent.get("status") != "working":
            continue
        p = os.path.join(REPO, "examples", ent["path"])
        try:
            src = open(p).read()
        except OSError:
            continue
        m = re.search(r"^\s*module\s+\S+", src, re.M)
        hdr = re.findall(r"^--\s?(.*)$", src[: m.start()] if m else "", re.M)
        header = " ".join(h.strip() for h in hdr if h.strip()).replace("Task:", "").strip()
        if not m or len(header) < 30:
            continue
        code = src[: m.start()] + "module benchmark/solution" + src[m.end():]
        caps = "IO"
        for c in ("FS", "Clock", "Rand", "Env", "Process", "Net", "Trace", "Stream"):
            if f"! {{{c}" in src or f", {c}" in src:
                caps += "," + c
        exs.append((ent["path"], header, exp, caps, code))

    def do_example(x):
        path, header, exp, caps, code = x
        got = run(code, caps)
        return x if got is not None and got.rstrip("\n") == exp.rstrip("\n") else None

    with cf.ThreadPoolExecutor(a.workers) as ex:
        good = [x for x in ex.map(do_example, exs) if x]
    xeps = []
    for path, header, exp, caps, code in good:
        ws = fake_ws(rnd, "example")
        t = {"id": "example:" + path, "caps": caps.split(","), "expected": exp, "task_prompt": example_task(header, exp, caps, ws)}
        e = solve_episode(rnd, dict(t, task_prompt=t["task_prompt"].replace(ws, WS_TOKEN)), code, exp)
        e["kind"] = "example"
        xeps.append(e)
    rep["example"] = {"candidates": len(exs), "kept": len(xeps)}

    with open(os.path.join(out, "episodes.jsonl"), "w") as f:
        for e in eps + reps + xeps:
            f.write(json.dumps(e) + "\n")
    json.dump(rep, open(os.path.join(out, "report.json"), "w"), indent=1)
    print(json.dumps(rep, indent=1))


if __name__ == "__main__":
    main()
