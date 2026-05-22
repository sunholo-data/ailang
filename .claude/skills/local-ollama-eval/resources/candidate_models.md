# Candidate models for the local Ollama eval rotation

Cross-referenced from [ollama.com/library](https://ollama.com/library) on 2026-05-22. For each candidate the table records the Ollama tag, approximate disk size at Q4_K_M, and what it would teach us. Sizing assumes Apple M4 Max + 128 GB unified memory; the per-instance VRAM column is empirically derived from `gemma4:26b`'s observed 25.76 GB footprint.

**Llama family deliberately excluded** — Meta has stalled on open-model leadership; Qwen / DeepSeek / Gemma / GLM are the OS frontier in May 2026.

## Tier 1: Direct peer comparison to gemma4:26b (15–30 GB class — 2–4 concurrent on 128 GB)

| Model | Ollama tag | Disk Q4 | Per-instance VRAM | Why test |
|---|---|---|---|---|
| **Gemma 4 26B (current baseline)** | `gemma4:26b` | 17 GB | 25.76 GB measured | Already in rotation |
| Gemma 4 31B | `gemma4:31b` | ~18 GB | ~28 GB | Larger sibling; size-vs-capability in same family |
| Gemma 3 27B | `gemma3:27b` | ~15 GB | ~22 GB | Prior-gen comparison |
| Qwen 2.5 Coder 32B | `qwen2.5-coder:32b` | ~18 GB | ~28 GB | Coding-specialized; strongest contender in class |
| Qwen3 32B | `qwen3:32b` | ~20 GB | ~30 GB | Newer Qwen with reasoning + tools |
| Qwen3.5 35B | `qwen3.5:35b` | ~20 GB | ~30 GB | Newest Qwen (released May 2026) |
| Mistral Small 24B | `mistral-small:24b` | ~13 GB | ~18 GB | Efficient performer |
| Nemotron 3 Nano 30B | `nemotron-3-nano:30b` | ~17 GB | ~26 GB | NVIDIA's agentic-tuned model |
| Granite Code 34B | `granite-code:34b` | ~19 GB | ~28 GB | IBM, code-specialized |

## Tier 2: Large (70B class — single-instance only on this rig)

| Model | Ollama tag | Disk | Notes |
|---|---|---|---|
| DeepSeek R1 70B | `deepseek-r1:70b` | ~39 GB | Reasoning-tuned DeepSeek |
| Qwen3 72B (when released) | `qwen3:72b` | ~40 GB | Largest dense Qwen3 |

Force `OLLAMA_MAX_LOADED_MODELS=1` and cap at ~2 concurrent inference slots. Useful for "does scale alone matter for AILANG?" question.

## Tier 3: Small (7–14B — fast iteration, weaker results)

| Model | Ollama tag | Disk | Notes |
|---|---|---|---|
| Qwen 2.5 Coder 7B | `qwen2.5-coder:7b` | 4 GB | Already in models.yml as `ollama-qwen-coder` |
| DeepSeek Coder v2 16B | `deepseek-coder-v2:16b` | ~9 GB | Newer than `deepseek-coder:6.7b` we have |
| Phi-4 14B | `phi4:14b` | ~8 GB | Microsoft frontier-class for its size |
| Phi-4 Reasoning 14B | `phi4-reasoning:14b` | ~8 GB | Reasoning variant — relevant for AILANG's typed errors |
| Starcoder 2 15B | `starcoder2:15b` | ~8 GB | Code-specialized |
| Granite Code 8B | `granite-code:8b` | ~5 GB | IBM smaller variant |

## Tier 4: Cloud-only OS (route via OpenRouter — NOT on Ollama)

These are NOT installable locally. They're the cloud OS peers our local rig competes against. Already curated in `internal/eval_harness/models.yml`:

| Model | OR ID | Status |
|---|---|---|
| GLM 5 (z.ai) | `z-ai/glm-5` | PASS 3/3 smoke |
| GLM 5.1 (newer) | `z-ai/glm-5.1` | **NOT YET in models.yml** — worth adding |
| GLM 4.7 Flash | `z-ai/glm-4.7-flash` | PASS 3/3 with budgets |
| MiniMax M2.7 | `minimax/minimax-m2.7` | PASS 3/3 |
| DeepSeek V4 Flash | `deepseek/deepseek-v4-flash` | NEAR-MISS, has `:free` tier |
| Kimi K2.6 | `moonshotai/kimi-k2.6` | NEAR-MISS |
| Qwen 3.6 35B-A3B (newer MoE) | `qwen/qwen3.6-35b-a3b` | **NOT YET in models.yml** |

## Recommended pull order for the rotation

If building the rotation up incrementally:

1. **Baseline (already done)**: `gemma4:26b`
2. **Tier 1 priority**: `qwen2.5-coder:32b` (coding-specialized, near gemma4 size)
3. **Tier 1 followup**: `qwen3:32b` (reasoning + tools)
4. **Tier 3 fast iteration**: `phi4-reasoning:14b` (smaller, lots of concurrency)
5. **Tier 3 coding**: `deepseek-coder-v2:16b` (newer than the one already in models.yml)
6. **Tier 1 latest**: `qwen3.5:35b` (May 2026 release)
7. **Tier 2 (when curious)**: `deepseek-r1:70b` (cap at -parallel 1, single benchmark experiments)

Each pull is multi-GB. Don't pull all at once — disk and bandwidth ramp up linearly.
