# Forbid Duplicate Top-Level Function Definitions

- **Date**: 2026-10-19
- **Class**: bug | feature | question | already-covered | not-actionable
- **Recommend**: design-doc
- **Searched**: `duplicate`, `redefin`, `shadowing`, `conflict`, `FuncDecl`
- **Estimate**: `~40 lines in internal/parser/parser_decl.go, internal/types/errors.go`

<One paragraph: the reason, not a restatement of the title. Name the file and
the mechanism if you found them.>

## Problem Statement

AILANG currently allows duplicate top-level function definitions in the same module scope to coexist silently, resulting in the last definition shadowing the first.

```ailang
module dup
export func f() -> int = 1
export func f() -> int = 2
export func main() -> () ! {IO} = println(show(f()))
```

Running `ailang check` reports no errors, but the first function is dead code and unreachable. This is particularly hazardous in large (600+ lines) codebases where a splice or copy-paste refactor leaves an old definition above the new one. Edits to the first copy have no effect, and no diagnostic indicates why.

## Proposed Solution

To ensure the compiler fails fast on this ambiguity, we propose the following changes:

1. **Duplicate detection**: Modify the declaration registration phase (likely in `internal/parser/parser_decl.go` or the type checker's symbol scope setup) to track top-level names.
2. **Error reporting**: Extend `internal/types/errors.go` to include a `TC_DECL_002: Duplicate Definition` error. The error should report the name, the location of the first definition, and the location of the subsequent conflicting definition.
3. **Scope handling**: We should determine whether "Duplicate" means strict equality (e.g., same name and type signature) or simply same name. Usually, duplicate names in the same scope are forbidden in strictly typed languages.

## Affected Files

- `internal/parser/parser_decl.go`: Where declarations are gathered.
- `internal/types/errors.go`: Add error code for duplicate definitions.
- `internal/types/typechecker.go`: Wire up the check during the typechecking phase.
