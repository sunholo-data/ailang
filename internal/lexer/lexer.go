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
}

// New creates a new Lexer
func New(input string, filename string) *Lexer {
	l := &Lexer{
		input:  input,
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
		} else {
			tok = NewToken(LT, string(l.ch), line, column, l.file)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(GTE, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(GT, string(l.ch), line, column, l.file)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = NewToken(AND, string(ch)+string(l.ch), line, column, l.file)
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), line, column, l.file)
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
	case '\\':
		tok = NewToken(BACKSLASH, string(l.ch), line, column, l.file)
	case '"':
		// Check for quasiquote or regular string
		if l.checkQuasiquotePrefix() {
			return l.readQuasiquote(line, column)
		}
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = line
		tok.Column = column
		tok.File = l.file
		return tok
	case '\'':
		tok.Type = CHAR
		tok.Literal = l.readCharLiteral()
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

// readString reads a string literal
func (l *Lexer) readString() string {
	var out strings.Builder
	l.readChar() // skip opening quote

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				out.WriteRune('\n')
			case 't':
				out.WriteRune('\t')
			case 'r':
				out.WriteRune('\r')
			case '\\':
				out.WriteRune('\\')
			case '"':
				out.WriteRune('"')
			default:
				out.WriteRune(l.ch)
			}
		} else {
			out.WriteRune(l.ch)
		}
		l.readChar()
	}

	l.readChar() // skip closing quote
	return out.String()
}

// readChar reads a character literal
func (l *Lexer) readCharLiteral() string {
	var out strings.Builder
	l.readChar() // skip opening quote

	if l.ch == '\\' {
		l.readChar()
		switch l.ch {
		case 'n':
			out.WriteRune('\n')
		case 't':
			out.WriteRune('\t')
		case 'r':
			out.WriteRune('\r')
		case '\\':
			out.WriteRune('\\')
		case '\'':
			out.WriteRune('\'')
		default:
			out.WriteRune(l.ch)
		}
	} else {
		out.WriteRune(l.ch)
	}

	l.readChar()
	if l.ch == '\'' {
		l.readChar() // skip closing quote
	}

	return out.String()
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
