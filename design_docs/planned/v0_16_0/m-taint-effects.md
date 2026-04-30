# M-TAINT-EFFECTS: Taint-Labelled Effect Rows

**Status**: Planned
**Target**: v0.16.0
**Priority**: P1 — Medium (strategic; differentiator vs Python guardians)
**Estimated**: ~60 hours (~2 sprints)
**Dependencies**:
  - [M-SMT-CROSS-MODULE-TYPES](../v0_13_0/m-smt-cross-module-types.md) — must land first; cross-module taint depends on cross-module Z3 type resolution
  - [M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md) — adjacent design (parameterised effects `!{E[mode=...]}`); labels here reuse the same row-parameter machinery
  - Tier 1 demo (already landed): [examples/runnable/contracts/inbox_injection.ail](../../../examples/runnable/contracts/inbox_injection.ail)

**Inspirations:**
- Erik Meijer, *Guardians of the Agents: Formal Verification of AI Workflows*, CACM January 2026
- [metareflection/guardians](https://github.com/metareflection/guardians) — ~1900-LOC Python reference implementation (taint analysis + security automata + Z3)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to runtime semantics |
| A2: Replayability | 0 | No change to traces |
| A3: Effect Legibility | +1 | Effects now carry origin labels — a tainted value's source is visible in the type |
| A4: Explicit Authority | +1 | Sink annotations make "where dangerous data may flow" a typed capability constraint |
| A5: Bounded Verification | +1 | Most taint flows resolve at unification time; Z3 only needed for value-level constraints |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents can read a function's effect row and see what taint it propagates without inspecting the body |
| A8: Minimal Syntax | 0 | Adds `<label>` annotation on effect names; localised to effect rows |
| A9: Cost Visibility | 0 | No runtime cost change |
| A10: Composability | +1 | Labels are row-polymorphic; compose with M-EFFECT-REFINEMENT modes/scopes |
| A11: Structured Failure | +1 | Taint violations are typed errors with source/sink trace, not opaque runtime panics |
| A12: System Boundary | +1 | Sink boundaries (`Net<external>`) are typed system boundaries |

**Net Score: +7** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): Strengthens effect legibility, doesn't weaken it
- [x] A4 (Authority): Labels are typed; no ambient flow
- [x] A7 (Machines First): Pure type-level reasoning; agents can decide statically

## Problem Statement

AILANG ships with an unusually strong static-verification story for an agent host language: row-polymorphic effects + Z3-backed `requires`/`ensures`. But agent workflows expose a class of attack — prompt injection — that today's effect rows cannot describe. A function declared `! {Net}` says it talks to the network; it does not say *whose data* reaches the wire. The same gap underlies SQL injection (we already know how to fix it: separate code from data) and the inbox-injection attack from Erik Meijer's *Guardians of the Agents* (CACM Jan 2026).

The Tier 1 demo at [`examples/runnable/contracts/inbox_injection.ail`](../../../examples/runnable/contracts/inbox_injection.ail) shows that the **Z3 half** of the guardians architecture works today: `ailang verify` catches both attack variants with concrete counterexamples, using string-equality and `endsWith` contracts as a stand-in for taint propagation. What it cannot do — and why a real workflow author would still need a Python guardians sidecar — is express *origin* in the type. The demo's safe-vs-injected distinction relies on a literal contract `result.body == "[summary]"`; in a real workflow the body would be a derived string and the contract would have to be replicated by hand at every call site.

**Current State:**
- Effect rows carry only effect names: `! {Net, FS, IO}`. Two distinct sources of FS data are indistinguishable in the type.
- A function returning a value derived from `fetchMail()` has the same type as one returning a value derived from `readConfig()`. The unifier sees no difference.
- The Tier 1 demo verifies 5 functions: 3 verified, 2 violations (`injectedForward`, `externalLeak`) — but only because the bodies are simple enough to inline as string equalities.
- Cross-module taint is doubly blocked: even if labels existed, [M-SMT-CROSS-MODULE-TYPES](../v0_13_0/m-smt-cross-module-types.md) shows Z3 currently skips functions that import record/ADT types.

