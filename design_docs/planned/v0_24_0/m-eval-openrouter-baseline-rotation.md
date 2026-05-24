# M-EVAL-OPENROUTER-BASELINE-ROTATION: Cloud Baseline Three Candidate Models

**Status**: Planned — **awaiting user approval before launch** (each cloud rotation = ~$1-5 OR credit spend)
**Target**: v0.24.0 (or sooner, opportunistic)
**Priority**: P1 (Medium — informs the next 2-4 weeks of local-model selection work)
**Estimated**: 1 day of wall-clock, ~$5-20 of OpenRouter credits, ~2h of human review at the end
**Dependencies**: M-EVAL-LOCAL-OLLAMA (this doc) provides the operational ground; runbook research already identified the 3 candidates with full hardware-fit justification.

## Problem Statement

The runbook's "Vetted candidates ready for OpenRouter baseline" section names 3 models that satisfy the 128 GB Mac Studio hardware envelope AND are dual-availability (OpenRouter + ollama). We have not actually run them yet on AILANG benchmarks. Without a cloud baseline, we don't know whether a local pull is worth the disk/setup cost.

The weekend run made one thing clear: **gemma4:26b has a model-capability ceiling** that compiler error-quality work *can* lift, but only so far. The current 80% best-case is not the language ceiling — it's this model's ceiling on the rig. We need to know what the next floor up looks like before investing more local-rig effort tuned to gemma4.

**Current State:**
- gemma4:26b-ailang: validated, 70-82% pass rate on smoke tier (N=3)
- 3 candidates identified per hardware-fit analysis, none yet evaluated on AILANG
- `models.yml` already has the OpenRouter scaffolding (`opencode-or-*` pattern); adding the 3 candidates is `~10 lines of YAML each`
- `opencode-or-gemma-4-26b` already exists as the cloud-control for direct cloud-vs-local diff

