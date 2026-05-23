# M-EVAL-SLIM-PROMPT-SELF-DISCOVERY

**Status**: Planned
**Target**: v0.23.0
**Priority**: P1
**Estimated**: 2 days (1 day prompt + harness; 1 day A/B measurement)
**Dependencies**: None (purely additive — new prompt version, existing harness)

## Problem statement

The local-Ollama eval rig spends most of its compute on **prefill**, not generation. Verified numbers from the 2026-05-22 / 2026-05-23 sessions:

- Per-trial input tokens: **~109,000**
- Per-trial output tokens: **~1,000–3,000**
- Prefill share of the cost: **~95%** (compute-bound; bandwidth-bound for generation; see [llama.cpp #4167](https://github.com/ggml-org/llama.cpp/discussions/4167))
- Of those 109k input tokens, our AILANG agent teaching prompt contributes **~3,000**. The rest (**~106,000**) is opencode's own framework prompt: tool descriptions, permission system, session protocol, ACP plumbing.

This creates two coupled failure modes the rig hit empirically:

1. **VRAM blow-up with `OLLAMA_NUM_PARALLEL>1`**: ollama pre-allocates per-slot KV cache at the full context length, so 4 slots × 110k-token KV ≈ 25 GB on top of the 17 GB weights. Observed 43 GB VRAM allocation matches.
2. **15-minute TTFT timeouts**: the single integrated GPU cannot prefill 2–4 of 110k-token prompts in parallel. Slots compete for the same compute and the per-request prefill latency blows past the 15-min cap.

The remediation we've already settled on for the harness side ([local-ollama-eval skill](../../../.claude/skills/local-ollama-eval/SKILL.md) as of 2026-05-23) is `OLLAMA_NUM_PARALLEL=1` + `KEEP_ALIVE=-1` + serial harness execution. That makes the rig stable, but doesn't address the underlying issue: **we are forcing the model to read 109k tokens of context before it can write a single AILANG token**, on every single trial, with cold-start every benchmark.

A second, separately verified observation: the agent **doesn't use the discovery tools** the existing 13KB agent prompt instructs it to. Of 56 shell commands invoked across yesterday's successful sessions, all 56 were `ailang run benchmark/solution.ail`. Zero `ailang prompt`, zero `ailang docs std/*`, zero `ailang examples search`. The teaching prompt is treated as the complete reference — the model never needs to discover anything because everything's already in the prompt.

## Goals

**Primary**: Make the rig usable for *any* local model size we expect to host (gemma4:26b today, 70B+ tomorrow) at the cost ceiling of one model load + warm prefix cache, not N × full prefill.

**Strategic framing (added 2026-05-23 after user feedback)**: This experiment isn't a one-shot A/B — its real purpose is to **map the threshold between model-capability tiers and prompt-strategy regimes**. Different prompt strategies will work at different model intelligence levels. The longitudinal eval rig exists precisely to characterize where those boundaries are.

**The strategy axis** (refined after confirming opencode supports remote MCP servers via [`opencode.jsonc → mcp`](https://opencode.ai/docs/mcp-servers)):

| ID | Seed prompt | Discovery path | Notes |
|---|---|---|---|
| **S0** | 13KB embedded reference (v0.9.0) | None used in practice | Current baseline. Empirically: agents only invoke `ailang run`, never the discovery CLI |
| **S1** | ~500 token slim seed pointing at CLI tools | Agent shells out to `ailang docs`, `ailang examples search`, `ailang check --json`, `ailang verify --json` | Text outputs; agent must parse prose |
| **S2** | ~500 token slim seed pointing at MCP tools | Native MCP tool calls via `mcp.ailang.sunholo.com` (wired into `~/.config/opencode/opencode.jsonc`): `ailang.prompt_get`, `ailang.stdlib_search`, `ailang.examples_for_concept`, `ailang.limitations_list`, `ailang.effects_catalog` | Structured JSON returns — strictly superior to S1 for parsing |
| **S3** | Same slim seed | Both MCP (primary) + CLI shell-out (fallback if MCP unreachable) | Robustness — covers the case where mcp.ailang.sunholo.com is down |

**The model-capability matrix** to populate over future rotations:

|  | S0 embedded (13KB) | S1 slim + CLI (~500 tok) | S2 slim + MCP (~500 tok) | S3 slim + both (~500 tok) |
|---|---|---|---|---|
| Strong cloud (claude-sonnet-4.6) | already 90%+ | TBD | TBD | TBD |
| 30B+ local (qwen3-coder:30b) | TBD | TBD | TBD | TBD |
| 26B local (gemma4:26b) | currently 0/17 | **A/B candidate** | **A/B candidate** | TBD |
| 4–8B local (gemma3:4b) | TBD | likely 0 | likely 0 | TBD |

Each cell becomes a published per-release leaderboard delta. The boundary contour — "at what intelligence level does each strategy stop being viable?" — becomes a citable piece of empirical AI/language-design data we can publish.

This shifts the v0.23.0 immediate goal from "pick the best prompt" to "establish a measurement methodology for prompt-vs-model threshold mapping, and gather the first cells of data". The threshold map is the long-term artifact.

Cells fill in over future rotations. Each cell is a published per-release leaderboard delta. The boundary contour — "at what intelligence level does each strategy stop being viable?" — becomes a citable piece of empirical AI/language-design data we can publish.

This shifts the v0.23.0 immediate goal from "pick the best prompt" to "establish a measurement methodology for prompt-vs-model threshold mapping, and gather the first cell of data". The threshold map is the long-term artifact.

**Success metrics** (to measure in the A/B experiment):

| Metric | Current (v0.9.0 embedded reference) | Target (slim seed + discovery) |
|---|---|---|
| Median wall-clock per benchmark (cold trial) | ~90 s (fizzbuzz) up to ~1200 s (record_update) | ≤ +20% of current; we accept slower for stability |
| Median wall-clock per benchmark (warm trials 2+) | Unknown (cold-start every trial) | ≤ 30% of cold trial (prefix cache hit on the slim seed) |
| Pass rate on smoke tier | 9/17 ≈ 53% (best observed, 2026-05-22 01:00 cluster) | ≥ 80% of that = 7/17 minimum |
| Total tokens per trial (input) | ~109,000 | ≤ 30,000 (slim seed + discovery responses) |
| Trials completed in a 4 h overnight window | 3 trials × 17 benchmarks = 51, but only ~9/17 pass | Same 51 trials, ≥40 pass |
| Number of distinct AILANG-CLI invocations by the agent | ~1 per trial (`ailang run`) | ≥ 3 per cold trial (one of `prompt`/`docs`/`examples` + check + run) |

The primary success criterion is **"slow but steady that finishes successfully without crashing"** — the rig is unattended overnight, so a 30 % wall-clock slowdown that yields actually-usable data is a win.

## Hypothesis

**AILANG's type-check and verify outputs are educational enough that a model receiving only a slim seed prompt can recover from poor first attempts via iterative `ailang check → fix → run` loops, *especially* on weaker models where the previous embedded-reference approach didn't help (since today's gemma4:26b smoke shows 0/17 AILANG pass — embedded reference clearly isn't being internalized either).**

Two sub-hypotheses, separately testable:

1. **H1**: A 500–1,000 token seed prompt that *names* the tools (`ailang prompt --kind agent`, `ailang docs std/<module>`, `ailang examples search <concept>`) and instructs the model to call them on first AILANG error, produces ≥ 80% of the current pass rate on a smoke tier.
2. **H2**: Even if pass rate drops modestly, the per-trial wall-clock drops more — net throughput (passes per hour) is the same or better. The cold prefill goes from 110k tokens to ~5k tokens (slim seed + opencode framework cache hit), and the model recovers the rest of the context dynamically.

## Counter-evidence we should NOT pretend doesn't exist

Two pieces of prior data warn that the obvious version of this won't work:

1. **[ADR-002](../../decisions/ADR-002-pretooluse-microrag-disabled.md)** (2026-04-27): The `PreToolUse` μRAG hook on `.ail` files was disabled because embedding-similarity retrieval of syntax chunks was noisier than signal. Generic terms in the file dominated the cosine; the specific anti-pattern contributed ~1% of the vector.
2. **The v0.8.2 agent prompt's own notes**: *"Data shows 0/63 agents use runtime discovery (cat std/*.ail, ailang docs), so key stdlib functions are embedded directly."* Claude/Sonnet on cloud APIs didn't self-discover — the team chose to embed the reference because telling agents to discover was demonstrably ignored.

**Why this design might succeed despite that**:

- Both prior datapoints were measured against models that *easily* handle the 13KB embedded reference. They didn't self-discover because they didn't need to. Gemma4:26b currently produces 0/17 pass even WITH the embedded reference (cold prefill of the 13KB prompt is being processed but the model writes wrong syntax anyway). The embedded-reference approach is already failing for our actual rig workload, so the comparison baseline isn't "self-discovery vs working embedded ref" — it's "self-discovery vs broken embedded ref."
- ADR-002 was about *content-similarity retrieval on file edits*. This proposal is about *agent-initiated tool calls on need* — different mechanism. The agent does its own retrieval (it knows it doesn't know, asks the CLI), not a hook trying to guess what's needed.

## Solution sketch

### The slim seed prompt (v0.10.0 candidate, ~500 tokens target)

```markdown
You are writing AILANG (file extension: `.ail`). AILANG is a pure-functional
language; it has NO `for`, `while`, or `class`. You use recursion + pattern
matching + explicit effect rows.

**You don't know AILANG syntax yet. To discover it, run these CLI commands as
tools in your workspace:**

- `ailang prompt --kind agent` — load the full agent coding guide (~300 lines)
- `ailang docs std/<module>` — view a stdlib module's exports
- `ailang examples search "<concept>"` — find a working example file
- `ailang check <file.ail>` — type-check (errors are precise and educational)
- `ailang run --caps <CAPS> --entry main <file.ail>` — execute

**Workflow for any AILANG task:**

1. `ailang prompt --kind agent` to load the language reference.
2. If you need a specific stdlib function, `ailang docs std/list` (or similar).
3. Write `solution.ail` starting with `module benchmark/solution`.
4. `ailang check solution.ail` — read every error carefully. The errors tell
   you exactly which token is wrong and suggest fixes.
5. Fix and re-check until clean, then `ailang run`.

**Required structure** (the only thing you commit to memory upfront):

```ailang
module benchmark/solution
export func main() -> () ! {IO} {
  println("hello")  -- println is in prelude, no import needed
}
```

The task is below. **Call `ailang prompt --kind agent` first.**
```

### Harness wiring

Two new prompt versions in `cmd/ailang/prompts/agent/`:

- `v0.10.0-slim.md` — the seed above (~500 tokens)
- Existing `v0.9.0.md` — kept as the embedded-reference comparison baseline

A new flag on `eval-suite`:

```
-prompt-version v0.10.0-slim   # forces this version regardless of versions.json active
```

So an A/B run is one rotation each:

```bash
ailang eval-suite -agent -prompt-version v0.9.0 -models opencode-gemma4-26b ... -output a/
ailang eval-suite -agent -prompt-version v0.10.0-slim -models opencode-gemma4-26b ... -output b/
```

Then diff the rotation `summary.json` from each.

### Files to touch

| File | Change | LOC est. |
|---|---|---|
| `cmd/ailang/prompts/agent/v0.10.0-slim.md` | new — the seed prompt | 50 |
| `cmd/ailang/prompts/agent/versions.json` | register v0.10.0-slim (do NOT mark active yet — it's an experiment) | 10 |
| `cmd/ailang/eval_suite.go` | already has `-prompt-version`; verify it threads to agent_prompt.LoadSystemPromptForLanguage correctly | check + maybe 5 |
| `.claude/skills/local-ollama-eval/SKILL.md` | mention the A/B experiment + results once we have them | 30 |
| `internal/eval_harness/rotation_summary.go` | add `prompt_version_used` to BenchmarkSummary (already tracked per-result, lift to summary) | 30 |

### Conflict surface

This design does NOT touch any of:
- `internal/parser/`, `internal/lexer/`, `internal/ast/` — purely runtime prompt selection
- `internal/types/`, `internal/elaborate/`, `internal/iface/` — pure prompt content
- `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/` — no compiler changes
- `cmd/ailang/exec.go` — no execution-path changes

The conflict surface is the existing `-prompt-version` flag wiring, which is harness-only.

## Experiment design

### Phase 1: Stability check (slim seed alone)

Run a single fizzbuzz with the slim seed prompt against gemma4:26b. Confirm:

- Agent does invoke `ailang prompt --kind agent` (verifiable in chains DB)
- After loading the reference, agent produces compilable AILANG
- Total input tokens for the trial is < 30k (vs current ~109k)

If the agent doesn't call `ailang prompt` after the slim seed, we have a model-following-instructions problem and the experiment is over — bail to "iterate on the seed wording" before scaling up.

### Phase 2: Smoke tier A/B

Two rotations of the same 17-benchmark smoke tier, same N=3 trials, same model, same time-of-day (back-to-back), same configured ollama (NUM_PARALLEL=1, KEEP_ALIVE=-1):

- **A**: `-prompt-version v0.9.0` (current embedded reference)
- **B**: `-prompt-version v0.10.0-slim`

Compare via `eval-publish` with trend-delta or `eval-trend candidates --json` diff.

### Phase 3: Decision matrix

| Outcome on B vs A | Decision |
|---|---|
| B pass rate ≥ A pass rate AND B wall-clock ≤ A wall-clock | Promote v0.10.0-slim to `active` in versions.json. Default for the rig |
| B pass rate within 80% of A AND B wall-clock ≤ 50% of A | Promote — net throughput win matters more than per-trial pass rate for $0 compute |
| B pass rate < 80% of A but B is dramatically faster (≤ 30%) | Investigate the failure modes; the seed prompt may need a "fallback to embedded" knob |
| B is worse on both axes | Park v0.10.0-slim under `tags: experimental` in versions.json. Document the negative result and stay on v0.9.0 |

### Phase 4: Larger-model validation (deferred)

Repeat the A/B against one larger model (qwen3-coder:30b or similar) once we have a stable rig running it. The hypothesis that "slim seed scales to larger models" needs a different model's data point to even be claimable.

## Risks and mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Gemma4:26b ignores the "call ailang prompt first" instruction | Medium — v0.8.2 baseline showed this happens to cloud models | Phase 1 is a single-benchmark sanity check before committing to a full rotation. If the agent skips the instruction, abort and revise the seed wording before burning more compute |
| Discovery loop is so chatty it overruns context window | Low — agent prompt is 13KB, individual `ailang docs` outputs are sub-1KB | Cap discovery calls at 5 per trial; if a benchmark needs more, treat as a fail |
| `ailang prompt --kind agent` doesn't yet exist as a CLI flag | Need to verify | Trivial fix — add the flag if missing; falls through to existing `ailang prompt` behavior otherwise |
| Pass rate drops to 0 with slim seed | Possible — gemma4:26b may be too weak | This is itself a useful negative result. It tells us the rig needs a stronger model, not that the prompt strategy is wrong |
| Result inconclusive due to variance | Likely — local Ollama models have high variance | N=3 trials per benchmark is the minimum; consider N=5 if the first A/B is ambiguous |

## What "done" looks like

1. `cmd/ailang/prompts/agent/v0.10.0-slim.md` exists, registered in `versions.json` as `tags: experimental` (not active by default).
2. A Phase-2 A/B rotation has been completed; results are committed to `eval_results/rotation/<date>/<time>_gemma4-26b_smoke_slim_vs_embedded/` with both summary.jsons.
3. A short report (markdown, ≤ 200 lines) at `design_docs/implemented/<version>/m-eval-slim-prompt-self-discovery-report.md` documents:
   - The actual measured numbers vs the targets in Goals
   - The decision matrix outcome
   - Whether `versions.json` `active:` was updated
4. The local-ollama-eval skill's "Prompt strategy" section is updated to reflect what the experiment actually showed (positive OR negative).
5. ADR-003 is opened if the result contradicts ADR-002's "embedded reference is the right default" implicit conclusion.

## Open questions for the user before starting

1. Should the slim seed prompt mention `ailang verify` (Z3 contract verification) as a discovery tool, or only the type-check / run flow? Verify is for *contract* checking; if the agent has no contracts in scope, mentioning it is noise. I'd lean toward leaving it out of v0.10.0-slim and adding it later if we want to test contract-aware benchmarks.
2. Is it worth introducing a *second* experimental variant — a "tiny seed + force one `ailang prompt` call on every fresh session" mode — to differentiate "the model can't discover" from "the model won't discover"? Adds one rotation, gives sharper conclusions.
3. Do we want to gate this experiment on **also** pulling a second local model (e.g. qwen3-coder:30b)? The user has held off until the rig is stable; running A/B on a single weak model risks generalizing from a flaky datapoint. Counter-argument: validating on gemma4:26b first prevents wasted disk if the approach is fundamentally broken.
