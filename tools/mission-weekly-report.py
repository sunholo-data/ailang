#!/usr/bin/env python3
"""mission-weekly-report.py — a manager's report across every live mission loop.

WHY THIS EXISTS. Per-mission detail lives in three places that each answer a different
question, and none answers "what happened this week": charters carry direction, logs
carry per-iteration evidence (verbose by design — the Ruled-out ledger is the point),
and `ailang chains stats` carries cost but no outcomes. Reading all three weekly is the
tax this replaces. Target: one screen.

USAGE
  python3 tools/mission-weekly-report.py              # last 168h
  python3 tools/mission-weekly-report.py --hours 24
  python3 tools/mission-weekly-report.py --full       # untruncated headlines

HONESTY RULES (they are why some cells say n/a — do not "fix" them by guessing):
  * Metered $ is a LOWER BOUND: `chains post-iteration` is fail-soft, so an iteration
    that did not post leaves no row. Never present it as a total.
  * The binding resource is the QUOTA BUCKET (opus stages), not dollars.
  * Per-mission TOKENS are not exposed by the rollup (top_stages is top-N). Summing
    them would understate silently, so only the fleet total is printed.
  * v1 and motoko SHARE a repo, so commits/lines are attributed by subject and are
    approximate. Marked (~).
  * Lines-added is VOLUME, never efficiency — see the closing note.
"""
import argparse, json, os, re, subprocess, sys
from datetime import datetime, timedelta

MISSIONS = [
    ("v1",     "~/dev/sunholo-data/ailang",        "design_docs/v1-mission-log.md",     "design_docs/v1-mission.md",     "exclude"),
    ("world",  "~/dev/sunholo-data/ailang-world",  "design_docs/world-mission-log.md",  "design_docs/world-mission.md",  None),
    ("motoko", "~/dev/sunholo-data/ailang-motoko", "design_docs/motoko-mission-log.md", "design_docs/motoko-mission.md", "include"),
]
# Header shapes differ by historical accident: "## 7 — date — x" (v1, motoko) vs
# "## Iteration 7 — date — x" (world). Match both rather than rewriting the logs.
ITER_RE = re.compile(r'^##\s+(?:Iteration\s+)?(\d+)\s+—\s+(\d{4}-\d{2}-\d{2})\s+—\s*(.*)$')
# Two conventions in the wild, both counted: v1 writes "**Ruled out.** (a) … (b) …"
# (lettered, inline); motoko and world write "**Ruled out**:" followed by bullets.
# Matching only one scored v1 at 4 refutations across 39 iterations instead of ~38 —
# an instrument that reports a real signal as near-zero.
RULED = re.compile(r'\*\*Ruled out[.:]{0,2}\*\*:?(.*?)(?=\n\*\*(?:Retro|Next|Picked|Shipped)|\Z)', re.S)
ITEM  = re.compile(r'(?:^\s*[-*]\s)|(?:\([a-z]\)\s)', re.M)


def sh(cmd, cwd=None):
    try:
        return subprocess.run(cmd, shell=True, cwd=cwd, capture_output=True,
                              text=True, timeout=90).stdout.strip()
    except Exception:
        return ""


def clean(s, n):
    """Strip markdown emphasis/links so a headline fits on one line."""
    s = re.sub(r'\[([^\]]+)\]\([^)]*\)', r'\1', s)
    s = re.sub(r'[*`_]', '', s).strip().rstrip(':—- ')
    return s if len(s) <= n else s[:n - 1].rsplit(' ', 1)[0] + '…'


def parse_log(path, since):
    if not os.path.exists(path):
        return [], 0
    lines = open(path, errors="replace").read().split("\n")
    idx = [(i, m) for i, l in enumerate(lines) for m in [ITER_RE.match(l)] if m]
    out = []
    for n, (i, m) in enumerate(idx):
        if m.group(2) < since:
            continue
        end = idx[n + 1][0] if n + 1 < len(idx) else len(lines)
        body = "\n".join(lines[i:end])
        rm = RULED.search(body)
        out.append({"n": m.group(1), "date": m.group(2), "headline": m.group(3),
                    "refs": len(ITEM.findall(rm.group(1))) if rm else 0})
    return out, len(idx)


