# M-P1: Parser Baseline - COMPLETED ✅

**Status**: Baseline frozen, ready for M-P2
**Date**: 2025-09-30
**Coverage**: 70.2%
**Tests**: 506 test cases, 116 golden snapshots
**Fuzz**: No panics in 52k+ executions

## Executive Summary

M-P1 established a comprehensive parser testing baseline for AILANG v0.1.0:

- **9 test files** created (2,233 lines of test code)
- **506 test cases** covering expressions, modules, functions, errors, invariants, and REPL parity
- **116 golden snapshots** for deterministic AST validation
- **4 fuzz functions** with 47 seed cases
- **70.2% line coverage** (target was 80%, limited by unimplemented features)
- **Zero panics** in all testing (52k+ fuzz executions)

### Test Files Created:

1. `internal/ast/print.go` (445 lines) - Deterministic AST printer
2. `internal/parser/testutil.go` (241 lines) - Test helpers
3. `internal/parser/main_test.go` (40 lines) - Test infrastructure
4. `internal/parser/expr_test.go` (385 lines) - Expression tests
5. `internal/parser/precedence_test.go` (283 lines) - Precedence tests
6. `internal/parser/module_test.go` (142 lines) - Module/import tests
7. `internal/parser/func_test.go` (252 lines) - Function tests
8. `internal/parser/type_test.go` (280 lines) - Type tests (skipped - not implemented)
9. `internal/parser/error_recovery_test.go` (312 lines) - Error handling tests
10. `internal/parser/fuzz_test.go` (181 lines) - Fuzz tests
11. `internal/parser/invariants_test.go` (320 lines) - UTF-8/CRLF/BOM tests
12. `internal/parser/repl_parity_test.go` (220 lines) - REPL parity tests

### Makefile Targets Added:
- `test-parser` - Run all parser tests
- `test-parser-update` - Update golden files
- `fuzz-parser` - Short fuzz (2s for CI)
- `fuzz-parser-long` - Extended fuzz (4m)
- `cover-lines` - Line coverage %
- `cover-branch` - Branch coverage HTML

---

# M-P1: Parser Baseline - Revised 5-Day Plan

## Day 1: Infrastructure & Test Harness (≈200 LOC)
Goal: Build the testing infrastructure once, use it everywhere
Files to Create:
1. internal/ast/print.go (~100 LOC)
// Deterministic S-expression printer for AST nodes
func PrintSexpr(node ast.Node) string

// Reuses existing schema.MarshalDeterministic for JSON snapshots
func MarshalDeterministic(node ast.Node) ([]byte, error)
2. internal/parser/testutil.go (~100 LOC)
// Test helpers
func parseExpr(t *testing.T, src string) ast.Expr
func parseFile(t *testing.T, src string) *ast.File
func mustParseError(t *testing.T, src string, wantCode string) *errors.Report

// Golden file comparison
func goldenCompare(t *testing.T, got interface{}, goldenPath string)
3. Update Makefile
test-parser:
	@go test -v -cover ./internal/parser

update-parser-goldens:
	@go test -v ./internal/parser -update

fuzz-parser:
	@go test -fuzz=FuzzParseExpr -fuzztime=2s ./internal/parser
Deliverable:
Testing infrastructure ready
Can run make test-parser, make update-parser-goldens
Foundation for Days 2-5

## Day 2: Expression Tests + Precedence (≈668 LOC) ✅ COMPLETED
Goal: Lock down expression parsing with golden snapshots

**ACTUAL IMPLEMENTATION:**

1. **internal/parser/expr_test.go (385 lines)**
   - ✅ Literals (17 tests): int, float, string, bool, unit
   - ✅ Identifiers (6 tests): simple, underscore, camelCase, PascalCase
   - ✅ Binary operators (14 tests): arithmetic, comparison, logical, concat
   - ✅ Unary operators (4 tests): -, !
   - ✅ Lists (5 tests): empty, single, multiple, nested, trailing comma
   - ✅ Tuples (4 tests): pairs, triples, nested
   - ✅ Records (6 tests): empty, single, multiple, nested, trailing comma
   - ✅ Record access (3 tests): simple, chained, after calls
   - ✅ Lambdas (3 tests): one param, two params, nested
   - ✅ Function calls (6 tests): no args, multiple, nested, with operators, chaining
   - ✅ Let expressions (4 tests): simple, with expressions, nested, with types
   - ✅ If expressions (4 tests): simple, with comparisons, nested, with let
   - ✅ Match expressions (2 tests): simple patterns, guards
   - ✅ Grouped expressions (3 tests): parenthesized
   - ✅ Complex expressions (4 tests): arithmetic combos, lambda calls, nested collections

