---
sidebar_position: 7
title: motoko Extension Author Workflow
---

# motoko Extension Author Workflow

The end-to-end developer loop for writing a motoko_agent extension as a reusable AILANG package: scaffold → iterate locally over a path-dep → smoke → publish → version-pin. The four steps mirror Arni's proposal in [arniwesth/motoko_agent#23](https://github.com/arniwesth/motoko_agent/discussions/23) and are the canonical author flow as of AILANG `v0.20.1`.

If you want a guided tutorial of *one specific extension* (an OpenKB search tool), read [Build Your First motoko Extension](./build-a-motoko-extension.md) first — this page is the reference for the *workflow itself* and the conventions that make extensions safe to ship to production agents.

## Why extensions exist (and when to write one)

A motoko_agent extension is a self-contained AILANG package that adds tools, system-prompt material, budget rules, response intercepts, or pre-step decisions to an existing agent — without forking `motoko_agent` itself. The 10-hook `ExtensionHooks` ABI lets a single package contribute across all of those concerns at once.

**Write an extension when:**
- The tools you want to expose are reusable across agents (search, code-execute, MCP-bridge, KB-fetch).
- You need provider-portable tool definitions that work on Anthropic, OpenAI, Bedrock, Vertex, and Gemini without per-provider forks.
- You want to ship the capability behind a registry version other agents can pin.

**Don't write an extension when:**
- It's one ad-hoc tool used by exactly one agent — put it in that agent's `src/core/ext/` and revisit when the second consumer asks.
- It encodes a business-policy decision specific to one deployment — that lives in the agent's config, not the package.

## The canonical 4-step loop

```
1. Implement in ailang-packages/packages/<name>/    (your local checkout)
2. Path-dep from the consuming agent's ailang.toml  (no publishes during iteration)
3. Smoke locally with _smoke.ail                     (catches breakage before users do)
4. Publish + version-pin                             (one command, gate runs at registry)
```

### Step 1 — Implement in `ailang-packages/`

