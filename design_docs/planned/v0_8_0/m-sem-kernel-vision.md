# M-SEM-KERNEL: AILANG as a Symbolic Reasoning Kernel

**Status**: Planned
**Target**: v0.8.0
**Priority**: P1 (Foundational)
**Estimated**: Vision document (no direct implementation)
**Dependencies**: None (conceptual grounding for D4, SMT verification, budget system)
**Author**: AILANG Core Team
**Inspired by**: Contemporary work on neural computers, symbolic interfaces, and effectful reasoning systems

---

## Abstract

AILANG is not intended to be "another functional language," nor a thin syntax layer over LLM tool calls.
It is designed as a **symbolic reasoning kernel**: a deterministic, verifiable substrate that mediates between neural program synthesis and concrete execution environments.

This document formalizes that role and makes explicit several semantic boundaries that are already present implicitly in AILANG's design.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a foundational vision document that defines AILANG's semantic positioning. It strengthens 9 of 12 axioms.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Core thesis: "deterministic, verifiable substrate." Symbolic programs represent intent, concrete values arise at execution in constrained environment. |
| A2: Replayability | +2 | Pillar 4: "Traces are replayable, auditable, training data." Traces capture symbolic plan + effect intents + contract checks + budget deltas. |
| A3: Effect Legibility | +2 | Pillar 2: Effects as relations `(Intent × EffContext) ⇒ { Outcome | Constraints }` - more rigorous than type annotations. |
| A4: Explicit Authority | +2 | Pillar 5: "All authority must be declared, bounded, and enforced." Eliminates ambient authority by construction. |
| A5: Bounded Verification | +1 | Contracts as symbolic specs enable local verification. Verification reasons about intent satisfaction. |
| A6: Safe Concurrency | 0 | Not directly addressed (conceptual doc). No concurrency implications. |
| A7: Machines First | +2 | **Core differentiator**: Primary author is AI. Purpose is trustworthy AI execution. This IS the axiom manifested. |
| A8: Minimal Syntax | 0 | No syntax changes proposed (conceptual document). |
| A9: Cost Visibility | +1 | Budget deltas in traces, budget-limited effects. "Cost is physical reality." |
| A10: Composability | +1 | Five pillars form coherent system: symbolic intent → effects → contracts → traces → authority. |
| A11: Structured Failure | +1 | "Violations are structured failures." Contract checks in traces. |
| A12: System Boundary | +2 | Core thesis IS the boundary: "sits between probabilistic synthesis (LLMs) and deterministic execution." |

**Net Score: +15** → **Decision: Strong Accept (Foundational Document)**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): +1 - Strengthens determinism through symbolic separation
- [x] A3 (Effects): +2 - Makes effects explicit relations, not opaque functions
- [x] A4 (Authority): +2 - Eliminates ambient authority by design
- [x] A7 (Machines First): +2 - Core thesis: AI-first design

**No hard violations. This document defines what the axioms mean in practice.**

---

## Problem Statement

Large language models can synthesize programs, but they lack:

1. **Stable symbolic memory** - No persistent reasoning state
2. **Enforceable authority boundaries** - Can do anything the API allows
3. **Verifiable intent** - Output doesn't prove it matches specification
4. **Replayable reasoning artifacts** - No trace of why decisions were made

Traditional programming languages are ill-suited to fill this gap because they assume:

- **Human authorship** - Syntax optimized for reading/writing by humans
- **Direct execution semantics** - Programs run immediately on hardware
- **Implicit authority** - File access, network calls happen without declaration

**Impact:**

AILANG addresses this gap by separating symbolic intent from concrete execution, while preserving determinism, auditability, and bounded authority.

---

## Core Thesis

> **AILANG is a symbolic reasoning kernel that constrains, verifies, and executes programs synthesized by neural systems.**

It sits between:
- **Probabilistic program synthesis** (LLMs)
- **Deterministic execution environments** (runtime, OS, network, cloud)

