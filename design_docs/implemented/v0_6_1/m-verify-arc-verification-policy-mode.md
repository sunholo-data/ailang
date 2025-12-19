# M-VERIFY: ARC-Style Verification & Policy Mode

**Status**: IMPLEMENTED
**Target**: v0.6.2+
**Priority**: P1 (Medium-High)
**Estimated**: 6-10 weeks (phased implementation, high uncertainty for SMT phase)
**Dependencies**:
- Core type system (complete)
- Effects system (complete)
- Go codegen backend (complete)
- SharedMem effect (planned, Phase 3 only)

---

## Implementation Status Update (December 2025)

**Last reviewed**: 2025-12-18

### Verdict: Still a Strong Fit for AILANG

The design remains architecturally sound. AILANG has evolved favorably since the original design, with several developments that **reduce implementation effort**:

| Factor | Status | Notes |
|--------|--------|-------|
| AST Foundation | ✅ Ready | `FuncDecl.Properties`, `Property` struct with `Binders` already exist |
| Parser Infrastructure | ✅ Ready | `property` syntax with `forall` binders can be extended |
| Effect System | ✅ Mature | `EffContext` in `internal/effects/` provides runtime context |
| Codegen | ⚠️ Larger | Now 12,500 LOC (well-organized, typed wrappers help) |
| SMT Backend | ❌ Not started | Still needs `internal/smt/` package |
| Trace Runtime | ❌ Not started | Still needs `internal/trace/` package |

### Key Finding: Existing Property Infrastructure

AILANG already has substantial property-testing infrastructure that can be **directly reused** for contracts:

**Existing AST** (`internal/ast/ast_decl.go`):
```go
type Property struct {
    Name    string
    Binders []*Binder  // forall bindings - maps to contract variables
    Expr    Expr       // boolean expression - maps to contract predicate
    Pos     Pos
}

type Binder struct {
    Name string
    Type Type
    Pos  Pos
}

type FuncDecl struct {
    // ...
    Properties []*Property  // Already wired to function declarations!
    // ...
}
```

**Existing Parser** (`internal/parser/parser_testing.go`):
- `parseProperty()` - parses `forall(x: Type, y: Type) => expr`
- `parsePropertiesBlock()` - parses `[ property1, property2, ... ]`
- `parseBinder()` - parses `name: Type`

**Existing Syntax** (works today):
```ailang
property "addition is commutative" (x: int, y: int) =
  x + y == y + x
```

**Existing Tokens** (`internal/lexer/token.go`):
- ✅ `FORALL`, `PROPERTY`, `TEST`, `ASSERT` - all exist
- ❌ `REQUIRES`, `ENSURES`, `INVARIANT` - need to add (trivial)

### Mapping: Properties → Contracts

| Existing Property | Contract Equivalent | Adaptation Needed |
|------------------|--------------------|--------------------|
| `Property.Binders` | Not used for contracts | Skip for `requires`/`ensures` (no universal quantification) |
| `Property.Expr` | Contract predicate | Direct reuse |
| `FuncDecl.Properties` | Contract storage | Add `ContractKind` field |

**Proposed Extension**:
```go
type ContractKind int
const (
    PropertyKind ContractKind = iota  // Existing property-based tests
    RequiresKind                       // Precondition
    EnsuresKind                        // Postcondition
    InvariantKind                      // Module/type invariant
)

type Property struct {
    Name    string
    Kind    ContractKind  // NEW: Distinguish contract types
    Binders []*Binder     // For forall properties, empty for contracts
    Expr    Expr
    Pos     Pos
}
```

### Revised Estimates

| Phase | Original | Revised | Savings | Reason |
|-------|----------|---------|---------|--------|
| Phase 0 (Plumbing) | 15-20h | 8-12h | ~35% | Property infrastructure exists |
| Phase 1 (SMT) | 25-35h | 25-35h | 0% | Still high uncertainty |
| Phase 2 (Redundant Gen) | 10-15h | 10-15h | 0% | Unchanged |
| **Total** | **50-70h** | **43-62h** | **~12%** | |

### Recent Codegen Changes (Favorable)

Since December 2025, codegen has evolved with:
- **M-DX26**: Typed wrappers - functions generate both `_impl` and typed API
- **M-DX27**: Better type tracking in match expressions
- **M-CODEGEN-UNIFIED-SLICE**: Robust slice type conversion

These changes **help** M-VERIFY because:
1. Runtime contract checks can be injected at typed wrapper boundary
2. Type information is preserved for SMT encoding
3. Well-organized codegen structure (12,500 LOC across 20+ focused files)

### Existing Telemetry Infrastructure

AILANG has two telemetry systems that can be leveraged for M-VERIFY:

**1. Pipeline Metrics** (`internal/pipeline/metrics.go`):
- `PipelineMetrics` struct - Structured timing for compilation phases
- `MetricsCollector` - Opt-in via `AILANG_METRICS=1`
- Hub integration via `AILANG_HUB_URL` for centralized collection

**2. Debug Effect** (`internal/effects/debug.go`) - **Directly Applicable to Contract Traces**:
```go
type DebugContext struct {
    logs       []LogEntry
    assertions []AssertionResult  // ← Maps directly to contract checks!
    timestamp  int64
}

type LogEntry struct {
    Message   string
    Location  string  // "file.ail:42" (auto-injected by compiler)
    Timestamp int64   // Logical time (host-defined)
}

type AssertionResult struct {
    Passed   bool
    Message  string
    Location string
}
```

**Key insight**: The `DebugContext` pattern is **exactly** what M-VERIFY needs for contract traces:

| DebugContext Feature | M-VERIFY Equivalent |
|---------------------|---------------------|
| `AssertionResult` | `ContractCheck` (Passed/Failed + Message + Location) |
| `Collect()` host-only | Contract traces are host-only |
| `LogEntry` with timestamps | Function call trace with timing |
| Wired to `EffContext` | Contract context integration is trivial |

