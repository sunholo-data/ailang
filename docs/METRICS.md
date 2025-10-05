# AILANG Metrics & Statistics

**Current Version**: v0.3.0 (Clock & Net Effects + Type System Fixes)
**Last Updated**: October 5, 2025

---

## 🚨 Notice: Historical Documentation

This document contains **historical metrics from v0.1.0**. For current v0.3.0 status, see:

- **[CHANGELOG.md](../CHANGELOG.md)** - Complete version history with LOC counts
- **[examples/STATUS.md](../examples/STATUS.md)** - Current example pass rate (48/66 = 72.7%)
- **[docs/reference/implementation-status.md](reference/implementation-status.md)** - Current component status

---

## Quick Stats (v0.3.0)

### Code Statistics
- **Implementation LOC**: ~30,000+ (estimated, includes v0.2.0 + v0.3.0 additions)
- **Test Coverage**: 27.9% (as of latest badge)
- **Example Success Rate**: 48/66 passing (72.7%)

### Major Milestones Shipped
- ✅ v0.1.0: Type system, REPL, parser (~27,610 LOC)
- ✅ v0.2.0: Module execution, effects (IO, FS) (~1,900 LOC)
- ✅ v0.3.0-alpha2: Recursion, blocks, records (~1,800 LOC)
- ✅ v0.3.0-alpha4: Clock & Net effects (~600 LOC)
- ✅ v0.3.0: Type system fixes (modulo, float eq) (~100 LOC)

### What Works Now (v0.3.0)
- ✅ **Module execution** - Run .ail files with `func`, `import`, `export`
- ✅ **Effect system** - IO, FS, Clock, Net with capability security
- ✅ **Recursion** - Self-recursive and mutually-recursive functions
- ✅ **Block expressions** - `{ stmt1; stmt2; result }`
- ✅ **Records** - Literals `{name: "Alice", age: 30}` and field access `person.name`
- ✅ **Type system** - Hindley-Milner inference, type classes, ADTs
- ✅ **REPL** - Professional interactive development environment

### What Doesn't Work (v0.3.0)
- ❌ Record update syntax `{r | field: val}`
- ❌ Pattern guards (parsed but not evaluated)
- ❌ Error propagation operator `?`
- ❌ Deep let nesting (4+ levels)
- ❌ Typed quasiquotes
- ❌ CSP concurrency

---

## Detailed Evolution

### v0.1.0 → v0.3.0 Growth

| Milestone | LOC Added | Key Features |
|-----------|-----------|--------------|
| v0.1.0 | 27,610 | Type system, REPL, parser, module type-checking |
| v0.2.0 | +1,900 | Module execution runtime, effect system (IO, FS) |
| v0.3.0-alpha2 | +1,800 | Recursion, block expressions, basic records |
| v0.3.0-alpha4 | +600 | Clock & Net effects with security hardening |
| v0.3.0 | +100 | Type fixes (modulo, float comparison) |
| **Total** | **~32,010** | **Fully functional language** |

### Example Success Rate Evolution

| Version | Passing | Total | Rate |
|---------|---------|-------|------|
| v0.1.0 | 12 | 47 | 25.5% |
| v0.2.0 | ~30 | ~55 | ~55% |
| v0.3.0 | 48 | 66 | 72.7% |

**Note**: Total examples increased as new features were added.

### Test Coverage

| Package | v0.1.0 Coverage | Status |
|---------|-----------------|--------|
| `types` | 20.3% | Core type system |
| `eval` | 15.6% | Evaluator |
| `parser` | 75.8% | Parser |
| `elaborate` | 66.4% | Core AST elaboration |
| `schema` | 87.9% | Error schemas |
| Overall | **24.8%** → **27.9%** | Improving |

---

## Historical Metrics (v0.1.0)

*The following sections document v0.1.0 metrics for historical reference.*

### Package Breakdown (v0.1.0)

| Package | LOC | Purpose | Status (v0.1.0) |
|---------|-----|---------|-----------------|
| `types` | 7,291 | Type system, inference, unification | ✅ Complete |
| `eval` | 3,712 | Evaluator & runtime | ⚠️ Partial (non-module only) |
| `parser` | 2,656 | Parser & AST construction | ✅ Complete (some limitations) |
| `elaborate` | 2,059 | Surface AST → Core AST elaboration | ✅ Complete |
| `pipeline` | 1,496 | Compilation pipeline orchestration | ✅ Complete |
| `link` | 1,418 | Dictionary linking for type classes | ✅ Complete |
| `repl` | 1,351 | Interactive REPL | ✅ Complete |
| `ast` | 1,298 | AST definitions | ✅ Complete |
| `module` | 1,030 | Module resolution & validation | ✅ Complete (type-checking) |
| `lexer` | 978 | Tokenization | ✅ Complete |

