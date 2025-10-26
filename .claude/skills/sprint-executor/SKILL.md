---
name: AILANG Sprint Executor
description: Execute approved sprint plans with test-driven development, continuous linting, progress tracking, and pause points. Use when user says "execute sprint", "start sprint", or wants to implement an approved sprint plan.
---

# AILANG Sprint Executor

Execute an approved sprint plan with continuous progress tracking, testing, and documentation updates.

## Quick Start

**Most common usage:**
```bash
# User says: "Execute the sprint plan in design_docs/20251019/M-S1.md"
# This skill will:
# 1. Validate prerequisites (tests pass, linting clean)
# 2. Create TodoWrite tasks for all milestones
# 3. Execute each milestone with test-driven development
# 4. Run checkpoint after each milestone (tests + lint)
# 5. Update CHANGELOG and sprint plan progressively
# 6. Pause after each milestone for user review
```

## When to Use This Skill

Invoke this skill when:
- User says "execute sprint", "start sprint", "begin implementation"
- User has an approved sprint plan ready to implement
- User wants guided execution with built-in quality checks
- User needs progress tracking and pause points

## Core Principles

1. **Test-Driven**: All code must pass tests before moving to next milestone
2. **Lint-Clean**: All code must pass linting before moving to next milestone
3. **Document as You Go**: Update CHANGELOG.md and sprint plan progressively
4. **Pause for Breath**: Stop at natural breakpoints for review and approval
5. **Track Everything**: Use TodoWrite to maintain visible progress

## Documentation URLs

When adding error messages, help text, or documentation links in code:

**Website**: https://sunholo-data.github.io/ailang/

**Documentation Source**: The website documentation lives in this repo at `docs/`
- Markdown files: `docs/docs/` (guides, reference, etc.)
- Static assets: `docs/static/`
- Docusaurus config: `docs/docusaurus.config.js`

**Common Documentation Paths**:
- Language syntax: `/docs/reference/language-syntax`
- Module system: `/docs/guides/module_execution`
- Getting started: `/docs/guides/getting-started`
- REPL guide: `/docs/guides/getting-started#repl`
- Implementation status: `/docs/reference/implementation-status`

**Full URL Example**:
```
https://sunholo-data.github.io/ailang/docs/reference/language-syntax
```

**Best Practices**:
- Check that documentation URLs actually exist before using them in error messages or help text
- Look in `docs/docs/` to verify the file exists locally
- Use `ls docs/docs/reference/` or `ls docs/docs/guides/` to find available pages

## Available Scripts

### `scripts/validate_prerequisites.sh`
Validate prerequisites before starting sprint execution.

**Usage:**
```bash
.claude/skills/sprint-executor/scripts/validate_prerequisites.sh
```

**Output:**
```
Validating sprint prerequisites...

1/4 Checking working directory...
  ✓ Working directory clean

2/4 Checking current branch...
  ✓ On branch: dev

3/4 Running tests...
  ✓ All tests pass

4/4 Running linter...
  ✓ Linting passes

✓ All prerequisites validated!
Ready to start sprint execution.
```

**Exit codes:**
- `0` - All prerequisites pass
- `1` - One or more prerequisites fail

### `scripts/milestone_checkpoint.sh <milestone_name>`
Run checkpoint after completing a milestone.

**Usage:**
```bash
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh "M-S1.1: Parser foundation"
```

**Output:**
```
Running checkpoint for: M-S1.1: Parser foundation

1/3 Running tests...
  ✓ Tests pass

2/3 Running linter...
  ✓ Linting passes

3/3 Files changed in this milestone...
 internal/parser/parser.go   | 125 ++++++++++++++++++
 internal/parser/parser_test.go | 89 +++++++++++++
 2 files changed, 214 insertions(+)

✓ Milestone checkpoint passed!
Ready to proceed to next milestone.
```

## Execution Flow

### Phase 1: Initialize Sprint

#### 1. Read Sprint Plan
- Parse sprint plan document (e.g., `design_docs/20251019/M-S1.md`)
- Extract all milestones and tasks
- Note dependencies and acceptance criteria
- Identify estimated LOC and duration

#### 2. Validate Prerequisites

**Use the validation script:**
```bash
.claude/skills/sprint-executor/scripts/validate_prerequisites.sh
```

