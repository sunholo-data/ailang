# M-PKG-CASCADE-PROMPT-CLARITY

**Status**: Planned
**Target**: v0.16.x (follow-up to M-PKG-AUTONOMOUS-CASCADE-SAFE)
**Priority**: P1 — High value (unblocks cheaper models on cascade workflow), low risk (no new infrastructure)
**Estimated**: 1.5 days (~230 LOC)
**Dependencies**: M-PKG-AUTONOMOUS-CASCADE-SAFE (implemented v0.16.x) — IAM separation + Source attribute already in place

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure prompt restructure + template routing — no semantic changes |
| A2: Replayability | 0 | No impact on traces; agent runs remain reproducible by task ID |
| A3: Effect Legibility | +1 | Splitting templates makes the "bump" effect path distinct from the "file-issue" effect path |
| A4: Explicit Authority | +1 | The IAM-restricted Source check moves from prompt-conditional (model-trusted) to coordinator-side dispatch (compiler-enforced) |
| A5: Bounded Verification | 0 | No type-system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Major win: action-first prompts have ~10× higher tool-use rate on smaller models. Removing conditional reasoning burden is a direct cost-of-inference reduction |
| A8: Minimal Syntax | 0 | New `template_by_source` config field is opt-in; existing single-template agents unchanged |
| A9: Cost Visibility | +1 | Cheaper models become viable for cascade work — measured cost ceiling drops from $0.30/run (sonnet-required) to ~$0.03/run (haiku-capable) |
| A10: Composability | +1 | `template_by_source` composes orthogonally with existing `template_by_message_type` |
| A11: Structured Failure | 0 | Failure modes unchanged |
| A12: System Boundary | +1 | Defense-in-depth made stronger: coordinator does the routing, agent only sees pure-action template |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — same Pub/Sub message → same template selection
- [x] A3 (Effects): No hidden side effects — split templates make effects MORE explicit (one template = one effect path)
- [x] A4 (Authority): No ambient access — Source check moves from prompt to coordinator code path, strictly stricter
- [x] A7 (Machines First): Action-first restructure IS the machine-first optimization

## Problem Statement

M-PKG-AUTONOMOUS-CASCADE-SAFE shipped end-to-end cascade infrastructure: `ailang publish` → IAM-restricted Pub/Sub topic → coordinator → Cloud Run Job → agent → branch pushed → PR opened. Today's first real cascade observation against `ailang-multivac-dev` proved the **infrastructure works perfectly**.

But the agent didn't actually do the bump.

**Observed runs (2026-04-30, ailang-multivac-dev):**

| Task ID | Model | Cost | Turns | Tools used | Outcome |
|---|---|---|---|---|---|
| task-feac88d6 | haiku | $0.0132 | 1 | 0 | Generated `AGENTS.md` placeholder; no version bump |
| task-977a4925 | haiku | $0.0152 | 1 | 0 | Parroted prompt as hypothetical: "If this is a genuine cascade-triggered bump, the message should have `Source: cascade` set, and I can proceed..." |

**Root cause:** the `pkg-update.md` template structure. The current template:

1. **Opens with 200 words of "DO NOT" guards** — three bolded "DO NOT bump", "DO NOT publish", "DO NOT push" directives in the first paragraph
2. **Buries the action behind a conditional qualifier** — "The single exception: messages with `Source: cascade`..." comes AFTER the long list of don'ts
3. **Forces conditional reasoning before action** — "Required Steps (only if Source: cascade)" header demands the model evaluate a condition before executing

Smaller models (haiku in particular) cannot climb out of the conditional reasoning level to execute the action branch. They read the prompt structure as a question to discuss rather than a directive to act. **Sonnet handles it correctly but at ~10× the cost.**

**The interesting analytical finding:** our eval-suite benchmarks (`lang_harness_suite`, `harness`) are also multi-turn agentic tool-use tasks where haiku scores well. The capability gap isn't real — it's a **prompt-structure gap that benchmarks don't measure**. Real-world prompts often include safety guards, conditional dispatch, and negation-loaded preambles. Models with shallower instruction-following depth fall apart on this shape; benchmarks with action-first framing don't catch it.

**Current State:**
- 21 pkg-* agents in production (auth, gcp_auth, billing-*, etc.) all on sonnet for this reason
- Per-cascade cost: ~$0.10-0.30 on sonnet (acceptable but not minimal)
- Smaller models are blocked from this workflow despite having the capability
- Defense-in-depth is currently model-trusted (template guard) rather than compiler-enforced (code-path routing)

