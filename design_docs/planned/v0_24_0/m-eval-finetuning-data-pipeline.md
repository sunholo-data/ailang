# M-EVAL-FINETUNING-DATA-PIPELINE: Capture-to-QLoRA on the Local Rig

**Status**: Planned
**Target**: v0.24.0 (data extraction) → v0.25.0 (first end-to-end fine-tune)
**Priority**: P2 — the rig is producing training data passively today; this sprint formalizes the path from capture → JSONL → MLX QLoRA → eval-on-rig
**Estimated**: 3 days (1 day extractor, 1 day training-recipe sanity, 1 day eval round-trip)
**Dependencies**:
- M-EVAL-OS-LONGITUDINAL (rotation infrastructure — shipped)
- M-EVAL-METRICS-TAXONOMY (richer per-trial metrics — planned for v0.24.0)
- 128 GB Apple M4 Max rig with MLX (already operational)

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

**Anti-recommendations** (do NOT pick these for fine-tuning even though they appear in ollama's library):

- ❌ `llama-3.3-70b` and variants — see §"Why NOT llama" above
- ❌ Older code models (`starcoder2`, `codellama`) — superseded by the Qwen 3 generation
- ❌ Anything under 13B for AILANG-specialist training — too small to absorb a new language effectively in a reasonable corpus size
- ❌ MoE models > 60 GB total parameter size at full precision — even at 4-bit, the routing overhead in MLX can spike memory beyond budget

## Data extraction: `ailang export-training-data`

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
- `internal/eval_harness/models.yml` (additive) — entry for the fine-tuned model

Does NOT touch parser, typechecker, codegen, runtime, ollama config, opencode config, harness — pure data-pipeline + new subcommand.

## Open questions

1. **Self-distillation vs cross-model distillation**: should we fine-tune gemma4:26b on its OWN passing trajectories (cheap, fast, but limited ceiling), or on qwen3-coder:30b's (or claude-sonnet's) passing trajectories (better quality input, but the trajectory style might not transfer)?
2. **MLX → GGUF for Qwen models**: known limitation per the capacity research — MLX export to GGUF is Llama-style only. For Qwen we need to either (a) serve via mlx-server in production, or (b) wait for the conversion to support Qwen, or (c) use opencode's MLX provider if/when one exists. Need to verify the runtime path before committing.
3. **Training corpus quality threshold**: what's the minimum number of PASS trajectories we need before fine-tuning makes sense vs improving the prompt? My intuition: ~5k SFT examples (~10 rotations × 500 ex/rotation) for a meaningful first run. Worth confirming with a small ablation (1k vs 5k vs 25k).
4. **Continuous fine-tuning loop**: once we have a v1 fine-tune, do we re-train monthly with fresh rotation data? Or only when smoke-tier pass rate drops below a threshold?

## What "done" looks like

1. `ailang export-training-data --filter pass-only --output ailang-sft-v1.jsonl` produces a valid mlx-lm `tools` JSONL from the current opencode.db
2. A documented training recipe (this doc + a `tools/finetune/README.md` operational guide) so a non-expert can launch a fine-tune with one command
3. The first fine-tuned model is registered in `models.yml` and runs through the eval harness without code changes
4. The leaderboard renders the baseline vs fine-tuned cells side-by-side under M-EVAL-METRICS-TAXONOMY's cross-axis schema
5. A published `docs/docs/reference/os-model-leaderboard/v0.25.0.md` shows the first concrete evidence (positive or negative) that local fine-tuning works for AILANG-specialist code generation

## Why this matters strategically

Two reasons we want this pipeline ready (even if we don't run the first fine-tune for months):

1. **Every rotation collects free training data**. Without the extraction pipeline, that data accumulates but isn't usable. Building the extractor early means we're maximally prepared whenever the corpus is large enough.

2. **The leaderboard's `first_attempt_pass_rate` metric is the cleanest signal for "is this model actually competent at AILANG"**. A fine-tune that lifts first-attempt rate from (say) 33% to 70% on the same benchmarks proves the local rig isn't just a measurement tool — it's the **engine** for producing better AILANG models. That's the strategic differentiator the rig was built to enable.

The 128 GB Mac Studio + MLX QLoRA path is uniquely well-suited to specialized fine-tuning of mid-size code models. Most teams don't have this setup. We do. The data is there. The capacity is there. The eval pipeline is there. This sprint connects the wires.