**Manual checks:**
- Working directory clean: `git status --short`
- Current tests pass: `make test`
- Current linting passes: `make lint`
- On correct branch (usually `dev`)

**If validation fails:**
- Fix issues before starting
- Don't proceed with dirty working directory
- Don't start with failing tests or linting

#### 3. Create Todo List

**Use TodoWrite to create tasks:**
- Extract all milestones from sprint plan
- Mark first milestone as `in_progress`
- Keep remaining tasks as `pending`
- This provides real-time progress visibility

#### 4. Initial Status Update
- Update sprint plan with "🔄 In Progress" status
- Add start timestamp
- Commit sprint plan update (optional)

### Phase 2: Execute Milestones

**For each milestone in the sprint:**

#### Step 1: Pre-Implementation
- Mark milestone as `in_progress` in TodoWrite
- Review milestone goals and acceptance criteria
- Identify files to create/modify
- Estimate LOC if not already specified

#### Step 2: Implement
- Write implementation code following the task breakdown
- Follow design patterns from sprint plan
- Add inline comments for complex logic
- Keep functions small and focused

#### Step 3: Write Tests
- Create/update test files (*_test.go)
- Aim for comprehensive coverage (all acceptance criteria)
- Include edge cases and error conditions
- Test both success and failure paths

**Parser tests (M-DX9):**
- Use helpers from `internal/parser/test_helpers.go`:
  - `AssertNoErrors(t, p)` - Check for parser errors
  - `AssertLiteralInt(t, expr, 42)` - Check integer literals
  - `AssertIdentifier(t, expr, "name")` - Check identifiers
  - `AssertFuncCall(t, expr)` - Check function calls
  - See full list in [internal/parser/test_helpers.go](internal/parser/test_helpers.go)
- Reference [docs/guides/parser_development.md](docs/guides/parser_development.md) for test patterns
- Common gotchas documented in [internal/ast/ast.go](internal/ast/ast.go) (e.g., int64 vs int)

#### Step 4: Verify Quality

**Run checkpoint script:**
```bash
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh "Milestone name"
```

**Manual verification:**
```bash
make test  # MUST PASS
make lint  # MUST PASS
```

**CRITICAL**: If tests or linting fail, fix immediately before proceeding.

#### Step 5: Update Documentation

**Update CHANGELOG.md:**
- What was implemented
- LOC counts (implementation + tests)
- Key design decisions
- Files modified/created

**Update sprint plan:**
- Mark milestone as ✅
- Add actual LOC vs estimated
- Note any deviations from plan

#### Step 6: Pause for Breath

**After each milestone:**
- Show summary of what was completed
- Show current sprint progress (X of Y milestones done)
- Show velocity (LOC/day vs planned)
- Ask user: "Ready to continue to next milestone?" or "Need to review/adjust?"
- If user says "pause" or "stop", save current state and exit gracefully

### Phase 3: Finalize Sprint

**When all milestones are complete:**

#### 1. Final Testing
```bash
make test                # Full test suite
make lint                # All linting
make test-coverage-badge # Coverage check
```

#### 2. Documentation Review
- Verify CHANGELOG.md is complete
- Verify sprint plan shows all milestones as ✅
- Update sprint plan with final metrics:
  - Total LOC (actual vs estimated)
  - Total time (actual vs estimated)
  - Velocity achieved
  - Test coverage achieved
  - Any deviations from plan

#### 3. Final Commit
```bash
git commit -m "Complete sprint: <sprint-name>

Milestones completed:
- <Milestone 1>: <LOC>
- <Milestone 2>: <LOC>

Total: <actual-LOC> LOC in <actual-time>
Velocity: <LOC/day>
Test coverage: <percentage>"
```

#### 4. Summary Report
- Show sprint completion summary
- Compare planned vs actual (LOC, time, milestones)
- Highlight any issues or deviations
- Suggest next steps (new sprint, release, etc.)

#### 5. Identify Bumps
- What could AILANG do better to make a smoother coding sprint?
- Is it worth adding a new design doc to help ease how we make AILANG?

## Key Features

### Continuous Testing
- Run `make test` after every file change
- Never proceed if tests fail
- Show test output for visibility
- Track test count increase

