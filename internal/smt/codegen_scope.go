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
func encodeLetScoped(expr core.CoreExpr, name string) (string, error) {
	restore := withoutParameterRenames([]string{name})
	defer restore()
	return EncodeExpr(expr)
}
