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
| local func in record-update field (#327) | `{ s \| dls: extendFm(s.dls, ...) }` where `extendFm` is a module-local func | `undefined variable: extendFm at ... (extendFm is defined in this module but not resolvable in this position — known bug #327; workaround: bind it with let first)` | truth-telling + workaround (interim); real fix in `m-record-update-local-resolution` retires this | `internal/types/local_resolution_hint_test.go` | (none — this is a bug, not a prompt line) | shipped-this-sprint |
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

**Count:** 14 rows (≥ 10 required).

- **Fixtured footgun rows: 7** — `++` (covered), import-after-decl (shipped-this-sprint, contributes
  **2** fixtures: `import_placement` + `import_placement_rule_stated`), and the 4 promoted this sprint
  (reserved keyword, hyphen, `;`-in-expr-func, stdlib-import hint). So the fixture *row* count (7)
  equals the fixture *entry* count in `footgunFixtures` — but it maps to **6 distinct footgun rows**
  in this table because import-after-decl has two fixtures.
- **CI-fixtured footgun rows in this table: 6** (1 `covered` `++` + 1 `shipped-this-sprint`
  import-after-decl + 4 promoted-this-sprint `covered`). The #327-interim row is `shipped-this-sprint`
  but its contract lives in `internal/types/local_resolution_hint_test.go`, not `footgunFixtures`.
- **Promote-to-`covered` candidates remaining: 1** (down from 5) — only `%`/Fractional, and it is
  **blocked**: its claimed diagnostic is not reachable via `ailang check` on an .ail snippet (see its
  current-diagnostic cell), so it cannot be fixtured without new diagnostic work (out of scope).

> **Follow-up (not this sprint):** the #327 interim truth-telling hint (`internal/types/import_hint.go:43`,
> aka `localResolutionHint`) is a **dead-code retirement candidate** now that the real record-update
> resolver fix shipped (`01fb8676a`, `M2_RECORD_UPDATE_FIX`). Its own comment says it "becomes dead code
> and is removed the moment the resolver fix ships." Confirming deadness needs reproducing the original
> cross-module repro — tracked separately, NOT retired here.

## Deletion gate (do NOT skip)

Per the strategy review R1 safety gate: a teaching-prompt line is deleted **only after**
its replacement diagnostic ships **and** the rig A/B shows no pass-rate loss. This milestone
ships the diagnostics + fixtures; the prompt-line deletion pass + A/B are a **separate,
gated step** and are intentionally NOT done here.

## KPI

- Prompt lines deletable once the shipped diagnostics clear A/B: the `++` section (≈18 lines)
  and the import-placement pre-emption — reported at the next baseline as "prompt tokens
  deleted per release".
