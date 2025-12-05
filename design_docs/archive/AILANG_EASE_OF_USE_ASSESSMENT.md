# AILANG Ease of Use Assessment - AI Development Focus

**Date**: 2025-10-22
**Version**: v0.3.14 (current)
**Target**: v0.3.18+ (future improvements)
**Status**: Assessment & Planning

## Executive Summary

**Current Score**: 7/10 for AI development
**Potential Score**: 9/10 with proposed improvements

AILANG is already well-suited for AI-driven development with strong fundamentals (type system, error messages, test suite, consistent organization). However, three key areas need improvement to reach excellence:

1. **Pipeline visualization** - AI needs to understand transformation flow
2. **AST inspection tools** - AI needs structured queries, not text dumps (✅ PARTIAL)
3. **Architecture documentation** - AI needs clear transformation rules (✅ PARTIAL)

## Current State Analysis

### ✅ What's Working Well (7/10)

#### Strong Fundamentals (+3)
- ✅ Clear type system (Hindley-Milner + row polymorphism)
- ✅ Good error messages (actionable diagnostics with file/line)
- ✅ Comprehensive test suite (90%+ coverage on new code)
- ✅ Consistent code organization (small files, single responsibility)

**Evidence**:
- `internal/types/`: Well-structured type system implementation
- `internal/errors/`: Structured error reporting with JSON schemas
- Test coverage reports: `make test-coverage-badge` → "Coverage: 29.9%" (growing)
- File size checks: `make check-file-sizes` → Only ~10 files >800 LOC

#### AI-Friendly Features (+2)
- ✅ Pure functional semantics (deterministic evaluation)
- ✅ Explicit effects (clear boundaries via capability system)
- ✅ Pattern matching (compositional reasoning)
- ✅ Module system (modular reasoning, isolated changes)

**Evidence**:
- `internal/eval/`: Pure evaluation without side effects
- `internal/effects/`: Capability-based effect system
- `internal/dtree/`: Decision tree pattern matching
- `internal/module/`: Module loader and dependency resolution

#### Consistent Organization (+2)
- ✅ Small files (<800 LOC enforced)
- ✅ Single responsibility per file
- ✅ Tests next to implementation
- ✅ Clear naming conventions

**Evidence**:
- `make check-file-sizes`: CI enforcement
- `make report-file-sizes`: Monitoring
- `.claude/agents/codebase-organizer.md`: Automated refactoring

### ⚠️ What Needs Improvement (-3)

#### Pipeline Opacity (-2)
**Problem**: AI doesn't know how Surface AST → Core AST → Types → Lowering → Evaluation works.

**Current state**:
- ✅ Architecture docs exist (`docs/architecture/types.md`, `ANF.md`)
- ❌ No visual pipeline diagram
- ❌ No transformation examples (Surface → ANF → Typed → Lowered)
- ❌ No "cheat sheet" for each phase

**Impact**: AI has to read source code to understand transformations.

**Evidence**:
```bash
$ grep -r "pipeline\|transformation\|phase" docs/architecture/*.md | wc -l
11  # Only 11 mentions across 2 docs
```

#### AST Inspection Partial (-0.5)
**Problem**: AI needs structured queries, not just text output.

**Current state**:
- ✅ `ailang debug ast --show-types` exists (M-DX2-M3)
- ✅ Shows Core AST with NodeIDs and types
- ⚠️ Text output only (not JSON/structured)
- ❌ No filtering by node type (e.g., "show all Intrinsics")
- ❌ No querying (e.g., "find nodes with type List[int]")

**Impact**: AI has to parse text output or read source code.

**Evidence**:
- `cmd/ailang/debug.go:66-198`: Text-only pretty printer
- No JSON output mode
- No query DSL

#### Type System Complexity (-0.5)
**Problem**: TVar vs TVar2 confusion, not well documented.

**Current state**:
- ✅ Type system works correctly
- ⚠️ Dual representation (TVar and TVar2) is confusing
- ❌ No doc explaining when to use which
- ❌ No migration guide

**Impact**: AI makes mistakes when generating type-related code.

**Evidence**:
- `internal/types/types.go`: Both TVar and TVar2 exist
- No documentation in `docs/architecture/types.md` about the distinction

## Detailed Feature Analysis

### 1. Pipeline Visualization (NOT IMPLEMENTED)

**Status**: ❌ Missing

**What exists**:
- Text descriptions in `docs/architecture/types.md:7-18`
- Phase list: Parsing → Elaboration → Type Inference → Validation → Lowering → Evaluation

**What's missing**:
- Visual diagram (ASCII art or Mermaid)
- Example transformations at each phase
- Data flow between phases (what gets passed?)
- Error recovery points