**Proposed extension** (follows Debug pattern):
```go
// ContractContext - follows DebugContext pattern exactly
type ContractContext struct {
    checks     []ContractCheck
    timestamp  int64
}

type ContractCheck struct {
    Kind      ContractKind    // Requires, Ensures, Invariant
    Passed    bool
    Message   string          // Contract expression as string
    Location  string          // "module.ail:42"
    Function  string          // "module.funcName"
    Args      map[string]any  // Argument values at time of check
    Hash      string          // SHA256 of contract expression
}
```

**Impact on estimates**: Component 4 (Execution Traces) can reuse ~70% of the Debug effect architecture, further reducing Phase 0 effort.

### Design Review Feedback (December 2025)

**Reviewer verdict**: ✅ **Still a go** — more so than the earlier shape because it front-loads real value (Phase 0.5 runtime checks) and leans hard on existing infrastructure.

#### What Got Better (Materially)

| Improvement | Why It Matters |
|-------------|----------------|
| **Phase 0 is genuinely low-risk** | Not inventing a new AST channel; extending `Property` with `ContractKind` and reusing parser/testing plumbing |
| **Phase 0.5 is a killer "ship something useful" slice** | Runtime contract checks + trace artifacts pay off immediately for agents and humans, even if SMT slips |
| **Profiles are the right product surface** | `runtime-only`, `smt-best-effort`, `smt-strict` let you stay sound (no false "verified" claims) while providing guardrails |

#### Design Adjustments Required

**1. Don't panic by default — make runtime contract failures structured**

Panics are fine for early bring-up, but for "policy mode" you need deterministic, auditable failure without taking down the host.

**Updated behavior**:
| Mode | Contract Violation Behavior |
|------|----------------------------|
| `debug` / `--verify-contracts` | Panic (fast feedback during development) |
| `policy` / `--verify-contracts=report` | Return `ContractViolation` error value + structured trace, then stop |

The Go embedding surfaces this as an error return at the typed wrapper boundary, not a Go panic.

**Updated `ContractContext`**:
```go
type ContractContext struct {
    checks     []ContractCheck
    timestamp  int64
    mode       ContractMode  // Panic | Report | Off
}

type ContractMode int
const (
    ContractModePanic  ContractMode = iota  // Development: panic on violation
    ContractModeReport                       // Policy: return error + trace
    ContractModeOff                          // Disabled
)
```

**2. Make verifiable fragment enforcement mechanically obvious**

The fragment definition is great, but UX lives or dies on "why was my contract skipped?".

**Required implementation**:
```go
// IsSMTEncodable checks if a function can be verified with SMT
// Returns (encodable, reasons[]) where reasons explain any rejections
func IsSMTEncodable(funcDecl *ast.FuncDecl) (bool, []SMTRejectionReason)

type SMTRejectionReason struct {
    Code     string  // "RECURSIVE", "HIGHER_ORDER", "DEEP_ADT", "QUANTIFIED", "EFFECTFUL"
    Message  string  // Human-readable explanation
    Location Pos     // Where the issue is
    Hint     string  // Suggested fix
}
```

**This function must be used consistently in**:
- Compiler warnings (`warning: contract ignored for SMT; function not in verifiable fragment`)
- `ailang verify` output (show exact reasons)
- Trace output (`verification.static=false, reasons=[...]`)

#### Medium-Risk Items to Watch

**1. SMT "function body encoding" — scope balloon risk**

Even with the fragment restrictions, encoding the body is the hard part, not the contracts.

**Recommended Phase 1 approach** (incremental, not all-at-once):

| Step | What to Encode | SMT Approach |
|------|----------------|--------------|
| 1 | Contracts only | Uninterpreted function model (sanity harness) |
| 2 | Straight-line arithmetic | `let` chains → SMT `let` |
| 3 | `if/else` | SMT `ite` |
| 4 | `match` over enum | SMT datatype match |
| 5 | `match` over single-level ADT | SMT datatype constructors |

**Ship "verify works for these shapes"** without pretending it's general.

**2. `float` in SMT — semantic mismatch**

SMT-LIB `Real` is **not** IEEE754 float semantics. If AILANG `float` is IEEE-ish, proving things with `Real` can be misleading.

**Recommendation for v0.6.2 SMT MVP**:
- Option A: Restrict SMT to `int`/`bool`/`enums` initially
- Option B: Treat `float` as `Real` but:
  - Label results as "mathematical reals"
  - Only enable under `smt-best-effort` unless user explicitly opts in
  - Add warning: `note: float verified as mathematical real, not IEEE754`

**3. Trace schema — keep it stable and minimal**

**Recommendation**:
- Keep contract identifiers/hashes, pass/fail, location (always)
- Make `Args map[string]any` **optional** behind `--trace-args` flag
  - Can explode trace size
  - May leak sensitive values in policy domains

#### Naming: Property vs Contract

The AST reuses `Property` for contracts — pragmatic, but can confuse tooling.

**Resolution** (no AST churn):
- AST stays as-is (`Property` struct)
- User-facing output uses **"Contract"** terminology for `requires`/`ensures`
- **"Property"** reserved for `forall`-style tests in CLI/docs

Example:
```bash
$ ailang verify module.ail
Verifying contracts for module.funcName...    # Not "properties"
  requires: x >= 0                     [OK]
  ensures:  result > x                 [OK]
```

---

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Contracts are declarative annotations, not procedural boilerplate |
| Preserve Semantic Clarity | ++ | +2 | Contracts make invariants explicit and machine-verifiable |
| Increase Determinism | ++ | +2 | SMT verification provides formal guarantees that deterministically defined behavior satisfies stated invariants |
| Lower Token Cost | ++ | +2 | Contracts + redundant generation severely cut dead-ends and hallucinated logic; compounds over time |
| **Net Score** | | **+7** | **Decision: Move forward** |

