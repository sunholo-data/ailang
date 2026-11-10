# M-V1-STDLIB-SURFACE-POLISH: the last cheap fixes to the Stable stdlib before it freezes

**Status**: Planned
**Target**: v0.58.x (the last breaking window before the v1.0.0 soak)
**Priority**: P1 — not a soundness blocker, but every item is now-or-2.0
**Estimated**: 9–11.5 working days (estimate; M4 is 3–4 of them and shrinks to 0.5 if Mark picks
option B for `Json` numbers)
**Dependencies**: none to start M1. M4 interacts with [m-v1-match-exhaustiveness](m-v1-match-exhaustiveness.md);
the tier fields feed the Stable-golden diff in [m-v1-release-gates](m-v1-release-gates.md).
**Planner-Lane**: opus-required for M1 and M4 (parser, iface, Json ADT); codex-ok for M2, M3, M5
**Implements**: §3.3 of [v1-release-readiness-review-2026-11-09](../v1-release-readiness-review-2026-11-09.md)

## 1. What this doc decides

The ratified Stability Promise ([stability.md](../../../docs/docs/reference/stability.md)) freezes
the golden interface of 17 Stable stdlib modules (280 exports). After the tag, a rename, an
argument reorder, a removed constructor or a changed result is a 2.0 change, and the promise
itself lists "compile-time deprecation warnings" as a follow-up: **there is no deprecation
mechanism today** (V1). This doc gives the complete change list for those 280 exports, the small
mechanism the renames need (`@deprecated`, `@experimental`), and a ruling on `Json` numbers.

It is deliberately conservative. Of the 12 naming breaks the review lists, it renames 10 and keeps
`PI()`/`E()`. Of the argument-order complaints, it changes one function (`list.member`, by
deprecating it in favour of `list.contains`) and keeps `take`/`drop`/`join` as documented
exceptions, because the measured usage makes a rename cost more than the consistency buys.

## 2. Method and instruments

