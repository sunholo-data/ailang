# M-PROMPT-FOOTGUNS-TO-DIAGNOSTICS: Route Prompt-Teaching Footguns to Compiler Diagnostics

> **⚠️ PARKED — needs-human-review (mission iteration 47, 2026-07-17).** Authored + revised by the
> Fable designer (`claude:claude-fable-5`), reality-check-grounded at HEAD. Quorum (gpt5-6-sol +
> gemini-3-1-pro + controller) ran the full bounded round: **author → reject → revise → re-quorum →
> reject**. Per the one-revision Gate-2 bound, it parks here. **Part A (PRIMARY: MOD002/
> PAR_MODULE_PLACEMENT module diagnostics) and Part B (ghost-close of split-list-operations with a
> CI-gated guard) were UNANIMOUSLY ACCEPTED by both reviewers across both rounds.** Two narrow
> blocking objections remain, both with named fixes:
> 1. **gpt5-6-sol (on Part C / Phase 3 SECONDARY only):** the primitive-detection premise (matching
>    TCon names `string/int/float/bool`) is unverified — could misfire on a user ADT/alias with a
>    primitive-like name. The reviewer's own remedy offers "**defer Phase 3**". Phase 3 is already
>    severable (2% frequency, de-scope trigger built in).
> 2. **gemini-3-1-pro (on Part A step 4):** error-recovery should **set** `seenModule`/`firstModulePos`
>    from the first late module so a file with two late modules emits `PAR_MODULE_PLACEMENT` (1st) +
>    `MOD002` (2nd, the genuine duplicate) — a one-line design fix + golden-test expectation change.
> **RECOMMENDED UNBLOCK (human ratify):** drop Phase 3 to an extension-lane backlog doc
> (`m-diag-primitive-field-suggestions.md`, already stubbed as a Phase-3 task), apply gemini's
> `seenModule`-on-recovery fix to Part A, then ship the accepted Part A + Part B (a clean ~1.25-day
> sprint that kills the 10%-frequency multi-module footgun). Mission iteration 47 report: GH #399.

