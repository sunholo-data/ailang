# Sprint Plan: M-STD-YAML-MAPPING-KEY-ORDER

## Summary

Preserve YAML mapping document order through `yamlToJson` and `decode`, using the approved yaml.v3 Node walker design. Preserve scalar bytes, alias resolution, merge precedence, typed failures, purity, and first-document-only behavior.

**Design:** [Approved design](m-std-yaml-mapping-key-order.md)
**Target:** v0.53.0
**Duration:** 1 engineering day (7 hours work + 1 hour contingency).
**Estimated total:** 200 changed LOC: 110 implementation, 70 tests, 20 documentation/comments.
**Risk:** Medium; merge expansion and alias recursion deserve more attention than the basic order change.
**Authorization:** Design approved in task-5801b699 handoff. This artifact plans implementation; coordinator plan approval/merge triggers sprint-executor. No implementation occurs in this planning stage.

## Current Status and Velocity

The current builtin decodes into `interface{}` and marshals Go maps, losing source order. `_json_decode` already preserves object token order; no Json ADT or public signature change is needed. Existing YAML tests cover scalars, basic aliases, first document, malformed input, non-string keys and non-finite floats. The design's systemic audit locates the order loss solely in this bridge, including nested mappings.

The approved design was recovered verbatim from origin/coordinator/task-5801b699, commit ef055a9a3655eb241b3d172f9d22f478b319024b.

Velocity analysis script was run for 7 days. This shallow checkout exposes one recent triage commit and no usable diff statistics; the script found no recent LOC metrics. A measured LOC/day rate cannot be established. The 200 LOC/day capacity is a planning assumption based on the approved design's 150–200 LOC and 7-hour breakdown, with one extra hour reserved. Historical changelog numbers are not treated as recent measurements.

## Registry Reuse Audit

`ailang pkg search yaml` returned no packages (binary emitted a stale-build warning). No candidates existed to inspect with pkg info/docs. This changes an existing compiler builtin using the already pinned `gopkg.in/yaml.v3 v3.0.1`, rather than adding a package capability; an external AILANG package cannot replace the internal bridge.

| Milestone | Decision | Rationale |
|---|---|---|
| M1 | none | Repair existing Go builtin; reuse existing yaml.v3 and encoding/json dependencies. |
| M2 | none | Regression coverage for the same builtin and existing stdlib integration. |
| M3 | none | Update the existing contract and examples; no package implementation. |

## Milestones

### M1: Node walker and order regression (~130 LOC)

**Estimate:** 110 implementation + 20 tests; 3 hours, Day 1 morning.
**Dependencies:** none.
**Files:** internal/builtins/yaml.go, internal/builtins/yaml_test.go.
**Example coverage:** existing examples/runnable/yaml_config.ail is retained for M3 smoke validation; M2 extends tests/yaml_bridge_test.ail with an observable order example.

Implement compiled probes for first-document-only Node parsing, alias pointers, merge tags, scalar tags and timestamps. Replace the generic-map conversion with ordered mapping/sequence emission. Delegate scalar Decode to yaml.v3 and scalar/key escaping to json.Marshal. Collect explicit keys before merge insertion; insert merged keys at the merge position, omit overridden keys, and retain earlier-source-wins precedence. Detect duplicate and non-string keys with yaml-prefixed Err results.

Before trusting recursion, compare the existing decoder's behavior for cyclic aliases and invalid merge structures; keep failures typed and avoid unbounded recursion. This is compatibility hardening within the approved alias-resolution guarantee, not a new public feature. If this exceeds the buffer, report revised effort rather than weakening guarantees.

**Acceptance criteria:**
- [ ] b,a,c order is pinned in block and flow mappings, nested objects, and mappings within sequences.
- [ ] Compiled tests pin Node first-document behavior, alias pointers, merge tags, explicit tags and timestamps.
- [ ] Explicit keys override merged keys irrespective of position; earlier merge sources win; nested merges preserve source order.
- [ ] Empty input emits null; empty mappings/sequences emit {} and []; public signature and purity remain unchanged.
- [ ] Focused Go YAML tests pass; recursive alias failures cannot panic or recurse without bound.

**Risk:** Direct Node traversal bypasses yaml.v3 map-decoder validation. Mitigate with explicit checks and M2 regression matrix.

### M2: Guarantee matrix and AILANG integration (~50 LOC)