- **Surface:** every `^export` line of the 17 Stable `std/*.ail` files (280 funcs, 6 types; matches
  the review's count). Signatures from the review-time `ailang iface` dump, cross-checked by count.
- **Usage counter:** a scratch script (not committed) that attributes a reference to a module only
  through a selective import (`import std/list (nth_or)`, aliases followed) or a module alias
  (`L.nth_or`); import lines and `--` comments are excluded. Bare `import std/list` exposes nothing
  (probe: `head_or` undefined), so these two forms are complete.
- **Corpora:** this repo's `std/`, `examples/` (556 .ail), `benchmarks/` (incl. `frontier_refs`),
  `tests/`; banked model solutions = the `code` field of every `lang: ailang` row under
  `eval_results/` (23,280 rows; 8,900 since 2026-08-01); read-only sibling checkouts
  `ailang-parse` (93), `docparse` (44), `email-parse` (72), `daneel` (80), `ailang-packages` (171),
  `ailang-world` (39), `mk-main` (motoko, 560), `stapledons_voyage` (23) — worktree directories
  excluded, no git operations, so local refs may be stale.
- **Probes:** `ailang` built from `51f6636` (v0.58.0-dev), evaluator and `--bytecode`.

## 3. Conventions, derived from the data

**Naming.** 268 of 280 Stable exports are lowerCamelCase. The breaks: 10 snake_case names
(`head_or`, `last_or`, `nth_or`, `charAt_or`, `abs_Int`, `abs_Float`, `rand_int`, `rand_float`,
`rand_bool`, `rand_seed`), 2 upper-case nullary functions (`PI`, `E`), and two deliberate styles:
the terse `Json` builders (`jn`, `jb`, `jnum`, `jint`, `js`, `ja`, `jo`, `kv`) and the
`Str`/`string` abbreviation split (`intToStr`, `floatToStr` vs `stringToInt`, `stringToFloat`).
`std/fs` uses a `*Result` suffix for the `Result`-returning twin of an aborting function;
`readFileBytes` and `readFileRaw` return `Result` without the suffix.

**Argument order.** Of the 125 exports with two or more parameters:

| Rule | Conforming | Outliers |
|---|---|---|
| R1 — higher-order: function first, traversed collection last | 24 of 27 (`map`, `filter`, `foldl`, `foldChars`, `option.map`, `result.flatMap`, …; `takeMap(n, f, xs)` keeps f before xs) | `string.foldSlices(s, delim, acc, f)`, `string.mapSlicesJoin(s, delim, f)`, `array.accumulate(arr, writes, combine)` |
| R2 — everything else: the subject (collection, string, path, `Json`) first | 92 of 98 conform or have no subject such as `pow` or `range` (`nth(xs, i)`, `map.lookup(m, k)`, `set.member(s, x)`, `option.getOrElse(o, d)`, every `string` and `fs` function) | `list.member(x, xs)`, `list.take(n, xs)`, `list.drop(n, xs)`, `string.join(sep, xs)`, `array.scale(s, v)`, `array.axpy(a, x, y)` |

So the library has one convention, data-first except for the function in a higher-order call,
and nine outliers. `map.member(m, k)` and `set.member(s, x)` conform; the review's "member is
inconsistent" is really "`list.member` is the outlier, and `list.contains` already has the right
order" (`std/list.ail:106`; the comment at list.ail:320 says "argument order flipped vs contains").

## 4. The change list

Every row was read at the cited line; behaviour rows were reproduced (§12). "Aliased" means the old
name stays as a `@deprecated` forwarding wrapper through the v0.58.x line (§6).

| # | Module | Current (file:line) | Proposed | Kind | Rationale |
|---|---|---|---|---|---|
| R1 | list | `head_or(xs, d) -> a` (list.ail:138) | `headOr` | rename, aliased | snake_case |
| R2 | list | `last_or(xs, d) -> a` (list.ail:142) | `lastOr` | rename, aliased | snake_case |
| R3 | list | `nth_or(xs, i, d) -> a` (list.ail:129) | `nthOr` | rename, aliased | snake_case |
| R4 | string | `charAt_or(s, i, d) -> string` (string.ail:175) | `charAtOr` | rename, aliased | snake_case |
| R5 | math | `abs_Int(int) -> int` (math.ail:85) | `absInt` | rename, aliased | snake_case |
| R6 | math | `abs_Float(float) -> float` (math.ail:82) | `absFloat` | rename, aliased | snake_case |
| R7 | rand | `rand_int(min, max) -> int ! {Rand}` (rand.ail:12) | `randInt` | rename, aliased | snake_case |
| R8 | rand | `rand_float(min, max) -> float ! {Rand}` (rand.ail:16) | `randFloat` | rename, aliased | snake_case |
| R9 | rand | `rand_bool() -> bool ! {Rand}` (rand.ail:20) | `randBool` | rename, aliased | snake_case |
| R10 | rand | `rand_seed(int) -> () ! {Rand}` (rand.ail:24) | `randSeed` | rename, aliased | snake_case |
| R11 | fs | `readFileBytes(path) -> Result[string, string] ! {FS}` (fs.ail:26) | `readFileBase64` | rename, aliased | returns base64 text, not `bytes` (probe: `"hi"` → `Ok(aGk=)`); `readFileRaw` stays the bytes reader |
| R12 | fs | `glob(root, suffix) -> [string] ! {FS}` (fs.ail:209) | `walkWithSuffix` | rename, aliased | suffix filter, not a glob (probe: `"*.ail"` → `[]`) |
| R13 | io | `printErr(s) -> () ! {IO}` (io.ail:38) | `eprint` | rename, aliased | not a duplicate of `eprintln` (no newline vs newline); makes `print`/`println`/`eprint`/`eprintln` |
| R14 | bytes | `slice(b, start, len) -> Option[bytes]` (bytes.ail:70) | `sliceLen` | rename, aliased | *new finding*: length-based, while `string.substring(s, start, end)` and every mainstream `slice` are end-based; same-typed ints, so a confused call type-checks |
| D1 | list | `member(elem, xs) -> bool` (list.ail:322) | use `contains(xs, x)` | deprecate → remove | the one R2 outlier with an existing correctly-ordered twin; also not SMT-encodable (source comment) |
| D2 | array | `unsafeGet(arr, i) -> a` (array.ail:70) | use `get` | deprecate → remove | identical behaviour: both bounds-check and abort with the same message in both engines (probe) |
| B1 | json | `decode(s)` accepts trailing data (json.ail:18 → `internal/builtins/json_decode.go:291`, loop stops after the first value) | `Err("decode: trailing data at offset N")`; whitespace still allowed | behaviour | `decode("1 2")`, `decode("[1]]")`, `decode("{\"a\":1} xyz")` are all `Ok`, while `decodeFloatArray` in the same module rejects trailing data |
| B2 | json | `decode("{\"\": 1, \"b\": 2}")` drops the `""` key in the evaluator, keeps it in the VM (`json_decode.go:432` treats `lastKey == ""` as "no key") | keep the key; one decoder for both engines | behaviour (bug) | *new finding*: silent data loss and an engine divergence; there are three decode implementations (`internal/builtins`, `internal/vm`, a legacy one in `internal/eval`) |
| B3 | json | `type Json = … JNumber(float) …` (json.ail:10) | add `JInt(int)` (option A, §7) | behaviour + type | decided now or never: a constructor added after 1.0 breaks every exhaustive match |
| B4 | json | `getInt` truncates (json.ail:205; `1187.50` → `Some(1187)`) | `None` unless the number is integral | behaviour | resolves triage `json-getint-silent-float-truncation`; same number contract as B3 |
| B5 | json | `jint(n) = JNumber(intToFloat(n))` (json.ail:44) | `JInt(n)` | behaviour | AILANG cannot round-trip its own `int`: `encode(jint(1234567890123456789))` = `1234567890123456800` (probe) |
| T1 | json | `repair`, `asBoolLoose`, `getStringArrayStrict`, `getNumberArrayStrict`, `decodeFloatArray` (json.ail:25, 131, 283, 307, 327) | `@experimental` | retier | boundary/niche helpers; 0 downstream files use any of them |
| T2 | string | `decodeQuotedPrintable`, `foldSlices`, `mapSlicesJoin`, `startsWithIgnoreCase` (string.ail:212, 190, 198, 217) | `@experimental` | retier | 0 downstream uses except one `decodeQuotedPrintable`; `foldSlices`/`mapSlicesJoin` are R1 outliers that can then be reordered in a 1.x minor if wanted |
| T3 | list | `takeMap`, `takeFlatMap` (list.ail:193, 187) | `@experimental` | retier | workarounds for strict evaluation; redundant if fusion ever lands |
| T4 | array | `dot`, `axpy`, `scale`, `add`, `sub`, `mul`, `sum`, `argmax`, `scatterAdd`, `accumulate`, `encodeF64LE`, `decodeF64LE`, `encodeF32LE`, `decodeF32LE` (array.ail:105–170) | `@experimental` | retier | the numerics kernels (M-NUMERICS); still growing; 0 downstream files; `axpy`/`scale` keep BLAS order |
| P1 | prompt | `if member(key, m) && lookup(key, m) == target` (`prompts/v0.16.12.md:1296`) | `if member(m, key) && lookup(m, key) == Some(target)` | teaching fix | the taught line fails: `cannot unify Map[α1, α2] with string`; the fix type-checks (probe) |

**Totals:** 14 renames, 2 deprecate-and-remove, 5 behaviour changes (B3+B5 are one change), 25
functions retiered to Experimental, 1 prompt fix. No signature of a surviving Stable name changes.

### Kept, and documented instead

| Symbol | Why it stays |
|---|---|
| `math.PI()`, `math.E()` | Mathematical notation, and the prompt already says "CALL them with `()`". A lowercase `e` would be shadowed by the idiomatic `Err(e) =>`. `PI` is in 15 banked rows, all passing. |
| `list.take(n, xs)`, `list.drop(n, xs)` | Count-first, as in Haskell and OCaml. 621 and 353 banked rows; 38 and 14 downstream files; `warn_take_after_flatmap.go` keys on it. |
| `string.join(sep, xs)` | Python's mental model. 853 banked rows, 132 downstream files, and the `++`-on-strings diagnostic teaches `join(sep, parts)`. |
| `intToStr`/`floatToStr` vs `stringToInt`/`stringToFloat` | 361 and 1,964 banked rows; 59 and 79 downstream files. |
| `string.find` returning `-1` | The one `Option`-convention outlier among lookups; 232 banked rows, 71 downstream files. An additive `findOpt` can come in 1.x. |
| `foldSlices`/`mapSlicesJoin`/`accumulate` order; `axpy`/`scale` order | Become Experimental (T2, T4); `(s, delim)` mirrors `split`, `axpy` is BLAS. |
| `getStringArray*OrEmpty`, permissive `getStringArray`/`getNumberArray` | Prompt-taught; `getStringArrayOrEmpty` is in 96 banked rows and 7 downstream files. Stay Stable. |
| Json builders `jn`, `jb`, `jnum`, `js`, `ja`, `jo`, `kv` | Deliberately terse ("reduce AI fumbles", json.ail:29). |
| `net.httpGet` | `std/net` is Experimental, so it is outside this promise. The prompt teaches it 7 times; delete the stale "deprecated" comment (net.ail:30) instead. Its error-as-body result (`"E_NET_REQUEST_FAILED: …"`, net.ail:64) is an Experimental-tier issue for the backlog. |

## 5. Blast radius, per symbol

`files/refs`. Banked = rows using it all-time (passing) | since 2026-08-01 (passing). Prompt =
lines in the active v0.16.12 prompt. Live docs exclude the frozen `docs/docs/prompts/` copies.

| Symbol | examples | tests+bench | downstream (repo: files/refs) | banked all | banked recent | prompt | live docs |
|---|---|---|---|---|---|---|---|
| `head_or` | 1/4 | 0 | daneel 1/1 | 7 (7) | 6 (6) | 0 | 0 |
| `last_or` | 1/2 | 0 | ailang-parse 1/1, email-parse 1/4, daneel 2/6 | 10 (10) | 10 (10) | 0 | 0 |
| `nth_or` | 4/12 | 0 | ailang-parse 4/13, email-parse 2/2 | 50 (46) | 50 (46) | 0 | 0 |
| `charAt_or` | 1/2 | 0 | — | 4 (4) | 4 (4) | 0 | 0 |
| `abs_Int` / `abs_Float` | 1/2 each | 0 | ailang-parse 1/1 (`abs_Float`) | 3 (0) / 7 (6) | 1 / 2 | 2 | 0 |
| `rand_int` | 4/12 | 0 | docparse 2/9, motoko 3/19, stapledons 1/1 | 0 | 0 | 3 | 3 |
| `rand_float` / `rand_bool` | 2/4, 2/3 | 0 | — | 0 | 0 | 0 | 2 |
| `rand_seed` | 2/3 | 0 | motoko 2/6 | 0 | 0 | 1 | 2 |
| `readFileBytes` | 3/3 | 0 | ailang-parse 8/15, docparse 1/1, email-parse 1/1, daneel 2/2 | 0 | 0 | 1 | 2 |
| `glob` | 1/4 | 0 | email-parse 2/6 | 0 | 0 | 3 | 0 |
| `printErr` | 1/1 | 0 | — | 2 (1) | 2 (1) | 1 | 1 |
| `bytes.slice` | 4/10 | tests 1/1 | packages 1/1, motoko 1/1 | 1 (1) | 1 (1) | 0 | 0 |
| `list.member` | 2/5 | 0 | ailang-parse 1/7 | 13 (9) | 9 (8) | 0 | 0 |
| `array.unsafeGet` | 0 | 0 | motoko 2/2 | 3 (2) | 3 (2) | 1 | 0 |
| `JNumber` (pattern/ctor) | 3 files | tests 3 files | motoko 2 files, world 1 file | 102 (12) | 1 (0) | 2 | — |
| `getInt` (B4) | 5/7 | 0 | 54 files across 7 repos | 786 (605) | 470 (377) | 3 | — |
| `json.decode` (B1, B2) | 14/19 | tests 4/7 | 171 files | 1,211 | 601 | many | — |

- **This repo:** 15 `.ail` files use R1–R13 or D2 (`examples/`, two `expected_fail/` rand fixtures),
  plus glob/member/slice users above. Inside `std/`, the old names appear only in their own
  definitions, `head_or`'s call to `nth_or`, and comments in `std/ai.ail`, `std/net.ail`,
  `std/string.ail`. Go code calls the `_rand_int`/`_fs_readFileBytes`/`IO.printErr` builtins and
  effect ops, which are Internal and do **not** change; the Go codegen table
  (`internal/gen/golang/codegen_expr_simple.go:227`) keys on `rand_int` and needs the new names.
- **Eval cost.** Since 2026-08-01, 67 of 8,900 banked AILANG rows (0.75%) use any renamed or
  removed name; 65 of them are agent-mode, where the model discovered the name with tools — none
  of R1–R4 is in the prompt. With aliases those rows lose nothing during v0.58.x, and after
  removal the error names the replacement (§6.3). No rename here is heavy enough to drop on
  eval-cost grounds; the heavy ones (`take`, `join`, `intToStr`, `stringToInt`) are already kept.
- **Downstream:** about 40 files in 6 repos need edits, all mechanical except 1 `member` file.
  `ailang-packages` needs one (`bytes.slice`), `ailang-world` none for renames; world and motoko
  each have one direct `JNumber` match that M4 affects.

## 6. Mechanism: `@deprecated` and `@experimental`

### 6.1 Syntax and compiler path

The parser already has a generic declaration-annotation path: `parseAnnotation`
(`internal/parser/parser_decl.go:17`) handles 19 names, stores them in `FuncDecl.Annotations`
(`internal/ast/ast_decl.go:58`), and `@verify(depth: 2)` before `export pure func` parses today
(probe). `@deprecated` is currently `PAR_UNKNOWN_ATTRIBUTE` (probe). Add two cases:

```
@deprecated("headOr")
export pure func head_or[a](xs: [a], default: a) -> a = headOr(xs, default)

@experimental
export func repair(s: string) -> Result[string, string] { _json_repair(s) }
```

`@deprecated` takes one or two string literals (the replacement, then an optional note such as
`"arguments swap: contains(xs, x)"`) via the existing `parseStringListAnnotation(…, 1, 2, …)`.
`@experimental` takes none. Both are accepted on any exported function, so package authors can
use them; the annotation syntax itself is tiered Experimental for 1.0.

1. **Interface.** This follows the contract in m-v1-release-gates §4.3. `FuncJSON`
   (`internal/iface/json.go:33`) and `TypeJSON` gain `Tier string json:"tier"`
   (`stable|experimental|internal`) on **every** entry. The value is the module default from
   release-gates' `std/STABILITY.toml`, overridden per function by `@experimental`; types take the
   module default. `FuncJSON` also gains `ReplacedBy string json:"replaced_by,omitempty"`, set by
   `@deprecated`. Deprecation is a separate axis from tier: a deprecated Stable function stays
   `tier: stable`, so removing it still fails the gate after 1.0. The vocabulary matches
   `builtins.Stability` (metadata.go:30). Emitting `tier` everywhere changes all 49 goldens once.
2. **Warning pass.** `DetectDeprecatedUses` beside `DetectArgOrderWarnings` in
   `pipeline_module_phases.go:281` and `pipeline_single.go:205`, keyed on `VarGlobal{Module, Name}`
   exactly as `warn_split_args.go` is. It must visit every `VarGlobal`, not only call heads, so
   `map(abs_Int, xs)` warns too. Output goes to stderr like the existing split-arg warning, which
   appears on a second (cached) run and keeps stdout clean (probe), so stdout-graded evals are
   unaffected:
   `warning[DEPRECATED]: std/list.head_or is deprecated; use headOr` with the location and, for D1,
   the rewritten call.
3. **`ailang docs`** (`cmd/ailang/docs.go`) is a line scraper, and any non-comment line resets the
   pending doc summary (docs.go:327). It must skip `@…` lines and print `[experimental]` or
   `[deprecated → headOr]`. That makes two readers of one annotation, so add a test that the
   scraper's tier set equals the golden's tier set for every std module.
4. **Hashes.** `HashProjection` (`internal/iface/hash_projection.go`) copies `FuncJSON` wholesale,
   so it must clear `Tier` and `ReplacedBy`: marking a function does not change a package's
   `interface_hash`. Bump `cacheKeyVersion` v10 → v11 (`internal/pipeline/cache_key.go:50`), because
   a cached iface without the field would silently drop warnings (the v4 → v5 `DerivedEq` rule).

### 6.2 What the Stable-golden gate must read (contract for m-v1-release-gates)

- The tier lives **per function** in `.stdlib-golden/<module>.json` as `tier`, always present,
  which is what release-gates §4.3 asks for. `@experimental` is the only per-function source;
  module defaults come from release-gates' manifest. One annotation and one manifest, with no
  doc-comment tags, so the gate and `ailang docs` both read the iface or the golden.
- `replaced_by` is informational. Removing a `tier: stable` entry fails after the tag whether or
  not it is deprecated, which matches stability.md ("can only be removed in a later major").
  Lowering a tier fails; Experimental → Stable is allowed.
- Constructor lists are part of `types[].ctors`, so a `Json` constructor change shows as a type
  change in a Stable module and must fail after the tag.
- `make freeze-stdlib` is re-run once per milestone below, and each commit says why (the script's
  own header asks for that).

### 6.3 Alias lifetime (decision)

| Option | 1.0 surface | Downstream | Fits "1.0 = last 0.58.x + docs"? |
|---|---|---|---|
| (a) keep aliases in 1.0 as Stable | 16 extra names frozen forever, two names per function in `ailang docs` | no work | yes |
| (b) **remove in the last v0.58.x, before the soak** | clean | migrate during 0.58.x; the soak proves it | **yes** |
| (c) keep into 1.0 marked deprecated, outside the promise; remove in 1.1 | clean from 1.1 | longer window | yes, but needs a promise clause, and the removal is never soaked |

**Recommendation: (b).** The review's own sequence makes v1.0.0 a docs-and-version cut of the last
v0.58.x, so "remove at 1.0" in the literal sense would put code changes into the tag. Removing in
the v0.58.x that starts the soak means the 7-day soak, with downstream pins moved, tests exactly
the alias-free surface. Every downstream consumer is first-party and covered by the planned
downstream gate. Removal must not leave AI authors stranded, so two diagnostics stay permanently
(they are Internal, not promised):

- **IMP010 did-you-mean** ranks a name that normalises (case-folded, `_` removed) to an export
  first. Today `nthOr` suggests `nth` (probe), so after removal `nth_or` would get the wrong hint.
  Add a 16-row removed-name table for the renames that normalisation cannot find
  (`readFileBytes → readFileBase64`, `member → contains (swap args)`, …).
- **Qualified access to a missing member** currently reports the *alias*:
  `L.nthOr` → `undefined variable: L` (probe), while `L.nth_or` works. After removal every
  `L.head_or` would say `undefined variable: L`. Fix it to name the member and apply the same hint.

A name removed for semantic reasons (`glob`, `bytes.slice`) is retired for all of 1.x: re-adding
`glob(string, string)` as a pattern glob would let 0.58-era code compile again with a silently
different meaning.

## 7. `Json` numbers (B3–B5)

Facts (probes): `decode("9007199254740993")` gives `9007199254740992`;
`{"id": 1234567890123456789}` re-encodes as `1234567890123456800`; `jint(1234567890123456789)`
encodes the same wrong value. A literal outside float64 range is already an `Err`
(`json_decode.go:35`), so the module already prefers loud failure to silent rounding.

| Option | What changes | Breaks | Cost |
|---|---|---|---|
| **A. Add `JInt(int)` now** | Decode gives `JInt` for integer-syntax literals that fit int64 (not `-0`), `JNumber` otherwise; `jint` builds `JInt`; `encode` prints `JInt` exactly; `asNumber`, `getNumber`, `filterNumbers`, `allNumbers`, `getNumberArray*` accept both; `getInt` takes `JInt`, or an integral `JNumber` | Exhaustive `Json` matches gain an arm (loud once non-exhaustive is an error). Matches with a `_` arm that expect `JNumber` for integers fall through **silently**: `std/jwt.ail:180`, `std/sem.ail:274`, `mk-main/src/eval/journal/association.ail:230`, `ailang-world/.../se_tools/read.ail:197`. `decode("1") == jnum(1.0)` becomes false. | 7 Go files / 12 `"JNumber"` sites (decode, encode, yaml, `embed/convert.go`, VM copies), std json/jwt/sem, prompt; 3–4 days |
| B. Keep `JNumber(float)` | Document the 2^53 limit as part of the Stable contract; still do B4 | Nothing | 0.5 day; the limit is permanent for 1.x, since an int constructor is 2.0 |
| C. Lossless `JNumber(string)` | Keep the literal text | Every `JNumber(x)` binding changes type: loud, but the widest churn, and the worst ergonomics for models | 4+ days |

**Round trip through text (ruled 2026-11-10, implementation PR #2223).** `encode` keeps printing a
whole-number `JNumber` with integer syntax (`jnum(3.0)` → `3`), so the numeric value round-trips
and the constructor does not: `decode(encode(jnum(3.0)))` is `Ok(JInt(3))`. The alternative —
printing float syntax (`3.0`, `100000000000000000000.0`) so the constructor survives — was measured
on a scratch build and rejected: 0 changes to in-repo example or program output, but the
`json_encode` benchmark's expectation would change; 37 of 236 banked passing std/json solutions
would change output (all `json_encode`; those reproducing their expected output would fall from 53
to 16); and about 514 of 545 downstream `jnum(` call sites (368 `jnum(intToFloat(…))`-style, 146
whole literals, across motoko, docparse, ailang-packages, ailang-parse, email-parse, daneel) would
print `.0`. Do not reopen without new data. (`std/yaml` does write float syntax, because YAML's own
`!!int`/`!!float` distinction must survive its bridge.)

This agrees with m-v1-match-exhaustiveness, which finds 1 `Json` match with no catch-all
(motoko). That one fails loudly with the witness `JInt(_)`. The catch-all matches are the silent
ones.

**Recommendation: A**, with one mitigation: a transitional warning in v0.58.x, keyed on
constructor identity like §6.1, for any match over `Json` that has a `JNumber` arm and no `JInt`
arm. That turns the silent fall-through into a visible one for the whole window. Why A: AILANG's
`int` is 64-bit and the language sells determinism, yet today it cannot round-trip its own ints
through its own JSON. The review's case against `show` elision ("anything that serialises,
compares or hashes … is silently wrong") applies here too. Python and serde users, who are most
models, expect an int/float split. Direct `JNumber` matching is rare in recent model output (1 of
8,900 rows), and the accessors, which carry the volume (`getInt` 470 recent rows), keep working.
If Mark prefers to protect the soak's risk budget, B is a coherent fallback, but it should be
chosen knowingly, since it is the irreversible one.

## 8. Behaviour fixes (B1, B2)

- **B1:** after the top-level value, read one more token and require `io.EOF`; otherwise return
  `Err("decode: trailing data at offset N")` using `Decoder.InputOffset()`, matching
  `decodeFloatArray`'s wording. Leading and trailing whitespace stay legal. Callers that fed
  model output with trailing prose to `decode` (std/ai.ail:24 suggests `decode(raw)`) now get
  `Err`; `json.repair` remains the lenient path.
- **B2:** treat an empty key as a key, using an explicit "expecting key" state instead of
  `lastKey == ""`. Collapse the three `_json_decode` implementations to one shared Go function
  used by both engines, or the next fix lands in one engine only. This is the seam the bug lives
  in.

## 9. Conflict surface (parser, iface, elaborate-adjacent)

1. **Position extended:** the `@name` slot before a top-level `func` declaration. Two new names.
2. **Already there:** 19 annotations (`@verify`, `@route`, `@mcp_*`, `@optional`,
   `@allow_empty_ok`, `@raw`, `@nowrap`, `@noexpose`, `@nomcp`). User code cannot contain
   `@deprecated` or `@experimental` today (`PAR_UNKNOWN_ATTRIBUTE`), so no program changes meaning.
3. **Disambiguation:** by identifier in the existing `switch name`; no new tokens.
4. **Must still work:** `examples/runnable/contracts/per_function_depth_verify.ail` (`@verify`),
   `examples/runnable/mcp_tools.ail` (`@mcp_*`), `examples/runnable/json_jint.ail`,
   `examples/runnable/json_parsing.ail`, `examples/runnable/yaml_roundtrip.ail`,
   `tests/json_accessors_test.ail`.
5. **Deliberately changes:** new stderr warnings; `decode` rejects trailing data; `""` keys are
   kept; `getInt` stops truncating; under A, `Json` has seven constructors.

## 10. Milestones

Each milestone ships with a breaking changelog fragment, a deliberate `make freeze-stdlib`, and
`make test-core`, `make verify-examples` and `make verify-stdlib` green. Engine parity means the
evaluator and the strict VM.

**M1 — Annotation mechanism (opus-required, ~3 d).** Parser cases, `FuncJSON` fields,
`HashProjection` clearing, cache key v11, `DetectDeprecatedUses`, `ailang docs` tiers plus the
consistency test, IMP010 normalised suggestion, qualified-missing-member error.
*Accept:* a fixture module with `@deprecated("g") export func f` warns at each use (including
`map(f, xs)` and `M.f`), on a second cached run too, and nothing reaches stdout; the golden for that
fixture carries `"replaced_by": "g"` and every entry carries `tier`; its `interface_hash` equals
the unannotated one;
`import std/list (nthOr)` suggests `nthOr` once it exists. *Mutation:* drop the `VarGlobal`
non-head visit and confirm the `map(f, xs)` test fails; drop the `HashProjection` clearing and
confirm the hash test fails.

**M2 — Renames, aliases, prompt (codex-ok, ~1.5 d, needs M1).** R1–R14 as new definitions; old
names become `@deprecated` one-line forwarders; D1/D2 `@deprecated` with notes; migrate the 15+
repo files; prompt v0.16.13 via prompt-manager (P1 plus the renamed lines 54, 574, 593–601, 692,
694, 711, 803, 1117, 2597); Go codegen table; `tools/migrate-stdlib-0.58.sed`.
*Accept:* `ailang check` over `examples/` and `tests/` reports zero `DEPRECATED`; a fixture using
every old name still runs with 16 warnings; the corrected P1 line type-checks.
*Mutation:* point one alias at the wrong target (`lastOr` → `headOr`) and confirm the alias
equivalence test fails.

**M3 — `json.decode` (codex-ok, ~1–1.5 d).** B1, B2, and one shared implementation.
*Accept:* `decode("1 2")`, `"[1]]"` and `"{} x"` return `Err`; `" 1 \n"` is `Ok`;
`{"": 1, "b": 2}` round-trips in both engines; the decode test table runs under both engines.
*Mutation:* restore the early `break` and confirm the trailing-data test fails.

**M4 — `Json` number contract (opus-required, 3–4 d under A; 0.5 d under B).** Under A: `JInt`,
the decode rule, encode, accessors, `jint`, every Go producer (yaml, `embed/convert.go`, VM),
std jwt/sem, the transitional match warning, the prompt JSON section ("JSON numbers are floats"
at line 1709). B4 under either option.
*Accept:* `9007199254740993` and `1234567890123456789` round-trip exactly; `getInt` on `1187.50`
is `None` and on `30.0` is `Some(30)`; a `Json` match with a `_` arm and no `JInt` arm warns.
*Mutation:* make decode emit `JNumber` for integers and confirm the round-trip test fails.

**M5 — Retier and promise text (codex-ok, ~0.5–1 d, needs M1).** `@experimental` on T1–T4;
`stability.md` gains a per-function Experimental list, the deprecation mechanism (closing the
follow-up at stability.md:483), the `Json` number contract and §3's conventions; delete the stale
`net.ail:30` comment. *Accept:* the golden marks exactly the 25 T-rows; the docs test passes.

**M6 — Remove aliases (last v0.58.x before the soak, ~0.5 d).** Delete the 16 forwarders and
re-freeze. *Accept:* the downstream gate is green with no `DEPRECATED` output; each removed name
gives the replacement hint, both bare and qualified.

## 11. Migration guide, changelog, docs

**Migration (for the 0.x → 1.0 guide the review asks for):** run `ailang check` and fix every
`warning[DEPRECATED]`; or apply `sed -E -f tools/migrate-stdlib-0.58.sed` to `.ail` files. That is
safe for R1–R11 and R13, whose old names are unique tokens. Edit by hand: `glob(` (a common word),
`bytes.slice`, `member(x, xs)` → `contains(xs, x)`, and `unsafeGet` → `get`, which can collide
with an imported `json.get` (use `get as arrayGet`). A rewrite mode in `ailang fmt` is not worth
building for about 40 downstream files.

**Changelog fragment drafts** (`changelogs/unreleased/`):

```markdown
### Changed — Stable stdlib renames before 1.0; old names warn until v0.58.N (BREAKING at removal)
`headOr`, `lastOr`, `nthOr`, `charAtOr`, `absInt`, `absFloat`, `randInt`, `randFloat`, `randBool`,
`randSeed`, `readFileBase64` (was `readFileBytes`), `walkWithSuffix` (was `glob`), `eprint` (was
`printErr`), `bytes.sliceLen` (was `slice`). `list.member` → `list.contains(xs, x)`;
`array.unsafeGet` → `array.get`. Old names warn and keep working through v0.58.x. See
design_docs/planned/v1_0_0/m-v1-stdlib-surface-polish.md.

### Fixed — `json.decode` rejected nothing after the first value, and dropped `""` keys (BREAKING)
`decode("1 2")` is now `Err("decode: trailing data at offset 2")`. An empty-string key is kept;
the evaluator dropped it and the VM did not.

### Changed — `Json` gains `JInt(int)`; `getInt` no longer truncates (BREAKING)
### Added — `@deprecated("replacement")` and `@experimental` function annotations
```

**Docs:** `reference/effects.md`, `guides/parameterised-effects.md`, `guides/go-interop.md`,
`guides/ai-effect.mdx` (renamed names); `reference/stdlib.md` (tiers); `stability.md` (M5). Do not
edit the frozen `docs/docs/prompts/v*.md` snapshots.

## 12. Verification log

| # | Claim | How verified |
|---|---|---|
| V1 | No deprecation or experimental annotation exists | `parseAnnotation` switch read; `@deprecated` → `PAR_UNKNOWN_ATTRIBUTE` (probe) |
| V2 | Annotations work before `export pure func` | `@verify(depth: 2)` probe: no errors |
| V3 | Prompt line 1296 fails; fix passes | `ailang check`: unify error / "No errors found" |
| V4 | decode accepts trailing data; `decodeFloatArray` does not | probe, evaluator and `--bytecode` |
| V5 | `""` key dropped in evaluator only | probe output differs between engines |
| V6 | 2^53+1 and 19-digit ids lose precision; `jint` too | probe |
| V7 | `get` and `unsafeGet` behave identically | probe, both engines |
| V8 | `readFileBytes` returns base64; `glob("*.ail")` is `[]` | probe |
| V9 | Split-arg warnings reach stderr on cached runs | probe ran twice; `2>/dev/null` leaves stdout clean |
| V10 | `L.nthOr` reports `undefined variable: L` | probe |
| V11 | Proposed names, `JInt`, `DEPRECATED` unallocated | grep over `std/`, `internal/`: no hits |
| V12 | `ailang docs` resets on non-comment lines | read docs.go:255–330 |

**Not verified:** whether `DetectDeprecatedUses` can see an imported std module's iface when that
module is served from cache (left to M1, with the cached-run acceptance test); the downstream
repos' remote state (local checkouts only, and `docparse`/`ailang-parse` may overlap); Go code in
sibling repos (world imports `serveapi`) that might construct `JNumber`; the WASM and REPL paths;
whether any `--policy` file names `std/fs` wrappers rather than effect ops (I believe ops, which do
not change).

## 13. Risks

- **M4 is the biggest blast radius in the window.** Silent fall-through is mitigated by the
  transitional warning, but only for code that is compiled during v0.58.x. If the soak budget is
  tight, take option B.
- **Window length.** Under 6.3(b), aliases may live only one or two weeks. External users, if any,
  get a short deprecation. Fallback: option (c).
- **Prompt churn.** A new prompt version perturbs eval comparisons; bundle P1, renames and the
  `Json` text into one version (v0.16.13).
- **Seams:** three decoders, two readers of the tier (golden, `ailang docs`), codegen's name table.
  Each has a consistency test in its milestone.

## 14. Axiom compliance

A1 Determinism +1 (exact int JSON round-trip; one decoder for both engines) · A2 Replayability 0 ·
A3 Effect legibility 0 · A4 Authority 0 · A5 Bounded verification 0 · A6 Concurrency 0 ·
A7 Machines first +1 (one naming rule; diagnostics name the replacement) · A8 Minimal syntax 0
(two annotations on an existing slot, no new tokens) · A9 Cost 0 · A10 Composability 0 · A11 Structured failure +1
(trailing data and lossy numbers become errors or values, not silent) · A12 Boundary 0.
**Net +3.** No hard violations.

## 15. Decisions for Mark

1. **Alias lifetime** — recommended (b): remove in the last v0.58.x, before the soak, and keep
   the permanent removed-name hints.
2. **`Json` numbers** — recommended A (`JInt` now, with a transitional warning). B is the coherent
   fallback; it makes the 2^53 limit permanent for 1.x.
3. **Keep list** — accept keeping `PI`/`E`, `take`/`drop`, `join`, `intToStr` and `find`?
   Recommended yes.
4. **R13 `printErr → eprint` and R14 `bytes.slice → sliceLen`** are beyond the review's list.
   Recommended yes, since each costs about 3 files; drop them if you want the minimum.
5. **25 Experimental retiers**, including the whole array kernel family — recommended yes.
6. **`@deprecated`/`@experimental` allowed in user packages** (syntax Experimental in 1.0)? Recommended yes.

## 16. Out of scope

Additive functions (`list.sort`, int `min`/`max`, `padLeft`, path helpers, `result.getOrElse`,
`findOpt`, a generic `abs`); the freeze-gate tooling (m-v1-release-gates); snake_case in
Experimental modules (`std/embedding.add_vectors`, `std/sem.decode_frame`); the `std/clock`
milliseconds documentation fix (triage `datetime-clock-ms-vs-seconds-docs`); `show` (§3.2 of the
review); `httpGet`'s error-as-body result.

## References

- [v1-release-readiness-review-2026-11-09.md](../v1-release-readiness-review-2026-11-09.md) §3.3
- [stability.md](../../../docs/docs/reference/stability.md); [m-v1-stability-promise](../../implemented/v0_29_0/m-v1-stability-promise.md)
- Triage: [json-getint-silent-float-truncation](../ailang-core-triage/json-getint-silent-float-truncation.md),
  [stdlib-number-naming-and-result-option-convention](../ailang-core-triage/stdlib-number-naming-and-result-option-convention.md),
  [serveapi-mcp-int-args-bind-float64](../ailang-core-triage/serveapi-mcp-int-args-bind-float64.md)
- Siblings: [m-v1-match-exhaustiveness](m-v1-match-exhaustiveness.md), [m-v1-release-gates](m-v1-release-gates.md)
- Precedent passes: `internal/pipeline/warn_split_args.go`, `strict_fallbacks.go`
