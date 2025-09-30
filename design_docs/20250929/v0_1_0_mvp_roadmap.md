# AILANG v0.1.0 MVP Roadmap

## Executive Summary

This document synthesizes feedback from Claude Sonnet 4.5 and GPT-5, assesses current implementation status (v0.0.7), and defines a focused MVP for v0.1.0 that proves AILANG's "one-shot + secure by construction" thesis.

**Primary Goal**: Run single `.ail` files hermetically with explicit effects, resource budgets, and reproducible artifacts.

---

## Current Implementation Status (v0.0.8)

### 🆕 Recent Progress (September 30, 2025)

**REPL Fixed**:
- ✅ Fixed "Empty expression" bug by updating `Elaborate()` to handle `prog.File.Statements`
- ✅ Added `Intrinsic` support to ANF verifier for arithmetic operators
- ✅ Integrated `OpLowering` pass into REPL pipeline
- ✅ All basic expressions now work: `42`, `1 + 2`, `"hello" ++ "world"`, etc.

**Module System Verified**:
- ✅ `func` declarations work in files (proven by test_export_func.ail)
- ✅ `module`/`import` statements work for basic cases
- ✅ Export/import mechanism functional

**Metrics Updated**:
- Corrected test coverage from inflated 31.3% to actual 24.9%
- Updated LOC count from 7,860 to accurate 23,384
- Identified critical gaps: parser (0% tests), eval (14.9%), types (15.4%)

### ✅ What We Have (Working)

**Currently at 24.9% test coverage with ~23,384 LOC** *(Updated 2025-09-30)*

1. **Type System** (Foundation - 15.4% coverage)
   - Hindley-Milner inference with let-polymorphism (~6,815 LOC)
   - Type classes: Num, Eq, Ord, Show with dictionary-passing
   - Row-polymorphic records with principal row unification
   - Value restriction for sound polymorphism
   - Kind system (Effect, Record, Row)
   - Linear capability capture analysis

2. **Module System** (v0.0.6-v0.0.7)
   - Path resolution (relative, stdlib, project) (~405 LOC)
   - Dependency management with cycle detection (~607 LOC)
   - Module caching (thread-safe, concurrent)
   - Import conflict detection (IMP011)
   - **Structured error reporting** (v0.0.7, ~680 LOC)
   - **Golden file testing** (byte-for-byte reproducibility)
   - **CLI JSON output** (`--json`, `--compact` flags)

3. **Evaluation** (Working - 14.9% coverage)
   - Tree-walking interpreter (~3,362 LOC)
   - Lambda expressions with closures
   - Arithmetic, strings, conditionals, let bindings
   - Records (creation + field access)
   - Lists, built-ins (print, show, toText)

4. **REPL** (NOW Operational - Fixed v0.0.8)
   - Professional interactive REPL (~1,351 LOC)
   - **Recent fix**: Elaborate() now handles bare expressions, Intrinsic support added, OpLowering pass integrated
   - Arrow key history, tab completion, persistent history
   - Type class resolution with dictionary-passing
   - Module import system
   - Rich diagnostic commands (`:type`, `:instances`, `:dump-core`)
   - Auto-imports std/prelude

5. **Parser** (Nearly Complete - 0% test coverage ⚠️)
   - Recursive descent + Pratt parsing (~1,436 LOC)
   - ✅ Expressions, let bindings, if-then-else
   - ✅ Binary/unary operators (spec-compliant precedence)
   - ✅ Lambda expressions (`\x.` syntax, currying)
   - ✅ Record field access (correct precedence)
   - ✅ Module declarations, import statements
   - ⚠️ Pattern matching parsed but not evaluated
   - ❌ `?` operator, effect handlers, tuples

6. **AI-First Features** (v0.0.4-v0.0.7)
   - Schema registry (versioned JSON, ~176 LOC, 88.5% coverage)
   - Error JSON encoder (~192 LOC, 50.0% coverage ⚠️)
   - Test reporter (~95 LOC, 95.7% coverage)
   - Effects inspector stub (~41 LOC)
   - Golden test framework (~310 LOC)
   - **Note**: Coverage ranges 50-95.7%, not 100%

7. **Infrastructure**
   - Lexer (~935 LOC, 57.0% coverage)
   - Error taxonomy (60+ error codes)
   - Manifest system (~390 LOC)
   - CI/CD with automated testing
   - Example verification system

