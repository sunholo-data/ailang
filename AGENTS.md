# AILANG Repository Guidelines

This document summarizes the key facts an agent should know while working in this repository. Review it before making changes.

## Start Here (Required)
- **Read `CLAUDE.md` first.** It is the operational source of truth for AILANG workflows, message handling, coordinator commands, and critical guardrails.
- **Session start routine:** Check for messages with `ailang messages list --unread`, summarize any to the user, and acknowledge with `ailang messages ack --all` after handling.
- **Programming in AILANG:** Use `ailang prompt` to get the current teaching prompt before writing or editing `.ail` code.

## Project Overview
- **Language focus**: AILANG is a purely functional, effect-typed language designed as a deterministic execution substrate for AI-generated code.
- **Status**: The compiler, interpreter, capability/effect runtime, standard library, package tooling, coordinator, and evaluation harness are implemented and actively developed. Use `design_docs/PROGRAM.md` for the north star and `design_docs/v1-mission.md` for current priorities; do not infer status from old design documents.

## Repository Structure & Tooling
- `cmd/ailang/`: Go CLI entry point.
- `internal/`: Compiler/runtime plus application and tooling packages. Respect the layer boundaries documented in `ARCHITECTURE.md`.
- `std/`: AILANG standard library modules.
- `examples/`: Example `.ail` programs.
- `design_docs/PROGRAM.md`: Program north star and routing rules.
- `design_docs/v1-mission.md`: Current v1 mission status, queue, and human decision points.
- Use `make build`, `make test`, `make fmt`, and `make lint` for common workflows.

## Operational Commands (Quick)
- **Messages:** `ailang messages list --unread`, `ailang messages read <id>`, `ailang messages ack <id>`, `ailang messages ack --all`
- **Coordinator:** `ailang coordinator status`, `ailang coordinator start`, `ailang coordinator stop`, `ailang coordinator pending`
- **Prompt:** `ailang prompt --version V` (see `CLAUDE.md` for version guidance)

## AILANG Coding Quickstart
- **Start with the prompt:** `ailang prompt` to get the current teaching prompt and idioms.
- **Browse working examples:** `examples/` has runnable `.ail` programs.
- **Use the REPL:** `ailang repl` for quick experiments (see `docs/docs/reference/repl-commands.md`).
- **Type-check fast:** `ailang check <file>` before running.

## Key Design Details
- **Type system**: Hindley–Milner inference with row-polymorphic records and effects, capability annotations, and explicit dictionary passing.
- **Runtime**: The tree-walking evaluator and capability-gated effect handlers are production code. Treat `internal/types/`, `internal/eval/`, and `internal/effects/` as security- and semantics-sensitive.
- **Architecture**: Core packages must not import dashboard packages, and dashboard packages reach compiler behavior through `internal/embed`. Run `make check-boundaries` for cross-cutting changes.

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
- If you add subdirectories with specialized conventions, create additional `AGENTS.md` files there to override or extend these guidelines.
