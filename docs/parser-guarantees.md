# Parser Guarantees (v0.1.0)

**Baseline Frozen**: 2025-09-30
**Milestone**: M-P1 Complete
**Coverage**: 70.2% line coverage
**Test Suite**: 506 test cases, 116 golden snapshots

---

## What the Parser Guarantees

### 1. **Deterministic AST Generation**

The parser produces **identical AST structures** for the same source code, regardless of:
- Platform (macOS, Linux, Windows)
- Line endings (LF, CRLF, CR)
- Whitespace variations (spaces, tabs, mixed)
- Multiple parsing runs

**Verification**: Tested with 5-iteration determinism tests on 506 test cases.

**Example**:
```ailang
let x = 5 in x + 1
```

Always produces the same AST, whether run on:
- Unix (LF line endings)
- Windows (CRLF line endings)
- With spaces or tabs
- Multiple times in succession

### 2. **No Panics**

The parser **never panics** on any input, no matter how malformed.

**Verification**: Fuzz tested with 52k+ random inputs across:
- Valid expressions
- Module declarations
- Intentionally malformed syntax
- Unicode edge cases

**Panic Recovery**:
- All panics caught in `Parse()` function
- Converted to structured `PAR999_INTERNAL_ERROR`
- Includes GitHub issue link for reporting

**Example**:
```ailang
[[[[[     -- Unterminated brackets
let x =   -- Incomplete let
@#$%^&    -- Invalid tokens
```

All produce structured errors, no panics.

### 3. **Unicode String Support**

Full support for Unicode in string literals:
- ✅ Chinese: `"你好世界"`
- ✅ Arabic: `"مرحبا بالعالم"`
- ✅ Hebrew: `"שלום עולם"`
- ✅ Emoji: `"Hello 🌍🚀✨"`
- ✅ Mixed: `"Café résumé π ∞"`

**Verification**: Tested with 50+ Unicode test cases covering major scripts and emoji.

### 4. **Cross-Platform Line Endings**

All line ending styles produce **identical ASTs**:
- Unix LF (`\n`)
- Windows CRLF (`\r\n`)
- Old Mac CR (`\r`)
- Mixed line endings

**Verification**: Line ending consistency tests verify identical AST structure.

**Example**:
```ailang
let x = 1\n     -- Unix
let y = 2\r\n   -- Windows
let z = 3\r     -- Old Mac
```

All parse identically (modulo line ending normalization).

### 5. **REPL/File Parsing Parity**

Expressions parse **identically** in REPL and file contexts:

**REPL**:
```ailang
λ> 1 + 2 * 3
```

**File**:
```ailang
-- test.ail
1 + 2 * 3
```

Both produce the same AST structure.

**Verification**: 50+ parity tests comparing REPL and file parsing.

### 6. **Error Recovery**

Parser attempts to recover from errors and report **multiple issues**, not just the first:

**Example**:
```ailang
[1, 2, 3      -- Missing ]
let x =       -- Incomplete let
import        -- Incomplete import
```

Reports all three errors, not just the first.

**Verification**: 70+ error recovery tests.

---

## What the Parser Does NOT Guarantee (Yet)

### ⚠️ Limitations

**UTF-8 BOM Not Stripped**:
- UTF-8 Byte Order Mark (`\xEF\xBB\xBF`) produces `ILLEGAL` token
- Workaround: Strip BOM before parsing
- Future: May add BOM stripping to lexer

**Partial Error Structure**:
- Some errors use `fmt.Errorf` (plain Go errors)
- Some use structured `ParserError` (with codes, positions, fix suggestions)
- Future: Migrate all errors to structured format

**Unimplemented Features**:
- Type declarations (`type Foo = Bar`) - parser stub exists, returns nil
- Class declarations (`class Show a where...`) - not implemented
- Instance declarations (`instance Show Int where...`) - not implemented
- Inline test blocks (`tests [...]`) - keyword recognized but not parsed

