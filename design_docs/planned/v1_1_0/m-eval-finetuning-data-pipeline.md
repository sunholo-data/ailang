# M-EVAL-FINETUNING-DATA-PIPELINE: Capture-to-QLoRA on the Local Rig

**Status**: Planned — **revised 2026-11-05** (see §"2026-11-05 revision" directly below; it supersedes the sections it names)
**Target**: Phase 0 (size baseline) + extractor can run now; first production fine-tune at AILANG 1.0.0 (standing decision, 2026-08-18). The original v0.24.0 → v0.25.0 targets are stale.
**Priority**: P2 — the rig is producing training data passively today; this sprint formalizes the path from capture → JSONL → MLX QLoRA → eval-on-rig
**Estimated**: Phase 0 ≈ 0.5 day (lanes + one baseline window); extractor ≈ 2 days (four sources, not one); recipe sanity + round-trip ≈ 2 days
**Dependencies**:
- M-EVAL-OS-LONGITUDINAL (rotation infrastructure — shipped)
- M-EVAL-METRICS-TAXONOMY (richer per-trial metrics — planned for v0.24.0)
- 128 GB Apple M4 Max rig with MLX (already operational)

## 2026-11-05 revision

Revisited with Mark because the rig is saturated: `qwen3.8:27b-mxfp8` occupies 49 GB at a 131K context, and ollama loads one model at a time. The question was whether **smaller AILANG-expert models** (faster, $0, little or no teaching prompt) are worth building. Answer: **viable, and worth a staged experiment with a kill criterion**, with four corrections to this doc and to the pitch.

### R1. "No AILANG prompt" does not hold while the language moves

There were 84 release tags in the 90 days to 2026-11-05 (latest v0.57.1). A fine-tune freezes one snapshot of the language. With no prompt, nothing corrects it after the next syntax or stdlib change, and it will write the old form with confidence. The realistic target is **shrinking** the teaching prompt (`ailang prompt` = 105,244 bytes, about 26k tokens) to a few KB of version-specific deltas and stdlib signatures, **not removing it**. This is the reason the production fine-tune waits for 1.0.0. The stdlib freeze gate (`make freeze-stdlib`) is the signal that the surface has stopped moving.

### R2. "Faster" mostly comes from model size, not from specialisation

Locally, the prompt costs prefill time and KV memory, not money, and prefix caching already spreads the prefill over a session. The binding constraint is **memory per lane**. A 4–8B model frees room for parallel lanes whether or not it is fine-tuned. Report throughput as **concurrent lanes × tokens/s**, and credit the fine-tune only with the part the base small model does not already provide.

### R3. The "nothing under 13B" anti-recommendation is unsourced: replaced by a measurement

The line arrived in this doc's first commit (`c26c58599`, 2026-05-23) with no citation and no measurement. It merges two different abilities:

- **Knowing the language** (syntax, types, effects, stdlib). Published work on fine-tuning for low-resource programming languages (the MultiPL-T line, which included 1B-class models on Racket/OCaml/Lua) reports large gains at small sizes. *Recalled, not re-verified: check the papers before quoting numbers.* AILANG is also not wholly novel: its ML-family syntax and HM types match priors from OCaml, Haskell and Rust.
- **Running an agent loop** (planning, reading compiler errors, recovering, tool calls over a long context). Small models really are weak here, and syntax SFT does not fix it. Error-recovery triples may help somewhat.

The rig has **zero banked results** for its small models (`gemma4:e4b`, `qwen3:8b`), so neither claim is measured. The anti-recommendation is withdrawn in favour of **Phase 0** below. The small end is also the cheapest place to test: a 4–8B QLoRA trains in hours, a 27B in days.

### R4. The corpus is four sources, not opencode alone

The original extractor read only `opencode.db`. Measured on the attended machine on 2026-11-05:

| Source | Size | Contents | Outcome label? |
|---|---|---|---|
| **Chains / observatory** (`~/.ailang/state/observatory.db`, 1.1 GB) | 3,804 chains (2,539 `motoko`, 905 `eval_suite`, 360 `manual`), 15,753 stages, 55,464 chat messages over ~1,100 sessions | 21,395 messages mention `.ail`; 15,632 carry tool results | **Yes**: `trial_history` holds 3,503 graded trials (agent 3,074 with 2,857 passes; standard 429 with 399 passes) |
| **motoko session JSONL** (`~/dev/mk-*/.motoko/logfile`) | ~3,180 files, ~2.2 GB over 9 checkouts | Full turns **including tool results** | Via `ailang chains import-motoko` → chains |
| **opencode.db** (`~/.local/share/opencode/opencode.db`) | 1.7 GB (was 1.1 GB / 9,111 sessions on 2026-08-18) | Full turns, mostly local Qwen | Only via eval result JSONs. **opencode has no chains importer** |
| **`eval_results/`** | 39,593 result JSONs over 82 run dirs | Final code, `error_category`, compile/runtime/stdout flags, standard + agent | **Yes**. The cheapest source for one-shot SFT and repair pairs |
| Claude Code transcripts (`~/.claude/projects/*ailang*`) | ~1,820 sessions (world repo alone: 861 / 1.1 GB) | Frontier models working in these repos | No. **Excluded by default** (see licensing below) |

**Not yet counted:** the cloud chain store (Firestore, `ailang-multivac`), which may hold more than the local observatory, and pi sessions, which may not be captured anywhere.