**Impact:**
- Cost ceiling per cascade is forced higher than necessary
- Agent template UX is brittle — any new conditional makes it worse for smaller models
- Our benchmark suite has a blind spot: it doesn't predict which models can run agentic workflows under adversarial prompt framing

## Goals

**Primary Goal:** Make the cascade workflow execute correctly on any model (haiku, sonnet, opus, third-party CLIs) by removing the conditional reasoning burden from the prompt and moving the Source check to the coordinator dispatch path.

**Success Metrics:**
- Haiku tool-use rate on cascade bump task: from current 0% → ≥90% (measured via fresh smoke runs against `pkg-sunholo-test-pkg-consumer`)
- Per-cascade cost (haiku-capable): from current ~$0.20 (sonnet) → ~$0.03 (haiku)
- New `agentic_under_guards` benchmark added to eval-suite, run for haiku/sonnet/opus baseline
- Zero regression on existing 21 production pkg-* agents (still on sonnet, still working)
- Smoke test scripts (`test_cascade_e2e.sh`) green against haiku-configured fixture

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Action-first restructure of pkg-update.md | Fundamental UX shift for the template; affects every cascade run going forward | human | design | low (markdown only) |
| Split into 2 templates routed by Source | Architectural — eliminates conditional from agent prompt entirely; introduces new config field | human | design | med (config schema + Go) |
| Keep IAM Pub/Sub guard as primary security | Removes prompt-level guard reliance; security relies on M1 coordinator-SA-only IAM | human | design | low (no infra change) |
| Add agentic_under_guards benchmark | Closes a real benchmark gap; could change which models we recommend for autonomous workflows | human | design | low (eval fixtures + harness wiring) |
| Backwards compat: existing single-template agents unchanged | Avoids forced migration of 21 production agents; new field is opt-in | human | design | low (additive Go change) |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Action-first template structure (verified by today's haiku failure — locked in)
- [x] Split into pkg-cascade-bump.md + pkg-feedback-handle.md (locked — eliminates conditional)
- [x] Defense-in-depth via Pub/Sub IAM is the primary security layer (M1 already enforces this)
- [x] `template_by_source` is an additive AgentConfig field (no breaking change to existing agents)
- [ ] Coordinator behavior when `template_by_source` matches but template file doesn't exist — fail loudly or fall back to default? **Recommendation: fail loudly** (no silent fallback per CLAUDE.md principle)
- [ ] Whether to remove the conditional guard from pkg-update.md once split templates land (cleanup vs leave-as-archive) — **Recommendation: simplify pkg-update.md to be the bump-action template; delete the conditional preamble**

## Solution Design

### Overview

Three coordinated changes that together eliminate the prompt-structure gap:

1. **Restructure `pkg-update.md` to be action-first** (immediate fix — works for all models without code change)
2. **Add `template_by_source` routing** so cascade and feedback are physically separate templates the model never sees in the same context (architectural fix — removes conditional entirely)
3. **Add `agentic_under_guards` benchmark** that catches this class of regression in future template changes

### Architecture

**Components:**

1. **Restructured `pkg-update.md`** (markdown only): Lead with the bump action and concrete tool sequence. Move the Source guard to a one-line footer ("If Source ≠ cascade, this message should not have reached you — file an issue and stop"). The IAM layer ensures this footer never executes in practice; it's vestigial defense-in-depth.

2. **New `pkg-cascade-bump.md` template** (markdown): Pure action template. Assumes cascade has been validated upstream. Format:
   ```
   Bump dependency `{{.RootPackage}}` in this package's ailang.toml.

   Steps (in order):
   1. Read ailang.toml with Bash
   2. Edit ailang.toml — change the dep version to `{{.RootVersion}}`
   3. Run `ailang check --package .` with Bash
   4. Run `ailang test --package .` with Bash
   5. Commit with message: "[cascade] bump {{.RootPackage}} to {{.RootVersion}}"
   6. Push branch and exit
   ```
   No conditionals. No "DO NOT". No safety preamble. The coordinator has already enforced that we're here because cascade fired.

3. **New `pkg-feedback-handle.md` template** (markdown): Pure issue-creation template. Used when public MCP feedback arrives. No bump capability — the model literally never sees the bump instructions.

4. **`template_by_source` field on AgentConfig** (Go schema change in `internal/coordinator/agent.go`):
   ```go
   type AgentConfig struct {
       // ... existing fields ...
       Invoke struct {
           TemplateFile          string            `yaml:"template_file"`
           TemplateByMessageType map[string]string `yaml:"template_by_message_type"`
           TemplateBySource      map[string]string `yaml:"template_by_source"` // NEW
       }
   }
   ```

5. **Coordinator template selection** (in `internal/coordinator/stage_execution.go`): Resolution order becomes:
   ```
   template_by_source[task.Source] (if set)
   → template_by_message_type[task.Kind] (if set)
   → template_file (default)
   ```
   `task.Source` is already populated from M-PKG-AUTONOMOUS-CASCADE-SAFE M1; we're just adding a new resolution rule.

6. **`agentic_under_guards` benchmark** (in `benchmarks/`): A pair of identical tasks differing only in prompt framing — measures the capability gap directly so future template changes can be regression-tested.

### Implementation Plan

**Phase 1: Action-first restructure** (~3 hours)
- [ ] Rewrite `pkg-update.md` to lead with action; move guard to footer
- [ ] Deploy via config-only fast path (~60s)
- [ ] Run smoke test against haiku-configured `pkg-sunholo-test-pkg-consumer`
- [ ] Measure: tool-use rate, completion rate, cost
- [ ] Acceptance: haiku completes the bump (≥90% over 5 runs)

**Phase 2: Template routing** (~6 hours)
- [ ] Add `template_by_source` field to `AgentConfig` Go struct
- [ ] Wire `template_by_source` resolution in `stage_execution.go` (priority: source > message_type > default)
- [ ] Create `pkg-cascade-bump.md` (pure action) and `pkg-feedback-handle.md` (pure issue-create)
- [ ] Update test fixture agents to use the new routing
- [ ] Unit test: AgentConfig parses `template_by_source`; resolution order is correct
- [ ] Integration test: cascade message routes to bump template; non-cascade routes to feedback template
- [ ] Backwards compat test: agents without `template_by_source` still work

**Phase 3: Benchmark addition** (~3 hours)
- [ ] Design `agentic_under_guards` benchmark fixture (paired direct/guarded prompts)
- [ ] Wire benchmark into eval-suite harness
- [ ] Run baseline for haiku, sonnet, opus
- [ ] Document the capability gap in benchmark output
- [ ] Add benchmark to default eval-suite run

### Files to Modify/Create

**New files:**
- `ailang-multivac/config/templates/pkg-cascade-bump.md` — pure action template (~30 LOC markdown)
- `ailang-multivac/config/templates/pkg-feedback-handle.md` — pure issue-create template (~25 LOC markdown)
- `benchmarks/agentic_under_guards/` — paired benchmark fixtures (~150 LOC YAML + verifier)

**Modified files:**
- `ailang-multivac/config/templates/pkg-update.md` — restructure to action-first OR mark deprecated once routing lands (~50 LOC change)
- `internal/coordinator/agent.go` — add `TemplateBySource` field (~10 LOC)
- `internal/coordinator/stage_execution.go` — wire source-based resolution (~20 LOC)
- `internal/coordinator/agent_test.go` — config parse + resolution order tests (~50 LOC)
- `ailang-multivac/config/config.cloud.yaml` — opt in `pkg-sunholo-test-pkg-consumer` (and ideally all 14 prod pkg agents) to `template_by_source` routing (~25 LOC YAML)

## Examples

### Example 1: Same cascade message, two template structures

**Before (current pkg-update.md, haiku response on 2026-04-30):**

Prompt fragment:
```
## CRITICAL: Cascade-trigger guard
**Source of this message:** `cascade`
This message is an authoritative bump trigger ONLY when `Source: cascade`...
DO NOT bump the version
DO NOT run `ailang publish`
DO NOT push to the registry
...
The single exception: messages with `Source: cascade` are real bump triggers...
## Required Steps (only if Source: cascade)
1. Read AGENT.md...
```

Haiku's response (turn 1, tools=0, $0.0152):
> "If this is a genuine cascade-triggered bump, the message should have `Source: cascade` set, and I can proceed with reading AGENT.md, running checks/tests, bumping the version, and publishing."

(No tools invoked. Branch pushed with placeholder `AGENTS.md`. No bump.)

**After (proposed pkg-cascade-bump.md):**

Prompt:
```
Bump dependency `sunholo/test_pkg` to `0.0.17` in this package's ailang.toml.

Steps:
1. Use Bash to read ailang.toml
2. Use Edit to change the dep version
3. Use Bash to run `ailang check --package .`
4. Use Bash to run `ailang test --package .`
5. Use Bash to commit with message: "[cascade] bump sunholo/test_pkg to 0.0.17"
6. Use Bash to push the branch
```

Expected haiku behavior: invokes Bash → reads toml → invokes Edit → updates version → invokes Bash for check/test → commits → pushes. Same workflow, ~$0.03, 6 turns, 5 tools.

### Example 2: Coordinator template resolution

**Agent config:**
```yaml
- id: pkg-sunholo-test-pkg-consumer
  invoke:
    template_file: /etc/ailang-config/templates/pkg-update.md  # legacy fallback
    template_by_source:
      cascade: /etc/ailang-config/templates/pkg-cascade-bump.md
    template_by_message_type:
      feedback: /etc/ailang-config/templates/pkg-feedback-handle.md
```

**Resolution behavior:**
- Cascade message arrives (Source: "cascade") → use `pkg-cascade-bump.md` (Source match wins)
- Public feedback arrives (Source: "messages", message_type: "feedback") → use `pkg-feedback-handle.md` (message_type match)
- Anything else → fall back to `template_file`

## Success Criteria

- [ ] Haiku completes the bump workflow on `pkg-sunholo-test-pkg-consumer` (≥90% success rate over 5 smoke runs)
- [ ] Cost per cascade run on haiku ≤ $0.05
- [ ] All 21 existing pkg-* production agents continue to work without config changes
- [ ] `template_by_source` field validates and routes correctly
- [ ] Source resolution order documented in `docs/docs/guides/coordinator.md`
- [ ] `agentic_under_guards` benchmark added with haiku/sonnet/opus baselines
- [ ] Smoke test scripts (`test_cascade_e2e.sh`) green
- [ ] All existing tests passing
- [ ] Design doc moved to `implemented/v0_16_x/`

## Testing Strategy

**Unit tests (in `internal/coordinator/`):**
- AgentConfig YAML parsing accepts `template_by_source`
- Template resolution priority: source > message_type > default
- Missing template file in `template_by_source` map → fail loudly (no silent fallback)
- Backwards compat: agents without `template_by_source` use existing resolution

**Integration tests (in `internal/coordinator/`):**
- Cascade-source message + agent with `template_by_source.cascade` set → bump template loaded
- Non-cascade message + agent with both routing rules → message_type rule wins
- No matching rules → default template

**Manual smoke (against `ailang-multivac-dev`):**
- Bump `sunholo/test_pkg` → cascade fires → haiku-configured `pkg-sunholo-test-pkg-consumer` agent → PR opened with proper toml bump
- Verify cost in `ailang chains view <task-id>`
- Verify tool-use count > 0

**Benchmark validation:**
- Run `agentic_under_guards` for haiku/sonnet/opus
- Expected: haiku divergence between direct/guarded ≥ 60 points; sonnet divergence ≤ 15 points

## Deferred Decisions

- Whether to migrate all 21 production pkg-* agents to `template_by_source` routing immediately or just the test fixtures (agent may decide based on smoke test results — recommend leaving prod agents on existing single-template config until cascade behavior is verified across multiple real publishes, then bulk-migrate)
- Whether `pkg-update.md` becomes the cascade-bump template (after deprecating the conditional guard) or stays as a legacy fallback (agent may decide — recommend the former, fewer files to maintain)
- Exact fixture content for `agentic_under_guards` benchmark — task should be representative but not specifically about packages (agent may choose: dependency bump, file rename, config edit, etc.)
- Whether to surface the prompt-structure capability gap publicly in the eval-suite gallery (project decision — recommend yes, it's a useful signal for the AI eval community)

## Non-Goals

- Removing the IAM Pub/Sub guard — that remains the primary security layer; this work strengthens it by removing reliance on prompt-conditional guarding
- Rewriting all 21 pkg-* agent prompts — only test fixtures + the cascade workflow templates
- Auto-merge of cascade PRs — still always-PR with human review per parent doc M-PKG-AUTONOMOUS-CASCADE-SAFE
- Adding new Pub/Sub topics or changing the cascade dispatch flow — purely a prompt + config + routing change
- Adding new agent capabilities — same effects, same git_mode, same auto_merge=false

## Timeline

**Day 1 morning** (~3 hours):
- Phase 1: Restructure pkg-update.md, deploy, smoke test against haiku
- Validate the action-first hypothesis with real cost numbers

**Day 1 afternoon** (~6 hours):
- Phase 2: Add `template_by_source` field, write tests, deploy
- Migrate test fixtures to new routing
- Integration smoke

**Day 2 morning** (~3 hours):
- Phase 3: agentic_under_guards benchmark + baselines
- Document in coordinator guide
- Move design doc to implemented

**Total: ~12 hours across 1.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Phase 1 alone (markdown change) doesn't fix haiku — model still asks clarifying questions | Med | Phase 2 is the real fix; Phase 1 is a fast win that may or may not work standalone. Plan accommodates both outcomes. |
| Splitting templates introduces a new failure mode if `template_by_source` is misconfigured | Low | Fail-loudly on missing template file (per CLAUDE.md no-silent-fallback principle); unit tests cover misconfiguration |
| `agentic_under_guards` benchmark is unfair to smaller models, becomes a "haiku is bad" PR risk | Low | Frame the benchmark as measuring a specific gap (instruction-following depth under conditional framing), not overall capability. Headline finding is "this informs which models suit which workflows," not "model X is bad." |
| 14 prod agents need migration to get full benefit | Med | Phased: test fixtures first; bulk migration of prod agents after one week of stable cascade observation. Backwards compat means no forced cutover. |
| Prompt-structure changes break existing benchmarks | Low | Templates are not exercised by current benchmarks (no eval covers cascade workflow). The new benchmark is additive. |

## Related Documents

<!-- Auto-populated by Ollama neural search; cleaned up to focus on actually relevant prior art -->

**Implemented (parent / sibling work):**
- [design_docs/implemented/v0_16_x/m-pkg-autonomous-cascade-safe.md](../../implemented/v0_16_x/m-pkg-autonomous-cascade-safe.md) — direct parent: built the cascade infrastructure this work optimizes
- [design_docs/implemented/v0_16_x/m-pkg-autonomous-cascade-safe-sprint-plan.md](../../implemented/v0_16_x/m-pkg-autonomous-cascade-safe-sprint-plan.md) — sprint plan showing the M1 IAM separation we now lean on
- [design_docs/implemented/v0_10_0/m-pkg-autonomous-updates.md](../../implemented/v0_10_0/m-pkg-autonomous-updates.md) — grandparent: the original autonomous-publish pipeline
- [design_docs/implemented/v0_3_10/M-EVAL-LOOP_self_improving_feedback.md](../../implemented/v0_3_10/M-EVAL-LOOP_self_improving_feedback.md) — eval-suite background relevant to the new benchmark addition

**Planned (check for overlap):**
- [design_docs/planned/v0_15_0/m-eval-gap-fixes.md](../v0_15_0/m-eval-gap-fixes.md) — possible coordination point if both touch eval harness

## References

- [Design Axioms](/docs/references/axioms) — A7 (Machines First) is the primary justification for this work
- [Today's smoke test observations] — task-feac88d6 and task-977a4925 in `ailang-multivac-dev` Cloud Run Job logs (2026-04-30, 17:01-17:14 UTC)
- [pkg-update.md template](https://github.com/sunholo-data/ailang-multivac/blob/main/config/templates/pkg-update.md) — the template being restructured
- [internal/coordinator/agent.go](https://github.com/sunholo-data/ailang/blob/dev/internal/coordinator/agent.go) — where AgentConfig schema is defined
- [internal/coordinator/stage_execution.go:220](https://github.com/sunholo-data/ailang/blob/dev/internal/coordinator/stage_execution.go) — existing `{{.Source}}` substitution that proves Source field is already plumbed end-to-end

## Future Work

- **Bulk migration of 14 prod pkg-* agents** to `template_by_source` routing (separate small sprint, after this lands and one week of cascade observation passes cleanly)
- **Cheaper-model evaluation across all autonomous workflows** — once we know prompt structure unblocks haiku for cascade, audit other agentic workflows (design-doc-creator, sprint-planner, sprint-executor) for the same restructuring opportunity
- **Prompt linter** that flags conditional-heavy / negation-loaded templates before they ship (could be a Go tool that checks template files in CI)
- **Public write-up of the agentic_under_guards finding** — a blog post on prompt-structure-vs-capability could be valuable for the AI eval community

---

**Document created**: 2026-04-30
**Last updated**: 2026-04-30
