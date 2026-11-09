package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_imports.go — enforcement at IMPORTED call sites (M-IFC-CROSS-MODULE M3,
// #1134; design_docs/implemented/v0_57_5/m-ifc-cross-module-labels.md).
//
// M1–M2 export an IFC summary (ImportedIFCSig) for every value a module
// exports. The importer's checker reads them here: an imported function
// referenced by the name the importer binds it under becomes a callable that
// carries the export's declared demands — {not ℓ} refinements (Check A) and
// positive parameter labels (Check C) — and the export's result label (its
// declared return, its Declassify authority, or its intrinsic body label
// joined with the arguments). Every route the callable domain already follows
// (aliases, higher-order parameters, wrappers, records) therefore applies to
// imported functions exactly as to local ones.
//
// Identity: the caller (pipeline, REPL) resolves names exactly as the
// elaborator does — selective imports under their bound (possibly renamed)
// name, qualified module-alias access as "Alias.name". Lexical bindings and
// module declarations shadow an import (identVal order).

// IFCImport is one imported value as the importing module spells it.
type IFCImport struct {
	Module string          // defining module path, e.g. "std/secret"
	Name   string          // export name in the defining module
	Sig    *ImportedIFCSig // that export's IFC summary
}

// display is the canonical module.name identity named in diagnostics.
func (i IFCImport) display() string {
	if i.Module == "" {
		return i.Name
	}
	return i.Module + "." + i.Name
}

// IFCImports maps the name a module binds (a selective import's local name,
// or "Alias.name" for a module alias) to the imported value's summary.
type IFCImports map[string]IFCImport

// Sigs projects the summaries keyed by bound name (the iface builder's input).
func (m IFCImports) Sigs() map[string]*ImportedIFCSig {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]*ImportedIFCSig, len(m))
	for name, imp := range m {
		out[name] = imp.Sig
	}
	return out
}

// Validate rejects a missing or malformed summary: an import without IFC
// metadata must never be treated as clean (no silent widening).
func (m IFCImports) Validate() error {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := m[name].Sig.Validate(); err != nil {
			return fmt.Errorf("imported %s (as %s): %w", m[name].display(), name, err)
		}
	}
	return nil
}

// CheckModuleIFCWithImports is CheckModuleIFC for a module whose imported
// functions' IFC summaries are known: calls to them are checked against their
// exported demands and take their exported result labels.
func CheckModuleIFCWithImports(file *ast.File, imports IFCImports) []*TypeCheckError {
	errs, _ := CheckModuleIFCWithTraceSites(file, imports)
	return errs
}

// setImports installs validated import summaries on a checker.
func (c *ifcChecker) setImports(imports IFCImports) {
	if len(imports) == 0 {
		return
	}
	c.imports = imports.Sigs()
	c.importNames = make(map[string]string, len(imports))
	for name, imp := range imports {
		c.importNames[name] = imp.display()
	}
}

// importedName is the diagnostic identity of an imported binding.
func (c *ifcChecker) importedName(bound string) string {
	if n, ok := c.importNames[bound]; ok {
		return n
	}
	return bound
}

// qualifiedImport resolves `Alias.name` to an imported binding when Alias is
// neither a lexical binding nor a module declaration (it then names a module,
// as in the elaborator's normalizeRecordAccess, #1467).
func (c *ifcChecker) qualifiedImport(e *ast.RecordAccess, env ifcEnv) (string, bool) {
	id, ok := e.Record.(*ast.Identifier)
	if !ok || len(c.imports) == 0 {
		return "", false
	}
	if _, bound := env[id.Name]; bound {
		return "", false
	}
	if _, local := c.sigs[id.Name]; local {
		return "", false
	}
	name := id.Name + "." + e.Field
	_, ok = c.imports[name]
	return name, ok
}

// importedFn is the callable of an imported function value: an opaque
// callee (callable arguments take the F2 fallback) that also carries the
// export's declared parameter demands.
func importedFn(name string, s *ImportedIFCSig) *ifcFn {
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

// importedDemandErrors reports Check A / Check C at an edge into an imported
// function. It names the caller location, the canonical imported callee and
// the parameter, so the fix needs no trip into the library's source. A direct
// call by the imported name is not repeated as a via trail; any other route
// (an alias, a higher-order parameter) is.
func (c *ifcChecker) importedDemandErrors(call *ast.FuncCall, callee string, d ifcDemand, arg Label, via []string) []*TypeCheckError {
	route := ""
	if len(via) > 0 && (len(via) != 1 || c.imports[via[0]] == nil) {
		route = " (via " + strings.Join(via, " → ") + ")"
	}
	edge := fmt.Sprintf("edge: %s → %s(%s)", call.Pos, callee, d.param)
	var out []*TypeCheckError
	for _, cv := range d.covers {
		if labelNotCoveredBy(arg, cv) == "" {
			continue
		}
		out = append(out, &TypeCheckError{
			Kind:     ParamLabelCoverError,
			Position: call.Pos.String(),
			Message: fmt.Sprintf("information-flow violation: value labelled %s reaches parameter %q of imported function %s%s declared %s, which does not cover it",
				arg, d.param, callee, route, cv),
			Suggestion: edge + " — relabel the value explicitly through a scoped declassifier, or widen the library parameter's declared label",
		})
	}
	for _, f := range d.forbid {
		if CheckSinkLabel(arg, f) == nil {
			continue
		}
		out = append(out, &TypeCheckError{
			Kind:     SinkRefinementError,
			Position: call.Pos.String(),
			Message: fmt.Sprintf("information-flow violation: value labelled %s reaches parameter %q of imported function %s%s, which forbids it (string{not %s})",
				arg, d.param, callee, route, f),
			Suggestion: edge + " — declassify the value through a function declaring ! {Declassify}, or do not pass it across this edge",
		})
	}
	return out
}
