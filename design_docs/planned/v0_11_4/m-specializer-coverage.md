---
status: Planned
target: v0.11.4
priority: P1
estimated: 1 day (~150 LOC)
dependencies: None
discovered_during: M-V0_11_3-HOTFIX (M1 — short-circuit)
---

# Specializer & Cloner Coverage Gaps (DEBUG_STRICT=1)

## Problem

The monomorphization pass in `internal/pipeline/specialize_expr.go` and its
partner `internal/pipeline/specialize_clone.go` both have `default:` arms that
silently return the expression unchanged. Under `DEBUG_STRICT=1` they panic,
which makes the invariant visible: several common Core node types are never
recursed into during specialization / cloning.

Concretely, `DEBUG_STRICT=1 make test` fails — first on `*core.List`, then
on `*core.DictRef`, and likely more. In non-strict mode these nodes flow
through unchanged, which masks the gap for fully-concrete programs but is
a silent-fallback hazard (violates CLAUDE.md §2).

Discovered while implementing M1 of the v0.11.3 hotfix. Not a blocker for
that release — `&&`/`||` desugar is correctness-complete and `make test`
is green under normal (non-strict) mode.

## Goals

- `DEBUG_STRICT=1 make test` passes with no "unhandled node type" panics.
- Every `core.CoreExpr` variant has an explicit case in both
  `specializeExpr` and `cloneExpr`.
- Default arm becomes: "panic unconditionally" (not gated on env var) —
  once coverage is complete, reaching the default is a bug.

## Scope

Node types currently unhandled (at least):
- `*core.List`, `*core.Array`, `*core.Tuple`
- `*core.Record`, `*core.RecordAccess`, `*core.RecordUpdate`
- `*core.DictRef` (atomic — just return as-is)
- `*core.DictAbs` (recurse into Body)

Audit checklist:
- [ ] Enumerate every concrete `CoreExpr` implementer via
      `grep -n 'coreExpr()' internal/core/*.go`
- [ ] Each has a case in `specializeExpr`
- [ ] Each has a case in `cloneExpr`
- [ ] Remove `os.Getenv("DEBUG_STRICT")` guard — panic unconditionally
- [ ] Remove `// TODO: Handle other expression types` comment
- [ ] Delete `internal/pipeline/specialize.go.backup` (unused, misleading)

## Acceptance Criteria

- [ ] `DEBUG_STRICT=1 make test` passes (all packages)
- [ ] `DEBUG_STRICT=1 make verify-examples` passes
- [ ] `specializeExpr` and `cloneExpr` default arms panic unconditionally
- [ ] No behavioral change under normal mode (existing tests stay green)
- [ ] Unit test that walks a fabricated Core tree containing every node
      type and confirms the specializer traverses all of them

## Out of Scope

- Deeper specializer refactoring (e.g. extracting a generic `walk`
  combinator) — this doc is a narrowly scoped coverage patch.
- Fixing any `cloneExpr` semantic bugs that surface — if cloning a
  `Record` reveals a type-info propagation bug, file a separate doc.

## Axiom Scoring

| Axiom | Score | Notes |
|-------|-------|-------|
| A1 (correctness) | +2 | Removes silent fallback on unknown types |
| A3 (observability) | +1 | DEBUG_STRICT becomes trustworthy again |
| A4 (determinism) | 0 | Behavior unchanged in normal mode |
| A7 (simplicity) | +1 | Removes conditional-panic idiom |
| Others | 0 | |

**Net: +4** (well above +2 bar)

## References

- Discovered in: sprint `M-V0_11_3-HOTFIX`, M1 checkpoint
- Related: `design_docs/archive/2025-10/completion_reports/M-DX8-PHASE1-QUICK-WINS.md` (original motivation for the DEBUG_STRICT panics)
- Related: `design_docs/archive/v0_4_1_m-dx8-silent-failure-prevention.md`
- Files: `internal/pipeline/specialize_expr.go`, `internal/pipeline/specialize_clone.go`
