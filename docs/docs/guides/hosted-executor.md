---
reviewed: 2026-11-10
reviewBy: 2027-02-10
---

# Hosted AILANG executor

The hosted executor is an MCP server that lets an AI assistant **write, check and run AILANG code
without installing anything**. It is the same toolset the local `ailang_bootstrap` plugin gives an
agent, run on our infrastructure inside a locked-down sandbox, plus the AILANG documentation tools.

```
https://executor.ailang.sunholo.com/mcp/connect/
```

Sign in with any Google account whose email address is verified. The executor keeps your Firebase
user ID, the OAuth client you signed in through, and digests (not copies) of the tokens it issued
you, each until it expires.

It is also the reference example of an AILANG service published as a connector: everything on this
page is built with `serve-api` and `sunholo/mcp_oauth`, as described in
[Publish an AILANG service as an MCP connector](./mcp-connectors.md).

## Tools

| Group | Tools | What they do |
|---|---|---|
| **Code** | `ailang_prompt` | The current AILANG syntax prompt. Read it before writing code |
| | `ailang_check` | Type-check a program (and run its contracts through Z3) without running it |
| | `ailang_run` | Run a program in the sandbox and return its output |
| | `ailang_eval` | Evaluate one expression |
| | `ailang_builtins` | Search the builtin functions |
| **Docs** | `docs_search`, `docs_nav`, `stdlib_modules`, `stdlib_module`, `stdlib_search`, `effects_catalog`, `limitations_list`, `examples_list`, `example_get`, `example_for_concept`, `prompt_get`, `changelog_for_version`, `ailang_versions`, `install_guide`, `onboarding_guide`, `roadmap`, `design_docs_list`, `design_doc`, `benchmarks_list`, `benchmark_run`, `benchmarks_for_model`, `benchmarks_compare` | The AILANG documentation, versioned. These are the tools of the open docs server `https://mcp.ailang.sunholo.com/mcp/`, which needs no sign-in |
| **Account** | `executor_whoami` | The account and OAuth client the call was made as |
| | `submit_feedback` | Report a bug or a missing feature to the AILANG team (no sign-in needed) |

Every tool except `submit_feedback` needs sign-in. All of them are read-only: running code changes
nothing outside its own temporary directory.

## The sandbox

Each `ailang_run`, `ailang_eval` and `ailang_check` call gets a fresh directory holding one file,
`main.ail`, which is deleted afterwards. The program runs through `ailang run --policy` with a fixed
policy (`docker/executor-policy.toml`); the caller cannot change it.

| Limit | Value |
|---|---|
| Capabilities | `IO` only: no network, no environment, no clock, no process, no file system outside the request directory |
| Security mode | `restricted` |
| Time | 60 seconds; the supervisor then kills the whole process group |
| Source size | 64 KB |
| Output | 64 KB |
| Entry point | `main` |
| Rate | 30 calls a minute per account, with bursts of up to 30. Over the limit, the call gets HTTP 429 and a `Retry-After` header |

A program that asks for a capability it does not have gets a named policy refusal, not a crash.

**What is logged.** One line per call: the tool, the outcome, the duration, the size of the code,
a short hash of it (so repeated retries of the same code are visible) and the first error with the
request directory removed. **The code itself is never logged.**

## Add it to your assistant

**Claude (claude.ai, desktop, mobile).** Settings → Connectors → **Add custom connector**, and paste
the URL above. Choose **"Sign in when needed"** and **"Use Claude's published identity (CIMD)"**:
there is no client ID or secret to enter. The first tool call opens the Google sign-in.

**Claude Code.**

```bash
claude mcp add --transport http ailang-executor https://executor.ailang.sunholo.com/mcp/connect/
```

Then run `/mcp` in a session and choose the server to sign in.

**ChatGPT.** Turn on developer mode, add a connector with the URL above, and sign in when ChatGPT
asks. ChatGPT registers itself with CIMD and the executor returns `iss`, so there is nothing to
configure. A live ChatGPT sign-in has not been recorded yet.

**Gemini Enterprise, Microsoft Copilot.** Not available yet for this server. Both need an OAuth
client model (a pre-registered client or dynamic registration) that the executor has not switched
on. The status of each platform is in [Where an AILANG MCP service can appear](./mcp-connectors.md#where-an-ailang-mcp-service-can-appear).

**Without sign-in.** For the documentation tools only, use `https://mcp.ailang.sunholo.com/mcp/`.
To run code on your own machine instead, install AILANG and the `ailang_bootstrap` plugin, which
serves the same `ailang_*` tools over a local stdio MCP server.

## Check it yourself

```bash
ailang mcp check https://executor.ailang.sunholo.com/mcp/connect/ --target both
```

This checks the server name, the tool titles and hints, that no tool takes a credential, that
sign-in starts with a 401 naming the protected resource, the authorization server's metadata and
that the sign-in page cannot be framed.
