# M-JSON-CODECS: typed JSON codecs as values, for typed messages and typed model output

**Status**: Planned (r3; Quorum guardrail spent: r1 and r2 both BLOCKED; every r2 objection is dispositioned below with new evidence rows. The sprint-planner re-verifies at plan time. Design Freeze open for Mark)
**Target**: v1.0.0, clause 4
**Priority**: P0 for v1.0. [m-typed-message-plane](m-typed-message-plane.md) M3 needs it, and so does the flagship's typed model output.
**Estimated**: ~1 week (std types and functions 1d, generator 2–3d, one schema mapper 1d, tests and docs 1–2d)
**Dependencies**: none for the std functions, generator and `callTyped` (V1–V12). The serve-api/MCP schema switch ships behind [m-typed-message-plane](m-typed-message-plane.md) M5's CI diff.
**Created**: 2026-10-09 · **Author**: attended session with Mark (Claude Opus 5.5)
**Quorum trigger**: #1 (Design Freeze) and #2 (it changes a shared mapper) → the quorum runs.

---

## Problem

Typed model output and typed messages both need an AILANG type ↔ JSON mapping with a schema. Today:

- **It is hand-written.** `callJson(string, string) -> string ! {AI}` takes a hand-written schema
  string and returns a string, and decoding one record through `std/json` accessors took ~15 lines
  in the 2026-10-09 capability tests (V1).
- **Deriving cannot produce it.** `deriving (Json)` fails with `PAR_DERIVING_UNSUPPORTED … only 'Eq'
  is currently supported` (V2).
- **The repo has two type → schema mappers that disagree** (V7), so a third encoder per feature would
  add duplication.

## Why codecs are values, not type-directed

The r1 draft proposed `decodeAs(s) -> Result[T, JsonError]` with `T` chosen from the annotation,
"the way Eq dictionaries are resolved". Quorum r1 rejected that 3/3, and checking showed why:

- AILANG has **no call-site type application**: `idt[int](3)` is a type error (V3).
- The dictionary registry's built-in classes are `Eq` and `Num` only (V4).
- Choosing an instance from a *return* type is unproven machinery.

What **does** work today is a codec as an ordinary value. A record of functions, passed explicitly,
type-checks and runs (V5):

```ailang
export type JsonError = { path: string, expected: string, got: string }
export type JsonCodec[a] = { encode: a -> Json, decode: Json -> Result[a, JsonError], schema: string }
export func decodeWith[a](c: JsonCodec[a], s: string) -> Result[a, JsonError] = ...
-- decodeWith(pJson(), "{\"name\":\"Ann\"}")  →  Ok({ name: "Ann" })   (ran, V5)
```

No type-system change is needed. The type checker already checks that `pJson()` is a
`JsonCodec[P]` and that `decodeWith` returns `Result[P, JsonError]`.

## Design

**1. std types and functions** in `std/json`. None of these names exists today (V8).
- `type JsonError = { path: string, expected: string, got: string }`
- `type JsonCodec[a] = { encode: a -> Json, decode: Json -> Result[a, JsonError], schema: string }`
- `encodeWith[a](c: JsonCodec[a], v: a) -> string`
- `decodeWith[a](c: JsonCodec[a], s: string) -> Result[a, JsonError]`

**2. Typed model output** in `std/ai`.
- `callTyped[a](prompt: string, c: JsonCodec[a]) -> Result[a, AIError] ! {AI}` sends `c.schema` to the
  provider's existing structured-output path (V11), then decodes the reply with `c.decode`.
- A decode failure is `AIError{code: "decode", message: "<path>: expected <x>, got <y>", retryable:
  false}`, using the existing `AIError` record (V10). It is never a panic and never a silently
  defaulted field.

**3. Codecs are generated as source, not by the compiler.**
- `ailang generate-json-codecs [--dry-run] <file.ail> <Type>…` writes `<module>_json.ail` next to the
  module, holding `<lowerType>Json() -> JsonCodec[Type]` for each named record or ADT.
- This follows the precedent of `ailang generate-extension-registry`, which writes AILANG source and
  has a `--dry-run` (V6).
