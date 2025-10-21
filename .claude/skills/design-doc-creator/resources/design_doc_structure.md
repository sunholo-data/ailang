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

### References

```markdown
## References

- **Motivation**: `design_docs/planned/easier-ailang-dev.md`
- **Prior art**: Ruby DSL for types, Rust procedural macros
- **Related issues**: #42, #58
- **Discussion**: Slack thread (Oct 10, 2024)
```

**Purpose**: Link to context and background.

**Tips:**
- Link to related design docs
- Reference issues/PRs
- Cite prior art or research
- Include discussion links

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

**After creating:**
- [ ] Review for clarity and completeness
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
