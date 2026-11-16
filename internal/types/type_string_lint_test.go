package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
)

// A type's String() is exact and unbounded: through a doubling alias it is
// gigabytes (#2363 review: a non-exhaustive match on such an alias was killed
// at 3 GB building its message). Messages format types with fmt (bounded by
// Format, type_format.go) or displayType. This test type-checks the package
// and fails on any call of String() on a Type outside the functions listed
// here, each of which needs the exact string: String() itself, keys, digests,
// equality, sorting and debug output.
var exactTypeStringUsers = map[string]string{
	// String() itself renders its children.
	"types.go:TList.String":             "String()",
	"types.go:TArray.String":            "String()",
	"types.go:TMap.String":              "String()",
	"types.go:TTuple.String":            "String()",
	"types.go:TApp.String":              "String()",
	"types.go:recordFieldStrings":       "String() of a record",
	"types.go:sortedFieldStrings":       "String() of a record",
	"types.go:TypeScheme.String":        "String()",
	"types.go:Constraint.String":        "String()",
	"types.go:Qualified.String":         "String()",
	"types_v2.go:Row.String":            "String()",
	"types_v2.go:TFunc2.String":         "String()",
	"types_v2.go:Scheme.String":         "String()",
	"types_v2.go:TRecord2.String":       "String()",
	"label_helpers.go:TLabelled.String": "String()",
	// The bounded renderers' exact branches.
	"type_format.go:formatType":            "within maxTypeTreeNodes",
	"type_display.go:displayType":          "within displayTypeNodes",
	"type_display.go:boundedPrinter.write": "leaf types",
	// Keys, sort orders and names.
	"instances.go:CanonicalDictParams":                "sort order",
	"instances.go:typeToName":                         "dictionary identifier",
	"scheme_canonical.go:CanonicalizeScheme":          "sort order",
	"scheme_constraints.go:schemeConstraintsFor":      "dedup key",
	"typechecker_operators.go:typesEqual":             "equality",
	"type_report.go:CoreTypeChecker.getProvenanceFor": "a type variable's name, as a key",
	"unification_records.go:Unifier.unifyRows":        "a row variable's name",
	// Debug output, off by default.
	"debug_sink.go:VerboseDebugSink.OnFreshTypeVar":                         "debug sink",
	"debug_sink.go:VerboseDebugSink.OnDefault":                              "debug sink",
	"typechecker_core.go:CoreTypeChecker.DumpInstantiations":                "debug dump",
	"typechecker_substitution.go:CoreTypeChecker.propagateTypeNameToCoreTI": "DEBUG_TYPENAME",
}

func TestFormatLint_TypeStringOnlyWhereExact(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	names := map[*ast.File]string{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
		names[f] = name
	}
	// Imports are stubbed: only this package's own types are resolved, which
	// is all the check needs (Type is declared here).
	info := &gotypes.Info{Types: map[ast.Expr]gotypes.TypeAndValue{}}
	conf := gotypes.Config{Importer: stubImporter{}, Error: func(error) {}}
	pkg, _ := conf.Check("github.com/sunholo-data/ailang/internal/types", fset, files, info)
	typeObj := pkg.Scope().Lookup("Type")
	if typeObj == nil {
		t.Fatal("Type not found")
	}
	iface, ok := typeObj.Type().Underlying().(*gotypes.Interface)
	if !ok {
		t.Fatal("Type is not an interface")
	}
	used := map[string]bool{}
	found := 0
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			key := names[f] + ":" + funcKey(fd)
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 0 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "String" {
					return true
				}
				tv, ok := info.Types[sel.X]
				if !ok || tv.Type == nil || !gotypes.Implements(tv.Type, iface) {
					return true
				}
				found++
				if _, ok := exactTypeStringUsers[key]; ok {
					used[key] = true
					return true
				}
				t.Errorf("%s: %s calls %s.String() on a type; format it with fmt (bounded) or displayType, "+
					"or list the function in exactTypeStringUsers if it needs the exact string",
					fset.Position(call.Pos()), key, exprText(sel.X))
				return true
			})
		}
	}
	if found == 0 {
		t.Fatal("no String() call on a type found: the type resolution is broken")
	}
	var stale []string
	for key := range exactTypeStringUsers {
		if !used[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale)
	for _, key := range stale {
		t.Errorf("exactTypeStringUsers lists %s, which no longer calls String() on a type: remove it", key)
	}
}

func funcKey(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	recv := fd.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	return exprText(recv) + "." + fd.Name.Name
}

// stubImporter gives every import an empty package, so the check resolves
// this package's declarations without compiling its dependencies.
type stubImporter struct{}

func (stubImporter) Import(p string) (*gotypes.Package, error) {
	pkg := gotypes.NewPackage(p, path.Base(p))
	pkg.MarkComplete()
	return pkg, nil
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
