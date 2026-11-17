package pipeline

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// The copy of a declaration for one numeric variant (#2378,
// numeric_variants.go): fresh node ids, each with the node's type and
// resolved constraint in the variant, and its uses of bindings with variants
// named for the variant's instance.

// variantCloner copies a declaration for the variant sigma (variable → float).
type variantCloner struct {
	tc    *types.CoreTypeChecker
	sigma types.Substitution
	next  *uint64
	group map[string]string // the declaration's members → their names here
	// use names a copied use of a binding with variants (variantPlanner.use).
	use func(ref types.NumericRef, floats []bool) (string, error)
}

// decl copies a top-level declaration: its members named by names, keeping
// only the members of keep (the rest exist already, as the original or an
// earlier variant).
func (c *variantCloner) decl(decl core.CoreExpr, names map[string]string, keep []string) (core.CoreExpr, error) {
	switch d := decl.(type) {
	case *core.Let:
		value, err := c.expr(d.Value, nil)
		if err != nil {
			return nil, err
		}
		body := &core.Var{CoreNode: c.node(d.Body), Name: names[d.Name]}
		return &core.Let{CoreNode: c.node(d), Name: names[d.Name], Value: value, Body: body}, nil
	case *core.LetRec:
		c.group = names
		kept := map[string]bool{}
		for _, m := range keep {
			kept[m] = true
		}
		out := &core.LetRec{CoreNode: c.node(d)}
		for _, b := range d.Bindings {
			if !kept[b.Name] {
				continue
			}
			value, err := c.expr(b.Value, nil)
			if err != nil {
				return nil, err
			}
			out.Bindings = append(out.Bindings, core.RecBinding{Name: names[b.Name], Value: value})
		}
		// The body names a member (or is unit); the name it takes here is
		// defined by this copy, the original or an earlier variant.
		body, err := c.expr(d.Body, nil)
		if err != nil {
			return nil, err
		}
		out.Body = body
		return out, nil
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
		if k == nil {
			continue
		}
		kk := *k
		if k.Expr != nil {
			e, err := c.expr(k.Expr, nil)
			if err != nil {
				return nil, err
			}
			kk.Expr = e
		}
		out.Contracts = append(out.Contracts, &kk)
	}
	return &out, nil
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

func (c *variantCloner) exprs(xs []core.CoreExpr, bound map[string]bool) ([]core.CoreExpr, error) {
	if xs == nil {
		return nil, nil
	}
	out := make([]core.CoreExpr, len(xs))
	for i, x := range xs {
		y, err := c.expr(x, bound)
		if err != nil {
			return nil, err
		}
		out[i] = y
	}
	return out, nil
}

func (c *variantCloner) fields(m map[string]core.CoreExpr, bound map[string]bool) (map[string]core.CoreExpr, error) {
	out := make(map[string]core.CoreExpr, len(m))
	for _, k := range sortedExprKeys(m) {
		y, err := c.expr(m[k], bound)
		if err != nil {
			return nil, err
		}
		out[k] = y
	}
	return out, nil
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

// useName is the name a copied use of name (node old) takes in this variant.
func (c *variantCloner) useName(old core.CoreExpr, name string, bound map[string]bool) (string, error) {
	if ref, ok := c.tc.NumericRefs()[old.ID()]; ok {
		floats, err := c.tc.NumericRefInstance(old.ID(), c.sigma)
		if err != nil {
			return "", err
		}
		return c.use(ref, floats)
	}
	if n, ok := c.group[name]; ok && !bound[name] {
		return n, nil
	}
	return name, nil
}

func (c *variantCloner) expr(e core.CoreExpr, bound map[string]bool) (core.CoreExpr, error) {
	if e == nil {
		return nil, nil
	}
	var err error
	switch n := e.(type) {
	case *core.Var:
		name, err := c.useName(n, n.Name, bound)
		if err != nil {
			return nil, err
		}
		return &core.Var{CoreNode: c.node(n), Name: name}, nil
	case *core.VarGlobal:
		ref := n.Ref
		if _, ok := c.tc.NumericRefs()[n.ID()]; ok {
			if ref.Name, err = c.useName(n, ref.Name, bound); err != nil {
				return nil, err
			}
		}
		return &core.VarGlobal{CoreNode: c.node(n), Ref: ref}, nil
	case *core.Lit:
		return &core.Lit{CoreNode: c.node(n), Kind: n.Kind, Value: n.Value}, nil
	case *core.Lambda:
		node := c.node(n)
		body, err := c.expr(n.Body, binding(bound, n.Params...))
		if err != nil {
			return nil, err
		}
		return &core.Lambda{CoreNode: node, Params: n.Params, Body: body}, nil
	case *core.Let:
		node := c.node(n)
		value, err := c.expr(n.Value, bound)
		if err != nil {
			return nil, err
		}
		body, err := c.expr(n.Body, binding(bound, n.Name))
		if err != nil {
			return nil, err
		}
		return &core.Let{CoreNode: node, Name: n.Name, Value: value, Body: body}, nil
	case *core.LetRec:
		node := c.node(n)
		names := make([]string, len(n.Bindings))
		for i, b := range n.Bindings {
			names[i] = b.Name
		}
		inner := binding(bound, names...)
		out := &core.LetRec{CoreNode: node}
		for _, b := range n.Bindings {
			value, err := c.expr(b.Value, inner)
			if err != nil {
				return nil, err
			}
			out.Bindings = append(out.Bindings, core.RecBinding{Name: b.Name, Value: value})
		}
		if out.Body, err = c.expr(n.Body, inner); err != nil {
			return nil, err
		}
		return out, nil
	case *core.App:
		node := c.node(n)
		fn, err := c.expr(n.Func, bound)
		if err != nil {
			return nil, err
		}
		args, err := c.exprs(n.Args, bound)
		if err != nil {
			return nil, err
		}
		return &core.App{CoreNode: node, Func: fn, Args: args, TraceRedact: n.TraceRedact}, nil
	case *core.If:
		node := c.node(n)
		xs, err := c.exprs([]core.CoreExpr{n.Cond, n.Then, n.Else}, bound)
		if err != nil {
			return nil, err
		}
		return &core.If{CoreNode: node, Cond: xs[0], Then: xs[1], Else: xs[2]}, nil
	case *core.Match:
		node := c.node(n)
		scrut, err := c.expr(n.Scrutinee, bound)
		if err != nil {
			return nil, err
		}
		out := &core.Match{CoreNode: node, Scrutinee: scrut, Exhaustive: n.Exhaustive}
		for _, a := range n.Arms {
			inner := binding(bound, patternBoundVars(a.Pattern)...)
			guard, err := c.expr(a.Guard, inner)
			if err != nil {
				return nil, err
			}
			body, err := c.expr(a.Body, inner)
			if err != nil {
				return nil, err
			}
			out.Arms = append(out.Arms, core.MatchArm{Pattern: a.Pattern, Guard: guard, Body: body})
		}
		return out, nil
	case *core.BinOp:
		node := c.node(n)
		xs, err := c.exprs([]core.CoreExpr{n.Left, n.Right}, bound)
		if err != nil {
			return nil, err
		}
		return &core.BinOp{CoreNode: node, Op: n.Op, Left: xs[0], Right: xs[1]}, nil
	case *core.UnOp:
		node := c.node(n)
		x, err := c.expr(n.Operand, bound)
		if err != nil {
			return nil, err
		}
		return &core.UnOp{CoreNode: node, Op: n.Op, Operand: x}, nil
	case *core.Intrinsic:
		node := c.node(n)
		args, err := c.exprs(n.Args, bound)
		if err != nil {
			return nil, err
		}
		return &core.Intrinsic{CoreNode: node, Op: n.Op, Args: args}, nil
	case *core.Record:
		node := c.node(n)
		fs, err := c.fields(n.Fields, bound)
		if err != nil {
			return nil, err
		}
		return &core.Record{CoreNode: node, Fields: fs}, nil
	case *core.RecordAccess:
		node := c.node(n)
		r, err := c.expr(n.Record, bound)
		if err != nil {
			return nil, err
		}
		return &core.RecordAccess{CoreNode: node, Record: r, Field: n.Field}, nil
	case *core.RecordUpdate:
		node := c.node(n)
		base, err := c.expr(n.Base, bound)
		if err != nil {
			return nil, err
		}
		us, err := c.fields(n.Updates, bound)
		if err != nil {
			return nil, err
		}
		return &core.RecordUpdate{CoreNode: node, Base: base, Updates: us}, nil
	case *core.List:
		node := c.node(n)
		xs, err := c.exprs(n.Elements, bound)
		if err != nil {
			return nil, err
		}
		return &core.List{CoreNode: node, Elements: xs}, nil
	case *core.Array:
		node := c.node(n)
		xs, err := c.exprs(n.Elements, bound)
		if err != nil {
			return nil, err
		}
		return &core.Array{CoreNode: node, Elements: xs}, nil
	case *core.Tuple:
		node := c.node(n)
		xs, err := c.exprs(n.Elements, bound)
		if err != nil {
			return nil, err
		}
		return &core.Tuple{CoreNode: node, Elements: xs}, nil
	case *core.Forall:
		node := c.node(n)
		xs, err := c.exprs([]core.CoreExpr{n.Lo, n.Hi}, bound)
		if err != nil {
			return nil, err
		}
		body, err := c.expr(n.Body, binding(bound, n.Var))
		if err != nil {
			return nil, err
		}
		return &core.Forall{CoreNode: node, Var: n.Var, Lo: xs[0], Hi: xs[1], Body: body}, nil
	}
	// Dictionary nodes do not exist before dictionary elaboration; anything
	// else is a Core form this copy does not know, and copying it by reference
	// would share node ids (types) between variants.
	return nil, fmt.Errorf("numeric variant: cannot copy Core node %T", e)
}
