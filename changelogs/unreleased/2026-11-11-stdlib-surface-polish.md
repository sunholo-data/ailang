### Changed — Stable stdlib renames before 1.0; the old names still work and warn `TC_DEPRECATED_001` until they are removed before the v1.0.0 soak (BREAKING at removal) (2026-11-11)

Sixteen Stable stdlib names change (M-V1-STDLIB-SURFACE-POLISH M2,
`design_docs/planned/v1_0_0/m-v1-stdlib-surface-polish.md` §4). Each old name stays as a one-line
`@deprecated` forwarder with the same type, effects and behaviour on both engines, so nothing breaks
today; every use of one from another module warns `TC_DEPRECATED_001` naming the replacement. The
aliases are removed in the last 0.x release before the v1.0.0 soak (roadmap step 16); after that,
importing an old name is `IMP010` naming the replacement.

| Module | Old | New | Notes |
|---|---|---|---|
| `std/list` | `head_or` | `headOr` | |
| `std/list` | `last_or` | `lastOr` | |
| `std/list` | `nth_or` | `nthOr` | |
| `std/string` | `charAt_or` | `charAtOr` | |
| `std/math` | `abs_Int` | `absInt` | |
| `std/math` | `abs_Float` | `absFloat` | |
| `std/rand` | `rand_int` | `randInt` | |
| `std/rand` | `rand_float` | `randFloat` | |
| `std/rand` | `rand_bool` | `randBool` | |
| `std/rand` | `rand_seed` | `randSeed` | |
| `std/fs` | `readFileBytes` | `readFileBase64` | it returns base64 text; `readFileRaw` returns bytes |
| `std/fs` | `glob` | `walkWithSuffix` | a suffix filter, not a glob; `glob` stays retired for all of 1.x |
| `std/io` | `printErr` | `eprint` | `print`/`println`/`eprint`/`eprintln` |
| `std/bytes` | `slice(b, start, len)` | `sliceLen` | the third argument is a length; `slice` stays retired for all of 1.x |
| `std/list` | `member(x, xs)` | `contains(xs, x)` | **the arguments swap** |
| `std/array` | `unsafeGet` | `get` | identical behaviour; import `get as arrayGet` if it collides with `json.get` |

**Order:** none of the 14 new names exists in v0.59.0 or v0.60.0, so downstream code moves to them
only after its pin is on a release containing this change (`member` → `contains` and `unsafeGet` →
`get` use names that already exist and can change any time). Release → downstream migrates →
aliases removed (roadmap step 16).

**Migrate:** `sed -E -i.bak -f tools/migrate-stdlib-renames.sed <files>` rewrites the twelve unique
names; edit `glob`, `slice`, `member` and `unsafeGet` by hand; then `ailang check` until no
`TC_DEPRECATED_001` is left. Every use in this repository (std/, examples/, tests, docs, the teaching
prompt) has moved; only the alias-parity tests use the old names. Teaching prompt **v0.16.14** (active)
teaches the new names, adds a one-paragraph rename note and fixes the `member`/`lookup` sample, which
had the `std/map` arguments reversed and did not type-check.

### Added — `@deprecated("replacement"[, "note"])` and `@experimental` function annotations; warning `TC_DEPRECATED_001` (2026-11-11)

`@deprecated` on an exported function makes every use of it from another module — a call, a value
(`map(abs_Int, xs)`), a module-qualified `L.f`, an aliased import `f as g` — warn once per use site, on
stderr, in `check --json` `warnings[]` (code, file, line, column, message, suggestion) and in
`check --format agent`, cold and warm compile cache alike. A deprecated name imported and never used warns
once, at that name in the import list (the import alone fails once the alias is removed); defining
it and uses inside the defining module do not warn; a wrapper module that forwards to a
deprecated function warns at its own use, not at its callers. Uses inside an installed dependency
package (`pkg/<other>/...`) are not reported to the project that imports it — that package's own
`check` reports them (docparse would otherwise get 15 sites from `sunholo/ailang_parse`, repeated in 21
files). `@experimental` puts one function in the
Experimental tier. Both are allowed in packages; the annotation syntax is Experimental in 1.0. Neither
changes a type, an interface digest or `interface_hash`.

`ailang iface` now emits `"tier"` (`stable|experimental|internal`) on every std function and type, and
`"replaced_by"`/`"deprecation_note"` on deprecated ones; `iface --compact` and `ailang docs` mark both.
Module tiers live in `std/STABILITY.toml`; `docs/docs/reference/stability.md` is checked against it by a
test. Compile-cache key v10 → v12 (v11 belongs to #2228): the cache's `iface.json` codec dropped the new fields, so a warm
`ailang iface std/list` printed no `tier` (found while freezing; fixed and tested).

`ailang check --package` now reports the pipeline's coded warnings too (`TC_DEPRECATED_001`,
`TC_MATCH_005`, ...) — on stderr and as a `warnings` array in `--json`, once each even when several
files import the module — where it used to drop them all; `errors` and the exit code are unchanged.

The warning also covers uses that `check` never elaborates — named `test` blocks, `property`
blocks, inline `tests` rows and `properties [forall …]` — resolved from the parsed source
(imports and binders in scope), and `ailang test` prints the file's warnings on stderr.
`@deprecated("x")` must name an export of the same module or an imported `"mod/path.name"`:
otherwise `TC_DEPRECATED_002` at the annotation, with a did-you-mean. That is the whole rule —
a replacement that is itself deprecated is legal (it is not in the module's cache key, so an
error would depend on the cache); the use-site warning names the first live function along the
chain instead. A std function deprecated in a later release therefore gives an unchanged library warnings only, never an error, and those warnings are not reported inside an installed dependency. `@experimental(...)` is
`PAR_EXPERIMENTAL_ARG` ("takes no arguments"). `make check-llms-txt` (CI and `ci-quick`) now fails
when the committed `llms.txt` copies do not embed the active teaching prompt; they had sat at
v0.16.12 and are regenerated here.

Also: `L.nthOrx` through a module alias is now `IMP010: symbol 'nthOrx' not exported by 'std/list'`
with the did-you-mean, instead of `undefined variable: L`; IMP010's suggestion ranks a name equal up to
case and underscores first (`nth_or` → `nthOr`, not `nth`).

### Changed — 25 functions in Stable modules are now Experimental (2026-11-11)

`@experimental`: `std/json` `repair`, `asBoolLoose`, `getStringArrayStrict`, `getNumberArrayStrict`,
`decodeFloatArray`; `std/string` `decodeQuotedPrintable`, `foldSlices`, `mapSlicesJoin`,
`startsWithIgnoreCase`; `std/list` `takeMap`, `takeFlatMap`; `std/array` `dot`, `axpy`, `scale`, `add`,
`sub`, `mul`, `sum`, `argmax`, `scatterAdd`, `accumulate`, `encodeF64LE`, `decodeF64LE`, `encodeF32LE`,
`decodeF32LE`. They keep working unchanged; they may change in a 1.x minor with a changelog entry.
All 49 stdlib goldens were re-frozen once (every entry gains `"tier"`; std/array, bytes, fs, io, json,
list, math, rand and string also gain the new names and marks). `stability.md` now lists the
Experimental functions, the deprecated names, the naming and argument-order conventions and the `Json`
number contract.
