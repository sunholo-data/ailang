package pkg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestPkgAuthorityBoundary is the bypass detector of #1720 (AC3, D6). It
// parses every non-test file of the package surface — internal/pkg plus the
// cmd/ailang files that define a pkg verb's Run function or the two
// sanctioned openers — and fails on any raw network, git, home-dir,
// messaging-store or Pub/Sub primitive outside authority.go. The cmd file
// set is computed from pkgSubcommands() in the AST, so a new verb's file is
// scanned without anyone listing it.
//
// A raw primitive is not wrong everywhere; it is wrong HERE, where code is
// reachable from a confined agent and the only gate is authority.go.
func TestPkgAuthorityBoundary(t *testing.T) {
	files := authoritySurfaceFiles(t)
	var violations []string
	for _, f := range files {
		violations = append(violations, scanAuthorityFile(t, f)...)
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Error(v)
	}
	if len(files) < 30 {
		t.Fatalf("boundary scan saw only %d files — the surface computation is broken (instrument blind)", len(files))
	}
}

// allowExec names the local-exec sites: each runs a local ailang child that
// inherits AILANG_AGENT_POLICY, so the child is confined by the same gates.
var allowExec = map[string]string{
	"iface_subprocess.go":  "execs `ailang internal-dump-iface` (local, inherits the policy env)",
	"publish_validator.go": "execs the package's smoke run with a local ailang (inherits the policy env)",
	"pkg_quality.go":       "execs `ailang test` for the quality report (inherits the policy env)",
}

// sanctionedOpener is the one function per primitive allowed to call it.
var sanctionedOpener = map[string]string{
	"messaging.OpenStore": "openPkgMsgStore",
	"pubsub.NewClient":    "newCascadePublisher",
}

func authoritySurfaceFiles(t *testing.T) []string {
	t.Helper()
	pkgFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range pkgFiles {
		if !strings.HasSuffix(f, "_test.go") && f != "authority.go" {
			out = append(out, f)
		}
	}

	cmdDir := filepath.Join("..", "..", "cmd", "ailang")
	fset := token.NewFileSet()
	table, err := parser.ParseFile(fset, filepath.Join(cmdDir, "commands_pkg.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"openPkgMsgStore": true, "newCascadePublisher": true}
	runs := 0
	ast.Inspect(table, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "pkgSubcommands" {
			return true
		}
		ast.Inspect(fd, func(n ast.Node) bool {
			if kv, ok := n.(*ast.KeyValueExpr); ok {
				if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Run" {
					if v, ok := kv.Value.(*ast.Ident); ok {
						wanted[v.Name] = true
						runs++
					}
				}
			}
			return true
		})
		return false
	})
	if runs < 15 {
		t.Fatalf("found only %d Run entries in pkgSubcommands() — parser drift", runs)
	}

	cmdFiles, err := filepath.Glob(filepath.Join(cmdDir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, f := range cmdFiles {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		inSurface := false
		for _, d := range af.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && wanted[fd.Name.Name] {
				found[fd.Name.Name] = true
				inSurface = true
			}
		}
		if inSurface {
			out = append(out, f)
		}
	}
	for name := range wanted {
		if !found[name] {
			t.Errorf("pkg surface function %s not found in cmd/ailang", name)
		}
	}
	return out
}

// importNames maps an import path's local name for the paths we police.
func importNames(f *ast.File) map[string]string {
	names := map[string]string{}
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		local := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			local = imp.Name.Name
		}
		switch path {
		case "net/http":
			names[local] = "http"
		case "os/exec":
			names[local] = "exec"
		case "os":
			names[local] = "os"
		case "github.com/sunholo-data/ailang/internal/messaging":
			names[local] = "messaging"
		case "github.com/sunholo-data/ailang/internal/pubsub":
			names[local] = "pubsub"
		}
	}
	return names
}

func scanAuthorityFile(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	names := importNames(f)
	base := filepath.Base(path)
	var out []string
	report := func(pos token.Pos, what, fix string) {
		p := fset.Position(pos)
		out = append(out, fmt.Sprintf("%s:%d: %s outside internal/pkg/authority.go — %s so it is gated under AILANG_AGENT_POLICY (#1720)", filepath.ToSlash(p.Filename), p.Line, what, fix))
	}
	qualified := func(e ast.Expr) string {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok {
			return ""
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || names[id.Name] == "" {
			return ""
		}
		return names[id.Name] + "." + sel.Sel.Name
	}

	for _, d := range f.Decls {
		fn := ""
		if fd, ok := d.(*ast.FuncDecl); ok {
			fn = fd.Name.Name
		}
		ast.Inspect(d, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				switch q := qualified(x); q {
				case "http.Get", "http.Post", "http.Head", "http.PostForm":
					report(x.Pos(), q, "use pkg.HTTPClient(what, timeout)")
				case "http.DefaultClient":
					report(x.Pos(), q, "use pkg.HTTPClient(what, timeout)")
				case "os.UserHomeDir":
					report(x.Pos(), q, "use pkg.UserDir / pkg.EnsureUserDir")
				case "messaging.OpenStore", "pubsub.NewClient":
					if sanctionedOpener[q] != fn {
						report(x.Pos(), q, "call "+sanctionedOpener[q])
					}
				case "exec.Command", "exec.CommandContext":
					if _, ok := allowExec[base]; !ok {
						report(x.Pos(), q, "run git via pkg.Git, or add a reasoned allowExec entry for a local ailang child")
					}
				}
			case *ast.CompositeLit:
				if qualified(x.Type) == "http.Client" {
					report(x.Pos(), "http.Client{...} literal", "use pkg.HTTPClient(what, timeout)")
				}
			}
			return true
		})
	}
	return out
}

// The detector must see a positive (mutation in miniature): a file with each
// forbidden construct is reported, and the sanctioned opener is not.
func TestPkgAuthorityBoundaryDetects(t *testing.T) {
	src := `package main
import (
	"net/http"
	gexec "os/exec"
	"os"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)
func a() { http.Get("x"); _ = http.DefaultClient; _ = &http.Client{} }
func b() { gexec.Command("git", "clone"); os.UserHomeDir() }
func c() { messaging.OpenStore(""); pubsub.NewClient(nil, "", "") }
func openPkgMsgStore() { messaging.OpenStore("") }
`
	dir := t.TempDir()
	p := filepath.Join(dir, "probe.go")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := scanAuthorityFile(t, p)
	if len(got) != 7 {
		t.Fatalf("detector found %d violations, want 7:\n%s", len(got), strings.Join(got, "\n"))
	}
}
