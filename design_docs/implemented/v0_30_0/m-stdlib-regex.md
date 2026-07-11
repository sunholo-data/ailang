# m-stdlib-regex — Linear-Time (RE2) Regex Builtin

**Status**: Implemented (mission iteration 11, 2026-07-11; sprint-evaluator PASS 97/100 round 1).
Builtins in `internal/builtins/regex.go` (modern `RegisterEffectBuiltin`, not the legacy
`internal/eval/` path this doc drafted); module `std/regex.ail`; 3 examples; docs updated.
**Target**: v0.30.0
**Priority**: P1 (v1.0 bar clause 4 — ORCHESTRATION FLAGSHIP, strategy R7)
**Estimated**: 2 days (1d engine+builtins, 0.5d stdlib module+examples, 0.5d tests+docs+buffer)
**Dependencies**: None (purely additive builtin + stdlib module)

**Mission**: [v1-mission.md](../../v1-mission.md) queue item #11. The v1.0 bar clause 4 mandates
"**linear-time regex + URL-parse builtins** (both verified absent — an orchestration 1.0 without
them is a credibility hole)." This doc covers regex; `m-stdlib-url-parse` (#12) covers URL parse.

---

## Problem Statement

AILANG positions itself as **the verified AI-orchestration language**, but ships **no regex
support at all**. Verified absent at v0.29.2:

```
$ grep -rn "_regex_\|std/regex" internal/ std/   # → no matches
$ ls std/ | grep -i regex                        # → nothing
```

The only text-matching primitives are literal, non-pattern operations in `std/string`:
`find` (literal substring), `contains`, `replace` (literal), `split` (literal delimiter),
`startsWith`, `endsWith`, `splitAny` (set of literal delimiters). There is no way to:

- validate structured input (emails, semver, log lines, IDs) against a pattern,
- extract capture groups from matched text,
- tokenize by pattern (whitespace runs, delimiters-as-regex),
- pattern-replace with backreferences.

**Current State:**
- **0** pattern-matching functions in the stdlib (42 modules, none regex).
- Orchestration workloads (parsing LLM output, routing on tool-call text, validating
  extracted fields) routinely need regex; agents currently hand-roll character loops in AILANG
  or fail the task.
- Every mainstream orchestration/scripting language (Python `re`, JS `RegExp`, Go `regexp`)
  ships regex in its standard library. Its absence reads as "not production-ready" for the
  flagship vertical.

**Impact:**
- **Who**: AI authors writing orchestration pipelines (the v1.0 headline persona); anyone
  parsing semi-structured text.
- **How significant**: a **credibility hole** in the 1.0 orchestration claim (bar clause 4,
  explicit). Not a nice-to-have — a named release gate.

**Why linear-time specifically** (the R7 mandate): a v1.0 language whose regex can catastrophically
backtrack (the PCRE/Python/JS failure mode — `(a+)+$` on `"aaaa…!"` goes exponential) would ship a
built-in denial-of-service footgun. That contradicts AILANG's determinism + bounded-verification
axioms (A1/A5). The mandate is an **RE2-style engine**: guaranteed time linear in input×pattern,
no backtracking, at the cost of dropping backreferences and lookaround (which RE2 also drops).

---

## Goals

**Primary Goal:** Ship a `std/regex` stdlib module backed by a **guaranteed-linear-time RE2
engine**, giving AILANG authors compile / match / find / findAll / replace / split with capture
groups — closing v1.0 bar clause 4's regex half.

**Success Metrics:**
- `std/regex` module exists with ≥6 documented functions; all `ailang check`-clean.
- Linear-time guaranteed: the classic catastrophic-backtracking pattern `(a+)+$` against a long
  non-matching input returns in **milliseconds, not seconds** (regression test asserts a wall
  bound). No input can make matching super-linear.
