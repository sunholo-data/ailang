# Sprint Plan: M-CONCAT-DISAMBIG — Eliminate `++` Ambiguity

## Summary
Implement string interpolation (Phase 1, ships as **v0.12.1**) and restrict `++` to lists only (Phase 2, ships as **v0.13.0**), completing a 4-release effort to remove the #1 source of agent compile-error failures.

**Duration:** 5 days (14–19 hours)
**Target versions:** v0.12.1 (Phase 1), v0.13.0 (Phase 2)
**Design doc:** [m-concat-disambiguation.md](m-concat-disambiguation.md)
**Dependencies:** None — Phase 2 depends on Phase 1 shipping but the sprint executes both sequentially.
**Risk Level:** Medium — compiler touch-points are well understood, but the ~135-file example migration has high file-count blast radius.

---

## Current Status Analysis

### Completed Recently (context)
- ✅ v0.12.0 released (2026-04-14) — std/tar + std/gzip, Gemini config fix, poly Ord/Eq defaulting
- ✅ M-PERF5 3/4 milestones complete (DocParse optimizations)
- ✅ v0.11.4 released with prompt tweaks — confirmed prompt alone cannot close the `++` gap

### v0.12.0 Baseline Evidence (driving this sprint)
From `eval_results/baselines/v0.12.0/summary.jsonl` (408 runs, 6 frontier models, prompt v0.11.4):
- `type_unify`: **5/8 models fail** with `cannot unify string with *types.TList`
- `config_file_parser`: **3/8 models fail** from runtime + codegen fallout of same heuristic
- Frontier model upgrades (Sonnet 4.6, Opus 4.7, GPT-5.4, Gemini-3.1 Pro) did **not** resolve the issue — language defect, not prompt defect

### Velocity Snapshot
- M-PERF5 perf sprint: ~400-600 LOC/day measured
- M-STDLIB-TAR-GZIP delivered in a single day
- Conservative target for this sprint: ~200-300 LOC/day (examples sweep dominates file count but LOC is mechanical)

### Remaining from Design Doc
- Phase 1 (interpolation): ~165 LOC new, 0 LOC removed → +165 net
- Phase 2 (compiler): ~95 LOC added, ~182 LOC removed → **-87 LOC (compiler gets simpler)**
- Phase 2 (examples/prompts/docs): 135 files + 2 yml + 1 prompt + ~30 docs

---

## Proposed Milestones

### Milestone M1_LEXER_INTERP: String Interpolation Lexer
**Goal:** Emit `STRING_PART`, `INTERP_START`, `INTERP_END` tokens for `"...${expr}..."` literals.
**Estimated:** ~40 LOC impl + ~30 LOC tests = ~70 LOC
**Duration:** 2 hours (Day 1)

**Tasks:**
- Add `STRING_PART`, `INTERP_START`, `INTERP_END` token kinds in `internal/lexer/token.go`
- Detect `${` inside string literals; emit token sequence with brace counting for nested `{}`
- Handle escaped `\${` → literal dollar-brace
- Preserve existing `STRING` token path for non-interpolated strings (zero regression)

**Acceptance Criteria:**
- [ ] `"hi ${name}"` tokenizes to `STRING_PART("hi ") INTERP_START IDENT(name) INTERP_END STRING_PART("")`
- [ ] Nested braces work: `"${compute({a: 1}.a)}"`
- [ ] `"\${literal}"` produces a single `STRING` token with literal `${literal}`
- [ ] Plain string `"hello"` still emits a single `STRING` token (no change)
- [ ] Lexer unit tests cover all four cases above
- [ ] `make test` and `make lint` clean

**Risks:**
- Nested brace counting edge cases — Mitigation: mirror the JS/Kotlin/Dart algorithm, add explicit tests for depth ≥3.

---

### Milestone M2_PARSER_TYPECHECK_INTERP: Parser Desugar + Type Checker
**Goal:** Parse interpolated strings into a concat chain AST and type-check with auto-`show()` insertion.
**Estimated:** ~45 LOC impl + ~30 LOC tests = ~75 LOC
**Duration:** 2.5 hours (Day 1)

