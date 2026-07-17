# M-CHECK-STRICT-FALLBACKS — Static detection of "Ok contains default-valued literal" anti-pattern

**Status**: UNPARKED — **ARCHITECTURE DECIDED by Mark 2026-07-17 ("go with 2"): the pass runs
AFTER name resolution** (resolved-callee identity), with a **curated known-empty-builder registry**
matched by resolved identity (std/json `jo` with empty args, etc. — never bare name-match, per
gpt5-6-sol's soundness warning), so `Ok(jo([]))` — the motivating incident — IS caught. Channel
decision (same directive): **WARNING in dev `ailang check`, HARD ERROR (exit 1) under
`check --package`** — fail loudly at the publish boundary, don't nag mid-development. Precedent
layer: the M-DX-SPLIT-ARG warning's post-resolution hook (`internal/pipeline/warn_split_args.go`).
Literal-empties (`Ok([])`/`Ok("")`/`Ok({})`) are caught by the same resolved-layer pass trivially.
Revised estimate ~2d. **Route to sprint-planner** (quorum already run twice; the decided
architecture answers both standing objections — do NOT re-quorum, record the decision as the
resolution). The earlier option-(a) analysis below is RETAINED as history; it is superseded.

## REBLOCK (iter 42 clean re-quorum — the real blocker)

After resolving the iter-41 "OPEN design decision" to option (a) (syntactic surface-AST pass) and
addressing gemini's Pattern C objection (see below), a re-quorum on a **rebuilt binary** (the iter-41
`#407` quorum-schema fix was not in the stale installed binary, so gpt5-6-sol had been silently
`unreachable`; rebuilding restored it) surfaced a **goal-contradicting** objection:

- **`gpt5-6-sol` (present, reject):** the motivating incident is `None => Ok(jo([]))`. `jo` is a
  **lowercase function call** (the `std/json` object builder), and the doc's own "does NOT flag"
  rules — plus the iter-42 Pattern C rule (uppercase head = constructor, lowercase = function, never
  flags) — mean `Ok(jo([]))` **would not emit `STRICT_FALLBACK_001`**. So the purely-syntactic pass
  **fails its own primary goal, examples, tests, and success criteria.** Catching it requires
  resolving that `jo` is the canonical JSON-empty-object builder (name resolution / known-symbol
  identity), which a surface-AST-only pass cannot do soundly (shadowing / qualification / import
  aliases). This **refutes the option-(a) resolution** and resurrects a resolved-layer (option (b))
  or hybrid architecture — the exact fork the doc must re-decide.
- **`gemini-3-1-pro` (response truncated → recorded `invalid`; recurring tooling issue this iter):**
  partial objection — the pass returns **warnings** but `ailang check --package` needs an **ERROR /
  exit-1** publish gate; the warning-channel wiring can't produce the required failure exit. (Also a
  real design point for the planner: the warning channel vs. an error/exit path.)

**Human decision needed (the architecture fork):** (1) run the pass AFTER name resolution to resolve
callee identity (`jo`, etc.) — contradicts "purely syntactic", closer to warn_split_args' Core hook;
OR (2) narrow the goal: drop the claim it catches `Ok(jo([]))` and only catch literal empties
(`Ok([])`/`Ok("")`/`Ok({})`/all-zero record literals), filing the `jo([])`-class under the interproc
follow-up; OR (3) a curated known-empty-builder list with resolved identity (gpt5-6-sol warns a bare
name-match is unsound). Plus: resolve the warning-vs-error/exit-1 channel for `--package`. This is a
genuine architecture decision, not a quick patch — hence the park (Standing rule 2: never force a
guardrail).

---

### (retained) iter-41→42 OPEN design decision analysis (option a) and Pattern C grounding

**Target**: v0.21.0
**Priority**: P1 — closes a class of silent-failure bugs that just bit production
**Estimated**: ~1 day (~300 LOC impl + ~200 LOC tests + parser whitelist + docs)
**Dependencies**: None (uses existing AST + annotation infrastructure)

## Premise Verification (added 2026-07-17, mission iteration 41 pick-time quorum)

This doc pre-dates the design-quorum gate; the pick-time quorum was run and did NOT converge in
its one bounded revision+re-quorum round, so the item is **PARKED needs-human-review**. The
verification below is a durable improvement kept regardless (it corrected a real premise error).
Verified against the source tree at `origin/dev` `b417d02c6`:

| Claim | Verified? | Evidence |
|-------|-----------|----------|
| `internal/parser/parser_decl.go::parseAnnotation` is a name-keyed annotation whitelist | ✅ TRUE | `parser_decl.go:15 func (p *Parser) parseAnnotation()`; cases `verify` (l.30), `route` (l.32), `mcp_name` (l.34), `raw` (l.36) — adding `case "allow_empty_ok":` is a pure additive switch case, as the doc claims |
| `internal/parser/route_attr_test.go` exists (parser-test home) | ✅ TRUE | `internal/parser/route_attr_test.go` present (13 KB) |
| The check pass lives in `internal/check/` | ❌ **FALSE — CORRECTED** | **No `internal/check/` package exists.** `ailang check` is implemented in `cmd/ailang/check.go` and runs `internal/pipeline`. Existing compile-time static passes live in **`internal/pipeline/`** (e.g. `warn_split_args.go`, the M-DX-SPLIT-ARG warning from iter-17) and `internal/elaborate/warnings.go`. |
| Post-parse surface-AST hooks exist in BOTH pipelines where a pass can read `FuncDecl`s AND append to `result.Warnings` | ✅ TRUE (verified iter 42) | `pipeline_single.go`: `result.Artifacts.AST = astFile` (~l.159) is set BEFORE elaboration (`ElaborateFile(astFile)` ~l.174), and `result.Warnings` is appended in the SAME function scope (~l.189-198). `pipeline_module.go`: each module's surface AST is `mod.File` (elaborated ~l.318), `result.Warnings` appended ~l.337/388. |

**Corrected target (the planner must use this, NOT `internal/check/`):** implement the pass in
**`internal/pipeline/`**, in the same style as `internal/pipeline/warn_split_args.go`: a
compile-time, non-blocking pass whose result is appended to `result.Warnings` at the two existing
call-sites `internal/pipeline/pipeline_single.go:198` (single-file) and
`internal/pipeline/pipeline_module.go:388` (multi-module), and surfaced through `ailang check`
via `cmd/ailang/check.go` (`runCheckWithContext`), not a new command file.

### RESOLVED design decision: option (a) — syntactic pass on the post-parse surface AST (mission iter 42)

`warn_split_args.go` walks the **post-elaboration Core** program (`DetectArgOrderWarnings(prog
*core.Program)`), which needs no return-type information. This pass is different: its trigger is
"function whose *declared return type* is `Result[_,_]`", and the Solution Design below inspects
`FuncDecl.Annotations` + syntactic `Result[_,_]` — both of which live on the **post-parse surface
AST** (`*ast.File`), not on Core. (Earlier drafts said "pre-elaboration *typed* AST" — that wording
was imprecise: the check needs **no type inference**. Its trigger is the syntactic return-type
annotation on the surface `FuncDecl`, the Ok(<empty-literal>) patterns A/B/C are syntactic, and
`@allow_empty_ok` is a parse-time annotation. It is a purely **syntactic** pass.)

**DECISION: option (a).** Run `DetectStrictFallbacks(astFile *ast.File) []elaborate.Warning` as a
syntactic pass over the post-parse surface AST, wired at BOTH pipelines:

- **Single-file** (`internal/pipeline/pipeline_single.go`): the parsed surface AST is set as
  `result.Artifacts.AST = astFile` at ~line 159, BEFORE elaboration (~line 174), and
  `result.Warnings` is a plain `[]elaborate.Warning` appended to in the SAME function scope
  (~lines 189-198). The new pass runs right after parse, its result appended alongside the
  existing warnings.
- **Multi-module** (`internal/pipeline/pipeline_module.go`): each module's surface AST is
  `mod.File` (elaborated at ~line 318 via `elaborator.ElaborateFile(mod.File)`); the pass runs
  per-module over `mod.File` in that loop, appending to `result.Warnings` (existing appends at
  ~lines 337/388).

The pre-elaboration hook option (a) originally required verifying therefore **exists** — the
surface AST and `result.Warnings` are already in the same scope in both pipelines; no new
elaboration hook is needed.

**Option (b) — threading return-type + annotation info into a Core-level pass (e.g. via the
`iface` layer) — is REJECTED:** it is unnecessary. Everything the check needs is syntactically
present on the surface `FuncDecl`; a Core-level pass would add plumbing (and lose the surface
`FuncDecl` nodes) for zero benefit.

