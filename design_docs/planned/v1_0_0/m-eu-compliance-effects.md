# M-EU-COMPLIANCE-EFFECTS: Regulatory Trigger Effects for Agentic Systems

**Status**: Planned
**Target**: v1.0.0+ (phased; minimum-viable demo possible by v0.16.x)
**Priority**: P1 — Strategic (compliance + enterprise adoption differentiator; downgraded from P0 per v1-mission iteration 0, Mark 2026-07-10: not v1.0-gating)
**Estimated**: 3–5 weeks (phased across milestones)
**Dependencies**:
  - Effect system (stable) — [m_r2_effect_system.md](../../implemented/v0_2_0/m_r2_effect_system.md) ✅
  - [M-EFFECT-REFINEMENT](m-effect-refinement.md) — parameterised effects unlock `!{Compliance[trigger=...]}`
  - [M-CAPABILITY-BUDGETS](../../implemented/v0_6_2/m-capability-budgets.md) ✅ — capability plumbing for `ComplianceAuthority`
  - M-PKG (package ecosystem) — required to ship `compliance/eu` as a third-party package
  - M-COORDINATOR — for the human-approval pathway (async oversight)
  - Audit/logging infra — for tamper-evident `AuditRecord` chain

**Author**: Mark + AI review
**Created**: 2026-04-28

---

## TL;DR

Encode EU regulatory triggers (AI Act, GDPR, CRA, DSA) as **first-class effects** in AILANG, shipped as an external `compliance/eu` package family rather than stdlib. Programs that touch personal data, make high-risk decisions, or communicate externally carry compliance effects in their type signature; the compiler enforces oversight requirements, capability binding, and audit-log emission at compile time.

> Compliance is not metadata. It is part of the program's effect signature.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Deterministic replay of regulated decisions is a direct primary outcome |
| A2: Replayability | +2 | `AuditRecord` chain provides hash-linked, fully replayable provenance |
| A3: Effect Legibility | +2 | Compliance triggers (`PersonalData`, `HighRiskDecision`, …) live in the effect row, not in policy comments |
| A4: Explicit Authority | +2 | `ComplianceAuthority` capability is required to approve, log, or execute regulated actions |
| A5: Bounded Verification | +1 | Per-function effect rules (e.g. "PersonalData ⇒ AuditLog required") are locally checkable |
| A6: Safe Concurrency | 0 | No direct concurrency change; async approval reuses existing oversight machinery |
| A7: Machines First | +2 | Legal triggers visible to AI agents from the type signature alone — no NLP over policy docs |
| A8: Minimal Syntax | 0 | New effect tokens, but no new syntax forms; rides existing effect-row algebra |
| A9: Cost Visibility | 0 | No direct resource cost tracking (deferred — see Open Questions) |
| A10: Composability | +1 | Compliance effects compose across modules and with M-EFFECT-REFINEMENT modes |
| A11: Structured Failure | +1 | "Missing oversight" / "Missing audit" surface as typed compile errors with fix hints |
| A12: System Boundary | +1 | External-communication and third-party-data triggers make boundary crossings legally explicit |

**Net Score: +14** → **Decision: Proceed to design freeze, then phased implementation**

### Hard Violation Check

- [x] A1: Strengthens determinism (deterministic legal-decision replay); does not introduce hidden non-determinism
- [x] A3: All compliance triggers are visible in the effect row; nothing is ambient
- [x] A4: `ComplianceAuthority` capability gates every regulated action; no ambient authority
- [x] A7: Triggers are machine-parseable enums, not free-form policy strings

---

## Problem Statement

EU regulation (AI Act, GDPR, CRA, DSA) applies to **what systems do**, not **how they're implemented**. Agentic AI systems are particularly hard to regulate because:

- Decision paths are opaque
- Behavioural drift across runs is the norm
- Authority boundaries are unclear
- Auditability is bolted on after the fact

**Current State (industry-wide, including AILANG):**
- Compliance is enforced at the **policy/runtime** layer (logging, monitoring, after-the-fact audits)
- Personal-data flow is documented in PIAs and DPIAs, not derived from code
- "High-risk classification" (AI Act Annex III) is asserted by humans, not provable from the program
- Replay of a regulated decision requires snapshot-restore plumbing the program does not own

