# M-V1-RC1-READINESS-AND-DX-SWARM: v1.0.0-rc.1 Compiler Soundness, Engine Parity, AI Ergonomics & Public Surface Hygiene

**Status**: Implemented in v1.0.0-rc.1 (`2e583399d`, PR #2486; prior status: Planned)
**Target**: v0.64.2 / v1.0.0-rc.1
**Priority**: P0 (v1.0.0-rc.1 Release Gate)
**Estimated**: 4 days (~1,920 LOC implementation + tests across 5 parallel author slices + integration + verification)
**Baseline Commit**: `301e57507ff13f2dd7520114bce060675264d6d6` (`origin/dev`; live CLI probes run against `/Users/voightkampff/go/bin/ailang` `v0.64.1`)
**Dependencies**: `M-V1-PUBLIC-SURFACE-HYGIENE` (M1–M3 landed), `M-BYTECODE-VM`, `M-DERIVE-JSON`, `M-DX-INLINE-TESTS`
**Related Issues**: `#506`, `#2323`, `#2399`, `#2404`, `#2410`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Eliminates cross-module `Eq` witness hijacking (`SOUND-1`), VM closure pointer-equality nondeterminism (`RT_EQ_001`), `--args-json` float truncation (`#2399`), and evaluator-vs-VM operator lowering divergence (`#2323`). |
| A2: Replayability | +1 | Stops `--bytecode` from replaying effectful entrypoints a second time on the evaluator after a post-dispatch VM runtime trap (`PARITY-B4` / `#506`). |
| A3: Effect Legibility | +1 | Restores `ValidateEffectsWithCalls` and `VerifyMatchExhaustiveness` parity in `ModuleRegistry.LoadModule` (`#2410` WASM/REPL) and makes `--caps auto` disclose granted capabilities on `stderr` unless `--quiet` is explicitly passed (`D1`). |
| A4: Explicit Authority | +1 | Excludes ambient `Process` from `--caps auto`, pins `[bin]` shim capabilities at `ailang install` time (`--grant-new-caps` to widen), and adds pre-flight `E_CAPS_MISSING` before executing effectful entrypoints (`D1`). |
| A5: Bounded Verification | +1 | Enforces module-qualified `NominalKey(TypeModule, TypeName)` on `TRecord` `Eq` synthesis and keeps `[] => Ok([])` base-case exemption strictly local to list pattern matches without weakening `STRICT_FALLBACK_001` on `Err(_)`/`None`/`_`. |
| A6: Safe Concurrency | 0 | No concurrency scheduling or channel semantics changed. |
| A7: Machines First | +1 | Fixes `--debug-compile --json` stdout corruption, ships accurate `v0.16.21` + `v0.16.21-compact` prompts (`D3`), automates `IMP010` cross-module `ailang check --fix` moves, and adds embedded stdlib fallback to `ailang docs search`. |
| A8: Minimal Syntax | 0 | Reuses existing `_` token as explicit call-argument placeholder on `|>` RHS (`D2=a`) and existing `?name` typed hole token on newline block statements without adding lexer tokens or precedence tiers. |
| A9: Cost Visibility | 0 | No hidden runtime allocation or effect budget changes; `--debug-compile` phase timings remain visible on `stderr`. |
| A10: Composability | +1 | Fixes multi-module `export type T = ... deriving (Json)` interface exports + prelude hygiene (`DERIVE-JSON`), binds imported/prelude constructors in inline `tests [...]`, and adds `std/result.{mapResult,collect}` + `std/math.{minInt,maxInt,clampInt}` (`D4`). |
| A11: Structured Failure | +1 | Introduces structured `RT_EQ_001`, `E_CAPS_MISSING`, and `TC_PIPE_001` diagnostics, removes duplicate `vm: vm:` error wrapping, and populates source spans on `W_SPLIT_ARGS` / `LIST_TAKE_AFTER_FLATMAP`. |
| A12: System Boundary | +1 | Rejects lossy float-to-int JSON boundary coercion (`#2399`), binds `std/json.Json` ADT values at the `serve-api` HTTP boundary, and moves `third_party/goopus` under `internal/` before Go module `v1.0.0` SemVer freeze (`SEMVER-GO`). |

**Net Score: +9** — Proceed to sprint planning and swarm execution.

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced; closes closure pointer comparison (`SOUND-1`) and float truncation (`#2399`).
- [x] A3 (Effects): No hidden side effects; closes WASM/REPL `pure func` effect bypass (`#2410`) and VM fallback effect replay (`#506`).
- [x] A4 (Authority): No ambient authority granted; narrows `--caps auto` by removing `Process` and pins `[bin]` shims (`D1`).
- [x] A7 (Machines First): Optimizes deterministic machine feedback (`--json` stdout purity, `check --fix`, compact prompt accuracy, offline stdlib search).

---

## Problem Statement

An end-to-end v1.0.0 readiness and AI-ergonomics audit against `origin/dev` (`301e57507`, CLI `v0.64.1`) uncovered five categories of defects that must be closed before cutting `v1.0.0-rc.1`:

1. **Compiler Soundness, Engine Parity & Boundary Decoding (`M1`)**:
   - **`SOUND-1` (Cross-module record alias `==` hijacks nominal `Eq` and compares closures)**: `InstanceEnv.eqRecordNames` (`internal/types/instances.go:52-57`) is keyed by bare `TypeName` (`"Regex"`), and `types.TRecord` (`internal/types/types.go:188`) carries `TypeName string` with no `TypeModule`. Importing `std/regex` (`export type Regex = Regex(string) deriving (Eq)`) causes `synthesizeEq` (`internal/types/instances_eq_synth.go:77-84`) to grant `StructuralEq` to an unrelated user `type Regex = { pattern: string, matcher: string -> bool }` in another module. At runtime, the evaluator crashes (`internal error: == reached a function value`) while `--strict-bytecode` silently exits `0` returning `false` via pointer equality (`internal/vm/vm_arith.go:114` -> `bytecode/value.go:469`).
   - **`PARITY-B4` (`#506` — `--bytecode` replays side effects on evaluator after VM trap)**: `runModuleOnVM` (`internal/runner/vm.go:283-289`) computes `attempted := machine.EffectCalls > 0 || effCtx.EffectAttempts() != attemptsBefore` after `machine.Run`. Any post-dispatch VM runtime failure before a counted foreign effect call (`RT001` division by zero, `ErrStackOverflow`, `RT_EQ_001` closure equality) returns `(false, err)`, causing `ExecuteModuleEntrypoint` (`internal/runner/entrypoint.go:108`) to print `⚠ bytecode path unavailable (vm: vm: RT001: ...); falling back to evaluator` (with a doubled `vm: vm:` prefix) and re-execute the entrypoint on the evaluator.
   - **`#2323` (`OpLowerer` & `Linker` incomplete `CoreExpr` traversal)**: `(1, \x. x + 1)` crashes the evaluator (`intrinsic operations require OpLowering pass`) while succeeding on `--strict-bytecode` because `OpLowerer.lowerExpr` (`internal/pipeline/op_lowering.go:346-353`) skips `*core.Tuple`, `*core.Array`, `*core.RecordUpdate`, `*core.DictAbs`, `*core.DictApp`, and `*core.Forall`, and `Linker` (`internal/link/linker.go:142-223`) omits `prog.Meta[*].Contracts[*].Expr`, `Match` arm `Guard`, `Tuple`, `Array`, `RecordUpdate`, `Intrinsic`, and `Forall`.
   - **`#2399` (`--args-json` silent float->int truncation & `serve-api` `Json` HTTP 500)**: `decodeInt` (`internal/runtime/argdecode/argdecode.go:154-169`) converts `float64` via `int(v)`, silently truncating `2.9 -> 2` and losing >53-bit integer precision without `UseNumber()`. Separately, `serve-api` (`internal/apiserver/param_schema.go:22-30`, `arg_bind.go:63-79`) classifies `std/json.Json` parameters as `Kind: ""` and passes raw `embed.FromGo(v)` values instead of `std/json.Json` ADT `TaggedValue` constructors, causing HTTP 500 at call time.

2. **`deriving (Json)`, Inline `tests [...]` Scope & WASM/REPL Pipeline Parity (`M2`)**:
   - **`DERIVE-JSON` (a) Multi-module `IMP010` + (b) Unhygienic `Option`/`Result` in library modules**: (a) `ModuleLoader.Load` (`internal/loader/loader.go:352,436`) populates `loaded.Exports` from AST `file.Funcs` before `synthesizeDerivedJsonDecls` runs during elaboration; when an importing module checks `moduleLoader.GetImportExport` (`internal/elaborate/file.go:260`), `encode<T>` and `decode<T>` are missing and fail with `IMP010`. (b) `injectPreludeImports` (`internal/loader/prelude_imports.go:102,128`) lexes raw source tokens for `Option`/`Result` in library modules; a library module declaring `export type Item = { name: string, count: int } deriving (Json)` with only `import std/json (Json)` never mentions `Option` or `Result` in raw text, so `std/option` and `std/result` are not injected and type-checking fails with `type Option expects 1 type argument(s), but got 0`.
   - **`INLINE-TESTS-SCOPE`**: `ailang test` fails with `undefined variable: None` / `Some` because `Executor.injectADTConstructors` (`internal/testing/executor_helpers.go:517-556`) only injects constructors declared locally in `e.sourceFile.Decls`, omitting prelude (`Option`, `Result`) and imported constructors from `e.modules[*].Iface.Constructors`.
   - **`#2410` (WASM / Playground / REPL `ModuleRegistry.LoadModule` pipeline bypass)**: `internal/repl/module_registry_load.go:248-356` skips `SetMatchUniverse` + `VerifyMatchExhaustiveness`, `pipeline.ValidateEffectsWithCalls`, `registerDerivedEq`, and `prog.Meta[*].Contracts` type-checking, allowing `pure func` side effects and non-exhaustive matches in WASM/REPL.

3. **Parser Block Hole Disambiguation, Pipe Placeholder, `STRICT_FALLBACK_001` & `check --fix` (`M3`)**:
   - **`?name` Typed Hole After Newline in Blocks**: `peekSameLineQuestion` (`internal/parser/parser_try_let.go:59`) already restricts postfix `expr?` to the same source line, but `peekIsLiteralStatementStart` (`internal/parser/parser_func.go:425-445`) omits `lexer.QUESTION`. Consequently, `let y = x * 2 \n ?finish` inside `{ ... }` stops block parsing after `x * 2` and fails with `PAR_UNEXPECTED_TOKEN: expected next token to be }, got ? instead`.
   - **Pipe `|>` Multi-Argument Ergonomics (`D2`)**: Stdlib functions follow data-last conventions in `std/list` (`map(f, xs)`, `filter(p, xs)`), data-first in `std/string` (`split(s, delim)`), and mixed positions in `std/map`, forcing nested calls or ad-hoc lambdas `xs |> (\ys. filter(p, ys))` because `|>` only supports unary `f(lhs)` / `f(a)(lhs)`.
   - **`STRICT_FALLBACK_001` False Positive on `[] => Ok([])`**: `internal/pipeline/strict_fallbacks.go:116-148` flags the canonical structural recursion base case `match xs { [] => Ok([]), x :: rest => ... }`, blocking pure `Result` list traversals under `--strict-fallbacks`.
   - **`IMP010` Cross-Module `ailang check --fix` & Diagnostic Polish**: `IMP010` already identifies `'join' is exported by std/string`, yet `ailang check --fix` cannot move symbols across modules; `PAR014`/`PAR015` suggest `let x = expr in body` inside `{ ... }` blocks where `;`/newlines are canonical; `W_SPLIT_ARGS` and `LIST_TAKE_AFTER_FLATMAP` emit empty locations.

4. **Stdlib Combinators, Offline `docs search`, `--json` Purity, Capability & SemVer Surface Hygiene (`M4`)**:
   - Missing `std/result.{mapResult,collect}` and `std/math.{minInt,maxInt,clampInt}` (`D4`); `ailang docs search <query>` fails outside a git checkout without GitHub network/token instead of falling back to embedded stdlib docs; `ailang check --debug-compile --json` writes human timing lines to `stdout`, breaking `jq`/JSON parsers; `--caps auto` stays silent by default because `quietFlag` defaults to `true` (`cmd/ailang/main_run.go:31`), grants ambient `Process`, leaves `[bin]` shims unpinned, and lacks pre-flight `E_CAPS_MISSING` (`D1`); `third_party/goopus` sits outside `internal/` where a Go `v1.0.0` tag would freeze it under Go module SemVer (`SEMVER-GO`).

5. **CI Skill Sync, Teaching Prompt `v0.16.21` + `--compact`, Dead Clutter Sweep & Limitations Sync (`M5`)**:
   - `TestQuorumArtifactShellSuite` fails on `origin/dev` because `.agents/skills/{mission-loop-change,local-model-onboarding,local-ollama-eval}` drifted from `.claude/skills/*` (`CI-1`); `ailang prompt --compact` serves stale `v0.7.4-compact.md` (`++`, `**`, `abs_Int`) and `cmd/ailang/prompt.go:150` sorts compact versions with ASCII `sort.Strings` (`"v0.7.4-compact" > "v0.16.21-compact"`); `v0.16.20.md` has 6 documented teaching contradictions (`D3`); tracked scratch files (`.claude/fmt_hook_events.jsonl`, `dist/error_codes.json`, `examples/manifest.json.bak`, `examples/manifest.tmp`) clutter the tree (`#164`/`#165`); `docs/LIMITATIONS.md` lags `v0.64.1+`.

---

## Measured Verification Log (Mandatory Pre-Design Gate)

Every claim in this design document was verified against `/tmp/v1-polish-plan` (`301e57507`) and `/Users/voightkampff/go/bin/ailang` (`v0.64.1`) across all four mandatory verification classes:

### Class 1: Live CLI Transcripts (`/Users/voightkampff/go/bin/ailang` `v0.64.1`)

| ID | Probe | Command & Measured Outcome on `v0.64.1` |
|----|-------|-----------------------------------------|
| `V1` | `SOUND-1` cross-module `Regex` record alias with function field | Module `s1/a` exports `type Regex = { pattern: string, matcher: string -> bool }`; `s1/b` imports `std/regex (Regex as StdRegex, compile)` + `s1/a (Regex, mk)` and evaluates `r1 == r2`. `ailang check`: `exit 0` (`No errors found!`); `ailang run`: `exit 1` (`internal error: == reached a function value (record vs record)`); `ailang run --strict-bytecode`: `exit 0`, prints `false`. |
| `V2` | `PARITY-B4` (`#506`) `--bytecode` runtime trap fallback & `vm: vm:` | `export func main() -> int ! {IO} { println("SIDE_EFFECT"); 10 / 0 }`: `ailang run --bytecode --caps IO` prints `SIDE_EFFECT`, then `⚠ bytecode path unavailable (vm: vm: RT001: division by zero); falling back to evaluator`, then prints `SIDE_EFFECT` a **second time** before failing. |
| `V3` | `#2323` operator inside lambda in tuple | `let p = (1, \x. x + 1); match p { (_, f) => f(41) }`: `ailang run` fails with `intrinsic operations require OpLowering pass (standalone BinOp is deprecated)` (`exit 1`), whereas `ailang run --strict-bytecode` prints `42` (`exit 0`). |
| `V4` | `#2399` `--args-json` float->int truncation | `export pure func takeInt(n: int) -> int { n }`: `ailang run --entry takeInt --args-json '2.9'` exits `0` and prints `2`. |
| `V5` | `DERIVE-JSON` (a) sibling import & (b) library module | (a) `import dj2/types (Item, encodeItem, decodeItem)` fails with `Error [IMP010]: symbol 'encodeItem' not exported by 'dj2/types'`. (b) Library module `dj/types.ail` with `import std/json (Json)` + `export type Item = { name: string, count: int } deriving (Json)` fails `ailang check` with `type error in dj/types (decl 1): type Option expects 1 type argument(s), but got 0`. |
| `V6` | `INLINE-TESTS-SCOPE` prelude `None`/`Some` in `tests [...]` | `export pure func safeDiv(a: int, b: int) -> Option[int] tests [((10, 2), Some(5)), ((10, 0), None)] { ... }`: `ailang test` fails both cases with `Error: error evaluating expected expression: evaluation error: undefined variable: Some` / `undefined variable: None`. |
| `V7` | `?finish` typed hole after newline in `{ ... }` block | `export pure func double(x: int) -> int { let y = x * 2 \n ?finish }`: `ailang check` fails with `PAR_UNEXPECTED_TOKEN at hole.ail:4:3: expected next token to be }, got ? instead`. |
| `V8` | `STRICT_FALLBACK_001` on `[] => Ok([])` | `match xs { [] => Ok([]), x :: rest => ... }`: `ailang check --strict-fallbacks` fails with `STRICT_FALLBACK_001: match arm returns trivial identity fallback Ok([])`. |
| `V9` | `ailang prompt --compact` & `check --debug-compile --json` | `ailang prompt --compact` emits frozen `v0.7.4-compact.md` containing invalid `**`, `abs_Int`, and unbraced match examples; `ailang check --debug-compile --json` writes `[DEBUG-COMPILE] Phase timings` lines to `stdout` ahead of the JSON payload. |

### Class 2: Error Code Allocation Greps (`grep -rn "CODE" internal/ cmd/`)

| Code | Grep Result Across `internal/` & `cmd/` | Planned Owner & Meaning |
|------|-----------------------------------------|-------------------------|
| `RT_EQ_001` | `0` matches (unallocated; existing VM codes are `RT001` div-by-zero, `RT_REC_003` recursion overflow) | `M1` (`internal/vm/vm_arith.go`): runtime trap when `OpEq`/`OpNeq` compares `TagClosure` values. |
| `TC_PIPE_001` | `0` matches (unallocated) | `M3` (`internal/parser/parser_expr.go` / `internal/elaborate/expr_simple.go`): multiple `_` placeholders in a single `|>` stage (`x |> f(_, _)`). |
| `E_CAPS_MISSING` | `0` matches (unallocated; `internal/errors/codes.go` has `CAP001`/`CAP002` compile codes) | `M4` (`internal/runner/caps_preflight.go`): `ailang run` pre-flight refusal when entrypoint's declared effect row requires capabilities absent from `--caps`. |

### Class 3: Code-Path Mechanism Reads (`file:line`)

| ID | Mechanism Claim | Verified Source Location & Exact Behavior |
|----|-----------------|-------------------------------------------|
| `V10` | `SOUND-1` bare-name `Eq` map & `TRecord` missing module | `internal/types/types.go:183-189` (`TRecord` has `TypeName string`, no `TypeModule`); `internal/types/instances.go:52-57` (`env.eqRecordNames[c.Name] = true`); `internal/types/instances_eq_synth.go:77-84` & `:255-258` (`env.HasDerivedEqRecord(rec.TypeName)`); `internal/vm/vm_arith.go:114` (`runtimeEq` calls `lhs.Equal(rhs)` at `internal/bytecode/value.go:469`, comparing closure pointer + captures). |
| `V11` | `PARITY-B4` fallback replay & double `vm: vm:` | `internal/runner/vm.go:281-289` (`attempted := machine.EffectCalls > 0 || effCtx.EffectAttempts() != attemptsBefore; return attempted, fmt.Errorf("vm: %w", err)`); `internal/vm/vm.go:45-51` (`VMError.Error()` already prefixes `"vm: "`). All compile/eligibility checks (`BC001`, `StrictEvalOnlyError`, `proto.EvalOnly`) return BEFORE `machine.Run` at lines `183-279`. |
| `V12` | `#2323` 21 `CoreExpr` variants vs `OpLowerer` & `Linker` | `internal/core/core.go` defines 21 `CoreExpr` types. `OpLowerer.lowerExpr` (`internal/pipeline/op_lowering.go:167-354`) handles 15 and falls through `default: return expr` on `Tuple`, `Array`, `RecordUpdate`, `DictAbs`, `DictApp`, `Forall` (`Lower` at `:102-121` already walks `prog.Meta[*].Contracts[*].Expr`). `Linker.collectDictRefs` (`internal/link/linker.go:130-224`) omits `prog.Meta[*].Contracts[*].Expr` and `collectFromExpr` omits `Match` arm `Guard`, `Tuple`, `Array`, `RecordUpdate`, `Intrinsic`, `Forall`. |
| `V13` | `DERIVE-JSON` pre-elaboration export & prelude token scan | `internal/loader/loader.go:352,436` populates `loaded.Exports` at parse time before `synthesizeDerivedJsonDecls` (`internal/elaborate/derive_json.go:39`); `internal/elaborate/file.go:260` checks `moduleLoader.GetImportExport`. `internal/loader/prelude_imports.go:102,128` (`fileUsesOption`/`fileUsesResult`) scans raw lexer tokens, missing compiler-synthesized `Option`/`Result` AST nodes in `deriving (Json)`. |
| `V14` | `?finish` newline hole failure is block-starter, NOT `TryExpr` | `internal/parser/parser_try_let.go:59` (`peekSameLineQuestion`) **already** enforces `p.peekToken.Line == p.curToken.Line`. The failure occurs in `internal/parser/parser_func.go:401-450`: `peekIsLiteralStatementStart` omits `lexer.QUESTION`, terminating `parseFunctionBody` (`:351`) and `parseBlockOrExpression` (`parser_expr.go:419`) before `?finish`. |
| `V15` | `--caps auto` silence & `--caps` blast radius census | `cmd/ailang/main_run.go:31` defaults `quietFlag := fs.Bool("quiet", true, ...)`, so `if !opts.Quiet` at `internal/runner/run.go:282` never fires unless `--verbose` is passed! Census across `cmd/`, `internal/`, `make/`, `tools/`, `scripts/`, `Makefile`: `1,432` `--caps` references; scripts/Makefiles pass explicit `--caps IO` / `--caps IO,FS,Clock,Rand`, while `[bin]` shim tests (`internal/pkg/bin_test.go:145,240`) assert `--quiet --caps auto`. |
| `V16` | Prompt freeze gate & compact ASCII sort bug | `prompts/versions.json:710-726` marks `"v0.7.4-compact"` as `"frozen": {"at": "2026-08-27", "reason": "legacy"}`; `cmd/ailang/prompt_freeze_check_git.go:30-53` fails `make check-prompt-freeze` if any frozen prompt file is deleted or edited! Furthermore, `cmd/ailang/prompt.go:150` uses ASCII `sort.Strings(compacts)`, where `"v0.7.4-compact" > "v0.16.21-compact"` (`'7' > '1'`), which would keep selecting `v0.7.4-compact` unless replaced by semver sorting! |

### Class 4: Read Regression Fixtures & Negative-Existence Greps

| ID | Fixture / Negative Claim | Verified Fact |
|----|--------------------------|---------------|
| `V17` | Existing `.ail` fixtures read in full | Read `examples/runnable/pipe_operator.ail` (curried `5 |> makeAdder(1)` at line 27), `examples/runnable/record_patterns.ail` (open/closed patterns), `examples/runnable/derive_json.ail`, `examples/runnable/parser_block_trailing_record.ail` (newline expression continuation in `{ }` blocks). |
| `V18` | `docs search` confinement & SimHash guard tests read | Read `cmd/ailang/docs_search_confinement_test.go`, `docs_search_policy_tool_test.go`, `docs_search_guard_test.go`, `docs_outside_repo_test.go`: (1) in-repo `docs search` MUST run SimHash over `design_docs/` (`TestDocsSearch_StillDesignDocSimHash`); (2) under `AILANG_AGENT_POLICY` confinement without local `design_docs`, `docs search` MUST exit non-zero mentioning `"confined by AILANG_AGENT_POLICY"` (`TestDocsSearch_ConfinementActualCLI`, `TestDocsSearch_ConfinementLocalAndDisabled`). |
| `V19` | Systemic check on `Ord` / `Show` synthesis | Grepped `internal/types/` for `ordRecordNames`, `DeriveOrd`, `DeriveShow`: `0` matches. `Ord` has no `deriving (Ord)` record-name bypass; `Show` is total builtin `show` (`M-V1-SHOW-TOTAL`). `eqRecordNames` is the sole bare-`TypeName` record-deriving map. |
| `V20` | Dead-code & accidental public-surface callers (`#164` / `#165`) | (a) `cmd/astdump` **HAS live callers** (`internal/parser/corpus_astdiff_test.go:125`, `scripts/test_check_protocol_closure.sh:108`) -> **KEEP `cmd/astdump/`**. (b) `prompts/devtools/v0.8.{0,1}-compact.md` **HAVE live callers** (`cmd/ailang/eval_suite_agent.go:123`, `prompts/devtools/versions.json:46`) -> **KEEP**. (c) `third_party/goopus` has `1` internal importer (`internal/builtins/audio_ogg.go:11`) -> safe to move to `internal/third_party/goopus`. (d) `dist/error_codes.json` is regenerated fresh on disk by `make error-codes` in `.github/workflows/release.yml:333` with `0` `//go:embed` or test readers -> safe to `git rm` + ignore `/dist/`. (e) `.claude/fmt_hook_events.jsonl` is runtime hook scratch (`scripts/hooks/format_ail.sh:47`, worked around in `scripts/worktree_sweep.sh:111`) -> safe to `git rm` + ignore. (f) `tests/codegen-harness/` has `0` tracked files (`tests/golden/codegen/` is active evaluator-vs-VM parity tests -> keep). |

---

## Goals

**Primary Goal:** Close all verified soundness, bytecode-parity, multi-module `deriving (Json)`, inline-test scoping, WASM/REPL validation, parser/pipe/quickfix ergonomics, capability-hygiene, SemVer public-surface, CI skill-sync, and teaching-prompt defects before tagging `v1.0.0-rc.1`, structured into **5 strictly file-disjoint parallel author slices (`M1..M5`)**.

**Success Metrics:**
- **Soundness & Parity**: `0` cross-module `Eq` witness leaks (`SOUND-1`), `0` evaluator side-effect replays on VM runtime traps (`#506`), `21/21` `CoreExpr` variants covered in `OpLowerer` + `Linker` (`#2323`), `100%` parity between CLI and `ModuleRegistry.LoadModule` on effects/exhaustiveness/contracts (`#2410`).
- **AI & Developer Ergonomics**: `0` dummy `Option`/`Result` imports needed for `deriving (Json)` in library modules, `100%` of imported/prelude constructors bound in inline `tests [...]`, `\n ?hole` and `xs |> f(a, _)` working end-to-end, `ailang check --fix` moving `IMP010` mis-imported stdlib symbols automatically, `ailang docs search` returning stdlib hits outside the repo.
- **v1 Surface & CI Hygiene**: `third_party/goopus` moved under `internal/`, `4` dead tracked scratch files removed + gitignored, `TestQuorumArtifactShellSuite` green, `v0.16.21` + `v0.16.21-compact` passing `make check-prompt-freeze` and `make check-doc-samples`, and every modified `.go` file strictly `< 800` lines (`make check-file-sizes`).

---

## High-Impact Decisions

| ID | Decision | Options & `(Recommended)` Default | Why High Impact | Chosen By | Deadline | Change Cost |
|----|----------|-----------------------------------|-----------------|-----------|----------|-------------|
| `D1` | Capability hygiene (`m-v1-public-surface-hygiene.md` M4 & M5) | **`(Recommended) D1=a`**: Ship M4 + M5 together: `--caps auto` prints `auto-granted capabilities: ...` to `stderr` unless explicit `--quiet` was passed; `ResolveAutoCaps` excludes `Process`; `ailang install` pins resolved `[bin]` shim capabilities (`--grant-new-caps` on widen); `ailang run` pre-flights entrypoint's declared direct effect row with `E_CAPS_MISSING` (exit `1`) before running.<br>`D1=b`: Ship M4 disclosure + `Process` exclusion + `[bin]` pinning only; defer `E_CAPS_MISSING` pre-flight.<br>`D1=c`: Documentation-only capability guidance. | Defines the v1.0 CLI authority contract (`A3`/`A4`) and `[bin]` shim security model. Note: pre-flight `E_CAPS_MISSING` checks `ResolveAutoCaps(iface, entry)` (the entrypoint's declared effect row) against granted caps (`--caps`, `--ai`/`--ai-stub`, `--debug`, `--release`), skipping when `--verify-contracts` or pure entrypoints run. | human | design | high |
| `D2` | Pipe `\|>` multi-argument placeholder syntax | **`(Recommended) D2=a`**: Support a single explicit `_` direct call-argument placeholder on the RHS of `\|>`: `lhs \|> f(a, _, c)` desugars to `let __pipe_arg_<n> = lhs in f(a, __pipe_arg_<n>, c)`; `0` `_` args preserves curried `lhs \|> makeAdder(1)` (`makeAdder(1)(lhs)`); `>1` `_` args raises `TC_PIPE_001`; `_?` remains an unnamed typed hole.<br>`D2=b`: Keep `\|>` strictly unary for `1.0` and teach unary/lambda idioms in `v0.16.21`. | Shapes everyday AI code generation across `std/list` (data-last) and `std/string` (data-first) before freezing `1.0` grammar. | human | design | med |
| `D3` | Active teaching prompt `v0.16.21` & `--compact` prompt | **`(Recommended) D3=a`**: Keep frozen `v0.16.20.md` and `v0.7.4-compact.md` untouched in git; author & activate `prompts/v0.16.21.md` + `prompts/v0.16.21-compact.md` (~3.5 KB accurate v0.64+ cheat-sheet), mirror to `cmd/ailang/prompts/`, and fix `resolveCompactPromptVersion` semver sorting.<br>`D3=b`: Ship `v0.16.21-compact.md` + semver sort fix only; keep `v0.16.20` active. | Controls token efficiency and zero-shot accuracy for all agents invoking `ailang prompt` and `ailang prompt --compact`. | human | design | med |
| `D4` | Additive stdlib combinators (`std/result`, `std/math`) | **`(Recommended) D4=a`**: Add `mapResult` and `collect` to `std/result.ail` plus `minInt`, `maxInt`, `clampInt` to `std/math.ail`, with `M6` regenerating `.stdlib-golden/*` via `make freeze-stdlib`.<br>`D4=b`: Zero new stdlib exports before `1.0.0-rc.1`. | Frozen in `.stdlib-golden/` once `1.0.0` tags; eliminates boilerplate recursive boilerplate + `STRICT_FALLBACK_001` friction. | human | design | med |

