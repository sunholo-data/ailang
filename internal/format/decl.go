package format

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// decl.go prints top-level declaration nodes: functions, type declarations,
// type classes, instances, test/property declarations, and top-level bindings.

// decl renders a single top-level declaration into the writer. It errors on a
// nil-required child or an unknown concrete declaration node.
func (p *printer) decl(n ast.Node) error {
	if n == nil {
		return fmt.Errorf("nil declaration node")
	}
	switch d := n.(type) {
	case *ast.FuncDecl:
		return p.funcDecl(d)
	case *ast.TypeDecl:
		return p.typeDecl(d)
	case *ast.TypeClass:
		return p.typeClass(d)
	case *ast.Instance:
		return p.instance(d)
	case *ast.TestDecl:
		return p.testDecl(d)
	case *ast.PropertyDecl:
		return p.propertyDecl(d)
	case *ast.AssertStmt:
		return p.assertExpr(d)
	case *ast.Let:
		// Top-level binding: `let name = value` (nil Body). A non-nil Body means an
		// explicit let..in used as a top-level expression statement.
		return p.topLevelLet(d)
	case ast.Expr:
		// A bare top-level expression statement (script-style file).
		return p.expr(d, precLowest)
	default:
		return fmt.Errorf("unsupported declaration node: %T", n)
	}
}

func (p *printer) funcDecl(d *ast.FuncDecl) error {
	// Leading annotations, one per line.
	for _, a := range d.Annotations {
		if err := p.annotation(a); err != nil {
			return err
		}
		p.w.hardline()
	}
	if d.IsExport {
		p.w.write("export ")
	}
	if d.IsPure {
		p.w.write("pure ")
	}
	if d.IsExtern {
		p.w.write("extern ")
	}
	p.w.write("func " + d.Name)
	if len(d.TypeParams) > 0 {
		p.w.write("[" + strings.Join(d.TypeParams, ", ") + "]")
	}
	if err := p.declParams(d.Params); err != nil {
		return err
	}
	if d.ReturnType != nil {
		rt, err := p.typeString(d.ReturnType)
		if err != nil {
			return err
		}
		p.w.write(" -> " + rt)
	}
	if eff := formatEffectRow(d.Effects); eff != "" {
		p.w.write(" " + eff)
	}
	// FuncDecl.Properties is a mixed-kind slice: RequiresKind/EnsuresKind contract
	// clauses (parsed by parseContractBlocks, signature position) and PropertyKind
	// forall properties (parsed from a `properties [...]` block). Partition by Kind
	// and emit each in the ONLY position the parser accepts:
	//   - requires/ensures contract clauses go in SIGNATURE position (after the
	//     effect row, before tests/properties and the body);
	//   - PropertyKind entries route through the existing `properties [...]` block.
	requires, ensures, props := partitionProperties(d.Properties)
	if err := p.contractClauses("requires", requires); err != nil {
		return err
	}
	if err := p.contractClauses("ensures", ensures); err != nil {
		return err
	}
	if err := p.testsAndProperties(d.Tests, props); err != nil {
		return err
	}
	// Extern functions have no body.
	if d.IsExtern || d.Body == nil {
		return nil
	}
	// Body: `= expr` for a single expression, or a braced newline block.
	return p.funcBody(d.Body)
}

// declParams prints a parameter list, canonicalising the implicit unit parameter
// that `func f()` desugars to back to `()`, so the output re-parses identically.
func (p *printer) declParams(params []*ast.Param) error {
	if isImplicitUnitParam(params) {
		p.w.write("()")
		return nil
	}
	p.w.write("(")
	if err := p.params(params); err != nil {
		return err
	}
	p.w.write(")")
	return nil
}

// isImplicitUnitParam reports whether a parameter list is exactly the single
// synthetic `_: ()` parameter the parser inserts for zero-arg `func f()`.
func isImplicitUnitParam(params []*ast.Param) bool {
	if len(params) != 1 {
		return false
	}
	p := params[0]
	if p.Name != "_" || p.Type == nil {
		return false
	}
	st, ok := p.Type.(*ast.SimpleType)
	return ok && st.Name == "()"
}

