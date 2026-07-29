# Sprint Plan: M-Z3-ADT-RECORD-SORT — Declaration Closure Through Records + Honest `ai-check` Exit

**Design doc**: [m-z3-adt-record-sort.md](m-z3-adt-record-sort.md)
**Sprint ID**: `M-Z3-ADT-RECORD-SORT`
**Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-iter117` (branch `sprint/m-z3-adt-record-sort`, base `origin/dev @ 7e75fa511`)
**Mission iteration**: 117
**Planned**: 2026-07-29
**Duration**: **2.0 working days (16h)** — *reduced from the doc's 3–4 days; justification below*
**Risk Level**: **Low-Medium** (mechanism proven by a live planning spike; residual risk is test surface, not design)

---

## Summary

Close the SMT declaration-closure defect so a contracted function whose parameter is a named
record containing a user ADT is **verified** rather than handed malformed SMT-LIB, and make
`ai-check`'s process exit agree with its own JSON. The planner ran a **throwaway production spike
at HEAD** that reproduces the defect, implements the mechanism, and proves the outcome end-to-end;
the spike was fully reverted before this plan was written. This sprint is therefore an
**implementation-and-hardening** sprint, not a discovery sprint.

**Headline measured result of the spike** (see "Planning Spike" below):
`ai-check` on the doc's own repro goes `errors=1, exit 0` → `verified=1, exit 0`; the examples
corpus goes **verified 76 → 79, skipped 10 → 7**, counterexamples and errors unchanged;
`go test ./internal/smt ./cmd/ailang ./internal/eval_harness` stays **green with zero changes to
existing tests**.

---

## PLANNER REFUTATIONS — read before implementing

The design doc was routed unchanged by Mark's decision (option B, `53d3ac727`). The **direction**
is correct and is not re-litigated. But five load-bearing premises about *where the defect lives*
are wrong, and one is wrong in a way that would have made M1 **not fix the doc's own repro**.

### R-1 (BLOCKING) — The defect is TWO layers, not one. M1 as written cannot fix the repro.

The doc localises the bug entirely inside `internal/smt/codegen.go` Step 0. That is **half** of it.

There is a **second, independent** drop in the CLI driver. `filterADTTypesForFunction`
(`cmd/ailang/verify_filter.go:230`) seeds from params/return/body and walks the transitive closure
**only through ADT variant fields — never through record-alias fields**. For
`hasName(p: Proposal, n: string)` the seed set is `{Proposal}`; `Proposal` is an alias, not an ADT,
so the closure terminates and `funcADTTypes` is **empty**. `Evidence`'s variant list is never handed
to `EncodeFunction`. **No encoder-side graph can emit a datatype whose body it was never given.**

Measured proof (worktree HEAD, `./bin/ailang`): adding `e: Evidence` as an extra parameter — which
makes the driver seed `Evidence` — changes the emitted SMT from *no declarations at all* to
`(declare-datatype Evidence ...)` while `Proposal` is **still** missing. That isolates the two
layers cleanly:

| Fixture | Driver supplies `Evidence`? | HEAD result |
|---|---|---|
| `/tmp/i117/adtsort.ail` (doc's repro) | no | **no declarations at all**; Z3 `unknown sort 'Proposal'` |
| `/tmp/i117/layer.ail` (`+ e: Evidence` param) | yes | `Evidence` declared, `Proposal` **still** missing |

**Consequence for the plan:** M1 must change `cmd/ailang/verify_filter.go` / `verify.go` as well
as `internal/smt/codegen.go`. The doc's "Files Expected to Change" table does **not list
`verify_filter.go` at all** and allots `verify.go` only `+5/−5` "neutral skip reason". That table
is wrong and is superseded by this plan.

### R-2 (BLOCKING) — `ai-check` and `verify` are NOT the same driver. `ai-check` has **no** demand filtering, and it is measurably worse today.

`cmd/ailang/verify.go:383-396` computes `buildNeededSortSet` → `filterExtraDeclarationsForFunction`
→ `widenedAliases`. `cmd/ailang/ai_check.go`'s `runVerification` does **only**
`filterADTTypesForFunction` and passes the **full, unfiltered** `recordAliases` and the **full**
`adtRecordDecls` (`encOpts.ExtraDeclarations = adtRecordDecls`, set once at line 238 and never
narrowed per function). `rg "buildNeededSortSet" cmd/ailang/ai_check.go` → **zero hits**.

This is not theoretical. Measured at HEAD on one file (`/tmp/i117/casc.ail`, a `Doc` alias plus a
`Block ↔ Record_blocks_kind` cyclic ADT plus a trivial `inc(x: int) -> int`):

```
verify   : inc VERIFIED, titleOf ERROR (Z3 unknown sort 'Doc')
ai-check : BOTH skipped/UNENCODABLE_TYPE  — including inc(x:int)->int
```

`ai-check` poisons an unrelated primitive-only function. This is precisely the cascade
`verify_filter.go`'s own header comment says the filters exist to prevent — the fix was never
ported to `ai-check`.

**This also refutes the doc's Impact framing.** The doc says the v1.0 KPI is "not inflated". True,
but it misses that the KPI is **deflated**: `internal/observatory/cost_per_verified_success.go:99-104`
requires `VerifySkipped == 0`, so every cascade skip that `verify` would have verified turns a run
into a non-verified-success. The eval harness calls `RunAICheck` — i.e. **the broken driver** — so
this gap sits directly under the headline v1.0 metric.

**Consequence for the plan:** driver parity is **mandatory and in M1**, not a nice-to-have. Without
it the sprint's headline AC ("`ai-check` verifies the repro") is **unreachable** — proven: with the
encoder fix and the `verify.go` fix in place but `ai_check.go` untouched, `ai-check` still reported
`skipped=1` on all three record fixtures.

### R-3 — There is a THIRD ordering bug: `buildNeededSortSet` is fed the already-narrowed ADT map.

`verify.go:386` passes `funcADTTypes` (the output of `filterADTTypesForFunction`) into
`buildNeededSortSet`. The comment two lines above explicitly says *"buildNeededSortSet must see the
FULL alias map (not the filtered one)"* — they fixed that for **aliases** and not for **ADTs**.
Result: an inline record reachable only through an ADT variant field (the canonical
`Block → Record_blocks_kind`) is never added to `needed`, so
`filterExtraDeclarationsForFunction` drops it. Measured with an instrumented build:

```
DBG titleOf needed=map[Block:true Doc:true] extra=[]      <- Record_blocks_kind missing
```

Passing the full `adtTypes` fixes it, and the cycle then emits **one plural
`declare-datatypes` group that Z3 accepts** (measured, see spike).

### R-4 — The doc's V6 conclusion is wrong, and the code comment it dismisses is NOT cargo-cult.

V6 reports the canonical `ParsedDocument → Block → Record_blocks_kind → Block` cycle was *"not
found in the `.ail` corpus"* and concludes the Step-0 comment's circular-dependency justification is
"unverified and must not justify exclusion". **The cycle is trivially expressible in ordinary
AILANG and the planner constructed one in 4 lines:**

```ailang
export type Block = Para(string) | Container({ blocks: list[Block], kind: string })
export type Doc   = { title: string, blocks: list[Block] }
```

`Container`'s inline record becomes `Record_blocks_kind`, which references `Block` → genuine
2-node SCC. Absence from `examples/` is a coverage gap in the corpus, not evidence about the
language. **M1 AC4 (plural-group emission for a real cycle) is therefore mandatory, and the cyclic
fixture must be a first-class `.ail` example, not a hand-built decl list.**

### R-5 — `go build ./...` is ALREADY RED at HEAD. The doc's Sprint Exit criterion is unusable as written.

```
$ go build ./...
# github.com/sunholo-data/ailang/cmd/wasm
runtime.main_main·f: function main is undeclared in the main package
```

`cmd/wasm` is `//go:build js && wasm`-gated; on darwin the package has no `main`. This is
pre-existing and unrelated. **Sprint Exit must say `go build ./cmd/ailang ./internal/...` or
`go vet` on the touched packages**, otherwise the executor will chase a phantom.

