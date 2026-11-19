package lower

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// CheckUniqueLocals verifies, on one lowered function, the property the
// bytecode compiler's flat name space needs and uniqueLocals establishes
// (#2375): no source-named local is declared while a local of the same name
// is still in scope. In the statement IR a declaration's scope runs to the
// end of its statement list, so a hoisted block let that reused an outer
// name broke this; declarations in sibling branches (an if's arms, a switch's
// cases) do not overlap and may share a name, which match lowering relies on
// when it copies an arm into two branches. Lambda bodies are nested scopes:
// a lambda parameter or local must not reuse an enclosing name either, or
// capture analysis could not tell them apart.
//
// Compiler-made names (`$...`, the `ctorArgN__` parameters of a constructor
// used as a function) and `_` are exempt. Tests run it over real programs;
// the lowering itself does not call it.
func CheckUniqueLocals(fd stmt.FuncDecl) error {
	c := &uniqueChecker{fn: fd.Name}
	c.push()
	for _, p := range fd.Params {
		c.declare(p.Name)
	}
	c.stmts(fd.Body)
	c.expr(fd.Return)
	if len(c.errs) > 0 {
		return fmt.Errorf("%s", strings.Join(c.errs, "; "))
	}
	return nil
}

type uniqueChecker struct {
	fn     string
	frames []map[string]bool
	errs   []string
}

func (c *uniqueChecker) push() { c.frames = append(c.frames, map[string]bool{}) }
func (c *uniqueChecker) pop()  { c.frames = c.frames[:len(c.frames)-1] }

func (c *uniqueChecker) declare(name string) {
	if name == "_" || strings.HasPrefix(name, "$") || strings.HasPrefix(name, "ctorArg") {
		return
	}
	for _, f := range c.frames {
		if f[name] {
			c.errs = append(c.errs, fmt.Sprintf("function %s declares %q while an outer %q is in scope", c.fn, name, name))
			break
		}
	}
	c.frames[len(c.frames)-1][name] = true
}

func (c *uniqueChecker) branch(ss []stmt.Stmt, bind []stmt.Binding) {
	c.push()
	for _, b := range bind {
		c.declare(b.Name)
	}
	c.stmts(ss)
	c.pop()
}

func (c *uniqueChecker) stmts(ss []stmt.Stmt) {
	for _, s := range ss {
		switch s := s.(type) {
		case stmt.VarDecl:
			c.expr(s.Value)
			c.declare(s.Name)
		case stmt.AssignStmt:
			c.expr(s.Value)
		case stmt.ReturnStmt:
			c.expr(s.Value)
		case stmt.ExprStmt:
			c.expr(s.Value)
		case stmt.MatchFail:
			c.expr(s.Scrutinee)
		case stmt.IfStmt:
			c.expr(s.Cond)
			c.branch(s.Then, nil)
			c.branch(s.Else, nil)
		case stmt.SwitchStmt:
			c.expr(s.Scrutinee)
			for _, sc := range s.Cases {
				c.branch(sc.Body, sc.Bindings)
			}
			c.branch(s.Default, nil)
		}
	}
}

func (c *uniqueChecker) exprs(es []stmt.Expr) {
	for _, e := range es {
		c.expr(e)
	}
}

// expr finds the lambdas in e and checks each as a nested scope.
func (c *uniqueChecker) expr(e stmt.Expr) {
	switch e := e.(type) {
	case nil:
	case stmt.Lambda:
		c.push()
		for _, p := range e.Params {
			c.declare(p.Name)
		}
		c.stmts(e.Body)
		c.expr(e.Return)
		c.pop()
	case stmt.BinOp:
		c.exprs([]stmt.Expr{e.Left, e.Right})
	case stmt.UnOp:
		c.expr(e.Operand)
	case stmt.Call:
		c.expr(e.Func)
		c.exprs(e.Args)
	case stmt.FieldAccess:
		c.expr(e.Record)
	case stmt.RecordLit:
		for _, f := range e.Fields {
			c.expr(f.Value)
		}
	case stmt.RecordUpdate:
		c.expr(e.Base)
		for _, f := range e.Fields {
			c.expr(f.Value)
		}
	case stmt.ListLit:
		c.exprs(e.Elems)
	case stmt.ArrayLit:
		c.exprs(e.Elems)
	case stmt.TupleLit:
		c.exprs(e.Elems)
	case stmt.Cons:
		c.exprs([]stmt.Expr{e.Head, e.Tail})
	case stmt.ADTConstructor:
		c.exprs(e.Args)
	case stmt.TypeAssert:
		c.expr(e.Value)
	case stmt.IfExpr:
		c.exprs([]stmt.Expr{e.Cond, e.Then, e.Else})
	case stmt.BuiltinCall:
		c.exprs(e.Args)
	case stmt.ADTTagEq:
		c.expr(e.Value)
	}
}
