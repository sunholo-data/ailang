#!/usr/bin/env bash
# rig-value-report.sh — what did the local-rig rotation buy this week?
#
# The rotation is free in dollars and expensive in GPU time. Between 10-20 and
# 11-03 it banked 894 runs whose pass rates did not move across six versions,
# while two benchmarks that every harness fails took most of the signal-first
# hours, and for five of those days it measured a stale binary nobody noticed.
# This report exists so those three things are visible every week:
#
#   1. Is the rig measuring the release we think it is? (binary vs std/VERSION
#      vs the version the rows banked under)
#   2. Where did the time go? (runs, banked wall-hours, timeouts banked as 0 ms,
#      which are therefore NOT in the hours figure)
#   3. What did it learn? (harness gaps, regressions vs the previous version,
#      saturated and parked benchmarks)
#
# Usage: tools/rig-value-report.sh [--days N] [--send]
#   --send  posts the report to the prod `user` inbox (reaches Discord). Without
#           it the markdown goes to stdout only.
#
# The filler calls this with --send at most once per 7 days (state stamp
# ~/.ailang/state/rig-value-report.last).
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1

DAYS=7; SEND=0
while [ $# -gt 0 ]; do
  case "$1" in
    --days) DAYS="$2"; shift 2;;
    --send) SEND=1; shift;;
    *) echo "usage: $0 [--days N] [--send]" >&2; exit 2;;
  esac
done

ROLL="eval_results/rotation/os-rolling"
MODELS="${OS_FILLER_MODELS:-opencode-qwen3-8-27b,pi-qwen3-8-27b,motoko-local-qwen3-8-27b-microrag}"
STD_VERSION="$(tr -d '[:space:]' < std/VERSION 2>/dev/null)"
BIN_VERSION="$(ailang --version 2>/dev/null | head -1 | sed -E 's/^AILANG[[:space:]]+//')"
PARKED="$(bash tools/eval-park-set.sh --roll "$ROLL" --models "$MODELS" 2>/dev/null | tr '\n' ' ')"
# Dark models on the current release's bank (zero valid rows while peers bank).
# rc 1/2 = the check itself broke: say so in the report rather than print "none".
DARK="$(bash tools/rig-model-health.sh --vdir "$ROLL/$STD_VERSION/agent" --models "$MODELS" 2>&1)"
case $? in 0) DARK="";; 3) DARK="$(printf '%s\n' "$DARK" | sed -n 's/^DARK //p' | tr '\n' ' ')";; *) DARK="CHECK-FAILED";; esac

REPORT="$(python3 - "$ROLL" "$MODELS" "$DAYS" "$STD_VERSION" "$BIN_VERSION" "$PARKED" "$DARK" "$MODELS" <<'PYEOF'
import collections, datetime, glob, json, os, re, sys
roll, models, days, std_v, bin_v, parked = sys.argv[1], set(sys.argv[2].split(",")), int(sys.argv[3]), sys.argv[4], sys.argv[5], sys.argv[6].split()
dark, active = sys.argv[7].split(), [m for m in sys.argv[8].split(",") if m]
now = datetime.datetime.now(datetime.timezone.utc)
since = now - datetime.timedelta(days=days)
def vkey(v): return tuple(int(x) for x in re.findall(r"\d+", v))

rows, byver = [], collections.defaultdict(list)
for path in glob.glob(os.path.join(roll, "v[0-9]*", "agent", "*_ailang_*.json")):
    try: d = json.load(open(path))
    except Exception: continue
    if d.get("model") not in models: continue
    ver = path.split(os.sep)[-3]
    d["_ver"] = ver
    byver[ver].append(d)
    try: ts = datetime.datetime.fromisoformat(d.get("timestamp", ""))
    except ValueError: continue
    if ts >= since: rows.append(d)

out = []
p = out.append
p("## Rig value report — last %d days (%s)" % (days, now.strftime("%Y-%m-%d")))
p("")
banked = sorted({r["_ver"] for r in rows}, key=vkey)
bin_base = re.sub(r"-dirty$", "", re.sub(r"-\d+-g[0-9a-f]+", "", bin_v.strip()))  # == eval_harness.ReleaseTag
p("**Measuring:** std/VERSION `%s` · binary `%s` · rows banked under %s"
  % (std_v, bin_v or "<none>", ", ".join("`%s`" % v for v in banked) or "nothing"))
if bin_base != std_v or (banked and banked[-1] != std_v):
    p("")
    p("⚠ **MISMATCH** — the rig is not measuring the current release. Check `ailang --version` vs `std/VERSION` and the filler log.")
if dark == ["CHECK-FAILED"]:
    p("")
    p("⚠ **Model-health check FAILED** — could not tell whether every model is banking. Run `tools/rig-model-health.sh` by hand.")
