#!/usr/bin/env bash
# Smoke-test a hosted AILANG executor MCP endpoint (M-AILANG-EXECUTOR-GE P1).
#
#   executor_tools/smoke.sh https://<service>.run.app/mcp/
#   executor_tools/smoke.sh http://127.0.0.1:8791/mcp/
#
# The dev service is IAM-only: pass an identity token via MCP_TOKEN, e.g.
#   MCP_TOKEN=$(gcloud auth print-identity-token) executor_tools/smoke.sh <url>/mcp/
#
# Checks: tools/list carries the five executor tools plus docs + submit_feedback;
# ailang_run prints; an FS-using program is refused by policy; an infinite loop
# is killed at the policy timeout; ailang_check reports a type error; ailang_eval
# evaluates. Never calls submit_feedback (it writes to the message store).
set -euo pipefail
URL="${1:?usage: smoke.sh <mcp-url>}"

python3 -I - "$URL" <<'PY'
import json, os, sys, time, urllib.request

url = sys.argv[1]
token = os.environ.get("MCP_TOKEN", "")
fails = 0

def rpc(method, params=None, timeout=90):
    body = {"jsonrpc": "2.0", "id": 1, "method": method}
    if params is not None:
        body["params"] = params
    headers = {"content-type": "application/json",
               "accept": "application/json, text/event-stream"}
    if token:
        headers["authorization"] = "Bearer " + token
    req = urllib.request.Request(url, data=json.dumps(body).encode(), headers=headers)
    text = urllib.request.urlopen(req, timeout=timeout).read().decode()
    return json.loads(text[text.find("{"):])

def call(name, args):
    r = rpc("tools/call", {"name": name, "arguments": args})["result"]
    return json.loads(r["content"][0]["text"])

def check(label, ok, detail=""):
    global fails
    print(("PASS " if ok else "FAIL ") + label + ("" if ok else f"  -- {detail}"))
    if not ok:
        fails += 1

names = {t["name"] for t in rpc("tools/list")["result"]["tools"]}
want = {"ailang_prompt", "ailang_check", "ailang_run", "ailang_eval", "ailang_builtins",
        "docs_search", "submit_feedback"}
check("tools/list has executor + docs + feedback tools", want <= names, sorted(want - names))

hello = 'module main\nimport std/io (println)\nexport func main() -> () ! {IO} = println("hello")\n'
r = call("ailang_run", {"code": hello})
check("ailang_run prints", r.get("ok") and r.get("stdout") == "hello\n", r)

fs = 'module main\nimport std/fs (readFile)\nimport std/io (println)\nexport func main() -> () ! {IO, FS} = println(readFile("/etc/passwd"))\n'
r = call("ailang_run", {"code": fs})
check("FS program refused by policy", not r.get("ok") and "policy_violation" in r.get("stdout", ""), r)

loop = 'module main\nexport func spin(n: int) -> int = spin(n + 1)\nexport func main() -> int = spin(0)\n'
t0 = time.time()
r = call("ailang_run", {"code": loop})
check("infinite loop killed by timeout", not r.get("ok") and "timeout" in r.get("stderr", ""), r)
print(f"     (loop returned after {time.time() - t0:.1f}s)")

bad = 'module main\nexport func f(x: int) -> string = x\n'
r = call("ailang_check", {"code": bad})
check("ailang_check reports a type error", not r.get("ok") and '"passed": false' in r.get("stdout", ""), r)

r = call("ailang_eval", {"expression": "1 + 2 * 3"})
check("ailang_eval evaluates", r.get("stdout") == "7\n", r)

print("OK" if fails == 0 else f"{fails} FAILED")
sys.exit(1 if fails else 0)
PY
