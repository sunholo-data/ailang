# M-TAINT-TYPES: Value-Labelled Information-Flow Types

**Status**: Implemented (2026-04-30)
**Target**: v0.16.0
**Priority**: P1 — Medium (strategic; differentiator vs Python guardians)
**Estimated**: ~70 hours (~3 sprints)
**Dependencies**:
  - [M-SMT-CROSS-MODULE-TYPES](../v0_13_0/m-smt-cross-module-types.md) — must land first; cross-module label flow depends on cross-module Z3 type resolution
  - [M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md) — adjacent design; shares row-parameter machinery, but **labels and effect modes are orthogonal axes** (see §"Effects vs Labels" below)
  - Tier 1 demo (already landed): [examples/runnable/contracts/inbox_injection.ail](../../../examples/runnable/contracts/inbox_injection.ail)

**Inspirations:**
- Erik Meijer, *Guardians of the Agents: Formal Verification of AI Workflows*, CACM January 2026
- [metareflection/guardians](https://github.com/metareflection/guardians) — ~1900-LOC Python reference implementation
- Volpano & Smith, *A Sound Type System for Secure Flow Analysis* (1996); Pottier & Simonet, *FlowCaml*; Myers et al., *Jif* — the standard IFC type-system literature this milestone tracks

**Note:** This doc supersedes the v1 draft *M-TAINT-EFFECTS*. Round-2 review surfaced three corrections that reshape the abstraction: (1) labels form a join-semilattice, not row-poly; (2) declassification must be explicit and capability-gated; (3) labels live on **values**, not on effects. The v2 design here applies all three.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to runtime semantics |
| A2: Replayability | 0 | No change to traces |
| A3: Effect/Data-Flow Legibility | +1 | Provenance becomes visible in the type — agents read where data came from without reading the body |
| A4: Explicit Authority | +2 | Declassification is capability-gated (`! {Declassify}`); raising/lowering labels requires explicit authority |
| A5: Bounded Verification | +1 | Most flow checks resolve at unification + lattice join; Z3 only for value-level constraints |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents reason from `(signature, effect row, label)` alone — no body inspection |
| A8: Minimal Syntax | 0 | Adds `T<label>` annotation on value types and `T{constraint}` refinement; localised |
| A9: Cost Visibility | 0 | No runtime cost change |
| A10: Composability | +1 | Lattice join propagates through all pure operations uniformly |
| A11: Structured Failure | +1 | Sink violations are typed errors carrying source label, sink, and binding chain |
| A12: System Boundary | +1 | Sink-typed parameters (`string{not external}`) are typed system boundaries |

**Net Score: +8** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): Strengthens data-flow legibility, doesn't weaken it
- [x] A4 (Authority): Declassification is the *only* label-lowering operation, and it requires capability — no ambient laundering
- [x] A7 (Machines First): Pure type-level reasoning; agents decide statically

## Problem Statement

AILANG ships with a strong static-verification story for an agent host language: row-polymorphic effects + Z3-backed `requires`/`ensures`. But agent workflows expose a class of attack — prompt injection — that today's effect rows cannot describe. A function declared `! {Net}` says it talks to the network; it does not say *whose data* reaches the wire. The same gap underlies SQL injection (we already know how to fix it: separate code from data) and the inbox-injection attack from Erik Meijer's *Guardians of the Agents* (CACM Jan 2026).

The Tier 1 demo at [`examples/runnable/contracts/inbox_injection.ail`](../../../examples/runnable/contracts/inbox_injection.ail) shows the **Z3 half** of the guardians architecture works today: `ailang verify` catches both attack variants with concrete counterexamples, using string-equality contracts as a stand-in for taint propagation. What it cannot do is express *origin* in the type — the demo's safe-vs-injected distinction relies on a literal contract `result.body == "[summary]"`, which doesn't generalise.

What's needed is information-flow control (IFC): every value carries a *provenance label* describing what sources contributed to it; operations propagate labels by lattice join; sinks reject values whose labels violate a flow constraint; and the *only* way to lower a label is an explicit, capability-gated declassification.

