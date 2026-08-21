package builtins

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	builtinsSourceDir = "."
	stdlibSourceDir   = "../../std"
)

var listDelegationExemptions = map[string]string{
	"_list_any":       "DELEGATION-CANDIDATE: any is recursive; see the m-stdlib-list-delegation-sweep queue row",
	"_list_contains":  "NOT-NEEDED: std/list exposes member with explicit equality constraints instead of this legacy contains helper",
	"_list_drop":      "DELEGATION-CANDIDATE: drop is recursive; see the m-stdlib-list-delegation-sweep queue row",
	"_list_extract":   "NOT-NEEDED: std/list has no public extract operation to delegate",
	"_list_filterE":   "DELEGATION-CANDIDATE: filterE is recursive; see the m-stdlib-list-delegation-sweep queue row",
	"_list_findIndex": "DELEGATION-CANDIDATE: findIndex is recursive; see the m-stdlib-list-delegation-sweep queue row",
	"_list_flatMap":   "DELEGATION-CANDIDATE: flatMap is recursive and repeatedly concatenates; see the m-stdlib-list-delegation-sweep queue row",
	"_list_flatMapE":  "DELEGATION-CANDIDATE: flatMapE is recursive and repeatedly concatenates; see the m-stdlib-list-delegation-sweep queue row",
	"_list_foldlE":    "DELEGATION-CANDIDATE: foldlE is recursive; see the m-stdlib-list-delegation-sweep queue row",
	"_list_foldr":     "DELEGATION-CANDIDATE: foldr is recursive; see the m-stdlib-list-delegation-sweep queue row",
	"_list_forEachE":  "DELEGATION-CANDIDATE: forEachE is recursive; see the m-stdlib-list-delegation-sweep queue row",
	"_list_head":      "NOT-NEEDED: head is already O(1) through list pattern matching",
	"_list_last":      "NOT-NEEDED: last already routes through _list_length and _list_nth",
	"_list_mapE":      "DELEGATION-CANDIDATE: mapE is recursive and appends each result; see the m-stdlib-list-delegation-sweep queue row",
	"_list_sortBy":    "DELEGATION-CANDIDATE: sortBy is recursive and depends on recursive take and drop; see the m-stdlib-list-delegation-sweep queue row",
	"_list_tail":      "NOT-NEEDED: tail is already O(1) through list pattern matching",
	"_list_take":      "DELEGATION-CANDIDATE: take is recursive and repeatedly appends; see the m-stdlib-list-delegation-sweep queue row",
	"_list_zip":       "DELEGATION-CANDIDATE: zip is recursive and repeatedly appends; see the m-stdlib-list-delegation-sweep queue row",
}

// registeredListBuiltins parses registration source rather than hardcoding names.
// Registrations currently use three shapes: Name: fields, registerIfMissing
// calls, and table rows whose first field is the builtin name.
func registeredListBuiltins(t *testing.T) map[string]struct{} {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(builtinsSourceDir, "*.go"))
	if err != nil {
		t.Fatalf("instrument failure: enumerate builtin sources: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("instrument failure: no internal/builtins/*.go files found")
	}

	names := make(map[string]struct{})
	files := token.NewFileSet()
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(files, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.KeyValueExpr:
				key, ok := n.Key.(*ast.Ident)
				if ok && key.Name == "Name" {
					addListBuiltinLiteral(names, n.Value)
				}
			case *ast.CallExpr:
				fn, ok := n.Fun.(*ast.Ident)
				if ok && fn.Name == "registerIfMissing" && len(n.Args) > 0 {
					addListBuiltinLiteral(names, n.Args[0])
				}
			case *ast.CompositeLit:
				if len(n.Elts) > 0 {
					addListBuiltinLiteral(names, n.Elts[0])
				}
			}
			return true
		})
	}
	return names
}

func addListBuiltinLiteral(names map[string]struct{}, expr ast.Expr) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}
	name, err := strconv.Unquote(lit.Value)
	if err == nil && strings.HasPrefix(name, "_list_") {
		names[name] = struct{}{}
	}
}

func stdlibListCallSites(t *testing.T, names map[string]struct{}) map[string]int {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(stdlibSourceDir, "*.ail"))
	if err != nil {
		t.Fatalf("instrument failure: enumerate stdlib sources: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("instrument failure: no std/*.ail files found")
	}

	counts := make(map[string]int, len(names))
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for name := range names {
			counts[name] += len(regexp.MustCompile(regexp.QuoteMeta(name)+`\(`).FindAll(body, -1))
		}
	}
	return counts
}

func TestEveryListBuiltinIsDelegatedOrExplained(t *testing.T) {
	names := registeredListBuiltins(t)
	if len(names) < 25 {
		t.Fatalf("instrument failure: found only %d registered _list_* builtins; want at least 25", len(names))
	}

	counts := stdlibListCallSites(t, names)
	totalCalls := 0
	for _, count := range counts {
		totalCalls += count
	}
	if totalCalls == 0 {
		t.Fatal("instrument failure: std/ call-site scan found zero _list_* calls")
	}

	for name := range listDelegationExemptions {
		if _, registered := names[name]; !registered {
			t.Errorf("stale exemption %s: name is not a registered _list_* builtin", name)
		}
	}

	var unexplained []string
	for name := range names {
		if counts[name] == 0 {
			if reason := listDelegationExemptions[name]; reason == "" {
				unexplained = append(unexplained, name)
			}
		}
	}
	sort.Strings(unexplained)
	for _, name := range unexplained {
		t.Errorf("%s has zero exact call sites in std/*.ail and no delegation exemption", name)
	}

	if t.Failed() {
		t.Logf("scan summary: %d registered _list_* builtins, %d exact std/ call sites", len(names), totalCalls)
	} else {
		t.Logf("scan summary: %d registered _list_* builtins, %d exact std/ call sites", len(names), totalCalls)
	}
}
