// Package smt provides SMT-LIB encoding for AILANG contract verification.
// It translates AILANG Core AST functions with contracts into SMT-LIB format
// for verification by Z3 or other SMT solvers.
package smt

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

// RejectionCode identifies why a function cannot be SMT-verified.
type RejectionCode string

const (
	RejectNoContracts  RejectionCode = "NO_CONTRACTS"
	RejectNotPure      RejectionCode = "NOT_PURE"
	RejectRecursive    RejectionCode = "RECURSIVE"
	RejectHigherOrder  RejectionCode = "HIGHER_ORDER"
	RejectDeepPatterns RejectionCode = "DEEP_PATTERNS"
	RejectUnencodable  RejectionCode = "UNENCODABLE_TYPE"
)

// SMTRejectionReason describes why a function cannot be SMT-verified.
type SMTRejectionReason struct {
	Code     RejectionCode
	Message  string
	Location string // Source location if available
	Hint     string // Suggestion to make it encodable
}

func (r SMTRejectionReason) String() string {
	s := fmt.Sprintf("[%s] %s", r.Code, r.Message)
	if r.Hint != "" {
		s += " (" + r.Hint + ")"
	}
	return s
}

// IsSMTEncodable checks whether a function can be SMT-verified.
// Returns true with empty reasons if the function is in the decidable fragment.
// Returns false with rejection reasons otherwise.
func IsSMTEncodable(funcName string, meta *core.DeclMeta, body core.CoreExpr) (bool, []SMTRejectionReason) {
	var reasons []SMTRejectionReason

	// Check 1: Must have contracts
	if !hasContracts(meta) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectNoContracts,
			Message: fmt.Sprintf("Function %q has no contracts to verify", funcName),
			Hint:    "Add requires/ensures clauses",
		})
	}

	// Check 2: Must be pure
	if !isPure(meta) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectNotPure,
			Message: fmt.Sprintf("Function %q has effects", funcName),
			Hint:    "Remove effect annotations or use ! {} for pure",
		})
	}

	// Check 3: Must be non-recursive
	if isRecursive(body, funcName) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectRecursive,
			Message: fmt.Sprintf("Function %q is recursive", funcName),
			Hint:    "SMT verification requires non-recursive functions",
		})
	}

	// Check 4: No higher-order functions
	if hasHigherOrder(body) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectHigherOrder,
			Message: fmt.Sprintf("Function %q uses higher-order functions", funcName),
			Hint:    "Inline function arguments or extract to named functions",
		})
	}

	// Check 5: Shallow patterns only (depth ≤ 1)
	if hasDeepPatterns(body) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectDeepPatterns,
			Message: fmt.Sprintf("Function %q has deeply nested patterns", funcName),
			Hint:    "Flatten nested pattern matches",
		})
	}

	// Check 6: Encodable types only (no string/list/record in logic)
	if hasUnencodableTypes(body) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectUnencodable,
			Message: fmt.Sprintf("Function %q uses types not encodable in SMT (string, list, record)", funcName),
			Hint:    "Use int, float, bool, or enum ADTs",
		})
	}

	return len(reasons) == 0, reasons
}

// hasContracts checks if the function has any contracts.
func hasContracts(meta *core.DeclMeta) bool {
	return meta != nil && len(meta.Contracts) > 0
}

// isPure checks if the function is marked as pure.
// Functions with empty effect sets (! {}) are pure.
func isPure(meta *core.DeclMeta) bool {
	if meta == nil {
		return false
	}
	return meta.IsPure
}

// isRecursive checks if the body references the function name (self-recursion).
func isRecursive(body core.CoreExpr, funcName string) bool {
	if body == nil {
		return false
	}
	return containsRef(body, funcName)
}

