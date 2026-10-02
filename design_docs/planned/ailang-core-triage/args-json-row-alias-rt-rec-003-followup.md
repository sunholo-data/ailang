# Follow-up reply: args-json Row alias + RT_REC_003 in tests (fixes already landed)

- **Date**: 2026-10-02
- **Class**: already-covered
- **Recommend**: drop
- **Searched**: `args-json`, `max-recursion-depth`, `RT_REC_003`, `Row` across `design_docs/`; commits and tests in the repo
- **Estimate**: n/a (nothing to fix — this is a status reply, not a defect report)

This message (`inbox_1790924413393_307ce5e7`, reply from ailang-core on `stapledons_godot`) is a
follow-up to two 2026-10-01 reports, and everything it claims is verifiably already landed in this
tree, so there is nothing to triage into work:

1. **args-json + renamed Row alias**: fixed and pinned by
   `internal/pipeline/renamed_type_import_test.go` (`TestRenamedTypeImport_Expands`), whose comment
   cites this exact case ("stapledons_godot, 2026-10-01"): `import M (Row as R)` previously left `R`
   an opaque constructor; the fix registers the alias under the bind name in
   `internal/pipeline/pipeline_module_imports.go` (`import M (Row as R)` handling, ~:226). A plain
   `import M (Row)` already worked before it.
2. **RT_REC_003 in tests**: `ailang test --max-recursion-depth N` exists —
   `cmd/ailang/commands_language.go:240` (flag, default 10000, "same as ailang run") and
   `internal/testing/config.go:39` — and is pinned by `internal/testing/max_recursion_depth_test.go`.
   The real fix (tail-call elimination, #1486) and the `--bytecode` test-mode request (#1487) are
   open issues; #1485 (checker accepts undeclared type names) was also filed from the same work.
   (Issues not verifiable here — `gh` unavailable in this environment — but the code-level claims
   all check out.)

The message was filed to `stapledons_godot`, which serves no agent, which is why it stalled; the
content itself needs no design doc, no direct fix, and no backlog row beyond this record. If the
original reporter's case was something other than a renamed import, the reply asks for the exact
import line — that hypothetical follow-up would be a new report and should be triaged on arrival.

`gh` was unavailable, so #1485/#1486/#1487 existence is taken on the reply's word; nothing here
depends on it.