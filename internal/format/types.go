package format

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// types.go prints type-annotation nodes to canonical source. Types appear in
// parameter annotations, return types, let/binder annotations, method
// signatures, type aliases, and constructor fields.

// typeString renders any ast.Type to a canonical source string, or errors on a
// nil-required child or unknown concrete type.
func (p *printer) typeString(t ast.Type) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil type node")
	}
	switch n := t.(type) {
	case *ast.SimpleType:
		return n.Name, nil
	case *ast.TypeVar:
		return n.Name, nil
	case *ast.ListType:
		el, err := p.typeString(n.Element)
		if err != nil {
			return "", err
		}
		return "[" + el + "]", nil
	case *ast.ArrayType:
		el, err := p.typeString(n.Element)
		if err != nil {
			return "", err
		}
		return "Array[" + el + "]", nil
	case *ast.TupleType:
		parts, err := p.typeList(n.Elements)
		if err != nil {
			return "", err
		}
		return "(" + strings.Join(parts, ", ") + ")", nil
	case *ast.TypeApp:
		args, err := p.typeList(n.Args)
		if err != nil {
			return "", err
		}
		return n.Constructor + "[" + strings.Join(args, ", ") + "]", nil
	case *ast.FuncType:
		return p.funcTypeString(n)
	case *ast.RecordType:
		return p.recordTypeString(n)
	case *ast.LabelledType:
		return p.labelledTypeString(n)
	default:
		return "", fmt.Errorf("unsupported type node: %T", t)
	}
}

// typeList renders a slice of types, propagating the first error.
func (p *printer) typeList(ts []ast.Type) ([]string, error) {
	parts := make([]string, len(ts))
	for i, t := range ts {
		s, err := p.typeString(t)
		if err != nil {
			return nil, err
		}
		parts[i] = s
	}
	return parts, nil
}

// funcTypeString renders a function type `(P1, P2) -> R ! {E}`.
func (p *printer) funcTypeString(n *ast.FuncType) (string, error) {
	params, err := p.typeList(n.Params)
	if err != nil {
		return "", err
	}
	ret, err := p.typeString(n.Return)
	if err != nil {
		return "", err
	}
	eff := ast.FormatEffects(n.Effects)
	if eff != "" {
		eff = " " + eff
	}
	return "(" + strings.Join(params, ", ") + ") -> " + ret + eff, nil
}

// recordTypeString renders a record type `{ a: T, b: U }` or open `{ a: T | r }`.
func (p *printer) recordTypeString(n *ast.RecordType) (string, error) {
	fields := make([]string, len(n.Fields))
	for i, f := range n.Fields {
		ft, err := p.typeString(f.Type)
		if err != nil {
			return "", err
		}
		fields[i] = f.Name + ": " + ft
	}
	body := strings.Join(fields, ", ")
	if n.Row != nil {
		if body == "" {
			return "{ | " + n.Row.Name + " }", nil
		}
		return "{ " + body + " | " + n.Row.Name + " }", nil
	}
	if body == "" {
		return "{}", nil
	}
	return "{ " + body + " }", nil
}

// labelledTypeString renders IFC label / refinement syntax `T<label>` / `T{not L}`.
func (p *printer) labelledTypeString(n *ast.LabelledType) (string, error) {
	base, err := p.typeString(n.Base)
	if err != nil {
		return "", err
	}
	if n.Label != nil {
		return base + "<" + n.Label.Name + ">", nil
	}
	if n.Refinement != nil {
		return base + "{not " + n.Refinement.NotLabel + "}", nil
	}
	return base, nil
}
