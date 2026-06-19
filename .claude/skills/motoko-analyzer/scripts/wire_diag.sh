#!/usr/bin/env bash
# Gate 2 (DIFF) — capture motoko's EXACT HTTP wire request+response for failing
# benchmarks and classify each disengaged turn's per-call finish_reason. This is the
# step that surfaces truncation / dropped fields / parse gaps that the rotation
# result-JSON (summary finish_reason only) hides. Small GPU; lock-respecting.
#
#   wire_diag.sh <benchmark>[,<benchmark>...]
#   wire_diag.sh                       # default: top always-disengage benchmarks
#
# Requires the wire logger (internal/ai/openai logAIWire) in the installed ailang.
# Output: /tmp/motoko-analyzer-wire.jsonl + a per-call finish_reason breakdown.
set -uo pipefail
REPO="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$REPO" || exit 1
[ -f "$HOME/.config/ailang/secrets.env" ] && . "$HOME/.config/ailang/secrets.env"
export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"

BENCHES="${1:-csv_to_json_converter,graph_bfs,config_file_parser,symbolic_diff}"
MODEL="${MOTOKO_ANALYZER_MODEL:-motoko-local-qwen3-6-35b-a3b-mxfp8}"
WIRE=/tmp/motoko-analyzer-wire.jsonl
OUT=/tmp/motoko-analyzer-wirecap
LOCKD="$HOME/.ailang/state/rig.lock.d"; SENT="$HOME/.ailang/state/ai-http-log"
cleanup(){ rm -rf "$LOCKD"; rm -f "$SENT"; }
trap cleanup EXIT
t=0; while ! mkdir "$LOCKD" 2>/dev/null; do t=$((t+1)); [ $((t%10)) -eq 1 ] && echo "[wire] rig busy ($t)…"; sleep 30; [ $t -gt 80 ] && { echo "[wire] lock timeout"; exit 1; }; done
echo "$$ $(date -u +%FT%TZ) MOTOKO-ANALYZER-WIRE" > "$LOCKD/holder"
echo "$WIRE" > "$SENT"; rm -f "$WIRE"; rm -rf "$OUT"
echo "[wire] capturing $MODEL on: $BENCHES"
perl -e 'alarm 4000; exec @ARGV' \
  ailang eval-suite --agent --models "$MODEL" --benchmarks "$BENCHES" \
  --langs ailang --parallel 1 --trials 1 --no-rig-lock --output "$OUT" 2>&1 | tail -3
rm -f "$SENT"
python3 - "$WIRE" <<'PY'
import json, sys, collections
fin=collections.Counter(); reqfields=set(); reqmax=set(); n_resp=0
for l in open(sys.argv[1]):
    r=json.loads(l)
    if r.get('kind')=='http_request':
        b=json.loads(r['body']); reqfields|=set(b.keys()); reqmax.add(b.get('max_tokens') or b.get('max_completion_tokens'))
    elif r.get('kind')=='http_response':
        try: b=json.loads(r['body'])
        except: continue
        ch=(b.get('choices') or [{}])[0]; msg=ch.get('message',{})
        if (msg.get('tool_calls') or []): continue   # engaged turn
        n_resp+=1; fin[ch.get('finish_reason')]+=1
print(f"[wire] motoko request fields: {sorted(reqfields)}  max_tokens={reqmax}")
print(f"[wire] DISENGAGED turns by per-call finish_reason: {dict(fin)}  (length=TRUNCATION, stop=genuine)")
print("[wire] => if 'length' dominates: token-budget/inference-config lever. if 'stop': prompt/model-decision lever.")
PY
echo "[wire] raw capture: $WIRE"
