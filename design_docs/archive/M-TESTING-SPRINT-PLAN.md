# M-TESTING Sprint Plan: Property-Based Testing Framework

**Created**: 2025-10-26
**Status**: Approved - Ready to Execute
**Duration**: 2 weeks (10 working days)
**Estimated LOC**: ~2,410 (1,450 implementation + 960 tests)

## Executive Summary

Implement property-based testing framework with:
- Inline tests (`tests [(input, expected)]`)
- Property declarations (`properties [forall(...) => predicate]`)
- Test/property blocks (`test "name" { }`)
- `ailang test` CLI command with generators and shrinking

**Key Discovery**: Lexer keywords already exist! Parser has TODO stubs. ~20% head start.

## Week 1: Foundation (Days 1-5)

### Milestone 1: Parser & AST Support (Days 1-3, ~450 LOC)

**Day 1: Inline Tests**
- [ ] Implement `parseTestsBlock()` for `tests [(input, expected)]`
- [ ] Add `Tests` field to `FuncDecl` AST node
- [ ] 20 parser unit tests
- **Output**: `factorial.ail` with `tests [...]` parses correctly

**Day 2: Property Parsing**
- [ ] Implement `parsePropertiesBlock()` for `properties [forall(...) => expr]`
- [ ] Parse `forall(var: type) => predicate` syntax
- [ ] Add `Properties` field to `FuncDecl`
- [ ] 25 parser unit tests
- **Output**: `quicksort.ail` with `properties [...]` parses correctly

**Day 3: Test/Property Blocks**
- [ ] Parse `test "name" { assert expr; ... }` blocks
- [ ] Parse `property "name" { forall(...) => expr }` blocks
- [ ] Add `TestDecl`, `PropertyDecl`, `AssertStmt` AST nodes
- [ ] 30 parser unit tests
- **Output**: Standalone test files parse correctly

### Milestone 2: Test Execution Engine (Days 4-5, ~400 LOC)

**Day 4: Collector & Basic Runner**
- [ ] Create `internal/testing/collector.go` (extract tests from AST)
- [ ] Create `internal/testing/runner.go` (execute inline tests)
- [ ] Implement assertion (`assert expr` → panic on false)
- [ ] Test result data structures
- [ ] 20 unit tests
- **Output**: `factorial.ail` inline tests actually run

**Day 5: Test Reporting**
- [ ] Aggregate test results (pass/fail counts)
- [ ] JSON output format (machine-readable)
- [ ] Human-readable output (colored, clear)
- [ ] Failure details (expected vs actual)
- [ ] 15 unit tests
- **Output**: Clear test output shows passes/failures

## Week 2: Property Testing & Polish (Days 6-10)

### Milestone 3: Property-Based Testing (Days 6-8, ~500 LOC)

**Day 6: Basic Generators**
- [ ] Create `internal/testing/generator.go`
- [ ] Generators: int, float, bool, string
- [ ] List generator (recursive)
- [ ] Deterministic seed support
- [ ] 30 unit tests
- **Output**: Generate random int/string/list values

**Day 7: Advanced Generators**
- [ ] ADT generators (Option, Result, custom types)
- [ ] Record generators
- [ ] Generator combinators (map, filter, sized)
- [ ] 25 unit tests
- **Output**: Generate all AILANG types

**Day 8: Shrinking Algorithm**
- [ ] Create `internal/testing/shrink.go`
- [ ] Shrink primitives (int, string, list)
- [ ] Shrink ADTs
- [ ] Minimal failing case finder
- [ ] 20 unit tests
- **Output**: Property failures show minimal inputs

### Milestone 4: CLI & Integration (Days 9-10, ~200 LOC)

**Day 9: CLI Command**
- [ ] Create `cmd/ailang/cmd_test.go`
- [ ] `ailang test file.ail` command
- [ ] Flags: `--property-tests=N`, `--seed=N`
- [ ] Exit codes (0=pass, 1=fail)
- [ ] 10 CLI tests
- **Output**: Working `ailang test` command

**Day 10: Examples & Documentation**
- [ ] Update `factorial.ail` with inline tests (4 cases)
- [ ] Update `quicksort.ail` with properties (3 properties)
- [ ] Create 2 new test examples
- [ ] Update CLAUDE.md, teaching prompt
- [ ] Verify all examples pass
- **Output**: Complete, documented feature

## Success Criteria

