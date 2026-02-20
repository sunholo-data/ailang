# M-ENTROPY: Semantic Entropy Budgets

**Status**: Planned
**Target**: v0.7.0
**Priority**: P1 - Medium (AI-First Innovation)
**Estimated**: 42 hours (~1.5 sprints)
**Dependencies**: M-CAPABILITY-BUDGETS (v0.6.2 ✅)

## Executive Summary

This document proposes **entropy budgets** as a first-class concept in AILANG, providing a formal framework for managing the semantic phase transition from natural language intent to executable code.

**Core Insight**: Natural language is "cheap" because it borrows entropy from shared priors. Code is "expensive" because it must pay that entropy back. AILANG's design documents are where this debt becomes visible and negotiable.

**The Entropy Budget Equation:**
```
Entropy Budget = Permitted Ambiguity × Designated Resolver × Collapse Deadline
```

**Anchor Principle:**
> Entropy budgets do not measure uncertainty; they assign responsibility for its elimination.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Entropy budgets make nondeterminism explicit and bounded |
| A2: Replayability | +1 | Turn-count tracked in traces; convergence auditable |
| A3: Effect Legibility | +1 | Entropy visible in design doc envelopes and source annotations |
| A4: Explicit Authority | +1 | Resolvers explicitly assigned (compiler/human/llm/none) |
| A5: Bounded Verification | +1 | Budgets are locally checkable; layered validation |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Budgets are machine-parseable YAML; enables AI reasoning |
| A8: Minimal Syntax | 0 | `@entropy` annotation is minimal; metadata in YAML |
| A9: Cost Visibility | +1 | **Primary goal** - entropy cost explicit in design docs |
| A10: Composability | +1 | Layered validation composes (envelope → source → compiler) |
| A11: Structured Failure | +1 | EntropyViolation is typed error with fix hints |
| A12: System Boundary | +1 | Explicit boundary between intent and execution |

**Net Score: +10** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Entropy budgets ADD determinism by bounding uncertainty
- [x] A3 (Effects): Entropy is visible, not hidden
- [x] A4 (Authority): Resolvers explicitly assigned, no ambient delegation
- [x] A7 (Machines First): YAML schema optimized for machine parsing

## Problem Statement

**The Entropy Debt Problem:**

When AI agents generate code from natural language, they navigate a semantic phase transition:

```
Intent space (many degrees of freedom)
        ↓
Constraint saturation (design docs)
        ↓
Executable specification (few degrees of freedom)
```

**Current State:**
- Design docs consume more tokens than resulting code (expected!)
- No explicit tracking of where ambiguity is resolved
- No mechanism to declare "who decides" for uncertain choices
- Turn-count (clarification turns) not tracked as entropy metric
- AILANG has effect budgets (`@limit=N`) but no semantic entropy budgets

**Impact:**
- AI agents cannot reason about entropy boundaries
- Humans cannot audit where uncertainty was resolved
- No quality signal for design doc completeness
- "Build me a dashboard" works via hidden priors; "deterministic replay" explodes into clarifications

**Observation:**
> Turn count ≈ ∫ (unresolved entropy) dt

High turn counts indicate unresolved degrees of freedom leaking into conversation.

## Goals

**Primary Goal:** Make semantic entropy explicit, bounded, and verifiable across the AILANG development lifecycle.

**Success Metrics:**
1. Entropy YAML schema defined with JSON Schema validation
2. `ailang check --entropy` validates source against design doc envelope
3. Turn-count visible in message threads as quality signal
4. ≥3 design docs use entropy budgets as examples
5. Zero impact on existing code (additive only)

## The Five Axes of Entropy

Entropy is **vector-valued**, not scalar. Each axis has independent failure modes.

| Axis | Definition | Metric Proxy |
|------|------------|--------------|
| **Semantic** | Meanings left implicit | Undefined nouns/verbs; LLM clarification turns |
| **Behavioral** | Execution paths unconstrained | Effect cardinality; trace divergence |
| **Authority** | Permissions unspecified | Authority surface area per module |
| **Temporal** | Timing undefined | Implicit time references per function |
| **Interpretive** | Resolver unassigned | Unbounded choice points |

