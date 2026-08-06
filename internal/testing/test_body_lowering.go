package testing

import (
	"github.com/sunholo-data/ailang/internal/ast"
)

// This file owns the AST-level lowering of named test bodies — the transform
// that runs between parsing `test "name" { ... }` and printing the body back to
// AILANG source for re-elaboration through the general pipeline
// (EvaluateNamedTestBodyExprs, executor.go).
//
// It lives apart from executor_helpers.go for a mundane reason: that file is
// already 717 lines and `make check-file-sizes` fails at 800.

// FoldBodyExprs collapses a slice of AST expressions — as produced by parsing a
// named test block where semicolons separate statements — into a single nested
// expression.  The parser emits standalone `let x = val` nodes (Body == nil)
// for each binding; this function re-chains them:
//
//	[let x = val, let y = ..., finalExpr]
//	  → let x = val in (let y = ... in finalExpr)
//
// Non-let expressions that are not the last item are dropped (they were
// evaluated for side-effects in the original source, which is meaningless for
// pure bodies, so stripping them is safe).
//
// This is the LEGACY path, retained verbatim: it is what FoldTestBody returns
// for any body that contains no `assert`, so assert-free named tests are
// byte-for-byte unchanged by #590's fix.
func FoldBodyExprs(exprs []ast.Expr) ast.Expr {
	if len(exprs) == 0 {
		return nil
	}
	// Walk in reverse, threading each let-binding around the accumulated tail.
	result := exprs[len(exprs)-1]
	for i := len(exprs) - 2; i >= 0; i-- {
		switch e := exprs[i].(type) {
		case *ast.Let:
			// Clone to avoid mutating the parser's AST in place.
			result = &ast.Let{
				Name:  e.Name,
				Type:  e.Type,
				Value: e.Value,
				Body:  result,
				Pos:   e.Pos,
			}
		case *ast.LetRec:
			result = &ast.LetRec{
				Name:  e.Name,
				Type:  e.Type,
				Value: e.Value,
				Body:  result,
				Pos:   e.Pos,
			}
		default:
			// Side-effect expression before the final value — wrap in a let_ binding.
			// AILANG has no sequencing operator, so we emit "let _ = expr in rest".
			// This is the closest approximation; pure bodies shouldn't have real side
			// effects anyway, so semantics are preserved.
			result = &ast.Let{
				Name:  "_seq",
				Value: exprs[i],
				Body:  result,
				Pos:   exprs[i].Position(),
			}
		}
	}
	return result
}

// CheckInfo describes one check in a named test body that took the assert
// lowering path.  Checks are numbered left-to-right starting at 1, and the
// ordinal doubles as the failure sentinel the lowered expression evaluates to.
type CheckInfo struct {
	Ordinal int     // 1-based position among the body's checks
	Source  string  // AILANG source text of the condition, for the failure message
	Pos     ast.Pos // original position of the check in the user's source
}