### 📋 Coming in M-P2

The following are **intentionally deferred** to M-P2:

- Algebraic Data Types (ADTs): `type Option[a] = Some(a) | None`
- Tuple literals: `(1, 2, 3)`
- Pattern exhaustiveness checking
- Effect semantics (syntax parsed, semantics in type checker)

---

## Error Codes

### Structured Error Format

When available, errors follow this format:

```json
{
  "code": "PAR999_INTERNAL_ERROR",
  "message": "internal parser panic: ...",
  "position": {
    "file": "test.ail",
    "line": 5,
    "column": 12
  },
  "near_token": {
    "type": "LBRACKET",
    "literal": "["
  },
  "expected": ["RBRACKET", "COMMA"],
  "fix": {
    "suggestion": "Add closing bracket ']'",
    "confidence": 0.85
  }
}
```

### Error Code Reference

| Code | Description | Example |
|------|-------------|---------|
| `PAR001_UNEXPECTED_TOKEN` | Got X, expected Y | `expected next token to be }, got ( instead` |
| `PAR014_UNTERMINATED` | Unclosed delimiter | `unterminated list literal` |
| `MOD006` | Cannot export private names | `cannot export underscore-prefixed name '_foo'` |
| `IMP012_UNSUPPORTED_NAMESPACE` | Bare import not supported | `import Foo` (use `import Foo (bar)`) |
| `PAR999_INTERNAL_ERROR` | Parser panic (please report) | `internal parser panic: nil pointer` |

---

## Testing Infrastructure

### Test Files (12 files, 2,633 lines)

1. **internal/ast/print.go** (445 lines) - Deterministic AST printer
2. **internal/parser/testutil.go** (241 lines) - Test helpers
3. **internal/parser/main_test.go** (40 lines) - Test infrastructure
4. **internal/parser/expr_test.go** (385 lines) - Expression tests (85 cases)
5. **internal/parser/precedence_test.go** (283 lines) - Precedence tests (25 cases)
6. **internal/parser/module_test.go** (142 lines) - Module/import tests (20 cases)
7. **internal/parser/func_test.go** (252 lines) - Function tests (30+ cases)
8. **internal/parser/type_test.go** (280 lines) - Type tests (35+ cases, skipped)
9. **internal/parser/error_recovery_test.go** (312 lines) - Error tests (70+ cases)
10. **internal/parser/fuzz_test.go** (181 lines) - Fuzz tests (4 functions, 47 seeds)
11. **internal/parser/invariants_test.go** (320 lines) - Invariant tests (100+ cases)
12. **internal/parser/repl_parity_test.go** (220 lines) - REPL parity tests (50+ cases)

### Golden Snapshots

116 deterministic AST comparison files in `testdata/parser/`:
- Expression golden files
- Precedence golden files
- Module golden files
- Function golden files

### Makefile Targets

```bash
make test-parser              # Run all parser tests
make test-parser-update       # Update golden files
make fuzz-parser              # Fuzz for 2s (CI)
make fuzz-parser-long         # Fuzz for 4 minutes
make cover-lines              # Show line coverage %
make cover-branch             # Show branch coverage HTML
```

### CI Integration

Parser tests run in CI with:
- ✅ Full test suite (`make test-parser`)
- ✅ Coverage gate (≥70% required)
- ✅ Fuzz testing (2s per run)
- ✅ All tests must pass

See `.github/workflows/ci.yml` for details.

---

## Coverage Analysis

### Day-by-Day Progress

| Day | Coverage | Increment | Tests Added |
|-----|----------|-----------|-------------|
| Day 0 (baseline) | 0% | - | - |
| Day 1 (infrastructure) | 28.5% | +28.5% | Smoke test |
| Day 2 (expressions) | 44.6% | +16.1% | 85 expression + 25 precedence |
| Day 3 (modules/functions) | 65.6% | +21.0% | 20 module + 30+ function |
| Day 4 (error recovery) | 69.8% | +4.2% | 70+ error + 4 fuzz functions |
| Day 5 (invariants) | 70.2% | +0.4% | 100+ invariants + 50+ REPL |