**Tasks:**
- Parser: consume `STRING_PART / INTERP_START expr INTERP_END` sequence, desugar to `_str_concat` chain
- Desugar rule: `"Hi ${x}"` → `_str_concat(_str_concat("Hi ", show(x)), "")`
- Skip `show()` wrapping when interpolated expression is already string-typed
- Type checker: verify interpolated exprs are `string`-showable; emit clear error if not
- Evaluator + codegen: **zero change expected** (desugaring happens before eval)

**Acceptance Criteria:**
- [ ] `"Hello, ${name}"` evaluates correctly when `name: string`
- [ ] `"count: ${x + 1}"` evaluates correctly for `x: int` (auto-show)
- [ ] String-typed expression doesn't get double-wrapped: `"${s}"` → `s`, not `show(s)`
- [ ] Non-showable type produces clear error
- [ ] Parser/type checker unit tests
- [ ] `make test` clean

**Risks:**
- Desugaring trailing empty `STRING_PART("")` adds a needless concat — Mitigation: trim empty parts in desugar pass.

---

### Milestone M3_INTERP_TESTS_PROMPT: Integration Tests + Prompt Update (Phase 1 Ships)
**Goal:** End-to-end integration tests pass; teaching prompt updated to prefer `"${...}"`; Phase 1 ready to release as v0.12.1.
**Estimated:** ~60 LOC tests + ~10 LOC prompt = ~70 LOC
**Duration:** 2 hours (Day 2)

**Tasks:**
- Write integration tests in `internal/pipeline/` covering interpolation end-to-end
- Add 2-3 runnable `.ail` examples demonstrating interpolation (e.g., `examples/string_interpolation.ail`)
- Update active prompt (`prompts/v0.9.0.md` or new `v0.12.1.md`): add interpolation section with examples
- Keep string `++` still working — agents migrate naturally
- Update `docs/docs/reference/limitations.md`: mark string interpolation as **implemented**
- Tag Phase 1 as v0.12.1 (defer release to post-milestone)

**Acceptance Criteria:**
- [ ] `make verify-examples` passes — no existing examples broken
- [ ] `make ci` passes
- [ ] Active teaching prompt includes `"${...}"` as preferred string-building syntax
- [ ] 2+ new examples in `examples/` showing interpolation
- [ ] CHANGELOG.md entry for v0.12.1

**Risks:**
- Desugaring interaction with module-level string constants — Mitigation: test `let s = "${x}"` at module scope explicitly.

---

### Milestone M4_CONCAT_STDLIB_LIST_ONLY: `concat()` Stdlib + Restrict `++` to Lists
**Goal:** Add `concat(parts: [string])` stdlib and remove the type-checker string/list heuristic — `++` types as `[a] -> [a] -> [a]`, period.
**Estimated:** ~95 LOC added, ~120 LOC removed → net **-25 LOC** (compiler simpler)
**Duration:** 3 hours (Day 3)

**Tasks:**
- Add `_str_concat_list` builtin (alias for `join("", parts)`) in `internal/builtins/`
- Register stdlib `concat(parts: [string]) -> string` in std/string
- `internal/types/typechecker_operators.go`: delete 118-line heuristic (lines 104-221), replace with ~30 lines of list-only typing
- Emit helpful error on string operand: `"++ is for lists. Use string interpolation or join()/concat() for strings."`
- `internal/pipeline/op_table.go`: remove "String" from concat types
- `internal/pipeline/op_lowering.go`: remove string fallback
- Evaluator: `OpConcat` always calls `_list_concat`
- Go/WASM/SMT codegen: `++` always emits `ConcatList`

**Acceptance Criteria:**
- [ ] `[1,2] ++ [3,4]` works (unchanged)
- [ ] `xs ++ ys` works when both are `[a]` type vars (recursive/polymorphic context)
- [ ] `"a" ++ "b"` produces helpful error pointing to interpolation
- [ ] `concat(["a", "b"])` returns `"ab"`
- [ ] `make test` passes (expect breakage in concat_operator_test.go — handled in M5)
- [ ] `DEBUG_OPERATOR_LOWERING=1` confirms `++` always routes to `concat_List`

