package lexer

import "unicode/utf8"

// comment_scan.go provides an OPT-IN, lossless comment scan used by `ailang fmt`.
// It is deliberately separate from NextToken(): the parser-visible token stream
// is byte-for-byte unchanged. This scanner never allocates parser tokens and
// never mutates a Lexer; it walks the raw (normalized) source directly.
//
// The scan mirrors the lexer's literal-skipping structure so that comment
// introducers (`--` and `//`) inside strings, char literals, regex literals, and
// triple-quoted quasiquote templates are NOT mistaken for real comments. This is
// a true state machine, never a substring search.
//
// Phase 1 (ScanForComment) answered a single boolean: "any real comment?".
// Phase 2 (CollectComments) additionally yields:
//   - every comment's exact byte span + kind (the formatter's trivia source), and
//   - a literal-region map: the byte ranges of every string/char/regex/
//     quasiquote literal, EXCLUDING the interiors of `${...}` interpolation holes
//     (which are CODE, not literal text). A comment introducer that falls inside
//     an interpolation hole is therefore reported as a real comment whose byte
//     span lies OUTSIDE every literal region — which is exactly the signal the
//     formatter's fail-closed interpolation carve-out keys on.
//
// Because the skipString state machine now tracks `${...}` nesting (fixing the
// Phase-1 naive first-`"` termination, design V19), the literal-region map is
// correct for nested interpolation, so the carve-out cannot miss an interior
// comment.

// CommentKind distinguishes the two comment introducer spellings AILANG accepts.
type CommentKind int

const (
	// LineCommentDash is a `--`-introduced line comment.
	LineCommentDash CommentKind = iota
	// LineCommentSlash is a `//`-introduced line comment.
	LineCommentSlash
)

// Comment is a single scanned comment with byte-exact span into the NORMALIZED
// source. Start is the byte offset of the introducer's first byte; End is the
// exclusive byte offset one past the last comment byte (the newline, or EOF, is
// NOT included). Text is the verbatim comment bytes [Start, End) from normalized
// source, including the introducer.
type Comment struct {
	Kind       CommentKind
	Text       string
	Start, End int // byte offsets into the normalized source
}

// LiteralRegion is a half-open byte range [Start, End) covering a string, char,
// regex, or quasiquote literal in the NORMALIZED source. Interpolation-hole
// interiors are NOT part of any region (they are code). Regions are emitted in
// ascending, non-overlapping order.
type LiteralRegion struct {
	Start, End int
}

// ScanForComment reports whether the given source contains at least one real
// AILANG comment. Retained for the Phase-1 boolean preflight; implemented on top
// of the collector so the two can never drift.
func ScanForComment(source []byte) bool {
	comments, _ := CollectComments(source)
	return len(comments) > 0
}

// CollectComments walks the normalized source and returns every real comment
// (byte-exact span + kind) and the literal-region map. Input is normalized at the
// boundary exactly as New() does, so offsets match what the parser and the
// formatter's token-anchored envelope see.
//
// A comment introducer inside an interpolation hole (`${ -- ... }`) is a REAL
// comment: the hole interior is code, not covered by any literal region, so the
// returned comment's span lies outside every region — the fail-closed carve-out
// signal. Malformed/unterminated literals run to end-of-input.
func CollectComments(source []byte) (comments []Comment, regions []LiteralRegion) {
	src := string(Normalize(source))
	s := &commentScanner{input: src}
	s.scan()
	return s.comments, s.regions
}

type commentScanner struct {
	input    string
	pos      int // byte offset of the next rune to read
	comments []Comment
	regions  []LiteralRegion
}

// cur returns the current rune and its size without advancing.
func (s *commentScanner) cur() (rune, int) {
	if s.pos >= len(s.input) {
		return 0, 0
	}
	return utf8.DecodeRuneInString(s.input[s.pos:])
}

// peek returns the rune n bytes-decoded ahead (0 = current). Used sparingly for
// two-/three-character lookahead (`--`, `//`, `"""`).
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

