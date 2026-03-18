# M-CONTRACT-PIPELINE-DX: Contract Pipeline Coverage & Developer Experience

**Status**: Planned
**Target**: v0.9.2
**Priority**: P2 (Medium — prevents future regressions, improves debug speed)
**Estimated**: 4-6 hours total (3 quick wins + 2 larger improvements)
**Dependencies**: M-CONTRACT-OPLOWERING-FIX (just shipped)
**Milestone ID**: M-CONTRACT-PIPELINE-DX
**Created**: 2026-03-18
**Source**: Retrospective on M-CONTRACT-OPLOWERING-FIX — a "25 LOC, 2-4 hour" fix became a multi-layered investigation because 5+ pipeline passes independently skip contract expressions with no test catching it.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime semantics change |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Pipeline integration test catches contract regressions locally |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Telemetry and assertions help agents debug pipeline issues faster |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Deferred-to-shim telemetry makes hidden costs visible |
| A10: Composability | 0 | No composability changes |
| A11: Structured Failure | +1 | Better error messages when pipeline passes skip contracts |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No runtime changes
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Directly improves machine analysis

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **exact anti-pattern** described in CLAUDE.md's "Systemic Fixes" principle. The contract pipeline has been patched incrementally:

1. **v0.7.1** (M-VERIFY-CONTRACTS): Added contract parsing — contracts stored in `prog.Meta`
2. **v0.8.0** (M-CONTRACTS-OPLOWERING): Added lowerer traversal of `Meta` — but forgot type checker and dict elaboration
3. **v0.9.2** (M-CONTRACT-OPLOWERING-FIX): Added type checker and dict elaboration traversal of `Meta`
4. **Next bug?**: The next pipeline pass added will silently skip contracts again

**Root cause**: `prog.Meta` is an opt-in traversal. Every new pipeline pass must independently remember to process it. There's no assertion, test, or abstraction preventing the omission.

**The "growing switch statement" warning sign** from CLAUDE.md applies: every pipeline pass has the same copy-pasted Meta traversal pattern.

---

## Problem Statement

### Debugging Cost

The M-CONTRACT-OPLOWERING-FIX sprint revealed 6 specific DX gaps:

| Gap | Time Wasted | Description |
|-----|-------------|-------------|
| No pipeline integration test for contracts | ~30 min | Had to manually test each contract example to find which passed/failed |
| `prog.Meta` traversal copy-pasted in 4 places | ~20 min | Each pipeline pass independently decided to skip contracts |
| Lowering telemetry hides "Deferred-to-shim" | ~40 min | Showed "0 misses" when ops were actually deferred — misleading |
| No `--debug-contracts` flag | ~30 min | Added/removed `fmt.Printf` debug prints 5+ times |
| Design doc marked "Implemented" but incomplete | ~15 min | v0.8.0 doc didn't mention type checker or dict elaboration gaps |
| ~~TFunc vs TFunc2 no adapter~~ | ~~15 min~~ | ~~Already being tackled in separate sprint~~ |

**Total estimated waste: ~2.5 hours** on a fix that should have taken 30 minutes with proper tooling.

---

## Goals

**Primary Goal:** Prevent future pipeline passes from silently skipping contract expressions, and make contract pipeline issues debuggable in < 10 minutes.

**Success Metrics:**
- Pipeline integration test catches any new pass that forgets contracts
- `--debug-compile` shows deferred-to-shim count in telemetry summary
- `prog.AllExpressions()` eliminates copy-pasted Meta traversal
- Design doc checklist prevents incomplete "Implemented" status

---

## Solution Design

### Phase 1: Quick Wins (< 30 min each)

#### QW1: Add "Deferred-to-shim" to lowering telemetry summary (~5 min)

**File:** `internal/pipeline/pipeline_telemetry.go`

The `reportLoweringTelemetry` function counts CoreTI-hit, CoreTI-miss, ResolvedConstraints, and Default — but NOT "Deferred-to-shim" or "OperandType". These are tracked by `trackFallback` but invisible in the summary.

```go
// Add to the counting loop:
case "Deferred-to-shim":
    deferred++
case "OperandType":
    operandType++

// Add to the output:
fmt.Fprintf(os.Stderr, "[DEBUG]   Deferred to shim: %d (%.1f%%)\n", ...)
fmt.Fprintf(os.Stderr, "[DEBUG]   OperandType fallback: %d (%.1f%%)\n", ...)

// WARN if deferred > 0 and shim not enabled:
if deferred > 0 {
    fmt.Fprintf(os.Stderr, "[WARN] %d operators deferred to shim — these will fail at runtime without --experimental-binop-shim\n", deferred)
}
```

#### QW2: Pipeline integration test for contract lowering (~20 min)

**File:** `internal/pipeline/contract_pipeline_test.go` (new)

Test that compiles a module with `ensures { result >= 0 }` and asserts:
- No `*core.Intrinsic` nodes remain in the lowered contract expressions
- No `*core.BinOp` nodes remain in the lowered contract expressions
- CoreTI has entries for all operator nodes in contracts

```go
func TestContractExpressionsFullyLowered(t *testing.T) {
    src := `module test/contract_pipeline
export func abs(x: int) -> int
ensures { result >= 0 }
= if x >= 0 then x else 0 - x`

    result, err := pipeline.RunWithContext(ctx, cfg, src)
    require.NoError(t, err)

    // Walk contract expressions and assert no Intrinsic/BinOp remain
    for _, meta := range result.Modules["test/contract_pipeline"].Core.Meta {
        for _, contract := range meta.Contracts {
            assertNoIntrinsicOrBinOp(t, contract.Expr)
        }
    }
}
```

