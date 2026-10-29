# M-RUN-ENTRY-NO-EXPORTS-FAIL-LOUD: `ailang run` on a module with no exports must fail loudly, not exit 0

**Status**: Implemented. v0.54.13 (b6c9f592e, #1797) made a declared-but-unexported entry an exit-1 error. The follow-up makes every zero-export module run exit 1 too, with a message naming `export func main()` or `ailang check` (`internal/runner/entry_unexported.go`). The ~10 export-less example modules are now type-checked by `scripts/verify_examples.go` (`checkExample`) instead of run, so verify-examples stays green. `--batch` takes the same branch and also exits 1; `--caps auto` is unchanged.
**Target**: v0.54.8
**Priority**: P1 (silent-success class — violates "NO SILENT FALLBACKS, FAIL LOUDLY"; second instance of the class fixed in v0.30.0)
**Estimated**: ~0.5 day (guard change + fix-carrying message + tests)
**Dependencies**: none; complements [m-module-less-run-fail-loud (implemented, v0.30.0)](../../implemented/v0_30_0/m-module-less-run-fail-loud.md) — that doc named this exact guard as root cause #2 but fixed it only at the pipeline level

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The run outcome becomes a function of the file's declared mode (module vs script), not of the export table's size — removes a mode-dependent silent branch |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | No effect-row changes; effects already declared in signatures |
| A4: Explicit Authority | +1 | Preserves `Iface.Exports` as the authority surface: nothing private executes merely because it is named `main` |
| A5: Bounded Verification | 0 | Typecheck path unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | An agent reading exit 0 + empty stdout no longer needs human inference to discover the missing `export` |
| A8: Minimal Syntax | +1 | No new syntax; error text carries the fix |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | 0 | No language surface changes |
| A11: Structured Failure | +1 | Converts a silent success into a named, fix-carrying failure |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

A `.ail` file with a `module` declaration whose only `func main …` is **not exported** — plus any
`--entry` name — **exits 0 with no output and no error**. The entry never runs; the flags
(`--entry`, `--caps`, `--batch`) are silently ignored. Verified live on the repo binary
(v0.54.6, `71f4cf6`) and reported by the user on v0.54.5/v0.54.6:

```
$ cat nomain.ail
module nomain
import std/io (println)
func main() -> () ! {IO} { println("hi") }

$ ailang run --caps IO --entry main nomain.ail; echo $?
0        (prints nothing — no error either)
```

Adding `export` prints "hi" (verified: `examples/hello_world.ail` runs and prints, exit 0).

**Current State:**
- Exit 0 + empty stdout/stderr reads as *success* to every downstream consumer (eval harnesses,
  benchmark sweeps, shell loops, agents). A model-generated program missing one `export` keyword
  sinks as an empty-output "success" with no cause — the same trap class the v0.30.0 fix documented.
- The cost is real: the reporter "lost a debugging round to it while writing a
  semantic-similarity spike".

**Impact:** every AI author and every shell script that runs `.ail` files.

## Root Cause (traced, verified)

`internal/runner/run.go:269`:

```go
// Only attempt entrypoint resolution if the module has exports
if result.Interface != nil && len(result.Interface.Exports) > 0 {
    ... runBatch / runSingle ...
} else {
    // Non-module mode - print result if evaluated by pipeline (ModeEval)
    PrintNonModuleResult(result.Value, opts.Print, opts.NoPrint)
}
```

The guard uses **export count** as a proxy for **script-vs-module mode**, but mode is already
computed from the same source earlier (`isModuleFile` at run.go:199-207 → `mode` at run.go:212-213). A
module file *without* exports is misclassified into the script branch:

1. For module files the pipeline **always** sets `result.Interface` (`pipeline_module_phases.go:454`,
   `st.result.Interface = rootUnit.Iface`; `iface/builder.go:344` `NewIface` — non-nil even with zero
   exports), so `Interface != nil` holds and `len(Exports) == 0` is what fails the guard.
2. `PrintNonModuleResult` prints only a **non-unit** `result.Value`; in ModeCheck `result.Value` is
   nil (`pipeline_module_phases.go:440-446` evaluates first decl **only** in ModeEval), so nothing
   prints.
3. `Run` falls through to `return 0`. `--entry`, `--caps auto` resolution, and `--batch` — all inside
   the guard — are silently skipped.

**Second instance of a known class.** [m-module-less-run-fail-loud (v0.30.0)](../../implemented/v0_30_0/m-module-less-run-fail-loud.md)
fixed the *mirror* case (file with declarations but **no `module` line**, → MOD014) and named this
exact `len(Exports) > 0` guard as its root cause #2 — but fixed it at the pipeline level only. A file
that *has* `module` but no `export`s still slips through the same guard. This doc fixes the guard
itself (systemic, not another special case).

## Goals

**Primary Goal:** `ailang run` on a module whose entry (or nothing) is not exported **fails loudly**
with a fix-carrying error, exit 1 — while every legitimate mode (scripts, `ailang test`, modules
with exports) is preserved byte-for-byte.

**Success Metrics:**
- `ailang run --entry <any> <zero-export module>` → exit 1, stderr names the module and says to add `export`
- `ailang run <module with exports>` → unchanged output/exit
- `ailang run <non-module script>` → unchanged output/exit
- `ailang test <zero-export module>` → unchanged (separate harness path)
- `--batch` on a zero-export module → exit 1 with the same error (currently also silent)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fail loudly (exit 1) rather than auto-run the non-exported `main` | Running a private function by name bypasses `Iface.Exports` — the public contract that also drives `--caps auto` — and is inconsistent with today's behavior when *other* exports exist (case B already errors) | human (reporter offered either; this doc recommends fail-loud) | design | low |
| Branch on the already-computed mode (`isModuleFile`), not on export count | Makes mode the single source of truth for script-vs-module dispatch; fixes `--batch` and `--caps auto` silent skips in the same change | agent | design | low |
| No new `MOD` diagnostic code; plain runner error in the existing entrypoint-resolution style | `MODxxx` codes are the pipeline/loader namespace (MOD010–MOD015 allocated); a zero-export module is *well-typed* (`ailang check` must keep passing — only run policy changed). Sibling entrypoint errors carry no code | agent | design | low |
| Error text carries the concrete fix (``export func main() → …``), not just the fact | The reporter's ask; matches A7/A8 (agent can self-repair from stderr alone) | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Fail loudly vs run-anyway — **fail loudly** (recommendation above; reporter's "or run it anyway" alternative rejected for the encapsulation/consistency reasons in the row)
- [x] Guard refactor vs pre-check in `Run` — **guard refactor** (systemic; covers batch + auto-caps)
- [x] Diagnostic code allocation — **none** (style-consistent plain error)

