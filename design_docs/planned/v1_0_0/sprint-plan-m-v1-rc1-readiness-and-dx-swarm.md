# Sprint Plan: M-V1-RC1-READINESS-AND-DX-SWARM

## Placement and Human-Decision Rules

- **Artifact paths**:
  - Design Doc: `design_docs/planned/v1_0_0/m-v1-rc1-readiness-and-dx-swarm.md`
  - Sprint Plan: `design_docs/planned/v1_0_0/sprint-plan-m-v1-rc1-readiness-and-dx-swarm.md`
  - Sprint JSON: `.ailang/state/sprints/sprint_M-V1-RC1-READINESS-AND-DX-SWARM.json`
- **Rulings vs. Human-Decision Gates**:
  - High-impact decisions `D1..D4` are pre-ruled with their `(Recommended)` defaults in `design_docs/planned/v1_0_0/m-v1-rc1-readiness-and-dx-swarm.md` (§High-Impact Decisions & §Design Freeze): **`D1=a`** (full M4+M5 capability hygiene: `--caps auto` stderr disclosure unless explicit `--quiet`, `Process` excluded from `auto`, `[bin]` shim capability pinning + `--grant-new-caps`, and `E_CAPS_MISSING` pre-flight), **`D2=a`** (single explicit `_` direct call-argument placeholder on RHS of `|>`), **`D3=a`** (`v0.16.21` + `v0.16.21-compact` + semver compact sort while keeping frozen `v0.16.20.md` and `v0.7.4-compact.md` untouched), and **`D4=a`** (`std/result.{mapResult,collect}` + `std/math.{minInt,maxInt,clampInt}`).
  - No decision remains OPEN under the default rulings. If a human reviewer unchecks or re-opens any `D1..D4` checkbox in `m-v1-rc1-readiness-and-dx-swarm.md` §Design Freeze before execution, **STOP: commit current work, report `BLOCKED: unratified Design Freeze decision in m-v1-rc1-readiness-and-dx-swarm.md`, and end the run**.

---

## Summary

Close all verified `v1.0.0-rc.1` soundness, bytecode-parity, multi-module `deriving (Json)`, inline-test constructor scoping, WASM/REPL validation, parser/pipe/quickfix ergonomics, stdlib combinator, capability-hygiene, Go module SemVer public-surface, CI skill-sync, and teaching-prompt defects across **5 strictly file-disjoint parallel author milestones (`M1..M5`)**, followed by dedicated integration (`M6`) and independent read-only verification (`M7`).

**Duration:** 4 days (swarm wall-clock ~1 day via 5-way parallel `share` execution)
**Dependencies:** `origin/dev` (`301e57507`), `M-V1-PUBLIC-SURFACE-HYGIENE` (M1–M3 landed)
**Risk Level:** Medium-High (mitigated by strict pairwise file disjointness, companion-file `<800L` splits, and pre-verified confinement/prompt-freeze invariants)
**Linked GitHub Issues:** `#506`, `#2323`, `#2399`, `#2404`, `#2410`

---

## Current Status Analysis

### Completed Recently
- `M-DOCS-VISUAL-NAVIGATION-OVERHAUL` (`2a0f4aeab`, `ba39fb17e`): IA overhaul, TierBanner, `llms.txt` split, and broken-anchor cleanup (~1,450 LOC in 2 days).
- `M-V1-PUBLIC-SURFACE-HYGIENE` M1–M3: stability reference doc, deprecation table enforcement (`TestStabilityDocListsEveryDeprecation`), and initial CLI surface audit (~620 LOC in 1.5 days).
- `#1720` (`301e57507`): git package checkouts ignore user `autocrlf`/`eol` (~110 LOC).

### Velocity
- Recent average: ~480–650 LOC/day single-agent; ~1,800+ LOC/day across 5 disjoint `/swarm` workers.
- Estimated capacity: **1,920 LOC** across 4 sprint-equivalent days (`480 LOC/day` nominal).

### Remaining from Design Doc
- `M1_SOUNDNESS_VM_LOWERING`: ~520 LOC (`SOUND-1`, `PARITY-B4`/`#506`, `#2323`, `#2399`)
- `M2_DERIVE_TESTS_REPL`: ~390 LOC (`DERIVE-JSON` a+b, `INLINE-TESTS-SCOPE`, `#2410`)
- `M3_PARSER_PIPE_QUICKFIX`: ~380 LOC (`\n ?hole`, `D2` `|>` `_` placeholder, `STRICT_FALLBACK_001` `[] => Ok([])`, `IMP010` `--fix`, `PAR014`/`PAR015` + warning spans)
- `M4_STDLIB_CLI_CAPS`: ~380 LOC (`D4` `std/result` + `std/math`, `docs search` stdlib fallback, `check --debug-compile --json`, `D1` capability hygiene, `SEMVER-GO` `goopus` + `stability.md` + `serveapi` `doc.go`)
- `M5_PROMPT_CI_DOCS`: ~170 LOC (`CI-1` skill sync, `D3` `v0.16.21` + `v0.16.21-compact` + semver sort, `#164`/`#165` dead clutter sweep, `LIMITATIONS.md` sync, `v1_dx_showcase.ail`)
- `M6_INTEGRATION`: ~80 LOC (merge `M1..M5`, regenerate `.stdlib-golden/*`, `docs/docs/prompts/current.md`, `llms.txt`, full cross-slice gates)
- `M7_VERIFICATION`: 0 LOC (independent read-only `sprint-evaluator` + adversarial checks)