**Current State:**
- Effect rows carry only effect names: `! {Net, FS, IO}`. Two values derived from different sources are indistinguishable in the type.
- The Tier 1 demo verifies 5 functions: 3 verified, 2 violations (`injectedForward`, `externalLeak`) — but only because bodies are simple enough to inline as string equalities.
- Cross-module taint is doubly blocked: even if labels existed, [M-SMT-CROSS-MODULE-TYPES](../v0_13_0/m-smt-cross-module-types.md) shows Z3 currently skips functions that import record/ADT types.

**Impact:**
- AILANG cannot make the "agent host language with built-in injection safety" claim that the guardians paper sets up perfectly for us.
- Agents writing AILANG today have no way to encode the policy "don't let email content flow to send_email" — the type system silently allows it.
- The marketing/eval angle (AILANG vs Python on prompt-injection benchmarks) cannot be tested at the language level.

## Goals

**Primary Goal:** Add value-level information-flow labels to AILANG types, propagated by lattice join through pure computation, lowered only via capability-gated declassification, with sink constraints expressible as type refinements — closing the remaining gap between AILANG's existing Z3 verifier and Erik Meijer's full guardians architecture.

**Success Metrics:**
- `inbox_injection.ail` rewritten using actual labels (no string-equality stand-ins) still verifies with the same outcomes (3 verified, 2 violations, equivalent counterexamples).
- At least one prompt-injection benchmark in `ai-coding-lang-bench` shows AILANG rejecting the attack at the type level where Python equivalent does not.
- Labels propagate through `let`-bindings, function calls, match arms, record field access, and across modules (assuming M-SMT-CROSS-MODULE-TYPES has landed).
- Zero changes required to existing unlabelled code: missing label = `⊥` (untainted); existing code typechecks unchanged.
- A pure function cannot launder taint. The only operation that lowers labels is `declassify`, which carries `! {Declassify}` and requires a capability.

## Effects vs Labels (orthogonality)

This is load-bearing and worth a section of its own.

> **Effect refinements describe how an operation behaves; taint labels describe where data came from. They share row-parameter machinery, but they are separate semantic axes.**

| Axis | What it captures | Example |
|---|---|---|
| **Effect names** | Which side-effect categories the function performs | `! {Net, FS}` |
| **Effect modes** ([M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md)) | Replay contract / scope of an effect | `! {Rand[mode=crypto], Clock[mode=pinned]}` |
| **Value labels** (this doc) | Provenance of data values | `Email<email>`, `string{not external}` |

The reviewer's diagnosis was right: the v1 draft put labels on effects (`! {FS<email>}`), which conflated *what the function does* with *what the data is*. v2 puts labels on the **types of values flowing through the function**, leaving effects to describe operations alone.

## High-Impact Decisions

All four design-freeze items resolved 2026-04-30 after a "from an AI agent's perspective" review pass — see "AI-friendliness rationale" notes per row.

