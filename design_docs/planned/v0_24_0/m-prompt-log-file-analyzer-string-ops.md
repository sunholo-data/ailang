# M-PROMPT-LOG-FILE-ANALYZER: String ops pipeline example (split→filter→map→join)

> **📊 RECENT-VERIFIED: dot-notation only 2% of recent failures. log_file_analyzer is multi-causal (++ + alias + parse); no single fix.** (verified 2026-06-03 against Apr-Jun 2026 data only — not all-time aggregate.)

**Status**: Planned
**Target**: v0.24.0

## Summary

`log_file_analyzer` is the **single highest-impact benchmark** — fails in 9/9 frontier
models. Primary cause: import aliases (see m-import-alias.md). Secondary cause after
that fix: models use Python dot-notation string ops (`.split()`, `.contains()`,
`.trim()`) instead of AILANG's function-call style.

**Evidence (from actual generated code):**
```
import std/string (split, contains, trim, join, substring, find, length, compare)
import std/list (map, filter, length as listLen, ...)
-- then uses: content.split("\n") instead of split(content, "\n")
```

**Fix:** Add a "log file processing" canonical example showing:
```ailang
import std/string (split, contains, trim, join)
import std/list (map, filter, foldl)

-- Split a string: split(str, delimiter)  NOT str.split(delimiter)
let lines = split(content, "\n");
-- Filter lines containing a pattern:
let errors = filter(\line. contains(line, "ERROR"), lines);
-- Extract a field:
let msgs = map(\line. trim(substring(line, 20, length(line))), errors);
-- Join back:
let result = join("\n", msgs)
```

**Status**: Planned
**Target**: v0.24.0
**Priority**: P2 (Medium) — fix m-import-alias first; this is the secondary cause
**Estimated**: 0.5 day
**Dependencies**: m-import-alias (fix that first to reveal which failures remain)
**Priority**: [P0/P1/P2 - High/Medium/Low]
**Estimated**: [Time estimate, e.g., 2 days]
**Dependencies**: [None or list other features]

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No determinism impact |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification change |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Teaches function-call string ops, reducing dot-method compile failures |
| A8: Minimal Syntax | 0 | No syntax change (prompt only) |
| A9: Cost Visibility | 0 | No resource-cost changes |
| A10: Composability | +1 | split→filter→map→join pipeline composes cleanly |
| A11: Structured Failure | 0 | No error-handling change |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Proceed** (P3 — low recent frequency; ship with other prompt updates)


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

## High-Impact Decisions

<!-- What choices are being made? Not "what we're building" (that's Solution Design) but
     "what we're deciding." Chosen By: human = needs approval, agent = implementer decides,
     compiler = language semantics decide. Deadline: design = before coding, compile = before
     shipping, runtime = may remain flexible. Change Cost: high = architectural ripple,
     med = multi-file, low = localized. Aim for 3-7 rows. -->

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| [Decision 1] | [Why it matters] | [human/agent/compiler] | [design/compile/runtime] | [high/med/low] |
| [Decision 2] | [Why it matters] | [human/agent/compiler] | [design/compile/runtime] | [high/med/low] |

### Design Freeze

<!-- Every "high" change-cost decision above must appear here as a checkbox.
     Unchecked items = sprint-executor should PAUSE for human input. -->

Before implementation begins, these must be resolved:

- [ ] [Decision that must be made before coding]
- [ ] [Decision that must be made before coding]

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

## Deferred Decisions

<!-- NOT the same as Non-Goals. Non-Goals = "we won't do X."
     Deferred Decisions = "we WILL do X but haven't decided HOW yet."
     This tells agents where they have latitude. Always say who may resolve. -->

The following are intentionally left open for the implementer:

- [Decision 1] — [who may resolve, e.g., "agent may choose"]
- [Decision 2] — [who may resolve]

## Non-Goals

**Not attempted in this feature:**
- [Thing 1] - [Why out of scope]
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

<!-- Auto-populated by Ollama neural search on "prompt log file analyzer string ops" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_5_7/m-codegen-typed-slices.md](design_docs/implemented/v0_5_7/m-codegen-typed-slices.md) (1.00)
- [design_docs/implemented/v0_5_3/m-dx12-typed-adt-slices.md](design_docs/implemented/v0_5_3/m-dx12-typed-adt-slices.md) (0.95)
- [design_docs/implemented/v0_5_9/m-codegen-getopt-typed-slices.md](design_docs/implemented/v0_5_9/m-codegen-getopt-typed-slices.md) (0.90)

**Planned (check for overlap):**
- [design_docs/planned/v0_13_0/m-arch3-task-classification.md](design_docs/planned/v0_13_0/m-arch3-task-classification.md) (1.00)
- [design_docs/planned/v1_0_0/m-csp-session-types.md](design_docs/planned/v1_0_0/m-csp-session-types.md) (0.95)
- [design_docs/planned/v0_13_0/m-arch5-error-handling-strategy.md](design_docs/planned/v0_13_0/m-arch5-error-handling-strategy.md) (0.90)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [Link to related design docs]
- [Link to issues or discussions]

## Future Work

[Features that build on this but are out of scope for now]

---

**Document created**: 2026-06-03
**Last updated**: 2026-06-03
