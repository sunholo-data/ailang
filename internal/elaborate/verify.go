package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// VerifyANF checks that a Core program has the shape the elaborator produces.
//
// Operands are atomic (Var, VarGlobal, Lit, Lambda, DictRef): the arguments
// of a call, an operator, an intrinsic or a dictionary method, the elements
// of a list, array or tuple, the fields of a record (literal, access or
// update), the condition of an if and the scrutinee of a match. Anything
// else is let-bound, and a let's value may be any expression of this shape:
// a call, a constructor, an if, a match, or another let-chain. Since #2388
// ANF completion floats only its own temporaries, so a let the program
// wrote inside a value (`{ let a = 1; a } + 2`, `let s = (let q = 2 in q)
// in s`) stays nested as the value of the temporary or of the binding; the
// block form produced the same shape before. Lambda bodies, if branches,
// match guards and bodies, and let bodies are checked the same way.
//
// Returns nil if the program has that shape, an error naming the first
// violation otherwise. The REPL runs it after dictionary elaboration (#2407:
// the old description rejected every let, if, match and tuple used as a
// let's value).
func VerifyANF(prog *core.Program) error {
	verifier := &anfVerifier{}
	for i, decl := range prog.Decls {
		if err := verifier.verifyExpr(decl); err != nil {
			return fmt.Errorf("declaration %d: %w", i, err)
		}
	}
	return nil
}

type anfVerifier struct{}

// atomic checks that every operand is atomic, naming the first that is not.
func (v *anfVerifier) atomic(what string, operands ...core.CoreExpr) error {
	for i, o := range operands {
		if !core.IsAtomic(o) {
			return fmt.Errorf("%s operand %d must be atomic, got %T", what, i, o)
		}
	}
	return nil
}

func (v *anfVerifier) atomicFields(what string, fields map[string]core.CoreExpr) error {
	for name, value := range fields {
		if !core.IsAtomic(value) {
			return fmt.Errorf("%s field '%s' must be atomic, got %T", what, name, value)
		}
	}
	return nil
}

// verifyExpr checks one expression in a binding position (a declaration, a
// let's value or body, a branch, an arm, a lambda body).
func (v *anfVerifier) verifyExpr(expr core.CoreExpr) error {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.Var, *core.VarGlobal, *core.Lit, *core.DictRef:
		return nil

	case *core.Lambda:
		if err := v.verifyExpr(e.Body); err != nil {
			return fmt.Errorf("lambda body: %w", err)
		}
		return nil

	case *core.Let:
		if err := v.verifyExpr(e.Value); err != nil {
			return fmt.Errorf("let binding '%s' value: %w", e.Name, err)
		}
		return v.verifyExpr(e.Body)

	case *core.LetRec:
		for _, binding := range e.Bindings {
			if err := v.verifyExpr(binding.Value); err != nil {
				return fmt.Errorf("let rec binding '%s': %w", binding.Name, err)
			}
		}
		return v.verifyExpr(e.Body)

	case *core.App:
		if err := v.atomic("function application", e.Func); err != nil {
			return err
		}
		return v.atomic("function application", e.Args...)

	case *core.BinOp:
		return v.atomic("binary operation '"+e.Op+"'", e.Left, e.Right)

	case *core.UnOp:
		return v.atomic("unary operation '"+e.Op+"'", e.Operand)

	case *core.Intrinsic:
		return v.atomic("intrinsic", e.Args...)

	case *core.DictApp:
		if err := v.atomic("dictionary application", e.Dict); err != nil {
			return err
		}
		return v.atomic("dictionary application", e.Args...)

	case *core.If:
		if err := v.atomic("if condition", e.Cond); err != nil {
			return err
		}
		if err := v.verifyExpr(e.Then); err != nil {
			return fmt.Errorf("if then branch: %w", err)
		}
		if err := v.verifyExpr(e.Else); err != nil {
			return fmt.Errorf("if else branch: %w", err)
		}
		return nil

	case *core.Match:
		if err := v.atomic("match scrutinee", e.Scrutinee); err != nil {
			return err
		}
		for i, arm := range e.Arms {
			if err := v.verifyExpr(arm.Guard); err != nil {
				return fmt.Errorf("match arm %d guard: %w", i, err)
			}
			if err := v.verifyExpr(arm.Body); err != nil {
				return fmt.Errorf("match arm %d: %w", i, err)
			}
		}
		return nil

	case *core.Record:
		return v.atomicFields("record", e.Fields)

	case *core.RecordAccess:
		return v.atomic("record field access", e.Record)

	case *core.RecordUpdate:
		if err := v.atomic("record update", e.Base); err != nil {
			return err
		}
		return v.atomicFields("record update", e.Updates)

	case *core.List:
		return v.atomic("list", e.Elements...)

	case *core.Array:
		return v.atomic("array", e.Elements...)

	case *core.Tuple:
		return v.atomic("tuple", e.Elements...)

	case *core.DictAbs:
		return v.verifyExpr(e.Body)

	case *core.Forall:
		return v.verifyExpr(e.Body)

	default:
		return fmt.Errorf("unknown expression type in ANF verification: %T", expr)
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
	// For now, we just check that we have the same number of declarations
	// A full implementation would recursively compare the AST structure
	return len(p1.Decls) == len(p2.Decls)
}
