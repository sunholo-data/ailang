---
sidebar_position: 5
title: Known Limitations
description: Live-verified capability & cliff matrix, workarounds, and design constraints in AILANG
reviewed: 2026-11-27
reviewBy: 2027-05-26
---

# AILANG Known Limitations

**Live-verified at AILANG `v0.64.1` (`2026-11-26`); rows tagged `rc.1` track the `v0.64.2 / v1.0.0-rc.1` changes (`2026-11-27`).** Every entry below reflects measured compiler, verifier, or runtime behavior rather than historical speculation. See [Design Documents](/docs/design-docs) for implemented features and [Stability Promise](/docs/reference/stability) for tier guarantees.

## Capability & Cliff Matrix

| Area / Construct | Measured Behavior (`v0.64.1`) | Workaround / Guard |
|---|---|---|
| **`int` wrap vs `verify --int-model=int64`** | Runtime `int` wraps at signed 64-bit (`9223372036854775807 + 1 == -9223372036854775808`). `ailang verify` defaults to unbounded Z3 `Int` (emitting an `int_overflow` caveat note); `--int-model=int64` promotes overflow to `VIOLATION`. | Pass `ailang verify --int-model=int64` or bound inputs in `requires`. |
| **Non-tail recursion (~5 KB/frame) & depth cap** | `ailang run` caps recursion at `10,000` levels/frames by default (`RT_REC_003` on evaluator, `stack overflow` on VM). Non-tail recursion allocates ~5 KB/frame (evaluator hops goroutines every `131,072` levels). | Use tail recursion (constant depth) or `std/list` (`map`, `foldl`, `takeMap`), or pass `--max-recursion-depth N`. |
| **List prepend (`::`, `++`) & strict `take`** | `ListValue` is a flat Go slice, so `x :: xs` and `xs ++ [x]` copy `O(n)` per step (`O(n²)` in loops). Arguments are strict: `take(n, flatMap(f, xs))` materialises the whole `flatMap` first. | Use `map`, `takeMap`, `takeFlatMap`, or `foldl` with a scalar/record accumulator. |
| **Bounded Z3 recursion & verification scope** | `ailang verify` unrolls named recursion (`--verify-recursive-depth N`, default `2`) and literal-lambda HOFs (`map`/`filter`/`foldl`/`any`/`flatMap`, depth `3`). Unbounded recursion is `inconclusive` (no induction). Bitwise ops (`&`, `^`, `~`, `<<`, `>>`) and function parameters are `skipped`; floats model as SMT `Real` (non-NaN input assumed; `/0`, `sqrt(<0)`, `log(<=0)` are violations); opaque helpers note `helpers_not_checked`. | Bound depth/lengths in `requires`, guard floats with `std/math.isNaN`, and run runtime contract checks via `ailang run --verify-contracts` (evaluator only; `--bytecode --verify-contracts` errors `CTR004`). |
| **Explicit-flow IFC (`T<label>`, `Declassify`)** | Information-flow labels track **explicit data flow** only, not implicit control-flow channels (branching on a tainted boolean to return untainted literals). Parameters allow one `not` refinement (`string{not a, not b}` does not parse). | Use explicit `Declassify` at boundary functions; group forbidden labels under a shared parent label (`string{not sensitive}`). |
| **`--bytecode` VM coverage & parity** | `186 / 205` runnable examples run natively on the VM (`~40×` faster than the evaluator on call-heavy code). Remaining gaps (unwired effect builtins, contract-module ADTs, user ADTs/closures passed to evaluator-only helpers, newtype `w.x` access) warn on stderr (`BC001` under `--strict-bytecode`). `Map.insert`/`Set.insert`/`Array.set` copy `O(n)` per update; module-level `let` initializers recompute per read on VM. | Run `ailang check --strict-bytecode` in CI to catch fallbacks; build collections via `fromList`/`updateMany`/`accumulate`; cache expensive module `let` values in locals. |
| **Constructors through a module alias** | `import M as C` binds functions and types (`C.f`, `C.T`), including names a facade re-exports with `export import`, but a qualified constructor (`C.Red`) fails with `IMP010` for every module. | Import constructors by name: `import M (Color, Red)`. |
| **No ad-hoc user typeclasses** | Built-in classes (`Num`, `Eq`, `Ord`, `Show`, `Fractional`) and `deriving (Eq, Json)` work; user `class`/`instance` definitions, explicit constraints (`Num a => ...`), and higher-kinded type params (`m[int]`, `PAR_TYPE_PARAM_APPLIED`) are unsupported. | Pass explicit record-of-functions dictionaries or use ADTs. |
| **Type size & WASM checker budgets** | `check` rejects types >`65,536` nodes / `1,000` depth (`TC_TYPE_SIZE_001`) or excessive alias expansion work (`TC_TYPE_WORK_001`). Browser WASM adds a `2 s` wall-clock type-check budget (`ailangSetTypeCheckBudget` adjusts; CLI unaffected). | Extract helper functions or nominal ADTs for huge record/match chains; flatten nested matches. |
| **Syntax & stdlib constraints** | Multi-statement `if`/`else` branches require `{ ... }`; `let x = expr?` works on `let` bindings along a `Result`/`Option` return spine (`PAR_TRY_LET_ONLY` elsewhere); no Y-combinator/infinite types (`TC004`); no typed quasiquotes or CSP channels; `std/regex` uses RE2 (no backreferences/lookaround); `std/json` integers outside int64 decode as `JNumber(float)`; `std/terminal` native keys are macOS/Linux only. | Wrap multi-statement `else` in `{ ... }`; use `"${expr}"` interpolation; keep >53-bit JSON IDs as strings. |
| **WASM / REPL pipeline parity (`#2410`, `rc.1`)** | `ModuleRegistry.LoadModule` (browser playground, REPL) used to skip effect validation, match exhaustiveness and contract type-checking, so a `pure func` calling `println` or a non-exhaustive `match` loaded there while `ailang check` rejected it. In `v1.0.0-rc.1` it runs the same checks (`TC_MATCH_002`, effect-row validation, contracts, derived `Eq`). | On `v0.64.1` or older playgrounds, re-check snippets with `ailang check` before trusting them. |
| **Z3 `requires` and uncontracted callers (`#2404`)** | `ailang verify` proves each `ensures` assuming that function's own `requires`; it does not prove that a caller *without* contracts satisfies a callee's `requires` (the call is an unchecked boundary, reported as `helpers_not_checked` for opaque helpers). | Put `requires`/`ensures` on callers too, or run `ailang run --verify-contracts` (evaluator only) to check the boundary at runtime. |
| **Nominal record `Eq` (`rc.1`)** | `==` on a record type needs that type's own `deriving (Eq)`; an imported type with the same bare name no longer lends its `Eq` (`SOUND-1`), so a record holding a function field is rejected at check time (`No instance for Eq`) instead of crashing at runtime. | Add `deriving (Eq)` when every field supports `Eq`; compare the relevant fields otherwise. |
| **Pipe `_` placeholder (`rc.1`)** | `x \|> f` still feeds a unary `f` and `5 \|> makeAdder(1)` still calls the curried result; `xs \|> filter(p, _)` puts the left side at one explicit `_` argument. Two or more `_` in one stage fail with `TC_PIPE_001`; `_?` stays a typed hole. | Wrap extra uses in a lambda: `xs \|> (\ys. zipWith(f, ys, ys))`. |
| **Policy confinement residuals (`--policy`)** | Confined execution covers mediated effects (`IO`, `FS`, `Net`, `Clock`, `Rand`, `Stream`). CPU/RAM limits require containers/cgroups; HTTP proxies (`E_NET_PROXY_REFUSED`) and loopback/metadata are refused in restricted mode; `std/ai` HTTP bypasses `Net` pinning (spend capped by `[budgets] AI`); Windows/WASM lack restricted process teardown. | Run untrusted workers in a container/cgroup (`AILANG_EVAL_MAX_RSS`), use `trusted_host` for local Ollama, and set `fs_deny_write` on supply-chain files. |

