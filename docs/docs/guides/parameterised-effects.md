---
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Parameterised Effects

Effect rows in AILANG can carry `[key=value]` parameters alongside an effect name — for example `!{Rand[mode=os]}`, `!{Rand[mode=seeded]}`, `!{AI[mode=routeable]}`, or `!{Net[scope=public]}`. Bare `!{Rand}` and `!{AI}` desugar to their default modes (`Rand[mode=os]`, `AI[mode=fixed]`) so existing programs type-check unchanged.

## Quick Start

```ailang
module examples/modal_rand

import std/rand (randInt)
import std/io (println)

export func roll_d6() -> int ! {Rand} = randInt(1, 6)
export func roll_d20() -> int ! {Rand[mode=os]} = randInt(1, 20)
export func seeded_roll() -> int ! {Rand[mode=seeded]} = randInt(1, 100)
export func crypto_roll() -> int ! {Rand[mode=crypto]} = randInt(1, 100)

export func main() -> () ! {Rand, IO} = {
  let r1 = roll_d6(()) in println("roll_d6 (bare Rand): ${show(r1)}")
  let r2 = roll_d20(()) in println("roll_d20 (Rand[mode=os]): ${show(r2)}")
}

export func main_seeded() -> () ! {Rand[mode=seeded], IO} = {
  let r = seeded_roll(()) in println("seeded roll: ${show(r)}")
}

export func main_crypto() -> () ! {Rand[mode=crypto], IO} = {
  let r = crypto_roll(()) in println("crypto roll: ${show(r)}")
}
```

```bash
ailang run --caps Rand,IO --entry main examples/modal_rand.ail
AILANG_SEED=42 ailang run --caps Rand,IO --entry main_seeded examples/modal_rand.ail
ailang run --caps Rand,IO --entry main_crypto examples/modal_rand.ail
```

## Syntax & Parser Errors

| Form | Meaning |
|---|---|
| `!{E}` | Bare effect (desugars to default mode if registered) |
| `!{E[k=v]}` / `!{E[k=v, k2=v2]}` | Single or multiple `key=value` parameters (identifiers or string literals) |
| `!{E[k=v] @limit=10}` | Parameters combined with an effect budget annotation |
| `!{E[k=v] \| row}` | Polymorphic row tail |

Malformed syntax fails with `PAR_EFF010` (empty `[]`), `PAR_EFF011` (missing key), `PAR_EFF012` (missing value), `PAR_EFF013` (expected `=`), or `PAR_EFF014` (missing `,` / duplicate key).

## Closed Effect Mode Schema

The parameter schema per effect is **closed and compiler-enforced** in `internal/types/effects.go`:

| Effect | Default | Allowed Parameters |
|---|---|---|
| `Rand` | `mode=os` | `mode ∈ {os, seeded, crypto}` |
| `AI` | `mode=fixed` | `mode ∈ {fixed, routeable, replay-only}`, `scope ∈ {byok}` |
| `Net` | `(none)` | `scope ∈ {public}` |
| `Clock`, `FS`, `IO`, `Env`, `Process`, `Debug`, `Declassify` | `(none)` | Bare only (`EFF_PARAMS_NOT_SUPPORTED`) |

Unknown keys or values fail at elaboration with `EFF_UNKNOWN_PARAM_KEY` or `EFF_UNKNOWN_MODE` listing allowed choices.

## Unification & Validation Semantics

Function-value effect-row unification is **invariant on parameters** (two rows unify iff they share the same effect name and parameter map; bare `!{Rand}` equals `!{Rand[mode=os]}`).

| Left | Right | Unifies? |
|---|---|---|
| `!{Rand[mode=os]}` | `!{Rand}` | Yes (default mode `os`) |
| `!{Rand[mode=os]}` | `!{Rand[mode=seeded]}` | **No** (invariant) |
| `!{Rand[mode=os] \| a}` | `!{Rand[mode=os], FS \| a}` | Yes |

Separately, closed-row **call validation** allows `Rand[mode=seeded]` and `Rand[mode=crypto]` declarations to satisfy bare/`mode=os` calls (asymmetric — `os` never covers `seeded`/`crypto`). `Net[scope=public]` is a narrowing parameter compatible in both directions with `!{Net}`.

## `Rand` Runtime Modes & Replay Contracts

| Mode | Runtime Source | Replay Contract | Seeding / Failure |
|---|---|---|---|
| `os` (default) | Global OS-entropy PRNG | `re-sampleable` | Reseedable via `randSeed` |
| `seeded` | Dedicated deterministic PRNG | `deterministic` | Requires `AILANG_SEED` (fails loudly with `RAND_SEEDED_NO_SEED` if unset; ignores `randSeed`) |
| `crypto` | `crypto/rand` CSPRNG | `opaque` | Panics loudly on entropy failure |

## `Net[scope=public]` {#net-scope-public}

`!{Net[scope=public]}` blocks loopback (`127.0.0.0/8`, `::1`, `localhost`) and cloud metadata (`169.254.169.254`) even when `--net-allow-localhost` or `--net-allow-metadata` is enabled, pinning the dial after DNS resolution (`E_NET_IP_BLOCKED` / `E_NET_DNS_REBINDING`). HTTP redirects never target loopback/private IPs for any `Net` or `Stream` call, and `--bytecode` dispatches `Net[scope=public]` on the evaluator so the guard holds on both engines.

```ailang
import std/net (httpGet)
export func fetchClient(u: string) -> string ! {Net[scope=public]} = httpGet(u)
```

Runnable examples: `examples/modal_rand.ail` and `examples/runnable/net_scope_public.ail`.

## See Also

- [Effects Reference](../reference/effects.md) — full capability & effect catalog
- [AI Routing Guide](./ai-routing.md) — `!{AI[mode=routeable]}` workflow
- [M-EFFECT-REFINEMENT Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_1_0/m-effect-refinement.md)
