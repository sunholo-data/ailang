package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// CheckUniqueLocals reaches a lambda wherever it sits: in every statement
// kind and every expression kind. Each case hides `\r. r` there, inside a
// function whose parameter is r, so the checker must flag it.
func TestCheckUniqueLocals_ReachesEveryPosition(t *testing.T) {
	lam := stmt.Lambda{Params: []stmt.Param{{Name: "r"}}, Return: stmt.VarRef{Name: "r"}}
	one := stmt.LitInt{Value: 1}
	exprs := map[string]stmt.Expr{
		"binop":        stmt.BinOp{Left: one, Right: lam},
		"unop":         stmt.UnOp{Operand: lam},
		"call":         stmt.Call{Func: stmt.VarRef{Name: "f"}, Args: []stmt.Expr{lam}},
		"call-func":    stmt.Call{Func: lam, Args: []stmt.Expr{one}},
		"field":        stmt.FieldAccess{Record: lam},
		"record":       stmt.RecordLit{Fields: []stmt.FieldInit{{Name: "a", Value: lam}}},
		"recordupdate": stmt.RecordUpdate{Base: one, Fields: []stmt.FieldInit{{Name: "a", Value: lam}}},
		"list":         stmt.ListLit{Elems: []stmt.Expr{lam}},
		"array":        stmt.ArrayLit{Elems: []stmt.Expr{lam}},
		"tuple":        stmt.TupleLit{Elems: []stmt.Expr{one, lam}},
		"cons":         stmt.Cons{Head: lam, Tail: one},
		"adt":          stmt.ADTConstructor{TypeName: "T", Tag: "C", Args: []stmt.Expr{lam}},
		"typeassert":   stmt.TypeAssert{Value: lam},
		"ifexpr":       stmt.IfExpr{Cond: one, Then: one, Else: lam},
		"builtin":      stmt.BuiltinCall{Name: "_x", Args: []stmt.Expr{lam}},
		"tageq":        stmt.ADTTagEq{Value: lam},
		"nested":       stmt.Lambda{Params: []stmt.Param{{Name: "q"}}, Body: []stmt.Stmt{stmt.ExprStmt{Value: lam}}},
	}
	for name, e := range exprs {
		stmts := map[string]stmt.Stmt{
			"vardecl": stmt.VarDecl{Name: "v", Value: e},
			"assign":  stmt.AssignStmt{Name: "v", Value: e},
			"return":  stmt.ReturnStmt{Value: e},
			"expr":    stmt.ExprStmt{Value: e},
			"fail":    stmt.MatchFail{Scrutinee: e},
			"if":      stmt.IfStmt{Cond: e},
			"switch": stmt.SwitchStmt{Scrutinee: e,
				Cases:   []stmt.SwitchCase{{Tag: "C", Body: []stmt.Stmt{stmt.ExprStmt{Value: one}}}},
				Default: []stmt.Stmt{stmt.ExprStmt{Value: one}}},
			"case-body": stmt.SwitchStmt{Scrutinee: one, Cases: []stmt.SwitchCase{{Tag: "C", Body: []stmt.Stmt{stmt.ExprStmt{Value: e}}}}},
			"default":   stmt.SwitchStmt{Scrutinee: one, Default: []stmt.Stmt{stmt.ExprStmt{Value: e}}},
			"then":      stmt.IfStmt{Cond: one, Then: []stmt.Stmt{stmt.ExprStmt{Value: e}}},
		}
		for sname, s := range stmts {
			fd := stmt.FuncDecl{Name: name + "/" + sname, Params: []stmt.Param{{Name: "r"}}, Body: []stmt.Stmt{s}}
			if err := CheckUniqueLocals(fd); err == nil {
				t.Errorf("%s in %s: lambda parameter r reusing the parameter r was not flagged", name, sname)
			}
		}
		if err := CheckUniqueLocals(stmt.FuncDecl{Name: name + "/result", Params: []stmt.Param{{Name: "r"}}, Return: e}); err == nil {
			t.Errorf("%s as the function's result was not flagged", name)
		}
	}
	// Switch-case bindings are declarations in the case's scope.
	sw := stmt.FuncDecl{Name: "sw", Params: []stmt.Param{{Name: "r"}}, Body: []stmt.Stmt{
		stmt.SwitchStmt{Scrutinee: one, Cases: []stmt.SwitchCase{{Tag: "C", Bindings: []stmt.Binding{{Name: "r"}}}}},
	}}
	if err := CheckUniqueLocals(sw); err == nil {
		t.Error("a switch binding reusing the parameter r was not flagged")
	}
}
