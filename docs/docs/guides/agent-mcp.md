# Agent MCP — Live AILANG Knowledge for Coding Agents

`mcp.ailang.sunholo.com` is a remote [Model Context Protocol](https://modelcontextprotocol.io/) server that exposes ~21 typed tools so AI coding agents can query AILANG documentation, stdlib, examples, design docs, benchmarks, and prompts as **structured data** — instead of scraping markdown.

If you're building with AILANG and your harness supports remote MCP (Claude Desktop, Cursor, Cline, Continue, Claude Code), you can add the endpoint in one click from [the landing page](/) and your agent gains live, version-pinned AILANG knowledge.

## Why use it

| Without the MCP | With the MCP |
|---|---|
| Agent fetches `llms.txt` (huge, no filter) | Agent calls `stdlib_search("file")` and gets just the matching functions |
| Agent guesses URLs from the sidebar | Agent calls `docs_nav()` and gets the structured tree |
| Agent doesn't know which prompt fits its language version | Agent passes `for_version=$AILANG_VERSION` and never gets stale content |
| Agent has no way to learn which models pass which benchmarks | Agent calls `benchmarks_for_model("claude-haiku-4-5")` and sees pass-rate by category |

The server is **public, anonymous, read-only**. There's one write tool (`submit_feedback`) for filing AILANG bug reports / feature requests / docs gaps directly from inside an agent session — those land in our triage queue.

## Connecting

### Quick: one-click deeplinks

The buttons on the [landing page](/) pre-fill the MCP config for Claude Desktop, Cursor, and Cline. Click, confirm, you're done.

### Manual: add to MCP config

For other harnesses (Continue, etc.) or if the deeplink fails, paste this into your MCP config file:

```json
{
  "mcpServers": {
    "ailang-docs": {
      "url": "https://mcp.ailang.sunholo.com/mcp/",
      "transport": "streamable-http"
    }
  }
}
```

For Claude Desktop the file is at `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows).

## Tool catalog

All tools return JSON. Version-scoped tools take an optional `for_version` argument; passing an empty string resolves to "latest". Responses for version-scoped tools are wrapped in a `{served_for, data}` envelope so callers can detect cross-version content leak.

### Language reference (version-scoped)

| Tool | Purpose |
|---|---|
| `ailang_versions` | List all versions the server can answer for |
| `prompt_get` | Fetch the canonical teaching prompt (`kind`: `agent`/`devtools`/`compact`/empty for full) |
| `stdlib_modules` | All stdlib modules with one-line summaries + export counts |
| `stdlib_module` | Full exports for one module (signatures + docstrings) |
| `stdlib_search` | Find stdlib functions by name/signature/keyword |
| `effects_catalog` | All declared effects with capability mapping + introduction version |
| `limitations_list` | Known design limitations (Y-combinator, etc.) optionally by category |

### Examples & patterns (version-scoped)

| Tool | Purpose |
|---|---|
| `examples_list` | Filter examples passing under `for_version`, by features/status |
| `example_get` | Full source + metadata for one example by path |
| `example_for_concept` | Best-fit example for a concept query ("how do I use effects?") |

### Docs & navigation (version-scoped)

| Tool | Purpose |
|---|---|
| `docs_nav` | Sidebar tree as JSON (replaces scraping the Docusaurus sidebar) |
| `docs_search` | Full-text search across the docs Markdown |

### Design, roadmap, changelog (unscoped — historical record)

| Tool | Purpose |
|---|---|
| `design_docs_list` | List planned/implemented/rejected design docs, optionally by version |
| `design_doc` | Full design doc by slug |
| `roadmap` | What's shipping next, what's been rejected and why |
| `changelog_for_version` | What changed in a given release |

### Benchmarks (unscoped)

| Tool | Purpose |
|---|---|
| `benchmarks_list` | List benchmark suites and runs, optionally `since_version` |
| `benchmark_run` | One run's full result (code, compile_ok, runtime_ok, cost, tokens) |
| `benchmarks_for_model` | Where does a given model pass/fail? |
| `benchmarks_compare` | Pairwise model comparison (a-only, b-only, both, neither) |

### Public feedback (the only write tool)

| Tool | Purpose |
|---|---|
| `submit_feedback` | Anonymous bug / feature / docs / limitation report — queued for human review |

Submissions land in our `ailang messages list --inbox public-feedback` queue. Anonymous is fine; the value is the signal, not the identity.

## Versioning model

Each Cloud Run image bakes in a snapshot of the AILANG corpus pinned to the release that built it. Multi-version is handled inside the tool calls (`for_version`), not at the URL — there's exactly one endpoint, `mcp.ailang.sunholo.com/mcp/`, and the data inside it is version-scoped.

If you ask for a version that isn't in the snapshot, you get `{error: "unknown_version", nearest: "..."}` rather than a silent downgrade. The CLI's `ailang prompt --source auto` flow uses this to fall back to its embedded copy without ever surfacing wrong-version content to the user.

## Caps & determinism

- Cloud Run service runs `ailang serve-api --mcp-http --routes-only --caps FS,Env`
- `--routes-only` means only `@mcp_name`/`@route` exports register as tools — no helper leakage
- `--caps FS,Env` is the only ambient authority (filesystem reads from the baked snapshot, env reads for snapshot dir config). `Net` is added in v0.15+ when `submit_feedback` publishes to the existing `ailang-messages` Pub/Sub topic
- Tool replies are deterministic for a given image SHA — replays are byte-identical

## Source

The MCP tools are written in AILANG itself ([`mcp_tools/`](https://github.com/sunholo-data/ailang/tree/dev/mcp_tools) in the AILANG repo) and registered automatically via the `@mcp_name`/`@route` annotations. Adding a new tool = writing one `.ail` file. The Go-side framework is [`internal/apiserver/mcp.go`](https://github.com/sunholo-data/ailang/blob/dev/internal/apiserver/mcp.go).

The snapshot is built by [`tools/build-snapshot/`](https://github.com/sunholo-data/ailang/tree/dev/tools/build-snapshot) which reads stdlib `.ail` files, baseline benchmarks, prompts, changelog, and design docs and writes JSON. The snapshot is baked into the Cloud Run image at build time so each revision is a frozen point in time.

Design doc: [M-AGENT-MCP](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_15_0/m-agent-mcp-website.md).