All change costs are low; nothing here requires sprint-executor to pause.

## Solution Design

### Overview

Change the dispatch predicate in `internal/runner/run.go` from export count to the mode decision
already computed, and give `ExecuteModuleEntrypoint` a dedicated fix-carrying error for the
zero-export case. Two small edits, one behavior class fixed.

### Architecture

**Components:**
1. **`internal/runner/run.go:269`** — replace
   `if result.Interface != nil && len(result.Interface.Exports) > 0 {`
   with `if isModuleFile {`. The variable already exists in scope (computed at run.go:199-207,
   assigned at 207; it is the same value that chose `mode` at run.go:212-213); module files keep flowing to `runBatch`/`runSingle`, scripts keep
   the `else` branch. `result.Interface` is guaranteed non-nil on the module path (V2), so
   `runSingle`'s and `ExecuteModuleEntrypoint`'s uses are safe.
2. **`internal/runner/entrypoint.go`** — in `ExecuteModuleEntrypoint`, before the auto-select block
   (entrypoint.go:65), add the empty-exports case:

   ```go
   if !exists && len(params.Iface.Exports) == 0 {
       return fmt.Errorf("module '%s' has no exported functions; entrypoint '%s' cannot run.\n"+
           "Add `export` to the function you want to execute, e.g. `export func main() -> () ! {IO} { ... }`.\n"+
           "Use `ailang test` for files whose functions are inline tests.", params.Iface.Module, entry)
   }
   ```

   The auto-select (zero-arg) and "Available exports:" paths are untouched — they already behave
   correctly for modules that export *something*.

