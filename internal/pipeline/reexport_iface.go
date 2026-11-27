package pipeline

import (
	"fmt"
	"maps"
	"slices"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/types"
)

// attachReexports records a facade's `export import` names on its interface
// (M-PKG-PARALLEL-AGENTS C1). Each entry is copied from the module the line
// names, whose interface is already built (dependencies compile first) and,
// when that module is itself a facade, already carries its own re-exports, so
// chains resolve one hop at a time. A name the source does not have is
// skipped: the facade's own import of it already failed with IMP010.
func attachReexports(ifc *iface.Iface, rx map[string]loader.Reexport, getIface func(string) *iface.Iface) {
	if len(rx) == 0 {
		return
	}
	out := iface.NewReexports(ifc.Module)
	for _, local := range slices.Sorted(maps.Keys(rx)) {
		r := rx[local]
		src := getIface(r.From)
		if src == nil {
			continue
		}
		if copyReexport(out.Items, src, local, r.Orig) {
			out.Origins[local] = reexportOrigin(src, r)
		}
	}
	if len(out.Origins) > 0 {
		ifc.Reexported = out
	}
}

// copyReexport copies src's entries named orig into items under local. It
// reports whether src had any.
func copyReexport(items, src *iface.Iface, local, orig string) bool {
	fn, typ, ctor, typeCtors := src.Lookup(orig)
	if fn != nil {
		c := *fn
		c.Name = local
		items.Exports[local] = &c
	}
	if typ != nil {
		items.Types[local] = &iface.TypeExport{Name: local, Arity: typ.Arity}
		if body, params, ok := src.LookupAlias(orig); ok {
			items.TypeAliases[local] = body
			if params != nil {
				items.AliasParams[local] = params
			}
		}
		// A re-exported type brings its constructors, as importing it does.
		for name, c := range typeCtors {
			cc := *c
			cc.TypeName = local
			if _, explicit := items.Constructors[name]; !explicit {
				items.Constructors[name] = &cc
			}
		}
	}
	if ctor != nil {
		cc := *ctor
		items.Constructors[local] = &cc
		// A constructor is rendered under its type; name the type even when the
		// line re-exports only the constructor.
		if _, ok := items.Types[cc.TypeName]; !ok {
			if _, t, _, _ := src.Lookup(cc.TypeName); t != nil {
				items.Types[cc.TypeName] = &iface.TypeExport{Name: cc.TypeName, Arity: t.Arity}
			}
		}
	}
	return fn != nil || typ != nil || ctor != nil
}

// reexportOrigin is the definition r stands for: src's own entry, or, when
// src re-exports it too, src's origin for it.
func reexportOrigin(src *iface.Iface, r loader.Reexport) iface.ReexportOrigin {
	o := iface.ReexportOrigin{Module: src.Module, Name: r.Orig, MovedTo: r.MovedTo}
	if src.Reexported != nil {
		if up, ok := src.Reexported.Origins[r.Orig]; ok {
			o.Module, o.Name = up.Module, up.Name
			if o.MovedTo == "" {
				o.MovedTo = up.MovedTo
			}
		}
	}
	return o
}

// bindAliasedReexports binds `Alias.name` for each name a facade re-exports,
// with the origin's identity: a function's global ref is already the origin's,
// and a type resolves to its defining module, never the facade.
func bindAliasedReexports(imports *moduleImports, alias string, dep *iface.Iface) {
	rx := dep.Reexported
	if rx == nil {
		return
	}
	for _, name := range slices.Sorted(maps.Keys(rx.Origins)) {
		o := rx.Origins[name]
		qualified := alias + "." + name
		if item := rx.Items.Exports[name]; item != nil {
			key := item.Ref.Module + "." + item.Ref.Name
			imports.ExternalTypes[key] = item.Type
			imports.GlobalRefs[qualified] = item.Ref
			imports.IFC.Add(qualified, key, item.IFC)
		}
		if rx.Items.Types[name] != nil {
			imports.ExplicitTypes[qualified] = types.TypeRef{Module: o.Module, Name: o.Name}
		}
	}
}

// validateReexportCeiling holds a facade to its package's [effects].max for
// the functions it re-exports (M-PKG-PARALLEL-AGENTS C1, TST-PKGPAR-16): a
// package's advertised ceiling covers everything it exports, re-exports
// included. validateEffectCeiling cannot see them, because a re-export has no
// declaration or Core in the facade, so this reads the facade's interface.
// Importers are protected either way: their own reachability check walks
// into the origin function.
func validateReexportCeiling(ifc *iface.Iface, modID string) error {
	if currentPackageManifest == nil || ifc == nil || ifc.Reexported == nil {
		return nil
	}
	maxEffects := currentPackageManifest.Effects.Max
	pkgName := currentPackageManifest.Package.Name
	if maxEffects == nil || !isOwnPackageModule(modID, pkgName) {
		return nil
	}
	rx := ifc.Reexported
	for _, name := range slices.Sorted(maps.Keys(rx.Items.Exports)) {
		item := rx.Items.Exports[name]
		if item.Type == nil {
			continue
		}
		if err := pkg.CheckEffectCeiling(pkgName, typeEffectLabels(item.Type.Type), maxEffects); err != nil {
			o := rx.Origins[name]
			return fmt.Errorf("in re-exported function %s in %s (defined in %s as %s): %w", name, modID, o.Module, o.Name, err)
		}
	}
	return nil
}