// containsRef walks the Core AST looking for references to the given name.
func containsRef(expr core.CoreExpr, name string) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *core.Var:
		return e.Name == name
	case *core.VarGlobal:
		return e.Ref.Name == name
	case *core.Lit:
		return false
	case *core.Lambda:
		// If the lambda re-binds the name, it shadows — not a self-reference
		for _, p := range e.Params {
			if p == name {
				return false
			}
		}
		return containsRef(e.Body, name)
	case *core.App:
		if containsRef(e.Func, name) {
			return true
		}
		for _, arg := range e.Args {
			if containsRef(arg, name) {
				return true
			}
		}
		return false
	case *core.If:
		return containsRef(e.Cond, name) || containsRef(e.Then, name) || containsRef(e.Else, name)
	case *core.Let:
		if containsRef(e.Value, name) {
			return true
		}
		// If the let re-binds the name, it shadows
		if e.Name == name {
			return false
		}
		return containsRef(e.Body, name)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if containsRef(b.Value, name) {
				return true
			}
		}
		return containsRef(e.Body, name)
	case *core.Match:
		if containsRef(e.Scrutinee, name) {
			return true
		}
		for _, arm := range e.Arms {
			if containsRef(arm.Body, name) {
				return true
			}
			if arm.Guard != nil && containsRef(arm.Guard, name) {
				return true
			}
		}
		return false
	case *core.BinOp:
		return containsRef(e.Left, name) || containsRef(e.Right, name)
	case *core.UnOp:
		return containsRef(e.Operand, name)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			if containsRef(arg, name) {
				return true
			}
		}
		return false
	case *core.Record:
		for _, v := range e.Fields {
			if containsRef(v, name) {
				return true
			}
		}
		return false
	case *core.RecordAccess:
		return containsRef(e.Record, name)
	case *core.List:
		for _, elem := range e.Elements {
			if containsRef(elem, name) {
				return true
			}
		}
		return false
	case *core.Tuple:
		for _, elem := range e.Elements {
			if containsRef(elem, name) {
				return true
			}
		}
		return false
	case *core.DictApp:
		return containsRef(e.Dict, name) || containsRefsInSlice(e.Args, name)
	case *core.DictAbs:
		return containsRef(e.Body, name)
	case *core.DictRef:
		return false
	default:
		return false
	}
}

func containsRefsInSlice(exprs []core.CoreExpr, name string) bool {
	for _, e := range exprs {
		if containsRef(e, name) {
			return true
		}
	}
	return false
}

// hasHigherOrder checks if the body contains Lambda expressions as arguments.
// Top-level Lambda (function body) is fine; Lambda inside App args is higher-order.
func hasHigherOrder(body core.CoreExpr) bool {
	if body == nil {
		return false
	}
	return walkForHigherOrder(body, false)
}

func walkForHigherOrder(expr core.CoreExpr, inArgPosition bool) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *core.Lambda:
		if inArgPosition {
			return true
		}
		return walkForHigherOrder(e.Body, false)
	case *core.App:
		if walkForHigherOrder(e.Func, false) {
			return true
		}
		for _, arg := range e.Args {
			if walkForHigherOrder(arg, true) {
				return true
			}
		}
		return false
	case *core.If:
		return walkForHigherOrder(e.Cond, false) || walkForHigherOrder(e.Then, false) || walkForHigherOrder(e.Else, false)
	case *core.Let:
		return walkForHigherOrder(e.Value, false) || walkForHigherOrder(e.Body, false)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if walkForHigherOrder(b.Value, false) {
				return true
			}
		}
		return walkForHigherOrder(e.Body, false)
	case *core.Match:
		if walkForHigherOrder(e.Scrutinee, false) {
			return true
		}
		for _, arm := range e.Arms {
			if walkForHigherOrder(arm.Body, false) {
				return true
			}
		}
		return false
	case *core.BinOp:
		return walkForHigherOrder(e.Left, false) || walkForHigherOrder(e.Right, false)
	case *core.UnOp:
		return walkForHigherOrder(e.Operand, false)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			if walkForHigherOrder(arg, false) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// hasDeepPatterns checks if match arms have nested constructor patterns (depth > 1).
func hasDeepPatterns(body core.CoreExpr) bool {
	if body == nil {
		return false
	}
	return walkForDeepPatterns(body)
}