elif dark:
    p("")
    p("⚠ **DARK MODEL(S)** — %s banked ZERO valid rows on `%s` while the others banked. The rotation is running without them and coverage cannot complete. Check the canary lines in `/tmp/ailang-os-filler.log`." % (", ".join("`%s`" % m for m in dark), std_v))
p("")

if not rows:
    p("No runs banked in the window. The rig was idle or every cycle yielded — check `/tmp/ailang-os-filler.log`.")
    print("\n".join(out)); sys.exit(0)

hours = sum(r.get("duration_ms", 0) for r in rows) / 3.6e6
zero_to = sum(1 for r in rows if r.get("error_category") == "timeout" and not r.get("duration_ms"))
p("**Time:** %d runs, %.1f banked wall-hours, $0. %d timeout row(s) banked `duration_ms=0` — their hours are NOT in that figure."
  % (len(rows), hours, zero_to))
p("")
p("| model | runs | pass | failures |")
p("|---|---|---|---|")
bym = collections.defaultdict(list)
for r in rows: bym[r["model"]].append(r)
for m in sorted(set(bym) | set(active)):
    rs = bym.get(m, [])
    if not rs:
        p("| %s | 0 | — | **no rows this window** |" % m); continue
    ok = sum(bool(r.get("stdout_ok")) for r in rs)
    fails = collections.Counter(r.get("error_category") for r in rs if not r.get("stdout_ok"))
    p("| %s | %d | %d%% | %s |" % (m, len(rs), round(100 * ok / len(rs)), ", ".join("%s %d" % kv for kv in fails.most_common()) or "—"))
p("")

# What it learned.
byb = collections.defaultdict(lambda: collections.defaultdict(lambda: [0, 0]))
for r in rows:
    c = byb[r["id"]][r["model"]]; c[1] += 1; c[0] += bool(r.get("stdout_ok"))
gaps, saturated = [], 0
for b, per in sorted(byb.items()):
    tot = sum(n for _, n in per.values()); ok = sum(a for a, _ in per.values())
    if ok == tot: saturated += 1
    rated = {m: a / n for m, (a, n) in per.items() if n >= 3}
    if len(rated) >= 2 and max(rated.values()) >= 0.8 and min(rated.values()) <= 0.3:
        gaps.append("`%s` — %s" % (b, ", ".join("%s %d/%d" % (m.split("-")[0], a, n) for m, (a, n) in sorted(per.items()))))
p("**Learned:**")
p("- Harness gaps (one harness ≥80%%, another ≤30%%, ≥3 rows each): %s" % (len(gaps) or "none"))
for g in gaps: p("  - " + g)

vers = sorted(byver, key=vkey)
cur = banked[-1] if banked else None
prev = next((v for v in reversed(vers) if cur and vkey(v) < vkey(cur)), None)
regress = []
if cur and prev:
    def rate(v):
        acc = collections.defaultdict(lambda: [0, 0])
        for r in byver[v]: a = acc[r["id"]]; a[1] += 1; a[0] += bool(r.get("stdout_ok"))
        return {b: (a / n, n) for b, (a, n) in acc.items()}
    rc, rp = rate(cur), rate(prev)
    for b in sorted(set(rc) & set(rp)):
        (c, cn), (q, qn) = rc[b], rp[b]
        if cn >= 3 and qn >= 3 and q - c >= 0.5:
            regress.append("`%s` %d%% → %d%%" % (b, round(100 * q), round(100 * c)))
p("- Regressions `%s` → `%s` (pass rate down ≥50pp, ≥3 rows each): %s" % (prev, cur, ", ".join(regress) or "none"))
p("- Saturated (every run passed this window): %d of %d benchmarks run" % (saturated, len(byb)))
p("- Parked (no harness passes; 1 probe trial per release): %s" % (", ".join("`%s`" % x for x in parked) or "none"))
p("")
if dark:
    p("**Verdict:** fix the dark model(s) first — every other number above is missing a harness.")
elif not gaps and not regress:
    p("**Verdict:** no new information this week. If this repeats, the rotation is spending GPU on confirmation only — shrink it or point it at a question.")
else:
    p("**Verdict:** %d harness gap(s), %d regression(s) worth a `chains chat` look." % (len(gaps), len(regress)))
print("\n".join(out))
PYEOF
)" || { echo "rig-value-report: report generation failed" >&2; exit 1; }

printf '%s\n' "$REPORT"
if [ "$SEND" = "1" ]; then
  # Prod message plane, set on THIS command only (CLAUDE.md: never export it globally).
  AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
    ailang messages send user "$REPORT" --title "Rig value report $(date +%F)" --from rig-value-report --force
fi
