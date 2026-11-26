package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// TST-TL-01: `ailang fmt` preserves `?` across inline let..in, multiline
// commented let..in chains, block statement lets, destructuring lets, and
// precedence-sensitive RHS expressions (`if`, `match`, `|>`), round-tripping
// idempotently to the identical AST.
func TestFormatTryLet_RoundTripAndIdempotence(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		mustHave []string
	}{
		{
			name: "block_statement_lets_and_patterns",
			src: `module m
export pure func load(s: string) -> Result[int, string] {
  let raw: string = step1(s)?
  let (a, b) = step2(raw)?
  let {x, y} = step3(a)?
  Ok(b + x + y)
}
`,
			mustHave: []string{
				"let raw: string = step1(s)?",
				"let (a, b) = step2(raw)?",
				"let { x, y } = step3(a)?",
			},
		},
		{
			name: "inline_let_in_chain",
			src: `module m
export pure func calc(s: string) -> Option[int] = let a = p(s)? in let (x, y) = q(a)? in Some(x + y)
`,
			mustHave: []string{
				"let a = p(s)? in let (x, y) = q(a)? in Some(x + y)",
			},
		},
		{
			name: "precedence_safe_parenthesization",
			src: `module m
export pure func tricky(c: bool, s: string) -> Result[int, string] {
  let a = (if c then Ok(1) else Ok(2))?
  let b = s |> parse?
  Ok(a + b)
}
`,
			mustHave: []string{
				"let a = (if c then Ok(1) else Ok(2))?",
				"let b = (s |> parse)?",
			},
		},
		{
			name: "commented_multiline_chain_and_block",
			src: `module m

export pure func chained(s: string) -> Result[int, string] =
  -- first step
  let a = parseA(s)? in  -- unwrap a
  let b = parseB(a)? in
  Ok(a + b)

export pure func blocked(s: string) -> Result[int, string] {
  let a = parseA(s)?  -- trailing comment in block
  Ok(a + 1)
}
`,
			mustHave: []string{
				"let a = parseA(s)? in  -- unwrap a",
				"let b = parseB(a)? in",
				"let a = parseA(s)?  -- trailing comment in block",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p1 := parser.New(lexer.New(tc.src, "test.ail"))
			prog1 := p1.Parse()
			if errs := p1.Errors(); len(errs) > 0 {
				t.Fatalf("initial parse failed: %v", errs[0])
			}

			out1Bytes, err := SourceWithComments(prog1, []byte(tc.src), Options{})
			if err != nil {
				t.Fatalf("SourceWithComments failed: %v", err)
			}
			out1 := string(out1Bytes)
			for _, want := range tc.mustHave {
				if !strings.Contains(out1, want) {
					t.Errorf("formatted output missing %q:\n%s", want, out1)
				}
			}

			// Re-parse formatted output and check AST equality + byte idempotence.
			p2 := parser.New(lexer.New(out1, "test.ail"))
			prog2 := p2.Parse()
			if errs := p2.Errors(); len(errs) > 0 {
				t.Fatalf("re-parse failed: %v\nformatted:\n%s", errs[0], out1)
			}
			if got, want := prog1.File.Funcs[0].Body.String(), prog2.File.Funcs[0].Body.String(); got != want {
				t.Errorf("AST drifted across format:\n  before: %s\n  after:  %s", got, want)
			}

			out2Bytes, err := SourceWithComments(prog2, []byte(out1), Options{})
			if err != nil {
				t.Fatalf("second SourceWithComments failed: %v", err)
			}
			if string(out2Bytes) != out1 {
				t.Errorf("formatter not idempotent:\n--- pass 1 ---\n%s\n--- pass 2 ---\n%s", out1, string(out2Bytes))
			}
		})
	}
}
