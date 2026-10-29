package golang

import "github.com/sunholo-data/ailang/internal/core"

// localReferences records lexical identity before Block lowering, which preserves
// Var pointers. Names alone are insufficient: another branch can bind the same
// name while an unbound first-class builtin still needs registry resolution.
func localReferences(prog *core.Program) map[*core.Var]bool {
	refs := make(map[*core.Var]bool)
	bound := make(map[string]int)
	var walk func(core.CoreExpr)
	with := func(names []string, body func()) {
		for _, name := range names {
			bound[name]++
		}
		body()
		for _, name := range names {
			bound[name]--
		}
	}
	walk = func(expr core.CoreExpr) {
		switch e := expr.(type) {
		case *core.Var:
			if bound[e.Name] > 0 {
				refs[e] = true
			}
		case *core.Lambda:
			with(e.Params, func() { walk(e.Body) })
		case *core.Let:
			walk(e.Value)
			with([]string{e.Name}, func() { walk(e.Body) })
		case *core.LetRec:
			names := make([]string, len(e.Bindings))
			for i, b := range e.Bindings {
				names[i] = b.Name
			}
			with(names, func() {
				for _, b := range e.Bindings {
					walk(b.Value)
				}
				walk(e.Body)
			})
		case *core.App:
			walk(e.Func)
			for _, arg := range e.Args {
				walk(arg)
			}
		case *core.If:
			walk(e.Cond)
			walk(e.Then)
			walk(e.Else)
		case *core.Match:
			walk(e.Scrutinee)
			for _, arm := range e.Arms {
				with(patternBoundNames(arm.Pattern), func() { walk(arm.Guard); walk(arm.Body) })
			}
		case *core.BinOp:
			walk(e.Left)
			walk(e.Right)
		case *core.UnOp:
			walk(e.Operand)
		case *core.Record:
			for _, value := range e.Fields {
				walk(value)
			}
		case *core.RecordAccess:
			walk(e.Record)
		case *core.RecordUpdate:
			walk(e.Base)
			for _, value := range e.Updates {
				walk(value)
			}
		case *core.List:
			for _, value := range e.Elements {
				walk(value)
			}
		case *core.Array:
			for _, value := range e.Elements {
				walk(value)
			}
		case *core.Tuple:
			for _, value := range e.Elements {
				walk(value)
			}
		case *core.Intrinsic:
			for _, arg := range e.Args {
				walk(arg)
			}
		case *core.DictApp:
			walk(e.Dict)
			for _, arg := range e.Args {
				walk(arg)
			}
		case *core.DictAbs:
			names := make([]string, len(e.Params))
			for i, p := range e.Params {
				names[i] = p.Name
			}
			with(names, func() { walk(e.Body) })
		case *core.Forall:
			walk(e.Lo)
			walk(e.Hi)
			with([]string{e.Var}, func() { walk(e.Body) })
		}
	}
	for _, decl := range prog.Decls {
		// Top-level Let/LetRec bindings are emitted as global declarations, not
		// lexical locals. Their residual Body can share later declaration nodes;
		// traversing it with the global name bound would mislabel those references.
		switch d := decl.(type) {
		case *core.Let:
			walk(d.Value)
		case *core.LetRec:
			for _, binding := range d.Bindings {
				walk(binding.Value)
			}
		default:
			walk(decl)
		}
	}
	return refs
}

func patternBoundNames(p core.CorePattern) []string {
	var names []string
	var walk func(core.CorePattern)
	walk = func(p core.CorePattern) {
		switch p := p.(type) {
		case *core.VarPattern:
			if p.Name != "_" {
				names = append(names, p.Name)
			}
		case *core.ConstructorPattern:
			for _, arg := range p.Args {
				walk(arg)
			}
		case *core.ListPattern:
			for _, arg := range p.Elements {
				walk(arg)
			}
			if p.Tail != nil {
				walk(*p.Tail)
			}
		case *core.TuplePattern:
			for _, arg := range p.Elements {
				walk(arg)
			}
		case *core.RecordPattern:
			for _, arg := range p.Fields {
				walk(arg)
			}
		}
	}
	walk(p)
	return names
}

func (g *Generator) generateLocalCall(app *core.App) error {
	g.write("CallFunc(")
	if err := g.generateExpr(app.Func); err != nil {
		return err
	}
	for _, arg := range app.Args {
		g.write(", ")
		if err := g.generateExpr(arg); err != nil {
			return err
		}
	}
	g.write(")")
	return nil
}
