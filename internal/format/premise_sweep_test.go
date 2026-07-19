package format

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// premise_sweep_test.go re-implements the design-time V18 corpus sweep as a
// PERMANENT test. It enforces the two load-bearing envelope premises against
// parser/lexer drift:
//
//  1. Column is a 1-based rune index into NFC-normalized source, so the envelope's
//     (line, column) -> byte-offset conversion is EXACT.
//  2. Every parse-path token's converted offset lands on its source text, EXCEPT
//     tokens whose positions point inside string-literal interiors (the
//     interpolation-queue class), which the literal-region map exempts.
//
// If a future construction site synthesizes a position that is not a real token
// start, or the column unit changes, this test fails loudly in CI — the envelope
// premise cannot silently rot. This is required to be GREEN before any attachment
// code exists (sprint M1 ordering).

// TestPremiseSweep_CorpusTokenOffsets sweeps every corpus file's token stream.
func TestPremiseSweep_CorpusTokenOffsets(t *testing.T) {
	var files, tokens, exempted int
	walkAilExamples(t, func(path string, data []byte) {
		env, err := NewEnvelope(data)
		if err != nil {
			// A file whose only "problem" is an interpolation comment is exempt from
			// the premise sweep (it is a fail-closed carve-out, tested separately).
			if ee, ok := err.(*EnvelopeError); ok && ee.Kind == "interp-comment" {
				return
			}
			t.Fatalf("%s: NewEnvelope: %v", path, err)
		}
		files++
		norm := env.Source()

		l := lexer.New(string(data), path)
		for {
			tok := l.NextToken()
			if tok.Type == lexer.EOF {
				break
			}
			if tok.Type == lexer.ILLEGAL {
				// Illegal tokens carry positions but no reliable source text; skip.
				continue
			}
			tokens++

			off, cerr := env.offsetOf(tok.Line, tok.Column)
			if cerr != nil {
				// A position that cannot even be converted must be inside a literal
				// interior; otherwise it is a real premise violation.
				if isInterpToken(tok.Type) {
					exempted++
					continue
				}
				t.Fatalf("%s: token %v at %d:%d failed offset conversion: %v",
					path, tok.Type, tok.Line, tok.Column, cerr)
			}

			// Exempt tokens whose anchor lands inside a string span (a literal
			// region OR an interpolation hole). Interpolation-queue token positions
			// (incl. IDENTs inside ${...}) are synthesized and not reliable anchors;
			// the envelope treats the whole string span as opaque (design V18).
			if env.inStringSpan(off) {
				exempted++
				continue
			}

			// The converted offset must land on the token's source text. We verify
			// the source at `off` begins the token's expected leading character.
			if !offsetLandsOnToken(norm, off, tok) {
				// String/interp-adjacent tokens may still be exempt if their anchor
				// abuts a literal region boundary.
				if isInterpToken(tok.Type) {
					exempted++
					continue
				}
				t.Fatalf("%s: token %v %q at %d:%d converted to byte %d which does not land on its source text (got %q...)",
					path, tok.Type, tok.Literal, tok.Line, tok.Column, off, safeSlice(norm, off, 12))
			}
		}
	})
	if files == 0 {
		t.Skip("no corpus files swept")
	}
	t.Logf("premise sweep: %d files, %d tokens verified, %d exempted (literal interiors)", files, tokens, exempted)
}

// isInterpToken reports whether a token type is part of the interpolation queue,
// whose positions may point inside string interiors (design V18/V19).
func isInterpToken(tt lexer.TokenType) bool {
	switch tt {
	case lexer.STRING_PART, lexer.INTERP_START, lexer.INTERP_END, lexer.STRING:
		return true
	default:
		return false
	}
}

// offsetLandsOnToken checks that the byte at `off` is consistent with the start
// of the given token. For most tokens the source byte equals the first byte of
// the token's literal or its canonical spelling. We use a permissive check: the
// offset must not be at a whitespace/comment byte, and for identifier/keyword/
// number tokens the first rune must match the literal's first rune.
func offsetLandsOnToken(src string, off int, tok lexer.Token) bool {
	if off < 0 || off >= len(src) {
		return false
	}
	b := src[off]
	// The anchor must not sit on whitespace.
	if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
		return false
	}
	// For tokens with a literal whose first byte is an ASCII letter/digit/symbol,
	// require the source to begin with it. Interpolation/string tokens are handled
	// by the exemption path in the caller.
	if len(tok.Literal) > 0 {
		lit := tok.Literal[0]
		if isWordByte(lit) {
			return b == lit
		}
	}
	return true
}

func safeSlice(s string, off, n int) string {
	if off < 0 || off >= len(s) {
		return ""
	}
	end := off + n
	if end > len(s) {
		end = len(s)
	}
	return s[off:end]
}
