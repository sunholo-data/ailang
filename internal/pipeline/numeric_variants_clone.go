package pipeline

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// The copy of a declaration for one numeric variant (#2378,
// numeric_variants.go): fresh node ids, each with the node's type and
// resolved constraint in the variant, and its uses of bindings with variants
// named for the variant's instance.

// variantCloner copies a declaration for the variant sigma (variable → float).
// The first error is kept (err) and returned by decl or meta.
type variantCloner struct {
	tc    *types.CoreTypeChecker
	sigma types.Substitution
	next  *uint64
	group map[string]string // the declaration's members → their names here
	// use names a copied use of a binding with variants (variantPlanner.use).
	use func(ref types.NumericRef, floats []bool) (string, error)
	err error
	// frames: for each let-bound lambda copied per use (let node id), a stack
	// of the variants its uses name, one entry per copy of the let being made.
	frames      map[uint64][]map[string][]bool
	innerCopies int // copies of let-bound lambdas made
}

func (c *variantCloner) fail(err error) {
	if c.err == nil {
		c.err = err
	}
}

// decl copies a top-level declaration: its members named by names, keeping
// only the members of keep (the rest exist already, as the original or an
// earlier variant).
func (c *variantCloner) decl(decl core.CoreExpr, names map[string]string, keep []string) (core.CoreExpr, error) {
	switch d := decl.(type) {
	case *core.Let:
		value := c.expr(d.Value, nil)
		body := &core.Var{CoreNode: c.node(d.Body), Name: names[d.Name]}
		return &core.Let{CoreNode: c.node(d), Name: names[d.Name], Value: value, Body: body}, c.err
	case *core.LetRec:
		c.group = names
		kept := map[string]bool{}
		for _, m := range keep {
			kept[m] = true
		}
		out := &core.LetRec{CoreNode: c.node(d)}
		for _, b := range d.Bindings {
			if kept[b.Name] {
				out.Bindings = append(out.Bindings, core.RecBinding{Name: names[b.Name], Value: c.expr(b.Value, nil)})
			}
		}
		// The body names a member (or is unit); the name it takes here is
		// defined by this copy, the original or an earlier variant.
		out.Body = c.expr(d.Body, nil)
		return out, c.err
	}
	return nil, fmt.Errorf("numeric variant: unexpected declaration %T", decl)
}

// meta copies a binding's metadata for its variant name; contracts are copied
// at the variant's types, so a float call is checked by float predicates.
func (c *variantCloner) meta(m *core.DeclMeta, name string) (*core.DeclMeta, error) {
	out := *m
	out.Name = name
	out.IsExport = false // exported through the binding it varies (runtime.evaluateModule)
	out.VariantOf = m.Name
	out.Contracts = nil
	for _, k := range m.Contracts {
		if k != nil {
			kk := *k
			kk.Expr = c.expr(k.Expr, nil)
			out.Contracts = append(out.Contracts, &kk)
		}
	}
	return &out, c.err
}

// node returns a fresh node for old, with old's type and resolved constraint
// in this variant.
func (c *variantCloner) node(old core.CoreExpr) core.CoreNode {
	id := *c.next
	*c.next++
	oldID := old.ID()
	if t, ok := c.tc.VariantType(oldID, c.sigma); ok {
		c.tc.CoreTI.Set(id, t)
	} else if t, ok := c.tc.CoreTI.Get(oldID); ok {
		c.tc.CoreTI.Set(id, t)
	}
	if rc := c.tc.VariantResolved(oldID, c.sigma); rc != nil {
		c.tc.AddResolvedConstraint(id, rc)
	}
	if c.tc.DefaultedNodes()[oldID] {
		c.tc.MarkDefaultedNode(id)
	}
	return core.CoreNode{NodeID: id, CoreSpan: old.Span(), OrigSpan: old.OriginalSpan()}
}

func (c *variantCloner) exprs(xs []core.CoreExpr, bound map[string]bool) []core.CoreExpr {
	if xs == nil {
		return nil
	}
	out := make([]core.CoreExpr, len(xs))
	for i, x := range xs {
		out[i] = c.expr(x, bound)
	}
	return out
}

func (c *variantCloner) fields(m map[string]core.CoreExpr, bound map[string]bool) map[string]core.CoreExpr {
	out := make(map[string]core.CoreExpr, len(m))
	for _, k := range sortedExprKeys(m) {
		out[k] = c.expr(m[k], bound)
	}
	return out
}

// binding returns bound plus names (a scope that shadows group members).
func binding(bound map[string]bool, names ...string) map[string]bool {
	out := make(map[string]bool, len(bound)+len(names))
	for k := range bound {
		out[k] = true
	}
	for _, n := range names {
		out[n] = true
	}
	return out
}