This layer choice was flagged by the `gemini-3-1-pro` reviewer both quorum rounds; it is now
resolved in the doc. All other design DECISIONS (A/B/C detection, `@allow_empty_ok`,
error-on-publish/warn-on-dev) are unchanged and were not objected to.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure static analysis; no new nondeterminism |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | **+1** | Strengthens local reasoning: a `Result`-returning function is now machine-verifiable to distinguish failure from default-success |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | **+1** | The check teaches agents the "NO SILENT FALLBACKS" principle without requiring them to memorize it from CLAUDE.md prose. Failure is no longer indistinguishable from happy-path-with-zeros. |
| A8: Minimal Syntax | 0 | One new annotation (`@allow_empty_ok`), used only where needed |
| A9: Cost Visibility | 0 | Compile-time only |
| A10: Composability | 0 | No composition impact |
| A11: Structured Failure | **+2** | Direct application of CLAUDE.md's "NO SILENT FALLBACKS" principle. Converts an unchecked convention into a static check. |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Pure static analysis
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Strengthens machine-readability of `Result` semantics

## Problem Statement

CLAUDE.md states:

> **NO SILENT FALLBACKS - FAIL LOUDLY**
> If the fallback value affects data integrity, business logic, or user decisions → **NO FALLBACK**. Return errors/zero instead of guessing when data affects integrity.

This principle is well-articulated in our docs but has no static enforcement. The exact class of bug it warns against keeps re-emerging because nothing prevents an author from writing `Ok(emptyDefault)` instead of `Err("...")` in a fallback branch.

### Real production incident (May 2026)

`firestore/client@0.7.1` had this code in `getDoc`:

```ailang
-- Returns Err if document doesn't exist (404) or on network error.
export func getDoc(...) -> Result[Json, string] = ...
  ...
  match get(json, "fields") {
      Some(fields) => Ok(fields),
      None => Ok(jo([]))    -- ← THIS LINE
  }
```

The doc comment said "Returns Err if document doesn't exist or on network error." The body silently violated that contract by returning `Ok({})` for a corrupt-document state. Downstream:

1. `billing_store/entitlements_repo.getEntitlements` pattern-matched `Ok(fields) =>` with empty fields
2. Used `firestore/fields.asStr`/`asInt`/`asBoolField` helpers (which return zero defaults for missing fields)
3. Constructed an all-zero `Entitlements` record
4. Returned `Ok(allZeroEntitlements)`
5. `billing_service_api/entitlements_handler` took `Ok(e) => e` branch — never fell through to its `Err(_) => freeEntitlements(...)` fallback
6. Customer saw `{"plan":"","monthlyRequestLimit":"0","canOperate":"false",...}` with no error signal

The bug ran in production until customer complaints surfaced it. Diagnosis required reading the package source across three repositories. M-SERVEAPI-SURFACE-DROPS chased a red herring about module dropping before the actual mechanism was found.

**Bug history (immutable refs):** ailang messages `e1814c9f` → `952d6ef0` → `3375cbf5` → `8154269d`. Fix: ailang-packages commit `495be81` (firestore@0.7.2).

**Impact:** every package author writing a `Result`-returning function is one keystroke away from the same bug. There's no review-time signal that "this Ok branch returns a zero default" is different from "this Ok branch returns a real value."

## Goals

**Primary Goal:** Detect, at static-check time, the "Ok contains default-valued literal" anti-pattern in `Result`-returning functions, and refuse to publish a package that contains one without an explicit `@allow_empty_ok` opt-out annotation.

**Success Metrics:**
- Running the check against `firestore/client@0.7.1` source emits a diagnostic on the offending line; the v0.7.2 patched source emits none.
- The check is callable via `ailang check --strict-fallbacks` for app code and runs automatically inside `ailang check --package` for package authors.
- AILANG's own stdlib + the project's existing package code pass the check (with `@allow_empty_ok` annotations added where the empty-Ok is legitimate, e.g. `listDocs`).
- Zero false positives on stdlib `Result`-returning functions after annotation pass.
- An agent reading the diagnostic text alone (no CLAUDE.md memorization) understands what to do: change `Ok(default)` to `Err("...")` or add `@allow_empty_ok` with a rationale.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Annotation name: `@allow_empty_ok` (snake_case) vs `@allow-empty-ok` (kebab) vs `@result_may_be_empty` | Sets the package-author-facing API. Snake_case matches existing whitelist (`mcp_name`); kebab-case would require lexer changes. | human (this doc) | design | high (rename after ship = breaking) |
| Detection breadth: just `None => Ok(emptyLit)` vs full ad-hoc literal classification (records, lists, strings, ints, bools) | Narrow detection is precise but misses the record-of-zeros class; broad detection has false-positive risk on legitimate `Ok(0)` returns | human (this doc) | design | med |
| Failure modality: error vs warning by default | Error breaks existing packages on upgrade; warning is noise-floor. Strict-on-publish, warn-elsewhere is the middle path. | human (this doc) | design | med |
| Inter-procedural analysis: catch `Err(_) => someHelperReturningZeroes()` | Catches more bugs but requires call-graph analysis | agent | compile | high |