### Design Freeze

Before sprint execution begins, the following rulings govern implementation (pre-populated with `(Recommended)` defaults; if human review overrides an option, update the checkbox before launching `sprint-executor`):

- [x] **`D1 = D1=a` (Recommended default)**: Full M4 + M5 capability hygiene (`--caps auto` stderr disclosure unless explicit `--quiet`, `Process` excluded from `auto`, `[bin]` shim capability pinning + `--grant-new-caps`, and `E_CAPS_MISSING` pre-flight check on entrypoint effect row).
- [x] **`D2 = D2=a` (Recommended default)**: Single explicit `_` call-argument placeholder on RHS of `|>` (`lhs |> f(a, _, c)`), preserving `0`-placeholder curried `lhs |> makeAdder(1)` and `_?` typed holes.
- [x] **`D3 = D3=a` (Recommended default)**: Author and activate `v0.16.21` + `v0.16.21-compact` with semver compact resolution while leaving frozen `v0.16.20.md` and `v0.7.4-compact.md` untouched.
- [x] **`D4 = D4=a` (Recommended default)**: Add `std/result.{mapResult,collect}` and `std/math.{minInt,maxInt,clampInt}`; regenerate `.stdlib-golden/` in `M6` only.
- [x] **Keep `cmd/astdump/` and `prompts/devtools/v0.8.{0,1}-compact.md`** (`V20`: verified live callers in `corpus_astdiff_test.go`, `test_check_protocol_closure.sh`, and `eval_suite_agent.go`).

