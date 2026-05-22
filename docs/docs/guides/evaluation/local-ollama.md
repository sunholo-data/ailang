---
sidebar_position: 9
title: Local Ollama Eval
---

# Local Ollama Eval

Running AILANG agent-mode evaluations against local Ollama models — typically
**gemma4:26b** — on a dedicated Mac Studio (128 GB unified memory, M4 Max).
Companion to [model-configuration.md](./model-configuration.md) which covers
cloud OS models routed via OpenRouter.

This page reflects the M-EVAL-LOCAL-OLLAMA + M-EVAL-LOCAL-OBSERVABILITY
milestones (v0.22.0).

## TL;DR — canonical commands

After one-time setup (below), the rotation runs via:

```bash
# Smoke tier (17 benchmarks, ~80 min wall, requires ~75 GB memory headroom)
make eval-smoke \
  MODELS=opencode-gemma4-26b \
  EXTRA='-agent -langs ailang \
    -benchmarks fizzbuzz,adt_option,balanced_parens,binary_tree_sum,canonical_convergence,canonical_normalization,dense_operator_program,explicit_state_threading,gcd_lcm,immutable_data_structures,inline_tests,nested_records,numeric_modulo,record_update,records_book,recursion_fibonacci,type_safe_record_access \
    -output eval_results/rotation/$(date +%Y-%m-%d)/$(date +%H%M)_gemma4-26b_smoke \
    -parallel 4 \
    -agent-timeout 2400'

# Watch progress live
ailang chains live $(ailang chains list --limit 1 --since 5m | tail -1 | awk '{print $1}')
```

## One-time setup

### 1. Install prerequisites

```bash
# Go (for building ailang)
brew install go

# node + npm (for opencode and pi CLIs)
brew install node

# Ollama (the model runtime)
brew install ollama
ollama serve &  # or start the app from Applications

# opencode CLI (agent-mode harness used by AILANG)
npm install -g opencode-ai
opencode --version  # confirm 1.15.7 or newer

# Build and install ailang itself
cd $REPO
make install
```

### 2. Pull the model

```bash
ollama pull gemma4:26b
ollama show gemma4:26b
```

Expected: 25.8B params, Q4_K_M, 17 GB on disk, 25.76 GB resident VRAM, 262k context.

### 3. Configure opencode's Ollama provider

`~/.config/opencode/opencode.jsonc`:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "ollama": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Ollama Local",
      "options": { "baseURL": "http://localhost:11434/v1" },
      "models": {
        "gemma4:26b": { "name": "Gemma 4 26B (local)" }
      }
    }
  }
}
```

Verify with `opencode models | grep ollama` — should print `ollama/gemma4:26b`.

### 4. Configure Ollama parallelism (recommended)

```bash
# Set in your shell init or via launchctl setenv if running Ollama.app
launchctl setenv OLLAMA_MAX_LOADED_MODELS 1   # one model resident at a time
launchctl setenv OLLAMA_NUM_PARALLEL 4        # 4 concurrent requests/model
launchctl setenv OLLAMA_MAX_QUEUE 64          # back-pressure threshold
```

Restart Ollama for these to take effect.

### 5. Start the AILANG observability server

```bash
# Manual (for development)
make services-start
curl -s http://localhost:1957/health

# Persistent (for 24/7 rotation)
cp tools/launchd/dev.ailang.server.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/dev.ailang.server.plist
launchctl list | grep dev.ailang.server   # confirm it's loaded
```

### 6. Enable OTLP export from eval-suite

Add to your shell init (`.zshrc` / `.bashrc`):

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957
```

This routes per-step OTEL spans from each opencode subprocess into the local
observatory.db (`~/.ailang/state/observatory.db`). Required for
`ailang chains live` to show per-stage progress. **Without this set, eval
runs still complete but you get no live monitoring.**

## How parallelism behaves on M4 Max

128 GB unified memory + 40 GPU cores; gemma4:26b uses 25.76 GB of unified
memory. Empirical measurements (2026-05-22):

| `-parallel N` | Per-benchmark wall | Aggregate (4 benchmarks) | Note |
|---|---|---|---|
| 1 (serial) | 4–7 min (variance dominates) | ~22 min (extrapolated) | Baseline |
| 2 | ~7 min | ~7m30s | Best wall-clock, sometimes thrashes a benchmark to ~7K tokens |
| 4 | ~15 min | ~24m39s | Slower per-bench but **stabler** — token rate throttled to ~110 tok/s prevents thrash |
| 8+ | not tested | (queues in Ollama) | Won't beat 4 unless `OLLAMA_NUM_PARALLEL` is raised |

`-parallel 4` is the **recommended default** for the 24/7 rotation. The
per-benchmark slowdown is more than offset by improved stability.

## Per-model config in `models.yml`

