package builtins

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-STDLIB-XML-LENIENT: _xml_sanitize escapes bare '&' that does not begin a
// valid entity reference, so strict encoding/xml accepts real-world input.

func TestSanitizeBareAmpersands(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// The reported production failure.
		{"bare amp in text", "Apex Consulting & Partners", "Apex Consulting &amp; Partners"},
		{"R&D", "R&D", "R&amp;D"},
		{"url query", "?a=1&b=2", "?a=1&amp;b=2"},

		// Valid entities must pass through unchanged (no double-escape).
		{"amp entity", "&amp;", "&amp;"},
		{"lt entity", "&lt;", "&lt;"},
		{"gt entity", "&gt;", "&gt;"},
		{"apos entity", "&apos;", "&apos;"},
		{"quot entity", "&quot;", "&quot;"},
		{"numeric dec", "&#123;", "&#123;"},
		{"numeric hex lower", "&#xab;", "&#xab;"},
		{"numeric hex upper", "&#XAB;", "&#XAB;"},
		{"named entity syntactic", "&nbsp;", "&nbsp;"}, // syntactically valid; left as-is (v1 scope)

		// Edge cases that are NOT valid entities → escape.
		{"trailing amp EOF", "foo&", "foo&amp;"},
		{"amp space", "a & b", "a &amp; b"},
		{"amp hash no semicolon", "&#", "&amp;#"},
		{"amp hash digits no semicolon", "&#123", "&amp;#123"},
		{"amp hash x no hex", "&#x;", "&amp;#x;"},
		{"amp name no semicolon", "&amp", "&amp;amp"},
		{"empty name", "&;", "&amp;;"},

		// Mixed: valid + bare in same string.
		{"mixed", "&amp; & &lt;", "&amp; &amp; &lt;"},

		// No ampersand at all (fast path).
		{"no amp", "<r><p>hello</p></r>", "<r><p>hello</p></r>"},
		{"empty", "", ""},

		// UTF-8 multibyte must be preserved untouched.
		{"utf8 with bare amp", "café & thé", "café &amp; thé"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBareAmpersands(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeBareAmpersands(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Idempotence: sanitizing twice equals sanitizing once.
func TestSanitizeIdempotent(t *testing.T) {
	inputs := []string{
		"Apex Consulting & Partners",
		"&amp; & &lt; &#123; &#xAB; &nbsp;",
		"R&D and a=1&b=2",
		"trailing&",
	}
	for _, in := range inputs {
		once := sanitizeBareAmpersands(in)
		twice := sanitizeBareAmpersands(once)
		if once != twice {
			t.Errorf("not idempotent for %q: once=%q twice=%q", in, once, twice)
		}
	}
}

// The reporter's exact repro must round-trip through sanitize + strict decode.
func TestSanitizedXMLParsesStrictly(t *testing.T) {
	dirty := "<r><p>Apex Consulting & Partners</p></r>"

	// Sanity: the raw dirty input is rejected by strict encoding/xml.
	if err := decodeAll(dirty); err == nil {
		t.Fatalf("expected strict decode of dirty input to fail, but it succeeded")
	}

	clean := sanitizeBareAmpersands(dirty)
	if err := decodeAll(clean); err != nil {
		t.Fatalf("sanitized input still fails strict decode: %v (sanitized=%q)", err, clean)
	}
	if !strings.Contains(clean, "&amp;") {
		t.Errorf("expected bare & to be escaped, got %q", clean)
	}
}

func decodeAll(s string) error {
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func TestXmlSanitizeImpl_TypeError(t *testing.T) {
	_, err := xmlSanitizeImpl(nil, []eval.Value{&eval.IntValue{Value: 42}})
	if err == nil {
		t.Fatal("expected error for non-string arg")
	}
	if !strings.Contains(err.Error(), "_xml_sanitize") {
		t.Errorf("expected error to mention _xml_sanitize, got %v", err)
	}
}

func TestXmlSanitizeImpl_Happy(t *testing.T) {
	res, err := xmlSanitizeImpl(nil, []eval.Value{&eval.StringValue{Value: "R&D"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv, ok := res.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", res)
	}
	if sv.Value != "R&amp;D" {
		t.Errorf("got %q, want %q", sv.Value, "R&amp;D")
	}
}
