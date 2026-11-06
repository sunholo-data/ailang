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

Measured attended on 2026-11-05 at AILANG v0.57.1. Lanes: the `FINE-TUNE PHASE 0` block in `internal/modelreg/models.yml` (commit `c39576ef3`). Raw results are in `eval_results/finetune-phase0/` (gitignored, on the rig). Benchmarks: core (23) + stretch (28). The agent subset is 24 benchmarks, alternating core and stretch.

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
- **Cloud spend: $12.15 metered** for 918 runs, against $5 approved and an $8 `--budget-usd` cap. The cap did not stop dispatch; that is fixed in `c39576ef3`. qwen3.8-27b alone cost $7.20, because OpenRouter bills its hidden reasoning tokens.

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
4. **Training data: frontier-authored data first; the qwen3.8 teacher only fills gaps** (Mark, 2026-11-05: "why create training data with qwen3.8 via the GPU when we have incredible results from opus/astra"). Already on disk, needing no GPU:
   - **≈ 225k lines of AILANG in ≈ 1,600 files** across our repos, mostly frontier-agent authored: world 100k, ailang-parse 31k, packages 22k, daneel 21k, examples 20k, docparse 13k, email-parse 12k, std 6k. stapledons-godot shows 879k lines but needs a dedupe/generated-code check first.
   - **Frontier agent trajectories** (Claude Code sessions in AILANG repos, ~1.4k+; motoko/chains runs). These are exactly the "write the file, run it, read the error, fix it" behaviour 12B lacks.

   Conditions:
   - **(a) Contamination:** frontier *eval* results are solutions to the scored benchmarks. Exclude them, or hold out a benchmark set.
   - **(b) Licence: D-FT-1 = YES** (Mark, attended, 2026-11-05). Frontier-generated code and trajectories (Anthropic and OpenAI outputs) **may** be used to train the internal AILANG model.
   - **(c) Re-grade:** everything gets re-checked with today's `ailang check`; keep only what compiles.
   - **(d) Tool mapping:** Claude Code's tool names (Edit/Write/Bash) are mapped onto pi's schema.

   qwen3.8-27b (94% one-shot, $0 locally) is kept for gap-filling, e.g. native pi-format trajectories or licence-clean extra data.
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

## Data spec (2026-11-05)

Supersedes §"Data extraction" and §R4 where they conflict. The binding constraint is **task diversity, not volume**.

### Inventory (measured on the attended machine, 2026-11-05)

