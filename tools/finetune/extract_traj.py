#!/usr/bin/env python3
"""Fine-tune extractor v0, trajectory half (m-eval-finetuning-data-pipeline §Data spec).

Turns agent sessions into mlx-lm chat samples in **pi's tool schema**
(bash / read / write / edit), teaching the behaviour 12B lacks in agent mode:
use tools, write the solution to the file, verify with `ailang`, fix on error.

Sources
  claude  ~/.claude/projects/*/*.jsonl      (Claude Code; frontier, D-FT-1 = yes)
  pi      ~/.pi/agent/sessions/*/*.jsonl    (pi-native)
  motoko  ~/dev/mk-*/.motoko/logfile/*.jsonl (eval runs; task = benchmark prompt)

Episode = from the active user task through the FIRST PASSING `ailang check|run|test`
that follows at least one .ail write/edit. Only passing episodes are kept.

Samples: mlx-lm --mask-prompt trains only the LAST message, so each episode yields
up to 4 samples (context + one target assistant turn): the first action, fix turns
(.ail write/edit after a failed verification), the last .ail write, and the final
verification call. Failed attempts appear only as context, never as targets
(data-spec rule 4). Thinking blocks are dropped (shorter outputs are a goal).

Contamination: episodes mentioning a held-out benchmark id, or writing code whose
normalised hash matches a held-out solution, are dropped.
"""
import argparse
import glob
import hashlib
import json
import os
import re
import shlex
import sys
from collections import Counter

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", ".."))
sys.path.insert(0, HERE)
from extract_lang import load_heldout, norm_hash, task_prompt  # noqa: E402

SYSTEM = ("You are an AILANG coding agent. Work with the tools: write the solution to the file, "
          "verify it with `ailang check` / `ailang run`, read the errors, and fix until it passes. "
          "Do not answer with code in chat.")
PI_TOOLS = [
    {"type": "function", "function": {"name": "bash", "description": "Run a shell command.",
     "parameters": {"type": "object", "properties": {"command": {"type": "string"}}, "required": ["command"]}}},
    {"type": "function", "function": {"name": "read", "description": "Read a file.",
     "parameters": {"type": "object", "properties": {"path": {"type": "string"}, "offset": {"type": "integer"},
                    "limit": {"type": "integer"}}, "required": ["path"]}}},
    {"type": "function", "function": {"name": "write", "description": "Write a whole file.",
     "parameters": {"type": "object", "properties": {"path": {"type": "string"}, "content": {"type": "string"}},
                    "required": ["path", "content"]}}},
    {"type": "function", "function": {"name": "edit", "description": "Replace exact text in a file.",
     "parameters": {"type": "object", "properties": {"path": {"type": "string"}, "edits": {"type": "array", "items": {
         "type": "object", "properties": {"oldText": {"type": "string"}, "newText": {"type": "string"}}}}},
         "required": ["path", "edits"]}}},
]
VERIFY = re.compile(r"(^|[;&|(\s])ailang\s+(check|run|test|ai-check)\b")
ERRTXT = re.compile(r"\bError\b|\berror:|PAR\d{3}|TC\d{3}|IMP\d{3}|LDR\d{3}")
MAX_TOOL_OUT = 2500
CTX_CHARS = 22000          # ~7k tokens of context + target
MAX_TARGETS = 4


