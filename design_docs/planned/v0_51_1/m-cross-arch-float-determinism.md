# M-CROSS-ARCH-FLOAT-DETERMINISM: std/math transcendental results (exp, log, sin, …) differ by 1 ulp between arm64 and x86_64 — all three backends delegate to host-Go `math`, which is not bit-stable across GOARCH

**Status**: PLANNED
**Target**: v0.51.1
**Priority**: P1 — direct violation of AILANG's core determinism promise ("deterministic: all non-determinism explicit", CLAUDE.md): a *pure* builtin returns architecture-conditional bits. Not P0: within one arch nothing is corrupted, VM and interpreter agree per-arch, and consumers carry a per-arch-goldens workaround.
**Estimated**: 3–4 days (one sprint; ~1,700 LOC incl. vendored algorithms, golden-bit tables and tests)
**Dependencies**: None. Self-contained to the three `std/math` delegation surfaces in `internal/builtins`, `internal/vm`, `internal/gen/golang`.
**Author**: Claude (design-doc-creator, coordinator task `task-5077e633`, 2026-10-01). Report source: v0.50.0 release report (binary `v0.50.0-6-g021c46907-dirty`, md5 `caeabea…`); reporter machines darwin/linux arm64 + linux x86_64; consumer impact carried by stapledons-godot per-arch replay goldens. All in-container repro commands were run with `ailang` v0.50.1 @ `021c469` (same commit as the report binary) on **x86_64** — this container has no arm64 box, no Go toolchain and no `make` binary, so arm64-side and Go-internal claims are either the reporter's measurements or spec citations, each marked as such in the Verification Log.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes implicit architecture-conditional nondeterminism from *pure* builtins: after the fix, `exp(0.2064590551107192)` produces the same bits on every GOARCH/GOOS, in interpreter, VM and compiled-Go mode. This is the axiom the report is enforcing. |
| A2: Replayability | +1 | A seed's simulated output becomes byte-identical on every machine (stapledons-godot replays); per-arch goldens collapse back to one golden set. |
| A3: Effect Legibility | 0 | No effect changes — all affected builtins stay `pure`. |
| A4: Explicit Authority | 0 | No capability surface touched. |
| A5: Bounded Verification | 0 | No type-system change. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | Replaces "whatever the host CPU's libm does" — a property an AI cannot check — with a machine-checkable law: same input → same bits everywhere, pinned by committed golden-bit tables and a cross-arch CI gate. |
| A8: Minimal Syntax | 0 | No syntax change. |
| A9: Cost Visibility | 0 | No resource-cost change (perf note in Risks: amd64 already runs this algorithm class). |
| A10: Composability | 0 | Same API surface, same signatures. |
| A11: Structured Failure | 0 | No new failure modes; special values (NaN/±Inf/±0) keep fdlibm semantics. |
| A12: System Boundary | +1 | The language stops leaking a host-CPU property (FMA contraction / GOARCH-specific libm) through its most basic numeric boundary. |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): the fix *removes* implicit nondeterminism (architecture-conditional last-bit results).
- [x] A3 (Effects): no hidden side effects introduced.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): the determinism law becomes machine-checkable, not human-annotated.

---

## Problem Statement

`std/math`'s transcendental functions are thin AILANG wrappers (`std/math.ail:46` `exp(x) = _math_exp(x)`, `:49` `log(x) = _math_log(x)`) whose implementations on **all three execution surfaces** are the host Go `math` package (VL-4, VL-5). Go does not guarantee cross-GOARCH bit-stability for `math` results, and on arm64 the results measurably differ from x86_64:

### Reported (v0.50.0 release, reporter's machines; x86_64 column independently reproduced here — VL-1)

| Expression | darwin/linux **arm64** | linux **x86_64** | Reference |
|---|---|---|---|
| `exp(0.2064590551107192)` | `1.229317398921793` | `1.2293173989217931` (VL-1) | True value `1.22931739892179304107…`; x86_64 = glibc = Python `math.exp` → **correctly rounded**; arm64 is **1 ulp off** |
| `log(exp(0.2064590551107192))` | `0.2064590551107191` | `0.20645905511071927` (VL-1) | Divergence propagates downstream from `exp`'s differing output |
| `log(1.229317398921793)` (identical input both arches) | `0.2064590551107191` | `0.2064590551107191` (VL-3) | `log` agrees for identical input — the *source* is `exp`; downstream diffs are value propagation |

