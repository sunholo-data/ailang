package pipeline

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

type effectValueLookup func(uint64) (types.Type, bool)

type latentMaskLookup func(uint64) (types.ApplicationEffects, bool)

type effectCollector struct {
	typeInfo     types.CoreTypeInfo
	declared     map[string]*types.Row
	maskLookup   latentMaskLookup
	valueLookup  effectValueLookup
	lambdaLookup declaredLambdaLookup // source-declared lambda rows (#2238)
	invariant    *error
}

// collectDeclared is collect for a top-level binding's value. For a `func`
// declaration (isFunc: it has a source signature) whose value is its lambda,
// the body's effects — row variables included — are what the declaration is
// checked against, so the own-tail rule below (which is about charging an
// ENCLOSING function) must not apply to it. A module-level `let` of a function
// literal has no signature: the literal's own annotation is checked by
// validateLambdaAnnotations, and the let is charged as an enclosing scope.
func (c *effectCollector) collectDeclared(value core.CoreExpr, isFunc bool) *types.Row {
	if lam, ok := value.(*core.Lambda); ok && isFunc {
		return c.shadow(lam.Params...).collect(lam.Body)
	}
	return c.collect(value)
}

// lambdaOwnTailDropped removes from a lambda's body effects a row variable that
// belongs to the lambda itself: the tail it declares
// (`func(g: int -> () ! {e}) -> () ! {e} { g(1) }`) or a row variable of its
// own parameter types (`func(g: int -> () ! {e}) -> () { g(1) }`). Such a
// variable is the literal's effect polymorphism: every call of the literal
// instantiates it, and the call is charged what it resolves to there —
// exactly as a top-level `! {e}` function's own tail stops at its declaration,
// and as an unannotated parameter's implicit row already was (ImplicitOwnedTail).
// Left in, it surfaced as an unresolved tail of the ENCLOSING function once the
// literal's parameter annotations were enforced (#2238). Labels stay.
func (c *effectCollector) lambdaOwnTailDropped(lam *core.Lambda, body *types.Row) *types.Row {
	if body == nil || body.Tail == nil || !c.lambdaOwnsRow(lam, body.Tail) {
		return body
	}
	out := cloneEffectRow(body)
	out.Tail = nil
	return out
}

func (c *effectCollector) lambdaOwnsRow(lam *core.Lambda, tail *types.RowVar) bool {
	if c.lambdaLookup != nil {
		if declared, ok := c.lambdaLookup(lam.ID()); ok && declared != nil && declared.Tail != nil && declared.Tail.Equals(tail) {
			return true
		}
	}
	t, ok := c.typeInfo.Get(lam.ID())
	if !ok {
		return false
	}
	fn, ok := t.(*types.TFunc2)
	if !ok {
		return false
	}
	for _, p := range fn.Params {
		for _, name := range types.FreeEffectRowVarNames(p) {
			if name == tail.Name {
				return true
			}
		}
	}
	return false
}

func collectRequiredEffects(expr core.CoreExpr, typeInfo types.CoreTypeInfo, declared map[string]*types.Row) *types.Row {
	return (&effectCollector{typeInfo: typeInfo, declared: declared, invariant: new(error)}).collect(expr)
}

// Shadow declarations only inside the lexical scope of a binder. Top-level
// references keep their source rows, avoiding recursive inference contamination.
func (c *effectCollector) shadow(names ...string) *effectCollector {
	child := *c
	child.declared = make(map[string]*types.Row, len(c.declared))
	for k, v := range c.declared {
		child.declared[k] = v
	}
	for _, name := range names {
		delete(child.declared, name)
	}
	return &child
}

func (c *effectCollector) latentEffects(expr core.CoreExpr) *types.Row {
	if v, ok := expr.(*core.Var); ok {
		if row, found := c.declared[v.Name]; found {
			return cloneEffectRow(row)
		}
	}
	if t, ok := c.effectType(expr); ok {
		return extractEffectFromType(t)
	}
	return nil
}

func (c *effectCollector) applicationEffects(app *core.App) types.ApplicationEffects {
	if c.maskLookup == nil {
		return types.ApplicationEffects{}
	}
	record, ok := c.maskLookup(app.ID())
	if c.typeInfo.Has(app.ID()) && (!ok || record.CallRow == nil || record.CallRow.Kind == nil || !record.CallRow.Kind.Equals(types.EffectRow) || (record.CallRow.Tail != nil && !record.CallTailOwned)) && *c.invariant == nil {
		*c.invariant = fmt.Errorf("internal invariant: missing or malformed ApplicationEffects (LatentParamMask) for typed application %d at %s", app.ID(), app.Span())
	}
	return record
}

func recBindingNames(bindings []core.RecBinding) []string {
	names := make([]string, len(bindings))
	for i, b := range bindings {
		names[i] = b.Name
	}
	return names
}

func patternBindingNames(pattern core.CorePattern) []string {
	var names []string
	var walk func(core.CorePattern)
	walk = func(p core.CorePattern) {
		switch p := p.(type) {
		case *core.VarPattern:
			names = append(names, p.Name)
		case *core.ConstructorPattern:
			for _, a := range p.Args {
				walk(a)
			}
		case *core.TuplePattern:
			for _, a := range p.Elements {
				walk(a)
			}
		case *core.ListPattern:
			for _, a := range p.Elements {
				walk(a)
			}
			if p.Tail != nil {
				walk(*p.Tail)
			}
		case *core.RecordPattern:
			for _, a := range p.Fields {
				walk(a)
			}
		}
	}
	walk(pattern)
	return names
}

func (c *effectCollector) effectType(expr core.CoreExpr) (types.Type, bool) {
	if c.valueLookup != nil {
		return c.valueLookup(expr.ID())
	}
	return c.typeInfo.Get(expr.ID())
}

func (c *effectCollector) union(a, b *types.Row) *types.Row {
	if _, err := types.CheckedUnionEffectRows(a, b); err != nil {
		if *c.invariant == nil {
			*c.invariant = err
		}
		return nil
	}
	return unionRequiredEffectRows(a, b)
}
