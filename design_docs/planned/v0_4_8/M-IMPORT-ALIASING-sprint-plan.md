# Sprint Plan: M-IMPORT-ALIASING

## Summary

Add module and symbol aliasing to the import system, enabling qualified imports (`import std/list as List`) and symbol renaming (`import std/string (length as stringLength)`) to resolve name clashes between stdlib modules.

**Duration:** 4 days (16-20 hours)
**Dependencies:** None (builds on existing import system)
**Risk Level:** Low-Medium

## Current Status Analysis

### Completed Recently
- v0.4.7: Inline tests feature (~400 LOC)
- M-TESTING-DEPS: Cross-function dependency support (~400 LOC)
- Concat operator fix (~260 LOC)

### Velocity
- Recent average: ~200-400 LOC/day
- Estimated capacity: 600-800 LOC for this sprint (4 days)

### Current Import Implementation
- `ImportDecl` struct: `Path`, `Symbols` fields only
- Parser handles selective imports: `import std/io (println)`
- Namespace imports currently error with "not yet supported"
- No `as` keyword in lexer
- No qualified name resolution (`List.map`)

### Remaining Work (from Design Doc)
- Phase 1: Lexer & Parser (~100 LOC + ~80 LOC tests)
- Phase 2: Module Resolution (~180 LOC + ~100 LOC tests)
- Phase 3: Prelude Curation (~30 LOC)
- Phase 4: Documentation & Examples (~100 LOC)

**Total Estimated:** ~590 LOC (within velocity capacity)

## Proposed Milestones

### Milestone 1: Lexer & AST Changes (Day 1 Morning)

**Goal:** Add `AS` keyword and extend ImportDecl for aliasing

**Estimated:** 40 LOC implementation + 40 LOC tests = 80 LOC
**Duration:** 2-3 hours

**Files to modify:**
- `internal/lexer/token.go` - Add AS token (~8 LOC)
- `internal/lexer/lexer.go` - Recognize `as` keyword (already handled by keyword map)
- `internal/ast/ast.go` - Extend ImportDecl (~25 LOC)

**Tasks:**
```
1. Add AS token constant to TokenType enum
2. Add "as" to keywords map
3. Add to tokens string map
4. Extend ImportDecl:
   - Add ModuleAlias field (string)
   - Add SymbolAliases field (map[string]string - original -> alias)
5. Update ImportDecl.String() method
6. Write lexer test for AS keyword
```

**Acceptance Criteria:**
- [ ] `lexer.LookupIdent("as")` returns `lexer.AS`
- [ ] ImportDecl has ModuleAlias and SymbolAliases fields
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- None - straightforward additions to existing structures

---

### Milestone 2: Parser Changes (Day 1 Afternoon + Day 2 Morning)

**Goal:** Parse all alias syntax variations

**Estimated:** 100 LOC implementation + 80 LOC tests = 180 LOC
**Duration:** 4-5 hours

**Files to modify:**
- `internal/parser/parser_file.go` - Extend parseImportDecl (~100 LOC)

**Tasks:**
```
1. Parse module aliasing: import path as Alias
   - After parsing path, check for AS token
   - Expect capitalized IDENT (convention for namespaces)
   - Store in imp.ModuleAlias

2. Parse symbol aliasing: import path (sym as alias, ...)
   - Inside selective import loop, check for AS after symbol
   - Parse alias identifier
   - Store in imp.SymbolAliases map

3. Parse combined: import path as Alias (sym1, sym2 as alias2)
   - Both module alias AND selective with optional aliases

4. Add parser tests for all syntax variations:
   - import std/list as List
   - import std/string (length as stringLength)
   - import std/list as List (map, filter)
   - import std/option (map as optionMap, flatMap as optionFlatMap)
   - Error cases: lowercase module alias, missing alias name
```

**Acceptance Criteria:**
- [ ] Parser produces correct AST for all syntax variations
- [ ] Error messages for invalid syntax (missing alias, etc.)
- [ ] Existing import tests still pass
- [ ] 15+ new parser tests for aliasing
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Parser complexity for combined syntax - start simple, add combined last
- Token position bugs - use DEBUG_PARSER=1 for tracing

---

### Milestone 3: Module Resolution (Day 2 Afternoon + Day 3)

**Goal:** Resolve module aliases and qualified names at evaluation time

**Estimated:** 180 LOC implementation + 100 LOC tests = 280 LOC
**Duration:** 6-8 hours

**Files to modify:**
- `internal/loader/loader.go` - Track module aliases (~60 LOC)
- `internal/eval/eval.go` - Qualified name lookup (~100 LOC)
- Possibly `internal/types/typechecker.go` - Type check qualified names (~20 LOC)

