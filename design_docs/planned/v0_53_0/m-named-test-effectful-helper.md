# Named Test Effectful Helper Honesty (strip policy + truthful diagnostics)

**Status**: Planned
**Target**: v0.53.0
**Priority**: P1 (Medium)
**Estimated**: 4 days
**Dependencies**: None (builds on the named-test batch of `m-test-runner-compile-once.md`, same directory, whose D1 fallback it reuses unchanged)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Named-test entries stay `pure func`; the repair removes a source-corruption step (deleting declarations) from the compile path, and the seeded property entries keep their purity contract |
| A2: Replayability | 0 | No change to seeds, replay, or traces |
| A3: Effect Legibility | +1 | An effectful test body now reports the actual effect row ("Missing effects: FS") and the purity contract instead of a false "undefined variable" / "add `import std/debug (check)`" |
| A4: Explicit Authority | +1 | The wrong-suggestion class ("add `import std/debug (check)`" for a local function) disappears because the local name resolves; capability gating (`--caps FS`) is untouched and stays the documented route for effectful verification |
| A5: Bounded Verification | +1 | The strip's blast radius shrinks: only test/property blocks are removed from the base source, so per-test compiles see the module the user actually wrote |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | An agent reading the new error gets cause, contract, and a verified workaround in one message; today it gets a name-match lie that no edit sequence can satisfy |
| A8: Minimal Syntax | +1 | No new syntax; one misused surface (source rewriting) is removed |
| A9: Cost Visibility | 0 | Effect costs unchanged; declarations are bound, not called |
| A10: Composability | +1 | Batch and per-body paths, evaluator and VM, tests and properties share one base-source policy instead of four strip variants |
| A11: Structured Failure | +1 | Failures name the user's file, line, and construct instead of synthetic `$tmp1` / `__namedtest_0` symbols |
| A12: System Boundary | 0 | No boundary crossings added or removed |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced (entries stay pure; kept effectful declarations are closures, never invoked by the harness)
- [x] A3 (Effects): No hidden side effects (nothing new executes; the *diagnosis* of effects becomes honest)
- [x] A4 (Authority): No ambient access granted (no capability surface is added to `ailang test` — that is Future Work)
- [x] A7 (Machines First): The change exists to make machine-facing errors truthful, not for human convenience

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

A named test that calls a module-level helper fails with "undefined variable: check — add `import std/debug (check)`" on BOTH engines whenever the helper is not *syntactically* marked `pure` or `! {}` — even though the helper exists in the user's file, even when it is exported, and even when it is perfectly legal. The reporter's repro (verified below):

```ail
module repro
import std/fs (readFile)
func check() -> bool ! {FS} = readFile("fixture.txt") != ""
test "effectful helper" { check() }
```

```
pipeline error: type error in .../ailang-namedtest-*/repro (decl 0):
undefined variable: check at .../repro.ail:4:3 — add `import std/debug (check)`
```

**Current State (all rows verified live on v0.52.5, `ailang test` and `ailang test --bytecode`, 2026-10-08; see Verification Log):**

| # | Construct | `ailang check` / `ailang run` | `ailang test` today | Verdict |
|---|---|---|---|---|
| 1 | `pure func check() -> bool = ...` or `func check() -> bool ! {} = ...` helper | legal | passes | fine today, fine after |
| 2 | **unannotated** helper with a pure body (`func check() -> bool = "hello" != ""`) | legal | **fails**: "undefined variable: check … add `import std/debug (check)`" | **false positive — a legal module's tests cannot run** |
| 3 | helper with a declared row (`func check() -> bool ! {FS} = ...`), exported or not | legal | **fails**: same "undefined variable" + import lie | must fail *honestly*: purity contract |
| 4 | direct effectful stdlib call in the body (`test "x" { readFile(...) != "" }`) | legal inside a `! {FS}` func | **fails**: "Effect checking failed for function `'__namedtest_0'`" (batch) / `'$tmp1'` (per-body), with "Suggested fix: func `$tmp1`(...) -> T ! {FS}" — a fix the user cannot apply to a synthetic binder | must fail honestly, naming the contract |
| 5 | unannotated helper with an effectful body | **illegal** (`ailang check` demands `! {FS}`) | "undefined variable" + import lie — masks the module's real error | must surface the module's real error |

