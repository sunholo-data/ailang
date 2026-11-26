---
reviewed: 2026-11-26
reviewBy: 2027-05-26
---

# AILANG Implementation Status

## Current Version

Check [GitHub Releases](https://github.com/sunholo-data/ailang/releases/latest) for the current stable version (`v0.64.1` baseline) and [CHANGELOG.md](https://github.com/sunholo-data/ailang/blob/main/CHANGELOG.md) for detailed release notes.

## Feature Documentation

All implemented features are documented in [Design Documents](/docs/design-docs), which is auto-generated from `design_docs/implemented/` and always lists the latest versions.

---

## Component Status

### Core Language (Complete)

| Component | Status | Notes |
|-----------|--------|-------|
| **Lexer** | Complete | Unicode, all token types |
| **Parser** | Complete | Recursive descent + Pratt parsing |
| **Type System** | Complete | Hindley-Milner, row polymorphism, type classes, nominal ADTs |
| **Evaluator** | Complete | Tree-walking interpreter with modules & `--verify-contracts` |
| **REPL** | Complete | History, completion, type/effect inspection (`λ[Debug,IO]>`) |
| **Effects** | Complete | IO, FS, Net, Clock, Rand, Env, Process, Stream, AI, Secret, Trace, SharedMem, SharedIndex, DOM, Msg, Cog |
| **Modules** | Complete | Imports, exports, cross-module constraints & nominal identity |
| **Bytecode VM** | Partial | `run --bytecode`, ~40× faster than the evaluator; 186 of 205 runnable examples run natively on the VM, the rest warn and fall back to the evaluator (`--strict-bytecode` fails with `BC001`) |
| **Go Codegen** | Removed | `compile --emit-go` / `--emit-go-v2` were removed after v0.60.5; run with `ailang run` (`--bytecode` for the VM) |

### AI-First Features (Complete)

| Feature | Status | Notes |
|---------|--------|-------|
| **Inline Tests** | Complete | `tests [(input, expected)]` syntax |
| **Structured Errors** | Complete | JSON output, error codes, `check --fix` & LSP `codeAction` |
| **Semantic Inspection** | Complete | `ailang inspect` (`--format=agent`, `--json`) 1-line-per-symbol X-ray |
| **JSON Deriving** | Complete | `deriving (Json)` for monomorphic records & sum ADTs |
| **Error Propagation** | Complete | `let x = expr?` spine unwrapping for `Result`/`Option` |
| **Schema Registry** | Complete | Versioned JSON schemas |
| **Teaching Prompts** | Complete | `ailang prompt` (active prompt `v0.16.20`) |
| **Multi-Model Evals** | Complete | Claude, GPT, Gemini, local Ollama support |
| **Semantic Caching** | Complete | SimHash + neural embeddings |

### Verification (Z3 SMT, `v0.64.0`+)

| Feature | Status | Notes |
|---------|--------|-------|
| **Contracts** | Complete | `requires`/`ensures` clauses and bounded `forall i: lo..hi => p` |
| **Policy Mode** | Complete | Redundant verification & `--strict` / `--allow-inconclusive` |
| **SMT Backend** | Complete | Z3 `4.16.0` integration, cross-function inlining, `Tuple0` (`()`) & `Tuple2..Tuple8` sorts, `--int-model=int64` overflow mode, NaN/Inf float checks, AILANG-syntax counterexample rendering (`(0, 1)`, `{ a: 1 }`, `Some(42)`) |
| **Bounded Recursion** | Complete | `--verify-recursive-depth N` (default `2`) + HOF lambda unrolling (depth `3`); unbounded recursion is `inconclusive` (no induction yet) |

---

## Known Limitations

See [Known Limitations](/docs/reference/limitations) for the full capability & cliff matrix.

**Not yet implemented:**
- `export import` module re-exports (pending PR `#2475`; emits `PAR_EXPORT_REQUIRES_FUNC`)
- Arbitrary sub-expression `?` propagation (use `let x = expr?` along the return spine)
- User-defined `class`/`instance` typeclasses and higher-kinded type parameters
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
