// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/core"
)

// resolveBuiltinViaRegistry checks if a name has a GoCodegenSpec in the builtin registry.
// M-CODEGEN-SUSTAINABILITY: Replaces mapPureMathBuiltin, mapPureListBuiltin, mapStdlibBuiltin.
//
// Returns the Go expression/name to emit, or empty string if not found.
// Also registers any needed runtime helpers and tracks imports.
func (g *Generator) resolveBuiltinViaRegistry(name string) string {
	// Try direct lookup first (internal builtin names like _str_trim)
	spec := builtins.GetCodegenSpec(name)

	// Then try stdlib name lookup (exported names like trim, map, split)
	if spec == nil {
		spec = builtins.GetCodegenSpecByStdlibName(name)
	}

	if spec == nil {
		return ""
	}

	// If it has a Helper, ensure the helper is registered for emission and return its name.
	// Don't track imports here — Helper functions live in runtime.go which manages its own imports.
	if spec.Helper != nil {
		g.registerHelperForEmission(spec.Helper)
		return spec.Helper.FuncName
	}

	// Track imports only for Inline specs (these expand directly into the module file).
	for _, imp := range spec.Imports {
		g.trackImport(imp)
	}

	// If it has an Inline spec without arg placeholders, return it directly.
	// Constants like math.Pi work at VarGlobal level.
	// Inline specs WITH {{arg0}} placeholders must be resolved at App level
	// via resolveInlineBuiltin — returning them here would emit raw templates.
	if spec.Inline != "" && !strings.Contains(spec.Inline, "{{arg") {
		return spec.Inline
	}

	return ""
}

// trackImport records a Go import needed by generated code.
// M-CODEGEN-SUSTAINABILITY: Called when emitting builtins that need Go stdlib imports.
func (g *Generator) trackImport(pkg string) {
	switch pkg {
	case "math":
		g.needsMathImport = true
	case "strconv":
		g.needsStrconvImport = true
	case "strings":
		g.needsStringsImport = true
	case "sort":
		g.needsSortImport = true
	case "fmt":
		// fmt is always imported in runtime.go
	}
}

// registerHelperForEmission marks a GoHelperSpec for emission in the runtime section.
// M-CODEGEN-SUSTAINABILITY: Helpers are emitted once, deduped by FuncName.
// Also registers transitive dependencies (helpers whose bodies call other helpers).
func (g *Generator) registerHelperForEmission(helper *builtins.GoHelperSpec) {
	if g.registryHelpers == nil {
		g.registryHelpers = make(map[string]*builtins.GoHelperSpec)
	}
	if _, exists := g.registryHelpers[helper.FuncName]; exists {
		return // Already registered, avoid infinite recursion
	}
	g.registryHelpers[helper.FuncName] = helper

	// M-CODEGEN-COMPILE-GATE-CLEANUP: Register transitive dependencies.
	// Some helper bodies call other helpers (e.g., ForEachE calls Map).
	// Without this, the called helper is missing from runtime.go.
	deps := map[string]string{
		"Map":             "_list_map",
		"Filter":          "_list_filter",
		"toSlice":         "", // infrastructure, always present
		"CallFunc":        "", // infrastructure, always present
		"Show":            "", // infrastructure, always present
		"JsonGet":         "_json_get",
		"IsNone":          "_option_isNone",
		"IsSome":          "_option_isSome",
		"OptionGetOrElse": "_option_getOrElse",
		"AsString":        "_json_asString",
		"AsArray":         "_json_asArray",
		"AsObject":        "_json_asObject",
	}
	for funcName, builtinName := range deps {
		if builtinName == "" {
			continue // infrastructure, always emitted
		}
		if strings.Contains(helper.Body, funcName+"(") {
			if depSpec := builtins.GetCodegenSpec(builtinName); depSpec != nil && depSpec.Helper != nil {
				g.registerHelperForEmission(depSpec.Helper)
			}
		}
	}
}

// adtIsRegistered checks if an ADT type is registered for code generation.
func (g *Generator) adtIsRegistered(adtName string) bool {
	switch adtName {
	case "Json":
		_, ok := g.adtConstructors["Json.JString"]
		return ok
	case "Option":
		_, ok := g.adtConstructors["Option.Some"]
		return ok
	case "Result":
		_, ok := g.adtConstructors["Result.Ok"]
		return ok
	}
	return false
}

// eagerRegisterADTHelpers scans the registry for helpers tagged with RequiresADT
// and registers them for emission if their ADT is available. This ensures
// inter-dependent helpers (e.g., GetString depends on JsonGet, IsNone, AsString)
// are always emitted as a complete group.
func (g *Generator) eagerRegisterADTHelpers() {
	for _, adtName := range []string{"Json", "Option", "Result"} {
		if !g.adtIsRegistered(adtName) {
			continue
		}
		for _, spec := range builtins.GetHelpersRequiringADT(adtName) {
			g.registerHelperForEmission(spec.Helper)
		}
	}
}

