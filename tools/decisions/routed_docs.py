#!/usr/bin/env python3
# Helper for routed_docs.sh: reads "path<TAB>lane<TAB>evidence" rows on stdin, emits JSONL.

import sys, json, re
def section(text, names, limit):
    for n in names:
        m = re.search(r"^##+\s*" + n + r"[^\n]*\n(.*?)(?=^##\s|\Z)", text, re.S | re.M | re.I)
        if m and m.group(1).strip():
            return m.group(1).strip()[:limit]
    return ""
for line in sys.stdin:
    line = line.rstrip("\n")
    if not line: continue
    path, lane, evidence = line.split("\t")
    text = open(path, encoding="utf-8", errors="replace").read()
    title = text.splitlines()[0].lstrip("# ").strip() if text else path
    # Label leakage guard, applied BEFORE extraction: drop any routing section
    # and the exact evidence fragment, so the state never contains the label.
    clean = re.sub(r"(?ims)^##+\s*[^\n]*routing[^\n]*\n.*?(?=^##\s|\Z)", "", text)
    clean = clean.replace(evidence, "")
    clean = re.sub(r"(?im)^.*\*\*lane\b.*$", "", clean)
    problem = section(clean, ["Problem Statement", "Problem", "Motivation", "Summary"], 3000)
    goals = section(clean, ["Goals?", "Solution Design", "Proposed Solution", "Overview"], 1500)
    if not problem and not goals:
        # Auto-generated docs leave "## Problem Statement" empty; fall back to
        # the opening body (after the H1 + header block).
        body = re.sub(r"(?s)\A#[^\n]*\n(?:\*\*[^\n]*\n|\s*\n)*", "", clean)
        problem = body.strip()[:3500]
    state = {
        "title": title,
        "problem_statement": problem,
        "goals": goals,
        "files": section(clean, ["Files to Modify/Create", "Files to Modify", "Files"], 1200),
    }
    print(json.dumps({"path": path, "lane": lane, "evidence": evidence, "state": state}, ensure_ascii=False))