### v0.7.0 Scope: Three Normative Axes

**Design Decision:** For v0.7.0, only three axes are **normative** (enforced by compiler):

| Axis | v0.7 Status | Rationale |
|------|-------------|-----------|
| **Semantic** | ✅ Normative | Core to entropy budgets; new capability |
| **Behavioral** | ✅ Normative | Already partially exists via `@limit=N` |
| **Interpretive** | ✅ Normative | Novel; who decides is the key question |
| Authority | ℹ️ Derived | Checked via existing capability system |
| Temporal | ⚠️ Warn-only | Conceptually right but subtle; informational for now |

**Why this scoping:**
- Avoids "the place where everything goes" syndrome
- Authority is already solved by capabilities (would be redundant)
- Temporal entropy needs more design work (v0.8.0+)

**Current AILANG Coverage:**

| Axis | Mechanism | Gap |
|------|-----------|-----|
| Semantic | Type inference | No budget annotation |
| Behavioral | Effect types + `@limit=N` | ✅ Covered |
| Authority | Capability system | ✅ Covered (derived) |
| Temporal | Clock effect | ⚠️ Warn-only |
| Interpretive | AI effect | No "who decides" annotation |

## Solution Design

### Overview

Entropy budgets exist at **three layers** with compiler-enforced consistency:

```
┌─────────────────────────────────────────────────────────────┐
│                 LAYERED ENTROPY VALIDATION                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LAYER 1: DESIGN DOC ENVELOPE                               │
│  ─────────────────────────────────────────                  │
│  Location: YAML frontmatter in design docs                  │
│  Purpose:  Declare module-level entropy policy              │
│  Verified: `ailang check --design-doc`                      │
│                                                              │
│                        ↓ constrains ↓                        │
│                                                              │
│  LAYER 2: SOURCE ANNOTATIONS                                │
│  ─────────────────────────────────────────                  │
│  Location: @entropy annotations in AILANG source            │
│  Purpose:  Override/refine at function level                │
│  Rule:     Source can TIGHTEN but not LOOSEN envelope       │
│                                                              │
│                        ↓ verified by ↓                       │
│                                                              │
│  LAYER 3: COMPILER ENFORCEMENT                              │
│  ─────────────────────────────────────────                  │
│  Location: Type checking phase                              │
│  Purpose:  Validate source matches declared entropy         │
│  Error:    EntropyViolation with source location            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Architecture

**Components:**

1. **Entropy Schema** (`internal/entropy/schema.go`)
   - YAML schema for entropy envelopes
   - JSON Schema validation
   - Go types for entropy budget representation

2. **Envelope Validator** (`internal/entropy/validate.go`)
   - Parse design doc frontmatter
   - Validate envelope against schema
   - Report missing/invalid fields

3. **Layered Validator** (`internal/entropy/layered.go`)
   - Compare source annotations to envelope
   - Enforce tighten-only rule
   - Generate EntropyViolation errors

4. **Source Annotations** (parser/ast)
   - `@entropy` token and AST node
   - Module-level and function-level support
   - Elaboration to typed AST

5. **Turn Tracker** (`internal/messages/turns.go`)
   - Track conversation turns per thread
   - Classify clarification vs progress turns
   - Surface in `ailang messages list --stats`
   - **IMPORTANT: Observational only** - never a hard failure condition

**Turn Count Design Constraint:**

Turn count is:
- ✅ A quality signal
- ✅ A heuristic for design doc completeness
- ✅ A trend metric for entropy collapse speed
- ❌ **Never** a budget violation or hard failure

**Why:** If turn count becomes normative, agents will optimise for fewer turns, not better entropy collapse. Keep it observational.

### Entropy Envelope Schema

**v0.7.0 Schema (3 normative axes, 3 canonical deadlines):**

```yaml
# design_docs/planned/v0_7_0/my-feature.md
---
milestone: M-FEATURE
entropy:
  # Three normative axes (enforced)
  semantic:
    permitted: bounded     # none | bounded | open
    resolver: human        # compiler | runtime | human | llm | none
    deadline: design       # design | compile | runtime
  behavioral:
    permitted: none        # Strict: no unconstrained execution paths
    resolver: compiler
    deadline: compile
  interpretive:
    permitted: bounded
    resolver: llm
    deadline: runtime
    scope: ["layout", "wording"]  # Allowed choices
    forbidden: ["business-logic", "security"]  # Never delegated

  # Informational axes (warn-only in v0.7.0)
  # authority: derived from capability system
  # temporal: warn-only, deadline tracking
