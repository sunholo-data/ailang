# M-MOTOKO-LOCAL-OLLAMA: Motoko Agent Loop Against Local Ollama on the Eval Rig

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 — unlocks zero-cost agent evals on the M4 Max rig; currently all motoko runs burn OpenRouter credits
**Estimated**: 0.5 day (~20 lines of JSON + 5 lines in models.yml)
**Dependencies**: M-EVAL-LOCAL-OLLAMA (local Ollama rig operational), Qwen 3.5 35B-A3B validated at 17/17 on tier:smoke (commit `f842e842`)

## Problem Statement

Every motoko eval run today routes through OpenRouter (profile `dogfood` → `openrouter/...`). The local Ollama rig already runs Qwen 3.5 35B-A3B at mxfp8 and has proven it can drive the opencode harness — but motoko has never been wired to it.

The gap is purely configuration:

1. No `ollama` profile in `motoko_agent/.motoko/config/ollama/config.json`
2. No `motoko-local-qwen3-5-35b-a3b-mxfp8` entry in `internal/eval_harness/models.yml`

The executor side is already correct: `internal/executor/motoko/motoko.go:201` passes `MOTOKO_CONFIG=<profile>` as an env var, and `internal/executor/factory.go:49` exposes `MotokoProfile` (default `"dogfood"`) as a per-entry config field.

The `local` profile at `.motoko/config/local/config.json` proves the concept: it points `openai_base_url` at a remote vLLM endpoint (`http://100.79.48.75:8000/v1`) and sets `ai_options_json` for thinking-budget. Ollama is the same pattern pointed at `localhost:11434/v1`.

## Goals

1. Add a `ollama` profile to `motoko_agent/.motoko/config/ollama/config.json` that routes to the local Ollama OpenAI-compat API.
2. Add a `motoko-local-qwen3-5-35b-a3b-mxfp8` entry to `models.yml` with `agent_cli: motoko` and `motoko_profile: ollama`.
3. Run one smoke-tier eval to confirm motoko converges correctly against Ollama's tool-call surface.

## Implementation

### 1. New motoko config profile: `ollama`

File: `/Users/voightkampff/dev/arniwesth/motoko_agent/.motoko/config/ollama/config.json`

```json
{
  "agent": {
    "model": "ollama/qwen3.5:35b-a3b-mxfp8",
    "workdir": ".",
    "max_steps": 50,
    "step_delay_ms": 0,
    "max_retries": 3,
    "retry_base_ms": 1000,
    "retry_cap_ms": 30000,
    "semi_formal_verifier_mode": false,
    "system_prompt": "",
    "openai_base_url": "http://localhost:11434/v1",
    "ai_options_json": "{\"enable_thinking\":false}"
  },
  "backend": {
    "mode": "external_http",
    "url": "http://127.0.0.1:8080",
    "port": 8080,
    "auto_start": true,
    "command": "bun",
    "args": ["src/tui/src/env-server-main.ts"],
    "startup_timeout_ms": 5000
  },
  "tools": {
    "hybrid": true,
    "ohmy_pi": false,
    "snippet_caps": ["IO", "FS", "Process"],
    "delegated_timeout_ms": 30000,
    "delegated_poll_ms": 100,
    "delegated_timeout_slack_ms": 5000,
    "edit_mode": "hashline"
  },
  "extensions": {
    "order": ["compaction_ai", "context_mode"],
    "strict": false
  },
  "verification": {}
}
```

Key differences from `local` profile:
- `openai_base_url`: `localhost:11434/v1` (Ollama) vs `100.79.48.75:8000/v1` (remote vLLM)
- `ai_options_json`: `enable_thinking: false` — Ollama's OpenAI-compat layer does not handle vLLM-style `chat_template_kwargs`; sending thinking params causes a 400
- `extensions.order`: drops `exa_search` and `omnigraph` — these make outbound HTTP calls that inflate eval cost and add non-determinism; local evals should be self-contained
- `model`: `ollama/qwen3.5:35b-a3b-mxfp8` — motoko's model-name prefix routing; Ollama strips the `ollama/` prefix internally

### 2. New models.yml entry

Add after the `motoko-gemma-4` block in `internal/eval_harness/models.yml`:

```yaml
  motoko-local-qwen3-5-35b-a3b-mxfp8:
    api_name: "qwen3.5:35b-a3b-mxfp8"
    provider: "ollama"
    description: "Qwen 3.5 35B-A3B (mxfp8) via motoko_agent on local Ollama — zero-cost agent eval on M4 Max rig"
    env_var: null                                      # No API key needed — local inference
    agent_cli: "motoko"
    motoko_profile: "ollama"                           # Routes to .motoko/config/ollama/config.json
    agent_model_name: "ollama/qwen3.5:35b-a3b-mxfp8"
    model_family: "qwen3.5-35b"
    max_output_tokens: 32768
    pricing:
      input_per_1k: 0.0
      output_per_1k: 0.0
    notes: |
      Qwen 3.5 35B-A3B (mxfp8) via motoko_agent on the local Ollama rig.
      Pairs with opencode-local-qwen3-5 and bare qwen3-5-35b-a3b-mxfp8 to
      isolate the motoko harness lift at zero marginal cost.
      Requires: Ollama running on localhost:11434, model pulled, motoko CLI.
      No OPENROUTER_API_KEY needed.
```

### 3. Executor wiring (already correct — no changes needed)

`internal/executor/motoko/motoko.go:201` already passes `MOTOKO_CONFIG=<profile>` to the motoko subprocess. `internal/executor/factory.go:49` already reads `MotokoProfile` from the models.yml entry. The new field name `motoko_profile` in models.yml must match the struct field name — verify at compile time via `make build`.

## Key Risk: Tool-Call Compatibility

**Risk**: Motoko's agent loop relies on structured tool calls (file edit, run, verify). Ollama's OpenAI-compat layer (`/v1/chat/completions`) passes `tools` and `tool_choice` through to the underlying model, but the model's tool-call compliance depends on the Modelfile's chat template.

**What could go wrong**:
- Qwen 3.5's Ollama Modelfile may not set the correct chat template for structured tool calling (vs the vLLM-served version which sets `chat_template_kwargs` explicitly)
- Ollama may silently reformat tool-call results in a way that motoko's AILANG parser doesn't expect
- Thinking tokens (`<think>...</think>` blocks) may bleed into tool-call JSON if `enable_thinking` is not fully suppressed

**Mitigation**:
- `"enable_thinking": false` in `ai_options_json` — eliminates thinking-token bleed
- Validate via `ailang eval-suite --models motoko-local-qwen3-5-35b-a3b-mxfp8 --tier smoke` before adding to nightly rotation
- If tool-call failures are systematic, the fallback is `motoko`'s `edit_mode: hashline` — this uses line-addressed diffs rather than structured JSON edits, which degrades gracefully when the model can't produce valid JSON tool calls
- If smoke fails entirely, compare raw Ollama tool-call output against the opencode run for the same benchmark (motoko logs per-step in session JSONL)

**Compatibility data point**: the `local` profile (remote vLLM) works with `enable_thinking: true` + `thinking_token_budget: 256`. Ollama on the same model needs `enable_thinking: false` because the thinking-budget parameter is a vLLM-specific extension not forwarded by Ollama's proxy layer.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `enable_thinking: false` vs `true` | vLLM-style `chat_template_kwargs` silently no-ops in Ollama, risks thinking-token JSON bleed | agent | design | low |
| Drop `exa_search` + `omnigraph` extensions | Both make outbound HTTP; eval should be hermetic on the rig | agent | design | low |
| `env_var: null` for local model | No API key wiring needed; executor must handle null without error | agent | compile | low |

### Design Freeze

Before implementation begins, no open questions — the design is deterministic:
- [ ] Confirm `motoko_profile` YAML key name matches `MotokoProfile` struct field in factory.go (camelCase vs snake_case — check YAML tags)
- [ ] Confirm `env_var: null` doesn't break the executor's key-lookup path (it shouldn't — opencode/ollama entries already use `null`)

## Acceptance Criteria

1. `ailang eval-suite --models motoko-local-qwen3-5-35b-a3b-mxfp8 --tier smoke` completes without executor-level errors
2. ≥10/17 smoke benchmarks pass (lower bar than opencode because motoko hasn't been tuned for local inference)
3. No `OPENROUTER_API_KEY` required to run
4. Session JSONL is emitted (confirms motoko started and completed its agent loop, even if some benchmarks fail)

## Out of Scope

- Tuning motoko's system prompt or verifier config for local inference — that's a follow-on sprint
- Adding this model to the nightly rotation — gated on smoke passing first
- Testing other Ollama models through motoko — the profile is reusable; add entries in a follow-on
