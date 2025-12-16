# DX Improvement Patterns

**Comprehensive guide to improving AILANG development experience during sprint execution.**

This resource documents patterns for identifying and implementing DX improvements as you work.

## When to Think About DX

**DX-aware implementation:**
- If you're writing boilerplate, could it be a helper function?
- If you're debugging something, could a debug flag help?
- If you're looking things up repeatedly, should it be documented?
- If an error message confused you, would it confuse others?
- If a test is verbose, could test helpers make it cleaner?

## When to Act on DX Ideas

- 🟢 **Quick (<15 min)**: Do it now as part of this milestone
- 🟡 **Medium (15-30 min)**: Note in TODO list, do at end of milestone if time allows
- 🔴 **Large (>30 min)**: Note for design doc in reflection step

## Common DX Improvement Patterns

### 1. Repetitive Boilerplate → Helper Functions

**Signals:**
- Copying/pasting the same test setup code
- Same validation logic repeated across functions
- Common error handling patterns duplicated

**Quick fixes (5-15 min):**
- Extract to helper function in same package
- Add to `*_helpers.go` file
- Document with usage example
- Add tests for helper if complex

**Example:** M-DX9 added `AssertNoErrors(t, p)` after noticing parser test boilerplate.

**Before DX thinking:**
```go
if p.Errors() != nil {
    // Manually check each error...
}
```

**After DX thinking - Add helper:**
```go
AssertNoErrors(t, p)  // Helper added for reuse
```

### 2. Hard-to-Debug Issues → Debug Flags

**Signals:**
- Adding temporary `fmt.Printf()` statements
- Manually tracing execution flow
- Repeatedly inspecting internal state

**Quick fixes (5-10 min):**
- Add `DEBUG_<SUBSYSTEM>=1` environment variable check
- Gate debug output behind flag (zero overhead when off)
- Document in CLAUDE.md or code comments

**Example:** M-DX9 added `DEBUG_PARSER=1` for token flow tracing.

**Before DX thinking:**
```go
// Manually inspecting tokens with fmt.Printf
fmt.Printf("cur=%v peek=%v\n", p.curToken, p.peekToken)
```

**After DX thinking - Add debug mode:**
```go
// DEBUG_PARSER=1 automatically traces token flow
```

### 3. Manual Workflows → Make Targets

**Signals:**
- Running multi-step commands repeatedly
- Forgetting command flags or order
- Different team members using different commands

**Quick fixes (3-5 min):**
- Add `make <target>` with clear name
- Document what it does in `make help`
- Show example usage in relevant docs

**Example:** `make update-golden` for parser test golden files.

### 4. Confusing APIs → Documentation

**Signals:**
- Looking up API signatures multiple times
- Trial-and-error with function arguments
- Grep-diving to understand usage

**Quick fixes (10-20 min):**
- Add package-level godoc with examples
- Document common patterns in CLAUDE.md
- Add usage examples to function comments
- Create `make doc PKG=<package>` target if missing

**Example:** M-TESTING documented common API patterns in CLAUDE.md.

### 5. Poor Error Messages → Actionable Errors

**Signals:**
- Error doesn't explain what went wrong
- No suggestion for how to fix
- Missing context (line numbers, file names)

**Quick fixes (5-15 min):**
- Add context to error message
- Suggest fix or workaround
- Link to documentation if relevant
- Include values that triggered error

**Example:**
```go
// ❌ Before
return fmt.Errorf("parse error")

// ✅ After
return fmt.Errorf("parse error at %s:%d: expected RPAREN, got %s. Did you forget to close the argument list? See: https://ailang.sunholo.com/docs/guides/parser_development#common-issues",
    p.filename, p.curToken.Line, p.curToken.Type)
```

### 6. Painful Testing → Test Utilities

**Signals:**
- Verbose test setup/teardown
- Repeated value construction
- Brittle test assertions

**Quick fixes (10-20 min):**
- Create test helper package (e.g., `testctx/`)
- Add value constructors (e.g., `MakeString()`, `MakeInt()`)
- Add assertion helpers (e.g., `AssertNoErrors()`)

**Example:** M-DX1 added `testctx` package for builtin testing.

## DX ROI Calculator

**When deciding whether to implement a DX improvement:**

```
Time saved per use × Expected uses = Total savings
If Total savings > Implementation time + Maintenance → DO IT
```

**Examples:**
- Helper function: 2 min × 20 uses = 40 min saved, costs 10 min → ROI = 4x ✅
- Debug flag: 15 min × 5 uses = 75 min saved, costs 8 min → ROI = 9x ✅
- Documentation: 5 min × 30 uses = 150 min saved, costs 20 min → ROI = 7.5x ✅
- New skill: 30 min × 2 uses = 60 min saved, costs 120 min → ROI = 0.5x ❌ (create design doc for later)

**Note:** ROI compounds over time as more developers/sprints benefit!

