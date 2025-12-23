# M-D4: Design-Doc-Driven Development (D4)

**Status**: Planned
**Target**: v0.8.0
**Priority**: P1 (High) - Foundation for trusted AI code generation
**Estimated**: 3-4 weeks
**Dependencies**:
- Runtime contracts (v0.6.2) - COMPLETE
- Effect/capability system (v0.2.0) - COMPLETE
- Budget system - NEW (this feature)
- Sprint planner/executor skills - COMPLETE

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Design docs become deterministic specifications |
| A2: Replayability | +2 | Full audit trail from requirement → design → code |
| A3: Effect Legibility | +2 | Effects declared in design doc, enforced at runtime |
| A4: Explicit Authority | +2 | Capabilities explicitly declared in spec |
| A5: Bounded Verification | +2 | Local verification: does code match its design doc? |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | YAML frontmatter is machine-readable spec |
| A8: Minimal Syntax | -1 | Adds YAML schema requirements |
| A9: Cost Visibility | +2 | Budgets declared in design docs |
| A10: Composability | +1 | Design docs compose with contracts |
| A11: Structured Failure | +1 | Spec violations are typed errors |
| A12: System Boundary | +2 | Clear boundary between spec and implementation |

**Net Score: +17** → **Decision: Strong go - core to AILANG's mission**

### Hard Violation Check

- [x] A1 (Determinism): Specifications are static documents
- [x] A3 (Effects): Effects are declared, not hidden
- [x] A4 (Authority): Capabilities are explicitly granted
- [x] A7 (Machines First): Primary goal is machine-readable specs

---

## Design Review Feedback (December 2025)

**Reviewer verdict:** ✅ On-mission. Directly operationalizes A3/A4/A5/A9/A12 with a credible trust chain from spec → compile → runtime → audit.

### Key Refinements Incorporated

| Issue | Original | Revised |
|-------|----------|---------|
| Verification result | Boolean OK/FAIL | Tri-state: PROVED / CHECKED_AT_RUNTIME / UNKNOWN |
| Contract matching | String comparison | AST normalization + stable hash IDs |
| Adoption model | All-or-nothing | Spec Profiles (strict/warn/off per category) |
| Budget metering | Generated wrappers | Effect boundary handlers |
| Effect constraints | `map[string]any` | First-class typed per effect |
| Conceptual model | "Spec" | Grant + Obligations + Envelope |

### Scope Control (v0.8.0 MVP)

**Must-ship:**
1. Spec parser + schema validation
2. BudgetContext + runtime enforcement (api_calls + execution_ms)
3. Effect allow/deny (permitted/forbidden capabilities)
4. Contract injection from spec (requires/ensures)
5. `ailang verify --spec` as consistency checker

**Deferred:**
- Static "max API calls" proofs (unless restricted fragment)
- Token/cost budgets (depends on LLM execution boundary control)
- Host allowlist via URL string scanning (enforce in Net effect instead)
- Sprint-planner integration (nice, not core correctness)

---

## Problem Statement

**The Trust Gap in AI Code Generation:**

Users specify requirements → AI generates design docs → AI implements code → But how does the user know the implementation matches their specification?

**Current State:**
- Design docs are human-readable markdown with light YAML frontmatter
- No formal link between design doc specifications and code contracts
- Effects/capabilities declared in code, not in requirements
- No budget constraints from requirements to implementation
- User must manually verify implementation matches specification

**Impact:**
- Users cannot trust AI-generated code matches their intent
- No formal verification that effects used are within spec
- No budget enforcement from design to runtime
- Audit trail is manual and incomplete
- AI agents operate with implicit authority, not declared bounds

---

## Vision: The D4 Workflow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    DESIGN-DOC-DRIVEN DEVELOPMENT                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. USER REQUIREMENTS                                                │
│     "I need a function to fetch user data from API"                  │
│     "Max 5 API calls, no file system access, budget $0.10"           │
│                                                                      │
│  2. DESIGN DOC (structured YAML frontmatter)                         │
│     ┌──────────────────────────────────────────────────────────┐     │
│     │ effects:                                                  │     │
│     │   permitted: [Net]                                        │     │
│     │   forbidden: [FS, Shell]                                  │     │
│     │ budgets:                                                  │     │
│     │   api_calls: 5                                            │     │
│     │   cost_usd: 0.10                                          │     │
│     │ contracts:                                                │     │
│     │   requires:                                               │     │
│     │     - "userId > 0"                                        │     │
│     │   ensures:                                                │     │
│     │     - "result.status in [OK, NOT_FOUND]"                  │     │
│     └──────────────────────────────────────────────────────────┘     │
│                                                                      │
│  3. CODE GENERATION (constrained by spec)                            │
│     - Compiler extracts spec from design doc YAML                    │
│     - Generates code with contracts from spec                        │
│     - Injects budget tracking into effects                           │
│     - Runtime enforces spec constraints                              │
│                                                                      │
│  4. VERIFICATION (automatic)                                         │
│     ┌──────────────────────────────────────────────────────────┐     │
│     │ ailang verify --spec design_docs/user_fetch.md module.ail │     │
│     │                                                           │     │
│     │ ✅ Effects: Net permitted, FS forbidden (no FS calls)     │     │
│     │ ✅ Contracts: requires/ensures match spec                 │     │
│     │ ✅ Budgets: api_calls=3/5, cost=$0.07/$0.10              │     │
│     │                                                           │     │
│     │ VERIFICATION PASSED                                       │     │
│     └──────────────────────────────────────────────────────────┘     │
│                                                                      │
│  5. USER TRUST                                                       │
│     "The code does what I specified - I have formal proof"           │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Goals