### Deferred Decisions

The following implementation details are intentionally delegated to slice workers (`agent may choose`):

- Internal helper/companion file symbol names (`internal/loader/derive_json_exports.go`, `internal/quickfix/imp010_move.go`, `internal/runner/caps_preflight.go`, `cmd/ailang/docs_search_stdlib_fallback.go`, `cmd/ailang/check_debug_timings.go`) — `agent may choose`.
- Exact hygienic temporary binder prefix for pipe placeholder desugaring (e.g. `$pipe_arg_<line>_<col>` vs monotonic elaborator counter) — `agent may choose`, provided nested pipes (`a |> f(1, _) |> g(_, 2)`) never shadow each other.
- Exact formatting and ranking score of embedded stdlib symbol hits in `ailang docs search --stdlib` / unconfined fallback — `agent may choose`, provided module path, signature, and doc comment summary are shown.
- Exact wording of `v0.16.21-compact.md` sections within the `3–4 KB` budget — `agent may choose`, provided every code snippet passes `ailang check` / `make check-doc-samples`.

---

## Conflict Surface

Because this design touches `internal/parser/`, `internal/elaborate/`, `internal/types/`, `internal/vm/`, and `cmd/ailang/`, the full conflict surface is enumerated below.

### Syntactic & Semantic Positions Touched

