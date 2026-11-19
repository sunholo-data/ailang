# Sprint Plan — M-ARRAY-SHOW-DIVERGES-RUN-VS-COMPILE

**Design doc**: [`m-array-show-diverges-run-vs-compile.md`](m-array-show-diverges-run-vs-compile.md) (PLANNED, quorum-cleared via carve-out)
**Sprint ID**: `M-ARRAY-SHOW-DIVERGES-RUN-VS-COMPILE`
**Base commit**: `404226a48` (`origin/dev`); worktree HEAD `90211aa50` adds only the design doc — `git diff --name-only 404226a48 HEAD` = 1 file, so the code tree under test **is** `404226a48`.
**Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-iter248`, branch `sprint/iter248-array-show`
**Duration**: **4 days** (see *Estimate* — the doc's 3–4 is kept, but re-allocated across milestones)
**Total LOC estimate**: ~520 (impl ~300, tests ~220)
**Risk**: **medium** — M1/M2/M4 are low-risk and have an exact in-repo precedent (`Tuple`); M3 carries the entire sprint's risk and the design doc under-scopes it in three measured ways (R2, R3, R4 below).
**GitHub issues**: none (no dedicated issue; `gh issue list --search "array show compile"` → only the week's bookkeeping row #745)

---

## 0. Instrument note — READ THIS BEFORE RUNNING ANY ACCEPTANCE COMMAND

`tests/golden/codegen/golden_test.go` shells out to a **bare `ailang`** at lines 37, 102 and 109.
It therefore tests whatever binary is first in `PATH`, not the tree.

Measured in this worktree, same test, same commit, two binaries:

| `PATH` head | `go test ./tests/golden/codegen/ -count=1` |
|---|---|
| `~/go/bin/ailang` (dated **Aug 18**, predates PR #822) | **rc=1** — 6 failing subtests incl. `TestGoldenCompile/string_charat` and all 5 `show_differential*` |
| `/tmp/i248-plan/ailang` (built from this tree, `go build -o /tmp/i248-plan/ailang ./cmd/ailang`, rc=0) | **rc=0** — `ok … 5.065s` |

**Every command in this plan that names `go test ./tests/golden/codegen/` must be run as:**

```bash
go build -o /tmp/i248-plan/ailang ./cmd/ailang      # rc must be 0
PATH="/tmp/i248-plan:$PATH" go test ./tests/golden/codegen/ -count=1
```

Do **not** run `make quick-install` — `~/go/bin/ailang` is shared with concurrent agents.
Rebuild `/tmp/i248-plan/ailang` after **every** code change; a stale scratch binary is the same trap one level down.

The design doc's baseline note covers `go build ./...` but not this, and this one is far more likely to burn a day.

---

## 1. Measured baselines (all at the worktree HEAD, fresh binary first in `PATH`)

Capture exit codes directly (`cmd > /tmp/out 2>&1; rc=$?`) — never `${PIPESTATUS[0]}` (empty in zsh), never `|| echo 0` around `grep -c`.

| # | Command | Baseline | Is it a usable gate? |
|---|---|---|---|
| B1 | `go build ./...` | **rc=1** — sole error `# github.com/sunholo-data/ailang/cmd/wasm … function main is undeclared` | **No.** Already red. Scope build gates to touched packages. (The doc says `cmd/wasm`; the brief also named `gen/main` — measured, only `cmd/wasm` errors.) |
| B2 | `go build ./internal/gen/... ./cmd/ailang` | rc=0 | Yes |
| B3 | `go vet ./internal/gen/... ./tests/golden/...` | rc=0 | Yes |
| B4 | `go test ./internal/gen/golang/ -count=1` | rc=0 (`ok … 0.388s`) | Yes |
| B5 | `PATH=… go test ./tests/golden/codegen/ -count=1` | rc=0 | Yes — **only** with a fresh binary (§0) |
| B6 | `PATH=… go test ./tests/golden/codegen/ -run TestInterpreterCompiledDifferential -count=1 -v` | rc=0, **5** `--- PASS` subtests | Yes; the subtest count is the non-vacuity control |
| B7 | `make verify-examples` | rc=0 — `211 passed, 0 failed, 6 skipped`, manifest in sync | Yes. Uses `bin/ailang` (built by its own `build` dep), **not** the `PATH` binary — immune to §0 |
| B8 | `make check-file-sizes` | rc=0 | Yes — but see the headroom warning in §4 |