**Primary Goal:** Enable verified trust that AI-generated code matches user specifications through formal linking of design documents to code contracts, effects, and budgets.

**Success Metrics:**
- Design doc YAML schema for effects, budgets, contracts
- `ailang verify --spec DESIGN_DOC CODE` command
- Runtime budget enforcement with spec limits
- Effect usage validated against spec permissions
- Contract generation from spec predicates
- Audit trail JSON linking spec → code → runtime

**Non-Goals (v0.8.0):**
- Natural language → spec generation (future NLP integration)
- Visual spec editor (CLI/YAML only)
- Distributed budget coordination (single-process only)

---

## Solution Design

### Overview

Three integrated subsystems:

1. **Spec Schema** - Extended YAML frontmatter in design docs
2. **Spec Compiler** - Extract and validate specs, inject into compilation
3. **Spec Verifier** - Static and runtime verification against specs

### Component 1: Design Doc Spec Schema

**Extended YAML frontmatter:**

```yaml
---
# Standard design doc metadata
Status: Planned
Target: v0.8.0
Priority: P1
Estimated: 2 hours
Dependencies: []

# D4: Specification Section (NEW)
spec:
  # Version of the spec schema (for evolution)
  schema_version: "1.0"

  # Spec Profiles - gradual adoption (NEW from feedback)
  policy:
    effects: strict      # strict | warn | off
    budgets: runtime     # strict | runtime | warn | off
    contracts: strict    # strict | warn | off

  # ═══════════════════════════════════════════════════════════════
  # GRANTS - Capability permissions (what the code is allowed to do)
  # ═══════════════════════════════════════════════════════════════
  grants:
    effects:
      permitted:
        - IO           # Can print to stdout
        - Net          # Can make network requests
      forbidden:
        - FS           # No file system access
        - Shell        # No shell execution

    # First-class typed constraints per effect (not map[string]any)
    Net:
      allowed_hosts: ["api.example.com"]
      require_https: true
      # max_requests moved to envelope (budget)

  # ═══════════════════════════════════════════════════════════════
  # ENVELOPE - Resource budgets (what the code is allowed to consume)
  # ═══════════════════════════════════════════════════════════════
  envelope:
    api_calls: 5                  # Max API invocations
    execution_ms: 5000            # Max execution time
    # tokens/cost_usd deferred until LLM boundary is clear

  # ═══════════════════════════════════════════════════════════════
  # OBLIGATIONS - Contracts (what the code must guarantee)
  # ═══════════════════════════════════════════════════════════════
  obligations:
    # Preconditions for functions
    requires:
      - name: "valid_user_id"
        predicate: "userId > 0"
        message: "User ID must be positive"
      - name: "valid_format"
        predicate: "format in ['json', 'xml']"

    # Postconditions
    ensures:
      - name: "valid_response"
        predicate: "result.status in [OK, NOT_FOUND]"
      - name: "format_matches"
        predicate: "result.format == format"

  # Which functions this spec applies to
  applies_to:
    - "fetchUser"
    - "fetchUserBatch"

  # Link to implementation
  implementation:
    module: "examples/user_api.ail"
    entry: "fetchUser"
---
```

### Conceptual Model: Grant + Obligations + Envelope

A spec explicitly declares three orthogonal concerns:

| Concern | Description | Violation Type |
|---------|-------------|----------------|
| **Grants** | Capability permissions | "You used FS but FS is forbidden" |
| **Envelope** | Resource budgets | "You exceeded api_calls limit" |
| **Obligations** | Contract invariants | "Postcondition failed" |

This maps cleanly to AILANG's existing effects/contracts worldview.

### Component 2: Spec Compiler

**New package: `internal/spec/`**

**Workflow:**
1. Parse design doc markdown
2. Extract YAML frontmatter
3. Validate against spec schema
4. Generate `SpecConstraints` struct
5. Pass to compilation pipeline

```go
// internal/spec/spec.go

type SpecConstraints struct {
    SchemaVersion string

    // Effect permissions
    Effects EffectSpec

    // Budget limits
    Budgets BudgetSpec

    // Contract predicates (pre-parsed AST)
    Contracts ContractSpec

    // Target functions
    AppliesTo []string

    // Source design doc
    SourceDoc string
}

type EffectSpec struct {
    Permitted  []string            // ["IO", "Net"]
    Forbidden  []string            // ["FS", "Shell"]
    Constraints map[string]any     // Net -> {allowed_hosts: [...]}
}

type BudgetSpec struct {
    APIcalls     *int     // Max API calls
    Tokens       *int     // Max tokens
    CostUSD      *float64 // Max cost
    ExecutionMS  *int     // Max execution time
}

type ContractSpec struct {
    Requires   []ContractPredicate
    Ensures    []ContractPredicate
    Invariants []ContractPredicate
}

type ContractPredicate struct {
    Name       string
    Predicate  string      // AILANG expression as string
    Message    string
    ParsedAST  ast.Expr    // Pre-parsed for codegen
    ContractID string      // Stable hash of normalized AST (NEW)
}
```

### Contract Normalization and Hashing (Avoids Drift)

**Problem:** If spec contracts are matched by string, you get constant friction from syntactic differences.

**Solution:** Treat spec as source of truth, compile to canonical contract IDs:

```go
// internal/spec/normalize.go

// NormalizeContract produces a stable hash for contract matching
func NormalizeContract(predicate string) (normalizedAST ast.Expr, contractID string, err error) {
    // 1. Parse predicate into AST
    parsed, err := parser.ParseExpression(predicate)
    if err != nil {
        return nil, "", err
    }

    // 2. Normalize the AST:
    //    - Alpha-rename all bound variables to canonical names
    //    - Sort commutative operators (a + b → canonical order)
    //    - Constant fold where possible
    normalized := normalize.AlphaRename(parsed)
    normalized = normalize.SortCommutative(normalized)
    normalized = normalize.ConstantFold(normalized)

    // 3. Hash the normalized AST to produce stable ID
    serialized := ast.Serialize(normalized)  // Deterministic
    hash := sha256.Sum256(serialized)
    contractID = hex.EncodeToString(hash[:8])  // First 8 bytes

    return normalized, contractID, nil
}
```

**Matching logic:**

```go
// Does function contain contract IDs required by spec?
func VerifyContracts(spec *SpecConstraints, core *core.Program) []VerificationItem {
    var items []VerificationItem

    for _, required := range spec.Obligations.Requires {
        found := false
        for _, fn := range core.Functions {
            for _, contract := range fn.Contracts {
                if contract.ContractID == required.ContractID {
                    found = true
                    break
                }
            }
        }

        if found {
            items = append(items, VerificationItem{
                Category:   "obligations",
                Name:       required.Name,
                State:      Proved,
                ContractID: required.ContractID,
            })
        } else {
            items = append(items, VerificationItem{
                Category:   "obligations",
                Name:       required.Name,
                State:      Unknown,
                ContractID: required.ContractID,
                Details:    fmt.Sprintf("Contract ID %s not found", required.ContractID),
            })
        }
    }

    return items
}
```

**Parsing flow:**

```go
// internal/spec/parse.go

func ParseDesignDoc(path string) (*SpecConstraints, error) {
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    // Extract YAML frontmatter between --- markers
    frontmatter, err := extractFrontmatter(content)
    if err != nil {
        return nil, err
    }

    // Parse YAML into SpecDoc
    var doc SpecDoc
    if err := yaml.Unmarshal(frontmatter, &doc); err != nil {
        return nil, err
    }

    // Validate against schema
    if err := validateSpecSchema(&doc); err != nil {
        return nil, err
    }

    // Convert to SpecConstraints
    return buildConstraints(&doc)
}
```

### Component 3: Budget System (Metered at Effect Boundaries)

**This is a dependency that must be built for D4.**

**Key design:** Budget increments happen in effect handlers, not in generated wrappers. This keeps enforcement aligned with "authority lives at the boundary."

| Budget Type | Where Metered | Why |
|-------------|---------------|-----|
| `api_calls` | Net effect handler | Each network call increments |
| `execution_ms` | Entry/exit of main | Measured centrally, not per-call |
| `tokens` | AI effect handler | Only where token usage is known |
| `cost_usd` | AI effect handler | Deferred until LLM boundary clear |

**New package: `internal/effects/budget.go`**

```go
// BudgetContext tracks resource consumption against limits
type BudgetContext struct {
    mu sync.Mutex  // Thread-safe for concurrent effects

    // Configured limits (from spec)
    Limits BudgetLimits

    // Current consumption
    Usage BudgetUsage

    // Start time for execution_ms tracking
    StartTime time.Time

    // Violation handler (configurable per policy)
    OnViolation func(kind string, limit, actual any) error
}

type BudgetLimits struct {
    APIcalls    *int      // Max network/API calls
    ExecutionMS *int      // Max wall-clock time
    // tokens/cost_usd deferred until AI effect boundary is defined
}

type BudgetUsage struct {
    APIcalls    int
    ExecutionMS int64  // Computed on demand from StartTime
}

// ConsumeAPICall is called by Net effect handler (not generated code)
func (bc *BudgetContext) ConsumeAPICall() error {
    bc.mu.Lock()
    defer bc.mu.Unlock()

    bc.Usage.APIcalls++

    if bc.Limits.APIcalls != nil && bc.Usage.APIcalls > *bc.Limits.APIcalls {
        return bc.OnViolation("api_calls", *bc.Limits.APIcalls, bc.Usage.APIcalls)
    }
    return nil
}

// CheckExecutionTime is called periodically or at key points
func (bc *BudgetContext) CheckExecutionTime() error {
    if bc.Limits.ExecutionMS == nil {
        return nil
    }

    elapsed := time.Since(bc.StartTime).Milliseconds()
    bc.Usage.ExecutionMS = elapsed

    if elapsed > int64(*bc.Limits.ExecutionMS) {
        return bc.OnViolation("execution_ms", *bc.Limits.ExecutionMS, elapsed)
    }
    return nil
}
```

**Integration with Net effect handler:**

```go
// internal/effects/net.go

func (n *NetContext) Get(ctx *EffContext, url string) (Response, error) {
    // Budget check FIRST (at effect boundary)
    if ctx.Budget != nil {
        if err := ctx.Budget.ConsumeAPICall(); err != nil {
            return Response{}, fmt.Errorf("envelope violation: %w", err)
        }
    }

    // Then actual network call
    return n.doRequest("GET", url)
}
```

**Integration with EffContext:**

