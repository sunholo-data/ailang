# Design Document Structure Guide

Complete reference for AILANG design documents.

## Template Breakdown

### Header Section

```markdown
# [Feature Name]

**Status**: Planned | Implemented
**Target**: v0.4.0
**Priority**: P0 (High) | P1 (Medium) | P2 (Low)
**Estimated**: 3 days
**Dependencies**: None | Feature X, Feature Y
```

**Purpose**: Quick metadata for understanding scope and priority.

**Tips:**
- Use title case for feature name
- Status starts as "Planned", changes to "Implemented" when done
- Target should be specific version (v0.X.Y)
- Priority determines scheduling (P0 = next release, P1 = soon, P2 = eventually)
- Estimated should be realistic (2x initial guess)
- Dependencies help with scheduling

### Problem Statement

```markdown
## Problem Statement

[What problem does this solve? Why is it needed?]

**Current State:**
- Pain point 1 (with metrics)
- Pain point 2 (with data)
- Pain point 3 (with impact)

**Impact:**
- Who is affected? (developers, users, AI models)
- How significant? (blocker, annoyance, nice-to-have)
```

**Purpose**: Justify why this feature matters.

**Tips:**
- Include real metrics (hours, lines of code, error rates)
- Be specific about pain points
- Quantify impact when possible
- Link to issues or discussions if available

**Good example:**
```markdown
Adding a builtin function currently takes **7.5 hours** due to:
- Scattered registration across 4 files (2+ hours debugging)
- Verbose type construction (1+ hour trial-and-error)
- Poor error messages (1+ hour debugging)

**Impact:** Slows language evolution, hard for new contributors.
```

**Bad example:**
```markdown
Development is slow and hard.
```

### Goals

```markdown
## Goals

**Primary Goal:** [One sentence main objective with measurable outcome]

**Success Metrics:**
- Metric 1 (e.g., reduce time from X to Y)
- Metric 2 (e.g., improve coverage from X% to Y%)
- Metric 3 (e.g., reduce files from X to Y)
```

**Purpose**: Define what success looks like.

**Tips:**
- Primary goal should be measurable
- Success metrics should be objective
- Use before/after comparisons
- Include 3-5 metrics maximum

**Good example:**
```markdown
**Primary Goal:** Reduce builtin development time from 7.5h to 2.5h (-67%)

**Success Metrics:**
- Files touched: 4 → 1 (-75%)
- Type construction LOC: 35 → 10 (-71%)
- Registration errors caught at compile/startup
```

**Bad example:**
```markdown
**Primary Goal:** Make things better

**Success Metrics:**
- Easier to use
- Faster
```

### High-Impact Decisions

```markdown
## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| [Decision 1] | [Why it matters] | [human/agent/compiler] | [design/compile/runtime] | [high/med/low] |
```

**Purpose**: Force authors to name the actual choices being made and who has authority to make them. This is the bridge between constraints (axioms) and implementation.

