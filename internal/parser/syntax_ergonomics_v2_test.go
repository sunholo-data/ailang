package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

func TestE4_LambdaTupleDestructuring(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"simple_pair", `module m
export pure func f(xs: [(int, int)]) -> [int] = map(\(a, b). a + b, xs)`},
		{"mixed_ident_and_tuple", `module m
export pure func f(xs: [(string, int)]) -> int = foldl(\acc (k, v). acc + v, 0, xs)`},
		{"nested_with_wildcard", `module m
export pure func f(xs: [(int, (int, int))]) -> [int] = map(\(a, (b, _)). a + b, xs)`},
		{"two_tuple_params", `module m
export pure func f(p: (int, int), q: (int, int)) -> int = (\(a, b) (c, d). a + b + c + d)(p, q)`},
		{"tuple_with_record_inside", `module m
export pure func f(xs: [(int, {x: int, y: int})]) -> [int] = map(\(id, {x, ...}). id + x, xs)`},
		{"record_lambda_closed", `module m
export pure func f(xs: [{name: string, age: int}]) -> [int] = map(\{name, age}. age, xs)`},
		{"record_lambda_rest_and_acc", `module m
export pure func f(xs: [{x: int, y: int}]) -> int = foldl(\acc {x, ...}. acc + x, 0, xs)`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(lexer.New(tc.src, "test.ail"))
			file := p.ParseFile()
			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parse errors: %v", p.Errors())
			}
			if file == nil || len(file.Funcs) != 1 {
				t.Fatalf("expected 1 func, got %#v", file)
			}
		})
	}
}

func TestE4_LambdaTupleDestructuringRejectsRefutable(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"literal_in_tuple", `module m
export pure func f(xs: [(int, int)]) -> [int] = map(\(1, b). b, xs)`},
		{"constructor_in_tuple", `module m
export pure func f(xs: [(Option[int], int)]) -> [int] = map(\(Some(x), b). x + b, xs)`},
		{"single_paren_not_tuple", `module m
export pure func f(xs: [int]) -> [int] = map(\(x). x + 1, xs)`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(lexer.New(tc.src, "test.ail"))
			_ = p.ParseFile()
			if len(p.Errors()) == 0 {
				t.Fatalf("expected PAR_LAMBDA_PATTERN error, got none")
			}
			if !strings.Contains(p.Errors()[0].Error(), "PAR_LAMBDA_PATTERN") {
				t.Fatalf("expected PAR_LAMBDA_PATTERN, got: %v", p.Errors()[0])
			}
		})
	}
}

func TestE5_LetRecordDestructuring(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"closed_block", `module m
export pure func f(r: {name: string, age: int}) -> int {
  let {name, age} = r
  age
}`},
		{"renamed_and_rest", `module m
export pure func f(r: {host: string, port: int, tls: bool}) -> int {
  let {host, port: p, ...} = r
  p
}`},
		{"in_form_with_rest", `module m
export pure func f(r: {a: int, b: int}) -> int = let {a, ...} = r in a + 1`},
		{"nested_tuple_and_record", `module m
export pure func f(pair: (int, {x: int, y: int})) -> int {
  let (id, {x, ...}) = pair
  id + x
}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(lexer.New(tc.src, "test.ail"))
			file := p.ParseFile()
			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parse errors: %v", p.Errors())
			}
			if file == nil || len(file.Funcs) != 1 {
				t.Fatalf("expected 1 func, got %#v", file)
			}
		})
	}
}

func TestE5_LetRecordDestructuringRejectsRefutable(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"literal_field", `module m
export pure func f(r: {status: int}) -> int {
  let {status: 200} = r
  1
}`},
		{"empty_closed_record", `module m
export pure func f(r: {a: int}) -> int {
  let {} = r
  1
}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(lexer.New(tc.src, "test.ail"))
			_ = p.ParseFile()
			if len(p.Errors()) == 0 {
				t.Fatalf("expected PAR_LET_PATTERN error, got none")
			}
			if !strings.Contains(p.Errors()[0].Error(), "PAR_LET_PATTERN") {
				t.Fatalf("expected PAR_LET_PATTERN, got: %v", p.Errors()[0])
			}
		})
	}
}

func TestE6_TopLevelLetTrailingSemicolon(t *testing.T) {
	src := `module m
let MOD = 1000000007;
let (A, B) = (10, 20);
export pure func norm(x: int) -> int = (x + A + B) % MOD
`
	p := New(lexer.New(src, "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(file.Statements) != 2 {
		t.Fatalf("expected 2 top-level let statements, got %d", len(file.Statements))
	}
	if _, ok := file.Statements[0].(*ast.Let); !ok {
		t.Fatalf("expected *ast.Let, got %T", file.Statements[0])
	}
	if _, ok := file.Statements[1].(*ast.LetPattern); !ok {
		t.Fatalf("expected *ast.LetPattern, got %T", file.Statements[1])
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "norm" {
		t.Fatalf("expected func norm, got %#v", file.Funcs)
	}
}
