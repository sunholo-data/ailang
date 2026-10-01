# M-PARSER-NULLARY-SINGLE-CTOR-CURSOR: Leading-Pipe Nullary Constructor Swallows the Next Declaration's First Token

**Status**: Planned
**Target**: v0.51.1
**Priority**: P0 (High) — silently corrupts module export surfaces; the plain form of a one-constructor ADT is unusable
**Estimated**: 1 day (1h fix + 3h tests/verification + buffer)
**Dependencies**: None
**Bug Report**: v0.50.0 release issue (package `rx/p`, IMP010 on a correctly-exported symbol) — coordinator task `task-c0dad6ad`

## Problem Statement

A one-constructor ADT written with the leading pipe — `export type J = | Idle` — and **no
`deriving` clause** over-advances the parser cursor past the last token of its own declaration.
The `ParseFile` main loop (`internal/parser/parser_file.go:78-80`) advances exactly one token
between declarations under the convention *every decl parser leaves the cursor AT the last token
of its declaration*; the extra advance therefore **swallows the first token of the next
declaration**.

**Consequences (all reproduced against v0.50.1, commit `021c469`, see Verification Log):**

| Next declaration after `type J = \| Idle` | Symptom |
|---|---|
| `export type W = { ... }` | `export` keyword swallowed → W parsed non-exported → importer gets **`IMP010: symbol 'W' not exported`** while the source visibly says `export type W` |
| `export pure func mk(...)` | `export` swallowed → `pure` seen → mk parsed as **non-exported** pure func (importer: IMP010) |
| `pure func mk(...)` | `pure` swallowed → mk parsed as plain **non-pure** func — **check passes, semantics silently wrong** |
| `func mk() -> int = ...` | `func` swallowed → cascading `PAR_NO_PREFIX_PARSE` / `PAR015` errors on mk's own signature (the "undefined variable"-style noise the reporter saw in a bigger module) |
| nothing (last decl in file) | no symptom — the bug needs a follower, which is why it survived |

