---
name: local-ollama-eval
description: Run AILANG agent-mode evaluations against local Ollama models on a dedicated Mac Studio rig. Use when user mentions "local ollama", "gemma4", "run eval locally", "smoke tier on the rig", "ollama benchmark", or wants to start/monitor a local-model eval rotation. Covers the M-EVAL-LOCAL-OLLAMA + M-EVAL-LOCAL-OBSERVABILITY workflow.
---

# Local Ollama Eval

Run AILANG agent-mode evals against **local Ollama models** (typically `gemma4:26b`) on a dedicated Apple Silicon rig with 64+ GB unified memory. This is the canonical workflow for cost-free continuous OS-model evaluation, replacing pay-per-token OpenRouter routing.

Born from the M-EVAL-LOCAL-OLLAMA + M-EVAL-LOCAL-OBSERVABILITY milestones (v0.22.0). The user guide at [docs/docs/guides/evaluation/local-ollama.md](../../../docs/docs/guides/evaluation/local-ollama.md) is the human-facing reference; this skill is the Claude-facing operational checklist.

## Current State

- **ollama serve**: !'curl -s http://localhost:11434/api/ps >/dev/null 2>&1 && echo "✓ running" || echo "✗ NOT running"'
- **ailang server (OTLP receiver)**: !'curl -s http://localhost:1957/health 2>/dev/null | grep -q healthy && echo "✓ running" || echo "✗ NOT running"'
- **Loaded ollama models**: !'curl -s http://localhost:11434/api/ps 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); [print(m[\"name\"], \"vram=%.1fGB\"%(m.get(\"size_vram\",0)/1e9)) for m in d.get(\"models\",[])]" 2>/dev/null || echo "(none)"'
- **OTEL endpoint env**: !'echo "${OTEL_EXPORTER_OTLP_ENDPOINT:-(unset — eval-suite will write to whatever DEFAULT it picks)}"'
- **Recent chains (last hour)**: !'PATH="$HOME/go/bin:$PATH" ailang chains list --since 1h --limit 3 2>/dev/null | tail -3 || echo "(none)"'
- **Spans in observatory.db**: !'sqlite3 ~/.ailang/state/observatory.db "SELECT COUNT(*) FROM spans;" 2>/dev/null || echo "(db missing or empty)"'

> **Use the state above to skip work that is already done.** If ollama isn't running, start it before doing anything else. If the ailang server isn't running, `make services-start`. If the env var is unset, you'll lose live span observability.

## Quick Start

### Run the canonical smoke tier

```bash
# Activate observability (one-shot per shell session)
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957

# Verify rig is ready
.claude/skills/local-ollama-eval/scripts/verify_setup.sh

# Run smoke tier on gemma4:26b at p=2 (recommended default)
.claude/skills/local-ollama-eval/scripts/run_smoke.sh opencode-gemma4-26b

# Watch progress in another terminal/turn
.claude/skills/local-ollama-eval/scripts/watch_active.sh
```

### Single-benchmark test

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957
make eval-smoke MODELS=opencode-gemma4-26b \
  EXTRA='-agent -langs ailang -benchmarks fizzbuzz \
    -output /tmp/quick_check -parallel 1 -agent-timeout 1800'
