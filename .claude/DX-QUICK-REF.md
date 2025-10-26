# DX Quick Reference - Common Tasks

**Quick reference for AILANG development tasks.** Load when you need to know "how do I...?"

## I need to...

### ...look up an API (NEW v0.3.15!)

**⚠️ Use `make doc` FIRST - 80% faster than grep!**

```bash
# Find constructor signature
make doc PKG=internal/testing | grep "NewCollector"
# Output: func NewCollector(modulePath string) *Collector

# Find struct fields
make doc PKG=internal/ast | grep -A 20 "type FuncDecl"
# Output: Tests []*TestCase, Properties []*Property

# View full package API
make doc PKG=internal/elaborate
# Shows: func NewElaborator() *Elaborator (and all exports)

# Without arguments, shows usage help
make doc
```

**Common packages:**
- `internal/testing` - Test collection (M-TESTING)
- `internal/elaborate` - Surface AST → Core AST
- `internal/types` - Type checking and inference
- `internal/parser` - Parser (also see docs/guides/parser_development.md)
- `internal/eval` - Core AST evaluator
- `internal/builtins` - Builtin function registry

**Time savings**: ~30 sec vs 5-10 min manual grep (80% reduction)

**See also**: CLAUDE.md "Common API Patterns" section for common mistakes and gotchas

### ...build and test my changes
```bash
make quick-install                    # Fast reinstall after code changes
ailang run --caps IO examples/hello.ail   # Test it works
make test                             # Run all tests
make ci                               # Full CI locally
```

### ...update parser golden files (after parser changes)
```bash
make test-parser                      # Run parser tests (may fail if AST changed)
make test-parser-update               # Update golden files
git diff tests/golden/                # Review changes
make test-parser                      # Verify tests now pass
```

### ...verify stdlib changes don't break compatibility
```bash
make verify-stdlib                    # Should pass (checks SHA256 hashes)
# If fails and change was intentional:
make freeze-stdlib                    # Update golden hashes
make test-stdlib-canaries             # Verify std/io, std/net still work
```

### ...run evals to test AI code generation
```bash
# Development (3 cheap models: gpt5-mini, claude-haiku, gemini-flash)
make eval-suite

# Full suite (all 6 models)
make eval-suite FULL=true

# Baseline for release (REQUIRES version!)
make eval-baseline EVAL_VERSION=v0.3.15              # Dev models
make eval-baseline EVAL_VERSION=v0.3.15 FULL=true    # All models

# Resume interrupted run (v0.3.14+)
ailang eval-suite --full --skip-existing
```

### ...compare eval results
```bash
ailang eval-compare eval_results/baselines/v0.3.14 eval_results/baselines/v0.3.15
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=markdown
```

### ...validate a specific eval fix
```bash
ailang eval-validate fizzbuzz eval_results/baselines/v0.3.14
# Runs just fizzbuzz benchmark to verify fix works
```

### ...update the benchmark dashboard
```bash
# Update JSON (preserves history automatically!)
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=json
# ✅ Writes to docs/static/benchmarks/latest.json

# Update markdown
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=docusaurus 2>/dev/null > docs/docs/benchmarks/performance.md

# Verify JSON
jq -r '.version, .aggregates.finalSuccess' docs/static/benchmarks/latest.json

# Test locally
cd docs && npm run clear && npm start
```

### ...check code organization (file sizes)
```bash
make check-file-sizes                 # Fails if any file >800 lines (CI check)
make report-file-sizes                # Lists all files >500 lines
make codebase-health                  # Full codebase health metrics
make largest-files                    # Top 20 largest files

# To refactor large files:
# Use the codebase-organizer agent
```

### ...release a new version
```bash
# Use the release-manager skill (auto-invoked):
# User says: "Ready to release v0.3.15"

# The skill handles:
# - Pre-release checks (tests, linting, file sizes)
# - Version updates in docs
# - Git tagging and pushing
# - CI/CD monitoring
# - Release verification
```

### ...run post-release tasks (benchmarks, dashboard)
```bash
# Use the post-release skill (auto-invoked):
# User says: "Update benchmarks for v0.3.15"

# The skill handles:
# - Running eval baseline
# - Updating dashboard JSON (with history)
# - Updating dashboard markdown
# - Extracting CHANGELOG metrics
# - Moving design docs to implemented/
```

