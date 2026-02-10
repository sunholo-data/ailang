# M-STDLIB-XML: XML Parsing Standard Library

**Status**: Planned
**Target**: v0.7.3
**Priority**: P2 (Medium)
**Estimated**: 2-3 days
**Dependencies**: None (Go `encoding/xml` stdlib)
**Author**: Claude (Opus 4.6)
**Date**: 2026-02-08
**Requested by**: docparse-demo agent (message 62662386)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same XML string → same XmlNode tree (pure, deterministic) |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | All functions are pure — no hidden effects |
| A4: Explicit Authority | 0 | No capabilities needed (pure functions) |
| A5: Bounded Verification | +1 | XmlNode ADT enables exhaustive pattern matching |
| A6: Safe Concurrency | 0 | Pure functions, no shared state |
| A7: Machines First | +1 | Tree ADT is machine-queryable; enables AI document processing |
| A8: Minimal Syntax | 0 | No new syntax — builtins only |
| A9: Cost Visibility | 0 | No new cost model |
| A10: Composability | +1 | Composes with std/zip, std/json; tree queries compose naturally |
| A11: Structured Failure | +1 | `parseXml` returns `Result[XmlNode, string]`; query functions return `Option` |
| A12: System Boundary | 0 | No boundary crossing (pure string → tree transform) |

**Net Score: +6** → **Decision: Accept**

### Hard Violation Check

- [x] A1 (Determinism): Pure functions, fully deterministic
- [x] A3 (Effects): No hidden side effects — all functions are pure
- [x] A4 (Authority): No ambient access needed
- [x] A7 (Machines First): Tree ADT is ideal for machine analysis

## Related Documents

- [m-stdlib-zip.md](m-stdlib-zip.md) — Companion module for reading ZIP archives (DOCX = ZIP + XML)
- [m-stdlib-datetime.md](../../implemented/v0_7_0/m-stdlib-datetime.md) — Similar stdlib pattern (pure math on data)
- [m-stdlib-gaps.md](../v0_7_2/m-stdlib-gaps.md) — Stdlib gap analysis from eval results
- [m-wasm-stdlib.md](../v0_7_2/m-wasm-stdlib.md) — WASM stdlib embedding (new modules need WASM inclusion)
- `internal/builtins/json_decode.go` — Reference implementation for tree ADT parsing (Json ADT pattern)

## Problem Statement

AILANG has JSON parsing (`std/json`) but no XML support. Many enterprise document formats embed XML:
- **DOCX/PPTX/XLSX** (Office Open XML) — ZIP containers with XML payloads
- **SVG** — vector graphics
- **RSS/Atom** — feed formats
- **XHTML** — structured web content

Without XML parsing, AILANG cannot process any of these formats. Combined with `std/zip` (M-STDLIB-ZIP), XML support enables a compelling DocParse demo that showcases AILANG's pattern matching on tree-structured data.

### Motivating Example

```ailang
module demos/docparse

import std/zip (readEntry)
import std/xml (parseXml, findAll, getText, getAttr)

-- Extract slide titles from a PPTX file
func extractSlideTitles(path: string) -> [string] ! {FS} =
  let entries = _zip_listEntries(path)
  in match entries with
    | Ok(files) ->
      let slideFiles = _list_filter(\f. _str_startsWith(f, "ppt/slides/slide"), files)
      in _list_map(\f.
        match parseXml(readEntry(path, f)) with
          | Ok(root) ->
            let titles = findAll(root, "a:t")
            in _list_join(_list_map(\t. getText(t), titles), " ")
          | Err(_) -> ""
      , slideFiles)
    | Err(msg) -> []
```

## Goals

**Primary Goal:** Parse XML strings into a queryable tree ADT for pattern matching.

**Success Metrics:**
- Parse well-formed XML to an `XmlNode` ADT
- Query by tag name (findAll, findFirst)
- Extract text content and attributes
- Pattern match on XML structure using AILANG's `match` expressions
- Parse OOXML content from DOCX/PPTX files (real-world validation)

**Non-Goals (v0.7.3):**
- XML generation / serialization (future)
- XPath or CSS selectors
- XML namespaces as first-class concept (namespace prefixes treated as part of tag name)
- DTD/Schema validation
- Streaming/SAX-style parsing

## Solution Design

### ADT Design

The docparse-demo proposed:
```
type XmlNode = Element(string, [{name: string, value: string}], [XmlNode]) | Text(string)
```

This is a clean recursive ADT, but AILANG doesn't yet support user-defined ADTs in builtins directly. Instead, we use `TaggedValue` at the Go level (same pattern as `Json` ADT in `std/json`).