---

## Registry Reuse Audit

Pre-verified against the live Sunholo package registry (`ailang pkg search` / `pkg info` data provided in swarm brief `#163`):

| Milestone | `ailang pkg search` Terms | Packages Inspected | Decision | Rationale |
|-----------|---------------------------|--------------------|----------|-----------|
| `M1_SOUNDNESS_VM_LOWERING` | `json` | `http_helpers`, `logging`, `a2ui`, `duckdb`, `linkedin`, `external_backend`, `email`, `agui`, `decisions`, `crew_core`, `audio` (11 domain packages) | `none` | `TRecord.TypeModule`, VM `OpEq` closure trap, `OpLowerer`, `Linker`, `argdecode`, and `serve-api` `Json` binding live in the Go compiler/VM/runtime, not AILANG userland packages. |
| `M2_DERIVE_TESTS_REPL` | `json` | 11 `sunholo/*` domain packages (`http_helpers`..`audio`) | `none` | Compiler `deriving (Json)` export/prelude synthesis, `internal/testing` scope injection, and `internal/repl` `ModuleRegistry.LoadModule` are Go compiler/tooling internals. |
| `M3_PARSER_PIPE_QUICKFIX` | `result` | `sunholo/duckdb@0.2.1`, `sunholo/external_backend@0.2.0` | `none` | Parser lookahead (`parser_func.go`), pipe elaboration (`expr_simple.go`), `strict_fallbacks.go`, and `internal/quickfix` operate on Go AST/CoreIR in the compiler. |
| `M4_STDLIB_CLI_CAPS` | `result`, `math` | `sunholo/duckdb@0.2.1`, `sunholo/external_backend@0.2.0` (`result`); `sunholo/deontic@0.3.0`, `sunholo/relativity@0.14.0`, `sunholo/audio@0.6.1` (`math`) | `none` | Registry packages are domain-specific SQL/process/physics/audio libraries; foundational `Result` list traversal (`mapResult`, `collect`) and scalar `int` clamping (`minInt`, `maxInt`, `clampInt`) belong in `std/result` and `std/math` (`std/*` cannot depend on external registry packages). |
| `M5_PROMPT_CI_DOCS` | `gcp_auth` | `sunholo/gcp_auth` (referenced in prompt teaching examples only) | `none` | Prompt markdown, `.agents/skills` sync, `.gitignore` cleanup, `LIMITATIONS.md`, and `examples/runnable/v1_dx_showcase.ail` require zero external package dependencies. |
| `M6_INTEGRATION` | `result`, `math` | None beyond `M1..M5` | `none` | Integration milestone merges `M1..M5` and regenerates repo-internal `.stdlib-golden/`, `docs/docs/prompts/current.md`, and `llms.txt`. |
| `M7_VERIFICATION` | `none` | None (read-only verification) | `none` | Read-only `sprint-evaluator` milestone executing test, parity, and boundary gates. |

---

## AILANG Syntax Gate (Mandatory Pre-Authoring Check for `M4` & `M5` `.ail` Files)

- **AILANG prompt version loaded**: `v0.16.20` (verified via `/Users/voightkampff/go/bin/ailang prompt --version-active` and `/Users/voightkampff/go/bin/ailang prompt | head -n 60` on CLI `v0.64.1`).
- **3-Row Showcase Feature Syntax Checklist** (governing `std/result.ail`, `std/math.ail`, and `examples/runnable/v1_dx_showcase.ail`):

| Feature | Exact Valid AILANG Syntax (`v0.16.20` / `v0.64.1`) | Forbidden Hallucinations to Avoid |
|---------|-----------------------------------------------------|-----------------------------------|
| **Contracts** (`requires` / `ensures`) | Placed between return/effect signature and `{` body: `export pure func clampInt(x: int, lo: int, hi: int) -> int requires { lo <= hi } ensures { result >= lo, result <= hi } { ... }` | Never `pre`/`post`/`where`, never `;` inside `requires { a, b }` (comma-separated predicates, `result` identifier in `ensures`). |
| **Effects & Budgets** | Suffix `! {IO}` or `! {IO @limit=5}` after `-> ReturnType` on `func` (never combine `pure` with `! {...}`): `export func main() -> () ! {IO @limit=5} { println("ok") }` | Never `effects [IO]`, never `throws`, never `pure func f() ! {IO}`. |
| **Inline `tests [...]`** | Placed on pure functions before `{` body, pairing `(argTuple)` with expected value: `export pure func minInt(a: int, b: int) -> int tests [((3, 5), 3), ((-2, 4), -2)] { if a <= b then a else b }` (single-arg: `[("in", "out")]`; multi-arg: `[((a, b), out)]`) | Never `assert_eq`, never top-level `test "..." {}` when writing inline function `tests [...]`, never unparenthesized multi-arg `[1, 2 => 3]`. |

---

## Inner-Loop Test Command Policy (Mandatory for All Executors)

- **During `M1..M5` implementation**: NEVER run full `make test` (240+ packages, multi-minute wall clock). Run targeted `go test <owned-packages> -count=1`, `make test-changed`, and `make check-file-sizes`.
- **Shared generated files**: `M1..M5` workers MUST NOT touch `.stdlib-golden/*`, `docs/docs/prompts/current.md`, or `llms.txt`; `M6_INTEGRATION` regenerates all three once after merging `M1..M5`.