### ...update documentation
```bash
# Sync prompts to docs site
make sync-prompts                     # Copies prompts/ to docs/prompts/

# Generate LLM-friendly context
make generate-llms-txt                # Creates docs/static/llms.txt

# Docusaurus dev server
make docs-serve                       # Start at http://localhost:3000

# Clear Docusaurus cache (for webpack errors)
make docs-clean                       # Clear cache
make docs-restart                     # Clear cache + restart

# Nuclear option if webpack still broken:
cd docs && npm run clear && rm -rf .docusaurus build && npm start
```

### ...fix linting/formatting issues
```bash
make fmt                              # Auto-format Go code
make fmt-check                        # Check if formatted (CI)
make lint                             # Run golangci-lint
make vet                              # Run go vet
```

### ...work on a sprint
```bash
# Use the sprint-executor skill (auto-invoked):
# User says: "Execute the sprint plan in design_docs/20251019/M-S1.md"

# The skill handles:
# - Validating prerequisites (tests pass, lint clean)
# - Creating TodoWrite tasks for all milestones
# - Executing with test-driven development
# - Running checkpoints after each milestone
# - Updating CHANGELOG and sprint plan progressively
# - Pausing after each milestone for review

# Manual checkpoint during sprint:
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh "Milestone name"

# Manual prerequisites check:
.claude/skills/sprint-executor/scripts/validate_prerequisites.sh
```

### ...create a new skill
```bash
# Use the skill-builder skill (auto-invoked):
# User says: "Create a skill for managing database migrations"

# See also: .claude/skills/SKILLS_GUIDE.md
```

### ...create a design document
```bash
# Use the design-doc-creator skill (auto-invoked):
# User says: "Create a design doc for the module system"

# Manual:
# 1. Create in design_docs/planned/vX_Y_Z/
# 2. After implementation, move to design_docs/implemented/vX_Y_Z/
```

### ...update import error goldens
```bash
make regen-import-error-goldens       # Regenerate goldens
git diff tests/golden/import_errors/  # Review changes
make test-import-errors               # Verify tests pass
```

### ...check test coverage
```bash
make test-coverage                    # Full coverage report (coverage.html)
make test-coverage-badge              # Quick coverage check (shows %)

# Specialized coverage:
make cover-parser                     # Parser coverage HTML
make cover-lexer                      # Lexer coverage HTML
make cover-all-packages               # All packages coverage
```

### ...verify examples still work
```bash
make verify-examples                  # Verify all working examples
make verify-examples-all              # Verify all (including broken)
make update-readme                    # Update README with example status
make flag-broken                      # Add warning headers to broken examples
```

### ...debug builtin registry issues
```bash
make doctor                           # Validate builtin registry
ailang doctor builtins                # Same as above
ailang builtins list                  # List all builtins
ailang builtins list --by-module      # Group by module
ailang builtins list --by-effect      # Group by effect
```

### ...run specialized tests
```bash
# Type system
make test-row-properties              # Row unification properties
make test-golden-types                # Builtin type snapshots

# Builtins
make test-builtin-consistency         # Builtin three-way parity
make test-builtin-freeze              # Freeze builtin type signatures

# REPL
make test-repl-smoke                  # REPL smoke tests (:type command)

# Regression
make test-regression-guards           # Regression guard tests
make test-recursion                   # Recursion limits

# Operators
make test-lowering                    # Operator lowering (desugaring)
make verify-lowering                  # Verify operator lowering
make test-operator-assertions         # Test operator assertions
```

### ...build WASM for browser REPL
```bash
make build-wasm                       # Builds to docs/static/wasm/ailang.wasm
```

### ...fuzz test the parser
```bash
make fuzz-parser                      # Short fuzz test
make fuzz-parser-long                 # Long-running fuzz test
```

---

## Common Workflows

### Standard Development Cycle
```bash
# 1. Make code changes
# 2. Reinstall
make quick-install

# 3. Test changes
ailang run --caps IO examples/my_test.ail
make test

# 4. Lint
make lint

# 5. Check file sizes
make check-file-sizes
```

### After Parser/Type System Changes
```bash
# 1. Run tests
make test-parser

# 2. Update goldens if needed
make test-parser-update

# 3. Review changes
git diff tests/golden/

# 4. Verify related systems
make test-lowering                    # Operator lowering
make test-row-properties              # Row unification
make test-golden-types                # Builtin types
```

### After Stdlib Changes
```bash
# 1. Verify no breaking changes
make verify-stdlib

# 2. If intentional breaking change
make freeze-stdlib

# 3. Run health checks
make test-stdlib-canaries

# 4. Test specific modules
ailang iface stdlib/std/io.ail        # Check interface
```