| Decision | Locked choice | Rationale (AI-perspective) |
|----------|---------------|---|
| Label syntax on types | **`T<label>` for single-label sugar; `T{not LABEL}` for sink refinement.** `<>` is unused at the AILANG type level today, so the parser disambiguates trivially. The refinement keyword is `not` (ASCII), NOT the `¬` symbol — agent tokenizers handle keywords reliably and the symbol form was the most fragile part of the v2 draft. | Composes cleanly with existing `T[a]` generics and `! {E[mode=…]}` effect parameters (different brackets, different axes). `not` is self-documenting and copies losslessly across all prompt corpora. |
| Lattice element encoding | **Free-form set of label constants** — labels are arbitrary lowercase identifiers. No predeclared kinds in MVP. | Smallest surface area; agents can introduce new labels without reading a schema. Risk of label drift is mitigated by a naming convention added to the v0.16 prompt (lowercase, kebab-case for compounds: `<email>`, `<user-input>`, `<sql-text>`). Structured kinds remain a future-work option. |
| Declassification primitive | **No special primitive — declassification is "any function whose effect row contains `Declassify`".** A function may freely change a value's label between input and output if and only if `! {Declassify}` is declared. | Reuses two concepts agents already know (effect rows + capability discipline) instead of introducing a new builtin syntax. Audit story: `grep -r "! {Declassify}"`, identical shape to auditing any other dangerous capability. |
| `Declassify` effect | **Separate effect name `Declassify`**, sibling to `Net`/`FS`/`IO`, NOT a `[mode=…]` parameter on an existing effect. | Declassification is an authority capability; M-EFFECT-REFINEMENT modes are replay-contract distinctions. Different axes — keeping them separate respects the same orthogonality argument that motivated putting labels on values rather than effects. |
| Default label | **Missing label = `⊥` (untainted)** — agent may choose, locked here for clarity. | Preserves backwards compatibility: every existing AILANG file typechecks unchanged. Forcing opt-in on every type would explode the migration surface. |
| Refinement grammar (MVP scope) | **`not IDENT` only.** Conjunctions, disjunctions, and label-equality predicates are deferred to future work. | Hard-restricting the parser prevents agents from generating richer predicates the verifier will silently fail to enforce. Future grammar lands when the verifier supports it, not before. |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Type-label syntax — `T<label>` + `T{not LABEL}` (locked 2026-04-30)
- [x] Lattice element encoding — free-form constants, naming convention in prompt (locked 2026-04-30)
- [x] Declassification primitive form — function-with-`! {Declassify}` capability, no special primitive (locked 2026-04-30)
- [x] `Declassify` effect representation — sibling effect name (locked 2026-04-30)
- [x] MVP refinement grammar — `not IDENT` only (locked 2026-04-30)

## Solution Design

### Overview

Types gain an optional label annotation: `Email<email>`, `string<user>`, `int<⊥>`. Labels form a join-semilattice; every operation that combines values produces an output whose label is the **join** of the inputs. Function parameters may carry sink refinements (`string{not email}`); the type checker enforces the constraint at call sites against the resolved label of the argument. The only way to *lower* a label is to call a function that declares `! {Declassify}` in its effect row — declassification is a capability, not a special primitive.

The Z3 half stays unchanged for value-level reasoning (`requires`/`ensures` over numeric/string content). The label half is purely structural: it resolves at unification + lattice join, with no SMT round-trip in the common case.

The framing is unchanged from v1: **the Z3 half is already in the box.** v2 just gets the type-system half right.

### Formal Semantics

This section is intentionally short. Full mechanisation is out of scope; this is enough to pin the design and let implementers check rules against it.

**Label lattice.**
```
L ::= ⊥                  (untainted)
    | ℓ                  (constant, e.g. <email>, <user>, <external>)
    | α                  (label variable)
    | L₁ ⊔ L₂            (join)

⊔ is associative, commutative, idempotent.
⊥ ⊔ L = L.
ℓ₁ ⊔ ℓ₂ where ℓ₁ ≠ ℓ₂ is the formal join (kept symbolic; not collapsed).
```

**Labelled types.**
```
τ ::= base                (int, string, etc., implicit ⊥)
    | τ^L                 (type at label L)
    | τ{φ}                (refinement, where φ is a predicate over labels)
    | (τ₁ -> τ₂) ! ε      (function with effect row ε)
    | record / variant / list (label propagates structurally)
```

**Selected typing rules.**
```
(VAR)        Γ(x) = τ^L                                  Γ ⊢ x : τ^L

(APP-PURE)   Γ ⊢ f : (τ₁^L₁ -> τ₂^L₂) ! ε
             Γ ⊢ a : τ₁^L_a
             L_a ⊑ L₁                                    [or L_a ⊑ φ when φ is a refinement]
             ──────────────────────────────────────────
             Γ ⊢ f a : τ₂^(L₂ ⊔ L_a) ! ε

(JOIN)       Any pure operation combining values v₁^L₁ … vₙ^Lₙ
             produces a result at label L₁ ⊔ … ⊔ Lₙ.

(DECLASS)    Γ ⊢ f : (τ^L -> τ^L') ! ε    Declassify ∈ ε
             Γ ⊢ x : τ^L
             ──────────────────────────────────────────
             Γ ⊢ f x : τ^L' ! ε

(SINK)       At any call where the parameter's refinement is τ{not ℓ},
             the argument's label L must satisfy ℓ ⊄ L
             (i.e. ℓ is not in the join). Otherwise: type error.
```

