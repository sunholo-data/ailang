---
reviewed: 2026-11-27
reviewBy: 2027-05-26
---

# AILANG Known Limitations

> **Canonical page:** [docs/docs/reference/limitations.md](docs/docs/reference/limitations.md) ([ailang.sunholo.com/docs/reference/limitations](https://ailang.sunholo.com/docs/reference/limitations)) carries full workaround snippets and diagnostic codes.

**Live-verified at AILANG `v0.64.1` (`2026-11-26`); `rc.1` rows track `v0.64.2 / v1.0.0-rc.1` (`2026-11-27`).**

## Capability & Cliff Matrix

| Area | Constraint / Cliff (`v0.64.1`) | Workaround |
|---|---|---|
| **Int64 wrap vs Z3** | Runtime `int` wraps at 64 bits; `verify` defaults to unbounded `Int` (with `int_overflow` caveat note). | Run `ailang verify --int-model=int64` or bound inputs in `requires`. |
| **Non-tail recursion** | Default `10,000` level/frame cap (`RT_REC_003` / `stack overflow`); non-tail frames use ~5 KB. | Use tail recursion, `map`/`foldl`, or `--max-recursion-depth N`. |
| **List prepend & strict `take`** | `::` and `++` copy `O(n)` per step; `take(n, flatMap(f, xs))` materialises `flatMap` first. | Use `map`, `takeMap`, `takeFlatMap`, or scalar `foldl`. |
| **Bounded Z3 verification** | Recursion unrolls to depth `2` (`--verify-recursive-depth`), HOF lambdas to depth `3` (no induction). Bitwise ops & fn args skipped; floats model as non-NaN `Real`; `--bytecode --verify-contracts` refused (`CTR004`). | Bound depth/lists in `requires`, check NaN with `std/math.isNaN`, run `--verify-contracts` on evaluator. |
| **Explicit-flow IFC** | Taint labels (`T<label>`) track explicit data flow, not implicit branch channels; single `not` refinement per param. | Use `Declassify` at trust boundaries and shared parent labels for multi-exclusion. |
| **`--bytecode` VM coverage** | `186 / 205` runnable examples run natively; remaining fallbacks warn on stderr (`BC001` under `--strict-bytecode`). `Map.insert`/`Set.insert`/`Array.set` copy `O(n)`. | Gate with `ailang check --strict-bytecode`; bulk-build via `fromList`/`updateMany`/`accumulate`. |
| **Constructors through a module alias** | `import M as C` binds `C.f` and `C.T`, but `C.Red` is `IMP010` for every module. | Import constructors by name: `import M (Color, Red)`. |
| **No ad-hoc user typeclasses** | Built-in `Num`/`Eq`/`Ord`/`Show`/`Fractional` + `deriving (Eq, Json)` only; no user `class`/`instance` or higher-kinded types. | Pass explicit record dictionaries or use ADTs. |
| **Syntax & WASM budgets** | Multi-statement `else` requires `{ ... }`; `expr?` is `let`-spine only (`PAR_TRY_LET_ONLY`); RE2 regex has no backreferences; WASM has a `2 s` type-check wall-clock budget. | Wrap multi-statement branches in `{ ... }`; flatten nested matches; raise `ailangSetTypeCheckBudget` in WASM hosts. |
| **WASM / REPL parity `#2410` (`rc.1`)** | `ModuleRegistry.LoadModule` used to skip effect, exhaustiveness (`TC_MATCH_002`) and contract checks; `v1.0.0-rc.1` matches the CLI. | Re-check older playground snippets with `ailang check`. |
| **Z3 `requires` boundary `#2404`** | `verify` assumes each function's `requires` but does not prove an uncontracted caller meets a callee's. | Add contracts to callers or run `--verify-contracts`. |
| **Nominal record `Eq` (`rc.1`)** | `==` on a record needs its own `deriving (Eq)`; imported same-named types no longer lend one. | Add `deriving (Eq)` when all fields support it. |
| **Pipe `_` placeholder (`rc.1`)** | `xs \|> filter(p, _)` takes one `_` per stage (`TC_PIPE_001` for more); `5 \|> makeAdder(1)` still works. | Use a lambda for multi-use. |

```ailang
module docs/root_limitations_check

import std/list (takeMap)

export pure func incBounded(x: int) -> int
  requires { x >= 0 && x < 9223372036854775807 }
  ensures { result > x }
{
  x + 1
}

export pure func topDoubled(xs: [int], k: int) -> [int] =
  takeMap(k, \x. x * 2, xs)
```

## Policy Confinement Residuals (`--policy`)

Restricted mode mediates `IO`, `FS`, `Net`, `Clock`, `Rand`, and `Stream`. Everything below sits outside that boundary:

| Residual | Impact | Mitigation |
|---|---|---|
| **CPU & memory** | No portable effect-level CPU/RAM bound. | Run inside a container/cgroup (`AILANG_EVAL_MAX_RSS`). |
| **HTTP proxies & loopback** | Restricted mode refuses proxies (`E_NET_PROXY_REFUSED`) and loopback/metadata. | Use `trusted_host` for local model servers or unset proxies. |
| **`std/ai` HTTP client** | Uses Go `http.DefaultClient` rather than pinned `Net` dialer; capped by `[budgets] AI`. | Pin `ai_provider` and set `[budgets] AI`. |
| **Hard links / `/proc` / artifacts** | Root-relative `os.Root` APIs cannot sever pre-seeded links or post-run artifact commits. | Provision clean task roots; enforce `fs_deny_write` and PR review. |
| **Windows / WASM** | No restricted process-group teardown; restricted mode refuses at startup. | Run confined policy workloads on Linux/macOS. |

## Reporting New Limitations

File an issue at [github.com/sunholo-data/ailang/issues](https://github.com/sunholo-data/ailang/issues) with `ailang --version`, a minimal repro, and expected vs actual behavior.
