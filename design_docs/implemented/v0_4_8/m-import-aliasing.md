# M-IMPORT-ALIASING: Module and Symbol Aliasing

**Status**: Planned
**Target**: v0.4.8
**Priority**: P1 (Medium-High)
**Estimated**: 3-4 days
**Dependencies**: None (builds on existing import system)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables qualified calls without import clutter |
| Preserve Semantic Clarity | + | +1 | `List.map` vs `Option.map` is self-documenting |
| Increase Determinism | 0 | 0 | No change to runtime behavior |
| Lower Token Cost | + | +1 | Models can write `List.map` without ambiguity errors |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The stdlib has multiple modules exporting functions with the same names, following standard FP conventions:

| Function | Modules | Pattern |
|----------|---------|---------|
| `length` | `std/string`, `std/list` | Container size |
| `map` | `std/list`, `std/option`, `std/result` | Functor operation |
| `flatMap` | `std/option`, `std/result` | Monad bind |
| `filter` | `std/list`, `std/option` | Filterable operation |

**This is not a bug** - these are exactly the names you want in those modules. Every FP/ML ecosystem uses:
- `List.length`, `String.length`
- `List.map`, `Option.map`, `Result.map`

**Current State:**
- Importing from multiple modules causes name clashes
- No way to disambiguate without changing stdlib names
- Users must choose which module to import or avoid certain functions
- Teaching prompts can't show combined Option/Result/List workflows cleanly

**Impact:**
- AI models get ambiguous resolution errors when generating code
- Human readers can't tell which `map` is being called
- Stdlib names would need ugly prefixes (`listMap`, `optionMap`) as a workaround

## Goals

**Primary Goal:** Enable qualified and aliased imports so multiple modules with same-named exports can coexist cleanly.

**Success Metrics:**
- Module aliasing works: `import std/list as List` → `List.map(f, xs)`
- Symbol aliasing works: `import std/string (length as stringLength)`
- Prelude exports one canonical `map`/`filter`/`length` (list versions)
- Teaching prompts show qualified style for Option/Result operations
- Zero changes to stdlib function names

## Solution Design

### Overview

Add two import features:

1. **Module aliasing**: `import std/list as List` creates `List.` namespace
2. **Symbol aliasing**: `import std/string (length as stringLength)` renames on import

Keep stdlib names canonical; let call sites decide disambiguation.

### Architecture

**Components:**

1. **Lexer**: Add `AS` keyword token
2. **Parser**: Extend import statement parsing for `as` clauses
3. **Module System**: Track aliases in import resolution
4. **Name Resolution**: Support qualified lookups (`List.map`)
5. **Prelude Definition**: Curate which names export unqualified

### Syntax Design

```ailang
-- Module aliasing (creates qualified namespace)
import std/list as List
import std/option as Option
import std/result as Result

-- Usage: qualified calls
let xs = List.map(f, [1, 2, 3])
let oy = Option.map(g, Some(x))
let rz = Result.flatMap(h, Ok(v))

-- Symbol aliasing (renames on import)
import std/list (length, map, filter)                    -- specific symbols
import std/string (length as stringLength, split)        -- with rename
import std/option (map as optionMap, flatMap as optionFlatMap)

-- Combined
import std/list as List (length, map)   -- both: List.map AND unqualified map

-- Wildcard (existing behavior)
import std/list (*)                      -- all symbols unqualified
```

### Implementation Plan

**Phase 1: Lexer & Parser** (~4 hours)
- [ ] Add `AS` keyword to lexer token types
- [ ] Update import statement AST node to hold alias info
- [ ] Parse `import path as Alias` syntax
- [ ] Parse `import path (sym as alias, ...)` syntax
- [ ] Add parser tests for new syntax variations

**Phase 2: Module Resolution** (~6 hours)
- [ ] Extend import resolution to track module aliases
- [ ] Implement qualified name lookup (e.g., `List.map`)
- [ ] Handle symbol renaming in import binding
- [ ] Ensure aliases are scoped to the importing module
- [ ] Add integration tests for cross-module usage

**Phase 3: Prelude Curation** (~2 hours)
- [ ] Define canonical prelude exports (list versions of `map`/`filter`/`length`)
- [ ] Document that Option/Result versions require qualified import
- [ ] Update std/prelude.ail if it exists, or document implicit prelude behavior

**Phase 4: Documentation & Teaching** (~4 hours)
- [ ] Update teaching prompt to show qualified style
- [ ] Add examples showing `List.map` vs `Option.map` vs `Result.map`
- [ ] Update CLAUDE.md with import aliasing patterns
- [ ] Create examples/imports/ directory with aliasing examples

### Files to Modify/Create

**Modified files:**
- `internal/lexer/token.go` - Add AS token (~5 LOC)
- `internal/lexer/lexer.go` - Recognize `as` keyword (~10 LOC)
- `internal/ast/ast.go` - Extend ImportDecl for aliases (~20 LOC)
- `internal/parser/parser.go` - Parse alias syntax (~80 LOC)
- `internal/loader/loader.go` - Handle alias resolution (~60 LOC)
- `internal/module/resolver.go` - Qualified name lookup (~100 LOC)
- `prompts/` - Update teaching prompt for aliasing patterns