## Minimal Verified Workarounds

### 1. Int64 Overflow & Z3 Verification (`--int-model=int64`)

```ailang
module docs/limits_int64

export pure func incBounded(x: int) -> int
  requires { x >= 0 && x < 9223372036854775807 }
  ensures { result > x }
{
  x + 1
}
```

Without the upper bound in `requires`, `ailang verify --int-model=int64` flags `x = 9223372036854775807` as `VIOLATION (int_overflow)` instead of an advisory caveat.

### 2. Tail Recursion & Builtin-Backed Loops (`RT_REC_003`)

```ailang
module docs/limits_recursion

import std/list (foldl, takeMap)

export pure func sumAcc(n: int, acc: int) -> int =
  if n <= 0 then acc else sumAcc(n - 1, acc + n)

export pure func topDoubled(xs: [int], k: int) -> [int] =
  takeMap(k, \x. x * 2, xs)

export pure func sumList(xs: [int]) -> int =
  foldl(\acc x. acc + x, 0, xs)
```

`sumAcc` runs in constant stack depth on both engines; `takeMap` avoids allocating unvisited elements under strict evaluation.

### 3. Constructors Through a Module Alias

```ailang parse
-- Fails today with IMP010: constructors are not reachable through a module alias.
module docs/limits_alias_ctor_fail
import std/json as J
export func nul() -> J.Json = J.JNull
```