3. **No other callers.** Only `cmd/ailang/main_run_exec.go:96` constructs `runner.Options` (V3);
   the `ailang test` harness synthesizes its own entry (`__namedtest_entry`, `internal/testing/`)
   and never touches this guard (V4); `internal/embed` does not route through `runner.Run`.

### Implementation Plan

**Phase 1: Fix + tests (~3 hours)**
- [ ] Guard change in `run.go` (branch on `isModuleFile`)
- [ ] Empty-exports error in `entrypoint.go`
- [ ] E2E test (`cmd/ailang`): zero-export module → exit 1, stderr matches `no exported functions.*export func`; `--batch` variant → same
- [ ] Regression tests: exported-`main` module runs unchanged; non-module script evaluates unchanged; `ailang test` on a zero-export module unchanged

**Phase 2: Docs + changelog (~1 hour)**
- [ ] `docs/docs/reference/modules.md` (Exports section): note that only exported functions can be an `--entry`, with the new error text
- [ ] Changelog entry for v0.54.8

### Files to Modify/Create

**New files:**
- `cmd/ailang/run_no_exports_test.go` — e2e: silent-exit regression + fix-carrying message (~60 LOC)

**Modified files:**
- `internal/runner/run.go` — one-line guard change (~1 LOC)
- `internal/runner/entrypoint.go` — empty-exports error (~5 LOC)
- `docs/docs/reference/modules.md` — short paragraph (~5 LOC)
- changelog for v0.54.8

## Examples

### Example 1: the reported bug

**Before (v0.54.5/0.54.6):**
```
$ ailang run --caps IO --entry main nomain.ail; echo $?
0        (nothing printed)
```

**After:**
```
$ ailang run --caps IO --entry main nomain.ail; echo $?
Error: module 'nomain' has no exported functions; entrypoint 'main' cannot run.
Add `export` to the function you want to execute, e.g. `export func main() -> () ! {IO} { ... }`.
Use `ailang test` for files whose functions are inline tests.
1
```

### Example 2: what stays identical

```
$ ailang run --caps IO --entry main with_export.ail   # export func main …
hi                                                     # unchanged
$ ailang run --entry main other_export.ail             # exports exist, none named main (previously exit 1)
Error: entrypoint 'main' not found in module
Available exports: [run]                              # unchanged — this path was always loud
```

## Verification Log

