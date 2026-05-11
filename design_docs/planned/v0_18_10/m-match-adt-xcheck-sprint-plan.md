# Sprint Plan: M-MATCH-ADT-XCHECK (v0.18.10)

**Design doc**: [m-match-adt-xcheck.md](./m-match-adt-xcheck.md)
**Target**: v0.18.10 (patch release)
**Estimated duration**: 2 days (~450 LOC)
**Risk level**: Low — infrastructure already exists, surgical change
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-11

---

## Discovery

Pre-planning investigation found the relevant infrastructure already exists:

1. **`tc.constructorTypes map[string]string`** in `internal/types/typechecker_core.go:78` already maps constructor names → ADT type names (populated by `pipeline/pipeline_*.go`). Comment says: "M-DX25.4: Constructor name → ADT type name (e.g., 'Up' → 'Direction')".

2. **`tc.adtTypeParams map[string]int`** at line 79 tracks ADT → number-of-type-params (used for TApp construction).

3. **`internal/types/typechecker_patterns.go`** is the single file that handles `ConstructorPattern` (line 146). The check needs to be inserted at line 149 BEFORE the existing TApp/TCon construction.

4. **`internal/types/errors.go`** has `TypeErrorKind` enum + `TypeCheckError` struct ready to extend.

**Implication**: This sprint is smaller than the design doc's ~400-600 LOC estimate. Realistic total: ~450 LOC (Go implementation ~80, tests ~250, docs+examples ~120).

---

## Milestones

### M1 — Core typechecker cross-check (1 day, ~250 LOC)

**Goal**: `match Option[T] { Err(_) => ... }` rejected at `ailang check` time with a clear error.

**Tasks**:

1. **Add new `TypeErrorKind`** (`internal/types/errors.go`):
   ```go
   MatchForeignConstructorError TypeErrorKind = "match_foreign_constructor"
   ```

2. **Extend `TypeCheckError` formatting** (`internal/types/errors.go`):
   - Handle the new error kind in `Error()` method
   - Produce the error message format specified in design doc (names both ADTs, suggests correct constructors)

3. **Add cross-check in `checkPattern`** (`internal/types/typechecker_patterns.go` ~line 149):
   - For `ConstructorPattern`: look up `tc.constructorTypes[p.Name]` to get the ADT the constructor belongs to
   - Look up the scrutinee's ADT via `extractScrutADTName(scrutType)` helper (new — handles `*TCon{Name}`, `*TApp{Constructor: *TCon{Name}}`, unifies polymorphic vars)
   - If both resolved AND they differ → emit `MatchForeignConstructorError`
   - If scrutinee ADT is still a fresh type var (not yet resolved by unification), defer the check (constraint-based path — see "Open question" below)

4. **Helper: `lookupADTConstructors(adtName) []string`**:
   - New method on `CoreTypeChecker` that reverses `constructorTypes` map to list constructors of a given ADT
   - Used in error message ("Option's constructors are: Some, None")

5. **Unit test** (`internal/types/typechecker_patterns_xcheck_test.go` — NEW):
   - `TestMatch_ForeignConstructor_OptionVsResult` — verifies `match Option { Err(_) => ... }` produces the new error
   - `TestMatch_ForeignConstructor_ResultVsOption` — symmetric
   - `TestMatch_ForeignConstructor_ListVsOption` — `match list[T] { Some(_) => ... }`
   - `TestMatch_ValidConstructor_PassesThrough` — `match Option { Some(_) => ..., None => ... }` compiles cleanly (regression)
   - `TestMatch_ForeignConstructor_ErrorMessage` — asserts the error message includes both ADT names + their constructor lists

**Open question** (to resolve during M1): when the scrutinee type is unresolved (e.g. `let x = something_polymorphic in match x { Some(_) => ... }`), do we eagerly check (could mis-fire on polymorphic code) or defer until unification settles? The original design says "if both resolved AND they differ" — i.e. defer. We'll go with this conservative approach in M1 and broaden in a future sprint if needed.

**Acceptance criteria**:
- [ ] `match Option { Err(_) => ... }` produces `MatchForeignConstructorError` with the specified message format
- [ ] `match Result { Some(_) => ... }` symmetric
- [ ] `match list { Some(_) => ... }` rejected
- [ ] `match Option { Some(_) => ..., None => ... }` still compiles
- [ ] `match _ { Some(_) => ..., None => ... }` (wildcard scrutinee — rare but possible) doesn't crash
- [ ] All 5 unit tests pass

---

### M2 — Regression coverage + edge cases (0.5 day, ~150 LOC)

**Goal**: existing AILANG code unaffected; corner cases handled correctly.

**Tasks**:

1. **Run full test suite** — `make test` must pass with zero regressions
2. **Run `ailang check` on every stdlib file** — `make verify-stdlib` or equivalent, must pass
3. **Run `ailang check` on every example** — `make verify-examples`, must pass
4. **Add edge-case tests** (`internal/types/typechecker_patterns_xcheck_test.go`):
   - Wildcard arm with foreign-constructor arms: `match Option { _ => "any", Err(_) => "bad" }` — the wildcard works but the Err arm still fails (single error, not silenced by wildcard)
   - Variable binding alongside foreign constructor: `match Option { x => f(x), Err(_) => "bad" }` — variable arm works, Err arm fails
   - Nested constructors: `match Option[Result[int, string]] { Some(Err(e)) => ..., None => ... }` — both Some and Err must reference the RIGHT ADTs at their respective positions
   - Polymorphic scrutinee: `func f[T](x: Option[T]) -> ... { match x { Err(_) => ... } }` — should still fail despite T being polymorphic
