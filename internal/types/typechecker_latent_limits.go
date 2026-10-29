package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// latentDataDepthExceeded unwinds a pathological alias expansion. Unlike an
// incomplete callback list, this can never grant a caller missing effects.
type latentDataDepthExceeded struct{}

func (latentDataDepthExceeded) Error() string {
	return fmt.Sprintf("latent callback type nesting exceeds limit %d", maxDataArrowDepth)
}

// recoverLatentDataDepth converts only this bounded traversal failure to a
// compile error. Compiler invariant panics retain their original behavior.
func recoverLatentDataDepth(err *error) {
	if failure := recover(); failure != nil {
		if limit, ok := failure.(latentDataDepthExceeded); ok {
			*err = limit
		} else {
			panic(failure)
		}
	}
}

// InferWithConstraints infers a Core expression, refusing incomplete latent rows.
func (tc *CoreTypeChecker) InferWithConstraints(expr core.CoreExpr, env *TypeEnv) (node typedast.TypedNode, updated *TypeEnv, typ Type, constraints []Constraint, err error) {
	defer recoverLatentDataDepth(&err)
	return tc.inferWithConstraints(expr, env)
}

// CheckCoreExpr checks a Core expression with the same latent traversal boundary.
func (tc *CoreTypeChecker) CheckCoreExpr(expr core.CoreExpr, env *TypeEnv) (node typedast.TypedNode, updated *TypeEnv, err error) {
	defer recoverLatentDataDepth(&err)
	return tc.checkCoreExpr(expr, env)
}
