### Changed — connector guide: `sunholo/mcp_files` is released, new "Google platforms" section (2026-11-05)

`docs/docs/guides/mcp-connectors.md` §5 no longer calls `sunholo/mcp_files` unreleased, because 0.1.4
is in the registry. A new §6 says which Google surfaces an AILANG service can use today:
- Gemini Enterprise custom MCP data store: works IAM-only, but not with OAuth sign-in.
- Google Connected Apps: does not work yet.
- Gemini Enterprise A2A: does not work yet.

The submission checklist is now §7. The design that closes these gaps is
`design_docs/planned/v0_57_2/m-mcp-oauth-client-models.md` (`sunholo/mcp_oauth` 0.2.0 client models,
the serve-api identity verifier, and listing hygiene). Its Design Freeze items are open.