**Root cause (one mechanism, all five rows):** the named-test path does not compile the user's module. `Executor.prepareNamedTests` (`internal/testing/named_batch.go:109`) and the per-body fallback (`internal/testing/executor.go:194`) build the base source with `stripWithLineMap(src, file, nil)` / `stripNonPureFunctions(src, file)`, and `functionSkipRanges` (`internal/testing/source_strip.go:46`) deletes **every** top-level function that is not *syntactically* `pure` or `! {}` (`isEffectivelyPure`, `internal/testing/source_strip.go:71` — `f.IsPure || (f.Effects != nil && len(f.Effects) == 0)`; an unannotated func has `Effects == nil`). The batched entries are then appended as `pure func __namedtest_<k>() { <body> }` (`internal/testing/entrySource.add`, `internal/testing/named_batch.go:307`) and compiled in a private temp dir (D1 of `m-test-runner-compile-once.md` falls back to one compile per body on batch failure). So the helper is *deleted from the compile* before the type checker ever sees the call: the checker's report is true of the temp file and false of the user's file.

The misleading hint is a symptom, not a second bug to patch: `importHint` (`internal/types/import_hint.go:16`) fires because `check` happens to be a std/debug export (`std/debug.ail:37`). Once the helper is kept in the base source the name resolves, no "undefined variable" is raised, and the hint — and its wrong advice — cannot occur for this class. (Genuinely undefined names keep the hint; there it is correct.)

