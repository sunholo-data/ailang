package eval

import (
	"errors"

	"github.com/sunholo-data/ailang/internal/ast"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// matchFailure builds the RT002 error for a match none of whose arms matched
// (M-V1-MATCH-EXHAUSTIVENESS M0). pos is the match's source position; a zero
// position leaves Pos empty for the caller to fill (the decision-tree walker
// does not know the match). The scrutinee is described by shape only, never
// by payload — see ailerrors.MatchFailureError.
func matchFailure(pos ast.Pos, scrutinee Value) error {
	e := &ailerrors.MatchFailureError{}
	if pos.Line > 0 {
		e.Pos = pos.String()
	}
	switch v := scrutinee.(type) {
	case *TaggedValue:
		e.Ctor = v.CtorName
	case *ListValue:
		e.Shape = ailerrors.ListShape(len(v.Elements))
	case nil:
	default:
		e.Shape = v.Type()
	}
	return e
}

// matchPos is the position a match failure reports: the surface span when the
// elaborator recorded one, else the Core span.
func matchPos(orig, core ast.Pos) ast.Pos {
	if orig.Line > 0 {
		return orig
	}
	return core
}

// attachMatchPos fills an RT002 error's empty Pos with pos. Only the
// decision-tree path raises RT002 without a position; every other site
// already set its own, so an inner match's position is never overwritten.
func attachMatchPos(err error, pos ast.Pos) {
	var mf *ailerrors.MatchFailureError
	if pos.Line == 0 || !errors.As(err, &mf) || mf.Pos != "" {
		return
	}
	mf.Pos = pos.String()
}
