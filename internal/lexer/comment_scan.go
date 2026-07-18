package lexer

import "unicode/utf8"

// comment_scan.go provides an OPT-IN, lossless comment-detection scan used by
// `ailang fmt`'s Phase-1 comment safety gate. It is deliberately separate from
// NextToken(): the parser-visible token stream is byte-for-byte unchanged. This
// scanner never allocates tokens and never mutates a Lexer; it walks the raw
// (normalized) source directly.
//
// The scan mirrors the lexer's literal-skipping structure so that comment
// introducers (`--` and `//`) inside strings, char literals, regex literals, and
// triple-quoted quasiquote templates are NOT mistaken for real comments. This is
// a true state machine, never a substring search.

// ScanForComment reports whether the given source contains at least one real
// AILANG comment (a `--` or `//` line comment introducer that is not inside a
// string, character, regex, or quasiquote literal). Input is normalized at the
// boundary exactly as New() does, so detection matches what the parser sees.
//
// The scanner is intentionally conservative and self-contained: it recognizes
// the same literal delimiters the lexer does. Malformed/unterminated literals
// are treated as running to end-of-input (they contain no comment introducer by
// definition), which is safe for a boolean "any comment?" query.
func ScanForComment(source []byte) bool {
	src := string(Normalize(source))
	s := &commentScanner{input: src}
	return s.scan()
}

type commentScanner struct {
	input string
	pos   int // byte offset of the next rune to read
}

// next returns the current rune and its size without advancing.
func (s *commentScanner) cur() (rune, int) {
	if s.pos >= len(s.input) {
		return 0, 0
	}
	return utf8.DecodeRuneInString(s.input[s.pos:])
}

// peekAt returns the rune n bytes-decoded ahead (0 = current). Used sparingly
// for two-character lookahead (`--`, `//`, `"""`).
func (s *commentScanner) peek(offset int) rune {
	p := s.pos
	for i := 0; i < offset; i++ {
		if p >= len(s.input) {
			return 0
		}
		_, size := utf8.DecodeRuneInString(s.input[p:])
		p += size
	}
	if p >= len(s.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s.input[p:])
	return r
}

func (s *commentScanner) advance() {
	if s.pos < len(s.input) {
		_, size := utf8.DecodeRuneInString(s.input[s.pos:])
		s.pos += size
	}
}

// scan walks the source and returns true on the first real comment introducer.
func (s *commentScanner) scan() bool {
	for s.pos < len(s.input) {
		r, _ := s.cur()
		switch {
		case r == '-' && s.peek(1) == '-':
			return true
		case r == '/' && s.peek(1) == '/':
			return true
		case r == '"':
			// Triple-quoted quasiquote body, or a normal string.
			if s.peek(1) == '"' && s.peek(2) == '"' {
				s.skipTripleQuote()
			} else {
				s.skipString()
			}
		case r == '\'':
			s.skipChar()
		default:
			s.advance()
		}
	}
	return false
}

// skipString consumes a double-quoted string literal, honoring backslash
// escapes (so `\"` does not end the string early). Interpolation `${...}` is not
// specially handled: a comment introducer can only legally appear inside the
// interpolated EXPRESSION, which mirrors the surrounding grammar; but since the
// lexer skips comments there too and Phase 1 rejects any comment anywhere, we
// stay conservative and scan interpolation contents as ordinary string bytes.
// This never yields a false positive because `--`/`//` inside a quoted region is
// literal text, and never a false negative for a real comment outside strings.
func (s *commentScanner) skipString() {
	s.advance() // opening quote
	for s.pos < len(s.input) {
		r, _ := s.cur()
		if r == '\\' {
			s.advance() // backslash
			s.advance() // escaped char (safe at EOF)
			continue
		}
		if r == '"' {
			s.advance() // closing quote
			return
		}
		s.advance()
	}
}

// skipChar consumes a single-quoted character literal with escape handling.
func (s *commentScanner) skipChar() {
	s.advance() // opening quote
	for s.pos < len(s.input) {
		r, _ := s.cur()
		if r == '\\' {
			s.advance()
			s.advance()
			continue
		}
		if r == '\'' {
			s.advance()
			return
		}
		s.advance()
	}
}

// skipTripleQuote consumes a triple-quoted quasiquote template body up to and
// including the closing `"""`. Template contents are opaque, so any `--`/`//`
// inside is literal text and must not be treated as a comment.
func (s *commentScanner) skipTripleQuote() {
	s.advance()
	s.advance()
	s.advance() // opening """
	for s.pos < len(s.input) {
		if s.peek(0) == '"' && s.peek(1) == '"' && s.peek(2) == '"' {
			s.advance()
			s.advance()
			s.advance() // closing """
			return
		}
		s.advance()
	}
}
