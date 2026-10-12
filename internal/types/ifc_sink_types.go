package types

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
)

// sinkShape resolves local non-generic aliases and label wrappers with a
// cycle guard. No arbitrary alias-depth limit may turn a declared sink invisible.
func (ts *ifcTypes) sinkShape(t ast.Type) ast.Type {
	seen := map[string]bool{}
	for {
		switch x := t.(type) {
		case *ast.LabelledType:
			t = x.Base
		case *ast.SimpleType:
			if seen[x.Name] {
				return nil
			}
			seen[x.Name] = true
			td := ts.decls[x.Name]
			if td == nil || len(td.TypeParams) > 0 {
				return t
			}
			switch def := td.Definition.(type) {
			case *ast.TypeAlias:
				t = def.Target
			case *ast.RecordType:
				return def
			default:
				return t
			}
		default:
			return t
		}
	}
}

func (ts *ifcTypes) sinkFunction(t ast.Type) *ast.FuncType {
	fn, _ := ts.sinkShape(t).(*ast.FuncType)
	return fn
}

// sinkLabels resolves aliases at the receiving position. A function's domain
// is checked at application; it is never a sink on the function value itself.
func (ts *ifcTypes) sinkLabels(t ast.Type) []string {
	seen := map[string]bool{}
	var labels []string
	for {
		switch x := t.(type) {
		case *ast.LabelledType:
			if x.Refinement != nil {
				labels = append(labels, x.Refinement.NotLabel)
			}
			t = x.Base
		case *ast.SimpleType:
			if seen[x.Name] {
				return labels
			}
			seen[x.Name] = true
			td := ts.decls[x.Name]
			if td == nil || len(td.TypeParams) > 0 {
				return labels
			}
			alias, ok := td.Definition.(*ast.TypeAlias)
			if !ok {
				return labels
			}
			t = alias.Target
		default:
			return labels
		}
	}
}

func (c *ifcChecker) checkSinkLabelAt(pos ast.Pos, name string, t ast.Type, label Label) {
	if c.silent {
		return
	}
	for _, not := range c.types.sinkLabels(t) {
		if CheckSinkLabel(label, not) != nil {
			c.errs = append(c.errs, &TypeCheckError{Kind: SinkRefinementError, Position: pos.String(),
				Message:    fmt.Sprintf("information-flow violation: value labelled %s reaches %s which forbids it ({not %s})", label, name, not),
				Suggestion: "declassify the value through an authorized function before writing to this sink"})
		}
	}
}

// checkSinkTransfer checks receiving fields using a conservative full label.
// Literals are checked field by field by handOff; opaque records cannot hide a
// labelled constituent by being assigned a refined type.
func (c *ifcChecker) checkSinkTransfer(pos ast.Pos, t ast.Type, label Label, seen map[string]bool) {
	if t == nil {
		return
	}
	if key := t.String(); seen[key] {
		return
	} else {
		seen[key] = true
		defer delete(seen, key)
	}
	c.checkSinkLabelAt(pos, "declared value", t, label)
	if rt := c.types.recordTypeOf(t); rt != nil {
		for _, f := range rt.Fields {
			if f != nil {
				c.checkSinkTransfer(pos, f.Type, label, seen)
			}
		}
	}
}