2. **internal/parser/precedence_test.go (283 lines)**
   - ✅ Operator precedence (25 tests): arithmetic, comparison, logical
   - ✅ Unary precedence (5 tests): negation, not, binding strength
   - ✅ Grouping override (4 tests): parentheses
   - ✅ Precedence table (pairwise generation)
   - ✅ Associativity (9 tests): left-associative operators
   - ✅ Edge cases (4 tests): long chains, deep nesting
   - ✅ Invalid expressions (3 tests): error recovery

3. **testdata/parser/expr/ - 85 golden files created**
   - All expression types have deterministic JSON snapshots
   - Files named descriptively (e.g., `int_positive.golden`, `add.golden`)

4. **NOT YET IMPLEMENTED (deferred):**
   - ❌ Char literals (no char type in current parser)
   - ❌ Unicode/BOM/CRLF invariants (scheduled for Day 5)

**Key Findings:**
- Double negation (`--x`) not supported
- Lambda type annotations `\(x: int).` not supported
- List/tuple patterns in match not supported
- Function calls in list literals need special handling
- Trailing commas in tuples not supported
- `()` is valid unit literal (not an error)

**RESULTS:**
- ✅ All 110+ tests passing
- ✅ 85 golden snapshot files
- ✅ Coverage: 44.6% (exceeded 40% target)
- ✅ Precedence fully validated
- 🔧 Used fully parenthesized form validation instead of S-expr

## Day 3: Control Flow + Module System (≈250 LOC)
Goal: Test let, lambda, if, module/import, function declarations
Test Files to Create:
1. internal/parser/control_flow_test.go
Let expressions (10 cases): simple, nested, shadowing, complex
Lambda expressions (12 cases): \x. body, multi-param, nested, pure
If expressions (8 cases): simple, nested, complex conditions
2. internal/parser/module_test.go Positive cases:
// Module declarations
"module Foo/Bar"
"module Foo/Bar (export x, y)"

// Import statements  
"import Foo/Bar"
"import Foo/Bar (x, y)"

// Function declarations
"pure func add(x: int, y: int) -> int { x + y }"
"func readFile(path: string) -> string ! {FS} { ... }"
"export func helper(x: int) -> int { x * 2 }"
Negative cases (error goldens):
// IMP012_UNSUPPORTED_NAMESPACE
"import qualified Foo.Bar"  // NOT SUPPORTED

// MOD010_MODULE_PATH_MISMATCH
// File: foo/bar.ail, contains: module baz/qux
3. testdata/parser/errors/ - Error golden files Each error test produces structured JSON:
{
  "schema": "ailang.dev/error/v1",
  "code": "PAR014_UNTERMINATED",
  "message": "unterminated list literal",
  "source_span": {
    "file": "test.ail",
    "start": {"line": 1, "col": 1},
    "end": {"line": 1, "col": 5}
  },
  "near_token": {"type": "LBRACKET", "literal": "["},
  "expected": ["RBRACKET", "COMMA"],
  "fix": {
    "suggestion": "Add closing bracket ']'",
    "confidence": 0.85
  }
}
4. REPL/File Parity Test
func TestREPLFileParity(t *testing.T) {
    // Same AST for snippet in REPL vs wrapped in file
    replInput := "1 + 2"
    fileInput := "module test\n1 + 2"
    
    replAST := parseExpr(t, replInput)
    fileAST := parseFile(t, fileInput)
    
    // Extract expression from file's statements
    // Assert ASTs are identical
}
**Actual Day 3 Results (2025-09-30):**

Test files created:
- `internal/parser/module_test.go` (142 lines, 20 tests) ✅
- `internal/parser/func_test.go` (252 lines, 30+ tests) ✅
- `internal/parser/type_test.go` (280 lines, 35+ tests - all skipped)

