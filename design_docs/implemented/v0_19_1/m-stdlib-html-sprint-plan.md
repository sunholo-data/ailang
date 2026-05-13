# Sprint Plan: M-STDLIB-HTML — `std/html` HTML5 Parser Module

## Summary
Ship `std/html` as a thin Go wrapper around `golang.org/x/net/html` that returns the existing `std/xml` `XmlNode` ADT — eliminating ~300 LOC of HTML5-tolerance shims in downstream projects (ailang-parse).

**Duration:** 2 days (~8 working hours)
**Dependencies:** v0.19.0 cross-module ADT import fix (shipped, msg 4ada0a1b)
**Risk Level:** Low — wraps an existing Go stdlib pattern; deps already indirect; no parser/typechecker changes.

## Current Status Analysis

### Completed Recently (informs velocity)
- ✅ M-WASM-AI-STEP-BYO-KEY M1+M2+M3+M4 — wired ai.step variants to JS handlers + docs (4abe9d38, 27854881, 7f4353df) — landed in ~1 day each.
- ✅ M-STDLIB-XML / M-STDLIB-XML-IMPROVEMENTS (v0.7.3, v0.9.4) — the direct template for this sprint. xml.go is 763 LOC and was built in a comparable 2-day window.
- ✅ M-BYTECODE-XML-BUILTINS (v0.11.0) — confirms the bytecode VM picks up new builtins from `init()` registration; nothing extra to wire.

### Velocity
- Recent stdlib additions: ~300–400 LOC of Go + ~30–80 LOC of `.ail` + tests, delivered in 1–2 days.
- Estimated capacity: ~600 LOC implementation + tests over 2 days.

### Remaining from Design Doc
- ⏳ M1: Builtins (`internal/builtins/html.go` + go.mod promote) — ~300 LOC Go.
- ⏳ M2: Module + Example (`std/html.ail` + `examples/runnable/html_parser.ail`) — ~70 LOC AILANG.
- ⏳ M3: Tests (`internal/builtins/html_test.go` + fixture) — ~200 LOC Go.
- ⏳ M4: Docs + announce (`docs/docs/reference/stdlib/std-html.md`, CHANGELOG, inbox reply) — ~80 LOC docs.

## Proposed Milestones

### Milestone 1: Builtins (M1)
**Goal:** `_html_parse` and `_html_parseFragment` builtins return `Result[XmlNode, string]` from real HTML input.
**Estimated:** ~300 LOC Go implementation
**Duration:** Day 1 morning (~4h)

**Tasks:**
- Pre-flight: smoke test that `import std/xml (XmlNode)` works from a throwaway module under v0.19.0.
- `go get golang.org/x/net@latest && go mod tidy` — promote indirect → direct.
- Create `internal/builtins/html.go`:
  - `init()` registering `registerHtmlParse()` + `registerHtmlParseFragment()`.
  - `htmlNodeToXmlNode(*html.Node, depth int) (eval.Value, error)` converter.
  - Drop `DoctypeNode`; unwrap `DocumentNode` to its first `ElementNode` child.
  - Reuse `makeXmlElement` / `makeXmlText` / `makeXmlComment` / `makeXmlAttr` from xml.go (same package — directly callable).
  - Enforce 50MB size + 256 depth caps; return `Err` strings on overflow.
- Build + `make test` to ensure no regressions in xml.go.

**Acceptance Criteria:**
- [ ] `make build` succeeds with new builtins registered.
- [ ] `go.mod` lists `golang.org/x/net` as direct dep, `go mod tidy` clean.
- [ ] Existing `internal/builtins/xml_test.go` still passes (no shared-helper drift).
- [ ] Manual REPL smoke: parse `"<p>hi</p>"` returns `Ok(Element("html", ..., [...Element("p", ..., [Text("hi")])...]))`.

**Risks:**
- `import std/xml (XmlNode)` from another stdlib module hits an edge case — *Mitigation: pre-flight test before writing Go code; if it breaks, fall back to duplicating ADT constructors in std/html.ail (still single source of truth at the Go layer).*
- net/html synthesizes `<html><head></head><body>…</body></html>` even for fragments. Document this in the example; surface fragment-mode via `parseFragment`.

### Milestone 2: Module surface + Example (M2)
**Goal:** AILANG-level `std/html` module with two `pure func`s, plus a runnable example.
**Estimated:** ~70 LOC (.ail) + verification
**Duration:** Day 1 afternoon (~2h)

**Tasks:**
- Create `std/html.ail`:
  - `module std/html`
  - `import std/result (Result)`, `import std/xml (XmlNode)`
  - `export pure func parse(s: string) -> Result[XmlNode, string] = _html_parse(s)`
  - `export pure func parseFragment(s: string) -> Result[[XmlNode], string] = _html_parseFragment(s)`
