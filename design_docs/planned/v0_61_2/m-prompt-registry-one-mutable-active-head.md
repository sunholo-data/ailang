# Prompt Registry One-Mutable-Active-Head Reconciliation

**Status**: Planned
**Target**: v0.61.2
**Priority**: P0 (red REQUIRED `test` CI context — blocks every PR in the repo)
**Estimated**: 1 day
**Dependencies**: None (consumes the already-shipped freeze tooling from D-41 / `m-prompt-version-freeze-on-first-bank`)

**Issue**: [#2453](https://github.com/sunholo-data/ailang/issues/2453) (triage iteration 57)
**Failing commit**: `e3595b4f953b22863c284dca490c6f89277c083b` (origin/dev); regression introduced by `2da0c4592` (activated `v0.16.20` with `v0.16.19` still mutable)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Freezes the registry into a deterministic lifecycle state machine (mutable head → frozen superseded head); removes the ambiguous two-mutable-entry state that made `TestRealRegistry_PostMigrationSplitCounts`'s `id` value map-iteration-nondeterministic |
| A2: Replayability | +1 | A frozen `v0.16.19` with its enforced sha256 (`f435927d…`, verified byte-exact) means any future baseline pinned to `v0.16.19` refers to immutable, replayable bytes |
| A3: Effect Legibility | 0 | No effect-surface change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | Extends the local, repo-only `ailang prompt freeze --check` gate so the invariant is verifiable in CI without any network or corpus store |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | The invariant becomes machine-checkable (`make check-prompt-freeze` goes red) instead of discoverable only by reading the migration design doc |
| A8: Minimal Syntax | +1 | No new syntax; reuses `ailang prompt freeze --migrate` and the existing `FrozenMarker` schema unchanged |
| A9: Cost Visibility | 0 | No resource accounting change |
| A10: Composability | 0 | Reuses `migrateRegistries` / `marshalEntryPreserving` as-is |
| A11: Structured Failure | +1 | The new `--check` violation is a typed, named diagnostic naming the offending version, not a panic or a silent drift |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (removes one)
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`TestRealRegistry_PostMigrationSplitCounts` (`cmd/ailang/prompt_freeze_test.go:154`) fails on Linux and Windows in the REQUIRED `test` CI context (runs 45644929576, first failing Linux job from 45581519333) and in focused local runs, at origin/dev `e3595b4f`. The test asserts the **one-mutable-active-head invariant** on the real prompt registry:

```go
// The migrated corpus is the floor; subsequent prompt releases add frozen
// versions while keeping exactly the current active version mutable.
if b < 19 || l < 39 || m != 1 || id != source.Active || mirror.Active != source.Active {
```

**Current State (measured first-party at `e3595b4f`, both registries byte-identical, `cmp` clean):**

- `prompts/versions.json` and mirror `cmd/ailang/prompts/versions.json`: 75 entries, `active = "v0.16.20"`
- Split: **banked 19 / legacy 54 / mutable 2** — mutable = `["v0.16.19", "v0.16.20"]`
- The test requires `m == 1` → **2 ≠ 1 → FAIL**. With two mutable entries the returned `id` is also map-iteration-nondeterministic, so `id != source.Active` fails on some runs regardless.
- All other assertions pass: `b=19 ≥ 19`, `l=54 ≥ 39`, the two registries' `versions` maps marshal equal, `active` agrees, and `aver`'s recorded hash `1f5c9654…` matches `prompts/aver.md` byte-exact (sha256 verified).

**Root cause:** commit `2da0c4592` added and activated `v0.16.20` (created 2026-11-25) but did not freeze the superseded head `v0.16.19` (created 2026-11-24). Every previous activation froze its predecessor on the supersession date — measured: `v0.16.15` frozen `2026-11-12`, `v0.16.16` frozen `2026-11-23`, `v0.16.17` frozen `2026-11-24`, `v0.16.18` frozen `2026-11-24`, all `reason: "legacy"`, `evidence_count: 0`. `v0.16.19` is the first head in the series left mutable through an activation.

**Second, measured defect — the CI gate is blind to this class:** `ailang prompt freeze --check` exits **0 (green) at `e3595b4f`** — executed live in this session: `checked 75 registry entries`, no violations, rc=0. `checkRegistries` (`cmd/ailang/prompt_freeze_core.go:336`) reports only: (a) corpus-evidenced-but-not-frozen, (b) unenforceable/missing/mismatched hashes, (c) mirror-entry divergence, (d) mirror-only entries. **No violation class covers "a non-active version is still mutable."** `v0.16.19` has zero corpus citations (measured: 20 distinct `prompt_version` values across `eval_results/baselines`, top `v0.3.21` at 210; **no `v0.16.x` value appears**), so it trips nothing in `--check`. Only the Go test catches the invariant break, and only after the violating commit is already on origin/dev.

**Impact:**

- The REQUIRED `test` context is red on `origin/dev` → every PR in the repo fails its required checks until this lands (same failure class as D-41's `1a3104a49` incident, which blocked merges until iteration 283 repaired it).
- `ailang prompt` serves `v0.16.20`; a mutable `v0.16.19` can drift in place with no version bump and no gate firing, silently invalidating what the frozen-marker system promises about every non-active version.

## Goals

**Primary Goal:** Restore the one-mutable-active-head invariant on the real prompt registry — `v0.16.19` frozen, `v0.16.20` (active) the sole mutable entry — in both the source registry and the `cmd/ailang` mirror, with frozen prompt bytes untouched and freeze evidence published, and wire the invariant into the `--check` CI gate so the next activation cannot re-break it silently.

**Success Metrics:**

1. `TestRealRegistry_PostMigrationSplitCounts` green: split `19/55/1`, `id == "v0.16.20" == source.Active == mirror.Active` (deterministic with one mutable entry).
2. `git diff` of the fixing commit touches exactly two files — `prompts/versions.json` and `cmd/ailang/prompts/versions.json` — and the only content delta in each is the `v0.16.19` entry gaining its 6-line `frozen` block; all 74 other entries byte-preserved (`marshalEntryPreserving` returns raw bytes whenever frozen-state is unchanged).
3. Zero prompt-file bytes change: `sha256(prompts/v0.16.19.md) == f435927d97f7637bdca7f019c28c7da022bb114558e61e3fee007f64348ce06b` before and after (already matches the recorded hash — measured).
4. `ailang prompt freeze --check` goes RED on a registry with a superseded mutable head (new violation class), GREEN at the fixed HEAD; `make check-prompt-freeze` (already in `make ci`, `make/ci.mk:11`) inherits this coverage.
5. All existing freeze tests stay green with zero test-text edits (verified interaction: see Solution Design M2).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Freeze `v0.16.19` with `reason: "legacy"` (not `"banked"`), via `ailang prompt freeze --migrate` | Selects which of the tool's two freeze paths applies; `banked` is impossible here (0 citations) and the single-version path **refuses** unevidenced versions (`prompt_freeze_core.go:305`: "has no banked corpus evidence") | agent (mechanically forced: measured 0 citations + precedent of `v0.16.15`–`v0.16.18` all legacy/0) | design | low |
| D2: Reconcile D-41(c) ("mutable until first banked use") with freeze-on-supersession: D-41(c) governs the **active** head's edit rights; supersession (not first banked use) is the freeze trigger for a head that is no longer active | This is the only ruling-level tension in the fix; the shipped acceptance criterion (`m == 1`, test comment "keeping exactly the current active version mutable") already decides it | agent (the invariant is already ratified by the shipped test + landed migration design; recorded here so a reviewer can veto) | design | low |
| D3: Extend `checkRegistries` with a "superseded mutable head" violation class, and exclude corpus-evidenced entries from it | Makes the CI gate (`make check-prompt-freeze`) enforce the invariant; the exclusion is load-bearing — without it the existing `TestPromptFreezeCheck_RedOnMissingMarker` sees 2 violations where it asserts exactly 1 | agent | design | med (CI-gate semantics + existing-test interaction) |
| D4: The `frozen.at` value is the date of the freeze run (`time.Now().UTC()` via `migrateRegistries`), i.e. the fixing commit's date, not the version's `created` date | Publishes honest freeze evidence; consistent with `v0.16.17`/`v0.16.18` whose `at` equals their supersession date, not creation date | agent (tool default) | design | low |
| D5: Scope = the teaching-prompt registry pair only; `prompts/agent/versions.json` (3 mutable) and `prompts/devtools/versions.json` (4 mutable) are NOT migrated here | The failing test and `promptRegistryPair` (`prompt_freeze_core.go:234`) hardcode the syntax registry; the agent/devtools series have never been migrated (zero frozen markers in either) and need their own corpus/decision work | agent (scoping), human may override to widen | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — resolved by measurement: `v0.16.19` has 0 corpus citations (see Verification Log V4); `--migrate` is the only path that freezes it, and it does so as `legacy`.
- [x] D2 — resolved by the shipped acceptance criterion; recorded for reviewer veto, not open.
- [x] D3 — resolved by reading `checkRegistries` (V6) and the existing test bodies (V7); the exclusion keeps every existing assertion green.
- [x] D4 — tool default; no code change involved.
- [x] D5 — scoped to the registry pair the failing test names; the other kinds are Future Work.

(All decisions are resolvable within this design; none require a human ruling to start the sprint. A reviewer who rejects D2's reconciliation should say so at quorum — it is the one judgment call.)

## Solution Design

### Overview

Two moves, one commit each:

- **M1 — Reconcile the registries (data fix, no code):** run `ailang prompt freeze --migrate` at the repo root. `migrateRegistries` (`prompt_freeze_core.go:240`) walks every unfrozen, non-active version, stamps `FrozenMarker{At: <run date>, Reason: "legacy", EvidenceCount: 0, EvidenceExample: ""}`, and writes **both** `prompts/versions.json` and `cmd/ailang/prompts/versions.json` through the same ordered writer. Expected output: `migrated prompt registry: 19 banked, 55 legacy, 1 mutable`.
- **M2 — Close the gate blindness (small code fix):** add one violation class to `checkRegistries` so `--check` reports a superseded mutable head, and pin it with a test.

### Architecture

**Components:**

1. **`migrateRegistries`** — unchanged. It already: validates every entry's hash is enforceable 64-hex sha256 and matches file bytes (refuses otherwise — `TestPromptFreezeMigrate_RefusesStaleHash` pins this), skips the active version, freezes the rest as `legacy` when unevidenced, and writes both registries identically (`TestPromptFreezeMigrate_WritesBothRegistries` pins the byte-equality).
2. **`marshalEntryPreserving`** (`prompt_freeze_core.go:119`) — unchanged. For entries whose frozen-state is unchanged it returns the **raw JSON bytes verbatim**, so the other 74 entries are byte-preserved; only `v0.16.19`'s raw entry gains a `"frozen": {…}` block appended at its original indentation.
3. **`checkRegistries`** — one new class. Today (`prompt_freeze_core.go:336-371`) the "not frozen but should be" family covers only corpus-evidenced entries. Add, in the same per-entry loop over `source.VersionKeys`:

```go
if e.Frozen == nil && id != source.Active && evidence[id].Count == 0 {
    v = append(v, fmt.Sprintf(
        "mutable version %s: superseded head not frozen — freeze it before the next release (ailang prompt freeze --migrate)", id))
}
```

   The `evidence[id].Count == 0` guard is what keeps the classes disjoint: a corpus-evidenced mutable entry already gets the existing `"corpus-evidenced but not frozen: %s (%d citations, e.g. %s)"` violation, and would otherwise be reported twice. The active entry is exempt by the invariant itself (exactly the active version is mutable).

4. **Frozen marker schema** — unchanged (`internal/eval_harness/prompt_frozen.go:12`, mirrored in `internal/prompt/loader.go:82`): `at` / `reason` / `evidence_count` / `evidence_example`.

### Implementation Plan

**Phase 1: Reconcile (M1)** (~1 hour)
- [ ] Run `ailang prompt freeze --migrate` at the repo root.
- [ ] Verify `git diff --stat` shows exactly `prompts/versions.json` + `cmd/ailang/prompts/versions.json`; inspect the diff: only the `v0.16.19` entry gains a 6-line `frozen` block, byte-identical between the two files.
- [ ] Verify `cmp prompts/versions.json cmd/ailang/prompts/versions.json` clean and `sha256sum prompts/v0.16.19.md` still `f435927d…`.
- [ ] Focused test green: `go test ./cmd/ailang/ -run 'TestRealRegistry_PostMigrationSplitCounts|TestPromptFreeze' -count=1`.

**Phase 2: Gate extension (M2)** (~4 hours)
- [ ] Add the violation class to `checkRegistries` as specified above.
- [ ] Add `TestPromptFreezeCheck_RedOnSupersededMutableHead` to `prompt_freeze_test.go`: from `freezeFixture` + `migrateRegistries`, **un-freeze an unevidenced non-active version in both registries** (e.g. `l1` — the stubbed corpus scanner only evidences `b1`/`b2`/`b3`), then assert `checkRegistries` returns exactly 1 violation naming `l1` and containing `superseded head not frozen`.
- [ ] Re-run the whole `prompt_freeze*` test set — all 24 existing tests, **zero text edits** (the `Count == 0` exclusion is what keeps `TestPromptFreezeCheck_RedOnMissingMarker`'s `len(v) != 1` assertion green: `b1` is evidenced, so it yields exactly its one existing violation).
- [ ] `ailang prompt freeze --check` green at fixed HEAD; re-verify red-arm by hand on a scratch clone with `v0.16.19` unfrozen.

**Phase 3: Evidence + docs (~1 hour)**
- [ ] PR body carries the measured before/after split (`19/54/2` → `19/55/1`), the two CI run links from the issue, and the `--check` red-arm output.
- [ ] Note the operational rule where the prompt-release flow is documented: activating a new teaching-prompt version requires freezing the superseded head **in the same commit** — now enforced by the gate rather than by memory.

### Files to Modify/Create

**New files:** none.

**Modified files:**

- `prompts/versions.json` — `v0.16.19` entry gains 6 lines (`frozen` block). Data only.
- `cmd/ailang/prompts/versions.json` — identical 6 lines. Data only.
- `cmd/ailang/prompt_freeze_core.go` — +4 LOC in `checkRegistries` (violation class). No signature changes.
- `cmd/ailang/prompt_freeze_test.go` — +15 LOC (one new test).

No `internal/` changes. No error-code allocations (all messages are free-form `--check` violation strings, and no `MODxxx`/`PARxxx` code is introduced).

## Examples

### Example 1: The registry delta (M1)

**Before** (`prompts/versions.json`, and identically in the mirror):

```json
    "v0.16.19": {
      "file": "prompts/v0.16.19.md",
      "hash": "f435927d97f7637bdca7f019c28c7da022bb114558e61e3fee007f64348ce06b",
      "description": "Document let (a, b) tuple destructuring, module-qualified types (S.Set[T], M.Map[K, V]), newline block delimiters, and std/list takeWhile/dropWhile",
      "created": "2026-11-24",
      "tags": ["production", "latest"],
      "notes": "Document let (a, b) tuple destructuring, module-qualified types (S.Set[T], M.Map[K, V]), newline block delimiters, and std/list takeWhile/dropWhile"
    },
```

**After:**

```json
    "v0.16.19": {
      ...same fields, byte-identical...,
      "notes": "Document let (a, b) tuple destructuring, module-qualified types (S.Set[T], M.Map[K, V]), newline block delimiters, and std/list takeWhile/dropWhile",
      "frozen": {
        "at": "<freeze-run date, tool default: today UTC>",
        "reason": "legacy",
        "evidence_count": 0,
        "evidence_example": ""
      }
    },
```

`prompts/v0.16.19.md` itself: **no change** — its recorded hash already matches its bytes (measured), so the freeze is honest evidence, not a paper trail over drifted content.

### Example 2: The gate going red on a recurrence (M2)

Next release, suppose `v0.16.21` is added and activated, again without freezing `v0.16.20`:

```
$ make check-prompt-freeze
mutable version v0.16.20: superseded head not frozen — freeze it before the next release (ailang prompt freeze --migrate)
checked 76 registry entries
make: *** [make/code-health.mk:236: check-prompt-freeze] Error 1
```

Today that same tree yields `checked 75 registry entries`, rc=0 — measured live in this session. After M2, the PR cannot merge: `check-prompt-freeze` is in the `ci` target (`make/ci.mk:11`).

## Success Criteria

- [ ] `TestRealRegistry_PostMigrationSplitCounts` green with split `19/55/1`, `id == source.Active == mirror.Active == "v0.16.20"` (acceptance: the one-mutable-active-head invariant).
- [ ] The fixing commit's diff touches exactly `prompts/versions.json` + `cmd/ailang/prompts/versions.json` (M1) — frozen prompt bytes preserved, `v0.16.19` hash still `f435927d…` and still matching the file.
- [ ] `ailang prompt freeze --check` red on a synthetic superseded-mutable-head registry (new test), green at fixed HEAD.
- [ ] All 24 existing `prompt_freeze*` tests green with zero test-text edits.
- [ ] `make check-prompt-freeze` green in CI on both Linux and Windows (the failing test and the gate are content-deterministic; no platform dependence — all paths go through `filepath.Join` and the writer emits `\n`).
- [ ] All tests passing (`go test ./cmd/ailang/ ./internal/eval_harness/ ./internal/prompt/`).
- [ ] Documentation updated: the operational same-commit freeze rule noted in the PR/issue thread (#2453) with the measured evidence.

## Testing Strategy

**Unit tests:**
- `TestPromptFreezeCheck_RedOnSupersededMutableHead` (new): fixture + migrate, un-freeze an **unevidenced** non-active entry (`l1`) in both registries, expect exactly 1 violation naming `l1`.
- Keep `TestPromptFreezeCheck_GreenOnMigratedFixture` as the green control — unchanged, still passes (the active entry is exempt by design).

**Regression-surface tests (existing tests that MUST stay green, all read in full):**
- `TestPromptFreezeCheck_RedOnMissingMarker` (`prompt_freeze_test.go:141`): asserts `len(v) == 1` + `"corpus-evidenced but not frozen"` after un-freezing `b1` (evidenced) — the `Count == 0` exclusion in M2 is what preserves this; if the exclusion is dropped this test goes red and the design is wrong, not the test.
- `TestPromptFreezeMigrate_SplitCounts` / `_WritesBothRegistries` / `_Idempotent` — pin the writer behavior M1 relies on.
- `TestFreezeCheck_*` (`prompt_freeze_check_git_test.go:17-289`, 19 tests) — merge-base immutability, mirror divergence classes, checked-count line: M2 adds no new error paths and does not touch `checked` counting.

**Manual testing:**
- Red-arm on a scratch clone: `git revert` the registry fix → `--check` must exit 1 naming `v0.16.19`.
- `ailang prompt --version`-style smoke: active prompt still resolves to `v0.16.20` bytes; `ailang prompt` unaffected (registry `active` unchanged).

## Verification Log

Every load-bearing claim in this doc, with its instrument:

| # | Claim | Instrument / Result |
|---|-------|---------------------|
| V1 | Current split is `19/54/2`, active `v0.16.20`, mutable = `v0.16.19`+`v0.16.20`, 75 entries | Python enumeration over `prompts/versions.json` at `e3595b4f` (this clone, clean tree): banked 19, legacy 54, mutable `[v0.16.19, v0.16.20]` |
| V2 | Source and mirror registries agree today | `cmp prompts/versions.json cmd/ailang/prompts/versions.json` clean; also `cmp` clean on `v0.16.19.md` and `v0.16.20.md` mirror copies. The disagreement is with the **invariant**, not between the files |
| V3 | Test assertion set (what exactly fails) | Read `cmd/ailang/prompt_freeze_test.go:154-177`: `b<19 \|\| l<39 \|\| m!=1 \|\| id!=source.Active \|\| mirror.Active!=source.Active`, registries-equal marshal, `aver` hash. Only `m != 1` (and consequentially `id`) fails. Deterministic content failure → matches the observed Linux **and** Windows red |
| V4 | `v0.16.19` has **zero** banked corpus evidence | `grep -h '"prompt_version"' eval_results/baselines` (recursive, `--include='*.json'`): 20 distinct values, top `v0.3.21`×210; **no `v0.16.x`**. Therefore `ailang prompt freeze v0.16.19` refuses at `prompt_freeze_core.go:305` ("has no banked corpus evidence") and `--migrate` stamps `reason: "legacy"`, `evidence_count: 0` — the same marker shape as `v0.16.15`–`v0.16.18` (read from the registry) |
| V5 | `v0.16.19`'s recorded hash is enforceable and byte-true | `sha256sum prompts/v0.16.19.md` = `f435927d97f7637bdca7f019c28c7da022bb114558e61e3fee007f64348ce06b` = recorded hash in both registries. Same check green for `v0.16.20` (`466da96c…`) and `aver` (`1f5c9654…`) |
| V6 | `--check` is green at `e3595b4f` (gate blindness) | **Executed live**: `ailang prompt freeze --check` → `checked 75 registry entries`, rc=0. Read `checkRegistries` (`prompt_freeze_core.go:336`): no violation class covers a non-active mutable entry |
| V7 | M2 does not break `TestPromptFreezeCheck_RedOnMissingMarker` | Read the test body (`:141-152`): un-freezes `b1`, asserts exactly 1 violation + `"corpus-evidenced but not frozen"`. `b1` is evidenced in the fixture scanner (`prompt_freeze_test.go:26`), so the M2 class (guarded by `Count == 0`) does not fire for it. Disjoint by construction |
| V8 | M1 byte-preserves the other 74 entries | Read `marshalEntryPreserving` (`prompt_freeze_core.go:119-146`): returns raw entry bytes verbatim whenever the frozen-state is unchanged; appends the marker only for the newly-frozen entry. Writer determinism pinned by `TestPromptFreezeMigrate_Idempotent` |
| V9 | `check-prompt-freeze` is a CI gate on every PR | `make/code-health.mk:236-237` → `go run ./cmd/ailang prompt freeze --check`; listed in the `ci` target at `make/ci.mk:11` |
| V10 | The invariant is pre-ratified, not new policy | Test comment at `:164-165` ("subsequent prompt releases add frozen versions while keeping exactly the current active version mutable") + landed migration work (D-41 ruling (c), `design_docs/v1-mission.md` row 110; PRs #936/#937/#949, `design_docs/implemented/v0_35_0/m-prompt-version-freeze-on-first-bank-sprint-plan.md`) |
| V11 | Negative existence: no other unfrozen non-active entry hides in the registry | V1's enumeration is exhaustive over the 75-entry map — mutable set is exactly `{v0.16.19, v0.16.20}` |
| V12 | Negative existence: no new error code is allocated | No `MODxxx`/`PARxxx`/`TCxxx` code appears in the design; all new strings are free-form `--check` violations, which share no namespace (verified against the existing violation strings at `prompt_freeze_core.go:340-370`) |
| V13 | `agent`/`devtools` registries are out of scope but measured | `prompts/agent/versions.json`: active `v0.9.1`, mutable `[v0.8.2, v0.9.0, v0.9.1]`; `prompts/devtools/versions.json`: active `v0.8.1`, mutable `[v0.8.0, v0.8.0-compact, v0.8.1, v0.8.1-compact]`. Zero frozen markers in either — these series were never migrated; `promptRegistryPair` (`prompt_freeze_core.go:234`) hardcodes the syntax registry. Future Work, not this sprint |
| V14 | `"latest"` is a request sentinel, not a registry-tag constraint | `internal/prompt/loader.go:53`: `""` or `"latest"` resolves to the **active** version; the `tags: ["production","latest"]` on `v0.16.19` is descriptive metadata only. Leaving it untouched keeps M1's diff minimal |
| V15 | The Go test was NOT re-executed in this session (no Go toolchain in this environment) | The failing/fixed assertions are derived by code reading + data measurement (V1-V3), consistent with the reported CI red on both platforms. The sprint executor re-runs the focused test as Phase 1's exit check |

**Deliberately NOT executed:** `ailang prompt freeze --migrate` and `ailang prompt freeze v0.16.19` — both **write** the registries, and this is a design-only intake; executing them here would be implementing the fix. The migration was validated by reading `migrateRegistries` + the migrate test bodies instead (V4, V8).

## Deferred Decisions

The following are intentionally left open for the implementer:

- **`frozen.at` concrete date** — the tool stamps the UTC run date; the executor runs the migration, so the date falls out of execution. *Agent may choose* (no code change).
- **Exact violation message wording** — the proposed string is advisory; only the class (unevidenced, non-active, unfrozen) and single-report disjointness are fixed. *Agent may choose.*
- **Whether to drop the now-stale `latest` tag from `v0.16.19`** — tags are descriptive (V14) and no gate reads them; default is leave untouched. *Agent may choose; default = leave.*
- **Where the operational same-commit-freeze rule is durably documented** (issue thread vs. release checklist vs. prompt-manager skill) — *agent may choose*, provided it is written down somewhere a prompt release will see it.

## Non-Goals

**Not attempted in this feature:**

- **Migrating the `agent`/`devtools` prompt series** — measured 3 and 4 mutable entries respectively (V13), never migrated, different corpus semantics; needs its own design doc and evidence pass. Pulling them in here would couple a P0 unblock to unsettled policy.
- **Any change to `ailang prompt freeze --migrate`/`freezeVersion` semantics** — the tooling is correct; the defect was a skipped step in a release process plus a blind gate. Fix the data, fix the gate, don't touch the writer.
- **Backfilling `banked` evidence for `v0.16.x`** — there is none to backfill (V4); `legacy` is the honest marker.
- **Editing any prompt `.md` bytes** — explicitly forbidden by this design; the invariant exists to protect those bytes.
- **Parser/typechecker changes** — none; Conflict Surface section not required (no `internal/parser|types|eval|vm|effects` surface touched).

## Timeline

**Week 1** (1 day):
- Phase 1: registry reconciliation + diff verification (~1 h)
- Phase 2: gate extension + new test + full freeze-test re-run (~4 h)
- Phase 3: PR evidence, docs note, CI green on both platforms (~1 h + CI latency)

**Total: ~1 day** (estimate 2×'d from a 4-hour raw guess, per house convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| M2's new violation class double-reports an evidenced mutable entry, breaking `TestPromptFreezeCheck_RedOnMissingMarker`'s exact-count assertion | Med | The `evidence[id].Count == 0` guard makes the classes disjoint by construction (V7); the test re-run in Phase 2 is the proof |
| A future release "fixes" the gate by unfreezing instead of freezing | Med | The merge-base immutability loop (`checkGitPromptFreezeInvariants`, `prompt_freeze_check_git.go:15`) makes unfreezing a landed frozen entry a registry-fields violation — the escape hatch is already closed |
| `--migrate` stamps today's date, and the release lands later (PR queue) | Low | `at` is the freeze-run date, which is the honest publish date; no test reads it back except through marker equality (idempotence test uses whole-marker comparison and is date-stable because the marker is only written when `Frozen == nil`) |
| Windows line-ending drift rewrites the whole registry file | Low | Writer emits `\n` and Go `os.WriteFile` performs no translation; the current failure is content-identical on both platforms, and the observed pre-fix state has both registries byte-identical across platforms in CI |

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_35_0/m-prompt-version-freeze-on-first-bank-sprint-plan.md` — the landed freeze-on-first-bank migration this registry floor came from (PRs #936/#937, iter-291)
- `design_docs/v1-mission.md` row **D-41** (RESOLVED, ruling (c)) — the mutable-until-first-banked-use lifecycle; this doc's D2 records the supersession reconciliation

**Planned (check for overlap):**
- None — the doc-search (SimHash + neural) over planned/ returned no prompt-freeze topic (top matches were unrelated: `m-motoko-ext-per-task-sprint-plan.md`, `m-d4-design-doc-driven-development.md`); no duplicate gate triggered

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Issue [#2453](https://github.com/sunholo-data/ailang/issues/2453) — triage evidence, CI run links (45644929576; first failing Linux job from 45581519333)
- `cmd/ailang/prompt_freeze_test.go:154` — the acceptance criterion
- `cmd/ailang/prompt_freeze_core.go` — `migrateRegistries` (:240), `freezeVersion` (:297), `checkRegistries` (:336)
- `internal/eval_harness/prompt_frozen.go` — `FrozenMarker` schema (D-41c)

## Future Work

- **Migrate the `agent` and `devtools` prompt series** onto the frozen/mutable lifecycle (V13) — separate design doc; must first decide what corpus (if any) evidences those series, and whether `promptRegistryPair` generalizes per `Kind` (`internal/prompt/loader.go`).
- **`ailang prompt release <new-version>`** — a single command that adds the entry, flips `active`, **and freezes the superseded head atomically**, making the invariant structurally unbreakable rather than gate-enforced. Deferred: the gate (M2) already catches the failure at PR time; the command is ergonomics.
- **Surface `frozen` state in `ailang prompt list`-style output** so humans can see the lifecycle without reading JSON.

---

**Document created**: 2026-11-25
**Last updated**: 2026-11-25