**Impact:**
- Enterprise buyers cannot get **machine-checkable compliance artifacts** from any current language
- AI-assisted code synthesis cannot reason about whether generated code triggers regulation — the signal isn't in the type
- AILANG already has the structural ingredients (effects = external actions, capabilities = authority, determinism = reproducibility) but does not expose them as legal primitives

**Strategic gap:** No language today encodes regulation in the type system. AILANG can be the first.

---

## External Reference

**Paper**: *Agentic AI Systems and EU Law — Compliance Through Action Triggers* (arXiv: 2604.04604)

**Key paper observations** mapped to AILANG:

| Paper finding | AILANG primitive |
|---|---|
| Compliance is triggered by external actions | Effects (`!{Net}`, `!{IO}`, …) |
| Triggered by personal-data processing | New effect: `!{Compliance[PersonalData]}` |
| Triggered by affected-individual class | Compliance trigger taxonomy (this doc) |
| Authority boundaries must be explicit | Capabilities (`ComplianceAuthority`) |
| High-risk classification depends on runtime behaviour | Effect-row analysis at type-check |
| Required: traceability, oversight, reproducibility, risk class | `AuditRecord`, approval gates, deterministic replay, trigger taxonomy |

---

## Goals

**Primary Goal:** Make compliance properties of AILANG programs **statically enforceable from the type signature alone**, not operationally inferred from logs.

**Success Metrics:**
- ≥ 90% of regulated actions in the demo programs are statically detectable from effect rows
- 100% audit-trace completeness for regulated functions (no silent paths)
- ≥ 95% of compliance-violation cases (missing approval, missing audit) caught at compile time
- ≥ 99% replay determinism for regulated decisions
- A **legal advisor** can read an AILANG signature and identify the regulatory triggers without reading the body

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Ship as third-party package family, not stdlib** | Determines governance, release cadence, breakage policy, and liability surface | human | design | high |
| Compliance effects are **distinct effect tokens** (not refined `!{IO}` modes) | Affects effect-row algebra; if collapsed later, every importer breaks | human | design | high |
| `ComplianceAuthority` is a **single composite capability** vs. one capability per trigger | Affects ergonomics and security granularity | human | design | high |
| Enforcement default: **error** (not warn) when triggers present without oversight | Affects every user's first experience and back-compat for existing programs | human | design | high |
| `AuditRecord` is **hash-chained Merkle** vs. flat append-only | Affects auditor tooling and storage cost | agent | implementation | med |
| Trigger taxonomy is **closed enum** (Phase 1) vs. open extension | Affects whether legal updates need a compiler release | human | design | med |
| Jurisdiction modelled as **first-class type parameter** vs. runtime config | Affects whether multi-jurisdiction programs are typeable | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Package-vs-stdlib delivery model (see "Delivery Model" section below)
- [ ] Effect-token collapse policy (will compliance effects ride M-EFFECT-REFINEMENT modes, or stand alone?)
- [ ] Capability granularity — composite `ComplianceAuthority` vs. per-trigger split
- [ ] Default enforcement strictness (error vs. warn vs. configurable)
- [ ] Closed vs. open trigger taxonomy for v1
- [ ] Jurisdiction representation (type param vs. runtime)

---

## Delivery Model: Package vs. Stdlib

**Recommendation: Ship as third-party package family `sunholo/compliance-eu` (and siblings).**

The compliance effect *tokens* themselves (`PersonalData`, `HighRiskDecision`, `RequiresHumanApproval`, `AuditLog`, `ExternalCommunication`) are a small set of effect-row labels that the compiler must understand to enforce cross-effect rules. Two viable splits:

**Option A — Effects in stdlib, mappings in packages (recommended):**
- Effect tokens + the cross-effect rules (e.g. "PersonalData ⇒ AuditLog must be present") live in `std/compliance` (≤ 200 LOC, frozen)
- Trigger taxonomies (AI Act Annex III, GDPR Art. 6, CRA, DSA), capability shapes, audit-record schemas, and jurisdiction logic live in **separate packages**: `sunholo/compliance-eu-ai-act`, `sunholo/compliance-eu-gdpr`, `sunholo/compliance-eu-cra`, `sunholo/compliance-eu-dsa`
- Demo / reference programs live in `sunholo/compliance-eu-examples`

