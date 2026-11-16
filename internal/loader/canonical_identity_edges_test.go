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
	if got, ok := ml.modulePathOf("acme/p", in); !ok || got != "/x" {
		t.Errorf("derived path = %q, %v; want /x", got, ok)
	}
	// Unknown layout, a file outside the package dir, and a file no candidate
	// maps back to: no derived path.
	for name, c := range map[string][2]string{
		"unknown layout":  {"acme/unknown", in},
		"outside package": {"acme/p", outside},
		"no round-trip":   {"acme/p", writeModule(t, dir, "pkg/shp/y.ail", "module shp/y\n")},
	} {
		if got, ok := ml.modulePathOf(c[0], c[1]); ok {
			t.Errorf("%s: derived %q, want none", name, got)
		}
	}
	plain := NewModuleLoader(t.TempDir())
	plain.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})
	if _, ok := plain.modulePathOf("acme/p", in); ok {
		t.Error("a resolver without layouts derives nothing")
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
	if first, rest := splitFirstSegment("solo"); first != "solo" || rest != "" {
		t.Errorf("splitFirstSegment(solo) = %q, %q", first, rest)
	}
	if err := ml.canonicalizeImportDecl("acme/root", nil); err != nil { // nil import: no-op
		t.Fatal(err)
	}
}

func TestRecordIdentity_IgnoresEmptyFile(t *testing.T) {
	ml := NewModuleLoader(t.TempDir())
	ml.recordIdentity("m", "")
	if len(ml.identityOfFile) != 0 || len(ml.IdentityConflicts()) != 0 {
		t.Fatal("a module without a file records nothing")
	}
}

func TestCaseOnDisk(t *testing.T) {
	dir := t.TempDir()
	f := writeModule(t, dir, "cc/init.ail", "module cc/init\n")
	if _, _, bad := caseOnDisk(dir, f); bad {
		t.Fatal("exact case must not be a mismatch")
	}
	if w, a, bad := caseOnDisk(dir, filepath.Join(dir, "cc", "Init.ail")); !bad || w != "Init.ail" || a != "init.ail" {
		t.Fatalf("case mismatch not reported: %q %q %v", w, a, bad)
	}
	if w, a, bad := caseOnDisk(dir, filepath.Join(dir, "CC", "init.ail")); !bad || w != "CC" || a != "cc" {
		t.Fatalf("directory case mismatch not reported: %q %q %v", w, a, bad)
	}
	if _, _, bad := caseOnDisk(dir, filepath.Join(dir, "..", "x.ail")); bad {
		t.Fatal("a file outside the dir is not judged")
	}
	if _, _, bad := caseOnDisk(dir, filepath.Join(dir, "cc", "missing.ail")); bad {
		t.Fatal("a missing file is not a case mismatch")
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
