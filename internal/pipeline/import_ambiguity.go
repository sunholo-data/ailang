package pipeline

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-NOMINAL-TYPE-IDENTITY: a bare type or constructor name must mean the
// same thing whatever order the imports are written in. Two cases used to be
// decided by "first import wins":
//
//   - two explicit selective imports binding one bare type or constructor name
//     to different modules (`import a (T)` + `import b (T)`) — now MOD015 at
//     the second import;
//   - a constructor auto-imported (M-CTOR-AUTO) from two modules — now
//     resolved by the scrutinee's type in a pattern, and TC_AMBIGUOUS_CTOR_001
//     where nothing decides it.

// importSite is where an explicit selective import bound a name.
type importSite struct {
	module string
	sym    string
	decl   *ast.ImportDecl
}

// ambiguousCtorCandidate is one module offering an auto-imported constructor.
type ambiguousCtorCandidate struct {
	module string
	ctor   *iface.ConstructorScheme
}

// importNamespaces tracks what each import binds, for the checks above.
type importNamespaces struct {
	explicitTypes map[string]importSite
	explicitCtors map[string]importSite
	autoCtors     map[string][]ambiguousCtorCandidate
	collisions    []error
}

func newImportNamespaces() *importNamespaces {
	return &importNamespaces{
		explicitTypes: make(map[string]importSite),
		explicitCtors: make(map[string]importSite),
		autoCtors:     make(map[string][]ambiguousCtorCandidate),
	}
}

// bindExplicit records an explicit binding and reports MOD015 when the same
// bare name is already bound to a different module's declaration.
func (n *importNamespaces) bindExplicit(table map[string]importSite, kind, name string, site importSite, file *ast.File) {
	prev, ok := table[name]
	if !ok {
		table[name] = site
		return
	}
	if prev.module == site.module && prev.sym == site.sym {
		return
	}
	types.ObserveAmbiguity(types.AmbiguityEvent{Kind: "explicit-dup", Name: name,
		Candidates: []types.TypeRef{{Module: prev.module, Name: prev.sym}, {Module: site.module, Name: site.sym}}})
	if !types.AmbiguityIsError() {
		return
	}
	// Name both in module order, so the text does not depend on which import
	// was written first (only the positions do).
	first, second := prev, site
	if second.module < first.module {
		first, second = second, first
	}
	n.collisions = append(n.collisions, elaborate.ImportImportCollision(file, kind, name,
		elaborate.ImportSite{Path: first.decl.Path, Sym: first.sym, Pos: first.decl.Pos},
		elaborate.ImportSite{Path: second.decl.Path, Sym: second.sym, Pos: second.decl.Pos}))
}

// addAutoCtor records that an import offers constructor ctor of module.
func (n *importNamespaces) addAutoCtor(module string, ctor *iface.ConstructorScheme) {
	for _, c := range n.autoCtors[ctor.CtorName] {
		if c.module == module {
			return
		}
	}
	n.autoCtors[ctor.CtorName] = append(n.autoCtors[ctor.CtorName], ambiguousCtorCandidate{module, ctor})
}

// finishAutoCtors marks every auto-imported constructor name offered by two or
// more modules (and not imported by name) as ambiguous: the module's patterns
// resolve it by the scrutinee's type, an expression using it is an error, and
// no candidate is silently picked. Every candidate's factory type is in scope
// under its own module-qualified key.
func (n *importNamespaces) finishAutoCtors(imports *moduleImports) {
	names := make([]string, 0, len(n.autoCtors))
	for name := range n.autoCtors {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cands := n.autoCtors[name]
		if _, explicit := n.explicitCtors[name]; explicit || len(cands) < 2 {
			continue
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].module < cands[j].module })
		refs := make([]types.TypeRef, len(cands))
		for i, c := range cands {
			refs[i] = types.TypeRef{Module: c.module, Name: c.ctor.TypeName}
			registerCtorFactoryType(c.module, c.ctor, imports)
		}
		imports.AmbiguousCtors[name] = refs
		first := cands[0]
		paramCount := 0
		if tapp, ok := first.ctor.ResultType.(*types.TApp); ok {
			paramCount = len(tapp.Args)
		}
		imports.ImportedCtorInfos[name] = &importedCtorInfo{
			TypeName: first.ctor.TypeName, Module: first.module, CanonicalName: name,
			Arity: first.ctor.Arity, TypeParamCount: paramCount,
		}
		// No silent pick: the bare name is not a value, and the typechecker's
		// name-keyed constructor table does not hold one candidate for it.
		delete(imports.GlobalRefs, name)
		delete(imports.ImportedCtorTypes, name)
		delete(imports.ImportedCtorModules, name)
	}
}

// registerCtorFactoryType puts one constructor's factory type in scope under
// its module-qualified key (and its ADT's arity), without binding the bare name.
func registerCtorFactoryType(module string, ctor *iface.ConstructorScheme, imports *moduleImports) {
	var factoryType types.Type = ctor.ResultType
	if ctor.Arity > 0 {
		factoryType = &types.TFunc2{Params: ctor.FieldTypes, Return: ctor.ResultType}
	}
	var typeVars []string
	if ctor.ResultType != nil {
		typeVars = extractTypeVarsFromType(ctor.ResultType)
	}
	imports.ExternalTypes[core.ADTFactoryKey(module, ctor.TypeName, ctor.CtorName)] = &types.Scheme{TypeVars: typeVars, Type: factoryType}
	paramCount := 0
	if tapp, ok := ctor.ResultType.(*types.TApp); ok {
		paramCount = len(tapp.Args)
	}
	if key := types.NominalKey(module, ctor.TypeName); imports.ImportedADTTypeParams[key] == 0 {
		imports.ImportedADTTypeParams[key] = paramCount
	}
}
