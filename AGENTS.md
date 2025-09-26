# AILANG Repository Guidelines

This document summarizes the key facts an agent should know while working in this repository. Review it before making changes.

## Project Overview
- **Language focus**: AILANG is a purely functional language optimized for AI-assisted development, emphasizing explicit algebraic effects, typed quasiquotes, CSP/session-type concurrency, and deterministic execution traces. Refer to `design_docs/20250926/initial_design.md` for the conceptual specification.
- **Status**: Lexer, basic parser, AST, and foundational type system are implemented. Effect system, interpreter, and several advanced features are still TODO according to the top-level README.

## Repository Structure & Tooling
- `cmd/ailang/`: Go CLI entry point.
- `internal/`: Core compiler/interpreter packages (lexer, parser, AST, types, effects, eval, etc.). Many subpackages are still under construction.
- `examples/`: Example `.ail` programs.
- `design_docs/20250926/`: Canonical language design references.
- Use `make build`, `make test`, `make fmt`, and `make lint` for common workflows.

## Key Design Details
- **Type system**: Hindley–Milner style with row-polymorphic algebraic effects and capability annotations. Review `initial_design.md` for type/effect constructs and idioms.
- **Row unification**: Reference Go implementation for effect/record row handling lives in `design_docs/20250926/gpt5-reference-code.md`; it defines `Row`, `Subst`, and `UnifyRows` helpers for deterministic effect reasoning.
- **Typeclass dictionaries**: Explicit dictionary passing is the intended elaboration strategy; see the same reference doc for `Class`, `Instance`, and `ElabMethodCall` scaffolding.

## Contribution Expectations
- Prefer idiomatic Go style for implementation code (run `gofmt` or `make fmt`).
- Keep language semantics aligned with the design docs; if behaviour diverges, document the rationale.
- When adding new features, ensure effect annotations, session types, and deterministic trace guarantees remain explicit.
- Provide or update examples/tests when extending the language.

## Additional Notes
- No existing AGENT instructions were present; this file acts as the root scope guide.
- If you add subdirectories with specialized conventions, create additional `AGENTS.md` files there to override or extend these guidelines.
