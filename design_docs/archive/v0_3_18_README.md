# v0.3.18 Planned Features - AI Development Experience Improvements

**Theme**: Making AILANG Transparent for AI Development
**Target**: Improve from 7/10 to 9/10 for AI ease of use
**Total Effort**: 5-7 hours
**Priority**: P1 (High)

## Overview

This release focuses on making AILANG's compilation pipeline and AST structure visible and queryable for AI developers. The goal is to eliminate source code reading and text parsing, replacing them with comprehensive documentation and structured tooling.

## Features

### M-DX6: Pipeline Visualization & Documentation
**Status**: Planned
**Effort**: 2-3 hours
**Doc**: [m-dx6-pipeline-visualization.md](m-dx6-pipeline-visualization.md)

**What**: Comprehensive pipeline documentation with visual diagrams, transformation examples, and debugging guide.

**Why**: AI currently doesn't understand how Surface AST → Core AST → Types → Lowering → Evaluation works without reading source code.

**Deliverables**:
- `docs/architecture/PIPELINE.md` with Mermaid diagram
- Transformation examples for each phase
- Data flow documentation
- Debugging guide (which phase to check for which bugs)

**Impact**: AI can describe the pipeline from memory, knows where to look for bugs.

### M-DX7: JSON AST Output & Structured Querying
**Status**: Planned
**Effort**: 3-4 hours
**Doc**: [m-dx7-ast-inspection-json.md](m-dx7-ast-inspection-json.md)

**What**: Extend `ailang debug ast` with JSON output, filtering, and querying.

**Why**: Current text output requires parsing (grep/awk or LLM). JSON enables programmatic queries with zero tokens.

**Deliverables**:
- `--format json` flag for structured output
- `--filter <NodeKind>` for filtering by node type
- `--query <expr>` for type-based querying
- `--surface` flag for pre-elaboration AST view

**Impact**: AI can extract AST info via JSON (no text parsing), integrate with jq/scripts.

## Rationale

From the [Ease of Use Assessment](AILANG_EASE_OF_USE_ASSESSMENT.md):

**Current state (7/10)**:
- ✅ Strong fundamentals (type system, errors, tests, organization)
- ✅ AI-friendly features (pure functions, explicit effects, pattern matching)
- ⚠️ Pipeline opacity (-2 points)
- ⚠️ Partial AST inspection (-0.5 points)

**After v0.3.18 (9/10)**:
- ✅ Pipeline transparency (+1 point) via M-DX6
- ✅ Complete AST tooling (+0.5 points) via M-DX7
- **Total improvement: +2 points**

## Implementation Order

1. **M-DX6 first** (2-3h) - Documentation enables understanding
2. **M-DX7 second** (3-4h) - Tooling enables querying

**Rationale**: Documentation provides context for using the tooling.

## Success Metrics

**Quantitative**:
- PIPELINE.md: 300-400 LOC with Mermaid diagram
- JSON output: <100ms latency for typical files
- Query filtering: Works with jq integration

**Qualitative**:
- AI can describe pipeline from memory (no source reading)
- AI can extract AST info via JSON (no text parsing)
- New contributors understand pipeline in <15 minutes

**Validation tests**:
- Ask AI: "How does AILANG compile code?" → Should reference PIPELINE.md
- Ask AI: "Find all Intrinsic nodes with List type" → Should use JSON + jq
- Ask AI: "Where do type errors come from?" → Should reference Type Checking phase

## Future Work (v0.3.19+)

**Not in v0.3.18**, deferred to future releases:

### Architecture Documentation (v0.3.19)
- `docs/architecture/ELABORATION.md` - Surface → Core transformation rules
- `docs/architecture/LOWERING.md` - Type-directed code generation
- `docs/architecture/MODULES.md` - Module loading and resolution
- `docs/architecture/EFFECTS.md` - Effect system runtime

**Estimated**: 6-8 hours

### Type System Clarity (v0.3.19)
- `docs/architecture/TYPE_VARIABLES.md` - TVar vs TVar2 explanation

**Estimated**: 1-2 hours

### Advanced Tooling (v0.4.0+)
- Streaming JSON (for large files)
- Advanced query DSL (boolean operators, regex)
- Diff mode (compare two ASTs)
- Interactive pipeline visualizer (web UI)

**Estimated**: TBD

## References

**Assessment**:
- [AILANG Ease of Use Assessment](AILANG_EASE_OF_USE_ASSESSMENT.md) - Detailed analysis

**Design docs**:
- [M-DX6: Pipeline Visualization](m-dx6-pipeline-visualization.md)
- [M-DX7: JSON AST Output](m-dx7-ast-inspection-json.md)

**Related work**:
- M-DX2-M3: Debug CLI (v0.3.16) - Text-based AST inspection
- M-DX4: CoreTypeInfo Validation (v0.3.15) - Type information completeness

**Existing docs**:
- `docs/architecture/types.md` - Type system
- `docs/architecture/ANF.md` - ANF explanation

---

**Created**: 2025-10-22
**Status**: Ready for implementation
**Next step**: Implement M-DX6 (PIPELINE.md)
