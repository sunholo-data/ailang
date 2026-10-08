# M-STD-YAML-MAPPING-KEY-ORDER: `std/yaml` preserves YAML mapping key order

**Status**: Implementation complete; awaiting coordinator evaluation
**Target**: v0.53.0
**Priority**: P1 (silent data loss in a shipped stdlib bridge; no correctness/security emergency)
**Estimated**: ~1 day, ~150–200 LOC including tests (walker ~110 Go LOC in `internal/builtins/yaml.go`, tests ~+50, docs/comments small)
**Dependencies**: none — `gopkg.in/yaml.v3 v3.0.1` already in `go.mod`; no new stdlib surface, no type changes.
**Triage**: `design_docs/planned/ailang-core-triage/yaml-decode-loses-mapping-key-order.md`, section "Primary" (merged on dev, PR #1620).
**Reported**: user session via `ailang-core`, inbox message `inbox_1791394438696_00b01c33`.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Document order is exactly as deterministic as sorted order was. The old "sorted by `encoding/json`" note in `m-std-yaml.md` was an *incidental* A1 justification, never a documented contract; the new output is a pure function of the input string either way. |
| A2: Replayability | 0 | Pure builtin, no trace impact. |
| A3: Effect Legibility | +1 | Unchanged — `yamlToJson`/`decode` stay pure (`IsPure: true`), zero effects. |
| A4: Explicit Authority | +1 | Unchanged — no ambient authority; caller supplies the string. |
| A5: Bounded Verification | +1 | Return type `Result[string,string]` unchanged; every failure stays a typed `Err`. |
| A6: Safe Concurrency | 0 | No concurrency surface. |
| A7: Machines First | +1 | Removes a silent loss that breaks fixture comparison and key-order-sensitive consumers; output becomes the *natural* machine representation (source order) instead of an arbitrary re-sort. |
| A8: Minimal Syntax | +1 | No syntax, no new builtin, no new module. One implementation swap behind an unchanged signature. |
| A9: Cost Visibility | 0 | Same O(n) walk; same yaml.v3 parse. |
| A10: Composability | +1 | `yamlToJson` output becomes order-consistent with `std/json.decode`, so YAML and JSON sources feeding the same `Json`-ADT pipeline behave identically. |
| A11: Structured Failure | +1 | All existing `Err` paths (non-string keys, NaN/Inf, malformed YAML, duplicate keys, bad merge value) are preserved as typed `Err`, no panics, no coercion. |
| A12: System Boundary | +1 | Unchanged — YAML→JSON boundary stays at one named, pure function. |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism introduced — Go map iteration is *removed* from the ordering story; the only order used is the parsed document's.
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): the fix removes a silent data-integrity loss; it does not trade machine analyzability for human convenience.

## Problem Statement

`std/yaml` destroys YAML mapping key order, silently, at the Go bridge — after the parse and before the `Json` ADT.

`yamlToJSONImpl` (`internal/builtins/yaml.go`) does `yaml.Unmarshal(..., &v interface{})`, so yaml.v3 materializes every mapping as `map[string]interface{}`, then `json.Marshal(v)`, which serializes Go maps in **sorted** key order. Reproduced live on dev (v0.52.5 binary, `ailang run`, temp module):

```ailang
-- tmp probe, run 2026-10-07
match yamlToJson("b: 1\na: 2\nc: 3\n") { ... }   -- => {"a":2,"b":1,"c":3}
match decode("b: 1\na: 2\nc: 3\n") { ... }       -- => keys: [a, b, c]
```

The same data entering as JSON keeps its order, because `std/json` is order-preserving end to end:

```ailang
match jsonDecode("{\"b\":1,\"a\":2,\"c\":3}") { ... }   -- => keys: [b, a, c]
```

Mechanism, verified by reading the dispatch (`std/json.ail:18` `decode = _json_decode`): `_json_decode` (`internal/builtins/json_decode.go`) is a token-streaming builder over `json.Decoder.Token()` that appends `{key, value}` pairs **in token order** (`popObject`, line ~381). So the `Json` ADT itself is fine: `JObject` is a list of pairs, `keys(j)` returns source order. The loss is entirely in the YAML bridge: parse → `map[string]interface{}` → sorted `json.Marshal`.