def clip(s, n=MAX_TOOL_OUT):
    s = s or ""
    return s if len(s) <= n else s[: n * 6 // 10] + "\n…[truncated]…\n" + s[-n * 3 // 10:]


def strip_meta(text):
    text = re.sub(r"<system-reminder>.*?</system-reminder>", "", text, flags=re.S)
    text = re.sub(r"<(command|local-command)[^>]*>.*?</\1[^>]*>", "", text, flags=re.S)
    return text.strip()


def relpath(p, cwd):
    if not p:
        return p
    m = re.search(r"/(benchmark/[^\s\"']+)$", p)
    if m:
        return m.group(1)
    if cwd and p.startswith(cwd.rstrip("/") + "/"):
        return p[len(cwd.rstrip("/")) + 1:]
    return p


def is_ail(p):
    return isinstance(p, str) and p.endswith(".ail")


# ---------- source parsers -> list of steps ----------
# step: ("user", text) | ("assistant", text, [call]) | ("tool", call_id, output, is_error)
# call: {"id", "name", "args"} in pi schema, or None when unmappable

def map_claude(name, inp, cwd):
    if name == "Bash":
        return "bash", {"command": inp.get("command", "")}
    if name == "Read":
        a = {"path": relpath(inp.get("file_path"), cwd)}
        if inp.get("offset"):
            a["offset"] = inp["offset"]
        if inp.get("limit"):
            a["limit"] = inp["limit"]
        return "read", a
    if name == "Write":
        return "write", {"path": relpath(inp.get("file_path"), cwd), "content": inp.get("content", "")}
    if name == "Edit":
        return "edit", {"path": relpath(inp.get("file_path"), cwd),
                        "edits": [{"oldText": inp.get("old_string", ""), "newText": inp.get("new_string", "")}]}
    if name == "MultiEdit":
        return "edit", {"path": relpath(inp.get("file_path"), cwd),
                        "edits": [{"oldText": e.get("old_string", ""), "newText": e.get("new_string", "")}
                                  for e in inp.get("edits", [])]}
    if name == "Grep":
        return "bash", {"command": f"grep -rn {shlex.quote(inp.get('pattern', ''))} {shlex.quote(inp.get('path') or '.')}"}
    if name == "Glob":
        return "bash", {"command": f"find {shlex.quote(inp.get('path') or '.')} -path {shlex.quote('*' + inp.get('pattern', ''))}"}
    return None


def parse_claude(path):
    steps, cwd = [], None
    for line in open(path, errors="replace"):
        try:
            d = json.loads(line)
        except ValueError:
            continue
        if d.get("isSidechain") or d.get("type") not in ("user", "assistant"):
            continue
        cwd = d.get("cwd") or cwd
        m = d.get("message") or {}
        c = m.get("content")
        if d["type"] == "user":
            if isinstance(c, str):
                t = strip_meta(c)
                if t and not t.startswith("Caveat:"):
                    steps.append(("user", t))
                continue
            for b in c or []:
                if b.get("type") == "tool_result":
                    out = b.get("content")
                    if isinstance(out, list):
                        out = "\n".join(x.get("text", "") for x in out if isinstance(x, dict))
                    steps.append(("tool", b.get("tool_use_id"), str(out or ""), bool(b.get("is_error"))))
                elif b.get("type") == "text":
                    t = strip_meta(b.get("text", ""))
                    if t:
                        steps.append(("user", t))
        else:
            text, calls = [], []
            for b in c or []:
                if b.get("type") == "text":
                    text.append(b.get("text", ""))
                elif b.get("type") == "tool_use":
                    mp = map_claude(b.get("name"), b.get("input") or {}, cwd)
                    calls.append(None if mp is None else {"id": b.get("id"), "name": mp[0], "args": mp[1]})
            if text or calls:
                steps.append(("assistant", "\n".join(text).strip(), calls))
    return steps


PI_MAP = {"bash": "bash", "read": "read", "write": "write", "edit": "edit", "ailang_read": "read"}


def map_pi(name, a):
    a = a or {}
    if name in ("bash", "read", "write"):
        return name, a
    if name == "edit":
        return "edit", {"path": a.get("path"), "edits": a.get("edits") or []}
    if name in ("ailang_read", "world_ailang-read"):
        return "read", {"path": a.get("path")}
    if name in ("ailang_check", "world_ailang-check"):
        return "bash", {"command": f"ailang check {a.get('path', '')}"}
    if name in ("ailang_run", "world_ailang-run"):
        return "bash", {"command": f"ailang run {a.get('path', '')}"}
    if name == "world_ailang-edit":
        return "edit", {"path": a.get("path"), "edits": [{"oldText": a.get("old_text", ""), "newText": a.get("new_text", "")}]}
    return None


def parse_pi(path):
    steps = []
    for line in open(path, errors="replace"):
        try:
            d = json.loads(line)
        except ValueError:
            continue
        m = d.get("message") or {}
        role, c = m.get("role"), m.get("content")
        if role == "user":
            t = c if isinstance(c, str) else "\n".join(x.get("text", "") for x in c or [] if x.get("type") == "text")
            t = strip_meta(t)
            if t:
                steps.append(("user", t))
        elif role == "assistant":
            text, calls = [], []
            for b in c or []:
                if b.get("type") == "text":
                    text.append(b.get("text", ""))
                elif b.get("type") == "toolCall":
                    mp = map_pi(b.get("name"), b.get("arguments"))
                    calls.append(None if mp is None else {"id": b.get("id"), "name": mp[0], "args": mp[1]})
            if text or calls:
                steps.append(("assistant", "\n".join(text).strip(), calls))
        elif role == "toolResult":
            out = "\n".join(x.get("text", "") for x in c or [] if isinstance(x, dict))
            steps.append(("tool", m.get("toolCallId"), out, bool(m.get("isError"))))
    return steps


def map_motoko(name, a):
    a = a or {}
    if name in ("BashExec", "Bash", "Exec"):
        return "bash", {"command": a.get("cmd") or a.get("command", "")}
    if name in ("ReadFile", "Read"):
        return "read", {"path": relpath(a.get("path"), None)}
    if name in ("WriteFile", "Write", "CreateFile"):
        return "write", {"path": relpath(a.get("path"), None), "content": a.get("content", "")}
    if name in ("EditFile", "Edit", "StrReplace", "ReplaceInFile"):
        old = a.get("old") or a.get("old_string") or a.get("oldText") or a.get("search", "")
        new = a.get("new") or a.get("new_string") or a.get("newText") or a.get("replace", "")
        return "edit", {"path": relpath(a.get("path"), None), "edits": [{"oldText": old, "newText": new}]}
    if name in ("ListDir", "Ls"):
        return "bash", {"command": f"ls {relpath(a.get('path') or '.', None)}"}
    # motoko AILANG extension tools
    if name == "ailang_write":
        return "write", {"path": relpath(a.get("path"), None), "content": a.get("content", "")}
    if name == "ailang_edit":
        return "edit", {"path": relpath(a.get("path"), None),
                        "edits": [{"oldText": a.get("old_text", ""), "newText": a.get("new_text", "")}]}
    if name == "ailang_read":
        return "read", {"path": relpath(a.get("path"), None)}
    if name == "ailang_check":
        return "bash", {"command": f"ailang check {relpath(a.get('path', ''), None)}"}
    if name == "ailang_run":
        return "bash", {"command": f"ailang run {relpath(a.get('path', ''), None)}"}
    if name == "ailang_cli":
        return "bash", {"command": f"ailang {a.get('op', '')}"}
    return None


def parse_motoko(path, unknown):
    bid = re.sub(r"^session_|_[0-9a-f]{8}$", "", os.path.basename(path)[:-6])
    task = task_prompt(bid)
    if not task:
        return [], bid
    steps = [("user", task)]
    for line in open(path, errors="replace"):
        try:
            d = json.loads(line)
        except ValueError:
            continue
        if d.get("type") == "native_tool_calls":
            calls = []
            for tc in d.get("tool_calls") or []:
                mp = map_motoko(tc.get("tool"), tc.get("arguments"))
                if mp is None:
                    unknown[tc.get("tool")] += 1
                calls.append(None if mp is None else {"id": tc.get("id"), "name": mp[0], "args": mp[1]})
            steps.append(("assistant", "", calls))
        elif d.get("type") == "native_tool_results":
            for r in d.get("results") or []:
                pl = r.get("payload") or {}
                out = (r.get("stdout") or "") + (r.get("stderr") or "")
                if not out and isinstance(pl, dict):
                    out = pl.get("content") or pl.get("message") or ""
                steps.append(("tool", r.get("tool_call_id"), str(out), (r.get("exit_code") or 0) != 0))
    return steps, bid


OPENCODE_DB = os.path.expanduser("~/.local/share/opencode/opencode.db")


def map_opencode(name, a):
    a = a or {}
    p = relpath(a.get("filePath") or a.get("path"), None)
    if name == "bash":
        return "bash", {"command": a.get("command", "")}
    if name == "read":
        return "read", {"path": p}
    if name == "write":
        return "write", {"path": p, "content": a.get("content", "")}
    if name == "edit":
        return "edit", {"path": p, "edits": [{"oldText": a.get("oldString", ""), "newText": a.get("newString", "")}]}
    if name == "glob":
        return "bash", {"command": f"find . -path {shlex.quote('*' + a.get('pattern', ''))}"}
    return None


def iter_opencode():
    """Yield (session_id, benchmark_id, steps) for opencode eval sessions."""
    import sqlite3
    con = sqlite3.connect(f"file:{OPENCODE_DB}?mode=ro", uri=True)
    sessions = con.execute("select id, directory from session").fetchall()
    for sid, directory in sessions:
        m = re.match(r"(.+?)_opencode-", os.path.basename(os.path.dirname(directory or "")))
        if not m:
            continue
        bid = m.group(1)
        task = task_prompt(bid)
        if not task:
            yield sid, bid, []
            continue
        steps = [("user", task)]
        msgs = con.execute("select id, json_extract(data,'$.role') from message where session_id=? order by time_created, id", (sid,)).fetchall()
        for mid, role in msgs:
            if role != "assistant":
                continue
            parts = con.execute("select data from part where message_id=? order by time_created, id", (mid,)).fetchall()
            text, calls, results = [], [], []
            for (pd,) in parts:
                try:
                    p = json.loads(pd)
                except ValueError:
                    continue
                if p.get("type") == "text":
                    text.append(p.get("text", ""))
                elif p.get("type") == "tool":
                    st = p.get("state") or {}
                    mp = map_opencode(p.get("tool"), st.get("input"))
                    cid = p.get("callID")
                    calls.append(None if mp is None else {"id": cid, "name": mp[0], "args": mp[1]})
                    out = st.get("output") or st.get("error") or ""
                    results.append(("tool", cid, str(out), st.get("status") == "error"))
            if text or calls:
                steps.append(("assistant", "\n".join(text).strip(), calls))
                steps.extend(results)
        yield sid, bid, steps
    con.close()


# ---------- episodes -> samples ----------

def to_msgs(steps):
    """Convert steps to OpenAI-style messages; drop unmappable calls together with their results."""
    msgs, keep_ids = [], set()
    for s in steps:
        if s[0] == "user":
            msgs.append({"role": "user", "content": clip(s[1], 6000)})
        elif s[0] == "assistant":
            calls = [c for c in s[2] if c]
            for c in calls:
                keep_ids.add(c["id"])
            m = {"role": "assistant", "content": s[1][:2000]}
            if calls:
                m["tool_calls"] = [{"id": c["id"], "type": "function",
                                    "function": {"name": c["name"], "arguments": c["args"]}} for c in calls]
            if calls or s[1]:
                msgs.append(m)
        else:
            if s[1] in keep_ids:
                msgs.append({"role": "tool", "tool_call_id": s[1], "content": clip(s[2])})
    return msgs


def size(msgs):
    return sum(len(json.dumps(m)) for m in msgs)


def episodes(steps):
    """Yield (start, end_idx, fail_count) for passing AILANG episodes."""
    calls_by_id = {}
    task_idx, wrote, fails, start = None, 0, 0, None
    for i, s in enumerate(steps):
        if s[0] == "user":
            task_idx, wrote, fails, start = i, 0, 0, i
        elif s[0] == "assistant":
            for c in s[2]:
                if not c:
                    continue
                calls_by_id[c["id"]] = (i, c)
                if c["name"] in ("write", "edit") and is_ail(c["args"].get("path")):
                    wrote += 1
        elif s[0] == "tool" and s[1] in calls_by_id and start is not None:
            ai, c = calls_by_id[s[1]]
            if c["name"] == "bash" and VERIFY.search(c["args"].get("command", "")):
                passed = (not s[3]) and not ERRTXT.search(s[2] or "")
                if not passed:
                    fails += 1
                elif wrote:
                    yield start, ai, fails
                    start, wrote, fails = i + 1, 0, 0


def targets(steps, start, end):
    idx, after_fail, last_write, first = [], False, None, None
    calls = {}
    for i in range(start, end + 1):
        s = steps[i]
        if s[0] == "assistant" and any(s[2]):
            if first is None:
                first = i
            for c in s[2]:
                if c:
                    calls[c["id"]] = c
            if any(c and c["name"] in ("write", "edit") and is_ail(c["args"].get("path")) for c in s[2]):
                last_write = i
                if after_fail:
                    idx.append(("fix", i))
                    after_fail = False
        elif s[0] == "tool" and s[1] in calls and calls[s[1]]["name"] == "bash":
            # same failure test as episodes(): many harnesses report a failed
            # `ailang check` as a completed call with the error only in the text
            cmd = calls[s[1]]["args"].get("command", "")
            if s[3] or (VERIFY.search(cmd) and ERRTXT.search(s[2] or "")):
                after_fail = True
    out = [("first", first)] if first is not None else []
    out += idx[:2]
    if last_write is not None:
        out.append(("final_write", last_write))
    out.append(("verify", end))
    seen, res = set(), []
    for k, i in out:
        if i not in seen:
            seen.add(i)
            res.append((k, i))
    return res[:MAX_TARGETS]


def build_samples(steps, start, end, task_text):
    out = []
    for kind, k in targets(steps, start, end):
        ctx = [("user", task_text)] + [s for s in steps[start:k] if s[0] != "user"]
        tgt = to_msgs([steps[k]])
        if not tgt or tgt[-1]["role"] != "assistant":
            continue
        body = to_msgs(ctx)
        while size(body) + size(tgt) > CTX_CHARS and len(body) > 2:
            # drop the oldest assistant/tool turn after the task; never orphan a tool result
            del body[1]
            while len(body) > 1 and body[1]["role"] == "tool":
                del body[1]
        if size(body) + size(tgt) > CTX_CHARS:
            continue
        out.append((kind, [{"role": "system", "content": SYSTEM}] + body + tgt))
    return out


def written_code(steps, start, end):
    for s in steps[start:end + 1]:
        if s[0] == "assistant":
            for c in s[2]:
                if c and c["name"] == "write" and is_ail(c["args"].get("path")):
                    yield c["args"].get("content", "")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--sources", default="claude,pi,motoko,opencode")
    ap.add_argument("--name", default="traj")
    a = ap.parse_args()
    held = load_heldout()
    held_re = re.compile(r"\b(" + "|".join(map(re.escape, sorted(held, key=len, reverse=True))) + r")\b")
    held_hashes = set()
    for root, _, files in os.walk(os.path.join(REPO, "eval_results")):
        for fn in files:
            if fn.endswith(".json") and any(b in fn for b in held):
                try:
                    d = json.load(open(os.path.join(root, fn)))
                except (OSError, ValueError):
                    continue
                if isinstance(d, dict) and d.get("id") in held and d.get("code"):
                    held_hashes.add(norm_hash(d["code"]))

    want = set(a.sources.split(","))
    srcs = []
    if "claude" in want:
        srcs += [("claude", p) for p in glob.glob(os.path.expanduser("~/.claude/projects/*/*.jsonl"))]
    if "pi" in want:
        srcs += [("pi", p) for p in glob.glob(os.path.expanduser("~/.pi/agent/sessions/*/*.jsonl"))]
    if "motoko" in want:
        srcs += [("motoko", p) for p in glob.glob(os.path.expanduser("~/dev/mk-*/.motoko/logfile/*.jsonl"))]
    rep = {s: Counter() for s in ("claude", "pi", "motoko", "opencode") if s in want}
    unknown_motoko = Counter()
    seen_targets = set()
    os.makedirs(a.out, exist_ok=True)

    def emit(fo, src, name, steps):
        r = rep[src]
        got = False
        for start, end, fails in episodes(steps):
            task_text = next((s[1] for s in reversed(steps[:start + 1]) if s[0] == "user"), "")
            blob = task_text + json.dumps([s for s in steps[start:end + 1] if s[0] == "assistant"])[:200000]
            if held_re.search(blob) or any(norm_hash(c) in held_hashes for c in written_code(steps, start, end)):
                r["contaminated_episode"] += 1
                continue
            r["episodes"] += 1
            r["recovery_episodes"] += int(fails > 0)
            got = True
            for kind, msgs in build_samples(steps, start, end, task_text):
                key = hashlib.sha1(json.dumps(msgs[-1]).encode()).hexdigest()
                if key in seen_targets:
                    r["dup_target"] += 1
                    continue
                seen_targets.add(key)
                fo.write(json.dumps({"messages": msgs, "tools": PI_TOOLS,
                                     "meta": {"source": src, "session": name, "target": kind,
                                              "recovery": fails > 0}}) + "\n")
                r["samples"] += 1
                r["samples_" + kind] += 1
                r["chars"] += size(msgs)
        r["sessions_with_episodes"] += int(got)

    with open(os.path.join(a.out, a.name + ".jsonl"), "w") as fo:
        for src, p in srcs:
            r = rep[src]
            r["sessions"] += 1
            try:
                if src == "claude":
                    steps = parse_claude(p)
                elif src == "pi":
                    steps = parse_pi(p)
                else:
                    steps, bid = parse_motoko(p, unknown_motoko)
                    if bid in held:
                        r["heldout_session"] += 1
                        continue
            except Exception:  # malformed session: count, never crash the batch
                r["parse_error"] += 1
                continue
            emit(fo, src, os.path.basename(p), steps)
        if "opencode" in want:
            for sid, bid, steps in iter_opencode():
                r = rep["opencode"]
                r["sessions"] += 1
                if bid in held:
                    r["heldout_session"] += 1
                    continue
                if not steps:
                    r["no_task"] += 1
                    continue
                emit(fo, "opencode", sid, steps)
    report = {k: dict(v, est_tokens=int(v["chars"] / 3.2)) for k, v in rep.items()}
    report["motoko_unmapped_tools"] = unknown_motoko.most_common(10)
    json.dump(report, open(os.path.join(a.out, a.name + "_report.json"), "w"), indent=1)
    print(json.dumps(report, indent=1))


if __name__ == "__main__":
    sys.exit(main())
