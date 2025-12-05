# Sprint Plan: M-TESTING-DEPS (Cross-Function Dependencies)

## Summary

Enable inline tests for functions that depend on other user-defined functions in the same module, using SCC-based dependency analysis to build cluster harnesses.

**Duration:** 2-3 days
**Dependencies:** M-TESTING-INLINE-CORE (v0.4.5) - COMPLETE
**Risk Level:** Medium (SCC algorithm adds complexity)

## Current Status Analysis

### Completed Recently (M-TESTING-INLINE-CORE)
- ✅ Test harness builder: ~400 LOC in 0.5 days
- ✅ Test executor with Let/LetRec support: ~300 LOC in 0.5 days
- ✅ Example migration (98 tests, 9 files): ~500 LOC in 0.5 days
- ✅ Documentation updates: ~200 LOC in 0.5 days

**Total M-TESTING-INLINE-CORE:** ~1,467 LOC in 1 day (524% of estimate)

### Velocity
- Recent average: **~730 LOC/day** (based on M-TESTING-INLINE-CORE)
- Conservative estimate: **~400 LOC/day** (accounting for complexity)
- Estimated capacity for 2-3 day sprint: **800-1200 LOC**

### Remaining from Design Doc
- ⏳ Call Graph Builder: ~200 LOC
- ⏳ SCC Algorithm: ~100 LOC
- ⏳ Pure Cluster Extraction: ~100 LOC
- ⏳ Cluster Harness Builder: ~50 LOC changes
- ⏳ Integration + Tests: ~350 LOC

**Total estimated:** ~800 LOC

## Proposed Milestones

### Milestone 1: Call Graph & SCC Foundation
**Goal:** Build dependency graph from Core AST and compute strongly connected components
**Estimated:** 200 LOC implementation + 250 LOC tests = 450 LOC
**Duration:** 1 day

**Tasks:**
- **Morning (4h):**
  - Create `internal/testing/callgraph.go`
  - Implement `BuildCallGraph(module *core.Program) -> Graph`
  - Walk Core AST declarations, collect `Var` references per binding
  - Filter to only user-defined functions (exclude builtins)

- **Afternoon (4h):**
  - Implement Tarjan's SCC algorithm (or adapt existing Go graph lib)
  - Return `[][]string` where each inner slice is an SCC
  - Unit tests:
    - Single self-recursive function → singleton SCC
    - Chain (a→b→c) → three singleton SCCs
    - Mutual recursion (isEven↔isOdd) → one SCC with both
    - Diamond (a→b, a→c, b→d, c→d) → correct SCCs

**Acceptance Criteria:**
- [ ] `BuildCallGraph` correctly identifies function references
- [ ] Tarjan's SCC groups mutually recursive functions
- [ ] Unit tests cover single, chain, mutual, diamond patterns
- [ ] All tests passing
- [ ] Linting clean

**Example Files to Create:**
- `examples/test_mutual_recursion.ail` - isEven/isOdd with inline tests (prep for M2)

**Risks:**
- SCC algorithm complexity - Mitigation: Use well-tested algorithm from literature
- Edge case: builtins appearing as dependencies - Mitigation: Filter to module-local names only

---

### Milestone 2: Pure Cluster Extraction
**Goal:** Extract the pure dependency closure for a function under test
**Estimated:** 100 LOC implementation + 100 LOC tests = 200 LOC
**Duration:** 0.5 days

**Tasks:**
- **Morning (4h):**
  - Create `internal/testing/cluster.go`
  - Implement `ExtractPureCluster(funcName string, graph Graph, sccs [][]string, typeInfo *types.TypeInfo) ([]core.RecBinding, error)`
  - Find SCC containing target function
  - BFS/DFS from that SCC to collect all reachable SCCs
  - For each function in cluster, check effect row is `{}` (pure)
  - Return error if any dependency has effects

**Acceptance Criteria:**
- [ ] `ExtractPureCluster` returns correct bindings for lcm→gcd case
- [ ] Mutual recursion returns both functions in cluster
- [ ] Chain dependencies include full chain
- [ ] Effectful dependency returns clear error message
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Type info availability - Mitigation: TypeInfo already populated from pipeline, reuse it

---

### Milestone 3: Cluster Harness Builder
**Goal:** Modify harness builder to accept multiple bindings and synthesize cluster LetRec
**Estimated:** 50 LOC changes + 100 LOC tests = 150 LOC
**Duration:** 0.5 days

