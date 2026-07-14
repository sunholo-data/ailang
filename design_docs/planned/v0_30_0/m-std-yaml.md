# M-STD-YAML: `std/yaml` — YAML Ingestion via JSON Bridge

**Status**: Planned
**Target**: v0.30.0
**Priority**: P1 (unblocks an external consumer; no hard deadline)
**Estimated**: ~1 day (0.5–1.5 days)
**Dependencies**: `std/json` (existing — reuses `Json` ADT and `decode`); `gopkg.in/yaml.v3 v3.0.1` (already vendored)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure function of the input string; yaml.v3 decode + json.Marshal are deterministic (map keys sorted by encoding/json). |
| A2: Replayability | 0 | Pure builtin, no trace impact. |
| A3: Effect Legibility | +1 | Zero effects — `yamlToJson`/`decode` are pure, matching `std/json`. No hidden IO. |
| A4: Explicit Authority | +1 | No ambient authority; reads no files, no env. Caller supplies the string. |
| A5: Bounded Verification | +1 | Return type `Result[string,string]` / `Result[Json,string]` makes failure local and checkable. |
| A6: Safe Concurrency | 0 | No concurrency surface. |
| A7: Machines First | +1 | Removes the shell-out-to-Python workaround; keeps the pipeline single-language and analyzable. |
| A8: Minimal Syntax | +1 | No new syntax. One new module, one new builtin. |
| A9: Cost Visibility | 0 | O(n) in input size; no surprising cost. |
| A10: Composability | +1 | `decode` is literally `yamlToJson` ∘ `std/json.decode` — composes with the entire `std/json` accessor surface (`get`, `getString`, …). |
| A11: Structured Failure | +1 | All failure paths return typed `Err(string)`; no panics, no silent coercion (see edge cases). |
| A12: System Boundary | +1 | YAML→JSON is an explicit boundary crossing at a named function, not an implicit conversion. |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — pure string→string transform.
- [x] A3 (Effects): No hidden side effects — module is effect-free.
- [x] A4 (Authority): No ambient access granted.
- [x] A7 (Machines First): Optimizes for a single-language, WASM-portable, machine-analyzable pipeline.

## Problem Statement

Building a fully-AILANG data pipeline is blocked on a single missing capability: **reading a YAML source file.** AILANG ships 42 stdlib modules at v0.29.2 with no `std/yaml`; `docs_search("yaml")` returns nothing.

**Current State:**
- Consumer `cph-uni-stx-bench` (STX Fysik-A exam benchmark harness, msg `msg_20260714_181417_5dde3021`) has everything else covered: answer keys are JSON (`std/json`), problem texts are `.txt` (`std/fs`), outputs are JSONL. Only the human-authored corpus catalogue (~79 KB) is YAML.
- The documented workaround is shelling out to Python for a one-line YAML→JSON, which (a) breaks the zero-external-dependency goal and (b) **is not WASM-portable** — the grader core is destined for in-browser answer-checking, and a Python pre-step can't ship to the browser.

**Impact:**
- Blocks Stage 0 (catalogue ingestion) of the STX harness.
- Generalizes well beyond STX: bench-config, tool manifests, and corpus metadata are commonly YAML. YAML config ingestion is a recurring agent need.

## Goals

**Primary Goal:** Provide a pure, WASM-portable `std/yaml` module that turns a YAML string into AILANG's existing `Json` ADT, with zero new external dependencies.