Every "does/doesn't" claim above was checked live (repo binary v0.54.6, `71f4cf6`) or by reading
the cited file:line:

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| V1 | Zero-export module exits 0 silently on `--entry main` | `ailang run --caps IO --entry main nomain.ail` (repro above, temp dir) | exit 0, no output — bug confirmed |
| V2 | For module files `result.Interface` is always non-nil (so the guard fails on `Exports == 0`, not on `Interface == nil`) | `pipeline_module_phases.go:454` sets `st.result.Interface = rootUnit.Iface`; `iface/builder.go:343-350` `Build` starts `NewIface(b.module)` and only returns nil on error | confirmed |
| V3 | `runner.Options` has exactly one non-test constructor | `grep -rn "runner.Options{" internal/ cmd/ --include="*.go" \| grep -v _test` | only `cmd/ailang/main_run_exec.go:96` |
| V4 | `ailang test` does not route through the run guard | `internal/testing/bytecode_engine.go:38-40,111,128` synthesizes `__namedtest_entry`; live: `ailang test examples/inline_tests_nullary.ail` | exit 0, tests pass — unaffected path confirmed live |
| V5 | `--entry` on a module *with* exports but a missing name is already loud | `ailang run --caps IO --entry main other_entry.ail` | `Error: entrypoint 'main' not found in module`, `Available exports: [run]`, exit 1 |
| V6 | `--entry <non-main>` on a zero-export module is also silent today | `ailang run --caps IO --entry whatever nomain.ail` | exit 0, no output — same bug, any entry name |
| V7 | Non-module script path (the guard's legitimate `else`) runs unchanged | `ailang run --print bareexpr.ail` (`41 + 1`) | prints `42`, exit 0 — ModeEval path untouched by this design. Pre-existing quirk noted: a top-level-IO script does not receive `--caps` grants today (caps are applied in `runSingle`, which the script path never reaches) — out of scope, recorded in Future Work |
| V8 | `MOD` codes MOD001–MOD015 are allocated; none needed here | `grep -rn "MOD01[0-9]" internal/ -o \| sort -u` → MOD010–MOD015 live; `dist/error_codes.json` → MOD001–MOD013+ | plain runner error chosen; **no code allocated**, so no collision to check |
| V9 | `--batch` on a zero-export module is silently skipped today (inside the same guard) | `run.go:289` — `if opts.Batch` is inside the guard; same `else` swallows it | covered by the same fix (systemic row) |
| V10 | `--caps auto` resolution is inside the guard too | `run.go:271-283` | covered by the same fix |
| V11 | Zero-export example fixtures exist and are referenced only as `ailang test` fixtures, not via `run` | `grep` over repo for `pattern_matching_adt\|inline_tests_nullary\|record_in_result` outside `examples/` | no references — `ailang run` on them starts erroring, which is correct new behavior |
| V12 | `examples/hello_world.ail` (exported main) keeps running | live `ailang run --caps IO --entry main examples/hello_world.ail` | prints `Hello, World!`, exit 0 |
| V13 | `check` on a zero-export module passes (must stay that way) | `ailang check` on `nomain.ail` | `✓ No errors found!` — run-only fix preserves it |
| V14 | Planning validator already warns on zero-export modules | `internal/planning/validator.go:185` VAL_M04 "Module has no exports" | precedent exists at warning level; this doc is the run-level fix |

## Conflict Surface

No parser, lexer, grammar, typechecker, or effect-row changes — the mandatory parser-oriented
analysis does not apply. The touched *semantic* position is the script-vs-module dispatch in
`runner.Run`, and its existing claimers:

### Dispatch positions touched

| Position | Existing claimers | Shape | After change |
|---|---|---|---|
| Guard at `run.go:269` | (a) module file with ≥1 export → `runSingle`/`runBatch` | unchanged | unchanged |
| | (b) non-module file (ModeEval) → `PrintNonModuleResult(result.Value)` | unchanged | unchanged (`isModuleFile == false`) |
| | (c) module file with 0 exports → script branch (silent) | **the bug** | → `runSingle` → `ExecuteModuleEntrypoint` → loud error |

### Disambiguation strategy

`isModuleFile` (run.go:199-207) is computed before the pipeline call and *is* the value that chose
`mode` — there is no second source of truth. A file is a module iff a line starts with `module `;
that heuristic is pre-existing and untouched by this change.

### Programs that MUST still work (regression fixtures)

- `examples/hello_world.ail` — exported `main`, must print (V12)
- `examples/inline_tests_nullary.ail` — zero-export module via `ailang test` (V4, live-verified)
- `examples/pattern_matching_adt.ail` — zero-export module; `ailang test` semantics unchanged
- `examples/inline_tests_arithmetic.ail` — zero-export module, test-only fixture
- A bare non-module script — `printf '41 + 1\n'` with `--print` → `42`, ModeEval path unchanged (V7)

### What deliberately changes

- `ailang run` on any module file with zero exports (with or without `--batch`, any `--entry`)
  **exits 1** with the fix-carrying error instead of silently exiting 0. Migration: add `export`.
  `ailang check` on the same file **still passes** (V13) — a zero-export module is well-typed;
  only run policy changed. Not listed here and breaking = regression.

## Success Criteria

- [ ] `ailang run --caps IO --entry main <zero-export module>` exits 1 and stderr contains "no exported functions" and "export func main" (acceptance: new e2e test)
- [ ] Same for `--batch` on a zero-export module (acceptance: e2e test variant)
- [ ] `examples/hello_world.ail` output/exit unchanged (acceptance: existing fixtures + new regression test)
- [ ] Non-module script run unchanged (acceptance: regression test)
- [ ] `ailang check` unchanged on zero-export modules (acceptance: existing test corpus)
- [ ] All tests passing
- [ ] Documentation updated (`modules.md` Exports section)
- [ ] Changelog entry added

## Testing Strategy

**Unit/integration tests (`cmd/ailang/run_no_exports_test.go`, e2e via `runAilangBin`):**
- Zero-export module → exit 1, message matches (fix-carrying text asserted with a loose regex — exact wording may be tuned)
- `--batch` on zero-export module → exit 1, same class
- Exported-`main` module → output + exit 0 byte-identical
- Non-module script → output + exit unchanged

**Regression-surface tests:** one per "Programs that MUST still work" fixture above (run/test level).

**Manual testing:**
- `make test` (incl. golden sweep) — confirm no fixture relied on the silent exit
- Live repro of the reporter's transcript (the `nomain.ail` case) before/after

## Deferred Decisions

- Exact error wording — agent may tune while keeping the three semantic anchors: module name,
  "no exported functions", and the ``export func`` fix. (Message asserted loosely in tests for that reason.)
- Whether `--caps auto` should print its grant line *before* failing on a zero-export module —
  implementer may pick the cleaner ordering; both are loud.
- Whether the fix-carrying hint should also mention `ailang check`'s silence (VAL_M04 exists only
  in planning) — optional, agent's call.