```go
// internal/effects/context.go

type EffContext struct {
    IO       IOContext
    FS       FSContext
    Net      NetContext
    Clock    ClockContext
    Debug    *DebugContext
    Contract *ContractContext
    Budget   *BudgetContext    // NEW
    Spec     *SpecConstraints  // NEW - source spec for verification
}
```

### Component 4: Spec Verifier (Tri-State)

**Verification produces tri-state results, not booleans:**

| State | Meaning | Symbol |
|-------|---------|--------|
| **PROVED** | Statically verified, sound | ✅ |
| **CHECKED_AT_RUNTIME** | Instrumented, enforced at execution | 🔵 |
| **UNKNOWN** | Cannot be verified | ⚠️ |

**New command: `ailang verify --spec`**

```bash
# Verify implementation against design doc spec
ailang verify --spec design_docs/planned/v0_8_0/user_api.md examples/user_api.ail

# Output:
Verifying examples/user_api.ail against spec: design_docs/planned/v0_8_0/user_api.md

Grants (Effects):
  ✅ PROVED: Net permitted, used in fetchUser
  ✅ PROVED: FS forbidden, not used
  ✅ PROVED: Shell forbidden, not used
  🔵 RUNTIME: Net.allowed_hosts enforced in handler

Envelope (Budgets):
  🔵 RUNTIME: api_calls <= 5 (enforced per-call in Net handler)
  🔵 RUNTIME: execution_ms <= 5000 (measured at entry/exit)

Obligations (Contracts):
  ✅ PROVED: requires "valid_user_id" [hash:a1b2c3] found in fetchUser
  ✅ PROVED: ensures "valid_response" [hash:d4e5f6] found in fetchUser
  ⚠️ UNKNOWN: ensures "format_matches" [hash:g7h8i9] NOT FOUND
     Hint: Add 'ensures { result.format == format }' to fetchUser

Summary: 5 PROVED, 3 RUNTIME, 1 UNKNOWN
Overall: WARN (1 missing obligation)
```

**Implementation:**

```go
// internal/verify/spec_verify.go

// VerificationState is tri-state, not boolean
type VerificationState int

const (
    Proved          VerificationState = iota  // Static, sound
    CheckedAtRuntime                          // Instrumented, depends on execution
    Unknown                                   // Cannot be verified
)

type VerificationItem struct {
    Category    string            // "grants", "envelope", "obligations"
    Name        string            // e.g., "valid_user_id"
    State       VerificationState
    Details     string            // Human-readable
    ContractID  string            // Stable hash for obligations
}

type SpecVerificationResult struct {
    SpecPath     string
    ModulePath   string

    Items        []VerificationItem

    // Counts by state
    ProvedCount        int
    RuntimeCount       int
    UnknownCount       int

    Overall      VerificationStatus  // PASS, WARN, FAIL
}

func VerifyAgainstSpec(specPath, modulePath string) (*SpecVerificationResult, error) {
    // 1. Parse spec
    spec, err := spec.ParseDesignDoc(specPath)
    if err != nil {
        return nil, fmt.Errorf("parse spec: %w", err)
    }

    // 2. Compile module
    result, err := pipeline.CompileFile(modulePath, pipeline.DefaultConfig())
    if err != nil {
        return nil, fmt.Errorf("compile module: %w", err)
    }

    // 3. Verify effects
    effectResult := verifyEffects(spec.Effects, result.EffectUsage)

    // 4. Verify budgets (static analysis where possible)
    budgetResult := verifyBudgets(spec.Budgets, result.Core)

    // 5. Verify contracts
    contractResult := verifyContracts(spec.Contracts, result.Core)

    return &SpecVerificationResult{
        SpecPath:     specPath,
        ModulePath:   modulePath,
        EffectsOK:    effectResult.OK,
        EffectIssues: effectResult.Issues,
        BudgetsOK:    budgetResult.OK,
        BudgetIssues: budgetResult.Issues,
        ContractsOK:  contractResult.OK,
        ContractIssues: contractResult.Issues,
        Overall:      computeOverall(effectResult, budgetResult, contractResult),
    }, nil
}
```

### Component 5: Runtime Spec Enforcement

**At runtime, enforce spec constraints:**

```go
// Generated Go code with spec enforcement

func fetchUser_impl(ctx *effects.EffContext, userId interface{}) interface{} {
    // Spec: requires userId > 0
    if err := ctx.Contract.CheckRequires(
        runtime.GtInt(userId, 0),
        "valid_user_id: userId > 0",
        "user_api.ail:15",
    ); err != nil {
        panic(err)
    }

    // Spec: budget api_calls <= 5
    if err := ctx.Budget.ConsumeAPICall(); err != nil {
        panic(fmt.Errorf("spec budget exceeded: %w", err))
    }

    // Spec: Net only to api.example.com
    if err := ctx.Net.ValidateHost("api.example.com"); err != nil {
        panic(fmt.Errorf("spec effect constraint: %w", err))
    }

    result := // ... actual implementation

    // Spec: ensures result.status in [OK, NOT_FOUND, ERROR]
    if err := ctx.Contract.CheckEnsures(
        validateStatus(result),
        "valid_response: result.status in [OK, NOT_FOUND, ERROR]",
        "user_api.ail:30",
    ); err != nil {
        panic(err)
    }

    return result
}
```

### Component 6: Sprint Integration

**Enhance sprint-planner to use specs:**

