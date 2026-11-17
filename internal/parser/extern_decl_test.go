package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// #2390: every `extern func` shape parses cleanly, with no EXT001-EXT003 shape
// errors, so the elaborator's EXT004 is the one error an author sees. The
// declaration is positioned at the `extern` keyword and the next declaration
// survives.
func TestExternFuncDeclarationsParse(t *testing.T) {
	cases := []struct {
		name, decl string
		pure       bool
		typeParams int
		params     int
		returnType bool
		effects    bool
	}{
		{"plain", "extern func hostAdd(x: int, y: int) -> int", false, 0, 2, true, false},
		{"underscore name (was EXT001)", "extern func _bad(x: int) -> int", false, 0, 1, true, false},
		{"type parameters (was EXT002)", "extern func ident[a](x: a) -> a", false, 1, 1, true, false},
		{"generic unit param", "extern func make[a]() -> a", false, 1, 1, true, false},
		{"no return type (was EXT003)", "extern func noRet(x: int)", false, 0, 1, false, false},
		{"unit param and effects", "extern func now() -> int ! {Clock}", false, 0, 1, true, true},
		{"spaced empty params", "extern func spaced(   ) -> int", false, 0, 1, true, false},
		{"extern pure func (was PAR_EXTERN_REQUIRES_FUNC)", "extern pure func p(x: int) -> int", true, 0, 1, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := New(lexer.New("module probe\n"+c.decl+"\nexport func next() -> int = 42\n", "probe.ail"))
			file := p.ParseFile()
			if errs := p.Errors(); len(errs) != 0 {
				t.Fatalf("parse errors: %v", errs)
			}
			if len(file.Funcs) != 2 || file.Funcs[1].Name != "next" {
				t.Fatalf("funcs: %+v", file.Funcs)
			}
			fn := file.Funcs[0]
			if !fn.IsExtern || fn.IsPure != c.pure || fn.Body != nil {
				t.Errorf("IsExtern=%v IsPure=%v Body=%v", fn.IsExtern, fn.IsPure, fn.Body)
			}
			if fn.Pos.Line != 2 || fn.Pos.Column != 1 {
				t.Errorf("position %d:%d, want 2:1 (the extern keyword)", fn.Pos.Line, fn.Pos.Column)
			}
			if len(fn.TypeParams) != c.typeParams || len(fn.Params) != c.params {
				t.Errorf("type params %d, params %d; want %d, %d", len(fn.TypeParams), len(fn.Params), c.typeParams, c.params)
			}
			if (fn.ReturnType != nil) != c.returnType || (fn.Effects != nil) != c.effects {
				t.Errorf("return type %v, effects %v", fn.ReturnType, fn.Effects)
			}
		})
	}
}

// `extern` followed by neither `func` nor `pure func` stays
// PAR_EXTERN_REQUIRES_FUNC.
func TestExternWithoutFuncIsAParseError(t *testing.T) {
	for _, src := range []string{"extern type T = int", "extern pure let x = 1"} {
		p := New(lexer.New("module probe\n"+src+"\n", "probe.ail"))
		p.ParseFile()
		errs := p.Errors()
		if len(errs) == 0 {
			t.Fatalf("%q: no parse error", src)
		}
		if pe, ok := errs[0].(*ParserError); !ok || pe.Code != "PAR_EXTERN_REQUIRES_FUNC" {
			t.Errorf("%q: first error %v, want PAR_EXTERN_REQUIRES_FUNC", src, errs[0])
		}
	}
}

// A malformed extern declaration is a parse error, not a panic or a silently
// dropped declaration: a missing name, or a missing parameter list.
func TestExternMalformedDeclarationIsAParseError(t *testing.T) {
	for _, src := range []string{"extern func 1() -> int", "extern func f -> int"} {
		p := New(lexer.New("module probe\n"+src+"\n", "probe.ail"))
		file := p.ParseFile()
		if len(p.Errors()) == 0 {
			t.Errorf("%q: no parse error", src)
		}
		for _, fn := range file.Funcs {
			if fn != nil && fn.IsExtern { // a failed declaration leaves a nil entry (pre-existing)
				t.Errorf("%q: a malformed extern produced a declaration %+v", src, fn)
			}
		}
	}
}
