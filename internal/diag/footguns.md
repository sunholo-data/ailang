# Footgun Coverage Table

**Owner:** `m-diagnostic-coverage` (R1.1 of the Fable strategy review)
**CI enforcement:** `internal/diag/footgun_fixtures_test.go` — every `covered` /
`shipped-this-sprint` row has a fixture asserting the diagnostic **code** and that
the message/suggestion carries the **fix substring**.

## What this is

AILANG's edge for AI code synthesis is **error-time teaching**: when a model hits a
footgun, the diagnostic should state the rule and carry the concrete fix, so a single
self-repair round can act on it — instead of paying a per-run prompt tax to warn about
every footgun on every call. This table inventories footguns from the teaching prompt's
**Common Mistakes** (`prompts/v0.16.2.md`) + **`docs/LIMITATIONS.md`**, maps each to its
current and target diagnostic, and tracks whether the diagnostic is a tested contract yet.

## Status legend

| Status | Meaning |
|--------|---------|
| `covered` | Fix-carrying diagnostic already shipped + CI-fixtured. `++`-on-strings is the gold standard. |
| `shipped-this-sprint` | Diagnostic added/extended by this milestone (#325, #327-interim) + CI-fixtured. |
| `inventoried` | Footgun catalogued; a fix-carrying diagnostic may already exist but is not yet a CI contract, or the target diagnostic is future work. |

## Verification

Every "current diagnostic" below was **live-verified** by running the sprint binary
(`ailang check`) on a temp `.ail` file reproducing the footgun (2026-07-09,
`v0.28.0-33-g` + this branch). Diagnostics are quoted verbatim (module-path MOD010
temp-dir warnings elided). The 4 rows promoted to `covered` on 2026-07-10
(reserved keyword, hyphen-in-module, `;`-in-expr-func, stdlib-import hint) and the
`%`/Fractional non-reachability were re-verified live that day via the fixture
pipeline (`internal/diag/footgun_fixtures_test.go`, driving `pipeline.Run` in
`ModeCheck`).

## Coverage table

| footgun | trigger snippet | current diagnostic (VERBATIM, live-verified) | target diagnostic | fixture | prompt lines to delete | status |
|---------|-----------------|----------------------------------------------|-------------------|---------|------------------------|--------|
| `++` on strings | `"a" ++ "b"` | ``type error: ++ operator at ...: `++` is for lists only. For strings use "${expr}" interpolation, concat([parts]), or join(sep, parts).`` | (met — gold standard) | `footgun_fixtures_test.go:plusplus_strings` | `prompts/v0.16.2.md:199`; String and List Concatenation section (≈1193–1210) | covered |
| import after a declaration (#325) | `type Op = Add`<br>`import std/list (map)` | `PAR_IMPORT_PLACEMENT at ...: imports must appear immediately after the module declaration` + suggestion `move this import above the first type/func declaration` | (met this sprint) | `footgun_fixtures_test.go:import_placement`, `import_placement_rule_stated` | (none in prompt today — pre-empts a future line) | shipped-this-sprint |
| duplicate `module` declaration in one file (M-PROMPT-FOOTGUNS) | `module benchmark/a`<br>...<br>`module benchmark/b` | `MOD002 at ...: duplicate module declaration 'benchmark/b' — AILANG requires exactly one module declaration per file (first module 'benchmark/a' declared at ...)` + suggestions `keep the single module declaration at the top ...` / `to model multiple namespaces, split into separate .ail files ...` | (met — wires the dormant MOD002; kills the PAR_NO_PREFIX_PARSE cascade) | `footgun_fixtures_test.go:duplicate_module` | (none in prompt — pre-empts a future line) | shipped-this-sprint |
| misplaced (non-first) `module` declaration (M-PROMPT-FOOTGUNS) | `import std/list (map)`<br>`module test/late` | `PAR_MODULE_PLACEMENT at ...: the module declaration must be the first declaration in the file` + suggestion `move 'module test/late' above the other declarations at the top of the file` | (met — new PAR_MODULE_PLACEMENT; distinct from duplicate) | `footgun_fixtures_test.go:misplaced_module` | (none in prompt) | shipped-this-sprint |
| module-local func not resolvable in some position (#323→#327→#366 family) | e.g. `let sub4 = \y. subtract(4, y)` where `subtract` is a module func | `undefined variable: subtract at ... (subtract is defined in this module but not resolvable in this position — please report as #366; workaround: declare subtract as a \`func\`)` — residual net; the known members (record-update fields #327, module-let/letrec decl class #366) are FIXED, so this now fires only for a not-yet-discovered position | truth-telling residual (cites LIVE #366, drops closed #327, gives the VERIFIED `func` workaround) | `internal/types/local_resolution_hint_test.go` | (none — this is a bug, not a prompt line) | retired-trigger/residual-net |
| duplicate module-scope binding (let/func same name) (#366, MOD007) | `let helper = 5`<br>`export func helper() -> int = 10` | `Error MOD007: 'helper' is declared as both a module-level let (at ...) and a func (at ...) — a module-scope name may have only one binding.` + `Fix: rename one of them, or fold the let into the func body` | (met — fix-carrying, both positions) | `footgun_fixtures_test.go:TestFootgunFixture_MOD007_DuplicateModuleBinding` (module pipeline, needs a real filename, per MOD014 precedent) | (none in prompt) | shipped-this-sprint |
| `println(42)` / `print(42)` (needs string) | `println(42)` | `type error: ... No instance for Num[string] in scope. Arithmetic operators (+, -, *, /) need numbers, but this is a string. Use ++ to concatenate strings, or stringToInt to convert a string to a number.` | should name the print-arg-must-be-string rule + suggest `show(42)`; current message is about arithmetic, misdirecting | (future) | `prompts/v0.16.2.md:2338` | inventoried |
| `import "std/io"` (quoted path) | `import "std/io"` | `IMP012_UNSUPPORTED_NAMESPACE at ...: namespace imports not yet supported` + suggestion `Use selective import: import module/path (symbol1, symbol2) / Or module alias: import std/list as List` | should detect the quoted-path shape and suggest dropping the quotes: `import std/io (println)` | (future) | `prompts/v0.16.2.md:190`, `:2339` | inventoried |
| `%` / division on wrong numeric type | `3.5 % 2.0` **type-checks clean** (floats satisfy the operator — no diagnostic fires); the nearest mismatch `func f(a:int,b:int)->float{a/b}` yields a plain `cannot unify int vs float`, not the Fractional hint | the `No instance for Fractional[int] ... intToFloat` message (`internal/types/instances.go`) is **NOT reachable via `ailang check` on an .ail snippet** — it only fires through the internal `InstanceEnv.Lookup` path exercised by `instances_test.go` (live-verified 2026-07-10). Kept for the record; not fixturable | carries a fix (intToFloat) once reachable; would also need to name the actual operator (`%` vs `/`) | (not fixturable — see current-diagnostic cell) | `prompts/v0.16.2.md:1094` (defaulting note) | inventoried |
| reserved keyword as name | `let exists = 1` | `PAR_RESERVED_KEYWORD at ...: expected identifier, got reserved keyword 'exists'` + suggestions incl. `'exists' is reserved for existential types (future feature)` and `Try: let found = ... or let doesExist = ...` | (met — fix-carrying) | `footgun_fixtures_test.go:reserved_keyword` | `prompts/v0.16.2.md:2346`, Reserved Keywords Common-mistake block (≈2259–2266) | covered |
| bare assignment (missing `let`) | `x = 1` | `PAR015 at ...: bare assignment not supported (missing 'let' keyword)` + suggestion `Use: let x = ... in` (preceded by a `PAR_UNEXPECTED_TOKEN` on `=`) | collapse the leading `PAR_UNEXPECTED_TOKEN` so only the fix-carrying PAR015 shows | (future) | `prompts/v0.16.2.md:188` | inventoried |
| `const` keyword (JS/TS) | `const x = 1` | `PAR020 at ...: missing ';' between block statements (found 'x' ...)` (block context masks the const-specific PAR014 at top level) | recognise `const` in block-statement position and emit the PAR014 const→let fix | (future) | `prompts/v0.16.2.md:188` | inventoried |
| hyphen in module/import path | `module benchmark/my-solution` | `PAR_HYPHEN_IN_MODULE at ...: hyphens in module paths are parsed as subtraction (in 'benchmark/my')` + suggestion `Use underscores instead: module benchmark/my_solution` | (met — fix-carrying) | `footgun_fixtures_test.go:hyphen_in_module` | (none in prompt) | covered |
| `\|` as bitwise OR | `1 \| 2` | `PAR016 at ...: '\|' is reserved for pattern alternatives in 'match', not a binary operator` + De Morgan / `\|\|` suggestions (preceded by `PAR_UNEXPECTED_TOKEN`) | collapse the leading `PAR_UNEXPECTED_TOKEN` so only PAR016 shows | (future) | (none in prompt) | inventoried |
| `;` in expression-body func | `func f() -> int = let x = 1; x` | `PAR017 at ...: ';' is only valid inside '{ ... }' block bodies, not in expression-body functions` + `let .. in ..` vs `{ }` suggestions | (met — fix-carrying) | `footgun_fixtures_test.go:semicolon_in_expr_func` | `prompts/v0.16.2.md:2349` | covered |
| stdlib func used without import | `map(\x. x, [1,2,3])` (no `import std/list`) | ``undefined variable: map at ... — `map` is exported by std/list, std/option, std/result; add the matching import`` | (met — fix-carrying via ImportSuggester) | `footgun_fixtures_test.go:stdlib_import_hint` (needs `AILANG_STDLIB_PATH` + `types.ImportSuggester` wired in TestMain — the CLI wires these in `init()`) | `prompts/v0.16.2.md:2347` (transitive-imports line) | covered |
| `list.map(f)` (method syntax) | `list.map(f)` | (field-access / type error — method-call syntax does not exist) | detect `x.method(args)` where `method` is a known stdlib function and suggest `method(x, args)` | (future) | `prompts/v0.16.2.md:2340` | inventoried |
| unknown effect mode value (M-EFFECT-MODE-VALIDATION) | `Rand[mode=banana]` | `EFF_UNKNOWN_MODE at ...: effect 'Rand' has no mode 'banana'. Allowed values: crypto, os, seeded.` + `Fix: use one of the allowed values, or drop the parameter for the default.` | (met — fix-carrying) | `footgun_fixtures_test.go:effect_unknown_mode` | (none in prompt) | shipped-this-sprint |
| unknown effect param key (M-EFFECT-MODE-VALIDATION) | `Rand[flavor=hot]` | `EFF_UNKNOWN_PARAM_KEY at ...: effect 'Rand' has no parameter 'flavor'. Allowed keys: mode.` + `Fix: use one of the allowed keys, or drop the parameter for the default.` | (met — fix-carrying) | `footgun_fixtures_test.go:effect_unknown_param_key` | (none in prompt) | shipped-this-sprint |
| param on schema-less effect (M-EFFECT-MODE-VALIDATION) | `Clock[mode=pinned]` | `EFF_PARAMS_NOT_SUPPORTED at ...: effect 'Clock' does not support parameters (found: mode). Only Rand and AI accept parameters in v1.0.0; Clock/Net/FS modes are tracked in m-effect-clock-net-fs-modes.` + `Fix: drop the parameter and use the bare effect 'Clock'.` | (met — fix-carrying) | `footgun_fixtures_test.go:effect_params_not_supported` | (none in prompt) | shipped-this-sprint |
| module-less file with top-level funcs (M-MODLESS-FAIL-LOUD) | `export func main() -> () ! {IO} { ... }` with **no** `module` line | `Error MOD014: no 'module' declaration — this file has top-level declarations but no module, so nothing is exported and the entry never runs.` + `Fix: add 'module <canonical/path>' as the first line of the file` | (met — fix-carrying) | `footgun_fixtures_test.go:TestFootgunFixture_MOD014_ModuleLess` (needs a real filename — MOD014 fires only in the module pipeline, so it is a standalone test, not a `footgunFixtures` inline-Code row; the `BareExpressionPreserved` sibling guards the `1 + 1` eval escape hatch) | (none in prompt) | shipped-this-sprint |

**Count:** 21 rows (≥ 10 required).

- **Fixtured footgun rows: 12** — `++` (covered), import-after-decl (shipped-this-sprint, contributes
  **2** fixtures: `import_placement` + `import_placement_rule_stated`), the 4 promoted by
  M-DIAG-FIXTURE-PROMOTION (reserved keyword, hyphen, `;`-in-expr-func, stdlib-import hint), the
  **3** added by M-EFFECT-MODE-VALIDATION (unknown-mode, unknown-param-key, params-not-supported),
  and the **2** added by M-PROMPT-FOOTGUNS (`duplicate_module` MOD002, `misplaced_module`
  PAR_MODULE_PLACEMENT). So the fixture *entry* count in `footgunFixtures` is 12, mapping to **11
  distinct footgun rows** in this table (import-after-decl has two fixtures).
- **CI-fixtured footgun rows in this table: 11** (1 `covered` `++` + 1 `shipped-this-sprint`
  import-after-decl + 4 promoted-this-sprint `covered` + 3 `shipped-this-sprint` effect-mode + 2
  `shipped-this-sprint` module-placement). The
  #327-interim row is `shipped-this-sprint` but its contract lives in
  `internal/types/local_resolution_hint_test.go`, not `footgunFixtures`.
- **Promote-to-`covered` candidates remaining: 1** (down from 5) — only `%`/Fractional, and it is
  **blocked**: its claimed diagnostic is not reachable via `ailang check` on an .ail snippet (see its
  current-diagnostic cell), so it cannot be fixtured without new diagnostic work (out of scope).

> **Status update (#366, M-MODULE-LET-FUNC-RESOLUTION):** the truth-telling hint
> (`internal/types/import_hint.go:43`, `localResolutionHint`) is KEPT as a residual safety net but
> **no longer cites the closed #327 or the no-op "bind it with let first" workaround**. Both known
> members of the family are fixed — expression positions by #327 (`m-record-update-local-resolution`)
> and the module-let/letrec DECL class by #366 (this work: unified SCC over lets+funcs, `wrapInLets`
> deleted). The clause now cites the LIVE #366 and the VERIFIED workaround "declare it as a `func`".
> Full deletion is deferred: it would need a proof that NO syntactic position can still mis-resolve a
> module func, and the residual net is cheap insurance against the next family member.

## Deletion gate (do NOT skip)

Per the strategy review R1 safety gate: a teaching-prompt line is deleted **only after**
its replacement diagnostic ships **and** the rig A/B shows no pass-rate loss. This milestone
ships the diagnostics + fixtures; the prompt-line deletion pass + A/B are a **separate,
gated step** and are intentionally NOT done here.

## KPI

- Prompt lines deletable once the shipped diagnostics clear A/B: the `++` section (≈18 lines)
  and the import-placement pre-emption — reported at the next baseline as "prompt tokens
  deleted per release".
