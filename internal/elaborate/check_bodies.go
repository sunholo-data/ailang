package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
)

// CodeTestBody (TC_TEST_001) is the error for a named test block, a property
// or an inline test row that does not type-check (#2232).
//
// Those bodies are never part of the program: the test harness compiles them
// separately. Before #2232 nothing else did, so `check` passed a module whose
// test could not compile. The elaborator now also elaborates each one, in the
// form the harness compiles it, and the module pipeline type-checks it (and,
// for test and property bodies, checks it pure, as the harness does). The
// Core built here is checked only: it is never lowered, run or cached, and
// only the root module's bodies are checked (a dependency's tests are not
// part of its importers).
const CodeTestBody = "TC_TEST_001"

// CheckBody is one test-only body for the module pipeline to type-check.
type CheckBody struct {
	Kind string // "test", "property" or "test row"
	Name string // the test or property title, or "<func> row <n>"
	Pos  ast.Pos
	// Expr is a Lambda for a test (no params) or a property (one per binder),
	// the shape of the harness's `pure func __namedtest_<k>(...)` entry, so
	// Pure is true. For a row it is `if true then f(inputs) else expected`,
	// which unifies the call with the expected value; Pure is false (a row
	// calls the function under test, whatever its effects).
	Expr core.CoreExpr
	Pure bool
}

// fixFor is the TC_TEST_001 suggestion for a body of this kind. For a test or
// property it is the harness's own verdict; a row is stricter than the
// harness, which compares row values without type-checking them.
func fixFor(kind string) string {
	if kind == "test row" {
		return "make the row's inputs fit the function's parameters and its expected value its result type; " +
			"`ailang test` compares row values without type-checking them, so this check is stricter than the harness"
	}
	return "fix the " + kind + "; `ailang test` would fail to compile it"
}

// CheckBodies returns the test-only bodies of the last ElaborateFile.
func (e *Elaborator) CheckBodies() []CheckBody { return e.checkBodies }

// TestBodyError is the TC_TEST_001 report for a body; cause is why it does
// not compile.
func TestBodyError(b CheckBody, cause error) error {
	return errors.WrapReport(&errors.Report{
		Schema:  "ailang.error/v1",
		Code:    CodeTestBody,
		Phase:   "typecheck",
		Message: fmt.Sprintf("%s %q at %s does not type-check: %v", b.Kind, b.Name, b.Pos, cause),
		Span:    &ast.Span{Start: b.Pos, End: b.Pos},
		Data:    map[string]any{"kind": b.Kind, "name": b.Name},
		Fix:     &errors.Fix{Suggestion: fixFor(b.Kind)},
	})
}

// elaborateCheckBodies elaborates every named test block, property block,
// inline forall property and inline test row of file into e.checkBodies. Run
// after the module's declarations, so module scope and imports resolve as in
// a function body.
func (e *Elaborator) elaborateCheckBodies(file *ast.File) error {
	e.checkBodies = nil
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.TestDecl:
			if err := e.addTestBody(d); err != nil {
				return err
			}
		case *ast.PropertyDecl:
			if err := e.addPropertyBody(d.Name, d.Property); err != nil {
				return err
			}
		}
	}
	for _, fn := range file.Funcs {
		for i, prop := range fn.Properties {
			if prop.Kind != ast.PropertyKind {
				continue // requires/ensures are contracts (DeclMeta.Contracts)
			}
			if err := e.addPropertyBody(fmt.Sprintf("%s property %d", fn.Name, i+1), prop); err != nil {
				return err
			}
		}
		for i, row := range fn.Tests {
			if err := e.addRowBody(fn, i, row); err != nil {
				return err
			}
		}
	}
	return nil
}

// addTestBody folds a named test body exactly as the harness does
// (ast.FoldTestBody) and elaborates it as a no-argument entry. A body that
// took the legacy path (no assert, no bare non-final check) passes only when
// its final expression is true, so the entry returns bool: the runner fails
// any other value ("expected bool result"), and check now says so first.
func (e *Elaborator) addTestBody(d *ast.TestDecl) error {
	folded, checks := ast.FoldTestBody(d.Body)
	if folded == nil {
		return nil // an empty body has nothing to type
	}
	b := CheckBody{Kind: "test", Name: d.Name, Pos: d.Pos, Pure: true}
	body, err := e.elaborateExpr(folded)
	if err != nil {
		return TestBodyError(b, err)
	}
	lam := &core.Lambda{CoreNode: e.makeNode(d.Pos), Body: body}
	if checks == nil {
		e.returnTypeAnnots[lam.ID()] = types.TBool
	}
	b.Expr = lam
	e.checkBodies = append(e.checkBodies, b)
	return nil
}

// addPropertyBody elaborates a forall property as the harness's entry: one
// parameter per binder, annotated with the binder's type, returning bool (the
// runner fails a property whose predicate yields anything else).
func (e *Elaborator) addPropertyBody(name string, p *ast.Property) error {
	if p == nil || p.Expr == nil {
		return nil
	}
	b := CheckBody{Kind: "property", Name: name, Pos: p.Pos, Pure: true}
	params := make([]string, len(p.Binders))
	annots := make([]types.Type, len(p.Binders))
	annotated := false
	for i, bd := range p.Binders {
		params[i] = bd.Name
		if bd.Type != nil {
			annots[i] = e.astTypeToInternalType(bd.Type)
			annotated = true
		}
	}
	e.pushScope(params...)
	body, err := e.elaborateExpr(p.Expr)
	e.popScope()
	if err != nil {
		return TestBodyError(b, err)
	}
	lam := &core.Lambda{CoreNode: e.makeNode(p.Pos), Params: params, Body: body}
	if annotated {
		e.paramTypeAnnots[lam.ID()] = annots
	}
	e.returnTypeAnnots[lam.ID()] = types.TBool
	b.Expr = lam
	e.checkBodies = append(e.checkBodies, b)
	return nil
}

// addRowBody elaborates an inline row `(inputs..., expected)` of fn as
// `if true then fn(inputs...) else expected`: the call must type, and its
// result must have the expected value's type.
func (e *Elaborator) addRowBody(fn *ast.FuncDecl, i int, row *ast.TestCase) error {
	if row == nil || row.Expected == nil {
		return nil
	}
	b := CheckBody{Kind: "test row", Name: fmt.Sprintf("%s row %d", fn.Name, i+1), Pos: row.Pos}
	// The call the harness makes for this row (ast.RowInput + RowCallArgs):
	// `((a, b), r)` and `(((a, b)), r)` both call fn(a, b).
	args := ast.RowCallArgs(ast.RowInput(row))
	call := &ast.FuncCall{Func: &ast.Identifier{Name: fn.Name, Pos: row.Pos}, Args: args, Pos: row.Pos}
	expr := &ast.If{
		Condition: &ast.Literal{Kind: ast.BoolLit, Value: true, Pos: row.Pos},
		Then:      call,
		Else:      row.Expected,
		Pos:       row.Pos,
	}
	body, err := e.elaborateExpr(expr)
	if err != nil {
		return TestBodyError(b, err)
	}
	b.Expr = body
	e.checkBodies = append(e.checkBodies, b)
	return nil
}
