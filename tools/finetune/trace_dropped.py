#!/usr/bin/env python3
"""Count tool calls the model generated but the agent never received (pilot-4, D-FT-15/16).

A traced eval (tools/finetune/trace_proxy.py) records every request and the raw SSE response.
A response that generated many tokens yet carries no tool call and no visible text is a call the
server's tool-call parser dropped: a `write` that ran away or ended malformed. The agent harness
reads it as the end of the run, so the banked row shows a quiet, early failure. Measured
2026-11-10: pilot-3 1.8% of responses, pilot-2 0.05%.

  python3 -I tools/finetune/trace_dropped.py TRACE.jsonl [--min-tokens 500] [--json]

Per benchmark: requests, dropped responses, and whether the run ENDED on a drop.
"""
import argparse
import collections
import json
import re

WS_RE = re.compile(r"ailang_eval/((.+?)_pi-gemma4[\w.-]*?_\d{8}_\d{6}[\w-]*)")  # one workspace = one run


def scan(resp):
    """(has_tool_call, visible_text_chars, completion_tokens) from a raw SSE response."""
    tool, text, comp = False, 0, None
    for line in str(resp or "").splitlines():
        if not line.startswith("data: {"):
            continue
        try:
            d = json.loads(line[6:])
        except json.JSONDecodeError:
            continue
        for ch in d.get("choices", []):
            dl = ch.get("delta") or {}
            tool = tool or bool(dl.get("tool_calls"))
            text += len(dl.get("content") or "")
        if d.get("usage"):
            comp = d["usage"].get("completion_tokens")
    return tool, text, comp


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("trace")
    ap.add_argument("--min-tokens", type=int, default=500)
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args()
    per = collections.defaultdict(lambda: {"requests": 0, "dropped": 0, "last_dropped": False})
    tot = drop = 0
    with open(a.trace) as f:
        for line in f:
            r = json.loads(line)
            rq = r.get("request")
            if not isinstance(rq, dict) or "messages" not in rq:
                continue
            m = WS_RE.search(json.dumps(rq["messages"][:2]))
            run = m.group(1) if m else "?"
            b = f"{m.group(2)}#{run[-6:]}" if m else "?"
            tool, text, comp = scan(r.get("response"))
            d = (not tool) and text == 0 and (comp or 0) >= a.min_tokens
            per[b]["requests"] += 1
            per[b]["dropped"] += d
            per[b]["last_dropped"] = d  # requests arrive in order within a run (--parallel 1)
            tot += 1
            drop += d
    out = {"requests": tot, "dropped": drop, "rate": round(drop / tot, 4) if tot else None,
           "runs_ended_on_drop": sorted(b for b, v in per.items() if v["last_dropped"]),
           "per_benchmark": {b: v for b, v in sorted(per.items()) if v["dropped"]}}
    if a.json:
        print(json.dumps(out, indent=1))
    else:
        print(f"{drop}/{tot} responses dropped ({100 * drop / max(tot, 1):.2f}%); "
              f"runs that ended on a drop: {len(out['runs_ended_on_drop'])} {out['runs_ended_on_drop']}")


if __name__ == "__main__":
    main()
