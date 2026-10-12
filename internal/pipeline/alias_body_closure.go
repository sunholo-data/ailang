package pipeline

import (
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// aliasBodyCloser rebuilds types in the defining module's alias environment.
// Only acyclic expansions are memoized: a cycle's opaque reference depends on
// the current root, and must never leak into a later root's expansion.
type aliasBodyCloser struct {
	aliases map[string]types.Type
	params  map[string][]string
	active  map[string]bool
	memo    map[string]types.Type
	cycles  int
}

func newAliasBodyCloser(aliases map[string]types.Type, params map[string][]string) *aliasBodyCloser {
	return &aliasBodyCloser{aliases: aliases, params: params, active: make(map[string]bool), memo: make(map[string]types.Type)}
}

func (c *aliasBodyCloser) alias(name string) types.Type {
	if c.active[name] {
		c.cycles++
		return &types.TCon{Name: name}
	}
	if t, ok := c.memo[name]; ok {
		return t
	}
	c.active[name] = true
	before := c.cycles
	t := c.walk(c.aliases[name])
	delete(c.active, name)
	if r, ok := t.(*types.TRecord); ok && r.TypeName == "" {
		copy := *r
		copy.TypeName = name
		t = &copy
	}
	if before == c.cycles {
		c.memo[name] = t
	}
	return t
}

func (c *aliasBodyCloser) slice(in []types.Type) []types.Type {
	if in == nil {
		return nil
	}
	out := make([]types.Type, len(in))
	for i, t := range in {
		out[i] = c.walk(t)
	}
	return out
}

func (c *aliasBodyCloser) fields(in map[string]types.Type) map[string]types.Type {
	if in == nil {
		return nil
	}
	out := make(map[string]types.Type, len(in))
	for _, k := range sortedTypeKeys(in) {
		out[k] = c.walk(in[k])
	}
	return out
}

func (c *aliasBodyCloser) row(in *types.Row) *types.Row {
	if in == nil {
		return nil
	}
	copy := *in
	copy.Labels = c.fields(in.Labels)
	// Tail, kind, budgets, refinements and provenance are unchanged metadata.
	return &copy
}

// Keep this switch aligned with every types.Type Substitute implementor.
// The source-enumeration test fails when a new concrete type needs coverage.
func (c *aliasBodyCloser) walk(t types.Type) types.Type {
	switch t := t.(type) {
	case nil:
		return nil
	case *types.TCon:
		if _, ok := c.aliases[t.Name]; ok && len(c.params[t.Name]) == 0 {
			return c.alias(t.Name)
		}
		return t
	case *types.TVar, *types.TVar2, *types.RowVar:
		return t
	case *types.TList:
		return &types.TList{Element: c.walk(t.Element)}
	case *types.TArray:
		return &types.TArray{Element: c.walk(t.Element)}
	case *types.TMap:
		return &types.TMap{Key: c.walk(t.Key), Value: c.walk(t.Value)}
	case *types.TTuple:
		return &types.TTuple{Elements: c.slice(t.Elements)}
	case *types.TRecord:
		return &types.TRecord{Fields: c.fields(t.Fields), Row: c.walk(t.Row), TypeName: t.TypeName}
	case *types.TRecordOpen:
		return &types.TRecordOpen{Fields: c.fields(t.Fields), Row: c.walk(t.Row)}
	case *types.TApp:
		// Applied heads stay opaque: polymorphic aliases require substitution, which
		// is deliberately deferred. Closing arguments is safe independently.
		return &types.TApp{Constructor: t.Constructor, Args: c.slice(t.Args)}
	case *types.Row:
		return c.row(t)
	case *types.TFunc2:
		// Preserve effect-contract metadata while closing the structural fields.
		copy := *t
		copy.Params, copy.Return, copy.EffectRow = c.slice(t.Params), c.walk(t.Return), c.row(t.EffectRow)
		return &copy
	case *types.TRecord2:
		return &types.TRecord2{Row: c.row(t.Row)}
	case *types.TLabelled:
		return &types.TLabelled{Inner: c.walk(t.Inner), L: t.L}
	default:
		// Unknown types remain opaque, rather than guessing at their semantics.
		return t
	}
}

// closeInterfaceTypes operates on copies of shared schemes/constructor entries.
// Local aliases include private dependencies and override post-shadow imports.
func closeInterfaceTypes(ifc *iface.Iface, imported, local map[string]types.Type, params map[string][]string) {
	env := make(map[string]types.Type, len(imported)+len(local))
	for name, t := range imported {
		env[name] = t
	}
	for name, t := range local {
		env[name] = t
	}
	c := newAliasBodyCloser(env, params)
	for _, name := range sortedTypeKeys(ifc.TypeAliases) {
		// Expand the root even when parameterized; its body retains bound TVars.
		if _, ok := env[name]; ok {
			ifc.AddTypeAlias(name, c.alias(name))
		}
	}
	for name, item := range ifc.Exports {
		if item.Type == nil {
			continue
		}
		copyItem, copyScheme := *item, *item.Type
		copyScheme.Type = c.walk(item.Type.Type)
		copyItem.Type = &copyScheme
		ifc.Exports[name] = &copyItem
	}
	for name, ctor := range ifc.Constructors {
		copy := *ctor
		copy.FieldTypes = c.slice(ctor.FieldTypes)
		copy.ResultType = c.walk(ctor.ResultType)
		ifc.Constructors[name] = &copy
	}
}
