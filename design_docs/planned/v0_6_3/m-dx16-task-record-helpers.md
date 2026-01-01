# [Feature Name]

**Status**: Planned
**Target**: [Version, e.g., v0.4.0]
**Priority**: [P0/P1/P2 - High/Medium/Low]
**Estimated**: [Time estimate, e.g., 2 days]
**Dependencies**: [None or list other features]

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | [+1/0/−1] | [e.g., "Enables reproducible traces"] |
| A2: Replayability | [+1/0/−1] | [e.g., "No impact on traces"] |
| A3: Effect Legibility | [+1/0/−1] | [e.g., "Makes IO effects explicit"] |
| A4: Explicit Authority | [+1/0/−1] | [e.g., "Enforces capability constraints"] |
| A5: Bounded Verification | [+1/0/−1] | [e.g., "Enables local type checks"] |
| A6: Safe Concurrency | [+1/0/−1] | [e.g., "No concurrency changes"] |
| A7: Machines First | [+1/0/−1] | [e.g., "Reduces AI token cost"] |
| A8: Minimal Syntax | [+1/0/−1] | [e.g., "No new syntax required"] |
| A9: Cost Visibility | [+1/0/−1] | [e.g., "Resource costs remain visible"] |
| A10: Composability | [+1/0/−1] | [e.g., "Composes with existing effects"] |
| A11: Structured Failure | [+1/0/−1] | [e.g., "Errors remain typed"] |
| A12: System Boundary | [+1/0/−1] | [e.g., "Boundary crossings explicit"] |

**Net Score: [Total]** → **Decision: [Move forward / Reject / Redesign]**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [ ] A1 (Determinism): No implicit nondeterminism introduced
- [ ] A3 (Effects): No hidden side effects
- [ ] A4 (Authority): No ambient access granted
- [ ] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

[What problem does this solve? Why is it needed?]

**Current State:**
- [Describe current pain points]
- [Include metrics if available]

**Impact:**
- [Who is affected?]
- [How significant is the problem?]

## Goals

**Primary Goal:** [Main objective in one sentence]

**Success Metrics:**
- [Measurable outcome 1]
- [Measurable outcome 2]
- [Measurable outcome 3]

## Solution Design

### Overview

[High-level description of the solution]

### Architecture

[Describe the technical approach]

**Components:**
1. **Component 1**: [Description]
2. **Component 2**: [Description]
3. **Component 3**: [Description]

### Implementation Plan

**Phase 1: [Name]** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

**Phase 2: [Name]** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

**Phase 3: [Name]** (~X hours)
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

### Files to Modify/Create

**New files:**
- `path/to/new_file.go` - [Purpose, ~XXX LOC]

**Modified files:**
- `path/to/existing_file.go` - [Changes needed, ~XXX LOC]

## Examples

### Example 1: [Use Case]

**Before:**
```
[Code or workflow before the change]
```

**After:**
```
[Code or workflow after the change]
```

### Example 2: [Use Case]

[Additional examples as needed]

## Success Criteria

- [ ] Criterion 1 (with acceptance test)
- [ ] Criterion 2 (with acceptance test)
- [ ] Criterion 3 (with acceptance test)
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- [What to test]

**Integration tests:**
- [What to test]

**Manual testing:**
- [What to verify manually]

## Non-Goals

**Not in this feature:**
- [Thing 1] - [Why deferred]
- [Thing 2] - [Why out of scope]

## Timeline

**Week 1** (X hours):
- Phase 1 implementation

**Week 2** (X hours):
- Phase 2 implementation
- Testing

**Week 3** (X hours):
- Phase 3 implementation
- Documentation
- Release

**Total: ~X hours across Y weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| [Risk 1] | [High/Med/Low] | [How to address] |
| [Risk 2] | [High/Med/Low] | [How to address] |

## Related Documents

<!-- Auto-populated by semantic search on "task record helpers" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_5_10/m-codegen-nested-record-type.md](design_docs/implemented/v0_5_10/m-codegen-nested-record-type.md) (1.00)
- [design_docs/implemented/v0_4_8/m-bug-record-update-inference-sprint-plan.md](design_docs/implemented/v0_4_8/m-bug-record-update-inference-sprint-plan.md) (0.95)
- [design_docs/implemented/v0_5_10/m-cross-module-record-unification.md](design_docs/implemented/v0_5_10/m-cross-module-record-unification.md) (0.90)

**Planned (check for overlap):**
- [design_docs/planned/v0_6_3/m-coord-taskchain-tests-sprint-plan.md](design_docs/planned/v0_6_3/m-coord-taskchain-tests-sprint-plan.md) (1.00)
- [design_docs/planned/v0_6_2/m-collab-provider-stats-sprint-plan.md](design_docs/planned/v0_6_2/m-collab-provider-stats-sprint-plan.md) (0.95)
- [design_docs/planned/v0_7_0/m-poly-arithmetic-fix.md](design_docs/planned/v0_7_0/m-poly-arithmetic-fix.md) (0.90)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [Link to related design docs]
- [Link to issues or discussions]

## Future Work

[Features that build on this but are out of scope for now]

---

**Document created**: 2026-01-01
**Last updated**: 2026-01-01