**Column definitions:**
- **Decision**: What is being decided (not "what we're building" — that's Solution Design)
- **Why High Impact**: What breaks or gets expensive if this decision changes later
- **Chosen By**: `human` (requires explicit approval), `agent` (implementer may choose), `compiler` (language semantics decide)
- **Deadline**: `design` (before coding), `compile` (before shipping), `runtime` (may remain flexible)
- **Change Cost**: `high` (architectural), `med` (multi-file), `low` (localized)

**Tips:**
- Put this BEFORE Solution Design — decisions should inform the design, not follow from it
- 3-7 rows is typical. If you have fewer than 3, the feature may not need a design doc
- If "Chosen By" is always "human", the doc may be over-constrained for agent execution
- If "Change Cost" is always "low", consider whether a design doc is needed at all

**Good example:**
```markdown
| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Validation at compile time, not runtime | Shapes compiler architecture | human | design | high |
| Single registry file per module | Affects file layout conventions | agent | implementation | low |
| Error messages use structured format | Determines downstream tooling | human | design | med |
```

**Bad example:**
```markdown
| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Build the feature | It's the whole point | human | design | high |
```

### Design Freeze

```markdown
### Design Freeze

Before implementation begins, these must be resolved:

- [ ] [Decision that must be locked before coding starts]
- [ ] [Decision that must be locked before coding starts]
```

**Purpose**: Separate what's decided from what's still open. Tells agents when to pause for human input vs. proceed.

**Tips:**
- Every "high" change-cost row from High-Impact Decisions should appear here
- Use checkboxes — check them off during design review, before sprint execution
- If a Design Freeze item is unchecked when sprint-executor starts, it should pause
- Keep to 3-7 items. More than 7 means the design is too uncertain to implement

**Good example:**
```markdown
- [x] Registry lives in `internal/builtins/` (not `internal/runtime/`)
- [x] Type builder uses fluent API (not struct literals)
- [ ] Error message format for validation failures
```

### Deferred Decisions

```markdown
## Deferred Decisions

The following are intentionally left open for the implementer:

- [Decision 1] — [who may resolve, e.g., "agent may choose"]
- [Decision 2] — [who may resolve]
```

**Purpose**: Explicitly grant latitude. This is NOT the same as Non-Goals. Non-Goals = "we won't do X." Deferred Decisions = "we will do X but haven't decided how yet, and that's intentional."

**Why this matters for agents**: Without this section, agents either over-constrain (treating everything not specified as forbidden) or under-constrain (guessing at unspecified details). This section tells them: "you have authority here."

**Tips:**
- Always specify who may resolve: `agent`, `human at review`, `compiler`
- If something appears here, it should NOT also appear in Design Freeze
- 3-10 items is typical
- Review after implementation — deferred items that caused problems should become Design Freeze items in future docs

**Good example:**
```markdown
- CLI output formatting — agent may choose
- Internal helper function naming — agent may choose
- Test fixture organization — agent may choose
- Caching strategy for large inputs — human at review
```

**Bad example (too vague):**
```markdown
- Details TBD
- Will figure it out later
```

### Solution Design

```markdown
## Solution Design

### Overview

[High-level description in 2-3 paragraphs]

### Architecture

[Technical approach, component breakdown]

**Components:**
1. **Component 1**: Description + responsibility
2. **Component 2**: Description + responsibility
3. **Component 3**: Description + responsibility

### Implementation Plan

**Phase 1: Foundation** (~X hours)
- [ ] Task 1 with clear deliverable
- [ ] Task 2 with clear deliverable
- [ ] Task 3 with clear deliverable

**Phase 2: Integration** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

**Phase 3: Polish** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
```

**Purpose**: Explain how you'll solve the problem.

**Tips:**
- Overview should be understandable by non-experts
- Break into 3-5 phases maximum
- Each task should have a clear definition of done
- Estimate each phase separately
- Use checkboxes for tracking

**Pipeline Pass Checklist** (include when adding/modifying compiler pipeline passes):

If your feature adds or modifies a compiler pipeline pass, verify it handles ALL expression sources. Omitting contract expressions has caused repeated regressions (M-CONTRACTS-OPLOWERING v0.8.0, M-CONTRACT-OPLOWERING-FIX v0.9.2).

```markdown
### Pipeline Pass Coverage

- [ ] `prog.Decls` — top-level declarations
- [ ] `prog.Meta[*].Contracts[*].Expr` — contract expressions (requires/ensures)
- [ ] Integration test: `TestContractExpressionsFullyLowered` still passes
```

**Good example:**
```markdown
### Overview

Create a central builtin registry where all builtins are registered once with:
- Function implementation
- Type signature (using builder DSL)
- Arity and effect metadata
- Validation on startup

### Architecture

**Components:**
1. **BuiltinSpec**: Struct holding all builtin metadata
2. **Registry**: Map of name → spec with validation
3. **Type Builder**: Fluent DSL for type construction
4. **Validator**: Startup checks for consistency

### Implementation Plan

**Phase 1: Registry Core** (~4 hours)
- [ ] Create BuiltinSpec struct
- [ ] Implement Register() function
- [ ] Add validation logic
- [ ] Write unit tests (100% coverage)
```

### Files to Modify/Create

```markdown
### Files to Modify/Create

**New files:**
- `internal/builtins/spec.go` (~300 LOC) - BuiltinSpec struct and registry
- `internal/builtins/validator.go` (~150 LOC) - Validation logic
- `internal/types/builder.go` (~200 LOC) - Type builder DSL

**Modified files:**
- `internal/runtime/builtins.go` (+50/-100 LOC) - Use registry instead of manual registration
- `internal/link/builtin_module.go` (+20/-200 LOC) - Read types from registry
```

**Purpose**: Help estimate scope and identify affected areas.

**Tips:**
- List all files that will change
- Estimate LOC for new files
- Show +/- LOC for modified files
- Include purpose for each file
- Group by package

### Examples

```markdown
## Examples

### Example 1: Adding a String Builtin

**Before (scattered across 4 files):**
```go
// internal/effects/string.go
func strReverse(...) { ... }

// internal/runtime/builtins.go
case "_str_reverse": result = effects.strReverse(...)

// internal/builtins/registry.go
"_str_reverse": {NumArgs: 1, IsPure: true}

// internal/link/builtin_module.go
"_str_reverse": &types.TFunc2{...} // 35 lines
```

**After (single file):**
```go
// internal/builtins/register.go
RegisterEffectBuiltin(BuiltinSpec{
    Name: "_str_reverse",
    NumArgs: 1,
    IsPure: true,
    Type: func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.String()).Returns(T.String())
    },
    Impl: strReverseImpl,
})
```
```

