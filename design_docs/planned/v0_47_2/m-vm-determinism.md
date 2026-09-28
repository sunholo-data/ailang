# M-VM-DETERMINISM: Bytecode VM nondeterminism — Go map iteration order selects record types/field indices at compile time

**Status**: Planned
**Target**: v0.47.2
**Priority**: P0 (High) — breaks the VM/interpreter bit-identity gate (`make parity`) and blocks the game's M1.6 landing
**Estimated**: 4 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Eliminates per-process randomness in bytecode generation; restores VM/interpreter bit-identity |
| A2: Replayability | +1 | Identical bytecode for identical source makes traces reproducible across runs |
| A3: Effect Legibility | 0 | No effect-surface change |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | A compile-idempotence test makes determinism locally checkable per build |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | VM/interpreter output equivalence is a machine-decidability guarantee the game relies on |
| A8: Minimal Syntax | +1 | No syntax changes; fixes existing semantics |
| A9: Cost Visibility | 0 | The by-name field fallback is a known O(n) cost, unchanged from today's `_record_get` path |
| A10: Composability | 0 | No change to module composition |
| A11: Structured Failure | +1 | Ambiguity that today silently mis-reads a slot now either resolves correctly or fails loudly |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes implicit nondeterminism (the current state IS a hard A1 violation in the VM path)
- [x] A3 (Effects): no hidden side effects introduced
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Problem Statement

The `--bytecode` VM produces **different output for the same program + stdin across runs** (~1 run in 4
divergent; the independent evaluator measured two stable output hashes over 20 runs, split 15/5). The
interpreter is stable. This breaks the VM/interpreter bit-identity gate the game (`stapledons-godot`,
M1.6 free motion) treats as its determinism guarantee, and blocks M1.6.

**Reported evidence** (commit f3d6975 of the game branch `m1.6-free-motion-iter1`, AILANG v0.45.0/v0.47 rig build):

- 1-line NDJSON input: wrong second line in **5 of 25** VM runs — `step()`'s result is lost while the
  handler still takes the success branch (`"status":"ok"`, tick 0, untouched initial state).
- 17-line `tests/fixtures/offaxis.ndjson`: **6–9 of 30** VM runs diverge from the interpreter.
- Interpreter: **0 of 40** wrong.
- Controls: the 600-line v1.0 input (never sends a nested `heading` object) is 0 of 30 divergent; a
  standalone `std/json` probe (decode → nested `get` → `getNumber`) is 25/25 stable; a standalone
  `turn()` → `step()` reimplementation with an `ensures` contract is 30/30 stable.
- Renaming the colliding field `ship` in `TurnResult {accepted, ship}` / `Response {ship, reply}` still
  left 6 of 30 divergent — not the same *field-slot* bug as the companion strict GET_FIELD report, but
  consistent with the same root cause (see below).

**Impact:**

- Every `ailang run --bytecode` user. The VM is the deterministic execution substrate for
  AI-generated code; nondeterminism here violates A1 directly.
- The game's `make parity` gate and M1.6 landing are blocked on it.

### Root Cause (verified by code reads — see Verification Log)

The bytecode compiler selects **record types and field indices by iterating a Go map**, whose
iteration order Go randomizes. `ailang run` recompiles the program on every invocation
([internal/pipeline/cache_alias_digest_stable_test.go:13](../../internal/pipeline/cache_alias_digest_stable_test.go)
documents per-run recompiles), so each run samples a fresh map order and bakes the outcome into that
run's bytecode — exactly the "two stable output hashes per run, 15/5" signature.

Three sites in `internal/bytecode/compiler/`:

1. **`lookupFieldIndex` fallback** (`collections.go:274`) — when `stmt.FieldAccess.KnownFields` is
   empty (the lower pass could not resolve the record's static type — e.g. values flowing out of
   JSON decoding or through polymorphic paths), it scans `fc.recordTypes` (a
   `map[string]recordTypeInfo`, built at `compiler.go:47`) and returns the field's sorted index within
   the **first type that merely contains the field name**. With ≥2 registered record types sharing a
   field name (a sim full of `{x,y,z}` headings/positions/velocities has many), the chosen index
   depends on map order → `GET_FIELD` reads the wrong slot.