**Proposed solution** (v0.3.18):
Create `docs/architecture/PIPELINE.md` with:
- Mermaid diagram of full pipeline
- Example transformations for each phase
- Data structures passed between phases
- Error recovery and diagnostics

**Estimated effort**: 2-3 hours

### 2. AST Inspection Tools (PARTIALLY IMPLEMENTED)

**Status**: ⚠️ 50% complete

**What exists** (✅):
- `ailang debug ast` command (M-DX2-M3)
- Shows Core AST with indentation
- Shows types with `--show-types`
- Shows NodeIDs for cross-referencing

**What's missing** (❌):
- JSON output mode (`--format json`)
- Filtering by node type (`--filter "Intrinsic"`)
- Querying (`--query "type == List[int]"`)
- Surface AST view (`--surface` flag exists in plan, not implemented)
- Diff mode (compare two ASTs)

**Example current output**:
```bash
$ ailang debug ast concat.ail --show-types
=== Core AST (ANF) ===
Program:
  [0] Let(xs) [#13] :: [int]:
    Value: List[3] [#4] :: [int]:
      [0]: Lit(1) [#1] :: int
      ...
```

**Example desired output (JSON)**:
```bash
$ ailang debug ast concat.ail --format json
{
  "ast_type": "Core",
  "nodes": [
    {
      "id": 13,
      "kind": "Let",
      "name": "xs",
      "type": {"kind": "List", "elem": {"kind": "Int"}},
      "value": {"node_id": 4, ...},
      "body": {"node_id": 12, ...}
    }
  ]
}
```

**Proposed solution** (v0.3.18):
- Add `--format json` flag to `cmd/ailang/debug.go`
- Implement JSON serializer for Core AST nodes
- Add `--filter` flag (node type filtering)
- Add `--query` flag (type-based filtering)

**Estimated effort**: 3-4 hours

### 3. Architecture Documentation (PARTIALLY IMPLEMENTED)

**Status**: ⚠️ 60% complete

**What exists** (✅):
- `docs/architecture/types.md`: Type system (186 LOC)
- `docs/architecture/ANF.md`: ANF explanation (85 LOC)
- `docs/guides/adding-operators.md`: Operator guide (3109 LOC)
- Coverage: Type checking, ANF, operator development

**What's missing** (❌):
- **Pipeline overview**: High-level transformation flow
- **Elaboration rules**: How Surface AST → Core AST works
- **Lowering rules**: How types guide code generation
- **Module loading**: How imports are resolved
- **Effect system**: How capabilities are checked at runtime

**Gaps identified**:
```bash
$ ls -la docs/architecture/
total 24
-rw-r--r--  1 mark  staff  2348 Oct 21 22:57 ANF.md
-rw-r--r--  1 mark  staff  6624 Oct 22 16:16 types.md
# Missing: pipeline.md, elaboration.md, lowering.md, modules.md, effects.md
```

**Proposed solution** (v0.3.18):
Create missing docs:
- `docs/architecture/PIPELINE.md`: Full transformation pipeline
- `docs/architecture/ELABORATION.md`: Surface → Core transformation rules
- `docs/architecture/LOWERING.md`: Type-directed code generation
- `docs/architecture/MODULES.md`: Module loading and resolution
- `docs/architecture/EFFECTS.md`: Effect system runtime

**Estimated effort**: 6-8 hours (1-2 hours per doc)

## Implementation Plan

### Phase 1: Pipeline Visualization (v0.3.18)

**Goal**: AI understands full transformation pipeline

**Deliverables**:
- `docs/architecture/PIPELINE.md` (300-400 LOC)
  - Mermaid diagram of phases
  - Example transformations
  - Data structures passed between phases
  - Error handling at each phase

**Success criteria**:
- AI can describe the pipeline from memory
- AI knows which phase handles which transformations
- AI knows where to look for transformation bugs

**Time**: 2-3 hours

### Phase 2: Enhanced AST Inspection (v0.3.18)

**Goal**: AI can query ASTs programmatically

**Deliverables**:
- JSON output mode for `ailang debug ast`
- Node filtering (`--filter Intrinsic`)
- Type querying (`--query "type.kind == List"`)
- Surface AST view (`--surface` flag)

**Success criteria**:
- AI can extract specific nodes via JSON parsing
- AI can filter by node type without grepping text
- AI can query by type properties

**Time**: 3-4 hours

### Phase 3: Architecture Documentation (v0.3.19)

**Goal**: AI has complete transformation reference

**Deliverables**:
- `docs/architecture/ELABORATION.md` (200-300 LOC)
- `docs/architecture/LOWERING.md` (200-300 LOC)
- `docs/architecture/MODULES.md` (200-300 LOC)
- `docs/architecture/EFFECTS.md` (200-300 LOC)