### ⚠️ What's Broken/Missing

**Parser Issues** (MOSTLY FIXED in v0.0.8):
- ✅ `func` declarations work in files (test_export_func.ail passes)
- ✅ `module`/`import` statements work (basic cases proven)
- ❌ `type` definitions not supported
- ❌ Test/property syntax broken

**Not Started**:
- ❌ Effect system (no tracking/inference)
- ❌ Quasiquotes
- ❌ CSP/channels

### 📊 Current Metrics (v0.0.8 - Updated 2025-09-30)
- **Test Coverage**: 24.9% (down from inflated 31.3% claim)
- **Examples**: 22 passing, 24 failing (60 total)
- **Production Code**: ~23,384 lines (3x larger than previously stated)
- **Well-tested**: test (95.7%), manifest (89.9%), schema (88.5%), module (67.7%)
- **Needs tests**: parser (0%), eval (14.9%), types (15.4%), errors (50.0%)

---

## Feature Analysis: What Both AIs Agreed On

### Priority Matrix

| Feature | V4.0 Rating | GPT-5 Priority | Current Status | v0.1.0 MVP? |
|---------|-------------|----------------|----------------|-------------|
| **Effect System** | ⭐⭐⭐⭐⭐ | Critical | ❌ None | ✅ **Core** |
| **Capability Budgets** | ⭐⭐⭐⭐⭐ | Critical | ❌ None | ✅ **Core** |
| **@oneshot Runner** | N/A | Critical | ❌ None | ✅ **Core** |
| **Refinement Types** | ⭐⭐⭐⭐⭐ | High | ❌ None | ✅ **Starter set** |
| **Effect Composition** | ⭐⭐⭐⭐ | High | ❌ None | ✅ **Basic** |
| **Linear/Affine Types** | N/A | High | ❌ None | ⬜ v0.2.0 |
| **Info-Flow Labels** | N/A | High | ❌ None | ⬜ v0.2.0 |
| **Semantic Annotations** | ⭐⭐⭐⭐⭐ | Medium | ❌ None | ⬜ v0.2.0 |
| **Session Types** | N/A | Medium | ❌ None | ⬜ v0.3.0 |
| **Policy DSL** | N/A | Medium | ❌ None | ⬜ v0.2.0 |
| **Gradual Typing** | ⭐⭐⭐⭐ | Low | ❌ None | ⬜ v0.3.0 |

---

## v0.1.0 MVP Scope (REVISED - Conservative & Achievable)

### Design Philosophy (Updated 2025-09-30)

**Revised Goal**: Build a **solid foundation** with comprehensive testing rather than rushing all features.

**Core Principle**: Ship features that are **well-tested and production-ready**, not features that are "mostly done."

**What v0.1.0 Proves**:
1. **Parser is robust** - 80%+ test coverage, no "REPL vs file" discrepancies
2. **Effects are tracked** - Type system enforces effect discipline
3. **Type system is complete** - ADTs, pattern matching, type classes all work
4. **Ready for v0.2.0** - Solid foundation for runtime features

### What's In (Must-Have)

#### A. Parser Testing & Fixes (HIGHEST PRIORITY - 5 days)

**Goal**: Eliminate #1 risk factor (currently 0% test coverage)

**Tasks**:
1. **Write 100+ parser tests** (3 days)
   - Expression parsing (arithmetic, lambdas, let, if-then-else)
   - Module/import parsing
   - Function declarations
   - Type definitions
   - Pattern matching syntax
   - Error recovery

2. **Add ADT support** (2 days)
   - Sum types: `type Option[a] = Some(a) | None`
   - Product types: `type Point = {x: float, y: float}`
   - Recursive types: `type List[a] = Cons(a, List[a]) | Nil`
   - Tuple syntax: `(1, "hello", true)` with type `(int, string, bool)`

**Lines**: ~500 new (tests) + ~200 modified (parser)
**Acceptance**:
- Parser test coverage >80%
- All ADT examples parse and type-check correctly
- No more "works in REPL, fails in files"

#### B. Pattern Matching Evaluation (3 days)

**Goal**: Make pattern matching actually work (currently parsed but not evaluated)

