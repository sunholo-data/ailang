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

Then `ailang eval-suite --agent --models pi-qwen3-5-35b-a3b-mxfp8 ...` works locally.
Add more local models by appending `{ "id": "<ollama-tag>" }` to the models array.
