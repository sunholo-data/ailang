package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// envConstants parses this package's non-test sources and returns every
// top-level `Env* = "LITERAL"` constant as name -> value.
func envConstants(t *testing.T) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	out := map[string]string{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.CONST {
					continue
				}
				for _, spec := range gd.Specs {
					vs := spec.(*ast.ValueSpec)
					for i, name := range vs.Names {
						if !strings.HasPrefix(name.Name, "Env") || i >= len(vs.Values) {
							continue
						}
						lit, ok := vs.Values[i].(*ast.BasicLit)
						if !ok || lit.Kind != token.STRING {
							continue
						}
						v, err := strconv.Unquote(lit.Value)
						if err != nil {
							t.Fatalf("%s: %v", name.Name, err)
						}
						out[name.Name] = v
					}
				}
			}
		}
	}
	return out
}

// TestEveryGetenvNameIsRegistered is the drift gate for the generated
// reference: every Env* constant in the package appears in Registry exactly
// once, every Registry entry names an Env* constant, and every entry is
// complete enough to render.
func TestEveryGetenvNameIsRegistered(t *testing.T) {
	consts := envConstants(t)
	// Control: the instrument must see the constants that predate this test.
	for _, must := range []string{"EnvCloudProject", "EnvStorage", "EnvStrict"} {
		if _, ok := consts[must]; !ok {
			t.Fatalf("instrument check failed: %s not found among Env* constants (%d parsed)", must, len(consts))
		}
	}
	if len(Registry) < 100 {
		t.Fatalf("instrument check failed: Registry has %d entries, expected the S4 M2 census (>=100)", len(Registry))
	}

	registered := map[string]int{}
	for _, v := range Registry {
		registered[v.Name]++
	}
	byValue := map[string]string{}
	for name, value := range consts {
		byValue[value] = name
		switch registered[value] {
		case 0:
			t.Errorf("%s (%s) is read by this package but missing from Registry — add a Var beside its getter", name, value)
		case 1:
		default:
			t.Errorf("%s appears %d times in Registry", value, registered[value])
		}
	}
	areas := map[string]bool{}
	for _, a := range Areas {
		areas[a] = true
	}
	for _, v := range Registry {
		if _, ok := byValue[v.Name]; !ok {
			t.Errorf("Registry entry %s has no Env* constant; getters must read the name from a constant", v.Name)
		}
		if !areas[v.Where] {
			t.Errorf("Registry entry %s: Where %q is not one of Areas", v.Name, v.Where)
		}
		if strings.TrimSpace(v.Doc) == "" {
			t.Errorf("Registry entry %s has no Doc", v.Name)
		}
	}
}

// TestNoLiteralGetenvInConfig pins the shape the migration relies on: inside
// this package os.Getenv / os.LookupEnv take an identifier (an Env* constant
// or a helper parameter), never a string literal, so the census above cannot
// be bypassed by a quoted name.
func TestNoLiteralGetenvInConfig(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
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
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "os" || (sel.Sel.Name != "Getenv" && sel.Sel.Name != "LookupEnv") {
				return true
			}
			seen++
			if len(call.Args) == 1 {
				if _, lit := call.Args[0].(*ast.BasicLit); lit {
					t.Errorf("%s: os.%s with a string literal; use an Env* constant", fset.Position(call.Pos()), sel.Sel.Name)
				}
			}
			return true
		})
	}
	if seen == 0 {
		t.Fatal("instrument check failed: no os.Getenv calls found in the package that reads every variable")
	}
}

// TestGetOrUsesRegistryDefault proves the default a getter returns is the
// one the reference page prints.
func TestGetOrUsesRegistryDefault(t *testing.T) {
	t.Setenv(EnvTraceMaxSpans, "")
	if got := TraceMaxSpans(); got != 500 {
		t.Errorf("TraceMaxSpans() = %d with the variable unset, want the registered 500", got)
	}
	t.Setenv(EnvTraceMaxSpans, "abc")
	if got := TraceMaxSpans(); got != 500 {
		t.Errorf("TraceMaxSpans() = %d with a malformed value, want 500", got)
	}
	t.Setenv(EnvTraceMaxSpans, "7")
	if got := TraceMaxSpans(); got != 7 {
		t.Errorf("TraceMaxSpans() = %d, want 7", got)
	}
	t.Setenv(EnvRelaxModules, "YES")
	if !RelaxModules() {
		t.Error("RelaxModules() false for YES")
	}
	t.Setenv(EnvRelaxModules, "0")
	if RelaxModules() {
		t.Error("RelaxModules() true for 0")
	}
	t.Setenv(EnvAnthropicRation, "")
	if !AnthropicRation() {
		t.Error("AnthropicRation() false when unset; unset means rationed")
	}
	t.Setenv(EnvClaudeCodeOAuthToken, "")
	if _, present := ClaudeCodeOAuthToken(); !present {
		t.Error("ClaudeCodeOAuthToken(): an empty-but-set token must report present (it bypasses the keychain)")
	}
	os.Unsetenv(EnvClaudeCodeOAuthToken)
	if _, present := ClaudeCodeOAuthToken(); present {
		t.Error("ClaudeCodeOAuthToken(): unset must report absent")
	}
	if _, ok := Lookup("NOT_A_REGISTERED_NAME"); ok {
		t.Error("Lookup of an unregistered name succeeded")
	}
	defer func() {
		if recover() == nil {
			t.Error("getOr on an unregistered name must panic")
		}
	}()
	_ = getOr("NOT_A_REGISTERED_NAME")
}
