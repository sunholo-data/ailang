# Model Watch — staying current on candidates for the rig

> **Why this file exists.** The assistant's training data is stale (months
> behind) and model *names* are misleading (e.g. `Qwen3-Coder-30B` is **older**
> than `qwen3.5`). So "what's the latest good model?" must NEVER be answered from
> memory. Always **fetch the canonical sources below live**, then verify each
> candidate's *release date* and *shape* before recommending it.

## Canonical sources (fetch these, in this order)

> The source list is **user-curated** — the assistant is too stale to know which
> external boards are still maintained. **Dropped: Aider polyglot leaderboard
> (unmaintained as of 2026-06).** Do not re-add it.

**Quality / ranking (user-confirmed, 2026-06):**
1. **SWE-bench (Verified / Pro)** — https://www.swebench.com/
   Agentic coding ground truth. Good for separating "coder" tuning from base.
2. **Terminal-Bench** — https://www.tbench.ai/leaderboard
   Real-terminal agentic tasks. Versioned (2.0, 2.1) — scores are **not**
   comparable across versions; note which version a number came from.
3. **LMArena** — https://lmarena.ai/leaderboard
   Crowd ranking; use the coding category.

**Availability (what can actually run / be screened):**
4. **Ollama library** — https://ollama.com/library
   Ground truth for what's *pullable on-device*, with sizes/quants. If it's not
   here (or on HF in GGUF/MLX), it can't run on the rig.
5. **OpenRouter models** — https://openrouter.ai/models
   What can be *quality-screened* via OR before pulling, with pricing. Filter "newest".
6. **Hugging Face trending** — https://huggingface.co/models?pipeline_tag=text-generation&sort=trending
   Earliest signal of brand-new releases (GGUF/MLX availability = on-device).

**Ours (always valid — the only board that measures AILANG):**
7. **The rig's own OS leaderboard** — `docs/static/benchmarks/os/latest.json`
   Beats any external board for *our* question. The OpenRouter quality-screen
   (skill step 2) is how a new candidate earns a row here.

## The point of the scan: split every promising model into two tracks

For each model that ranks well on the boards (1–3), classify it — this is what
the watch is *for*:

- **🖥 LOCAL track (the rig):** does it have an on-device build (Ollama / HF
  GGUF/MLX) that passes the shape gate (MoE small-active, on-disk ≤ ~45 GB)? →
  it's a rotation candidate. Run `check_model_fit.sh`, then onboard.
- **☁️ CLOUD track (OpenRouter):** is it on OpenRouter? → add it as an `or-*`
  quality-screen / ceiling reference. Big frontier-OS models (GLM-5.1,
  DeepSeek-V4-Pro, Kimi, MiniMax, MiMo) live **only** here — too big for the rig.

Many strong models qualify for BOTH (e.g. a Qwen MoE: screen the hosted version
on OpenRouter, run the quantized version on-device) — that pairing is exactly
what the opencode-vs-pi-vs-OR comparison in the Explorer measures.

## The two gates every candidate must pass

- **Newer & better than the incumbent** — confirm by *release date*, not name
  (Qwen lesson). The current rig incumbent is **`qwen3.5:35b-a3b`** (Feb 2026).
  A candidate must post-date and out-score it on something we care about.
- **Fits the hardware shape** — MoE small-active, on-disk ≤ ~45 GB, active ≤ ~8B.
  Run `scripts/check_model_fit.sh`. Dense > ~14B is too slow on this bandwidth-
  bound box.

## Candidate shortlist — scanned 2026-06-14 (re-verify before pulling)

Incumbent on the rig: **`qwen3.5:35b-a3b`** — scores **0.405** on Terminal-Bench 2.0.

### 🖥 LOCAL track — on-device candidates (worth pulling/screening)

