#!/usr/bin/env bash
# Summarise how callers (Gemini Enterprise, Claude Code, ...) use the hosted
# AILANG executor, from the one "executor_call" log line each tool call emits
# (executor_tools/executor.ail).
#
#   executor_tools/usage.sh            # last 24h, dev
#   executor_tools/usage.sh 7d         # any gcloud --freshness value
#   PROJECT=ailang-multivac EXECUTOR_SERVICE=ailang-executor-mcp executor_tools/usage.sh
#
# Answers the questions that drive teaching changes: which tools get called,
# how often generated code fails to check or run (and with what first error),
# how many attempts a caller needs before code passes (same code_sha repeated
# = retrying unchanged code), and whether ailang_prompt is called at all.
set -euo pipefail
FRESHNESS="${1:-24h}"
PROJECT="${PROJECT:-ailang-multivac-dev}"
SERVICE="${EXECUTOR_SERVICE:-ailang-dev-executor-mcp}"

SUMMARY=$(cat <<'PY'
import collections, json, sys
rows = [e["jsonPayload"] | {"ts": e["timestamp"]} for e in json.load(sys.stdin)]
rows.sort(key=lambda r: r["ts"])
if not rows:
    print("no executor_call lines in window"); sys.exit(0)
first, last = rows[0]["ts"][:19], rows[-1]["ts"][:19]
print(f"{len(rows)} calls, {first} .. {last} UTC\n")

by_tool = collections.defaultdict(collections.Counter)
dur = collections.defaultdict(list)
for r in rows:
    by_tool[r["tool"]][r.get("outcome", "?")] += 1
    dur[r["tool"]].append(r.get("duration_ms", 0))
print("%-17s%6s%9s  outcomes" % ("tool", "calls", "p50 ms"))
for tool, oc in sorted(by_tool.items(), key=lambda kv: -sum(kv[1].values())):
    d = sorted(dur[tool]); p50 = d[len(d) // 2]
    print(f"{tool:<17}{sum(oc.values()):>6}{p50:>9}  " + ", ".join(f"{k} {v}" for k, v in oc.most_common()))

code = [r for r in rows if r["tool"] in ("ailang_check", "ailang_run", "ailang_eval")]
if code:
    ok = sum(r.get("outcome") == "ok" for r in code)
    print(f"\ncode calls: {len(code)}, pass rate {ok / len(code):.0%}")
    shas = collections.Counter(r.get("code_sha") for r in code)
    repeats = {s: n for s, n in shas.items() if n > 1}
    if repeats:
        print(f"identical code resubmitted: {len(repeats)} snippets, {sum(repeats.values())} calls")

errs = collections.Counter(r["error"][:120] for r in rows if r.get("error"))
if errs:
    print("\ntop first errors:")
    for e, n in errs.most_common(10):
        print(f"  {n:>4}  {e}")

prompt = by_tool.get("ailang_prompt")
if code and not prompt:
    print("\nWARNING: code tools called but ailang_prompt never was — the connector description is not steering the caller.")
PY
)

gcloud logging read \
  "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${SERVICE}\" AND jsonPayload.event=\"executor_call\"" \
  --project="${PROJECT}" --freshness="${FRESHNESS}" --limit=5000 --format=json |
python3 -I -c "$SUMMARY"