**Tasks**:
1. **Implement evaluation** (2 days)
   - Literal patterns: `42`, `"hello"`, `true`
   - Constructor patterns: `Some(x)`, `Cons(head, tail)`
   - Tuple patterns: `(x, y, z)`
   - Wildcard/variable patterns: `_`, `x`

2. **Exhaustiveness checking** (1 day)
   - Warn on non-exhaustive matches
   - Suggest missing patterns

**Lines**: ~300 new (eval) + ~100 new (exhaustiveness)
**Acceptance**:
```ailang
type Option[a] = Some(a) | None

match option {
  Some(x) => x,
  None => 0
}
```
Works correctly and warns if `None` case is missing.

#### C. Effect System - Type Level Only (4 days)

**Goal**: Track effects in types, **NO runtime enforcement yet**

**Core Effects**:
```ailang
type Effect = IO | FS | Net | Clock | Rand
```

**Function Signatures**:
```ailang
func readFile(path: string) -> Result[string, string] ! {FS}
func httpGet(url: string) -> Result[string, string] ! {Net}
func print(s: string) -> () ! {IO}
```

**Effect Inference**:
```ailang
// Compiler infers {FS, Net} from body
func process(path: string) -> Result[string] ! {FS, Net} {
  let config = readFile(path)?
  let data = httpGet(config)?
  Ok(data)
}
```

**Implementation**:
- Effect syntax in parser (~100 LOC)
- Effect tracking in type checker (~300 LOC)
- Effect propagation (~200 LOC)
- Export signature enforcement (~100 LOC)

**Lines**: ~700 new
**Acceptance**:
- Effect mismatch = compile error
- Calling `readFile` without `! {FS}` is rejected
- **NO runtime enforcement** (deferred to v0.2.0)

#### D. Minimal Stdlib (2 days)

**Goal**: Essential functions for examples and testing

**Modules**:
```ailang
std/prelude    -- Num, Eq, Ord, Show (already exists)
std/list       -- map, filter, fold, length, head, tail
std/string     -- concat (++), length, substring
std/option     -- Option[a], map, flatMap, getOrElse
std/io         -- print (stub implementation with effects)
```

**Implementation**: ~600 LOC
**Acceptance**: Example programs use stdlib functions

#### E. Fix Examples & Documentation (2 days)

**Goal**: Make existing examples work and document accurately

**Tasks**:
1. **Fix broken examples** (~1 day)
   - Update to use new ADT syntax
   - Add pattern matching examples
   - Add effect-annotated examples

2. **Update documentation** (~1 day)
   - README.md with accurate metrics
   - CLAUDE.md with current status
   - Example comments explaining effects

**Lines**: ~500 modified (examples + docs)
**Acceptance**: >35 examples passing (up from 22)

---

### 🚫 EXPLICITLY DEFERRED to v0.2.0

**These features are OUT OF SCOPE for v0.1.0:**

#### Refinement Types
- **Why defer**: Requires SMT solver or extensive runtime guards
- **Complexity**: ~400-800 LOC + external dependencies
- **Not essential** for core thesis proof

#### Capability Budgets
- **Why defer**: Requires runtime instrumentation
- **Complexity**: ~600-1000 LOC + testing overhead
- **Dependencies**: Needs effect runtime first

#### Effect Composition (Runtime)
- **Why defer**: Needs runtime effect handlers (retry, timeout)
- **Note**: Syntax can be added in parser, implementation deferred

#### @oneshot Hermetic Bundler
- **Why defer**: Complex tooling (bundling, SBOM, signing)
- **For v0.1.0**: Basic `ailang run file.ail` is sufficient
- **For v0.2.0**: Add hermetic execution, signing, SBOM generation

---

### Timeline Summary

**Total Time**: 13 days (~2.6 weeks)

| Week | Task | Days |
|------|------|------|
| Week 1 | Parser tests + ADT support | 5 |
| Week 2 | Pattern matching + Effect system | 7 |
| Week 3 | Stdlib + Examples + Docs | 3 (partial) |

**Milestone**: Ship v0.1.0 with **solid foundations** for v0.2.0 runtime features

---

## Implementation Plan (REVISED)

### Sprint Structure

**Philosophy**: Quality over quantity. Ship robust features, not rushed features.

### Week 1: Parser Foundation (5 days)