**Tasks:**
```
1. LoadedModule changes:
   - Add Aliases map[string]string (alias -> canonical path)
   - Track which modules are imported with which aliases

2. Symbol binding changes:
   - When importing with module alias, don't add symbols to global scope
   - Store under qualified namespace: "List.map", "List.filter"
   - Symbol aliases: bind alias name to original symbol

3. Name resolution changes:
   - Check for dot in identifier: "List.map"
   - Split into namespace and symbol
   - Look up namespace in alias map
   - Resolve symbol from that module

4. Type checking:
   - Ensure qualified names type check correctly
   - Same type as unqualified version

5. Integration tests:
   - Cross-module imports with qualified calls
   - Mixed qualified and unqualified usage
   - Error messages for undefined qualified names
```

**Acceptance Criteria:**
- [ ] `List.map(f, xs)` resolves to std/list.map
- [ ] Symbol renaming works: `stringLength` -> `std/string.length`
- [ ] Type inference works for qualified names
- [ ] Error messages for unknown qualifiers: "unknown module alias 'Foo'"
- [ ] 10+ integration tests
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Name resolution complexity - keep lookup simple (map + split on dot)
- Performance - should be O(1) map lookup

---

### Milestone 4: Prelude & Documentation (Day 4)

**Goal:** Curate prelude exports and update documentation

**Estimated:** 30 LOC implementation + 100 LOC examples = 130 LOC
**Duration:** 4-5 hours

**Files to modify/create:**
- `std/prelude.ail` - Verify canonical exports
- `prompts/` - Update teaching prompt
- `CLAUDE.md` - Add import aliasing patterns

**Files to create:**
- `examples/imports/module_aliasing.ail` (~30 LOC)
- `examples/imports/symbol_aliasing.ail` (~30 LOC)
- `examples/imports/qualified_stdlib.ail` (~50 LOC)

**Tasks:**
```
1. Prelude verification:
   - Confirm map/filter/length export from std/list
   - Document that Option/Result versions require qualified import

2. Teaching prompt update:
   - Add qualified import syntax
   - Show List.map vs Option.map examples
   - Update "Working with Multiple Containers" section

3. CLAUDE.md update:
   - Document import aliasing patterns
   - Common conventions (List, Option, Result capitalization)

4. Example files:
   - module_aliasing.ail: Basic List/Option/Result usage
   - symbol_aliasing.ail: Renaming length, map, etc.
   - qualified_stdlib.ail: Full workflow example

5. Verify all examples:
   - Run each example file
   - Check output matches expectations
```

**Acceptance Criteria:**
- [ ] All 3 example files run successfully
- [ ] Teaching prompt shows qualified import style
- [ ] CLAUDE.md documents aliasing patterns
- [ ] `make verify-examples` passes
- [ ] `make test` passes

**Risks:**
- Teaching prompt token increase - qualified style is actually clearer

---

## Day-by-Day Schedule

### Day 1 (6 hours)
| Time | Task | Deliverable |
|------|------|-------------|
| Morning (3h) | Milestone 1: Lexer & AST | AS token, ImportDecl extension |
| Afternoon (3h) | Milestone 2: Parser (start) | Module alias parsing |

### Day 2 (6 hours)
| Time | Task | Deliverable |
|------|------|-------------|
| Morning (3h) | Milestone 2: Parser (finish) | Symbol alias parsing, tests |
| Afternoon (3h) | Milestone 3: Resolution (start) | LoadedModule changes |

### Day 3 (4 hours)
| Time | Task | Deliverable |
|------|------|-------------|
| Full day (4h) | Milestone 3: Resolution (finish) | Qualified lookup, integration tests |

### Day 4 (4 hours)
| Time | Task | Deliverable |
|------|------|-------------|
| Morning (2h) | Milestone 4: Prelude & Docs | Prelude verification, CLAUDE.md |
| Afternoon (2h) | Milestone 4: Examples | 3 example files, teaching prompt |

---

## Success Metrics

- [ ] All 4 milestones complete
- [ ] ~590 LOC added (implementation + tests + examples)
- [ ] Test coverage maintained or improved
- [ ] All tests passing: `make test`
- [ ] All linting clean: `make lint`
- [ ] Examples passing: 3 new import examples
- [ ] Teaching prompt updated for qualified imports

## Dependencies

- No external dependencies
- Builds on existing import system (parser, loader)
- No blocking issues identified

## Open Questions

1. **Module alias naming convention**: Should we enforce PascalCase for module aliases? (Design doc suggests yes for consistency with ML/Haskell)

2. **Wildcard with alias**: Should `import std/list as List (*)` be supported? (Probably yes for consistency)

3. **Re-exports**: `export * from std/list as List` deferred to future work per design doc

## Notes

- Design doc AI-First alignment score: +3 (Move forward)
- Parser leaves cursor AT last token (not after) - follow convention
- No NEWLINE tokens exist in lexer - don't check for them
- Use DEBUG_PARSER=1 for debugging token positions
- Recent v0.4.7 release provides stable foundation

---

**Plan created:** 2025-11-28
**Design doc:** [m-import-aliasing.md](m-import-aliasing.md)