#### XmlNode ADT (registered in Go, usable from AILANG)

```ailang
-- Conceptual type (implemented as TaggedValue in Go)
type XmlNode =
  | Element(string, [{name: string, value: string}], [XmlNode])
  | Text(string)
  | CData(string)
  | Comment(string)
```

| Constructor | Fields | Description |
|-------------|--------|-------------|
| `Element(tag, attrs, children)` | tag: `string`, attrs: `[{name: string, value: string}]`, children: `[XmlNode]` | An XML element with tag name, attributes, and child nodes |
| `Text(content)` | content: `string` | Text content between tags |
| `CData(content)` | content: `string` | CDATA section (preserved verbatim) |
| `Comment(content)` | content: `string` | XML comment |

**Design decisions:**
- `CData` and `Comment` included for round-trip fidelity — callers can ignore via pattern matching
- Attributes as list of records (not a map) to preserve order — important for some XML formats
- Namespace prefixes included in tag name string (e.g., `"w:p"`, `"a:t"`) — simplest approach that works for OOXML

### Effect Classification

XML parsing is a **pure** operation — it transforms a string into a tree. No I/O, no effects.

| Function | Effect | Rationale |
|----------|--------|-----------|
| `parseXml(s)` | Pure | String → XmlNode tree transform |
| `findAll(node, tag)` | Pure | Tree traversal |
| `findFirst(node, tag)` | Pure | Tree traversal |
| `getText(node)` | Pure | Node content extraction |
| `getAttr(node, name)` | Pure | Attribute lookup |
| `getChildren(node)` | Pure | Child access |
| `getTag(node)` | Pure | Tag name access |

### API Design

#### std/xml Module

```ailang
module std/xml

-- Parse an XML string into an XmlNode tree
-- Returns: Result[XmlNode, string]
-- Example: parseXml("<root><item>hello</item></root>")
--   => Ok(Element("root", [], [Element("item", [], [Text("hello")])]))
export func parseXml(xml: string) -> Result[XmlNode, string]

-- Find ALL descendant elements matching a tag name (depth-first)
-- Example: findAll(root, "item") => [Element("item", ...), Element("item", ...)]
export func findAll(node: XmlNode, tagName: string) -> [XmlNode]

-- Find FIRST descendant element matching a tag name
-- Example: findFirst(root, "title") => Some(Element("title", ...))
export func findFirst(node: XmlNode, tagName: string) -> Option[XmlNode]

-- Get concatenated text content of a node and all descendants
-- Example: getText(Element("p", [], [Text("hello "), Element("b", [], [Text("world")])])) => "hello world"
export func getText(node: XmlNode) -> string

-- Get an attribute value by name
-- Example: getAttr(Element("div", [{name: "class", value: "main"}], []), "class") => Some("main")
export func getAttr(node: XmlNode, attrName: string) -> Option[string]

-- Get direct child nodes
-- Example: getChildren(Element("root", [], [Text("a"), Element("b", [], [])])) => [Text("a"), Element("b", [], [])]
export func getChildren(node: XmlNode) -> [XmlNode]

-- Get tag name of an Element node (empty string for non-Element nodes)
-- Example: getTag(Element("div", [], [])) => "div"
-- Example: getTag(Text("hello")) => ""
export func getTag(node: XmlNode) -> string
```

### Implementation

#### New Files

| File | LOC (est.) | Purpose |
|------|-----------|---------|
| `internal/builtins/xml.go` | ~350 | Builtin registration + XML→XmlNode conversion |
| `internal/builtins/xml_query.go` | ~200 | Query functions (findAll, findFirst, getText, etc.) |
| `internal/builtins/xml_test.go` | ~400 | Unit tests for parsing and queries |

#### Builtin Specifications

```go
// _xml_parse: string -> Result[XmlNode, string]
BuiltinSpec{
    Module: "std/xml", Name: "_xml_parse", NumArgs: 1,
    IsPure: true, Effect: "",
    Type: func() types.Type {
        T := types.NewBuilder()
        xmlNodeType := T.Con("XmlNode")
        return T.Func(T.String()).Returns(
            T.App("Result", xmlNodeType, T.String()),
        ).Build()
    },
}

// _xml_findAll: XmlNode -> string -> [XmlNode]
BuiltinSpec{
    Module: "std/xml", Name: "_xml_findAll", NumArgs: 2,
    IsPure: true, Effect: "",
}

// _xml_findFirst: XmlNode -> string -> Option[XmlNode]
BuiltinSpec{
    Module: "std/xml", Name: "_xml_findFirst", NumArgs: 2,
    IsPure: true, Effect: "",
}

// _xml_getText: XmlNode -> string
BuiltinSpec{
    Module: "std/xml", Name: "_xml_getText", NumArgs: 1,
    IsPure: true, Effect: "",
}

// _xml_getAttr: XmlNode -> string -> Option[string]
BuiltinSpec{
    Module: "std/xml", Name: "_xml_getAttr", NumArgs: 2,
    IsPure: true, Effect: "",
}

// _xml_getChildren: XmlNode -> [XmlNode]
BuiltinSpec{
    Module: "std/xml", Name: "_xml_getChildren", NumArgs: 1,
    IsPure: true, Effect: "",
}

// _xml_getTag: XmlNode -> string
BuiltinSpec{
    Module: "std/xml", Name: "_xml_getTag", NumArgs: 1,
    IsPure: true, Effect: "",
}
```

