# Sprint Plan: M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (v0.20.0)

**Design doc**: [m-typecheck-no-auto-unwrap-result.md](./m-typecheck-no-auto-unwrap-result.md)
**Target**: v0.20.0
**Estimated**: ~3 days (~20 hours) per design doc, but per recent velocity (M-WASM-AI-STEP-BYO-KEY ~835 LOC in 1 session × 2 evaluator rounds; M-EXT-PORTABILITY-GATE ~1500 LOC in 2 sessions) — realistic estimate is **2 sessions / ~10 hours of focused work**.
**Risk level**: **Medium-High** — touches `internal/types/typechecker.go` (regression-surface trigger per evaluator rubric), changes the semantics of an established language construct, requires audit-fix sweep across stdlib + downstream packages
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-14

---

## Discovery (pre-planning)

Verified what's in place vs needs creating:

| Component | State |
|---|---|
| `internal/types/typechecker_data.go::inferRecordAccess` (line 60) | EXISTS. Constraint-based: unifies receiver type with `TRecordOpen{field: T \| r}`. Need to add a pre-check that rejects tagged-union receivers BEFORE this constraint fires. |
| `internal/types/typechecker.go` family | 8 typechecker_*.go files, ~3,000 LOC total. `inferRecordAccess` is the single touch point. |
| ADT registry | Lives in TypeEnv. Constructor count per ADT is needed for the `isTaggedUnion` predicate. |
| Existing `RowVar` / `TRecordOpen` machinery | EXISTS. The new gate runs BEFORE the row-unification path, so no interaction. |
| Structured error envelope (`internal/elaborate/error_codes.go`) | EXISTS. New error code `TYP_RECORD_ACCESS_ON_TAGGED_UNION` registered alongside the existing TYP_* codes. |
| `cmd/ailang/main.go` flag plumbing | EXISTS. Pattern for `--debug-compile`, `--no-mono` etc. Adding `--allow-unsafe-field-access` follows the same shape. |
| `make verify-examples` | EXISTS. Will catch any in-repo example that regresses. |
| Registry-validator (M-EXT-PORTABILITY-GATE infra) | EXISTS. Reusable for the published-package drift check. |
| Test infrastructure | `internal/types/*_test.go` covers ~600 tests. Adding `tagged_union_field_access_test.go` follows the same pattern. |
| Related conditional rule | `.claude/rules/type-system.md` — reminds: ast.Type switch exhaustiveness, cycle-safe traversal. The `isTaggedUnion` predicate touches Type values; must use `traverse.Walk` or carry a `visited` parameter. |

**Velocity calibration**: This sprint is small in LOC (~265 implementation + ~150 tests + ~30-60 audit fixups). Comparable to M-WASM-AI-STEP-BYO-KEY's M1+M2+M3 code (no demo). Realistic: **1 session of focused work for M1, half a session each for M2 and M3** — call it 2 sessions total.

---

## Milestones

### M1 — Type-checker core: `isTaggedUnion` predicate + record-access gate (~280 LOC, ~3 hours)

**Goal**: AILANG type-checker rejects `.field` access on a tagged-union-typed receiver. Single-constructor ADTs and ordinary records continue to work.

**Tasks**:

1. **Add `isTaggedUnion(t Type, env *TypeEnv) bool` predicate** (NEW `internal/types/tagged_union_predicate.go`, ~80 LOC):
   - Resolves type aliases, unwraps `TApp` to head `TCon`
   - Looks up the type-constructor name in `env`'s ADT registry
   - Returns `true` if registered ADT has >1 constructor
   - Returns `false` for: `TCon` of primitive (`int`, `string`, `bool`, ...), `TRecord` / `TRecordOpen`, `TFunc`, `TVar`, function types, single-constructor ADTs, unregistered names
   - Cycle-safe: uses `traverse.Walk` or carries a `visited` map (per type-system.md rule)

2. **Wire the gate into `inferRecordAccess`** (`internal/types/typechecker_data.go`, ~+30 LOC):
   - After `tc.inferCore(ctx, acc.Record)` resolves the receiver
   - Before the existing `TRecordOpen` constraint
   - Call `isTaggedUnion(getType(recordNode), ctx.env)`; on true, return error
   - Error message includes: receiver type name, first 3 constructor names, prescriptive `match` template

3. **Register `TYP_RECORD_ACCESS_ON_TAGGED_UNION` error code** (`internal/elaborate/error_codes.go`, ~+15 LOC):
   - Standard structured-error envelope shape (matches existing TYP_* codes)
   - Hint template: `"$RECEIVER_TYPE returns Result[T, E] (or similar tagged union). Use match: match $EXPR { Ok(x) => x.field, Err(e) => /* handle e */ }"`

