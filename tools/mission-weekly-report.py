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
    # name, repo, log, charter, shared-repo attribution, launchd job (v1's is legacy-named)
    ("v1",     "~/dev/sunholo-data/ailang",        "design_docs/v1-mission-log.md",     "design_docs/v1-mission.md",     "exclude", "dev.ailang.mission-control"),
    ("world",  "~/dev/sunholo-data/ailang-world",  "design_docs/world-mission-log.md",  "design_docs/world-mission.md",  None,      "dev.ailang.mission-world"),
    ("motoko", "~/dev/sunholo-data/ailang-motoko", "design_docs/motoko-mission-log.md", "design_docs/motoko-mission.md", "include", "dev.ailang.mission-motoko"),
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
    RETIRED = re.compile(r'~~|\bWas:|RESOLVED|LANDED|ANSWERED|DONE\b|Escalation|leave as-is', re.I)
    ROW = re.compile(r'^\s*(?:[-*]|\d+\.)\s')
    # PARKED-ON-A-HUMAN only. `[PARKED — Phase-0 gated]` waits on ARNI, and
    # `[PARKED until 2-5 land]` waits on other queue items — neither is an ask of Mark,
    # and listing them turns this section into the queue it is meant to summarise.
    LIVE = re.compile(r'needs-human-review|DECISION\s+D-?\d+|awaiting Mark|PARKED[^]]*\bhuman', re.I)
    GATED = re.compile(r'gated|until \d|blocked on #|green tree', re.I)
    out = []
    lines = open(charter, errors="replace").read().split("\n")
    for i, l in enumerate(lines):
        if ROW.match(l) and LIVE.search(l) and not RETIRED.search(l) and not GATED.search(l):
            # Render as a sentence, not a raw queue row (Mark 2026-08-31: the one section
            # built for decisions was the least decidable part of the report). Charter
            # bullets wrap, so pull up to 2 indented continuation lines before cleaning —
            # a single-line read cut every wrapped ask mid-question. Emphasis strips
            # BEFORE the bracket-tag strip: the rows are written `**[PARKED …]**`, and
            # a `[` behind `**` defeats a start-anchored pattern.
            row = l
            for j in range(i + 1, min(i + 3, len(lines))):
                if not lines[j].strip() or ROW.match(lines[j]) or lines[j].startswith("#"):
                    break
                row += " " + lines[j].strip()
            row = re.sub(r'^\s*(?:[-*]|\d+\.)\s*', '', row).strip()
            row = re.sub(r'[*`_]', '', row)
            row = re.sub(r'^\[([^\]]*)\]\s*', r'\1 — ', row)
            did = re.search(r'\bD-[A-Z]*-?\d+\b', row)
            row = clean(row, 250)
            out.append(f"**{did.group(0)}** — {row}" if did else row)
    return out[:cap]


NEXT_RE = re.compile(r'^\*\*Next\*\*\s*[:—-]?\s*(.+)$')
PROG_RE = re.compile(r'^\s*-?\s*\*\*Progress\*\*\s*[:—-]?\s*(.+)$')
CLASS_RE = re.compile(r'\[(PRODUCT|HARNESS|ADMIN|REFUTATION)\]')


def goal_distance(log):
    """The mission's own last `**Progress**:` claim — goal distance in the charter's unit.

    Same philosophy as roadmap(): one authoritative line the mission wrote about itself
    beats a heuristic over the charter. Mandated in the Gate-5 digest and mirrored into
    the Gate-4 log entry since 2026-08-31; entries predating that have no such line, so
    None here means "no iteration has reported since the mandate", not "no progress".
    """
    if not os.path.exists(log):
        return None
    for l in reversed(open(log, errors="replace").read().split("\n")):
        m = PROG_RE.match(l)
        if m and len(m.group(1).strip()) > 8:
            return clean(m.group(1), 200)
    return None


