# pi + Ollama (local models) setup

pi (`@mariozechner/pi-coding-agent`) has NO built-in ollama provider, but it can
talk to ollama's OpenAI-compatible endpoint via a custom provider in
`~/.pi/agent/models.json`. Without this file, `pi --model ollama/<id>` fails with
"model not found".

One-time setup on a rig with ollama:

```bash
mkdir -p ~/.pi/agent
cp tools/setup/pi-ollama-models.json ~/.pi/agent/models.json
# verify:
pi --list-models | grep ollama
```

Then `ailang eval-suite --agent --models pi-qwen3-6-35b-a3b-mxfp8 ...` works locally.
Add more local models by appending `{ "id": "<ollama-tag>" }` to the models array.

## The rig lease header

The `ollama` provider sends `X-Rig-Lease` so the rig GPU gateway on :11434
(M-RIG-GPU-ADMISSION-GATEWAY) admits the lock holder's long work. It is a `!`
command value, `printf %s "${AILANG_RIG_LEASE:-none}"`, not a `${AILANG_RIG_LEASE}`
template: pi refuses to start when a templated variable is unset, and pi run by
hand or by a mission loop has no lease. Measured on pi 0.85.1 (2026-09-28): unset
sends `none`, set sends the token, and pi treats the gateway's 423 as final.
