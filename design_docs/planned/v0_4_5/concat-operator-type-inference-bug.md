# [Feature Name]

**Status**: Planned
**Target**: [Version, e.g., v0.4.0]
**Priority**: [P0/P1/P2 - High/Medium/Low]
**Estimated**: [Time estimate, e.g., 2 days]
**Dependencies**: [None or list other features]

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | [+/0/−] | [+1/0/−1] | [e.g., "Removes import boilerplate"] |
| Preserve Semantic Clarity | [+/0/−] | [+1/0/−1] | [e.g., "Effects remain explicit in types"] |
| Increase Determinism | [+/0/−] | [+1/0/−1] | [e.g., "Injection is deterministic per entry module"] |
| Lower Token Cost | [+/0/−] | [+1/0/−1] | [e.g., "~30 token reduction per example"] |
| **Net Score** | | **[Total]** | **Decision: [Move forward / Reject / Redesign]** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

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

## References

- [Link to related design docs]
- [Link to issues or discussions]
- [Link to prior art or research]

## Future Work

[Features that build on this but are out of scope for now]

---

**Document created**: 2025-11-16
**Last updated**: 2025-11-16
