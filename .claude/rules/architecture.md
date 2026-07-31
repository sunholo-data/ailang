---
paths:
  - "internal/**"
  - "cmd/**"
  - "ui/**"
---

# Architecture Rules

Loads whenever you touch `internal/`, `cmd/`, or `ui/`. Full rationale and the
allow/deny table: [ARCHITECTURE.md](../../ARCHITECTURE.md#architecture-boundaries).

## Layers

`internal/` is organized into logical layers. Know which one you're touching:

- **core** (compiler/runtime): `internal/{parser,types,eval,core,elaborate,effects,builtins,lexer,ast,pipeline,runtime,link,iface}`
- **dashboard/apps** (services + UI): `internal/{server,coordinator,observatory,messaging}` (+ `ui/`)
- **bridge**: `internal/embed` — the ONLY sanctioned path from dashboard → compiler.

## Never cross these import directions

1. A **core** package must NOT import a **dashboard** package.
2. A **dashboard** package must NOT import the compiler surface
   (`parser`/`types`/`core`/`elaborate`/`pipeline`) directly — go through `internal/embed`.

Run before committing any cross-cutting change (CI gate):

```bash
make check-boundaries
```

## AI provider vs executor

Picking the wrong one is a recurring mistake — they are not interchangeable.

| Package | Purpose | Use for |
|---------|---------|---------|
| `internal/ai/` | Text generation via HTTP APIs | Research, docs, Q&A, single-shot calls |
| `internal/executor/` | Agentic coding with file editing | Bug fixes, features, refactoring |