### Before Committing
```bash
make test                             # All tests must pass
make lint                             # All linting must pass
make check-file-sizes                 # No files >800 lines
make verify-stdlib                    # Stdlib unchanged (or frozen)
```

### Before Releasing
```bash
make ci-strict                        # Full CI with A2 gates
make check-file-sizes                 # No files >800 lines
make verify-stdlib                    # Stdlib unchanged (or frozen)
make test-coverage-badge              # Coverage check

# Or use the release-manager skill:
# "Ready to release v0.3.15"
```

### After Releasing
```bash
# Use the post-release skill:
# "Update benchmarks for v0.3.15"

# Or manually:
make eval-baseline EVAL_VERSION=v0.3.15 FULL=true
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=json
ailang eval-report eval_results/baselines/v0.3.15 v0.3.15 --format=docusaurus > docs/docs/benchmarks/performance.md
cd docs && npm run clear && npm start
```

---

## Tool Discovery

| Category | Tool | Quick Command |
|----------|------|---------------|
| **API Lookup** | Package API | `make doc PKG=internal/testing` |
| | Constructor signature | `make doc PKG=<pkg> \| grep "New"` |
| | Struct fields | `make doc PKG=<pkg> \| grep -A 20 "type Foo"` |
| **Build** | Build locally | `make build` |
| | Install to system | `make quick-install` |
| **Test** | All tests | `make test` |
| | With coverage | `make test-coverage` |
| | Full CI | `make ci` or `make ci-strict` |
| **Parser** | Parser tests | `make test-parser` |
| | Update goldens | `make test-parser-update` |
| **Stdlib** | Verify unchanged | `make verify-stdlib` |
| | Freeze after change | `make freeze-stdlib` |
| **Eval** | Dev models | `make eval-suite` |
| | All models | `make eval-suite FULL=true` |
| | Baseline | `make eval-baseline EVAL_VERSION=vX.Y.Z` |
| | Compare | `ailang eval-compare <dir1> <dir2>` |
| | Validate fix | `ailang eval-validate <bench> <baseline>` |
| **Dashboard** | Update JSON | `ailang eval-report <dir> <ver> --format=json` |
| | Update markdown | `ailang eval-report <dir> <ver> --format=docusaurus > docs/docs/benchmarks/performance.md` |
| **Quality** | Format | `make fmt` |
| | Lint | `make lint` |
| | File sizes | `make check-file-sizes` |
| **Docs** | Dev server | `make docs-serve` |
| | Clear cache | `make docs-clean` |
| | Sync prompts | `make sync-prompts` |
| **Skills** | Release | `release-manager` skill |
| | Post-release | `post-release` skill |
| | Sprint | `sprint-executor` skill |
| | Create skill | `skill-builder` skill |

---

## Emergency Recovery

### Tests Failing
```bash
make test                             # See what's failing
# Fix the issue, then:
make quick-install && make test
```

### Linting Failing
```bash
make fmt                              # Auto-fix formatting
make lint                             # Check what's left
# Fix manually, then verify:
make lint
```

### Stdlib Accidentally Changed
```bash
make verify-stdlib                    # See what changed
# If unintentional:
git restore stdlib/std/*.ail
# If intentional:
make freeze-stdlib
```

### Golden Files Out of Sync
```bash
# Parser goldens
make test-parser-update
git diff tests/golden/

# Import error goldens
make regen-import-error-goldens
git diff tests/golden/import_errors/

# Stdlib goldens
make freeze-stdlib
git diff .stdlib-golden/
```

### Docusaurus Webpack Errors
```bash
# Try cache clear first
make docs-clean && make docs-restart

# Nuclear option
cd docs && npm run clear && rm -rf .docusaurus build && npm start
```

### Eval Results Accidentally Overwritten
```bash
# ⚠️ Prevention: Always run all models in ONE command
ailang eval-suite --models model1,model2,model3

# Or use different output directories:
ailang eval-suite --models gpt5 --output eval_results/gpt5_only
```

---

---

## Parser Development DX (M-DX9)

### Common Parser Gotchas

**Token position bugs (30% of dev time waste!):**
```go
// ❌ WRONG - parseExpression leaves you AT last token, not AFTER
expr := p.parseExpression(LOWEST)
if p.curTokenIs(lexer.COMMA) { ... }  // BUG!

// ✅ CORRECT - Call nextToken after parsing
expr := p.parseExpression(LOWEST)
p.nextToken()  // Move to next token
if p.curTokenIs(lexer.COMMA) { ... }  // Works!
```

