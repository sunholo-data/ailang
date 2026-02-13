# AILANG Implementation Status

## Current Version

Check [GitHub Releases](https://github.com/sunholo-data/ailang/releases/latest) for the current stable version and [CHANGELOG.md](https://github.com/sunholo-data/ailang/blob/main/CHANGELOG.md) for detailed release notes.

## Feature Documentation

All implemented features are documented in [Design Documents](/docs/design-docs), which is auto-generated from `design_docs/implemented/`. Each design doc describes a feature, bug fix, or architectural decision.

**Recent versions with design docs:**
- [v0.6.1](/docs/design-docs#v061) — M-VERIFY contracts, multi-executor support, bug fixes
- [v0.6.0](/docs/design-docs#v060) — Semantic caching, SharedIndex, codegen improvements
- [v0.5.10](/docs/design-docs#v0510) — Unified AI providers, type debugging tools
- [v0.5.9](/docs/design-docs#v059) — Flat codegen, GitHub messaging, cyclic type diagnostics

---

## Component Status

### Core Language (Complete)

| Component | Status | Notes |
|-----------|--------|-------|
| **Lexer** | Complete | Unicode, all token types, ~550 LOC |
| **Parser** | Complete | Recursive descent + Pratt parsing, ~1,200 LOC |
| **Type System** | Complete | Hindley-Milner, row polymorphism, type classes |
| **Evaluator** | Complete | Tree-walking interpreter with modules |
| **REPL** | Complete | History, completion, type checking |
| **Effects** | Complete | IO, FS, Net, Clock, Rand, DB, AI |
| **Modules** | Complete | Imports, exports, cross-module calls |
| **Go Codegen** | Complete | Full compilation to Go |

### AI-First Features (Complete)

| Feature | Status | Notes |
|---------|--------|-------|
| **Inline Tests** | Complete | `tests [(input, expected)]` syntax |
| **Structured Errors** | Complete | JSON output, error codes |
| **Schema Registry** | Complete | Versioned JSON schemas |
| **Teaching Prompts** | Complete | `ailang prompt` command |
| **Multi-Model Evals** | Complete | Claude, GPT, Gemini support |
| **Semantic Caching** | Complete | SimHash + neural embeddings |

### Verification (v0.6.1+)

| Feature | Status | Notes |
|---------|--------|-------|
| **Contracts** | Complete | `requires`/`ensures` clauses |
| **Policy Mode** | Complete | Redundant verification |
| **SMT Backend** | Complete | Z3 integration, cross-function inlining |
| **Bounded Recursion** | Complete | `--verify-recursive-depth N` (Dafny-style unrolling) |

---

## Known Limitations

See [Limitations](/docs/reference/limitations) for the full list.

**Active bugs:**
- Polymorphic arithmetic in lambdas panics (use named functions as workaround)
- Pattern guards parsed but not evaluated

**Not yet implemented:**
- String interpolation (use `++` concatenation)
- `?` error propagation operator
- Typed quasiquotes
- CSP concurrency (deferred)

---

## Testing & Quality

[![CI](https://github.com/sunholo-data/ailang/actions/workflows/ci.yml/badge.svg)](https://github.com/sunholo-data/ailang/actions/workflows/ci.yml)

### CI Pipeline

Every push runs comprehensive checks:

| Check | Command | Purpose |
|-------|---------|---------|
| Unit tests | `go test ./...` | 50+ packages with timeout protection |
| Parser tests | `make test-parser` | Golden file comparison |
| Coverage gates | `make gate-all-packages` | Per-package minimum thresholds |
| Golden drift | `make check-golden-drift` | Detect unintended output changes |
| Fuzzing | `make fuzz-parser` | Random input testing |
| Import system | `make test-imports` | Module loading verification |

### Run Locally

```bash
# Quick test
make test

# Full coverage report
make test-coverage

# Coverage percentage
make test-coverage-badge

# Specific test suites
make test-parser          # Parser golden tests
make test-imports         # Import system
make test-stdlib-canaries # Standard library health
```

### Example Validation

Example files serve as integration tests. Check current status in [README](https://github.com/sunholo-data/ailang#readme) or run:

```bash
make verify-examples      # Verify all examples
make update-readme        # Update pass/fail counts
```

---

## See Also

- [Design Documents](/docs/design-docs) — Auto-generated feature documentation
- [Roadmap](/docs/roadmap) — Planned features
- [CHANGELOG](https://github.com/sunholo-data/ailang/blob/main/CHANGELOG.md) — Release history
- [Limitations](/docs/reference/limitations) — Known issues