**Parser test best practices (M-DX9):**
- Use test helpers from `internal/parser/test_helpers.go` for cleaner assertions
- Print errors BEFORE `t.Fatalf()` or use `AssertNoErrors(t, p)` helper
- Reference [docs/guides/parser_development.md](docs/guides/parser_development.md) for patterns
- See [internal/ast/ast.go](internal/ast/ast.go) comments for AST usage examples

### Continuous Linting
- Run `make lint` after implementation
- Fix linting issues immediately
- Use `make fmt` for formatting issues
- Verify with `make fmt-check`

### Progress Tracking
- TodoWrite shows real-time progress
- Sprint plan updated at each milestone
- CHANGELOG.md grows incrementally
- Git commits create audit trail

### Pause Points
- After each milestone completion
- When tests fail (fix before continuing)
- When linting fails (fix before continuing)
- When user requests "pause"
- When encountering unexpected issues

### Error Handling
- **If tests fail**: Show output, ask how to fix, don't proceed
- **If linting fails**: Show output, ask how to fix, don't proceed
- **If implementation unclear**: Ask for clarification, don't guess
- **If milestone takes much longer than estimated**: Pause and reassess

**Parser debugging (M-DX9):**
- Use `DEBUG_PARSER=1 ailang run test.ail` to trace token flow
- Check [docs/guides/parser_development.md](docs/guides/parser_development.md) for troubleshooting
- Common issues documented in CLAUDE.md "Parser Developer Experience Guide" section

## Resources

### Parser Development Tools (M-DX9)

**For parser-related sprints, use these M-DX9 tools:**

1. **Comprehensive Guide**: [docs/guides/parser_development.md](../../docs/guides/parser_development.md)
   - Quick start with example (adding new expression type)
   - Token position convention (AT vs AFTER) - prevents 30% of bugs
   - Common AST types reference
   - Parser patterns (delimited lists, optional sections, precedence)
   - Test infrastructure guide
   - Debug tools reference
   - Common gotchas and troubleshooting

2. **Test Helpers**: [internal/parser/test_helpers.go](../../internal/parser/test_helpers.go)
   - 15 helper functions for cleaner parser tests
   - `AssertNoErrors(t, p)` - Check for parser errors
   - `AssertLiteralInt/String/Bool/Float(t, expr, value)` - Check literals
   - `AssertIdentifier(t, expr, name)` - Check identifiers
   - `AssertFuncCall/List/ListLength(t, expr)` - Check structures
   - `AssertDeclCount/FuncDecl/TypeDecl(t, file, ...)` - Check declarations
   - All helpers call `t.Helper()` for clean stack traces

3. **Debug Tooling**: [internal/parser/debug.go](../../internal/parser/debug.go)
   - `DEBUG_PARSER=1` environment variable for token flow tracing
   - Shows ENTER/EXIT with cur/peek tokens for parseExpression, parseType
   - Zero overhead when disabled
   - Example: `DEBUG_PARSER=1 ailang run test.ail`

4. **AST Usage Examples**: [internal/ast/ast.go](../../internal/ast/ast.go)
   - Comprehensive documentation on 6 major AST types
   - Usage examples for Identifier, Literal, Lambda, FuncCall, List, FuncDecl
   - ⚠️ **CRITICAL**: int64 vs int gotcha prominently documented
   - Common parser patterns for each type

5. **Quick Reference**: CLAUDE.md "Parser Developer Experience Guide" section
   - Token position convention
   - Common AST types
   - Quick token lookup
   - Parsing optional sections pattern
   - Test error printing pattern

**When to use these tools:**
- ✅ Any sprint touching `internal/parser/` code
- ✅ Any sprint adding new expression/statement/type syntax
- ✅ Any sprint modifying AST nodes
- ✅ When encountering token position bugs
- ✅ When writing parser tests

**Impact**: M-DX9 tools reduce parser development time by 30% by eliminating token position debugging overhead.

### Common API Patterns (M-TESTING Learnings)

**⚠️ ALWAYS check `make doc PKG=<package>` before grepping or guessing APIs!**

#### Quick API Lookup

```bash
# Find constructor signatures
make doc PKG=internal/testing | grep "NewCollector"
# Output: func NewCollector(modulePath string) *Collector

# Find struct fields
make doc PKG=internal/ast | grep -A 20 "type FuncDecl"
# Shows: Tests []*TestCase, Properties []*Property
```

