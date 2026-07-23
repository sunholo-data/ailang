## M-PURE-PRNG: `std/prng` — a pure, capability-free, reproducible random generator

**Status**: PARKED — needs-human-review (iter 93, 2026-07-23). Revised once + re-quorumed once per the
QUORUM-AT-PICK gate. **Core generator PROVEN sound** — round-2 accepted the stream step, `ushr`,
constants, and `int_range`; the controller independently re-ran §3.3 at HEAD and it matches canonical
Python SplitMix64 **bit-for-bit** (seed 0 & 42). **Sole remaining round-2 block: `split`.** Both
reviewers converge: `split` is labeled "standard SplitMix64 split" but a single-word `Gen(int)`
structurally cannot provide it — the reference SplittableRandom (Steele/Lea/Flood, Java
`SplittableRandom`) needs a **2-word state (`state` + per-instance odd `gamma`)** to guarantee
independent cycles; §3.4 also defers the exact construction and references an undefined `GOLDEN_ALT`.
This is a **v1 scope/value decision for Mark** (see the two resolution paths below), not a mechanical
fix — hence parked rather than force-passed (Standing rule 2; the gate is one-revision-one-requorum).

> **PARKED DECISION (for @MarkEdmondson1234):** does v1 want *independently-forkable* PRNG streams?
> - **Path X — defer `split` (lowest risk, ships the proven core now):** cut `split` from v1 scope;
>   the four primitives + `int_range`/`ints`/`pick`/`shuffle` cover the Snake motivation and every
>   stated use case. Add `split` later only if a real fork/independent-stream demand appears
>   (demand-evidence gate). The core generator is already interop-grade and verified.
> - **Path Y — proper splittable RNG:** widen `Gen(int)` → a 2-word `Gen(state, gamma)` carrying a
>   per-instance odd `gamma`, implementing the real SplittableRandom split with its own known-answer
>   vectors. Correct forking, but a larger change + a distinct verification surface.
>
> Everything ELSE in this doc is round-2-clean and live-verified. Unpark = pick a path; Path X routes
> straight to sprint-planner with a one-line scope cut, Path Y needs a short design delta first.

Prior header (round-1 revision, retained for provenance): *REVISED after 2-reviewer quorum block; all
round-1 blocking objections resolved and live-verified at HEAD (see Verification Log).*
**Target**: v0.31.0 (was v0.28.0 — stale)
**Priority**: P3 (Medium-low — unblocks nothing outright, but removes a recurring friction *and* makes cross-language benchmark comparison meaningful, which is high-leverage for the harness)
**Estimated**: ~1 day (pure stdlib + known-answer tests; **zero runtime changes** — this claim was
previously asserted, it is now *proven*: a full-width, bit-exact SplitMix64 runs today at HEAD using
only existing builtins — see Verification Log, probe 3)
**Dependencies**: None. Existing bitwise operators (`&`/`^`/`<<`/`>>`/`~`, lowering to
`internal/builtins/math_bitwise.go`) plus `std/math.intToFloat` are sufficient. Optional stretch (§6)
adds one builtin for ergonomics/perf only — it is **not required for correctness**.

**Reported from**: Snake Showdown build team (external 0→1 user), AILANG v0.26.1 — `ailang-felix-gap-analysis-1.md` Gap 6. Maintainer-proposed extension beyond what the report asked for (the report only requested docs).
**Verified against**: v0.30.0-126-g82084c1a9 (`int` is 64-bit signed; `+`/`*` wrap mod 2⁶⁴; `>>` is
arithmetic; full-width hex literals are rejected by the lexer — every one of these is live-verified
below, not assumed).

---

## Revision summary — what the quorum blocked and how each block is resolved