**New files:**
- `examples/imports/module_aliasing.ail` - Module alias examples (~30 LOC)
- `examples/imports/symbol_aliasing.ail` - Symbol alias examples (~30 LOC)
- `examples/imports/qualified_stdlib.ail` - Full stdlib usage patterns (~50 LOC)

## Examples

### Example 1: Module Aliasing for Functor Pattern

**Before (name clash):**
```ailang
import std/list (map)
import std/option (map)    -- ERROR: map already imported!

let xs = map(f, [1,2,3])   -- Which map?
let oy = map(g, Some(x))   -- Ambiguous!
```

**After (qualified imports):**
```ailang
import std/list as List
import std/option as Option

let xs = List.map(f, [1, 2, 3])      -- Clear: list map
let oy = Option.map(g, Some(x))      -- Clear: option map
```

### Example 2: Symbol Aliasing for Specific Renames

**Before:**
```ailang
import std/list (length)
import std/string (length)   -- ERROR: length already imported!
```

**After:**
```ailang
import std/list (length)
import std/string (length as stringLength)

let n = length([1, 2, 3])      -- list length
let m = stringLength("hello")   -- string length
```

### Example 3: Combined Workflow (Real-World Pattern)

```ailang
module app/userService

import std/list as List
import std/option as Option
import std/result as Result
import std/string (split, trim)

-- Process user input, handling errors at each stage
func processUserIds(input: string) -> Result[[int], string] ! {IO} {
  let parts = split(input, ",")                           -- string.split
  let trimmed = List.map(trim, parts)                     -- list.map
  let parsed = List.map(parseInt, trimmed)                -- list.map
  let validated = List.map(\opt. Option.flatMap(validate, opt), parsed)
  Result.flatMap(collectResults, sequence(validated))      -- result.flatMap
}
```

### Example 4: Teaching Prompt Style

**New standard for examples involving multiple container types:**
```ailang
-- Always use qualified imports for Option/Result
import std/option as Option
import std/result as Result

-- List operations can be unqualified (prelude)
let doubled = map(\x. x * 2, [1, 2, 3])

-- Option/Result always qualified
let maybeValue = Option.map(\x. x + 1, Some(42))
let result = Result.flatMap(parseJson, Ok(input))
```

## Success Criteria

- [ ] `import std/list as List` parses and creates `List.` namespace
- [ ] `List.map(f, xs)` resolves correctly to std/list.map
- [ ] `import std/string (length as stringLength)` renames on import
- [ ] Can import from multiple modules with same-named exports without error
- [ ] Prelude exports `map`, `filter`, `length` from std/list only
- [ ] Teaching prompt updated with qualified import style
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- Lexer recognizes `as` keyword
- Parser produces correct AST for all alias syntax variations
- Import resolution handles module aliases
- Symbol renaming works correctly

**Integration tests:**
- Cross-module imports with qualified calls
- Mixed qualified and unqualified usage
- Error messages for undefined qualified names
- Prelude interaction with explicit imports

**Manual testing:**
- Run examples/imports/*.ail files
- REPL with qualified imports
- AI model generates code using qualified style

## Non-Goals

**Not in this feature:**
- **Ad-hoc overloading** - Not implementing type-based disambiguation (too heavy)
- **User-defined type classes** - Deferred to v0.4.0+ (structural reflection)
- **Automatic qualification** - User chooses when to qualify
- **Re-exports** - `export * from std/list as List` (future work)

## Timeline

**Day 1** (6 hours):
- Phase 1: Lexer & parser changes
- Parser tests passing

**Day 2** (6 hours):
- Phase 2: Module resolution
- Integration tests passing

**Day 3** (4 hours):
- Phase 3: Prelude curation
- Phase 4: Documentation begins

**Day 4** (4 hours):
- Complete documentation
- Update teaching prompt
- Examples and polish

**Total: ~20 hours across 4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Parser complexity for combined syntax | Medium | Start with simple cases, add combined last |
| Qualified lookup performance | Low | Simple map lookup, O(1) |
| Breaking existing imports | Medium | All existing syntax remains valid |
| Teaching prompt token increase | Low | Qualified style is actually clearer |

## References

- [std/list.ail](../../../std/list.ail) - Current list module
- [std/option.ail](../../../std/option.ail) - Current option module
- [std/result.ail](../../../std/result.ail) - Current result module
- [Haskell qualified imports](https://wiki.haskell.org/Import) - Prior art
- [OCaml module aliasing](https://ocaml.org/manual/moduleexamples.html) - Prior art
- [Rust use as](https://doc.rust-lang.org/reference/items/use-declarations.html) - Prior art

## Future Work

Features that build on this but are out of scope for now:

1. **Re-exports**: `export * from std/list as List` for library authors
2. **Automatic prelude customization**: Per-project prelude configuration
3. **Type class instances**: When reflection lands, `Functor.map` could unify all three maps
4. **IDE/tooling support**: Auto-suggest qualified imports on name clash

---

**Document created**: 2025-11-27
**Last updated**: 2025-11-27