#### Go Implementation (sketch)

The parser uses Go's `encoding/xml` token-based decoder to build the XmlNode tree:

```go
func xmlParseImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    input := args[0].(*eval.StringValue).Value
    decoder := xml.NewDecoder(strings.NewReader(input))

    root, err := parseElement(decoder)
    if err != nil {
        return makeErr(fmt.Sprintf("XML parse error: %v", err)), nil
    }
    return makeOk(root), nil
}

func parseElement(d *xml.Decoder) (eval.Value, error) {
    var children []eval.Value

    for {
        tok, err := d.Token()
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, err
        }

        switch t := tok.(type) {
        case xml.StartElement:
            // Build attributes list
            attrs := make([]eval.Value, len(t.Attr))
            for i, a := range t.Attr {
                name := a.Name.Local
                if a.Name.Space != "" {
                    name = a.Name.Space + ":" + a.Name.Local
                }
                attrs[i] = &eval.RecordValue{
                    Fields: map[string]eval.Value{
                        "name":  &eval.StringValue{Value: name},
                        "value": &eval.StringValue{Value: a.Value},
                    },
                }
            }

            // Recursively parse children
            childNodes, err := parseChildren(d)
            if err != nil {
                return nil, err
            }

            tagName := t.Name.Local
            if t.Name.Space != "" {
                tagName = t.Name.Space + ":" + t.Name.Local
            }

            children = append(children, &eval.TaggedValue{
                ModulePath: "std/xml",
                TypeName:   "XmlNode",
                CtorName:   "Element",
                Fields: []eval.Value{
                    &eval.StringValue{Value: tagName},
                    &eval.ListValue{Elements: attrs},
                    &eval.ListValue{Elements: childNodes},
                },
            })

        case xml.CharData:
            text := strings.TrimSpace(string(t))
            if text != "" {
                children = append(children, &eval.TaggedValue{
                    ModulePath: "std/xml",
                    TypeName:   "XmlNode",
                    CtorName:   "Text",
                    Fields:     []eval.Value{&eval.StringValue{Value: text}},
                })
            }

        case xml.Comment:
            children = append(children, &eval.TaggedValue{
                ModulePath: "std/xml",
                TypeName:   "XmlNode",
                CtorName:   "Comment",
                Fields:     []eval.Value{&eval.StringValue{Value: string(t)}},
            })

        case xml.EndElement:
            break // Return to parent
        }
    }

    // If single root element, return it directly; otherwise wrap
    if len(children) == 1 {
        return children[0], nil
    }
    // Multiple top-level nodes: wrap in synthetic root
    return &eval.TaggedValue{
        ModulePath: "std/xml",
        TypeName:   "XmlNode",
        CtorName:   "Element",
        Fields: []eval.Value{
            &eval.StringValue{Value: ""},
            &eval.ListValue{Elements: []eval.Value{}},
            &eval.ListValue{Elements: children},
        },
    }, nil
}
```

Query functions operate on `TaggedValue` by matching constructor names:

```go
func xmlFindAllImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    node := args[0]
    tagName := args[1].(*eval.StringValue).Value

    var results []eval.Value
    findAllRecursive(node, tagName, &results)
    return &eval.ListValue{Elements: results}, nil
}

func findAllRecursive(node eval.Value, tagName string, results *[]eval.Value) {
    tv, ok := node.(*eval.TaggedValue)
    if !ok || tv.CtorName != "Element" {
        return
    }
    // Check if this element matches
    if tv.Fields[0].(*eval.StringValue).Value == tagName {
        *results = append(*results, node)
    }
    // Recurse into children
    children := tv.Fields[2].(*eval.ListValue).Elements
    for _, child := range children {
        findAllRecursive(child, tagName, results)
    }
}

func xmlGetTextImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    var buf strings.Builder
    collectText(args[0], &buf)
    return &eval.StringValue{Value: buf.String()}, nil
}

func collectText(node eval.Value, buf *strings.Builder) {
    tv, ok := node.(*eval.TaggedValue)
    if !ok {
        return
    }
    switch tv.CtorName {
    case "Text", "CData":
        buf.WriteString(tv.Fields[0].(*eval.StringValue).Value)
    case "Element":
        children := tv.Fields[2].(*eval.ListValue).Elements
        for _, child := range children {
            collectText(child, buf)
        }
    }
}
```

