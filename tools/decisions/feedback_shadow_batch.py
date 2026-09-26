#!/usr/bin/env python3
"""feedback_shadow_batch — run the feedback-gate shadow program over a batch of
public-feedback messages (the same program the coordinator runs live) and
report what the gate WOULD do on System One answers, plus category agreement
with the submitter's declared category (the gate's mismatch check).

This is the offline stand-in for live shadowing while the feedback gate is not
enabled in prod (config.cloud.yaml has no feedback_gate block — #900).

Usage:
  ailang messages list --inbox public-feedback --json --limit 100 > pf.json
  tools/decisions/feedback_shadow_batch.py pf.json out.jsonl [direct|openrouter] > report.md
Needs TYPESAFE_API_KEY (direct) or OPENROUTER_API_KEY (openrouter). Nothing acts.
"""
import json, os, subprocess, sys, time
from collections import Counter

HERE = os.path.dirname(os.path.abspath(__file__))
PROGRAM_DIR = os.path.normpath(os.path.join(HERE, "..", "..", "internal", "feedbackgate", "shadow"))

def would(declared, r):
    """Mirror of internal/feedbackgate/shadow.go shadowVerdict (p >= 0.5 == true)."""
    if r.get("error"):
        return "file", "classifier_error"
    if r["injection"].get("p", 0) >= 0.5:
        return "reject", "classifier_prompt_injection"
    if r["value"].get("label") == "none":
        return "file", "classifier_no_value"
    if r["genuine"].get("p", 0) < 0.5:
        return "file", "classifier_not_genuine"
    if r["category"].get("choice") != declared:
        return "file", "classifier_category_mismatch"
    return "dispatch", "passed"

def main():
    src, out, transport = sys.argv[1], sys.argv[2], (sys.argv[3] if len(sys.argv) > 3 else "direct")
    msgs = json.load(open(src))
    msgs = [m for m in msgs if m.get("message_type") == "feedback" and m.get("category") and m.get("payload")]
    rows = []
    with open(out, "w") as f:
        for m in msgs:
            body = m["payload"]
            inp = json.dumps({"category": m["category"], "from": m.get("from_agent") or "", "inbox": m.get("to_inbox") or "", "body": body[:8000]})
            t0 = time.time()
            p = subprocess.run(["ailang", "run", "--caps", "Net,Env,IO,Clock", "--entry", "main", "feedback_shadow.ail", transport, "-", inp],
                               cwd=PROGRAM_DIR, capture_output=True, text=True, env={**os.environ, "AILANG_RELAX_MODULES": "1"})
            line = next((l for l in reversed(p.stdout.splitlines()) if l.strip().startswith("{")), None)
            r = json.loads(line) if line else {"error": "no output: " + (p.stderr.strip().splitlines() or ["?"])[-1], "genuine": {}, "injection": {}, "category": {}, "value": {}}
            a, reason = would(m["category"], r)
            row = {"id": m["id"], "title": m.get("title"), "declared": m["category"], "would_action": a, "would_reason": reason, "wall_ms": int((time.time() - t0) * 1000), "shadow": r}
            rows.append(row); f.write(json.dumps(row) + "\n"); f.flush()
            print(f"{m['id']}  {m['category']:<11} -> {a:<8} {reason:<30} {r.get('latency_ms','?')}ms", file=sys.stderr)

    ok = [r for r in rows if not r["shadow"].get("error")]
    P = print
    P(f"# Feedback-gate shadow — offline batch over `public-feedback` ({len(rows)} submissions, transport `{transport}`)")
    P("")
    P("The gate is NOT enabled in prod (no `feedback_gate` block in config.cloud.yaml, #900), so there is no Haiku arm to compare against and no human disposition label. This table shows what the gate **would do** on System One answers, and how often the model's category agrees with the **submitter's declared category** — the gate's own mismatch check. Nothing acted.")
    P("")
    P("## Would-do distribution")
    P("")
    P("| would_action | would_reason | n |")
    P("|---|---|---|")
    for (a, rsn), n in sorted(Counter((r["would_action"], r["would_reason"]) for r in rows).items(), key=lambda x: -x[1]):
        P(f"| {a} | {rsn} | {n} |")
    P("")
    agree = sum(1 for r in ok if r["shadow"]["category"].get("choice") == r["declared"])
    inj = [r for r in ok if r["shadow"]["injection"].get("p", 0) >= 0.5]
    lat = [r["shadow"]["latency_ms"] for r in ok if "latency_ms" in r["shadow"]]
    cost = sum(r["shadow"].get("list_price_usd", 0) for r in ok)
    P(f"**Category agrees with the submitter's declared category:** {agree}/{len(ok)}. **Injection ≥ 0.5:** {len(inj)}. **Model errors:** {len(rows)-len(ok)}. **Mean model latency:** {sum(lat)/len(lat):.0f} ms. **List-price cost for the batch:** ${cost:.4f}.")
    P("")
    P("## Per submission")
    P("")
    P("| id | declared | Jev category (conf) | genuine | injection | value | would |")
    P("|---|---|---|---|---|---|---|")
    for r in rows:
        s = r["shadow"]
        if s.get("error"):
            P(f"| {r['id']} | {r['declared']} | ERR {s['error'][:40]} | | | | file |"); continue
        c = s["category"]; mark = "✓" if c.get("choice") == r["declared"] else "✗"
        P(f"| {r['id']} | {r['declared']} | {mark} {c.get('choice')} ({c.get('confidence',0):.2f}) | {s['genuine'].get('p',0):.2f} | {s['injection'].get('p',0):.2f} | {s['value'].get('label')} ({s['value'].get('score',0):.2f}) | {r['would_action']} / {r['would_reason'].replace('classifier_','')} |")
    P("")
    P(f"Titles for the disagreements (declared vs Jev):")
    P("")
    for r in ok:
        c = r["shadow"]["category"]
        if c.get("choice") != r["declared"]:
            P(f"- `{r['id']}` declared **{r['declared']}**, Jev **{c.get('choice')}** ({c.get('confidence',0):.2f}): {(r['title'] or '')[:90]}")

if __name__ == "__main__":
    main()
