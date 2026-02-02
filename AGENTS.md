# AILANG Repository Guidelines

This document summarizes the key facts an agent should know while working in this repository. Review it before making changes.

## Start Here (Required)
- **Read `CLAUDE.md` first.** It is the operational source of truth for AILANG workflows, message handling, coordinator commands, and critical guardrails.
- **Session start routine:** Check for messages with `ailang messages list --unread`, summarize any to the user, and acknowledge with `ailang messages ack --all` after handling.

## Project Overview
- **Language focus**: AILANG is a purely functional language optimized for AI-assisted development, emphasizing explicit algebraic effects, typed quasiquotes, CSP/session-type concurrency, and deterministic execution traces. Refer to `design_docs/20250926/initial_design.md` for the conceptual specification.
- **Status**: Lexer, basic parser, AST, and foundational type system are implemented. Effect system, interpreter, and several advanced features are still TODO according to the top-level README.

## Repository Structure & Tooling
- `cmd/ailang/`: Go CLI entry point.
- `internal/`: Core compiler/interpreter packages (lexer, parser, AST, types, effects, eval, etc.). Many subpackages are still under construction.
- `examples/`: Example `.ail` programs.
- `design_docs/20250926/`: Canonical language design references.
- Use `make build`, `make test`, `make fmt`, and `make lint` for common workflows.

## Operational Commands (Quick)
- **Messages:** `ailang messages list --unread`, `ailang messages read <id>`, `ailang messages ack <id>`, `ailang messages ack --all`
- **Coordinator:** `ailang coordinator status`, `ailang coordinator start`, `ailang coordinator stop`, `ailang coordinator pending`
- **Prompt:** `ailang prompt --version V` (see `CLAUDE.md` for version guidance)

## Key Design Details
- **Type system**: Hindley–Milner style with row-polymorphic algebraic effects and capability annotations. Review `initial_design.md` for type/effect constructs and idioms.
- **Row unification**: Reference Go implementation for effect/record row handling lives in `design_docs/20250926/gpt5-reference-code.md`; it defines `Row`, `Subst`, and `UnifyRows` helpers for deterministic effect reasoning.
- **Typeclass dictionaries**: Explicit dictionary passing is the intended elaboration strategy; see the same reference doc for `Class`, `Instance`, and `ElabMethodCall` scaffolding.

## Critical Guardrails (Do Not Skip)
- **No destructive git operations.** Do not run `git reset --hard`, `git clean -fd`, or switch branches with uncommitted changes. Ask the user first.
- **Use existing tools first.** Check `make` targets, `tools/`, and `CLAUDE.md` workflows before adding scripts.
- **No silent fallbacks.** If config or model data is missing, return error/zero and surface it.
- **Parser note:** Lexer skips newlines; do not expect NEWLINE tokens.

## Contribution Expectations
- Prefer idiomatic Go style for implementation code (run `gofmt` or `make fmt`).
- Keep language semantics aligned with the design docs; if behaviour diverges, document the rationale.
- When adding new features, ensure effect annotations, session types, and deterministic trace guarantees remain explicit.
- Provide or update examples/tests when extending the language.

## Additional Notes
- No existing AGENT instructions were present; this file acts as the root scope guide.
- If you add subdirectories with specialized conventions, create additional `AGENTS.md` files there to override or extend these guidelines.