**Option B — Everything in packages:**
- Even the core effect tokens are package-defined; compiler treats them as opaque labels with no built-in cross-effect rules
- Cross-effect enforcement happens via package-provided lints/passes

### Pros and Cons

| Aspect | Stdlib (only) | Package family (Option A — recommended) | Package only (Option B) |
|---|---|---|---|
| **Release cadence** | Tied to compiler releases (slow) | Compiler ships frozen tokens; legal mappings update independently | Independent of compiler |
| **Liability surface** | AILANG project carries legal interpretation risk | Sunholo (or chosen vendor) carries legal mapping; AILANG ships only the mechanism | Mapping vendor carries everything |
| **Compile-time enforcement** | Strong — compiler knows cross-effect rules | Strong — same | Weak — compiler can't enforce "PersonalData ⇒ AuditLog" without knowing the tokens |
| **Multi-jurisdiction** | One blessed taxonomy → blocks non-EU users | Per-jurisdiction packages compose; EU/US/UK side-by-side | Same as A |
| **Versioning regulation** | Compiler bump for every legal change | `compliance-eu-ai-act@2026.04` style — semantic, dated | Same as A |
| **Discoverability** | High (in the box) | Medium (must `ailang pkg add`) — mitigate with starter template | Low |
| **Trust** | High (audited as part of language) | Medium-High (signed package, versioned) | Low (any package can claim compliance) |
| **Aligns with M-DX-PACKAGE-DOGFOODING** | No | Yes — exercises the package ecosystem | Yes |
| **Forward-compat with non-EU regulation** | Forces stdlib to grow indefinitely | Clean — `compliance-us`, `compliance-uk`, `compliance-jp` | Clean |
| **Risk if mapping vendor disappears** | N/A | Effect mechanism still works; users swap mapping package | Compiler enforcement disappears |

**Why Option A wins:**

1. **Effect mechanism vs. legal interpretation are separable concerns.** The compiler should know that `PersonalData` requires `AuditLog`. It should not know what counts as "personal data" under GDPR Art. 4(1) — that's a legal interpretation that changes by case law.
2. **Liability is parked in the right place.** A regulator question "why did your AILANG program emit this audit record?" gets answered by the *mapping package* the user chose, not by the language project.
3. **Multi-jurisdiction is native.** A program may need to comply with EU AI Act + US state-level + UK ICO simultaneously. Packages compose; one stdlib doesn't.
4. **Regulation updates faster than languages.** AI Act delegated acts will keep landing through 2027+. Package versioning (`@2026.04.15`) lets users pin a legal interpretation date — invaluable for audit defence.
5. **Dogfoods M-PKG.** This is a real, high-stakes use case for the package ecosystem (see [M-DX-PACKAGE-DOGFOODING](m-dx-package-dogfooding.md)).

**Risks of the package model:**

- **Discoverability**: users may not know they need it. Mitigation: ship a `ailang init --compliance=eu` template; surface in `ailang doctor` when regulated effects are detected without a mapping package present.
- **Token collisions**: two competing mapping packages could each define `HighRiskDecision`. Mitigation: core token namespace is reserved in `std/compliance`; mapping packages classify *into* core tokens, they don't define new ones.
- **Trust**: anyone can publish a package called `compliance-eu`. Mitigation: a verified-publisher mark for compliance packages; legal review attestation in the package manifest.

---

## Solution Design

### Overview

Three layers:

1. **Core effect tokens** (`std/compliance`, frozen, ≤ 200 LOC) — define the closed token set and the cross-effect rules the compiler enforces.
2. **Capability shape** (`std/compliance` interface, mapping packages provide concrete implementations) — `ComplianceAuthority { approve, log }`.
3. **Mapping packages** (`sunholo/compliance-eu-*`, versioned by date) — turn legal categories into core tokens, provide approval/audit primitives, ship example programs.

### Architecture

**Components:**

