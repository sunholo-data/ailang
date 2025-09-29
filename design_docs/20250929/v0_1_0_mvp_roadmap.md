# AILANG v0.1.0 MVP Roadmap

## Executive Summary

This document synthesizes feedback from Claude Sonnet 4.5 and GPT-5, assesses current implementation status (v0.0.7), and defines a focused MVP for v0.1.0 that proves AILANG's "one-shot + secure by construction" thesis.

**Primary Goal**: Run single `.ail` files hermetically with explicit effects, resource budgets, and reproducible artifacts.

---

## Current Implementation Status (v0.0.7)

### ✅ What We Have (Working)

**Already at 31.3% test coverage with ~7,860 LOC!**

1. **Type System** (Complete)
   - Hindley-Milner inference with let-polymorphism (~5,000 LOC)
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

3. **Evaluation** (Working)
   - Tree-walking interpreter (~700 LOC)
   - Lambda expressions with closures
   - Arithmetic, strings, conditionals, let bindings
   - Records (creation + field access)
   - Lists, built-ins (print, show, toText)

4. **REPL** (Fully Operational)
   - Professional interactive REPL (~850 LOC)
   - Arrow key history, tab completion, persistent history
   - Type class resolution with dictionary-passing
   - Module import system
   - Rich diagnostic commands (`:type`, `:instances`, `:dump-core`)
   - Auto-imports std/prelude

5. **Parser** (Nearly Complete)
   - Recursive descent + Pratt parsing (~1,200 LOC)
   - ✅ Expressions, let bindings, if-then-else
   - ✅ Binary/unary operators (spec-compliant precedence)
   - ✅ Lambda expressions (`\x.` syntax, currying)
   - ✅ Record field access (correct precedence)
   - ✅ Module declarations, import statements
   - ⚠️ Pattern matching parsed but not evaluated
   - ❌ `?` operator, effect handlers, tuples

6. **AI-First Features** (v0.0.4-v0.0.7)
   - Schema registry (versioned JSON, ~145 LOC)
   - Error JSON encoder (~190 LOC)
   - Test reporter (~206 LOC)
   - Effects inspector stub (~41 LOC)
   - Golden test framework (~309 LOC)
   - **100% test coverage** for these packages!

7. **Infrastructure**
   - Lexer (~550 LOC, all tests passing)
   - Error taxonomy (60+ error codes)
   - Manifest system (~390 LOC)
   - CI/CD with automated testing
   - Example verification system

### ⚠️ What's Broken/Missing

**Parser Issues** (block file execution):
- ❌ `func` declarations work in REPL, fail in files
- ❌ `module`/`import` statements fragile
- ❌ `type` definitions not supported
- ❌ Test/property syntax broken

**Not Started**:
- ❌ Effect system (no tracking/inference)
- ❌ Quasiquotes
- ❌ CSP/channels

### 📊 Current Metrics
- **Test Coverage**: 31.3% (was 19.7% in v0.0.6!)
- **Examples**: 20 passing, 23 failing
- **Production Code**: ~7,860 lines
- **Well-tested**: test (95.7%), schema (87.9%), parser (75.8%), errors (75.9%)

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

## v0.1.0 MVP Scope (The "Ship It" Version)

### Design Philosophy

**GPT-5's Wisdom**: "Smallest set that proves AILANG's 'one-shot + secure by construction' thesis"

**Core Thesis**: A single `.ail` file can be:
1. **Type-safe** with explicit effects
2. **Resource-bounded** with compile-time budgets
3. **Reproducible** with deterministic execution
4. **Hermetic** with signed artifacts
5. **Zero-boilerplate** via `@oneshot`

### What's In (Must-Have)

#### A. Fix Parser (Make Files Work Like REPL)

**Goal**: Unblock file execution

**Tasks**:
- ✅ Fix `func` declarations in files (~2 days)
- ✅ Fix `module` declarations (~1 day)
- ✅ Fix `import` statements (~1 day)
- ✅ Fix `type` definitions (ADTs) (~2 days)
- ✅ Complete pattern matching evaluation (~2 days)

