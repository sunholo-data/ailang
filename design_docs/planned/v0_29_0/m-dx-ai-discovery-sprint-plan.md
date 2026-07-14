# M-DX-AI-DISCOVERY — Sprint Plan (Executor Handoff)

**Sprint JSON**: `.ailang/state/sprints/sprint_M-DX-AI-DISCOVERY.json`
**Design doc**: `design_docs/planned/v0_29_0/m-dx-ai-discovery.md`
**Mission iteration**: 30
**Planned at HEAD**: `648a919beeca76ec56b658e744c15178585d9765` (origin/dev)
**Target**: v0.30.0 · **Priority**: P2 · **Risk**: low
**Estimate**: 1 day (~11h) across 4 code milestones + docs; ≤1.5 days with buffer. LOC ≈ 420.

> The design doc was re-scoped/quorumed at HEAD `39d671a52`. Actual HEAD is `648a919be`
> (4 intervening commits, dashboard-data + CI only). `git diff 39d671a52..HEAD` over **every**
> M-relevant file is **empty** — all citations and probes below re-verified at `648a919be`.

---

## Executor Constraints (read before touching code)

- **Worktree**: branch from `origin/dev`, isolated git worktree (a sibling agent shares the working tree). Commit promptly.
- **FORBIDDEN**: `internal/types/`, `internal/effects/`, and `internal/parser/` grammar/AST changes.
  `internal/parser` is used **read-only** via its public `New(lexer)`/`Parse()` API + the `ast`
  package's existing `Type.String()` / `FormatEffects` renderers. No new grammar, no AST fields.
- **Do NOT** bump `cacheKeyVersion` (no compilation-semantics change).
- **Do NOT** touch `cmd/ailang/docs_search.go` or the `ailang docs search` behavior — it must stay byte-identical (regression-guarded in M5).
- **Do NOT** touch/delete `internal/module/resolver.go:145` or `internal/module/loader.go:276` — dead siblings (package has 0 importers, V11).
- **No** GPU/ollama/benchmark steps. Success is behavioral + test-locked.

---

## Premise discrepancies found

The mission's planners have caught doc errors in 5 of the last 6 iterations. I reality-checked
every file:line citation and mechanism claim. **Two discrepancies found — both REFINE the plan,
neither breaks it. The core scope is sound.**

### D1 (informational) — V11 audit missed a second LIVE not-found path in `internal/loader`

The doc's systemic audit (V11) only checked the **dead** `internal/module` siblings. There is a
**second live** module-not-found path in the very package M3 edits:

- `internal/loader/loader.go:202` and `:251` call `newLDR001` (`loader_errors.go:52`) with
  `ml.suggestSimilar(path)` (`loader_errors.go:92`) — a structured `LDR001` error that **already
  carries a `similar[]` suggestion** (a substring/path-component heuristic, not Levenshtein).

**Why scope still holds**: the loader branches on `strings.HasPrefix(canonPath, "std/")` *before*
the stdlib resolver. `LDR001`/`suggestSimilar` fires only for **non-stdlib** (project/package/self)
imports. The `std/time → std/clock` trap is exclusively the stdlib branch → `errWithSearchTrace`
(`stdlib_resolver.go:325`), reached via `loader.go:165` (ResolveStdlib fail) → embedded fallback
fail → `loader.go:181 return nil, err` (raw). That IS M3's target and has **no** suggestion today.
Verified by caller trace + the V4 live probe (`ailang check import std/time` prints the raw
`errWithSearchTrace` text, not the `LDR001` "module not found" format).

**Action for executor**: record in the PR body's systemic-audit note that there are **two** live
not-found paths; M3 owns the stdlib one; `LDR001`/`suggestSimilar` (local modules) already has a
suggestion and is intentionally out of scope. **Do not** unify them (separate follow-up if ever wanted).

### D2 (informational) — three accessors the doc calls "trivial/if-absent" are genuinely ABSENT

