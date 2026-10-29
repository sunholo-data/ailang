package types

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"strings"
	"testing"
)

func TestIFCExportAnalysisLimitIsDiagnostic(t *testing.T) {
	var typ ast.Type = &ast.SimpleType{Name: "string"}
	for i := 0; i < ifcAnalysisDepth+2; i++ {
		typ = &ast.FuncType{Return: typ}
	}
	file := &ast.File{Funcs: []*ast.FuncDecl{{Name: "factory", Body: &ast.Lambda{Params: []*ast.Param{{Name: "g", Type: typ}}, Body: &ast.Identifier{Name: "g"}}}}}
	summaries, err := AnalyzeModuleIFC(file, nil)
	if err == nil || summaries != nil || !strings.Contains(err.Error(), "information-flow analysis limit exceeded") {
		t.Fatalf("analysis must fail closed with an ordinary diagnostic: summaries=%v err=%v", summaries, err)
	}
}

func TestIFCExportSummaryRejectsMissingImportFacts(t *testing.T) {
	summaries, err := AnalyzeModuleIFC(&ast.File{}, map[string]*ImportedIFCSig{"broken": nil})
	if err == nil || summaries != nil {
		t.Fatal("missing import metadata treated clean")
	}
}