### Design Freeze

- [x] **Annotation name: `@allow_empty_ok`** (snake_case). Matches existing whitelist convention (`@mcp_name`). Required argument: a rationale string. Example: `@allow_empty_ok("empty list is a legitimate empty-collection result")`. Without the rationale, the annotation is rejected.
- [x] **Detection breadth: cover patterns A/B/C below.** Empty list `[]`, empty record `{}`, empty string `""` in `Ok`, AND record constructor application where ALL fields are explicit zero literals (`""`, `0`, `false`, `[]`, `{}`). NOT inter-procedural — pure literal-classification at the use site.
- [x] **Failure modality: ERROR when running `ailang check --package` (publish-time gate), WARNING when running `ailang check --strict-fallbacks file.ail` (development).** Package-publish strictness; app-development advisory.
- [x] **Inter-procedural is out of scope for this sprint.** Tracked as M-CHECK-STRICT-FALLBACKS-INTERPROC follow-up; the literal-pattern version catches the firestore@0.7.1 incident, which is the bulk of real-world cases.

## Conflict Surface

This change touches `internal/parser/parser_decl.go` (annotation whitelist) and adds a new static pass in `internal/pipeline/` (precedent: `warn_split_args.go`). Per the design-doc-creator's required Conflict Surface analysis:

### What syntactic positions does this change extend?

1. **Function annotations**: `@allow_empty_ok("rationale")` joins the existing set (`@route`, `@raw`, `@nowrap`, `@noexpose`, `@mcp_name`, `@verify`).
2. **Static check pipeline**: new syntactic pass added to `ailang check` after parsing (surface AST), before elaboration.

### What OTHER valid constructs already live in those positions?

- Existing annotations on function declarations — `@route("METHOD", "/path")`, `@raw`, etc. — use the same `@<name>(args)` syntax. The parser's annotation whitelist switch (`internal/parser/parser_decl.go::parseAnnotation`) handles them by exact name match.
- No syntactic ambiguity: an annotation only appears on a top-level decl, and adding `allow_empty_ok` to the whitelist is a pure additive change.

### How does the parser/typechecker disambiguate?

- The parser's annotation handler is a name-keyed switch. Adding a new case is mechanical — no precedence change, no token-class shift, no lookahead change.
- The typechecker doesn't interact with annotations (they're metadata-only on AST). The new check pass reads `FuncDecl.Annotations` directly off the post-parse surface AST.

### Programs that MUST still work post-change

1. **Existing stdlib `Result`-returning functions** (e.g. `std/result.unwrap`, `std/option.toResult`) — the check WILL fire on these if they have empty-Ok fallbacks. Plan: audit pass to add `@allow_empty_ok` where legitimate.
2. **Existing user packages with `None => Ok([])` patterns** — would fail strict publish. Migration path: add `@allow_empty_ok` annotation with rationale.
3. **`firestore/client@0.7.2` `listDocs`** (the corrected red-herring case I told docparse about) — `None => Ok(jo([]))` on missing `"documents"` key is legitimate (empty collection). Will need `@allow_empty_ok("missing 'documents' key in Firestore list response means empty collection")` annotation.

### What deliberately changes (intentional incompatibilities)

- `firestore/client@0.7.1`-style code (`None => Ok(jo([]))` in `getDoc`) will fail `ailang check --package`. Authors must either fix the bug (return Err) or annotate with rationale.

### Fixture tests (will be added in sprint)

The sprint plan will include positive + negative fixtures for each pattern A/B/C, plus a "legitimate empty list" fixture verifying the `@allow_empty_ok` opt-out.

## Solution Design

### Overview

A new check pass in `internal/pipeline/strict_fallbacks.go` —
`DetectStrictFallbacks(astFile *ast.File) []elaborate.Warning` — a purely **syntactic** pass over
the post-parse surface AST (no type inference needed), called at two pipeline sites:

- `pipeline_single.go`: right after `astFile` is set (~line 159), result appended to
  `result.Warnings` (~lines 189-198)
- `pipeline_module.go`: per-module over `mod.File` in the compile loop, appended to
  `result.Warnings` (~lines 337/388)

The pass itself:

1. **Find candidates**: walk every surface `*ast.FuncDecl` whose declared return-type annotation is `Result[_, _]`.
2. **Walk the body**: look for `Ok(<literal>)` expressions where the literal is structurally empty/zero-default.
3. **Check opt-out**: if the surrounding `FuncDecl` has `@allow_empty_ok("rationale")`, skip.
4. **Emit diagnostic**: `STRICT_FALLBACK_001` with file:line:col, the offending expression, and a suggested fix.

### Detection rules (patterns A/B/C)

**Pattern A: `Ok` with empty-collection or empty-string literal in a match-arm**

```ailang
match get(x, "key") {
    Some(v) => Ok(v),
    None => Ok([])         -- ← Pattern A: empty list in Ok
}
match parse(s) {
    Ok(v) => Ok(v),
    Err(_) => Ok("")       -- ← Pattern A: empty string in Ok
}
match getDoc(coll, id) {
    Ok(fields) => Ok(fields),
    Err(_) => Ok(jo([]))   -- ← Pattern A: empty record (json object) in Ok
}
```

**Pattern B: `Ok` with explicit all-zero record literal**

```ailang
match parseUser(s) {
    Ok(u) => Ok(u),
    Err(_) => Ok({name: "", age: 0, active: false})  -- ← Pattern B
}
```

Detection rule: all field initializers are literal zero values (`""`, `0`, `false`, `[]`, `{}`, `0.0`). Mixed records (`{name: someVar, age: 0}`) do NOT flag.

**Pattern C: `Ok` with constructor application whose all arguments are zero literals**

```ailang
type Plan = {name: string, limit: int}

func parsePlan(s: string) -> Result[Plan, string] = 
  match decode(s) {
    Ok(v) => Ok({name: getStr(v), limit: getInt(v)}),
    Err(_) => Ok({name: "", limit: 0})   -- ← Pattern C (record constructor)
  }
```

Same logic as Pattern B but for tagged-union constructors as well. `Ok(MyCtor("", 0))` flags if MyCtor's all positional args are zero literals.

**Constructor-vs-function is decided SYNTACTICALLY** (resolves the `gemini-3-1-pro` iter-42 quorum
objection that Pattern C "isn't purely syntactic"). AILANG **language-enforces** uppercase-first
data constructors: a variant/constructor decl that is not UpperCamelCase is a parse error
(`PAR_VARIANT_NEEDS_UIDENT` at `internal/parser/parser_type_decl.go:241`; the elaborator's
`isUpperIdent` at `internal/elaborate/patterns.go:78` likewise treats an unresolved uppercase
identifier as a nullary constructor). So the pass classifies an application head purely by its
first character:

- **Uppercase-first head** (e.g. `MyCtor(…)`, `Ok(…)`) → a data-constructor application → eligible
  for Pattern C when ALL positional args are zero literals.
- **Lowercase-first head** (e.g. `computeThing(…)`, `getStr(…)`) → a function call → **never flags**
  (already covered by the "`Ok(v)` where `v` is a function call does NOT flag" rule below).

No name resolution or type information is required — the uppercase rule is a hard syntactic
invariant of the language, so the pass remains purely syntactic over the surface AST.

### What does NOT flag

- `Ok(0)` or `Ok("")` at top-level of a function body where the function's success type semantically IS `int`/`string` and zero is a legitimate value. Detection rule: only flag in match-arm contexts (fallback positions) or after explicit `Err` checks. Top-level `Ok(0)` in a function whose whole job is to return a constant value is fine.
- `Ok(v)` where `v` is any non-literal expression (variable, function call, etc.).
- Records/lists/strings constructed from non-literal expressions.

### Opt-out annotation

```ailang
-- @allow_empty_ok("Empty list is a legitimate empty-collection result")
export func listDocs(coll: string, pageSize: int) -> Result[Json, string] ! {Net, FS, Env} = ...
```

The annotation requires a string-literal rationale. The check pass records the rationale in its output (visible via `ailang check --json`) so reviewers can audit.