**Success Metrics:**
- `std/yaml.yamlToJson(s) -> Result[string, string]` and `std/yaml.decode(s) -> Result[Json, string]` both ship and pass tests.
- The requester's 79 KB catalogue round-trips: `yamlToJson` output parses under `std/json.decode` to the expected structure.
- Module + underlying builtin compile and run under `GOOS=js GOARCH=wasm` (verified: yaml.v3 already builds for js/wasm).
- `docs_search("yaml")` returns the module; `ailang docs std/yaml` lists exports.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **One builtin, not two.** Implement a single `_yaml_to_json(s) -> Result[string,string]` in Go; derive `decode` in pure AILANG as `yamlToJson ∘ json.decode`. | Minimizes the frozen Go/core surface (Systemic-fix principle). Both requested APIs fall out of one builtin. | agent | design | low |
| **Reuse `std/json`'s `Json` ADT** (`import std/json (Json, decode)`) rather than a new YAML ADT. | Requester explicitly wants "the same Json ADT"; gives the entire `std/json` accessor surface for free. Proven: `std/jwt`/`std/sem` already import `Json` cross-module. | agent | design | low |
| **Single-document only in v1.** yaml.v3 `Unmarshal` reads the first document; multi-doc streams (`---` separated) are out of scope. | Simplicity; requester's catalogue is single-doc. Widening later is additive (new `decodeAll`), not a breaking change. | agent | design | low |
| **No silent coercion on unconvertible YAML.** Any YAML that `json.Marshal` rejects (non-string map keys, NaN/±Inf floats) returns `Err(msg)`, never a guessed value. | Data-integrity per No-Silent-Fallbacks. A wrong catalogue value is worse than a loud failure. | agent | design | low |

### Design Freeze

Before implementation begins:

- [x] Builtin surface = exactly one Go builtin `_yaml_to_json`. (change cost: low — but freezing avoids scope creep into a native YAML→ADT walker)

*(No "high" change-cost decisions — this is an isolated additive module.)*

## Conflict Surface

**None.** This adds (a) one new pure builtin in a new file `internal/builtins/yaml.go`, registered via `init()` into the builtin registry, and (b) one new stdlib module `std/yaml.ail`. It introduces **no new syntax**, extends **no existing syntactic or semantic position**, and changes **no parser/typechecker/codegen disambiguation**. Existing programs are unaffected; there is no construct that now parses differently. The only shared namespace touched is the builtin name registry — verified free below.

## Solution Design

### Overview

A thin bridge. The hard part of YAML (significant whitespace, block scalars, the tag system) is delegated to `gopkg.in/yaml.v3`, exactly as `std/json`/`std/xml`/`std/html` delegate their parsers to Go "for correctness." YAML 1.2's core schema maps onto JSON types, so the natural target is a JSON string, which `std/json.decode` already turns into the `Json` ADT.

### Architecture

**Components:**
1. **Go builtin `_yaml_to_json`** (`internal/builtins/yaml.go`): `yaml.Unmarshal([]byte(s), &v interface{})` → `json.Marshal(v)` → `Ok(string)`. Any error from either step → `Err(message)`. Pure, `IsPure: true`, same `RegisterEffectBuiltin` pattern as `_json_decode` ([internal/builtins/json_decode.go](internal/builtins/json_decode.go)).
2. **Module `std/yaml.ail`**: `import std/json (Json, decode)`. Exposes:
   - `yamlToJson(s: string) -> Result[string, string]` — direct wrapper over `_yaml_to_json`.
   - `decode(s: string) -> Result[Json, string]` — `match yamlToJson(s) { Ok(j) => json.decode(j), Err(e) => Err(e) }`. Zero extra Go.
3. **Docs + examples**: reference page, stdlib index entry, `examples/yaml_*.ail`.