2. **`compileRecordUpdate`** (`collections.go:294`) — for `{base | field: value}` it picks the
   record type as the **first map entry whose field set contains all updated fields**, then rebuilds
   the record in that type's sorted field order. Wrong pick → wrong field list and wrong `GET_FIELD`
   slots in the copy step.
3. **`inferADTFromCases`** (`switch.go:140`) — same first-match pattern over `fc.adtTypes`; a wrong
   ADT pick gives different tag ordinals and mis-dispatches the switch.

**Why the wrong output is silent:** the VM's `GET_FIELD` (`internal/vm/vm.go:443`) only bounds-checks
the index and returns `fields[idx].Value` — an in-range wrong index reads another field's value with
no name check. That maps 1:1 onto the reported symptom: `step()`'s result is lost while the handler
still takes the success branch.

**Why the interpreter is stable:** `evalCoreRecordAccess`
([internal/eval/eval_expressions.go:438](../../internal/eval/eval_expressions.go)) looks the field up
**by name** on the runtime record — no compile-time index guess.

**Why the controls are stable:** a single registered type containing the field is found
deterministically even in random order (the standalone JSON probe / standalone `turn`→`step` repro
have no second candidate); the 600-line input never touches a nested decoded object, so
`KnownFields` resolves and the map scan is never entered. Renaming `ship` removed one colliding
name but left the others (`x`/`y`/`z`, etc.), hence 6 of 30 stayed divergent. This also explains the
companion "strict GET_FIELD wrong slot" report: the same first-match scan can pick the wrong type
*deterministically* — same root cause, different coin flip.

**This is the third measured instance of the map-order pattern in the pipeline:** the iface digest
bug (`ailang run` recompiled every run because record printing was map-ordered; fixed by 282c02315
"print record fields in sorted order", confirmed by Daneel 2026-09-28) and the record-print ordering
were instances 1–2. This doc fixes instance 3 and adds a guard against instance 4.

## Goals

**Primary Goal:** `ailang run --bytecode` on any program is bit-identical across runs and to the
interpreter — compile-time candidate selection in the bytecode compiler is deterministic and
semantically correct.

**Success Metrics:**

- The game's 1-line and offaxis repros: 30/30 VM runs identical to the interpreter output (was 5/25
  and 6–9/30 divergent).
- New compile-idempotence test: 200 in-process compiles of an ambiguity fixture produce identical
  serialized bytecode **and** the correct values (currently fails immediately).
- New map-order guard test: `internal/bytecode/compiler` contains zero order-sensitive `for … range`
  over maps (or every remaining one is explicitly allowlisted with a justification comment).
