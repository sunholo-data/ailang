# M-STDLIB-XML-LENIENT — Sprint Plan

**Design doc**: [m-stdlib-xml-lenient.md](m-stdlib-xml-lenient.md)
**Risk**: Low (additive, pure, no change to existing `parse`)
**Duration**: ~1 day (single milestone)
**Total LOC estimate**: ~180 (impl ~80 + tests ~80 + stdlib/example ~20)
**Inbox refs**: `msg_20260602_203603_276a4d86` (+ dup `c60a9501`)

## Goal

Let docparse recover real-world XML that carries a bare/unescaped `&`, **without** weakening
the strict `parse` path. Ship `_xml_sanitize` Go builtin + `sanitizeXml` / `parseLenient`
in `std/xml`.

## Single Milestone: M1 — Lenient XML sanitize + parse

### Tasks (TDD order)

1. **Tests first** (`internal/builtins/xml_test.go`, ~80 LOC):
   - bare `&` → escaped; text round-trips through parse
   - no double-escape of `&amp;`, `&#123;`, `&#xAB;`, `&lt;`, `&name;`
   - trailing `&` at EOF
   - `&` inside attribute values
   - idempotence: `sanitize(sanitize(x)) == sanitize(x)`
   - reporter's exact repro `<r><p>Apex Consulting & Partners</p></r>` via `parseLenient`
   - strict `parse` unchanged (existing tests untouched)

2. **Go builtin** `_xml_sanitize` (new `internal/builtins/xml_sanitize.go`, ~80 LOC):
   manual single-pass scanner. On each `&`, peek for `#[0-9]+;` / `#x[0-9a-fA-F]+;` /
   `[A-Za-z][A-Za-z0-9]*;`; if none, emit `&amp;`, else copy through. Register as pure
   builtin (mirror `_escapeXml` in `xml_serialize.go`).

3. **stdlib** ([std/xml.ail](../../std/xml.ail), ~10 LOC):
   ```ailang
   export pure func sanitizeXml(xml: string) -> string = _xml_sanitize(xml)
   export pure func parseLenient(xml: string) -> Result[XmlNode, string] = parse(sanitizeXml(xml))
   ```

4. **Example** `examples/runnable/xml_lenient.ail` (~10 LOC): `parseLenient` on dirty input,
   print recovered text. Verify with `make verify-examples`.

5. **Docs**: CHANGELOG entry; update `std/xml` usage header doc comment.

### Acceptance Criteria

- [ ] `parseLenient("<r><p>Apex Consulting & Partners</p></r>")` → `Ok`, text `Apex Consulting & Partners`
- [ ] `sanitizeXml` does not double-escape valid entities; is idempotent
- [ ] strict `parse` behavior unchanged — full `make test` green
- [ ] `examples/runnable/xml_lenient.ail` runs clean via `make verify-examples`
- [ ] `make lint` clean

## Commits

- Dev commits: `refs` the inbox-derived issue if one exists.
- Single milestone → one focused commit set on `dev`.
