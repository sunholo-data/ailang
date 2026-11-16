# Strict-VM coverage census — M-BYTECODE-STRICT-256-REG-CEILING

**Measured**: 2026-11-09, v0.57.4-75 (`origin/dev` eb5af29be plus the Phase A branch), with
`bin/ailang` built from the checkout. Phase B1 and B2 had already landed, so this is the
**post-B** census. No pre-B baseline was banked: the sprint's M1 census never ran
before B shipped, and rebuilding a v0.54.12 binary only to count a fixed cause was not
worth it. The two B acceptance tests (`cmd/ailang/bytecode_long_list_literal_test.go`)
are the evidence for the before/after on the reported shapes.

## Method

The instrument is the Phase A gate itself, plus `disasm` for the image-level count.
Nothing is executed.

- **Files**: `git ls-files 'std/*.ail' 'examples/*.ail'` (recursive), plus `goldens/`.
  `goldens/` holds only JSON fixtures, with 0 `.ail` files.
- **Per file**: `ailang check --strict-bytecode --json --relax-modules <f>`. Each BC001
  entry is an evaluator-only function reachable from the file's own declarations.
  `ailang disasm --relax-modules <f>` gives every prototype in the compiled image,
  including imported prototypes that nothing reaches.
- **Deduplication**: functions are counted once by canonical name across files, because
  `std/*` modules are compiled into many images. The denominator is unique prototype
  names.
- **Exclusions**: a file whose plain check fails is excluded. These are negative,
  archived and expected-fail examples. None of the 63 exclusions came from the bytecode
  step; each fails type, effect, parse or module checking first.

## Result

| Measure | Count |
|---|---|
| `.ail` files attempted (51 `std/`, 550 `examples/`, 0 `goldens/`) | 601 |
| Excluded (plain check fails) | 63 |
| Checked | 538 |
| Checked, no BC001 | 408 |
| Checked, at least one BC001 | 130 |
| Unique prototypes across all images | 3295 |
| Unique evaluator-only prototypes (image level) | 232 |
| Unique BC001 functions (reachable from some file's own declarations) | 232 |

The two 232s coincide because every `std/` module is also checked as a file in its own
right. Through `std/list.ail`, for example, its own `takeMap` is a root. In a
*program* that only imports `std/list`, `takeMap` is unreachable and is not reported.
That is the case `TestCheckStrictBytecodeReportsBC001` pins.

### BC001 by cause

| Cause (compiler reason) | `std/` | `examples/` | Total |
|---|---|---|---|
| **Register ceiling** (`contiguous block of N would exceed 256`): the two causes this design targets | 0 | 0 | **0** |
| Effectful builtin not wired (`effectful builtin "…" not yet wired (Phase 2E)`) | 127 | 6 | 133 |
| Bridge-unsafe caller (`references evaluator-only X, whose … cannot cross the VM/evaluator bridge`) | 11 | 51 | 62 |
| Budget or mode frame the VM cannot push (`IO @limit=N`, `Rand[mode=…]`, `Net[scope=…]`) | 0 | 22 | 22 |
| Pure builtin with no VM implementation (closure-arg HOFs such as `takeMap`) | 11 | 0 | 11 |
| `unknown ADT type` (contract-module ADTs: `examples/runnable/contracts/{invoice,park,scoring}.main`) | 0 | 3 | 3 |
| `unsupported expression stmt.ArrayLit` (`examples/runnable/array_adt.main`) | 0 | 1 | 1 |
| **Total** | 149 | 83 | **232** |

## Reading

- **The targeted cause is at zero.** No file in the corpus hits the 256-register ceiling
  after B1 (chunked list literals) and B2 (else-if chain reuse). The two shapes from the
  original report are covered by tests: 601-element literals and an 80-arm chain over a
  16-field record. A record or tuple literal with more than 255 fields is still
  evaluator-only by design (Non-Goals). The corpus has none. The Phase A fixture builds
  one on purpose (`cmd/ailang/check_strict_bytecode_test.go`).
- **The residual is VM coverage work, not register pressure.** 133 functions are
  unwired effectful builtins and 62 are callers that pass a value the bridge cannot
  carry. These match the "unwired effectful builtins, contract-module ADTs, and user
  ADTs or closures passed to evaluator-only functions" already listed in
  `docs/LIMITATIONS.md` and planned in
  [M-VM-COVERAGE](../v0_55_0/m-vm-coverage-letrec-prelude-bridge.md). Each is now named
  with a position by BC001, so nothing new needs filing here.
- **Unreachable imported helpers are excluded on purpose.** The sprint plan proposed
  gating on every prototype in the image. The census shows why that would not work:
  `std/list` alone contributes two evaluator-only prototypes to every image that imports
  it, so the strict check would fail almost every real program. Phase A reports what can
  actually trap: everything the file declares, plus whatever that reaches.