---

## 2. What my measurements contradict in the design doc

A refutation here is the point of this pass, not a problem. Six findings; R1–R4 change the work.

### R1 — The silent `!ok → return nil` fallback exists at **5** emitter sites, not 7. VL-9 is wrong.

VL-9 claims *"emitted body: `slice, ok := v.([]interface{})` / `if !ok { return nil }` — 7 such sites (lines 15, 39, 67, 102, 137, 220, 340)"*.

Measured at `internal/gen/golang/codegen_runtime_slices.go`:

- `grep -n 'if !ok' codegen_runtime_slices.go` → **9** sites: 16, 40, 68, 103, 138, 221, 264, 341, 384.
- The 5 literal converters (16/40/68/103/138) write `return nil`.
- The **two template-loop sites the doc cites as 220 and 340 (measured: 221 and 341) already `panic`**, with a converter-specific message and an explicit tag:
  `panic(fmt.Sprintf("%s: expected []interface{}, got %%T", v))`, commented `// M-DX12: Fail-fast - panics on type mismatch (compiler bug detection).` and `// M-CODEGEN-UNIFIED-SLICE: Record slice converter.`
- Sites 264 and 384 are per-*element* assertions inside the same loops; they also panic. The doc counts neither.

First-party confirmation in generated output (probe program using `std/array`, `out/main/runtime.go`): 7 `ConvertTo*Slice` functions; the 5 literal ones return `nil` on `!ok` at lines 598/614/632/654/676; `ConvertToOptionSlice` (853) and `ConvertToResultSlice` (877) **panic**.

**Consequences, all binding on M2:**

1. The quorum's mandated *removal* of `if !ok { return nil }` applies to **5** emitter sites. The two template loops need **no fallback change** — they already satisfy CLAUDE.md Principle 2 and have since M-DX12.
2. **AC4 clause (b) of the design doc already passes at base.** It requires *"a generated-runtime test [showing] an unsupported input panicking with a stable, converter-specific message for … (b) one converter produced by the `:192`/`:315` template loops."* That is true today. The doc's *"Fails at base: every `!ok` branch returns `nil` today (VL-9)"* is false for 2 of the 7. Keep the test — as a **regression pin**, correctly labelled — but it cannot serve as the fail-at-base evidence for M2.
3. The doc's *controller scope correction* is right that the converter set is generated and unbounded and must be fixed at the emitter, and right that a seven-name enumeration is a defect. Its conclusion *"M2 must edit **both emitters** [for the fallback]"* is wrong: only one emitter carries a fallback. Both emitters do need an `ArrayVal` arm, for the different reason in R2.

### R2 — An `ArrayVal` reaching a template-loop converter **panics** today; it does not silently return nil. This is a crash-regression risk on a shipped example.

Measured probe (`type Dir = North | South; type Plan = MkPlan(Array[Dir])`):

```
generated:  return NewPlanMkPlan(ConvertToDirSlice(tmp2))
signature:  func NewPlanMkPlan(v0 []*Dir) *Plan
converter:  func ConvertToDirSlice(v interface{}) []*Dir   // template loop, panics on !ok
```

So `Array[UserADT]` routes through a **template-generated** converter. The moment M3 maps `ast.ArrayType` to `ArrayVal`, any residual call site feeding `ArrayVal` into that converter **panics at runtime** rather than degrading quietly. `examples/runnable/array_adt.ail` is exactly this shape (`PatternPatrol(Array[Direction])`), is covered by `make verify-examples` (B7, green at base), and would go red loudly. The design doc does not flag this path at all.

This makes B7 a **first-class M3 gate**, not the afterthought AC6 implies.

### R3 — M3's edit-site list is both over- and under-stated. Converter selection is a single chokepoint.

The doc: *"`codegen_expr_app.go` / `codegen_record.go` must not select a `ConvertTo*Slice` conversion for those fields."*

Measured: every converter selection funnels through one function, `func (g *Generator) getSliceConversion(goType string) string` at `codegen.go:715`, keyed purely on the mapped Go type string. Non-test call sites, exhaustively (`grep -rn 'getSliceConversion' internal/gen/golang/*.go`):

| Site | Path |
|---|---|
| `codegen_decl.go:324` | function **return** coercion (`coerceReturnExpr`) — **absent from the doc's "exhaustive" edit-site list** |
| `codegen_expr_app.go:139` | ADT constructor arguments |
| `codegen_record.go:174` | record literal fields |
| `codegen_record.go:337` | second record path |

