#!/usr/bin/env python3
"""lane_shadow_report — aggregate a lane_shadow JSONL run into the markdown report.

Usage: tools/decisions/lane_shadow_report.py <lane_shadow_<date>.jsonl> [interpretation.md — appended verbatim]
Prints markdown to stdout. Pure aggregation over banked rows (D6): nothing here calls a model.
"""
import json, math, statistics, sys
from collections import defaultdict

def load(path):
    return [json.loads(l) for l in open(path) if l.strip()]

def entropy_norm(dist):
    ps = [p for p in dist.values() if p > 0]
    if len(dist) < 2 or not ps:
        return 0.0
    h = -sum(p * math.log(p) for p in ps)
    return h / math.log(len(dist))

def tertiles(xs):
    s = sorted(xs)
    n = len(s)
    return s[n // 3], s[(2 * n) // 3]

def band(x, lo, hi):
    return "low" if x <= lo else ("mid" if x <= hi else "high")

def main():
    rows = load(sys.argv[1])
    by = defaultdict(dict)
    for r in rows:
        by[r["path"]][r["arm"]] = r
    docs = sorted(by)
    out = []
    P = out.append

    arms = {}
    for arm in ("jev", "llm"):
        rs = [by[d][arm] for d in docs if arm in by[d]]
        ok = [r for r in rs if not r["error"]]
        timeouts = [r for r in rs if r["error"] and ("timeout" in r["error"].lower() or "deadline" in r["error"].lower() or r["latency_ms"] >= 30000)]
        errs = [r for r in rs if r["error"] and r not in timeouts]
        def pred(r, arm=arm):
            a = r["answers"].get("lane", {})
            return a.get("choice") if arm == "jev" else a.get("label")
        def conf(r, arm=arm):
            a = r["answers"].get("lane", {})
            if arm == "jev":
                return a.get("confidence")
            d = a.get("self_reported") or {}
            return (1.0 - entropy_norm(d)) if isinstance(d, dict) and d else None
        agree = [r for r in ok if pred(r) == r["label"]]
        lat = [r["latency_ms"] for r in rs]
        arms[arm] = dict(rs=rs, ok=ok, timeouts=timeouts, errs=errs, agree=agree, lat=lat,
                         cost=sum(r["cost_usd"] for r in rs), tin=sum(r["input_tokens"] for r in rs), tout=sum(r["output_tokens"] for r in rs),
                         pred=pred, conf=conf)

    P("# M-AI-DECIDE-SYSTEM-ONE — shadow lane-router report")
    P("")
    P(f"**Run**: `{sys.argv[1].split('/')[-1]}` · **docs**: {len(docs)} · **arms**: jev = `{arms['jev']['rs'][0]['model'] if arms['jev']['rs'] else '?'}` via OpenRouter `/api/alpha/decisions`; llm = `{arms['llm']['rs'][0]['model'] if arms['llm']['rs'] else '?'}` via OpenRouter chat completions, strict `json_schema` from the same `Question` list, per-label probabilities requested. Same `std/net` transport and 30 s deadline for both. **Nothing acted on any answer.**")
    P("")
    P("**Label set**: hand-read declared PROGRAM.md lanes (`tools/decisions/routed_docs.sh`, each row carries the quoted sentence). The design doc's \"45 routed docs\" were 45 docs with a *routing section*; **20** state their lane legibly. Treat every number below as a pilot on n=20, not a verdict.")
    P("")
    P("## Headline")
    P("")
    P("| Arm | Answered | Timeouts (30 s) | Other errors | Lane agreement with declared label | Mean / median latency | Total cost | Tokens in/out |")
    P("|---|---|---|---|---|---|---|---|")
    for arm in ("jev", "llm"):
        a = arms[arm]
        n_ok = len(a["ok"])
        agr = f"{len(a['agree'])}/{n_ok} = {100*len(a['agree'])/n_ok:.0f}%" if n_ok else "—"
        P(f"| {arm} | {n_ok}/{len(a['rs'])} | {len(a['timeouts'])} | {len(a['errs'])} | **{agr}** | {statistics.mean(a['lat']):.0f} / {statistics.median(a['lat']):.0f} ms | ${a['cost']:.4f} | {a['tin']}/{a['tout']} |")
    # jev vs llm agreement
    both = [d for d in docs if "jev" in by[d] and "llm" in by[d] and not by[d]["jev"]["error"] and not by[d]["llm"]["error"]]
    jl = [d for d in both if arms["jev"]["pred"](by[d]["jev"]) == arms["llm"]["pred"](by[d]["llm"])]
    P("")
    P(f"**Jev ↔ LLM lane agreement** (docs both answered): {len(jl)}/{len(both)}" + (f" = {100*len(jl)/len(both):.0f}%" if both else ""))
    P("")
    # per-label breakdown
    P("## By declared lane")
    P("")
    P("| Declared lane | n | Jev correct | LLM correct (of answered) |")
    P("|---|---|---|---|")
    labels = sorted({by[d]["jev"]["label"] for d in docs if "jev" in by[d]})
    for lab in labels:
        ds = [d for d in docs if by[d].get("jev", {}).get("label") == lab]
        jc = sum(1 for d in ds if "jev" in by[d] and not by[d]["jev"]["error"] and arms["jev"]["pred"](by[d]["jev"]) == lab)
        la = [d for d in ds if "llm" in by[d] and not by[d]["llm"]["error"]]
        lc = sum(1 for d in la if arms["llm"]["pred"](by[d]["llm"]) == lab)
        P(f"| {lab} | {len(ds)} | {jc}/{len(ds)} | {lc}/{len(la)} |")
    P("")
    # confidence stratification
    P("## Does confidence predict correctness?")
    P("")
    P("Jev `confidence` is the vendor's calibrated statistic; the LLM column is `1 − normalised entropy` of its *self-reported* distribution (a proxy, not a calibration claim). Tertiles are within-arm.")
    P("")
    P("| Arm | Band | n | Correct | Mean confidence |")
    P("|---|---|---|---|---|")
    for arm in ("jev", "llm"):
        a = arms[arm]
        rs = [r for r in a["ok"] if a["conf"](r) is not None]
        if len(rs) < 3:
            P(f"| {arm} | — | {len(rs)} | — | — |")
            continue
        lo, hi = tertiles([a["conf"](r) for r in rs])
        groups = defaultdict(list)
        for r in rs:
            groups[band(a["conf"](r), lo, hi)].append(r)
        for b in ("low", "mid", "high"):
            g = groups.get(b, [])
            if not g:
                continue
            c = sum(1 for r in g if a["pred"](r) == r["label"])
            P(f"| {arm} | {b} | {len(g)} | {c}/{len(g)} | {statistics.mean(a['conf'](r) for r in g):.2f} |")
    P("")
    # gate simulation — what a confidence-gated consumer would have done (Jev only: the LLM proxy is not a calibration)
    P("## If a consumer had gated on Jev confidence")
    P("")
    P("`gate(answer, t)` per the package: `Act` when confidence ≥ t, else `Escalate` to a frontier turn. Counts over the 20 Jev rows.")
    P("")
    P("| Threshold | Acted | Acted & correct | Acted & wrong | Escalated | Escalated & would have been wrong |")
    P("|---|---|---|---|---|---|")
    jr = arms["jev"]["ok"]; jp = arms["jev"]["pred"]; jc = arms["jev"]["conf"]
    for t in (0.5, 0.7, 0.9, 0.95):
        act = [r for r in jr if jc(r) >= t]; esc = [r for r in jr if jc(r) < t]
        ac = sum(1 for r in act if jp(r) == r["label"]); ew = sum(1 for r in esc if jp(r) != r["label"])
        P(f"| {t:.2f} | {len(act)} | {ac} | {len(act)-ac} | {len(esc)} | {ew} |")
    P("")
    # per-doc table
    P("## Per document")
    P("")
    P("| Doc | Declared | Jev lane (conf) | Jev touches_core | Jev severity | LLM lane (top p) | Jev ms | LLM ms |")
    P("|---|---|---|---|---|---|---|---|")
    for d in docs:
        j = by[d].get("jev"); l = by[d].get("llm")
        name = d.split("/")[-1].replace(".md", "")
        if j and not j["error"]:
            la = j["answers"]["lane"]; jl_ = f"{la['choice']} ({la['confidence']:.2f})"
            tc = f"{j['answers']['touches_core']['p']:.2f}"
            sv = f"{j['answers']['severity']['score']:.2f}"
        else:
            jl_ = f"ERR {j['error'][:40] if j else '—'}"; tc = sv = "—"
        if l and not l["error"]:
            ll = l["answers"]["lane"]; ll_ = f"{ll['label']} ({ll['top_p']:.2f})"
        else:
            ll_ = "TIMEOUT" if l and l["latency_ms"] >= 30000 else f"ERR {l['error'][:40] if l else '—'}"
        mark = lambda pred, lab: "✓" if pred == lab else "✗"
        jm = mark(j["answers"]["lane"]["choice"], j["label"]) if j and not j["error"] else " "
        lm = mark(l["answers"]["lane"]["label"], l["label"]) if l and not l["error"] else " "
        P(f"| {name} | {j['label'] if j else '?'} | {jm} {jl_} | {tc} | {sv} | {lm} {ll_} | {j['latency_ms'] if j else '—'} | {l['latency_ms'] if l else '—'} |")
    P("")
    P("## Reading it")
    P("")
    P("- Rows are banked in full (model, id, distributions, usage, latency) under `.ailang/state/decisions/`; this file is a pure aggregation and can be regenerated with `tools/decisions/lane_shadow_report.py`.")
    P("- Timeouts are the Net effect's 30 s deadline — the bounded wait the design doc committed to; they are counted, never retried.")
    P("- n=20 with a 12/6/1/1 label skew: an arm that always says `extension` scores 60%. Compare arms to each other and to that baseline, not to 100%.")
    if len(sys.argv) > 2:
        out.append("")
        out.append(open(sys.argv[2]).read().rstrip())
    print("\n".join(out))

if __name__ == "__main__":
    main()
