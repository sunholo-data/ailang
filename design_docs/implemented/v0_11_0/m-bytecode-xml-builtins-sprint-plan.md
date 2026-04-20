# M-BYTECODE-XML-BUILTINS Sprint Plan

**Design doc**: [m-bytecode-xml-builtins.md](m-bytecode-xml-builtins.md)
**Sprint ID**: `M-BYTECODE-XML-BUILTINS`
**Estimated**: 1 day (~850 LOC)
**Dependencies**: M-BYTECODE-REGALLOC-FIX (complete)

## Milestones

### M1: XmlNode Value Converters + Constructor Builtins (3 builtins)

**Scope**: Create `internal/vm/builtins_xml.go` with XmlNode↔bytecode conversion helpers, then wire the 3 constructor builtins (`xmlElement`, `xmlText`, `xmlComment`).

**Strategy**: The VM builtins call the existing Go implementations in `internal/builtins/xml*.go` which return `eval.TaggedValue`. A converter transforms `eval.TaggedValue` (XmlNode) → `bytecode.ADTObj` recursively. This is the foundation all other XML builtins depend on.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_xml.go` | New: converters + 3 constructor builtins | 200 |
| `internal/vm/builtins_xml_test.go` | New: converter + constructor tests | 100 |
| `internal/vm/builtins.go` | Add 3 entries to BuiltinTable | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 3 names to BuiltinTable | 5 |

**Acceptance criteria**:
- `evalToXmlNode(bytecode.Value) → eval.Value` converts ADT → TaggedValue
- `xmlNodeToBytecode(eval.Value) → bytecode.Value` converts TaggedValue → ADT
- `xmlText("hello")` produces `ADT{tag=1, fields=[String("hello")]}`
- `xmlElement("div", [...], [...])` produces correct nested ADT with record attrs
- Converter handles recursive Element trees (depth ≤ 256)
- Unit tests with `-count=20` determinism check
- `make test` passes, `make lint` passes

**Est. LOC**: 310

---

### M2: String-Returning Builtins (4 builtins)

**Scope**: Wire `getText`, `getTag`, `serialize`, `serializeWithDecl`. These take XmlNode ADT args, convert to eval.TaggedValue, call existing Go implementation, return string.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_xml.go` | Add 4 builtin handlers | 80 |
| `internal/vm/builtins_xml_test.go` | Add 4 test suites | 80 |
| `internal/vm/builtins.go` | Add 4 entries | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 4 names | 5 |

**Acceptance criteria**:
- `getText(Element("p", [], [Text("hello")]))` returns `"hello"`
- `getTag(Element("div", [], []))` returns `"div"`
- `serialize(Element("p", [], [Text("hi")]))` returns `"<p>hi</p>"`
- `serializeWithDecl(...)` returns `<?xml ...?><p>hi</p>`
- Unit tests with `-count=20`
- `make test` + `make lint` pass

**Est. LOC**: 170

---

### M3: Parse Builtins (3 builtins)

**Scope**: Wire `parse`, `parseElements`, `parseWithLimit`. These parse XML strings and return `Result[XmlNode, string]` ADT values.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_xml.go` | Add 3 builtin handlers | 60 |
| `internal/vm/builtins_xml_test.go` | Add 3 test suites | 80 |
| `internal/vm/builtins.go` | Add 3 entries | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 3 names | 5 |

**Acceptance criteria**:
- `parse("<root>hi</root>")` returns `Ok(Element("root", [], [Text("hi")]))`
- `parse("bad xml")` returns `Err("...")`
- `parseElements("<items><item>a</item></items>", "item", 10)` returns `Ok([Element("item",...)])`
- `parseWithLimit("<root>...</root>", 100)` returns `Ok(...)` or `Err(...)`
- Unit tests with `-count=20`
- `make test` + `make lint` pass

**Est. LOC**: 150

---

### M4: Query Builtins (6 builtins)

**Scope**: Wire `findAll`, `findFirst`, `getAttr`, `getChildren`, `findAllTexts`, `findAllAttrs`.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_xml.go` | Add 6 builtin handlers | 120 |
| `internal/vm/builtins_xml_test.go` | Add 6 test suites | 120 |
| `internal/vm/builtins.go` | Add 6 entries | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 6 names | 5 |

**Acceptance criteria**:
- `findAll(root, "item")` returns `[XmlNode]` list
- `findFirst(root, "item")` returns `Some(XmlNode)` or `None`
- `getAttr(element, "class")` returns `Some("main")` or `None`
- `getChildren(element)` returns `[XmlNode]` list
- `findAllTexts(root, "item")` returns `[string]`
- `findAllAttrs(root, "item", "id")` returns `[string]`
- Unit tests with `-count=20`
- `make test` + `make lint` pass

**Est. LOC**: 250

---

### M5: parseFold HOF Builtin + Verify and Close (1 builtin)

**Scope**: Wire `parseFold` as HOF builtin using `HOFBuiltinTable` and `ClosureCaller`. Then verify EvalOnly reduction and update docs.

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/vm/builtins_xml.go` | Add parseFold HOF handler | 60 |
| `internal/vm/builtins_xml_test.go` | Add parseFold test | 40 |
| `internal/vm/builtins_hof.go` | Add 1 entry to HOFBuiltinTable | 5 |
| `internal/bytecode/compiler/builtins.go` | Add 1 name to HOFBuiltinTable | 5 |
| `changelogs/v0.10-current.md` | Add changelog entry | 10 |

**Acceptance criteria**:
- `parseFold(xml, "item", [], \acc node. acc ++ [getText(node)])` returns `Ok([...])`
- EvalOnly count ≤ 57 (down from 74, saving 17)
- Parity: ≥ 129 MATCH, no regressions
- CHANGELOG updated
- `make test` + `make lint` pass

**Dependencies**: M1, M2, M3, M4
**Est. LOC**: 120
