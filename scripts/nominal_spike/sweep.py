#!/usr/bin/env python3
"""Prototype downstream-compat sweep: one arm (binary) over a set of corpora.

READ-ONLY on corpora: cwd is set inside the corpus but every cache/state/home
path points into the scratch dir. Writes JSONL rows to --out.
Usage: sweep.py --arm NAME --bin PATH --out FILE corpus=root [corpus=root ...]
"""
import argparse, json, os, re, subprocess, sys, time
from concurrent.futures import ThreadPoolExecutor

PRUNE = {".git", "node_modules", ".ailang", ".claude", "worktrees", "build", ".godot",
         "dist", "runtime", ".motoko", "eval_results", "wip"}
CODE_RE = re.compile(r"\b([A-Z]{2,6}_?[A-Z]*\d{3})\b")
MOD_RE = re.compile(r"^\s*module\s+([A-Za-z0-9_/]+)", re.M)


def discover(root):
    for dp, dns, fns in os.walk(root):
        dns[:] = [d for d in dns if d not in PRUNE and not d.startswith(".wt-")
                  and not (dp.endswith("/docs/ailang") and d == "docparse")]
        for f in fns:
            if f.endswith(".ail"):
                yield os.path.join(dp, f)


def entry(path, root):
    """Return (cwd, arg, rule) for one file: module-suffix root, else nearest ailang.toml."""
    try:
        src = open(path, encoding="utf-8", errors="replace").read(4096)
    except OSError:
        return os.path.dirname(path), os.path.basename(path), "unreadable"
    m = MOD_RE.search(src)
    noext = path[:-4]
    if m and noext.endswith("/" + m.group(1)):
        return noext[: -len(m.group(1)) - 1], m.group(1) + ".ail", "module-suffix"
    d = os.path.dirname(path)
    while d.startswith(root):
        if os.path.exists(os.path.join(d, "ailang.toml")):
            return d, os.path.relpath(path, d), "package-root"
        if d == root:
            break
        d = os.path.dirname(d)
    return os.path.dirname(path), os.path.basename(path), "dirname"


def classify(err):
    code, msg = err.get("code", ""), err.get("message", "")
    if code and code != "ERROR":
        return code
    m = CODE_RE.search(msg) or re.search(r"\b((?:PAR|TC|IMP|MOD|LDR|EFF|IFC)_[A-Z_]*[A-Z0-9])\b", msg)
    if m:
        return m.group(1)
    if "pure keyword contradicts" in msg:
        return "EFFECT/pure-with-row"
    if "information-flow violation" in msg:
        return "IFC/flow-violation"
    if "effect ceiling violation" in msg:
        return "EFFECT/ceiling"
    if "Missing effects" in msg or "uses effects not declared" in msg:
        return "EFFECT/undeclared"
    if msg.startswith("effect checking failed"):
        return "EFFECT/other"
    if "failed to resolve package import" in msg or "not found in ailang.lock" in msg:
        return "DEP/unresolved"
    if "package resolution setup failed" in msg:
        return "DEP/setup"
    if "type error" in msg[:40] and "information-flow" in msg:
        return "IFC/flow-violation"
    if msg.startswith("type error"):
        if "cannot unify" in msg or "unification failed" in msg:
            return "TYPE_ERROR/unify"
        if "undefined" in msg or "unbound" in msg:
            return "TYPE_ERROR/unbound"
        if "effect" in msg:
            return "TYPE_ERROR/effect"
        return "TYPE_ERROR/other"
    return "UNCODED:" + re.sub(r"[^a-z ]", "", msg.lower())[:40].strip()


def run_one(binp, armdir, corpus, root, path):
    cwd, arg, rule = entry(path, root)
    env = {k: v for k, v in os.environ.items() if not k.startswith("AILANG_")}
    env.update({
        "HOME": os.path.join(armdir, "home"),
        "AILANG_CACHE_DIR": os.path.join(armdir, "cache"),
        "AILANG_STATE_DIR": os.path.join(armdir, "state"),
        "AILANG_PACKAGE_ROOT": os.path.expanduser("~/.ailang/cache/registry"),
        "AILANG_TRACE": "off",
    })
    if os.environ.get("SPIKE_LOG"):
        env["AILANG_NOMINAL_SPIKE_LOG"] = os.environ["SPIKE_LOG"]
    t0 = time.time()
    cmd = ["nice", "-n", "15", "perl", "-e", "alarm 120; exec @ARGV", binp, "check", "--json", arg]
    try:
        p = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True)
        rc, out, errout = p.returncode, p.stdout, p.stderr
    except Exception as e:  # noqa: BLE001
        rc, out, errout = -1, "", str(e)
    dt = time.time() - t0
    codes, passed = [], None
    try:
        j = json.loads(out[out.index("{"):]) if "{" in out else None
    except ValueError:
        j = None
    if j is not None:
        passed = bool(j.get("passed"))
        codes = sorted({classify(e) for e in j.get("errors") or []})
    elif rc == 142 or rc == -14:
        codes = ["TIMEOUT"]
    else:
        codes = ["NO_JSON:rc=%d" % rc]
    if passed is None:
        passed = False
    return {"corpus": corpus, "file": os.path.relpath(path, root), "cwd": os.path.relpath(cwd, root),
            "rule": rule, "rc": rc, "passed": passed, "codes": codes, "secs": round(dt, 2),
            "first_msg": ((j or {}).get("errors") or [{}])[0].get("message", errout[:200])[:300]}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--arm", required=True)
    ap.add_argument("--bin", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--jobs", type=int, default=3)
    ap.add_argument("corpora", nargs="+")
    a = ap.parse_args()
    armdir = os.path.join(os.path.dirname(os.path.abspath(a.out)), "arm", a.arm)
    for sub in ("home", "cache", "state"):
        os.makedirs(os.path.join(armdir, sub), exist_ok=True)
    jobs = []
    for c in a.corpora:
        name, root = c.split("=", 1)
        root = os.path.abspath(os.path.expanduser(root))
        jobs += [(name, root, p) for p in sorted(discover(root))]
    t0 = time.time()
    with open(a.out, "w") as fh, ThreadPoolExecutor(a.jobs) as ex:
        for row in ex.map(lambda j: run_one(a.bin, armdir, *j), jobs):
            fh.write(json.dumps(row) + "\n")
    print("arm=%s files=%d wall=%.1fs" % (a.arm, len(jobs), time.time() - t0), file=sys.stderr)


if __name__ == "__main__":
    main()