1. **Block newline statement-starter lookahead** (`internal/parser/parser_func.go:425-445`, `peekIsLiteralStatementStart`): adds `lexer.QUESTION` (`?name` / `?`) as a newline statement starter inside `{ ... }` blocks and function bodies.
2. **Pipe `|>` RHS call-argument placeholder** (`internal/parser/parser_expr.go` / `internal/elaborate/expr_simple.go`): inspects top-level `*ast.FuncCall` on the RHS of `|>` (`BinaryOp{Op: "|>"}`) for direct `*ast.WildcardExpr` (`_`) arguments.
3. **Nominal record `Eq` identity** (`internal/types/types.go:183-189`, `instances.go:52-57`, `instances_eq_synth.go:77-84`): adds `TypeModule string` to `types.TRecord` and keys `InstanceEnv.eqRecordNames` by `NominalKey(Module, Name)`.
4. **`--strict-fallbacks` match-arm inspection** (`internal/pipeline/strict_fallbacks.go:116-148`): inspects `*core.Match` arms before flagging `Ok([])`.
5. **`ailang docs search` backend selection** (`cmd/ailang/docs_search.go`): adds `--stdlib` flag and fallback branch after local `design_docs` and GitHub backend discovery.

### What Else Lives Here

| Position | Existing Valid Form | Token / AST Shape | Interaction Analysis |
|----------|--------------------|-------------------|----------------------|
| Inside `{ stmt1 \n <next> }` after expression | Postfix `TryExpr` (`expr?`) on **same** line | `let y = f(x)?` (`peekToken.Line == curToken.Line`) | Already consumed inside `parsePostfixExpression` / `peekSameLineQuestion` (`parser_try_let.go:59`) **before** the block loop checks `peekStartsNewlineBlockStatement`. |
| Inside `{ stmt1 \n <next> }` on **next** line | Named or anonymous typed hole (`?finish`, `?`) | `\n ? IDENT` or `\n ?` (`peekToken.Line > curToken.Line`) | Currently rejected with `PAR_UNEXPECTED_TOKEN`; adding `lexer.QUESTION` to `peekIsLiteralStatementStart` routes it to `parsePrefixExpression` -> `parseHoleExpression`. |
| Inside `{ stmt1 \n <next> }` on **next** line | Trailing record literal (`\n { a: 1 }`) | `\n LBRACE IDENT COLON ...` | Handled by `LBRACE` case in `peekIsLiteralStatementStart`; untouched. |
| RHS of `lhs \|> <rhs>` | Bare unary identifier / lambda | `x \|> double`, `x \|> (\y. y + 1)` | RHS is `Identifier` or `Lambda`, not `FuncCall` with `_` arg -> unchanged `App(rhs, [lhs])`. |
| RHS of `lhs \|> <rhs>` | Curried call returning unary function | `5 \|> makeAdder(1)` (`FuncCall` with `0` `_` args) | `0` direct `_` arguments in `makeAdder(1)` -> **unchanged** `App(makeAdder(1), [5])`! |
| RHS of `lhs \|> <rhs>` | Nested lambda containing wildcard parameter | `xs \|> map(\_. 0, _)` or `5 \|> (\_. makeAdder(1))()` | Only **direct** `FuncCall.Args[i]` matching bare `*ast.WildcardExpr` counts; `_` inside nested lambdas, patterns, or inner calls is NOT counted or substituted. |
| Call argument position | Unnamed typed hole `_?` | `f(a, _?)` (`*ast.HoleExpr{Name: ""}`) | Lexed/parsed as `HoleExpr`, NOT `WildcardExpr`; unaffected by pipe `_` desugaring. |
| Call argument position outside `\|>` | Bare `_` in normal call `f(1, _)` | `FuncCall` NOT directly on RHS of `\|>` | Continues to elaborate to existing error `wildcard '_' cannot be used as an expression`. |
| `match` arm returning `Ok([])` | Legitimate structural list base case | `[] => Ok([])` alongside `x :: rest => ...` | Exempted ONLY when arm pattern is empty `*core.ListPattern` (`Elements==0, Tail==nil`) and another arm destructures a non-empty `*core.ListPattern`. |
| `match` arm returning `Ok([])` | Error/option/wildcard swallower | `Err(_) => Ok([])`, `None => Ok([])`, `_ => Ok([])` | Pattern is `ConstructorPattern` or `WildcardPattern`, NOT empty `ListPattern` -> **still flagged** by `STRICT_FALLBACK_001`! |

### Disambiguation Strategy

- **Newline `?hole` vs Postfix `expr?`**: `parseLetRHS` and `parseCallExpression` invoke `peekSameLineQuestion()` (`p.peekTokenIs(lexer.QUESTION) && p.peekToken.Line == p.curToken.Line`). Therefore, same-line `f(x)?` is eagerly parsed as `*ast.TryExpr` while still inside expression parsing. A `?` on a new line (`p.peekToken.Line > p.curToken.Line`) is never consumed as postfix `?`; adding `lexer.QUESTION` to `peekIsLiteralStatementStart()` causes `peekStartsNewlineBlockStatement()` (which already checks `p.peekToken.Line > p.curToken.Line`) to start the next block expression cleanly.
- **Pipe `|>` `_` Placeholder vs Curried Calls**: During elaboration of `BinaryOp{Op: "|>", Left: lhs, Right: rhs}` in `internal/elaborate/expr_simple.go` (before `rhs` is elaborated as an expression, where bare `*ast.WildcardExpr` would error):
  - Check whether `rhs` (unwrapping parens if any) is `*ast.FuncCall` with `k` direct arguments `arg_i` that are `*ast.WildcardExpr`.
  - If `k == 0`: elaborate `App(elaborate(rhs), [elaborate(lhs)])` (exact existing behavior — `5 |> makeAdder(1)` works identically).
  - If `k == 1`: bind fresh hygienic `tmp := "$pipe_" + freshID`, elaborate `lhs`, substitute `Identifier{Name: tmp}` at the single `WildcardExpr` slot, and emit `Let{Name: tmp, Value: elabLHS, Body: elabModifiedCall}` (preserving left-to-right evaluation of `lhs` before call arguments and avoiding duplicate evaluation).
  - If `k > 1`: return structured error `TC_PIPE_001: pipe stage has multiple '_' placeholders; use a lambda (\x. ...) instead`.
