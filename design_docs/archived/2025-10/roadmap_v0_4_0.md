# AILANG v0.4.0 Roadmap - Language Refinements & Developer Experience

**Status**: Planning
**Target Date**: Q1 2026
**Focus**: Developer ergonomics, language completeness, stdlib expansion

---

## Version Context

**Current Version**: v0.3.7 (October 15, 2025)
- ✅ Full module execution with effects (IO, FS, Clock, Net)
- ✅ Anonymous function syntax, letrec, numeric conversions
- ✅ Record update syntax, auto-import std/prelude
- ✅ 38.6% AILANG M-EVAL success rate (58.8% combined with Python)
- ✅ Complete type system (HM + type classes + row polymorphism)

**Next Patch**: v0.3.8 (Week of October 21, 2025)
- Focus: Bug fixes (multi-line ADT syntax, ADT runtime regression)
- Goal: Restore v0.3.6 parity (42%+ AILANG success)
- Timeline: 3-4 days
- See: [roadmap_v0_3_8.md](roadmap_v0_3_8.md)

**Next Major Version**: v0.4.0 (Q1 2026)
- Focus: Language refinements for AI codegen ergonomics
- Goal: 75%+ AILANG M-EVAL success, 85%+ example success
- Theme: "Developer Experience & Language Completeness"

**Future Versions**:
- v0.5.0: Typed quasiquotes, metaprogramming (Q2-Q3 2026)
- v0.6.0: CSP concurrency, session types (Q4 2026)
- v1.0.0: Deterministic execution, training data export (2027)

---

## Objectives for v0.4.0

### 1. Language Completeness (P2 Items from 20251013 Audit)

**Goal**: Address deferred P2 features for better ergonomics

#### ✨ P2a: Capability Inference (AUTO_CAPS)

**Problem**: Users must manually pass `--caps IO,FS,Clock` flags
```bash
# Current (verbose):
ailang run --caps IO,FS,Clock --entry main module.ail

# Desired (automatic):
ailang run --entry main module.ail  # Infers IO,FS,Clock from main's type
```

**Solution**: Static analysis of entry function's effect row
- Parse entry function signature: `func main() -> () ! {IO, FS}`
- Extract effect set: `{IO, FS}`
- Auto-pass to runtime capability checker

**Implementation**:
- `internal/loader/`: Add `InferCapabilities(module, entryFn) -> []string`
- `cmd/ailang/run.go`: Call inference if `--caps` not provided
- `internal/pipeline/`: Thread inferred capabilities to runtime

**Estimate**: 2-3 days, ~200 LOC
**Impact**: Removes friction for new users, reduces command-line noise

---

#### ✨ P2b: Better List Syntax

**Problem**: Current list syntax is verbose (ADT constructors)
```ailang
-- Current (verbose):
let xs = Cons(1, Cons(2, Cons(3, Nil)));

-- Desired (native):
let xs = [1, 2, 3];
```

**Options**:

**Option 1: Native List Type** (preferred)
- Add `List<T>` as builtin type (like records, not ADT)
- Syntax: `[1, 2, 3]` → `List<int>`
- Builtins: `head`, `tail`, `length`, `map`, `filter`, `fold`
- Pattern matching: `[x, y, ...rest]` or `[x, y, z]`

**Option 2: Improved Sugar** (fallback)
- Keep ADT approach, add syntactic sugar
- `[1, 2, 3]` desugars to `Cons(1, Cons(2, Cons(3, Nil)))`
- Less work, but still exposes ADT in error messages

**Recommendation**: Option 1 (native lists) for better UX

**Implementation**:
- Lexer: `LBRACKET`, `RBRACKET` already exist
- Parser: `parseListLiteral()` in expressions (~80 LOC)
- AST: `ListLiteral{Elems []Expr}` (~20 LOC)
- Core: `core.ListLiteral` (~15 LOC)
- Types: `TList{Elem Type}` (~50 LOC)
- Eval: `ListValue{Elems []Value}` (~40 LOC)
- Builtins: `head`, `tail`, `length`, `map`, `filter`, `fold` (~300 LOC)

**Estimate**: 5-7 days, ~500 LOC
**Impact**: Major ergonomics improvement, matches user expectations

---

#### ✨ P2c: Error Propagation Operator `?`