**Lines**: ~500 modified
**Acceptance**: All REPL examples work in files

#### B. Effect System (MVP)

**Goal**: Explicit effects with inference

**Core Effects**:
```ailang
type Effect = IO | FS | Net | Clock | Rand
```

**Function Signatures**:
```ailang
func readFile(path: Path) -> Result[string, IOError] ! {FS}
func httpGet(url: Url) -> Result[Response, NetError] ! {Net}
func now() -> Timestamp ! {Clock}
```

**Effect Inference** (internal only):
```ailang
// Infers {FS, Net} from function body
func process(path: Path) -> Result[Data] ! {FS, Net} {
  let config = readFile(path)?
  let data = httpGet(config.url)?
  Ok(transform(data))
}
```

**Implementation**:
- Effect tracking in type checker (~300 LOC)
- Effect propagation (~200 LOC)
- Export signature enforcement (~100 LOC)
- Error messages with suggestions (~200 LOC)

**Lines**: ~800 new
**Acceptance**: Effect mismatch = compile error with suggestion

#### C. Refinement Types (Starter Set)

**Goal**: Type-level constraints

**Built-in Refinements**:
```ailang
type PositiveInt = int where (x > 0)
type NonZero = int where (x != 0)
type NonEmptyString = string where (length(x) > 0)
type Percentage = float where (x >= 0.0 && x <= 100.0)
```

**Runtime Guards** (compile-time in v0.2.0):
```ailang
func divide(a: int, b: NonZero) -> int {
  // Compiler inserts: if b == 0 then panic
  a / b
}
```

**Implementation**:
- Refinement type definitions (~100 LOC)
- Guard insertion pass (~200 LOC)
- std/refinement helpers (~100 LOC)

**Lines**: ~400 new
**Acceptance**: `divide(x, 0)` fails at runtime with clear message

#### D. Capability Budgets (MVP)

**Goal**: Resource limits in type signatures

**Budget Syntax**:
```ailang
func processAll(items: [Item]) -> [Result]
  ! {Net with budget(requests: 100, bandwidth: 1.MB),
     Clock with budget(wall_time: 30.seconds)}
{
  items.map(fetchAndProcess)
}
```

**Runtime Enforcement**:
```ailang
type BudgetExceeded = {
  kind: string,      -- "Net.requests"
  limit: int,        -- 100
  used: int          -- 101
}
```

**Implementation**:
- Budget counters in runtime (~200 LOC)
- Budget tracking per effect (~200 LOC)
- Typed errors (~200 LOC)

**Lines**: ~600 new
**Acceptance**: Exceeding budget stops execution with structured error

#### E. Effect Composition (MVP)

**Goal**: Declarative retry/timeout

**Core Combinators**:
```ailang
! {Net with timeout(5.seconds)}
! {Net with retry(3, Exponential)}
! {FS with trace(Debug)}
```

**Implementation**:
- Combinator parsing (~100 LOC)
- Runtime wrappers (~200 LOC)
- std/effects module (~100 LOC)

**Lines**: ~400 new
**Acceptance**: `retry(3)` retries 3 times on failure

#### F. @oneshot Runner (The Centerpiece)

**Goal**: Hermetic single-file execution

**Syntax**:
```ailang
@oneshot
@cli "--file Path --webhook Url?"
func main(args: {file: Path, webhook: Option[Url]})
  -> Result[{summary: NonEmptyString}, string]
  ! {FS with budget(reads: 5, writes: 2, bytes: 5.MB),
     Net with timeout(3.s) with retry(2, Exponential),
     Clock with budget(wall_time: 5.s)}
{
  let text = readFile(args.file)?
  let summary = summarize(text)?
  match args.webhook {
    Some(url) => httpPost(url, json{summary})?,
    None => ()
  }
  Ok({summary})
}
```

**Build & Run**:
```bash
$ ailang build --oneshot main.ail
# Produces:
# - main.airun (signed hermetic artifact)
# - main.sbom.json (dependencies + stdlib)
# - main.ledger.json (compiler decisions + budgets)

$ ailang run main.airun --file notes.txt
# Output: {"summary": "..."}
```