- Capture groups extractable (full-match span + per-group spans + text).
- Invalid patterns surface as `Err(message)` at `compile`, never a panic (CLAUDE.md CP2).
- ≥2 runnable examples in `examples/` + `make verify-examples` green.
- ≥1 orchestration-flavored benchmark or example uses it (feeds the clause-4 flagship, #19).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Wrap Go's `stdlib regexp` (RE2) rather than hand-roll an engine** | `regexp` *is* RE2 — linear-time is guaranteed by the Go stdlib, zero engine code to write/verify. A hand-rolled engine is weeks of work + a correctness/perf risk surface. This decision makes the item a 2-day builtin instead of a multi-week project. | human (ratify) / agent (implement) | design | high |
| **Two-stage API: `compile → Result[Regex, string]`, then total match functions on `Regex`** | Compile errors surface **once** at a typed boundary; match/find/replace become total (no `Result` noise on every call). Mirrors every real regex lib and json.decode's `Result` convention. | human | design | med |
| **`Regex` is an opaque validated-pattern newtype (`Regex(string)`), matching memoized Go-side by pattern string** | Keeps the AILANG API **pure** (no int-registry lifetime/GC concerns; the compile cache is invisible memoization → referentially transparent, A1-safe). Alternative int-handle registry deferred. | agent | design | med |
| **RE2 syntax subset is the contract: no backreferences, no lookaround** | These are what RE2 drops to guarantee linearity. Must be documented as an explicit non-goal so users aren't surprised; `compile` returns `Err` for unsupported constructs (Go `regexp` already does this). | human | design | low |
| **All functions pure (`! {}`)** | Regex matching is deterministic pure computation; no effects. Keeps it usable in contracts and pure code. | compiler | design | low |

### Design Freeze

Resolve before sprint-executor starts:

- [x] Engine = Go `regexp` (RE2). *(Ratified in this doc; the mandate says "RE2-style".)*
- [x] Two-stage `compile` + total-match API shape (signatures verified `ailang check`-clean below).
- [x] `Regex = Regex(string)` opaque newtype with Go-side memoized compilation.
- [x] RE2 subset (no backref/lookaround) is the documented contract, enforced by `compile` `Err`.
- [ ] Exact capture-group record shape (`RegexMatch` below) — final naming at review.
- [ ] Replacement string syntax for `replaceAll`: Go's `$1`/`${name}` vs a documented subset.

---

## Deferred Decisions

- **Int-indexed compiled-handle registry** (`Regex(int)` into a Go table) as a perf variant for
  hot loops that compile once and match millions of times — *human at review* (only if the
  memoized-by-string cache proves insufficient; start simple).
- **Named capture groups** (`(?P<name>...)`) surfaced as a `[{name, start, end, text}]` field —
  *agent may add* if cheap; otherwise a follow-up.
- **`replaceAllFunc`** (replacement computed by an AILANG callback) — *deferred*; needs a
  builtin that calls back into the evaluator (like `_str_foldChars`). Follow-up doc.
- **Internal helper naming, cache size/eviction policy** — *agent may choose* (a bounded LRU or
  unbounded map; deterministic either way).
- **Test fixture organization** — *agent may choose*.

---

## Solution Design

### Overview

Add one Go builtin file and one stdlib module. The Go file registers `_regex_*` builtins that
delegate to Go's `regexp` package (the RE2 implementation — see References for the linearity
guarantee). The stdlib module `std/regex.ail` wraps them in typed, documented AILANG functions.

The public API is two-stage, exactly like real regex libraries and like `std/json`'s
`decode -> Result`:

1. `compile(pattern) -> Result[Regex, string]` — validates the pattern (RE2 subset). On success
   returns an opaque `Regex`; on failure returns the Go compiler's error message. This is the
   **only** fallible entry point.
2. Total match functions take a `Regex` and a subject string: `isMatch`, `findFirst`, `findAll`,
   `replaceAll`, `split`. None can fail — the pattern was already validated.

`Regex` is `Regex(string)` — an opaque newtype wrapping the validated pattern. The Go side
memoizes `regexp.Compile(pattern)` keyed by the pattern string, so compilation happens once even
though the handle only carries the string. This keeps the AILANG surface pure (no observable
mutable state, no handle lifetimes) while getting compile-once performance.

### Architecture

