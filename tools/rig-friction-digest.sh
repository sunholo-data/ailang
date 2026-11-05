#!/usr/bin/env bash
# rig-friction-digest.sh — turn the local rig's failures into routed work.
#
# The rig runs signal benchmarks only (os-rotation-filler OS_FILLER_SIGNAL_ONLY) so
# that its failures can feed PROGRAM.md §4's lanes: the AILANG roadmap and the
# motoko harness. Two views, over each benchmark's own last N version banks (the
# eval-park-set.sh window), rotation tiers only, and only benchmarks whose newest
# evidence is among the last 6 banks (older evidence is not current):
#
#   ailang       compile/runtime errors GROUPED BY ERROR KIND across benchmarks
#                (paths, positions, numbers stripped) -> AILANG roadmap: error
#                message, missing builtin, dialect trap, teaching prompt. A kind that
#                recurs across benchmarks or harnesses is the signal.
#   motoko-gap   per benchmark: motoko <= 1/3 while pi or opencode >= 2/3 (>= 2 rows
#                each) -> motoko harness: what does the winning harness do differently?
#   harness-gap  the same split between other harnesses (e.g. opencode 1/8, pi 7/7)
#                -> harness evidence for motoko: which harness behaviour wins
#   budget       every harness fails, mostly timeouts -> wall clock / model, not the
#                language (the parked probe tests it)
#   sporadic     anything else (single misses): listed, not routed
# plus a harness-infrastructure line (non_agentic rows: the harness never used tools).
#
# A lane is a starting point for triage, not a verdict: whoever reads the
# transcripts (ailang-core-triage, the motoko mission) decides.
#
# Usage: tools/rig-friction-digest.sh [--roll DIR] [--versions N] [--json] [--route]
#   --route  sends NEW items (signature = item:lane, remembered in
#            ~/.ailang/state/rig-friction-routed) on the prod message plane:
#            ailang kinds seen >= 2 times -> ailang-core (dispatches
#            ailang-core-triage; a one-off slip is not routed); motoko-gap and
#            harness-gap -> the user inbox while the motoko mission is paused
#            (~/.ailang/state/mission-motoko.disabled), else mission-motoko. One
#            message per destination per call; a known signature is never re-sent;
#            a failed send is not remembered (it retries next time) and exits 1.
#            The filler runs this weekly, next to the value report.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1

NVER=3; RECENT=6; JSON=0; ROUTE=0; ROLL_ARG=""
while [ $# -gt 0 ]; do
  case "$1" in
    --versions) NVER="$2"; shift 2;;
    --roll) ROLL_ARG="$2"; shift 2;;
    --json) JSON=1; shift;;
    --route) ROUTE=1; shift;;
    *) echo "usage: $0 [--roll DIR] [--versions N] [--json] [--route]" >&2; exit 2;;
  esac
done
ROLL="${ROLL_ARG:-eval_results/rotation/os-rolling}"
MODELS="${OS_FILLER_MODELS:-opencode-qwen3-8-27b,pi-qwen3-8-27b,motoko-local-qwen3-8-27b-microrag}"
STATE="$HOME/.ailang/state/rig-friction-routed"

DIGEST_JSON="$(python3 - "$ROLL" "$MODELS" "$NVER" "$RECENT" <<'PYEOF'
import collections, glob, json, os, re, sys
roll, models, nver, recent = sys.argv[1], [m for m in sys.argv[2].split(",") if m], int(sys.argv[3]), int(sys.argv[4])
def vkey(p): return tuple(int(x) for x in re.findall(r"\d+", os.path.basename(p)))
# Only benchmarks the rotation actually runs (core/stretch/frontier tiers).
rot = set()
for f in glob.glob("benchmarks/*.yml"):
    t = open(f).read()
    m = re.search(r"^tier:\s*\"?(\w+)", t, re.M)
    if m and m.group(1) in ("core", "stretch", "frontier"):
        rot.add(os.path.basename(f)[:-4])
banks = sorted(glob.glob(os.path.join(roll, "v[0-9]*")), key=vkey)
live = {os.path.basename(v) for v in banks[-recent:]}          # "recent evidence" horizon
by = collections.defaultdict(list)                               # id -> [(ver, [rows])]
for vdir in banks:
    per = collections.defaultdict(list)
    for path in glob.glob(os.path.join(vdir, "agent", "*_ailang_*.json")):
        try: d = json.load(open(path))
        except Exception: continue
        if d.get("model") not in models or d.get("lang") not in (None, "ailang"): continue
        v = d.get("validity")
        if v is not None and not v.get("valid"): continue
        d["_path"] = path
        per[d.get("id")].append(d)
    for bid, rows in per.items():
        by[bid].append((os.path.basename(vdir), rows))