---

## Proposed Milestones

### M1: Soundness, Bytecode Trap Isolation, Exhaustive Lowering & Boundary Decoding (~520 LOC)
**Goal:** Eliminate cross-module `Eq` witness hijacking & VM closure equality (`SOUND-1`), stop `--bytecode` side-effect replay on post-dispatch VM traps (`#506`), complete 21-variant `CoreExpr` + contract coverage in `OpLowerer` & `Linker` (`#2323`), and harden `--args-json` integer decoding + `serve-api` `std/json.Json` binding (`#2399`).
**Estimated:** 310 LOC implementation + 210 LOC tests = ~520 LOC
**Duration:** 1 day (parallel slice 1 — worker `rc1-soundness-vm`)
**Dependencies:** None

**Owned Files (Strictly Disjoint from `M2..M5`):**
- Modify: `internal/types/{types.go,instances.go,instances_eq_synth.go,unification_records.go,unification_core.go,unification_substitution.go,typechecker_substitution.go}`, `internal/elaborate/core.go`, `internal/pipeline/{alias_body_closure.go,cache_coreti_dag.go,cache_key.go,op_lowering.go}`, `internal/link/linker.go`, `internal/runner/vm.go`, `internal/vm/vm_arith.go`, `internal/runtime/argdecode/argdecode.go`, `internal/apiserver/{param_schema.go,arg_bind.go}`
- Create: `internal/pipeline/op_lowering_containers.go` (keeps `op_lowering.go` < 715L), `internal/types/sound1_cross_module_eq_test.go`, `internal/runner/vm_trap_no_fallback_test.go`, `internal/pipeline/op_lowering_containers_test.go`, `internal/link/linker_contracts_test.go`, `internal/runtime/argdecode/argdecode_strict_int_test.go`, `internal/apiserver/arg_bind_json_adt_test.go`

**Tasks:**
- Stamp `TypeModule string` on `types.TRecord`, thread through substitution/unification/elaboration/alias-closure/CoreTI-cache (+ bump `cache_key.go`), key `InstanceEnv.eqRecordNames` by `NominalKey(c.Module, c.Name)`, add open-record pattern hint on contravariant parameter mismatches in `unification_records.go`, and return `RT_EQ_001` from `runtimeEq` (`internal/vm/vm_arith.go`) on direct or container-nested `TagClosure` values.
- In `internal/runner/vm.go:281-289`, return `true, err` unconditionally after `machine.Run` returns non-nil error (eliminating evaluator replay and duplicate `vm: vm:` prefix).
- Add `Tuple`, `Array`, `RecordUpdate`, `DictAbs`, `DictApp`, `Forall` lowering in `op_lowering_containers.go` and `Match.Guard`, `Tuple`, `Array`, `RecordUpdate`, `Intrinsic`, `Forall`, + `prog.Meta[*].Contracts[*].Expr` traversal in `internal/link/linker.go`.
- Switch `argdecode.DecodeJSON` to `json.NewDecoder` + `UseNumber()`, reject fractional/out-of-range floats in `decodeInt`, and bind `std/json.Json` parameters to `JNull`/`JBool`/`JInt`/`JNumber`/`JString`/`JArray`/`JObject` `TaggedValue`s in `internal/apiserver/{param_schema,arg_bind}.go`.

**Acceptance Criteria:**
- [x] Cross-module `type Regex = { pattern: string, matcher: string -> bool }` alongside `import std/regex` fails `ailang check` (`No instance for Eq`), while valid `deriving (Eq)` records pass on evaluator and `--strict-bytecode` — `TST-RC1-01`
- [x] VM `OpEq`/`OpNeq` on direct or container-nested closures traps with `RT_EQ_001` while `bytecode.Value.Equal` constant-pool dedup remains intact — `TST-RC1-02`
- [x] Effectful entrypoint hitting `RT001` / stack overflow / `RT_EQ_001` under `--bytecode` executes side effects **once**, emits zero evaluator fallback warning, and formats error with single `vm:` prefix — `TST-RC1-03`
- [x] Lambdas with operators inside `Tuple`, `Array`, `RecordUpdate`, `DictAbs`, `DictApp`, `Forall`, `Match` guards, and `requires`/`ensures` contracts pass on both evaluator and `--strict-bytecode` (`TestContractExpressionsFullyLowered` green) — `TST-RC1-04`
- [x] `--args-json '2.9'` fails on `int` parameter, 64-bit integers decode losslessly via `UseNumber()`, and `serve-api` endpoints taking `std/json.Json` return HTTP 200 instead of HTTP 500 — `TST-RC1-05`
- [x] `go test ./internal/types/... ./internal/elaborate/... ./internal/pipeline/... ./internal/link/... ./internal/runner/... ./internal/vm/... ./internal/runtime/argdecode/... ./internal/apiserver/... -count=1` and `make check-file-sizes` pass

**Risks:**
- `CoreTI` compile cache stale hits after adding `TRecord.TypeModule` — Mitigation: serialize `TypeModule` in `cache_coreti_dag.go` AND bump schema version in `cache_key.go`.

---