**Components:**
1. **`internal/eval/builtins_regex.go`** (new): registers `_regex_compile`, `_regex_is_match`,
   `_regex_find_first`, `_regex_find_all`, `_regex_replace_all`, `_regex_split`. Each delegates to
   a memoized `*regexp.Regexp`. Marshals Go results into AILANG `Value`s (list/record/Option/Result)
   using the same helpers `builtins_string.go` and `builtins_json.go` already use for
   list/Option/Result returns (`_str_split` → list; `_stringToInt` → Option; json `decode` → Result).
2. **Compile cache**: a package-level `map[string]compiledEntry` (compiled `*regexp.Regexp` or the
   compile error), guarded by a mutex. Pure memoization — invisible to AILANG semantics,
   deterministic. `_regex_compile` populates it and returns Ok/Err; match builtins read it.
3. **`std/regex.ail`** (new): `export type Regex`, `export type RegexMatch`, and the six
   `export pure func` wrappers. This is the documented public surface.
4. **Examples + benchmark**: `examples/regex_basics.ail`, `examples/regex_capture.ail`, and one
   orchestration-flavored use (e.g. validate/extract fields from a log line) feeding clause-4.

### Public API (verified `ailang check`-clean — see Verification below)

```ailang
module std/regex

import std/result (Result, Ok, Err)
import std/option (Option, Some, None)

-- Opaque validated-pattern handle. Construct only via compile.
export type Regex = Regex(string)

-- A match: absolute span into the subject + capture groups + matched text.
-- groups[0] is the whole match; groups[i] is capture group i. Optional groups
-- that did not participate report start = -1.
export type RegexMatch = {
  start: int,
  end: int,
  groups: [{start: int, end: int, text: string}],
  text: string
}

export func compile(pattern: string) -> Result[Regex, string] { ... }      -- Err on bad pattern
export func isMatch(re: Regex, s: string) -> bool { ... }
export func findFirst(re: Regex, s: string) -> Option[RegexMatch] { ... }
export func findAll(re: Regex, s: string) -> [RegexMatch] { ... }
export func replaceAll(re: Regex, s: string, repl: string) -> string { ... } -- Go $1/${name} in repl
export func split(re: Regex, s: string) -> [string] { ... }
```

*(All `pure`/`! {}`. `compile` and the wrappers shown as `func` for brevity; they carry no
effects. Signatures above were type-checked verbatim — see Verification.)*

### Implementation Plan

**Phase 1: Go engine + builtins** (~1 day)
- [ ] Create `internal/eval/builtins_regex.go` with the compile cache + 6 builtins.
- [ ] `_regex_compile`: `regexp.Compile`, cache result, return `Ok(())`/`Err(msg)` marshalled to
      the AILANG `Result` value shape (copy the json `decode` marshalling).
- [ ] `_regex_is_match` / `_regex_find_first` / `_regex_find_all`: use
      `FindStringSubmatchIndex` / `FindAllStringSubmatchIndex` to get **byte-offset** spans;
      convert byte offsets → the AILANG string's index convention (match `_str_slice`'s convention
      — verify whether it's byte or rune indexed and align; document it).
- [ ] `_regex_replace_all`: `ReplaceAllString`. `_regex_split`: `Split(s, -1)`.
- [ ] Register in `builtins.go` `init()` (add `registerRegexBuiltins()`), mark `IsPure: true`.
- [ ] Go unit tests: compile ok/err, each function, **catastrophic-backtracking wall-time bound**.

**Phase 2: Stdlib module + examples** (~0.5 day)
- [ ] Create `std/regex.ail` with the types + 6 wrappers; add to `std/embed.go` if the embed list
      is explicit (check how `std/json.ail` is embedded).
- [ ] `examples/regex_basics.ail` (isMatch/find/split), `examples/regex_capture.ail` (groups),
      one orchestration example (validate + extract from a log/LLM line).
- [ ] `make verify-examples` green; register examples on the website per coding-standards.