**Impact:**
- AILANG cannot make the "agent host language with built-in injection safety" claim that the guardians paper sets up perfectly for us.
- Agents writing AILANG today have no way to encode the policy "don't let email content flow to send_email" — the type system silently allows it.
- The marketing/eval angle (AILANG vs Python on prompt-injection benchmarks) cannot be tested at the language level; only at the contract level, which doesn't generalise.

## Goals

**Primary Goal:** Add origin labels to effect rows so that taint propagates through the type system: `fetchMail: () -> [Email] ! {FS<email>}`, `sendEmail: (string, string) -> () ! {Net<external>}`, with sink annotations forbidding labelled flows from reaching specific parameters — closing the remaining gap between AILANG's existing Z3 verifier and Erik Meijer's full guardians architecture.

**Success Metrics:**
- `inbox_injection.ail` rewritten using actual taint labels (not string-equality stand-ins) still verifies with the same outcomes (3 verified, 2 violations, equivalent counterexamples).
- At least one new prompt-injection benchmark in `ai-coding-lang-bench` showing AILANG vs Python head-to-head — generated AILANG code is rejected by the type checker for the injected variant; Python equivalent is not.
- Effect-row labels propagate through `let`-bindings, function calls, and across modules (assuming M-SMT-CROSS-MODULE-TYPES has landed).
- Zero changes required to existing `! {Net}`-style code: unlabelled effects remain valid (label `_` is the identity).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Label syntax: `! {Net<external>}` vs `! {Net[label=external]}` | Affects every effect annotation in user-written code; must compose cleanly with M-EFFECT-REFINEMENT's `[mode=..., scope=...]` parameters | human | design | high |
| Label-row algebra: free labels unify like row variables, or labels are a separate lattice | Determines whether `Net<a>` and `Net<b>` are unifiable (row-poly) or whether labels join (lattice) | human | design | high |
| Where labels surface to Z3: type-level only, or also as SMT-LIB constraints | Pure type-level keeps verification cheap; SMT round-trip lets us enforce numeric/string predicates *on* labels | human | design | med |
| Sink annotation: function-level keyword (`sink`) vs contract clause (`forbids { Net<external> reaches body }`) | Affects sink ergonomics and how guardians-style policies map to AILANG | human | design | med |
| Default labels: do all effects implicitly carry an anonymous label, or only annotated ones? | Implicit labels propagate everywhere (more safety, more friction); explicit-only is opt-in | agent | compile | low |
| Built-in label vocabulary: ship a stdlib of common labels (`<external>`, `<user>`, `<email>`) or leave entirely user-defined? | Affects whether the eval benchmark needs to import labels or define its own | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Label syntax decision (`<...>` vs `[label=...]`)
- [ ] Label-row algebra (row-poly vs lattice)
- [ ] Sink annotation form (keyword vs contract clause)

## Solution Design

### Overview

Effect rows gain an optional label parameter on each effect name. The unifier treats labels as a second row variable (one for effect *presence*, one for effect *origin*), so existing row-polymorphism does most of the work. Sinks are declared with a contract-like clause that forbids specific labels from reaching specific parameters. The Z3 half stays unchanged for value-level reasoning; labels are checked at the unification layer with no SMT round-trip in the common case.

The framing for the rest of this doc: **the Z3 half is already in the box.** What this milestone adds is the type-system half that makes existing Z3 reasoning *useful* in the agent-workflow setting.

### Architecture

**Components:**

1. **Label syntax & AST** — extend `EffectRow` AST node with an optional label field per effect; parser accepts `! {Net<external>}` syntax. (`internal/ast/types.go`, `internal/parser/parser_effects.go`)

2. **Type-system extension** — extend `Effect` representation to carry a label term (variable, constant, or `_`). Unifier treats labels with the same rules as row variables: free labels unify, distinct constants conflict. (`internal/types/effects.go`, `internal/types/unify.go`)

