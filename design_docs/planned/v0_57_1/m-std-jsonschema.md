# M-STD-JSONSCHEMA: validate a Json against a JSON Schema (Draft 2020-12) in the stdlib

**Status**: Planned (DOC-READY)
**Target**: v0.57.1
**Priority**: P1 (client deliverable correctness; the pack build cannot self-check envelopes against their schema today)
**Estimated**: 5 days (3d Go engine, 0.5d std surface + `std/list.all`, 1.5d tests/docs/gates)
**Dependencies**: None new. Consumes the existing `std/json` `Json` ADT (unchanged) and Go's `regexp` (RE2). Reported via `ailang messages` (cli), 2026-11-03, context dated 2026-10-19.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/docs/references/axioms.mdx)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure function of `(schema, doc)`; errors collected in a fixed depth-first document order with **no Go map iteration** in the emission path — the same input yields a byte-identical error list on every run |
| A2: Replayability | 0 | No trace-format or execution-model change |
| A3: Effect Legibility | +1 | The whole validator is `pure` — schema+instance in, error list out, zero effects declared, zero performed |
| A4: Explicit Authority | 0 | No capability or ambient-access change; local `$ref` only by design (see Non-Goals) |
| A5: Bounded Verification | +1 | A client deliverable (envelope ↔ schema conformance) becomes checkable *locally in AILANG* instead of by an outside Python `jsonschema` run |
| A6: Safe Concurrency | 0 | Single-threaded pure builtin |
| A7: Machines First | +1 | Errors are a typed record list with RFC 6901 pointers and the failing keyword as a field — a machine routes on `keyword`, a human reads `message`; no string scraping |
| A8: Minimal Syntax | +1 | Zero new language syntax; one new std module and one new std export |
| A9: Cost Visibility | 0 | No resource-surface change |
| A10: Composability | +1 | Composes with `std/json.decode`/`encode` (same ADT) and with future `std/yaml`-decoded documents — any `Json` validates |
| A11: Structured Failure | +1 | Failure mode is the product: `[ValidationError]`, not a thrown string; schema-side and instance-side failures are both in-band and distinguishable |
| A12: System Boundary | 0 | No boundary crossing; non-local `$ref` (network) is refused, not fetched |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (ordered keyword lists compiled ahead of any map; fixed evaluation order)
- [x] A3 (Effects): no hidden side effects (builtin registered `IsPure: true`, same class as `_json_decode`)
- [x] A4 (Authority): no ambient access granted (`$ref` resolution is pointer-only, in-memory)
- [x] A7 (Machines First): the errors *are* machine-parseable records; strict mode exists so a machine never mistakes "unsupported" for "valid"

## Problem Statement

On 2026-10-19 the Holosun Decision Budget shipped `decision-envelope.json` beside `decision-envelope.schema.json` as a **client deliverable**. An outside check (Python `jsonschema`) found the real client envelope **failing its own schema in 4 places** and the exemplar **in 6**. The AILANG-side validator enforces business rules, not the schema's shape — and **nothing in AILANG compares a document against its schema**. The gap was papered over with a stopgap: `_holosun/decision-budget/schema/envelope/schemacheck.ail` (~150 lines, pure) in the sunholo-data clients repo, which validates only the keyword subset that schema uses and reports unsupported keywords as errors rather than skipping them.

The stopgap had to exist because the stdlib has no JSON Schema module at all. Every future client schema (this pack's or any other's) re-routes through a bespoke, per-repo validator with a per-repo keyword subset and no shared tests. And the failure mode the stopgap guards against — a validator that silently skips keywords it does not implement — is exactly the failure mode a *shared* module must guard against by construction, or it inherits the same bug as a feature.

The reported keyword census of `decision-envelope.schema.json` (22 keywords, counted in the clients repo — **reported, not re-verified here**):

| Keyword | Count | Keyword | Count |
|---|---|---|---|
| type | 193 | minimum | 8 |
| required | 57 | allOf | 6 |
| properties | 48 | exclusiveMinimum | 3 |
| $ref (`#/$defs/…`) | 40 | anyOf | 2 |
| enum | 35 | not | 2 |
| items | 34 | maxLength | 2 |
| additionalProperties (false + schema) | 33 | dependentRequired | 1 |
| minLength | 19 | oneOf | 1 |
| pattern | 17 | maximum | 1 |
| if/then | 12 | | |