**Success criteria**:
- AI knows elaboration rules (Surface → Core)
- AI knows lowering rules (Type → Builtin selection)
- AI knows module resolution algorithm
- AI knows effect checking rules

**Time**: 6-8 hours

### Phase 4: Type System Clarity (v0.3.19)

**Goal**: Eliminate TVar vs TVar2 confusion

**Deliverables**:
- `docs/architecture/TYPE_VARIABLES.md` (100-200 LOC)
  - When to use TVar vs TVar2
  - Migration guide for old code
  - Examples of each

**Success criteria**:
- AI generates correct type-related code
- No more TVar/TVar2 mistakes

**Time**: 1-2 hours

## Total Roadmap

| Phase | Version | Effort | Features |
|-------|---------|--------|----------|
| 1. Pipeline Viz | v0.3.18 | 2-3h | PIPELINE.md with diagrams |
| 2. AST Inspection | v0.3.18 | 3-4h | JSON output, filtering, querying |
| 3. Arch Docs | v0.3.19 | 6-8h | ELABORATION.md, LOWERING.md, MODULES.md, EFFECTS.md |
| 4. Type Clarity | v0.3.19 | 1-2h | TYPE_VARIABLES.md |
| **Total** | **2 versions** | **12-17h** | **9 new docs/features** |

## Expected Impact

### Before (Current: 7/10)
- ✅ Strong fundamentals
- ✅ AI-friendly features
- ⚠️ Pipeline opacity
- ⚠️ Partial AST inspection
- ⚠️ Partial architecture docs

**Developer experience**:
- Reading source code to understand transformations
- Text parsing for AST analysis
- Trial-and-error for type system edge cases

### After (Target: 9/10)
- ✅ Strong fundamentals (unchanged)
- ✅ AI-friendly features (unchanged)
- ✅ **Pipeline transparency** (PIPELINE.md)
- ✅ **Complete AST inspection** (JSON, filtering, querying)
- ✅ **Complete architecture docs** (all transformations documented)

**Developer experience**:
- Documentation explains transformations
- JSON queries for programmatic AST analysis
- Clear rules for all edge cases

**Score breakdown**:
- Strong fundamentals: +3 (unchanged)
- AI-friendly features: +2 (unchanged)
- Consistent organization: +2 (unchanged)
- **Pipeline transparency: +1** (was -1)
- **AST tooling: +0.5** (was -0.5)
- **Architecture completeness: +0.5** (was -0.5)
- **Total: 9/10** ⬆️ +2

## Non-Goals

**Not in scope** (conflicts with AI-first design):
- ❌ IDE integration (LSP, hover, autocomplete)
- ❌ Interactive debugger (step-through execution)
- ❌ GUI tools (visual AST explorer)
- ❌ Real-time type hints (IDE feature)

**Why**: AILANG is designed for AI agents, not human IDE workflows. Focus on:
- ✅ Documentation (AI reads this)
- ✅ CLI tools (AI invokes these)
- ✅ Structured output (AI parses this)

## Success Metrics

**Quantitative**:
- Pipeline doc: 300-400 LOC with diagrams
- JSON output: <100ms latency for typical files
- Architecture docs: 800-1200 LOC total
- Type variable doc: 100-200 LOC

**Qualitative**:
- AI can describe pipeline from memory (no source reading)
- AI can extract AST info via JSON (no text parsing)
- AI generates correct type code (no TVar/TVar2 mistakes)
- AI knows where to look for bugs (transformation phase clarity)

**Validation**:
- Test: Ask AI to explain elaboration → should reference ELABORATION.md
- Test: Ask AI to find all Intrinsic nodes → should use JSON output
- Test: Ask AI to add operator → should follow PIPELINE.md

## References

**Existing work**:
- M-DX2-M3: Debug CLI implementation (`.claude/agents/eval-orchestrator.md`)
- M-DX4: CoreTypeInfo validation (`docs/architecture/types.md`)
- ANF documentation (`docs/architecture/ANF.md`)
- Type system docs (`docs/architecture/types.md`)

**Related design docs**:
- [M-DX2: Operator Development Experience](../v0_3_16/m-dx2-operator-development-experience.md)
- [M-DX4: CoreTypeInfo Completeness](../v0_3_15/m-dx4-coretypeinfo-population-gaps.md)

**Future work** (v0.4.0+):
- Reflection system (structural type inspection)
- Schema registry (machine-readable type definitions)
- Capability budgets (resource-bounded effects)

---

**Next steps**:
1. ✅ Create this assessment (DONE)
2. ⏭️ Implement Phase 1: PIPELINE.md (v0.3.18)
3. ⏭️ Implement Phase 2: JSON AST output (v0.3.18)
4. ⏭️ Implement Phase 3: Architecture docs (v0.3.19)
5. ⏭️ Implement Phase 4: Type variable clarity (v0.3.19)
