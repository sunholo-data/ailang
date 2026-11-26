# M-VERIFY-COVERAGE-TAIL: callee instances, the builtin tail, and the undecided rows

**Status**: Planned. Written 2026-11-26 as the handoff of [M-VERIFY-SOUNDNESS-COVERAGE](../v0_61_0/m-verify-soundness-coverage-program.md) item 5. Every lever that program ran without a design is done (items 1–4f); what is left is below.
**Target**: v0.64.x
**Priority**: P1 (T1, T2 and T4 are coverage; T1 also retires the S7 gate, a soundness guard)
**Estimated**: T0 0.25d · T1 3–4d · T2 1.5–2d · T3 1.5d · T4 1d · T5 1d (spike) · T6 2–4d (after its ruling)
**Dependencies**: none to start T0–T5. T6 waits for ruling D-T1. Coordinates with [m-v1-nominal-type-identity](../v1_0_0/m-v1-nominal-type-identity.md): that doc keys SMT *types* (`adtTypes`/`recordAliases`) by module; this one keys SMT *callees* by module. Both use `module::name` identity, and neither should introduce a second notation.
**Planner-Lane**: opus-required for T1 (it changes how every call is resolved); T3–T5 are agent-resolvable.

**Quorum trigger:** #1 fires (freeze item D-T1, for T6 only) and #2 fires (T1 replaces the S7 bare-name gate, which is shared machinery). Run `ailang design-quorum` on this doc before T1 is planned. T0, T3, T4 and T5 have no freeze item and may start before the quorum.

## How to pick this up

1. Read the program doc's §Progress and items 4b–4f. They record what VERIFIED covers today and the encoder bugs earlier rounds found.
2. Re-measure (T0) before writing code. The numbers below come from the gap-map snapshot of 2026-11-26 (`2acbd253e`). Downstream corpora drift.
3. Working practice from the program, all of it still in force:
   - every lever goes in one commit with its fixture, CLI pin, changelog fragment and program-doc row;
   - mutation-test every new branch on a scratchpad copy, never with `git checkout <file>`;
   - run the gates from a fresh worktree;
   - read `go test ./cmd/ailang/` output by grepping `^(ok|FAIL|--- FAIL)`, never with `tail`;
   - re-measure the gap map against the last snapshot and accept **0 verdict regressions**.
4. The rulings still in force are D1 (a), D2 (a′), D4 (b′), rowless admission, and opaque helpers with a caveat (a refutation through an opaque helper is inconclusive).

## Problem Statement

The gap map of 2026-11-26 has 650 verified, 58 skipped, 21 inconclusive and 16 counterexamples. Against the program's M0 baseline: verified 590 → 649, skipped 125 → 58, errors 1 → 0. One extra verified row comes from another session's `reexport_facade` example. All 16 counterexamples are intended: example demos and one world fixture.

The 58 skips group into eight levers. In the table, "Any" counts the skips where the lever's blocker appears, and "Sole" counts those where it is the only blocker. The counts are recomputed from each row's `blockers` field in the snapshot.

