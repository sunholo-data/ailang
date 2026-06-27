package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// TestMatchArrowSuggestion (M-AGENT-ERGONOMICS) — a `->` where a match arm expects `=>` is a
// common dialect slip (agents revert to type-signature/lambda arrow habits). It must produce a
// precise, actionable suggestion ("use '=>'"), not a bare "expected =>, got ->".
func TestMatchArrowSuggestion(t *testing.T) {
	input := `module test
export func f(x: int) -> string = match x { 1 -> "one", _ -> "other" }`
	l := lexer.New(input, "test.ail")
	p := New(l)
	p.Parse()
	errs := p.Errors()
	if len(errs) == 0 {
		t.Fatalf("expected a match-arrow suggestion error, got none")
	}
	found := false
	for _, e := range errs {
		m := e.Error()
		if strings.Contains(m, "=>") && strings.Contains(m, "->") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a '=>' vs '->' suggestion; got: %v", errs)
	}
}

// TestMatchArrowCorrect — the correct `=>` arm must NOT trigger the suggestion (no false positive).
func TestMatchArrowCorrect(t *testing.T) {
	input := `module test
export func f(x: int) -> string = match x { 1 => "one", _ => "other" }`
	l := lexer.New(input, "test.ail")
	p := New(l)
	p.Parse()
	for _, e := range p.Errors() {
		if strings.Contains(e.Error(), "PAR_MATCH_ARROW") {
			t.Errorf("correct '=>' arms should not trigger the arrow suggestion; got: %v", e)
		}
	}
}