```bash
# Sprint planner reads design doc specs
$ /sprint-planner

Analyzing: design_docs/planned/v0_8_0/user_api.md

Spec Constraints Detected:
  - Effects: Net (permitted), FS (forbidden)
  - Budgets: 5 API calls, $0.10 max
  - Contracts: 2 requires, 2 ensures

Sprint Plan Generated:
  Day 1: Implement fetchUser with contracts
    - Add requires { userId > 0 }
    - Add ensures { result.status in [OK, NOT_FOUND, ERROR] }
  Day 2: Add budget tracking
    - Inject BudgetContext into Net calls
  Day 3: Integration tests
    - Verify spec compliance
```

---

## Implementation Plan

### Phase 1: Budget System (~3 days)

**Prerequisite for D4 - shared with [M-CAPABILITY-BUDGETS (v0.7.0)](../v0_7_0/m-capability-budgets.md).**

This phase implements the **runtime layer** of the unified budget system. The type-level syntax (`@limit=N`) is in v0.7.0; this phase builds the runtime infrastructure both features use.

- [ ] Create `internal/effects/budget.go` (shared by v0.7 + v0.8)
- [ ] Add `BudgetContext` with:
  - `EffectLimits map[string]*EffectBudget` - per-effect budgets
  - `GlobalLimits BudgetLimits` - envelope constraints
  - `Policy BudgetPolicy` - strict/warn/runtime
- [ ] Integrate `BudgetContext` into `EffContext`
- [ ] Add budget-related builtins (`_budget_consume`, `_budget_check`)
- [ ] Unit tests for budget system
- [ ] CLI flag: `--budget api_calls=5,execution_ms=5000`

