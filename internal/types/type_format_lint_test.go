package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// Diagnostics format types through fmt, which renders a type bounded
// (type_format.go). An explicit .String() inside fmt.Errorf bypasses that and
// writes the whole tree (#2363 review: field access through a doubling alias
// reached 3 GB). This test fails when a fmt.Errorf in this package calls
// .String() on anything but the receivers listed here, which are not types.
func TestFormatLint_ErrorfDoesNotStringifyTypes(t *testing.T) {
	notTypes := map[string]bool{ // receiver expression → why it is safe
		"msg":       true, // strings.Builder
		"row1.Tail": true, // *RowVar: a name
		"row2.Tail": true,
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isFmtErrorf(call.Fun) {
				return true
			}
			for _, arg := range call.Args {
				ast.Inspect(arg, func(n ast.Node) bool {
					inner, ok := n.(*ast.CallExpr)
					if !ok || len(inner.Args) != 0 {
						return true
					}
					sel, ok := inner.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "String" {
						return true
					}
					if recv := exprText(sel.X); !notTypes[recv] {
						t.Errorf("%s: fmt.Errorf formats %s.String(); pass the type itself (bounded by Format) or displayType(...)",
							fset.Position(inner.Pos()), recv)
					}
					return true
				})
			}
			return true
		})
	}
}

func isFmtErrorf(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "fmt" && sel.Sel.Name == "Errorf"
}

func exprText(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return exprText(x.X) + "." + x.Sel.Name
	}
	return "?"
}