### M2: Multi-Module `deriving (Json)`, Inline `tests [...]` Scope & WASM/REPL Parity (~390 LOC)
**Goal:** Fix multi-module `export type T = ... deriving (Json)` interface exports + library-module `Option`/`Result` prelude hygiene (`DERIVE-JSON`), bind imported/prelude constructors in inline `tests [...]` (`INLINE-TESTS-SCOPE`), and enforce exhaustiveness/effects/contracts/`deriving (Eq)` in `ModuleRegistry.LoadModule` (`#2410`).
**Estimated:** 225 LOC implementation + 165 LOC tests = ~390 LOC
**Duration:** 1 day (parallel slice 2 — worker `rc1-derive-tests-repl`)
**Dependencies:** None

**Owned Files (Strictly Disjoint from `M1, M3..M5`):**
- Modify: `internal/loader/loader.go` (`793L` — add 1-line call to `derive_json_exports.go` ONLY!), `internal/loader/prelude_imports.go`, `internal/elaborate/{derive_json.go,derive_json_decode.go}` (only if helper adjustment needed), `internal/testing/{executor_helpers.go,module_scope.go}`, `internal/repl/{module_registry_load.go,repl_eval.go}`
- Create: `internal/loader/derive_json_exports.go`, `internal/loader/derive_json_multimodule_test.go`, `internal/testing/inline_tests_ctor_scope_test.go`, `internal/repl/module_registry_parity_test.go`

**Tasks:**
- Implement `registerDerivedJsonExports(file, loaded)` in `internal/loader/derive_json_exports.go` so `export type T = ... deriving (Json)` registers `encode<T>` and `decode<T>` in `loaded.Exports` at load time, and update `prelude_imports.go` so `fileHasDerivingJson(file)` injects `std/option` and `std/result` in library modules (respecting local shadowing).
- Extend `Executor.injectADTConstructors` (`internal/testing/executor_helpers.go` / `module_scope.go`) to bind prelude (`Some`, `None`, `Ok`, `Err`) and explicitly imported/aliased constructors from `e.modules[*].Iface.Constructors` in the inline test evaluation environment.
- Wire `registerDerivedEq`, `SetMatchUniverse` + `VerifyMatchExhaustiveness`, `pipeline.ValidateEffectsWithCalls`, and contract expression type-checking into `ModuleRegistry.LoadModule` (`internal/repl/module_registry_load.go`).

**Acceptance Criteria:**
- [x] Sibling module `import ./types (Item, encodeItem, decodeItem)` compiles and round-trips on evaluator + `--strict-bytecode` when `types.ail` is a library module with ONLY `import std/json (Json)` — `TST-RC1-06`, `TST-RC1-07`
- [x] `ailang test` passes inline `tests [...]` expressions using `Some`/`None`, `Ok`/`Err`, and imported/aliased ADT constructors — `TST-RC1-08`
- [x] `ModuleRegistry.LoadModule` rejects `pure func` calling `println` and non-exhaustive `match` (`TC_MATCH_002`), and supports `deriving (Eq)` — `TST-RC1-09`
- [x] `internal/loader/loader.go` remains `< 800` lines; `go test ./internal/loader/... ./internal/testing/... ./internal/repl/... -count=1` and `make check-file-sizes` pass

**Risks:**
- `internal/loader/loader.go` is `793L` — Mitigation: place 100% of export synthesis helper code in `internal/loader/derive_json_exports.go` and add a single call line in `loader.go`.

---

### M3: Block Hole Disambiguation, Pipe `_` Placeholder, `STRICT_FALLBACK_001` & `check --fix` (~380 LOC)
**Goal:** Allow `\n ?hole` statements inside `{ ... }` blocks, implement single `_` call-argument placeholder on `|>` RHS (`D2=a`), exempt structural list recursion base case `[] => Ok([])` from `STRICT_FALLBACK_001`, add cross-module `IMP010` move to `ailang check --fix`, and polish block `let` hints + warning spans.
**Estimated:** 215 LOC implementation + 165 LOC tests = ~380 LOC
**Duration:** 1 day (parallel slice 3 — worker `rc1-parser-quickfix`)
**Dependencies:** None

**Owned Files (Strictly Disjoint from `M1, M2, M4, M5`):**
- Modify: `internal/parser/{parser_func.go,parser_expr.go,parser_decl.go,parser_error.go}`, `internal/elaborate/expr_simple.go`, `internal/pipeline/{strict_fallbacks.go,warn_split_args.go,warn_take_after_flatmap.go}`, `internal/quickfix/quickfix.go` (`728L` — dispatch call to `imp010_move.go` ONLY!)
- Create: `internal/quickfix/imp010_move.go`, `internal/parser/hole_newline_block_test.go`, `internal/elaborate/pipe_placeholder_test.go`, `internal/pipeline/strict_fallbacks_basecase_test.go`, `internal/quickfix/imp010_move_test.go`

