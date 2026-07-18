package format

import "github.com/sunholo-data/ailang/internal/lexer"

// comments.go exposes the Phase-1 comment safety gate. The AILANG AST carries no
// comment or trivia fields, so a naive AST reprint would silently delete user
// comments. Phase 1 is comment-non-preserving but NOT comment-destructive: the
// fmt CLI calls HasComments before parsing/formatting and refuses any file that
// contains a real comment (exit 2, file left byte-identical). Lossless comment
// attachment is deferred to Phase 2 (see the design doc).

// HasComments reports whether source contains at least one real AILANG comment.
//
// It is a lossless lexical preflight, not a substring search: `--` and `//`
// introducers inside string literals, character literals, regex literals, and
// triple-quoted quasiquote templates are correctly recognized as literal text
// and do NOT count as comments. It delegates to the lexer's opt-in comment scan,
// which leaves the parser-visible token stream unchanged.
//
// The error return is reserved for future preflight failures; the current
// implementation never returns a non-nil error, but callers must still handle it
// so the fail-closed CLI contract holds if that changes.
func HasComments(source []byte) (bool, error) {
	return lexer.ScanForComment(source), nil
}
