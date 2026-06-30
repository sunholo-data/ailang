package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes AILANG source code
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           rune // current char under examination
	line         int
	column       int
	file         string
	// tokenQueue buffers tokens produced by string interpolation (M1_LEXER_INTERP).
	// When `"...${expr}..."` is encountered, readStringOrInterp emits the whole
	// token sequence (STRING_PART / INTERP_START / expr-tokens / INTERP_END / ...)
	// into this queue; NextToken drains the queue before advancing the main lexer.
	tokenQueue []Token
}

// New creates a new Lexer with normalized input.
// Input is normalized at the lexer boundary:
// - UTF-8 BOM is stripped
// - Unicode NFC normalization is applied
//
// This ensures lexically equivalent source produces identical token streams.
func New(input string, filename string) *Lexer {
	// Normalize input at the boundary
	normalized := Normalize([]byte(input))

	l := &Lexer{
		input:  string(normalized),
		file:   filename,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next character
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		l.position = l.readPosition
	} else {
		var size int
		l.ch, size = utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.position = l.readPosition
		l.readPosition += size
		l.column++
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
	}
}

// peekChar returns the next character without advancing
func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return ch
}

// NextToken returns the next token
func (l *Lexer) NextToken() Token {
	// Drain queued tokens (produced by string interpolation) first.
	if len(l.tokenQueue) > 0 {
		tok := l.tokenQueue[0]
		l.tokenQueue = l.tokenQueue[1:]
		return tok
	}

	var tok Token

	l.skipWhitespace()

	// Save position for token
	line := l.line
	column := l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(EQ, string(ch)+string(l.ch), line, column, l.file)
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = NewToken(FARROW, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(ASSIGN, string(l.ch), line, column, l.file)
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = NewToken(APPEND, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(PLUS, string(l.ch), line, column, l.file)
		}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = NewToken(ARROW, string(ch)+string(l.ch), line, column, l.file)
		} else if l.peekChar() == '-' {
			// Handle single-line comments
			l.skipComment()
			return l.NextToken()
		} else {
			tok = NewToken(MINUS, string(l.ch), line, column, l.file)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(NEQ, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(BANG, string(l.ch), line, column, l.file)
		}
	case '*':
		tok = NewToken(STAR, string(l.ch), line, column, l.file)
	case '/':
		// Check for single-line comment (//)
		if l.peekChar() == '/' {
			l.skipComment()
			return l.NextToken()
		}
		// Check for regex literal
		if l.isRegexStart() {
			return l.readRegex(line, column)
		}
		tok = NewToken(SLASH, string(l.ch), line, column, l.file)
	case '%':
		tok = NewToken(PERCENT, string(l.ch), line, column, l.file)
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(LTE, string(ch)+string(l.ch), line, column, l.file)
		} else if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = NewToken(LARROW, string(ch)+string(l.ch), line, column, l.file)
		} else if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			tok = NewToken(SHL, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(LT, string(l.ch), line, column, l.file)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(GTE, string(ch)+string(l.ch), line, column, l.file)
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = NewToken(SHR, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(GT, string(l.ch), line, column, l.file)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = NewToken(AND, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(AMPERSAND, string(l.ch), line, column, l.file)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = NewToken(OR, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(PIPE, string(l.ch), line, column, l.file)
		}
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			tok = NewToken(DCOLON, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(COLON, string(l.ch), line, column, l.file)
		}
	case '.':
		if l.peekChar() == '.' && l.peekAhead(2) == '.' {
			l.readChar()
			l.readChar()
			tok = NewToken(ELLIPSIS, "...", line, column, l.file)
		} else if l.peekChar() == '.' {
			// ".." range operator (DOTDOT)
			// ELLIPSIS ("...") is handled above, so this is always two dots.
			// Float literals (e.g., 1.5) are handled by readNumber(), so the lexer
			// only reaches here when a DOT follows a non-numeric context.
			l.readChar()
			tok = NewToken(DOTDOT, "..", line, column, l.file)
		} else {
			tok = NewToken(DOT, string(l.ch), line, column, l.file)
		}
	case ',':
		tok = NewToken(COMMA, string(l.ch), line, column, l.file)
	case ';':
		tok = NewToken(SEMICOLON, string(l.ch), line, column, l.file)
	case '(':
		if l.peekChar() == ')' {
			l.readChar() // Move to ')'
			tok = NewToken(UNIT, "()", line, column, l.file)
		} else {
			tok = NewToken(LPAREN, string(l.ch), line, column, l.file)
		}
	case ')':
		tok = NewToken(RPAREN, string(l.ch), line, column, l.file)
	case '{':
		tok = NewToken(LBRACE, string(l.ch), line, column, l.file)
	case '}':
		tok = NewToken(RBRACE, string(l.ch), line, column, l.file)
	case '[':
		tok = NewToken(LBRACKET, string(l.ch), line, column, l.file)
	case ']':
		tok = NewToken(RBRACKET, string(l.ch), line, column, l.file)
	case '?':
		tok = NewToken(QUESTION, string(l.ch), line, column, l.file)
	case '@':
		tok = NewToken(AT, string(l.ch), line, column, l.file)
	case '$':
		tok = NewToken(DOLLAR, string(l.ch), line, column, l.file)
	case '#':
		tok = NewToken(HASH, string(l.ch), line, column, l.file)
	case '^':
		tok = NewToken(CARET, string(l.ch), line, column, l.file)
	case '~':
		tok = NewToken(TILDE, string(l.ch), line, column, l.file)
	case '\\':
		tok = NewToken(BACKSLASH, string(l.ch), line, column, l.file)
	case '"':
		// Check for quasiquote or regular string
		if l.checkQuasiquotePrefix() {
			return l.readQuasiquote(line, column)
		}
		// readStringOrInterp returns either a single STRING token (no `${` found,
		// regression-safe) or drops a STRING_PART/INTERP_START/... sequence into
		// the token queue and returns the first of those tokens.
		return l.readStringOrInterp(line, column)
	case '\'':
		lit, errMsg := l.readCharLiteral()
		if errMsg != "" {
			tok.Type = ILLEGAL
			tok.Literal = errMsg
		} else {
			tok.Type = CHAR
			tok.Literal = lit
		}
		tok.Line = line
		tok.Column = column
		tok.File = l.file
		return tok
	case 0:
		tok = NewToken(EOF, "", line, column, l.file)
	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			tokType := LookupIdentContextual(literal)

			// Check for quasiquote keywords followed by quotes
			if l.checkQuasiquoteKeyword(literal) {
				return l.readQuasiquoteWithKeyword(literal, line, column)
			}

			tok = NewToken(tokType, literal, line, column, l.file)
			return tok
		} else if isDigit(l.ch) {
			literal, isFloat := l.readNumber()
			if isFloat {
				tok = NewToken(FLOAT, literal, line, column, l.file)
			} else {
				tok = NewToken(INT, literal, line, column, l.file)
			}
			return tok
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), line, column, l.file)
		}
	}

	l.readChar()
	return tok
}