```
┌─────────────────────────────────────────────────────────────┐
│                    AILANG POSITIONING                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Probabilistic World          Deterministic World           │
│   ───────────────────          ──────────────────            │
│                                                              │
│   ┌─────────────┐              ┌─────────────────┐          │
│   │    LLM      │              │   OS / Runtime   │          │
│   │  (Neural    │              │   (Concrete      │          │
│   │   Synthesis)│              │    Execution)    │          │
│   └──────┬──────┘              └────────▲────────┘          │
│          │                              │                    │
│          │  Symbolic                    │  Constrained       │
│          │  Program                     │  Execution         │
│          ▼                              │                    │
│   ┌──────────────────────────────────────────────┐          │
│   │              AILANG KERNEL                    │          │
│   │                                               │          │
│   │  • Symbolic intent representation             │          │
│   │  • Effect relations (not functions)           │          │
│   │  • Contract verification                      │          │
│   │  • Budget enforcement                         │          │
│   │  • Trace generation                           │          │
│   │  • Authority bounds                           │          │
│   └──────────────────────────────────────────────┘          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Design Pillars

### Pillar 1: Symbolic vs Concrete Separation

**Principle:**
> AILANG programs represent **symbolic intent**, not concrete computation.

Concrete values arise only at execution time, within a constrained environment.

**Consequences:**
- The AST, contracts, and specs operate over **symbols and relations**, not concrete state
- Execution produces **traces**, not just results
- Verification reasons about **intent satisfaction**, not operational behavior alone

**Conceptual Flow:**

```
Symbolic Program (AILANG)
        ↓
Constrained Execution
        ↓
Concrete Trace + Result
```

This separation enables:
- SMT verification (reasoning over symbols)
- Replayability (traces are deterministic)
- AI reasoning over plans rather than values

---

### Pillar 2: Effects as Relations, Not Functions

**Principle:**
> Effects are **relations** over intent × environment × constraints, not pure functions.

They are:
- Environment-dependent
- Policy-constrained
- Budget-limited
- Potentially nondeterministic (but explicitly so)

**Formal View:**

Instead of:
```
effect : Input → Output
```

AILANG models effects as:
```
effect : (Intent × EffContext) ⇒ { Outcome | Constraints }
```

**Implications:**
- Effects cannot be reasoned about via referential transparency
- Verification treats effects as **assumptions with bounds**
- Budgets and policies are first-class semantic components

This aligns effect semantics with real-world systems while remaining machine-verifiable.

---

### Pillar 3: Contracts as Symbolic Specifications

**Principle:**
> Contracts are **symbolic predicates** expressing intentional invariants, not runtime assertions.

They serve as:
- An **AI-facing specification language**
- A **verification boundary**
- A **trace annotation mechanism**

Contracts may be:
- **Statically verified** (SMT)
- **Dynamically enforced** (runtime)
- Or both

**Crucially, contracts live above execution, not inside it.**

```ailang
-- Contract as symbolic specification
export func fetchUser(userId: int) -> UserResult ! {Net}
requires { userId > 0 }                        -- Symbolic precondition
ensures  { result.status in [OK, NOT_FOUND] }  -- Symbolic postcondition
```

---

### Pillar 4: Traces as Semantic Artifacts

**Principle:**
> Execution traces are **semantic artifacts**, not logs.

A trace captures:
- Symbolic plan (what was intended)
- Effect intents (what capabilities were requested)
- Contract checks (what was verified)
- Budget deltas (what resources were consumed)
- Verification provenance (what was proved vs runtime-checked)

**Unified Trace Model (Conceptual):**

```
Trace =
  SymbolicBindings
  + EffectInvocations
  + ContractResults
  + BudgetUsage
  + VerificationMetadata
