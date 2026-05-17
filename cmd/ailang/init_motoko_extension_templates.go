package main

// M-EXT-SCAFFOLD-AI-FIRST (v0.18.5) — file templates for `ailang init
// motoko-extension`.
//
// The scaffolder emits 5 files, all derived from these Go templates:
//
//   1. ailang.toml           — package metadata + registry deps + exports + effects
//   2. register.ail          — canonical register_with_config wrapper
//   3. types.ail             — placeholder type stub (user fills in)
//   4. <short>.ail           — full make_hooks impl with all 8 ExtensionHooks
//                              fields populated as no-op defaults
//   5. README.md             — points at tutorial + publishing guide
//
// Adding fields to the canonical ExtensionHooks shape (currently 8 hook
// fields) means updating these templates AND the snapshot golden files
// in M3. The integration test (`ailang lock + ailang check` against
// generated output) catches drift loudly.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

// templateData feeds all 6 file templates with deterministic placeholders.
type templateData struct {
	Name        string   // "arniwesth/motoko_ext_openkb"
	ShortName   string   // "openkb"
	Tools       []string // ["OpenKBSearch", "OpenKBList"]
	Effects     []string // ["FS", "Process"]
	EffectsCSV  string   // ailang.toml-friendly CSV: `"FS", "Process"`
	ToolsAilang string   // .ail-friendly list literal: `["OpenKBSearch", "OpenKBList"]`
	ToolSchemas string   // .ail-friendly on_describe_tools body — per-tool ToolSchema records with valid empty JSON-Schema placeholders
}

// scaffoldMotokoExtension renders all 5 files and writes them under outDir.
// Refuses to overwrite an existing directory (the caller already checked,
// but we re-check here for defense-in-depth in case of races).
func scaffoldMotokoExtension(outDir, name, shortName string, tools, effects []string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %q: %w", outDir, err)
	}

	data := templateData{
		Name:        name,
		ShortName:   shortName,
		Tools:       tools,
		Effects:     effects,
		EffectsCSV:  toQuotedCSV(effects),
		ToolsAilang: toAilangStringList(tools),
		ToolSchemas: toAilangToolSchemas(tools),
	}

	files := []struct {
		path string
		tmpl string
	}{
		{filepath.Join(outDir, "ailang.toml"), tmplAilangToml},
		{filepath.Join(outDir, "register.ail"), tmplRegisterAil},
		{filepath.Join(outDir, "types.ail"), tmplTypesAil},
		{filepath.Join(outDir, shortName+".ail"), tmplImplAil},
		{filepath.Join(outDir, "_smoke.ail"), tmplSmokeAil},
		{filepath.Join(outDir, "README.md"), tmplReadmeMd},
	}

	funcs := template.FuncMap{
		"title": capitalizeFirst, // {{ .ShortName | title }} → "Openkb"
	}

	for _, f := range files {
		if err := renderTemplateToFile(f.path, f.tmpl, data, funcs); err != nil {
			return err
		}
	}
	return nil
}