**Why the bridge is correct here (verified):**
```
in:  "name: STX\nitems:\n  - a\n  - b\ncount: 3\nnested:\n  x: 1.5\n  ok: true\n"
out: {"count":3,"items":["a","b"],"name":"STX","nested":{"ok":true,"x":1.5}}
```
yaml.v3 decodes string-keyed mappings into `map[string]interface{}`, which `json.Marshal` serializes directly (unlike yaml.v2's `map[interface{}]interface{}`).

**Edge cases (must be tested):**
- Empty input → `Unmarshal` yields `nil` → `json.Marshal(nil)` = `"null"` → `Ok("null")` (decodes to `JNull`).
- Non-string map keys / NaN / ±Inf → `json.Marshal` errors → `Err(...)`. No coercion.
- Large integers → become `JNumber(float)` via `std/json` — inherited precision limit of the existing `Json` ADT, documented, not new.
- Multi-document stream → only the first document is read (documented Non-Goal).

### Implementation Plan

**Phase 1: Go builtin** (~3h)
- [ ] `internal/builtins/yaml.go`: `registerYAMLToJSON()` + `yamlToJSONImpl` + `makeYAMLToJSONType` (`string -> Result[string,string]`), following `json_decode.go`.
- [ ] Register in `init()`; add `BuiltinMetadata` (Since `v0.30.0`, `StabilityStable`, tags `yaml,parsing,data,result`).
- [ ] `internal/builtins/yaml_test.go`: happy path, empty, nested, sequences, scalars (int/float/bool/null), non-string-key → Err, malformed indentation → Err.

**Phase 2: AILANG module** (~2h)
- [ ] `std/yaml.ail`: `module std/yaml`, `import std/json (Json, decode)`, exports `yamlToJson`, `decode`. Doc comments matching `std/json.ail`.
- [ ] Register `std/yaml` in the stdlib resolver if new modules aren't auto-discovered ([internal/loader/stdlib_resolver.go](internal/loader/stdlib_resolver.go)) and in `std/embed.go`.
- [ ] `.ail` integration test (`tests/yaml_*_test.ail`) mirroring `tests/json_*_test.ail`.

**Phase 3: Docs, examples, WASM verify** (~2h)
- [ ] `examples/yaml_config.ail` — read a YAML string, `decode`, pull fields via `std/json` accessors, print. Must pass `make verify-examples`.
- [ ] Reference doc `docs/docs/reference/stdlib/yaml.md` (mirror the html/json reference); add `std/yaml` to the Stdlib Index ([docs/docs/reference/stdlib.md](docs/docs/reference/stdlib.md)).
- [ ] CHANGELOG entry.
- [ ] `GOOS=js GOARCH=wasm go build ./...` sanity for the WASM target.

### Files to Modify/Create

**New files:**
- `internal/builtins/yaml.go` — the `_yaml_to_json` builtin, ~120 LOC.
- `internal/builtins/yaml_test.go` — Go unit tests, ~150 LOC.
- `std/yaml.ail` — module wrapper, ~40 LOC.
- `tests/yaml_bridge_test.ail` — integration test, ~30 LOC.
- `examples/yaml_config.ail` — runnable example, ~25 LOC.
- `docs/docs/reference/stdlib/yaml.md` — reference doc, ~80 lines.

**Modified files:**
- `std/embed.go` / `internal/loader/stdlib_resolver.go` — register `std/yaml` if not auto-discovered, ~2 LOC.
- `docs/docs/reference/stdlib.md` — index entry, ~1 line.
- `CHANGELOG.md` — feature entry.

## Examples

### Example 1: The requester's use case (catalogue ingestion)

**Before** (blocked — required a Python shell-out, not WASM-portable):
```
# outside AILANG:
python -c "import yaml,json,sys; json.dump(yaml.safe_load(open('catalogue.yaml')), sys.stdout)"
```

**After** (pure AILANG, WASM-portable):
```ailang
module stx/ingest

import std/fs (readFile)
import std/yaml (decode)
import std/json (getString)
import std/result (Result, Ok, Err)

export func main() -> () ! {IO, FS} {
  let raw = readFile("catalogue.yaml");
  match decode(raw) {
    Ok(cat)  => println("loaded: " ++ show(getString(cat, "title"))),
    Err(msg) => println("catalogue parse failed: " ++ msg)
  }
}
```

### Example 2: Minimal bridge (string in, JSON string out)

```ailang
import std/yaml (yamlToJson)
import std/result (Ok, Err)

match yamlToJson("a: 1\nb: [x, y]\n") {
  Ok(j)  => println(j),   -- {"a":1,"b":["x","y"]}
  Err(e) => println("err: " ++ e)
}
```

## Success Criteria

- [ ] `_yaml_to_json` builtin registered, pure, appears in `ailang builtins list`.
- [ ] `std/yaml.yamlToJson` and `std/yaml.decode` type-check and run; `decode` returns the same `Json` ADT as `std/json`.
- [ ] Requester's 79 KB catalogue: `decode` succeeds and yields the expected top-level structure (acceptance fixture from the consumer, or a representative subset).
- [ ] Non-string-key / NaN inputs return `Err`, never a coerced value (unit test).
- [ ] `GOOS=js GOARCH=wasm go build ./...` succeeds.
- [ ] All tests passing; `make verify-examples` green with `examples/yaml_config.ail`.
- [ ] Docs updated: reference page + stdlib index; `ailang docs std/yaml` works.

## Testing Strategy

**Unit tests (Go):** happy path, nested maps, sequences, all scalar types, empty string→`"null"`, non-string map key→Err, NaN/Inf→Err, malformed indentation→Err, flow style (`{a: 1}`) equals block style.

**Integration tests (.ail):** `decode` round-trip through `std/json` accessors; `yamlToJson` output fed to `std/json.decode` yields identical `Json` to decoding the equivalent JSON directly.

**Manual:** run `examples/yaml_config.ail` via `ailang run`; build WASM target and confirm no missing-symbol/`syscall/js` errors.

## Deferred Decisions

- Exact `Err` message format for parse failures — *agent may choose* (recommend echoing yaml.v3's error, prefixed `yaml: `).
- Whether to also expose `encode(j: Json) -> Result[string,string]` (Json→YAML) now or later — *agent may defer*; not requested, adds a second builtin. Recommend deferring to Future Work.

## Non-Goals

- **Native YAML→`Json` walker** (no string round-trip) — the bridge covers 100% of the stated need; a native walker is a nice-to-have, not worth the extra Go surface now.
- **Multi-document streams** (`---` separated) — v1 reads the first document only. Additive `decodeAll` later if needed.
- **YAML anchors/aliases/custom tags round-trip fidelity** — yaml.v3 resolves anchors/aliases during decode (they work), but we do not preserve or re-emit them; output is plain JSON.
- **YAML emission (`Json`→YAML)** — deferred to Future Work.

## Timeline

Single focused day:
- Phase 1 (builtin + Go tests): ~3h
- Phase 2 (module + .ail tests): ~2h
- Phase 3 (docs, example, WASM verify): ~2h

**Total: ~7 hours.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `json.Marshal` fails on some real catalogue construct (non-string keys) | Med | Return `Err` loudly; test against the actual 79 KB fixture before closing. Document the constraint. |
| Int precision loss (int64 → float `JNumber`) | Low | Inherited from existing `std/json`; document. Not a regression. |
| New module not auto-discovered by resolver | Low | Mirror how `std/html` (v0.19.1) was registered; verify `ailang docs std/yaml` in Phase 2. |
| yaml.v3 behavior differs under js/wasm | Low | Already verified `GOOS=js GOARCH=wasm go build` of the yaml.v3→json path succeeds (exit 0). |

## Related Documents

**Implemented (structural template — same Go-builtin + `.ail`-wrapper shape):**
- [design_docs/implemented/v0_19_1/m-stdlib-html.md](design_docs/implemented/v0_19_1/m-stdlib-html.md) — closest analog: a pure format parser (HTML5) as Go builtin + thin `std/html.ail`.
- [design_docs/implemented/v0_7_3/m-stdlib-xml.md](design_docs/implemented/v0_7_3/m-stdlib-xml.md) — `std/xml`, same pattern.
- [design_docs/implemented/v0_7_4/m-stdlib-gaps.md](design_docs/implemented/v0_7_4/m-stdlib-gaps.md) — stdlib gap-filling precedent.

**Distinction from the above:** those implement *native* parsers producing their own ADTs. `std/yaml` deliberately does *not* — it bridges to JSON and reuses the `Json` ADT, so it's strictly smaller (one builtin, no ADT design).

## References

- [Design Axioms](/docs/references/axioms)
- Request: agent message `msg_20260714_181417_5dde3021` from `cph-uni-stx-bench`.
- `gopkg.in/yaml.v3 v3.0.1` — already in `go.mod`; verified js/wasm-buildable.
- Pattern source: [internal/builtins/json_decode.go](internal/builtins/json_decode.go), [std/json.ail](std/json.ail).

## Future Work

- `std/yaml.decodeAll(s) -> Result[List[Json], string]` for multi-document streams.
- `std/yaml.encode(j: Json) -> Result[string, string]` for Json→YAML emission (second builtin `_yaml_from_json`).
- Native YAML→`Json` walker if the string round-trip ever shows measurable cost on large inputs.

---

**Document created**: 2026-07-14
**Last updated**: 2026-07-14