def class_mix(iters):
    """Work-class mix from Gate-4 headline tags ([PRODUCT] etc., mandated 2026-08-31).

    This is the instrument for "is the loop stuck in admin?" — untagged counts stay
    visible rather than being folded away, so a mission that never tags reads as
    unmeasured, not as healthy.
    """
    mix = {}
    for i in iters:
        m = CLASS_RE.search(i["headline"])
        k = m.group(1).lower() if m else "untagged"
        mix[k] = mix.get(k, 0) + 1
    order = ["product", "harness", "admin", "refutation", "untagged"]
    return " · ".join(f"{k} {mix[k]}" for k in order if mix.get(k))


def harness_drift():
    """Installed vs registry-latest for the agent harness CLIs (Mark 2026-08-31:
    "careful on upgrades but keep on top of them" — weekly SURFACING here; APPLYING
    an upgrade stays attended-only, per docs/internal/harness-upgrade-runbook.md).
    Fail-soft throughout: any probe error renders n/a rather than wedging rotation.
    """
    rows = []
    for name, cmd, pkg in [
        ("pi", "pi --version", "@earendil-works/pi-coding-agent"),
        ("codex", "codex --version", "@openai/codex"),
        ("opencode", "opencode --version", "opencode-ai"),
    ]:
        out = sh(cmd)
        cur = out.split()[-1] if out else "n/a"
        latest = sh(f"npm view {pkg} version") or "n/a"
        flag = "" if "n/a" in (cur, latest) or cur == latest else " — **behind**"
        rows.append(f"- **{name}** {cur} (npm latest {latest}){flag}")
    cl = sh("claude --version")
    rows.append(f"- **claude** {cl.split()[0] if cl else 'n/a'} (self-updating channel)")
    ol = sh("ollama --version")
    rows.append(f"- **ollama** {ol.split()[-1] if ol else 'n/a'} — upgrades are BANKED eval "
                "boundaries, never mid-rotation")
    nb = sh("brew outdated --quiet | wc -l").strip()
    rows.append(f"- **brew outdated**: {nb or 'n/a'} formulae")
    return rows


