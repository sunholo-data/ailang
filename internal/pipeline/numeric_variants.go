package pipeline

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/types"
)

// Numeric variants (#2378, #2333; the type checker's side and the rationale
// are in internal/types/numeric_variants.go).
//
// A top-level binding generic over `Num a` is compiled as written, at the
// default int, and once more for every other assignment of its Num variables
// to int and float: `inc$numF` for `func inc(x) = x + 1`. A variant is a copy
// of the declaration with fresh node ids whose recorded types, dictionaries
// and literals are read from the generic body with the chosen variables at
// float, so every back-end consumer of a node's type (dictionary elaboration,
// ShowNormalizer, operator lowering, the VM compiler) sees float there, not a
// default it must second-guess. Every use of such a binding, in this module or
// an importer, is renamed to the variant its instantiation names; a use at int
// is left alone, so int code runs the code it always ran.

// numVariantIDBase is where variant node ids start: above the elaborator's and
// the specializer's (1000000) ranges of the same module.
const numVariantIDBase uint64 = 1 << 40

// numericVariants is the plan for one module: the variant declarations to
// insert after each original one, the declarations to replace by a copy (one
// holding a let-bound lambda copied per use), all still unelaborated, and the
// renames of the original program's uses.
type numericVariants struct {
	after   map[int][]core.CoreExpr // index in Decls → variant declarations
	replace map[int]core.CoreExpr   // index in Decls → its copy
	meta    map[string]*core.DeclMeta
	renames map[uint64]string // use node id → variant name
	missing map[string]string // exported variants not compiled → why
}

// importedBinding reports whether module (another one) defines a binding
// named name; known is false when the module's compiled program is not at
// hand; why says why the module could not compile it, when it recorded that.
type importedBinding func(module, name string) (has, known bool, why string)