3. **Label propagation through inference** — when a function is applied, its return type's effect row labels propagate to the binding site. `let x = fetchMail() in sendEmail(to, x)` infers `x: [Email] ! {FS<email>}` and refuses to unify with `sendEmail`'s sink-annotated body parameter. (`internal/types/typechecker.go`)

4. **Sink declaration** — function signatures may declare `forbids { LABEL on PARAM }` clauses; the type checker ensures the resolved type of `PARAM` does not carry `LABEL`. (`internal/parser/parser_contracts.go`, `internal/types/sink_check.go`)

5. **Cross-module taint** — depends on M-SMT-CROSS-MODULE-TYPES; once that ships, label propagation works across module boundaries via the iface.

6. **Z3 surface (optional, Phase 3)** — for label predicates expressible in SMT (`label != "external"`), emit them as constraints alongside existing `requires`/`ensures`. Most checks resolve at unification and never reach Z3.

### Implementation Plan

**Phase 1: AST + parser + single-module type-level checking** (~24 hours)
- [ ] Extend `EffectRow` AST with optional label field per effect
- [ ] Parser support for `! {Net<label>}` syntax
- [ ] Unifier: labels unify like row variables, conflict on distinct constants
- [ ] `forbids { ... }` contract clause parsing
- [ ] Sink check after type inference
- [ ] Single-module test: rewrite `inbox_injection.ail` using labels, confirm same outcomes
- [ ] No-regressions: existing `! {Net}` code without labels continues to work

**Phase 2: Cross-module + iface plumbing** (~16 hours)
- [ ] Iface serialises labelled effect rows
- [ ] Cross-module unification preserves labels
- [ ] Verify after M-SMT-CROSS-MODULE-TYPES has landed
- [ ] Multi-module test: split inbox demo across `tools/mail.ail` and `app/inbox.ail`

**Phase 3: Eval + benchmark + docs** (~16 hours)
- [ ] Add prompt-injection benchmark to `ai-coding-lang-bench` (AILANG vs Python; same scenario, same evaluator model)
- [ ] Update `ailang prompt` to teach agents about sink annotations
- [ ] Blog post draft: "Half of guardians is already a language feature"
- [ ] Optional: SMT-level label predicates (defer to v0.17 if Phase 1+2 stretch)

**Phase 4 (deferred to v0.17 or later):** Z3 round-trip for label predicates; built-in label stdlib; security automata (separate milestone — see Non-Goals).

### Files to Modify/Create

**New files:**
- `internal/types/labels.go` — Label representation, lattice/row algebra (~250 LOC)
- `internal/types/sink_check.go` — Post-inference sink validation (~200 LOC)
- `examples/runnable/contracts/inbox_injection_v2.ail` — Rewritten demo using real labels (~80 LOC)
- `benchmarks/prompt_injection/` — Eval benchmark (~120 LOC across `.ail` + `.py`)

**Modified files:**
- `internal/ast/types.go` — Extend `EffectRow` with labels (~30 LOC)
- `internal/parser/parser_effects.go` — Parse `<label>` syntax (~60 LOC)
- `internal/parser/parser_contracts.go` — Parse `forbids { ... }` clause (~40 LOC)
- `internal/types/effects.go` — Effect carries label term (~80 LOC)
- `internal/types/unify.go` — Label unification (~120 LOC)
- `internal/types/typechecker.go` — Propagate labels through inference (~60 LOC)
- `internal/iface/builder.go` — Serialise labels in interface files (~40 LOC)
- `cmd/ailang/prompts/v0.16.0.md` — Teach labels and sinks (~50 LOC)

## Examples

### Example 1: The inbox-injection demo with real labels

**Before (Tier 1 — string-equality stand-in, current state of the demo):**
```ailang
export pure func safeForward(rawEmail: string, recipient: string) -> SendAction ! {}
requires { endsWith(recipient, "@company.com") }
ensures  { endsWith(result.to, "@company.com"), result.body == "[summary]" }
{
  { to: recipient, body: summarize(rawEmail) }
}
```
The `result.body == "[summary]"` postcondition substitutes for taint propagation — Z3 proves the literal, not the principle.