4. **Add `--allow-unsafe-field-access` migration flag** (`cmd/ailang/main.go`, ~+10 LOC):
   - Default off → `TYP_RECORD_ACCESS_ON_TAGGED_UNION` is a hard error
   - When set → downgrades to WARN-level diagnostic (logged loudly per occurrence)
   - One-version grace; removed in v0.21.0

5. **Failing-tests-first** (NEW `internal/types/tagged_union_field_access_test.go`, ~150 LOC):
   - `TestTaggedUnionFieldAccess_Rejects_Result` — `Result[Int, String].foo` → error
   - `TestTaggedUnionFieldAccess_Rejects_Option` — `Option[Record{a:int}].a` → error
   - `TestTaggedUnionFieldAccess_Rejects_UserADT` — multi-variant user ADT → error
   - `TestTaggedUnionFieldAccess_Allows_SingleConstructorADT` — `type Wrap = Wrap({x:int})` then `w.x` → OK
   - `TestTaggedUnionFieldAccess_Allows_PlainRecord` — `Record{a:int}.a` → OK
   - `TestTaggedUnionFieldAccess_Allows_InsideMatchArm` — `match r { Ok(x) => x.field }` → OK
   - `TestTaggedUnionFieldAccess_MigrationFlag_DowngradesToWarning` — flag set → no error, warning logged
   - `TestIsTaggedUnion_HandlesAliases` — type alias to a Result resolves correctly
   - `TestIsTaggedUnion_CycleSafe` — recursive type doesn't loop forever

**Acceptance**:
- [ ] `internal/types/tagged_union_predicate.go` exists with `isTaggedUnion`, cycle-safe per `.claude/rules/type-system.md`
- [ ] `inferRecordAccess` calls the gate before the `TRecordOpen` constraint
- [ ] `TYP_RECORD_ACCESS_ON_TAGGED_UNION` registered with prescriptive hint template
- [ ] `--allow-unsafe-field-access` flag plumbed through `cmd/ailang/main.go`, downgrades to warning
- [ ] All 9 unit tests pass
- [ ] Existing `internal/types/` test suite (~600 tests) still passes — no regression
- [ ] `make test` + `make lint` clean
- [ ] Single-constructor ADT exception verified: `type Wrap = Wrap({x:int})` then `w.x` works

**Risk**: Med — touches the typechecker. Mitigation: failing-tests-first, single-ctor exception in test fixture, full existing test suite must stay green.

---

### M2 — In-repo audit sweep + fix every flagged callsite (~50-150 LOC, ~2-4 hours)

**Goal**: Every `.ail` file in this repo type-checks under the new strict rule. Catalog any legitimate auto-field-access uses and decide: fix with `match`, or qualify with the same-field-shape exception (if landed).

**Depends on**: M1 (the strict checker must exist before we can sweep)

**Tasks**:

1. **Build `tools/find_unsafe_field_access.sh`** (NEW, ~50 LOC):
   - Greps for `let X = funcThatReturnsResult(...)` followed by `X.field`
   - Limited (regex, not type-aware) but catches obvious cases
   - Output: `file:line:expression`

2. **Run the strict checker against every `.ail` in repo**:
   - `examples/` (root + subdirs, including `examples/runnable/`)
   - `std/` (stdlib `.ail` modules)
   - `examples/expected_fail/` (these expect failures, but check the failure mode is the right one)
   - Capture full failure list

3. **Fix every flagged callsite**:
   - Most fixes are 1-line: wrap in `match` with `Ok` and `Err` arms
   - Where the original code had no error handling at all, add a sensible default (e.g., empty list, error string, propagate via `?` if AILANG had it — falls back to fold)
   - If count >150 → STOP, reassess: ship the migration flag with WARN-default for one cycle (per design-doc fallback plan), defer hard-error to v0.20.1

4. **Add the compaction_ai 0.1.3 source as a regression fixture**:
   - `examples/expected_fail/compaction_ai_field_access_on_result.ail` — reproduces the exact bug
   - Pinned by file-name to v0.1.3 of the package for traceability
   - `make verify-examples` exercises the expected-fail case

**Acceptance**:
- [ ] `tools/find_unsafe_field_access.sh` exists and is documented
- [ ] Audit-list captured (committed under design_docs as `m-typecheck-audit-results.md`)
- [ ] Every `.ail` file in repo type-checks under strict mode
- [ ] `examples/expected_fail/compaction_ai_field_access_on_result.ail` exists and triggers TYP_RECORD_ACCESS_ON_TAGGED_UNION
- [ ] `make verify-examples` green
- [ ] Total fix count ≤150 (or fallback path documented + executed if not)