- **Cross-Module `TRecord` Nominal `Eq`**: `types.TRecord` gains `TypeModule string`. Same-module or imported `deriving (Eq)` records stamp both `TypeModule` (canonical module path, e.g. `"std/regex"` or `"pkg/a"`) and `TypeName`. `InstanceEnv.eqRecordNames` stores `NominalKey(c.Module, c.Name)` (falling back to bare `TypeName` ONLY when both `rec.TypeModule == ""` and an intra-module legacy test constructs `TRecord` without module context, never matching across distinct non-empty modules).

### Programs That MUST Still Work (Mandatory Regression Surface)

1. `examples/runnable/pipe_operator.ail` — exercises `5 |> double`, multi-stage pipelines, record/list piping, and curried `5 |> makeAdder(1)` (line 27).
2. `examples/runnable/parser_block_trailing_record.ail` — exercises multi-statement `{ let ... \n { field: val } }` newline statement boundaries inside blocks.
3. `examples/runnable/record_patterns.ail` — exercises closed `{name, age}` and open `{name, ...}` record destructuring in `match`, `let`, and lambdas.
4. `examples/runnable/derive_json.ail` — exercises single-module record/ADT `deriving (Json)` round-trip encoding and decoding.
5. `cmd/ailang/docs_search_confinement_test.go` & `cmd/ailang/docs_search_guard_test.go` — exercises `AILANG_AGENT_POLICY` confinement refusal and in-repo `design_docs/` SimHash search.

### What Deliberately Changes

1. **Cross-module record alias `==` without `deriving (Eq)` (`SOUND-1`)**: Programs that accidentally relied on an imported ADT/record of the same bare name `T` (`deriving (Eq)`) to compare a local `type T = { ... }` without `deriving (Eq)` now fail type-checking with `No instance for Eq[...]`. Migration: add `deriving (Eq)` to the record declaration (if all fields support `Eq`).
2. **VM closure equality (`RT_EQ_001`)**: Any closure reaching `OpEq`/`OpNeq` on the bytecode VM now traps with structured runtime error `RT_EQ_001` instead of returning pointer-identity `bool`.
3. **`--bytecode` post-dispatch runtime traps (`#506`)**: Division by zero (`RT001`), recursion overflow (`RT_REC_003`), and closure equality (`RT_EQ_001`) on `--bytecode` fail immediately with single `vm: ...` prefix instead of printing `bytecode path unavailable` and re-running on the evaluator.
4. **`--args-json` non-integral floats (`#2399`)**: Passing `2.9` or `1e30` to an `int` entrypoint via `--args-json` now fails argument decoding instead of silently truncating to `2`.
5. **Capability hygiene (`D1=a`)**:
   - `ailang run --caps auto file.ail` prints `auto-granted capabilities: <list|none>` to `stderr`; pass `--quiet` explicitly (`ailang run --quiet --caps auto file.ail`) to suppress.
   - `--caps auto` no longer grants `Process`; pass `--caps Process` or `--caps auto,Process`.
   - Running an effectful entrypoint without required `--caps` fails pre-flight with `E_CAPS_MISSING` (exit `1`) before executing user expressions.
6. **`third_party/goopus` import path (`SEMVER-GO`)**: Moves to `github.com/sunholo-data/ailang/internal/third_party/goopus`.

---

## Solution Design

### Overview

The sprint is partitioned into **five strictly file-disjoint author milestones (`M1..M5`)** that run concurrently in isolated `share` workspaces, followed by a single **Integration & Artifact Regeneration milestone (`M6`)** that owns all shared generated artifacts (`.stdlib-golden/*`, `docs/docs/prompts/current.md`, `llms.txt`) and an independent **Verification milestone (`M7`)**.

Every existing `.go` file near the 800-line ceiling (`loader.go` `793L`, `check.go` `761L`, `run.go` `743L`, `quickfix.go` `728L`, `op_lowering.go` `703L`) is protected by extracting new logic into focused companion `.go` files in the same package so `make check-file-sizes` stays 100% green.

### Component Architecture by Milestone

#### M1: Soundness, Bytecode Trap Isolation, Exhaustive Lowering & Boundary Decoding (`M1_SOUNDNESS_VM_LOWERING`)

1. **`SOUND-1` Nominal Record `Eq` + Contravariant Pattern Hint**:
   - Add `TypeModule string` to `types.TRecord` (`internal/types/types.go`) and preserve `(TypeModule, TypeName)` through `Equals`, `Substitute` (`unification_substitution.go`, `typechecker_substitution.go`), record unification (`unification_records.go`, `unification_core.go`), `elaborate/core.go` (record alias expansion stamps `tc.currentModule` or imported alias origin module), `pipeline/alias_body_closure.go`, and `pipeline/cache_coreti_dag.go` (+ bump `cache_key.go` schema salt).
   - Key `InstanceEnv.eqRecordNames` (`internal/types/instances.go`) by `NominalKey(c.Module, c.Name)`. Update `HasDerivedEqRecord(module, name string)` in `instances_eq_synth.go` (`synthesizeEq` and `reduceEq`) so cross-module bare-name collisions never synthesize `StructuralEq`.
   - In `internal/types/unification_records.go` (`recordFieldMismatchError`), when `missingFields` is non-empty on a contravariant parameter / pattern mismatch, append a concise secondary hint noting open record patterns `{a, b, ...}` if destructuring a record parameter.
   - Change `runtimeEq` in `internal/vm/vm_arith.go` to return `(bool, error)` and return `NewVMError("RT_EQ_001", "cannot compare function/closure values with == or !=")` when either operand has `bytecode.TagClosure` (including nested inside `TagRecord`, `TagTuple`, `TagList`, `TagArray`, `TagADT`), while leaving `bytecode.Value.Equal` untouched for compile-time constant-pool deduplication (`internal/bytecode/image.go:143`).
2. **`PARITY-B4` (`#506`) No Evaluator Replay After VM Dispatch + Clean Prefix**:
   - In `internal/runner/vm.go:281-289`, once `machine.Run(proto, bcArgs)` has been invoked, any non-nil `err` is a post-dispatch runtime failure: return `true, err` directly (no `fmt.Errorf("vm: %w", err)` double-wrapping when `err` is already a `*vm.VMError` or prefixed with `"vm: "`). All pre-dispatch ineligibility paths (`Compile`, `Validate`, `StrictEvalOnlyError`) remain before line `244`.
3. **`#2323` Exhaustive 21-Variant `CoreExpr` Coverage in `OpLowerer` & `Linker`**:
   - Extend `OpLowerer.lowerExpr` (`internal/pipeline/op_lowering.go` — extract composite helpers into `internal/pipeline/op_lowering_containers.go` to keep `op_lowering.go` < 710L) to lower sub-expressions inside `*core.Tuple`, `*core.Array`, `*core.RecordUpdate`, `*core.DictAbs`, `*core.DictApp`, and `*core.Forall`.
   - Extend `Linker.collectDictRefs` (`internal/link/linker.go`) to walk `prog.Meta[*].Contracts[*].Expr` and extend `collectFromExpr` to walk `arm.Guard` in `*core.Match`, `*core.Tuple`, `*core.Array`, `*core.RecordUpdate`, `*core.Intrinsic`, and `*core.Forall`.

| `CoreExpr` Variant (21 total in `internal/core/core.go`) | `OpLowerer` Before -> After | `Linker` Before -> After |
|----------------------------------------------------------|-----------------------------|--------------------------|
| `Var`, `VarGlobal`, `Lit`, `DictRef`, `Atomic` (leaf/atomic) | Handled -> Handled | Handled -> Handled |
| `Lambda`, `Let`, `LetRec`, `App`, `If`, `Record`, `RecordAccess`, `List` | Handled -> Handled | Handled -> Handled |
| `Match` (scrutinee + arm `Guard` + arm `Body`) | Handled -> Handled | `Guard` missing -> **Fixed** |
| `Intrinsic` | Handled -> Handled | Missing -> **Fixed** |
| `Tuple`, `Array`, `RecordUpdate`, `Forall` | **Missing -> Fixed** | **Missing -> Fixed** |
| `DictAbs`, `DictApp` | **Missing -> Fixed** | Handled -> Handled |
| `prog.Meta[*].Contracts[*].Expr` (`requires`/`ensures`) | Handled (`:102-121`) -> Handled | **Missing -> Fixed** |

4. **`#2399` Strict Integer Decoding & `serve-api` `std/json.Json` Binding**:
   - In `internal/runtime/argdecode/argdecode.go`: switch `DecodeJSON` to `json.NewDecoder` with `UseNumber()`; in `decodeInt`, accept `json.Number` (via `Int64()` or exact integral `Float64()` within `[math.MinInt64, math.MaxInt64]` with zero fractional part) and reject `2.9`, `NaN`, `Inf`, and out-of-range numbers with a clear error.
   - In `internal/apiserver/param_schema.go` and `internal/apiserver/arg_bind.go`: recognize `std/json.Json` (`SimpleType{Name: "Json"}` / `Json` ADT) as `Kind: "json"` (`schema: {}` in OpenAPI) and recursively convert `any` (`nil`, `bool`, `json.Number`/`float64`/`int`, `string`, `[]any`, `map[string]any`) into canonical `std/json.Json` `*eval.TaggedValue` constructors (`JNull`, `JBool`, `JInt`, `JNumber`, `JString`, `JArray`, `JObject` with `{key, value}` records).