---
```

**Minimal envelope (quick start):**

```yaml
---
milestone: M-FEATURE
entropy:
  semantic: { permitted: bounded, resolver: human, deadline: design }
  behavioral: { permitted: none, resolver: compiler, deadline: compile }
  interpretive: { permitted: none }  # No LLM delegation
---
```

**Permitted Values:**
- `none` - No uncertainty allowed
- `bounded` - Uncertainty allowed within declared scope
- `open` - No constraint (not recommended for production)

**Resolver Values:**
- `compiler` - Type inference, exhaustive pattern matching
- `runtime` - Effect handlers, budget enforcement
- `human` - Design doc clarification, code review
- `llm` - AI-assisted resolution (bounded by scope)
- `none` - Uncertainty forbidden (must be resolved before this phase)

**Deadline Values (Canonical Phases):**

For v0.7.0, deadlines collapse into **3 canonical phases** (internal refinement possible later):

| Phase | Meaning | Includes |
|-------|---------|----------|
| `design` | Must resolve before implementation | parse, type-check conceptually |
| `compile` | Must resolve before codegen | link, type-check, codegen |
| `runtime` | May remain until execution | effect handlers, budget enforcement |

**Full deadline vocabulary** (internal use, maps to canonical phases):
- `parse` → `compile`
- `type-check` → `compile`
- `link` → `compile`
- `design-freeze` → `design`
- `runtime` → `runtime`

**Why this simplification:** Avoids the compiler becoming a project manager. The language surface sees 3 phases; internal tooling can use finer distinctions.

### Source Annotations

```ailang
-- Module-level (inherits from design doc, can tighten)
@entropy { interpretive: none }  -- Override: no LLM delegation allowed
module myModule

-- Function-level (scoped refinement)
let fetch: String -> Result ! {Net @limit=5}
  @entropy(resolver=llm, scope=[retry-policy])  -- LLM can choose retry
  = ...
```

### Consistency Rules

| Design Doc | Source | Result |
|------------|--------|--------|
| `bounded` | `none` | ✅ Allowed (tighter) |
| `bounded` | `bounded` | ✅ Allowed (same) |
| `bounded` | `open` | ❌ Error (looser) |
| `zero` | anything | ❌ Error unless `zero` |
| `open` | any | ✅ Allowed (no constraint) |

### Entropy Locality (Critical Invariant)

**Named concept:** Entropy must be **locally exhaustible**.

A bounded entropy declaration must identify where uncertainty disappears. If it doesn't collapse by the declared deadline, that's an error.

**Implications:**
- `bounded` + `deadline: compile` → compiler must see the collapse point
- `bounded` + `resolver: human` → design doc must contain the resolution
- `bounded` + `resolver: llm` → must specify `scope` of allowed choices

**Error example:**
```
EntropyLocalityError: Bounded semantic entropy not exhausted
  Axis: semantic
  Permitted: bounded
  Resolver: human
  Deadline: design
  Problem: No resolution found in design doc

  Hint: Add explicit decision to design doc, or change resolver.