**Status**: **RATIFIED by Mark 2026-07-18** ("ratify") — the PARK-NOTE's recommendation is adopted
verbatim: drop Phase 3 → extension backlog, apply gemini's `seenModule`-on-recovery fix, ship the
unanimously-accepted Part A (MOD002/PAR_MODULE_PLACEMENT module diagnostics) + Part B (ghost-close
guard), ~1.25d. Route to sprint-planner; do NOT re-quorum (Parts A+B were unanimous both rounds).
**Target**: v0.30.0
**Priority**: P1 (High — clause-3 fleet-tier accessibility burn-down)
**Estimated**: 2.25 days (PRIMARY ~1d · ghost-close ~0.25d · SECONDARY ~0.5d · verification/docs ~0.5d)
**Dependencies**: None
**Verified against**: HEAD `v0.29.2-354-gc9fb89d32` on 2026-07-17 (all repros + code reads in the
[Premise Verification Log](#premise-verification-log))
**Revision**: quorum round 2 (2026-07-17) — Part C reworked to a **generic, library-agnostic core
diagnostic** per the frozen-core objection (no stdlib symbol catalog in `internal/types`; see
[Frozen-Core Routing](#frozen-core-routing-part-c) and verification rows 20–24); gemini's two
Phase-1 catches folded in (rows 20, and the error-recovery state rule in Part A step 4)

## Supersedes

This doc consolidates and supersedes three stale v0_29_0 prompt-teaching docs (all targeted the
ancient v0.24.0; all evidence pre-dates HEAD). The controller archives them on landing — this doc
does not modify them.

| Superseded doc | Reality-check at HEAD (2026-07-17) | Disposition here |
|---|---|---|
| `design_docs/planned/v0_29_0/m-prompt-split-list-operations.md` | **GHOST** — embedded prompt v0.16.2 already teaches everything the doc asks for (verified line numbers below) | Ghost-closed with a CI-gated runnable example as durable regression guard |
| `design_docs/planned/v0_29_0/m-prompt-single-file-module.md` | **REAL** — multi-`module` file still yields only an opaque `PAR_NO_PREFIX_PARSE` cascade (live repro below) | **PRIMARY**: coded, teaching parse diagnostic (reusing dormant `MOD002`) |
| `design_docs/planned/v0_29_0/m-prompt-log-file-analyzer-string-ops.md` | **REAL but MARGINAL** — `content.split("\n")` still yields raw `cannot unify type constructor string with *types.TRecordOpen` (live repro below); doc's own 2026-06-03 correction: dot-notation ≈ 2% of recent failures | **SECONDARY**: primitive-field-access teaching diagnostic at the existing unifier gate; explicit scope recommendation below |

**Strategic frame (decisive constraint):** the embedded teaching prompt is **2535 lines** (verified);
the mission's clause-3 target is ≤1500 (m-eval-slim-prompt-self-discovery / R3.1). Per the ratified
m-diagnostic-coverage strategy, prompt deletion stays gated on replacement diagnostics landing. So
this doc adds **zero prompt lines** and routes both real footguns to the **diagnostic lane** —
coded, directional, fix-carrying errors that fire at the exact failure site regardless of model or
prompt, following the established conventions: `MOD014` (m-module-less-run-fail-loud),
`TC_ARITY_001` (`internal/types/errors.go:344-346`), `PAR_IMPORT_PLACEMENT` (#325), and the
`swapTraps` registry (`internal/pipeline/warn_split_args.go:43-45`).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Diagnostics only; no runtime semantics change |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Errors fire earlier (parse/unify time) with a bounded, local check |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Coded, directional, fix-carrying errors are exactly what a self-repairing model needs; enables future prompt-line deletion (token-cost reduction) |
| A8: Minimal Syntax | +1 | Zero new syntax; converts existing failure into teaching failure |
| A9: Cost Visibility | 0 | N/A |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | +1 | Replaces a 2-error opaque cascade with one coded structured error |
| A12: System Boundary | 0 | N/A |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Optimizes for machine self-repair, not human convenience

## Problem Statement

Fleet-tier models (the clause-3 accessibility cohort) burn benchmark cells on two footguns that the
compiler currently answers with opaque, rule-free errors, and the historical answer — add prompt
teaching — is the wrong direction while the prompt is 1035 lines over its diet target.

**Footgun 1 — multiple `module` declarations per file (PRIMARY).**
Models habitually simulate Python/JS/Go multi-file layouts inside the single allowed file. Live
repro at HEAD (two top-level `module` declarations):

```
PAR_NO_PREFIX_PARSE at file.ail:5:1: unexpected token in expression: module
Suggestion: This token cannot start an expression
PAR_UNEXPECTED_TOKEN at file.ail:5:12: expected test name (string)
Suggestion: Add test name: test "my test" {...}
```

Neither error states the one-module-per-file rule, names a fix, or even acknowledges a module
declaration was seen. The second error is pure cascade noise. The same opaque cascade fires for a
*misplaced* (non-first) single `module` declaration (verified separately).
Historical frequency: 10% of Apr–Jun 2026 compile failures, 2nd-highest cause (stale doc's
RECENT-VERIFIED banner; mechanism — not frequency — re-verified at HEAD).

**Footgun 2 — Python-style dot-notation string methods (SECONDARY).**
`content.split("\n")` parses as record field access; live repro at HEAD:

```
Error: type error in <module> (decl 0): type unification failed at
[field access at file.ail:5:22]: cannot unify type constructor string with *types.TRecordOpen
```

True but unhelpful: no hint that AILANG uses `split(content, "\n")` function-call form.
Historical frequency: ~2% of recent failures (the stale doc's own 2026-06-03 correction).

**Non-footgun — `split` returns `[string]` (GHOST).**
The embedded prompt (v0.16.2, `ailang prompt --source=embedded`) already
teaches everything m-prompt-split-list-operations asked for, verified at HEAD:
- line 704: `` `split(s, delim) -> [string]` - Split string by delimiter `` (return type inline)
- line 715: `splitAny(s, delims: [string]) -> [string]`
- lines 1071–1072: `foldSlices` / `mapSlicesJoin` table rows (`split → map → join` in O(n))
- lines ~1073–1085: worked example block using `foldSlices(text, "\n", ...)` and
  `mapSlicesJoin(csv, ",", \field. toUpper(field))`

The doc's ask is fully satisfied; what's missing is only a **durable regression guard** so a future
prompt-diet pass can't silently delete the pipeline teaching before the (future) replacement
diagnostic exists: the existing `examples/runnable/string_split.ail` exercises `split` + recursion
only — **no `map`, no `join`** (read at HEAD) — so it does not guard the pipeline.

**Impact:** every fleet-tier model on module-layout or string-processing benchmarks; hard compile
failures with failed self-repair loops (models re-emit the same shape because the error never
states the rule).

## Goals

**Primary Goal:** Convert the two verified real footguns from opaque errors into coded,
directional, fix-carrying compiler diagnostics — adding zero prompt lines — and close the ghost
with a CI-gated guard.

**Success Metrics:**
- Two-module file produces exactly one `MOD002` error stating the one-module-per-file rule with a
  concrete fix, and no `PAR_NO_PREFIX_PARSE`/`PAR_UNEXPECTED_TOKEN` cascade for that token
- Misplaced (non-first) module declaration produces one `PAR_MODULE_PLACEMENT` error, same shape
- `content.split("\n")` produces a coded, generic teaching error (`TC_PRIMITIVE_FIELD_ACCESS_001`)
  stating that primitives have no fields and AILANG uses functions rather than methods — **no
  stdlib names embedded in `internal/types`** (symbol-specific suggestions are extension-lane
  Future Work)
- `examples/runnable/split_map_join.ail` green in `make verify-examples` (CI gate)
- Prompt line count unchanged (2535 → 2535); the corresponding prompt-deletion candidates are
  *recorded* for the gated diet pass (not deleted here)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Reuse dormant `MOD002` (not allocate `MOD015`) for duplicate-module | `MOD002` is already defined (`internal/errors/codes.go:67-68`), registered (`codes.go:267`, `codes_test.go:114`), and published in `dist/error_codes.json:161` as exactly "multiple module declarations in single file" — but has **zero emission sites** (verified). Reuse keeps the public registry truthful; a new code would orphan it | compiler (this doc) | design | low |
| Fire at parse time in `parseTopLevelDecl` (not pipeline) | Mirrors the proven `PAR_IMPORT_PLACEMENT` pattern (#325) at `internal/parser/parser_decl.go:336`; kills the cascade at its source; pipeline never sees the file (parse already fails today) | compiler (this doc) | design | low |
| Two variants: duplicate → `MOD002`, misplaced-first → `PAR_MODULE_PLACEMENT` | A late-but-only module is *not* "multiple modules"; conflating them makes the registry lie. Placement variant mirrors `PAR_IMPORT_PLACEMENT` naming | compiler (this doc) | design | low |
| SECONDARY fires in the unifier as a sibling of the tagged-union gate | The exact hook exists at `unification_core.go:343-346` (+ mirror `unification_records.go:404-407`), proven by M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (v0.20.0). This refutes the stale doc's "cross-cutting complexity" concern — see scope recommendation | compiler (this doc) | design | low |
| SECONDARY is **GENERIC-ONLY** — no stdlib symbol catalog in `internal/types` | Frozen-core routing (quorum round 2): a std/string name→call-form table inside the unifier couples the type-system core to library policy. The unifier emits only `TC_PRIMITIVE_FIELD_ACCESS_001` (receiver primitive + field + path); symbol-specific suggestions need a pipeline/extension enrichment hook that **does not exist at HEAD** (verified: rows 21–24) — deferred to an extension-owned backlog doc rather than inventing a new core hook this sprint | compiler (this doc, per PROGRAM.md route-to-extension bias) | design | low |
| Include SECONDARY in this sprint (severable Phase 3) | See [Scope Recommendation](#scope-recommendation-dot-notation-secondary) | human (ratify at plan time) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Error code for duplicate module: **`MOD002`** (verified unallocated-in-emission, published in registry)
- [x] Error code for misplaced module: **`PAR_MODULE_PLACEMENT`** (verified no existing use: string code, `NewSuggestionError` pattern)
- [x] Error code for primitive field access: **`TC_PRIMITIVE_FIELD_ACCESS_001`** (grep-verified unallocated, row 13); generic-only per frozen-core routing
- [ ] SECONDARY in-sprint vs backlog — recommendation is IN (severable); human ratifies at sprint planning

## Solution Design

### Part A (PRIMARY): teaching diagnostic for duplicate/misplaced `module` declarations

**Fire site.** `internal/parser/parser_decl.go` `parseTopLevelDecl` (line 209): add
`case lexer.MODULE:` → `return p.reportMisplacedModule()`. Today a `module` token at declaration
level has **no case** and falls through to expression parsing → `PAR_NO_PREFIX_PARSE` (verified:
the only parser consumer of `lexer.MODULE` is the file-start check at `parser_file.go:48`). This is
the exact shape of the `case lexer.IMPORT:` precedent at `parser_decl.go:336`.

**Mechanism** (new `reportMisplacedModule` in `parser_file.go`, mirroring
`reportMisplacedImport` at `parser_file.go:372-398`):

1. Parser gains a `seenModule bool` + `firstModulePos` / `firstModulePath`, set **ONLY** by
   `ParseFile` when it consumes the valid leading module at `parser_file.go:48-51`. The leading
   module already routes through the standalone `parseModuleDecl()` helper
   (`parser_file.go:89-…`, called at `:49` — verified at HEAD, row 20; no extraction needed), which
   `reportMisplacedModule` reuses for consume-and-continue.
2. If `seenModule` → **duplicate**, emit `errors.MOD002` via `NewSuggestionError`:
   ```
   MOD002 at file.ail:5:1: duplicate module declaration 'benchmark/string_utils' —
   AILANG requires exactly one module declaration per file
   (first module 'benchmark/math_utils' declared at 1:1)
   Suggestion: keep the single module declaration at the top and define all
     functions inside it (module-per-file, like Go packages — not Python modules)
   Suggestion: to model multiple namespaces, split into separate .ail files, one
     module declaration each
   ```
3. Else → **misplaced** (module exists but isn't first; verified repro: same opaque cascade today),
   emit `PAR_MODULE_PLACEMENT`:
   ```
   PAR_MODULE_PLACEMENT at file.ail:3:1: the module declaration must be the first
   declaration in the file
   Suggestion: move 'module test/late' above the imports on line 1
   ```
4. Consume-and-continue: call `parseModuleDecl()`, truncate any errors it appended (the
   `errCountBefore` pattern from `reportMisplacedImport`, `parser_file.go:391-395`), return `nil`.
   `parseModuleDecl` leaves the cursor AT its last token (parser convention), so the `ParseFile`
   loop's `nextToken()` advances correctly. **The error-recovery path MUST NOT set
   `seenModule`/`firstModulePos`/`firstModulePath`** — only the valid leading-module consumption in
   `ParseFile` (step 1) sets them. Otherwise, in a module-less file with two late `module`
   declarations, the first recovery would flip `seenModule` and the second would emit a `MOD002`
   referencing a non-existent "first module". Golden test locks this in: module-less file with TWO
   late modules → first late = `PAR_MODULE_PLACEMENT`, second late = **also**
   `PAR_MODULE_PLACEMENT` (never `MOD002`). Exactly **one placement error per offending
   declaration** — the documented m-diagnostic-coverage design choice (`parser_file.go:369-371`):
   models read per-line compiler output, so each offending line is pointed at individually rather
   than one collapsed "found N" error. (The message text still states the "exactly one" rule, which
   is the teaching payload the controller asked for.)

**Note on subsequent decls:** functions following a swallowed duplicate `module` continue parsing
into the single real module. They may produce follow-on errors (e.g. duplicate function names) —
acceptable: the first, coded error states the rule, and the cascade for the `module` token itself
is gone.

### Conflict Surface (PRIMARY — parser change)

1. **Position extended:** the top-level declaration dispatch (`parseTopLevelDecl` switch). We add a
   case for a token that today always errors there.
2. **Other valid constructs in that position:** `@`-annotations, `export`, `extern`, `pure`,
   `func`, `type`, `import`, `class`, `instance`, `test`, `property`, `IDENT`/expressions — all
   dispatch on disjoint token types; a `MODULE` case cannot shadow any of them.
3. **Disambiguation:** single-token dispatch. `module` is a hard keyword
   (`internal/lexer/token.go:279`), so it can never be an identifier, function name, or record
   label; there is **no valid program** with a `MODULE` token at declaration level (beyond the
   file-start position consumed at `parser_file.go:48`). False-positive risk: none — we convert an
   unconditional existing error into a better one.
4. **Positions NOT touched:** a `module` token *inside* a function body or block still routes
   through expression parsing → unchanged `PAR_NO_PREFIX_PARSE`. REPL/bare-expression input: files
   with a leading `module` consume it at `parser_file.go:48` exactly as today; module-less files
   never hit the new case unless they actually contain a late `module` token (which errors today
   anyway).
5. **Loader interaction:** none beyond error text. Parse errors already fail loading (verified
   repro: `module loading error: … parse errors in …`); the new diagnostic travels the same
   channel. The file never reaches pipeline-stage module validation, so no interaction with
   `MOD010` (path mismatch), `MOD011` (cross-file collision — `pipeline_module.go:498`; NOT
   per-file duplicates, despite its stale `codes.go:93` comment, which this sprint should fix in
   passing), or `MOD014` (module-less funcs).
6. **Programs that MUST still work (regression fixtures — all verified to exist):**
   - `examples/runnable/string_split.ail` — normal leading module (read at HEAD)
   - bare-expression eval: guarded by `internal/pipeline/module_less_test.go` (Decls-only files
     must not trip module gates; read via grep at HEAD)
   - module-less file with funcs → still `MOD014`: `internal/diag/footgun_fixtures_test.go:206`
   - misplaced-import behavior unchanged: `internal/parser/import_placement_test.go`
7. **Deliberate change:** for a decl-level `module` token, `PAR_NO_PREFIX_PARSE` +
   `PAR_UNEXPECTED_TOKEN` cascade → single `MOD002`/`PAR_MODULE_PLACEMENT`. No existing parser
   test asserts the old cascade for module tokens (grep at HEAD: `PAR_NO_PREFIX` appears only in
   `expr_test.go` and `import_placement_test.go`, neither on `module` input).

### Part B (GHOST-CLOSE): split-list-operations, with durable guard

The doc's ask is already satisfied by the embedded prompt (evidence in Problem Statement +
Verification Log). Close it as a ghost with a **runnable, CI-gated guard** — never bare
bookkeeping:

**New** `examples/runnable/split_map_join.ail` — exercises BOTH pipeline forms. This exact program
(modulo module path) **compiles clean at HEAD** (`ailang check` transcript in the log; note the
live-verified argument orders: `join(delim, list)`, `map(f, list)`):

```ailang
module examples/runnable/split_map_join

import std/string (split, join, toUpper, mapSlicesJoin)
import std/list (map)

export func main() -> () ! {IO} {
  let csv = "alice,bob,carol";
  let names = split(csv, ",");            -- split(s, delim) -> [string]
  let upper = map(\n. toUpper(n), names); -- transform the list
  let joined = join(",", upper);          -- join(delim, list) -> string
  println(joined);
  -- O(n) fused form, no intermediate list:
  println(mapSlicesJoin(csv, ",", \f. toUpper(f)))
}
```

**CI gating mechanism (verified):** `make verify-examples` (`make/examples.mk:18-28`) runs every
example in `examples/runnable/` AND `go run ./scripts/validate_manifest.go --ci`; it is a real CI
gate (`.github/workflows/ci.yml:205-209`, exits non-zero on any red example or manifest drift).
The executor must add the manifest entry **and update the hand-maintained statistics block** —
forgetting the statistics block is the classic way this gate goes red (verify-examples red is
usually manifest drift, not a type regression).

### Part C (SECONDARY): primitive-field-access teaching diagnostic (dot-notation)

**Fire sites (both verified at HEAD).** The generic error comes from the unifier's `TCon` case
fallthrough:
- `internal/types/unification_core.go:343-346` — `TCon` vs `*TRecordOpen`: today, if the TCon is a
  tagged union → prescriptive `makeUnificationTaggedUnionError` (the v0.20.0 precedent); otherwise
  → raw `cannot unify type constructor %s with %T`. **Our repro takes exactly this fallthrough.**
- `internal/types/unification_records.go:397-407` — the swapped-order mirror (`TRecordOpen` vs T),
  which already mirrors the tagged-union gate; extend identically.

**Mechanism (GENERIC-ONLY — revised per quorum round 2).** Add a sibling branch: when the TCon is
a primitive (`string`, `int`, `float`, `bool`) and the other side is `*TRecordOpen`, the user
wrote `expr.field` on a primitive. The `TRecordOpen.Fields` map carries the accessed field name
(this is how `makeUnificationTaggedUnionError` already recovers it,
`tagged_union_predicate.go:182-188`), and the constraint `Path` carries the
`field access at <pos>` context (verified in the repro output). Emit a coded, **library-agnostic**
teaching error, `TC_PRIMITIVE_FIELD_ACCESS_001` (grep-verified unallocated), styled on
`TC_ARITY_001` (code + Suggestion embedded inline in the error text,
`internal/types/errors.go:344-374`):

```
TC_PRIMITIVE_FIELD_ACCESS_001: type 'string' has no fields — AILANG uses
functions rather than methods; tried to access '.split' at
[field access at file.ail:5:22]
Suggestion: call a function with the value as an argument instead of
  accessing '.split' on it
```

- The error carries exactly three data points, all recovered from the types/constraint already at
  hand: receiver primitive name, requested field name, source path. **No stdlib names, no
  call-form templates, no import advice live in `internal/types`** — those are library policy and
  belong to the extension lane (see [Frozen-Core Routing](#frozen-core-routing-part-c)). This is
  still the **systemic** fix for the confusion *class*: `x.length`, `n.toString`, `s.split` all
  get the same rule-stating, coded diagnostic.
- The message wording ("AILANG uses functions rather than methods") is language semantics, not
  library policy — the same nature of teaching the tagged-union error already embeds in
  `internal/types` (`buildTufaMessage`, v0.20.0 precedent).
- The receiver's *expression text* is not available in the unifier (only types reach it) — same
  trade-off the tagged-union error already accepts.

### Frozen-Core Routing (Part C)

Per PROGRAM.md ("default bias: if it can be an extension, it is an extension — not a core
change"), the original draft's `methodTraps` table (std/string symbol → call-form + import
suggestion, hosted in `internal/types`) is **withdrawn**: it would couple the type-system core to
library policy and grow with the stdlib. The quorum-directed split is:

- **Core (this sprint):** the unifier emits only the generic coded diagnostic above — a
  structural fact about the type system (primitive vs. open record), zero library knowledge.
- **Extension (deferred):** symbol-specific repairs (`.split` → `split(receiver, ...)` +
  `import std/string (split)`) require a diagnostic-enrichment point OUTSIDE `internal/types`.
  **Live audit at HEAD found no such hook exists today** (verification rows 21–24):
  - `swapTraps`/`DetectArgOrderWarnings` (`warn_split_args.go:45,129`) is a **non-blocking
    warning** pass walking successfully-elaborated Core — called at `pipeline_single.go:198`
    (pre-typecheck) and `pipeline_module.go:388` (post-compile of ALL units). A unification
    failure aborts compilation before either could decorate anything, and the pass never sees
    error values at all.
  - Type errors cross the types→pipeline boundary as **unstructured `fmt.Errorf` chains**
    (`unification_core.go:316,346`; even the tagged-union error is `fmt.Errorf` +
    `[record_access_on_tagged_union]` string tag, `tagged_union_predicate.go:194`), wrapped
    verbatim at `pipeline_module_compile.go:213` and returned immediately. **Zero
    `errors.As`/`errors.Is` on type errors anywhere in `internal/pipeline`** (grep) — there is no
    decoration seam to reuse.
  - `internal/diag` is a footgun coverage **table + CI fixture contract** (`doc.go`), not a
    runtime enrichment mechanism.
  - The #327 `SetModuleFuncNames`/`localResolutionHint` pattern (`pipeline_module_compile.go:193-205`)
    injects *program facts* into the checker pre-inference; reusing it for a stdlib suggestion
    catalog would still require a NEW setter + consumption logic in `internal/types` — new core
    surface.

  Building a clean hook (structured error type + `errors.As` decorator in both pipeline paths, or
  a data-injection setter) is a **new extension-point design of its own**, out of scope for a
  ≤3-day diagnostics sprint. It is deferred to a one-item extension-lane backlog doc
  (`m-diag-primitive-field-suggestions.md`, Future Work) that records both candidate routes; until
  it lands, models get the generic rule-stating diagnostic — which is the teaching payload — and
  the specific call-form remains taught by the prompt (those prompt lines stay until the
  enrichment extension exists).

**Conflict Surface (SECONDARY — types change):**
1. Position extended: the two unifier fallthroughs above; both currently produce generic errors
   for primitive-vs-open-record, so no successful program changes outcome — only error text.
2. Valid constructs sharing the shape: type *aliases* to records unify TCon-vs-`TRecord` via
   `expandAlias` (`unification_core.go:322-331`) — that path runs BEFORE our branch and is
   untouched; user ADTs hit the tagged-union gate first (`:343-345`). Our branch matches only
   built-in primitive TCon names, which cannot be user record aliases.
3. Must-still-work fixtures: tagged-union error tests (`internal/types/tagged_union_predicate.go`
   tests + `internal/pipeline/tagged_union_field_access_test.go`), record field access on real
   records, alias-to-record access.
4. Deliberate change: error text/code for `primitive.field` only.
5. Residual risk (why this is "med" change cost): a `TVar`-receiver that *later* resolves to
   string surfaces through the same constraint solve — covered, since the gate fires at
   unification time, after substitution reaches the constraint. The deferred-verification pass
   (`VerifyTaggedUnionFieldAccesses`) is NOT needed here and is not touched.

### Scope Recommendation (dot-notation SECONDARY)

**Recommendation: INCLUDE in this sprint, as a severable Phase 3.** Explicit reasoning, no
hand-wave:

- The stale doc's "cross-cutting type-site complexity" concern is **refuted by code-reading at
  HEAD**: the exact hook (TCon-vs-TRecordOpen fallthrough) exists at two known lines, with a
  shipped sibling precedent (tagged-union gate, v0.20.0) that already solved the hard parts
  (recovering the field name from `TRecordOpen`, position via constraint `Path`, the swapped-order
  mirror). Generic-only shrinks the marginal cost further: an error builder + two 3-line branches
  + golden tests, **no symbol table**: **~0.5 day**, fitting the ≤3-day envelope at 2.25 days
  total.
- The 2% frequency alone would argue for backlog, but the mission's ratified sequencing ("prompt
  deletion gated on replacement diagnostics") means the dot-notation *rule* prompt lines stay
  frozen until a diagnostic states the rule. The generic diagnostic states it; only the
  symbol-specific call-form lines remain gated on the deferred enrichment extension.
- **Severability / de-scope trigger:** Phase 3 has zero coupling to Phases 1–2. If the day-2
  checkpoint shows Phase 1 incomplete, the executor drops Phase 3 and the evaluator files it as a
  one-item backlog doc (`m-diag-primitive-field-access.md`) citing this section verbatim — no
  redesign needed.

## Implementation Plan

**Phase 1: PRIMARY — module diagnostics** (~1 day)
- [ ] `Parser` fields `seenModule`/`firstModulePos`/`firstModulePath`; set **only** in
      `ParseFile`'s leading-module branch (`parser_file.go:48-51`, which calls the existing
      `parseModuleDecl()` helper at `:49` — verified present, no extraction needed); the
      `reportMisplacedModule` recovery path must NOT set them
- [ ] `reportMisplacedModule()` in `parser_file.go` (mirror `reportMisplacedImport`): MOD002
      duplicate variant + PAR_MODULE_PLACEMENT misplaced variant, consume-and-truncate
- [ ] `case lexer.MODULE:` in `parseTopLevelDecl` (`parser_decl.go`, beside the IMPORT case)
- [ ] Fix stale `MOD011` comment at `codes.go:93` (says "per file"; actual: cross-file collision)
- [ ] Tests: `internal/parser/module_placement_test.go` styled on `import_placement_test.go`
      (duplicate → MOD002 + no cascade; misplaced → PAR_MODULE_PLACEMENT; 3 modules → 2 errors;
      **module-less file with TWO late modules → both PAR_MODULE_PLACEMENT, second never MOD002**;
      leading module unaffected); run `make test-imports` + parser suite
- [ ] Footgun contract test in `internal/diag/footgun_fixtures_test.go` (MOD014 pattern)

**Phase 2: Ghost-close guard** (~0.25 day)
- [ ] Add `examples/runnable/split_map_join.ail` (verified-compiling program above)
- [ ] Manifest entry + statistics block update; `make verify-examples` green locally
- [ ] Record prompt-deletion candidates (lines 704/715/1071-1085 stay until the R3.1 diet pass)

**Phase 3 (severable): SECONDARY — primitive field access, GENERIC-ONLY** (~0.5 day)
- [ ] `TC_PRIMITIVE_FIELD_ACCESS_001` + `makePrimitiveFieldAccessError` in `internal/types`
      (styled on `tagged_union_predicate.go` + `TC_ARITY_001` conventions) — receiver primitive +
      field + path only; **no symbol table, no stdlib names, no import advice**
- [ ] Branches at `unification_core.go:346` fallthrough + `unification_records.go:407` mirror
- [ ] Golden tests styled on `internal/types/arity_diagnostic_test.go`; tagged-union tests stay green
- [ ] File the extension-lane backlog doc stub `m-diag-primitive-field-suggestions.md` (one item:
      symbol→call-form enrichment OUTSIDE `internal/types`; candidate routes recorded in
      [Frozen-Core Routing](#frozen-core-routing-part-c))

**Verification & docs** (~0.5 day)
- [ ] `make ci` locally; `ailang check` on both repro files shows new diagnostics
- [ ] Regenerate `dist/error_codes.json` (`tools/gen-error-codes`) — MOD002 gains a live emitter;
      new TC code registered
- [ ] CHANGELOG.md; LIMITATIONS.md if applicable

### Files to Modify/Create

**New files:**
- `examples/runnable/split_map_join.ail` (~15 LOC)
- `internal/parser/module_placement_test.go` (~120 LOC)
- `internal/types/primitive_field_access.go` + `_test.go` (Phase 3, ~120 LOC — generic-only, no
  symbol table)
- `design_docs/planned/v0_30_0/m-diag-primitive-field-suggestions.md` (Phase 3 backlog stub, ~40
  lines)

**Modified files:**
- `internal/parser/parser_decl.go` (+3 LOC), `internal/parser/parser_file.go` (+50 LOC)
- `internal/parser/parser.go` (Parser struct fields, +3 LOC)
- `internal/errors/codes.go` (MOD011 comment fix; TC code registration)
- `internal/types/unification_core.go` / `unification_records.go` (+4 LOC each, Phase 3)
- `internal/diag/footgun_fixtures_test.go` (+40 LOC)
- `examples/manifest.json` (+ entry + statistics)

## Success Criteria

- [ ] Two-module repro file: exactly one `MOD002` naming both module paths + first-decl position;
      zero `PAR_NO_PREFIX_PARSE` for the module token (acceptance: golden test + manual repro)
- [ ] Misplaced-module repro: one `PAR_MODULE_PLACEMENT` (acceptance: golden test)
- [ ] `content.split("\n")` repro: `TC_PRIMITIVE_FIELD_ACCESS_001` stating primitive + field +
      "functions rather than methods" rule; **grep of `internal/types` shows zero std/string
      symbol names or import-advice strings introduced** (Phase 3)
- [ ] `make verify-examples` green including `split_map_join.ail`
- [ ] All existing parser/types/pipeline tests green (`make test`, `make test-imports`)
- [ ] `dist/error_codes.json` regenerated; CHANGELOG updated

## Testing Strategy

**Unit:** module placement golden tests (duplicate/misplaced/triple/leading-only/**two-late-modules
state-isolation**); primitive-field-access golden tests (string + non-string primitives, arbitrary
field names — one generic shape, no per-symbol cases); tagged-union + alias-expansion regression
tests untouched and green.
**Integration:** `make verify-examples`; `make test-imports`; both repro files via `ailang check`.
**Manual:** REPL smoke (`module foo` as first input), `ailang run` of a bare-expression file.

## Deferred Decisions

- Which enrichment route the extension-lane suggestion catalog takes (structured error +
  `errors.As` decorator in the pipeline vs. #327-style data-injection setter) — decided in
  `m-diag-primitive-field-suggestions.md`, NOT here; both candidates recorded in
  [Frozen-Core Routing](#frozen-core-routing-part-c)
- Whether the misplaced-import and misplaced-module helpers share a common consume-and-truncate
  utility — agent may refactor if it stays within file-size targets
- Wording micro-tuning of Suggestion lines — agent decides; must keep code + rule + fix structure
  (and, for `TC_PRIMITIVE_FIELD_ACCESS_001`, must stay library-agnostic)

## Non-Goals

- **Any prompt additions** — decisive constraint; the prompt is 1035 lines over target
- **Prompt deletions** — gated on this landing plus an eval pass (m-diagnostic-coverage sequencing);
  candidates recorded, not removed
- Multi-module-per-file support (rejected by design: module = file is load-bearing for the loader)
- Dot-notation *syntax* support (method calls) — contradicts A8/minimal-syntax and the
  function-call teaching
- **Any stdlib symbol catalog or import advice inside `internal/types`** — frozen-core violation;
  symbol-specific repairs are extension-lane work (`m-diag-primitive-field-suggestions.md`)
- Building the pipeline-level diagnostic-enrichment hook itself — a separate extension-point
  design (candidate routes recorded, not chosen, here)
- Touching `MOD011` cross-file collision behavior (comment fix only)
- Archiving the three superseded docs (controller does this on landing)

## Timeline

Single sprint, ≤3 days: Day 1 = Phase 1; Day 2 AM = Phase 2 + Phase 1 hardening,
**day-2 checkpoint** (de-scope trigger for Phase 3); Day 2 PM = Phase 3 (generic-only) +
verification start; Day 3 AM = verification/docs buffer. **Total: 2.25 days.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Error-cascade suppression swallows a genuine error inside a malformed 2nd module decl | Low | Mirror the proven `errCountBefore` truncation exactly; golden test with malformed duplicate (`module a-b`) asserting one placement error |
| Manifest statistics drift turns verify-examples red | Med | Explicit task; known failure mode (hand-maintained statistics block in `examples/manifest.json`) |
| Phase 3 TCon name matching hits a user type named `string` | Low | Impossible: primitive names are reserved type constructors; alias expansion runs first (`unification_core.go:322-331`) |
| Phase 3 overruns | Low | Severable; day-2 checkpoint de-scope trigger defined above |

## Premise Verification Log

All commands run at HEAD `v0.29.2-354-gc9fb89d32` on 2026-07-17, binary rebuilt at HEAD.

| # | Claim / premise | Command | Result | Verdict |
|---|---|---|---|---|
| 1 | Prompt is 2535 lines | `ailang prompt --source=embedded \| wc -l` | `2535` | ✅ Confirmed |
| 2 | Prompt already teaches `split -> [string]` inline | `grep -n split` on prompt dump | line 704: `` `split(s, delim) -> [string]` ``; line 715 `splitAny`; line 632 std/string import list | ✅ Confirmed — ghost |
| 3 | Prompt already teaches split→map→join pipeline with worked example | `sed -n '1060,1090p'` on dump | lines 1071–1072 `foldSlices`/`mapSlicesJoin` table; worked block with `foldSlices(text, "\n", 0, \acc line. …)` and `mapSlicesJoin(csv, ",", \field. toUpper(field))` | ✅ Confirmed — ghost |
| 4 | Two-module file → opaque cascade, no rule teaching | `ailang check` on 2-module tmp file | `PAR_NO_PREFIX_PARSE …: unexpected token in expression: module` + `PAR_UNEXPECTED_TOKEN …: expected test name (string)`; exit 1 | ✅ Confirmed — PRIMARY real |
| 5 | Misplaced (non-first, single) module → same cascade | `ailang check` on import-then-module tmp file | identical 2-error cascade at 3:1/3:12 | ✅ Confirmed — placement variant needed |
| 6 | `content.split("\n")` → raw TCon/TRecordOpen error, no hint | `ailang check` on dot-notation tmp file | `type unification failed at [field access at …:5:22]: cannot unify type constructor string with *types.TRecordOpen`; exit 1 | ✅ Confirmed — SECONDARY real |
| 7 | MOD0xx allocation: next-free + MOD002 dormant | `grep -rhoE "MOD0[0-9]{2}" internal/ cmd/ \| sort -u`; `grep -rn MOD002\|MOD011` | MOD001–MOD014 defined; MOD002 = "multiple module declarations in single file" (`codes.go:67-68`, registry `:267`, `codes_test.go:114`, `dist/error_codes.json:161`) with **zero emission sites**; MOD011 emission = cross-**file** collision only (`pipeline_module.go:498,584`) — its `codes.go:93` "per file" comment is stale | ✅ Confirmed — reuse MOD002; refutes need for MOD015 |
| 8 | `module` at decl level has no parser case; only consumer is file start | `grep lexer.MODULE internal/parser/` + read `parser_decl.go:208-345`, `parser_file.go:13-86` | only `parser_file.go:48`; `parseTopLevelDecl` switch has no MODULE case → default → expression | ✅ Confirmed — fire site |
| 9 | Misplaced-import precedent exists to mirror | read `parser_file.go:361-398`, `parser_decl.go:336` | `reportMisplacedImport` + `PAR_IMPORT_PLACEMENT`, consume-and-truncate, per-line design choice documented | ✅ Confirmed |
| 10 | `module` is a hard keyword (no identifier conflicts) | `grep '"module"' internal/lexer/` | `token.go:279: "module": MODULE` | ✅ Confirmed — zero false-positive surface |
| 11 | SECONDARY fire sites + precedent exist in unifier | read `unification_core.go:305-346`, `unification_records.go:397-407`, `tagged_union_predicate.go:166-182` | TCon-vs-TRecordOpen fallthrough at `:346` (generic error); tagged-union prescriptive gate at `:343-345` recovers field name from `TRecordOpen.Fields`; swapped mirror at records `:404-407` | ✅ Confirmed — refutes "cross-cutting complexity" premise of stale doc |
| 12 | Diagnostic conventions to follow | read `types/errors.go:344-390`, `pipeline/warn_split_args.go:28-60`; grep MOD014 tests | `TC_ARITY_001` inline code+Suggestion; `swapTraps` extensible symbol-keyed table; MOD014 module-pipeline gate + footgun contract test (`internal/diag/footgun_fixtures_test.go:200-234`) | ✅ Confirmed |
| 13 | `TC_PRIMITIVE_FIELD_ACCESS_001` unallocated (quorum-round-2 code, replacing draft's `TC_METHOD_CALL_001`) | `grep -rn TC_PRIMITIVE_FIELD_ACCESS internal/ cmd/ dist/` | no hits | ✅ Confirmed — free |
| 14 | Guard example compiles at HEAD (incl. `join`/`map` arg orders) | `ailang check` on `split_map_join` tmp file (program in Part B) | `✓ No errors found!` exit 0; `join(",", list)` and `map(f, list)` orders live-verified | ✅ Confirmed |
| 15 | `string_split.ail` does NOT already guard the pipeline | read `examples/runnable/string_split.ail` | uses `split`+`chars`+recursion; no `map`, no `join` | ✅ Confirmed — new example not redundant |
| 16 | verify-examples is a real CI gate incl. manifest | read `make/examples.mk:5-34`, `.github/workflows/ci.yml:205-224` | runs examples/runnable + `validate_manifest.go --ci`; non-zero exit on red/drift | ✅ Confirmed |
| 17 | Regression fixtures exist; no test asserts old module cascade | `ls examples/runnable/string_split.ail`; grep tests | `module_less_test.go` (bare-expr guards), `footgun_fixtures_test.go:206` (MOD014), `import_placement_test.go` (asserts PAR_IMPORT_PLACEMENT, no cascade); `PAR_NO_PREFIX` in parser tests only in `expr_test.go`/`import_placement_test.go`, neither on module input | ✅ Confirmed |
| 18 | Frequency claims (10% / 2%) | not re-measured | cited from the superseded docs' own 2026-06-03 RECENT-VERIFIED banners (Apr–Jun 2026 segment); mechanisms re-verified at HEAD (rows 4–6), frequencies NOT re-measured — treat as historical | ⚠️ Cited-historical, flagged |
| 19 | Duplicate/coverage gate for THIS doc | `create_planned_doc.sh` dual search | top neural match 0.38 (`m-trace-feedback`) < 0.45 threshold; three superseded docs are the intended supersession targets, not duplicates | ✅ Proceed |
| 20 | (gemini catch 1) leading module is handled by a reusable helper, not inline | `grep -rn parseModuleDecl internal/parser/` + read `parser_file.go:43-101` | standalone `parseModuleDecl()` helper exists at `parser_file.go:89` (also carries the hyphen-in-path diagnostic); called from `ParseFile` at `:49` | ✅ Confirmed — reviewer's inline concern moot; no extraction task needed |
| 21 | (quorum round 2) enrichment-hook inventory: `swapTraps` API + emission timing | read `warn_split_args.go` (full); `grep -rn DetectArgOrderWarnings --include=*.go internal/ cmd/` | registry `warn_split_args.go:45`, walker `:129`; returns `[]elaborate.Warning` (non-blocking); called at `pipeline_single.go:198` (post-elaborate, PRE-typecheck) and `pipeline_module.go:388` (after ALL units compile). Walks successful Core only; never sees error values | ✅ Confirmed — cannot enrich a unification FAILURE (compile aborts first) |
| 22 | (quorum round 2) type errors cross types→pipeline as unstructured strings; no decoration seam | read `unification_core.go:300-360`, `tagged_union_predicate.go:176-196`, `pipeline_module_compile.go:207-237`; `grep -n "errors\.As\|errors\.Is" internal/pipeline/*.go` | unifier returns `fmt.Errorf` (`:316`, `:346`); tagged-union error is `fmt.Errorf` + `[record_access_on_tagged_union]` string tag (`:194`); pipeline wraps `type error in %s (decl %d): %w` and returns immediately (`pipeline_module_compile.go:213`); **zero** `errors.As`/`errors.Is` on type errors in `internal/pipeline` | ✅ Confirmed — NO existing post-unification enrichment hook at HEAD |
| 23 | (quorum round 2) `internal/diag` is not a runtime enrichment mechanism | `ls internal/diag/`; read `doc.go` | package = footgun coverage table (`footguns.md`) + CI fixture contract (`footgun_fixtures_test.go`); no runtime hook, no registry API | ✅ Confirmed |
| 24 | (quorum round 2) closest injection precedent: #327 `SetModuleFuncNames` | read `pipeline_module_compile.go:193-205` | pipeline injects module function names pre-inference; enrichment message logic (`localResolutionHint`) lives in `internal/types`; reusing the shape for a stdlib catalog would need a NEW types-side setter + formatter = new core surface | ✅ Confirmed — pattern exists but is not a reusable hook; supports GENERIC-ONLY decision (option b): new core hook rejected this sprint, enrichment deferred to extension-lane backlog doc |

## Related Documents

**Superseded (see table at top):** the three v0_29_0 prompt-teaching docs.

**Convention precedents (informing design):**
- `design_docs/planned/v0_29_0/m-diagnostic-coverage*` — ratified diagnostics-before-prompt-deletion strategy (PAR_IMPORT_PLACEMENT, #325)
- m-module-less-run-fail-loud (MOD014) + its reality-check corrections — the case study this doc's verification discipline follows
- m-dx-split-argument-warning (`swapTraps`) — extensible trap-table pattern; NOTE (quorum round
  2): it lives at the **pipeline** level (`internal/pipeline/warn_split_args.go`), which is
  precisely why it does NOT license a symbol catalog inside `internal/types` — it is the shape the
  deferred extension-lane suggestion catalog should mirror
- M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (v0.20.0) — the unifier prescriptive-error precedent Phase 3 extends
- m-eval-slim-prompt-self-discovery (R3.1) — the ≤1500-line prompt target this doc serves

**Auto-search results (neural, all < 0.45 — no overlap):**
- `design_docs/planned/v1_1_0/m-trace-feedback.md` (0.38), `design_docs/planned/v0_29_0/m-eval-regression-detector-contract.md` (0.33), `design_docs/planned/v1_1_0/m-oracle-adequacy.md` (0.31)

## Future Work

- The R3.1 prompt-diet pass deletes the now-diagnostic-covered teaching lines (gated on this doc's
  diagnostics landing + one eval pass confirming self-repair on the new errors)
- **`m-diag-primitive-field-suggestions.md` (extension lane):** symbol-specific enrichment of
  `TC_PRIMITIVE_FIELD_ACCESS_001` (`.split` → `split(receiver, ...)` + import advice), hosted
  OUTSIDE `internal/types`, catalog populated from verified stdlib metadata. Requires first
  designing the enrichment point — candidate routes (structured error + `errors.As` decorator in
  both pipeline paths; or #327-style data-injection setter) recorded in
  [Frozen-Core Routing](#frozen-core-routing-part-c). Extending to std/list names (`xs.map(f)` →
  `map(f, xs)`) belongs there too
- A/B eval: fleet-tier pass-rate on `multi_module_imports` and `log_file_analyzer` pre/post

---

**Document created**: 2026-07-17
**Last updated**: 2026-07-17 (quorum round 2 revision: Part C generic-only per frozen-core
objection; gemini Phase-1 catches folded in; verification rows 20–24 added)
