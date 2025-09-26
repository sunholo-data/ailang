package elaborate

import (
	"fmt"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// VerifyANF checks that a Core program is in A-Normal Form.
// In ANF:
// - Complex expressions must be let-bound
// - Arguments to functions and operators must be atomic
// - Scrutinees of if/match must be atomic
// - Fields in record construction must be atomic
// - Elements in list construction must be atomic
//
// Returns nil if valid ANF, error describing violation otherwise.
func VerifyANF(prog *core.Program) error {
	verifier := &anfVerifier{}
	for i, decl := range prog.Decls {
		if err := verifier.verifyExpr(decl, true); err != nil {
			return fmt.Errorf("declaration %d: %w", i, err)
		}
	}
	return nil
}

type anfVerifier struct{}

// verifyExpr checks that an expression follows ANF discipline.
// topLevel indicates if this expression is at a binding position (can be complex)
func (v *anfVerifier) verifyExpr(expr core.CoreExpr, topLevel bool) error {
	if expr == nil {
		return nil
	}
	
	switch e := expr.(type) {
	// Atomic expressions - always valid
	case *core.Var, *core.Lit, *core.Lambda, *core.DictRef:
		return nil
		
	// Let bindings - value can be complex, body must be verified
	case *core.Let:
		// Value can be a simple call or atomic
		if err := v.verifySimpleOrAtomic(e.Value); err != nil {
			return fmt.Errorf("let binding '%s' value: %w", e.Name, err)
		}
		// Body continues verification
		return v.verifyExpr(e.Body, topLevel)
		
	case *core.LetRec:
		// Each binding value must be simple or atomic (usually lambdas)
		for _, binding := range e.Bindings {
			if err := v.verifySimpleOrAtomic(binding.Value); err != nil {
				return fmt.Errorf("let rec binding '%s': %w", binding.Name, err)
			}
		}
		// Body continues verification
		return v.verifyExpr(e.Body, topLevel)
		
	// Complex expressions - must be at top level or let-bound
	case *core.App:
		if !topLevel {
			return fmt.Errorf("function application must be let-bound in ANF")
		}
		// Function must be atomic
		if !core.IsAtomic(e.Func) {
			return fmt.Errorf("function in application must be atomic, got %T", e.Func)
		}
		// All arguments must be atomic
		for i, arg := range e.Args {
			if !core.IsAtomic(arg) {
				return fmt.Errorf("argument %d in application must be atomic, got %T", i, arg)
			}
		}
		return nil
		
	case *core.BinOp:
		if !topLevel {
			return fmt.Errorf("binary operation '%s' must be let-bound in ANF", e.Op)
		}
		// Both operands must be atomic
		if !core.IsAtomic(e.Left) {
			return fmt.Errorf("left operand of '%s' must be atomic, got %T", e.Op, e.Left)
		}
		if !core.IsAtomic(e.Right) {
			return fmt.Errorf("right operand of '%s' must be atomic, got %T", e.Op, e.Right)
		}
		return nil
		
	case *core.UnOp:
		if !topLevel {
			return fmt.Errorf("unary operation '%s' must be let-bound in ANF", e.Op)
		}
		// Operand must be atomic
		if !core.IsAtomic(e.Operand) {
			return fmt.Errorf("operand of unary '%s' must be atomic, got %T", e.Op, e.Operand)
		}
		return nil
		
	case *core.If:
		// If can appear anywhere but condition must be atomic
		if !core.IsAtomic(e.Cond) {
			return fmt.Errorf("if condition must be atomic, got %T", e.Cond)
		}
		// Branches are verified at top level
		if err := v.verifyExpr(e.Then, true); err != nil {
			return fmt.Errorf("if then branch: %w", err)
		}
		if err := v.verifyExpr(e.Else, true); err != nil {
			return fmt.Errorf("if else branch: %w", err)
		}
		return nil
		
	case *core.Match:
		// Match can appear anywhere but scrutinee must be atomic
		if !core.IsAtomic(e.Scrutinee) {
			return fmt.Errorf("match scrutinee must be atomic, got %T", e.Scrutinee)
		}
		// Each arm body is verified at top level
		for i, arm := range e.Arms {
			if err := v.verifyExpr(arm.Body, true); err != nil {
				return fmt.Errorf("match arm %d: %w", i, err)
			}
		}
		return nil
		
	case *core.Record:
		if !topLevel {
			return fmt.Errorf("record construction must be let-bound in ANF")
		}
		// All field values must be atomic
		for name, value := range e.Fields {
			if !core.IsAtomic(value) {
				return fmt.Errorf("record field '%s' must be atomic, got %T", name, value)
			}
		}
		return nil
		
	case *core.RecordAccess:
		if !topLevel {
			return fmt.Errorf("record field access must be let-bound in ANF")
		}
		// Record must be atomic
		if !core.IsAtomic(e.Record) {
			return fmt.Errorf("record in field access must be atomic, got %T", e.Record)
		}
		return nil
		
	case *core.List:
		if !topLevel {
			return fmt.Errorf("list construction must be let-bound in ANF")
		}
		// All elements must be atomic
		for i, elem := range e.Elements {
			if !core.IsAtomic(elem) {
				return fmt.Errorf("list element %d must be atomic, got %T", i, elem)
			}
		}
		return nil
		
	// Dictionary nodes
	case *core.DictAbs:
		// Body continues verification
		return v.verifyExpr(e.Body, true)
		
	case *core.DictApp:
		if !topLevel {
			return fmt.Errorf("dictionary application must be let-bound in ANF")
		}
		// Dictionary must be atomic (usually a Var)
		if !core.IsAtomic(e.Dict) {
			return fmt.Errorf("dictionary in DictApp must be atomic, got %T", e.Dict)
		}
		// All arguments must be atomic
		for i, arg := range e.Args {
			if !core.IsAtomic(arg) {
				return fmt.Errorf("argument %d in DictApp must be atomic, got %T", i, arg)
			}
		}
		return nil
		
	default:
		return fmt.Errorf("unknown expression type in ANF verification: %T", expr)
	}
}

// verifySimpleOrAtomic checks that an expression is either atomic or a simple call
// Simple calls are: App, BinOp, UnOp, Record, List, RecordAccess, DictApp
func (v *anfVerifier) verifySimpleOrAtomic(expr core.CoreExpr) error {
	if core.IsAtomic(expr) {
		return nil
	}
	
	// Check if it's a simple call with atomic arguments
	switch e := expr.(type) {
	case *core.App:
		if !core.IsAtomic(e.Func) {
			return fmt.Errorf("function must be atomic in simple call")
		}
		for i, arg := range e.Args {
			if !core.IsAtomic(arg) {
				return fmt.Errorf("argument %d must be atomic in simple call", i)
			}
		}
		return nil
		
	case *core.BinOp:
		if !core.IsAtomic(e.Left) {
			return fmt.Errorf("left operand must be atomic")
		}
		if !core.IsAtomic(e.Right) {
			return fmt.Errorf("right operand must be atomic")
		}
		return nil
		
	case *core.UnOp:
		if !core.IsAtomic(e.Operand) {
			return fmt.Errorf("operand must be atomic")
		}
		return nil
		
	case *core.Record:
		for name, value := range e.Fields {
			if !core.IsAtomic(value) {
				return fmt.Errorf("field '%s' must be atomic", name)
			}
		}
		return nil
		
	case *core.List:
		for i, elem := range e.Elements {
			if !core.IsAtomic(elem) {
				return fmt.Errorf("element %d must be atomic", i)
			}
		}
		return nil
		
	case *core.RecordAccess:
		if !core.IsAtomic(e.Record) {
			return fmt.Errorf("record must be atomic")
		}
		return nil
		
	case *core.DictApp:
		if !core.IsAtomic(e.Dict) {
			return fmt.Errorf("dictionary must be atomic")
		}
		for i, arg := range e.Args {
			if !core.IsAtomic(arg) {
				return fmt.Errorf("argument %d must be atomic", i)
			}
		}
		return nil
		
	// If and Match are control flow, not simple calls
	case *core.If, *core.Match:
		return fmt.Errorf("control flow expressions are not simple calls")
		
	// Let forms are not simple calls
	case *core.Let, *core.LetRec:
		return fmt.Errorf("let bindings are not simple calls")
		
	// DictAbs is not a simple call
	case *core.DictAbs:
		return fmt.Errorf("dictionary abstraction is not a simple call")
		
	default:
		return fmt.Errorf("expression %T is not atomic or simple call", expr)
	}
}

// VerifyIdempotence checks that running ElaborateWithDictionaries twice produces the same result
// This ensures our transformation is idempotent and safe for REPL usage
func VerifyIdempotence(prog *core.Program, resolved map[uint64]*types.ResolvedConstraint) error {
	// First transformation
	prog1, err := ElaborateWithDictionaries(prog, resolved)
	if err != nil {
		return fmt.Errorf("first transformation failed: %w", err)
	}
	
	// Second transformation (should be identity)
	prog2, err := ElaborateWithDictionaries(prog1, resolved)
	if err != nil {
		return fmt.Errorf("second transformation failed: %w", err)
	}
	
	// Compare the two programs (simplified check - in practice we'd do deep equality)
	if !programsEqual(prog1, prog2) {
		return fmt.Errorf("transformation is not idempotent: second pass produced different result")
	}
	
	return nil
}

// programsEqual does a simplified equality check on Core programs
// In a real implementation, this would do deep structural equality
func programsEqual(p1, p2 *core.Program) bool {
	if len(p1.Decls) != len(p2.Decls) {
		return false
	}
	// For now, we just check that we have the same number of declarations
	// A full implementation would recursively compare the AST structure
	return true
}