**Test error printing:**
```go
// ❌ WRONG - Errors hidden (t.Fatalf stops before loop)
if len(p.Errors()) != 0 {
    t.Fatalf("parser had %d errors:", len(p.Errors()))
    for _, err := range p.Errors() { ... }  // Never executes!
}

// ✅ CORRECT - Print BEFORE Fatalf
if len(p.Errors()) != 0 {
    for _, err := range p.Errors() {
        t.Errorf("  %s", err)
    }
    t.Fatalf("parser had %d errors", len(p.Errors()))
}

// ✅ BETTER - Use helper
AssertNoErrors(t, p)  // From internal/parser/test_helpers.go
```

**AST type gotchas:**
```go
// ❌ WRONG type names
expr.(*ast.Variable)      // Use ast.Identifier
expr.(*ast.IntLiteral)    // Use ast.Literal
lit.Value.(int)           // Lexer returns int64!

// ✅ CORRECT
expr.(*ast.Identifier)
lit := expr.(*ast.Literal)
val := lit.Value.(int64)  // Not int!
```

### Parser Test Helpers

**Location:** `internal/parser/test_helpers.go`

```go
// Basic assertions
AssertNoErrors(t, p)
AssertTokenPosition(t, p, lexer.INT, lexer.COMMA)

// Literal assertions
AssertLiteralInt(t, expr, 42)
AssertLiteralString(t, expr, "hello")

// Structure assertions
AssertIdentifier(t, expr, "x")
fn := AssertFuncDecl(t, file.Decls[0], "factorial")
AssertSimpleType(t, fn.ReturnType, "int")
```

### Debug Flags (v0.3.15+)

```bash
# Show token flow through parser (✅ IMPLEMENTED)
DEBUG_PARSER=1 ailang run test.ail
# Output: [ENTER parseExpression] cur=INT(42) peek=PLUS(+)
#         [EXIT parseExpression] cur=INT(42) peek=PLUS(+)

# Catch incomplete switch statements (v0.4.1+)
DEBUG_STRICT=1 ailang run test.ail

# Debug monomorphization (v0.4.0+)
DEBUG_MONO_VERBOSE=1 ailang run --debug-compile test.ail
```

### Quick API & Type Lookup

**⚠️ NEW (v0.3.15): Use `make doc` for 80% faster API discovery!**

```bash
# Look up package API (constructors, types, functions)
make doc PKG=internal/testing
# Shows: func NewCollector(modulePath string) *Collector

make doc PKG=internal/elaborate
# Shows: func NewElaborator() *Elaborator (no arguments!)

make doc PKG=internal/ast | grep -A 20 "type FuncDecl"
# Shows struct fields: Tests []*TestCase, Properties []*Property

# Without arguments, shows usage examples
make doc
```

**Legacy grep methods (slower, but still work):**
```bash
# Find AST types
grep "^type.*struct" internal/ast/ast.go | head -20

# Check if keyword exists
grep -i "forall" internal/lexer/token.go

# Find constructors
grep "^func New" internal/testing/collector.go
```

**Time savings**: `make doc` takes ~30 sec vs manual grep ~5-10 min (80% reduction)

**See also:**
- CLAUDE.md "Parser Developer Experience Guide" section
- CLAUDE.md "Common API Patterns (M-TESTING Days 3-4 Learnings)" section

---

## See Also

- **CLAUDE.md** - Comprehensive development guide (detailed workflows, principles, examples)
  - **Parser Developer Experience Guide** - Full parser DX documentation (M-DX9)
- **sprint-executor/resources/developer_tools.md** - Complete tool reference (all make targets, detailed descriptions)
- **design_docs/planned/v0_3_16/dx-tools-documentation-audit.md** - Full audit of all tools
- **.claude/skills/README.md** - Skills overview and usage
- **docs/docs/guides/development.md** - Development guide for humans
- **docs/docs/guides/evaluation/** - Detailed eval system documentation
- **internal/parser/test_helpers.go** - Parser test helper functions (M-DX9)

---

**Last updated**: 2025-10-26
- M-DX9: Parser DX improvements (test helpers, DEBUG_PARSER, comprehensive guide)
- M-TESTING DX: API discovery (`make doc`), common API patterns, error printing fixes

**Purpose**: Quick task-based reference for AILANG developers (human and AI)
