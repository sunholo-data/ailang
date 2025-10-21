# M-DX5: Inference Visibility & Debug Tooling

**Status**: Planned
**Target**: v0.3.15
**Priority**: P1 - Medium (DX Improvement)
**Estimated**: 6 hours (3h CLI implementation + 2h trace instrumentation + 1h docs)
**Dependencies**: M-DX4 (CoreTypeInfo validation provides foundation)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Debugging tool, no syntax changes |
| Preserve Semantic Clarity | + | +1 | Makes type inference transparent and debuggable |
| Increase Determinism | + | +1 | Exposes inference steps, making behavior predictable |
| Lower Token Cost | + | +1 | Faster debugging = fewer AI conversation rounds |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

Type inference is a black box. When types are missing or wrong, developers have no way to see what the type checker inferred, why it failed, or where CoreTypeInfo is incomplete.

**Current State:**
- No visibility into type inference process
- Can't inspect CoreTypeInfo for specific expressions
- No way to see constraints generated during unification
- Debugging requires adding print statements to compiler source
- Error messages don't explain why inference failed

**Impact:**
- **Who is affected?** All AILANG developers (especially AI agents debugging type issues)
- **How significant?** P1 - Slows development, makes type errors opaque
- **Example**: "Type mismatch" error with no explanation of what was inferred vs expected

## Goals

**Primary Goal:** Make type inference inspectable and debuggable through CLI tooling.

**Success Metrics:**
- `ailang debug types --trace-inference` shows all inference steps
- `ailang debug types --show-gaps` lists nodes missing CoreTypeInfo
- `ailang debug types --query <expr>` shows inferred type for expression
- Colorized, compact output modes for readability
- Documentation includes example debug sessions

## Solution Design

### Overview

Add `ailang debug types` subcommand with multiple inspection modes. Instrument the type checker to collect trace data (NodeID, expression kind, inferred type, constraints). Integrate with M-DX4's CoreTypeInfo validator to show gaps.

### Architecture

**Components:**
1. **Debug Types CLI** (`cmd/ailang/debug_types.go`): New subcommand with flags for different modes
2. **Inference Tracer** (`internal/types/trace.go`): Collect trace data during type checking
3. **CoreTypeInfo Inspector** (`internal/pipeline/inspect_coretypeinfo.go`): Query and display CoreTypeInfo entries
4. **Pretty Printer** (`internal/debug/typeinfo_printer.go`): Format trace data with colors and compact mode

### Implementation Plan

**Phase 1: CLI Infrastructure** (~1.5 hours)
- [ ] Add `ailang debug types` subcommand
- [ ] Add flags: `--trace-inference`, `--show-gaps`, `--query <expr>`, `--compact`, `--no-color`
- [ ] Create output formatting framework (table, tree, colorized)
- [ ] Add help text and examples

**Phase 2: Inference Tracing** (~2 hours)
- [ ] Add `types.InferenceTrace` struct (NodeID, ExprKind, InferredType, Constraints)
- [ ] Instrument unifier to collect trace entries
- [ ] Instrument constraint solver to log constraint additions
- [ ] Store traces in `Pipeline.typeTrace` field
- [ ] Output trace as structured table or tree

**Phase 3: CoreTypeInfo Inspection** (~1.5 hours)
- [ ] Reuse M-DX4's validator to find gaps
- [ ] Add `--show-gaps` mode: list all missing CoreTypeInfo entries
- [ ] Add `--query <expr>` mode: look up type for specific expression
- [ ] Format output with NodeID, expression kind, type, location

**Phase 4: Documentation & Examples** (~1 hour)
- [ ] Add docs/architecture/types.md with literal resolution table
- [ ] Document CoreTypeInfo contract (total function, validation enforced)
- [ ] Add example debug session walkthrough
- [ ] Include screenshots/examples of each debug mode

### Files to Modify/Create