5. **Add expected-fail examples** under `examples/expected_fail/`:
   - `match_foreign_constructor_option.ail` — the exact bug shape that bit motoko_ext_compaction_ai
   - `match_foreign_constructor_result.ail` — symmetric

**Acceptance criteria**:
- [ ] `make test` passes (zero new failures)
- [ ] `make verify-stdlib` passes (all stdlib modules type-check)
- [ ] `make verify-examples` passes (all examples type-check OR are listed as expected-fail)
- [ ] 4+ edge-case tests added and passing
- [ ] 2+ expected-fail example files added (with `.expected_error.txt` companions if convention used)

---

### M3 — Docs + CHANGELOG + release artifacts (0.5 day, ~100 LOC)

**Goal**: ship-ready release artifacts.

**Tasks**:

1. **New doc**: `docs/docs/reference/option-vs-result.md` (~80 lines markdown):
   - "Option vs Result: which to use when"
   - Table of std/json + std/fs functions and which they return
   - Pattern-match examples for each
   - Cross-link from main syntax reference
2. **CHANGELOG entry** under `changelogs/v0.10-current.md`:
   - New `[v0.18.10]` section
   - M-MATCH-ADT-XCHECK summary + rationale + the bug it prevents
   - Acceptance verification breadcrumbs (tests, regression status)
3. **Move design doc** from `planned/v0_18_10/` to `implemented/v0_18_10/` (move script in release-manager skill handles this)
4. **Bump `std/VERSION`** to v0.18.10
5. **Update `docs/src/constants/version.js`** STABLE_RELEASE

**Acceptance criteria**:
- [ ] `docs/docs/reference/option-vs-result.md` exists and lints clean (docusaurus build doesn't break)
- [ ] CHANGELOG entry under `[v0.18.10] - 2026-05-XX`
- [ ] Design doc in `implemented/v0_18_10/`
- [ ] `std/VERSION` shows `v0.18.10`
- [ ] Website version constants updated

---

## Day-by-day breakdown

| Day | Milestones | Hours | Deliverable |
|---|---|---|---|
| 1 | M1: Core typechecker check + unit tests | 5-6h | typechecker_patterns.go + errors.go modified, 5 new unit tests pass |
| 2 (am) | M2: Regression + edge cases | 3h | Full test suite green, edge cases covered, expected-fail examples |
| 2 (pm) | M3: Docs + CHANGELOG + release prep | 2h | Ready to tag v0.18.10 |

---

## Success metrics

- **Bug class closed**: `match Option[T] { Err(_) => ... }` produces a clear compile-time error pointing at the wrong constructor and naming both ADTs.
- **Zero regression**: All 1000+ existing match expressions in stdlib + examples + tests still compile.
- **AI workflow benefit**: motoko_agent + similar AI agents can now self-correct this bug class inside their `ailang check` → fix → retry loop, without needing the downstream runtime probes to catch it.
- **Documentation**: First-time AILANG users have a canonical "Option vs Result" reference.

---

## Risks

| Risk | Mitigation |
|---|---|
| Polymorphic scrutinee types confuse the check (false-positive on `match (x: a) { Some(_) => ... }`) | Conservative: only check when BOTH the constructor's ADT and the scrutinee's ADT are concretely resolved. Polymorphic cases pass through (existing behavior). |
| Some stdlib code relies on the existing loose behavior (unlikely but possible) | Run `make test` + `make verify-stdlib` + `make verify-examples` in M2 to catch any regression early. If found: case-by-case decision (fix the stdlib code OR add an exception). |
| `constructorTypes` map missing entries for some ADTs (e.g. built-in `list`'s `::` and `[]` constructors) | M1 explicitly handles ListPattern separately (already in the switch). Constructor cross-check only fires on `ConstructorPattern` (named ctors). |
| Error message clarity for nested patterns | M2 includes a nested-constructors test — verify the position field correctly points at the OFFENDING inner constructor, not the outer match. |

---

## Dependencies

- None. This is a self-contained typechecker change.

---

## Files modified

| File | Change | Est LOC |
|---|---|---|
| `internal/types/errors.go` | New `MatchForeignConstructorError` kind + error formatting | ~40 |
| `internal/types/typechecker_patterns.go` | Cross-check in `checkPattern` for `ConstructorPattern` | ~50 |
| `internal/types/typechecker_core.go` | New helper `lookupADTConstructors` | ~15 |
| `internal/types/typechecker_patterns_xcheck_test.go` | NEW — 9-10 test cases | ~250 |
| `examples/expected_fail/match_foreign_constructor_option.ail` | NEW | ~15 |
| `examples/expected_fail/match_foreign_constructor_result.ail` | NEW | ~15 |
| `docs/docs/reference/option-vs-result.md` | NEW | ~80 |
| `changelogs/v0.10-current.md` | New v0.18.10 section | ~30 |
| `std/VERSION` | v0.18.9 → v0.18.10 | 1 |
| `docs/src/constants/version.js` | STABLE_RELEASE bump | 1 |
| **Total** | | **~500** |
