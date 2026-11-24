package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

func TestNewlineBlockDelimiterStarters(t *testing.T) {
	cases := []struct {
		name      string
		src       string
		wantExprs int
	}{
		{
			name: "block_ending_tuple_after_let",
			src: `module test
pure func pair(n: int) -> (int, int) {
  let a = n + 1
  let b = n * 2
  (a, b)
}`,
			wantExprs: 3,
		},
		{
			name: "block_ending_record_update_after_let",
			src: `module test
pure func bump(r: {a: int}) -> {a: int} {
  let x = 1
  {r | a: x}
}`,
			wantExprs: 2,
		},
		{
			name: "block_ending_record_literal_after_let",
			src: `module test
pure func mk(x: int) -> {a: int, b: int} {
  let y = x + 1
  {a: x, b: y}
}`,
			wantExprs: 2,
		},
		{
			name: "block_ending_list_after_let",
			src: `module test
pure func two(x: int) -> [int] {
  let y = x + 1
  [x, y]
}`,
			wantExprs: 2,
		},
		{
			name: "block_ending_unit_after_call",
			src: `module test
func log(s: string) -> () ! {IO} {
  println(s)
  ()
}`,
			wantExprs: 2,
		},
		{
			name: "block_ending_parenthesized_expr",
			src: `module test
pure func calc(a: int) -> int {
  let b = a + 1
  (b + 2) * 3
}`,
			wantExprs: 2,
		},
		{
			name: "same_line_curried_call_still_works",
			src: `module test
pure func f(x: int) -> int {
  let add = \a. \b. a + b
  add(x)(1)
}`,
			wantExprs: 2,
		},
		{
			name: "multiline_operator_continuation_not_split",
			src: `module test
pure func f(x: int) -> int {
  let y = x +
    (1 + 2)
  y
}`,
			wantExprs: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(lexer.New(tc.src, "test.ail"))
			prog := p.Parse()
			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parse errors: %v", p.Errors())
			}
			if prog == nil || prog.File == nil || len(prog.File.Funcs) != 1 {
				t.Fatalf("expected 1 func decl")
			}
			blk, ok := prog.File.Funcs[0].Body.(*ast.Block)
			if !ok {
				t.Fatalf("expected *ast.Block body, got %T", prog.File.Funcs[0].Body)
			}
			if len(blk.Exprs) != tc.wantExprs {
				t.Errorf("got %d block exprs, want %d", len(blk.Exprs), tc.wantExprs)
			}
		})
	}
}