1. **Core effect tokens** (`std/compliance`):
   - `!{Compliance[PersonalData]}` — runtime processes personal data
   - `!{Compliance[HighRiskDecision]}` — output drives a regulated decision
   - `!{Compliance[ExternalCommunication]}` — system communicates outside the trust boundary
   - `!{Compliance[RequiresHumanApproval]}` — function output must pass an approval gate
   - `!{Compliance[AuditLog]}` — function emits an audit record
   - Sub-categories for Phase 2: `EmploymentDecision`, `FinancialDecision`, `MedicalDecision`, `CriticalInfrastructure`, `ProductWithDigitalElements`

2. **Cross-effect rules** (compiler pass):
   - `PersonalData` ∈ effects ⇒ `AuditLog` ∈ effects
   - `HighRiskDecision` ∈ effects ⇒ `RequiresHumanApproval` ∈ effects ∧ `AuditLog` ∈ effects
   - `ExternalCommunication` ∈ effects ⇒ `AuditLog` ∈ effects
   - `RequiresHumanApproval` ∈ effects ⇒ function calls `requireApproval` before any return path
   - All four ⇒ caller must possess `ComplianceAuthority` capability

3. **Capability** (`std/compliance`):
   ```
   capability ComplianceAuthority {
     approve : ApprovalRequest -> ! {Compliance[RequiresHumanApproval]} ApprovalResult
     log     : AuditEvent       -> ! {Compliance[AuditLog]} Unit
   }
   ```

4. **Audit record** (mapping package responsibility, `std/compliance` provides shape):
   ```
   type AuditRecord = {
     inputHash:  Hash,
     outputHash: Hash,
     effects:    List[Effect],
     triggers:   List[RegulatoryTrigger],
     timestamp:  Time,
     version:    ProgramHash,
     prevHash:   Hash         -- hash chain
   }
   ```

5. **Mapping packages**:
   - `sunholo/compliance-eu-ai-act@2026.04` — Annex III classification helpers, conformity assessment hooks
   - `sunholo/compliance-eu-gdpr@2026.04` — lawful-basis tagging, DPIA helpers, data-subject-rights primitives
   - `sunholo/compliance-eu-cra@2026.04` — vulnerability disclosure shape, SBOM hooks
   - `sunholo/compliance-eu-dsa@2026.04` — moderation-decision logging, statement-of-reasons emission

### Implementation Plan

**Phase 1 — Core types and effect tokens (Week 1, ~25h)**
- [ ] Define closed `RegulatoryTrigger` enum in `std/compliance`
- [ ] Define `OversightLevel` enum
- [ ] Wire core effect tokens into the effect-row algebra (likely as `!{Compliance[trigger=...]}` riding M-EFFECT-REFINEMENT, OR as standalone tokens — see Design Freeze)
- [ ] Unit tests for parsing/elaborating compliance-effect signatures

**Phase 2 — Compiler rules and diagnostics (Week 2–3, ~40h)**
- [ ] Implement cross-effect rule checker (PersonalData ⇒ AuditLog, etc.)
- [ ] Implement "approval-before-return" path analysis for `RequiresHumanApproval`
- [ ] Implement capability-gate check (regulated effects require `ComplianceAuthority`)
- [ ] Diagnostics: structured errors with fix hints ("add `RequiresHumanApproval` and call `requireApproval(...)` before return")
- [ ] Configurable strictness (error / warn) — feature-flagged for gradual adoption

**Phase 3 — Audit + replay integration (Week 3–4, ~30h)**
- [ ] `AuditRecord` shape in `std/compliance`
- [ ] Hash-chained emission helper in capability impl
- [ ] Deterministic replay hook (ties to M-EFFECT-REFINEMENT replay contracts)
- [ ] Integration test: replay an audit chain and verify hashes

**Phase 4 — Mapping packages + demo (Week 4–5, ~30h)**
- [ ] `sunholo/compliance-eu-ai-act@2026.04` skeleton + Annex III helpers
- [ ] `sunholo/compliance-eu-gdpr@2026.04` skeleton + lawful-basis types
- [ ] End-to-end demo: loan-application evaluator (PersonalData + FinancialDecision + HighRiskDecision + RequiresHumanApproval + AuditLog) — refuses to compile if any effect is dropped
- [ ] Tutorial doc + example AILANG file

### Files to Create

