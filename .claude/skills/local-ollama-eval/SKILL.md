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

## Recommended Configuration (data-driven, 2026-05-23 investigation)

> **Read this section before changing anything.** Prior versions of this skill recommended `OLLAMA_NUM_PARALLEL=4` and `-parallel 2`. That was guess-driven and turned out to be wrong for 100k+ token prompts. The settings below are derived from observed VRAM allocation, ollama documented behavior, and chains-DB measurements. See [`resources/prompt_strategy_findings.md`](resources/prompt_strategy_findings.md) for the full analysis.

### Why most "concurrency" tuning was wrong

The opencode CLI adds ~100k tokens of framework prompt (tool descriptions, permission system, session protocol) to every request. Our AILANG teaching prompt is only ~3k of the 109k input-tokens observed per trial. Two consequences:

1. **`OLLAMA_NUM_PARALLEL>1` pre-allocates KV cache for *every* slot at the full context length.** 4 slots × 110k-token KV cache ≈ 25 GB. With 17 GB weights this matches the observed 43 GB VRAM allocation. The Ollama FAQ confirms: *"Required RAM will scale by `OLLAMA_NUM_PARALLEL` × `OLLAMA_CONTEXT_LENGTH`."* ([source](https://docs.ollama.com/faq))
2. **The single integrated GPU cannot parallelise prefill of multiple 110k-token prompts.** The 15-min TTFT timeouts we kept hitting were scheduler starvation, not OOM — slots all compete for the same compute.

The single biggest win is **prefix caching across trials**: the 100k+ opencode framework prefix is identical between trials, so if ollama keeps the model resident and the cache hot, the second trial onward can skip the bulk of prefill.

### `internal/eval_harness/models.yml` per-Ollama-model entry

```yaml
opencode-gemma4-26b:
  api_name: "gemma4:26b"
  provider: "ollama"
  agent_cli: "opencode"
  agent_model_name: "ollama/gemma4:26b"
  max_output_tokens: 8192
  ttft_timeout: 600            # 10 min — at NUM_PARALLEL=1 a cold 110k-token prefill takes ~90s; warm is ~5s. 10 min covers cold + headroom
  generation_timeout: 1200     # 20 min — opencode per-session hard cap
  budgets:
    hard_timeout_secs: 2400    # 40 min — wall-clock safety net (only gate for $0 models)
  pricing:
    input_per_1k: 0.0
    output_per_1k: 0.0
```

### Ollama env vars (set via `launchctl setenv` then restart ollama)

```
OLLAMA_MAX_LOADED_MODELS=1     # only one model resident at a time
OLLAMA_NUM_PARALLEL=1          # 100k+ token prompts make >1 KV-cache-thrash. Hard-coded floor.
OLLAMA_KEEP_ALIVE=-1           # never unload model between trials — prefix-cache hits depend on this
OLLAMA_KV_CACHE_TYPE=q8_0      # halves KV memory footprint (~25 GB → ~12 GB) for ~5% gen slowdown on Metal
OLLAMA_FLASH_ATTENTION=1       # required for quantized KV (q8_0 silently falls back to fp16 without this)
OLLAMA_MAX_QUEUE=64            # back-pressure if eval queue gets ahead
```

VRAM math under these settings: 17 GB weights + ~12 GB q8 KV cache = **~29 GB allocated**, leaving ~99 GB free for the OS + future larger models.

### CLI flags for `make eval-smoke`

| Setting | Recommended | Why |
|---|---|---|
| `-agent-parallel 1` | **mandatory** | Match server. Oversubscribing puts requests behind a 15-min TTFT cliff |
| `-agent-timeout 1200` | default | Matches per-session cap; bump only if a benchmark is known to need more |
| `-langs ailang` | always | Local rig is for OS-model AILANG eval, not multi-lang |
| `-output eval_results/rotation/<date>/<time>_<model>_<tier>/` | recommended | Time-series-queryable structure |

> **Throughput expectation under these settings**: ~90-120s per benchmark on first cold trial, ~30-60s on subsequent trials within the same rotation (prefix cache hits). For 17 benchmarks × 3 trials = 51 trials sequentially, expect ~1.5-2 h wall clock — slower than the broken `-parallel 4` rotation that *completed in 40 min* by failing 100% of runs, but actually produces data.

## Prompt strategy (why we don't progressively disclose)

The 109k-token prefill cost looks like it should be addressable by progressive context disclosure — start with a tiny seed, let the agent pull in syntax via `ailang prompt`, `ailang docs std/X`, `ailang examples search`. This was tried and **deliberately abandoned**:

| Mechanism | Status | Why disabled |
|---|---|---|
| μRAG `PreToolUse` injection on `.ail` files | **DISABLED** ([ADR-002](../../../design_docs/decisions/ADR-002-pretooluse-microrag-disabled.md), 2026-04-27) | Embedding cosine averages over tokens — generic words like `import`/`module`/`! {IO}` dominate the vector, and a 1-char anti-pattern can't surface a relevant chunk |
| `ailang prompt` / `ailang docs` self-discovery | Encouraged in agent prompt | **Not used in practice** — the [v0.8.2 prompt notes](../../../cmd/ailang/prompts/agent/versions.json) record "0/63 agents use runtime discovery". The 13KB prompt embeds the stdlib reference directly because that observation was overwhelming |
| AILANG MCP server (`mcp.ailang.sunholo.com`) | Active for [Claude Code / Cursor / Continue](../../../docs/docs/guides/agent-mcp.md) | **Not used by opencode** — opencode doesn't run our MCP server as a tool source. The eval rig's agent path goes through opencode subprocess, never touches the MCP |
| PostToolUse `micro-rag lint-builtin` | Active | Surface "first use of a builtin" reminders. Tight scope, works fine |

The honest reading: the bulk of the 109k tokens are **opencode's own framework prompt**, not ours (we contribute ~3k). Shrinking our 3k buys little. The leverage is in:
1. Setting `OLLAMA_KEEP_ALIVE=-1` so the opencode framework prefix stays cached across trials
2. Setting `OLLAMA_NUM_PARALLEL=1` so we don't pre-allocate per-slot KV caches we never use
3. Accepting that the cold-start of trial-1 in a rotation is expensive; trials 2+ are cheap

If progressive disclosure becomes worth revisiting, the best fit per ADR-002 is a `UserPromptSubmit` hook (intent-bearing natural-language query) — not a content-similarity retrieval hook on file edits.

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