**Final**: 70.2% line coverage (506 tests, 116 golden files)

### Why Not 80%?

Coverage stopped at 70.2% due to:
- **Type declarations**: `parseTypeDeclaration()` returns nil (not implemented)
- **Class declarations**: `parseClassDeclaration()` returns nil (not implemented)
- **Instance declarations**: `parseInstanceDeclaration()` returns nil (not implemented)
- **Error paths**: Some error recovery paths difficult to trigger
- **Complex expressions**: Some nested expression recovery paths

**Next Steps for 80%+**:
- Implement type declaration parsing (M-P2)
- Implement class/instance parsing (M-P2)
- Add more error path tests
- Expand complex expression error recovery

---

## Parser Architecture

### Parse Pipeline

```
Input (string)
  ↓
Lexer (tokenization)
  ↓
Parser (AST generation)
  ↓
Deterministic Printer (for testing)
  ↓
Golden Comparison (validation)
```

### Key Design Decisions

1. **JSON over S-expressions**: Used JSON for golden files instead of S-expressions for better tooling support and readability.

2. **Fully Parenthesized Form**: Used for precedence validation instead of S-expressions, making tests more intuitive.

3. **Skip vs Error**: Tests for unimplemented features (types, classes) are skipped with notes rather than failing.

4. **Panic Recovery**: Added at top-level `Parse()` to catch all panics and convert to errors.

5. **REPL Parity**: Used same parser for REPL and files to ensure consistency.

---

## Known Issues & Workarounds

### Issue 1: UTF-8 BOM Not Stripped

**Problem**: Files with UTF-8 BOM produce `ILLEGAL` token error.

**Workaround**: Strip BOM before parsing:
```go
if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
    data = data[3:]
}
```

### Issue 2: Inline Tests Not Parsed

**Problem**: Function inline tests recognized but not parsed:
```ailang
func factorial(n: int) -> int
  tests [(0, 1), (5, 120)]
{ ... }
```

**Workaround**: Tests syntax is recognized (won't error) but test cases not captured in AST.

**Status**: Implementation planned for M-P2.

### Issue 3: Type Declarations Not Implemented

**Problem**: Type declarations parse as nil:
```ailang
type Option[a] = Some(a) | None
```

**Workaround**: None - feature not implemented.

**Status**: ADT parsing planned for M-P2.

---

## Future Improvements (M-P2+)

### Short Term (M-P2)

- [ ] Implement type declaration parsing
- [ ] Implement class/instance declaration parsing
- [ ] Parse inline test blocks
- [ ] Migrate all errors to structured `ParserError`
- [ ] Strip UTF-8 BOM in lexer

### Medium Term (M-P3)

- [ ] Add more error recovery strategies
- [ ] Improve error messages with context
- [ ] Add syntax error suggestions (did you mean?)
- [ ] Implement incremental parsing for IDE support

### Long Term (M-P4+)

- [ ] Streaming parser for large files
- [ ] Parallel parsing for multi-file projects
- [ ] AST caching for build performance
- [ ] Language server protocol (LSP) integration

---

## References

- **Design Doc**: [design_docs/20250930/M-P1.md](../design_docs/20250930/M-P1.md)
- **Test Files**: [internal/parser/*_test.go](../internal/parser/)
- **Golden Files**: [internal/parser/testdata/parser/](../internal/parser/testdata/parser/)
- **Issue Tracker**: [GitHub Issues](https://github.com/sunholo/ailang/issues)

---

**Baseline Status**: ✅ FROZEN - Ready for M-P2
**Last Updated**: 2025-09-30
**Next Milestone**: M-P2 (Type System + ADT Parsing)