- All existing tests pass (`make test-core`, golden corpus in `multimodule_test.go`).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Unresolved field access resolves **by name** (existing `_record_get` path), never via a first-match map scan — even when only one candidate type matches | Uniqueness among *registered* types says nothing about the runtime record (the reason the fallback exists); this is the only option that matches interpreter semantics exactly | compiler (A1 semantics) | design | low |
| Record update on an unresolvable base type compiles to a new runtime `OpUpdateRecord` (mirrors `evalCoreRecordUpdate`) instead of guessing a type | Fail-loud would break currently-working programs and parity; guessing is the bug | agent (with human approval via doc review) | design | med |
| New opcode = image-format surface change (`opcode.go`, `vm.go`, `disasm.go`, `image.go`) | Serialized bytecode images gain an opcode; the switch at `image.go:329` must handle it | agent | compile | med |
| ADT inference ambiguity de-randomized to **source declaration order**, tag-name soundness deferred | Deterministic-wrong vs random-wrong; full soundness fix (tag identity at runtime) is a separate workstream | human (deferred decision ratified) | design | low |
| Guard scope: `internal/bytecode/compiler` package only (this doc's blast radius) | Whole-pipeline sweep is worthwhile but unbounded; the guard is extensible later | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Unresolved field access → always by-name (chosen in this doc; ratify in review)
- [x] Record update unresolvable base → `OpUpdateRecord` runtime path (chosen in this doc; ratify in review)
- [ ] Human approval of this design doc (per work-routing gate)

## Solution Design

### Overview

Remove every order-sensitive map iteration from the bytecode compiler's candidate selection, and
make each fallback semantically equal to the interpreter instead of a compile-time guess. Add a
determinism regression test and a static map-range guard so the pattern cannot return.

### Architecture

**Fix A — Field access (`internal/bytecode/compiler/collections.go`):**

- Keep the primary paths: `KnownFields` (from the lower pass, already sorted —
  `internal/gen/lower/expr.go:246-280` `recordFieldSet` collects map keys then `sort.Strings`) and
  tuple `_N` positional access (unchanged, positional).
- Delete the `for _, info := range fc.recordTypes` scan in `lookupFieldIndex`'s fallback. When the
  static resolution fails, always emit `compileFieldAccessByName` (already exists, `collections.go:228`)
  → `OpBuiltinCall _record_get`, a runtime linear scan by name over the record's own fields
  (`internal/vm/builtins.go:143`, fields stored sorted by `NewRecord`, `internal/bytecode/value.go:267`).
  This is exactly `evalCoreRecordAccess` semantics — correct for row-polymorphic and anonymous
  records regardless of which other types exist in the program.

**Fix B — Record update (lower pass + compiler + VM):**

- Add `KnownFields []string` to `stmt.RecordUpdate` (`internal/gen/stmt/stmt.go`), populated in
  `lowerRecordUpdate` (`internal/gen/lower/expr.go:464`) with `recordFieldSet(cti, e.Base)` — the
  same helper already used for `FieldAccess`. The compiler uses it when present: deterministic and
  type-correct (replaces the map scan at `collections.go:294` entirely).
- When absent (base type unresolvable — the JSON-decode path), emit a new `OpUpdateRecord`:
  `A=dst, B=base record, C=override count`, override values in contiguous registers, followed by C
  pseudo-`LOAD_CONST` name constants — the exact encoding pattern `OpMakeRecord` already uses
  (`vm.go:405`, `collections.go:150-180`). VM handler: copy the base record's runtime field slice,
  apply overrides by name, `bytecode.NewRecord` (which re-sorts and rejects duplicates). This mirrors
  `evalCoreRecordUpdate` (`internal/eval/eval_expressions.go:465+`) operation-for-operation.
- Teach `opcode.go`, `disasm.go`, and `image.go:329`'s serializer switch the new opcode.

**Fix C — ADT inference (`internal/bytecode/compiler/switch.go:140`, `compiler.go`):**

- Keep an ordered `adtTypeNames []string` (registration order = source order, already available in
  the `prog.TypeDecls` loop at `compiler.go:49`) alongside the map; `inferADTFromCases` iterates the
  slice. If more than one ADT matches all case tags, first-in-source-order wins — deterministic.
  (Soundness of cross-ADT tag collisions, e.g. two `Ok`/`Err` result types, is documented as a
  known limitation in Future Work — this doc only removes the randomness.)

**Fix D — Systemic guard:**

1. `internal/bytecode/compiler/determinism_test.go`: compile an ambiguity fixture (two record types
   sharing field names at **different sorted positions**, e.g. `Heading {x,y,z}` and
   `Ship {heading,x,y}` — `y` sits at index 2 vs 1 — exercised through both field access and record
   update with empty `KnownFields`) 200 times in one process; assert every compile yields identical
   serialized bytecode **and** the correct value. Go randomizes map range order per range call, so
   in-process repetition exposes the current bug with high probability (fixture style:
   `runProgram` in `collections_test.go:129` already builds programs with `TypeDecls` directly).
2. Map-range guard: reject `for … range` over map-typed expressions in
   `internal/bytecode/compiler/**` (go/parser-based test or a make target; mechanism is a deferred
   decision). `lowerRecordUpdate`'s existing `range e.Updates` (`internal/gen/lower/expr.go:468`) is
   order-insensitive downstream (fields are keyed by name in `compileRecordUpdate`'s `overrides`
   map) but should be sorted at lowering for tidiness — included as a low-cost task.
3. Parity fixture: add the game-shaped minimal repro (NDJSON decode of a nested `heading` object +
   colliding record types + a record update carrying the state forward) to the VM/interpreter
   parity corpus so the shape is continuously gated.

### Implementation Plan

**Phase 1: Determinism + correctness of record paths** (~1 day)
- [ ] Fix A: delete the map scan in `lookupFieldIndex`, always by-name on unresolved access
- [ ] Fix B lower half: `stmt.RecordUpdate.KnownFields` + `recordFieldSet` population
- [ ] Fix B compiler half: use `KnownFields`; remove the type-guess scan
- [ ] Fix C: ordered ADT iteration

**Phase 2: `OpUpdateRecord` runtime path** (~1 day)
- [ ] Opcode + VM handler + disasm + image serialization
- [ ] Compiler emission for unresolvable-base record updates

**Phase 3: Guards + fixtures** (~1 day)
- [ ] `determinism_test.go` (compile-idempotence + correct values)
- [ ] Map-range guard for the compiler package
- [ ] Parity fixture from the game shape
- [ ] Sort `e.Updates` at lowering (tidiness)

**Phase 4: Validation** (~1 day)
- [ ] Full `make test`, `make parity`, `make simplicity-audit` (new opcode touches the instruction
      surface gate), `make check-boundaries`
- [ ] Verify against the actual game repo repro (branch `m1.6-free-motion-iter1`, 25-run loop) if
      the rig can reach GitHub; otherwise the in-repo parity fixture is the acceptance proxy

### Files to Modify/Create

**New files:**
- `internal/bytecode/compiler/determinism_test.go` - compile-idempotence + correctness fixture (~150 LOC)
- `internal/bytecode/compiler/maprange_guard_test.go` (or `make` target) - static guard (~80 LOC)
- `test/parity/` fixture(s) - game-shaped nested-record NDJSON program (~40 LOC AILANG)

**Modified files:**
- `internal/bytecode/compiler/collections.go` - remove both map scans; `KnownFields` use (~60 LOC changed)
- `internal/bytecode/compiler/switch.go` + `compiler.go` - ordered ADT names (~25 LOC)
- `internal/gen/stmt/stmt.go` - `RecordUpdate.KnownFields` (~5 LOC)
- `internal/gen/lower/expr.go` - populate `KnownFields`; sort `e.Updates` (~15 LOC)
- `internal/bytecode/opcode.go`, `internal/vm/vm.go`, `internal/bytecode/disasm.go`, `internal/bytecode/image.go` - `OpUpdateRecord` (~70 LOC total)

## Examples

### Example 1: The wrong-slot read (before/after)

Given `type Heading = {x: int, y: int, z: int}` and `type Ship = {heading: Heading, x: int, y: int}`
and a field access `h.y` where `h` came from JSON decoding (`KnownFields` empty):

**Before:** `lookupFieldIndex` scans `recordTypes` in random order; when `Ship` comes first it
returns 2 (`y` in `[heading,x,y]`) — an in-range index into the 3-field Heading record — and
`GET_FIELD 2` silently reads `z`'s value. Output: wrong-but-`ok`, ~1 run in 4.

**After:** the compiler emits `BUILTIN_CALL _record_get(h, "y")`; the VM scans the record's own
fields by name and returns `y` — identical to the interpreter, in every run.

### Example 2: Record update on an unresolvable base

`{ship | pos: step(ship.pos, heading)}` where `ship` is a JSON-decoded value:

**Before:** the compiler picks whichever registered type whose fields include `pos` comes first in
map order and rebuilds the record in *that* type's shape — sometimes silently emitting a record
with the wrong field set.

**After:** if the base's static type is known, its (sorted) field list is used; otherwise
`OpUpdateRecord` copies the runtime base's fields and applies the override by name — the same thing
`evalCoreRecordUpdate` does.

## Success Criteria

- [ ] New compile-idempotence test: 200 compiles → 1 distinct bytecode image, correct values (AC: test green in CI)
- [ ] Parity fixture (game shape): VM output == interpreter output, byte-identical, 30/30 runs (AC: `make parity`-style loop in the test)
- [ ] Existing `multimodule_test.go` golden corpus unchanged or intentionally regenerated (AC: diff review)
- [ ] Map-range guard green on `internal/bytecode/compiler` (AC: guard test passes; introduce-map-range PRs fail CI)
- [ ] Game repro loop 30/30 stable when run against a rebuilt binary (AC: evaluator or rig run; in-repo fixture is the fallback)
- [ ] All tests passing (`make test-core`, `make test`)
- [ ] Documentation updated: `docs/LIMITATIONS.md` entry for the ADT tag-collision limitation; changelog entry
- [ ] No simplicity-audit regression (`make simplicity-audit` — instruction surface +1 opcode, documented)

## Testing Strategy

**Unit tests:**
- `determinism_test.go`: in-process compile-idempotence (200 iterations), field-access correctness
  through the unresolved path (expects the *named* field's value), record update on known and
  unknown base types, ADT inference determinism with two same-tag ADTs.
- `OpUpdateRecord` VM semantics: override applied, untouched fields preserved in order, duplicate
  override names panic via `NewRecord`, empty-override update is a copy.

**Integration tests:**
- Parity fixture run through both VM and interpreter; assert byte-identical stdout on an NDJSON
  input containing a nested `heading` object.
- Golden corpus (multimodule, pattern tests) — must not change unexpectedly; any intentional
  bytecode diff (by-name fallbacks replacing index guesses) is regenerated and reviewed.

**Manual testing:**
- The reported 25-run loop on the game branch, if the rig can clone GitHub repos.

## Deferred Decisions

- Guard mechanism: go/parser-based test vs `make` grep target vs both — agent may choose.
- Exact `OpUpdateRecord` encoding details (register layout nuances) — agent may choose within the
  existing pseudo-LOAD_CONST pattern.
- Whether the map-range guard later extends to `internal/gen/` and `internal/eval/` — separate
  follow-up; this doc scopes it to the compiler package.

## Non-Goals

**Not attempted in this feature:**
- Cross-ADT tag-collision **soundness** (two ADTs sharing tag names → deterministic-but-possibly-wrong
  switch dispatch). This doc removes the randomness only; the soundness fix (tag identity at
  runtime) is Future Work.
- Whole-pipeline map-order audit (iface digests already fixed at 282c02315; broader sweep deferred).
- Performance work on the by-name fallback path (O(n) field scan for unresolved types — same cost
  as today's `_record_get` fallback; if it shows up in benchmarks, a follow-up can cache field
  layouts per prototype).
- Any change to the interpreter.

## Timeline

**Week 1** (~4 days):
- Days 1–2: Phase 1 + Phase 2
- Day 3: Phase 3 guards and fixtures
- Day 4: Phase 4 validation, golden-corpus diff review, changelog

**Total: ~4 days across 1 week**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-------------|
| By-name fallback slows hot paths (JSON-heavy programs) | Med | It only applies where static resolution fails — unchanged from today's `_record_get` fallback frequency; benchmark before/after with `ailang eval-elo`-style timing on the fixture |
| Golden-corpus bytecode churn from by-name lowering breaks downstream consumers (compile-cache, replay) | Med | Corpus diffs reviewed line-by-line; images are recompiled per run (no persistent bytecode cache), so no stored-image migration is needed |
| New opcode breaks image serializer/deserializer round-trip | Med | `image.go` case added alongside `OpMakeRecord`; round-trip unit test with `disasm` |
| An unresolvable record update currently "works" via a lucky guess in a program where the guess is actually right; runtime path changes its observable behavior | Low | `OpUpdateRecord` mirrors `evalCoreRecordUpdate` exactly, so parity is the acceptance test — if interpreter and VM agree, the change is correct by definition |
| 200-iteration test is flaky-slow | Low | Iterations bounded and parallelizable; measured runtime target < 2s |

## Related Documents

**Implemented (may inform design):**
- [m-bytecode-multimodule-sprint-plan.md](../implemented/v0_11_0/m-bytecode-multimodule-sprint-plan.md) — M3 introduced the `_record_get` by-name fallback and `KnownFields` hint this fix generalizes
- [m-bytecode-vm.md](../implemented/v0_11_0/m-bytecode-vm.md) — VM/opcode architecture the new `OpUpdateRecord` extends

**Planned (check for overlap):**
- [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md) — different VM parity bugs (effect-replay classification, #505 pattern arity, #506 unsafe replay); no overlap with compile-time determinism, but the companion "strict GET_FIELD wrong-slot" report and this doc share the root cause fixed here
- [m-perf4-bytecode-interpreter.md](../v1_1_0/m-perf4-bytecode-interpreter.md) — SimHash keyword-adjacent (0.90) but a P3 perf stretch goal, not a determinism fix; no overlap
- [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) — #505 spin-out; distinct soundness bug

## References

- [Design Axioms](/docs/references/axioms) — A1 determinism is the violated principle
- Bug report: `--bytecode VM nondeterministic` (this task) + companion strict GET_FIELD field-slot report — same root cause
- Precedent of the pattern class: iface digest map-order fix 282c02315 ("print record fields in sorted order", Daneel 2026-09-28)
- Go spec: map iteration order is not specified and is randomized per iteration

## Verification Log

Every load-bearing claim above was checked against the code at commit 198d38d1 (working tree), not
asserted from memory:

| # | Claim | Evidence |
|---|-------|----------|
| 1 | `lookupFieldIndex` fallback ranges over the Go map `fc.recordTypes` and returns the first matching type's index | Read `internal/bytecode/compiler/collections.go:258-283` (`for _, info := range fc.recordTypes`) |
| 2 | `compileRecordUpdate` picks the record type by first-match over the same map | Read `collections.go:291-303` (`for tn, ti := range fc.recordTypes`) |
| 3 | `inferADTFromCases` uses first-match over `fc.adtTypes` | Read `internal/bytecode/compiler/switch.go:139-155` |
| 4 | `recordTypes`/`adtTypes` are Go maps populated in source order at compile time | Read `internal/bytecode/compiler/compiler.go:47-68,119-120,202-203` |
| 5 | VM `GET_FIELD` bounds-checks only — an in-range wrong index silently returns another field's value | Read `internal/vm/vm.go:443-465` (no name/tag check on the record path) |
| 6 | A by-name runtime fallback already exists (`_record_get` → `compileFieldAccessByName`) | Read `collections.go:225-263`, `internal/vm/builtins.go:143-165` |
| 7 | Record values store fields alphabetically sorted; `NewRecord` sorts and panics on duplicates | Read `internal/bytecode/value.go:267-280,316-318` |
| 8 | The interpreter resolves field access by name on the runtime record (hence stable) | Read `internal/eval/eval_expressions.go:438-465` |
| 9 | The lower pass computes `KnownFields` for `FieldAccess` from the static type, sorted | Read `internal/gen/lower/expr.go:70-80,244-280` (`recordFieldSet` ends with `sort.Strings`) |
| 10 | `stmt.RecordUpdate` has NO `KnownFields` today (the asymmetry this fix closes) | Read `internal/gen/stmt/stmt.go` RecordUpdate struct; `lowerRecordUpdate` at `internal/gen/lower/expr.go:464-477` attaches none |
| 11 | `lowerRecordUpdate` ranges over `e.Updates` (map) but is order-insensitive downstream (name-keyed `overrides`) | Read `lower/expr.go:465-477` + `collections.go:311-313` |
| 12 | `ailang run` recompiles per run (no persistent bytecode image between runs) — per-run map-order sampling | Read `internal/pipeline/cache_alias_digest_stable_test.go:10-16` comment ("`ailang run` recompiled every time (Daneel, 2026-09-28)"); compile-cache digests source, not bytecode images |
| 13 | Test helpers exist to build programs with `TypeDecls` + record updates directly (fixture style for the determinism test) | Read `internal/bytecode/compiler/compiler_test.go:13` (`runCompiled`), `collections_test.go:129-190` (`TestCompile_RecordUpdate` builds a full `stmt.Program`) |
| 14 | No existing design doc covers this topic (duplicate/coverage gate) | `ailang docs search` on "vm determinism" / "bytecode determinism record field order": no neural matches; SimHash top hits reviewed and are distinct (perf stretch goal, parity harness lanes, #505/#506) |
| 15 | `image.go` has an opcode-class switch that must learn `OpUpdateRecord` | Read `internal/bytecode/image.go:329` (`case OpMakeList, OpMakeTuple, OpMakeRecord, OpMakeADT:`) |
| 16 | MAKE_RECORD's pseudo-LOAD_CONST field-name pattern is the established encoding the new opcode mirrors | Read `internal/vm/vm.go:405-433` + `collections.go:150-183` |

Language-surface note: this fix changes **no AILANG syntax** — all verification above is Go-code
reading plus `ailang docs search`; no `ailang check` claims about language constructs are made in
this doc (none were needed).

## Future Work

- Cross-ADT tag-collision soundness: switch dispatch on tag *identity* rather than per-type ordinals
  (would also fix deterministic-wrong dispatch when two ADTs share tag names). Needs its own design
  doc — touches pattern matching semantics.
- Pipeline-wide map-order audit/guard (`internal/gen`, `internal/eval`, image digests) once the
  compiler-package guard lands.
- Field-layout caching for the by-name fallback if benchmarks justify it.

---

**Document created**: 2026-09-28
**Last updated**: 2026-09-28