The relevant entry (already in `internal/eval_harness/models.yml`):

```yaml
opencode-gemma4-26b:
  api_name: "gemma4:26b"
  provider: "ollama"
  agent_cli: "opencode"
  agent_model_name: "ollama/gemma4:26b"
  max_output_tokens: 8192
  ttft_timeout: 900            # 15 min — local thinking + p=4 contention
  generation_timeout: 1200     # 20 min — opencode per-session hard cap
  budgets:
    hard_timeout_secs: 2400    # 40 min — wall-clock safety net
  pricing:
    input_per_1k: 0.0
    output_per_1k: 0.0
```

Two design choices worth knowing:

1. **`pricing: 0` means cost gate is unused** — wall-clock is the only cap.
2. **`budgets.hard_timeout_secs` wins over benchmark-spec `timeout:` fields**
   (M-EVAL-LOCAL-OLLAMA precedence fix). Local thinking models can iterate
   long even on benchmarks that have a cloud-tuned `timeout: 90s`.

## Live monitoring

While a run is in flight:

```bash
# Find the active chain (most recent)
ailang chains list --since 5m --limit 1

# Live view with 3-second refresh (default)
ailang chains live <chain-id>

# Faster refresh
ailang chains live <chain-id> --interval 1

# Single render then exit (useful in scripts)
ailang chains live <chain-id> --once
```

Output:

```
Chain: c68f0cc6  Source: eval_suite  Status: active  Elapsed: 12m
Ollama: gemma4:26b  (VRAM 25.76 GB)
────────────────────────────────────────────────────────────────────────────────
#    Benchmark / Agent           Status     Turns   Tokens    Last span
────────────────────────────────────────────────────────────────────────────────
1    eval-agent:fizzbuzz         running    12      47K       3s ago
2    eval-agent:adt_option       running    8       31K       12s ago
3    eval-agent:balanced_parens  running    0       0         ⚠ 540s ago (stuck?)
4    eval-agent:recursion_fib    running    14      52K       1s ago
────────────────────────────────────────────────────────────────────────────────
```

The `⚠ stuck?` indicator fires when the most recent span for a stage is
>300s old AND status is still `running`. Local thinking models can spend
minutes in pure reasoning before emitting visible output — see if `Ollama:`
header still shows the model with non-zero VRAM and check `ollama runner`
CPU% (`ps aux | grep "ollama runner"`). If runner CPU is >20%, the model is
generating; if <1%, it really is stuck.

## Troubleshooting

**Symptom: `observatory.db` stays at 0 spans**

- Verify `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957` is set in the
  shell where you launched eval-suite (not just in your config). Confirm with
  `ps eww -p <pid>` listing the var.
- Verify server is up: `curl -s http://localhost:1957/health`.
- Check `~/.ailang/logs/server.log` for `FOREIGN KEY constraint failed`
  (should be absent post-M-EVAL-LOCAL-OBSERVABILITY M1).

**Symptom: benchmarks fail with "opencode produced no output within 8m0s
(prefill timeout)"**

- The `ttft_timeout` is too tight for your `-parallel N` level. At p=4 we
  observed prefill needing ~12 min for some benchmarks. Bump
  `opencode-gemma4-26b.ttft_timeout` in models.yml.

**Symptom: high token thrashing (>1M tokens for a simple benchmark)**

- Try raising `-parallel` from 2 to 4. The reduced per-request token rate
  acts as a "think before emitting" governor that suppresses runaway loops.
- This is expected behavior for some benchmarks (e.g. `dense_operator_program`
  consistently thrashes regardless of config — a real model gap).

**Symptom: "non-agentic result: 1 turns, 0 tool calls"**

- The model decided to one-shot the answer instead of using tools. opencode
  rejects this as not-agentic. Currently no clean workaround beyond
  re-running. See M-EVAL-LOCAL-OBSERVABILITY notes for "make non-agentic a
  warning not error" deferred work.

**Symptom: `ailang chains live` shows "(no spans yet)" for every stage even
mid-run**

- Spans are landing but not joined to stages. This is a known follow-up:
  per-stage `chain_id`/`stage_id` resource attrs need to be added at the
  eval-suite OTLP-resource layer. Use `ailang chains diagnose` or
  `sqlite3 ~/.ailang/state/observatory.db 'SELECT COUNT(*) FROM spans'` to
  confirm spans are still flowing.

## Related

- [M-EVAL-LOCAL-OLLAMA design doc](/design_docs/planned/m-eval-local-ollama.md)
- [M-EVAL-LOCAL-OBSERVABILITY design doc](/design_docs/planned/v0_22_0/m-eval-local-observability.md)
- [model-configuration.md](./model-configuration.md) — cloud OS models via OpenRouter
- [Ollama library](https://ollama.com/library) — full model catalog