**Why the strip exists at all:** history. The strip predates batching (it is the per-body path's base builder) and was repaired in v0.31.0 to delete declarations *whole* (`m-strip-contract-awareness-sprint-plan.md`) because the pre-fix line-based version corrupted files. That sprint explicitly recorded this exact defect as out of scope: *"A named test that calls a genuinely effectful function still gets `undefined variable` … a real defect with a verified repro … Deliverable: file it as a new issue during landing."* This design doc is that filing, seven versions later.

**Purity of named-test bodies is a deliberate contract, not an accident:** `m-named-test-blocks.md` (implemented v0.29.0) pins it in its axiom table — "Test bodies are pure expressions evaluated deterministically" (A1), "Named test bodies are checked pure — pins the discipline" (A3) — and in Non-Goals: "bodies stay pure bool expressions." The property machinery (seeded generation, replay) sits on the same `pure func` entries. So the defect is **not** that effectful tests fail — they must fail — but that the *enforcement* is broken: the strip deletes the very declarations the checker needs to produce an honest failure, and the failure it produces instead names a synthetic symbol and prescribes an import that cannot fix anything.

**Impact:**
- **Agent-driven development (primary audience):** an agent writing a named test against a module helper gets an error that (a) denies the existence of a function in the same file, (b) prescribes `import std/debug (check)` — which, applied, does not compile the test either (the local helper still vanishes). Every model that trusts the message is sent down a dead end; the reporter's own workaround (stapledons-godot iteration 21, M4.5s) was to route the FS-reading test through an exported entry with `ailang run --caps FS` and keep strict tests on byte mirrors — i.e. agents already pay the discovery cost per project.
- **Determinism of the harness itself (class 2):** the strip makes `ailang test` results depend on *annotation style*, not semantics: `func check() -> bool = ...` and `func check() -> bool ! {} = ...` are the same function to the compiler, but one runs and one cannot.

## Goals

**Primary Goal:** Make `ailang test`'s named-test compile see the module the user wrote (keep every top-level function in the base source) so that legal modules test green and effectful bodies fail with the actual purity-contract error — no synthetic names, no import lies.

**Success Metrics:**
1. Class 2 (unannotated pure helper) goes from **fail** to **pass** on both engines; classes 3–5 go from "undefined variable + import lie" to an effect-row failure naming the test, the contract, the missing effects, and the verified workaround.
2. `grep`-able invariant: no `ailang test` failure message ever suggests importing a name that is declared in the same file.
3. Zero behavior change for classes 1 and all currently passing fixtures (`make test-core`, engine-parity suite).
4. The D1 fallback notice stays truthful (a batch failure caused by a user's effectful body is never reported as a harness bug).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Keep ALL top-level functions in the named-test base source (strip only test/property blocks); do NOT special-case "functions referenced by test bodies" | Determines whether the type checker enforces purity at all; a keep-referenced-only variant would need transitive call-graph analysis and would keep the "undefined variable" class alive for unreferenced broken helpers | agent (this doc), ratified by design review | design | med |
| Named-test entries stay `pure func` — the purity contract from `m-named-test-blocks.md` is NOT relaxed in this feature | Changing it would drag in a caps surface for `ailang test`, determinism/replay questions for the seeded property path, and a language-semantics decision | human (v0.29.0 ruling stands; re-opening it is explicitly Future Work) | design | high |
| Diagnose effectful bodies by MAPPING the entry's effect-check failure (batch + per-body), not by pre-scanning AST free variables | Pre-scan duplicates effect inference (declared rows AND inferred rows AND direct calls) and drifts from the checker; post-map has one source of truth and covers class 5 (illegal helper) for free | agent | design | low |
| The import-hint wrongness is dissolved by the strip fix, NOT patched inside `internal/types` | Patching `importHint` for names that happen to match local functions would require the checker to know about the user's file it never sees (it compiles the temp source); keeping the helper makes the question vanish | agent | design | low |
| `stripNonPureFunctions` (the non-pure policy) stays for the binding-extraction call site (`internal/testing/executor.go:440`), only the named-test/property base stops using it | That site has its own documented module-file/module-less routing; silently changing it would alter inline-test binding extraction | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Keep-all-funcs policy for the named-test and property base source (vs. referenced-only keep-set)
- [x] Entries remain `pure func`; effectful named tests are NOT supported in this feature
- [x] Post-map of effect-check failures, not AST pre-scan
- [x] `internal/types` hint machinery is not modified
- [x] `executor.go:440` binding path keeps today's strip policy

## Solution Design

### Overview

Two changes, one policy and one diagnostic, both local to `internal/testing`:

1. **M1 — the base source keeps every module-level function.** The batch (`prepareNamedTests`, `internal/testing/named_batch.go:109` — one strip serves both the test entries and the property entries of the batch), the per-property fallback (`forallCaller`, `internal/testing/property_batch.go:66`), and the per-body evaluator path (`internal/testing/executor.go:194`) stop building their base with the non-pure strip. They build it with a new `stripTestBlocks` that removes only test/property declarations (today's `testAndPropertySkipRanges`, unchanged) and returns the same line map. Effect: class 2 compiles and passes; classes 3–5 reach the effect checker with the helper present.

2. **M2 — honest diagnostics for the effectful-body class.** The harness already maps batched positions to the user's file (`mapPositions`, `named_batch.go:219`). Add a sibling mapper for the *class* of error: when an entry compile fails effect checking — `Effect checking failed for function '__namedtest_<k>'` (batch entries), `'__namedtest_entry'` (per-body VM wrap), or `'$tmp<N>'` (per-body evaluator wrap) — the per-test error is rewritten to name the contract, the missing effects, and the verified workaround, preserving the original text as detail:

```
named test "effectful helper" (repro.ail:4:1): test bodies are checked pure
(m-named-test-blocks.md, v0.29.0), but this body requires effects not allowed
in a pure signature:
  Missing effects: FS

Effectful verification belongs in an exported entry run with
`ailang run --caps FS`, or in a pure test over byte-mirror fixtures.
```

### Architecture

**Components:**
1. **`stripTestBlocks` (new, `internal/testing/source_strip.go`, ~25 LOC):** returns `(string, []int)` — the source minus test/property declaration line ranges, plus the kept-line map. Implemented as today's `stripWithLineMap` with an empty function-skip set; `functionSkipRanges` is not called. `stripNonPureFunctions` and `isEffectivelyPure` remain for the binding site.
2. **Base-source call sites (3 sites, 3 files, ~15 LOC total):** `named_batch.go:109` (batch: test + property entries), `property_batch.go:66` (`forallCaller`), `executor.go:194` (per-body evaluator path) switch to `stripTestBlocks`. The reserved-`__namedtest_` guard and the line-mapping machinery (`lineMap`, `ent.first/last/testLine`) are untouched — the map is still built from whatever is kept.
3. **Effect-contract mapper (~45 LOC + table):** a `mapEntryEffectError(msg, ent)` called from the existing error-mapping points (`mapBatchPositions` in the batch path; the per-body error returns in `executor.go` and `bytecode_engine.go`). It keys on the three synthetic-symbol shapes and keeps `formatEffectError`'s "Missing effects:" lines verbatim. It must not fire for non-synthetic effect errors (a user's own `main` mis-annotation inside the kept base is that user's real error and maps through `lineMap` like any module line).

### Implementation Plan

**Phase 1: Strip policy** (~1 day)
- [ ] Add `stripTestBlocks` with line map; unit tests beside `source_strip_test.go`'s existing table (its `strip: true` expectations for `named_test_effectful.ail` move to the new policy's test: `shout` now present, test block absent)
- [ ] Switch `named_batch.go:109` (batch: test + property entries), `property_batch.go:66` (`forallCaller`), `executor.go:194` (per-body evaluator)
- [ ] Fixture `internal/testing/testdata/strip/named_test_unannotated_helper.ail` (class 2): `ailang test` must pass (1 passed, 0 failed)

**Phase 2: Honest diagnostics** (~1.5 days)
- [ ] `mapEntryEffectError` + wire into batch and per-body paths (both engines)
- [ ] Fixtures: `named_test_effectful_helper.ail` (class 3, the reporter's repro), `named_test_effectful_direct.ail` (class 4), `named_test_inferred_effectful.ail` (class 5) — each asserts the contract message, the missing-effects row, and the `--caps FS` workaround text; none asserts on synthetic symbols or temp paths
- [ ] Engine-parity: the effectful fixtures produce the same report under `--bytecode`, `--strict-bytecode` (strict fails the test, not the batch), and evaluator

**Phase 3: Documentation & regression** (~1.5 days)
- [ ] User-facing note: named-test bodies are pure by contract; effectful verification goes in exported entries run with `ailang run --caps FS` (canonical limitations page and/or the testing reference — exact page per sprint-planner)
- [ ] `make test-core`, `make check-boundaries`, engine-parity suite green; changelog entry
- [ ] Sweep `docs/` and prompts for any claim that named tests support effects (none found — see Verification Log V20)

### Files to Modify/Create

**New files:**
- `internal/testing/testdata/strip/named_test_unannotated_helper.ail` — class 2 fixture (~8 LOC)
- `internal/testing/testdata/strip/named_test_effectful_helper.ail` — class 3 fixture, the reporter's repro (~6 LOC)
- `internal/testing/testdata/strip/named_test_effectful_direct.ail` — class 4 fixture (~5 LOC)
- `internal/testing/testdata/strip/named_test_inferred_effectful.ail` — class 5 fixture (~6 LOC)
- `internal/testing/named_test_effectful_diagnostic_test.go` — mapper + end-to-end assertions (~150 LOC)

**Modified files:**
- `internal/testing/source_strip.go` — `stripTestBlocks` + comments (~25 LOC)
- `internal/testing/named_batch.go` — base-source switch + mapper call (~30 LOC)
- `internal/testing/property_batch.go` — base-source switch + mapper call (~15 LOC)
- `internal/testing/executor.go` — per-body base switch + mapper call (~15 LOC)
- `internal/testing/bytecode_engine.go` — per-body VM error mapping (~10 LOC)
- `internal/testing/source_strip_test.go` — policy table split (~25 LOC)
- `docs/docs/reference/limitations.md` (or testing reference) + `changelogs/` — contract + workaround note

## Conflict Surface

**Required because the change alters what the named-test/property compile pipeline consumes (base source of `runNamedTestPipeline`) and how its errors are reported.**

1. **Syntactic/semantic positions extended:** none. No grammar, annotation, or effect-row position changes. What changes is *which declarations the temp module retains*.

2. **Other constructs already living in those positions:**
   - The base source also carries `import` lines, module-level `let` bindings, ADTs/types, and contracts. None are touched by either strip today; none are touched after (only FuncDecl ranges were ever removed, and now none are).
   - `test`/`property` blocks: still removed from the base (they are re-added per-body as entries). The string-blind brace scan in `testAndPropertySkipRanges` is unchanged and keeps its own known bug (`source-strip-string-brace-skip-ranges.md`, ailang-core-triage) — out of scope here, unchanged behavior.
   - The reserved-name guard (`__namedtest_` prefix) and batch entry numbering: unchanged.

3. **How disambiguation works:** the effect mapper distinguishes *synthetic* entry symbols (`__namedtest_<k>` batch entries, the `__namedtest_entry` per-body VM wrap, the `$tmp<N>` evaluator wrap) from user symbols by exact pattern on the "Effect checking failed for function 'X'" line, and only rewrites the former. Collisions are closed at the source: a `__namedtest_*` user declaration is already rejected by the reserved-name guard (`prepareNamedTests`), and a `$tmp1()` *call inside a test body* is a parse error (verified live: `PAR_INFINITE_LOOP`), so neither synthetic shape can name a user call that reaches an entry.

4. **Programs that MUST still work post-change (fixtures):**
   - `internal/testing/testdata/strip/named_test_effectful.ail` (the v0.31.0 AC fixture — after the change its `shout` stays in the base, uncalled; the test calls `big` only and still passes)
   - `internal/testing/testdata/strip/named_test_annotated.ail`, `named_test_multiline.ail`, `malformed_control.ail`, `moduleless_contract*.ail`
   - `internal/testing/testdata/engine_parity/*` (parity report identical on both engines)
   - The reporter's control: `pure func check() -> bool = "hello" != ""` + test — passes before and after
   - `examples/` programs using named tests (`make test-core` covers)

5. **What deliberately changes (intentional incompatibilities):**
   - Class 2 turns from fail to pass (that is the point).
   - Classes 3–5 error *text* changes: "undefined variable … add import std/debug (check)" and the `$tmp1`-targeted "Suggested fix" line are replaced by the contract message. Any test asserting the old strings must be updated (`TestNamedBatch_IllTypedBodyFallsBackLoudly` and `source_strip_test.go`'s policy table are the known ones — both assert *mechanism*, and their fixtures are not effectful-helper cases, so they are expected to survive unchanged; the implementer verifies).
   - The batch-failure notice's first line will now carry an effect-check (not type-error) reason for effectful files — still one line, still user-caused, never "harness bug".

## Examples

### Example 1: The reporter's repro (class 3)

**Before (v0.52.5, both engines):**
```
pipeline error: type error in tmp/ailang-namedtest-*/repro (decl 0):
undefined variable: check at .../repro.ail:4:3 — add `import std/debug (check)`
```

**After:**
```
named test "effectful helper" (repro.ail:4:1): test bodies are checked pure, but
this body requires effects not allowed in a pure signature:
  Missing effects: FS

Effectful verification belongs in an exported entry run with `ailang run --caps FS`,
or in a pure test over byte-mirror fixtures.
```

### Example 2: Legal module, unannotated helper (class 2)

```ail
module norow
func check() -> bool = "hello" != ""
test "helper no row" { check() }
```

**Before:** fails "undefined variable: check … add `import std/debug (check)`" (verified, this doc's V3).
**After:** `1 tests: 1 passed, 0 failed` — identical to `ailang run`'s judgment of the same module.

### Example 3: The verified workaround (unchanged, now documented)

```ail
module run
import std/fs (readFile)
export func main() -> bool ! {FS} = readFile("fixture.txt") != ""
```
```
$ ailang run --caps FS run.ail
true
```
(Verified V6 — this is the stapledons-godot M4.5s pattern and stays the route for effectful verification.)

## Success Criteria

- [ ] Class 2 fixture passes on both engines; class 3/4/5 fixtures fail with the contract message (cause + missing effects + workaround), zero synthetic symbols, zero temp paths
- [ ] No `ailang test` output suggests importing a name declared in the same file (assert: mapper tests + a grep gate in the test)
- [ ] All existing fixtures pass unchanged: strip testdata, engine-parity suite, `make test-core`, `make check-boundaries`
- [ ] D1 fallback semantics preserved: a user's effectful body degrades the batch to per-body compiles, is never labeled a harness bug, and non-effectful sibling tests still pass
- [ ] Documentation updated (limitations/testing reference + changelog)
- [ ] All tests passing

## Verification Log

Hard-gate claims, each verified live (binary `AILANG v0.52.5`, 2026-10-08, on this repo's checkout) unless marked as a citation:

| # | Claim | Check | Result |
|---|---|---|---|
| V1 | Classes 1–5 behave as the table says under `ailang test` (evaluator) | repros in `/tmp/repro`, run 2026-10-08 | Confirmed (V2–V5 detail each) |
| V2 | Reporter's repro fails on evaluator AND VM (`ailang test`, `ailang test --bytecode`) | both run; VM adds "0 named-test bodies ran on the VM, 1 fell back …" | Confirmed — both fail with "undefined variable: check … add `import std/debug (check)`" |
| V3 | Unannotated pure helper (class 2) fails today | `func check() -> bool = "hello" != ""` + test → fail, "undefined variable" + hint | Confirmed |
| V4 | `pure` / `! {}` helper passes today (reporter's control) | both run → 1 passed | Confirmed |
| V5 | Direct effectful call in body → "Effect checking failed for function `'__namedtest_0'`" (batch) / `'$tmp1'` (per-body), incl. "Suggested fix: func `$tmp1`(...) -> T ! {FS}" | run captured verbatim | Confirmed |
| V6 | Workaround works: exported entry + `ailang run --caps FS` | runs → `true`, exit 0 | Confirmed |
| V7 | `ailang run` WITHOUT caps fails: "effect 'FS' requires capability, but none provided / Hint: Run with --caps FS" | run | Confirmed — capability gating is runtime-enforced; nothing in this design bypasses it |
| V8 | `ailang test` has no `--caps` flag | `ailang test --help` | Confirmed (only seed/bytecode/strict flags) — hence effectful named-test *support* needs a caps surface and is Future Work |
| V9 | Exporting the helper does not help | `export func check() -> bool ! {FS}` + test → same failure | Confirmed |
| V10 | The strip deletes non-`pure`/`!{}` funcs: `isEffectivelyPure` = `f.IsPure \|\| (f.Effects != nil && len(f.Effects) == 0)`; unannotated → `Effects == nil` → stripped | `internal/testing/source_strip.go:71-73`; parser sets `Effects` only from an annotation (`internal/parser/parser_func.go:113,317`) | Confirmed by code read |
| V11 | Batch entries are emitted `pure func __namedtest_<k>() { <body> }`; batch base built with `stripWithLineMap(src, file, nil)`; per-body base with `stripNonPureFunctions(src, file)`; property paths share the same base builder | `internal/testing/named_batch.go:107,109,305-307`; `property_batch.go:66`; `executor.go:194` | Confirmed by code read |
| V12 | Purity of named-test bodies is a pinned contract, not accidental | `m-named-test-blocks.md` (implemented v0.29.0): A1/A3 rows + "bodies stay pure bool expressions" | Confirmed by citation |
| V13 | This defect class was explicitly deferred as a filing, with a verified repro, in the strip sprint | `m-strip-contract-awareness-sprint-plan.md` (planned v0.31.0), "Deliberately OUT of scope" section | Confirmed by citation |
| V14 | `check` is a std/debug export (why the hint fires) | `std/debug.ail:37` `export func check(cond: bool, msg: string) -> () ! {Debug}` | Confirmed |
| V15 | The effect checker already emits "Missing effects:" and signature rows (`formatEffectError`) — the mapper preserves detail instead of inventing it | `internal/pipeline/validate_effects.go:520-551` | Confirmed by code read |
| V16 | Unannotated effectful helper is illegal module-wide, not just in tests (class 5 premise) | `ailang check inferred.ail` → demands `! {FS}` | Confirmed |
| V17 | The per-body evaluator path wraps the folded body as a bare top-level block `{ … }` (`executor.go:249-255`), whose effect errors name `$tmp<N>` (observed V5); the per-body VM path wraps `pure func __namedtest_entry() -> <type>` (`bytecode_engine.go:111`); a `$tmp1()` call inside a test body is a parse error | both wraps read; PAR observed live (`PAR_INFINITE_LOOP`) | Confirmed |
| V18 | `stripNonPureFunctions` has a third caller (binding extraction, `executor.go:440`) that must NOT change | grep: 3 call sites (executor.go:194, 440; source_strip.go internal) | Confirmed — enumerated in Design Freeze |
| V19 | Cited regression fixtures exist | `ls internal/testing/testdata/strip/` (6 files), `testdata/engine_parity/` | Confirmed |
| V20 | No current doc/prompts claim named tests support effects | grep of `docs/`, `prompts/` for named-test + effect claims | Confirmed — nothing to correct; documentation deliverable is additive |

## Testing Strategy

**Unit tests:**
- `stripTestBlocks` table: funcs present (annotated, unannotated, effectful), test/property blocks absent, line map correct across removed blocks
- `mapEntryEffectError`: fires on all three synthetic shapes, never on user symbols, preserves "Missing effects:" rows

**Integration tests:**
- The four new fixtures through the full `ailang test` surface on both engines, plus `--strict-bytecode` (strict fails the one test, siblings unaffected)
- `TestNamedBatch_OutcomesMatchPerBody` extended with an effectful sibling: batch degrades loudly, outcomes still equal per-body

**Manual testing:**
- The reporter's exact repro before/after; the stapledons-godot workaround entry still runs under `--caps FS`

## Deferred Decisions

- Exact wording of the contract message — agent may choose (must contain: contract name + doc, missing effects, `--caps FS` route)
- Whether the message shows the helper's signature when the callee is statically known — agent may choose (nice-to-have; the effect row suffices for the gate)
- Which reference page carries the user-facing contract note (limitations vs. testing reference) — sprint-planner may choose
- Whether `source_strip_test.go`'s old policy table is kept as-is beside the new one or trimmed — agent may choose

## Non-Goals

**Not attempted in this feature:**
- **Supporting effectful named tests** (a `--caps` surface for `ailang test`, per-test capability grants) — changes the v0.29.0 purity contract, drags in determinism/replay questions for seeded properties, and needs a caps/security design of its own. This feature makes the contract *truthful*, not optional.
- **Touching `internal/types` hint machinery** — dissolved by M1 (V10/V14 reasoning); genuinely undefined names keep the hint, where it is correct.
- **Changing the binding-extraction strip site** (`executor.go:440`) — separate surface, separate doc if it ever bites.
- **Fixing the string-blind brace scan** in `testAndPropertySkipRanges` — already triaged (`source-strip-string-brace-skip-ranges.md`); this feature keeps that behavior byte-identical.
- **Unifying the per-body wraps** (bare block vs `pure func` entry) — the mapper keys on both shapes; unification is a refactor with its own regression risk and no user-visible gain here.

## Timeline

**Week 1** (2 days):
- Phase 1: strip policy + class 2 fixture + strip-test table split

**Week 2** (2 days):
- Phase 2: diagnostics mapper + class 3/4/5 fixtures + parity
- Phase 3: documentation + regression sweep (`make test-core`, `make check-boundaries`)

**Total: ~4 days across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Kept effectful declarations change outcomes for files whose tests pass today *only because a broken helper was stripped* (an unreferenced effectful helper that fails `ailang check`, e.g. an unannotated `func` calling `println` — stripped today, kept after) | Med | That is consistency, not regression: a broken **pure** helper is kept today and already fails every test in the file, so after M1 both classes behave identically — the module's real error surfaces, mapped to the user's lines through `lineMap`. Such a file already fails `ailang check`; `ailang test` now agrees with it (A3/A11). Fixtures V19 + full `make test-core` sweep; D1 fallback still isolates per-test failures. |
| A user's file that fails `ailang check` now also fails every `ailang test` compile (class 5), where today it fails with a lie | Low | That is the fix, not the regression: the module's real error maps to the user's lines through `lineMap`, same as any other module-level error today. |
| The mapper misses a synthetic shape (fourth wrap form appears later) | Med | Mapper is table-driven over the three known shapes (V17); a missed shape falls through to today's mapped (untruthful but positioned) error — never worse than status quo. Parity fixtures pin all three. |
| Effect-row syntax drift (e.g. `! {}` vs `pure` semantics change) invalidates `isEffectivelyPure`-free policy | Low | Policy no longer *uses* `isEffectivelyPure` for the test base; the binding site keeps its own predicate with its own tests. |

## Related Documents

**Implemented (may inform design):**
- [m-named-test-blocks.md](implemented/v0_29_0/m-named-test-blocks.md) (v0.29.0) — the named-test execution model and the purity contract this feature repairs the enforcement of
- [m-property-seed-determinism-sprint-plan.md](implemented/v0_31_0/m-property-seed-determinism-sprint-plan.md) (v0.31.0) — the strip-bug history that motivated declaration-aware stripping
- [m-dx26-ensures-result-binding-sprint-plan.md](implemented/v0_20_0/m-dx26-ensures-result-binding-sprint-plan.md) (v0.20.0) — the third strip call site's context

**Planned (check for overlap):**
- [m-test-runner-compile-once.md](v0_53_0/m-test-runner-compile-once.md) (same directory) — the batching this feature extends; its D1 fallback is reused unchanged
- [m-strip-contract-awareness-sprint-plan.md](../v0_31_0/m-strip-contract-awareness-sprint-plan.md) (v0.31.0) — the sprint that explicitly deferred this defect as a filing
- [ai-deadline-and-named-test-roundtrip.md](../ailang-core-triage/ai-deadline-and-named-test-roundtrip.md), [source-strip-string-brace-skip-ranges.md](../ailang-core-triage/source-strip-string-brace-skip-ranges.md), [named-test-float-dict-resolution.md](../ailang-core-triage/named-test-float-dict-resolution.md) — sibling strip/roundtrip defects, kept out of scope (Non-Goals)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- `internal/testing/source_strip.go`, `internal/testing/named_batch.go`, `internal/testing/property_batch.go`, `internal/testing/executor.go`, `internal/testing/bytecode_engine.go`, `internal/pipeline/validate_effects.go`, `internal/types/import_hint.go`
- Reporter: task repro (repro.ail + fixture.txt), stapledons-godot iteration 21 / M4.5s workaround

## Future Work

- **Effectful named tests, properly:** a `--caps` surface for `ailang test` with per-file capability grants, a determinism ruling for seeded properties that share the batch, and an explicit human decision re-opening the v0.29.0 purity contract. Blocked on the capability design, not on this diagnostic.
- **Pre-scan naming of the offending callee** (decorating the mapped error with the helper's signature when statically known) — cheap follow-up once M2 lands.
- **Unifying per-body wraps** (bare block vs `pure func` entry) to shrink the mapper's shape table to one row.

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08