```

**Why This Matters:**
- Traces are **replayable** (deterministic reconstruction)
- Traces are **auditable** (third-party verification)
- Traces are **training data** (AI learns from structured execution)
- AI agents reason over traces, not raw execution

This elevates observability into a semantic layer.

---

### Pillar 5: Explicit Authority & Bounded Power

**Principle:**
> All authority must be **declared**, **bounded**, and **enforced**.

AILANG enforces this via:
- **Explicit effect capabilities** - `! {Net, FS}` declares what's allowed
- **Budget contexts** - Quantitative limits on resource consumption
- **Spec-driven verification** - Design docs become enforcement boundaries
- **Runtime guards** - Violations are structured failures, not silent misbehavior

This eliminates **ambient authority** and makes AI behavior auditable by construction.

```ailang
-- Authority is explicit in the signature
func processData(input: bytes) -> Result ! {FS}  -- Can only use FS
func analyzeText(text: string) -> Analysis       -- Pure: no effects allowed
```

---

## Relationship to Functional Languages

AILANG deliberately diverges from traditional functional languages:

| Aspect | Haskell | AILANG |
|--------|---------|--------|
| Primary author | Human | AI |
| Execution model | Direct | Mediated |
| Effects | Encoded purity (IO monad) | Explicit relations |
| Equality | Referential | Contextual |
| Verification | Type-centric | Contract + trace-centric |
| Purpose | Program correctness | Trustworthy AI execution |

**AILANG borrows discipline, not assumptions.**

We take from Haskell:
- Strong static typing
- Effect tracking
- Algebraic data types

We reject:
- Human-oriented syntax
- Referential transparency as primary reasoning tool
- Direct execution semantics

---

## Implications for AI Code Generation

AILANG enables a **closed trust loop**:

```
User Intent
     ↓
Symbolic Spec (Design Doc YAML)
     ↓
AILANG Program (with contracts)
     ↓
Verified Execution (budget/effect bounded)
     ↓
Auditable Trace (semantic artifact)
```

An AI cannot exceed its authority because:
1. **Authority is symbolic** - declared in spec, not assumed
2. **Enforcement is automatic** - runtime/compiler checks
3. **Violations are structured failures** - typed, inspectable, recoverable

---

## Non-Goals

This document does not propose:

- **Dependent type theory** - AILANG uses contracts, not proof terms
- **Proof assistants** - Verification is bounded, not whole-program
- **Full denotational semantics** - Pragmatic, not academic purity
- **General-purpose theorem proving** - SMT for contracts, not arbitrary proofs

AILANG remains intentionally pragmatic.

---

## Success Criteria

- [ ] This framing incorporated into website Vision section
- [ ] Referenced in M-D4 (Design-Doc-Driven Development)
- [ ] Used as conceptual grounding for:
  - SMT verification design
  - Budget system design
  - Semantic cache evolution
- [ ] Clear differentiation from "just another functional language"

---

## Related Documents

**Foundational (this document informs):**
- [m-d4-design-doc-driven-development.md](m-d4-design-doc-driven-development.md) - D4 workflow
- [m-verify-smt-verification.md](m-verify-smt-verification.md) - SMT verification
- [m-contracts-assert.md](m-contracts-assert.md) - Contract syntax

**References:**
- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why

---

## Summary

AILANG is best understood as:

> **A symbolic reasoning kernel for neural program synthesis.**

Its novelty is not syntax, but **where it draws the boundaries**:
- Between intent and execution
- Between symbols and values
- Between authority and action

This positioning aligns naturally with emerging work on neural computers while remaining grounded in executable systems.

---

## Philosophical Foundation

AILANG treats computation as **navigation through a fixed semantic structure**, not creation of new possibilities.

Key implications:
- **Effects are permissions**, not actions
- **Traces are primary artifacts**, not debugging aids
- **Cost is physical reality**, not incidental overhead

The universe of valid programs exists; AILANG helps AI navigate it safely.

---

**Document created**: 2025-12-23
**Last updated**: 2025-12-23
