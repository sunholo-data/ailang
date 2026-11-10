package loader

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestModulePathOf_Fallbacks(t *testing.T) {
	dir := t.TempDir()
	in := writeModule(t, dir, "pkg/shp/x.ail", "module shp/x\n")
	outside := writeModule(t, dir, "elsewhere/x.ail", "module shp/x\n")
	lr := &layoutResolver{
		mockPkgResolver: mockPkgResolver{files: map[string]string{"acme/p/x": in}},
		layouts:         map[string]layout{"acme/p": {dir: filepath.Join(dir, "pkg"), prefix: "shp"}},
	}
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(lr)
	if got := ml.modulePathOf("acme/p", in, "/spelled"); got != "/x" {
		t.Errorf("derived path = %q, want /x", got)
	}
	// Unknown layout, a file outside the package dir, and a file no candidate
	// maps back to: the spelling's own path is kept.
	if got := ml.modulePathOf("acme/unknown", in, "/spelled"); got != "/spelled" {
		t.Errorf("unknown layout = %q", got)
	}
	if got := ml.modulePathOf("acme/p", outside, "/spelled"); got != "/spelled" {
		t.Errorf("file outside the package = %q", got)
	}
	other := writeModule(t, dir, "pkg/shp/y.ail", "module shp/y\n")
	if got := ml.modulePathOf("acme/p", other, "/spelled"); got != "/spelled" {
		t.Errorf("no round-trip = %q", got)
	}
	// A resolver without layouts: the spelling decides, as before scoping.
	plain := NewModuleLoader(t.TempDir())
	plain.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})
	if got := plain.modulePathOf("acme/p", in, "/spelled"); got != "/spelled" {
		t.Errorf("no layout resolver = %q", got)
	}
	if got, ok := plain.pinOwnPrefix("acme/p", "shp/x"); ok || got != "" {
		t.Errorf("no layout resolver: nothing pinned, got %q", got)
	}
}

func TestPackageModuleID_RootForms(t *testing.T) {
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})
	ml.SetModulePrefixMap(map[string]string{"acme/root": "rt"})
	ml.SetCurrentPackageName("acme/root")
	for rest, want := range map[string]string{"/a/b": "rt/a/b", "": "acme/root"} {
		if got := ml.packageModuleID("acme/root", rest); got != want {
			t.Errorf("root %q -> %q, want %q", rest, got, want)
		}
	}
	if got := ml.packageModuleID("acme/dep", "/a"); got != "pkg/acme/dep/a" {
		t.Errorf("dependency -> %q", got)
	}
	if got := ml.prefixOf("acme/nobody"); got != "" {
		t.Errorf("prefix of an unknown package = %q", got)
	}
	if got, ok := ml.pinOwnPrefix("acme/root", "rt/x"); ok || got != "" {
		t.Errorf("the root's own spelling is never pinned, got %q", got)
	}
	if first, rest := splitFirstSegment("solo"); first != "solo" || rest != "" {
		t.Errorf("splitFirstSegment(solo) = %q, %q", first, rest)
	}
	ml.canonicalizeImportDecl("acme/root", nil) // nil import: no-op
}

func TestRecordIdentity_IgnoresEmptyFile(t *testing.T) {
	ml := NewModuleLoader(t.TempDir())
	ml.recordIdentity("m", "")
	if len(ml.identityOfFile) != 0 || len(ml.IdentityConflicts()) != 0 {
		t.Fatal("a module without a file records nothing")
	}
}

func TestDeclaredModuleDisplay(t *testing.T) {
	file := func(p string) *ast.File { return &ast.File{Module: &ast.ModuleDecl{Path: p}} }
	for _, tc := range []struct {
		id   string
		file *ast.File
		want string
	}{
		{"pkg/acme/shapes/types/shape", file("shp/types/shape"), "pkg/acme/shapes/types/shape (module shp/types/shape)"},
		{"pkg/acme/shapes/types/shape", file("acme/shapes/types/shape"), "pkg/acme/shapes/types/shape"}, // only pkg/ differs
		{"shp/types/shape", file("shp/types/shape"), "shp/types/shape"},                                 // root's own
		{"var/folders/x/main", file("main"), "var/folders/x/main"},                                      // entry file
		{"pkg/a/b/c", nil, "pkg/a/b/c"},
		{"pkg/a/b/c", &ast.File{}, "pkg/a/b/c"},
	} {
		if got := DeclaredModuleDisplay(tc.id, tc.file); got != tc.want {
			t.Errorf("DeclaredModuleDisplay(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
