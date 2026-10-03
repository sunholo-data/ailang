package golang

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sync"

	"github.com/sunholo-data/ailang/internal/mathx"
)

// mathxPrefix namespaces the portable math helpers inside generated programs.
// Generated user identifiers never start with it: top-level AILANG names are
// emitted as PascalCase/module-prefixed Go names or *_impl, so a lowercase
// "ailmathx_" prefix cannot collide (pinned by TestMathxHelperNamesAreReserved).
const mathxPrefix = "ailmathx_"

// #1465: compiled programs must compute std/math transcendentals with the same
// bits as the interpreter and VM, on every GOARCH. Host math.Exp & co. are not
// arch-stable, so the generator emits internal/mathx's own source (embedded)
// with its top-level names prefixed. One algorithm, three backends.

var mathxHelperSource = sync.OnceValues(buildMathxHelperSource)

// buildMathxHelperSource parses the embedded mathx files, prefixes every
// top-level identifier (and every reference to one), and prints the
// declarations without package clause, imports or comments.
func buildMathxHelperSource() (string, error) {
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range mathx.SourceFiles {
		src, err := mathx.Source.ReadFile(name)
		if err != nil {
			return "", fmt.Errorf("mathx source %s: %w", name, err)
		}
		f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			return "", fmt.Errorf("parse mathx source %s: %w", name, err)
		}
		files = append(files, f)
	}

	top := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				top[d.Name.Name] = true
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.ValueSpec:
						for _, n := range s.Names {
							top[n.Name] = true
						}
					case *ast.TypeSpec:
						top[s.Name.Name] = true
					}
				}
			}
		}
	}

	var out bytes.Buffer
	out.WriteString("// Portable std/math (#1465): internal/mathx, derived from the Go math\n")
	out.WriteString("// package. Copyright The Go Authors; BSD-style licence (see Go's LICENSE).\n\n")
	for _, f := range files {
		// Selector fields (math.Abs → "Abs") are not ours to rename.
		sel := map[*ast.Ident]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok {
				sel[s.Sel] = true
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && top[id.Name] && !sel[id] {
				id.Name = mathxPrefix + id.Name
			}
			return true
		})
		for _, d := range f.Decls {
			if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
				continue
			}
			switch d := d.(type) {
			case *ast.FuncDecl:
				d.Doc = nil
			case *ast.GenDecl:
				d.Doc = nil
			}
			if err := printer.Fprint(&out, fset, d); err != nil {
				return "", err
			}
			out.WriteString("\n\n")
		}
	}
	return out.String(), nil
}

// writeMathxHelpers emits the portable math helpers when generated code calls
// any std/math transcendental. usesMathx is sticky across Generate calls, so the
// shared multi-file runtime (generated after every module) sees it too.
func (g *Generator) writeMathxHelpers() {
	if !g.usesMathx {
		return
	}
	src, err := mathxHelperSource()
	if err != nil {
		g.errors = append(g.errors, err)
		return
	}
	g.buf.WriteString(src)
}