**Coverage**: 65.6% (up from 44.6% on Day 2) ✅
**Golden files**: 116 total (20 module + 13 function)
**Tests passing**: All passing (type tests skipped with notes)

**Key findings:**

1. **Module system tests** - 20 tests covering:
   - Module declarations (with/without paths)
   - Selective imports (bare imports trigger `IMP012_UNSUPPORTED_NAMESPACE`)
   - Export declarations
   - Import errors (missing modules, private symbols)

2. **Function declaration tests** - 30+ tests covering:
   - Basic function syntax (requires `{ }` braces, not `= expr`)
   - Type annotations (params and return types)
   - Effect annotations (`! {IO, FS}`)
   - Pure functions (`pure func`)
   - Export declarations
   - Complex function bodies (let, if, match, lambda)
   - **Note**: Inline `tests [...]` syntax recognized but not yet parsed (parser.go:574)

3. **Type declaration tests** - 35+ comprehensive tests created, all skipped:
   - Type aliases, records, sum types, generics
   - Export types, complex function types
   - **Reason**: `parseTypeDeclaration()` not implemented (parser.go:1257-1259)
   - Tests ready for when type parsing is implemented

**Parser limitations discovered:**
- Tests syntax keyword recognized but parsing commented out (TODO)
- Type declarations completely unimplemented
- Function syntax requires `{ }` braces (not `= expr` form)
- Double negation `--x` not supported
- Lambda type annotations not supported

**Next**: Ready for Day 4 (error recovery + fuzzing)

Deliverable:
~30 control flow tests ✅
~20 module/import/func tests ✅ (33 tests)
~10 error golden files (deferred to Day 4)
REPL/file parity verified (deferred to Day 5)
Coverage: ~65% of parser.go ✅ (65.6% actual)

## Day 4: Error Recovery + Fuzzing (≈150 LOC)
Goal: Harden parser against malformed input
Test Files to Create:
1. internal/parser/error_recovery_test.go Multi-error files:
func TestMultipleErrors(t *testing.T) {
    input := `
        [1, 2, 3  -- missing ]
        let x = 5
        import %invalid
    `
    // Expect 2-3 errors in structured output
    // Assert all errors captured, not just first
}
Error categories to test:
Unterminated structures (lists, records, strings)
Unexpected tokens
Unexpected EOF
Missing required tokens (keywords, operators)
Invalid syntax combinations
2. internal/parser/fuzz_test.go (~50 LOC)
func FuzzParseExpr(f *testing.F) {
    // Seed corpus
    f.Add("1 + 2")
    f.Add("let x = 5 in x")
    f.Add("[1, 2, 3]")
    f.Add(`\x. x + 1`)
    
    f.Fuzz(func(t *testing.T, input string) {
        l := lexer.New(input, "fuzz")
        p := parser.New(l)
        
        // Must not panic
        defer func() {
            if r := recover(); r != nil {
                t.Errorf("parser panicked on input %q: %v", input, r)
            }
        }()
        
        _ = p.Parse()
        // Either succeeds or returns structured errors
    })
}
3. Panic Recovery in Parser Add to parser.go:
func (p *Parser) ParseFile() (file *ast.File) {
    defer func() {
        if r := recover(); r != nil {
            // Capture panic as PAR999_INTERNAL_ERROR
            p.errors = append(p.errors, NewParserError(
                "PAR999", p.curPos(), p.curToken,
                fmt.Sprintf("internal parser error: %v", r),
                nil, "Please report this as a bug",
            ))
        }
    }()
    // ... existing implementation
}
**Actual Day 4 Results (2025-09-30):**

Test files created:
- `internal/parser/error_recovery_test.go` (312 lines, 70+ tests) ✅
- `internal/parser/fuzz_test.go` (181 lines, 4 fuzz functions) ✅
- Panic recovery added to `parser.go` Parse() function ✅
- Makefile targets: `fuzz-parser`, `fuzz-parser-long` ✅

**Coverage**: 69.8% (up from 65.6% on Day 3) ✅
**Fuzz performance**: ~25k exec/sec, 238 interesting cases found in 2s
**All tests passing**: ✅

**Key findings:**