The doc hedges ("expose the module list if not already exported", "needs a trivial exported
accessor", "a small exported enumerator"). All three are in fact **absent** and are **required**
work (small additive exports, already in the LOC estimate):

1. `internal/stdlibindex` has **no** all-modules accessor (only `Modules(symbol)` and
   `SymbolsOf(module)`). **Add** `AllModules() []string` (sorted, from the built `idx` set). — M3
2. `internal/loader.preludeModuleSymbols` (`prelude_imports.go:16`) has **no** exported accessor.
   **Add** `EntryPreludeSymbols(modPath string) []string` (or a `PreludeModuleSymbols()` map copy). — M4
3. `pipeline.InjectPrelude` (`prelude.go:35`) **hardcodes** a single binding
   (`env.ExtendScheme("println", ...)`) and has **no** enumerator. **Add** a small surface
   enumerator (names + `scheme.String()`) for the reverse drift test. — M4

**Everything else in the doc verified clean** (see the Verified-premises table below).

### Verified premises (no discrepancy)

| Claim | Status at `648a919be` |
|---|---|
| V1 `--all-functions` undefined | ✅ `flag provided but not defined: -all-functions` |
| V5 413 exports / 44 modules | ✅ `grep -c` = 413; `ls std/*.ail` = 44 |
| V16 effect-row truncation | ✅ `ailang docs std/clock` → `now() -> int ! ` (dropped `{Clock}`); root cause `exportSigRe` at `docs.go:191` stops at `[^{=]+` |
| M1 AST path viable | ✅ `ast.FuncDecl` (`ast_decl.go:45`) has Name/TypeParams/Params/ReturnType/Effects/IsExport; all `ast.Type` variants have `String()` (`ast_type.go`); `ast.FormatEffects` (`ast.go:126`) → `! {A, B}`; `Program.File.Funcs` populated (`parser_file.go:73`) |
| docs.go anchors | ✅ flags :44-47; search dispatch :34; discoverModules :142; parseModuleFile :170; findStdlibDir :85; showModuleDocs :274; formatExportSignature :421; listModules :123 |
| V3 `docs prelude` | ✅ `Error: module 'std/prelude' not found` |
| V4 unknown module (live path) | ✅ raw `errWithSearchTrace` text, no suggestion/list |
| M3 live-path trace | ✅ `loader.go:165→181` returns raw `errWithSearchTrace` (`stdlib_resolver.go:325`, msg at :327) — the only live stdlib not-found path |
| V11 dead siblings | ✅ `go list -deps ./cmd/ailang \| grep -c internal/module$` = 0 |
| V17 engine reuse | ✅ `internal/importhint` (`levenshtein` :103, `closestExport` :65), fed by `internal/stdlibindex` via `diagnostics_wiring.go:11-20`. `levenshtein` is unexported but same-package → variant reuses it directly |
| M4 sources | ✅ `EntryPreludeModules()` exported :39; `preludeModuleSymbols` :16; `InjectPrelude` injects only `println : string -> () ! {IO}`; `show` is builtin-only |
| No new dep edge | ✅ `cmd/ailang` already links parser/loader/pipeline |

---

## Milestone M1 — `ailang docs --all-functions [filter]` (~220 LOC, ~4h)

One deterministic, grep-able line per stdlib export, **signatures rendered from the AST** (this also
fixes the V16 effect-row truncation in per-module `ailang docs std/X`). Optional positional filter.

**Anchors / approach**
- Add `--all-functions` bool to the flag set at `cmd/ailang/docs.go:44-47`. Handle it after
  `findStdlibDir` (`docs.go:60`), before the `--list` branch. Trailing positional (`docsFlags.Arg(0)`)
  = optional case-insensitive filter.
- **New shared signature helper**: for each `std/*.ail`, parse via `lexer.New` + `parser.New` +
  `Parse()` (same API as `debug.go:183-185`, `test.go:97-98`), walk `prog.File.Funcs`, keep
  `IsExport == true`. Render `name[T1,T2](p1, p2) -> ret ! {Eff}` from `FuncDecl` fields:
  `TypeParams` (bracketed if non-empty), `Params[].Type.String()`, `ReturnType.String()`,
  `ast.FormatEffects(Effects)`.
- **Replace the `exportSigRe` regex signature path in BOTH** `--all-functions` **and**
  `showModuleDocs`/`formatExportSignature` (`docs.go:306,421`). Keep `parseModuleFile`'s doc-comment
  scan for the `-- description` **prose only** (the AST carries no comments).
- Output line: `std/clock.now: () -> int ! {Clock} -- Returns epoch time...`. Modules sorted
  (`discoverModules` already sorts, `docs.go:163`), exports in file order.
- Prelude entries under pseudo-module `prelude.` — source from M4's shared prelude renderer (one
  prelude source of truth). If M1 lands first, stub and wire in the same PR when M4 lands.
- Filter: keep lines whose lowercased **full rendered line** (module + name + signature +
  description) contains the lowercased filter arg. Substring, no regex.