def roadmap(charter, log, plist_name):
    """What each mission says it will do next — its OWN declaration, not my inference.

    Source is the LAST `**Next**:` line in the mission log, not the charter queue.
    That was a deliberate switch: the queues are 700-1200 lines of human prose with
    three different retirement conventions, and parsing them produced a confident
    wrong answer (v1's "next pick" came back as a DO-NOT-MERGE draft row, world's as
    a status line). The `**Next**` field is one line, written by the mission at Gate 4
    about the iteration it just finished, and is present in the latest entry of all
    three logs. One authoritative sentence beats a heuristic over a thousand.

    Blocked count still comes from the queue — a count survives format drift where an
    extract does not.

    Returns (next_declaration, blocked_count, nominal_fires_per_week).
    """
    nominal = None
    pl = os.path.expanduser(f"~/Library/LaunchAgents/{plist_name}.plist")
    if os.path.exists(pl):
        iv = sh(f"plutil -extract StartInterval raw '{pl}'")
        if iv.isdigit() and int(iv) > 0:
            nominal = round(168 * 3600 / int(iv))

    nxt = None
    if os.path.exists(log):
        for l in reversed(open(log, errors="replace").read().split("\n")):
            m = NEXT_RE.match(l)
            if m and len(m.group(1).strip()) > 12:   # skip the template placeholder
                nxt = clean(m.group(1), 150)
                break

    blocked = 0
    if os.path.exists(charter):
        text = open(charter, errors="replace").read().split("\n")
        st = next((i for i, l in enumerate(text) if re.match(r'^##\s+Queue\b', l)), None)
        if st is not None:
            en = next((i for i in range(st + 1, len(text)) if text[i].startswith("## ")), len(text))
            for l in text[st:en]:
                if re.match(r'^\s*(?:\d+\.|[-*])\s', l) and re.search(r'\[PARKED', l, re.I) \
                   and '~~' not in l:
                    blocked += 1
    return nxt, blocked, nominal


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--hours", type=int, default=168)
    ap.add_argument("--full", action="store_true")
    ap.add_argument("--mission", help="scope to ONE mission (v1|world|motoko). Each mission rotates "
                                      "its bookkeeping thread independently, so a fleet-wide report "
                                      "posted at rotation would land 3x, two thirds of it off-topic "
                                      "for the thread it is in. This emits that mission's rows plus a "
                                      "one-line fleet comparison for context.")
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

    sel = a.mission
    if sel and sel not in [m[0] for m in MISSIONS]:
        sys.exit(f"unknown mission {sel!r}; expected one of {[m[0] for m in MISSIONS]}")
    title = f"Mission `{sel}` — weekly report" if sel else "Mission fleet — weekly report"
    print(f"# {title}\n")
    print(f"**Window** last {a.hours}h (since {since}) · **generated** {datetime.now():%Y-%m-%d %H:%M}\n")

    rows, landed, decisions, plan, dist = [], {}, {}, {}, {}
    for name, repo, log, charter, share, job in MISSIONS:
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
        plan[name] = roadmap(os.path.join(repo, charter), os.path.join(repo, log), job)
        dist[name] = goal_distance(os.path.join(repo, log))

    print("| mission | iters (wk/all) | commits | lines+ | metered $ | opus | opus/it | refuted | ref/it |")
    print("|---|---:|---:|---:|---:|---:|---:|---:|---:|")
    for r in ([x for x in rows if x["n"] == sel] if sel else rows):
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

    print("## Goal distance — each mission's own claim\n")
    print("_Source: the last `**Progress**` line in the mission log (self-reported at Gate 4,")
    print("mandated 2026-08-31). \"not yet reported\" = no iteration has run since the mandate._\n")
    for name, _, _, _, _, _ in [m for m in MISSIONS if not sel or m[0] == sel]:
        print(f"- **{name}** — {dist.get(name) or 'not yet reported'}")
    print()

    print("## Landed\n")
    for name, _, _, _, _, _ in [m for m in MISSIONS if not sel or m[0] == sel]:
        its = landed.get(name) or []
        if not its:
            print(f"**{name}** — no iterations in window\n"); continue
        print(f"**{name}** — {len(its)} iterations ({class_mix(its)}), latest first")
        for i in reversed(its[-5:]):
            print(f"- `#{i['n']}` {i['date']} · {clean(i['headline'], W)}")
        if len(its) > 5:
            print(f"- …and {len(its)-5} earlier")
        print()

    print("## Next week — each mission's own declaration\n")
    print("`next` is the last iteration's `**Next**` field — what the mission itself said it would")
    print("take. `expected` is last week's OBSERVED rate (nominal capacity in brackets); observed is")
    print("the better predictor because iterations overrun the interval and the overlap guard yields.\n")
    print("| mission | expected iters | blocked | next |")
    print("|---|---:|---:|---|")
    for nm, _, _, _, _, _ in [m for m in MISSIONS if not sel or m[0] == sel]:
        nxt, blk, nom = plan.get(nm, (None, 0, None))
        obs = next((r["it"] for r in rows if r["n"] == nm), 0)
        print(f"| **{nm}** | {obs}{f' ({nom})' if nom else ''} | {blk} | {nxt or '—'} |")
    print()

    print("## Harness drift\n")
    print("_Surfaced weekly; upgrading is attended-only — see"
          " docs/internal/harness-upgrade-runbook.md._\n")
    for r in harness_drift():
        print(r)
    print()

    print("## Needs you\n")
    print("_Conservative — under-reports by design; the charter is the source of truth._\n")
    any_d = False
    for name, _, _, _, _, _ in [m for m in MISSIONS if not sel or m[0] == sel]:
        for d in decisions.get(name) or []:
            print(f"- **{name}** — {d}"); any_d = True
    if any_d:
        print("\n_Answer by replying on that mission's bookkeeping thread with the D-id —"
              " one word usually suffices._")
    else:
        print("Nothing parked on a human decision.\n")

    print("\n---\n*Lines-added is volume, not efficiency: the highest-value output of a loop is often a")
    print("refutation that deletes a wrong premise at near-zero line count. Rank cost on **opus/it**,")
    print("value on **ref/it**; read lines as context.*")


if __name__ == "__main__":
    sys.exit(main())