The annotation MUST be added per the API server rules' annotation-whitelist discipline:

- [ ] `internal/parser/parser_decl.go` — add `case "allow_empty_ok":` to `parseAnnotation()` switch
- [ ] Update error message in `default:` case
- [ ] `internal/parser/route_attr_test.go` — add parser test
- [ ] `internal/pipeline/strict_fallbacks.go` — implement runtime behavior
- [ ] `docs/docs/guides/strict-fallbacks.md` — document the annotation
- [ ] CHANGELOG entry

### CLI integration

```bash
# Development: warning-level, exits 0
ailang check --strict-fallbacks file.ail
ailang check --strict-fallbacks dir/

# Package publish: error-level, exits 1 on any unannotated violation
ailang check --package    # ← strict-fallbacks is implicit in package check

# Manually upgrade to strict in dev:
ailang check --strict-fallbacks --strict-mode file.ail
```

`--strict-fallbacks` is opt-in for plain `ailang check` to avoid breaking app development. `--package` is opt-out via `--no-strict-fallbacks` for emergency publish scenarios.

### Diagnostic format

```
STRICT_FALLBACK_001 at client.ail:78:23

    None => Ok(jo([]))
                ^^^^^^^
    Ok branch returns empty record literal in a fallback position.

    A Result-returning function's Ok branch with a default/empty value
    can't be distinguished from a populated success by the caller.
    The downstream pattern:
        match yourFunc(...) {
            Ok(v)  => useV(v),     -- silently sees zero-defaulted v
            Err(e) => handleErr(e) -- never reached
        }
    will take the Ok arm with empty data instead of falling through
    to its Err handler.

    Suggestions:
      1. Return Err with a descriptive message:
         None => Err("response has no 'fields' key")
      2. If empty-Ok is legitimate here (e.g. empty-collection result),
         annotate the function:
           @allow_empty_ok("rationale for why empty-Ok is correct here")
           export func getDoc(...) -> Result[Json, string] = ...
```

### Files to Modify/Create

**New:**
- `internal/pipeline/strict_fallbacks.go` — the check pass, `DetectStrictFallbacks(astFile *ast.File) []elaborate.Warning`, syntactic walk of the surface AST (~300 LOC)
- `internal/pipeline/strict_fallbacks_test.go` — unit tests for pattern detection (~200 LOC)
- `examples/runnable/strict_fallbacks_demo.ail` — demonstrates the diagnostic + the `@allow_empty_ok` opt-out
- `docs/docs/guides/strict-fallbacks.md` — author-facing docs

**Modified:**
- `internal/pipeline/pipeline_single.go` — call the pass after `astFile` is set (~l.159), append to `result.Warnings` (~l.189-198) (~3 LOC)
- `internal/pipeline/pipeline_module.go` — call the pass per-module over `mod.File`, append to `result.Warnings` (~l.337/388) (~3 LOC)
- `internal/parser/parser_decl.go` — annotation whitelist (~5 LOC)
- `internal/parser/route_attr_test.go` — parser test for the new annotation (~30 LOC)
- `cmd/ailang/check.go` — wire `--strict-fallbacks` flag (~20 LOC)
- `cmd/ailang/check_package.go` — invoke strict check in package mode (~10 LOC)
- `cmd/ailang/help.go` — document `--strict-fallbacks` flag
- `changelogs/v0.10-current.md` — `[Unreleased]` entry
- `docs/docs/guides/package-development.md` — document the publish-time check

### Stdlib audit pass

A separate small task within this sprint: audit `stdlib/*.ail` for `Result`-returning functions with empty-Ok fallbacks. Add `@allow_empty_ok` where legitimate. Refactor to `Err` where not.

Estimated stdlib touch points: ~5-10 functions across `std/option`, `std/result`, `std/json`.

## Examples

### Example 1: The firestore@0.7.1 → 0.7.2 bug

**Before (v0.7.1 — fails strict check):**

```ailang
-- Returns Err if document doesn't exist (404) or on network error.
export func getDoc(collection: string, docId: string) -> Result[Json, string] ! {Net, FS, Env} =
  ...
  match get(json, "fields") {
      Some(fields) => Ok(fields),
      None => Ok(jo([]))    -- ← STRICT_FALLBACK_001
  }
```

`ailang check --package` would emit:

```
STRICT_FALLBACK_001 at client.ail:78:23
    Ok branch returns empty record literal in a fallback position...
```

Author choice: fix the bug.

