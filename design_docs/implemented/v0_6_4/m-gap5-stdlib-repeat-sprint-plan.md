# Sprint Plan: M-GAP5 - Add `repeat` to std/string

## Summary
Add a pure `repeat(s, n)` function to the std/string module for string repetition - a common operation needed for formatting, box-drawing, and test data generation.

**Duration:** 2 hours (single session)
**Dependencies:** None
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- std/string already exists with ~13 exported functions
- v0.6.5 added `join(delimiter, xs)` function (latest stdlib addition)
- Pattern: Pure functions wrapping builtins or using AILANG recursion

### Velocity
- Recent stdlib additions: ~10-30 LOC per function
- This sprint: ~25 LOC total (implementation + tests + example)
- Single session estimate: 2 hours

### Design Doc Reference
- Design doc: `design_docs/planned/v0_6_4/m-gap5-stdlib-repeat.md`
- Recommended approach: Pure AILANG implementation (Option A)
- API: `repeat(s: string, n: int) -> string`

## Proposed Milestones

### Milestone 1: Implementation
**Goal:** Add `repeat` function to std/string module
**Estimated:** ~5 LOC implementation
**Duration:** 30 minutes

**Tasks:**
1. Add `repeat` function to `std/string.ail` after existing functions
2. Add `export` declaration for the function
3. Verify syntax with `ailang check std/string.ail`

**Implementation:**
```ailang
-- | Repeat a string n times.
-- | repeat("ab", 3) = "ababab"
-- | repeat("x", 0) = ""
export pure func repeat(s: string, n: int) -> string =
  if n <= 0 then "" else s ++ repeat(s, n - 1)
```

**Acceptance Criteria:**
- [ ] Function added to std/string.ail
- [ ] Function exported in module
- [ ] `ailang check std/string.ail` passes

**Risks:**
- None - simple pure function addition

### Milestone 2: Testing
**Goal:** Verify `repeat` function works correctly
**Estimated:** ~15 LOC test file
**Duration:** 45 minutes

**Tasks:**
1. Create test file `examples/runnable/string_repeat.ail`
2. Test basic repetition cases
3. Test edge cases (n=0, n=1, empty string, negative n)
4. Test Unicode support
5. Run tests with `ailang run --caps IO --entry main examples/runnable/string_repeat.ail`

**Test Cases:**
- `repeat("x", 3)` = `"xxx"`
- `repeat("ab", 2)` = `"abab"`
- `repeat("x", 0)` = `""`
- `repeat("x", 1)` = `"x"`
- `repeat("", 5)` = `""`
- `repeat("x", -1)` = `""`

**Acceptance Criteria:**
- [ ] Test file created in examples/runnable/
- [ ] All test cases pass
- [ ] Edge cases covered
- [ ] `make verify-examples` includes new file

**Risks:**
- None - straightforward testing

### Milestone 3: Verification
**Goal:** Ensure stdlib integrity and documentation
**Estimated:** ~5 LOC (example usage)
**Duration:** 45 minutes

**Tasks:**
1. Run `make test` to verify no regressions
2. Run `make lint` to verify code quality
3. Verify import works: `import std/string (repeat)`
4. Update design doc status from "Planned" to "Implemented"
5. Add CHANGELOG entry (if releasing)

**Acceptance Criteria:**
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Import syntax verified working
- [ ] Design doc status updated
- [ ] All success criteria from design doc met

**Risks:**
- None - verification only

## Success Metrics
- Test coverage: All test cases passing
- Examples passing: string_repeat.ail works
- Documentation: Design doc updated to Implemented
- All tests passing: `make test`
- All linting passing: `make lint`

## Dependencies
- None - self-contained stdlib addition

## Open Questions
- None - implementation approach clearly defined in design doc

## Notes
- Following Option A from design doc: Pure AILANG implementation
- O(n) string allocations acceptable for typical use cases (n < 1000)
- If performance becomes an issue, can add builtin later (Option B)
- Follows existing pattern in std/string (see `join` function for similar recursive approach)