// renderTemplateToFile parses + renders + writes. Newly-created file is
// always 0o644; we don't preserve any existing perms because the caller
// guarantees the dir didn't exist.
func renderTemplateToFile(path, tmplSrc string, data templateData, funcs template.FuncMap) error {
	t, err := template.New(filepath.Base(path)).Funcs(funcs).Parse(tmplSrc)
	if err != nil {
		return fmt.Errorf("parse template for %s: %w", path, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("render template for %s: %w", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// toQuotedCSV converts ["FS","Process"] → `"FS", "Process"` for ailang.toml.
// Empty list returns empty string (caller handles the default differently).
func toQuotedCSV(items []string) string {
	if len(items) == 0 {
		return ""
	}
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = `"` + s + `"`
	}
	return strings.Join(quoted, ", ")
}

// toAilangStringList converts ["A","B"] → `["A", "B"]` for .ail source.
// Empty list returns `[]`.
func toAilangStringList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = `"` + s + `"`
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// toAilangToolSchemas generates the on_describe_tools body — one ToolSchema
// per declared tool with an empty-but-valid JSON-Schema parameters block.
// Empty `{}` parameters silently degrade against Anthropic typed tool use
// (the bug class M-EXT-AUTHOR-DX M2 closes); the scaffolded
// `{"type":"object","required":[],"properties":{}}` is the minimum valid
// schema, with a TODO comment marking where to fill in real properties.
//
// Output shape (for tools=["A","B"]):
//
//	[
//	    { name: "A", description: "TODO: ...", parameters: "{...empty schema...}" },
//	    { name: "B", description: "TODO: ...", parameters: "{...empty schema...}" }
//	  ]
//
// Empty tools list returns "[]" — no schemas to advertise.
func toAilangToolSchemas(tools []string) string {
	if len(tools) == 0 {
		return "[]"
	}
	const emptyJSONSchema = `"{\"type\":\"object\",\"required\":[],\"properties\":{}}"`
	entries := make([]string, len(tools))
	for i, t := range tools {
		entries[i] = "    { name: \"" + t + "\", description: \"TODO: describe " + t + "\", parameters: " + emptyJSONSchema + " }"
	}
	return "[\n" + strings.Join(entries, ",\n") + "\n  ]"
}

// capitalizeFirst capitalises just the first character of s. Used by the types.ail
// template to spell e.g. "openkb" → "Openkb" for a placeholder type name.
// (Avoids the deprecated strings.Title.)
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ============================================================================
// File templates (Go text/template syntax)
// ============================================================================

const tmplAilangToml = `[package]
name = "{{.Name}}"
version = "0.1.0"
edition = "1"
ailang = ">=0.18.4"
description = "TODO: describe your extension here"
license = "Apache-2.0"

[exports]
modules = [
  "{{.Name}}/register",
  "{{.Name}}/{{.ShortName}}",
  "{{.Name}}/types"
]

[dependencies]
"sunholo/motoko_ext_abi" = "2.2.0"

[effects]
max = [{{.EffectsCSV}}]

[stability]
level = "experimental"
`

const tmplRegisterAil = `-- Canonical entry point for this motoko extension.
-- The registry generator (` + "`ailang generate-extension-registry`" + `) imports
-- ` + "`register_with_config`" + ` from this module and dispatches by short name.
--
-- DO NOT rename ` + "`register_with_config`" + ` — it's the contract.

module {{.Name}}/register

import pkg/sunholo/motoko_ext_abi/types (ExtensionHooks)
import pkg/{{.Name}}/{{.ShortName}} (make_hooks)

-- _cfg has type variable 'a' so this extension is host-config-agnostic.
-- The caller (generated registry) passes its host's RuntimeConfig which
-- unifies with 'a' at the call site.
export func register_with_config(_cfg: a) -> ExtensionHooks ! {Env, FS} {
  make_hooks()
}
`

const tmplTypesAil = `-- Placeholder for your extension's types. Edit me.
--
-- A common pattern: define a result type for your tool's output here, then
-- import it from your impl module ({{.ShortName}}.ail).

module {{.Name}}/types

-- TODO: replace with your real types
export type {{.ShortName | title}}Result = {
  ok: bool,
  message: string
}
`

// tmplImplAil generates the make_hooks function with all 8 ExtensionHooks
// fields populated as no-op defaults. Mirrors motoko-ext-test-dummy's
// register.ail structure exactly. The user replaces no-op handlers with
// real logic incrementally.
const tmplImplAil = `-- {{.ShortName}}: implementation module for this motoko extension.
--
-- ` + "`make_hooks()`" + ` returns an ExtensionHooks record. ALL 8 fields are
-- populated with no-op defaults so the package compiles immediately and
-- you can iterate by replacing fields one at a time.
--
-- Hook-by-hook responsibilities:
--   id                      stable dispatch key (matches the registry short name)
--   provided_tools          tool names this extension contributes (becomes the model's tool catalog entry)
--   on_describe_tools       returns ToolSchema array for each provided tool — fill in when you wire real tools
--   on_build_system_prompt  prepend/append hints to the system prompt
--   on_budget_plan          patch the agent's budget allocation per call
--   on_pre_step             pre-step compaction / message-list rewrite hook (PassThrough = no change)
--   on_tool_policy          Allow / Deny / NoOpinion gate for tool calls
--   on_tool_handle          Handled(result) / Delegate (let host run it) / Continue
--   on_response_intercept   inspect/modify model output before next turn
--   on_solver_candidate     (DP7 contract solver) Accept / ContinueWithFeedback / NoDecision

module {{.Name}}/{{.ShortName}}

import std/option (None)
import pkg/sunholo/motoko_ext_abi/types (
  ExtensionHooks, ExtCtx, Msg,
  BudgetPlan, BudgetPatch, PromptPatch,
  PreStepDecision, PassThrough,
  ToolCallEnvelope, ToolSchema,
  ToolPolicyDecision, ToolHandleDecision,
  ResponseInterceptDecision, FinalizeDecision,
  NoOpinion, Delegate, NoIntercept, NoDecision
)

-- describe_tools advertises one ToolSchema per provided tool. The
-- scaffolded {"type":"object","required":[],"properties":{}} parameters
-- block is the minimum valid JSON-Schema object — replace required +
-- properties with your real schema per tool before publish. Empty
-- properties silently degrade against Anthropic typed tool use, so
-- this is the TODO that matters most for getting typed args from the
-- model.
func describe_tools() -> [ToolSchema] {
  {{.ToolSchemas}}
}

export func make_hooks() -> ExtensionHooks ! {Env, FS} {
  {
    id: "{{.ShortName}}",
    provided_tools: {{.ToolsAilang}},
    on_describe_tools: \_ . describe_tools(),
    on_build_system_prompt:
      \_ctx. { prepend: [], append: [] },
    on_budget_plan:
      func(_ctx: ExtCtx, _plan: BudgetPlan) -> BudgetPatch ! {Env, FS} {
        { requested_total: None, requested_solver: None, requested_verifier: None }
      },
    on_pre_step:
      func(_ctx: ExtCtx, _msgs: [Msg]) -> PreStepDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} {
        PassThrough
      },
    on_tool_policy:
      \_ctx _call. NoOpinion,
    on_tool_handle:
      func(_ctx: ExtCtx, _call: ToolCallEnvelope) -> ToolHandleDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} {
        Delegate
      },
    on_response_intercept:
      func(_ctx: ExtCtx, _resp: string) -> ResponseInterceptDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} {
        NoIntercept
      },
    on_solver_candidate:
      func(_ctx: ExtCtx, _candidate: string) -> FinalizeDecision ! {IO, Process, FS, AI, Env, Net, SharedMem, Clock, Stream} {
        NoDecision
      }
  }
}
`

const tmplReadmeMd = `# {{.Name}}

A motoko_agent extension. Scaffolded with ` + "`ailang init motoko-extension`" + `.

## Status

Experimental — generated from template. Replace the no-op hook implementations in [{{.ShortName}}.ail]({{.ShortName}}.ail) with your real logic.

## Develop

` + "```bash" + `
ailang lock                  # resolve registry deps
ailang check --package .     # type-check every module in this package
` + "```" + `

The package's ` + "`_smoke.ail`" + ` runs in the publish sandbox at ` + "`ailang publish`" + ` time and blocks publish on a panic. Edit it to assert anything that's load-bearing for your extension; drop the ` + "`-- optional`" + ` sections that don't apply.

### Path-dep dev loop (recommended for iterating against a host)

While iterating on this extension against ` + "`motoko_agent`" + ` (or any host that consumes it), use a path-dep in the host's ` + "`ailang.toml`" + ` so you don't have to publish for every change:

` + "```toml" + `
[dependencies]
"{{.Name}}" = { path = "../path/to/this/package" }

[extensions]
packages = [
  "{{.Name}}@0.1.0",   # version still matches [package].version above
]
` + "```" + `

Then from the host: ` + "`ailang lock && ailang generate-extension-registry && make verify_extensions`" + ` (or the host's equivalent). Once the loop closes, switch the host back to the published version pin and publish this package.

## Wire into a host (production, after publish)

In the host's ` + "`ailang.toml`" + `:

` + "```toml" + `
[dependencies]
"{{.Name}}" = "0.1.0"   # registry version

[extensions]
packages = [
  # ... existing entries ...
  "{{.Name}}@0.1.0",
]
` + "```" + `

Then re-lock + regenerate the dispatch:

` + "```bash" + `
ailang lock
ailang generate-extension-registry
` + "```" + `

## Publish to the AILANG registry

When the extension is stable and you want others to consume it:

` + "```bash" + `
ailang publish --dry-run     # tarball + smoke test, no upload
ailang publish               # the real thing (requires AILANG_REGISTRY_API_KEY)
` + "```" + `

### Provider-safe tool naming

Tool names advertised via ` + "`provided_tools`" + ` (and the ` + "`name`" + ` field of ` + "`on_describe_tools`" + `) MUST match ` + "`[A-Za-z0-9_]`" + ` — Anthropic Bedrock + Vertex AI reject names containing ` + "`.`" + `, ` + "`-`" + `, or other characters at the tool-name validator. Use ` + "`ctx_execute`" + ` or ` + "`CtxExecute`" + `, never ` + "`ctx.execute`" + `. ` + "`ailang publish`" + ` enforces this gate; the ` + "`--allow-dotted-tool-names`" + ` flag provides one-cycle migration grace if you're upgrading an older package.

### Publish checklist

- [ ] Bump ` + "`[package].version`" + ` in ` + "`ailang.toml`" + ` (semver: patch for fixes, minor for new tools, major for ExtensionHooks-breaking changes)
- [ ] ` + "`ailang check --package .`" + ` passes
- [ ] ` + "`ailang publish --dry-run`" + ` succeeds (smoke runs in sandbox)
- [ ] ` + "`ailang publish`" + ` (real upload — irreversible)
- [ ] Bump the host's pin: ` + "`\"{{.Name}}\" = \"<new-version>\"`" + ` + matching ` + "`[extensions].packages`" + ` entry
- [ ] Host: ` + "`ailang lock && ailang generate-extension-registry`" + ` + verify extensions boot

See: https://ailang.sunholo.com/docs/guides/package-publishing

## Documentation

- [Build Your First motoko Extension](https://ailang.sunholo.com/docs/guides/build-a-motoko-extension) (tutorial)
- [Motoko Extension Development workflow](https://ailang.sunholo.com/docs/guides/motoko-extension-development) (path-dep dev loop)
- [Extension Packages reference](https://ailang.sunholo.com/docs/guides/extension-packages)
- [Publishing Your Package](https://ailang.sunholo.com/docs/guides/package-publishing)
`

// tmplSmokeAil generates a smoke test template — register/dispatch/policy
// assertion patterns mirroring sunholo/motoko_ext_compaction_ai 0.1.5+ and
// the proven motoko_ext_context_mode 0.2.3 shape. The publish sandbox runs
// `_smoke.ail` automatically; failing it blocks publish. Authors delete
// the `-- optional` sections that don't apply to their extension.
const tmplSmokeAil = `-- Smoke test for {{.Name}}.
--
-- The publish sandbox runs this file with effects [{{.EffectsCSV}}]; a
-- panic blocks publish. Three assertion patterns are scaffolded — keep the
-- ones that apply, drop the rest (marked ` + "`-- optional`" + `).
--
-- Why a smoke? It catches the class of bugs that pass ` + "`ailang check`" + ` but
-- crash at runtime:
--   - register_with_config panics (missing config defaults, wrong ADT pattern)
--   - provided_tools / on_describe_tools mismatch
--   - on_tool_handle policy logic raises an exception on a Bedrock-style call

module {{.Name}}/_smoke

import std/io (println)
import std/list (length)
import pkg/{{.Name}}/register (register_with_config)
import pkg/{{.Name}}/{{.ShortName}} (make_hooks)

export func main() -> () ! {Env, FS, IO} {
  -- Assertion 1: register doesn't panic
  let _ = register_with_config(0);
  println("OK: register_with_config returned without panic");

  -- optional — assertion 2: provided_tools shape (delete if your
  -- extension doesn't advertise any tools yet)
  let hooks = make_hooks();
  let tool_count = length(hooks.provided_tools);
  println("OK: provided_tools returned ${show(tool_count)} entries");

  -- optional — assertion 3: on_describe_tools schema parity (delete if
  -- not advertising tools, or replace with your real schema verification)
  -- NOTE: when you advertise N tools, on_describe_tools should return N
  -- schemas with non-empty {"type":"object",...} parameters. Empty {}
  -- parameters silently degrade against provider-native typed tool use.
  println("OK: smoke complete")
}
`
