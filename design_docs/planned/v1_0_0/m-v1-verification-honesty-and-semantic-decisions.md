# M-V1-VERIFICATION-HONESTY-AND-SEMANTIC-DECISIONS: fail loudly, then write the rules down before the freeze

**Status**: Planned
**Target**: v0.58.x (the last breaking window before v1.0.0)
**Priority**: P0 for M1 and M5 (a Stable command passes silently; the verifier proves claims the runtime breaks). P1 for the rest
**Estimated**: about 15–19 working days in total. This is an estimate, not a plan; the per-milestone sizes are in [Milestones](#milestones)
**Dependencies**: [m-int-overflow-trap-and-bigint](../v0_55_0/m-int-overflow-trap-and-bigint.md) (M5 executes it; its quorum round 2 is still pending), [m-runtime-contract-parity](../m-runtime-contract-parity.md) (M2 is its M1), [m-effect-latent-function-values](../v0_48_0/m-effect-latent-function-values.md) Phase 3 (M6c). Sibling: `m-v1-nominal-type-identity.md` (being written in parallel), which proposes the soundness clause for `stability.md`
**Planner-Lane**: opus-required (parser, type checker, runtime arithmetic, and a Stable exit-code contract)
**Source**: [v1 release-readiness review §3.4](../v1-release-readiness-review-2026-11-09.md#34-smaller-semantic-decisions)
**Lane** (PROGRAM.md): AILANG fix. None of these can be an extension: they are exit codes, parse rules, typing rules and arithmetic semantics.

All behaviour below was reproduced on a build of `dev` at `51f6636ae` (`AILANG dev`, built 2026-11-09)
with Z3 4.16.0. The probe files are in the session scratchpad under `dd-semantics/` and `sec/`.
Commands and outputs are in the [Verification Log](#verification-log).

## Problem Statement

Under the [1.x Stability Promise](../../../docs/docs/reference/stability.md), any change that makes
a previously accepted program fail, or changes a Stable command's observable behaviour, needs 2.0.
The six items in review §3.4 each change either an exit code of a Stable command (`check`, `run`,
`test`) or the meaning of Stable syntax (`::`, function types, `+`). They are cheap in v0.58.x and
a major version afterwards.

Part 1 is four bugs with a clear right answer. Part 2 is two semantic rules that are currently
unwritten, or written in a way that contradicts what the checker does.

**Who is affected:** AI agents and CI. Both read the **exit code** first and the JSON second. Three
documents in this repo record the same incident: `check --verify` / `ai-check` passing while
verifying nothing (`docker/Dockerfile.agent-base:44` says the ailang_only lane "type-checked only,
silently, until 2026-09-16"; `.claude/skills/cloud-setup/resources/troubleshooting.md:58`;
`docker/test-agent-pi.sh:95`). Each was worked around locally. None fixed the exit code.

## Goals

**Primary goal:** every outcome of a verification or contract command is visible in its exit code,
and the two unwritten semantic rules (integer overflow, unannotated callback rows) are written down,
implemented and frozen before v1.0.0.

**Success metrics:**
1. `AILANG_Z3_PATH=/nonexistent ailang check --verify c1.ail` exits **non-zero**. Today it exits 0.
2. Silent-skip paths in the contract and verification surface go from 4 to 0: Z3 missing; VM
   contracts; inconclusive passing by default; named-test bodies ignoring contracts (if D3 is accepted).
3. `x :: xs` passes `ailang test` in every position where `ailang run` accepts it, and a parse →
   print → parse round-trip test covers every expression in `examples/`.
4. `9223372036854775807 + 1` raises RT010 on the evaluator, the VM and the REPL, and the `inc`
   program in M5 no longer prints a negative number after `check --verify` reported `verified`.
5. One sentence in the teaching prompt states what `T -> U` means, and the checker agrees with
   it on all 12 probes in [§6](#6-what-an-unannotated-callback-type-means).

## High-Impact Decisions

| # | Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|---|
| D1 | `check --verify` exit contract: inconclusive **fails** by default, skipped **passes** with a loud summary, and a missing solver fails when there is at least one obligation | Stable exit code; every agent loop and CI gate keys on it | human | design | high |
| D2 | `--bytecode` + `--verify-contracts` is a refusal (CTR004) until the VM enforces contracts | Turns a silent no-op into an error | human (already recommended in parity doc F3) | design | low |
| D3 | Run-time contracts stay **off** by default in `run`, and come **on** in `ailang test` named-test bodies | Changing either default after 1.0 is breaking | human | design | high |
| D4 | `::(x, xs)` in **expression** position is not syntax: drop it from the prompt and from the strict-mode hint | Documented syntax that never parsed | human | design | low |
| D5 | Integer `+ - * neg abs` and `MIN / -1` **trap** (RT010); wrapping is explicit (`wrapAdd`…) | Changes the result of existing programs; defines what `verified` means | human (F1 of the overflow doc) | design | high |
| D6 | Unannotated `T -> U` in a parameter is an open row that is **rigid in the body**; concrete rows are upper bounds | Decides which definitions type-check; makes signatures the contract | human | design | high |
| D7 | Parser rejects non-identifier parameter tokens; `()` is sugar for `_: ()` | Closes an untyped-parameter hole found while reproducing #2156 | agent | compile | low |

### Design Freeze

- [ ] D1: the exit-code table in §1, including whether "not proved" gets its own exit code (recommended: no)
- [ ] D3: contracts on in `ailang test` named-test bodies
- [ ] D5: trap by default. This is F1 of the overflow doc, so its quorum round 2 must run first
- [ ] D6: the callback rule, and whether the upper-bound half (M6c) lands in v0.58.x or under the soundness clause
- [ ] D4: retire the `::(x, xs)` expression spelling

---

# Part 1: fail loudly

## 1. `ailang check --verify` exits 0 when it has proved nothing

**Reproduced.** `c1.ail` is `half(x) = x - 100` with `requires { x >= 0 } ensures { result >= 0 }`.
Fake solvers are in `sec/fakez3_{fail,hang,unknown}`.

| Outcome | How produced | `check --verify` (= `ai-check`) | `verify` | `verify --strict` |
|---|---|---|---|---|
| verified | sound contract | 0 | 0 | 0 |
| counterexample | `c1.ail`, real Z3 | **1** | 1 | 1 |
| solver error | `fakez3_fail` | **1** (`errors: 1`, `verifier_error`) | 1 | 1 |
| solver `unknown` / timeout | `fakez3_unknown`, `fakez3_hang --verify-timeout 1s` | **1** (`errors: 1`, `solver_unknown` / `solver_timeout`) | 1 | 1 |
| inconclusive (bounded unrolling) | `r1.ail`, recursive `sumTo` with `ensures { result >= 100 }` | **0** | 0 | 1 |
| skipped (outside the fragment) | `s2.ail` (HOF param), `s3.ail` (`! {IO}` function) | **0** | 0 | 1 |
| **solver unavailable** | `AILANG_Z3_PATH=/nonexistent` | **0**, `"available": false`, `results: []` | **1**, "Z3 solver not found" | 1 |
| check failed | type error | 1 (and `available: false`, which reads as "no solver") | 1 | 1 |

Two corrections to the review. Solver `unknown` and timeout already fail, because they are counted as
`errors`. And `ailang verify` fails loudly only when Z3 is missing; on skipped and inconclusive it
also exits 0 unless `--strict` is passed. `check --verify` has no `--strict`.

`r1.ail` shows why inconclusive must not pass. `sumTo(0) = 0` violates `result >= 100` without
touching the recursion frontier, yet it is reported `bounded_unrolling_inconclusive`, and the exit
code is 0.

**Root cause.**
- `cmd/ailang/ai_check.go:168`: `if checkSection.Passed && smt.Z3Available() { … }`. With no Z3 the
  verify section stays at its zero value, `Available: false`, and nothing records that obligations
  were dropped.
- `cmd/ailang/ai_check.go:214`, `aiCheckExitCode`: `!check.Passed || Counterexample > 0 || Errors > 0`.
  `Available`, `Inconclusive` and `Skipped` are not read. The comment above it documents
  "skipped-only → 0" and "inconclusive-only → 0" as deliberate.
- `cmd/ailang/verify.go:72` is the contrast: it refuses up front when Z3 is missing, and
  `verify.go:149` applies `--strict`. The two commands share `smt.Verify` (S4 M3A) but not an exit
  function, so they drifted.

**Proposed contract (D1).** One exit function, `verification.ExitCode(report, strict)`, shared by
`check --verify`, `ai-check`, `verify` and `verify --package`.

| Outcome | Exit | `verify.outcome` (new JSON field) |
|---|---|---|
| check failed | 1 | `not_run` (and `available` keeps meaning "solver found") |
| no contract obligations in the file | 0 | `no_obligations`, even with no solver |
| all obligations verified | 0 | `verified` |
| only verified + skipped | 0, **plus a stderr line** `N contract(s) not verified: f (has effects), …`; 1 with `--strict` | `partial` |
| any inconclusive | **1**; 0 with `--allow-inconclusive` | `inconclusive` |
| any counterexample | 1 | `counterexample` |
| any solver error, unknown or timeout | 1 | `error` |
| obligations exist and the solver is unavailable | **1**, stderr names the install command | `unavailable`, plus `obligations: N` |

Why skipped passes and inconclusive fails: a skip is a stated boundary of the decidable fragment
("has effects", HOF parameter). Failing it by default would make the eval agent prompt's "only finish
once BOTH ai-check passes AND output matches" (`internal/eval_harness/agent_prompt.go:614`)
unreachable for any effectful function with a contract. Agents would learn to delete contracts. An
inconclusive result is the solver's candidate counterexample, and `r1.ail` shows it can be a real
one. Counting obligations without Z3 needs only the Core metadata the verifier already walks; it does
not need the solver.

**Alternatives.** (a) A distinct exit code for "not proved" (for example 3, as `design-quorum` uses).
Rejected as the default: it is one more frozen number, and naive consumers treat every non-zero exit
as failure anyway. The JSON `outcome` carries the distinction. (b) Fail skipped by default. Rejected
for the reason above. (c) Keep `available: false` → 0 and document it. Rejected: that is the silent
fallback CLAUDE.md principle 2 forbids.

**Blast radius (grep of the repo).**
- Eval harness (`internal/eval_harness/verify.go:152,166`) already requires `Available &&
  Counterexample == 0 && Errors == 0` and parses the JSON on a non-zero exit (`:91`), so it is
  unaffected. Inconclusive now also fails, and the harness should bank it as `verify_ok=false`.
- **Hosted executor `ailang_check` (a Stable MCP tool)**: `docker/Dockerfile.executor-mcp` runs on
  `distroless/static` with no z3, and `executor_tools/executor.ail:216` calls `ai-check`. So the
  hosted tool most likely never verifies anything. This is inferred from the Dockerfile; the live tool
  could not be called (the MCP session needed re-authentication). After M1, contracted code sent to
  it exits 1 with `outcome: unavailable` until the image ships z3. M1 includes adding it.
- `docker/Dockerfile.agent-base:54` and `.github/workflows/ci.yml:150` already install z3.
  `docker/test-agent-pi.sh:99` asserts `available: true` and keeps passing.
- Skills: `cloud-setup` troubleshooting ("passes but verified nothing") becomes stale and is deleted.
  `mission-control` uses `ai-check` as a done-gate and becomes stricter, as intended.
- `cmd/ailang/ai_check_exit_test.go` pins the old lanes and is rewritten (coding standard: delete
  out-of-date tests).

## 2. Run-time contracts: silent on the VM, nameless on the evaluator, off by default

**Reproduced** with `k1.ail` (`half(5)` violating `ensures { result >= 0 }`):

```
ailang run --caps IO --entry main k1.ail                                     -> -95, exit 0
ailang run --verify-contracts --caps IO --entry main k1.ail                  -> Error: execution failed:
    contract violation: ensures failed in  at k1.ail:5:11: (result >= 0), exit 1
ailang run --bytecode --verify-contracts --caps IO --entry main k1.ail       -> -95, exit 0   (#1956)
ailang run --strict-bytecode --verify-contracts --caps IO --entry main k1.ail -> -95, exit 0
ailang test kt.ail   (test "half five" { half(5) == -95 })                    -> ✓ passes
```

**Root causes.**
- VM: `internal/runner/run.go:537-538` installs `effCtx.Contracts` in panic mode, and nothing on the
  VM path reads it. The full audit (0 of 22 constructs checked on the VM) is in
  [m-runtime-contract-parity](../m-runtime-contract-parity.md#problem-statement).
- Empty name: `ContractContext.SetFunction` (`internal/effects/contracts.go:110`) has **no production
  caller**, only tests (`git grep 'SetFunction('`). `eval.ContractSpec` (`internal/eval/value.go:370`)
  has no function field, and `FunctionValue` has no name. The name is in scope where contracts are
  attached (`internal/runtime/runtime.go:366-370` and `:390-393`, `attachContracts(fn, …)` with `name`
  / `e.Name` in hand) and is dropped there. Trace `contract_check` events
  (`internal/effects/context.go:651,675`) carry the same empty `CurrentFunction()`.
- Named tests: no contract context is installed for `test` bodies (`internal/testing` has no
  `ContractMode` setup). Contracts reach `ailang test` only as generated property tests (kt.ail's
  `half_property_1` did catch `x=-457`).

**The default.** Contracts are not checked by `ailang run` unless `--verify-contracts` is passed.
This is documented in `docs/docs/guides/contracts.mdx:68-71` ("Checked APIs should guard bad inputs
in ordinary execution rather than rely on that opt-in flag") and implied by the teaching prompt
("Runtime contract checking (alternative to static verification)"). It is not stated in
`run --help`, and the guide's example message (`panic: contract violation: requires: …`) no longer
matches the real one.

**Recommendation for 1.0 (D3).**
- **Keep `run` default-off and freeze it.** Turning it on before the VM enforces contracts would make
  every `--bytecode` run of a contracted program refuse (after M2a), and it would add predicate
  evaluation to every call on hot paths. Turning it on in 1.x would be breaking, so say "off" in
  `stability.md` deliberately rather than by omission.
- **Turn contracts on in `ailang test` named-test bodies.** A test is the place where a violated
  `ensures` should fail. Today `test "half five" { half(5) == -95 }` passes against a function whose
  contract forbids `-95`. This makes some passing tests fail, so it is v0.58.x or never.
- M2a refuses `--bytecode`/`--strict-bytecode`/`--batch` with `--verify-contracts` (CTR004, before
  execution), which is the parity doc's M1, verbatim. VM enforcement (that doc's M6–M7) lifts the
  refusal later, which is additive.
- M2b names the function: `contract violation: ensures failed in half at k1.ail:5:11: (result >= 0)`.

**Who breaks.** M2a: anyone passing both flags, who gets no checking today. `run --bytecode` is an
Experimental flag. M2c: test suites with a test that exercises a contract violation and asserts the
violating value. Count with the downstream gate (review §4.3 step 4) before landing.

## 3. `::` does not parse inside a `test` body (#1914)

**Reproduced** with the issue's `r.ail`. `ailang check` → no errors; `ailang test` →
`PAR_NO_PREFIX_PARSE at r.ail:5 (test body): unexpected token in expression: ::`. The minimal
`test "c" { 1 :: [2] == [1, 2] }` fails the same way.

**Root cause.** Named tests are not compiled from their source text. `internal/testing/named_batch.go:123`
re-prints each folded body with `PrintAILANGSource` and re-parses the result as a synthetic
`pure func`. The parser desugars `x :: xs` to `FuncCall{Func: Identifier("::"), Args: [x, xs]}`
(`internal/parser/parser_expr.go:497-501`). The printer's generic `*ast.FuncCall` case
(`internal/testing/executor_helpers.go:318-335`) prints that as `::(x, xs)`, **and `::(x, xs)` does
not parse in expression position** (`pure func f() -> [int] = ::(1, [2])` → `PAR_NO_PREFIX_PARSE`).
The prefix spelling parses only as a pattern (`match xs { ::(x, rest) => … }` checks clean).

The same root cause produces two more defects:
- The teaching prompt's syntax table (`List cons | x :: xs or ::(x, xs)`, prompt line 443) teaches an
  expression form that has never parsed.
- `--strict-syntax` rejects `x :: xs` and suggests `::(x, xs)` (`parser_expr.go:478`), so under strict
  syntax a cons expression cannot be written at all.

This is the fourth defect from the same re-printing seam (#590 `assert`, #1448 float literals,
#1821, #1914). CLAUDE.md principle 3 asks for the systemic fix.

**Proposed.** (M3a) Print a two-argument `::` call as `(a :: b)`. (M3b) Add a round-trip property test:
for every expression in `examples/**/*.ail` and `std/*.ail`, `parse(print(e))` is AST-equal to `e`.
It would have caught all four bugs. (M3c, D4) Remove the `::(x, xs)` expression spelling from the
prompt and the strict-mode hint, and make strict mode accept infix `::`. Removing documented syntax
that never parsed breaks nobody. The alternative, adding a prefix parselet for `::(`, adds a second
spelling of one thing (A8).

## 4. #2156: IFC export analysis rejects `func f(())`, and treats private functions as exports

**Reproduced.** `module u1; func f(()) -> int = 1; export func main() -> () ! {IO} = ()` →
`interface build error in u1: IFC export analysis: export f: missing IFC parameter name`. `f` is not
exported. `func f(u: ())` checks clean.

**Root cause, and a correction to the issue.** `f(())` is not a unit pattern: AILANG has no pattern
parameters. `parseParams` (`internal/parser/parser_lambda.go:174-183`) builds an `ast.Param` for
whatever token it sees and fills in `Name`/`Type` only when that token is an identifier. Any other
token becomes a **nameless, untyped** parameter. On a v0.47 build (`ailang.dev-backup-20260928`),
`func f(5) -> int = 1` checks clean and `f("hello")` prints `1`, so a parameter written as a literal
accepted a string. The regression only made this leniency visible. `types.AnalyzeModuleIFC`
(`internal/types/ifc_summary.go:173-199`) loops over **all** of `file.Funcs`, not the exports, and
`Validate` (`:101`) rejects the empty name. The error string hard-codes "export". The same build fails
on v0.57.1 and the current HEAD-ish binary.

**Proposed (M4).**
- Parser: a parameter that is not `IDENT [: Type]` is `PAR_PARAM_NAME` ("parameter must be a name;
  write `_: ()` for an ignored unit parameter"). The one exception is `()`, which becomes the same node
  `newUnitParam` builds (`_: ()`, `parser_func.go:335`). That keeps stapledons' `func f(())` working
  and closes `func f(5)`. Lambda `func(...)` literals share `parseParams` and get the same rule.
- IFC: the summary keys parameters by position (the code comment already says names are "not
  security information"). The error reads `function f` for non-exported functions. Whether
  non-exported functions need a persisted summary at all is left to the agent (Deferred).

**Who breaks.** `func f(5)`-style parameters: already rejected since the regression, so no
v0.57.4 program that checks today starts failing. `func f(())`: fixed.

---

# Part 2: decide and write down

## 5. Integer overflow, division and float edges

**Current behaviour** (`ar.ail`, `dz.ail`, `fz.ail`; the evaluator and `--strict-bytecode` agree on every line):

| Expression | Result | Expression | Result |
|---|---|---|---|
| `MAX + 1` | `-9223372036854775808` (wraps) | `7 / 0`, `7 % 0` | RT001, both engines |
| `MAX * 2` | `-2` | `-7 / 2`, `-7 % 2` | `-3`, `-1` (truncating) |
| `MIN - 1` | `MAX` | `7 / -2`, `7 % -2` | `-3`, `1` |
| `0 - MIN` | `MIN` | `MIN / -1`, `MIN % -1` | `MIN` (wraps), `0` |
| `floatToInt(1.0e30)` | `MAX` (saturates) | `floatToInt(NaN)` | `0` |
| `1 << 64`, `1 << 63` | `0`, `MIN` | `x << -1` | `[RT_SHIFT] negative shift amount` |
| `-1 >> 1` | `-1` (arithmetic) | literal `9223372036854775808` or `-9223372036854775808` | parse error (MIN needs `0x8000000000000000`) |
| `1.0 / 0.0`, `-1.0 / 0.0` | `Inf`, `-Inf` | `0.0 / 0.0`, `5.0 % 0.0` | `NaN` |
| `-7.5 % 2.0` | `-1.5` (sign follows dividend) | `0.0 * -1.0` | `-0.0` |
| `1.0e308 * 10.0` | `Inf` | `intToFloat(9007199254740993)` | `9007199254740992.0` |

**Does wrap make Z3-verified contracts unsound? Yes, measured.** The SMT encoding gives `int` the
unbounded sort `Int` (`numeric_model.int_sort: "Int"` in every evidence block). `inc(x) = x + 1` with
`requires { x >= 0 } ensures { result > x }` is reported `"status": "verified"` by
`check --verify` (exit 0). `ailang run` with `inc(MAX)` prints `-9223372036854775808`, and
`run --verify-contracts` reports `ensures failed`. The evidence says `runtime_equivalence:
"not_established"`, but agents read `verified`. Division is already encoded with truncation
semantics (`internal/smt/codegen_operators.go:105-134`, #1960), so `/` and `%` are not part of this gap.

**Options.** (a) Documented two's-complement wrap: honest, cheapest. `verified` stays unsound unless
the encoder moves to 64-bit bit-vectors, which makes currently provable goals much harder. (b) Trap
(RT010) on `+ - * neg abs`, `MIN / -1` and out-of-range or NaN `floatToInt`, with explicit
`wrapAdd/wrapSub/wrapMul/wrapNeg`. A Z3 `Int` proof then holds for every execution that returns a
value (partial correctness). (c) Checked by default with an opt-out flag: one program, two semantics
(A1); rejected. (d) Auto-promote to bignum: the closest match to Z3 `Int` and to what AI authors
expect from Python, but it changes the value representation on every engine.

All four are deterministic. Performance is measured in the overflow doc: a checked add costs +1.1 ns
over 0.31 ns in Go, an upper bound of about 6% on a pure-arithmetic VM loop, and noise on the
evaluator. WASM is the same Go interpreter (`cmd/wasm`), so it gets the semantics for free. Go codegen
needs type-directed int emission.

**Recommendation: (b), trap.** It is the RT001 precedent (division by zero already traps), it is the
only cheap option that makes `verified` mean what agents read it as, and an AI author gets a
positioned error with a fix hint instead of a plausible wrong number. That is the `cp_60166` evidence
in the overflow doc. Scope for v0.58.x: that doc's Phase 0 (census), Phase 1 (RT010 everywhere) and
the `wrap*`/`checked*` half of Phase 3. **`std/bigint` (Phase 2) is additive and can ship in 1.x.**
Bit operations keep bit-pattern semantics (D6 there).

**Blast radius.** Repo grep for 64-bit constants and hash names: `examples/runnable/fnv1a.ail` and
`examples/runnable/splitmix64.ail` rely on wrap. `examples/integer_literals.ail` only uses hex literals.
`benchmarks/frontier_refs/stream_lcg_topk.ail` multiplies by 48271 under mod 2^31−1, which does not
overflow. `gauntlet_10` matches only the word "murmur". No `std/*.ail` arithmetic relies on wrap. This
is a grep, not an execution census. Phase 0's record-mode run over `make verify-examples`, the
benchmark references and the downstream repos (docparse, daneel, world, the motoko fork,
`ailang-packages`) is a gate before the trap merges.

**Frozen table.** Once D5 is ruled, the table above (with trap rows replacing wrap rows) goes into
`docs/docs/reference/language-syntax.md` as "Numeric semantics" and is referenced from `stability.md`.

## 6. What an unannotated callback type means

**The documented rule.** Teaching prompt (line 461): "`T -> U`: Implicit open row; hand-off charges
the caller". LIMITATIONS (line 100): "a concrete `! {IO}` callback mapped with closed-pure
`std/list.map` is a type error; use `mapE`". LIMITATIONS (line 25): concrete upper bounds are
unenforced (Phase 3 of M-EFFECT-LATENT-FUNCTION-VALUES).

**Probes** (`dd-semantics/cb/`; `logIt: ! {IO}`, `wide: ! {IO, Env}`; every caller declares the
effects it is handed unless noted):

| # | Shape | Result |
|---|---|---|
| c01 | `func applyTo(f: int -> int, x) = f(x)`; `applyTo(logIt, 1)` from `! {IO}` main | accepted |
| c02 | same, called from a `pure func` | rejected, `Missing effects: IO` |
| c03 | **`pure func`** `applyP(f: int -> int, x) = f(x)` with `logIt` from `! {IO}` main | **accepted** (same cross-module) |
| c05 | `std/list.map(logIt, …)` from `! {IO}` main (`map` is `pure func map[a,b](f: a -> b, …)`) | **rejected**, `incompatible closed rows … [IO]` |
| c06 | `mapE(logIt, …)` | accepted |
| c08 | `f: int -> int ! {}` given `logIt` | rejected |
| c20 | `func via(f: int -> int, x) = applyC(f, x)` where `applyC` takes `! {}` | **rejected for an IO caller** |
| c21 | same `via`, given a pure lambda | accepted |
| c23 | same signature as c22, body recursive `f(x) :: myMap(f, rest)` | accepted (c22, body `_list_map(f, xs)`: rejected) |
| c09b | `applyIO(f: int -> int ! {IO}, x) -> int ! {IO}` given `wide` | **accepted**; run with `--caps IO` → `effect 'Env' requires capability` |
| c11b | `{f: wide}` stored in `{f: int -> int ! {IO}}`, called from `! {IO}` | **accepted**; same run-time denial |
| c13b, c17 | unannotated field `{f: int -> int}` holding `logIt`, called via `pure` helpers | rejected at the pure caller (hand-off works) |

**The rule the checker actually implements.** An unannotated arrow in a parameter gets a fresh effect
row variable, **solved by unification over the definition's body**. If the body passes the callback to
a closed position (a `! {}` parameter, or a builtin such as `_list_map`), the row closes, and callers
can no longer pass effectful functions (c20, c22, c05). If not, it stays open and is charged to the
caller (c01, c03). `pure` on the definition does not affect it (c03). The surface signature,
`ailang iface` and the stdlib golden show the same text, `((a)->b, list[a]) -> list[b]` with
`"effects": []`, in both cases (`.stdlib-golden/list.json:956`). Concrete rows are lower bounds only
(c09b, c11b). The run-time capability check is the only backstop, and `--caps auto` grants whatever is
declared.

**Where docs contradict it.** The prompt's "implicit open row" is true only when the body leaves it
open. The `map` case looks like a stdlib exception but is the general rule. Two functions with
identical signatures accept different callers (c22/c23). Under the Stability Promise this means **a
Stable function's implementation change can break callers without any visible signature change**,
and the Stable-golden diff proposed in review §4.2 cannot see it.

**Proposed rule to freeze (D6): the signature is the contract.**
1. An unannotated `T -> U` in a parameter type, including nested in list, tuple, record or ADT
   parameter types, means `T -> U ! {ρ}` with a fresh `ρ` that is **rigid inside the body** (a skolem,
   like a declared type variable) and is charged to the caller at each call. A body that needs a closed
   callback is a check error: "callback `f` is effect-polymorphic in its signature but the body
   requires it pure; write `f: T -> U ! {}`".
2. In type declarations (record fields, constructor payloads) and return types, an unannotated arrow
   keeps today's latent-effect behaviour: charged where it is called or handed off (c13b, c17).
3. A concrete row is an exact upper bound: a value with row R fits a slot with closed row S iff
   R ⊆ S. This is the Phase 3 / #573 / #1326 width hole (c09b, c11b).
4. `iface`, the goldens and `ailang docs` render rows: `! {}` and `! {e}` are visible.

Consequence for the stdlib: `map`, `filter`, `foldl`, `sortBy` and every other function whose body
closes the row get an explicit `! {}` in v0.58.x. Callers are unaffected, because the behaviour is
already closed and is now visible. **The `map`/`mapE` split stays for 1.0.** Making `map`
effect-polymorphic later accepts strictly more programs, so it is additive in 1.x.

**Who breaks.** (1) rejects *definitions*, never call sites: library functions whose body closes an
unannotated callback. Fix by annotating. M6a measures this before any rule changes. (3) rejects
programs that check today: wider functions stored in narrower slots (motoko's `ExtensionHooks` cases E
and G in #573). (1) is a signature-honesty rule, not a soundness fix, so the proposed soundness
clause does not clearly cover it; **it must land in v0.58.x**. (3) is a soundness fix, because the
declared row is false, so it can be covered by the soundness clause from `m-v1-nominal-type-identity`
if Phase 3 does not fit the window. Recommendation: land (1) and (4) in v0.58.x; land (3) there too if
the M6c spike sizes it at 4 days or less, and otherwise ship 1.0 with the LIMITATIONS row plus the
soundness clause.

**Rejected alternatives.** Unannotated means closed `! {}`: matches "omit = pure", but breaks the most
common AI-written shape (c01, `flatMap` with `! {FS}` lambdas, pinned by
`TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback`). Status quo plus docs: honest, but
leaves the signature not being the contract, which is the stability hazard above.

---

## Milestones

Part 1 milestones are independent and separately landable. Each Part 2 milestone starts with its decision.

| # | Milestone | Size (est.) | Gate |
|---|---|---|---|
| M1 | Shared `verification.ExitCode`; `verify.outcome` + `obligations`; obligation count without Z3; `--strict`/`--allow-inconclusive` on `check --verify`; z3 in `Dockerfile.executor-mcp`; rewrite `ai_check_exit_test.go` | 2 d | D1 |
| M2a | CTR004 refusal for `--bytecode`/`--strict-bytecode`/`--batch` + `--verify-contracts` (parity doc M1) | 0.5 d | — |
| M2b | `ContractSpec.Function` set in `attachContracts`; message and trace events carry it; fix `contracts.mdx` example | 0.5 d | — |
| M2c | Panic-mode contracts in named-test bodies; `stability.md` states `run` default-off | 1 d | D3 |
| M3 | Printer prints `(a :: b)`; parse→print→parse property test over `examples/` + `std/`; prompt and strict-mode hint drop `::(x, xs)` | 1.5 d | D4 for the prompt half |
| M4 | `PAR_PARAM_NAME`; `()` → `_: ()`; IFC positional names and "function" wording | 1 d | — |
| M5 | Overflow doc: quorum round 2 → F1–F3 ruling → Phase 0 census (in-repo + downstream) → Phase 1 RT010 → `wrap*`/`checked*`; numeric table in docs | 5 d | D5 |
| M6a | Census build: report every definition whose body closes an unannotated callback row (std, examples, benchmarks, packages, downstream) | 1 d | D6 |
| M6b | Rigid parameter rows + explicit `! {}` on the stdlib + row rendering in `iface`/goldens/docs (`make freeze-stdlib`) | 3 d | M6a |
| M6c | Upper bounds (Phase 3): 1 d spike, then the build if ≤ 4 d, else soundness clause + LIMITATIONS | 1–5 d | D6 |

## Acceptance criteria and tests

- **M1:** a table test over the eight outcomes in §1, driving `ExitCode` directly plus one subprocess
  run per command spelling (`check --verify`, `ai-check`, `verify`), using the `sec/fakez3_*` shapes
  as fixtures. A no-contract file with no solver exits 0. **Mutation:** drop the `unavailable` arm and
  the `/nonexistent` case must fail. Make `verify` use its old private exit logic and the
  cross-command parity test must fail.
- **M2:** `k1.ail` under `--bytecode --verify-contracts` exits with CTR004 before printing. The
  evaluator message contains `in half at`. `kt.ail`'s "half five" fails. **Mutation:** remove the
  `Function:` assignment and the message test must fail.
- **M3:** #1914's `r.ail` passes `ailang test`. The round-trip test covers at least every expression
  node type in `ast` (an unhandled type fails the test rather than being skipped). **Mutation:**
  revert the printer case and the round-trip test must fail on an example containing `::`.
- **M4:** `func f(())` checks and runs. `func f(5)` gives `PAR_PARAM_NAME`. A private function never
  produces an "export" diagnostic. **Mutation:** restore the empty-name path and the IFC test must fail.
- **M5:** as in the overflow doc's Testing Strategy, including its route-independent differential
  test against `math/big`. Plus a verifier golden: `inc` is still `verified`, and `run` now traps.
- **M6:** probes c01–c23 become regression tests with the outcomes rule (1)–(3) predicts: c20 and c22
  fail **at the definition**, c09b and c11b are rejected (if M6c lands), and the rest are unchanged.
  **Mutation:** make the parameter row flexible again and c20's definition-site test must fail.

## Changelog fragment drafts

`changelogs/unreleased/<date>-verify-exit-contract.md`:
```markdown
### Breaking — `check --verify` / `ai-check` / `verify` fail when nothing was proved (M-V1-VERIFICATION-HONESTY M1)

A file with contracts now exits 1 when Z3 is unavailable (`verify.outcome: "unavailable"`) or when an
obligation is inconclusive (`--allow-inconclusive` restores exit 0). Skipped obligations still exit 0
and are listed on stderr; `--strict` fails them. All four spellings share one exit function. New
JSON fields: `verify.outcome`, `verify.obligations`.
```
`…-bytecode-contracts-refused.md`: `### Breaking — run --bytecode --verify-contracts is refused (CTR004) instead of checking nothing (#1956)`
`…-contract-violation-names-function.md`: `### Fixed — contract violations name the function ("ensures failed in half at …")`
`…-test-bodies-check-contracts.md`: `### Breaking — ailang test checks requires/ensures inside named-test bodies`
`…-cons-in-test-bodies.md`: `### Fixed — :: in named-test bodies (#1914); the never-parsing ::(x, xs) expression spelling is removed from the prompt`
`…-param-names.md`: `### Fixed — func f(()) checks again (#2156); a non-name parameter is PAR_PARAM_NAME`
`…-int-overflow-trap.md`: per the overflow doc (`### Breaking` + `### Added`)
`…-callback-rows-rigid.md`: `### Breaking — an unannotated callback parameter's effect row is fixed by its signature, not inferred from the body`

## Docs, prompt and LIMITATIONS updates

- **Teaching prompt** (via `prompt-manager`, syntax verified with `ailang check`): remove `::(x, xs)`;
  replace the "signed hash semantics" wrap wording with the RT010 rule and `wrapMul`; make the callback
  table's `T -> U` row state rule (1); add "contracts are not checked by `run` unless
  `--verify-contracts`; `ailang test` checks them"; state the `ai-check` exit lanes.
- **LIMITATIONS.md:** delete line 25 if M6c lands, otherwise reword it to point at the soundness
  clause; rewrite line 100 (`map` is explicitly `! {}`); add an int-overflow row only if D5 is not
  trap; add "the verifier models `int` as unbounded" until M5 lands.
- **stability.md:** a numeric-semantics reference; `run` contracts default-off as a stated promise;
  the `check --verify` exit table as Stable (the JSON stays Experimental, the `outcome` enum is
  additive).
- `docs/docs/guides/contracts.mdx`: real violation message; exit table. `cloud-setup` skill: delete
  the "passes but verified nothing" entry (and re-sync the `.agents` mirror).

## Conflict Surface

Parser: `parseParams` (function and lambda parameters), strict-mode cons. Printer:
`PrintAILANGSource`. Type checker: row-variable rigidity for parameter arrows, row subsumption at
slots. Runtime: int arithmetic on every route (see the overflow doc's seam table), contract context in
the test harness. CLI: `check --verify`, `ai-check`, `verify`. Programs that must not change:
`tests/binops_int.ail`, `examples/float_nan.ail`, `examples/poly_arith_lambda.ail`, every
`examples/runnable/contracts/*.ail` under `ailang verify`, and probes c01, c06, c13b, c17.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| M1 turns agent done-gates red on inconclusive recursive contracts | Agents grind on a false inconclusive | `--allow-inconclusive`; agent prompt text names it; bank inconclusive separately in eval |
| The hosted executor ships without z3 after M1 | `ailang_check` exits 1 on contracted code | Add z3 to the image in M1. A distroless-static base needs a static z3 or a different base; verify behaviourally |
| M6b census finds many closed-body definitions in packages | Breaking fan-out | Rule (1) is fixable by a one-token annotation; the error names it; downstream gate first |
| Overflow trap breaks unknown wrap-reliant downstream code | Wrong-number code now fails loudly | Phase 0 census including downstream repos; RT010 hint names `wrapMul` |
| Six breaking fragments in one window | Soak resets | Land in one v0.58.x; no breaking fragment during the soak (review §4.3) |

## Non-Goals

Making the VM enforce contracts (parity doc M6–M7, additive later). `std/bigint` (1.x). Merging
`map`/`mapE` (additive in 1.x). BitVec-64 encoding. Raising verification coverage
(`m-contract-verification-coverage`, v1.1). The bounded-unrolling precision bug shown by `r1.ail`:
it is filed as follow-up, and D1 makes it fail closed meanwhile. Contract checking in `run` by default.

## Deferred Decisions

- Exact `verify.outcome` strings and stderr wording — agent.
- Whether non-exported functions keep a persisted IFC summary — agent.
- The `PAR_PARAM_NAME` code name and the RT010 message wording — agent.
- Static z3 build or base-image change for the executor — agent, verified by a behavioural probe.

## Decisions for Mark

1. **D1 exit contract.** Recommended: unavailable (with obligations) and inconclusive fail; skipped
   passes loudly; one non-zero code with the JSON `outcome` as the discriminator.
2. **D3 contracts default.** Recommended: `run` off (frozen and stated), `ailang test` named bodies on.
3. **D5 overflow.** Recommended: trap + explicit `wrap*` in v0.58.x, bigint in 1.x. Run the overflow
   doc's quorum round 2 first; its F1–F3 are this decision.
4. **D6 callbacks.** Recommended: rigid parameter rows + visible rows in v0.58.x; upper bounds in
   v0.58.x if the spike says ≤ 4 days, otherwise under the soundness clause.
5. **D4.** Recommended: retire `::(x, xs)` as an expression rather than implement it.

## Verification Log

| # | Claim | Command | Result |
|---|---|---|---|
| V1 | §1 table | `check --verify`, `verify`, `verify --strict` on `c1 r1 s2 s3`; `AILANG_Z3_PATH=` real, `/nonexistent`, `fakez3_{fail,unknown,hang}` | as tabled |
| V2 | executor image has no z3 | read `docker/Dockerfile.executor-mcp` (distroless/static, 0 z3 mentions) | confirmed statically; live tool not callable (MCP re-auth needed) |
| V3 | §2 transcripts, empty name | `run` ± `--verify-contracts` ± `--bytecode`/`--strict-bytecode` on `k1`, `k2`; `ailang test kt.ail`; `git grep 'SetFunction('` | as shown; only test callers |
| V4 | #1914 cause | `ailang test r.ail`, `r2.ail`; `check r4.ail` (`::(1, [2])`); `check --strict-syntax r5.ail`; `check pc.ail` (pattern form) | expression prefix form fails; pattern form passes; strict hint suggests the failing form |
| V5 | #2156 cause | `check u1 u2 u3 u5` (params `5`, `"x"`, `()`, `_`); same on `ailang.dev-backup-20260928T183545` (v0.47) and a v0.57.1 build | v0.47 accepts `func f(5)` and runs `f("hello")` → `1`; v0.57.1 and HEAD reject all non-name params |
| V6 | §5 table | `run` and `--strict-bytecode` on `ar.ail`, `dz.ail`, `fz.ail` variants | identical on both engines |
| V7 | wrap breaks `verified` | `check --verify inc.ail`; `run`; `run --verify-contracts` | `verified` / `-9223372036854775808` / `ensures failed` |
| V8 | §6 probes | `check` on `cb/c01…c23`; `run --caps IO` on c09b, c11b | as tabled |
| V9 | wrap census | `git grep -l -E '0x[0-9a-fA-F]{12,}'` and hash-name grep over `*.ail` | fnv1a, splitmix64 only rely on wrap |

**Not verified:** the live hosted `ailang_check` output; downstream repos (no census run); REPL and
Go-codegen behaviour for the §5 table (the overflow doc's V3 covers codegen at v0.54.19); the M6a
census count, which needs the census build; `make test` (no code was changed).