**After (v0.7.2 — passes strict check):**

```ailang
match get(json, "fields") {
    Some(fields) => Ok(fields),
    None => Err("Document has no 'fields' key: ${collection}/${docId}")
}
```

No diagnostic.

### Example 2: Legitimate empty-Ok with opt-out

**listDocs returning empty list for empty collection:**

```ailang
-- @allow_empty_ok("Empty list is the natural empty-collection result for a list endpoint")
export func listDocs(collection: string, pageSize: int) -> Result[Json, string] ! {Net, FS, Env} =
  ...
  match get(json, "documents") {
      Some(docs) => Ok(docs),
      None => Ok(jo([]))    -- legitimate: missing key means empty collection
  }
```

No diagnostic. The rationale is preserved in `ailang check --json` output for audit.

### Example 3: Non-empty Ok with non-literal value (does NOT flag)

```ailang
export func freshUserPlan() -> Result[Plan, string] =
  let defaultName = generateName()
  in let defaultLimit = computeLimit()
  in match getStoredPlan() {
      Ok(p) => Ok(p),
      Err(_) => Ok({name: defaultName, limit: defaultLimit})  -- non-literal values
  }
```

The fallback `Ok` is constructed from runtime values, not literals. Caller still gets a populated record. No diagnostic.

## Success Criteria

- [ ] `internal/pipeline/strict_fallbacks.go` implements pattern A/B/C detection
- [ ] Running the check against the v0.7.1 `getDoc` body (preserved as a fixture) emits `STRICT_FALLBACK_001` on the correct line
- [ ] Running against the v0.7.2 patched body emits no diagnostic
- [ ] `@allow_empty_ok("rationale")` annotation suppresses the diagnostic on the annotated function
- [ ] Annotation without rationale (e.g. `@allow_empty_ok()`) is rejected by the parser
- [ ] `ailang check --strict-fallbacks file.ail` exits 0 with warnings
- [ ] `ailang check --package` exits 1 on any unannotated violation
- [ ] Stdlib audit complete — all legitimate empty-Ok functions have annotations
- [ ] `make ci` passes
- [ ] Example file `examples/runnable/strict_fallbacks_demo.ail` shows both the violation and the opt-out
- [ ] CHANGELOG entry + `docs/docs/guides/strict-fallbacks.md` shipped

## Testing Strategy

**Unit tests** (`internal/pipeline/strict_fallbacks_test.go`):

- Pattern A positive: `None => Ok([])`, `Err(_) => Ok("")`, `Err(_) => Ok(jo([]))` each flag
- Pattern B positive: `Err(_) => Ok({name: "", age: 0})` flags; `Err(_) => Ok({name: "real", age: 0})` does NOT
- Pattern C positive: `Err(_) => Ok(MyCtor("", 0))` flags; `Err(_) => Ok(MyCtor(name, 0))` does NOT
- Function-return-type filter: same patterns in non-`Result`-returning functions do NOT flag
- `@allow_empty_ok` suppression: annotated function does NOT flag
- Annotation requires rationale: parse error when rationale string is missing

**Integration tests:**

- Fixture file = literal copy of `firestore/client@0.7.1` `getDoc` body → flags
- Fixture file = literal copy of `firestore/client@0.7.2` `getDoc` body → no flag
- Fixture file = `firestore/client@0.7.2` `listDocs` with `@allow_empty_ok` → no flag

**Acceptance test:**

- Run `ailang check --package` against the entire AILANG stdlib + packages directory after annotation audit pass. Exit code 0, no unannotated violations.

## Deferred Decisions

- **Inter-procedural detection.** Catching `Err(_) => zeroHelper()` where `zeroHelper` returns a zero record requires call-graph analysis. Punt to M-CHECK-STRICT-FALLBACKS-INTERPROC.
- **Type-system-level enforcement.** A `NonEmpty[T]` newtype or `Required[T]` field marker would make this redundant at the type level. Larger design discussion, defer to v0.22+.
- **Other Result anti-patterns.** `Result` chaining patterns where downstream code unwraps Err silently. Could be caught by a related rule (STRICT_FALLBACK_002+); not in scope for this sprint.

## Non-Goals