### R-6 — The doc's own A5 / "Quorum round 2 — PARKED" section is STALE and must be corrected in this sprint.

`#510` landed as `9253ec8a8` (in this worktree's history). `internal/smt/solver.go` now uses
`exec.CommandContext` + `cmd.WaitDelay` + a `hardTimeout = max(config.Timeout, effectiveTimeout) +
solverKillGrace` deadline, on **both** `Solve` and `Z3Version()`. The doc's A5 justification, its
last Risks row ("Residual risk: production uses `exec.Command`/`CombinedOutput`"), and the entire
"Quorum round 2 — PARKED" block describe code that no longer exists. Doc closeout is a milestone
deliverable (M4), not optional.

### Premises the planner CONFIRMED first-party (do not re-verify)

| ID | Claim | Status |
|---|---|---|
| V1 | repro checks clean, `ai-check` reports `errors=1` and exits 0 | **confirmed** (`rc=0`, `errors:1`) |
| V4 | Step-0 `if !allFieldsPrimitiveOrDeclared { continue }` + `if !progress { break }` silently abandons | **confirmed**, `codegen.go:240,259` |
| V9-second-drop | `if err != nil { delete(remaining, aliasName); continue }` is a second silent drop | **confirmed**, `codegen.go:233-236` |
| V5 | second fixpoint is NOT a silent give-up; routes to `findSCCs`/`DeclareDatatypesMutual` | **confirmed**, `codegen.go:317-355` |
| V8 | Tarjan SCC + plural emitter exist and work | **confirmed** by code read **and by live spike** (Z3 accepted the emitted plural group) |
| V11 | `RejectUnencodable = "UNENCODABLE_TYPE"`, `ErrUnresolvableTypes` → skip | **confirmed**, `encodable.go:22` |
| V12 | `validateDeclarations` waves `declare-const` through | **confirmed**, `codegen.go:705` verbatim comment |
| V13 | `ai-check` exit ignores `Errors`; JSON written first | **confirmed**, `ai_check.go:161-163` |
| V14 | no in-repo Makefile/CI/shell consumer of `ai-check` | **confirmed** — only `cmd/ailang/*`, `internal/eval_harness/*`, and `prompts/agent/convergence-workflow.md:72` |
| V15/V16 | `RunAICheck` parses JSON from a nonzero child (`rawOutput == "" && g.waitErr != nil`); comment at `verify.go:80` is inverted | **confirmed**, `internal/eval_harness/verify.go:79-84` |
| V17 | `verify` has `-strict` (3 hits), `ai-check` has none (0 hits) | **confirmed** |
| V19/A5 | solver bounded — **now by a hard wall-clock deadline**, not only `-T:` | **confirmed, upgraded** (see R-6) |
| V20 | `(Seq X)` is the only parameterized sort `MapType` emits | **confirmed**, `types.go:35-70`; `TTuple` falls to the error default |

### Premises the planner found IRRELEVANT (drop them from the risk register)

- **No SMT golden files exist.** The only `.golden` files near this diff are
  `cmd/ailang/testdata/debug_ast_simple.golden` and
  `cmd/ailang/testdata/ext_registry_gen/two_packages.ail.golden`; neither contains SMT-LIB. The
  "stale golden broke `make test`" hazard from iteration 13 **does not apply to this diff**.
- **`examples/manifest.json` does not record verify counts.** Entries carry `path`/`status`/`tags`
  only. Flipping a contract example from `skipped` to `verified` cannot break
  `validate_manifest --ci`.

---

## Planning Spike — the mechanism is proven, not hypothesised

A throwaway spike was applied at HEAD, measured, and **fully reverted** (`git status` clean,
`go build ./cmd/ailang` green after revert). It is reproduced here because it converts most of M1
from design work into transcription work.

**Spike diff: `+65 / −42` across 3 files** (`internal/smt/codegen.go`, `cmd/ailang/verify.go`,
`cmd/ailang/ai_check.go`).

Three changes:

1. **`internal/smt/codegen.go` Step 0** — replace the silent fixpoint with: iterate aliases in
   **sorted name order**, register `activeRecordTypes` / `activeFieldSetToSort` **eagerly for every
   mappable alias**, and stage `DeclareRecordDatatype(...)` output into a slice that is appended to
   the **existing** Step-0.5/1 `pending` list. No new graph, no new SCC code, no new emitter.
2. **`cmd/ailang/verify.go`** — pass the **full** `adtTypes` to `buildNeededSortSet` (R-3), then
   widen `funcADTTypes` with every ADT in `needed` (R-1).
3. **`cmd/ailang/ai_check.go`** — port verify.go's demand-filter block verbatim (R-2).

Measured outcomes:

| Fixture | HEAD `verify` | HEAD `ai-check` | Spike `verify` | Spike `ai-check` |
|---|---|---|---|---|
| `adtsort.ail` (doc repro) | Z3 error | `errors=1` | **verified** | **verified=1** |
| `multi.ail` (4 aliases/ADTs) | Z3 error | `errors=1` | **verified** | **verified=1** |
| `casc.ail` (real `Block ↔ Record_blocks_kind` cycle + unrelated `inc`) | 1 verified, 1 error | **2 skipped** | 2 verified | **verified=2** |
| `layer.ail` | Z3 error | `errors=1` | **verified** | **verified=1** |

Emitted SMT for the repro (dependency-first, exactly as the doc's Solution Design predicts):

```smt2
(declare-datatype Evidence ((CompilerOutput (CompilerOutput_0 String)) (TestReport (TestReport_0 String) (TestReport_1 Bool))))
(declare-datatype Proposal ((mk_Proposal (evidence (Seq Evidence)) (name String))))
(declare-const $p_p Proposal)
```

Emitted SMT for the genuine cycle — **one plural group, Z3 `unsat`, function VERIFIED**:

```smt2
(declare-datatypes ((Record_blocks_kind 0) (Block 0)) (((mk_Record_blocks_kind (blocks (Seq Block)) (kind String))) ((Para (Para_0 String)) (Container (Container_0 Record_blocks_kind)))))
(declare-datatype Doc ((mk_Doc (blocks (Seq Block)) (title String))))
```

Regression evidence under the spike:

- `go test ./internal/smt ./cmd/ailang ./internal/eval_harness` → **all ok**, **no existing test edited**.
- Examples-corpus sweep (`ai-check` over all 40 contract-bearing `.ail` under `examples/` + `stdlib/`),
  HEAD vs spike: **verified 76 → 79, counterexamples 23 → 23, skipped 10 → 7, errors 3 → 3**.
  The single file that moved is `examples/runnable/contracts/cross_module_types.ail`
  (`v=1 s=4` → `v=4 s=1`). Nothing regressed.

**The spike is a sizing artefact, not a deliverable.** The executor must re-derive it with tests
first (TDD), name the helpers properly, and add the failure paths the spike skipped (M2).

---

## Velocity

- Last 7 days: **146 commits**, 583 files, +86,225/−5,546 (dominated by generated/eval data — not a
  usable LOC/day signal).
- Comparable compiler-surface sprints (`m-smt-callee-sort-gate`, `m-z3-hard-timeout` in iteration
  116) landed in **~1 day each**.
- **Basis for this estimate:** production change is *measured* at ~65 insertions. The cost is
  **tests, fixtures and the doc closeout**, not implementation.

**Estimate: 2.0 days (16h)** — M1 6h, M2 4h, M3 3h, M4 2h, +1h buffer.

### Scope cut vs the doc's 3–4 days — what was cut and why

| Doc item | Decision | Why |
|---|---|---|
| "Build a **new** declaration graph with node structs, sort-reference collector, SCC dispatch" (D3/M1) | **CUT — reuse the existing Step-0.5/1 pending list + `declReferencesUndeclaredSort` + `findSCCs`** | The spike proves the existing machinery already does dependency-ordering and plural-group emission correctly once aliases are fed into it. Building a parallel graph is ~200 LOC of duplicate mechanism for zero measured benefit. |
| M1 AC6 "sort-reference collector rejects `(Future X)`" | **KEPT but retargeted** to the two extraction points that actually exist (`isSortPrimitiveOrDeclared`, and the new `declare-const` sort parser in M2) | There is no new collector to write. The fail-closed obligation is real; the new component is not. |
| M1 AC7 "adversarial recursive-datatype timeout regression" | **CUT from this sprint** | `#510` (`9253ec8a8`) landed a hard wall-clock deadline **with its own tests** in iteration 116. Re-testing it here is duplicate coverage of another sprint's ACs. Recorded as R-6; the doc text is corrected in M4 instead. |
| D4 option C "full SMT-LIB parser" | **stays deferred** (doc already defers it) | — |
| **ADDED (not in the doc): driver parity + ADT-closure widening** | **ADDED to M1** | R-1/R-2/R-3. Without it the sprint's headline AC is unreachable. |
| **ADDED: doc closeout of the stale A5 / PARKED section** | **ADDED as M4** | R-6. |

Net: the doc's 3–4 days assumed writing a new graph engine. It isn't needed. The added driver work
is smaller than the graph work removed.

---

## Milestones

> **Sandbox note applies to M1 (partly), M2 (partly), M3 (wholly), M4 (no).**
> Every milestone below that touches `cmd/ailang` is **UNINFORMATIVE UNDER THE EXECUTOR'S
> `workspace-write` SANDBOX** if its verification spawns a subprocess or binds a socket. Those ACs
> are tagged **[CONTROLLER-RERUN]** and the controller must re-run them outside the sandbox before
> Gate 3b. The executor must still write them and must record `UNINFORMATIVE UNDER SANDBOX`
> verbatim rather than reporting a pass or a failure.

---

### M1 — Declaration closure: encoder graph unification + driver demand parity (6h)

**Goal.** A contracted function whose parameter is a named record reaching a user ADT (directly,
under `list`, through another record, or through a genuine record↔ADT cycle) is **verified** by
**both** `verify` and `ai-check`.

**Production changes (fixed; helper names are implementer discretion):**

1. `internal/smt/codegen.go` — Step 0 becomes *eager metadata registration + deferred emission*:
   - iterate `opts[0].RecordTypeAliases` in **sorted name order** (`sort.Strings`), never by map range;
   - for every alias whose `MapRecordFields` succeeds, register `activeRecordTypes` and
     `activeFieldSetToSort` **before** `collectAndDeclareRecordTypes` runs;
   - stage `DeclareRecordDatatype(...)` into the Step-0.5/1 `pending` list instead of appending to
     `result.Declarations`;
   - a `MapRecordFields` **error** must no longer be a bare `continue` — it must be recorded so M2
     can turn a *needed* unmappable alias into `ErrUnresolvableTypes` (V9 second silent drop).
2. `cmd/ailang/verify.go` — pass the **full** `adtTypes` (not `funcADTTypes`) to
   `buildNeededSortSet`; widen `funcADTTypes` with every ADT in `needed`.
3. `cmd/ailang/ai_check.go` — port the demand-filter block so `ai-check` and `verify` compute
   identical `ExtraDeclarations`, `RecordTypeAliases` and ADT sets. **Preferred: extract the shared
   block into one function in `cmd/ailang/verify_filter.go` and call it from both drivers**, so the
   two can never drift again (CLAUDE.md Principle 3: one unified fix).

**File-size constraint (HARD, CI gate).** `internal/smt/codegen.go` is **759 lines**;
`make check-file-sizes` fails at **>800** and runs in `.github/workflows/ci.yml:97`. That is **41
lines of headroom**. The doc's `+100/−60` estimate lands at 799 — one line from red. **Any new
helper longer than a few lines goes in a new file** (suggested `internal/smt/codegen_decl_graph.go`).
`cmd/ailang/verify.go` is 672 (headroom 128); `ai_check.go` is 420.

#### Acceptance Criteria

**AC1.1 — the doc's repro verifies under BOTH drivers.**
A committed fixture equal to the doc's `adtsort.ail` yields `verified=1, counterexample=0,
skipped=0, errors=0` from `ai-check`, and `1 verified` from `verify`.
**Red mutation:** in production `codegen.go`, restore `if !allFieldsPrimitiveOrDeclared(fieldSorts, ctx) { continue }`
before staging. Builds; the assertion must fail with a non-verified result.
**[CONTROLLER-RERUN]** if asserted through the CLI subprocess; the `EncodeFunction`-level half is
sandbox-safe.

**AC1.2 — the driver widens the ADT closure through record-alias fields.**
A unit test on `filterADTTypesForFunction` (or its replacement) proves that for
`params=[p: Proposal]` with `aliases={Proposal:{evidence: list[Evidence]}}` and
`adtTypes={Evidence:...}`, the returned map **contains `Evidence`**.
**Red mutation:** remove the alias-field walk from the production closure. Builds; the unit test
must fail. *(This is the R-1 guard and it is the single most important test in the sprint —
without it a future refactor silently re-breaks the repro at the driver.)*
**Sandbox-safe.**

**AC1.3 — `ai-check` and `verify` agree, function for function.**
A table-driven test runs both code paths over ≥4 committed fixtures (repro, list-of-ADT,
nested record→record→ADT, and the cascade file with an unrelated `inc(x:int)->int`) and asserts
**identical per-function status**. The cascade file must show `inc` **verified** under `ai-check`.
**Red mutation:** in production `ai_check.go`, restore `funcEncOpts.ExtraDeclarations = adtRecordDecls`
(unfiltered). Builds; `inc` must flip to `skipped` and the parity assertion must fail.
**[CONTROLLER-RERUN]** if driven through the CLI; prefer calling `runVerification` in-process to
keep it sandbox-informative.

**AC1.4 — a genuine record↔ADT cycle emits ONE plural group Z3 accepts.**
A committed `.ail` fixture containing `type Block = Para(string) | Container({blocks: list[Block], kind: string})`
plus an alias over `list[Block]` verifies, and the emitted SMT contains exactly one
`(declare-datatypes ((` group naming both member sorts.
**Red mutation:** in production SCC dispatch, emit each SCC member as a separate singular
`declare-datatype`. Builds; Z3 must reject the forward reference and the AC must fail.
**Sandbox-safe** at the `EncodeFunction` level.

**AC1.5 — byte-identical SMT across repeated encodes.**
Encode a fixture with ≥3 record aliases and ≥2 ADTs **N ≥ 50** times in one test and assert all
outputs are byte-identical.
**Red mutation:** replace the `sort.Strings(aliasNames)` iteration with a bare `range` over
`opts[0].RecordTypeAliases`. Builds; with N=50 the test must fail. *(Note: this AC is **already
red at HEAD** for the alias pass — the current Step-0 loop ranges a Go map. It is a bug fix, not
only a guard. The executor must confirm it fails before the fix and passes after.)*
**Sandbox-safe.**

**AC1.6 — controls do not regress.**
`examples/runnable/contracts/*.ail` sweep: `verified` must be **≥ 79**, `errors` **≤ 3**,
`counterexample` **== 23**, `skipped` **≤ 7** (baseline HEAD: 76 / 3 / 23 / 10; spike: 79 / 3 / 23 / 7).
The three controller control fixtures (`list[string]` record, direct ADT param, pattern-matched
ADT param) must be **committed as `.ail` fixtures in-repo** and must stay verified — the doc's
`/tmp/iter115repro/*` paths must not appear in any test.
**Red mutation:** n/a (this is a corpus control, not a unit gate) — instead the executor must
record the before/after numbers in the commit message.
**[CONTROLLER-RERUN]** (runs the CLI binary over the corpus).

---

### M2 — Fail-loud declaration gate (4h)

**Goal.** No constant with an unresolved nonprimitive sort can reach Z3. When closure cannot be
formed, the driver reports a structured `skipped` / `UNENCODABLE_TYPE`, never a raw Z3 error.

**Production changes:**

1. `internal/smt/codegen.go` `validateDeclarations` — remove the `declare-const`/`define-const`
   free pass at line ~705. Parse the sort of `(declare-const NAME SORT)` and typed
   `(define-const NAME SORT ...)` and require it to be primitive, `(Seq <resolved>)` recursively, or
   already declared **at that position in the list**.
2. `validateDeclarations` must recognise the **plural** form. Today `strings.HasPrefix(decl, "(declare-datatype ")`
   (trailing space) **does not match** `"(declare-datatypes ("`, so a plural group is invisible to the
   validator **both as a declarer and as a consumer**. Plural members must be installed **atomically**.
3. Unmappable-but-**needed** aliases (V9 second silent drop) return wrapped `ErrUnresolvableTypes`
   naming the alias and the offending field, instead of `delete(...); continue`.
4. Both drivers: replace the literal reason
   `"Uses cross-module types not yet supported in Z3 encoding (%v)"` and the hint
   `"Cross-module record type aliases and recursive ADTs are not yet supported..."` with neutral,
   accurate wording that names the unresolved sort and does **not** assert a cross-module cause.
   These strings are duplicated verbatim in `verify.go:403-412` and `ai_check.go:349-358` — **fix
   both from one shared constant/helper**.

#### Acceptance Criteria

**AC2.1 — dangling `declare-const` sort is rejected before the solver.**
A unit test passes a hand-built decl list containing `(declare-const $p_p Missing)` to
`validateDeclarations`; it returns an error satisfying `errors.Is(err, ErrUnresolvableTypes)` whose
message names both `$p_p` and `Missing`. A companion test asserts `smt.Solve` is never reached
(assert on the `EncodeFunction` error return, not on a solver spy).
**Red mutation:** restore `if !strings.HasPrefix(decl, "(declare-datatype ") { continue }` ahead of
the const check. Builds; the test must fail. **Sandbox-safe.**

**AC2.2 — `(Seq X)` is validated recursively and positionally.**
`(declare-const xs (Seq Evidence))` passes **after** `Evidence` is declared and fails **before** it.
**Red mutation:** in the production sort parser, accept any `(Seq ...)` without recursing.
Builds; the before-declaration case must fail to be rejected. **Sandbox-safe.**

**AC2.3 — plural groups declare all members atomically.**
A decl list `[(declare-datatypes ((A 0) (B 0)) (...)), (declare-const x A), (declare-const y B)]`
validates clean; the same list with `(declare-const z C)` appended is rejected naming `C`.
**Red mutation:** in production plural handling, register only the first member sort. Builds; the
`y B` case must be wrongly rejected and the test must fail. **Sandbox-safe.**
*(Note: this AC is red at HEAD in a second way — plural decls currently bypass validation entirely.)*

**AC2.4 — an unresolvable shape becomes a structured skip, never a Z3 error.**
An integration fixture the encoder genuinely cannot close reports `status:"skipped"` with
rejection code `UNENCODABLE_TYPE`, and its `reason` contains **no** Z3 text (`assert !strings.Contains(reason, "Z3")`).
**Red mutation:** in production `EncodeFunction`, swallow the `ErrUnresolvableTypes` return and
continue assembling. Builds; the fixture must surface `status:"error"` + Z3 text and fail.
**[CONTROLLER-RERUN]** if driven through the CLI.

**AC2.5 — neutral skip wording in BOTH drivers, from one source.**
A table-driven test over `verify` and `ai-check` asserts the skip reason names the unresolved sort
and does **not** contain the substring `cross-module`.
**Red mutation:** restore the literal `Uses cross-module types not yet supported` in **either**
driver. Builds; the corresponding table row must fail. **Sandbox-safe** if asserted on the
reason-building helper rather than on process output.

---

### M3 — Honest `ai-check` exit (3h) — **WHOLLY [CONTROLLER-RERUN]**

**Goal.** `ai-check`'s process status agrees with its own JSON, without breaking the eval harness.

**Production changes:**

1. `cmd/ailang/ai_check.go:161-163` → `if !checkSection.Passed || verifySection.Counterexample > 0 || verifySection.Errors > 0 { os.Exit(1) }`.
   `outputAICheck(output)` **stays above** the exit. No new flag (design decision D1-C).
2. `internal/eval_harness/verify.go:79` — correct the inverted comment. Behaviour unchanged.

**Breaking-change assessment (the doc asks; here is the answer).**
`ai-check` is listed **Experimental** in `docs/docs/reference/stability.md:163` ("JSON shape still
stabilising"), so the exit-code correction is **permitted without a compatibility flag**. In-repo
consumers (re-verified, R-14 above): `internal/eval_harness.RunAICheck` tolerates a nonzero exit by
construction (`if rawOutput == "" && g.waitErr != nil`), and `prompts/agent/convergence-workflow.md:72`
tells agents `ai-check` is the green signal — which is exactly the contract being *repaired*. There
is **no Makefile, CI job, or shell script** that runs `ai-check`. **Verdict: no `-strict` flag, no
migration shim; a CHANGELOG "Changed (breaking for out-of-repo shell callers)" entry is required.**

#### Acceptance Criteria

**AC3.1 — errors exit 1 with complete JSON.**
A subprocess test builds the CLI, runs `ai-check` on a fixture producing `verify.errors > 0`,
asserts exit status 1 **and** that stdout parses as complete JSON with `verify.errors > 0`.
**Red mutation:** delete `|| verifySection.Errors > 0`. Builds; the test observes exit 0 and fails.
**[CONTROLLER-RERUN].**

> **Fixture hazard — read this.** After M1+M2 the record repro is **verified** and the previous
> unresolvable shapes become **skips**, so `verify.errors > 0` is *hard to produce naturally*. The
> executor must **not** reach for the repaired record shape. Use a genuine solver-error path (e.g.
> a fixture whose `Solve` returns an error, or an injected `aiVerifySection` in an in-process test
> plus one real subprocess test on any fixture that still errors — `examples/runnable/contracts/cross_module_functions.ail`
> currently reports `errors=3` and is the natural candidate; **re-confirm it still errors after M1/M2**
> before relying on it).

**AC3.2 — the other four lanes are unchanged.**
counterexample → exit 1; skip-only → exit 0; verified-only → exit 0; check-failure → exit 1.
**Red mutation:** replace `Errors > 0` with `Skipped > 0`; the skip-only row must turn red.
Separately remove `Counterexample > 0`; the counterexample row must turn red.
**[CONTROLLER-RERUN].**

**AC3.3 — JSON precedes the exit on every exit-1 path.**
**Red mutation:** move `outputAICheck(output)` below the `os.Exit(1)` branch. Builds; the
subprocess test receives empty stdout and fails. **[CONTROLLER-RERUN].**

**AC3.4 — `RunAICheck` still parses a nonzero child (V15 re-verified empirically, permanently).**
A test in `internal/eval_harness` drives the **real** `RunAICheck` against a child that writes valid
`ai-check` JSON with `verify.errors: 1` and then exits 1; asserts `err == nil` and
`result.Verify.Errors == 1`.
**Red mutation:** change the guard to `if g.waitErr != nil`. Builds; the test must fail.
**[CONTROLLER-RERUN]** (spawns a subprocess). *Prefer a `go test`-compiled helper binary or a
`TestHelperProcess` pattern over a shell script, so it works on any PATH.*

**AC3.5 — the inverted comment is corrected.**
`internal/eval_harness/verify.go:79` states that counterexamples **and verifier errors** are
nonzero, and that nonempty stdout is still parsed. Behavioural ACs above remain the gate; the
comment is documentation, not a test.

---

### M4 — Doc closeout, CHANGELOG, corpus coverage (2h) — sandbox-safe

- **Correct the design doc** (R-6): delete/replace the "Quorum round 2 — PARKED" block with a
  one-paragraph resolution note pointing at `#510` / `9253ec8a8`; rewrite the A5 row to cite
  `exec.CommandContext` + `WaitDelay` + `hardTimeout`; delete the stale last Risks row; correct the
  V6 row per R-4; correct the "Files Expected to Change" table per R-1/R-2/R-3; correct the Sprint
  Exit `go build ./...` criterion per R-5.
- **Add the record→ADT and cyclic shapes to `examples/runnable/contracts/`** with manifest entries
  (`status: working`, tags including `contracts`, `smt-verification`). This is the doc's
  "Programs and tests that must remain valid" list turned into permanent in-repo controls, and it
  closes the corpus gap R-4 exposed.
- **CHANGELOG.md** — three entries: (a) *Fixed*: records reaching user ADTs now verify;
  (b) *Fixed*: `ai-check` no longer cascades unrelated functions into skips (driver parity with
  `verify`) — call out the KPI impact; (c) *Changed (breaking for out-of-repo shell callers)*:
  `ai-check` exits 1 when `verify.errors > 0`.

#### Acceptance Criteria

**AC4.1** — `make check-file-sizes` green (`internal/smt/codegen.go` **≤ 800**).
**AC4.2** — `make check-boundaries` green (no new cross-layer import; `cmd/ailang` → `internal/smt`
is existing and allowed).
**AC4.3** — `make verify-examples` green with the new examples in `examples/manifest.json`
(regenerate/patch statistics — hand-maintained block, known CI trap).
**AC4.4** — the design doc contains **no** occurrence of `CombinedOutput` and **no**
`Quorum round 2 — PARKED` heading.
**AC4.5** — `gofmt -l` clean and `make lint` green on touched packages.

---

## Conflict Surface (MANDATORY — compiler surface)

### Direct blast radius

| Surface | Who else uses it | Risk | Guard |
|---|---|---|---|
| `EncodeFunction` Step 0 | **only** `cmd/ailang/verify.go` + `cmd/ailang/ai_check.go` (`rg EncodeFunction` → 2 non-test callers) | Low | AC1.3 parity table |
| `validateDeclarations` | called once at the end of `EncodeFunction`; **M-SMT-CALLEE-SORT-GATE**'s `define-fun` branch lives in the same function | **Medium — do not weaken the `define-fun` branch**; a careless rewrite reopens the callee-sort leak | AC2.1-2.3 must be **added to**, not replace, `codegen_test.go`'s existing callee-gate tests |
| `findSCCs` / `DeclareDatatypesMutual` | `internal/smt/codegen.go` only; tested in `codegen_mutual_test.go` | Low — this sprint **adds inputs**, changes no algorithm | existing `codegen_mutual_test.go` must stay green untouched |
| `buildNeededSortSet` / `filterADTTypesForFunction` / `filterExtraDeclarationsForFunction` | `verify.go` today, `ai_check.go` after M1 | **Medium — widening the needed-set is the exact thing M-SMT-CROSS-MODULE-TYPES narrowed** | AC1.3's cascade fixture (`inc(x:int)->int` in a module with a poisoned type) is the anti-regression; AC1.6 corpus sweep is the population check |
| `activeRecordTypes` / `activeFieldSetToSort` (package-level mutable state) | `encodeRecord`, `encodeRecordUpdate`, `collectAndDeclareRecordTypes` | **Medium** — eager registration now happens for aliases that are *never emitted*; a record literal could resolve to a named sort with no declaration | AC2.1's `declare-const` gate catches it as a structured skip rather than a Z3 error; `codegen_record_discovery_test.go` + `codegen_nested_records_test.go` must stay green |
| `RunAICheck` | `agent_runner.go`, `eval_suite.go --verify`, `agentAICheck` var | Low — proven tolerant of nonzero exit | AC3.4 |

### What could break that is easy to miss

1. **`internal/smt/codegen.go` at 759/800.** The CI gate is 41 lines away. This is the most likely
   accidental red in the sprint.
2. **Existing `internal/smt` tests that construct `EncodeFunctionOpts{RecordTypeAliases: ...}` and
   assert a declaration appears at a fixed index** in `result.Declarations`. Deferring alias
   emission from Step 0 into the Step-0.5/1 pass **changes declaration ORDER**. The spike happened
   not to trip any (`go test ./internal/smt` green), but an added test could. Assert on *set
   membership and relative order*, never absolute index.
3. **`examples/runnable/contracts/cross_module_types.ail` changes behaviour** (`v=1 s=4` → `v=4 s=1`).
   If any test or doc asserts its skip count, it must be updated. None found — re-check after M1.
4. **Banked eval data.** `eval_results/` is gitignored; no committed baseline pins verify counts.
   But `docs/static/benchmarks/*.json` are committed — **do not regenerate them in this sprint**;
   the coverage change will shift future `cost_per_verified_success` numbers and that must be an
   observed post-release effect, not a hand-edited one.
5. **`ai-check` skip→verified is a KPI-affecting change.** `isVerifiedSuccess` requires
   `VerifySkipped == 0`; M1 will move runs from "not a verified success" into "verified success".
   That is the *intent*, but the next eval baseline will show a **discontinuity**. Flag it in the
   CHANGELOG so the next benchmark comparison is not read as model improvement.
6. **Duplicate `ExtraDeclarations`.** Observed during the spike: `adtRecordDecls` contained
   `Record_blocks_kind` **twice** for one module. Harmless today (the pending loop skips already-
   declared sorts) but it will produce a duplicate node if a real graph is built. Dedup by sort
   name when staging.
7. **NOT a risk (verified):** no SMT golden files; `examples/manifest.json` records no verify
   counts; `make check-boundaries` unaffected (no new package edges).

---

## Files Expected to Change (supersedes the design doc's table)

| File | Purpose | Est. |
|---|---|---:|
| `internal/smt/codegen.go` | Step-0 eager-register/defer-emit; `validateDeclarations` const + plural handling. **Keep ≤ 800 lines.** | +45/−45 |
| `internal/smt/codegen_decl_graph.go` *(new, if needed)* | overflow home for any helper that would push `codegen.go` over the gate | +0-60 |
| `cmd/ailang/verify_filter.go` | ADT closure through alias fields; **shared** demand-filter helper for both drivers | +30/−5 |
| `cmd/ailang/verify.go` | call the shared helper; neutral skip reason from a shared constant | +10/−12 |
| `cmd/ailang/ai_check.go` | call the shared helper (**new capability**); `Errors > 0` exit; neutral skip reason | +12/−8 |
| `internal/smt/codegen_*_test.go` | closure, cycle, determinism (N≥50), leak-gate, plural-group tests | +300 |
| `cmd/ailang/*_test.go` | driver parity table; exit-code subprocess tests | +180 |
| `internal/eval_harness/verify.go` | inverted comment only | +2/−2 |
| `internal/eval_harness/*_test.go` | nonzero-child JSON parse test | +60 |
| `examples/runnable/contracts/*.ail` + `examples/manifest.json` | committed fixtures for the 3 controls + record→ADT + cycle | +80 |
| `design_docs/planned/v0_31_0/m-z3-adt-record-sort.md` | R-6/R-4/R-1/R-5 closeout | +25/−45 |
| `CHANGELOG.md` | 3 entries | +14 |

**Total ≈ 820 LOC changed, of which ~110 is production.** (M1 340 · M2 200 · M3 160 · M4 120.)

---

## Sprint Exit

Complete only when **all** hold:

1. M1–M4 acceptance criteria pass, with every listed **red mutation** demonstrated to *build and
   then fail* (a mutation that fails to compile is **not** a mutation proof).
2. `go test ./internal/smt/... ./cmd/ailang/... ./internal/eval_harness/...` green.
3. `go vet ./internal/smt/... ./cmd/ailang/...` and `go build ./cmd/ailang` green.
   **Do NOT gate on `go build ./...` — it is red at HEAD for `cmd/wasm` (R-5).**
4. `make check-file-sizes`, `make check-boundaries`, `make verify-examples`, `make lint` green.
5. Corpus sweep recorded in the commit message: `verified ≥ 79`, `errors ≤ 3`, `counterexample == 23`,
   `skipped ≤ 7`.
6. CHANGELOG documents the coverage expansion, the `ai-check` driver-parity fix, and the exit-code
   correction (marked breaking for out-of-repo shell callers).
7. Any sandbox-blocked AC recorded verbatim as **UNINFORMATIVE UNDER SANDBOX** — never as a pass.

---

## Things the executor must NOT do

- **Do NOT build a new declaration-graph engine.** D3-C is satisfied by feeding record aliases into
  the *existing* Step-0.5/1 pending list. Measured: it already does dependency ordering and plural
  SCC emission correctly.
- **Do NOT fix `ai-check` only in the encoder.** Without the driver parity port (R-2) the headline
  AC is unreachable. This was measured, not guessed.
- **Do NOT reference `/tmp/iter115repro/*` or `/tmp/i117/*` from any test.** Commit fixtures.
- **Do NOT weaken or restructure the `define-fun` branch of `validateDeclarations`** — it is
  M-SMT-CALLEE-SORT-GATE's live defense.
- **Do NOT add an `ai-check -strict` flag** or make skips nonzero (design decision D1-C / D-option-D
  rejected).
- **Do NOT touch `internal/smt/solver.go`.** `#510` landed in iteration 116; timeout work is out of
  scope (R-6).
- **Do NOT regenerate `docs/static/benchmarks/*.json`.**
- **Do NOT let `internal/smt/codegen.go` exceed 800 lines.**
- **Do NOT edit the main checkout** at `/Users/voightkampff/dev/sunholo-data/ailang` — it holds a
  sibling agent's uncommitted work. All work stays in `/Users/voightkampff/dev/sunholo-data/.wt-iter117`.
- **Do NOT commit or push.** The controller handles git.

---

## Open Questions (non-blocking; implementer discretion unless noted)

1. Should the shared demand-filter helper live in `cmd/ailang/verify_filter.go` or move to
   `internal/smt`? Recommendation: keep it in `cmd/ailang` — it consumes `ast`/`loader` shapes and
   moving it risks a boundary violation.
2. Should the eager alias registration also register aliases whose `MapRecordFields` **fails**?
   Recommendation: **no** — register only mappable aliases; record the failure for M2's error path.
3. If AC3.1's natural error fixture disappears after M1/M2 (see the fixture hazard), is an
   in-process injected `aiVerifySection` acceptable as the primary proof with one subprocess smoke
   test? Recommendation: yes; note it in the sprint log.