### Parser (Week 1)
- ✅ `tests [...]` and `properties [...]` parse correctly
- ✅ `test "name" { }` and `property "name" { }` blocks work
- ✅ `assert` and `forall` expressions parse
- ✅ 100+ parser tests passing

### Test Execution (Week 1-2)
- ✅ Inline tests run automatically
- ✅ Property tests generate 100 cases per property
- ✅ Assertions fail with clear messages
- ✅ JSON + human-readable output

### Property Testing (Week 2)
- ✅ Generators for int, bool, string, list, ADT, record
- ✅ Shrinking finds minimal failing inputs
- ✅ Deterministic with `--seed=N`
- ✅ Properties verify invariants

### CLI (Week 2)
- ✅ `ailang test file.ail` runs all tests
- ✅ `--property-tests=N` configures test count
- ✅ `--seed=N` enables reproducible tests
- ✅ Exit code 0 (pass) or 1 (fail)

### Examples (Week 2)
- ✅ `factorial.ail` with inline tests works
- ✅ `quicksort.ail` with properties works
- ✅ 2 additional test examples
- ✅ All pass `make verify-examples`

### Documentation (Week 2)
- ✅ CLAUDE.md documents test syntax
- ✅ Teaching prompt includes examples
- ✅ `internal/testing/README.md` exists

## LOC Estimates by Component

| Component | Implementation | Tests | Total |
|-----------|----------------|-------|-------|
| Parser extensions | 300 | 150 | 450 |
| Test collector | 150 | 50 | 200 |
| Test runner | 250 | 100 | 350 |
| Generators | 300 | 150 | 450 |
| Shrinking | 200 | 80 | 280 |
| CLI command | 150 | 30 | 180 |
| Examples | 100 | 400 | 500 |
| **Total** | **1,450** | **960** | **2,410** |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Generator complexity | Medium | Start simple, add sophistication iteratively |
| Shrinking performance | Low | Limit iterations (100 max), configurable |
| Parser breaks existing code | High | Extensive testing, backward compatible |
| Property tests too slow | Medium | Configurable test count (default 100) |

## Velocity Calibration

**Recent velocity**: ~31 LOC/day net (last 14 days)
**Required velocity**: ~241 LOC/day (for 2 weeks)
**Gap**: 7.8x recent pace

**Recommendation**:
- 2 weeks = MVP (inline tests + basic properties)
- Defer watch mode, advanced features to v0.4.3
- Focus on factorial.ail and quicksort.ail working

## Dependencies

**Prerequisites** (all met):
- ✅ Keywords in lexer (test, property, assert, forall)
- ✅ Parser has TODO stubs
- ✅ Module system stable
- ✅ Effect system works

**No blockers** - ready to start!

## Open Questions

1. **Scope**: Include `--watch` or defer to v0.4.3? **Decision: Defer**
2. **Coverage**: All 44 examples or just 4-5 key ones? **Decision: 4-5 key ones**
3. **Auto-run**: Run tests on file load or only `ailang test`? **Decision: Only ailang test**
4. **Generators**: Full type coverage or start small? **Decision: Start with primitives + list**

## Files to Create

**New files**:
- `internal/testing/testing.go` - Framework core (~400 LOC)
- `internal/testing/collector.go` - Extract tests from AST (~150 LOC)
- `internal/testing/runner.go` - Execute tests (~250 LOC)
- `internal/testing/generator.go` - Property-based generators (~300 LOC)
- `internal/testing/shrink.go` - Input shrinking (~200 LOC)
- `internal/testing/result.go` - Test results (~100 LOC)
- `cmd/ailang/cmd_test.go` - CLI command (~150 LOC)
- `internal/testing/testing_test.go` - Unit tests (~500 LOC)

**Modified files**:
- `internal/parser/parser_decl.go` - Parse tests/properties (~300 LOC)
- `internal/ast/ast.go` - Add test AST nodes (~100 LOC)
- `internal/elaborate/elaborate.go` - Handle test nodes (~50 LOC)
- `cmd/ailang/main.go` - Register test command (~5 LOC)
- `examples/factorial.ail` - Add inline tests
- `examples/quicksort.ail` - Add properties

## Next Steps

1. Start Day 1: Implement `parseTestsBlock()`
2. Run parser tests continuously
3. Update this plan as we discover issues
4. Pause after each milestone for review

---

**Sprint Approved**: 2025-10-26
**Ready to Execute**: Yes
