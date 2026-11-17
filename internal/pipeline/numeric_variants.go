package pipeline

import (
	"fmt"
	"sort"

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
// insert after each original one (still unelaborated), and the renames of the
// original program's uses.
type numericVariants struct {
	after   map[int][]core.CoreExpr // index in Decls → variant declarations
	meta    map[string]*core.DeclMeta
	renames map[uint64]string // use node id → variant name
}

// planNumericVariants builds the variants of every binding of prog that has
// them, records their node types and resolved constraints in tc, and computes
// the renames of prog's uses. It runs on the type-checked, unelaborated
// program; nil when the module has no binding with variants and uses none.
//
// A declaration over at most types.NumVariantEager variables gets every
// variant, so any importer finds the one it names. One over more (a record
// constant of nine integer fields) gets the variants that this module's own
// uses, and its variants' uses, name: another module may use it only at int,
// and is told so at its use (TC_NUMVAR_001).
func planNumericVariants(prog *core.Program, tc *types.CoreTypeChecker) (*numericVariants, error) {
	decls, refs := tc.NumericDecls(), tc.NumericRefs()
	if len(decls) == 0 && len(refs) == 0 {
		return nil, nil
	}
	p := &variantPlanner{
		tc:       tc,
		meta:     prog.Meta,
		plan:     &numericVariants{after: map[int][]core.CoreExpr{}, meta: map[string]*core.DeclMeta{}, renames: map[uint64]string{}},
		byMember: map[string]*variantGroup{},
		next:     numVariantIDBase,
	}
	for i, decl := range prog.Decls {
		g := newVariantGroup(i, decl, decls)
		if g == nil {
			continue
		}
		for _, m := range g.members {
			p.byMember[m] = g
		}
		if len(g.vars) <= types.NumVariantEager {
			for mask := 1; mask < 1<<len(g.vars); mask++ {
				p.queue = append(p.queue, variantDemand{g, mask})
			}
		}
	}
	ids := make([]uint64, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	for _, id := range ids {
		if id >= numVariantIDBase {
			continue // a variant's own use, named when it was copied
		}
		ref := refs[id]
		floats, err := tc.NumericRefInstance(id, nil)
		if err != nil {
			return nil, err
		}
		name, err := p.use(ref, floats)
		if err != nil {
			return nil, err
		}
		if name != ref.Name {
			p.plan.renames[id] = name
		}
	}
	for len(p.queue) > 0 {
		d := p.queue[0]
		p.queue = p.queue[1:]
		if err := p.generate(d.group, d.mask); err != nil {
			return nil, err
		}
	}
	return p.plan, nil
}

// variantGroup is one top-level declaration with numeric variants: a Let, or
// a LetRec group whose members are copied together.
type variantGroup struct {
	idx     int
	decl    core.CoreExpr
	members []string
	vars    []string     // the union of the members' variables, in order
	done    map[int]bool // masks over vars already generated
	emitted map[string]bool
}

func newVariantGroup(idx int, decl core.CoreExpr, numDecls map[string][]string) *variantGroup {
	g := &variantGroup{idx: idx, decl: decl, done: map[int]bool{}, emitted: map[string]bool{}}
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

// variantPlanner generates a module's variants, eagerly or on demand.
type variantPlanner struct {
	tc       *types.CoreTypeChecker
	meta     map[string]*core.DeclMeta
	plan     *numericVariants
	byMember map[string]*variantGroup
	queue    []variantDemand
	next     uint64
}

// use returns the name a use of ref at floats takes, and asks for that
// variant when this module generates it on demand.
func (p *variantPlanner) use(ref types.NumericRef, floats []bool) (string, error) {
	name := types.NumVariantName(ref.Name, floats)
	if name == ref.Name {
		return name, nil
	}
	if ref.Module != "" {
		if n := len(types.NumVariantVars(ref.Scheme)); n > types.NumVariantEager {
			return "", fmt.Errorf("%s: at %s: %s from %s is generic over %d numeric types, and a binding generic over more than %d is compiled for other modules only at int, but this use needs %s at float. Annotate its numeric types in %s (x: float, or a field `v: float`)",
				types.NumVariantLimitCode, ref.Where, ref.Name, ref.Module, n, types.NumVariantEager, floatCount(floats), ref.Module)
		}
		return name, nil
	}
	g := p.byMember[ref.Name]
	if g == nil {
		return "", fmt.Errorf("numeric variant: %s at %s names no declaration of this module", ref.Name, ref.Where)
	}
	if mask := g.maskFor(ref.Name, floats, p.tc.NumericDecls()); mask != 0 && !g.done[mask] {
		p.queue = append(p.queue, variantDemand{g, mask})
	}
	return name, nil
}

func floatCount(floats []bool) string {
	n := 0
	for _, f := range floats {
		if f {
			n++
		}
	}
	if n == 1 {
		return "one of them"
	}
	return fmt.Sprintf("%d of them", n)
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

// generate copies g's declaration for the variant mask (its variables of mask
// at float), each member named for its own variables; a member whose variant
// another mask already produced is not copied again.
func (p *variantPlanner) generate(g *variantGroup, mask int) error {
	if g.done[mask] {
		return nil
	}
	g.done[mask] = true
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
			g.emitted[names[m]] = true
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
	p.plan.after[g.idx] = append(p.plan.after[g.idx], v)
	for _, m := range fresh {
		if mm := p.meta[m]; mm != nil {
			vm, err := c.meta(mm, names[m])
			if err != nil {
				return err
			}
			p.plan.meta[names[m]] = vm
		}
	}
	return nil
}

// install elaborates the variants' dictionaries, inserts each after its
// original declaration, and renames the original program's uses. prog is the
// dictionary-elaborated program; its Var nodes are the type-checked ones
// (ElaborateWithDictionaries keeps atoms), so the renames apply by node id.
func (p *numericVariants) install(prog *core.Program, tc *types.CoreTypeChecker) (*core.Program, error) {
	if p == nil {
		return prog, nil
	}
	renameUses(prog, p.renames)
	if len(p.after) == 0 {
		return prog, nil
	}
	resolved := tc.GetResolvedConstraints()
	out := &core.Program{Meta: prog.Meta, Flags: prog.Flags}
	if len(p.meta) > 0 {
		out.Meta = make(map[string]*core.DeclMeta, len(prog.Meta)+len(p.meta))
		for k, v := range prog.Meta {
			out.Meta[k] = v
		}
	}
	var added []core.CoreExpr
	for i, decl := range prog.Decls {
		out.Decls = append(out.Decls, decl)
		vs := p.after[i]
		if len(vs) == 0 {
			continue
		}
		el, err := elaborate.ElaborateWithDictionaries(&core.Program{Decls: vs}, resolved)
		if err != nil {
			return nil, fmt.Errorf("numeric variant: %w", err)
		}
		out.Decls = append(out.Decls, el.Decls...)
		added = append(added, el.Decls...)
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
