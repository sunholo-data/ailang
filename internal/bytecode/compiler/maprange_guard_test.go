package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompilerHasNoMapRanges prevents candidate selection from depending on
// Go's randomized map iteration order. It deliberately scans production files
// only; a range over a map must first materialize and sort its keys.
func TestCompilerHasNoMapRanges(t *testing.T) {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	mapNames := map[string]bool{}
	var files []*ast.File
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Clean(entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		files = append(files, file)
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Field:
				if _, ok := n.Type.(*ast.MapType); ok {
					for _, name := range n.Names {
						mapNames[name.Name] = true
					}
				}
			case *ast.ValueSpec:
				if _, ok := n.Type.(*ast.MapType); ok {
					for _, name := range n.Names {
						mapNames[name.Name] = true
					}
				}
			case *ast.AssignStmt:
				for i, rhs := range n.Rhs {
					if !isMapCreation(rhs) || i >= len(n.Lhs) {
						continue
					}
					if id, ok := n.Lhs[i].(*ast.Ident); ok {
						mapNames[id.Name] = true
					}
				}
			}
			return true
		})
	}

	// These ranges do not select an output value: alias registration writes
	// independent keys, isPinned asks an existential question, and copyMap is
	// a set clone. Keep the allowlist expression-specific and justified.
	allowed := map[string]string{
		"aliases": "independent alias keys",
		"names":   "existential register membership",
		"m":       "order-insensitive map copy",
	}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			rng, ok := n.(*ast.RangeStmt)
			if !ok {
				return true
			}
			name, isMap := rangedMapName(rng.X, mapNames)
			if !isMap || allowed[name] != "" {
				return true
			}
			pos := fset.Position(rng.For)
			t.Errorf("order-sensitive map range at %s; collect and sort keys first", pos)
			return true
		})
	}
}

func isMapCreation(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.CompositeLit:
		_, ok := expr.Type.(*ast.MapType)
		return ok
	case *ast.CallExpr:
		id, ok := expr.Fun.(*ast.Ident)
		return ok && id.Name == "make" && len(expr.Args) > 0 && func() bool {
			_, ok := expr.Args[0].(*ast.MapType)
			return ok
		}()
	}
	return false
}

func rangedMapName(expr ast.Expr, mapNames map[string]bool) (string, bool) {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name, mapNames[expr.Name]
	case *ast.SelectorExpr:
		return expr.Sel.Name, mapNames[expr.Sel.Name]
	}
	return "<literal>", isMapCreation(expr)
}
