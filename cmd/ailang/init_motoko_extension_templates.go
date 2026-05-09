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

// templateData feeds all 5 file templates with deterministic placeholders.
type templateData struct {
	Name        string   // "arniwesth/motoko_ext_openkb"
	ShortName   string   // "openkb"
	Tools       []string // ["OpenKBSearch", "OpenKBList"]
	Effects     []string // ["FS", "Process"]
	EffectsCSV  string   // ailang.toml-friendly CSV: `"FS", "Process"`
	ToolsAilang string   // .ail-friendly list literal: `["OpenKBSearch", "OpenKBList"]`
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
	}

	files := []struct {
		path string
		tmpl string
	}{
		{filepath.Join(outDir, "ailang.toml"), tmplAilangToml},
		{filepath.Join(outDir, "register.ail"), tmplRegisterAil},
		{filepath.Join(outDir, "types.ail"), tmplTypesAil},
		{filepath.Join(outDir, shortName+".ail"), tmplImplAil},
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
"sunholo/motoko_ext_abi" = "1.0.0"

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
--   on_tool_policy          Allow / Deny / NoOpinion gate for tool calls
--   on_tool_handle          Handled(result) / Delegate (let host run it) / Continue
--   on_response_intercept   inspect/modify model output before next turn
--   on_solver_candidate     (DP7 contract solver) Accept / ContinueWithFeedback / NoDecision

module {{.Name}}/{{.ShortName}}

import std/option (None)
import pkg/sunholo/motoko_ext_abi/types (
  ExtensionHooks, ExtCtx,
  BudgetPlan, BudgetPatch, PromptPatch,
  ToolCallEnvelope,
  ToolPolicyDecision, ToolHandleDecision,
  ResponseInterceptDecision, FinalizeDecision,
  NoOpinion, Delegate, NoIntercept, NoDecision
)

export func make_hooks() -> ExtensionHooks ! {Env, FS} {
  {
    id: "{{.ShortName}}",
    provided_tools: {{.ToolsAilang}},
    on_describe_tools: \_ . [],
    on_build_system_prompt:
      \_ctx. { prepend: [], append: [] },
    on_budget_plan:
      func(_ctx: ExtCtx, _plan: BudgetPlan) -> BudgetPatch ! {Env, FS} {
        { requested_total: None, requested_solver: None, requested_verifier: None }
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
ailang lock                 # resolve registry deps
ailang check register.ail   # type-check the package
` + "```" + `

## Wire into a host (e.g. motoko_agent)

In the host's ` + "`ailang.toml`" + `:

` + "```toml" + `
[dependencies]
"{{.Name}}" = "0.1.0"   # registry version, NOT path-based for production

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
export AILANG_REGISTRY_API_KEY=<your-key>
ailang publish
` + "```" + `

See: https://ailang.sunholo.com/docs/guides/package-publishing

## Documentation

- [Build Your First motoko Extension](https://ailang.sunholo.com/docs/guides/build-a-motoko-extension) (tutorial)
- [Extension Packages reference](https://ailang.sunholo.com/docs/guides/extension-packages)
- [Publishing Your Package](https://ailang.sunholo.com/docs/guides/package-publishing)
`