**Risks:**
- String interpolation desugaring still uses `_str_concat` internally — must remain wired up. Mitigation: explicit test that interpolation works after Phase 2 changes.
- Unused `_str_concat` builtin — keep it; still used by interpolation desugaring.

---

### Milestone M5_GO_TEST_MIGRATION: Migrate Go Test Suite
**Goal:** Update Go tests to reflect `++` list-only semantics; add regression tests for helpful-error path.
**Estimated:** +60 LOC / -80 LOC = **-20 LOC net**
**Duration:** 1.5 hours (Day 3)

**Tasks:**
- `internal/pipeline/concat_operator_test.go`:
  - Remove `TestConcatRecursiveString`, `TestConcatConcreteString`
  - Keep `TestConcatListWithSignature`, `TestConcatConcreteList`
  - Add `TestConcatStringErrorMessage` — verify error text
  - Add `TestConcatListPolymorphic` — `[a]` type var case
- `internal/types/operator_method_test.go`: update `++` inference expectations
- `internal/pipeline/op_lowering_test.go`: remove `concat_String` paths
- `internal/builtins/string_test.go`: add `concat()` tests
- `internal/gen/golang/codegen_test.go`: verify `ConcatList` only

**Acceptance Criteria:**
- [ ] `make test` clean
- [ ] No skipped tests
- [ ] Error-message test asserts the exact helpful-error string

**Risks:**
- Test file imports may need trimming — Mitigation: run `go vet` after edits.

---

### Milestone M6_EXAMPLE_MIGRATION: Sweep 135 `.ail` Examples
**Goal:** Migrate all string `++` usages to interpolation; verify every example still runs.
**Estimated:** ~400 LOC edits (mechanical)
**Duration:** 3 hours (Day 4)

**Tasks:**
- Runnable examples (`examples/runnable/` ~100 files): `"..." ++ show(x)` → `"...${show(x)}"`
- Bug/debug examples (`examples/bugs/`, `examples/debug/`):
  - `concat_operator_list_inference.ail` — now tests correct behavior
  - `list_concat_match.ail`, `list_concat.ail` — simplify
- Doc/test examples (`examples/docs/`, `examples/tests/`): sweep remaining
- Leave list `++` as-is (now correct)
- Consider migration script to avoid manual edits

**Acceptance Criteria:**
- [ ] 0 string `++` in `examples/` (grep check)
- [ ] `make verify-examples` passes — every example still runs
- [ ] Example verification status updated
- [ ] List `++` usages preserved unchanged

**Risks:**
- Mechanical regex misses edge cases like `a ++ "literal"` — Mitigation: make the script print untouched `++` occurrences for manual review; require zero remaining before completing.
- `make verify-examples` latency — Mitigation: batch-verify rather than per-file.

---

### Milestone M7_BENCHMARK_PROMPT_DOCS: Benchmarks + Active Prompt + Docs Sweep
**Goal:** Update benchmark YAMLs, active teaching prompt, and ~30 docs files to reflect list-only `++`.
**Estimated:** ~80 LOC edits
**Duration:** 2.5 hours (Day 5 morning)

**Tasks:**
- **Benchmarks:**
  - `benchmarks/type_unify.yml` — no change needed (spec already uses `++` for lists; **previously impossible task is now solvable**)
  - `benchmarks/symbolic_diff.yml` — migrate 4 description lines to interpolation
  - Sweep `benchmarks/**/*.ail` solution files
- **Active prompt** (`prompts/v0.12.1.md` or successor):
  - Remove "NO `++` for lists" caveat
  - Add operator table with `++`, `::`, `"${expr}"`, `join`, `concat`
  - Update 5+ inline examples to use interpolation
- **Docs** (`docs/docs/` ~30 files):
  - `reference/language-syntax.md` — operator table
  - `LIMITATIONS.md` — add resolved note
  - `reference/effects.md`, `guides/testing.md`, `guides/debugging.md`, `architecture/anf.md` — example code
  - Sweep remaining ~24 files