Consequences for the extractor (they supersede §"Data extraction" where the two conflict):

1. **Chains are the index.** They are the only place that links trajectory → stage → task → graded trial **across harnesses**. Use them to select and label examples; pull bodies from the motoko JSONL / `opencode.db` / `eval_results`.
2. **Model attribution must be recovered by join.** 45,747 of the 55,464 chat messages have a blank `model` column. Join chain → trial to recover it. Without it you cannot filter by quality or by licence.
3. **Licensing filter.** 8,758 chat messages are `claude-haiku-4-5`, and the Claude Code transcripts are frontier output. Provider terms generally restrict training models on their outputs, so check them before including any of it. Local open-weight output (Qwen, Gemma) is the bulk of the volume and is unrestricted. Default: **open-weight-generated examples only**; anything else needs an explicit ruling.
4. **Re-grade against today's compiler.** The corpus spans dozens of AILANG versions. Re-run `ailang check` (and the benchmark's grader where cheap) on every extracted program, and keep only what passes **at the training-time version**. This is free, and it is the main defence against training stale syntax in. Note that `ailang check` checks only its first file argument.
5. **The corpus is PASS-heavy** (93% of agent trials pass), so DPO pairs will be scarce. The error-recovery triples *inside* passing trajectories remain the gold data.

### Phase 0: size baseline (run before any training)

Run each model in **standard mode** (one-shot, full prompt), which measures knowledge of the language, and in **agent mode** (full prompt), which measures language plus agentic skill:

| Model | Role |
|---|---|
| `gemma4:e4b` | small (lane `opencode-gemma4-e4b` exists) |
| `qwen3:8b` | small (pulled; **no lane yet**) |
| `qwen3.8:27b` | reference |

- Tiers: **core + stretch**. Core alone saturates and has inverted verdicts before.
- Lanes: add `qwen3:8b` to `internal/modelreg/models.yml`, plus motoko/pi lanes for both small models so harness effects are not mistaken for size effects.
- Rig: $0, but it contends with the rotation for the GPU (one loaded model). Schedule a window or pause the rotation. Do not interleave.

How to read the results:

| Standard | Agent | Reading | Next |
|---|---|---|---|
| OK | collapses | the ceiling is agentic skill | a fine-tuned small model targets one-shot / short-loop fleet work, not full agent runs |
| collapses | collapses | it does not know the language | the best case for SFT: go to the pilot |
| OK | OK | the fine-tune mostly buys prompt shrinkage + lanes | pilot measures prompt-size reduction |

### Pilot and kill criterion

QLoRA one ~4–8B model on the R4-filtered corpus, then compare three arms of the **same** model: base + full prompt, fine-tuned + slim prompt, fine-tuned + no prompt. Report pass rate, `first_attempt_pass_rate`, concurrent lanes and tokens/s.

**Kill criterion:** fine-tuned + slim prompt must at least match base + full prompt on pass rate (core + stretch, paired, `ailang eval-paired`), while running more lanes. Otherwise stop, and record the negative result.

### Program fit

The program uses weak models as a **discovery mechanism**: friction they expose gets fixed in AILANG or the harness, and the fix lifts frontier models too. A fine-tune hides friction in its weights instead. So the specialist is its **own lane** (a cheap fleet worker), and it never replaces the prompt-based weak-model evals that feed the routing lanes in `design_docs/PROGRAM.md`. It is also not the "Sonnet-5 level at $0" goal, which stays on motoko + mid-size local.

## 2026-11-05 Phase 0 results (first pass)

Measured attended on 2026-11-05 at AILANG v0.57.1. Lanes: the `FINE-TUNE PHASE 0` block in `internal/modelreg/models.yml` (commit `74857f3c7`). Raw results are in `eval_results/finetune-phase0/` (gitignored, on the rig). Benchmarks: core (23) + stretch (28). The agent subset is 24 benchmarks, alternating core and stretch.

### Model choice

