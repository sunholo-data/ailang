## M-PURE-PRNG: `std/prng` — a pure, capability-free, reproducible random generator

**Status**: PLANNED
**Target**: v0.28.0 (tentative)
**Priority**: P3 (Medium-low — unblocks nothing outright, but removes a recurring friction *and* makes cross-language benchmark comparison meaningful, which is high-leverage for the harness)
**Estimated**: ~0.5–1 day (pure stdlib + known-answer tests; **zero runtime changes** in the default plan)
**Dependencies**: None. Existing bitwise builtins (`shiftLeft`/`shiftRight`/`bitwiseXor`/`bitwiseAnd`) are sufficient. Optional stretch (§6) adds one builtin.

**Reported from**: Snake Showdown build team (external 0→1 user), AILANG v0.26.1 — `ailang-felix-gap-analysis-1.md` Gap 6. Maintainer-proposed extension beyond what the report asked for (the report only requested docs).
**Verified against**: v0.27.0 (`int` is 64-bit signed; bitwise ops present; `>>` is arithmetic — see §3).

---

## Verdict: VALID extension — fills a real hole between "no randomness" and "the `Rand` capability"

Today AILANG has exactly one source of randomness: `std/rand`, which is **effectful** (`! {Rand}`)
and backed by **hidden global seed state** ([std/rand.ail:13](../../../std/rand.ail#L13)). That is the
right design for "I just want a random number and I don't care about reproducibility." But it leaves
two real gaps the Snake benchmark hit:

1. **Capability-minimal programs can't use it.** Felix's run command was pinned to `--caps IO,Clock`.
   Adding `Rand` changes the program's audited capability surface — the whole point of the effect
   system is that `--caps` is a truthful manifest, so widening it just to get a deterministic food
   sequence is a real cost. He hand-rolled an LCG in pure AILANG to stay within the declared caps.

2. **`std/rand` + a seed is still not a portable, value-level generator.** `rand_seed` makes the
   *global* sequence reproducible within one process, but you cannot fork it, thread two independent
   streams, or get the same bytes on another machine/runtime as a pure value. Felix's hand-rolled LCG
   *was* reproducible — but being hand-rolled and low-quality, its food sequence diverged from Python's
   Mersenne Twister, which is **why the 656-vs-337 tick comparison in the benchmark is meaningless**
   (different food → different trajectories). A documented, shared-algorithm PRNG on both sides makes
   that comparison real.

`std/prng` is the missing piece: a **pure, `pure func`, capability-free generator whose state is an
explicit value you thread through your program**. Same seed + same call sequence → identical output,
deterministically, on any machine. It is the functional-RNG pattern (Haskell `System.Random`'s pure
interface, JAX `PRNGKey`) and it fits AILANG's "no hidden state, machine-verified purity" thesis
exactly. It does **not** replace `std/rand` — it sits beside it for the auditable/reproducible case.

---

## Design

### API surface (`std/prng`)

A generator is a newtype over a 64-bit state word, so you can't accidentally thread a plain `int`:

```ailang
module std/prng

-- Opaque generator state. Construct with seed(); never inspect the inner int.
export type Gen = Gen(int)

-- Create a generator from any seed. All seeds are valid.
export pure func seed(n: int) -> Gen

-- Core step: a uniform 64-bit-ish value and the next generator.
-- Every primitive returns (value, nextGen) — you thread the Gen forward.
export pure func next_int(g: Gen) -> (int, Gen)

-- Uniform integer in [lo, hi] inclusive (unbiased via rejection).
export pure func int_range(g: Gen, lo: int, hi: int) -> (int, Gen)

-- Uniform float in [0.0, 1.0).
export pure func next_float(g: Gen) -> (float, Gen)

-- Uniform bool.
export pure func next_bool(g: Gen) -> (bool, Gen)
```

Ergonomic layer (the verbosity of threading is the only downside of purity — these pay it down once):

```ailang
-- n values in [lo,hi]; returns the list and the advanced generator.
export pure func ints(g: Gen, n: int, lo: int, hi: int) -> ([int], Gen)

-- Pick one element (None on empty list).
export pure func pick[a](g: Gen, xs: [a]) -> (Option[a], Gen)

-- Fisher–Yates shuffle.
export pure func shuffle[a](g: Gen, xs: [a]) -> ([a], Gen)

-- Split into two independent generators (for sub-computations).
export pure func split(g: Gen) -> (Gen, Gen)
```

### Usage (Snake food, the original motivation)

```ailang
import std/prng (seed, int_range, Gen)

-- Before (Felix's hand-rolled, undocumented, low-quality LCG):
--   pure func lcg_next(s: int) -> int = (s * 1664525 + 1013904223) % 4294967296
-- After: a documented, tested generator, still capability-free.
let (fx, g1) = int_range(seed(42), 0, 19)   -- food x
let (fy, g2) = int_range(g1, 0, 19)         -- food y, threaded gen
```

`--caps IO,Clock` is untouched — `std/prng` needs **no capability** because it has no effect.

---

## §3. The one real gotcha: `>>` is an *arithmetic* shift

This is the fact that decides the implementation, so it's called out loudly. AILANG `int` is Go `int`
= **64-bit signed**, and `shiftRight` lowers to Go's `>>`, which is **arithmetic (sign-extending)** for
signed values ([internal/builtins/math_bitwise.go:45](../../../internal/builtins/math_bitwise.go#L45)).
Two's-complement `+` and `*` wrap mod 2⁶⁴ exactly as a textbook 64-bit PRNG wants — but every modern
mixer (SplitMix64, PCG, xoshiro) does `z ^ (z >>> k)` with a **logical** right shift. On a value with
the high bit set, AILANG's `>>` sign-extends and corrupts the mix.

Two ways to handle it; the default plan needs no runtime change:

- **Default (pure): mask the state to 63 bits.** Keep `Gen`'s state in `[0, 2⁶³)` by masking with
  `bitwiseAnd(x, 0x7FFFFFFFFFFFFFFF)` after each step. The high bit is then always 0, so arithmetic and
  logical right-shift coincide, and every emitted `int` is conveniently non-negative. Cost: one bit of
  state width — irrelevant for game/sampling randomness. A logical-shift-right helper is then just:
  ```ailang
  pure func lsr(x: int, k: int) -> int = bitwiseAnd(shiftRight(x, k), shiftLeft(1, 63 - k) - 1)
  ```
- **Stretch (§6): add a `>>>` / `ushr` builtin** for textbook full-width 64-bit algorithms without
  masking gymnastics. Nice-to-have, not required.

### Algorithm: SplitMix64 (default)

Single 64-bit state, trivial to implement, high statistical quality, and *designed* for exactly this
value-threading/`split` use. PCG-XSH-RR and xoshiro256** are documented alternatives if quality needs
grow; SplitMix64 is more than adequate for games, sampling, shuffles, and benchmark food sequences.
The exact constants (`0x9E3779B97F4A7C15`, `0xBF58476D1CE4E5B9`, `0x94D049BB133111EB`) ship in the doc
comment so a peer language can reproduce the sequence bit-for-bit (see §5).

---

## §4. Implementation plan (default: pure stdlib, zero runtime changes)

1. `std/prng.ail`: `Gen` newtype, SplitMix64 step with 63-bit masking, the core 4 primitives, the
   ergonomic layer, `split`.
2. `int_range` uses **rejection sampling** (not modulo) so the distribution is unbiased.
3. `examples/runnable/prng_demo.ail` — seeded, capability-free, deterministic output; gated by
   `make verify-examples`.
4. Docs: `std/prng` page; cross-link from `std/rand` ("need reproducible, cap-free randomness? see
   `std/prng`") and from the effects/capability table.
5. `ailang prompt`: one line teaching `std/prng` for the auditable/reproducible case vs `std/rand` for
   the convenience case.

## §5. Testing

- **Known-answer tests**: golden first-N outputs for `seed(0)`, `seed(42)`, `seed(-1)` — locks the
  sequence so it can never silently change (and so Python/Go can match it).
- **Reproducibility**: `seed(k)` twice → identical streams; threading vs `ints(...)` agree.
- **Distribution sanity**: `int_range(_, 0, 9)` over many draws is roughly uniform; `next_float` in
  `[0,1)`; `int_range` never returns out of bounds; `lo == hi` returns `lo`.
- **Sign safety**: every emitted state/int is non-negative (the 63-bit-mask invariant) — the direct
  regression test for the §3 gotcha.

## §6. Stretch (optional, separate decision)

Add an unsigned/logical right-shift builtin (`ushr` or `>>>`). Tiny runtime change; lets `std/prng`
(and any future hashing/crypto code) use full-width 64-bit mixers without the masking helper. Defer
unless a second use case appears — masking is sufficient for v1.

---

## Why this is worth doing (harness lens)

Per PROGRAM.md §4 this routes as an **AILANG fix / stdlib extension** (missing primitive), not a core
change. Beyond removing Friction #6's recurring "hand-roll an LCG" tax, it has a specific payoff for the
benchmark loop: **cross-language runtime comparisons become valid.** If the bench harness pins both
AILANG and Python to the same documented SplitMix64 seed, the food sequences match and a tick-count
delta finally measures the *language*, not the PRNG. That turns the single most-caveated number in the
Snake report ("656 vs 337 — not comparable") into a real signal for the next run.

## Out of scope

- Cryptographic randomness (use `std/crypto` / OS entropy via a capability — pure ≠ secure).
- Replacing or deprecating `std/rand` — the effectful global generator stays for the convenience case.
- A full distributions library (normal, poisson, …) — `std/prng` is the uniform substrate they'd build on.