### Namespace Handling

XML namespaces are complex. For v0.7.3, we use the **prefix-in-name** approach:

```xml
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Hello</w:t></w:r></w:p>
  </w:body>
</w:document>
```

Go's `encoding/xml` resolves prefixes to full URIs. We reverse this for OOXML usability:

| Go `xml.Name` | AILANG tag name | Why |
|----------------|-----------------|-----|
| `{http://...}document` | `w:document` (from prefix) | OOXML users think in prefixes |

**Implementation note:** Go's decoder resolves to URIs. We need to track prefix→URI mappings from `xmlns:` attributes and reverse-map. For v0.7.3, we handle this by:
1. Extracting `xmlns:*` attributes during parsing
2. Building a prefix map per element
3. Using the prefix form in tag names

If prefix resolution proves too complex, we fall back to using the raw local name only (dropping namespace entirely), which still works for most OOXML queries since tag names like `document`, `body`, `p`, `t` are unique within their context.

### Security Considerations

1. **Entity expansion attacks (Billion Laughs)**: Go's `encoding/xml` does NOT expand external entities by default — safe out of the box
2. **Depth limit**: Maximum 256 nesting levels to prevent stack overflow on pathological input
3. **Size limit**: Maximum 50MB input string (configurable via environment variable)
4. **No external references**: DTD and external entity references are ignored

### Testing Strategy

1. **Basic parsing**: Simple XML → XmlNode tree verification
2. **Attributes**: Elements with multiple attributes, namespace prefixes
3. **Query functions**: findAll, findFirst on nested structures
4. **getText**: Recursive text extraction across mixed content
5. **Error paths**: Malformed XML, empty input, huge input
6. **OOXML fragments**: Real DOCX `word/document.xml` excerpts
7. **Pattern matching**: AILANG `match` on XmlNode constructors (integration test)

#### Test Cases

```go
// Basic element
"<root/>" => Element("root", [], [])

// Nested with text
"<p>hello <b>world</b></p>" => Element("p", [], [Text("hello"), Element("b", [], [Text("world")])])

// Attributes
`<div class="main" id="1"/>` => Element("div", [{name:"class",value:"main"},{name:"id",value:"1"}], [])

// OOXML fragment
`<w:p><w:r><w:t>Hello</w:t></w:r></w:p>` => Element("w:p", [], [Element("w:r", [], [Element("w:t", [], [Text("Hello")])])])
```

## Milestones

| # | Task | Est. |
|---|------|------|
| 1 | Register builtin specs for all 7 functions | 2h |
| 2 | Implement `_xml_parse` with Go `encoding/xml` decoder | 4h |
| 3 | Implement query functions (findAll, findFirst, getText, getAttr, getChildren, getTag) | 3h |
| 4 | Namespace prefix handling for OOXML compatibility | 2h |
| 5 | Security hardening (depth limit, size limit) | 1h |
| 6 | Tests for parsing, queries, OOXML fragments | 4h |
| 7 | Example file + documentation | 1h |

**Total: ~17 hours (2-3 days)**

## Alternatives Considered

### A: Return JSON-like untyped tree (reuse Json ADT)
Rejected. XML has attributes, namespaces, mixed content, and processing instructions that don't map cleanly to JSON. A dedicated `XmlNode` ADT provides type-safe pattern matching on XML-specific constructs.

### B: DOM-style API with methods
Rejected. AILANG is functional — free functions operating on an ADT is the idiomatic pattern (same as `std/json` query functions). Methods would require object orientation.

### C: Full namespace support with URI-based lookup
Deferred. Full namespace resolution adds significant complexity. Prefix-in-name is sufficient for OOXML (the primary use case) and can be extended later without breaking changes.

### D: SAX/streaming parser
Deferred. Tree-based parsing is simpler, more composable, and better for pattern matching. Streaming makes sense for multi-GB XML files, which is not our use case (document parsing = kilobytes to megabytes).

## Future Extensions (post v0.7.3)

- `serializeXml(node: XmlNode) -> string` — XML generation
- `querySelector(node, selector)` — CSS-like selectors for convenience
- Proper namespace-aware queries (`findAllNS(node, uri, localName)`)
- XML Schema validation
- XSLT-like transformations via AILANG functions
