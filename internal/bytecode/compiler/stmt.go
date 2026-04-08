package compiler

import (
	"fmt"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
)

// compileStmt lowers a single statement.
//
// Statement IR statements that produce no value (VarDecl, AssignStmt, ExprStmt)
// emit instructions that side-effect register state. ReturnStmt emits a
// terminating OpReturn. Control-flow statements (IfStmt, SwitchStmt) arrive in
// later milestones.
func (fc *funcCompiler) compileStmt(s stmt.Stmt) error {
	switch s := s.(type) {
	case stmt.VarDecl:
		return fc.compileVarDecl(s)
	case stmt.AssignStmt:
		return fc.compileAssign(s)
	case stmt.ReturnStmt:
		return fc.compileReturn(s)
	case stmt.ExprStmt:
		return fc.compileExprStmt(s)
	case stmt.IfStmt:
		return fc.compileIfStmt(s)
	case stmt.SwitchStmt:
		return fc.compileSwitch(s)
	}
	return fmt.Errorf("compiler: unsupported statement %T", s)
}

func (fc *funcCompiler) compileVarDecl(s stmt.VarDecl) error {
	// Materialize the initializer into a fresh register, then bind it as a
	// pinned local. Using compileExprToFresh ensures we don't alias a parameter
	// register — if the user writes `let x = param`, x must be a separate
	// (and now mutable-via-shadowing) slot.
	src, err := fc.compileExprToFresh(s.Value)
	if err != nil {
		return err
	}
	// Promote the temp to a pinned local. The free list never sees it again.
	fc.locals.bind(s.Name, src)
	return nil
}

func (fc *funcCompiler) compileAssign(s stmt.AssignStmt) error {
	dst, ok := fc.locals.lookup(s.Name)
	if !ok {
		return fmt.Errorf("compiler: assignment to unbound variable %q", s.Name)
	}
	src, err := fc.compileExpr(s.Value)
	if err != nil {
		return err
	}
	if src != dst {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, dst, src, 0))
	}
	if !fc.isPinned(src) {
		fc.regs.freeTemp(src)
	}
	return nil
}

func (fc *funcCompiler) compileReturn(s stmt.ReturnStmt) error {
	return fc.compileReturnExpr(s.Value)
}

func (fc *funcCompiler) compileExprStmt(s stmt.ExprStmt) error {
	r, err := fc.compileExpr(s.Value)
	if err != nil {
		return err
	}
	if !fc.isPinned(r) {
		fc.regs.freeTemp(r)
	}
	return nil
}
