package compiler

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// compileStmt lowers a single statement.
//
// Statement IR statements that produce no value (VarDecl, AssignStmt, ExprStmt)
// emit instructions that side-effect register state. ReturnStmt emits a
// terminating OpReturn. Control-flow statements (IfStmt, SwitchStmt) arrive in
// later milestones.
func (fc *funcCompiler) compileStmt(s stmt.Stmt) error {
	// Snapshot the statement's source line so emit() can stamp it onto every
	// instruction this statement produces. We restore the previous line on
	// exit so nested constructs (e.g. an IfStmt body) inherit the enclosing
	// line if their own statements happen to have no Line set.
	prevLine := fc.currentLine
	if l := stmtLine(s); l > 0 {
		fc.currentLine = l
	}
	defer func() { fc.currentLine = prevLine }()

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

// stmtLine returns the source line attached to a statement, or 0 if none.
func stmtLine(s stmt.Stmt) int {
	switch s := s.(type) {
	case stmt.VarDecl:
		return s.Line
	case stmt.AssignStmt:
		return s.Line
	case stmt.ReturnStmt:
		return s.Line
	case stmt.ExprStmt:
		return s.Line
	case stmt.IfStmt:
		return s.Line
	case stmt.SwitchStmt:
		return s.Line
	}
	return 0
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
	// Use bindScoped so the register is recycled when the enclosing scope pops.
	// Root-scope bindings (function-level lets) are never popped, so they stay
	// pinned for the function's lifetime — matching the previous behavior.
	fc.locals.bindScoped(s.Name, src)
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
