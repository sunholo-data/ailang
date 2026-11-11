# M-V1-SHOW-TOTAL: make `show` total and unambiguous before it is frozen

**Status**: Implemented on branch `v1/show-total` (M1–M5), in review. See [Implementation](#implementation-2026-11-11)
**Target**: v0.58.x (the last breaking window before v1.0.0)
**Priority**: P0. `show` is Stable, it is the function the teaching prompt tells every model to use for every type, and its output is graded byte for byte. After the freeze this change needs 2.0
**Estimated**: about 7 working days (estimate, not a plan; per-milestone sizes below)
**Dependencies**: none blocking. Sibling: [m-v1-downstream-compat-gate.md](m-v1-downstream-compat-gate.md) (its gate is how M5 measures downstream), [m-v1-docs-for-1-0.md](m-v1-docs-for-1-0.md) (`<!-- SIBLING:show -->` placeholder and the LIMITATIONS rows), [m-bytecode-vm-parity-bugs.md](m-bytecode-vm-parity-bugs.md)
**Planner-Lane**: opus-required (a Stable output contract shared by two engines and the Go emitter)
**Source**: [v1 release-readiness review §3.2](../v1-release-readiness-review-2026-11-09.md#32-show); earlier triage row [v047-numerics-followups-show-ingest-docs-macos.md](../ailang-core-triage/v047-numerics-followups-show-ingest-docs-macos.md) item 1, which asked for this doc and was never acted on
**Lane** (PROGRAM.md): AILANG fix. A builtin's output format cannot be an extension.

All measurements below come from a build of `dev` at `51f6636ae` (`ailang-base`) and from the same
tree with a 40-line prototype patch to `internal/eval/show.go` and `internal/vm/builtins.go`
(`ailang-new`). The patch was built in the session scratchpad only, and no repo file was changed. Scripts and
raw output are in `scratchpad/dd-show/` (see [Verification log](#verification-log)).

## Problem

`show : ∀a. a -> string` is registered `StabilityStable` (`internal/builtins/show.go:24-41`) and
documented as "canonical string form". Its value has three defects that are visible in the returned string itself,
not just in what gets printed:

1. **Elision.** Every list, array, tuple, record and map whose rendering is wider than 80 bytes has its
   middle replaced by `...` (`internal/eval/show.go:50-56`, `truncateIfNeeded` at `:140-145`), and
   anything nested deeper than 3 levels becomes `...` (`:64-66`). `show(range(0, 50))` is the
   43-character `[0, 1, 2, 3, 4, 5, 6... 45, 46, 47, 48, 49]`, which still has both brackets and so looks complete.
   Measured side effects that the review did not list:
   - The cut is by **byte**, so it can split a UTF-8 sequence: `show(map(\i. "éé", range(0, 40)))`
     returns a string that is **not valid UTF-8** (`iconv` rejects it; bytes `c3 2e 2e 2e a9`).
   - A tuple of two wide lists, `(range(0,30), range(0,30))`, renders as
     `([0, 1, 2, 3, 4, 5, ...25, 26, 27, 28, 29])`, which reads as one list.
   - `bytes` has a second, independent cut: more than 32 bytes renders as `<bytes:<64 hex>...>`, and
     upload-backed bytes render as `<bytes:N:mime:filename>` with no content (`internal/eval/value.go:54-62`).
2. **Non-injective.** Strings are rendered bare at every depth (`show.go:171`, VM `builtins.go:297`):
   `show(["a", "b, c"])` = `[a, b, c]` = `show(["a", "b", "c"])`; `show(Some("x"))` = `Some(x)`;
   `show(["", "x"])` = `[, x]`. The teaching prompt itself ships an example that cannot be read:
   `R.split(re, "x a=1 y")` is documented as `-- [x ,  y]`.
3. **Deep values are cheap only because they are cut.** Wide containers already cost O(n), because the
   renderer builds the full string and then cuts it. The depth limit is the only thing that keeps deep values cheap,
   and the renderer concatenates child strings at each level. Removing the limits naively is
   **quadratic in depth** (measured on the prototype: a 100 000-deep user ADT takes 5.2 s and 3.5 GB RSS,
   50 000 takes 1.5 s and 1.5 GB, 25 000 takes 0.47 s and 0.66 GB; a flat 1 000 000-element list takes 0.19 s and 222 MB, the same as base).

Who is affected: any program that serialises, compares, hashes, or parses back a `show` result, and
every benchmark grader, because they compare stdout exactly (`internal/eval_harness/runner.go:354`,
trim + exact, else JSON-equal).

## Current behaviour, traced

### Who renders values today (six spellings, not one)

| # | Renderer | File | Used by | Elides? | Nested strings | Floats |
|---|----------|------|---------|---------|----------------|--------|
| 1 | `eval.RenderShow` | `internal/eval/show.go:59` | `show` builtin (`internal/builtins/show.go:63-72`), `_show` in the simple/typed evaluator (`eval_simple.go:40`, `eval_typed_helpers.go:199`, `eval_simple_helpers.go:15`), VM `builtinShow` (`internal/vm/builtins.go:263-276`), string interpolation (parser wraps every `${…}` hole in `show`, `internal/parser/parser_literals.go:163-197`) | yes | bare | `'f'` shortest, `.0` |
| 2 | `Value.String()` | `internal/eval/value.go:21-426` | REPL (`internal/repl/repl_format.go:76`), `run --print` results (`internal/runner/entrypoint.go:167,206`, `vm.go:276` via the bridge), test-runner failures (`internal/testing/runner.go:195`, `%v`) | no | bare | `%g` (`1e+300`) |
| 3 | `ShowBounded` / `ShowTraceBounded` | `internal/eval/show_bounded.go:23`, `trace_redact.go:71` | trace collector (`internal/effects/context_trace.go:38,52`), typed-evaluator trace (`eval_typed_helpers.go:104-122`) | head-only, byte budget, `…(+N bytes elided)` | bare (mirrors #2) | as #2 |
| 4 | Go emitter runtime | `internal/gen/golang/codegen_runtime_misc.go:57-235` | `ailang compile` (Experimental) | yes, a copied constant set | bare | as #1 |
| 5 | WASM `formatValue` | `cmd/wasm/main.go:490` | browser playground | no | bare | `%g`; record fields in **Go map order** (non-deterministic; read in code, not run) |
| 6 | `bytecode.Value.String()` | `internal/bytecode` | debug only | — | Go `%q` | — |

Evaluator and VM **share** #1. The probe below printed byte-identical output on `run` and
`run --bytecode --strict-bytecode` (array literals excluded because they still bridge as BC001).
**No internal logging path calls `show`.** Traces already use their own bounded renderer (#3), and REPL and
test output use #2, which never elided. So the "display helper" in this doc is for user code only;
no internal caller needs to switch (see [Contradictions with the review](#contradictions-with-the-review)).

### Current output per value kind (measured, `ailang-base`)

| Value | `show(v)` today | Defect |
|-------|-----------------|--------|
| `42`, `-7`, `9223372036854775807` | `42`, `-7`, `9223372036854775807` | — |
| `3.14`, `2.0`, `-0.0`, `0.1+0.2` | `3.14`, `2.0`, `-0.0`, `0.30000000000000004` | — |
| `1e300`, `1e-300` | 301 digits + `.0`; `0.` + 299 zeros + `1` | long, but exact |
| `0.0/0.0`, `±1.0/0.0` | `NaN`, `Inf`, `-Inf` | — |
| `true` | `true` | — |
| `"hi \"q\"\n"` (top level) | `hi "q"` + newline | — (identity) |
| `()` | `()` | — |
| `[1, 2, 3]`, `[]` | `[1, 2, 3]`, `[]` | — |
| `["a", "b, c"]`, `["", "x"]` | `[a, b, c]`, `[, x]` | ambiguous |
| `#[1, 2, 3]` | `#[1, 2, 3]` | — |
| `(1, "x", true)` | `(1, x, true)` | ambiguous |
| `{b: "two", a: 1}` | `{a: 1, b: two}` (fields sorted) | ambiguous |
| `Circle(3)`, `Dot`, `Rect(1, 2)` | same | — |
| `Some("x")`, `None`, `Ok("y")`, `Err("e")` | `Some(x)`, `None`, `Ok(y)`, `Err(e)` | ambiguous |
| 5-deep `Node(…)` tree | `Node(Node(Node(Node(..., ..., ...), 2, Leaf), 3, Leaf), 4, Leaf)` | lossy |
| `[[[[[1]]]]]` | `[[[[...]]]]` | lossy |
| function, lambda | `<function>` | not injective (no `==` on functions) |
| `fromString("hi")` | `<bytes:6869>`; >32 bytes cut with `...` | lossy |
| `M.fromList([(9,"a"),(10,"b"),(2,"c")])` | `Map{10: b, 2: c, 9: a}` (encoded-key order) | ambiguous |
| `S.fromList(["b","a"])` | `Set(Map{a: (), b: ()})` | ambiguous |
| `range(0, 50)` | `[0, 1, 2, 3, 4, 5, 6... 45, 46, 47, 48, 49]` | lossy |
| `"${["a","b"]}"`, `"${Some("x")}"` | `[a, b]`, `Some(x)` | ambiguous |

## Specification: the new `show`

**Contract (to be frozen):** `show` is *total* (it renders every part of a value; no width, depth or
byte limit), *deterministic* (it is a pure function of the value, the same on every engine and OS), and *injective per type*:
for any type `T` other than function types, and values `x, y : T` that contain no NaN,
`show(x) == show(y)` implies `x` and `y` are structurally identical. It is **not** promised to parse back
as AILANG source; it reads like a literal wherever AILANG has a literal.

| Kind | Format | Change |
|------|--------|--------|
| int | base 10, leading `-` | none |
| float | `strconv.FormatFloat(f,'f',-1,64)`, `.0` appended when no `.`; `NaN`, `Inf`, `-Inf`; `-0.0` kept distinct from `0.0` | none |
| bool | `true` / `false` | none |
| string, **top level** | the string itself, unquoted | none |
| string, **nested** (anywhere inside a list, array, tuple, record, map, constructor) | double-quoted, escaped per the table below | **changed** |
| unit | `()` | none |
| list / array / tuple | `[a, b]` / `#[a, b]` / `(a, b)`; separator `", "`; never elided | **no elision** |
| record | `{f: v, g: w}`, fields in byte order of name; `{}` | nested strings quoted |
| constructor | `Name` if nullary, else `Name(a, b)`; never cut by depth | **no depth limit** |
| map | `Map{k: v, …}` in the map's own iteration order (encoded-key order, the order `M.toList` and `M.keys` already use); keys and values rendered as nested; `Map{}` | nested strings quoted |
| set | `Set(Map{"a": (), "b": ()})`, the honest rendering of std/set's exported `Set(Map[a, ()])` constructor (`std/set.ail:25`); no special case | nested strings quoted |
| bytes | `<bytes:HEX>`, the full content in lowercase hex at any length; the filename/mime form is dropped | **total** |
| function / builtin / closure | `<function>` | none (outside the injectivity domain) |

**Nested string escaping** is the escaper `ailang fmt` already uses for canonical literals
(`internal/format/literal.go:25`, which round-trips through the lexer's `readEscape`), moved into a leaf package
both can import:

| Input | Output |
|-------|--------|
| `"` `\` | `\"` `\\` |
| LF, TAB, CR | `\n`, `\t`, `\r` |
| `$` immediately followed by `{` | `\$` |
| other C0 controls, DEL, C1, non-graphic runes | `\u{X}` (uppercase hex, no padding) |
| a byte that is not valid UTF-8 | `\xHH` (uppercase). Printable runes are never written as `\xHH`, so this stays injective. It is the one spelling that would not re-lex to the same value, which is acceptable because round-tripping is not promised |
| everything else, including non-ASCII printable text | verbatim |

### Why top-level strings stay bare

Recommendation: **keep `show("x") == "x"` and quote only nested strings.** Reasons, strongest first:

1. **Interpolation is `show`.** The parser wraps every `${…}` hole in `show` before types are known
   (`parser_literals.go:163-197`). `internal/pipeline/show_normalize.go` removes the wrapper for holes
   that are already strings, but it deliberately leaves holes whose type is still a type variable.
   Quoting at top level would put quotes into every polymorphic `"${x}"` that receives a string, and would make
   `show_String` stop being the identity the prompt documents.
2. **Python does the same thing, and Python is what models expect by default.** `str("a")` is `a`, but `str(["a"])` is `['a']`.
   The asymmetry is the familiar `str` vs `repr` split, applied in the way models already expect.
3. **Injectivity is a per-type property, and identity is injective on `string`.** Across types `show` was never
   injective (`show("[1]") == show([1])`), and it cannot be without type tags.
4. **Blast radius.** `println(show(s))` on strings is everywhere: 942 `show(` call sites in `examples/`
   and 1 490 in the motoko fork. Quoting at top level would change most of them, while quoting only nested strings changes 23
   example programs (measured below).

An always-quoted `repr`/`debug` form is a clean *addition*, and it can come in any 1.x minor if
someone needs it. It is not needed for the freeze.

### Deep values: required implementation shape

The prototype measured quadratic cost in depth, so M1 **must** render into a single `strings.Builder`
and walk the value with an explicit work stack. A Go stack overflow is a fatal runtime error that
`recover` cannot catch, so `show` must not rely on Go recursion. Required cost: O(size of output) time and memory.
Huge values are then exactly as expensive as their rendering, and code that only wants a glimpse uses
the helper below.

### The truncating helper

`std/string.showPreview(x: a, maxChars: int) -> string` is additive to a Stable module and backed by a
`_show_preview` builtin:

- If `length(show(x)) <= maxChars`, the result is exactly `show(x)`.
- Otherwise the result is the first `maxChars` code points of `show(x)` (`length` in std/string counts code points), followed by
  `…(+N chars)`, where N is the exact number of code points left out. The marker is deliberately not
  valid AILANG or JSON, and the cut is head-only, so a truncated preview never looks like a complete value.
- It never builds more than `maxChars` of output: it uses a bounded writer in count-only mode after the budget, the same
  technique as `ShowBounded`. `maxChars <= 0` returns just the marker.
- There is no default argument. The teaching prompt recommends `showPreview(v, 200)` for logging.

Internal callers: **none switch**, because none use `show` (table above). M4 optionally moves the display paths
(#2, REPL / `run --print` / test failures) onto the new `show` spelling, so that a user sees one
rendering everywhere.

## Blast radius (measured unless marked)

**(a) Repo outputs.** I ran every `.ail` under `examples/`, `benchmarks/` and `tests/` (674 files; 394 run
cleanly on base with `IO,FS,Clock,Rand,Env`, no args, no stdin) on both binaries:

- 26 programs have different stdout; 3 of those are noise (`modal_rand`, `uuid`, `xml_walk_perf` timing). That leaves **23 real changes**: 22
  from quoting (`[a, usage]` becomes `["a", "usage"]`, `Some(eu-west)` becomes `Some("eu-west")`) and 1 from elision
  (`directory_listing.ail` printed `[.ailang, README.md,...l, zip_xml_fold.ail]`).
- `make verify-examples` checks status, not stdout, so **none of these fail a gate**. Only comments that
  document output (for example `regex_basics.ail`, the source of the prompt lines) need editing.
- Go tests under the prototype: **9 fail in 3 packages**. They are `internal/eval`
  `TestEnvShowIsTheCanonicalRenderer`; `internal/builtins` `TestShow_{Lists,Records,Constructors,DepthLimit,LongStrings,RemainingValueForms}`;
  `internal/vm` `TestArrayAccumulateBothEngines`, `TestStructuredMapKeysBothEngines`. `runtime`, `pipeline`
  and `testing` pass. All 9 pin the old format and are rewritten, not kept (coding standards).
- The teaching prompt (v0.16.12) has **3 lines** showing old output (`ailang prompt` lines 1201, 1203, 1207) and the
  interpolation note at line 1330, which stays true.

**(b) Benchmarks and banked history.**
- `expected_stdout` heuristics across 99 specs flag **1** benchmark whose expected output *is* the old
  rendering: `run_length_encode.yml` expects `[(3, a), (2, b), (4, c), (1, a)]`, which is exactly the lossy
  `show` of `[(int, string)]`. No hidden-test `.out` matched.
- I re-ran **every banked passing AILANG solution** in `eval_results/` (14 652 unique code×benchmark
  pairs) on both binaries. 1 049 reproduce a pass on base; the rest fail on today's binary, mostly from
  syntax drift, missing stdin or input files, and agent workspaces. Of the 1 049, **1 flips pass→fail**: one of
  12 reproducible `run_length_encode` solutions, the one that printed `show(encoded)`.
  The other 1 048 keep the same stdout.
- Historical comparability: the grader is exact or JSON-equal, so banked rows stay comparable on every benchmark except
  `run_length_encode`. One side effect: a nested string list now prints as valid JSON (`["a", "b"]`),
  so it can start matching JSON-shaped expected outputs through the JSON-equal path. Measured
  effect on the 1 049: none.

**(c) Downstream (static census, estimate).** `show(` call sites: docparse 88, ailang-parse 130,
email-parse 322, daneel 729, ailang-packages 276, ailang-world 199, motoko `mk-main` 1 490, std 9
(all in comments). Grep for results that are compared, hashed, used as map keys or parsed back found one real case:
`mk-main/packages/motoko-ext-compaction-structural/compaction_structural.ail:333`, an inline test
asserting `length(show(out)) < length(show(input))`. Elision caps both sides near 43 characters today, so it
probably does not test what its name says (inferred from code, not run). Interpolation holes on containers are
invisible to grep. **The real downstream number comes from the downstream gate** (M5), run once with
`ailang-base` and once with the candidate, diffing stdout.

## Milestones

**M1: total, streaming renderer (2 d).** `show.go`: add `ShowString`, so engines report strings as
strings and `RenderShow` decides quoting by depth. Delete `maxDepth`, `ShowMaxWidth`,
`elisionPrefix/Suffix` and `truncateIfNeeded`. Render into one builder with an explicit stack. Move `escapeString` to a
leaf package shared with `internal/format`. Make `bytes` total. Update `inspectVMShow`. Update the builtin
`LongDesc` (drop "limits depth"), and add a nested-string example.
*Acceptance:* (1) `TestShowSpecGolden`, a data-driven table holding every row of the spec tables above, run
through `eval.Show` **and** through the VM (`vm.showValue` on the bridged value): byte-identical, and equal
to the table. (2) A 1 000 000-deep user ADT and a 10 000 000-element list render with allocation
linear in output size (assert `testing.AllocsPerRun`/bytes ratio, not RSS). (3) The rendering of
`map(\i. "éé", range(0, 40))` is valid UTF-8. (4) The 9 old tests are rewritten.
*Mutation check:* restore `truncateIfNeeded`'s body. The golden rows for `range(0,50)`, the depth-5 tree and
the UTF-8 case must fail. Separately, return `n.Text` for nested strings, and every quoted row must fail. Record
both runs in the sprint log.

**M2: injectivity property test (1 d).** A generator over a closed type grammar: int, float without NaN,
bool, string (biased toward `"`, `\`, `, `, `${`, control characters, astral-plane characters, invalid UTF-8,
`""`), unit, `[T]`, `Array[T]`, tuples, records, `Map[K, V]`, `Option`/`Result`, and a recursive
user ADT, up to depth 8. Property: for 10 000 pairs of the same generated type,
`show(x) == show(y)` implies `x` and `y` are structurally equal. Also: every output is valid UTF-8 except
where an input string held invalid bytes, and the result is identical on both engines.
*Mutation check:* drop the `\\` escape. The property must find a collision within the seed budget (a
string holding a backslash before a quote can then render the same as a differently split pair of strings).

**M3: `showPreview` plus the Go emitter (1.5 d).** Add the builtin, the `std/string` export and its golden, then
`make freeze-stdlib` (an additive change to a Stable golden). Port the spec to the emitted runtime in
`codegen_runtime_misc.go` (delete its copied elision constants). *Acceptance:* a preview of `range(0, 1000000)` with
200 chars allocates under 4 KB beyond the value; cuts fall on code-point boundaries; the marker count is exact;
`ailang compile` output of the golden program matches the evaluator.

**M4: one display spelling (1 d, may slip to 1.x).** Point REPL `formatValue`, `run --print` and
test-runner failure messages at `eval.Show` (unbounded; this matches today's `%v`, which never elided). The trace
renderer (#3) keeps its byte budget but adopts nested-string quoting so traces and `show` agree. File the
non-deterministic record order in WASM `formatValue` as its own bug. Not a freeze blocker: REPL text
and test messages are not in the Stable promise.

**M5: migration and measurement (1.5 d).** Update the 23 example comments that document output, the 3
prompt lines (a new prompt version through prompt-manager), `docs/LIMITATIONS.md` (remove the elision row
that [m-v1-docs-for-1-0.md](m-v1-docs-for-1-0.md) adds, if it landed first), and the `show` row in the
language reference. Run [m-v1-downstream-compat-gate.md](m-v1-downstream-compat-gate.md) with base and candidate, and
list every changed stdout line per repo in the sprint log. Fix `run_length_encode` per Decision 5.
Write the changelog fragment.

## Changelog fragment (draft, `changelogs/unreleased/YYYY-MM-DD-show-total.md`)

```markdown
### Changed — `show` is total and unambiguous (BREAKING)

`show` no longer elides: containers wider than 80 columns used to lose their middle
(`show(range(0, 50))` was `[0, 1, 2, 3, 4, 5, 6... 45, 46, 47, 48, 49]`) and values nested more than
three deep became `...`. It now renders every element. Strings **inside** lists, tuples, records,
maps and constructors are quoted and escaped: `show(["a", "b, c"])` is `["a", "b, c"]` (was
`[a, b, c]`) and `show(Some("x"))` is `Some("x")`. A top-level string is still returned unchanged,
so `"${name}"` and `println(show(s))` print exactly as before. `bytes` render in full.
For bounded output in logs use `std/string.showPreview(value, maxChars)`. The old elided form also
could return invalid UTF-8; that is fixed. Design: design_docs/planned/v1_0_0/m-v1-show-total.md
```

Downstream guidance: search for `show(` whose argument is a container holding strings, and for
interpolation of such values (`"${xs}"`). If the old bare form was intended, render explicitly
(`join(", ", xs)`). If `show` output is parsed, hashed or compared, the new form is what you wanted.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Keep `show` lossy and add `showFull` | Freezes the lossy function as the default. The prompt tells every model to "prefer `show` … works on ALL types", so the lossy one stays the one actually used, and generated code keeps hashing truncated strings |
| Make elision visibly invalid (unterminated bracket) | Still loses data, and a byte-exact grader still sees a value nobody asked for |
| Raise the width (e.g. 1 000) | Moves the cliff without removing it. Output becomes correct or wrong depending on input size, which is the worst failure mode for generated code |
| Round-trippable literal form | Maps, functions and bytes have no literal syntax. It would need a map literal and would freeze more surface. Injectivity gives the comparison guarantee without that |
| Quote top-level strings too (Haskell `show`) | Breaks interpolation and every `println(show(s))`; see [above](#why-top-level-strings-stay-bare) |
| JSON as `show` | Loses tuple versus list and constructor names, and changes the rendering of ints and floats |

For a language whose outputs are compared byte for byte in benchmarks and written by models that
assume Python's `str` behaviour, the cheapest and most predictable rule is: everything is shown, and
strings are quoted when they are inside something.

## Risks

- **Log volume:** user code that relied on elision (`log(show(state))`) now prints everything. Mitigation:
  `showPreview`, a prompt note, and M5's downstream diff that finds such sites.
- **Cost on deep values:** fixed by the M1 shape (linear); the M1 test enforces it.
- **The Go emitter drifts:** M3 deletes its copy of the elision constants; compile-vs-eval golden.
- **Benchmark series break:** limited to `run_length_encode` (measured); handled by Decision 5.
- **Prompt churn:** a new prompt version shifts banked `prompt_version` keys; it is bundled with other v0.58.x prompt edits.

## Non-goals

A literal form that parses back; a `Show` type class or user-defined `show`; pretty-printing with line breaks; changing float
spelling; changing map iteration order (if the stdlib-polish doc makes int keys numeric, `show` follows
automatically); always-quoted `repr` (additive, 1.x).

## Decisions for Mark

1. **Top-level strings bare, nested strings quoted?** Recommend **yes** (interpolation, the Python precedent, blast radius).
2. **Helper name and home:** recommend `std/string.showPreview(x, maxChars)` with a head-only cut and a
   `…(+N chars)` marker. Alternative: a prelude builtin, which costs a prelude name.
3. **Float spelling:** keep shortest-round-trip without an exponent (recommend; exact and already banked, at most about 330 chars).
   Alternative: JS-style exponent outside `[1e-6, 1e21)`. That is a second breaking change and needs its own census.
4. **Set rendering:** keep the honest `Set(Map{…})` (recommend, no special cases in a frozen spec) or
   special-case `Set{…}`.
5. **`run_length_encode`:** recommend a new id (`run_length_encode_v2`) with an explicit output format that no
   language's default printing produces, and retire the old id so ELO and banked rows stay separable.
   The current expected output was written to match the lossy `show`.
6. **M4 before the freeze or in 1.x?** Recommend v0.58.x if there is time; it is not a blocker.

## Contradictions with the review

- The review says truncation "moves to" a display helper, which implies internal callers rely on it.
  Measured: **none do**. Traces have their own bounded renderer, and REPL and test output never elided. The
  helper is new user-facing API, and nothing internal moves.
- The review's evidence is the 80-column cut. The census found three more defects of the same kind: invalid UTF-8 from
  byte-level cuts, a separate 32-byte cut for `bytes`, and tuples whose elision hides the separator.
- "Make it total" is not a constant deletion. Done naively it is quadratic in depth (3.5 GB at 100 000
  levels), so M1 is a renderer rewrite, not a one-line change.

## Could not verify

Sibling repos were grepped, not run (the downstream gate does that). WASM ordering was read in code, not run.
13 603 banked solutions do not reproduce on today's binary, so their behaviour under the change is unknown.
`ailang compile` output was not probed. The 394-of-674 coverage excludes programs that need args, stdin or network.

## Verification log

Scratch: `/private/tmp/claude-501/-Users-voightkampff-dev-sunholo-data-ailang/795df2bf-8bbb-44ec-86b9-f8166df96f9c/scratchpad/dd-show/`

```
probe.ail / utf.ail / utf3.ail      per-kind table, UTF-8 split (iconv: INVALID_UTF8 on base)
ailang-base run vs run --bytecode --strict-bytecode probe_vm.ail   -> SAME
prototype.patch                     the 112-line diff ailang-new was built from (scratch tree since deleted)
census.py examples benchmarks tests -> files 674 ran_ok_base 394 stdout_changed 26   (census_main.txt)
bench_census.py                     -> 99 specs, 1 matching old rendering (run_length_encode)
banked_census.py                    -> 14652 unique, 1049 reproduced, 1 pass_to_fail   (banked_census.txt)
static_census.py <repos>            -> show( counts and compare/hash/parse sites above
deep1_{25000,50000,100000}.ail      -> 0.47s/664MB, 1.54s/1.53GB, 5.21s/3.53GB (prototype, naive)
go test ./internal/{eval,builtins,vm,runtime,pipeline,testing}  (prototype) -> 9 FAIL in 3 packages
printf '["a", "b, c"]\n1.0e300\n' | ailang-base repl   -> [a, b, c] :: [String]; 1e+300 :: Float
```

## Implementation (2026-11-11)

Branch `v1/show-total`. All five milestones landed; the open decisions were taken as recommended.

| Decision | Taken |
|---|---|
| 1. Top-level strings bare, nested quoted | Yes |
| 2. Helper name and home | `std/string.showPreview(x, maxChars)`, head-only cut, `…(+N chars)`; backed by `_show_preview` (native on the VM as `__show_preview`) |
| 3. Float spelling | Unchanged (shortest round-trip, no exponent) |
| 4. Set rendering | Honest `Set(Map{…})`, no special case |
| 5. `run_length_encode` | Retired to `benchmarks/retired/`; `run_length_encode_v2` states `3a 2b 4c 1a` |
| 6. M4 now or 1.x | Now: REPL echo, `run --print`, test messages, traces and the WASM display all use the show walk |

Where it differs from the plan:

- **One walk, three budgets.** `eval.RenderShow` (unbounded), `RenderShowPreview` (code-point budget)
  and `ShowBounded`/`ShowTraceBounded` (byte budget, trace redaction as an inspector wrapper) share
  one explicit-stack walk in `internal/eval/show.go`. The old `boundedWriter` (a second renderer
  that mirrored `Value.String()`) is gone, so traces now spell floats and nested strings as `show`.
- **State is two stacks**: one byte per open container plus a frame only for containers with
  children left, so `S(S(…))` a million deep costs about 1 MB of state. Output buffers grow by
  doubling (append's 1.25x policy re-copied a 30 MB rendering about five times).
- `ShowNode` is generic over the engine's value type, so neither engine boxes its children.
- The escaper moved to `internal/strutil.Quote` (an existing stdlib-leaf package), used by `ailang fmt` too.
- **NFC (review rounds 1–3).** The loader NFC-normalises source, so the escaper writes `\u{X}`
  for a character that is not an NFC-stable starter whenever normalisation *might* change the
  string, and right after an escape. "Might" is one conservative linear pass (`nfcMayChange`):
  the UAX #15 quick check plus x/text's stream-safe limit (after 30 non-starters it inserts
  U+034F), counted as x/text counts them — over the compatibility decomposition, with
  backward-combining starters (Hangul vowels and trailing jamo) as non-starters. It never
  normalises or sorts. The tables are generated from x/text and must match the Unicode version
  the toolchain selects (x/text switches to Unicode 17 tables under go1.27):
  `TestNFCTablesUpToDate` fails with the regenerate commands on a toolchain or x/text bump.
- `Value.String()` is unchanged: it is still the Go-side spelling of error messages and debug
  output, and no longer reaches stdout.
- `examples/manifest.json` now pins stdout (`verify-examples` compares it), so 10 pins were
  re-pinned; the doc's census predates that gate.
- **M2's mutation note does not hold.** Dropping only the `\\` escape does not break injectivity
  (an opening quote is never preceded by a backslash, so the text stays decodable); it breaks the
  literal round trip. The property test therefore also re-lexes every nested string, and that check
  catches the mutation. Dropping `\"` is caught by the injectivity property itself.
- Measured downstream (gate + `ailang test` on every inline-test file of all eight corpora, both
  binaries): one change, a fix — the motoko compaction test that §(c) predicted tested nothing.