| # | Objection | Resolution |
|---|-----------|------------|
| 1 | 63-bit state masking is **unsound**: the mixer's multiplications overflow into the sign bit *before* the shift, so `z >> k` still sign-extends even on masked state | **Accepted; plan retracted.** The old "mask the state to 63 bits" default is gone. Replaced by a correct pure logical-shift helper (§3) — verified bit-for-bit against a Python reference. |
| 2 | Proposed `lsr` helper mask `shiftLeft(1, 63-k) - 1` is off-by-one (needs `64-k`), and even that is fragile at `k=0` | **Accepted; helper replaced.** The new `ushr` uses a two-step construction with **no width-dependent mask at all** (§3). (Side note, live-verified: `1 << 64` evaluates to `0` at HEAD, so the *corrected* `(1<<(64-k))-1` form would coincidentally also work at `k=0` — but we don't rely on that fragility.) |
| 3 | The SplitMix64 constants exceed signed-int64 positive range and **do not parse** as hex literals | **Confirmed live** (probe 1). Constants ship as **signed two's-complement decimal literals** with the hex in a comment (§3.2). No lexer change — consistent with default-bias-not-core. A full-width-hex lexer extension is noted as an independent nice-to-have, *not* a dependency. |
| 4 | Value fork: a 63-bit-masked variant is a non-standard algorithm, defeating the headline cross-language-comparability payoff | **Path A chosen and proven**: true full-width, standard SplitMix64, implemented purely, output verified **bit-identical** to a Python reference for seeds 0 and 42 (probe 3 vs. probe 0). The interop payoff stands, un-downgraded. No renamed/AILANG-specific variant is needed. |
| 5 | `int_range` API safety: unbounded rejection loop; `hi-lo+1` overflow; `lo>hi` undefined; `lo==hi` | **Fully specified** in §4: bounded rejection (cap 128 draws, exhaustion is a defined `Err`, P(exhaustion) ≤ 2⁻¹²⁸), span-overflow defined (`RangeTooWide`), `lo>hi` is a defined `Err` (no silent fallback), `lo==hi` returns `lo`. The API is total: it returns `Result`. |

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

import std/result (Result)

-- Opaque generator state. Construct with seed(); never inspect the inner int.
export type Gen = Gen(int)

-- Defined errors for range requests. No silent fallbacks (NO-SILENT-FALLBACKS rule).
export type PrngError
  = InvalidRange(int, int)   -- lo > hi
  | RangeTooWide(int, int)   -- hi - lo + 1 exceeds 2^63 - 1 (use next_int for full width)
  | Exhausted                -- rejection cap hit; P <= 2^-128, defined rather than looping forever

-- Create a generator from any seed. All int64 seeds are valid.
export pure func seed(n: int) -> Gen

-- Core step: the next full-width 64-bit word (as AILANG's signed int — the
-- bit pattern is the standard SplitMix64 output; negative values are simply
-- outputs with the top bit set) and the advanced generator.
export pure func next_int(g: Gen) -> (int, Gen)

-- Uniform integer in [lo, hi] inclusive (unbiased, bounded rejection — see §4).
-- Total function: invalid input is a defined Err, never a crash or a fallback.
export pure func int_range(g: Gen, lo: int, hi: int) -> Result[(int, Gen), PrngError]

-- Uniform float in [0.0, 1.0) — top 53 bits of the next word / 2^53 (see §3.4).
export pure func next_float(g: Gen) -> (float, Gen)

-- Uniform bool (top bit of the next word).
export pure func next_bool(g: Gen) -> (bool, Gen)
```

Ergonomic layer (the verbosity of threading is the only downside of purity — these pay it down once).
The list-level helpers take *validated* bounds by construction (`pick`/`shuffle` derive their ranges
from list length, which cannot produce `lo > hi` or overflow), so they stay `Result`-free:

```ailang
-- n values in [lo,hi]; same defined errors as int_range.
export pure func ints(g: Gen, n: int, lo: int, hi: int) -> Result[([int], Gen), PrngError]

-- Pick one element (None on empty list).
export pure func pick[a](g: Gen, xs: [a]) -> (Option[a], Gen)

-- Fisher–Yates shuffle.
export pure func shuffle[a](g: Gen, xs: [a]) -> ([a], Gen)

-- Split into two independent generators (standard SplitMix64 split:
-- left = advance, right = seed with a fresh mixed word).
export pure func split(g: Gen) -> (Gen, Gen)
```

### Usage (Snake food, the original motivation)

```ailang
import std/prng (seed, int_range, Gen)

-- Before (Felix's hand-rolled, undocumented, low-quality LCG):
--   pure func lcg_next(s: int) -> int = (s * 1664525 + 1013904223) % 4294967296
-- After: a documented, tested, standard generator, still capability-free.
match int_range(seed(42), 0, 19) {
  Ok((fx, g1)) => ...,          -- food x, threaded gen; bounds are literals, always Ok
  Err(_) => ...                 -- unreachable for literal 0..19, but defined
}
```

`--caps IO,Clock` is untouched — `std/prng` needs **no capability** because it has no effect.

---

## §3. Implementation: full-width standard SplitMix64, purely (Path A)

### §3.1 The gotcha, stated correctly this time

AILANG `int` is Go `int` = **64-bit signed**. Live-verified facts (Verification Log, probe 2):

- `+` and `*` **wrap mod 2⁶⁴** in two's complement — exactly what SplitMix64 needs.
- `>>` is **arithmetic** (sign-extending): `-1 >> 1` → `-1`.
- `1 << 63` → minimum int (the sign bit); `1 << 64` → `0` (Go shift semantics).

Every modern mixer does `z ^ (z >>> k)` with a **logical** right shift. The previous revision of this
doc proposed keeping the *state* masked to 63 bits so arithmetic and logical shift coincide. **That was
unsound** (quorum objection 1): the mixer multiplies *before* it shifts —
`z = (z ^ (z >>> 30)) * MIX1` — and the multiplication's wrapped product freely lands in the sign bit
regardless of how the incoming state was masked. So the value being shifted can be negative even with a
63-bit state invariant, and `>>` corrupts the mix. Masking between steps does not fix the inside of a
step. Retracted.

### §3.2 The constants: signed-decimal literals (lexer rejects full-width hex)

Live-verified (probe 1): `let a = 0x9E3779B97F4A7C15` fails to parse — the lexer uses signed
`ParseInt`, so hex above `2^63-1` is rejected. Also `-9223372036854775808` as a literal fails (unary
minus is a separate token; the magnitude overflows) — min-int must be written `1 << 63` where needed.
Two's-complement decimal literals of the same bit patterns parse fine. The module therefore ships:

```ailang
-- SplitMix64 constants (Steele, Lea & Flood 2014). AILANG's lexer rejects hex
-- literals above 2^63-1, so these are the signed two's-complement decimals of
-- the standard unsigned constants — the BIT PATTERNS are identical:
--   GOLDEN = 0x9E3779B97F4A7C15 = -7046029254386353131
--   MIX1   = 0xBF58476D1CE4E5B9 = -4658895280553007687
--   MIX2   = 0x94D049BB133111EB = -7723592293110705685
```

(All three decimals machine-computed and live-verified — see Verification Log probes 0 and 3. Do not
hand-derive these.)

*Considered and declined*: extending the lexer to accept full-width unsigned hex. It would be a small
runtime change, and by PROGRAM.md's default-bias it must clear a higher bar than "would read nicer in
one stdlib module." Signed decimals + hex comments are fully adequate; a full-width-hex literal
extension can be proposed independently if a second consumer (hashing, crypto, bit-twiddling examples)
shows demand. **This design has no lexer dependency.**

### §3.3 The logical-shift helper and the exact mixer

The corrected pure logical shift, valid for `k ∈ [1, 63]` (SplitMix64 only ever uses k = 30, 27, 31):

```ailang
-- Logical (unsigned) right shift for k in [1,63], from arithmetic >>.
-- Step 1: (x >> 1) & 0x7FFF_FFFF_FFFF_FFFF clears the sign bit copied in by the
--         arithmetic shift -> a true logical shift by exactly 1.
-- Step 2: the intermediate is now non-negative, so arithmetic >> equals logical
--         >> for the remaining k-1 positions.
-- (9223372036854775807 = 0x7FFF_FFFF_FFFF_FFFF = maxInt)
pure func ushr(x: int, k: int) -> int =
  ((x >> 1) & 9223372036854775807) >> (k - 1)
```

This replaces the quorum-rejected `lsr` (whose `63-k` mask was off-by-one, objection 2). It has **no
width-dependent mask**, so there is no `k=0` edge to get wrong; `k=0` is simply outside its contract
(`ushr(x, 0)` would compute `x >> 1 ... >> -1` → the runtime's defined `RT_SHIFT` negative-shift
error, i.e. loud, not silent). It is an internal helper in v1 — not exported — so the `[1,63]`
precondition is enforced by the module's own call sites (all constant). Edge cases live-verified:
`ushr(-1,1)`, `ushr(-1,63)`, `ushr(minInt,63)`, `ushr(minInt,1)` all correct (probe 3), including the
exact objection-1 scenario (a negative post-multiplication value being logically shifted).

The exact step, reproducible bit-for-bit in any peer language (all arithmetic mod 2⁶⁴; `>>>` denotes
logical shift):

```
state' = state + 0x9E3779B97F4A7C15                 // wraps mod 2^64
z0 = state'
z1 = (z0 xor (z0 >>> 30)) * 0xBF58476D1CE4E5B9      // wraps mod 2^64
z2 = (z1 xor (z1 >>> 27)) * 0x94D049BB133111EB      // wraps mod 2^64
out = z2 xor (z2 >>> 31)
```

and its AILANG transliteration (this exact code ran at HEAD and matched the Python reference on all
six checked outputs — probe 3):

```ailang
pure func mix(s: int) -> int = {
  let z1 = (s ^ ushr(s, 30)) * -4658895280553007687;
  let z2 = (z1 ^ ushr(z1, 27)) * -7723592293110705685;
  z2 ^ ushr(z2, 31)
}

pure func next(s: int) -> (int, int) = {
  let s2 = s + -7046029254386353131;
  (mix(s2), s2)
}
```

**Consequence for the value fork (objection 4): this IS standard SplitMix64.** Output words are the
standard unsigned values reinterpreted as signed int64 (AILANG prints `-2152535657050944081` where
Python prints `16294208416658607535`; same 64 bits: `0xE220A8397B1DCDAF`). Cross-language
comparability — the doc's headline payoff — is retained at full strength, with zero runtime changes.
Path B (renamed AILANG-specific variant) is moot and rejected.

### §3.4 Derived generators (exact constructions)

- `next_bool`: sign bit of the next word — `next_int` value `< 0` ⇔ top bit set. (Top bits of the
  SplitMix64 finalizer are its best bits.)
- `next_float`: `intToFloat(ushr(v, 11)) / intToFloat(1 << 53)` — top 53 bits, uniform in `[0, 1)`.
  Live-verified identical to Python's `(v >> 11) / 2**53` (probe 5: `0.8833108082136426` both sides).
- `split`: standard SplitMix64 split — `(Gen(state'), seed_from(mix(state' + GOLDEN_ALT)))` per the
  reference algorithm; exact formulation finalized in implementation with its own known-answer vectors.

---

## §4. `int_range` specification (objection 5, all four points)

`int_range(g, lo, hi) -> Result[(int, Gen), PrngError]`, defined for **all** inputs:

- **(c) `lo > hi`** → `Err(InvalidRange(lo, hi))`. Defined error, no clamping, no swapping, no silent
  fallback (NO-SILENT-FALLBACKS). Pure AILANG code has no trap channel (verified: no panic/error
  builtin exists; div-by-zero is an undesigned Go panic), so `Result` is the honest total API.
- **(d) `lo == hi`** → `Ok((lo, g'))` where `g'` is the advanced generator (one state step is still
  consumed, so sequences stay aligned whether or not a range is degenerate).
- **(b) span overflow**: `span = hi - lo + 1` computed in (wrapping) int64 arithmetic. For `lo ≤ hi`
  the true span is in `[1, 2^64]`; it is representable as a positive int64 iff true span `≤ 2^63 - 1`.
  If the wrapped `span ≤ 0` (true span `≥ 2^63`) → `Err(RangeTooWide(lo, hi))`. Callers wanting the
  full 64-bit width use `next_int` directly. (v1 keeps the unsigned-comparison gymnastics out of the
  hot path; widening support can come later if anyone hits it.)
- **(a) bounded rejection**: unbiased sampling via bitmask rejection with a **hard cap**:
  1. `mask` = smallest `2^m − 1 ≥ span − 1` (computed by constant-time mask folding: `n |= n>>1; n |= n>>2; … n |= n>>32` — no loop).
  2. Draw `v, g' = next_int(g)`; candidate `u = v & mask` (uniform on `[0, 2^m)`; masking with
     `mask ≤ 2^63−1` also makes `u` non-negative, so no logical-shift concerns here).
  3. Accept `Ok((lo + u, g'))` if `u < span`, else redraw — **at most 128 draws total**.
  4. Draw 129 is never made: after 128 rejections → `Err(Exhausted)`.

  Because `2^m < 2·span`, each draw accepts with probability `> 1/2`, so
  `P(Exhausted) ≤ 2^-128` — cryptographically never, but per the mission's standing "every wait is
  bounded" rule the loop bound and its exhaustion behavior are *defined* rather than relying on
  "it terminates almost surely." The unbiasedness of accepted samples is unaffected by the cap.

`ints` composes `int_range` and short-circuits the first `Err`.

---

## §5. Implementation plan (pure stdlib, zero runtime changes — proven, not assumed)

1. `std/prng.ail`: `Gen` newtype, `PrngError`, `ushr` (internal), SplitMix64 step exactly as §3.3,
   the core 4 primitives, the ergonomic layer, `split`.
2. `int_range` per §4 (bounded bitmask rejection, total `Result` API).
3. `examples/runnable/prng_demo.ail` — seeded, capability-free, deterministic output; gated by
   `make verify-examples`.
4. Docs: `std/prng` page; cross-link from `std/rand` ("need reproducible, cap-free randomness? see
   `std/prng`") and from the effects/capability table. The page includes the §3.3 pseudocode + the
   signed↔unsigned printing note so peer-language implementers aren't confused by negative values.
5. `ailang prompt`: one line teaching `std/prng` for the auditable/reproducible case vs `std/rand` for
   the convenience case.

**Routing (PROGRAM.md):** pure **AILANG fix / stdlib extension**. No lexer change (§3.2), no builtin
change (§3.3), no core change. The optional §6 builtin, if ever pursued, is an *extension* — and this
design does not depend on it.

## §6. Testing

- **Known-answer tests** — these exact vectors, produced by the live probe at HEAD and independently by
  the Python reference (unsigned form in parentheses):
  - `seed(0)`: `-2152535657050944081` (`0xE220A8397B1DCDAF`), `7960286522194355700`
    (`0x6E789E6AA1B965F4`), `487617019471545679` (`0x06C45D188009454F`)
  - `seed(42)`: `-4767286540954276203` (`0xBDD732262FEB6E95`), `2949826092126892291`
    (`0x28EFE333B266F103`), `5139283748462763858` (`0x47526757130F9F52`)
  - `next_float` after `seed(0)`: `0.8833108082136426`
  Locks the sequence so it can never silently change (and so Python/Go can match it). Extend to
  `seed(-1)` and first-10 at implementation time, always cross-generated from the Python reference.
- **Reproducibility**: `seed(k)` twice → identical streams; threading vs `ints(...)` agree.
- **`ushr` edge cases**: `ushr(-1,1) = 2^63−1`, `ushr(-1,63) = 1`, `ushr(1<<63, 63) = 1`,
  `ushr(1<<63, 1) = 2^62` — the direct regression tests for objections 1–2.
- **`int_range` contract**: `lo > hi` → `Err(InvalidRange)`; `lo == hi` → `Ok(lo)` and generator
  advances; span `≥ 2^63` → `Err(RangeTooWide)`; outputs always in `[lo, hi]`; roughly uniform over
  many draws; no draw sequence longer than 128 (structural — cap is a constant).

## §7. Stretch (optional, separate decision — NOT a dependency)

Add an unsigned/logical right-shift builtin (`ushr` / `>>>`). The quorum review forced the question
"is this actually required?" — the answer, live-verified, is **no**: the pure two-step helper is
bit-exact. A builtin would buy one fewer mask-and-shift per mixer step (micro-perf) and nicer teaching
material for future hashing/crypto code. Defer unless a second use case appears.

---

## Why this is worth doing (harness lens)

Per PROGRAM.md §4 this routes as an **AILANG fix / stdlib extension** (missing primitive), not a core
change. Beyond removing Friction #6's recurring "hand-roll an LCG" tax, it has a specific payoff for the
benchmark loop: **cross-language runtime comparisons become valid.** If the bench harness pins both
AILANG and Python to the same documented SplitMix64 seed, the food sequences match and a tick-count
delta finally measures the *language*, not the PRNG. That turns the single most-caveated number in the
Snake report ("656 vs 337 — not comparable") into a real signal for the next run. The quorum review
put this payoff at risk (a masked variant wouldn't interop); the revised plan keeps it at full
strength because the output is standard SplitMix64, verified bit-for-bit against Python.

## Out of scope

- Cryptographic randomness (use `std/crypto` / OS entropy via a capability — pure ≠ secure).
- Replacing or deprecating `std/rand` — the effectful global generator stays for the convenience case.
- A full distributions library (normal, poisson, …) — `std/prng` is the uniform substrate they'd build on.
- Full-width (`span ≥ 2^63`) `int_range` — defined as `RangeTooWide` in v1; `next_int` covers the need.
- A full-width-hex lexer extension (§3.2) — independent proposal if demand appears.

---

## Verification Log

All probes run 2026-07-23 at HEAD in this repo. Binary:

```
$ ailang --version
AILANG v0.30.0-126-g82084c1a9
Commit: 82084c1
Built:  2026-07-23_13:19:56
```

### Probe 0 — reference vectors + constants (Python 3, ground truth; machine-computed, not hand-derived)

```
$ python3 - <<'EOF'
M = (1<<64)-1
def sm64_stream(seed, n):
    s = seed & M; out=[]
    for _ in range(n):
        s = (s + 0x9E3779B97F4A7C15) & M
        z = s
        z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M
        z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M
        out.append(z ^ (z >> 31))
    return out
def signed(u): return u - (1<<64) if u >= (1<<63) else u
...
EOF
seed=0
  unsigned=16294208416658607535  signed=-2152535657050944081  hex=0xE220A8397B1DCDAF
  unsigned=7960286522194355700  signed=7960286522194355700  hex=0x6E789E6AA1B965F4
  unsigned=487617019471545679  signed=487617019471545679  hex=0x06C45D188009454F
seed=42
  unsigned=13679457532755275413  signed=-4767286540954276203  hex=0xBDD732262FEB6E95
  unsigned=2949826092126892291  signed=2949826092126892291  hex=0x28EFE333B266F103
  unsigned=5139283748462763858  signed=5139283748462763858  hex=0x47526757130F9F52
constants as signed decimal:
  0x9E3779B97F4A7C15 = -7046029254386353131
  0xBF58476D1CE4E5B9 = -4658895280553007687
  0x94D049BB133111EB = -7723592293110705685
```

### Probe 1 — full-width hex literal is rejected (objection 3 confirmed)

```
$ ailang check /tmp/prng_probe1.ail        # contains: let a = 0x9E3779B97F4A7C15;
Error: module loading error: failed to load /tmp/prng_probe1.ail: parse errors in /tmp/prng_probe1.ail:
could not parse "0x9E3779B97F4A7C15" as integer
```

Also: a `-9223372036854775808` literal fails the same way (`could not parse "9223372036854775808" as
integer`) — unary minus is tokenized separately; write min-int as `1 << 63`.

### Probe 2 — wrap and shift semantics (the doc's arithmetic premises)

```
$ ailang run --caps IO /tmp/prng_probe2.ail
wrap+:   -9223372036854775808        # 9223372036854775807 + 1  → wraps mod 2^64 ✓
wrap*:   -2691343689449507777        # -7046029254386353131 * 3 → matches 3·GOLDEN mod 2^64 ✓
sar -1:  -1                          # -1 >> 1 → arithmetic (sign-extending) shift confirmed
sar neg: -2004705051                 # negative >> 30 sign-extends (the objection-1 hazard, live)
shl63:   -9223372036854775808        # 1 << 63 = min int (sign bit)
shl64:   0                           # 1 << 64 = 0 (Go semantics; noted re: objection 2's k=0 edge)
mask-1:  9223372036854775807         # -1 & maxInt sanity
```

### Probe 3 — full SplitMix64, pure, at HEAD: bit-exact vs Python (objections 1, 2, 4 resolved)

`/tmp/prng_probe3.ail` contains **exactly** the `ushr`/`mix` code shown in §3.3 (state threaded via
plain lets; constants as the §3.2 signed decimals):

```
$ ailang run --caps IO /tmp/prng_probe3.ail
seed=0:  -2152535657050944081 7960286522194355700 487617019471545679
seed=42: -4767286540954276203 2949826092126892291 5139283748462763858
ushr(-1,1)=9223372036854775807 want 9223372036854775807
ushr(-1,63)=1 want 1
ushr(minint,63)=1 want 1
ushr(minint,1)=4611686018427387904 want 4611686018427387904
```

All six stream values equal Probe 0's signed reference values. All four `ushr` edge cases pass,
including logical-shifting a negative (post-multiplication-style) value — the exact scenario the
63-bit-masking plan mishandled.

### Probe 5 — `next_float` construction matches Python

```
$ ailang run --caps IO /tmp/prng_probe5.ail   # intToFloat(ushr(v,11)) / intToFloat(1 << 53)
float from seed(0) draw 1: 0.8833108082136426
$ python3 -c "print((16294208416658607535 >> 11) / (1<<53))"
0.8833108082136426
```

### Probe 6 — error-channel premise for §4 (no trap builtin in pure code)

```
$ grep -o 'registerBuiltinWithMeta("[a-zA-Z_]*"' internal/builtins/*.go | ... | grep -i "panic\|error\|fail\|trap\|abort\|assert"
(no matches)
$ ailang run --caps IO /tmp/prng_probe4.ail   # println(show(1 / 0))
(Go panic + stack trace — a crash, not a designed error value)
```

Hence `int_range`'s invalid-input behavior is specified via `Result` (`std/result.ail`:
`export type Result[a, e] = Ok(a) | Err(e)`), not via a trap.

### Incidental findings (implementation notes, verified while probing)

- Bitwise builtins are surfaced as **operators** (`&`, `^`, `|`, `<<`, `>>`, `~` — see
  `examples/runnable/bitwise.ail`), not as named functions; bare `shiftRight(...)` is an undefined
  variable. The stdlib module will use the operators.
- Statement-level `let (a, b) = ...` tuple destructuring does not parse; tuples destructure via
  `match` (`examples/runnable/guards_basic.ail`). API examples in the docs use `match`.
- Negative shift amounts are a defined runtime error (`RT_SHIFT`,
  `internal/builtins/math_bitwise.go`) — loud, which is what `ushr`'s out-of-contract `k=0` hits.