**New stdlib files** (`std/compliance/`):
- `triggers.ail` (~80 LOC) — `RegulatoryTrigger`, `OversightLevel` types
- `effects.ail` (~60 LOC) — effect-token declarations
- `audit.ail` (~50 LOC) — `AuditRecord` shape
- `authority.ail` (~40 LOC) — `ComplianceAuthority` capability
- `requireApproval.ail` (~30 LOC) — approval-gate helper

**New compiler files** (`internal/effects/compliance/`):
- `rules.go` (~250 LOC) — cross-effect rule checker
- `approval_path.go` (~200 LOC) — "approval-before-return" path analysis
- `diagnostics.go` (~150 LOC) — structured error messages with fixits
- `*_test.go` — full coverage

**Modified files:**
- `internal/types/effect_row.go` (+~80 LOC) — recognise compliance tokens
- `internal/elaborate/effects.go` (+~60 LOC) — surface compliance effects in elaboration
- `internal/iface/digest.go` (+~20 LOC) — include compliance effects in interface digest

**New packages** (`ailang-packages/packages/`):
- `compliance-eu-ai-act/` — Annex III mappings (~400 LOC AILANG)
- `compliance-eu-gdpr/` — GDPR helpers (~400 LOC AILANG)
- `compliance-eu-examples/` — loan demo, content-moderation demo (~600 LOC AILANG)

---

## Examples

### Example 1: Loan-application evaluator (motivating example)

```ailang
import std/compliance (PersonalData, FinancialDecision, HighRiskDecision,
                       RequiresHumanApproval, AuditLog, requireApproval)
import sunholo/compliance-eu-gdpr (lawfulBasis)
import sunholo/compliance-eu-ai-act (annexIII)

export func evaluateLoanApplication(app: LoanApplication, auth: ComplianceAuthority)
  ! {Compliance[PersonalData], Compliance[FinancialDecision],
     Compliance[HighRiskDecision], Compliance[RequiresHumanApproval],
     Compliance[AuditLog]}
  -> Result[ApprovedDecision, Error]
= {
    let decision = scoreApplication(app);
    let approved = requireApproval(decision, auth);   -- mandatory before return
    Ok(approved)
}
```

**Compiler guarantees:**
- ❌ Removing `RequiresHumanApproval` from the effect row → compile error: *"HighRiskDecision present; oversight required"*
- ❌ Removing the `requireApproval` call → compile error: *"function declares RequiresHumanApproval but no approval call dominates all return paths"*
- ❌ Calling without an `auth: ComplianceAuthority` argument → compile error: *"regulated effects require ComplianceAuthority capability"*
- ✅ Audit record automatically emitted via the capability; signature-visible to AI agents and reviewers.

### Example 2: Drift detection (regression scenario)

A maintenance commit removes the explicit `HighRiskDecision` annotation:

```diff
- ! {Compliance[PersonalData], Compliance[FinancialDecision],
-    Compliance[HighRiskDecision], Compliance[RequiresHumanApproval],
-    Compliance[AuditLog]}
+ ! {Compliance[PersonalData], Compliance[AuditLog]}
```

Compiler:
```
error[E-CMPL-002]: function calls `scoreApplication` which has effect
  Compliance[HighRiskDecision], but caller does not propagate it.
  --> evaluateLoanApplication
  hint: re-add HighRiskDecision and RequiresHumanApproval, or factor out
        the high-risk computation into a separately reviewed function.
```

### Example 3: Multi-jurisdiction composition

```ailang
import sunholo/compliance-eu-gdpr@2026.04
import sunholo/compliance-us-ccpa@2026.04

export func processCustomerData(c: Customer, auth: ComplianceAuthority)
  ! {Compliance[PersonalData], Compliance[AuditLog]}
  -> Profile = ...
```

Both jurisdictions' mapping packages classify the call site into the core `PersonalData` token — the program is provably compliant under both regimes simultaneously.

---

## Success Criteria