def short(m): return "motoko" if m.startswith("motoko") else m.split("-")[0]
def err_line(d):
    for ln in (d.get("stderr") or "").splitlines():
        ln = ln.strip()
        if re.match(r"(API )?Error:|error:|panic:|Missing effects|execution timed out", ln):
            return re.sub(r"\s+", " ", ln)[:220]
    cat = d.get("error_category") or "unknown"
    return {"logic_error": "wrong output (stdout != expected)"}.get(cat, "(%s, no error line)" % cat)
def signature(line):  # stable across benchmarks: drop paths, positions, decl numbers, quoted names
    s = re.sub(r"\S*\.ail(:\d+)*", "<file>", line)
    s = re.sub(r"\(decl \d+\)", "", s)
    s = re.sub(r"\[[^\]]*\]", "[...]", s)
    s = re.sub(r"\d+", "N", s)
    return re.sub(r"\s+", " ", s).strip()[:120]
COMPILE = {"compile_error", "runtime_error"}
items, ailang = [], collections.defaultdict(lambda: {"count": 0, "harnesses": set(), "benchmarks": set(), "example": "", "rows": []})
infra = collections.Counter()
for bid in sorted(by):
    if bid not in rot: continue
    win = by[bid][-nver:]
    if win[-1][0] not in live: continue                          # stale evidence: not current
    rows = [d for _, rs in win for d in rs]
    fails = [d for d in rows if not d.get("stdout_ok")]
    for d in fails:
        cat = d.get("error_category")
        if cat == "non_agentic": infra["%s non_agentic" % short(d["model"])] += 1
        if cat in COMPILE and not (d.get("stderr") or "").startswith("API Error"):
            ln = err_line(d); sig = signature(ln); g = ailang[sig]
            g["count"] += 1; g["harnesses"].add(short(d["model"])); g["benchmarks"].add(bid)
            g["example"] = g["example"] or ln; g["rows"].append(d["_path"])
    if not fails: continue
    score = {}
    for m in models:
        rs = [d for d in rows if d["model"] == m]
        score[short(m)] = [sum(bool(d.get("stdout_ok")) for d in rs), len(rs)]
    ok2 = {k for k, (a, n) in score.items() if n >= 2 and a / n >= 2 / 3}
    bad2 = {k for k, (a, n) in score.items() if n >= 2 and a / n <= 1 / 3}
    cats = collections.Counter(d.get("error_category") or "unknown" for d in fails)
    if "motoko" in bad2 and ok2 - {"motoko"}: lane = "motoko-gap"
    elif bad2 and ok2: lane = "harness-gap"          # pi/opencode lose to another harness
    elif sum(a for a, _ in score.values()) == 0 and cats.get("timeout", 0) >= len(fails) / 2: lane = "budget"
    else: lane = "sporadic"
    errs = []
    for d in fails:
        ln = err_line(d)
        if ln not in [e[1] for e in errs]: errs.append((short(d["model"]), ln))
    items.append({"id": bid, "lane": lane, "window": [v for v, _ in win], "score": score,
                  "categories": dict(cats), "errors": errs[:4],
                  "rows": sorted(d["_path"] for d in fails)[:4]})
for sig, g in ailang.items():
    items.append({"id": "err:" + sig, "lane": "ailang", "signature": sig, "count": g["count"],
                  "harnesses": sorted(g["harnesses"]), "benchmarks": sorted(g["benchmarks"]),
                  "example": g["example"], "rows": sorted(g["rows"])[:3]})
print(json.dumps({"items": items, "infra": dict(infra)}))
PYEOF
)" || { echo "rig-friction-digest: digest failed" >&2; exit 1; }

if [ "$JSON" = "1" ]; then printf '%s\n' "$DIGEST_JSON"; exit 0; fi