**Acceptance Criteria:**
- [ ] 0 string `++` in active prompt
- [ ] 0 string `++` in `docs/` (grep check)
- [ ] Operator table present in prompt + language-syntax doc
- [ ] CHANGELOG.md updated with v0.13.0 entry

**Risks:**
- Historical prompt versions (v0.6.x–v0.9.x) — Mitigation: **only update active prompt**; leave older versions as historical artifacts.

---

### Milestone M8_FINAL_VALIDATION: Eval Subset + Release Gate
**Goal:** Full CI green; eval subset proves the defect is closed; v0.13.0 ready to tag.
**Estimated:** ~20 LOC (release notes, CHANGELOG polish)
**Duration:** 1–1.5 hours (Day 5 afternoon)

**Tasks:**
- `make ci` clean (all tests, lint, example verify)
- Run eval subset on `type_unify`, `config_file_parser`, `graph_bfs` across 2-3 models
- Compare pass rates vs v0.12.0 baseline; capture in CHANGELOG
- Update `design_docs/planned/v0_13_0/m-concat-disambiguation.md` → move to `design_docs/implemented/v0_13_0/`
- Tag handoff to release-manager

**Acceptance Criteria:**
- [ ] `make ci` green
- [ ] `type_unify` benchmark: pass rate improves from ~3/8 to ≥6/8 on frontier models
- [ ] `config_file_parser` pass rate improves
- [ ] CHANGELOG.md includes measured before/after eval numbers
- [ ] Design doc moved to `implemented/`
- [ ] Ready to hand off to `release-manager` skill

**Risks:**
- Eval subset cost — Mitigation: run on 2 models (Sonnet 4.6, Opus 4.7) rather than all 6; record as preliminary signal.

---

## Success Metrics

- **Compiler:** ~87 LOC net simpler in `typechecker_operators.go` + `op_lowering.go`
- **Eval gate:** `type_unify` pass rate ≥6/8 frontier models (from ~3/8)
- **Teaching prompt:** Zero "NO `++` for lists" caveats
- **Examples:** 135 files migrated, `make verify-examples` green
- **Docs:** String interpolation moved from "Not implemented" to "Implemented"
- **Test coverage:** Maintained (no drop vs v0.12.0 baseline)

---

## Release Plan

| Version | Contents | Milestones |
|---------|----------|------------|
| **v0.12.1** | String interpolation (additive, zero breakage) | M1 + M2 + M3 |
| **v0.13.0** | `++` list-only + full migration sweep | M4 + M5 + M6 + M7 + M8 |

Phase 1 can ship independently as v0.12.1 even if Phase 2 slips. This de-risks the sprint — worst case, v0.13.0 is a separate follow-up sprint.

---

## Dependencies

**Blocking:** None.
**Blocked by this sprint:**
- `benchmarks/type_unify.yml` becomes solvable → expect improved eval baselines in v0.13.0
- Future type-class work (Monoid, Semigroup) — cleaner starting point with `++` unambiguous

---

## Open Questions

1. **Prompt versioning:** Should v0.12.1 get its own prompt file (`prompts/v0.12.1.md`) or update the latest in place? Design doc says update in place — confirming.
2. **Example migration script:** Write a one-shot Go/Python migration script, or do it manually? Script recommended given the 135-file scope.
3. **Benchmark re-eval:** Run full 8-model eval suite after v0.13.0 ships, or defer to normal post-release cadence? Defer to post-release.

---

## Notes

- The design doc is unusually well-scoped: Phase 2A-2I sub-tasks map 1:1 to milestone breakdown.
- String `++` remains functional after Phase 1 — no breakage until Phase 2 lands.
- The `_str_concat` builtin is **kept** throughout — it's the desugaring target for interpolation.
- Historical prompt versions (`prompts/v0.6.6.md` through `prompts/v0.9.0.md`) left untouched; only the active prompt changes. Archival consistency isn't worth the churn.
- Migration script for examples (M6) could be authored in Python or as a `tools/migrate-plus-plus.sh` shell script using `sed` + verification loop.

---

**Plan created:** 2026-04-20
**Author:** sprint-planner (Opus 4.7)
**Supersedes:** n/a (first sprint plan for this feature)