**See:** [Unified Budget Architecture](../v0_7_0/m-capability-budgets.md#unified-budget-architecture) for how type-level and spec-driven budgets compose.

### Phase 2: Spec Schema (~2 days)

- [ ] Define YAML schema in `internal/spec/schema.go`
- [ ] Create `internal/spec/parse.go` for frontmatter extraction
- [ ] Validate schema with clear error messages
- [ ] Unit tests for spec parsing

### Phase 3: Spec Compiler Integration (~3 days)

- [ ] Add `--spec` flag to `ailang compile`
- [ ] Parse spec and pass to pipeline
- [ ] Generate contracts from spec predicates
- [ ] Inject budget limits into `EffContext`
- [ ] Integration tests

### Phase 4: Spec Verifier (~3 days)

- [ ] Create `internal/verify/spec_verify.go`
- [ ] Implement effect verification
- [ ] Implement budget verification (static analysis)
- [ ] Implement contract verification
- [ ] Add `ailang verify --spec` command
- [ ] Output formats: text, JSON

### Phase 5: Runtime Enforcement (~3 days)

- [ ] Generate budget checks in codegen
- [ ] Generate effect constraint checks
- [ ] Wire spec to runtime context
- [ ] Integration tests for violations

### Phase 6: Sprint Integration (~2 days)

- [ ] Update sprint-planner to parse specs
- [ ] Include spec constraints in sprint plans
- [ ] Update sprint-executor to verify against spec
- [ ] Documentation

### Phase 7: Documentation & Examples (~2 days)

- [ ] Create `docs/docs/guides/design-doc-driven.mdx`
- [ ] Add 5+ example design docs with specs
- [ ] Add 5+ example AILANG modules matching specs
- [ ] Update CLAUDE.md with D4 workflow

---

## Files to Modify/Create

**New files:**

```
internal/
├── spec/
│   ├── spec.go           -- SpecConstraints type (~100 LOC)
│   ├── schema.go         -- YAML schema definition (~150 LOC)
│   ├── parse.go          -- Frontmatter parsing (~200 LOC)
│   └── validate.go       -- Schema validation (~100 LOC)
├── effects/
│   └── budget.go         -- BudgetContext (~250 LOC) [NEW]
├── verify/
│   ├── spec_verify.go    -- Verification engine (~300 LOC)
│   ├── effect_verify.go  -- Effect checking (~100 LOC)
│   ├── budget_verify.go  -- Budget static analysis (~150 LOC)
│   └── contract_verify.go -- Contract matching (~150 LOC)
└── gen/golang/
    └── codegen_budget.go -- Budget codegen (~150 LOC)

cmd/ailang/
├── verify.go             -- `ailang verify --spec` (~100 LOC)
└── compile.go            -- Add --spec flag (~50 LOC)

.claude/skills/
└── sprint-planner/
    └── scripts/          -- Update for spec parsing (~50 LOC)
```

**Modified files:**

```
internal/
├── effects/context.go    -- Add BudgetContext, SpecConstraints (~+30 LOC)
├── pipeline/pipeline.go  -- Accept spec constraints (~+50 LOC)
└── gen/golang/codegen_decl.go -- Generate budget checks (~+100 LOC)
```

**Total new code:** ~1,800 LOC
**Total modified code:** ~180 LOC

---

## Examples

### Example 1: User API with Full Spec

**Design doc: `design_docs/planned/v0_8_0/user_fetch.md`**

```yaml
---
Status: Planned
Target: v0.8.0

spec:
  schema_version: "1.0"

  effects:
    permitted: [Net]
    forbidden: [FS, Shell, IO]
    constraints:
      Net:
        allowed_hosts: ["api.example.com"]
        max_requests: 3

  budgets:
    api_calls: 3
    cost_usd: 0.05

  contracts:
    requires:
      - name: "valid_id"
        predicate: "userId > 0"
    ensures:
      - name: "valid_status"
        predicate: "result.status in [OK, NOT_FOUND]"

  applies_to: ["fetchUser"]
  implementation:
    module: "examples/user_api.ail"
---

# User Fetch Feature

Fetch user data from the API with strict constraints...
```

**Implementation: `examples/user_api.ail`**

```ailang
module examples/user_api

export type Status = OK | NOT_FOUND | ERROR
export type UserResult = { status: Status, data: string }

-- This function's contracts come from spec, but can be overridden
export func fetchUser(userId: int) -> UserResult ! {Net}
requires { userId > 0 }
ensures  { result.status in [OK, NOT_FOUND] }
{
  -- Budget automatically tracked
  let response = _net_get("https://api.example.com/users/" ++ intToString(userId))
  match response {
    { status: 200, body: b } => { status: OK, data: b },
    { status: 404, body: _ } => { status: NOT_FOUND, data: "" },
    _ => { status: ERROR, data: "" }
  }
}
```

**Verification:**

```bash
$ ailang verify --spec design_docs/planned/v0_8_0/user_fetch.md examples/user_api.ail

Spec Verification Report
========================

Source: design_docs/planned/v0_8_0/user_fetch.md
Module: examples/user_api.ail

Effects:
  ✅ Net: permitted and used
  ✅ FS: forbidden and not used
  ✅ Shell: forbidden and not used
  ✅ IO: forbidden and not used
  ✅ Net constraint: only api.example.com accessed

Budgets:
  ✅ api_calls: max 3, implementation uses 1 per call
  ✅ cost_usd: $0.05 limit configured

Contracts:
  ✅ requires "valid_id": found in fetchUser
  ✅ ensures "valid_status": found in fetchUser

Overall: ✅ VERIFIED
```

### Example 2: Budget Violation at Runtime

```bash
$ ailang run --spec design_docs/user_fetch.md --entry test examples/user_api.ail

Calling fetchUser(1)... OK
Calling fetchUser(2)... OK
Calling fetchUser(3)... OK
Calling fetchUser(4)...

SpecViolation: Budget exceeded
  Spec: api_calls <= 3
  Actual: 4 API calls attempted
  Source: design_docs/user_fetch.md:12

  Hint: Spec allows only 3 API calls, but 4 were attempted.
        Consider batching requests or increasing the budget limit.
```

---

## Success Criteria

- [ ] Spec YAML schema documented and validated
- [ ] `ailang verify --spec` command works
- [ ] Budget system tracks API calls, cost, tokens
- [ ] Effects verified against permitted/forbidden
- [ ] Contracts generated from spec predicates
- [ ] Runtime enforcement with clear errors
- [ ] Sprint planner understands specs
- [ ] 5+ example design docs with specs
- [ ] 5+ matching AILANG implementations
- [ ] All tests passing
- [ ] Documentation complete

---

## Testing Strategy

**Unit tests:**
- Spec parsing (valid and invalid YAML)
- Schema validation (missing fields, wrong types)
- Budget context (increment, check, violation)
- Effect verification (permitted, forbidden, constraints)

**Integration tests:**
- Full pipeline with spec constraints
- Verification against matching/mismatching code
- Runtime budget enforcement
- Contract generation from spec

**End-to-end tests:**
- `ailang verify --spec` command
- Sprint planner with specs
- Example workflows

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Spec schema too complex | Medium | Start minimal, extend later |
| Budget tracking overhead | Low | Opt-in, negligible cost |
| Predicate parsing errors | Medium | Use existing AILANG parser |
| Spec drift from code | Medium | CI/CD verification step |
| False verification positives | High | Conservative static analysis |

---

## Future Work (v0.9.0+)

- **NLP → Spec**: Natural language to spec YAML generation
- **Spec Templates**: Reusable spec patterns (e.g., "read-only API")
- **Distributed Budgets**: Budget tracking across services
- **Spec Inheritance**: Base specs with overrides
- **Visual Spec Editor**: UI for designing specs
- **Spec Diff**: Show changes between spec versions

---

## Related Documents

**Dependencies:**
- [m-verify-smt-verification.md](m-verify-smt-verification.md) - SMT verification (Phase 1 uses contracts)
- [m-contracts-assert.md](m-contracts-assert.md) - `@assert` syntax

**Axiom References:**
- [Design Axioms](/docs/references/axioms) - A5 (Bounded Verification), A7 (Machines First)

**Skills:**
- [sprint-planner](/.claude/skills/sprint-planner/SKILL.md) - Enhanced with spec awareness
- [sprint-executor](/.claude/skills/sprint-executor/SKILL.md) - Verify against spec during execution

---

## Philosophical Foundation

D4 embodies AILANG's core mission: **making AI code synthesis trustworthy through formal verification**.

The trust chain:
1. **User intent** → expressed in natural language
2. **Spec** → formalized in design doc YAML
3. **Contracts** → compiled into AILANG code
4. **Runtime** → enforced at execution
5. **Audit** → verifiable from spec to outcome

This creates a **closed loop** where users can trace from their requirements to the actual behavior. The AI cannot exceed its authority because the authority is formally bounded by the spec.

---

## Design Decisions (Reviewer Questions)

### Q1: Where should tokens/cost_usd be metered?

**Decision:** Inside an AI effect handler (when one exists), NOT in planner/executor layer.

**Rationale:**
- AILANG's philosophy: "authority lives at the boundary"
- If we add an `AI` effect (e.g., `_ai_complete`, `_ai_embed`), that handler owns token counting
- The planner/executor layer operates at a different abstraction (task orchestration, not effect execution)
- For v0.8.0 MVP, we defer `tokens/cost_usd` entirely until we have an `AI` effect boundary

**Future work (v0.9.0):** When `AI` effect is implemented, add `BudgetContext.ConsumeTokens(input, output int)`.

### Q2: Should `--spec` be required at runtime or only for verification/CI?

**Decision:** Optional at runtime, encouraged for CI.

| Usage | Behavior |
|-------|----------|
| `ailang verify --spec DOC CODE` | Static verification, CI/CD integration |
| `ailang run --spec DOC --entry main CODE` | Runtime enforcement (optional) |
| `ailang run --entry main CODE` | No spec constraints (existing behavior) |

**Rationale:**
- Don't break existing workflows
- Spec enforcement is opt-in, not forced
- CI pipelines can require `--spec` via policy

### Q3: Can code further restrict itself beyond the spec?

**Decision:** Yes, code can use fewer effects/tighter contracts. Verifier reports this informatively.

**Example:**
```yaml
# Spec allows:
grants:
  effects:
    permitted: [Net, IO]
```

```ailang
-- Code only uses Net (stricter than spec)
export func fetchData() -> string ! {Net}
```

**Verifier output:**
```
Grants (Effects):
  ✅ PROVED: Net permitted, used
  ℹ️ INFO: IO permitted by spec, not used in code (stricter)
```

**Rationale:**
- Restrictive code is never a violation
- Users should know when code is stricter than required
- Makes it clear the spec is a bound, not a prescription

---

## Obligation Primitives v0 (Design Review Feedback)

**Source:** External design review, December 2025

A minimal, high-leverage set of obligation primitives (10) that map cleanly from "GitHub issue intent" → "compiler-enforceable checks" → "build manifest evidence". These are AILANG-native: effects, properties, modules, tests, traces, codegen.

### Primitive 1: EffectEnvelope

**What it enforces:** Exported API stays within an allowed effect set.

**Why it matters:** Stops "AI quietly calls Net/FS/AI" when the issue didn't allow it.

- **Example intent:** "This feature must be pure / offline / no network"
- **Check:** For all exported funcs in scope, effects ⊆ allowed

**Status:** ✅ Covered by `grants.effects.permitted` in current schema

---

### Primitive 2: NoEffect

**What it enforces:** A scope is strictly pure (no effects).

**Why it matters:** Gives you a hard "pure core" boundary for neurosymbolic separation.

- **Example intent:** "Computation must be deterministic and side-effect free"
- **Check:** Effect row is empty on all symbols in scope

**Status:** 🔜 Add `grants.effects.pure_scope: true` as shorthand for empty permitted + all forbidden

---

### Primitive 3: RequiresProperty

**What it enforces:** Certain functions/modules must carry a compiler-recognized property (your contracts layer).

**Why it matters:** This is the bridge to "design doc → contract".

- **Example intent:** "Must validate inputs", "must enforce policy", "must be verified"
- **Check:** Property X attached to symbols; property rules satisfied

**Status:** ⚠️ NEW - Add `obligations.requires_properties` list

```yaml
obligations:
  requires_properties:
    - name: "InputValidation"
      applies_to: ["createUser", "updateUser"]
    - name: "PolicyEnforcement"
      applies_to: ["accessResource"]
```

---

### Primitive 4: ForbidsProperty

**What it enforces:** Disallow properties that imply unsafe behavior.

**Why it matters:** Lets you ban classes of implementation tactics (e.g., "unsafe", "unchecked", "raw").

- **Example intent:** "No raw SQL", "no bypass policy"
- **Check:** Absence of property RawSQL / property Unsafe in scope

**Status:** ⚠️ NEW - Add `obligations.forbidden_properties` list

```yaml
obligations:
  forbidden_properties:
    - "RawSQL"
    - "Unsafe"
    - "BypassPolicy"
```

---

### Primitive 5: RequiresHandler

**What it enforces:** If a module uses effect E, it must route through an approved handler/boundary module.

**Why it matters:** Prevents ad hoc IO; centralizes enforcement/logging/tracing.

- **Example intent:** "All DB writes must go through audited layer"
- **Check:** Effect operations originate only from std/db (or your boundary module)

**Status:** ⚠️ NEW - Add `grants.effect_handlers` constraint

```yaml
grants:
  effect_handlers:
    FS:
      required_handler: "std/fs/audited"
      allowed_operations: ["read"]
    DB:
      required_handler: "modules/db_boundary"
```

---

### Primitive 6: Budget

**What it enforces:** Quantitative limits on effectful operations (compile-time where possible, otherwise runtime-instrumented with trace-backed failure).

**Why it matters:** Makes cost/safety constraints real.

- **Example intent:** "No more than 3 external calls", "no large file reads"
- **Check:**
  - Static: bounded by structure where possible
  - Otherwise: inject runtime counters + fail with typed violation event

**Status:** ✅ Covered by `envelope` in current schema (api_calls, execution_ms)

**Open question from reviewer:**
> Do you want Budgets to be hard runtime failures or "trace + warn" initially?

**Answer:** Configurable via `policy.budgets`:
- `strict` = hard runtime failure
- `warn` = trace + warn, continue execution
- `runtime` = enforce at runtime but log before failing

---

### Primitive 7: RequiresAcceptanceTests

**What it enforces:** A feature cannot be "implemented" unless tests tagged to that feature exist.

**Why it matters:** Prevents "done" without executable intent.

- **Example intent:** "Edge case X must be covered"
- **Check:** Tests exist with `@acceptance(M-XYZ)` (or similar tag)

**Status:** ⚠️ NEW - Add `obligations.requires_acceptance_tests`

```yaml
obligations:
  requires_acceptance_tests:
    - feature_id: "M-XYZ"
      min_tests: 1
      test_pattern: "test_*_m_xyz_*"
```

**Verifier output:**
```
Obligations (AcceptanceTests):
  ✅ PROVED: M-XYZ has 3 tests matching pattern
  ⚠️ UNKNOWN: M-ABC has no tests (required: 1)
```

---

### Primitive 8: TraceShape

**What it enforces:** Certain trace frames/events must exist for a feature.

**Why it matters:** Ensures AI/humans can audit behavior without reading code.

- **Example intent:** "Contract failures must be traceable", "emit dedupe stats"
- **Check:** Compiler inserts/validates trace frame hooks on boundaries; runtime verifies schema

**Status:** ⚠️ NEW - Add `obligations.trace_shape`

```yaml
obligations:
  trace_shape:
    - event: "contract_violation"
      fields: ["predicate", "actual_value", "source_location"]
    - event: "api_call"
      fields: ["endpoint", "response_status", "latency_ms"]
    - event: "dedupe_stats"
      fields: ["total_items", "unique_items", "duplicates_removed"]
```

---

### Primitive 9: ArtifactMarker

**What it enforces:** Codegen output includes a stable marker linking build artifacts to FeatureSpec IDs.

**Why it matters:** You can prove "this binary implements M-XYZ" in CI without guessing.

- **Example intent:** "This feature is auditable in builds"
- **Check:** Manifest + generated Go includes `@ailang_feature(M-XYZ, hash=...)`

**Status:** ⚠️ NEW - Add to codegen + manifest output

**Generated Go:**
```go
// @ailang_feature(M-XYZ, spec_hash=a1b2c3d4, code_hash=e5f6g7h8)
func fetchUser_impl(ctx *effects.EffContext, userId interface{}) interface{} {
    // ...
}
```

**Manifest (build-manifest.json):**
```json
{
  "features": [
    {
      "id": "M-XYZ",
      "spec_path": "design_docs/planned/v0_8_0/user_fetch.md",
      "spec_hash": "a1b2c3d4",
      "code_hash": "e5f6g7h8",
      "implemented_functions": ["fetchUser", "fetchUserBatch"]
    }
  ]
}
```

---

### Primitive 10: Compat

**What it enforces:** Versioned interface compatibility constraints (types/effects/contracts) for a module boundary.

**Why it matters:** Enables component reuse "by contract" instead of by code.

- **Example intent:** "Do not break consumers", "maintain stable API"
- **Check:** Exported signatures/effects/properties match a declared interface version

**Status:** ⚠️ NEW - Add `obligations.compat`

```yaml
obligations:
  compat:
    interface_version: "1.0"
    exported_types:
      - name: "UserResult"
        signature: "{ status: Status, data: string }"
    exported_functions:
      - name: "fetchUser"
        signature: "(userId: int) -> UserResult ! {Net}"
```

**Verifier output:**
```
Compat (v1.0):
  ✅ PROVED: UserResult signature matches
  ✅ PROVED: fetchUser signature matches
  ⚠️ BREAKING: fetchUserBatch added new effect {IO} not in v1.0
```

---

### Mapping to Pipeline

**Issue → FeatureSpec (AI interviewer extracts):**
- Allowed effects → EffectEnvelope, NoEffect
- Safety constraints → RequiresHandler, ForbidsProperty, Budget
- Correctness evidence → RequiresAcceptanceTests, RequiresProperty
- Auditability → TraceShape, ArtifactMarker
- Reuse guarantees → Compat

**Compiler → Manifest (CI evidence):**
- Which primitives passed
- Which scope they applied to
- Hashes of FeatureSpec + code artifacts

---

### Recommended Starting Set (MVP)

If implementing only 7 primitives first:

| Priority | Primitive | Rationale |
|----------|-----------|-----------|
| 1 | EffectEnvelope | ✅ Already in schema |
| 2 | RequiresHandler | Safety: centralized effect routing |
| 3 | RequiresProperty | Bridge from design doc to contracts |
| 4 | ForbidsProperty | Ban unsafe implementation tactics |
| 5 | RequiresAcceptanceTests | Prevent "done" without tests |
| 6 | TraceShape | Auditability without code reading |
| 7 | ArtifactMarker | CI provenance: "binary implements M-XYZ" |

This gives safety + auditability + "intent → enforcement" with minimal complexity.

---

### Open Questions (From Reviewer)

**Q1: Do you want Budgets to be hard runtime failures or "trace + warn" initially?**

**A:** Configurable via `policy.budgets` (see Primitive 6 above). Default to `runtime` (enforce + log) for v0.8.0.

**Q2: Do you already have a property mechanism suitable for RequiresProperty, or should FeatureSpec obligations target effects/tests first?**

**A:** We have partial property support via contracts (`requires/ensures`). For v0.8.0:
- Target effects/tests first (EffectEnvelope + RequiresAcceptanceTests)
- Add RequiresProperty/ForbidsProperty as structured contract variants in v0.8.1

---

**Document created**: 2025-12-21
**Last updated**: 2025-12-23