- **Failure mode**: a `std/*.ail` that does not parse → name it on stderr, exit non-zero. Never
  drop/partial-render a row silently. (Stdlib always parsing is CI's contract.)

**Tests** (`cmd/ailang/docs_all_functions_test.go` or add to `docs_test.go`)
- **Exact name-set (NOT count floor)**: independent Go scan of `std/*.ail` export names ==
  `--all-functions` name set, **both directions**.
- Every line has a non-empty, non-`[signature unparsed]` signature.
- **Corpus-wide fidelity**: for every export, emitted signature == an in-test independent AST re-parse rendering.
- Tricky-form goldens: generics `name[a,b](...)`, effect rows `! {Clock}`/multi-effect, `pure func`, zero-return, multi-line params.
- **V16 golden**: `ailang docs std/clock` now shows `now() -> int ! {Clock}`.
- Filter: `--all-functions timestamp` includes `std/clock.now`, excludes unrelated modules.
- Unparseable-file self-test: temp bad `.ail` → non-zero exit naming the file.

**Verify**
```
./bin/ailang docs --all-functions | head
./bin/ailang docs --all-functions timestamp
./bin/ailang docs std/clock | grep 'now'   # expect: now() -> int ! {Clock}
go test ./cmd/ailang/ -run AllFunctions
```

---

## Milestone M3 — Unknown-stdlib-module recovery (~90 LOC, ~3h)

Append `did you mean: std/clock?` + the valid module list to `errWithSearchTrace`. Alias table
FIRST, then Levenshtein ≤2, reusing `internal/importhint` + `internal/stdlibindex`.

**Anchors / approach**
- **Add a module-name suggestion variant in `internal/importhint`** (same package as `levenshtein`
  at :103 — reuse it, do **not** re-implement). Curated **alias table** next to
  `autoImportedBuiltins` (its curated-data precedent): `time→clock, date/dates→datetime,
  http/url→net, regexp→regex, strings→string, lists→list, os→process, ...`.
  - Order: alias-table hit **always** outranks a distance hit; among equal Levenshtein distances the
    **lexicographically-smallest** module wins; module list sorted first. **Exactly one**
    `did you mean` line, ever. `time→clock` is edit-distance **5** — only the alias table catches it.
- **Add `stdlibindex.AllModules() []string`** (D2, sorted, from the built `idx` module set). Inject
  into `importhint` via `cmd/ailang/diagnostics_wiring.go` (one line, same convention as
  `Locator`/`SymbolsOf` at :16,:19). `importhint` stays a leaf package.
- **Edit `errWithSearchTrace`** (`stdlib_resolver.go:325`): after the existing
  `stdlib module not found` + `searched:` + `tip:` block, append (only if a suggestion exists)
  `  did you mean: std/<X>?` and `  available: std/a, std/b, ... (N modules)`. `internal/loader`
  already imports `importhint` (`loader_errors.go:11`).
- **No silent degradation**: if the injected module list is nil/empty (no stdlib root resolved),
  print `  available: (module list unavailable — no stdlib root resolved; see 'searched' below)`.
  Alias-table suggestions (static data) still print. Formatter never throws, never silently skips.

**Tests**
- `importhint` unit: alias hit (`time→clock`), distance hit (`lst→list`), no-match (`""`),
  tie-break lexicographic, alias-outranks-distance.
- `stdlib_resolver` golden (`stdlib_resolver_test.go`): full `errWithSearchTrace` text for
  `std/time` incl did-you-mean + available list.
- module-list-unavailable path prints the explicit note (inject nil module locator).
- Regression: a genuinely-unknown module with no close match prints **no** misleading line.

**Verify**
```
printf 'module t\nimport std/time (now)\nexport func main() -> () { () }\n' > /tmp/t.ail
./bin/ailang check /tmp/t.ail   # expect: did you mean: std/clock?  + available list
go test ./internal/importhint/ ./internal/stdlibindex/ ./internal/loader/
```

---

## Milestone M4 — `ailang docs prelude` (~80 LOC, ~3h)

Rendered from live mechanisms + bidirectional drift test. `docs --list` footer line.

**Anchors / approach**
- In `docsCommand`, **before** the file-lookup at `docs.go:74-78`, special-case
  `moduleName == "prelude"` → `renderPreludeDocs()`.
- **Source 1 (implicit imports)**: iterate `loader.EntryPreludeModules()` (`prelude_imports.go:39`)
  + new `loader.EntryPreludeSymbols(mod)` (D2, over `preludeModuleSymbols` :16). Adding a
  module/symbol to the implicit prelude updates the page automatically.
- **Source 2 (injected bindings)**: add a small exported enumerator over `pipeline.InjectPrelude`'s
  surface (currently just `println : string -> () ! {IO}`). Build a base `TypeEnv`, snapshot, call
  `InjectPrelude`, enumerate the **added** names + `scheme.String()`. Render from that — no
  hardcoded `"println"` string in `docs.go`.
- **Source 3 (`show`)**: builtin-only. One explicit entry, guarded by a forward compile-probe fixture.
- Print the scope notes: entry-only, lowest precedence, silent shadowing.
- **`--list` footer**: `listModules` (`docs.go:123`) gains one footer line pointing at `ailang docs prelude`.

**Tests** (`cmd/ailang/docs_prelude_test.go` + fixtures)
- **Bidirectional drift**: FORWARD — every rendered name compiles import-free in an entry fixture;
  REVERSE — the enumerated live surface (loader accessors + `InjectPrelude` env names+`scheme.String()`)
  == the rendered set exactly. An added AND a removed binding both fail `make test`.
- V13 fixture: a **library** module (no `main`) using `Some(1)` → `undefined variable: Some`.
- Local `type Option` shadowing regression fixture exists + is linked (add if absent — verify iter-27 suite).
- `show` forward compile-probe fixture.

**Verify**
```
./bin/ailang docs prelude
./bin/ailang docs --list | tail -3   # expect the new footer line
go test ./cmd/ailang/ -run Prelude
```

---

## Milestone M5 — Docs + CHANGELOG + docs-search guard (~30 LOC, ~1h)

- **CHANGELOG**: `changelogs/v0.18-current.md` (the active file — confirmed). Entry covering
  `--all-functions [filter]`, unknown-module recovery, `docs prelude`. Grouped, semver.
- **Guide note**: short AI/stdlib-discovery note under `docs/docs/guides/`. Import any code blocks
  from `examples/` (coding-standards: no inline-embedded runnable code).
- **docs-search regression guard** (`cmd/ailang/docs_test.go`): assert `ailang docs search <query>`
  behavior byte-identical (still SimHash over design docs). Note: the `search` subcommand is
  dispatched at `docs.go:34` (`args[0] == "search"`) **before** flag parsing — verify M1's
  flag/positional handling never intercepts it.

**Verify**
```
./bin/ailang docs search "timestamp" | head   # unchanged: design-doc SimHash
make test
```

---

## Sequencing

M1, M3, M4 are **independent** (any order / parallel). M5 last. M1 and M4 share the prelude
renderer — whichever lands second wires M1's `prelude.*` entries to M4's accessor (same PR).

**Hard rules**
- M5 after M1+M3+M4.
- docs-search byte-identical guard green (M1's positional filter must not intercept the `search` subcommand).
- `internal/types/`, `internal/parser/` (grammar/AST), `internal/effects/` untouched.
- `internal/module` dead siblings untouched.

---

## Declared-deviation protocol

If reality diverges from this plan mid-execution:
1. **Stop** at the first surprise; do not silently re-scope.
2. Record the deviation in the sprint JSON milestone `notes` field (what, why, evidence: file:line or command output).
3. If a deviation would touch a FORBIDDEN zone (`internal/types`, `internal/parser` grammar/AST,
   `internal/effects`) or bump `cacheKeyVersion`: **do not proceed** — quarantine the milestone and
   file a follow-up issue instead. These constraints are non-negotiable for this sprint.
4. If a doc premise is found false at execution time (a 6th-of-7 catch), record it in the PR body's
   systemic-audit note alongside D1/D2, and adjust only the affected milestone.
5. Report calibrated status: verified ✓ / not-yet-verified ✗ / could-still-break — never a premature
   "it works" until `make test` is green end-to-end.

---

## Definition of done

- [x] `ailang docs --all-functions` → all exports + prelude, one deterministic line each; exact
      name-set equality vs independent scan (no count floors); unparseable file → loud non-zero.
- [x] Effect rows render in full (`! {Clock}`) in `--all-functions` AND per-module docs (V16 fixed).
      Note: `now` renders as `now(())` — the AST faithfully shows its unit param; the golden asserts
      the full `! {Clock}` row (AST fidelity is the source of truth).
- [x] Corpus-wide AST signature-fidelity test + tricky-form goldens.
- [x] `ailang docs --all-functions timestamp` filters over the full rendered line (substring incl
      description), so it keeps std/clock + std/datetime "timestamp" lines and excludes unrelated
      modules (per the plan body: substring over the full line, not module-name-only).
- [x] `ailang check` on `import std/time` → `did you mean: std/clock?` + module list (golden);
      module-list-unavailable → explicit note.
- [x] M3 reuses `importhint.levenshtein` + `stdlibindex` module list via `diagnostics_wiring.go`
      injection; no parallel engine; dead siblings untouched.
- [x] `ailang docs prelude` rendered from live mechanisms; bidirectional drift test (added OR
      removed binding fails `make test`); `--list` footer line.
- [x] `ailang docs search` byte-identical (regression guard).
- [x] CHANGELOG + guide note; `make test` green (104 pkg ok / 0 FAIL); zero test deletions; no
      `cacheKeyVersion` bump.

---

SPRINT_PLAN_PATH: design_docs/planned/v0_29_0/m-dx-ai-discovery-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-DX-AI-DISCOVERY.json
