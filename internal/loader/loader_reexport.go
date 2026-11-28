package loader

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/errors"
)

// Re-exports (M-PKG-PARALLEL-AGENTS C1).
//
// `export import lib/color (Color, Red)` makes lib/facade a facade: Color and
// Red are its exports with lib/color's identity. Identity is kept by rewriting
// at load time rather than aliasing in every later stage: once LoadAll has
// loaded every module, each selective import of a re-exported name is
// rewritten to the name's origin module, following chains of facades. The
// typechecker, linker, elaborator, effect reachability and both engines then
// see the origin, so a constructor imported through a facade IS the origin's.
//
// The importer keeps the facade in LoadedModule.Imports, so the facade stays
// in its compile-cache closure: dropping a re-export changes the facade's
// interface and misses the importer's cache entry.

// Reexport is one name a facade re-exports.
type Reexport struct {
	From    string // canonical module the facade imports it from
	Orig    string // the name in From
	MovedTo string // the line's @deprecated("path"), if any
}

// buildReexports collects a parsed module's `export import` names, keyed by
// the name the facade exports (its alias, when the line renames). Import
// paths are canonical by the time Load calls it.
func buildReexports(file *ast.File) map[string]Reexport {
	if file == nil {
		return nil
	}
	var out map[string]Reexport
	for _, imp := range file.Imports {
		if imp == nil || !imp.Reexport {
			continue
		}
		if out == nil {
			out = make(map[string]Reexport)
		}
		for _, sym := range imp.Symbols {
			local := sym
			if alias := imp.SymbolAliases[sym]; alias != "" {
				local = alias
			}
			out[local] = Reexport{From: imp.Path, Orig: sym, MovedTo: imp.MovedTo}
		}
	}
	return out
}

// resolveReexports rewrites every loaded module's imports of re-exported
// names to their origins. Modules are visited in sorted order so errors are
// deterministic.
func resolveReexports(modules map[string]*LoadedModule) error {
	ids := make([]string, 0, len(modules))
	for id := range modules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		mod := modules[id]
		if mod == nil || mod.File == nil {
			continue
		}
		var out []*ast.ImportDecl
		changed := false
		for _, imp := range mod.File.Imports {
			split, err := splitReexportImport(modules, imp)
			if err != nil {
				return err
			}
			changed = changed || len(split) != 1 || split[0] != imp
			out = append(out, split...)
		}
		if changed {
			mod.File.Imports = out
		}
	}
	return nil
}

// splitReexportImport returns imp unchanged when it names no re-exported
// symbol. Otherwise it returns imp with those symbols removed (dropped
// entirely when nothing else is left and it has no module alias) followed by
// one import per origin, in first-appearance order.
func splitReexportImport(modules map[string]*LoadedModule, imp *ast.ImportDecl) ([]*ast.ImportDecl, error) {
	if imp == nil || len(imp.Symbols) == 0 {
		return []*ast.ImportDecl{imp}, nil
	}
	target := lookupModule(modules, imp.Path)
	if target == nil || len(target.Reexports) == 0 {
		return []*ast.ImportDecl{imp}, nil
	}
	kept := *imp
	kept.Symbols, kept.SymbolPos, kept.SymbolAliases = nil, nil, map[string]string{}
	var origins []*ast.ImportDecl
	byKey := map[string]*ast.ImportDecl{}
	for i, sym := range imp.Symbols {
		origin, orig, movedTo, ok, err := resolveReexportChain(modules, imp.Path, sym)
		if err != nil {
			return nil, err
		}
		if !ok {
			kept.Symbols = append(kept.Symbols, sym)
			if i < len(imp.SymbolPos) {
				kept.SymbolPos = append(kept.SymbolPos, imp.SymbolPos[i])
			}
			if alias, has := imp.SymbolAliases[sym]; has {
				kept.SymbolAliases[sym] = alias
			}
			continue
		}
		key := origin + "\x00" + movedTo
		d := byKey[key]
		if d == nil {
			d = rewrittenImport(imp, origin, movedTo)
			byKey[key] = d
			origins = append(origins, d)
		}
		local := sym
		if alias := imp.SymbolAliases[sym]; alias != "" {
			local = alias
		}
		d.Symbols = append(d.Symbols, orig)
		if i < len(imp.SymbolPos) {
			d.SymbolPos = append(d.SymbolPos, imp.SymbolPos[i])
		}
		if local != orig {
			d.SymbolAliases[orig] = local
		}
	}
	if len(origins) == 0 {
		return []*ast.ImportDecl{imp}, nil
	}
	var out []*ast.ImportDecl
	if len(kept.Symbols) > 0 || kept.ModuleAlias != "" {
		out = append(out, &kept)
	}
	return append(out, origins...), nil
}

// rewrittenImport is an import of origin that stands in for part of imp,
// remembering the facade it went through.
func rewrittenImport(imp *ast.ImportDecl, origin, movedTo string) *ast.ImportDecl {
	d := &ast.ImportDecl{
		Path:          origin,
		SymbolAliases: map[string]string{},
		Reexport:      imp.Reexport,
		MovedTo:       imp.MovedTo,
		ViaModule:     imp.Path,
		ViaMovedTo:    movedTo,
		Implicit:      imp.Implicit,
		Pos:           imp.Pos,
		Span:          imp.Span,
	}
	if strings.HasPrefix(origin, "pkg/") {
		d.IsPackage = true
		if parts := strings.SplitN(strings.TrimPrefix(origin, "pkg/"), "/", 3); len(parts) >= 2 {
			d.PackageName = parts[0] + "/" + parts[1]
		}
	}
	return d
}

// resolveReexportChain follows name through facades starting at path. ok is
// false when path does not re-export name. movedTo is the first @deprecated
// path met on the chain. A chain that revisits a name is MOD016.
func resolveReexportChain(modules map[string]*LoadedModule, path, name string) (origin, orig, movedTo string, ok bool, err error) {
	origin, orig = path, name
	seen := map[string]bool{}
	var hops []string
	for {
		mod := lookupModule(modules, origin)
		if mod == nil {
			return origin, orig, movedTo, ok, nil
		}
		r, has := mod.Reexports[orig]
		if !has {
			return origin, orig, movedTo, ok, nil
		}
		hop := origin + "." + orig
		hops = append(hops, hop)
		if seen[hop] {
			return "", "", "", false, reexportCycle(hops)
		}
		seen[hop] = true
		if movedTo == "" {
			movedTo = r.MovedTo
		}
		origin, orig, ok = r.From, r.Orig, true
	}
}

func reexportCycle(hops []string) error {
	chain := strings.Join(hops, " -> ")
	return errors.WrapReport(&errors.Report{
		Schema:  "ailang.error/v1",
		Code:    errors.MOD016,
		Phase:   "loader",
		Message: fmt.Sprintf("re-export cycle: %s", chain),
		Data:    map[string]any{"cycle": hops},
		Fix: &errors.Fix{
			Suggestion: "Re-export each name from the module that defines it; an `export import` chain must end at a definition",
			Confidence: 0.9,
		},
	})
}

// lookupModule finds a loaded module by canonical import path.
func lookupModule(modules map[string]*LoadedModule, path string) *LoadedModule {
	if m := modules[path]; m != nil {
		return m
	}
	return modules[CanonicalModuleID(path)]
}
