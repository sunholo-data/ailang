package golang

import (
	"bytes"
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// emitted_quote_runtime.txt is the quoting rule of internal/strutil
// (quote_rule.go, nfc.go, nfc_tables.go) with every top-level name prefixed
// by showq_, so compiled programs quote strings exactly as the interpreters
// do. One definition: change the rule in strutil, then
//
//	go test ./internal/gen/golang -run TestEmittedQuoteRuntimeUpToDate -update-quote
var updateQuote = flag.Bool("update-quote", false, "regenerate emitted_quote_runtime.txt from internal/strutil")

const emittedQuotePrefix = "showq_"

var emittedQuoteSources = []string{"quote_rule.go", "nfc.go", "nfc_tables.go"}

func generateEmittedQuoteRuntime(t *testing.T) []byte {
	t.Helper()
	fset := token.NewFileSet()
	var files []*ast.File
	names := map[string]bool{}
	for _, name := range emittedQuoteSources {
		f, err := parser.ParseFile(fset, filepath.Join("..", "..", "strutil", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			switch imp.Path.Value {
			case `"sort"`, `"strconv"`, `"strings"`, `"unicode/utf8"`:
			default:
				t.Fatalf("%s imports %s: the emitted runtime may only use sort, strconv, strings and unicode/utf8", name, imp.Path.Value)
			}
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				names[d.Name.Name] = true
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						names[s.Name.Name] = true
					case *ast.ValueSpec:
						for _, n := range s.Names {
							names[n.Name] = true
						}
					}
				}
			}
		}
		files = append(files, f)
	}
	var out bytes.Buffer
	out.WriteString("// Quoting rule copied from internal/strutil by emitted_quote_gen_test.go. DO NOT EDIT.\n\n")
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && names[id.Name] {
				id.Name = emittedQuotePrefix + id.Name
			}
			return true
		})
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
				continue
			}
			if err := printer.Fprint(&out, fset, d); err != nil {
				t.Fatal(err)
			}
			out.WriteString("\n\n")
		}
	}
	return out.Bytes()
}

func TestEmittedQuoteRuntimeUpToDate(t *testing.T) {
	want := generateEmittedQuoteRuntime(t)
	if *updateQuote {
		if err := os.WriteFile("emitted_quote_runtime.txt", want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if !bytes.Equal([]byte(emittedQuoteRuntime), want) {
		t.Fatal("emitted_quote_runtime.txt is out of date with internal/strutil's quoting rule: regenerate with -update-quote")
	}
}