**Tasks:**
- Add `lexer.QUESTION` to `peekIsLiteralStatementStart()` in `internal/parser/parser_func.go` (covering all 4 block loops while keeping same-line `expr?` in `parser_try_let.go` untouched).
- Desugar `lhs |> f(a, _, c)` -> `let $pipe_n = lhs in f(a, $pipe_n, c)` in `internal/elaborate/expr_simple.go` when RHS `FuncCall` has `1` direct `WildcardExpr` argument (`0` keeps `App(rhs, [lhs])`; `>1` returns `TC_PIPE_001`).
- Exempt empty `*core.ListPattern` arm `[] => Ok([])` in `internal/pipeline/strict_fallbacks.go` iff a sibling arm destructures a non-empty `*core.ListPattern`.
- Implement cross-module symbol move for `IMP010` in `internal/quickfix/imp010_move.go` and suppress `in` suggestions inside `{ ... }` blocks (`parser_expr.go`/`parser_decl.go`) + attach file:line spans in `warn_split_args.go` and `warn_take_after_flatmap.go`.

**Acceptance Criteria:**
- [x] `{ let y = x * 2 \n ?finish }` parses `?finish` as `*ast.HoleExpr`, same-line `f(x)?` stays `*ast.TryExpr`, and `examples/runnable/parser_block_trailing_record.ail` passes — `TST-RC1-10`
- [x] `xs |> filter(\x. x > 0, _)` and `"a,b" |> split(_, ",")` pass on evaluator + `--strict-bytecode`, `5 |> makeAdder(1)` returns `6`, and `x |> add(_, _)` fails with `TC_PIPE_001` — `TST-RC1-11`
- [x] `match xs { [] => Ok([]), x :: rest => ... }` passes `--strict-fallbacks`, while `Err(_) => Ok([])`, `None => Ok([])`, and `_ => Ok([])` still fail `STRICT_FALLBACK_001` — `TST-RC1-12`
- [x] `ailang check --fix` moves `join` from `import std/list (map, join)` to `import std/string (join)` (and deletes single-symbol `import std/list (join)`) — `TST-RC1-13`
- [x] Block `let` syntax errors do not suggest `let ... in` and `W_SPLIT_ARGS`/`LIST_TAKE_AFTER_FLATMAP` include source spans — `TST-RC1-14`
- [x] `go test ./internal/parser/... ./internal/elaborate/... ./internal/pipeline/... ./internal/quickfix/... -count=1` and `make check-file-sizes` pass

**Risks:**
- `internal/quickfix/quickfix.go` is `728L` — Mitigation: put all `IMP010` cross-module AST/line rewrite logic in `internal/quickfix/imp010_move.go`.

---

### M4: Stdlib Ergonomics, Offline `docs search`, `--json` Purity, Capability & SemVer Hygiene (~380 LOC)
**Goal:** Add `std/result.{mapResult,collect}` + `std/math.{minInt,maxInt,clampInt}` (`D4=a`), add embedded stdlib fallback + `--stdlib` to `ailang docs search`, fix `check --debug-compile --json` stdout purity, implement `D1=a` capability hygiene, and move `third_party/goopus` under `internal/` + update `stability.md` / `serveapi` `doc.go` (`SEMVER-GO`).
**Estimated:** 240 LOC implementation + 140 LOC tests = ~380 LOC
**Duration:** 1 day (parallel slice 4 — worker `rc1-stdlib-cli-caps`)
**Dependencies:** None (leave `.stdlib-golden/*` untouched for `M6_INTEGRATION`)

**Owned Files (Strictly Disjoint from `M1..M3, M5`):**
- Modify: `std/result.ail`, `std/math.ail`, `cmd/ailang/{docs_search.go,check.go,main_run.go,pkg_bin.go}`, `internal/runner/{run.go,caps.go}`, `internal/pkg/bin.go`, `internal/builtins/audio_ogg.go`, `docs/docs/reference/stability.md`, `git mv third_party/goopus internal/third_party/goopus`
- Create: `cmd/ailang/{docs_search_stdlib_fallback.go,docs_search_stdlib_fallback_test.go,check_debug_timings.go,check_json_debug_test.go}`, `internal/runner/{caps_preflight.go,caps_preflight_test.go}`, `serveapi/doc.go` (+ `serveapi/**/doc.go` if subpackages exist)

**Tasks:**
- Implement `mapResult` and `collect` in `std/result.ail` and `minInt`, `maxInt`, `clampInt` in `std/math.ail` following the AILANG Syntax Gate (`v0.16.20` idioms).
- Add `--stdlib` flag to `cmd/ailang/docs_search.go` and implement embedded `std/*.ail` search in `docs_search_stdlib_fallback.go`, auto-triggering ONLY when `*pathFlag == ""` AND `localDocsErr != nil` AND `!pkg.IsConfined()` AND `!config.NoGitHubSearch()` AND `!*noGitHubFlag` AND `githubsearch.NewBackend(ctx)` fails.
- Extract `[DEBUG-COMPILE]` timing output from `cmd/ailang/check.go` (`761L`) into `check_debug_timings.go` and route to `os.Stderr` when `--json` is set.
- Implement `D1=a`: `ExplicitQuiet` flag visit tracking (`main_run.go` + `run.go`), exclude `Process` from `ResolveAutoCaps` (`caps.go`), pin `[bin]` shim caps + `--grant-new-caps` (`internal/pkg/bin.go` + `cmd/ailang/pkg_bin.go`), and pre-flight `E_CAPS_MISSING` (`internal/runner/caps_preflight.go`).
- `git mv third_party/goopus internal/third_party/goopus`, update `internal/builtins/audio_ogg.go`, add `D-67` Go API exclusion + Internal binary split section to `docs/docs/reference/stability.md` (preserving all 16 deprecation table rows), and add `Internal` package comments in `serveapi/doc.go`.

