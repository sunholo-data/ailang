# M-AGENT-LOOP-ARCHITECTURE: Where Should the Multi-Turn Agent Loop Live?

**Status**: Planned (design exploration — no code, no estimate yet)
**Target**: v0.17.x or v0.18.0 (depends on conclusion)
**Priority**: P1 (blocking M3 of motoko_agent's tool-loop migration; informs the v0.17.0 `runTools` API freeze)
**Estimated**: TBD — pure design exploration; outcome determines whether implementation is days or weeks
**Dependencies**: M-AI-TOOL-LOOP shipped (v0.17.0-targeted, dev-only as of 2026-05-05)

---

## Framing

> **Where does the multi-turn agent loop driver — including tool dispatch, extension intercepts, capability gating, backend routing, and continuation-intent feedback — belong: upstream in `std/ai`, downstream in each consumer, or split via a hook-rich `runTools`?**

M-AI-TOOL-LOOP shipped the primitive — `step(model, messages, tools) -> Result[StepResult, AIError]` and a thin `runTools(...)` driver — exactly enough for an agent that emits tool calls, dispatches them, and feeds results back. It is **structurally insufficient** for [arniwesth/motoko_agent](https://github.com/arniwesth/motoko_agent), which already had a much richer loop with 6 distinct decision points the upstream API doesn't model.

This blocks M3 of [motoko_agent PR #4](https://github.com/arniwesth/motoko_agent/pull/4) (the tool-loop migration follow-up to PR #3). M1 + M2 (the dispatch adapter and `[ToolSchema]` catalog) shipped — they're useful standalone. M3 is "swap rpc.ail's loop for `runTools`" and that swap is not 1:1.

This document examines three options for where motoko's missing decision points should live and recommends a path. The recommendation is **explicitly contingent on assumptions stated in §Assumptions** — when arni weighs in those assumptions become testable, and the recommendation can flip cleanly.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a design *exploration*, not a feature. Axiom scores below are scored *per option* in §Option Comparison rather than as a single net for the doc. Hard-violation check (A1/A3/A4/A7) is applied to each option there.

### Hard Violation Check

- [x] A1 (Determinism): No option introduces implicit nondeterminism.
- [x] A3 (Effects): All three options preserve effect-row legibility (the dispatch callback's effects propagate via row polymorphism in all options).
- [x] A4 (Authority): No option grants ambient capabilities — all three keep the dispatch callback as a typed AILANG function with explicit effects.
- [x] A7 (Machines First): The choice between A/B/C IS an A7 question — see §Option Comparison.

### Decision Thresholds

Per-option in §Option Comparison. Whichever option scores highest A7 + A10 wins, subject to no hard violations.

## Problem Statement

**Current state**: M-AI-TOOL-LOOP shipped these primitives in `std/ai`:

```ailang
step(model: string, messages: [Message], tools: [ToolSchema])
  -> Result[StepResult, AIError] ! {AI}

runTools(model: string, messages: [Message], tools: [ToolSchema],
         dispatch: (ToolCall) -> string, step_budget: int)
  -> Result[[Message], AIError] ! {AI}
```

`runTools` is a thin loop: call `step`, dispatch any returned `tool_calls`, append `tool` messages, repeat. It is the smallest viable agent loop.

**Mismatch**: `motoko_agent`'s loop in [src/core/rpc.ail](https://github.com/arniwesth/motoko_agent/blob/ailang-tool-loop-migration/src/core/rpc.ail) has SIX additional decision points that fire between turns and that `runTools` doesn't expose:

### The 6 decision points (catalogued from rpc.ail walking)

For each: line reference, what it does, why it exists, what `runTools` would need to expose to support it.

| # | Decision point | rpc.ail location | What it does | Why motoko has it |
|---|---|---|---|---|
| 1 | **Extension intercept** (`ContinueWithFeedback` / `Accept` / `NoIntercept` / `NoDecision`) | ~line 970 (`dispatch_response_intercept`) | Runs BEFORE tool dispatch each turn; can short-circuit the loop with a final answer (`Accept`), inject a feedback message and re-prompt (`ContinueWithFeedback`), or pass through (`NoIntercept` / `NoDecision`). | Lets compose extensions (claimcheck, author_loop) overlay validation logic without forking the loop driver. |
| 2 | **Tool-call parser branching** (`NoToolCalls` / `ToolParseError` / `ParsedToolCalls`) | line 999 (`parse_tool_calls`) | Distinguishes "no calls emitted" (continuation-intent path), "malformed call format" (inject parse-error feedback), "calls present" (dispatch). | Models often emit malformed tool-call envelopes; motoko teaches the model how to fix them rather than failing the turn. |
| 3 | **Tool gating policy** | line 1069 (`apply_tool_policy`) | Filters allowed/denied calls per extension hooks; denied calls become `ToolErrorResult` items fed back to the model. | Lets extensions reject dangerous calls (e.g. shell commands matching a denylist) without aborting the loop. |
| 4 | **Tool-handle routing** | line 1079 (`route_tool_handles`) | Lets extensions handle specific tool names directly (returning `ToolResultEnvelope`s) before the native dispatcher sees them. | Lets compose extensions implement custom tools (e.g. `compose_check_premise`) without modifying `tool_runtime.ail`. |
| 5 | **Backend split** | line 1080 (`split_by_backend`) | Divides tool calls between native execution (`run_native_batch`) and delegated execution (`ohmy_pi`). | ohmy_pi is the fork's experimental remote-execution backend; some tools run on a different machine than the agent itself. |
| 6 | **Continuation-intent detection** | ~line 1029 (`indicates_continuation_intent`) | When the model emits no tool calls AND its prose suggests it wants to continue, inject a "you said you'd continue but didn't emit a tool call" feedback message rather than terminating. | Models occasionally lapse into prose when they should be acting; motoko coaches them back. |

Plus per-step `emit_event` calls for telemetry: `ext_tool_calls`, `ext_tool_results`, `native_tool_calls`, `native_tool_results`, `done`, `thinking_*`, `compose_*`. These aren't decision points — they're observability — but they too need a place to land.

**Symptom**: PR #4's M3 was originally estimated at 1.5–2 hours ("just swap the loop driver"). The actual scope is 1–2 days at minimum, with substantial design ambiguity about whether several of the 6 decision points should even survive the migration.

## Goals

**Primary goal**: Decide where the agent loop driver — including the 6 decision points above — should live, with a recommendation that survives at least the next 2-3 motoko-class consumers.

**Success metrics**:

1. **Clear decision** about each of the 6 decision points: upstream hook, motoko-internal, or droppable feature
2. **Cost-of-change estimate** for each option (LOC, breaking changes, upstream API surface impact)
3. **Concrete next steps** for whichever option wins — either a sprint plan to extend `runTools`, or a sprint plan to inline a custom loop in motoko, or a deprecation path for features motoko drops
4. **Reusable framework** for future "should this live upstream or downstream" questions about agent infrastructure (eval-harness, docparse legal review, hypothetical SDK)

**Non-goal**: pick a winner in this document. Recommendation is explicit and conditional on §Assumptions; arni's input flips the call cleanly.

## High-Impact Decisions

| Decision | Options | Who decides | Change cost |
|----------|---------|-------------|-------------|
| Where the loop driver lives | A: motoko / B: upstream `runTools` (extended) / C: motoko simplifies | AILANG team + arni jointly | A=motoko-side (~1–2d), B=upstream-side (~1 sprint, public API impact), C=motoko-side (~3–5d, feature loss) |
| If B wins: what hooks does extended `runTools` expose? | Subset of {1,2,3,4,5,6} above + telemetry hooks | AILANG team primarily | Per-hook surface area; each hook is a function-typed parameter |
| If A wins: what is motoko's reusable loop module shape? | One file in motoko, OR a shared `ailang-agent-utils` package | arni primarily | Either way ~600 LOC; package adds publishing overhead |
| If C wins: which of the 6 decision points are dropped? | Feature-by-feature triage | arni primarily | Each dropped feature → user-facing breaking change for the motoko-agent product |
| Whether to formalise this as M-AGENT-COMPOSABILITY upstream | Define a v1.0+ axis: "agent loops are user-extensible kernels" | AILANG team | Long-running design axis; does NOT block this milestone |

## Solution Design

### Option A — Custom loop in motoko, calling `step()` directly

**Sketch**:

```ailang
-- motoko_agent/src/core/agent_loop.ail (new module replacing rpc.ail's loop body)
import std/ai (step, Message, ToolSchema, StepResult, AIError)
import src/core/tool_dispatch_adapter (dispatch_one)
import src/core/tool_catalog (tools)

func turn_loop(
  model: string,
  msgs: [Message],
  ext: ExtRuntime,
  budget: BudgetPlan,
  settings: RunSettings,
  step_budget: int
) -> Result[[Message], AIError] ! {AI, FS, Process, IO} {
  if step_budget <= 0 then Err({code: "Internal", message: "step budget", retryable: false})
  else match step(model, msgs, tools()) {
    Err(e) => Err(e),
    Ok(result) => {
      -- Decision point 1: extension intercept
      match dispatch_response_intercept(ext, result.message.content) {
        Accept(output) => Ok(msgs ++ [{role: "assistant", content: output, ...}]),
        ContinueWithFeedback(fb) => turn_loop(model, msgs ++ [{role: "user", content: fb}, ...], ext, budget, settings, step_budget - 1),
        NoIntercept => continue_with_tool_calls(result, model, msgs, ext, budget, settings, step_budget),
        NoDecision => maybe_continuation_intent(result, model, msgs, ext, budget, settings, step_budget)
      }
    }
  }
}
-- ... continue_with_tool_calls dispatches the 6 decision points using upstream step + motoko's existing helpers
```

**Pros**:
- Zero impact on upstream `runTools` API
- Motoko keeps total control over loop semantics
- Extensions, telemetry, ohmy_pi backend split — all preserved verbatim
- Other AILANG consumers (eval-harness, docparse) keep using simple `runTools` without inheriting motoko's complexity

**Cons**:
- Net deletion in `rpc.ail` is small (~50–150 LOC, mostly just the message-construction → step call wrapping)
- The "value prop" of M-AI-TOOL-LOOP for motoko collapses to "use `step` instead of `_ai_call_json`" — typed errors + multi-turn message shape, not loop reuse
- Future motoko-class consumers (e.g. an enterprise AILANG agent SDK) would each need to re-implement their own loop driver; no upstream amortisation

**LOC estimate**: ~600 LOC custom loop module + ~50 LOC deletions in rpc.ail = net flat or slight increase

**Axiom scores**:
- A7 Machines First: 0 (no change — motoko stays as complex as today)
- A10 Composability: 0 (motoko's loop composes only with motoko's extensions; not reusable)
- A11 Structured Failure: +1 (gains typed `AIError` from upstream `step`)
- Net: +1

### Option B — Extend upstream `runTools` with hook parameters

**Sketch**:

```ailang
-- New std/ai surface (would replace existing runTools)
type RunToolsConfig = {
  step_budget: int,                                                          -- existing
  on_response: Option[(StepResult, [Message]) -> InterceptDecision],         -- DP1: extension intercept
  on_parse_error: Option[(string, [Message]) -> string],                     -- DP2: parse error feedback
  policy: Option[(ToolCall) -> ToolDecision],                                -- DP3: tool gating
  custom_handlers: Option[[(string, ToolCall) -> Option[string]]],           -- DP4: handle routing
  -- DP5 (backend split) is dispatch-callback-internal — caller can fan out themselves
  on_no_calls: Option[(StepResult, [Message]) -> NoCallDecision],            -- DP6: continuation-intent
  on_event: Option[(AgentEvent) -> ()]                                       -- telemetry
}

type InterceptDecision = Accept(string) | ContinueWithFeedback(string) | Pass
type ToolDecision = Allow | Deny(string)
type NoCallDecision = Terminate | InjectFeedback(string)

runTools(model, messages, tools, dispatch, config: RunToolsConfig)
  -> Result[[Message], AIError] ! {AI}
```

All 5 callable hooks are `Option[func]` so existing callers (eval-harness, docparse, simple agents) pass `None` and get the current behaviour. Motoko (and other complex consumers) supply real callbacks.

**Pros**:
- Motoko's loop body collapses to ~30 LOC: build a `RunToolsConfig`, call `runTools`
- ALL motoko's decision points become typed, composable, replayable through the upstream trace machinery
- Future agent consumers (eval-harness, docparse legal review, hypothetical AILANG agent SDK) can opt into individual hooks without reinventing the loop
- Hook decisions become inspectable in traces (huge for A2 Replayability + A9 Cost Visibility)

**Cons**:
- Significant upstream design + implementation: 5 new ADTs (`InterceptDecision`, `ToolDecision`, `NoCallDecision`, `AgentEvent`, `RunToolsConfig`), ~400 LOC + tests
- `runTools` signature change is a breaking change to a function that just shipped — needs versioning thought
- Hook callbacks have effect-row implications (each is `(...) -> ... ! ε`) — row polymorphism propagation is correct in theory but adds typechecker stress
- Risk of designing for motoko's specific shape and missing what other consumers actually need (cargo-culting one consumer's architecture into the stdlib)

**LOC estimate**: ~400 LOC upstream (`std/ai.ail` + tests) + ~30 LOC motoko loop + ~600 LOC deletions in rpc.ail = ~600 LOC net deletion across the system

**Axiom scores**:
- A7 Machines First: +2 (hook-rich loop is much more decomposable than monolithic motoko-style loops; agents become smaller programs)
- A10 Composability: +2 (the central reason this option exists)
- A11 Structured Failure: +1 (typed decisions for each hook)
- A12 System Boundary: +1 (each hook is an explicit boundary; the loop driver becomes a kernel, hooks are user code)
- Net: +6

### Option C — Motoko simplifies to fit current `runTools` shape

**Sketch**: Drop decision points 1, 3, 4, 5, 6 (keep only 2 — parse error feedback, which is small enough to handle inside motoko's dispatch callback). Use upstream `runTools` directly. Migrate compose extensions to a different surface (e.g. validate after `runTools` returns rather than between turns).

**Pros**:
- Zero upstream change
- Smallest motoko-side rewrite (~30 LOC `runTools` call + ~150 LOC compose-extension migration)
- Clean test of "is the upstream API enough?" — if motoko can drop the features, others probably can too

**Cons**:
- **User-facing feature loss**: ohmy_pi backend split, mid-loop compose-extension overlay, continuation-intent coaching all go away
- This is a product decision, not just an architectural one. Affects motoko_agent users.
- May not actually be feasible — extensions like `claimcheck` rely on intercept for their entire reason for being

**LOC estimate**: ~30 LOC `runTools` call + ~150 LOC extension migration + ~600 LOC deletions = ~400 LOC net deletion + feature loss

**Axiom scores**:
- A7 Machines First: +1 (simpler agent shape; lower cognitive load)
- A10 Composability: +1 (cleanly composes with upstream — but at the cost of dropping motoko's compose extensions)
- A11 Structured Failure: +1 (typed errors)
- Net: +3 — but the feature-loss is not an axiom-scored concern; it's a product concern

### Option Comparison

| | A: motoko custom | B: upstream hooks | C: motoko simplifies |
|---|------------------|-------------------|----------------------|
| Upstream change | None | Significant (5 ADTs + sig change) | None |
| Motoko change | Medium (~600 LOC) | Small (~30 LOC) | Medium (~150 LOC + feature loss) |
| Net LOC system-wide | Flat or slight increase | ~600 LOC deletion | ~400 LOC deletion |
| Axiom net | +1 | +6 | +3 |
| Feature regression | None | None | Yes — user-visible |
| Reusability for future consumers | None | High | None |
| Risk to v0.17.0 timeline | None | High (would block freeze) | None |
| Risk to motoko PR #4 | Low (can ship as-is) | High (waits for upstream) | Medium (feature decision needed) |
| Reversibility | High (motoko-side) | Low (public API change) | Low (feature loss is product-visible) |

### Pipeline Pass Coverage (Option B only)

If Option B is chosen, the affected pipeline passes are:
- **Parser**: no change (no new syntax)
- **Type checker**: rich impact — `RunToolsConfig` adds 4 new ADTs that need to flow through type inference; `Option[func]` parameters with row-polymorphic effects test the unifier
- **Effect inference**: each hook contributes its effects via row polymorphism; the `runTools` call site's effect row becomes the union — needs golden tests
- **Iface JSON**: `RunToolsConfig` and the 4 decision ADTs need iface serialisation
- **Eval / VM**: hook dispatch is just function application; no new opcodes
- **Trace**: each hook fires a trace event distinguishable from `AI/step` — new `AI/hook[on_response]` etc. trace shape

## Conflict Surface (REQUIRED for Option B)

This section is mandatory for Option B because it modifies a public stdlib surface (`std/ai.runTools`) and adds new types that flow through the type-checker + iface layer. Per the new design-doc gate, the author must enumerate what else uses these positions before merging.

### Syntactic positions touched (Option B)

- New ADTs in `std/ai.ail`: `InterceptDecision`, `ToolDecision`, `NoCallDecision`, `AgentEvent`, `RunToolsConfig` — flow through iface JSON, type inference, monomorphisation
- `runTools` signature change: adds `config: RunToolsConfig` parameter (or replaces `step_budget: int` with config record)
- Hook function types `(StepResult, [Message]) -> InterceptDecision` — row-polymorphic effects on user-supplied callbacks

### What else lives here

| Position | Existing valid form | Shape | Conflict risk |
|----------|--------------------|-------|---------------|
| `std/ai` exports | `step`, `runTools`, `call`, `callJson`, `callJsonSimple`, `callImage`, `callImageBase64`, `callResult`, `callJsonResult` | function exports | None — adds new exports + revises one |
| `runTools` signature | Currently `(model, messages, tools, dispatch, step_budget)` | 5-arity function | **Adding a config param breaks ALL existing callers.** Need a `runToolsWithConfig` companion + deprecation path, or a default-arg pattern. |
| Hook callback types | None analogous in std/ai today | `(...) -> Decision` | Decision ADT names: `Accept`, `Pass`, `Deny`, `Terminate` — common names; check for conflicts with existing variant tags in user code. Found: `Accept` is used in motoko's existing `dispatch_response_intercept` (different shape — needs aliasing). |
| Iface JSON serialisation | `Result`, `Option`, etc. existing types | tagged-union JSON | Standard pattern; iface builder handles via `tlabelled` envelope. No new iface mechanism needed. |

### Disambiguation strategy

- New ADTs use unique names where possible. The `Accept` collision with motoko's existing intercept ADT is an alias issue — motoko-side users need to qualify or rename.
- `runTools` signature breakage: ship as `runToolsWithConfig` initially, deprecate `runTools` over 2 minor versions, eventually rename. Documented migration path in the v0.17.x → v0.18.0 changelog.

### Programs that MUST still work post-change

If Option B ships:
1. `examples/runnable/ai_tool_loop.ail` — the existing motoko-style canned tool loop
2. eval-harness benchmarks that use `step` directly (none migrated to `runTools` yet — this is pre-emptive)
3. motoko_agent's `runTools` adoption path (the whole point of this milestone)
4. Hypothetical SDK consumers — passing config = None / default should be byte-identical to current `runTools` behaviour
5. `examples/runnable/ai_tool_loop.ail` AST-snapshot — verify the existing `runTools(model, msgs, tools, dispatch, 8)` call still type-checks (via deprecation shim or default-config helper)

### What deliberately changes (Option B)

- `runTools`'s signature gains a config parameter. Existing 5-arity callers go through a thin `runToolsWithConfig(model, msgs, tools, dispatch, defaultConfig(8))` shim or `runTools` becomes a wrapper around `runToolsWithConfig` with `defaultConfig`.
- 4 new public ADTs in `std/ai`. Names finalized in implementation sprint.

## Examples (option-by-option illustration)

**Option A** — motoko's `agent_loop.ail` (sketch):

```ailang
match step(model, msgs, tools()) {
  Ok(result) => {
    let intercept = dispatch_response_intercept(ext, result.message.content);
    match intercept {
      Accept(output) => emit_done_and_return(state, output),
      ContinueWithFeedback(fb) => recurse(msgs ++ feedback_msg(fb), step_budget - 1),
      NoIntercept => continue_dispatching_calls(result),
      NoDecision => maybe_continuation_intent(result)
    }
  },
  Err(e) => Err(e)
}
```

**Option B** — same logic with hook-rich `runTools`:

```ailang
let config = {
  step_budget: 8,
  on_response: Some(\result, msgs. dispatch_response_intercept(ext, result.message.content)),
  policy: Some(\call. apply_tool_policy_one(call, ext, ctx)),
  custom_handlers: Some([compose_check_premise_handler(ext)]),
  on_no_calls: Some(\result, msgs. if indicates_continuation_intent(result.message.content) then InjectFeedback(continuation_intent_feedback()) else Terminate),
  on_event: Some(\evt. emit_event_to_telemetry(evt))
};
runTools(model, msgs, tools(), dispatch_one(workdir, _), config)
```

**Option C** — motoko using current `runTools` after dropping intercept/gating/backend-split:

```ailang
runTools(model, msgs, tools(), \call. dispatch_one(workdir, call), 8)
-- Compose extensions migrated to post-loop validation:
-- After runTools returns Ok(messages), run the validators against the final transcript.
```

## Success Criteria

This is a design doc, not an implementation. Success is **arriving at a decision the AILANG and motoko teams both endorse**. Concrete success criteria:

- [ ] arni reviews and either confirms or pushes back on each of the 6 decision points (which are essential vs droppable)
- [ ] AILANG team reviews and either confirms or pushes back on the runTools API impact analysis
- [ ] Joint decision: A, B, C, or hybrid
- [ ] If A: motoko PR #4 unblocks immediately with a sprint plan for the custom loop
- [ ] If B: AILANG team writes a sprint plan for the runTools extension (~1 sprint, public API change)
- [ ] If C: motoko team writes a deprecation plan for the dropped features
- [ ] **Reusable framework**: the §Option Comparison table becomes a template for future "where does this agent feature live" questions

## Testing Strategy

This document doesn't test code — it tests *decisions*. The "tests" are:

1. **Survey**: are there other AILANG agent consumers (eval-harness, docparse legal review, hypothetical SDK) whose loops would benefit from each of the 6 decision points? See §Downstream Impact.
2. **Cost-of-change**: each option's LOC + breaking-change estimate, validated against actual motoko + std/ai source.
3. **Axiom scoring**: each option scored against all 12 axioms; the winner has the highest net + no hard violations.

If Option B is chosen, the implementation milestone gets its own design doc with full Conflict Surface analysis, regression tests, and a sprint plan.

## Downstream Impact (other potential consumers)

This is the load-bearing question for choosing between A and B. **Is motoko's complexity unique, or is it the leading edge of a class of consumers?**

| Potential consumer | Status | Loop complexity | Hooks needed |
|--------------------|--------|-----------------|--------------|
| **eval-harness** (`internal/eval_harness`) | Production | Medium — runs tools but doesn't have extension intercepts; has its own benchmark-scoring loop | Maybe `on_event` for cost tracking; otherwise simple |
| **docparse legal review** (planned, ailang-parse v0.18) | Planned | Likely high — multi-stage validation, claim checking similar to motoko's claimcheck extension | DP1 (intercept), DP3 (gating for "don't run X if no IFC label"), DP6 (continuation intent) |
| **Hypothetical AILANG agent SDK** | Speculative | Variable — depends on SDK's product positioning | If SDK aims for "claude-code-class agents", high. If "simple research agents", low. |
| **motoko_agent itself** | Production | High — all 6 decision points |
| **examples/runnable/ai_tool_loop.ail** | Demo | Low — canned dispatch, no intercepts |

**Reading**: motoko is *not* unique. Decision points 1, 3, 4, 6 (intercept, gating, custom handlers, continuation-intent) are likely shared by 2 of the 5 consumers above. Decision point 5 (backend split) IS motoko-specific (ohmy_pi is fork-local). Decision point 2 (parse-error feedback) is small enough to live inside the dispatch callback.

This pattern argues weakly toward Option B — but only if the AILANG team is willing to take on the API-design work AND the motoko team is willing to wait for that work to complete.

## Assumptions (the recommendation hinges on these)

The recommendation in §Recommendation is contingent on these assumptions. **If any assumption is wrong, the recommendation flips.**

1. **A.docparse-real**: docparse legal review (planned v0.18) WILL want at least 3 of the 6 decision points (intercept, gating, continuation-intent). If docparse's design instead lands a fundamentally different agent shape (pure-functional pipeline, not a tool-loop), this argument weakens.
2. **A.timeline**: motoko_agent can wait ~1 sprint (2 weeks) for upstream Option B before unblocking PR #4. If arni needs to ship migration THIS WEEK, Option A wins regardless of long-term cost.
3. **A.ailang-bandwidth**: The AILANG team has ~1 sprint of capacity for an upstream API extension between v0.16.0 and v0.17.0 freeze. If the runTools API needs to freeze before that capacity opens, Option B is impossible.
4. **A.compose-extensions-essential**: motoko's compose extensions (claimcheck, author_loop) are essential to the product, not vestigial. If arni is willing to drop them, Option C becomes viable.
5. **A.api-stability**: We're willing to ship `runTools` as v0 (breaking changes allowed in v0.x.x). If arni or other early consumers need stability guarantees on `runTools` already, Option B's signature breakage is harder.

## Recommendation (conditional)

**If all 5 assumptions hold: Option B (extend upstream `runTools` with hooks).**

Rationale:
- Highest axiom net (+6 vs +1 for A and +3 for C)
- Largest net LOC deletion (~600 LOC system-wide)
- Pre-empts re-implementation by docparse legal review (per A.docparse-real)
- Frames `runTools` as a **kernel** with **user-extensible hooks** rather than a one-shot loop driver — matches AILANG's broader philosophy of "small composable primitives" (A10)

**If A.timeline OR A.ailang-bandwidth fails: Option A (motoko keeps custom loop).**

Rationale:
- Unblocks PR #4 immediately with no upstream risk
- Future migration to Option B remains possible (motoko's custom loop becomes the prototype that informs the eventual upstream design)
- Trade-off: net LOC stays flat, no reusability for future consumers

**If A.compose-extensions-essential fails (arni drops the extensions): Option C (motoko simplifies).**

Rationale:
- Smallest motoko rewrite + zero upstream cost
- Clean answer to "is current runTools enough?" — yes, if you're willing to drop the features it doesn't model
- Most likely to be wrong long-term: another consumer (docparse) is likely to add equivalent extension features later, regenerating the same problem from the other side

**Hybrid: Option A short-term, Option B long-term.**

Land Option A in PR #4 immediately. Use motoko's custom loop as the validated prototype for Option B's eventual upstream design (after v0.17.0 freezes). Option B becomes a M-AGENT-COMPOSABILITY milestone targeting v0.18.0+ with the lessons motoko teaches about which hooks matter.

**This is my actual recommendation pending arni's input.** It avoids the upstream-design risk while preserving the long-term compositionality goal.

## Timeline (for the recommendation, per option)

**If Option A wins**:
- Week 1: motoko team writes `agent_loop.ail` mirroring rpc.ail's existing structure but calling `step` directly. Migrates rpc.ail to use it.
- Week 2: smoke test + finalize PR #4

**If Option B wins**:
- Sprint 1 (AILANG team): write M-RUNTOOLS-HOOKS design doc with full Conflict Surface analysis, sprint plan, golden tests, and 5 ADT definitions
- Sprint 2-3 (AILANG team): implement, test, ship as v0.17.x or v0.18.0
- Sprint 4 (motoko team): migrate to extended `runTools`; PR #4 lands

**If Option C wins**:
- Week 1: motoko team writes deprecation plan for dropped features; gathers user feedback
- Week 2-3: implements simplified loop + migrates compose extensions to post-loop validation
- Week 4: PR #4 lands with feature regressions documented

**If Hybrid wins (recommended)**:
- Week 1-2: Option A in PR #4 (ships immediately)
- Sprint 2+: Option B as a separate milestone using motoko's loop as the validated prototype

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| arni's compose extensions are tightly coupled to fork-only Go code we don't see | Medium | High (could invalidate any of A/B/C) | This doc + the §6 decision points catalog is designed to surface that — review it with arni and have him flag anything we missed |
| Option B's hook design over-fits motoko, doesn't generalise to docparse | Medium | Medium (would need redesign in v0.18+) | Defer Option B until docparse design exists; lean toward Hybrid |
| Option A becomes permanent and Option B never happens | Medium | Low (motoko works either way; just no reusability) | Document Option B's design as part of Hybrid so the path is preserved |
| `runTools` API change breaks early external adopters | Low | High | Ship `runToolsWithConfig` as a sibling, deprecate `runTools` over 2 versions |
| Conflict-surface analysis misses something | Low | Medium | The Conflict Surface section above explicitly enumerates affected positions; review by AILANG team can challenge |

## Related Documents

**Companion docs**:
- [M-AI-TOOL-LOOP design](../implemented/v0_17_x/m-ai-tool-loop.md) — the milestone that shipped `runTools` and `step` (the API surface this doc proposes extending)
- [M-AI-CALL-STREAM-HELPER design](../implemented/v0_15_1/m-ai-call-stream-helper.md) — companion streaming-side migration; established the "ship simple primitive, extend later if needed" pattern
- [M-AI-PROVIDER-CONFIG design](../implemented/v0_15_0/m-ai-provider-config.md) — the upstream change that motoko's PR #3 consumes

**Motoko-side docs**:
- [motoko-agent v0.15.0 migration](../motoko-agent-v0.15.0-migration.md) — the streaming-migration plan; PR #3 implementation
- [motoko-agent tool-loop migration](https://github.com/sunholo-voight-kampff/motoko_agent/blob/ailang-tool-loop-migration/design_docs/planned/ailang-tool-loop-migration.md) — sprint plan for PR #4; references this doc for the M3 design call

**External consumer signals**:
- [arniwesth/motoko_agent#3](https://github.com/arniwesth/motoko_agent/pull/3) — streaming migration (ready for review)
- [arniwesth/motoko_agent#4](https://github.com/arniwesth/motoko_agent/pull/4) — tool-loop migration (draft, blocked on this doc)
- [`docparse legal review`](https://github.com/sunholo/ailang-parse) (planned v0.18) — second potential consumer of hook-rich `runTools`

**Process docs**:
- [Conflict Surface design-doc gate](../../../.claude/skills/design-doc-creator/resources/design_doc_structure.md) — the gate this doc tests against (Option B specifically)
- [Regression Surface evaluation rubric](../../../.claude/skills/sprint-evaluator/resources/scoring_rubric.md) — the evaluator-side complement

## Future Work

If Option B (or Hybrid) wins, follow-ups include:

1. **M-AGENT-COMPOSABILITY** (v1.0+ axis): formalize "agent loops as user-extensible kernels" as an AILANG design principle. Document the hook-rich `runTools` shape as the canonical pattern.
2. **Tracing for hook decisions**: each hook fires a trace event (`AI/hook[on_response]`, `AI/hook[policy]`, etc.) so the agent loop is fully replayable.
3. **Cost-aware policy hook**: extend `policy` hook to receive a budget snapshot, enabling "deny call if it would exceed remaining budget" patterns (links to v0.15.1's M-EVAL-COST-AND-SPEED-BUDGETS).
4. **Per-hook capability scoping**: hooks could declare narrower effect rows than the parent runTools call, enabling fine-grained capability auditing.

---

**Document created**: 2026-05-05
**Last updated**: 2026-05-05
**Author**: Claude (with input from arni's PR #4 thread)
**Status**: Awaiting joint review (AILANG team + arni)