Gemma 4 is the primary family (Mark's preference). It is dense at E4B / 12B / 31B, plus a 26B-A4B MoE, and **Apache 2.0** (Hugging Face licence tag; confirm against the model card before publishing weights). No official Qwen 3.8 exists below 27B; the small Qwen sizes are all 3.5. The local rungs ran MXFP8 on Ollama's MLX runner. The cloud rungs ran on OpenRouter (open-weight models only, which the first-party routing rule allows).

### Standard mode (one-shot, full teaching prompt, self-repair on)

| Model | Where | Pass [95% CI] | Core | Stretch | Benchmarks solved at least once |
|---|---|---|---|---|---|
| qwen3.8-27b (reference) | cloud | **94%** [89–97] (153) | 100% | 89% | 50/51 |
| gemma-4-31b | cloud | 58% [50–66] (153) | 81% | 39% | 34/51 |
| **gemma4-12b** | local | 48% (66); **54% excluding 7 harness timeouts** | 65% | 31% (40% excl.) | 26/48 |
| gemma-4-26b-a4b (MoE) | cloud | 46% [39–54] (153) | 61% | 35% | 30/51 |
| ministral-14b | cloud | 33% (153) | 38% | 30% | 19/51 |
| ministral-8b | cloud | 30% (153) | 45% | 18% | 20/51 |
| qwen3.5-9b | cloud | 21% (152) | 30% | 13% | 16/51 |
| gemma4-e4b | local | 13% (110) | 14% | 11% | 9/51 |

- **The gap is language knowledge.** For every model below 27B, 71–89% of failures are compile errors. That is the branch of the §Phase 0 reading table that SFT addresses.
- **Harness confound (12B).** The 7 `api_error`s are `context deadline exceeded` from the Ollama client's **300 s per-call default** (`AILANG_OLLAMA_HTTP_TIMEOUT_SEC` unset), not the lane's 1800 s `generation_timeout`. 12B was still thinking (~46 tok/s × 300 s ≈ 13.8k tokens) on hard stretch benchmarks. Set `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800` for any local re-run.
- **Cloud spend: $12.15 metered** for 918 runs, against $5 approved and an $8 `--budget-usd` cap. The cap did not stop dispatch; that is fixed in `74857f3c7`. qwen3.8-27b alone cost $7.20, because OpenRouter bills its hidden reasoning tokens.

### Agent mode (pi harness, local, 1 trial)

| Model | Pass | Failure mix |
|---|---|---|
| gemma4-e4b | **0/24** | 13 `non_agentic` (no tool call), 11 runtime errors. **Ruled out as an agent target.** |
| gemma4-12b | **11/24** (core 7/12, stretch 4/12), **all 11 on the first attempt** | 6 `non_agentic`, 6 banked as `runtime_error` but **template-only**, 1 genuine `compile_error` (`emit_exact_bytes`, after 244k output tokens) |

**12 of 12B's 13 agent failures are tool-protocol misses, not wrong code.** When it did write a solution, it was correct 11 times out of 12. The protocol misses take two forms:
1. It answers in chat after one long think, with no tool call (`non_agentic`).
2. It uses tools but delivers the final code in chat, so the graded `solution.ail` is still the untouched template (`// TODO: Add your solution code below`). The harness banks that as `runtime_error` ("exports no functions").

There is only one graded wrong answer. "Finish by writing the file and running it" is exactly what agent-trajectory SFT teaches, since every training example ends in a write call and a verified run.

**Harness gap (separate item):** an unedited template should bank as its own category (e.g. `no_solution_written`), not `runtime_error`.

### Speed (measured on the rig)

- **Prefix cache works.** The Ollama log showed 266 hits and 7 misses; every miss was a first load or an attended probe. A cache hit costs about 1 s per request. A cold 33k-token prefill on 12B costs 70 s (467 tok/s).
- **Decode runs at the bandwidth ceiling.** E4B ≈ 65 tok/s; 12B ≈ 46 tok/s (≈ 12 GB read per token ≈ the M4 Max's ~546 GB/s). Ollama already uses the model's draft (MTP) speculative decode, with ~0.69 acceptance.
- **Per agent pass (same 21 benchmarks):**

| | Median time per pass | Median output tokens per pass | Wall time per verified success |
|---|---|---|---|
| 12B | 5.9 min | 12.7k | ~20 min |
| qwen3.8-27b | 9.3 min | 7.9k | **~12 min** |

  12B's ~2× decode advantage is mostly spent on ~1.6× longer thinking, with occasional blow-ups (one run produced 218k output tokens; pi does not enforce the lane's 32k cap). Its 48% pass rate then makes it slower per verified success. The fine-tune should therefore be measured on **output tokens per pass** and **wall time per verified success**, not pass rate alone.

### Parallelism (measured 2026-11-05: mlx-lm `batch_generate`, 12B 4-bit, 512/256 output tokens)

| Context | Streams | Total tok/s | Per stream | Gain over 1 stream | Peak memory |
|---|---|---|---|---|---|
| short (43 tokens) | 1 | 58.7 | 58.7 | 1.0× | 7.1 GB |
| short | 2 | 113.9 | 56.9 | 1.94× | 7.4 GB |
| short | **4** | **162.6** | 40.6 | **2.77×** | 8.1 GB |
| short | 8 | 177.7 | 22.2 | 3.03× | 9.3 GB |
| **33k (the AILANG agent prompt)** | 1 | 51.3 | 51.3 | 1.0× | 11.2 GB |
| **33k** | **4** | **64.4** | **16.1** | **1.26×** | 23.1 GB |

- **Short context: batching works**, about 2.8× at 4 streams, levelling off past 4 (probably 4-bit dequantization becoming the bottleneck).
- **At a real agent context it collapses to ~1.26×.** Memory grew ~3 GB per extra stream. So mlx-lm's batched cache does not get the sliding-window benefit that my pre-measurement KV estimate assumed (~0.6 GB per stream), and attention over the long context dominates each step. Each stream also pays its own cold prefill (4 × 33k tokens ≈ 5 min at 428 tok/s).
- **With today's prompts, parallel 12B agents on the rig give only ~1.3× throughput.** This measures mlx-lm's batching only; Ollama's runner and vLLM on a cloud L4 (paged attention) may differ.
- **Fine-tuning changes this.** Most of the 33k context is the teaching prompt, which a fine-tuned model drops. At 5–15k-token agent contexts, batching should move back towards the short-context curve. **Dropping the prompt is what makes parallel agents pay off**: a second, independent reason to fine-tune.
- Rig constraints still apply: `OLLAMA_NUM_PARALLEL=1` is rig-wide; mlx-lm disables batching when a draft model is used; rig evals assume `--parallel 1`, and the cooperative GPU yield only works at 1.

Script: `~/models/finetune/batchbench.py`; results in `~/models/finetune/batchbench-12b.json`.

### Candidate ranking

1. **Gemma 4 12B dense: primary (speed target).** 54% one-shot. Agent failures are protocol-only. 22 GB, 46 tok/s. The `gemma4_unified` architecture loads in stock mlx-lm **0.32.2** (remapped to `gemma4`), so it needs neither mlx-lm from git main nor `mlx-optiq`.
2. **Gemma 4 31B dense: quality-ceiling arm.** 58% one-shot, plain `gemma4`, ~20 tok/s.
3. **Gemma 4 26B-A4B MoE: optional speed arm.** 46%, 4B active, so fastest decode. MLX MoE training is the risk.
4. **Teacher, not target: qwen3.8-27b.** 94% one-shot, $0 locally. It can generate verified SFT data in bulk with none of the closed-model licensing issues in §R4. That makes the exporter less urgent for a first pilot.
5. Ruled out: E4B, qwen3.5-9b, Ministral 8B/14B. Deferred: the Empero Qwen3.8-9B community distill (unverified provenance).

### Tooling check (2026-11-05): training works; serving through Ollama needs one more fix

Run with stock **mlx-lm 0.32.2** (PyPI) in an isolated uv env at `~/models/finetune/.venv`, on `mlx-community/gemma-4-12B-it-4bit` and `gemma-4-31b-it-4bit`. Training data: 136 verified qwen3.8 solutions with no teaching prompt (plumbing only; see the contamination rule below). LoRA settings: `--mask-prompt --num-layers 16 --batch-size 1 --iters 40 --lr 1e-4 --grad-checkpoint`. Script: `~/models/finetune/toolcheck.sh`.

| Step | 12B | 31B |
|---|---|---|
| Load | ✅ `gemma4_unified` is remapped to `gemma4` in stock mlx-lm, so neither git main nor `mlx-optiq` is needed | ✅ |
| LoRA, 40 steps | ✅ **177 s**, **13.4 GB** peak, ~150 tok/s training; train loss 1.03 → ~0.65, val 0.759 → 0.750 | ✅ **398 s**, **25.7 GB** peak, val 2.27 |
| Adapter size | 44 MB | — |
| Generation (4-bit) | 58.6 tok/s, 7.1 GB | 25.2 tok/s, 18.1 GB |
| Base, no teaching prompt | reasons that "AILANG is not a known language" | invalid output |
| After 40 steps | **skips thinking**, emits a correct `module …` header and `export func main() -> () ! {IO}`, then a repetition loop | a one-line program that **type-checks up to one missing import** (`std/list (range)`), then a loop |
| `mlx_lm.fuse --dequantize` | ✅ 21 s, 22 GB | not tested |
| `ollama create` from the fused safetensors | ⚠️ **imports** (`gemma4_unified`, 11.9B, tools capability, no GGUF conversion needed) but **fails to load**: `config declares a gemma4_unified_audio embedder but embed_audio weights are missing`. mlx-lm strips the vision/audio weights, and `fuse` keeps the original config. Next fix to try: drop the audio/vision sections from the fused `config.json` before `ollama create`; alternatives are restoring the base model's audio/vision weights or serving via `mlx_lm.server` | not tested (plain `gemma4`, so it may import cleanly) |

A real run is a different scale: a pilot on ~2–5k teacher solutions is about 3–7M tokens per epoch, so **~6–13 h per epoch for 12B** (overnight, with the rotation paused) and about 2.5× that for 31B. The toy output loops are the expected signature of a badly undertrained adapter decoded greedily; they say nothing about quality. Still open: the Ollama load failure above (open question 2 is narrowed, not resolved), and importing quantized (q8) rather than at 23 GB.

**Contamination rule:** never train on the eval benchmarks' own solutions. Teacher data must come from held-out tasks, or the before/after A/B is void. The 2026-11-05 tooling-check adapter (`~/models/finetune/adapters-toolcheck/`) was trained on eval solutions on purpose, as plumbing, and **must never be scored**.

## Serving cost reference (2026-11-05)

The question was how a fine-tuned 12B would be served beyond the rig, and what each option costs. These are list prices from web search on 2026-11-05 (USD). They may be stale (the Cloud Run page metadata was ~15 months old), so verify before budgeting. Per-run figures use today's measured 12B agent profile: ≈ 0.7–2M input tokens (mostly a cacheable prefix) and ≈ 10–35k output tokens per benchmark.

**The agent loop needs only CPU (Cloud Run already runs fleet agents). Only model inference needs a GPU.**

| Option | Billing | Rate found | ≈ per agent run | Fits our fine-tune? |
|---|---|---|---|---|
| **Rig (M4 Max)** | electricity; hardware sunk | ~100 W under GPU load ≈ €22/month 24/7 at an assumed ~€0.30/kWh | **≈ €0.005** | yes; **private**; capacity = one box (~4–8 batched streams) |
| **Cloud Run + NVIDIA L4** | per second while an instance is up; scales to zero; GPU needs instance-based billing; idle min-instances bill at full rate | L4 **$0.0001867/s ≈ $0.67/h** (non-zonal) + required 4 vCPU / 16 GB ≈ **$1/h all-in** (my sum); L4 regions include europe-west1 and europe-west4 | **≈ $0.04** batched ×4; ≈ $0.15–0.30 single-stream | yes (vLLM + adapter). Best cloud overflow option |
| **Vertex AI dedicated endpoint** (g2-standard-4, 1× L4) | per node-hour, **including idle**, while deployed | **$0.81/h ≈ $585/month** always-on (1-yr CUD $0.73/h; 3-yr $0.65/h) | ≈ $0.04 when busy | yes; only pays at steady load (~15k runs/month to beat Cloud Run) |
| **Vertex managed Gemma 4 (MaaS)** | per token | Gemma 4 26B: **$0.15/M input, $0.60/M output, $0.015/M cached** | ≈ $0.03–0.05 if the cache hits | **no**: no 12B listed, and no evidence it serves custom LoRA. Useful only as a base-model comparison |
| **Fireworks serverless multi-LoRA** | per token; adapter on a shared base pool at base-model price | 4–16B tier ≈ **$0.20/M** (third-party figure) | ≈ $0.20 (input-heavy) | maybe: Gemma 4 12B support not confirmed |

Notes:
- The L4's memory bandwidth (~300 GB/s) is about half the rig's (~546 GB/s), so a single stream is slower in the cloud. **Batching is what makes Cloud Run pay.**
- Our runs are about 98% input tokens, so **prompt-cache pricing dominates every per-token option**. We have measured OpenRouter billing ≈ 4× traced on cached runs; any per-token option needs a real billed test before we rely on it.
- Recommended shape: **rig first, cloud overflow.** Batched 12B on the rig handles routine AILANG work at ≈ $0. The same adapter goes to a scale-to-zero Cloud Run L4 service for bursts. Watch the known `minScale`/`cpu_idle` idle-billing trap.

Sources: [Cloud Run pricing](https://cloud.google.com/run/pricing) · [Cloud Run GPU (services)](https://docs.cloud.google.com/run/docs/configuring/services/gpu) · [Cloud Run GPU (jobs)](https://docs.cloud.google.com/run/docs/configuring/jobs/gpu) · [Cloud Run GPUs GA](https://cloud.google.com/blog/products/serverless/cloud-run-gpus-are-now-generally-available) · [Vertex AI pricing](https://cloud.google.com/vertex-ai/pricing) · [Vertex inference compute](https://docs.cloud.google.com/vertex-ai/docs/predictions/configure-compute) · [Model Garden pricing](https://cloud.google.com/model-garden/pricing) · [Gemma 4 pricing (markaicode)](https://markaicode.com/pricing/gemma-4-pricing/) · [Fireworks pricing (Spheron)](https://www.spheron.network/blog/fireworks-ai-pricing-2026-inference-api/) · [Fireworks vs Together (Morph)](https://www.morphllm.com/comparisons/fireworks-vs-together)

## Problem statement

We've been capturing agentic eval trajectories for months as a side-effect of normal `ailang eval-suite` runs: `~/.local/share/opencode/opencode.db` already has 3,751+ structured "part" rows (every reasoning block, tool call, write, bash command, MCP discovery query, model response) across 176+ sessions. The per-trial result JSONs have the model's final code, the error category, the outcome.

This is **production-grade SFT/DPO training data** sitting on disk, unused. The bottleneck preventing us from fine-tuning a custom AILANG-specialist model isn't data — it's:

1. We don't have a pipeline to extract opencode.db → mlx-lm-ready JSONL
2. We haven't decided on a target model (and the choice matters — see §"Model selection" below)
3. We haven't established the eval round-trip so a fine-tune can be measured against the same smoke tier its training data came from

This design doc establishes those three pieces.

## Goals

1. **Extraction**: a `ailang export-training-data` subcommand that converts opencode.db sessions into mlx-lm `tools` JSONL, with explicit filtering (PASS trajectories only for SFT; PASS+FAIL pairs for DPO)
2. **Modern model selection**: an explicit recommendation that DOES NOT use llama (Meta's OS line has plateaued); first-attempt target is `qwen3-coder:30b` with a clear scale-up path informed by the rig's own leaderboard data
3. **Training recipe**: a documented MLX QLoRA config that fits comfortably in the 128 GB budget (per the 2026-05-23 capacity research: 32B QLoRA peak ≈ 50 GB, ~46 GB headroom remaining)
4. **Round-trip evaluation**: same smoke tier that produced the training data evaluates the fine-tune. Baseline vs fine-tuned A/B becomes a cell in the leaderboard cross-product (per M-EVAL-METRICS-TAXONOMY)

## Why NOT llama (explicit anti-recommendation)

The 2026-05-23 capacity research suggested `llama-3.3-70b` as a production scale-up target. That recommendation is **rejected** here for the following concrete reasons:

| Concern | Evidence |
|---|---|
| Meta's Llama release cadence has slowed dramatically | No major Llama release in ~12+ months as of 2026-05; Meta's focus shifted to product/multimodal work |
| Qwen 3, GLM 5, DeepSeek v3, Kimi K2 each surpass Llama 3.3 on code benchmarks | Multiple 2025-2026 community evals (LMSYS Arena code subset, BigCodeBench, EvalPlus) put each of these ahead of Llama 3.3-70b |
| Apple Silicon community is converging on Qwen 3 / DeepSeek as the de facto local-frontier code models | mlx-community quants prioritize these; r/LocalLLaMA top threads in 2026 are about Qwen 3 / DeepSeek, not Llama |
| Llama 3.3 license is more restrictive than Qwen / DeepSeek | Both Qwen 3 and DeepSeek v3 ship under Apache 2.0; Llama has an acceptable-use policy with size triggers |

The local-Ollama leaderboard the rig publishes (M-EVAL-OS-LONGITUDINAL Phase 5, shipped) is the authoritative ranking for our specific workload. Once we have rotation data for 3-4 candidate models, the choice of which to fine-tune is **data-driven**, not researcher-suggested.

## Modern model candidates (2026-05 OS frontier)

Ordered by my current best guess of likelihood-to-be-the-winner, with the understanding that the rig data will refine this:

| Tier | Candidate | Why | Fits in 128 GB QLoRA? |
|---|---|---|---|
| **Initial** | `qwen3-coder:30b` | Current OS code leader (Alibaba), code-specialized, modern Apache-2.0 license, MLX-community quants available | Yes, ~50 GB peak (comfortable) |
| Alt initial | `glm-5:30b` (z.ai) | Observed 3/3 PASS in agent mode on prior cross-harness eval (2026-05-04); strong code generation | Yes, similar footprint |
| Stretch | `qwen3:72b` | If we want to push to the larger tier; Alibaba's general-purpose flagship | Yes, ~80-90 GB peak (tight but routine with `iogpu.wired_limit_mb=115000`) |
| Stretch alt | `deepseek-coder-v3` | DeepSeek's code-specialized v3 family; competitive with Qwen on hardest benchmarks | Yes if 33B variant; the 236B MoE doesn't fit even at QLoRA |
| Stretch alt | `kimi-k2` | Moonshot's frontier; strong agentic eval scores | Depends on release size — verify before committing |
| Already-running | `gemma4:26b-ailang` (our existing variant) | Currently producing the training data; could fine-tune on its own outputs as a self-distillation experiment | Yes — already verified |

**Concrete first-attempt recipe**: fine-tune `qwen3-coder:30b` (~50 GB peak QLoRA footprint, ~7-12 days for 3 epochs on 100k examples per the researcher's M4 Max throughput numbers).

> **2026-08-18 refresh** (candidates table above is the 2026-05 frontier — kept for the reasoning, superseded for the pick):
>
> - **Refreshed first-attempt target: `qwen3.8:27b` (dense).** Same shape the table asks for (dense, mid-size, Qwen family, ~45-50 GB QLoRA peak), one generation newer than `qwen3-coder:30b`. Already on the rig: ollama upgraded to 0.32.14 and `qwen3.8:27b-mxfp8` pulled 2026-08-17; cleared the OpenRouter quality screen (+15.9pp AILANG vs qwen3.6-35b-a3b, McNemar p=0.0094, see `or-qwen3-8-27b` in models.yml); on-device A/B lanes registered (`opencode-/pi-/motoko-local-qwen3-8-27b`), rotation promotion pending the quality-vs-throughput A/B. Being family-adjacent to the qwen3.6 trajectories that dominate the corpus, the training data transfers with no tokenizer mismatch.
> - The incumbent `qwen3.6:35b-a3b` remains the **data source**, not a fine-tune target — it's MoE, which the anti-recommendations below already exclude for MLX training.
> - **Corpus readiness is no longer the bottleneck**: opencode.db measured 2026-08-18 at 9,111 sessions / 272,451 parts / 1.1 GB (vs 176 sessions / 3,751 parts when this doc was written). The extractor is now the only thing between us and knowing the true PASS-trajectory count.
> - Unsloth re-checked 2026-08 (their Desktop app launch): still **no Apple Silicon training** — MLX QLoRA remains the stack. The MLX→GGUF/serving question (open question 2) may have moved with ollama ≥0.32.12's qwen3.8 support; re-verify at execution time.

**Anti-recommendations** (do NOT pick these for fine-tuning even though they appear in ollama's library):

- ❌ `llama-3.3-70b` and variants — see §"Why NOT llama" above
- ❌ Older code models (`starcoder2`, `codellama`) — superseded by the Qwen 3 generation
- ~~❌ Anything under 13B for AILANG-specialist training — too small to absorb a new language effectively in a reasonable corpus size~~ **Withdrawn 2026-11-05**: unsourced. Replaced by the Phase 0 size baseline (§R3).
- ❌ MoE models > 60 GB total parameter size at full precision — even at 4-bit, the routing overhead in MLX can spike memory beyond budget

## Data extraction: `ailang export-training-data`

> **2026-11-05:** the single-source design below is superseded by §R4. The extractor indexes on chains/observatory and reads bodies from motoko JSONL, `opencode.db` and `eval_results/`, with model attribution, a licence filter and a re-grade against the current compiler. Under the S5 CLI consolidation this is probably a subcommand of an existing namespace rather than a new top-level `export-*`; check `make simplicity-audit` before adding a route. The opencode SQL notes below remain valid for that one source.

New subcommand under `ailang export-*`. Reads `~/.local/share/opencode/opencode.db` + the rotation result JSONs, emits mlx-lm-format JSONL.

### Output shape (mlx-lm `tools` format)

```jsonc
{
  "messages": [
    {"role": "system", "content": "<the agent prompt that was active for this trial>"},
    {"role": "user",   "content": "<the benchmark task description>"},
    // ... assistant turns with reasoning + tool_calls + tool returns ...
    {"role": "assistant", "tool_calls": [
      {"id": "call_abc", "type": "function",
       "function": {"name": "write", "arguments": "{\"filePath\":\"solution.ail\",\"content\":\"...\"}"}}
    ]},
    {"role": "tool", "tool_call_id": "call_abc", "content": "Wrote file successfully."},
    // ... more turns until PASS ...
    {"role": "assistant", "content": "Solution verified. Output matches expected."}
  ],
  "tools": [
    // The OpenAI-style schema for each tool the agent had access to.
    // We pull these from opencode's published schemas + our MCP server's tools/list output.
  ]
}
```

### Filtering modes

```bash
ailang export-training-data --filter pass-only       # SFT: only successful trajectories
ailang export-training-data --filter dpo-pairs       # DPO: (pass, fail) pairs per (benchmark, model)
ailang export-training-data --filter error-recovery  # Only (write, ailang-check error, write-that-fixed-it) sequences — gold for teaching error→fix patterns
ailang export-training-data --filter all             # Raw dump
```

### Implementation notes

- SQL query joins `session` → `message` → `part` ordered by `time_created`, reshapes into the messages array
- For DPO pairs: GROUP BY (benchmark_id, model_id) with both PASS and FAIL trials; pick the median-length PASS as `chosen`, the highest-quality (most-attempts) FAIL as `rejected`
- For error-recovery extraction: find consecutive `(write, bash[ailang check|run with error output], write)` triples in the part sequence where the second write was followed by a PASS. Each triple becomes one SFT example: input = (prior code + error message), output = (fixed code)
- Tool schemas: pull from a static `internal/eval_harness/tool_schemas.json` (we author this once; opencode's catalog + MCP's tools/list become the source)

### Estimated corpus size

> **2026-11-05:** the estimates below are from 2026-05 and are superseded by the measured source table in §R4. The true count of usable examples is unknown until the extractor runs with the re-grade filter.

Today's database: 176 sessions, ~3,751 parts. Most sessions had 5-20 parts. Roughly:
- 50-150 PASS trajectories per rotation, average ~10 parts each (with reasoning, write, bash) → ~500-1500 SFT examples per rotation
- 10-30 PASS-FAIL pairs per rotation → DPO corpus grows slowly
- 100-300 error-recovery triples per rotation → high-value error-specific training examples

After ~50 overnight rotations (3 months of continuous running): ~25k-75k SFT examples, sufficient for an initial fine-tune. The 100k target is reachable in ~4-5 months of normal operation.

## Training recipe (MLX QLoRA on qwen3-coder:30b)

Per the 2026-05-23 capacity research, MLX is the only mature Apple Silicon training path. PyTorch-MPS has open memory-leak bugs (pytorch/pytorch#154329, #164299, #121113); unsloth doesn't support Apple Silicon; llama.cpp is inference-only.

### Recipe

```bash
# 1. Pull base model (4-bit MLX quant)
huggingface-cli download mlx-community/Qwen3-Coder-30B-Instruct-4bit --local-dir ~/models/qwen3-coder-30b-4bit

# 2. Train (mlx-lm CLI)
mlx_lm.lora \
    --model ~/models/qwen3-coder-30b-4bit \
    --train \
    --data ~/training-data/ailang-agentic-sft-v1.jsonl \
    --batch-size 2 \
    --seq-length 4096 \
    --num-layers 16 \
    --learning-rate 1e-4 \
    --optimizer adamw \
    --iters 12500 \
    --steps-per-eval 500 \
    --save-every 1000 \
    --adapter-path ~/adapters/qwen3-coder-30b-ailang-v1
```

**Expected peak memory**: ~50 GB
**Expected wall time**: 3 epochs over 100k examples ≈ 7-12 days
**Disk**: 4-bit base ~17 GB + adapter checkpoints ~500 MB each
**iogpu.wired_limit_mb**: bump to ~96000 before the run; ~115000 only needed for 70B+

### Validation during training

- Save adapter every 1000 iters
- Every 5000 iters, fuse adapter into a checkpoint and run smoke-tier evaluation via the same `ailang eval-suite -agent` we use for baselines
- Track: pass_rate, first_attempt_pass_rate (the killer signal of whether the FINE-TUNE actually helped vs the SAMPLING was just lucky), median iterations to pass, error_recovery_rate per error_category
- Compare against baseline cell (model=qwen3-coder:30b, sampling=ailang-tuned, prompt=v0.10.0-slim) — the leaderboard already has this slot from M-EVAL-METRICS-TAXONOMY

## Eval round-trip

After fine-tuning produces an adapter:

```bash
# 1. Fuse adapter into a deployable model
mlx_lm.fuse \
    --model ~/models/qwen3-coder-30b-4bit \
    --adapter-path ~/adapters/qwen3-coder-30b-ailang-v1 \
    --save-path ~/models/qwen3-coder-30b-ailang-finetuned

# 2. Make it available to ollama (convert MLX → GGUF → import)
#    Note: MLX → GGUF conversion limited to Llama-style architectures.
#    For Qwen-style, keep the fused model in MLX format and serve via mlx-server,
#    register as a separate opencode provider entry pointing at the mlx-server endpoint.

# 3. Add to models.yml as opencode-qwen3-coder-30b-ailang-finetuned
$EDITOR internal/eval_harness/models.yml

# 4. Run the same smoke tier the training data came from
make eval-smoke MODELS=opencode-qwen3-coder-30b-ailang-finetuned \
  EXTRA="-trials 3 -parallel 1 -agent-timeout 1200 -langs ailang \
         -output eval_results/rotation/$(date +%Y-%m-%d)/finetuned_v1_smoke"

# 5. Compare in the leaderboard
ailang eval-publish v0.25.0 \
  --rotation eval_results/rotation/<finetuned-date> \
  --prev    eval_results/rotation/<baseline-date> \
  --prev-tag v0.24.0
```

The fine-tune's value is measured by the **delta** in the metric set from M-EVAL-METRICS-TAXONOMY:
- Pass rate should rise (headline)
- First-attempt pass rate should rise more steeply (the model genuinely knows AILANG better, not just iterates better)
- Median iterations to pass should DROP (model converges faster on the answer)
- Error recovery rate per AILANG-specific error code should rise (the fine-tune internalized the language's error semantics)

## Conflict surface

This sprint touches:
- `cmd/ailang/export_training_data.go` (new) — JSONL extractor
- `internal/eval_harness/tool_schemas.json` (new) — canonical tool schema list for the `tools` field of mlx-lm JSONL
- `internal/modelreg/models.yml` (additive): Phase 0 lanes (`qwen3:8b`, motoko/pi lanes for the small models) and the fine-tuned model entry. *(The path was `internal/eval_harness/models.yml` when this doc was written.)*
- Read-only access to `~/.ailang/state/observatory.db` and the motoko session JSONL (2026-11-05, §R4)

Does NOT touch parser, typechecker, codegen, runtime, ollama config, opencode config, harness — pure data-pipeline + new subcommand.

## Open questions

1. **Self-distillation vs cross-model distillation**: should we fine-tune gemma4:26b on its OWN passing trajectories (cheap, fast, but limited ceiling), or on qwen3-coder:30b's (or claude-sonnet's) passing trajectories (better quality input, but the trajectory style might not transfer)?
2. **MLX → GGUF for Qwen models**: known limitation per the capacity research — MLX export to GGUF is Llama-style only. For Qwen we need to either (a) serve via mlx-server in production, or (b) wait for the conversion to support Qwen, or (c) use opencode's MLX provider if/when one exists. Need to verify the runtime path before committing.
3. **Training corpus quality threshold**: what's the minimum number of PASS trajectories we need before fine-tuning makes sense vs improving the prompt? My intuition: ~5k SFT examples (~10 rotations × 500 ex/rotation) for a meaningful first run. Worth confirming with a small ablation (1k vs 5k vs 25k).
4. **Continuous fine-tuning loop**: once we have a v1 fine-tune, do we re-train monthly with fresh rotation data? Or only when smoke-tier pass rate drops below a threshold? *(2026-11-05: before 1.0.0 the release cadence, 84 tags in 90 days, makes this the main cost. Re-train per minor version, or keep a slim delta prompt, as in R1.)*
5. **(2026-11-05) Cloud chain store and pi coverage**: does the Firestore chain store (`ailang-multivac`) hold trajectories the local observatory lacks, and are pi sessions captured anywhere? Count both before sizing the extractor.
6. **(2026-11-05) Licensing ruling**: may examples generated by closed models (Claude Haiku in chains, Claude Code transcripts) enter the training set? Default is no until ruled.

## What "done" looks like

0. **(2026-11-05)** Phase 0 baseline banked: standard + agent × core + stretch for `gemma4:e4b`, `qwen3:8b` and `qwen3.8:27b`, with a written reading against the §Phase 0 table
1. The extractor (`ailang export-training-data --filter pass-only --output ailang-sft-v1.jsonl` in the original design) produces a valid mlx-lm `tools` JSONL from **all four §R4 sources**, labelled by model, licence-filtered and re-graded against the current compiler, and reports the real count of usable examples per source
2. A documented training recipe (this doc + a `tools/finetune/README.md` operational guide) so a non-expert can launch a fine-tune with one command
3. The first fine-tuned model is registered in `models.yml` and runs through the eval harness without code changes
4. The leaderboard renders the baseline vs fine-tuned cells side-by-side under M-EVAL-METRICS-TAXONOMY's cross-axis schema
5. A published `docs/docs/reference/os-model-leaderboard/v0.25.0.md` shows the first concrete evidence (positive or negative) that local fine-tuning works for AILANG-specialist code generation

## Why this matters strategically

Two reasons we want this pipeline ready (even if we don't run the first fine-tune for months):

1. **Every rotation collects free training data**. Without the extraction pipeline, that data accumulates but isn't usable. Building the extractor early means we're maximally prepared whenever the corpus is large enough.

2. **The leaderboard's `first_attempt_pass_rate` metric is the cleanest signal for "is this model actually competent at AILANG"**. A fine-tune that lifts first-attempt rate from (say) 33% to 70% on the same benchmarks proves the local rig isn't just a measurement tool — it's the **engine** for producing better AILANG models. That's the strategic differentiator the rig was built to enable.

The 128 GB Mac Studio + MLX QLoRA path is uniquely well-suited to specialized fine-tuning of mid-size code models. Most teams don't have this setup. We do. The data is there. The capacity is there. The eval pipeline is there. This sprint connects the wires.
