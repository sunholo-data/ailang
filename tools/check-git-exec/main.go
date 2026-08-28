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

// execNames returns the set of identifiers that refer to package os/exec in f.
//
// Resolving through the import declarations rather than matching the literal
// identifier "exec" is what makes an aliased import (`import exe "os/exec"`)
// visible. A name-only check is evaded by one keystroke, and the evasion is
// gofmt-canonical, so nothing else in the toolchain would surface it.
func execNames(f *ast.File) map[string]bool {
	names := map[string]bool{}
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil || p != "os/exec" {
			continue
		}
		switch {
		case imp.Name == nil:
			names["exec"] = true
		case imp.Name.Name == "." || imp.Name.Name == "_":
			// A dot-import puts Command/LookPath in file scope unqualified and a
			// blank import cannot call anything. Neither is matchable by the
			// selector walk below; both are declared residuals, not silent passes.
		default:
			names[imp.Name.Name] = true
		}
	}
	return names
}

// gitLiteralAt reports whether call's argument at idx is the string literal "git".
func gitLiteralAt(call *ast.CallExpr, idx int) bool {
	if len(call.Args) <= idx {
		return false
	}
	lit, ok := call.Args[idx].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	v, err := strconv.Unquote(lit.Value)
	return err == nil && v == "git"
}

// enumerate walks every non-test .go file under root/paths and returns, per
// AST rather than per line, the bare-name git exec sites and the LookPath("git")
// sites.
//
// Both are AST-based for the same reason (design HID-6): grep is line-anchored,
// and gofmt wraps an argument list as soon as it grows or a comment anchors it,
// so a line-oriented matcher cannot see the shape the formatter itself produces.
// The LookPath arm was line-oriented until the iteration-298 evaluator planted a
// gofmt-canonical multi-line duplicate resolver and the gate reported OK.
func enumerate(root string, paths []string) (cmds []finding, lookPaths []finding, err error) {
	for _, base := range paths {
		walkErr := filepath.Walk(filepath.Join(root, base), func(path string, info os.FileInfo, err error) error {
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
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			names := execNames(f)
			if len(names) == 0 {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
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
				if !ok || !names[id.Name] {
					return true
				}
				line := fset.Position(call.Pos()).Line
				switch sel.Sel.Name {
				case "Command":
					if gitLiteralAt(call, 0) {
						cmds = append(cmds, finding{rel, line})
					}
				case "CommandContext":
					if gitLiteralAt(call, 1) {
						cmds = append(cmds, finding{rel, line})
					}
				case "LookPath":
					if gitLiteralAt(call, 0) {
						lookPaths = append(lookPaths, finding{rel, line})
					}
				}
				return true
			})
			return nil
		})
		if walkErr != nil {
			return nil, nil, walkErr
		}
	}
	sortFindings(cmds)
	sortFindings(lookPaths)
	return cmds, lookPaths, nil
}

func sortFindings(fs []finding) {
	sort.Slice(fs, func(i, j int) bool {
		if fs[i].File == fs[j].File {
			return fs[i].Line < fs[j].Line
		}
		return fs[i].File < fs[j].File
	})
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"cmd", "internal"}
	}
	cmds, lookPaths, err := enumerate(*root, paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	counts := map[string]int{}
	for _, f := range cmds {
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
	fmt.Printf("TOTAL %d\n", len(cmds))
	for _, f := range lookPaths {
		fmt.Printf("LOOKPATH %s:%d\n", f.File, f.Line)
	}
	fmt.Printf("LOOKPATH_TOTAL %d\n", len(lookPaths))
}
