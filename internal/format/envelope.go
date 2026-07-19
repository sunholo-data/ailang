package format

import (
	"fmt"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// envelope.go implements the token-anchored envelope: the formatter-owned
// structure that supplies the structural byte boundaries comment attachment
// needs. It derives all ranges from the byte-accurate lossless scan
// (lexer.CollectComments) plus verified rune-walk conversion of AST/token
// (line, column) positions into byte offsets — NEVER from AST Span fields
// (measured unusable at design time; see the design doc V15–V17).
//
// The two load-bearing premises are:
//  1. Column is a 1-based rune index into NFC-normalized source (lexer.readChar).
//  2. Every parser-recorded Pos is copied verbatim from a real token, so its
//     (line, column) converts EXACTLY to the byte offset of that token's text —
//     except for tokens whose positions point inside string-literal interiors
//     (interpolation-queue tokens), which are clamped to the enclosing region.
//
// Premise 1's exactness is enforced permanently by premise_sweep_test.go over
// the whole example corpus; premise 2's exception class is neutralized by
// literal-region clamping. Any residual inconsistency is a fail-closed
// envelope error, never a silent misattachment.

// EnvelopeError is a fail-closed envelope inconsistency. When one is returned the
// CLI leaves the file byte-identical (exit 2) rather than guess-attaching a
// comment. The Kind enables the envelope-error taxonomy (design §"Fail closed").
type EnvelopeError struct {
	Kind string // taxonomy tag: "anchor-out-of-range", "anchor-no-code-byte", "interp-comment", "unbalanced-bracket"
	Msg  string
}

func (e *EnvelopeError) Error() string { return e.Msg }

func envErr(kind, format string, args ...any) *EnvelopeError {
	return &EnvelopeError{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

// Envelope holds the per-file structural index derived from the normalized
// source bytes.
type Envelope struct {
	src       string                // normalized source
	lineStart []int                 // byte offset of the first byte of each 1-based line (lineStart[0] unused)
	comments  []lexer.Comment       // every real comment, ascending
	regions   []lexer.LiteralRegion // literal regions, ascending, non-overlapping
}

// NewEnvelope builds the envelope for source. The source is normalized here (the
// same boundary the lexer and collector use) so all offsets are consistent.
// It returns an EnvelopeError if the interpolation carve-out fires (a comment
// inside a `${...}` hole) — the caller must fail closed.
func NewEnvelope(source []byte) (*Envelope, error) {
	norm := string(lexer.Normalize(source))
	comments, regions := lexer.CollectComments(source)
	e := &Envelope{
		src:      norm,
		comments: comments,
		regions:  regions,
	}
	e.buildLineStarts()
	if err := e.checkInterpolationCarveOut(); err != nil {
		return nil, err
	}
	return e, nil
}

// buildLineStarts records the byte offset at which each 1-based line begins.
// lineStart[1] == 0; lineStart[n+1] is one past the n-th '\n'.
func (e *Envelope) buildLineStarts() {
	e.lineStart = []int{0, 0} // index 0 unused; line 1 starts at byte 0
	for i := 0; i < len(e.src); i++ {
		if e.src[i] == '\n' {
			e.lineStart = append(e.lineStart, i+1)
		}
	}
}

// Comments returns the scanned comments (ascending by Start).
func (e *Envelope) Comments() []lexer.Comment { return e.comments }

// Source returns the normalized source the envelope indexes.
func (e *Envelope) Source() string { return e.src }

// offsetOf converts a 1-based (line, column) rune position into a byte offset in
// the normalized source, by walking `column-1` runes from the line start. It
// returns an envelope error if the position is out of range.
func (e *Envelope) offsetOf(line, column int) (int, error) {
	if line < 1 || line >= len(e.lineStart) {
		return 0, envErr("anchor-out-of-range", "line %d out of range (1..%d)", line, len(e.lineStart)-1)
	}
	if column < 1 {
		return 0, envErr("anchor-out-of-range", "column %d < 1", column)
	}
	pos := e.lineStart[line]
	for i := 0; i < column-1; i++ {
		if pos >= len(e.src) {
			return 0, envErr("anchor-out-of-range", "column %d past end of line %d", column, line)
		}
		if e.src[pos] == '\n' {
			return 0, envErr("anchor-out-of-range", "column %d past end of line %d", column, line)
		}
		_, size := utf8.DecodeRuneInString(e.src[pos:])
		pos += size
	}
	return pos, nil
}

// AnchorOf converts an AST position to a byte offset, clamping any anchor that
// lands inside a literal region to that region's start (the interpolation-queue
// token class, design V18). It returns an envelope error for an anchor that maps
// to no code byte.
func (e *Envelope) AnchorOf(p ast.Pos) (int, error) {
	off, err := e.offsetOf(p.Line, p.Column)
	if err != nil {
		return 0, err
	}
	if r := e.regionContaining(off); r != nil {
		return r.Start, nil
	}
	return off, nil
}

// regionContaining returns the literal region whose half-open range contains
// off, or nil. Regions are ascending and non-overlapping, so a linear scan is
// correct; callers convert few anchors per file so this is not hot.
func (e *Envelope) regionContaining(off int) *lexer.LiteralRegion {
	for i := range e.regions {
		r := &e.regions[i]
		if off >= r.Start && off < r.End {
			return r
		}
		if r.Start > off {
			break
		}
	}
	return nil
}

// inLiteral reports whether byte offset off is inside any literal region.
func (e *Envelope) inLiteral(off int) bool {
	return e.regionContaining(off) != nil
}

// inStringSpan reports whether byte offset off lies within the full byte span of
// a string/quasiquote literal INCLUDING its interpolation holes and the `${`/`}`
// delimiters — i.e. anywhere between a literal's opening delimiter and its
// closing delimiter. Interpolation-queue token positions (design V18) point
// inside these spans (an IDENT inside `${text}` is anchored at the `{`); they are
// synthesized by the interpolation lexer and are NOT reliable code anchors, so
// the envelope treats the whole string span as opaque for anchoring. A comment
// can never occur inside a string span except in an interpolation hole, which is
// refused fail-closed (checkInterpolationCarveOut), so opacity here is safe.
func (e *Envelope) inStringSpan(off int) bool {
	for _, sp := range e.stringSpans() {
		if off >= sp.start && off < sp.end {
			return true
		}
	}
	return false
}

// stringSpans merges the literal-region map plus interpolation holes into the
// full outer byte span of each string/quasiquote literal. Adjacent regions
// separated only by an interpolation hole (a region End that abuts a `${` and the
// hole running to the next region Start) belong to the SAME string and merge into
// one span.
func (e *Envelope) stringSpans() []interpHole {
	holes := e.interpolationHoles()
	// Collect all region + hole intervals, then merge touching/overlapping ones.
	intervals := make([]interpHole, 0, len(e.regions)+len(holes))
	for _, r := range e.regions {
		intervals = append(intervals, interpHole{start: r.Start, end: r.End})
	}
	for _, h := range holes {
		// A hole's `${` and `}` delimiters bracket it; extend the hole interval to
		// include them so it touches the adjacent regions and merges.
		start := h.start - 2 // include `${`
		if start < 0 {
			start = 0
		}
		intervals = append(intervals, interpHole{start: start, end: h.end + 1}) // include `}`
	}
	if len(intervals) == 0 {
		return nil
	}
	// Sort by start (regions are already ascending; holes interleave). Simple
	// insertion is fine for the small counts involved.
	for i := 1; i < len(intervals); i++ {
		for j := i; j > 0 && intervals[j].start < intervals[j-1].start; j-- {
			intervals[j], intervals[j-1] = intervals[j-1], intervals[j]
		}
	}
	merged := []interpHole{intervals[0]}
	for _, iv := range intervals[1:] {
		last := &merged[len(merged)-1]
		if iv.start <= last.end {
			if iv.end > last.end {
				last.end = iv.end
			}
		} else {
			merged = append(merged, iv)
		}
	}
	return merged
}

// checkInterpolationCarveOut enforces BINDING CONSTRAINT 2: if any comment falls
// inside a `${...}` interpolation hole, the whole file is refused fail-closed.
// The collector reports interior comments as real comments whose spans lie
// OUTSIDE every literal region but INSIDE the byte span of an enclosing string
// literal (between its opening and closing quote). We detect that condition by
// checking whether a code-level comment sits between a literal region that ends
// with an open interpolation and the region that resumes after it — operationally,
// a comment whose Start lies strictly between the first and last literal-region
// boundaries of a single string that contains a hole.
//
// Simpler and exact: a comment is "interior to an interpolation" iff it is NOT
// inside a literal region (it is code) BUT it is bracketed by literal regions on
// BOTH sides that belong to the same string literal — i.e. there is a region
// ending at or before the comment AND a region starting at or after the comment,
// with no newline-delimited top-level code between... which is fragile. Instead
// we reconstruct interpolation spans directly from the region map.
func (e *Envelope) checkInterpolationCarveOut() error {
	holes := e.interpolationHoles()
	for _, c := range e.comments {
		for _, h := range holes {
			if c.Start >= h.start && c.Start < h.end {
				return envErr("interp-comment",
					"comment inside string interpolation ${...} at byte %d is not supported by ailang fmt", c.Start)
			}
		}
	}
	return nil
}

// interpHole is the byte range strictly between a literal segment that abuts a
// `${` and the literal segment that resumes at the matching `}`.
type interpHole struct {
	start, end int
}

// interpolationHoles reconstructs, from the source and region map, the byte
// ranges of interpolation-hole interiors. A hole exists wherever a literal
// region is immediately followed (at its End byte) by a `{` in the source (the
// collector's pushRegion includes the `$` but stops before the `{`), running to
// the byte where the next literal region of the same string resumes.
//
// Rather than pair regions heuristically, we scan the source: every `${` at a
// byte position that is the End of a literal region opens a hole; the matching
// `}` (brace-depth aware, skipping nested literals via the region map) closes it.
func (e *Envelope) interpolationHoles() []interpHole {
	var holes []interpHole
	// Set of region-end offsets for O(1) "does a region end here?" checks.
	regionEnd := make(map[int]bool, len(e.regions))
	for _, r := range e.regions {
		regionEnd[r.End] = true
	}
	i := 0
	for i < len(e.src)-1 {
		if e.src[i] == '$' && e.src[i+1] == '{' && regionEnd[i+1] {
			// Hole opens after the `{`.
			holeStart := i + 2
			end := e.matchInterpClose(holeStart)
			holes = append(holes, interpHole{start: holeStart, end: end})
			i = end
			continue
		}
		i++
	}
	return holes
}

// MinAnchor returns the minimum (leftmost) byte anchor over an AST node's whole
// subtree — the leftmost recorded token position. For `x + 42` the BinaryOp node
// is positioned at the operator `+`, so MinAnchor recovers `x` by walking
// children. It returns an envelope error if the node has no convertible anchor.
func (e *Envelope) MinAnchor(n ast.Node) (int, error) {
	min := -1
	visitAnchors(n, func(p ast.Pos) {
		if p.Line == 0 {
			return
		}
		off, err := e.AnchorOf(p)
		if err != nil {
			// A single unconvertible child anchor is tolerated if others convert;
			// only fail if NO anchor converts (checked after the walk).
			return
		}
		if min == -1 || off < min {
			min = off
		}
	})
	if min == -1 {
		return 0, envErr("anchor-no-code-byte", "node %T has no convertible anchor", n)
	}
	return min, nil
}

// WidenLeft applies closed-class left widening to a child's min-anchor: it moves
// left over a CLOSED token class that can only belong to this child — opening
// brackets whose byte-level match lies at/after the min-anchor, and the modifier
// keywords `export`/`pure`/`extern`. It STOPS at the hard left wall: the nearest
// enclosing-list open delimiter (parentOpen, a byte offset, or -1 for the file
// level). A child may never widen across a delimiter its PARENT owns. Comments
// before the first child therefore attach to the parent boundary 0, not the child.
//
// This is the load-bearing "hard left wall" clause (design V-Rev3, gemini-3-1-pro
// objection): in `[ /* C */ x ]`, x must NOT consume the list's `[`.
func (e *Envelope) WidenLeft(minAnchor, parentOpen int) int {
	pos := minAnchor
	for pos > 0 {
		// Skip whitespace immediately left.
		j := pos - 1
		for j >= 0 && (e.src[j] == ' ' || e.src[j] == '\t' || e.src[j] == '\n' || e.src[j] == '\r') {
			j--
		}
		if j < 0 {
			break
		}
		// Hard left wall: never cross the parent's open delimiter.
		if parentOpen >= 0 && j <= parentOpen {
			break
		}
		// Never widen into a literal region or comment.
		if e.inStringSpan(j) {
			break
		}
		c := e.src[j]
		switch {
		case c == '[' || c == '(' || c == '{':
			// An opening bracket belongs to this child only if its match lies
			// at/after the current widened start (it wraps part of this child).
			match := e.matchBracket(j)
			if match >= pos {
				pos = j
				continue
			}
			return pos
		case isWordBoundaryLeft(e.src, j):
			// Check for a modifier keyword ending at j+1.
			if kw, start := e.wordEndingAt(j + 1); isModifierKeyword(kw) {
				pos = start
				continue
			}
			return pos
		default:
			return pos
		}
	}
	return pos
}

// matchBracket returns the byte offset of the closing bracket matching an opening
// bracket at open, doing byte-level matching over CODE bytes only (literal-region
// and interpolation-hole interiors are skipped via stringSpans). Returns -1 if
// unbalanced.
func (e *Envelope) matchBracket(open int) int {
	var close byte
	switch e.src[open] {
	case '[':
		close = ']'
	case '(':
		close = ')'
	case '{':
		close = '}'
	default:
		return -1
	}
	depth := 0
	i := open
	for i < len(e.src) {
		if e.inStringSpan(i) {
			// Jump past the string span.
			i = e.skipStringSpanFrom(i)
			continue
		}
		c := e.src[i]
		if c == e.src[open] {
			depth++
		} else if c == close {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// skipStringSpanFrom returns the byte offset just past the string span containing
// off, or off+1 if off is not in a span.
func (e *Envelope) skipStringSpanFrom(off int) int {
	for _, sp := range e.stringSpans() {
		if off >= sp.start && off < sp.end {
			return sp.end
		}
	}
	return off + 1
}

// wordEndingAt returns the identifier/keyword word ending exclusively at `end`
// and its start offset. Returns "" if the byte before `end` is not a word byte.
func (e *Envelope) wordEndingAt(end int) (string, int) {
	if end <= 0 || end > len(e.src) {
		return "", end
	}
	if !isWordByte(e.src[end-1]) {
		return "", end
	}
	start := end
	for start > 0 && isWordByte(e.src[start-1]) {
		start--
	}
	return e.src[start:end], start
}

// isWordByte reports whether b can appear in an identifier/keyword.
func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// isWordBoundaryLeft reports whether the byte at j is the last byte of a word.
func isWordBoundaryLeft(src string, j int) bool {
	return j >= 0 && j < len(src) && isWordByte(src[j])
}

// isModifierKeyword reports whether kw is a closed-class declaration modifier
// that left-widening may absorb into a child.
func isModifierKeyword(kw string) bool {
	switch kw {
	case "export", "pure", "extern":
		return true
	default:
		return false
	}
}

// matchInterpClose returns the byte offset of the matching `}` for a hole opened
// at holeStart (one past `${`), tracking brace depth and skipping literal regions
// (nested strings) so a `}` inside a nested string does not close the hole.
func (e *Envelope) matchInterpClose(holeStart int) int {
	depth := 1
	i := holeStart
	for i < len(e.src) {
		if r := e.regionContaining(i); r != nil {
			i = r.End
			continue
		}
		switch e.src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i // exclusive of the `}` is the hole end
			}
		}
		i++
	}
	return i // unterminated: run to EOF
}
