package lexer

// String, character, and escape-sequence lexing. Split from lexer.go to keep
// each file under the 800-line gate (M-SNAKE-FEEDBACK added C-style escapes).

import (
	"fmt"
	"strings"
	"unicode"
)

// readStringOrInterp reads a string literal with optional `${expr}` interpolation.
//
// Behaviour:
//   - No `${` in the string → emits a single STRING token (byte-for-byte compatible
//     with the old readString path; zero regression).
//   - `${expr}` found → emits the token sequence
//     STRING_PART(prefix) INTERP_START <expr tokens> INTERP_END STRING_PART(next) ...
//     where `<expr tokens>` come from a sub-lexer invoked on the expression source.
//   - `\${literal}` → literal `${` stays inside a STRING; no interpolation triggered.
//
// The first token is returned directly; any additional tokens are appended to
// l.tokenQueue and drained by subsequent NextToken calls.
func (l *Lexer) readStringOrInterp(line, column int) Token {
	l.readChar() // skip opening quote

	// Always buffer the full token sequence first, then decide whether to emit
	// a single STRING token (no interpolation) or the STRING_PART/INTERP_* stream.
	var segment strings.Builder
	var out []Token
	sawInterp := false

	// partLine/partColumn track where the current segment started so that
	// STRING_PART tokens carry the position of their opening quote / closing `}`.
	partLine, partColumn := line, column

	flushPart := func() {
		out = append(out, NewToken(STRING_PART, segment.String(), partLine, partColumn, l.file))
		segment.Reset()
	}

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			r, ok, msg := l.readEscape()
			if !ok {
				// Malformed escape — surface a positioned ILLEGAL token rather than
				// silently dropping the backslash (No Silent Fallbacks). Consume the
				// rest of the literal so the lexer resumes cleanly (no cascade).
				l.skipRestOfString()
				return NewToken(ILLEGAL, msg, line, column, l.file)
			}
			segment.WriteRune(r)
			l.readChar()
			continue
		}
		if l.ch == '$' && l.peekChar() == '{' {
			// Interpolation start: flush the accumulated literal as a STRING_PART.
			sawInterp = true
			flushPart()
			interpLine, interpColumn := l.line, l.column
			l.readChar() // consume '$'
			l.readChar() // consume '{'
			out = append(out, NewToken(INTERP_START, "${", interpLine, interpColumn, l.file))

			// Collect expression source up to the matching `}` (balanced braces).
			exprSrc, exprLine, exprColumn, ok := l.readInterpolationExpr()
			if !ok {
				// Unterminated `${...` — emit an ILLEGAL token and bail so the
				// parser surfaces a clear error rather than getting silent garbage.
				out = append(out, NewToken(ILLEGAL, "unterminated ${...} in string literal", interpLine, interpColumn, l.file))
				break
			}

			// Tokenize the expression source via a sub-lexer, preserving line/column.
			sub := New(exprSrc, l.file)
			sub.line = exprLine
			sub.column = exprColumn - 1 // readChar increments column before use
			for {
				tok := sub.NextToken()
				if tok.Type == EOF {
					break
				}
				out = append(out, tok)
			}
			out = append(out, NewToken(INTERP_END, "}", l.line, l.column, l.file))
			partLine, partColumn = l.line, l.column
			continue
		}
		segment.WriteRune(l.ch)
		l.readChar()
	}

	l.readChar() // skip closing quote (safe even at EOF)

	if !sawInterp {
		// No interpolation → emit a single STRING token, byte-identical to legacy path.
		return NewToken(STRING, segment.String(), line, column, l.file)
	}

	// Trailing STRING_PART captures what follows the last `}` (may be empty).
	flushPart()

	// Return the first token; queue the rest.
	first := out[0]
	l.tokenQueue = append(l.tokenQueue, out[1:]...)
	return first
}

// readInterpolationExpr consumes characters up to (but not including) the matching
// `}` of a `${...}` interpolation. Brace depth is tracked so nested `{}` inside
// the expression do not prematurely close the interpolation.
//
// Returns the expression source text, the starting line/column for sub-lexer
// positioning, and ok=false if EOF is reached before the closing `}`.
// On success, l.ch is the closing `}` at return time; readInterpolationExpr
// advances past it before returning.
func (l *Lexer) readInterpolationExpr() (src string, line, column int, ok bool) {
	line = l.line
	column = l.column
	var out strings.Builder
	depth := 0
	for l.ch != 0 {
		switch l.ch {
		case '{':
			depth++
			out.WriteRune(l.ch)
		case '}':
			if depth == 0 {
				l.readChar() // consume the closing `}`
				return out.String(), line, column, true
			}
			depth--
			out.WriteRune(l.ch)
		case '"':
			// Nested string literal inside an interpolation — copy verbatim
			// (including its own interior) so sub-lexer can re-tokenize it.
			out.WriteRune(l.ch)
			l.readChar()
			for l.ch != 0 && l.ch != '"' {
				if l.ch == '\\' {
					out.WriteRune(l.ch)
					l.readChar()
					if l.ch == 0 {
						break
					}
				}
				out.WriteRune(l.ch)
				l.readChar()
			}
			if l.ch == '"' {
				out.WriteRune(l.ch)
			}
		default:
			out.WriteRune(l.ch)
		}
		l.readChar()
	}
	return "", line, column, false
}

// readCharLiteral reads a character literal. It returns the decoded literal and,
// on a malformed escape, a non-empty error message (the caller emits ILLEGAL).
func (l *Lexer) readCharLiteral() (string, string) {
	var out strings.Builder
	l.readChar() // skip opening quote

	if l.ch == '\\' {
		l.readChar()
		r, ok, msg := l.readEscape()
		if !ok {
			l.skipRestOfChar()
			return "", msg
		}
		out.WriteRune(r)
	} else {
		out.WriteRune(l.ch)
	}

	l.readChar()
	if l.ch == '\'' {
		l.readChar() // skip closing quote
	}

	return out.String(), ""
}