**Current State:**
- The contract is **undocumented either way**: `docs/docs/reference/std-yaml.md` and `design_docs/implemented/v0_30_0/m-std-yaml.md` never rule on key order. The only trace of the behavior is a test comment ("encoding/json sorts object keys") and one incidental phrase in the old doc's A1 row.
- No other Go code calls the builtin (grep: `_yaml_to_json`/`yamlToJSONImpl` appear only in `yaml.go`/`yaml_test.go`), so the blast radius is `std/yaml.ail` + tests + docs.

**Impact:**
- Any consumer that compares `yamlToJson` output against a fixture, or that relies on document order (display order, manifest field ordering, precedence lists that humans read), gets silently wrong data — no `Err`, no warning. That is a silent fallback affecting data integrity, which the program policy says to eliminate rather than document around.
- Inconsistency: the same mapping decoded from JSON keeps order; from YAML it doesn't. Same ADT, same pipeline, two contracts.

## Goals

**Primary Goal:** `_yaml_to_json` emits mapping keys in YAML document order, preserving every existing guarantee, with the contract written down in the reference doc.

**Success Metrics:**
1. `yamlToJson("b: 1\na: 2\nc: 3\n")` → `Ok("{\"b\":1,\"a\":2,\"c\":3}")` (pinned test).
2. Every behavior in the guarantees table below still holds, each pinned by a test (many already exist in `yaml_test.go`).
3. `std/yaml.decode` and `std/json.decode` agree on key order for equivalent documents (pinned test).
4. `docs/docs/reference/std-yaml.md` states the order contract in "Semantics and limits".
5. All existing tests pass except the two that intentionally re-pin (below).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1: (a) preserve document order via the yaml.v3 `Node` API, vs (b) keep the map path and document "order not preserved".** | (a) changes byte-for-byte output of every mapping whose document order ≠ sorted order; (b) keeps bytes but ships a permanent order-lossy contract inconsistent with `std/json`. | **human** (ratify this doc; recommendation: **(a)**) | design | med (single file + tests + docs) |
| D2: Scalar typing stays delegated to yaml.v3 (`node.Decode(&interface{})` per scalar) and scalar JSON emission stays delegated to `json.Marshal` per scalar. | Guarantees tags (`!!str`, `!!int`), timestamps (`!!timestamp` → `time.Time` → RFC3339), number resolution (hex/octal/int64), string escaping, and the NaN/Inf `json.Marshal` error are preserved *verbatim* instead of re-implemented and drifted. | agent | design | low |
| D3: Merge keys (`<<`) are expanded by the walker (not by a wholesale node decode), with explicit-override and earlier-merge-wins precedence, at the merge key's position. | Today's expansion happens inside `yaml.Unmarshal` into a map; the Node walk must reproduce it or `<<` silently stops working. Precedence rules are observable behavior (probed, see Verification Log). | agent | design | low |
| D4: Duplicate mapping keys and non-string keys stay loud `Err`s in the walker (explicit checks), keeping today's failure set. | Node decoding performs no duplicate detection; without a check, `a: 1\na: 2` would silently drop data — a new silent fallback. | agent | design | low |
| D5: Exact `Err` message text for the re-implemented checks may differ from today's (`json: unsupported type: map[interface {}]interface{}` → a clear `yaml: non-string mapping key …`). | Both old and new are `Err(...)`; no test pins the old text beyond the `yaml:` prefix (verified). | agent | runtime | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — decided in this doc: **(a)**. Human ratifies at doc review (this is the one decision that changes a public byte surface; everything else is agent latitude).
- [x] D2, D3, D4 — decided as stated above.
- [ ] If the human overrides D1 to **(b)**: no code change ships; only the docs phase runs, stating "mapping keys are emitted in sorted order, document order is not preserved" in `std-yaml.md` and the builtin doc comment, plus test-comment updates. That outcome should be recorded here before closing.

## Conflict Surface