**After (with taint labels):**
```ailang
export func fetchMail(folder: string) -> [Email] ! {FS<email>} { ... }

export func sendEmail(to: string, body: string) -> () ! {Net<external>}
forbids { <email> on body }
{ ... }

export func safeForward(rawEmail: Email, recipient: string) -> () ! {FS<email>, Net<external>}
{
  let summary = summarize(rawEmail.body);   -- summarize: (string) -> string ! {} (no taint propagated; pure transform that drops the label)
  sendEmail(recipient, summary)              -- type-checks: summary has no <email> label
}

export func injectedForward(rawEmail: Email, recipient: string) -> () ! {FS<email>, Net<external>}
{
  sendEmail(recipient, rawEmail.body)        -- TYPE ERROR: rawEmail.body carries <email>, sendEmail.body forbids <email>
}
```
The unifier rejects `injectedForward` at compile time, with a typed error pointing to the source-sink path. No Z3 round-trip needed.

### Example 2: SQL effect alignment (foreshadowing v0.17)

**Today (no SQL effect):**
```ailang
-- No way to express "this query parameter must be parameterised, not interpolated"
```

**With taint effects + future SQL effect:**
```ailang
export func userInput() -> string ! {IO<user>} { ... }

export func runQuery(template: string, params: [string]) -> [Row] ! {Db<param>}
forbids { <user> on template }    -- raw user input must not become SQL text
{ ... }

let q = userInput()
runQuery("SELECT * FROM t WHERE x = ${q}", [])  -- TYPE ERROR: <user> reached template
runQuery("SELECT * FROM t WHERE x = $1", [q])    -- OK: <user> reaches params, which is allowed
```
This is the same machine. SymRef (guardians) ≅ parameterised query placeholder ≅ AILANG `let`-bound labelled value.

## Success Criteria

- [ ] `inbox_injection_v2.ail` (real labels, no string-equality stand-ins) verifies with same outcomes as Tier 1: 3 verified, 2 violations
- [ ] Type checker rejects `injectedForward` at unification time with a structured error containing source label, sink, and binding chain
- [ ] At least one prompt-injection benchmark in `ai-coding-lang-bench` shows AILANG catching the attack at the type level where Python equivalent does not
- [ ] Cross-module taint works (post-M-SMT-CROSS-MODULE-TYPES): label flows from `tools/mail.ail` to `app/inbox.ail` correctly
- [ ] All existing examples (no labels) continue to compile and verify with no changes
- [ ] All tests passing
- [ ] `ailang prompt` updated with sink annotations and label propagation rules
- [ ] CHANGELOG entry referencing this design doc and the Tier 1 demo

## Testing Strategy

**Unit tests:**
- Label unification: free × free, free × const, const × const (same), const × const (different — must fail)
- Label propagation through `let`, function application, match arms
- Sink-clause violation detection with structured error
- Iface round-trip: label survives serialisation

**Integration tests:**
- `inbox_injection_v2.ail` end-to-end through `ailang verify`
- Cross-module label flow through a two-module test fixture
- No-regression: every existing example in `examples/runnable/contracts/` still produces the same `verify` outcome

**Benchmark / eval:**
- New `prompt_injection` task in `ai-coding-lang-bench`: prompt the model with a guardians-style scenario, score whether the generated AILANG type-checks under the sink policy
- Compare against Python with `# type: ignore` (no taint), Python + guardians DSL, AILANG with labels

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Anonymous label syntax** (`! {Net<_>}` vs `! {Net}`) — agent may choose, document the chosen form
- **Error message phrasing for sink violations** — agent may choose; structured payload is what matters
- **Whether `summarize` should drop or preserve labels by default** — agent may choose a default for pure functions; designer is reviewable

## Non-Goals

