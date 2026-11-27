package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// TST-RC1-10
func TestHoleNewlineInBlock(t *testing.T) {
	t.Run("named_hole_after_newline_in_func_body", func(t *testing.T) {
		src := `module test/hole_block
export pure func double(x: int) -> int {
  let y = x * 2
  ?finish
}
`
		l := lexer.New(src, "hole_block.ail")
		p := New(l)
		file := p.ParseFile()
		if len(p.Errors()) != 0 {
			for _, err := range p.Errors() {
				t.Errorf("  %v", err)
			}
			t.Fatalf("expected 0 parse errors, got %d", len(p.Errors()))
		}
		if len(file.Funcs) != 1 {
			t.Fatalf("expected 1 function, got %d", len(file.Funcs))
		}
		blk, ok := file.Funcs[0].Body.(*ast.Block)
		if !ok || len(blk.Exprs) != 2 {
			t.Fatalf("expected 2-expr Block body, got %T (%v)", file.Funcs[0].Body, file.Funcs[0].Body)
		}
		hole, ok := blk.Exprs[1].(*ast.Hole)
		if !ok || hole.Name != "finish" {
			t.Fatalf("expected *ast.Hole{Name: \"finish\"}, got %T (%v)", blk.Exprs[1], blk.Exprs[1])
		}
	})

	t.Run("anonymous_question_hole_after_newline_in_blocks", func(t *testing.T) {
		cases := []struct {
			name string
			src  string
		}{
			{
				name: "func_body",
				src: `module test/anon_hole
export pure func double(x: int) -> int {
  let y = x * 2
  ?
}
`,
			},
			{
				name: "match_arm_block",
				src: `module test/match_hole
export pure func checkVal(x: int) -> int =
  match x {
    0 => {
      let y = x + 1
      ?zero_case
    },
    _ => {
      let z = x * 2
      ?
    }
  }
`,
			},
			{
				name: "expr_block",
				src: `module test/expr_hole
pure func step(x: int) -> int = x + 1
export pure func compute(x: int) -> int = {
  step(x)
  ?next
}
`,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				l := lexer.New(tc.src, tc.name+".ail")
				p := New(l)
				_ = p.ParseFile()
				if len(p.Errors()) != 0 {
					for _, err := range p.Errors() {
						t.Errorf("  %v", err)
					}
					t.Fatalf("%s: expected 0 parse errors, got %d", tc.name, len(p.Errors()))
				}
			})
		}
	})

	t.Run("same_line_try_preserved", func(t *testing.T) {
		src := `module test/same_line_try
export pure func step(x: int) -> Result[int, string] {
  let y = parseVal(x)?
  Ok(y + 1)
}
`
		l := lexer.New(src, "same_line_try.ail")
		p := New(l)
		file := p.ParseFile()
		if len(p.Errors()) != 0 {
			for _, err := range p.Errors() {
				t.Errorf("  %v", err)
			}
			t.Fatalf("expected 0 parse errors, got %d", len(p.Errors()))
		}
		blk, ok := file.Funcs[0].Body.(*ast.Block)
		if !ok || len(blk.Exprs) != 2 {
			t.Fatalf("expected 2-expr Block, got %T", file.Funcs[0].Body)
		}
		letStmt, ok := blk.Exprs[0].(*ast.Let)
		if !ok || !letStmt.IsTry {
			t.Fatalf("expected same-line ? to parse as try-let (IsTry=true), got %T (IsTry=%v)", blk.Exprs[0], letStmt != nil && letStmt.IsTry)
		}
	})

	t.Run("regression_parser_block_trailing_record", func(t *testing.T) {
		fixture := filepath.Join("..", "..", "examples", "runnable", "parser_block_trailing_record.ail")
		raw, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read %s: %v", filepath.ToSlash(fixture), err)
		}
		l := lexer.New(string(raw), filepath.ToSlash(fixture))
		p := New(l)
		_ = p.ParseFile()
		if len(p.Errors()) != 0 {
			for _, perr := range p.Errors() {
				t.Errorf("  %v", perr)
			}
			t.Fatalf("regression fixture %s failed with %d parse errors", filepath.ToSlash(fixture), len(p.Errors()))
		}
	})
}

// TST-RC1-14
func TestBlockLetHintsNoLetIn(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		wantCode string
	}{
		{
			name: "const_inside_func_block",
			src: `module test/block_const
export pure func f(x: int) -> int {
  const y = x + 1
  y
}
`,
			wantCode: "PAR014",
		},
		{
			name: "bare_assignment_inside_func_block",
			src: `module test/block_assign
export pure func f(x: int) -> int {
  y = x + 1
  y
}
`,
			wantCode: "PAR015",
		},
		{
			name: "let_rec_inside_func_block",
			src: `module test/block_letrec
export pure func f(x: int) -> int {
  let rec go = \n. n
  go(x)
}
`,
			wantCode: "PAR018",
		},
		{
			name: "const_inside_match_arm_block",
			src: `module test/match_block_const
export pure func f(x: int) -> int =
  match x {
    _ => {
      const y = x + 1;
      y
    }
  }
`,
			wantCode: "PAR014",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.src, tc.name+".ail")
			p := New(l)
			_ = p.ParseFile()
			if len(p.Errors()) == 0 {
				t.Fatalf("%s: expected parse error %s, got none", tc.name, tc.wantCode)
			}
			var matched *ParserError
			for _, err := range p.Errors() {
				if pe, ok := err.(*ParserError); ok && pe.Code == tc.wantCode {
					matched = pe
					break
				}
			}
			if matched == nil {
				t.Fatalf("%s: expected error code %s, got errors: %v", tc.name, tc.wantCode, p.Errors())
			}
			rendered := matched.Error()
			for _, forbidden := range []string{"= value in", "= ... in", "let ... in", "`in`"} {
				if strings.Contains(rendered, forbidden) {
					t.Errorf("%s: block error must NOT suggest %q:\n%s", tc.name, forbidden, rendered)
				}
			}
			if !strings.Contains(rendered, ";") && !strings.Contains(rendered, "newline") {
				t.Errorf("%s: block error should suggest ';' or newline separator:\n%s", tc.name, rendered)
			}
		})
	}
}
