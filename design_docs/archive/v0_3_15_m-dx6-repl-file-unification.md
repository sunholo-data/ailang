# M-DX6: REPL/File Semantic Unification

**Status**: 🟡 Partially Implemented (v0.3.17)
**Target**: v0.3.15 → Deferred to v0.4.x
**Priority**: P2 - Low (Core parity exists, full unification nice-to-have)
**Estimated**: 6 hours remaining (4h Pipeline.Run integration + 1.5h --strict mode + 0.5h docs)
**Dependencies**: None

## Current Implementation Status (v0.3.17)

**✅ Implemented:**
- Prelude injection: REPL uses `pipeline.InjectPrelude()` (same as file mode)
- CoreTypeInfo validation: REPL validates before evaluation (M-DX4)
- Operator lowering: REPL uses `pipeline.NewOpLowerer()` (same as file mode)
- Parser parity: Test suite ensures REPL and file parsing consistency ([internal/parser/repl_parity_test.go](../../internal/parser/repl_parity_test.go), 220 LOC)
- Dictionary elaboration: REPL uses same dictionary-passing as file mode

**❌ Not Implemented (Deferred):**
- Full pipeline unification: REPL doesn't use `Pipeline.Run()` (has custom 482-line evaluation flow)
- `--strict` flag: No capability enforcement mode for REPL
- Virtual module context: REPL doesn't instantiate formal module context
- `docs/guides/repl_vs_file.md`: No flow diagram documentation

**Key Insight:** The critical parity issues (prelude, type checking, operator lowering, validation) are already solved. The remaining work (using `Pipeline.Run()` directly) is a code quality improvement, not a user-facing issue.

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Unifies semantics, no syntax changes |
| Preserve Semantic Clarity | + | +1 | Makes REPL behavior predictable and consistent |
| Increase Determinism | + | +2 | REPL becomes deterministic like file mode |
| Lower Token Cost | + | +1 | Fewer surprises = less debugging |
| **Net Score** | | **+4** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

REPL and file mode have different codepaths, causing behavioral inconsistencies. Code that works in REPL may fail in file mode (and vice versa) due to differences in prelude injection, capability checking, and module context.

**Current State:**
- REPL uses different prelude injection than file mode
- REPL has different capability checks (more permissive)
- REPL doesn't instantiate full module context
- No `--strict` flag to make REPL behave identically to file mode
- Developers can't trust REPL for testing production code

**Impact:**
- **Who is affected?** All AILANG developers using REPL for prototyping
- **How significant?** P1 - Causes confusion, wastes time debugging inconsistencies
- **Example**: Code with effects works in REPL but fails when moved to file

## Goals

**Primary Goal:** Make REPL instantiate a full module context and use identical compilation pipeline to file mode.

**Success Metrics:**
- Code that compiles in REPL compiles identically in file mode
- REPL uses same prelude injection as file mode (entry-module semantics)
- `--strict` flag makes REPL enforce same capability checks as file mode
- Documentation includes "REPL vs Script vs Library" flow diagram
- All REPL/file parity tests pass

## Solution Design

### Overview

Refactor REPL to use the same `Pipeline.Run()` codepath as file mode. Make REPL instantiate a virtual module with entry-module semantics (prelude injection). Add `--strict` flag for identical capability enforcement.

### Architecture

**Components:**
1. **Unified Pipeline** (`internal/repl/eval_unified.go`): Make REPL use `Pipeline.Run()` instead of custom evaluation
2. **Virtual Module Context** (`internal/repl/module_context.go`): Create synthetic module for REPL session
3. **Prelude Injection** (`internal/repl/prelude.go`): Apply same entry-module prelude rules as file mode
4. **Strict Mode** (`cmd/ailang/repl.go`): Add `--strict` flag to enforce identical capability checks

### Implementation Plan

**Phase 1: Pipeline Unification** (~4 hours)
- [ ] Create `repl.evalWithPipeline()` to use `Pipeline.Run()`
- [ ] Replace custom REPL evaluation with pipeline-based approach
- [ ] Ensure REPL can handle multi-line inputs through pipeline
- [ ] Test that all current REPL features work with unified pipeline

