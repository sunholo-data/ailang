package parser

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

func TestMatchArmBlockSuggestion(t *testing.T) {
	t.Run("unbraced newline let chain in match arm parses identically to braced block", func(t *testing.T) {
		unbraced := `module test
func f(xs: [int]) -> int = match xs {
  [] => 0,
  x :: rest =>
    let y = x * 2
    let z = y + 1
    z
}`
		braced := `module test
func f(xs: [int]) -> int = match xs {
  [] => 0,
  x :: rest => {
    let y = x * 2
    let z = y + 1
    z
  }
}`
		uBody := funcBody(t, parseOK(t, unbraced))
		bBody := funcBody(t, parseOK(t, braced))
		if diff := cmp.Diff(bBody, uBody, ignorePos); diff != "" {
			t.Errorf("unbraced match arm AST differs from braced (-braced +unbraced):\n%s", diff)
		}
	})

	t.Run("unbraced semicolon sequence in match arm parses identically to braced block", func(t *testing.T) {
		unbraced := `module test
func f(x: int) -> () ! {IO} = match x {
  0 => println("a"); println("b"),
  _ => ()
}`
		braced := `module test
func f(x: int) -> () ! {IO} = match x {
  0 => { println("a"); println("b") },
  _ => ()
}`
		uBody := funcBody(t, parseOK(t, unbraced))
		bBody := funcBody(t, parseOK(t, braced))
		if diff := cmp.Diff(bBody, uBody, ignorePos); diff != "" {
			t.Errorf("unbraced semicolon match arm AST differs from braced (-braced +unbraced):\n%s", diff)
		}
	})

	t.Run("dangling bare let without body expression in match arm emits PAR_MATCH_ARM_BLOCK", func(t *testing.T) {
		input := `module test
func f(xs: [int]) -> int = match xs {
  [] => 0,
  x :: rest => let y = x * 2
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
			t.Errorf("expected PAR_MATCH_ARM_BLOCK for dangling bare let in match arm; got: %v", p.Errors())
		}
	})

	t.Run("comma-less match arms with unbraced let chain and nested match", func(t *testing.T) {
		input := `module test
func f(xs: [int]) -> int = match xs {
  x :: rest =>
    let y = x * 2
    match y {
      0 =>
        let z = 10
        z
      n => n + 1
    }
  [] => 0
}`
		f := parseOK(t, input)
		blk, ok := f.Funcs[0].Body.(*ast.Block)
		if !ok || len(blk.Exprs) != 1 {
			t.Fatalf("expected 1-expr Block wrapping match, got %T", f.Funcs[0].Body)
		}
		m, ok := blk.Exprs[0].(*ast.Match)
		if !ok || len(m.Cases) != 2 {
			t.Fatalf("expected Match with 2 cases, got %T (%v)", blk.Exprs[0], m)
		}
		armBlk, ok := m.Cases[0].Body.(*ast.Block)
		if !ok || len(armBlk.Exprs) != 2 {
			t.Fatalf("expected 2-expr Block in first arm, got %T", m.Cases[0].Body)
		}
		innerMatch, ok := armBlk.Exprs[1].(*ast.Match)
		if !ok || len(innerMatch.Cases) != 2 {
			t.Fatalf("expected nested Match with 2 cases, got %T", armBlk.Exprs[1])
		}
	})

	t.Run("destructuring let in unbraced match arm", func(t *testing.T) {
		input := `module test
func f(opt: Option[(int, int)]) -> int = match opt {
  Some(p) =>
    let (a, b) = p
    a + b
  None => 0
}`
		parseOK(t, input)
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