- [ ] Closed core trigger taxonomy lands in `std/compliance` (frozen for v1)
- [ ] Cross-effect rule checker rejects all 4 documented violation classes with structured errors
- [ ] `requireApproval` path analysis catches missing-gate cases (≥ 95% on test corpus)
- [ ] `ComplianceAuthority` capability gates execution of regulated effects
- [ ] Hash-chained `AuditRecord` deterministically replays (≥ 99% on demo programs)
- [ ] `sunholo/compliance-eu-ai-act@2026.04` ships with Annex III helpers
- [ ] `sunholo/compliance-eu-gdpr@2026.04` ships with lawful-basis primitives
- [ ] Loan-application demo compiles, runs, and replays from audit log
- [ ] Tutorial doc walks through "regulating an existing AILANG program"
- [ ] Legal-advisor read-through validates a sample signature against AI Act Annex III
- [ ] All tests passing; coverage ≥ 90% on new compiler code
- [ ] CHANGELOG, examples, and `docs/docs/guides/compliance.md` updated

---

## Testing Strategy

**Unit tests:**
- Effect-row parser accepts/rejects compliance-effect syntax
- Cross-effect rule checker: 4 violation classes × positive/negative cases
- Approval-path analysis: dominator-set correctness on branchy CFGs
- Capability check: ambient vs. explicit, polymorphic propagation
- `AuditRecord` hash-chain construction and verification

**Integration tests:**
- End-to-end loan demo compiles, runs, replays
- Multi-jurisdiction demo (GDPR + CCPA mapping packages co-resident)
- Drift regression: mutation-test the demo by removing each effect; assert compile error

**Manual / acceptance tests:**
- Legal-advisor read-through (find a friendly EU lawyer)
- Enterprise prospect demo (loan or content-moderation use case)
- AI-agent benchmark: can a code-synthesis agent add a regulated function and get the effects right on first compile?

---

## Deferred Decisions

The following are intentionally left open:

- Concrete diagnostic-message wording — agent may choose, subject to fix-hint coverage
- Internal layout of `internal/effects/compliance/` (one file vs. several) — agent may choose
- Audit-log on-disk format (JSONL vs. CBOR vs. proto) — agent may choose; mapping packages can override
- `sunholo/compliance-eu-cra` and `sunholo/compliance-eu-dsa` content depth — agent may choose minimum-viable in Phase 4; full coverage in a follow-up
- Whether `requireApproval` is a builtin or a stdlib function — agent may choose during Phase 1
- Mapping-package distribution channel (registry index entry, tarball, git pin) — human at review

## Non-Goals

- **SMT-based formal verification of compliance properties** — out of scope; deferred to follow-up doc
- **Cost-tracking for regulated actions** (A9) — deferred (would need integration with M-CAPABILITY-BUDGETS pricing)
- **Dynamic jurisdiction switching at runtime** — out of scope; jurisdictions are package-import-time
- **GDPR data-subject-rights workflow runtime** (DSR portal, etc.) — out of scope; the package provides primitives, not infrastructure
- **Non-EU jurisdictions** (US, UK, Japan, Brazil) in v1 — the design supports them; mappings ship later
- **Auto-classification of existing code** — Phase 5+; first version requires explicit annotation

---

## Timeline

**Week 1** (~25h): Phase 1 — core types + effect tokens, parser/elaborator wiring, basic tests.

**Week 2** (~25h): Phase 2a — cross-effect rule checker + diagnostics.

**Week 3** (~20h): Phase 2b — approval-path analysis + capability-gate check; Phase 3a — `AuditRecord` shape + hash chain.

**Week 4** (~20h): Phase 3b — replay integration; Phase 4a — `compliance-eu-ai-act` + `compliance-eu-gdpr` package skeletons.

**Week 5** (~20h): Phase 4b — loan-application demo, multi-jurisdiction demo, tutorial doc, legal-advisor read-through.

**Total: ~110 hours across 5 weeks** (≈3 sprints).

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Legal-mapping ambiguity (what counts as "high risk"?) | High | Mapping lives in versioned packages with legal-advisor sign-off in manifest; AILANG project does not own the interpretation |
| Over-constraining real programs (false positives at compile) | High | Configurable strictness (error / warn); per-effect opt-in during migration; "compliance noir mode" off by default until v1 |
| Jurisdiction complexity (28 EU member states + UK + EEA) | Medium | Default `compliance-eu-*` packages target the harmonised baseline; member-state deltas as separate packages |
| Token-namespace collisions across mapping packages | Medium | Core token set is `std/compliance` and frozen; mapping packages classify into it, do not extend it |
| Trust / "compliance washing" by unverified packages | Medium | Verified-publisher attestation in package manifest; `ailang doctor compliance` flags unsigned mapping packages |
| Effect-row machinery not stable enough to absorb compliance tokens | Medium | Hard-block on M-EFFECT-REFINEMENT; do not start Phase 1 until effect-refinement is in beta |
| AILANG project taking on regulatory liability | High | Stdlib stays mechanism-only (no legal text); all legal interpretation lives in third-party packages with explicit terms |
| Performance regression from per-call audit emission | Low | Audit emission is on-demand via capability; benchmark in Phase 3 |

