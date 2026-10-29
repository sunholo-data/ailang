# M-INT-OVERFLOW-TRAP-AND-BIGINT: integer overflow fails loudly; exact big integers are a library

**Status**: Planned. Quorum round 1 BLOCKED; both objections accepted and revised in; round 2 pending. Design-freeze items F1–F3 need a human ruling before the sprint.
**Target**: v0.55.0
**Priority**: P1 — a silent-wrong-answer class (CLAUDE.md principle 2) and a verifier soundness gap
**Estimated**: 6–8 days (Phase 0: 0.5d census · Phase 1: 3d trap · Phase 2: 2d std/bigint · Phase 3: 1d helpers + prompt)
**Dependencies**: None. Composes with [m-smt-well-sorted-preflight](../m-smt-well-sorted-preflight.md) (division operator) and [m-runtime-contract-parity](../m-runtime-contract-parity.md) (its ledger's "Z3 `Int` vs runtime `int64`" row is the gap this doc closes)
**Planner-Lane**: opus-required (runtime arithmetic on every engine)
**Quorum trigger**: #1 (design-freeze items) and #3 (changes the banked `ailang.verification-evidence/v1` `numeric_model` field). See [Quorum log](#quorum-log).

All facts below were re-derived first-party at `v0.54.19-1-g379a9e20e` (installed `ailang`, built
2026-10-27). The programs live in the session scratchpad. Every row in the [Verification Log](#verification-log) gives
the command that produced it.

## Routing Lane (PROGRAM.md §4)

**Lane: AILANG fix**, split into two halves with different costs:

- **Trap on overflow** is a runtime-semantics change on every engine. It cannot be an extension: no
  library can make `+` stop wrapping. PROGRAM.md §4 routes "type-system gap / dialect trap" to
  AILANG fix.
- **Exact big integers** go the extension way as far as they can. Phase 2 ships `std/bigint` as an
  ADT over a canonical decimal string, with Go `math/big` builtins. It adds no new value kind, no VM
  tag and no codegen mapping. A native value kind is deferred until a measurement asks for it (see
  [Future Work](#future-work)).

## Problem Statement

AILANG `int` is a signed 64-bit integer, and `+`, `-`, `*`, unary `-`, `abs` and `MIN / -1` wrap
**silently**. This holds on the evaluator, the bytecode VM and Go codegen, which agree bit-for-bit
(V1–V3). Nothing in the docs says so: neither `docs/LIMITATIONS.md` nor
`docs/docs/reference/limitations.md` mentions overflow (V14). The only description is the teaching
prompt's "signed hash semantics" paragraph, which says how wrapping *looks* and never says it
happens without an error.

Division by zero has been a loud, positioned `RT001` on every route since
[M-INT-DIV-ZERO-ERROR](../../implemented/v0_52_0/m-int-div-zero-error.md) (#1449, V4). Overflow is
the same class of event (an arithmetic result with no value in the type), but it produces a
plausible wrong number and carries on. That is the "silent fallback affecting data integrity" that
CLAUDE.md principle 2 forbids.

### Evidence 1: models lose correctness to it (competition-problem spike, 2026-10-28)

The spike ran AetherCode problems with Claude Haiku 4.5 in agent mode and graded saved solutions
against **all** hidden tests (scratchpad `cp/grade_ha3.json`, `cp/grade_c.json`).

| Problem | Python (small hidden tests all correct) | AILANG | Mechanism (read from the saved `.ail`) |
|---|---|---|---|
| **cp_60166** (CCPC Final 2024: maximize the sum of segment products; answer mod 1e9+7) | **4/4 trials**, 11/11 small tests each | **0/3 valid trials** (1 void: `api_error`, no code) | Python runs an O(n²) DP over **exact** big products and reduces mod only at the end. Two AILANG trials reduced mod at every step and took `max` over the **residues** (wrong: 2/11 each). The third wrote Python's DP on raw `int` (`x * product(...)`, no mod). That is overflow-exposed and silent; it scored 10/11, with one timeout |
| **cp_60149** (house numbering, K ≤ 1e18; weak) | 1/1 valid trial, 33/33 | 1 valid trial, 27/33 | Unguarded `power(d, t) - 1 >= k` loop. `power(30, 13)` wraps to `-2503514073709551616`, so the `>= k` check is silently **false** (V5). Overflow is reachable. Whether it caused the 6 wrong answers is **not proven**; the file also has a logic bug |

Neither language passes `cp_60166`'s big tests: Python's exact-bigint DP times out on 5/6 of them.
So this is a delta at the **naive-correct tier**. The language that has exact integers can write the
obvious correct program; the one that wraps cannot, and gets no signal that it went wrong. This is
the tier a coding agent works at.

### Evidence 2: the verifier proves things the runtime then violates (V6)

```ailang
export func inc(x: int) -> int ! {}
requires { x >= 0 }
ensures { result > x } {
  x + 1
}
-- main: println(show(inc(9223372036854775807)))
```

- `ailang check --verify` reports **`"status": "verified"`** and declares
  `numeric_model: {int_sort: "Int", runtime_equivalence: "not_established"}`.
- `ailang run` prints `-9223372036854775808`.
- `ailang run --verify-contracts` catches it: `contract violation: ensures failed ... (result > x)`.

The evidence schema is honest about the gap (`not_established`), but `verified` is the word a model
or a reviewer reads. The 2026-09-28 triage row
[verify-int-width-mismatch-z3-int-vs-int64](../ailang-core-triage/verify-int-width-mismatch-z3-int-vs-int64.md)
asked for a design doc to choose a verifier-side fix (docs note, BitVec-64 encoding, or overflow
lint). None was written. **This doc answers that question from the runtime side instead.** If
`+ - * neg abs` trap rather than wrap, every execution that returns a value returns the ℤ value. A
Z3 proof over unbounded `Int` is then sound for partial correctness without changing the encoder.

### Evidence 3: the conversions are silent too (V7)

- `floatToInt(1.0e30)` returns `9223372036854775807` (saturates).
- `floatToInt(-1.0e30)` returns `MIN`.
- `floatToInt(NaN)` returns `0`.
- `stringToInt("99999999999999999999")` returns `None`. It is already the correct total shape and is
  the model for the rest.

## Goals

**Primary goal:** No integer operation in AILANG produces a value that differs from its
mathematical result without either a runtime error or an explicitly wrapping operation in the
source.

**Success metrics:**
1. `9223372036854775807 + 1` (and each of `- * neg abs`, `MIN / -1`, out-of-range `floatToInt`)
   fails with **one** code on the evaluator, VM and REPL. Go codegen panics with the same code.
2. The `inc` program above: `ailang check --verify` keeps `verified`, and its `numeric_model` says
   the int model is exact modulo trap. `ailang run` traps instead of printing a negative number.
3. `examples/runnable/fnv1a.ail` and `examples/runnable/splitmix64.ail` print **byte-identical**
   output after migrating to `wrapMul`/`wrapAdd`.
4. A `cp_60166`-derived benchmark (built by the eval-suite track) has an exact-arithmetic reference
   AILANG solution using `std/bigint` that passes its small hidden tests.
5. The checked-arithmetic cost on the VM is ≤ 5% on a pure-int loop benchmark (estimate below).

## High-Impact Decisions

| # | Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|---|
| D1 | `+ - * neg abs` **trap by default** (vs. opt-in flag, vs. auto-promote to big) | Changes the meaning of existing programs that rely on wrapping. Defines what `verified` means | **human (F1)** | design | high |
| D2 | One new code, **RT010 "integer overflow"**, raised from a shared helper next to `CheckIntDivisor` | Error namespace and seam: one helper or seven drifting copies | agent (recommended) | design | med |
| D3 | Explicit wrapping ops (`wrapAdd/wrapSub/wrapMul/wrapNeg`) in `std/math` | The migration path for hash and PRNG code; they become the only way to get wraparound | **human (F2)** names/home | design | med |
| D4 | `std/bigint` as an ADT over a canonical decimal string, not a native value kind | Decides whether exact integers cost ~1 module or ~32 files (the `bytes` precedent, V12) | **human (F3)** | design | high |
| D5 | `numeric_model.runtime_equivalence` gains a value for int (e.g. `"exact_or_trap"`) | Banked schema `ailang.verification-evidence/v1`; consumers read it | agent within F1 | compile | med |
| D6 | Bit ops (`& ^ ~ << >>`, `shiftRightLogical`) keep bit-pattern (wrapping) semantics | They are pattern operations, not arithmetic. Changing them breaks every hash | compiler | design | low |

### Design Freeze

- [ ] **F1 — trap by default.** Recommended: yes. Alternatives and why they lose are in
  [Options](#options-considered). This deliberately breaks programs that rely on wrapping `*`/`+`.
  Known instances: two examples, the teaching prompt's hash paragraph, and the parked
  `m-pure-prng` design (V8, V9). Phase 0 measures the rest before anything changes.
- [ ] **F2 — the explicit-wrap API.** Recommended: `wrapAdd`, `wrapSub`, `wrapMul`, `wrapNeg` in
  `std/math`, next to `shiftRightLogical` and `bitwiseOr`, which already serve hash code. Plus
  `checkedAdd/Sub/Mul -> Option[int]` and `mulMod(a, b, m)` / `powMod(b, e, m)` (exact 128-bit
  intermediate). All ten names are unallocated (V11).
- [ ] **F3 — `std/bigint` representation.** Recommended: an ADT over a canonical decimal string,
  with Go builtins operating on strings. A native value kind is deferred (see Future Work).

## Options considered

| Option | What it does | Verdict |
|---|---|---|
| **A. Status quo + docs** | Document the wrap in LIMITATIONS and the prompt | Rejected. Honest, but `cp_60166`-style programs still return wrong numbers, and `verified` stays unsound. This is the triage row's option (a) |
| **B. Trap by default + explicit wrap ops** | RT010 on overflow everywhere; `wrapMul` etc. for code that wants wraparound | **Recommended.** It matches the RT001 precedent (V4) and makes Z3 `Int` proofs sound modulo trap (Evidence 2). Cost: one shared helper, the seam sweep, and migrating wrap-reliant code |
| C. Opt-in flag (`--int-overflow=trap`) | Two runtime semantics chosen per run | Rejected. The same program prints different results under different flags (an A1 hazard); the eval harness, REPL, serve-api and codegen would each need to pick one; and the verifier still cannot assume either |
| D. Auto-promote to big on overflow (Python semantics) | `int` becomes a small/big tagged value | Deferred. It is the most model-friendly and perfectly matches Z3 `Int`, but it changes the value representation on every engine (VM `Value.Int` is a bare `int64`, V10) and adds a branch on every op. Revisit only if B plus `std/bigint` measurably fails models |
| E. Verifier-side fix (BitVec-64 encoding) | Change what `verified` means; leave the runtime wrapping | Rejected as primary. It fixes the proof, not the wrong answer, and makes currently-provable ℤ goals harder. B makes it unnecessary for `+ - * neg abs` |

## Solution Design

### Phase 0: census before changing anything (0.5 day)

Build a scratch binary in which the shared helper (Phase 1) **records** overflow sites instead of
trapping. Run it over:
- `make verify-examples`, `make test-core` and `make test`;
- every `benchmarks/frontier_refs/*.ail` reference solution, with its benchmark's stdin;
- the `std/*.ail` inline tests.

Output: the list of programs and sites that overflow today. Every hit is either migrated to
`wrap*`/`checked*` or exposes a latent bug, and the list is pasted into this doc before Phase 1
merges. The record mode is a test-only build tag, **not** a new env var or flag; new env vars go
through the `internal/config` Registry and this one must not ship.

Known before the census (V8):
- `examples/runnable/fnv1a.ail`: `(hash ^ charCode(c)) * fnvPrime`.
- `examples/runnable/splitmix64.ail`: three wrapping `*` with 64-bit constants.
- No `benchmarks/*.yml` and no `std/*.ail` code was found relying on wraparound (grep over hex
  constants and hash names; `audit_chain_replay` computes `fib_sum(20) = 17710`, which does not
  overflow).

### Phase 1: RT010 on every route (3 days)

**One helper, every seam.** `internal/errors/arith.go` gains functions next to `CheckIntDivisor`:

```go
// IntOverflowError is RT010: an int + - * neg abs, or MIN / -1, whose
// mathematical result is outside [MinInt64, MaxInt64]. Pos is filled by the
// evaluator like DivByZeroError.
type IntOverflowError struct{ Op, Pos string }

func AddInt64(a, b int64) (int64, error) // (a^s)&(b^s) < 0 → overflow
func SubInt64(a, b int64) (int64, error)
func MulInt64(a, b int64) (int64, error) // bits.Mul64 + signed high-word correction
func NegInt64(a int64) (int64, error)    // a == MinInt64 → overflow
```

`CheckIntDivisor`'s doc comment already records that Go wraps `MinInt64 / -1` to `MinInt64` (and
gives `MinInt64 % -1` = 0, which is the ℤ answer and stays). Division gains the
`a == MinInt64 && b == -1` check in the **same** helper call sites, so `/` stays a single
call per route.

**Seams.** These are all the Go sites that perform AILANG int `+ - *` today (V13). Each one calls the helper, and
none keeps its own check:

| Route | File | Sites |
|---|---|---|
| `Num[int]` dictionary (evaluator live path; V1 of the div-zero doc) | `internal/types/dictionaries.go` | add, sub, mul, neg, abs |
| Evaluator builtins | `internal/eval/builtins_arithmetic.go:17/26/35/68` | `add_Int`, `sub_Int`, `mul_Int`, neg |
| Evaluator binop shim / simple / typed paths | `internal/eval/eval_operations.go:402/404/406/520`, `eval_simple.go:393/414/427/611`, `eval_typed.go:600/614/628/777` | 12 (incl. unary `-`) |
| Shared builtins | `internal/builtins/math_arithmetic.go:16/18/20/34/36` (`add_Int`, `sub_Int`, `mul_Int`, `neg_Int`, `double_Int`), `math_trig.go` (`_math_abs_Int`), `numeric.go` (`_float_to_int`) | 7 |
| VM | `internal/vm/vm.go:652/654/656` `arith` (`OpAdd/OpSub/OpMul`), `:212` `OpNeg` | 4 |
| REPL dictionaries | `internal/repl/module_registry_prelude.go:171/172/173` and `:324/325/326` (`wrapInt2("add"/"sub"/"mul")`) | 6 |
| Go codegen | `internal/gen/golang/codegen_ops.go` `mapIntrinsicOp` (raw `+ - *` shared with float), `codegen_runtime_arith.go` (`AddInt/SubInt/MulInt`), `codegen_dictionaries.go` (`dict_Num_Int`) | 3 generators |

**Codegen.** `mapIntrinsicOp` is type-agnostic: it emits `+` for both int and float. The int case
must become type-directed and emit `rtAddInt(a, b)`. The generated helper panics with an
`IntOverflowError` value whose `Error()` carries `RT010`. That is consistent with how generated code
already reports runtime faults: generated pure functions have no error return, and the div-zero
doc left codegen `/` as a Go panic (its V11).

**`floatToInt`.** It traps with RT010 when the argument is NaN or outside `[-2^63, 2^63)`.
`floatToIntOpt -> Option[int]` is added for callers that want a total version (agent may name it).

**Verifier evidence (D5).** `internal/verification/evidence.go` `EncodedEvidence()` reports the int
half honestly, e.g. `int_sort: "Int", runtime_equivalence: "exact_or_trap"` for int while floats stay
`not_established`. The exact field shape is agent-resolvable inside the v1 schema, or a v2 bump if
consumers key on the string (grep consumers first). The claim covers `+ - * neg abs` and division's
`MIN / -1` only. Bit ops are encoded as `bv*` over `Int`-sorted terms (V15, ill-sorted), and `div`/`mod`
truncation-vs-Euclidean is [m-smt-well-sorted-preflight](../m-smt-well-sorted-preflight.md)'s S1.
Both stay out of the claim.

### Phase 2: `std/bigint` as an extension (2 days)

```ailang
module std/bigint
export type BigInt = BigInt(string)   -- canonical decimal: no leading zeros, no "-0". NO deriving (Eq)

export pure func eq(a: BigInt, b: BigInt) -> bool        -- numeric equality, via the builtin
export pure func fromInt(n: int) -> BigInt
export pure func parse(s: string) -> Option[BigInt]      -- rejects non-canonical input
export pure func toInt(b: BigInt) -> Option[int]         -- None when out of range
export pure func toString(b: BigInt) -> string
export pure func add(a: BigInt, b: BigInt) -> BigInt
export pure func sub(a: BigInt, b: BigInt) -> BigInt
export pure func mul(a: BigInt, b: BigInt) -> BigInt
export pure func divMod(a: BigInt, b: BigInt) -> Option[(BigInt, BigInt)]  -- None on zero divisor
export pure func modInt(a: BigInt, m: int) -> int        -- the "reduce at the end" step of cp_60166
export pure func compare(a: BigInt, b: BigInt) -> int    -- -1 / 0 / 1
export pure func max(a: BigInt, b: BigInt) -> BigInt
export pure func pow(b: BigInt, e: int) -> BigInt
```

- **Why an ADT over a string.** AILANG has no user instance declarations:
  `parseInstanceDeclaration` is `// TODO ... return nil` (V12), so no library type can get `+` or `<`.
  A named-function API is therefore the only extension-lane shape.
- **Why no `deriving (Eq)` (quorum round 1).** The constructor is exported, and AILANG has no
  constructor-hiding export (agent verifies in Phase 2; if one exists, use it in addition). A
  hand-built `BigInt("007")` or `BigInt("-0")` would make derived structural `==` silently disagree
  with numeric equality. Without `deriving`, `==` on `BigInt` is a **compile-time type error**:
  `No instance for Eq[BigInt] in scope` (V21). So the only equality is `bigint.eq`, which goes
  through the builtin. This choice trades `==` ergonomics for no silent path.
- **Every builtin validates.** Each `_bigint_*` builtin, **including `toString`, `eq` and
  `compare`**, rejects non-canonical input with a runtime error: an empty string, a leading zero,
  `"-0"`, a non-digit, or a leading `+`. A malformed value therefore cannot be printed, compared or
  computed with, and canonical form is an *enforced* precondition rather than an assumed invariant.
- **Why not a new value kind.** That needs a type constructor, an evaluator `Value`, a VM tag and a
  codegen mapping. The `bytes` precedent spans 32 files (V12). It is deferred until a
  measurement asks for it.
- **Builtins.** Each operation is a Go builtin over strings that parses with `math/big`, computes and
  re-formats. The AILANG wrapper unwraps the ADT, so builtins see only strings and the VM's interop
  bridge never carries an ADT. Cost: one parse and one format per op, O(digits) for the digit counts
  these programs produce. Malformed contents in a hand-built `BigInt("x")` are a **runtime error**
  from the builtin, never a silent zero (A11).

### Phase 3: helpers and teaching (1 day)

- `std/math`: `wrapAdd/wrapSub/wrapMul/wrapNeg` (Go `int64` ops as builtins), `checkedAdd/Sub/Mul ->
  Option[int]`, and `mulMod/powMod` (exact via `bits.Mul64` + `bits.Div64`).
- Migrate `examples/runnable/fnv1a.ail` and `splitmix64.ail` to `wrapMul`/`wrapAdd`, with
  byte-identical output (success metric 3).
- Teaching prompt (`prompt-manager` skill): replace the "signed hash semantics" paragraph's implicit
  wrap with a short rule. `int` is 64-bit; `+ - *` trap with RT010 on overflow; hash/PRNG code uses
  `wrapMul`/`wrapAdd`; exact large values use `std/bigint`; modular arithmetic uses
  `mulMod`/`powMod`. The rule is verified with `ailang check` before it lands (memory: verify syntax
  claims).
- `docs/LIMITATIONS.md`: a row "int is 64-bit; overflow is RT010" with a link here.
- The `RT010` registry row: `{"runtime", "arithmetic", "Integer overflow"}`, plus a fix hint in
  `json_encoder.go` in the RT001 style ("use std/bigint for exact large values, wrapMul for hashes").

### Performance (estimate, measured parts marked)

- **Measured** (scratchpad `ovf/gobench`, Go microbenchmark): a raw int64 add costs 0.31 ns/op and a
  checked add 1.40 ns/op (+1.1 ns). A raw mul costs 1.08 ns/op and a checked `bits.Mul64` mul
  1.22 ns/op (+0.14 ns). The signed-mul check passed an 8-case table including
  `-1 * MIN` and `3037000500²`.
- **Measured.** A 5M-iteration loop with about 5 int ops per iteration: `--strict-bytecode` takes
  0.48 s (about 96 ns/iteration including startup); the evaluator takes 29.36 s.
- **Estimate.** On the VM that is ≤ +1.1 ns on about 19 ns/op, i.e. ≤ ~6% on a pure-arithmetic
  loop, an upper bound since startup is included in the denominator. It is noise on the evaluator.
  The acceptance gate is success metric 5, measured with `bench/` after Phase 1.

## Conflict Surface

### Semantic positions touched
- The result of int `+ - *`, unary `-`, `abs_Int`, `MIN / -1` and `floatToInt` when the
  mathematical result is out of range, on every route in the seam table.
- The verifier evidence's `numeric_model` string for int.
- Go codegen's emission of int `+ - *` (it becomes type-directed).

### What else lives here
- **Float `+ - * /`** shares `mapIntrinsicOp`, the VM `arith` switch, the binop shim and the
  `Num`/`Fractional` dictionaries. Float must stay IEEE and unchanged (`examples/float_nan.ail`).
- **Bit ops** `& ^ ~ << >>` and `shiftRightLogical` are separate intrinsics (`core.OpBitwise*`,
  `OpShiftLeft/Right`, V15). They keep pattern semantics (D6). `1 << 63 == MIN` is unchanged.
- **Polymorphic `Num a`** arithmetic (`examples/poly_arith_lambda.ail`) goes through the dictionary.
  The int instantiation traps; the float one is unchanged.
- **Literals.** A decimal literal above MAX is already an error (teaching prompt). Hex literals with
  the top bit set are two's-complement patterns, are not arithmetic, and are unchanged.
- **Contracts.** `requires`/`ensures` expressions are evaluated by the same evaluator, so an
  overflow inside a contract expression reports RT010.
- **`stringToInt`** already returns `None` on out-of-range input and is unchanged.
- **No constant folding** exists in the Go pipeline (V16), so there is no compile-time arithmetic to
  keep in step.

### Programs that MUST still work (outputs unchanged)
- `tests/binops_int.ail` → `14`
- `examples/float_nan.ail` → its 18 output lines unchanged
- `examples/poly_arith_lambda.ail` → `5.85 7 8 5.0 42 3.0 [2, 4, 6] [3.0, 5.0] 24 2.0 [6, 10]` (one per line, run at HEAD)
- `examples/runnable/fnv1a.ail` and `examples/runnable/splitmix64.ail` → byte-identical **after** the Phase 3 migration
- `benchmarks/audit_chain_replay` → `17710` twice

### What deliberately changes
- A silent wrong number becomes `RT010: integer overflow in <op> at <pos>`, everywhere.
- `floatToInt` of NaN or an out-of-range value: a saturated or zero value becomes RT010.
- Generated Go: int arithmetic becomes helper calls, and an overflow becomes a panic carrying RT010.
- Programs relying on wraparound `*`/`+` must move to `wrap*`. Phase 0 lists them all.

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | Evaluator wraps `MAX+1`, `MAX*2`, `-MAX-2`, `3037000500²`, `MIN / -1` | `ailang run benchmark/ovf.ail` | `-9223372036854775808`, `-2`, `9223372036854775807`, `-9223372036854775808`, `-9223372036854775808`. Confirmed |
| V2 | VM agrees with the evaluator | same program under `--strict-bytecode` | identical five lines. Confirmed |
| V3 | Go codegen wraps too | `ailang compile --emit-go` of `sim/arith.ail`; Go test calling `arith__AddOne(MAX)`, `arith__MulTwo(MAX)`, `arith__Divv(MIN,-1)` | `-9223372036854775808 -2 -9223372036854775808`. Generated code calls `dict_Num_Int.Add` → `toInt64(a) + toInt64(b)`. Confirmed |
| V4 | Int `/ 0` and `% 0` already trap with RT001 on both engines | `ailang run` (± `--strict-bytecode`) on `7 / z`, `7 % z` | `RT001: integer division by zero at …:6:18`, VM `… op DIV`, `… op BUILTIN_CALL`. Confirmed |
| V5 | cp_60149's overflow is reachable | `ailang run benchmark/pw.ail` | `power(30,12)=531441000000000000`, `power(30,13)=-2503514073709551616`, `power(30,13)-1 >= 1e18` → `false` (Python: 30¹³ = 15943230000000000000). Confirmed reachable; **causality for the 6 WA not proven** |
| V6 | Z3 verifies `inc`; runtime violates it; `--verify-contracts` catches it | `ailang check --verify benchmark/contract.ail`; `ailang run` (± `--verify-contracts`) | `"status": "verified"`, `numeric_model {float_sort: Real, int_sort: Int, runtime_equivalence: not_established}`; run prints `-9223372036854775808`; `--verify-contracts` → `contract violation: ensures failed … (result > x)`. Confirmed |
| V7 | Conversions | `ailang run benchmark/conv.ail`, `benchmark/nan.ail` (± `--strict-bytecode`) | `stringToInt("99999999999999999999") = None`; `floatToInt(1e30) = MAX`, `floatToInt(-1e30) = MIN`, `floatToInt(NaN) = 0`; `0 - MIN = MIN`. Confirmed on both engines |
| V8 | Wrap-reliant code in repo: two examples, nothing in benchmarks/std | `grep` for 64-bit hex constants and hash names over `benchmarks/ examples/ std/`; read hits; `ailang run` both examples | `fnv1a.ail:22`, `splitmix64.ail:13,14,19` multiply by 64-bit constants and print negative hashes (wrapping). `gauntlet_10` hit was the word "murmur" in test data; `audit_chain_replay` = `fib_sum(20)`, no overflow. **Census is grep-level; Phase 0 replaces it with an execution census** |
| V9 | Parked `m-pure-prng` relies on wrapping `*` | read `design_docs/planned/v0_29_0/m-pure-prng.md:38` | "`int` is 64-bit signed; `+`/`*` wrap mod 2⁶⁴". Confirmed; it migrates to `wrapMul` |
| V10 | VM ints are a bare int64 field | `internal/vm/vm.go` `arith`: `l, r := lhs.Int, rhs.Int … bytecode.NewInt(l + r)` | Confirmed (relevant to option D's cost) |
| V11 | Proposed names and code are unallocated | `grep -rn` over `std/ internal/ cmd/` (excluding tests) for `wrapAdd wrapSub wrapMul wrapNeg checkedAdd checkedMul mulMod powMod std/bigint _int_wrap RT010 IntOverflowError` | 0 hits each. `_bigint_` and `addMod` hits are all in vendored `internal/executor/opencode/plugins/node_modules` (TypeScript). Free |
| V12 | No user instances; ADT-over-string gets `==`; `bytes` precedent size | read `internal/parser/parser_func.go:534` (`parseInstanceDeclaration`: `// TODO … return nil`); `ailang run benchmark/bigadt.ail` (± `--strict-bytecode`); `grep -rln "BytesValue\|TagBytes" internal/ \| wc -l` | instances unimplemented; `BigInt("123") == BigInt("123")` → `true`, `… == BigInt("124")` → `false` on both engines; `bytes` touches 32 files. Confirmed |
| V13 | Seam census of int `+ - * neg` sites | (1) `grep -rnE "IntValue\{Value: [^}]*[-+*]…\}\|NewInt\([^)]*[-+*]…\)" internal/` (every Go site that builds an AILANG int from an arithmetic expression: 29 hits; the rest are index/length/position arithmetic plus `effects/clock.go:53`, the epoch+elapsed clock reading, which is not user arithmetic); (2) every `intToInt(`/`intIntToInt(` registration in `internal/builtins` (9: `add sub mul neg double` + 4 bitwise); (3) `wrapInt2(` in `internal/repl`; (4) `x + y` in `internal/types/dictionaries.go`; (5) `mapIntrinsicOp`/`AddInt`/`dict_Num_Int` in `internal/gen/golang` | Exact lines in the seam table (round 1 found `double_Int` and the four unary-`-` evaluator sites missing from the first grep). Static census only; the [route-independent differential test](#testing-strategy) is the acceptance backstop, not this row |
| V21 | `==` on an ADT without `deriving (Eq)` is a compile error | `ailang check`/`run` on `type BigInt = BigInt(string)` + `BigInt("007") == BigInt("7")` | `No instance for Eq[BigInt] in scope. Equality (==, !=) is not defined on BigInt.` Confirmed |
| V22 | `power(d, n)` in V5 is model-written AILANG, not a builtin | read `cp/work_ha3/cp_60149_ailang_claude-haiku-4-5_t1/benchmark/solution.ail:79-81` (`func power(base, exp) = … base * power(base, exp - 1)`); `std/math` exports no int `pow` (only `pow(float, float)`) | Confirmed |
| V14 | No doc states int overflow semantics | `grep -iE "int64\|overflow\|wrap\|64-bit\|bigint"` over `docs/LIMITATIONS.md`, `docs/docs/reference/limitations.md`; `grep -rli "integer overflow\|wraps silently\|two's complement" docs/docs` | Only unrelated hits (brace wrapping, stack overflow, RE2). Confirmed |
| V15 | SMT encodes bit ops as `bv*` on Int terms | read `internal/smt/codegen_operators.go:12-41` | `& → bvand`, `<< → bvshl`, `>> → bvashr` next to `+ → +` (Int). Confirmed; outside this doc's soundness claim |
| V16 | No constant folding of int arithmetic | `grep -rlnE "constant.?fold\|constFold\|foldConst" internal/` | Only JSON fixtures and `models.yml`. Confirmed |
| V17 | Checked-arithmetic overhead | Go microbenchmark `ovf/gobench` (200M iterations) | add 0.31 → 1.40 ns/op; mul 1.08 → 1.22 ns/op; signed-mul check passes 8 cases. Measured |
| V18 | VM vs evaluator per-iteration cost | `/usr/bin/time` on `benchmark/loop.ail` (5M iterations) | VM 0.48 s, evaluator 29.36 s. Measured |
| V20 | Cited regression fixtures exist with the stated outputs | `ls` each; `ailang run` `tests/binops_int.ail`, `examples/poly_arith_lambda.ail`, `examples/float_nan.ail` | all exist; `14`; the 11 lines quoted; 18 lines. Confirmed |
| V19 | cp_60166 trial outcomes and mechanisms | `cp/grade_ha3.json`, `cp/grade_c.json`; read the 3 valid AILANG `.ail` files and 2 Python `sol.py` | Python 4/4 trials at 11/11 small (1/6 big, TLE); AILANG valid trials 2/11 WA, 10/11 (+1 TLE), 2/11 WA; one void (`api_error`, no code). Mechanisms as in the table. Confirmed |

## Testing Strategy

- **Route-independent differential test (the backstop for V13; quorum round 1).** The seam table
  is a grep census, and Phase 0's record mode instruments only the seams it lists, so neither can
  find a seam nobody listed. This test does not use the table. It generates AILANG programs whose
  expressions mix `+ - *`, unary `-`, `abs_Int`, `/`, `floatToInt` and the `std/math` int
  functions, with operands drawn from boundary values (`MAX`, `MIN`, `±1`, `0`, `3037000499`,
  `3037000500`, `2^32`) and random int64s. It then runs each one through every public entrypoint
  (`ailang run`, `--strict-bytecode`, the REPL, `serve-api`, compiled Go). Ground truth is
  arbitrary-precision evaluation of the same expression tree in the Go test (`math/big`): the
  expected output is the exact value, or RT010 iff some intermediate leaves int64. Any route that
  prints a wrapped value fails, **whichever Go function produced it**. The generator also emits
  each operator as a direct builtin call (`add_Int(a, b)`, `double_Int(a)`), so builtins reached by
  name are covered too.
- **Seam table test** (the div-zero doc's shape): one table of `(op, a, b)` overflow cases ×
  {evaluator, `--strict-bytecode`, REPL, codegen}. Each must report RT010 with the same text,
  except codegen, which is a panic carrying RT010. Plus the boundary cases that must **not** trap:
  `MAX + 0`, `MIN + 0`, `MIN % -1 = 0`, `-MAX`, `3037000499²`.
- **Mutation test** (memory: mutation-test your own tests). Revert one seam to raw `+`; the table
  test must fail for that route.
- **Verifier.** A golden test asserts the `inc` evidence: `verified`, plus the new int
  `runtime_equivalence` value. A float program keeps `not_established`.
- **`std/bigint`.** Property tests against Go `math/big`: add/sub/mul/divMod/compare on random
  canonical strings; `parse` rejects `"007"`, `"-0"`, `""`, `"1e3"`.
- **Examples.** `make verify-examples` stays green after the fnv1a/splitmix64 migration, with
  byte-identical output.
- **Perf gate.** `bench/` int-loop delta on the VM ≤ 5% (success metric 5).

## Success Criteria

- [ ] Phase 0 census pasted into this doc; every hit migrated or filed
- [ ] RT010 on all seam-table routes; the boundary cases do not trap
- [ ] `inc` evidence carries the new int `runtime_equivalence` value; float is unchanged
- [ ] `std/bigint` shipped with property tests; a `cp_60166` reference AILANG solution passes the small hidden tests
- [ ] `wrap*`, `checked*`, `mulMod`, `powMod` in `std/math`; `make freeze-stdlib` run (the pre-push stdlib gate)
- [ ] fnv1a/splitmix64 byte-identical after migration
- [ ] Teaching prompt + LIMITATIONS updated, with syntax verified by `ailang check`
- [ ] Changelog fragment `changelogs/unreleased/<date>-int-overflow-trap.md` (`### Changed` breaking + `### Added`)
- [ ] `examples/int_overflow.ail` and `examples/bigint_basics.ail`

## Deferred Decisions

- The exact `runtime_equivalence` string, and whether it is a v1 field value or a v2 bump. Agent
  decides after grepping consumers (`internal/verification`, dashboard, eval banking).
- The name of the total `floatToInt` variant (`floatToIntOpt` vs `tryFloatToInt`). Agent decides.
- Whether `std/bigint` also exports `gcd`, `isqrt` and `fromString` aliases. Agent may add them if
  cheap and tested.
- How the RT010 message names the operation (`addition`/`multiplication`, mirroring
  `DivByZeroError.Op`). Agent decides.

## Non-Goals

- **Bit-op and shift semantics** (D6) and their SMT encoding: that is
  [m-smt-well-sorted-preflight](../m-smt-well-sorted-preflight.md).
- **`div`/`mod` truncation vs Z3 Euclidean**: m-smt-well-sorted-preflight S1.
- **Float** overflow and IEEE semantics: unchanged.
- **Operators on `BigInt`** (`a + b` for BigInt): needs user instance declarations, which are
  unimplemented. Future Work.
- **Auto-promotion** (option D): deferred until measured.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Unknown wrap-reliant code (packages, demos, the world repo) breaks on upgrade | High | Phase 0 execution census over everything in-repo; a `### Changed` breaking changelog entry naming `wrapMul`; RT010's hint text points at `wrapMul` and `std/bigint` |
| Models writing hash code (FNV, SplitMix) now trap instead of passing | Med | Teaching-prompt rule (Phase 3) lands in the **same release** as the trap; the RT010 hint names the fix |
| A seam is missed and engines drift (the exact pattern RT001 fixed) | Med | One helper; the seam table test is parameterized by route; mutation test |
| VM slowdown above 5% | Low | Measured +1.1 ns/add against about 19 ns/op. If over the gate, the VM can use `bits.Add64`'s carry form or check only in `arith`'s int arm |
| A string-backed BigInt is too slow for some workload | Low | Measure first; a native value kind is the Future Work escape hatch, with the API unchanged |

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | Overflow is a deterministic, positioned error identical across engines, where today engines agree only on a wrong number. No flag-dependent semantics (option C rejected) |
| A2: Replayability | 0 | Traces are unchanged in shape; an RT010 replays identically |
| A3: Effect Legibility | 0 | No effects added; `std/bigint` is pure |
| A4: Explicit Authority | 0 | No capabilities involved |
| A5: Bounded Verification | +1 | Z3 `Int` proofs over `+ - * neg abs` become sound modulo trap: a verified `ensures` cannot be violated by a returned value |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | A model gets a coded error with a fix hint instead of a plausible wrong number it cannot detect (`cp_60166`) |
| A8: Minimal Syntax | 0 | No syntax added; library functions only |
| A9: Cost Visibility | 0 | `std/bigint` op costs documented in the module header (the `std/map` cost-model style) |
| A10: Composability | +1 | `wrap*`/`checked*`/`std/bigint` are ordinary pure functions; the wrap intent becomes visible in the source |
| A11: Structured Failure | +1 | RT010 is a typed error in the RT001 style; `checked*`, `toInt` and `parse` return `Option` |
| A12: System Boundary | 0 | No boundary changes |

**Net score: +5.** No −1 on A1/A3/A4/A7. Proceed, subject to F1–F3.

## Related Documents

- [m-int-div-zero-error](../../implemented/v0_52_0/m-int-div-zero-error.md) (v0.52.0): the precedent (shared helper, route table, RT001)
- [verify-int-width-mismatch-z3-int-vs-int64](../ailang-core-triage/verify-int-width-mismatch-z3-int-vs-int64.md) (triage, 0.47): this doc answers it from the runtime side. **Distinct**: the triage weighed only verifier-side remedies
- [int-div-zero-go-panic](../ailang-core-triage/int-div-zero-go-panic.md) (triage, 0.42): the predecessor row of the precedent
- [m-runtime-contract-parity](../m-runtime-contract-parity.md): its ledger lists "Z3 `Int` vs runtime `int64` overflow" as the one expected divergence; this doc removes it for `+ - * neg abs`
- [m-smt-well-sorted-preflight](../m-smt-well-sorted-preflight.md): owns bit-op sorts and division semantics
- [m-pure-prng](../v0_29_0/m-pure-prng.md) (parked): relies on wrapping `*`; migrates to `wrapMul`
- Sibling docs from the same 2026-10-28 competition-problem spike: [m-persistent-collections](m-persistent-collections.md) and [m-vm-coverage-letrec-prelude-bridge](m-vm-coverage-letrec-prelude-bridge.md)

## Future Work

- **Native `BigInt` value kind with operators.** Only once user `instance` declarations exist and a
  measurement shows the string-backed library is the bottleneck.
- **Auto-promotion (option D)** if models still lose exactness deltas with `std/bigint` available.
- **A `checked`/`wrapping` lint** suggesting `wrapMul` when a hash-shaped loop traps in tests.

## Quorum log

**Round 1 (2026-10-28T09:31:33Z): BLOCKED.** $0.2369. Artifact:
`.ailang/state/mission-quorum/m-int-overflow-trap-and-bigint-2026-10-28T09-31-33Z.json`.

- **Seated reviewers:** oc-kimi-k3 **reject**, gemini-4-argon **reject**, oc-glm-5-3 (reserve)
  **reject**. gpt6-1-sol was **ABSENT** (unreachable), so the quorum degraded to N−1. Controller
  (in-session) **pass**.

| Objection | Reviewer(s) | Accepted? | Revision |
|---|---|---|---|
| Exported `BigInt(string)` constructor + `deriving (Eq)` lets `BigInt("007") == BigInt("7")` be silently false, and `toString(BigInt("x"))` return malformed data | oc-kimi-k3, gemini-4-argon | **Yes** (correct) | `deriving (Eq)` dropped. `==` on `BigInt` is now a compile error (V21); `bigint.eq` is the only equality. Every builtin, including `toString`/`eq`/`compare`, rejects non-canonical input |
| The seam census (V13) is grep-only, and the Phase 0 record mode can only see listed seams, so an unlisted Go-side arithmetic site would ship wrapping | oc-glm-5-3 | **Yes** (correct, and the re-grep proved it: `double_Int` and 4 unary-`-` sites were missing) | V13 widened to exact lines from five independent sweeps; added a route-independent differential test against `math/big` ground truth as the acceptance backstop; V22 pins `power(d, n)` as model-written code |

**Not re-quorumed** in this session. The author ran round 1 once as directed. The revision answers
both objections, and **round 2 is the next step before planning.**

**Trigger 3 also fires** (D5 changes the banked `numeric_model` string). It was not separately
argued in round 1.

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28