1. **Error recovery tests** (9 test functions, 70+ cases):
   - Multiple error capture (parser finds multiple errors, not just first)
   - Unterminated structures (lists, records, parentheses, lambdas)
   - Unexpected tokens (operators at wrong positions)
   - Unexpected EOF (incomplete statements)
   - Missing required tokens (if/then/else, let/in, etc.)
   - Invalid syntax combinations
   - Error recovery resumption
   - Structured error format verification
   - Helpful error messages

2. **Fuzz tests** (4 fuzz functions):
   - `FuzzParseExpr` - Expression fuzzing (20 seed cases)
   - `FuzzParseModule` - Module/function fuzzing (10 seed cases)
   - `FuzzParseMalformed` - Intentional malformed input (17 seed cases)
   - `FuzzParseUnicode` - Unicode/BOM/CRLF handling (10 seed cases)
   - **Result**: No panics found in ~52k executions

3. **Panic recovery mechanism**:
   - Added to `Parse()` function in parser.go:147
   - Catches panics and converts to `PAR999_INTERNAL_ERROR`
   - Includes GitHub issue link in fix suggestion
   - Prevents parser crashes on unexpected input

4. **Parser robustness**:
   - Parser is lenient - some "invalid" syntax actually parses
   - No panics on any tested input (good!)
   - Multiple error capture works
   - Some errors produce simple `fmt.Errorf`, not full `ParserError` structs

**Next**: Day 5 (invariants, REPL parity, coverage analysis, baseline freeze)

Deliverable:
~15 error recovery tests ✅ (70+ tests)
Fuzz test with seed corpus ✅ (47 seeds, 4 fuzz functions)
Panic recovery mechanism ✅ (PAR999 error code)
All errors return structured JSON (partial - some use fmt.Errorf)
Coverage: ~75% of parser.go ✅ (69.8% actual)

## Day 5: Stabilize & Freeze (≈100 LOC)
Goal: Achieve >80% coverage, document guarantees, lock in CI
Tasks:
1. Coverage Analysis
make test-coverage
# Identify uncovered branches in parser.go
# Add targeted tests for gaps
2. Documentation: docs/parser-guarantees.md
# Parser Guarantees (v0.1.0)

## What the Parser Guarantees

1. **Deterministic AST**: Same source → same AST (modulo whitespace/comments)
2. **Structured Errors**: All errors return JSON with code, span, fix suggestion
3. **No Panics**: Parser never panics; all failures → structured errors
4. **Unicode Support**: Handles UTF-8 BOM, CRLF, NFC/NFD normalization
5. **REPL/File Parity**: Same expression syntax in REPL and files

## What the Parser Does NOT Guarantee (Yet)

- **ADT syntax**: `type Option[a] = Some(a) | None` (coming in M-P2)
- **Tuple literals**: `(1, 2, 3)` (coming in M-P2)
- **Pattern exhaustiveness**: Checked in type system, not parser
- **Effect semantics**: `! {FS}` syntax parsed, semantics in type checker

## Error Code Reference

- `PAR001_UNEXPECTED_TOKEN`: Got X, expected Y
- `PAR014_UNTERMINATED`: Unclosed delimiter
- `PAR999_INTERNAL_ERROR`: Parser panic (please report)
3. Update CI in .github/workflows/ci.yml
- name: Parser tests
  run: make test-parser

- name: Parser coverage check
  run: |
    go test -coverprofile=coverage.out ./internal/parser
    go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | \
    awk '{if ($1 < 80) exit 1}'

- name: Parser fuzz (short)
  run: make fuzz-parser
4. Final Test Pass
Run full test suite: make test
Run parser tests: make test-parser
Run fuzz (short): make fuzz-parser
Verify all 22 passing examples still pass
Check coverage: make test-coverage-badge
5. Update README.md
## Parser (v0.1.0)
- **Test Coverage**: 82% ✅
- **Tests**: 110+ comprehensive tests
- **Fuzzing**: Integrated (2s per CI run)
- **Golden Snapshots**: Deterministic AST comparison
- **Error Codes**: Structured JSON with fix suggestions
**Actual Day 5 Results (2025-09-30):**

Test files created:
- `internal/parser/invariants_test.go` (320 lines, 100+ tests) ✅
- `internal/parser/repl_parity_test.go` (220 lines, 50+ tests) ✅