**Purpose**: Show concrete impact of the change.

**Tips:**
- Use real code examples
- Show before/after comparison
- Include 2-3 different use cases
- Make examples runnable
- Highlight key improvements

### Success Criteria

```markdown
## Success Criteria

- [ ] All builtins migrated to new registry
- [ ] `ailang doctor builtins` validates registration
- [ ] Type Builder DSL reduces construction from 35→10 LOC
- [ ] Development time measured at <3 hours for new builtin
- [ ] All tests passing (90%+ coverage on new code)
- [ ] Documentation updated (CLAUDE.md, ADDING_BUILTINS.md)
- [ ] Examples working (`make verify-examples`)
```

**Purpose**: Define what "done" means.

**Tips:**
- Use checkboxes
- Make criteria objective and testable
- Include test coverage requirements
- Always include "All tests passing"
- Always include "Documentation updated"
- Include performance/metric targets

### Conflict Surface (REQUIRED for parser/lexer/typechecker/codegen changes)

**This section is MANDATORY when the design touches any of:**
- `internal/parser/`, `internal/lexer/`, `internal/ast/`
- `internal/types/`, `internal/elaborate/`, `internal/iface/`
- `internal/codegen/`, `internal/eval/`, `internal/vm/`
- `internal/effects/` (effect-row algebra changes)
- `cmd/ailang/exec.go` and other compilation entry points

**Purpose**: Force the author to enumerate, at design time, what existing valid programs could break. Catches the "we didn't think about that interaction" regressions before they ship — see [M-PARSER-REFINEMENT-LOOKAHEAD](../../../changelogs/v0.10-v0.17-bytecode-vm.md) for the case study where M-TAINT-TYPES added `T{not LABEL}` syntax without enumerating that `func ... -> bool { not f(x) }` already used the same `{ not <ident>` prefix in function bodies. Cost a real consumer (motoko_agent fork) ~14 mis-parses.

**Required content**:

```markdown
## Conflict Surface

### Syntactic positions touched

What grammar productions / parser entry points / token positions does this
change extend? List them concretely:

- "Adds a new suffix `T<...>` after type expressions, parsed by
  `parseLabelOrRefinementSuffix` at parser_type.go:222"
- "Modifies `parseInfixExpression` to recognize a new operator at
  precedence band 3"
- "Changes the AST shape of `*ast.FuncDecl` to add a new field"

### What else lives here

For each position above, enumerate the OTHER valid constructs that already
appear in that position. Do NOT just say "function bodies use `{`" —
write out the SHAPE of what comes after the `{`:

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| After type in return-type position | function body | `{ <expr> }` where `<expr>` can be any expression |
| After type in return-type position | refinement (this PR) | `{ not <ident> }` |
| After type in return-type position | (any third claimer? e.g. record literal in a let-binding) | enumerate |

If two positions look identical at the lookahead depth your parser uses,
either:
  (a) extend the lookahead (preferred — see M-PARSER-REFINEMENT-LOOKAHEAD's
      4-token approach), OR
  (b) explain why the disambiguation is sound (e.g. "refinements only
      appear in type-annotation contexts, which the caller tracks")

### Disambiguation strategy

How does the parser/typechecker decide which interpretation applies?
Show the exact token-stream check or context flag that picks the right
path. If you say "context X never appears with construct Y", explain
WHY the grammar prevents that.

### Programs that MUST still work

List 3-5 existing programs (file paths in `examples/` or `std/`, or
external consumers like motoko_agent) that exercise the same syntactic
position your change touches. These become regression test fixtures
in M1. If you can't name 3-5, your test corpus is too narrow — search
harder before merging.

### What deliberately changes

If the change WILL break some previously-valid construct (e.g. a
deprecated form is being removed), say so explicitly. List the affected
syntax + the migration path. Anything not listed here that breaks is
a regression, not an intentional change.
```

**Why this section exists**: most language regressions come from "I didn't realize that other thing also uses this position." The author is closer to the new feature than anyone else; they're the only one who can credibly enumerate the conflict surface. Reviewers can sanity-check but can't generate the list.

**Failure mode without this section**: tests cover what the author wrote (positive cases for the new feature, specific negative cases the author thought of). Sibling syntax — valid pre-change — silently breaks. Caught only when an external consumer hits it.

**Reviewer rule**: if this section says "no conflicts" for a parser/typechecker change, push back. The honest answer is almost always "here are the candidates and here's why each is safe."

### Testing Strategy

```markdown
## Testing Strategy

**Unit tests:**
- Registry validation (invalid specs rejected)
- Type Builder DSL (all type combinations)
- Each builtin impl (hermetic, mocked effects)

**Integration tests:**
- End-to-end builtin calls from REPL
- Cross-module builtin usage
- Error propagation

**Regression-surface tests** (REQUIRED if Conflict Surface section was filled in):
- One test per "Programs that MUST still work" entry. Pin the exact
  parse/typecheck/eval output (or AST snapshot). Failures here are
  regressions, not test churn.

**Manual testing:**
- Add new builtin in <3 hours
- Run `ailang doctor builtins`
- Check REPL `:type` output
```

**Purpose**: Ensure quality and catch regressions.

**Tips:**
- Break into unit/integration/manual
- List what each test level covers
- Include acceptance tests
- Specify coverage targets
- Mention test files to create

### Non-Goals

```markdown
## Non-Goals

**Not in this feature:**
- Auto-generated documentation from specs - [Deferred to M-DX2]
- Hot-reload builtins for development - [Out of scope]
- Performance benchmarks - [Not critical]
- CI checks for legacy patterns - [Nice-to-have]
```

**Purpose**: Set boundaries and manage expectations.

**Tips:**
- List what you're NOT doing
- Explain why (deferred, out of scope, not valuable)
- Link to future work if applicable
- Prevents scope creep

### Timeline

```markdown
## Timeline

**Week 1** (12 hours):
- Phase 1: Registry core (4h)
- Phase 2: Type Builder (3h)
- Phase 3: Validation (2h)
- Testing + Polish (3h)

**Week 2** (8 hours):
- Migrate 10 builtins (5h)
- Documentation (2h)
- Final testing (1h)

**Total: ~20 hours across 2 weeks**
```

**Purpose**: Realistic scheduling and resource planning.

**Tips:**
- Break by week or phase
- Include buffer time
- Account for testing and docs
- 2x your initial estimates
- Show cumulative total

### Risks & Mitigations

```markdown
## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Migration breaks existing builtins | High | Incremental migration, test after each builtin |
| Type Builder too complex | Medium | Start simple, add features as needed |
| Performance regression | Low | Benchmark before/after, registry is cached |
```

**Purpose**: Identify potential issues early.

**Tips:**
- List 3-5 main risks
- Rate impact (High/Medium/Low)
- Include concrete mitigation plan
- Update as risks change

### Axiom Compliance

```markdown
## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Enables reproducible X |
| A2: Replayability | 0 | No impact |
| A3: Effect Legibility | +1 | Makes Y effects explicit |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Enables local Z checks |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Reduces token cost for AI |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Composes with existing W |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** ✅ Proceed to implementation

### Hard Violation Check

- [ ] A1 (Determinism): No implicit nondeterminism introduced
- [ ] A3 (Effects): No hidden side effects
- [ ] A4 (Authority): No ambient access granted
- [ ] A7 (Machines First): Not optimizing for human convenience over machine analysis
```

**Purpose**: Verify feature aligns with AILANG's core principles.

**Tips:**
- Score EVERY axiom (use 0 for no impact)
- Justify non-zero scores with specific reasoning
- Hard violations (−1 on A1, A3, A4, A7) require redesign
- Net score ≥ +2 to proceed
- Include the hard violation checklist

**Decision thresholds:**
- **≥ +2**: Proceed to draft
- **0 to +1**: Needs stronger justification
- **< 0**: Reject or redesign
- **Any −1 on A1, A3, A4, A7**: Automatic rejection

### References

```markdown
## References

- **Motivation**: `design_docs/planned/easier-ailang-dev.md`
- **Prior art**: Ruby DSL for types, Rust procedural macros
- **Related issues**: #42, #58
- **Discussion**: Slack thread (Oct 10, 2024)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)
```

**Purpose**: Link to context and background.

**Tips:**
- Link to related design docs
- Reference issues/PRs
- Cite prior art or research
- Include discussion links
- Always include axiom reference link

### Future Work

```markdown
## Future Work

[Features that build on this:]
- Auto-documentation generation from BuiltinSpec
- LSP integration for builtin type hints
- Hot-reload for faster development iteration
```

**Purpose**: Capture ideas without committing to them.

**Tips:**
- List logical next steps
- Don't commit to timeline
- Link to non-goals if applicable
- Keep brief (1-2 sentences each)

## Implementation Report (for Implemented docs)

Add this section when moving to `implemented/`:

```markdown
## Implementation Report

**Completed**: 2024-10-15
**Version**: v0.3.10

### What Was Built

[Summary of actual implementation vs plan]
- What changed from original design
- Why changes were made
- What worked well

### Code Locations

**New files:**
- `internal/builtins/spec.go` (342 LOC) - BuiltinSpec and registry
- `internal/builtins/validator.go` (187 LOC) - Validation
- `internal/types/builder.go` (234 LOC) - Type Builder DSL

**Modified files:**
- `internal/runtime/builtins.go` (+45/-120 LOC) - Uses registry
- `internal/link/builtin_module.go` (+18/-215 LOC) - Reads from registry

### Test Coverage

- Unit tests: 57 passing
- Integration tests: 12 passing
- Coverage: 92% on new code
- Test files: `internal/builtins/*_test.go`, `internal/types/builder_test.go`

### Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Development time | 7.5h | 2.5h | -67% |
| Files to edit | 4 | 1 | -75% |
| Type construction LOC | 35 | 10 | -71% |

### Known Limitations

- REPL `:type` command not yet implemented (deferred to M-DX1.6)
- Enhanced diagnostics pending (M-DX1.7)
- `_json_encode` migration pending (complex ADT handling)

### Future Work

See design_docs/planned/m-dx1-future-polish.md for:
- M-DX1.6: REPL `:type` command
- M-DX1.7: Enhanced error diagnostics
- M-DX1.8: Comprehensive documentation
```

**Purpose**: Document what actually happened.

**Tips:**
- Include actual LOC and metrics
- Show before/after data
- List known limitations
- Link to future work
- Reference test files
- Note deviations from plan

## Common Mistakes to Avoid

### 1. Vague Problem Statements

❌ Bad:
```markdown
Development is hard.
```

✅ Good:
```markdown
Adding a builtin takes 7.5 hours due to scattered registration (4 files),
verbose types (35 LOC), and poor errors (1+ hour debugging).
```

### 2. Unmeasurable Goals

❌ Bad:
```markdown
Make development easier and faster.
```

✅ Good:
```markdown
Reduce builtin development time from 7.5h to 2.5h (-67%)
```

### 3. Missing Implementation Details

❌ Bad:
```markdown
Implement the feature.
```

✅ Good:
```markdown
**Phase 1**: Create BuiltinSpec struct (~4h)
- [ ] Define struct in internal/builtins/spec.go
- [ ] Add Register() function
- [ ] Implement validation
- [ ] Write tests (100% coverage)
```

### 4. No Examples

❌ Bad:
```markdown
The new system is better.
```

✅ Good:
```markdown
**Before (4 files, 35 LOC):**
[code example]

**After (1 file, 10 LOC):**
[code example]
```

### 5. Unrealistic Estimates

❌ Bad:
```markdown
Should take about 2 hours.
```

✅ Good:
```markdown
**Estimated**: 3 days (6h implementation + 4h testing + 2h docs + buffer)
```

### 6. Missing Success Criteria

❌ Bad:
```markdown
Feature is done when it works.
```

✅ Good:
```markdown
- [ ] All 49 builtins migrated
- [ ] `ailang doctor builtins` passes
- [ ] Development time < 3 hours (measured)
- [ ] 90%+ test coverage
- [ ] Documentation updated
```

## Version Folder Organization

```
design_docs/
├── planned/
│   ├── unversioned-feature.md       # Version TBD
│   └── v0_4_0/                      # Targeted for v0.4.0
│       ├── reflection-system.md
│       └── schema-registry.md
└── implemented/
    ├── v0_3_10/                     # Shipped in v0.3.10
    │   ├── M-DX1_developer_experience.md
    │   └── M-DASH.md
    ├── v0_3_12/                     # Shipped in v0.3.12
    │   └── show-function-recovery.md
    └── v0_3_14/                     # Shipped in v0.3.14
        └── feature-x.md
```

**Rules:**
- Use underscores in folder names: `v0_3_14`
- Match CHANGELOG.md tags exactly
- Create version folder when first doc needs it
- Move entire folder when version ships
- Keep planned/ docs unversioned if target unclear

## Checklist

Use this when creating a new design doc:

**Before creating:**
- [ ] Feature is well-understood
- [ ] Priority and version target are known
- [ ] Similar features reviewed for patterns
- [ ] Dependencies identified
- [ ] **Axiom quick-check**: Does this obviously violate A1, A3, A4, or A7?

**When creating:**
- [ ] Run `create_planned_doc.sh`
- [ ] Fill in all header metadata
- [ ] Write specific problem statement with metrics
- [ ] Define measurable success criteria
- [ ] Break implementation into phases
- [ ] Include concrete before/after examples
- [ ] Estimate realistically (2x initial guess)
- [ ] List files to create/modify
- [ ] Add testing strategy
- [ ] Define non-goals
- [ ] **Complete Axiom Compliance section** (score all 12 axioms)
- [ ] **Verify net score ≥ +2** (or provide justification)
- [ ] **Check hard violations** (A1, A3, A4, A7 must not be −1)

**After creating:**
- [ ] Review for clarity and completeness
- [ ] Verify axiom scores are justified
- [ ] Get feedback from others if major feature
- [ ] Commit to git
- [ ] Link from related docs if needed

**When moving to implemented:**
- [ ] Run `move_to_implemented.sh`
- [ ] Add implementation report
- [ ] Include actual metrics
- [ ] List known limitations
- [ ] Update design_docs/README.md
- [ ] Update CHANGELOG.md
- [ ] Commit and delete from planned/