**Problem**: Explicit error handling is verbose
```ailang
-- Current (verbose):
func processFile(path: string) -> Result<string, Error> ! {FS} {
  match readFile(path) {
    | Ok(content) -> match parseContent(content) {
        | Ok(data) -> Ok(data)
        | Err(e) -> Err(e)
      }
    | Err(e) -> Err(e)
  }
}

-- Desired (concise):
func processFile(path: string) -> Result<string, Error> ! {FS} {
  let content = readFile(path)?;  -- Early return on Err
  let data = parseContent(content)?;
  Ok(data)
}
```

**Solution**: Rust-style `?` operator for Result/Option types
- `expr?` → Early return on `Err(e)` or `None`
- Only works in functions returning `Result<T, E>` or `Option<T>`
- Desugars to match expression

**Implementation**:
- Lexer: Add `QUESTION` token (~5 LOC)
- Parser: Postfix operator `parsePostfix()` (~30 LOC)
- AST: `TryExpr{Expr}` (~15 LOC)
- Elaborate: Desugar to match (~50 LOC)
- Types: Verify return type compatibility (~40 LOC)

**Estimate**: 3-4 days, ~150 LOC
**Impact**: Reduces boilerplate, improves readability

---

#### ✨ P2d: List Comprehensions

**Problem**: Functional list operations are verbose
```ailang
-- Current (verbose):
let evens = filter(\x. x % 2 == 0, map(\x. x * 2, xs));

-- Desired (concise):
let evens = [x * 2 | x <- xs, x % 2 == 0];
```

**Solution**: Haskell-style list comprehensions
- Syntax: `[expr | var <- list, predicate, ...]`
- Supports multiple generators and guards
- Desugars to `map`/`filter`/`flatMap` chains

**Implementation**:
- Parser: `parseListComprehension()` (~100 LOC)
- AST: `ListComprehension{Expr, Generators, Guards}` (~40 LOC)
- Elaborate: Desugar to nested map/filter (~150 LOC)

**Estimate**: 4-5 days, ~300 LOC
**Impact**: Improves code clarity for data transformations

---

### 2. Standard Library Expansion

**Goal**: Add commonly-needed stdlib modules

#### ✨ std/json - JSON Parsing & Serialization

**Functions**:
- `parseJSON(string) -> Result<Value, Error>`
- `stringify(Value) -> string`
- `get(Value, string) -> Option<Value>`
- `set(Value, string, Value) -> Value`

**Types**:
```ailang
type JSON =
  | JNull
  | JBool(bool)
  | JNumber(float)
  | JString(string)
  | JArray(List<JSON>)
  | JObject(Record<string, JSON>)  -- If we have maps
```

**Estimate**: 3-4 days, ~400 LOC

---

#### ✨ std/cli - Command-Line Arguments

**Functions**:
- `getArgs() -> List<string> ! {IO}`
- `parseArgs(List<string>, Schema) -> Result<Args, Error>`
- `printUsage(Schema) -> () ! {IO}`

**Example**:
```ailang
import std/cli (getArgs, parseArgs)

func main() -> () ! {IO} {
  let args = getArgs();
  match parseArgs(args, mySchema) {
    | Ok(parsed) -> runApp(parsed)
    | Err(e) -> println("Error: " ++ show(e))
  }
}
```

**Estimate**: 2-3 days, ~250 LOC

---

#### ✨ std/string - Extended String Operations

**Functions**:
- `split(string, string) -> List<string>`
- `join(List<string>, string) -> string`
- `trim(string) -> string`
- `startsWith(string, string) -> bool`
- `endsWith(string, string) -> bool`
- `replace(string, string, string) -> string`
- `toUpper(string) -> string`
- `toLower(string) -> string`

**Estimate**: 2 days, ~200 LOC

---

#### ✨ std/result - Result Utilities

**Functions**:
- `isOk(Result<T, E>) -> bool`
- `isErr(Result<T, E>) -> bool`
- `unwrapOr(Result<T, E>, T) -> T`
- `mapOr(Result<T, E>, U, func(T) -> U) -> U`
- `andThen(Result<T, E>, func(T) -> Result<U, E>) -> Result<U, E>`

**Estimate**: 1-2 days, ~150 LOC

---

### 3. REPL Improvements

**Goal**: Better developer experience in interactive mode

#### ✨ Multi-Line Editing

**Current**: Single-line input only (workaround: backslash continuation)
**Desired**: True multi-line editing with smart indentation

