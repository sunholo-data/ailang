# Parameterised Effects (Phase 1)

> Available since v0.15.0 (M-EFFECT-REFINEMENT Phase 1).

AILANG effect rows can now carry `[key=value]` parameters that ride
alongside the effect name: `!{Rand[mode=os]}`, `!{Rand[mode=seeded]}`,
`!{E[k=v, k2=v2] | tail}`. The syntax, AST, row algebra, and invariant
unification rules ship in v0.15.0 with `Rand` as the pilot effect. The
runtime is unchanged — Phase 1 is the language-feature scaffolding, not
the dispatch refactor. Mode-aware runtime dispatch, capability scoping
(`scope=...`), and the `Clock` / `Net` / `FS` / `AI` ports are tracked
in [the parent M-EFFECT-REFINEMENT design
doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md)
and ship in follow-up sprints.

The point of doing the language-level work first is **back-compat
without forks**. Every existing AILANG program that wrote `!{Rand}`
continues to type-check unchanged: bare `!{Rand}` desugars to
`!{Rand[mode=os]}` via a per-effect default-mode table, and the
unifier treats the bare and explicit forms as equal. 332/332 example
`.ail` files produce byte-identical typecheck output pre-/post-sprint.

## Quick start

```ailang
module examples/modal_rand

import std/rand (rand_int)
import std/io (println)

-- Bare !{Rand} — desugars to !{Rand[mode=os]} via the default-mode table.
export func roll_d6() -> int ! {Rand} = rand_int(1, 6)

-- Explicit !{Rand[mode=os]} — same effect signature as bare !{Rand}.
export func roll_d20() -> int ! {Rand[mode=os]} = rand_int(1, 20)

export func seeded_roll() -> int ! {Rand[mode=seeded]} =
  rand_int(1, 100)

export func crypto_roll() -> int ! {Rand[mode=crypto]} =
  rand_int(1, 100)

export func main() -> () ! {Rand, IO} = {
  let r1 = roll_d6(()) in println(concat_String("roll_d6 (bare Rand): ", show(show(r1))))
  let r2 = roll_d20(()) in println(concat_String("roll_d20 (Rand[mode=os]): ", show(show(r2))))
}

export func main_seeded() -> () ! {Rand[mode=seeded], IO} = {
  let r = seeded_roll(()) in println(concat_String("seeded roll: ", show(show(r))))
}

export func main_crypto() -> () ! {Rand[mode=crypto], IO} = {
  let r = crypto_roll(()) in println(concat_String("crypto roll: ", show(show(r))))
}
```

Run it:

```bash
ailang run --caps Rand,IO --entry main examples/modal_rand.ail
AILANG_SEED=42 ailang run --caps Rand,IO --entry main_seeded examples/modal_rand.ail
ailang run --caps Rand,IO --entry main_crypto examples/modal_rand.ail
```