| Candidate | Shape | ~Disk | Evidence | Ollama | Priority |
|---|---|---|---|---|---|
| **Qwen3.6-35B-A3B** | MoE 35B/3B | ~24 GB | **TB2.0 0.515 vs incumbent 0.405 (+11)**; Apr 2026 | yes (1wk) | **#1 — proven upgrade, same shape** |
| **GLM-4.7-Flash** | ~30B class | ~18–20 GB | "strongest in 30B class"; new (1wk); diff lineage | yes (1wk) | #2 — screen it (verify MoE/active) |
| **Laguna-XS.2** | MoE 33B/3B | ~20 GB | "agentic coding + long-horizon"; new (1mo) | yes (1mo) | #3 — right shape, needs our screen |
| **Nemotron-Cascade-2** | MoE 30B/3B | ~18 GB | reasoning+agentic; NVIDIA (2mo) | yes (2mo) | #3 — right shape, needs our screen |
| **Devstral-Small-2** | dense 24B coder | ~24 GB | 68% SWE-bench Verified; Apache-2.0 (Mistral) | yes | #4 — lineage diversity; dense (slower) |
| Qwen3.6-27B | dense 27B | ~17 GB | 77.2% SWE-bench but **dense** → slower here | yes | low (dense penalty on this box) |
| Granite4.1-30B | 30B | ~18 GB | IBM; RAG/tool-use focus | yes | low (not coding-led) |

Notes: external coding scores are thin for the new MoE entrants (GLM-4.7-Flash,
Laguna, Nemotron-Cascade) — that's exactly what our OpenRouter screen + rig
rotation is for. The Qwen3.6 upgrade is the only one with a hard external number
beating the incumbent today.

### ☁️ CLOUD track — OpenRouter screen / ceiling refs (too big for the rig)

| Model | Size | Evidence | Status |
|---|---|---|---|
| DeepSeek-V4-Pro-Max | 1.6T | SWE-bench Verified **80.6%** (top OSS) | already `or-deepseek-v4-pro` |
| MiniMax M3 | — | SWE 80.5% | already `or-minimax-m3` |
| GLM-5.1 | 744B | TB2.0 **0.690** (top OSS) | already `or-glm-5-1` |
| DeepSeek-V4-Flash-Max | 284B | TB2.0 0.569 | already `or-deepseek-v4-flash` |
| Qwen3.7-Max | — | SWE 80.4% (API-only/proprietary) | optional add |

The cloud ceiling is already well-covered by the existing `or-*` suite; the new
value this scan surfaces is the **local track**.

> **Naming traps seen so far:** `Qwen3-Coder-30B-A3B` = Qwen3.0 coder (2025),
> *older* than `qwen3.5` despite the "Coder" label (and TB2.0 0.375 on the 480B
> coder — coder-line is not winning). `Qwen 3.6-27B` is newer but **dense** (no
> A3B) → slower on this box. `Qwen 3.7-Max` is API-only. Always check release
> date + MoE-vs-dense, not the name.

> **Screen availability (checked 2026-06-14 via OpenRouter API):** screenable on
> OR *with a matching on-device build* = Qwen3.6-35B-A3B (`qwen/qwen3.6-35b-a3b`),
> GLM-4.7-Flash (`z-ai/glm-4.7-flash`), Laguna-XS.2 (`poolside/laguna-xs.2:free`).
> **NOT screenable:** Devstral-Small-2 — OR only hosts `mistralai/devstral-2512`
> which is the **123B** Devstral 2 (too big for the rig), not the 24B Small-2;
> and Nemotron-Cascade-2 is not on OR at all. Those two can only be evaluated by a
> *direct local pull* (no OR pre-screen possible) — decide pull-or-drop per case.

> **Data caveat:** TB2.0 numbers via the llm-stats aggregator, SWE via search
> summaries — directionally reliable (3.6 > 3.5 same-shape is consistent), but
> re-confirm exact figures on tbench.ai / swebench.com before committing.

## Refresh cadence

Re-scan **per release** (or monthly): fetch sources 1–5, update the table above
with date + score + shape, drop anything superseded, and run the onboarding
workflow on any candidate that clears both gates. Update the "scanned" date.