- **Generic linter framework.** This is a single targeted rule. We don't need a plugin system to land one rule.
- **Reforming `firestore/fields.asStr/asInt`.** The zero-default helpers are documented as such; the issue was using them where Required-variants belonged. Companion sprint M-STDLIB-REQUIRED-FIELD-HELPERS.
- **Dynamic runtime detection.** The user-visible incident manifested at runtime (all-zero JSON), but the cause is statically detectable. Runtime detection is a different layer (telemetry / chaos testing).
- **Coverage of arbitrary "empty" patterns in non-Result contexts.** The rule is specifically about `Result.Ok` lying about success.

## Timeline

**Day 1 (~7 hours)**:
- 09:00-10:00: Add `@allow_empty_ok` to parser annotation whitelist + parser test
- 10:00-12:00: Implement `internal/pipeline/strict_fallbacks.go` patterns A/B/C
- 12:00-13:00: Lunch
- 13:00-14:30: Unit tests for pattern detection
- 14:30-15:30: Wire `--strict-fallbacks` flag in `cmd/ailang/check.go` + integrate into `--package`
- 15:30-16:30: Stdlib audit pass: scan + annotate legitimate empty-Ok functions
- 16:30-17:00: Docs + CHANGELOG + example file
- 17:00: `make ci`, commit, push

**Total: ~1 day** (consistent with M-DX26 Phase 5 and M-SERVEAPI-SURFACE-DROPS velocity).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| False positives on stdlib `Result`-returning functions | Medium | Audit pass + `@allow_empty_ok` annotations as part of this sprint |
| Annotation churn in existing user packages | Medium | `--package` mode is strict; `ailang check` default is warning-only. Packages can adopt at their own pace; only fails on publish. |
| Pattern detection misses subtler cases (e.g. `let zero = ""; Ok({name: zero})`) | Low | Document the literal-only scope. M-CHECK-STRICT-FALLBACKS-INTERPROC follow-up handles harder cases. |
| Diagnostic message confuses readers who don't know about Result | Low | Diagnostic includes the downstream-caller pattern example so the WHY is in the message itself. |

## Related Documents

**Implemented (informs design):**
- M-DX26 Phase 5 (just shipped) — verified `ensures` clauses are the complementary dynamic check. Strict-fallbacks is the static side of the same hygiene push.

**Planned (related sprints):**
- [m-serveapi-surface-drops.md](m-serveapi-surface-drops.md) — same incident-driven family. Surface-drops handles loud failure at startup; strict-fallbacks handles loud failure at publish time. Both encode "NO SILENT FALLBACKS."
- M-STDLIB-REQUIRED-FIELD-HELPERS (not yet filed) — `firestore/fields` Required-variants. Complementary fix at the field-helper layer.
- M-CHECK-STRICT-FALLBACKS-INTERPROC (not yet filed) — inter-procedural detection (helper functions returning zero records).

**Bug history (motivation):**
- ailang messages: `e1814c9f` (initial repro), `952d6ef0` (red-herring bisect), `3375cbf5` (deployment question), `8154269d` (correct diagnosis), `79c86f8e` (fix shipped notification)
- ailang-packages commit `495be81` — firestore@0.7.2 fix
- AILANG dev commits: `06450b85`, `93f4b6fb`, `eea88f33`, `000da04a` — M-SERVEAPI-SURFACE-DROPS sprint

## References

- [Design Axioms A11](/docs/references/axioms#a11-structured-failure) — Structured Failure
- [CLAUDE.md "NO SILENT FALLBACKS"](../../../CLAUDE.md) — the principle this check enforces
- [.claude/rules/api-server.md annotation-whitelist discipline](../../../.claude/rules/api-server.md) — annotation-add checklist (applied here to a new annotation)

## Future Work

- **Inter-procedural detection** (M-CHECK-STRICT-FALLBACKS-INTERPROC) — catch `Err(_) => helperReturningZeros()`.
- **`STRICT_FALLBACK_002` etc. — other Result anti-patterns** — silent error swallow on chain, ignored `Result` return values, etc.
- **Doc-comment-as-contract** — parse "Returns Err if X" from doc comments and verify the function body actually returns Err in those cases. Heavier lift; deserves its own design doc.
- **`@allow_empty_ok` audit log** — `ailang check --json` already includes the rationale; surface this in `/api/_meta/strict-fallbacks` health endpoint or similar so operators can audit accumulated suppressions.

---

**Document created**: 2026-05-15
**Last updated**: 2026-07-17 (mission iter 42 — OPEN design decision resolved to option (a))

DESIGN_DOC_PATH: design_docs/planned/v0_21_0/m-check-strict-fallbacks.md