Scaffold the package shape with [`ailang init motoko-extension`](./build-a-motoko-extension.md#step-1-scaffold-the-package-one-command) — it generates a package that already passes `ailang check --package .` and `ailang lock`, so you can focus on logic, not boilerplate.

The 10-hook layout the scaffold produces (as of v0.20.1):

| Hook | Purpose | Default |
|---|---|---|
| `provided_tools` | Names this extension advertises | `[]` (you fill in) |
| `on_describe_tools` | JSON-schema-ish placeholders the model sees | `\_ . [...]` |
| `on_tool_handle` | Pattern-match `call.name` → run | `\_ . Delegate` |
| `on_system_prompt` | Append/replace prompt text | `\msgs . msgs` |
| `on_budget` | Decide if a call should proceed under budget | `\_ . Allow` |
| `on_policy` | Decide if a call is permitted by policy | `\_ . Allow` |
| `on_response` | Intercept a model response | `\r . r` |
| `on_solver` | Custom solver dispatch | `\_ . None` |
| `on_pre_step` | **New in abi 2.2.0**: short-circuit before model is called | `\_ . PassThrough` |
| `on_step_complete` | Telemetry on step finish | `\_ . unit` |

You only need to replace the hooks relevant to your tool — leave the rest as no-ops.

### Step 2 — Path-dep from the consuming agent

In the consuming agent's `ailang.toml`:

```toml
[dependencies]
sunholo/motoko_ext_context_mode = { path = "../ailang-packages/packages/motoko-ext-context-mode" }
```

This is **the most important friction-removal in the whole loop**. Without path-deps you'd be publishing a new package version every time you tweak a tool's schema or fix a typo in `on_tool_handle`. With path-deps:

- `ailang check` and `ailang lock` resolve directly from your local checkout.
- Edits in `packages/motoko-ext-foo/` are picked up by the consumer on the next `ailang check` — no publish, no `ailang lock` thrash, no version bump.
- The consumer's `ailang.lock` records the path source so collaborators see immediately that this is a local-iteration setup, not a published-version setup.

Switch back to a registry version (`{ version = "0.2.3" }`) when you're ready to ship. The diff in `ailang.toml` is the *only* change required — no consumer-side code edits.

### Step 3 — Smoke locally with `_smoke.ail`

Every extension package the scaffold generates ships with a `_smoke.ail` file containing 3-5 assertions that exercise the basic shape of the extension:

```ailang
module sunholo/motoko_ext_foo/_smoke

import std/io (println)
import sunholo/motoko_ext_foo/register (register_with_config)

export func main() -> Unit ! {IO, Env, FS} {
  let hooks = register_with_config({});
  let names = hooks.provided_tools;
  let count = length(names);
  if count == 0 then
    println("FAIL: no provided_tools")
  else
    println("OK: register returns ${show(count)} tools")
}
```

Run with:

```bash
cd packages/motoko-ext-foo
ailang run _smoke.ail
```

The smoke file is **part of the publishable package** — the registry's validator runs it during the publish smoke step. If it panics or asserts during local iteration, you catch the breakage before users do. If it panics at the registry, the publish is rejected with the same error message you'd have seen locally.

The patterns the scaffold seeds:
1. **Constructor doesn't panic** — `register_with_config({})` returns a value, doesn't crash.
2. **Advertised tool count matches expectation** — `length(hooks.provided_tools) > 0` (or `== N` if you know the exact number).
3. **`on_describe_tools` returns schemas** — typically `length(hooks.on_describe_tools(unit)) == length(hooks.provided_tools)`.

Add more assertions as the extension grows — e.g. a synthetic `on_tool_handle` invocation with a fabricated `ToolCall` to verify routing, or an `on_pre_step` dry-run for cache-key generation.

### Step 4 — Publish + version-pin

```bash
cd packages/motoko-ext-foo
ailang pkg publish
```

The registry validator runs the package through:
1. **Compile check** — `ailang check --package .` must pass.
2. **Smoke run** — `_smoke.ail` must `main()` without panicking.
3. **Tool-name gate** — every name in `provided_tools` and every `name:` field inside an `on_describe_tools` `ToolSchema` must match `[A-Za-z0-9_]{1,128}`. (See the next section for *why*.)

On success, the registry returns the published tarball URL. Update the consumer's `ailang.toml` to pin the new version:

```toml
sunholo/motoko_ext_foo = { version = "0.3.0" }
```

Then `ailang lock` to write a frozen hash into `ailang.lock`.

---

## Provider-safe tool naming (and why dots break Bedrock)

**Rule**: advertised tool names must match `[A-Za-z0-9_]{1,128}`. No dots, no hyphens, no colons, no whitespace.

This is the intersection of what the major model providers accept:

| Provider | Accepts |
|---|---|
| Anthropic API | broad (regex permissive) |
| **Anthropic Bedrock** | **`[A-Za-z0-9_]{1,128}` only** |
| OpenAI | broad |
| **Google Vertex AI** | **`[A-Za-z0-9_]{1,128}` only** |
| Gemini AI Studio | broad |
| OpenRouter | broad |

A tool named `ctx.execute` works on Anthropic-API + OpenAI + AI-Studio + OpenRouter, *fails silently* on direct Anthropic in some configurations, and **fails loudly with a 400 from Bedrock and Vertex**. That last category is what made the v0.18.1 incident untraceable — the agent worked in development, broke in production, and the error message named the wrong layer.

As of `v0.20.1` the **registry validator rejects dotted/hyphenated/colon'd tool names at publish time**. The rejection response includes:
- The first offending name.
- A precise reason (`contains '.'`, `contains invalid character '@'`, etc.).
- Two safe-rewrite suggestions: snake_case (`ctx_execute`) and PascalCase (`CtxExecute`).

So an AI agent authoring an extension can correct on the first publish attempt rather than learning about the constraint from a Bedrock deployment failure weeks later.

### Migration: I have a package with dotted names already published

If you maintained a pre-`v0.20.1` package whose `provided_tools` includes dotted forms (e.g. `ctx.execute`), the v0.20.1 publish will reject it. Two paths:

**Recommended**: bump the major version, rename the advertised names to snake_case or PascalCase, keep the dotted forms ONLY as **input aliases** inside `canonical_tool_name(raw)` (the agent-side normalization that maps whatever the model says back to your canonical name). The advertised name the model sees is safe; the input matcher still accepts legacy forms.

**Migration escape hatch**: `ailang pkg publish --allow-dotted-tool-names` sets the `X-Allow-Dotted-Tool-Names: true` header, which the validator honors with a warning. Use this for the one-time republish of a package you're about to deprecate or rename — not for new packages.

The pattern mirrors `--allow-unsafe-field-access` (shipped in M-WASM-AI-STEP-BYO-KEY, v0.19.1).

---

## `on_describe_tools` cookbook

The scaffold seeds placeholder schemas with `'{"type":"object","required":[],"properties":{}}'` (a literal JSON string the model reads as JSON Schema). Replace with concrete shapes per tool:

**Command-execution tool** (a `language` + `code` pair):

```ailang
{ name: "CodeExecute",
  description: "Execute a code snippet in a sandboxed runtime",
  parameters: '{"type":"object","required":["language","code"],"properties":{"language":{"type":"string","enum":["python","node","go"]},"code":{"type":"string"}}}' }
```

**Search tool** (a single `query`):

```ailang
{ name: "KBSearch",
  description: "Search the KB for relevant chunks",
  parameters: '{"type":"object","required":["query"],"properties":{"query":{"type":"string"},"top_k":{"type":"integer","default":5}}}' }
```

**Fetch tool** (a structured URL):

```ailang
{ name: "PageFetch",
  description: "Fetch a URL and return rendered text",
  parameters: '{"type":"object","required":["url"],"properties":{"url":{"type":"string","format":"uri"}}}' }
```

**Structured-data tool** (free-form object):

```ailang
{ name: "RecordIngest",
  description: "Append a record to the structured log",
  parameters: '{"type":"object","required":["record"],"properties":{"record":{"type":"object","additionalProperties":true}}}' }
```

The model's tool-calling layer reads these schemas verbatim. They drive both the **what arguments the model knows to pass** and **the JSON-schema validation the provider runs on the model's output** before the call reaches your `on_tool_handle`.

---

## Publish checklist

Before `ailang pkg publish`:

- [ ] Bumped `version =` in `ailang.toml` (semver: patch for bugfix, minor for new tool, major for renamed tool).
- [ ] All advertised names in `provided_tools` and `on_describe_tools` match `[A-Za-z0-9_]{1,128}`.
- [ ] `ailang check --package .` clean.
- [ ] `ailang run _smoke.ail` exits 0.
- [ ] CHANGELOG entry inside the package's `CHANGELOG.md` (if it has one).
- [ ] ABI pin in `[dependencies] sunholo/motoko_ext_abi = "X.Y.Z"` matches the hook fields you're using (2.2.0 if you use `on_pre_step`).
- [ ] `[effects].max` lists every effect any hook touches (`Env`, `FS`, `IO`, `Process`, `Net`, `Trace`, etc.).

---

## Canonical examples to read

- **[`sunholo/motoko_ext_test_dummy`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-test-dummy)** — minimal one-tool extension. The first thing to read if you've never seen the layout.
- **[`sunholo/motoko_ext_context_mode`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-context-mode)** — production-scale extension with 18 tools, real `on_tool_handle` routing, JSON-schema `on_describe_tools`, and an `on_pre_step` cache-key calculator. Bumped from a stub to a real implementation in M-EXT-AUTHOR-DX M1.
- **[`sunholo/motoko_ext_mcp`](https://github.com/sunholo-data/ailang-packages/tree/main/packages/motoko-ext-mcp)** — bridges MCP servers as tools. Demonstrates the `Delegate` + `Handled` discriminated-return pattern.

---

## Historical context

This workflow crystallized over three sprints:

| Sprint | Version | Contribution |
|---|---|---|
| M-EXT-SCAFFOLD-AI-FIRST | v0.18.5 | `ailang init motoko-extension` generates a working package shape |
| M-EXT-PORTABILITY-GATE | v0.18.11 | `ailang check --package .` + registry-side smoke at publish time |
| **M-EXT-AUTHOR-DX** | **v0.20.1** | Scaffold seeds `_smoke.ail` + `on_describe_tools` placeholders; registry rejects Bedrock-unsafe names at publish |

The motivation, gaps, and rationale are written up in [arniwesth/motoko_agent#23](https://github.com/arniwesth/motoko_agent/discussions/23) and the [M-EXT-AUTHOR-DX design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_20_1/m-ext-author-dx.md).
