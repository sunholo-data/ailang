import json, sys, os, glob
P = "workspace routing discovery"
root = os.path.expanduser("~/.codex/sessions/2026/10")
for f in sorted(glob.glob(root + "/*/*.jsonl")):
    with open(f, "rb") as fh:
        data = fh.read()
    if P.encode() not in data:
        continue
    cwd = "?"
    for ln in data.splitlines():
        try:
            r = json.loads(ln)
        except Exception:
            continue
        if r.get("type") == "session_meta" and cwd == "?":
            cwd = (r.get("payload") or {}).get("cwd", "?")
        s = ln.decode("utf-8", "replace")
        i = s.find(P)
        if i < 0:
            continue
        p = r.get("payload") or {}
        ctx = s[max(0, i - 110):i + 50].replace("\\n", " ")
        print(f"{os.path.basename(f)[9:28]}\t{os.path.basename(cwd)}\t{r.get('type')}/{p.get('type','')}\t{ctx}")
