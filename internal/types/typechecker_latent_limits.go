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

// recoverLatentDataDepth converts only the bounded traversal failures to a
// compile error: this one and the type-size backstop (TC_TYPE_SIZE_001,
// type_size.go). Compiler invariant panics retain their original behavior.
func recoverLatentDataDepth(err *error) {
	if failure := recover(); failure != nil {
		switch limit := failure.(type) {
		case latentDataDepthExceeded:
			*err = limit
		case typeTooLarge:
			*err = &TypeCheckError{Kind: TypeTooLargeError, Message: limit.Error()}
		case typeWorkExceeded:
			*err = &TypeCheckError{Kind: TypeTooLargeError, Message: limit.Error()}
		default:
			panic(failure)
		}
	}
}

// InferWithConstraints infers a Core expression, refusing incomplete latent rows.
func (tc *CoreTypeChecker) InferWithConstraints(expr core.CoreExpr, env *TypeEnv) (node typedast.TypedNode, updated *TypeEnv, typ Type, constraints []Constraint, err error) {
	defer func() { err = withWrittenNames(err) }() // #2273: never show `a@{f}`
	defer recoverLatentDataDepth(&err)
	tc.declNodes = make([]uint64, 0, 256) // markDefaulted reads this declaration's nodes
	return tc.inferWithConstraints(expr, env)
}

// CheckCoreExpr checks a Core expression with the same latent traversal boundary.
func (tc *CoreTypeChecker) CheckCoreExpr(expr core.CoreExpr, env *TypeEnv) (node typedast.TypedNode, updated *TypeEnv, err error) {
	defer func() { err = withWrittenNames(err) }() // #2273: never show `a@{f}`
	defer recoverLatentDataDepth(&err)
	return tc.checkCoreExpr(expr, env)
}
