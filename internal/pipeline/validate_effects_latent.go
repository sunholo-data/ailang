package pipeline

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

type effectValueLookup func(uint64) (types.Type, bool)

type latentMaskLookup func(uint64) ([]bool, bool)

type effectCollector struct {
	typeInfo    types.CoreTypeInfo
	declared    map[string]*types.Row
	maskLookup  latentMaskLookup
	valueLookup effectValueLookup
	invariant   *error
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

func (c *effectCollector) applicationMask(app *core.App) []bool {
	if c.maskLookup == nil {
		return nil
	} // Legacy standalone validator callers.
	mask, ok := c.maskLookup(app.ID())
	if !ok && c.typeInfo.Has(app.ID()) && *c.invariant == nil {
		*c.invariant = fmt.Errorf("internal invariant: missing LatentParamMask for typed application %d at %s", app.ID(), app.Span())
	}
	return mask
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