---

## Strategic Impact

This positions AILANG as **the first language where compliance is statically enforceable, not operationally inferred.**

Expected effects:
- Enterprise adoption — buyers can ask for "machine-checkable compliance artifacts" and get them
- Regulatory differentiation — alignment with EU AI Act Articles 9, 12, 14 (risk management, record-keeping, human oversight) at the language layer
- AI-agent compliance — code-synthesis agents reason about regulation from types, not policy docs
- Mapping-package marketplace — opens an ecosystem for legal tech vendors

---

## Open Questions

1. **Should compliance effects be core language features or library extensions?**
   *Current answer: hybrid — frozen tokens in stdlib, taxonomies + mappings in packages. Locked at Design Freeze.*
2. **How strict should enforcement be by default?**
   *Current proposal: warn during Phase 1 migration; error from v1.0 onward.*
3. **Multi-jurisdiction simultaneously?** Yes (compositional via packages).
4. **Dynamic jurisdiction switching?** No — jurisdictions are import-time.
5. **SMT-based compliance verification?** Deferred to a follow-up design doc; would integrate with [M-EFFECT-REFINEMENT](m-effect-refinement.md) replay contracts.
6. **Cost / budget integration?** Deferred — could attach to [M-CAPABILITY-BUDGETS](../../implemented/v0_6_2/m-capability-budgets.md) for "audit-record budget" or "approval-call budget".

---

## Suggested Next Steps

- Draft full effect taxonomy for AI Act Annex III (separate doc)
- Build minimum-viable end-to-end demo (loan-decision) on a feature branch
- Validate with: legal advisor (EU); enterprise prospect (financial-services or content-moderation buyer)
- Coordinate with M-EFFECT-REFINEMENT — confirm whether compliance effects ride `mode=` or stand alone

---

## Related Documents

**Implemented (informs design):**
- [m_r2_effect_system.md](../../implemented/v0_2_0/m_r2_effect_system.md) — base effect system
- [m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) — capability plumbing pattern
- [m-type-effect-row-regression.md](../../implemented/v0_9_4/m-type-effect-row-regression.md) — effect-row regression lessons

**Planned (check for overlap and ordering):**
- [m-effect-refinement.md](m-effect-refinement.md) — parameterised effects; this doc rides on it
- [m-dx-package-dogfooding.md](m-dx-package-dogfooding.md) — package ecosystem friction; compliance is a high-stakes dogfooding case
- [m-entropy-budgets.md](m-entropy-budgets.md) — envelope-level policy; potential integration point for compliance budgets

## References

- *Agentic AI Systems and EU Law — Compliance Through Action Triggers* (arXiv: 2604.04604)
- EU AI Act, Regulation (EU) 2024/1689 — Articles 9, 12, 14; Annex III
- GDPR, Regulation (EU) 2016/679 — Articles 5, 6, 22, 35
- Cyber Resilience Act, Regulation (EU) 2024/2847
- Digital Services Act, Regulation (EU) 2022/2065
- [Design Axioms](/docs/references/axioms)
- [Philosophical Foundations](/docs/references/philosophical-foundations)

## Future Work

- `compliance-us-ccpa`, `compliance-uk-ico`, `compliance-jp-appi` mapping packages
- SMT-backed verification of compliance invariants (separate design doc)
- Automatic classification: infer compliance triggers from code patterns
- Integration with Conformity Assessment Body workflows (signed compliance artifacts)
- Compliance-aware code synthesis benchmark (does an AI agent generate regulated code correctly?)

---

**Document created**: 2026-04-28
**Last updated**: 2026-04-28
