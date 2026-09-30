#!/usr/bin/env bash
# measure_ollama_credit_rate.sh — empirical Ollama Cloud credit-rate A/B (V36/V46 method)
# Usage: measure_ollama_credit_rate.sh <tag1> [tag2 ...]
#   e.g. measure_ollama_credit_rate.sh glm-5.3-flash glm-5.3
#
# For each model: snapshot the /api/usage session numerator, burn 24 one-shot
# requests with FRESH ~27KB prompts (anti-caching — V46: cached/agentic shapes
# meter ~2x cheaper; always record the workload shape alongside the rate),
# snapshot again, then rate = numerator delta / real tokens burned.
# Cross-tag ratio = relative credit cost.
#
# Method facts (design doc m-ollama-cloud-provider):
#   - /api/usage session usage is ROUNDED TO 3 DECIMALS — an arm needs roughly
#     12k tokens per 0.001 expected delta (default 24 requests ≈ 145k tokens) (level-1 models need ~150k+ to resolve)
#   - the numerator has NO published denominator; these are RELATIVE credit
#     units, not dollars — dollars come from the OpenRouter-twin imputation (D1)
#   - keep this single command's runtime free of other traffic on the account:
#     anything else on the same flat route (e.g. a chat agent) between the two
#     snapshots silently inflates the rate
#
# Published side-data: model pages state a usage level (low/medium/high,
# ~3-4x per level) — cross-check measured ratios against it.

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <ollama-tag> [tag2 ...]  (NO :cloud suffix — appended automatically)"
    echo "Burns 24 x ~27KB prompts per tag of flat-rate quota."
    exit 1
fi

if [ -z "${OLLAMA_API_KEY:-}" ]; then
    echo "OLLAMA_API_KEY not set — the quota gauge needs it (device-key inference does not)"
    exit 1
fi

WORK=$(mktemp -d /tmp/oc_credit.XXXXXX)
trap 'rm -rf "$WORK"' EXIT

N=24
python3 - "$WORK" "$N" "$@" <<'EOF'
import json, sys
workdir, n, tags = sys.argv[1], int(sys.argv[2]), sys.argv[3:]
base = ("AILANG determinism doctrine: every effect is typed, every result reproducible. " * 340)
for i in range(n):
    for tag in tags:
        safe = tag.replace(".", "_")
        payload = {"model": f"{tag}:cloud", "max_tokens": 200,
                   "messages": [{"role": "user",
                                 "content": f"credit-rate probe {i} for {tag}\n{base}\nReply with only: {i}"}]}
        with open(f"{workdir}/payload_{safe}_{i}.json", "w") as f:
            json.dump(payload, f)
EOF

snap() { curl -s https://ollama.com/api/usage -H "Authorization: Bearer $OLLAMA_API_KEY" \
  | python3 -c 'import sys,json; print("%.3f" % json.load(sys.stdin)["limits"]["session"]["usage"])'; }

for TAG in "$@"; do
    SAFE=$(echo "$TAG" | tr '.' '_')
    LOG="$WORK/rate_${SAFE}.txt"; : > "$LOG"
    B0=$(snap)
    for i in $(seq 0 $((N - 1))); do
        curl -s localhost:11434/v1/chat/completions -H "Content-Type: application/json" \
            -d @"$WORK/payload_${SAFE}_$i.json" \
            | OC_LOG="$LOG" python3 -c 'import sys,json,os; d=json.load(sys.stdin); u=d.get("usage"); print((u["prompt_tokens"]+u["completion_tokens"]) if u else "ERR", file=open(os.environ["OC_LOG"],"a"))' \
            || echo CURL_FAIL >> "$LOG"
    done
    B1=$(snap)
    echo "MARK $TAG $B0 $B1" >> "$WORK/ab.log"
    echo "burned $TAG: $B0 → $B1"
done

python3 - "$WORK" "$@" <<'EOF'
import sys
workdir, tags = sys.argv[1], sys.argv[2:]
marks = [l.split() for l in open(f"{workdir}/ab.log") if l.startswith("MARK ")]
print(f"\n{'model':<22} {'tokens':>8} {'credits':>9} {'units/M':>8}")
rates = {}
for tag, b0, b1 in [(m[1], float(m[2]), float(m[3])) for m in marks]:
    safe = tag.replace(".", "_")
    lines = [x.strip() for x in open(f"{workdir}/rate_{safe}.txt")]
    toks = sum(int(x) for x in lines if x.isdigit())
    errs = sum(1 for x in lines if not x.isdigit())
    delta = b1 - b0
    rate = round(delta / toks * 1e6, 4) if toks and delta else 0.0
    rates[tag] = rate
    res = "  ⚠ at/below the 0.001 rounding resolution — re-run with more tokens" if not rate else ""
    fail = f"  ⚠ {errs} failed requests" if errs else ""
    print(f"{tag:<22} {toks:>8} {delta:>+9.3f} {rate:>8.4f}{res}{fail}")
if len(rates) == 2 and all(rates.values()):
    (a, b) = list(rates)
    print(f"\ncredit ratio: {b} costs {rates[b]/rates[a]:.2f}x the credits per token of {a}")
    print("(rates are one-shot-shape units/M — agentic workloads meter ~2x cheaper, V46)")
EOF