**Tasks:**
- **Afternoon (4h):**
  - Modify `BuildTestHarness` in `internal/testing/harness.go`
  - Accept `cluster []core.RecBinding` instead of single binding
  - Synthesize: `LetRec([all cluster bindings], test_body(target_func))`
  - Ensure binding order doesn't matter (LetRec is mutually recursive)
  - Integration tests with lcm/gcd example

**Acceptance Criteria:**
- [ ] Cluster harness compiles and evaluates correctly
- [ ] lcm tests pass when gcd is in cluster
- [ ] isEven/isOdd tests pass (mutual recursion)
- [ ] Backward compatible: single-function tests still work
- [ ] All tests passing
- [ ] Linting clean

**Example Files to Update:**
- `examples/snippets/v3_3/math/gcd.ail` - Enable lcm tests (currently commented)

**Risks:**
- Binding order issues - Mitigation: LetRec handles mutual recursion; order shouldn't matter

---

### Milestone 4: Integration & Polish
**Goal:** Wire everything together and add debug tooling
**Estimated:** 80 LOC implementation + 120 LOC tests = 200 LOC
**Duration:** 0.5 days

**Tasks:**
- **Morning (4h):**
  - Update `TestExecutor` to use call graph + cluster extraction
  - Add `--dump-inline-harness` flag (optional, for debugging)
  - Enable lcm tests in `examples/snippets/v3_3/math/gcd.ail`
  - Create `examples/test_mutual_recursion.ail` with isEven/isOdd tests
  - Run full test suite

- **Afternoon (2h):**
  - Update documentation
  - Add entry to CHANGELOG.md
  - Final verification: `make test && make lint`

**Acceptance Criteria:**
- [ ] `ailang test gcd.ail` passes lcm tests
- [ ] `ailang test mutual_recursion.ail` passes isEven/isOdd tests
- [ ] `--dump-inline-harness` shows cluster composition (if implemented)
- [ ] Documentation updated
- [ ] CHANGELOG entry added
- [ ] All tests passing
- [ ] Linting clean

**Example Files:**
- `examples/test_mutual_recursion.ail` - NEW: isEven/isOdd flagship example
- `examples/snippets/v3_3/math/gcd.ail` - UPDATED: lcm tests enabled

**Risks:**
- Integration complexity - Mitigation: Test each layer independently first

---

## Day-by-Day Schedule

| Day | Milestone | Tasks | LOC Target |
|-----|-----------|-------|------------|
| 1 | M1: Call Graph & SCC | Build graph, implement Tarjan's, unit tests | 450 LOC |
| 2 AM | M2: Pure Cluster | Cluster extraction, purity gating | 200 LOC |
| 2 PM | M3: Cluster Harness | Multi-binding harness, integration tests | 150 LOC |
| 3 | M4: Integration | Wire up, debug flag, docs, examples | 200 LOC |

**Total:** ~1000 LOC across 2-3 days

---

## Success Metrics

- [ ] Test coverage for new code: >80%
- [ ] Examples passing:
  - [ ] `gcd.ail` with lcm tests
  - [ ] `test_mutual_recursion.ail` with isEven/isOdd
- [ ] Documentation updated:
  - [ ] `docs/docs/reference/language-syntax.md` - inline test section
  - [ ] `CHANGELOG.md` - v0.4.8 entry
- [ ] All tests passing: ✅
- [ ] All linting passing: ✅

---

## Dependencies

- **M-TESTING-INLINE-CORE** (COMPLETE): Provides base harness builder and executor
- **TypeInfo availability**: Need effect row info from type checker (already available)

---

## Open Questions

1. **Harness dump format**: Should `--dump-inline-harness` emit pretty-printed Core AST or pseudo-code?
2. **Performance threshold**: At what cluster size should we warn about large test harnesses?
3. **Property tests**: After DEPS, should property tests use same cluster mechanism? (Answer: Yes, per user feedback)

---

## Notes

- SCC algorithm is well-understood (Tarjan's 1972); focus on correct implementation
- Cluster extraction is the key innovation; pure gating prevents surprises
- isEven/isOdd is the flagship example for mutual recursion support
- lcm/gcd is the flagship example for simple dependencies

---

**Document created**: 2025-11-27
**Last updated**: 2025-11-27
