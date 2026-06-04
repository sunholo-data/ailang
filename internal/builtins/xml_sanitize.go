package builtins

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-STDLIB-XML-LENIENT (v0.24.0): lenient XML repair.
//
// Go's encoding/xml is a strict XML 1.0 parser: a bare '&' that does not begin
// a valid entity reference aborts the entire document. Real-world XML — Office
// content.xml, HTML5, third-party invoicing tools — routinely emits an
// unescaped '&' (company names, "R&D", URL query strings).
//
// _xml_sanitize repairs that single, well-defined offender so std/xml.parseLenient
// can recover the document. It is deliberately conservative (A1 Determinism): it
// ONLY escapes a '&' that does not begin a syntactically valid entity reference.
// Unknown-but-syntactic named entities (&nbsp;) and stray '<' are out of scope —
// see design_docs/planned/m-stdlib-xml-lenient.md.
//
// Strict parse is unchanged; leniency is opt-in (CLAUDE.md: no silent fallbacks).

func init() {
	registerXmlSanitize()
}

func registerXmlSanitize() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_sanitize",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeXmlSanitizeType,
		Impl:    xmlSanitizeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Escape a bare '&' that does not begin a valid entity reference, so strict XML parsers accept real-world input",
			Returns:     "string - sanitized XML",
			Since:       "v0.24.0",
			Stability:   StabilityStable,
			Tags:        []string{"xml", "sanitize", "lenient"},
			Category:    "xml",
		},
	})
	if err != nil {
		panic("failed to register _xml_sanitize: " + err.Error())
	}
}

func makeXmlSanitizeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

func xmlSanitizeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_sanitize: expected string, got %T", args[0])
	}
	return &eval.StringValue{Value: sanitizeBareAmpersands(strVal.Value)}, nil
}

// sanitizeBareAmpersands rewrites every '&' that does NOT begin a valid entity
// reference into "&amp;", in a single O(n) pass. Valid references are passed
// through verbatim, so the function is idempotent and never double-escapes.
//
// The grammar of a "valid" reference (matching XML 1.0 well-formedness):
//   - &#[0-9]+;             decimal char ref
//   - &#x[0-9a-fA-F]+;      hex char ref (x or X)
//   - &[A-Za-z][A-Za-z0-9]*; named ref
//
// All delimiters in that grammar are ASCII; multi-byte UTF-8 sequences have the
// high bit set on every byte, so byte-level scanning never splits a rune.
func sanitizeBareAmpersands(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s // fast path: nothing to repair
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		if s[i] != '&' {
			b.WriteByte(s[i])
			continue
		}
		if isValidEntityAt(s, i) {
			b.WriteByte('&')
		} else {
			b.WriteString("&amp;")
		}
	}
	return b.String()
}

// isValidEntityAt reports whether the '&' at index i begins a syntactically
// valid entity reference (see grammar above). Caller guarantees s[i] == '&'.
func isValidEntityAt(s string, i int) bool {
	j := i + 1
	if j >= len(s) {
		return false // trailing '&'
	}
	if s[j] == '#' {
		j++
		if j < len(s) && (s[j] == 'x' || s[j] == 'X') {
			j++
			start := j
			for j < len(s) && isHexDigit(s[j]) {
				j++
			}
			return j > start && j < len(s) && s[j] == ';'
		}
		start := j
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		return j > start && j < len(s) && s[j] == ';'
	}
	// named reference: [A-Za-z][A-Za-z0-9]*;
	if !isAlpha(s[j]) {
		return false
	}
	j++
	for j < len(s) && isAlphaNum(s[j]) {
		j++
	}
	return j < len(s) && s[j] == ';'
}

func isAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isAlphaNum(c byte) bool {
	return isAlpha(c) || (c >= '0' && c <= '9')
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