// planNumericVariants builds the variants of every binding of prog that has
// them, records their node types and resolved constraints in tc, and computes
// the renames of prog's uses. It runs on the type-checked, unelaborated
// program; nil when the module has no binding with variants and uses none.
//
// An exported declaration over at most types.NumVariantEager variables gets
// every variant, so an importer finds the one it names. Any other gets the
// variants a use names: a use in this program, or in a variant generated for
// one. A variant that cannot be compiled (it needs another module's variant
// that does not exist) is an error only at a use in this program that needs
// it, located there (TC_NUMVAR_001); generating it eagerly never fails the
// module, and an importer that needs it is told at its own use.
func planNumericVariants(prog *core.Program, tc *types.CoreTypeChecker, imported importedBinding) (*numericVariants, error) {
	decls, refs := tc.NumericDecls(), tc.NumericRefs()
	if len(decls) == 0 && len(refs) == 0 {
		return nil, nil
	}
	p := &variantPlanner{
		tc:       tc,
		meta:     prog.Meta,
		imported: imported,
		plan: &numericVariants{after: map[int][]core.CoreExpr{}, replace: map[int]core.CoreExpr{},
			meta: map[string]*core.DeclMeta{}, renames: map[uint64]string{}, missing: map[string]string{}},
		byMember: map[string]*variantGroup{},
		next:     numVariantIDBase,
	}
	var eager []variantDemand
	for i, decl := range prog.Decls {
		g := newVariantGroup(i, decl, decls)
		if g == nil {
			continue
		}
		exported := false
		for _, m := range g.members {
			p.byMember[m] = g
			exported = exported || (prog.Meta[m] != nil && prog.Meta[m].IsExport)
		}
		if exported && len(g.vars) <= types.NumVariantEager {
			for mask := 1; mask < 1<<len(g.vars); mask++ {
				eager = append(eager, variantDemand{g, mask})
			}
		}
	}
	for _, d := range eager {
		// A variant that cannot be built is simply absent; an importer that
		// needs it is told why (NumVariantMissing).
		if err := p.ensure(d.group, d.mask); err != nil {
			for _, m := range d.group.members {
				p.plan.missing[types.NumVariantName(m, d.group.floatsOf(m, d.mask, p.tc.NumericDecls()))] = d.group.causes[d.mask]
			}
		}
	}
	ids := make([]uint64, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	for _, id := range ids {
		ref := refs[id]
		if id >= numVariantIDBase || ref.InnerLet != 0 {
			continue // a copy's own use, or a let-bound lambda's: named when copied
		}
		floats, err := tc.NumericRefInstance(id, nil)
		if err != nil {
			return nil, err
		}
		name, err := p.use(ref, floats)
		if err != nil {
			return nil, err // a use in this program: report it
		}
		if name != ref.Name {
			p.plan.renames[id] = name
		}
	}
	// A declaration holding a let-bound lambda copied per use is replaced by
	// its own copy, which adds the lambda's copies its uses need.
	for i, decl := range prog.Decls {
		if !p.holdsInnerLet(decl) {
			continue
		}
		names := map[string]string{}
		for _, m := range declBindingNames(decl) {
			names[m] = m
		}
		c := &variantCloner{tc: tc, sigma: types.Substitution{}, next: &p.next, use: p.use}
		var cp core.CoreExpr
		var err error
		if core.IsModuleDecl(decl) {
			cp, err = c.decl(decl, names, declBindingNames(decl))
		} else {
			// A top-level expression is copied whole, its body included.
			cp, err = c.expr(decl, nil), c.err
		}
		if err != nil {
			return nil, err
		}
		if c.innerCopies > 0 {
			p.plan.replace[i] = cp
		}
	}
	return p.plan, nil
}

// declBindingNames lists every binding of decl (all are kept in a same-name copy).
func declBindingNames(decl core.CoreExpr) []string {
	switch d := decl.(type) {
	case *core.Let:
		return []string{d.Name}
	case *core.LetRec:
		out := make([]string, len(d.Bindings))
		for i, b := range d.Bindings {
			out[i] = b.Name
		}
		return out
	}
	return nil
}

// holdsInnerLet reports whether decl contains a let-bound lambda copied per use.
func (p *variantPlanner) holdsInnerLet(decl core.CoreExpr) bool {
	found := false
	walkCoreExprs(decl, func(e core.CoreExpr) {
		if (isLetOrLetRec(e)) && p.tc.InnerLetVars(e.ID()) != nil {
			found = true
		}
	})
	return found
}

// variantGroup is one top-level declaration with numeric variants: a Let, or
// a LetRec group whose members are copied together.
type variantGroup struct {
	idx     int
	decl    core.CoreExpr
	members []string
	vars    []string     // the union of the members' variables, in order
	done    map[int]bool // masks over vars already generated (or being)
	failed  map[int]error
	causes  map[int]string // why a failed mask failed: the binding at fault and the chain to it
	emitted map[string]bool
}

func newVariantGroup(idx int, decl core.CoreExpr, numDecls map[string][]string) *variantGroup {
	g := &variantGroup{idx: idx, decl: decl, done: map[int]bool{}, failed: map[int]error{}, causes: map[int]string{}, emitted: map[string]bool{}}
	switch d := decl.(type) {
	case *core.Let:
		g.members = []string{d.Name}
	case *core.LetRec:
		for _, b := range d.Bindings {
			g.members = append(g.members, b.Name)
		}
	default:
		return nil
	}
	seen := map[string]bool{}
	for _, m := range g.members {
		for _, v := range numDecls[m] {
			if !seen[v] {
				seen[v] = true
				g.vars = append(g.vars, v)
			}
		}
	}
	if len(g.vars) == 0 {
		return nil
	}
	return g
}

type variantDemand struct {
	group *variantGroup
	mask  int
}

// variantPlanner generates a module's variants.
type variantPlanner struct {
	tc       *types.CoreTypeChecker
	meta     map[string]*core.DeclMeta
	imported importedBinding
	plan     *numericVariants
	byMember map[string]*variantGroup
	next     uint64
	// cause: the last missing imported variant a use ran into, phrased as the
	// binding at fault and the chain to it (use).
	cause string
}

// use returns the name a use of ref at floats takes, generating that variant
// when this module defines it; an error when the variant does not exist.
func (p *variantPlanner) use(ref types.NumericRef, floats []bool) (string, error) {
	name := types.NumVariantName(ref.Name, floats)
	if name == ref.Name {
		return name, nil
	}
	if ref.Module != "" {
		if p.imported != nil {
			if has, known, why := p.imported(ref.Module, name); known && !has {
				if why != "" {
					// The variant exists in principle; compiling it needed
					// another binding's missing variant: name that one.
					p.cause = fmt.Sprintf("%s from %s (used at %s), whose float copy needs %s", ref.Name, ref.Module, ref.Where, why)
					return "", numVarError(ref, fmt.Sprintf("%s from %s has no variant for this use at float: it needs %s", ref.Name, ref.Module, why), ref.Module)
				}
				p.cause = fmt.Sprintf("%s from %s (used at %s), which is not compiled with %s of its %d numeric types at float: a binding over more than %d of them gets only the variants its own module uses",
					ref.Name, ref.Module, ref.Where, floatCount(floats), len(floats), types.NumVariantEager)
				return "", numVarError(ref, fmt.Sprintf("%s from %s is not compiled with %s of its %d numeric types at float: a binding over more than %d of them gets only the variants its own module uses",
					ref.Name, ref.Module, floatCount(floats), len(floats), types.NumVariantEager), ref.Module)
			}
		}
		return name, nil
	}
	g := p.byMember[ref.Name]
	if g == nil {
		return "", fmt.Errorf("numeric variant: %s at %s names no declaration of this module", ref.Name, ref.Where)
	}
	if mask := g.maskFor(ref.Name, floats, p.tc.NumericDecls()); mask != 0 {
		if err := p.ensure(g, mask); err != nil {
			if c := g.causes[mask]; c != "" {
				p.cause = fmt.Sprintf("%s (used at %s), whose float copy needs %s", ref.Name, ref.Where, c)
				return "", numVarError(ref, fmt.Sprintf("%s at float needs %s", ref.Name, c), "this module")
			}
			return "", numVarError(ref, fmt.Sprintf("%s at float needs a variant that cannot be compiled: %v", ref.Name, err), "this module")
		}
	}
	return name, nil
}

// numVarError is TC_NUMVAR_001 located at the use ref.
func numVarError(ref types.NumericRef, why, where string) error {
	return ailerrors.WrapReport(&ailerrors.Report{
		Schema:  "ailang.error/v1",
		Code:    types.NumVariantLimitCode,
		Phase:   "typecheck",
		Message: fmt.Sprintf("at %s: %s", ref.Where, why),
		Span:    &ast.Span{Start: ref.Pos, End: ref.Pos},
		Data:    map[string]any{"binding": ref.Name, "module": ref.Module},
		Fix:     &ailerrors.Fix{Suggestion: "annotate the numeric parameter types of " + ref.Name + " in " + where + " (x: float), or call it at int"},
	})
}

func floatCount(floats []bool) string {
	n := 0
	for _, f := range floats {
		if f {
			n++
		}
	}
	return fmt.Sprintf("%d", n)
}

// maskFor is the mask over g.vars that member's variant floats names.
func (g *variantGroup) maskFor(member string, floats []bool, numDecls map[string][]string) int {
	mask := 0
	for i, v := range numDecls[member] {
		if i >= len(floats) || !floats[i] {
			continue
		}
		for k, gv := range g.vars {
			if gv == v {
				mask |= 1 << k
			}
		}
	}
	return mask
}

// ensure generates g's variant for mask once, and returns why it could not be
// generated. A mask being generated counts as done: a use of the declaration
// inside its own copy names its group member, not ensure.
func (p *variantPlanner) ensure(g *variantGroup, mask int) error {
	if g.done[mask] {
		return g.failed[mask]
	}
	g.done[mask] = true
	p.cause = ""
	err := p.generate(g, mask)
	if err != nil {
		g.failed[mask] = err
		g.causes[mask] = p.cause
	}
	return err
}

// floatsOf is member's variant flags for the group mask.
func (g *variantGroup) floatsOf(member string, mask int, numDecls map[string][]string) []bool {
	own := numDecls[member]
	floats := make([]bool, len(own))
	for k, v := range own {
		for i, gv := range g.vars {
			if gv == v && mask&(1<<i) != 0 {
				floats[k] = true
			}
		}
	}
	return floats
}

// generate copies g's declaration for the variant mask (its variables of mask
// at float), each member named for its own variables; a member whose variant
// another mask already produced is not copied again. Nothing is recorded when
// the copy fails.
func (p *variantPlanner) generate(g *variantGroup, mask int) error {
	numDecls := p.tc.NumericDecls()
	sigma := types.Substitution{}
	for k, v := range g.vars {
		if mask&(1<<k) != 0 {
			sigma[v] = types.TFloat
		}
	}
	names := map[string]string{} // member → its name in this variant
	var fresh []string
	for _, m := range g.members {
		own := numDecls[m]
		floats := make([]bool, len(own))
		for k, v := range own {
			floats[k] = sigma[v] != nil
		}
		names[m] = types.NumVariantName(m, floats)
		if names[m] != m && !g.emitted[names[m]] {
			fresh = append(fresh, m)
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	c := &variantCloner{tc: p.tc, sigma: sigma, next: &p.next, use: p.use}
	v, err := c.decl(g.decl, names, fresh)
	if err != nil {
		return err
	}
	metas := map[string]*core.DeclMeta{}
	for _, m := range fresh {
		if mm := p.meta[m]; mm != nil {
			vm, err := c.meta(mm, names[m])
			if err != nil {
				return err
			}
			metas[names[m]] = vm
		}
	}
	for _, m := range fresh {
		g.emitted[names[m]] = true
	}
	for k, vm := range metas {
		p.plan.meta[k] = vm
	}
	p.plan.after[g.idx] = append(p.plan.after[g.idx], v)
	return nil
}

// install elaborates the variants' dictionaries, inserts each after its
// original declaration (or its copy), and renames the original program's
// uses. prog is the dictionary-elaborated program; its Var nodes are the
// type-checked ones (ElaborateWithDictionaries keeps atoms), so the renames
// apply by node id.
func (p *numericVariants) install(prog *core.Program, tc *types.CoreTypeChecker) (*core.Program, error) {
	// The plan names nodes and bindings as the type checker saw them; after
	// core.UniqueProgramLocals a variant's copied body would keep the original
	// (shadowing) local names inside a renamed program, and a use could bind
	// to the wrong local: a wrong answer, not an error (#2393 review 3). So the
	// order is enforced, not just documented.
	if prog != nil && prog.Flags.LocalsUnique {
		return nil, fmt.Errorf("numeric variant: internal error: variants must be installed before core.UniqueProgramLocals renames the program's locals")
	}
	if p == nil {
		return prog, nil
	}
	renameUses(prog, p.renames)
	if len(p.after) == 0 && len(p.replace) == 0 {
		return prog, nil
	}
	resolved := tc.GetResolvedConstraints()
	elab := func(ds []core.CoreExpr) ([]core.CoreExpr, error) {
		el, err := elaborate.ElaborateWithDictionaries(&core.Program{Decls: ds}, resolved)
		if err != nil {
			return nil, fmt.Errorf("numeric variant: %w", err)
		}
		return el.Decls, nil
	}
	out := &core.Program{Meta: prog.Meta, Flags: prog.Flags}
	if len(p.meta) > 0 {
		out.Meta = make(map[string]*core.DeclMeta, len(prog.Meta)+len(p.meta))
		for k, v := range prog.Meta {
			out.Meta[k] = v
		}
	}
	var added []core.CoreExpr
	for i, decl := range prog.Decls {
		if cp, ok := p.replace[i]; ok {
			ds, err := elab([]core.CoreExpr{cp})
			if err != nil {
				return nil, err
			}
			decl = ds[0]
			added = append(added, decl)
		}
		out.Decls = append(out.Decls, decl)
		if vs := p.after[i]; len(vs) > 0 {
			ds, err := elab(vs)
			if err != nil {
				return nil, err
			}
			out.Decls = append(out.Decls, ds...)
			added = append(added, ds...)
		}
	}
	if len(p.meta) > 0 {
		el, err := elaborate.ElaborateWithDictionaries(&core.Program{Meta: p.meta}, resolved)
		if err != nil {
			return nil, fmt.Errorf("numeric variant: %w", err)
		}
		for k, v := range el.Meta {
			out.Meta[k] = v
		}
	}
	if err := ValidateCoreTypeInfo(&core.Program{Decls: added}, tc.CoreTI); err != nil {
		return nil, fmt.Errorf("numeric variant: %w", err)
	}
	return out, nil
}

// renameUses renames, in place, every use node of prog listed in renames.
func renameUses(prog *core.Program, renames map[uint64]string) {
	if len(renames) == 0 {
		return
	}
	visit := func(e core.CoreExpr) {
		switch v := e.(type) {
		case *core.Var:
			if n, ok := renames[v.ID()]; ok {
				v.Name = n
			}
		case *core.VarGlobal:
			if n, ok := renames[v.ID()]; ok {
				v.Ref.Name = n
			}
		}
	}
	for _, d := range prog.Decls {
		walkCoreExprs(d, visit)
	}
	for _, m := range prog.Meta {
		if m == nil {
			continue
		}
		for _, c := range m.Contracts {
			if c != nil {
				walkCoreExprs(c.Expr, visit)
			}
		}
	}
}

// walkCoreExprs calls visit on every expression of e, e included.
func walkCoreExprs(e core.CoreExpr, visit func(core.CoreExpr)) {
	if e == nil {
		return
	}
	visit(e)
	each := func(xs ...core.CoreExpr) {
		for _, x := range xs {
			walkCoreExprs(x, visit)
		}
	}
	switch n := e.(type) {
	case *core.Lambda:
		each(n.Body)
	case *core.Let:
		each(n.Value, n.Body)
	case *core.LetRec:
		for _, b := range n.Bindings {
			each(b.Value)
		}
		each(n.Body)
	case *core.App:
		each(n.Func)
		each(n.Args...)
	case *core.If:
		each(n.Cond, n.Then, n.Else)
	case *core.Match:
		each(n.Scrutinee)
		for _, a := range n.Arms {
			each(a.Guard, a.Body)
		}
	case *core.BinOp:
		each(n.Left, n.Right)
	case *core.UnOp:
		each(n.Operand)
	case *core.Record:
		for _, k := range sortedExprKeys(n.Fields) {
			each(n.Fields[k])
		}
	case *core.RecordAccess:
		each(n.Record)
	case *core.RecordUpdate:
		each(n.Base)
		for _, k := range sortedExprKeys(n.Updates) {
			each(n.Updates[k])
		}
	case *core.List:
		each(n.Elements...)
	case *core.Array:
		each(n.Elements...)
	case *core.Tuple:
		each(n.Elements...)
	case *core.Intrinsic:
		each(n.Args...)
	case *core.DictApp:
		each(n.Dict)
		each(n.Args...)
	case *core.DictAbs:
		each(n.Body)
	case *core.Forall:
		each(n.Lo, n.Hi, n.Body)
	}
}

func sortedExprKeys(m map[string]core.CoreExpr) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// isLetOrLetRec reports whether e binds a let-bound function (a let or a letrec).
func isLetOrLetRec(e core.CoreExpr) bool {
	switch e.(type) {
	case *core.Let, *core.LetRec:
		return true
	}
	return false
}