```

Fizzbuzz solo takes 1–4 min wall clock and tells you the stack works end-to-end.

## When to Use This Skill

Invoke when the user says any of:
- "Run eval locally" / "run on the rig"
- "Try gemma4 on the smoke tier"
- "Start the rotation"
- "Watch the current eval"
- "Why does the local benchmark X fail?"
- "Add Qwen3 to the local rotation"
- Anything mentioning `ollama-*` or `opencode-gemma4-*` model names

Do NOT use this skill when the user is talking about cloud OS models (those route via OpenRouter — see [model-manager](../model-manager/SKILL.md)) or about cloud-only Claude/GPT/Gemini evaluation.

## Setup Checklist (one-time, per rig)

Run [`scripts/verify_setup.sh`](scripts/verify_setup.sh) — it checks all of these:

| Component | How to install | Verify |
|---|---|---|
| Go toolchain | `brew install go` | `go version` |
| Ollama runtime | `brew install ollama` then start app | `curl -s localhost:11434/api/ps` |
| `gemma4:26b` model | `ollama pull gemma4:26b` (~17 GB) | `ollama show gemma4:26b` |
| Node + npm | `brew install node` | `node --version` |
| opencode CLI | `npm install -g opencode-ai` | `opencode --version` |
| opencode Ollama provider | `~/.config/opencode/opencode.jsonc` (see resources) | `opencode models | grep ollama` |
| AILANG binary | `cd ailang && make install` | `which ailang` |
| AILANG server | `make services-start` (or launchd plist for 24/7) | `curl -s localhost:1957/health` |
| Ollama parallelism | `launchctl setenv OLLAMA_NUM_PARALLEL 4` + restart Ollama | env var visible in Ollama logs |
| OTLP export env | `export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957` | check `env \| grep OTEL` |

For 24/7 operation, install the launchd plist:

```bash
cp tools/launchd/dev.ailang.server.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/dev.ailang.server.plist
```

## Canonical Commands

### Smoke tier (17 benchmarks)

```bash
.claude/skills/local-ollama-eval/scripts/run_smoke.sh opencode-gemma4-26b
```

Expected: ~80 minutes wall clock at `-parallel 2`, 12–14/17 pass rate (was 7/17 before the timeout bumps; was thrashing-prone at `-parallel 4`).

### Single benchmark (~1–10 min)

```bash
.claude/skills/local-ollama-eval/scripts/run_smoke.sh opencode-gemma4-26b fizzbuzz
```

Use for: quick sanity check, debugging a specific benchmark, validating env setup.

### Core tier (~20 benchmarks including the harder ones)

```bash
make eval-core MODELS=opencode-gemma4-26b \
  EXTRA='-agent -langs ailang \
    -benchmarks <list-from-benchmarks-dir> \
    -output eval_results/rotation/$(date +%Y-%m-%d)/$(date +%H%M)_gemma4-26b_core \
    -parallel 2 -agent-timeout 2400'
```

## Live Monitoring

Once a run is in flight, **the most useful tool is `ailang chains live`**:

```bash
# Most recent active chain
CHAIN_ID=$(ailang chains list --since 5m --limit 1 | tail -1 | awk '{print $1}')

# Live refreshing view (Ctrl-C to exit)
ailang chains live $CHAIN_ID

# Snapshot (single render then exit — useful in scripts)
ailang chains live $CHAIN_ID --once
```

Output columns:

- **Benchmark / Agent**: shows `eval-agent:<benchmark>` per stage (from M-EVAL-LOCAL-OBSERVABILITY M2)
- **Turns**: agent turn count so far
- **Tokens**: input+output tokens accumulated
- **Last span**: time since most recent OTEL span — your "is the model alive" heartbeat

### Distinguishing "thinking" from "stuck"

| Symptom | Likely state | What to check |
|---|---|---|
| Last span < 30s ago | Actively generating | nothing to do |
| Last span 1–5 min ago | Thinking hard (normal for thinking models) | check `ollama runner` CPU% |
| Last span > 5 min + `⚠ stuck?` indicator | Possibly stuck OR very long single thinking phase | `pgrep -fc 'opencode run'` to confirm subprocess alive |
| Last span > 10 min + 0 turns | Likely TTFT timeout imminent | bump `ttft_timeout` in models.yml or reduce `-parallel` |

The `ailang chains live` view automatically marks stages with `⚠ stuck?` when the last span is >300s old AND status is still `running`.

## Recommended Configuration (verified empirically)

These are the settings that produced the best stability/throughput trade-off on M4 Max + 128 GB during the M-EVAL-LOCAL-OLLAMA investigation:

### `internal/eval_harness/models.yml` per-Ollama-model entry

```yaml
opencode-gemma4-26b:
  api_name: "gemma4:26b"
  provider: "ollama"
  agent_cli: "opencode"
  agent_model_name: "ollama/gemma4:26b"
  max_output_tokens: 8192
  ttft_timeout: 900            # 15 min — p=2 prefill budget; bump higher for p=4
  generation_timeout: 1200     # 20 min — opencode per-session hard cap
  budgets:
    hard_timeout_secs: 2400    # 40 min — wall-clock safety net (only gate for $0 models)
  pricing:
    input_per_1k: 0.0
    output_per_1k: 0.0
