# File Organization Guide

**AILANG is designed to be maintained by AI assistants. Keep files small and focused.**

## File Size Guidelines

**Target file sizes:**
- **Sweet spot**: 200-500 lines per file
- **Acceptable**: 500-800 lines
- **Problematic**: 800-1200 lines (consider splitting)
- **Critical**: 1200+ lines (MUST split before adding features)

**Why small files matter for AI:**
- Fits in AI context window (AI can see the whole file at once)
- Single responsibility principle naturally enforced
- Easy to understand the full structure in one read
- Reduces merge conflicts
- Enables better testing isolation

**Check file sizes:**
```bash
make check-file-sizes    # Fails CI if any file >800 lines
make report-file-sizes   # Shows all files >500 lines
wc -l internal/path/file.go  # Check specific file
```

## Current Technical Debt

**Check current status:**
```bash
make report-file-sizes    # Detailed report of files >500 lines
make codebase-health      # Overall codebase metrics
make largest-files        # Top 20 largest files
```

As of October 2025, ~10 files exceed the 800 line limit (out of 183 total). Run `make report-file-sizes` for the current list.

**Before modifying large files:**
1. Check if splitting is needed first
2. Run tests before/after: `make test`
3. Use the `codebase-organizer` agent for safe refactoring

## File Organization Patterns

### Pattern 1: One Concept Per File

```
❌ BAD: Everything in one file
internal/parser/parser.go (2518 lines)
  - Expression parsing
  - Statement parsing
  - Type parsing
  - Pattern parsing
  - Module parsing

✅ GOOD: Split by responsibility
internal/parser/
  ├── parser.go (200 lines)         # Main struct, entry points, package docs
  ├── expressions.go (300 lines)    # parseExpression, parseLambda, parseCall
  ├── statements.go (250 lines)     # parseLetDecl, parseFuncDecl, parseType
  ├── types.go (200 lines)          # parseType, parseEffects, parseTypeParams
  ├── patterns.go (280 lines)       # parsePattern, parseConstructor
  ├── modules.go (150 lines)        # parseModule, parseImport, parseExport
  └── helpers.go (140 lines)        # parseParams, parseBlock, utility functions
```

### Pattern 2: Main File as Table of Contents

Every package should have a main file (usually `pkg.go` or matching package name) that serves as navigation:

```go
// internal/parser/parser.go (200 lines max)
package parser

// Package parser implements AILANG source code parsing.
//
// # Architecture
//
// The parser is split into several files by responsibility:
//   - parser.go: Main Parser struct and entry points (THIS FILE)
//   - expressions.go: Expression parsing (literals, lambdas, calls, etc.)
//   - statements.go: Top-level declarations (func, type, let)
//   - types.go: Type annotation parsing
//   - patterns.go: Pattern matching syntax
//   - modules.go: Module system (import/export)
//
// # Usage
//
//   p := parser.New(lexer)
//   file, err := p.Parse()
//
// # See Also
//
//   - internal/ast: AST node definitions
//   - internal/lexer: Token generation
//   - docs/parser/README.md: Detailed parser documentation

// Parser is the main entry point for parsing AILANG source code.
type Parser struct { /* ... */ }

// Parse parses a complete AILANG source file.
// Implementation delegates to parseFile() in statements.go.
func (p *Parser) Parse() (*ast.File, error) { /* ... */ }
```

### Pattern 3: Tests Next to Implementation

```
✅ GOOD: Focused test files
internal/parser/
  ├── expressions.go
  ├── expressions_test.go (300 lines focused tests)
  ├── statements.go
  ├── statements_test.go (250 lines focused tests)
  └── integration_test.go (end-to-end tests)

❌ BAD: One giant test file
  └── parser_test.go (5000 lines)
```

### Pattern 4: Clear File Naming

File names should match the main functions they contain:

```
✅ GOOD:
expressions.go → parseExpression(), parseCall(), parseLambda()
statements.go  → parseLetDecl(), parseFuncDecl(), parseTypeDecl()
patterns.go    → parsePattern(), parseConstructor()

❌ BAD:
parse_stuff.go → everything mixed together
utils.go       → vague, no clear responsibility
```

## Adding New Features (File Size Rules)

**Before adding any new feature to a file:**

```bash
# 1. Check current file size
wc -l internal/types/typechecker_core.go
# Output: 2736 lines

# 2. If >800 lines, STOP and split first
# 3. If 500-800 lines, consider if new feature pushes it over 800
# 4. If <500 lines, proceed normally

# 5. After changes, verify size
wc -l internal/types/typechecker_core.go
make check-file-sizes  # Fails if >800 lines
```

**Splitting workflow:**

```bash
# Option 1: Use the codebase-organizer agent (recommended)
# This agent safely refactors files while ensuring tests pass

# Option 2: Manual split (if you understand the code deeply)
make test                    # Baseline - all tests pass
# ... split files ...
make test                    # Verify - all tests still pass
git add internal/types/*.go
git commit -m "Split typechecker_core.go into 8 files (AI-friendly)"
```

## Package Documentation Standards

Every package with >3 files MUST have a README.md:

```markdown
# internal/parser

Parser for AILANG source code.

## Files

- `parser.go` - Main Parser struct, entry points
- `expressions.go` - Expression parsing: literals, lambdas, calls, operators
- `statements.go` - Declarations: func, type, let, import, export
- `types.go` - Type annotations: simple types, effects, type parameters
- `patterns.go` - Pattern matching: constructors, literals, wildcards, guards
- `modules.go` - Module system: module declarations, import resolution
- `helpers.go` - Shared utilities: parameter parsing, block parsing

## Entry Points

- `Parse()` → `parseFile()` in statements.go
- `parseExpression()` in expressions.go
- `parseType()` in types.go
- `parsePattern()` in patterns.go

## Cross-references

- Consumes: `internal/lexer` (tokens)
- Produces: `internal/ast` (syntax tree)
- Used by: `internal/pipeline`, `internal/repl`
```

## Automated Code Organization

**Use the codebase-organizer agent** for safe refactoring:

The `codebase-organizer` agent is available in `.claude/agents/codebase-organizer.md`. It:
- Monitors file sizes across the codebase
- Identifies files that need splitting
- Safely refactors large files into smaller, focused modules
- Ensures all tests pass before/after refactoring
- Maintains git history and commit hygiene

**Example usage:**
```bash
# Ask Claude to invoke the agent:
"Please use the codebase-organizer agent to check for files that need splitting"

# Or for specific refactoring:
"Use the codebase-organizer agent to split internal/parser/parser.go"
```

## Measuring Success

```bash
# CI checks (automatically run on PRs)
make check-file-sizes     # Fails if any file >800 lines

# Status reports
make report-file-sizes    # Lists all files >500 lines
make codebase-health      # Full codebase metrics
```

**Goal metrics:**
- 0 files over 800 lines ✅
- <5 files between 500-800 lines ⚠️
- Average file size: 300-400 lines 🎯

## See Also

- **CLAUDE.md** - Quick reference and navigation
- **CONTRIBUTING.md** - Development workflow
- **codebase-organizer agent** - Automated refactoring