**Estimate:** 50 test LOC; 3 hours, Day 1 afternoon.
**Dependencies:** M1.
**Files:** internal/builtins/yaml_test.go, tests/yaml_bridge_test.ail.
**Examples:** extend the existing bridge test with b,a,c keys and equivalent std/json decode comparison. Obtain `ailang prompt` before editing .ail and use supported syntax.

Audit every row of the approved guarantees table. Add missing cases for mapping/sequence aliases, merge ordering and invalid values, duplicates, escaped keys/HTML, tags, timestamps, block strings, large integer JSON digits, hex/octal and sexagesimal strings. Preserve numeric/bool/null/compound-key errors, NaN and both infinities, malformed YAML, and first-document-only behavior. Include quoted string keys and repeated merge-key validation where supported by the old decoder.

Re-pin TestYAMLToJSON_BlockMapping and bridge Test 1 to document order. Keep Tests 2–7 behavior intact. Retain the 100-pass determinism assertion with an updated comment. Audit all sorted-output expectations and examples; bytes change only where mapping order differs from sorted order.

**Acceptance criteria:**
- [ ] Every design guarantee row has a test; preserved scalar JSON bytes and yaml-prefixed Err results remain covered.
- [ ] Bridge Test 1 expects name,count,items; other existing assertions remain passing.
- [ ] std/yaml.decode keys match equivalent std/json.decode keys in b,a,c order.
- [ ] Alias mapping order, merge precedence, invalid merges and duplicate rejection are explicitly tested.
- [ ] Focused Go tests and the rebuilt-binary AILANG bridge integration pass.

**Risk:** Current tests pin too little of the decoder's edge behavior. Compare old and new results where the design is silent, and document material discrepancies before completing the milestone.

### M3: Contract, examples and final validation (~20 LOC)

**Estimate:** 20 documentation/comment LOC; 1 hour, Day 1 late afternoon, followed by 1 hour contingency.
**Dependencies:** M2.
**Files:** internal/builtins/yaml.go metadata/comments, std/yaml.ail comments, docs/docs/reference/std-yaml.md, design_docs/implemented/v0_30_0/m-std-yaml.md, changelogs/v0.32-current.md.
**Examples:** verify existing examples/runnable/yaml_config.ail; the M2 bridge case is the executable order example. No additional standalone file is required.

State document order and merge insertion/override rules in the reference, stdlib comments and builtin metadata. Amend the old design's incidental sorted-order assertions as historical behavior superseded by this design. Audit doc-comment examples and exact JSON strings; replace sorted expectations where needed. Add a current changelog entry flagging changed mapping bytes while preserving signatures and error conditions.

**Acceptance criteria:**
- [ ] Reference, stdlib comments and builtin metadata state order and merge semantics without implying sorted output.
- [ ] Historical design and pinned doc examples accurately distinguish the old behavior from the new contract.
- [ ] Changelog explicitly flags byte changes for mappings whose source order differs from sorted order.
- [ ] make build, make test, make fmt and make lint pass; focused Go tests use `go test ./internal/builtins -run YAML -count=1`.
- [ ] Rebuilt CLI passes `ailang check std/yaml.ail`, `ailang run --caps IO --entry main tests/yaml_bridge_test.ail`, and `ailang run --caps IO --entry main examples/runnable/yaml_config.ail`.
- [ ] `GOOS=js GOARCH=wasm go build ./...` succeeds, confirming the pure bridge remains WASM-buildable.

**Risk:** Long repository checks exceed the allotted hour or expose unrelated failures. Record exact failures and distinguish unrelated baseline failures; never report a check passed unless run successfully.

## Success Metrics and Handoff

All three milestones pass, all guarantee rows are covered, both existing AILANG fixtures run, and the user-facing order contract is explicit. Use behavioral coverage of the matrix rather than an unsupported global coverage percentage. No parser/type/effect/ADT changes, package additions, sorting flag, encode or decodeAll work is included.

Execution is sequential M1 → M2 → M3. Progress lives in `.ailang/state/sprints/sprint_M-STD-YAML-MAPPING-KEY-ORDER.json`, initially not_started with null passes. PR #1620 is merged triage context, not an open bug issue to close; no issue number is invented. Coordinator approval/merge handles the sprint-executor handoff; do not launch duplicate execution before that gate. After execution, run sprint-evaluator against the design and this acceptance matrix.

No unresolved product decisions remain: the handoff approves option (a). Node mechanics, cycles and merge validation are implementation risks resolved by compiled tests, not assumed verified in this planning session.