## Non-Goals

**Not attempted in this feature:**
- Auto-running non-exported `main` — rejected (see High-Impact Decisions; breaks the exports-as-contract model and diverges from the already-loud case B)
- Making `ailang check` warn/error on zero-export modules — planning validator already warns (VAL_M04, V14); `check` is a typechecking command, a private-only module is well-typed
- Deeper `hasModuleKeyword` heuristic changes (mid-line `module`, strings) — pre-existing, orthogonal, no new risk introduced here
- MOD-code allocation for runner-level run-policy errors (V8)

## Timeline

**Total: ~0.5 day** (single sprint slice)

- Phase 1: guard + message + e2e/regression tests (~3 hours)
- Phase 2: docs + changelog (~1 hour)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Some harness/eval fixture somewhere *relied* on silent exit 0 for a zero-export module | Med | grep found no references (V11); sprint runs `make test` + golden sweep; `ailang test` path unaffected (V4) |
| `--caps auto` ordering change observable on zero-export modules | Low | Both orderings are loud (error wins either way); deferred to implementer |
| Message-text churn breaks strict-output tests | Low | New test asserts loosely; no existing test asserts on this (previously silent) path |

## Related Documents

**Implemented (informs this design):**
- [m-module-less-run-fail-loud (v0.30.0)](../../implemented/v0_30_0/m-module-less-run-fail-loud.md) — the mirror case: module-less file → MOD014. Named this guard as root cause #2; this doc finishes the job at the guard.

**Planned (check for overlap):**
- (none found — SimHash and neural search returned no matches; v0_54_8 folder contains only this doc)

**Search note (duplicate gate):** dual search (SimHash + neural) on "run entry no exports fail loud"
returned zero matches ≥ 0.45 on both methods, so no warn/reject band applies. The nearest neighbor
is the v0.30.0 doc above — same *class* (silent run success), **opposite trigger** (missing `module`
declaration vs. missing `export` keyword) and a different fix site (pipeline diagnostic vs. runner
dispatch). Distinct, complementary.

## References

- [Design Axioms](/docs/references/axioms)
- [docs/docs/reference/modules.md — Exports](../../../docs/docs/reference/modules.md)
- CLAUDE.md §2 "No silent fallbacks — fail loudly"
- Reporter transcript: v0.54.5/0.54.6, `ailang run --caps IO --entry main nomain.ail` → exit 0

## Future Work

- Audit other `len(Exports)` proxies for the same conflation (`cmd/ailang/docs.go:420,443` are
  display-only; `internal/planning` already warns) — nothing further found in this pass.
- Consider a lint that flags a `main` function that is not exported in a *root* module (would have
  caught this at author time).
- Pre-existing (observed while writing V7, **not** this design's scope): the non-module script
  path never receives capability grants — `--caps IO` on a top-level-IO script still fails with
  `effect 'IO' requires capability` because `GrantCapabilities` runs inside `runSingle`. Same
  script-vs-module dispatch family; fix belongs in a separate doc if it bites.

---

**Document created**: 2026-10-13
**Last updated**: 2026-10-13