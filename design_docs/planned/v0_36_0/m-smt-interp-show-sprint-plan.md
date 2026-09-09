# Sprint Plan — M-SMT-INTERP-SHOW

## Summary

Add a type-directed Core→Core `ShowNormalizer` pass so that string-building functions
verify under Z3 without any source change, then teach the SMT encoder the one remaining
conversion (`int`) and make the residual skip message honest.

**Design doc**: [m-smt-interp-show.md](m-smt-interp-show.md) (r3, both freeze items ratified 2026-09-09)
**Duration**: 2–2.5 days (~17h)
**Dependencies**: None. Freeze items granted — pipeline-wide placement, Z3 `str.from_int` dependence.
**Risk Level**: Medium — the pass runs on every compiled program; mitigated by a
value-equivalence suite and a whole-corpus output diff.

## Current Status Analysis

### Verified before planning (design doc §Verification Log, 21 rows)
- `concat_String` already encodes to `str.++`; hand-folded chains verify **today** (V4)
- The `bool` rewrite target verifies with **zero** verifier changes (V5)
- The `int` encoding was proved exact against Z3 4.15.4; the naive form is wrong (V7)
- `_string_intToStr` is a `$builtin` ref needing no import (V17, V18)
- `ValidateCoreTypeInfo` requires an entry for **every** node, `Lit` included (V19)

### Velocity
Recent comparable single-pass work (`m-std-base64url-encode`, 2026-09-08) landed
implementation + gates in one day. `DebugEraser` (252 LOC) is the structural template for M1,
so the pass is a known shape rather than a new one. Target ~150 LOC/day of production code
plus tests.

### Remaining from design doc
- ⏳ M1: `ShowNormalizer` — string elision + bool rewrite (~6h)
- ⏳ M2: SMT encoding for `_string_intToStr` + the `int` rewrite row (~4h)
- ⏳ M3: honest residue diagnostic via a `DeclMeta` note (~4h)
- ⏳ M4: gates, docs, changelog, feedback reply (~3h)

## Proposed Milestones

### M1 — `ShowNormalizer`: string elision + bool rewrite
**Goal**: the reporter's minimal pair goes green; `safeConcat` and `prefixedLength` verify.
**Estimated**: ~180 impl + ~250 test = ~430 LOC · ~6h

**Tasks:**
- `internal/pipeline/show_normalize.go` — walk mirroring `DebugEraser.eraseExpr`, taking
  `*types.CoreTypeInfo` + an ID allocator
- `string` row: replace `App($builtin.show,[x])` with `x`
- `bool` row: mint `core.If{Cond:x, Then:Lit "true", Else:Lit "false"}`, registering a
  `CoreTypeInfo` entry for **each** minted node (`If`, both `Lit`s) — never for the reused `x`
- Missing `CoreTypeInfo` entry ⇒ **hard error** naming function, node ID and pass
  (a present-but-unsupported type leaves `show` alone — the two are not the same case)
- Wire into `pipeline_module_compile.go` and `pipeline_single.go` after `Specialize`,
  unconditionally (so the `DisableMonomorphization` path still gets it)
- Delete the false `show_Int`/`show_String≡id` comment at `parser_literals.go:103-105`
- Tests: per-row Core shape, minted-node type registration, hard error on a missing entry,
  nested `show(show(x))`, and the value-equivalence table (empty, embedded `"`, embedded `\n`,
  100-char string, `0`, `-5`, `MinInt`, `true`/`false`)

**Acceptance:**
- [ ] `withInterp` VERIFIED; `noInterp` still VERIFIED
- [ ] `safeConcat` (string_verify.ail) and `prefixedLength` (showcase.ail) VERIFIED
- [ ] `${b}` (bool) hole verifies
- [ ] `make verify-examples` — zero output diffs
- [ ] `internal/format/interp_test.go` byte-identical; `#386` tests green **including** its
      must-reject controls
- [ ] `make test`, `make lint`, `make check-file-sizes` clean

**Risks:** `show(s:string)` not exactly identity → mitigated by the value-equivalence table
plus the whole-corpus diff. Minted nodes missing a `CoreTypeInfo` entry → caught by
`ValidateCoreTypeInfo`, and asserted directly in unit tests.

### M2 — SMT encoding for `_string_intToStr` + the `int` rewrite row
**Goal**: `${n}` (int) holes verify, with an encoding proved exact rather than plausible.
**Estimated**: ~40 impl + ~120 test = ~160 LOC · ~4h

**Tasks:**
- `StringBuiltinSpecial["_string_intToStr"] = {Op:"str.from_int", IntToStrMode:true}`
  in `internal/smt/types.go`; emit
  `(ite (>= n 0) (str.from_int n) (str.++ "-" (str.from_int (- n))))` in `codegen_apps.go`
- `int` row in `ShowNormalizer`: emit the **application** —
  `core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module:"$builtin", Name:"_string_intToStr"}}, Args: []core.CoreExpr{x}}`
  — registering `string` for the `App` and `int -> string` for the `VarGlobal`
- Bonus: `StdlibStringToSMT["intToStr"] = "_string_intToStr"` so a user-written
  `import std/string (intToStr)` verifies too
- Tests: Z3 exactness (assert `unsat` on the negation of `showInt(n) = strconv.Itoa(n)` over a
  sampled range, plus symbolic `str.len(showInt(n)) > 0`); a no-import regression fixture

**Acceptance:**
- [ ] `${n}` verifies for positive, negative and zero
- [ ] A module using `${n}` with **no** `import std/string` compiles, links and runs
- [ ] Exactness assertions in CI, so a solver upgrade that breaks them fails loudly

### M3 — honest residue diagnostic
**Goal**: the surviving skip names the measured argument **type** and never claims an origin.
**Estimated**: ~30 impl + ~80 test = ~110 LOC · ~4h

**Tasks:**
- `ShowResidueNote` + `ShowResidue []ShowResidueNote` on `core.DeclMeta`
- `ShowNormalizer` records one note per `show` it deliberately leaves, keyed to the enclosing
  function (`LetRec.Bindings[].Name` / `Let.Name`)
- `encodable.go`: when the blocker is `show` and a note exists, name the type; put the
  interpolation fact in the **Hint** as conditional advice; fall back to today's type-free
  message when no note exists
- Tests: float hole message; **explicit** `show(p:float)` call gets the same message and is
  not misdiagnosed; no-note fallback

**Acceptance:**
- [ ] `${f}` (float) skips with the type named and no origin claimed
- [ ] An explicit `show(p: float)` call is not misdiagnosed as interpolation

### M4 — gates, docs, close the loop
**Estimated**: ~3h
- [ ] `CHANGELOG.md` entry; remove the "string building cannot be verified" limitation from
      `docs/LIMITATIONS.md`
- [ ] `examples/runnable/contracts/` — update any comment documenting the old skip
- [ ] New example exercising a verified string builder (per the project example rule)
- [ ] Reply to `fb_913ee851c83c0d8c` with the diagnosis and what shipped
- [ ] Move design doc to `design_docs/implemented/v0_36_0/` with an implementation report

## Success Metrics
- `show` skips across `examples/runnable/contracts/` — **2 → 0**
- `make verify-examples` — zero output diffs
- `make test` / `make lint` / `make check-file-sizes` — clean
- All new files under the 800-line AI-maintainability gate

## Open Questions
None. Both freeze items ratified 2026-09-09; the three Deferred Decisions in the design doc
are explicitly agent-resolvable.
