# AILANG Core Backlog (triage rows)

| Date | Title | Class | Recommend | Why |
|---|---|---|---|---|
| 2026-09-15 | Local dependency override (Go-style `replace`) for development vs registry resolution | feature | design-doc | Introduces a new manifest surface (`ailang.local.toml` vs `[replace]`+`AILANG_DEV=1`), resolve-time override semantics, and lock/visibility behaviour — a trade-off between workable designs that needs a ruling; no existing doc covers it (searched "override", "ailang.local", "[replace]", "path dep", "dev-dependenc" across design_docs/; the related absolute-path-in-lock bug is tracked separately). |