```

This becomes critical for **entropy composition** in v0.8.0+.

### Implementation Plan

**Phase 1: Schema & Infrastructure** (~8h)
- [ ] Define entropy YAML schema with JSON Schema for validation
- [ ] Create `internal/entropy/` package
- [ ] Implement design doc frontmatter parser
- [ ] Unit tests for schema parsing

**Phase 2: Design Doc Validation** (~6h)
- [ ] `ailang check --design-doc` command
- [ ] Validate entropy envelopes
- [ ] Report entropy status in `ailang docs search`
- [ ] Integration with design doc viewer

**Phase 3: Source Annotations** (~10h)
- [ ] Parser support for `@entropy` annotations
- [ ] AST nodes for entropy metadata
- [ ] Store in typed AST during elaboration
- [ ] Lexer tokens: `AT_ENTROPY`, `RESOLVER`, `SCOPE`

**Phase 4: Compiler Integration** (~8h)
- [ ] Layered validation in type checker
- [ ] Load envelope from linked design doc
- [ ] Compare source annotations against envelope
- [ ] `EntropyViolation` error type with fix hints

**Phase 5: Turn Tracking** (~6h)
- [ ] Extend message store with turn metrics
- [ ] Track `TurnCount`, `ClarifyTurns`, `ConvergedAt`
- [ ] Surface in `ailang messages list --stats`
- [ ] Report in sprint completion summaries

**Phase 6: Documentation & Examples** (~4h)
- [ ] User guide: `docs/docs/guides/entropy-budgets.md`
- [ ] Update 3+ existing design docs with entropy envelopes
- [ ] CLI help text

**Total: ~42 hours (1.5 sprints)**

### Files to Modify/Create

**New files:**

| File | Purpose | LOC |
|------|---------|-----|
| `internal/entropy/schema.go` | YAML schema, parsing, types | ~200 |
| `internal/entropy/validate.go` | Envelope validation logic | ~250 |
| `internal/entropy/layered.go` | Source vs envelope comparison | ~150 |
| `internal/entropy/schema_test.go` | Schema parsing tests | ~200 |
| `internal/entropy/validate_test.go` | Validation tests | ~250 |
| `internal/messages/turns.go` | Turn-count metrics | ~100 |
| `docs/docs/guides/entropy-budgets.md` | User documentation | ~400 |

**Modified files:**

| File | Changes | LOC |
|------|---------|-----|
| `internal/lexer/token.go` | Add `AT_ENTROPY` token | ~5 |
| `internal/lexer/lexer.go` | Tokenize `@entropy` | ~20 |
| `internal/parser/parser.go` | Parse `@entropy` annotations | ~80 |
| `internal/ast/ast.go` | Add `EntropyAnnotation` node | ~30 |
| `internal/types/typecheck.go` | Entropy validation phase | ~60 |
| `internal/errors/errors.go` | Add `EntropyViolation` error | ~20 |
| `cmd/ailang/check.go` | `--entropy` and `--design-doc` flags | ~40 |
| `internal/messages/store.go` | Turn tracking fields | ~30 |
| `internal/messages/list.go` | `--stats` output | ~20 |

**Grand Total: ~1,855 LOC (new: 1,550, modified: 305)**

## Examples

### Example 1: Design Doc with Entropy Envelope

**Design doc frontmatter:**
```yaml
---
milestone: M-SEMANTIC-CACHE
entropy:
  semantic:
    permitted: bounded
    resolver: human
    deadline: design-freeze
  behavioral:
    permitted: zero
    resolver: compiler
    deadline: compile
  interpretive:
    permitted: bounded
    resolver: llm
    deadline: runtime
    scope: ["cache-key-format"]
    forbidden: ["eviction-policy", "security"]
---

# M-SEMANTIC-CACHE: Semantic Caching

...
```

### Example 2: Compiler Validation

```
$ ailang check --entropy module.ail

Entropy validation:
  ✓ semantic: bounded (human) - design-freeze
  ✓ behavioral: zero (compiler) - compile
  ✗ interpretive: source declares 'open' but envelope requires 'bounded'
    → line 42: let processInput = @entropy(interpretive: open) ...
    → fix: tighten to 'bounded' or update design doc

1 entropy violation found
```

### Example 3: Turn-Count Metrics

```
$ ailang messages list --stats

Thread Statistics:
  Thread ID  | Turns | Clarify | Converged | Status
  -----------|-------|---------|-----------|--------
  thread-123 |    12 |       8 | 2h ago    | ✅ resolved
  thread-456 |    47 |      31 | -         | ⚠️ high entropy
  thread-789 |     3 |       0 | 10m ago   | ✅ resolved