**Implementation**:
- CLI parser from `@cli` spec (~200 LOC)
- Hermetic bundler (~200 LOC)
- SBOM generation (~100 LOC)
- Decision ledger export (~100 LOC)
- Signing (dev key) (~100 LOC)
- Runner with budget enforcement (~100 LOC)

**Lines**: ~800 new
**Acceptance**: One artifact, deterministic output

#### G. Minimal Stdlib

**Modules**:
```ailang
std/io         -- print, readText, writeText
std/net        -- httpGet, httpPost, jsonEncode, jsonDecode
std/time       -- Duration, sleep, now
std/rand       -- seed, nextInt, nextFloat
std/refinement -- positive, nonzero, nonEmpty
std/effects    -- retry, timeout, trace
```

**Lines**: ~800 new
**Acceptance**: All demos use stdlib functions

#### H. Tooling

**CLI Commands**:
```bash
ailang build file.ail           # Typecheck + emit
ailang build --oneshot file.ail # Hermetic bundle
ailang run file.airun           # Execute artifact
ailang fmt file.ail             # Format (defer to v0.2.0)
ailang test                     # Run tests
ailang lint-sec                 # Security lint
```

**REPL Commands** (add to existing):
```
:effects <expr>    -- Show effects without eval
:budget <expr>     -- Show budget estimates
:oneshot <fn>      -- Test oneshot function
```

**Lines**: ~400 new
**Acceptance**: All commands work as documented

---

## Implementation Plan

### Sprint Structure

**Note**: You ship fast, so no dates. Just order:

### 1. Parser Fixes (First)
- Fix `func` in files
- Fix `module` declarations
- Fix `import` statements
- Fix `type` definitions
- Complete pattern matching
- **Blocker**: Without this, nothing else works in files
- **Lines**: ~500 modified
- **Acceptance**: REPL examples work in files

### 2. Effect System (Second)
- Effect tracking in type checker
- Effect propagation
- Export enforcement
- Error messages
- **Critical**: Core thesis depends on this
- **Lines**: ~800 new
- **Acceptance**: Effect discipline enforced

### 3. Refinements + Budgets (Third)
- Refinement type definitions
- Runtime guards
- Budget syntax parsing
- Budget runtime enforcement
- **Important**: Proves resource safety
- **Lines**: ~1,000 new
- **Acceptance**: Constraints enforced

### 4. Effect Composition + Stdlib (Fourth)
- Combinator syntax
- timeout/retry/trace implementation
- std/io, std/net, std/time, std/refinement, std/effects
- **Enabling**: Makes effects usable
- **Lines**: ~1,200 new
- **Acceptance**: Declarative patterns work

### 5. @oneshot Runner (Fifth)
- `@oneshot` annotation parsing
- CLI spec parsing (`@cli`)
- Hermetic bundler
- SBOM + ledger generation
- Signing + runner
- **Centerpiece**: The "wow" feature
- **Lines**: ~800 new
- **Acceptance**: Hermetic execution works

### 6. Polish + Demos (Sixth)
- Security lint (`lint-sec`)
- 5 demo programs
- Documentation
- Bug fixes
- **Final**: Ship-ready
- **Lines**: ~500 new
- **Acceptance**: All demos pass

---

## Total Code Estimate

| Component | New Code | Modified Code |
|-----------|----------|---------------|
| Parser fixes | - | ~500 LOC |
| Effect system | ~800 LOC | ~200 LOC |
| Refinements | ~400 LOC | - |
| Budgets | ~600 LOC | - |
| Effect composition | ~400 LOC | - |
| @oneshot runner | ~800 LOC | - |
| Stdlib | ~800 LOC | - |
| Tooling | ~400 LOC | ~200 LOC |
| Tests | ~1,500 LOC | - |
| **Total** | **~6,200 new** | **~900 modified** |

**Starting Point**: 7,860 LOC at 31.3% coverage
**Target**: ~14,000 LOC at >40% coverage

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