// params prints comma-separated `name` or `name: Type` parameters (no delimiters).
func (p *printer) params(params []*ast.Param) error {
	for i, param := range params {
		if i > 0 {
			p.w.write(", ")
		}
		p.w.write(param.Name)
		if param.Type != nil {
			ts, err := p.typeString(param.Type)
			if err != nil {
				return err
			}
			p.w.write(": " + ts)
		}
	}
	return nil
}

// funcBody prints a function body, choosing the form that re-parses to the SAME
// AST node (round-trip requires Block-vs-bare identity, which the parser encodes
// distinctly):
//
//   - An *ast.Block body came from equation form `= ...` (parseEquationBody
//     always wraps, even a single expression, in a Block). Emit `= expr` for one
//     expression, or a braced newline block for zero/many.
//   - A NON-Block body came from block form `{ expr }` with a single expression
//     (parseFunctionBody unwraps a one-expression block to the bare expression).
//     Emit a braced single-statement block so it re-parses to the same bare node.
func (p *printer) funcBody(body ast.Expr) error {
	blk, isBlock := body.(*ast.Block)
	if !isBlock {
		// Block-form single expression: must stay braced to re-parse identically.
		p.w.write(" ")
		return p.bodyBraced(body)
	}
	if len(blk.Exprs) == 1 {
		// Equation-form single expression: `= expr`.
		// M-AILANG-FMT-INLINE-INTERIOR: when that expression is an attached let chain,
		// introduce continuation layout — write ` =`, hardline, and emit the chain one
		// indent level deeper (chain siblings hold a constant indent inside letChainMultiline).
		if let, ok := blk.Exprs[0].(*ast.Let); ok && let.Body != nil && p.hasAnyAttachment(let) {
			p.w.write(" =")
			p.w.hardline()
			var err error
			p.w.indented(func() {
				err = p.letChainMultiline(let)
			})
			return err
		}
		p.w.write(" = ")
		return p.expr(blk.Exprs[0], precLowest)
	}
	// Zero or many expressions → braced newline block.
	p.w.write(" ")
	return p.blockBraced(blk)
}

func (p *printer) annotation(a *ast.Annotation) error {
	p.w.write("@" + a.Name)
	if len(a.Args) == 0 {
		return nil
	}
	// @verify parses the surface form `@verify(depth: N)` but stores only the int
	// literal N in Args (the `depth:` key is dropped by parseVerifyAnnotation). The
	// generic positional emission below would print `@verify(N)`, which the parser
	// then rejects (PAR_VERIFY_ATTR_KEY: "expected 'depth' key"). Re-emit the key so
	// the annotation round-trips.
	if a.Name == "verify" && len(a.Args) == 1 {
		p.w.write("(depth: ")
		if err := p.expr(a.Args[0], precLowest); err != nil {
			return err
		}
		p.w.write(")")
		return nil
	}
	p.w.write("(")
	for i, arg := range a.Args {
		if i > 0 {
			p.w.write(", ")
		}
		if err := p.expr(arg, precLowest); err != nil {
			return err
		}
	}
	p.w.write(")")
	return nil
}

// partitionProperties splits a mixed-kind FuncDecl.Properties slice into its
// requires clauses, ensures clauses, and remaining (PropertyKind) properties,
// each preserving the original relative order. InvariantKind is not produced by
// the current grammar; any non-requires/ensures entry is treated as a property
// so it routes through the `properties [...]` block rather than being silently
// dropped.
func partitionProperties(all []*ast.Property) (requires, ensures, props []*ast.Property) {
	for _, pr := range all {
		switch pr.Kind {
		case ast.RequiresKind:
			requires = append(requires, pr)
		case ast.EnsuresKind:
			ensures = append(ensures, pr)
		default:
			props = append(props, pr)
		}
	}
	return requires, ensures, props
}

// contractClauses prints one merged `requires { p1, p2, … }` / `ensures { … }`
// clause in signature position (column 0, matching the prevailing corpus style),
// or nothing when the slice is empty. Duplicate requires/ensures blocks are a
// parse-time diagnostic, so emitting a single merged block per kind is canonical
// and re-parses to the identical slice order.
func (p *printer) contractClauses(keyword string, preds []*ast.Property) error {
	if len(preds) == 0 {
		return nil
	}
	p.w.hardline()
	p.w.write(keyword + " { ")
	for i, pr := range preds {
		if i > 0 {
			p.w.write(", ")
		}
		if err := p.property(pr); err != nil {
			return err
		}
	}
	p.w.write(" }")
	return nil
}