**Risk**: Med — audit count is the unknown. Mitigation: fallback plan in design doc (warn-then-error over two minor versions).

---

### M3 — Ecosystem migration + docs + CHANGELOG + ship (~150 LOC, ~3 hours)

**Goal**: Every published `sunholo/*` package type-checks under v0.20.0 strict mode. Documentation + prompts + CHANGELOG updated. Sprint design doc moved to implemented.

**Depends on**: M1 + M2

**Tasks**:

1. **Drift-check published packages**:
   - Reuse the M-EXT-PORTABILITY-GATE registry-validator infrastructure
   - Iterate every `sunholo/*` package on the registry
   - Type-check each with the v0.20.0 strict checker
   - Catalog failures (list of `package@version: file:line`)

2. **PR fixes per affected package**:
   - For each failure: fix the package, bump patch version, publish, comment on the package's repo
   - Mirror the pattern used for `motoko_ext_compaction_ai 0.1.4` (the reactive fix)
   - Aim for clean migration before v0.20.0 tag

3. **Update AILANG prompts** (`prompts/v0.20.0/syntax.md`, `devtools.md`, `agent.md`):
   - Lead with the discipline: every `Result`/`Option`/multi-variant ADT consumer MUST `match`
   - Show the new prescriptive error message verbatim
   - Mark the old `result.field` pattern as REJECTED with a fix template

4. **Ship docs guide** (`docs/docs/guides/result-discipline.md`, ~100 LOC):
   - Why the gate exists (link to compaction_ai post-mortem)
   - The `isTaggedUnion` predicate's exact rules (single-ctor ADT exempt, primitive / record exempt)
   - Migration flag usage + when to remove it
   - Three concrete before/after examples
   - Wire into `docs/sidebars.js` under "Reference"

5. **CHANGELOG entry under `[v0.20.0]`** in `changelogs/v0.10-current.md`:
   - Breaking change clearly flagged
   - Migration path: `--allow-unsafe-field-access` for one cycle
   - Link design doc + audit results + ecosystem migration list
   - Acknowledge motoko_ext_compaction_ai 0.1.3 as the surfacing case

6. **Move design doc to implemented**:
   - `design_docs/planned/v0_20_0/m-typecheck-no-auto-unwrap-result.md` → `design_docs/implemented/v0_20_0/`
   - Same for sprint plan
   - Update Status: Planned → Implemented

**Acceptance**:
- [ ] Every published `sunholo/*` package type-checks under strict mode (or has a queued PR fix)
- [ ] AILANG prompts updated to teach the new discipline
- [ ] `docs/docs/guides/result-discipline.md` ships, wired into sidebar
- [ ] CHANGELOG entry under `[v0.20.0]` with migration path
- [ ] Design doc + sprint plan moved to `implemented/v0_20_0/`
- [ ] All sprint tests + lint clean
- [ ] Sprint-evaluator scores ≥85/100

**Risk**: Med — ecosystem migration is the slowest step (cross-repo PRs, registry republish). Mitigation: parallelize with the in-repo work; ship AILANG with the strict checker even if 1-2 packages are still mid-migration (their consumers can use `--allow-unsafe-field-access` until they update).

---

## Day-by-day breakdown

| Session | Milestones | Hours | Deliverable |
|---|---|---|---|
| 1 (am) | M1: predicate + gate + tests | 3h | `tagged_union_predicate.go`, `inferRecordAccess` updated, 9 unit tests passing, error code + flag |
| 1 (pm) | M2: audit sweep + fixes | 3h | `find_unsafe_field_access.sh`, audit list, all in-repo `.ail` files type-check |
| 2 (am) | M3 part 1: ecosystem drift-check + per-package PRs | 2h | Published packages catalog, fix PRs queued |
| 2 (pm) | M3 part 2: docs + prompts + CHANGELOG | 2h | Result-discipline guide, prompts updated, CHANGELOG entry, design doc moved to implemented |

**Total: ~10 hours = 2 focused sessions.**

---

## Repo coordination

| Repo | Branch | What lands |
|---|---|---|
| `sunholo-data/ailang` | `dev` | M1 + M2 + M3 (typechecker, audit, docs) |
| `sunholo-data/ailang-packages` | `dev` | Per-package fix branches if drift-check finds any |
| Downstream consumer projects | per-project | Migration via `--allow-unsafe-field-access` flag if needed; package bumps land via cascade |

