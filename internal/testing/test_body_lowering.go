package testing

import (
	"github.com/sunholo-data/ailang/internal/ast"
)

// The AST-level lowering of named test bodies — the transform that runs
// between parsing `test "name" { ... }` and printing the body back to AILANG
// source for re-elaboration through the general pipeline
// (EvaluateNamedTestBodyExprs, executor.go) — is ast.FoldTestBody. It lives in
// internal/ast because the compiler folds the same body to type-check it in
// `check` (#2232); this file only adds the harness's failure-message text.

// FoldBodyExprs is ast.FoldBodyExprs (the legacy let-chaining fold).
func FoldBodyExprs(exprs []ast.Expr) ast.Expr { return ast.FoldBodyExprs(exprs) }

// CheckInfo describes one check in a named test body that took the sentinel
// lowering path (an assert, or a top-level bare expression).  Checks are numbered left-to-right starting at 1, and the
// ordinal doubles as the failure sentinel the lowered expression evaluates to.
type CheckInfo struct {
	Ordinal int     // 1-based position among the body's checks
	Source  string  // AILANG source text of the condition, for the failure message
	Pos     ast.Pos // original position of the check in the user's source
	// IsAssert records whether the user actually wrote `assert`.  A top-level bare
	// expression is also a check, but quoting it back as `assert <expr>` would
	// show the user source they never wrote.
	IsAssert bool
}

// FoldTestBody is ast.FoldTestBody (the rule and the lowering are documented
// there) with each check's condition printed for the failure message. Checks
// are nil exactly when the body took the legacy path.
func FoldTestBody(exprs []ast.Expr) (ast.Expr, []CheckInfo) {
	folded, folds := ast.FoldTestBody(exprs)
	if folds == nil {
		return folded, nil
	}
	checks := make([]CheckInfo, 0, len(folds))
	for _, c := range folds {
		checks = append(checks, CheckInfo{
			Ordinal:  c.Ordinal,
			Source:   PrintAILANGSource(c.Cond),
			Pos:      c.Pos,
			IsAssert: c.IsAssert,
		})
	}
	return folded, checks
}
