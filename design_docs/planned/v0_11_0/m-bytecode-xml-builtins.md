# M-BYTECODE-XML-BUILTINS: Wire XML Builtins to VM

**Status**: Planned
**Priority**: P1 (17 EvalOnly prototypes, largest single-category reduction)
**Target**: v0.11.0
**Estimated LOC**: ~800-1000
**Dependencies**: M-BYTECODE-REGALLOC-FIX (complete)

## 1. Problem Statement

After M-BYTECODE-REGALLOC-FIX, **74 / 1228** prototypes remain EvalOnly. Of these, **17 are XML builtins** — the largest single category. All 17 are functionally pure (ignore their `EffContext` param) except `parseFold` which uses `ctx.FnCallerN` for HOF callback. XML builtins are heavily used in docparse for DOCX/EPUB/EML parsing.

### Current EvalOnly Breakdown (74 total)

| Category | Count | Blocker |
|----------|-------|---------|
| XML | 17 | No XmlNode ADT in VM, bridge rejects TaggedValue |
| Bytes | 10 | No TagBytes in VM |
| Map | 10 | No TagMap in VM |
| FS | 13 | Truly effectful |
| ZIP | 6 | Truly effectful (uses FS sandbox) |
| AI | 5 | Truly effectful |
| IO | 5 | Truly effectful |
| Env | 3 | Truly effectful |
| Clock | 2 | Truly effectful |
| Net | 1 | Truly effectful |
| Unbound var | 2 | Nested pattern match lowering |

## 2. XmlNode ADT

Declared in `std/xml.ail`:
```
type XmlNode =
  | Element(string, [{name: string, value: string}], [XmlNode])
  | Text(string)
  | Comment(string)
```

Tag ordinals (matching declaration order): Element=0, Text=1, Comment=2.

The `Element` constructor has 3 fields:
- Field 0: tag name (`string`)
- Field 1: attributes (`[{name: string, value: string}]`) — list of anonymous records
- Field 2: children (`[XmlNode]`) — recursive list

## 3. XML Builtin Analysis

### 3.1 By Implementation Strategy

**Group A — Pure, no XmlNode in args or return (5 builtins)**

These operate on strings and lists of strings. Can be wired directly as `BuiltinFunc` with no new types.

| Builtin | Signature | Notes |
|---------|-----------|-------|
| `__xml_getText` | XmlNode → string | Walk ADT tree, extract text |
| `__xml_getTag` | XmlNode → string | Read Element tag name (field 0) |
| `__xml_serialize` | XmlNode → string | ADT tree → XML string |
| `__xml_serializeWithDecl` | XmlNode → string | Same with `<?xml?>` header |
| `__escapeXml` | string → string | Already wired as pure stdlib builtin! |

Note: `__escapeXml` is already in BuiltinTable from M-BYTECODE-STDLIB-BUILTINS. Only 4 new in this group.

**Group B — Return XmlNode ADT (6 builtins)**

These parse XML strings into XmlNode ADT trees. The VM builtin must construct `bytecode.ADTObj` values (Element/Text/Comment) natively.

| Builtin | Signature | Notes |
|---------|-----------|-------|
| `__xml_parse` | string → Result[XmlNode, string] | Full XML parse |
| `__xml_parseElements` | string, string, int → Result[[XmlNode], string] | Streaming parse |
| `__xml_parseWithLimit` | string, int → Result[XmlNode, string] | Bounded parse |
| `__xmlElement` | string, [{name,value}], [XmlNode] → XmlNode | Constructor helper |
| `__xmlText` | string → XmlNode | Constructor helper |
| `__xmlComment` | string → XmlNode | Constructor helper |

**Group C — XmlNode in args, return lists/options (6 builtins)**

These query XmlNode trees and return lists or Option values.

| Builtin | Signature | Notes |
|---------|-----------|-------|
| `__xml_findAll` | XmlNode, string → [XmlNode] | Recursive tag search |
| `__xml_findFirst` | XmlNode, string → Option[XmlNode] | First match |
| `__xml_getAttr` | XmlNode, string → Option[string] | Attribute lookup |
| `__xml_getChildren` | XmlNode → [XmlNode] | Direct children |
| `__xml_findAllTexts` | XmlNode, string → [string] | Fused findAll+getText |
| `__xml_findAllAttrs` | XmlNode, string, string → [string] | Fused findAll+getAttr |

**Group D — HOF (1 builtin)**

| Builtin | Signature | Notes |
|---------|-----------|-------|
| `__xml_parseFold` | string, string, a, (a, XmlNode) → a → Result[a, string] | HOF callback via ClosureCaller |

### 3.2 Key Design Decision: Native ADT Construction

The VM needs to construct XmlNode as `bytecode.ADTObj` values. This follows the same pattern established by JSON builtins in M-BYTECODE-PURE-EFFECTS:

```go
// XmlNode tag constants (must match std/xml.ail declaration order)
const (
    xmlNodeTagElement = 0
    xmlNodeTagText    = 1
    xmlNodeTagComment = 2
)
```

**Element construction** is more complex than JSON because field 1 is a list of anonymous records (`[{name: string, value: string}]`). Each attribute becomes a `bytecode.RecordObj` with sorted field names `["name", "value"]`.

### 3.3 Approach: Reuse Evaluator Implementations