#### QW3: Design doc pipeline pass checklist (template update, ~5 min)

**File:** `.claude/skills/design-doc-creator/resources/design_doc_structure.md`

Add to the template under "Implementation Plan":

```markdown
### Pipeline Pass Checklist

If this feature adds or modifies a compiler pipeline pass, verify it handles:

- [ ] `prog.Decls` — top-level declarations
- [ ] `prog.Meta` — contract expressions (requires/ensures)
- [ ] `prog.Flags` — compilation flags (if applicable)
```

### Phase 2: Larger Improvements (future sprint)

#### P2.1: `prog.AllExpressions()` unified iterator (~2-3 hours)

**Files:** `internal/core/core.go`, all pipeline passes

Add to `core.Program`:

```go
// ExprVisitor visits all expressions in a program, including contract expressions.
type ExprVisitor func(expr CoreExpr, context string) CoreExpr

// TransformAll applies a transformation to all expressions: decls AND contracts.
func (p *Program) TransformAll(fn ExprVisitor) *Program {
    newDecls := make([]CoreExpr, len(p.Decls))
    for i, decl := range p.Decls {
        newDecls[i] = fn(decl, "decl")
    }

    var newMeta map[string]*DeclMeta
    if p.Meta != nil {
        newMeta = make(map[string]*DeclMeta, len(p.Meta))
        for name, meta := range p.Meta {
            if len(meta.Contracts) == 0 {
                newMeta[name] = meta
                continue
            }
            newMetaCopy := *meta
            newMetaCopy.Contracts = make([]*Contract, len(meta.Contracts))
            for i, contract := range meta.Contracts {
                newContract := *contract
                if contract.Expr != nil {
                    newContract.Expr = fn(contract.Expr, "contract:"+name)
                }
                newMetaCopy.Contracts[i] = &newContract
            }
            newMeta[name] = &newMetaCopy
        }
    }

    return &Program{Decls: newDecls, Meta: newMeta, Flags: p.Flags}
}
```

Then each pipeline pass simplifies to:
```go
// Before (copy-pasted in 4 places):
for _, decl := range prog.Decls { transform(decl) }
if prog.Meta != nil { for name, meta := range prog.Meta { ... } }

// After (one-liner):
return prog.TransformAll(func(expr core.CoreExpr, ctx string) core.CoreExpr {
    return transform(expr)
})
```

#### P2.2: `--debug-contracts` flag (~1 hour)

**Files:** `cmd/ailang/main_run.go`, `internal/eval/eval_evaluator.go`

When enabled, prints:
- Contract expressions after each pipeline pass (type, node count)
- Contract expression types at `attachContracts` time
- Contract evaluation results (pass/fail with values)

---

## Implementation Plan

**Phase 1: Quick Wins** (~30 min total)
- [ ] QW1: Add Deferred-to-shim to telemetry summary
- [ ] QW2: Pipeline integration test for contract lowering
- [ ] QW3: Design doc pipeline pass checklist (template update)

**Phase 2: Larger Improvements** (future sprint, ~3-4 hours)
- [ ] P2.1: `prog.TransformAll()` unified iterator
- [ ] P2.2: `--debug-contracts` flag

### Files to Modify/Create

**Phase 1:**
- `internal/pipeline/pipeline_telemetry.go` (~+10 LOC) — Add deferred-to-shim counting
- `internal/pipeline/contract_pipeline_test.go` (~60 LOC, new) — Integration test
- `.claude/skills/design-doc-creator/resources/design_doc_structure.md` (~+10 LOC) — Checklist

**Phase 2:**
- `internal/core/core.go` (~+40 LOC) — `TransformAll` method
- `internal/elaborate/dictionaries.go` (~-20/+5 LOC) — Use TransformAll
- `internal/pipeline/op_lowering.go` (~-20/+5 LOC) — Use TransformAll
- `internal/pipeline/specialize.go` (~-5/+5 LOC) — Use TransformAll
- `cmd/ailang/main_run.go` (~+5 LOC) — `--debug-contracts` flag
- `internal/eval/eval_evaluator.go` (~+20 LOC) — Contract debug output

---

## Success Criteria

- [ ] `reportLoweringTelemetry` shows Deferred-to-shim count with warning
- [ ] `TestContractExpressionsFullyLowered` passes and catches regressions
- [ ] Design doc template includes pipeline pass checklist
- [ ] All existing tests pass
- [ ] `make lint` clean

## Testing Strategy

**Unit tests:**
- QW1: Check telemetry output includes "Deferred to shim" line
- QW2: Test module with int, float, and string comparisons in contracts

**Integration:**
- QW2: Compile module → walk contract AST → assert no Intrinsic/BinOp nodes

## Non-Goals

- **TFunc/TFunc2 unification** — Being tackled in M-TYPE-V2-MIGRATION sprint
- **Contract type error reporting** — Separate feature (contracts are best-effort for CoreTI)
- **SMT solver integration** — Separate pipeline, not affected by this change

## Related Documents

- [M-CONTRACT-OPLOWERING-FIX](m-contract-oplowering-fix.md) — The fix that exposed these DX gaps
- [M-CONTRACTS-OPLOWERING](../../implemented/v0_8_0/m-contracts-oplowering.md) — v0.8.0 partial fix
- [M-VERIFY-CONTRACTS](../../implemented/v0_7_1/m-verify-contracts.md) — Original contract system

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-18
