# Model Watch — staying current on candidates for the rig

> **Why this file exists.** The assistant's training data is stale (months
> behind) and model *names* are misleading (e.g. `Qwen3-Coder-30B` is **older**
> than `qwen3.5`). So "what's the latest good model?" must NEVER be answered from
> memory. Always **fetch the canonical sources below live**, then verify each
> candidate's *release date* and *shape* before recommending it.

## Canonical sources (fetch these, in this order)

1. **Ollama library** — https://ollama.com/library
   Ground truth for what's *pullable on-device*, with sizes/quants. If it's not
   here (or on HF in GGUF/MLX), it can't run on the rig.
2. **OpenRouter models** — https://openrouter.ai/models
   What can be *quality-screened* via OR before pulling, with pricing. Filter by
   "newest". Many models land here before/without an Ollama build.
3. **Aider polyglot leaderboard** — https://aider.chat/docs/leaderboards/
   Coding-specific, includes open-weight models with $/task. (Note: has its own
   refresh lag — cross-check dates.)
4. **SWE-bench (Verified / Pro)** — https://www.swebench.com/
   Agentic coding ground truth. Good for separating "coder" tuning from base.
5. **Hugging Face trending** — https://huggingface.co/models?pipeline_tag=text-generation&sort=trending
   Earliest signal of brand-new releases.
6. **The rig's own OS leaderboard** — `docs/static/benchmarks/os/latest.json`
   The only source that measures AILANG specifically. Beats any external board
   for *our* question.

## The two gates every candidate must pass

- **Newer & better than the incumbent** — confirm by *release date*, not name
  (Qwen lesson). The current rig incumbent is **`qwen3.5:35b-a3b`** (Feb 2026).
  A candidate must post-date and out-score it on something we care about.
- **Fits the hardware shape** — MoE small-active, on-disk ≤ ~45 GB, active ≤ ~8B.
  Run `scripts/check_model_fit.sh`. Dense > ~14B is too slow on this bandwidth-
  bound box.

## Candidate shortlist — scanned 2026-06-14 (re-verify before pulling)

On-device-feasible (the ones worth actually testing on the rig):

| Candidate | Shape | ~Disk | Status vs incumbent | Notes |
|---|---|---|---|---|
| **Qwen 3.6 35B-A3B** | MoE 35B / 3B | ~24 GB | **Newer** (Apr 2026) than qwen3.5 | Same shape, one gen up — the natural incumbent upgrade |
| **Devstral Small 2** | ~24B coder | ~24 GB | Different lineage (Mistral) | 68% SWE-bench Verified, Apache-2.0; diversifies beyond Qwen |
| **Gemma 4 31B** | MoE | ~20 GB | Sibling of the 26b we run | 256K ctx, Apache-2.0; compare vs our gemma4:26b |

Out of scope for on-device (keep as **OpenRouter** ceiling references only — too
big to run on the rig): **GLM-5.1**, **DeepSeek-V4-Pro**, **Kimi K2.6**,
**MiniMax-m3**, **MiMo-V2.5-Pro** (1T/42B-active). These are the frontier-OS
comparison anchors, not rotation members.

> **Naming traps seen so far:** `Qwen3-Coder-30B-A3B` = Qwen3.0 coder (2025),
> *older* than qwen3.5 despite the "Coder" label. `Qwen 3.6-27B` is newer but
> **dense** (no A3B) → slower on this box. `Qwen 3.7-Max` is API-only.

## Refresh cadence

Re-scan **per release** (or monthly): fetch sources 1–5, update the table above
with date + score + shape, drop anything superseded, and run the onboarding
workflow on any candidate that clears both gates. Update the "scanned" date.