This swaps the body of one pure Go builtin behind an unchanged signature. No new syntax, no new type, no new stdlib export, no new builtin name.

1. **Positions extended:** none. `internal/builtins/yaml.go`'s `_yaml_to_json` keeps `string -> Result[string, string]`, `IsPure: true`, same registration.
2. **Other constructs already living in those positions:** the builtin registry (`RegisterEffectBuiltin`) and `BuiltinMetadata` doc-comment examples; nothing else consumes the function (grep: no callers outside `yaml.go`/`yaml_test.go`).
3. **Parser/typechecker disambiguation:** none — post-parse builtin behavior only; no parser, elaboration, or codegen surface is touched.
4. **Existing programs that MUST still work (fixtures):**
   - `std/yaml.ail` (module unchanged: `yamlToJson`, `decode` — verify with `ailang check std/yaml.ail`).
   - `tests/yaml_bridge_test.ail` — Test 1 re-pins (intentional), Tests 2–7 unchanged assertions must still pass.
   - `examples/runnable/yaml_config.ail` — field access by name only (`getString`/`getInt`), unaffected; must still run via `ailang run --caps IO --entry main`.
   - Any user program calling `yamlToJson`/`decode` — same types, same `Err` conditions.
5. **What deliberately changes (intentional incompatibilities):**
   - Byte-for-byte JSON output for every mapping with ≥2 keys whose document order differs from sorted order (single-key mappings, sequences, and all scalars are unchanged).
   - `Err` message *text* on the non-string-key and duplicate-key paths (D5); both remain `Err` with the `yaml:` prefix.
   - Nothing else: anchors/aliases, tags, merge keys, first-document-only, empty→`null`, NaN/Inf → `Err` — all preserved (guarantees table).

## Solution Design

### Overview

Replace the `interface{}` round-trip with a direct walk of yaml.v3's **Node tree**, emitting JSON into a `bytes.Buffer`. Structure (order, nesting, mapping pairs) is walked by us; **scalar semantics are delegated** to yaml.v3 (`node.Decode`) and **JSON scalar bytes** to `json.Marshal` (D2). This is the same shape as `_json_decode`'s streaming builder: a small recursive walker, no frames needed because yaml.v3 already gives us the tree.

### Architecture

```go
// sketch (internal/builtins/yaml.go)
func yamlToJSONImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    // arg check unchanged
    var root yaml.Node
    if err := yaml.Unmarshal([]byte(sv.Value), &root); err != nil {   // parse errors -> Err (unchanged)
        return wrapErr("yaml: " + err.Error()), nil
    }
    var buf bytes.Buffer
    if err := emitNode(&root, &buf); err != nil {                     // includes Marshal errors -> Err (unchanged)
        return wrapErr(err.Error()), nil
    }
    return wrapOk(&eval.StringValue{Value: buf.String()}), nil
}
```

**`emitNode(n *yaml.Node, buf *bytes.Buffer) error`:**

