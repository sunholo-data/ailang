# M-STD-BASE64URL-ENCODE: close the base64url encode/decode asymmetry in std/bytes

**Status**: Implemented on `dev`, unreleased (ships in v0.35.4) — see Delivery Notes
**Target**: v0.35.4 (small additive stdlib change)
**Priority**: P1 — Medium-High. Blocks a named downstream deliverable (`sunholo/gmail`), and the
workaround's failure mode is silent-wrong-output rather than a compile error.
**Estimated**: ~2.5–3.5 hours
**Dependencies**: none — `std/bytes`, the builtin registration path and the `bytes` runtime value
all already exist. No parser, type-checker, codegen or VM change.

**Commissioning context**: reported from the field as
[`fb_dfb699d91224be9c`](#references) while building a `sunholo/gmail` package. The Gmail API's
`users.messages.send` / `users.drafts.create` endpoints take the RFC 5322 message in a `raw` field
that must be **base64url** encoded. `std/bytes` can decode base64url and cannot encode it.

---

## Problem Statement

`std/bytes` exports three of the four members of the base64 encode/decode matrix:

| | encode (bytes → string) | decode (string → bytes) |
|---|---|---|
| **standard** (RFC 4648 §4) | `toBase64` ✅ | `fromBase64` ✅ |
| **URL-safe** (RFC 4648 §5) | **missing** ❌ | `fromBase64URL` ✅ |

There is no other route to the encoding: `std/crypto` exposes only hex digests, and no hex
byte-codec exists either (verified — see V7). The only way to produce base64url from AILANG today
is to post-process `toBase64` by hand.

**Current State — the workaround, and why it is not fine:**

```ailang
module c
import std/bytes (fromString, toBase64)
import std/string (replaceMany)
export func main() -> string ! {} {
  replaceMany(toBase64(fromString("a+b/c?")), [("+","-"),("/","_")])
}
```

```
$ ailang run c.ail
YStiL2M_          -- correct here, and that is the trap
```

Three problems with leaving it there:

1. **It is silently wrong when it is wrong.** `toBase64(fromString("a+b/c?"))` is `YStiL2M/` —
   a `/` that Gmail rejects. A hand-rolled encoder that forgets one substitution, or forgets to
   strip `=` padding, still produces a value that *looks* like valid base64. Nothing in the type
   system, the compiler, or `ailang verify` can see the difference. The error surfaces as a 400
   from a third-party API at runtime, in an effectful shell, i.e. in exactly the layer AILANG's
   contract story cannot reach.
2. **Every consumer re-implements it.** The encoding is required by the Gmail API, by JWT
   segments, and by several other Google APIs. `std/jwt` already ships in the stdlib and already
   consumes `fromBase64URL`, so the encoding is unambiguously in language scope.
3. **The asymmetry itself is a discoverability defect.** An agent reading `ailang docs std/bytes`
   sees `fromBase64URL` and reasonably infers a `toBase64URL`. The reporter's first action was to
   import it; it produced `IMP010: symbol 'toBase64URL' not exported by 'std/bytes'`.

**Impact:**

- **Blocked**: the `sunholo/gmail` package. Its pure message-composition core (header-injection
  guards, contract-bearing, Z3-verifiable) is precisely the shape AILANG is good at; the final
  encode step is the one part with no stdlib primitive. The reporter explicitly chose to wait for
  the primitive rather than carry a hand-rolled encoder in a published package.
- **Latent**: any future `std/jwt` signing support, which needs base64url encode for the header
  and payload segments.
- **Severity**: low blast radius, high annoyance-per-occurrence, and a failure mode that is
  invisible until an external service rejects the payload.

## Goals

**Primary Goal:** make the base64 matrix complete and symmetric by adding
`toBase64URL(b: bytes) -> string` — RFC 4648 §5, URL-safe alphabet, no padding — as the exact
inverse of the existing `fromBase64URL`.

**Success Metrics:**

- `toBase64URL(fromString("a+b/c?"))` returns `YStiL2M_` (no `+`, no `/`, no `=`).
- Round-trip property holds over all 256 byte values and all three padding residues:
  `fromBase64URL(toBase64URL(b)) == Some(b)`.
- The reporter's `sunholo/gmail` package compiles and drafts a message with no hand-rolled encoder
  and no `std/string.replaceMany` call.
- `ailang docs std/bytes` shows a complete 2×2 base64 matrix.

## Systemic Audit (done before writing this doc)

Per the "is this part of a larger pattern?" gate, the whole `std/bytes` surface was audited for
encode/decode asymmetries rather than patching only the reported symptom:

| Pair | Status |
|---|---|
| `fromString` / `toString` | symmetric ✅ |
| `fromInts` / `toInts` | symmetric ✅ (`toInts` was itself an asymmetry fix — [M-BYTES-TOINTS-BYTEAT](../../implemented/v0_21_0/m-bytes-toints-byteAt.md)) |
| `toBase64` / `fromBase64` | symmetric ✅ |
| `fromBase64URL` / `toBase64URL` | **asymmetric** ❌ ← this doc |
| hex | **neither direction exists** — a missing *pair*, not an asymmetry. Different class, no demonstrated consumer. Out of scope (see Non-Goals). |

`base64url` is the only asymmetry on the surface. The unified fix is therefore: add the one
missing member **and** add a test that asserts the symmetry invariant, so the next codec added to
`std/bytes` cannot land half-finished the way `toInts` did for 2.5 months.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Unpadded (`RawURLEncoding`) only, no padded variant | Determines whether `fromBase64URL(toBase64URL(b))` round-trips. `fromBase64URL` is `base64.RawURLEncoding` and **rejects** padded input (V6), so a padded encoder would produce output the sibling decoder cannot read | agent (precedent-bound) | design | low |
| Name is `toBase64URL`, matching `fromBase64URL`'s casing exactly | The whole point is symmetry; `toBase64Url` or `toBase64URLEncode` re-creates the discoverability defect | agent | design | low |
| Return `string`, not `Result`/`Option` | Encoding is a total function — no input byte sequence can fail. An `Option` return would force pointless unwrapping at every call site and weaken the contract story | agent | design | low |
| Do NOT add the builtin to `compiler.BuiltinTable` / `vm.BuiltinTable` | Those two tables are index-matched; a bytes builtin cannot be added to one alone, and no `_bytes_*` builtin is in either today (V3). Adding it would be a VM-dispatch corruption, not an optimization | agent (compiler-constrained) | design | high if got wrong |
| New teaching-prompt version `v0.16.7` rather than editing `v0.16.6` in place | Pinned eval baselines must keep resolving `v0.16.6` byte-identically | agent (precedent-bound) | compile | low |

### Design Freeze

No decision above requires human ratification: each is either forced by an existing behavior
(verified below) or bound by in-repo precedent. **No design-freeze items** — sprint-executor may
run start-to-finish without pausing.

### Quorum trigger check (attended session)

| # | Trigger | Fires? |
|---|---------|--------|
| 1 | Design-freeze items present | No — every decision is agent-resolvable, see above |
| 2 | Overrides shared machinery (any Conflict Surface row deciding "override") | No — every row is "reuse" |
| 3 | Touches cost/KPI semantics or banked-data schema | No |
| 4 | Load-bearing premises about external systems we do not control | No — the Gmail contract is *motivation*, not a premise. The design is justified entirely by in-repo symmetry with `fromBase64URL` and by `std/jwt`, both of which are re-checkable at any time. If Gmail changed its `raw` encoding tomorrow, this change would still be correct |

**Decision: skip the design quorum.** All four triggers false.

## Verification Log

Every load-bearing claim in this doc, with the command that proved it. All run at
`v0.35.1-16-g5082ca3ed-dirty`, darwin/arm64, 2026-09-08.

| # | Claim | Evidence | Result |
|---|---|---|---|
| V1 | `toBase64URL` does not exist | `ailang docs std/bytes \| grep -i base64` → three entries, no `toBase64URL`; reporter's import produced `IMP010 … did you mean 'toBase64'` | **Confirmed** (negative existence) |
| V2 | `_bytes_to_base64url` is registered in exactly one place | `grep -rn "_bytes_to_base64\|_bytes_from_base64url" --include="*.go" .` → only `internal/builtins/bytes.go` plus doc cross-references in `gzip.go`/`net.go`/`fs.go`/`zip.go` | **Confirmed** — single registration site, no second wiring path |
| V3 | No `_bytes_*` builtin is in the bytecode compiler's `BuiltinTable` | `grep -n "_bytes_" internal/bytecode/compiler/builtins.go` → empty (92 entries, none of them bytes). Corroborated by commit `3ecccb717`: *"Map/bytes builtins deferred — no TagMap/TagBytes in VM"* | **Confirmed** (negative existence) — the VM path needs no change, and must not get one |
| V4 | `std/string.replaceMany` exists and the documented workaround produces the right bytes | ran the workaround program → `YStiL2M_` | **Confirmed** |
| V5 | Plain `toBase64` produces base64url-invalid output on realistic input | `ailang run` on `toBase64(fromString("a+b/c?"))` → `YStiL2M/` | **Confirmed** — the `/` is the reported failure mode |
| V6 | `fromBase64URL` is `RawURLEncoding` (**rejects** padding) and `fromBase64` is `StdEncoding` (**requires** padding) | ran a 6-case matrix: `fromBase64URL("SGVsbG8=")`→`None`, `fromBase64URL("SGVsbG8")`→`Some`, `fromBase64("aGVsbG8")`→`None`, `fromBase64("aGVsbG8=")`→`Some`, `fromBase64URL("YStiL2M_")`→`Some`, `fromBase64("YStiL2M_")`→`None`. Source: `base64.RawURLEncoding.DecodeString` at `internal/builtins/bytes.go:307` | **Confirmed** — pins the encoder to `RawURLEncoding` for round-tripping |
| V7 | No hex byte-codec exists anywhere in the stdlib | `grep -rn "hexEncode\|toHex\|fromHex\|_bytes_to_hex" --include="*.go" --include="*.ail" internal/builtins/ std/` → empty. `std/crypto` exports only `sha256Hex`/`sha256Bytes`/`hmacSha256` (hex *digests*, not a codec) | **Confirmed** (negative existence) — justifies scoping hex out as a different class |
| V8 | `std/jwt` is verify-only, so no in-repo caller regresses | `grep -n export std/jwt.ail` → `decodeJWT`, `verifyRS256`, `verifyWithKid`, `isExpired`, `checkIssuer`, `checkAudience`. `std/crypto` has `rsaVerifyPKCS1v15` and no sign | **Confirmed** — JWT *signing* is separately blocked on RSA-sign; out of scope |
| V9 | Adding a `std/bytes` export breaks a CI gate unless the golden is regenerated | `.stdlib-golden/bytes.json` + `.sha256` exist; `.github/workflows/ci.yml:154` runs `make verify-stdlib`; regenerated by `tools/freeze-stdlib.sh` (module list derived from `find std -maxdepth 1 -name '*.ail'`, so `bytes` is in scope) | **Confirmed** — golden regen is a required task, not optional |
| V10 | The active teaching prompt is `v0.16.6` and is not frozen | `ailang prompt --list` → `* = active version (v0.16.6)`; `versions.json` has no `frozen` block on `v0.16.6` (`v0.16.5` does) | **Confirmed** — but precedent (V11) still says add a version |
| V11 | Precedent for a new prompt version on a `std/bytes` discoverability change | `versions.json` `v0.16.5` notes: created specifically to add `fromInts`/`toInts`/`byteAt` to the `import std/bytes (...)` line, *"v0.16.4 remains served byte-identical for pinned eval baselines"* | **Confirmed** — identical situation, follow it |
| V12 | `ailang docs std/bytes` is generated from the embedded stdlib, not a checked-in doc file | `ailang docs std/bytes` renders the doc-comments verbatim from `std/bytes.ail`, and warns `⚠ Binary may be stale` against the working tree | **Confirmed** — no separate docs file to update; a rebuild suffices |
| V13 | `internal/builtins/bytes.go` stays inside the acceptable file-size band | 639 LOC now; +~55 → ~694, inside the 500–800 "acceptable" band from `.claude/rules/coding-standards.md` | **Confirmed** — no split needed |
| V14 | Example 2's AILANG syntax parses and runs (minus the not-yet-existing `toBase64URL` line) | ran the control form with `toBase64` + `fromBase64URL` + `match` + `${}` interpolation → `std=YStiL2M/ back=a+b/c?`. Note: the first draft used `"a+b/c?" |> fromString` inside a call argument and did **not** parse — the example uses plain `let` bindings instead | **Confirmed** — the example is copy-runnable once the function exists |

## Conflict Surface

The Conflict Surface section is only *mandatory* for parser / lexer / ast / types / elaborate /
iface / codegen / eval / vm / effects changes, and this change touches none of them — it adds one
entry to the builtin registry and one export to a `.ail` file. It is written anyway, per the
2026-08-26 lesson that the section earns its keep even when the rule exempts you. It did: row 2
below is the one way this small change could do real damage.

**1. What position does this extend?** The `std/bytes` export list and the `internal/builtins`
registry namespace. No syntactic position; no new tokens, no new type constructors.

**2. What else lives in those positions?**

| Occupant | Interaction | Decision |
|---|---|---|
| The other 13 `std/bytes` exports | Purely additive; `toBase64URL` collides with no existing name (V1) | **reuse** the existing registration path unchanged |
| `compiler.BuiltinTable` ↔ `vm.BuiltinTable`, index-matched and validated at startup (`internal/vm/builtins.go:41`) | If a `_bytes_to_base64url` entry were added to one table and not the other, dispatch shifts by one index and *every subsequent builtin call in the VM silently invokes the wrong function*. No `_bytes_*` builtin is in either table (V3) | **reuse**: add to neither. The builtin resolves through the eval bridge like the other 13 bytes builtins |
| `.stdlib-golden/bytes.json` freeze gate | A new export changes the module interface digest; `make verify-stdlib` fails in CI until `tools/freeze-stdlib.sh` is re-run (V9) | **reuse** the existing freeze tooling — regeneration is a task, not a gate override |
| The `Option`-returning bytes builtins (`fromBase64`, `fromBase64URL`, `slice`, `byteAt`) | `toBase64URL` returns a bare `string`, matching `toBase64`, not them. Encoding is total | **reuse** `toBase64`'s shape |
| Teaching prompt `v0.16.6`, pinned by eval baselines | Editing in place would change the served bytes for pinned runs | **reuse** the versioning mechanism: new `v0.16.7` (V11) |

**3. How is it disambiguated?** No disambiguation needed — this is name-level addition into a
namespace where the name is provably free (V1).

**4. Programs that MUST still work post-change** (all exist today, all use `std/bytes`):

- `examples/runnable/http_put_bytes.ail`
- `examples/runnable/stdlib_gzip.ail`
- `examples/runnable/zip_build_in_memory.ail`
- `examples/runnable/std_deflate_pdf_objstm.ail`
- `std/jwt.ail`'s `decodeJWT` path, which imports `fromBase64URL` (V8)

**5. What deliberately changes?** The interface digest of `std/bytes`, and nothing else. No
existing behavior changes; no existing program's meaning changes.

## Solution Design

### Overview

One Go builtin, one stdlib export, mirroring `fromBase64URL` line for line with the direction
reversed. `base64.RawURLEncoding.EncodeToString` is the exact inverse of the
`base64.RawURLEncoding.DecodeString` already at `internal/builtins/bytes.go:307`, so the
round-trip property is guaranteed by the Go standard library rather than by our own alphabet
substitution.

### Architecture

**Components:**

1. **`_bytes_to_base64url` builtin** — `internal/builtins/bytes.go`, registered via
   `RegisterEffectBuiltin` with `IsPure: true, Effect: ""`, type `bytes -> string`, implementation
   `base64.RawURLEncoding.EncodeToString(bytesVal.Value)`. Placed immediately after
   `registerBytesFromBase64URL` so the pair reads together. `Since: "v0.35.4"`,
   `SeeAlso: ["_bytes_from_base64url", "_bytes_to_base64"]`, and — importantly — the existing
   `registerBytesFromBase64URL` metadata's `SeeAlso` gains `_bytes_to_base64url` so the pointer
   works in both directions.
2. **`toBase64URL` export** — `std/bytes.ail`, a `pure func` wrapper with a doc comment that
   states the alphabet, the absence of padding, and the round-trip guarantee against
   `fromBase64URL`.
3. **Symmetry regression test** — a table test asserting `fromBase64URL(toBase64URL(b)) == Some(b)`
   over all 256 single-byte values and over inputs of length 1, 2 and 3 mod 3 (all three padding
   residues), plus a cross-alphabet negative: `toBase64URL` output must never contain `+`, `/` or
   `=`.

### Implementation Plan

**Phase 1: builtin + export** (~1.5 hours)
- [ ] Add `registerBytesToBase64URL`, `makeBytesToBase64URLType`, `bytesToBase64URLImpl` to `internal/builtins/bytes.go`, and the `registerBytesToBase64URL()` call in `init()`
- [ ] Add `_bytes_to_base64url` to `registerBytesFromBase64URL`'s `SeeAlso` (both directions)
- [ ] Add the `toBase64URL` export + doc comment to `std/bytes.ail`
- [ ] Add unit tests to `internal/builtins/bytes_test.go` (round-trip, alphabet, padding, empty input, Gmail-shaped RFC 5322 blob)
- [ ] `make test` green

**Phase 2: gates, example, changelog** (~0.75 hours)
- [ ] `make quick-install`, then `tools/freeze-stdlib.sh` to regenerate `.stdlib-golden/bytes.{json,sha256}`
- [ ] `make verify-stdlib` green (V9)
- [ ] Add `examples/runnable/bytes_base64url.ail` — encode/decode round-trip plus the `a+b/c?` case that shows the standard-vs-URL alphabet difference
- [ ] `make verify-examples` green
- [ ] Changelog entry in `changelogs/v0.32-current.md`

**Phase 3: teach it** (~0.75 hours)
- [ ] Create `cmd/ailang/prompts/v0.16.7.md` from `v0.16.6.md`: add `toBase64URL` to the canonical `import std/bytes (...)` line (line ~635), add the `toBase64URL(b) -> string` entry to the Bytes functions list (line ~726), and extend the §5 note at line ~1983 to name both directions
- [ ] Register `v0.16.7` in `cmd/ailang/prompts/versions.json` with hash, `tags: [production, latest, stdlib-discoverability]`, and a note recording that `v0.16.6` remains served byte-identically for pinned eval baselines
- [ ] Reply to `fb_dfb699d91224be9c` via the `ailang-feedback` skill and ack it

### Files to Modify/Create

**New files:**
- `examples/runnable/bytes_base64url.ail` — round-trip + alphabet-difference example, ~25 LOC
- `cmd/ailang/prompts/v0.16.7.md` — copy of v0.16.6 plus three edits, ~3 changed lines

**Modified files:**
- `internal/builtins/bytes.go` — register/type/impl for `_bytes_to_base64url`, plus a `SeeAlso` line on the existing decoder, ~+56 LOC (639 → ~695, V13)
- `std/bytes.ail` — one `pure func` export + doc comment, ~+9 LOC
- `internal/builtins/bytes_test.go` — round-trip, alphabet and padding tests, ~+65 LOC
- `.stdlib-golden/bytes.json` — regenerated, not hand-edited
- `.stdlib-golden/bytes.sha256` — regenerated, not hand-edited
- `cmd/ailang/prompts/versions.json` — one new version entry
- `changelogs/v0.32-current.md` — one entry

## Examples

### Example 1: the reported use case — a Gmail draft

**Before** (hand-rolled, and wrong if you forget the `=` strip):

```ailang
module gmail/encode
import std/bytes (fromString, toBase64)
import std/string (replaceMany)

-- Nothing here is checkable. If a substitution is dropped, this still
-- returns a plausible-looking base64 string and Gmail 400s at runtime.
export pure func rawField(rfc5322: string) -> string =
  replaceMany(toBase64(fromString(rfc5322)), [("+","-"), ("/","_"), ("=","")])
```

**After:**

```ailang
module gmail/encode
import std/bytes (fromString, toBase64URL)

export pure func rawField(rfc5322: string) -> string =
  toBase64URL(fromString(rfc5322))
```

### Example 2: the alphabet difference, made visible

```ailang
module bytes_base64url
import std/bytes (fromString, toBase64, toBase64URL, fromBase64URL, toString)
import std/option (Option, Some, None)

export func main() -> string ! {} {
  let src  = fromString("a+b/c?");
  let enc  = toBase64(src);           -- "YStiL2M/"  <- '/' breaks URLs
  let url  = toBase64URL(src);        -- "YStiL2M_"  <- URL-safe, unpadded
  let back = match fromBase64URL(url) { Some(b) => toString(b), None => "ROUND-TRIP FAILED" };
  "std=${enc} url=${url} back=${back}"
}
```

### Example 3: the invariant the test asserts

```
for every byte sequence b:
    fromBase64URL(toBase64URL(b)) == Some(b)
    toBase64URL(b) contains none of '+', '/', '='
```

## Success Criteria

- [x] `toBase64URL(b: bytes) -> string` exported from `std/bytes`, and visible in `ailang docs std/bytes`
- [x] `toBase64URL(fromString("a+b/c?")) == "YStiL2M_"` (acceptance test: `ailang run examples/runnable/bytes_base64url.ail`)
- [x] Round-trip test over all 256 byte values and all three length-mod-3 residues passes (`TestBytesBase64URLRoundTrip`)
- [x] Alphabet test: encoder output contains no `+`, `/` or `=` for 1000 pseudorandom inputs (`TestBytesBase64URLAlphabet`)
- [x] `.stdlib-golden/bytes.{json,sha256}` regenerated; `make verify-stdlib` green
- [x] `make test` and `make verify-examples` green
- [x] `examples/runnable/bytes_base64url.ail` added and runs clean
- [x] Teaching prompt `v0.16.7` registered; `v0.16.6` still served byte-identically
- [x] Changelog entry in `changelogs/v0.32-current.md`
- [ ] `fb_dfb699d91224be9c` answered and acked

## Delivery Notes

Implemented 2026-09-08 in one session, as estimated. Four things the doc did not anticipate,
all mechanical, all found by a gate rather than by review:

1. **`tools/freeze-stdlib.sh` resolves `./bin/ailang` before `$PATH`.** `make quick-install`
   alone is not enough — the script regenerated `bytes.json` with a stale v0.34 binary and
   *emptied* the golden rather than failing loudly about the unknown builtin. `make build`
   first, then freeze. (V9 named the script but not which binary it picks.)
2. **`examples/manifest.json` carries a `statistics` block** that `verify-examples` validates
   against the entry count. Adding an example means updating `total`, `working` and `coverage`,
   not just appending the entry. Undocumented in the doc's Phase 2.
3. **`internal/pipeline/testdata/builtin_types.golden` is a second golden** covering every
   builtin signature — a new builtin fails `TestBuiltinTypes_GoldenSnapshot` until
   `UPDATE_GOLDEN=1 go test ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot` is run.
   The doc only knew about the stdlib-interface golden.
4. **`cmd/ailang/prompts/` is a mirror, not the source.** The source of truth is the repo-root
   `prompts/`; a build step re-syncs the mirror and silently deleted a new prompt file written
   only into `cmd/`. Edits go in `prompts/`, then get copied across; `ailang prompt freeze
   --check` validates the mirror (60 entries, clean).

Verified after the change: `ailang prompt` (active v0.16.7) mentions `toBase64URL` 4 times and
`ailang prompt --version v0.16.6` mentions it 0 times — the pinned-baseline guarantee holds.
The reporter's exact repro returns `YStiL2M_`.

Merged 194 upstream commits mid-sprint; v0.35.2 and v0.35.3 shipped while this was in flight,
so the target moved from v0.35.2 to v0.35.4 and the changelog entry moved to a fresh
`[Unreleased]`.


## Testing Strategy

**Unit tests** (`internal/builtins/bytes_test.go`):
- Round-trip against `_bytes_from_base64url` over all 256 single-byte values
- Round-trip over lengths 0,1,2,3,4,5 — covers all three padding residues, which is where a
  hand-rolled encoder characteristically breaks
- Alphabet: output never contains `+`, `/` or `=`
- Empty input → empty string (and back)
- The exact reported case: `"a+b/c?"` → `"YStiL2M_"`, and its standard-base64 sibling `"YStiL2M/"`,
  asserted side by side so the difference is legible in the test source
- A realistic RFC 5322 blob with CRLF line endings and a UTF-8 subject

**Integration tests:**
- `examples/runnable/bytes_base64url.ail` under `make verify-examples`
- `make verify-stdlib` — proves the golden was regenerated rather than the gate bypassed

**Manual testing:**
- Rebuild and confirm `ailang docs std/bytes` shows a complete 2×2 base64 matrix
- Rebuild the reporter's repro: their original `b64url.ail` should run and print `YStiL2M_`

## Deferred Decisions

- **Which file the builtin lives in** — `internal/builtins/bytes.go` next to its sibling is the
  default; if the implementer would rather start `internal/builtins/bytes_base64.go` and move both
  URL functions there, that is fine. *Agent may choose.*
- **Exact wording of the `std/bytes.ail` doc comment**, provided it states the alphabet, the
  absence of padding, and the `fromBase64URL` round-trip. *Agent may choose.*
- **Whether the round-trip test is table-driven or a Go fuzz target.** *Agent may choose.*

## Non-Goals

- **A padded base64url variant** (`toBase64URLPadded`). `fromBase64URL` rejects padded input (V6),
  so a padded encoder would emit strings its own sibling decoder returns `None` for. If a consumer
  ever needs padded base64url, it needs a padded *decoder* too, and that is its own doc.
- **Hex codec** (`toHex` / `fromHex`). No hex byte-codec exists in either direction (V7) — that is
  a missing pair, not an asymmetry, and no consumer has asked. Bundling it would delay the
  unblock this doc exists to deliver.
- **JWT signing** (`signRS256`). Blocked on RSA *signing* in `std/crypto`, which has only
  `rsaVerifyPKCS1v15` (V8). `toBase64URL` is a prerequisite for it, not a delivery of it.
- **Wiring bytes builtins into the bytecode VM.** Deliberately excluded — see Conflict Surface
  row 2. A `TagBytes` VM representation is a separate piece of work with its own parity run.
- **Building `sunholo/gmail`.** That is the downstream package, in the packages repo, not here.

## Timeline

Single sitting, ~3 hours:

- Phase 1 (builtin, export, unit tests): ~1.5h
- Phase 2 (golden regen, example, changelog, gates green): ~0.75h
- Phase 3 (prompt v0.16.7, feedback reply): ~0.75h

**Total: ~3 hours, one session.** No multi-week timeline — this is a P1 unblock, not a project.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Golden not regenerated → CI red on an unrelated later PR | Med | It is an explicit Phase 2 task with `make verify-stdlib` as its acceptance gate (V9), not an afterthought |
| Implementer "optimizes" by adding the builtin to `compiler.BuiltinTable` | High — silent VM mis-dispatch of every later builtin | Called out as a Conflict Surface row and a High-Impact Decision. The two tables are index-matched and validated at startup, so a one-sided addition fails loudly at boot; a two-sided one without a `TagBytes` representation does not |
| Prompt edited in place, breaking pinned eval baselines | Med | New `v0.16.7` version, per the `v0.16.5` precedent (V11), with the byte-identical note in `versions.json` |
| Padding choice wrong (padded encoder + raw decoder) | Med — silent round-trip failure | Pinned to `RawURLEncoding` by V6, and the round-trip test over all three residues would catch a regression |
| Scope creep into hex / JWT signing | Low | Both explicitly in Non-Goals with the verification rows that justify the split |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure, total, deterministic — `RawURLEncoding` has no configuration and no ambient state |
| A2: Replayability | 0 | No trace surface; a pure builtin call |
| A3: Effect Legibility | 0 | Declared `pure`, `Effect: ""`; introduces no effect |
| A4: Explicit Authority | 0 | No ambient access of any kind |
| A5: Bounded Verification | +1 | Total function returning `string` — no `Option`, no panic path, so a contract over a base64url-producing function stays inside Z3's reach |
| A6: Safe Concurrency | 0 | No shared state |
| A7: Machines First | +2 | Directly removes a machine-legibility defect: today the correct encoding is a `replaceMany` incantation an agent must know to write, and getting it wrong type-checks, runs, exits 0, and fails at a third-party API. Naming it makes the intent recoverable from the source |
| A8: Minimal Syntax | 0 | Pure stdlib addition; no syntax change |
| A9: Cost Visibility | 0 | Same O(n) cost as `toBase64`, and cheaper than the `toBase64`+`replaceMany` workaround it replaces |
| A10: Composability | +1 | Completes the codec matrix; `std/jwt`, `sunholo/gmail` and any future signer compose on it instead of each carrying a private encoder |
| A11: Structured Failure | 0 | Encoding is total — there is no failure to structure |
| A12: System Boundary | +1 | base64url is a *system-boundary* encoding by definition. Making it a named primitive puts the boundary crossing in the type signature rather than in an anonymous string substitution |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — pure, total function
- [x] A3 (Effects): no hidden side effects — registered `IsPure: true`, `Effect: ""`
- [x] A4 (Authority): no ambient access granted — operates only on its argument
- [x] A7 (Machines First): scores +2; this is a machine-legibility fix, not human convenience

## Related Documents

**Implemented (may inform design):**
- [M-BYTES-TOINTS-BYTEAT](../../implemented/v0_21_0/m-bytes-toints-byteAt.md) — the closest
  precedent: the same class of `std/bytes` asymmetry (`fromInts` with no `toInts`), the same fix
  shape, and a Delivery History section documenting how it sat half-done for 2.5 months. Its
  lesson is why this doc adds a symmetry regression test rather than only the missing function.
- [M-STDLIB-CRYPTO-JWT](../../implemented/v0_10_0/m-stdlib-crypto-jwt.md) — added `fromBase64URL`
  and `std/jwt`; the source of the asymmetry this doc closes.
- [M-BYTECODE-STDLIB-BUILTINS](../../implemented/v0_11_0/m-bytecode-stdlib-builtins-sprint-plan.md) —
  the VM `BuiltinTable` wiring, and the record that bytes builtins were deliberately deferred (V3).
- [design_docs/implemented/v0_19_0/m-net-binary-bodies.md](../../implemented/v0_19_0/m-net-binary-bodies.md) (0.36, neural)

**Planned (checked for overlap — none found):**
- [design_docs/planned/v0_36_0/m-cache-module-id-encoding.md](../v0_36_0/m-cache-module-id-encoding.md)
  (0.36, neural) — "encoding" in the module-cache-key sense, unrelated to byte codecs.
- [design_docs/planned/v0_29_0/m-stdlib-html-streaming.md](../v0_29_0/m-stdlib-html-streaming.md) (0.32, neural)
- [design_docs/planned/m-prompt-version-freeze-on-first-bank.md](../m-prompt-version-freeze-on-first-bank.md)
  — governs Phase 3. Not yet implemented, but its rule is what the new-version-not-in-place-edit
  decision anticipates.

Highest neural similarity across all docs was **0.36**, well under the 0.45 warn threshold. No
duplicate or prior coverage.

## References

- Field report `fb_dfb699d91224be9c` — *"std/bytes: toBase64URL missing — decode exists, encode
  does not"*, from `mcp-public`, 2026-09-08, contact mark@aitanalabs.com. Read with
  `ailang messages list --unread --json` against the canonical prod store.
- [RFC 4648 §5](https://datatracker.ietf.org/doc/html/rfc4648#section-5) — base64url alphabet
- [Gmail API `users.messages.send`](https://developers.google.com/gmail/api/reference/rest/v1/users.messages/send) — the `raw` field's encoding requirement
- `internal/builtins/bytes.go:259-316` — `registerBytesFromBase64URL`, the function this mirrors
- [Design Axioms](/docs/references/axioms)

## Future Work

- **`std/crypto.rsaSignPKCS1v15`** → unblocks `std/jwt.signRS256`, which needs `toBase64URL` for
  the header and payload segments. `toBase64URL` is the prerequisite this doc delivers.
- **`toHex` / `fromHex` in `std/bytes`** — the other codec gap the audit found (V7). Worth doing
  when a consumer appears; the symmetry test added here gives it a template.
- **`sunholo/gmail` package** — the downstream deliverable, in the packages repo.
- **A `TagBytes` VM representation**, which would let all 14 `std/bytes` builtins join the bytecode
  `BuiltinTable` and drop out of the EvalOnly count. Separate work, separate parity run.

---

**Document created**: 2026-09-08
**Last updated**: 2026-09-08