// FoldTestBody folds a named test body into a single expression, lowering any
// top-level `assert` statements on the way.
//
// # Why the lowering lives here and not in the printer
//
// `assert` has a lexer keyword, a parser production, an AST node and a
// formatter case — but no prefix parselet in the general expression grammar and
// no elaborator case.  Since EvaluateNamedTestBodyExprs works by printing the
// body back to AILANG source and re-running the general pipeline, an
// `*ast.AssertStmt` that reaches the printer becomes `assert <cond>` in a
// context that cannot parse it: PAR_NO_PREFIX_PARSE, 100% of the time (#590).
//
// Lowering it in the printer would not work either, because short-circuiting is
// a property of the SEQUENCE, not of a single node: printing `assert c` as a
// bare `c` in non-final position gets swallowed by the surrounding
// `let _seq = ... in ...` and the test goes vacuously green.  So the lowering
// belongs where the sequence is built — here.
//
// # The lowering
//
// A body's CHECKS, left to right, 1-based, are: every top-level
// `*ast.AssertStmt`, plus the final expression (counted once if it is itself an
// assert).  The body lowers to an int sentinel:
//
//	0     — every check passed
//	k ≥ 1 — check k was the FIRST to evaluate false; later checks are NOT evaluated
//
// Concretely, `{ assert a; assert b }` becomes
//
//	if a then (if b then 0 else 2) else 1
//
// All sentinels are ≥ 0 deliberately, so the lowering never needs a unary-minus
// literal in `else` position.  The caller (EvaluateNamedTestBodyExprs) decodes
// the resulting *eval.IntValue back into the runner's bool pass/fail contract.
//
// # The legacy path is preserved exactly
//
// A body containing NO `*ast.AssertStmt` returns exactly what FoldBodyExprs
// returns, plus a nil []CheckInfo.  Every named test that passes today is
// necessarily assert-free (assert bodies are 100% broken), so no currently
// passing test changes code path.
//
// Note that non-final NON-assert expressions keep their legacy `let _seq = ...`
// treatment even on the sentinel path.  Making those short-circuit too would
// turn a non-bool non-final expression into a type error in bodies that pass
// today — a semantics change rather than a bug fix, and deliberately out of
// scope here (tracked separately as #604).  Adding it later is a one-case
// extension of the switch below.
func FoldTestBody(exprs []ast.Expr) (ast.Expr, []CheckInfo) {
	if len(exprs) == 0 {
		return nil, nil
	}
	if !containsAssertStmt(exprs) {
		return FoldBodyExprs(exprs), nil
	}

	n := len(exprs)

	// Pass 1: number the checks left to right, so ordinals read in source order.
	// ordinals[i] == 0 means "expression i is not a check".
	ordinals := make([]int, n)
	next := 1
	for i, e := range exprs {
		if isCheck(e, i == n-1) {
			ordinals[i] = next
			next++
		}
	}

	checks := make([]CheckInfo, 0, next-1)
	for i, e := range exprs {
		if ordinals[i] == 0 {
			continue
		}
		checks = append(checks, CheckInfo{
			Ordinal: ordinals[i],
			Source:  PrintAILANGSource(checkCondition(e)),
			Pos:     e.Position(),
		})
	}

	// Pass 2: build the short-circuiting chain right to left.
	var result ast.Expr
	if ordinals[n-1] != 0 {
		// Final check: passing it means the whole body passed → sentinel 0.
		result = &ast.If{
			Condition: checkCondition(exprs[n-1]),
			Then:      intLit(0, exprs[n-1].Position()),
			Else:      intLit(ordinals[n-1], exprs[n-1].Position()),
		}
	} else {
		// Degenerate final element (e.g. a trailing `let` with no body). Leave it
		// exactly as the legacy fold would, so we do not invent a new failure mode.
		result = exprs[n-1]
	}

	for i := n - 2; i >= 0; i-- {
		switch e := exprs[i].(type) {
		case *ast.AssertStmt:
			// Short-circuit: if the condition holds, carry on with the rest of the
			// body; otherwise stop here and report this check's ordinal.
			result = &ast.If{
				Condition: e.Condition,
				Then:      result,
				Else:      intLit(ordinals[i], e.Pos),
				Pos:       e.Pos,
			}
		case *ast.Let:
			result = &ast.Let{
				Name:  e.Name,
				Type:  e.Type,
				Value: e.Value,
				Body:  result,
				Pos:   e.Pos,
			}
		case *ast.LetRec:
			result = &ast.LetRec{
				Name:  e.Name,
				Type:  e.Type,
				Value: e.Value,
				Body:  result,
				Pos:   e.Pos,
			}
		default:
			// Unchanged legacy treatment — see the doc comment above.
			result = &ast.Let{
				Name:  "_seq",
				Value: exprs[i],
				Body:  result,
				Pos:   exprs[i].Position(),
			}
		}
	}

	return result, checks
}

// containsAssertStmt reports whether any top-level body expression is an assert.
//
// A flat scan is exhaustive: the parser dispatches on ASSERT only at statement
// start (parseStatement, internal/parser/parser_test_decl.go), so `assert`
// cannot appear nested inside another expression — `test "x" { let y = assert
// true; y }` is a parse error.
func containsAssertStmt(exprs []ast.Expr) bool {
	for _, e := range exprs {
		if _, ok := e.(*ast.AssertStmt); ok {
			return true
		}
	}
	return false
}

// isCheck reports whether a body expression contributes a check. Every assert is
// a check; so is the final expression, unless it is a binding form (a trailing
// `let` is a degenerate body that has no value to check).
func isCheck(e ast.Expr, isFinal bool) bool {
	if _, ok := e.(*ast.AssertStmt); ok {
		return true
	}
	if !isFinal {
		return false
	}
	switch e.(type) {
	case *ast.Let, *ast.LetRec:
		return false
	default:
		return true
	}
}

// checkCondition returns the bool-valued expression a check tests: an assert's
// condition, or the expression itself for a trailing bare expression.
func checkCondition(e ast.Expr) ast.Expr {
	if a, ok := e.(*ast.AssertStmt); ok {
		return a.Condition
	}
	return e
}

// intLit builds an int literal sentinel. ast.Literal's IntLit value must be an
// int64 — a plain int panics downstream.
func intLit(v int, pos ast.Pos) ast.Expr {
	return &ast.Literal{Kind: ast.IntLit, Value: int64(v), Pos: pos}
}
