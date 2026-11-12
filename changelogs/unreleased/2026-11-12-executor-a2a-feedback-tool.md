### Added
- **The hosted executor's A2A agent can file feedback.** `executor_tools/agent.ail` gives the model a
  third tool, `submit_feedback` (title, body, category `bug|feature|docs|limitation`), which it calls
  only when the user asks to report something. The report goes through the public docs MCP's
  `submit_feedback` (`EXECUTOR_FEEDBACK_MCP_URL`, default `https://mcp.ailang.sunholo.com/mcp/`), so
  it gets the same validation, the `public-feedback` inbox and the per-IP limit as any MCP client;
  the body starts with a line naming the A2A agent as its origin, and `ailang_version` is
  `hosted-executor-a2a`. The user is told the ticket id. One log line per report
  (`executor_a2a_feedback`: the HTTP status only). Gemini Enterprise users of the agent had no way to
  send feedback; the MCP surfaces already list the tool.