def open_decisions(charter, cap=4):
    """Rows that are waiting on a human RIGHT NOW.

    Heuristic, and deliberately conservative. The three charters retire rows by
    different conventions (`~~strike~~`, `Was:`, `RESOLVED`), and `needs-human-review`
    also appears in prose describing the escalation rule itself. Matching the marker
    alone re-raised every decision ever made — 9 rows for v1, none of them open. So:
    the line must look like a QUEUE ROW (list marker or number), must carry a live
    tag, and must not read as history. Under-reporting is the intended failure mode;
    the charter remains the source of truth and is linked in the output.
    """
    if not os.path.exists(charter):
        return []
    RETIRED = re.compile(r'~~|\bWas:|RESOLVED|LANDED|ANSWERED|DONE\b|Escalation', re.I)
    ROW = re.compile(r'^\s*(?:[-*]|\d+\.)\s')
    # PARKED-ON-A-HUMAN only. `[PARKED — Phase-0 gated]` waits on ARNI, and
    # `[PARKED until 2-5 land]` waits on other queue items — neither is an ask of Mark,
    # and listing them turns this section into the queue it is meant to summarise.
    LIVE = re.compile(r'needs-human-review|DECISION\s+D-?\d+|awaiting Mark|PARKED[^]]*\bhuman', re.I)
    GATED = re.compile(r'gated|until \d|blocked on #|green tree', re.I)
    out = []
    for l in open(charter, errors="replace").read().split("\n"):
        if ROW.match(l) and LIVE.search(l) and not RETIRED.search(l) and not GATED.search(l):
            out.append(clean(l, 130))
    return out[:cap]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--hours", type=int, default=168)
    ap.add_argument("--full", action="store_true")
    a = ap.parse_args()
    since = (datetime.now() - timedelta(hours=a.hours)).strftime("%Y-%m-%d")
    W = 200 if a.full else 96

    cost = {}
    try:
        for m in json.loads(sh(f"ailang chains stats --by-mission --hours {a.hours} --json")).get("missions", []):
            cost[m["mission"].replace("mission:", "")] = m
    except Exception:
        pass
    fleet = sh(f"ailang chains stats --hours {a.hours}")
    ftok = next((l.split(":", 1)[1].strip() for l in fleet.split("\n")
                 if l.strip().startswith("Tokens:")), "n/a")

    print("# Mission fleet — weekly report\n")
    print(f"**Window** last {a.hours}h (since {since}) · **generated** {datetime.now():%Y-%m-%d %H:%M}\n")

    rows, landed, decisions = [], {}, {}
    for name, repo, log, charter, share in MISSIONS:
        repo = os.path.expanduser(repo)
        iters, total = parse_log(os.path.join(repo, log), since)
        c = cost.get(name, {})
        opus = (c.get("quota_by_bucket") or {}).get("opus", 0)

        # v1 and motoko share sunholo-data/ailang — split by subject, and say so.
        filt = ""
        if share == "include":
            filt = "| grep -i motoko"
        elif share == "exclude":
            filt = "| grep -vi motoko"
        n_commits = sh(f"git log origin/dev --format='%s' --author='Voight-Kampff' "
                       f"--since='{since}' {filt} | wc -l", repo) or "0"
        # numstat interleaves with subjects, so attribution has to be per-commit inside awk —
        # filtering the commit list alone left v1 and motoko reporting the SAME line count.
        keep = {"include": 'tolower($0) ~ /motoko/',
                "exclude": 'tolower($0) !~ /motoko/'}.get(share, '1')
        churn = sh("git log origin/dev --numstat --format='@@%s' --author='Voight-Kampff' "
                   f"--since='{since}' | awk '/^@@/{{k=({keep}); next}} "
                   "k && NF==3 && $1 ~ /^[0-9]+$/ {a+=$1} END{print a+0}'", repo) or "0"
        rows.append({"n": name, "it": len(iters), "tot": total, "cost": c.get("reported_cost", 0.0),
                     "opus": opus, "c": n_commits.strip(), "churn": churn.strip(),
                     "refs": sum(i["refs"] for i in iters), "shared": share is not None})
        landed[name] = iters
        decisions[name] = open_decisions(os.path.join(repo, charter))

    print("| mission | iters (wk/all) | commits | lines+ | metered $ | opus | opus/it | refuted | ref/it |")
    print("|---|---:|---:|---:|---:|---:|---:|---:|---:|")
    for r in rows:
        tilde = "~" if r["shared"] else ""
        print("| **%s** | %d / %d | %s%s | %s%s | $%.2f | %d | %s | %d | %s |" % (
            r["n"], r["it"], r["tot"], tilde, r["c"], tilde, r["churn"], r["cost"], r["opus"],
            f"{r['opus']/r['it']:.2f}" if r["it"] else "—",
            r["refs"], f"{r['refs']/r['it']:.1f}" if r["it"] else "—"))
    print("| **fleet** | **%d** | | | **$%.2f** | **%d** | | **%d** | |" % (
        sum(r["it"] for r in rows), sum(r["cost"] for r in rows),
        sum(r["opus"] for r in rows), sum(r["refs"] for r in rows)))

    print(f"\n`$` is a **lower bound** (fail-soft posting) · `~` shared repo, subject-attributed · "
          f"fleet tokens **{ftok}** (all chains incl. evals; per-mission not exposed)\n")

    print("## Landed\n")
    for name, _, _, _, _ in MISSIONS:
        its = landed.get(name) or []
        if not its:
            print(f"**{name}** — no iterations in window\n"); continue
        print(f"**{name}** — {len(its)} iterations, latest first")
        for i in reversed(its[-5:]):
            print(f"- `#{i['n']}` {i['date']} · {clean(i['headline'], W)}")
        if len(its) > 5:
            print(f"- …and {len(its)-5} earlier")
        print()

    print("## Needs you\n")
    print("_Conservative — under-reports by design; the charter is the source of truth._\n")
    any_d = False
    for name, _, _, _, _ in MISSIONS:
        for d in decisions.get(name) or []:
            print(f"- **{name}** — {d}"); any_d = True
    if not any_d:
        print("Nothing parked on a human decision.\n")

    print("\n---\n*Lines-added is volume, not efficiency: the highest-value output of a loop is often a")
    print("refutation that deletes a wrong premise at near-zero line count. Rank cost on **opus/it**,")
    print("value on **ref/it**; read lines as context.*")


if __name__ == "__main__":
    sys.exit(main())