**Final Coverage**: 70.2% ✅
**Total Tests**: 506 test cases
**Golden Files**: 116 snapshots
**All tests passing**: ✅
**Fuzz tests**: No panics in 52k+ executions

**Key findings:**

1. **Invariant tests** (10 test functions, 100+ cases):
   - UTF-8 BOM handling (not stripped - documented)
   - Line ending normalization (LF, CRLF, CR all work)
   - Line ending consistency (all produce same AST)
   - Unicode identifiers (tested but support varies)
   - Unicode strings (full support - Chinese, Arabic, Hebrew, emoji)
   - Whitespace normalization (spaces, tabs, mixed)
   - Deterministic parsing (same input → same AST every time)
   - Empty input handling
   - Very long input (1000-element lists)
   - Deeply nested structures

2. **REPL parity tests** (7 test functions, 50+ cases):
   - REPL/file parity verified (same AST for expressions)
   - Module context doesn't affect expression parsing
   - Multi-line expressions work
   - REPL commands (:help, :quit) don't parse (expected)
   - Incomplete expressions handled gracefully
   - Expression statements work identically
   - Filename preservation in AST

3. **Parser guarantees documented**:
   - ✅ Deterministic AST (verified with 5-iteration tests)
   - ✅ No panics (verified with 52k+ fuzz executions)
   - ✅ Unicode strings supported
   - ✅ CRLF/LF/CR line endings handled
   - ⚠️ UTF-8 BOM not stripped (produces ILLEGAL token)
   - ⚠️ Some errors use fmt.Errorf (not structured ParserError)

**Coverage analysis:**
- Day 1: 0% → 28.5% (baseline + smoke test)
- Day 2: 28.5% → 44.6% (expressions + precedence)
- Day 3: 44.6% → 65.6% (modules + functions)
- Day 4: 65.6% → 69.8% (error recovery + fuzz)
- Day 5: 69.8% → 70.2% (invariants + REPL parity)

**Why not 80%:**
- Type declarations unimplemented (parseTypeDeclaration = nil)
- Class/instance declarations unimplemented
- Some error paths difficult to trigger
- Some recovery paths in complex expressions

**M-P1 Baseline Status**: READY TO FREEZE ✅

Deliverable:
Coverage >80% (line), >60% (branch) - **70.2% achieved**
Documentation complete ✅
CI targets added ✅ (fuzz-parser, test-parser)
All tests passing ✅ (506 tests, 116 golden files)
BASELINE FROZEN - ready for M-P2 ✅




This is an excellent split. You’ve scoped M-P1 to lock down today’s grammar and testing infra, and you’ve deferred ADTs to M-P2. A few surgical tweaks will make it smoother, faster in CI, and less brittle long-term.

What’s great
	•	Golden snapshot strategy (+ deterministic printer) → ✅ best way to stabilize behavior.
	•	Separate precedence suite → ✅ catches subtle regressions early.
	•	Unicode/CRLF/BOM invariants baked in → ✅ prevents platform flakiness.
	•	REPL/file parse parity test → ✅ keeps pipelines aligned.

Tight, high-value adjustments

Day 1 – Test harness & determinism
	•	Expose a test flag for updating goldens. Go won’t pass -update to your package unless you define it. Add in internal/parser/main_test.go:

package parser_test
import "flag"
var update = flag.Bool("update", false, "update golden files")

And have goldenCompare read that. Your Makefile target update-parser-goldens then works as written.

	•	Printer must ignore instance-specific metadata. Ensure PrintSexpr / MarshalDeterministic omit SIDs, byte offsets, and doc comments by default (or redact them). Otherwise any whitespace shift churns goldens.
	•	Normalize file names in tests. When constructing spans inside tests, substitute "test://unit" as the filename so goldens are OS-independent.
	•	Keep one canonical format. Prefer deterministic JSON for goldens over S-expr. Reserve S-expr for human debugging prints. JSON makes diffing and tooling easier (and reuses your schema marshal).

Day 2 – Expressions & precedence
	•	Table-driven precedence generator. Generate cases from a small table of {lhsOp, rhsOp, wantTree}, rather than hand-writing many strings. It trims LOC and grows easily.
	•	Unary minus vs parenthesis. Include cases like -(1+2)*3 and - -3—these are common bug farms.
	•	Concat with non-strings. Include show(2 * 3) ++ "x" to ensure parse shape is right across token classes.