**Hard rule:** No typing rule lowers a label except via a function whose effect row contains `Declassify`. Pure functions can only preserve or join labels — never erase them. This is the property that makes `sanitize = identity` impossible as a bypass: an identity function with effect row `! {}` cannot change its input's label.

### Architecture

**Components:**

1. **Label lattice** — `L` representation, join, ordering, predicate evaluation. `internal/types/labels.go`
2. **Labelled type AST + parser** — `T<label>` and `T{predicate}` syntax; AST extension to carry labels on every type node. `internal/ast/types.go`, `internal/parser/parser_types.go`
3. **Inference + propagation** — every pure operation joins input labels into the output. Match arms join their patterns. Records/variants propagate labels structurally. `internal/types/typechecker.go`
4. **Sink check** — at function application, refinement predicates on parameters are evaluated against the argument's resolved label. `internal/types/sink_check.go`
5. **Declassify-effect capability check** — type checker rule (DECLASS): a function may change its input's label between input and output type if and only if its effect row contains `Declassify`. No special primitive; reuses existing effect-row + capability machinery. `internal/types/declassify.go` (the file holds the rule + the capability registration, ~80 LOC).
6. **Cross-module label flow** — iface serialises labels on exported types. Depends on M-SMT-CROSS-MODULE-TYPES. `internal/iface/builder.go`

### Implementation Plan

**Phase 1: Lattice + AST + parser + single-module type-level checking** (~28 hours)
- [ ] Label lattice (`internal/types/labels.go`) — constants, vars, join
- [ ] AST extension: every type node carries an optional `Label` field
- [ ] Parser: `T<label>` (label sugar) and `T{not LABEL}` (sink refinement) — MVP grammar restricts the refinement body to a single `not IDENT`
- [ ] Inference: pure operations join labels into the output
- [ ] Sink-check pass: refinement violations produce structured errors carrying source label, sink, and binding chain
- [ ] (DECLASS) typing rule: a function may change a value's label between its input and output type if and only if its effect row contains `Declassify`. Pure functions can only preserve or join labels. **No special primitive.**
- [ ] `Declassify` capability registered as a sibling effect alongside `Net`, `FS`, `IO`
- [ ] Single-module test: rewrite `inbox_injection.ail` using labels, confirm same outcomes
- [ ] No-regressions: every existing example continues to typecheck unchanged (missing label = `⊥`)

**Phase 2: Cross-module + iface plumbing** (~16 hours)
- [ ] Iface serialises labels on exported types
- [ ] Cross-module unification preserves labels
- [ ] Verify after M-SMT-CROSS-MODULE-TYPES has landed (✅ shipped v0.14.3)
- [ ] Multi-module test: split inbox demo across `tools/mail.ail` (sources) and `app/inbox.ail` (sink)

**Phase 3: Eval + benchmark + docs** (~16 hours)
- [ ] New prompt-injection benchmark in `ai-coding-lang-bench`
- [ ] Update `ailang prompt` to teach labels, refinements, declassification, AND the label-naming convention (lowercase, kebab-case for compounds — `<email>`, `<user-input>`, `<sql-text>`)
- [ ] Blog post draft: "Half of guardians is already a language feature"

**Phase 4 (deferred):** Richer refinement grammar (conjunctions, disjunctions, equality predicates) once the verifier supports them; lattice predicates that need Z3 (e.g. depth bounds on join expressions); built-in label stdlib; security automata (separate milestone, see Non-Goals).

### Files to Modify/Create