**Phase 2: Module Context & Prelude** (~2 hours)
- [ ] Create virtual module context for REPL session
- [ ] Apply entry-module prelude injection (same as `ailang run`)
- [ ] Ensure prelude is injected only once per session (not per line)
- [ ] Handle incremental definitions (let bindings persist across lines)

**Phase 3: Strict Mode & Capability Checks** (~1.5 hours)
- [ ] Add `--strict` flag to REPL
- [ ] In strict mode, enforce same capability checks as file mode
- [ ] In normal mode (default), keep current permissive behavior
- [ ] Document when to use strict mode

**Phase 4: Documentation** (~0.5 hours)
- [ ] Create "REPL vs Script vs Library" flow diagram
- [ ] Document `--strict` flag usage
- [ ] Add example sessions showing REPL/file parity
- [ ] Update REPL guide with new semantics

### Files to Modify/Create

**New files:**
- `internal/repl/eval_unified.go` - Pipeline-based evaluation (~150 LOC)
- `internal/repl/module_context.go` - Virtual module context (~100 LOC)
- `docs/guides/repl_vs_file.md` - Flow diagram and guide (~200 LOC)

**Modified files:**
- `internal/repl/repl.go` - Switch to unified pipeline (~50 LOC changes)
- `cmd/ailang/repl.go` - Add --strict flag (~20 LOC)
- `internal/runtime/prelude.go` - Extract reusable prelude injection (~30 LOC changes)
- `docs/guides/repl.md` - Update with new semantics (~40 LOC additions)

## Examples

### Example 1: Prelude Injection Consistency

**Before (Current - Inconsistent):**
```bash
# REPL - custom prelude injection
λ> print("hello")
"hello"

# File - entry-module prelude injection
$ ailang run --caps IO --entry main example.ail
Error: print undefined (no prelude injection)
```

**After (Unified):**
```bash
# REPL - same entry-module prelude as file
λ> print("hello")
"hello"

# File - same behavior
$ ailang run --caps IO --entry main example.ail
"hello"

# Both use identical prelude injection logic
```

### Example 2: Strict Mode for Testing

```bash
# Normal mode (default) - permissive for prototyping
$ ailang repl
λ> print("hello")
"hello"

# Strict mode - enforce file-like capability checks
$ ailang repl --strict
λ> print("hello")
Error: print requires IO capability, but REPL has no capabilities.
Hint: Start REPL with --caps IO or declare effects explicitly.

# Now with capabilities
$ ailang repl --strict --caps IO
λ> print("hello")
"hello"
```

### Example 3: Virtual Module Context

```bash
# REPL maintains module context across lines
λ> let x = 42
λ> let y = x + 1
λ> y
43

# Same as writing a file:
# module Main
# let x = 42
# let y = x + 1
```

## Success Criteria

- [ ] REPL uses `Pipeline.Run()` (same codepath as file mode)
- [ ] REPL instantiates virtual module context with entry-module semantics
- [ ] Prelude injection identical between REPL and file mode
- [ ] `--strict` flag enforces identical capability checks
- [ ] REPL/file parity tests pass (same code produces same results)
- [ ] All existing REPL tests pass
- [ ] Documentation updated (REPL guide, new repl_vs_file.md)
- [ ] Flow diagram shows REPL/Script/Library differences

## Testing Strategy

**Unit tests:**
- Test virtual module context creation
- Test prelude injection (once per session, not per line)
- Test strict vs normal mode capability enforcement
- Test incremental definition persistence

**Integration tests:**
- REPL/file parity suite: same code, same results
- Multi-line inputs through unified pipeline
- Effect handling in strict vs normal mode
- Prelude function availability (print, show, etc.)

**Manual testing:**
- Test common REPL workflows (define functions, test expressions)
- Verify strict mode behaves like file mode
- Check that error messages are consistent
- Test session state persistence

