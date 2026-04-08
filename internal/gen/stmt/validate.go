package stmt

import "fmt"

// Validate checks IR invariants on a Program.
// Returns nil if valid, or an error describing the violation.
func Validate(prog *Program) error {
	if prog.Package == "" {
		return fmt.Errorf("program has empty package name")
	}
	for _, td := range prog.TypeDecls {
		if td.Name == "" {
			return fmt.Errorf("type declaration has empty name")
		}
		if td.Kind == nil {
			return fmt.Errorf("type %s has nil kind", td.Name)
		}
	}
	for _, fd := range prog.FuncDecls {
		if err := validateFunc(&fd); err != nil {
			return fmt.Errorf("function %s: %w", fd.Name, err)
		}
	}
	return nil
}

func validateFunc(fd *FuncDecl) error {
	if fd.Name == "" {
		return fmt.Errorf("has empty name")
	}
	// EvalOnly stubs produced by lower-pass panic recovery carry only a
	// name + LowerError; they deliberately skip type/body validation
	// because the compiler will tag them EvalOnly and never emit bytecode
	// for them. Calls at runtime go through the bridge to the evaluator.
	if fd.LowerError != "" {
		return nil
	}
	if fd.ReturnType == nil {
		return fmt.Errorf("has nil return type")
	}
	for i, p := range fd.Params {
		if p.Name == "" {
			return fmt.Errorf("param %d has empty name", i)
		}
		if p.Type == nil {
			return fmt.Errorf("param %s has nil type", p.Name)
		}
	}
	for i, s := range fd.Body {
		if err := validateStmt(s); err != nil {
			return fmt.Errorf("body stmt %d: %w", i, err)
		}
	}
	if fd.Return != nil {
		if err := validateExpr(fd.Return); err != nil {
			return fmt.Errorf("return expr: %w", err)
		}
	}
	return nil
}

func validateStmt(s Stmt) error {
	switch s := s.(type) {
	case VarDecl:
		if s.Name == "" {
			return fmt.Errorf("VarDecl has empty name")
		}
		if s.Value == nil {
			return fmt.Errorf("VarDecl %s has nil value", s.Name)
		}
		return validateExpr(s.Value)
	case IfStmt:
		if s.Cond == nil {
			return fmt.Errorf("IfStmt has nil condition")
		}
		if err := validateExpr(s.Cond); err != nil {
			return fmt.Errorf("IfStmt cond: %w", err)
		}
		for i, ts := range s.Then {
			if err := validateStmt(ts); err != nil {
				return fmt.Errorf("IfStmt then[%d]: %w", i, err)
			}
		}
		for i, es := range s.Else {
			if err := validateStmt(es); err != nil {
				return fmt.Errorf("IfStmt else[%d]: %w", i, err)
			}
		}
		return nil
	case SwitchStmt:
		if s.Scrutinee == nil {
			return fmt.Errorf("SwitchStmt has nil scrutinee")
		}
		return validateExpr(s.Scrutinee)
	case ReturnStmt:
		if s.Value == nil {
			return fmt.Errorf("ReturnStmt has nil value")
		}
		return validateExpr(s.Value)
	case AssignStmt:
		if s.Name == "" {
			return fmt.Errorf("AssignStmt has empty name")
		}
		return validateExpr(s.Value)
	case ExprStmt:
		return validateExpr(s.Value)
	default:
		return fmt.Errorf("unknown statement type %T", s)
	}
}

func validateExpr(e Expr) error {
	if e == nil {
		return fmt.Errorf("nil expression")
	}
	// Expressions are structurally valid by construction.
	// Deep validation would check that VarRefs are in scope, etc.
	// For now, just verify non-nil.
	return nil
}
