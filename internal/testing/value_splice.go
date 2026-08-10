package testing

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// valueToLiteral converts an eval.Value to an ast.Expr expression.
func (r *Runner) valueToLiteral(value eval.Value) ast.Expr {
	switch v := value.(type) {
	case *eval.IntValue:
		return &ast.Literal{
			Kind:  ast.IntLit,
			Value: v.Value,
		}
	case *eval.FloatValue:
		return &ast.Literal{
			Kind:  ast.FloatLit,
			Value: v.Value,
		}
	case *eval.BoolValue:
		return &ast.Literal{
			Kind:  ast.BoolLit,
			Value: v.Value,
		}
	case *eval.StringValue:
		return &ast.Literal{
			Kind:  ast.StringLit,
			Value: v.Value,
		}
	case *eval.ListValue:
		// Convert list elements
		elements := make([]ast.Expr, len(v.Elements))
		for i, elem := range v.Elements {
			elements[i] = r.valueToLiteral(elem)
		}
		return &ast.List{Elements: elements}
	case *eval.UnitValue:
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: struct{}{},
		}
	case *eval.RecordValue:
		// Iterate field names in sorted order so the produced AST is
		// deterministic (RecordValue.Fields is a Go map and ranging over it
		// directly would produce a nondeterministic field order).
		names := make([]string, 0, len(v.Fields))
		for name := range v.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		fields := make([]*ast.Field, 0, len(names))
		for _, name := range names {
			fields = append(fields, &ast.Field{
				Name:  name,
				Value: r.valueToLiteral(v.Fields[name]),
			})
		}
		return &ast.Record{Fields: fields}
	case *eval.TupleValue:
		elements := make([]ast.Expr, len(v.Elements))
		for i, elem := range v.Elements {
			elements[i] = r.valueToLiteral(elem)
		}
		return &ast.Tuple{Elements: elements}
	case *eval.TaggedValue:
		// Arity-split: injectADTConstructors binds nullary constructors to a
		// *eval.TaggedValue rather than a closure, so emitting a FuncCall over
		// a nullary constructor would die at runtime with
		// "cannot apply non-function value: *eval.TaggedValue".
		if len(v.Fields) == 0 {
			return &ast.Identifier{Name: v.CtorName}
		}
		args := make([]ast.Expr, len(v.Fields))
		for i, field := range v.Fields {
			args[i] = r.valueToLiteral(field)
		}
		return &ast.FuncCall{
			Func: &ast.Identifier{Name: v.CtorName},
			Args: args,
		}
	default:
		// TODO(M2): this silent default becomes a loud error in M2 (see
		// m-property-generator-coverage-lane-b1-sprint-plan.md §M2)
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: struct{}{},
		}
	}
}
