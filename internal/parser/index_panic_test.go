package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// TestNoParserPanicOnIndexAccess is a regression guard: index access `x[i]` in a function
// body previously made parseFunctionDeclaration call body.Position() on a nil body, crashing
// the parser with an internal PAR999 panic instead of emitting a clean parse error.
// (Surfaced by an AI-generated docx parser using s[0]; see motoko-harness-analysis-log.md.)
func TestNoParserPanicOnIndexAccess(t *testing.T) {
	input := "module test/idx\nexport func f(s: string) -> int ! {} = { let c = s[0]; 0 }\n"
	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.Parse() // must not panic
	errs := p.Errors()
	if len(errs) == 0 {
		t.Fatal("expected a clean parse error for index access x[i], got none")
	}
	for _, e := range errs {
		if strings.Contains(e.Error(), "panic") || strings.Contains(e.Error(), "PAR999") {
			t.Errorf("parser panicked instead of emitting a clean error: %v", e)
		}
	}
}