Rather than reimplementing XML parsing in the VM, the most efficient approach is to **call the existing evaluator-side implementations** but convert their return values to bytecode values. This requires:

1. A `TaggedValue → bytecode.Value` converter for XmlNode
2. Registration in both compiler and VM BuiltinTables

The existing Go implementations in `internal/builtins/xml*.go` already do the hard work (XML parsing, tree walking, serialization). The VM builtins can call these directly and convert the results.

**Alternative**: Implement XML operations directly on `bytecode.ADTObj` trees. This avoids the conversion overhead but requires reimplementing all tree-walking logic. Better suited for a future optimization if XML parsing becomes a bottleneck.

## 4. Implementation Plan

### Phase 1: XmlNode Value Helpers

Create `internal/vm/builtins_xml.go` with:
- XmlNode tag constants
- `xmlNodeToBytecode(eval.Value) → bytecode.Value` converter
- `bytecodeToXmlNode(bytecode.Value) → eval.Value` converter (for builtins that take XmlNode args)
- Helper to construct `Result[XmlNode, string]` ADT values

### Phase 2: Wire Group A (string→string builtins)

Wire `getText`, `getTag`, `serialize`, `serializeWithDecl`. These take XmlNode ADT args from the VM, convert to eval.TaggedValue, call the existing Go implementation, and return the string result.

### Phase 3: Wire Group B (XmlNode constructors + parsers)

Wire `parse`, `parseElements`, `parseWithLimit`, `xmlElement`, `xmlText`, `xmlComment`. These call the existing Go parsers and convert the returned TaggedValue trees to bytecode ADT trees.

### Phase 4: Wire Group C (query builtins)

Wire `findAll`, `findFirst`, `getAttr`, `getChildren`, `findAllTexts`, `findAllAttrs`. Similar pattern: convert args, call existing Go, convert results.

### Phase 5: Wire Group D (parseFold HOF)

Wire `parseFold` using `HOFBuiltinTable` and `ClosureCaller` (same infrastructure as `list_map`/`list_filter` from M-BYTECODE-HOF-BUILTINS).

## 5. Files

| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_xml.go` | New: 17 XML builtin handlers + value converters | 500 |
| `internal/vm/builtins_xml_test.go` | New: unit tests for all builtins | 300 |
| `internal/vm/builtins.go` | Add 16 entries to BuiltinTable | 20 |
| `internal/vm/builtins_hof.go` | Add 1 entry to HOFBuiltinTable (parseFold) | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 16+1 names to tables | 20 |

**Total estimated**: ~850 LOC

## 6. XmlNode ↔ bytecode.Value Conversion

The critical helper converts between evaluator's `*eval.TaggedValue` and VM's `bytecode.ADTObj`:

```go
// xmlNodeToBytecode converts an evaluator XmlNode (TaggedValue) to a
// bytecode ADT value. Recursively converts Element children.
func xmlNodeToBytecode(v eval.Value) (bytecode.Value, error) {
    tv, ok := v.(*eval.TaggedValue)
    if !ok {
        return bytecode.Value{}, fmt.Errorf("expected TaggedValue, got %T", v)
    }
    switch tv.CtorName {
    case "Element":
        tag := tv.Fields[0].(eval.StringVal)      // tag name
        attrs := tv.Fields[1]                       // [{name, value}]
        children := tv.Fields[2]                    // [XmlNode]
        // Convert attrs list: each attr is a record {name: string, value: string}
        // Convert children list: recursive xmlNodeToBytecode
        ...
    case "Text":
        text := tv.Fields[0].(eval.StringVal)
        return bytecode.NewADT(xmlNodeTagText, []bytecode.Value{
            bytecode.NewString(string(text)),
        }), nil
    case "Comment":
        // similar to Text
    }
}
```

## 7. Acceptance Criteria

- All 17 XML "effectful builtin not wired" EvalOnly prototypes compile to bytecode
- EvalOnly count: ≤ 57 (down from 74, saving 17)
- Parity: ≥ 129 MATCH, no regressions
- Unit tests for all 17 builtins with `-count=20` determinism checks
- `make test` passes
- `make lint` passes

## 8. Risks

- **TaggedValue ↔ ADT conversion overhead**: Converting between evaluator and VM value representations adds per-call overhead. For hot-path XML parsing in docparse, this could negate the VM's speed advantage. Mitigation: profile after wiring; if slow, reimplement core operations directly on ADT values.
- **Anonymous record sorting**: The attribute records `{name, value}` have sorted field order in the VM (`["name", "value"]`). This must match the evaluator's field order. Mitigation: verify in tests.
- **parseFold HOF complexity**: The polymorphic callback type and accumulator threading are non-trivial. Mitigation: implement last, after all other builtins are validated.
- **Recursive XmlNode trees**: Deep XML documents produce deep ADT trees. The conversion is recursive and could stack-overflow on pathological inputs. Mitigation: the existing Go implementations already handle this.

## 9. Future Work

After XML builtins, the remaining effectful categories are:
- **Bytes (10)**: Requires `TagBytes` VM value type — separate design doc
- **Map (10)**: Requires `TagMap` VM value type — separate design doc  
- **FS/IO/Env/Clock/Net/AI/ZIP (30)**: Truly effectful, need EffContext plumbing in VM — Phase 3 scope
