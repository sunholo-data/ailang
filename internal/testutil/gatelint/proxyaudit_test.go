package gatelint

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestProxyAudit_SelfTest proves the detector on a checked-in positive
// fixture: blinding or deleting the scanner fails here, not silently.
func TestProxyAudit_SelfTest(t *testing.T) {
	findings, scanned, err := ProxyAudit(filepath.Join("testdata", "proxyaudit"))
	if err != nil {
		t.Fatal(err)
	}
	if scanned != 2 {
		t.Fatalf("scanned %d production files, want 2 (test files and testdata are out of scope)", scanned)
	}
	var got []string
	for _, f := range findings {
		got = append(got, fmt.Sprintf("%s:%d:%s", f.Path, f.Line, f.Func))
	}
	want := []string{
		"internal/fake/aliased.go:6:aliased",
		"internal/fake/transports.go:10:",
		"internal/fake/transports.go:13:explicitNil",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("findings mismatch\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestProxyAudit_Repo (AC-M4.1): zero first-party production http.Transport
// literals bypass the process proxy, outside the reasoned exemption table —
// and every exemption still matches a real literal.
func TestProxyAudit_Repo(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, scanned, err := ProxyAudit(root)
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 1000 {
		t.Fatalf("scanned only %d production files; the repository root or walker scope is broken", scanned)
	}
	used := map[string]bool{}
	var bad []string
	for _, f := range findings {
		if _, ok := proxyAuditExemptions[f.Key()]; ok {
			used[f.Key()] = true
			continue
		}
		bad = append(bad, fmt.Sprintf("%s:%d (func %q): http.Transport with a nil Proxy dials past HTTP(S)_PROXY — set Proxy: http.ProxyFromEnvironment", f.Path, f.Line, f.Func))
	}
	for key := range proxyAuditExemptions {
		if !used[key] {
			bad = append(bad, fmt.Sprintf("stale exemption %q matches no literal; delete it", key))
		}
	}
	if len(bad) != 0 {
		t.Fatalf("proxy audit after scanning %d production files:\n%s", scanned, strings.Join(bad, "\n"))
	}
	t.Logf("scanned %d first-party production files; %d exempted nil-Proxy literal(s)", scanned, len(used))
}
