#!/usr/bin/env python3
"""On-policy repair items from a traced fine-tune run (pilot-3; pilot-2.md §Diagnosis).

Input: the trace_proxy.py JSONL from collect_onpolicy.sh. For each agent session (one
workspace) the longest request holds the whole conversation; the streamed response to it
holds the last action. Every write/edit is replayed to rebuild each file state the model
produced, and each state is re-checked here with the current `ailang` (check, then run against
the benchmark's expected stdout), never trusting the transcript's own tool output.

Output (stale_fix.py `fix --items` format, one line per distinct failing state):
  {benchmark, code, error, expected_stdout, task, kind, prev}
  prev = {code, error} when this state was the model's own attempt to fix a previous failing
         state with the same error kind (a failed fix: the moment to look something up).

  python3 -I tools/finetune/onpolicy_pairs.py --trace ~/models/finetune/onpolicy/trace.jsonl \
      --out ~/models/finetune/onpolicy/items.jsonl
"""
import argparse
import collections
import concurrent.futures as cf
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from extract_lang import load_heldout, norm_hash  # noqa: E402
from stale_fix import check, run_ok, spec  # noqa: E402

WS_RE = re.compile(r"ailang_eval/(.+?)_pi-gemma4-12b-ft\d[\w-]*?_\d{8}_\d{6}")


def args_of(tc):
    a = tc["function"].get("arguments")
    if isinstance(a, str):
        try:
            return json.loads(a)
        except ValueError:
            return None
    return a


def last_action(sse):
    """Tool calls in a streamed chat.completion response (OpenAI deltas)."""
    calls = collections.OrderedDict()
    for ln in (sse or "").splitlines():
        if not ln.startswith("data: ") or ln.strip() == "data: [DONE]":
            continue
        try:
            d = json.loads(ln[6:])
        except ValueError:
            continue
        for ch in d.get("choices") or []:
            for tc in (ch.get("delta") or {}).get("tool_calls") or []:
                c = calls.setdefault(tc.get("index", 0), {"function": {"name": "", "arguments": ""}})
                f = tc.get("function") or {}
                c["function"]["name"] += f.get("name") or ""
                c["function"]["arguments"] += f.get("arguments") or ""
    return list(calls.values())


def sessions(trace):
    best = {}
    for ln in open(trace):
        try:
            r = json.loads(ln)
        except ValueError:
            continue
        req = r.get("request")
        if not isinstance(req, dict) or not req.get("messages"):
            continue
        sysmsg = str(req["messages"][0].get("content"))
        m = WS_RE.search(sysmsg) or WS_RE.search(json.dumps(req["messages"][1].get("content")))
        if not m:
            continue
        key = sysmsg[sysmsg.find("Current working directory"):][:400]
        if key not in best or len(req["messages"]) > len(best[key][1]["messages"]):
            best[key] = (m.group(1), req, r.get("response"))
    return list(best.values())


def file_states(req, resp):
    """Replay write/edit calls on benchmark/solution.ail; stop at the first edit that does not apply."""
    calls = []
    for m in req["messages"]:
        if m.get("role") == "assistant":
            calls += m.get("tool_calls") or []
    calls += last_action(resp if isinstance(resp, str) else "")
    cur, states = None, []
    for tc in calls:
        name, a = tc["function"]["name"], args_of(tc)
        if not a or not str(a.get("path", "")).endswith("solution.ail"):
            continue
        if name == "write" and "content" in a:
            cur = a["content"]
        elif name == "edit" and cur is not None:
            for e in a.get("edits") or [a]:
                old, new = e.get("oldText"), e.get("newText")
                if old is None or new is None or cur.count(old) != 1:
                    return states  # desync (bash-side writes, or a failed edit): stop here
                cur = cur.replace(old, new)
        else:
            continue
        if not states or states[-1] != cur:
            states.append(cur)
    return states


def kind_of(err):
    m = re.search(r"\b(PAR[_A-Z0-9]+|TC_[A-Z0-9_]+|IMP\d+|MOD\d+)\b", err)
    if m:
        return m.group(1)
    for k in ("type unification failed", "undefined variable", "effect checking", "Fractional", "exhaustive",
              "output changed", "timeout"):
        if k in err:
            return k
    return "other"


def classify(code, b):
    ok, err = check(code)
    if not ok:
        return "\n".join(err.splitlines()[:12])
    good, got = run_ok(code, b)
    if good:
        return None
    exp = (b.get("expected_stdout") or "")
    return ("The program now compiles, but its output changed. It must print exactly:\n"
            f"{exp[:1500]}\n\nIt printed:\n{(got or '')[:1500]}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--trace", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--workers", type=int, default=8)
    a = ap.parse_args()
    held = load_heldout()
    sess = sessions(os.path.expanduser(a.trace))
    jobs, rep = [], collections.Counter()
    for bid, req, resp in sess:
        if bid in held:
            sys.exit(f"held-out benchmark {bid} in the trace: refusing")
        b = spec(bid)
        if not b:
            rep["no_spec"] += 1
            continue
        jobs.append((bid, b, file_states(req, resp)))
    rep["sessions"] = len(jobs)

    def do(job):
        bid, b, states = job
        return bid, b, [(s, classify(s, b)) for s in states]

    seen, items = set(), []
    with cf.ThreadPoolExecutor(a.workers) as ex:
        for bid, b, res in ex.map(do, jobs):
            rep["states"] += len(res)
            for i, (code, err) in enumerate(res):
                if err is None:
                    rep["passing_states"] += 1
                    continue
                h = norm_hash(code)
                if h in seen:
                    rep["duplicate"] += 1
                    continue
                seen.add(h)
                prev = None
                if i and res[i - 1][1] is not None and kind_of(res[i - 1][1]) == kind_of(err):
                    prev = {"code": res[i - 1][0], "error": res[i - 1][1]}
                    rep["failed_fix"] += 1
                task = (b.get("task_prompt") or b.get("prompt") or "").replace("<LANG>", "AILANG")
                items.append({"benchmark": bid, "code": code, "error": err, "expected_stdout": b.get("expected_stdout"),
                              "task": task, "kind": kind_of(err), "prev": prev, "state_index": i})
    with open(os.path.expanduser(a.out), "w") as f:
        for it in items:
            f.write(json.dumps(it) + "\n")
    rep["items"] = len(items)
    rep["kinds"] = collections.Counter(it["kind"] for it in items).most_common(25)
    print(json.dumps(rep, indent=1, default=str))


if __name__ == "__main__":
    main()
