package gatelint

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Proxy audit (m-net-effect-proxy-boundary M4, AC-M4.1).
//
// Every first-party PRODUCTION http.Transport must honor the process egress
// policy: a composite literal with no Proxy field, or Proxy: nil, dials past
// HTTP(S)_PROXY — and past CI's poison proxy — which is the escape this
// sprint closed. ProxyAudit finds those literals.
//
// Scope, by human ruling D-6 = Option A (2026-08-06): this is the cheap
// per-file source gate. It parses each file (so a multi-line literal or an
// aliased net/http import is still seen) but has no type information, so it
// cannot see new(http.Transport), a Transport assigned after construction, a
// factory returning one, or a custom RoundTripper. The durable go/packages
// type-aware analyzer for those shapes is
// https://github.com/sunholo-data/ailang/issues/612 — extend that, not this.

// ProxyFinding is one production http.Transport literal with a nil Proxy.
type ProxyFinding struct {
	Path string // slash-separated, relative to the scan root
	Line int
	Func string // enclosing function ("" at package level)
}

// Key identifies a finding for the exemption table: "<path>:<func>".
func (f ProxyFinding) Key() string { return f.Path + ":" + f.Func }

// proxyAuditExemptions are nil-Proxy literals that are correct by
// construction. Each needs a reason; an exemption that matches nothing is
// itself a failure (the test reports it), so the table cannot rot.
var proxyAuditExemptions = map[string]string{
	"internal/effects/net_proxy.go:directTransport": "the per-request direct route of netProxyRoundTripper: http.ProxyFromEnvironment already selected NO proxy for this request, and the dialer is pinned to the validated target IP. Giving it a Proxy would re-route a NO_PROXY request and break pinning.",
}

// ProxyAudit scans first-party production Go files below root.
func ProxyAudit(root string) (findings []ProxyFinding, scanned int, err error) {
	for _, scanRoot := range scanRoots {
		base := filepath.Join(root, scanRoot)
		walkErr := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != base && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "testdata" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
					return filepath.SkipDir
				}
				return nil
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			found, parseErr := auditFile(path, filepath.ToSlash(rel))
			if parseErr != nil {
				return parseErr
			}
			scanned++
			findings = append(findings, found...)
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
			return nil, scanned, walkErr
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Line < findings[j].Line
	})
	return findings, scanned, nil
}

func auditFile(path, rel string) ([]ProxyFinding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	httpName := ""
	for _, imp := range file.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == "net/http" {
			httpName = "http"
			if imp.Name != nil {
				httpName = imp.Name.Name
			}
		}
	}
	if httpName == "" || httpName == "_" {
		return nil, nil
	}
	var findings []ProxyFinding
	for _, decl := range file.Decls {
		funcName := ""
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcName = fn.Name.Name
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isHTTPTransport(lit.Type, httpName) || hasNonNilProxy(lit) {
				return true
			}
			findings = append(findings, ProxyFinding{Path: rel, Line: fset.Position(lit.Pos()).Line, Func: funcName})
			return true
		})
	}
	return findings, nil
}

func isHTTPTransport(expr ast.Expr, httpName string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Transport" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == httpName
}

func hasNonNilProxy(lit *ast.CompositeLit) bool {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Proxy" {
			v, isIdent := kv.Value.(*ast.Ident)
			return !isIdent || v.Name != "nil"
		}
	}
	return false
}
