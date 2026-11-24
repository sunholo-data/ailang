package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func formatWithCommentsForTest(t *testing.T, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src, "test.ail"))
	prog := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse failed: %v", p.Errors())
	}
	out, err := SourceWithComments(prog, []byte(src), Options{Indent: "  "})
	if err != nil {
		t.Fatalf("SourceWithComments failed: %v", err)
	}
	return string(out)
}

func TestBodyBracedSingleExprComments(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "func_decl_leading_comment",
			src: `module test/fmt_body

func handle(x: string) -> string {
  -- placeholder comment
  "hello ${x}"
}
`,
			want: []string{"-- placeholder comment", `"hello ${x}"`},
		},
		{
			name: "func_decl_trailing_comment",
			src: `module test/fmt_body

import std/fs (readFile)

func bad() -> string ! {IO} {
  readFile("data.txt") -- trailing comment
}
`,
			want: []string{"-- trailing comment", `readFile("data.txt")`},
		},
		{
			name: "func_decl_leading_and_closing_comments",
			src: `module test/fmt_body

func f(x: int) -> int {
  -- before expr
  x + 1
  -- after expr
}
`,
			want: []string{"-- before expr", "x + 1", "-- after expr"},
		},
		{
			name: "func_lit_leading_and_trailing_comment",
			src: `module test/fmt_body

export func main() -> int {
  let g = func(x: int) -> int {
    -- inside func lit
    x * 2 -- double it
  }
  g(21)
}
`,
			want: []string{"-- inside func lit", "-- double it"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s1 := formatWithCommentsForTest(t, tc.src)
			for _, w := range tc.want {
				if !strings.Contains(s1, w) {
					t.Errorf("formatted output missing %q:\n%s", w, s1)
				}
			}
			// Idempotency: second format must be byte-identical.
			s2 := formatWithCommentsForTest(t, s1)
			if s2 != s1 {
				t.Errorf("format not idempotent:\nfirst:\n%s\nsecond:\n%s", s1, s2)
			}
		})
	}
}