**New files:**
- `internal/types/labels.go` — Lattice representation, join, refinement evaluation (~280 LOC)
- `internal/types/sink_check.go` — Post-inference refinement check (~180 LOC)
- `internal/types/declassify.go` — DECLASS typing rule + Declassify capability registration (~80 LOC; no primitive emission)
- `examples/runnable/contracts/inbox_injection_v2.ail` — Rewritten demo using real labels (~100 LOC)
- `benchmarks/prompt_injection/` — Eval benchmark (~120 LOC across `.ail` + `.py`)

**Modified files:**
- `internal/ast/types.go` — Type nodes carry optional label (~40 LOC)
- `internal/parser/parser_types.go` — Parse `T<label>` and `T{not IDENT}` (~80 LOC)
- `internal/types/typechecker.go` — Join propagation through pure operations (~120 LOC)
- `internal/iface/builder.go` — Serialise labels on exported types (~50 LOC)
- `internal/effects/registry.go` — Register `Declassify` effect + capability (~30 LOC)
- `cmd/ailang/prompts/v0.16.0.md` — Teach labels, refinements, declassification (~70 LOC)

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
Z3 proves the literal, not the principle.

**After (with value-level labels):**
```ailang
-- Source: file-system reader, return type carries <email> label
export func fetchMail(folder: string) -> [Email<email>] ! {FS} { ... }

-- Pure transform: PRESERVES the input label by lattice join (cannot launder)
export pure func summarize(s: string) -> string { ... }
-- Inferred: ∀L. string<L> -> string<L>

-- Sink: parameter refinement excludes <email>
export func sendEmail(to: string, body: string{not email}) -> () ! {Net} { ... }

-- Trusted declassifier: a function whose effect row contains Declassify
-- may change its input's label between input and output. No special primitive.
export func sanitizeEmail(s: string<email>) -> string<sanitized> ! {Declassify} {
  -- ... actual sanitization logic returns a string ...
  -- The type checker authorises the label change because of `! {Declassify}`.
}

-- SAFE: explicit declassification before sink
export func safeForward(rawEmail: Email<email>, recipient: string) -> () ! {FS, Declassify, Net} {
  let summary = summarize(rawEmail.body);          -- summary : string<email>  (label preserved)
  let safeSummary = sanitizeEmail(summary);        -- safeSummary : string<sanitized>
  sendEmail(recipient, safeSummary)                -- typechecks: `not email` holds
}

-- ATTACK 1: forward without declassification
export func injectedForward(rawEmail: Email<email>, recipient: string) -> () ! {FS, Net} {
  let summary = summarize(rawEmail.body);          -- summary : string<email>
  sendEmail(recipient, summary)
  -- TYPE ERROR: argument has label <email>, parameter requires {not email}
  -- Source: rawEmail.body  ->  summarize  ->  sendEmail.body
}

-- ATTACK 2: try to launder via a "pure" wrapper — STILL FAILS
pure func laundered(s: string) -> string { s }    -- inferred: ∀L. string<L> -> string<L>
export func attemptLaunder(rawEmail: Email<email>, recipient: string) -> () ! {FS, Net} {
  sendEmail(recipient, laundered(rawEmail.body))
  -- TYPE ERROR: laundered preserves the label; <email> still on the argument
}
```

The `summarize` function cannot drop the label. The only path to a `<sanitized>`-labelled string is through a function whose effect row contains `Declassify` (here `sanitizeEmail`). There is no `sanitize = identity` bypass — an identity function has effect row `! {}`, so the (DECLASS) rule refuses to authorise its label change.

### Example 2: SQL effect alignment (foreshadowing v0.17)

**With taint types + future SQL effect:**
```ailang
export func userInput() -> string<user> ! {IO} { ... }

export func runQuery(template: string{not user}, params: [string]) -> [Row] ! {Db} { ... }

let q = userInput()
runQuery("SELECT * FROM t WHERE x = ${q}", [])  -- TYPE ERROR: <user> reached template
runQuery("SELECT * FROM t WHERE x = $1", [q])    -- OK: <user> only reaches params
```
Same machine. The user-input label travels with the value; the template parameter rejects it; `params` is an unconstrained list. SymRef (guardians) ≅ parameterised query placeholder ≅ AILANG label-typed value.

