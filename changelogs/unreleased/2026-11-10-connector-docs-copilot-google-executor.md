### Changed
- **Connector guide covers every host an AILANG MCP service can reach**
  (`docs/docs/guides/mcp-connectors.md`). A platform table at the top gives the client model each
  host uses and the status today. §6 (Google) is rewritten for `sunholo/mcp_oauth` 0.2.0:
  pre-registered confidential clients for Gemini Enterprise, DCR, stable refresh and revocation for
  Connected Apps, a checked `ClientConfig` snippet, and A2A still unsupported (`serve-api --a2a` is
  pre-0.2; GE needs v0.3). New §7, **Microsoft Copilot**: Microsoft 365 Copilot declarative agents and
  Copilot Studio, their sign-in options (static client, DCR, Entra SSO, API key), redirect URLs, the
  widget-renderer CORS origin, and which MCP Apps features Copilot lacks (`openai/fileParams`, all
  hints but `readOnlyHint`), from Microsoft Learn pages read 2026-11-10.

### Added
- **[Hosted AILANG executor](docs/docs/guides/hosted-executor.md)**: the public page for
  `https://executor.ailang.sunholo.com/mcp/connect/`. Its tools, the sandbox limits (`IO` only,
  restricted mode, 60 s, 64 KB in and out, 30 calls a minute per account), what is logged (never the
  code), and how to add it to Claude, Claude Code and ChatGPT.
