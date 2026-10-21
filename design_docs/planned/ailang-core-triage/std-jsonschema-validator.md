# std/jsonschema — validate a Json against a JSON Schema (Draft 2020-12)

- **Date**: 2026-10-19
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `json schema` / `jsonschema` over `design_docs/` (only archive hits: M-TOOLING-DETERMINISTIC describes eval *tooling* schemas validated by gojsonschema in Go, not an AILANG stdlib module; benchmark analysis is LLM tool-schema fields); `jsonschema` over `design_docs/planned/` (only serveapi protocol-closure docs naming `google/jsonschema-go` as an external module root); `ls std/` (no `jsonschema.ail`); `go.mod` (see Why). Terms for the gap claim: `jsonschema`, `json schema`, `schema validat`.

The reporter hit this in production: the Holosun Decision Budget ships `decision-envelope.json` beside its schema and nothing validated one against the other — a Python `jsonschema` check found 4 failures in the real client envelope and 6 in the exemplar, while AILANG's own checks enforce business rules only. The stopgap is ~150 lines of pure AILANG (`sunholo-data/clients`, `_holosun/decision-budget/schema/envelope/schemacheck.ail`) covering the keyword subset the schema actually uses, and the request is to upstream it as `std/jsonschema validate(schema: Json, doc: Json) -> [ValidationError]` with `{ path, keyword, message }` errors, Draft 2020-12 core + validation vocabulary, local `$ref`, fail-closed on unknown keywords (strict mode), and Go backing "like std/json decode" for speed and correctness. This is squarely design-doc territory on three counts: (1) there is more than one acceptable implementation — pure-AILANG port of the stopgap vs a Go builtin (`internal/builtins/jsonschema.go` + `_jsonschema_validate` in the manner of `_json_decode` in `internal/eval/builtins_json.go`) vs wrapping `google/jsonschema-go` (already a direct dependency at `go.mod:20` for serveapi/MCP, but wiring it into the *core* builtin set would pull it into the interpreter's dependency closure, which `scripts/check_protocol_closure.sh`-style gates police — so which backend, and whether the dep moves roots, is a real decision); (2) it is a new public stdlib surface whose contract (error shape, strict-mode default, RE2-vs-ECMA-262 pattern semantics) should be pinned before code exists — the reporter explicitly leaves ECMA-262-vs-RE2 as "or document the differences"; (3) it spans several files: `std/jsonschema.ail`, a Go builtin (if Go-backed), tests, and `docs/docs/reference/stdlib.md`. The 20-keyword frequency table (`type` 193 … `maximum` 1) is excellent scoping input for that doc; `if/then` (12 uses) and `dependentRequired` are the non-trivial ones. Note the strict-mode default is the right call and should be stated as a requirement in the doc, not rediscovered. **Recommend: design-doc** (via `design-doc-creator`), with the stopgap in the clients repo as the reference implementation to port.

---

## Secondary item in the same report: `std/list` lacks `all`

- **Class**: feature
- **Recommend**: direct-fix
- **Estimate**: ~3 lines in `std/list.ail`

Verified: `std/list.ail:156` exports `any` (backed by builtin `_list_any`) and there is no `all` anywhere in `std/`. This does **not** need a new Go builtin: `all(p, xs) = !any(\x -> not(p x), xs)` is pure, O(n), and short-circuits identically (any stops at the first `!p` true, i.e. the first element failing `p`). One file, ~3 lines including the doc comment — passes the direct-fix gate. It can ride the same PR as the jsonschema work or land independently.

### Correction to the report

The report says "IMP010 suggests any" — that's a misreading. `IMP010` is the error code for *unknown import form* (`internal/errors/codes.go:310`). The reason the function is named `any` rather than `exists` is that `exists` is a reserved keyword (`std/list.ail:155` comment). Worth fixing before it propagates into the design doc's motivation section.