Average turns to convergence: 20.6
High-entropy threads (>30 turns): 1
```

### Example 4: Source Annotation Tightening

**Design doc envelope:**
```yaml
entropy:
  interpretive:
    permitted: bounded
    resolver: llm
    scope: ["layout", "wording", "retry-policy"]
```

**Source (valid - tightens envelope):**
```ailang
@entropy { interpretive: none }  -- No LLM delegation in this module
module securityModule

let authenticate: Credentials -> Result ! {Net @limit=3}
  -- This function has zero interpretive entropy
  = ...
```

## Success Criteria

- [ ] Entropy YAML schema defined with JSON Schema validation
- [ ] `ailang check --design-doc` validates entropy envelopes
- [ ] `ailang check --entropy` validates source against envelope
- [ ] `@entropy` annotation parses in lexer/parser
- [ ] `EntropyViolation` error type with fix hints
- [ ] Turn-count fields in message store (`TurnCount`, `ClarifyTurns`)
- [ ] `ailang messages list --stats` shows turn metrics
- [ ] At least 3 design docs use entropy budgets as examples
- [ ] User guide published at docs site
- [ ] All tests passing (schema, validation, parser, compiler)

## Testing Strategy

**Unit tests:**
- Schema parsing (valid/invalid YAML)
- Envelope validation (all 5 axes)
- Layered consistency (tighten ok, loosen fail)
- Parser: `@entropy` annotations
- Turn tracking metrics

**Integration tests:**
- End-to-end: design doc → source → compiler validation
- CLI: `ailang check --entropy` workflow
- Message stats with turn counts

**Golden tests:**
- Sample design docs with expected validation output
- Error message formatting

### Error Message Style Guide

**EntropyViolation errors must be boring.**

Errors should:
- ✅ Look like type errors
- ✅ Have predictable, mechanical wording
- ✅ Include concrete fix suggestions
- ❌ Never be philosophical
- ❌ Never explain "entropy theory"
- ❌ Never use abstract language or metaphors

**Good error:**
```
EntropyViolation: source exceeds envelope constraint
  Axis: interpretive
  Envelope: bounded (scope: [layout, wording])
  Source: open (line 42)
  Fix: tighten to 'bounded' or update design doc envelope
```

**Bad error:**
```
EntropyViolation: The entropy budget for interpretive uncertainty
has been exceeded. Entropy represents the degree of freedom in
decision-making that remains unresolved...
```

Reserve theory for documentation; keep errors actionable.

## Non-Goals

**Not in this feature (deferred to v1.0+):**
- **Automatic entropy inference** - Would require AI reasoning about code semantics
- **Entropy transfer/negotiation** - Complex composition rules between modules
- **Runtime entropy tracking** - Design-time and compile-time focus first
- **Memory/time entropy axes** - Use existing budget system (`@limit=N`)

## Timeline

**Sprint 1** (20 hours):
- Phase 1: Schema & Infrastructure
- Phase 2: Design Doc Validation
- Phase 3 (partial): Parser groundwork

**Sprint 2** (22 hours):
- Phase 3 (complete): Source Annotations
- Phase 4: Compiler Integration
- Phase 5: Turn Tracking
- Phase 6: Documentation & Examples

**Total: ~42 hours across 2 sprints**

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Schema too complex | Medium | Low | Start with 3 axes (semantic, behavioral, interpretive), add others incrementally |
| Parser conflicts | Low | Medium | `@entropy` is distinct token, no ambiguity with effects |
| Adoption friction | Medium | Medium | Make entropy optional, default to "open" for legacy code |
| Design doc overhead | Medium | Low | Tooling generates skeleton; `ailang init` adds envelope |

## Related Documents

**Canonical references:**
- [Design Axioms](/docs/references/axioms) - Core principles (A1, A4, A9)
- [M-CAPABILITY-BUDGETS](../../implemented/v0_6_2/m-capability-budgets.md) - Effect budgets (`@limit=N`)
- [M-D4 Design-Doc-Driven Development](m-d4-design-doc-driven-development.md) - **KEY INTEGRATION** (see below)

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) - Budget system architecture
- [design_docs/implemented/v0_6_0/semantic-caching-complete.md](../../implemented/v0_6_0/semantic-caching-complete.md) - Deterministic caching patterns

---

## Integration with M-D4 (Design-Doc-Driven Development)

**M-D4 and M-ENTROPY are complementary:**

| Concern | M-D4 (D4) | M-ENTROPY |
|---------|-----------|-----------|
| **Focus** | What is *allowed/required* | What *uncertainty* exists |
| **Grants** | Effect permissions | Authority entropy axis |
| **Envelope** | Resource budgets | Behavioral entropy axis |
| **Obligations** | Contract predicates | Semantic entropy axis |
| **Resolver** | Not addressed | **NEW** - Who collapses uncertainty |
| **Deadline** | Not addressed | **NEW** - When must it resolve |

**Unified Schema Vision:**

M-ENTROPY extends D4's YAML schema with entropy metadata:

```yaml
---
# D4 Spec (existing)
spec:
  schema_version: "1.0"

  grants:
    effects:
      permitted: [Net]
      forbidden: [FS]

  envelope:
    api_calls: 5
    execution_ms: 5000

  obligations:
    requires:
      - name: "valid_user_id"
        predicate: "userId > 0"