| Lever | Any | Sole | Rows (sole in bold) |
|---|---|---|---|
| Effects (program non-goal) | 17 | 12 | `generate{Docx,Pptx,Xlsx}`, `layout_ai` ×5, docparse key/auth generators ×5, … |
| Solver undecided: facts about an uninterpreted builtin | 17 | 17 | `renderA2UI` ×2, `questionsToJsonSchema` (`$json_encode`); `parseEml`/`parseMbox`/`parseMarkdown`/`parseLatex` (`$list_reverse`, string UFs); `indexedColour`, `normaliseHex`, `emlStripQuotedText`, `mdJoinLines`, `texStripComments`, `parseLatexMetadata`, `pdfAiPagesCountLines`, `extractSignatureParts`, `docxTplExtractSectPr`, `docxTplSectPrAt` |
| Generic callee signature (type variable) | 10 | 4 | **`decodeEvent`, `validateRun`, `walk`** (`std/result.isErr`), **`gradeNumeric`** (the `unencodable_callee_skip.ail` demo); `evalJaccard`, `parseRtf`, `workbookColIndex`, `workbookRefRow`, `enhanceBlocks`, `getPptxGridSpan` |
| The bare-name gate (S7) | 9 | 3 | **`parsePptxCells`, `parsePptxRows`, `extractSharedStrings`** (`std/option.map` vs `std/list.map`); `getPptxGridSpan`, `parseXlsxCells`, `inferTableStructure`, `generateOd{p,s,t}` |
| Solver undecided: through bounded unrolling | 6 | 6 | `emlUnique`, `apiKeyParseTimestamp`, `referralParseEpoch`, `editApply_*` ×3 |
| Builtins: `foldChars` 3, `show` in a contract 2, `intersect` 1, `mapE` 1 | 7 | 2 | **world `showInt` ×2**; `parseRtf`, `workbookColIndex`, `workbookRefRow`, `evalJaccard`, `enhanceBlocks` |
| Encoder bugs, one row each | 3 | 3 | **`docxUsableWidth`** (ADT `XmlNode` in an opaque helper's signature not declared), **`parseMboxThreaded`** (`Record_content_info` not kept for a map capture), **`parseCsv`** (`map(simpleCell, …)` inside an unrolled body) |
| `Map` in a callee signature | 1 | 0 | `parseXlsxCells` |

The 21 **inconclusive** rows are all bounded-unrolling exhaustion: the model may depend on the unrolling frontier. Replay of their witnesses either hit the contract holding or was not convertible. They need induction (T6) or a stronger fact (T5).

Three things block the generic, higher-order and bare-name rows. They are one problem.

1. **Callees are keyed by bare name** (V4). The define-fun symbol, `SurfaceParams`/`SurfaceReturnSorts`, the sort gate's AST lookup and the body lookup `findFuncBodyInAnyProg` all use the bare name, first match wins. The S7 gate refuses the calls this would bind wrongly. That turns `std/option.map` into a skip because `std/list` also has a `map`.
   - The bare key also corrupts the sort gate's diagnosis (V5). A probe calling `O.map(\v. v + 1, o)` is skipped as "calls `map` whose signature uses an unencodable type `list[b]`". `std/option.map`'s signature has no list in it: the gate read `std/list.map`'s declaration.
2. **A callee has one SMT definition**, so a type variable has nowhere to go (V2). `MapType(TVar)` is an error by design ("needs monomorphization"), and nothing in `internal/smt` instantiates (V15).
3. **A function-typed parameter is rejected** (V11). Literal-lambda calls to `std/list`'s map/filter/foldl/any/flatMap/findIndex are specialised. A user or stdlib callee that *takes* a function is not, even when its body is a single `match` (`std/option.map`, V6).

## Goals

1. **T1.** A call to a generic callee is encoded as an *instance* of that callee at the call's type arguments, keyed by `(module, name, type arguments)`. The bare-name gate is retired because every lookup is qualified. S7's two-module probe gives VIOLATION on 6 of 6 runs.
2. **T2.** A call that passes a literal lambda or a named function to a callee's function-typed parameter is encoded as an instance specialised on that argument. `std/option.map`/`flatMap`/`filter` and `std/result.map`/`mapErr`/`flatMap` need no stdlib-specific code.
3. **T3.** `foldChars`, `intersect` and `show` inside a contract are encoded soundly.
4. **T4.** The three one-row encoder bugs are reproduced in fixtures and fixed.
5. **T5.** Every "solver could not decide" skip and every inconclusive row is classified by cause. No code is written for this before the table exists.
6. **T6.** Induction for self-recursive functions over lists, once D-T1 is ruled.
7. Throughout: 0 verdict regressions per lever; every remaining skip names its blocker; every new branch has a test that kills its mutation.

## High-Impact Decisions

| Decision | Options | Who decides | Change cost |
|---|---|---|---|
| D-T1: what counts as an induction hypothesis (T6) | (a) structural recursion on a list parameter, recognised syntactically (the self-call's argument is the tail bound by a `_ :: rest` pattern on that parameter); (b) a `decreases` clause, which is new syntax; (c) no induction: inconclusive plus replay stays the answer | **Mark**: it changes what VERIFIED means for recursion | (a) is local to `internal/smt`; (b) touches parser, typechecker and docs (A8); (c) costs nothing |
| D-T2: symbol for an instance | bare `name` when the instance is monomorphic and the name is unique among loaded modules, else a mangled `name@module<S1,…>` | Agent (deferred, constraints below) | Internal |
| D-T3: the S7 gate after T1 | delete it, or keep it as a test-only invariant | Agent: delete it, and keep its tests as behaviour tests (coding standards: remove out-of-date code) | Internal |

### Design Freeze

- [ ] D-T1 ruled by Mark (blocks T6 only)
- [x] D-T2 and D-T3 are agent-resolvable within the constraints in §Solution Design

## Deferred Decisions

- The mangling alphabet for instance symbols. It must be deterministic, collision-free against every existing prefix (§Conflict Surface), and readable in a skip reason. `$list_reverse<Seq.String>` is precedent.
- Whether a recursive callee that passes its function argument to itself unchanged is specialised in T2 or left for later.
- Which facts T3 states for `intersect`. Each fact must hold for every input of the Go implementation (`_list_intersect`) and must be cited in its comment.

## Solution Design

### Overview

One concept replaces three mechanisms: a **callee instance**.

```
instance = (module, name, [type argument sorts], [function arguments])
```

Every place that today looks a callee up by bare name, emits its define-fun, gates its signature or emits its `$spec_`/`$ob_` symbols takes an instance instead. Some of those definitions are monomorphic and have unique names: they must produce byte-identical scripts. This is what keeps the regression surface at zero.

### T0: re-measure (0.25d)

- Build a fresh `ailang`, then run `make verify-gap-map AILANG=<that binary>`. It diffs against the last banked snapshot in `~/.ailang/state/verify-gaps/` and exits 2 on a regression. Snapshots stay out of the repo because three of the corpora are private.
- Recompute the lever table above from the `blockers` field (the script is in §Testing Strategy).
- Update the counts in this doc if they drifted.

### T1: callee instances: qualified identity and type instantiation (3–4d)

**Instantiation from checked types.** No new inference is needed. The type checker already records the instantiated type at the call head and the generic type inside the callee body (V1):

```
caller:  App            : bool
           Var isErr    : Result[int, string] -> bool     ← instantiated
isErr:   Lambda         : Result[α8, α9] -> bool          ← generic
```

For a call `f(args)` where `f` resolves to `(module, name)`:

1. Read the head's checked type in the caller's program TI. Apply the caller's own instance substitution first, so a generic callee calling a generic callee works.
2. Read the callee's generic Lambda type in the callee program's TI (`programTI[calleeProg]`, which the resolver already switches to, V17).
3. Match the two structurally. A type variable on the callee side binds to the concrete type on the caller side. Any mismatch, or a variable left unbound, is a skip that names the variable. It never falls back to a default.
4. `MapType` each bound type. A type with no sort (function, `bytes`, `Map`) is a skip naming it, as the sort gate does today.

**Encoding a body under a substitution.** The parametric context gains one field, the active substitution. Every read of checked types goes through one accessor that applies the substitution before `MapType`. Today there are six readers (V3):
- `parametric_context.go:62`
- `parametric_patterns.go:29`
- `tuple_sorts.go:162`
- `resolve_sort.go:48`
- `hof_inline.go:667`
- `opaque_helpers.go:175`

They become calls to the accessor, and the `ti` field becomes private to it. That is the seam: a reader that bypasses the accessor sees `α8` and fails loudly through `MapType`, never with a wrong sort. The accessor covers:
- operator lowering inside a constrained generic body, where `bigger[a]`'s `x > y` is an `Intrinsic` over `a@{bigger}` (V13);
- constructor annotation;
- empty-list sorts;
- record and tuple sorts.

**Signatures from the instance, not the AST.** An instance's parameter and return sorts come from the instantiated head type. A monomorphic callee's head type equals its declaration, so its sorts do not change. This removes the bare-name `allSurfaceParams`/`allSurfaceReturnSorts` merge (`verify.go:237-280`, first-wins over sorted modules) for every resolved call.

`astTypeEncodable`'s `TypeVar` rejection moves from the AST signature to the instantiated one. A type variable never reaches the gate after instantiation.

**Qualified lookup.** The body of a call whose head is a `VarGlobal` is looked up as `findFuncBody(programForModule(ref.Module), ref.Name)`. A plain `Var` head is looked up in the program whose body contains it, as the identity walker already does (`callee_identity.go:122-137`). `findFuncBodyInAnyProg` remains only where no module is known, if anywhere.

The S7 gate then cannot fire, so it is deleted (D-T3). `TestCalleeIdentityNeverBindsAnotherModulesFunction` is kept and must still pass. Its two-module probe must give VIOLATION on every run.

**Everything keyed by callee, per instance:**
- the define-fun;
- the bounded unrolling of a recursive callee (`unrollRecursiveCallee`, one chain per instance);
- the mutual-recursion grouping;
- the opaque helper's `$spec_` declaration and its `helpers_not_checked` caveat, which names the instance as `isErr[int, string]`;
- the contract helper's `$spec_` constant;
- the `$ob_<name>@<def>` obligation define-funs;
- the call-facts channel;
- `activeContractSpecs`/`activeContractCallees`.

**Skip reasons.** A skip names the instance (`isErr[int, string]`), and the sort gate names the type the call's instantiation produced.

### T2: specialisation on function arguments (1.5–2d)

The instance key gains the function arguments: each literal lambda or named top-level function passed for a function-typed parameter. `hofFunctionArg` (`hof_inline.go:679`) already accepts both. The callee body is encoded with the parameter replaced by the lambda: each `f(x)` becomes the lambda body with `x` bound, under `$l_` hygiene. Free variables of the lambda become captures, with `$c_` hygiene (`hof_captures.go`). A specialised instance is non-recursive in T2, and a recursive one is a skip that says so (deferred decision).

`std/option.map`'s body is a two-arm `match` (V6). Once T1 and T2 land, it is an ordinary instance (`map@std/option<Int,Int>[λ1]`), with no entry in `ResolveStdlibToBuiltin`.
- **Rejected alternative:** direct encodings for the six Option/Result HOFs. That special-cases what is one shape, the same mistake the program avoided for list HOFs (program §M2). User helpers with a function parameter would still skip.
- **Rule change:** `hasHigherOrder && !AllHigherOrderIsInlinable` (`encodable.go:75`, `callee_resolver.go:572`) admits a call whose function arguments specialise.

### T3: the builtin tail (1.5d)

- **`std/string.foldChars(f, acc, s)`** is `_str_foldChars` (V8). Encode it as `foldl(f, acc, chars(s))` through the existing foldl chain over the existing `$str_chars` UF, with T1 supplying the accumulator sort. Expect inconclusive more often than verified: `chars` is uninterpreted and the chain is bounded. This needs T1.
- **`std/list.intersect`** is `_list_intersect`. It becomes a per-sort UF with facts true of the Go implementation: each result element is in both inputs, and the result is no longer than the first input. Each fact is checked against the implementation and cited. A model through it is inconclusive, and replay runs the real builtin (the opaque-builtin precedent, program item 2).
- **`show` in a contract.** `ShowNormalizer` rewrites `show` in function bodies only. It never visits `DeclMeta.Contracts` (V9). Run the same rewrite over contract expressions, so the encodable argument types (string, bool, int) encode there too. This is a **downstream behaviour change**: the two rows are world fixtures whose comments say they exist to pin "interpolation in ensures is unencodable" (V10). Tell the world owners before landing. Their evidence tests will see VERIFIED when world next moves its ailang pin. World holds that pin back deliberately, so the change lands there on world's schedule, not this one.
- **`mapE`:** effectful (`std/list.ail:270`), so it is out of scope.
- **`Map`:** no sort, so it is out of scope.

### T4: three encoder bugs (1d)

Each one is a latent bug of a kind the program has already fixed elsewhere:
1. An ADT in an opaque or contract helper's signature is not declared (`docxUsableWidth`: `XmlNode` in `$spec_findFirst`). Item 4c did the same for records only.
2. An inline record reached only through a `map` specialisation's capture constant is dropped by the sort filter (`parseMboxThreaded`: `Record_content_info`).
3. A named-function `map` inside an unrolled helper body is not specialised (`parseCsv`: `std/list.map(simpleCell, …)` at depth 1).

For each one:
- reduce it to a fixture in `internal/smt/testdata/`;
- fix it;
- mutation-test the fix;
- re-measure the gap map.

### T5: the undecided tail, classified first (1d spike)

This covers the 23 "solver could not decide" skips and the 21 inconclusive rows. For each row, record:
- `timeout` or `unknown`;
- the verdict at 4× the timeout;
- which builtin facts or unrolling the contract depends on (read the contract);
- the replay outcome, where there is a model.

Then route each row to exactly one of:
- **a stronger sound fact:** derived from the Go implementation, and cited;
- **a budget change:** only if the 4× run decides it and the solve time is reported;
- **T6:** it needs induction;
- **"contract is false":** report it to the owning repo, as `renderA2UI`'s `length(result) > 2` already is for `[]`;
- **"stays undecided":** the skip reason already says why.

The table goes into this doc. No fact is added without a row that needs it and a fixture where it decides a contract.

### T6: induction (after D-T1; 2–4d)

The work depends on the ruling.
- **Under D-T1 (a):** for a self-recursive verified function whose recursive calls all pass the tail of a list parameter matched by `_ :: rest`, prove the base arm, then the step arm assuming the `ensures` at each self-call. Sound only for that syntactic shape, because the list strictly shrinks: the recursion terminates and the hypothesis is well-founded. Any other shape stays bounded.
- **Under D-T1 (b):** design the clause first, in its own doc.
- **Under D-T1 (c):** nothing is built, and the rows stay inconclusive.

Helpers' map/filter/foldl frontiers over unbounded lists are a second step under (a) and are not promised here.

### Files to Modify/Create

| File | Change |
|---|---|
| `internal/smt/callee_instance.go` (new) | the instance key, the head-vs-generic matcher, the symbol mangler |
| `internal/smt/parametric_context.go` | the substitution field and the one checked-type accessor; `ti` private |
| `parametric_patterns.go`, `tuple_sorts.go`, `resolve_sort.go`, `hof_inline.go`, `opaque_helpers.go` | read checked types through the accessor |
| `internal/smt/callee_resolver.go`, `callee_collect.go` | resolve calls to instances; qualified body lookup; signatures from the instance |
| `internal/smt/verify.go` | drop the bare-name surface-param merge for resolved calls |
| `internal/smt/callee_identity.go`, `callee_gates.go` | delete the S7 gate (D-T3); the sort gate checks instantiated signatures |
| `internal/smt/hof_specialize.go`, `hof_captures.go`, `encodable.go` | T2 specialisation and the HOF rule |
| `internal/smt/codegen_string_extra.go`, `codegen_opaque_builtins.go` | T3 `foldChars`, `intersect` |
| `internal/pipeline/show_normalize.go` | T3: rewrite contract expressions |
| `examples/runnable/contracts/unencodable_callee_skip.ail` | T1 makes `gradeNumeric` verify. Move the demo to a shape that stays unencodable (a `bytes` or function-typed result) and update its header, manifest row and CLI pin |
| `internal/smt/testdata/generic/`, `testdata/hof_callees/` (new) | fixtures |
| `cmd/ailang/verify_*_e2e_test.go` | CLI pins: a generic instance, `option.map`, S7 probe still VIOLATION |
| `docs/docs/guides/contracts.mdx` | supported callee shapes |

Several of these files are close to the 800-line limit: `verify.go` 745, `codegen.go` 749, `encodable.go` 754, `hof_inline.go` 755. Put the new logic in new files.

## Conflict Surface

T1 and T2 add symbols to one namespace, the SMT script. Every existing prefix must stay unambiguous:

| Prefix / shape | Owner |
|---|---|
| bare `name`, `name_0…name_N` | define-funs and unroll levels |
| `$spec_name` | opaque or contract helpers |
| `$ob_<name>@<def>` | obligations |
| `_map_spec_<fn>_<n>`, `$g` suffixes | HOF chains |
| `$list_reverse<Seq.X>`, `$json_encode`, `$str_*` | builtin UFs |
| `$p_`, `$a_`, `$l_`, `$c_` | parameters, helper arguments, lets, captures |
| `mk_TupleN`, `TupleN`, `Tuple0` | tuple datatypes |
| `Record_*`, `Record_*__S1__S2` | record sorts |

**Disambiguation rule:**
- An instance symbol is the bare name only when the callee is monomorphic, has no function arguments and its name is unique among the loaded modules.
- Otherwise it is mangled with `@` and `<…>`. Neither character appears in any AILANG identifier or in the prefixes above, except `@` inside `$ob_`, which always starts with `$`.
- A test enumerates the prefixes and asserts that no instance symbol collides with any of them.

**Programs that MUST still work** (each exists today):
- `internal/smt/testdata/tuples/tuples.ail`
- `testdata/records/same_fields.ail`
- `testdata/tuples/unit.ail` (zero-argument unrolling)
- `examples/runnable/contracts/shapes_verify.ail`
- `callee_identity_test.go`'s two-module probe (VIOLATION, every run)
- the full gap map, with 0 verdict regressions

**What deliberately changes:**
- `gradeNumeric` verifies, and the demo moves (above).
- S7 skips disappear. Their rows reach their next blocker.
- World's two `showInt` fixtures verify.

## Examples

```ailang
-- T1: std/result.isErr at Result[int, string] → instance isErr@std/result<Int,String>
export func useIsErr(r: Result[int, string]) -> bool
  ensures { result == (match r { Ok(_) => false, Err(_) => true }) }
{ isErr(r) }

-- T2: std/option.map specialised on the lambda
export func useOptMap(o: Option[int]) -> Option[int]
  ensures { O.isSome(result) == O.isSome(o) }
{ O.map(\v. v + 1, o) }
```

Both are skipped on `2acbd253e` (V5, V7), and both must verify after T1 and T2.

## Success Criteria

- [ ] T1: `useIsErr`, `useFirst` (`firstOr[a]`) and `useBigger` (`bigger[a]` with `>`) verify. `decodeEvent`, `validateRun` and `walk` reach a verdict or their next blocker.
- [ ] T1: the S7 probe gives VIOLATION on 6 of 6 runs. The S7 gate's code is gone.
- [ ] T1: verified rows produce byte-identical SMT for monomorphic unique-name callees. Check this with a script diff over `examples/runnable/contracts/*.ail` before and after.
- [ ] T2: `useOptMap`, `parsePptxCells`, `parsePptxRows` and `extractSharedStrings` reach a verdict or a named next blocker.
- [ ] T3: `foldChars`, `intersect` and contract-`show` encode. World owners are told about the fixture flip before it lands.
- [ ] T4: three fixtures, three fixes, three rows moved.
- [ ] T5: the classification table is in this doc, with one route per row.
- [ ] T6: built under the ruling, or explicitly not built under (c).
- [ ] Every lever: 0 verdict regressions in the gap map, mutation-tested, and full gates green (gate output grepped, not tailed).

## Testing Strategy

- **Unit:**
  - the matcher: concrete-vs-generic pairs including nested `Result[[a], e]`, tuples and records; an unbound variable; a mismatch;
  - mangler determinism and the prefix-collision test;
  - the accessor applies the substitution at each of the six readers. Write one fixture per reader shape (operator, constructor, empty list, record literal, tuple, opaque return).
- **Fixtures:** `testdata/generic/` (isErr, firstOr, bigger, a generic recursive `len[a]`, a generic callee calling a generic callee) and `testdata/hof_callees/` (option.map, result.mapErr, a user `applyTwice(f, x)`, a capture).
- **Mutation:** for each new branch, break it in a scratch worktree or copy, never the shared checkout. The program used a short script per lever: replace one string in the source, run the named tests, record killed or survived, restore. A mutation that survives means either the test is missing or the code is redundant; the program deleted redundant code in four such cases.
- **Gap map:** after each lever, run `make verify-gap-map` against the previous snapshot. The lever table is recomputed by grouping each skipped row's `blockers` strings (effects / bare name / type variable / higher-order / solver-undecided split by "bounded unrolling" / builtin / other).

## Non-Goals

- **Effectful functions** (17 rows): a program non-goal (`contracts.mdx`: effects break proofs).
- **A `Map` sort.**
- **Lean or another proof assistant:** program §Why not Lean.
- **New syntax.** That includes a `decreases` clause unless D-T1 picks (b), which then gets its own doc.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| A checked-type reader bypasses the substitution and encodes `α8` | `ti` becomes private behind the accessor; `MapType(TVar)` already fails loudly; one fixture per reader |
| Instance blow-up (many instantiations of one helper) | Instances are created per call site actually reached. Report the count per function in `--json`, and measure the maximum over the corpus in T1 |
| Retiring S7 reintroduces a misbinding | The probe test stays, and every lookup is qualified by construction. Grep that no `VarGlobal` head reaches `findFuncBodyInAnyProg` |
| T3 `show` flips world's pinned fixtures | Tell world before landing. World's corpus is advisory in the gap map |
| T6 (a) misses a non-structural recursion that looks structural | Recognise only the exact shape. Anything else stays bounded, and the shape tests include near-misses (the tail of a different list, a rebuilt list) |

## Verification Log

All checks ran on `2acbd253e` with Z3 4.16.0. The probes were throwaway, but each can be rebuilt from what is quoted here.
- **The type probe (V1, V13)** compiles a source with `pipeline.Run(pipeline.Config{DryLink: true, RelaxModules: true}, …)` and prints `Artifacts.CoreTI.GetForExpr` for every Core node of each declaration.
- **The verify probe (V5, V7, V9)** is the two Examples functions plus `firstOr`, `bigger` and `showNeg`, run with `AILANG_RELAX_MODULES=1 ailang verify`.

| # | Claim | Evidence |
|---|---|---|
| V1 | The call head carries the instantiated type; the generic body carries type variables | Probe test over `pipeline.Run` + `Artifacts.CoreTI`: `caller`'s head `Var id=22 type=Result[int, string] -> bool`; `isErr`'s `Lambda id=5 type=Result[α8, α9] -> bool`; `firstOr`'s `Lambda type=(list[a@{firstOr}], a@{firstOr}) -> a@{firstOr}`; `useFirst`'s head `(list[int], int) -> int` |
| V2 | `MapType` rejects type variables | `internal/smt/types.go:42-45` ("needs monomorphization") |
| V3 | The six checked-type readers | `grep -n "\.ti\b\|\.ti\.\|programTI" internal/smt/*.go` (non-test): `parametric_context.go:62`, `parametric_patterns.go:29`, `tuple_sorts.go:162`, `resolve_sort.go:48-49`, `hof_inline.go:664-667`, `opaque_helpers.go:171-175`, plus the resolver's switch at `callee_resolver.go:204-263` |
| V4 | Surface params and return sorts are keyed by bare name, first-wins | `internal/smt/verify.go:237-280` (`if _, exists := allSurfaceParams[fd.Name]; exists { continue }`); `findFuncBodyInAnyProg` at `callee_resolver.go:312-328` |
| V5 | The bare key corrupts the sort gate's diagnosis | `ailang verify` of the probe's `useOptMap` (`O.map(\v. v + 1, o)`): skip reasons "uses higher-order functions; … calls std/option.map, but another loaded module (std/list) also defines "map" …; … calls "map" whose signature uses an unencodable type "list[b]"". `std/option.ail:13` is `map[a, b](f: a -> b, opt: Option[a]) -> Option[b]`, which has no list |
| V6 | `std/option.map`/`flatMap` are single matches | `std/option.ail:13-25` |
| V7 | Named generic blockers | `std/result.ail:43` `isErr[a, e]`; `examples/runnable/contracts/unencodable_callee_skip.ail` `firstOr[a]`; probe skips `useIsErr` ("`Result[a, e]`"), `useFirst` ("`a`"), `useBigger` ("`a`") |
| V8 | Builtin definitions | `std/string.ail:159` `foldChars(f: (a, string) -> a, acc: a, s: string) -> a = _str_foldChars(…)`; `std/list.ail:395` `intersect[a] = _list_intersect`; `std/list.ail:270` `mapE` is `! {e}` |
| V9 | `ShowNormalizer` does not visit contracts; `show` in a body encodes | `grep -c Contract internal/pipeline/show_normalize.go` = 0; `Normalize` walks `prog.Decls` only (`:80-100`); contracts live on `DeclMeta.Contracts` (`internal/core/core.go:458`). Probe `showNeg(x) = show(x)` with `ensures { S.length(result) > 0 }`: VERIFIED. `ailang run` prints `show(-3)` as `-3` |
| V10 | World's `showInt` rows are deliberate fixtures | `ailang-world/host/evidence/testdata/aicheck/encerror.ail`: "Scratch fixture (row 26 design evidence): interpolation in ensures is unencodable"; `mixed.ail` likewise |
| V11 | Function-typed parameters are rejected unless the call is a std/list HOF | `encodable.go:75-81`, `callee_resolver.go:572-579`; `matchHOFCall` matches only `$builtin` and `std/list` heads (`hof_inline.go:145-172`) |
| V12 | The S7 probe test exists | `internal/smt/callee_identity_test.go:32` `TestCalleeIdentityNeverBindsAnotherModulesFunction` |
| V13 | A constrained generic body has no dictionary abstraction; `>` is an Intrinsic over the rigid variable | Probe: `bigger`'s `Lambda type=(a@{bigger}, a@{bigger}) -> a@{bigger}` → `Let` → `Intrinsic id=10 type=bool`. The concrete caller's `+` in a lambda is a `DictApp` at `int`, which the encoder already handles |
| V14 | Gap-map counts | Snapshot `2026-11-26T130740Z` totals: verified 650, skipped 58, inconclusive 21, counterexample 16, skip_mp 28, uncontracted 877; lever table recomputed from `blockers` |
| V15 | Nothing in `internal/smt` instantiates type variables today | `grep -rn "monomorph\|Instantiate" internal/smt/*.go` (non-test): only comments (`types.go:44-45`, `:457`, `from_ast.go:287-321`, `codegen_seq_hint.go:11`, `verify.go:306`) |
| V16 | `hofFunctionArg` accepts a literal lambda or a named function | `internal/smt/hof_inline.go:679` (eta-expansion, program M5 row) |
| V17 | The resolver encodes a callee body under the callee program's TI | `callee_resolver.go:201-205` (`activeADTContext.ti = activeADTContext.programTI[calleeProg]`) |

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | Qualified lookup removes the last first-wins bare-name merge from verdict paths |
| A2: Replayability | 0 | Replay is unchanged; instances replay through the verified function |
| A3: Effect Legibility | 0 | No effect change; effectful functions stay out |
| A4: Explicit Authority | 0 | — |
| A5: Bounded Verification | +1 | Generic and higher-order helpers are checked, not skipped; T6 turns bounded into inductive where it is sound |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | Skip reasons name the instance and the real type; the misleading `list[b]` diagnosis goes |
| A8: Minimal Syntax | 0 | No syntax, unless D-T1 picks (b), which then needs its own doc |
| A9: Cost Visibility | 0 | Instance count reported in `--json` |
| A10: Composability | +1 | Stdlib generic helpers compose into verified code without special cases |
| A11: Structured Failure | 0 | Existing reason codes |
| A12: System Boundary | 0 | — |

**Net Score: +4.** No −1 on A1/A3/A4/A7.

## Related Documents

- [m-verify-soundness-coverage-program](../v0_61_0/m-verify-soundness-coverage-program.md): the program this continues (items 4b–4f, item 5)
- [m-v1-nominal-type-identity](../v1_0_0/m-v1-nominal-type-identity.md): module-qualified *type* identity, including SMT `adtTypes`; share the notation
- [m-contract-verification-coverage](../v1_1_0/m-contract-verification-coverage.md): the skipped vs not_applicable split
- [m-verify-call-site-requires](../../implemented/v0_61_0/m-verify-call-site-requires.md): obligations that T1 must emit per instance

## Future Work

- Specialising recursive callees on an invariant function argument (T2's deferred decision)
- Induction for helpers' map/filter/foldl frontiers (after T6)
- A bit-vector int model (program Future Work)
