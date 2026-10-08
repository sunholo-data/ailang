# M-STD-YAML-ENCODE — `std/yaml` encode: the `Json` ADT → idiomatic block YAML

**Status**: Planned
**Target**: v0.53.0 (next minor planning folder; P2, no release commitment)
**Priority**: P2 (Medium) — additive stdlib feature, no consumer blocked
**Estimated**: ~1 day (≈4h Go builtin + tests, ≈2h stdlib surface + `.ail` tests, ≈2h docs/golden/example)
**Dependencies**: `std/json`'s `Json` ADT and `kv`/`jo`/`js`/`jint` constructors (existing); `internal/builtins/yaml.go` (existing file, extended); **no new Go dependency**. Soft dependency: the decode-side key-order work item (triage Primary) decides which form of the round-trip caveat is live — see *Round-Trip Contract*.
**Source**: [ailang-core triage, Secondary 2](../ailang-core-triage/yaml-decode-loses-mapping-key-order.md) (merged on `dev`, PR #1620; recommendation: design-doc). Original user report: `inbox_1791394438696_00b01c33` (via ailang-core). The predecessor doc [m-std-yaml](../../implemented/v0_30_0/m-std-yaml.md) explicitly deferred YAML emission to Future Work ("`std/yaml.encode(j: Json) -> Result[string, string]`"); this doc picks that up.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure total function of the ADT value; one fixed output dialect (always-quote strings, fixed indent), byte-stable and golden-pinned — no host-map iteration order anywhere in the emit path. |
| A2: Replayability | 0 | Pure builtin; no trace impact. |
| A3: Effect Legibility | +1 | Zero effects, `IsPure: true`, same as `_yaml_to_json`/`_json_encode` — no hidden IO. |
| A4: Explicit Authority | +1 | No ambient authority; the caller supplies the value, output goes to the returned string. Reads no files/env. |
| A5: Bounded Verification | +1 | Output is locally checkable in-module: `decode(encode(j))` is a runnable round-trip property (see Testing Strategy). |
| A6: Safe Concurrency | 0 | No concurrency surface. |
| A7: Machines First | +1 | One tested builtin replaces hand-concatenated YAML strings (indentation and scalar-escaping fumbles by agents); uniform quoting removes per-scalar judgement. |
| A8: Minimal Syntax | 0 | No new syntax; one new builtin + one stdlib export. |
| A9: Cost Visibility | 0 | O(n) in the ADT size; no surprising cost. |
| A10: Composability | +1 | Composes with the whole `std/json` constructor/accessor surface (`jo`, `kv`, `get`, …) and with `std/yaml.decode` (`decode ∘ encode` is the contract). |
| A11: Structured Failure | +1 | The only unrepresentable values (NaN/±Inf) return typed `Err(string)` — no silent coercion to `null`, no panics. |
| A12: System Boundary | +1 | ADT→YAML-text is an explicit boundary crossing at a named function, mirroring `decode` on the other side. |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — the emitter walks an ordered `ListValue`; no Go map is ever constructed.
- [x] A3 (Effects): No hidden side effects — pure builtin.
- [x] A4 (Authority): No ambient access granted.
- [x] A7 (Machines First): Uniform quoting is chosen *for* machine legibility over human-pretty plain scalars (documented trade-off, see Dialect).

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`std/yaml` ([std/yaml.ail](../../../std/yaml.ail)) is one-directional: it exposes only `yamlToJson` and `decode`. Reading YAML into AILANG is a one-builtin bridge; **writing YAML back out has no std path at all.** A program that decodes a config, edits a field, and wants to re-emit YAML has exactly two options today:

**Current State:**

- **Workaround 1 — JSON flow style.** `std/json.encode` produces text that is *valid* YAML 1.2 (JSON ⊂ YAML), so the bytes technically round-trip through any YAML tool — but it is not the YAML anyone means: flow style, quoted keys, one line, no block structure. Verified live this session: `encode(jo([kv("title", js("Fysik A")), kv("year", jint(2026))]))` → `{"title":"Fysik A","year":2026}`.
- **Workaround 2 — hand-assembly.** Emitting block YAML means string-concatenating hand-managed indentation, hand-quoted scalars and hand-joined newlines (the JSON-quoted-scalar hand-emission the triage records). Nothing checks the result until the next `decode` — every indentation slip and every un-quoted `true`/`42`/`null`/`a: b` scalar is silent output corruption.

**Impact:**

- The reporting user session (`inbox_1791394438696_00b01c33`) hit this directly.
- Any agent pipeline that produces YAML at a system boundary — configs, tool manifests, benchmark specs, CI definitions — re-implements the same quoting rules per program, untested. This is the mirror of the gap `m-std-yaml` closed on the ingestion side in v0.30.0.

## Goals

**Primary Goal:** Ship `std/yaml.encode(j: Json) -> Result[string, string]` — one pure builtin emitting deterministic, idiomatic block-style YAML — with a total, documented round-trip contract against `std/yaml.decode`.

**Success Metrics:**

- `decode(encode(j)) == Ok(j)` for every finite `j` with unique-key objects, **modulo the documented `JObject` key-order caveat** (exact once the sibling decode fix lands) — pinned as a runnable property test.
- `encode` preserves `JObject` insertion order in the emitted bytes (never sorts) — pinned byte test.
- Every emitted fixture in the test corpus re-parses through the *existing* `_yaml_to_json` unchanged — acceptance test in Go against the real impl.
- One new pure Go builtin (`_yaml_encode`), zero new Go dependencies, WASM-portable (pure Go; the emitter does not even use yaml.v3).
- Docs updated: `docs/docs/reference/std-yaml.md`, stdlib index, changelog entry.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| **D1: Input is the `Json` ADT, not arbitrary AILANG values.** | The public surface. Arbitrary values would need a record→mapping policy, a rejection policy for lambdas/refs/channels/other ADTs, and lose the free round-trip with `decode` (whose output *is* `Json`). Reversing later re-opens the whole std surface. | human | design | high |
| **D2: Hand-written block-style emitter; NOT `yaml.Marshal` of a reconstructed `map[string]interface{}`, and NOT a `yaml.Node` tree.** | `yaml.Marshal` on a Go map **sorts keys** — it would destroy `JObject` order at the emit side and permanently break the round-trip contract (the triage Primary exists precisely because order loss hurts). `yaml.Marshal(yaml.Node)` preserves order but inherits emitter internals we cannot pin from this repo (80-column wrapping, implicit tag choice, block-literal selection) and which shift with yaml.v3 upgrades. A hand emitter over the ordered `ListValue` walk gives one deterministic byte shape we own. | agent | design | med |
| **D3: Round-trip contract: `decode(encode(j)) == Ok(j)` for every finite `j`, modulo `JObject` key order (see *Round-Trip Contract* for the sibling interaction).** | This is what callers will rely on; the key-order caveat's wording is entangled with the decode-side work item. | human | design | high |
| **D4: NaN/±Inf → `Err`, not `.nan`/`.inf`, and not `"null"`.** | `eval.FormatJSONNumber` already returns `"null"` for non-finite floats (read this session) — emitting that would be a silent data-destroying fallback. `.nan`/`.inf` would be honest YAML but output our own `decode` rejects (it bridges through `json.Marshal`, which errors on non-finite), breaking round-trip. `Err` mirrors `decode`'s documented rejection of the same values and keeps the guarantee exception-free. | human | design | med |
| **D5: Builtin `_yaml_encode(j: Json) -> Result[string, string]`, std export `encode`.** | Mirrors `std/json.encode` naming and the module's own `decode`; `_yaml_to_json`/`_json_encode` establish the underscore-builtin + thin-wrapper pattern. | agent | design | low |
| **D6: Output dialect: block style, indent 2, strings and keys ALWAYS double-quoted, numbers plain via `FormatJSONNumber`, `null` plain, flow style only for empty collections.** | The always-quote policy removes the entire "which plain scalars are ambiguous" decision tree (`true`, `42`, `null`, `a: b`, `#`, leading spaces, `---`…) from the emitter — every string scalar is unambiguous *by construction*, not by case analysis. | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — input type: the `Json` ADT (this doc; **requires user approval** before sprint execution).
- [x] D2 — emitter approach: hand-written ordered walker.
- [x] D3 — round-trip contract wording incl. key-order interaction.
- [x] D4 — NaN/±Inf: `Err`, fail loudly.
- [x] D5 — names: `_yaml_encode` / `std/yaml.encode`.
- [x] D6 — dialect pinned (byte level, below).

*(No unresolved high-cost items; all six are decided in this doc. D1/D3/D4 are marked "Chosen By: human" — they ride the design-doc approval.)*

## Conflict Surface

**Not a parser/typechecker/codegen change** — no syntactic or semantic position is extended; no disambiguation logic changes. The touched namespaces and the honest enumeration:

- **Builtin name registry (shared):** `_yaml_encode` — repo-wide grep finds no allocation (Verification Log, row 9); registration goes through the modern `RegisterEffectBuiltin` path only, which also feeds the VM adapter and `builtin_types.golden` (one added line — regenerated, not hand-edited).
- **`std/yaml` export surface:** new name `encode`; existing exports `yamlToJson`/`decode` untouched. `std/json`'s `encode` is a *different module's* export — no collision (the only cross-module names both export are types).
- **No existing construct changes meaning.** Output of `_yaml_to_json`, `std/json.encode`, `std/json.decode` is byte-identical before/after; nothing previously valid becomes invalid.

**Programs that MUST still work (regression fixtures, all verified present this session):**
- [examples/runnable/yaml_config.ail](../../../examples/runnable/yaml_config.ail) — decode + accessor flow (ran green this session).
- [tests/yaml_bridge_test.ail](../../../tests/yaml_bridge_test.ail) — bridge + `JNumber` semantics (ran green this session).
- [std/yaml.ail](../../../std/yaml.ail) — type/effect check clean (verified this session).
- [internal/pipeline/testdata/builtin_types.golden](../../../internal/pipeline/testdata/builtin_types.golden) — diff must be exactly the one new `_yaml_encode : Json -> Result[string, string]` line.

**What deliberately changes:** nothing. Purely additive.

## Solution Design

### Overview

One new pure Go builtin, `_yaml_encode`, added to the existing `internal/builtins/yaml.go` beside `_yaml_to_json`, plus one export in `std/yaml.ail`. The builtin walks the `Json` ADT `TaggedValue` (the same walk shape `_json_encode`'s `encodeValue` performs) and writes block-style YAML text into a `strings.Builder`, reusing two existing tested pieces: `escapeString` (RFC 8259 escaping, [json_encode.go](../../../internal/builtins/json_encode.go)) and `eval.FormatJSONNumber` (the single source of truth for float text, [internal/eval/json_number.go](../../../internal/eval/json_number.go)). The emitter never constructs a Go map, so `JObject` insertion order survives to the bytes by construction. `std/yaml.encode` is a one-line wrapper, mirroring `std/json.encode`.

The result: `std/yaml` becomes symmetric — `decode` ingests YAML into the shared `Json` ADT, `encode` emits the ADT back as YAML — and both directions run pure and WASM-portable.

### The Emitted Dialect (scalar quoting and block-style policy)

The emitter produces exactly one deterministic byte shape per ADT value — **there is no style negotiation, no pretty-printing pass, no line wrapping.** Scalars:

| ADT | Emitted | Style |
|-----|---------|-------|
| `JNull` | `null` | plain |
| `JBool(true)` / `JBool(false)` | `true` / `false` | plain |
| `JNumber(f)` (finite) | `FormatJSONNumber(f)` — fixed notation in `1e-6 ≤ \|f\| < 1e21`, exponent outside; `-0.0` keeps sign | plain |
| `JString(s)` | `"` + `escapeString(s)` + `"` | **always double-quoted** |
| object key (any) | `"` + `escapeString(k)` + `"` | **always double-quoted** |
| `JArray([])` / `JObject([])` | `[]` / `{}` | flow (only empty collections ever use flow style) |

Structure:

- **Mappings** are block mappings: `"key": value` per line, in `JObject` list order (never sorted, never deduplicated).
- **Sequences** are block sequences: `- item` per line. Under a mapping key the dash is indented +2; **mapping values inside a sequence item** place the first key after `- ` and continuation keys aligned with the first key's column.
- Indent step is 2 spaces. Nested mappings under a key indent +2.
- The document is terminated by exactly one trailing `\n`. No document markers (`---`), no anchors/aliases, no explicit tags, no block literals/folded styles (`|`/`>`), no multi-document output.

Why always-quote (D6): YAML plain scalars are a resolution game (`42`, `true`, `null`, `a: b`, `#x`, `---` all re-resolve to non-strings or re-parse structurally wrong). Delegating that decision to yaml.v3's emitter is exactly the unverifiable-from-here dependency D2 rejects; case-analysis-ing it ourselves is a correctness trap the round-trip tests would have to enumerate forever. Quoting every string (and every key) makes scalar semantics unambiguous *by construction* — and the escape set needs no YAML-specific work: YAML double-quoted scalars accept the JSON escape set (`\" \\ \b \f \n \r \t \uXXXX`), verified against the yaml.v3 v3.0.1 scanner this session (see Verification Log, row 5). Multi-line strings therefore emit as single-line escaped scalars — readable-enough, unambiguous, and always round-trip; block literals are a deliberate Non-Goal.

Cost of this choice: the output is machine-legible rather than human-pretty (`title: "Fysik A"` instead of `title: Fysik A`). That is the A7 trade — one tested path with no per-scalar judgement. Plain-scalar emission could be added later *behind the same signature* only by changing the dialect policy (a second, incompatible byte shape) — hence D6 is frozen here, and dialect changes would need a new decision.

### Round-Trip Contract

**Contract:** For every `j: Json` whose `JNumber` fields are finite and whose `JObject` key lists contain no duplicate keys:

> `decode(encode(j)) == Ok(j)` — **modulo `JObject` key order.**

The two sides of the caveat:

1. **Encode preserves order — always.** The emitter walks the `JObject` pair `ListValue` front-to-back; no Go map and no sort ever exists in the emit path. This is independent of the decode side and is pinned by a byte test (`encode(jo([kv("b", …), kv("a", …)]))` emits `b` first). Whatever the sibling item does, `encode` never needs to change.

2. **Decode re-sorts keys today.** `_yaml_to_json` decodes into `interface{}` (yaml.v3 produces `map[string]interface{}`) and re-emits via `json.Marshal`, which serializes string-keyed maps in sorted key order — the mechanism the triage Primary records and `m-std-yaml` noted. So with today's `decode`, equality holds exactly for single-key objects and objects already in sorted-key order; otherwise the round-trip is equal *as a set of pairs*, not *as an ordered list*.

3. **Sibling interaction.** The triage Primary (yaml-decode-loses-mapping-key-order) recommends making `_yaml_to_json` preserve document order via the yaml.v3 `Node` API. When that lands, `decode` stops re-sorting and `decode(encode(j)) == Ok(j)` holds **exactly** for unique-key objects — the caveat sentence is then deleted from this doc's contract and from `docs/docs/reference/std-yaml.md`. If the sibling instead keeps the map path and documents non-preservation (its option b), the caveat wording above simply becomes the permanent contract. Either way this doc's encode-side claims are unaffected; **this doc does not depend on, duplicate, or implement any decode-side mechanism.**

4. **Duplicate keys.** `encode` is a function of the *list*, not the set: duplicates pass through in list order. But yaml.v3 rejects duplicate mapping keys on `Unmarshal` by default (`uniqueKeys: true`, decode.go:344/767) — so `decode(encode(j))` on a duplicate-key object returns `Err`, loudly, with the message naming the duplicated key. No dedup, no silent last-wins. This is why the guarantee is scoped to unique-key objects.

5. **Non-finite floats.** `encode(JNumber(NaN))` and `±Inf` return `Err` (D4). These are the *only* `Json` values without a round-trip; excluding them at the emit side keeps the contract exception-free. In practice they are unreachable from `decode`d input (the JSON bridge rejects them on the way in).

6. **JSON is YAML — but that is not the feature.** `std/json.encode` output already parses as YAML 1.2; `encode` adds the idiomatic block dialect, uniform quoting, and order preservation. A caller that only needs YAML-parseable bytes can keep using `std/json.encode`; the round-trip contract is why this module's `encode` exists.

### Architecture

**Components:**

1. **`yamlEncodeImpl`** (in [internal/builtins/yaml.go](../../../internal/builtins/yaml.go), ~130 LOC): walks the `TaggedValue` ADT exactly as `_json_encode`'s `encodeValue` does (`JObject` → `ListValue` of `RecordValue` pairs with `key`/`value` fields), writing into a `strings.Builder` with an explicit indent parameter. `JNumber` accepts both `FloatValue` and `IntValue` fields — the same WASM-bridge rationale as `_json_encode`. Pre-checks `math.IsNaN`/`math.IsInf` → `wrapErr(...)` before any formatting (D4).
2. **Reuse, not re-implementation:** `escapeString` (unexported, same `builtins` package, [json_encode.go](../../../internal/builtins/json_encode.go)) for string/key escaping; `eval.FormatJSONNumber` for number text (all three backends' shared source of truth); `wrapOk`/`wrapErr` ([json_decode.go](../../../internal/builtins/json_decode.go)) for the `Result` wrapping.
3. **`makeYAMLEncodeType`**: `Json -> Result[string, string]` via `types.NewBuilder` — `T.Con("Json")` (the `makeJSONEncodeType` pattern) and `T.App("Result", T.String(), T.String())` (the `makeYAMLToJSONType` pattern).
4. **Registration:** `registerYAMLToJSON()` and the new `registerYAMLEncode()` both called from the file's existing `init()`; `BuiltinMetadata` (Since `v0.53.0`, `StabilityStable`, tags `yaml,encoding,data,result`, SeeAlso `std/yaml.decode`, `std/json.encode`). This is the modern `RegisterEffectBuiltin` path only — no legacy `Registry[...]`/eval-table/bytecode-name edits: the VM adapts every spec via `AllSpecs()` ([internal/bytecode/builtin_adapt.go](../../../internal/bytecode/builtin_adapt.go):179), and `_yaml_to_json` already runs end-to-end with only its `yaml.go` registration (Verified, row 10).
5. **`std/yaml.ail`**: `export func encode(j: Json) -> Result[string, string] { _yaml_encode(j) }` — `Json` is already imported. The module doc comment updates from "YAML ingestion" to "YAML ingestion and emission".
6. **Golden:** `internal/pipeline/testdata/builtin_types.golden` gains one line (`UPDATE_GOLDEN=1 go test ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot`, per the file's own header).

### Implementation Plan

**Phase 1: Go builtin** (~4h)
- [ ] `registerYAMLEncode()` + `makeYAMLEncodeType` + `yamlEncodeImpl` + emit helpers in `internal/builtins/yaml.go`, per the Dialect spec above (byte level).
- [ ] `internal/builtins/yaml_test.go`: scalar table (all ADT cases incl. `"true"`, `"42"`, `"null"`, `"a: b"`, `"#x"`, `" leading space"`, embedded newline/quote/backslash, empty string, non-ASCII), numbers window (`42`, `42.5`, `1e21`, `1e-7`, `-0.0`, `0`, int64-large), nesting byte-pins (mapping→sequence→mapping), empty `[]`/`{}`, **order preservation pin**, duplicate-key pass-through, NaN/±Inf → `Err`.
- [ ] **Acceptance through the real decode:** for each fixture, `yamlToJSONImpl(yamlEncodeImpl(j))` must equal `std/json` decoding of `j` (single-key or sorted-key fixtures for exact equality; unsorted fixtures compared order-insensitively today).

**Phase 2: stdlib surface + `.ail` tests** (~2h)
- [ ] `std/yaml.ail`: add `encode` (+ imports unchanged), update module header comment.
- [ ] `tests/yaml_encode_roundtrip.ail` (new; mirror `tests/yaml_bridge_test.ail` style): round-trip `decode(encode(j)) == Ok(j)` via `match` + `Json`'s `Eq` (sorted-key fixtures today, with a TODO referencing the sibling for the exact-order assertion); byte-pin of `encode` output; `Err` path on `JNumber(NaN)`… (NaN needs a literal: use `jnum(0.0/0.0)` if the expression surface allows, else Go-side only — agent resolves).
- [ ] Regenerate `builtin_types.golden`; `ailang builtins list` shows `_yaml_encode  [pure] std/yaml`.

**Phase 3: docs + example** (~2h)
- [ ] `docs/docs/reference/std-yaml.md`: `encode` section + "Semantics and limits" additions (quoting policy, order preservation, `Err` cases, round-trip contract & caveat, single-document).
- [ ] `docs/docs/reference/stdlib.md` line 57: add `encode` to the `std/yaml` entry.
- [ ] `changelogs/v0.32-current.md` `[Unreleased]` feature entry.
- [ ] `examples/runnable/yaml_roundtrip.ail` (new): decode a config → mutate one field via `jo`/`kv` → `encode` → print; must pass the examples verification target.

### Files to Modify/Create

**New files:**
- `tests/yaml_encode_roundtrip.ail` — `.ail` round-trip + byte-pin tests, ~50 LOC.
- `examples/runnable/yaml_roundtrip.ail` — runnable example, ~35 LOC.

**Modified files:**
- `internal/builtins/yaml.go` — +~140 LOC: registration, type, emitter walker + indent helpers.
- `internal/builtins/yaml_test.go` — +~200 LOC: dialect table + acceptance round-trip.
- `std/yaml.ail` — +~10 LOC: `encode` export + header comment.
- `internal/pipeline/testdata/builtin_types.golden` — +1 line, regenerated (`UPDATE_GOLDEN=1`).
- `docs/docs/reference/std-yaml.md` — +~55 lines.
- `docs/docs/reference/stdlib.md` — ~1 line (index entry).
- `changelogs/v0.32-current.md` — feature entry under `[Unreleased]`.

## Examples

### Example 1: The reported gap — emit a config after decoding it

**Before** (today — flow-style JSON is the only std path out):
```ailang
import std/json (encode, jo, js, jint, kv)
let j = jo([kv("title", js("Fysik A")), kv("year", jint(2026))])
println(encode(j))   -- {"title":"Fysik A","year":2026}  (valid YAML 1.2, but not block YAML)
```
*(Run this session — output exact.)* For block style you hand-concatenate: `"title: " ++ title ++ "\nyear: " ++ show(year) ++ "\n"` — per-scalar escaping and indentation done by hand, unchecked until the next `decode`.

**After (planned API):**
```ailang
import std/yaml (encode)
import std/json (jo, js, jint, kv)
import std/result (Ok, Err)

match encode(jo([kv("title", js("Fysik A")), kv("year", jint(2026))])) {
  Ok(y)  => println(y),
  Err(e) => println("encode failed: ${e}")
}
-- title: "Fysik A"
-- year: 2026
```

### Example 2: Round-trip with `decode` (the contract, as a program)

```ailang
import std/yaml (decode, encode)
import std/json (Json, jo, js, kv)
import std/result (Ok, Err)

let src = "title: Fysik A\nyear: 2026\n";
match decode(src) {
  Ok(j) => match encode(j) {
    Ok(y) => match decode(y) {
      Ok(j2) => println(if j2 == j then "round-trip: exact" else "round-trip: keys reordered"),  -- today: reordered (decode side)
      Err(e) => println("re-decode failed: ${e}")
    },
    Err(e) => println("encode failed: ${e}")
  },
  Err(e) => println("decode failed: ${e}")
}
```
*(Structure check-validated this session against a stub with the planned signature — `Json` derives `Eq`; `Result` does not, hence the `match`, not `==`. Once the sibling decode fix lands, this prints `exact` and the caveat paragraph in the reference doc is deleted.)*

## Success Criteria

- [ ] `_yaml_encode` registered, pure, appears in `ailang builtins list` as `[pure] std/yaml`; `builtin_types.golden` regenerated.
- [ ] `std/yaml.encode` type-checks and runs; `encode(jo(...))` output pinned byte-for-byte in Go and `.ail` tests.
- [ ] `encode` preserves `JObject` order in bytes (pinned test — unsorted fixture).
- [ ] `decode(encode(j)) == Ok(j)` for the finite, unique-key fixture corpus — modulo key order today (documented in-test), exact assertion queued on the sibling.
- [ ] NaN/±Inf → `Err`, never `"null"`, never `.nan` (Go test).
- [ ] Duplicate-key `JObject` passes through in order; re-decode errs loudly (Go test, message pinned).
- [ ] Emitted fixtures re-parse through the *existing* `_yaml_to_json` unchanged (Go acceptance test).
- [ ] WASM-portable: `GOOS=js GOARCH=wasm go build ./...` still green (emitter is pure Go; yaml.v3 unchanged).
- [ ] All tests passing; docs (`std-yaml.md`, stdlib index) and changelog updated; runnable example passes `ailang run`.

## Testing Strategy

**Unit tests (Go, `internal/builtins/yaml_test.go`):** the scalar dialect table; number window via `FormatJSONNumber`; nested-structure byte pins; empty collections; order preservation; duplicate keys; NaN/±Inf `Err`; indent correctness for mapping-in-sequence (first key inline, continuation aligned).

**Integration tests (`tests/yaml_encode_roundtrip.ail`):** `decode ∘ encode` equality on sorted-key fixtures; `encode` byte pins through `yamlToJson` re-parse; `Err` path surfaced to AILANG.

**Regression-surface tests:** existing fixtures that must keep working untouched — [examples/runnable/yaml_config.ail](../../../examples/runnable/yaml_config.ail), [tests/yaml_bridge_test.ail](../../../tests/yaml_bridge_test.ail) (ran green this session), `std/json.encode` path, `builtin_types.golden` diff = exactly one added line. No existing output changes (purely additive; see Conflict Surface).

**Manual:** `ailang builtins list | grep yaml`; run the new example; WASM build.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact `Err` message text — *agent may choose* (recommend mirroring decode's prefix style: `yaml: cannot encode non-finite number: …`).
- Emitter internal structure (single walker vs per-constructor helpers, indent representation) — *agent may choose*, keeping the file's per-file size convention.
- Whether `tests/yaml_encode_roundtrip.ail` asserts the NaN `Err` path or leaves it Go-only (depends on whether a NaN literal reaches `JNumber` from the expression surface) — *agent may choose*.
- Example file name and scenario — *agent may choose*.
- A `GoCodegenSpec` for `_yaml_encode` (compiled-Go mode): `_yaml_to_json` has none today (Verified, row 13), so parity says none is required; add one later only if the codegen lane demands it — *agent may defer*.

## Non-Goals

- **`encodeAll` / multi-document emission** — decode is single-document; symmetric single-doc only.
- **Anchors/aliases/tags round-trip fidelity** — the ADT has no anchor structure; none is emitted (same as decode's documented non-preservation).
- **Block literal/folded scalars (`|`, `>`)** for multi-line strings — always double-quoted escapes in this dialect; a later dialect change is additive but changes pinned bytes.
- **Encoding arbitrary AILANG values** (records, `Option`, other ADTs) — rejected by D1; would need its own boundary policy doc.
- **Flow-style or indent-configurable output, "pretty plain scalar" mode** — one fixed dialect by design (A1/A7).
- **Key sorting / canonicalization** — explicitly never; order preservation is the point.
- **Any decode-side change** — owned by the sibling work item (triage Primary).

## Timeline

**Single focused day (~8h):**
- Phase 1 — Go builtin + unit/acceptance tests (~4h)
- Phase 2 — stdlib export + `.ail` tests + golden regen (~2h)
- Phase 3 — docs, changelog, example, WASM verify (~2h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Hand emitter produces YAML the yaml.v3 scanner rejects in some corner (indent/edge scalar) | Med | Always-quote removes the plain-scalar ambiguity space; every fixture is acceptance-tested through the *real* `yamlToJSONImpl` (not just our own decoder); bytes are golden-pinned. |
| Always-quoted output surprises users expecting plain scalars | Med | Documented dialect decision (D6) with rationale; plain-style emission is a future *additive* dialect decision, not a fix. |
| yaml.v3 upgrade changes decode semantics (dup-key handling etc.) | Low | The emitter does not use yaml.v3 at all; only the round-trip *tests* depend on decode — pinned tests catch drift on upgrade. |
| Round-trip caveat wording drifts out of sync with the sibling item | Med | Contract section states both outcomes explicitly; the strict-order assertion in tests carries a TODO naming the sibling; reference doc keeps one caveat sentence. |
| Deep nesting recursion in the walker | Low | Same recursion shape as `_json_encode` (inherited, not new); depth limits are a pre-existing std-level property, out of scope. |

## Related Documents

**Predecessor (this feature is its Future Work):**
- [design_docs/implemented/v0_30_0/m-std-yaml.md](../../implemented/v0_30_0/m-std-yaml.md) — the decode-side bridge; *distinct from this doc*: it built ingestion (`yamlToJson`/`decode`) and explicitly deferred `encode` to Future Work. No overlap in shipped surface.

**Sibling (decode-side key order):**
- [design_docs/planned/ailang-core-triage/yaml-decode-loses-mapping-key-order.md](../ailang-core-triage/yaml-decode-loses-mapping-key-order.md) — source triage (PR #1620). Its **Primary** item is the decode-side fix this doc's round-trip caveat interacts with; at sprint-planning time the two should be **split into separate work items** (per the triage's dispatch note) and sequenced decode-first if convenient, though encode does not block on it.

**Structural templates (Go builtin + thin `.ail` wrapper):**
- [design_docs/implemented/v0_3_23/M-JSON-ENCODE-BUILTIN.md] — `_json_encode`: the `TaggedValue` walk, `escapeString`, and `FormatJSONNumber` this design reuses.
- [design_docs/implemented/v0_52_0/m-json-number-roundtrip.md] — the float-text window (`FormatJSONNumber`) every backend shares; encode inherits it wholesale.
- [design_docs/implemented/v0_19_1/m-stdlib-html.md](../../implemented/v0_19_1/m-stdlib-html.md), [design_docs/implemented/v0_7_3/m-stdlib-xml.md](../../implemented/v0_7_3/m-stdlib-xml.md) — format-module precedent.

## References

- [Design Axioms](/docs/references/axioms) — scoring basis above.
- PR #1620 (ailang-core-triage merge, `dev`); user report `inbox_1791394438696_00b01c33`.
- [internal/builtins/yaml.go](../../../internal/builtins/yaml.go) (`_yaml_to_json`, to be extended), [internal/builtins/json_encode.go](../../../internal/builtins/json_encode.go) (`encodeValue`/`escapeString`), [internal/eval/json_number.go](../../../internal/eval/json_number.go) (`FormatJSONNumber`), [std/yaml.ail](../../../std/yaml.ail), [std/json.ail](../../../std/json.ail).
- `gopkg.in/yaml.v3 v3.0.1` (go.mod:51) — decode-side authority; scanner escape table and resolver consulted (see Verification Log).
- [docs/docs/reference/std-yaml.md](../../../docs/docs/reference/std-yaml.md) — current documented decode semantics ("no silent coercion", single-document).

## Verification Log

*Every load-bearing claim above was checked against the code or the pinned dependency this session (2026-10-07). "Confirmed" rows carry their evidence here; rows that sprint review re-verifies on implementation are the pinned-byte tests.*

| # | Claim | Evidence | Status |
|---|-------|----------|--------|
| 1 | `std/yaml` exposes only `yamlToJson`/`decode` (no encode) | Read [std/yaml.ail](../../../std/yaml.ail) — both exports listed | Confirmed |
| 2 | `_yaml_to_json` is std/yaml's only registered builtin | `ailang builtins list \| grep yaml` → one row; repo grep → no other registration | Confirmed |
| 3 | Decode re-sorts mapping keys today | [internal/builtins/yaml.go](../../../internal/builtins/yaml.go) `yamlToJSONImpl`: `yaml.Unmarshal`→`interface{}` (map) → `json.Marshal` (documented Go map-key sorting); mechanism recorded in triage Primary | Confirmed |
| 4 | Current workaround output is JSON flow style | Ran `encode(jo([kv("title", js("Fysik A")), kv("year", jint(2026))]))` this session → `{"title":"Fysik A","year":2026}` | Confirmed |
| 5 | YAML double-quoted scalars accept the JSON escape set (`\" \\ \b \f \n \r \t \uXXXX`) | yaml.v3 v3.0.1 `scannerc.go` escape table (module zip, pinned go.mod:51) — all JSON escapes present | Confirmed |
| 6 | `FormatJSONNumber` returns `"null"` for NaN/±Inf (silent-fallback hazard), `-0.0` for negative zero, fixed in `1e-6 ≤ \|f\| < 1e21`, exponent outside | [internal/eval/json_number.go](../../../internal/eval/json_number.go):23–51 | Confirmed |
| 7 | yaml.v3 resolves every `FormatJSONNumber` output to a number (round-trip re-parse) | yaml.v3 `resolve.go` `yamlStyleFloat = ^[-+]?(\.[0-9]+ \| [0-9]+(\.[0-9]*)?)([eE][-+]?[0-9]+)?$` + int resolution; checked against window outputs (`42`, `42.5`, `-0.0`, `1e+21`, `1e-7`) | Confirmed |
| 8 | yaml.v3 rejects duplicate mapping keys on `Unmarshal` (round-trip scope = unique keys) | yaml.v3 `decode.go`:344 `uniqueKeys: true`; :767–786 duplicate-key terrors | Confirmed |
| 9 | Builtin name `_yaml_encode` unallocated; `encode` export name free in std/yaml | Repo-wide grep (no hits); std/yaml.ail exports list | Confirmed |
| 10 | One `RegisterEffectBuiltin` site suffices (eval + VM); no legacy-table edits needed | [internal/builtins/spec.go](../../../internal/builtins/spec.go):68; [internal/bytecode/builtin_adapt.go](../../../internal/bytecode/builtin_adapt.go):179 (`AllSpecs()` loop); `_yaml_to_json` lives only in yaml.go and `tests/yaml_bridge_test.ail` ran green end-to-end this session | Confirmed |
| 11 | Golden regeneration path exists | `internal/pipeline/testdata/builtin_types.golden` header + `builtin_golden_types_test.go` `UPDATE_GOLDEN=1` flow | Confirmed |
| 12 | Doc's AILANG snippets type-check (import/match/`Result` shapes) | `ailang check` on this-session stubs: before-variant ran (output row 4); after-variant with local `encode(j: Json) -> Result[string,string]` stub — no errors | Confirmed (planned-API call sites stubbed) |
| 13 | std/yaml has no `GoCodegenSpec` today (codegen-parity baseline) | `grep yaml internal/builtins/registry_codegen*.go` → empty | Confirmed |
| 14 | `Result` has no `Eq` derivation → round-trip equality compared via `match`, `Json` does derive `Eq` | [std/result.ail](../../../std/result.ail):11; [std/json.ail](../../../std/json.ail) `Json … deriving (Eq)` | Confirmed |
| 15 | No planned/implemented doc already covers YAML encoding (duplicate gate) | Related-doc search: top matches are unrelated (`m-cloud-health` keyword noise); the only YAML feature doc is the v0_30_0 predecessor, which defers encode to Future Work — distinct, cited | Confirmed (neural search unavailable this session — Ollama offline; SimHash + manual grep) |
| 16 | `decode` contract facts relied on: single-document, no silent coercion (non-string keys/NaN/Inf → `Err`), empty input → `JNull` | `yaml.go` `BuiltinMetadata`/impl; `yaml_test.go` cases; [docs/docs/reference/std-yaml.md](../../../docs/docs/reference/std-yaml.md) "Semantics and limits" | Confirmed |
| 17 | Sibling decode-side doc does not exist yet (reference phrasing) | `design_docs/planned/` grep: no yaml key-order doc; only the triage Primary describes it | Confirmed (link points at triage until the sibling doc lands) |
| 18 | D2 mechanism — yaml.v3 **sorts** map keys on encode (so a reconstructed-map emitter would destroy `JObject` order *deterministically*) | yaml.v3 v3.0.1 `encode.go`:188–189 (`keyList` + `sort.Sort`), `sorter.go` | Confirmed |
| 19 | D2 mechanism — yaml.v3 `Node` marshal **preserves** `Content` insertion order (the viable-but-unpinnable alternative) | yaml.v3 v3.0.1 `encode.go`:428 (`nodev`), :515 (`node.Content[i]` walk) | Confirmed |

## Future Work

- Sibling: preserve mapping key order in `_yaml_to_json` (triage Primary) → deletes this doc's round-trip caveat.
- `decodeAll`/`encodeAll` for multi-document streams (decode-side first).
- Plain-scalar (unquoted) emission mode as a *separate dialect decision*, if human-facing prettiness is ever demanded — additive behind a new policy, never a change to this dialect's pinned bytes.
- Block literal (`|`) emission for multi-line strings, same constraints.
- `GoCodegenSpec` for `_yaml_encode` if the compiled-Go lane starts requiring std/yaml coverage.

---

**Document created**: 2026-10-07
**Last updated**: 2026-10-07
### Execution correction: YAML-sensitive Unicode

Real decode-bridge acceptance tests found that reusing JSON escaping alone rejects
literal U+007F and normalizes literal U+0085 to a space. The emitter therefore
reuses `escapeString` for ordinary text and emits U+007F–U+009F and U+2028/U+2029
as JSON-compatible `\uXXXX` escapes. This retains the approved double-quoted
dialect and round-trip contract without changing JSON escaping or YAML decode.

Real boundary tests also found that literal U+FFFE/U+FFFF are rejected, so those
use Unicode escapes too. Quoted keys reaching 1024 characters exceed yaml.v3's
simple-key scanner limit. Oversized keys therefore emit deterministic explicit
block-key syntax (`? "key"` followed by `: value`) at the same indentation.
Ordinary key byte fixtures remain unchanged; this exception preserves the input
and round-trip contract for long keys. Evaluator review should check this
necessary correction to the design's one-line mapping illustration.