**Acceptance Criteria:**
- [x] `std/result.{mapResult,collect}` and `std/math.{minInt,maxInt,clampInt}` pass `ailang check` and unit/inline tests on evaluator + `--strict-bytecode` — `TST-RC1-15`
- [x] `ailang docs search --stdlib "regex"` and unconfined out-of-repo fallback work offline while `go test ./cmd/ailang -run TestDocsSearch -count=1` passes 100% — `TST-RC1-16`
- [x] `ailang check --debug-compile --json` writes clean JSON to `stdout` and timings to `stderr`, with `cmd/ailang/check.go` reduced below `740` lines — `TST-RC1-17`
- [x] `--caps auto` discloses caps on `stderr` unless explicit `--quiet`, excludes `Process`, pins `[bin]` shims (`--grant-new-caps` to widen), and missing caps fail pre-flight with `E_CAPS_MISSING` — `TST-RC1-18`
- [x] `third_party/goopus` lives at `internal/third_party/goopus` and `TestStabilityDocListsEveryDeprecation` passes — `TST-RC1-19`
- [x] `go test ./cmd/ailang/... ./internal/runner/... ./internal/pkg/... ./internal/builtins/... ./serveapi/... -count=1` and `make check-file-sizes` pass

**Risks:**
- Breaking `TestDocsSearch_Confinement*` — Mitigation: run `go test ./cmd/ailang -run TestDocsSearch -count=1` immediately after wiring `docs_search_stdlib_fallback.go`.

---

### M5: CI Skill Sync, Prompt `v0.16.21` + `--compact`, Dead Clutter Sweep & Limitations Sync (~170 LOC)
**Goal:** Sync `.agents/skills/*` (`CI-1`), author & register `v0.16.21` + `v0.16.21-compact` with semver compact resolution (`D3=a`), remove & gitignore dead tracked scratch clutter (`#164`/`#165`), update `LIMITATIONS.md`, and author `examples/runnable/v1_dx_showcase.ail`.
**Estimated:** 130 LOC implementation/docs/examples + 40 LOC tests = ~170 LOC (plus prompt markdown files)
**Duration:** 0.5 days (parallel slice 5 — worker `rc1-prompt-ci-hygiene`)
**Dependencies:** None (note: `examples/runnable/v1_dx_showcase.ail` is authored in `M5` and executed end-to-end in `M6` once `M2..M4` merge; **NEVER edit frozen `prompts/v0.16.20.md` or `prompts/v0.7.4-compact.md`**, and **KEEP `cmd/astdump/` and `prompts/devtools/v0.8.{0,1}-compact.md`**)

**Owned Files (Strictly Disjoint from `M1..M4`):**
- Modify: `.agents/skills/{mission-loop-change,local-model-onboarding,local-ollama-eval}/**`, `prompts/versions.json`, `cmd/ailang/prompts/versions.json`, `cmd/ailang/prompt.go`, `.gitignore`, `docs/LIMITATIONS.md`, `docs/docs/reference/limitations.md`, `examples/runnable/pipe_operator.ail`
- Create: `prompts/v0.16.21.md`, `prompts/v0.16.21-compact.md`, `cmd/ailang/prompts/v0.16.21.md`, `cmd/ailang/prompts/v0.16.21-compact.md`, `cmd/ailang/prompt_compact_test.go`, `examples/runnable/v1_dx_showcase.ail`
- Delete (`git rm`): `.claude/fmt_hook_events.jsonl`, `dist/error_codes.json`, `examples/manifest.json.bak`, `examples/manifest.tmp`

**Tasks:**
- Run `bash tools/launchd/sync-agents-skills.sh` and verify `go test ./internal/mission/quorum/... -run TestQuorumArtifactShellSuite -count=1`.
- Author `prompts/v0.16.21.md` (fixing all 6 audit contradictions and adding `@limit`, `./sibling`, `tests [...]`, `|>` `_` placeholder) and `prompts/v0.16.21-compact.md` (~3.5 KB accurate v0.64+ syntax/CLI sheet), mirror both to `cmd/ailang/prompts/`, register in both `versions.json` files, and replace ASCII `sort.Strings` in `cmd/ailang/prompt.go:150` with semver comparison.
- `git rm` `.claude/fmt_hook_events.jsonl`, `dist/error_codes.json`, `examples/manifest.json.bak`, `examples/manifest.tmp` and update `.gitignore` (ignoring those 4 paths/dirs and re-admitting `!/.ailang/state/evaluations/**` + `!/.ailang/state/implementation/**`).
- Sync `docs/LIMITATIONS.md` + `docs/docs/reference/limitations.md` (`reviewBy` preserved) and create `examples/runnable/v1_dx_showcase.ail` (using `requires`/`ensures`, `! {IO @limit=5}`, `|>` `_`, `mapResult`, and inline `tests [...]`).

**Acceptance Criteria:**
- [x] `go test ./internal/mission/quorum/... -run TestQuorumArtifactShellSuite -count=1` passes — `TST-RC1-20`
- [x] `ailang prompt --version-active` outputs `v0.16.21`, `ailang prompt --compact` outputs `v0.16.21-compact` (zero `**`/`abs_Int`), and `make check-prompt-freeze` exits `0` — `TST-RC1-20`
- [x] `.claude/fmt_hook_events.jsonl`, `dist/error_codes.json`, `examples/manifest.json.bak`, `examples/manifest.tmp` are untracked and ignored by `.gitignore`, while `cmd/astdump/` and `prompts/devtools/v0.8.{0,1}-compact.md` remain intact — `TST-RC1-20`
- [x] `docs/LIMITATIONS.md`, `docs/docs/reference/limitations.md`, and `examples/runnable/v1_dx_showcase.ail` authored cleanly — `TST-RC1-21`