**Smaller finding, same report:** `std/list` exports `any` (std/list.ail:156) but not its obvious pair `all` — models that reach for `all(pred, xs)` hit `IMP010: symbol 'all' not exported by 'std/list'`, and the import machinery points them at `any`. One export fixes the class.

**Impact:**
- **Client-facing correctness**: schema violations shipped in a deliverable because no in-language check existed; the schema is documentation, not enforcement.
- **Every pack build** that wants to self-check envelope-vs-schema has to import, port, or reinvent a validator; the stopgap is now load-bearing in a clients repo.
- **The silent-pass trap**: a validator that skips what it does not implement reports "valid" for documents it did not actually check. This must be a *default-denying* property of the shared module, not a convention each stopgap re-argues.

## Goals

**Primary Goal:** Ship `std/jsonschema` with `validate(schema: Json, doc: Json) -> [ValidationError]` — pure, Go-backed, Draft 2020-12 core + validation vocabulary, strict-by-default on unknown keywords — so the Decision Budget pack build (and any future pack) can self-check documents against their schema in-language.

**Success Metrics:**
- All 22 census keywords validated correctly, pinned by a fixture that reproduces the census (Go table tests + `.ail` suite).
- Strict mode: a schema carrying an out-of-vocabulary keyword yields ≥ 1 `ValidationError` naming it; `validateLenient` is the explicit opt-out. No path returns "valid" for a schema it could not fully interpret.
- Deterministic: 100 repeated runs → byte-identical error lists; permuted schema object keys → identical error lists (test-pinned).
- `validate` on a ~1,000-property document against a census-shaped (~460-schema-node) schema in < 50 ms (benchmark in `internal/builtins/jsonschema_test.go`).
- `std/list.all` exported and symmetric with `any` (both engines, empty-list edge pinned).
- All touched gates green: `make test`, `test-stdlib-ail`, `verify-stdlib`, `verify-examples`, `check-file-sizes`, `check-boundaries`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Go-backed builtin engine (`internal/builtins`), not a pure `.ail` module | Correctness for a spec this subtle (evaluation order, `unevaluated*` annotation tracking, `$ref` cycles) + RE2 direct + speed; `_json_decode` precedent | human (this doc) | design | high |
| **Strict-by-default**: unknown schema keywords are errors, never skips | This is the product's core guarantee — a schema that outgrows the validator must fail loudly; `validateLenient` is the explicit opt-out | human (this doc) | design | high |
| Hand-rolled engine, not a wrapped Go JSON Schema library (incl. already-vendored `google/jsonschema-go`) | Strict unknown-keyword rejection is not any library's default posture; error shape must be our `ValidationError` with deterministic ordering; no behavior drift from upstream; wasm build stays dep-free | human (this doc) | design | high |
| Draft 2020-12 dialect only, `$schema`-checked; array-form `items` rejected with a `prefixItems` migration hint | Dialect drift (draft-07 vs 2020-12 `items`) is the #1 real-world validator bug; one dialect, checked, beats two dialects guessed | human (this doc) | design | med |
| Local `$ref` only (`#/...` RFC 6901 pointers + `$anchor`/`$dynamicAnchor`), non-local URI refs are errors | A pure builtin cannot fetch; pretending otherwise would be a silent fallback (CLAUDE.md §2) | human (this doc) | design | med |
| RE2 patterns via Go `regexp`, ECMA-262 divergences documented, unsupported constructs fail schema *compile* naming keyword + pattern | `pattern` is 17 census hits; the alternative (silent ECMA emulation) does not exist in RE2 | human (this doc) | design | med |
| Full error list (no short-circuit), fixed deterministic order, depth cap on `$ref` chains | Machines consume the whole list; ordering must not be map-iteration roulette; cycles must error, not overflow the stack | agent | design | low |
| `std/list.all` as a Go `_list_all` builtin (symmetric with `_list_any`), not a `.ail` negation of `any` | Symmetry + the codegen registry already models list HOF rows; the `.ail` fallback compiles (V13) if the sprint prefers zero new Go surface | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Go-backed engine under `internal/builtins` (not pure `.ail`)
- [x] Strict-by-default with `validateLenient` opt-out
- [x] Hand-rolled engine; no new third-party validation dependency
- [x] Draft 2020-12 core + validation vocabulary scope; `format` annotation-only
- [x] Local `$ref` only; non-local refs error
- [x] RE2 patterns with documented divergences and compile-time failure

All frozen at design time — no open human decision gates remain for the sprint.

## Solution Design

### Overview

