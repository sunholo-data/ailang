package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// M2_PARSER_TYPECHECK_INTERP: Parser desugars `"${expr}"` to a concat_String
// chain with auto-show() wrapping.
//
// Desugaring rules:
//   "pre${expr}post"   → concat_String(concat_String("pre", show(expr)), "post")
//   "${expr}"          → show(expr)             (no string parts)
//   "${s}"             → show(s)                (string-typed expr; show is identity for String)
//   "no interp"        → StringLit("no interp") (unchanged — lexer emits STRING)

func parseInterpExpr(t *testing.T, input string) ast.Expr {
	t.Helper()
	// Wrap input in a minimal module so the parser sees a complete program.
	src := "module test/interp\n\nexport func main() -> string {\n" + input + "\n}"
	l := lexer.New(src, "test.ail")
	p := New(l)
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			t.Logf("parser error: %v", e)
		}
		t.Fatalf("unexpected parser errors for %q", input)
	}
	if prog.File == nil || len(prog.File.Decls) == 0 {
		t.Fatalf("no decls parsed for %q", input)
	}
	fn, ok := prog.File.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("first decl is not a FuncDecl, got %T", prog.File.Decls[0])
	}
	return fn.Body
}

// stringLit asserts that e is a StringLit with the given value.
func assertStringLit(t *testing.T, e ast.Expr, want string) {
	t.Helper()
	lit, ok := e.(*ast.Literal)
	if !ok || lit.Kind != ast.StringLit {
		t.Fatalf("expected StringLit(%q), got %T", want, e)
	}
	if lit.Value != want {
		t.Fatalf("StringLit: expected %q, got %q", want, lit.Value)
	}
}

// showCall asserts that e is `show(arg)` and returns arg.
func assertShowCall(t *testing.T, e ast.Expr) ast.Expr {
	t.Helper()
	call, ok := e.(*ast.FuncCall)
	if !ok {
		t.Fatalf("expected FuncCall, got %T", e)
	}
	ident, ok := call.Func.(*ast.Identifier)
	if !ok || ident.Name != "show" {
		t.Fatalf("expected FuncCall with identifier 'show', got %v", call.Func)
	}
	if len(call.Args) != 1 {
		t.Fatalf("expected show() to have 1 arg, got %d", len(call.Args))
	}
	return call.Args[0]
}

// concatCall asserts e is `concat_String(left, right)` or the sugared `++` BinaryOp.
// Both forms are acceptable so long as the left/right operands match the expected
// desugar tree shape.
func assertConcatCall(t *testing.T, e ast.Expr) (left, right ast.Expr) {
	t.Helper()
	switch v := e.(type) {
	case *ast.FuncCall:
		ident, ok := v.Func.(*ast.Identifier)
		if !ok || ident.Name != "concat_String" {
			t.Fatalf("expected concat_String call, got FuncCall(%v)", v.Func)
		}
		if len(v.Args) != 2 {
			t.Fatalf("concat_String expects 2 args, got %d", len(v.Args))
		}
		return v.Args[0], v.Args[1]
	case *ast.BinaryOp:
		if v.Op != "++" {
			t.Fatalf("expected concat_String or ++, got BinaryOp(%q)", v.Op)
		}
		return v.Left, v.Right
	default:
		t.Fatalf("expected concat call or ++ BinaryOp, got %T", e)
	}
	return nil, nil
}

func TestInterp_PlainStringUnchanged(t *testing.T) {
	// Plain string (no ${...}) → single StringLit — lexer emits STRING, not STRING_PART.
	expr := parseInterpExpr(t, `"hello world"`)
	assertStringLit(t, expr, "hello world")
}

func TestInterp_SingleExprNoParts(t *testing.T) {
	// "${x}" → show(x)  (leading and trailing STRING_PART are empty, elided)
	expr := parseInterpExpr(t, `"${x}"`)
	arg := assertShowCall(t, expr)
	ident, ok := arg.(*ast.Identifier)
	if !ok || ident.Name != "x" {
		t.Fatalf("expected show(x), got show(%v)", arg)
	}
}

