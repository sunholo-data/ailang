package format

import (
	"fmt"
	"strconv"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/strutil"
)

// literal.go renders literals to canonical source. String payloads go through
// strutil.Quote, the one escaper shared with `show` (M-V1-SHOW-TOTAL), so
// escaping is uniform and precedence-safe. Character literals are parsed as
// single-char ast.StringLit (parser_literals.go: "Treat chars as single-char
// strings for now"), so they share the same escaping path; there is no
// separate ast.CharLit kind to print. The AST stores the DECODED payload (the
// lexer resolves \n, \u{...}, etc.), so the formatter re-escapes canonically
// rather than echoing source bytes. Debug String() methods are never used as a
// fallback (design non-negotiable).

// escapeString renders a decoded string payload as a canonical double-quoted
// AILANG string literal: the minimal, round-trip-safe escape set the lexer's
// readEscape accepts. See strutil.Quote for the table.
func escapeString(s string) string { return strutil.Quote(s) }

// literalString renders any ast.Literal to canonical source, or errors on an
// unsupported/malformed payload. Integer and float literals reuse the lexer's
// decoded numeric value; there is no source-text echo.
func literalString(l *ast.Literal) (string, error) {
	switch l.Kind {
	case ast.IntLit:
		switch v := l.Value.(type) {
		case int64:
			if v < 0 {
				// A negative literal can only come from a full-width
				// base-prefixed one (0x9e3779b97f4a7c15, #1481); `-` is an
				// operator, not part of a literal. Print the 64-bit pattern:
				// decimal "-7046029254386353131" re-parses as unary minus (a
				// different AST), and "-9223372036854775808" not at all.
				return fmt.Sprintf("0x%x", uint64(v)), nil
			}
			return strconv.FormatInt(v, 10), nil
		case int:
			return strconv.Itoa(v), nil
		default:
			return "", fmt.Errorf("int literal has unexpected value type %T", l.Value)
		}
	case ast.FloatLit:
		switch v := l.Value.(type) {
		case float64:
			return ast.FormatFloat(v), nil
		default:
			return "", fmt.Errorf("float literal has unexpected value type %T", l.Value)
		}
	case ast.StringLit:
		s, ok := l.Value.(string)
		if !ok {
			return "", fmt.Errorf("string literal has unexpected value type %T", l.Value)
		}
		return escapeString(s), nil
	case ast.BoolLit:
		switch v := l.Value.(type) {
		case bool:
			if v {
				return "true", nil
			}
			return "false", nil
		default:
			return "", fmt.Errorf("bool literal has unexpected value type %T", l.Value)
		}
	case ast.UnitLit:
		return "()", nil
	default:
		return "", fmt.Errorf("unsupported literal kind %d", int(l.Kind))
	}
}
