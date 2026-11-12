### Added
- **Executor directory submission kit** (`design_docs/planned/v0_51_0/m-executor-directory-submission-kit.md`).
  It covers the hosted executor against the Anthropic connector directory (M1) and the OpenAI
  plugin directory (M2), in the same shape as Parse's kit:
  - a gap table with live evidence from 2026-11-12;
  - the listing text: name, a 153-character one-liner, a 25-character short description and a
    1,418-character long description;
  - five Anthropic prompts and OpenAI test cases (five positive, three negative), each checked
    against the prod tools;
  - reviewer instructions with credential placeholders, and `plugin.json`/`mcp.json` values;
  - Mark's remaining steps in order, including the Firebase console and terraform changes.

  **Main finding:** six docs tools (`roadmap`, `design_doc`, `example_get`, `prompt_get`,
  `benchmarks_compare`, `benchmark_run`) have never worked on either hosted server, because
  `tools/build-snapshot` does not generate the files they read. Neither directory submission can go
  ahead until that is fixed.
