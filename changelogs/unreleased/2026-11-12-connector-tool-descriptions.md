### Changed
- **Tool descriptions on listed connectors now say only what each tool does.** Directory reviewers
  read every description: Anthropic requires that they match behaviour, and OpenAI forbids internal
  identifiers and anything that steers the model.
  - `submit_feedback` (the built-in serve-api tool, on every serve-api MCP server including the
    executor, the docs server and AILANG Parse) no longer mentions `pkg:` inboxes, "autonomous
    agents", Pub/Sub tags, coordinator filtering or "a separate sprint". It now tells the model to
    send a report only when the user asks, and to leave out personal or confidential information.
    The `contact` and `auto_dispatch` descriptions also say "only if the user gives one / asks".
    The arguments and behaviour are unchanged.
  - `docs_search`, `docs_nav`, `onboarding_guide` and the executor's `executor_whoami` lose the
    implementation notes in their descriptions ("Replaces scraping the Docusaurus sidebar", "v2 may
    filter… record-projection helpers", "too much for a tool result in a listed connector", "the
    gate… _auth"). `onboarding_guide` now says it returns both roles whatever `role` says, which is
    what it does.
