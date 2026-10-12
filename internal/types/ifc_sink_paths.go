package types

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
)

type callableSink struct{ path, label string }

// callableSinks records function-domain obligations separately from ordinary
// record write sinks. A function reference must not lose these when forwarded.
func (ts *ifcTypes) callableSinks(t ast.Type, path string, domain bool, seen map[string]bool) []callableSink {
	if t == nil {
		return nil
	}
	var out []callableSink
	switch x := t.(type) {
	case *ast.SimpleType:
		if seen[x.Name] {
			return nil
		}
		seen[x.Name] = true
		defer delete(seen, x.Name)
		if td := ts.decls[x.Name]; td != nil {
			for _, body := range declBodyTypes(td) {
				out = append(out, ts.callableSinks(body, path, domain, seen)...)
			}
		}
	case *ast.LabelledType:
		if domain && x.Refinement != nil {
			out = append(out, callableSink{path, x.Refinement.NotLabel})
		}
		out = append(out, ts.callableSinks(x.Base, path, domain, seen)...)
	case *ast.RecordType:
		for _, f := range x.Fields {
			if f != nil {
				out = append(out, ts.callableSinks(f.Type, path+"."+f.Name, domain, seen)...)
			}
		}
	case *ast.FuncType:
		for i, p := range x.Params {
			out = append(out, ts.callableSinks(p, fmt.Sprintf("%s.arg%d", path, i), true, seen)...)
		}
	}
	return out
}

func (c *ifcChecker) checkCallableTransfer(pos ast.Pos, source, target ast.Type) {
	if c.silent {
		return
	}
	have := c.types.callableSinks(source, "", false, map[string]bool{})
	want := c.types.callableSinks(target, "", false, map[string]bool{})
	for _, sink := range have {
		found := false
		for _, other := range want {
			if sink == other {
				found = true
				break
			}
		}
		if !found {
			c.errs = append(c.errs, &TypeCheckError{Kind: SinkRefinementError, Position: pos.String(),
				Message:    fmt.Sprintf("{not %s} on function domain %s is not enforceable after this transfer", sink.label, sink.path),
				Suggestion: "preserve the function domain refinement in the receiving parameter or annotation"})
		}
	}
}

// Record destructuring retains the known field types as well as the existing
// conservative scrutinee label. Tuple/list/ADT sink annotations are rejected.
func (c *ifcChecker) bindPatternTypes(p ast.Pattern, t ast.Type, env ifcEnv) ifcEnv {
	switch x := p.(type) {
	case *ast.Identifier:
		v := env[x.Name]
		v.typ = t
		return extendEnv(env, x.Name, v)
	case *ast.RecordPattern:
		for _, f := range x.Fields {
			env = c.bindPatternTypes(f.Pattern, c.types.fieldType(t, f.Name), env)
		}
	}
	return env
}

func functionAnnotation(params []*ast.Param, result ast.Type, pos ast.Pos) ast.Type {
	types := make([]ast.Type, len(params))
	for i, p := range params {
		types[i] = p.Type
		if types[i] == nil {
			types[i] = &ast.TypeVar{Name: "_"}
		}
	}
	if result == nil {
		result = &ast.TypeVar{Name: "_"}
	}
	return &ast.FuncType{Params: types, Return: result, Pos: pos}
}