`getSliceConversion("ArrayVal")` falls to `default`, does not match `strings.HasPrefix(goType, "[]*")`, and returns `""` — so **all four sites stop emitting a converter call automatically** once the two type mappers return `ArrayVal`. No per-call-site surgery is required. This *removes* budgeted M3 work.

The fall-through at each of the four is a bare type assertion (`expr.(ArrayVal)`, `coerceReturnExpr`'s terminal `%s.(%s)`). That is only correct if **every** producer of an array value actually yields `ArrayVal` at runtime — which is precisely what M1 (literals) and M2 (helper return types) establish. **M1 → M2 → M3 is therefore a hard order; the milestones cannot be reordered or parallelised.**

### R4 — `ArrayVal` is a *named* type and 5 non-test sites branch on `strings.HasPrefix(goType, "[]")`. Unbudgeted.

`grep -rn 'HasPrefix([^,]*, "\[\]")' internal/gen/golang/*.go | grep -v _test` → 5 sites:
`adt.go:451`, `codegen_match_patterns.go:34`, `codegen_decl.go:323`, `codegen_record.go:171`, `codegen_record.go:401`.

`"ArrayVal"` fails all five. This is a *smaller* instance of exactly the reason option (A) was rejected — a representation change that invalidates textual/structural discrimination — and the doc lists none of them. The `Tuple` precedent proves the mechanism survives this class (28 non-test `Tuple` mentions in the same package), but the audit is real M3 work, itemised in §4.

### R5 — `std/array.empty` and `std/array.append` do not compile. Keep them out of the fixtures.

```
$ ailang run  … probe_app.ail        →  #[1, 2]           rc=0
$ ailang compile --emit-go --no-verify-go …   rc=0
$ go build -o … ./main/              →  rc=1
     main/probe_app.go:5:22: undefined: Empty
     main/probe_app.go:7:22: undefined: Append
```

`std/array.ail` exports **10** functions; codegen emits runtime helpers for only 8 (`FromList`, `ToList`, `Length`, `Get`, `GetOpt`, `UnsafeGet`, `Set`, `Make` at `codegen_runtime_collections.go:141/177/211/238/270/321/335/357`). `empty` and `append` have **no emitter**. `--no-verify-go` (which the differential harness passes) hides it; only `go build` catches it.

Pre-existing, out of scope, **but** the practical rule is binding: **the M2 fixture must not call `append` or `empty`**, or it will be red for an unrelated reason and burn executor time. The doc's M2 fixture list (`fromList`/`get`/`set`/`make`/`toList`) already avoids them — that is luck, not design, so it is now written down. File as an adjacent defect alongside the doc's `std/prelude` diagnostic finding.

### R6 — `toSlice` carries the same Principle-2 fallback, in shared infrastructure. Recorded, not fixed.

`codegen_runtime_collections.go:8-14` emits `func toSlice(v interface{}) []interface{}` whose body is `if v == nil { return nil }; if s, ok := v.([]interface{}); ok { return s }; return nil` — a silent nil on a failed assertion, in a helper the source itself calls *"core infrastructure used by 24+ runtime helpers"*. It is reached from builtin codegen templates (`internal/builtins/registry_codegen_json.go:67,150`; `registry_codegen_effects.go:52,245,258`).

The type checker keeps arrays out of those list builtins (VL-4), so it is **out of this sprint's scope**. It is nevertheless a second live instance of the exact pattern the round-2 quorum objected to, and both reviewers' stated goal ("universally enforce the no-silent-fallbacks axiom") is not achieved by M2 alone. Recorded here so the next reader does not believe it was.

### Minor
- The doc's `go build ./...` note is right (`cmd/wasm` only); the controller brief's "and `gen/main`" is not reproduced.
- `internal/gen/golang/*_test.go` contains **zero** references to `TArray` or `ast.ArrayType` (control: `codegen_datastructures_test.go` is the only test file matching `Array` at all). The array type mappers have **no unit coverage today**. Good news for churn — M3 breaks no existing test. Bad news for safety — nothing would catch a regression, so M3's new mapper tests are load-bearing, not decorative.

---

## 3. Deliberate deviation from the doc: fixtures land progressively, not all in M4

The brief requires *"milestones independently committable and bisectable, with the relevant test package green at every boundary."* The doc's milestone shape conflicts with that: it puts both fixtures in M4, so M1/M2/M3 have **no end-to-end gate** and M4 carries all the end-to-end risk on the last half-day.

It also cannot be done the doc's other way round: adding the full multi-helper fixture at M1 would leave it **red** at the M1 boundary (helpers still return `[]interface{}` until M2), breaking bisectability.

Resolution — the fixture *files* land with the milestone that makes them green, and `expectedDifferentialFixtureCount` steps 5 → 6 → 7:

| Boundary | Fixtures present | `expectedDifferentialFixtureCount` | `go test ./tests/golden/codegen/` |
|---|---|---|---|
| base | 5 | 5 | green |
| end M1 | 6 (`…_array.ail`, **literals only**) | 6 | green |
| end M2 | 6 (same file, **body extended** with `fromList`/`get`/`set`/`make`/`toList`) | 6 | green |
| end M3 | 7 (+ `…_array_adt_field.ail`) | 7 | green |
| end M4 | 7 | 7 | green |

Final state is identical to the doc's AC1 (two fixtures, count = 7). Every milestone gains a real end-to-end gate. The count guard (`golden_test.go:88-90`, `if len(fixtures) != expectedDifferentialFixtureCount { t.Fatalf }`) makes each step self-enforcing: adding a file without bumping the literal reds the package.

**Confirmed the literal is still 5**: `sed -n '14p' tests/golden/codegen/golden_test.go` → `const expectedDifferentialFixtureCount = 5`.

---

## 4. Milestones

Common to all four: rebuild `/tmp/i248-plan/ailang` before running any golden test; run `make check-file-sizes` at each boundary — **`codegen.go` is at 753 of 800 lines, 47 lines of headroom**, and M1 (+2 preamble sites) and M3 (+`getSliceConversion` handling) both write into it. If headroom runs out, split rather than cram; do not silently raise the limit.

---

### M1 — representation + rendering (Day 1, ~0.5 day) · ~110 LOC (impl 55 / test 55)

Make an array *literal* carry a distinct Go identity and render as `#[…]`.

**Tasks**

1. Emit `type ArrayVal []interface{}` at **both** runtime-preamble sites — `codegen.go:556` and `codegen.go:664` — immediately after the existing `type Tuple []interface{}` write at each. Two sites, one grep to prove it: `grep -c 'type ArrayVal' internal/gen/golang/codegen.go` must be `2`.
2. `codegen_ops.go` `generateArray` (357-375): emit `ArrayVal{` in place of `[]interface{}{`.
   **Also audit the sibling typed branch** at `codegen_ops.go:368` (`g.writef("[]%s{", elemType)`, taken when `elemType != "" && elemType != "interface{}"` and not inside an `_impl`). The doc mentions only the `[]interface{}` branch. Every probe I ran emitted the `interface{}` branch (M-DX26 forces `interface{}` inside `_impl`) and **no** `[]int64{…}` array literal appeared in any generated output — so this branch may be unreachable for array literals today. Decide and record: either route it to `ArrayVal{` too, or add a `DEBUG_STRICT`-visible comment stating why it cannot fire. Do not leave it undecided.
3. `codegen_runtime_misc.go`: add `case ArrayVal:` → `return showSequence(reflect.ValueOf(x), depth, "#[", "]")` directly above `case Tuple:` (line 100) in the emitted `showValue`. Mirrors the proven `Tuple` arm exactly.
4. Rewrite the misattributed comment at `codegen_ops.go:357` (`// M-TYPE1: Arrays use the same Go representation as lists (slices).`) to name this design doc. VL-5 established M-TYPE1 (`743f6a539`) touched only `internal/types/unification.go` — the tag is borrowed and has cost the backlog a false blocker once already.
5. Add `tests/golden/codegen/show_differential_array.ail`, **literals only** (module `test/show_differential_array`; the harness synthesises `func main() { show_differential_array__Main() }`, so the filename must match the module tail). Bump `expectedDifferentialFixtureCount` 5 → **6**.

**Acceptance**

| # | Command | Required | Baseline |
|---|---|---|---|
| M1.1 | `go build -o /tmp/i248-plan/ailang ./cmd/ailang` | rc=0 | rc=0 (B2) |
| M1.2 | `PATH=… go test ./tests/golden/codegen/ -run TestInterpreterCompiledDifferential -count=1 -v` | rc=0 **and** exactly **6** `--- PASS` subtest lines | rc=0, 5 subtests (B6) |
| M1.3 | `go test ./internal/gen/golang/ -count=1` | rc=0 | rc=0 (B4) |
| M1.4 | `grep -c 'type ArrayVal' internal/gen/golang/codegen.go` | `2` | `0` |
| M1.5 | New unit test: generate a program containing an array literal; assert the emitted source contains `ArrayVal{` **and** the preamble contains `type ArrayVal []interface{}` **and** `showValue` contains `case ArrayVal:` | pass | fails (no such text) |
| M1.6 | `PATH=… go test ./tests/golden/codegen/ -run TestGoldenCompile -count=1` | rc=0 | rc=0 (B5) |
| M1.7 | `make check-file-sizes` | rc=0 | rc=0 (B8) |

**Kills which mutation.** M1.2's observable is *byte-identical stdout between the interpreter and a compiled binary*, produced by `exec`ing two real processes (`golden_test.go:102,133`) — strictly downstream of the emitter, the Go compiler, and `showValue`. Nothing else in the system produces those bytes. The `-v` subtest count is the non-vacuity control: a fixture-count guard that passes because the glob matched nothing would show 0 `--- PASS` lines, and `go test` reports `ok` for a package with no tests, so **`rc=0` alone is not sufficient evidence** — the count is.
M1.4/M1.5 are *source-text* assertions, adjacent to the mechanism rather than downstream of it; they are cheap localisers, not the gate. They are listed after M1.2 for that reason.

---

### M2 — helpers return `ArrayVal`; the silent fallback is removed (Day 2, ~1 day) · ~180 LOC (impl 100 / test 80)

**Tasks**

1. `codegen_runtime_collections.go`: `FromList` (141), `Set` (335), `Make` (357) return `ArrayVal`. `ToList` (177) **keeps returning plain `[]interface{}`** — pinned by the fixture (Conflict Surface #4). Add `ArrayVal` fast paths to the 8 helpers where cheap; the pre-existing `reflect.Kind()==Slice` fallbacks (VL-8) already compute correct answers for `ArrayVal`, so this is an optimisation, not a correctness requirement — say so in the commit message rather than implying otherwise.
2. **Remove `if !ok { return nil }` from the 5 literal converters** in `codegen_runtime_slices.go` (emitter lines 16/40/68/103/138). Replace with:
   - an explicit `ArrayVal` acceptance arm (convert, do not reject), and
   - `panic(fmt.Sprintf("ConvertToInt64Slice: expected list or array slice, got %T", v))` on anything else — converter-specific message, verbatim per the round-2 quorum text.
   **Preserve the adjacent `v == nil { return nil }` guards** at emitter lines 10/34/62/93/128 (generated `runtime.go:598` region; the doc cites generated `848/872`). A nil input is not a type error; deleting these is the failure mode the doc explicitly warns about.
3. **Add an `ArrayVal` acceptance arm to both template loops** (`codegen_runtime_slices.go:192` and `:315`) — per R2 these already fail loudly, so this is *not* a fallback removal; it is preventing a **panic regression** when M3 starts feeding them `ArrayVal`. Leave their existing `panic` messages untouched.
4. Extend `show_differential_array.ail` in place to flow through `fromList` / `get` / `set` / `make` / `toList`. Count stays 6. **Do not add `append` or `empty`** (R5 — they do not compile).

**Acceptance**

| # | Command | Required | Baseline |
|---|---|---|---|
| M2.1 | `PATH=… go test ./tests/golden/codegen/ -run TestInterpreterCompiledDifferential -count=1 -v` | rc=0, **6** `--- PASS` | at base the extended fixture body diverges on 4 of 7 lines — measured interp `#[1, 2, 3] / #[4, 5, 6] / #[9, 2, 3] / #[7, 7, 7] / [1, 2, 3] / 3 / 2` vs compiled `[1, 2, 3] / [4, 5, 6] / [9, 2, 3] / [7, 7, 7] / [1, 2, 3] / 3 / 2` |
| M2.2 | Generated-runtime behavioural test, **one of the 5 literal converters**: compile a program, run `ConvertToInt64Slice(int64(7))` in the generated package, assert it **panics** and the recovered message equals `ConvertToInt64Slice: expected list or array slice, got int64` | pass | fails at base — returns `nil`, no panic |
| M2.3 | Generated-runtime behavioural test, **one template-loop converter for a user-defined ADT** (the case a five- or seven-name enumeration cannot reach): assert `ConvertTo<UserADT>Slice(int64(7))` panics with `ConvertTo<UserADT>Slice: expected []interface{}, got int64` | pass | **passes at base** — labelled a REGRESSION PIN, not fail-at-base (R1) |
| M2.4 | Generated-runtime behavioural test: `ConvertToInt64Slice(ArrayVal{int64(1),int64(2),int64(3)})` returns `[]int64{1,2,3}` — assert **length 3 AND element-wise equality**; and `ConvertTo<UserADT>Slice(ArrayVal{…})` returns the right elements | pass | fails at base (literal → nil; template → panic) |
| M2.5 | Generated-runtime behavioural test: `ConvertToInt64Slice(nil)` returns `nil` **without** panicking — proves the legitimate `v == nil` branch survived the edit | pass | passes at base; REGRESSION PIN |
| M2.6 | Emitter lint: over the `!ok` branches of `codegen_runtime_slices.go`, the count of `return nil` writes is **0** | 0 | 5 at base |
| M2.7 | `go test ./internal/gen/golang/ -count=1`; `PATH=… go test ./tests/golden/codegen/ -count=1`; `go vet ./internal/gen/...` | all rc=0 | all rc=0 |

**Kills which mutation.** M2.4's original doc form — *"input `ArrayVal{1,2,3}` → non-nil, len 3"* — is too coarse: a converter that did `make([]int64, len(src))` and never filled it returns a non-nil length-3 slice of zeros and passes. Element-wise equality is required, and is written into M2.4 above.
M2.2's observable is the **recovered panic string**, not "it panicked" — a boolean, or a comparison against an error enum, would be satisfied by *any* panic from anywhere in the converter, including an unrelated index-out-of-range. Only the exact converter-specific message is downstream of the specific branch M2 edits.
M2.6 is a source-text lint, adjacent to the mechanism. It is deliberately **secondary** to M2.2: a grep can be satisfied by moving the text, and the mission's recorded failure shape is *guard the helper, miss the call site*. M2.2/M2.3 are behavioural and run the generated code. Note also that M2.6, restricted to the two template loops, is **already 0 at base** — so the doc's AC4 grep, stated over *both* emitters, is partly vacuous and is written here over the one emitter where it can actually move.
M2.3 exercises a converter that appears **nowhere in the codegen source by name** (`grep -rn 'func ConvertToOptionSlice' internal/gen/golang/*.go` → 0 hits; control `grep -rn 'ConvertToInt64Slice' internal/gen/golang/*.go` → 3 hits), so it is the one test that proves the fix lives at the emitter and not in an enumeration.

---

### M3 — typed aggregate preservation (Days 3–4, ~2 days) · ~170 LOC (impl 105 / test 65)

**This is the milestone that closes the user-visible defect, and it holds all the sprint's risk.** Budgeted at 2 days, not the doc's 1, because of R2 and R4.

Measured repro at base (single probe, both boundaries):

```ailang
type Box = MkBox(Array[int])
type Holder = { items: Array[int] }
```
```go
return NewBoxMkBox(ConvertToInt64Slice(tmp5))     // ADT constructor path
return &Holder{Items: ConvertToInt64Slice(tmp3)}  // named-record path
```
interpreted `#[1, 2, 3]\n#[4, 5, 6]\n` vs compiled `[1, 2, 3]\n[4, 5, 6]\n`. This extends VL-16, which measured only the ADT path.

**Tasks**

1. `types.go:79` — `case *types.TArray:` returns `GoType("ArrayVal")`. Leave `case *types.TList:` (line 72) returning `[]%s` **untouched**.
2. `adt.go:382` — `case *ast.ArrayType:` returns `"ArrayVal"`. Leave `case *ast.ListType:` (320, 364) untouched.
3. **Decide the `adtSliceTypes` side-effect.** `adt.go:386-389` registers `g.adtSliceTypes[baseType] = true` for user-defined array element types, and that registration is what *causes* the template loop to emit `ConvertTo<T>Slice`. Removing it may delete a converter another (list) path still needs; keeping it emits a now-unused converter. Measure both ways with `go vet` (unused funcs are fine in Go, so vet will not tell you — use the golden `TestGoldenCompile` and a grep on generated output) and record the decision in the commit.
4. **Audit all 5 `HasPrefix(goType, "[]")` sites** (R4) — `adt.go:451`, `codegen_match_patterns.go:34`, `codegen_decl.go:323`, `codegen_record.go:171`, `codegen_record.go:401`. For each, record: does `"ArrayVal"` need to take this branch? `codegen_match_patterns.go:34` (`isTypedSlice`) and `codegen_record.go:171` (M-DX15 empty-literal) are the two most likely to bite.
5. **Do not** hand-edit the four converter call sites. R3: they all read `getSliceConversion(goType)` at `codegen.go:715`, which returns `""` for `"ArrayVal"` automatically. Verify that is what happened rather than assuming it — assert on generated output (M3.4), not on the source.
6. New unit tests for both mappers, **each with its list control in the same test** (an empty or negative result is a claim; a mutation that returns `"ArrayVal"` for everything passes an array-only assertion).

**Acceptance**

| # | Command | Required | Baseline |
|---|---|---|---|
| M3.1 | Add `tests/golden/codegen/show_differential_array_adt_field.ail` containing **both** `Box = MkBox(Array[int])` (construct, match-extract, `show`) **and** a named-record `Array[int]` field (construct, field-get, `show`); bump count 6 → **7**. `PATH=… go test ./tests/golden/codegen/ -run TestInterpreterCompiledDifferential -count=1 -v` | rc=0, **7** `--- PASS` | fails at base both ways — measured above |
| M3.2 | `make verify-examples` | rc=0, `0 failed`, manifest in sync | rc=0, `211 passed, 0 failed, 6 skipped`. **The R2 gate**: `examples/runnable/array_adt.ail` is `PatternPatrol(Array[Direction])` and would panic at runtime if an `ArrayVal` reached `ConvertToDirSlice` without M2 task 3 |
| M3.3 | Unit test: `mapTypeWithVisited(&types.TArray{Element: TInt})` → `"ArrayVal"`; **control in the same test**: `mapTypeWithVisited(&types.TList{Element: TInt})` → `"[]int64"`. Same pair for `adt.mapASTType` over `*ast.ArrayType` / `*ast.ListType` | pass | fails at base (both return `[]int64`); **no such test exists today** — `grep -rn 'TArray\|ArrayType' internal/gen/golang/*_test.go` → 0 hits, control `grep -rln 'Array' internal/gen/golang/*_test.go` → 1 file |
| M3.4 | Generated-output test over the M3.1 program: emitted source contains `ArrayVal` in the `Box` constructor signature **and** the `Holder` field declaration, and contains **neither** `NewBoxMkBox(ConvertToInt64Slice(` **nor** `Items: ConvertToInt64Slice(`. Non-vacuity: the same test asserts a `[int]` **list** field still *does* emit `ConvertToInt64Slice` | pass | fails at base — both converter calls present verbatim |
| M3.5 | Third element-type case: unit or golden coverage for `Array[UserADT]` (`Array[Dir]`) — generated code no longer calls `ConvertToDirSlice` on it, and the compiled program prints `#[North, South]` matching the interpreter | pass | fails at base — measured `NewPlanMkPlan(ConvertToDirSlice(tmp2))`, compiled prints `[North, South]` |
| M3.6 | `go test ./internal/gen/golang/ -count=1`; `PATH=… go test ./tests/golden/codegen/ -count=1`; `go build ./internal/gen/... ./cmd/ailang`; `go vet ./internal/gen/...`; `make check-file-sizes` | all rc=0 | all rc=0 |

**Kills which mutation.** M3.1 and M3.2 are process-level: real binaries, real stdout, real example suite. M3.4 is a source-text assertion and therefore *adjacent*, so it carries its own list-field non-vacuity control — without it, a mutation that disabled `getSliceConversion` **entirely** (breaking every list conversion) would pass the "contains no `ConvertToInt64Slice(`" clause. That control is the whole value of the row. M3.3's list control does the same job one layer up: it is the only thing separating "arrays map to `ArrayVal`" from "everything maps to `ArrayVal`".
M3.5 exists because the doc's element-type list (primitive / record / ADT) is right but its risk ordering is not: the ADT-element case is the one that reaches a **panicking** converter (R2), so it is the case that fails loudest and latest.

---

### M4 — docs, changelog, adjacent-defect capture (Day 4, ~0.5 day) · ~60 LOC (docs only)

Both fixtures and the 5 → 7 count bump already landed in M1/M3 (§3). M4 is the write-up.

**Tasks**

1. `changelogs/v0.18-current.md` — extend the existing `## [Unreleased]` → *"`show` is deterministic and agrees across runtimes"* section from PR #822 rather than opening a new one; this sprint closes the last member of that class. State that generated array identity is preserved across dynamic, ADT-field and record-field representations, and that the 5 primitive slice converters now fail loudly instead of returning `nil`.
2. Move the design doc `design_docs/planned/` → `design_docs/implemented/v0_34/` per `.claude/rules/coding-standards.md`, and correct VL-9 in place (R1) so the next reader does not inherit "7 silent fallbacks".
3. Record the two adjacent defects found while planning, as backlog rows — **not** fixed here:
   - `std/array.empty` / `std/array.append` emit undefined `Empty` / `Append` (R5). Include the reproducer.
   - `toSlice`'s silent `return nil` (R6), and the fact that the round-2 quorum's stated universal goal is not met by M2 alone.
   - (The doc already records the `std/prelude` diagnostic defect, VL-17.)
4. No example-output pins change — `make verify-examples` is green at base and stays green; option (B) moves compiled output *toward* what `docs/docs/reference/arrays.md` already shows (33 `#[` occurrences).

**Acceptance**

| # | Command | Required |
|---|---|---|
| M4.1 | Full gate sweep: `go build ./internal/gen/... ./cmd/ailang` · `go vet ./internal/gen/... ./tests/golden/...` · `go test ./internal/gen/golang/ -count=1` · `PATH=… go test ./tests/golden/codegen/ -count=1` · `make verify-examples` · `make check-file-sizes` | all rc=0 |
| M4.2 | `sed -n '14p' tests/golden/codegen/golden_test.go` | `const expectedDifferentialFixtureCount = 7` |
| M4.3 | V1 repro byte-match: the doc's Framing program, interpreted vs compiled, both `#[1, 2, 3]\n[1, 2, 3]\n` | match (fails at base — measured compiled `[1, 2, 3]\n[1, 2, 3]\n`) |
| M4.4 | Both adjacent defects filed with reproducers; VL-9 corrected in the design doc | done |

**Not gated by `go build ./...`** — rc=1 at base (B1). Any criterion using it would be broken on arrival.

---

## 5. Estimate — is 3–4 days right?

**Keep 4 days. Re-allocate: M1 0.5 · M2 1.0 · M3 2.0 · M4 0.5.**

The design doc's per-milestone shape (1/1/1/0.5) is wrong even though its total is defensible.

- **Downward pressure.** Iteration 247 (PR #822) is the near-exact analogue — same package, same harness, same `showValue`, and it created all 5 existing differential fixtures. It landed **482 insertions / 46 deletions across 17 files in a single mission iteration**. The `Tuple` mechanism this sprint copies is already in the tree at 28 non-test sites. R3 further *removes* budgeted work: converter selection is one chokepoint, not four call sites.
- **Upward pressure, and it wins.** R2 (template converters panic, so `Array[UserADT]` is a crash path and `make verify-examples` becomes a real gate), R4 (5 unbudgeted `HasPrefix "[]"` sites), M3 task 3 (the `adtSliceTypes` registration has no obviously-right answer and must be measured both ways), and the fact that the array type mappers have **zero** existing unit coverage — so M3 is being done without a safety net and has to build one first.
- Net: M3 is a 2-day milestone that the doc budgets at 1. Total stays 4.

**Do not compress to 3 days by shortening M3.** If time pressure appears, the honest cut is to ship M1+M2 (the dynamic path, fully gated, user-visible improvement) and re-scope M3+M4 as a follow-up — not to thin M3's tests, which are the only coverage the type mappers have ever had.

## 6. Dependencies and ordering

`M1 → M2 → M3 → M4`, strictly serial, **not parallelisable**. R3 is the reason: M3's correctness rests on the fall-through type assertion `expr.(ArrayVal)` succeeding, which requires every array producer to already yield `ArrayVal` — literals (M1) and helper return types (M2). Running M3 first produces a tree that compiles and panics at runtime.

## 7. Out of scope (unchanged from the design doc)

- Any interpreter change. `#[…]` is the reference behaviour the compiled backend is being made to match.
- `Eq`/ordering for arrays — a type error at HEAD, and lists lack `Eq` identically (VL-10b), so it is not an array-specific gap.
- `toSlice` (R6), `std/array.empty`/`append` (R5), and the `std/prelude` diagnostic (VL-17) — all recorded as backlog rows in M4, none fixed here.