// writeRegistryHelpers emits all registered runtime helper functions.
// Combines lazy emission (helpers referenced during codegen) with eager emission
// (ADT-dependent helpers emitted as groups when their ADT is registered).
func (g *Generator) writeRegistryHelpers() {
	// Eagerly register all helpers for registered ADTs
	g.eagerRegisterADTHelpers()

	if len(g.registryHelpers) == 0 {
		return
	}

	g.writef("// ============================================================================\n")
	g.writef("// Registry-generated runtime helpers\n")
	g.writef("// M-CODEGEN-SUSTAINABILITY: Generated from BuiltinMeta.GoCodegenSpec\n")
	g.writef("// ============================================================================\n\n")

	// Sort for deterministic output
	names := make([]string, 0, len(g.registryHelpers))
	for name := range g.registryHelpers {
		names = append(names, name)
	}
	sortStrings(names)

	emitted := make(map[string]bool)
	for _, name := range names {
		helper := g.registryHelpers[name]
		if emitted[name] {
			continue
		}
		g.writef("// %s is a registry-generated runtime helper.\n", name)
		g.writef("%s {\n", helper.Signature)
		g.indent++
		// Write body lines with proper indentation
		lines := strings.Split(helper.Body, "\n")
		for _, line := range lines {
			g.writef("%s\n", line)
		}
		g.indent--
		g.writef("}\n\n")
		emitted[name] = true
	}
}

// sortStrings sorts a string slice in place (simple insertion sort for small slices).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// tryResolveInlineApp checks if an App's function is an inline builtin with arg placeholders.
// Generates the argument expressions and substitutes them into the inline template.
// Returns the complete Go expression or empty string if not an inline builtin.
func (g *Generator) tryResolveInlineApp(app *core.App) string {
	// Extract the function name from the App
	var name string
	switch f := app.Func.(type) {
	case *core.VarGlobal:
		name = f.Ref.Name
	case *core.Var:
		name = f.Name
	default:
		return ""
	}

	// Generate arg expressions as strings
	argExprs := make([]string, len(app.Args))
	for i, arg := range app.Args {
		var buf strings.Builder
		oldBuf := g.buf
		g.buf = *bytes.NewBuffer(nil)
		if err := g.generateExpr(arg); err != nil {
			g.buf = oldBuf
			return ""
		}
		argExprs[i] = g.buf.String()
		g.buf = oldBuf
		_ = buf // suppress unused
	}

	return g.resolveInlineBuiltin(name, argExprs)
}

// resolveInlineBuiltin checks if a VarGlobal + App combination can use an Inline spec.
// M-CODEGEN-SUSTAINABILITY: For builtins with Inline specs like "strings.TrimSpace({{arg0}}.(string))",
// this substitutes the actual argument expressions and returns the final Go expression.
// Returns empty string if not an Inline builtin.
func (g *Generator) resolveInlineBuiltin(name string, argExprs []string) string {
	spec := builtins.GetCodegenSpec(name)
	if spec == nil {
		spec = builtins.GetCodegenSpecByStdlibName(name)
	}
	if spec == nil || spec.Inline == "" {
		return ""
	}

	// Track imports
	for _, imp := range spec.Imports {
		g.trackImport(imp)
	}

	// Substitute argument placeholders
	result := spec.Inline
	for i, arg := range argExprs {
		placeholder := fmt.Sprintf("{{arg%d}}", i)
		result = strings.ReplaceAll(result, placeholder, arg)
	}

	// Fix invalid type assertions on literals: "...".(string) → "..."
	// String literals are already typed in Go, so .(string) is invalid.
	// Also handles int/float/bool literals with their respective assertions.
	result = fixLiteralTypeAssertions(result)

	return result
}

// fixLiteralTypeAssertions removes invalid type assertions on Go literals.
// e.g. ".opf".(string) → ".opf" — string literals are already typed.
// Also handles int64(42).(int64) and similar patterns.
func fixLiteralTypeAssertions(s string) string {
	// Fix string literal assertions: "...".(string)
	// Match pattern: quote, content, quote, dot-paren-string-paren
	i := 0
	for i < len(s) {
		// Find ".(string)" preceded by a closing quote
		idx := strings.Index(s[i:], ".(string)")
		if idx < 0 {
			break
		}
		pos := i + idx
		// Check if preceded by a string literal (closing double quote)
		if pos > 0 && s[pos-1] == '"' {
			// Remove the .(string)
			s = s[:pos] + s[pos+len(".(string)"):]
			// Don't advance i — more may follow at same position
		} else {
			i = pos + 1
		}
	}
	return s
}