// skipRestOfString consumes input up to and including the closing `"` (or EOF)
// after a malformed escape, so the lexer resumes on the next real token rather
// than re-lexing the remainder of the string literal as garbage.
func (l *Lexer) skipRestOfString() {
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar() // skip the escaped char so `\"` doesn't end the string early
		}
		l.readChar()
	}
	if l.ch == '"' {
		l.readChar()
	}
}

// skipRestOfChar consumes input up to and including the closing `'` (or EOF)
// after a malformed escape in a character literal.
func (l *Lexer) skipRestOfChar() {
	for l.ch != '\'' && l.ch != 0 {
		l.readChar()
	}
	if l.ch == '\'' {
		l.readChar()
	}
}

// readEscape decodes the escape sequence introduced by the current rune: l.ch is
// the character immediately after the backslash. On success it returns the decoded
// rune and leaves l.ch on the LAST character of the escape, so the caller advances
// exactly once (l.readChar()) to move past it. On a malformed escape it returns
// ok=false and a human-readable message; the caller surfaces it as an ILLEGAL
// token — unknown escapes are NEVER silently passed through.
//
// Supported: \n \t \r \\ \" \' \$ \e(ESC) \0(NUL) \xHH(2 hex) \u{H..H}(1-6 hex).
// Hex/unicode escapes are Unicode code points, so strings stay valid UTF-8; raw
// bytes >0x7F are the job of std/bytes.fromInts. Octal (\012) is unsupported.
func (l *Lexer) readEscape() (rune, bool, string) {
	switch l.ch {
	case 'n':
		return '\n', true, ""
	case 't':
		return '\t', true, ""
	case 'r':
		return '\r', true, ""
	case '\\':
		return '\\', true, ""
	case '"':
		return '"', true, ""
	case '\'':
		return '\'', true, ""
	case '$':
		// `\$` → literal `$` (suppresses interpolation of a following `{`).
		return '$', true, ""
	case 'e':
		// Convenience alias for ESC (0x1B) — the dominant terminal use case.
		return '\x1b', true, ""
	case '0':
		// `\0` → NUL. A following octal digit signals an attempted octal escape,
		// which is unsupported — flag it loudly rather than emitting NUL + digits.
		if d := l.peekChar(); d >= '0' && d <= '7' {
			return 0, false, `octal escapes are not supported; use \xHH or \u{...}`
		}
		return 0, true, ""
	case 'x':
		return l.readHexEscape(2)
	case 'u':
		// Two forms: braced `\u{H..H}` (1-6 hex) and the fixed JS/JSON-style
		// `\uXXXX` (exactly 4 hex). v0.27.0's escape rework accepted ONLY the
		// braced form and hard-errored on `\uXXXX` (PAR_ILLEGAL_TOKEN) — a
		// breaking change that silently broke existing code (e.g. `—` for
		// the em-dash in the motoko harness core), so the file no longer parsed.
		// Accept both, matching JS/JSON/Rust. (M-LEXER-U4-COMPAT, 2026-07-02)
		if l.peekChar() == '{' {
			return l.readBracedUnicodeEscape()
		}
		return l.readHexEscape(4)
	default:
		if l.ch >= '1' && l.ch <= '9' {
			return 0, false, `octal escapes are not supported; use \xHH or \u{...}`
		}
		return 0, false, fmt.Sprintf("unknown escape sequence: \\%c", l.ch)
	}
}

// readHexEscape consumes exactly n hex digits following `\x` (l.ch is currently
// 'x') and leaves l.ch on the final hex digit.
func (l *Lexer) readHexEscape(n int) (rune, bool, string) {
	kind := l.ch // the escape letter ('x' or 'u') — keep the message accurate for both
	var v rune
	for i := 0; i < n; i++ {
		if !isHexDigit(l.peekChar()) {
			return 0, false, fmt.Sprintf(`\%c requires exactly %d hex digits`, kind, n)
		}
		l.readChar()
		v = v*16 + hexValue(l.ch)
	}
	return v, true, ""
}

// readBracedUnicodeEscape consumes `\u{H..H}` (l.ch is currently 'u') and leaves
// l.ch on the closing `}`.
func (l *Lexer) readBracedUnicodeEscape() (rune, bool, string) {
	if l.peekChar() != '{' {
		return 0, false, `\u must be followed by {HEX}, e.g. \u{1F40D}`
	}
	l.readChar() // consume '{' (l.ch == '{')
	var v rune
	count := 0
	for isHexDigit(l.peekChar()) {
		l.readChar()
		v = v*16 + hexValue(l.ch)
		count++
		if count > 6 {
			return 0, false, `\u{...} accepts at most 6 hex digits`
		}
	}
	if count == 0 {
		return 0, false, `\u{} requires at least one hex digit`
	}
	if l.peekChar() != '}' {
		return 0, false, `unterminated \u{...} escape (expected '}')`
	}
	l.readChar() // consume '}' (l.ch == '}', last char of the escape)
	if v > unicode.MaxRune || (v >= 0xD800 && v <= 0xDFFF) {
		return 0, false, fmt.Sprintf(`\u{%X} is not a valid Unicode code point`, v)
	}
	return v, true, ""
}

// hexValue returns the numeric value of a hex digit (assumes isHexDigit(ch)).
func hexValue(ch rune) rune {
	switch {
	case ch >= '0' && ch <= '9':
		return ch - '0'
	case ch >= 'a' && ch <= 'f':
		return ch - 'a' + 10
	default: // 'A'..'F'
		return ch - 'A' + 10
	}
}
