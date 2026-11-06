#!/bin/bash
# Replay one captured agent turn against Ollama with request variants, under this
# script's OWN rig lease (cooperative handoff). Diagnostics for the pilot-1 finding:
# after a tool result the gemma4 parser starts in thinking mode, so a fine-tune that
# never closes the thought channel has its tool call filed as reasoning.
#
#   tools/finetune/replay_turn.sh <trace.jsonl> <request-index> <model>
set -u
REPO=/Users/voightkampff/dev/sunholo-data/ailang
TRACE=${1:?trace}; IDX=${2:?index}; MODEL=${3:?model}
cd "$REPO" || exit 1
source tools/launchd/rig-lock.sh
REQ=ft-replay
rig_lock_request_yield "$REQ" 900 || true
rig_lock_acquire wait "$REQ" || exit 1
rig_lock_clear_yield "$REQ"
OUTD=$(mktemp -d)
python3 -I - "$TRACE" "$IDX" "$MODEL" "$OUTD" <<'EOF'
import json,sys
trace,idx,model,outd=sys.argv[1],int(sys.argv[2]),sys.argv[3],sys.argv[4]
rows=[json.loads(l) for l in open(trace)]
chat=[r for r in rows if isinstance(r.get('request'),dict) and r['request'].get('messages')]
base=dict(chat[idx]['request']); base.update(model=model, stream=False, max_completion_tokens=2000); base.pop('stream_options',None)
variants={"as_is":{}, "think_false":{"think":False}, "reasoning_effort_none":{"reasoning_effort":"none"}}
for k,v in variants.items():
    json.dump(dict(base,**v),open(f"{outd}/{k}.json","w"))
print(" ".join(variants))
EOF
for v in as_is think_false reasoning_effort_none; do
  curl -s --max-time 900 http://127.0.0.1:11434/v1/chat/completions -H "Content-Type: application/json" \
    -H "X-Rig-Lease: $AILANG_RIG_LEASE" -d @"$OUTD/$v.json" > "$OUTD/$v.out"
  python3 -I -c "
import json,sys; d=json.load(open(sys.argv[1]))
if 'choices' not in d: print(sys.argv[2], 'ERROR', str(d)[:200]); raise SystemExit
m=d['choices'][0]['message']; tc=m.get('tool_calls') or []
print(sys.argv[2], 'finish=',d['choices'][0]['finish_reason'], 'tool_calls=',[(t['function']['name'],t['function']['arguments'][:80]) for t in tc],
      '| content=',repr((m.get('content') or '')[:80]), '| reasoning=',repr((m.get('reasoning') or '')[:80]))" "$OUTD/$v.out" "$v"
done