# render ITEMS_JSON [LANE...] -> markdown on stdout
render() {
  python3 - "$@" <<'PYEOF'
import json, sys
doc, lanes = json.loads(sys.argv[1]), sys.argv[2:]
items = doc["items"] if isinstance(doc, dict) else doc
want = lambda lane: not lanes or lane in lanes
sel = sorted([i for i in items if i["lane"] == "ailang"], key=lambda i: -i["count"])
if sel and want("ailang"):
    print("### ailang — error kinds, across benchmarks (%d)" % len(sel))
    for i in sel:
        print("- **%d×** in %s (%s): `%s`" % (i["count"], ", ".join("`%s`" % b for b in i["benchmarks"]), ", ".join(i["harnesses"]), i["example"].replace("`", "'")))
        for r in i["rows"][:2]: print("  - row: `%s`" % r)
    print()
for lane in ["motoko-gap", "harness-gap", "budget", "sporadic"]:
    sel = [i for i in items if i["lane"] == lane]
    if not sel or not want(lane): continue
    print("### %s (%d)" % (lane, len(sel)))
    for i in sel:
        sc = "  ".join("%s %d/%d" % (k, a, n) for k, (a, n) in i["score"].items())
        print("- **`%s`** — %s — %s — window %s" % (i["id"], sc, ", ".join("%s %d" % kv for kv in i["categories"].items()), ",".join(i["window"])))
        if lane == "sporadic": continue
        for h, e in i["errors"]: print("  - %s: `%s`" % (h, e.replace("`", "'")))
        for r in i["rows"][:2]: print("  - row: `%s`" % r)
    print()
infra = doc.get("infra", {}) if isinstance(doc, dict) else {}
if infra and want("infra"):
    print("### harness infrastructure")
    print("- " + ", ".join("%s %d" % kv for kv in sorted(infra.items())) + " — the harness returned without using tools (counted as failures above)")
    print()
PYEOF
}

printf '## Rig friction digest (%s)\n\n' "$(date +%F)"
printf 'Signal benchmarks with failures, per-benchmark window of the last %s banks that ran each. Lane hints follow PROGRAM.md §4; triage decides.\n\n' "$NVER"
render "$DIGEST_JSON"

[ "$ROUTE" = "1" ] || exit 0

# ── routing: one message per lane, NEW signatures only ──────────────────────────
mkdir -p "$(dirname "$STATE")"; touch "$STATE"
NEW_JSON="$(python3 - "$DIGEST_JSON" "$STATE" <<'PYEOF'
import json, sys
doc = json.loads(sys.argv[1]); seen = set(open(sys.argv[2]).read().splitlines())
# An AILANG error kind is routed only once it RECURS (>= 2 occurrences): a single
# small-model slip is not worth a triage run. Every kind still shows in the digest.
print(json.dumps({"items": [i for i in doc["items"] if i["lane"] in ("ailang", "motoko-gap", "harness-gap")
                            and (i["lane"] != "ailang" or i["count"] >= 2)
                            and "%s:%s" % (i["id"], i["lane"]) not in seen], "infra": {}}))
PYEOF
)"
send() { # INBOX TITLE BODY
  AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac \
    ailang messages send "$1" "$3" --title "$2" --from rig-friction-digest --force
}
remember() { # ITEMS_JSON LANE...
  python3 - "$@" <<'PYEOF' >> "$STATE"
import json, sys
for i in json.loads(sys.argv[1])["items"]:
    if i["lane"] in sys.argv[2:]: print("%s:%s" % (i["id"], i["lane"]))
PYEOF
}
count() { python3 -c 'import json,sys; print(sum(1 for i in json.loads(sys.argv[1])["items"] if i["lane"] in sys.argv[2:]))' "$@"; }
rc=0
if [ "$(count "$NEW_JSON" ailang)" -gt 0 ]; then
  BODY="$(printf 'From the local rig (qwen3.8 27B × opencode, pi, motoko; signal benchmarks only). Compile/runtime errors the local agents hit, grouped by error kind across benchmarks — candidate AILANG friction (error message, missing builtin, dialect trap, teaching prompt). Counts are small-model slips; a kind that recurs across benchmarks or harnesses is the signal. Read the transcripts (`ailang chains chat`, or the row files) before deciding the lane; a benchmark-spec defect is also possible.\n\n%s' "$(render "$NEW_JSON" ailang)")"
  if send ailang-core "Rig friction: $(count "$NEW_JSON" ailang) AILANG candidate(s)" "$BODY"; then remember "$NEW_JSON" ailang; else rc=1; fi
fi
if [ "$(count "$NEW_JSON" motoko-gap harness-gap)" -gt 0 ]; then
  if [ -f "$HOME/.ailang/state/mission-motoko.disabled" ]; then INBOX=user; NOTE="The motoko mission is PAUSED (mission-motoko.disabled; D-MOTOKO-RESTART-1), so this comes to you — nothing will act on it until the mission restarts or someone runs the motoko-analyzer cycle."; else INBOX=mission-motoko; NOTE="For the motoko mission's Gate 1 (segment.sh) — same model, same benchmarks, harness differs."; fi
  BODY="$(printf 'Harness gaps on the local rig (same model, same benchmarks). %s\n\n%s' "$NOTE" "$(render "$NEW_JSON" motoko-gap harness-gap)")"
  if send "$INBOX" "Rig friction: $(count "$NEW_JSON" motoko-gap harness-gap) motoko harness gap(s)" "$BODY"; then remember "$NEW_JSON" motoko-gap harness-gap; else rc=1; fi
fi
exit $rc
