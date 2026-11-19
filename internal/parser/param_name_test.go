package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// M-V1-VERIFICATION-HONESTY M4 (#2156, D7). parseParams built an ast.Param for
// WHATEVER token it saw and filled Name/Type only for an identifier, so any
// other token became a nameless, untyped parameter: on v0.47 `func f(5)`
// checked clean and `f("hello")` returned 1. A parameter is now `name` or
// `name: Type`; the one other spelling, `()`, is the ignored unit parameter.

func parseParamsOf(t *testing.T, src string) (*ast.File, []error) {
	t.Helper()
	p := New(lexer.New(src, "p.ail"))
	prog := p.Parse()
	if prog == nil {
		return nil, p.Errors()
	}
	return prog.File, p.Errors()
}

// TestParams_UnitParamIsUnderscoreUnit: `func f(())` is the node `func f()`
// and `func f(_: ())` produce. Red mutation: drop the UNIT case — f(())
// becomes PAR_PARAM_NAME.
func TestParams_UnitParamIsUnderscoreUnit(t *testing.T) {
	for _, src := range []string{
		"module m\nfunc f(()) -> int = 1\n",
		"module m\nfunc f(_: ()) -> int = 1\n",
		"module m\nfunc f() -> int = 1\n",
	} {
		file, errs := parseParamsOf(t, src)
		if len(errs) > 0 {
			t.Fatalf("%q: unexpected errors %v", src, errs)
		}
		ps := file.Funcs[0].Params
		if len(ps) != 1 || ps[0].Name != "_" || ps[0].Type == nil || ps[0].Type.String() != "()" {
			t.Fatalf("%q: params = %+v, want exactly `_: ()`", src, ps)
		}
	}
}

// TestParams_UnitParamAmongOthers: `()` may sit beside named parameters.
func TestParams_UnitParamAmongOthers(t *testing.T) {
	file, errs := parseParamsOf(t, "module m\nfunc f(x: int, ()) -> int = x\n")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors %v", errs)
	}
	ps := file.Funcs[0].Params
	if len(ps) != 2 || ps[0].Name != "x" || ps[1].Name != "_" {
		t.Fatalf("params = %+v", ps)
	}
}

// TestParams_NonNameIsParParamName: every other token is an error naming the
// token, in function declarations and in func literals alike, and the parser
// recovers at the next parameter. Red mutation: restore the nameless-param
// path (append param for any token) — every row stops erroring.
func TestParams_NonNameIsParParamName(t *testing.T) {
	for _, tc := range []struct{ src, token string }{
		{"module m\nfunc f(5) -> int = 1\n", "`5`"},
		{"module m\nfunc f(\"x\") -> int = 1\n", "`\"x\"`"},
		{"module m\nfunc f((a, b): (int, int)) -> int = a + b\n", "`(`"},
		{"module m\nfunc f(x: int, 5) -> int = x\n", "`5`"},
		{"module m\nlet g = func(7) -> int { 1 }\n", "`7`"},
	} {
		_, errs := parseParamsOf(t, tc.src)
		if len(errs) == 0 {
			t.Fatalf("%q: parsed without error; a non-name parameter must be rejected", tc.src)
		}
		found := false
		for _, e := range errs {
			if strings.Contains(e.Error(), "PAR_PARAM_NAME") && strings.Contains(e.Error(), "parameter must be a name, got "+tc.token) {
				found = true
			}
		}
		if !found {
			t.Fatalf("%q: no PAR_PARAM_NAME naming %s in %v", tc.src, tc.token, errs)
		}
	}
}

// TestParams_RecoveryKeepsLaterParams: the bad parameter is dropped and the
// next one still parses (one diagnostic, no cascade).
func TestParams_RecoveryKeepsLaterParams(t *testing.T) {
	file, errs := parseParamsOf(t, "module m\nfunc f(5, y: int) -> int = y\n")
	if len(errs) != 1 {
		t.Fatalf("want exactly one error, got %v", errs)
	}
	if file == nil || len(file.Funcs) != 1 || len(file.Funcs[0].Params) != 1 || file.Funcs[0].Params[0].Name != "y" {
		t.Fatalf("recovery lost the following parameter: %+v", file)
	}
}
