#!/usr/bin/env bash
# Embedder bake-off for microRAG: compares retrieval embedders head-to-head on
# eval pass rate. Three arms on the same model/benchmarks ($0, local):
#   off      — microRAG disabled (baseline)
#   gemma_on — microRAG on, corpus embedded with embeddinggemma (+ task prefixes)
#   nomic_on — microRAG on, corpus embedded with nomic-embed-text (+ task prefixes)
#
# microRAG uses ONE configured embedder at a time (corpus + query share a vector
# space), so each embedder arm re-embeds the corpus before evaluating. Switches
# ~/.ailang/config.yaml between arms and restores it to the winner at the end is
# left to the caller — this script leaves the LAST-RUN embedder configured.
#
# Usage: tools/embedder-ab.sh [model] [benchmarks-comma-list]
set -uo pipefail
cd "$(dirname "$0")/.."
MODEL="${1:-opencode-qwen3-5-35b-a3b-mxfp8}"
CORE="${2:-$(cat /tmp/core_benchmarks.txt)}"
CFG="$HOME/.ailang/config.yaml"
OUT=/tmp/embedder_cmp
TRIALS="${TRIALS:-2}"
rm -rf "$OUT"; mkdir -p "$OUT"
log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a "$OUT/run.log"; }
reap() { lsof -ti tcp:8080 2>/dev/null | xargs -r kill -9 2>/dev/null; pkill -f env-server 2>/dev/null; pkill -f run-agent 2>/dev/null; sleep 1; }
run_eval() { # $1=mode $2=outdir
  reap
  ailang eval-suite --agent --models "$MODEL" --benchmarks "$CORE" \
    --langs ailang --microrag "$1" --trials "$TRIALS" --parallel 1 \
    --output "$2" >> "$OUT/eval.log" 2>&1
}
rate() { python3 -c "import json,glob; r=[json.load(open(f)) for f in glob.glob('$1/agent/*.json')]; p=sum(1 for x in r if x.get('stdout_ok')); t=len(r); print('%d/%d (%.1f%%)'%(p,t,100*p/max(t,1)))"; }
set_embedder() { sed -i '' "s/^\( *model: \).*/\1$1/" "$CFG"; log "embedder -> $1"; }

log "=== embedder A/B: model=$MODEL benchmarks=$(echo $CORE|tr ',' ' '|wc -w) trials=$TRIALS ==="

# Arm 1: OFF baseline (no embedder used)
log "arm 1/3: microRAG OFF"; run_eval off "$OUT/off"

# Arm 2: gemma + ON (corpus already gemma+prefix from M-EMBED-TASK-PREFIX; re-embed to be safe)
set_embedder embeddinggemma
log "re-embed (embeddinggemma)"; make brain-index-syntax-reset >> "$OUT/reembed_gemma.log" 2>&1
log "arm 2/3: microRAG ON + embeddinggemma"; run_eval on "$OUT/gemma_on"

# Arm 3: nomic + ON
ollama list 2>/dev/null | grep -q nomic-embed-text || ollama pull nomic-embed-text >/dev/null 2>&1
set_embedder nomic-embed-text
log "re-embed (nomic-embed-text)"; make brain-index-syntax-reset >> "$OUT/reembed_nomic.log" 2>&1
log "arm 3/3: microRAG ON + nomic-embed-text"; run_eval on "$OUT/nomic_on"

reap
log "=== RESULTS ($MODEL) ==="
log "off (baseline) : $(rate "$OUT/off")"
log "on + gemma     : $(rate "$OUT/gemma_on")"
log "on + nomic     : $(rate "$OUT/nomic_on")"
log "(config left at: $(grep -A3 '^embeddings:' "$CFG" | grep 'model:' | tr -d ' '))"