**Not attempted in this feature:**
- **Security automata** (state-machine on tool-call sequences). Guardians' second pillar. Worth a separate milestone after taint lands; a small fraction of real attacks need automaton expressivity that taint can't cover.
- **Frame conditions over mutable state** (the McCarthy/Hayes "all unchanged files" form). Out of scope until AILANG has mutable-state semantics worth verifying. Taint covers the parts of "frame" that real workflows actually use.
- **A built-in tool registry / executor.** AILANG remains a language; the runtime side stays user-built. Guardians-the-library is roughly 60% executor; we are not porting that 60%.
- **Generic dependent types or refinement types.** Labels are a constrained extension, not a foothold for general refinement-type machinery.

## Timeline

**Sprint 1** (~24 hours):
- Phase 1 — AST, parser, unifier, sink check, single-module demo

**Sprint 2** (~16 hours):
- Phase 2 — iface, cross-module, multi-module test (gated on M-SMT-CROSS-MODULE-TYPES)

**Sprint 3** (~16 hours):
- Phase 3 — benchmark, prompt updates, blog post, docs

**Total: ~56–60 hours across ~3 sprints**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Label-row algebra interacts badly with M-EFFECT-REFINEMENT's `[mode=...]` parameters | High | Coordinate with M-EFFECT-REFINEMENT author before Phase 1 freeze; share the same row-parameter machinery |
| M-SMT-CROSS-MODULE-TYPES slips, blocking Phase 2 | Med | Ship Phase 1 (single-module) standalone; deliver value even without cross-module |
| Adding labels breaks existing user code with non-trivial effects | High | Default: unlabelled effects = anonymous label; existing code typechecks unchanged. Add a regression suite over every example before Phase 1 starts. |
| Prompt-injection benchmark is gameable (model just refuses to write any networking code) | Med | Two scenarios: a "safe" workflow that *should* type-check, and an "injected" workflow that *should not* — both must produce the right verdict. |
| Sink syntax (`forbids { ... }`) collides with future contract syntax | Low | Reserve the keyword; design freeze must rule on syntax before parser work begins |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-bug-effect-checker-conflation.md](../../implemented/v0_6_2/m-bug-effect-checker-conflation.md) (0.35) — prior bug in effect-row unification; instructive precedent
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) (0.34) — capability layer that effect-row labels complement

**Planned (check for overlap):**
- [design_docs/planned/v1_0_0/m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) (0.34) — parameterised effects `!{E[mode=..., scope=...]}`; this milestone reuses its row-parameter machinery
- [design_docs/planned/v0_13_0/m-smt-cross-module-types.md](../v0_13_0/m-smt-cross-module-types.md) — required dependency for Phase 2
- [design_docs/planned/v0_13_0/20251013_auto_caps_capability_inference.md](../v0_13_0/20251013_auto_caps_capability_inference.md) (0.30) — capability inference; orthogonal but related

**External:**
- Erik Meijer, *Guardians of the Agents: Formal Verification of AI Workflows*, [CACM January 2026](https://cacm.acm.org/practice/guardians-of-the-agents/)
- [metareflection/guardians](https://github.com/metareflection/guardians) — Python reference implementation

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Tier 1 demo](../../../examples/runnable/contracts/inbox_injection.ail) — Inbox-injection caught with current contracts (proof the Z3 half works today)
- [SMT codegen fix](../../../internal/smt/codegen.go) — Pre-existing `ensures { a, b }` bug found and fixed during demo authoring (April 2026)

## Future Work

- **Security automata** (post-taint, separate milestone) — state-machine policies over tool-call sequences. Smaller scope than this doc; needs taint as a foundation.
- **Label-aware Z3 predicates** — for properties like "no label deeper than depth N", round-trip labels into SMT.
- **SQL effect with parameterisation** — natural extension; same source/sink mechanism. Likely v0.17 or v0.18.
- **Stdlib of canonical labels** — `<user>`, `<external>`, `<email>`, `<sql-text>`, etc. — once the lattice settles.
- **Compatibility shim with metareflection/guardians** — emit guardians-compatible policy JSON from AILANG type-checked workflows, so users can verify AILANG-generated plans against their existing Python policy fleet during migration.

---

**Document created**: 2026-04-30
**Last updated**: 2026-04-30