func walkForDeepPatterns(expr core.CoreExpr) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *core.Match:
		for _, arm := range e.Arms {
			if patternDepth(arm.Pattern) > 1 {
				return true
			}
			if walkForDeepPatterns(arm.Body) {
				return true
			}
		}
		return walkForDeepPatterns(e.Scrutinee)
	case *core.If:
		return walkForDeepPatterns(e.Cond) || walkForDeepPatterns(e.Then) || walkForDeepPatterns(e.Else)
	case *core.Let:
		return walkForDeepPatterns(e.Value) || walkForDeepPatterns(e.Body)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if walkForDeepPatterns(b.Value) {
				return true
			}
		}
		return walkForDeepPatterns(e.Body)
	case *core.App:
		if walkForDeepPatterns(e.Func) {
			return true
		}
		for _, arg := range e.Args {
			if walkForDeepPatterns(arg) {
				return true
			}
		}
		return false
	case *core.Lambda:
		return walkForDeepPatterns(e.Body)
	default:
		return false
	}
}

// patternDepth returns the nesting depth of a pattern.
// VarPattern, LitPattern, WildcardPattern have depth 0.
// ConstructorPattern with no nested constructors has depth 1.
// ConstructorPattern with nested constructors has depth > 1.
func patternDepth(pat core.CorePattern) int {
	if pat == nil {
		return 0
	}
	switch p := pat.(type) {
	case *core.ConstructorPattern:
		maxChildDepth := 0
		for _, arg := range p.Args {
			d := patternDepth(arg)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
		return 1 + maxChildDepth
	case *core.TuplePattern:
		maxChildDepth := 0
		for _, elem := range p.Elements {
			d := patternDepth(elem)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
		return 1 + maxChildDepth
	default:
		return 0
	}
}

// hasUnencodableTypes checks if the body uses types that can't be encoded in SMT-LIB.
// Specifically: String operations, List operations.
// Records are now supported (M-SMT-RECORDS).
// Note: we check structural usage, not type annotations.
func hasUnencodableTypes(body core.CoreExpr) bool {
	if body == nil {
		return false
	}
	return walkForUnencodableTypes(body)
}

func walkForUnencodableTypes(expr core.CoreExpr) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *core.Lit:
		return e.Kind == core.StringLit
	case *core.Record:
		// Records with encodable field values are allowed
		for _, v := range e.Fields {
			if walkForUnencodableTypes(v) {
				return true
			}
		}
		return false
	case *core.RecordAccess:
		return walkForUnencodableTypes(e.Record)
	case *core.RecordUpdate:
		if walkForUnencodableTypes(e.Base) {
			return true
		}
		for _, v := range e.Updates {
			if walkForUnencodableTypes(v) {
				return true
			}
		}
		return false
	case *core.List:
		return true
	case *core.Array:
		return true
	case *core.VarGlobal:
		// Check for string/list builtins
		if e.Ref.Module == "$builtin" {
			return isStringOrListBuiltin(e.Ref.Name)
		}
		return false
	case *core.App:
		if walkForUnencodableTypes(e.Func) {
			return true
		}
		for _, arg := range e.Args {
			if walkForUnencodableTypes(arg) {
				return true
			}
		}
		return false
	case *core.If:
		return walkForUnencodableTypes(e.Cond) || walkForUnencodableTypes(e.Then) || walkForUnencodableTypes(e.Else)
	case *core.Let:
		return walkForUnencodableTypes(e.Value) || walkForUnencodableTypes(e.Body)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if walkForUnencodableTypes(b.Value) {
				return true
			}
		}
		return walkForUnencodableTypes(e.Body)
	case *core.Match:
		if walkForUnencodableTypes(e.Scrutinee) {
			return true
		}
		for _, arm := range e.Arms {
			if walkForUnencodableTypes(arm.Body) {
				return true
			}
		}
		return false
	case *core.BinOp:
		return walkForUnencodableTypes(e.Left) || walkForUnencodableTypes(e.Right)
	case *core.UnOp:
		return walkForUnencodableTypes(e.Operand)
	case *core.Intrinsic:
		if e.Op == core.OpConcat {
			return true
		}
		for _, arg := range e.Args {
			if walkForUnencodableTypes(arg) {
				return true
			}
		}
		return false
	case *core.Lambda:
		return walkForUnencodableTypes(e.Body)
	default:
		return false
	}
}

// isStringOrListBuiltin checks if a builtin name involves strings or lists.
func isStringOrListBuiltin(name string) bool {
	// String builtins: concat_String, eq_String, lt_String, etc.
	// List builtins: concat_List
	for _, suffix := range []string{"_String", "_List"} {
		if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}
