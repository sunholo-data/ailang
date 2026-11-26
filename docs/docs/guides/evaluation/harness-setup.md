---
title: Agent Harness Setup
description: Install, authenticate, and verify agentic coding harnesses (Claude Code, Vertex Managed Agents, Codex CLI, opencode) for ailang eval-suite --agent
tier: internal
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Agent Harness Setup

AILANG's agent eval mode runs benchmarks through agentic CLI tools and remote sandboxes via `ailang eval-suite --agent`.

## Supported Harnesses

| Harness | Transport | Sample `models.yml` IDs | Install / Auth |
|---|---|---|---|
| **claude** | `claude` CLI | `claude-sonnet-4-6`, `claude-haiku-4-5` | `npm install -g @anthropic-ai/claude-code` + `ANTHROPIC_API_KEY` |
| **managed_agents** | Vertex HTTP API | `gemini-3-5-flash` | `gcloud auth application-default login` (replaces retired `gemini` CLI) |
| **codex** | `codex` CLI | `gpt5-4`, `gpt5-1-instant` | `npm install -g @openai/codex` + `OPENAI_API_KEY` |
| **opencode** | `opencode` CLI | `opencode-haiku`, `opencode-sonnet-4-6`, `opencode-gemini-3-flash` | `npm install -g opencode-ai` + provider creds or Ollama |

```bash
# Quick health check
claude --version
gcloud auth application-default print-access-token   # managed_agents
codex --version
opencode --version
```

## Harness Setup & Verification

<details>
<summary>Claude Code (`claude`) and OpenAI Codex CLI (`codex`)</summary>

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code && export ANTHROPIC_API_KEY=sk-ant-...
echo "Write hello world to solution.py" | claude --print \
  --output-format stream-json --permission-mode bypassPermissions

# Codex CLI (supports both modern thread/item NDJSON and legacy flat format)
npm install -g @openai/codex && export OPENAI_API_KEY=sk-...
echo "Write hello world to solution.py" | codex exec --json \
  --model gpt-5.4 --dangerously-bypass-approvals-and-sandbox
```

</details>

<details>
<summary>Vertex Managed Agents (`managed_agents`) & --clone-repo</summary>

Replaces the retired legacy `gemini` CLI executor via `aiplatform.googleapis.com/v1beta1/.../interactions` (region `global`, pinned `Api-Revision: 2026-05-20`, one sandbox per `Execute()`):

- **File bridge (`executor.CapRemoteSandbox`)**: `eval_harness/managed_agents_bridge.go` captures the final fenced code block from `Result.Output` into `<workspace>/benchmark/solution.ail`.
- **In-sandbox repo verification (`--clone-repo`)**: Opt-in shallow clone (`--depth 1`) gated by `executor.CapNetworkEgress` for **public** repos only. Requires `git rev-parse HEAD` match (or stamps `verification_degraded: true`); errors if combined with `--api-only` or non-Gemini providers.

```bash
gcloud auth application-default login
gcloud config set project ailang-dev

# Verify HEAD or pinned-SHA review
ailang exec gemini "Review the repo for correctness" \
  --clone-repo https://github.com/sunholo-data/ailang.git
ailang exec gemini "Review this revision" \
  --clone-repo https://github.com/sunholo-data/ailang.git \
  --clone-sha 806b3b4a4c0000000000000000000000000000ab
```

</details>

<details>
<summary>opencode (`opencode`) provider strings & local Ollama</summary>

`opencode` requires `provider/model` IDs (use `google-vertex/...`, never `google/...` which triggers `ProviderModelNotFoundError`):

- `anthropic/claude-haiku-4-5`, `openai/gpt-5.4`, `google-vertex/gemini-3-flash-preview`, `ollama/gemma4:latest`

Configure local Ollama in `~/.config/opencode/opencode.jsonc` (`baseURL: "http://localhost:11434/v1"`; see `internal/executor/opencode/testdata/opencode_ollama_config.jsonc`):

```bash
opencode models google-vertex
cd /tmp && mkdir -p oc_test && cd oc_test
echo "Write hello world to solution.py" | opencode run \
  --format json --dangerously-skip-permissions --model anthropic/claude-haiku-4-5
```

</details>

## Running a Cross-Harness Smoke Eval

```bash
ailang eval-suite --agent --models harness_suite \
  --benchmarks fizzbuzz,gcd_lcm,balanced_parens --langs ailang,python --dry-run

ailang eval-suite --agent --models harness_suite \
  --benchmarks fizzbuzz,gcd_lcm,balanced_parens --langs ailang,python --agent-parallel 5
```

`harness_suite` pairs `claude-sonnet-4-6`, `opencode-sonnet-4-6`, `opencode-gemini-3-flash`, and `gpt5-4`; run `gemini-3-5-flash` opt-in via `managed_agents`. View results at `/docs/benchmarks/by-harness` after `ailang eval-report --format=json`.

## Benchmark Semantics & Troubleshooting

| Topic | Behavior / Fix |
|---|---|
| **`grade_entrypoint` (agent-only)** | Auto-skipped by standard-mode `eval-suite`; forced runs short-circuit with `error_category: skipped_mode_incompatible` |
| **`hidden_tests_dir` & `exec_timeout`** | Runs held-out `<name>.in`/`.ans` smallest-first without leaking to prompts (`hidden_total`, `hidden_passed`, `hidden_first_fail`). Set `exec_timeout: 30` to override default 10s AILANG vs 30s Python limits; avoid `data/` or `.out` (`.gitignore`d). See [M-EVAL-HIDDEN-TESTS](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_55_0/m-eval-hidden-tests.md) |
| **0 turns, 0 tool calls** | Agent printed text instead of editing, failed auth (`OPENAI_API_KEY` / ADC), or used wrong prefix (`google/` instead of `google-vertex/`) |