### Pipeline Pass Coverage

- [x] `prog.Decls` — top-level declarations walked by both `OpLowerer.Lower` and `Linker.collectDictRefs`
- [x] `prog.Meta[*].Contracts[*].Expr` — contract expressions (`requires`/`ensures`) walked by `OpLowerer.Lower` (`op_lowering.go:102-121`) AND added to `Linker.collectDictRefs` (`linker.go`)
- [x] Integration test: `TestContractExpressionsFullyLowered` + new `TestLinkerCollectsContractDictRefs` pass

#### M2: Multi-Module `deriving (Json)`, Inline `tests [...]` Scope & WASM/REPL Parity (`M2_DERIVE_TESTS_REPL`)

1. **`DERIVE-JSON` Multi-Module Exports + Prelude Hygiene**:
   - Create `internal/loader/derive_json_exports.go` (`loader.go` is `793L` — add only a 1-line call `registerDerivedJsonExports(file, loaded)` in `loader.go`!): scan `file.Decls` for exported `*ast.TypeDecl` with `ast.DeriveJson` in `Deriving` and synthesize stub exported `*ast.FuncDecl` entries (`encode<T>` and `decode<T>`) in `loaded.Exports` so `moduleLoader.GetImportExport` succeeds before sibling elaboration.
   - In `internal/loader/prelude_imports.go`: add `fileHasDerivingJson(file *ast.File) bool`; when true, treat the file as using both `Option` (`std/option`) and `Result` (`std/result`) during `injectPreludeImports` (preserving existing local-declaration shadowing guards `declaresOption` / `declaresResult`).
2. **`INLINE-TESTS-SCOPE` Imported & Prelude Constructors in `ailang test`**:
   - In `internal/testing/executor_helpers.go` (`injectADTConstructors`) and `internal/testing/module_scope.go`: in addition to local `e.sourceFile.Decls`, bind constructors from `std/option` (`Some`, `None`) and `std/result` (`Ok`, `Err`) (unless shadowed locally) plus any constructor explicitly imported in `e.sourceFile.Imports` (honouring `SymbolAliases`, e.g. `import pkg/m (Foo as MyFoo)`) by resolving the constructor metadata from `e.modules[imp.Path].Iface.Constructors`.
3. **`#2410` WASM / Playground / REPL `ModuleRegistry.LoadModule` Parity**:
   - In `internal/repl/module_registry_load.go`: wire `registerDerivedEq(typeChecker, elabProg)` (or shared helper), `typeChecker.SetMatchUniverse(...)` + `typeChecker.VerifyMatchExhaustiveness(decl, typedNode)` (`TC_MATCH_002` errors/warnings), `pipeline.ValidateEffectsWithCalls(astFile, coreProg, coreTI, ...)` (so `pure func` side effects fail on `:load` and WASM `loadModule`), and contract expression type-checking (`prog.Meta[*].Contracts`).

#### M3: Block Hole Disambiguation, Pipe Placeholder, `STRICT_FALLBACK_001` & `check --fix` (`M3_PARSER_PIPE_QUICKFIX`)

1. **`?name` Typed Hole After Newline in Blocks**:
   - Add `lexer.QUESTION` to `peekIsLiteralStatementStart()` in `internal/parser/parser_func.go:425-445` so `\n ?finish` and `\n ?` start a new statement in both function bodies and `{ ... }` blocks while same-line `expr?` remains postfix `TryExpr`.
2. **Pipe `|>` Explicit `_` Argument Placeholder (`D2=a`)**:
   - In `internal/elaborate/expr_simple.go` (`BinaryOp` `"|>"` branch): implement single-placeholder call-argument desugaring (`lhs |> f(a, _, c)` -> `let $pipe_n = lhs in f(a, $pipe_n, c)`), preserving `0`-placeholder curried calls (`5 |> makeAdder(1)`) and rejecting `>1` `_` placeholders with `TC_PIPE_001`.
3. **`STRICT_FALLBACK_001` Base-Case Exemption**:
   - In `internal/pipeline/strict_fallbacks.go`: exempt a `Match` arm returning `Ok([])` iff (a) the arm's pattern is an empty `*core.ListPattern` (`len(Elements) == 0 && Tail == nil`) AND (b) the enclosing `*core.Match` has at least one sibling arm with a non-empty `*core.ListPattern` (`len(Elements) > 0 || Tail != nil`). Keep `Err(_) => Ok([])`, `None => Ok([])`, and `_ => Ok([])` flagged.
4. **`IMP010` Cross-Module `ailang check --fix`**:
   - Create `internal/quickfix/imp010_move.go` (`quickfix.go` is `728L`): recognize `IMP010` diagnostics whose suggestion/message identifies a single canonical exporting module (`is exported by <mod>`), compute line-accurate edits removing `<sym>` from the offending `import ... (...)` list (or removing the entire `import` line when `<sym>` is the sole imported symbol) and merging `<sym>` into an existing `import <mod> (...)` declaration or inserting `import <mod> (<sym>)` immediately after the import block.
5. **Block `let` Hints & Warning Spans**:
   - Track `blockDepth` in `internal/parser/parser_expr.go` / `parser_decl.go` (`PAR014`/`PAR015`) so `let` syntax errors inside `{ ... }` suggest `;` or newline-separated `let x = expr` instead of top-level `let x = expr in body`.
   - Pass `warningFiler` source locations in `internal/pipeline/warn_split_args.go` and `internal/pipeline/warn_take_after_flatmap.go`.

#### M4: Stdlib Ergonomics, Offline `docs search`, `--json` Purity, Capability & SemVer Hygiene (`M4_STDLIB_CLI_CAPS`)

1. **Additive Stdlib Combinators (`D4=a`)**:
   - `std/result.ail`: add `export pure func mapResult[a, b, e](f: a -> Result[b, e], xs: [a]) -> Result[[b], e]` and `export pure func collect[a, e](rs: [Result[a, e]]) -> Result[[a], e]` with inline/module tests (do NOT touch `.stdlib-golden/*` in `M4`; `M6` freezes stdlib).
   - `std/math.ail`: add `export pure func minInt(a: int, b: int) -> int`, `maxInt(a: int, b: int) -> int`, and `clampInt(x: int, lo: int, hi: int) -> int`.
2. **`ailang docs search` Embedded Stdlib Fallback + `--stdlib`**:
   - Add `--stdlib` flag in `cmd/ailang/docs_search.go` and implement embedded `std/*.ail` search in new `cmd/ailang/docs_search_stdlib_fallback.go`.
   - Trigger automatic stdlib fallback **only** when `*pathFlag == ""` AND `localDocsErr != nil` AND `!pkg.IsConfined()` AND `!config.NoGitHubSearch()` AND `!*noGitHubFlag` AND `githubsearch.NewBackend(ctx)` returns an error — keeping all `TestDocsSearch_Confinement*` and `TestDocsSearch_StillDesignDocSimHash` invariants 100% green while making `ailang docs search "regex"` work offline outside the repo.
3. **`ailang check --debug-compile --json` Stdout Purity**:
   - Extract debug timing printer from `cmd/ailang/check.go` (`761L`) into `cmd/ailang/check_debug_timings.go` and route `[DEBUG-COMPILE]` output to `os.Stderr` whenever `--json` is active.
4. **Capability Hygiene (`D1=a`) & Go Public Surface (`SEMVER-GO`)**:
   - `cmd/ailang/main_run.go` + `internal/runner/run.go` + `internal/runner/caps.go` + new `internal/runner/caps_preflight.go`:
     - Track `ExplicitQuiet` via `fs.Visit` so `--caps auto` prints `auto-granted capabilities: <list|none>` to `stderr` unless `--quiet` was explicitly passed.
     - Exclude `"Process"` in `ResolveAutoCaps` (emitting a stderr note when `Process` is on the entrypoint row during `--caps auto` telling the user to pass `--caps auto,Process`).
     - Pre-flight check `CheckMissingEntryCaps` in `caps_preflight.go`: compare `ResolveAutoCaps(result.Interface, entry)` (filtering `"Debug"` when `opts.Release` or `!opts.Debug` ghost-erased — verify exact Debug ghost behavior in `runSingle`!) against granted capabilities (`--caps`, `--ai`/`--ai-stub` -> `AI`, `--debug` -> `Debug`); if missing, print structured `E_CAPS_MISSING` error naming the exact `--caps <list>` flag to add and return exit code `1` before executing user code.
   - `internal/pkg/bin.go` + `cmd/ailang/pkg_bin.go`: pin resolved capability list into installed `[bin]` shims at `ailang install` time (`--caps IO,FS` instead of `--caps auto` when `caps` is unspecified in `[bin]`), refusing silent privilege escalation on re-install unless `--grant-new-caps` is passed.
   - Move `third_party/goopus/*` -> `internal/third_party/goopus/*`, update `internal/builtins/audio_ogg.go`, add `doc.go` `Internal` notices to `serveapi/...`, and add the `D-67` Go module API exclusion + Internal binary split section to `docs/docs/reference/stability.md` (preserving all 16 deprecation table rows required by `TestStabilityDocListsEveryDeprecation`).

#### M5: CI Skill Sync, Prompt `v0.16.21` + `--compact`, Dead Clutter Sweep & Limitations Sync (`M5_PROMPT_CI_DOCS`)

