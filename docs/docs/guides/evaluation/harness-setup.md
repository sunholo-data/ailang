# Agent Harness Setup

AILANG's agent eval mode runs benchmarks through agentic CLI tools — the same tools
developers use interactively. This guide covers how to install and authenticate each
supported harness for `ailang eval-suite --agent`.

## Supported Harnesses

| Harness | CLI tool | Models in `models.yml` | Install |
|---------|----------|------------------------|---------|
| **claude** | Claude Code (`claude`) | `claude-sonnet-4-6`, `claude-haiku-4-5` | `npm install -g @anthropic-ai/claude-code` |
| **managed_agents** | (HTTP API, no CLI) | `gemini-3-5-flash` | `gcloud auth application-default login` |
| **codex** | OpenAI Codex CLI (`codex`) | `gpt5-4`, `gpt5-1-instant` | `npm install -g @openai/codex` |
| **opencode** | opencode (`opencode`) | `opencode-haiku`, `opencode-sonnet-4-6`, `opencode-gemini-3-flash` | `npm install -g opencode-ai` |

> **Note (v0.22.0, M-MANAGED-AGENTS):** The legacy `gemini` CLI executor was
> retired. Google shut off Gemini CLI for free/Pro/Ultra tiers on 2026-06-18.
> It was replaced by the `managed_agents` executor, which calls the Vertex AI
> Managed Agents API (the Antigravity `antigravity-preview` agent) directly via
> ADC. Older Gemini models (2.5, 3, 3.1) lose agent-mode coverage but keep
> standard-mode via direct Vertex `generateContent`.

## Quick Check

```bash
claude --version
gcloud auth application-default print-access-token   # for managed_agents
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

## Managed Agents API (`managed_agents`)

The Managed Agents executor calls the Vertex AI Managed Agents endpoint
(`aiplatform.googleapis.com/v1beta1/.../interactions`) directly via HTTP using
Application Default Credentials. There is no local CLI — the agent runs in a
Google-hosted Linux sandbox per interaction, with full tool execution + multi-turn
state managed server-side.

### Setup

```bash
# 1. Authenticate via ADC (same flow as direct Vertex generateContent calls)
gcloud auth application-default login

# 2. Set the default project (or set Task.GCPProject per-call via models.yml)
gcloud config set project ailang-dev

# 3. First call to a fresh project provisions the service (HTTP 400
#    "Provisioning is in progress" for ~3 min, then ready). The executor's
#    error message includes this hint.
```

### Verify

```bash
ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
curl -sN -X POST \
  "https://aiplatform.googleapis.com/v1beta1/projects/ailang-dev/locations/global/interactions" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Api-Revision: 2026-05-20" \
  -d '{
    "stream": true, "background": true, "store": true,
    "agent": "antigravity-preview-05-2026",
    "environment": {"type": "remote"},
    "input": [{"type":"user_input","content":[{"type":"text","text":"reply PONG"}]}]
  }'
```

Successful output is an SSE stream ending with
`event: interaction.completed` followed by `data: [DONE]`.

### Cross-environment file bridge

Because the agent runs in a remote sandbox, file edits the agent makes do NOT
touch the local workspace. The eval harness handles this by:

1. Appending an instruction to the system prompt that tells the agent to dump
   its complete solution as a fenced code block at the end of its response
2. Extracting that fenced block from `Result.Output` and writing it to
   `<workspace>/benchmark/solution.ail` after the run

This is automatic — handled by `eval_harness/managed_agents_bridge.go` for any
executor that advertises `executor.CapRemoteSandbox`. Other backend callers
that don't need file bridging (e.g. plain reasoning queries) get a
policy-free executor.

### Clone-over-egress: in-sandbox verification (`--clone-repo`)

By default the sandbox has no network egress and no local repo, so a Gemini
review can only *reason* about a prompt-packed diff. With **opt-in egress** the
hosted agent can instead `git clone` the **public** AILANG repo at a target
revision and run review / `ailang check` **in-sandbox**, returning a structured
verdict — real verification, not reasoning-only.

Egress is strictly opt-in and visible in the request payload. It is gated by the
`executor.CapNetworkEgress` capability (only `managed_agents` advertises it):
setting `Task.RequiresEgress` on any other executor is a **loud pre-dispatch
error**, never a silent fallback.

```bash
# HEAD review — shallow clone of the repo's default branch
ailang exec gemini "Review the repo for correctness" \
  --clone-repo https://github.com/sunholo-data/ailang.git

# Pinned-SHA review — shallow fetch-by-SHA of exactly one commit
ailang exec gemini "Review this revision" \
  --clone-repo https://github.com/sunholo-data/ailang.git \
  --clone-sha 806b3b4a4c0000000000000000000000000000ab
```

- Both modes are **shallow** (`--depth 1`) and bounded by construction — the HEAD
  path clones HEAD; the pinned path does `git fetch --depth 1 origin <sha>` (no
  full-history walk).
- `--clone-repo` on any provider other than `gemini`, or together with
  `--api-only` (which has no sandbox), or `--clone-sha` without `--clone-repo`,
  **exits non-zero with a clear error**.
- The agent must echo `git rev-parse HEAD`. The eval bridge
  (`EvalOptions.CloneRepoURL` / `CloneSHA` in `gemini_evaluator_bridge.go`)
  verifies it: a pinned review must match the requested SHA; a HEAD review must
  produce a valid 40-hex SHA (recorded as the reviewed revision). Missing or
  mismatched evidence — or a deadline-exceeded run — stamps
  `verification_degraded: true` with a reason. Absent evidence is **never** a
  clean pass.
- **Security:** `network.allowlist` is wildcard (`{"domain":"*"}`) — the only
  shape Vertex accepts today. It is scoped to a **read-only reviewer cloning a
  PUBLIC repo**, so no secret/PAT/credential ever enters the sandbox. The
  private-repo path is out of scope.

### Limits

- **No multi-turn yet.** Each Execute() provisions a fresh sandbox.
- **Region locked to `global`.** No regional Vertex endpoints for this API yet.
- **`Api-Revision: 2026-05-20` header pinned** in the executor — guards
  against schema drift. Bump when Google publishes a new revision.
- **Cost: $1.50/$9.00 per 1M** (Vertex gemini-3.5-flash pricing).

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
- `opencode-gemini-3-flash` → opencode harness (Google Vertex backend)
- `gpt5-4` → codex harness

> Gemini-family models have no CLI-subprocess harness since v0.22.0. For Google
> agent data, run `gemini-3-5-flash` opt-in via the `managed_agents` executor
> (see above); the Vertex backend is also reachable through opencode
> (`opencode-gemini-3-flash`).

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

**opencode-gemini: "ProviderModelNotFoundError"**

You're using `google/...` instead of `google-vertex/...`. Check `agent_model_name`
in `models.yml`.
