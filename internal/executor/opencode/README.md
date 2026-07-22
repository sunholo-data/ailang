# opencode Executor

AILANG executor for the [opencode CLI](https://opencode.ai) (`opencode-ai` npm package).

opencode is a multi-provider agentic coding harness that supports Anthropic, OpenAI,
Google, and — crucially — local Ollama models via its custom provider config. This
makes it the gateway for open-source / offline agent eval in AILANG.

## Installation

```bash
npm install -g opencode-ai
opencode --version   # e.g. 1.14.20
```

## Flags Used by the Executor

| Flag | Value | Purpose |
|---|---|---|
| `run` | — | Non-interactive execution subcommand |
| `--format json` | `json` | NDJSON event stream on stdout |
| `--dangerously-skip-permissions` | — | Auto-approve tool permission prompts |
| `--model <provider/model>` | e.g. `anthropic/claude-haiku-4-5` | Model selection |
| `--session <sessionID>` | from prior result | Resume a session (iteration > 1) |
| `--dir <path>` | task workspace | Working directory |

## Model String Format

opencode uses `provider/model` strings:

```
anthropic/claude-haiku-4-5                    # Anthropic via API key
openai/gpt-5                                   # OpenAI via API key
google-vertex/gemini-3-flash-preview           # Google via Vertex AI (ADC)
opencode/claude-opus-4-5                       # opencode Zen proxy
ollama/gemma4:latest                           # Local Ollama model
ollama/llama3.3:latest                         # Local Llama
```

> **Google models require `google-vertex/` prefix**, not `google/`.
> `google/` is not a registered provider in opencode — it will fail with
> `ProviderModelNotFoundError`. Run `opencode models google-vertex` to list
> available Gemini model IDs.

## Ollama / Local Model Setup

opencode supports local models via `~/.config/opencode/opencode.jsonc`:

```jsonc
{
  "provider": {
    "ollama": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Ollama Local",
      "options": { "baseURL": "http://localhost:11434/v1" },
      "models": {
        "gemma4:latest":   { "name": "Gemma 4" },
        "gemma3:4b":       { "name": "Gemma 3 4B" },
        "llama3.3:latest": { "name": "Llama 3.3" }
      }
    }
  }
}
```

Then eval with: `--models opencode-haiku` (or override model to `ollama/gemma4:latest`
in models.yml `agent_model_name`).

See `testdata/opencode_ollama_config.jsonc` for a full example config.

## Auth

| Provider | Env Var / Method |
|---|---|
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Google | ADC (`gcloud auth application-default login`) |
| opencode Zen | `OPENCODE_API_KEY` or `opencode providers login` |
| Ollama | No auth required; `ollama serve` must be running |

## Event Schema

opencode emits NDJSON via `--format json`. Four event types:

### `step_start`
```json
{"type":"step_start","timestamp":1776860240950,"sessionID":"ses_…","part":{"id":"…","messageID":"…","type":"step-start"}}
```

### `text`
```json
{"type":"text","timestamp":…,"sessionID":"…","part":{"type":"text","text":"…","time":{"start":…,"end":…}}}
```

### `tool_use`
```json
{"type":"tool_use","timestamp":…,"sessionID":"…","part":{"type":"tool","tool":"write","callID":"…","state":{"status":"completed","input":{…},"output":"…","title":"…"}}}
```

### `step_finish`
```json
{"type":"step_finish","timestamp":…,"sessionID":"…","part":{"reason":"stop","tokens":{"total":17240,"input":1,"output":26,"reasoning":0,"cache":{"write":17213,"read":0}},"cost":0.02164725}}
```

## Token Semantics

**Per-step deltas** — different from Codex (which emits cumulative running totals).

Each `step_finish` contains the tokens consumed by THAT step only. To get
conversation totals: **sum** all `step_finish.part.tokens.input` and `.output`.
Cost is also per-step; sum `step_finish.part.cost` for total.

The executor's `CostUSD` field reflects the summed per-step cost (reported by
opencode directly, not estimated from token counts × pricing).

### Cache counters are EXCLUSIVE of input

`tokens.cache.write` and `tokens.cache.read` are **not** a subset of
`tokens.input` — unlike OpenAI/codex, where `cached_input_tokens ⊆
input_tokens`. The fixture proves it:

```
total 17240 == input 1 + output 26 + cache.write 17213 + cache.read 0
```

So the four counters sum to `total`, and the executor sums cache into
`Result.CacheCreationInputTokens` (write) and `Result.CacheReadInputTokens`
(read) **additively**. This matters because the eval harness banks agent-mode
input as `InputTokens + CacheCreationInputTokens + CacheReadInputTokens`:
before this was wired up, a run consuming ~52K tokens banked `input_tokens: 5` —
the uncached remainder was all that survived.

## Session Resume

The `sessionID` from any event can be used to resume:
```
opencode run --session <sessionID> "continue with the task"
```

The executor propagates `result.SessionID` for AILANG's feedback-loop
iterations (M-TRANSCRIPT pattern): `task.ResumeSessionID` → `--session` flag.

## Known Limits

- opencode does not support `--skip-git-repo-check` (Codex-specific flag);
  it handles non-git workspaces natively
- `CostModel()` returns zeros: actual cost comes from per-step `step_finish.cost`
  field reported by opencode; pre-flight estimates are provider-dependent
- Session resume (`--session`) depends on opencode's local DB (~/.local/share/opencode/);
  sessions are not portable across machines
- Non-JSON preamble on stdout (e.g. "Performing one time database migration...")
  is skipped cleanly by the parser
- opencode Zen ($200/month subscription) is optional; direct provider API keys work

## Cost Model

Cost is reported directly by opencode in `step_finish.part.cost` (USD). The
executor sums these per-step costs into `Result.CostUSD` rather than estimating
from token counts × pricing rates. This gives higher accuracy than the pricing
table approach used by Claude/Gemini executors.

## μRAG Plugin (microrag-plugin.ts)

The `plugins/` directory contains a TypeScript plugin for opencode's plugin system
that injects AILANG μRAG context on `Edit`, `Write`, `Read`, and `MultiEdit` tool
events — identical semantics to the Claude Code / Gemini CLI / Codex CLI bash shims.

**Install:**
```bash
cp internal/executor/opencode/plugins/microrag-plugin.ts ~/.config/opencode/plugins/
# Add to opencode.jsonc:
# "plugins": ["~/.config/opencode/plugins/microrag-plugin.ts"]
```

**Toggle:**
```bash
export AILANG_MICRORAG_ENABLED=0   # disable
export AILANG_MICRORAG_ENABLED=1   # enable (default)
```

See [plugins/README.md](plugins/README.md) for full install and test instructions.

## Conformance

This package conforms to the uniform CLI-subprocess executor contract defined in
[`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md).

Required symbols: `New()`, `Register()`, `init()`.
Factory name: `"opencode"`.
Coordinator wiring: blank import in `internal/coordinator/provider_executor.go`.
models.yml wiring: `agent_cli: "opencode"` + `agent_model_name: "provider/model"`.