// skipWhitespace skips whitespace characters except newlines (which may be significant)
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// skipComment skips single-line comments
func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

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
		return l.readBracedUnicodeEscape()
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
	var v rune
	for i := 0; i < n; i++ {
		if !isHexDigit(l.peekChar()) {
			return 0, false, fmt.Sprintf(`\x requires exactly %d hex digits (e.g. \x1b)`, n)
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

// readIdentifier reads an identifier
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '\'' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() (string, bool) {
	position := l.position
	isFloat := false

	// Hex (0x), binary (0b), octal (0o) integer literals. Read the radix digits
	// and return early — these are always integers (no float/exponent suffix).
	if l.ch == '0' {
		switch l.peekChar() {
		case 'x', 'X':
			l.readChar() // consume '0'
			l.readChar() // consume radix marker
			for isHexDigit(l.ch) {
				l.readChar()
			}
			return l.input[position:l.position], false
		case 'b', 'B':
			l.readChar()
			l.readChar()
			for l.ch == '0' || l.ch == '1' {
				l.readChar()
			}
			return l.input[position:l.position], false
		case 'o', 'O':
			l.readChar()
			l.readChar()
			for l.ch >= '0' && l.ch <= '7' {
				l.readChar()
			}
			return l.input[position:l.position], false
		}
	}

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	if l.ch == 'e' || l.ch == 'E' {
		isFloat = true
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position], isFloat
}

// checkQuasiquotePrefix checks if we're at the start of a quasiquote
func (l *Lexer) checkQuasiquotePrefix() bool {
	// Look behind to see if we have a quasiquote keyword
	// This is handled by checkQuasiquoteKeyword instead
	return false
}

// checkQuasiquoteKeyword checks if the identifier is a quasiquote keyword
func (l *Lexer) checkQuasiquoteKeyword(ident string) bool {
	switch ident {
	case "sql", "html", "shell", "url", "json", "regex":
		return true
	}
	return false
}

// readQuasiquote reads a quasiquote literal
func (l *Lexer) readQuasiquote(line, column int) Token {
	// This would be more complex in reality, handling interpolations
	// For now, just read until the closing quotes
	var content strings.Builder

	// Determine quote type from context
	// For now, default to SqlQuote

	// Skip the opening quotes
	for i := 0; i < 3; i++ {
		l.readChar()
	}

	// Read content until closing quotes
	for {
		if l.ch == '"' && l.peekChar() == '"' && l.peekAhead(2) == '"' {
			l.readChar()
			l.readChar()
			l.readChar()
			break
		}
		if l.ch == 0 {
			break
		}
		content.WriteRune(l.ch)
		l.readChar()
	}

	return NewToken(SQLQuote, content.String(), line, column, l.file)
}

// readQuasiquoteWithKeyword reads a quasiquote that starts with a keyword
func (l *Lexer) readQuasiquoteWithKeyword(keyword string, line, column int) Token {
	l.skipWhitespace()

	switch keyword {
	case "sql":
		// Handle SQL quasiquote
	case "html":
		// Handle HTML quasiquote
	case "shell":
		// Handle shell quasiquote
	case "url":
		// Handle URL quasiquote
	case "json":
		if l.ch == '{' {
			// Read JSON literal
			return l.readJSONQuote(line, column)
		}
	case "regex":
		if l.ch == '/' {
			return l.readRegex(line, column)
		}
	}

	// Expect triple quotes for most quasiquotes
	if l.ch == '"' && l.peekChar() == '"' && l.peekAhead(2) == '"' {
		return l.readQuasiquote(line, column)
	}

	// Otherwise it's just a regular identifier
	return NewToken(LookupIdentContextual(keyword), keyword, line, column, l.file)
}

// readJSONQuote reads a JSON quasiquote
func (l *Lexer) readJSONQuote(line, column int) Token {
	var content strings.Builder
	braceCount := 0

	for l.ch != 0 {
		if l.ch == '{' {
			braceCount++
		} else if l.ch == '}' {
			braceCount--
			if braceCount == 0 {
				l.readChar()
				break
			}
		}
		content.WriteRune(l.ch)
		l.readChar()
	}

	return NewToken(JSONQuote, content.String(), line, column, l.file)
}

// readRegex reads a regex literal
func (l *Lexer) readRegex(line, column int) Token {
	var content strings.Builder
	l.readChar() // skip opening /

	for l.ch != '/' && l.ch != 0 {
		if l.ch == '\\' {
			content.WriteRune(l.ch)
			l.readChar()
			if l.ch != 0 {
				content.WriteRune(l.ch)
				l.readChar()
			}
		} else {
			content.WriteRune(l.ch)
			l.readChar()
		}
	}

	if l.ch == '/' {
		l.readChar() // skip closing /
	}

	// Read any regex flags
	for isLetter(l.ch) {
		content.WriteRune(l.ch)
		l.readChar()
	}

	return NewToken(RegexQuote, content.String(), line, column, l.file)
}

// isRegexStart checks if we're at the start of a regex literal
func (l *Lexer) isRegexStart() bool {
	// Simple heuristic: regex starts with / and isn't division
	// In a real implementation, this would need more context
	return false // For now, regex requires explicit "regex/" prefix
}

// peekAhead looks ahead n characters
func (l *Lexer) peekAhead(n int) rune {
	pos := l.readPosition
	for i := 1; i < n; i++ {
		if pos >= len(l.input) {
			return 0
		}
		_, size := utf8.DecodeRuneInString(l.input[pos:])
		pos += size
	}
	if pos >= len(l.input) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.input[pos:])
	return ch
}

// Helper functions

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}

// isHexDigit reports whether ch is a hexadecimal digit (0-9, a-f, A-F).
func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// Error represents a lexer error
type Error struct {
	Message string
	Line    int
	Column  int
	File    string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
}