Single-repo for the AILANG-side work. Cross-repo cascade only if drift-check finds package failures.

---

## Success metrics

- **Type-checker rejection works**: `result.message.content` on a `Result`-typed value → compile error, not runtime crash
- **Single-ctor ADT exception works**: `type Wrap = Wrap({x:int})` then `w.x` continues to compile
- **In-repo green**: every `.ail` file under `examples/` + `std/` type-checks under strict mode
- **Ecosystem green**: every published `sunholo/*` package type-checks (or has a queued fix PR)
- **AI agents teach correctly**: `ailang prompt` v0.20.0 leads with the new discipline; prescriptive error messages enable one-shot fixes
- **Migration flag works both ways**: error by default; warning under `--allow-unsafe-field-access`
- **Regression fixture**: compaction_ai 0.1.3 source pinned in `examples/expected_fail/` triggers the right error code

---

## Risks

| Risk | Mitigation |
|---|---|
| `inferRecordAccess` change breaks something subtle in row-unification | M1 keeps full ~600-test suite green as gating criterion. Failing-tests-first style. |
| `isTaggedUnion` cycle-unsafe on recursive types | Cycle-safe per `.claude/rules/type-system.md`. Test fixture pins behavior on a recursive ADT. |
| Audit count balloons past 150 | Fallback plan in design doc: ship migration flag with WARN-default, hard-error in v0.20.1. Documented in M2 acceptance. |
| Published packages slow to migrate | `--allow-unsafe-field-access` flag gives consumers a one-version escape hatch. Ship anyway. |
| Single-constructor ADT exception accidentally rejects struct-emulation patterns | Test fixture explicitly pins the "type Wrap = Wrap({...})" pattern as ALLOWED. |
| AI agents emit `result.field` patterns from training | New error message includes prescriptive fix template — agents one-shot to correct via the same loop they used for the M-WASM smoke fix. |

---

## Files modified

| File | Change | LOC |
|---|---|---|
| `internal/types/tagged_union_predicate.go` | NEW: isTaggedUnion predicate | +80 |
| `internal/types/tagged_union_field_access_test.go` | NEW: 9 unit tests | +150 |
| `internal/types/typechecker_data.go` | inferRecordAccess gate | +30 |
| `internal/elaborate/error_codes.go` | TYP_RECORD_ACCESS_ON_TAGGED_UNION | +15 |
| `cmd/ailang/main.go` | --allow-unsafe-field-access flag | +10 |
| `tools/find_unsafe_field_access.sh` | NEW: audit script | +50 |
| `examples/expected_fail/compaction_ai_field_access_on_result.ail` | NEW: regression fixture | +20 |
| `examples/runnable/*.ail` + `std/*.ail` | Audit-fix sweep (~30-60 sites) | +60 |
| `prompts/v0.20.0/syntax.md` | Result-discipline lead | +30 |
| `docs/docs/guides/result-discipline.md` | NEW: full guide | +100 |
| `docs/sidebars.js` | Sidebar wiring | +1 |
| `changelogs/v0.10-current.md` | [v0.20.0] entry | +60 |
| `design_docs/planned/v0_20_0/` → `implemented/v0_20_0/` | Move on completion | 0 |
| **Total** | | **~600 LOC** |

---

## Notes for the executor

- **No design doc gaps known** — design doc is comprehensive (333 LOC, axiom score +8, conflict surface analyzed)
- **No cross-repo coordination required for M1+M2** — only M3 ecosystem migration touches other repos
- **No new dependencies** — pure type-checker change + helper script
- **Pre-verified infrastructure** — `inferRecordAccess`, `TypeEnv` ADT registry, error envelope, registry-validator infra all already shipped
- **Hand-off pattern**: after each milestone, run `make test` + `make lint` + `make verify-examples` before proceeding
- **Type-system rules apply** — see `.claude/rules/type-system.md` (ast.Type 8-variant exhaustiveness, cycle-safe traversal, CoreTypeInfo invariant). The `isTaggedUnion` predicate must respect these.
- **Conflict surface trigger** — this sprint touches `internal/types/`. The sprint-evaluator's regression-surface conditional category will fire. Make sure the regression fixtures listed in the design doc's Conflict Surface section are added in M2.
- **Browser smoke-test not applicable** — this is a typechecker change, not a runtime/WASM change. Standard `make test` + `make verify-examples` is sufficient verification.

---

**Document created**: 2026-05-14
**Last updated**: 2026-05-14