## DX Reflection After Each Milestone

**After each milestone, reflect on the development experience:**

**Ask yourself:**
- What was painful during this milestone?
- What took longer than expected due to tooling gaps?
- What did we have to lookup multiple times?
- What errors/bugs could better tooling prevent?

**Categorize DX improvements:**

**🟢 Quick wins (<15 min) - Do immediately:**
- Add helper function to reduce boilerplate
- Add debug flag for better visibility
- Improve error message with actionable suggestion
- Add make target for common workflow
- Document pattern in code comments

**🟡 Medium improvements (15-30 min) - Add to current sprint if time allows:**
- Create test utility package
- Add validation script
- Improve CLI flag organization
- Add comprehensive examples

**🔴 Large improvements (>30 min) - Create design doc:**
- New skill for complex workflow
- Major architectural change
- New developer tool or subsystem
- Significant codebase reorganization

**Document in milestone summary:**
```markdown
## DX Improvements (Milestone X)

✅ **Applied**: Added `AssertNoErrors(t, p)` test helper (5 min)
📝 **Deferred**: Created M-DX10 design doc for parser AST viewer tool (estimated 2 hours)
💡 **Considered**: Better REPL error messages (added to backlog)
```

## DX Impact Summary Template

**Consolidate all DX improvements made during sprint:**

```markdown
## DX Improvements Summary (Sprint M-XXX)

### Applied During Sprint
✅ **Test Helpers** (Day 2, 10 min): Added `AssertNoErrors()` and `AssertLiteralInt()` helpers
   - Impact: Reduced test boilerplate by ~30%
   - Files: internal/parser/test_helpers.go

✅ **Debug Flag** (Day 4, 5 min): Added `DEBUG_PARSER=1` for token tracing
   - Impact: Eliminated 2 hours of token position debugging
   - Files: internal/parser/debug.go

✅ **Make Target** (Day 6, 3 min): Added `make update-golden` for parser test updates
   - Impact: Simplified golden file workflow
   - Files: Makefile

### Design Docs Created
📝 **M-DX10**: Parser AST Viewer Tool (estimated 2 hours)
   - Rationale: Spent 45 min manually inspecting AST structures
   - Expected ROI: Save ~30 min per future parser sprint
   - File: design_docs/planned/v0_4_0/m-dx10-ast-viewer.md

📝 **M-DX11**: Unified Error Message System (estimated 4 hours)
   - Rationale: Error messages inconsistent across lexer/parser/type checker
   - Expected ROI: Easier debugging for AI and humans
   - File: design_docs/planned/v0_4_0/m-dx11-error-system.md

### Considered But Deferred
💡 **REPL history search**: Nice-to-have, low impact vs effort
💡 **Syntax highlighting**: Human-focused, AILANG is AI-first
💡 **Auto-completion**: Deferred until reflection system complete

### Total DX Investment This Sprint
- Time spent: 18 min (quick wins)
- Time saved: ~3 hours (estimated, based on future sprint projections)
- Design docs: 2 (total estimated effort: 6 hours for future sprints)
- **Net impact**: Positive ROI even in current sprint
```

## Examples from Real Sprints

### M-DX9: Parser Developer Experience (October 2025)

**Applied:**
- Test helpers (`AssertNoErrors`, `AssertLiteralInt`, etc.) - 10 min
- Debug flag (`DEBUG_PARSER=1`) - 5 min
- Enhanced error messages with context - 15 min

**Impact:**
- Reduced parser test boilerplate by 40%
- Eliminated token position debugging time
- 30% faster parser development

**ROI:** 30 min investment → 2+ hours saved per parser sprint

### M-DX1: Builtin Developer Experience (September 2025)

**Applied:**
- Central registry system - 2 hours (larger improvement)
- Type Builder DSL - 1 hour
- Mock test context - 30 min

**Impact:**
- Reduced builtin development time from 7.5h to 2.5h per builtin
- 67% faster development
- Zero-friction testing with hermetic mocks

**ROI:** 3.5 hours investment → 5 hours saved per builtin (14x ROI after 3 builtins)

### M-TESTING: Test Infrastructure (November 2025)

**Applied:**
- API discovery with `make doc` - 5 min
- Common constructor reference in CLAUDE.md - 15 min

**Impact:**
- 80% faster API lookups (5-10 min → 30 sec)
- Prevented 23 API misuse errors

**ROI:** 20 min investment → 2+ hours saved on Day 7 alone

## Quick Reference Card

| Problem | Solution | Time | ROI |
|---------|----------|------|-----|
| Test boilerplate | Helper functions | 10 min | 4x |
| Hard to debug | Debug flag | 5 min | 9x |
| Manual workflow | Make target | 3 min | High |
| Confusing API | Documentation | 15 min | 7.5x |
| Poor error | Better message | 5 min | Medium |
| Verbose tests | Test utilities | 15 min | High |