# M-ENTROPY Extension (NEW)
entropy:
  semantic:
    permitted: bounded
    resolver: human
    deadline: design-freeze
  behavioral:
    permitted: zero     # Enforced by D4 envelope
    resolver: compiler
    deadline: compile
  authority:
    permitted: zero     # Enforced by D4 grants
    resolver: capability
    deadline: runtime
  interpretive:
    permitted: bounded
    resolver: llm
    deadline: runtime
    scope: ["retry-policy", "error-message-wording"]
    forbidden: ["business-logic", "security"]
---
```

**Validation Coordination:**

| Check | Tool | Phase |
|-------|------|-------|
| D4 grants (effects) | `ailang verify --spec` | Compile |
| D4 envelope (budgets) | Runtime budget enforcement | Runtime |
| D4 obligations (contracts) | Contract checker | Compile + Runtime |
| **M-ENTROPY envelopes** | `ailang check --entropy` | Design + Compile |
| **M-ENTROPY annotations** | Layered validator | Compile |

**Sequencing:**

1. M-ENTROPY validation happens FIRST (design-time)
2. D4 verification happens SECOND (compile-time)
3. Runtime enforcement uses both

This ordering ensures entropy is collapsed before specifications are verified.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- Discussion: "Semantic entropy in autonomous systems" - Original insight
- Shannon, C.E. (1948) - "A Mathematical Theory of Communication"

## Future Work

**v0.8.0+:**
- Automatic entropy inference from code analysis
- Entropy visualization in traces (like effect visualization)
- Budget negotiation between caller/callee
- Cross-module entropy composition rules

**v1.0.0+:**
- Entropy-aware code generation
- AI training on entropy-annotated programs
- Entropy budgets as optimization hints

---

**Document created**: 2026-01-16
**Last updated**: 2026-01-16

---

## Appendix: Theoretical Foundation

### The Conservation Law (Informal)

> You cannot reduce total system entropy; you can only relocate it.

In software systems, entropy lives in exactly one of three places:

| Location | Who pays the cost |
|----------|-------------------|
| Design docs | Human / LLM reasoning time |
| Code | Runtime + maintenance complexity |
| Operations | Incidents, outages, undefined behavior |

AILANG's entropy budgets front-load entropy into design-time semantics, where it is:
- Inspectable
- Auditable
- Machine-checkable

### Why "Build a Dashboard" is Short

The phrase "build a dashboard" works because:
- It indexes a massive pretrained prior
- Defaults are socially standardized (CRUD, charts, pagination, auth)

This is **semantic inheritance**, not entropy elimination.

Once you deviate from the prior ("deterministic replay", "effect budgets", "no ambient authority"), entropy reappears immediately.

**AILANG removes implicit inheritance — so the cost becomes visible.**

### Reframing the Problem

> Natural language is cheap because it borrows entropy from the world.
> Code is expensive because it must pay it back.
> AILANG makes the debt visible early, when it is still negotiable.