**Implementation**:
- Detect incomplete expressions (unclosed braces, etc.)
- Show continuation prompt `... `
- Allow arrow keys to navigate multiple lines

**Estimate**: 3-4 days, ~200 LOC

---

#### ✨ Better Error Messages

**Current**: Raw compiler errors (can be cryptic)
**Desired**: Contextual help with suggestions

**Example**:
```
Error: Undefined variable 'mpa'
Did you mean: map, max, main?

Suggestion: import std/list (map)
```

**Implementation**:
- Levenshtein distance for typo detection
- Scope analysis for available suggestions
- Import suggestions from stdlib

**Estimate**: 2-3 days, ~150 LOC

---

### 4. Example & Documentation Quality

**Goal**: Reach 85%+ example success rate (currently 72.7%)

#### ✨ Fix Remaining 18 Failing Examples

**Current**: 48/66 examples passing (72.7%)
**Target**: 56/66 examples passing (85%+)

**Approach**:
1. Categorize failures by root cause
2. Fix low-hanging fruit (simple bugs)
3. Document limitations for unfixable cases

**Estimate**: 1 week, ~500 LOC (bug fixes)

---

#### ✨ Comprehensive Tutorial Series

**Goal**: Progressive tutorial from basics to advanced

**Structure**:
1. **Getting Started**: Installation, hello world, REPL
2. **Language Basics**: Functions, types, pattern matching
3. **Type System**: Type inference, type classes, polymorphism
4. **Effects & Capabilities**: IO, FS, Clock, Net
5. **Modules & Projects**: Multi-file programs, stdlib usage
6. **Advanced Topics**: Higher-order functions, ADTs, records

**Estimate**: 1 week (documentation only)

---

## Sprint Plan (v0.4.0 Development)

### Phase 1: Language Features (3-4 weeks)

**Week 1**: Capability Inference + Better List Syntax
- Days 1-3: Implement AUTO_CAPS (~200 LOC)
- Days 4-7: Implement native lists (~500 LOC)

**Week 2**: Error Propagation & List Comprehensions
- Days 1-3: Implement `?` operator (~150 LOC)
- Days 4-7: Implement list comprehensions (~300 LOC)

**Week 3**: Standard Library Expansion
- Days 1-2: std/json (~400 LOC)
- Days 3-4: std/cli (~250 LOC)
- Days 5-7: std/string, std/result (~350 LOC)

**Week 4**: REPL & Polish
- Days 1-3: Multi-line editing (~200 LOC)
- Days 4-5: Better error messages (~150 LOC)
- Days 6-7: Bug fixes, testing

---

### Phase 2: Quality & Documentation (1-2 weeks)

**Week 5**: Examples & Testing
- Days 1-3: Fix failing examples (~500 LOC)
- Days 4-5: Add new examples for v0.4.0 features
- Days 6-7: M-EVAL validation (target: 75%+)

**Week 6**: Documentation
- Days 1-3: Tutorial series
- Days 4-5: API reference updates
- Days 6-7: Migration guide from v0.3.x

---

## Success Criteria

### Quantitative Metrics

| Metric | v0.3.7 Baseline | v0.4.0 Target | Stretch Goal |
|--------|----------------|--------------|-------------|
| M-EVAL Success | 58.8% | **75%+** | 80%+ |
| Example Success | 72.7% (48/66) | **85%+ (56/66)** | 90%+ (59/66) |
| Test Coverage | 28.8% | **35%+** | 40%+ |
| Stdlib Functions | ~30 | **50+** | 60+ |
| Documentation Pages | ~15 | **25+** | 30+ |

### Qualitative Goals

✅ **Ergonomics**: Auto-capability inference, native lists, `?` operator
✅ **Completeness**: Essential stdlib modules (json, cli, string)
✅ **Developer Experience**: Multi-line REPL, better errors
✅ **Documentation**: Comprehensive tutorials, API reference
✅ **Quality**: 85%+ examples working, 75%+ M-EVAL success

---

## Risk Assessment

### High Risk Items

**1. Native List Type** (P2b)
- Risk: Breaking change if existing code uses ADT lists
- Mitigation: Keep ADT lists working alongside native lists (compatibility mode)

**2. Error Propagation Operator** (P2c)
- Risk: Complex type system integration (Result return type checking)
- Mitigation: Implement as desugaring first, optimize later

