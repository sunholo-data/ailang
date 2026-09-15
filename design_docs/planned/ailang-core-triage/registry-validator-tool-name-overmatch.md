# Registry validator over-matches `name:` fields as advertised tool names (daneel_ext_help publish rejected)

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `provider-safe`, `provided_tools|on_describe_tools`, `register_with_config` across `internal/`, `cmd/`, `design_docs/`
- **Estimate**: omitted (design-doc)

The gate lives in `cmd/registry-validator/tool_names.go` (`validateToolNames`, M-EXT-AUTHOR-DX M3 / v0.20.1). Pattern 2 (`nameFieldPattern = \bname\s*:\s*"([^"]+)"`) deliberately over-matches every `name: "X"` string literal in any `.ail` file in the package — the code comment itself admits it "will also catch `name:` fields from other record literals." So a human-readable `Authority.name` title returned by `register_with_config` ("Explain how to ask") is treated as an advertised tool name and blocks publish. This is the same false-positive class the #1133 fix patched by shape (header key/value records carrying a `value` sibling), and this report shows the pattern class recurs — each recurrence is currently met with another shape-based exemption. The proposed fix (scope the scan to `provided_tools` / `on_describe_tools` ToolSchema blocks, or a package-level "advertises no tools" declaration) changes the provider-safety gate's contract, and narrowing naively risks provider-unsafe names slipping through (the v0.18.1 Bedrock/Vertex incident class the gate exists to close). That trade-off — scanner scoping vs. explicit declaration vs. more exemptions — is a decision someone could disagree with, so it warrants a design doc rather than another exemption patch.

Note: the same report is already logged in `design_docs/planned/ailang-core-backlog.md` (2026-09-15, same classification and file citation); this row exists as the per-report triage artifact.