// testsAndProperties prints inline `tests [...]` and `properties [...]` blocks on
// their own indented lines before the function body. props MUST be pre-filtered to
// PropertyKind entries only (contract clauses are emitted separately in signature
// position by contractClauses), so this cannot re-emit contract predicates.
func (p *printer) testsAndProperties(tests []*ast.TestCase, props []*ast.Property) error {
	if len(tests) > 0 {
		p.w.hardline()
		var err error
		p.w.indented(func() {
			err = p.testsBlock(tests)
		})
		if err != nil {
			return err
		}
	}
	if len(props) > 0 {
		p.w.hardline()
		var err error
		p.w.indented(func() {
			err = p.propertiesBlock(props)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *printer) testsBlock(tests []*ast.TestCase) error {
	p.w.write("tests [")
	p.w.hardline()
	var err error
	p.w.indented(func() {
		for i, tc := range tests {
			if err != nil {
				return
			}
			if err = p.testCase(tc); err != nil {
				return
			}
			if i < len(tests)-1 {
				p.w.write(",")
			}
			p.w.hardline()
		}
	})
	if err != nil {
		return err
	}
	p.w.write("]")
	return nil
}

func (p *printer) testCase(tc *ast.TestCase) error {
	p.w.write("(")
	if len(tc.Inputs) == 1 {
		if err := p.expr(tc.Inputs[0], precLowest); err != nil {
			return err
		}
	} else {
		p.w.write("(")
		for i, in := range tc.Inputs {
			if i > 0 {
				p.w.write(", ")
			}
			if err := p.expr(in, precLowest); err != nil {
				return err
			}
		}
		p.w.write(")")
	}
	p.w.write(", ")
	if err := p.expr(tc.Expected, precLowest); err != nil {
		return err
	}
	p.w.write(")")
	return nil
}

func (p *printer) propertiesBlock(props []*ast.Property) error {
	p.w.write("properties [")
	p.w.hardline()
	var err error
	p.w.indented(func() {
		for i, pr := range props {
			if err != nil {
				return
			}
			if err = p.property(pr); err != nil {
				return
			}
			if i < len(props)-1 {
				p.w.write(",")
			}
			p.w.hardline()
		}
	})
	if err != nil {
		return err
	}
	p.w.write("]")
	return nil
}

// property prints a single property/contract clause.
func (p *printer) property(pr *ast.Property) error {
	if len(pr.Binders) > 0 {
		p.w.write("forall(")
		for i, b := range pr.Binders {
			if i > 0 {
				p.w.write(", ")
			}
			p.w.write(b.Name)
			if b.Type != nil {
				ts, err := p.typeString(b.Type)
				if err != nil {
					return err
				}
				p.w.write(": " + ts)
			}
		}
		p.w.write(") => ")
	}
	return p.expr(pr.Expr, precLowest)
}

func (p *printer) typeDecl(d *ast.TypeDecl) error {
	if d.Exported {
		p.w.write("export ")
	}
	p.w.write("type " + d.Name)
	if len(d.TypeParams) > 0 {
		p.w.write("[" + strings.Join(d.TypeParams, ", ") + "]")
	}
	p.w.write(" = ")
	if err := p.typeDef(d.Definition); err != nil {
		return err
	}
	if len(d.Deriving) > 0 {
		names := make([]string, 0, len(d.Deriving))
		for _, dk := range d.Deriving {
			names = append(names, dk.String())
		}
		p.w.write(" deriving (" + strings.Join(names, ", ") + ")")
	}
	return nil
}

// typeDef prints a type-declaration body: algebraic sum, record, or alias.
func (p *printer) typeDef(def ast.TypeDef) error {
	switch t := def.(type) {
	case *ast.AlgebraicType:
		return p.algebraicType(t)
	case *ast.RecordType:
		s, err := p.recordTypeString(t)
		if err != nil {
			return err
		}
		p.w.write(s)
		return nil
	case *ast.TypeAlias:
		s, err := p.typeString(t.Target)
		if err != nil {
			return err
		}
		p.w.write(s)
		return nil
	default:
		return fmt.Errorf("unsupported type definition node: %T", def)
	}
}

func (p *printer) algebraicType(t *ast.AlgebraicType) error {
	parts := make([]string, len(t.Constructors))
	for i, c := range t.Constructors {
		s, err := p.constructor(c)
		if err != nil {
			return err
		}
		parts[i] = s
	}
	p.w.write(strings.Join(parts, " | "))
	return nil
}

func (p *printer) constructor(c *ast.Constructor) (string, error) {
	if len(c.Fields) == 0 {
		return c.Name, nil
	}
	fields := make([]string, len(c.Fields))
	for i, f := range c.Fields {
		ft, err := p.typeString(f.Type)
		if err != nil {
			return "", err
		}
		if f.Name != "" {
			fields[i] = f.Name + ": " + ft
		} else {
			fields[i] = ft
		}
	}
	return c.Name + "(" + strings.Join(fields, ", ") + ")", nil
}

func (p *printer) typeClass(d *ast.TypeClass) error {
	p.w.write("class " + d.Name + "[" + d.TypeParam + "]")
	if d.Superclass != "" {
		p.w.write(" : " + d.Superclass)
	}
	p.w.write(" {")
	p.w.hardline()
	var err error
	p.w.indented(func() {
		for _, m := range d.Methods {
			if err != nil {
				return
			}
			ts, terr := p.typeString(m.Type)
			if terr != nil {
				err = terr
				return
			}
			p.w.write(m.Name + ": " + ts)
			p.w.hardline()
		}
	})
	if err != nil {
		return err
	}
	p.w.write("}")
	return nil
}

func (p *printer) instance(d *ast.Instance) error {
	ts, err := p.typeString(d.Type)
	if err != nil {
		return err
	}
	p.w.write("instance " + d.ClassName + "[" + ts + "] {")
	p.w.hardline()
	// Instance methods are stored in an unordered map; the parser does not
	// currently produce Instance nodes (parseInstanceDeclaration returns nil), so
	// this path exists for exhaustive coverage. Emit an explicit error rather than
	// non-deterministic map iteration if a populated instance is ever formatted.
	if len(d.Methods) > 0 {
		return fmt.Errorf("cannot format instance %q: method emission order is undefined (map-backed)", d.ClassName)
	}
	p.w.write("}")
	return nil
}

func (p *printer) testDecl(d *ast.TestDecl) error {
	p.w.write("test " + escapeString(d.Name) + " {")
	p.w.hardline()
	var err error
	p.w.indented(func() {
		for _, e := range d.Body {
			if err != nil {
				return
			}
			if err = p.expr(e, precLowest); err != nil {
				return
			}
			p.w.hardline()
		}
	})
	if err != nil {
		return err
	}
	p.w.write("}")
	return nil
}

func (p *printer) propertyDecl(d *ast.PropertyDecl) error {
	p.w.write("property " + escapeString(d.Name) + " {")
	p.w.hardline()
	var err error
	p.w.indented(func() {
		if d.Property != nil {
			err = p.property(d.Property)
			p.w.hardline()
		}
	})
	if err != nil {
		return err
	}
	p.w.write("}")
	return nil
}

// topLevelLet prints a top-level binding. A nil Body is a plain binding; a
// non-nil Body is an explicit let..in expression used as a statement.
func (p *printer) topLevelLet(d *ast.Let) error {
	if d.Body != nil {
		return p.letIn(d)
	}
	p.w.write("let " + d.Name)
	if d.Type != nil {
		ts, err := p.typeString(d.Type)
		if err != nil {
			return err
		}
		p.w.write(": " + ts)
	}
	// M-AILANG-FMT-INLINE-INTERIOR: when the binding VALUE is an attached let chain
	// (the comments live inside the value chain, keyed on the value's root let), write
	// ` =`, hardline, and emit the chain one indent level deeper.
	if val, ok := d.Value.(*ast.Let); ok && val.Body != nil && p.hasAnyAttachment(val) {
		p.w.write(" =")
		p.w.hardline()
		var err error
		p.w.indented(func() {
			err = p.letChainMultiline(val)
		})
		return err
	}
	p.w.write(" = ")
	return p.expr(d.Value, precLowest)
}
