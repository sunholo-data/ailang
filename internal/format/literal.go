package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// literal.go holds the single canonical escaping routine for string payloads.
// Every string emitter routes through escapeString so escaping is uniform and
// precedence-safe. Character literals are parsed as single-char ast.StringLit
// (parser_literals.go: "Treat chars as single-char strings for now"), so they
// share the same escaping path; there is no separate ast.CharLit kind to print.
// The AST stores the DECODED payload (the lexer resolves \n, \u{...}, etc.), so
// the formatter re-escapes canonically rather than echoing source bytes. Debug
// String() methods are never used as a fallback (design non-negotiable).

// escapeString renders a decoded string payload as a canonical double-quoted
// AILANG string literal. It emits the minimal, round-trip-safe escape set that
// the lexer's readEscape accepts (\n \t \r \\ \" plus \u{...} for other
// control/non-graphic runes). A literal "${" is escaped as "\${" so the
// re-lexer does not treat it as an interpolation start.
func escapeString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '$':
			// Only "${" would be misread as interpolation; escape the '$' there.
			if i+1 < len(runes) && runes[i+1] == '{' {
				b.WriteString(`\$`)
			} else {
				b.WriteRune('$')
			}
		default:
			if isPrintableRune(r) {
				b.WriteRune(r)
			} else {
				fmt.Fprintf(&b, `\u{%X}`, r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// isPrintableRune reports whether a rune can be emitted verbatim inside a
// literal. Space is printable; other ASCII control characters and non-graphic
// Unicode runes are escaped.
func isPrintableRune(r rune) bool {
	if r == ' ' {
		return true
	}
	if r < 0x20 || r == 0x7f {
		return false
	}
	return strconv.IsPrint(r)
}

// literalString renders any ast.Literal to canonical source, or errors on an
// unsupported/malformed payload. Integer and float literals reuse the lexer's
// decoded numeric value; there is no source-text echo.
func literalString(l *ast.Literal) (string, error) {
	switch l.Kind {
	case ast.IntLit:
		switch v := l.Value.(type) {
		case int64:
			return strconv.FormatInt(v, 10), nil
		case int:
			return strconv.Itoa(v), nil
		default:
			return "", fmt.Errorf("int literal has unexpected value type %T", l.Value)
		}
	case ast.FloatLit:
		switch v := l.Value.(type) {
		case float64:
			return formatFloat(v), nil
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

// formatFloat renders a float64 so that re-parsing yields the same value and so
// that whole-number floats retain a decimal point (canonical AILANG float form).
func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'g', -1, 64)
	// Ensure a float always reads back as a float, never as an int literal.
	if !strings.ContainsAny(s, ".eEnN") { // n/N guards Inf/NaN spellings defensively
		s += ".0"
	}
	return s
}
