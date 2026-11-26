# Sprint Plan: M-AI-GO-TO-LANGUAGE (v0.63.0)

## Placement and Human-Decision Rules

Stored in `design_docs/planned/v0_63_0/sprint-plan-m-ai-go-to-language.md` alongside [`design_docs/planned/v0_63_0/m-ai-go-to-language.md`](m-ai-go-to-language.md).

**Open Human-Decision Gate (D1–D4):**
Decisions **D1** (`deriving (Json)` naming & `Option`/ADT wire semantics), **D2** (`let`-bound `?` scope vs full sub-expression `?`), **D3** (`check --fix` auto-stamping `! {Effects}` + `string ++`), and **D4** (`ailang inspect` output schema & CLI group placement) in [`m-ai-go-to-language.md` § High-Impact Decisions](m-ai-go-to-language.md#high-impact-decisions-ratify-before-swarm) are currently **OPEN** awaiting Mark's ratification before spawning author workers.
- If any executor is invoked before Mark ratifies D1–D4: **STOP: commit current work, report `BLOCKED: D1-D4 awaiting human ratification in m-ai-go-to-language.md`, and end the run**.
- Once Mark ratifies D1–D4 in the attended session, the Coordinator records the rulings in `m-ai-go-to-language.md`'s Design Freeze checklist and launches Slices 1–5 in parallel.

---

## Summary

Execute the 5 strictly file-disjoint workstreams of `M-AI-GO-TO-LANGUAGE` (`derive-json`, `try-let`, `check-fix-types`, `ailang-inspect`, `stdlib-scripting`) in parallel `workspace:"share"` worktrees, followed by Coordinator merge + shared-hotspot freeze and a read-only `verifier` pass (`--strict-bytecode` parity + `stretch` benchmark check).

**Duration:** 1–2 days wall-clock (5 parallel author workers + Coordinator integration + Verifier)
**Dependencies:** `origin/dev` (`94026beac` — `M-AI-PREFERRED-V2` + `estimate_task_elo.sh`)
**Risk Level:** Medium (Slices 1 & 2 touch `internal/parser` and `internal/elaborate`, mitigated by strict file-level partitioning and Frontier-band model routing; Slices 3–5 are Core/Routine additive)

---

## Current Status & Velocity Analysis

### Completed Recently (on `origin/dev` `94026beac`)
- ✅ **M-AI-PREFERRED-LANGUAGE (`v0.62.0`)**: Typed holes `_?`, `ailang check --fix` Phase 1, `std/map` & `std/list` Phase 1 (~1,450 LOC).
- ✅ **M-AI-PREFERRED-V2 (`94026beac`, 2026-11-26)**: Unbraced multi-statement `match` arms & `if` branches, shared `internal/quickfix/` + LSP `codeAction` + `TC_MATCH_002` stubs, `std/map` (`update`, `mapValues`, `filter`) + `std/list` (`span`, `chunk`, `groupBy`), and deterministic `estimate_task_elo.sh` (~1,150 LOC in 1 `/swarm` session).

### Velocity & Capacity
- Recent `/swarm` throughput: ~1,100–1,600 verified LOC/session across 3–5 isolated `share` workers.
- Planned sprint scope: ~1,260 LOC implementation + ~940 LOC tests/examples = **~2,200 total LOC** across 5 independent workers (~350–550 LOC per worker, well inside a single 60-minute worker turn).

---

## Registry Reuse Audit

Checked against core language / stdlib boundaries (`.agents/skills/sprint-planner/resources/registry_reuse_gate.md`):
- **Milestone 1 (`derive-json`)**: Compiler syntax + elaboration pass (`internal/ast`, `internal/parser`, `internal/elaborate`) — Decision: `none` (language-level `deriving` construct; cannot be an external package).
- **Milestone 2 (`try-let`)**: Surface syntax + elaboration desugaring (`internal/ast`, `internal/parser`, `internal/format`, `internal/elaborate`) — Decision: `none` (compiler grammar/desugaring feature).
- **Milestone 3 (`check-fix-types`)**: Compiler/LSP quick-fix engine (`internal/quickfix`, `internal/lsp`, `cmd/ailang`) — Decision: `none` (core CLI/LSP diagnostic repair tooling).
- **Milestone 4 (`ailang-inspect`)**: Core CLI subcommand (`cmd/ailang`) over `internal/pipeline` — Decision: `none` (in-process `<100ms` compiler artifact inspector).
- **Milestone 5 (`stdlib-scripting`)**: Fundamental `std/string` and `std/fs` line/padding/prefix primitives — Decision: `none` (foundational stdlib primitives alongside existing `split`, `trim`, `readFileResult`).

---

## AILANG Syntax Gate (§5.1 Mandatory Checklist)

- **AILANG prompt version loaded**: `v0.16.20` (loaded via `ailang prompt` and verified via `ailang prompt --version-active`).
- **Showcase / runnable demo modules checklist**:

| Showcase Module | Milestone | Contracts (`requires`/`ensures`) | Effects (`! {...}`) | Inline Tests (`tests [...]` / `test "..."`) |
|-----------------|-----------|----------------------------------|---------------------|---------------------------------------------|
| `examples/runnable/derive_json.ail` | M1 (`derive-json`) | `include`: `pure func validPort(p: int) -> bool ensures { result == (p >= 1 && p <= 65535) }` on decoded config validator | `include`: `export func main() -> () ! {IO}` printing deterministic round-trip JSON | `include`: companion `examples/tests/derive_json_test.ail` `test "..."` blocks (`TST-DJ-02`, `TST-DJ-03`) |
| `examples/runnable/error_propagation.ail` | M2 (`try-let`) | `include`: `pure func parsePositive(s: string) -> Result[int, string]` helper with `ensures` on positive bound check | `include`: `export func main() -> () ! {IO}` running `Result` and `Option` `?`-chains | `include`: companion `examples/tests/try_let_test.ail` `test "..."` blocks (`TST-TL-02`) |
| `examples/runnable/string_fs_scripting.ail` | M5 (`stdlib-scripting`) | `include`: `pure func formatRow(rank: int, label: string) -> string requires { rank >= 0 } ensures { length(result) >= 6 }` | `include`: `export func main() -> () ! {IO, FS}` exercising `writeLinesResult` + `readLinesResult` | `include`: companion `examples/tests/stdlib_scripting_test.ail` `test "..."` blocks (`TST-SS-01`, `TST-SS-02`) |

---

## Proposed Milestones

### Milestone 1 — Slice 1 (`derive-json`): `deriving (Json)` for Records & ADTs
**Goal:** Parse `deriving (Json)` on monomorphic record and sum ADT `type` declarations and synthesize pure `encode<Type>(v: <Type>) -> Json` and `decode<Type>(j: Json) -> Result[<Type>, string]` functions in `internal/elaborate/derive_json.go` per ratified **D1**.
**Estimated:** ~340 LOC implementation + ~220 LOC tests/examples = **~560 LOC**
**Owned Paths:** `internal/ast/ast_decl.go`, `internal/parser/parser_type.go`, `internal/parser/derive_json_test.go` (NEW), `internal/elaborate/file.go`, `internal/elaborate/file_funcs.go`, `internal/elaborate/derive_json.go` (NEW), `internal/elaborate/derive_json_test.go` (NEW), `examples/tests/derive_json_test.ail` (NEW), `examples/runnable/derive_json.ail` (NEW)

**Tasks:**
1. Add `DeriveJson` to `ast.DeriveKind` (`internal/ast/ast_decl.go`) and `"Json"` case in `parseDeriveName` (`internal/parser/parser_type.go`); verify `ailang fmt` round-trip (`TST-DJ-01`).
2. Implement `synthesizeDerivedJsonDecls` in `internal/elaborate/derive_json.go` and hook into `ElaborateFile` (`internal/elaborate/file.go`) + polymorphic guard `TC_DERIVE_JSON_POLY` in `file_funcs.go`.
3. Add Go unit tests + `.ail` evaluator & `--strict-bytecode` tests (`examples/tests/derive_json_test.ail`, `examples/runnable/derive_json.ail`).

**Acceptance Criteria:**
- [ ] `deriving (Json)` and `deriving (Eq, Json)` parse and `ailang fmt` round-trips cleanly — `TST-DJ-01`
- [ ] Record `encode<Type>`/`decode<Type>` round-trips primitives, `[T]`, nested derived types, and `Option[T]` optional/null keys — `TST-DJ-02`
- [ ] Sum ADT (nullary, positional, named fields, recursive) round-trips under `--strict-bytecode` and polymorphic types fail with `TC_DERIVE_JSON_POLY` — `TST-DJ-03`
- [ ] Focused gates pass: `go test ./internal/parser/... ./internal/elaborate/... ./internal/format/...` + `./bin/ailang test --strict-bytecode examples/tests/derive_json_test.ail` + `./bin/ailang run --caps IO --entry main --strict-bytecode examples/runnable/derive_json.ail` + `make check-file-sizes`

---

### Milestone 2 — Slice 2 (`try-let`): `let`-Bound `?` Propagation for `Result` & `Option`
**Goal:** Support `let x = expr?` in `{ ... }` blocks and `let x = expr? in body` inside `Result[_, E]` and `Option[_]` functions via surface-AST desugaring, with `PAR_TRY_LET_ONLY` on nested call-arg `?` and full `ailang fmt` round-trip per ratified **D2**.
**Estimated:** ~300 LOC implementation + ~230 LOC tests/examples = **~530 LOC**
**Owned Paths:** `internal/ast/ast_expr.go`, `internal/parser/parser_expr.go`, `internal/parser/parser_branch.go`, `internal/parser/parser_try_let.go` (NEW), `internal/parser/try_let_test.go` (NEW), `internal/format/expr.go`, `internal/format/format.go`, `internal/format/try_let_test.go` (NEW), `internal/elaborate/expr_simple.go`, `internal/elaborate/expr_control.go`, `internal/elaborate/rigid_sig.go`, `internal/elaborate/try_let.go` (NEW), `internal/elaborate/try_let_test.go` (NEW), `examples/tests/try_let_test.ail` (NEW), `examples/runnable/error_propagation.ail` (NEW)

**Tasks:**
1. Add `IsTry bool` + `TryPos Pos` to `ast.Let` (`internal/ast/ast_expr.go`); parse trailing `?` on `let` bindings and emit `PAR_TRY_LET_ONLY` in `internal/parser/parser_try_let.go` (keeping `parser_expr.go <= 800` lines and preserving `_?` / `?name` typed holes).
2. Emit `?` in `internal/format/expr.go` (`letInInline`, `letChainMultiline`) and `internal/format/format.go` (`statementLet`) with precedence-safe parenthesization.
3. Desugar `let x = expr?` along the enclosing `FuncDecl`/`FuncLit` return spine (`Result` → `Ok`/`Err`, `Option` → `Some`/`None`) in `internal/elaborate/try_let.go`, rejecting non-`Result`/`Option` boundaries (`TC_TRY_RETURN_TYPE`) and non-spine RHS blocks (`TC_TRY_NON_SPINE`).
4. Add unit + `.ail` `--strict-bytecode` tests (`examples/tests/try_let_test.ail`, `examples/runnable/error_propagation.ail`).

**Acceptance Criteria:**
- [ ] `let x = expr?` parses in `in`-chains, `{ ... }` blocks, and unbraced match/if chains; `ailang fmt` round-trips (`TestPrinterRoundTrip`); `_?`/`?h` holes stay intact; `f(g()?)` emits `PAR_TRY_LET_ONLY` — `TST-TL-01`
- [ ] `Result[T, E]` and `Option[T]` multi-step propagation + single-evaluation guarantees pass on evaluator and `--strict-bytecode` — `TST-TL-02`
- [ ] `TC_TRY_RETURN_TYPE` and `TC_TRY_NON_SPINE` fire cleanly with rewrite hints — `TST-TL-03`
- [ ] Focused gates pass: `go test ./internal/parser/... ./internal/format/... ./internal/elaborate/... ./internal/testing/...` + `./bin/ailang test --strict-bytecode examples/tests/try_let_test.ail` + `./bin/ailang run --caps IO --entry main --strict-bytecode examples/runnable/error_propagation.ail` + `make check-file-sizes`

---

### Milestone 3 — Slice 3 (`check-fix-types`): `ailang check --fix` & LSP Phase 2
**Goal:** Teach `internal/quickfix/` (shared by `ailang check --fix`, `check --json`, and LSP `codeAction`) to auto-stamp missing `! {Effects}` onto function signatures and auto-rewrite `s1 ++ s2` on `string` operands to `"${s1}${s2}"` per ratified **D3**.
**Estimated:** ~240 LOC implementation + ~190 LOC tests = **~430 LOC**
**Owned Paths:** `internal/quickfix/quickfix.go`, `internal/quickfix/effect_stamp.go` (NEW), `internal/quickfix/string_concat.go` (NEW), `internal/quickfix/quickfix_test.go`, `internal/lsp/code_action.go`, `internal/lsp/code_action_test.go`, `internal/lsp/diagnostics.go`, `internal/lsp/diagnostics_test.go`, `cmd/ailang/check_fix.go`, `cmd/ailang/check_fix_test.go`, `cmd/ailang/check_output.go`

**Tasks:**
1. Implement `effect_stamp.go` in `internal/quickfix/` (balanced-delimiter scan of `func <name>` header, inserting/merging `! {Effects}`, preserving `@limit=N` budgets, and stripping `pure ` when stamping effects).
2. Implement `string_concat.go` in `internal/quickfix/` (rewriting `s1 ++ s2` on strings to `"${s1}${s2}"` with literal-interpolation simplification) and ensure embedded `at file:line:col` spans surface on `checkJSONError` / LSP `Diagnostic` so range-filtered LSP `codeAction` requests match.
3. Add unit/integration tests across `internal/quickfix`, `internal/lsp`, and `cmd/ailang` (`quickfix.go` MUST stay `<= 800` lines).

**Acceptance Criteria:**
- [ ] Missing `! {Effects}` auto-stamps accurately across unannotated, partially-annotated, record-returning, and `pure func` signatures and converges to exit 0 under `ailang check --fix` — `TST-QF-01`
- [ ] `s1 ++ s2` on strings produces identical `TextEdit`s in `ailang check --json`, `check --fix`, and LSP `textDocument/codeAction` — `TST-QF-02`
- [ ] Focused gates pass: `go test ./internal/quickfix/... ./internal/lsp/... ./cmd/ailang -run "TestComputeCheckEdits|TestRunCheckFix|TestDistill"` + `make check-boundaries` + `make check-file-sizes`

---

### Milestone 4 — Slice 4 (`ailang-inspect`): `<100ms` Semantic Module X-Ray CLI
**Goal:** Implement `ailang inspect <file-or-dir> [--json | --format=agent | --exports-only | --relax-modules]` in `cmd/ailang/inspect.go` per ratified **D4**.
**Estimated:** ~230 LOC implementation + ~160 LOC tests = **~390 LOC**
**Owned Paths:** `cmd/ailang/inspect.go` (NEW), `cmd/ailang/inspect_test.go` (NEW), `cmd/ailang/commands_language.go`, `cmd/ailang/commands_table_test.go` (+ `cmd/ailang/commands_groups_test.go` iff D4=b)

**Tasks:**
1. Implement `runInspectCommand` in `cmd/ailang/inspect.go`: run `pipeline.Run` (`ModeCheck`), collect declared types, exported + local functions, resolved signatures, `! {Effects}`, `requires`/`ensures` contracts (`Core.Meta`), and sorted direct callees (`Core` AST walk), and format as 1-line-per-symbol `--format=agent` (default) or `--json`.
2. Support directory walking (sorted `.ail` files) and `--exports-only` filter.
3. Register `inspect` in `languageCommands()` (`cmd/ailang/commands_language.go`) + `s5AddedLanguageRoutes` (`cmd/ailang/commands_table_test.go`) and verify `<100ms` runtime + 0 startup probes.

**Acceptance Criteria:**
- [ ] `ailang inspect` emits accurate types, `! {Effects}`, `requires`/`ensures` contracts, line numbers, and direct `calls` in both `--format=agent` and `--json` for single files and directories — `TST-IN-01`
- [ ] Single-module inspection completes in `<100ms` and passes `TestTable_*`, `TestGroups_*`, and `TestProbes_LanguageCommandsRunNone` — `TST-IN-02`
- [ ] Focused gates pass: `go test ./cmd/ailang -run "TestInspect|TestTable|TestGroups|TestProbes"` + `make check-boundaries` + `make check-file-sizes`

---

### Milestone 5 — Slice 5 (`stdlib-scripting`): `std/string` & `std/fs` Scripting Batteries
**Goal:** Add `lines`, `unlines`, `padLeft`, `padRight`, `stripPrefix`, `stripSuffix`, `indexOf` to `std/string.ail` and `readLines`, `writeLines`, `readLinesResult`, `writeLinesResult` to `std/fs.ail` with 0 VM fallbacks under `--strict-bytecode`.
**Estimated:** ~150 LOC stdlib + ~140 LOC tests/examples = **~290 LOC**
**Owned Paths:** `std/string.ail`, `std/fs.ail`, `examples/tests/stdlib_scripting_test.ail` (NEW), `examples/runnable/string_fs_scripting.ail` (NEW)

**Tasks:**
1. Implement `indexOf`, `stripPrefix`, `stripSuffix`, `padLeft`, `padRight`, `lines`, `unlines` in `std/string.ail` using O(n) builtins (`_str_find`, `_str_startsWith`, `_str_endsWith`, `_str_slice`, `_str_repeat`, `_str_split`, `_str_join`).
2. Implement `readLines`, `writeLines`, `readLinesResult`, `writeLinesResult` in `std/fs.ail` importing `lines, unlines` from `std/string`.
3. Write `examples/tests/stdlib_scripting_test.ail` and `examples/runnable/string_fs_scripting.ail` and verify under both evaluator and `--strict-bytecode` (do **NOT** run `make freeze-stdlib` in worker share).

**Acceptance Criteria:**
- [ ] All 7 new `std/string` functions handle empty, Unicode, `\r\n`, and boundary inputs on evaluator and `--strict-bytecode` — `TST-SS-01`
- [ ] All 4 new `std/fs` functions pass round-trip and `Err` recovery tests with 0 VM fallbacks under `--strict-bytecode` — `TST-SS-02`
- [ ] Focused gates pass: `./bin/ailang test examples/tests/stdlib_scripting_test.ail` + `./bin/ailang test --strict-bytecode examples/tests/stdlib_scripting_test.ail` + `./bin/ailang run --caps IO,FS --entry main --strict-bytecode examples/runnable/string_fs_scripting.ail` + `make check-file-sizes`

---

### Milestone 6 — Coordinator Merge, Shared Hotspots & Verifier Gate
**Goal:** Coordinator merges Slices 1 → 2 → 3 → 4 → 5 on main workspace, runs `make freeze-stdlib && make verify-stdlib`, updates `prompts/v0.16.20.md` (or bumps to `v0.16.21` + `versions.json`), `llms.txt`, `examples/manifest.json`, and `changelogs/unreleased/2026-11-26-m-ai-go-to-language.md`. `verifier` runs post-merge verification.

**Acceptance Criteria:**
- [ ] `make test-changed`, `make verify-stdlib`, `make check-file-sizes`, `make check-boundaries`, and `make check-changelog` all exit 0
- [ ] All 3 new runnable examples (`derive_json.ail`, `error_propagation.ail`, `string_fs_scripting.ail`) and all 3 new test suites pass `--strict-bytecode`

---

## Swarm Worker & Deterministic Task ELO Model Assignment

Computed deterministically on `94026beac` via `.agents/skills/sprint-planner/scripts/estimate_task_elo.sh`:

### Summary Routing Table

| Worker | Slice / Milestone | Workspace | Base ELO | Modifiers | Total ELO | Band | Assigned Model |
|--------|-------------------|-----------|---------:|----------:|----------:|------|----------------|
| `coordinator` | M6 (merge, `.stdlib-golden/`, `prompts/`, `llms.txt`, `manifest.json`, `changelogs/`) | main | — | — | **1900+** | **Frontier** | `gemini-4-hydrogen-high` (or inherit top reasoning) |
| `derive-json` | Slice 1 (M1) | `share` | 1700 (`internal/parser/`) | +50 fan-in, +75 cross-pkg (4), +25 churn | **1850** | **Frontier** | `gemini-4-hydrogen-high` |
| `try-let` | Slice 2 (M2) | `share` | 1700 (`internal/parser/`) | +50 >600L, +50 fan-in, +75 cross-pkg (5), +25 churn | **1900** | **Frontier** | `gemini-4-hydrogen-high` |
| `check-fix-types` | Slice 3 (M3) | `share` | 1500 (`internal/quickfix/`) | +50 >600L (`quickfix.go` 719L), +75 cross-pkg (3), +25 churn | **1650** | **Core** | `gemini-4-neon-medium` |
| `ailang-inspect` | Slice 4 (M4) | `share` | 1500 (`cmd/ailang/`) | +25 churn (`commands_language.go` 10 commits/30d) | **1525** | **Core** | `gemini-4-neon-medium` |
| `stdlib-scripting` | Slice 5 (M5) | `share` | 1300 (`std/`) | +0 (all 4 modifiers clear) | **1300** | **Routine** | `gemini-3.8-flash-high` |
| `verifier` | M6 read-only verification & evals | `inherit` | 1300 (`examples/`) | +50 >600L (`manifest.json`), +25 churn | **1375** | **Routine** | `gemini-3.8-flash-high` |

- **Mid-run quota fallback**: Switch any `waiting` worker in-place via `swarm(action:"model", ...)` (`gemini-4-hydrogen-high` ↔ `claude-opus-5-5-high` for Frontier; `gemini-4-neon-medium` ↔ `claude-sonnet-5-5-medium` for Core; `gemini-3.8-flash-high` ↔ `gemini-3.8-flash-medium` for Routine) without losing conversation history or deleting its workspace.

### Exact Per-Slice `estimate_task_elo.sh` Signal Breakdowns

#### 1. `derive-json` — **1850 (Frontier → `gemini-4-hydrogen-high`)**
`estimate_task_elo.sh internal/ast/ast_decl.go internal/parser/parser_type.go internal/elaborate/file.go internal/elaborate/file_funcs.go internal/elaborate/derive_json.go examples/tests/derive_json_test.ail examples/runnable/derive_json.ail`

| Signal | Delta | Status | Detail |
|--------|------:|--------|--------|
| Base location (`internal/parser/`) | 1700 | Frontier | `internal/parser/parser_type.go` |
| `large_file_near_800l` | +0 | clear | no target file > 600 lines |
| `high_fan_in_go_pkg` | +50 | triggered | `internal/ast` (27 internal importers), `internal/parser` (17 internal importers) |
| `cross_package_span` | +75 | triggered | 4 package(s): `examples`, `internal/ast`, `internal/elaborate`, `internal/parser` |
| `recent_churn_30d` | +25 | triggered | `internal/elaborate/file.go` (7 commits/30d), `internal/elaborate/file_funcs.go` (9 commits/30d) |
| **Total ELO** | **1850** | **Frontier** | `gemini-4-hydrogen-high / claude-opus-5-5-high` |

#### 2. `try-let` — **1900 (Frontier → `gemini-4-hydrogen-high`)**
`estimate_task_elo.sh internal/ast/ast_expr.go internal/parser/parser_expr.go internal/parser/parser_branch.go internal/format/expr.go internal/format/format.go internal/elaborate/expr_simple.go internal/elaborate/expr_control.go internal/elaborate/rigid_sig.go internal/elaborate/try_let.go examples/tests/try_let_test.ail examples/runnable/error_propagation.ail`

| Signal | Delta | Status | Detail |
|--------|------:|--------|--------|
| Base location (`internal/parser/`) | 1700 | Frontier | `internal/parser/parser_expr.go` |
| `large_file_near_800l` | +50 | triggered | `internal/ast/ast_expr.go` (613L), `internal/parser/parser_expr.go` (633L), `internal/format/expr.go` (608L) |
| `high_fan_in_go_pkg` | +50 | triggered | `internal/ast` (27 internal importers), `internal/parser` (17 internal importers) |
| `cross_package_span` | +75 | triggered | 5 package(s): `examples`, `internal/ast`, `internal/elaborate`, `internal/format`, `internal/parser` |
| `recent_churn_30d` | +25 | triggered | `internal/parser/parser_expr.go` (6 commits/30d), `internal/elaborate/expr_simple.go` (5 commits/30d) |
| **Total ELO** | **1900** | **Frontier** | `gemini-4-hydrogen-high / claude-opus-5-5-high` |

#### 3. `check-fix-types` — **1650 (Core → `gemini-4-neon-medium`)**
`estimate_task_elo.sh internal/quickfix internal/lsp cmd/ailang/check_fix.go cmd/ailang/check_fix_test.go cmd/ailang/check_output.go`

| Signal | Delta | Status | Detail |
|--------|------:|--------|--------|
| Base location (`internal/quickfix/`) | 1500 | Core | `internal/quickfix` |
| `large_file_near_800l` | +50 | triggered | `internal/quickfix/quickfix.go` (719L) — split new rules into `effect_stamp.go` & `string_concat.go` |
| `high_fan_in_go_pkg` | +0 | clear | max `internal/quickfix` (2 internal importers, < 10) |
| `cross_package_span` | +75 | triggered | 3 package(s): `cmd/ailang`, `internal/lsp`, `internal/quickfix` |
| `recent_churn_30d` | +25 | triggered | `cmd/ailang/check_output.go` (15 commits/30d) |
| **Total ELO** | **1650** | **Core** | `gemini-4-neon-medium / claude-sonnet-5-5-medium` |

#### 4. `ailang-inspect` — **1525 (Core → `gemini-4-neon-medium`)**
`estimate_task_elo.sh cmd/ailang/inspect.go cmd/ailang/inspect_test.go cmd/ailang/commands_language.go cmd/ailang/commands_table_test.go`

| Signal | Delta | Status | Detail |
|--------|------:|--------|--------|
| Base location (`cmd/ailang/`) | 1500 | Core | `cmd/ailang/inspect.go` |
| `large_file_near_800l` | +0 | clear | no target file > 600 lines |
| `high_fan_in_go_pkg` | +0 | clear | max `cmd/ailang` (0 internal importers, < 10) |
| `cross_package_span` | +0 | clear | 1 package(s): `cmd/ailang` |
| `recent_churn_30d` | +25 | triggered | `cmd/ailang/commands_language.go` (10 commits/30d) |
| **Total ELO** | **1525** | **Core** | `gemini-4-neon-medium / claude-sonnet-5-5-medium` |

#### 5. `stdlib-scripting` — **1300 (Routine → `gemini-3.8-flash-high`)**
`estimate_task_elo.sh std/string.ail std/fs.ail examples/tests/stdlib_scripting_test.ail examples/runnable/string_fs_scripting.ail`

| Signal | Delta | Status | Detail |
|--------|------:|--------|--------|
| Base location (`std/`) | 1300 | Routine | `std/string.ail` |
| `large_file_near_800l` | +0 | clear | no target file > 600 lines |
| `high_fan_in_go_pkg` | +0 | clear | no Go target packages |
| `cross_package_span` | +0 | clear | 2 package(s): `examples`, `std` |
| `recent_churn_30d` | +0 | clear | no target path with >= 5 commits in last 30d |
| **Total ELO** | **1300** | **Routine** | `gemini-3.8-flash-high / gemini-3.8-flash-medium` |

#### 6. `verifier` — **1375 (Routine → `gemini-3.8-flash-high`)**
`estimate_task_elo.sh examples benchmarks`

| Signal | Delta | Status | Detail |
|--------|------:|--------|--------|
| Base location (`examples/`) | 1300 | Routine | `examples` |
| `large_file_near_800l` | +50 | triggered | `examples/manifest.json` (4516L), `benchmarks/docx_reimplement.yml` (619L) |
| `high_fan_in_go_pkg` | +0 | clear | max `examples` (1 internal importers, < 10) |
| `cross_package_span` | +0 | clear | 2 package(s): `benchmarks`, `examples` |
| `recent_churn_30d` | +25 | triggered | `examples` (65 commits/30d), `benchmarks` (13 commits/30d) |
| **Total ELO** | **1375** | **Routine** | `gemini-3.8-flash-high / gemini-3.8-flash-medium` |

---

## Success Metrics
- All `TST-DJ-*`, `TST-TL-*`, `TST-QF-*`, `TST-IN-*`, and `TST-SS-*` acceptance tests passing on both evaluator and `--strict-bytecode` (0 VM fallbacks)
- `ailang inspect std/string.ail --format=agent` latency `<100ms`
- Zero files > 800 lines (`make check-file-sizes`)
- Zero architecture boundary violations (`make check-boundaries`)
- Zero startup probes on `ailang inspect` (`TestProbes_LanguageCommandsRunNone`)

## Open Questions
- **D1–D4** in [`m-ai-go-to-language.md` § High-Impact Decisions](m-ai-go-to-language.md#high-impact-decisions-ratify-before-swarm) await Mark's ratification before spawning Slices 1–5 (**STOP: commit current work, report `BLOCKED: D1-D4 awaiting human ratification in m-ai-go-to-language.md`, and end the run** if invoked unratified).