```

### Ollama env vars (set via `launchctl setenv`)

```
OLLAMA_MAX_LOADED_MODELS=1  # one model resident at a time on M4 Max
OLLAMA_NUM_PARALLEL=4       # up to 4 concurrent requests on same model
OLLAMA_MAX_QUEUE=64         # back-pressure when eval queue gets ahead
```

### CLI flags for `make eval-smoke`

| Setting | Recommended | Why |
|---|---|---|
| `-parallel 2` | default | Best stability; p=4 thrashes some benchmarks (token rate too high) |
| `-agent-timeout 2400` | always | Matches `hard_timeout_secs` budget |
| `-langs ailang` | always | Local rig is for OS-model AILANG eval, not multi-lang |
| `-output eval_results/rotation/<date>/<time>_<model>_<tier>/` | recommended | Time-series-queryable structure |

## Interpreting Results

| Outcome | Meaning | Action |
|---|---|---|
| ✅ PASS | Compile + runtime + stdout all correct | Nothing to do |
| ❌ `compile_error` with low tokens (<50K) | Model produced syntactically broken AILANG | Real model gap — note for stdlib/prompt analysis (use `eval-analyzer` skill) |
| ❌ `compile_error` with high tokens (>500K) | Model thrashed — kept emitting broken code | Variance: re-run; if persistent, real gap |
| ❌ `logic_error` | Compiled + ran but wrong output | Real model gap — note for benchmark-spec or prompt-clarity work |
| ❌ `api_error` "ttft timeout" / "idle timeout" | Infrastructure: timeout too tight for current `-parallel` | Bump timeouts in models.yml or reduce `-parallel` |
| ❌ `api_error` "non-agentic result" | Model one-shotted without using tools | Soft fail; re-run usually helps |
| ❌ `timeout` with dur=0s | Hard wall-clock cap hit | Bump `budgets.hard_timeout_secs` |

**Variance warning**: single-trial pass/fail is unreliable for local OS models. The M-EVAL-LOCAL-OLLAMA investigation showed 5 of 17 benchmarks flipped outcome between two consecutive runs of the same model with the same seed. For trustworthy assessment, run N≥3 trials and report median or best-of-N. The `ailang eval-suite` doesn't natively support N-trial mode yet — invoke it N times to separate output directories and compare manually.

## Adding a New Model to the Rotation

```bash
# 1. Check Ollama library has the model
.claude/skills/local-ollama-eval/scripts/check_ollama_model.sh <tag>

# 2. Pull it (warning: multi-GB download)
ollama pull <tag>

# 3. Add to opencode config: edit ~/.config/opencode/opencode.jsonc
#    (See resources/opencode-config-example.jsonc)

# 4. Add models.yml entry — use opencode-gemma4-26b as template,
#    swap api_name, agent_model_name, description.
#    Use the recommended timeouts above (ttft 900, gen 1200, budget 2400).

# 5. Smoke test the new model
.claude/skills/local-ollama-eval/scripts/run_smoke.sh opencode-<new-model> fizzbuzz

# 6. If it passes, run full smoke tier
.claude/skills/local-ollama-eval/scripts/run_smoke.sh opencode-<new-model>
```

See [`resources/candidate_models.md`](resources/candidate_models.md) for the curated rotation candidate list (verified from Ollama library).

## Continuous Rotation (24/7 mode)

Run multiple models per day via launchd plists. Each plist invokes the canonical `make eval-smoke` for a different model at a different time. Template:

```bash
# Copy and customize:
cp tools/launchd/dev.ailang.server.plist \
   ~/Library/LaunchAgents/dev.ailang.eval-<model>-<time>.plist
# Edit:
# - Label: dev.ailang.eval-gemma4-26b-0100
# - StartCalendarInterval: hour/minute when this run kicks off
# - ProgramArguments: replace `ailang serve` with the canonical make command
```

A proper `ailang eval-rotation` daemon (sketch in the M-EVAL-LOCAL-OLLAMA design doc) is a deferred Phase-4 item.

## Resources

- **Candidate models list**: [`resources/candidate_models.md`](resources/candidate_models.md) — what to pull next
- **opencode config sample**: [`resources/opencode_jsonc_example.txt`](resources/opencode_jsonc_example.txt)
- **Troubleshooting**: [`resources/troubleshooting.md`](resources/troubleshooting.md)

## Notes

- This skill is operational, not exploratory. For analysis of WHY a benchmark fails (stdlib gaps, prompt issues), use [`eval-analyzer`](../eval-analyzer/SKILL.md).
- For adding NEW benchmarks (different from adding NEW models), see [`benchmark-manager`](../benchmark-manager/SKILL.md).
- For cross-comparison with cloud OS models (GLM, Kimi, MiniMax via OpenRouter), see [`model-manager`](../model-manager/SKILL.md) — those models are NOT on Ollama.
- Variance is real: do not draw conclusions from single runs. The rotation's value comes from longitudinal trends, not single-day snapshots.
