package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// TestPAR020_MissingBlockSemicolon guards M-AILANG-ERROR-QUALITY: a missing ';'
// between block statements (the mirror of PAR017's "extra ';' in =-body") must
// produce an actionable PAR020 error naming the fix, not a bare
// "expected }, got X". This is the #1 unactionable thrash-causer on small models
// — config_file_parser burned 66 agent turns on a generic "expected }, got if".
func TestPAR020_MissingBlockSemicolon(t *testing.T) {
	t.Run("func body missing semicolon (the 66-turn pattern)", func(t *testing.T) {
		input := "module t\n" +
			"pure func f(n: int) -> int {\n" +
			"  let x = n\n" + // <-- missing ';'
			"  if x > 0 then 1 else 0\n" +
			"}\n"
		err := firstParserErrorWithCode(t, input, "PAR020")
		if err == nil {
			t.Fatal("expected PAR020 for a missing ';' between block statements")
		}
		msg := err.Error()
		if !strings.Contains(msg, "missing ';'") {
			t.Errorf("message should name the missing ';': %s", msg)
		}
		if !strings.Contains(msg, "separated by `;`") {
			t.Errorf("message should explain block-statement separation: %s", msg)
		}
		if len(err.Suggestions) == 0 {
			t.Error("PAR020 should carry concrete fix suggestions")
		}
	})

	t.Run("missing semicolon before another let", func(t *testing.T) {
		input := "module t\n" +
			"pure func f(n: int) -> int {\n" +
			"  let x = n\n" + // <-- missing ';'
			"  let y = x\n" +
			"  y\n" +
			"}\n"
		if firstParserErrorWithCode(t, input, "PAR020") == nil {
			t.Error("expected PAR020 when a let-statement is followed by another without ';'")
		}
	})

	t.Run("valid block does NOT trigger PAR020", func(t *testing.T) {
		input := "module t\n" +
			"pure func f(n: int) -> int { let x = n; let y = x + 1; y }\n"
		if err := firstParserErrorWithCode(t, input, "PAR020"); err != nil {
			t.Errorf("valid semicolon-separated block must not trigger PAR020: %s", err.Error())
		}
	})

	t.Run("single-expression body does NOT trigger PAR020", func(t *testing.T) {
		input := "module t\n" +
			"pure func f(n: int) -> int { n + 1 }\n"
		if err := firstParserErrorWithCode(t, input, "PAR020"); err != nil {
			t.Errorf("single-expression body must not trigger PAR020: %s", err.Error())
		}
	})
}

// firstParserErrorWithCode parses input and returns the first *ParserError whose
// Code matches, or nil if none.
func firstParserErrorWithCode(t *testing.T, input, code string) *ParserError {
	t.Helper()
	l := lexer.New(input, "<test>")
	p := New(l)
	_ = p.Parse()
	for _, e := range p.errors {
		if pe, ok := e.(*ParserError); ok && pe.Code == code {
			return pe
		}
	}
	return nil
}
