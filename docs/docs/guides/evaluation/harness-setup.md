# Agent Harness Setup

AILANG's agent eval mode runs benchmarks through agentic CLI tools — the same tools
developers use interactively. This guide covers how to install and authenticate each
supported harness for `ailang eval-suite --agent`.

## Supported Harnesses

| Harness | CLI tool | Models in `models.yml` | Install |
|---------|----------|------------------------|---------|
| **claude** | Claude Code (`claude`) | `claude-sonnet-4-6`, `claude-haiku-4-5` | `npm install -g @anthropic-ai/claude-code` |
| **gemini** | Gemini CLI (`gemini`) | `gemini-3-flash`, `gemini-3-1-pro` | `npm install -g @google/generative-ai-cli` |
| **codex** | OpenAI Codex CLI (`codex`) | `gpt5-4`, `gpt5-1-instant` | `npm install -g @openai/codex` |
| **opencode** | opencode (`opencode`) | `opencode-haiku`, `opencode-sonnet-4-6`, `opencode-gemini-3-flash` | `npm install -g opencode-ai` |

## Quick Check

```bash
claude --version
gemini --version
codex --version
opencode --version
```

## Claude Code (`claude`)

```bash
npm install -g @anthropic-ai/claude-code
export ANTHROPIC_API_KEY=sk-ant-...
claude --version
```

Verify agentic mode works:

```bash
echo "Write hello world to solution.py" | claude --print \
  --output-format stream-json --permission-mode bypassPermissions
```

The `--permission-mode bypassPermissions` flag is what the executor uses to auto-approve
file edits. If you see JSON events with `"type":"tool_use"` the harness is working.

## Gemini CLI (`gemini`)

The Gemini CLI is a Node.js package distributed via npm. It is also installed by
`@google-labs/gemini-cli`.

```bash
npm install -g @google/generative-ai-cli
# OR: npm install -g @google-labs/gemini-cli
export GEMINI_API_KEY=AIza...
gemini --version
```

Verify:

```bash
echo "Write hello world to solution.py" | gemini -p - --yolo --output-format stream-json
```

The `--yolo` flag auto-approves all tool calls. Events with `"type":"tool_result"` confirm
agentic file operations are working.

> **Note:** The Gemini CLI binary may be installed under an NVM node version. If
> `gemini` is not in PATH, add your NVM node bin dir:
> `export PATH="$HOME/.nvm/versions/node/$(node --version)/bin:$PATH"`

## OpenAI Codex CLI (`codex`)

```bash
npm install -g @openai/codex
export OPENAI_API_KEY=sk-...
codex --version
```

The executor uses `codex exec --json --model <model> --dangerously-bypass-approvals-and-sandbox`.

Verify:

```bash
echo "Write hello world to solution.py" | codex exec --json \
  --model gpt-5.4 --dangerously-bypass-approvals-and-sandbox
```

You should see NDJSON events including `thread.started`, `turn.started`,
`item.completed` with `type: "file_change"`, and `turn.completed` with usage stats.

> **Note:** Codex CLI v0.1+ uses the thread/item event format. Older versions
> used a flat message/tool_use format. The AILANG executor handles both.

## opencode (`opencode`)

opencode is a multi-provider gateway that supports Anthropic, OpenAI, Google Vertex,
and local Ollama models through a single CLI.

```bash
npm install -g opencode-ai
opencode --version   # e.g. 1.14.20
```

### Provider Authentication

Each provider opencode talks to needs credentials:

| Provider | Setup |
|----------|-------|
| Anthropic | `export ANTHROPIC_API_KEY=sk-ant-...` |
| OpenAI | `export OPENAI_API_KEY=sk-...` |
| Google Vertex | `gcloud auth application-default login` |
| Ollama (local) | `ollama serve` running; no key needed |

### Model String Format

opencode uses `provider/model` strings — **not bare model names**:

```
anthropic/claude-haiku-4-5                 # Anthropic
openai/gpt-5.4                             # OpenAI
google-vertex/gemini-3-flash-preview       # Google Vertex AI
ollama/gemma4:latest                       # Local Ollama
```

> **Important:** Google models require the `google-vertex/` prefix.
> `google/` is not a registered provider and causes `ProviderModelNotFoundError`.
> Run `opencode models google-vertex` to list available model IDs.

To discover all available providers and models:

```bash
opencode models              # all providers
opencode models anthropic    # Anthropic models only
opencode models google-vertex  # Google Vertex models
```

### Verify opencode Works

```bash
cd /tmp && mkdir oc_test && cd oc_test
echo "Write hello world to solution.py" | opencode run \
  --format json --dangerously-skip-permissions \
  --model anthropic/claude-haiku-4-5
```

You should see NDJSON events with `"type":"tool_use"` for file writes.

### Local Models via Ollama

opencode can route to local Ollama models with a custom provider config at
`~/.config/opencode/opencode.jsonc`:

```jsonc
{
  "provider": {
    "ollama": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Ollama Local",
      "options": { "baseURL": "http://localhost:11434/v1" },
      "models": {
        "gemma4:latest":   { "name": "Gemma 4" },
        "gemma3:4b":       { "name": "Gemma 3 4B" }
      }
    }
  }
}
```

Then use `ollama/gemma4:latest` as the model string, or add an entry to `models.yml`
pointing at `opencode-gemma4` with `agent_cli: "opencode"` and
`agent_model_name: "ollama/gemma4:latest"`.

See `internal/executor/opencode/testdata/opencode_ollama_config.jsonc` for a
complete config example.

## Running a Cross-Harness Smoke Eval

Once all harnesses are installed and authenticated, run the cross-harness comparison:

```bash
# Dry run to confirm 5 models × 3 benchmarks × 2 languages = 30 runs
ailang eval-suite --agent --models harness_suite \
  --benchmarks fizzbuzz,gcd_lcm,balanced_parens \
  --langs ailang,python --dry-run

# Full run (5 parallel agent sessions)
ailang eval-suite --agent --models harness_suite \
  --benchmarks fizzbuzz,gcd_lcm,balanced_parens \
  --langs ailang,python --agent-parallel 5
```

`harness_suite` expands to:
- `claude-sonnet-4-6` → claude harness
- `opencode-sonnet-4-6` → opencode harness (Anthropic backend)
- `gemini-3-flash` → gemini harness
- `opencode-gemini-3-flash` → opencode harness (Google Vertex backend)
- `gpt5-4` → codex harness

This gives Δ delta comparison between same-model, different-harness pairs (Sonnet via
claude vs opencode; Flash via gemini vs opencode). Results appear in
`/docs/benchmarks/by-harness` once `ailang eval-report --format=json` is re-run.

## Troubleshooting

**"non-agentic result: 0 turns, 0 tool calls"**

The executor ran but the agent produced 0 tool calls — it either:
- Printed an answer directly instead of writing a file (0-shot behavior)
- Failed to auth (no key / expired token) and exited immediately
- Used the wrong model string (`google/` instead of `google-vertex/` for opencode)

Run the verify command for that harness above and check the raw event output.

**Codex: "openai: 401 Unauthorized"**

`OPENAI_API_KEY` is not set or expired. Check `echo $OPENAI_API_KEY`.

**Gemini: binary not in PATH**

NVM issue. Add the node bin dir to PATH: see the Gemini section above.

**opencode-gemini: "ProviderModelNotFoundError"**

You're using `google/...` instead of `google-vertex/...`. Check `agent_model_name`
in `models.yml`.
