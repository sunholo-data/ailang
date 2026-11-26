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
	case *core.ListPattern:
		var names []string
		for _, el := range x.Elements {
			names = append(names, patternBinders(el)...)
		}
		if x.Tail != nil {
			names = append(names, patternBinders(*x.Tail)...)
		}
		return names
	case *core.TuplePattern:
		var names []string
		for _, el := range x.Elements {
			names = append(names, patternBinders(el)...)
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

// encodePatternScoped encodes an arm's body or guard with each variable the
// pattern binds referring to its letSymbol, the symbol every pattern encoder
// binds it under. A raw name would collide with a declared accessor or
// function of the same name (`Some(p) => p.p` was a Z3 error), as for let (#215).
func encodePatternScoped(expr core.CoreExpr, p core.CorePattern) (string, error) {
	previous := activeParamRenames
	local := make(map[string]string, len(previous))
	for k, v := range previous {
		local[k] = v
	}
	for _, name := range patternBinders(p) {
		local[name] = letSymbol(name)
	}
	activeParamRenames = local
	defer func() { activeParamRenames = previous }()
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

// argSymbol is the SMT name a define-fun binds a source parameter under: a
// helper's or an unrolled function's parameter named like a record accessor
// (`guidance(s, phase, target)` reading `t.target`) otherwise shadows the
// accessor and Z3 rejects the application. Same hygiene as letSymbol ($l_)
// and captureSymbol ($c_).
func argSymbol(name string) string {
	return "$a_" + name
}

// withArgRenames runs encode with each of params' source names bound to its
// argSymbol, and returns params renamed to match, for the define-fun header.
func withArgRenames(params []FunctionParam, encode func() (string, error)) ([]FunctionParam, string, error) {
	previous := activeParamRenames
	local := make(map[string]string, len(previous)+len(params))
	for k, v := range previous {
		local[k] = v
	}
	renamed := make([]FunctionParam, len(params))
	for i, p := range params {
		renamed[i] = p
		renamed[i].Name = argSymbol(p.Name)
		local[p.Name] = renamed[i].Name
	}
	activeParamRenames = local
	defer func() { activeParamRenames = previous }()
	out, err := encode()
	return renamed, out, err
}
