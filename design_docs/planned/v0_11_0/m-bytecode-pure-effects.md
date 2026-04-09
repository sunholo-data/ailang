# M-BYTECODE-PURE-EFFECTS: Wire Pure-in-Practice Builtins to VM

**Status**: Planned
**Priority**: P1 (next EvalOnly reduction after lambda resolution)
**Target**: v0.11.0
**Estimated LOC**: ~600
**Dependencies**: M-BYTECODE-LAMBDA-RESOLUTION (complete)

## 1. Problem Statement

After M-BYTECODE-LAMBDA-RESOLUTION, **92 / 1204** prototypes remain EvalOnly. Of these, **73 are "effectful builtins"** that the compiler cannot wire because they weren't in `BuiltinTable`. However, investigation reveals that **many are functionally pure** — they ignore their `EffContext` parameter and perform no I/O:

| Category | Count | IsPure | Uses EffContext | Can Wire Now |
|---|---|---|---|---|
| XML | 17 | true | No (underscore param) | Yes, but needs ADT construction |
| Bytes | 10 | true | No | Yes, needs TagBytes |
| Map | 10 | true | No | Yes, needs TagMap |
| JSON | 3 | true | No | **Yes — strings + ADTs only** |
| ZIP | 6 | false | Yes (Sandbox) | No — truly effectful (FS) |
| FS | 13 | false | Yes | No — truly effectful |
| IO | 5 | false | Yes | No — truly effectful |
| AI | 5 | false | Yes | No — truly effectful |
| Clock | 2 | false | Yes | No — truly effectful |
| Env | 3 | false | Yes | No — truly effectful |
| Net | 1+2 | mixed | Yes | No — truly effectful |

**This sprint targets JSON (3 builtins)** — the simplest category with the highest certainty of success. JSON builtins use only strings and Result ADTs, both already supported by the VM. XML and Bytes/Map require new VM value types (TagBytes, TagMap, TagXmlNode or ADT-based XmlNode construction), which are separate, larger efforts.

## 2. Why JSON First

1. **Zero new VM types needed** — JSON encode returns string, decode returns `Result[Json, string]` (ADT), repair returns string. All types exist.
2. **Proven pattern** — `stringToInt`/`stringToFloat` already create ADT values natively in the VM via `bytecode.NewADT()`.
3. **JSON is heavily used in docparse** — formatters, serializers, MCP tools all use JSON encode/decode.
4. **De-risks the approach** — Validates that pure-in-practice builtins from `internal/builtins/` can be wired to the VM without the EffContext plumbing. If this works cleanly, XML/Bytes/Map follow the same pattern.

## 3. JSON Builtin Analysis

### 3.1 `__json_encode`: Json -> string
Recursively encodes a `Json` ADT value to a JSON string. The `Json` ADT has variants: `JsonNull`, `JsonBool(bool)`, `JsonInt(int)`, `JsonFloat(float)`, `JsonString(string)`, `JsonArray([Json])`, `JsonObject([(string, Json)])`.

**VM implementation**: Receive `TagADT` value, switch on variant tag, recursively build string. Return `TagString`.

### 3.2 `__json_decode`: string -> Result[Json, string]
Parses a JSON string into a `Json` ADT tree.

**VM implementation**: Parse with Go's `encoding/json`, construct `Json` ADT values via `bytecode.NewADT()`. Return `Result` ADT: `Ok(json)` or `Err(message)`.

### 3.3 `__json_repair`: string -> string
Repairs truncated JSON by closing unclosed brackets/strings.

**VM implementation**: Pure string manipulation. Return `TagString`.

## 4. ADT Tag Mapping Challenge

The VM needs to know the integer tag indices for ADT variants at compile time. Currently, `stringToInt` uses hardcoded `optionTagSome = 0` and `optionTagNone = 1`. For JSON builtins, we need tag indices for:

- `Result`: `Ok = 0`, `Err = 1`
- `Json`: `JsonNull = 0`, `JsonBool = 1`, `JsonInt = 2`, `JsonFloat = 3`, `JsonString = 4`, `JsonArray = 5`, `JsonObject = 6`

**Approach**: Define these as constants in `internal/vm/builtins_json.go`, matching the order declared in the AILANG stdlib. Add a comment noting the dependency on stdlib ADT declaration order.

## 5. Implementation Plan

### Files

| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_json.go` | New: 3 JSON builtin handlers | 250 |
| `internal/vm/builtins_json_test.go` | New: unit tests for all 3 | 200 |
| `internal/vm/builtins.go` | Add 3 entries to BuiltinTable | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 3 names to BuiltinTable | 5 |

### Implementation Details

```go
// builtins_json.go

// Json ADT tag indices (must match stdlib declaration order)
const (
    jsonTagNull   = 0
    jsonTagBool   = 1
    jsonTagInt    = 2
    jsonTagFloat  = 3
    jsonTagString = 4
    jsonTagArray  = 5
    jsonTagObject = 6
)

// Result ADT tag indices
const (
    resultTagOk  = 0
    resultTagErr = 1
)

func builtinJsonEncode(args []bytecode.Value) (bytecode.Value, error) {
    // Walk Json ADT tree, build string
}

func builtinJsonDecode(args []bytecode.Value) (bytecode.Value, error) {
    // Parse JSON string, build Json ADT tree via NewADT
    // Return Result[Json, string]
}

func builtinJsonRepair(args []bytecode.Value) (bytecode.Value, error) {
    // Pure string manipulation
}
```

## 6. Success Criteria

| Metric | Current | Target |
|---|---|---|
| EvalOnly count | 92 / 1204 | **≤ 89 / 1204** (direct: -3) |
| Parity MATCH | 129 | **≥ 129** (no regression) |
| JSON encode/decode | EvalOnly bridge | Native VM dispatch |

## 7. Future Phases (Out of Scope)

After JSON validates the approach:

1. **Phase 2: XML builtins (17)** — Requires mapping XmlNode ADT tags, recursive tree construction in VM. ~400 LOC.
2. **Phase 3: Bytes builtins (10)** — Requires TagBytes VM value type. ~200 LOC.
3. **Phase 4: Map builtins (10)** — Requires TagMap VM value type. ~300 LOC.
4. **Phase 5: Effectful builtins (34)** — Requires new `OpEffectCall` opcode with EffContext plumbing. Separate design doc.

## 8. Risks

### R1: ADT tag order mismatch
**Risk**: If stdlib changes the Json or Result ADT declaration order, the hardcoded tag indices will silently produce wrong results.
**Mitigation**: Add a compile-time or startup-time assertion that verifies tag indices match. Document the dependency clearly.

### R2: Json ADT has nested structure
**Risk**: `JsonArray` contains `[Json]` and `JsonObject` contains `[(string, Json)]` — deep nesting requires recursive VM value construction.
**Mitigation**: The evaluator already handles this; port the recursive logic. Test with deeply nested JSON.

### R3: Tuple representation for JsonObject entries
**Risk**: `JsonObject([(string, Json)])` uses tuples `(string, Json)`. The VM may not have a tuple type.
**Mitigation**: Check how tuples are represented — likely as TagADT with a tuple variant, or as TagList of 2 elements. Match the evaluator's representation.

## 9. Related Documents

- [m-bytecode-vm.md §18.9](../../implemented/v0_11_0/m-bytecode-vm.md) — Lambda resolution results
- [m-bytecode-vm.md §18.10](../../implemented/v0_11_0/m-bytecode-vm.md) — Next steps
- [m-bytecode-hof-builtins.md](m-bytecode-hof-builtins.md) — HOF infrastructure (prerequisite)
