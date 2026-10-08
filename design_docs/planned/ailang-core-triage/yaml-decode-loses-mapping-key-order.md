# std/yaml decode loses mapping key order; reserved words with/recv; no yaml encode

- **Date**: 2026-10-07
- **Class**: bug (primary) + feature ×2 (secondary items in the same report)
- **Recommend**: design-doc
- **Searched**: `yamlToJson`, `std/yaml`, `yaml encode`, `reserved keyword`, `reserved word`, `with`, `recv` across design_docs/ and docs/
- **Estimate**: ~100–200 lines in `internal/builtins/yaml.go` (+ `yaml_test.go`); far past DIRECT_FIX_MAX_LINES, so design-doc regardless of the semantics ruling

## Primary: mapping key order is destroyed in `_yaml_to_json`

Verified the reported mechanism exactly. `yamlToJSONImpl` (`internal/builtins/yaml.go`) does
`yaml.Unmarshal(..., &v)` into `interface{}` — yaml.v3 decodes string-keyed mappings into
`map[string]interface{}`, exactly as the original design doc records
(`design_docs/implemented/v0_30_0/m-std-yaml.md`, "decodes string-keyed mappings into
map[string]interface{}") — then `json.Marshal(v)`, which serializes maps in sorted key
order. So `yamlDecode("b: 1\na: 2\nc: 3\n")` yields keys `a,b,c`. Downstream this is lossy
in a way AILANG itself could otherwise absorb: `std/json.decode` builds
`JObject(List[{key, value}])` (`internal/eval/builtins_json.go`, JObject construction), a
list of pairs, so order **would** flow through `keys(j)` untouched if the bridge preserved
it — the loss is entirely in the Go bridge.

Why design-doc and not direct-fix:

1. **Row 3 — two acceptable ways.** The reporter names both: (a) preserve document order by
   switching to the yaml.v3 `Node` API and emitting JSON manually (bytes.Buffer with
   per-scalar `json.Marshal` to keep escaping/float rules), or (b) keep the map path and
   document that order is not preserved. These have different consequences for callers, and
   the choice is exactly the kind a reviewer could disagree with.
2. **Row 4 — changes a public surface.** `_yaml_to_json`'s documented output is an
   "equivalent JSON string" (`BuiltinMetadata` in the same file); switching to ordered
   emission changes byte-for-byte output for every mapping, including existing examples in
   the doc comment (`{"a":1,"b":["x","y"]}` ordering) and any test/consumer pinning it.
3. **Row 6 — size.** The Node-API path is a real walker, not a two-liner: mapping/sequence/
   scalar nodes, anchor/alias resolution (`node.Alias`), tag handling, non-string-key Err
   behavior, NaN/Inf Err behavior, first-document-only semantics — all currently documented
   guarantees that the rewrite must keep. Estimate ~100–200 lines including tests.

Search for prior coverage: none found. `design_docs/implemented/v0_30_0/m-std-yaml.md`
describes the map-based bridge (line 102 area) but never rules on key order;
`docs/docs/reference/std-yaml.md` has no "order" statement either way — so today the
contract is undocumented, which is itself part of the defect.

## Secondary 1: `with` and `recv` are reserved with no parser use; error cascade buries the cause

Verified in `internal/lexer/token.go`: both are in the keyword map (`"with": WITH` line ~275,
`"recv": RECV` line ~295). `WITH` has exactly one parser use — detecting the ML-style
`match x with` trap (`internal/parser/parser_expr.go`, `parseMatchExpression`, PAR019) — and
`RECV` has zero parser uses (it sits in the reserved session-type cluster SEND/RECV/TIMEOUT/
AS/DERIVING). So two natural parameter/local names (`with`, `recv`) are unusable as
identifiers for features that mostly don't exist, and the resulting
`expected identifier, got reserved keyword` error (via `internal/parser/parser_error.go`)
cascades into unrelated diagnostics. Related but not covered: `design_docs/planned/v1_0_0/
m-dialect-keyword-diagnostics.md` rules narrowly on the `case` spelling and explicitly
defers keyword-scope decisions behind an evidence trigger — it does not cover unreserving
or contextualizing `with`/`recv`. Class: bug (DX). Recommend: **design-doc** — row 4
(grammar/keyword surface) and row 3 (unreserve vs contextual keyword vs better diagnostic).

## Secondary 2: no std/yaml encode

`std/yaml` (`std/yaml.ail`) exposes only `yamlToJson`/`decode`; writing YAML back means
hand-emitting JSON-quoted scalars. Class: feature. Recommend: **design-doc** — new builtin
plus stdlib surface (row 4), and there are real design choices: accept the `Json` ADT vs
arbitrary values, scalar quoting/block-style policy, round-trip relationship to decode.

## Dispatch note

One report, three independently actionable items. The three recommendations above are all
design-doc; the sprint planner should split them into separate work items rather than one
combined doc — they touch different subsystems (`internal/builtins/yaml.go`, parser keyword
policy, stdlib surface) and have different urgency.