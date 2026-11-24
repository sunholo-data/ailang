package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

func TestMatchArmBlockSuggestion(t *testing.T) {
	t.Run("bare let without braces in match arm emits PAR_MATCH_ARM_BLOCK", func(t *testing.T) {
		input := `module test
func f(xs: [int]) -> int = match xs {
  [] => 0,
  x :: rest =>
    let y = x * 2
    let z = y + 1
    z
}`
		p := New(lexer.New(input, "test.ail"))
		p.Parse()
		found := false
		for _, e := range p.Errors() {
			if strings.Contains(e.Error(), "PAR_MATCH_ARM_BLOCK") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected PAR_MATCH_ARM_BLOCK for bare let in match arm; got: %v", p.Errors())
		}
	})

	t.Run("semicolon sequence without braces in match arm emits PAR_MATCH_ARM_BLOCK", func(t *testing.T) {
		input := `module test
func f(x: int) -> () ! {IO} = match x {
  0 => println("a"); println("b"),
  _ => ()
}`
		p := New(lexer.New(input, "test.ail"))
		p.Parse()
		found := false
		for _, e := range p.Errors() {
			if strings.Contains(e.Error(), "PAR_MATCH_ARM_BLOCK") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected PAR_MATCH_ARM_BLOCK for semicolon sequence in match arm; got: %v", p.Errors())
		}
	})

	t.Run("braced multi-let match arm has no error", func(t *testing.T) {
		input := `module test
func f(xs: [int]) -> int = match xs {
  [] => 0,
  x :: rest => {
    let y = x * 2
    let z = y + 1
    z
  }
}`
		p := New(lexer.New(input, "test.ail"))
		p.Parse()
		if len(p.Errors()) > 0 {
			t.Errorf("expected no errors for braced match arm; got: %v", p.Errors())
		}
	})

	t.Run("let-in match arm has no error", func(t *testing.T) {
		input := `module test
func f(xs: [int]) -> int = match xs {
  [] => 0,
  x :: rest => let y = x * 2 in y + 1
}`
		p := New(lexer.New(input, "test.ail"))
		p.Parse()
		if len(p.Errors()) > 0 {
			t.Errorf("expected no errors for let-in match arm; got: %v", p.Errors())
		}
	})
}

func TestLetTupleDestructuringSuggestion(t *testing.T) {
	input := `module test
func f(p: (int, int)) -> int {
  letrec (a, b) = p
  a + b
}`
	p := New(lexer.New(input, "test.ail"))
	p.Parse()
	found := false
	for _, e := range p.Errors() {
		if strings.Contains(e.Error(), "PAR_LET_TUPLE") && strings.Contains(e.Error(), "match") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected PAR_LET_TUPLE with match suggestion for letrec tuple; got: %v", p.Errors())
	}
}
