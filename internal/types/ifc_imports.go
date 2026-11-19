package types

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_imports.go — imported IFC contracts at the caller (M-IFC-CROSS-MODULE M3,
// #1134).
//
// A library's sink refinements ({not ℓ}), positive parameter labels (<ℓ>),
// declared/intrinsic return labels and Declassify authority travel in its
// interface as an ImportedIFCSig per export (M1/M2). The caller's surface walk
// reads them through IFCImports, so an imported callee is checked exactly like a
// local one: Check A and Check C run at every application of the imported
// function value, and its result label is the library's declared return (or,
// when transparent, its intrinsic body label joined with the arguments).

// IFCImports is the caller's resolved import identity map. Keys are the names a
// surface expression uses for an imported value: a selective import (`gate`),
// a renamed one (`g` for `gate as g`) and a module-alias qualified name
// (`L.gate` for `import lib as L`). The pipeline and REPL build it from the same
// resolution that binds those names for elaboration; the checker never guesses
// identities from spelling.
type IFCImports struct {
	Sigs  map[string]*ImportedIFCSig // local name -> exporting module's summary
	Names map[string]string          // local name -> canonical "module.symbol", for diagnostics
}

// Add records one resolved imported binding.
func (im *IFCImports) Add(local, canonical string, sig *ImportedIFCSig) {
	if im.Sigs == nil {
		im.Sigs = map[string]*ImportedIFCSig{}
		im.Names = map[string]string{}
	}
	im.Sigs[local] = sig
	im.Names[local] = canonical
}

// validate fails closed on missing or malformed metadata: an import without a
// summary is never treated as clean (no silent widening).
func (im IFCImports) validate() *TypeCheckError {
	names := make([]string, 0, len(im.Sigs))
	for name := range im.Sigs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := im.Sigs[name].Validate(); err != nil {
			return &TypeCheckError{
				Kind:       IFCAnalysisLimitError,
				Message:    fmt.Sprintf("information-flow metadata for imported %s is unusable: %v", im.canonical(name), err),
				Suggestion: "recompile the imported module from source (clear its cached interface) so its IFC summary is regenerated",
			}
		}
	}
	return nil
}

func (im IFCImports) canonical(local string) string {
	if n := im.Names[local]; n != "" {
		return n
	}
	return local
}

// hasRequirements reports whether any imported summary carries a demand
// (refinement or positive label) or a label source. Such an import can make an
// otherwise unannotated module's calls ill-flowing, so the module's local
// summaries must be built (moduleHasIFCRequirements alone would skip them).
func (im IFCImports) hasRequirements() bool {
	for _, s := range im.Sigs {
		if s != nil && !s.IsUnlabelled() {
			return true
		}
	}
	return false
}

// CheckModuleIFCWithImports is CheckModuleIFC with the caller's imported IFC
// contracts in scope. Production gates (pipeline, REPL) call this form.
func CheckModuleIFCWithImports(file *ast.File, imports IFCImports) []*TypeCheckError {
	if file == nil {
		return nil
	}
	if err := imports.validate(); err != nil {
		return []*TypeCheckError{err}
	}
	return guardIFCAnalysis(func() []*TypeCheckError {
		c := newIFCChecker(file)
		c.useImports(imports)
		return c.run(file)
	})
}

// useImports puts the caller's resolved imported contracts in scope.
func (c *ifcChecker) useImports(imports IFCImports) {
	c.imports = imports.Sigs
	c.importNames = imports.Names
	c.importDemands = imports.hasRequirements()
}

// importedFn is the callable of an imported function value: an opaque
// alternative (its body is not visible, so callable arguments still take the
// conservative fallback) that carries the library's declared demands.
func (c *ifcChecker) importedFn(local string, s *ImportedIFCSig) *ifcFn {
	name := local
	if canonical := c.importNames[local]; canonical != "" {
		name = "imported function " + canonical
	}
	f := &ifcFn{kind: fnOpaque, name: name, imported: s}
	for _, p := range s.Params {
		d := ifcDemand{param: p.Name, forbid: append([]string(nil), p.Forbidden...)}
		if l := p.Label.Decode(); !LabelEqual(l, LabelBottom()) {
			d.covers = []Label{l}
		}
		f.params = append(f.params, canonicalDemand(d))
	}
	return f
}

// qualifiedImport resolves `Alias.name` written as a record access on a module
// alias. A lexical binding named like the alias is a record, not the module
// (the elaborator's rule, #1467).
func (c *ifcChecker) qualifiedImport(e *ast.RecordAccess, env ifcEnv) (string, *ImportedIFCSig) {
	id, ok := e.Record.(*ast.Identifier)
	if !ok || len(c.imports) == 0 {
		return "", nil
	}
	if _, bound := env[id.Name]; bound {
		return "", nil
	}
	name := id.Name + "." + e.Field
	return name, c.imports[name]
}
