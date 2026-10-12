### IFC cross-module summary and cache foundation (Refs #1134; Refs #1132 part 1)

- Export deterministic v2 interface summaries carrying parameter labels and sink refinements, declared and intrinsic return labels, and scoped Declassify authority.
- Preserve joined constant/variable labels, recursive intrinsic taint, and explicit clean metadata in compact JSON and gob interfaces.
- Imported call enforcement and CLI/REPL integration remain scheduled for M3; this partial sprint does not yet enforce cross-module sinks.
