# Sprint Plan — M-PROPERTY-GENERATOR-COVERAGE **Lane B1 only**

**Design doc**: [m-property-generator-coverage.md](m-property-generator-coverage.md) (Lane B1 = the `B1. Structural derivation` block)
**Sprint ID**: `M-PROPERTY-GENERATOR-LANE-B1`
**Upstream filing**: sunholo-data/ailang#517
**Predecessor**: Lane A **LANDED** (`a81d66983`, PR #536) — see [lane-a sprint plan](m-property-generator-coverage-lane-a-sprint-plan.md)
**Base**: `22ba8626d` (branch `dev`)
**Planner**: V1 mission iteration 168. Every fact in §1 and §2 was measured first-party in the **main checkout** at `22ba8626d` with `go build -o /tmp/ailang_planner168 ./cmd/ailang` (binary self-reports `Commit: 22ba862`).
**Estimate**: **2.25 days** (doc says ~1.5–2 d; the doc's estimate predates Lane A and omits three real work items — see §7)
**Risk**: medium-high (derivation touches a hot path; 87 never-executed properties start running; two genuine determinism/termination defects must be fixed *inside* this sprint)

**Scope**: Lane B1 **only**.
**B2 is OUT** — deferred by the 2026-07-29 quorum, blocked on a deterministic evaluator fuel budget, on the
reviewer's own proposal. Do not implement `gen<TypeName>`, do not implement the alias-first hint text
(B-9), do not touch refined types. **Lane A is done** — do not re-do the list arm, the skip taxonomy,
`Success()`, the reporter, or the JSON preamble.

---

## 0. Executor operating constraints (read first)

1. **You cannot run `git` at all.** The controller reconstructs each commit from your per-milestone
   `.snap/M<k>/` snapshot. Every milestone below states its **exact file set** (NEW / MOD). Snapshot
   exactly those files, and report any file you touched that is not on the list — an unlisted file is a
   planning defect, not a silent extra.
2. **Flags BEFORE the positional path, always.** `ailang test f.ail --format json` silently ignores
   `--format` (root cause `cmd/ailang/main.go`'s `flag.Parse` stopping at the first non-flag arg; filed
   as **#534**, out of scope). Every command in this plan is written flags-first. A flags-last
   acceptance check tests nothing.
3. **`perl -e 'alarm N; exec @ARGV'` DOES NOT BOUND A GO BINARY.** The Lane A plan's §0.8/R-9 recipe is
   a silent no-op against `ailang`: the Go runtime classifies `SIGALRM` as notify-only, and with no
   `os/signal` listener it is **ignored**. Measured this sprint with a positive control (§2, F-6). Use
   the background-and-kill recipe in §1.6 instead. This matters: an unbounded sweep wedges the loop.
4. **Sandbox-uninformative gates.** `go test ./internal/testing/... -count=1` binds no sockets —
   **informative**, and it is your primary gate. `go test ./cmd/ailang/...` and `make test`
   (= `go test ./...`) bind loopback sockets in six `cmd/ailang` test files and MUST be reported as
   **`UNINFORMATIVE UNDER SANDBOX`**, never pass/fail. The controller re-runs them.
5. **`go build ./...` IS RED ON THE PRISTINE TREE** (rc 1: `cmd/wasm` and `gen/main` declare no native
   `main`). It is **not** a gate. Use `go build ./internal/... ./cmd/ailang/...` (measured rc **0** at
   base) wherever you want a build gate.
6. **`make lint` prints `1 issues: * unused: 1` and still exits 0** at base. Gate on the **exit code**,
   never on the absence of that line.
7. **File-size gate is live and binding.** `internal/testing/runner.go` is **749** lines against the
   800-line limit at `make/code-health.mk:122` — **51 lines of headroom**. B1 adds far more than that.
   The plan therefore *removes* two functions from `runner.go` into new files (M1, M3); this is
   deliberate, not drive-by refactoring. `make check-file-sizes` is a gate on **every** milestone.
8. **Do not modify existing test assertions.** As in Lane A, `git diff -U0 -- '*_test.go'` should show
   **no removed lines** in pre-existing test files. Two exceptions are pre-authorised and named
   explicitly in M3 (`generator_advanced_test.go`, only if the determinism fix changes an
   order-dependent assertion) — if you use them, say so per milestone.
9. **Adding a file under `examples/runnable/` requires a matching `examples/manifest.json` entry**
   (`path` / `status` / `tags` / `description`), and the file must `ailang run` successfully — every
   contracts example in the manifest has an `export func main() -> int`. See M6.
10. **Never assert `stderr == ""`.** A legitimate `Warning: stdlib version mismatch: expected dev, found
    v0.33.0 …` already goes to stderr on this rig (observed in the §1.4 sweep). Assert on stdout only.

---

## 1. Ground truth measured for this plan

### 1.1 The generator surface at `22ba8626d`

`createGeneratorForType` (`internal/testing/runner.go:596`) has **three** arms:

| Arm | Line | Status |
|---|---|---|
| `*ast.SimpleType` ∈ {`int`,`float`,`bool`,`string`} | 598–612 | live |
| `*ast.TypeApp{Constructor:"list", len(Args)==1}` | **615** | **live — shipped in Lane A (`a81d66983`)** |
| `*ast.ListType` | 625 | dead from parsed programs; kept for Go-test constructions |
| fallthrough `return nil, nil` | 636 | the vacuous-skip source |

**`list[T]` is NOT B1 work.** It already ships and it recurses into `createGeneratorForType`, so
`list[<anything B1 derives>]` falls out of B1 for free.

### 1.2 There are **THREE** generator call sites, across **TWO** files

The controller's brief names two (`runner.go:281`, `runner.go:415`). A third — **the most important
one** — lives in a different file:

| # | Call site | Path | Skip text on `nil` |
|---|---|---|---|
| 1 | `internal/testing/runner.go:281` | `runProperty` (forall binders) | `no generator for type %v` |
| 2 | `internal/testing/runner.go:415` | `runRequiresProperty` | `no generator for parameter %s: %v` |
| 3 | **`internal/testing/contract_domain.go:74`** | **`runEnsuresProperty`** | `no generator for parameter %s: %v` |

`runEnsuresProperty` was moved out of `runner.go` into `contract_domain.go` (194 lines) at some point
after the design doc was written. **The doc's "Files to Modify" list for Lane B does not mention
`contract_domain.go` at all**, and every `ensures`-derived property in the corpus flows through it.
Wiring only `runner.go` would leave the majority of the corpus unfixed.

### 1.3 The value-splice surface — **THREE** `valueToLiteral` call sites, across **TWO** files

| # | Call site | Consumer |
|---|---|---|
| 1 | `internal/testing/contract_domain.go:101` | `EnsuresParam.Value = astExprToCore(r.valueToLiteral(v))` |
| 2 | `internal/testing/runner.go:438` | requires-path `EnsuresParam.Value` |
| 3 | `internal/testing/runner.go:652` | `bindPropertyValues` (forall path; also reached from `shrinkCounterexample` at `:725`) |

Plus one internal recursive call at `runner.go:693` (list elements).

`valueToLiteral`'s default arm is at **`runner.go:696-701`** and silently returns
`&ast.Literal{Kind: ast.UnitLit, Value: struct{}{}}`. Changing its signature to `(ast.Expr, error)`
therefore ripples through **`bindPropertyValues`** (whose own signature must change) and its two callers
`runProperty:315` and `shrinkCounterexample:725`. Budget for this; it is not a one-file edit.

### 1.4 Full 33-file corpus sweep at `22ba8626d` (measured; supersedes V34)

Command per file (bounded, §1.6): `ailang test --format json --no-color <file>`.
**V34 was a lower bound and understated the corpus**: it never reached `scoring.ail` or `showcase.ail`,
which together hold **15** vacuous skips and four type names V34 never lists (`SkillLevel`, `Tenure`,
`Performance`, `Department`).

`vac` = JSON `vacuous_skips`. `B1 fixes` = vacuous skips whose shape B1 derives **and** whose type is
resolvable same-file.

| File | succ | p/f/s | vac | Vacuous shapes | B1 fixes | After B1 |
|---|---|---|---|---|---|---|
| access_control | false | 0/0/4 | 4 | `Role` (ADT, same-file) ×4 | **4** | rc 1→**0** |
| basic | false | 3/2/3 | 0 | — (3 × out_of_contract) | 0 | rc 1 (2 real fails) |
| cross_function | false | 0/0/5 | 5 | `Region` ×4, `Priority` ×1 (ADT, same-file) | **5** | rc 1→**0** |
| cross_module_functions | **true** | 3/0/2 | 0 | — (ooc) | 0 | rc 0 — **must stay 0** |
| cross_module_functions_lib | **true** | 2/0/1 | 0 | — (ooc) | 0 | rc 0 — **must stay 0** |
| cross_module_types | false | 2/0/6 | 5 | `Cell` ×3 **(IMPORTED)**, `list[Tree]` ×1 **(IMPORTED)**, `()` ×1 | **1** | rc 1 (4 vacuous remain) |
| cross_module_types_lib | false | 0/0/0 | 0 | — (no tests) | 0 | rc 1 |
| ensures_violation_demo | false | 1/1/0 | 0 | — | 0 | rc 1 |
| finance | false | 0/1/5 | 4 | `TaxBracket` (ADT, same-file) ×4 | **4** | rc 1 (1 real fail) |
| hof_verify | **true** | 2/0/0 | 0 | — | 0 | rc 0 — **must stay 0** |
| inbox_injection | false | 3/1/6 | 0 | — (ooc) | 0 | rc 1 |
| inbox_injection_v2 | false | 1/0/10 | 10 | `string<email>` **(REFINED — B2)** ×10 | **0** | rc 1 — **unchanged** |
| inbox_v2_app | false | 1/0/10 | 10 | `Mail` **(IMPORTED)** ×9, `string<email>` ×1 | **0** | rc 1 — **unchanged** |
| inbox_v2_lib | false | 0/0/0 | 0 | — (no tests) | 0 | rc 1 |
| insurance | false | 0/0/6 | 6 | `AgeBand` ×4, `RiskTier`, `Coverage` (ADT, same-file) | **6** | rc 1→**0** |
| invoice | false | 4/1/16 | 13 | `CustomerTier` ×5, `TaxRegion` ×3 (ADT), `{subtotal,tax,discount}` ×5 | **13** | rc 1 (1 real fail) |
| list_recursive_verify | false | 0/0/6 | 0 | — (ooc; see Lane A H1) | 0 | rc 1 |
| list_verify | false | 5/1/1 | 0 | — (ooc) | 0 | rc 1 (1 deliberate fail) |
| nested_record_verify | false | 0/0/5 | 5 | `{ name: string, pos: { x: int, y: int } }` **(NESTED anon)** ×5 | **5** | rc 1→**0** |
| park | false | 1/0/7 | 6 | `Season` (ADT, same-file) ×6 | **6** | rc 1→**0** |
| per_function_depth_verify | **UNMEASURED** | — | — | exceeds a 60 s bound (rc 137) | ? | report UNMEASURED |
| quantifier_verify | false | 2/1/4 | 0 | — (4 × ooc) | 0 | rc 1 |
| record_adt_cycle_verify | false | 0/0/1 | 1 | `Doc` (named record, same-file, **MUTUALLY RECURSIVE**) | **1** | rc 1→**0** or honest fail |
| record_adt_sort_verify | false | 0/0/1 | 1 | `Proposal` (named record, same-file, recursive-ish) | **1** | rc 1→**0** or honest fail |
| record_discovery_verify | false | 2/0/8 | 6 | `{x:int,y:int}` ×4, `{a:int,b:int}` ×2 | **6** | rc 1 (2 ooc remain) |
| record_pattern_verify | false | 0/0/5 | 5 | `{x:int,y:int}` ×5 | **5** | rc 1→**0** |
| record_verify | false | 0/0/15 | 15 | `{x:int,y:int}` ×15 | **15** | **expect ≥1 GENUINE FAIL** (`brokenDistance`, `:59`) |
| recursive_verify | false | 0/0/9 | 0 | — (ooc) | 0 | rc 1 |
| scoring | false | 0/0/6 | 6 | `SkillLevel` ×3, `Tenure`, `Performance`, `Department` (ADT, same-file) | **6** | rc 1→**0** |
| showcase | false | 5/1/9 | 9 | `{subtotal,tax,discount}` ×5, `Priority` (ADT) ×4 | **9** | rc 1 (1 real fail) |
| string_verify | false | 4/1/0 | 0 | — | 0 | rc 1 |
| temperature | **true** | 2/0/2 | 0 | — (ooc) | 0 | rc 0 — **must stay 0** |
| unencodable_callee_skip | **true** | 1/0/1 | 0 | — (ooc) | 0 | rc 0 — **must stay 0** |

**Totals**: **111** vacuous skips across 32 measured files. **B1 fixes 87. 24 remain and stay honest
vacuous skips**: `Cell` ×3 + `list[Tree]` ×1 (imported), `Mail` ×9 (imported), `string<email>` ×11
(refined). Those 24 are B2/out-of-scope by design.

**Files that MUST keep rc 0** (false-red guards): `cross_module_functions`,
`cross_module_functions_lib`, `hof_verify`, `temperature`, `unencodable_callee_skip`.

### 1.5 Type-declaration shapes B1 must resolve (parser-verified)

Measured against `internal/parser/parser_type_decl.go:79-198`:

| Surface syntax | `TypeDecl.Definition` | Doc covers it? |
|---|---|---|
| `type Doc = { title: string, blocks: list[Block] }` | `*ast.RecordType` | yes |
| `type Season = SPRING \| SUMMER` | `*ast.AlgebraicType` | yes |
| `type Block = Para(string) \| Container({...})` | `*ast.AlgebraicType` (positional `ConstructorField`, `Name == ""`) | yes |
| `type Pair = (int, string)` | **`*ast.TypeAlias{Target:*ast.TupleType}`** | **NO — doc omits `TypeAlias` entirely** |
| `type Names = [string]` | **`*ast.TypeAlias{Target:*ast.TypeApp{"list"}}`** | **NO** |
| `type UserId = int` | **`*ast.TypeAlias{Target:*ast.SimpleType}`** | **NO** |
| parameter `()` | `*ast.SimpleType{Name:"()"}` (`parser_type.go:96`, `parser_func.go:72`) | doc says "unit"; **the concrete node is `SimpleType`, not an empty tuple** |
| parameter `{x: int, y: int}` | `*ast.RecordType` (also implements `ast.Type`) | yes |
| parameter `string<email>` | `*ast.LabelledType` | out of scope (B2) |

**`*ast.TypeAlias` is a required derivation arm the design doc never lists.** It is one line
(recurse on `.Target`) but without it every `type UserId = int` parameter stays vacuous. There is **no
corpus occurrence**, so its acceptance is unit-level only — say so, do not claim a corpus observable.

### 1.6 Bounded-wait recipe that actually works

```bash
# /tmp/bwait.sh <seconds> <cmd...>
lim=$1; shift
"$@" & pid=$!
( sleep "$lim"; kill -9 "$pid" 2>/dev/null ) & w=$!
wait "$pid"; rc=$?
kill "$w" 2>/dev/null; wait "$w" 2>/dev/null
exit $rc
```
Verified this sprint: bounds a Go binary at the limit (rc 137) and does **not** kill a fast one (rc 0).
`timeout` and `gtimeout` are **not on PATH** on this rig.

### 1.7 Measured pristine-tree baselines for every gate used below

| Gate | Baseline at `22ba8626d` | Usable as a gate? |
|---|---|---|
| `go test ./internal/testing/... -count=1` | **rc 0** (`ok … 1.347s`) | **yes — primary** |
| `go vet ./internal/testing/...` | **rc 0**, no output | yes |
| `make check-file-sizes` | **rc 0** (`✓ All files within 800 line limit`) | yes |
| `make lint` | **rc 0** — but prints `1 issues: * unused: 1` | yes, **on exit code only** |
| `go build ./internal/... ./cmd/ailang/...` | **rc 0** | yes |
| `go build ./...` | **rc 1** — `cmd/wasm`, `gen/main` have no native `main` | **NO — already red** |
| `go test ./cmd/ailang/...`, `make test` | socket-binding | **NO — `UNINFORMATIVE UNDER SANDBOX`** |
| `make verify-examples` | **rc 0** (`186 modules checked, 0 drift, 1 missing-on-disk`; `✅ all examples pass and manifest is in sync`). It invokes `ailang run`, never `ailang test` | yes, but only meaningful in M6 |
| `ailang test --format json --no-color <corpus file>` | per-file rc/counters in §1.4 | yes, per-file |

---

## 2. Refutations and new findings (read before executing)

### F-1 — REFUTES the doc's Success metric: **B1 does not fix the prompt-injection safety demos**

The doc's headline motivation is that `inbox_injection_v2.ail` and `inbox_v2_app.ail` — "the
prompt-injection safety demos — their safety properties have never executed" — and its success metric
says "after Lane B1, the five silent-shape example files above run their skipped properties".

**Measured: B1 fixes ZERO properties in either file.**
`inbox_injection_v2`'s 10 vacuous skips are all `string<email>` — a **refined type**, explicitly out of
scope for B1 and parked with B2. `inbox_v2_app`'s are `Mail` ×9 — declared in
`inbox_v2_lib.ail` and **imported** (`inbox_v2_app.ail:24`), so the same-file derivation bound leaves
it vacuous — plus one `string<email>`. Verified with `grep -E '^(export )?type Mail' inbox_v2_app.ail`
→ no match, and `grep '^import' inbox_v2_app.ail` → the lib import.

**Action**: do not write an acceptance criterion that expects those two files to change. §1.4 pins them
as *unchanged*. The mission-visible claim that B1 makes the injection demos honest is **false**; the
honest claim is that Lane A already made them *loud* (they exit 1 today) and B2 makes them *run*.

### F-2 — REFUTES B-3 and the "prerequisite fix" for `ADTGenerator`

The doc calls parameterising `ADTGenerator.Generate`'s hardcoded `ModulePath:"test", TypeName:"ADT"` a
**prerequisite**, "or generated `TaggedValue`s won't match/typecheck in the harness", and makes
restoring the hardcode B-3's red-turning mutation. Both legs are refuted:

1. **Nothing matches on `TypeName`/`ModulePath`.** `matchPattern`'s `*core.ConstructorPattern` arm
   (`internal/eval/eval_patterns.go:178-194`) compares **only** `tagged.CtorName != p.Name` and arity.
   `taggedValuesEqual` (`:410`) compares only `CtorName` + fields. `isTag` — the one function that does
   compare `TypeName` (`:749`) — has **zero non-test callers** (only `adt_runtime_test.go:140`).
   `TaggedValue.String()` (`value.go:362`) prints `CtorName(fields…)` and never the type name, so it is
   not observable in counterexample text either.
2. **The splice throws the generated `TaggedValue` away.** `valueToLiteral(TaggedValue)` becomes
   `ast.FuncCall(Identifier(ctor), fields)` → `core.App{Func: core.Var{ctor}}` → the harness env's
   `ConstructorClosure.Apply` (`eval_operations.go:241`, `value.go:395`) **rebuilds** a fresh
   `TaggedValue` using the `TypeName` injected by `injectADTConstructors`
   (`executor_helpers.go:461-501`) — which is the *correct* one. The generator's own tag fields never
   reach the evaluator.
3. **There is no typechecking on this path.** `evaluateEnsuresHarnessCore` (`executor.go:144-167`)
   builds a `core.Program` and calls `EvalCoreProgram` directly. Nothing typechecks the spliced value.

**So B-3's stated mutation kills nothing** — it is an observable *adjacent* to the mechanism. Replaced
in §4 by **B1-6** (constructor coverage, downstream of the `OneOf` sum) and **B1-7** (nullary-vs-n-ary
splice, downstream of the arity split), both of which have real downstream writes. Parameterising
`ADTGenerator` is retained as a **cosmetic/diagnostic** change in M4 with an honest *unit-level*
criterion, not as a prerequisite.

### F-3 — NEW, load-bearing: `RecordGenerator` is **NONDETERMINISTIC**; wiring it as-is breaks axiom A1

`RecordGenerator.Generate` (`generator_advanced.go:203-212`) iterates `map[string]Generator` with
`for fieldName, fieldGen := range g.fieldGens` and calls `fieldGen.Generate(rng)` inside the loop.
**Go randomises map iteration order per range statement** — measured this sprint with a 6-run probe
over a 4-key map: 3 distinct orders observed. So the *order of RNG draws* varies run to run, and a
fixed seed no longer reproduces a fixed record.

This directly contradicts the doc's own axiom table ("A1: Determinism | 0 | Derived generators use the
existing seeded RNG; no new nondeterminism") and defeats the whole `propertySeed` / `SetSeedMetadata`
replayability machinery — a counterexample would not reproduce from its reported seed.

**Action (M3, mandatory)**: give `RecordGenerator` a deterministic field order (sort the keys once in
`NewRecordGenerator`, iterate the sorted slice in `Generate`). Constructor signature unchanged, so no
call-site churn. Acceptance **B1-4** in §4.

### F-4 — NEW, load-bearing: a depth budget alone does **not** bound generation cost

`DefaultConfig().MaxSize == 100` (`generator.go:30`) and the list arm generates lengths `0..MaxSize`.
The in-repo fixture `record_adt_cycle_verify.ail` is **mutually recursive through a record, an ADT and
a list**:

```
export type Block = Para(string) | Container({ blocks: list[Block], kind: string })
export type Doc   = { title: string, blocks: list[Block] }
```

At depth 3 with uncapped list lengths, one `Doc` draw has ~50 `Block`s, each `Container` ~50 more →
~2,500 nodes with strings up to 100 chars, **×100 cases**. This is the *same* critique the quorum
levelled at B2 (depth bounds recursion, not total work), applied to B1's own structural generation —
and the doc does not address it.

**Action (M4, mandatory)**: thread a **size budget** alongside the depth budget — cap list/collection
length as a function of remaining depth (e.g. `maxLen = min(MaxSize, remainingDepth)` for
derivation-internal lists; top-level `list[scalar]` keeps today's `0..MaxSize` so Lane A behaviour is
unchanged). Acceptance **B1-9** in §4 asserts a *deterministic size bound on the generated values*, not
a wall-clock time, so the gate cannot flake.

### F-5 — REFUTES the doc's shrinking bullet: derived shrinkers have **no downstream observable**

`createGeneratorForType` returns `(Generator, Shrinker)`, but **both** contract paths discard the
shrinker: `contract_domain.go:74` and `runner.go:415` are both `gen, _ := …`. The only consumer is
`shrinkCounterexample`, reached only from `runProperty` (the **forall** path), which is broken upstream
(`empty program`, #624 / doc V10, V36) and cannot produce a counterexample at all.

So a `RecordShrinker`/`TupleShrinker` cannot be observed through `ailang test` by any means. The doc's
own note ("treat quality of shrunk counterexamples as unvalidated") understates it: they are not merely
unvalidated, they are **unreachable**.

**Action**: M5 is scoped as **unit-observable only** and marked **DROPPABLE** (see §6). No acceptance
criterion in this sprint may claim a counterexample-quality improvement.

### F-6 — REFUTES the Lane A plan's bounded-wait guidance (and it bit this sprint)

Lane A §0.8 / R-9 tells the executor to bound sweeps with "`gtimeout`/`perl alarm`". Measured with a
positive control:

| Command | Elapsed | rc |
|---|---|---|
| `perl -e 'alarm 3; exec @ARGV' <20s Go binary>` | **20 s** | 0 — **not bounded** |
| `perl -e 'alarm 3; exec @ARGV' /bin/sleep 20` (control) | 3 s | 142 — bounded |

The Go runtime treats `SIGALRM` as notify-only; with no listener it is ignored. `gtimeout` is not
installed. This planner's first sweep attempt hung on `per_function_depth_verify.ail` because of it.
**Use §1.6's recipe.**

### F-7 — `runEnsuresProperty` moved files; the doc's Lane B file list is wrong

See §1.2. `internal/testing/contract_domain.go` is not in the doc's "Files to Modify (Lane B)" list and
must be. Also: the out-of-contract skip text on the ensures path is now
`unverified: requires filter accepted N of M generated inputs; need 100 (K discarded)` — a *different*
string from the requires path's `requires not satisfied by random input …`. Do not assert a single
out-of-contract substring across both paths.

### F-8 — V34 is confirmed a lower bound; two whole files were missing

`scoring.ail` (6 vacuous: `SkillLevel`, `Tenure`, `Performance`, `Department`) and `showcase.ail`
(9 vacuous: `{subtotal,tax,discount}`, `Priority`) appear in neither V34 nor the doc's blast-radius
paragraph. §1.4 is the complete 32-of-33 measurement (`per_function_depth_verify` exceeds a 60 s bound
and is reported UNMEASURED, not pass/fail).

---

## 3. Milestones

Each milestone is **independently committable and bisectable**, has its **own runnable test package
boundary**, and states its exact file set. **Every** milestone additionally requires
`make check-file-sizes` rc 0, `go vet ./internal/testing/...` rc 0, `make lint` rc 0, `gofmt` clean on
touched files, and `go test ./internal/testing/... -count=1` rc 0.

**Ordering is non-negotiable across M1 → M2**: `valueToLiteral` gains its arms **before** the default
becomes a loud error. Reversing it fails the harness for every unit-typed generated value (quorum
round 2, `gemini-3-1-pro`).

---

### M1 — Splice arms (`UnitValue` / `RecordValue` / `TupleValue` / `TaggedValue`), default still silent

**This is the first half of the mandated ordering.** No behaviour change is observable at the CLI: no
generator yet produces these value kinds, so the arms are exercised only by unit tests. That is the
point — M1 is provably inert, so a bisect that lands here is exonerated.

**Change**
1. **Move** `valueToLiteral` (currently `runner.go:666-703`) into a new file
   `internal/testing/value_splice.go`, unchanged. This buys ~38 lines of headroom in `runner.go`
   (749 → ~711) before M2/M3 add anything. Keep the receiver `(r *Runner)`.
2. Add four arms, **in this order before the default**:
   - `*eval.UnitValue` → `&ast.Literal{Kind: ast.UnitLit, Value: struct{}{}}`
   - `*eval.RecordValue` → `&ast.Record{Fields: []*ast.Field{...}}` — **iterate field names in sorted
     order** so the produced AST is deterministic (`RecordValue.Fields` is a map).
   - `*eval.TupleValue` → `&ast.Tuple{Elements: [...]}`
   - `*eval.TaggedValue` → **arity-split**: `len(Fields) == 0` ⇒ bare
     `&ast.Identifier{Name: v.CtorName}`; otherwise
     `&ast.FuncCall{Func: &ast.Identifier{Name: v.CtorName}, Args: [...]}`.
     The split is load-bearing: `injectADTConstructors` binds nullary constructors to a
     **`*eval.TaggedValue`**, not a closure (`executor_helpers.go:483-489`), so emitting `FuncCall` for
     a nullary constructor produces `core.App` over a non-function and dies with
     `cannot apply non-function value: *eval.TaggedValue` (`eval_operations.go:245`).
3. Default arm **unchanged** (still silently returns unit). Add a `TODO(M2)` comment naming this plan.

**Files**
- NEW `internal/testing/value_splice.go`
- MOD `internal/testing/runner.go` (remove `valueToLiteral`; no other change)
- NEW `internal/testing/value_splice_test.go`

**Gate M1**
```bash
go test ./internal/testing/... -count=1     # base rc 0 → must stay rc 0
go vet ./internal/testing/... && make check-file-sizes && make lint
go build ./internal/... ./cmd/ailang/...    # base rc 0
# INERTNESS PROOF (behaviour must be byte-identical to base):
#   re-run the §1.4 sweep over the 32 measurable files and diff succ/p/f/s/vac against §1.4
```

**Est.**: 90 impl / 140 test LOC · 0.25 d

---

### M2 — Loud default (the refusal), signature `(ast.Expr, error)`, error → property **Fail**

**This is the second half of the mandated ordering — it may not land before M1.**

**Change**
1. `valueToLiteral` returns `(ast.Expr, error)`. Default arm returns
   `nil, fmt.Errorf("no literal splice for generated value of type %T (%s)", value, value.Type())`.
2. Thread the error through **all four** internal/external call sites named in §1.3:
   - `contract_domain.go:101` (ensures) → on error set `Status: StatusFail`, `Error:` the message,
     `Duration`, and return. **Never `StatusSkip`** — a skip here would re-open the vacuous hole.
   - `runner.go:438` (requires) → same.
   - `bindPropertyValues` (`runner.go:643`) gains `(ast.Expr, error)`; `runProperty:315` → `StatusFail`.
   - `shrinkCounterexample:725` → a splice error means "this shrink candidate is unusable"; `continue`,
     exactly as the existing `EvaluateExpression` error branch already does. Best-effort by design;
     it must **not** panic and must **not** change the reported verdict.
   - internal recursion `runner.go:693` (list elements) propagates.

**Refusal branches and their required neutering mutations** (a guard is not a gate until something reds
when you remove it — write each as `if false && <cond>` so the mutant still compiles):

| # | Refusal branch | Neutering mutation | What must red |
|---|---|---|---|
| N-1 | `valueToLiteral` default returns an error | `default: if false && true { return nil, fmt.Errorf(...) }; return &ast.Literal{Kind: ast.UnitLit}, nil` | unit test asserting an unknown `eval.Value` (e.g. `&eval.FuncValue{}`) yields a non-nil error |
| N-2 | ensures site converts the error to `StatusFail` | `if false && err != nil { …fail… }` at `contract_domain.go` | ensures-path test asserting `Status == StatusFail` + message; with the mutant, `astExprToCore(nil)` panics ⇒ still red (do **not** "fix" that panic) |
| N-3 | requires site converts the error to `StatusFail` | `if false && err != nil { … }` at `runner.go:438` | requires-path test asserting `StatusFail` |
| N-4 | forall/`bindPropertyValues` site converts the error to `StatusFail` | `if false && err != nil { … }` at `runner.go:315` | forall-path unit test asserting `StatusFail` |
| N-5 | shrink site skips an unusable candidate instead of crashing | `if false && err != nil { continue }` | shrink test with one unspliceable candidate: must still return a minimal value, not panic |

Each neutering mutation must be **applied, observed red, and reverted** by the executor, and the
observation recorded in the milestone report (one line per branch). Five branches, five mutations.

**Files**
- MOD `internal/testing/value_splice.go`
- MOD `internal/testing/runner.go`
- MOD `internal/testing/contract_domain.go`
- MOD `internal/testing/value_splice_test.go`
- NEW `internal/testing/value_splice_refusal_test.go`

**Gate M2**: all §3 common gates, plus B1-1, B1-2, B1-3 (§4), plus the five recorded neutering
observations, plus the §1.4 inertness re-sweep (still no generator produces these shapes, so the corpus
must remain byte-identical).

**Est.**: 60 impl / 130 test LOC · 0.25 d

> ### ⚠ CONTROLLER CORRECTION — M2 AS WRITTEN IS NOT EXECUTABLE (V1 mission iteration 168, 2026-08-10)
>
> **N-2 … N-5 cannot be observed red, because at M2 the branches they pin are unreachable from any
> production path.** The four call-site branches only fire when `valueToLiteral` refuses a value, and
> refusal requires a generated value of a kind M1 did not give an arm to. Measured at `973803c65`:
> the only generators *reachable* at M2 are the five `createGeneratorForType` returns — `Int`, `Float`,
> `Bool`, `String`, `List` — all of which yield kinds `valueToLiteral` already splices. (The
> `generator_advanced.go` combinators are richer, but M2 has no path to them: they are unwired until
> M3/M4, which is the whole premise of Lane B.) And `Runner` (`runner.go:16`) has **four unexported
> fields and no seam**: `createGeneratorForType` is a plain method, so a test cannot inject a generator
> that yields an unspliceable value. The first executor run spent its entire slot deliberating about
> exactly this and wrote no code; the finding is what the slot bought, and it is a real one.
>
> Note this is **not** a "wait until M3" problem. M3/M4 derive generators for records, tuples, unit and
> ADTs — which are precisely the kinds M1 adds arms for — so the default arm stays unreachable *after*
> the whole of B1. It is a permanently defensive branch, which by this mission's own standard (rule 3j:
> a guard is not a gate until something reds when you remove it) is the kind most likely to rot.
>
> **Decision: add the seam.** M2 gains an unexported function field on `Runner`, defaulting to
> `(*Runner).createGeneratorForType`, that the three generator call sites route through
> (`contract_domain.go:74`, `runner.go:281`, `runner.go:415`). A test then injects a generator returning
> an unspliceable value (e.g. `&eval.FuncValue{}`) and each of N-2…N-5 becomes observable. This is a
> small, documented production change whose entire purpose is to make five guards real rather than
> decorative — the alternative, declaring them unreachable per rule 3j, would ship four error paths
> nothing protects.
>
> **N-1 is unaffected** — the default arm itself is reachable by unit-calling `valueToLiteral` directly,
> so it needs no seam.
>
> Re-estimate M2: **90 impl / 160 test LOC · 0.35 d** (was 60/130 · 0.25 d).

> ### ✅ M2 LANDED — completion record (V1 mission iteration 169, 2026-08-10)
>
> Executor `pi:openrouter/deepseek/deepseek-v4-flash-0731`, 51 turns, `metered=$0.086`. Delivered on
> `sprint/iter169-b1-m2`: `value_splice.go` 92→109, `runner.go` 710→754 (800 gate still green),
> `contract_domain.go` 194→203, `value_splice_test.go` 213→234, NEW `value_splice_refusal_test.go`
> and `value_splice_roundtrip_test.go`. The seam landed as an unexported `Runner.genForType` field
> bound in `NewRunnerWithConfig`, reached through a `genForTypeSeam` accessor that falls back to
> `createGeneratorForType` for any `Runner` built another way.
>
> **Mutation drill, controller-run, every mutant asserted LANDED (sha256) and BUILDS (`go build`
> rc=0) before its result was read, and every one restored byte-identical from a `cp` backup.** Each
> killer was scoped with `-run`, and each was paired with the inverse arm (`-skip` the killer → rest
> of the package rc=0) so it is the sole killer rather than a bystander.
>
> | # | Mutation | Killer | Verdict | Inverse arm |
> |---|---|---|---|---|
> | N-1 | `default:` arm neutered to `if false && true`, falls back to a unit literal | `TestRefusal_N1_DirectCall` | **KILLED** | also reds N-2/N-3/N-4 — broader protection, recorded rather than treated as a defect |
> | N-2 | ensures `if false && err != nil` | `TestRefusal_N2_Ensures` | **KILLED** (panics in `astExprToCore(nil)`, the plan's predicted path — not "fixed") | sole killer |
> | N-3 | requires `if false && err != nil` | `TestRefusal_N3_Requires` | **KILLED** | sole killer |
> | N-4 | forall/`bindPropertyValues` `if false && err != nil` | `TestRefusal_N4_Forall` | **KILLED** | sole killer |
> | N-5 | shrink `if false && err != nil { continue }` | `TestRefusal_N5_Shrink` | **SURVIVED** — see below | — |
> | B1-2 | `RecordValue` arm emits `ast.Tuple` (plan §4's named mutation for this AC) | `TestB12_RoundTrip` | **KILLED** (`got (1, true), want {x: 1, y: true}` — reads the *evaluated value*, one step past the AST, exactly as the AC row promises) | also reds other arms |
>
> **N-5 is DECLARED REDUNDANT rather than quietly shipped** (rule 3j: an unreachable branch is
> acceptable *when declared in the code and in the AC*; an undeclared one is a guard nobody is
> protecting). The executor self-reported this before the controller measured it, and its stated
> reason was checked rather than adopted (rule 3h): on a splice refusal `bindPropertyValues` returns
> `(nil, err)`, and `EvaluateExpression` builds its program by string-formatting the AST, so a nil
> expression yields unparseable source and **errors** — the adjacent pre-existing branch then
> `continue`s for the same effect. The guard is kept, because depending on that is an implicit
> contract, and the contract is now pinned: `TestShrinkNilExprContract` asserts
> `EvaluateExpression(nil)` returns an error and does not panic. That pin is itself non-vacuous —
> mutating `EvaluateExpression` to `panic` on nil BUILDS rc=0 and reds it. If it ever fires, N-5
> stops being decorative and its neutering mutation starts killing.
>
> **AC status**: B1-1 ✅ (M1), **B1-2 ✅ — the debt iteration 168 named as owed is paid here**,
> B1-3 ✅ with the N-5 caveat above stated in full.

---

### M3 — Structural derivation: records (named + anonymous + nested), tuples, unit, aliases

**Change**
1. NEW `internal/testing/derive.go`. **Move** `createGeneratorForType` (`runner.go:596-637`) into it
   verbatim (another ~42 lines out of `runner.go`, → ~669), then extend it:
   - a `deriveCtx` carrying `depth int` and `sizeBudget int`; `createGeneratorForType(typ)` becomes a
     thin wrapper calling `deriveType(typ, newDeriveCtx())` so all three call sites in §1.2 are wired
     by construction, with **no edit** to `contract_domain.go`.
   - `*ast.SimpleType{Name:"()"}` → `NewConstantGenerator(&eval.UnitValue{})`, `NewNoOpShrinker()`.
   - `*ast.RecordType` (anonymous, incl. nested) → `NewRecordGenerator` over per-field derived
     generators; **nil if any field is underivable** (no silent substitution — CLAUDE.md §2).
   - `*ast.TupleType` → `NewTupleGenerator`.
   - `*ast.SimpleType{Name: X}` not in the scalar set → look up `*ast.TypeDecl{Name: X}` in
     `r.executor.sourceFile.Decls`; if found and `Definition` is `*ast.RecordType`, derive it.
     **If `sourceFile` is nil or no decl matches, return `nil, nil`** — imported types stay honest
     vacuous skips (Lane A already makes them loud).
   - **`*ast.TypeAlias` → recurse on `.Target`** (§1.5; the doc omits this).
   - `*ast.AlgebraicType` and `*ast.TypeApp` over a user type are **M4**.
   - **Preserve the existing `list` `TypeApp` arm and the `ListType` arm untouched** — the SCOPE UPDATE
     block in the doc is explicit that B1 must add beside them, never replace them.
2. MOD `internal/testing/generator_advanced.go` — **F-3 determinism fix**: `NewRecordGenerator` sorts
   and stores the field names once; `Generate` iterates the sorted slice. Signature unchanged.

**Files**
- NEW `internal/testing/derive.go`
- MOD `internal/testing/runner.go` (remove `createGeneratorForType`)
- MOD `internal/testing/generator_advanced.go` (determinism fix only)
- NEW `internal/testing/derive_test.go`
- NEW `internal/testing/generator_determinism_test.go`
- MOD `internal/testing/generator_advanced_test.go` — **only if** an existing assertion depends on map
  order; report explicitly if used (pre-authorised exception to §0.8)

**Gate M3**: common gates, plus B1-4, B1-5, B1-8, B1-10, B1-11 (§4), plus the §1.4 corpus sweep showing
**exactly** the record/tuple/unit rows moving and every `must stay rc 0` file still rc 0.

**Est.**: 170 impl / 230 test LOC · 0.5 d

---

### M4 — ADTs, recursion depth budget, size budget, `TypeApp` substitution

**Change** (all in `derive.go` unless noted)
1. `*ast.AlgebraicType` → `NewOneOfGenerator` of one `NewADTGenerator(ctor.Name, fieldGens, true)` per
   constructor. A constructor whose any field is underivable makes the **whole type** underivable
   (`nil, nil`) — do not silently drop constructors, that would bias the distribution invisibly.
2. **Depth budget, default 3**, decremented on **every** derivation step (not only ADT arms), so
   mutual recursion through record→list→ADT→record is bounded. At the bound, restrict ADT choice to
   **non-recursive constructors** (compute per-constructor recursiveness once per decl, memoised). If
   no non-recursive constructor exists → `nil, nil` ⇒ honest vacuous skip.
3. **Size budget (F-4)**: inside a derivation (depth < max), cap generated list length by remaining
   depth instead of `0..MaxSize`. Top-level `list[<scalar>]` keeps today's `0..MaxSize` so Lane A's
   measured behaviour on `hof_verify` / `list_verify` / `invoice` is unchanged.
4. `*ast.TypeApp` over a **user** type with args → substitute args into the decl's `TypeParams` before
   deriving, bounded by depth. Keep the `list` arm first.
5. MOD `generator_advanced.go`: parameterise `ADTGenerator` with the real module path + type name
   (**cosmetic / diagnostic only — F-2**; unit-level criterion, no end-to-end claim).

**Files**
- MOD `internal/testing/derive.go`
- MOD `internal/testing/generator_advanced.go`
- MOD `internal/testing/derive_test.go`
- NEW `internal/testing/derive_adt_test.go`

**Gate M4**: common gates, plus B1-6, B1-7, B1-9, B1-12, B1-13 (§4), plus the §1.4 sweep showing the
ADT rows moving and `record_adt_cycle_verify.ail` completing **inside a 60 s bounded wait**.

**Est.**: 140 impl / 200 test LOC · 0.5 d

---

### M5 — Derived shrinkers (**DROPPABLE — unit-observable only**)

Per **F-5** there is no path from a derived shrinker to any `ailang test` output. Implement, unit-test,
and say so plainly. **If the sprint is behind at the end of M4, drop this milestone** and file it as a
follow-up; nothing downstream depends on it.

**Change**: `RecordShrinker` (per-field, composing existing field shrinkers) and `TupleShrinker`
(per-element) in `shrink.go`; return them from the M3/M4 derivation arms; ADTs reuse the existing
`NewADTShrinker`.

**Files**
- MOD `internal/testing/shrink.go`
- MOD `internal/testing/derive.go`
- NEW `internal/testing/shrink_derived_test.go`

**Gate M5**: common gates + B1-14 (§4). **No criterion may claim improved counterexample quality.**

**Est.**: 90 impl / 110 test LOC · 0.2 d

---

### M6 — Corpus triage, feature example, closeout

**This milestone is triage-dominated: 87 never-executed properties across 15 files start running.**
Per the doc's Conflict Surface, failures here are the feature working — but they must be triaged
**inside** this sprint, not left red.

**Change**
1. **Re-sweep all 33 files** with §1.6's bounded recipe and produce the after-table against §1.4.
   For **every** newly-`fail` property, classify as one of:
   - **(a) deliberate** — the example is documented as broken (`record_verify.ail:59 brokenDistance`,
     header at `:8` says "Expect: 4 verified, 1 violation"). Record and leave failing.
   - **(b) genuine latent bug in the example's contract** — fix the *example*, not the runner, and say
     which line.
   - **(c) a B1 defect** — fix B1 and re-run the affected milestone's gate.
   No newly-failing property may be left unclassified.
2. NEW `examples/runnable/contracts/shapes_verify.ail` — the doc's `shapes.ail` (record + ADT + tuple +
   scalar `ensures`) **plus an `export func main() -> int ! {}`** (every manifest'd contracts example
   has one; `ailang run` is what `make verify-examples` executes). MOD `examples/manifest.json` with a
   matching entry (`path`, `status: "working"`, `tags`, `description`). Then `make verify-examples`.
3. MOD `changelogs/v0.18-current.md` `[Unreleased]`: record/ADT/tuple/unit parameters now get derived
   generators; **87 previously-never-executed contract properties across 15 in-repo examples now run**;
   name the files that flip rc 1 → 0; note the 24 skips that remain (imported + refined) and that they
   are B2.
4. MOD `design_docs/planned/v0_31_0/m-property-generator-coverage.md`: mark B1 landed; record F-1
   (the injection demos are **not** fixed by B1 — the doc's success metric is wrong), F-2, F-3, F-4,
   F-5, F-7; correct the Lane B file list to include `contract_domain.go`.
5. File follow-ups (**do not implement**): derived shrinkers if M5 dropped; `TypeAlias`-shaped named
   aliases lack corpus coverage; imported-type resolution (typechecker env) is the fix for the
   remaining 13 imported-type skips.

**Files**
- NEW `examples/runnable/contracts/shapes_verify.ail`
- MOD `examples/manifest.json`
- MOD `changelogs/v0.18-current.md`
- MOD `design_docs/planned/v0_31_0/m-property-generator-coverage.md`
- MOD any `examples/runnable/contracts/*.ail` fixed under classification (b) — **list each one**

**Gate M6**: common gates + `make verify-examples` rc 0 + the complete after-sweep table with every
newly-failing property classified (a)/(b)/(c).

**Est.**: 40 impl / 60 test LOC (+ ~120 doc lines) · 0.55 d, of which **0.5 d is triage**

---

## 4. Acceptance criteria — each with its measured pristine-tree baseline and a downstream mutation

**Rule applied to every row (controller requirement 4)**: the "observable" column names the *write* the
assertion reads, and that write is **downstream** of the mechanism, not adjacent to it.

| # | M | Criterion | Baseline at `22ba8626d` | Observable (which write the assertion reads) | Red-turning mutation |
|---|---|---|---|---|---|
| **B1-1** | M1 | `valueToLiteral(&eval.RecordValue{...})` → `*ast.Record`; `TupleValue` → `*ast.Tuple`; `UnitValue` → `ast.Literal{UnitLit}`; nullary `TaggedValue` → `*ast.Identifier`; n-ary → `*ast.FuncCall` | today all five return `ast.Literal{UnitLit}` (`runner.go:696-701`) — **measured by reading the default arm**; the sweep shows no corpus path reaches them | the returned AST node type | delete any one arm ⇒ that value falls to the default and the type assertion fails |
| **B1-2** | M1 | **Round-trip**: for each of the five shapes, `astExprToCore(valueToLiteral(v))` evaluated through a `eval.CoreEvaluator` (with `injectADTConstructors` for the ADT case) is structurally equal to `v` | not reachable today | the **evaluated `eval.Value`** — one full step past the AST, so an AST that is well-formed but semantically wrong still reds | emit `ast.Tuple` for `RecordValue` (compiles, round-trip value differs) |
| **B1-3** | M2 | Unknown `eval.Value` ⇒ `valueToLiteral` returns a **non-nil error**, and each of the three property paths turns it into `Status == StatusFail` with that message — **never `StatusSkip`** | today: silent `()` splice, property can *pass* on a fabricated unit | `PropertyResult.Status` + `.Error` at each of the three sites | N-1…N-5 (§3 M2), each applied and observed |
| **B1-4** | M3 | **Determinism**: constructing the derived generator for `{x:int, y:int, s:string}` and drawing once from a fresh `newRNG(42)`, **200 times**, yields 200 byte-identical `RecordValue`s | **REFUTED at base**: map-range order randomised — measured 3 distinct orders in a 6-run probe (§F-3) | the **field→value assignment** in the produced `RecordValue`, which is what changes when draw order changes | revert `NewRecordGenerator` to `for k, g := range g.fieldGens` ⇒ the 200 draws diverge |
| **B1-5** | M3 | Anonymous record param `{x: int, y: int}` runs **100 cases**: `examples/runnable/contracts/record_pattern_verify.ail` goes 5 vacuous → 0 vacuous, rc 1 → **0** | measured: `succ=false p/f/s = 0/0/5, vac=5`, all `no generator for parameter p: { x: int, y: int }` | the CLI JSON `vacuous_skips` + per-property `tests_run` | restrict derivation to named `TypeDecl`s only (drop the direct `*ast.RecordType` arm) |
| **B1-6** | M4 | **ADT constructor coverage**: across one property run over a same-file ADT with ≥2 constructors (`park.ail` `Season`, 6 vacuous today), **every** constructor is observed at least once in the generated stream | measured: `park.ail succ=false p/f/s=1/0/7, vac=6`, `no generator for parameter season: Season` ×6 | the multiset of `CtorName`s **actually applied by `ConstructorClosure`/env lookup in the harness**, collected from the derived generator's stream | replace `NewOneOfGenerator(ctors…)` with the first constructor only ⇒ coverage assertion reds. *(This replaces the doc's B-3, whose mutation kills nothing — F-2.)* |
| **B1-7** | M4 | **Nullary-vs-n-ary splice**: a property over an ADT with **both** a nullary and an n-ary constructor (e.g. `Shade = Light \| Dark(level:int)`) runs 100 cases with no evaluation error | not reachable today | the harness's evaluated result — a nullary `FuncCall` dies with `cannot apply non-function value: *eval.TaggedValue` (`eval_operations.go:245`), so the assertion reads the **evaluator's** verdict | make the `TaggedValue` arm always emit `ast.FuncCall` regardless of arity |
| **B1-8** | M3 | **Nested** anonymous records derive: `nested_record_verify.ail` (`{ name: string, pos: { x: int, y: int } }`) 5 vacuous → 0, rc 1 → **0** | measured: `succ=false 0/0/5, vac=5` | CLI `vacuous_skips` + `tests_run` | make the record arm derive only scalar fields (return nil for a `*ast.RecordType` field) |
| **B1-9** | M4 | **Size bound**: for the in-repo mutually-recursive `Doc` (`record_adt_cycle_verify.ail:7-8`), **every** one of 200 derived draws has total node count ≤ a stated constant N, and the file completes under a 60 s bounded wait | measured: `succ=false 0/0/1, vac=1`, `no generator for parameter d: Doc`. `DefaultConfig().MaxSize == 100` measured at `generator.go:30` | the **size of the generated value**, deterministic — not wall clock, so it cannot flake | remove the depth-scaled list cap (restore `0..MaxSize` inside derivation) ⇒ sizes blow past N |
| **B1-10** | M3 | **Imported types stay honest**: `cross_module_types.ail`'s `Cell` ×3 and `list[Tree]` ×1 remain `no_generator` vacuous skips (rc stays 1), while its `()` param starts running | measured: `succ=false 2/0/6, vac=5` — `Cell`×3, `list[Tree]`×1, `()`×1; `Cell`/`Tree` verified **imported** (`cross_module_types.ail:18`) | CLI `vacuous_skips == 4` (was 5) + the surviving skip texts | make an unresolvable named type fall back to a unit-constant generator |
| **B1-11** | M3 | **No false reds**: `cross_module_functions`, `cross_module_functions_lib`, `hof_verify`, `temperature`, `unencodable_callee_skip` all still rc **0** with `vacuous_skips == 0` | measured rc 0 for all five (§1.4) | CLI exit code + `vacuous_skips` | make the record arm return a generator for *any* type (so out-of-contract files acquire vacuous labels) |
| **B1-12** | M4 | **Depth exhaustion is an honest skip, not a hang or a fabrication**: a same-file ADT with **no** non-recursive constructor (`type Endless = Wrap(Endless)`, unit-level fixture) derives to `nil, nil` ⇒ `skip_kind == "no_generator"` | not reachable today | `PropertyResult.SkipKind` written at `contract_domain.go:77` | at the depth bound, return the first constructor anyway ⇒ the derivation recurses past the bound and the test times out / stack-overflows |
| **B1-13** | M4 | **Lane A behaviour preserved**: `hof_verify` (2 pass ×100), `list_verify` (5 pass + 1 deliberate fail), `invoice`'s 2 `list[int]` properties all keep their exact `tests_run` counts | measured §1.4 | per-property `tests_run` in the CLI JSON | apply the depth-scaled size cap to top-level `list[scalar]` too ⇒ list lengths shrink and counts/behaviour drift |
| **B1-14** | M5 | `RecordShrinker`/`TupleShrinker` produce field-/element-wise simplifications and never return the input unchanged | no such shrinkers exist (`grep 'func New.*Shrinker' shrink.go` → Int/Float/String/List/ADT/NoOp only) | the returned `[]eval.Value` | delete the per-field loop ⇒ empty shrink set. **Unit-observable only — F-5** |
| **B1-15** | M6 | `make verify-examples` rc 0 with the new `shapes_verify.ail` in the manifest | **measured rc 0** at base (`186 modules checked, 0 drift`) — an already-green gate, so a red here is a real defect | gate exit code | omit the manifest entry ⇒ `validate_manifest.go --ci` reds |
| **B1-16** | M6 | Every newly-failing property in the after-sweep is classified (a) deliberate / (b) example bug fixed / (c) B1 defect fixed — **zero unclassified** | 111 vacuous today; 87 start running | the after-sweep table vs §1.4 | leave any newly-failing property unexplained |

---

## 5. Regression fixtures

| # | Fixture | Assertion |
|---|---|---|
| 1 | `cross_module_functions_lib.ail` | rc **0**, `vacuous_skips == 0`, its skip stays `out_of_contract` (Lane A's key false-red guard) |
| 2 | `temperature.ail`, `unencodable_callee_skip.ail`, `cross_module_functions.ail` | rc **0** preserved, `vacuous_skips == 0` |
| 3 | `hof_verify.ail` | rc **0**, both `list[int]` properties still `tests_run == 100` (Lane A's A-3, unchanged) |
| 4 | `list_verify.ail` | rc 1 preserved with its **one deliberate** failure; do not "fix" it |
| 5 | `record_verify.ail` | after B1: 15 properties run; `brokenDistance` (`:59`) **fails honestly** — the file header (`:8`) says "Expect: 4 verified, 1 violation" |
| 6 | `inbox_injection_v2.ail`, `inbox_v2_app.ail` | **unchanged** — 10 vacuous each. F-1: B1 does not touch refined or imported types |
| 7 | `per_function_depth_verify.ail` | report **UNMEASURED (exceeds bounded wait)**; never pass/fail. Do not let it block a gate |

---

## 6. Recommended descopes

| Item | Recommendation | Reason |
|---|---|---|
| **M5 derived shrinkers** | **Droppable** — cut if behind after M4 | **F-5**: both contract paths discard the shrinker (`gen, _ :=`); the only consumer is the broken forall path (#624). No downstream observable exists at any level above a unit test |
| **`ADTGenerator` module/type parameterisation** | Keep, but **demote from "prerequisite" to cosmetic** | **F-2**: `TypeName`/`ModulePath` are read by nothing on this path — not `matchPattern`, not `taggedValuesEqual`, not `TaggedValue.String()`, and the splice discards the value entirely |
| **`shapes_verify.ail` example** | **Keep** (unlike Lane A, which skipped its example) | B1's user-visible payoff *is* "these shapes now get tested"; the doc pre-wrote the module. Cost is one manifest entry + an `export func main` |
| **Refined types, imported types, `gen<TypeName>`, hint-text branching (B-9)** | **OUT — already deferred with B2** | Quorum 2026-07-29, blocked on the evaluator fuel budget. 24 of the corpus's 111 vacuous skips stay vacuous by design |

---

## 7. Velocity & sizing

| Milestone | impl LOC | test LOC | est. |
|---|---|---|---|
| M1 splice arms (default still silent) | 90 | 140 | 0.25 d |
| M2 loud default + 5 refusal mutations | 60 | 130 | 0.25 d |
| M3 records / tuples / unit / aliases + determinism fix | 170 | 230 | 0.5 d |
| M4 ADTs + depth budget + size budget + `TypeApp` | 140 | 200 | 0.5 d |
| M5 derived shrinkers (**droppable**) | 90 | 110 | 0.2 d |
| M6 corpus triage + example + closeout | 40 | 60 (+120 docs) | 0.55 d |
| **Total** | **590** | **870** | **2.25 d** |

**Why 2.25 d and not the doc's 1.5–2 d.** The doc's estimate predates Lane A and omits three items this
plan measured into existence:

1. **+0.15 d** — the `RecordGenerator` determinism fix (**F-3**) and its non-flaky 200-draw test. Not in
   the doc; without it the sprint would ship an axiom-A1 violation.
2. **+0.2 d** — the size budget (**F-4**) and its deterministic size assertion. The doc bounds depth
   only, and `MaxSize == 100` over a mutually-recursive in-repo type is a real blow-up.
3. **+0.4 d** — triage. The doc budgeted triage for "the five silent files"; the measured number is
   **87 newly-executing properties across 15 files**, ~5× Lane A's 16 (which cost Lane A 0.4 d).

Offsetting: `list[T]` is already done (Lane A), and the combinator layer already exists — which is why
the total is 2.25 d and not 3.

---

## 8. Open questions for the controller

1. **Depth default 3** is the doc's recommendation and this plan adopts it. If `record_adt_cycle_verify`
   turns out to need depth 4+ to produce interesting `Container` values, the executor should report the
   trade-off rather than silently raising it.
2. **F-1 is a mission-visible correction**: the "prompt-injection safety demos have never executed"
   framing survives into PROGRAM.md-level reporting. B1 does **not** fix them. Whoever writes the
   release note should say so.