1. **`CI-1` Skill Sync**: Run `bash tools/launchd/sync-agents-skills.sh` to synchronize `.agents/skills/{mission-loop-change,local-model-onboarding,local-ollama-eval}` from `.claude/skills/*`.
2. **Teaching Prompt `v0.16.21` + `v0.16.21-compact` + Semver Compact Sort (`D3=a`)**:
   - Leave frozen `prompts/v0.16.20.md` and `prompts/v0.7.4-compact.md` (and active `prompts/devtools/v0.8.{0,1}-compact.md`) completely untouched.
   - Author `prompts/v0.16.21.md` (fixing all 6 audited contradictions: library-module `Option`/`Result` notes, zero removed `@deprecated` identifiers, braced `match` / `= {`, canonical `\acc x. ...` multi-arg lambdas, `std/xml` high-level wrappers, canonical `ailang pkg` commands, `sunholo/gcp_auth` naming; adding `! {IO @limit=5}`, `import ./sibling`, `tests [((a, b), out)]`, and `xs |> filter(p, _)`) and `prompts/v0.16.21-compact.md` (~3.5 KB accurate v0.64+ cheat-sheet); mirror both into `cmd/ailang/prompts/` and register in both `prompts/versions.json` and `cmd/ailang/prompts/versions.json`.
   - Fix `resolveCompactPromptVersion` (`cmd/ailang/prompt.go:138-153`) to compare semantic version tuples `(major, minor, patch)` instead of ASCII `sort.Strings`, so `v0.16.21-compact` beats `v0.7.4-compact`.
3. **Dead Tracked Clutter Sweep (`#164` / `#165`)**:
   - `git rm` `.claude/fmt_hook_events.jsonl`, `dist/error_codes.json`, `examples/manifest.json.bak`, and `examples/manifest.tmp`.
   - Update `.gitignore` to ignore `.claude/fmt_hook_events.jsonl`, `/dist/`, `examples/manifest.json.bak`, `examples/manifest.tmp`, and re-admit `!/.ailang/state/evaluations/`, `!/.ailang/state/evaluations/*`, `!/.ailang/state/implementation/`, `!/.ailang/state/implementation/*`.
4. **Limitations Sync & Showcase Example**:
   - Update `docs/LIMITATIONS.md` and `docs/docs/reference/limitations.md` (keeping frontmatter `reviewBy` intact) for `v0.64.2 / v1.0.0-rc.1` reality (`#2404`, `#2410`, Z3 `requires` caller boundary, nominal `Eq`, pipe `_` placeholder).
   - Add `examples/runnable/v1_dx_showcase.ail` (exercising `requires`/`ensures` contracts, `! {IO @limit=5}` effects, pipe `_` placeholder, `std/result.mapResult`, and inline `tests [...]`) and extend `examples/runnable/pipe_operator.ail` with `_` placeholder examples.

---

## Pairwise-Disjoint File Ownership Matrix (`M1..M5`) & `<800L` File-Size Budget

Every modified or created path in `M1..M5` belongs to **at most one** author milestone (`Files(M_i) ∩ Files(M_j) = ∅` for all `1 <= i < j <= 5`). Shared generated artifacts (`.stdlib-golden/*`, `docs/docs/prompts/current.md`, `llms.txt`, `examples/manifest.json` if needed) are touched **exclusively** in `M6_INTEGRATION` after `M1..M5` merge.

| Milestone | Owned Production & Doc Files (Current Lines -> Target Ceiling) | New Companion / Test Files Owned |
|-----------|----------------------------------------------------------------|----------------------------------|
| **`M1_SOUNDNESS_VM_LOWERING`** (`rc1-soundness-vm`) | `internal/types/types.go` (`550L -> <575L`), `internal/types/instances.go` (`199L -> <225L`), `internal/types/instances_eq_synth.go` (`285L -> <320L`), `internal/types/unification_records.go` (`377L -> <405L`), `internal/types/unification_core.go` (`514L -> <535L`), `internal/types/unification_substitution.go` (`242L -> <260L`), `internal/types/typechecker_substitution.go` (`256L -> <275L`), `internal/elaborate/core.go` (`385L -> <405L`), `internal/pipeline/alias_body_closure.go` (`144L -> <165L`), `internal/pipeline/cache_coreti_dag.go` (`545L -> <570L`), `internal/pipeline/cache_key.go` (`145L -> <155L`), `internal/pipeline/op_lowering.go` (`703L -> <715L`), `internal/link/linker.go` (`415L -> <460L`), `internal/runner/vm.go` (`304L -> <315L`), `internal/vm/vm_arith.go` (`236L -> <275L`), `internal/runtime/argdecode/argdecode.go` (`253L -> <295L`), `internal/apiserver/param_schema.go` (`125L -> <150L`), `internal/apiserver/arg_bind.go` (`134L -> <210L`) | `internal/pipeline/op_lowering_containers.go` (~80L), `internal/types/sound1_cross_module_eq_test.go`, `internal/runner/vm_trap_no_fallback_test.go`, `internal/pipeline/op_lowering_containers_test.go`, `internal/link/linker_contracts_test.go`, `internal/runtime/argdecode/argdecode_strict_int_test.go`, `internal/apiserver/arg_bind_json_adt_test.go` |
| **`M2_DERIVE_TESTS_REPL`** (`rc1-derive-tests-repl`) | `internal/loader/loader.go` (**`793L -> <=796L`**, 1-line hook only!), `internal/loader/prelude_imports.go` (`201L -> <245L`), `internal/elaborate/derive_json.go` (`618L -> <640L` if needed), `internal/elaborate/derive_json_decode.go` (`428L -> <450L` if needed), `internal/testing/executor_helpers.go` (`572L -> <630L`), `internal/testing/module_scope.go` (`241L -> <280L`), `internal/repl/module_registry_load.go` (`412L -> <510L`), `internal/repl/repl_eval.go` (`389L -> <415L`) | `internal/loader/derive_json_exports.go` (~95L), `internal/loader/derive_json_multimodule_test.go`, `internal/testing/inline_tests_ctor_scope_test.go`, `internal/repl/module_registry_parity_test.go` |
| **`M3_PARSER_PIPE_QUICKFIX`** (`rc1-parser-quickfix`) | `internal/parser/parser_func.go` (`455L -> <470L`), `internal/parser/parser_expr.go` (`660L -> <690L`), `internal/parser/parser_decl.go` (`679L -> <700L`), `internal/parser/parser_error.go` (`198L -> <215L`), `internal/elaborate/expr_simple.go` (`488L -> <560L`), `internal/pipeline/strict_fallbacks.go` (`215L -> <260L`), `internal/pipeline/warn_split_args.go` (`118L -> <135L`), `internal/pipeline/warn_take_after_flatmap.go` (`142L -> <160L`), `internal/quickfix/quickfix.go` (**`728L -> <=740L`**, dispatch hook only!) | `internal/quickfix/imp010_move.go` (~160L), `internal/parser/hole_newline_block_test.go`, `internal/elaborate/pipe_placeholder_test.go`, `internal/pipeline/strict_fallbacks_basecase_test.go`, `internal/quickfix/imp010_move_test.go` |
| **`M4_STDLIB_CLI_CAPS`** (`rc1-stdlib-cli-caps`) | `std/result.ail` (`112L -> <175L`), `std/math.ail` (`148L -> <195L`), `cmd/ailang/docs_search.go` (`265L -> <300L`), `cmd/ailang/check.go` (**`761L -> <735L`**, extract debug timing helper!), `cmd/ailang/main_run.go` (`303L -> <325L`), `cmd/ailang/pkg_bin.go` (`142L -> <185L`), `internal/runner/run.go` (**`743L -> <=755L`**, call `caps_preflight.go`!), `internal/runner/caps.go` (`104L -> <135L`), `internal/pkg/bin.go` (`296L -> <360L`), `internal/builtins/audio_ogg.go` (`112L -> 112L`), `docs/docs/reference/stability.md` (`318L -> <380L`), `third_party/goopus/**` -> `internal/third_party/goopus/**` | `cmd/ailang/docs_search_stdlib_fallback.go` (~140L), `cmd/ailang/docs_search_stdlib_fallback_test.go`, `cmd/ailang/check_debug_timings.go` (~55L), `cmd/ailang/check_json_debug_test.go`, `internal/runner/caps_preflight.go` (~110L), `internal/runner/caps_preflight_test.go`, `serveapi/doc.go` (+ subpackage `doc.go` files) |
| **`M5_PROMPT_CI_DOCS`** (`rc1-prompt-ci-hygiene`) | `.agents/skills/{mission-loop-change,local-model-onboarding,local-ollama-eval}/**` (synced from `.claude/skills/`), `prompts/versions.json`, `cmd/ailang/prompts/versions.json`, `cmd/ailang/prompt.go` (`285L -> <325L`), `.gitignore` (`159L -> <178L`), `docs/LIMITATIONS.md`, `docs/docs/reference/limitations.md`, `examples/runnable/pipe_operator.ail`; `git rm`: `.claude/fmt_hook_events.jsonl`, `dist/error_codes.json`, `examples/manifest.json.bak`, `examples/manifest.tmp` | `prompts/v0.16.21.md`, `prompts/v0.16.21-compact.md`, `cmd/ailang/prompts/v0.16.21.md`, `cmd/ailang/prompts/v0.16.21-compact.md`, `cmd/ailang/prompt_compact_test.go`, `examples/runnable/v1_dx_showcase.ail` |

---

## Testing Strategy