**Priority**: HIGHEST - Parser has 0% test coverage

**Tasks**:
- Day 1-3: Write 100+ parser tests (expressions, modules, functions, patterns)
- Day 4-5: Add ADT support (sum types, product types, recursive types, tuples)

**Deliverable**: Parser test coverage >80%, ADTs work
**Blocker Removed**: "Works in REPL, fails in files"

### Week 2: Semantics (7 days)

**Priority**: HIGH - Complete language semantics

**Tasks**:
- Day 1-2: Pattern matching evaluation (literals, constructors, tuples, wildcards)
- Day 3: Exhaustiveness checking
- Day 4-7: Effect type system (parsing, tracking, propagation, enforcement)

**Deliverable**: Pattern matching works, effects tracked in types
**Foundation**: Type-level effect discipline proven

### Week 3: Polish (3 days)

**Priority**: MEDIUM - Make it usable

**Tasks**:
- Day 1-2: Stdlib modules (list, string, option, io stubs)
- Day 3: Fix examples + update documentation

**Deliverable**: >35 examples passing, accurate docs

---

## Total Code Estimate (REVISED)

| Component | New Code | Modified Code | Test Code |
|-----------|----------|---------------|-----------|
| Parser tests | - | - | ~500 LOC |
| ADT support | ~200 LOC | ~200 LOC | - |
| Pattern matching eval | ~300 LOC | ~100 LOC | - |
| Exhaustiveness check | ~100 LOC | - | - |
| Effect type system | ~700 LOC | ~200 LOC | - |
| Stdlib | ~600 LOC | - | - |
| Examples + docs | - | ~500 LOC | - |
| **Total** | **~1,900 new** | **~1,000 modified** | **~500 tests** |

**Starting Point (v0.0.8)**: 23,384 LOC at 24.9% coverage
**Target (v0.1.0)**: ~25,900 LOC at >35% coverage

**Note**: Much smaller scope than original plan (1,900 vs 6,200 new LOC), but **actually achievable** in 13 days

---

## 5 Demo Programs (Ship with v0.1.0)

### 1. `file_to_webhook.ail`
Read file → summarize → POST to webhook
```ailang
@oneshot
@cli "--file Path --webhook Url"
func main(args: {file: Path, webhook: Url})
  -> Result[{summary: NonEmptyString}, string]
  ! {FS with budget(reads: 1, bytes: 5.MB),
     Net with timeout(3.s) with retry(2, Exponential)}
{
  let text = readFile(args.file)?
  let summary = summarize(text)?
  httpPost(args.webhook, json{summary})?
  Ok({summary})
}
```

### 2. `safe_divide.ail`
Division with refinement types
```ailang
func divide(a: int, b: NonZero) -> int {
  a / b  -- Guaranteed no divide-by-zero
}

test "safe division" {
  assert divide(10, 2) == 5
  -- divide(10, 0) fails at runtime with clear error
}
```

### 3. `budget_guard.ail`
Exceed request budget
```ailang
@oneshot
func main(args: {urls: [Url]})
  -> Result[int] ! {Net with budget(requests: 5)}
{
  -- Trying 10 URLs with budget of 5
  let responses = args.urls.map(httpGet)
  Ok(responses.length)
}
-- Error: BudgetExceeded{kind: "Net.requests", limit: 5, used: 5}
```

### 4. `retry_timeout.ail`
Declarative robustness
```ailang
func fetchFlaky(url: Url) -> Result[Data]
  ! {Net with retry(3, Exponential) with timeout(2.s)}
{
  httpGet(url)  -- Retries on failure, times out after 2s
}
```

### 5. `pure_etl.ail`
FS-only pipeline with strict budgets
```ailang
@oneshot
@cli "--input Path --output Path"
func main(args: {input: Path, output: Path})
  -> Result[{processed: int}, string]
  ! {FS with budget(reads: 1, writes: 1, bytes: 50.MB)}
{
  let data = readFile(args.input)?
  let transformed = process(data)
  writeFile(args.output, transformed)?
  Ok({processed: length(transformed)})
}
```

---

## Acceptance Criteria (Go/No-Go)

### 1. Hello One-Shot ✅
```bash
$ ailang build --oneshot hello.ail
$ ailang run hello.airun --name "World"
Hello, World!
$ ls
hello.airun hello.sbom.json hello.ledger.json
```

