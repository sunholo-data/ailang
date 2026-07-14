# M-DX-AI-DISCOVERY: One-Shot Stdlib Discovery for AI Agents (RE-SCOPED)

**Status**: Planned (RE-SCOPED 2026-07-14, mission iteration 30 — original v0.10.1-era doc was
partially superseded; every premise below re-verified live at HEAD `39d671a52`, v0.29.2-194)
**Target**: v0.30.0
**Priority**: P2 (clause-3 accessibility; last starter in the Prelude/discovery group)
**Estimated**: 1–1.5 days
**Dependencies**: None
**Original evidence**: M-EVAL-XLANG benchmark analysis (v0.10.x era — see Superseded Premises)

## What changed since the original doc (Gate-2 scope check, 2026-07-14)

The original doc (Target v0.10.1) predates most of today's discovery surface. Live probes at
HEAD `39d671a52` (both binaries rebuilt, `--version` = `git describe`):

**Superseded — do NOT rebuild:**

| Original claim | Reality at HEAD (probed 2026-07-14) |
|---|---|
| Agents burn ~17 turns running `ailang docs std/X` per module | The canonical `ailang prompt` (91,300 chars, served in the system role per the prompt-serving decision) teaches the stdlib comprehensively — 171 `std/` references incl. per-function signatures. The cold-start numbers are from a CLAUDE.md-only era. |
| Need `ailang docs` per-module output | Exists and is good: `ailang docs std/clock` prints exports with types + usage. `--list` enumerates all 44 modules with descriptions. |
| Need examples linkage | `docs --examples` landed iteration 29 (PR #392); `ailang examples` download/list landed iteration 28. |
| "MOD010: Unknown module" needs suggestions | **Wrong error code.** MOD010 is module-path-MISMATCH (`internal/pipeline/pipeline_module.go:755`). Unknown stdlib module is a different path — see M3 below (the gap itself is REAL, the code cited was not). |
| `ailang docs search <query>` for stdlib functions | **Namespace collision**: `ailang docs search` is TAKEN — it is the SimHash/neural design-doc search (`cmd/ailang/docs_search.go:17`, dispatched at `docs.go:34`). Probed: `ailang docs search "timestamp"` scans 1,277 design docs, zero stdlib functions. Any function search must NOT touch this subcommand. |

**Still REAL at HEAD — the residual scope (all probed):**

| Gap | Probe evidence (2026-07-14, HEAD binary) |
|---|---|
| No one-shot signature dump | `ailang docs --all-functions` → `flag provided but not defined`. 44 modules / **413 exported functions** (`grep -c '^export func \|^export pure func ' std/*.ail`) require 44 separate CLI calls today. This is the surface for agents that DON'T get the 91KB prompt (generic/fleet-tier agents, humans, grep pipelines). |
| No stdlib function filter | No way to answer "which function does X" from the CLI without knowing the module. (`docs search` is design-docs only, see above.) |
| Unknown std module: no did-you-mean, no module list | `import std/time (now)` → `ailang check` prints "stdlib module not found: std/time" + searched paths + AILANG_STDLIB_PATH tip (`internal/loader/stdlib_resolver.go:327` `errWithSearchTrace`) — but NO nearest-module suggestion and NO list of valid `std/` modules. |
| No prelude documentation | `ailang docs prelude` → `Error: module 'std/prelude' not found`. Yet the prelude EXISTS and just GREW: `println` (`internal/pipeline/prelude.go:35` `InjectPrelude`), `show` (builtin, no import), and iteration 27's implicit lowest-precedence `std/option` + `std/result` imports for entry modules (PR #382). Probed: `println(show(42))` and bare `Some(1)`/`None` match both run import-free. Nothing documents this surface for a user or agent. |

## Problem Statement (re-scoped)

The per-module discovery surface is complete, but there is no **one-shot, grep-able view of the
whole stdlib**, no **wrong-module recovery** (the classic `std/time` → `std/clock` trap yields a
dead-end error), and the **prelude is invisible** (its contents changed in v0.30-cycle and are
documented nowhere queryable). These hit exactly the fleet-tier/agentic-CLI audience of clause 3:
anything not driven by the canonical prompt discovers the stdlib by trial and error.

## Goals

1. Entire stdlib surface (module.function: type ! effects — description) in ONE deterministic,
   line-oriented, grep-able command.
2. Optional case-insensitive filter over that surface (name + description), WITHOUT touching the
   existing `docs search` design-doc namespace.
3. Unknown-stdlib-module errors become recoverable: nearest-match suggestion + the valid module
   list.
4. `ailang docs prelude` documents the real, current prelude — and is test-locked to the
   mechanisms so it cannot silently drift.

## Non-Goals

- Fuzzy/SimHash/neural function search — `--all-functions [filter]` + grep covers the need;
  `docs search` (design docs) is untouched.
- Any change to MOD010 (path mismatch — different, correct, already-shipped diagnostic).
- Prompt (`prompts/`) changes — the prompt already teaches the stdlib; this is the CLI lane.
- Benchmark re-runs (rig/GPU/API-billed; the original doc's "re-benchmark" criterion is dropped —
  success is behavioral + test-locked CLI output).
- Changing prelude SEMANTICS (iteration 27 owns that; we only document + test-lock it).

## Solution Design

### M1: `ailang docs --all-functions [filter]` (~220 LOC + tests)

New flag in `cmd/ailang/docs.go` (flag set at `docs.go:44-47`). Iterates `discoverModules`
(`docs.go:142`, already exists) with parser-backed signature extraction (below) to emit one line
per export:

```
std/clock.now: () -> int ! {Clock} -- Returns epoch time in production mode, virtual time in deterministic mode
std/list.map: ((a) -> b, [a]) -> [b] -- Apply function to each element
prelude.println: (string) -> () ! {IO} -- Print with newline (no import needed)
```

- **Parser premise VERIFIED, not assumed (quorum r3)**: a full sweep at HEAD compared every
  `export [pure] func` in `std/*.ail` against per-module docs output — **413/413 enumerated, 0
  missing** (V15). The parser (`parseModuleFile`, `docs.go:171`) is complete on the real corpus
  today; the risk is FUTURE drift and signature fidelity, addressed by the two rules below.
- **No silent omission**: signatures come from the AST (below), so there is no per-signature
  failure mode — the only failure is a stdlib FILE that does not parse, and that FAILS THE
  COMMAND loudly (non-zero exit, file named), never a dropped or partial row. (Stdlib always
  parsing is already CI's contract; quorum r5 removed an earlier regex-fallback bullet this
  replaced — it was dead logic once rendering became parser-backed.)
- **Exact-set test, not a count floor (quorum r3)**: the test independently scans `std/*.ail`
  for export names (the V15 sweep, in Go) and asserts the `--all-functions` name set EQUALS it
  (both directions), plus every line carries a non-empty, non-`[signature unparsed]` signature —
  a missing, extra, or malformed export fails deterministically.
- **Signature rendering is PARSER-BACKED, not regex (quorum r4)**: V16 proves the regex already
  corrupts signatures (`exportSigRe` at `docs.go:191` stops at `{`, so `now() -> int ! {Clock}`
  renders `now() -> int ! `), and a "non-empty signature" test cannot certify fidelity. M1
  therefore renders each export's signature FROM the real AST: parse each `std/*.ail` with
  `internal/parser` and print `name[tyvars](params) -> ret ! {effects}` from `ast.FuncDecl`
  fields (params/return/effects are declaration-level facts — stdlib exports are fully
  annotated). This is the SAME pattern iteration 29's quorum drove for imports
  (parser-backed `ast.File.Imports` extraction replacing ad-hoc scanning) — one shared
  extraction function, `cmd/ailang` already links `internal/parser` (same binary). The regex
  path survives ONLY for doc-comment prose (`-- description` lines, non-load-bearing; AST does
  not carry comments). Fixes BOTH `--all-functions` and existing `ailang docs std/X` signatures
  (V16 golden on std/clock).
- **Fidelity test (quorum r4)**: corpus-wide assertion — for EVERY export in every `std/*.ail`,
  the emitted signature equals the AST-derived rendering (independent re-parse in the test),
  plus spot goldens for the tricky forms: generics `name[a,b]`, effect rows `! {Clock}`, `pure
  func`, zero-return, multi-line params. A stdlib file the parser cannot parse fails the test
  loudly (stdlib must always parse — that is already CI's contract).
- Deterministic order (modules sorted, exports in file order) — diffable and cacheable.
- A trailing positional arg filters case-insensitively over the FULL rendered line — module
  path, function name, signature, and description (quorum r5 fixed the earlier wording, which
  literally excluded the function name): `ailang docs --all-functions timestamp` → the
  `std/clock` lines; `ailang docs --all-functions foldl` → the fold family. This IS the search
  story (M2 of the original doc, collapsed here — no new subcommand, no collision with
  `docs search`).
- Prelude entries included under pseudo-module `prelude.` (sourced from the M4 table).
- Output goes through the SAME `findStdlibDir` resolution (`docs.go:85`) as existing docs —
  no new path machinery (iteration-29 quorum r1 lesson).

### M3: Unknown-stdlib-module recovery (~80 LOC + tests)

Primary site: `errWithSearchTrace` (`internal/loader/stdlib_resolver.go:325`) — the rich path
that `ailang check`/`run` hit (probed). Append:

```
stdlib module not found: std/time
  did you mean: std/clock?
  available: std/ai, std/array, ... (44 modules)
searched: ...
```

- **Existing machinery REUSED, none duplicated (quorum r5 audit, command-backed)**: repo-wide
  grep for `levenshtein|didYouMean|nearest|suggest` found the codebase's ONE suggestion engine:
  `internal/importhint` (leaf package; `closestExport` = prefix-relation + Levenshtein ≤ 2,
  `levenshtein` at `importhint.go:103`), fed by `internal/stdlibindex` (symbol↔module scan) via
  the established CLI injection convention (`cmd/ailang/diagnostics_wiring.go` init sets
  `importhint.Locator`/`SymbolsOf` = `stdlibindex` funcs, keeping loader/types free of scan
  deps). M3 EXTENDS this exact stack: a module-name variant in `importhint` (reusing its
  existing `levenshtein` + matching thresholds), the module list from `stdlibindex`, injected
  into the loader's error path via the same wiring file. No second edit-distance
  implementation, no new diagnostic convention.
- Suggestion = curated alias table FIRST (`time→clock, date/dates→datetime, http/url→net,
  regexp→regex, strings→string, lists→list, os→process, ...` — data, living in `importhint`
  next to `autoImportedBuiltins`, its existing curated-data precedent), then the importhint
  distance pass against the module list. `time→clock` is edit-distance 5; ONLY the alias table
  can catch the highest-value trap, which is why distance alone (original doc's M3) was the
  wrong spec. **Determinism**: module list sorted lexicographically; among equal
  Levenshtein distances the lexicographically-smallest module wins; alias-table hit always
  outranks a distance hit. Exactly one `did you mean` line, ever.
- Module list source is DEFINED, not discovered ad-hoc: `internal/stdlibindex`'s existing scan
  (it already enumerates stdlib modules to build its symbol↔module maps; expose the module list
  if not already exported), injected into the loader error path via `diagnostics_wiring.go` —
  the loader gains no scan dependency, same as every other diagnostic hint.
- **No silent degradation (quorum r1, both reviewers):** if no stdlib root resolved or the glob
  fails, the error explicitly says so — `available: (module list unavailable — no stdlib root
  resolved; see 'searched' below)` — alias-table suggestions still print (they are static data),
  but the ABSENCE of the dynamic list is always announced, never skipped silently. The
  diagnostic path itself never throws (an error formatter must not fail), but it also never
  pretends: unavailable enumeration is stated as unavailable.
- **Systemic-fix audit — RESOLVED at design time (quorum r1, gemini):** the two terser sibling
  sites (`internal/module/resolver.go:145`, `internal/module/loader.go:276`) are in package
  `internal/module`, which has ZERO importers repo-wide and is NOT in `cmd/ailang`'s dependency
  graph (`go list -deps ./cmd/ailang | grep -c internal/module$` → 0; V11). They CANNOT fire
  from the CLI. The live path is exactly one: `internal/loader/stdlib_resolver.go`. The sprint
  touches only the live site; the dead package is recorded as a cleanup candidate (NOT deleted
  in this sprint — coding-standards forbids blind unused-code deletion; understanding recorded
  here: superseded by `internal/loader`).

### M4: `ailang docs prelude` (~60 LOC + tests)

Special-case the module name `prelude` in `docsCommand` (`docs.go:74-78`) before stdlib file
lookup:

```
Prelude (available without import in ENTRY modules — library modules must import):
  println : (string) -> () ! {IO}      -- internal/pipeline/prelude.go InjectPrelude
  show    : (a) -> string              -- builtin
  Implicit imports (lowest precedence, entry modules only, since v0.30-cycle / PR #382):
    std/option  -- Some, None, Option[a] + combinators
    std/result  -- Ok, Err, Result[a,e] + combinators
  Shadowing: your own definitions win, silently (by design).
```

- **No duplicated table — the page is RENDERED FROM the live mechanisms** (quorum r2, gpt5-6-sol:
  a docs-side copy with a compile-only test detects stale entries but NOT additions/signature
  changes — structurally insufficient). Three sources, zero copies:
  1. **Implicit imports**: `internal/loader/prelude_imports.go` is ALREADY the single source of
     truth — `EntryPreludeModules()` (exported, `prelude_imports.go:39`) + `preludeModuleSymbols`
     (`prelude_imports.go:16`, needs a trivial exported accessor). `docs.go` iterates these
     directly; a module/symbol added to the implicit prelude appears in the docs page
     AUTOMATICALLY. (`cmd/ailang` already imports `internal/loader` — `run_helpers.go`,
     `main.go` — no new dependency edge.)
  2. **Injected bindings** (`println`): rendered from `pipeline.InjectPrelude`'s surface via a
     small exported enumerator; the REVERSE test walks the actual injected env (names + scheme
     `String()` incl. `! {IO}` effects) and asserts the docs renderer emits exactly that set —
     both a removed AND an added binding fail the test.
  3. **`show`**: builtin-sourced (not in either mechanism above); one explicit entry, guarded by
     a forward compile-probe fixture.
- **Bidirectional drift lock**: forward — every rendered name compiles import-free in an entry
  fixture; reverse — the enumerated live surface (loader accessors + InjectPrelude env) equals
  the rendered set exactly. Scope semantics on the page (entry-only, lowest precedence, silent
  shadowing) are each backed by a regression fixture: library-module rejection (V13),
  local-`type Option` shadowing (iter-27 suite — sprint verifies the test exists and links it,
  adds it if missing).
- `docs --list` gains one footer line pointing at `ailang docs prelude`.

## Conflict Surface

- `cmd/ailang/docs.go` — CLI-only; no parser/types/codegen contact.
- `internal/parser` — READ-ONLY reuse (M1 parses stdlib files to render signatures); zero
  grammar/AST changes. Why reuse is right (quorum r4 asked): the regex path demonstrably
  corrupts signatures (V16) and any hand-rolled fixer re-derives what `ast.FuncDecl` already
  holds; iteration 29 set the precedent (parser-backed import extraction). Full pipeline
  type-checking (iface schemes) was considered and REJECTED for this sprint: declaration-level
  annotations are authoritative for stdlib (fully annotated by policy), and compiling 44 modules
  on every docs call adds latency + failure modes a docs command shouldn't have.
- `internal/loader/stdlib_resolver.go` — error-message path only; the resolution logic itself
  is untouched (the two `internal/module` sibling sites are dead code, V11).
- `internal/importhint` (module-name suggestion variant + alias table) + `internal/stdlibindex`
  (module-list accessor if absent) + `cmd/ailang/diagnostics_wiring.go` (one injection line) —
  all following their existing patterns; no new packages, no second suggestion engine (V17).
- FORBIDDEN zones (`internal/types/`, `internal/parser/`, `internal/effects/`) untouched.
- No cacheKeyVersion bump: no change to compilation semantics or artifacts.
- Concurrent-lane check: iteration 29 (PR #392) touched `docs.go` for `--examples` — build on
  merged HEAD; no open PR currently touches these files (verified: no ai-discovery PRs open).

## Success Criteria

- [ ] `ailang docs --all-functions` emits all 44 modules' exports (413 at HEAD + prelude), one
      line each, deterministic order; EXACT name-set equality vs an independent source scan
      asserted in tests (no count floors); unparseable signatures render visibly and fail tests.
- [ ] Effect rows render in full (`! {Clock}`, not `! `) in both `--all-functions` and
      per-module docs (V16 regression golden).
- [ ] Corpus-wide signature-fidelity test: every emitted signature equals the AST-derived
      rendering (independent re-parse); tricky-form goldens (generics/effects/pure/multi-line).
- [ ] `ailang docs --all-functions timestamp` (filter) returns `std/clock.now` + `nowMillis`-class
      lines and nothing unrelated.
- [ ] `ailang check` on `import std/time (now)` prints `did you mean: std/clock?` + the module
      list (golden/CI fixture).
- [ ] `ailang docs prelude` is RENDERED from the live mechanisms (loader prelude accessors +
      `InjectPrelude` enumeration; `show` probe-guarded) with a bidirectional drift test —
      an added OR removed prelude name/signature fails `make test`.
- [ ] `ailang docs search` (design-doc SimHash) behavior byte-identical (regression guard).
- [ ] `make test` green; zero test deletions.

## Implementation Plan

- **M1** `--all-functions [filter]` — day 0.5
- **M3** not-found recovery + 3-site audit — day 0.5
- **M4** `docs prelude` + test-lock + `--list` footer — day 0.25
- Docs: CHANGELOG + `docs/docs/guides/` AI-discovery note — day 0.25

(Original M2 "search subcommand" is collapsed into M1's filter; original milestone numbers kept
for traceability.)

## Verification Log (all commands run 2026-07-14 at HEAD `39d671a52`, fresh binaries)

| # | Command | Observed |
|---|---|---|
| V1 | `ailang docs --all-functions` | `flag provided but not defined: -all-functions` |
| V2 | `ailang docs search "timestamp"` | SimHash over 1,277 DESIGN docs (collision confirmed) |
| V3 | `ailang docs prelude` | `Error: module 'std/prelude' not found` |
| V4 | `ailang check` w/ `import std/time (now)` | not-found + searched paths, NO suggestion/list |
| V5 | `grep -c '^export func \|^export pure func ' std/*.ail` | 413 exports across 44 modules |
| V6 | entry module `println(show(42))`, no imports | runs clean |
| V7 | entry module `match Some(1) {...}`, no imports | runs clean (iter-27 implicit prelude) |
| V8 | `ailang prompt \| wc -c` / `grep -ic std/` | 91,300 chars / 171 std/ refs |
| V9 | `grep -rn "stdlib module not found" internal/` | 3 sites: loader/stdlib_resolver.go:327 (rich), module/resolver.go:145, module/loader.go:276 |
| V10 | `grep -n MOD010 internal/pipeline/pipeline_module.go` | MOD010 = path mismatch, not unknown module |
| V11 | `go list -deps ./cmd/ailang \| grep -c "internal/module$"` + repo-wide import grep | 0 — package `internal/module` has no importers; both sibling not-found sites are DEAD code; live site is exactly `internal/loader/stdlib_resolver.go` |
| V12 | entry module `match Ok(7) { Ok(x) => ..., Err(e) => ... }`, no imports | runs clean (implicit std/result confirmed live, not just std/option) |
| V13 | LIBRARY module (no `main`) using `Some(1)` | `undefined variable: Some` — implicit prelude correctly entry-only |
| V14 | read `internal/loader/prelude_imports.go` | `EntryPreludeModules()` exported at :39; `preludeModuleSymbols` map at :16 (option: Option/Some/None, result: Result/Ok/Err); the manifest for M4's rendering already exists |
| V15 | full sweep: every `export [pure] func` name in `std/*.ail` vs `ailang docs std/<mod>` output, all 44 modules | **413/413 present, 0 missing** — parseModuleFile is complete on the current corpus (incl. generic `name[a](…)` forms) |
| V16 | `ailang docs std/clock` signature rendering | `now() -> int ! ` — effect row TRUNCATED (exportSigRe stops at `{`); real defect, in scope for M1 |
| V17 | `grep -rniE "levenshtein\|didYouMean\|nearest\|suggest" internal/ --include=*.go` (defs) | ONE suggestion engine exists: `internal/importhint` (closestExport: prefix + Levenshtein ≤2, levenshtein at :103), fed by `internal/stdlibindex` via `cmd/ailang/diagnostics_wiring.go` injection — M3 extends this stack, adds nothing parallel |