**Risks:**
- Editing frozen prompt files by mistake — Mitigation: verify `git diff prompts/v0.16.20.md prompts/v0.7.4-compact.md` is empty and run `make check-prompt-freeze` before committing `M5`.

---

### M6: Cross-Slice Integration & Shared Golden Regeneration (~80 LOC)
**Goal:** Merge `M1..M5` cleanly, regenerate shared derived artifacts (`.stdlib-golden/*`, `docs/docs/prompts/current.md`, `llms.txt`), and run full repository build, doc-sample, stdlib, bytecode-parity, and Docusaurus gates.
**Estimated:** 80 LOC (regenerated golden/sync diffs + `examples/manifest.json` registration if applicable)
**Duration:** 0.5 days (worker `rc1-integrator`)
**Dependencies:** M1_SOUNDNESS_VM_LOWERING, M2_DERIVE_TESTS_REPL, M3_PARSER_PIPE_QUICKFIX, M4_STDLIB_CLI_CAPS, M5_PROMPT_CI_DOCS

**Tasks:**
- Merge `M1..M5` branches in order (`M1` -> `M2` -> `M3` -> `M4` -> `M5`; zero file overlap guarantees conflict-free merges) and build `/tmp/.../bin/ailang`.
- Run `make freeze-stdlib`, `bash docs/scripts/sync-active-prompt.sh`, and `make generate-llms-txt && make check-llms-txt`.
- Execute `examples/runnable/v1_dx_showcase.ail` and `examples/runnable/pipe_operator.ail` across `ailang check --strict-fallbacks`, `ailang test`, `ailang run --caps IO`, and `ailang run --strict-bytecode --caps IO`.
- Run `make check-prompt-freeze`, `make check-file-sizes`, `make check-boundaries`, `make verify-stdlib`, `DOC_SAMPLES_AILANG=<built-bin> make check-doc-samples`, `go test ./internal/mission/quorum/... -run TestQuorumArtifactShellSuite -count=1`, `make test-changed`, `go run ./scripts/verify_bytecode_parity.go --strict`, and `cd docs && npm run build`.

**Acceptance Criteria:**
- [x] Clean conflict-free merge of `M1..M5` with `.stdlib-golden/*`, `docs/docs/prompts/current.md`, and `llms.txt` regenerated and passing `make verify-stdlib` + `make check-llms-txt` — `TST-RC1-21`
- [x] `examples/runnable/v1_dx_showcase.ail` passes `check --strict-fallbacks`, `test`, `run`, and `run --strict-bytecode` — `TST-RC1-21`
- [x] `make check-prompt-freeze`, `make check-file-sizes`, `make check-boundaries`, `make check-doc-samples`, `verify_bytecode_parity.go --strict`, and `cd docs && npm run build` exit `0` — `TST-RC1-21`

**Risks:**
- New `v0.16.21.md` doc sample syntax typo — Mitigation: `DOC_SAMPLES_AILANG=<built-bin> make check-doc-samples` validates every fenced block in `prompts/v0.16.21.md` and `docs/`.

---

### M7: Independent End-to-End & Adversarial Verification (~0 LOC)
**Goal:** Independent read-only `sprint-evaluator` audit across `TST-RC1-01..22`, all 5 Conflict Surface regression fixtures, WASM/REPL + `--strict-bytecode` parity, and negative controls.
**Estimated:** 0 LOC (read-only verification — worker `rc1-verifier`)
**Duration:** 0.5 days
**Dependencies:** M6_INTEGRATION

**Tasks:**
- Run adversarial probes for `SOUND-1` (cross-module record alias `==`), `#506` (side-effect count == 1 on VM trap), `#2323` (all container expressions), `#2399` (`--args-json '2.9'` & `serve-api` `Json`), `DERIVE-JSON` (sibling import + zero-import library module), `INLINE-TESTS-SCOPE`, `#2410` (`ModuleRegistry.LoadModule` effect/exhaustiveness rejection), `\n ?hole`, `|>` `_` vs `5 |> makeAdder(1)`, `STRICT_FALLBACK_001` (`[] => Ok([])` allowed vs `Err(_) => Ok([])` rejected), `IMP010` `--fix`, `D1` capabilities, and `D3` prompts.
- Verify all 5 Conflict Surface regression fixtures and confirm `git status --short` is clean and all gates pass >= 90/100 on the `sprint-evaluator` rubric.

**Acceptance Criteria:**
- [x] Independent `rc1-verifier` confirms `22/22` `TST-RC1-*` test IDs and all 5 Conflict Surface regression programs pass with zero regressions — `TST-RC1-22`
- [x] `sprint-evaluator` score `>= 85/100` with `0` open findings (`98/100` -> `100/100` after `serve_api_mcp_surface_test.go` `v15` fixture sync)

**Risks:**
- Subtle interaction between `E_CAPS_MISSING` pre-flight and existing CLI tests — Mitigation: `rc1-verifier` runs full `go test ./cmd/ailang/... ./internal/runner/... -count=1`.

