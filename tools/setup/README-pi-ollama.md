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

Then `ailang eval-suite --agent --models pi-qwen3-6-35b-a3b-mxfp8 ...` works locally
while the rig lock exports `AILANG_RIG_LEASE`.
Add more local models by appending `{ "id": "<ollama-tag>" }` to the models array.

## The rig lease header

The separate `ollama-rig` provider sends `X-Rig-Lease` so the rig GPU gateway on
:11434 admits the lock holder's long work. Its `${AILANG_RIG_LEASE}` template
deliberately fails loudly when the lease is absent. The everyday `ollama`
provider remains lease-free for interactive and cloud routes; eval registry
rows for on-device pi models must use `ollama-rig/...`.