### 2. Effect Discipline ✅
```ailang
func badRead() -> Config {
  readFile("config.json")  -- COMPILE ERROR
}
```
Error: `Effect mismatch: function uses {FS} but declares no effects`

### 3. Budgets Enforced ✅
```bash
$ ailang run budget_guard.airun --urls url1,...url10
Error: BudgetExceeded{kind: "Net.requests", limit: 5, used: 5}
Exit code: 1
```

### 4. Refinement Safety ✅
```ailang
func divide(a: int, b: NonZero) -> int
-- divide(10, 0) fails with: "Refinement violation: NonZero requires x != 0"
```

### 5. Retry + Timeout ✅
Flaky endpoint succeeds on retry 2, hanging endpoint fails at 5s

### 6. Reproducible Traces ✅
```bash
$ ailang run main.airun --seed 42 > out1.json
$ ailang run main.airun --seed 42 > out2.json
$ diff out1.json out2.json
# No differences
```

---

## Success Metrics for v0.1.0

### Technical
- [ ] All 5 demo programs build and run
- [ ] Test coverage: >40% (from 31.3%)
- [ ] Examples passing: >35 (from 20)
- [ ] Zero panics in production paths
- [ ] Deterministic output (100 runs)

### Developer Experience
- [ ] Effect errors suggest exact fix
- [ ] Budget errors show limit vs usage
- [ ] Refinement violations show constraint
- [ ] REPL and file execution have parity

### Documentation
- [ ] "Why Effects" explainer
- [ ] "Why Budgets" explainer
- [ ] "Write Your First Oneshot" tutorial
- [ ] Stdlib API reference
- [ ] Migration guide from v0.0.7

### Security
- [ ] `lint-sec` catches unbounded usage
- [ ] Budgets prevent resource exhaustion
- [ ] Signed artifacts verify at runtime
- [ ] SBOM includes all dependencies

---

## Deferred to Later Versions

### v0.2.0 (Next)
- Semantic Annotations (`@intent`, `@requires`, `@ensures`)
- Linear/Affine Types (resource lifecycle)
- Info-Flow Labels (PII/Secret tracking)
- Policy DSL (org-level constraints)
- Example-Driven Development

### v0.3.0 (Later)
- Session Types (protocol verification)
- Structured Concurrency (nurseries)
- Gradual Typing (`@prototype`)
- Proof-Carrying Refinements (SMT)

### v0.4.0+ (Future)
- Deterministic Math (IEEE-754 exact)
- Units of Measure
- Supply-Chain Receipts (Sigstore)
- LSP with AI hints
- Package manager

---

## Comparison: v0.1.0 vs Full v4.0

| Feature | v0.1.0 MVP | v4.0 Full |
|---------|------------|-----------|
| **Effects** | 5 basic | Full effect rows |
| **Refinements** | 4 built-ins, runtime | User-defined + SMT |
| **Budgets** | Basic counters | Advanced tracking |
| **Composition** | 3 combinators | Full library |
| **@oneshot** | Core runner | Policy DSL + receipts |
| **Type System** | HM + type classes | + Linear/affine |
| **Concurrency** | None | Structured |
| **Security** | Basic lint | Info-flow + session |
| **Tooling** | CLI + REPL | LSP + package manager |

**Ship Strategy**:
- v0.1.0 proves thesis
- v0.2.0 adds safety
- v0.3.0 adds power
- v0.4.0 completes vision

---

## Conclusion

v0.1.0 is the **minimal viable proof** that AILANG delivers:

> **"Zero-boilerplate, type-safe, resource-bounded, reproducible single-file programs"**

**Focus**:
1. ✅ Fix parser (files = REPL)
2. ✅ Add effects (explicit, inferred, enforced)
3. ✅ Add budgets (resource safety)
4. ✅ Add @oneshot (hermetic execution)
5. ✅ Add refinements (type constraints)

**Useful For**:
- AI agents: Safe code generation
- Scripts: Better than Python with types
- Serverless: Hermetic functions
- Research: Deterministic experiments

**Ship v0.1.0**, iterate toward v4.0 full vision.

---

*Synthesized from Claude Sonnet 4.5, GPT-5 feedback, and v0.0.7 implementation*
*September 29, 2025*