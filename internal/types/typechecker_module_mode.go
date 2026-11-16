package types

import (
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// Module-mode configuration: the module pipelines (internal/pipeline and the
// REPL's module registry) call InferWithConstraints once per module-level
// declaration and tell the checker so through these setters.

// SetModuleFuncNames records the module-level function names of the current
// module. Consumed ONLY by inferVar's undefined-variable diagnostic to detect
// the #327 shape (a local function that resolves as "undefined variable" in a
// record-update field). Interim; retired when the real resolution fix ships.
func (tc *CoreTypeChecker) SetModuleFuncNames(names map[string]bool) {
	tc.moduleFuncNames = names
}

// SetModuleDeclarationMode marks every InferWithConstraints call on this
// checker as checking ONE module-level declaration, whose own binding is then
// exempt from the integer-literal restriction. Set by the module pipelines;
// expression / REPL inference leaves it off, so every let is restricted.
func (tc *CoreTypeChecker) SetModuleDeclarationMode(on bool) {
	tc.moduleDeclMode = on
}

// InferBody infers a body that is NOT a module-level declaration although
// the module pipeline checks it with this checker — a test body, or (through
// InferPredicate) a contract clause. A `let` at its root is an inner let:
// generalised under the same rules as a let in a function body (the
// integer-literal restriction, an open constructor choice, an undecided
// field access), never exempted as the declaration's own binding. Before,
// `requires { let g = \q. match q { Ok(_) => true, _ => false } in g(r) }`
// generalised g with its by-name choice open and the use g(r) never decided
// it (#2362 review).
func (tc *CoreTypeChecker) InferBody(expr core.CoreExpr, env *TypeEnv) (typedast.TypedNode, *TypeEnv, Type, []Constraint, error) {
	prev := tc.notDeclRoot
	tc.notDeclRoot = expr
	defer func() { tc.notDeclRoot = prev }()
	return tc.InferWithConstraints(expr, env)
}