One new std module, `std/jsonschema`, backed by three new pure Go builtins in `internal/builtins`. The engine is two-stage internally — **compile** the schema (strict keyword audit, `$ref`/pointer resolution, pattern pre-compilation) then **validate** the document against the compiled tree — but the stdlib surface exposes it as one-call pure functions, matching the `std/regex` two-stage philosophy (`compile -> Result`, then total matching) *folded into* the `validate` call, because the requested API is `validate(schema, doc) -> [ValidationError]`.

Errors are `{ path: string, keyword: string, message: string }` records. Instance-side failures point `path` into the document (`/decisions/0/budget/amount`); schema-side failures (unknown keyword, dangling `$ref`, uncompilable pattern, malformed keyword value) point `path` into the *schema* and name the offending keyword — both in-band, distinguishable by which root they address, stated in `message`.

Rider: one new `std/list` export, `all`, implemented as the Go twin of `_list_any`.

### Architecture

**Components:**

1. **`internal/builtins/jsonschema.go`** — builtin registration. Three specs, all `Module: "std/jsonschema"`, `IsPure: true`, types via `types.NewBuilder()`:
   - `_jsonschema_validate(schema: Json, doc: Json) -> [ValidationError]` — strict
   - `_jsonschema_validate_lenient(schema: Json, doc: Json) -> [ValidationError]` — unknown keywords skipped
   - `_jsonschema_check_schema(schema: Json) -> [ValidationError]` — self-check only (strict audit + `$ref` integrity + pattern compilability), so a build can validate its *schema* without an instance
   Registration follows the `_json_decode` pattern (`RegisterEffectBuiltin`, `BuiltinMetadata` with examples/since/tags).
2. **`internal/builtins/jsonschema_compile.go`** — schema → compiled tree. Strict keyword whitelist (see vocabulary below); ordered keyword lists per node (insertion order, **no map iteration later**); `$defs` handling; `$ref` resolution over RFC 6901 pointers with `~0`/`~1` unescaping; `$anchor`/`$dynamicAnchor` name registry; `regexp.Compile` of every `pattern`/`patternProperties`/`propertyNames` pattern **at schema compile time** (failure = schema error, never a runtime skip); boolean schemas (`true`/`false` as whole schemas).
3. **`internal/builtins/jsonschema_validate.go`** — instance walk. Fixed evaluation order per node (assertions, then applicators in compiled order, `unevaluated*` last); annotation tracking for `unevaluatedItems`/`unevaluatedProperties` (which properties/items `properties`/`patternProperties`/`additionalProperties`/`prefixItems`/`items`/`contains` evaluated); `$ref`/`$dynamicRef` with a per-instance-location depth cap (64) so cycles **error**, not overflow; JSON Pointer path construction; error list assembly in depth-first document order.
4. **`std/jsonschema.ail`** — the surface: `ValidationError` record type + three `pure` wrappers, one per builtin.
5. **`std/list.ail` + `internal/builtins/list_iterative_search.go`** — `all` export over a new `_list_all` HOF registration (same `registerListHOF` family as `_list_any`), plus its `registry_codegen_list.go` row.

**Supported vocabulary** (strict whitelist — anything else is an error):

- **Core**: `$schema` (must be the 2020-12 meta-schema URI if present), `$id` (relative forms only), `$ref` (local), `$defs`, `$anchor`, `$dynamicAnchor`, `$dynamicRef`, `$vocabulary` (recognized; unknown vocabulary → strict error), `$comment`. Boolean schemas.
- **Applicators**: `properties`, `patternProperties`, `additionalProperties`, `propertyNames`, `prefixItems`, `items` (2020-12: single schema applied to **all** items), `contains`, `allOf`, `anyOf`, `oneOf`, `not`, `if`/`then`/`else`, `dependentSchemas`, `unevaluatedItems`, `unevaluatedProperties`.
- **Validation**: `type` (incl. `integer` = zero fractional part, per 2020-12), `enum`, `const`, `multipleOf`, `maximum`, `exclusiveMaximum`, `minimum`, `exclusiveMinimum`, `maxLength`, `minLength`, `pattern`, `maxItems`, `minItems`, `uniqueItems`, `maxContains`, `minContains`, `maxProperties`, `minProperties`, `required`, `dependentRequired`.
- **Meta-data (accepted, never asserted)**: `title`, `description`, `default`, `deprecated`, `readOnly`, `writeOnly`, `examples`.
- **`format`**: annotation only (2020-12 default). Never asserts.

