// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
)

// CoreSanityError represents a Core IR invariant violation
type CoreSanityError struct {
	Code       string
	Message    string
	NodeID     uint64
	Suggestion string
}

func (e *CoreSanityError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s (node %d). Suggestion: %s", e.Code, e.Message, e.NodeID, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s (node %d)", e.Code, e.Message, e.NodeID)
}

// AssertNoOperators ensures no operator nodes remain after lowering
func AssertNoOperators(prog *core.Program) error {
	var errors []error
	
	// Walk all declarations
	for _, decl := range prog.Decls {
		walkCore(decl, func(node core.CoreExpr) {
			switch n := node.(type) {
			case *core.Intrinsic:
				errors = append(errors, &CoreSanityError{
					Code:       "ELB_OP002",
					Message:    fmt.Sprintf("Intrinsic '%s' not lowered", GetOpSymbol(n.Op)),
					NodeID:     n.ID(),
					Suggestion: "Enable lowering pass or file internal bug",
				})
			case *core.BinOp:
				errors = append(errors, &CoreSanityError{
					Code:       "ELB_OP002",
					Message:    fmt.Sprintf("BinOp '%s' not lowered", n.Op),
					NodeID:     n.ID(),
					Suggestion: "Enable lowering pass or file internal bug",
				})
			case *core.UnOp:
				errors = append(errors, &CoreSanityError{
					Code:       "ELB_OP002",
					Message:    fmt.Sprintf("UnOp '%s' not lowered", n.Op),
					NodeID:     n.ID(),
					Suggestion: "Enable lowering pass or file internal bug",
				})
			case *core.Match:
				errors = append(errors, &CoreSanityError{
					Code:       "ELB_UNSUPPORTED_NODE",
					Message:    "Match nodes not yet supported",
					NodeID:     n.ID(),
					Suggestion: "Use if-then-else for conditionals",
				})
			}
		})
	}
	
	// Check VarGlobal references
	for _, decl := range prog.Decls {
		walkCore(decl, func(node core.CoreExpr) {
			if vg, ok := node.(*core.VarGlobal); ok {
				if vg.Ref.Module == "$builtin" {
					// Check if builtin is registered
					if !IsBuiltinRegistered(vg.Ref.Name) {
						errors = append(errors, &CoreSanityError{
							Code:       "LNK_BUILTIN404",
							Message:    fmt.Sprintf("Builtin '%s' not registered", vg.Ref.Name),
							NodeID:     vg.ID(),
							Suggestion: "Check builtin name spelling",
						})
					}
				} else {
					// TODO: Check against linker index for other modules
					// For now, we'll assume they're valid if not $builtin
				}
			}
		})
	}
	
	if len(errors) > 0 {
		return errors[0] // Return first error
	}
	return nil
}

// IsBuiltinRegistered checks if a builtin is registered in the evaluator
func IsBuiltinRegistered(name string) bool {
	_, ok := eval.Builtins[name]
	return ok
}

// walkCore recursively walks a Core expression tree
func walkCore(expr core.CoreExpr, visit func(core.CoreExpr)) {
	if expr == nil {
		return
	}
	
	visit(expr)
	
	switch e := expr.(type) {
	case *core.Let:
		walkCore(e.Value, visit)
		walkCore(e.Body, visit)
	
	case *core.LetRec:
		for _, binding := range e.Bindings {
			walkCore(binding.Value, visit)
		}
		walkCore(e.Body, visit)
	
	case *core.Lambda:
		walkCore(e.Body, visit)
	
	case *core.App:
		walkCore(e.Func, visit)
		for _, arg := range e.Args {
			walkCore(arg, visit)
		}
	
	case *core.If:
		walkCore(e.Cond, visit)
		walkCore(e.Then, visit)
		walkCore(e.Else, visit)
	
	case *core.Match:
		walkCore(e.Scrutinee, visit)
		for _, arm := range e.Arms {
			walkCore(arm.Guard, visit)
			walkCore(arm.Body, visit)
		}
	
	case *core.BinOp:
		walkCore(e.Left, visit)
		walkCore(e.Right, visit)
	
	case *core.UnOp:
		walkCore(e.Operand, visit)
	
	case *core.Intrinsic:
		for _, arg := range e.Args {
			walkCore(arg, visit)
		}
	
	case *core.Record:
		for _, field := range e.Fields {
			walkCore(field, visit)
		}
	
	case *core.RecordAccess:
		walkCore(e.Record, visit)
	
	case *core.List:
		for _, elem := range e.Elements {
			walkCore(elem, visit)
		}
	
	case *core.DictAbs:
		walkCore(e.Body, visit)
	
	case *core.DictApp:
		walkCore(e.Dict, visit)
		for _, arg := range e.Args {
			walkCore(arg, visit)
		}
	
	// Atomic nodes - no recursion needed
	case *core.Var, *core.VarGlobal, *core.Lit, *core.DictRef:
		// Nothing to recurse into
	}
}

// WalkCore is the exported version of walkCore
func WalkCore(prog *core.Program, visit func(core.CoreExpr)) {
	for _, decl := range prog.Decls {
		walkCore(decl, visit)
	}
}

// AssertProgramLowered ensures the program has been through the lowering pass
func AssertProgramLowered(prog *core.Program) error {
	if !prog.Flags.Lowered {
		return &CoreSanityError{
			Code:       "ELB_OP002",
			Message:    "Program not marked as lowered",
			NodeID:     0,
			Suggestion: "Enable lowering pass or file internal bug",
		}
	}
	return nil
}