| Source | Size | Use |
|---|---|---|
| **AILANG corpus in our repos**: sunholo-data/*, motoko checkouts `~/dev/mk-*`, daneel (worktrees included, de-duplicated by content hash) | 80,415 `.ail` files scanned, **3,684 unique files, ~843k lines**. Largest: motoko 861 files / 409k lines; then packages, demos, ailang-parse, docparse, stapledons (402 / 75k), daneel, email-parse | language knowledge; needs task descriptions (see G2) |
| **Past eval results** (`eval_results/`) | 40,503 files; 24,813 AILANG (9,628 agent); 17,112 passing; **14,373 distinct passing programs, but only 114 distinct tasks** (the eval set); 95 models; **709 clean error-to-fix repairs**; ~7.7k failing runs with category and stderr; 18k without a version stamp | language and repair data from **training-split benchmarks only** |
| **Agent trajectories** | motoko session JSONL (~3,200 files); opencode.db (1.7 GB); chains/observatory (3,804 chains); Claude Code sessions in AILANG repos (world alone: 861 sessions / 1.1 GB) | agent protocol (write, run, fix) and error recovery |
| **Cloud executors** (pi, claude and codex fleet tasks on the cloud plane) | **not yet counted**; in the prod observatory and chain store. pi on the rig kept no transcript for rejected runs (filed), so check whether cloud pi sessions are banked in full | real-task trajectories with outcomes; probably the best trajectory source |

### Rules

1. **Held-out split first.** The eval set is frozen before any extraction: the 24-benchmark agent subset in §Phase 0 (12 core + 12 stretch) is the held-out set for both modes. Base-model numbers on it already exist from Phase 0, so the baseline needs no re-run. Nothing from those 24 tasks enters training. That includes other models' solutions, near-duplicate tasks, and repo code that is a copy of a benchmark solution. Check exact and normalised-code hashes and task-text similarity.
2. **Re-grade everything with the current `ailang check`** (and `ailang run` against expected stdout where one exists). Keep only what compiles today, which strips stale syntax from 18k unversioned results and old repo code.
3. **Licence: D-FT-1 = YES** (Mark, 2026-11-05). Frontier-generated code and trajectories (Anthropic and OpenAI outputs) are in scope for the internal model, alongside open-weight-generated output and human-written code.
4. **Negatives: never as targets.**
   - (a) **Error-to-fix pairs**: input = failing code + the real compiler error; target = the fix.
   - (b) **Recovery trajectories**: failed attempts kept as context; **loss only on the turns that lead to the verified pass**.
   - (c) **Preference pairs** (chosen = verified pass, rejected = typical error or chat-answer protocol miss): a stage-2 DPO run. mlx-lm's `lora` has no DPO mode, so this needs another trainer.
   - (d) **Failure clusters become AILANG fixes** under PROGRAM.md routing (a mistake every model makes is usually a language or error-message fix, not training data).
   - Training directly on failed outputs is forbidden.
5. **Tool mapping:** every trajectory is converted to **pi's tool schema** (the harness it will run in). Claude Code Edit/Write/Bash, opencode and motoko tools are mapped to pi equivalents, and unmappable turns are dropped.
6. **No teaching prompt in training inputs.** The point is a model that does not need it; a slim prompt (G4) may be included as the system turn.

### Generation (only where the inventory has gaps)

- **G1, nothing for protocol or repairs.** Existing trajectories and repairs cover them.
- **G2, task descriptions for repo code (light).** A cheap model writes a one-paragraph task for each unit of repo code (back-translation), e.g. gemma-4-26b at $0.022/$0.065 per M (cents in total).
- **G3, new-task variety (optional for pilot 1): the Python oracle.** A model writes task + Python reference; the Python output is the expected stdout; an AILANG solution is kept **only if `ailang run` matches exactly**. Solver options for ~5k verified: gemma-4-31b ≈ $75 (58% yield), qwen3.8-27b ≈ $250 (94% yield, cost driven by reasoning tokens), frontier subscription lanes (quota; D-FT-1), or rig qwen3.8 ($0, GPU-days). **Always start with a ~50-run billed test.**
- **G4, slim prompt.** A 2–4 KB prompt with version deltas and a stdlib signature index, registered as a **non-active** prompt version (never change `active`). Also a `none` variant for the no-prompt arm.

### Pilot-1 mix (target ≈ 5k examples ≈ 5M tokens ≈ one overnight epoch on 12B)

≈ 1k agent trajectories (pi schema, loss-masked) · ≈ 1k error-to-fix pairs · ≈ 3k language examples (repo code with G2 tasks, plus eval passes from training-split benchmarks). These are estimates; the 1k/5k/20k ablation in open question 3 decides the real number.

## Weekend pilot plan (2026-11-06 → 2026-11-08, GPU granted by Mark)

### Friday prep (CPU) — status at 2026-11-05 ~21:15Z: **READY** (serving path verified)

- [x] D-FT-1 = YES (Mark, 2026-11-05).
- [x] Held-out set `tools/finetune/heldout.json`; arm-A agent baseline in `tools/finetune/baseline_armA.json`.
- [x] **Language data**: `tools/finetune/extract_lang.py` → 8,241 programs that compile today (repo 1,948 / 510k lines; eval 6,293 from 86 training benchmarks).
- [x] **Trajectory data**: `tools/finetune/extract_traj.py` → 23,740 samples in pi's tool schema from ~7,860 passing episodes.
  - By source: Claude Code 616 episodes · motoko 1,300 · opencode 5,907 · pi 41.
  - 3,303 of the samples are **fix turns** (error → corrected code).
  - Held-out guard applied; cloud chains (1,115, 397 substantive) **not pulled**: `chains chat --remote` returned no stages for the sampled chain. That is a follow-up.
- [x] **G2 (task text for repo code)** replaced for pilot 1 by each module's header comment (free; 536 repo samples). Model back-translation is deferred.
- [x] **Mix** `tools/finetune/build_mix.py` → `~/models/finetune/mix-pilot1`: 2,183 train / 52 valid, **4.79M tokens (1.62M target)**. Contents: 428 fix turns, 610 other trajectory turns, 604 eval programs, 536 repo modules.
- [x] **Mask bug found and fixed** (`tools/finetune/train.py`). Gemma 4's template adds an empty thinking block to the generation prompt that a finished turn lacks, so stock `--mask-prompt` hid the first 4 target tokens (including `<|tool_call>`) on 1,497 / 2,193 samples. The wrapper trains on exactly the inference-time prompt plus the target. Gate `tools/finetune/check_mix.py` passes with 0 render errors; 5 over-length samples dropped.
- [x] **Dry run** of `train.py` + `tools/finetune/pilot1.yaml` on the real mix: 4 steps OK, 21.9M trainable params (0.18%), 100–190 tok/s, so **one epoch ≈ 9–11 h**.
- [x] **Agent mode honours `--prompt-version`** (it hard-coded `""`; fixed + tests, `12fadb210`). Frozen non-active prompts `ft-slim-v1` (2.4 KB traps card) and `ft-none`. Private binary `~/models/finetune/bin/ailang-ft` (v0.57.4-17) embeds both.
- [x] Lanes `ollama-gemma4-12b-ft1` / `pi-gemma4-12b-ft1` and the pi model id; budget-cap fix live.
- [x] **Runner** `tools/finetune/weekend_pilot.sh` (resumable stages: preflight → train → serve → B_std, C_std, B_agent, C_agent, A_std → summary) and `tools/finetune/summarize_pilot.py` (kill criterion). Every arm is guarded against the prompt loader's silent fallback to the full prompt.
- [x] **Ollama serving path verified** (2026-11-05, toy adapter). After `mlx_lm.fuse --dequantize`, **dropping `audio_config` (strip level 1) is enough**: `ollama create -q mxfp8` imports it at 12 GB (`q8_0` is unsupported on this path; the supported types are int4/int8/nvfp4/mxfp4/mxfp8), it loads, and it generates AILANG with no teaching prompt. The runner now uses mxfp8, the same quantization as arm A. **Speed confound:** the imported fine-tune has no MTP draft model (the library `gemma4:12b-mxfp8` does), so it decoded at ~34 tok/s against ~46. Compare speed on **output tokens per pass** (draft-independent) and report wall time with this caveat.
- [ ] Not in pilot 1 (follow-ups): error→fix pairs from the ~1,400 stale `++` programs; 681 stdlib-skew repo rejects; cloud-chain transcripts; Python-oracle task generation (G3).

### Start procedure (Friday night, attended)

```bash
# 1. hold the GPU consumers (hold file + bootout; a bootout alone is undone by rig-watchdog in 60 s)
mkdir -p ~/.ailang/state/launchd-hold
for j in dev.ailang.os-rotation-filler dev.ailang.nightly-eval; do
  echo "finetune pilot-1 weekend (Mark 2026-11-05); release Sunday night" > ~/.ailang/state/launchd-hold/$j
  launchctl bootout gui/$(id -u)/$j 2>/dev/null || true
done
# 2. run (resumable; re-run the same command after any interruption)
tools/finetune/weekend_pilot.sh            # log: ~/models/finetune/pilot1/weekend.log
# 3. release (Sunday)
rm ~/.ailang/state/launchd-hold/dev.ailang.{os-rotation-filler,nightly-eval}
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/dev.ailang.os-rotation-filler.plist
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/dev.ailang.nightly-eval.plist
```

Daneel's intake defers while the lock is held (training does not yield; mail queues).

### Friday night → Saturday (GPU)

- [ ] **Take the GPU with the hold protocol** (hold file + bootout for `dev.ailang.os-rotation-filler` and `dev.ailang.nightly-eval`; a bootout alone is undone by rig-watchdog in 60 s). Training does not pass through eval-suite checkpoints, so **Daneel's intake defers while we hold the lock**; mail queues, and nothing is lost.
- [ ] **12B LoRA pilot**: ~5k examples, 1 epoch first. LoRA on all linear layers in the top 16 blocks, `--mask-prompt`, checkpoint every ~10% of steps. Estimated ~6–13 h. Watch val loss; stop early on divergence.
- [ ] Fuse, then `ollama import` (q8), then a smoke generation.

### Saturday → Sunday (GPU): the A/B on the held-out 24

| Arm | Model | Prompt | Modes |
|---|---|---|---|
| A | base 12B | full teaching prompt | agent ×1 **exists** (`tools/finetune/baseline_armA.json`: 11/24, 12 protocol misses). **Standard ×3 must be re-run** on the 24: Phase 0 covered only 30 runs on 22 of them, with 4 runs hitting the 300 s client timeout. Same binary and the 1800 s timeout as B/C (~2.5 h GPU) |
| B | ft1 | `ft-slim-v1` | standard ×3, agent ×1 |
| C | ft1 | `ft-none` | standard ×3, agent ×1 |

Metrics: pass rate, `first_attempt_pass_rate`, **protocol-miss rate** (no tool call / template-only), **output tokens per pass**, **wall time per verified success**; compared with `ailang eval-paired` and placed with `eval-elo`. Run time ≈ 3–5 h standard + 5–8 h agent.

**Kill criterion (unchanged):** arm B must at least match arm A on pass rate. Stretch goal: halve the protocol-miss rate and output tokens per pass.

### Sunday (optional, if time and the result is positive)

- [ ] 31B pilot on the same data (~2.5× the training time).
- [ ] Batching re-measure at fine-tuned context length (5–15k instead of 33k).
- [ ] **Release the holds** (remove the hold files, re-bootstrap the jobs), record results here, and route the failure clusters (rule 4d).

## Pilot-2 direction (2026-11-06, attended with Mark, pilot-1 still running)

**Reframe (Mark).** The model is for **agent mode**. Standard one-shot is a prompt problem, not
the target. The language was designed so an AI can *discover* it (`check` errors that name the fix,
`docs`, `examples`, `iface`). So the fine-tune should mainly instill **habits**, and memorise only the
core syntax. Habits survive language changes; memorised stdlib names go stale.

**Early pilot-1 signal (partial, standard mode only):** B_std 0/15, C_std 1/26.
- Both arms hit their wall-clock caps. Six of the 15 B runs used the full 32k output tokens on
  repetition loops.
- The runs that ended normally mostly failed on invented stdlib names, wrong module names and
  statement-style `x = …`.
- `${expr}` interpolation is valid AILANG (verified with `ailang run`). An earlier read that called
  it invalid was wrong.
- The fused model in Ollama has **no sampling defaults**; the base ships temperature 1 / top_k 64 /
  top_p 0.95. Fix that before reading anything into the loops.

### Habits to instill (agent mode)

1. **Write through the tool, never answer in chat.** This was base 12B's main failure: 12 of its
   13 agent failures were protocol misses.
2. **Verify every write.** Write → `ailang check` → read the error → apply the fix it names → run
   → compare stdout with the spec.
3. **Look up before guessing.** Unsure of a stdlib name or signature: `ailang docs std/X`,
   `ailang iface`, `ailang builtins`. Unsure of a feature: `ailang examples search <concept>`.
   Use targeted lookups, not a dump of the whole teaching prompt; every lookup grows the context,
   and 12B decode drops to ~51 tok/s at 33k.
4. **Trust the compiler over memory.** When memorised syntax conflicts with an error message,
   follow the message. This is the habit that makes an older fine-tune survive a newer language.
5. **Stop when verified.** Don't re-verify in a loop or keep generating after the program passes
   (see the pilot-1 runaways).

Anti-habits to keep out of the mix: filesystem search loops (`find /`, `ls -R`), grepping the repo
for syntax, re-reading the full prompt each turn.

### Data changes for pilot-2

| Change | Why | Measured gap |
|---|---|---|
| Take `examples/` from the main checkout first, weight ×2–3, drop the ones that fail `check` | Canonical one-feature-per-file syntax; kept current by `verify-examples`; not benchmark solutions | 481 on disk, ~397 pass `check`, **25** reached the repo pool and **1** reached the pilot-1 mix. Cause (corrected 2026-11-06): `extract_lang.py` checked files by **absolute path**, so every path-named module failed MOD010. Fixed (relative paths, main-checkout copy first): 27 → 466 kept on examples alone. The repo corpus as a whole was hit too, so pilot-2 re-extracts. 63 new examples added (D-FT-5), stdlib coverage 400 → 469 of 496 functions. |
| Up-weight trajectory turns that follow a lookup (`docs`, `examples`, `iface`, `builtins`) or a `check` error | Teaches habits 2–4 | 2,345 of 23,740 trajectory samples contain a lookup call; the mix builder doesn't favour them now |
| **Staleness pairs:** old code + the current error + the fixed version | Teaches habit 4 directly | Eval rejects are mostly `++` on strings (≈1,400) plus the 681 stdlib-skew repo rejects; the error text already names the fix |
| All 6,293 eval passes, lower per-benchmark repeat weight | More syntax signal (standard pairs still help agent writes) | Pilot-1 capped them at 10 per benchmark → 585 samples |
| Give repo programs the `benchmark/solution` module name only where the task requires it | Pilot-1 outputs carried repo-style module names | — |

Recipe: 2 epochs, LoRA on all layers, sampling defaults set at import. Add a **base 12B + slim
prompt** arm, so "the fine-tune didn't help" can be told apart from "the slim prompt is too thin".

### Examples coverage (feeds back into the language)

The fine-tune only learns what `examples/` shows. Fine-tuning therefore doubles as a check on
whether the examples cover the language. Measured 2026-11-06:
- 50 of the 51 `std/` modules are imported by at least one example. `std/trace` has none.
- 23 modules have only 1–2 examples: `cognition`, `crypto`, `datetime`, `debug`, `deflate`, `dom`,
  `embedding`, `extension`, `game`, `gzip`, `html`, `iter`, `jwt`, `package`, `sharedindex`,
  `sharedmem`, `tar`, `web`, `ai/streaming`, `simhash`, `terminal`, `yaml`, `smoke`.

  Treat these as a docs backlog.
- Coverage is by module import, not by function. A per-function coverage audit
  (`ailang iface` × example call sites) is the next measurement.

### Cadence: a weekend fine-tune per language release

The pipeline is reproducible end to end: extract → mix → gate → train (~6.5 h for 2,183 samples on
12B) → serve → held-out A/B. So a fine-tune can be rebuilt each weekend as the language moves.

- **Naming:** `gemma4-12b-ailang-<lang-version>`; the adapter is 87 MB.
- **Promotion gate:** the new fine-tune must match or beat the previous one on the held-out agent
  arm, and must beat base + full prompt.
- **Staleness check:** run the previous fine-tune against the new language. Habits 2–4 should make
  that decay slow; if it doesn't, the habit mix is wrong.

**D-FT-2 = YES (Mark, attended, 2026-11-06):** fine-tunes are published outside the rig,
through an artifact bucket linked from the website. The terms-of-use question was raised before
the decision. Confirm the base model's licence (Apache 2.0) against the model card at publish time.

**Decisions recorded 2026-11-06 (Mark):**
- **D-FT-3:** agent mode is the target; standard one-shot is a prompt issue.
- **D-FT-4:** train habits; memorise only the core.
- **D-FT-5:** expand `examples/` into a representative training set (also needed for the docs).
- **D-FT-2:** publish (above).

**Website:** a public research section, `docs/docs/finetuning/`, records measurements and
decisions as content: overview, decision log, baseline, data, pilot-1, costs. Every future
decision gets a row in `docs/docs/finetuning/decisions.md` when it is made.

### Pilot-1 agent arms void; gate fixed; re-run (2026-11-06, D-FT-6, Mark)

- **Measured:** a wire capture (`tools/finetune/trace_proxy.py` + `fake_upstream.py`, scripted
  model, no GPU) showed the pi session-protocol gate armed in every eval. It refused
  write/edit/mutating bash until `session_protocol_ack`.
- **Effect on base 12B:** every Phase 0 pass acked first.
- **Effect on the fine-tune:** it never acked, so B_agent and C_agent were 0/24 with every write
  refused.
- **Fixed in `5dc3ba42b`:** `Task.EvalRun` → harness-owned `AILANG_EVAL_RUN=1`, which is unset
  for non-eval children and refused in ExtraEnv. The gate stands down on it.
- **Re-run:** `tools/finetune/pilot1_rerun.sh` runs A/B/C agent with `~/models/finetune/bin/ailang-ft2`
  (B first). Arm A pins prompt v0.16.11.
- **Also found in the capture:**
  - The teaching prompt rides in the **user** message (pi prepends it to the directive); the
    system role is pi's own coding-assistant prompt.
  - Pilot-1's training samples used a custom system prompt instead. Pilot-2 episodes must mirror
    the captured layout.
  - Every bash result carries an OpenTelemetry deprecation line; every `read` carries a
    brain-context block.

**Pilot-2 data progress (CPU, 2026-11-06):**
- **Re-extraction:** `data-v1` with the relative-path fix. 2,733 repo programs (594k lines,
  +40%) and 6,299 eval programs.
- **Trajectory cleaning** (`tools/finetune/ft_clean.py`): 12,991 / 23,740 samples carried a stale
  absolute workspace; 1,359 had one in the target.
- **Staleness pairs** (`tools/finetune/stale_fix.py`): 1,828 programs passed when banked but fail
  `ailang check` today.
  - Causes: `++` on strings 1,259; `print` without an import 208; MOD010 166; parse errors ~170.
  - gemma-4-31b fixes are kept only if the output is exact: 23/50 on the billed test, at a
    median similarity of 0.98 (minimal edits).
- **Agent-format episodes:** `tools/finetune/agenttask/` renders the exact eval task prompt for
  synthesized episodes.

### Pilot-2 build (2026-11-06; D-FT-8 thinking format, D-FT-9 lean lane)

- **Lean pi profile** `--tool-policy lean` (`8faeacb73`): read/bash/edit/write, `pi.LeanSystemPrompt`,
  no skills/extensions/context/templates. The prefix drops from 7,338 to ~910 tokens, verified end
  to end against a scripted model.
- **Episodes** (`synth_episodes.py` + `thoughts.py`): 3,011 verified episodes — 1,236 solve, 1,671
  repair chains, 104 example.
  - Habit steps: targeted `ailang docs --all-functions` lookups, `ailang examples search`, and a
    check after each write.
  - Thoughts are built from facts in the episode, never generated by another model.
- **Mix** (`build_mix2.py` → `mix-pilot2`): 1,184 train / 50 valid episodes in the lean wire format,
  thinking on.
  - 4.6M tokens, of which 1.3M are trained.
  - Median 3.7k tokens, p95 5.7k; 7 episodes over 12,288 tokens dropped rather than truncated.
- **Trainer** (`train2.py`): whole-episode samples with the loss on every assistant turn. The masked
  spans decode to thought → tool call → `<|tool_response>`.
- **Recipe** (`pilot2.yaml`, provisional): all blocks, rank 32, 2 epochs, 12k sequence length.
- **Runner** (`pilot2_run.sh`): a dry run gates the full run. Serving gets the base model's
  sampling defaults.
- **Arms** on the held-out 24, all on the lean profile:

  | Arm | Model | Prompt |
  |---|---|---|
  | A | base | v0.16.12 |
  | A2 | base | ft-slim-v1 |
  | B | ft2 | ft-slim-v1 |
  | C | ft2 | ft-none |

- **Deferred:**
  - Self-distillation (base 12B's own verified thinking), if fact-based thoughts prove too thin.
  - The last ~100 Opus staleness repairs (the pass was stopped by a memory-pressure reaper at
    826 of ~930; 773 fixed).

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
6. **(2026-11-05) Licensing ruling: RESOLVED, D-FT-1 = yes** (Mark, 2026-11-05). Closed-model outputs may enter the training set for the internal model.

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
