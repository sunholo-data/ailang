package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

// flagSite is one bool-flag registration found in this package's source.
type flagSite struct {
	where        string // file:line
	defaultValue string // the literal as written: "false", "true", or "?" when not a literal
}

func (s flagSite) String() string { return s.where }

// flagRegistrations parses THIS package's non-test sources and returns every
// place a bool flag called name is registered, via either
// `fs.Bool(name, def, usage)` or `fs.BoolVar(&v, name, def, usage)`.
//
// It reads the AST rather than grepping because the question is "what default
// did this site register", and a regex over `.Bool("dry-run"` cannot answer it
// — the default is the NEXT argument, which a grep line may not even contain.
// go test runs with the package directory as its working directory, so "." is
// the package under test.
func flagRegistrations(t *testing.T, name string) []flagSite {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing package sources: %v", err)
	}

	var sites []flagSite
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				// Bool(name, def, usage) | BoolVar(&v, name, def, usage)
				var nameIdx int
				switch sel.Sel.Name {
				case "Bool":
					nameIdx = 0
				case "BoolVar":
					nameIdx = 1
				default:
					return true
				}
				if len(call.Args) < nameIdx+2 {
					return true
				}
				got, ok := stringLit(call.Args[nameIdx])
				if !ok || got != name {
					return true
				}

				def := "?"
				if id, ok := call.Args[nameIdx+1].(*ast.Ident); ok {
					def = id.Name
				}
				pos := fset.Position(call.Pos())
				sites = append(sites, flagSite{
					where:        fmt.Sprintf("%s:%d", pos.Filename, pos.Line),
					defaultValue: def,
				})
				return true
			})
		}
	}
	return sites
}

// dryRunRegistrations is the --dry-run slice of the audit.
func dryRunRegistrations(t *testing.T) []flagSite {
	t.Helper()
	return flagRegistrations(t, "dry-run")
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// TestFlagScannerSeesAKnownPositive is the scanner's own control. Every
// assertion built on flagRegistrations reads "we found nothing" as a pass, so
// a scanner that silently returned nothing would make all of them vacuous.
func TestFlagScannerSeesAKnownPositive(t *testing.T) {
	// --json is registered in output_flags.go through the two helpers.
	if got := flagRegistrations(t, flagJSON); len(got) < 2 {
		t.Fatalf("scanner found %d --json registrations, want >= 2 (output_flags.go registers it twice)", len(got))
	}
	// A name nothing registers must come back empty, or the scanner matches
	// everything and the control above proves nothing.
	if got := flagRegistrations(t, "definitely-not-a-flag-name"); len(got) != 0 {
		t.Fatalf("scanner matched a non-existent flag at %v", got)
	}
}
