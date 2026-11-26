# M-AI-GO-TO-LANGUAGE: Making AILANG the Default AI Go-To Language (v0.63.0)

**Status**: Implemented (Slices 1–5 — 2026-11-26)
**Target**: v0.63.0
**Priority**: P0 (High — closes the top structural boilerplate, error-handling, self-healing, context-orienting, and scripting gaps between AILANG and Python/TypeScript/Rust for AI agents)
**Estimated**: 5 independent author slices + 1 read-only verifier slice, designed for parallel `/swarm` execution with isolated `workspace:"share"` worktrees (~1–2 days wall-clock)
**Dependencies**:
- `design_docs/planned/v0_62_0/m-ai-preferred-language.md` (shipped in `v0.62.0`: typed holes `_?`, `check --fix` Phase 1, `std/map` & `std/list` Phase 1)
- `design_docs/planned/v0_63_0/m-ai-preferred-v2.md` (shipped on `origin/dev` `94026beac`: unbraced multi-statement `match`/`if` arms, shared `internal/quickfix/` + LSP `codeAction`, `TC_MATCH_002` stubs, deterministic `estimate_task_elo.sh`)
- `design_docs/planned/v1_1_0/m-error-propagation.md` (Quorum history informs Slice 2's `let`-bound `?` scope)
**Author**: Antigravity (`@planner`, `/swarm` planning session for Mark)
**Created**: 2026-11-26

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All 5 workstreams (`deriving (Json)`, `let`-bound `?` AST desugaring, `check --fix` Phase 2, `ailang inspect`, and stdlib scripting functions) are 100% deterministic with canonical output ordering |
| A2: Replayability | +1 | `let`-bound `?` and `deriving (Json)` preserve source spans (`OrigSpan` / `Pos`) onto generated Core nodes so execution traces map back to user declarations |
| A3: Effect Legibility | +2 | `check --fix` Phase 2 stamps missing `! {Effects}` explicitly into source signatures (never inferring hidden effects); `ailang inspect` surfaces resolved `! {Effects}` per symbol in 1 line; `std/fs` line helpers carry explicit `! {FS}` |
| A4: Explicit Authority | +1 | `std/fs` (`readLines`, `writeLines`, `readLinesResult`, `writeLinesResult`) delegate through capability-gated `_fs_*` primitives respecting `--caps FS` and `AILANG_FS_SANDBOX` |
| A5: Bounded Verification | +1 | `ailang inspect` surfaces `requires`/`ensures` contracts alongside types/effects; `deriving (Json)` generates total `decode<Type> -> Result[<Type>, string]` validators |
| A6: Safe Concurrency | 0 | No concurrency model changes |
| A7: Machines First | +2 | Eliminates ~60–80% of JSON codec and `Result`/`Option` pyramid boilerplate in agent-generated code, heals the top 2 remaining compiler-fixable errors (`! {Effects}` and `string ++`) in 1 pass, and gives agents a `<100ms` 1-line-per-symbol semantic X-ray (`ailang inspect`) |
| A8: Minimal Syntax | +1 | Zero new keywords; reuses existing `deriving (...)` clause (`Json` alongside `Eq`) and restricts `?` strictly to `let x = expr?` (avoiding general postfix grammar complexity) |
| A9: Cost Visibility | +1 | `std/string` and `std/fs` scripting functions compile cleanly to the bytecode VM (`0` evaluator fallbacks under `--strict-bytecode`) with explicit complexity bounds |
| A10: Composability | +2 | Derived `encode<Type>`/`decode<Type>` compose recursively across nested records, ADTs, `[T]`, and `Option[T]`; `readLinesResult` composes directly with `let lines = readLinesResult(path)?` |
| A11: Structured Failure | +2 | `decode<Type>` returns typed `Result[<Type>, string]` with field/variant path context; `let x = expr?` makes `Result`/`Option` error propagation flat instead of deeply nested; `readLinesResult`/`writeLinesResult` return `Result` instead of panicking |
| A12: System Boundary | +1 | `deriving (Json)` turns untrusted JSON boundary payloads into validated nominal AILANG types via `decode<Type>` without manual partial extractor chains |

**Net Score: +15** ✅ **Proceed to implementation** (≥ +2 threshold; zero −1 scores)

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (record fields sorted/emitted in declaration order; `inspect` symbols emitted in deterministic declaration order)
- [x] A3 (Effects): No hidden side effects (`check --fix` writes explicit `! {Effects}` into source code; compiler still rejects unannotated effects without `--fix`)
- [x] A4 (Authority): No ambient access granted (`std/fs` line helpers require `FS` capability)
- [x] A7 (Machines First): Every slice directly targets token efficiency, 1-shot compile convergence, and `<100ms` agent orientation

---

## Problem Statement

After `v0.62.0` (`M-AI-PREFERRED-LANGUAGE`) and `M-AI-PREFERRED-V2` (`94026beac`), five high-leverage gaps still separate AILANG from being an AI agent's **unconditional go-to language** for data pipelines, CLI tools, multi-module apps, and typed API boundaries:

1. **Manual JSON codec boilerplate (`40–90 LOC` per type)**: Writing `encodeUser` / `decodeUser` by hand using `jo([kv(...)])`, `getString`, `getInt`, and nested `match` blocks costs 40–90 tokens-heavy lines per record/ADT and is a major source of off-by-one field-key bugs. Meanwhile `deriving (Eq)` (`internal/parser/parser_type.go:409`, `internal/ast/ast_decl.go:220`) already parses `deriving (...)` clauses, rejecting `Json` with `PAR_DERIVING_UNSUPPORTED: unsupported deriving: Json, only 'Eq' is currently supported`.
2. **Nested `match` pyramids on `Result` and `Option`**: Chaining 4 fallible operations (`readFileResult`, `json.decode`, `decodeConfig`, `validate`) requires 16+ lines of right-drifting nested `match` arms. Full Rust-style sub-expression `?` (`design_docs/planned/v1_1_0/m-error-propagation.md`) was parked in Quorum because arbitrary sub-expression hoisting (`f(g(), h()?)`) required unverified ANF-hoisting premises (`gpt5-6-sol`) and a compiler↔stdlib `ConstructorID` lang-items registry (`gemini-3-1-pro`). However, **95%+ of real agent error propagation is `let x = expr?` on the function spine**, which desugars purely at the surface-AST level with **zero** ANF hoisting and **zero** Core IR / typechecker changes.
3. **Top remaining mechanically-fixable check failures (`missing ! {Effects}` & `string ++ string`)**:
   - Models frequently omit `! {IO}` or `! {FS}` on helper functions or write `s1 ++ s2` for strings (Haskell/Elm habit).
   - Live check confirms `validate_effects.go` already emits the exact missing effects and suggested signature (`Effect checking failed for function 'greet' ... Missing effects: FS ... Suggested fix: func greet(...) -> T ! {FS, IO}`), and `typechecker_operators.go:112` already emits `++ operator at file:line:col: '++' is for lists only` — yet `internal/quickfix/` (`ailang check --fix` + LSP `codeAction`) does not yet auto-apply either fix.
4. **Agent orientation across unfamiliar `.ail` files/directories**: `ailang iface --compact` shows *exported* types and signatures for a single module, but AI agents editing a file or package need a single `<100ms` command (`ailang inspect <file-or-dir>`) that shows **both exports and local helpers**, their inferred/declared types, resolved `! {Effects}`, `requires`/`ensures` contracts, and **direct callees** (call graph) in dense 1-line-per-symbol agent format or JSON without reading full function bodies.
5. **Missing everyday text/file scripting batteries**: Agents writing scripts and parsers repeatedly hand-roll `lines`, `unlines`, `padLeft`, `padRight`, `stripPrefix`, `stripSuffix`, `indexOf` (returning `Option[int]` instead of sentinel `-1` from `find`), and `readLines`/`writeLines`/`readLinesResult`/`writeLinesResult`.

---

## Goals

**Primary Goal:** Ship 5 orthogonal, strictly file-disjoint ergonomics and tooling upgrades in `v0.63.0` that cut JSON + `Result`/`Option` boilerplate by >70%, expand `ailang check --fix` + LSP quick-fixes to heal missing effect rows and `string ++`, add `<100ms` `ailang inspect` semantic X-ray, and complete `std/string` + `std/fs` scripting batteries with 0 bytecode VM fallbacks.

**Success Metrics:**
- **Boilerplate reduction**: A 5-field record + 3-variant ADT JSON round-trip drops from ~75 LOC to 2 lines (`deriving (Json)`), and a 4-step `Result` pipeline drops from 18 LOC of nested `match` to 5 flat lines (`let x = step()?`).
- **Self-healing rate**: `ailang check --fix` and LSP `textDocument/codeAction` deterministically repair missing `! {Effects}` (including removing contradictory `pure`) and `s1 ++ s2` on `string` operands in ≤2 passes.
- **Inspection latency & token density**: `ailang inspect <file.ail> --format=agent` completes in `<100ms` on single stdlib/example modules and emits 1 line per symbol including type, `! {Effects}`, contracts, and direct callees.
- **VM parity**: 100% of derived JSON functions, `let`-bound `?` desugarings, and new `std/string` / `std/fs` functions execute with **0 evaluator fallbacks** under `--strict-bytecode`.

---

## High-Impact Decisions (Ratify Before `/swarm`)

| ID | Decision | Options | Recommendation | Chosen By | Deadline | Change Cost |
|----|----------|---------|----------------|-----------|----------|-------------|
| **D1** | `deriving (Json)` generated function names, `Option[T]` record-field wire format, and sum-ADT representation | **(a)** Generate pure `encode<Type>(v: <Type>) -> Json` and `decode<Type>(j: Json) -> Result[<Type>, string]` (exported iff `type <Type>` is `export`). Record `Option[T]` fields: `None` omits the key on encode; `decode<Type>` accepts **both** missing key and `JNull` as `Ok(None)` (malformed non-null value returns `Err`). Sum ADTs encode as `{ "tag": "<Ctor>", "fields": [...] }` (or `{ "tag": "<Ctor>" }` when nullary; named constructor fields encode as object keys `{ "tag": "<Ctor>", "<f>": ... }`). **(b)** Emit `JNull` for `None` record fields on encode instead of omitting the key, and always use `"fields": [...]` array even for named ADT fields. | **(a)** Idiomatic JSON payloads (optional keys omitted when `None`, accepted when missing or `null` on decode) + predictable `encode<Type>`/`decode<Type>` names matching AILANG's monomorphic no-method convention | Mark (human) | design | high |
| **D2** | Scope of `?` error/option propagation in `v0.63.0` | **(a) `let`-bound `?` only**: Allow `let x = expr?` (and `let x: T = expr?`) in `{ ... }` blocks and `let x = expr? in body` on the return spine of functions/func-lits whose declared return type is `Result[_, E]` (`Ok`/`Err`) or `Option[_]` (`Some`/`None`). Emit `PAR_TRY_LET_ONLY` with hoist hint (`let x = g()?; f(x)`) if `?` appears in call args / general sub-expressions, and `TC_TRY_NON_SPINE` if used inside a non-tail `let` RHS block. **(b)** Full sub-expression postfix `expr?` (`f(a()?, b()?)`) via whole-body ANF hoisting (`m-error-propagation.md`). | **(a)** Captures >95% of real `Result`/`Option` chains via pure surface-AST desugaring, completely avoiding both Quorum blockers in `m-error-propagation.md` and staying forward-compatible with **(b)** in `v1.1+` | Mark (human) | design | high |
| **D3** | `ailang check --fix` & LSP Phase 2 scope (`! {Effects}` auto-stamping + `string ++` rewrite) | **(a)** Auto-stamp missing `! {Effects}` onto `func` signatures using the effect checker's exact inferred set (merging into existing `! {...}`, or inserting `! {...}` after `-> RetType` and stripping `pure ` if present), **AND** auto-rewrite binary `s1 ++ s2` on `string` operands to `"${s1}${s2}"` (inlining adjacent string literals, e.g. `"hi " ++ name` → `"hi ${name}"`). **(b)** Expose `! {Effects}` stamping and `string ++` only as LSP `codeAction` suggestions, not applied by CLI `ailang check --fix`. | **(a)** Both CLI `--fix` and LSP `codeAction` share `internal/quickfix/`; `--fix` is an explicit opt-in flag (`ailang check` without `--fix` still rejects undeclared effects, preserving A3) | Mark (human) | design | med |
| **D4** | `ailang inspect` output schema & CLI command-table group | **(a)** Register `inspect` in `languageCommands()` (`Language: true`, `Group: groupDev` — so **both** `ailang inspect <path>` and `ailang dev inspect <path>` work identically with zero startup DB/git probes AND `commands_top_level` stays at `17` in `make simplicity-audit`). Default `--format=agent` (1 line/symbol: `[export ]func name(params) -> Ret ! {Effects} [requires {...}] [ensures {...}] [calls: f, g]`) + `--json` structured schema (`module`, `file`, `types`, `symbols` with `name`, `kind`, `exported`, `line`, `type`, `effects`, `contracts`, `calls`). **(b)** Same schemas, but file `inspect` at top-level `Group: ""` (`commands_top_level` 17 → 18, updating `wantVisibleTopLevel` and simplicity snapshot). | **(a)** `ailang inspect` works directly at top level while keeping the 17-command top-level help screen and `make simplicity-audit` snapshot untouched | Mark (human) | design | med |

### Design Freeze

Before `/swarm` author workers launch, these must be ratified by Mark:

- [x] **D1** ratified (`deriving (Json)` naming `encode<Type>`/`decode<Type>` & `Option` / ADT wire semantics)
- [x] **D2** ratified (`let`-bound `?` scope on `Result[_, E]` and `Option[_]` functions + `PAR_TRY_LET_ONLY`)
- [x] **D3** ratified (`check --fix` + LSP auto-stamping `! {Effects}` and rewriting `string ++` to `"${...}"`)
- [x] **D4** ratified (`ailang inspect` agent + JSON output schema & CLI group placement)

---

## Verification Log (Premises Checked Against Live `94026beac`)

Every positive, negative, error-code, and mechanism claim in this document was verified against the live codebase (`94026beac`) before finalizing:

| ID | Claim | How Verified | Result |
|----|-------|--------------|--------|
| **V1** | `deriving (Json)` is currently rejected at parse time with `PAR_DERIVING_UNSUPPORTED`; `deriving (Eq)` is parsed in `internal/parser/parser_type.go:409-468` and formatted generically via `dk.String()` in `internal/format/decl.go:415-421`. | Live `ailang check --json` (`PAR_DERIVING_UNSUPPORTED at line 2:40`) + read `parser_type.go` & `format/decl.go` | **Confirmed** (`format/decl.go` needs zero edits once `DeriveJson.String() == "Json"`) |
| **V2** | `let x = r?` is currently a parse error (`PAR_UNEXPECTED_TOKEN`), while `_?` (anonymous hole, `parser_literals.go:17`) and `?name` (named hole, `parser.go:116`) already use `lexer.QUESTION` (`token.go:93`) in prefix/underscore position only; `QUESTION` has **no** entry in `infixParseFns`. | Live `ailang check --json` + read `parser.go:90-145` & `parser_literals.go:14-38` | **Confirmed** |
| **V3** | `Result` (`Ok`, `Err`) and `Option` (`Some`, `None`) are prelude-bound types/constructors available in every module without explicit `import std/result` or `import std/option`. | Live `ailang check --relax-modules --json` on `result_prelude.ail` & `option_prelude.ail` | **Confirmed** (`passed: true` on both with zero imports) |
| **V4** | `Json` is **not** a builtin type name (`json_without_import` treats bare `Json` as an opaque local type per `type_scope.go:78`), whereas `std/json` defines `export type Json = JNull \| JBool(bool) \| JNumber(float) \| JString(string) \| JArray([Json]) \| JObject([{ key: string, value: Json }]) \| JInt(int)`. | Live `ailang check` + read `std/json.ail:14` & `type_scope.go:70-97` | **Confirmed** (`deriving (Json)` must resolve `std/json`'s `Json` type & constructors or require/inject `std/json` import; see Slice 1 design) |
| **V5** | Missing function effect annotations already produce a deterministic, structured message from `validate_effects.go`: `Effect checking failed for function '<name>'\n ... Missing effects: <list>\n ... Suggested fix: func <name>(...) -> T ! {<merged>}`. | Live `ailang check --json` on `missing_effect_io.ail`, `missing_effect_pure.ail`, `missing_effect_partial.ail` | **Confirmed** (`internal/effects/` requires zero edits) |
| **V6** | String `++` already produces a deterministic message from `internal/types/typechecker_operators.go:112`: `++ operator at <file>:<line>:<col>: '++' is for lists only. For strings use "${expr}" interpolation...`. | Live `ailang check --json` on `str_concat.ail` + read `typechecker_operators.go:104-116` | **Confirmed** |
| **V7** | Proposed diagnostic codes `PAR_TRY_LET_ONLY`, `TC_TRY_RETURN_TYPE`, `TC_TRY_NON_SPINE`, and `TC_DERIVE_JSON_POLY` are unallocated across `internal/` and `cmd/`. | `grep -rn` across `internal/` and `cmd/` | **Confirmed** (0 hits for all four codes) |
| **V8** | Statement-form `let x = v` inside `{ ... }` blocks is formatted by `statementLet` in `internal/format/format.go:549` (separate from `letInInline`/`letChainMultiline` in `internal/format/expr.go:309-381`) and normalized in `internal/elaborate/expr_control.go:337` (separate from `expr_simple.go`). | `grep -rn "\*ast\.Let\b" internal/` + read `format.go:536-560` & `expr_control.go:83-357` | **Confirmed** (both companion files explicitly assigned to Slice 2) |
| **V9** | Every language CLI command added to `languageCommands()` in `cmd/ailang/commands_language.go` is checked against `s5AddedLanguageRoutes` in `cmd/ailang/commands_table_test.go:25`, and `Group: groupDev` keeps `commands_top_level = 17` while routing both `ailang inspect` and `ailang dev inspect`. | Read `commands_table_test.go:14-58`, `commands_groups_test.go:15-39,185-248`, `audit.sh:72-78` | **Confirmed** |
| **V10** | `internal/quickfix/quickfix.go` is currently `719` lines (near the 800-line CI gate); `internal/parser/parser_expr.go` is `633` lines; `internal/ast/ast_expr.go` is `613` lines; `internal/format/expr.go` is `608` lines. | `estimate_task_elo.sh` `large_file_near_800l` check | **Confirmed** (companion files named per slice to stay well under 800 lines) |

---

## Solution Design: Five Orthogonal Workstreams

### Slice 1 (`derive-json`) — `deriving (Json)` for Records & ADTs

#### Overview & Semantics (per D1)
Adding `deriving (Json)` (or `deriving (Eq, Json)`) to a monomorphic record or algebraic sum `type` declaration synthesizes two pure functions in the same module (with `IsExport = decl.Exported`):

1. `encode<Type>(v: <Type>) -> Json` (pure, `! {}`)
2. `decode<Type>(j: Json) -> Result[<Type>, string]` (pure, `! {}`)

**Supported field/payload types (recursive):**
- `int`: encodes via `JInt(n)`; decodes `JInt(n) => Ok(n)` and integral `JNumber(f)` within `int64` `=> Ok(floatToInt(f))`, else `Err("<Type>.<field>: expected int")`.
- `float`: encodes via `JNumber(x)`; decodes `JNumber(x) => Ok(x)` and `JInt(i) => Ok(intToFloat(i))`, else `Err(...)`.
- `string`: encodes via `JString(s)`; decodes `JString(s) => Ok(s)`, else `Err(...)`.
- `bool`: encodes via `JBool(b)`; decodes `JBool(b) => Ok(b)`, else `Err(...)`.
- `[T]`: encodes via `JArray(...)` mapping element encoder; decodes `JArray(xs)` failing fast with index context `Err("<Type>.<field>[i]: ...")` on first `Err`.
- `Option[T]`:
  - **As a record field** `f: Option[T]`: on encode (`D1=a`), `None` omits `"f"` from the `JObject` key-value list and `Some(x)` includes `{ key: "f", value: encodeT(x) }`; on decode, **both** missing key `"f"` and `"f": null` (`JNull`) yield `Ok(None)`, while `"f": <non-null>` decodes `x` via `T` (`Ok(Some(x))` or `Err(...)` if ill-typed).
  - **In non-field position** (`[Option[T]]` or ADT positional payload `Wrap(Option[int])`): `None` ↔ `JNull`, `Some(x)` ↔ `encodeT(x)`.
- **Nested user types** `<Other>`: delegates to `encode<Other>` / `decode<Other>` in scope (supporting mutually-recursive and nested derived types via `ElaborateFile`'s existing Tarjan SCC graph!).
- **Sum ADTs** (`type Shape = Point | Circle(float) | Rect({ w: float, h: float })`):
  - Nullary constructor `Point` ↔ `JObject([{ key: "tag", value: JString("Point") }])`.
  - Positional constructor `Circle(r)` ↔ `JObject([{ key: "tag", value: JString("Circle") }, { key: "fields", value: JArray([encodeFloat(r)]) }])`.
  - Named-field constructor ` Named(x: int, y: string)` ↔ `JObject([{ key: "tag", value: JString("Named") }, { key: "x", value: JInt(x) }, { key: "y", value: JString(y) }])`.
- **Polymorphic types** (`type Box[a] = ... deriving (Json)`): rejected cleanly with `TC_DERIVE_JSON_POLY` (`cannot derive Json for polymorphic type Box[a]; instantiate a concrete type or write encode/decode functions explicitly`), matching `deriving (Eq)` (`file_funcs.go:141`).

#### Architecture & Pipeline Integration
- **AST (`internal/ast/ast_decl.go`)**: Add `DeriveJson` to `DeriveKind` (`String()` returns `"Json"`). `internal/format/decl.go:415` already formats `d.Deriving` via `dk.String()`, so `ailang fmt` works automatically.
- **Parser (`internal/parser/parser_type.go`)**: Extend `parseDeriveName()` (`line 461`) with `case "Json": return ast.DeriveJson` and update the `PAR_DERIVING_UNSUPPORTED` hint to `"Use 'Eq' or 'Json'"`.
- **Elaboration (`internal/elaborate/derive_json.go` NEW, `file_funcs.go`, `file.go`)**:
  - In `ElaborateFile` (`internal/elaborate/file.go`), after type declarations are registered and before `collectFuncSigs(file)`, run `e.synthesizeDerivedJsonDecls(file)`.
  - Ensure `std/json` constructors (`JNull`, `JBool`, `JNumber`, `JString`, `JArray`, `JObject`, `JInt`) and `Json` type resolve hygienically to `std/json` (requiring `import std/json (Json)` or binding `std/json`'s `Json` + `$adt.make_std_json_Json_*` constructors when `std/json` is imported; if `Json` is not imported from `std/json`, emit a clear actionable diagnostic `IMP_DERIVE_JSON: deriving (Json) requires 'import std/json (Json)'` — which `ailang check --fix` can even auto-import!).
  - Synthesized `encode<Type>` and `decode<Type>` `*ast.FuncDecl`s are appended to `file.Funcs` (and `file.Decls`) so SCC analysis, Core typechecking, `iface` generation, and bytecode compilation treat them identically to hand-written functions — **0 new opcodes or VM changes**.

---

### Slice 2 (`try-let`) — `let`-Bound `?` Propagation for `Result` & `Option`

#### Overview & Why `let`-Bound `?` Avoids Both Quorum Blockers (per D2)
`design_docs/planned/v1_1_0/m-error-propagation.md` identified two load-bearing blockers for arbitrary sub-expression `expr?`:
1. **`gpt5-6-sol` (ANF hoisting premise)**: Floating `?` out of `f(g(), h()?)` or `let x = { let y = foo()? in y + 1 } in Ok(x)` relied on unverified `normalizeToAtomic` / `core.go:309` let-flattening invariants and a `TryOrigin` side-table in `CoreTypeChecker`.
2. **`gemini-3-1-pro` (Compiler↔stdlib linkage)**: Emitting Core `Match` directly required `ConstructorID` lookup for user-space `Result`.

**How Slice 2 eliminates both premises by construction:**
- **Syntax restricted to `let` bindings**:
  - Block form: `{ let x = expr?; rest }` or newline-separated `{ let x = expr?\n rest }` (and unbraced `match`/`if` bare-let chains from `M-AI-PREFERRED-V2`!).
  - Expression form: `let x = expr? in body` (and optional type annotation `let x: T = expr? in body`).
  - Destructuring `let (a, b) = expr?` (optional nice-to-have if irrefutable; at minimum identifier `let x = expr?`).
- **Helpful parse guard (`PAR_TRY_LET_ONLY`)**: `QUESTION` is **not** added to `infixParseFns`. Instead:
  - `parseLetExpression` (`internal/parser/parser_expr.go` + helper `internal/parser/parser_try_let.go`) checks `p.peekTokenIs(lexer.QUESTION)` right after parsing `let.Value`, setting `let.IsTry = true` and `let.TryPos = p.curPos()`.
  - Because `parseIdentifier` (`parser_literals.go:17`) already consumes `_?` into `*ast.Hole` and `parseHoleExpression` consumes `?name` into `*ast.Hole`, `let x = _?` and `let x = ?h` continue to parse as typed holes (`let.IsTry == false`)!
  - If `p.peekTokenIs(lexer.QUESTION)` appears after a non-`let` expression in call arguments (`f(g()?)`), binary operands, or block/return expressions (`returnExpr?`), the parser emits `PAR_TRY_LET_ONLY` with hint: `"AILANG allows '?' only on let bindings: rewrite as 'let v = expr?; ...'"`.
- **Return-boundary & spine check + Surface-AST desugaring (`internal/elaborate/try_let.go`)**:
  - Enclosing function (`FuncDecl` via `openSignature` in `rigid_sig.go` or `FuncLit` in `expr_simple.go`) provides its declared `ReturnType`:
    - `Result[_, E]` → unwraps `Ok(x) => <continuation>`, early-returns `Err($try_err) => Err($try_err)`.
    - `Option[_]` → unwraps `Some(x) => <continuation>`, early-returns `None => None`.
    - Missing return annotation or non-`Result`/`Option` return type → emits `TC_TRY_RETURN_TYPE` pointing at `let.TryPos` (`"cannot use '?' in function returning <Ret>; enclosing function must declare return type Result[T, E] or Option[T]"`).
  - **Spine-only invariant (`TC_TRY_NON_SPINE`)**: Within a function body, `let x = expr?` is valid on the **tail spine** of the function body (top-level `let ... in`, statements of a tail `Block`, or branches of a tail `if` / `match` whose arms are themselves on the return spine!). If `let x = expr?` is written inside a **non-tail** position (e.g., `let outer = { let inner = foo()?; inner + 1 } in Ok(outer)` or inside an `if` condition), elaboration rejects it with `TC_TRY_NON_SPINE` and concrete rewrite hint (`"hoist 'let inner = foo()?' to the outer block spine"`), guaranteeing `Ok(Err(e))` can **never** occur without touching `core.go:309`!
  - Because `Ok`/`Err` and `Some`/`None` are verified prelude constructors (**V3**), desugaring produces an ordinary `*ast.Match` before Core normalization — zero changes to `internal/core/`, `internal/types/`, `internal/eval/`, or `internal/vm/`!
- **Formatter round-trip (`internal/format/expr.go` & `internal/format/format.go`)**:
  - `letInInline`, `letChainMultiline`, and `statementLet` emit `?` immediately after `let.Value` when `let.IsTry` is true (wrapping lower-precedence `let.Value` such as `if`/`match`/`lambda` in parens `(if c then a else b)?` so re-parsing binds the entire RHS before `?`).

---

### Slice 3 (`check-fix-types`) — `ailang check --fix` & LSP Quick-Fixes Phase 2

#### Overview (per D3)
Extends the shared `internal/quickfix/` engine (used by `ailang check --fix`, `ailang check --json`, and `internal/lsp/` `textDocument/codeAction`) with two deterministic, compiler-verified Phase 2 repairs:

1. **Auto-stamp missing `! {Effects}` onto `func` signatures (`internal/quickfix/effect_stamp.go` NEW)**:
   - Matches `validate_effects.go`'s structured diagnostic (`Effect checking failed for function '<name>'` + `Suggested fix: func <name>(...) -> T ! {<Effects>}`).
   - Locates `func <name>` declaration header in `src` (handling `export`, `pure`, `[typeParams]`, multi-line parameter lists, return type `-> RetType`, existing `! {Existing}`, `requires`/`ensures` clauses, and `{` or `=` body opener using balanced delimiter scanning — never naive regex on nested `{}` record return types!).
   - **Case A (existing `! { ... }` annotation)**: Replaces the existing effect set inside `! { ... }` with the sorted union from `Suggested fix` (preserving any `@limit=N` budget annotations already present on existing effects!).
   - **Case B (no `! { ... }` annotation)**: Inserts ` ! {<Effects>}` immediately after the return type (or after `)` if return type is omitted), and if the declaration carries `pure `, removes `pure ` in the same non-overlapping edit set so `pure func f() ! {IO}` is never produced.
2. **Auto-rewrite `s1 ++ s2` on `string` operands to `"${s1}${s2}"` (`internal/quickfix/string_concat.go` NEW)**:
   - Matches `typechecker_operators.go:112` (`++ operator at <file>:<line>:<col>: \`++\` is for lists only`).
   - Also updates `cmd/ailang/check_output.go` and `internal/lsp/diagnostics.go` if needed so embedded `at <file>:<line>:<col>` spans populate `Line`/`Column` on `checkJSONError` / `protocol.Diagnostic` (allowing LSP cursor-range `codeAction` requests on the `++` line to match!).
   - Uses balanced token/expression boundary scanning around the `++` token at `<line>:<col>` to extract `lhs` and `rhs` safely:
     - Adjacent string literal optimization: `"Hello, " ++ name` → `"Hello, ${name}"`; `a ++ "!"` → `"${a}!"`; `"a" ++ "b"` → `"ab"`.
     - General expressions: `s1 ++ s2` → `"${s1}${s2}"` (skipping edit safely if either operand contains unescaped `"` or multiline block to avoid invalid interpolation syntax, or chaining `a ++ b ++ c` across iterative `runCheckFix` passes!).
- **File-size discipline**: `internal/quickfix/quickfix.go` is `719` lines (**V10**); keep `ComputeEdits` dispatch thin (~15 LOC added) and place all Phase 2 logic in `internal/quickfix/effect_stamp.go` and `internal/quickfix/string_concat.go`.

---

### Slice 4 (`ailang-inspect`) — `<100ms` Semantic Module X-Ray for AI Agents

#### Overview & Output Schema (per D4)
Adds `ailang inspect <file.ail | dir> [--json | --format=agent | --exports-only | --relax-modules]` in `cmd/ailang/inspect.go`:

- **Why agents need `inspect` beyond `iface`**:
  - `ailang iface <module>` only shows *exported* types and function types, omits `requires`/`ensures` contracts, omits *local* helper functions (where bugs often live), omits source line numbers, and omits direct intra/inter-module callees (`calls`).
  - `ailang inspect` runs `pipeline.Run` in `ModeCheck` (no Z3, no evaluator — typically `15–45ms` for single files) and extracts a complete semantic X-ray from `Result.Artifacts.AST`, `Result.Artifacts.Core` (`Prog.Meta` for contracts + Core AST walk for direct callees), `Result.Interface`, and `Result.TypeChecker` / `CoreTI`.

- **Default / `--format=agent` output (token-lean, 1 line per declaration, deterministic source order)**:
  ```text
  module examples/runnable/string_fs_scripting (examples/runnable/string_fs_scripting.ail)
  type LineStats = { count: int, nonEmpty: int } [export, line 8]
  export func summarizeFile(path: string) -> Result[LineStats, string] ! {FS} requires { length(path) > 0 } [line 12, calls: readLinesResult, countNonEmpty]
  func countNonEmpty(xs: [string]) -> int ! {} ensures { result >= 0 } [line 20, calls: std/list.filter, std/list.length]
  ```
- **`--json` output schema**:
  ```json
  {
    "files": [
      {
        "file": "examples/runnable/string_fs_scripting.ail",
        "module": "examples/runnable/string_fs_scripting",
        "types": [
          { "name": "LineStats", "kind": "record", "exported": true, "line": 8, "definition": "{ count: int, nonEmpty: int }", "deriving": [] }
        ],
        "symbols": [
          {
            "name": "summarizeFile",
            "kind": "func",
            "exported": true,
            "pure": false,
            "line": 12,
            "signature": "(string) -> Result[LineStats, string] ! {FS}",
            "effects": ["FS"],
            "contracts": [{ "kind": "requires", "expr": "(length(path) > 0)" }],
            "calls": ["countNonEmpty", "std/fs.readLinesResult"]
          }
        ]
      }
    ]
  }
  ```
- **Directory support**: If `<dir>` is passed, walks `.ail` files in sorted deterministic order and emits all modules in one response.
- **CLI Registration (`D4=a`)**: Added to `languageCommands()` in `cmd/ailang/commands_language.go` with `Name: "inspect", Group: groupDev, Language: true` and listed in `s5AddedLanguageRoutes` in `cmd/ailang/commands_table_test.go` (so `ailang inspect` runs zero observatory/git startup probes and `commands_top_level` remains `17`).

---

### Slice 5 (`stdlib-scripting`) — `std/string` & `std/fs` Scripting Batteries

#### Overview & Exact Signatures
Pure AILANG compositions over existing Go-backed iterative builtins (`_str_split`, `_str_join`, `_str_len`, `_str_slice`, `_str_startsWith`, `_str_endsWith`, `_str_find`, `_str_repeat`, `_fs_readFile`, `_fs_writeFile`, `_fs_readFileResult`, `_fs_writeFileResult`) — **0 new Go builtins required, 0 VM fallbacks under `--strict-bytecode`**:

1. **`std/string.ail` (all `export pure func`, `! {}`)**:
   - `indexOf(s: string, sub: string) -> Option[int]` — total `Option`-returning wrapper around `find(s, sub)` (`Some(i)` when `i >= 0`, `None` when `< 0`), eliminating sentinel `-1` bugs and composing directly with `let idx = indexOf(s, ":")?`!
   - `stripPrefix(s: string, prefix: string) -> string` — removes `prefix` once if `startsWith(s, prefix)` (slicing `substring(s, length(prefix), length(s))`), else returns `s` unchanged.
   - `stripSuffix(s: string, suffix: string) -> string` — removes `suffix` once if `endsWith(s, suffix)` (slicing `substring(s, 0, length(s) - length(suffix))`), else returns `s` unchanged.
   - `padLeft(s: string, width: int, pad: string) -> string` — pads `s` on the left to at least `width` rune code points using `pad` (if `length(s) >= width` or `length(pad) == 0`, returns `s` unchanged; uses `repeat` + `substring` so multi-char or single-char `pad` is exact $O(n)$ single allocation, never recursive `++`).
   - `padRight(s: string, width: int, pad: string) -> string` — mirror of `padLeft`, padding on the right to `width` runes.
   - `lines(s: string) -> [string]` — splits text into lines handling both `\n` and `\r\n` cleanly (`""` → `[]`; `"a\nb\n"` drops a single trailing newline just like Haskell/Python `splitlines()`, `"a\r\nb"` → `["a", "b"]` via `stripSuffix(line, "\r")`).
   - `unlines(xs: [string]) -> string` — joins lines with `"\n"` and trailing newline (`[]` → `""`, `["a", "b"]` → `"a\nb\n"`).

2. **`std/fs.ail` (all `export func ... ! {FS}`)**:
   - `readLines(path: string) -> [string] ! {FS}` — `lines(readFile(path))` (imports `lines`, `join` from `std/string`).
   - `writeLines(path: string, xs: [string]) -> () ! {FS}` — `writeFile(path, unlines(xs))` (imports `unlines` from `std/string`).
   - `readLinesResult(path: string) -> Result[[string], string] ! {FS}` — `match readFileResult(path) { Ok(c) => Ok(lines(c)), Err(e) => Err(e) }`.
   - `writeLinesResult(path: string, xs: [string]) -> Result[(), string] ! {FS}` — `writeFileResult(path, unlines(xs))`.
   - *(Import cycle check)*: `std/fs.ail` **already** imports `std/string (compare, endsWith)` at line 10; `std/string.ail` imports only `std/option`. Extending `std/fs`'s `import std/string (...)` with `lines, unlines` introduces **zero** new module edges!

---

## Conflict Surface (Mandatory Compiler & Grammar Audit)

### Syntactic & Semantic Positions Touched
1. **Type declaration suffix (`internal/parser/parser_type.go:454`)**: `deriving (<Ident>, ...)` after `type Name = <Body>`.
2. **`let` value suffix (`internal/parser/parser_expr.go:154`)**: `let <ident>[: <Type>] = <expr>?` (optional trailing `QUESTION` token after `let` RHS expression).
3. **Diagnostic-to-edit engine (`internal/quickfix/`)**: Rewriting function headers for missing `! {Effects}` and infix `++` expressions on strings.

### What Else Lives in Each Position & Disambiguation Strategy

| Position | Existing Construct | New Construct | Exact Disambiguation |
|----------|--------------------|---------------|----------------------|
| `deriving (...)` identifier | `deriving (Eq)` (`DeriveEq`) | `deriving (Json)`, `deriving (Eq, Json)` | Exact identifier switch in `parseDeriveName()` (`"Eq"` vs `"Json"`). `deriving` is already a reserved keyword (`lexer.DERIVING`). |
| `let x = _?` | Anonymous typed hole (`*ast.Hole{Name: ""}`) | `let`-bound try (`let.IsTry = true`) | `parseIdentifier()` (`parser_literals.go:17`) consumes `_` + `QUESTION` **inside** `parseExpression(LOWEST)` and returns `*ast.Hole`. When `parseExpression` returns to `parseLetExpression`, `peekToken` is no longer `QUESTION`, so `let x = _?` remains an anonymous typed hole! |
| `let x = ?foo` | Named typed hole (`*ast.Hole{Name: "foo"}`) | `let`-bound try | `parseHoleExpression()` (`parser_literals.go:28`) consumes `QUESTION` + `IDENT` as a prefix expression and returns `*ast.Hole`. Zero overlap. |
| `let x = f()? in body` vs `let x = f()?` in `{ ... }` | `let` expression (`in`) and block statement `let` (`;` or newline) | `let.IsTry = true` before `in`, `;`, or newline | `parseLetExpression` checks `if p.peekTokenIs(lexer.QUESTION)` **after** `let.Value = p.parseExpression(LOWEST)` and **before** `if p.peekTokenIs(lexer.IN)`. Works identically for `in`, `;`, and R2 newline-separated block statements (`parser_branch.go` checks `let.Body == nil`, unaffected by `let.IsTry`). |
| `f(g()?)` / `a? + b` | Parse error (`PAR_UNEXPECTED_TOKEN`) | `PAR_TRY_LET_ONLY` | `QUESTION` is **not** registered in `infixParseFns`. Call-arg / expression parsers check `p.peekTokenIs(lexer.QUESTION)` to emit actionable `PAR_TRY_LET_ONLY` instead of generic delimiter cascade errors. |
| `func f(r: {a: int}) -> {b: int}` under `check --fix` | Record type `{b: int}` in return annotation | Inserting ` ! {IO}` before function body `{` or `=` | `effect_stamp.go` scans balanced `(...)`, `[...]`, `{...}` in parameter and return type annotations after `->` so record return types `{b: int}` are never confused with the opening `{` of the function body. |

### Pipeline Pass Coverage
- [x] `prog.Decls` — top-level declarations (`derive_json` & `try_let` both elaborate into standard `prog.Decls` Core nodes)
- [x] `prog.Meta[*].Contracts[*].Expr` — contract expressions (`requires`/`ensures` forbid `?` via non-Result boolean predicate / non-spine check; `ailang inspect` reads `prog.Meta[*].Contracts`)
- [x] Integration test: `TestContractExpressionsFullyLowered` continues to pass unchanged

### Programs That MUST Still Work (Verified Existing Fixtures)
1. `std/json.ail` — uses `export type Json = ... deriving (Eq)` (verifies `deriving (Eq)` and existing `Json` ADT stay 100% intact).
2. `examples/runnable/records.ail` — record declarations, nested field access, and record updates.
3. `examples/runnable/adt_option.ail` & `examples/runnable/adt_tree.ail` — recursive and parameterized ADTs (`Option`, `Tree`) without `deriving (Json)`.
4. `examples/tests/syntax_ergonomics_v3_test.ail` — unbraced multi-statement `match` arms and `if`/`else` bare-`let` chains (`M-AI-PREFERRED-V2`), plus typed holes `_?` in `TC_MATCH_002` tests.
5. `examples/tests/stdlib_combinators_test.ail`, `std/string.ail`, `std/fs.ail` — existing string/fs/collection stdlib functions under `--strict-bytecode`.

### What Deliberately Changes
- `type T = ... deriving (Json)` changes from `PAR_DERIVING_UNSUPPORTED` to generating `encode<T>` and `decode<T>`. (Note: if a module manually declares both `deriving (Json)` on `T` **and** a hand-written `func encode<T>`, existing `MOD007` / duplicate-func check reports the name collision explicitly.)
- `let x = expr?` changes from `PAR_UNEXPECTED_TOKEN` to `Result`/`Option` early-return desugaring; `f(g()?)` changes from `PAR_UNEXPECTED_TOKEN` to targeted `PAR_TRY_LET_ONLY`.

---

## Examples (Before & After)

### Example 1: Typed JSON Boundary + Flat Error Propagation (Slices 1 + 2 + 5)

**Before (`~42 LOC` of manual extractors & 3-deep nested `match`):**
```ailang
module app/config
import std/fs (readFileResult)
import std/json (Json, decode, getString, getInt, jo, kv, js, jint)

export type AppConfig = { host: string, port: int, tag: Option[string] }

export pure func decodeAppConfig(j: Json) -> Result[AppConfig, string] =
  match getString(j, "host") {
    None => Err("missing host"),
    Some(h) => match getInt(j, "port") {
      None => Err("missing port"),
      Some(p) => Ok({ host: h, port: p, tag: getString(j, "tag") })
    }
  }

export func loadHostPort(path: string) -> Result[string, string] ! {FS} =
  match readFileResult(path) {
    Err(e) => Err(e),
    Ok(raw) => match decode(raw) {
      Err(e) => Err(e),
      Ok(j) => match decodeAppConfig(j) {
        Err(e) => Err(e),
        Ok(cfg) => Ok("${cfg.host}:${show(cfg.port)}")
      }
    }
  }
```

**After (`11 LOC`, flat, total, 0 boilerplate):**
```ailang
module app/config
import std/fs (readFileResult)
import std/json (Json, decode)

export type AppConfig = { host: string, port: int, tag: Option[string] } deriving (Json)

export func loadHostPort(path: string) -> Result[string, string] ! {FS} {
  let raw = readFileResult(path)?
  let j = decode(raw)?
  let cfg = decodeAppConfig(j)?
  Ok("${cfg.host}:${show(cfg.port)}")
}
```

### Example 2: Self-Healing `ailang check --fix` Phase 2 (Slice 3)

**Agent draft (2 classic dialect slips: missing `! {IO}` + `++` on strings):**
```ailang
module app/greet
export pure func greet(name: string) -> () {
  println("Hello, " ++ name)
}
```

**After `ailang check --fix app/greet.ail` (1 pass, 0 LLM repair turns):**
```ailang
module app/greet
export func greet(name: string) -> () ! {IO} {
  println("Hello, ${name}")
}
```

---

## Testing Strategy

Every behavioural requirement carries a `TST-*` ID referenced directly by the sprint plan milestones:

| ID | Slice | Test Location | What It Verifies |
|----|-------|---------------|------------------|
| `TST-DJ-01` | 1 (`derive-json`) | `internal/parser/derive_json_test.go` | `deriving (Json)` and `deriving (Eq, Json)` parse on records & ADTs; `ailang fmt` round-trips cleanly |
| `TST-DJ-02` | 1 (`derive-json`) | `examples/tests/derive_json_test.ail` | Record `encode<T>`/`decode<T>` round-trip across `int`, `float`, `string`, `bool`, `[T]`, nested derived records, and `Option[T]` (missing key + `null` both decode to `Ok(None)`; wrong type returns `Err`) under `--strict-bytecode` |
| `TST-DJ-03` | 1 (`derive-json`) | `examples/tests/derive_json_test.ail` + `internal/elaborate/derive_json_test.go` | Sum ADT (nullary, positional, named fields, recursive list-of-ADT) round-trips under `--strict-bytecode`; polymorphic `type Box[a] = ... deriving (Json)` fails with `TC_DERIVE_JSON_POLY` |
| `TST-TL-01` | 2 (`try-let`) | `internal/parser/try_let_test.go` + `internal/format/try_let_test.go` | `let x = expr?` in `in`-chains, `{ ... }` `;`/newline blocks, and unbraced match/if chains parses & round-trips in `ailang fmt`; `let x = _?` and `let x = ?h` still parse as `*ast.Hole`; `f(g()?)` emits `PAR_TRY_LET_ONLY` |
| `TST-TL-02` | 2 (`try-let`) | `examples/tests/try_let_test.ail` | Multi-step `Result[T, E]` (`Ok` pass-through + `Err` short-circuit + single evaluation of effectful RHS) and `Option[T]` (`Some`/`None` short-circuit) pass on evaluator and `--strict-bytecode` |
| `TST-TL-03` | 2 (`try-let`) | `internal/elaborate/try_let_test.go` | `?` in function returning `int` fails with `TC_TRY_RETURN_TYPE`; `?` inside non-spine `let` RHS block (`let x = { let y = r?; y + 1 } in Ok(x)`) fails with `TC_TRY_NON_SPINE` (never wraps `Ok(Err(_))`) |
| `TST-QF-01` | 3 (`check-fix-types`) | `internal/quickfix/quickfix_test.go` + `cmd/ailang/check_fix_test.go` | Missing `! {IO}` / `! {FS, IO}` auto-stamps onto unannotated, partially-annotated (preserving `@limit=N`), record-returning, and `pure func` headers; `ailang check --fix` converges to exit 0 |
| `TST-QF-02` | 3 (`check-fix-types`) | `internal/quickfix/quickfix_test.go` + `internal/lsp/code_action_test.go` | `s1 ++ s2` and `"Hello " ++ name` on `string` operands produce identical `TextEdit`s (`"${s1}${s2}"` / `"Hello ${name}"`) in `ailang check --json`, `check --fix`, and LSP `textDocument/codeAction` |
| `TST-IN-01` | 4 (`ailang-inspect`) | `cmd/ailang/inspect_test.go` | `ailang inspect <file> --format=agent` and `--json` emit exact types, `! {Effects}`, `requires`/`ensures` contracts, and direct `calls` for both exported and local functions; directory mode walks deterministically |
| `TST-IN-02` | 4 (`ailang-inspect`) | `cmd/ailang/inspect_test.go` + `commands_table_test.go` | `inspect` executes in `<100ms` on `std/string.ail`, triggers 0 observatory/git startup probes (`TestProbes_LanguageCommandsRunNone`), and keeps `TestGroups_VisibleTopLevelIsTheSpecifiedSet` green |
| `TST-SS-01` | 5 (`stdlib-scripting`) | `examples/tests/stdlib_scripting_test.ail` | `std/string` (`lines`, `unlines`, `padLeft`, `padRight`, `stripPrefix`, `stripSuffix`, `indexOf`) edge cases (empty string, Unicode runes, `\r\n`, width ≤ length, missing prefix/suffix) pass on evaluator and `--strict-bytecode` |
| `TST-SS-02` | 5 (`stdlib-scripting`) | `examples/tests/stdlib_scripting_test.ail` + `examples/runnable/string_fs_scripting.ail` | `std/fs` (`readLines`, `writeLines`, `readLinesResult`, `writeLinesResult`) round-trip files and return `Err` on missing paths without panicking, with 0 VM fallbacks under `--strict-bytecode` |

---

## Success Criteria

- [ ] `TST-DJ-01..03` passing (`deriving (Json)` on records + ADTs, evaluator + `--strict-bytecode`)
- [ ] `TST-TL-01..03` passing (`let`-bound `?` on `Result` + `Option`, `ailang fmt` round-trip, `PAR_TRY_LET_ONLY`, evaluator + `--strict-bytecode`)
- [ ] `TST-QF-01..02` passing (`check --fix` + LSP `codeAction` for `! {Effects}` and `string ++`)
- [ ] `TST-IN-01..02` passing (`ailang inspect` `<100ms` agent + JSON output, probe & command-table gates green)
- [ ] `TST-SS-01..02` passing (`std/string` + `std/fs` scripting additions, 0 VM fallbacks)
- [ ] All 3 new runnable showcase modules (`derive_json.ail`, `error_propagation.ail`, `string_fs_scripting.ail`) pass `--strict-bytecode`
- [ ] `make check-file-sizes` (all files ≤ 800 lines) and `make check-boundaries` pass
- [ ] Post-merge Coordinator updates `.stdlib-golden/` (`make freeze-stdlib && make verify-stdlib`), `prompts/`, `llms.txt`, `examples/manifest.json`, and `changelogs/unreleased/`

---

## Deferred Decisions (Agent Latitude)

The following implementation details are intentionally left open to each slice's worker:

- Internal helper function names and AST construction helpers inside `internal/elaborate/derive_json.go` and `internal/elaborate/try_let.go` — **agent may choose**.
- Exact synthetic variable prefix (`$try_val_N` / `$dj_N`) as long as it uses `$` or `_`-prefixed hygienic names that cannot collide with user bindings — **agent may choose**.
- Exact tokenization/scanner helper structure inside `internal/quickfix/effect_stamp.go` and `internal/quickfix/string_concat.go` — **agent may choose**.
- Exact spacing/punctuation of `ailang inspect --format=agent` metadata brackets as long as each symbol fits on 1 deterministic line containing signature, `! {Effects}`, contracts, line number, and direct callees — **agent may choose** (subject to D4).

---

## Non-Goals (Hard Scope Fence for `/swarm`)

- **No general postfix sub-expression `expr?`** (`f(a()?, b()?)`, `a? + b?`) — deferred to `v1.1+` (`m-error-propagation.md`).
- **No polymorphic `deriving (Json)`** (`type Box[a] = ... deriving (Json)`) — requires dictionary-passed `Json` typeclass instances; rejected cleanly with `TC_DERIVE_JSON_POLY` in `v0.63.0`.
- **No `From`/`Into` automatic error-type coercion in `?`** — operand error type `E` must unify directly with enclosing `Result[_, E]`.
- **No changes to `internal/core/`, `internal/types/typechecker_core.go`, `internal/eval/`, `internal/vm/`, `internal/bytecode/`, or `internal/smt/`**.
- **No `make freeze-stdlib`, `prompts/`, `llms.txt`, `examples/manifest.json`, or `changelogs/` edits inside worker shares** — Coordinator owns all shared hotspots post-merge.
- **No `git push`, tag, or release** without Mark's explicit approval.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `internal/quickfix/quickfix.go` (719L) or `internal/parser/parser_expr.go` (633L) exceeds 800-line CI gate | High | Mandatory companion files: `internal/quickfix/effect_stamp.go` + `string_concat.go` (Slice 3) and `internal/parser/parser_try_let.go` + `internal/elaborate/try_let.go` (Slice 2) |
| Two slices touch `internal/elaborate/` (`derive-json` and `try-let`) and cause merge conflicts | High | Strictly partitioned by file: Slice 1 owns `file.go`, `file_funcs.go`, `derive_json.go`; Slice 2 owns `expr_simple.go`, `expr_control.go`, `rigid_sig.go`, `try_let.go` — **zero shared files** |
| `let x = _?` typed hole broken by `let`-bound `?` | High | Disambiguated by parser execution order (**V2**): `parseIdentifier` consumes `_?` as `*ast.Hole` before `parseLetExpression` inspects `peekToken`; pinned by `TST-TL-01` |
| `pure func f() -> ()` with `println` gets `! {IO}` stamped while keeping `pure` keyword | Medium | `effect_stamp.go` explicitly strips `pure ` when stamping a non-empty effect row; pinned by `TST-QF-01` |
| `deriving (Json)` used in a module that didn't import `std/json` | Medium | `derive_json.go` checks module import / type scope and emits actionable `IMP_DERIVE_JSON` (`add 'import std/json (Json)'`) rather than an opaque type mismatch |

---

## Swarm Execution Contract

> Model assignments and deterministic Task ELO breakdowns live in [`sprint-plan-m-ai-go-to-language.md`](sprint-plan-m-ai-go-to-language.md).

### Strictly Disjoint File Ownership Matrix

| Worker | Workspace | Owns (Write — Pairwise Disjoint) | Read-Only | Do Not Touch |
|--------|-----------|----------------------------------|-----------|--------------|
| `derive-json` (Slice 1) | `share` | `internal/ast/ast_decl.go`, `internal/parser/parser_type.go`, `internal/parser/derive_json_test.go` (NEW), `internal/elaborate/file.go`, `internal/elaborate/file_funcs.go`, `internal/elaborate/derive_json.go` (NEW), `internal/elaborate/derive_json_test.go` (NEW), `examples/tests/derive_json_test.ail` (NEW), `examples/runnable/derive_json.ail` (NEW) | `std/json.ail`, `internal/format/decl.go`, `internal/core/**` | `internal/ast/ast_expr.go`, `internal/parser/parser_expr.go`, `internal/elaborate/expr_*.go`, `internal/elaborate/rigid_sig.go`, `std/**`, `cmd/**`, `.stdlib-golden/**` |
| `try-let` (Slice 2) | `share` | `internal/ast/ast_expr.go`, `internal/parser/parser_expr.go`, `internal/parser/parser_branch.go`, `internal/parser/parser_try_let.go` (NEW), `internal/parser/try_let_test.go` (NEW), `internal/format/expr.go`, `internal/format/format.go`, `internal/format/try_let_test.go` (NEW), `internal/elaborate/expr_simple.go`, `internal/elaborate/expr_control.go`, `internal/elaborate/rigid_sig.go`, `internal/elaborate/try_let.go` (NEW), `internal/elaborate/try_let_test.go` (NEW), `examples/tests/try_let_test.ail` (NEW), `examples/runnable/error_propagation.ail` (NEW) | `internal/parser/parser_literals.go`, `internal/elaborate/expressions.go`, `internal/elaborate/core.go` | `internal/ast/ast_decl.go`, `internal/parser/parser_type.go`, `internal/elaborate/file*.go`, `std/**`, `cmd/**`, `.stdlib-golden/**` |
| `check-fix-types` (Slice 3) | `share` | `internal/quickfix/**` (`quickfix.go`, `effect_stamp.go` NEW, `string_concat.go` NEW, `quickfix_test.go`), `internal/lsp/**` (`code_action*.go`, `diagnostics*.go`), `cmd/ailang/check_fix.go`, `cmd/ailang/check_fix_test.go`, `cmd/ailang/check_output.go` | `internal/pipeline/validate_effects.go`, `internal/types/typechecker_operators.go`, `internal/effects/**` | `cmd/ailang/commands_*.go`, `cmd/ailang/inspect*.go`, `internal/parser/**`, `internal/elaborate/**`, `std/**`, `.stdlib-golden/**` |
| `ailang-inspect` (Slice 4) | `share` | `cmd/ailang/inspect.go` (NEW), `cmd/ailang/inspect_test.go` (NEW), `cmd/ailang/commands_language.go`, `cmd/ailang/commands_table_test.go` (+ `cmd/ailang/commands_groups_test.go` iff D4=b) | `internal/pipeline/**`, `internal/iface/**`, `internal/core/**`, `cmd/ailang/check.go` | `cmd/ailang/check_fix*.go`, `cmd/ailang/check_output.go`, `internal/**`, `std/**`, `.stdlib-golden/**` |
| `stdlib-scripting` (Slice 5) | `share` | `std/string.ail`, `std/fs.ail`, `examples/tests/stdlib_scripting_test.ail` (NEW), `examples/runnable/string_fs_scripting.ail` (NEW) | `std/option.ail`, `std/result.ail`, `std/list.ail`, `internal/builtins/**` | `.stdlib-golden/**`, `prompts/**`, `llms.txt`, `examples/manifest.json`, `cmd/**`, `internal/**` |
| `verifier` | `inherit` | *(none — read-only)* | Entire repo | Everything (reports verification & eval metrics only) |
| **Coordinator** | main | Merges Slices 1 → 2 → 3 → 4 → 5; runs `make freeze-stdlib && make verify-stdlib`; updates `prompts/`, `llms.txt`, `examples/manifest.json`, `changelogs/unreleased/`; presents final diff for Mark's OK | Everything | Never pushes or cuts a release without explicit user approval |

### Hard Stop-Gates for Every Worker
1. **Strict ownership**: Write ONLY to files in your slice's `Owns (Write)` column. Verify with `git status --short` before committing. If an out-of-slice edit appears necessary, **STOP**, post to `@coordinator` on the board, and wait.
2. **800-line ceiling**: Every modified or created `.go` / `.ail` file MUST stay `<= 800` lines (`make check-file-sizes`).
3. **Engine parity**: Every `.ail` test and runnable example MUST pass both the tree-walking evaluator and `--strict-bytecode` with 0 VM fallbacks.
4. **No shared-hotspot writes**: Workers MUST NOT run `make freeze-stdlib`, edit `prompts/`, `llms.txt`, `examples/manifest.json`, or `changelogs/`, or run `git push`.
5. **Completion protocol**: Commit ONLY owned files on your share branch, post `@coordinator [done] <commit hash, workspace path, files changed, focused test output>`, and call `swarm_wait`.

---

## Related Documents & References

- [design_docs/PROGRAM.md](../../PROGRAM.md) — Program North Star & AILANG substrate routing rules
- [design_docs/planned/v0_62_0/m-ai-preferred-language.md](../v0_62_0/m-ai-preferred-language.md) — Phase 1 (`_?` holes, `check --fix` Phase 1, stdlib Phase 1)
- [design_docs/planned/v0_63_0/m-ai-preferred-v2.md](m-ai-preferred-v2.md) — Phase 2a (`94026beac`: unbraced match/if arms, `internal/quickfix/`, LSP `codeAction`, deterministic ELO routing)
- [design_docs/planned/v1_1_0/m-error-propagation.md](../v1_1_0/m-error-propagation.md) — Full postfix `?` Quorum analysis whose two blockers Slice 2 avoids
- [design_docs/implemented/v0_6_2/m-dx19-auto-derive-eq.md](../../implemented/v0_6_2/m-dx19-auto-derive-eq.md) & [m-eq-derive-containers.md](../../implemented/v0_42_0/m-eq-derive-containers.md) — `deriving (Eq)` precedent extended by Slice 1
- [Design Axioms](/docs/references/axioms)

---

**Document created**: 2026-11-26
**Last updated**: 2026-11-26