**New files:**
- `cmd/ailang/debug_types.go` - CLI subcommand (~200 LOC)
- `internal/types/trace.go` - Inference tracer (~150 LOC)
- `internal/pipeline/inspect_coretypeinfo.go` - CoreTypeInfo inspector (~100 LOC)
- `internal/debug/typeinfo_printer.go` - Pretty printer (~150 LOC)
- `docs/architecture/types.md` - Type system documentation (~300 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Add debug subcommand (~10 LOC)
- `internal/types/infer.go` - Add trace collection (~30 LOC)
- `internal/types/unify.go` - Log unification steps (~20 LOC)
- `internal/pipeline/pipeline.go` - Store type trace (~15 LOC)

## Examples

### Example 1: Trace Inference

```bash
$ ailang debug types --trace-inference example.ail

Type Inference Trace:
  NodeID 1 | Literal (42)          | Int          | []
  NodeID 2 | Literal (0.5)         | Float        | []
  NodeID 3 | BinOp (+)             | Float        | [unify(Int, Float) → Float]
  NodeID 4 | Lambda (\x -> ...)    | Int -> Float | [param: Int, body: Float]
  NodeID 5 | Application           | Float        | [fn: Int -> Float, arg: Int]

✓ All nodes have type information
```

### Example 2: Show Gaps

```bash
$ ailang debug types --show-gaps example.ail

CoreTypeInfo Gaps:
  ✗ NodeID 1337 | Float literal (0.5)         | line 4:10
  ✗ NodeID 1338 | Comparison (<=)             | line 5:15
  ✗ NodeID 1339 | Nested let binding          | line 6:5

Found 3 nodes without type information.
This indicates a compiler bug - all Core nodes should be typed.

Hint: Run validation with `make test` or report this issue.
```

### Example 3: Query Type

```bash
$ ailang debug types --query "(\x -> x + 0.5)" example.ail

Expression: (\x -> x + 0.5)
  NodeID: 4
  Inferred Type: Float -> Float
  Location: line 1:1-17

Breakdown:
  - Parameter x: Float (inferred from usage in +)
  - Body (x + 0.5): Float (numeric promotion: Int + Float → Float)
  - Result type: Float -> Float
```

## Success Criteria

- [ ] `ailang debug types --trace-inference` shows all inference steps with NodeID, type, constraints
- [ ] `ailang debug types --show-gaps` lists all nodes missing CoreTypeInfo
- [ ] `ailang debug types --query <expr>` shows inferred type for expression
- [ ] Colorized output with `--no-color` flag for CI/logs
- [ ] Compact mode with `--compact` flag for dense output
- [ ] Integration with M-DX4 validator (shares gap-finding logic)
- [ ] All tests passing
- [ ] Documentation updated (docs/architecture/types.md with literal resolution table)
- [ ] Example debug sessions in docs

## Testing Strategy

**Unit tests:**
- Test trace collection for various expressions (literals, lambdas, applications)
- Test gap detection matches M-DX4 validator
- Test query mode finds correct NodeID and type
- Test output formatting (table, tree, colorized, compact)

**Integration tests:**
- Run debug on example.ail files with known types
- Verify trace includes all Core nodes
- Test gap detection on files with missing CoreTypeInfo (simulated bugs)
- Test query mode with various expression patterns

**Manual testing:**
- Debug real AILANG programs with type errors
- Verify trace helps diagnose issues
- Check colorized output renders correctly in terminal
- Test `--compact` mode readability

## Non-Goals

**Not in this feature:**
- Performance profiling of type checker - Separate concern, addressed later if needed
- Interactive debugger - Too complex for v0.3.15, future work
- Query tool for arbitrary expressions - Only works on code in files for now
- Visual AST rendering - Nice-to-have, deferred to later
- Automatic fix suggestions - Out of scope, requires more sophistication

## Timeline

**Day 1** (3 hours):
- Phase 1: CLI infrastructure
- Phase 2: Inference tracing (start)

**Day 2** (3 hours):
- Phase 2: Inference tracing (finish)
- Phase 3: CoreTypeInfo inspection
- Phase 4: Documentation

**Total: ~6 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Trace collection slows down compilation | Medium | Make tracing opt-in (only when --trace-inference used) |
| Trace output too verbose | Low | Add --compact mode, colorize for readability |
| Gap detection duplicates M-DX4 validator | Low | Share code between validator and inspector |
| Query mode hard to implement | Medium | Start with simple lookup, defer advanced queries |

## References

- Field report: User's "AI-first DX reflection" from October 2025
- M-DX4: CoreTypeInfo Completeness (provides validation foundation)
- [CLAUDE.md](../../../CLAUDE.md#2-no-silent-fallbacks---fail-loudly) - Fail loudly principle
- Literal resolution policy (to be documented in docs/architecture/types.md)

## Future Work

- M-DX7: Interactive type debugger (REPL for type queries)
- M-DX8: Visual AST rendering (SVG/DOT output for `ailang debug ast --format=svg`)
- M-DX9: Automatic fix suggestions (suggest type annotations to resolve errors)
- Performance profiling: `ailang debug perf --profile-inference`
- Query tool for snippets: `echo "(\x -> x + 1)" | ailang debug types --query -`

---

**Document created**: 2025-10-22
**Last updated**: 2025-10-22
