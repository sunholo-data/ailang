# M-ARRAY-SHOW-DIVERGES-RUN-VS-COMPILE: `show` on arrays prints `#[…]` interpreted, `[…]` compiled

**Status**: PLANNED — iteration 248 of the V1 mission; last surviving member of the show-divergence class iteration 247 closed (records, ADT constructors, tuples, depth/width caps, floats — PR #822)
**Target**: v0.34.x
**Priority**: P2 — backend-dependent stdout in a language whose first stated principle is determinism, and noise in the run-vs-compile differential harness that two other queue rows depend on. NOT a soundness bug (see Framing).
**Estimated**: 3–4 days (one sprint, four milestones)
**Dependencies**: none. Self-contained to `internal/gen/golang/` plus two golden fixtures.
**Author**: Claude Fable 5 (design-doc-creator, mission iteration 248, 2026-08-22)

All commands in this doc were run at `origin/dev` = `404226a48c64be62a767c06330612f49b3038e98`
with a binary freshly built from that tree (`go build -o /tmp/ailang-i248-doc/ailang ./cmd/ailang`,
rc=0) prepended to PATH — never the shared `~/go/bin` binary.

---

## Framing — what this defect is and is not

The same well-typed program prints different bytes depending on the backend:

```ailang
module bench/v1
import std/array (fromList)
export func main() -> () ! {IO} {
  println(show(fromList([1, 2, 3])));
  println(show([1, 2, 3]))
}
```

| Backend | Line 1 (array) | Line 2 (list) |
|---|---|---|
| `ailang run` (interpreter) | `#[1, 2, 3]` | `[1, 2, 3]` |
| `ailang compile --emit-go` → binary | `[1, 2, 3]` | `[1, 2, 3]` |

**This is an output-consistency defect, not a soundness bug.** The type checker is the guard and
it holds: applying `std/array.get` to a list literal fails `ailang check` with
`cannot unify array type Array[α4] with [α5]` (VL-4). No well-typed source can pass a list where
an array is expected or vice versa. The erasure is confined to the Go *representation* after type
checking. The cost is (a) backend-dependent stdout, and (b) a hole in
`TestInterpreterCompiledDifferential` (iteration 247's harness): no array fixture can exist today
because it would fail.

**The erasure happens at construction, not at `Show`.** `show(#[1,2,3])` and `show([1,2,3])`
compile to byte-identical Go: `var tmpN interface{} = []interface{}{int64(1), int64(2), int64(3)}`
followed by `Show(tmpN)` (VL-2). `Show` is a single dynamic runtime function
`func Show(v interface{}) string` reached by type-switch + reflection — there is no dictionary
dispatch and no `Show` type class anywhere in `std/` (`show` is a builtin, VL-3). By the time any
value reaches `Show`, both types are `[]interface{}`. No change to `Show` alone can fix this.

## The false blocker — the "M-TYPE1 decision" does not exist

`internal/gen/golang/codegen_ops.go:357` carries the comment
*"M-TYPE1: Arrays use the same Go representation as lists (slices)"*. The backlog row inferred
from it that representation-sharing was a ratified design decision requiring a design pass to
overturn. **That inference is refuted by the history** (VL-5):

- The M-TYPE1 commit (`743f6a539`, named in
  `design_docs/implemented/v0_5_6/m-type1-array-tarray-unification.md`) touched **one file,
  `internal/types/unification.go`, 8 insertions** — a type-checker fix making `TApp(Array, T)`
  unify with `TArray{Element: T}`. It never touched `internal/gen/golang/` at all.
- That design doc mentions `[]interface` **0 times** (control: `Array` appears 63 times).
- The codegen comment entered via `ea88158ef` ("working array compiles to go") and `474adf0cf`
  ("code gen fixes") — bulk codegen bring-up commits, not M-TYPE1.

The shared Go representation is an **implementation convenience labelled with a tag borrowed from
an unrelated type-checker fix**. There is no ratified constraint to respect, and no design
decision to overturn. A later reader hitting that comment should not re-inherit the false blocker;
Milestone 1 rewrites the comment.

A second false premise, corrected while measuring: the backlog describes iteration 247's tuple fix
as a "struct wrapper". It is not. The generated runtime defines `type Tuple []interface{}` — a
**defined slice type** — discriminated by a plain type-switch case in `showValue`
(`case Tuple:` emitted from `codegen_runtime_misc.go:100`) (VL-6). The working precedent in the
codebase is option (B) below, not option (A).

## Options

### (A) Struct wrapper at the codegen representation layer — REJECTED

Wrap arrays in a Go struct. Breaks **all 31** discrimination sites in the generated runtime: the
21 exact `.([]interface{})` assertions *and* the 10 `reflect.Slice` branches (a struct's `Kind()`
is `reflect.Struct`, so every reflection fallback in the 8 array helpers dies too) (VL-7). Its
cited precedent — the tuple fix — turns out to be a defined slice type, not a struct (VL-6), so
the precedent argues for (B). Roughly 2× the blast radius of (B) for zero additional benefit.

### (B) Defined slice type `type ArrayVal []interface{}` — **RECOMMENDED**

Exactly the `Tuple` mechanism. `reflect.TypeOf(v)` differs, `Kind()` stays `reflect.Slice`, so:

- The **10 `reflect.Slice` branches survive unchanged**. This matters more than it first appears:
  every one of the 8 array-family runtime helpers (`FromList`, `ToList`, `Length`, `Get`,
  `GetOpt`, `UnsafeGet`, `Set`, `Make` — all `interface{}` in/out) already has a reflection
  fallback *behind* its fast-path assertion, added for typed slices like `[]int64`
  (`TestGetOptHasReflectionPath`, `codegen_datastructures_test.go:186-189`). An `ArrayVal` that
  misses the fast path lands in the reflection path and computes the right answer today (VL-8).
  The helper work is therefore mostly *return-type* changes (preserve array-ness through
  `fromList`/`set`/`make`), plus optional `ArrayVal` fast paths.
- The **21 exact assertions** decompose by site, not as an undifferentiated count (VL-7):
  - **7** in `codegen_runtime_slices.go` — the `ConvertTo*Slice` converters. These are the one
    real hazard: on `!ok` they **silently `return nil`** (VL-9), so an unhandled `ArrayVal`
    reaching a converter would produce an empty typed slice with no error — exactly the silent
    fallback CLAUDE.md bans. Each needs an explicit `ArrayVal` arm (or shared reflect fallback).
  - **~8** in `codegen_runtime_collections.go` — the array helpers' fast paths (work via
    reflection even if untouched; updated for return types anyway).
  - **~6** in `codegen_runtime_collections.go` — list-only helpers (`Cons`, head/tail/concat
    paths, lines 11–121). The type checker guarantees no array reaches them (VL-4). No change.
- `FromList`/`ToList` are **not** identity today — both already make an O(n) defensive copy
  ("Return a copy to preserve immutability", generated `runtime.go:689/714`) (VL-8). (B) changes
  only the *result type* of `FromList`; cost class unchanged.
- Array equality is **out of the conflict surface**: `#[1,2] == #[1,2]` is a type error at HEAD —
  `No instance for Eq[Array[int]] in scope` (VL-10). Note the conclusion is right and the reason is
  narrower than it looks: **lists have no `Eq` instance either** (`[1,2] == [1,2]` fails identically
  with `No instance for Eq[[int]] in scope`), so this is not an array-specific gap and a later
  reader must not infer that lists are comparable and arrays are not. No generated equality path
  compares either.
- Rendering: one new `case ArrayVal:` in the emitted `showValue`, mirroring `case Tuple:`
  one line above the default (VL-6).

Edit sites, exhaustively: `type ArrayVal` emitted at **both** runtime-preamble sites
(`codegen.go:556` and `codegen.go:664`); construction in `generateArray`
(`codegen_ops.go:357-375`); array mappings in `types.go` and `adt.go`; aggregate-boundary emission
in `codegen_expr_app.go` and both record paths in `codegen_record.go`; `case ArrayVal:` in
`codegen_runtime_misc.go` (~line 100); helper returns + all 7 converter arms in
`codegen_runtime_collections.go` / `codegen_runtime_slices.go`; corresponding unit and golden tests.

### (C) Change the interpreter: render arrays as `[1, 2, 3]` — REJECTED, with the argument made

Near-zero code cost (two sites: `internal/eval/value.go:110` `ArrayValue.String()` and
`internal/builtins/show.go:126`, VL-11). The "round-tripping is a promise `show` makes" objection
is indeed false as a general claim — iteration 247 deliberately shipped `Map{a: 1}` as
non-round-trippable. But the two cases fail differently, and the difference is the decision:

- `Map{a: 1}` does not parse. A reader (human or agent) cannot mistake it for source syntax.
- `[1, 2, 3]` for an array **parses cleanly — as a list**. It doesn't fail to round-trip; it
  round-trips to the *wrong type*. Since `#[…]` is genuine array-literal syntax (confirmed by
  iteration 247 against parser, docs, and `ailang prompt`), (C) would make `show` emit output
  that is actively misleading about the value's type.

AILANG's stated priorities include semantic transparency and structured traces consumed by AI
agents; an agent reading a trace could no longer distinguish array from list output, in a language
whose type checker treats them as non-unifiable (VL-4). That trades a language-visible property to
avoid ~2 days of codegen work whose mechanism is already proven by `Tuple`. If (C) were chosen
anyway, the measured update surface is: `internal/builtins/show_test.go` (pins `#[` rendering),
`docs/docs/reference/arrays.md` (33 `#[` occurrences), the served prompt (`ailang prompt` emits 3
`#[` occurrences in 2547 lines), `examples/runnable/array_adt.ail` + 3 trace JSONLs, and both
interpreter render sites — and it is a behaviour change to every existing interpreted program's
output (VL-12).

## Recommendation

**(B), across every generated array representation.** It fixes the defect where it occurs
(construction and typed aggregate boundaries), reuses the proven `Tuple` mechanism, and leaves the
interpreter — the reference semantics — untouched. `ArrayVal` becomes the single generated Go
identity for AILANG arrays; typed slices remain available for AILANG lists. The implementation must
change both type mappers: `TypeMapper.mapTypeWithVisited(*types.TArray)` in `types.go` and
`ADTGenerator.mapASTType(*ast.ArrayType)` in `adt.go` return `ArrayVal`, while their `TList` /
`ListType` branches remain `[]T`. Consequently ADT constructors and record fields declared
`Array[T]` accept/store `ArrayVal`, and `codegen_expr_app.go` / `codegen_record.go` must not select a
`ConvertTo*Slice` conversion for those fields. The 7 converters still gain explicit `ArrayVal`
input arms because array values can cross existing generated conversion boundaries; none may fall
through to silent `nil`. This closes both the dynamic literal path and the measured aggregate path.

## Conflict Surface (what depends on Array and List sharing a Go representation)

| # | Dependent | Evidence (command) | Impact under (B) |
|---|---|---|---|
| 1 | `ConvertTo*Slice` converters silently `return nil` on non-`[]interface{}` input | `sed -n '5,30p' internal/gen/golang/codegen_runtime_slices.go` → `if !ok { return nil }` (VL-9) | **The hazard, and it is wider than `ArrayVal`.** Unhandled input → silent empty slice, which CLAUDE.md Principle 2 forbids outright. Per the round-2 quorum, M2 **removes** the `if !ok { return nil }` branch and panics with a converter-specific message; **no converter retains a silent `nil` fallback**. Applied to the **5 literal converters** in `codegen_runtime_slices.go`, which are the only silent ones — the template loops at `:192`/`:315` already panic (`M-DX12`), and are the pattern to copy. Not enumerated by name; see M2's corrected scope note, and note `toSlice` shares the defect outside this family. |
| 2 | Array helpers' fast paths assume `[]interface{}`; reflection paths assume `Kind()==Slice` | VL-7, VL-8 | Fast paths miss → reflection path still correct. Return sites (`FromList`, `Set`, `Make`) must return `ArrayVal` or array-ness is lost after one operation. |
| 3 | List-only helpers share the same assertion pattern | VL-7 (sites at `codegen_runtime_collections.go:11-121`) | None — type checker blocks arrays from reaching them (VL-4). |
| 4 | `ToList` must keep returning plain `[]interface{}` | VL-8 | Explicitly pinned by the new fixture (`show(toList(arr))` must print `[…]`). |
| 5 | **Typed aggregate boundary**: ADT/record fields typed `Array[int]` map to `[]int64`; constructor/record generation therefore selects `ConvertToInt64Slice`, erasing array identity before `Show` | `sed -n '79,85p' internal/gen/golang/types.go`; `sed -n '382,398p' internal/gen/golang/adt.go`; ADT repro in VL-16 | **In scope; no residual.** Map `TArray` and `ast.ArrayType` to `ArrayVal` while leaving list mappings typed, then make constructor/record expression generation preserve that type instead of calling `ConvertTo*Slice`. The dedicated ADT fixture must remain red until this path byte-matches. |
| 6 | `showReflect`'s `reflect.Slice` case renders any remaining typed slice as `[…]` | VL-6 (generated `runtime.go`, `showReflect`) | Unchanged — pre-existing behaviour for typed slices of *any* provenance, lists included. |
| 7 | Array equality | VL-10 | Out of scope — `==` on arrays is a type error at HEAD; no generated code compares arrays. |
| 8 | Interpreter-side `ArrayValue` consumers (`internal/embed/convert.go`, observatory helpers, `canonical_key.go`) | `grep -rln 'ArrayValue' internal/ --include='*.go' \| grep -v _test` → 8 files | None — (B) touches only `internal/gen/golang/`; interpreter values and their renderings are unchanged. `canonical_key.go` contains no `#[` (VL-11 control). |
| 9 | Golden compile tests | `tests/golden/codegen/golden_test.go` `TestGoldenCompile` | All existing fixtures re-verify `go build` + `go vet` on generated code containing `ArrayVal` — free regression coverage. |

## Milestones

**M1 — representation + rendering (day 1).**
Emit `type ArrayVal []interface{}` at both preamble sites (`codegen.go:556`, `:664`); switch
`generateArray`'s `[]interface{}` branch to `ArrayVal{`; add `case ArrayVal:` →
`showSequence(…, "#[", "]")` in the emitted `showValue`; rewrite the misattributed M-TYPE1
comment at `codegen_ops.go:357` to name this doc. At end of M1 the new differential fixture
passes for literals.

**M2 — helpers + converters, and the silent-fallback removal (day 2).**
`FromList`/`Set`/`Make` return `ArrayVal`; `ToList` keeps returning plain (pinned by fixture);
optional `ArrayVal` fast paths in the 8 helpers. Extend the fixture to flow through
`fromList`/`get`/`set`/`make`/`toList`.

**Quorum round-2 requirement, applied verbatim (both reviewers, independently, same objection).**
`gpt5-6-sol`: *"Revise M2 to replace all seven silent fallbacks with deterministic explicit failure,
for example: handle `[]interface{}` and `ArrayVal` explicitly, then
`panic("ConvertToInt64Slice: expected list or array slice, got %T", v)` (with the corresponding
converter name/type). Add generated-runtime tests asserting valid list and `ArrayVal` inputs convert
correctly and unsupported inputs panic with stable, converter-specific messages. Update Conflict
Surface #1 and acceptance criterion 4 to state that no converter retains a silent `nil` fallback."*
`gemini-3-1-pro`: *"Update Milestone 2 to explicitly require removing the `if !ok { return nil }`
statement entirely across all 7 `ConvertTo*Slice` converters, replacing it with a generated panic
(e.g., `panic("type conversion failed")`) to universally enforce the 'no silent fallbacks' axiom."*
Adding an `ArrayVal` arm alone routes around the hazard for one type and leaves it live for every
other; CLAUDE.md Principle 2 forbids the fallback itself, not one instance of it. So M2 **removes**
`if !ok { return nil }` and fails loudly instead.

> **CONTROLLER SCOPE CORRECTION — ISSUED, THEN REFUTED BY THE PLANNER. Read the corrected version;
> the superseded one is kept because acting on it would have sent an executor to "fix" code that is
> already correct.**
>
> *What I asserted (iteration 248 controller):* that "all seven converters" was the wrong unit, and
> that the fix belonged at **both** emitters — the 5 literal writes in
> `internal/gen/golang/codegen_runtime_slices.go` (lines 8/32/60/89/124) **and** the two template
> loops (`funcName := "ConvertTo" + goTypeName + "Slice"` at `:192` and `:315`), which emit one
> converter per ADT/record type in the compiled program and are therefore an unbounded, generated
> set (`codegen_adt_test.go:31,36` already asserts `ConvertToDrawCmdSlice` / `ConvertToCameraSlice`
> for user types).
>
> *What is actually true, measured first-party after the planner refuted it:* the **template-loop
> converters already fail loudly and are the model to copy, not a defect to fix.** Generated
> `runtime.go:845-889`, `ConvertToOptionSlice` and `ConvertToResultSlice`, both carry
> `// M-DX12: Fail-fast - panics on type mismatch (compiler bug detection)` and both read
> `if !ok { panic(fmt.Sprintf("ConvertToOptionSlice: expected []interface{}, got %T", v)) }` —
> plus a second per-element panic. Only the **5 literal** converters `return nil`.
> So **VL-9's "7 silent sites" is wrong: there are 5**, and the quorum's removal applies to those 5.
>
> *How the error was made, since the mission records these:* I counted `return nil` occurrences and
> read the `src, ok := v.([]interface{})` line of a template converter **without reading its next
> line**, then generalised from the literal converters' shape. A transcribed pattern is not a
> measurement (rule 3b(v)(b)). The planner read the branch.
>
> *What survives, and it is the part worth keeping:* **do not enumerate converters by name.** The
> generated set really is unbounded, and the correct fix site is the emitter. The practical
> consequence is smaller than I claimed: M2 edits the **5 literal writes** to match the fail-fast
> pattern the template loops already use — the repo has the right answer in it, twice, and the
> divergence is the bug.
>
> *Two consequences for the ACs, from the planner and verified:* **(a)** AC4's clause about a
> template-loop converter panicking **already passes at base**, so it is a regression pin, not
> fail-at-base evidence, and must be labelled as such. **(b)** `toSlice`
> (`codegen_runtime_collections.go:8`) carries the same silent fallback and is **outside** the
> `ConvertTo*` family, so M2 as scoped does not fully discharge the reviewers' universal
> "no silent fallbacks" goal. Say so rather than implying it does; file `toSlice` as its own row.

**M3 — typed aggregate preservation (day 3).**
Change both array type mappings (`types.TArray` and `ast.ArrayType`) to `ArrayVal`, without changing
the adjacent list mappings. Update ADT-constructor and both pointer/value record-generation paths so
an `Array[T]` field is stored as `ArrayVal` and never routed through `ConvertTo*Slice`. Add focused
type-mapper/codegen tests for primitive, record, and ADT element arrays. This milestone closes the
measured `MkBox(Array[int])` user-visible defect; it is not deferred as a residual.

**M4 — two differential fixtures + docs (day 4, half-day buffer).**
Add `tests/golden/codegen/show_differential_array.ail` for literals/helpers and a separate
`tests/golden/codegen/show_differential_array_adt_field.ail` containing the VL-16 `Box` repro plus
a named-record `Array[int]` field round-trip; bump
`expectedDifferentialFixtureCount` **5 → 7**. Add the CHANGELOG entry and record that generated array
identity is preserved across dynamic, ADT-field, and record-field representations.

## Acceptance criteria (each can fail; baselines stated)

Baseline note: `go build ./...` is **rc=1 on pristine dev** (`cmd/wasm` has no native `main`;
VL-14) — build gates below are scoped to touched packages, which build rc=0 at base.

1. `go test ./tests/golden/codegen/ -run TestInterpreterCompiledDifferential -count=1` passes
   with both new array fixtures present and `expectedDifferentialFixtureCount = 7`.
   *Fails at base three ways*: the count literal is 5 (fixture-count guard trips), the dynamic
   fixture fails (VL-1), and the independent ADT-field fixture fails (VL-16). The first fixture must
   exercise: an array literal, `fromList`, `show` after `set`/`make`, and `show(toList(arr))`
   printing `[…]` (pins Conflict Surface #4). The second must contain `Box = MkBox(Array[int])`,
   construct `MkBox(#[1,2,3])`, extract the field, and show it as `#[1, 2, 3]` in both backends;
   the same fixture must construct/extract/show a named-record `Array[int]` field with identical
   bytes. Thus the ADT and record boundary generators are both end-to-end gated.
2. `go test ./tests/golden/codegen/ -run TestGoldenCompile -count=1` passes — all existing
   fixtures still compile, `go build` + `go vet` clean with `ArrayVal` in the preamble.
3. `go build ./internal/gen/... ./cmd/ailang` rc=0 (rc=0 at base, VL-14 control).
4. `go test ./internal/gen/golang/ -count=1` passes, including new tests asserting each
   `ConvertTo*Slice` handles `ArrayVal` (input `ArrayVal{1,2,3}` → non-nil, len 3 — fails at
   base by silent-nil, VL-9), both array type mappers return `ArrayVal`, both list type mappers
   remain typed slices, and ADT/record `Array[T]` fields do not emit a converter call.
   **And, per the round-2 quorum: no converter retains a silent `nil` fallback.** Assert it at the
   EMITTER, not by name — `grep -c 'return nil' ` over the `!ok` branches of *both* emitters must be
   **0**, and a generated-runtime test must show an unsupported input panicking with a stable,
   converter-specific message for (a) one of the 5 literal converters and (b) one converter produced
   by the `codegen_runtime_slices.go:192`/`:315` template loops — i.e. one for a **user-defined ADT**,
   which is the case a seven-name enumeration cannot reach. *Fails at base*: every `!ok` branch
   returns `nil` today (VL-9).
   Note the `v == nil` guards at generated `runtime.go:848/872` are a **different** branch and are
   legitimate (a nil input is not a type error); do not delete those while removing the `!ok` ones.
5. The V1 repro program byte-matches between backends: interpreted and compiled stdout both
   `#[1, 2, 3]\n[1, 2, 3]\n`. *Fails at base* (VL-1).
6. `make verify-examples` green (no example output pins change — (B) alters compiled output only,
   toward what the docs already show).
7. The VL-16 `MkBox(Array[int])` repro byte-matches independently of the fixture suite: interpreted
   and compiled output are both `#[1, 2, 3]\n`; generated code contains an `ArrayVal` field/argument
   and contains no `NewBoxMkBox(ConvertToInt64Slice(` call. *Fails at base* exactly as VL-16 records.

## Verification Log

Every row measured 2026-08-22 at `404226a48` with the freshly built binary first in PATH.

| # | Claim | Command | Observed |
|---|---|---|---|
| VL-1 | Divergence is real at HEAD | `ailang run --quiet --caps IO --relax-modules v1.ail` then compile → `go build` → run (program above) | run rc=0: `#[1, 2, 3]` / `[1, 2, 3]`. compile rc=0, `go build` rc=0, binary: `[1, 2, 3]` / `[1, 2, 3]` |
| VL-2 | Erasure at construction; array and list literals byte-identical | `grep -n 'interface{}{int64(1), int64(2), int64(3)}' out/main/v1.go` | 2 hits (`tmp1`, `tmp3`) — both `var tmpN interface{} = []interface{}{…}` |
| VL-3 | No dictionary dispatch for show; builtin not type class | `grep -n 'Show' out/main/dictionaries.go` (control: file exists); `grep -n 'func Show(' out/main/*.go` | 0 hits in dictionaries.go (file present); `runtime.go:392: func Show(v interface{}) string` |
| VL-4 | Type checker rejects list→array-helper | `ailang check v4.ail` where body is `get([1,2,3], 0)` with `import std/array (get)` | rc=1: `cannot unify array type Array[α4] with [α5]` |
| VL-5 | M-TYPE1 never touched codegen | `git show --stat 743f6a539`; `git log --oneline -S 'M-TYPE1' -- internal/gen/golang/`; `grep -c '\[\]interface' design_docs/implemented/v0_5_6/m-type1-array-tarray-unification.md` (control: `grep -c 'Array'` same file) | 1 file `internal/types/unification.go`, 8 insertions; tag introduced by `474adf0cf` + `ea88158ef`; 0 `[]interface` hits vs 63 `Array` hits |
| VL-6 | Tuple precedent is a defined slice type discriminated in showValue | `grep -n 'type Tuple' out/main/runtime.go`; `grep -rn 'case Tuple' internal/gen/golang/*.go`; `sed -n '380,470p' out/main/runtime.go` | `runtime.go:18: type Tuple []interface{}`; emitter `codegen_runtime_misc.go:100`; `case Tuple:` sits directly above the `default: showReflect` in `showValue` |
| VL-7 | Blast radius counts and their source files | `grep -c '\.(\[\]interface{})' out/main/runtime.go`; same grep on the two codegen files; `grep -c 'reflect\.Slice' out/main/runtime.go`; `grep -rlc` over `internal/gen/golang/` | 21 exact assertions (emitters: `codegen_runtime_collections.go` 14, `codegen_runtime_slices.go` 7; only other match is a test file); 10 `reflect.Slice` branches |
| VL-8 | All 8 array helpers have reflect fallbacks; FromList/ToList already copy | `sed -n '680,790p' out/main/runtime.go`; `grep -n 'func '` over runtime.go | `FromList`(689)/`ToList`(714)/`Length`(737)/`Get`(755)/`GetOpt`(776)/`UnsafeGet`(809)/`Set`(818)/`Make`(833), each: fast-path assertion, then `reflect.ValueOf` + `Kind()==Slice` path; both FromList and ToList `make`+`copy` |
| VL-9 | Converters silently return nil on `!ok` | `sed -n '5,30p' internal/gen/golang/codegen_runtime_slices.go` | emitted body: `slice, ok := v.([]interface{})` / `if !ok { return nil }` — 7 such sites (lines 15, 39, 67, 102, 137, 220, 340) |
| VL-10 | Array `==` is a type error (equality out of scope) | `ailang check eq.ail` with body `#[1,2] == #[1,2]` | `No instance for Eq[Array[int]] in scope` |
| VL-10b | Lists also lack `Eq`; reviewer proposal `import std/eq (==)` is refuted because that module is absent | `test -f std/eq.ail; echo`; control in same call: `test -f std/list.ail`; `rg -n 'instance Eq\|Eq\[' std/*.ail`; control in same scope: `rg -l 'export' std/*.ail`; then check bodies `[1,2] == [1,2]` and `#[1,2] == #[1,2]` | `std/eq.ail ABSENT`; control `std/list.ail EXISTS`; Eq search 0 hits; control 45 exporting files (45 `.ail` modules total). Checks rc=1: `No instance for Eq[[int]] in scope` and `No instance for Eq[Array[int]] in scope`. Neither type reaches generated equality code. |
| VL-11 | Interpreter render sites for `#[` are exactly two; canonical_key unaffected | `grep -rn '"#\["\|#\[' internal/eval/value.go internal/builtins/show.go`; `grep -n '#\[' internal/builtins/canonical_key.go` | `value.go:110` (`ArrayValue.String()`), `show.go:126` (`showSequence(…, "#[", "]")`); canonical_key.go: 0 hits (control: file matched `ArrayValue` grep in Conflict #8) |
| VL-12 | Option (C)'s doc/test footprint, measured | `grep -rc '#\[' docs/ --include='*.md'`; `ailang prompt \| grep -c '#\['` (control: `wc -l` = 2547); `grep -rln '#\[1' internal/ --include='*_test.go'`; `grep -rln '#\[' examples/` | `arrays.md`: 33; served prompt: 3; test pins: `show_test.go` (+2 format-syntax tests unaffected by rendering); examples: 1 `.ail` + 3 trace JSONLs |
| VL-13 | A typed parameter alone still routes through the dynamic path; aggregate fields are the distinct typed-slice path | typed.ail: `f(a: Array[int]) -> string { show(a) }`; run both backends; `grep -n '\[\]int64{' tout/main/typed.go` | interpreter `#[1, 2, 3]`, compiled `[1, 2, 3]`; **no** `[]int64` literal emitted — still `[]interface{}` via `_impl` (M-DX26) |
| VL-14 | Build baselines | `go build ./... ; echo rc=$?` (full log captured); `go build ./internal/gen/... ./cmd/ailang` | full: rc=1, sole error `cmd/wasm … function main is undeclared`; scoped: rc=0 |
| VL-15 | Differential harness shape | `Read tests/golden/codegen/golden_test.go`; `ls tests/golden/codegen/show_differential*.ail`; `grep -rn '#\[' tests/golden/codegen/*.ail` | `expectedDifferentialFixtureCount = 5` at line 14, glob `show_differential*.ail` at line 81, byte-equality at line 137; 5 fixtures exist; **no fixture contains `#[`** — arrays are exactly the uncovered case |
| VL-16 | Typed ADT field erases array identity and diverges end to end (this agent's rerun of controller repro) | Fresh binary first in PATH: `ailang run --quiet --caps IO --relax-modules resid.ail`; `ailang compile --emit-go --out /tmp/ailang-i248-r2/out --package-name resid --relax-modules --no-verify-go resid.ail`; copy the generated dot-prefixed module file to `resid_generated.go`; add same-package `TestRun { resid__Main() }`; `go test -run TestRun -v`; `rg -n 'return NewBoxMkBox' .tmp-i248-resid.go` | run rc=0: `#[1, 2, 3]`; compile rc=0 and generated-package test rc=0: `[1, 2, 3]`; generated expression: `return NewBoxMkBox(ConvertToInt64Slice(tmp2))`. Constructor signature is `NewBoxMkBox(v0 []int64)`. |
| VL-17 | `std/prelude` named by the equality diagnostic does not exist | Same call: `test -f std/prelude.ail`; `rg -n 'import std/prelude' examples std`; positive control `rg -n 'import std/' examples std`; `ailang check prelude.ail` containing `import std/prelude` | `std/prelude.ail ABSENT`; 0 prelude imports vs 597 control `std/` imports; check rc=1: `IMP012_UNSUPPORTED_NAMESPACE … namespace imports not yet supported`, suggesting a selective import from a module that is absent. |

## Adjacent defects found while verifying

The type checker's exact equality diagnostic says: `Import std/prelude, or derive/define one.`
However, `std/prelude.ail` is absent, there are 0 `import std/prelude` occurrences across `examples/`
and `std/` (positive control: 597 `import std/` occurrences), and attempting that import returns
`IMP012_UNSUPPORTED_NAMESPACE … namespace imports not yet supported` (VL-17). This is a separate
user-facing diagnostic defect: it directs users to a module that this stdlib does not provide. It
is recorded here but is not fixed and does not widen this sprint.

## Out of scope

- Interpreter changes of any kind. The interpreter's `#[…]` rendering is the reference behaviour
  this sprint makes the compiled backend match.
- Any `Eq`/ordering story for arrays (type error today, VL-10).

## Quorum verification log

| Round | Reviewers present | Verdict | Disposition |
|---|---|---|---|
| R1 (`…2026-08-22T03-40-39Z`) | `gpt5-6-sol`, `gemini-3-1-pro` (absent: **none**) + controller | **BLOCKED** | Two objections. Both were **measured by the controller before routing** (mission rule: a reviewer's objection is a claim too). `gpt5-6-sol` — typed-slice residual reachable via ADT/record fields: **CONFIRMED first-party**, `MkBox(Array[int])` prints `#[1, 2, 3]` interpreted / `[1, 2, 3]` compiled because the generated constructor calls `ConvertToInt64Slice`. `gemini-3-1-pro` — "re-measure with `std/eq`": **REFUTED**, `std/eq.ail` is absent (control `std/list.ail` present), 0 `Eq` instances across 45 std modules (control: 45 files match `export`). |
| R2 (`…2026-08-22T03-4xZ`) | `gpt5-6-sol`, `gemini-3-1-pro` (absent: **none**) + controller | **BLOCKED** | Round-1 objections satisfied (M3 brings the typed path in scope; VL-10b/VL-17 record the refutation). Both reviewers then converged **independently on one new objection**: the doc patched the `ConvertTo*Slice` silent `return nil` for `ArrayVal` only, leaving the Principle-2 violation live for every other input. |
| Carve-out | — | **APPLIED** | The mission's narrow-refinement carve-out (ratified by Mark 2026-07-24, "apply and route"). Criteria hold: the sole remaining objection carries a concrete reviewer-authored `proposed_fix` from **both** reviewers, and it disputes **completeness**, not the design direction — both reviewers accept option (B). Both fixes are applied **verbatim** and quoted in M2; AC4 and Conflict Surface #1 updated as their `proposed_fix` text requires. The controller added one **measured scope correction** (apply at the two emitters, not to seven names) that makes the reviewers' own fix achieve its stated goal — it satisfies the objection and does not override it. |

**Premise measurements taken by the controller during quorum** (all at `origin/dev` = `404226a48`,
freshly built binary prepended to PATH, `~/go/bin` untouched):
`std/eq.ail` ABSENT / `std/list.ail` EXISTS · 0 `instance Eq` hits across `std/*.ail` / 45 files
match `export` · `import std/prelude` is a parse error and occurs 0 times in `examples/`+`std/` ·
`MkBox(Array[int])` repro diverges as recorded in VL-16 · `codegen_runtime_slices.go` writes 5
converters literally and holds 7 `.([]interface{})` assertions, while the generated runtime holds 7
`ConvertTo*Slice` functions, the extra two coming from the template loops at `:192`/`:315`.