| Test ID | Milestone | Level | Target & Assertion |
|---------|-----------|-------|--------------------|
| `TST-RC1-01` | `M1` | Unit + Engine Parity | Cross-module `type Regex = { pattern: string, matcher: string -> bool }` alongside `import std/regex` is rejected by `ailang check` (`No instance for Eq`); intra-module `export type Pt = { x: int, y: int } deriving (Eq)` still passes on evaluator and `--strict-bytecode`. |
| `TST-RC1-02` | `M1` | Unit (VM) | VM `OpEq`/`OpNeq` on direct and record/tuple/list/ADT-nested `TagClosure` returns `RT_EQ_001` error; `bytecode.Value.Equal` constant-pool dedup tests still pass. |
| `TST-RC1-03` | `M1` | Integration (`#506`) | Effectful entrypoint printing `"SIDE_EFFECT"` before `10 / 0` or stack overflow under `--bytecode` prints `"SIDE_EFFECT"` **once**, emits zero fallback warning, and has single `vm: RT001:` prefix; `BC001` compile-ineligible module still falls back cleanly under `--bytecode`. |
| `TST-RC1-04` | `M1` | Unit + Parity (`#2323`) | Operators inside lambdas nested in `Tuple`, `Array`, `RecordUpdate`, `DictAbs`, `DictApp`, `Forall`, `Match` guards, and `requires`/`ensures` contracts evaluate identically on evaluator and `--strict-bytecode`; `TestContractExpressionsFullyLowered` passes. |
| `TST-RC1-05` | `M1` | Unit + HTTP (`#2399`) | `argdecode.DecodeJSON` rejects `2.9`, `1e30`, `NaN` for `int`, accepts `42` and `2.0` (exact integer) or documents exact JSON integer policy consistently, preserves 64-bit integer precision via `UseNumber()`, and `serve-api` binds `std/json.Json` requests without HTTP 500. |
| `TST-RC1-06` | `M2` | Integration | Multi-module `export type Item = { ... } deriving (Json)` in library module `types.ail` (with ONLY `import std/json (Json)`) imported via `import ./types (Item, encodeItem, decodeItem)` checks and runs on evaluator + `--strict-bytecode`. |
| `TST-RC1-07` | `M2` | Unit (`DERIVE-JSON` shadow) | Library module that defines its own local `type Option = ...` alongside `deriving (Json)` does not collide with prelude injection. |
| `TST-RC1-08` | `M2` | Integration (`ailang test`) | Inline `tests [...]` referencing `Some`/`None`, `Ok`/`Err`, and selectively imported/aliased ADT constructors pass `ailang test`. |
| `TST-RC1-09` | `M2` | Unit + WASM/REPL (`#2410`) | `ModuleRegistry.LoadModule` rejects `pure func` calling `println` (effect violation) and non-exhaustive `match` (`TC_MATCH_002`), and supports `deriving (Eq)` across loaded modules. |
| `TST-RC1-10` | `M3` | Parser + Regression | `{ let y = x * 2 \n ?finish }` parses `?finish` as `*ast.HoleExpr{Name: "finish"}`; same-line `let y = f(x)?` still parses `*ast.TryExpr`; `examples/runnable/parser_block_trailing_record.ail` passes. |
| `TST-RC1-11` | `M3` | Elaborator + Parity (`D2`) | `xs |> filter(\x. x > 0, _) |> map(\x. x * 10, _)` and `"a,b" |> split(_, ",")` pass on evaluator + `--strict-bytecode`; curried `5 |> makeAdder(1)` (`examples/runnable/pipe_operator.ail`) still returns `6`; `x |> add(_, _)` fails with `TC_PIPE_001`; `f(1, _?)` remains a typed hole. |
| `TST-RC1-12` | `M3` | Pipeline (`STRICT_FALLBACK_001`) | Structural recursion `match xs { [] => Ok([]), x :: rest => ... }` passes `--strict-fallbacks`; `match r { Ok(xs) => Ok(xs), Err(_) => Ok([]) }` and `_ => Ok([])` STILL fail with `STRICT_FALLBACK_001`. |
| `TST-RC1-13` | `M3` | Quickfix (`IMP010`) | `ailang check --fix` on `import std/list (map, join)` rewrites imports to `import std/list (map)` + `import std/string (join)` (and deletes single-symbol `import std/list (join)`), producing a zero-error file on re-check. |
| `TST-RC1-14` | `M3` | Diagnostics | `let` error inside `{ ... }` does not suggest `in`; `W_SPLIT_ARGS` and `LIST_TAKE_AFTER_FLATMAP` carry non-empty file:line spans. |
| `TST-RC1-15` | `M4` | Stdlib (`D4`) | `std/result.{mapResult,collect}` (short-circuit on first `Err`, order preservation on `Ok`) and `std/math.{minInt,maxInt,clampInt}` pass `ailang test` and `--strict-bytecode`. |
| `TST-RC1-16` | `M4` | CLI + Confinement | `ailang docs search --stdlib "regex"` and unconfined out-of-repo `ailang docs search "split"` (when GitHub backend unavailable) return stdlib matches; all `TestDocsSearch_Confinement*` and `TestDocsSearch_StillDesignDocSimHash` tests pass unchanged. |
| `TST-RC1-17` | `M4` | CLI JSON | `ailang check --debug-compile --json <file>` emits valid JSON on `stdout` (`jq -e .`) and `[DEBUG-COMPILE]` timings on `stderr`. |
| `TST-RC1-18` | `M4` | Capabilities (`D1`) | `--caps auto` logs to `stderr` unless explicit `--quiet`; `ResolveAutoCaps` omits `Process`; `[bin]` shims pin resolved caps and require `--grant-new-caps` to widen; missing caps fail pre-flight with `E_CAPS_MISSING`. |
| `TST-RC1-19` | `M4` | Surface (`SEMVER-GO`) | `third_party/goopus` no longer exists at module root; `go test ./internal/builtins/... ./serveapi/... ./docs/...` (`TestStabilityDocListsEveryDeprecation`) pass. |
| `TST-RC1-20` | `M5` | CI + Prompts (`CI-1`, `D3`) | `TestQuorumArtifactShellSuite` passes; `ailang prompt --version-active` returns `v0.16.21`; `ailang prompt --compact` returns `v0.16.21-compact` with zero `**`/`abs_Int`; `make check-prompt-freeze` passes. |
| `TST-RC1-21` | `M5` + `M6` | Showcase + Golden Gates | `examples/runnable/v1_dx_showcase.ail` passes `ailang check --strict-fallbacks`, `ailang test`, `ailang run --caps IO`, and `ailang run --strict-bytecode --caps IO`; `make verify-stdlib`, `make check-llms-txt`, `make check-doc-samples`, `make check-file-sizes`, `make check-boundaries`, and `cd docs && npm run build` all pass. |
| `TST-RC1-22` | `M7` | Adversarial Verification | Independent `rc1-verifier` confirms zero regressions across all 5 Conflict Surface fixtures and adversarial negative controls. |

---

## Non-Goals

- **Deleting `cmd/astdump/` or `prompts/devtools/v0.8.{0,1}-compact.md`**: Verified active in `corpus_astdiff_test.go`, `test_check_protocol_closure.sh`, and `eval_suite_agent.go` (`V20`).
- **Go codegen emitter work**: Removed in `#2389`; engine parity means **tree-walking evaluator + bytecode VM** only.
- **Multi-placeholder `|>` currying (`x |> f(_, _)`)**: Deliberately rejected (`TC_PIPE_001`) to preserve 1-to-1 evaluation order and unambiguous readability; multi-use binds use `\x. f(x, x)`.
- **Auto-granting `Process` under `--caps auto`**: Deliberately excluded under `A4` (Explicit Authority) so arbitrary subprocess execution always requires explicit opt-in.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Touching files near the 800-line ceiling (`loader.go` `793L`, `check.go` `761L`, `run.go` `743L`, `quickfix.go` `728L`) | High (CI gate `make check-file-sizes` hard-fails >= 800L) | Extract new logic into companion files (`derive_json_exports.go`, `check_debug_timings.go`, `caps_preflight.go`, `imp010_move.go`, `op_lowering_containers.go`) and enforce `make check-file-sizes` per slice. |
| `TRecord.TypeModule` compile-cache serialization mismatch | Medium | Update `internal/pipeline/cache_coreti_dag.go` encode/decode for `TRecord.TypeModule` AND bump `cache_key.go` schema version in `M1`. |
| `E_CAPS_MISSING` pre-flight false positive on erased `Debug` ghost effect or pure entrypoints | Medium | `CheckMissingEntryCaps` strips ghost `Debug` when `!opts.Debug` (matching runtime ghost erasure) and only checks the resolved entrypoint's `TFunc2.EffectRow.Labels`. |
| `docs search` stdlib fallback breaking `AILANG_AGENT_POLICY` confinement tests | High | Gate fallback strictly behind `!pkg.IsConfined() && !config.NoGitHubSearch() && !*noGitHubFlag && *pathFlag == ""` and run `go test ./cmd/ailang -run TestDocsSearch` in `M4`. |
| Modifying frozen prompts breaking `make check-prompt-freeze` | High | Leave `prompts/v0.16.20.md` and `prompts/v0.7.4-compact.md` byte-identical; add `v0.16.21` + `v0.16.21-compact` and fix semver sort in `cmd/ailang/prompt.go`. |

---

## Success Criteria

- [ ] All `TST-RC1-01` through `TST-RC1-22` tests implemented and passing.
- [ ] Pairwise file disjointness (`Files(M_i) ∩ Files(M_j) = ∅`) maintained across `M1..M5` with zero merge conflicts in `M6`.
- [ ] `make check-file-sizes`, `make check-boundaries`, `make check-prompt-freeze`, `make verify-stdlib`, `make check-llms-txt`, `make check-doc-samples`, `go test ./internal/mission/quorum/... -run TestQuorumArtifactShellSuite`, `go run ./scripts/verify_bytecode_parity.go --strict`, and `cd docs && npm run build` all pass `0` errors.

---

## References

- **Audit report**: `v1_readiness_and_ai_ergonomics_audit.md`
- **Public surface hygiene spec**: `design_docs/planned/v1_0_0/m-v1-public-surface-hygiene.md` (M4, M5, M6, §8.1)
- **Program north star & v1 mission**: `design_docs/PROGRAM.md`, `design_docs/v1-mission.md`
- **Related GitHub issues**: `#506`, `#2323`, `#2399`, `#2404`, `#2410`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)