Workaround: import the constructor by name.

```ailang
module docs/limits_alias_ctor_ok

import std/json (Json, JNull)

export func nul() -> Json = JNull
```

### 4. Named Recursion, Explicit Braces & RE2 Regex

```ailang
module docs/limits_syntax_regex

import std/regex (compile)

export pure func factorial(n: int) -> int =
  if n <= 1 then 1 else n * factorial(n - 1)

export pure func buildDoubles(x: int, maxX: int) -> [int] =
  if x > maxX then [] else {
    let v = x * 2;
    let rest = buildDoubles(x + 1, maxX);
    v :: rest
  }

export pure func checkRegex() -> bool =
  match compile("(a)\\1") {
    Ok(_) => true,
    Err(_) => false   -- RE2 rejects backreferences cleanly with Err
  }
```

### 5. Callback Effect Contracts & Type Bounds

- **Function-field annotations:** Storing an `! {IO, Env}` callback in a record typed `{f: int -> int ! {IO}}` still type-checks (closed upper-bound enforcement remains tracked in `#573`/`#1326`), though invoking annotated fields and `mapE` does charge effects. Use closed pure `T -> U ! {}` for pure callbacks (`std/list.map`).
- **Rigid signature variables & recursive groups:** Explicit `[a]` / implicit lowercase type variables are rigid (`TC_RIGID_001`). Calls inside a recursive group are monomorphic (no polymorphic recursion).
- **Named tests:** `test "..." { ... }` and `property "..." { ... }` bodies are checked pure (`TC_TEST_001`) and do not enforce callee contracts at runtime; verify effectful paths via an exported `main` with `ailang run --caps ... --verify-contracts`.

## Recently Resolved (Now Shipped)

- **Error/option spine propagation (`let x = expr?`):** Unwraps `Ok`/`Some` along a return spine and early-returns `Err`/`None` (see [Option vs Result](/docs/reference/option-vs-result)).
- **Two engines only (`Go` emitter removed):** `compile --emit-go` was removed after `v0.60.5`; AILANG runs exclusively on the reference evaluator and `--bytecode` VM.
- **Exhaustive pattern checks & nominal identity:** `TC_MATCH_002` checks pattern exhaustiveness, `TC_NOMINAL_001` enforces nominal ADT identity across modules, and `TC_CONSTRAINT_001` propagates class constraints across module boundaries.
- **Per-instance numeric compilation (`#2378`):** Generic numeric functions with integer literals (`func inc(x) = x + 1`) compile both `int` and `float` variants (up to 4 literal-carrying type vars; `TC_NUMVAR_001` otherwise).

## Reporting New Limitations

File an issue at [github.com/sunholo-data/ailang/issues](https://github.com/sunholo-data/ailang/issues) with `ailang --version`, a minimal `.ail` reproduction, and expected vs actual behavior.