#### Common Constructors

| Package | Constructor | Signature | Notes |
|---------|-------------|-----------|-------|
| `internal/testing` | `NewCollector(path)` | Takes module path | M-TESTING |
| `internal/elaborate` | `NewElaborator()` | No arguments | Surface → Core |
| `internal/types` | `NewTypeChecker(core, imports)` | Takes Core prog + imports | Type inference |
| `internal/link` | `NewLinker()` | No arguments | Dictionary linking |
| `internal/parser` | `New(lexer)` | Takes lexer instance | Parser |
| `internal/eval` | `NewEvaluator(ctx)` | Takes EffContext | Core evaluator |

#### Common API Mistakes

**Test Collection (M-TESTING):**
```go
// ✅ CORRECT
collector := testing.NewCollector("module/path")
suite := collector.Collect(file)
for _, test := range suite.Tests { ... }  // Tests is the slice!

// ❌ WRONG
collector := testing.NewCollector(file, modulePath)  // Wrong arg order!
for _, test := range suite.Tests.Cases { ... }      // No .Cases field!
```

**String Formatting:**
```go
// ✅ CORRECT
name := fmt.Sprintf("test_%d", i+1)

// ❌ WRONG - Produces "\x01" not "1"!
name := "test_" + string(rune(i+1))  // BUG!
```

**Field Access:**
```go
// ✅ CORRECT
funcDecl.Tests        // []*ast.TestCase
funcDecl.Properties   // []*ast.Property

// ❌ WRONG
funcDecl.InlineTests  // Doesn't exist! Use .Tests
```

#### API Discovery Workflow

1. **`make doc PKG=<package>`** (~30 sec) ← Start here!
2. Check source file if you know location (`grep "^func New" file.go`)
3. Check test files for usage examples (`grep "NewCollector" *_test.go`)
4. Read [docs/guides/](../../docs/guides/) for complex workflows

**Time savings**: 80% reduction (5-10 min → 30 sec per lookup)

**Full reference**: See CLAUDE.md "Common API Patterns" section

### Developer Tools Reference
See [`resources/developer_tools.md`](resources/developer_tools.md) for comprehensive reference of all available make targets, ailang commands, scripts, and workflows. Load this when you need to:
- Know which test targets to use
- Update golden files after parser changes
- Verify stdlib changes
- Run evals or compare baselines
- Troubleshoot build/test/lint issues
- Find the right tool for any development task

### Milestone Checklist
See [`resources/milestone_checklist.md`](resources/milestone_checklist.md) for complete step-by-step checklist per milestone.

## Prerequisites

- Working directory should be clean (or have only sprint-related changes)
- Current branch should be `dev` (or specified in sprint plan)
- All existing tests must pass before starting
- All existing linting must pass before starting
- Sprint plan must be approved and documented

## Failure Recovery

### If Tests Fail During Sprint
1. Show test failure output
2. Ask user: "Tests failing. Options: (a) fix now, (b) revert change, (c) pause sprint"
3. Don't proceed until tests pass

### If Linting Fails During Sprint
1. Show linting output
2. Try auto-fix: `make fmt`
3. If still failing, ask user for guidance
4. Don't proceed until linting passes

### If Implementation Blocked
1. Show what's blocking progress
2. Ask user for guidance or clarification
3. Consider simplifying the approach
4. Document the blocker in sprint plan

### If Velocity Much Lower Than Expected
1. Pause and reassess after 2-3 milestones
2. Calculate actual velocity
3. Propose: (a) continue as-is, (b) reduce scope, (c) extend timeline
4. Update sprint plan with revised estimates

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + execution workflow)
2. **Execute as needed**: Scripts in `scripts/` directory (validation, checkpoints)
3. **Load on demand**: `resources/milestone_checklist.md` (detailed checklist)

Scripts execute without loading into context window, saving tokens while ensuring quality.

## Notes

- This skill is long-running - expect it to take hours or days
- Pause points are built in - you're not locked into finishing
- Sprint plan is the source of truth - but reality may require adjustments
- Git commits create a reversible audit trail
- TodoWrite provides real-time visibility into progress
- Test-driven development is non-negotiable - tests must pass
