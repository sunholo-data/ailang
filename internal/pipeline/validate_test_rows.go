package pipeline

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// validateTestRows validates syntax only: inline rows bypass elaboration in
// the harness. Named bodies and properties have separate execution paths.
func validateTestRows(file *ast.File) error {
	if file == nil {
		return nil
	}
	for _, fn := range file.Funcs {
		for i, row := range fn.Tests {
			arms := append([]ast.Expr{}, row.Inputs...)
			arms = append(arms, row.Expected)
			for _, expr := range arms {
				if p := ast.RowExprSupported(expr); p != nil {
					pos := p.Pos
					if pos.Line == 0 {
						pos = row.Pos
					}
					return ailerrors.WrapReport(&ailerrors.Report{
						Schema: "ailang.error/v1", Code: p.Code, Phase: "typecheck",
						Message: fmt.Sprintf("%s row %d at %s: %s", fn.Name, i+1, pos, p.Message),
						Span:    &ast.Span{Start: pos, End: pos},
						Data:    map[string]any{"function": fn.Name, "row": i + 1},
						Fix:     &ailerrors.Fix{Suggestion: "write a value or use a named test block"},
					})
				}
			}
		}
	}
	return nil
}