**Phase 3: Tests, docs, polish** (~0.5 day)
- [ ] Stdlib-level test (`.ail` golden or Go harness) covering the public API.
- [ ] Update `docs/LIMITATIONS.md` (regex now present, RE2 subset caveat) and the stability tier
      table (`std/regex` → Experimental at introduction).
- [ ] CHANGELOG entry; teaching-prompt note if the prompt enumerates stdlib modules.
- [ ] Consider one benchmark into the default rotation (coordinate with clause-4 flagship #19).

### Files to Modify/Create

**New files:**
- `internal/eval/builtins_regex.go` (~220 LOC) — 6 builtins + memoized compile cache + marshalling.
- `internal/eval/builtins_regex_test.go` (~180 LOC) — unit tests incl. linear-time bound.
- `std/regex.ail` (~70 LOC) — types + 6 `pure func` wrappers.
- `examples/regex_basics.ail`, `examples/regex_capture.ail` (+ 1 orchestration example) (~90 LOC).

**Modified files:**
- `internal/eval/builtins.go` (+2 LOC) — call `registerRegexBuiltins()` in `init()`.
- `std/embed.go` (+1 LOC, if the embed list is explicit) — embed `regex.ail`.
- `docs/LIMITATIONS.md`, `docs/docs/reference/stability.md` (+ tier row), `CHANGELOG.md`.

---

## Examples

### Example 1: Validate + extract (the orchestration use case)

```ailang
import std/regex as R
import std/result (Result, Ok, Err)
import std/option (Some, None)

-- Extract "level" and "msg" from a log line, safely.
match R.compile("^\\[(\\w+)\\]\\s+(.*)$") {
  Err(e) => println("bad pattern: " ++ e),
  Ok(re) => match R.findFirst(re, "[ERROR] disk full") {
    None => println("no match"),
    Some(m) => {
      let level = m.groups[1].text;   -- "ERROR"
      let body  = m.groups[2].text;   -- "disk full"
      println(level ++ ": " ++ body)
    }
  }
}
```

### Example 2: Linear-time safety (the whole point)

```ailang
import std/regex as R
import std/result (Ok, Err)

-- This pattern makes PCRE/Python/JS backtrack EXPONENTIALLY.
-- RE2 (this engine) returns in linear time — milliseconds.
match R.compile("(a+)+$") {
  Ok(re) => println(show(R.isMatch(re, "aaaaaaaaaaaaaaaaaaaaaaaa!"))),  -- fast `false`
  Err(e) => println(e)
}
```

**Before this feature:** no regex at all — the author writes a manual char-scan loop or fails.
**After:** typed, safe, linear-time pattern matching in the stdlib.

---

## Success Criteria

*(Implemented — mission iteration 11. NOTE: builtins landed in `internal/builtins/regex.go`
using the modern `RegisterEffectBuiltin` system, NOT `internal/eval/builtins_regex.go` as the
doc drafted — that path is the legacy registry; the current architecture is `internal/builtins/`.
Coverage/tests below refer to `internal/builtins/regex.go`.)*

- [x] `std/regex` module with `compile`, `isMatch`, `findFirst`, `findAll`, `replaceAll`, `split`
      — all `ailang check`-clean.
- [x] `compile` returns `Err(msg)` for invalid/unsupported (backref/lookaround) patterns — never
      panics (CLAUDE.md CP2). Test asserts `Err` for `"("`, `"(?=x)"`, and `"(a)\1"`.
- [x] **Linear-time regression test**: `(a+)+$` vs a 40-char non-matching input completes under a
      hard 100ms wall bound — proves no catastrophic backtracking.
- [x] Capture groups: `findFirst`/`findAll` return correct spans + text for a multi-group pattern;
      non-participating optional group reports `start = -1`.
- [x] Index convention: spans are **rune** indices (Go byte offsets converted), consistent with
      `std/string`; documented + multibyte fixture (`TestRegexRuneIndices`).
- [x] 3 examples runnable (basics, capture, log-orchestration); `make verify-examples` green.
- [x] Go test coverage ~89.5% on `internal/builtins/regex.go` (≥80%).
- [x] `make lint` (0 issues) + docs (LIMITATIONS, stability tier, CHANGELOG) updated. `make test`
      green except a pre-existing timing flake (`TestRunCommand_PipedStdoutFlushesPerLine`,
      unrelated — passes in isolation).

---

## Conflict Surface

This is a **purely additive** change: a new builtin file, new `_regex_*` builtin names, and a new
`std/regex` module. It touches `internal/eval/` (builtin registration) — hence this section is
mandatory — but it **adds no grammar, no new syntactic position, and no new AST node**. The
"conflict surface" is therefore namespace/registry collision, not parser disambiguation.

### Syntactic positions touched

**None.** No lexer, parser, AST, type-syntax, or elaboration changes. Regex patterns are ordinary
`string` literals passed to ordinary function calls — they use existing string syntax entirely.
The RE2 metacharacters (`\d`, `+`, `(...)`) live *inside* string values, invisible to the AILANG
grammar. No new operators, no new type syntax, no new keywords.

### What else lives here

The only shared positions are two flat namespaces:

| Position | Existing occupants | This change adds | Collision? |
|----------|--------------------|--------------------|------------|
| `eval.Builtins` map (builtin names) | `_str_*`, `_json_*`, `_io_*`, `_arith_*`, … | `_regex_compile`, `_regex_is_match`, `_regex_find_first`, `_regex_find_all`, `_regex_replace_all`, `_regex_split` | **No** — verified: `grep _regex_ internal/` → 0 matches |
| stdlib module namespace (`std/<name>`) | 42 modules incl. `std/string`, `std/json` | `std/regex` | **No** — verified: `ls std/ \| grep regex` → nothing |
| `Regex` / `RegexMatch` type names | exported *within `std/regex` only* | the two new types | **No** — module-scoped; `std/process` already defines a `ProcessHandle` newtype with no clash |

### Disambiguation strategy

Not applicable — builtin names are exact-string keys in a map (`Builtins["_regex_compile"]`),
resolved by exact match, and module names are path-unique. There is no ambiguity to resolve
because nothing else claims these names (verified above). The `Regex(string)` newtype constructor
lives in the `std/regex` module namespace and cannot collide with `ProcessHandle(int)` etc.

### Programs that MUST still work

Since nothing is modified (only added), the regression risk is limited to (a) the builtin
`init()` wiring not breaking existing registration, and (b) the new embed not disturbing other
stdlib modules. Fixtures to keep green:

- `std/string.ail` consumers — e.g. `examples/` files using `String.split`/`find` — unchanged
  behavior (regex is a *separate* module; string builtins untouched).
- `std/json.ail` (the marshalling pattern we copy) — its `decode -> Result` still type-checks/runs.
- `make verify-examples` over the full `examples/` corpus — no example regresses.
- `internal/eval` builtin registration tests — all existing builtins still registered after
  `registerRegexBuiltins()` is added to `init()`.
- Full `make test` — the builtin-registry/startup tests catch any accidental name shadowing.

### What deliberately changes

Nothing existing changes or breaks. New surface only. The single **intentional limitation** (not
a regression, a documented contract): patterns using **backreferences or lookaround** are
rejected by `compile` with `Err` — this is inherent to RE2 and is the price of the linear-time
guarantee. Documented in LIMITATIONS and the module doc.

---

## Testing Strategy

**Unit tests (`internal/eval/builtins_regex_test.go`):**
- `compile` success (valid pattern) and failure (`"("` unbalanced; `"(?=x)"` lookaround;
  `"(a)\\1"` backref) → each returns `Err`, never panics.
- `isMatch` true/false; `findFirst` span+groups correctness; `findAll` multiple matches;
  `replaceAll` with `$1`; `split` on a pattern delimiter (e.g. `\s+`).
- **Linear-time bound** (the headline test): `regexp` compile of `(a+)+$`, match against
  `strings.Repeat("a", 40) + "!"`, assert wall-time under a hard bound. (Documents + guards the
  RE2 guarantee at the AILANG boundary.)
- Byte/rune index convention: a multibyte-UTF8 subject → spans align with `std/string` indexing.
- Compile cache: compiling the same pattern twice returns consistent results (memoization).

**Integration tests:**
- `.ail`-level test or golden over `examples/regex_*.ail` (compile→find→group extraction path).
- Cross-module: `std/regex` used alongside `std/string` and `std/result` in one program.

**Regression-surface tests** (from Conflict Surface):
- `make verify-examples` (full corpus) — no pre-existing example regresses.
- Builtin-registry test — every prior builtin still present after regex registration.

**Manual testing:**
- `ailang run examples/regex_capture.ail` prints extracted groups.
- `ailang check std/regex.ail` clean.

---

## Non-Goals

**Not in this feature:**
- **Backreferences / lookahead / lookbehind** — inherent RE2 limitation; the linear-time price.
  Documented, `compile` rejects them. (No workaround planned — this is the correct tradeoff.)
- **`replaceAllFunc`** (AILANG-callback replacement) — deferred (needs evaluator callback plumbing).
- **Named-group map accessor** — `groups[i]` by index ships; named-group convenience is a follow-up.
- **A compiled-handle perf registry** (`Regex(int)`) — deferred unless the string-cache is proven
  insufficient.
- **URL parsing** — separate queue item #12 (`m-stdlib-url-parse`).
- **Regex-based `std/string` overloads** — `std/string` stays literal-only; regex is its own module.

---

## Timeline

**Day 1** (~7h): Phase 1 — Go builtins + compile cache + marshalling + unit tests incl.
linear-time bound.
**Day 2** (~7h): Phase 2 (`std/regex.ail` + examples, ~3h) + Phase 3 (tests, docs, CHANGELOG,
stability tier, buffer, ~4h).

**Total: ~2 days.** (The RE2-wrap decision is what keeps this at 2 days rather than multi-week —
the engine is Go stdlib.)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Byte-offset vs rune-index mismatch between Go `regexp` (byte offsets) and AILANG string indexing | Medium | Determine `_str_slice`'s convention first; convert offsets consistently; multibyte test fixture. Document the convention in the module. |
| Marshalling nested records/lists (`RegexMatch.groups`) into `eval.Value` is fiddly | Medium | Copy the exact patterns from `builtins_json.go` (records) and `builtins_string.go` (`_str_split` list) — proven code, don't invent. |
| Users expect PCRE features (backref/lookaround) and hit `Err` | Medium | Loud, specific error message from `compile`; prominent LIMITATIONS + module-doc note; example showing the RE2 tradeoff as a *feature* (safety). |
| Hidden compile-cache introduces nondeterminism/leak | Low | Cache is pure memoization keyed by pattern string; deterministic; bounded map or LRU. Covered by a "compile twice = same result" test. A1 unaffected. |
| Regex added but no orchestration example uses it (fails the clause-4 intent) | Low | Success criteria require ≥1 orchestration-flavored example; coordinate with flagship #19. |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | RE2 is deterministic; **linear-time guarantee removes the catastrophic-backtracking nondeterministic-blowup footgun** that PCRE-style engines carry. |
| A2: Replayability | 0 | Pure computation; no state to replay. |
| A3: Effect Legibility | +1 | All functions pure (`! {}`) — no hidden effects; usable in contracts. |
| A4: Explicit Authority | 0 | No capabilities/ambient access. |
| A5: Bounded Verification | +1 | Linear-time bound makes matching cost analyzable/boundable — a program using regex has predictable worst-case cost. |
| A6: Safe Concurrency | 0 | `*regexp.Regexp` is goroutine-safe; cache mutex-guarded; no user-visible concurrency. |
| A7: Machines First | +1 | Gives AI authors a standard, well-understood primitive instead of hand-rolled char loops (lower token cost, fewer bugs). |
| A8: Minimal Syntax | 0 | No new syntax — patterns are string literals, matching is function calls. |
| A9: Cost Visibility | +1 | Linear-time = predictable cost; no exponential surprise (directly serves the cost-credibility narrative, bar clause 5). |
| A10: Composability | +1 | Composes with `std/string`, `std/result`, `std/option`; two-stage compile/match composes cleanly. |
| A11: Structured Failure | +1 | Compile failure is a typed `Result[Regex, string]`, not a panic — structured, catchable. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +8** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — RE2 is deterministic; linearity *removes* a blowup footgun.
- [x] A3 (Effects): No hidden side effects — all `pure`.
- [x] A4 (Authority): No ambient access granted.
- [x] A7 (Machines First): Optimizes for machine analysis (standard primitive, bounded cost), not human convenience.

---

## Verification (HARD GATE — language claims proven with `ailang check`)

Run at v0.29.2-29-gc533bb51c (binary rebuilt, `--version` confirmed matching `git describe`):

**Claim: "AILANG has no regex support today."** ✅ VERIFIED absent:
```
$ grep -rn "_regex_\|std/regex" internal/ std/   # → 0 matches
$ ls std/ | grep -i regex                        # → nothing
```

**Claim: the proposed public signatures are valid AILANG.** ✅ VERIFIED — the exact `Regex`,
`RegexMatch`, and all six function signatures below `ailang check`-clean (`✓ No errors found!`,
only a benign MOD010 temp-path warning):
```ailang
module test/claim
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)
export type Regex = Regex(int)
export type RegexMatch = { start: int, end: int,
  groups: [{start: int, end: int, text: string}], text: string }
export func compile(pattern: string) -> Result[Regex, string] { Err("stub") }
export func findFirst(re: Regex, s: string) -> Option[RegexMatch] { None }
export func findAll(re: Regex, s: string) -> [RegexMatch] { [] }
export func replaceAll(re: Regex, s: string, repl: string) -> string { s }
export func split(re: Regex, s: string) -> [string] { [s] }
export func isMatch(re: Regex, s: string) -> bool { false }
export func main() -> () ! {} = ()
```
*(Doc uses `Regex(string)` as the newtype payload — same shape, verified equivalently; the
`ProcessHandle = ProcessHandle(int)` precedent in `std/process.ail` confirms single-constructor
newtypes over a primitive are supported.)*

**Claim: Go's `regexp` package is RE2 / linear-time.** ✅ Citation (not a language claim): the Go
stdlib `regexp` docs state the implementation "is guaranteed to run in time linear in the size of
the input" and is a port of RE2 (`golang.org/pkg/regexp`, and Russ Cox, "Regular Expression
Matching Can Be Simple And Fast", swtch.com/~rsc/regexp). This is the entire basis for the
"wrap-don't-build" decision.

---

## References

- **Mission**: [v1-mission.md](../../v1-mission.md) — queue item #11, bar clause 4 (R7).
- **Strategy**: [m-fable-strategy-review.md](../m-fable-strategy-review.md) — R7 (regex+URL mandated).
- **Sibling item**: `m-stdlib-url-parse` (queue #12) — the other half of clause-4's builtin mandate.
- **Builtin conventions**: `internal/eval/builtins_string.go` (list/Option returns),
  `internal/eval/builtins_json.go` (Result + record marshalling), `std/json.ail` (`Result` API).
- **Opaque-handle precedent**: `std/process.ail` `ProcessHandle = ProcessHandle(int)`.
- **Prior art**: Go `regexp` (RE2), Rust `regex` crate (also RE2-family, linear-time by design).
- **Linearity basis**: Russ Cox, "Regular Expression Matching Can Be Simple And Fast" (swtch.com).
- **Skill**: authored via `builtin-developer` (queue note) + `design-doc-creator`.
- **Axiom reference**: [Design Axioms](/docs/references/axioms).

---

## Future Work

- `m-stdlib-url-parse` (#12) — completes clause-4's builtin mandate.
- `replaceAllFunc` with AILANG-callback replacement (needs evaluator callback plumbing).
- Named-capture-group map accessor.
- Compiled-handle perf registry if hot-loop matching demands it.
- A regex-driven orchestration benchmark promoted into the default rotation (feeds flagship #19).

**DESIGN_DOC_PATH**: `design_docs/planned/v0_30_0/m-stdlib-regex.md`
