# IFC Label Refinements Lost Across Module Boundaries (Daneel, v0.36.0)

- **Date**: 2026-09-17
- **Class**: bug
- **Recommend**: duplicate-of design_docs/planned/m-ifc-cross-module-labels.md
- **Searched**: `rg -il "M-TAINT|taint|label.*enforce|information-flow" design_docs/`; `ls design_docs/planned/`; then read `m-ifc-cross-module-labels.md` and `m-ifc-authority-scoping.md` in full; `grep -n "Record|field" internal/types/ifc_check.go`
- **Estimate**: n/a (duplicate)

The report is the exact subject of the already-planned M-IFC-CROSS-MODULE doc
(`design_docs/planned/m-ifc-cross-module-labels.md`, target v0.39.0), which cites
the same reporter and version ("GitHub issue #1134 (daneel, measured on
v0.36.0)") and the same repro shape — an exported `gate(input: string{not email})`
callee whose refinement is never checked by an importing module. The root cause
named there matches: `CheckModuleIFC` (`internal/types/ifc_check.go`) is a
surface-AST pass over local declarations only, and `labelOfCall` treats imported
callees as transparent, skipping the refinement check and under-approximating
the result label. The doc's proposed fix (serializable IFC summary in the iface,
schema v2, dep-side intrinsic label, caller-side wiring in pipeline + REPL) and
its success criterion #1 ("the #1134 repro (gate function moved to another
module) fails to compile") cover everything the reporter asked for. No new
design doc is warranted.

Two secondary items in the report, for whoever executes the doc:

1. **Record-field refinements are unenforced and untracked.** The report shows
   `{ body: <email> }` compiling clean against a field declared
   `string{not email}`; confirmed plausible in code — `labelOf` handles
   `*ast.Record`/`*ast.RecordUpdate` for *label propagation*
   (`internal/types/ifc_check.go:215-229`) but nothing in Check A walks
   field-declared refinements, and neither planned IFC doc claims that case.
   Not covered by the duplicate; recommend appending it to
   M-IFC-CROSS-MODULE's scope (it is the same sink-side refinement mechanism,
   just a record-field edge) or filing it as its own row if the executor
   prefers to keep the doc's scope fixed. The reporter's own workaround
   (constructor params carrying the refinement) then hits the module-boundary
   bug, so this is only a workaround once cross-module lands.
2. **Stale comment in `examples/runnable/contracts/inbox_injection_v2.ail`**
   (lines 17-19, 80, 102): says label enforcement is "Phase 2+" / "Today
   (Phase 1 demo)", but same-module type-level enforcement is live on v0.36.0.
   This is a genuine two-line doc-comment fix inside one file — but it sits
   inside the example this doc's sprint will touch anyway, so folding it into
   the M-IFC-CROSS-MODULE sprint's docs pass is cheaper than a standalone
   direct-fix PR.

Reporter's why-this-matters stands as written in the doc's Impact paragraph:
library-declared taint sinks are silently unenforceable for importers, which is
the exact use case (sunholo/gmail guarding Daneel's mail intake behind an
explicit `! {Declassify}` step).

**Note on the related planned doc**: `m-ifc-authority-scoping.md` (#752, also
target v0.39.0) explicitly defers cross-module propagation to the companion doc
and coordinates the `declassify` field shape with it — no conflict.
