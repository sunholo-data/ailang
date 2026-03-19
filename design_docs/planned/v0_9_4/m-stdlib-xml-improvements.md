# M-STDLIB-XML: XmlNode Constructors, writeEntryBytes, escapeXml

## Status: Implemented
## Version: v0.9.4
## Priority: Nice-to-have (P3)
## Effort: Medium (4-6 hours)

## Problem

DocParse agent builds XML/DOCX/PPTX documents and needs:
1. **escapeXml()** — Escape strings for XML text content (easy win)
2. **XmlNode constructors** — Build XML programmatically instead of string concatenation
3. **writeEntryBytes** — Write binary ZIP entries (needed for images in DOCX/PPTX)

## Design

### Phase 1: escapeXml (Easy Win)

**Builtin**: `_escapeXml`
- **Module**: `std/xml`
- **Signature**: `pure func escapeXml(text: string) -> string`
- **Behavior**: Escapes `<`, `>`, `&`, `"`, `'` to XML entity references
- **Pure**: Yes
- **Implementation**: Use Go's `html.EscapeString()` (already used in xml_serialize.go)

```ailang
-- In std/xml.ail
export pure func escapeXml(text: string) -> string = _escapeXml(text)
```

### Phase 2: XmlNode Constructors (Future)

Factory functions for building XML nodes:
- `xmlElement(tag: string, attrs: Map[string, string], children: List[XmlNode]) -> XmlNode`
- `xmlText(content: string) -> XmlNode`

These would create TaggedValues of the existing XmlNode ADT.

### Phase 3: writeEntryBytes (Future)

- `writeEntryBytes(writer: ZipWriter, path: string, data: bytes) -> unit ! {IO}`
- Needed for embedding images in DOCX/PPTX ZIP archives
- Requires bytes type support in ZIP builtins

## Testing

- Unit test for escapeXml with edge cases (<, >, &, quotes, already-escaped text)
- Integration test with XML serialization
- Example file: `examples/xml_escape.ail`

## Origin

Agent message ee0b27d2 from docparse (2026-03-19)