**Minimal reproduction** (reporter's case, reproduced verbatim):

```ailang
-- c.ail
module rx/p/c
export type J = | Idle
export type W = { j: J, n: int }
export pure func mk(n: int) -> W = { j: Idle, n: n }
```
```ailang
-- b.ail
module rx/p/b
import ./c (W, mk)
export pure func main() -> int = mk(2).n
```
```console
$ ailang run --quiet --package-dir . --entry main rx/p/b.ail
Error: IMP010: symbol 'W' not exported by 'pkg/rx/p/c'
```

The error points at the *wrong declaration*: W's `export` keyword is present in the source; the
parser ate it as the "advance between declarations" step. IMP010 is produced by the linker
(`internal/link/module_linker.go:124`) and the loader (`internal/importhint/importhint.go`
serves both producers), so the diagnostic honestly reports the (corrupted) module interface —
the corruption happened earlier, in the parser.

**Why the workaround works, and why that matters:** `export type J = | Idle deriving (Eq)`
succeeds only by accident of the cursor convention — `parseDeriving` ends with
`expectPeek(RPAREN)` and leaves the cursor AT `)`, the declaration's real last token, so the
main loop's advance lands correctly. This makes the bug look intermittent and
construct-dependent ("add `deriving (Eq)` and it compiles"), which is exactly the kind of
non-deterministic-seeming failure that wastes AI-agent debugging budget.

**Related DX gap (same report, addressed only as Future Work here):** without the leading pipe,
`export type J = Idle` parses as a **type alias** (`parser_type_decl.go:177-190`: single IDENT
body with no following PIPE is unconditionally an alias), so `Idle` never becomes a constructor
and `match x { Idle => ... }` fails with `undefined variable: Idle`. The leading pipe is
therefore the *only* way to write a one-constructor ADT — and this bug makes that form
unreliable. Fixing the cursor bug restores the plain spelling `type J = | Idle`.

**Second instance of a known bug class.** The v0.8.1 fix
([m-parser-export-multiline-adt](../../implemented/v0_8_1/m-parser-export-multiline-adt.md))
repaired the *with-fields* single-constructor path of the same function
(`parseTypeDeclBody`) for the identical symptom, and its systemic audit explicitly flagged the
nullary leading-pipe path: *"Nullary constructors (line 401-409 leadingPipe path):
`p.nextToken()` after name — needs review but different path (no RPAREN involved)"*. That
review never happened; the flagged line is this bug. This doc closes the class.

**Impact:**
- Any AI-generated or hand-written module using the idiomatic Haskell-style one-constructor ADT
  form fails confusingly (IMP010 naming an unrelated, correctly-exported symbol) or, worse,
  **silently** drops `export`/`pure` flags from the following declaration.
- No current `std/` or `examples/` program uses the broken form (verified by grep, see
  Verification Log), so nothing shipped regressed — but the form is the natural spelling for
  state-machine/status types (`Idle`, `Done`, `Empty`), a very common AI-generation pattern.

## Goals

**Primary Goal:** `export type J = | Idle` (single nullary constructor, leading pipe, no
deriving) leaves the parser cursor AT `Idle`, so every following declaration parses exactly
as written.

**Success Metrics:**
- The reporter's two-module `rx/p` case runs and prints `2`.
- A declaration following `type J = | Idle` retains its `Exported` and `IsPure` flags verbatim
  from source (unit-test asserted).
- No std/example/check behavior change for any other type-body form (full matrix in Testing
  Strategy).
- Zero regressions: `make test` and `make test-core` pass.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix by guarding the advance (`peek PIPE or DERIVING`), not by removing it | Removing the advance would break the multi-variant loop entry, which requires the cursor AT PIPE | agent | design | low |
| `type J = Idle` (no pipe) **stays a type alias** | Changing alias-vs-sum disambiguation for single uppercase idents would break every alias to a named type (`type Handler = Fn`-style) and contradicts the documented disambiguation rule | human (locked by precedent: parser_type_decl.go comment "No pipe → simple type alias") | design | high |
| No new diagnostic codes; no `deriving` requirement for one-ctor ADTs | The bug is cursor positioning; adding diagnostics would mask the real fix | agent | design | low |
| Match the v0.8.1 fix pattern (leave cursor AT last token) rather than a general cursor-invariant framework | Smallest change cost; a framework is a separate, larger design (Non-Goal) | agent | design | med |

## Design Freeze

- [x] The fix is confined to the nullary `leadingPipe` branch of `parseTypeDeclBody`
      (`internal/parser/parser_type_decl.go:168-176`); all other branches stay byte-identical.
- [x] `type J = Idle` (no leading pipe) remains a type alias — no disambiguation change.
- [x] No new syntax, no new error codes, no semantic changes beyond restoring source-faithful
      `Exported`/`IsPure` flags.

## Deferred Decisions

- Test fixture organization (table-driven cases in `type_test.go` vs a new
  `type_decl_cursor_test.go`) — agent may choose.
- Whether the regression suite also gains an end-to-end package-import fixture (loader-level
  IMP010 repro) or relies on parser-level flag assertions — agent may choose; parser-level is
  the minimum bar.
- A "did you mean `type J = | Idle`?" hint on `undefined variable: Idle` when the identifier
  matches a type-alias target — **human at review**; out of this sprint's scope (Future Work).

## Solution Design

### Overview

`ParseFile`'s declaration loop (`parser_file.go:78-80`) advances exactly one token between
declarations. The contract that makes this correct: *a decl parser returns with the cursor AT
the last token of its declaration*. The nullary leading-pipe branch of `parseTypeDeclBody`
violates the contract with an unconditional `p.nextToken()` after the constructor name. When
no `deriving` and no further `|` follows, that advance moves the cursor onto the first token
of the *next* declaration; the main loop then advances again, and `parseTopLevelDecl`
dispatches on the *second* token of the next declaration (`export type W` → dispatched as
`type`, i.e. non-exported).

### Architecture

The fix mirrors the guarded advance the with-fields branch already uses
(`parser_type_decl.go:162-164`):

```go
// internal/parser/parser_type_decl.go, leadingPipe nullary branch (line ~168)
if leadingPipe {
    // Definitely a sum type with no fields (e.g., Red, Green, Blue)
    firstVariant = &ast.Constructor{
        Name:   name,
        Fields: nil,
        Pos:    p.curPos(),
    }
    // Advance past the name ONLY when the next token belongs to THIS
    // declaration (another variant or a deriving clause). Otherwise the
    // name is the last token of the declaration; advancing would swallow
    // the first token of the NEXT declaration (cursor convention: leave
    // the cursor AT the last token — parser_file.go:78-80 advances between
    // declarations). Mirrors the guarded advance of the with-fields branch.
    if p.peekTokenIs(lexer.PIPE) || p.peekTokenIs(lexer.DERIVING) {
        p.nextToken()
    }
}
```

Trace of all four token shapes after `Idle` (lexer skips newlines, so "next token" may be on
any later line):

| peek after `Idle` | Behavior with fix | Cursor on return |
|---|---|---|
| `PIPE` (`\| Idle \| Busy`) | advance to PIPE → existing variant loop | AT last ctor name (unchanged today) |
| `DERIVING` (`\| Idle deriving (Eq)`) | advance to DERIVING → `parseDeriving` consumes through `)` | AT `)` (unchanged today) |
| anything else (incl. EOF) | stay AT `Idle` → single-ctor `AlgebraicType` → `parseTypeDeclaration`'s DERIVING checks both miss | AT `Idle` — the declaration's last token (FIXED) |

No other branch of `parseTypeDeclBody` changes:
- with-fields branch already leaves cursor AT `RPAREN` with a guarded PIPE advance (v0.8.1 fix);
- non-leading-pipe nullary branch only advances when peek is PIPE (else alias path via
  `parseType`, which follows the convention);
- record and alias bodies already return with the cursor at their last token.

### Implementation Plan

**Phase 1: Fix** (~1 hour)
- [ ] Apply the guarded advance in the `leadingPipe` nullary branch of `parseTypeDeclBody`.
- [ ] Add the cursor-convention comment citing `parser_file.go:78-80`.

**Phase 2: Regression tests** (~2 hours)
- [ ] Parser unit tests: two-decl programs `type J = | Idle` + each follower kind
      (`export type`, `export pure func`, `pure func`, `func`), asserting the follower's
      `Exported`/`IsPure` flags and clean parse (no PAR errors).
- [ ] Table cases for the unchanged forms: `| Idle | Busy`, `| Idle deriving (Eq)`,
      `| Wrap(int)` (fields), `Idle` (alias) — asserting current AST shapes to pin the
      conflict surface.
- [ ] End-to-end: the reporter's `rx/p` two-module package as a loader/linker fixture
      (imports `(W, mk)` from a module whose first decl is `export type J = | Idle`).

**Phase 3: Verification** (~1 hour)
- [ ] `make test`, `make test-core`, `make fmt`, `make lint`.
- [ ] Re-run the full isolation matrix (Verification Log rows V1-V10) against the fixed
      binary; all broken rows flip to the reporter's expected output (`2`).

### Pipeline Pass Coverage

Not applicable — this is a cursor-positioning fix inside `parseTypeDeclBody`; it adds no new
pipeline pass and touches no expression-lowering path.

## Files to Modify/Create

**Modified files:**
- `internal/parser/parser_type_decl.go` (+6/-2 LOC) — guarded advance in the `leadingPipe`
  nullary branch (line 168-176).
- `internal/parser/type_test.go` (+~80 LOC) — regression table cases for the cursor fix and
  conflict-surface pins.

**New files (optional, implementer's choice):**
- `internal/parser/type_decl_cursor_test.go` (~100 LOC) — if the implementer prefers a
  dedicated file for the two-decl follower matrix.

No std, docs, or changelog source changes required (CHANGELOG entry at release time only).

## Examples

### Before (broken — reporter's case)

```ailang
export type J = | Idle
export type W = { j: J, n: int }   -- ← parses as NON-exported (export token swallowed)
```
```console
$ ailang run --quiet --package-dir . --entry main rx/p/b.ail
Error: IMP010: symbol 'W' not exported by 'pkg/rx/p/c'   -- points at the wrong declaration
```

### After (fixed)

```ailang
export type J = | Idle
export type W = { j: J, n: int }   -- ← Exported=true, exactly as written
export pure func mk(n: int) -> W = { j: Idle, n: n }
```
```console
$ ailang run --quiet --package-dir . --entry main rx/p/b.ail
2
```

### The silent-corruption case (the reason this is P0)

```ailang
export type J = | Idle
pure func mk() -> int = 5          -- before: mk parses as NON-pure plain func, check passes
                                   -- after:  mk is a pure func, as written
```

## Success Criteria

- [ ] Reporter's `rx/p` case: `ailang run --quiet --package-dir . --entry main rx/p/b.ail` prints `2`
- [ ] `type J = | Idle` followed by `export type` / `export pure func` / `pure func` / `func`:
      follower parses with source-faithful flags (unit tests)
- [ ] `| Idle | Busy`, `| Idle deriving (Eq)`, `| Wrap(int)`, record and alias bodies: AST
      output unchanged (pinned by table tests)
- [ ] `make test` passes; `make test-core` passes; `make lint` clean
- [ ] CHANGELOG entry added for v0.51.1

## Conflict Surface

### Syntactic positions touched

One position only: the token immediately after a **nullary constructor name in the
leading-pipe branch** of `parseTypeDeclBody` (`internal/parser/parser_type_decl.go:168-176`),
i.e. the `Idle` in:

```
type <Name> = | Idle  <HERE>
```

The change converts an unconditional `p.nextToken()` at that position into a peek-guarded
advance. No grammar production is added, removed, or re-ordered; the accepted language is
unchanged — only the cursor position on return (and therefore the `Exported`/`IsPure`
attribution of the following declaration) changes.

### What else lives here

The token stream at `<HERE>` (the lexer skips newlines, so all of these may be on the same or
later lines):

| Token at `<HERE>` | Existing valid form | Shape | Fix behavior |
|---|---|---|---|
| `PIPE` | multi-variant ADT | `\| Idle \| Busy ...` | advance to PIPE, variant loop (unchanged) |
| `DERIVING` | deriving clause | `\| Idle deriving (Eq)` | advance, `parseDeriving` leaves AT `)` (unchanged) |
| `EXPORT` | next decl, exported | `export type/func ...` | **stay** — was swallowed (FIXED) |
| `PURE` / `FUNC` / `TYPE` / `EXTERN` / `@` / `TEST` / `PROPERTY` | next decl, unexported | any decl head | **stay** — was swallowed (FIXED) |
| `EOF` | last decl in file | — | stay; main loop's `!EOF` guard already handles it (unchanged result) |

### Disambiguation strategy

None needed — the guard is purely positional: *advance iff the peeked token belongs to the
current declaration* (`PIPE` continues the variant list; `DERIVING` is a suffix of the same
type declaration). Everything else begins a new declaration, and the cursor must stay put.
There is no context in which `PIPE` or `DERIVING` at that peek position belongs to the *next*
declaration: a declaration cannot begin with either token (`parseTopLevelDecl`'s switch has no
such entry), so the guard cannot misfire.

### Programs that MUST still work

Existing fixtures exercising the same syntactic position (all verified present; they become
regression fixtures in Phase 2):

1. `std/stream.ail:45-48` — `export type StreamConn = StreamConn(int) deriving (Eq)` followed
   by another exported single-ctor type (the v0.8.1 sibling path).
2. `std/regex.ail:22` — `export type Regex = Regex(string) deriving (Eq)` single-ctor ADT
   followed by exported decls.
3. `std/dom.ail:23-27` — multi-variant leading-pipe ADT (`| AddPanel(...) | ... | AddTimeline(string) deriving (Eq)`).
4. `std/option.ail:11` — `export type Option[a] = Some(a) | None` (non-leading-pipe multi-variant).
5. `std/env.ail:18-20` — multi-variant leading-pipe ADT ending in `deriving (Eq)`.

None of these pass through the broken branch (none is a single *nullary* ctor), so the fix
must leave all five byte-identical in AST terms — pinned by the table tests.

### What deliberately changes

Exactly one class of program changes behavior: any module containing
`type <T> = | <Nullary>` (no deriving) followed by another declaration. Before the fix such
programs either failed with misleading IMP010/PAR errors or — when the swallowed token was
`pure` — silently parsed the following func as non-pure/non-exported while `ailang check`
**passed**. After the fix they parse as written. A program could only "depend" on the old
behavior by relying on a follower being non-exported or non-pure *despite its source
keywords*; such a program was silently corrupted, not correct. No std or examples program
matches the pattern (verified by grep — Verification Log V8).

## Testing Strategy

**Unit tests** (`internal/parser/type_test.go` or new `type_decl_cursor_test.go`):
- Two-decl matrix: `type J = | Idle` × {`export type`, `export pure func`, `pure func`,
  `func`, `@route`-annotated `export func`} — assert follower flags + zero PAR errors.
- Single-decl forms: `| Idle` alone; `| Idle` at EOF — assert clean parse, cursor at EOF.
- Pin tests for unchanged forms: `| Idle | Busy`, `| Idle deriving (Eq)`, `| Wrap(int)`,
  `| Wrap(int) | Other`, `Idle` (alias), record body — assert AST/flags identical to today.

**Integration tests:**
- Loader/linker-level: reporter's `rx/p` two-module package fixture; `import ./c (W, mk)`
  resolves; `ailang run` prints `2`. Guards the IMP010 producer path
  (`internal/link/module_linker.go:124`) end-to-end.

**Regression-surface tests:**
- One test per "Programs that MUST still work" entry (the five std fixtures above), asserting
  their exported-symbol sets parse unchanged.

**Manual verification:**
- Re-run the Verification Log matrix (V1-V10) against the rebuilt binary.

## Non-Goals

- **Changing alias-vs-sum disambiguation** — `type J = Idle` stays a type alias. Making single
  uppercase idents into ADTs would break named-type aliases and contradicts the documented
  rule (`parser_type_decl.go`: "No pipe → simple type alias"). The plain one-ctor ADT form
  after this fix is `| Idle`.
- **A general cursor-invariant framework** for the parser — this fix follows the existing
  convention; a lint/test framework for the convention is a separate design if the class
  recurs a third time.
- **New diagnostics** for the `undefined variable: Idle` alias/match footgun — candidate
  Future Work, needs its own conflict-surface analysis in the typechecker.
- **Changing `deriving` semantics** — its accidental role as a workaround disappears; no
  deriving behavior changes.

## Timeline

**Single session (~4 hours + buffer = 1 day):**
- Phase 1: fix (~1h)
- Phase 2: tests (~2h)
- Phase 3: full verification + CHANGELOG (~1h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Guarded advance breaks a currently-working path through the branch | High | The only working multi-variant entry requires peek=PIPE, which still advances (trace table); full parser suite + five std fixtures pin it |
| Some caller depends on the old (over-advanced) cursor position | Medium | `parseTypeDeclBody` has exactly one caller (`parseTypeDeclaration`, same file) whose DERIVING peek-fallback already handles cursor-BEFORE-deriving; no other callers (grep, V7) |
| Test asserts flags but a downstream consumer re-derives export info differently | Low | End-to-end package fixture covers the real IMP010 path, not just AST flags |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a construct-dependent mis-parse — same source now always yields the same module interface |
| A2: Replayability | 0 | No trace/execution-path change |
| A3: Effect Legibility | +1 | Restores source-faithful `pure` annotation on declarations after a one-ctor ADT (effect rows were silently wrong) |
| A4: Explicit Authority | +1 | Restores source-faithful `export` — module authority surface was silently narrowed |
| A5: Bounded Verification | +1 | Correct module interfaces are checkable locally; corrupted exports made IMP010 undiagnosable from the importing side |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | The failure mode (IMP010 naming a correctly-exported symbol) is maximally misleading for AI repair loops; fix removes the false scent |
| A8: Minimal Syntax | +1 | Makes the existing minimal spelling `type J = \| Idle` work; removes the need for cargo-cult `deriving (Eq)` workarounds |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Cross-module one-ctor ADT composition (the rx/p pattern) currently broken |
| A11: Structured Failure | 0 | No error-handling changes (fixes the cause of a misleading error, not the error format) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — removes construct-position-dependent behavior
- [x] A3 (Effects): no hidden side effects — restores declared effects
- [x] A4 (Authority): no ambient access granted — restores declared export surface
- [x] A7 (Machines First): not optimizing for human convenience — removes a false diagnostic scent that misleads machine repair

## Verification Log

Every "does/doesn't work" claim above was verified against the v0.50.1 binary
(commit `021c469`, the reporter's build) on 2026-10-01. Repro tree: `eval/rx` package under
`/tmp/rxroot` (manifest `[package] name = "eval/rx"`), single-file probes under `/tmp`.

| # | Claim | Command | Result |
|---|---|---|---|
| V1 | Reporter's exact case fails IMP010 | `ailang run --quiet --package-dir . --entry main eval/rx/p/b.ail` on the c.ail/b.ail pair above | `Error: IMP010: symbol 'W' not exported by 'pkg/eval/rx/p/c'` — reproduced |
| V2 | `= \| Idle` followed by `export type W` breaks | package variant, body `\| Idle` | IMP010 on W (V1) |
| V3 | `= \| Idle` followed by `func mk() -> int` cascades | single-file `t1.ail` + `ailang check` | `PAR_NO_PREFIX_PARSE at 3:11: unexpected token in expression: ->` + 2× PAR015 |
| V4 | `= \| Idle` followed by `pure func` silently mis-parses, check passes | single-file `t2.ail` + `ailang check` | `✓ No errors found!` — mk silently non-pure (the P0 justification) |
| V5 | `= \| Idle deriving (Eq)` works (workaround) | package variant | prints `2` |
| V6 | `= \| Idle \| Busy` works | package variant | prints `2` |
| V7 | `= Idle` (no pipe) parses as alias; ctor undefined | package variant, body `Idle` | `type error ...: undefined variable: Idle at c.ail:4:41` |
| V8 | `parseTypeDeclBody` has exactly one caller; no test pins leadingPipe today | `grep -rn "parseTypeDeclBody\|leadingPipe" internal/ --include=*.go` | caller = `parseTypeDeclaration` (same file); zero test hits |
| V9 | No std/examples program uses the broken form; sibling forms are exercised | `grep -rn "^\s*|" std/*.ail`, `grep -rEn "type [A-Z]\w* = [A-Z]\w*(\(|$)" std/*.ail` | no single-nullary leading-pipe form in std; single-ctor ADTs in std all use `deriving (Eq)` (stream.ail:45,48, regex.ail:22, audio.ail:33, process.ail:78); multi-variant leading pipes: dom.ail:24-27, env.ail:19-20, net.ail:16-17 |
| V10 | v0.8.1 fixed the with-fields sibling; its audit flagged this line | `design_docs/implemented/v0_8_1/m-parser-export-multiline-adt.md` (read) | "Nullary constructors (line 401-409 leadingPipe path): `p.nextToken()` after name — needs review"; with-fields path verified fixed today (`Regex(string)` no deriving + follower exports → `✓ No errors`) |
| V11 | Match on a nullary ctor works once parsed as an ADT | `ver/b.ail`: `match j { Idle => "idle" }` + `ailang check` | `✓ No errors found!` |
| V12 | `= \| Idle` as last decl is fine today (bug needs a follower) | `ver/c.ail` + `ailang check` | `✓ No errors found!` |
| V13 | Leading-pipe with-fields single ctor (`\| Regex(string)`) works today | `ver/e.ail` + `ailang check` | `✓ No errors found!` |
| V14 | IMP010 producers are the linker and loader; hint shared | `grep -rn "IMP010" internal/ --include=*.go` | `internal/link/module_linker.go:124` (newIMP010), `internal/importhint/importhint.go` (shared suffix) |
| V15 | `parseDeriving` leaves cursor AT `)` (why the workaround works) | read `internal/parser/parser_type.go:404-467` | ends `expectPeek(lexer.RPAREN)` → cursor AT `)` |
| V16 | Main loop advances one token between decls (cursor convention) | read `internal/parser/parser_file.go:70-81` | `for !EOF { parseTopLevelDecl(); if !EOF { nextToken() } }`; convention stated in `parser_file.go:399` and `parser_type_decl.go:152-156` comments |
| V17 | Follower dispatched on second token explains IMP010 (not a loader bug) | read `internal/parser/parser_decl.go` `parseTopLevelDecl` switch | `case lexer.TYPE: parseTypeDeclaration(false)` — exported=false |
| V18 | Current version / target folder | `cat std/VERSION` = v0.51.0; changelogs latest `## [v0.51.0] - 2026-10-01` | target folder `v0_51_1` per create-script suggestion |

## Related Documents

- [m-parser-export-multiline-adt](../../implemented/v0_8_1/m-parser-export-multiline-adt.md) —
  the v0.8.1 fix for the with-fields sibling of this exact bug; its systemic audit flagged the
  nullary path fixed here. **This doc closes that audit item.**
- [m-gap2-lambda-arity-path-dependent-bug](../../implemented/v0_7_0/m-gap2-lambda-arity-path-dependent-bug.md) —
  prior parser position-dependent bug, cited by the v0.8.1 doc.
- Duplicate gate: `ailang docs search` (SimHash, planned + implemented, queries: "leading pipe
  ADT", "type declaration parser cursor", "export type constructor", "single constructor",
  "parser cursor convention") — no doc ≥ 0.45 on this topic; no duplicate.

## References

- AGENTS.md parser note: "Lexer skips newlines; do not expect NEWLINE tokens" — why the
  swallowed token can live on the next line.
- `internal/parser/parser_file.go:70-81` — declaration loop and the one-token advance.
- `internal/parser/parser_type_decl.go:84-91,162-164,168-190` — leading pipe, guarded
  with-fields advance (the pattern reused), buggy branch, alias disambiguation.
- Reporter: v0.50.0 release feedback, package `rx/p`, coordinator task `task-c0dad6ad`.

## Future Work

- **Alias-vs-ADT DX hint**: when `match` references an identifier that names a type-alias
  target (`type J = Idle` then `match x { Idle => ... }`), emit a suggestion
  ("did you mean `type J = | Idle`?"). Typechecker change; needs its own conflict-surface
  analysis. Deferred (human at review).
- **Cursor-convention invariant test**: a generative two-decl parse test over all decl heads,
  asserting the second decl's flags — would have caught both this bug and the v0.8.1 sibling
  at authoring time. Worth doing if the class recurs a third time.