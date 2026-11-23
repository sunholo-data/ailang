package smt

import "github.com/sunholo-data/ailang/internal/core"

// Local binders shadow the symbolic parameter constants. Keep expression
// encoding lexical: a parameter's $p_ rename cannot capture a let or pattern var.
func withoutParameterRenames(names []string) func() {
	previous := activeParamRenames
	local := make(map[string]string, len(previous))
	for name, value := range previous {
		local[name] = value
	}
	for _, name := range names {
		delete(local, name)
	}
	activeParamRenames = local
	return func() { activeParamRenames = previous }
}
func patternBinders(p core.CorePattern) []string {
	switch x := p.(type) {
	case *core.VarPattern:
		return []string{x.Name}
	case *core.ConstructorPattern:
		var names []string
		for _, arg := range x.Args {
			names = append(names, patternBinders(arg)...)
		}
		return names
	case *core.RecordPattern:
		var names []string
		for _, arg := range x.Fields {
			names = append(names, patternBinders(arg)...)
		}
		return names
	}
	return nil
}
func encodePatternScoped(expr core.CoreExpr, p core.CorePattern) (string, error) {
	restore := withoutParameterRenames(patternBinders(p))
	defer restore()
	return EncodeExpr(expr)
}

// letSymbol is the SMT symbol a Core let binds name under. The $l_ prefix keeps
// a local from colliding with a declared accessor, constructor or function of
// the same name (#215); $p_ is the parameter prefix, so the two cannot clash.
func letSymbol(name string) string {
	return "$l_" + name
}

// encodeLetRenamed encodes expr with name referring to the let-bound symbol.
func encodeLetRenamed(expr core.CoreExpr, name, symbol string) (string, error) {
	previous := activeParamRenames
	local := make(map[string]string, len(previous)+1)
	for k, v := range previous {
		local[k] = v
	}
	local[name] = symbol
	activeParamRenames = local
	defer func() { activeParamRenames = previous }()
	return EncodeExpr(expr)
}