**Decision rule:** Net score > +1 -> Move forward | <= 0 -> Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

---

## Problem Statement

AILANG is designed for autonomous AI code synthesis, but currently lacks infrastructure for:

1. **Formal verification** - No way to prove correctness of generated code
2. **Policy enforcement** - No structured way to express and verify business rules
3. **Confidence scoring** - No mechanism to assess reliability of AI-generated implementations
4. **Auditable execution** - Limited trace/proof artifacts for reasoning about program behavior

**Current State:**
- AI generates AILANG code but correctness relies solely on tests
- Business policies are encoded in ad-hoc ways without formal contracts
- No structural comparison between multiple AI generations
- Effect traces exist but lack contract verification metadata

**Impact:**
- **AI systems**: Cannot formally verify generated code correctness
- **Policy authors**: No machine-checkable way to express invariants
- **Auditors**: Limited proof artifacts for compliance verification
- **Development**: More debugging cycles due to late-caught invariant violations

**Research Foundation:**
This design implements, inside AILANG, the core techniques of the **ARC (Automated Reasoning Checks)** system:
- **"A Neurosymbolic Approach to Natural Language Formalization and Verification"** (Bayless et al., AWS, 2025) ([arXiv:2511.09008](https://arxiv.org/abs/2511.09008))
  - Key techniques: Redundant autoformalization, SMT-based verification, soundness-first design
  - AILANG adaptation: Contracts as AI-facing spec surface, structural confluence for confidence scoring

---

## Goals

**Primary Goal:** Make AILANG a first-class substrate for neurosymbolic verification - enabling provably correct AI code generation with auditable policy enforcement.

**Success Metrics:**
- Contracts can express pre/postconditions and invariants for pure functions
- SMT backend proves contract satisfaction or provides counterexamples
- Redundant generation yields confidence scores analogous to ARC's AV metric
- Execution traces include contract check results as proof artifacts
- Runtime contract checks catch violations deterministically in Go backend

**Non-Goals (for v0.6.0):**
- Full dependent types or proof assistant semantics
- Complete autoformalization of natural language policies (future integration)
- User-defined type classes (deferred to structural reflection v0.7.0+)

---

## Solution Design

### Overview

Four new components layered on existing AILANG:

```
AILANG source
   |
Type inference / effects / core AST
   |
[1] Go backend      [2] SMT backend       [3] AI-helper mode
   |                      |                      |
Executable + traces    Proof obligations      Redundant gen / confidence
```

1. **Contracts & Policy Modules** - Syntax for pre/postconditions + invariants
2. **SMT-LIB Backend** - Compiler pass emitting verification conditions
3. **Redundant AILANG Generation** - Multi-sample codegen with structural comparison
4. **Execution Traces & Effect Logging** - Standardized trace format with contract metadata

### Two-Layer Architecture

This feature provides two complementary layers:

**Layer 1: General Verification** (any pure AILANG function)
- Contracts + SMT + traces for game logic, simulation invariants, pricing, physics, etc.
- Examples: "HP never exceeds maxHP", "energy conservation", "no negative mass"

**Layer 2: Policy Mode** (curated stdlib for business/regulatory domains)
- `std/policy` module with Money, Date, AgeClass, Status types
- Patterns specifically for compliance, rules engines, decision logic
- Examples: refund eligibility, admission rules, pricing tiers

The underlying machinery is identical; the distinction is about examples and stdlib support.

### Verifiable Fragment (v0.6.0)

A function is eligible for SMT verification if **all** of the following hold:

1. **Has contracts**: Function is annotated with `requires` and/or `ensures`
2. **Pure effect set**: Effect annotation is `! {}` (no effects)
3. **Non-recursive**: Function body contains no direct or mutual recursion
4. **No higher-order functions**: No lambdas passed as arguments or returned
5. **Shallow ADT patterns**: Pattern matches on ADTs at most 1 level deep
6. **QF logic only**: Contract expressions use quantifier-free arithmetic/boolean logic

**Compiler behavior for non-verifiable functions:**
```
warning: contract ignored for SMT; function not in verifiable fragment
  --> module.ail:15:1
   |
15 | export func recursive(...) -> int ! {}
   |        ^^^^ recursive functions not supported in v0.6.0
   |
   = note: runtime contract checks will still be emitted
   = help: consider extracting a non-recursive helper for the core logic
```

Functions outside the fragment still get **runtime contract checks** (Phase 0), just no SMT proof.

### Verification Profiles

Per-module or global setting to control verification behavior:

```ailang
-- Module-level directive
#verification = smt-strict

module policies/refund
-- ...
```

| Profile | Behavior |
|---------|----------|
| `off` | Parse contracts, no checks (documentation only) |
| `runtime-only` | Go runtime asserts, no SMT |
| `smt-best-effort` | Try SMT; if unsupported, fall back to runtime (default) |
| `smt-strict` | If SMT cannot encode/verify, fail the build |

This aligns with ARC's soundness vs. recall tradeoffs - `smt-strict` maximizes soundness at the cost of coverage.

### Contracts as AI-Facing Spec Surface

**Key design intent:** For LLM coders, contracts are part of the specification surface.

When an AI agent is asked to "implement X", the recommended workflow is:

1. **Write the contract first** (`requires`/`ensures`) - express reasoning in machine-checkable form
2. **Implement the function** in terms of those contracts
3. **Call `ailang verify`** to get formal proof or counterexample
4. **Use redundant generation** to assess structural confidence

Contracts are how an AI coder expresses its internal reasoning in verifiable form. This matches ARC's "autoformalize -> verify" pattern but lifted into the language itself.

### Architecture

#### Component 1: Contracts & Policy Modules

**Syntax extension (minimal, preserves language feel):**

```ailang
module policies/refund

import std/io (println)
import std/policy (Money, Date)

export type FlightStatus = DEPARTED | CANCELLED | DELAYED
export type RefundEligibility = ELIGIBLE | NOT_ELIGIBLE

-- Contract-annotated function
export func refundEligibility(
  status: FlightStatus,
  hoursDelayed: int,
  customerCredits: int
) -> RefundEligibility ! {}
requires { hoursDelayed >= 0, customerCredits >= 0 }
ensures  { result == ELIGIBLE => hoursDelayed >= 5 }
{
  match status {
    CANCELLED => ELIGIBLE,
    DELAYED   => if hoursDelayed >= 5 { ELIGIBLE } else { NOT_ELIGIBLE },
    DEPARTED  => NOT_ELIGIBLE
  }
}
```

**Key design decisions:**
- `requires { ... }` - Boolean conjunction over arguments + globals
- `ensures { ... }` - Can reference `result` (distinguished name) plus arguments
- Contracts are refinement constraints, NOT dependent types
- Type inference unchanged; contracts are a separate logical layer
- Logical language restricted to QF fragment (matching ARC):
  - Arithmetic over `int`, `float`
  - Equality over enums / simple ADTs
  - Boolean connectives
  - No quantifiers (initially)

#### Component 2: SMT-LIB Backend

**Target fragment:** Quantifier-free with integers, reals, bools, enums (QF_LIA/QF_LRA + datatypes)

**Type mapping:**
| AILANG | SMT-LIB | Notes |
|--------|---------|-------|
| `int` | `Int` | Direct mapping |
| `float` | `Real` | ⚠️ **Not IEEE754** — only under `smt-best-effort` with warning |
| `bool` | `Bool` | Direct mapping |
| Enum variants | `declare-datatype` | Direct mapping |
| Simple ADTs | `declare-datatype` (single-level) | Single-level only in v0.6.2 |

> **Note on `float`**: SMT-LIB `Real` is mathematical real arithmetic, not IEEE754 floating-point. Verification results for `float` are sound for real number semantics but may not reflect IEEE754 edge cases (NaN, infinity, rounding). Enable only under `smt-best-effort` profile with explicit warning in output.

**Generated SMT structure:**
```smt2
; Type declarations
(declare-datatype FlightStatus ((DEPARTED) (CANCELLED) (DELAYED)))
(declare-datatype RefundEligibility ((ELIGIBLE) (NOT_ELIGIBLE)))

; Symbolic variables
(declare-const status FlightStatus)
(declare-const hoursDelayed Int)
(declare-const customerCredits Int)
(declare-const result RefundEligibility)

; Requires (preconditions)
(assert (>= hoursDelayed 0))
(assert (>= customerCredits 0))

; Function body encoded as definition of result
; (inlined or via axiom depending on complexity)

; Ensures negated (check for counterexample)
(assert (not (=> (= result ELIGIBLE) (>= hoursDelayed 5))))

(check-sat)
(get-model)
```

**Interpretation:**
- `sat` -> Contract violation found (counterexample in model)
- `unsat` -> Contract holds for all inputs

#### Component 3: Redundant AILANG Generation

**Protocol (mirrors ARC's redundant autoformalization):**

1. Request N candidate implementations from LLM (e.g., N=3)
2. Parse each to Core AST
3. **Hard filter: Contract verification** (key enhancement over basic structural comparison)
   - Run contracts (SMT or runtime) on each candidate
   - Discard candidates that violate contracts or fail SMT
   - Only proceed with candidates that pass
4. Normalize surviving candidates:
   - Alpha-renaming (consistent variable names)
   - Desugar pattern matches
   - Inline trivial lets
5. Compare normalized ASTs for structural equivalence among survivors

**Two-stage filtering:**
```
N candidates
   |
   v
[Contract Filter] -- discard: violates requires/ensures
   |
   v
K survivors (K <= N)
   |
   v
[Structural Comparison] -- cluster by AST hash
   |
   v
Confidence score
```

**Confidence classification (enhanced):**
| Classification | Meaning | Confidence Score |
|----------------|---------|------------------|
| `AllFailed` | No candidate passed contracts | 0.0 |
| `SingleSurvivor` | 1 of N passes, unique | 1/N (e.g., 0.33) |
| `EquivalentMajority` | >= 2 pass and agree structurally | K/N where K agree |
| `ValidEquivalent` | All N pass and agree | 1.0 |
| `DivergentSurvivors` | Multiple pass but differ structurally | flag for review |

**Output format:**
```json
{
  "function": "refundEligibility",
  "samples": 3,
  "survivors": 2,
  "classification": "EquivalentMajority",
  "confidence": 0.67,
  "contract_filter": {
    "passed": 2,
    "failed": 1,
    "failures": [
      { "sample": 3, "violation": "ensures: result == ELIGIBLE => hoursDelayed >= 5" }
    ]
  },
  "clusters": [
    { "count": 2, "hash": "abc123...", "diff": null }
  ]
}
```

This gives a hierarchy: **contracts as hard filter, structural equivalence as soft consensus**.

#### Component 4: Execution Traces & Effect Logging

**Enhanced trace format (JSON lines):**

Traces include contract identifiers, verification status, and stable hashes for:
- **Auditors**: Evidence trail linking execution to verified contracts
- **AI training**: Large-scale datasets of "contracts passed/failed" for learning

```json
{
  "fn": "policies.refund.refundEligibility",
  "args": { "status": "DELAYED", "hoursDelayed": 4, "customerCredits": 0 },
  "effects": [],
  "contracts": {
    "requires": { "status": "ok", "id": "req_a1b2c3", "hash": "sha256:..." },
    "ensures": { "status": "ok", "id": "ens_d4e5f6", "hash": "sha256:..." }
  },
  "verification": {
    "static": true,
    "method": "smt",
    "solver": "z3-4.12.2",
    "proven_at": "2025-12-06T09:00:00Z"
  },
  "result": "NOT_ELIGIBLE",
  "timestamp": "2025-12-06T10:30:00Z",
  "trace_id": "tr-abc123",
  "module_version": "v1.2.0"
}
```

**Key additions:**
- `contracts.*.id` - Stable identifier for specific contract clause
- `contracts.*.hash` - Content hash of contract expression (tracks changes)
- `verification.static` - Whether function was SMT-verified at compile time
- `verification.method` - "smt", "runtime", or "none"
- `module_version` - Links trace to specific code version

**On contract violation:**
```json
{
  "fn": "policies.refund.refundEligibility",
  "args": { "status": "DELAYED", "hoursDelayed": -1, "customerCredits": 0 },
  "effects": [],
  "contracts": {
    "requires": {
      "status": "FAILED",
      "id": "req_a1b2c3",
      "violation": "hoursDelayed >= 0",
      "actual": { "hoursDelayed": -1 },
      "counterexample": true
    }
  },
  "verification": { "static": false, "method": "runtime" },
  "result": null,
  "error": "ContractViolation",
  "timestamp": "2025-12-06T10:31:00Z",
  "trace_id": "tr-def456"
}
```

**Trace use cases:**
1. **Compliance audit**: "Show me all executions where contract X was checked"
2. **Training data**: "Million traces of contract pass/fail for fine-tuning"
3. **Debugging**: "Why did this specific call violate the contract?"
4. **Coverage**: "Which contracts have never been exercised at runtime?"

---

### Implementation Plan

> **Updated December 2025**: Estimates revised based on existing Property and Debug infrastructure.

**Phase 0: Plumbing** (~1 sprint / 8-12 hours) ⬇️ Reduced from 15-20h

Leveraging existing `Property` and `DebugContext` infrastructure:

- [ ] Add `ContractKind` enum to `ast.Property` (discriminate requires/ensures/invariant)
- [ ] Add `REQUIRES`, `ENSURES`, `INVARIANT` tokens to lexer (~10 LOC)
- [ ] Extend function parser for `requires { ... }` and `ensures { ... }` blocks
  - Reuse `parseExpression()` for contract predicates
  - Store in `FuncDecl.Properties` with appropriate `ContractKind`
- [ ] Add pretty-printer support for contracts
- [ ] Unit tests for contract parsing

**Phase 0.5: Runtime Contract Checks** (~0.5-1 sprint / 5-8 hours) 🆕 Quick Win

Delivers immediate value before SMT backend:

- [ ] Create `ContractContext` with `ContractMode` (Panic/Report/Off) (~120 LOC)
- [ ] Add `Contract *ContractContext` field to `EffContext`
- [ ] Generate Go runtime checks in typed wrappers:
  - `requires`: check at function entry
  - `ensures`: check before return
  - **Mode-dependent behavior**:
    - `Panic`: Go panic (fast feedback in development)
    - `Report`: Return `ContractViolation` error + structured trace (policy mode)
- [ ] Add `--verify-contracts[=panic|report]` flag (off by default)
- [ ] Wire `ContractContext.Collect()` to trace output
- [ ] Make `Args` optional in traces (`--trace-args` flag to enable)
- [ ] Integration tests for both panic and report modes

**Phase 1: SMT Backend MVP** (~2-3 sprints / 25-35 hours) ⚠️ HIGH UNCERTAINTY

**Prerequisite**: Implement `IsSMTEncodable()` function first (used everywhere):
- [ ] `IsSMTEncodable(funcDecl) -> (bool, []SMTRejectionReason)`
- [ ] Wire to compiler warnings, `ailang verify` output, and trace `reasons` field
- [ ] Rejection codes: `RECURSIVE`, `HIGHER_ORDER`, `DEEP_ADT`, `QUANTIFIED`, `EFFECTFUL`

**Incremental body encoding** (not all-at-once):
- [ ] Step 1: Contracts only (uninterpreted function model)
- [ ] Step 2: Straight-line `let` chains → SMT `let`
- [ ] Step 3: `if/else` → SMT `ite`
- [ ] Step 4: `match` over enum → SMT datatype match
- [ ] Step 5: `match` over single-level ADT → SMT datatype constructors

**Type mapping** (with `float` caveat):
- [ ] `int` → `Int`, `bool` → `Bool`, enums → `declare-datatype`
- [ ] `float` → `Real` **only under `smt-best-effort`** with warning:
  `note: float verified as mathematical real, not IEEE754`

**CLI and integration**:
- [ ] Add `ailang verify <module>` command
- [ ] Shell out to Z3 (or pluggable solver)
- [ ] Parse sat/unsat + optional model
- [ ] Integration tests with Z3

**Phase 2: Redundant Generation** (~1-2 sprints / 10-15 hours)
- [ ] Add CLI mode: `ailang ai-gen <spec> --redundant=N`
- [ ] Implement AST normalization:
  - [ ] Alpha-renaming pass
  - [ ] Pattern match desugaring
  - [ ] Trivial let inlining
- [ ] Implement structural equivalence check
- [ ] Define confidence semantics and JSON output
- [ ] Integration tests with mock LLM responses

**Phase 3: SharedMem Invariants** (~ongoing / 20+ hours)
- [ ] Extend verifier fragment:
  - [ ] Structural recursion
  - [ ] Pattern matching over ADTs
  - [ ] Limited quantification (optional)
- [ ] Add SharedMem-aware "check functions"
- [ ] Encode SharedMem invariants to SMT summaries
- [ ] Integrate traces + verification into semantic cache

---

### Files to Modify/Create

> **Updated December 2025**: File estimates revised to reflect reuse of existing infrastructure.

**New files:**

```
internal/
├── contracts/
│   ├── context.go       -- ContractContext with ContractMode (~120 LOC)
│   ├── checker.go       -- Contract type checking (~150 LOC)
│   ├── codegen.go       -- Runtime check generation (panic + error modes) (~180 LOC)
│   └── encodable.go     -- IsSMTEncodable() + SMTRejectionReason (~100 LOC) 🆕
├── smt/
│   ├── codegen.go       -- SMT-LIB generation (~400 LOC)
│   ├── types.go         -- AILANG->SMT type mapping + float caveat (~120 LOC)
│   ├── solver.go        -- Z3 invocation + result parsing (~150 LOC)
│   └── models.go        -- Counterexample parsing (~100 LOC)
├── redundant/
│   ├── normalize.go     -- AST normalization (~200 LOC)
│   ├── compare.go       -- Structural equivalence (~150 LOC)
│   ├── confidence.go    -- Confidence scoring (~100 LOC)
│   └── protocol.go      -- Multi-sample protocol (~100 LOC)
└── trace/                  -- NOT NEEDED: Reuse DebugContext pattern
    └── (merged into contracts/context.go)

cmd/ailang/
├── verify.go            -- `ailang verify` command (~100 LOC)
└── aigen.go             -- `ailang ai-gen` command (~100 LOC)

stdlib/
└── std/policy.ail       -- Policy type library (~50 LOC)
```

**Modified files (leveraging existing infrastructure):**

```
internal/
├── ast/ast_decl.go      -- Add ContractKind to Property (~+20 LOC) ⬇️ Reuse Property
├── lexer/token.go       -- Add REQUIRES, ENSURES, INVARIANT (~+10 LOC)
├── parser/parser_decl.go -- Parse requires/ensures blocks (~+80 LOC) ⬇️ Reuse parseExpression
├── effects/context.go   -- Add Contract *ContractContext (~+5 LOC) ⬇️ Follow Debug pattern
├── gen/golang/codegen_decl.go -- Generate runtime checks (~+100 LOC)
└── pipeline/pipeline.go -- Contract validation pass (~+30 LOC)
```

**Comparison: Original vs Revised Estimates**

| Component | Original | Revised | Savings | Reason |
|-----------|----------|---------|---------|--------|
| contracts/ast.go | 150 | 0 | 100% | Reuse `ast.Property` |
| contracts/parser.go | 200 | 0 | 100% | Reuse `parseExpression()` |
| trace/* | 300 | 0 | 100% | Reuse `DebugContext` pattern |
| AST modifications | 50 | 20 | 60% | Just add `ContractKind` |
| Parser modifications | 100 | 80 | 20% | Simpler wiring |
| **Total** | **2,500** | **~1,700** | **~32%** | |

**Total estimated new code:** ~1,450 LOC (down from ~2,200)
**Total estimated modified code:** ~245 LOC (down from ~300)

---

## Examples

### Example 0: Park Admission (ARC Paper Showcase)

This example reproduces the park admission policy from the ARC paper (Bayless et al., 2025), demonstrating that AILANG can express the same policy model with machine-checkable contracts.

```ailang
-- examples/contracts/park.ail
-- Reproduces ARC paper park admission example in AILANG
module examples/contracts/park

-- Type vocabulary (matching ARC's datatypes/variables)
export type Season = LOW_SEASON | HIGH_SEASON
export type AgeClass = CHILD | ADULT | SENIOR
export type AdmissionDecision = ADMIT | DENY

-- Core policy function with contracts
-- ARC rule: "Seniors pay $5 in low season, $10 in high season"
-- ARC rule: "Children under 5 free in low season"
export func admissionFee(age: int, season: Season) -> int ! {}
requires { age >= 0 }
ensures  { result >= 0 }
ensures  { age >= 65 && season == LOW_SEASON => result == 5 }
ensures  { age >= 65 && season == HIGH_SEASON => result == 10 }
ensures  { age < 5 && season == LOW_SEASON => result == 0 }
{
  match season {
    LOW_SEASON =>
      if age < 5 { 0 }
      else if age >= 65 { 5 }
      else { 15 },
    HIGH_SEASON =>
      if age >= 65 { 10 }
      else { 20 }
  }
}

-- Decision function: can visitor enter park given budget?
export func canEnterPark(age: int, season: Season, budget: int) -> AdmissionDecision ! {}
requires { age >= 0, budget >= 0 }
ensures  { result == ADMIT => budget >= admissionFee(age, season) }
{
  if budget >= admissionFee(age, season) { ADMIT } else { DENY }
}
```

**Verification output:**
```bash
$ ailang verify examples/contracts/park

Verifying examples.contracts.park.admissionFee...
  requires: age >= 0                              [OK]
  ensures:  result >= 0                           [OK]
  ensures:  age >= 65 && season == LOW_SEASON => result == 5   [OK]
  ensures:  age >= 65 && season == HIGH_SEASON => result == 10 [OK]
  ensures:  age < 5 && season == LOW_SEASON => result == 0     [OK]

  Result: VERIFIED (0.08s, Z3 4.12.2)

Verifying examples.contracts.park.canEnterPark...
  requires: age >= 0, budget >= 0                 [OK]
  ensures:  result == ADMIT => budget >= admissionFee(age, season) [OK]

  Result: VERIFIED (0.12s, Z3 4.12.2)

Summary: 2 functions verified, 0 violations, 0 skipped
```

This is a **killer showcase**: side-by-side with the ARC paper, showing AILANG can express the same semantics with formal guarantees.

---

### Example 1: Policy Function with Contracts

**Before (no contracts):**
```ailang
module policies/refund

export func refundEligibility(status: FlightStatus, hoursDelayed: int)
  -> RefundEligibility ! {}
{
  match status {
    CANCELLED => ELIGIBLE,
    DELAYED   => if hoursDelayed >= 5 { ELIGIBLE } else { NOT_ELIGIBLE },
    DEPARTED  => NOT_ELIGIBLE
  }
}
```

**After (with contracts):**
```ailang
module policies/refund

export func refundEligibility(status: FlightStatus, hoursDelayed: int)
  -> RefundEligibility ! {}
requires { hoursDelayed >= 0 }
ensures  { result == ELIGIBLE => hoursDelayed >= 5 }
{
  match status {
    CANCELLED => ELIGIBLE,
    DELAYED   => if hoursDelayed >= 5 { ELIGIBLE } else { NOT_ELIGIBLE },
    DEPARTED  => NOT_ELIGIBLE
  }
}
```

### Example 2: Game Logic Verification (General Use)

Demonstrating that contracts work beyond policy domains - here for game invariants:

```ailang
-- examples/contracts/game_hp.ail
-- Game health system with invariant contracts
module examples/contracts/game_hp

export type DamageResult = ALIVE | DEAD

-- Health can never go negative or exceed max
export func applyDamage(currentHP: int, maxHP: int, damage: int) -> int ! {}
requires { currentHP >= 0, currentHP <= maxHP, maxHP > 0, damage >= 0 }
ensures  { result >= 0 }
ensures  { result <= maxHP }
ensures  { damage > 0 => result < currentHP || result == 0 }
{
  let newHP = currentHP - damage
  if newHP < 0 { 0 } else { newHP }
}

-- Healing respects max cap
export func applyHealing(currentHP: int, maxHP: int, healing: int) -> int ! {}
requires { currentHP >= 0, currentHP <= maxHP, maxHP > 0, healing >= 0 }
ensures  { result >= currentHP }
ensures  { result <= maxHP }
{
  let newHP = currentHP + healing
  if newHP > maxHP { maxHP } else { newHP }
}
```

This shows the **General Verification** layer: same machinery, different domain (games vs policy).

---

### Example 3: Verification CLI

```bash
# Verify a module
$ ailang verify policies/refund

Verifying policies.refund.refundEligibility...
  requires: hoursDelayed >= 0              [OK]
  ensures:  result == ELIGIBLE => hoursDelayed >= 5

  Result: VERIFIED (0.12s, Z3 4.12.2)

# Contract violation found
$ ailang verify policies/buggy_refund

Verifying policies.buggy_refund.refundEligibility...
  ensures: result == ELIGIBLE => hoursDelayed >= 5

  Result: COUNTEREXAMPLE FOUND

  Inputs:
    status = CANCELLED
    hoursDelayed = 2

  Violation: result = ELIGIBLE but hoursDelayed < 5

  Hint: CANCELLED case returns ELIGIBLE unconditionally,
        but ensures clause requires hoursDelayed >= 5
```

### Example 4: Redundant Generation

```bash
$ ailang ai-gen "function to calculate refund eligibility" --redundant=3

Generating 3 candidate implementations...

Sample 1: [parsed OK]
Sample 2: [parsed OK]
Sample 3: [parsed OK]

Normalizing ASTs...

Comparison:
  Sample 1 hash: abc123
  Sample 2 hash: abc123  (equivalent to Sample 1)
  Sample 3 hash: def456  (DIVERGENT)

Classification: EquivalentMajority
Confidence: 0.67 (2/3 agree)

Difference in Sample 3:
  Line 5: Uses `hoursDelayed > 5` instead of `hoursDelayed >= 5`

Recommendation: Review Sample 3 logic for off-by-one error
```

### Example 5: Execution Trace with Contracts

```bash
$ ailang run --trace policies/refund --entry test

[TRACE] fn=policies.refund.refundEligibility
        args={status: DELAYED, hoursDelayed: 6}
        contracts.requires: OK
        result: ELIGIBLE
        contracts.ensures: OK
        duration: 0.001ms

[TRACE] fn=policies.refund.refundEligibility
        args={status: DELAYED, hoursDelayed: -1}
        contracts.requires: FAILED (hoursDelayed >= 0)
        violation: hoursDelayed = -1

ContractViolation: requires clause failed for refundEligibility
```

---

## Success Criteria

- [ ] Contract syntax parses correctly for requires/ensures blocks
- [ ] Runtime contract checks trigger panic on violation (debug mode)
- [ ] SMT backend generates valid SMT-LIB for restricted fragment
- [ ] Z3 invocation returns sat/unsat with model on sat
- [ ] Counterexamples display human-readable variable bindings
- [ ] Redundant generation produces normalized AST hashes
- [ ] Confidence classification matches expected for test cases
- [ ] Execution traces include contract check results
- [ ] `ailang verify` command works end-to-end
- [ ] `ailang ai-gen --redundant` command works end-to-end
- [ ] All tests passing (target: 90%+ coverage for new packages)
- [ ] Documentation updated (contracts guide, verification guide)
- [ ] Examples added (10+ contract examples in examples/contracts/)

---

## Testing Strategy

**Unit tests:**
- Contract AST construction and validation
- Contract expression parsing
- SMT-LIB generation for each supported type/operator
- AST normalization correctness (alpha-renaming, desugar)
- Structural equivalence detection
- Confidence score calculation

**Integration tests:**
- Full pipeline: AILANG -> SMT -> Z3 -> result
- Contract violations detected at runtime
- Trace generation with contract metadata
- Multi-sample generation protocol

**Property tests:**
- Random contract expressions round-trip through parser
- Normalized ASTs are idempotent (normalize(normalize(x)) == normalize(x))
- Valid contracts always produce valid SMT-LIB

**Manual testing:**
- Z3 solver installation and PATH detection
- Large contract expressions (stress test)
- Edge cases: empty contracts, trivially true/false contracts

---

## Non-Goals

**Not in this feature:**
- **Full dependent types** - Contracts are refinements, not types. Deferred to v0.7.0+ if needed.
- **Natural language policy parsing** - ARC's PMC-style autoformalization is future work. For v0.6.0, policy authors write AILANG directly.
- **Quantified formulas** - Initial SMT fragment is quantifier-free. Can extend later for bounded quantification.
- **Recursive function verification** - Requires induction; deferred to Phase 3+.
- **Distributed verification** - Single-machine Z3 invocation only.
- **IDE integration** - No LSP support; AI uses CLI/API.

---

## Timeline

> **Updated December 2025**: Revised based on reuse of existing Property and Debug infrastructure.

**Sprint 1** (8-12 hours) ⬇️ Reduced:
- Phase 0: Contract AST extension, parser, lexer tokens
- Deliverable: Contracts parse into `FuncDecl.Properties` with `ContractKind`
- Risk: Very Low - leveraging existing `Property` infrastructure

**Sprint 2** (5-8 hours) 🆕 Quick Win:
- Phase 0.5: Runtime contract checks
- Deliverable: `--verify-contracts` flag triggers runtime panics on violation
- Risk: Low - follows proven `DebugContext` pattern
- **Value**: Immediate usefulness before SMT backend

**Sprint 3-5** (25-35 hours, **HIGH UNCERTAINTY**):
- Phase 1: SMT backend, Z3 integration
- Deliverable: `ailang verify` works for simple functions
- Risk: **SMT encoding + solver integration + real error reporting** is where scope typically explodes
- Goal: "Small real examples verified" (like park admission), NOT "complete stdlib coverage"

**Sprint 6-7** (10-15 hours):
- Phase 2: Redundant generation protocol
- Deliverable: `ailang ai-gen --redundant` works
- Risk: Medium - depends on contract filter working reliably

**Sprint 8+** (ongoing):
- Phase 3: Extended verification, SharedMem invariants
- Deliverable: More complex contracts, policy state verification
- Risk: High - recursion/induction is hard

**Revised Total: ~48-70 hours across 6-10 weeks** (down from 50-70h / 8-12 weeks)

**Uncertainty notes:**
- Estimates are best-effort heuristics, especially for Phase 1 (SMT)
- **Key de-risk strategy**: Phase 0.5 delivers runtime checks first; SMT can slip without blocking all value
- If SMT encoding proves harder than expected, ship runtime-only in v0.6.2 and defer SMT to v0.6.3
- Success criterion for Phase 1: "park admission example verifies end-to-end"

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| SMT solver availability | High | Bundle Z3 binary or provide clear install instructions; support pluggable solvers |
| SMT encoding complexity | Medium | Start with restricted fragment; expand gradually |
| Performance (large modules) | Medium | Verify per-function, not whole module; cache results |
| False positives from incomplete encoding | High | Be conservative; only claim verified for fully-encoded functions |
| LLM variance in redundant gen | Medium | Use temperature=0; test with multiple models |
| Contract syntax conflicts | Low | Careful parser design; contracts are annotations, not expressions |

---

## References

### Research Foundation
- **ARC Paper**: Bayless et al., "A Neurosymbolic Approach to Natural Language Formalization and Verification" (AWS, 2025) ([arXiv:2511.09008](https://arxiv.org/abs/2511.09008))
  - Note: ARC = **Automated Reasoning Checks** (AWS system), not Anthropic's Alignment Research Center
  - Key techniques adapted: Redundant autoformalization, SMT-based verification, soundness-first design

### Technical Standards
- **SMT-LIB Standard**: [smtlib.org](https://smtlib.org/) - Target format for verification conditions
- **Z3 Theorem Prover**: [github.com/Z3Prover/z3](https://github.com/Z3Prover/z3) - Primary SMT solver

### AILANG Internal
- **Effects Design**: [design_docs/implemented/v0_2_0/effects.md](../../../design_docs/implemented/v0_2_0/effects.md) - Existing effect system to integrate with
- **AI-First DX Philosophy**: [example-parity-vision-alignment.md](../v0_3_15/example-parity-vision-alignment.md) - Design principles this feature follows

### Existing Infrastructure to Leverage (Added December 2025)
- **Property AST**: `internal/ast/ast_decl.go` - `Property`, `Binder`, `FuncDecl.Properties`
- **Property Parser**: `internal/parser/parser_testing.go` - `parseProperty()`, `parseBinder()`
- **Debug Effect**: `internal/effects/debug.go` - `DebugContext` pattern for trace collection
- **Pipeline Metrics**: `internal/pipeline/metrics.go` - Telemetry infrastructure model
- **Testing Examples**: `examples/testing_basic.ail` - Existing property-based testing syntax

### Showcase Examples (to be created)
- `examples/contracts/park.ail` - Reproduces ARC paper's park admission policy (killer showcase for README/blog)
- `examples/contracts/game_hp.ail` - Game health invariants (shows general verification beyond policy)
- `examples/contracts/refund.ail` - Airline refund policy with contracts

---

## Future Work

**v0.7.0+ potential extensions:**
- **Bounded quantification**: `forall x in xs. P(x)` for finite collections
- **Inductive verification**: Recursive function contracts with termination proofs
- **SharedMem invariants**: Policy constraints over semantic cache state
- **NL policy parsing**: Integration with ARC-style autoformalization pipeline
- **Verification caching**: Store proof results in semantic cache
- **IDE confidence display**: Show redundant generation confidence in hover
- **Cross-module contracts**: Verify contracts across module boundaries

---

**Document created**: 2025-12-06
**Last updated**: 2025-12-18
