package types

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"strings"
	"testing"
)

func TestIFCAnalysisLimitsFailClosed(t *testing.T) {
	t.Run("work budget", func(t *testing.T) {
		c := newIFCChecker(&ast.File{})
		c.analysisRemaining = 0
		errs := guardIFCAnalysis(func() []*TypeCheckError { c.analysisStep(); return nil })
		if len(errs) != 1 || errs[0].Kind != IFCAnalysisLimitError || !strings.Contains(errs[0].Message, "could not be verified") {
			t.Fatalf("limit must be an ordinary refusal: %v", errs)
		}
	})
	t.Run("nested written type", func(t *testing.T) {
		p := parser.New(lexer.New("module r/limit\npure func f(g: string -> string) -> int { 0 }", "limit.ail"))
		file := p.ParseFile()
		if len(p.Errors()) > 0 {
			t.Fatal(p.Errors())
		}
		var typ ast.Type = &ast.SimpleType{Name: "string"}
		for i := 0; i < ifcAnalysisDepth+2; i++ {
			typ = &ast.FuncType{Return: typ}
		}
		file.Funcs[0].Params[0].Type = typ
		errs := CheckModuleIFC(file)
		if len(errs) != 1 || errs[0].Kind != IFCAnalysisLimitError {
			t.Fatalf("type nesting limit must not accept silently or crash: %v", errs)
		}
	})
}

func TestIFCAnalysisDoesNotHideProgrammingPanic(t *testing.T) {
	defer func() {
		if p := recover(); p != "unrelated" {
			t.Fatalf("unexpected panic: %v", p)
		}
	}()
	guardIFCAnalysis(func() []*TypeCheckError { panic("unrelated") })
	t.Fatal("unrelated panic was hidden")
}