// scan walks the source at code level, recording comments and pushing into
// literal-skip subroutines for string/char/quasiquote literals.
func (s *commentScanner) scan() {
	for s.pos < len(s.input) {
		r, _ := s.cur()
		switch {
		case r == '-' && s.peek(1) == '-':
			s.collectLineComment(LineCommentDash)
		case r == '/' && s.peek(1) == '/':
			s.collectLineComment(LineCommentSlash)
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
}

// collectLineComment records a `--`/`//` comment from the introducer to the end
// of the line (exclusive of the newline), then leaves pos at the newline (or
// EOF) so scan continues at code level.
func (s *commentScanner) collectLineComment(kind CommentKind) {
	start := s.pos
	// Consume to end of line.
	for s.pos < len(s.input) {
		r, _ := s.cur()
		if r == '\n' {
			break
		}
		s.advance()
	}
	end := s.pos
	s.comments = append(s.comments, Comment{
		Kind:  kind,
		Text:  s.input[start:end],
		Start: start,
		End:   end,
	})
}

// skipString consumes a double-quoted string literal, honoring backslash escapes
// (so `\"` does not end the string early) AND `${...}` interpolation nesting.
// The literal-region map records the string bytes but EXCLUDES the interior of
// each interpolation hole: inside `${...}` the content is code, so it is scanned
// recursively (nested strings, and — critically — real comments) rather than
// treated as opaque literal bytes. This fixes the Phase-1 naive early-termination
// at the first unescaped `"` (design V19: nested interpolation in
// directory_ops.ail) and makes the interpolation carve-out exact.
func (s *commentScanner) skipString() {
	start := s.pos
	s.advance() // opening quote
	// regionStart tracks the start of the current literal SEGMENT (a maximal run
	// of literal bytes uninterrupted by an interpolation hole).
	regionStart := start
	for s.pos < len(s.input) {
		r, _ := s.cur()
		switch {
		case r == '\\':
			s.advance() // backslash
			s.advance() // escaped char (safe at EOF)
		case r == '$' && s.peek(1) == '{':
			// Close the literal segment before the hole (the `$` is literal; the
			// `{` opens code). The `${` and matching `}` themselves are delimiters,
			// not literal text, and not code either — treat them as region edges.
			s.pushRegion(regionStart, s.pos+len("$")) // include the `$` in the literal segment
			s.advance()                               // '$'
			s.advance()                               // '{'
			s.scanInterpolation()                     // scans hole interior at CODE level
			regionStart = s.pos                       // literal resumes after the closing '}'
		case r == '"':
			s.advance() // closing quote
			s.pushRegion(regionStart, s.pos)
			return
		default:
			s.advance()
		}
	}
	// Unterminated: region runs to EOF.
	s.pushRegion(regionStart, s.pos)
}

// scanInterpolation scans the interior of a `${...}` hole at CODE level, so real
// comments and nested strings are handled correctly, tracking brace depth to find
// the matching close. On return, pos is one past the hole's closing `}`.
func (s *commentScanner) scanInterpolation() {
	depth := 1
	for s.pos < len(s.input) && depth > 0 {
		r, _ := s.cur()
		switch {
		case r == '-' && s.peek(1) == '-':
			s.collectLineComment(LineCommentDash)
		case r == '/' && s.peek(1) == '/':
			s.collectLineComment(LineCommentSlash)
		case r == '"':
			if s.peek(1) == '"' && s.peek(2) == '"' {
				s.skipTripleQuote()
			} else {
				s.skipString()
			}
		case r == '\'':
			s.skipChar()
		case r == '{':
			depth++
			s.advance()
		case r == '}':
			depth--
			s.advance()
		default:
			s.advance()
		}
	}
}

// skipChar consumes a single-quoted character literal with escape handling.
func (s *commentScanner) skipChar() {
	start := s.pos
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
			s.pushRegion(start, s.pos)
			return
		}
		s.advance()
	}
	s.pushRegion(start, s.pos)
}

// skipTripleQuote consumes a triple-quoted quasiquote template body up to and
// including the closing `"""`. Template contents are opaque literal text, so any
// `--`/`//` (or `${`) inside is literal and must not be treated as a comment.
func (s *commentScanner) skipTripleQuote() {
	start := s.pos
	s.advance()
	s.advance()
	s.advance() // opening """
	for s.pos < len(s.input) {
		if s.peek(0) == '"' && s.peek(1) == '"' && s.peek(2) == '"' {
			s.advance()
			s.advance()
			s.advance() // closing """
			s.pushRegion(start, s.pos)
			return
		}
		s.advance()
	}
	s.pushRegion(start, s.pos)
}

// pushRegion appends a non-empty literal region, preserving ascending order.
func (s *commentScanner) pushRegion(start, end int) {
	if end > start {
		s.regions = append(s.regions, LiteralRegion{Start: start, End: end})
	}
}