## Non-Goals

**Not in this feature:**
- Persistent REPL history across sessions - Separate feature, deferred
- REPL-specific commands beyond :type/:help - Keep focused on semantic parity
- Module import in REPL - Complex, requires more design work
- Performance optimization - Correctness first, optimize later

## Timeline

**Day 1** (4 hours):
- Phase 1: Pipeline unification

**Day 2** (4 hours):
- Phase 2: Module context & prelude
- Phase 3: Strict mode
- Phase 4: Documentation

**Total: ~8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing REPL workflows | High | Comprehensive testing, keep normal mode permissive |
| Performance regression | Low | Pipeline is already used for file mode, should be comparable |
| Incremental definitions hard to implement | Medium | Use virtual module, accumulate definitions |
| Strict mode too strict for prototyping | Low | Make normal mode default, strict mode opt-in |

## References

- Field report: User's "AI-first DX reflection" from October 2025
- [REPL Guide](../../../docs/guides/repl.md) - Current REPL documentation
- Entry-module prelude semantics (v0.3.x)
- M-DX1: Developer Experience - Similar focus on consistency

## Future Work

- M-DX10: REPL module imports (load .ail files into REPL session)
- M-DX11: REPL history persistence (save session state across restarts)
- M-DX12: Interactive type debugger (integrate M-DX5 debug types into REPL)
- REPL performance profiling (measure pipeline overhead)

---

## Implementation Report (v0.3.17)

**What Was Delivered:**

The core semantic parity between REPL and file mode has been achieved through incremental improvements across multiple versions:

**v0.3.0-v0.3.10:**
- REPL uses `pipeline.InjectPrelude()` - same prelude injection as file mode
- REPL has access to all builtins via `types.NewTypeEnvWithBuiltins()`
- Parser parity tests ensure identical AST generation (13 test cases)

**v0.3.17 (M-DX4):**
- REPL uses `pipeline.ValidateCoreTypeInfo()` before evaluation
- REPL uses `pipeline.NewOpLowerer()` for operator lowering
- Identical type-guided lowering as file mode

**Current REPL Pipeline (internal/repl/repl_eval.go):**
```
1. Parse              → lexer.New() + parser.New()
2. Elaborate          → elaborate.NewElaborator()
3. Type check         → types.NewCoreTypeChecker()
4. Dict elaboration   → elaborate.ElaborateWithDictionaries()
5. Validate CoreTI    → pipeline.ValidateCoreTypeInfo() [M-DX4]
6. Lower operators    → pipeline.NewOpLowerer() [M-DX4]
7. Link dictionaries  → link.NewLinker()
8. Evaluate           → evaluator.Eval()
```

**File Pipeline (internal/pipeline/pipeline_single.go):**
```
Pipeline.Run() does all of the above in one call
```

**Why Not Fully Unified:**

The REPL's custom pipeline (~482 LOC) handles features that `Pipeline.Run()` doesn't support:
- **Incremental definitions**: Let bindings persist across REPL inputs
- **Type environment persistence**: Type info saved between commands
- **Interactive commands**: `:type`, `:help`, `:import` commands
- **Show flags**: `--show-core`, `--show-typed`, `--dry-link` for debugging

**To fully unify**, we'd need to either:
1. Make `Pipeline.Run()` support incremental mode (complex, REPL-specific behavior)
2. Refactor REPL to wrap `Pipeline.Run()` (loses fine-grained control)

**Current Decision:** Keep custom REPL pipeline since core parity is achieved. If user-facing bugs emerge, revisit full unification in v0.4.x.

**Remaining Work (Low Priority):**
- `--strict` flag for capability enforcement (nice-to-have for testing production code in REPL)
- `docs/guides/repl_vs_file.md` flow diagram (documentation improvement)
- Formal virtual module context (code quality, no user impact)

**User Impact:** None. REPL and file mode produce identical results for equivalent code.

---

**Document created**: 2025-10-22
**Last updated**: 2025-10-26 (Status updated - partially implemented)