- Confirm `std/embed.go`'s `*.ail` glob auto-includes html.ail (no edit).
- Create `examples/runnable/html_parser.ail` mirroring `examples/runnable/xml_parser.ail` — parse + `findAll("p")` + `getText` on a real-world snippet (unclosed tags, boolean attrs).
- `make verify-examples` clean.

**Acceptance Criteria:**
- [ ] `ailang run examples/runnable/html_parser.ail` prints expected paragraph text.
- [ ] `ailang prompt` / `ailang docs search` surfaces `std/html` (auto via stdlib index).
- [ ] `make verify-examples` passes.
- [ ] No new linter warnings.

**Risks:**
- Cross-module ADT import quirk — see M1 mitigation.

### Milestone 3: Tests (M3)
**Goal:** Golden tests covering the messy real-world HTML cases the design doc enumerates.
**Estimated:** ~200 LOC Go test code + fixture
**Duration:** Day 2 morning (~3h)

**Tasks:**
- `internal/builtins/html_test.go`:
  - `TestHtml_UnclosedParagraphs` — `<p>a<p>b` → two siblings.
  - `TestHtml_BooleanAttribute` — `<input disabled>` → attr value is `""`.
  - `TestHtml_LowerCasedTag` — `<DIV>` → `"div"`.
  - `TestHtml_CommentPreserved` — `<!-- x -->` → `Comment("x")`.
  - `TestHtml_ScriptPassthrough` — `<script>alert(1)</script>` content as Text child.
  - `TestHtml_WordNamespace` — `<o:p>` keeps prefix in tag name.
  - `TestHtml_OversizedInput` — 50MB+1 input → `Err`.
  - `TestHtml_EmptyInput` — empty string → `Ok(Element("html", ...))`.
  - `TestHtmlParseFragment_MultipleRoots` — `<p>a</p><p>b</p>` → list of 2.
  - `TestHtmlParse_Fixture_SunholoHomepage` — fixture file, depth ≤ 256, ≥1 `<article>`/`<main>`/`<section>`.
- Copy `sunholo_homepage.html` from ailang-parse repo into `internal/builtins/testdata/`.
- `make test` clean.

**Acceptance Criteria:**
- [ ] All 10 unit tests pass.
- [ ] `make test` reports no regressions elsewhere.
- [ ] `make lint` clean.
- [ ] Coverage on `internal/builtins/html.go` ≥ 80%.

**Risks:**
- Fixture file is large — keep under 200KB; check it in only if necessary, else synthesize a smaller equivalent.

### Milestone 4: Docs + Announce (M4)
**Goal:** Discoverable from docs site, CHANGELOG entry, reply to ailang-parse inbox.
**Estimated:** ~80 LOC markdown + reply
**Duration:** Day 2 afternoon (~1h)

**Tasks:**
- `docs/docs/reference/stdlib/std-html.md` — parallel structure to `std-xml.md`, two functions documented with examples.
- Add row to `docs/docs/reference/stdlib/index.md`.
- `CHANGELOG.md`: v0.20.0 entry under "Added" — `std/html` HTML5 parser, references inbox msg 62a9b106 and ailang-parse v0.13.0 follow-up.
- `ailang messages send ailang-core "Shipped: std/html in v0.20.0" --title "RE: std/html stdlib module"`
- Move design doc from `design_docs/planned/v0_20_0/` → `design_docs/implemented/v0_20_0/` after release.

**Acceptance Criteria:**
- [ ] `make docs` builds clean (if applicable).
- [ ] CHANGELOG entry present and credits source.
- [ ] Inbox reply sent.
- [ ] `ailang docs search "html"` surfaces the new module.

**Risks:** None.

## Success Metrics
- Test coverage on `internal/builtins/html.go`: ≥ 80%.
- Examples passing: `examples/runnable/html_parser.ail` works.
- Documentation: `std-html.md` shipped, stdlib index updated.
- All tests passing: ✅ (`make test`)
- All linting passing: ✅ (`make lint`)
- ailang-parse integration sanity: parse their canonical fixture and confirm ≥10 non-error blocks.

## Dependencies
- v0.19.0 cross-module ADT import fix — **shipped**.
- `golang.org/x/net` — already in `go.mod` (indirect v0.53.0).

## Open Questions
- None blocking. Design freeze locked the three high-impact choices (XmlNode reuse, drop Doctype/Document, pass <script> through).

## Notes
- This sprint is a textbook "Go-wraps-spec-compliant-library, expose via stdlib" pattern. The risk floor is low because `internal/builtins/xml.go` is the exact template.
- Final commit should reference inbox msg `msg_20260513_132546_62a9b106` so the audit trail is intact.
- No GitHub issue exists for this — the source is an internal inbox message from sunholo/ailang-parse.