## Success Criteria

- [ ] `inbox_injection_v2.ail` (real labels, no string-equality stand-ins, with explicit declassification) verifies with same outcomes as Tier 1: 3 typecheck-clean, 2 type errors at unification time
- [ ] No pure function can lower a label (formally checked: every label-lowering path goes through a function whose effect row contains `Declassify`)
- [ ] `attemptLaunder` (Example 1, attack 2) is a type error — the identity-style bypass is structurally impossible
- [ ] At least one prompt-injection benchmark in `ai-coding-lang-bench` shows AILANG catching the attack at the type level where Python equivalent does not
- [ ] Cross-module labels work (post-M-SMT-CROSS-MODULE-TYPES): label flows from `tools/mail.ail` to `app/inbox.ail` correctly
- [ ] All existing examples (no labels, all `⊥`) continue to compile and verify with no changes
- [ ] All tests passing
- [ ] `ailang prompt` updated with labels, refinements, declassification, and capability gating
- [ ] CHANGELOG entry referencing this design doc and the Tier 1 demo

## Testing Strategy

**Unit tests (lattice):**
- Join: `⊥ ⊔ ℓ = ℓ`, `ℓ ⊔ ℓ = ℓ`, `ℓ₁ ⊔ ℓ₂ kept symbolic`, associativity/commutativity/idempotence
- Refinement evaluation: `not ℓ` against various joins, including symbolic ones; reject malformed refinement bodies (anything beyond `not IDENT`)
- Ordering: `⊑` is reflexive and transitive

**Unit tests (inference):**
- Pure-op label propagation through `let`, function application, match arms, record construction, list cons
- No-laundering: `pure func id(x) { x }` is `∀L. T<L> -> T<L>`, not `T -> T`
- Sink violation produces structured error with source-sink trace

**Integration tests:**
- `inbox_injection_v2.ail` end-to-end through `ailang verify`
- `attemptLaunder` and other negative cases each produce the expected type error
- Cross-module label flow through a two-module test fixture
- No-regression suite: every existing example in `examples/runnable/contracts/` produces the same `verify` outcome

**Benchmark / eval:**
- New `prompt_injection` task in `ai-coding-lang-bench`: prompt the model with a guardians-style scenario; score whether the generated AILANG type-checks under the sink policy
- Compare against Python with `# type: ignore`, Python + guardians DSL, AILANG with labels

## Deferred Decisions

- **Anonymous label syntax** (`T<_>` vs `T<⊥>` vs missing) — agent may choose; document the chosen form
- **Error-message phrasing for sink violations** — agent may choose; structured payload (source, sink, binding chain) is what matters
- **Refinement grammar beyond `not LABEL`** — locked at `not IDENT` only for MVP (see High-Impact Decisions). Richer predicates (disjunctions, depth bounds, label equality) are a future-work item.

## Non-Goals

**Not attempted in this feature:**
- **Security automata** (state-machine on tool-call sequences). Guardians' second pillar. Worth a separate milestone after labels land; needs labels as a foundation.
- **Frame conditions over mutable state** (the McCarthy/Hayes "all unchanged files" form). Out of scope until AILANG has mutable-state semantics worth verifying.
- **A built-in tool registry / executor.** AILANG remains a language; the runtime side stays user-built.
- **Generic dependent types or full refinement types.** The MVP refinement is `not IDENT` only; this is *not* a foothold for general refinement-type machinery.
- **Merging labels with M-EFFECT-REFINEMENT modes.** They share row-parameter machinery but are separate semantic axes (see §"Effects vs Labels").

## Timeline

**Sprint 1** (~28 hours):
- Phase 1 — lattice, AST, parser, propagation, sink check, declassify primitive, single-module demo

**Sprint 2** (~16 hours):
- Phase 2 — iface, cross-module, multi-module test (gated on M-SMT-CROSS-MODULE-TYPES)