- The generated file is ordinary AILANG. The normal type checker, effect checker, IFC labels and
  `ailang verify` all apply to it, and a reviewer can read it.
- Field types must be `int`, `float`, `string`, `bool`, `bytes`, a list, `Option`, or another type the
  same run generates a codec for. Anything else is a generator error naming the field.
- Polymorphic types are refused, matching `deriving (Eq)`'s existing rule (V9).
- An in-language `deriving (Json)` that emits the same code can follow later (F1). The generated shape
  is the contract either way.

**4. One mapper.**
- The generator gets the schema string from `internal/apiserver/schema.TypeToSchema` (V7), extended
  with the ADT rule below and a resolver for named types (the generator holds the parsed module, so it
  can expand them).
- `mcp.go`'s `ailangTypeToJSONSchema` duplicate delegates to it ([m-typed-message-plane](m-typed-message-plane.md) M5).

**Encoding** (records and primitives follow `TypeToSchema`'s existing rules, V7):

| AILANG | JSON |
|---|---|
| `int` / `float` / `string` / `bool` | integer / number / string / boolean |
| `bytes` | base64 string |
| `[T]` | array |
| record | object with exactly those keys; unknown keys rejected on decode (F3) |
| `Option[T]` | `oneOf [null, T]`; the key is present. This is `TypeToSchema`'s **existing** `Option` rule (V12), unchanged |
| ADT `A \| B(x)` | `{"tag":"A"}` / `{"tag":"B","value":…}` (F2) |

## Conflict Surface

| Question | Answer |
|---|---|
| Positions extended | `std/json` and `std/ai` exports (new names); a new CLI command; the `TypeToSchema` mapper |
| What already lives there | `std/json`'s `encode` / `decode` / `get` / `as*` (unchanged); `std/ai`'s `call*` / `step` / `runTools` (unchanged); `TypeToSchema` callers in serve-api (`RequestSchema` / `ResponseSchema`, `schema.go:287,311`) |
| How it disambiguates | Additive names only (V8); no parser, elaborator or type-checker change; the mapper's existing outputs are unchanged for every type it handles today, and only named ADTs gain a real schema |
| Must still work | Every existing `std/json` and `std/ai` user; serve-api request/response schemas for existing exports (snapshot test on current output before the change) |
| Deliberate changes | Named-ADT parameters in serve-api/MCP schemas change from an undescribed object to a tagged `oneOf`; this ships behind the m-typed-message-plane M5 CI diff |

## Design Freeze

- [ ] **F1. Generator now, `deriving (Json)` later:** recommend yes. It needs no compiler change, the
  output is reviewable, and there is precedent (V6).
- [ ] **F2. ADT encoding:** recommend `{"tag","value"}`, which maps cleanly to JSON Schema `oneOf` and
  to model structured-output modes.
- [ ] **F3. Unknown keys on decode:** recommend reject.

## Success Criteria

- [ ] `generate-json-codecs` on `type P = { name: string, age: int }` produces a codec whose
  `decodeWith` round-trips. `{"name":"A","age":"thirty"}` returns `JsonError{path:"age",
  expected:"int", got:"string"}`.
- [ ] An unsupported field type is a generator error naming the field; a polymorphic type is refused.
- [ ] `callTyped(prompt, pJson())` under `--ai-stub-fixtures` returns `Ok(P)` for a good fixture and
  `Err(AIError{code:"decode"})` for a bad one.
- [ ] Property test: for generated codecs, `c.schema` equals `TypeToSchema(T)`.
- [ ] A snapshot test of serve-api schemas for existing exports is unchanged except for named ADTs.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Encoding is a total function of the value |
| A2 Replayability | +1 | Typed values serialise canonically |
| A3 Effect Legibility | 0 | `callTyped` keeps `! {AI}` |
| A4 Explicit Authority | +1 | A codec is an explicit argument, not ambient resolution |
| A5 Bounded Verification | +1 | Generated codecs are ordinary checked AILANG |
| A6 Safe Concurrency | 0 | — |
| A7 Machines First | +1 | Removes ~15 lines of hand-decoding per record |
| A8 Minimal Syntax | +1 | No syntax at all |
| A9 Cost Visibility | 0 | — |
| A10 Composability | +1 | One codec type serves model output, messages, serve-api and MCP |
| A11 Structured Failure | +1 | `JsonError` and `AIError{code:"decode"}` |
| A12 System Boundary | +1 | Every JSON boundary gets a declared type |

**Net +9.** No −1.

## Changes since r1 (quorum BLOCKED 3/3, 2026-10-09)

- **All three reviewers (return-type resolution unverified and undesigned):** removed. Codecs are
  explicit values (V5). The absence of call-site type application (V3) and the built-in-only class
  registry (V4) are now logged.
- **glm (`decodeAs[P]` notation never designed):** that notation is gone; every call passes a codec value.

## Round-2 objections and dispositions (quorum r2 BLOCKED, 2026-10-09; guardrail spent)

- **gemini, kimi (`AIError` shape unverified):** `AIError = { code: string, message: string,
  retryable: bool }` exists (V10). The path rides in `message`.
- **kimi (structured-output mode unverified):** providers already send a schema: Anthropic
  `ResponseSchema` and config-driven `response_format` `json_schema` (V11).
- **kimi ("Dependencies: none" vs the M5 diff):** the header now names the dependency.
- **glm (`Option` needs a third mapper rule):** it doesn't. `TypeToSchema` already maps `Option[T]`
  to `oneOf [null, T]` (V12). The table now uses that rule; "key optional" was wrong and is removed.

## Verification Log (2026-10-09, `ailang v0.52.3-44`, `origin/dev` `c92739681`)

| # | Claim | Evidence |
|---|---|---|
| V1 | Typed output is hand-decoded today | `ailang docs std/ai`: `callJson(string, string) -> string ! {AI}`; capability test `typed1.ail` (~15 lines of nested matches per record) |
| V2 | Only `Eq` can be derived | `der1.ail` `… deriving (Json)` → `PAR_DERIVING_UNSUPPORTED … only 'Eq' is currently supported` |
| V3 | No call-site type application | `tapp.ail`: `func idt[a](x: a) -> a = x` and `idt[int](3)` → `type unification failed … cannot unify function type with int` |
| V4 | Built-in classes only | `internal/types/dictionaries.go:83` `registerBuiltins` registers `Eq` and `Num`; `poly1.ail`: generic `==` over a `deriving (Eq)` ADT → ✓ No errors found! |
| V5 | Explicit codecs work today | `codec1.ail` declares `JsonCodec[a]` as above, `pJson() -> JsonCodec[P]` and `decodeWith[a]`; `ailang check` → ✓ No errors found!; `ailang run --caps IO --entry roundtrip` → `Ann` |
| V6 | Source-generator precedent | `ailang generate-extension-registry --help`: `-config`, `-dry-run` ("Print generated file to stdout instead of writing"), `-output` |
| V7 | Two mappers; the rich one's rules | `internal/apiserver/schema/schema.go:64` `TypeToSchema` (primitives, `[T]`, records, tuples, type applications; named type → `{"type":"object","description":"AILANG type: …"}`); `:287,311` `RequestSchema` / `ResponseSchema`; `internal/apiserver/mcp.go:581-582` |
| V8 | New names are free | `git show origin/dev:std/json.ail \| grep -E "JsonCodec\|JsonError\|encodeWith\|decodeWith"` → empty |
| V10 | `AIError` exists with a `code` field | `ailang docs std/ai`: `export type AIError = { code: string, message: string, retryable: bool }` |
| V11 | Providers already pass a response schema | `internal/ai/anthropic/client.go:275-276` (`if req.ResponseSchema != "" { schema = json.RawMessage(req.ResponseSchema) }`); `internal/ai/configdriven/shapes.go:49-54` (`response_format` with type `json_schema`) |
| V12 | `Option` is already mapped | `internal/apiserver/schema/schema.go:240-247`: `case "Option"` → `{"oneOf": [{"type":"null"}, typeToSchema(arg)]}` |
| V9 | `deriving (Eq)` refuses polymorphic types | `internal/elaborate/file_funcs.go:164-168` ("cannot derive Eq for polymorphic type … deferred") |

---

**Document created**: 2026-10-09 (r2)