### Medium Risk Items

**3. List Comprehensions** (P2d)
- Risk: Desugaring complexity with nested generators
- Mitigation: Start with simple single-generator case, extend incrementally

**4. Multi-Line REPL** (REPL improvement)
- Risk: Terminal handling complexity across platforms
- Mitigation: Use well-tested readline library

### Low Risk Items

**5. Capability Inference** (P2a)
- Risk: Minimal (static analysis only)

**6. Stdlib Expansion**
- Risk: Low (additive features, no breaking changes)

---

## Deferred to v0.5.0+

### P3-P4 Items (Long-term Vision)

**v0.5.0 (Q2-Q3 2026): Metaprogramming**
- ❌ Typed Quasiquotes
- ❌ Macro system
- ❌ AST manipulation

**v0.6.0 (Q4 2026): Concurrency**
- ❌ CSP-based channels
- ❌ Session types
- ❌ Actor model

**v1.0.0 (2027): AI Training**
- ❌ Deterministic execution
- ❌ Training data export
- ❌ Execution trace collection

---

## Migration from v0.3.x to v0.4.0

### Breaking Changes

**1. Native Lists Replace ADT Lists** (if implemented)
```ailang
-- v0.3.x (ADT lists):
let xs = Cons(1, Cons(2, Nil));

-- v0.4.0 (native lists):
let xs = [1, 2, 3];  -- Preferred

-- Migration: ADT lists still work (compatibility mode)
import std/adt/list (Cons, Nil)  -- Explicit import needed
```

**2. Capability Flags Optional**
```bash
# v0.3.x (explicit):
ailang run --caps IO,FS module.ail

# v0.4.0 (inferred):
ailang run module.ail  # Same behavior
```

### New Features Opt-In

**1. Error Propagation Operator**
```ailang
-- Only works if function returns Result/Option
func process() -> Result<T, E> {
  let x = mayFail()?;  -- New syntax
  Ok(x)
}
```

**2. List Comprehensions**
```ailang
-- Syntactic sugar (desugars to map/filter)
let evens = [x * 2 | x <- xs, x % 2 == 0];
```

---

## Stakeholder Communication

### For Users

**Value Proposition**: "v0.4.0 makes AILANG easier to use"
- Auto-capability inference (less typing)
- Native lists (familiar syntax)
- Better error messages (helpful suggestions)
- Comprehensive tutorials (faster onboarding)

### For AI Models

**Value Proposition**: "v0.4.0 reduces cognitive load for codegen"
- Fewer required flags (AUTO_CAPS)
- More intuitive list syntax (matches Python/JS)
- Error recovery with `?` operator (common pattern)
- List comprehensions (familiar from Python/Haskell)

### For Contributors

**Value Proposition**: "v0.4.0 improves codebase quality"
- 85%+ examples working (better test coverage)
- Comprehensive documentation (easier contributions)
- Well-tested stdlib (stable foundation)

---

## Changelog Template (v0.4.0)

```markdown
## [v0.4.0] - 2026-Q1 - Developer Experience & Language Completeness

### Added - Language Features
- **Capability Inference (AUTO_CAPS)** - No more `--caps` flags
- **Native List Type** - `[1, 2, 3]` syntax with builtin functions
- **Error Propagation Operator `?`** - Rust-style early returns
- **List Comprehensions** - `[x * 2 | x <- xs, x > 0]` syntax

### Added - Standard Library
- **std/json** - JSON parsing and serialization
- **std/cli** - Command-line argument parsing
- **std/string** - Extended string operations
- **std/result** - Result type utilities

### Improved - REPL
- **Multi-Line Editing** - True multi-line input with smart indentation
- **Better Error Messages** - Contextual help with typo suggestions

### Fixed
- **18 Example Files** - Now 85%+ examples passing (56/66)
- **Multiple Bug Fixes** - See detailed changelog

### Benchmark Results (M-EVAL)
- **Success Rate**: 75%+ (target achieved)
- **Example Success**: 85%+ (target achieved)
- **Total Cost**: $X.XX for full baseline
```

---

**Document Status**: ✅ **APPROVED** - Ready for v0.4.0 sprint planning
**Next Review**: After v0.4.0 release (Q1 2026)
**Authors**: Claude Code (based on 20251013 audit)
**Date**: 2025-10-15