The full file lives at
[`examples/modal_rand.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/modal_rand.ail).

## Syntax

| Form | Meaning |
|---|---|
| `!{E}` | Bare effect; if `E` has a default-mode entry, desugars to `!{E[mode=default]}` |
| `!{E[k=v]}` | Single parameter |
| `!{E[k=v, k2=v2]}` | Multiple parameters; comma-separated |
| `!{E[k=v] @limit=10}` | Parameters plus an effect budget annotation |
| `!{E[k=v], F}` | Mixed: one parameterised effect, one bare; same row |
| `!{E[k=v] \| row}` | Polymorphic row tail; the rest of the row is a row variable |

Keys are bare identifiers. Values are bare identifiers or string
literals. Whitespace inside `[...]` is permitted. Pretty-printer
output is alphabetical by key for golden-file stability.

### Parser errors

Malformed forms produce structured parser errors at the offending
token's line/column. The error codes (`PAR_EFF010` …
`PAR_EFF014`) are stable for tooling that wants to recognise specific
shapes:

| Input | Error | Code |
|---|---|---|
| `!{Rand[]}` | empty parameter list | `PAR_EFF010` |
| `!{Rand[=os]}` | expected key before `=` | `PAR_EFF011` |
| `!{Rand[mode=]}` | expected value after `=` | `PAR_EFF012` |
| `!{Rand[mode os]}` | expected `=` between key and value | `PAR_EFF013` |
| `!{Rand[mode:os]}` | expected `=` (got `:`) | `PAR_EFF013` |
| `!{Rand[mode=os mode=seeded]}` | missing `,` between params | `PAR_EFF014` |
| `!{Rand[mode=os, mode=seeded]}` | duplicate key | `PAR_EFF014` |

## Default modes (back-compat aliasing)

Bare `!{E}` desugars to `!{E[mode=default_for_E]}` via a per-effect
lookup table compiled into the typechecker. v0.15.0 ships two entries:
`Rand → mode=os` (Phase 1 pilot) and `AI → mode=fixed`
(M-AI-EFFECT-MODES). Effects that do not appear in the table (every
other effect today) keep their bare form unchanged.

| Effect | v0.15.0 default | Future modes (parent doc) |
|---|---|---|
| `Rand` | `mode=os` | `seeded`, `crypto` (Phase 1 syntax; runtime in Phase 3) |
| `AI` | `mode=fixed` | `routeable`, `replay-only`, `byok` (M-AI-EFFECT-MODES; routeable shipped) |
| `Clock` | (none yet) | `wall`, `pinned` (Phase 5) |
| `Net` | (none yet) | `live`, `recorded` (Phase 5) |
| `FS` | (none yet) | `real`, `fixture` (Phase 5) |
| `IO`, `Env`, `Process`, `Debug`, `Declassify` | (none) | not in scope |

`AI` joined the default-mode table in
[M-AI-EFFECT-MODES](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_15_x/m-ai-effect-modes.md)
(v0.15.0). Bare `!{AI}` desugars to `!{AI[mode=fixed]}`; functions
declared `!{AI[mode=routeable]}` opt into runtime provider routing at
the type level and skip the `--allow-routing` CLI gate. See the
[AI Routing guide](./ai-routing.md#type-level-mode-markers-v0150)
for the worked flow.

Adding a new mode means adding a row to
[`internal/types/effects.go`](https://github.com/sunholo-data/ailang/blob/dev/internal/types/effects.go)
under `defaultEffectModes`. There is no user-extensible mode set; see
[Mode set is closed](#mode-set-is-closed) below.

## Unification semantics

Phase 1 unification is **invariant on parameters**. Two effects unify
iff they share the same name AND the same parameter map. There is no
subtyping and no widening coercion. Polymorphic row tails behave as
they did before — only the per-effect parameter map is new.

| Left | Right | Unifies? |
|---|---|---|
| `!{Rand[mode=os]}` | `!{Rand[mode=os]}` | yes — same params |
| `!{Rand[mode=os]}` | `!{Rand}` | yes — bare desugars to `mode=os` |
| `!{Rand[mode=os]}` | `!{Rand[mode=seeded]}` | **no** — invariant; different values |
| `!{Rand[mode=os] \| a}` | `!{Rand[mode=os], FS \| a}` | yes — poly tail picks up `FS` |
| `!{Rand[mode=os], FS}` | `!{FS, Rand[mode=os]}` | yes — row swap is order-insensitive |
| `!{Rand[mode=os]}` | `!{Rand[mode=seeded]} \| a` | **no** — invariant on the named effect |

The `effectiveParamsOf` bridge in
[`internal/types/effects.go`](https://github.com/sunholo-data/ailang/blob/dev/internal/types/effects.go)
normalises rows during comparison: rows constructed by older
back-compat code paths (`stringSliceToEffectRow` in
`validate_effects.go`) have a nil `Params` field, while elaborator-built
rows have desugared `Params`. The bridge consults `DefaultModeFor` to
fill the gap so the two row sources unify cleanly.

### Validation-path subsumption

Function-value effect-row unification remains invariant. Separately, the
closed-row **validation path** allows an explicitly declared Rand mode to
cover a bare or explicit `mode=os` requirement through exactly two registered
edges:

- declared `Rand[mode=seeded]` covers required `Rand[mode=os]`;
- declared `Rand[mode=crypto]` covers required `Rand[mode=os]`.

This relation is asymmetric. An os declaration does not cover a seeded or
crypto requirement, and seeded and crypto do not cover each other. AI, Clock,
Net, and FS have no subsumption edges. Registering a default mode only
normalises a bare spelling (for example, bare `Rand` to `mode=os`); a
registered default grants no subsumption.

## Mode set is closed

The mode set per effect is **closed and enforced**: authors cannot
introduce new modes from user code. The typechecker validates every
parameterised effect against the frozen schema in
[`internal/types/effects.go`](https://github.com/sunholo-data/ailang/blob/dev/internal/types/effects.go)
at effect-row elaboration, rejecting unknown keys and values with a
structured, fix-carrying diagnostic:

| Offending form | Diagnostic |
|----------------|------------|
| `!{Rand[mode=banana]}` (unknown value) | `EFF_UNKNOWN_MODE` — lists the allowed values (`os, seeded, crypto`) |
| `!{Rand[flavor=hot]}` (unknown key) | `EFF_UNKNOWN_PARAM_KEY` — lists the allowed keys (`mode`) |
| `!{Clock[mode=pinned]}` (schema-less effect) | `EFF_PARAMS_NOT_SUPPORTED` — names the tracking doc for the effect's future modes |

Only `Rand` (`mode ∈ {os, seeded, crypto}`) and `AI`
(`mode ∈ {fixed, routeable, replay-only}`, `scope ∈ {byok}`) carry a
parameter schema today; every other effect accepts its bare form only,
and any explicit parameter is a hard error. Adding modes to `Clock`,
`Net`, and `FS` is tracked in
[m-effect-clock-net-fs-modes](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-clock-net-fs-modes.md).

The closed set is deliberate:

1. **Auditable.** A reviewer reading a function's effect row can map
   every parameter value to a known contract by consulting one table
   in `internal/types/effects.go`.
2. **Simpler unification.** With a closed set, parameter-value
   comparison is just string equality. Open sets would force a
   subtype lattice per effect.
3. **Compiler-enforced rollouts.** Adding a new mode is a compiler
   change, gated by review and CI. New modes cannot land silently in
   third-party libraries.

Open / user-extensible mode sets are tracked as a v1.0+ research
question in [the parent design
doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md).

## Trace and replay

**Phase 3 (v1.0.0, M-EFFECT-REPLAY-CONTRACTS) makes the Rand modes
differ at runtime.** The three modes now dispatch to three different
sources:

| Mode | Runtime source | Replay contract |
|------|----------------|-----------------|
| `os` (default) | global OS-entropy PRNG, reseedable via `rand_seed` | re-sampleable |
| `seeded` | dedicated deterministic PRNG pinned by `AILANG_SEED` | deterministic |
| `crypto` | `crypto/rand` CSPRNG draws | opaque |

The resolved mode reaches the `_rand_int` / `_rand_float` /
`_rand_bool` builtins via the effect context (the declared mode is
pushed at moded-function entry), not via distinct builtin names. Trace
events for moded Rand ops now carry the resolved `mode` and its
`contract` label — the replay-contract taxonomy in `internal/replay`
maps each legal `(effect, mode)` pair to `{deterministic,
re-sampleable, opaque}`, so replay tooling can dispatch pin / redraw /
substitute on the contract rather than the raw effect token.

**Seeding contract.** The `seeded` source is seeded ONLY by the
dedicated explicit path (`AILANG_SEED`), never by `rand_seed`.
`rand_seed` keeps its existing contract: it reseeds the **os** source
(so bare-`!{Rand}` programs that call `rand_seed` for reproducible
tests are byte-identical to before). A `seeded`-mode draw with no
`AILANG_SEED` provided is a loud typed error (`RAND_SEEDED_NO_SEED`),
never a silent random fallback. `crypto` entropy failure panics loudly.

## Future work

Phase 1 is one milestone of an eight-phase plan. The follow-up phases
all live in the parent doc:

- **Phase 3 — Replay contract registry.** Per-mode runtime dispatch.
  `!{Rand[mode=seeded]}` actually picks a deterministic handler at
  runtime; `mode=os` keeps the OS-entropy path.
- **Phase 4 — Capability scoping (`scope=...`).** Parameters extend
  beyond mode to scope; e.g. `!{FS[scope=fixture]}` constrains a
  function to a sandboxed filesystem.
- **Phase 5 — Clock / Net / FS / AI ports.** Each adds a row to the
  default-mode table and ports its stdlib. The `AI[mode=routeable]`
  marker shipped in v0.15.0 ([M-AI-EFFECT-MODES](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_15_x/m-ai-effect-modes.md))
  and now subsumes the runtime `--allow-routing` gate from
  M-AI-OPENROUTER (v0.16.0) as a type-level property — see [Modal AI
  in the parent
  doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md#example-4-modal-ai).
  Clock, Net, and FS ports remain pending.
- **Phase 6 — M-ENTROPY integration.** Envelope-level mode validation
  composes with the language-level rules.

Each phase is independently scopable; the parent doc gives the full
sequencing.

## Worked example

[`examples/modal_rand.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/modal_rand.ail)
demonstrates the three Rand-mode forms side by side and runs
end-to-end under `ailang run --caps Rand,IO`. Use it as the starter
template when writing parameterised-effect code.

## See also

- [M-EFFECT-REFINEMENT (parent design, 8 phases)](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-effect-refinement.md)
  — the canonical home for the full taxonomy and the deferred phases.
- [M-EFFECT-REFINEMENT Phase 1 design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_15_x/m-effect-refinement-phase1.md)
  — the design behind v0.15.0's deliverable.
- [M-AI-OPENROUTER (v0.16.0)](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_16_x/m-ai-openrouter-provider.md)
  — runtime `--allow-routing` gate that Phase 5 of the parent doc
  subsumes as `!{AI[mode=routeable]}`.
- [Effects reference](../reference/effects.md) — the catalogue of
  effects today, none of which yet declare parameters except `Rand`.
- [`internal/types/effects.go`](https://github.com/sunholo-data/ailang/blob/dev/internal/types/effects.go)
  — `defaultEffectModes` table, `DefaultModeFor`,
  `effectiveParamsOf`.