**Impact:**
- Without baselines, we either guess (waste local-rig time on a model that won't beat gemma4) or pull blindly (200 GB of model downloads, 4 hours of setup per false start)
- Each cloud baseline is ~$1-5; total spend cap easy to bound

## Goals

**Primary Goal:** Establish a cloud baseline pass rate (with documented sampling settings) for each of 3 candidate models AND the current gemma4 control, on the same 17-benchmark smoke tier, in <1 day of wall-clock and <$25 of OpenRouter spend.

**Success Metrics:**
- 4 leaderboard pages produced (3 candidates + 1 gemma4 cloud control) at `eval_results/rotation/2026-05-25/<model>_cloud_baseline_n3/`
- For each model: pass rate, mean agent_turns, mean wall-clock, first-attempt rate, distinct error codes triggered
- A decision table at the end recommending: pull-to-local | trial-once-more | discard
- All 3 OR entries committed to `models.yml` with full pricing metadata (so future runs are reproducible)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Run all 4 in one batch or staged | Staged stops early on a clear loser; batched ships faster | human | design | low |
| N=1 or N=3 per benchmark for cloud baseline | N=1 is $5 vs $15 but high variance; N=3 catches single-trial luck | human | design | low |
| Whether to include `nemotron-3-nano` (free tier) at higher trial count since it's free | Free tier means N=5 or N=10 is feasible and informative | agent | design | low |
| Whether to also include `opencode-or-gemma-4-26b` to refresh the cloud control | Fresh control eliminates cross-day drift questions | agent | design | low |

### Design Freeze

- [ ] **Trial count strategy approved**: recommend N=3 for paid models, N=5 for free-tier nemotron-3-nano. Total budget ~$15-20.
- [ ] **User authorizes OpenRouter spend** of up to $25 for this rotation
- [ ] **Model list confirmed**: qwen3-coder-next, nemotron-3-nano, gpt-oss-120b, gemma-4-26b (control). Drop or swap before launch.

## Solution Design

### Overview

Three new entries in `internal/eval_harness/models.yml`, then run `make eval-smoke` four times (once per model) into dated rotation directories. Compare results with the existing leaderboard generator + a short decision-table writeup at the end.

### Architecture

```
internal/eval_harness/models.yml
  + opencode-or-qwen3-coder-next        ← NEW
  + opencode-or-nemotron-3-nano          ← NEW (free tier)
  + opencode-or-gpt-oss-120b             ← NEW
  + opencode-or-gemma-4-26b              ← already exists; use as-is

                ↓ make eval-smoke MODELS=opencode-or-<model> ...

eval_results/rotation/2026-05-25/
  ├ qwen3_coder_next_cloud_n3/
  ├ nemotron_3_nano_cloud_n5/        ← higher N because free
  ├ gpt_oss_120b_cloud_n3/
  └ gemma_4_26b_cloud_control_n3/    ← refreshes the existing cloud control

                ↓ leaderboard generator (existing)

docs/docs/reference/os-model-leaderboard/
  + 2026-05-25-qwen3-coder-next-cloud-n3.md
  + 2026-05-25-nemotron-3-nano-cloud-n5.md
  + 2026-05-25-gpt-oss-120b-cloud-n3.md
  + 2026-05-25-gemma-4-26b-cloud-control-n3.md

                ↓ analyst writeup (this milestone's deliverable)

design_docs/implemented/v0_24_x/openrouter-baseline-rotation-2026-05-25.md
  - Decision table: pull-to-local | trial-once-more | discard
  - Per-model cost/benefit summary
  - Notes on observed failure modes
```

### Implementation Plan

**Step 1: Add 3 models to models.yml** (~30 min)
- [ ] `opencode-or-qwen3-coder-next` — slug `qwen/qwen3-coder-next`, pricing TBD from OR
- [ ] `opencode-or-nemotron-3-nano` — slug `nvidia/nemotron-3-nano:free`, pricing $0
- [ ] `opencode-or-gpt-oss-120b` — slug `openai/gpt-oss-120b`, pricing TBD from OR
- [ ] Verify each entry's `agent_model_name` is the opencode-format (`<provider>/<model>`)
- [ ] Commit: `models.yml: add 3 OR baseline candidates for hardware-fit eval`

**Step 2: Cost sanity check** (~10 min)
- [ ] Look up live OR pricing for each (`gh api` or just visit the page); update YAML
- [ ] Compute worst-case: 17 benchmarks × N trials × est input/output tokens × per-1k rate
- [ ] Reject any model whose worst-case run exceeds $10 unless approved

**Step 3: Run 4 cloud rotations** (~2-3 hours wall, sequential to avoid OR rate limits)
- [ ] `make eval-smoke MODELS=opencode-or-qwen3-coder-next EXTRA="-trials 3 -parallel 4 -agent-timeout 600 -langs ailang -output eval_results/rotation/2026-05-25/qwen3_coder_next_cloud_n3"`
- [ ] Same for nemotron-3-nano with `-trials 5`
- [ ] Same for gpt-oss-120b with `-trials 3`
- [ ] Same for gemma-4-26b with `-trials 3` (refresh control)
- [ ] All can run with `-parallel 4` (OR has no single-GPU constraint)

**Step 4: Generate leaderboards** (~30 min)
- [ ] Use the existing leaderboard generator on each rotation directory
- [ ] Copy outputs to `docs/docs/reference/os-model-leaderboard/`
- [ ] Side-by-side compare to gemma4:26b local

**Step 5: Decision writeup** (~1-2 hours)
- [ ] Create `design_docs/implemented/v0_24_x/openrouter-baseline-rotation-2026-05-25.md`
- [ ] Per-model: pass rate, error codes seen, mean agent_turns, distinct failure modes
- [ ] Decision table: pull-to-local (above gemma4 floor) | trial-once-more (close to floor, run N=10) | discard (below floor or refused tools)
- [ ] Send results to user in inbox message (`ailang messages send user`)

### Files to Modify/Create

**Modified files:**
- `internal/eval_harness/models.yml` — 3 new entries, ~40 lines added

**New files:**
- `eval_results/rotation/2026-05-25/<model>_*/` — 4 dirs of trial JSONs (auto-generated)
- `docs/docs/reference/os-model-leaderboard/2026-05-25-*-cloud-*.md` — 4 leaderboard pages
- `design_docs/implemented/v0_24_x/openrouter-baseline-rotation-2026-05-25.md` — decision writeup, ~200 lines

## Examples

### Example 1: New models.yml entry

```yaml
opencode-or-qwen3-coder-next:
  api_name: "qwen/qwen3-coder-next"
  provider: "openrouter"
  description: "Qwen3-Coder-Next (80B/3B MoE, 26.7x sparsity) via opencode — hardware-fit baseline for local pull decision"
  env_var: "OPENROUTER_API_KEY"
  agent_cli: "opencode"
  agent_model_name: "openrouter/qwen/qwen3-coder-next"
  model_family: "qwen3-coder-next"
  max_output_tokens: 8192
  pricing:
    input_per_1k: <TBD>
    output_per_1k: <TBD>
  notes: |
    Hardware-fit candidate per M-EVAL-LOCAL-OLLAMA runbook. Highest sparsity
    in our short list (active=3B). If passes cloud baseline above gemma4-26b
    local floor, pull to ollama as `qwen3-coder-next` and slot into local rig.
```

### Example 2: Decision table format (target output)

```markdown
## Decision Table — 2026-05-25 OpenRouter Baseline

| Model | Pass rate (cloud N=3) | Mean turns | $ for full rotation | Pull-to-local recommendation |
|---|---|---|---|---|
| gemma-4-26b (cloud control) | 70-82% | 6-12 | $1.20 | n/a (already local) |
| qwen3-coder-next | TBD | TBD | TBD | TBD |
| nemotron-3-nano | TBD | TBD | $0 | TBD |
| gpt-oss-120b | TBD | TBD | TBD | TBD |

**Recommendation**: pull X to local. Continue iterating on gemma4 in parallel.
```

## Success Criteria

- [ ] 3 new entries in `models.yml`, with pricing populated
- [ ] 4 successful cloud rotations completed (trial JSON files present, no `api_error` outcomes)
- [ ] 4 leaderboard pages generated
- [ ] Decision writeup published with explicit recommendation per model
- [ ] Total OR spend within $25 budget (verified after run via OR dashboard)
- [ ] Inbox message to user with the decision table

## Testing Strategy

**Unit tests:**
- None required (no code changes — just config + script runs)

**Integration tests:**
- Smoke-trial each model with `-trials 1 -benchmarks fizzbuzz` before committing to full N=3 rotation — catches missing API keys, bad slugs, opencode provider issues at $0.05 cost instead of $5

**Manual testing:**
- Verify each rotation completes without `api_error` rate or timeout cluster
- Spot-check 1 solution.ail per model to confirm reasonable output (not e.g. JSON wrapped, not refusal)

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Whether to run the 3 candidates concurrently (4 OR parallel calls each across 4 models = up to 16 parallel)** — agent may choose; OR rate limits and per-key cost are the constraints
- **Whether to use the AILANG v0.10.0-slim seed prompt or v0.16.0 full prompt** — agent should default to the same prompt used for the local rig so results are comparable; document choice
- **Whether to skip nemotron-3-nano if it consistently refuses or returns malformed responses** — agent may abort that model's rotation after 5 failed trials and document why
- **Whether to refresh the gemma-4-26b control if the previous run is <7 days old** — agent may use existing data if available and recent

## Non-Goals

**Not attempted in this milestone:**
- **Pulling any model to local ollama** — that's a follow-up triggered by the decision table
- **Building an AILANG-tuned Modelfile variant for any candidate** — only happens after pull-to-local decision
- **Adding the candidates to the leaderboard's "default rotation" set** — that's a v0.25 decision based on these results
- **Running the candidates through motoko-agent** — opencode is the rig we've stabilized; motoko parity is a separate question

## Timeline

**Single-day plan:**
- 09:00 — Step 1 + 2 (models.yml entries + price sanity)
- 10:00 — Step 3 begins (4 sequential rotations)
- ~14:00 — All rotations complete
- 14:00-16:00 — Steps 4 + 5 (leaderboards + writeup)
- 16:00 — Send results to user

**Total: ~7 hours wall, mostly idle compute time during cloud calls**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| OR spend exceeds $25 budget | Med | Per-step cost sanity check (Step 2); hard-stop monitor checks OR dashboard during run |
| One model has very high refusal/api_error rate, eating budget without info | Med | Smoke-trial first ($0.05); abort rotation if >50% api_error in first 5 trials |
| OR slug changes between runbook research and execution | Low | Step 2 verifies slug resolves to a live model before launch |
| Cloud baseline doesn't represent local-rig sampling | Med | Document seed prompt + sampling explicitly in models.yml notes; this is a comparison floor, not absolute claim |
| Free-tier nemotron-3-nano hits rate limits and slows down rotation | Low | Run nemotron last; if rate-limited, drop to N=3 to match others |

## Related Documents

**Operational ground:**
- [m-eval-local-ollama.md](m-eval-local-ollama.md) — Operational reliability for local rig; this rotation feeds its model-selection decisions
- [m-eval-rating-efficiency.md](m-eval-rating-efficiency.md) — ELO + selective reruns; the resulting cloud baseline data slots into ELO rankings cheaply
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` — Section "Vetted candidates ready for OpenRouter baseline" lists the 3 models with full hardware-fit justification

**Companion work:**
- [m-eval-finetuning-data-pipeline.md](m-eval-finetuning-data-pipeline.md) — The fine-tuning corpus extraction targets qwen3-coder:30b specifically; if qwen3-coder-next baselines well, may shift target
- [m-eval-metrics-taxonomy.md](m-eval-metrics-taxonomy.md) — The 4 rotation outputs should be tagged with the metrics vocabulary

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Configuration-only change; no AILANG-level determinism impact |
| A2: Replayability | +1 | Full sampling params + slug + Modelfile sha recorded per trial; cloud baseline reproducible (modulo provider drift) |
| A3: Effect Legibility | 0 | No AILANG effect surface change |
| A4: Explicit Authority | 0 | OR API key authority already explicit via env_var; no change |
| A5: Bounded Verification | 0 | Not applicable |
| A6: Safe Concurrency | 0 | OR is provider-side; our concurrency knob unchanged |
| A7: Machines First | +1 | Decision table is machine-parseable; per-trial JSON outputs feed leaderboard machinery |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | Explicit OR spend budget, per-model pricing in YAML, dashboard check during run |
| A10: Composability | 0 | n/a |
| A11: Structured Failure | +1 | Model-rejection criteria explicit (api_error rate, refusal pattern); auto-abort on threshold |
| A12: System Boundary | +1 | Cloud vs local boundary now has 4 data points to compare; previously had 1 |

**Net Score: +5** → **Decision: ✅ Proceed pending user approval of OR spend budget**

### Hard Violation Check

- [x] A1 (Determinism): No new nondeterminism
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): All output structured/parseable

### Conflict Surface

**Not applicable** — `models.yml` is a configuration file with append-only semantics; new entries cannot conflict with existing ones. No language-semantic surface touched.

## References

- [Design Axioms](/docs/references/axioms)
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md#vetted-candidates-ready-for-openrouter-baseline-2026-05-24-research` — Hardware-fit research that nominated these 3 models
- [OpenRouter model directory](https://openrouter.ai/models) — for live pricing lookup
- [Commit 1b7e06d4](commits/1b7e06d4) — Runbook section that this milestone executes

## Future Work

- **If qwen3-coder-next clears cloud baseline**: pull to ollama, build Modelfile variant, full local rig validation
- **If gpt-oss-120b clears cloud baseline AND MLX is workable**: investigate native MXFP4 + MLX path for direct decoder speed comparison
- **Quarterly cadence**: re-run this 4-model rotation each quarter to catch model improvements + identify new candidates as they ship

---

**Document created**: 2026-05-24
**Last updated**: 2026-05-24
