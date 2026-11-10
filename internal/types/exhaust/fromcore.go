package exhaust

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// FromCore converts a Core pattern to the package's form. A constructor's
// TypeName is the elaborator's resolved ADT name (core.ConstructorPattern);
// the caller's Universe decides what identity that is.
func FromCore(p core.CorePattern) (Pattern, error) {
	switch p := p.(type) {
	case *core.WildcardPattern, *core.VarPattern:
		return Wild{}, nil
	case *core.LitPattern:
		return Lit{Value: p.Value}, nil
	case *core.ConstructorPattern:
		args, err := fromCoreAll(p.Args)
		if err != nil {
			return nil, err
		}
		return Ctor{TypeName: p.TypeName, Name: p.Name, Args: args}, nil
	case *core.TuplePattern:
		elems, err := fromCoreAll(p.Elements)
		if err != nil {
			return nil, err
		}
		return Tuple{Elems: elems}, nil
	case *core.ListPattern:
		elems, err := fromCoreAll(p.Elements)
		if err != nil {
			return nil, err
		}
		l := List{Elems: elems}
		if p.Tail != nil {
			if l.Rest, err = FromCore(*p.Tail); err != nil {
				return nil, err
			}
		}
		return l, nil
	case *core.RecordPattern:
		fields := make(map[string]Pattern, len(p.Fields))
		for f, fp := range p.Fields {
			q, err := FromCore(fp)
			if err != nil {
				return nil, err
			}
			fields[f] = q
		}
		return Record{Fields: fields}, nil
	}
	return nil, fmt.Errorf("exhaust: unknown Core pattern %T", p)
}

func fromCoreAll(ps []core.CorePattern) ([]Pattern, error) {
	out := make([]Pattern, len(ps))
	for i, p := range ps {
		q, err := FromCore(p)
		if err != nil {
			return nil, err
		}
		out[i] = q
	}
	return out, nil
}