// useName is the name a copied use of name (node old) takes in this variant:
// the variant its instantiation names when it uses a binding with variants,
// a group member's name in this copy when it is the declaration's own
// (recursive) use, else name.
func (c *variantCloner) useName(old core.CoreExpr, name string, bound map[string]bool) string {
	if ref, ok := c.tc.NumericRefs()[old.ID()]; ok && ref.InnerLet != 0 {
		floats, err := c.tc.NumericRefInstance(old.ID(), c.sigma)
		if err != nil {
			c.fail(err)
			return name
		}
		v := types.NumVariantName(name, floats)
		if stack := c.frames[ref.InnerLet]; v != name && len(stack) > 0 {
			stack[len(stack)-1][v] = floats
		} else if v != name {
			c.fail(fmt.Errorf("numeric variant: a use of %s at %s outside its let", name, ref.Where))
		}
		return v
	}
	if ref, ok := c.tc.NumericRefs()[old.ID()]; ok {
		floats, err := c.tc.NumericRefInstance(old.ID(), c.sigma)
		if err == nil {
			name, err = c.use(ref, floats)
		}
		c.fail(err)
		return name
	}
	if n, ok := c.group[name]; ok && !bound[name] {
		return n
	}
	return name
}

func (c *variantCloner) expr(e core.CoreExpr, bound map[string]bool) core.CoreExpr {
	if e == nil {
		return nil
	}
	switch n := e.(type) {
	case *core.Var:
		return &core.Var{CoreNode: c.node(n), Name: c.useName(n, n.Name, bound)}
	case *core.VarGlobal:
		ref := n.Ref
		if _, ok := c.tc.NumericRefs()[n.ID()]; ok { // never a group member: another module's
			ref.Name = c.useName(n, ref.Name, bound)
		}
		return &core.VarGlobal{CoreNode: c.node(n), Ref: ref}
	case *core.Lit:
		return &core.Lit{CoreNode: c.node(n), Kind: n.Kind, Value: n.Value}
	case *core.Lambda:
		return &core.Lambda{CoreNode: c.node(n), Params: n.Params, Body: c.expr(n.Body, binding(bound, n.Params...))}
	case *core.Let:
		if vars := c.tc.InnerLetVars(n.ID()); vars != nil {
			return c.innerLet(n, vars, bound)
		}
		return &core.Let{CoreNode: c.node(n), Name: n.Name, Value: c.expr(n.Value, bound), Body: c.expr(n.Body, binding(bound, n.Name))}
	case *core.LetRec:
		if vars := c.tc.InnerLetVars(n.ID()); vars != nil && len(n.Bindings) == 1 {
			return c.innerLetRec(n, vars, bound)
		}
		out := &core.LetRec{CoreNode: c.node(n)}
		names := make([]string, len(n.Bindings))
		for i, b := range n.Bindings {
			names[i] = b.Name
		}
		inner := binding(bound, names...)
		for _, b := range n.Bindings {
			out.Bindings = append(out.Bindings, core.RecBinding{Name: b.Name, Value: c.expr(b.Value, inner)})
		}
		out.Body = c.expr(n.Body, inner)
		return out
	case *core.App:
		return &core.App{CoreNode: c.node(n), Func: c.expr(n.Func, bound), Args: c.exprs(n.Args, bound), TraceRedact: n.TraceRedact}
	case *core.If:
		return &core.If{CoreNode: c.node(n), Cond: c.expr(n.Cond, bound), Then: c.expr(n.Then, bound), Else: c.expr(n.Else, bound)}
	case *core.Match:
		out := &core.Match{CoreNode: c.node(n), Scrutinee: c.expr(n.Scrutinee, bound), Exhaustive: n.Exhaustive}
		for _, a := range n.Arms {
			inner := binding(bound, patternBoundVars(a.Pattern)...)
			out.Arms = append(out.Arms, core.MatchArm{Pattern: a.Pattern, Guard: c.expr(a.Guard, inner), Body: c.expr(a.Body, inner)})
		}
		return out
	case *core.BinOp:
		return &core.BinOp{CoreNode: c.node(n), Op: n.Op, Left: c.expr(n.Left, bound), Right: c.expr(n.Right, bound)}
	case *core.UnOp:
		return &core.UnOp{CoreNode: c.node(n), Op: n.Op, Operand: c.expr(n.Operand, bound)}
	case *core.Intrinsic:
		return &core.Intrinsic{CoreNode: c.node(n), Op: n.Op, Args: c.exprs(n.Args, bound)}
	case *core.Record:
		return &core.Record{CoreNode: c.node(n), Fields: c.fields(n.Fields, bound)}
	case *core.RecordAccess:
		return &core.RecordAccess{CoreNode: c.node(n), Record: c.expr(n.Record, bound), Field: n.Field}
	case *core.RecordUpdate:
		return &core.RecordUpdate{CoreNode: c.node(n), Base: c.expr(n.Base, bound), Updates: c.fields(n.Updates, bound)}
	case *core.List:
		return &core.List{CoreNode: c.node(n), Elements: c.exprs(n.Elements, bound)}
	case *core.Array:
		return &core.Array{CoreNode: c.node(n), Elements: c.exprs(n.Elements, bound)}
	case *core.Tuple:
		return &core.Tuple{CoreNode: c.node(n), Elements: c.exprs(n.Elements, bound)}
	case *core.Forall:
		return &core.Forall{CoreNode: c.node(n), Var: n.Var, Lo: c.expr(n.Lo, bound), Hi: c.expr(n.Hi, bound), Body: c.expr(n.Body, binding(bound, n.Var))}
	}
	// Dictionary nodes do not exist before dictionary elaboration; anything
	// else is a Core form this copy does not know, and copying it by reference
	// would share node ids (types) between variants.
	c.fail(fmt.Errorf("numeric variant: cannot copy Core node %T", e))
	return e
}