func TestInterp_PrefixAndExpr(t *testing.T) {
	// "hi ${name}" → concat_String("hi ", show(name))
	expr := parseInterpExpr(t, `"hi ${name}"`)
	left, right := assertConcatCall(t, expr)
	assertStringLit(t, left, "hi ")
	arg := assertShowCall(t, right)
	ident, ok := arg.(*ast.Identifier)
	if !ok || ident.Name != "name" {
		t.Fatalf("expected show(name), got show(%v)", arg)
	}
}

func TestInterp_ExprAndSuffix(t *testing.T) {
	// "${x}!" → concat_String(show(x), "!")
	expr := parseInterpExpr(t, `"${x}!"`)
	left, right := assertConcatCall(t, expr)
	// left should be show(x)
	arg := assertShowCall(t, left)
	ident, ok := arg.(*ast.Identifier)
	if !ok || ident.Name != "x" {
		t.Fatalf("expected show(x), got show(%v)", arg)
	}
	assertStringLit(t, right, "!")
}

func TestInterp_PrefixExprSuffix(t *testing.T) {
	// "Hello, ${name}!"
	// → concat_String(concat_String("Hello, ", show(name)), "!")
	expr := parseInterpExpr(t, `"Hello, ${name}!"`)
	outerL, outerR := assertConcatCall(t, expr)
	assertStringLit(t, outerR, "!")
	innerL, innerR := assertConcatCall(t, outerL)
	assertStringLit(t, innerL, "Hello, ")
	arg := assertShowCall(t, innerR)
	ident, ok := arg.(*ast.Identifier)
	if !ok || ident.Name != "name" {
		t.Fatalf("expected show(name), got show(%v)", arg)
	}
}

func TestInterp_MultipleExpressions(t *testing.T) {
	// "${a}-${b}" → concat_String(concat_String(show(a), "-"), show(b))
	expr := parseInterpExpr(t, `"${a}-${b}"`)
	outerL, outerR := assertConcatCall(t, expr)
	// outerR: show(b)
	arg := assertShowCall(t, outerR)
	if ident, ok := arg.(*ast.Identifier); !ok || ident.Name != "b" {
		t.Fatalf("expected show(b), got show(%v)", arg)
	}
	// outerL: concat_String(show(a), "-")
	innerL, innerR := assertConcatCall(t, outerL)
	assertStringLit(t, innerR, "-")
	argA := assertShowCall(t, innerL)
	if ident, ok := argA.(*ast.Identifier); !ok || ident.Name != "a" {
		t.Fatalf("expected show(a), got show(%v)", argA)
	}
}

func TestInterp_Arithmetic(t *testing.T) {
	// "count: ${x + 1}" → concat_String("count: ", show(x + 1))
	expr := parseInterpExpr(t, `"count: ${x + 1}"`)
	left, right := assertConcatCall(t, expr)
	assertStringLit(t, left, "count: ")
	arg := assertShowCall(t, right)
	bin, ok := arg.(*ast.BinaryOp)
	if !ok || bin.Op != "+" {
		t.Fatalf("expected show(x + 1), got show(%v)", arg)
	}
}

func TestInterp_EscapedDollarBrace(t *testing.T) {
	// "\${literal}" → single StringLit with literal "${literal}" (no interpolation)
	expr := parseInterpExpr(t, `"\${literal}"`)
	assertStringLit(t, expr, "${literal}")
}

func TestInterp_NestedFunctionCall(t *testing.T) {
	// "Got: ${show(x)}" → concat_String("Got: ", show(show(x)))
	// The inner show is the user's; the outer show is auto-inserted.
	// This verifies we don't special-case show() — we always wrap.
	expr := parseInterpExpr(t, `"Got: ${show(x)}"`)
	left, right := assertConcatCall(t, expr)
	assertStringLit(t, left, "Got: ")
	outerShowArg := assertShowCall(t, right)
	// outerShowArg should be show(x) — i.e. a user FuncCall
	innerShowArg := assertShowCall(t, outerShowArg)
	ident, ok := innerShowArg.(*ast.Identifier)
	if !ok || ident.Name != "x" {
		t.Fatalf("expected show(show(x)), inner arg = %v", innerShowArg)
	}
}
