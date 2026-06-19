#!/usr/bin/env python3
"""Gate 3 (CHEAP-CONFIRM) — replay a captured /v1 request with ONE field changed and
report engage (tool_calls) vs disengage, N samples. Confirms a parameter hypothesis in
~1 minute BEFORE any plumbing or A/B.

  cheap_confirm.py <request.json> --set max_tokens=16384 [--set temperature=0.6] [-n 3]
  cheap_confirm.py --from-wire /tmp/motoko-analyzer-wire.jsonl --set max_tokens=16384

<request.json> is a raw /v1 chat-completions body (e.g. a wire-logged http_request body,
or pi's tap body). --from-wire pulls the first http_request out of a wire-log JSONL.
Runs against $OLLAMA_HOST/v1 (default http://localhost:11434). Acquire the rig lock around
this yourself if a rotation may be running (it's a few short calls).
"""
import argparse, copy, json, os, sys, urllib.request

def coerce(v):
    for cast in (int, float):
        try: return cast(v)
        except ValueError: pass
    if v.lower() in ("true", "false"): return v.lower() == "true"
    return v

def load_request(args):
    if args.from_wire:
        for line in open(args.from_wire):
            r = json.loads(line)
            if r.get("kind") == "http_request":
                return json.loads(r["body"])
        sys.exit("no http_request found in wire log")
    return json.load(open(args.request))

def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("request", nargs="?")
    ap.add_argument("--from-wire")
    ap.add_argument("--set", action="append", default=[], metavar="key=value")
    ap.add_argument("-n", type=int, default=3)
    ap.add_argument("--host", default=os.environ.get("OLLAMA_HOST", "http://localhost:11434").rstrip("/"))
    a = ap.parse_args()
    if not a.request and not a.from_wire:
        ap.error("provide <request.json> or --from-wire <wire.jsonl>")
    body = load_request(a)
    body["stream"] = False
    for k in ("stream_options", "store"):
        body.pop(k, None)
    changes = {}
    for kv in a.set:
        k, _, v = kv.partition("=")
        body[k] = coerce(v); changes[k] = body[k]
    url = a.host + "/v1/chat/completions"
    print(f"replaying with changes {changes or '(none)'} x{a.n} against {url}")
    eng = 0
    for i in range(a.n):
        req = urllib.request.Request(url, data=json.dumps(body).encode(), headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=600) as r:
            d = json.load(r)
        ch = d["choices"][0]; msg = ch["message"]
        tc = msg.get("tool_calls") or []
        reason = msg.get("reasoning") or msg.get("reasoning_content") or ""
        kind = "ENGAGE" if tc else "DISENGAGE"
        eng += 1 if tc else 0
        print(f"  trial {i+1}: finish={ch.get('finish_reason')} tool_calls={len(tc)} reasoning_chars={len(reason)} -> {kind}")
    print(f"\nENGAGE {eng}/{a.n}  ({'CONFIRMS the change helps' if eng > a.n // 2 else 'does NOT confirm'})")

if __name__ == "__main__":
    main()