// validateSinkType is separate from deepLabel: source-label traversal of a
// function intentionally follows returns only; sink validation must see domains
// too. Unsupported positions fail loudly even in unused declarations.
func (c *ifcChecker) validateSinkType(t ast.Type, unsupported string, inDomain bool, seen map[string]bool) {
	if t == nil || c.silent {
		return
	}
	switch tt := t.(type) {
	case *ast.LabelledType:
		if tt.Refinement != nil && unsupported != "" {
			c.errs = append(c.errs, &TypeCheckError{Kind: SinkRefinementError, Position: tt.Pos.String(),
				Message:    fmt.Sprintf("{not %s} in %s is not enforceable here", tt.Refinement.NotLabel, unsupported),
				Suggestion: "use a plain record field or a first-order function domain; refine the producer's declared return instead"})
		}
		c.validateSinkType(tt.Base, unsupported, inDomain, seen)
	case *ast.SimpleType:
		key := tt.Name + "/" + unsupported
		if seen[key] {
			if unsupported == "" {
				c.validateSinkType(t, "recursive type", inDomain, seen)
			}
			return
		}
		seen[key] = true
		defer delete(seen, key)
		if td := c.types.decls[tt.Name]; td != nil {
			reason := unsupported
			if len(td.TypeParams) > 0 {
				reason = "generic type"
			}
			if _, ok := td.Definition.(*ast.AlgebraicType); ok {
				reason = "ADT constructor field"
			}
			for _, body := range declBodyTypes(td) {
				c.validateSinkType(body, reason, inDomain, seen)
			}
		}
	case *ast.RecordType:
		for _, f := range tt.Fields {
			if f != nil {
				c.validateSinkType(f.Type, unsupported, inDomain, seen)
			}
		}
	case *ast.FuncType:
		reason := unsupported
		if inDomain {
			reason = "higher-order function domain"
		}
		for _, param := range tt.Params {
			c.validateSinkType(param, reason, true, seen)
		}
		c.validateSinkType(tt.Return, "function-typed value's codomain", false, seen)
	case *ast.ListType:
		c.validateSinkType(tt.Element, "list element", inDomain, seen)
	case *ast.ArrayType:
		c.validateSinkType(tt.Element, "array element", inDomain, seen)
	case *ast.TupleType:
		for _, el := range tt.Elements {
			c.validateSinkType(el, "tuple element", inDomain, seen)
		}
	case *ast.TypeApp:
		c.validateSinkType(&ast.SimpleType{Name: tt.Constructor, Pos: tt.Pos}, "generic type application", inDomain, seen)
		for _, arg := range tt.Args {
			c.validateSinkType(arg, "generic type application", inDomain, seen)
		}
	}
}

func (c *ifcChecker) validateSinkAnnotation(t ast.Type) {
	c.validateSinkType(t, "", false, map[string]bool{})
}

// checkValueSinks complements the single argument flow walk. Label-only probes
// are silent so nested calls are not diagnosed a second time.
func (c *ifcChecker) checkValueSinks(e ast.Expr, t ast.Type, env ifcEnv) {
	if c.silent || e == nil {
		return
	}
	c.checkCallableTransfer(e.Position(), c.staticType(e, env), t)
	if t == nil {
		return
	}
	switch x := e.(type) {
	case *ast.Record:
		if c.types.recordTypeOf(t) != nil {
			c.checkStructuredSinkRoot(e, t, env)
			for _, f := range x.Fields {
				c.checkValueSinks(f.Value, c.types.fieldType(t, f.Name), env)
			}
			return
		}
	case *ast.RecordUpdate:
		c.checkStructuredSinkRoot(e, t, env)
		c.checkValueSinks(x.Base, t, env)
		for _, f := range x.Fields {
			c.checkValueSinks(f.Value, c.types.fieldType(t, f.Name), env)
		}
		return
	case *ast.Block:
		cur := env
		for i, sub := range x.Exprs {
			if i == len(x.Exprs)-1 {
				c.checkValueSinks(sub, t, cur)
				break
			}
			c.silent = true
			switch let := sub.(type) {
			case *ast.Let:
				if let.Body == nil {
					cur = extendEnv(cur, let.Name, c.bindValue(let.Value, let.Type, cur))
				}
			case *ast.LetRec:
				if let.Body == nil {
					cur = extendEnv(cur, let.Name, c.bindValue(let.Value, let.Type, cur))
				}
			}
			c.silent = false
		}
		return
	case *ast.Let:
		if x.Body != nil {
			c.silent = true
			v := c.bindValue(x.Value, x.Type, env)
			c.silent = false
			c.checkValueSinks(x.Body, t, extendEnv(env, x.Name, v))
			return
		}
	case *ast.Match:
		c.silent = true
		label := c.labelOf(x.Expr, env)
		c.silent = false
		for _, cs := range x.Cases {
			arm := env
			for _, name := range patternVars(cs.Pattern) {
				arm = extendEnv(arm, name, ifcVar{label: label})
			}
			arm = c.bindPatternTypes(cs.Pattern, c.staticType(x.Expr, env), arm)
			c.checkValueSinks(cs.Body, t, arm)
		}
		return
	case *ast.If:
		c.checkValueSinks(x.Then, t, env)
		c.checkValueSinks(x.Else, t, env)
		return
	}
	c.silent = true
	label := c.labelOf(e, env)
	c.silent = false
	c.checkSinkTransfer(e.Position(), t, label, map[string]bool{})
}

func (c *ifcChecker) checkStructuredSinkRoot(e ast.Expr, t ast.Type, env ifcEnv) {
	if c.silent || len(c.types.sinkLabels(t)) == 0 {
		return
	}
	c.silent = true
	label := c.labelOf(e, env)
	c.silent = false
	c.checkSinkLabelAt(e.Position(), "record value", t, label)
}
