# Common API Patterns

**Quick reference for AILANG internal APIs learned from M-TESTING and other sprints.**

## Key Principle

**⚠️ ALWAYS check `make doc PKG=<package>` before grepping or guessing APIs!**

API discovery with `make doc` is 80% faster than manual searching (5-10 min → 30 sec per lookup).

## Quick API Lookup

```bash
# Find constructor signatures
make doc PKG=internal/testing | grep "NewCollector"
# Output: func NewCollector(modulePath string) *Collector

# Find struct fields
make doc PKG=internal/ast | grep -A 20 "type FuncDecl"
# Shows: Tests []*TestCase, Properties []*Property
```

## Common Constructors

| Package | Constructor | Signature | Notes |
|---------|-------------|-----------|-------|
| `internal/testing` | `NewCollector(path)` | Takes module path | M-TESTING |
| `internal/elaborate` | `NewElaborator()` | No arguments | Surface → Core |
| `internal/types` | `NewTypeChecker(core, imports)` | Takes Core prog + imports | Type inference |
| `internal/link` | `NewLinker()` | No arguments | Dictionary linking |
| `internal/parser` | `New(lexer)` | Takes lexer instance | Parser |
| `internal/eval` | `NewEvaluator(ctx)` | Takes EffContext | Core evaluator |

## Common API Mistakes

### Test Collection (M-TESTING)

```go
// ✅ CORRECT
collector := testing.NewCollector("module/path")
suite := collector.Collect(file)
for _, test := range suite.Tests { ... }  // Tests is the slice!

// ❌ WRONG
collector := testing.NewCollector(file, modulePath)  // Wrong arg order!
for _, test := range suite.Tests.Cases { ... }      // No .Cases field!
```

### String Formatting

```go
// ✅ CORRECT
name := fmt.Sprintf("test_%d", i+1)

// ❌ WRONG - Produces "\x01" not "1"!
name := "test_" + string(rune(i+1))  // BUG!
```

### Field Access

```go
// ✅ CORRECT
funcDecl.Tests        // []*ast.TestCase
funcDecl.Properties   // []*ast.Property

// ❌ WRONG
funcDecl.InlineTests  // Doesn't exist! Use .Tests
```

## API Discovery Workflow

1. **`make doc PKG=<package>`** (~30 sec) ← Start here!
2. Check source file if you know location (`grep "^func New" file.go`)
3. Check test files for usage examples (`grep "NewCollector" *_test.go`)
4. Read [docs/guides/](../../../docs/guides/) for complex workflows

**Time savings**: 80% reduction (5-10 min → 30 sec per lookup)

## Full Reference

See CLAUDE.md "Common API Patterns" section for additional patterns and examples.
