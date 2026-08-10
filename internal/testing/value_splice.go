package testing

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// valueToLiteral converts an eval.Value to an ast.Expr expression.
//
// It returns (expr, nil) on success. It refuses — with a non-nil error — any
// value kind it has no splice arm for (the default arm), so callers can turn a
// refusal into a loud property failure rather than fabricating a unit literal
// (see m-property-generator-coverage-lane-b1-sprint-plan.md §M2).
func (r *Runner) valueToLiteral(value eval.Value) (ast.Expr, error) {
	switch v := value.(type) {
	case *eval.IntValue:
		return &ast.Literal{
			Kind:  ast.IntLit,
			Value: v.Value,
		}, nil
	case *eval.FloatValue:
		return &ast.Literal{
			Kind:  ast.FloatLit,
			Value: v.Value,
		}, nil
	case *eval.BoolValue:
		return &ast.Literal{
			Kind:  ast.BoolLit,
			Value: v.Value,
		}, nil
	case *eval.StringValue:
		return &ast.Literal{
			Kind:  ast.StringLit,
			Value: v.Value,
		}, nil
	case *eval.ListValue:
		// Convert list elements
		elements := make([]ast.Expr, len(v.Elements))
		for i, elem := range v.Elements {
			lit, err := r.valueToLiteral(elem)
			if err != nil {
				return nil, err
			}
			elements[i] = lit
		}
		return &ast.List{Elements: elements}, nil
	case *eval.UnitValue:
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: struct{}{},
		}, nil
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
			lit, err := r.valueToLiteral(v.Fields[name])
			if err != nil {
				return nil, err
			}
			fields = append(fields, &ast.Field{
				Name:  name,
				Value: lit,
			})
		}
		return &ast.Record{Fields: fields}, nil
	case *eval.TupleValue:
		elements := make([]ast.Expr, len(v.Elements))
		for i, elem := range v.Elements {
			lit, err := r.valueToLiteral(elem)
			if err != nil {
				return nil, err
			}
			elements[i] = lit
		}
		return &ast.Tuple{Elements: elements}, nil
	case *eval.TaggedValue:
		// Arity-split: injectADTConstructors binds nullary constructors to a
		// *eval.TaggedValue rather than a closure, so emitting a FuncCall over
		// a nullary constructor would die at runtime with
		// "cannot apply non-function value: *eval.TaggedValue".
		if len(v.Fields) == 0 {
			return &ast.Identifier{Name: v.CtorName}, nil
		}
		args := make([]ast.Expr, len(v.Fields))
		for i, field := range v.Fields {
			lit, err := r.valueToLiteral(field)
			if err != nil {
				return nil, err
			}
			args[i] = lit
		}
		return &ast.FuncCall{
			Func: &ast.Identifier{Name: v.CtorName},
			Args: args,
		}, nil
	default:
		return nil, fmt.Errorf("no literal splice for generated value of type %T (%s)", value, value.Type())
	}
}
