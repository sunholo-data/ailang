package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type finding struct {
	File string
	Line int
}

func enumerate(root string, paths []string) ([]finding, error) {
	var out []finding
	for _, base := range paths {
		err := filepath.Walk(filepath.Join(root, base), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				id, ok := sel.X.(*ast.Ident)
				if !ok || id.Name != "exec" {
					return true
				}
				idx := 0
				if sel.Sel.Name == "CommandContext" {
					idx = 1
				} else if sel.Sel.Name != "Command" {
					return true
				}
				if len(call.Args) <= idx {
					return true
				}
				lit, ok := call.Args[idx].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				v, err := strconv.Unquote(lit.Value)
				if err != nil || v != "git" {
					return true
				}
				rel, _ := filepath.Rel(root, path)
				out = append(out, finding{filepath.ToSlash(rel), fset.Position(call.Pos()).Line})
				return true
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File == out[j].File {
			return out[i].Line < out[j].Line
		}
		return out[i].File < out[j].File
	})
	return out, nil
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"cmd", "internal"}
	}
	fs, err := enumerate(*root, paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	counts := map[string]int{}
	for _, f := range fs {
		counts[f.File]++
		fmt.Printf("SITE %s:%d\n", f.File, f.Line)
	}
	files := make([]string, 0, len(counts))
	for f := range counts {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		fmt.Printf("COUNT %s %d\n", f, counts[f])
	}
	fmt.Printf("TOTAL %d\n", len(fs))
}
