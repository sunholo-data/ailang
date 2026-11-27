package pipeline

import "github.com/sunholo-data/ailang/internal/core"

// lowerCompositeOrDict lowers container, record-update, dictionary, and
// bounded-quantifier CoreExpr nodes (#2323). Extracted from op_lowering.go so
// OpLowerer covers all 22 CoreExpr variants while keeping op_lowering.go < 715L.
func (l *OpLowerer) lowerCompositeOrDict(expr core.CoreExpr) (core.CoreExpr, bool) {
	switch e := expr.(type) {
	case *core.Record:
		fields := make(map[string]core.CoreExpr, len(e.Fields))
		for k, v := range e.Fields {
			fields[k] = l.lowerExpr(v)
		}
		return &core.Record{
			CoreNode: e.CoreNode,
			Fields:   fields,
		}, true

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: e.CoreNode,
			Record:   l.lowerExpr(e.Record),
			Field:    e.Field,
		}, true

	case *core.RecordUpdate:
		updates := make(map[string]core.CoreExpr, len(e.Updates))
		for k, v := range e.Updates {
			updates[k] = l.lowerExpr(v)
		}
		return &core.RecordUpdate{
			CoreNode: e.CoreNode,
			Base:     l.lowerExpr(e.Base),
			Updates:  updates,
		}, true

	case *core.List:
		return &core.List{
			CoreNode: e.CoreNode,
			Elements: l.lowerExprs(e.Elements),
		}, true

	case *core.Array:
		return &core.Array{
			CoreNode: e.CoreNode,
			Elements: l.lowerExprs(e.Elements),
		}, true

	case *core.Tuple:
		return &core.Tuple{
			CoreNode: e.CoreNode,
			Elements: l.lowerExprs(e.Elements),
		}, true

	case *core.DictAbs:
		names := make([]string, len(e.Params))
		for i, p := range e.Params {
			names[i] = p.Name
		}
		restore := l.shadow(names...)
		body := l.lowerExpr(e.Body)
		restore()
		return &core.DictAbs{
			CoreNode: e.CoreNode,
			Params:   e.Params,
			Body:     body,
		}, true

	case *core.DictApp:
		return &core.DictApp{
			CoreNode: e.CoreNode,
			Dict:     l.lowerExpr(e.Dict),
			Method:   e.Method,
			Args:     l.lowerExprs(e.Args),
		}, true

	case *core.Forall:
		lo := l.lowerExpr(e.Lo)
		hi := l.lowerExpr(e.Hi)
		restore := l.shadow(e.Var)
		body := l.lowerExpr(e.Body)
		restore()
		return &core.Forall{
			CoreNode: e.CoreNode,
			Var:      e.Var,
			Lo:       lo,
			Hi:       hi,
			Body:     body,
		}, true
	}
	return nil, false
}