// innerLet copies a let-bound function that is copied per use: its body
// first, collecting the variants its uses name, then the value once as written
// and once per variant, each bound right after it (`let h = … in let h$numF =
// … in body`), so every use is in scope of the copy it names. A variant's
// value is the function's with its own numeric variables at float on top of
// this copy's.
func (c *variantCloner) innerLet(n *core.Let, vars []string, bound map[string]bool) core.CoreExpr {
	node := c.node(n)
	c.pushFrame(n.ID())
	value := c.expr(n.Value, bound)
	body := c.expr(n.Body, binding(bound, n.Name))
	need := c.popFrame(n.ID())
	for _, name := range sortedNeeds(need) {
		inner := c.innerCloner(vars, need[name], nil)
		v := inner.expr(n.Value, bound)
		c.fail(inner.err)
		c.innerCopies += inner.innerCopies + 1
		body = &core.Let{CoreNode: c.node(n), Name: name, Value: v, Body: body}
	}
	return &core.Let{CoreNode: node, Name: n.Name, Value: value, Body: body}
}

// innerLetRec is innerLet for a let-bound recursive function (a letrec of one
// binding): each variant is its own letrec, whose recursive calls name it.
func (c *variantCloner) innerLetRec(n *core.LetRec, vars []string, bound map[string]bool) core.CoreExpr {
	node := c.node(n)
	b := n.Bindings[0]
	inner := binding(bound, b.Name)
	c.pushFrame(n.ID())
	value := c.expr(b.Value, inner)
	body := c.expr(n.Body, inner)
	need := c.popFrame(n.ID())
	for _, name := range sortedNeeds(need) {
		vc := c.innerCloner(vars, need[name], map[string]string{b.Name: name})
		without := make(map[string]bool, len(bound))
		for k := range bound {
			if k != b.Name {
				without[k] = true
			}
		}
		v := vc.expr(b.Value, without)
		c.fail(vc.err)
		c.innerCopies += vc.innerCopies + 1
		body = &core.LetRec{CoreNode: c.node(n), Bindings: []core.RecBinding{{Name: name, Value: v}}, Body: body}
	}
	return &core.LetRec{CoreNode: node, Bindings: []core.RecBinding{{Name: b.Name, Value: value}}, Body: body}
}

func (c *variantCloner) pushFrame(id uint64) {
	if c.frames == nil {
		c.frames = map[uint64][]map[string][]bool{}
	}
	c.frames[id] = append(c.frames[id], map[string][]bool{})
}

func (c *variantCloner) popFrame(id uint64) map[string][]bool {
	stack := c.frames[id]
	need := stack[len(stack)-1]
	c.frames[id] = stack[:len(stack)-1]
	return need
}

// sortedNeeds lists a frame's variant names in reverse order, the order the
// copies are wrapped around the body (so they read sorted).
func sortedNeeds(need map[string][]bool) []string {
	names := make([]string, 0, len(need))
	for v := range need {
		names = append(names, v)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	return names
}

// innerCloner copies a let-bound function's value for one of its variants:
// its variables vars at float where floats says, on top of this copy's sigma;
// rename maps a recursive binding to the variant's name.
func (c *variantCloner) innerCloner(vars []string, floats []bool, rename map[string]string) *variantCloner {
	sigma := make(types.Substitution, len(c.sigma)+len(vars))
	for k, t := range c.sigma {
		sigma[k] = t
	}
	for k, f := range floats {
		if f && k < len(vars) {
			sigma[vars[k]] = types.TFloat
		}
	}
	group := c.group
	if len(rename) > 0 {
		group = make(map[string]string, len(c.group)+len(rename))
		for k, v := range c.group {
			group[k] = v
		}
		for k, v := range rename {
			group[k] = v
		}
	}
	return &variantCloner{tc: c.tc, sigma: sigma, next: c.next, group: group, use: c.use, frames: c.frames}
}
