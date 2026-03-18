// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/core"
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

	// Track imports
	for _, imp := range spec.Imports {
		g.trackImport(imp)
	}

	// If it has a Helper, ensure the helper is registered for emission and return its name
	if spec.Helper != nil {
		g.registerHelperForEmission(spec.Helper)
		return spec.Helper.FuncName
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
func (g *Generator) registerHelperForEmission(helper *builtins.GoHelperSpec) {
	if g.registryHelpers == nil {
		g.registryHelpers = make(map[string]*builtins.GoHelperSpec)
	}
	g.registryHelpers[helper.FuncName] = helper
}

// writeRegistryHelpers emits all registered runtime helper functions.
// M-CODEGEN-SUSTAINABILITY: Called during runtime generation. Only emits helpers
// that were actually referenced during code generation (lazy emission).
func (g *Generator) writeRegistryHelpers() {
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

	for _, name := range names {
		helper := g.registryHelpers[name]
		// Check if this function was already emitted by the legacy runtime helpers
		// (during migration, some functions exist in both old and new systems)
		if g.emittedHelpers[name] {
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
		g.emittedHelpers[name] = true
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
	return result
}