**Sprint 3** (~16 hours):
- Phase 3 — benchmark, prompt updates, blog post, docs

**Total: ~60–70 hours across ~3 sprints**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Lattice algebra is over-designed and the symbolic-join machinery hurts inference perf | Med | Keep MVP labels finite + flat; symbolic joins kept unsimplified except for trivial cases (`⊥ ⊔ L = L`). Benchmark inference on a labelled-heavy example before Phase 2. |
| Implicit declassification re-emerges via plumbing helpers (e.g. `unsafeRaw`) | High | The `Declassify` effect is the only mechanism, and capability discipline gates it. Add a lint-level check that flags any module declaring `! {Declassify}` for human review. |
| `T<label>` syntax conflicts with existing generics `T[a]` or future syntax | Med | `<...>` chosen because angle brackets are unused at the type level today. Reserve in design freeze before parser work begins. |
| M-SMT-CROSS-MODULE-TYPES slips, blocking Phase 2 | Med | Ship Phase 1 (single-module) standalone; deliver value even without cross-module. |
| Adding labels breaks existing user code with non-trivial effects | High | Default unlabelled = `⊥`; existing code typechecks unchanged. Regression suite covers every example before Phase 1 starts. |
| Prompt-injection benchmark is gameable (model just refuses to write any networking code) | Med | Two scenarios: a "safe" workflow that *should* type-check (with declassification), and an "injected" workflow that *should not* — both must produce the right verdict. |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-bug-effect-checker-conflation.md](../../implemented/v0_6_2/m-bug-effect-checker-conflation.md) — prior bug in effect-row unification
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) — capability layer that `Declassify` reuses

**Planned (check for overlap):**
- [design_docs/planned/v1_0_0/m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) — parameterised effects `!{E[mode=...]}`; orthogonal axis (see §"Effects vs Labels")
- [design_docs/planned/v0_13_0/m-smt-cross-module-types.md](../v0_13_0/m-smt-cross-module-types.md) — required dependency for Phase 2
- [design_docs/planned/v0_13_0/20251013_auto_caps_capability_inference.md](../v0_13_0/20251013_auto_caps_capability_inference.md) — capability inference; relevant for `Declassify` UX

**External:**
- Erik Meijer, *Guardians of the Agents: Formal Verification of AI Workflows*, [CACM January 2026](https://cacm.acm.org/practice/guardians-of-the-agents/)
- [metareflection/guardians](https://github.com/metareflection/guardians) — Python reference implementation
- Volpano & Smith, *A Sound Type System for Secure Flow Analysis* (1996)
- Pottier & Simonet, *Information Flow Inference for ML* (2003) — FlowCaml
- Myers et al., *Jif: Java + Information Flow*

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Tier 1 demo](../../../examples/runnable/contracts/inbox_injection.ail) — Inbox-injection caught with current contracts
- [SMT codegen fix](../../../internal/smt/codegen.go) — Pre-existing `ensures { a, b }` bug found and fixed during demo authoring (April 2026)

## Future Work

- **Security automata** (post-labels, separate milestone) — state-machine policies over tool-call sequences. Needs labels as a foundation.
- **Lattice-aware Z3 predicates** — for properties like "no label deeper than depth N", round-trip labels into SMT.
- **SQL effect with parameterisation** — natural extension; same mechanism. Likely v0.17 or v0.18.
- **Stdlib of canonical labels** — `<user>`, `<external>`, `<email>`, `<sql-text>`, etc.
- **Compatibility shim with metareflection/guardians** — emit guardians-compatible policy JSON from AILANG type-checked workflows.
- **Richer refinement predicates** — disjunctions, depth bounds, parametric flow constraints, label equality. MVP supports only `not IDENT`.

---

**Document created**: 2026-04-30
**Last updated**: 2026-04-30 (v3 — design freeze closed: `T<label>` + `T{not LABEL}` syntax, free-form constants, capability-based declassification, sibling `Declassify` effect, MVP refinement grammar restricted to `not IDENT`. AI-friendliness pass replaced `¬` with the `not` keyword.)