The VM and the interpreter agree on each arch (reporter's measurement; verified here on x86_64 for both paths — VL-2). This is **not** a backend-parity bug: it is the *host* leaking through all backends equally.

Because `show` renders floats as shortest-round-trip text (`internal/eval/show.go` `showFloat`, `strconv.FormatFloat(f, 'f', -1, 64)` — VL-7), a 1-ulp difference changes the **last printed digit** — so committed text goldens diverge byte-wise even though "the program is correct".

**Impact**: stapledons-godot requires byte-identical simulation output for a seed (replays). CI is x86_64, the developer's machine is arm64, so committed replay goldens diverge; the project currently carries **per-arch goldens as a workaround**. Any consumer running `exp`/`log`-derived physics across machines hits the same class. Determinism across platforms is a core VM promise (A1), so this is a language defect, not a consumer-tooling quirk.

### Root cause

Three delegation surfaces, all calling host Go `math` (exact lines, VL-4):

| Surface | File:line | exp | log |
|---|---|---|---|
| Interpreter builtins (live `std/math` path) | `internal/builtins/math_trig.go:121` / `:132` | `math.Exp` | `math.Log` |
| Bytecode VM (`--strict-bytecode`) | `internal/vm/builtins_math.go:36` / `:39` | `math.Exp` | `math.Log` |
| Compiled-Go codegen | `internal/builtins/registry_codegen_math.go:16–18` (Inline templates `math.Exp({{arg0}}.(float64))`); fallback map `internal/gen/golang/codegen_expr_simple.go:313–320` used from `codegen_expr_app.go:24` | emits `math.Exp(x.(float64))` into the generated program | same |

The js/wasm target (`cmd/wasm/main.go:13` imports `internal/eval`) reaches the same builtins — it inherits whatever the host Go runtime does and will inherit the fix too.

**Why host Go diverges across GOARCH.** Two documented Go behaviors, either of which suffices; which one applies to a given function is resolved empirically by the Phase-0 audit (the fix is immune to both):

1. **Compiler FMA contraction.** The Go spec ("Floating-point operators") permits an implementation to fuse `x*y + z` into a single FMA on FMA-capable GOARCH (`arm64`, `ppc64`, `s390x`, `riscv64`) — with a *different* (once-rounded) result than executing the multiply and add separately. Go's `math` algorithms are fdlibm/BSD-derived and written for individually-rounded IEEE 754 double operations. amd64 baseline codegen has no FMA, so it never fuses. The spec's own remedy for reproducible float code is to force rounding with explicit conversions — `float64(x*y) + z`.
2. **GOARCH-specific implementations inside Go's `math` package** (assembly paths on arm64 written against scalar FMA instructions), which can differ in the last ulp from the pure-Go path amd64 executes.

Go makes no cross-GOARCH bit-equality promise for `math` functions — so a language whose core promise is determinism must not delegate its float semantics to the host CPU's implementation choices.

---

## Goals

**Primary Goal:** every `std/math` transcendental function returns bit-identical `float64` results on every supported GOARCH/GOOS (linux/darwin/windows × amd64/arm64, plus js/wasm), across interpreter, `--strict-bytecode`, and compiled-Go modes — equal to today's x86_64 values.

**Success Metrics:**
1. `exp(0.2064590551107192)` → `1.2293173989217931` on arm64 **and** x86_64, on both execution paths (the report's case, byte-identical `show` text).
2. A committed golden-bit table per function (~200 boundary/special values + ≥10,000 sweep inputs, `math.Float64bits`-keyed) is asserted by unit tests that are arch-independent by construction — every arch asserts the *same* bits.
3. Zero delta between the portable implementation and current amd64 host `math` over the full sweep (expected; if nonzero, D1 re-escalates — see Risks).
4. Differential test: interpreter vs `--strict-bytecode` vs compiled-Go byte-identical over the golden table (compiled arm needs a Go toolchain — CI has one).
5. No churn on x86_64: existing examples, tests and goldens produced on x86_64 keep their current output; `internal/builtins/math_test.go` `TestExpImpl`/`TestLogImpl` pass unchanged (they use `InDelta(1e-10)` — VL-13).
6. stapledons-godot's per-arch goldens collapse back to a single golden set (consumer action at release time).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Canonical values = **today's amd64 host-Go results** (fdlibm-class, glibc/Python-compatible for `exp` at the reported input) — *not* a correctly-rounded (CRlibm-class) reimplementation | Zero churn on CI/x86_64 goldens; arm64 shifts by ≤1 ulp on affected inputs; a CRlibm port would change values on **every** arch including CI goldens, for no consumer-stated need | **human** — must sign off on the arm64 golden-shift consequence (arm64-generated goldens regenerate; x86_64 untouched) | design | low |
| D2: Implementation = **vendored pure-Go (fdlibm/BSD-derived) algorithms in a new leaf package, with every product wrapped in `float64(...)`** to forbid FMA contraction (the Go-spec-documented remedy) | Only strategy that is simultaneously portable, amd64-value-compatible and immune to *both* root-cause mechanisms (contraction and GOARCH-specific assembly) — we stop calling host `math` for the transcendental surface | agent | design | med |
| D3: Host = new leaf package `internal/mathx` (interpreter + VM swap one token each); **compiled codegen emits the same algorithms inline** as helpers (`__ail_*`), keeping generated programs stdlib-import-only (the compile-verify flow writes a dependency-free temp `go.mod`, `cmd/ailang/compile.go:516–526` — VL-11) | Keeps compiled artifacts self-contained; dual copy of the algorithm guarded by a CI differential test | agent | design | med |
| D4: Scope = the **11 transcendental functions** `sin cos tan asin acos atan atan2 pow exp log log10`; `sqrt` (IEEE 754 correctly-rounded, hardware-mandated identical), `floor/ceil/round/abs_Float` (exact bit operations with a unique correct result) stay on host `math` | No divergence is *possible* for the exact/IEEE-mandated functions; porting them would add churn without adding determinism | agent | design | low |
| D5: **Structural guard**: new CI test job on `ubuntu-24.04-arm` running the `mathx` golden-bit tests (+ `make test-core`) | Today CI never runs Go tests on arm64 (the only macos-latest job is bash-3.2 shell tests — VL-10; release `build.yml` cross-builds an arm64 binary without testing it) so this class regresses silently; with the guard it fails CI, not a replay diff | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1** — human confirms amd64-compat canonical values and the resulting arm64 ≤1-ulp golden shift (default recommendation of this doc).

All remaining decisions (D2–D5) are agent-chosen and frozen in this doc.

---

## Solution Design

### Overview

One portable math core, applied at all three surfaces, plus a cross-arch CI gate:

1. **`internal/mathx`** — vendored, fusion-proof pure-Go implementations of the 11 transcendental functions, with committed golden-bit tables and arch-independent tests.
2. **Interpreter + VM** — swap the `func(float64) float64` pointers (`math.Exp` → `mathx.Exp`, etc.) at `internal/builtins/math_trig.go` and `internal/vm/builtins_math.go`.
3. **Compiled-Go codegen** — registry Inline templates and the `mathFunctions` fallback map emit calls to lazily-emitted `__ail_*` helpers carrying the same algorithms; `codegen_math_test.go` text expectations updated intentionally.
4. **CI** — arm64 job asserting the same golden bits, so cross-arch divergence can never land silently again.

### Fusion-proofing — exact specification (D2)

- Every multiplication whose result feeds an addition or subtraction is wrapped in an explicit conversion: `x*y + z` → `float64(x*y) + z` (Go spec: an explicit conversion forces the intermediate rounding; parentheses alone do **not**).
- Untyped constant arithmetic is untouched — Go constant expressions are exact rationals rounded once at use; the algorithm's coefficient tables (`LnHi`, `LnLo`, `Sqrt2`…) stay verbatim.
- Special-value branches (±0, ±Inf, NaN, subnormals, argument reduction boundaries) are copied verbatim from Go's pure-Go sources — Go already pins their behavior in its own tests, and we inherit those guarantees by not editing them.
- Vendored files keep Go's BSD license attribution header (Go stdlib source is BSD-3-clause; compatible with this repo's existing third_party practice — executor confirms the exact notice form).

**Canonical-value definition:** `mathx` output = the value Go's pure-Go (fdlibm/BSD) algorithm produces under individually-rounded IEEE 754 double operations — i.e. today's amd64 output (D1). Phase 0 measures this claim; any amd64 delta found escalates D1 before wiring begins.

### Implementation Plan

**Phase 0 — cross-arch audit + golden tables (~0.5 day)**
- [ ] Cross-arch audit probe (a `go run`-able helper or test, executor picks the form): for each of the 11 functions, sweep boundary/special values + ≥10k random inputs, print `math.Float64bits` of host `math.f(x)`; run on x86_64 (CI/container) and arm64 (developer machine or the new D5 runner). Produce the divergence census (which host functions diverge on arm64 today — expected: at least `exp`; `log` unconfirmed by identical-input evidence — VL-3).
- [ ] Sweep `internal/{eval,vm,builtins}` for FMA-fusable multi-op float expressions (`x*y + z` in one Go expression). A crude grep found none (VL-8); the audit re-verifies on arm64 by running the VM arithmetic suite there (D5 job).
- [ ] Commit the golden-bit tables generated **from the x86_64 host** as the canonical reference (D1); record the arm64 delta table in this doc's implementation report.

**Phase 1 — `internal/mathx` + tests (~1 day)**
- [ ] Vendor the 11 pure-Go algorithm files (`exp.go`, `log.go`, `log10.go`, `sin.go`, `cos.go`, `tan.go`, `asin.go`, `acos.go`, `atan.go`, `atan2.go`, `pow.go`; `log10`/`atan2` may share internals with `log`/`atan` as Go's sources do), apply the fusion-proof transform, keep the BSD header.
- [ ] `mathx_test.go`: golden-bit assertions over the Phase-0 tables (bits, not `==`, so ±0/NaN payloads are caught); special-value table (±0, ±Inf, NaN, subnormal min/max, reduction boundaries like `exp(709.78)`, `log(1e-308)`).
- [ ] Assert the report's case: `mathx.Exp(0.2064590551107192)` bits == bits of `1.2293173989217931`.

**Phase 2 — wire the three surfaces (~1 day)**
- [ ] `internal/builtins/math_trig.go`: swap the 11 `math.X` arguments of `registerTrigFunc`/`registerTrigFunc2` to `mathx.X` (metadata, types, signatures untouched — the wrappers are pure delegation).
- [ ] `internal/vm/builtins_math.go`: same 11 swaps in `builtinMath*`.
- [ ] `internal/builtins/registry_codegen_math.go`: Inline templates `math.Exp({{arg0}}.(float64))` → `__ail_exp({{arg0}}.(float64))` etc.; imports move from `"math"` to the emitted-helper scheme.
- [ ] `internal/gen/golang`: new `writeMathHelpers()` (added to `writeRuntimeHelpers()`, `codegen_runtime.go`) emitting the `__ail_*` bodies lazily (only when a math function is referenced — mirror the existing lazy-emission pattern in `writeRegistryHelpers`); `mathFunctions` map values (`codegen_expr_simple.go:297–322`) → `__ail_*`; handle `skipRuntimeHelpers` mode (multi-file compilation) so helpers are still emitted when math is referenced (mirror the `needsMathImport` logic, `codegen.go:562–574`).
- [ ] `internal/gen/golang/codegen_math_test.go`: update the text assertions `math.Sin` → `__ail_sin` etc. (rows at :24, :35–37, :186–188, :289–291 — **intentional change**, listed in Conflict Surface).
- [ ] Differential test (CI, where Go exists): compile a generated program calling all 11 functions via `ailang compile`-style codegen; assert outputs bit-equal to `mathx` over the golden table; plus interpreter and `--strict-bytecode` over the same table from `.ail`.

**Phase 3 — guards, docs, goldens (~0.5–1 day)**
- [ ] `.github/workflows/ci.yml`: `ubuntu-24.04-arm` job (math path filter like the bash-3.2 job) running `go test ./internal/mathx/... ./internal/vm/... ./internal/builtins/...` + `make test-core`.
- [ ] `docs/LIMITATIONS.md`: add the compiled-mode user-arithmetic caveat (see Non-Goals) — today the file has no cross-arch float entry (VL-14).
- [ ] CHANGELOG entry at ship time (breaking-ish: arm64 last-ulp shifts, goldens regenerate).
- [ ] Consumer coordination: flag to stapledons-godot that per-arch goldens can collapse post-upgrade.
- [ ] `make test`, `make check-boundaries`, `make simplicity-audit` green (new internal package is core-layer, imports nothing above it — no boundary risk).

### Files to Modify/Create

**New files:**
- `internal/mathx/{exp,log,log10,sin,cos,tan,asin,acos,atan,atan2,pow}.go` — ~700 LOC vendored + fusion-proofed
- `internal/mathx/mathx_test.go` — ~250 LOC golden-bit tests
- `internal/mathx/golden/*.json` (or generated Go tables — executor's choice) — canonical bits, ~150 KB
- `internal/mathx/probe/main.go` — ~120 LOC cross-arch audit probe
- `internal/gen/golang/codegen_math_helpers.go` — emitted-helper templates carrying the same algorithms (~700 LOC; dual copy, guarded by the CI differential test)

**Modified files:**
- `internal/builtins/math_trig.go` — 11 one-token swaps
- `internal/vm/builtins_math.go` — 11 one-token swaps
- `internal/builtins/registry_codegen_math.go` — 11 Inline templates + import
- `internal/gen/golang/codegen_expr_simple.go` — `mathFunctions` map values
- `internal/gen/golang/codegen_runtime.go` — hook `writeMathHelpers()`
- `internal/gen/golang/codegen_math_test.go` — intentional expectation updates
- `.github/workflows/ci.yml` — arm64 job (~15 LOC)
- `docs/LIMITATIONS.md` — 1 entry

No `std/math.ail` change — `exp`/`log` are pure delegation (VL-5). No parser/typechecker/effect changes.

---

## Conflict Surface

This design touches `internal/vm`, `internal/builtins`, `internal/gen/golang` — the mandatory section applies. No syntactic/parse positions are touched; the surfaces are semantic.

### Positions touched

1. The `fn func(float64) float64` argument of `registerTrigFunc`/`registerTrigFunc2` for the 11 transcendental registrations (interpreter).
2. The Go stdlib function references inside `builtinMath*` (VM).
3. The `Inline` Go-expression templates in `GoCodegenSpec` for `_math_*` and the `mathFunctions`/`mathConstants` maps (compiled codegen); **`mathConstants` (`PI`, `E`) is untouched** — `math.Pi`/`math.E` are exact constants, arch-independent.
4. The emitted-program namespace: new helper functions `__ail_exp` etc. (naming per existing emitted-helper conventions — executor picks a collision-free convention and pins it with a test; note existing emitted helpers already face this class of collision, e.g. `Cons`/`RecordUpdate`).

### What else lives in these positions

| Position | Existing occupant | Disposition |
|---|---|---|
| `_math_exp`/`_math_log` builtin slots | host `math.Exp`/`math.Log` pointers | **swapped** to `mathx` |
| `_math_sqrt`, `_math_floor`, `_math_ceil`, `_math_round`, `_math_abs_Float`, `_math_abs_Int`, `_math_PI`, `_math_E` | host `math` exact/constant operations | **unchanged** (D4 — no divergence possible) |
| `_float_to_int`, `_int_to_float` | conversion builtins | unchanged |
| `std/math.ail` signatures, purity, metadata | `export pure func` wrappers | unchanged (byte-identical API) |
| Codegen `"math"` import emission (`needsMathImport`) | needed when math used | **condition extends** to helper emission in `skipRuntimeHelpers` mode; constants-only programs keep importing `"math"` |
| `internal/builtins/math_test.go` expectations | `InDelta(1e-10)` tolerances | **unchanged** — tolerant by construction, pin nothing at ulp scale |
| `internal/vm` math coverage | **none exists today** (no `internal/vm/builtins_math_test.go` — VL-12) | the new differential test adds the VM row |

### Disambiguation strategy

No parser disambiguation involved. The one discrimination kept: codegen distinguishes math *constants* (`math.Pi` emitted as-is) from math *functions* (now emitted as `__ail_*` helper calls) — the existing `getMathConstant` vs `getMathFunction` split (`codegen_expr_simple.go:286–356`), with only the function side changing.

### Programs that MUST still work (fixtures)

1. `examples/runnable/math_trig.ail` — calls all 12 math functions including `exp`, `log`, `sqrt`, `pow` (VL-15).
2. `examples/float_nan.ail` — `isNaN`/`sqrt` float edge paths (VL-15).
3. `internal/builtins/math_test.go` `TestExpImpl`/`TestLogImpl` (+ sin/cos siblings) — delta-tolerant, must stay green unchanged (VL-13).
4. `internal/gen/golang/codegen_math_test.go` `TestMathFunctionCall`/`TestMathConstantsInExpressions`/Inline-template table — updated **intentionally**, must stay green with `__ail_*` expectations.
5. The report's repro module (inlined in Verification Log VL-1) — must produce `1.2293173989217931` on every arch post-fix.

### What deliberately changes

(a) arm64/ppc64/s390x/riscv64 last-ulp values for the 11 functions on FMA-affected inputs (the point of the fix; arm64-generated goldens regenerate — D1); (b) generated-Go text: `math.Sin(x.(float64))` → `__ail_sin(x.(float64))` plus emitted helper bodies; (c) compiled programs gain ~a few KB of emitted helpers when math functions are used. x86_64 output values: **no change expected** (Phase 0 verifies; any exception escalates D1).

---

## Examples

### Example 1: The report's case

**Before** (v0.50.0/v0.50.1):
```ailang
module fma
import std/math (exp, log)
import std/string (concat)
export pure func main() -> string =
  concat([show(exp(0.2064590551107192)), " ", show(log(exp(0.2064590551107192)))])
-- x86_64 (this container, interpreter AND --strict-bytecode, byte-identical — VL-1/VL-2):
--   1.2293173989217931 0.20645905511071927
-- darwin/linux arm64 (reporter):
--   1.229317398921793 0.2064590551107191
```

**After** (both arches, both execution paths):
```ailang
--   1.2293173989217931 0.20645905511071927
```

### Example 2: stapledons-godot goldens collapse

A seed's sim output, run on CI (x86_64) and on the arm64 dev laptop, commits one golden file instead of `golden.arm64` + `golden.amd64` — the per-arch workaround's number branch retires on upgrade (same consumer-action pattern as m-json-number-roundtrip's `num` workaround).

---

## Success Criteria

- [ ] `mathx.Exp(0.2064590551107192)` bit-equals `math.Float64bits(1.2293173989217931)`; same asserted through `.ail` `show` text on interpreter and `--strict-bytecode`
- [ ] Golden-bit tables committed; `go test ./internal/mathx/` green on x86_64 **and** the arm64 CI runner (identical assertions)
- [ ] Phase-0 census recorded in the implementation report: per-function arm64-vs-amd64 host divergence (before) and mathx-vs-host delta on both arches (expected: zero on amd64)
- [ ] Differential interpreter / VM / compiled-Go test green over the golden table
- [ ] The five Conflict-Surface fixtures pass (with the intentional codegen-text expectation updates)
- [ ] `make test`, `make check-boundaries`, `make simplicity-audit` green; CHANGELOG entry at ship
- [ ] arm64 CI job green and wired into the required checks (D5)

## Testing Strategy

**Unit:** golden-bit tables per function (boundary + special + sweep inputs), asserted bit-exactly; arch-independent by construction.
**Differential:** three backends over the same table (compiled arm CI-only, needs Go toolchain); `mathx` vs emitted-`__ail_*` equality (guards the dual copy).
**Cross-arch:** the D5 arm64 runner asserting the *same* golden bits — the regression guard for the whole class.
**Manual:** none beyond regenerating consumer goldens on arm64 (D1 consequence).

## Deferred Decisions

- Exact emitted-helper naming convention (must be collision-checked against user-identifier mangling — executor picks and pins with a test).
- Golden-table storage format (JSON vs generated Go) — executor's choice.
- Whether `mathx` also grows `expm1`/`log1p` etc. — API additions, out of scope here.

## Non-Goals

- **Compiled-mode user-level float arithmetic FMA contraction.** An AILANG source expression `x*y + z` compiles to the Go expression `x*y + z` (`mapIntrinsicOp`, `codegen_ops.go:507–517`), which the Go compiler **may fuse on arm64** — so compiled programs can still differ from the interpreter/VM (and across arches) for fused user arithmetic. Fixing it requires type-aware codegen (`float64(x*y)` around float products feeding `+`/`-`) — a separate, larger design. This doc ships the LIMITATIONS entry (Phase 3) and records the follow-up in Future Work. *Not the reported bug* — the interpreter/VM execute each AILANG operator as its own Go statement (no fusion opportunity — VL-8), which is why the VM and interpreter agree per-arch today.
- **Correctly-rounded (CRlibm-class) transcendentals** — rejected in D1 (churn on all arches for no stated consumer need).
- **Arbitrary-precision / decimal math** — language change, different lane.
- **`show` float formatting** — presentation layer, untouched (m-json-number-roundtrip owns the JSON-boundary formatting).
- **Performance parity with arm64 assembly** — accepted tradeoff (amd64 already runs pure-Go fdlibm; see Risks).

## Timeline

**Week 1** (Phase 0: 0.5d, Phase 1: 1d, Phase 2: 1d, Phase 3: 0.5–1d ≈ 3–4 days incl. buffer):
- Phase 0: cross-arch audit + golden tables
- Phase 1: `internal/mathx` + golden-bit tests
- Phase 2: three-surface rewiring + differential tests
- Phase 3: arm64 CI job, LIMITATIONS, CHANGELOG, consumer coordination

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| arm64 goldens (consumer + dev machines) shift by ≤1 ulp on affected inputs | Med | Exactly the intended change; D1 human sign-off pre-sprint; CHANGELOG flags it; x86_64/CI goldens untouched |
| Phase 0 finds amd64 deltas between portable and host math (e.g. if amd64 also uses a non-pure-Go path for some function) | Med | Stops wiring, re-escalates D1 (canonical set = portable output either way; the *migration note* grows) — this is why Phase 0 precedes Phase 2 |
| Dual copy of the algorithms (`mathx` + codegen templates) drifts | Med | CI differential test compiles emitted helpers and bit-compares against `mathx` over the golden table |
| Perf regression on FMA arches (host asm paths are fast) | Low–Med | amd64 already runs this algorithm class (no change there); arm64 loses its asm path but keeps hardware FMA-free, well-optimized fdlibm; `bench/` comparison recorded in the implementation report |
| A future Go release changes pure-Go `math` values | None | We froze our fork — the point of the design; golden bits pin us regardless |
| `__ail_*` helper name collides with a user identifier in generated code | Low | Executor picks per existing emitted-helper conventions and pins with a collision test (existing helpers like `Cons` already establish the convention's collision policy) |

## Related Documents

The create script's doc search returned no matches (and crashed — adjacent defect 1, re-hit this session); these are curated by code citation:

**Planned (adjacent, same reporter/consumer — distinct defects):**
- [m-json-number-roundtrip](../../implemented/v0_51_1/m-json-number-roundtrip.md) — float *text* at the JSON system boundary (same stapledons-godot `num` workaround family); this doc is float *arithmetic bits* across CPUs. No overlap in files or behavior; both retire the same consumer's workarounds.

**Planned (adjacent genre):**
- [m-bytecode-vm-parity-bugs](../v1_0_0/m-bytecode-vm-parity-bugs.md) — backend parity for effect rows; this doc explicitly does *not* touch VM-vs-interpreter parity (they already agree per-arch).

**Implemented (precedents):**
- [m-numerics-vec-array-ingest](../implemented/v0_47_0/m-numerics-vec-array-ingest.md) — exact `strconv.ParseFloat` handling precedent in the same numeric-adjacent stdlib area
- [m-bytecode-stdlib-builtins-sprint-plan](../implemented/v0_11_0/m-bytecode-stdlib-builtins-sprint-plan.md) — the M2 VM wiring this doc rewires (tagged from `internal/vm/builtins_math.go` header)

**External:**
- stapledons-godot per-arch replay goldens — the consumer workaround this fix retires.

## References

- Go spec, "Floating-point operators" — FMA contraction permitted on FMA-capable GOARCH; explicit conversion `float64(x*y)` forces rounding (the fusion-proofing idiom)
- Go spec, "Constant expressions" — constant arithmetic exactness (why coefficient tables need no wrapping)
- IEEE 754-2019 — `sqrt` correctly-rounded requirement (D4's exclusion argument)
- Go `math` package sources (fdlibm/BSD-derived algorithms; GOARCH-specific implementation files)
- Report: v0.50.0 release, binary `v0.50.0-6-g021c46907-dirty` (task `task-5077e633` attachment); glibc/Python reference values measured by the reporter

## Future Work

- **Compiled-mode user-arithmetic fusion** (the Non-Goal above): type-aware codegen emitting rounding-forcing conversions for float products feeding `+`/`-`, or a documented language-level stance. Needs its own design doc (codegen semantics, perf tradeoff).
- Optional `expm1`/`log1p`/`exp2` stdlib additions built on `mathx` (API growth, separate trivial/feature lane).
- A "same seed, same bytes" cross-arch end-to-end replay CI check for the full VM (generalizing D5 beyond math).

---

## Verification Log

Run 2026-10-01 with `ailang` v0.50.1 @ `021c469` (same commit as the report's v0.50.0-6-g021c46907-dirty binary) on **x86_64** (CLOUD_RUN container, `uname -m`). This container has **no Go toolchain, no `make` binary, no arm64 box** — Go-behavior claims are live-probed through the AILANG binary, taken from the reporter's measurements, or cited to the Go spec, each marked. Repro programs passed `ailang check` first.

| # | Claim | Command | Observed |
|---|---|---|---|
| VL-1 | The x86_64 column of the report reproduces on the interpreter | `/tmp/fma/fma2.ail` (module `fma2`; imports `std/math (exp, log)` + `std/string (concat)`; body prints `show(exp(0.2064590551107192))`, `show(log(1.229317398921793))`, `show(log(exp(0.2064590551107192)))`), `ailang run` | `1.2293173989217931 0.2064590551107191 0.20645905511071927` — exp matches glibc/Python per the report; log-of-identical-input matches arm64's value |
| VL-2 | VM and interpreter agree per-arch (x86_64 side) | `ailang run --strict-bytecode /tmp/fma/fma2.ail` | byte-identical to VL-1 (all three values) |
| VL-3 | `log` agrees across arches for identical input — `exp` is the source | Reporter's arm64 `log(1.229317398921793)` = `0.2064590551107191` vs this container's x86_64 `log(1.229317398921793)` (VL-1, second value) | identical decimal text on both arches; the reported `log` divergence appears only via `log(exp(x))`, i.e. propagated from exp's differing argument |
| VL-4 | All three surfaces delegate `exp`/`log` to host Go `math` | `grep -n "math.Exp\|math.Log" internal/vm/builtins_math.go internal/builtins/math_trig.go internal/builtins/registry_codegen_math.go internal/gen/golang/codegen_expr_simple.go` | math_trig.go:121/:132 (interpreter), vm/builtins_math.go:36/:39 (VM), registry_codegen_math.go:16–18 (compiled Inline templates), codegen_expr_simple.go:313–320 + codegen_expr_app.go:24 (`mathFunctions` map, App path) |
| VL-5 | `std/math.ail` `exp`/`log` are pure delegation — no AILANG-level algorithm exists | `sed -n '45,52p' std/math.ail` | `:46 export pure func exp(x: float) -> float = _math_exp(x)`; `:49 … log … = _math_log(x)` |
| VL-6 | No portable/forked float implementation exists anywhere in `internal/` (negative-existence) | `grep -rniE "fdlibm\|frexp" internal/ --include='*.go'` | empty — nothing to reuse; the fork must be vendored |
| VL-7 | `show` renders floats shortest-round-trip, so 1 ulp changes printed text | `sed -n '113,129p' internal/eval/show.go`; `runtime/show.go:20` | `showFloat` = `strconv.FormatFloat(f, 'f', -1, 64)` (+ `.0` for whole); compiled-runtime Show uses `%g` — both are shortest-round-trip forms |
| VL-8 | No FMA-fusable multi-op float expression in the interpreter/VM arithmetic paths (per-arch VM/interpreter agreement is structural) | `grep -rnE '\* *[A-Za-z_.\[\]]+ *\+\|\+ *[A-Za-z_.\[\]]+ *\*' internal/eval/*.go internal/vm/*.go internal/builtins/math*.go` (non-test, float-relevant) | no hits — each AILANG arithmetic operator compiles to its own single-op Go statement; Phase 0 re-verifies on arm64 |
| VL-9 | js/wasm target reaches the same interpreter builtins (inherits the fix) | `grep -n "internal/eval" cmd/wasm/main.go` | `:13` imports `internal/eval` — same builtin registry |
| VL-10 | CI never runs Go tests on arm64 today (D5 premise) | `grep -n "runs-on" .github/workflows/ci.yml`; `sed -n '730,745p' .github/workflows/ci.yml`; `sed -n '15,45p' .github/workflows/build.yml` | Go-test jobs all `ubuntu-latest` (x86_64); the only `macos-latest` job is bash-3.2 shell tests ("No Go"); build.yml cross-builds `ailang-darwin-arm64` and other artifacts **without testing** them |
| VL-11 | Generated programs are stdlib-import-only (codegen emission must stay dependency-free — D3) | `sed -n '500,535p' cmd/ailang/compile.go` | `verifyGoCompilation` writes a dependency-free temp `go.mod` (`module <pkg>; go 1.21`) — generated code cannot import `github.com/sunholo-data/ailang/...` |
| VL-12 | VM math builtins have no test coverage today (differential test adds the row) | `ls internal/vm/*math*` | only `builtins_math.go` exists — no `builtins_math_test.go` |
| VL-13 | Existing interpreter math tests are ulp-tolerant (won't break, pin nothing) | `sed -n '375,410p' internal/builtins/math_test.go` | `TestExpImpl`/`TestLogImpl` use `assert.InDelta(…, 1e-10)` over spec `Impl` — unaffected by last-ulp swaps |
| VL-14 | `docs/LIMITATIONS.md` has no cross-arch float entry today (Phase-3 addition is new) | `grep -in "arch\|arm64\|ulp\|transcendental" docs/LIMITATIONS.md` | no match (existing float entry is SMT-real vs runtime NaN, #1274) |
| VL-15 | Conflict-Surface fixtures exist | `head -8 examples/runnable/math_trig.ail` (imports sin, cos, tan, sqrt, atan2, PI, E, pow, exp, log, floor, ceil, round, abs); `head -5 examples/float_nan.ail` | both exist; math_trig exercises all 12 functions |
| VL-16 | Codegen tests pin host-`math` text (intentional-change surface) | `grep -n "math.Sin\|math.Exp\|math.Log" internal/gen/golang/codegen_math_test.go` | rows at :24 (`{"sin builtin", "_math_sin", "math.Sin"}`), :35–37 (exp/log/log10), :186–188 and :289–291 (assert generated code contains `math.Sin`) |
| VL-17 | Duplicate/coverage gate: no existing or queued doc covers cross-arch float determinism | create-script doc search (crashed — adjacent defect 1); `ls design_docs/planned/ \| grep -iE "float\|math\|fma\|ulp\|arch"`; targeted read of m-json-number-roundtrip | no planned/implemented doc on float arithmetic determinism; nearest are m-json-number-roundtrip (float *text* at the JSON boundary) and m-ifc-cross-module-labels (unrelated); m-array-show-diverges-run-vs-compile is `show` divergence between backends on one machine — different defect class |
| VL-18 | Supported release GOOS/GOARCH (impact scope) | `sed -n '15,45p' .github/workflows/build.yml` | linux/amd64, darwin/amd64, darwin/arm64, windows/amd64 (+ js/wasm via `make/build.mk:103`); reporter also runs linux/arm64 locally |
| VL-19 | No new error codes / diagnostics proposed (namespace check N/A) | — (design proposes none) | n/a — no `MODxxx`/`PARxxx`/`TCxxx` allocations in this design |
| VL-20 | Go version pinning matters for the audit (asm layout varies by release) | `head -3 go.mod` | `go 1.26.6` — Phase 0 census is per-toolchain; the frozen fork makes future Go changes irrelevant |

**Go-internal mechanism caveat (honesty note):** whether the arm64 divergence for a given function comes from compiler FMA contraction of the pure-Go algorithm or from a GOARCH-specific assembly path in Go's `math` package is not resolvable in this container (no Go toolchain, no arm64). It does not need to be: the fix (D2) stops calling host `math` for the transcendental surface and is immune to both mechanisms. Phase 0 records which functions were affected, as the census baseline.

## Adjacent defects found while verifying (recorded, not fixed here)

1. **`create_planned_doc.sh` crashes before creating the file when doc search returns no matches** (PCRE `\d` under `grep -E` + `set -e`/pipefail; exit 1, no file created). Re-hit twice this session. Already recorded as adjacent defect 1 in m-json-number-roundtrip; this doc was scaffolded by hand to match the script's documented output. The one-liner fix (`\d` → `[0-9]`) remains queued for a trivial-fix task.
2. **No VM math coverage**: `internal/vm/builtins_math.go` has no test file (VL-12) — the VM path of every std/math function is untested; the differential test in Phase 2 is the first coverage.
3. **`examples/` has no golden-output fixture for std/math** — `examples/runnable/math_trig.ail` prints but asserts nothing; the committed golden-bit tables (Phase 1) become the first machine-checked cross-arch reference.

## Out of scope (see Non-Goals)

Compiled-mode user-arithmetic fusion; correctly-rounded transcendentals; arbitrary-precision math; `show` formatting; new stdlib API.