---

## Swarm Worker & Model Assignment (Computed via `estimate_task_elo.sh --json`)

All Task ELO scores below were computed deterministically in `/tmp/v1-polish-plan` via `bash .agents/skills/sprint-planner/scripts/estimate_task_elo.sh --json <paths...>`:

| Worker | Milestone(s) | Est. ELO | Workspace | Assigned Model (Fallback) | Deterministic ELO Breakdown & Rationale |
|--------|--------------|----------|-----------|---------------------------|-----------------------------------------|
| `coordinator` | Swarm orchestration, user gates, final sign-off | `2850+ (Deep Frontier)` | main | `inherit` (`claude-opus-5-5-high` / `gemini-4-carbon-high`) | Owns board dispatch, model switching, and branch landing |
| `rc1-soundness-vm` | `M1_SOUNDNESS_VM_LOWERING` | `2850 (Deep Frontier)` | `share` | `gemini-4-carbon-high` (`claude-opus-5-5-high`) | `base 2650` (`internal/types/`) + `50` (`op_lowering.go` 703L) + `50` (`internal/types` 24 importers) + `75` (8 pkgs) + `25` (churn) |
| `rc1-derive-tests-repl` | `M2_DERIVE_TESTS_REPL` | `2850 (Deep Frontier)` | `share` | `gemini-4-carbon-high` (`claude-opus-5-5-high`) | `base 2650` (`internal/elaborate/`) + `50` (`loader.go` 793L) + `50` (`internal/loader` 10 importers) + `75` (4 pkgs) + `25` (churn) |
| `rc1-parser-quickfix` | `M3_PARSER_PIPE_QUICKFIX` | `2850 (Deep Frontier)` | `share` | `gemini-4-argon-high` / `gemini-4-carbon-high` (`claude-opus-5-5-high`) | `base 2650` (`internal/parser/`) + `50` (`quickfix.go` 728L, `parser_decl.go` 679L) + `50` (`internal/parser` 17 importers) + `75` (4 pkgs) + `25` (churn) |
| `rc1-stdlib-cli-caps` | `M4_STDLIB_CLI_CAPS` | `2600 (Core SWE)` | `share` | `gemini-4-neon-high` (`claude-sonnet-5-5-high`) | `base 2400` (`cmd/ailang/`) + `50` (`check.go` 761L, `run.go` 743L) + `50` (`internal/pkg` 12 importers) + `75` (7 pkgs) + `25` (churn); touches capability security boundary |
| `rc1-prompt-ci-hygiene` | `M5_PROMPT_CI_DOCS` | `2550 (Core SWE)` | `share` | `gemini-4-neon-medium` (`claude-sonnet-5-5-high`) | `base 2400` (`cmd/ailang/`) + `50` (`versions.json` 1301L) + `75` (6 pkgs) + `25` (churn) |
| `rc1-integrator` | `M6_INTEGRATION` | `2250 -> 2700 (Frontier SWE)` | `share` | `gemini-4-neon-high` (`gemini-4-argon-high`) | Script base `2250` on `.stdlib-golden`/`docs`/`llms.txt`; escalated to `Frontier SWE` because `M6` merges 5 branches and runs cross-compiler golden/parity gates |
| `rc1-verifier` | `M7_VERIFICATION` | `2850 (Deep Frontier)` | `inherit` | `claude-opus-5-5-high` / `gemini-4-argon-high` | `base 2650` (`internal/types/`) + `200` modifiers across 5 packages; independent cross-model `sprint-evaluator` |
| `rc1-monitor` | Swarm stall watchdog | `2179 (Routine)` | `inherit` | `gemini-4-hydrogen-high` | Timer-driven `swarm_monitor` skill only (never `gemini-3.8-flash-*`) |

---

## Success Metrics
- Test coverage: `22/22` `TST-RC1-*` acceptance suites green; `>85%` coverage on new companion files
- Examples passing: `examples/runnable/{v1_dx_showcase,pipe_operator,record_patterns,derive_json,parser_block_trailing_record}.ail` passing on both evaluator and `--strict-bytecode`
- Documentation: `docs/docs/reference/stability.md`, `docs/LIMITATIONS.md`, `docs/docs/reference/limitations.md`, `docs/docs/prompts/current.md`, `llms.txt` synced
- File sizes: `make check-file-sizes` green (`0` `.go` files `>= 800` lines)
- Architecture & Freeze Gates: `make check-boundaries`, `make check-prompt-freeze`, `make verify-stdlib`, `make check-doc-samples`, `TestQuorumArtifactShellSuite`, `cd docs && npm run build` all green

## Dependencies
- `M1..M5` have **zero** inter-slice file or build dependencies and execute concurrently.
- `M6` depends on `M1..M5`; `M7` depends on `M6`.

## Open Questions
- None (`D1=a`, `D2=a`, `D3=a`, `D4=a` are locked in `m-v1-rc1-readiness-and-dx-swarm.md` §Design Freeze).

## Notes
- Engine parity means **evaluator + bytecode VM** (`--strict-bytecode`) only; the legacy Go emitter was deleted in `#2389`.
- `cmd/astdump/` and `prompts/devtools/v0.8.{0,1}-compact.md` have verified live callers (`V20`) and MUST NOT be deleted.