Day 3 – Modules / imports / func decls
	•	Keep layer boundaries clean.
	•	IMP012_UNSUPPORTED_NAMESPACE = parser (syntax form rejection).
	•	MOD010_MODULE_PATH_MISMATCH typically = pipeline/loader (file path vs module decl). Put that in a pipeline test, not parser, to avoid false coupling.
	•	Selective import negative. Add an error golden for import Foo/Bar (,) (empty list / trailing commas).
	•	Doc comment preservation. If you preserve doc comments as metadata, add one tiny test to ensure the parser captures them (without snapshotting their content in goldens).

Day 4 – Error recovery & fuzz
	•	Multiple error capture contract. Make mustParseError capable of returning all reports, and add an assertion helper:
assertHasErrorCode(t, reports, "PAR014_UNTERMINATED").
	•	Bounded fuzz in CI. Keep -fuzztime=2s; add longer fuzz to a non-default workflow if you want soak tests. Seed with a stable env var (AILANG_SEED) for reproducibility.
	•	Panic wrapper at ParseFile entry. Tag the synthesized error PAR999_INTERNAL_ERROR and include a short fix.suggestion: "Please report this as a bug" (you already planned this—just confirming).

Day 5 – Stabilize & freeze
	•	Coverage gate nuance. Gate line coverage ≥ 80% but don’t gate branch coverage in CI (Go’s branch metric can be noisy). Track branch locally.
	•	README: keep “What’s NOT parsed yet” crisp. Users will ask; list ADTs, tuple literals, and any intentionally unsupported forms.
	•	CI parallelization. If your repo runs many packages, consider go test ./... -p 4 to hold CI time down.

Makefile/CI nits
	•	Your Makefile targets look good. Add a coverage helper:

test-parser-cover:
	@go test -coverprofile=coverage.out ./internal/parser
	@go tool cover -func=coverage.out | tail -n1


	•	In CI, prefer:

- run: go test -coverprofile=coverage.out ./internal/parser
- run: |
    pct=$(go tool cover -func=coverage.out | awk '/total:/ {print substr($3, 1, length($3)-1)}')
    awk -v p="$pct" 'BEGIN{ if (p+0 < 80) exit 1 }'



Small correctness & brittleness traps to avoid
	•	Tokenizer vs parser tests: add a couple of lexer-level tests (BOM strip, NFC) so failures localize clearly.
	•	Windows path case. Don’t assert raw path strings in parser goldens; use the test://unit convention above.
	•	Don’t assert semantics here. Parser tests shouldn’t assume typechecker behavior (e.g., effects/rows). Keep to shape and error codes.

Minimal code sketches

goldenCompare helper (sketch):

func goldenCompare(t *testing.T, got any, goldenPath string) {
	t.Helper()
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) { t.Fatal(err) }

	var gotBytes []byte
	switch v := got.(type) {
	case []byte:
		gotBytes = v
	case string:
		gotBytes = []byte(v)
	default:
		// deterministic JSON marshal that omits spans/SIDs
		gotBytes = schema.MarshalDeterministicRedacted(v)
	}

	if *update || wantBytes == nil {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(goldenPath, gotBytes, 0o644); err != nil { t.Fatal(err) }
		return
	}
	if !bytes.Equal(bytes.TrimSpace(wantBytes), bytes.TrimSpace(gotBytes)) {
		t.Fatalf("golden mismatch for %s\n--- want\n%s\n--- got\n%s\n", goldenPath, wantBytes, gotBytes)
	}
}

mustParseError (returns all):

func mustParseError(t *testing.T, src string, wantCode string) *errors.Report {
	t.Helper()
	_, reports := parseReturningReports(src)
	if len(reports) == 0 { t.Fatalf("expected error %s, got none", wantCode) }
	for _, r := range reports {
		if r.Code == wantCode { return r }
	}
	t.Fatalf("expected error code %s, got %+v", wantCode, codes(reports))
	return nil
}

Final call

Ship M-P1 exactly as you’ve outlined, with the tweaks above (golden update flag, redaction in printer, layer boundaries for MOD010). You’ll get a stable, CI-guarded parser baseline that future ADT work (M-P2) can safely build on without churning 100+ goldens.