**Deliberate rejections with migration hints:** array-form `items` → error naming `prefixItems`; draft-07 `$schema` URI → error naming the 2020-12 URI; non-local `$ref` (any URI with a scheme/host) → error stating local-pointer-only.

**RE2 vs ECMA-262 patterns (documented, per the report's ask):** patterns compile under Go `regexp` (RE2). Divergences, shipped in `docs/docs/reference/std-jsonschema.md` and in the strict error text itself:
- Backreferences (`\1`) and lookaround (`(?=...)`, `(?!...)`, `(?<=...)`) — RE2 cannot express them; such a pattern is a **schema compile error** naming the keyword and the pattern, never a silent non-match.
- `\p{...}` uses Go/RE2 class syntax; `\u{...}`/`\x{...}` codepoint escapes differ in form.
- No `v` (`unicodeSets`) flag; `u` flag semantics are Go's, not ECMA's exactly.
- Both are unanchored by default (JSON Schema semantics); `^`/`$` match string ends without `(?m)`, matching ECMA without the `m` flag.
- Residual risk: a pattern RE2 accepts but compiles differently in an edge case (rare; mostly exotic Unicode classes). Accepted; the client census patterns were simple (`[a-z-]+`, ISO-date shapes).

**Numbers:** the `Json` ADT stores `JNumber(float64)` (since v0.2.0 decode). All numeric comparisons (`minimum`/`maximum`/`exclusive*`) are float64; `multipleOf` is `x/d` integral with float64 division. Deterministic (IEEE-754, same on every platform) but **not** arbitrary-precision: `0.1`-class edges may differ from a big.Rat validator. Documented as a known limitation; `big.Rat` re-parse is Future Work (the literal text is lost in the ADT — see V6).

### Implementation Plan

**Phase 1: Compile engine** (~1 day)
- [ ] `jsonschema.go`: three `BuiltinSpec` registrations, builder types, metadata
- [ ] `jsonschema_compile.go`: strict whitelist audit, ordered keyword lists, `$defs`/`$ref`/`$anchor` resolution, pattern pre-compile, boolean schemas
- [ ] Unit tests: unknown keyword (strict error), `~0`/`~1` escapes, dangling `$ref`, backref pattern rejected, array-`items` rejected with hint

**Phase 2: Validate engine** (~2 days)
- [ ] `jsonschema_validate.go`: assertions + applicators + `unevaluated*` annotation tracking, pointer paths, depth cap
- [ ] Unit tests: per-keyword table (positive + negative), `integer`-vs-`2.0`, `uniqueItems`, `oneOf` exclusive-match counting, `if`/`then`/`else`, `$ref` siblings evaluated per 2020-12 (ref does not replace siblings), cycle cap, determinism (permuted keys, 100 runs)

**Phase 3: Std surface + `all`** (~0.5 day)
- [ ] `std/jsonschema.ail` wrappers (pure); `std/list.ail` `all` + `_list_all` Go builtin + codegen registry row
- [ ] `.ail` tests for both; suite-count gate bump (V9)

**Phase 4: Docs, example, gates** (~1.5 days)
- [ ] `docs/docs/reference/std-jsonschema.md` (API + RE2 divergences + strict-mode contract), `stdlib.md` table row, `changelogs/v0.32-current.md` entry
- [ ] `examples/runnable/jsonschema_validate.ail` (envelope self-check, `verify-examples`-clean)
- [ ] `.stdlib-golden/jsonschema.{json,sha256}` golden (V10), census fixture pinned, benchmarks (`testing.B`) for the < 50 ms metric

### Files to Modify/Create

**New files:**
- `internal/builtins/jsonschema.go` (~120 LOC) — registration, types, metadata
- `internal/builtins/jsonschema_compile.go` (~450 LOC) — strict audit, `$ref`, pattern pre-compile
- `internal/builtins/jsonschema_validate.go` (~450 LOC) — assertions, applicators, pointers, ordering
- `internal/builtins/jsonschema_test.go` (~600 LOC) — keyword tables, census fixture, determinism, benchmarks
- `std/jsonschema.ail` (~60 LOC) — `ValidationError`, `validate`, `validateLenient`, `validateSchema`
- `tests/stdlib/jsonschema_test.ail` (~120 LOC) — `.ail` suite over the same fixtures
- `examples/runnable/jsonschema_validate.ail` (~50 LOC) — envelope self-check example
- `docs/docs/reference/std-jsonschema.md` (~150 LOC) — reference page incl. RE2-vs-ECMA table
- `.stdlib-golden/jsonschema.json` + `.sha256` — generated golden pair

**Modified files:**
- `std/list.ail` (+6 LOC) — `export pure func all`
- `internal/builtins/list_iterative_search.go` (+~30 LOC) — `_list_all` registration (or sibling file if that file is near the 800-line gate)
- `internal/builtins/registry_codegen_list.go` (+1 row) — `_list_all` codegen mapping
- `internal/builtins/list_test.go` or a new `list_all_test.go` (+~60 LOC) — `all` symmetry/edge tests
- `make/test.mk` (+1 LOC) — `STDLIB_AIL_SUITES_EXPECTED` 6 → 7 (V9)
- `docs/docs/reference/stdlib.md` (+1 table row)
- `changelogs/v0.32-current.md` (+ entry)

All new Go files stay under the 800-line `check-file-sizes` gate (V11) by the three-way split.

## Conflict Surface

Not a parser/typechecker/codegen change — no grammar production, AST shape, or pipeline pass is touched. The shared surfaces this *does* extend, and what already lives there:

| Surface | Existing occupants | Disambiguation / coexistence |
|---|---|---|
| Builtin registry (`internal/builtins`) | ~200 underscore builtins incl. `_json_decode`, `_json_encode`, `_json_repair`, `_regex_*` | `_jsonschema_*` names are unallocated (V3); no rename or override of any existing spec |
| Std module namespace (`std/*.ail`) | 40+ modules incl. `json`, `yaml`, `xml` | `std/jsonschema` is a new module name; no existing module or import path collides (V1) |
| `std/list` exports | `any` (line 156), `contains`, `member`, HOF family | `all` is a new symbol; `any` untouched; both engines share the same Go impl |
| `registry_codegen_list.go` rows | `_list_any → any` (line 184) | One new row `_list_all → all`; no row edited |

The one behavioral change to an *existing* error path: `import std/list (all)` previously failed with `IMP010: symbol 'all' not exported by 'std/list'`; it now resolves. That is the intent of the rider, not a regression.

**Programs that MUST still work (regression fixtures):**
- `tests/stdlib/list_drop_test.ail`, `tests/stdlib/list_iterative_test.ail` — `std/list` surface unchanged plus `all`
- `tests/json_minimal_test.ail`, `tests/json_accessors_test.ail` — `std/json` untouched; `validate` consumes the same ADT
- `examples/runnable/json_basic_decode.ail`, `examples/runnable/json_array_extraction.ail` — decode/encode paths untouched
- `std/regex.ail` consumers — pattern semantics untouched (the validator uses Go `regexp` directly, not `std/regex`)

**What deliberately changes:** nothing existing. New imports (`std/jsonschema`) resolve where they previously failed at module-not-found; strict mode rejects schemas carrying out-of-vocabulary keywords — schemas that no current in-repo code validates.

## Examples

### Example 1: The pack build self-check (the reported use case)

**Before (stopgap, clients repo):** ~150 lines of bespoke `.ail` per client schema, subset drift per repo, no shared tests; the 4+6 envelope violations of 2026-10-19 were found by *outside* Python `jsonschema`.

**After (in-language, once `std/jsonschema` ships):**

```ailang
module client/check
import std/json (Json, decode)
import std/jsonschema (validate, ValidationError)
import std/list (length)
import std/result (Result, Ok, Err)

func check(schemaText: string, envelopeText: string) -> Result[(), string] {
  match decode(schemaText) {
    Err(e) => Err("schema decode failed: ${e}"),
    Ok(schema) => match decode(envelopeText) {
      Err(e) => Err("envelope decode failed: ${e}"),
      Ok(doc) => {
        let errors = validate(schema, doc);
        match length(errors) {
          0 => Ok(()),
          _ => Err("schema violations: ${show(errors)}")
        }
      }
    }
  }
}
export func main() -> () ! {} = ()
```

Syntax of this exact shape is check-verified against a stand-in module (V4); the `_jsonschema_*` calls cannot type-check until the builtin exists in the same binary (V5) — which is why Phase 3 lands std + Go registration atomically.

An error comes back as, e.g.:

```ailang
{ path = "/decisions/0/budget/amount", keyword = "exclusiveMinimum",
  message = "value 5001 must be strictly greater than 10000 at /$defs/money/exclusiveMinimum" }
```

### Example 2: Strict mode refusing what it cannot check

```ailang
let schema = decode("{ \"algo\": { \"patternFormat\": \"ecma\" } }");
-- validate(schema, doc) returns:
-- [ { path = "/properties/algo/patternFormat",
--     keyword = "patternFormat",
--     message = "strict mode: not in the supported Draft 2020-12 vocabulary; use validateLenient to skip unknown keywords" } ]
```

The schema that outgrows the validator fails **at the schema**, loudly, naming the keyword — never a silent pass.

### Example 3: `std/list.all`

**Before:** `IMP010: symbol 'all' not exported by 'std/list'` (V2).

**After:**

```ailang
import std/list (all)
all(\b. b, [true, true])        -- true
all(\b. b, [true, false])        -- false
all(\b. b, [])                   -- true  (vacuous, symmetric with any([]) == false)
```

The definition compiles today as a `.ail` fallback (V13): `all[a](p, xs) = !any(\x. !p(x), xs)` — the chosen Go twin is symmetric with `_list_any` and avoids the closure per element.

## Success Criteria

- [ ] `std/jsonschema` exports `ValidationError`, `validate`, `validateLenient`, `validateSchema`; all `pure`; `ailang check std/jsonschema.ail` green
- [ ] All 22 census keywords validated correctly (Go table tests + census fixture, error set pinned)
- [ ] Strict mode: unknown keyword → ≥ 1 error naming it; `validateLenient` skips it; assertion that no schema-valid doc returns errors under a fully-understood schema
- [ ] Determinism: 100 runs byte-identical; permuted schema keys → identical error list (test-pinned)
- [ ] `$ref`: `#/$defs/...` resolution, `~0`/`~1` escapes, dangling-ref error, depth cap on cycles, sibling evaluation per 2020-12
- [ ] Patterns: RE2 compile at schema-compile; backref/lookaround → schema error naming keyword + pattern; divergences table in `std-jsonschema.md`
- [ ] `std/list.all` exported; `any`/`all` symmetry + empty-list edges pinned in Go and `.ail` tests
- [ ] Gates: `make test`, `test-stdlib-ail` (SUITES 6→7), `verify-stdlib` (new golden), `verify-examples` (new runnable), `check-file-sizes`, `check-boundaries` — all green
- [ ] Docs: `std-jsonschema.md` + `stdlib.md` row + `changelogs/v0.32-current.md` entry
- [ ] Benchmark: census-shaped schema × 1,000-property doc in < 50 ms (reported in the implementation report)
- [ ] Reporter follow-up delivered: stopgap `schemacheck.ail` in the clients repo can now be replaced by `std/jsonschema` (out of this repo's scope; noted to the coordinator)

## Testing Strategy

**Unit tests (Go, `internal/builtins/jsonschema_test.go`):**
- Per-keyword positive/negative tables for the full supported vocabulary
- Census fixture: synthetic schema exercising all 22 reported keywords in census proportions; expected error list pinned (a stand-in for the real envelope — the client schema lives in another repo)
- Strict-mode: unknown keyword, unknown `$vocabulary`, draft-07 `$schema`, array-`items` rejection
- `$ref`: escapes, dangling target, cycle depth cap, sibling evaluation
- Determinism: permuted schema keys, repeated runs, Go-map-iteration-proof assertion (identical lists)
- Benchmarks: `testing.B` on census-shaped schema × 1,000-property doc

**Integration tests (`.ail`):**
- `tests/stdlib/jsonschema_test.ail`: decode → `validateSchema` (schema asserts clean) → `validate` on good and bad envelopes; assert on `path`/`keyword` fields
- `std/list` `all`: symmetry with `any`, vacuous-empty case

**Gate coverage:** `test-stdlib-ail` (suite count 6→7, V9), `verify-stdlib` (new golden pair required — an unfrozen module *fails* the gate, V10), `verify-examples` (new runnable example), `check-file-sizes`, `check-boundaries`.

**Manual:** REPL walkthrough of Example 1; `ailang builtins list | grep jsonschema` shows the three new `[pure] std/jsonschema` rows.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact error `message` wording — agent may choose (only `path`/`keyword` fields are contractual)
- File split inside `internal/builtins` if a compile/validate file approaches the 800-line gate — agent may choose
- Test fixture organization (one table vs per-keyword funcs) — agent may choose
- `$dynamicRef` scope-stack depth handling beyond the base pointer semantics — agent may choose within 2020-12 rules, documented in the implementation report
- Whether `all` lands as `_list_all` (Go) or the `.ail` negation (V13) if the sprint prefers zero new Go builtin surface — agent may choose; default is the Go twin
- Vendoring a curated slice of the official JSON-Schema-Test-Suite as fixtures — human at review (repo weight vs coverage; the census fixture is the M1 bar)
- `maxErrors` / early-exit option — human at review, only if a caller exhibits a need

## Non-Goals

**Not attempted in this feature:**
- **Remote `$ref` / network fetch** — a pure builtin cannot, and must not, fetch; URI refs error (A4/A12)
- **`format` assertions** — 2020-12 makes format annotation-only by default; asserting date/time formats is a separate vocabulary decision
- **Other dialects (draft-07, 2019-09)** — rejected with a pointed error, not emulated by guessing
- **Wrapping a third-party validator** — rejected (see High-Impact Decisions); `google/jsonschema-go` stays what it is today: MCP tool-schema *construction* (V12)
- **Schema-driven codegen / CLI subcommand** — the stdlib surface is the whole deliverable
- **Migrating the clients-repo stopgap** — the clients repo consumes this after release; this doc ships the module only

## Timeline

**Week 1** (3.5 days):
- Phase 1: compile engine + strict audit tests (1d)
- Phase 2: validate engine + keyword tables (2d)
- Phase 3: std surface + `all` (0.5d)

**Week 2** (1.5 days):
- Phase 4: docs, example, goldens, suite-count bump, gates, benchmarks (1.5d)

**Total: ~5 days across 2 weeks** (estimate already 2× the naive one)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `unevaluatedItems`/`unevaluatedProperties` annotation semantics are the hardest part of any 2020-12 validator | High | Single-pass ordered evaluation with per-location annotation tracking; suite-derived fixtures for exactly these two keywords; M2 test budget reserved |
| RE2 ≠ ECMA-262 on some real client pattern | Medium | Compile-time failure names keyword + pattern (never a silent skip); divergence table shipped in the reference page; census patterns reviewed simple |
| Float64 numeric edges (`0.1`-class `multipleOf`) diverge from arbitrary-precision validators | Medium | Documented limitation (matches the `Json` ADT since v0.2.0); deterministic IEEE-754 semantics; `big.Rat` re-parse is Future Work |
| Strict mode rejects benign vendor annotations (`x-*`) in third-party schemas | Medium | That is the requested default; `validateLenient` is the documented opt-out; the in-repo census showed none |
| Deep documents + `$ref` chains recurse past stack limits | Medium | Depth cap per instance location (64) → deterministic error, mirroring the depth discipline of the iterative list builtins |
| The `.ail` wrapper cannot type-check until the Go builtin exists in the binary (V5) | Low | Std + Go registration land atomically in one PR; gates (`verify-stdlib`, `test-stdlib-ail`) run against the built binary by construction |

## Verification Log

| # | Claim | Evidence | Status |
|---|-------|----------|--------|
| V1 | The stdlib has no JSON Schema validation module | `ls std/` — 40+ modules, no schema-validation surface; `grep -rin jsonschema std/ internal/builtins/*.go` → empty | Confirmed |
| V2 | `std/list` exports `any` but not `all` | `std/list.ail:156` `export pure func any[a](p: a -> bool, xs: [a]) -> bool = _list_any(p, xs)`; `grep -n "func all" std/list.ail` → exit 1 | Confirmed |
| V3 | `_jsonschema_*` builtin names are unallocated | `ailang builtins list \| grep jsonschema` → empty; grep over `internal/builtins/` → empty | Confirmed |
| V4 | The Example 1 client syntax (imports, nested match, `"${...}"` interpolation) type-checks | `ailang check` on a stand-in `std/jsonschema` (pure-`.ail` stub) + `client.ail` → `✓ No errors found!` exit 0 (MOD010 temp-path auto-relax warning only, `/tmp` sandbox) | Confirmed |
| V5 | The `.ail` wrapper cannot land without the Go builtin in the same binary | `ailang check` on the stub once it calls `_jsonschema_validate` → `type error ... '_jsonschema_validate' follows the builtin naming convention; builtins are compiled into this ailang binary, not importable source` — the typechecker enforces builtin existence against the *binary* | Confirmed |
| V6 | `Json` numbers are `JNumber(float64)`, literal text lost at decode | `std/json.ail` ADT (`JNumber(float)`); `internal/builtins/json_decode.go` metadata: "Every number literal decodes to its nearest float64"; `1e400` is an `Err`, never silent Infinity | Confirmed |
| V7 | Pure Go builtin registration pattern exists to copy | `internal/builtins/json_decode.go`: `RegisterEffectBuiltin(BuiltinSpec{Module: "std/json", Name: "_json_decode", IsPure: true, Type: makeJSONDecodeType, Impl: ..., Metadata: ...})`; `ailang builtins list` shows `_json_decode [pure] std/json` | Confirmed |
| V8 | Go-side `Json` ADT walking precedent (records, tagged unions) | `internal/builtins/json_encode.go:173-202` walks `JObject`/`JArray` tagged values and `{key, value}` `RecordValue`s; `internal/builtins/regex.go:285` constructs a `RecordValue` return (the `ValidationError` shape has direct precedent) | Confirmed |
| V9 | Adding a stdlib `.ail` suite requires the suite-count gate bump | `make/test.mk:369` `STDLIB_AIL_SUITES_EXPECTED := 6` (currently 6 suites in `tests/stdlib/`); a 7th file fails the gate until the var is edited — the failure text itself instructs this | Confirmed |
| V10 | A new std module requires a golden pair or `verify-stdlib` fails | `tools/verify-stdlib.sh`: "fails ... ALSO if a module has no golden at all — an unfrozen module is uncovered, not 'passing'"; goldens tracked in git (`.stdlib-golden/`, 98 files incl. `json.json`) | Confirmed |
| V11 | New Go files must stay < 800 lines | `make/code-health.mk:164` `check-file-sizes` — CI gate over >800-line files; hence the three-way engine split | Confirmed |
| V12 | `google/jsonschema-go` is already a direct dep but is used for MCP tool-schema construction, not validation | `go.mod` v0.4.3; importers: `cmd/ailang-microrag-mcp/main.go:25`, `internal/apiserver/feedback_tool.go:8` (`&jsonschema.Schema{Properties: ...}` — building tool input schemas) | Confirmed |
| V13 | A pure-`.ail` `all` fallback compiles | `export pure func all[a](p: a -> bool, xs: [a]) -> bool = !any(\x. !p(x), xs)` appended to a std copy + `ailang check` → `✓ No errors found!` exit 0 | Confirmed |
| V14 | The 4+6 envelope violations of 2026-10-19 and the keyword census | Reported in the message (`decision-envelope.schema.json` lives in the sunholo-data clients repo, not here); counts are inputs to this design, not re-derived | **Reported** (external; not re-verifiable in this repo) |
| V15 | New runnable examples are gated | `make/examples.mk:18` `verify-examples` runs `examples/runnable/` and fails on any red example | Confirmed |

**Quorum:** not run — unattended coordinator task filing a P1 design doc whose premises are all in-repo-verified above (V14 marked Reported, not load-bearing for the API shape: the vocabulary is the full 2020-12 set, not just the census). `ailang design-quorum` remains available to a reviewer before sprint start if one is wanted.

## Related Documents

**Implemented (may inform design):** (none found — both SimHash and neural search returned no matches for "std jsonschema")

**Planned (check for overlap):** (none found)

Adjacent, read for context:
- [m-builtin-classification-surface](../m-builtin-classification-surface.md) — the builtin inventory this module extends (three new `[pure] std/jsonschema` rows become classifier input)
- [m-validator-check-budget-parity](../v0_56_1/m-validator-check-budget-parity.md) — a different "validator" (registry publish-gate); no overlap
- `design_docs/archive/2026-07/M-TOOLING-DETERMINISTIC.md` — prior art note that once proposed `xeipuuv/gojsonschema` for *tool* schemas; superseded here by the hand-rolled decision (V12)

## References

- [Design Axioms](/docs/docs/references/axioms.mdx) — the 12 non-negotiables scored above
- JSON Schema Draft 2020-12: core + validation specifications (the dialect this module implements)
- RFC 6901 — JSON Pointer (`path` format, `~0`/`~1` escaping)
- [std/regex](../../std/regex.ail) — the RE2 engine contract this design's pattern handling follows (compile-time failure, never silent skip)
- Reported message: `ailang messages` (cli), 2026-11-03, "std/jsonschema — validate a Json against a JSON Schema (2020-12)"

## Future Work

- `big.Rat` numeric comparisons if a client schema needs exact decimal `multipleOf` (requires re-parsing literals before the ADT loses them — see V6)
- `format` assertions behind an explicit opt-in vocabulary flag, should a client schema actually assert formats
- Official JSON-Schema-Test-Suite vendoring (curated) if coverage demands outgrow the census fixture
- `$id`-based base-URI resolution beyond local fragments, if a real schema needs it (still never remote fetch)

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03