**v0.2.0 additions**:
- `runtime` | ~1,200 LOC | Module execution runtime
- `effects` | ~700 LOC | Effect system (IO, FS)

**v0.3.0 additions**:
- Recursion support in `eval` | ~300 LOC
- Block expression desugaring | ~10 LOC
- Record types in `types` | ~400 LOC
- Clock effect | ~110 LOC
- Net effect | ~355 LOC
- Type fixes | ~100 LOC

### Development Velocity

**v0.0.1 → v0.1.0** (6 days):
- Implementation: ~20,000 LOC
- Tests: ~8,000 LOC
- Documentation: ~3,000 lines
- Average velocity: ~144 LOC/hour

**v0.1.0 → v0.2.0** (3.5 weeks planned, shipped in ~2 weeks):
- Implementation: ~1,900 LOC
- Module execution runtime + effects

**v0.2.0 → v0.3.0** (2 weeks planned, shipped in ~1 day):
- Implementation: ~2,500 LOC
- Recursion + blocks + records + Clock/Net + type fixes

### Git Activity

**Commit velocity**:
- v0.1.0 development: ~150 commits, ~25 commits/day
- v0.2.0 development: ~80 commits
- v0.3.0 development: ~40 commits

**Contributors**: 1 (initial development phase)

---

## Performance (Informal)

### REPL Performance (v0.3.0)
- Startup: <100ms (cold start)
- Simple expression: <5ms (`1 + 2`)
- Type inference: <10ms (`let double = \x. x * 2 in double(21)`)
- Complex type: <50ms (nested lambdas with constraints)

### File Execution (v0.3.0)
- Small file (<50 lines): <50ms
- Medium file (50-200 lines): <200ms
- Large module (with imports): <500ms
- Recursive algorithms: <100ms (factorial(10), fibonacci(20))

**No formal benchmarking yet** - estimates from development usage.

---

## Build & CI

### CI Checks (GitHub Actions)
- ✅ Test suite (`go test`)
- ✅ Linting (`golangci-lint`)
- ✅ Formatting (`gofmt`)
- ✅ Example verification (`tools/audit-examples.sh`)
- ✅ Build (4 platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)

### Build Times
- Clean build: ~5s
- Incremental build: ~2s
- Full CI pipeline: ~3 minutes

---

## Lessons Learned

### From v0.1.0
- ⚠️ **Parser limitation**: 3-deep let nesting (architectural)
- ⚠️ **Module execution gap**: Required significant runtime work
- ⚠️ **REPL vs files**: Different execution paths (intentional)
- ✅ **Documentation critical**: Honest limitation disclosure prevents frustration

### From v0.2.0
- ✅ **RefCell-based recursion**: OCaml/Haskell pattern works well in Go
- ✅ **Capability security**: Effect system prevents accidental side effects
- ✅ **Block expressions**: Syntactic sugar for let-chains is powerful

### From v0.3.0
- ✅ **Records subsumption**: Practical without full row polymorphism
- ✅ **Clock/Net effects**: Security hardening is complex but essential
- ✅ **Type fixes**: Small fixes have big impact (modulo, float comparison)
- ✅ **Rapid iteration**: v0.3.0 shipped 13 days early!

---

## What's Next

### v0.4.0 (Planned)
- Record update syntax
- Pattern guards
- Enhanced Net effect (custom headers, JSON parsing)
- Environment variable reading
- More developer guides

See [design_docs/planned/](../design_docs/planned/) for detailed roadmaps.

---

## Conclusion

AILANG has grown from a **type system prototype** (v0.1.0) to a **functional programming language with full module execution, recursion, effects, and basic records** (v0.3.0).

**Current state**: Production-quality type system, working runtime, growing standard library.

**Future**: Record extensions, syntactic sugar, concurrency, quasiquotes.

---

*For current implementation status, see [CHANGELOG.md](../CHANGELOG.md)*
*For limitations, see [LIMITATIONS.md](LIMITATIONS.md)*
*For examples, see [examples/STATUS.md](../examples/STATUS.md)*