- **DocumentNode** → recurse into `Content[0]`. Empty input / whitespace-only input yields a node with no content → emit `null` (matches today's `Ok("null")`).
- **AliasNode** → recurse on `n.Alias` (the anchored node; yaml.v3 resolves the pointer at parse time). This keeps anchor/alias resolution for scalars, mappings, and sequences exactly as today (`a: &x [1,2]\nb: *x` → `{"a":[1,2],"b":[1,2]}`).
- **ScalarNode** → `var v interface{}; n.Decode(&v)` then `json.Marshal(v)` into the buffer.
  - Keeps: explicit tags (`!!str 1` → `"1"`, `!!int "5"` → `5`), timestamps (`2026-10-07` → `time.Time` → `"2026-10-07T00:00:00Z"`), hex/octal → ints, block scalars → escaped strings, null → `null`.
  - Keeps the NaN/±Inf failure: `json.Marshal(math.NaN())` errors → `Err("yaml: cannot represent as JSON: json: unsupported value: NaN")`, byte-identical message to today.
- **SequenceNode** → `[`, children in order, `]`; empty → `[]` (matches `{s: []}` → `{"s":[]}`).
- **MappingNode** → `{` then per pair `(k, v)` from `Content[0], Content[1], Content[2], Content[3], …`:
  1. **Merge key** (`k.Tag == "!!merge"`, i.e. `<<`): expand (below), not emitted as a literal key.
  2. **Key check** (D4): resolve alias if needed; `k.Decode(&v)` must yield a `string` — otherwise `Err("yaml: cannot represent as JSON: non-string mapping key …")`. Keys that are mappings/sequences/nulls/ints/bools all fail here, matching today's `Err` set (int `1:`, bool `true:`, null `~:` keys today all `Err`; quoted `"1":` stays a string and passes).
  3. **Duplicate check** (D4): seen-set per mapping; second occurrence → `Err` (today yaml.v3 errors with `mapping key "a" already defined at line N`; keep the `yaml:` prefix, message may differ per D5).
  4. Emit `"key":` — the *key string* itself is JSON-escaped via `json.Marshal(string)` so `"quoted:weird"` keys round-trip — then recurse on `v`. Empty mapping → `{}` (matches `{m: {}}` → `{"m":{}}`).

**Merge expansion (D3), matching today's probed semantics:**
- Value must resolve to a mapping (via alias or inline) or a sequence of mappings; anything else → `Err("yaml: map merge requires map or sequence of maps as the value")` (today's message, keep verbatim).
- Insert the merged mapping's pairs **at the merge key's position, in the aliased mapping's own order**.
- Precedence: explicit keys of the containing mapping override merged pairs (probed: `<<: *b` + later or earlier explicit `c:` → explicit value wins, both orders); with a sequence `<<: [*a, *b]`, **earlier** sources win (probed: `k` from `*a` beats `*b`). Implementation: two passes over the mapping's pairs — collect explicit keys, then place merged pairs not already present.
- Merge keys inside merged mappings recurse (yaml spec allows nesting).

**Order rule (the contract, one sentence):** mapping keys are emitted in document order — the order the pairs appear in the YAML source, with `<<` expansions inserted at the merge key's position; sequences preserve element order; nothing is sorted anywhere.

### Implementation Plan

**Phase 1: Spike + walker (M1, ~3h)**
- [ ] Compiled spike pinning the yaml.v3 Node mechanics against V7 of the Verification Log (this authoring machine has no Go toolchain — the spike runs on the implementer's): `yaml.Unmarshal(data, &node)` reads **only the first document** (upstream docstring says so; pin `"a: 1\n---\nb: 2\n"` → `{"a":1}`); `node.Alias` is non-nil for alias nodes and points at the anchored node; merge keys carry `Tag == "!!merge"`; scalar `Decode` resolves timestamps/tags as above (pin `t: 2026-10-07` → `"2026-10-07T00:00:00Z"`, `a: !!str 1` → `"1"`).
- [ ] `emitNode` walker in `internal/builtins/yaml.go` per Architecture; delete the `interface{}` path.
- [ ] Unit tests: order (b,a,c), nested order, sequence order, flow `{b: 2, a: 1}` → `{"b":2,"a":1}`.

**Phase 2: Guarantee regression matrix (~3h)**
- [ ] Walk the guarantees table below and confirm each row has a test; add the missing ones (dup-key `Err`, merge ordering, merge bad-value `Err`, alias-of-mapping order, big-int digits).
- [ ] Re-pin the two intentional changes: `TestYAMLToJSON_BlockMapping` and `tests/yaml_bridge_test.ail` Test 1; fix stale comments ("encoding/json sorts object keys", "yamlToJson produces JSON with sorted keys").
- [ ] `TestYAMLToJSON_Deterministic` unchanged in assertion, updated in comment — document order must be stable across 100 passes (it is: no Go map participates).

**Phase 3: Contract on paper (~1h)**
- [ ] `BuiltinMetadata.LongDesc` + `std/yaml.ail` doc comments: state the order rule.
- [ ] `docs/docs/reference/std-yaml.md`: add a "**Key order**" bullet under "Semantics and limits" ("mapping keys are emitted in document order; nothing is sorted") and adjust the example comment if needed.
- [ ] CHANGELOG entry under the current version.

### Files to Modify/Create

**Modified files:**
- `internal/builtins/yaml.go` — replace `yamlToJSONImpl` body + add `emitNode` (+ ~110 LOC, − ~10).
- `internal/builtins/yaml_test.go` — order tests + guarantee matrix additions, re-pin 1 test (+ ~50 LOC).
- `tests/yaml_bridge_test.ail` — re-pin Test 1 expected string, fix comment (~3 lines).
- `std/yaml.ail` — doc comments only.
- `docs/docs/reference/std-yaml.md` — order-contract bullet (~6 lines).
- `CHANGELOG.md` / current changelog file — entry.

**No new files.** Total ~150–200 LOC, consistent with the triage estimate.

## Guarantees table (all probed on dev, v0.52.5, 2026-10-07)

| Input (YAML) | Today's output | Must remain |
|---|---|---|
| `b: 1\na: 2\nc: 3\n` | `{"a":2,"b":1,"c":3}` | `{"b":1,"a":2,"c":3}` ← the fix |
| `a: &x 1\nb: *x\n` | `{"a":1,"b":1}` | same |
| `a: &x\n  k: 1\nb: *x\n` | `{"a":{"k":1},"b":{"k":1}}` | same |
| `base: &b\n  a: 1\nchild:\n  <<: *b\n  c: 3\n` | `{"base":{"a":1},"child":{"a":1,"c":3}}` | same, `a` before `c` |
| `child:\n  c: 3\n  <<: *b` (explicit vs merge conflict) | explicit wins | same rule |
| `<<: [*a, *b]` conflict | earlier source wins | same rule |
| `child:\n  <<: 1\n` | `Err: yaml: map merge requires map or sequence of maps as the value` | same `Err` (message may vary per D5 only for *other* rows; keep this one) |
| `t: 2026-10-07\n` | `{"t":"2026-10-07T00:00:00Z"}` | same |
| `a: !!str 1\n` / `a: !!int "5"\n` | `{"a":"1"}` / `{"a":5}` | same |
| `x: .nan` / `.inf` / `-.inf` | `Err: yaml: cannot represent as JSON: json: unsupported value: NaN (+Inf/-Inf)` | same `Err` |
| `1: a` / `true: b` / `~: 1` | `Err` (non-string key) | same `Err` |
| `"1": a\n` (quoted key) | `{"1":"a"}` | same |
| `a: 1\na: 2\n` | `Err: mapping key "a" already defined at line 1` | same `Err` (text may vary, D5) |
| `a: 1\n---\nb: 2\n` | `{"a":1}` (first doc only) | same |
| `` (empty) / `\n` | `null` | same |
| `m: {}` / `s: []` | `{"m":{}}` / `{"s":[]}` | same |
| `s: \|\n  line1\n  line2\n` | `{"s":"line1\nline2\n"}` | same |
| `n: 9007199254740993\n` | `{"n":9007199254740993}` (int64 preserved in the JSON string) | same |
| `n: 0x1F` / `n: 0o17` | `{"n":31}` / `{"n":15}` | same |
| `n: 1:30\n` | `{"n":"1:30"}` (sexagesimal not resolved) | same |
| `a: 1\n  b: 2\n` (malformed) | `Err` (parse) | same |

## Examples

### Before (dev, v0.52.5 — verified live)

```ailang
match yamlToJson("b: 1\na: 2\nc: 3\n") {
  Ok(j) => println(j)    -- prints {"a":2,"b":1,"c":3}  (order silently destroyed)
}
```

### After

```ailang
module examples/key_order
import std/yaml (yamlToJson, decode)
import std/json (keys)
import std/result (Ok, Err)

export func main() -> () ! {IO} {
  match yamlToJson("b: 1\na: 2\nc: 3\n") {
    Ok(j) => println("json: ${j}"),   -- json: {"b":1,"a":2,"c":3}
    Err(e) => println("yaml failed: ${e}")
  };
  match decode("b: 1\na: 2\nc: 3\n") {
    Ok(doc) => println("keys: ${show(keys(doc))}"),   -- keys: [b, a, c]
    Err(e)  => println("yaml failed: ${e}")
  }
}
```

Both probes were run on dev today; the "after" outputs are what the walker produces given document order (V1, V2).

## Success Criteria

- [ ] `yamlToJson("b: 1\na: 2\nc: 3\n")` → `Ok("{\"b\":1,\"a\":2,\"c\":3}")` — pinned unit test.
- [ ] Every row of the guarantees table has a passing test; no `Err` path became `Ok` and vice versa.
- [ ] `std/yaml.decode` and `std/json.decode` agree on key order for equivalent docs (pinned test).
- [ ] M1 spike findings recorded (or pointer to test) for: first-document-only into `yaml.Node`, `Alias` pointers, `!!merge` tag, timestamp/tag resolution.
- [ ] `examples/runnable/yaml_config.ail` still runs unchanged; `ailang check std/yaml.ail` clean.
- [ ] `docs/docs/reference/std-yaml.md` states the order contract; builtin `LongDesc` states it.
- [ ] All tests passing; CHANGELOG entry present.

## Testing Strategy

**Unit tests (Go, `yaml_test.go`):** the guarantees table above as a table-driven suite (most rows already exist as `TestYAMLToJSON_*`); new: document-order, nested-order, flow-vs-block equality still holds (`TestYAMLToJSON_FlowStyleEqualsBlock` — block and flow produce the *same node order*, so the equality assertion survives), merge ordering/precedence/bad-value, dup-key `Err`, key JSON-escaping, alias-of-mapping order.

**Integration (`.ail`, `tests/yaml_bridge_test.ail`):** Test 1 re-pins to `{"name":"STX","count":3,"items":["a","b"]}`; Tests 2–7 unchanged.

**Manual:** `ailang run --caps IO --entry main examples/runnable/yaml_config.ail`; `GOOS=js GOARCH=wasm go build ./...` (yaml.v3 already verified wasm-buildable by m-std-yaml; the walker adds no imports beyond `bytes`).

## Deferred Decisions

- Exact `Err` text for the non-string-key and duplicate-key checks (D5) — agent may choose; must keep the `yaml:` prefix and mention the offending key.
- Whether the key string is emitted via `json.Marshal(string)` or `encoding/json`'s `Encoder` with `SetEscapeHTML(true)` equivalence — agent may choose, but scalar bytes must match today's `json.Marshal` output (HTML escaping included) so the change is order-only.
- Whether to fold the walker into a method vs free function — agent may choose.

## Non-Goals

- **Multi-document streams** (`decodeAll`) — unchanged future work.
- **YAML emission** (`encode`) — separate triage item (Secondary 2), separate doc.
- **Preserving anchors/aliases/comments in output** — output remains plain JSON; only *order* is new.
- **Sorting option or stable-order flag** — no knob; document order is the contract.

## Timeline

Single day: M1 spike + walker (~3h), guarantee matrix + re-pins (~3h), docs + changelog (~1h). **Total ~7 hours.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| yaml.v3 Node semantics assumed here are wrong on some edge (e.g. `Unmarshal` into `Node` and multi-doc) | Med | M1 spike pins each assumption as a test before the guarantee matrix lands; any surprise surfaces in Phase 1, not in production. |
| A current `Err` path silently becomes `Ok` (or reverse) in the rewrite | High | The guarantees table is the regression matrix; CI runs it verbatim. |
| Unknown consumers pin the old sorted bytes | Low | Grep shows no in-repo consumers beyond `std/yaml.ail` + tests; CHANGELOG entry flags the byte change explicitly. |
| Merge-key precedence detail diverges from yaml.v3's decoder | Low | Precedence rules were probed live on dev (V4) and pinned as tests, not inferred from upstream source. |

## Verification Log

| # | Claim | How verified |
|---|-------|--------------|
| V1 | Bug reproduces: `yamlToJson("b: 1\na: 2\nc: 3\n")` → `{"a":2,"b":1,"c":3}`; `decode` → keys `[a,b,c]` | Live `ailang run` on dev (v0.52.5) temp module, 2026-10-07. |
| V2 | `std/json` preserves order: `jsonDecode("{\"b\":1,\"a\":2,\"c\":3}")` → keys `[b,a,c]` | Live run **plus** code path read: `std/json.ail:18` (`decode` → `_json_decode`) → `internal/builtins/json_decode.go` token-streaming `JSONBuilder` (`Token()` at ~294, `popObject` appends `kvPairs` in token order, ~381). Not inferred from output. |
| V3 | `keys(obj: Json) -> [string]` exists in std/json | `std/json.ail:167`. |
| V4 | Every row of the guarantees table is today's actual behavior | Live probes through the shipped builtin (`ailang run`, 30+ inputs incl. merge variants, timestamps, tags, NaN/Inf, dup keys, multi-doc, empty, big-int, hex/octal, block scalars) — see table. |
| V5 | No other Go callers of `_yaml_to_json`/`yamlToJSONImpl` | `grep -rn` across `internal/` and `cmd/`: only `yaml.go`, `yaml_test.go`. |
| V6 | Order contract undocumented today | Read in full: `docs/docs/reference/std-yaml.md` (no order statement), `design_docs/implemented/v0_30_0/m-std-yaml.md` (mentions sorting only incidentally in A1 and an example, never as contract). |
| V7 | yaml.v3 Node mechanics (`Unmarshal` into `Node` = first document only; `node.Alias` pointers; `!!merge` tag on `<<` keys; scalar `node.Decode` resolves tags/timestamps like document decode) | **Upstream-cited, NOT machine-verified here**: this authoring machine has no Go toolchain (module cache absent; only the `ailang` binary is installed). Claims cite `gopkg.in/yaml.v3 v3.0.1` (`go.mod`) docs; the M1 spike in Phase 1 converts each into a compiled test before the walker is trusted. Scalar-level behavior was probed live through the current builtin (V4), so only the Node-API *mechanism* is assumption. |
| V8 | Negative-existence: no `decodeAll`/`encode` in `std/yaml` today; no order knob exists | Read `std/yaml.ail` in full (exports: `yamlToJson`, `decode` only). |
| V9 | ~150–200 LOC total is realistic | Walker is structurally comparable to `_json_decode`'s builder (~170 LOC) but tree-walking, no frames; triage's independent estimate (~100–200) agrees. |
| V10 | No prior/queued doc covers this (dup gate) | create-script SimHash + neural search over `implemented/` and `planned/`: "(none found)"; the triage doc's own search concurs. |

## Related Documents

- [design_docs/implemented/v0_30_0/m-std-yaml.md](../implemented/v0_30_0/m-std-yaml.md) — the shipped bridge this doc amends; its "map keys sorted" phrasing becomes historical.
- [docs/docs/reference/std-yaml.md](../../../docs/docs/reference/std-yaml.md) — the user-facing contract page to be updated.
- [design_docs/planned/ailang-core-triage/yaml-decode-loses-mapping-key-order.md](../ailang-core-triage/yaml-decode-loses-mapping-key-order.md) — triage source (Primary item). Secondary 1 (`with`/`recv` reserved words) and Secondary 2 (yaml encode) are **deliberately out of scope here** per the triage dispatch note: separate subsystems, separate docs.
- [internal/builtins/json_decode.go](../../../internal/builtins/json_decode.go) — the ordered-decoder pattern the walker mirrors.

## References

- [Design Axioms](/docs/references/axioms)
- `gopkg.in/yaml.v3 v3.0.1` (go.mod) — `Node`, `Kind` constants, `Alias`, `Tag`, `Decode`.
- Report: `ailang-core` inbox `inbox_1791394438696_00b01c33`.

---

**Document created**: 2026-10-07
**Last updated**: 2026-10-07
## Implementation handoff — 2026-10-08

Approved option (a) is implemented. See the companion sprint plan execution record and `.ailang/state/sprints/sprint_M-STD-YAML-MAPPING-KEY-ORDER.json` for milestones and targeted validation under the crash-recovery constraints. Upstream generic decoding is retained only as discarded validation for alias bounds and typed errors; emitted bytes come from an ordered Node walk with a shared buffer. Independent review is complete; coordinator evaluation and publication remain downstream.
